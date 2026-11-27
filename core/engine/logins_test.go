package engine

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"cloudsync/core/secrets"
)

func TestOrigin(t *testing.T) {
	for in, want := range map[string]string{
		"https://Example.com/login?x=1":   "https://example.com/",
		"example.com":                     "https://example.com/",
		"http://localhost:8080/a":         "http://localhost:8080/",
		"android://AbC=@com.example.app":  "android://AbC=@com.example.app/",
		"android://AbC=@com.example.app/": "android://AbC=@com.example.app/",
		"ssh://host":                      "",
		"":                                "",
		"not a url":                       "",
	} {
		if got := Origin(in); got != want {
			t.Errorf("Origin(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoginsOverSafe(t *testing.T) {
	ctx := context.Background()
	a := newWorld(t).device("laptop")

	// Typed in CloudSync with a URL: a login; without one, or of another
	// kind: not visible.
	ui, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindPassword, Title: "Mail",
		Username: "me", URL: "mail.example.com/login"}, secrets.Data{Password: "pw1", Notes: "n"})
	if err != nil {
		t.Fatal(err)
	}
	note, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindPassword, Title: "PIN"}, secrets.Data{Password: "1234"})
	if err != nil {
		t.Fatal(err)
	}
	list, more := a.LoginList("", "", false)
	if more || len(list) != 1 || list[0].ID != ui.ID || list[0].SignonRealm != "https://mail.example.com/" ||
		list[0].URL != list[0].SignonRealm || list[0].Password != "" || list[0].Username != "" || list[0].HasData {
		t.Fatalf("list without data: %+v", list)
	}
	got := a.LoginGet([]string{ui.ID, note.ID, "nope"})
	if len(got) != 1 || got[0].Username != "me" || got[0].Password != "pw1" || got[0].Notes != "n" ||
		got[0].DeviceName != "laptop" || got[0].PasswordModified.IsZero() {
		t.Fatalf("get: %+v", got)
	}
	if _, err := a.LoginPut(ctx, Login{ID: note.ID, SignonRealm: "https://x.com/"}, note.Version); !errors.Is(err, ErrNotFound) {
		t.Fatalf("an entry that is not a login was changed: %v", err)
	}
	if err := a.LoginDelete(ctx, note.ID, note.Version); !errors.Is(err, ErrNotFound) {
		t.Fatalf("an entry that is not a login was deleted: %v", err)
	}

	// CAS.
	l, err := a.LoginPut(ctx, Login{URL: "https://shop.example/signin", Username: "u", Password: "p1",
		UsernameElement: "email", Extra: map[string][]byte{"bezel.scheme": []byte("1")}}, "")
	if err != nil || l.SignonRealm != "https://shop.example/" || l.URL != "https://shop.example/signin" || l.Revision == "" {
		t.Fatalf("create: %+v %v", l, err)
	}
	if _, err := a.LoginPut(ctx, Login{ID: l.ID, URL: l.URL, Password: "p2"}, ""); !errors.Is(err, ErrLoginConflict) {
		t.Fatalf("stale base: %v", err)
	}
	l2 := l
	l2.Password = "p2"
	l2, err = a.LoginPut(ctx, l2, l.Revision)
	if err != nil || l2.Revision == l.Revision {
		t.Fatalf("update: %+v %v", l2, err)
	}
	if _, err := a.LoginPut(ctx, l, l.Revision); !errors.Is(err, ErrLoginConflict) {
		t.Fatalf("old base after an update: %v", err)
	}

	// Edited in CloudSync (whose UI knows nothing of the browser's
	// fields): they stay.
	en, err := a.OpenSecret(l.ID)
	if err != nil || en.Title != "shop.example" {
		t.Fatalf("title of a browser login: %+v %v", en.Meta, err)
	}
	if _, err := a.PutSecret(ctx, l.ID, l2.Revision, secrets.Meta{Kind: secrets.KindPassword, Title: "Shop",
		Username: "u", URL: en.URL}, secrets.Data{Password: "p2"}); err != nil {
		t.Fatal(err)
	}
	g := a.LoginGet([]string{l.ID})
	if len(g) != 1 || g[0].UsernameElement != "email" || !bytes.Equal(g[0].Extra["bezel.scheme"], []byte("1")) ||
		g[0].PasswordModified != l2.PasswordModified {
		t.Fatalf("after an edit in CloudSync: %+v (was %+v)", g, l2)
	}

	// Blocked sites are not in CloudSync's list.
	b, err := a.LoginPut(ctx, Login{SignonRealm: "https://never.example/", Blocked: true, Username: "x", Password: "y"}, "")
	if err != nil || !b.Blocked || b.Password != "" || b.Username != "" {
		t.Fatalf("blocked: %+v %v", b, err)
	}
	for _, e := range a.Safe().Entries {
		if e.ID == b.ID {
			t.Fatal("a blocked site is in the password list")
		}
	}
	if _, err := a.OpenSecret(b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a blocked site opens as a secret: %v", err)
	}
	if bl := a.LoginGet([]string{b.ID}); len(bl) != 1 || !bl[0].Blocked {
		t.Fatalf("blocked through the API: %+v", bl)
	}

	// Realm filter and delete.
	if l, _ := a.LoginList("https://shop.example/", "", true); len(l) != 1 || l[0].Password != "p2" {
		t.Fatalf("realm filter: %+v", l)
	}
	cur := a.LoginGet([]string{l.ID})[0]
	if err := a.LoginDelete(ctx, l.ID, l.Revision); !errors.Is(err, ErrLoginConflict) {
		t.Fatalf("delete with a stale base: %v", err)
	}
	if err := a.LoginDelete(ctx, l.ID, cur.Revision); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.LoginRevisions()[l.ID]; ok {
		t.Fatal("deleted login still listed")
	}
	if _, err := a.LoginPut(ctx, cur, cur.Revision); !errors.Is(err, ErrLoginConflict) {
		t.Fatalf("write to a deleted login: %v", err)
	}
}

