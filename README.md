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
* sync the cluster `gpbackup_history.db` to the standby coordinator after successful backup deletion and history cleanup.

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

After a successful `backup-delete`, `backup-clean`, or `history-clean`, gpBackMan syncs the cluster `gpbackup_history.db` to an up standby coordinator when sync conditions are met. Sync failures are logged as warnings and do not change the primary command result.

Sync runs only when the resolved history database path is the cluster history database at `<primary coordinator data directory>/gpbackup_history.db`. Custom history databases and the default working-directory `gpbackup_history.db` path are skipped. The sync is also skipped when no up standby coordinator is found, or when `--no-history-sync-standby` is specified for `backup-delete`, `backup-clean`, or `history-clean`.

Only `gpbackup_history.db` is synced. Report files, backup data, and any other backup artifacts are not synced by gpBackMan.

### Detail info about commands

Description of each command:
* [Delete all existing backups older than the specified time condition (`backup-clean`)](./COMMANDS.md#delete-all-existing-backups-older-than-the-specified-time-condition-backup-clean)
* [Delete a specific existing backup (`backup-delete`)](./COMMANDS.md#delete-a-specific-existing-backup-backup-delete)
* [Display information about backups (`backup-info`)](./COMMANDS.md#display-information-about-backups-backup-info)
* [Clean deleted backups from the history database (`history-clean`)](./COMMANDS.md#clean-deleted-backups-from-the-history-database-history-clean)
* [Migrate history database (`history-migrate`)](./COMMANDS.md#migrate-history-database-history-migrate)
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
