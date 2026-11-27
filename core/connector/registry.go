// Package connector serves the Connector API (proto/connector.proto) to
// other applications: an app pairs with CloudSync once per device (the
// user approves it in CloudSync) and then keeps small records in the vault
// (engine/apps.go), the same on every device.
//
// Pairings are per device and live in <data root>/apps.json: a desktop app
// is known by a token (only its SHA-256 is kept), an Android app by its
// package and signing certificate, which the bound service checks.
package connector

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"

	"cloudsync/core/crypto"
	"cloudsync/core/engine"
)

const (
	// RequestTTL is how long a pairing request waits for the user.
	RequestTTL = 5 * time.Minute
	// maxPending caps requests nobody answered (any local process may ask).
	maxPending = 8
)

// ScopePasswords lets an app read and change the logins of the password
// store (docs/CONNECTOR.md, "Password store").
const ScopePasswords = "passwords"

// ScopeSSHKeys lets an app read the SSH keys of the password safe
// (docs/CONNECTOR.md, "SSH keys").
const ScopeSSHKeys = "ssh_keys"

// ScopeServers lets an app read and change the user's servers
// (docs/CONNECTOR.md, "Servers").
const ScopeServers = "servers"

var knownScopes = []string{ScopePasswords, ScopeSSHKeys, ScopeServers}

// SecretScopes are the scopes that open secrets, in the order the UI
// names them.
var SecretScopes = []string{ScopePasswords, ScopeSSHKeys, ScopeServers}

