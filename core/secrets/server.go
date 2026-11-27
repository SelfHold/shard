package secrets

import (
	"net"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// ServerAuth is how a server is logged in to.
type ServerAuth string

const (
	AuthUnspecified ServerAuth = ""
	AuthPassword    ServerAuth = "password" // Data.Password
	AuthSSHKey      ServerAuth = "ssh_key"  // ServerMeta.SSHKeyID: a KindSSH entry
	AuthAgent       ServerAuth = "agent"    // the device's ssh-agent
)

// ServerMeta is what a server entry keeps besides its address, user,
// password and notes. None of it is secret: host keys are public.
type ServerMeta struct {
	Folder       string     `json:"folder,omitempty"` // "a/b" nests folders
	Auth         ServerAuth `json:"auth,omitempty"`
	SSHKeyID     string     `json:"ssh_key_id,omitempty"`
	HostKeys     []string   `json:"host_keys,omitempty"` // authorized_keys lines
	JumpServerID string     `json:"jump_server_id,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
}

// Limits of a server entry (it is in every op that changes it).
const (
	MaxHostKeys   = 64
	maxHostKey    = 16 << 10
	maxServerTags = 64
	maxServerTag  = 128
	maxServerID   = 128
)

// ServerAddr joins a host and a port into the URL of a server entry:
// "host", or "host:port" when the port is not 22 ("[::1]:2222" for IPv6).
func ServerAddr(host string, port int) string {
	if port == 0 || port == 22 {
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// SplitServerAddr reads "host", "host:port", "[v6]:port", a bare IPv6
// address, "user@host" or "ssh://user@host:port/" as the user may type
// it. port 0 = the default (22); user is "" when none was given.
func SplitServerAddr(addr string) (host string, port int, user string, err error) {
	a := strings.TrimSpace(addr)
	a = strings.TrimPrefix(a, "ssh://")
	a = strings.TrimSuffix(a, "/")
	if at := strings.LastIndex(a, "@"); at >= 0 {
		user, a = a[:at], a[at+1:]
	}
	host = a
	switch {
	case strings.HasPrefix(a, "["):
		h, p, e := net.SplitHostPort(a)
		if e != nil {
			if !strings.HasSuffix(a, "]") {
				return "", 0, "", invalidf("bad server address %q", addr)
			}
			h, p = strings.TrimSuffix(strings.TrimPrefix(a, "["), "]"), ""
		}
		host = h
		if p != "" {
			if port, err = parsePort(p); err != nil {
				return "", 0, "", err
			}
		}
	case strings.Count(a, ":") == 1:
		h, p, _ := strings.Cut(a, ":")
		if port, err = parsePort(p); err != nil {
			return "", 0, "", err
		}
		host = h
	}
	if host == "" || strings.ContainsAny(host, " \t\r\n/@") || len(host) > 255 {
		return "", 0, "", invalidf("bad server address %q", addr)
	}
	if port == 22 {
		port = 0
	}
	return host, port, user, nil
}

func parsePort(p string) (int, error) {
	n, err := strconv.Atoi(p)
	if err != nil || n < 1 || n > 65535 {
		return 0, invalidf("bad port %q", p)
	}
	return n, nil
}

// prepareServer checks a server entry and puts its fields in one form.
func prepareServer(m *Meta, d *Data) error {
	if m.Server == nil {
		m.Server = &ServerMeta{}
	}
	sv := m.Server
	host, port, user, err := SplitServerAddr(m.URL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(m.Username) == "" && user != "" {
		m.Username = user
	}
	m.Username = strings.TrimSpace(m.Username)
	m.URL = ServerAddr(host, port)
	if strings.TrimSpace(m.Title) == "" {
		m.Title = host
	}
	switch sv.Auth {
	case AuthUnspecified, AuthPassword, AuthSSHKey, AuthAgent:
	default:
		return invalidf("unknown way to log in %q", sv.Auth)
	}
	sv.SSHKeyID, sv.JumpServerID = strings.TrimSpace(sv.SSHKeyID), strings.TrimSpace(sv.JumpServerID)
	if len(sv.SSHKeyID) > maxServerID || len(sv.JumpServerID) > maxServerID {
		return invalid("an id is too long")
	}
	sv.Folder = cleanFolder(sv.Folder)
	if len(sv.Folder) > maxTitle {
		return invalid("folder is too long")
	}
	keys := sv.HostKeys[:0:0]
	for _, k := range sv.HostKeys {
		k = strings.TrimSpace(k)
		if k == "" || containsStr(keys, k) {
			continue
		}
		if len(k) > maxHostKey {
			return invalid("a host key is too long")
		}
		if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(k)); err != nil {
			return invalidf("not a host key (authorized_keys format): %q", short(k))
		}
		keys = append(keys, k)
	}
	if len(keys) > MaxHostKeys {
		return invalidf("more than %d host keys", MaxHostKeys)
	}
	sv.HostKeys = keys
	tags := sv.Tags[:0:0]
	for _, t := range sv.Tags {
		t = strings.TrimSpace(t)
		if t == "" || containsStr(tags, t) {
			continue
		}
		if len(t) > maxServerTag {
			return invalid("a tag is too long")
		}
		tags = append(tags, t)
	}
	if len(tags) > maxServerTags {
		return invalidf("more than %d tags", maxServerTags)
	}
	sv.Tags = tags
	if len(sv.HostKeys) == 0 {
		sv.HostKeys = nil
	}
	if len(sv.Tags) == 0 {
		sv.Tags = nil
	}
	d.PrivateKey, d.Passphrase, m.Comment = "", "", ""
	return nil
}

// cleanFolder trims a folder path: "/a//b /" → "a/b".
func cleanFolder(f string) string {
	var parts []string
	for _, p := range strings.Split(f, "/") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "/")
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func short(s string) string {
	if len(s) > 40 {
		return s[:40] + "…"
	}
	return s
}
