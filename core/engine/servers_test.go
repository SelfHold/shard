package engine

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/SelfHold/shard/core/secrets"
)

// testSSHKey returns an OpenSSH private key (encrypted with passphrase
// when it is not "") and its authorized_keys line.
func testSSHKey(t *testing.T, passphrase string) (string, string) {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	sp, _ := ssh.NewPublicKey(pub)
	var (
		block *pem.Block
		err   error
	)
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(passphrase))
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(block)), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sp)))
}

func TestSSHKeysOverSafe(t *testing.T) {
	ctx := context.Background()
	a := newWorld(t).device("laptop")
	g, err := a.PutSecretGroup(ctx, "", secrets.GroupMeta{Name: "Work"})
	if err != nil {
		t.Fatal(err)
	}
	plainKey, plainPub := testSSHKey(t, "")
	encKey, _ := testSSHKey(t, "secret")
	k1, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindSSH, Title: "deploy", Group: g.ID, Comment: "me@laptop"},
		secrets.Data{PrivateKey: plainKey})
	if err != nil {
		t.Fatal(err)
	}
	k2, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindSSH, Title: "kept"}, secrets.Data{PrivateKey: encKey, Passphrase: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	// Encrypted, the passphrase not kept: OpenSSH has the public key in the clear.
	k3, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindSSH, Title: "asks"}, secrets.Data{PrivateKey: encKey})
	if err != nil {
		t.Fatal(err)
	}
	pw, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindPassword, Title: "pw"}, secrets.Data{Password: "x"})
	if err != nil {
		t.Fatal(err)
	}

	list := a.SSHKeyList()
	if len(list) != 3 {
		t.Fatalf("list: %+v", list)
	}
	byID := map[string]SSHKey{}
	for _, k := range list {
		if k.PrivateKey != "" || k.Passphrase != "" {
			t.Fatalf("a secret in the list: %+v", k)
		}
		byID[k.ID] = k
	}
	if k := byID[k1.ID]; k.Group != "Work" || k.Encrypted || k.HasPassphrase || k.PublicKey != plainPub+" me@laptop" ||
		k.KeyType != "ssh-ed25519" || !strings.HasPrefix(k.Fingerprint, "SHA256:") || k.DeviceName != "laptop" || k.Comment != "me@laptop" {
		t.Fatalf("plain key: %+v", k)
	}
	if k := byID[k2.ID]; !k.Encrypted || !k.HasPassphrase {
		t.Fatalf("key with its passphrase: %+v", k)
	}
	if k := byID[k3.ID]; !k.Encrypted || k.HasPassphrase {
		t.Fatalf("key without its passphrase: %+v", k)
	}
	got := a.SSHKeyGet([]string{k2.ID, pw.ID, "nope", k2.ID})
	if len(got) != 1 || got[0].PrivateKey != encKey || got[0].Passphrase != "secret" {
		t.Fatalf("get: %+v", got)
	}
	revs := a.SSHKeyRevisions()
	if len(revs) != 3 || revs[k1.ID] != k1.Version {
		t.Fatalf("revisions: %v", revs)
	}
}

