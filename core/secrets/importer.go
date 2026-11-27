package secrets

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tobischo/gokeepasslib/v3"
)

// MaxImport is the largest export file ImportFile reads.
const MaxImport = 32 << 20

// ImportItem is one record read from another password manager. Group is
// the folder it was in there ("" = none); nested folders are joined
// with " / ".
type ImportItem struct {
	Meta
	Data  Data
	Group string
	Error error // why the record cannot be imported (Prepare); Data is empty then
}

// Import is what was read from an export file.
type Import struct {
	Format string // e.g. "KeePass (KDBX 4)", "Chrome / Edge / Brave"
	Items  []ImportItem
}

// ErrImportPassword means the file is encrypted: ImportFile needs its
// password (and key file).
var ErrImportPassword = errors.New("the file is protected by a password")

// ImportFile reads an export of another password manager: a KeePass
// database (.kdbx, with password and optional key file), a KeePass XML
// export, a Bitwarden JSON export or a CSV export of KeePass, KeePassXC,
// Chrome / Edge / Brave / Opera, Firefox, Safari / Apple Passwords,
// Bitwarden, 1Password, LastPass, Dashlane, Proton Pass, NordPass and
// the like (the columns are recognised by name). Every item is
// Prepared: an item that cannot be saved carries its Error. A login
// with a TOTP field gives a second, 2FA item.
func ImportFile(name string, data []byte, password string, keyFile []byte) (Import, error) {
	if len(data) > MaxImport {
		return Import{}, invalidf("the file is larger than %d MiB", MaxImport>>20)
	}
	var out Import
	var err error
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))
	low := strings.ToLower(name)
	switch {
	case len(data) == 0:
		return out, invalid("the file is empty")
	case isKDBX(data):
		out, err = readKDBX(data, password, keyFile)
	case bytes.HasPrefix(data, []byte{0x03, 0xd9, 0xa2, 0x9a, 0x65, 0xfb, 0x4b, 0xb5}):
		return out, invalid("KeePass 1 databases (.kdb) are not supported: open it in KeePass 2 or KeePassXC and save it as .kdbx")
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		return out, invalid("archives (.1pux, .zip) are not supported: export to CSV")
	case bytes.HasPrefix(trimmed, []byte("<")):
		out, err = readKeePassXML(trimmed)
	case bytes.HasPrefix(trimmed, []byte("{")):
		out, err = readBitwardenJSON(trimmed)
	case strings.HasSuffix(low, ".kdbx"):
		return out, invalid("not a KeePass database")
	default:
		out, err = readCSV(trimmed)
	}
	if err != nil {
		return out, err
	}
	out.Items = prepareImport(out.Items)
	if len(out.Items) == 0 {
		return out, invalid("the file holds no entries")
	}
	return out, nil
}

// prepareImport adds the 2FA items of logins with a TOTP field and checks
// every item. Items with nothing in them are dropped.
func prepareImport(in []ImportItem) []ImportItem {
	var out []ImportItem
	for _, it := range in {
		var otp string
		if it.Data.TOTP != nil {
			otp = it.Data.TOTP.Secret
			it.Data.TOTP = nil
		}
		it.Title = strings.TrimSpace(it.Title)
		if it.Title == "" {
			it.Title = titleFromURL(it.URL)
		}
		if it.Title == "" {
			it.Title = strings.TrimSpace(it.Username)
		}
		if it.Kind == "" {
			it.Kind = KindPassword
		}
		if it.Kind == KindPassword && it.Title == "" && it.Data.Password == "" && it.Data.Notes == "" && otp == "" {
			continue
		}
		if it.Title == "" {
			it.Title = "Imported"
		}
		it.Group = strings.TrimSpace(it.Group)
		if len(it.Group) > maxTitle {
			it.Group = it.Group[:maxTitle]
		}
		out = append(out, checked(it))
		if otp = strings.TrimSpace(otp); otp != "" {
			t := ImportItem{Group: it.Group}
			t.Kind, t.Title, t.Username, t.URL, t.Created = KindTOTP, it.Title, it.Username, it.URL, it.Created
			t.Data.TOTP = &TOTP{Secret: otp}
			out = append(out, checked(t))
		}
	}
	return out
}

func checked(it ImportItem) ImportItem {
	if err := Prepare(&it.Meta, &it.Data); err != nil {
		it.Error, it.Data = err, Data{}
	}
	return it
}

// titleFromURL is the host of a URL without "www.".
func titleFromURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

// --- KeePass ---

