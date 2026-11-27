package engine

import (
	"fmt"
	"path/filepath"

	"cloudsync/core/index"
	"cloudsync/core/secrets"
	"cloudsync/core/vault"
)

// OfflineSafe is the password safe read from this device's own copies
// (journal copies and pending ops in the data dir), for when the engine
// cannot open: no storage answers, or fewer than k+m are left. It only
// reads; nothing is sent anywhere and the data dir is not locked.
// It also reads the records of connected apps.
type OfflineSafe struct {
	safeView
	appsView
}

// OpenOfflineSafe reads the safe of v from dataDir. A data dir that was
// never used gives an empty safe.
func OpenOfflineSafe(v *vault.Vault, dataDir string) (*OfflineSafe, error) {
	e := &Engine{opt: Options{DataDir: dataDir}}
	if err := e.loadState(); err != nil {
		return nil, err
	}
	cfg := v.Config()
	if e.state.VaultID != "" && e.state.VaultID != cfg.VaultID {
		return nil, fmt.Errorf("engine: %s belongs to another vault", dataDir)
	}
	key := cfg.Key()
	j := index.NewJournal(e.state.Device, key, nil, 1)
	j.CacheDir = filepath.Join(dataDir, "journal")
	idx := index.New()
	idx.Add(e.state.Pending...)
	idx.Add(j.LoadCached()...)
	return &OfflineSafe{safeView{idx: idx, keys: secrets.NewKeys(key)}, appsView{idx, key}}, nil
}