func TestServersOverSafe(t *testing.T) {
	ctx := context.Background()
	a := newWorld(t).device("laptop")
	_, hostKey := testSSHKey(t, "")
	key, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindSSH, Title: "k"}, secrets.Data{PrivateKey: func() string {
		k, _ := testSSHKey(t, "")
		return k
	}()})
	if err != nil {
		t.Fatal(err)
	}

	sv, err := a.ServerPut(ctx, Server{Name: "db", Group: "/prod//eu/", Host: "db.example", Port: 2222, User: "root",
		Auth: secrets.AuthSSHKey, SSHKeyID: key.ID, HostKeys: []string{hostKey, hostKey}, Notes: "n", Password: "pw",
		Tags: []string{"pg", " pg "}, Extra: map[string][]byte{"cellar.dbs": {1}}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if sv.ID == "" || sv.Group != "prod/eu" || sv.Port != 2222 || len(sv.HostKeys) != 1 || len(sv.Tags) != 1 ||
		sv.DeviceName != "laptop" || sv.Created.IsZero() {
		t.Fatalf("put: %+v", sv)
	}
	list, more := a.ServerList("", false)
	if more || len(list) != 1 || list[0].Password != "" || list[0].Extra != nil || list[0].Notes != "n" ||
		list[0].Host != "db.example" || list[0].Port != 2222 || list[0].SSHKeyID != key.ID {
		t.Fatalf("list without data: %+v", list)
	}
	got := a.ServerGet([]string{sv.ID, key.ID, "nope"})
	if len(got) != 1 || got[0].Password != "pw" || got[0].Extra["cellar.dbs"][0] != 1 {
		t.Fatalf("get: %+v", got)
	}
	// What the user sees in Shard: a server entry with host:port.
	en, err := a.OpenSecret(sv.ID)
	if err != nil || en.Kind != secrets.KindServer || en.URL != "db.example:2222" || en.Username != "root" || en.Title != "db" {
		t.Fatalf("entry: %+v %v", en, err)
	}
	// Only keys and servers are servers.
	if _, err := a.ServerPut(ctx, Server{ID: key.ID, Host: "x"}, key.Version); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a key was changed as a server: %v", err)
	}
	if err := a.ServerDelete(ctx, key.ID, key.Version); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a key was deleted as a server: %v", err)
	}

	// CAS.
	if _, err := a.ServerPut(ctx, Server{ID: sv.ID, Host: "h"}, "old"); !errors.Is(err, ErrServerConflict) {
		t.Fatalf("stale base: %v", err)
	}
	if _, err := a.ServerPut(ctx, Server{Host: "h"}, "x"); !errors.Is(err, ErrServerConflict) {
		t.Fatalf("a new server with a base: %v", err)
	}
	up := sv
	up.Port, up.Host = 0, "2001:db8::1"
	up, err = a.ServerPut(ctx, up, sv.Revision)
	if err != nil || up.Revision == sv.Revision || up.Host != "2001:db8::1" || up.Port != 0 {
		t.Fatalf("update: %+v %v", up, err)
	}
	if en, _ := a.OpenSecret(sv.ID); en.URL != "2001:db8::1" {
		t.Fatalf("IPv6 address stored as %q", en.URL)
	}
	if err := a.ServerDelete(ctx, sv.ID, sv.Revision); !errors.Is(err, ErrServerConflict) {
		t.Fatalf("delete with a stale base: %v", err)
	}

	// Bad input.
	for name, bad := range map[string]Server{
		"no host":    {Name: "x"},
		"port":       {Host: "h", Port: 70000},
		"host:port":  {Host: "h:22"},
		"auth":       {Host: "h", Auth: "telnet"},
		"host key":   {Host: "h", HostKeys: []string{"not a key"}},
		"own jump":   {ID: up.ID, Host: "h", JumpServerID: up.ID},
		"extra size": {Host: "h", Extra: map[string][]byte{"a.b": make([]byte, secrets.MaxExtra)}},
	} {
		base := ""
		if bad.ID != "" {
			base = up.Revision
		}
		if _, err := a.ServerPut(ctx, bad, base); !errors.Is(err, secrets.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}

	// Edited in Shard without the server fields: they and extra stay.
	cur, _ := a.OpenSecret(up.ID)
	m := cur.Meta
	m.Server, m.Title = nil, "renamed"
	if _, err := a.PutSecret(ctx, up.ID, cur.Version, m, secrets.Data{Password: "pw2"}); err != nil {
		t.Fatal(err)
	}
	after := a.ServerGet([]string{up.ID})[0]
	if after.Name != "renamed" || after.Password != "pw2" || after.SSHKeyID != key.ID || len(after.HostKeys) != 1 ||
		after.Extra["cellar.dbs"] == nil {
		t.Fatalf("after an edit in Shard: %+v", after)
	}
	if err := a.ServerDelete(ctx, up.ID, after.Revision); err != nil {
		t.Fatal(err)
	}
	if len(a.ServerRevisions()) != 0 {
		t.Fatal("deleted server still listed")
	}
}

// A server changed on two devices while they could not see each other is
// kept twice: the second is a conflict copy, a server of its own.
func TestServersOfflineTwoDevices(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	a, b := w.device("laptop"), w.device("phone")
	shared, err := a.ServerPut(ctx, Server{Name: "web", Host: "web.example"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	w.offline(true, "mega1", "mega2", "gd", "s3")
	ea, eb := shared, shared
	ea.User, eb.User = "laptop-user", "phone-user"
	if _, err := a.ServerPut(ctx, ea, shared.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ServerPut(ctx, eb, shared.Revision); err != nil {
		t.Fatal(err)
	}
	w.offline(false, "mega1", "mega2", "gd", "s3")
	for _, e := range []*Engine{a, b, a} {
		if err := e.Sync(ctx); err != nil {
			t.Fatal(err)
		}
	}
	ra, rb := a.ServerRevisions(), b.ServerRevisions()
	if len(ra) != 2 || len(rb) != 2 {
		t.Fatalf("servers: %v / %v", ra, rb)
	}
	users := map[string]bool{}
	copies := 0
	for _, sv := range a.ServerGet(keys(ra)) {
		users[sv.User] = true
		if strings.HasPrefix(sv.ID, shared.ID+"~") {
			copies++
		}
		if rb[sv.ID] != sv.Revision {
			t.Fatalf("devices differ on %s", sv.ID)
		}
	}
	if !users["laptop-user"] || !users["phone-user"] || copies != 1 {
		t.Fatalf("concurrent edits: %v, %d copies", users, copies)
	}
}

func TestServerAddr(t *testing.T) {
	for in, want := range map[string][3]any{
		"host":               {"host", 0, ""},
		"host:22":            {"host", 0, ""},
		"host:2222":          {"host", 2222, ""},
		"me@host:2222":       {"host", 2222, "me"},
		"ssh://me@host:2/":   {"host", 2, "me"},
		"[2001:db8::1]:2222": {"2001:db8::1", 2222, ""},
		"[2001:db8::1]":      {"2001:db8::1", 0, ""},
		"2001:db8::1":        {"2001:db8::1", 0, ""},
		" 10.0.0.1 ":         {"10.0.0.1", 0, ""},
	} {
		h, p, u, err := secrets.SplitServerAddr(in)
		if err != nil || h != want[0] || p != want[1] || u != want[2] {
			t.Errorf("SplitServerAddr(%q) = %q %d %q %v", in, h, p, u, err)
		}
	}
	for _, bad := range []string{"", "host:0", "host:x", "a b", "[::1"} {
		if _, _, _, err := secrets.SplitServerAddr(bad); err == nil {
			t.Errorf("SplitServerAddr(%q) took it", bad)
		}
	}
	if got := secrets.ServerAddr("::1", 2222); got != "[::1]:2222" {
		t.Errorf("ServerAddr: %q", got)
	}
}
