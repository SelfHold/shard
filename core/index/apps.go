package index

import (
	"sort"
	"strings"
	"time"
)

// OpApp writes or deletes one record of a connected app (see
// core/connector). The data is sealed by the engine; the index only folds.
//
// A record is a multi-value register: its value is the set of heads, the
// writes no later write of the same record lists as a parent. A write
// based on every head replaces them all; writes made concurrently (offline
// devices) all stay heads until the app merges them. Parents always come
// earlier in the fold order (their Lamport time is lower), so the heads
// depend only on the set of ops.
const OpApp Kind = "app"

// AppChange is the payload of an OpApp.
type AppChange struct {
	App     string   `json:"app"`
	Key     string   `json:"key"`
	Parents []string `json:"parents,omitempty"` // op ids of the heads it replaces
	Deleted bool     `json:"deleted,omitempty"`
	Data    []byte   `json:"data,omitempty"` // sealed
}

// AppVersion is one head of an app record.
type AppVersion struct {
	ID         string // op id
	Deleted    bool
	Data       []byte // sealed
	Device     string
	DeviceName string
	Time       time.Time
}

// AppRecord is the current state of one app record.
type AppRecord struct {
	App, Key string
	// Heads in op id order, deleted ones included (they still have to be
	// replaced by the next write).
	Heads []AppVersion
}

// Revision names the set of heads: "" for a record never written.
func (r AppRecord) Revision() string {
	ids := make([]string, len(r.Heads))
	for i, h := range r.Heads {
		ids[i] = h.ID
	}
	return strings.Join(ids, ",")
}

// Live returns the heads that are not deletes.
func (r AppRecord) Live() []AppVersion {
	var out []AppVersion
	for _, h := range r.Heads {
		if !h.Deleted {
			out = append(out, h)
		}
	}
	return out
}

// ParseRevision splits a revision into op ids.
func ParseRevision(rev string) []string {
	if rev == "" {
		return nil
	}
	return strings.Split(rev, ",")
}

type appSlot map[string]AppVersion // op id → head

func (x *Index) applyApp(o Op) {
	c := o.App
	if c == nil || c.App == "" || c.Key == "" {
		return
	}
	byKey := x.apps[c.App]
	if byKey == nil {
		byKey = map[string]appSlot{}
		x.apps[c.App] = byKey
	}
	slot := byKey[c.Key]
	if slot == nil {
		slot = appSlot{}
		byKey[c.Key] = slot
	}
	for _, p := range c.Parents {
		delete(slot, p)
	}
	slot[o.ID()] = AppVersion{
		ID: o.ID(), Deleted: c.Deleted, Data: c.Data,
		Device: o.Device, DeviceName: o.DeviceName, Time: o.Time,
	}
}

func (s appSlot) record(app, key string) AppRecord {
	r := AppRecord{App: app, Key: key}
	for _, v := range s {
		r.Heads = append(r.Heads, v)
	}
	sort.Slice(r.Heads, func(i, j int) bool { return r.Heads[i].ID < r.Heads[j].ID })
	return r
}

// AppRecord returns one record of app; a record never written has no heads.
func (x *Index) AppRecord(app, key string) AppRecord {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.apps[app][key].record(app, key)
}

// AppRecords returns every record of app (deleted ones too), by key.
func (x *Index) AppRecords(app string) []AppRecord {
	x.mu.RLock()
	defer x.mu.RUnlock()
	out := make([]AppRecord, 0, len(x.apps[app]))
	for k, s := range x.apps[app] {
		out = append(out, s.record(app, k))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Apps returns the ids of apps that have records, sorted.
func (x *Index) Apps() []string {
	x.mu.RLock()
	defer x.mu.RUnlock()
	out := make([]string, 0, len(x.apps))
	for a := range x.apps {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
