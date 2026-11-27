package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"cloudsync/core/crypto"
	"cloudsync/core/index"
	"cloudsync/core/secrets"
)

// Servers: the user's servers, one list for every connected app with the
// scope "servers" (docs/CONNECTOR.md). A server is a KindServer entry of
// the password safe, so the user sees and edits it in CloudSync next to
// the SSH keys it logs in with.

const (
	// MaxServers is how many servers the API lets a vault have.
	MaxServers = 10000
	// MaxServerPage is the largest page of ServerList.
	MaxServerPage = 500
)

var (
	// ErrServerConflict: the server changed since the given revision.
	ErrServerConflict = errors.New("server: changed since the given revision")
	// ErrServersFull: MaxServers reached.
	ErrServersFull = fmt.Errorf("server: at most %d servers", MaxServers)
)

// Server is one server of the password safe.
type Server struct {
	ID, Revision string
	Name         string
	Group        string // folder, "a/b" nests
	Host         string
	Port         int // 0 = 22
	User         string
	Auth         secrets.ServerAuth
	Password     string // only with data
	SSHKeyID     string
	HostKeys     []string
	JumpServerID string
	Notes        string
	Tags         []string
	Created      time.Time
	Modified     time.Time
	DeviceName   string
	Extra        map[string][]byte // only with data
	HasData      bool
}

// serverOf turns an opened entry (with its data) into a server; false:
// not a server.
func serverOf(en secrets.Entry, deviceName string, withData bool) (Server, bool) {
	if en.Kind != secrets.KindServer || en.Data == nil {
		return Server{}, false
	}
	host, port, _, err := secrets.SplitServerAddr(en.URL)
	if err != nil {
		host, port = en.URL, 0 // as stored; Prepare would not have let it in
	}
	sv := secrets.ServerMeta{}
	if en.Server != nil {
		sv = *en.Server
	}
	out := Server{
		ID: en.ID, Revision: en.Version, Name: en.Title, Group: sv.Folder, Host: host, Port: port,
		User: en.Username, Auth: sv.Auth, SSHKeyID: sv.SSHKeyID, HostKeys: sv.HostKeys,
		JumpServerID: sv.JumpServerID, Notes: en.Data.Notes, Tags: sv.Tags,
		Created: en.Created, Modified: en.Updated, DeviceName: deviceName,
	}
	if withData {
		out.HasData = true
		out.Password, out.Extra = en.Data.Password, en.Data.Extra
	}
	return out, true
}

// openServer opens a record as a server. Notes are part of a server
// without data, and they are in the sealed data, so it is always opened;
// password and extra are kept only withData.
func (v safeView) openServer(s index.Secret, withData bool) (Server, bool) {
	en, err := v.keys.OpenEntry(s, true)
	if err != nil {
		return Server{}, false
	}
	return serverOf(en, s.DeviceName, withData)
}

// ServerList returns the servers with an id after after, by id, at most
// MaxServerPage; more tells whether others follow.
func (v safeView) ServerList(after string, withData bool) (out []Server, more bool) {
	for _, s := range v.kindRecords(secrets.KindServer) {
		if s.ID <= after {
			continue
		}
		sv, ok := v.openServer(s, withData)
		if !ok {
			continue
		}
		if len(out) == MaxServerPage {
			return out, true
		}
		out = append(out, sv)
	}
	return out, false
}

// ServerGet returns the servers with ids, with their secrets; unknown ids
// and entries of other kinds are left out.
func (v safeView) ServerGet(ids []string) []Server {
	var out []Server
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
		if sv, ok := v.openServer(s, true); ok {
			out = append(out, sv)
		}
	}
	return out
}

// ServerRevisions returns the revision of every server, by id.
func (v safeView) ServerRevisions() map[string]string {
	out := map[string]string{}
	for _, s := range v.kindRecords(secrets.KindServer) {
		out[s.ID] = s.VersionID
	}
	return out
}

// ServerList: see safeView.ServerList.
func (e *Engine) ServerList(after string, withData bool) ([]Server, bool) {
	return e.view().ServerList(after, withData)
}

// ServerGet: see safeView.ServerGet.
func (e *Engine) ServerGet(ids []string) []Server { return e.view().ServerGet(ids) }

// ServerRevisions: see safeView.ServerRevisions.
func (e *Engine) ServerRevisions() map[string]string { return e.view().ServerRevisions() }