// Logins written on two devices while they could not see each other end
// up on both; a login changed on both is kept twice (a conflict copy is a
// login of its own).
func TestLoginsOfflineTwoDevices(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	a, b := w.device("laptop"), w.device("phone")
	shared, err := a.LoginPut(ctx, Login{URL: "https://site.example/", Username: "u", Password: "p0"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	w.offline(true, "mega1", "mega2", "gd", "s3")
	la, err := a.LoginPut(ctx, Login{URL: "https://a.example/", Username: "a", Password: "pa"}, "")
	if err != nil {
		t.Fatal(err)
	}
	lb, err := b.LoginPut(ctx, Login{URL: "https://b.example/", Username: "b", Password: "pb"}, "")
	if err != nil {
		t.Fatal(err)
	}
	ea, eb := shared, shared
	ea.Password, eb.Password = "from-laptop", "from-phone"
	if _, err := a.LoginPut(ctx, ea, shared.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := b.LoginPut(ctx, eb, shared.Revision); err != nil {
		t.Fatal(err)
	}
	w.offline(false, "mega1", "mega2", "gd", "s3")
	for _, e := range []*Engine{a, b, a} {
		if err := e.Sync(ctx); err != nil {
			t.Fatal(err)
		}
	}
	ra, rb := a.LoginRevisions(), b.LoginRevisions()
	if len(ra) != 4 || len(rb) != 4 {
		t.Fatalf("logins: %v / %v", ra, rb)
	}
	for id, rev := range ra {
		if rb[id] != rev {
			t.Fatalf("devices differ on %s: %v / %v", id, ra, rb)
		}
	}
	if _, ok := ra[la.ID]; !ok {
		t.Fatal("laptop's offline login missing")
	}
	if _, ok := ra[lb.ID]; !ok {
		t.Fatal("phone's offline login missing")
	}
	pw := map[string]bool{}
	for _, l := range a.LoginGet(keys(ra)) {
		if l.SignonRealm == "https://site.example/" {
			pw[l.Password] = true
		}
	}
	if !pw["from-laptop"] || !pw["from-phone"] {
		t.Fatalf("concurrent edits: %v", pw)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
