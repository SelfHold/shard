// Package engine is the device-side core: it opens the storages from the
// vault, keeps the merged index, uploads and downloads files, caches
// blobs, keeps pinned files local and takes write locks.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/flock"

	"cloudsync/core/backend"
	"cloudsync/core/blob"
	"cloudsync/core/chunker"
	"cloudsync/core/crypto"
	"cloudsync/core/erasure"
	"cloudsync/core/eventlog"
	"cloudsync/core/index"
	"cloudsync/core/lock"
	"cloudsync/core/vault"
)

// Options configure an engine.
type Options struct {
	DataDir    string // per-device state, cache and journal copies
	DeviceName string // shown to other devices (locks, conflict copies)
	Platform   string
	CacheLimit int64 // bytes of unpinned cache; 0 = 2 GiB
	Chunk      chunker.Params
	// Files up to PackThreshold bytes are packed together into blobs of
	// about PackSize bytes, so many small files do not become many objects.
	PackThreshold int
	PackSize      int
	// OpenStore opens a storage; nil uses rclone.
	OpenStore func(ctx context.Context, vaultID string, b vault.Backend) (backend.Store, error)
	// Log gets storage failures (nil: not logged).
	Log *eventlog.Log
}

func (o *Options) defaults() {
	if o.CacheLimit == 0 {
		o.CacheLimit = 2 << 30
	}
	if o.Chunk == (chunker.Params{}) {
		o.Chunk = chunker.Default
	}
	if o.PackThreshold == 0 {
		o.PackThreshold = 1 << 20
	}
	if o.PackSize == 0 {
		o.PackSize = 4 << 20
	}
	if o.DeviceName == "" {
		o.DeviceName, _ = os.Hostname()
	}
}

// Engine is an opened vault on this device.
type Engine struct {
	v      *vault.Vault
	cfg    *vault.Config
	opt    Options
	device vault.Device

	stores    map[string]backend.Store
	storeList []backend.Store
	openErrs  map[string]error

	blobs   *blob.Store
	targets []erasure.Target // storages new shards go to
	space   spaceCache
	idx     *index.Index
	journal *index.Journal
	locks   *lock.Manager
	cache   *cache
	gear    *chunker.Gear
	hashKey crypto.Key

	mu      sync.Mutex // serializes commits
	appMu   sync.Mutex // app writes: base check + commit (apps.go)
	loginMu sync.Mutex // API writes of logins and servers: base check + commit

	entryKinds entryKinds
	state      localState
	pins       *pins
	reload     atomic.Bool // the vault changed: storages must be reopened
	dirLock    *flock.Flock

	fetchMu  sync.Mutex
	fetching map[string]*fetchCall // blob downloads in flight, by id
	ahead    chan struct{}         // slots for read-ahead downloads

	pinState pinStatus
	pinWake  chan struct{} // asks pinLoop for a pass
	pinStop  context.CancelFunc
	pinDone  chan struct{}
}

type localState struct {
	VaultID string     `json:"vault_id"`
	Device  string     `json:"device"`
	Seq     uint64     `json:"seq"`
	Pending []index.Op `json:"pending"` // committed locally, not yet in a journal
}

