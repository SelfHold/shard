package rpc

import (
	"context"
	"errors"
	"sort"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/SelfHold/shard/core/engine"
	"github.com/SelfHold/shard/core/rpc/pb"
	"github.com/SelfHold/shard/core/secrets"
)

var kindToPB = map[secrets.Kind]pb.SecretKind{
	secrets.KindPassword: pb.SecretKind_SECRET_PASSWORD,
	secrets.KindSSH:      pb.SecretKind_SECRET_SSH,
	secrets.KindWeb3:     pb.SecretKind_SECRET_WEB3,
	secrets.KindTOTP:     pb.SecretKind_SECRET_TOTP,
	secrets.KindServer:   pb.SecretKind_SECRET_SERVER,
}

var serverAuthToPB = map[secrets.ServerAuth]pb.ServerAuth{
	secrets.AuthPassword: pb.ServerAuth_SERVER_AUTH_PASSWORD,
	secrets.AuthSSHKey:   pb.ServerAuth_SERVER_AUTH_SSH_KEY,
	secrets.AuthAgent:    pb.ServerAuth_SERVER_AUTH_AGENT,
}

func serverAuthFromPB(a pb.ServerAuth) secrets.ServerAuth {
	for s, p := range serverAuthToPB {
		if p == a {
			return s
		}
	}
	return secrets.AuthUnspecified
}

func serverToPB(sv *secrets.ServerMeta, d *secrets.Data) *pb.SecretServer {
	if sv == nil {
		return nil
	}
	out := &pb.SecretServer{Folder: sv.Folder, Auth: serverAuthToPB[sv.Auth], SshKeyId: sv.SSHKeyID,
		HostKeys: sv.HostKeys, JumpServerId: sv.JumpServerID, Tags: sv.Tags}
	if d != nil {
		for k := range d.Extra {
			out.ExtraKeys = append(out.ExtraKeys, k)
		}
		sort.Strings(out.ExtraKeys)
	}
	return out
}

func serverFromPB(sv *pb.SecretServer) *secrets.ServerMeta {
	if sv == nil {
		return nil // PutSecret keeps the stored fields
	}
	return &secrets.ServerMeta{Folder: sv.Folder, Auth: serverAuthFromPB(sv.Auth), SSHKeyID: sv.SshKeyId,
		HostKeys: sv.HostKeys, JumpServerID: sv.JumpServerId, Tags: sv.Tags}
}

func kindFromPB(k pb.SecretKind) secrets.Kind {
	for s, p := range kindToPB {
		if p == k {
			return s
		}
	}
	return secrets.KindPassword
}

func networkToPB(n secrets.Network) pb.Web3Network {
	switch n {
	case secrets.NetworkEVM:
		return pb.Web3Network_WEB3_EVM
	case secrets.NetworkSolana:
		return pb.Web3Network_WEB3_SOLANA
	}
	return pb.Web3Network_WEB3_NETWORK_UNSPECIFIED
}

func networkFromPB(n pb.Web3Network) secrets.Network {
	if n == pb.Web3Network_WEB3_SOLANA {
		return secrets.NetworkSolana
	}
	return secrets.NetworkEVM
}

func secretToPB(e secrets.Entry) *pb.SecretEntry {
	out := &pb.SecretEntry{
		Id: e.ID, Version: e.Version, Kind: kindToPB[e.Kind], Title: e.Title, GroupId: e.Group,
		Username: e.Username, Url: e.URL, Comment: e.Comment, Network: networkToPB(e.Network),
		KeyType: e.KeyType, PublicKey: e.PublicKey, Fingerprint: e.Fingerprint, Address: e.Address,
		CreatedUnix: e.Created.Unix(), UpdatedUnix: e.Updated.Unix(), Device: e.Device, ConflictOf: e.ConflictOf,
		RecoveryLeft: int32(e.RecoveryLeft), Server: serverToPB(e.Server, e.Data),
	}
	if e.Data != nil {
		out.Opened = true
		out.Password, out.Notes, out.PrivateKey, out.Passphrase = e.Data.Password, e.Data.Notes, e.Data.PrivateKey, e.Data.Passphrase
		if t := e.Data.TOTP; t != nil {
			out.TotpSecret, out.TotpAlgorithm, out.TotpDigits, out.TotpPeriod = t.Secret, t.Algorithm, int32(t.Digits), int32(t.Period)
		}
		out.RecoveryCodes = e.Data.RecoveryCodes
	}
	return out
}

