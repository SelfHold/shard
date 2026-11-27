# Architecture notes and invariants

The [README](../README.md) explains how Shard works. This file lists the properties
the code relies on. A change that breaks one of them will not be merged, even if the
tests pass: several of them only show up with many devices, offline periods or slow
storages.

## Parts

- `core/`: the Go core, with all logic and all secrets. `proto/cloudsync.proto` is the
  gRPC API between the core and the app; `proto/connector.proto` is the published API
  for other apps ([CONNECTOR.md](CONNECTOR.md)).
- `cmd/cs` (CLI) and `cmd/cloudsyncd` (desktop daemon: gRPC for the app, FUSE mount,
  background sync, Connector API).
- `mobilebind`: the core for Android (gomobile AAR), used by the Kotlin side
  (`Core`, `CoreService`, `VaultProvider`, `ConnectorService`).
- `app/`: the Flutter UI. It only renders and never holds storage credentials or keys.
  Desktops talk to `cloudsyncd` (`endpoint.json` + bearer token); Android runs the core
  in-process.

## Storage and index

- A storage is identified by `vault.Backend.ID`, never by its type. Placement, journals
  and leases all key on the ID; several backends may share a type and a domain.
- The index tree is a pure function of the set of operations (fold in
  `(lamport, device, seq)` order). Anything the fold depends on must be inside the
  operation (e.g. `Op.DeviceName` for conflict names); never look things up from local
  state.
- A journal segment name is never reused with different content: `Journal.Append`
  burns the sequence number even when the write fails.
- One process per device data dir (flock in `engine.Open`). The lock manager relies on
  it to delete this device's stale leases.
- Nothing readable may reach a storage: object names are HMACs, payloads are AEAD.
- Storage calls go through `backend.Measured` (per-storage concurrency limit, and the
  response times used to read from the fastest storages). Loops over storages run in
  parallel (`backend.Each` / `backend.Limit`). Use `backend.PutNew` only for names
  never used before (segments, packs, leases): on Drive and Mega it skips the lookup
  and could duplicate a name.
- Shard names include the blob's generation (`Ref.Gen`): re-encoding with another k/m
  writes new names and records the new ref with `OpPlace.Refs`. Never overwrite shards
  of a live ref in place.
- GC deletes only objects this device has seen unused for the whole grace period
  (marks in `<data>/gc.json`). Live = blobs of the current tree + pending operations.
  Do not shorten the default grace: it covers in-flight uploads and offline devices.
- Only the owner compacts its journal (`j/<dev>/p<from>-<to>` packs); readers skip the
  segments a pack covers. A pack is written only from segments that were all readable.
- Vault merge: the newer version wins but keeps the local devices; equal versions with
  different content are merged deterministically (the larger hash is the base) and
  bumped.
- One account (or local folder, including nested ones) holds only one storage:
  `backend.SameLocation`, checked in `AddBackend` and `cs backend add`;
  `Backend.Account` is recorded when the storage is added.
- `local` storages need an absolute root; unknown rclone options are rejected
  (`backend.CheckRoot`, `ObscureParams`).
- The service may be unlocked but not ready (vault open, engine not: fewer than k+m
  storages). `AddBackend` and `SetRedundancy` work in that state; reloads keep the vault
  open.

## Locks and uploads

- A device wins a storage only if no other live lease is there after its own write; it
  holds the lock with a majority. Do not go back to "earliest timestamp wins": it lets
  two devices hold a lock. Each storage is "write, then list" on its own; the attempt is
  decided once a majority is won (or lost with a quorum answering).
- A lease is removed only after its pending writes (`Lock.pending`), and a renewal
  never rewrites a released lease. Leases are flat: `l/<path hash>.<device>-<nonce>`.
- Never lose an edit: the mount writes the queue record (staging dir) before the
  background upload starts and removes it only after the commit; failed or interrupted
  uploads are retried (`mount.FS.RetryQueued`, which skips `FS.inflight`).

## Password safe

- Records are `index.OpSecret` operations, folded like files (a stale parent gives a
  conflict copy `<id>~<op id>`; a delete of a changed record is dropped).
- Meta and Data are sealed with separate keys (AD = the `SealedAs` id). Never put the
  plaintext of a record into an operation, a log, a cache or any file. `ListSecrets`
  never returns Data; only `GetSecret` opens it.
- Group pictures are at most 64 KiB (they are in every operation that changes the
  group).
- TOTP entries (`KindTOTP`) keep the secret and the recovery codes in Data; only
  computed codes leave the core (`TotpCodes`). `Prepare` must stay idempotent (a used
  recovery code is removed by re-saving the opened entry).
- The safe lives only in journal operations, never in sharded blobs: every storage holds
  the whole journal (`Journal.Spread` on each sync, `Replicate` in repair; both also
  upload objects only the local copies still have). `engine.Open` loads the
  `<data>/journal` copies before any network call (`LoadCached`); `findSeq` counts the
  device's own local copies, so a wiped storage set never makes a segment name reused.
  Without an engine the RPC reads the safe from `engine.OpenOfflineSafe` (read only; the
  app shows the safe while not ready).

## Connected apps

- A record is an `index.OpApp` whose `Parents` are the heads it replaces; the record's
  value is the set of heads (a multi-value register, a pure function of the
  operations). `AppPut` refuses a stale `base` (`ErrAppConflict` → `ABORTED`);
  concurrent offline writes stay as versions for the app to merge. Never pick one in
  the core.
- Data is sealed with `Derive("connector app data", app)`, AD = app + key. `AppKey`
  (`Derive("connector app key", app)`) is the app's own key and must stay distinct from
  it.
- An app reaches only records of its own app id. Never expose files, storages or vault
  calls through the connector. The scopes `passwords` (logins), `ssh_keys` (read only)
  and `servers` open parts of the safe; servers are `KindServer` entries
  (`engine/servers.go`). API writes keep `extra` and the safe group, and each kind has
  its own change log (`login-changes.json`, `server-changes.json`).
- Pairings are per device (`<data root>/apps.json`, desktop tokens only as SHA-256).
  Android callers are identified by `ConnectorService` from Binder, never from request
  data.
- `proto/connector.proto` and the AIDL files are a published contract: v1 only gains
  fields and methods. [CONNECTOR.md](CONNECTOR.md) carries copies of them, checked by
  `TestDocCarriesContracts`; update the appendices when you change them. Other SelfHold
  apps copy `proto/connector.proto` byte for byte, so a contract change lands here
  first and then in each consumer.

## App

- UI strings go through `tr('English', 'Русский')` (`app/lib/core/i18n.dart`). The UI
  starts in English; Russian is chosen in the app's settings (per device, never synced,
  `AppSettings` → `app.json`). Android-side strings (notifications, the file picker
  root) follow the same choice through `AppLanguage.kt`, never the system locale.
- Screens use only the widgets of `app/lib/presentation/common` and the tokens of
  `HmColors` / `HmText`: dark-only palette, no raw colours, no Material elevation,
  bundled fonts. Below 768 px wide the app uses the phone layout on any platform.
- The Dart gRPC code in `app/lib/data/rpc/gen` is generated (`make proto`); import it
  `as pb` (`State` collides with Flutter).
- Check layouts with `app/tool/screenshots.dart` against a running daemon (see the
  README).

## Tests

Engine tests simulate devices sharing `backend.MemStore`s (with `SetOffline` / `Wipe`).
Real rclone code paths are covered with the `local` backend. The Linux FUSE mount needs
`libfuse.so.2` (default) or `-tags fuse3` with `libfuse3.so.3`.
