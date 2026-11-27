package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloudsync/core/crypto"
	"cloudsync/core/index"
	"cloudsync/core/secrets"
)

// ImportOptions says where imported entries go.
type ImportOptions struct {
	// Group every entry goes to; "" = see KeepGroups.
	Group string
	// With no Group: put entries into groups named like their folders in
	// the export (made when missing); otherwise they get no group.
	KeepGroups bool
	// Only work out what would be done.
	DryRun bool
}

// ImportStatus is what happens to one item of an import.
type ImportStatus int

const (
	ImportAdded     ImportStatus = iota
	ImportDuplicate              // the safe already has the same entry
	ImportFailed                 // item.Error
)

// ImportReport is the outcome of ImportSecrets.
type ImportReport struct {
	Status    []ImportStatus // one per item
	Group     []string       // name of the group each item goes to ("" = none)
	Added     int
	Duplicate int
	Failed    int
	NewGroups []string // groups made for the import
}

// ImportSecrets adds the items read by secrets.ImportFile. An item equal
// to an entry of the safe (or an earlier item) is skipped, so importing
// the same file again adds nothing. All entries go in one commit.
func (e *Engine) ImportSecrets(ctx context.Context, items []secrets.ImportItem, opt ImportOptions) (ImportReport, error) {
	k := e.secretKeys()
	safe := e.Safe()
	groupByName := map[string]string{}
	groupName := map[string]string{}
	for _, g := range safe.Groups {
		groupName[g.ID] = g.Name
		if _, ok := groupByName[strings.ToLower(g.Name)]; !ok {
			groupByName[strings.ToLower(g.Name)] = g.ID
		}
	}
	if opt.Group != "" {
		if _, ok := groupName[opt.Group]; !ok {
			return ImportReport{}, fmt.Errorf("group %s: %w", opt.Group, ErrNotFound)
		}
	}
	// Entries of the safe by what a duplicate must share; their secret
	// parts are opened only when an item matches.
	existing := map[string][]secrets.Entry{}
	for _, en := range safe.Entries {
		existing[dupKey(en.Meta)] = append(existing[dupKey(en.Meta)], en)
	}
	opened := map[string]*secrets.Data{}
	sameData := func(en secrets.Entry, d *secrets.Data) bool {
		od, ok := opened[en.ID]
		if !ok {
			if s, found := e.idx.Secret(en.ID); found {
				if full, err := k.OpenEntry(s, true); err == nil {
					od = full.Data
				}
			}
			opened[en.ID] = od
		}
		return od != nil && sameSecret(en.Kind, *od, *d)
	}
	importedKeys := map[string][]*secrets.Data{}
	newGroup := map[string]bool{} // made for the import, not committed yet

	rep := ImportReport{Status: make([]ImportStatus, len(items)), Group: make([]string, len(items))}
	var groupOps, entryOps []index.Op
	now := time.Now().UTC()
	for i := range items {
		it := &items[i]
		if it.Error != nil {
			rep.Status[i] = ImportFailed
			rep.Failed++
			continue
		}
		m := it.Meta
		switch {
		case opt.Group != "":
			m.Group = opt.Group
		case opt.KeepGroups && it.Group != "":
			id, ok := groupByName[strings.ToLower(it.Group)]
			if !ok {
				id = crypto.RandomID(12)
				groupByName[strings.ToLower(it.Group)] = id
				groupName[id] = it.Group
				newGroup[id] = true
			}
			m.Group = id
		default:
			m.Group = ""
		}
		rep.Group[i] = groupName[m.Group]

		key := dupKey(m)
		dup := false
		for _, en := range existing[key] {
			if sameData(en, &it.Data) {
				dup = true
				break
			}
		}
		for _, d := range importedKeys[key] {
			if !dup && sameSecret(m.Kind, *d, it.Data) {
				dup = true
			}
		}
		if dup {
			rep.Status[i] = ImportDuplicate
			rep.Duplicate++
			continue
		}
		importedKeys[key] = append(importedKeys[key], &it.Data)
		rep.Added++
		if newGroup[m.Group] {
			// A group is made only for entries that are added.
			newGroup[m.Group] = false
			rep.NewGroups = append(rep.NewGroups, groupName[m.Group])
			g := secrets.GroupMeta{Name: groupName[m.Group]}
			if err := secrets.PrepareGroup(&g); err != nil {
				return ImportReport{}, err
			}
			ch, err := k.SealGroup(m.Group, g)
			if err != nil {
				return ImportReport{}, err
			}
			groupOps = append(groupOps, index.Op{Kind: index.OpSecret, Path: "/", Secret: ch})
		}
		if opt.DryRun {
			continue
		}
		if m.Created.IsZero() || m.Created.After(now) {
			m.Created = now
		}
		m.Updated = now
		ch, err := k.SealEntry(crypto.RandomID(12), m, it.Data)
		if err != nil {
			return ImportReport{}, err
		}
		entryOps = append(entryOps, index.Op{Kind: index.OpSecret, Path: "/", Secret: ch})
	}
	if opt.DryRun || len(entryOps) == 0 {
		return rep, nil
	}
	if err := e.commit(ctx, append(groupOps, entryOps...)); err != nil {
		return ImportReport{}, err
	}
	return rep, nil
}

// dupKey is what an imported item must share with an entry to be its
// duplicate; the secret part is compared then (sameSecret).
func dupKey(m secrets.Meta) string {
	return strings.Join([]string{string(m.Kind), strings.ToLower(strings.TrimSpace(m.Title)),
		strings.TrimSpace(m.Username), strings.TrimSpace(m.URL)}, "\x00")
}

func sameSecret(k secrets.Kind, a, b secrets.Data) bool {
	switch k {
	case secrets.KindTOTP:
		return a.TOTP != nil && b.TOTP != nil && *a.TOTP == *b.TOTP
	case secrets.KindSSH, secrets.KindWeb3:
		return a.PrivateKey == b.PrivateKey
	}
	return a.Password == b.Password && strings.TrimSpace(a.Notes) == strings.TrimSpace(b.Notes)
}