// Open opens the vault on this device and loads the index from storages.
func Open(ctx context.Context, v *vault.Vault, opt Options) (*Engine, error) {
	opt.defaults()
	if opt.DataDir == "" {
		return nil, errors.New("engine: no data dir")
	}
	if err := os.MkdirAll(opt.DataDir, 0o700); err != nil {
		return nil, err
	}
	// One process per device: the journal numbering, pending changes and
	// lock leases of a device must not be shared by two processes.
	dirLock := flock.New(filepath.Join(opt.DataDir, "lock"))
	if ok, err := dirLock.TryLock(); err != nil || !ok {
		return nil, fmt.Errorf("engine: %s is in use by another process (is cloudsyncd running?)", opt.DataDir)
	}
	e := &Engine{
		v: v, opt: opt, stores: map[string]backend.Store{}, openErrs: map[string]error{}, dirLock: dirLock,
		fetching: map[string]*fetchCall{}, ahead: make(chan struct{}, readAheadParallel),
		pinWake: make(chan struct{}, 1),
	}
	ok := false
	defer func() {
		if !ok {
			dirLock.Unlock()
		}
	}()
	if err := e.loadState(); err != nil {
		return nil, err
	}
	cfg := v.Config()
	if e.state.VaultID != "" && e.state.VaultID != cfg.VaultID {
		return nil, fmt.Errorf("engine: %s belongs to another vault", opt.DataDir)
	}
	e.state.VaultID = cfg.VaultID
	if e.state.Device == "" {
		e.state.Device = crypto.RandomID(8)
	}
	if err := e.saveState(); err != nil {
		return nil, err
	}
	if err := e.registerDevice(); err != nil {
		return nil, err
	}
	e.cfg = v.Config()
	for _, d := range e.cfg.Devices {
		if d.ID == e.state.Device {
			e.device = d
		}
	}

	key := e.cfg.Key()
	e.hashKey = key.Derive("file-hash", nil)
	e.gear = chunker.NewGear(key)
	if err := e.openStores(ctx); err != nil {
		return nil, err
	}
	// A newer vault in the storages (a storage added elsewhere) is adopted
	// before anything is read, so shards on new storages are reachable.
	if _ = withinSync(ctx, e.syncVault); e.reload.Load() {
		e.closeStores()
		e.stores, e.storeList, e.openErrs = map[string]backend.Store{}, nil, map[string]error{}
		e.cfg = v.Config()
		e.reload.Store(false)
		if err := e.openStores(ctx); err != nil {
			return nil, err
		}
	}

	var targets []erasure.Target
	for _, b := range e.cfg.ActiveBackends() {
		if e.stores[b.ID] != nil {
			targets = append(targets, erasure.Target{ID: b.ID, Domain: b.FailureDomain()})
		}
	}
	e.targets = targets
	var err error
	if e.blobs, err = blob.New(key, e.cfg.DataShards, e.cfg.ParityShards, targets, e.stores); err != nil {
		e.closeStores()
		return nil, err
	}
	e.idx = index.New()
	e.journal = index.NewJournal(e.state.Device, key, e.storeList, e.cfg.ParityShards+1)
	e.journal.CacheDir = filepath.Join(opt.DataDir, "journal")
	e.locks = lock.NewManager(e.state.Device, e.device.Name, key, e.storeList)
	e.locks.Run()
	e.idx.Add(e.state.Pending...)
	// The local journal copies first: the index (and the password safe in
	// it) is complete even when no storage answers.
	e.addOps(e.journal.LoadCached())
	if e.cache, err = newCache(filepath.Join(opt.DataDir, "cache"), opt.CacheLimit); err != nil {
		return nil, err
	}
	if e.pins, err = loadPins(filepath.Join(opt.DataDir, "pins.json")); err != nil {
		return nil, err
	}
	e.cache.keep = e.keepPinned
	if err := withinSync(ctx, func(ctx context.Context) error { _, err := e.pull(ctx); return err }); err != nil {
		// Offline start is fine: the local journal copies are used later.
		e.openErrs["journal"] = err
	}
	_ = withinSync(ctx, e.syncVault) // best effort: spreads config changes, marks storages as ours
	pinCtx, stop := context.WithCancel(context.Background())
	e.pinStop, e.pinDone = stop, make(chan struct{})
	go e.pinLoop(pinCtx, e.pinDone)
	ok = true
	return e, nil
}

func (e *Engine) registerDevice() error {
	cfg := e.v.Config()
	for _, d := range cfg.Devices {
		if d.ID == e.state.Device {
			if d.Name == e.opt.DeviceName || e.opt.DeviceName == "" {
				return nil
			}
		}
	}
	return e.v.Update(func(c *vault.Config) error {
		for i, d := range c.Devices {
			if d.ID == e.state.Device {
				c.Devices[i].Name = e.opt.DeviceName
				return nil
			}
		}
		c.Devices = append(c.Devices, vault.Device{
			ID: e.state.Device, Name: e.opt.DeviceName, Platform: e.opt.Platform, Added: time.Now().UTC(),
		})
		return nil
	})
}

