package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/SelfHold/shard/core/backend"
	"github.com/SelfHold/shard/core/blob"
	"github.com/SelfHold/shard/core/eventlog"
	"github.com/SelfHold/shard/core/index"
)

// pins are the paths this device keeps fully local. A pinned directory
// covers everything below it, including files added later. Pinning "/"
// is the full local copy. Pins are per device and never synced.
type pins struct {
	file string
	mu   sync.Mutex
	set  map[string]bool
}

func loadPins(file string) (*pins, error) {
	p := &pins{file: file, set: map[string]bool{}}
	data, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return nil, err
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	for _, s := range list {
		p.set[s] = true
	}
	return p, nil
}

func (p *pins) save() error {
	data, _ := json.Marshal(p.list())
	return writeFileAtomic(p.file, data)
}

func (p *pins) list() []string {
	out := make([]string, 0, len(p.set))
	for s := range p.set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func (p *pins) covers(path string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for s := range p.set {
		if s == "/" || path == s || strings.HasPrefix(path, s+"/") {
			return true
		}
	}
	return false
}

// Pin keeps p (a file or a directory tree) local and downloads it now.
func (e *Engine) Pin(ctx context.Context, p string) error {
	if err := e.addPin(p); err != nil {
		return err
	}
	_, err := e.EnsurePinned(ctx)
	return err
}

// PinLater keeps p local; the download runs in the background (see
// PinProgress). Pinning "/" makes this a reference device: a full copy.
func (e *Engine) PinLater(p string) error {
	if err := e.addPin(p); err != nil {
		return err
	}
	e.wakePins()
	return nil
}

func (e *Engine) addPin(p string) error {
	p = index.Clean(p)
	if _, err := e.Stat(p); err != nil {
		return err
	}
	e.pins.mu.Lock()
	defer e.pins.mu.Unlock()
	e.pins.set[p] = true
	return e.pins.save()
}

// Unpin lets the cache drop p again.
func (e *Engine) Unpin(p string) error {
	p = index.Clean(p)
	e.pins.mu.Lock()
	delete(e.pins.set, p)
	err := e.pins.save()
	e.pins.mu.Unlock()
	e.cache.unpinned()
	return err
}

// Pins returns the pinned paths.
func (e *Engine) Pins() []string {
	e.pins.mu.Lock()
	defer e.pins.mu.Unlock()
	return e.pins.list()
}

// IsPinned reports whether p is kept local.
func (e *Engine) IsPinned(p string) bool { return e.pins.covers(index.Clean(p)) }

// IsLocal reports whether the whole content of file p is on this device.
func (e *Engine) IsLocal(p string) bool {
	en, ok := e.idx.Get(p)
	if !ok || en.Version == nil {
		return false
	}
	for _, c := range en.Version.Chunks {
		if !e.cache.has(c.Blob.ID.String()) {
			return false
		}
	}
	return true
}

func (e *Engine) keepPinned() map[string]bool {
	keep := map[string]bool{}
	for id := range e.pinnedBlobs() {
		keep[id] = true
	}
	return keep
}

func (e *Engine) pinnedBlobs() map[string]blob.Ref {
	out := map[string]blob.Ref{}
	e.idx.Walk(func(en index.Entry) {
		if en.Version == nil || !e.pins.covers(en.Path) {
			return
		}
		for _, c := range en.Version.Chunks {
			out[c.Blob.ID.String()] = c.Blob
		}
	})
	return out
}

// PinProgress says how much of the pinned data is on this device.
type PinProgress struct {
	Bytes   int64  // file data of pinned paths (each blob counted once)
	Missing int64  // of it, not downloaded yet
	Running bool   // a download pass is going on
	Err     string // why the last pass did not get everything
}

type pinStatus struct {
	mu sync.Mutex
	PinProgress
}

// PinProgress returns the state of the last (or current) download pass.
func (e *Engine) PinProgress() PinProgress {
	e.pinState.mu.Lock()
	defer e.pinState.mu.Unlock()
	return e.pinState.PinProgress
}

// EnsurePinned downloads every missing blob of pinned files and returns
// how many it fetched.
func (e *Engine) EnsurePinned(ctx context.Context) (int, error) {
	var todo []blob.Ref
	var total, missing int64
	for id, ref := range e.pinnedBlobs() {
		total += int64(ref.Size)
		if !e.cache.has(id) {
			todo = append(todo, ref)
			missing += int64(ref.Size)
		}
	}
	ps := &e.pinState
	ps.mu.Lock()
	ps.Bytes, ps.Missing, ps.Running = total, missing, len(todo) > 0
	if len(todo) == 0 {
		ps.Err = ""
	}
	ps.mu.Unlock()
	if len(todo) == 0 {
		return 0, nil
	}
	var mu sync.Mutex
	fetched := 0
	var errs []error
	backend.Limit(len(todo), pinParallel, func(i int) {
		if ctx.Err() != nil {
			return
		}
		_, err := e.blobData(ctx, todo[i])
		if err == nil {
			ps.mu.Lock()
			ps.Missing -= int64(todo[i].Size)
			ps.mu.Unlock()
		}
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, err)
			return
		}
		fetched++
	})
	if ctx.Err() != nil {
		errs = append(errs, ctx.Err())
	}
	err := errors.Join(errs...)
	ps.mu.Lock()
	ps.Running = false
	ps.Err = ""
	if len(errs) > 0 {
		// One line: the first reason and how many blobs failed.
		ps.Err = errs[0].Error()
		if len(errs) > 1 {
			ps.Err += fmt.Sprintf(" (and %d more)", len(errs)-1)
		}
	}
	ps.mu.Unlock()
	return fetched, err
}

// wakePins asks the background worker for a download pass.
func (e *Engine) wakePins() {
	select {
	case e.pinWake <- struct{}{}:
	default: // a pass is already queued
	}
}

// pinLoop downloads pinned files in the background, so that a large
// first download (a new reference device) does not hold up Sync.
func (e *Engine) pinLoop(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.pinWake:
		}
		n, err := e.EnsurePinned(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			e.opt.Log.Addf(eventlog.Error, "pin", "", "pinned files: %d downloaded, not all: %v", n, err)
		} else if n > 0 {
			e.opt.Log.Addf(eventlog.Info, "pin", "", "pinned files: %d blobs downloaded", n)
		}
	}
}

// pinParallel is how many blobs of pinned files download at the same time.
const pinParallel = 4
