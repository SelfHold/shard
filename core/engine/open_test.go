package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SelfHold/shard/core/backend"
	"github.com/SelfHold/shard/core/vault"
)

// stuckStore never answers: every call waits until its context ends (a
// provider that accepts connections and then stays silent).
type stuckStore struct{ *backend.MemStore }

func (s stuckStore) Get(ctx context.Context, _ string) ([]byte, error) { <-ctx.Done(); return nil, ctx.Err() }
func (s stuckStore) Put(ctx context.Context, _ string, _ []byte) error  { <-ctx.Done(); return ctx.Err() }
func (s stuckStore) List(ctx context.Context, _ string) ([]string, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s stuckStore) ListDirs(ctx context.Context, _ string) ([]string, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// A dead storage, at open or on every call, does not hold up Open.
func TestOpenWithDeadStorage(t *testing.T) {
	oldOpen, oldSync := StoreOpenTimeout, OpenSyncTimeout
	StoreOpenTimeout, OpenSyncTimeout = 200*time.Millisecond, 300*time.Millisecond
	t.Cleanup(func() { StoreOpenTimeout, OpenSyncTimeout = oldOpen, oldSync })

	w := newWorld(t)
	dir := t.TempDir()
	vp := filepath.Join(dir, "vault.shard")
	os.WriteFile(vp, w.vaultFile, 0o600)
	v, err := vault.Open(vp, []byte("pw"))
	if err != nil {
		t.Fatal(err)
	}
	block := make(chan struct{})
	defer close(block)
	start := time.Now()
	e, err := Open(context.Background(), v, Options{
		DataDir: filepath.Join(dir, "data"),
		OpenStore: func(_ context.Context, _ string, b vault.Backend) (backend.Store, error) {
			switch b.ID {
			case "mega1": // ignores its context, like a stuck client
				<-block
				return nil, context.Canceled
			case "gd":
				return stuckStore{w.mem[b.ID]}, nil
			}
			return w.mem[b.ID], nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close(context.Background())
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("open took %s", took)
	}
	if e.stores["mega1"] != nil || e.openErrs["mega1"] == nil {
		t.Fatalf("stuck storage opened: %v", e.openErrs)
	}
	if len(e.storeList) != 3 {
		t.Fatalf("storages: %d", len(e.storeList))
	}
}

// Too few storages open for k+m: Open says which ones failed and why, and
// which ones opened, so the app can show it per storage.
func TestOpenTooFewStorages(t *testing.T) {
	w := newWorld(t)
	dir := t.TempDir()
	vp := filepath.Join(dir, "vault.shard")
	os.WriteFile(vp, w.vaultFile, 0o600)
	v, err := vault.Open(vp, []byte("pw"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Open(context.Background(), v, Options{
		DataDir: filepath.Join(dir, "data"),
		OpenStore: func(_ context.Context, _ string, b vault.Backend) (backend.Store, error) {
			if b.Type == "mega" {
				return nil, errors.New("no answer in 30s")
			}
			return w.mem[b.ID], nil
		},
	})
	var se *StoragesError
	if !errors.As(err, &se) {
		t.Fatalf("open with 2 of 4 storages: %v", err)
	}
	if se.Opened != 2 || se.Enabled != 4 || se.Needed != 3 || len(se.Failed) != 2 || se.Failed["mega1"] == nil || len(se.OK) != 2 {
		t.Fatalf("storages error: %+v", se)
	}
	if want := "2 of 4 storages opened, k+m=3 needed; not opened: Mega personal, Mega work"; err.Error() != want {
		t.Fatalf("message %q", err)
	}
}

// Cancelling Open (the vault locked meanwhile) ends it promptly.
func TestOpenCancelled(t *testing.T) {
	w := newWorld(t)
	dir := t.TempDir()
	vp := filepath.Join(dir, "vault.shard")
	os.WriteFile(vp, w.vaultFile, 0o600)
	v, _ := vault.Open(vp, []byte("pw"))
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	start := time.Now()
	e, err := Open(ctx, v, Options{
		DataDir: filepath.Join(dir, "data"),
		OpenStore: func(_ context.Context, _ string, b vault.Backend) (backend.Store, error) {
			return stuckStore{w.mem[b.ID]}, nil
		},
	})
	if e != nil {
		e.Close(context.Background())
	}
	_ = err
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("cancelled open took %s", took)
	}
}