func isKDBX(b []byte) bool {
	return bytes.HasPrefix(b, []byte{0x03, 0xd9, 0xa2, 0x9a, 0x67, 0xfb, 0x4b, 0xb5}) ||
		bytes.HasPrefix(b, []byte{0x03, 0xd9, 0xa2, 0x9a, 0x66, 0xfb, 0x4b, 0xb5})
}

func readKDBX(data []byte, password string, keyFile []byte) (out Import, err error) {
	if password == "" && len(keyFile) == 0 {
		return out, ErrImportPassword
	}
	creds := &gokeepasslib.DBCredentials{}
	if password != "" {
		creds = gokeepasslib.NewPasswordCredentials(password)
	}
	if len(keyFile) > 0 {
		k, err := gokeepasslib.ParseKeyData(keyFile)
		if err != nil {
			return out, invalid("the key file cannot be read")
		}
		creds.Key = k
	}
	defer func() {
		// The library must not take the process down on a damaged file.
		if r := recover(); r != nil {
			err = invalid("the KeePass file is damaged")
		}
	}()
	db := gokeepasslib.NewDatabase()
	db.Credentials = creds
	if err := gokeepasslib.NewDecoder(bytes.NewReader(data)).Decode(db); err != nil {
		if errors.Is(err, gokeepasslib.ErrCorruptedDatabase) {
			return out, invalid("the KeePass file is damaged")
		}
		return out, invalid("wrong password or key file")
	}
	if err := db.UnlockProtectedEntries(); err != nil {
		return out, invalid("the KeePass file cannot be decrypted")
	}
	out.Format = "KeePass (KDBX 3)"
	if db.Header.IsKdbx4() {
		out.Format = "KeePass (KDBX 4)"
	}
	out.Items = keePassItems(db.Content)
	return out, nil
}

func readKeePassXML(data []byte) (Import, error) {
	var c gokeepasslib.DBContent
	if err := xml.Unmarshal(data, &c); err != nil || c.Root == nil {
		return Import{}, invalid("not a KeePass XML export")
	}
	return Import{Format: "KeePass (XML)", Items: keePassItems(&c)}, nil
}

func keePassItems(c *gokeepasslib.DBContent) []ImportItem {
	var bin gokeepasslib.UUID
	if c.Meta != nil && c.Meta.RecycleBinEnabled.Bool {
		bin = c.Meta.RecycleBinUUID
	}
	var out []ImportItem
	var walk func(g gokeepasslib.Group, path string, top bool)
	walk = func(g gokeepasslib.Group, path string, top bool) {
		if !top && !bin.IsZero() && g.UUID.Compare(bin) {
			return
		}
		if !top {
			path = joinGroup(path, g.Name)
		}
		for _, e := range g.Entries {
			out = append(out, keePassEntry(e, path))
		}
		for _, sub := range g.Groups {
			walk(sub, path, false)
		}
	}
	if c.Root != nil {
		// The root group is the database itself.
		for _, g := range c.Root.Groups {
			walk(g, "", true)
		}
	}
	return out
}

func joinGroup(path, name string) string {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return path
	case path == "":
		return name
	}
	return path + " / " + name
}

// KeePass fields that are not extra fields of the entry.
var keePassStd = map[string]bool{"Title": true, "UserName": true, "Password": true, "URL": true, "Notes": true}

func keePassEntry(e gokeepasslib.Entry, group string) ImportItem {
	it := ImportItem{Group: group}
	it.Title, it.Username, it.URL = e.GetContent("Title"), e.GetContent("UserName"), e.GetContent("URL")
	it.Data.Password, it.Data.Notes = e.GetContent("Password"), e.GetContent("Notes")
	if t := e.Times.CreationTime; t != nil && !t.Time.IsZero() {
		it.Created = t.Time.UTC()
	}
	fields := map[string]string{}
	var extra []string
	for _, v := range e.Values {
		if keePassStd[v.Key] {
			continue
		}
		fields[v.Key] = v.Value.Content
		if !keePassOTPField(v.Key) && strings.TrimSpace(v.Value.Content) != "" {
			extra = append(extra, v.Key+": "+v.Value.Content)
		}
	}
	if otp := keePassOTP(fields); otp != "" {
		it.Data.TOTP = &TOTP{Secret: otp}
	}
	if e.Tags != "" {
		extra = append(extra, "Tags: "+e.Tags)
	}
	it.Data.Notes = withExtra(it.Data.Notes, extra)
	return it
}

func withExtra(notes string, extra []string) string {
	if len(extra) == 0 {
		return notes
	}
	if strings.TrimSpace(notes) != "" {
		notes = strings.TrimRight(notes, "\n") + "\n\n"
	}
	return notes + strings.Join(extra, "\n")
}

