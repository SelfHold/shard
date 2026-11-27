package connector

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/SelfHold/shard/core/connector/pb"
	"github.com/SelfHold/shard/core/crypto"
	"github.com/SelfHold/shard/core/engine"
)

// The password store through the Connector API (scope "passwords"): the
// logins of Shard's safe (engine/logins.go).

// LoginReader reads logins: the open engine, or this device's own copy
// while no storage answers.
type LoginReader interface {
	LoginList(realm, after string, withData bool) ([]engine.Login, bool)
	LoginGet(ids []string) []engine.Login
	LoginRevisions() map[string]string
}

// LoginWriter also writes them (the open engine only).
type LoginWriter interface {
	LoginReader
	LoginPut(ctx context.Context, l engine.Login, base string) (engine.Login, error)
	LoginDelete(ctx context.Context, id, base string) error
}

// authScope is auth plus a granted scope.
func (s *Server) authScope(ctx context.Context, scope string) (Pairing, error) {
	p, err := s.auth(ctx)
	if err != nil {
		return p, err
	}
	if !p.Has(scope) {
		return p, status.Errorf(codes.PermissionDenied, "the scope %q was not granted to %s on this device", scope, p.App)
	}
	return p, nil
}

func (s *Server) loginPB(l engine.Login, vault string) *pb.Login {
	out := &pb.Login{
		Id: l.ID, Revision: l.Revision, SignonRealm: l.SignonRealm, Url: l.URL, ActionUrl: l.ActionURL,
		Username: l.Username, Password: l.Password, UsernameElement: l.UsernameElement,
		PasswordElement: l.PasswordElement, DisplayName: l.DisplayName, Notes: l.Notes,
		FederationOrigin: l.FederationOrigin, Blocked: l.Blocked, Extra: l.Extra, DeviceName: l.DeviceName,
	}
	if !l.Created.IsZero() {
		out.CreatedMs = l.Created.UnixMilli()
	}
	if !l.Modified.IsZero() {
		out.ModifiedMs = l.Modified.UnixMilli()
	}
	if !l.PasswordModified.IsZero() {
		out.PasswordModifiedMs = l.PasswordModified.UnixMilli()
	}
	out.LastUsedMs, out.TimesUsed = s.uses.get(vault, l.ID)
	return out
}

func loginFromPB(l *pb.Login) engine.Login {
	if l == nil {
		return engine.Login{}
	}
	out := engine.Login{
		ID: l.Id, SignonRealm: l.SignonRealm, URL: l.Url, ActionURL: l.ActionUrl, Username: l.Username,
		Password: l.Password, UsernameElement: l.UsernameElement, PasswordElement: l.PasswordElement,
		DisplayName: l.DisplayName, Notes: l.Notes, FederationOrigin: l.FederationOrigin, Blocked: l.Blocked,
		Extra: l.Extra,
	}
	if len(out.Extra) == 0 {
		out.Extra = nil
	}
	if l.CreatedMs > 0 {
		out.Created = time.UnixMilli(l.CreatedMs)
	}
	if l.PasswordModifiedMs > 0 {
		out.PasswordModified = time.UnixMilli(l.PasswordModifiedMs)
	}
	return out
}

// loginReader returns the logins and the vault they belong to.
func (s *Server) loginReader() (LoginReader, string, error) {
	r, err := s.Host.Logins()
	if err != nil {
		return nil, "", err
	}
	return r, s.Host.ConnectorState().VaultID, nil
}

// pageToken is the id the next page starts after (opaque to apps).
func pageToken(after string) string { return "a" + after }

func pageAfter(tok string) (string, error) {
	if tok == "" {
		return "", nil
	}
	after, ok := strings.CutPrefix(tok, "a")
	if !ok || after == "" {
		return "", status.Error(codes.InvalidArgument, "bad page_token")
	}
	return after, nil
}

