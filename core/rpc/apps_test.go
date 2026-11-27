package rpc

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	"cloudsync/core/connector"
	cpb "cloudsync/core/connector/pb"
	"cloudsync/core/rpc/pb"
	"cloudsync/core/vault"
)

// readyService is an unlocked service with three local storages.
func readyService(t *testing.T) *Service {
	t.Helper()
	vault.DefaultKDF = vault.KDFParams{Time: 1, Memory: 1024, Threads: 1}
	ctx := context.Background()
	dir := t.TempDir()
	s := NewService(filepath.Join(dir, "vault.cs"), dir, "linux")
	s.SyncEvery = time.Hour
	t.Cleanup(func() { s.Close(ctx) })
	if _, err := s.CreateVault(ctx, &pb.CreateVaultRequest{Password: "pw", DataShards: 2, ParityShards: 1, DeviceName: "laptop"}); err != nil {
		t.Fatal(err)
	}
	s.waitOpen()
	for i := 0; i < 3; i++ {
		p := filepath.Join(dir, "store", string(rune('a'+i)))
		os.MkdirAll(p, 0o700)
		if _, err := s.AddBackend(ctx, &pb.AddBackendRequest{Name: "disk", Type: "local", Root: p}); err != nil {
			t.Fatal(err)
		}
		waitReload(t, s)
	}
	if st, _ := s.GetState(ctx, &pb.Empty{}); !st.Ready {
		t.Fatalf("not ready: %+v", st)
	}
	return s
}

func connectorClient(t *testing.T, s *Service) cpb.ConnectorClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	cpb.RegisterConnectorServer(srv, s.Connector())
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	cc, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cc.Close() })
	return cpb.NewConnectorClient(cc)
}

