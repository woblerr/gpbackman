# AGENTS.md

## Project Overview

`gpbackman` is a Go CLI for managing backups created by `gpbackup` on Greenplum clusters or Greenplum compatible clusters.
It works with the `gpbackup_history.db` SQLite history database, YAML-to-SQLite migration,
backup deletion and cleanup, reporting, and standby coordinator history DB sync.

The main entrypoint is `gpbackman.go`.
CLI behavior is implemented with Cobra under `cmd/`.

## Project Structure

- `cmd/` contains Cobra commands, root flags, command validation, deletion and cleanup flows,
  reporting, wrappers, and standby history sync.
- `gpbckpconfig/` contains gpbackup history structs, SQLite/history helpers,
  file utilities, and Greenplum cluster queries.
- `textmsg/` centralizes user-facing error, info, and warning text.
- `e2e_tests/` contains Docker Compose based Greenplum/MinIO end-to-end tests.
- `docker_files/` contains container entrypoint logic.
- `vendor/` is committed dependency source; do not edit vendored files by hand.

## Build, Test, and Lint

Prefer existing `make` targets whenever the Makefile provides one.
Use direct tool commands only when no matching Make target exists.

- Build the local binary: `make build`.
- Run unit tests: `make test`.
- Build for Linux: `make build-linux`.
- Build for Darwin: `make build-darwin`.
- Build Linux from Darwin: `make build-linux-on-darwin`.
  This target requires Homebrew `musl-cross`.
- Build the Docker image: `make docker`.
- Build the Alpine Docker image: `make docker-alpine`.
- Build release artifact snapshots through Docker/GoReleaser: `make dist`.
- Run lint: `make lint`.
- Format Go files: `make fmt`.

CI uses Go 1.25, vendored modules, and `TZ=Etc/UTC`.
Normal builds use CGO because the project depends on `go-sqlite3`.
The Makefile test target runs:

```bash
TZ="Etc/UTC" go test -mod=vendor -timeout=60s -count 1 ./...
```

For fast iteration on a narrow change, use the full CI-style flags with a focused package and test name, for example:

```bash
TZ="Etc/UTC" go test -mod=vendor -timeout=60s -count 1 ./cmd -run TestName
```

Replace `./cmd` with the package you are changing, such as `./gpbckpconfig` or `./textmsg`.
Use `make test` for normal final unit verification.

## End-to-End Tests

Build the Docker image before running e2e tests:

```bash
make docker
```

Run the full e2e suite:

```bash
make test-e2e
```

Run a single command suite with one of:

```bash
make test-e2e_backup-info
make test-e2e_report-info
make test-e2e_backup-delete
make test-e2e_backup-clean
make test-e2e_history-clean
make test-e2e_history-migrate
make test-e2e_history-sync
```

The `backup-delete`, `backup-clean`, `history-clean`, and `history-sync` suites use the standby-aware fixture.

Stop and remove e2e containers and volumes with:

```bash
make test-e2e-down
```

E2E uses Docker Compose, Greenplum, MinIO, and destructive `docker compose down -v`.
Do not run e2e tests unless the user explicitly requests or approves them.

Keep e2e shell scripts simple and close to the existing style.
Add helper functions only when they are reused or materially improve clarity.
Avoid new bash-specific constructs unless nearby scripts already use them or the need is clear.
Prefer `gpbackman` CLI assertions over direct `gpbackup_history.db` reads in e2e tests.
For standby checks, run `gpbackman` on the standby host.
Do not remove no-op or regression e2e cases without an explicit decision.

## Code Style and Patterns

- Keep CLI behavior in Cobra command files under `cmd/`.
- Keep shared flag names and constants in `cmd/constants.go`.
- Keep user-facing text in `textmsg/` when practical.
- Prefer simple, concrete implementations within the existing package boundaries.
  Do not introduce new abstractions, interfaces, wrappers, or package seams unless they remove real duplication,
  reduce existing complexity, or are needed for a focused test seam.
- Prefer the existing standard-library test style: `testing`, `t.Fatalf`, table tests,
  `t.Helper`, and `t.Cleanup`.
- Use `DATA-DOG/go-sqlmock` for DB-facing unit tests.
- Call `testhelper.SetupTestLogger()` in tests that exercise logging paths.
- Use existing package-level test seams such as `execCommand`, `execOSExit`,
  and standby-sync hooks instead of introducing broad new abstractions.
- Do not introduce `testify` or another test framework just because another project uses it.
- For dependency changes, update `go.mod` and `go.sum`, then regenerate `vendor/` with `go mod vendor`.
  Do not hand-edit vendored code.

## Runtime and Safety Rules

- Deletion, cleanup, and migration commands can remove backup data, mutate `gpbackup_history.db`,
  create a new `gpbackup_history.db`, or rename `gpbackup_history.yaml` to `.migrated`.
  Manual tests must use temp or copied data unless the user explicitly points to real data.
- Preserve standby sync semantics.
  Explicit `history-sync` treats an ineligible source, no up standby, and discovery or transfer failures
  as errors and exits non-zero.
  Automatic sync runs only after successful `backup-delete`, `backup-clean`, or `history-clean` operations
  and remains best-effort.
  Both modes require the resolved history database path to be the cluster history database at
  `<primary coordinator data directory>/gpbackup_history.db`.
  For automatic sync, custom and default working-directory databases are skipped,
  `--no-history-sync-standby` disables the attempt, and no up standby is also a skip.
  Automatic sync failures warn and must not mask the primary command success.
- Standby sync copies only `gpbackup_history.db`.
  It does not sync reports, backup data, or other backup artifacts.
- Keep `README.md`, `COMMANDS.md`, `e2e_tests/README.md`, and relevant e2e scripts synchronized
  when flags, command output, examples, or e2e targets change.

## Generated, Local, and Release Artifacts

- Do not commit ignored local artifacts: `.ralphex/`, `.vscode/`, `dist/`,
  the built `gpbackman` binary, `*.db`, or coverage `*.out` files.
- GoReleaser has `changelog.disable: true`.
  Do not add changelog work unless a release task explicitly asks for it.

## Editing These Notes

Use semantic line breaks in this file.
Keep one sentence per line where practical so future diffs stay small.