// cleanScopes checks requested scopes; the result is sorted, without
// repeats.
func cleanScopes(in []string) ([]string, error) {
	var out []string
	for _, s := range in {
		if !slices.Contains(knownScopes, s) {
			return nil, fmt.Errorf("%w: unknown scope %q", ErrInvalid, s)
		}
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out, nil
}

// within returns the scopes of want that are in have.
func within(want, have []string) []string {
	var out []string
	for _, s := range have {
		if slices.Contains(want, s) && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// Caller is who calls: a desktop app by its token, an Android app by the
// package and certificate the bound service read from Binder.
type Caller struct {
	Token   string
	Package string
	Cert    string // SHA-256 of the signing certificate, hex
}

func (c Caller) android() bool { return c.Package != "" }

// Pairing is one approved app on this device.
type Pairing struct {
	ID          string    `json:"id"`
	App         string    `json:"app"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	TokenHash   string    `json:"token_hash,omitempty"` // desktop
	Package     string    `json:"package,omitempty"`    // android
	Cert        string    `json:"cert,omitempty"`       // android
	Approved    time.Time `json:"approved"`
	// Scopes granted besides the app's own records (ScopePasswords, ...).
	Scopes []string `json:"scopes,omitempty"`
	// SecretsReadMs is when it last read passwords (unix ms; 0 = never).
	SecretsReadMs int64 `json:"secrets_read_ms,omitempty"`
	// ReadMs is when it last read the secrets of the other scopes (SSH
	// private keys, server passwords), by scope.
	ReadMs map[string]int64 `json:"read_ms,omitempty"`
	// LastUsed is kept in memory only.
	LastUsed time.Time `json:"-"`
}

// Has reports whether scope was granted.
func (p Pairing) Has(scope string) bool { return slices.Contains(p.Scopes, scope) }

// Platform is "android" or "desktop".
func (p Pairing) Platform() string {
	if p.Package != "" {
		return "android"
	}
	return "desktop"
}

// Request is a pairing request waiting for the user.
type Request struct {
	ID          string // shown to the CloudSync UI (not the ticket)
	App         string
	Name        string
	Description string
	Code        string
	Package     string
	Cert        string
	Scopes      []string // asked for; the user may grant fewer
	Created     time.Time
	Expires     time.Time

	ticket  string
	status  Status
	token   string   // desktop: handed out once by Await
	granted []string // scopes the user allowed
	done    chan struct{}
}

// Status of a pairing request.
type Status int

const (
	Pending Status = iota
	Approved
	Denied
	Expired
)

var (
	ErrUnknown = errors.New("connector: no such pairing or request")
	ErrInvalid = errors.New("connector: invalid request")
	ErrTooMany = errors.New("connector: too many pairing requests waiting")
)

// Registry keeps the pairings of this device and the requests waiting.
type Registry struct {
	path string
	// OnChange runs (in its own goroutine) after requests or pairings
	// changed: the host tells the CloudSync UI.
	OnChange func()

	mu       sync.Mutex
	pairings []*Pairing
	requests map[string]*Request // by ticket
	watchers map[chan struct{}]struct{}
	now      func() time.Time
}

// NewRegistry loads the pairings kept in dir/apps.json.
func NewRegistry(dir string) *Registry {
	r := &Registry{path: filepath.Join(dir, "apps.json"), requests: map[string]*Request{},
		watchers: map[chan struct{}]struct{}{}, now: time.Now}
	if data, err := os.ReadFile(r.path); err == nil {
		var f struct {
			Pairings []*Pairing `json:"pairings"`
		}
		if json.Unmarshal(data, &f) == nil {
			r.pairings = f.Pairings
		}
	}
	return r
}

func (r *Registry) save() error {
	data, _ := json.MarshalIndent(map[string]any{"pairings": r.pairings}, "", "  ")
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

// changed wakes watchers and the host; r.mu is held.
func (r *Registry) changed() {
	for ch := range r.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	if r.OnChange != nil {
		go r.OnChange()
	}
}

// watch returns a channel that gets a value after every change.
func (r *Registry) watch() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	r.mu.Lock()
	r.watchers[ch] = struct{}{}
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		delete(r.watchers, ch)
		r.mu.Unlock()
	}
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

// Authenticate returns the pairing of c.
func (r *Registry) Authenticate(c Caller) (Pairing, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.find(c)
	if p == nil {
		return Pairing{}, false
	}
	p.LastUsed = r.now()
	return *p, true
}

// find: r.mu is held.
func (r *Registry) find(c Caller) *Pairing {
	switch {
	case c.android():
		for _, p := range r.pairings {
			if p.Package == c.Package && p.Cert == c.Cert {
				return p
			}
		}
	case c.Token != "":
		h := hashToken(c.Token)
		for _, p := range r.pairings {
			if p.TokenHash != "" && subtle.ConstantTimeCompare([]byte(p.TokenHash), []byte(h)) == 1 {
				return p
			}
		}
	}
	return nil
}

// expire drops requests past their time: r.mu is held.
func (r *Registry) expire() {
	now := r.now()
	for t, q := range r.requests {
		if q.status == Pending && now.After(q.Expires) {
			q.status = Expired
			close(q.done)
		}
		// Answered requests stay a little for Await to pick up the answer.
		if q.status != Pending && now.After(q.Expires.Add(time.Minute)) {
			delete(r.requests, t)
		}
	}
}

func sixDigits() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// Ask records a pairing request for scopes besides the app's own
// records. A newer request of the same app from the same kind of caller
// replaces an older one; an existing pairing stays valid until a new one
// is approved.
func (r *Registry) Ask(c Caller, app, name, desc string, scopes ...string) (*Request, error) {
	if !engine.ValidAppID(app) {
		return nil, fmt.Errorf("%w: app id %q (^[a-z0-9][a-z0-9._-]{1,63}$)", ErrInvalid, app)
	}
	scopes, err := cleanScopes(scopes)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = app
	}
	if len(name) > 64 || len(desc) > 200 {
		return nil, fmt.Errorf("%w: name or description too long", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire()
	pending := 0
	for t, q := range r.requests {
		if q.status != Pending {
			continue
		}
		if q.App == app && q.Package == c.Package {
			q.status = Expired
			close(q.done)
			delete(r.requests, t)
			continue
		}
		pending++
	}
	if pending >= maxPending {
		return nil, ErrTooMany
	}
	now := r.now()
	q := &Request{
		ID: crypto.RandomID(8), App: app, Name: name, Description: desc, Code: sixDigits(),
		Package: c.Package, Cert: c.Cert, Scopes: scopes, Created: now, Expires: now.Add(RequestTTL),
		ticket: crypto.RandomID(24), done: make(chan struct{}),
	}
	r.requests[q.ticket] = q
	r.changed()
	return q, nil
}

// Ticket is the secret the requester waits with.
func (q *Request) Ticket() string { return q.ticket }

// Answer is what Await returns.
type Answer struct {
	Status Status
	Token  string // desktop, approved: handed out once
	App    string
	Scopes []string // granted (approved)
}

// Await waits up to wait for the answer to the request with ticket. The
// token of an approved desktop request is returned once.
func (r *Registry) Await(ticket string, wait time.Duration, cancel <-chan struct{}) Answer {
	r.mu.Lock()
	r.expire()
	q := r.requests[ticket]
	r.mu.Unlock()
	if q == nil {
		return Answer{Status: Expired}
	}
	if wait > 0 {
		t := time.NewTimer(wait)
		defer t.Stop()
		select {
		case <-q.done:
		case <-t.C:
		case <-cancel:
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire()
	a := Answer{Status: q.status, App: q.App}
	if a.Status != Pending {
		a.Token, q.token = q.token, ""
		delete(r.requests, ticket)
	}
	if a.Status == Approved {
		a.Scopes = q.granted
	}
	return a
}

// Requests returns the requests waiting for the user, oldest first.
func (r *Registry) Requests() []Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire()
	var out []Request
	for _, q := range r.requests {
		if q.status == Pending {
			out = append(out, *q)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.Before(out[j].Created) })
	return out
}

// Pairings returns the approved apps, by name.
func (r *Registry) Pairings() []Pairing {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Pairing, len(r.pairings))
	for i, p := range r.pairings {
		out[i] = *p
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Decide answers the request with id (Request.ID). An approval grants the
// scopes of grant the request asked for; none = the app's records only.
func (r *Registry) Decide(id string, approve bool, grant ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expire()
	var q *Request
	for _, x := range r.requests {
		if x.ID == id && x.status == Pending {
			q = x
		}
	}
	if q == nil {
		return ErrUnknown
	}
	if !approve {
		q.status = Denied
		close(q.done)
		r.changed()
		return nil
	}
	p := &Pairing{ID: crypto.RandomID(8), App: q.App, Name: q.Name, Description: q.Description,
		Package: q.Package, Cert: q.Cert, Approved: r.now().UTC(), Scopes: within(grant, q.Scopes)}
	if !q.android() {
		q.token = crypto.RandomID(32)
		p.TokenHash = hashToken(q.token)
	}
	// One pairing per app and caller: pairing again replaces the old one
	// (a desktop app that lost its token, a reinstalled package).
	kept := r.pairings[:0:0]
	for _, o := range r.pairings {
		if o.App == p.App && o.Package == p.Package {
			continue
		}
		kept = append(kept, o)
	}
	prev := r.pairings
	r.pairings = append(kept, p)
	if err := r.save(); err != nil {
		r.pairings = prev
		q.token = ""
		return err
	}
	q.status, q.granted = Approved, p.Scopes
	close(q.done)
	r.changed()
	return nil
}

func (q *Request) android() bool { return q.Package != "" }

// SetScopes keeps only scopes of the pairing with id (the user takes a
// permission away without unpairing). Scopes it does not have are not
// added: an app asks for them by pairing again.
func (r *Registry) SetScopes(id string, scopes []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.pairings {
		if p.ID != id {
			continue
		}
		keep := within(scopes, p.Scopes)
		if slices.Equal(keep, p.Scopes) {
			return nil
		}
		prev := p.Scopes
		p.Scopes = keep
		if err := r.save(); err != nil {
			p.Scopes = prev
			return err
		}
		r.changed()
		return nil
	}
	return ErrUnknown
}

// LastRead is when the pairing last read the secrets of scope (unix ms;
// 0 = never).
func (p Pairing) LastRead(scope string) int64 {
	if scope == ScopePasswords {
		return p.SecretsReadMs
	}
	return p.ReadMs[scope]
}

// SecretsRead records that the pairing with id read the secrets of scope
// (passwords, SSH private keys, server passwords): which app and when,
// never what. It is saved at most once a minute: it is shown, not relied
// on.
func (r *Registry) SecretsRead(id, scope string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now().UnixMilli()
	for _, p := range r.pairings {
		if p.ID != id {
			continue
		}
		saved := p.LastRead(scope)
		if scope == ScopePasswords {
			p.SecretsReadMs = now
		} else {
			// A new map: copies handed out (Pairings, Authenticate) keep
			// reading the old one.
			m := make(map[string]int64, len(p.ReadMs)+1)
			for k, v := range p.ReadMs {
				m[k] = v
			}
			m[scope] = now
			p.ReadMs = m
		}
		if now-saved >= time.Minute.Milliseconds() {
			_ = r.save()
			r.changed()
		}
	}
}

// Dir is where the registry keeps its files.
func (r *Registry) Dir() string { return filepath.Dir(r.path) }

// Revoke forgets the pairing with id.
func (r *Registry) Revoke(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.pairings {
		if p.ID != id {
			continue
		}
		prev := r.pairings
		r.pairings = append(append([]*Pairing{}, r.pairings[:i]...), r.pairings[i+1:]...)
		if err := r.save(); err != nil {
			r.pairings = prev
			return err
		}
		r.changed()
		return nil
	}
	return ErrUnknown
}