func (e *Engine) openStores(ctx context.Context) error {
	open := e.opt.OpenStore
	if open == nil {
		open = func(ctx context.Context, vaultID string, b vault.Backend) (backend.Store, error) {
			return backend.OpenRclone(ctx, vaultID, b, e.saveParam)
		}
	}
	bs := e.cfg.Backends
	opened := make([]backend.Store, len(bs))
	errs := make([]error, len(bs))
	backend.Limit(len(bs), len(bs), func(i int) {
		opened[i], errs[i] = openWithin(ctx, StoreOpenTimeout, func(ctx context.Context) (backend.Store, error) {
			return open(ctx, e.cfg.VaultID, backend.SharePacing(bs[i], bs))
		})
	})
	for i, b := range bs {
		if errs[i] != nil {
			e.openErrs[b.ID] = errs[i]
			e.opt.Log.Addf(eventlog.Error, "storage", b.ID, "%s: cannot open: %v", b.Name, errs[i])
			continue
		}
		s := backend.Logged(backend.Measured(opened[i], backend.DefaultParallel), b.Name, e.opt.Log)
		e.stores[b.ID] = s
		// A disabled storage is only a source of shards for reads and
		// Repair: the journal, locks and vault copies leave it alone, so
		// one that is gone for good is not asked on every sync.
		if !b.Disabled {
			e.storeList = append(e.storeList, s)
		}
	}
	if len(e.storeList) == 0 {
		return errors.New("engine: no storage could be opened")
	}
	return nil
}

// StoreOpenTimeout and OpenSyncTimeout bound the network part of Open: a
// storage that does not answer at all (a dead provider) is left out until
// the next reopen instead of holding up the vault.
var (
	StoreOpenTimeout = 30 * time.Second
	OpenSyncTimeout  = time.Minute
)

// openWithin runs open, giving up after d even if open ignores its context
// (a store that comes back later is closed).
func openWithin(ctx context.Context, d time.Duration, open func(context.Context) (backend.Store, error)) (backend.Store, error) {
	ctx, cancel := context.WithTimeout(ctx, d)
	type result struct {
		s   backend.Store
		err error
	}
	done := make(chan result, 1)
	go func() {
		s, err := open(ctx)
		done <- result{s, err}
	}()
	select {
	case r := <-done:
		cancel()
		return r.s, r.err
	case <-ctx.Done():
		go func() {
			defer cancel()
			if r := <-done; r.s != nil {
				if c, ok := r.s.(interface{ Close() error }); ok {
					_ = c.Close()
				}
			}
		}()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("no answer in %s", d)
		}
		return nil, ctx.Err()
	}
}

// withinSync runs fn with the network time limit of Open.
func withinSync[T any](ctx context.Context, fn func(context.Context) T) T {
	ctx, cancel := context.WithTimeout(ctx, OpenSyncTimeout)
	defer cancel()
	return fn(ctx)
}

// saveParam stores an option rclone changed (a refreshed OAuth token).
func (e *Engine) saveParam(id, key, value string) {
	go func() {
		_ = e.v.Update(func(c *vault.Config) error {
			for i, b := range c.Backends {
				if b.ID == id {
					if c.Backends[i].Params == nil {
						c.Backends[i].Params = map[string]string{}
					}
					c.Backends[i].Params[key] = value
				}
			}
			return nil
		})
	}()
}

func (e *Engine) closeStores() {
	for _, s := range e.stores {
		if c, ok := s.(interface{ Close() error }); ok {
			_ = c.Close()
		}
	}
}

// Close releases locks and stops background work.
func (e *Engine) Close(ctx context.Context) {
	e.pinStop()
	<-e.pinDone
	e.locks.Close(ctx)
	e.closeStores()
	e.dirLock.Unlock()
}

// DeviceID returns this device's id.
func (e *Engine) DeviceID() string { return e.state.Device }

// Device returns this device's vault entry.
func (e *Engine) Device() vault.Device { return e.device }

// ReloadNeeded reports that a newer vault was adopted from a storage and
// the engine must be reopened to use it.
func (e *Engine) ReloadNeeded() bool { return e.reload.Load() }

