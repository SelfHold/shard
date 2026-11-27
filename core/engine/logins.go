package engine

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/SelfHold/shard/core/crypto"
	"github.com/SelfHold/shard/core/index"
	"github.com/SelfHold/shard/core/secrets"
)

// Logins: the website logins of the password safe as connected apps see
// them (Connector API, scope "passwords"; docs/CONNECTOR.md). A login is a
// KindPassword entry with a website or an Android app, or a KindNeverSave
// site. They are the entries the user sees in Shard, not copies: a
// login written by a browser is an ordinary entry, and an entry typed in
// Shard with a URL is a login.

const (
	// MaxLogins is how many logins the API lets a vault have.
	MaxLogins = 10000
	// MaxLoginPage is the largest page of LoginList.
	MaxLoginPage = 500
)

var (
	// ErrLoginConflict: the login changed since the given revision.
	ErrLoginConflict = errors.New("login: changed since the given revision")
	// ErrLoginsFull: MaxLogins reached.
	ErrLoginsFull = fmt.Errorf("login: at most %d logins", MaxLogins)
)

// Login is one login of the password safe.
type Login struct {
	ID, Revision     string
	SignonRealm      string
	URL, ActionURL   string
	Username         string // only with data
	Password         string // only with data
	UsernameElement  string
	PasswordElement  string
	DisplayName      string
	Notes            string // only with data
	FederationOrigin string
	Blocked          bool
	Created          time.Time
	Modified         time.Time
	PasswordModified time.Time         // zero: not known
	Extra            map[string][]byte // only with data
	DeviceName       string
	HasData          bool
}

// Origin is the signon realm of a URL as a user types it:
// "https://host[:port]/" (https when the scheme is missing), an
// "android://<hash>@<package>/" realm as it is. "" when it names no site.
func Origin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\n") {
		return ""
	}
	if strings.HasPrefix(raw, "android://") {
		rest := strings.TrimPrefix(raw, "android://")
		if at := strings.LastIndex(rest, "@"); at <= 0 || at == len(rest)-1 {
			return ""
		}
		return "android://" + strings.TrimSuffix(rest, "/") + "/"
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	return u.Scheme + "://" + strings.ToLower(u.Host) + "/"
}

// siteName is a short title for a realm: the host or the package.
func siteName(realm string) string {
	if rest, ok := strings.CutPrefix(realm, "android://"); ok {
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		return strings.TrimSuffix(rest, "/")
	}
	if u, err := url.Parse(realm); err == nil && u.Host != "" {
		return strings.TrimPrefix(u.Host, "www.")
	}
	return realm
}

// realmOf is the signon realm of an entry; "" = not a login.
func realmOf(m secrets.Meta) string {
	if m.Kind != secrets.KindPassword && m.Kind != secrets.KindNeverSave {
		return ""
	}
	if m.Login != nil && m.Login.Realm != "" {
		return m.Login.Realm
	}
	return Origin(m.URL)
}

// loginOf turns an opened entry into a login; false: not a login.
func loginOf(en secrets.Entry, deviceName string) (Login, bool) {
	realm := realmOf(en.Meta)
	if realm == "" {
		return Login{}, false
	}
	l := Login{
		ID: en.ID, Revision: en.Version, SignonRealm: realm, URL: realm,
		Blocked: en.Kind == secrets.KindNeverSave, Created: en.Created, Modified: en.Updated,
		DeviceName: deviceName, DisplayName: en.Title,
	}
	if en.PasswordChangedMs > 0 {
		l.PasswordModified = time.UnixMilli(en.PasswordChangedMs).UTC()
	}
	if lm := en.Login; lm != nil {
		// Written through the API: the page as the browser gave it.
		l.URL = en.URL
		l.ActionURL, l.UsernameElement, l.PasswordElement = lm.ActionURL, lm.UsernameElement, lm.PasswordElement
		l.DisplayName, l.FederationOrigin = lm.DisplayName, lm.FederationOrigin
	}
	if l.Blocked {
		l.DisplayName = ""
	}
	if en.Data != nil {
		l.HasData = true
		if !l.Blocked {
			l.Username, l.Password, l.Notes, l.Extra = en.Username, en.Data.Password, en.Data.Notes, en.Data.Extra
		}
	}
	return l, true
}

// entryKinds remembers what each record version is, so listing
// revisions (on every Watch wake-up) opens only new versions.
type entryKinds struct {
	mu sync.Mutex
	m  map[string]recKind // version id → what it is
}

// recKind is what the meta of one record version says.
type recKind struct {
	kind  secrets.Kind // "" = could not be opened
	login bool
}

