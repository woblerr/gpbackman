# End-to-end tests

The e2e suite uses two Docker Compose fixtures.

- `docker-compose.yml` runs the default single-node Greenplum fixture.
  It is used by `backup-info`, `report-info`, and `history-migrate`.
- `docker-compose.standby.yml` runs a standby-aware Greenplum fixture.
  It is used by `backup-delete`, `backup-clean`, `history-clean`, and `history-sync`.

Both fixtures include [minio/minio](https://hub.docker.com/r/minio/minio) and [minio/mc](https://hub.docker.com/r/minio/mc) containers for S3-compatible backup storage.
The `gpbackman-export` container copies the built `gpbackman` binary to the shared `gpbackman_bin` volume.

The Greenplum fixtures use the [docker-greenplum image](https://github.com/woblerr/docker-greenplum).
The standby-aware fixture uses its standby startup support:

- `master` is the primary coordinator service with `GREENPLUM_DEPLOYMENT=master` and `GREENPLUM_STANDBY_HOSTNAME=standby`;
- `standby` uses `GREENPLUM_DEPLOYMENT=standby` and `GREENPLUM_MASTER_HOSTNAME=master`;
- `segment1` and `segment2` use `GREENPLUM_DEPLOYMENT=segment`;
- `conf/gpinitsystem_config_no_mirrors`, `conf/hostfile_gpinitsystem`, and `conf/ssh/` provide the test cluster config and SSH fixtures.

The SSH fixtures are static e2e-only test material copied from the public docker-greenplum example key pattern.
Do not reuse them outside this disposable test environment.

## Running tests

Build the gpbackman image:

```bash
make docker
```

Run all tests sequentially:

```bash
make test-e2e
```

Run tests for a single command:

```bash
make test-e2e_backup-info
make test-e2e_report-info
make test-e2e_backup-delete
make test-e2e_backup-clean
make test-e2e_history-clean
make test-e2e_history-migrate
make test-e2e_history-sync
```

`backup-delete`, `backup-clean`, `history-clean`, and `history-sync` each start a fresh standby-aware cluster, prepare backups, run the full suite for that command, and remove disposable volumes.

Manually run a single-node command suite:

```bash
docker compose -f e2e_tests/docker-compose.yml up -d
docker exec greenplum bash -c 'su - gpadmin -c "/home/gpadmin/run_tests/run_test.sh backup-info"'
docker compose -f e2e_tests/docker-compose.yml down -v
```

Manually run a standby-aware command suite:

```bash
docker compose -f e2e_tests/docker-compose.standby.yml up -d
docker exec greenplum bash -c 'su - gpadmin -c "/home/gpadmin/run_tests/run_test.sh backup-delete"'
docker compose -f e2e_tests/docker-compose.standby.yml down -v
```

For the explicit history-sync scenario, replace `backup-delete` with `history-sync` in the command above.

If a manual run fails, recreate the fixture before retrying:

```bash
docker compose -f e2e_tests/docker-compose.standby.yml down -v --remove-orphans
docker compose -f e2e_tests/docker-compose.yml down -v --remove-orphans
```

## Standby-aware checks

The `backup-delete`, `backup-clean`, `history-clean`, and `history-sync` suites run against the cluster history database at `/data/master/gpseg-1/gpbackup_history.db`.
That path is eligible for standby history sync.

The mutating suites assert:

- successful commands print the standby sync success message;
- primary and standby history produce matching `gpbackman backup-info` output after successful sync;
- `--no-history-sync-standby` applies the primary mutation and leaves the selected standby state unchanged;
- a blocked fake `rsync` exceeding the configured timeout keeps the primary command successful, emits the standby sync warning, and leaves the selected standby state unchanged.

The `history-sync` suite first seeds the standby with an explicit sync, then disables automatic sync for a primary `backup-delete` so the histories differ.
It runs `history-sync --auto-load-history-db`, verifies the explicit sync success message, and confirms the primary and standby rows match again.
It also blocks a fake `rsync`, sets a one-second timeout, and verifies that explicit sync fails with a timeout error without waiting for the fake transport to finish.

## Notes

- Tests are executed as `gpadmin` inside the `greenplum` container, which is the container name for the `master` service.
- The runner waits for the primary cluster and, for mutating commands, standby replication state `streaming` or `catchup` before preparing backups.
- Scripts exit with a non-zero code on failure.
- `docker compose down -v` removes disposable e2e volumes.
  Do not run these commands against non-test data.