// Index returns the merged index (read only use).
func (e *Engine) Index() *index.Index { return e.idx }

// Config returns the vault config this engine was opened with.
func (e *Engine) Config() *vault.Config { return e.cfg }

// Sync pulls changes from other devices, pushes local ones that are not
// uploaded yet, exchanges the vault copy and starts downloading pinned
// files in the background (EnsurePinned waits for them).
func (e *Engine) Sync(ctx context.Context) error {
	var errs []error
	if _, err := e.pull(ctx); err != nil {
		errs = append(errs, err)
	}
	// Before flush: Spread uses the listing of this pull, which a new
	// segment would make stale.
	if _, err := e.journal.Spread(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := e.flush(ctx); err != nil {
		errs = append(errs, err)
	} else if _, err := e.journal.Compact(ctx, compactAfter, false); err != nil {
		errs = append(errs, err)
	}
	if err := e.syncVault(ctx); err != nil {
		errs = append(errs, err)
	}
	e.wakePins()
	return errors.Join(errs...)
}

// pull merges new journal segments and reports whether the tree changed.
func (e *Engine) pull(ctx context.Context) (bool, error) {
	ops, err := e.journal.Pull(ctx)
	return e.addOps(ops), err
}

// addOps merges ops read from a journal into the index.
func (e *Engine) addOps(ops []index.Op) bool {
	e.mu.Lock()
	for _, o := range ops {
		if o.Device == e.state.Device && o.Seq > e.state.Seq {
			e.state.Seq = o.Seq // a reinstalled device continues its numbering
		}
	}
	e.mu.Unlock()
	return e.idx.Add(ops...)
}

func (e *Engine) flush(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.state.Pending) == 0 {
		return nil
	}
	if err := e.journal.Append(ctx, e.state.Pending); err != nil {
		return err
	}
	e.state.Pending = nil
	return e.saveState()
}

// commit records ops made by build: it assigns seq and Lamport numbers,
// applies them locally at once and uploads them when storages answer.
func (e *Engine) commit(ctx context.Context, ops []index.Op) error {
	e.mu.Lock()
	lam := e.idx.Lamport()
	now := time.Now().UTC()
	for i := range ops {
		e.state.Seq++
		lam++
		ops[i].Device, ops[i].Seq, ops[i].Lamport, ops[i].Time = e.state.Device, e.state.Seq, lam, now
		ops[i].DeviceName = e.device.Name
	}
	e.idx.Add(ops...)
	e.state.Pending = append(e.state.Pending, ops...)
	err := e.saveState()
	e.mu.Unlock()
	if err != nil {
		return err
	}
	// Not uploaded is not lost: Sync retries pending ops.
	_ = e.flush(ctx)
	return nil
}

// Pending returns how many local changes are not uploaded yet.
func (e *Engine) Pending() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.state.Pending)
}

func (e *Engine) statePath() string { return filepath.Join(e.opt.DataDir, "state.json") }

func (e *Engine) loadState() error {
	data, err := os.ReadFile(e.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &e.state)
}

func (e *Engine) saveState() error {
	data, err := json.Marshal(e.state)
	if err != nil {
		return err
	}
	return writeFileAtomic(e.statePath(), data)
}

const vaultObject = "vault.bin"

