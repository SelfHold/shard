# Contributing to Shard

Thanks for your interest! Bug reports, ideas and pull requests are welcome. Shard is an
early, personal project, so expect the design to move; ask before investing in a large
change.

## Before you start

- For anything bigger than a small fix, open an issue first so we can agree on the
  approach.
- Read the [README](README.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). The
  latter lists the invariants the code relies on (deterministic index fold, storage
  identity by ID, quorum locks, nothing readable on a storage, …); changes that break
  them will not be merged.
- Changes to the Connector API (`proto/connector.proto`, the AIDL files) follow
  [docs/CONNECTOR.md](docs/CONNECTOR.md): v1 only gains fields and methods, and the
  appendices of that file must carry the new contract byte for byte.
- Security problems go through [SECURITY.md](SECURITY.md), not public issues.

## Development

Build requirements are listed in the README. Everything goes through `make`; outputs
land in `builds/`.

```bash
make vet test      # Go core (test runs are capped at 1 GB RAM, one at a time)
make race          # Go core with the race detector
make app-test      # flutter analyze + flutter test
make proto         # after editing proto/cloudsync.proto or proto/connector.proto
make build-linux   # desktop app in builds/linux
make build-apk     # Android APKs in builds/android
```

Generated code (`core/rpc/pb`, `core/connector/pb`, `app/lib/data/rpc/gen`) is never
edited by hand: change the `.proto` file and run `make proto`.

Engine tests simulate several devices sharing in-memory storages; real rclone code
paths are covered with the `local` backend.

## Rules

- **English only** in code, identifiers, comments, logs, error messages, commit
  messages and documentation. Other languages appear only in translations: UI strings,
  Android `values-<lang>/` resources and translated docs such as `README.ru.md` (a
  translation must say the same as its English original; update both together).
- App UI strings go through `tr('English', 'Русский')` (`app/lib/core/i18n.dart`). The
  app starts in English regardless of the system locale; never pick the language from
  the platform locale.
- Use the widgets from `app/lib/presentation/common` and the `HmColors` / `HmText`
  tokens: dark-only palette, no raw colours, no Material elevation. Every screen must
  also work in the phone layout (below 768 px).
- Targets are Linux and Android; macOS and iOS are planned, so keep platform specifics
  behind the existing seams (`mobilebind`, the data-dir lookup, the Kotlin glue).
  Windows is not maintained: do not extend or fix the Windows targets.
- Never rename persisted or published identifiers (the `cloudsync` Go module, the
  Android package, AIDL names, app ids, data dir and file names): they live on users'
  devices and in other apps.
- Nothing secret in logs, errors or RPC responses: no tokens, passwords or keys.
- Run `gofmt` and `dart format`; keep `make vet` and `flutter analyze` clean.
- Add tests for behaviour changes, especially in `core/index`, `core/lock` and
  `core/engine`.
- Add a line to the `[Unreleased]` section of [CHANGELOG.md](CHANGELOG.md).

## License

By contributing you agree that your contributions are licensed under the
[GNU GPL v3.0](LICENSE).
