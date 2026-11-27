package mobilebind

import (
	"context"
	"errors"
	"slices"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	"github.com/SelfHold/shard/core/connector"
	cpb "github.com/SelfHold/shard/core/connector/pb"
)

// The Connector API for other Android apps (docs/CONNECTOR.md). The Kotlin
// bound service (ConnectorService) checks who calls through Binder and
// passes the calling package and the SHA-256 of its signing certificate;
// requests and replies are serialized proto messages of connector.proto.

var (
	connMu   sync.Mutex
	conn     *connector.Server
	connOf   any // the service conn was made for
	watches  = map[int64]context.CancelFunc{}
	watchSeq int64
)

func connServer() *connector.Server {
	mu.Lock()
	s := svc
	mu.Unlock()
	if s == nil {
		return nil
	}
	connMu.Lock()
	defer connMu.Unlock()
	if conn == nil || connOf != any(s) {
		conn, connOf = s.Connector(), s
	}
	return conn
}

func notStarted() []byte {
	b, _ := proto.Marshal(&cpb.CallReply{Code: int32(codes.Unavailable), Message: "Shard core is not running"})
	return b
}

// ConnectorRefused is the CallReply for a caller the bound service could
// not identify.
func ConnectorRefused(msg string) []byte {
	b, _ := proto.Marshal(&cpb.CallReply{Code: int32(codes.Unauthenticated), Message: msg})
	return b
}

// ConnectorCall serves one unary method (Hello, RequestPairing,
// AwaitPairing, Get, List, Put, Delete, AppKey, Unpair) and returns a
// serialized CallReply.
func ConnectorCall(pkg, cert, method string, request []byte) []byte {
	s := connServer()
	if s == nil {
		return notStarted()
	}
	return s.Call(context.Background(), connector.Caller{Package: pkg, Cert: cert}, method, request)
}

// ConnectorSink gets the changes of one Watch (implemented in Kotlin). An
// error from OnChange (the client went away) ends the watch.
type ConnectorSink interface {
	OnChange(change []byte) error
	// OnEnd gets a serialized CallReply with the reason the watch ended.
	OnEnd(reply []byte)
}

// ConnectorWatch starts a Watch for the caller and returns its handle for
// ConnectorUnwatch.
func ConnectorWatch(pkg, cert string, sink ConnectorSink) int64 {
	s := connServer()
	if s == nil {
		sink.OnEnd(notStarted())
		return 0
	}
	ctx, cancel := context.WithCancel(context.Background())
	connMu.Lock()
	watchSeq++
	id := watchSeq
	watches[id] = cancel
	connMu.Unlock()
	go func() {
		end := s.WatchFunc(ctx, connector.Caller{Package: pkg, Cert: cert}, func(b []byte) error {
			if err := sink.OnChange(b); err != nil {
				return errors.Join(errors.New("client gone"), err)
			}
			return nil
		})
		connMu.Lock()
		delete(watches, id)
		connMu.Unlock()
		cancel()
		sink.OnEnd(end)
	}()
	return id
}

// ConnectorUnwatch ends a Watch.
func ConnectorUnwatch(id int64) {
	connMu.Lock()
	cancel := watches[id]
	connMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// PendingPairings returns how many pairing requests wait for the user
// (for a notification).
func PendingPairings() int {
	mu.Lock()
	s := svc
	mu.Unlock()
	if s == nil {
		return 0
	}
	return len(s.Apps.Requests())
}

// PasswordPairing returns the name of an app whose pairing request asks
// for the password store ("" = none), for the notification to say so.
func PasswordPairing() string { return ScopePairing(connector.ScopePasswords) }

// ScopePairing returns the name of an app whose pairing request asks for
// scope ("passwords", "ssh_keys", "servers"; "" = none), for the
// notification to say so.
func ScopePairing(scope string) string {
	mu.Lock()
	s := svc
	mu.Unlock()
	if s == nil {
		return ""
	}
	for _, q := range s.Apps.Requests() {
		if slices.Contains(q.Scopes, scope) {
			return q.Name
		}
	}
	return ""
}
