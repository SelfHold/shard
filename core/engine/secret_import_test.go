package engine

import (
	"context"
	"testing"

	"cloudsync/core/secrets"
)

func TestImportSecrets(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	a, b := w.device("laptop"), w.device("phone")

	work, err := a.PutSecretGroup(ctx, "", secrets.GroupMeta{Name: "Work"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.PutSecret(ctx, "", "", secrets.Meta{Kind: secrets.KindPassword, Title: "Mail", Username: "me"},
		secrets.Data{Password: "old"}); err != nil {
		t.Fatal(err)
	}
	csv := "folder,favorite,type,name,notes,fields,reprompt,login_uri,login_username,login_password,login_totp\n" +
		"work,,login,Mail,,,,,me,old,\n" + // the same as in the safe
		"work,,login,Mail,,,,,me,new,JBSWY3DPEHPK3PXP\n" + // another password: added
		"Home,,login,Router,,,,,admin,pw,\n" +
		"Home,,login,Router,,,,,admin,pw,\n" + // repeated in the file
		",,login,Bad,,,,,me,pw,!!\n"
	imp, err := secrets.ImportFile("bw.csv", []byte(csv), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	opt := ImportOptions{KeepGroups: true, DryRun: true}
	dry, err := a.ImportSecrets(ctx, imp.Items, opt)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Added != 4 || dry.Duplicate != 2 || dry.Failed != 1 || len(dry.NewGroups) != 1 || dry.NewGroups[0] != "Home" {
		t.Fatalf("dry run: %+v", dry)
	}
	if len(a.Safe().Entries) != 1 {
		t.Fatal("a dry run saved entries")
	}

	opt.DryRun = false
	rep, err := a.ImportSecrets(ctx, imp.Items, opt)
	if err != nil || rep.Added != 4 {
		t.Fatalf("%+v %v", rep, err)
	}
	if err := b.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	list := b.Safe()
	if len(list.Groups) != 2 || len(list.Entries) != 5 {
		t.Fatalf("other device: %+v", list)
	}
	groups := map[string]string{}
	for _, g := range list.Groups {
		groups[g.ID] = g.Name
	}
	for _, en := range list.Entries {
		want := map[string]string{"Mail": "Work", "Router": "Home", "Bad": ""}[en.Title]
		if en.Title == "Mail" && en.Kind == secrets.KindPassword {
			full, err := b.OpenSecret(en.ID)
			if err != nil {
				t.Fatal(err)
			}
			if full.Data.Password == "old" {
				want = "" // the one made by hand
			} else if en.Group != work.ID {
				t.Fatalf("not put into the existing group: %+v", en)
			}
		}
		if groups[en.Group] != want {
			t.Fatalf("%s %s in group %q, want %q", en.Kind, en.Title, groups[en.Group], want)
		}
	}

	// Importing again adds nothing; into a chosen group, nothing either.
	again, err := b.ImportSecrets(ctx, imp.Items, opt)
	if err != nil || again.Added != 0 || len(again.NewGroups) != 0 {
		t.Fatalf("again: %+v %v", again, err)
	}
	if _, err := b.ImportSecrets(ctx, imp.Items, ImportOptions{Group: "nope"}); err == nil {
		t.Fatal("unknown group accepted")
	}
}