// records returns the safe entries (no groups) whose kind want accepts,
// by id. Each is opened (meta only) at most once per version.
func (v safeView) records(want func(recKind) bool) []index.Secret {
	var out []index.Secret
	next := map[string]recKind{}
	for _, s := range v.idx.Secrets() {
		if s.Group {
			continue
		}
		k, known := recKind{}, false
		if v.kinds != nil {
			v.kinds.mu.Lock()
			k, known = v.kinds.m[s.VersionID]
			v.kinds.mu.Unlock()
		}
		if !known {
			if en, err := v.keys.OpenEntry(s, false); err == nil {
				k = recKind{kind: en.Kind, login: realmOf(en.Meta) != ""}
			}
		}
		next[s.VersionID] = k
		if want(k) {
			out = append(out, s)
		}
	}
	if v.kinds != nil {
		v.kinds.mu.Lock()
		v.kinds.m = next
		v.kinds.mu.Unlock()
	}
	return out
}

// loginRecords returns the safe records that are logins, by id.
func (v safeView) loginRecords() []index.Secret {
	return v.records(func(k recKind) bool { return k.login })
}

// kindRecords returns the safe records of kind, by id.
func (v safeView) kindRecords(kind secrets.Kind) []index.Secret {
	return v.records(func(k recKind) bool { return k.kind == kind })
}

func (v safeView) openLogin(s index.Secret, withData bool) (Login, bool) {
	en, err := v.keys.OpenEntry(s, withData)
	if err != nil {
		return Login{}, false
	}
	return loginOf(en, s.DeviceName)
}

// LoginList returns the logins of realm ("" = all) with an id after
// after, by id, at most MaxLoginPage; more tells whether others follow.
// Without data, username, password, notes and extra stay sealed.
func (v safeView) LoginList(realm, after string, withData bool) (out []Login, more bool) {
	for _, s := range v.loginRecords() {
		if s.ID <= after {
			continue
		}
		l, ok := v.openLogin(s, withData)
		if !ok || (realm != "" && l.SignonRealm != realm) {
			continue
		}
		if len(out) == MaxLoginPage {
			return out, true
		}
		out = append(out, l)
	}
	return out, false
}

// LoginGet returns the logins with ids, with their secrets; unknown ids
// and entries that are not logins are left out.
func (v safeView) LoginGet(ids []string) []Login {
	var out []Login
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		s, ok := v.idx.Secret(id)
		if !ok || s.Group {
			continue
		}
		if l, ok := v.openLogin(s, true); ok {
			out = append(out, l)
		}
	}
	return out
}

// LoginRevisions returns the revision of every login, by id.
func (v safeView) LoginRevisions() map[string]string {
	out := map[string]string{}
	for _, s := range v.loginRecords() {
		out[s.ID] = s.VersionID
	}
	return out
}

// LoginList: see safeView.LoginList.
func (e *Engine) LoginList(realm, after string, withData bool) ([]Login, bool) {
	return e.view().LoginList(realm, after, withData)
}

// LoginGet: see safeView.LoginGet.
func (e *Engine) LoginGet(ids []string) []Login { return e.view().LoginGet(ids) }

// LoginRevisions returns the revision of every login, by id.
func (e *Engine) LoginRevisions() map[string]string { return e.view().LoginRevisions() }

// LoginPut creates (l.ID = "", base = "") or replaces a login. base is the
// revision l was made from: another revision now gives ErrLoginConflict
// and nothing is written. Fields of the entry the API does not carry
// (group, title typed in Shard, kind-specific fields) are kept.
func (e *Engine) LoginPut(ctx context.Context, l Login, base string) (Login, error) {
	e.loginMu.Lock()
	defer e.loginMu.Unlock()
	keys := e.secretKeys()
	now := time.Now().UTC()
	var (
		m      secrets.Meta
		d      secrets.Data
		id     = l.ID
		parent string
	)
	if id == "" {
		if base != "" {
			return Login{}, ErrLoginConflict
		}
		if len(e.view().loginRecords()) >= MaxLogins {
			return Login{}, ErrLoginsFull
		}
		id = crypto.RandomID(12)
		m.Created = now
		if !l.Created.IsZero() && l.Created.Before(now) {
			m.Created = l.Created.UTC() // moved from the browser's own store
		}
	} else {
		cur, ok := e.idx.Secret(id)
		if !ok || cur.Group {
			if base != "" {
				return Login{}, ErrLoginConflict // deleted meanwhile
			}
			return Login{}, fmt.Errorf("login %s: %w", id, ErrNotFound)
		}
		en, err := keys.OpenEntry(cur, true)
		if err != nil {
			return Login{}, err
		}
		if realmOf(en.Meta) == "" {
			return Login{}, fmt.Errorf("login %s: %w", id, ErrNotFound)
		}
		if cur.VersionID != base {
			return Login{}, ErrLoginConflict
		}
		m, d, parent = en.Meta, *en.Data, cur.VersionID
	}
	if err := applyLogin(&m, &d, l, now); err != nil {
		return Login{}, err
	}
	if err := secrets.Prepare(&m, &d); err != nil {
		return Login{}, err
	}
	ch, err := keys.SealEntry(id, m, d)
	if err != nil {
		return Login{}, err
	}
	op := index.Op{Kind: index.OpSecret, Path: "/", Parent: parent, Secret: ch}
	if err := e.commitSecret(ctx, &op); err != nil {
		return Login{}, err
	}
	out, _ := loginOf(secrets.Entry{ID: id, Version: op.ID(), Device: op.Device, Meta: m, Data: &d}, op.DeviceName)
	return out, nil
}

