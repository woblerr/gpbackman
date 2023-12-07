# gpBackMan

[![Actions Status](https://github.com/woblerr/gpbackman/workflows/build/badge.svg)](https://github.com/woblerr/gpbackman/actions)
[![Coverage Status](https://coveralls.io/repos/github/woblerr/gpbackman/badge.svg?branch=master)](https://coveralls.io/github/woblerr/gpbackman?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/woblerr/gpbackman)](https://goreportcard.com/report/github.com/woblerr/gpbackman)


**gpBackMan** is designed to manage backups created by [gpbackup](https://github.com/greenplum-db/gpbackup) on [Greenplum clusters](https://greenplum.org/).

The utility works with both history database formats: `gpbackup_history.yaml` file format (before gpbackup `1.29.0`) and  `gpbackup_history.db` SQLite format (starting from gpbackup `1.29.0`).

**gpBackMan** provides the following features:
* display information about backups.
* delete existing backups from local storage or using storage plugins (for example, [S3 Storage Plugin](https://github.com/greenplum-db/gpbackup-s3-plugin));
* migrate history database from `gpbackup_history.yaml` format to `gpbackup_history.db` SQLite format.

## Commands
### Introduction

Available commands and global options:

```bash
./gpbackman --help
gpBackMan - utility for managing backups created by gpbackup

Usage:
  gpbackman [command]

Available Commands:
  backup-delete   Delete a specific backup set
  backup-info     Display a list of backups
  completion      Generate the autocompletion script for the specified shell
  help            Help about any command
  history-migrate Migrate data from gpbackup_history.yaml to gpbackup_history.db SQLite history database

Flags:
  -h, --help                       help for gpbackman
      --history-db string          full path to the gpbackup_history.db file
      --history-file stringArray   full path to the gpbackup_history.yaml file, could be specified multiple times
      --log-file string            full path to log file directory, if not specified, the log file will be created in the $HOME/gpAdminLogs directory
      --log-level-console string   level for console logging (error, info, debug, verbose) (default "info")
      --log-level-file string      level for file logging (error, info, debug, verbose) (default "info")
  -v, --version                    version for gpbackman

Use "gpbackman [command] --help" for more information about a command.
```

### Display information about backups (`backup-info`)

Available options for `backup-info` command and their description:

```bash
./gpbackman backup-info -h
Display a list of backups.

By default, only active backups or backups with deletion status "In progress" from gpbackup_history.db are displayed.

To additional display deleted backups, use the --show-deleted option.
To additional display failed backups, use the --show-failed option.
To display all backups, use --show-deleted  and --show-failed options together.

The gpbackup_history.db file location can be set using the --history-db option.
Can be specified only once. The full path to the file is required.

The gpbackup_history.yaml file location can be set using the --history-file option.
Can be specified multiple times. The full path to the file is required.

If no --history-file or --history-db options are specified, the history database will be searched in the current directory.

Only --history-file or --history-db option can be specified, not both.

Usage:
  gpbackman backup-info [flags]

Flags:
  -h, --help           help for backup-info
      --show-deleted   show deleted backups
      --show-failed    show failed backups

Global Flags:
      --history-db string          full path to the gpbackup_history.db file
      --history-file stringArray   full path to the gpbackup_history.yaml file, could be specified multiple times
      --log-file string            full path to log file directory, if not specified, the log file will be created in the $HOME/gpAdminLogs directory
      --log-level-console string   level for console logging (error, info, debug, verbose) (default "info")
      --log-level-file string      level for file logging (error, info, debug, verbose) (default "info")
```

The following information is provided about each backup:
* `TIMESTAMP` - backup name, timestamp (`YYYYMMDDHHMMSS`) when the backup was taken;
* `DATE`- date in format `Mon Jan 02 2006 15:04:05` when the backup was taken;
* `STATUS`- backup status: `Success` or `Failure`; 
* `DATABASE` - database name for which the backup was performed	(specified by `--dbname` option on the `gpbackup` command).
* `TYPE` - backup type:
    - `full` - contains user data, all global and local metadata for the database;
    - `incremental` – contains user data, all global and local metadata changed since a previous full backup;
    - `metadata-only` – contains only global and local metadata for the database;
    - `data-only` – contains only user data from the database.

* `OBJECT FILTERING` - whether the object filtering options were used when executing the `gpbackup` command:
    - `include-schema` – at least one `--include-schema` option was specified;
    - `exclude-schema` – at least one `--exclude-schema` option was specified;
    - `include-table` – at least one `--include-table` option was specified;
    - `exclude-table` – at least one `--exclude-table` option was specified;
    - `""` - no options was specified.

* `PLUGIN` - plugin name that was used to configure the backup destination;
* `DURATION` -  backup duration in the format `hh:mm:ss`;
* `DATE DELETED` - backup deletion status:
    - `In progress` - the deletion is in progress;
    - `Plugin Backup Delete Failed` - last delete attempt failed to delete backup from plugin storage;
    - `Local Delete Failed` - last delete attempt failed to delete backup from local storage.;
    - `""` - if backup is active;
    - date  in format `Mon Jan 02 2006 15:04:05` - if backup is deleted and deletion timestamp is set.

If gpbackup is launched without specifying `--metadata-only` flag, but there were no tables that contain data for backup, then gpbackup will only perform a `metadata-only` backup. The logs will contain messages like `No tables in backup set contain data. Performing metadata-only backup instead.` As a result, gpBackMan will display such backups as `metadata-only`.

#### Examples
Display info for active backups from `gpbackup_history.db`:
```bash
./gpbackman backup-info

 TIMESTAMP      | DATE                     | STATUS  | DATABASE | TYPE          | OBJECT FILTERING | PLUGIN             | DURATION | DATE DELETED 
----------------+--------------------------+---------+----------+---------------+------------------+--------------------+----------+--------------
 20230725101959 | Tue Jul 25 2023 10:19:59 | Success | demo     | incremental   |                  | gpbackup_s3_plugin | 00:00:22 |              
 20230725101152 | Tue Jul 25 2023 10:11:52 | Success | demo     | incremental   |                  | gpbackup_s3_plugin | 00:00:18 |              
 20230725101115 | Tue Jul 25 2023 10:11:15 | Success | demo     | full          |                  | gpbackup_s3_plugin | 00:00:20 |              
 20230724090000 | Mon Jul 24 2023 09:00:00 | Success | demo     | metadata-only |                  | gpbackup_s3_plugin | 00:25:17 |              
 20230723082000 | Sun Jul 23 2023 08:20:00 | Success | demo     | data-only     |                  | gpbackup_s3_plugin | 00:05:17 |              
```

Display info for all backups from `gpbackup_history.yaml`:
```bash
./gpbackman backup-info \
  --show-deleted \
  --show-failed \
  --history-file /data/master/gpseg-1/gpbackup_history.yaml

 TIMESTAMP      | DATE                     | STATUS  | DATABASE | TYPE          | OBJECT FILTERING | PLUGIN             | DURATION | DATE DELETED             
----------------+--------------------------+---------+----------+---------------+------------------+--------------------+----------+--------------------------
 20230809232817 | Wed Aug 09 2023 23:28:17 | Success | demo     | full          |                  |                    | 04:00:03 |                          
 20230806230400 | Sun Aug 06 2023 23:04:00 | Failure | demo     | full          |                  | gpbackup_s3_plugin | 00:00:38 |                          
 20230725110310 | Tue Jul 25 2023 11:03:10 | Success | demo     | incremental   |                  | gpbackup_s3_plugin | 00:00:18 | Wed Jul 26 2023 11:03:28 
 20230725101959 | Tue Jul 25 2023 10:19:59 | Success | demo     | incremental   |                  | gpbackup_s3_plugin | 00:00:22 |                          
 20230725101152 | Tue Jul 25 2023 10:11:52 | Success | demo     | incremental   |                  | gpbackup_s3_plugin | 00:00:18 |                          
 20230725101115 | Tue Jul 25 2023 10:11:15 | Success | demo     | full          |                  | gpbackup_s3_plugin | 00:00:20 |                          
 20230724090000 | Mon Jul 24 2023 09:00:00 | Success | demo     | metadata-only |                  | gpbackup_s3_plugin | 00:25:17 |                          
 20230723082000 | Sun Jul 23 2023 08:20:00 | Success | demo     | data-only     |                  | gpbackup_s3_plugin | 00:05:17 |                          
```

### Delete existing backup (`backup-delete`)

Available options for `backup-delete` command and their description:

```bash
./gpbackman backup-delete -h
Delete a specific backup set.

The --timestamp option must be specified. It could be specified multiple times.

By default, the existence of dependent backups is checked and deletion process is not performed,
unless the --cascade option is passed in.

If backup already deleted, the deletion process is skipped, unless --force option is specified.

By default, he deletion will be performed for local backup (in development).

The storage plugin config file location can be set using the --plugin-config option.
The full path to the file is required. In this case, the deletion will be performed using the storage plugin.

The gpbackup_history.db file location can be set using the --history-db option.
Can be specified only once. The full path to the file is required.

The gpbackup_history.yaml file location can be set using the --history-file option.
Can be specified multiple times. The full path to the file is required.

If no --history-file or --history-db options are specified, the history database will be searched in the current directory.

Only --history-file or --history-db option can be specified, not both.

Usage:
  gpbackman backup-delete [flags]

Flags:
      --cascade                 delete all dependent backups for the specified backup timestamp
      --force                   try to delete, even if the backup already mark as deleted
  -h, --help                    help for backup-delete
      --plugin-config string    the full path to plugin config file
      --timestamp stringArray   the backup timestamp for deleting, could be specified multiple times

Global Flags:
      --history-db string          full path to the gpbackup_history.db file
      --history-file stringArray   full path to the gpbackup_history.yaml file, could be specified multiple times
      --log-file string            full path to log file directory, if not specified, the log file will be created in the $HOME/gpAdminLogs directory
      --log-level-console string   level for console logging (error, info, debug, verbose) (default "info")
      --log-level-file string      level for file logging (error, info, debug, verbose) (default "info")
```

#### Examples
##### Delete existing backup from local storage

The functionality is in development.

gpBackMan returns a message:
```bash
[WARNING]:-The functionality is still in development
```
##### Delete existing backup using storage plugin
Delete specific backup:
```bash
./gpbackman backup-delete \
  --timestamp 20230725101959 \
  --plugin-config /tmp/gpbackup_plugin_config.yml
```

Delete specific backup and all dependent backups:
```bash
./gpbackman backup-delete\
  --timestamp 20230725101115 \
  --plugin-config /tmp/gpbackup_plugin_config.yml \
  --cascade
```

### Migrate history database (`history-migrate`)

Available options for `history-migrate` command and their description:

```bash
./gpbackman history-migrate -h
Migrate data from gpbackup_history.yaml to gpbackup_history.db SQLite history database.

The data from the gpbackup_history.yaml file will be uploaded to gpbackup_history.db SQLite history database.
If the gpbackup_history.db file does not exist, it will be created.
The gpbackup_history.yaml file will be renamed to gpbackup_history.yaml.migrated.

The gpbackup_history.db file location can be set using the  --history-db option.
Can be specified only once. The full path to the file is required.

The gpbackup_history.yaml file location can be set using the  --history-file option.
Can be specified multiple times. The full path to the file is required.

If no --history-file and/or --history-db options are specified, the files will be searched in the current directory.

Usage:
  gpbackman history-migrate [flags]

Flags:
  -h, --help   help for history-migrate

Global Flags:
      --history-db string          full path to the gpbackup_history.db file
      --history-file stringArray   full path to the gpbackup_history.yaml file, could be specified multiple times
      --log-file string            full path to log file directory, if not specified, the log file will be created in the $HOME/gpAdminLogs directory
      --log-level-console string   level for console logging (error, info, debug, verbose) (default "info")
      --log-level-file string      level for file logging (error, info, debug, verbose) (default "info")
```

#### Examples
Migrate data from several gpbackup_history.yaml files to gpbackup_history.db SQLite history database:
```bash
./gpbackman history-migrate \
  --history-file /data/master/gpseg-1/gpbackup_history.yaml \
  --history-file /tmp/gpbackup_history.yaml
```

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

#### Run container

```bash
docker run \
  --name gpbackman \
  -v /data/master/gpseg-1/gpbackup_history.db:/data/master/gpseg-1/gpbackup_history.db
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