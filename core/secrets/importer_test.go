package secrets

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

const testOTP = "JBSWY3DPEHPK3PXP"

func find(t *testing.T, imp Import, kind Kind, title string) ImportItem {
	t.Helper()
	for _, it := range imp.Items {
		if it.Kind == kind && it.Title == title {
			return it
		}
	}
	t.Fatalf("no %s %q in %+v", kind, title, imp.Items)
	return ImportItem{}
}

func TestImportChromeCSV(t *testing.T) {
	csv := "\xef\xbb\xbfname,url,username,password,note\n" +
		"github.com,https://github.com/login,me,pa,ss,\n" +
		`mail,https://mail.example.com,"a@b.c","p""q","line1` + "\nline2\"\n" +
		",https://www.example.org/x,user,secret,\n" +
		",,,,\n"
	imp, err := ImportFile("Chrome Passwords.csv", []byte(csv), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if imp.Format != "Chrome / Edge / Brave / Opera" || len(imp.Items) != 3 {
		t.Fatalf("%s %+v", imp.Format, imp.Items)
	}
	// A broken row (extra comma) still keeps what it can.
	m := find(t, imp, KindPassword, "mail")
	if m.Username != "a@b.c" || m.Data.Password != `p"q` || m.Data.Notes != "line1\nline2" || m.Error != nil {
		t.Fatalf("%+v", m)
	}
	if o := find(t, imp, KindPassword, "example.org"); o.Data.Password != "secret" {
		t.Fatalf("title from URL: %+v", o)
	}
}

func TestImportFirefoxCSV(t *testing.T) {
	csv := `"url","username","password","httpRealm","formActionOrigin","guid","timeCreated","timeLastUsed","timePasswordChanged"
"https://accounts.example.com","me","pw","","https://accounts.example.com","{1}","1700000000000","1700000000000","1700000000000"
`
	imp, err := ImportFile("logins.csv", []byte(csv), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	it := find(t, imp, KindPassword, "accounts.example.com")
	if imp.Format != "Firefox" || it.Data.Notes != "" || it.Created.Unix() != 1700000000 {
		t.Fatalf("%s %+v", imp.Format, it)
	}
}

func TestImportBitwardenCSVWithTOTP(t *testing.T) {
	csv := "folder,favorite,type,name,notes,fields,reprompt,login_uri,login_username,login_password,login_totp\n" +
		"Work/Mail,1,login,Mail,,custom: 1,0,https://mail.example.com,me,pw," + testOTP + "\n" +
		",,login,Bad 2FA,,,0,,me,pw2,not-base32!\n"
	imp, err := ImportFile("bitwarden.csv", []byte(csv), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if imp.Format != "Bitwarden (CSV)" || len(imp.Items) != 4 {
		t.Fatalf("%s %+v", imp.Format, imp.Items)
	}
	login := find(t, imp, KindPassword, "Mail")
	if login.Group != "Work / Mail" || login.Data.Notes != "fields: custom: 1" || login.Data.TOTP != nil {
		t.Fatalf("%+v", login)
	}
	otp := find(t, imp, KindTOTP, "Mail")
	if otp.Error != nil || otp.Data.TOTP.Secret != testOTP || otp.Username != "me" || otp.Group != "Work / Mail" {
		t.Fatalf("%+v", otp)
	}
	if bad := find(t, imp, KindTOTP, "Bad 2FA"); bad.Error == nil || bad.Data.TOTP != nil {
		t.Fatalf("bad TOTP accepted: %+v", bad)
	}
	if find(t, imp, KindPassword, "Bad 2FA").Error != nil {
		t.Fatal("the login of a bad TOTP is lost")
	}
}

func TestImportOtherCSVs(t *testing.T) {
	for _, c := range []struct{ format, csv, title string }{
		{"KeePassXC (CSV)", `"Group","Title","Username","Password","URL","Notes","TOTP","Icon","Last Modified","Created"
"Root/Internet","Site","me","pw","https://s.example","","otpauth://totp/S:me?secret=` + testOTP + `","0","2024-01-01T00:00:00Z","2024-01-01T00:00:00Z"
`, "Site"},
		{"KeePass (CSV)", `"Account","Login Name","Password","Web Site","Comments"
"Site","me","pw","https://s.example",""
`, "Site"},
		{"LastPass", "url,username,password,totp,extra,name,grouping,fav\nhttp://sn,,,,secret note,Note,Personal,0\n", "Note"},
		{"1Password", "Title,Url,Username,Password,OTPAuth,Favorite,Archived,Tags,Notes\nSite,https://s.example,me,pw,,false,false,a,\n", "Site"},
		{"Safari / Apple Passwords", "Title,URL,Username,Password,Notes,OTPAuth\nSite,https://s.example,me,pw,,\n", "Site"},
		{"Dashlane", "username,username2,username3,title,password,note,url,category,otpSecret\nme,me@x,,Site,pw,,s.example,Work,\n", "Site"},
		{"Proton Pass", "type,name,url,email,username,password,note,totp,createTime,modifyTime,vault\nlogin,Site,https://s.example,me@x,,pw,,,1700000000,1700000000,Personal\n", "Site"},
		{"CSV", "Name;Password;Website\nSite;pw;https://s.example\n", "Site"},
	} {
		imp, err := ImportFile("x.csv", []byte(c.csv), "", nil)
		if err != nil {
			t.Fatalf("%s: %v", c.format, err)
		}
		if imp.Format != c.format {
			t.Errorf("format %q, want %q", imp.Format, c.format)
		}
		it := find(t, imp, KindPassword, c.title)
		if it.Error != nil {
			t.Errorf("%s: %v", c.format, it.Error)
		}
		switch c.format {
		case "KeePassXC (CSV)":
			if it.Group != "Internet" || find(t, imp, KindTOTP, "Site").Error != nil {
				t.Errorf("KeePassXC: %+v", imp.Items)
			}
		case "LastPass":
			if it.URL != "" || it.Data.Notes != "secret note" || it.Group != "Personal" {
				t.Errorf("LastPass note: %+v", it)
			}
		case "Dashlane":
			if it.Username != "me" || !strings.Contains(it.Data.Notes, "me@x") || it.Group != "Work" {
				t.Errorf("Dashlane: %+v", it)
			}
		case "Proton Pass":
			if it.Username != "me@x" || it.Group != "Personal" {
				t.Errorf("Proton: %+v", it)
			}
		}
	}
}

func TestImportRejects(t *testing.T) {
	for name, data := range map[string]string{
		"empty":           "",
		"no password":     "name,url\na,b\n",
		"encrypted bw":    `{"encrypted":true,"passwordProtected":true,"data":"x"}`,
		"zip":             "PK\x03\x04....",
		"broken kdbx":     "\x03\xd9\xa2\x9a\x67\xfb\x4b\xb5garbage",
		"not keepass xml": "<html></html>",
	} {
		_, err := ImportFile("f", []byte(data), "pw", nil)
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestImportBitwardenJSON(t *testing.T) {
	js := `{"encrypted":false,"folders":[{"id":"f1","name":"Work"}],"items":[
	 {"type":1,"name":"Mail","notes":"n","folderId":"f1","login":{"username":"me","password":"pw","totp":"otpauth://totp/Mail:me?secret=` + testOTP + `","uris":[{"uri":"https://a"},{"uri":"https://b"}]},
	  "fields":[{"name":"PIN","value":"1234"}],"creationDate":"2023-05-01T10:00:00.000Z"},
	 {"type":3,"name":"Card","card":{"cardholderName":"Me","number":"4111"}},
	 {"type":2,"name":"Note","notes":"just text"}]}`
	imp, err := ImportFile("bitwarden.json", []byte(js), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	mail := find(t, imp, KindPassword, "Mail")
	if mail.Group != "Work" || mail.URL != "https://a" || mail.Data.Notes != "n\n\nURL: https://b\nPIN: 1234" || mail.Created.Year() != 2023 {
		t.Fatalf("%+v", mail)
	}
	find(t, imp, KindTOTP, "Mail")
	if c := find(t, imp, KindPassword, "Card"); c.Data.Notes != "cardholderName: Me\nnumber: 4111" {
		t.Fatalf("%+v", c)
	}
	find(t, imp, KindPassword, "Note")
}

func keePassDB(t *testing.T, password string) []byte {
	t.Helper()
	val := func(k, v string, protected bool) gokeepasslib.ValueData {
		return gokeepasslib.ValueData{Key: k, Value: gokeepasslib.V{Content: v, Protected: w.NewBoolWrapper(protected)}}
	}
	entry := func(title, user, pass string, extra ...gokeepasslib.ValueData) gokeepasslib.Entry {
		e := gokeepasslib.NewEntry()
		e.Values = append(e.Values, val("Title", title, false), val("UserName", user, false), val("Password", pass, true))
		e.Values = append(e.Values, extra...)
		return e
	}
	root := gokeepasslib.NewGroup()
	root.Name = "Database"
	root.Entries = []gokeepasslib.Entry{entry("Top", "me", "pw-top")}
	inet := gokeepasslib.NewGroup()
	inet.Name = "Internet"
	inet.Entries = []gokeepasslib.Entry{entry("Site", "me", "pw-site",
		val("URL", "https://s.example", false),
		val("otp", "otpauth://totp/Site:me?secret="+testOTP+"&period=30", true),
		val("PIN", "4321", true))}
	mail := gokeepasslib.NewGroup()
	mail.Name = "Mail"
	mail.Entries = []gokeepasslib.Entry{entry("KP2", "me", "pw-kp2",
		val("TimeOtp-Secret-Base32", testOTP, true), val("TimeOtp-Length", "8", false), val("TimeOtp-Algorithm", "HMAC-SHA-256", false))}
	inet.Groups = []gokeepasslib.Group{mail}
	bin := gokeepasslib.NewGroup()
	bin.Name = "Recycle Bin"
	bin.Entries = []gokeepasslib.Entry{entry("Deleted", "me", "gone")}
	root.Groups = []gokeepasslib.Group{inet, bin}

	db := gokeepasslib.NewDatabase(gokeepasslib.WithDatabaseKDBXVersion4())
	db.Credentials = gokeepasslib.NewPasswordCredentials(password)
	db.Content.Meta.RecycleBinEnabled = w.NewBoolWrapper(true)
	db.Content.Meta.RecycleBinUUID = bin.UUID
	db.Content.Root = &gokeepasslib.RootData{Groups: []gokeepasslib.Group{root}}
	if err := db.LockProtectedEntries(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := gokeepasslib.NewEncoder(&buf).Encode(db); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImportKDBX(t *testing.T) {
	data := keePassDB(t, "master")
	if _, err := ImportFile("db.kdbx", data, "", nil); !errors.Is(err, ErrImportPassword) {
		t.Fatalf("no password: %v", err)
	}
	if _, err := ImportFile("db.kdbx", data, "wrong", nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong password: %v", err)
	}
	imp, err := ImportFile("db.kdbx", data, "master", nil)
	if err != nil {
		t.Fatal(err)
	}
	if imp.Format != "KeePass (KDBX 4)" || len(imp.Items) != 5 {
		t.Fatalf("%s %+v", imp.Format, imp.Items)
	}
	if top := find(t, imp, KindPassword, "Top"); top.Group != "" || top.Data.Password != "pw-top" {
		t.Fatalf("%+v", top)
	}
	site := find(t, imp, KindPassword, "Site")
	if site.Group != "Internet" || site.Data.Password != "pw-site" || site.Data.Notes != "PIN: 4321" || site.URL != "https://s.example" {
		t.Fatalf("%+v", site)
	}
	if o := find(t, imp, KindTOTP, "Site"); o.Error != nil || o.Data.TOTP.Secret != testOTP {
		t.Fatalf("%+v", o)
	}
	kp2 := find(t, imp, KindTOTP, "KP2")
	if kp2.Error != nil || kp2.Group != "Internet / Mail" || kp2.Data.TOTP.Digits != 8 || kp2.Data.TOTP.Algorithm != "SHA256" {
		t.Fatalf("%+v %v", kp2.Data.TOTP, kp2.Error)
	}
	for _, it := range imp.Items {
		if it.Title == "Deleted" {
			t.Fatal("the recycle bin is imported")
		}
	}
}

func TestImportKeePassXML(t *testing.T) {
	x := `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<KeePassFile><Meta><RecycleBinEnabled>False</RecycleBinEnabled></Meta><Root><Group><Name>DB</Name>
<Group><Name>Banking</Name><Entry>
 <String><Key>Title</Key><Value>Bank</Value></String>
 <String><Key>UserName</Key><Value>me</Value></String>
 <String><Key>Password</Key><Value ProtectInMemory="True">pw &amp; more</Value></String>
</Entry></Group></Group></Root></KeePassFile>`
	imp, err := ImportFile("export.xml", []byte(x), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if b := find(t, imp, KindPassword, "Bank"); b.Group != "Banking" || b.Data.Password != "pw & more" || imp.Format != "KeePass (XML)" {
		t.Fatalf("%s %+v", imp.Format, b)
	}
}
