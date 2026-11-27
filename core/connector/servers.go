package connector

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/SelfHold/shard/core/connector/pb"
	"github.com/SelfHold/shard/core/engine"
	"github.com/SelfHold/shard/core/secrets"
)

// The user's servers through the Connector API (scope "servers"): one
// list for every app, the server entries of the safe (engine/servers.go).

// ServerReader reads servers: the open engine, or this device's own copy
// while no storage answers.
type ServerReader interface {
	ServerList(after string, withData bool) ([]engine.Server, bool)
	ServerGet(ids []string) []engine.Server
	ServerRevisions() map[string]string
}

// ServerWriter also writes them (the open engine only).
type ServerWriter interface {
	ServerReader
	ServerPut(ctx context.Context, sv engine.Server, base string) (engine.Server, error)
	ServerDelete(ctx context.Context, id, base string) error
}

var authToPB = map[secrets.ServerAuth]pb.Server_Auth{
	secrets.AuthPassword: pb.Server_PASSWORD,
	secrets.AuthSSHKey:   pb.Server_SSH_KEY,
	secrets.AuthAgent:    pb.Server_AGENT,
}

func authFromPB(a pb.Server_Auth) secrets.ServerAuth {
	for s, p := range authToPB {
		if p == a {
			return s
		}
	}
	return secrets.AuthUnspecified // an unknown value too
}

func serverPB(sv engine.Server) *pb.Server {
	out := &pb.Server{
		Id: sv.ID, Revision: sv.Revision, Name: sv.Name, Group: sv.Group, Host: sv.Host, Port: uint32(sv.Port),
		User: sv.User, Auth: authToPB[sv.Auth], Password: sv.Password, SshKeyId: sv.SSHKeyID,
		HostKeys: sv.HostKeys, JumpServerId: sv.JumpServerID, Notes: sv.Notes, Tags: sv.Tags,
		DeviceName: sv.DeviceName, Extra: sv.Extra,
	}
	if !sv.Created.IsZero() {
		out.CreatedMs = sv.Created.UnixMilli()
	}
	if !sv.Modified.IsZero() {
		out.ModifiedMs = sv.Modified.UnixMilli()
	}
	return out
}

func serverFromPB(sv *pb.Server) engine.Server {
	if sv == nil {
		return engine.Server{}
	}
	out := engine.Server{
		ID: sv.Id, Name: sv.Name, Group: sv.Group, Host: sv.Host, User: sv.User, Auth: authFromPB(sv.Auth),
		Password: sv.Password, SSHKeyID: sv.SshKeyId, HostKeys: sv.HostKeys, JumpServerID: sv.JumpServerId,
		Notes: sv.Notes, Tags: sv.Tags, Extra: sv.Extra,
	}
	if sv.Port > 65535 {
		out.Port = -1 // refused by ServerPut
	} else {
		out.Port = int(sv.Port)
	}
	if sv.CreatedMs > 0 {
		out.Created = time.UnixMilli(sv.CreatedMs)
	}
	return out
}

// serverReader returns the servers and the vault they belong to.
func (s *Server) serverReader() (ServerReader, string, error) {
	r, err := s.Host.Servers()
	if err != nil {
		return nil, "", err
	}
	return r, s.Host.ConnectorState().VaultID, nil
}

func (s *Server) ListServers(ctx context.Context, r *pb.ListServersRequest) (*pb.ListServersReply, error) {
	p, err := s.authScope(ctx, ScopeServers)
	if err != nil {
		return nil, err
	}
	rd, vault, err := s.serverReader()
	if err != nil {
		return nil, err
	}
	after, err := pageAfter(r.PageToken)
	if err != nil {
		return nil, err
	}
	// The cursor first, as for logins: a change in between is sent again.
	out := &pb.ListServersReply{Cursor: s.serverChanges.observe(vault, rd.ServerRevisions())}
	list, more := rd.ServerList(after, r.Data)
	for _, sv := range list {
		out.Servers = append(out.Servers, serverPB(sv))
	}
	if more && len(list) > 0 {
		out.NextPageToken = pageToken(list[len(list)-1].ID)
	}
	if r.Data && len(list) > 0 {
		s.Reg.SecretsRead(p.ID, ScopeServers)
	}
	return out, nil
}

func (s *Server) GetServers(ctx context.Context, r *pb.GetServersRequest) (*pb.ListServersReply, error) {
	p, err := s.authScope(ctx, ScopeServers)
	if err != nil {
		return nil, err
	}
	if len(r.Ids) > engine.MaxIDs {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d ids", engine.MaxIDs)
	}
	rd, vault, err := s.serverReader()
	if err != nil {
		return nil, err
	}
	out := &pb.ListServersReply{Cursor: s.serverChanges.observe(vault, rd.ServerRevisions())}
	list := rd.ServerGet(r.Ids)
	for _, sv := range list {
		out.Servers = append(out.Servers, serverPB(sv))
	}
	if len(list) > 0 {
		s.Reg.SecretsRead(p.ID, ScopeServers)
	}
	return out, nil
}

func (s *Server) PutServer(ctx context.Context, r *pb.PutServerRequest) (*pb.Server, error) {
	if _, err := s.authScope(ctx, ScopeServers); err != nil {
		return nil, err
	}
	if r.Server == nil {
		return nil, status.Error(codes.InvalidArgument, "no server")
	}
	w, err := s.Host.ServerWriter()
	if err != nil {
		return nil, err
	}
	sv, err := w.ServerPut(ctx, serverFromPB(r.Server), r.Base)
	if err != nil {
		return nil, toStatus(err)
	}
	s.wake()
	return serverPB(sv), nil
}

func (s *Server) DeleteServer(ctx context.Context, r *pb.DeleteServerRequest) (*pb.DeleteServerReply, error) {
	if _, err := s.authScope(ctx, ScopeServers); err != nil {
		return nil, err
	}
	w, err := s.Host.ServerWriter()
	if err != nil {
		return nil, err
	}
	if err := w.ServerDelete(ctx, r.Id, r.Base); err != nil {
		return nil, toStatus(err)
	}
	s.wake()
	return &pb.DeleteServerReply{}, nil
}

func (s *Server) ListServerChanges(ctx context.Context, r *pb.ListChangesRequest) (*pb.ListChangesReply, error) {
	if _, err := s.authScope(ctx, ScopeServers); err != nil {
		return nil, err
	}
	rd, vault, err := s.serverReader()
	if err != nil {
		return nil, err
	}
	cur := s.serverChanges.observe(vault, rd.ServerRevisions())
	changes, ok := s.serverChanges.since(r.Cursor)
	if !ok {
		return &pb.ListChangesReply{Cursor: cur, Reset_: true}, nil
	}
	return &pb.ListChangesReply{Changes: changes, Cursor: cur}, nil
}