// syncVault adopts a newer vault copy from storages and uploads ours where
// it is missing or older. Changing storages needs a reopen (see Status).
// Every storage is read once, all at the same time. A storage that does
// not answer is not a sync failure (its status and log show it; it gets
// the copy next time); only no answer at all is.
func (e *Engine) syncVault(ctx context.Context) error {
	copies := make([][]byte, len(e.storeList))
	getErrs := make([]error, len(e.storeList))
	backend.Limit(len(e.storeList), len(e.storeList), func(i int) {
		copies[i], getErrs[i] = e.storeList[i].Get(ctx, vaultObject)
	})
	var errs, unreachable []error
	for i, s := range e.storeList {
		if getErrs[i] != nil {
			if !errors.Is(getErrs[i], backend.ErrNotFound) {
				unreachable = append(unreachable, getErrs[i])
			}
			continue
		}
		if changed, err := e.v.Merge(copies[i]); err != nil {
			errs = append(errs, fmt.Errorf("vault copy on %s: %w", s.ID(), err))
		} else if changed {
			e.reload.Store(true)
		}
	}
	mine, err := e.v.Encode()
	if err != nil {
		return err
	}
	cur := e.v.Config().Version
	var mu sync.Mutex
	backend.Limit(len(e.storeList), len(e.storeList), func(i int) {
		switch {
		case getErrs[i] == nil:
			if ver, err := e.v.VersionOf(copies[i]); err == nil && ver >= cur {
				return
			}
		case !errors.Is(getErrs[i], backend.ErrNotFound):
			return // unreachable: try next time
		}
		if err := e.storeList[i].Put(ctx, vaultObject, mine); err != nil {
			mu.Lock()
			unreachable = append(unreachable, err)
			mu.Unlock()
		}
	})
	if len(e.storeList) > 0 && len(unreachable) >= len(e.storeList) {
		errs = append(errs, fmt.Errorf("vault copy: no storage reachable: %w", errors.Join(unreachable...)))
	}
	return errors.Join(errs...)
}

// BackendStatus describes one storage.
type BackendStatus struct {
	ID, Name, Type, Domain string
	Online                 bool
	// Behind: reachable but missing journal segments other storages have
	// (a new storage, or one that lost data). Repair fills it.
	Behind bool
	Error  string
	// Recent response times (0: no call finished yet).
	Get, Put, List time.Duration
}

// Status is a summary for the UI.
type Status struct {
	Device       vault.Device
	DataShards   int
	ParityShards int
	Backends     []BackendStatus
	Pending      int
	ReloadNeeded bool
	// DomainRisk: one provider holds more than ParityShards shards of a
	// blob, so its outage alone makes files unreadable.
	DomainRisk bool
}

// Status checks every storage (one listing each) and reports.
func (e *Engine) Status(ctx context.Context) Status {
	st := Status{
		Device: e.device, DataShards: e.cfg.DataShards, ParityShards: e.cfg.ParityShards,
		Pending: e.Pending(), ReloadNeeded: e.reload.Load(),
	}
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	counts := e.journal.Counts(c)
	max := 0
	for _, n := range counts {
		if n > max {
			max = n
		}
	}
	st.Backends = make([]BackendStatus, len(e.cfg.Backends))
	for i, b := range e.cfg.Backends {
		st.Backends[i] = BackendStatus{ID: b.ID, Name: b.Name, Type: b.Type, Domain: b.FailureDomain()}
		if e.stores[b.ID] == nil {
			if err := e.openErrs[b.ID]; err != nil {
				st.Backends[i].Error = eventlog.Clean(err.Error())
			}
			continue
		}
		stats := backend.StatsOf(e.stores[b.ID])
		st.Backends[i].Get, st.Backends[i].Put, st.Backends[i].List = stats.Get, stats.Put, stats.List
		if b.Disabled {
			continue // not listed (see openStores); shown as disabled
		}
		n, ok := counts[b.ID]
		st.Backends[i].Online = ok
		st.Backends[i].Behind = ok && n < max && !b.Disabled
		if !ok {
			st.Backends[i].Error = "not reachable"
			if last := e.opt.Log.Entries(b.ID, 1); len(last) > 0 {
				st.Backends[i].Error = last[0].Message
			}
		}
	}
	st.DomainRisk = DomainRisk(e.cfg)
	return st
}

// DomainRisk reports whether, with the vault's storages and k+m, one
// failure domain (provider) can hold more than m shards of a blob.
func DomainRisk(c *vault.Config) bool {
	var targets []erasure.Target
	for _, b := range c.ActiveBackends() {
		targets = append(targets, erasure.Target{ID: b.ID, Domain: b.FailureDomain()})
	}
	n := c.DataShards + c.ParityShards
	p, err := erasure.Place([]byte("probe"), targets, n)
	if err != nil {
		return true
	}
	return erasure.MaxPerDomain(p, targets) > c.ParityShards
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
