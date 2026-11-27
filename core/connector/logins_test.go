package connector

import (
	"testing"
	"time"
)

func TestChangeLog(t *testing.T) {
	dir := t.TempDir()
	c := newChangeLog(dir, "login-changes.json")
	cur := c.observe("v1", map[string]string{"a": "1", "b": "1"})
	if ch, ok := c.since(cur); !ok || len(ch) != 0 {
		t.Fatalf("nothing changed: %v %v", ch, ok)
	}
	// Two devices wrote offline; after the sync this device sees a new
	// login of each, one changed and one deleted.
	next := c.observe("v1", map[string]string{"a": "2", "c": "1", "d": "1"})
	ch, ok := c.since(cur)
	if !ok || len(ch) != 4 {
		t.Fatalf("changes: %v %v", ch, ok)
	}
	got := map[string]string{}
	for _, x := range ch {
		got[x.Id] = x.Revision
		if x.Deleted != (x.Id == "b") {
			t.Fatalf("deleted flag: %+v", x)
		}
	}
	if got["a"] != "2" || got["c"] != "1" || got["d"] != "1" || got["b"] != "" {
		t.Fatalf("changes: %v", got)
	}
	if ch, _ := c.since(next); len(ch) != 0 {
		t.Fatalf("after the latest cursor: %v", ch)
	}

	// The log survives a restart; another vault or a lost file resets.
	c2 := newChangeLog(dir, "login-changes.json")
	if c2.observe("v1", map[string]string{"a": "2", "c": "1", "d": "1"}) != next {
		t.Fatal("cursor changed over a restart without changes")
	}
	if _, ok := c2.since(cur); !ok {
		t.Fatal("old cursor refused after a restart")
	}
	c2.observe("v2", map[string]string{})
	if _, ok := c2.since(next); ok {
		t.Fatal("a cursor of another vault was taken")
	}
	if _, ok := newChangeLog(t.TempDir(), "login-changes.json").since(next); ok {
		t.Fatal("a cursor of a lost log was taken")
	}
	if _, ok := c.since("junk"); ok {
		t.Fatal("junk cursor taken")
	}
}

func TestChangeLogDropsOldDeletes(t *testing.T) {
	c := newChangeLog(t.TempDir(), "login-changes.json")
	first := c.observe("v", map[string]string{"a": "1"})
	c.observe("v", map[string]string{})
	c.f.Logins["a"].At -= (tombTTL + time.Hour).Milliseconds()
	now := c.observe("v", map[string]string{"b": "1"})
	if _, ok := c.since(first); ok {
		t.Fatal("a cursor from before a dropped delete was answered")
	}
	if ch, ok := c.since(now); !ok || len(ch) != 0 {
		t.Fatalf("current cursor: %v %v", ch, ok)
	}
}

func TestScopes(t *testing.T) {
	r := NewRegistry(t.TempDir())
	if _, err := r.Ask(Caller{}, "enclave", "Enclave", "", "files"); err == nil {
		t.Fatal("unknown scope taken")
	}
	q, err := r.Ask(Caller{}, "enclave", "Enclave", "", ScopePasswords, ScopePasswords)
	if err != nil || len(q.Scopes) != 1 {
		t.Fatalf("ask: %+v %v", q, err)
	}
	// Granting what was not asked for gives nothing extra.
	q2, _ := r.Ask(Caller{}, "chat", "Chat", "")
	if err := r.Decide(q2.ID, true, ScopePasswords); err != nil {
		t.Fatal(err)
	}
	if a := r.Await(q2.Ticket(), 0, nil); len(a.Scopes) != 0 {
		t.Fatalf("unasked scope granted: %v", a.Scopes)
	}
	if err := r.Decide(q.ID, true, ScopePasswords); err != nil {
		t.Fatal(err)
	}
	a := r.Await(q.Ticket(), 0, nil)
	p, ok := r.Authenticate(Caller{Token: a.Token})
	if !ok || !p.Has(ScopePasswords) || len(a.Scopes) != 1 {
		t.Fatalf("pairing: %+v %+v", p, a)
	}
	// Asking again for more keeps the old pairing valid while it waits.
	if _, err := r.Ask(Caller{}, "chat", "Chat", "", ScopePasswords); err != nil {
		t.Fatal(err)
	}
	for _, x := range r.Pairings() {
		if x.App == "chat" && x.Has(ScopePasswords) {
			t.Fatal("scope granted before the answer")
		}
	}
	if len(r.Pairings()) != 2 {
		t.Fatal("pairing dropped while a new request waits")
	}
	if err := r.SetScopes(p.ID, nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := NewRegistry(r.Dir()).Authenticate(Caller{Token: a.Token}); p.Has(ScopePasswords) {
		t.Fatal("revoked scope kept after a restart")
	}
}