func keePassOTPField(k string) bool {
	return k == "otp" || k == "TOTP Seed" || k == "TOTP Settings" || strings.HasPrefix(k, "TimeOtp-")
}

// keePassOTP turns the TOTP fields of KeePassXC ("otp": an otpauth://
// URI; older "TOTP Seed" + "TOTP Settings" = "30;6") or KeePass 2.47+
// ("TimeOtp-Secret-Base32" and friends) into a secret for ParseTOTP.
func keePassOTP(f map[string]string) string {
	if v := strings.TrimSpace(f["otp"]); v != "" {
		if strings.HasPrefix(strings.ToLower(v), "otpauth:") {
			return v
		}
		// KeePassXC also accepts "key=…&step=…&size=…" (KeeOTP style).
		if q, err := url.ParseQuery(v); err == nil && q.Get("key") != "" {
			return otpURI(q.Get("key"), q.Get("otpHashMode"), q.Get("size"), q.Get("step"))
		}
		return v
	}
	if v := strings.TrimSpace(f["TOTP Seed"]); v != "" {
		period, digits, _ := strings.Cut(f["TOTP Settings"], ";")
		return otpURI(v, "", digits, period)
	}
	var secret []byte
	switch {
	case f["TimeOtp-Secret-Base32"] != "":
		return otpURI(f["TimeOtp-Secret-Base32"], f["TimeOtp-Algorithm"], f["TimeOtp-Length"], f["TimeOtp-Period"])
	case f["TimeOtp-Secret-Hex"] != "":
		secret, _ = hex.DecodeString(strings.ReplaceAll(f["TimeOtp-Secret-Hex"], " ", ""))
	case f["TimeOtp-Secret-Base64"] != "":
		secret, _ = base64.StdEncoding.DecodeString(strings.TrimSpace(f["TimeOtp-Secret-Base64"]))
	case f["TimeOtp-Secret"] != "":
		secret = []byte(f["TimeOtp-Secret"])
	default:
		return ""
	}
	if len(secret) == 0 {
		return "-" // not readable: Prepare reports it
	}
	defer wipe(secret)
	return otpURI(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret),
		f["TimeOtp-Algorithm"], f["TimeOtp-Length"], f["TimeOtp-Period"])
}

// otpURI builds an otpauth://totp/ URI from separate parameters;
// algorithm may be "HMAC-SHA-256" (KeePass) or "SHA256".
func otpURI(secret, algorithm, digits, period string) string {
	q := url.Values{"secret": {strings.TrimSpace(secret)}}
	a := strings.ToUpper(strings.NewReplacer("HMAC", "", "-", "", " ", "").Replace(algorithm))
	if a != "" && a != "SHA1" {
		q.Set("algorithm", a)
	}
	if d := strings.TrimSpace(digits); d != "" {
		q.Set("digits", d)
	}
	if p := strings.TrimSpace(period); p != "" {
		q.Set("period", p)
	}
	return (&url.URL{Scheme: "otpauth", Host: "totp", Path: "/", RawQuery: q.Encode()}).String()
}

// --- Bitwarden JSON ---

