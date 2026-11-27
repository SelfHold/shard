package rpc

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/SelfHold/shard/core/connector"
	cpb "github.com/SelfHold/shard/core/connector/pb"
	"github.com/SelfHold/shard/core/crypto"
	"github.com/SelfHold/shard/core/engine"
	"github.com/SelfHold/shard/core/eventlog"
	"github.com/SelfHold/shard/core/rpc/pb"
)

// Connected apps: the Service is the connector's Host, and the Shard
// UI manages pairings through the methods below.

// Connector returns the Connector API server over this service (one per
// service: it keeps the login change log and use counters).
func (s *Service) Connector() *connector.Server {
	s.connOnce.Do(func() { s.conn = connector.NewServer(hostAdapter{s}, s.Apps) })
	return s.conn
}

func (s *Service) ConnectorState() connector.HostState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := connector.HostState{Version: Version}
	switch {
	case s.e != nil:
		st.State = cpb.VaultState_READY
		st.VaultID, st.DeviceName = s.e.Config().VaultID, s.e.Device().Name
	case s.v != nil:
		st.State = cpb.VaultState_OFFLINE
		st.VaultID, st.DeviceName = s.v.Config().VaultID, s.deviceName
	default:
		st.State = cpb.VaultState_LOCKED
		if _, err := os.Stat(s.VaultPath); err != nil {
			st.State = cpb.VaultState_NO_VAULT
		}
	}
	return st
}

func (s *Service) AppRecords() (connector.Records, error) {
	r, err := s.safe()
	if err != nil {
		return nil, err
	}
	rec, ok := r.(connector.Records)
	if !ok {
		return nil, status.Error(codes.Internal, "no app records here")
	}
	return rec, nil
}

func (s *Service) AppWriter() (connector.Writer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.e != nil:
		return s.e, nil
	case s.v != nil:
		return nil, status.Error(codes.Unavailable, "Shard has no storage connection yet: "+s.problem)
	}
	return nil, status.Error(codes.FailedPrecondition, "Shard is locked")
}

func (s *Service) Logins() (connector.LoginReader, error) {
	r, err := s.safe()
	if err != nil {
		return nil, err
	}
	rd, ok := r.(connector.LoginReader)
	if !ok {
		return nil, status.Error(codes.Internal, "no logins here")
	}
	return rd, nil
}

func (s *Service) LoginWriter() (connector.LoginWriter, error) {
	w, err := s.AppWriter()
	if err != nil {
		return nil, err
	}
	return loginWriter{w.(*engine.Engine), s}, nil
}

// loginWriter tells the Shard UI after an app changed a login.
type loginWriter struct {
	*engine.Engine
	s *Service
}

func (w loginWriter) LoginPut(ctx context.Context, l engine.Login, base string) (engine.Login, error) {
	out, err := w.Engine.LoginPut(ctx, l, base)
	if err == nil {
		w.s.secretsChanged()
	}
	return out, err
}

func (w loginWriter) LoginDelete(ctx context.Context, id, base string) error {
	err := w.Engine.LoginDelete(ctx, id, base)
	if err == nil {
		w.s.secretsChanged()
	}
	return err
}

func (s *Service) SSHKeys() (connector.KeyReader, error) {
	r, err := s.safe()
	if err != nil {
		return nil, err
	}
	rd, ok := r.(connector.KeyReader)
	if !ok {
		return nil, status.Error(codes.Internal, "no SSH keys here")
	}
	return rd, nil
}

func (s *Service) Servers() (connector.ServerReader, error) {
	r, err := s.safe()
	if err != nil {
		return nil, err
	}
	rd, ok := r.(connector.ServerReader)
	if !ok {
		return nil, status.Error(codes.Internal, "no servers here")
	}
	return rd, nil
}

func (s *Service) ServerWriter() (connector.ServerWriter, error) {
	w, err := s.AppWriter()
	if err != nil {
		return nil, err
	}
	return serverWriter{w.(*engine.Engine), s}, nil
}

// serverWriter tells the Shard UI after an app changed a server.
type serverWriter struct {
	*engine.Engine
	s *Service
}

func (w serverWriter) ServerPut(ctx context.Context, sv engine.Server, base string) (engine.Server, error) {
	out, err := w.Engine.ServerPut(ctx, sv, base)
	if err == nil {
		w.s.secretsChanged()
	}
	return out, err
}

func (w serverWriter) ServerDelete(ctx context.Context, id, base string) error {
	err := w.Engine.ServerDelete(ctx, id, base)
	if err == nil {
		w.s.secretsChanged()
	}
	return err
}

func (s *Service) MasterKey() (crypto.Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.v == nil {
		return crypto.Key{}, status.Error(codes.FailedPrecondition, "Shard is locked")
	}
	return s.v.Config().Key(), nil
}

