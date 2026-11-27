package connector

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"cloudsync/core/connector/pb"
	"cloudsync/core/crypto"
	"cloudsync/core/engine"
	"cloudsync/core/secrets"
)

// APIVersion is the Connector API version this core serves.
const APIVersion = 1

// Records reads app records: the open engine, or this device's own copy
// while no storage answers.
type Records interface {
	AppGet(app, key string) (engine.AppItem, error)
	AppList(app, prefix string) ([]engine.AppItem, error)
	AppRevisions(app string) map[string]engine.AppRevision
}

// Writer also writes them (the open engine only).
type Writer interface {
	Records
	AppPut(ctx context.Context, app, key, base string, data []byte) (engine.AppItem, error)
	AppDelete(ctx context.Context, app, key, base string) (engine.AppItem, error)
}

// HostState is what the host says about the vault.
type HostState struct {
	State      pb.VaultState
	VaultID    string
	DeviceName string
	Version    string // core version
}

// Host is the CloudSync service the connector works over (rpc.Service).
type Host interface {
	ConnectorState() HostState
	// AppRecords fails with a gRPC status while the vault is locked.
	AppRecords() (Records, error)
	// AppWriter fails with a gRPC status unless the engine is open.
	AppWriter() (Writer, error)
	// MasterKey fails with a gRPC status while the vault is locked.
	MasterKey() (crypto.Key, error)
	// Subscribe calls fn after anything may have changed (sync, state).
	Subscribe(fn func()) (cancel func())
	// Logins fails with a gRPC status while the vault is locked.
	Logins() (LoginReader, error)
	// LoginWriter fails with a gRPC status unless the engine is open.
	LoginWriter() (LoginWriter, error)
	// SSHKeys fails with a gRPC status while the vault is locked.
	SSHKeys() (KeyReader, error)
	// Servers fails with a gRPC status while the vault is locked.
	Servers() (ServerReader, error)
	// ServerWriter fails with a gRPC status unless the engine is open.
	ServerWriter() (ServerWriter, error)
}

// Server implements pb.ConnectorServer.
type Server struct {
	pb.UnimplementedConnectorServer
	Host Host
	Reg  *Registry

	mu    sync.Mutex
	wakes map[chan struct{}]struct{}

	changes       *changeLog // logins
	serverChanges *changeLog
	uses          *useCounters
}

// NewServer serves the connector API over host. Its files (change log,
// use counters) live next to the registry's.
func NewServer(host Host, reg *Registry) *Server {
	return &Server{Host: host, Reg: reg, wakes: map[chan struct{}]struct{}{},
		changes: newChangeLog(reg.Dir(), "login-changes.json"), serverChanges: newChangeLog(reg.Dir(), "server-changes.json"),
		uses: newUseCounters(reg.Dir())}
}

// Flush writes what is kept for later (use counters); call before exit.
func (s *Server) Flush() { s.uses.flush() }

type callerKey struct{}

// WithCaller marks ctx as a call from c (the Android bound service).
func WithCaller(ctx context.Context, c Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, c)
}

// callerOf reads the caller: set by WithCaller, or a bearer token.
func callerOf(ctx context.Context) Caller {
	if c, ok := ctx.Value(callerKey{}).(Caller); ok {
		return c
	}
	md, _ := metadata.FromIncomingContext(ctx)
	for _, v := range md.Get("authorization") {
		if t, ok := strings.CutPrefix(v, "Bearer "); ok && t != "" {
			return Caller{Token: t}
		}
	}
	return Caller{}
}

func (s *Server) auth(ctx context.Context) (Pairing, error) {
	p, ok := s.Reg.Authenticate(callerOf(ctx))
	if !ok {
		return Pairing{}, status.Error(codes.Unauthenticated, "not paired with CloudSync on this device")
	}
	return p, nil
}

