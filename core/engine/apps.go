package engine

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cloudsync/core/crypto"
	"cloudsync/core/index"
)

// Records of connected apps (core/connector): small values an app keeps in
// the vault, one OpApp per write, sealed with a key of the app.

const (
	// MaxAppValue is the largest app record. Every write is a whole op in
	// every device's journal (copied to every storage), so records are for
	// settings and vault files, not for bulk data.
	MaxAppValue = 256 << 10
	// MaxAppRecords is how many records one app may have.
	MaxAppRecords = 256
)

var (
	// ErrAppConflict: the record changed since the base revision.
	ErrAppConflict = errors.New("app record: changed since the given revision")
	// ErrAppInvalid: a bad app id, key or value.
	ErrAppInvalid = errors.New("app record: invalid")
)

var (
	appIDRe  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
	appKeyRe = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,200}$`)
)

// ValidAppID reports whether id may name an app.
func ValidAppID(id string) bool { return appIDRe.MatchString(id) }

func checkAppKey(key string) error {
	if !appKeyRe.MatchString(key) || strings.Contains(key, "..") {
		return fmt.Errorf("%w: key %q (1–200 of A-Z a-z 0-9 . _ / -)", ErrAppInvalid, key)
	}
	return nil
}

// AppVersion is one current value of an app record, opened.
type AppVersion struct {
	ID         string
	Data       []byte
	Size       int
	DeviceName string
	Time       time.Time
}

// AppItem is one app record: no versions = it does not exist; more than
// one = concurrent writes the app has to merge.
type AppItem struct {
	Key      string
	Revision string
	Versions []AppVersion
	// Broken counts heads that could not be opened (should not happen).
	Broken int
}

// AppKey is the app's own key: the same on every device of the vault.
func AppKey(master crypto.Key, app string) crypto.Key {
	return master.Derive("connector app key", []byte(app))
}

func appSealKey(master crypto.Key, app string) crypto.Key {
	return master.Derive("connector app data", []byte(app))
}

// appAD binds a sealed value to its record.
func appAD(app, key string) []byte { return []byte("app\x00" + app + "\x00" + key) }

// appsView reads app records of an index.
type appsView struct {
	idx    *index.Index
	master crypto.Key
}

func (v appsView) item(r index.AppRecord, withData bool) AppItem {
	it := AppItem{Key: r.Key, Revision: r.Revision()}
	k := appSealKey(v.master, r.App)
	for _, h := range r.Live() {
		data, err := crypto.Open(k, h.Data, appAD(r.App, r.Key))
		if err != nil {
			it.Broken++
			continue
		}
		av := AppVersion{ID: h.ID, Size: len(data), DeviceName: h.DeviceName, Time: h.Time}
		if withData {
			av.Data = data
		}
		it.Versions = append(it.Versions, av)
	}
	return it
}

// AppGet returns one record of app.
func (v appsView) AppGet(app, key string) (AppItem, error) {
	if !ValidAppID(app) {
		return AppItem{}, fmt.Errorf("%w: app id %q", ErrAppInvalid, app)
	}
	if err := checkAppKey(key); err != nil {
		return AppItem{}, err
	}
	return v.item(v.idx.AppRecord(app, key), true), nil
}

// AppList returns the existing records of app under prefix, without data.
func (v appsView) AppList(app, prefix string) ([]AppItem, error) {
	if !ValidAppID(app) {
		return nil, fmt.Errorf("%w: app id %q", ErrAppInvalid, app)
	}
	var out []AppItem
	for _, r := range v.idx.AppRecords(app) {
		if !strings.HasPrefix(r.Key, prefix) || len(r.Live()) == 0 {
			continue
		}
		out = append(out, v.item(r, false))
	}
	return out, nil
}

// AppRevisions returns the revision of every record of app (deleted ones
// too), for change detection.
func (v appsView) AppRevisions(app string) map[string]AppRevision {
	out := map[string]AppRevision{}
	for _, r := range v.idx.AppRecords(app) {
		out[r.Key] = AppRevision{Revision: r.Revision(), Deleted: len(r.Live()) == 0}
	}
	return out
}

// AppRevision is the state of a record for change detection.
type AppRevision struct {
	Revision string
	Deleted  bool
}

// AppUsage is what one app keeps in the vault.
type AppUsage struct {
	App     string
	Records int
	Bytes   int64 // sealed size of the current values
}

// AppsUsage lists every app that has records.
func (v appsView) AppsUsage() []AppUsage {
	var out []AppUsage
	for _, app := range v.idx.Apps() {
		u := AppUsage{App: app}
		for _, r := range v.idx.AppRecords(app) {
			live := r.Live()
			if len(live) == 0 {
				continue
			}
			u.Records++
			for _, h := range live {
				u.Bytes += int64(len(h.Data))
			}
		}
		if u.Records > 0 {
			out = append(out, u)
		}
	}
	return out
}

func (e *Engine) apps() appsView { return appsView{e.idx, e.cfg.Key()} }

// AppGet returns one record of app.
func (e *Engine) AppGet(app, key string) (AppItem, error) { return e.apps().AppGet(app, key) }

// AppList returns the existing records of app under prefix, without data.
func (e *Engine) AppList(app, prefix string) ([]AppItem, error) {
	return e.apps().AppList(app, prefix)
}

// AppRevisions returns the revision of every record of app.
func (e *Engine) AppRevisions(app string) map[string]AppRevision { return e.apps().AppRevisions(app) }

// AppsUsage lists every app that has records.
func (e *Engine) AppsUsage() []AppUsage { return e.apps().AppsUsage() }

// AppKey returns app's own key.
func (e *Engine) AppKey(app string) crypto.Key { return AppKey(e.cfg.Key(), app) }

// AppPut writes a record of app. base is the revision the value was made
// from; when the record has another revision now, nothing is written and
// the error is ErrAppConflict.
func (e *Engine) AppPut(ctx context.Context, app, key, base string, data []byte) (AppItem, error) {
	if len(data) > MaxAppValue {
		return AppItem{}, fmt.Errorf("%w: %d bytes (at most %d)", ErrAppInvalid, len(data), MaxAppValue)
	}
	return e.appWrite(ctx, app, key, base, data, false)
}

// AppDelete removes a record of app, with the same base rule as AppPut.
func (e *Engine) AppDelete(ctx context.Context, app, key, base string) (AppItem, error) {
	return e.appWrite(ctx, app, key, base, nil, true)
}

// AppDeleteAll removes every record of app (from the CloudSync UI).
func (e *Engine) AppDeleteAll(ctx context.Context, app string) (int, error) {
	if !ValidAppID(app) {
		return 0, fmt.Errorf("%w: app id %q", ErrAppInvalid, app)
	}
	e.appMu.Lock()
	defer e.appMu.Unlock()
	var ops []index.Op
	for _, r := range e.idx.AppRecords(app) {
		if len(r.Live()) == 0 {
			continue
		}
		ops = append(ops, index.Op{Kind: index.OpApp, Path: "/", App: &index.AppChange{
			App: app, Key: r.Key, Parents: index.ParseRevision(r.Revision()), Deleted: true}})
	}
	if len(ops) == 0 {
		return 0, nil
	}
	return len(ops), e.commit(ctx, ops)
}

func (e *Engine) appWrite(ctx context.Context, app, key, base string, data []byte, del bool) (AppItem, error) {
	if !ValidAppID(app) {
		return AppItem{}, fmt.Errorf("%w: app id %q", ErrAppInvalid, app)
	}
	if err := checkAppKey(key); err != nil {
		return AppItem{}, err
	}
	e.appMu.Lock()
	defer e.appMu.Unlock()
	cur := e.idx.AppRecord(app, key)
	if cur.Revision() != base {
		return AppItem{}, ErrAppConflict
	}
	if del && len(cur.Live()) == 0 {
		return e.apps().item(cur, true), nil // nothing to delete
	}
	if !del && len(cur.Heads) == 0 {
		n := 0
		for _, r := range e.idx.AppRecords(app) {
			if len(r.Live()) > 0 {
				n++
			}
		}
		if n >= MaxAppRecords {
			return AppItem{}, fmt.Errorf("%w: at most %d records per app", ErrAppInvalid, MaxAppRecords)
		}
	}
	ch := &index.AppChange{App: app, Key: key, Parents: index.ParseRevision(base), Deleted: del}
	if !del {
		sealed, err := crypto.Seal(appSealKey(e.cfg.Key(), app), data, appAD(app, key))
		if err != nil {
			return AppItem{}, err
		}
		ch.Data = sealed
	}
	ops := []index.Op{{Kind: index.OpApp, Path: "/", App: ch}}
	if err := e.commit(ctx, ops); err != nil {
		return AppItem{}, err
	}
	return e.apps().item(e.idx.AppRecord(app, key), true), nil
}

// KeyID is a short fingerprint of a key, safe to show.
func KeyID(k crypto.Key) string { return hex.EncodeToString(k.MAC([]byte("key id"))[:6]) }
