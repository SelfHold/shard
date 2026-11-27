package engine

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SelfHold/shard/core/vault"
)

func TestAppRecordsSync(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	a, b := w.device("laptop"), w.device("phone")

	if it, err := a.AppGet("parley", "vault.parley"); err != nil || len(it.Versions) != 0 || it.Revision != "" {
		t.Fatalf("empty record: %+v %v", it, err)
	}
	v1, err := a.AppPut(ctx, "parley", "vault.parley", "", []byte("config-1"))
	if err != nil || len(v1.Versions) != 1 || v1.Revision == "" {
		t.Fatalf("put: %+v %v", v1, err)
	}
	// A stale base is refused, nothing is written.
	if _, err := a.AppPut(ctx, "parley", "vault.parley", "", []byte("x")); !errors.Is(err, ErrAppConflict) {
		t.Fatalf("stale base: %v", err)
	}
	// Nothing readable reaches a storage.
	for _, m := range w.mem {
		for name, data := range m.Objects() {
			if bytes.Contains(data, []byte("config-1")) || strings.Contains(name, "parley") {
				t.Fatalf("plaintext in %s", name)
			}
		}
	}

	if err := b.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := b.AppGet("parley", "vault.parley")
	if err != nil || len(got.Versions) != 1 || string(got.Versions[0].Data) != "config-1" || got.Revision != v1.Revision ||
		got.Versions[0].DeviceName != "laptop" {
		t.Fatalf("other device: %+v %v", got, err)
	}
	if a.AppKey("parley") != b.AppKey("parley") || a.AppKey("parley") == a.AppKey("other") {
		t.Fatal("app keys differ between devices or collide between apps")
	}

	// Both write from the same revision before seeing each other: two
	// versions, the app merges them.
	if _, err := a.AppPut(ctx, "parley", "vault.parley", v1.Revision, []byte("from-laptop")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.AppPut(ctx, "parley", "vault.parley", v1.Revision, []byte("from-phone")); err != nil {
		t.Fatal(err)
	}
	for _, e := range []*Engine{a, b, a} {
		if err := e.Sync(ctx); err != nil {
			t.Fatal(err)
		}
	}
	ga, _ := a.AppGet("parley", "vault.parley")
	gb, _ := b.AppGet("parley", "vault.parley")
	if len(ga.Versions) != 2 || ga.Revision != gb.Revision {
		t.Fatalf("concurrent writes: %+v / %+v", ga, gb)
	}
	merged, err := b.AppPut(ctx, "parley", "vault.parley", gb.Revision, []byte("merged"))
	if err != nil || len(merged.Versions) != 1 {
		t.Fatalf("merge: %+v %v", merged, err)
	}
	if err := a.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if ga, _ = a.AppGet("parley", "vault.parley"); len(ga.Versions) != 1 || string(ga.Versions[0].Data) != "merged" {
		t.Fatalf("after merge: %+v", ga)
	}

	// List, usage, delete.
	if _, err := a.AppPut(ctx, "parley", "ui/theme", "", []byte("dark")); err != nil {
		t.Fatal(err)
	}
	l, _ := a.AppList("parley", "")
	if len(l) != 2 || l[0].Key != "ui/theme" || l[1].Key != "vault.parley" || l[0].Versions[0].Data != nil || l[0].Versions[0].Size != 4 {
		t.Fatalf("list: %+v", l)
	}
	if l, _ := a.AppList("parley", "ui/"); len(l) != 1 {
		t.Fatalf("prefix list: %+v", l)
	}
	if u := a.AppsUsage(); len(u) != 1 || u[0].App != "parley" || u[0].Records != 2 {
		t.Fatalf("usage: %+v", u)
	}
	th, _ := a.AppGet("parley", "ui/theme")
	if d, err := a.AppDelete(ctx, "parley", "ui/theme", th.Revision); err != nil || len(d.Versions) != 0 || d.Revision == "" {
		t.Fatalf("delete: %+v %v", d, err)
	}
	if n, err := a.AppDeleteAll(ctx, "parley"); err != nil || n != 1 {
		t.Fatalf("delete all: %d %v", n, err)
	}
	if err := b.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if l, _ := b.AppList("parley", ""); len(l) != 0 {
		t.Fatalf("after delete all: %+v", l)
	}
	// A deleted record is written again from its revision.
	gone, _ := b.AppGet("parley", "vault.parley")
	if _, err := b.AppPut(ctx, "parley", "vault.parley", gone.Revision, []byte("again")); err != nil {
		t.Fatal(err)
	}
}

func TestAppRecordLimits(t *testing.T) {
	ctx := context.Background()
	a := newWorld(t).device("laptop")
	for _, id := range []string{"", "A", "x", "-chat", "chat app", strings.Repeat("a", 65)} {
		if _, err := a.AppPut(ctx, id, "k", "", nil); !errors.Is(err, ErrAppInvalid) {
			t.Errorf("app id %q: %v", id, err)
		}
	}
	for _, k := range []string{"", "a b", "../x", "a..b", strings.Repeat("k", 201), "ключ"} {
		if _, err := a.AppPut(ctx, "chat", k, "", nil); !errors.Is(err, ErrAppInvalid) {
			t.Errorf("key %q: %v", k, err)
		}
	}
	if _, err := a.AppPut(ctx, "chat", "big", "", make([]byte, MaxAppValue+1)); !errors.Is(err, ErrAppInvalid) {
		t.Errorf("big value: %v", err)
	}
	if _, err := a.AppPut(ctx, "chat", "max", "", make([]byte, MaxAppValue)); err != nil {
		t.Errorf("value of max size: %v", err)
	}
}

func TestAppRecordsOffline(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	dir := t.TempDir()
	vp := filepath.Join(dir, "vault.shard")
	os.WriteFile(vp, w.vaultFile, 0o600)
	v, err := vault.Open(vp, []byte("pw"))
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(dir, "data")
	a := w.reopen(v, data, "laptop")
	if _, err := a.AppPut(ctx, "chat", "vault", "", []byte("cfg")); err != nil {
		t.Fatal(err)
	}
	a.Close(ctx)
	o, err := OpenOfflineSafe(v, data)
	if err != nil {
		t.Fatal(err)
	}
	if it, err := o.AppGet("chat", "vault"); err != nil || len(it.Versions) != 1 || string(it.Versions[0].Data) != "cfg" {
		t.Fatalf("offline read: %+v %v", it, err)
	}
}