func (s *Server) ListLogins(ctx context.Context, r *pb.ListLoginsRequest) (*pb.ListLoginsReply, error) {
	p, err := s.authScope(ctx, ScopePasswords)
	if err != nil {
		return nil, err
	}
	rd, vault, err := s.loginReader()
	if err != nil {
		return nil, err
	}
	after, err := pageAfter(r.PageToken)
	if err != nil {
		return nil, err
	}
	// The cursor first: a change after it but before the list is sent
	// again by ListChanges, never lost.
	out := &pb.ListLoginsReply{Cursor: s.changes.observe(vault, rd.LoginRevisions())}
	logins, more := rd.LoginList(r.SignonRealm, after, r.Data)
	for _, l := range logins {
		out.Logins = append(out.Logins, s.loginPB(l, vault))
	}
	if more && len(logins) > 0 {
		out.NextPageToken = pageToken(logins[len(logins)-1].ID)
	}
	if r.Data && len(logins) > 0 {
		s.Reg.SecretsRead(p.ID, ScopePasswords)
	}
	return out, nil
}

func (s *Server) GetLogins(ctx context.Context, r *pb.GetLoginsRequest) (*pb.ListLoginsReply, error) {
	p, err := s.authScope(ctx, ScopePasswords)
	if err != nil {
		return nil, err
	}
	if len(r.Ids) > engine.MaxLoginPage {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d ids", engine.MaxLoginPage)
	}
	rd, vault, err := s.loginReader()
	if err != nil {
		return nil, err
	}
	out := &pb.ListLoginsReply{Cursor: s.changes.observe(vault, rd.LoginRevisions())}
	logins := rd.LoginGet(r.Ids)
	for _, l := range logins {
		out.Logins = append(out.Logins, s.loginPB(l, vault))
	}
	if len(logins) > 0 {
		s.Reg.SecretsRead(p.ID, ScopePasswords)
	}
	return out, nil
}

func (s *Server) loginWriter() (LoginWriter, string, error) {
	w, err := s.Host.LoginWriter()
	if err != nil {
		return nil, "", err
	}
	return w, s.Host.ConnectorState().VaultID, nil
}

func (s *Server) PutLogin(ctx context.Context, r *pb.PutLoginRequest) (*pb.Login, error) {
	if _, err := s.authScope(ctx, ScopePasswords); err != nil {
		return nil, err
	}
	if r.Login == nil {
		return nil, status.Error(codes.InvalidArgument, "no login")
	}
	w, vault, err := s.loginWriter()
	if err != nil {
		return nil, err
	}
	l, err := w.LoginPut(ctx, loginFromPB(r.Login), r.Base)
	if err != nil {
		return nil, toStatus(err)
	}
	s.wake()
	return s.loginPB(l, vault), nil
}

func (s *Server) DeleteLogin(ctx context.Context, r *pb.DeleteLoginRequest) (*pb.DeleteLoginReply, error) {
	if _, err := s.authScope(ctx, ScopePasswords); err != nil {
		return nil, err
	}
	w, vault, err := s.loginWriter()
	if err != nil {
		return nil, err
	}
	if err := w.LoginDelete(ctx, r.Id, r.Base); err != nil {
		return nil, toStatus(err)
	}
	s.uses.forget(vault, r.Id)
	s.wake()
	return &pb.DeleteLoginReply{}, nil
}

func (s *Server) TouchLogin(ctx context.Context, r *pb.TouchLoginRequest) (*pb.TouchLoginReply, error) {
	if _, err := s.authScope(ctx, ScopePasswords); err != nil {
		return nil, err
	}
	rd, vault, err := s.loginReader()
	if err != nil {
		return nil, err
	}
	// Reading the index only: no journal op, no new revision.
	if _, ok := rd.LoginRevisions()[r.Id]; !ok {
		return nil, status.Errorf(codes.NotFound, "no login %q", r.Id)
	}
	used := r.UsedMs
	if now := time.Now().UnixMilli(); used <= 0 || used > now {
		used = now
	}
	s.uses.touch(vault, r.Id, used)
	return &pb.TouchLoginReply{}, nil
}

