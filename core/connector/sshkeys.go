package connector

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/SelfHold/shard/core/connector/pb"
	"github.com/SelfHold/shard/core/engine"
)

// The SSH keys of the password safe through the Connector API (scope
// "ssh_keys", read only): engine/sshkeys.go.

// KeyReader reads SSH keys: the open engine, or this device's own copy
// while no storage answers.
type KeyReader interface {
	SSHKeyList() []engine.SSHKey
	SSHKeyGet(ids []string) []engine.SSHKey
	SSHKeyRevisions() map[string]string
}

func sshKeyPB(k engine.SSHKey) *pb.SshKey {
	out := &pb.SshKey{
		Id: k.ID, Revision: k.Revision, Title: k.Title, KeyType: k.KeyType, Fingerprint: k.Fingerprint,
		PublicKey: k.PublicKey, Comment: k.Comment, Encrypted: k.Encrypted, HasPassphrase: k.HasPassphrase,
		Group: k.Group, DeviceName: k.DeviceName, PrivateKey: k.PrivateKey, Passphrase: k.Passphrase,
	}
	if !k.Created.IsZero() {
		out.CreatedMs = k.Created.UnixMilli()
	}
	if !k.Modified.IsZero() {
		out.ModifiedMs = k.Modified.UnixMilli()
	}
	return out
}

func (s *Server) ListSshKeys(ctx context.Context, _ *pb.ListSshKeysRequest) (*pb.ListSshKeysReply, error) {
	if _, err := s.authScope(ctx, ScopeSSHKeys); err != nil {
		return nil, err
	}
	rd, err := s.Host.SSHKeys()
	if err != nil {
		return nil, err
	}
	out := &pb.ListSshKeysReply{}
	for _, k := range rd.SSHKeyList() {
		out.Keys = append(out.Keys, sshKeyPB(k))
	}
	return out, nil
}

// GetSshKeys is a secret read: the registry records which app read keys
// and when (shown in Shard), never which keys.
func (s *Server) GetSshKeys(ctx context.Context, r *pb.GetSshKeysRequest) (*pb.ListSshKeysReply, error) {
	p, err := s.authScope(ctx, ScopeSSHKeys)
	if err != nil {
		return nil, err
	}
	if len(r.Ids) > engine.MaxIDs {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d ids", engine.MaxIDs)
	}
	rd, err := s.Host.SSHKeys()
	if err != nil {
		return nil, err
	}
	out := &pb.ListSshKeysReply{}
	keys := rd.SSHKeyGet(r.Ids)
	for _, k := range keys {
		out.Keys = append(out.Keys, sshKeyPB(k))
	}
	if len(keys) > 0 {
		s.Reg.SecretsRead(p.ID, ScopeSSHKeys)
	}
	return out, nil
}
