package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SelfHold/shard/core/secrets"
	"github.com/SelfHold/shard/core/vault"
)

func journalNames(w *world, id string) map[string]bool {
	out := map[string]bool{}
	for n := range w.mem[id].Objects() {
		if strings.HasPrefix(n, "j/") {
			out[n] = true
		}
	}
	return out
}

func putSafe(t *testing.T, e *Engine) (pw, totp secrets.Entry) {
	t.Helper()
	ctx := context.Background()
	pw, err := e.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindPassword, Title: "Mail"},
		secrets.Data{Password: "hunter2-very-secret"})
	if err != nil {
		t.Fatal(err)
	}
	totp, err = e.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindTOTP, Title: "GitHub"},
		secrets.Data{TOTP: &secrets.TOTP{Secret: "JBSWY3DPEHPK3PXP"}, RecoveryCodes: []string{"aaaa-bbbb"}})
	if err != nil {
		t.Fatal(err)
	}
	return pw, totp
}

type safeReader interface {
	Safe() SafeList
	OpenSecret(id string) (secrets.Entry, error)
	TOTPCodes(now time.Time) []TOTPCode
}

func checkSafe(t *testing.T, who string, r safeReader, pw, totp secrets.Entry) {
	t.Helper()
	if l := r.Safe(); len(l.Entries) != 2 || l.Broken != 0 {
		t.Fatalf("%s: safe %+v", who, l)
	}
	if got, err := r.OpenSecret(pw.ID); err != nil || got.Data.Password != "hunter2-very-secret" {
		t.Fatalf("%s: password %+v %v", who, got, err)
	}
	if got, err := r.OpenSecret(totp.ID); err != nil || len(got.Data.RecoveryCodes) != 1 {
		t.Fatalf("%s: totp entry %+v %v", who, got, err)
	}
	if c := r.TOTPCodes(time.Now()); len(c) != 1 || c[0].ID != totp.ID || len(c[0].Code) != 6 {
		t.Fatalf("%s: totp codes %+v", who, c)
	}
}

// Every storage gets the whole journal (the safe), never shards of it.
func TestSafeOnEveryStorage(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	a := w.device("laptop")
	w.offline(true, "s3")
	pw, totp := putSafe(t, a)
	w.offline(false, "s3")
	if len(journalNames(w, "s3")) != 0 {
		t.Fatal("offline storage got a segment")
	}
	if err := a.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	want := journalNames(w, "mega1")
	for id := range w.mem {
		got := journalNames(w, id)
		if len(got) != len(want) || len(want) == 0 {
			t.Fatalf("%s holds %d journal objects, mega1 %d", id, len(got), len(want))
		}
	}
	// s3 alone is enough to read the safe on a new device.
	w.offline(true, "mega1", "mega2", "gd")
	b := w.device("phone")
	checkSafe(t, "phone", b, pw, totp)
}

// All storages lose everything: the device still has the safe, and gives
// it back to the storages, so a new device sees it again.
func TestSafeSurvivesLosingEveryStorage(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	dir := t.TempDir()
	vp := filepath.Join(dir, "vault.shard")
	os.WriteFile(vp, w.vaultFile, 0o600)
	v, _ := vault.Open(vp, []byte("pw"))
	a := w.reopen(v, filepath.Join(dir, "data"), "laptop")
	pw, totp := putSafe(t, a)
	a.Close(ctx)
	for _, m := range w.mem {
		m.Wipe()
	}

	a = w.reopen(v, filepath.Join(dir, "data"), "laptop")
	checkSafe(t, "restarted laptop", a, pw, totp)
	if err := a.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	// A new edit continues the numbering instead of reusing a segment name.
	before := journalNames(w, "gd")
	if _, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindPassword, Title: "Bank"}, secrets.Data{Password: "x"}); err != nil {
		t.Fatal(err)
	}
	if after := journalNames(w, "gd"); len(after) != len(before)+1 {
		t.Fatalf("journal objects %d → %d", len(before), len(after))
	}
	b := w.device("phone")
	if l := b.Safe(); len(l.Entries) != 3 {
		t.Fatalf("phone after restore: %+v", l)
	}
	checkSafe(t, "phone", safeOnly{b, pw, totp}, pw, totp)
}

// safeOnly hides entries other than the two of putSafe.
type safeOnly struct {
	*Engine
	pw, totp secrets.Entry
}

func (s safeOnly) Safe() SafeList {
	l := s.Engine.Safe()
	var keep []secrets.Entry
	for _, e := range l.Entries {
		if e.ID == s.pw.ID || e.ID == s.totp.ID {
			keep = append(keep, e)
		}
	}
	l.Entries = keep
	return l
}

// Without any storage (the engine cannot open) the safe is read from the
// device's own copies, including edits never uploaded.
func TestOfflineSafe(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	dir := t.TempDir()
	vp := filepath.Join(dir, "vault.shard")
	os.WriteFile(vp, w.vaultFile, 0o600)
	v, _ := vault.Open(vp, []byte("pw"))
	data := filepath.Join(dir, "data")
	a := w.reopen(v, data, "laptop")
	pw, _ := putSafe(t, a)
	w.offline(true, "mega1", "mega2", "gd", "s3")
	totp, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindTOTP, Title: "Offline 2FA"},
		secrets.Data{TOTP: &secrets.TOTP{Secret: "JBSWY3DPEHPK3PXQ"}})
	if err != nil || a.Pending() != 1 {
		t.Fatalf("offline edit: %v, pending %d", err, a.Pending())
	}
	if err := a.DeleteSecret(ctx, pw.ID); err != nil {
		t.Fatal(err)
	}
	a.Close(ctx)

	o, err := OpenOfflineSafe(v, data)
	if err != nil {
		t.Fatal(err)
	}
	l := o.Safe()
	if len(l.Entries) != 2 {
		t.Fatalf("offline safe: %+v", l)
	}
	if c := o.TOTPCodes(time.Now()); len(c) != 2 {
		t.Fatalf("offline codes: %+v", c)
	}
	if got, err := o.OpenSecret(totp.ID); err != nil || got.Data.TOTP == nil {
		t.Fatalf("offline totp: %+v %v", got, err)
	}
	if _, err := o.OpenSecret(pw.ID); err == nil {
		t.Fatal("deleted entry still there")
	}

	// A data dir of another vault is refused.
	c, _ := vault.New(2, 1)
	op := filepath.Join(dir, "other.shard")
	vault.Create(op, []byte("pw"), c, fastKDF)
	ov, _ := vault.Open(op, []byte("pw"))
	if _, err := OpenOfflineSafe(ov, data); err == nil {
		t.Fatal("other vault's data dir accepted")
	}
	// A data dir never used is an empty safe.
	if o, err := OpenOfflineSafe(v, filepath.Join(dir, "none")); err != nil || len(o.Safe().Entries) != 0 {
		t.Fatalf("empty data dir: %v", err)
	}
}
