# Shard

**English** · [Русский](README.ru.md)

[![CI](https://github.com/SelfHold/shard/actions/workflows/ci.yml/badge.svg)](https://github.com/SelfHold/shard/actions/workflows/ci.yml)
[![License: GPL v3](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

File sync between Linux and Android devices with no server in the middle. Your data
is spread over several cloud storages you already have (Google Drive, Mega, S3, WebDAV,
Dropbox, OneDrive, Yandex Disk and anything else rclone supports, including several
accounts of the same type) with Reed-Solomon k+m redundancy (m=1 works like RAID 5).
Everything is encrypted and decrypted on your devices only. Shard also keeps a
password safe, and other [SelfHold](https://github.com/SelfHold) apps sync their
settings through it.

Shard was formerly called CloudSync; some identifiers still use the old name (the Go
module `cloudsync`, the binaries `cloudsyncd`, the vault file `vault.cs`, the config
directory `cloudsync/`, the Android package `dev.cloudsync.cloudsync`).

**Status:** an early, personal project. There are no published releases yet, and
before 1.0 the on-storage format and the APIs may change. Build it from source (see
below).

<p align="center">
  <img src="app/shots/unlock-desktop.png" alt="Desktop app" height="320">
  &nbsp;
  <img src="app/shots/unlock-phone.png" alt="Android app" height="320">
</p>

## Features

- **No server.** Devices talk only to the storages. The file tree is a deterministic
  fold of every device's operation journal.
- **Encrypted end to end.** Storages see only objects with random names: no file
  names, no sizes of single files, no content.
- **Survives a lost storage.** Each block is split into k data and m parity shards, one
  per storage. Any m storages can disappear; `repair` rebuilds what is missing.
- **Files on demand.** The full listing is always there; content is downloaded on read,
  only the chunks you need. Pin files or folders to keep them offline.
- **Mounted folder** on Linux (FUSE); system file picker on Android.
- **Write locks** across devices, so a file open on one device is read-only on others.
  Concurrent offline edits become conflict copies, never lost edits.
- **Password safe** with passwords, SSH keys, web3 keys (Ethereum/EVM, Solana),
  servers and 2FA (TOTP) codes with recovery codes, synced the same way as files.
  Imports from KeePass, Bitwarden, browsers and other password managers.
- **Connected apps.** Other apps (such as Parley) can keep their settings in the
  vault: the same on every device, in every storage, without a password of their own.

## Platforms

- **Linux** (desktop app, daemon, CLI) and **Android** (app with the core inside) are
  the targets.
- **macOS and iOS** are planned; the platform specifics are kept behind small seams
  (`mobilebind`, the data-dir lookup, the Kotlin glue) so they can be added later.
- **Windows is not a target.** Some Windows code paths and build targets remain in the
  tree, but Windows builds are not maintained or tested.

## Security model in brief

- One password opens the vault (`vault.cs`, Argon2id). It holds the master key; every
  other key (content, object names, journals, the safe, connected apps) is derived
  from it.
- Storages only ever see ciphertext under HMAC names: no file names, no sizes of single
  files, no record fields. Payloads are sealed with XChaCha20-Poly1305.
- On the device, the safe and the journals stay sealed on disk too; logs and RPC
  errors carry no tokens, passwords or keys. File content you open or pin is kept
  decrypted in the device's cache (as with any synced folder), so protect the device
  itself with disk encryption.
- Keys and storage credentials live only in the Go core; the Flutter UI gets a
  secret only when the user opens it. On desktops the app talks to the local daemon
  over a user-only socket with a bearer token; on Android the core runs inside the app.
- Connected apps reach only their own records, plus the parts of the safe the user
  grants them explicitly (scopes), per device and revocable.

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities and
[Current limitations](#current-limitations) for what the model does not cover.

## How it works

```
file → FastCDC chunks (~4 MB; small files are packed together)
     → XChaCha20-Poly1305 → Reed-Solomon k+m → one shard per storage
```

- **Vault** (`vault.cs`) is the only configuration file: master key, k/m, storages with
  their credentials, devices. It is encrypted with a key derived from your password
  (Argon2id). A copy lives in every storage, so changes (such as a new storage) reach
  the other devices.
- **A storage is identified by its `id`, not by its type**: two Mega accounts are two
  independent storages. The `domain` field (the type by default) is a failure zone:
  shards of one block are spread across different providers, and `status` warns you
  when losing a single provider would make files unreadable.
- **Index without a server.** Each device writes only its own operation journal
  (`j/<device>/<seq>`, encrypted, a copy in every storage). The file tree is a
  deterministic fold of all operations (Lamport clocks). Concurrent edits of one file
  give a conflict copy, not data loss.
- **Files on demand.** Pins (`cs pin`) keep files or folders on the device; `pin /`
  keeps a full local copy — a *reference device* (a switch in Maintenance). The app
  pins in the background (progress in `Status`); `cs pin` and `cs sync` wait for it.
- **Locks.** Opening a file for writing puts a lease into every storage. A device owns
  the file when a majority of storages hold no other lease besides its own. While a file
  is open on one device, other devices can open it read-only. A lease expires unless
  renewed (2 min), so a vanished device does not block the file. The decision is made as
  soon as a majority is reached, so a slow storage does not delay opening. After a file
  is closed the lease is kept for 30 s more, so saving again does not take the lock anew.
- **Speed.** All storages are queried in parallel, at most 6 requests at a time per
  storage. A journal write returns once copies are on m+1 storages; the rest are written
  in the background. File blocks are uploaded 4 at a time. Reads take shards from the
  storages that answered faster (response times are shown on the storages screen) and
  ask for one more shard when a storage answers much slower than usual. Sequential reads
  prefetch the next 2 blocks.
- **Maintenance.** `repair` reads every shard (verifying checksums), writes missing ones,
  moves them off disabled storages, re-encodes blocks after a k/m change (a new
  "generation" of shards with new names) and can rebalance data onto new storages. `gc`
  deletes objects that no file has needed for more than 7 days (each device keeps its own
  count, no coordination needed). Each device packs its own journal: every 64 segments
  into one object.
- **Password safe.** Passwords, SSH keys and web3 keys (Ethereum/EVM, Solana), in groups
  with a picture and a name. A record is a `secret` operation in the same device
  journals, so it syncs like the file tree (concurrent edits give a conflict copy of the
  record). Inside the operation two parts are sealed with separate keys derived from the
  master key: metadata (title, login, URL, public key, address) is opened for the list,
  secrets (password, notes, private key, key passphrase) only when the record card is
  open. Only ciphertext reaches the disk (journals, their local copies, `state.json`).
  The core derives the public SSH key, its fingerprint and the web3 address from the
  private key itself.
- **2FA codes (TOTP).** A `totp` record in the same safe: the secret (base32 or an
  `otpauth://totp/…` link), algorithm, digits, period and recovery codes are in the
  sealed part. The core computes the current codes (`TotpCodes`); the secret never goes
  into the list. Recovery codes are pasted as is (numbering, columns, headers and notes
  of the file are dropped); a used code is removed from the record (`UseRecoveryCode`) on
  every device. "Import from link" takes `otpauth://totp/…` links (several at once) and
  Google Authenticator exports (`otpauth-migration://`, every part of the export; the core
  expands them in `ReadTotpLinks`); HOTP is not supported.
- **Import from other password managers** ("Import" in the safe, `ImportSecrets`): a
  KeePass database (`.kdbx` 3/4 with password and optional key file; the recycle bin is
  skipped), KeePass XML, Bitwarden unencrypted JSON, and CSV exports of KeePass,
  KeePassXC, Chrome / Edge / Brave / Opera, Firefox, Safari / Apple Passwords, Bitwarden,
  1Password, LastPass, Dashlane, Proton Pass, NordPass or any CSV with name / URL and
  password columns. The core reads the file (nothing is written for the preview), keeps
  folders as groups (or puts everything in one group), moves extra fields into the notes,
  makes a separate 2FA record from a TOTP field, and skips entries the safe already has,
  so importing a file twice adds nothing. All records go in one commit.

- **Connected apps** ([docs/CONNECTOR.md](docs/CONNECTOR.md), `proto/connector.proto`).
  An app pairs with Shard once per device: it shows a 6-digit code, the user sees
  the same code in Shard ("Connect an app?", Vault → Connected apps, or
  `cs apps approve`) and allows it. Then the app keeps small records (≤ 256 KiB, e.g.
  its settings file) in the vault. A record is an `app` operation in the device
  journals, sealed with a key of the app. Writes carry the revision they were made
  from, and a stale one is refused. Writes made concurrently on offline devices
  are all kept as versions of the record until the app merges them, so nothing
  is lost. The app also gets a key of its own (`AppKey`, the same on every device),
  so it can protect its local copy without a password of its own. A paired app sees
  only its own records: not files, passwords or storages, unless the user grants it
  a scope: `passwords` (a browser: the website logins of the safe), `ssh_keys` (read
  only) or `servers` (one server list shared by the SSH and database apps); see
  § 12–14 of CONNECTOR.md. Vault → Connected apps shows each scope and takes it away
  on its own. Desktops serve the API on a separate socket found through
  `connector.json`; on Android it is the bound service `ConnectorService` (the caller
  is identified by its package and signing certificate).

## Repository layout

| Path | What |
|---|---|
| `core/crypto` | keys, HKDF, AEAD, HMAC names |
| `core/vault` | password-protected configuration file |
| `core/backend` | storages on top of rclone (+ in-memory for tests) |
| `core/chunker` | keyed FastCDC |
| `core/erasure` | Reed-Solomon and shard placement across domains |
| `core/blob` | encryption + RS + write/read/repair of blocks |
| `core/index` | operations, fold, journals |
| `core/lock` | quorum lease locks |
| `core/secrets` | password safe: record sealing, SSH and web3 public key derivation, TOTP, recovery code parsing |
| `core/engine` | all of the above together: files, cache, pins, sync, repair |
| `core/mount` | FUSE mount (Linux) |
| `core/rpc` | gRPC service for the app (`proto/cloudsync.proto`) |
| `core/connector` | Connector API for other apps (`proto/connector.proto`, [docs/CONNECTOR.md](docs/CONNECTOR.md)): pairing, app records |
| `cmd/cs` | CLI |
| `cmd/cloudsyncd` | desktop daemon: gRPC + mount + background sync |
| `mobilebind` | core for Android (`gomobile bind` → AAR) |
| `app/` | Flutter app for Linux and Android |
| `app/lib/presentation/common` | the app's widget kit: design tokens, dark-only palette, phone layout below 768 px |

## Build from source

There are no prebuilt releases yet. Every build output lands in `builds/`.

Requirements:

- **Go** (the version in `go.mod`) and a C compiler; FUSE headers and `libfuse.so.2`
  (package `fuse2` / `libfuse2`) for the Linux mount, or `libfuse3.so.3` with
  `FUSE_TAGS=-tags=fuse3`.
- **Flutter** (stable; CI uses the version in `.github/workflows/ci.yml`) for the app.
  On Linux also the GTK 3 development files, `clang`, `cmake`, `ninja`, `pkg-config`
  and `libayatana-appindicator3` (tray icon).
- **Android:** the Android SDK and NDK (found in `~/Android/Sdk/ndk`, or pass `NDK=`),
  JDK 17, and `gomobile` / `gobind` (`go install golang.org/x/mobile/cmd/gomobile
  golang.org/x/mobile/cmd/gobind && gomobile init`).
- `protoc` with `protoc-gen-go`, `protoc-gen-go-grpc` and the Dart plugin
  (`dart pub global activate protoc_plugin`), only to regenerate code (`make proto`).

```bash
make build         # builds/cs, builds/cloudsyncd (Linux, FUSE 2)
make build FUSE_TAGS=-tags=fuse3   # if libfuse3.so.3 is available
make build-linux   # desktop app in builds/linux (cloudsyncd next to it)
make aar           # the Go core for Android (gomobile AAR)
make build-apk     # Android APKs (arm64, x86_64) in builds/android; builds the AAR first
make app-run       # build cloudsyncd and run the app in development mode
make proto         # regenerate Go and Dart code from proto/
make vet test      # go vet + Go core tests (tests capped at 1 GB RAM)
make race          # the same with the race detector
make app-test      # flutter analyze + flutter test
```

A release APK is signed with the key given in `app/android/key.properties` or in the
`CS_KEYSTORE`, `CS_KEYSTORE_PASSWORD`, `CS_KEY_ALIAS`, `CS_KEY_PASSWORD` environment
variables; without them it is signed with the debug key.

## Quick start (CLI)

```bash
export CS_PASSWORD=...          # or you will be prompted
cs init -k 2 -m 1               # 2 data + 1 parity shard: needs ≥3 storages
cs backend add -name "Mega personal" -type mega user=me@x.com pass=...
cs backend add -name "Mega work" -type mega user=work@x.com pass=...
cs backend add -name "S3" -type s3 provider=AWS access_key_id=... secret_access_key=... region=eu-central-1 bucket_acl=private
cs backend add -name "MinIO" -type s3 -domain minio-home -root cloudsync \
  provider=Minio endpoint=http://nas.local:9000 access_key_id=... secret_access_key=...
cs backend add -name "Drive" -type drive token='{...}'   # token from `rclone authorize drive`
cs status
cs put ~/Documents /
cs ls -l /Documents
cs pin /Documents               # keep it on this device
cs get /Documents/report.pdf .
cs repair                       # write missing shards, move them off disabled storages
```

Maintenance:

```bash
cs repair [-rebalance]          # verify everything, repair; -rebalance spreads data onto new storages
cs redundancy -k 2 -m 2         # new scheme for new files, then run cs repair
cs gc                           # delete data unused for more than 7 days (-grace 168h)
cs backend reconnect ID         # OAuth token expired or revoked: sign in to the same account again
```

For `-type local` the folder is given with an absolute `-root`. Unknown options are
rejected.

S3-compatible servers (MinIO, Ceph, …): `-root` is `bucket[/folder]`; the bucket is
created on first write if the key may do it. The account of an S3 storage is
`access_key_id@endpoint host`, so two storages on one server need different keys.
Give a self-hosted server its own `-domain`: otherwise it shares the failure zone
`s3` with every other S3 storage.

A storage whose sign-in expired (401, “invalid_grant”) keeps its ID and data: in the
app use “Reconnect” on the Storages screen, or `cs backend reconnect ID`. Signing in
to another account is refused.

## App

One Flutter app on every platform, the same core underneath:

- **Linux.** The app is a client of the `cloudsyncd` daemon. It reads the
  address and token from `<config dir>/cloudsync/endpoint.json`. If the daemon is not
  running, the app starts it: it looks next to its own binary, in `$CLOUDSYNCD` or in
  `PATH`. After unlocking, the daemon mounts the vault at `~/CloudSync` (`-mount`
  changes it). "Open" in the app opens the file in that folder, so edits are saved and
  the file is locked meanwhile. Closing a file does not wait for the upload: the file
  is first written to an on-disk queue, then uploaded in the background and read from
  the local copy until then. If the daemon dies before the upload finishes, it
  continues on the next start.
- **Android.** The core runs inside the app: AAR from `make aar`, a foreground service,
  gRPC over a unix socket in the app's private folder. The vault is visible in the system
  file picker (DocumentsProvider). Reads are streamed, only the needed parts are
  downloaded. A write takes the vault lock and uploads the file when the other app
  closes it.
- **Screens:** unlock and first setup (new vault or a file from another device);
  passwords (groups, record card, password generator, copy with clipboard clearing after
  30 s; a "2FA codes" mode with live codes and recovery codes); files (upload, open,
  "keep on device", rename, delete, who holds a file open, conflict copies and resolving
  them, search); storages (state, add wizard with browser sign-in for Drive, OneDrive and
  Dropbox, disable, remove); maintenance (sync, "keep everything on device", verify and
  repair, k/m change, cleanup, devices); vault (change password, export the vault file
  for a new device, lock).
- **Language:** the UI starts in English whatever the system locale is; Russian can be
  chosen on the Vault screen. The choice is kept per device and is never synced.
- **A new vault** opens right away but works only with k+m storages. Until then the app
  shows the storages screen.
- **Browser sign-in** (OAuth): the core starts an rclone server on `127.0.0.1:53682` and
  the app opens the link. The browser must be on the same device.

A new device needs the `vault.cs` file (export it on the Vault screen) and the password.

Screenshots of every screen at 1440 and 390 px against a running daemon:
`SHOTS=/tmp/shots flutter run -d linux -t tool/screenshots.dart` (in `app/`).

## Current limitations

- Deleted and old versions of safe records stay in the journal (encrypted) until the
  journal is folded into a snapshot. Decrypted passwords live in the memory of the core
  and the app while the card is open; Go and Dart cannot guarantee wiping strings, and
  memory may end up in swap.
- Locks are "almost always": an offline device does not see them; such edits become
  conflict copies.
- Old file versions are kept only until GC (7 days): there is no version history in
  the UI.
- The journal is packed but not folded into a tree snapshot: the operation history
  grows (kilobytes per thousands of edits).
- Browser sign-in works only where the browser is on the same device. In the CLI,
  OAuth storages need a ready `token` from `rclone authorize`.
- Android has no mounted folder: files are visible through the system file picker
  (DocumentsProvider) and in the app itself.

## SelfHold apps and the Connector

Shard is part of [SelfHold](https://github.com/SelfHold), a set of independent,
self-hosted apps. Each app works on its own; Shard is the optional sync layer they
share through the **Connector API** ([docs/CONNECTOR.md](docs/CONNECTOR.md), the
canonical published contract):

- [Parley](https://github.com/SelfHold/parley) (formerly MegaChat): chat and mail
  client; keeps its settings vault in Shard.
- [Postern](https://github.com/SelfHold/postern) (formerly SSH Manager) and
  [Cellar](https://github.com/SelfHold/cellar) (formerly DB Manager): SSH and database
  tools; share one server list and the safe's SSH keys through Shard.
- [Enclave](https://github.com/SelfHold/enclave): a personal Chromium build; reads and
  saves website logins in Shard's safe.
- [Bezel](https://github.com/SelfHold/bezel): the matching GTK / GNOME Shell theme.

The repositories share no code: each consumer carries a byte-for-byte copy of
`proto/connector.proto`, whose owner is this repository. Version 1 only gains fields
and methods, so apps keep working against an older Shard.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and the invariants in
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). Security issues: [SECURITY.md](SECURITY.md).
Changes are listed in [CHANGELOG.md](CHANGELOG.md).

## License

Shard is licensed under the [GNU General Public License v3.0](LICENSE). The fonts
in `app/assets/fonts` are under the SIL Open Font License (`OFL-*.txt` next to them).
