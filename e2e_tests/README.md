# End-to-end tests

The following architecture is used to run the tests:

* Separate containers for MinIO and nginx. Official images [minio/minio](https://hub.docker.com/r/minio/minio), [minio/mc](https://hub.docker.com/r/minio/mc) and [nginx](https://hub.docker.com/_/nginx) are used. It's necessary for S3 compatible storage for WAL archiving and backups.
- Separate container gpbackman-export: runs the gpbackman image and copies the binary to a shared Docker volume (gpbackman_bin) for use inside the Greenplum container.
* Separate container for Greenplum. The [docker-greenplum image](https://github.com/woblerr/docker-greenplum) is used to run a single-node Greenplum cluster.

## Running tests

Buld gpbackman image:
```bash
make docker
```

Run all tests (sequentially for all commands):

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
```

Manually run a specific test (example for `backup-info`):

```bash
docker compose -f e2e_tests/docker-compose.yml up -d

docker exec greenplum bash -c 'su - gpadmin -c "/home/gpadmin/run_tests/run_test.sh backup-info"'

docker compose -f e2e_tests/docker-compose.yml down -v
```

If during manual execution the test fails, you should recreate containers.

## Notes
- Tests are executed as `gpadmin` inside the Greenplum container. The runner waits for the cluster to become ready and then prepares the backup set before executing checks.
- Scripts exit with a non-zero code on failure.