func TestConnectorFlow(t *testing.T) {
	ctx := context.Background()
	s := readyService(t)
	c := connectorClient(t, s)

	h, err := c.Hello(ctx, &cpb.HelloRequest{AppId: "megachat"})
	if err != nil || h.ApiVersion != 1 || h.State != cpb.VaultState_READY || h.Paired {
		t.Fatalf("hello: %+v %v", h, err)
	}
	if _, err := c.Get(ctx, &cpb.GetRequest{Key: "vault.mc"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("get without pairing: %v", err)
	}
	if _, err := c.RequestPairing(ctx, &cpb.PairingRequest{AppId: "Bad App"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad app id: %v", err)
	}

	// Pairing: the user sees the request with the same code and approves.
	tk, err := c.RequestPairing(ctx, &cpb.PairingRequest{AppId: "megachat", AppName: "MegaChat", Description: "MegaChat on linux"})
	if err != nil || len(tk.Code) != 6 || tk.Ticket == "" {
		t.Fatalf("request: %+v %v", tk, err)
	}
	apps, _ := s.ListApps(ctx, &pb.Empty{})
	if len(apps.Requests) != 1 || apps.Requests[0].Code != tk.Code || apps.Requests[0].Platform != "desktop" {
		t.Fatalf("requests: %+v", apps)
	}
	if r, _ := c.AwaitPairing(ctx, &cpb.AwaitPairingRequest{Ticket: tk.Ticket}); r.Status != cpb.PairingStatus_PENDING {
		t.Fatalf("before the answer: %+v", r)
	}
	go func() {
		time.Sleep(50 * time.Millisecond)
		s.DecidePairing(ctx, &pb.DecidePairingRequest{Id: apps.Requests[0].Id, Approve: true})
	}()
	res, err := c.AwaitPairing(ctx, &cpb.AwaitPairingRequest{Ticket: tk.Ticket, WaitSeconds: 10})
	if err != nil || res.Status != cpb.PairingStatus_APPROVED || res.Token == "" || res.AppId != "megachat" {
		t.Fatalf("await: %+v %v", res, err)
	}
	if r, _ := c.AwaitPairing(ctx, &cpb.AwaitPairingRequest{Ticket: tk.Ticket}); r.Status != cpb.PairingStatus_EXPIRED || r.Token != "" {
		t.Fatalf("token handed out twice: %+v", r)
	}
	// The token is not kept in clear.
	if data, _ := os.ReadFile(filepath.Join(s.DataRoot, "apps.json")); len(data) == 0 || strings.Contains(string(data), res.Token) {
		t.Fatalf("apps.json: %s", data)
	}

	auth := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+res.Token)
	if h, _ := c.Hello(auth, &cpb.HelloRequest{}); !h.Paired || h.AppId != "megachat" {
		t.Fatalf("hello when paired: %+v", h)
	}
	wctx, stopWatch := context.WithCancel(auth)
	defer stopWatch()
	w, err := c.Watch(wctx, &cpb.WatchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // the stream subscribes

	it, err := c.Put(auth, &cpb.PutRequest{Key: "vault.mc", Data: []byte("cfg-1")})
	if err != nil || len(it.Versions) != 1 || it.Revision == "" {
		t.Fatalf("put: %+v %v", it, err)
	}
	ch, err := w.Recv()
	if err != nil || ch.Kind != cpb.Change_RECORD || ch.Key != "vault.mc" || ch.Revision != it.Revision {
		t.Fatalf("watch: %+v %v", ch, err)
	}
	if _, err := c.Put(auth, &cpb.PutRequest{Key: "vault.mc", Data: []byte("x")}); status.Code(err) != codes.Aborted {
		t.Fatalf("stale base: %v", err)
	}
	got, err := c.Get(auth, &cpb.GetRequest{Key: "vault.mc"})
	if err != nil || string(got.Versions[0].Data) != "cfg-1" || got.Versions[0].DeviceName != "laptop" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if l, _ := c.List(auth, &cpb.ListRequest{}); len(l.Items) != 1 || l.Items[0].Versions[0].Data != nil {
		t.Fatalf("list: %+v", l)
	}
	k1, err := c.AppKey(auth, &cpb.AppKeyRequest{})
	if err != nil || len(k1.Key) != 32 || k1.KeyId == "" {
		t.Fatalf("app key: %+v %v", k1, err)
	}
	if apps, _ := s.ListApps(ctx, &pb.Empty{}); len(apps.Pairings) != 1 || len(apps.Data) != 1 || apps.Data[0].Records != 1 || len(apps.Requests) != 0 {
		t.Fatalf("apps: %+v", apps)
	}

	// Android: the bound service passes the package; the same API as bytes.
	srv := s.Connector()
	droid := connector.Caller{Package: "dev.megachat", Cert: "ab12"}
	var rep cpb.CallReply
	proto.Unmarshal(srv.Call(ctx, droid, "Get", mustMarshal(&cpb.GetRequest{Key: "vault.mc"})), &rep)
	if codes.Code(rep.Code) != codes.Unauthenticated {
		t.Fatalf("android before pairing: %+v", &rep)
	}
	proto.Unmarshal(srv.Call(ctx, droid, "RequestPairing", mustMarshal(&cpb.PairingRequest{AppId: "megachat", AppName: "MegaChat"})), &rep)
	var dtk cpb.PairingTicket
	if rep.Code != 0 || proto.Unmarshal(rep.Payload, &dtk) != nil {
		t.Fatalf("android request: %+v", &rep)
	}
	apps, _ = s.ListApps(ctx, &pb.Empty{})
	if len(apps.Requests) != 1 || apps.Requests[0].Package != "dev.megachat" || !apps.Requests[0].AlreadyPaired {
		t.Fatalf("android request seen as %+v", apps.Requests)
	}
	s.DecidePairing(ctx, &pb.DecidePairingRequest{Id: apps.Requests[0].Id, Approve: true})
	proto.Unmarshal(srv.Call(ctx, droid, "Get", mustMarshal(&cpb.GetRequest{Key: "vault.mc"})), &rep)
	var dit cpb.Item
	if rep.Code != 0 || proto.Unmarshal(rep.Payload, &dit) != nil || string(dit.Versions[0].Data) != "cfg-1" {
		t.Fatalf("android get: %+v", &rep)
	}
	// Another certificate is another caller.
	proto.Unmarshal(srv.Call(ctx, connector.Caller{Package: "dev.megachat", Cert: "ff"}, "Get", mustMarshal(&cpb.GetRequest{Key: "vault.mc"})), &rep)
	if codes.Code(rep.Code) != codes.Unauthenticated {
		t.Fatalf("other certificate: %+v", &rep)
	}
	if apps, _ := s.ListApps(ctx, &pb.Empty{}); len(apps.Pairings) != 2 {
		t.Fatalf("desktop and android pairings: %+v", apps.Pairings)
	}

	// Revoking ends the desktop app's access and its watch.
	var desk string
	for _, p := range s.Apps.Pairings() {
		if p.Package == "" {
			desk = p.ID
		}
	}
	if _, err := s.RevokeApp(ctx, &pb.IdRequest{Id: desk}); err != nil {
		t.Fatal(err)
	}
	for {
		ch, err := w.Recv()
		if err != nil {
			t.Fatalf("watch ended without UNPAIRED: %v", err)
		}
		if ch.Kind == cpb.Change_UNPAIRED {
			break
		}
	}
	if _, err := c.Get(auth, &cpb.GetRequest{Key: "vault.mc"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("get after revoke: %v", err)
	}

	// Locked: the app is told, records cannot be read.
	s.LockVault(ctx, &pb.Empty{})
	proto.Unmarshal(srv.Call(ctx, droid, "Hello", mustMarshal(&cpb.HelloRequest{})), &rep)
	var dh cpb.HelloReply
	proto.Unmarshal(rep.Payload, &dh)
	if dh.State != cpb.VaultState_LOCKED || !dh.Paired {
		t.Fatalf("hello when locked: %+v", &dh)
	}
	proto.Unmarshal(srv.Call(ctx, droid, "Put", mustMarshal(&cpb.PutRequest{Key: "vault.mc", Base: dit.Revision})), &rep)
	if codes.Code(rep.Code) != codes.FailedPrecondition {
		t.Fatalf("put when locked: %+v", &rep)
	}
	// Pairings survive a restart.
	if r := connector.NewRegistry(s.DataRoot); len(r.Pairings()) != 1 {
		t.Fatalf("pairings after restart: %+v", r.Pairings())
	}
}

func TestConnectorDeny(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := NewService(filepath.Join(dir, "vault.cs"), dir, "linux")
	c := connectorClient(t, s)
	if h, _ := c.Hello(ctx, &cpb.HelloRequest{}); h.State != cpb.VaultState_NO_VAULT {
		t.Fatalf("no vault: %+v", h)
	}
	tk, err := c.RequestPairing(ctx, &cpb.PairingRequest{AppId: "megachat"})
	if err != nil {
		t.Fatal(err)
	}
	// A second request of the same app replaces the first.
	tk2, _ := c.RequestPairing(ctx, &cpb.PairingRequest{AppId: "megachat"})
	if r, _ := c.AwaitPairing(ctx, &cpb.AwaitPairingRequest{Ticket: tk.Ticket}); r.Status != cpb.PairingStatus_EXPIRED {
		t.Fatalf("replaced request: %+v", r)
	}
	apps, _ := s.ListApps(ctx, &pb.Empty{})
	if len(apps.Requests) != 1 {
		t.Fatalf("requests: %+v", apps.Requests)
	}
	s.DecidePairing(ctx, &pb.DecidePairingRequest{Id: apps.Requests[0].Id})
	if r, _ := c.AwaitPairing(ctx, &cpb.AwaitPairingRequest{Ticket: tk2.Ticket, WaitSeconds: 1}); r.Status != cpb.PairingStatus_DENIED || r.Token != "" {
		t.Fatalf("denied: %+v", r)
	}
	for i := 0; i < 8; i++ {
		if _, err := c.RequestPairing(ctx, &cpb.PairingRequest{AppId: "app" + string(rune('a'+i))}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.RequestPairing(ctx, &cpb.PairingRequest{AppId: "one-more"}); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("flood: %v", err)
	}
}

func mustMarshal(m proto.Message) []byte {
	b, err := proto.Marshal(m)
	if err != nil {
		panic(err)
	}
	return b
}
