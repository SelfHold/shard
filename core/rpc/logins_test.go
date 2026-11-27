package rpc

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/SelfHold/shard/core/connector"
	cpb "github.com/SelfHold/shard/core/connector/pb"
	"github.com/SelfHold/shard/core/rpc/pb"
)

// pairApp pairs app over c and returns a context with its token; grant are
// the scopes the user allows of those asked.
func pairApp(t *testing.T, s *Service, c cpb.ConnectorClient, app string, ask, grant []string) (context.Context, *cpb.PairingResult) {
	t.Helper()
	ctx := context.Background()
	tk, err := c.RequestPairing(ctx, &cpb.PairingRequest{AppId: app, AppName: app, Scopes: ask})
	if err != nil {
		t.Fatal(err)
	}
	apps, _ := s.ListApps(ctx, &pb.Empty{})
	var id string
	for _, q := range apps.Requests {
		if q.AppId == app {
			id = q.Id
			if strings.Join(q.Scopes, ",") != strings.Join(ask, ",") {
				t.Fatalf("request shows scopes %v, asked %v", q.Scopes, ask)
			}
		}
	}
	if _, err := s.DecidePairing(ctx, &pb.DecidePairingRequest{Id: id, Approve: true, Scopes: grant}); err != nil {
		t.Fatal(err)
	}
	res, err := c.AwaitPairing(ctx, &cpb.AwaitPairingRequest{Ticket: tk.Ticket})
	if err != nil || res.Status != cpb.PairingStatus_APPROVED {
		t.Fatalf("await: %+v %v", res, err)
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+res.Token), res
}

