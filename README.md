# gpBackMan

[![Actions Status](https://github.com/woblerr/gpbackman/workflows/build/badge.svg)](https://github.com/woblerr/gpbackman/actions)
[![Coverage Status](https://coveralls.io/repos/github/woblerr/gpbackman/badge.svg?branch=master)](https://coveralls.io/github/woblerr/gpbackman?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/woblerr/gpbackman)](https://goreportcard.com/report/github.com/woblerr/gpbackman)

**gpBackMan** is designed to manage backups created by [gpbackup](https://github.com/greenplum-db/gpbackup) on [Greenplum clusters](https://greenplum.org/).

The utility works with `gpbackup_history.db` SQLite history database format. 

The utility provides functionality for migrating data from the old `gpbackup_history.yaml` YAML format to the new one. If you are using an old `gpbackup` version that supports only YAML format, then use `gpBackMan <= v0.6.0`.

**gpBackMan** provides the following features:
* display information about backups;
* display the backup report for existing backups;
* delete existing backups from local storage or using storage plugins (for example, [S3 Storage Plugin](https://github.com/greenplum-db/gpbackup-s3-plugin));
* delete all existing backups from local storage or using storage plugins older than the specified time condition;
* clean deleted backups from the history database;
* migrate history database from `gpbackup_history.yaml` format to `gpbackup_history.db` SQLite format;
* manually sync the cluster `gpbackup_history.db` to the standby coordinator;
* automatically sync the cluster `gpbackup_history.db` after successful backup deletion and history cleanup.

## Commands
### Introduction

Available commands and global options:

```bash
./gpbackman --help
gpBackMan - utility for managing backups created by gpbackup

Usage:
  gpbackman [command]

Available Commands:
  backup-clean    Delete all existing backups older than the specified time condition
  backup-delete   Delete a specific existing backup
  backup-info     Display information about backups
  completion      Generate the autocompletion script for the specified shell
  help            Help about any command
  history-clean   Clean deleted backups from the history database
  history-migrate Migrate history database
  history-sync    Synchronize the cluster history database to the standby coordinator
  report-info     Display the report for a specific backup

Flags:
  -h, --help                       help for gpbackman
      --auto-load-history-db       resolve gpbackup_history.db from $MASTER_DATA_DIRECTORY or $COORDINATOR_DATA_DIRECTORY when --history-db is unset
      --history-db string          full path to the gpbackup_history.db file
      --log-file string            full path to log file directory, if not specified, the log file will be created in the $HOME/gpAdminLogs directory
      --log-level-console string   level for console logging (error, info, debug, verbose) (default "info")
      --log-level-file string      level for file logging (error, info, debug, verbose) (default "info")
  -v, --version                    version for gpbackman

Use "gpbackman [command] --help" for more information about a command.
```

### Standby history DB sync

Run `history-sync` to explicitly synchronize the cluster `gpbackup_history.db` to an up standby coordinator. The source must resolve to `<primary coordinator data directory>/gpbackup_history.db`; a custom database or the default working-directory database is not eligible. Explicit sync treats every non-sync outcome as an error and exits non-zero, including no up standby coordinator, an ineligible source, discovery errors, and transfer errors.

For the usual cluster setup, prefer resolving the source from the coordinator data directory:

```bash
./gpbackman history-sync --auto-load-history-db
```

After a successful `backup-delete`, `backup-clean`, or `history-clean`, gpBackMan also attempts the same synchronization automatically. Automatic sync is best-effort: ineligible source paths and no standby are debug-only skips, while sync failures are warnings and do not change the successful primary command result. Pass `--no-history-sync-standby` to those mutation commands to disable their automatic sync.

Configure the synchronization timeout with `--history-sync-standby-timeout SECONDS` on `history-sync`, `backup-delete`, `backup-clean`, and `history-clean`. The default is 300 seconds, and the supported range is 1 to 86400 seconds. The timeout is one shared budget for `rsync` and remote install; it starts after SQLite snapshot creation and validation. Standby discovery, `VACUUM INTO`, and `PRAGMA quick_check` are outside this budget. Remote cleanup after a transport failure uses a separate fixed timeout of 120 seconds, independent of `--history-sync-standby-timeout`.

Only `gpbackup_history.db` is synced. Report files, backup data, and any other backup artifacts are not synced by gpBackMan.

### Detail info about commands

Description of each command:
* [Delete all existing backups older than the specified time condition (`backup-clean`)](./COMMANDS.md#delete-all-existing-backups-older-than-the-specified-time-condition-backup-clean)
* [Delete a specific existing backup (`backup-delete`)](./COMMANDS.md#delete-a-specific-existing-backup-backup-delete)
* [Display information about backups (`backup-info`)](./COMMANDS.md#display-information-about-backups-backup-info)
* [Clean deleted backups from the history database (`history-clean`)](./COMMANDS.md#clean-deleted-backups-from-the-history-database-history-clean)
* [Migrate history database (`history-migrate`)](./COMMANDS.md#migrate-history-database-history-migrate)
* [Synchronize the cluster history database to the standby coordinator (`history-sync`)](./COMMANDS.md#synchronize-the-cluster-history-database-to-the-standby-coordinator-history-sync)
* [Display the report for a specific backup (`report-info`)](./COMMANDS.md#display-the-report-for-a-specific-backup-report-info)

## Getting Started
### Building and running

```bash
git clone https://github.com/woblerr/gpbackman.git
cd gpbackman
make build
./gpbackman <flags>
```

### Running as docker container

Environment variables supported by this image:
* `TZ` - container's time zone, default `Etc/UTC`;
* `GPBACKMAN_USER` - non-root user name for execution of the command, default `gpbackman`;
* `GPBACKMAN_GROUP` - non-root user group name for execution of the command, default `gpbackman`;
* `GPBACKMAN_UID` - UID of internal `${GPBACKMAN_USER}` user, default `1001`;
* `GPBACKMAN_GID` - GID of internal `${GPBACKMAN_USER}` user, default `1001`.

#### Build container

```bash
make docker
```

or manual:

```bash
docker build  -f Dockerfile  -t gpbackman .
```

For Alpine image:

```bash
make docker-alpine
```
or manual:

```bash
docker build  -f Dockerfile.alpine  -t gpbackman-alpine .
```

#### Run container

```bash
docker run \
  --name gpbackman \
  -v /data/master/gpseg-1/gpbackup_history.db:/data/master/gpseg-1/gpbackup_history.db \
  gpbackman \
  gpbackman backup-info \
  --history-db /data/master/gpseg-1/gpbackup_history.db
```

### Running tests

Run the unit tests:

```bash
make test
```

Run the end-to-end tests:

```bash
make test-e2e
```