func (s *Server) ListChanges(ctx context.Context, r *pb.ListChangesRequest) (*pb.ListChangesReply, error) {
	if _, err := s.authScope(ctx, ScopePasswords); err != nil {
		return nil, err
	}
	rd, vault, err := s.loginReader()
	if err != nil {
		return nil, err
	}
	cur := s.changes.observe(vault, rd.LoginRevisions())
	changes, ok := s.changes.since(r.Cursor)
	if !ok {
		return &pb.ListChangesReply{Cursor: cur, Reset_: true}, nil
	}
	return &pb.ListChangesReply{Changes: changes, Cursor: cur}, nil
}

// --- change log --------------------------------------------------------------

// changeLog numbers what this device saw of the logins (or servers), so
// ListChanges can answer "what changed since cursor" with ids and
// revisions only. It is local (<data root>/login-changes.json,
// server-changes.json), like the cursors it hands
// out: a cursor of another device, another vault or a lost file asks for
// a reset.
type changeLog struct {
	path string

	mu     sync.Mutex
	loaded bool
	f      changeFile
}

type changeFile struct {
	Vault string `json:"vault"`
	Epoch string `json:"epoch"` // random per log: a new log does not take old cursors
	Seq   uint64 `json:"seq"`
	// Floor: changes at or below it were dropped (old deletes); cursors
	// below it are reset.
	Floor  uint64               `json:"floor"`
	Logins map[string]*seenItem `json:"logins"`
}

type seenItem struct {
	Rev     string `json:"rev"`
	Deleted bool   `json:"deleted,omitempty"`
	Seq     uint64 `json:"seq"`
	At      int64  `json:"at"` // unix ms of the change
}

// Deletes are kept this long (and at most maxTombs) for cursors to see.
const (
	tombTTL  = 90 * 24 * time.Hour
	maxTombs = 20000
)

// newChangeLog keeps a change log in dir/name (one per kind of item: a
// cursor of one log is reset by the others, their epochs differ).
func newChangeLog(dir, name string) *changeLog {
	return &changeLog{path: filepath.Join(dir, name)}
}

func (c *changeLog) load() {
	if c.loaded {
		return
	}
	c.loaded = true
	if data, err := os.ReadFile(c.path); err == nil {
		_ = json.Unmarshal(data, &c.f)
	}
}

func (c *changeLog) reset(vault string) {
	c.f = changeFile{Vault: vault, Epoch: crypto.RandomID(6), Logins: map[string]*seenItem{}}
}

func (c *changeLog) cursor() string { return c.f.Epoch + "." + strconv.FormatUint(c.f.Seq, 10) }

// observe records the current revisions of vault and returns the cursor
// after them.
func (c *changeLog) observe(vault string, revs map[string]string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()
	dirty := false
	if c.f.Vault != vault || c.f.Epoch == "" || c.f.Logins == nil {
		c.reset(vault)
		dirty = true
	}
	now := time.Now().UnixMilli()
	ids := make([]string, 0, len(revs))
	for id := range revs {
		ids = append(ids, id)
	}
	sort.Strings(ids) // a stable order of seq numbers
	for _, id := range ids {
		it := c.f.Logins[id]
		if it != nil && !it.Deleted && it.Rev == revs[id] {
			continue
		}
		c.f.Seq++
		c.f.Logins[id] = &seenItem{Rev: revs[id], Seq: c.f.Seq, At: now}
		dirty = true
	}
	var tombs []*seenItem
	var tombIDs []string
	for id, it := range c.f.Logins {
		if _, ok := revs[id]; !ok && !it.Deleted {
			c.f.Seq++
			it.Deleted, it.Seq, it.At = true, c.f.Seq, now
			dirty = true
		}
		if it.Deleted {
			tombs = append(tombs, it)
			tombIDs = append(tombIDs, id)
		}
	}
	// Drop old deletes; a cursor from before them can no longer be
	// answered.
	order := make([]int, len(tombs))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return tombs[order[a]].Seq < tombs[order[b]].Seq })
	for n, i := range order {
		it := tombs[i]
		if len(tombs)-n <= maxTombs && now-it.At < tombTTL.Milliseconds() {
			break
		}
		c.f.Floor = max(c.f.Floor, it.Seq)
		delete(c.f.Logins, tombIDs[i])
		dirty = true
	}
	if dirty {
		c.save()
	}
	return c.cursor()
}