// hostAdapter is the Service as a connector.Host: every event may change
// what an app sees.
type hostAdapter struct{ *Service }

func (h hostAdapter) Subscribe(fn func()) func() {
	return h.Service.Subscribe(func(*pb.Event) { fn() })
}

func (s *Service) ListApps(context.Context, *pb.Empty) (*pb.AppsReply, error) {
	out := &pb.AppsReply{}
	pairings := s.Apps.Pairings()
	paired := map[string]bool{}
	for _, p := range pairings {
		paired[p.App] = true
		ap := &pb.AppPairing{Id: p.ID, AppId: p.App, AppName: p.Name, Description: p.Description,
			Platform: p.Platform(), Package: p.Package, Cert: p.Cert, ApprovedUnixMs: p.Approved.UnixMilli(),
			Scopes: p.Scopes, SecretsReadUnixMs: p.SecretsReadMs, KeysReadUnixMs: p.LastRead(connector.ScopeSSHKeys),
			ServersReadUnixMs: p.LastRead(connector.ScopeServers)}
		if !p.LastUsed.IsZero() {
			ap.LastUsedUnixMs = p.LastUsed.UnixMilli()
		}
		out.Pairings = append(out.Pairings, ap)
	}
	for _, q := range s.Apps.Requests() {
		platform := "desktop"
		if q.Package != "" {
			platform = "android"
		}
		out.Requests = append(out.Requests, &pb.PairingRequestInfo{
			Id: q.ID, AppId: q.App, AppName: q.Name, Description: q.Description, Code: q.Code,
			Platform: platform, Package: q.Package, Cert: q.Cert,
			CreatedUnixMs: q.Created.UnixMilli(), ExpiresUnixMs: q.Expires.UnixMilli(),
			AlreadyPaired: paired[q.App], Scopes: q.Scopes,
		})
	}
	if e, err := s.engine(); err == nil {
		for _, u := range e.AppsUsage() {
			out.Data = append(out.Data, &pb.AppDataInfo{AppId: u.App, Records: int32(u.Records), Bytes: u.Bytes})
		}
	}
	return out, nil
}

// scopeNames name the secret scopes in the event log.
var scopeNames = map[string]string{
	connector.ScopePasswords: "passwords",
	connector.ScopeSSHKeys:   "SSH keys",
	connector.ScopeServers:   "servers",
}

func appsStatus(err error) error {
	if errors.Is(err, connector.ErrUnknown) {
		return status.Error(codes.NotFound, err.Error())
	}
	return toStatus(err)
}

func (s *Service) DecidePairing(_ context.Context, r *pb.DecidePairingRequest) (*pb.Empty, error) {
	var name string
	var granted []string // secret scopes asked for and allowed
	for _, q := range s.Apps.Requests() {
		if q.ID == r.Id {
			name = q.Name + " (" + q.App + ")"
			for _, sc := range connector.SecretScopes {
				if slices.Contains(q.Scopes, sc) && slices.Contains(r.Scopes, sc) {
					granted = append(granted, scopeNames[sc])
				}
			}
		}
	}
	if err := s.Apps.Decide(r.Id, r.Approve, r.Scopes...); err != nil {
		return nil, appsStatus(err)
	}
	if r.Approve {
		msg := "paired " + name
		if len(granted) > 0 {
			msg += ", with access to " + strings.Join(granted, ", ")
		}
		s.Log.Add(eventlog.Info, "apps", "", msg)
	} else {
		s.Log.Add(eventlog.Info, "apps", "", "pairing refused: "+name)
	}
	return &pb.Empty{}, nil
}

func (s *Service) RevokeApp(_ context.Context, r *pb.IdRequest) (*pb.Empty, error) {
	if err := s.Apps.Revoke(r.Id); err != nil {
		return nil, appsStatus(err)
	}
	s.Log.Add(eventlog.Info, "apps", "", "pairing removed")
	return &pb.Empty{}, nil
}

func (s *Service) SetAppScopes(_ context.Context, r *pb.SetAppScopesRequest) (*pb.Empty, error) {
	if err := s.Apps.SetScopes(r.Id, r.Scopes); err != nil {
		return nil, appsStatus(err)
	}
	s.Log.Add(eventlog.Info, "apps", "", "app permissions changed")
	return &pb.Empty{}, nil
}

func (s *Service) DeleteAppData(ctx context.Context, r *pb.IdRequest) (*pb.Empty, error) {
	e, err := s.engine()
	if err != nil {
		return nil, err
	}
	n, err := e.AppDeleteAll(ctx, r.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	if n > 0 {
		s.Log.Addf(eventlog.Info, "apps", "", "deleted %d records of %s", n, r.Id)
		s.publish(pb.Event_TREE_CHANGED, "")
		s.publish(pb.Event_APPS_CHANGED, "")
	}
	return &pb.Empty{}, nil
}