func toStatus(err error) error {
	switch {
	case err == nil:
		return nil
	case status.Code(err) != codes.Unknown:
		return err
	case errors.Is(err, engine.ErrAppConflict), errors.Is(err, engine.ErrLoginConflict), errors.Is(err, engine.ErrServerConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, engine.ErrAppInvalid), errors.Is(err, ErrInvalid), errors.Is(err, secrets.ErrInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrTooMany), errors.Is(err, engine.ErrLoginsFull), errors.Is(err, engine.ErrServersFull):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, engine.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}

func (s *Server) Hello(ctx context.Context, r *pb.HelloRequest) (*pb.HelloReply, error) {
	st := s.Host.ConnectorState()
	out := &pb.HelloReply{ApiVersion: APIVersion, CoreVersion: st.Version, State: st.State,
		VaultId: st.VaultID, DeviceName: st.DeviceName}
	if p, ok := s.Reg.Authenticate(callerOf(ctx)); ok && (r.AppId == "" || r.AppId == p.App) {
		out.Paired, out.AppId, out.Scopes = true, p.App, p.Scopes
	}
	return out, nil
}

func (s *Server) RequestPairing(ctx context.Context, r *pb.PairingRequest) (*pb.PairingTicket, error) {
	c := callerOf(ctx)
	c.Token = "" // a desktop request is anonymous until approved
	q, err := s.Reg.Ask(c, r.AppId, r.AppName, r.Description, r.Scopes...)
	if err != nil {
		return nil, toStatus(err)
	}
	return &pb.PairingTicket{Ticket: q.Ticket(), Code: q.Code, ExpiresUnixMs: q.Expires.UnixMilli()}, nil
}

func (s *Server) AwaitPairing(ctx context.Context, r *pb.AwaitPairingRequest) (*pb.PairingResult, error) {
	wait := time.Duration(min(r.WaitSeconds, 60)) * time.Second
	a := s.Reg.Await(r.Ticket, wait, ctx.Done())
	out := &pb.PairingResult{Token: a.Token, AppId: a.App, Scopes: a.Scopes}
	switch a.Status {
	case Pending:
		out.Status = pb.PairingStatus_PENDING
	case Approved:
		out.Status = pb.PairingStatus_APPROVED
	case Denied:
		out.Status = pb.PairingStatus_DENIED
	default:
		out.Status = pb.PairingStatus_EXPIRED
	}
	return out, nil
}

func itemPB(it engine.AppItem) *pb.Item {
	out := &pb.Item{Key: it.Key, Revision: it.Revision}
	for _, v := range it.Versions {
		out.Versions = append(out.Versions, &pb.Version{Id: v.ID, Data: v.Data, Size: int64(v.Size),
			DeviceName: v.DeviceName, TimeUnixMs: v.Time.UnixMilli()})
	}
	return out
}

func (s *Server) Get(ctx context.Context, r *pb.GetRequest) (*pb.Item, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	rec, err := s.Host.AppRecords()
	if err != nil {
		return nil, err
	}
	it, err := rec.AppGet(p.App, r.Key)
	if err != nil {
		return nil, toStatus(err)
	}
	return itemPB(it), nil
}

func (s *Server) List(ctx context.Context, r *pb.ListRequest) (*pb.ListReply, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	rec, err := s.Host.AppRecords()
	if err != nil {
		return nil, err
	}
	items, err := rec.AppList(p.App, r.Prefix)
	if err != nil {
		return nil, toStatus(err)
	}
	out := &pb.ListReply{}
	for _, it := range items {
		out.Items = append(out.Items, itemPB(it))
	}
	return out, nil
}

func (s *Server) Put(ctx context.Context, r *pb.PutRequest) (*pb.Item, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	w, err := s.Host.AppWriter()
	if err != nil {
		return nil, err
	}
	it, err := w.AppPut(ctx, p.App, r.Key, r.Base, r.Data)
	if err != nil {
		return nil, toStatus(err)
	}
	s.wake()
	return itemPB(it), nil
}

func (s *Server) Delete(ctx context.Context, r *pb.DeleteRequest) (*pb.Item, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	w, err := s.Host.AppWriter()
	if err != nil {
		return nil, err
	}
	it, err := w.AppDelete(ctx, p.App, r.Key, r.Base)
	if err != nil {
		return nil, toStatus(err)
	}
	s.wake()
	return itemPB(it), nil
}

func (s *Server) AppKey(ctx context.Context, _ *pb.AppKeyRequest) (*pb.AppKeyReply, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	mk, err := s.Host.MasterKey()
	if err != nil {
		return nil, err
	}
	k := engine.AppKey(mk, p.App)
	return &pb.AppKeyReply{Key: k[:], KeyId: engine.KeyID(k)}, nil
}

func (s *Server) Unpair(ctx context.Context, _ *pb.UnpairRequest) (*pb.UnpairReply, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.Reg.Revoke(p.ID); err != nil && !errors.Is(err, ErrUnknown) {
		return nil, toStatus(err)
	}
	return &pb.UnpairReply{}, nil
}

// wake tells every Watch to look for changes (after a local write).
func (s *Server) wake() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.wakes {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Server) Watch(_ *pb.WatchRequest, stream pb.Connector_WatchServer) error {
	return s.watch(stream.Context(), stream.Send)
}

// watch sends changes of the caller's records and of the vault state to
// send until ctx ends or the pairing is revoked.
func (s *Server) watch(ctx context.Context, send func(*pb.Change) error) error {
	p, err := s.auth(ctx)
	if err != nil {
		return err
	}
	caller := callerOf(ctx)
	wake := make(chan struct{}, 1)
	s.mu.Lock()
	s.wakes[wake] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.wakes, wake)
		s.mu.Unlock()
	}()
	poke := func() {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
	cancel := s.Host.Subscribe(poke)
	defer cancel()
	regCh, stopReg := s.Reg.watch()
	defer stopReg()

	state := s.Host.ConnectorState()
	revs := s.revisions(p.App)
	scopes := p.Scopes
	feeds := s.feeds()
	for _, f := range feeds {
		f.take(p)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-wake:
		case <-regCh:
		}
		now, ok := s.Reg.Authenticate(caller)
		if !ok {
			_ = send(&pb.Change{Kind: pb.Change_UNPAIRED})
			return status.Error(codes.Unauthenticated, "the pairing was revoked")
		}
		if !slices.Equal(now.Scopes, scopes) {
			scopes, p = now.Scopes, now
			if err := send(&pb.Change{Kind: pb.Change_SCOPES}); err != nil {
				return err
			}
			for _, f := range feeds {
				if !p.Has(f.scope) {
					f.last = nil
				}
			}
		}
		if st := s.Host.ConnectorState(); st != state {
			state = st
			if err := send(&pb.Change{Kind: pb.Change_STATE, State: st.State}); err != nil {
				return err
			}
		}
		for _, f := range feeds {
			if err := f.send(p, send); err != nil {
				return err
			}
		}
		cur := s.revisions(p.App)
		if cur == nil {
			continue // locked: keep what was sent, compare after unlock
		}
		for k, r := range cur {
			if old, ok := revs[k]; !ok || old != r {
				if err := send(&pb.Change{Kind: pb.Change_RECORD, Key: k, Revision: r.Revision, Deleted: r.Deleted}); err != nil {
					return err
				}
			}
		}
		revs = cur
	}
}

