package secrets

import (
	"encoding/base64"
	"net/url"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func otpParams(secret []byte, name, issuer string, algo, digits, typ uint64) []byte {
	var m []byte
	m = protowire.AppendTag(m, otpSecret, protowire.BytesType)
	m = protowire.AppendBytes(m, secret)
	m = protowire.AppendTag(m, otpName, protowire.BytesType)
	m = protowire.AppendString(m, name)
	if issuer != "" {
		m = protowire.AppendTag(m, otpIssuer, protowire.BytesType)
		m = protowire.AppendString(m, issuer)
	}
	for num, v := range map[protowire.Number]uint64{otpAlgorithm: algo, otpDigits: digits, otpType: typ} {
		m = protowire.AppendTag(m, num, protowire.VarintType)
		m = protowire.AppendVarint(m, v)
	}
	return m
}

func TestMigration(t *testing.T) {
	// The example of the format's public write-ups: "Example:alice@google.com".
	links, b, err := ParseMigration("otpauth-migration://offline?data=CjEKCkhlbGxvId6tvu8SGEV4YW1wbGU6YWxpY2VAZ29vZ2xlLmNvbRoHRXhhbXBsZSABKAEwAhABGAEgACjr4JKK%2Bf%2F%2F%2F%2F8B")
	if err != nil || len(links) != 1 || b.Size != 1 || b.Index != 0 {
		t.Fatalf("%+v %+v %v", links, b, err)
	}
	g, info, err := ParseTOTP(links[0].Link, TOTP{})
	if err != nil || g.Secret != "JBSWY3DPEHPK3PXP" || g.Algorithm != "SHA1" || g.Digits != 6 ||
		info != (TOTPInfo{Issuer: "Example", Account: "alice@google.com"}) {
		t.Fatalf("%s: %+v %+v %v", links[0].Link, g, info, err)
	}

	var p []byte
	for _, e := range [][]byte{
		otpParams([]byte("12345678901234567890"), "Acme:bob", "", 2, 2, 2), // SHA256, 8 digits; issuer in the name
		otpParams([]byte("abcdefghij"), "carol?#x", "Shop", 1, 1, 2),       // characters to escape
		otpParams([]byte("abcdefghij"), "dave", "Bank", 1, 1, 1),           // HOTP
		otpParams([]byte("12345678901234567890"), "Acme:bob", "", 2, 2, 2), // a repeat
	} {
		p = protowire.AppendTag(p, migOTP, protowire.BytesType)
		p = protowire.AppendBytes(p, e)
	}
	for num, v := range map[protowire.Number]uint64{migBatchSize: 2, migBatchIndex: 1, migBatchID: 77} {
		p = protowire.AppendTag(p, num, protowire.VarintType)
		p = protowire.AppendVarint(p, v)
	}
	link := "otpauth-migration://offline?data=" + url.QueryEscape(base64.StdEncoding.EncodeToString(p))
	all, batches := ReadTOTPLinks("junk\n" + link + "  otpauth://totp/X?secret=JBSWY3DP " + link)
	if len(all) != 4 || len(batches) != 2 || batches[0] != (Batch{ID: 77, Index: 1, Size: 2}) {
		t.Fatalf("%+v %+v", all, batches)
	}
	g, info, err = ParseTOTP(all[0].Link, TOTP{})
	if err != nil || g.Algorithm != "SHA256" || g.Digits != 8 || info != (TOTPInfo{Issuer: "Acme", Account: "bob"}) {
		t.Fatalf("%s: %+v %+v %v", all[0].Link, g, info, err)
	}
	if _, info, err = ParseTOTP(all[1].Link, TOTP{}); err != nil || info != (TOTPInfo{Issuer: "Shop", Account: "carol?#x"}) {
		t.Fatalf("%s: %+v %v", all[1].Link, info, err)
	}
	if all[2].Link != "" || all[2].Error == nil || all[2].Name != "Bank / dave" {
		t.Fatalf("HOTP: %+v", all[2])
	}
	if all[3].Link != "otpauth://totp/X?secret=JBSWY3DP" {
		t.Fatalf("plain link: %+v", all[3])
	}
	// A '+' that became a space, and a cut-off link.
	if _, _, err := ParseMigration("otpauth-migration://offline?data=" + base64.StdEncoding.EncodeToString(p)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"otpauth-migration://offline", "otpauth-migration://offline?data=CjEKCkhl", "otpauth-migration://offline?data=%%%"} {
		if _, _, err := ParseMigration(bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
}