func groupToPB(g secrets.Group) *pb.SecretGroup {
	return &pb.SecretGroup{Id: g.ID, Name: g.Name, Icon: g.Icon, ConflictOf: g.ConflictOf}
}

func (s *Service) secretsChanged() { s.publish(pb.Event_SECRETS_CHANGED, "") }

func (s *Service) ListSecrets(context.Context, *pb.Empty) (*pb.SecretsReply, error) {
	e, err := s.safe()
	if err != nil {
		return nil, err
	}
	list := e.Safe()
	out := &pb.SecretsReply{Broken: int32(list.Broken)}
	for _, g := range list.Groups {
		out.Groups = append(out.Groups, groupToPB(g))
	}
	for _, en := range list.Entries {
		out.Entries = append(out.Entries, secretToPB(en))
	}
	return out, nil
}

func (s *Service) GetSecret(_ context.Context, r *pb.IdRequest) (*pb.SecretEntry, error) {
	e, err := s.safe()
	if err != nil {
		return nil, err
	}
	en, err := e.OpenSecret(r.Id)
	if err != nil {
		return nil, toStatus(err)
	}
	return secretToPB(en), nil
}

func (s *Service) PutSecret(ctx context.Context, r *pb.SecretEntry) (*pb.SecretEntry, error) {
	e, err := s.engine()
	if err != nil {
		return nil, err
	}
	m := secrets.Meta{
		Kind: kindFromPB(r.Kind), Title: r.Title, Group: r.GroupId, Username: r.Username, URL: r.Url,
		Comment: r.Comment,
	}
	if m.Kind == secrets.KindWeb3 {
		m.Network = networkFromPB(r.Network)
	}
	if m.Kind == secrets.KindServer {
		m.Server = serverFromPB(r.Server)
	}
	d := secrets.Data{Password: r.Password, Notes: r.Notes, PrivateKey: r.PrivateKey, Passphrase: r.Passphrase}
	if m.Kind == secrets.KindTOTP {
		d.TOTP = &secrets.TOTP{Secret: r.TotpSecret, Algorithm: r.TotpAlgorithm, Digits: int(r.TotpDigits), Period: int(r.TotpPeriod)}
		d.RecoveryCodes = r.RecoveryCodes
	}
	en, err := e.PutSecret(ctx, r.Id, r.Version, m, d)
	if err != nil {
		return nil, toStatus(err)
	}
	s.secretsChanged()
	return secretToPB(en), nil
}

func (s *Service) DeleteSecret(ctx context.Context, r *pb.IdRequest) (*pb.Empty, error) {
	return s.deleteSecret(ctx, r.Id)
}

func (s *Service) DeleteSecretGroup(ctx context.Context, r *pb.IdRequest) (*pb.Empty, error) {
	return s.deleteSecret(ctx, r.Id)
}

func (s *Service) deleteSecret(ctx context.Context, id string) (*pb.Empty, error) {
	e, err := s.engine()
	if err != nil {
		return nil, err
	}
	if err := e.DeleteSecret(ctx, id); err != nil {
		return nil, toStatus(err)
	}
	s.secretsChanged()
	return &pb.Empty{}, nil
}

func (s *Service) PutSecretGroup(ctx context.Context, r *pb.SecretGroup) (*pb.SecretGroup, error) {
	e, err := s.engine()
	if err != nil {
		return nil, err
	}
	g, err := e.PutSecretGroup(ctx, r.Id, secrets.GroupMeta{Name: r.Name, Icon: r.Icon})
	if err != nil {
		return nil, toStatus(err)
	}
	s.secretsChanged()
	return groupToPB(g), nil
}

// DeriveKey needs no open vault: it only computes.
func (s *Service) DeriveKey(_ context.Context, r *pb.DeriveKeyRequest) (*pb.DerivedKey, error) {
	switch r.Kind {
	case pb.SecretKind_SECRET_SSH:
		k, err := secrets.SSHPublic(r.PrivateKey, r.Passphrase, r.Comment)
		if err != nil {
			return nil, toStatus(err)
		}
		return &pb.DerivedKey{KeyType: k.Type, PublicKey: k.AuthorizedKey, Fingerprint: k.Fingerprint}, nil
	case pb.SecretKind_SECRET_WEB3:
		k, err := secrets.Web3Public(networkFromPB(r.Network), r.PrivateKey)
		if err != nil {
			return nil, toStatus(err)
		}
		return &pb.DerivedKey{KeyType: k.Type, PublicKey: k.PublicKey, Address: k.Address}, nil
	}
	return nil, status.Error(codes.InvalidArgument, "only SSH and web3 keys have a public key")
}