// applyLogin writes the fields of l into an entry.
func applyLogin(m *secrets.Meta, d *secrets.Data, l Login, now time.Time) error {
	realm := strings.TrimSpace(l.SignonRealm)
	if realm == "" {
		realm = Origin(l.URL)
	}
	if realm == "" || !strings.Contains(realm, "://") {
		return fmt.Errorf("%w: a signon_realm (or a url to take it from) is needed", secrets.ErrInvalid)
	}
	m.Updated = now
	if l.Blocked {
		*m = secrets.Meta{Kind: secrets.KindNeverSave, Title: siteName(realm), URL: realm,
			Login: &secrets.LoginMeta{Realm: realm}, Created: m.Created, Updated: now}
		*d = secrets.Data{}
		return nil
	}
	created := m.Kind == ""
	m.Kind = secrets.KindPassword
	if created || d.Password != l.Password || m.PasswordChangedMs == 0 {
		m.PasswordChangedMs = now.UnixMilli()
		if ms := l.PasswordModified.UnixMilli(); !l.PasswordModified.IsZero() && created && ms > 0 && ms < m.PasswordChangedMs {
			m.PasswordChangedMs = ms
		}
	}
	m.Username, d.Password, d.Notes, d.Extra = l.Username, l.Password, l.Notes, l.Extra
	m.URL = strings.TrimSpace(l.URL)
	if m.URL == "" {
		m.URL = realm
	}
	lm := &secrets.LoginMeta{ActionURL: l.ActionURL, UsernameElement: l.UsernameElement,
		PasswordElement: l.PasswordElement, DisplayName: l.DisplayName, FederationOrigin: l.FederationOrigin}
	// The realm is kept only when the URL does not give it, so a site
	// edited in Shard moves the realm with it.
	if realm != Origin(m.URL) {
		lm.Realm = realm
	}
	m.Login = lm
	switch {
	case l.DisplayName != "":
		m.Title = l.DisplayName
	case strings.TrimSpace(m.Title) == "":
		m.Title = siteName(realm)
	}
	return nil
}

// LoginDelete removes a login, with the same base rule as LoginPut.
func (e *Engine) LoginDelete(ctx context.Context, id, base string) error {
	e.loginMu.Lock()
	defer e.loginMu.Unlock()
	cur, ok := e.idx.Secret(id)
	if !ok || cur.Group {
		if base != "" {
			return ErrLoginConflict
		}
		return fmt.Errorf("login %s: %w", id, ErrNotFound)
	}
	if _, ok := e.view().openLogin(cur, false); !ok {
		return fmt.Errorf("login %s: %w", id, ErrNotFound)
	}
	if cur.VersionID != base {
		return ErrLoginConflict
	}
	op := index.Op{Kind: index.OpSecret, Path: "/", Parent: cur.VersionID,
		Secret: &index.SecretChange{ID: cur.ID, Deleted: true}}
	return e.commitSecret(ctx, &op)
}

// keepLoginFields carries what a browser stored with an entry over an
// edit in Shard, whose UI does not show it; old is the entry before.
func keepLoginFields(m *secrets.Meta, d *secrets.Data, old secrets.Entry, now time.Time) {
	if old.Data == nil {
		return
	}
	if old.Kind == m.Kind {
		if m.Login == nil && old.Login != nil {
			lm := *old.Login
			if Origin(m.URL) != Origin(old.URL) {
				// Another site: what the browser knew of the old form
				// no longer applies.
				lm = secrets.LoginMeta{DisplayName: lm.DisplayName}
			}
			m.Login = &lm
		}
		if d.Extra == nil {
			d.Extra = old.Data.Extra
		}
		if m.Server == nil && old.Server != nil {
			sv := *old.Server // an editor that does not know the fields
			m.Server = &sv
		}
	}
	m.PasswordChangedMs = old.PasswordChangedMs
	if d.Password != old.Data.Password || m.PasswordChangedMs == 0 && d.Password != "" {
		m.PasswordChangedMs = now.UnixMilli()
	}
	if m.Login != nil && m.Login.DisplayName != "" && m.Title != old.Title {
		m.Login.DisplayName = m.Title // renamed in Shard
	}
}