func (c *changeLog) save() {
	data, err := json.Marshal(c.f)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return
	}
	tmp := c.path + ".tmp"
	if os.WriteFile(tmp, data, 0o600) == nil {
		_ = os.Rename(tmp, c.path)
	}
}

// since returns the changes after cursor, oldest first; false: the cursor
// cannot be answered (list everything again).
func (c *changeLog) since(cursor string) ([]*pb.LoginChange, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	epoch, n, ok := strings.Cut(cursor, ".")
	seq, err := strconv.ParseUint(n, 10, 64)
	if !ok || err != nil || epoch != c.f.Epoch || seq > c.f.Seq || seq < c.f.Floor {
		return nil, false
	}
	type change struct {
		id string
		it *seenItem
	}
	var list []change
	for id, it := range c.f.Logins {
		if it.Seq > seq {
			list = append(list, change{id, it})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].it.Seq < list[j].it.Seq })
	out := make([]*pb.LoginChange, len(list))
	for i, ch := range list {
		out[i] = &pb.LoginChange{Id: ch.id, Revision: ch.it.Rev, Deleted: ch.it.Deleted}
		if ch.it.Deleted {
			out[i].Revision = ""
		}
	}
	return out, true
}

// --- use counters ------------------------------------------------------------

// useCounters keeps last_used_ms / times_used of logins on this device
// only (<data root>/login-use.json): a fill is not a change of the login,
// so it never becomes a journal op.
type useCounters struct {
	path string

	mu     sync.Mutex
	loaded bool
	f      useFile
	timer  *time.Timer
}

type useFile struct {
	Vault  string             `json:"vault"`
	Logins map[string]*useRec `json:"logins"`
}

type useRec struct {
	Last  int64 `json:"last"`
	Count int64 `json:"n"`
}

// saveDelay batches the writes of many fills.
const saveDelay = 2 * time.Second

func newUseCounters(dir string) *useCounters {
	return &useCounters{path: filepath.Join(dir, "login-use.json")}
}

// of returns the counters of vault; u.mu is held.
func (u *useCounters) of(vault string) map[string]*useRec {
	if !u.loaded {
		u.loaded = true
		if data, err := os.ReadFile(u.path); err == nil {
			_ = json.Unmarshal(data, &u.f)
		}
	}
	if u.f.Vault != vault || u.f.Logins == nil {
		u.f = useFile{Vault: vault, Logins: map[string]*useRec{}}
	}
	return u.f.Logins
}

func (u *useCounters) get(vault, id string) (int64, int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if r := u.of(vault)[id]; r != nil {
		return r.Last, r.Count
	}
	return 0, 0
}

func (u *useCounters) touch(vault, id string, used int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	m := u.of(vault)
	r := m[id]
	if r == nil {
		r = &useRec{}
		m[id] = r
	}
	r.Last, r.Count = max(r.Last, used), r.Count+1
	u.later()
}

func (u *useCounters) forget(vault, id string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	m := u.of(vault)
	if _, ok := m[id]; ok {
		delete(m, id)
		u.later()
	}
}

// later saves the counters soon; u.mu is held.
func (u *useCounters) later() {
	if u.timer != nil {
		return
	}
	u.timer = time.AfterFunc(saveDelay, u.flush)
}

// flush writes the counters now.
func (u *useCounters) flush() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.timer != nil {
		u.timer.Stop()
		u.timer = nil
	}
	data, err := json.Marshal(u.f)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(u.path), 0o700); err != nil {
		return
	}
	tmp := u.path + ".tmp"
	if os.WriteFile(tmp, data, 0o600) == nil {
		_ = os.Rename(tmp, u.path)
	}
}
