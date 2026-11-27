package connector

import (
	"testing"
	"time"
)

func TestRequestExpires(t *testing.T) {
	r := NewRegistry(t.TempDir())
	now := time.Unix(1700000000, 0)
	r.now = func() time.Time { return now }
	q, err := r.Ask(Caller{}, "chat", "Chat", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Requests()) != 1 {
		t.Fatal("request not listed")
	}
	now = now.Add(RequestTTL + time.Second)
	if len(r.Requests()) != 0 {
		t.Fatal("expired request still listed")
	}
	if err := r.Decide(q.ID, true); err != ErrUnknown {
		t.Fatalf("approving an expired request: %v", err)
	}
	if a := r.Await(q.Ticket(), 0, nil); a.Status != Expired || a.Token != "" {
		t.Fatalf("await: %+v", a)
	}
	if len(r.Pairings()) != 0 {
		t.Fatal("expired request paired")
	}
}

func TestPairAgainReplaces(t *testing.T) {
	r := NewRegistry(t.TempDir())
	pair := func(c Caller) string {
		q, err := r.Ask(c, "chat", "Chat", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Decide(q.ID, true); err != nil {
			t.Fatal(err)
		}
		return r.Await(q.Ticket(), 0, nil).Token
	}
	old := pair(Caller{})
	tok := pair(Caller{})
	pair(Caller{Package: "dev.chat", Cert: "aa"})
	if n := len(r.Pairings()); n != 2 {
		t.Fatalf("%d pairings, want desktop + android", n)
	}
	if _, ok := r.Authenticate(Caller{Token: old}); ok {
		t.Fatal("replaced token still works")
	}
	if p, ok := r.Authenticate(Caller{Token: tok}); !ok || p.App != "chat" {
		t.Fatal("new token refused")
	}
}
