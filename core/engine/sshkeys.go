package engine

import (
	"time"

	"cloudsync/core/index"
	"cloudsync/core/secrets"
)

// SSH keys: the KindSSH entries of the password safe as connected apps
// see them (Connector API, scope "ssh_keys", read only; docs/CONNECTOR.md).

// MaxIDs is how many ids one GetSshKeys / GetServers call may ask for.
const MaxIDs = 500

// SSHKey is one SSH key of the password safe.
type SSHKey struct {
	ID, Revision  string
	Title         string
	KeyType       string
	Fingerprint   string
	PublicKey     string // authorized_keys line
	Comment       string
	Encrypted     bool   // the private key parses only with a passphrase
	HasPassphrase bool   // the safe keeps that passphrase
	Group         string // name of the safe group; "" = none
	Created       time.Time
	Modified      time.Time
	DeviceName    string
	PrivateKey    string // only from SSHKeyGet
	Passphrase    string // only from SSHKeyGet
}

// groupNames maps the safe's group ids to their names.
func (v safeView) groupNames() map[string]string {
	out := map[string]string{}
	for _, s := range v.idx.Secrets() {
		if !s.Group {
			continue
		}
		if g, err := v.keys.OpenGroup(s); err == nil {
			out[g.ID] = g.Name
		}
	}
	return out
}

// openSSHKey opens one record as an SSH key; false: not one. The data is
// always opened (encrypted / has_passphrase come from it); the secrets
// are kept only when withSecret.
func (v safeView) openSSHKey(s index.Secret, groups map[string]string, withSecret bool) (SSHKey, bool) {
	en, err := v.keys.OpenEntry(s, true)
	if err != nil || en.Kind != secrets.KindSSH {
		return SSHKey{}, false
	}
	k := SSHKey{
		ID: en.ID, Revision: en.Version, Title: en.Title, KeyType: en.KeyType, Fingerprint: en.Fingerprint,
		PublicKey: en.PublicKey, Comment: en.Comment, Group: groups[en.Group],
		Created: en.Created, Modified: en.Updated, DeviceName: s.DeviceName,
		Encrypted:     secrets.SSHKeyEncrypted(en.Data.PrivateKey),
		HasPassphrase: en.Data.Passphrase != "",
	}
	if withSecret {
		k.PrivateKey, k.Passphrase = en.Data.PrivateKey, en.Data.Passphrase
	}
	return k, true
}

// SSHKeyList returns every SSH key of the safe, by id, without secrets.
func (v safeView) SSHKeyList() []SSHKey {
	groups := v.groupNames()
	var out []SSHKey
	for _, s := range v.kindRecords(secrets.KindSSH) {
		if k, ok := v.openSSHKey(s, groups, false); ok {
			out = append(out, k)
		}
	}
	return out
}

// SSHKeyGet returns the SSH keys with ids, with their secrets; unknown ids
// and entries of other kinds are left out.
func (v safeView) SSHKeyGet(ids []string) []SSHKey {
	groups := v.groupNames()
	var out []SSHKey
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
		if k, ok := v.openSSHKey(s, groups, true); ok {
			out = append(out, k)
		}
	}
	return out
}

// SSHKeyRevisions returns the revision of every SSH key, by id.
func (v safeView) SSHKeyRevisions() map[string]string {
	out := map[string]string{}
	for _, s := range v.kindRecords(secrets.KindSSH) {
		out[s.ID] = s.VersionID
	}
	return out
}

// SSHKeyList: see safeView.SSHKeyList.
func (e *Engine) SSHKeyList() []SSHKey { return e.view().SSHKeyList() }

// SSHKeyGet: see safeView.SSHKeyGet.
func (e *Engine) SSHKeyGet(ids []string) []SSHKey { return e.view().SSHKeyGet(ids) }

// SSHKeyRevisions: see safeView.SSHKeyRevisions.
func (e *Engine) SSHKeyRevisions() map[string]string { return e.view().SSHKeyRevisions() }
