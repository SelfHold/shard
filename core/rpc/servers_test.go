package rpc

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"cloudsync/core/connector"
	cpb "cloudsync/core/connector/pb"
	"cloudsync/core/rpc/pb"
)

func newSSHKey(t *testing.T) (private, authorized string) {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	sp, _ := ssh.NewPublicKey(pub)
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(block)), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sp)))
}

// recvUntil reads the stream until a change of kind for key arrives.
func recvUntil(t *testing.T, w cpb.Connector_WatchClient, kind cpb.Change_Kind, key string) *cpb.Change {
	t.Helper()
	for {
		ch, err := w.Recv()
		if err != nil {
			t.Fatalf("waiting for %v %s: %v", kind, key, err)
		}
		if ch.Kind == kind && (key == "" || ch.Key == key) {
			return ch
		}
	}
}

func pairingOf(t *testing.T, s *Service, app string) *pb.AppPairing {
	t.Helper()
	apps, _ := s.ListApps(context.Background(), &pb.Empty{})
	for _, p := range apps.Pairings {
		if p.AppId == app {
			return p
		}
	}
	t.Fatalf("no pairing of %s", app)
	return nil
}

func TestConnectorSSHKeysAndServers(t *testing.T) {
	ctx := context.Background()
	s := readyService(t)
	c := connectorClient(t, s)

	// Without the scopes every call is refused.
	plain, _ := pairApp(t, s, c, "chat", []string{"passwords"}, []string{"passwords"})
	denied := map[string]error{}
	_, denied["ListSshKeys"] = c.ListSshKeys(plain, &cpb.ListSshKeysRequest{})
	_, denied["GetSshKeys"] = c.GetSshKeys(plain, &cpb.GetSshKeysRequest{Ids: []string{"x"}})
	_, denied["ListServers"] = c.ListServers(plain, &cpb.ListServersRequest{})
	_, denied["GetServers"] = c.GetServers(plain, &cpb.GetServersRequest{Ids: []string{"x"}})
	_, denied["PutServer"] = c.PutServer(plain, &cpb.PutServerRequest{Server: &cpb.Server{Host: "h"}})
	_, denied["DeleteServer"] = c.DeleteServer(plain, &cpb.DeleteServerRequest{Id: "x"})
	_, denied["ListServerChanges"] = c.ListServerChanges(plain, &cpb.ListChangesRequest{})
	for m, err := range denied {
		if status.Code(err) != codes.PermissionDenied {
			t.Errorf("%s without the scope: %v", m, err)
		}
	}

	// Asked for both, the user allows the servers only.
	sm, res := pairApp(t, s, c, "sshmanager", []string{"servers", "ssh_keys"}, []string{"servers"})
	if len(res.Scopes) != 1 || res.Scopes[0] != "servers" {
		t.Fatalf("granted: %v", res.Scopes)
	}
	if _, err := c.ListSshKeys(sm, &cpb.ListSshKeysRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("keys without their scope: %v", err)
	}
	db, res := pairApp(t, s, c, "dbmanager", []string{"servers", "ssh_keys"}, []string{"servers", "ssh_keys"})
	if len(res.Scopes) != 2 {
		t.Fatalf("granted: %v", res.Scopes)
	}
	wctx, stopWatch := context.WithCancel(db)
	defer stopWatch()
	w, err := c.Watch(wctx, &cpb.WatchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // the stream subscribes

	// An SSH key made in CloudSync.
	priv, pub := newSSHKey(t)
	key, err := s.PutSecret(ctx, &pb.SecretEntry{Kind: pb.SecretKind_SECRET_SSH, Title: "deploy", PrivateKey: priv, Comment: "ci"})
	if err != nil {
		t.Fatal(err)
	}
	if ch := recvUntil(t, w, cpb.Change_SSH_KEY, key.Id); ch.Revision != key.Version || ch.Deleted {
		t.Fatalf("key change: %+v", ch)
	}
	kl, err := c.ListSshKeys(db, &cpb.ListSshKeysRequest{})
	if err != nil || len(kl.Keys) != 1 || kl.Keys[0].PrivateKey != "" || kl.Keys[0].PublicKey != pub+" ci" ||
		kl.Keys[0].Encrypted || kl.Keys[0].Revision != key.Version {
		t.Fatalf("list keys: %+v %v", kl, err)
	}
	if p := pairingOf(t, s, "dbmanager"); p.KeysReadUnixMs != 0 {
		t.Fatal("listing keys counted as a secret read")
	}
	kg, err := c.GetSshKeys(db, &cpb.GetSshKeysRequest{Ids: []string{key.Id, "nope"}})
	if err != nil || len(kg.Keys) != 1 || kg.Keys[0].PrivateKey != priv {
		t.Fatalf("get keys: %+v %v", kg, err)
	}
	if p := pairingOf(t, s, "dbmanager"); p.KeysReadUnixMs == 0 || p.SecretsReadUnixMs != 0 {
		t.Fatalf("key read times: %+v", p)
	}
	if _, err := c.GetSshKeys(db, &cpb.GetSshKeysRequest{Ids: make([]string, 501)}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("501 ids: %v", err)
	}

	// Servers: one list for both apps.
	first, err := c.ListServers(db, &cpb.ListServersRequest{})
	if err != nil || len(first.Servers) != 0 || first.Cursor == "" {
		t.Fatalf("empty list: %+v %v", first, err)
	}
	sv, err := c.PutServer(sm, &cpb.PutServerRequest{Server: &cpb.Server{Name: "db1", Group: "prod", Host: "db1.example",
		Port: 2222, User: "root", Auth: cpb.Server_SSH_KEY, SshKeyId: key.Id, HostKeys: []string{pub},
		Password: "srv-secret", Extra: map[string][]byte{"sshmanager.color": []byte("red")}}})
	if err != nil || sv.Id == "" || sv.DeviceName != "laptop" || sv.Port != 2222 || sv.Auth != cpb.Server_SSH_KEY {
		t.Fatalf("create: %+v %v", sv, err)
	}
	if ch := recvUntil(t, w, cpb.Change_SERVER, sv.Id); ch.Revision != sv.Revision {
		t.Fatalf("server change: %+v", ch)
	}
	l, err := c.ListServers(db, &cpb.ListServersRequest{})
	if err != nil || len(l.Servers) != 1 || l.Servers[0].Password != "" || len(l.Servers[0].Extra) != 0 ||
		l.Servers[0].Host != "db1.example" || l.Servers[0].SshKeyId != key.Id {
		t.Fatalf("list: %+v %v", l, err)
	}
	if p := pairingOf(t, s, "dbmanager"); p.ServersReadUnixMs != 0 {
		t.Fatal("a list without data counted as a secret read")
	}
	l, _ = c.ListServers(db, &cpb.ListServersRequest{Data: true})
	if l.Servers[0].Password != "srv-secret" || string(l.Servers[0].Extra["sshmanager.color"]) != "red" {
		t.Fatalf("list with data: %+v", l.Servers[0])
	}
	if p := pairingOf(t, s, "dbmanager"); p.ServersReadUnixMs == 0 {
		t.Fatal("server passwords read, not shown")
	}

	// CAS; the other app keeps the extra of the first.
	if _, err := c.PutServer(db, &cpb.PutServerRequest{Server: &cpb.Server{Id: sv.Id, Host: "x"}}); status.Code(err) != codes.Aborted {
		t.Fatalf("stale base: %v", err)
	}
	up := proto.Clone(l.Servers[0]).(*cpb.Server)
	up.Extra["dbmanager.dbs"] = []byte("[]")
	up.HostKeys = append(up.HostKeys, "ssh-ed25519 not*base64")
	if _, err := c.PutServer(db, &cpb.PutServerRequest{Server: up, Base: l.Servers[0].Revision}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("a bad host key: %v", err) // not base64
	}
	_, pub2 := newSSHKey(t)
	up.HostKeys[1] = pub2
	up, err = c.PutServer(db, &cpb.PutServerRequest{Server: up, Base: l.Servers[0].Revision})
	if err != nil || len(up.HostKeys) != 2 || len(up.Extra) != 2 {
		t.Fatalf("update: %+v %v", up, err)
	}
	if _, err := c.DeleteServer(db, &cpb.DeleteServerRequest{Id: sv.Id, Base: sv.Revision}); status.Code(err) != codes.Aborted {
		t.Fatalf("delete with a stale base: %v", err)
	}
	if _, err := c.PutServer(db, &cpb.PutServerRequest{Server: &cpb.Server{Id: key.Id, Host: "x"}, Base: key.Version}); status.Code(err) != codes.NotFound {
		t.Fatalf("a key changed as a server: %v", err)
	}
	if _, err := c.PutServer(db, &cpb.PutServerRequest{Server: &cpb.Server{Name: "x"}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("no host: %v", err)
	}

	// The user sees it in CloudSync as a server entry.
	en, err := s.GetSecret(ctx, &pb.IdRequest{Id: sv.Id})
	if err != nil || en.Kind != pb.SecretKind_SECRET_SERVER || en.Url != "db1.example:2222" || en.Server.GetSshKeyId() != key.Id ||
		strings.Join(en.Server.ExtraKeys, ",") != "dbmanager.dbs,sshmanager.color" {
		t.Fatalf("entry: %+v %v", en, err)
	}

	// Changes since the first list; a login cursor is not a server cursor.
	chg, err := c.ListServerChanges(db, &cpb.ListChangesRequest{Cursor: first.Cursor})
	if err != nil || chg.Reset_ || len(chg.Changes) != 1 || chg.Changes[0].Id != sv.Id || chg.Changes[0].Revision != up.Revision {
		t.Fatalf("changes: %+v %v", chg, err)
	}
	ll, err := c.ListLogins(plain, &cpb.ListLoginsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := c.ListServerChanges(db, &cpb.ListChangesRequest{Cursor: ll.Cursor}); !r.Reset_ {
		t.Fatalf("a login cursor was taken: %+v", r)
	}
	if _, err := c.DeleteServer(db, &cpb.DeleteServerRequest{Id: sv.Id, Base: up.Revision}); err != nil {
		t.Fatal(err)
	}
	for !recvUntil(t, w, cpb.Change_SERVER, sv.Id).Deleted { // the update first
	}
	chg2, _ := c.ListServerChanges(db, &cpb.ListChangesRequest{Cursor: chg.Cursor})
	if len(chg2.Changes) != 1 || !chg2.Changes[0].Deleted {
		t.Fatalf("changes after a delete: %+v", chg2)
	}
	if _, err := s.DeleteSecret(ctx, &pb.IdRequest{Id: key.Id}); err != nil {
		t.Fatal(err)
	}
	if ch := recvUntil(t, w, cpb.Change_SSH_KEY, key.Id); !ch.Deleted {
		t.Fatalf("key delete change: %+v", ch)
	}

	// Nothing secret in the connector's own files.
	for _, f := range []string{"apps.json", "server-changes.json"} {
		data, _ := os.ReadFile(filepath.Join(s.DataRoot, f))
		for _, secret := range []string{"srv-secret", "db1.example", "PRIVATE KEY"} {
			if strings.Contains(string(data), secret) {
				t.Fatalf("%s holds %s", f, secret)
			}
		}
	}

	// Android: the call switch knows the new methods.
	srv := s.Connector()
	droid := connector.Caller{Package: "dev.nobody", Cert: "00"}
	for _, m := range []string{"ListSshKeys", "GetSshKeys", "ListServers", "GetServers", "PutServer", "DeleteServer", "ListServerChanges"} {
		var rep cpb.CallReply
		proto.Unmarshal(srv.Call(ctx, droid, m, nil), &rep)
		if codes.Code(rep.Code) != codes.Unauthenticated {
			t.Errorf("Android %s: %d %s", m, rep.Code, rep.Message)
		}
	}

	// Taking the keys away: SCOPES, then PERMISSION_DENIED; servers stay.
	if _, err := s.SetAppScopes(ctx, &pb.SetAppScopesRequest{Id: pairingOf(t, s, "dbmanager").Id, Scopes: []string{"servers"}}); err != nil {
		t.Fatal(err)
	}
	recvUntil(t, w, cpb.Change_SCOPES, "")
	if _, err := c.GetSshKeys(db, &cpb.GetSshKeysRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("after the scope was taken: %v", err)
	}
	if _, err := c.ListServers(db, &cpb.ListServersRequest{}); err != nil {
		t.Fatalf("servers after the keys were taken: %v", err)
	}
}

func TestConnectorServersOffline(t *testing.T) {
	ctx := context.Background()
	s := readyService(t)
	c := connectorClient(t, s)
	auth, _ := pairApp(t, s, c, "sshmanager", []string{"servers", "ssh_keys"}, []string{"servers", "ssh_keys"})
	sv, err := c.PutServer(auth, &cpb.PutServerRequest{Server: &cpb.Server{Host: "a.example", Password: "p"}})
	if err != nil {
		t.Fatal(err)
	}
	priv, _ := newSSHKey(t)
	key, err := s.PutSecret(ctx, &pb.SecretEntry{Kind: pb.SecretKind_SECRET_SSH, Title: "k", PrivateKey: priv})
	if err != nil {
		t.Fatal(err)
	}
	// Unlocked but no engine: reads from this device's copy, writes refused.
	s.closeEngine(ctx, true)
	g, err := c.GetServers(auth, &cpb.GetServersRequest{Ids: []string{sv.Id}})
	if err != nil || len(g.Servers) != 1 || g.Servers[0].Password != "p" {
		t.Fatalf("offline read: %+v %v", g, err)
	}
	k, err := c.GetSshKeys(auth, &cpb.GetSshKeysRequest{Ids: []string{key.Id}})
	if err != nil || len(k.Keys) != 1 || k.Keys[0].PrivateKey != priv {
		t.Fatalf("offline key read: %+v %v", k, err)
	}
	if _, err := c.PutServer(auth, &cpb.PutServerRequest{Server: &cpb.Server{Host: "b.example"}}); status.Code(err) != codes.Unavailable {
		t.Fatalf("offline write: %v", err)
	}
	if _, err := c.DeleteServer(auth, &cpb.DeleteServerRequest{Id: sv.Id, Base: sv.Revision}); status.Code(err) != codes.Unavailable {
		t.Fatalf("offline delete: %v", err)
	}
	if _, err := s.LockVault(ctx, &pb.Empty{}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListServers(auth, &cpb.ListServersRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("locked: %v", err)
	}
	if _, err := c.ListSshKeys(auth, &cpb.ListSshKeysRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("locked keys: %v", err)
	}
}
