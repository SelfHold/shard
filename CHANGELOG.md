# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Before 1.0.0 the on-storage format and the gRPC API may change between minor versions.

## [Unreleased]

### Changed

- The project is now called **Shard** (formerly CloudSync) and lives at
  <https://github.com/SelfHold/shard>. Identifiers that are stored on devices or
  published (Go module `cloudsync`, `cloudsyncd`, `vault.cs`, the Android package, the
  AIDL names) keep the old name.
- Android notifications and the file picker root follow the language chosen in the
  app (English by default) instead of the system locale.
- Documentation for the public repository: English README with a Russian translation,
  `docs/ARCHITECTURE.md` with the invariants, updated CONTRIBUTING and SECURITY.
  Windows is no longer documented as a platform (its builds are not maintained).

### Added

- **Connected apps** (Connector API v1, `proto/connector.proto`, `docs/CONNECTOR.md`):
  other apps keep small records (settings files) in the vault, the same on all devices.
  Pairing per device with a 6-digit code (app UI dialog, Vault → Connected apps,
  `cs apps`), compare-and-swap writes, concurrent offline writes kept as versions for
  the app to merge, a per-app key (`AppKey`), `Watch`. Served by `cloudsyncd` on its own
  socket (`connector.json`) and on Android by the bound service `ConnectorService`
  (AIDL). New journal operation `app`; older devices carry it without reading it.
- **Password store in the Connector API** (`docs/CONNECTOR.md` § 12): a pairing scope
  `passwords` (asked in `PairingRequest.scopes`, shown plainly in the pairing dialog and
  the Android notification, grantable as "records only" or "all", revocable without
  unpairing → `Change{SCOPES}`). With it an app (the Enclave browser) lists, reads and
  writes the website logins of the safe (`ListLogins`, `GetLogins`, `PutLogin` /
  `DeleteLogin` with compare-and-swap, `ListChanges` with a per-device cursor,
  `Change{LOGIN}`); `TouchLogin` keeps use counters on the device only. Browser fields
  (form fields, `extra` ≤ 16 KiB) stay when the entry is edited in Shard; "never
  save" sites are a hidden entry kind. Vault → Connected apps shows which apps have the
  scope and when they last read passwords. `cs apps approve ID passwords`, `cs apps scopes`.
- **SSH keys and servers in the Connector API** (`docs/CONNECTOR.md` § 13–14), for
  Postern and Cellar. Scope `ssh_keys`: `ListSshKeys` (no secrets) and
  `GetSshKeys` (private key + passphrase, logged as a secret read), `Change{SSH_KEY}`;
  read only. Scope `servers`: one server list shared by every app (`ListServers`,
  `GetServers`, `PutServer` / `DeleteServer` with compare-and-swap, `ListServerChanges`
  with its own per-device cursor, `Change{SERVER}`; ≤ 10 000 servers, `extra` ≤ 16 KiB
  kept for the apps). Servers are a new safe entry kind (`SECRET_SERVER`): the app
  lists, shows and edits them (host, user, login method, SSH key, jump server, folder,
  tags, trusted host keys), and an SSH key's card lists the servers that use it. The
  pairing dialog, the Android notification and `cs apps` name each scope; Vault →
  Connected apps shows when each app last read passwords, keys and server passwords,
  and takes each scope away on its own.

## [0.1.0] - 2026-11-13

First version (not published as a release).

### Added

- **Core (Go).**
  - Password-protected vault (`vault.cs`, Argon2id) with the master key, k/m, storages
    and devices; a copy is kept in every storage and merged across devices.
  - Storages on top of rclone (Google Drive, Mega, S3, WebDAV, Dropbox, OneDrive,
    Yandex Disk, local folders and more), identified by ID: several accounts of one
    type are independent storages; failure domains guide shard placement.
  - Keyed FastCDC chunking with packing of small files, XChaCha20-Poly1305 encryption,
    HMAC object names and Reed-Solomon k+m erasure coding (one shard per storage).
  - Serverless index: per-device encrypted operation journals folded deterministically
    by Lamport clock; concurrent edits produce conflict copies. Journals are packed by
    their owner every 64 segments.
  - Quorum lease locks for files open for writing, with expiry and a 30 s hold after close.
  - Files on demand with a local cache, pins and "keep everything on device";
    sequential-read prefetch, reads from the fastest storages and hedged shard requests.
  - Maintenance: `repair` (scrub with checksums, rewrite missing shards, move data off
    disabled storages, re-encode after a k/m change, optional rebalance) and `gc` with
    a 7-day grace period.
  - Password safe: passwords, SSH keys, web3 keys (Ethereum/EVM, Solana) and TOTP 2FA
    records with recovery codes, in groups with a picture; metadata and secrets are
    sealed with separate keys. Public keys, fingerprints and addresses are derived by
    the core.
- **CLI `cs`:** `init`, `passwd`, `backend add|ls|enable|disable`, `status`, `ls`,
  `put`, `get`, `cat`, `mkdir`, `rm`, `mv`, `pin`, `unpin`, `pins`, `lock`, `sync`,
  `repair`, `redundancy`, `gc`.
- **Daemon `cloudsyncd`:** gRPC API for the app, vault mount (FUSE on Linux, WinFsp on
  Windows), background sync and an on-disk upload queue that survives restarts.
- **App (Flutter) for Linux, Windows and Android:** unlock and first setup, files,
  storages (add wizard with browser sign-in for Drive, OneDrive, Dropbox), maintenance,
  vault settings, password safe with 2FA codes, event log; English and Russian UI;
  desktop tray; fingerprint unlock on Android.
- **Android:** in-process core (gomobile AAR), foreground sync service and a
  DocumentsProvider that exposes the vault in the system file picker with streamed reads.

[Unreleased]: https://github.com/SelfHold/shard/commits/main