func (s *Service) TotpCodes(context.Context, *pb.Empty) (*pb.TotpCodesReply, error) {
	e, err := s.safe()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := &pb.TotpCodesReply{NowUnix: now.Unix()}
	for _, c := range e.TOTPCodes(now) {
		out.Codes = append(out.Codes, &pb.TotpCode{
			Id: c.ID, Code: c.Code, Next: c.Next, Period: int32(c.Period), ExpiresUnix: c.Expires.Unix(),
		})
	}
	return out, nil
}

func (s *Service) UseRecoveryCode(ctx context.Context, r *pb.RecoveryCodeRequest) (*pb.SecretEntry, error) {
	e, err := s.engine()
	if err != nil {
		return nil, err
	}
	en, err := e.UseRecoveryCode(ctx, r.Id, r.Code)
	if err != nil {
		return nil, toStatus(err)
	}
	s.secretsChanged()
	return secretToPB(en), nil
}

// CheckTotp needs no open vault: it only computes.
func (s *Service) CheckTotp(_ context.Context, r *pb.CheckTotpRequest) (*pb.TotpPreview, error) {
	out := &pb.TotpPreview{RecoveryCodes: secrets.ParseRecoveryCodes(r.RecoveryText)}
	if r.Secret != "" {
		previewTOTP(out, r.Secret, secrets.TOTP{Algorithm: r.Algorithm, Digits: int(r.Digits), Period: int(r.Period)})
	}
	return out, nil
}

func previewTOTP(out *pb.TotpPreview, secret string, def secrets.TOTP) {
	t, info, err := secrets.ParseTOTP(secret, def)
	if err != nil {
		out.Error = err.Error()
		return
	}
	out.Issuer, out.Account = info.Issuer, info.Account
	out.Algorithm, out.Digits, out.Period = t.Algorithm, int32(t.Digits), int32(t.Period)
	out.Code, _ = t.Code(time.Now())
}

// ReadTotpLinks needs no open vault: it only computes.
func (s *Service) ReadTotpLinks(_ context.Context, r *pb.TotpLinksRequest) (*pb.TotpLinksReply, error) {
	links, batches := secrets.ReadTOTPLinks(r.Text)
	out := &pb.TotpLinksReply{}
	for _, l := range links {
		p := &pb.TotpPreview{}
		if l.Error != nil {
			p.Error, p.Issuer = l.Error.Error(), l.Name
		} else {
			previewTOTP(p, l.Link, secrets.TOTP{})
		}
		link := l.Link
		if p.Error != "" {
			link = ""
		}
		out.Links = append(out.Links, &pb.TotpLink{Link: link, Preview: p})
	}
	for _, b := range batches {
		out.Batches = append(out.Batches, &pb.TotpBatch{Id: int32(b.ID), Index: int32(b.Index), Size: int32(b.Size)})
	}
	return out, nil
}

func (s *Service) ImportSecrets(ctx context.Context, r *pb.SecretImportRequest) (*pb.SecretImportReply, error) {
	e, err := s.engine()
	if err != nil {
		return nil, err
	}
	imp, err := secrets.ImportFile(r.FileName, r.Data, r.Password, r.KeyFile)
	if errors.Is(err, secrets.ErrImportPassword) {
		return &pb.SecretImportReply{NeedPassword: true, Format: "KeePass (KDBX)"}, nil
	}
	if err != nil {
		return nil, toStatus(err)
	}
	rep, err := e.ImportSecrets(ctx, imp.Items, engine.ImportOptions{Group: r.GroupId, KeepGroups: r.KeepGroups, DryRun: r.DryRun})
	if err != nil {
		return nil, toStatus(err)
	}
	out := &pb.SecretImportReply{
		Format: imp.Format, Added: int32(rep.Added), Duplicates: int32(rep.Duplicate), Failed: int32(rep.Failed),
		NewGroups: rep.NewGroups,
	}
	for i, it := range imp.Items {
		item := &pb.SecretImportItem{
			Kind: kindToPB[it.Kind], Title: it.Title, Username: it.Username, Url: it.URL, Group: rep.Group[i],
			Status: pb.SecretImportStatus(rep.Status[i]),
		}
		if it.Error != nil {
			item.Error = it.Error.Error()
		}
		out.Items = append(out.Items, item)
	}
	if !r.DryRun && rep.Added > 0 {
		s.secretsChanged()
	}
	return out, nil
}
