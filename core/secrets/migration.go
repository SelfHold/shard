package secrets

import (
	"encoding/base64"
	"net/url"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
)

// Batch tells which part of a Google Authenticator export a link was:
// a large export is split over several QR codes.
type Batch struct {
	ID, Index, Size int
}

// TOTPLink is one account found in pasted text: an otpauth://totp/ link
// ready for PutSecret, or why the account cannot be imported.
type TOTPLink struct {
	Link  string
	Name  string // issuer / account, for an error without a link
	Error error
}

// ReadTOTPLinks finds the otpauth://totp/ links in pasted text (split by
// white space) and expands the otpauth-migration:// links of a Google
// Authenticator export into such links. Repeats are dropped.
func ReadTOTPLinks(text string) ([]TOTPLink, []Batch) {
	var links []TOTPLink
	var batches []Batch
	seen := map[string]bool{}
	add := func(l TOTPLink) {
		key := l.Link
		if key == "" {
			key = "error:" + l.Name
		}
		if !seen[key] {
			seen[key] = true
			links = append(links, l)
		}
	}
	for _, s := range strings.Fields(text) {
		low := strings.ToLower(s)
		switch {
		case strings.HasPrefix(low, "otpauth-migration:"):
			found, b, err := ParseMigration(s)
			if err != nil {
				add(TOTPLink{Name: "otpauth-migration", Error: err})
				continue
			}
			batches = append(batches, b)
			for _, l := range found {
				add(l)
			}
		case strings.HasPrefix(low, "otpauth:"):
			add(TOTPLink{Link: s})
		}
	}
	return links, batches
}

// Fields of the export (Google Authenticator's MigrationPayload).
const (
	migOTP        = 1 // repeated OtpParameters
	migBatchSize  = 3
	migBatchIndex = 4
	migBatchID    = 5

	otpSecret    = 1
	otpName      = 2
	otpIssuer    = 3
	otpAlgorithm = 4 // 1 SHA1, 2 SHA256, 3 SHA512, 4 MD5
	otpDigits    = 5 // 1 six, 2 eight
	otpType      = 6 // 1 HOTP, 2 TOTP
)

// ParseMigration reads an otpauth-migration://offline?data=… link.
func ParseMigration(link string) ([]TOTPLink, Batch, error) {
	var b Batch
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil {
		return nil, b, invalid("not an otpauth-migration:// link")
	}
	data := u.Query().Get("data")
	if data == "" {
		return nil, b, invalid("the export link has no data")
	}
	// A '+' that was not escaped reads back as a space.
	data = strings.ReplaceAll(data, " ", "+")
	raw, err := decodeBase64(data)
	if err != nil {
		return nil, b, invalid("the export data is not base64: copy the whole link")
	}
	defer wipe(raw)
	var out []TOTPLink
	err = eachField(raw, func(num protowire.Number, v uint64, msg []byte) error {
		switch num {
		case migOTP:
			out = append(out, migrationEntry(msg))
		case migBatchSize:
			b.Size = int(v)
		case migBatchIndex:
			b.Index = int(v)
		case migBatchID:
			b.ID = int(int32(v))
		}
		return nil
	})
	if err != nil {
		return nil, b, invalid("the export data is damaged: copy the whole link")
	}
	if len(out) == 0 {
		return nil, b, invalid("the export holds no accounts")
	}
	return out, b, nil
}

func decodeBase64(s string) ([]byte, error) {
	var err error
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		var b []byte
		if b, err = enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, err
}

// eachField walks the top-level fields of a protobuf message: varints
// come as v, length-delimited fields as msg; other types are skipped.
func eachField(b []byte, f func(num protowire.Number, v uint64, msg []byte) error) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return protowire.ParseError(n)
		}
		b = b[n:]
		var v uint64
		var msg []byte
		switch typ {
		case protowire.VarintType:
			v, n = protowire.ConsumeVarint(b)
		case protowire.BytesType:
			msg, n = protowire.ConsumeBytes(b)
		default:
			n = protowire.ConsumeFieldValue(num, typ, b)
		}
		if n < 0 {
			return protowire.ParseError(n)
		}
		b = b[n:]
		if err := f(num, v, msg); err != nil {
			return err
		}
	}
	return nil
}

func migrationEntry(msg []byte) TOTPLink {
	var secret []byte
	var name, issuer string
	var algo, digits, typ uint64
	err := eachField(msg, func(num protowire.Number, v uint64, m []byte) error {
		switch num {
		case otpSecret:
			secret = m
		case otpName:
			name = string(m)
		case otpIssuer:
			issuer = string(m)
		case otpAlgorithm:
			algo = v
		case otpDigits:
			digits = v
		case otpType:
			typ = v
		}
		return nil
	})
	account := strings.TrimSpace(name)
	if issuer == "" {
		if i := strings.IndexByte(account, ':'); i >= 0 {
			issuer, account = strings.TrimSpace(account[:i]), strings.TrimSpace(account[i+1:])
		}
	} else {
		account = strings.TrimSpace(strings.TrimPrefix(account, issuer+":"))
	}
	shown := strings.Trim(issuer+" / "+account, " /")
	switch {
	case err != nil:
		return TOTPLink{Name: shown, Error: invalid("the account is damaged in the export")}
	case typ == 1:
		return TOTPLink{Name: shown, Error: invalidf("%s: counter-based (HOTP) codes are not supported", shown)}
	case len(secret) == 0:
		return TOTPLink{Name: shown, Error: invalidf("%s: no secret in the export", shown)}
	}
	q := url.Values{"secret": {b32.EncodeToString(secret)}}
	switch algo {
	case 0, 1:
	case 2:
		q.Set("algorithm", "SHA256")
	case 3:
		q.Set("algorithm", "SHA512")
	default:
		return TOTPLink{Name: shown, Error: invalidf("%s: MD5 codes are not supported", shown)}
	}
	if digits == 2 {
		q.Set("digits", "8")
	}
	label := account
	if issuer != "" {
		q.Set("issuer", issuer)
		label = issuer + ":" + account
	}
	u := url.URL{Scheme: "otpauth", Host: "totp", Path: "/" + label, RawQuery: q.Encode()}
	return TOTPLink{Link: u.String(), Name: shown}
}