type bwExport struct {
	Encrypted bool `json:"encrypted"`
	Folders   []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"folders"`
	Collections []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"collections"`
	Items *[]struct {
		Type          int      `json:"type"`
		Name          string   `json:"name"`
		Notes         string   `json:"notes"`
		FolderID      string   `json:"folderId"`
		CollectionIDs []string `json:"collectionIds"`
		CreationDate  string   `json:"creationDate"`
		Fields        []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"fields"`
		Login *struct {
			Username string `json:"username"`
			Password string `json:"password"`
			TOTP     string `json:"totp"`
			URIs     []struct {
				URI string `json:"uri"`
			} `json:"uris"`
		} `json:"login"`
		Card     map[string]any `json:"card"`
		Identity map[string]any `json:"identity"`
		SSHKey   *struct {
			PrivateKey string `json:"privateKey"`
		} `json:"sshKey"`
	} `json:"items"`
}

func readBitwardenJSON(data []byte) (Import, error) {
	var x bwExport
	if err := json.Unmarshal(data, &x); err != nil || x.Items == nil {
		if x.Encrypted {
			return Import{}, invalid("the Bitwarden export is encrypted: export it as unencrypted JSON or CSV")
		}
		return Import{}, invalid("not a Bitwarden JSON export")
	}
	if x.Encrypted {
		return Import{}, invalid("the Bitwarden export is encrypted: export it as unencrypted JSON or CSV")
	}
	folders := map[string]string{}
	for _, f := range x.Folders {
		folders[f.ID] = strings.ReplaceAll(f.Name, "/", " / ")
	}
	for _, c := range x.Collections {
		folders[c.ID] = strings.ReplaceAll(c.Name, "/", " / ")
	}
	var out []ImportItem
	for _, b := range *x.Items {
		it := ImportItem{Group: folders[b.FolderID]}
		if it.Group == "" && len(b.CollectionIDs) > 0 {
			it.Group = folders[b.CollectionIDs[0]]
		}
		it.Title, it.Data.Notes = b.Name, b.Notes
		if t, err := time.Parse(time.RFC3339, b.CreationDate); err == nil {
			it.Created = t.UTC()
		}
		var extra []string
		if l := b.Login; l != nil {
			it.Username, it.Data.Password = l.Username, l.Password
			for i, u := range l.URIs {
				if i == 0 {
					it.URL = u.URI
				} else if u.URI != "" {
					extra = append(extra, "URL: "+u.URI)
				}
			}
			if l.TOTP != "" {
				it.Data.TOTP = &TOTP{Secret: l.TOTP}
			}
		}
		if b.SSHKey != nil && b.SSHKey.PrivateKey != "" {
			it.Kind, it.Data.PrivateKey = KindSSH, b.SSHKey.PrivateKey
		}
		extra = append(extra, mapLines(b.Card)...)
		extra = append(extra, mapLines(b.Identity)...)
		for _, f := range b.Fields {
			if f.Value != "" {
				extra = append(extra, f.Name+": "+f.Value)
			}
		}
		it.Data.Notes = withExtra(it.Data.Notes, extra)
		out = append(out, it)
	}
	return Import{Format: "Bitwarden (JSON)", Items: out}, nil
}

// mapLines lists the filled fields of a card or identity, in a stable order.
func mapLines(m map[string]any) []string {
	var keys []string
	for k, v := range m {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var out []string
	for _, k := range keys {
		out = append(out, k+": "+m[k].(string))
	}
	return out
}

// --- CSV ---

// Column roles of a CSV export.
type col int

const (
	colNone col = iota
	colTitle
	colURL
	colUser
	colUser2 // Dashlane's username2 / email: used when there is no username
	colPassword
	colNotes
	colTOTP
	colGroup
	colType
	colCreated
)

// csvColumns maps a normalised header (lower case, letters and digits
// only) to its role. Unknown columns go into the notes.
var csvColumns = map[string]col{
	"title": colTitle, "name": colTitle, "account": colTitle, "itemname": colTitle, "accountname": colTitle,
	"url": colURL, "website": colURL, "loginuri": colURL, "uri": colURL, "urls": colURL, "webaddress": colURL,
	"username": colUser, "loginname": colUser, "loginusername": colUser, "login": colUser, "user": colUser,
	"email": colUser2, "username2": colUser2,
	"password": colPassword, "loginpassword": colPassword, "pass": colPassword,
	"notes": colNotes, "note": colNotes, "comments": colNotes, "comment": colNotes, "extra": colNotes,
	"totp": colTOTP, "otp": colTOTP, "otpauth": colTOTP, "logintotp": colTOTP, "otpsecret": colTOTP,
	"otpurl": colTOTP, "onetimepassword": colTOTP, "twofactorsecret": colTOTP,
	"group": colGroup, "folder": colGroup, "grouping": colGroup, "vault": colGroup, "category": colGroup,
	"type":    colType,
	"created": colCreated, "createtime": colCreated, "timecreated": colCreated, "creationdate": colCreated,
}

// Columns that are noise in notes.
var csvSkip = map[string]bool{
	"favorite": true, "fav": true, "reprompt": true, "archived": true, "guid": true, "icon": true,
	"lastmodified": true, "modifytime": true, "timelastused": true, "timepasswordchanged": true,
	"httprealm": true, "formactionorigin": true, "username3": true, "lastused": true, "modified": true,
}

func normHeader(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// csvFormat names the program that wrote a CSV export, from its header.
func csvFormat(h map[string]bool) string {
	switch {
	case h["loginuri"] && h["loginpassword"]:
		return "Bitwarden (CSV)"
	case h["httprealm"] || h["formactionorigin"]:
		return "Firefox"
	case h["grouping"] && h["extra"]:
		return "LastPass"
	case h["group"] && h["title"] && h["totp"] && h["lastmodified"]:
		return "KeePassXC (CSV)"
	case h["account"] && h["loginname"] && h["website"]:
		return "KeePass (CSV)"
	case h["username2"] || h["otpsecret"] && h["category"]:
		return "Dashlane"
	case h["vault"] && h["createtime"]:
		return "Proton Pass"
	case h["cardholdername"] || h["folder"] && h["note"] && h["name"] && h["url"] && !h["type"]:
		return "NordPass"
	case h["otpauth"] && (h["tags"] || h["archived"]):
		return "1Password"
	case h["otpauth"] && h["title"]:
		return "Safari / Apple Passwords"
	case h["name"] && h["url"] && h["username"] && h["password"] && h["note"] && len(h) <= 5:
		return "Chrome / Edge / Brave / Opera"
	}
	return "CSV"
}

func readCSV(data []byte) (Import, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		data = latin1(data)
	}
	first, _, _ := bytes.Cut(data, []byte("\n"))
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord, r.LazyQuotes = -1, true
	// Some programs (and spreadsheets in some locales) use ; or tabs.
	if n := bytes.Count(first, []byte(",")); bytes.Count(first, []byte(";")) > n {
		r.Comma = ';'
	} else if bytes.Count(first, []byte("\t")) > n {
		r.Comma = '\t'
	}
	header, err := r.Read()
	if err != nil {
		return Import{}, invalid("the file is not a CSV export, a KeePass database or a Bitwarden JSON export")
	}
	roles := make([]col, len(header))
	names := map[string]bool{}
	var hasPassword, hasName bool
	for i, h := range header {
		n := normHeader(h)
		names[n] = true
		roles[i] = csvColumns[n]
		if csvSkip[n] {
			roles[i] = -1
		}
		hasPassword = hasPassword || roles[i] == colPassword
		hasName = hasName || roles[i] == colTitle || roles[i] == colURL
	}
	if !hasPassword || !hasName {
		return Import{}, invalid("the CSV file has no password and name / URL columns: export it from the password manager again")
	}
	format := csvFormat(names)
	var out []ImportItem
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Import{}, invalid("the CSV file is damaged: " + err.Error())
		}
		it := ImportItem{}
		var extra []string
		var user2, typ string
		for i, v := range rec {
			if i >= len(roles) {
				break
			}
			if strings.TrimSpace(v) == "" {
				continue
			}
			switch roles[i] {
			case colTitle:
				it.Title = v
			case colURL:
				if it.URL == "" {
					// 1Password and Proton list several, one a line / comma.
					u, rest, _ := strings.Cut(strings.TrimSpace(v), "\n")
					it.URL = strings.TrimSpace(u)
					if rest != "" {
						extra = append(extra, header[i]+": "+rest)
					}
				} else {
					extra = append(extra, header[i]+": "+v)
				}
			case colUser:
				it.Username = v
			case colUser2:
				user2 = v
			case colPassword:
				it.Data.Password = v
			case colNotes:
				it.Data.Notes = v
			case colTOTP:
				it.Data.TOTP = &TOTP{Secret: v}
			case colGroup:
				it.Group = v
			case colType:
				typ = strings.ToLower(v)
			case colCreated:
				it.Created = parseTime(v)
			case colNone:
				extra = append(extra, header[i]+": "+v)
			}
		}
		if it.Username == "" {
			it.Username = user2
		} else if user2 != "" && user2 != it.Username {
			extra = append(extra, "Email: "+user2)
		}
		if format == "LastPass" && it.URL == "http://sn" {
			it.URL = "" // a secure note
		}
		if format == "KeePassXC (CSV)" {
			// "Root/Internet": the first part is the database.
			_, rest, _ := strings.Cut(it.Group, "/")
			it.Group = rest
		}
		if strings.Contains(it.Group, "/") || strings.Contains(it.Group, "\\") {
			it.Group = strings.Join(strings.FieldsFunc(it.Group, func(r rune) bool { return r == '/' || r == '\\' }), " / ")
		}
		if typ != "" && typ != "login" && typ != "password" && typ != "note" && typ != "securenote" {
			extra = append(extra, "Type: "+typ)
		}
		it.Data.Notes = withExtra(it.Data.Notes, extra)
		out = append(out, it)
	}
	return Import{Format: format, Items: out}, nil
}

// latin1 reads Windows-1252 / Latin-1 text (old KeePass CSV) as UTF-8.
func latin1(b []byte) []byte {
	var out bytes.Buffer
	for _, c := range b {
		out.WriteRune(rune(c))
	}
	return out.Bytes()
}

// parseTime reads the dates of exports: RFC 3339, Unix seconds or
// milliseconds, "2006-01-02 15:04:05".
func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		if n > 1e12 {
			return time.UnixMilli(n).UTC()
		}
		return time.Unix(n, 0).UTC()
	}
	for _, f := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