// ServerPut creates (sv.ID = "", base = "") or replaces a server. base is
// the revision sv was made from: another revision now gives
// ErrServerConflict and nothing is written. The safe group the user put
// the entry in is kept.
func (e *Engine) ServerPut(ctx context.Context, sv Server, base string) (Server, error) {
	e.loginMu.Lock()
	defer e.loginMu.Unlock()
	keys := e.secretKeys()
	now := time.Now().UTC()
	var (
		m      secrets.Meta
		id     = sv.ID
		parent string
	)
	if id == "" {
		if base != "" {
			return Server{}, ErrServerConflict
		}
		if len(e.view().kindRecords(secrets.KindServer)) >= MaxServers {
			return Server{}, ErrServersFull
		}
		id = crypto.RandomID(12)
		m.Created = now
		if !sv.Created.IsZero() && sv.Created.Before(now) {
			m.Created = sv.Created.UTC() // moved from the app's own list
		}
	} else {
		cur, ok := e.idx.Secret(id)
		if !ok || cur.Group {
			if base != "" {
				return Server{}, ErrServerConflict // deleted meanwhile
			}
			return Server{}, fmt.Errorf("server %s: %w", id, ErrNotFound)
		}
		en, err := keys.OpenEntry(cur, false)
		if err != nil {
			return Server{}, err
		}
		if en.Kind != secrets.KindServer {
			return Server{}, fmt.Errorf("server %s: %w", id, ErrNotFound)
		}
		if cur.VersionID != base {
			return Server{}, ErrServerConflict
		}
		m, parent = en.Meta, cur.VersionID
	}
	if sv.JumpServerID != "" && sv.JumpServerID == id {
		return Server{}, fmt.Errorf("%w: a server cannot be its own jump server", secrets.ErrInvalid)
	}
	if sv.Port < 0 || sv.Port > 65535 {
		return Server{}, fmt.Errorf("%w: bad port %d", secrets.ErrInvalid, sv.Port)
	}
	host := strings.TrimSpace(sv.Host)
	if host == "" {
		return Server{}, fmt.Errorf("%w: a host is needed", secrets.ErrInvalid)
	}
	if strings.Contains(host, ":") {
		// Only an IPv6 address has colons; the port is never read from it.
		if net.ParseIP(strings.Trim(host, "[]")) == nil {
			return Server{}, fmt.Errorf("%w: bad host %q (the port goes in port)", secrets.ErrInvalid, host)
		}
		host = "[" + strings.Trim(host, "[]") + "]"
	}
	m.Kind, m.Title, m.Username, m.Updated = secrets.KindServer, sv.Name, sv.User, now
	m.URL = host
	if sv.Port != 0 && sv.Port != 22 {
		m.URL = fmt.Sprintf("%s:%d", host, sv.Port)
	}
	m.Server = &secrets.ServerMeta{Folder: sv.Group, Auth: sv.Auth, SSHKeyID: sv.SSHKeyID,
		HostKeys: sv.HostKeys, JumpServerID: sv.JumpServerID, Tags: sv.Tags}
	d := secrets.Data{Password: sv.Password, Notes: sv.Notes, Extra: sv.Extra}
	if len(d.Extra) == 0 {
		d.Extra = nil
	}
	if err := secrets.Prepare(&m, &d); err != nil {
		return Server{}, err
	}
	ch, err := keys.SealEntry(id, m, d)
	if err != nil {
		return Server{}, err
	}
	op := index.Op{Kind: index.OpSecret, Path: "/", Parent: parent, Secret: ch}
	if err := e.commitSecret(ctx, &op); err != nil {
		return Server{}, err
	}
	out, _ := serverOf(secrets.Entry{ID: id, Version: op.ID(), Device: op.Device, Meta: m, Data: &d}, op.DeviceName, true)
	return out, nil
}

// ServerDelete removes a server, with the same base rule as ServerPut.
func (e *Engine) ServerDelete(ctx context.Context, id, base string) error {
	e.loginMu.Lock()
	defer e.loginMu.Unlock()
	cur, ok := e.idx.Secret(id)
	if !ok || cur.Group {
		if base != "" {
			return ErrServerConflict
		}
		return fmt.Errorf("server %s: %w", id, ErrNotFound)
	}
	en, err := e.secretKeys().OpenEntry(cur, false)
	if err != nil || en.Kind != secrets.KindServer {
		return fmt.Errorf("server %s: %w", id, ErrNotFound)
	}
	if cur.VersionID != base {
		return ErrServerConflict
	}
	op := index.Op{Kind: index.OpSecret, Path: "/", Parent: cur.VersionID,
		Secret: &index.SecretChange{ID: cur.ID, Deleted: true}}
	return e.commitSecret(ctx, &op)
}