// feed follows the revisions of one kind of item of the safe for Watch
// (logins, SSH keys, servers).
type feed struct {
	scope string
	kind  pb.Change_Kind
	revs  func() (map[string]string, error) // fails while locked
	last  map[string]string                 // nil: nothing known yet
}

func (s *Server) feeds() []*feed {
	return []*feed{
		{scope: ScopePasswords, kind: pb.Change_LOGIN, revs: func() (map[string]string, error) {
			rd, err := s.Host.Logins()
			if err != nil {
				return nil, err
			}
			return rd.LoginRevisions(), nil
		}},
		{scope: ScopeSSHKeys, kind: pb.Change_SSH_KEY, revs: func() (map[string]string, error) {
			rd, err := s.Host.SSHKeys()
			if err != nil {
				return nil, err
			}
			return rd.SSHKeyRevisions(), nil
		}},
		{scope: ScopeServers, kind: pb.Change_SERVER, revs: func() (map[string]string, error) {
			rd, err := s.Host.Servers()
			if err != nil {
				return nil, err
			}
			return rd.ServerRevisions(), nil
		}},
	}
}

// current returns the revisions an app with p may see; nil without the
// scope or while locked.
func (f *feed) current(p Pairing) map[string]string {
	if !p.Has(f.scope) {
		return nil
	}
	revs, err := f.revs()
	if err != nil {
		return nil
	}
	return revs
}

// take remembers the current state without sending it.
func (f *feed) take(p Pairing) { f.last = f.current(p) }

// send sends the changes since f.last (nil: nothing known yet, as after a
// grant or an unlock: only the state is taken).
func (f *feed) send(p Pairing, send func(*pb.Change) error) error {
	cur := f.current(p)
	if cur == nil {
		return nil
	}
	if f.last != nil {
		ids := make([]string, 0, len(cur))
		for id, rev := range cur {
			if f.last[id] != rev {
				ids = append(ids, id)
			}
		}
		for id := range f.last {
			if _, ok := cur[id]; !ok {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			rev, ok := cur[id]
			if err := send(&pb.Change{Kind: f.kind, Key: id, Revision: rev, Deleted: !ok}); err != nil {
				return err
			}
		}
	}
	f.last = cur
	return nil
}

func (s *Server) revisions(app string) map[string]engine.AppRevision {
	rec, err := s.Host.AppRecords()
	if err != nil {
		return nil
	}
	return rec.AppRevisions(app)
}