// storeFiles counts the objects in the service's local storages.
func storeFiles(t *testing.T, s *Service) int {
	t.Helper()
	n := 0
	filepath.WalkDir(filepath.Join(filepath.Dir(s.VaultPath), "store"), func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func TestConnectorLogins(t *testing.T) {
	ctx := context.Background()
	s := readyService(t)
	c := connectorClient(t, s)

	if _, err := c.RequestPairing(ctx, &cpb.PairingRequest{AppId: "enclave", Scopes: []string{"files"}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("unknown scope: %v", err)
	}

	// Records only: every password call is refused.
	plain, res := pairApp(t, s, c, "chat", nil, nil)
	if len(res.Scopes) != 0 {
		t.Fatalf("scopes granted unasked: %v", res.Scopes)
	}
	denied := map[string]error{}
	_, denied["ListLogins"] = c.ListLogins(plain, &cpb.ListLoginsRequest{})
	_, denied["GetLogins"] = c.GetLogins(plain, &cpb.GetLoginsRequest{Ids: []string{"x"}})
	_, denied["PutLogin"] = c.PutLogin(plain, &cpb.PutLoginRequest{Login: &cpb.Login{Url: "https://a.example/"}})
	_, denied["DeleteLogin"] = c.DeleteLogin(plain, &cpb.DeleteLoginRequest{Id: "x"})
	_, denied["TouchLogin"] = c.TouchLogin(plain, &cpb.TouchLoginRequest{Id: "x"})
	_, denied["ListChanges"] = c.ListChanges(plain, &cpb.ListChangesRequest{})
	for m, err := range denied {
		if status.Code(err) != codes.PermissionDenied {
			t.Errorf("%s without the scope: %v", m, err)
		}
	}
	if _, err := c.ListLogins(ctx, &cpb.ListLoginsRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("not paired: %v", err)
	}
	// Asked, but the user allowed records only.
	if _, res := pairApp(t, s, c, "other", []string{"passwords"}, nil); len(res.Scopes) != 0 {
		t.Fatalf("records only granted %v", res.Scopes)
	}

	auth, res := pairApp(t, s, c, "enclave", []string{"passwords"}, []string{"passwords"})
	if len(res.Scopes) != 1 || res.Scopes[0] != "passwords" {
		t.Fatalf("granted: %v", res.Scopes)
	}
	if h, _ := c.Hello(auth, &cpb.HelloRequest{}); len(h.Scopes) != 1 {
		t.Fatalf("hello: %+v", h)
	}
	wctx, stopWatch := context.WithCancel(auth)
	defer stopWatch()
	w, err := c.Watch(wctx, &cpb.WatchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // the stream subscribes

	// Made in Shard: listed as a login.
	ui, err := s.PutSecret(ctx, &pb.SecretEntry{Kind: pb.SecretKind_SECRET_PASSWORD, Title: "Mail",
		Username: "me", Url: "https://mail.example/inbox", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	if ch, err := w.Recv(); err != nil || ch.Kind != cpb.Change_LOGIN || ch.Key != ui.Id {
		t.Fatalf("watch after an entry made in Shard: %+v %v", ch, err)
	}
	l, err := c.ListLogins(auth, &cpb.ListLoginsRequest{})
	if err != nil || len(l.Logins) != 1 || l.Logins[0].SignonRealm != "https://mail.example/" ||
		l.Logins[0].Url != "https://mail.example/" || l.Logins[0].Password != "" || l.Cursor == "" {
		t.Fatalf("list: %+v %v", l, err)
	}
	cursor := l.Cursor

	// CAS.
	nl, err := c.PutLogin(auth, &cpb.PutLoginRequest{Login: &cpb.Login{Url: "https://shop.example/login",
		Username: "u", Password: "p1", Extra: map[string][]byte{"enclave.scheme": {1}}}})
	if err != nil || nl.Id == "" || nl.DeviceName != "laptop" {
		t.Fatalf("create: %+v %v", nl, err)
	}
	if _, err := c.PutLogin(auth, &cpb.PutLoginRequest{Login: &cpb.Login{Id: nl.Id, Url: nl.Url, Password: "p2"}}); status.Code(err) != codes.Aborted {
		t.Fatalf("stale base: %v", err)
	}
	up := proto.Clone(nl).(*cpb.Login)
	up.Password = "p2"
	up, err = c.PutLogin(auth, &cpb.PutLoginRequest{Login: up, Base: nl.Revision})
	if err != nil || up.Revision == nl.Revision {
		t.Fatalf("update: %+v %v", up, err)
	}
	if _, err := c.DeleteLogin(auth, &cpb.DeleteLoginRequest{Id: nl.Id, Base: nl.Revision}); status.Code(err) != codes.Aborted {
		t.Fatalf("delete with a stale base: %v", err)
	}
	if _, err := c.PutLogin(auth, &cpb.PutLoginRequest{Login: &cpb.Login{Id: "nope", Url: "https://x.example/"}}); status.Code(err) != codes.NotFound {
		t.Fatalf("unknown id: %v", err)
	}
	if _, err := c.PutLogin(auth, &cpb.PutLoginRequest{Login: &cpb.Login{Username: "x"}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("no site: %v", err)
	}
	g, err := c.GetLogins(auth, &cpb.GetLoginsRequest{Ids: []string{nl.Id}})
	if err != nil || len(g.Logins) != 1 || g.Logins[0].Password != "p2" || g.Logins[0].Extra["enclave.scheme"][0] != 1 {
		t.Fatalf("get: %+v %v", g, err)
	}
	if apps, _ := s.ListApps(ctx, &pb.Empty{}); func() bool {
		for _, p := range apps.Pairings {
			if p.AppId == "enclave" {
				return p.SecretsReadUnixMs == 0
			}
		}
		return true
	}() {
		t.Fatal("the time passwords were read is not shown")
	}

	// Blocked: visible to the app, not in Shard's list.
	bl, err := c.PutLogin(auth, &cpb.PutLoginRequest{Login: &cpb.Login{SignonRealm: "https://never.example/", Blocked: true}})
	if err != nil || !bl.Blocked {
		t.Fatalf("blocked: %+v %v", bl, err)
	}
	list, _ := s.ListSecrets(ctx, &pb.Empty{})
	for _, e := range list.Entries {
		if e.Id == bl.Id {
			t.Fatal("a blocked site is in Shard's password list")
		}
	}

	// ListChanges since the first list: the two new logins.
	chg, err := c.ListChanges(auth, &cpb.ListChangesRequest{Cursor: cursor})
	if err != nil || chg.Reset_ || len(chg.Changes) != 2 || chg.Changes[0].Id != nl.Id || chg.Changes[0].Revision != up.Revision ||
		chg.Changes[1].Id != bl.Id {
		t.Fatalf("changes: %+v %v", chg, err)
	}
	if _, err := c.DeleteLogin(auth, &cpb.DeleteLoginRequest{Id: nl.Id, Base: up.Revision}); err != nil {
		t.Fatal(err)
	}
	chg2, _ := c.ListChanges(auth, &cpb.ListChangesRequest{Cursor: chg.Cursor})
	if len(chg2.Changes) != 1 || chg2.Changes[0].Id != nl.Id || !chg2.Changes[0].Deleted {
		t.Fatalf("changes after a delete: %+v", chg2)
	}
	if r, _ := c.ListChanges(auth, &cpb.ListChangesRequest{Cursor: "old.1"}); !r.Reset_ || r.Cursor != chg2.Cursor {
		t.Fatalf("unknown cursor: %+v", r)
	}

	// TouchLogin: counted on this device, no journal write, no new revision.
	before := storeFiles(t, s)
	e, _ := s.engine()
	pending := e.Pending()
	for i := 0; i < 20; i++ {
		if _, err := c.TouchLogin(auth, &cpb.TouchLoginRequest{Id: ui.Id}); err != nil {
			t.Fatal(err)
		}
	}
	if storeFiles(t, s) != before || e.Pending() != pending {
		t.Fatal("TouchLogin wrote to the journal")
	}
	g, _ = c.GetLogins(auth, &cpb.GetLoginsRequest{Ids: []string{ui.Id}})
	if g.Logins[0].TimesUsed != 20 || g.Logins[0].LastUsedMs == 0 || g.Logins[0].Revision != ui.Version {
		t.Fatalf("after touches: %+v", g.Logins[0])
	}
	if _, err := c.TouchLogin(auth, &cpb.TouchLoginRequest{Id: "nope"}); status.Code(err) != codes.NotFound {
		t.Fatalf("touch of an unknown login: %v", err)
	}
	s.Connector().Flush()
	if data, _ := os.ReadFile(filepath.Join(s.DataRoot, "login-use.json")); !strings.Contains(string(data), ui.Id) {
		t.Fatalf("use counters not kept: %s", data)
	}

	// Nothing secret in the local files the connector keeps.
	for _, f := range []string{"apps.json", "login-changes.json", "login-use.json"} {
		data, _ := os.ReadFile(filepath.Join(s.DataRoot, f))
		for _, secret := range []string{"p1", "p2", "\"pw\"", "shop.example", "mail.example"} {
			if strings.Contains(string(data), secret) {
				t.Fatalf("%s holds %s", f, secret)
			}
		}
	}

	// Taking the scope away: SCOPES on the stream, then PERMISSION_DENIED.
	for {
		ch, err := w.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if ch.Kind == cpb.Change_SCOPES {
			t.Fatal("SCOPES before the scope changed")
		}
		if ch.Kind == cpb.Change_LOGIN && ch.Key == nl.Id && ch.Deleted {
			break // the last change made above
		}
	}
	var pid string
	for _, p := range s.Apps.Pairings() {
		if p.App == "enclave" {
			pid = p.ID
		}
	}
	if _, err := s.SetAppScopes(ctx, &pb.SetAppScopesRequest{Id: pid}); err != nil {
		t.Fatal(err)
	}
	for {
		ch, err := w.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if ch.Kind == cpb.Change_SCOPES {
			break
		}
	}
	if _, err := c.ListLogins(auth, &cpb.ListLoginsRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("after the scope was taken: %v", err)
	}
	if h, _ := c.Hello(auth, &cpb.HelloRequest{}); !h.Paired || len(h.Scopes) != 0 {
		t.Fatalf("hello after the scope was taken: %+v", h)
	}
	// A scope cannot be added back without pairing again.
	if _, err := s.SetAppScopes(ctx, &pb.SetAppScopesRequest{Id: pid, Scopes: []string{connector.ScopePasswords}}); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.Apps.Authenticate(connector.Caller{Token: strings.TrimPrefix(tokenOf(auth), "Bearer ")}); p.Has(connector.ScopePasswords) {
		t.Fatal("SetAppScopes added a scope")
	}
}

func tokenOf(ctx context.Context) string {
	md, _ := metadata.FromOutgoingContext(ctx)
	return md.Get("authorization")[0]
}

func TestConnectorLoginsOffline(t *testing.T) {
	ctx := context.Background()
	s := readyService(t)
	c := connectorClient(t, s)
	auth, _ := pairApp(t, s, c, "enclave", []string{"passwords"}, []string{"passwords"})
	nl, err := c.PutLogin(auth, &cpb.PutLoginRequest{Login: &cpb.Login{Url: "https://a.example/", Username: "u", Password: "p"}})
	if err != nil {
		t.Fatal(err)
	}
	// Unlocked but no engine: reads from this device's copy, writes refused.
	s.closeEngine(ctx, true)
	if st := s.ConnectorState(); st.State != cpb.VaultState_OFFLINE {
		t.Fatalf("state: %v", st.State)
	}
	g, err := c.GetLogins(auth, &cpb.GetLoginsRequest{Ids: []string{nl.Id}})
	if err != nil || len(g.Logins) != 1 || g.Logins[0].Password != "p" {
		t.Fatalf("offline read: %+v %v", g, err)
	}
	if _, err := c.PutLogin(auth, &cpb.PutLoginRequest{Login: &cpb.Login{Url: "https://b.example/"}}); status.Code(err) != codes.Unavailable {
		t.Fatalf("offline write: %v", err)
	}
	if _, err := s.LockVault(ctx, &pb.Empty{}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListLogins(auth, &cpb.ListLoginsRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("locked: %v", err)
	}
}
