package cmd

const (
	commandName = "gpbackman"

	// Plugin commands.
	// To be able to work with various plugins,
	// it is highly desirable to use the commands from the plugin specification.
	// See https://github.com/greenplum-db/gpbackup/blob/710fe53305958c1faed2e6008b894b4923bed253/plugins/README.md
	deleteBackupPluginCommand = "delete_backup"
	restoreDataPluginCommand  = "restore_data"

	historyFileNameBaseConst           = "gpbackup_history"
	historyFileNameSuffixConst         = ".yaml"
	historyFileNameMigratedSuffixConst = ".migrated"
	historyFileDBSuffixConst           = ".db"
	historyFileNameConst               = historyFileNameBaseConst + historyFileNameSuffixConst
	historyDBNameConst                 = historyFileNameBaseConst + historyFileDBSuffixConst

	// Flags.
	historyDBFlagName            = "history-db"
	historyFilesFlagName         = "history-file"
	logFileFlagName              = "log-file"
	logLevelConsoleFlagName      = "log-level-console"
	logLevelFileFlagName         = "log-level-file"
	timestampFlagName            = "timestamp"
	pluginConfigFileFlagName     = "plugin-config"
	reportFilePluginPathFlagName = "plugin-report-file-path"
	deletedFlagName              = "deleted"
	failedFlagName               = "failed"
	cascadeFlagName              = "cascade"
	forceFlagName                = "force"
	olderThenDaysFlagName        = "older-than-days"
	beforeTimestampFlagName      = "before-timestamp"
	typeFlagName                 = "type"
	tableFlagName                = "table"
	schemaFlagName               = "schema"
	excludeFlagName              = "exclude"
	backupDirFlagName            = "backup-dir"
	parallelProcessesFlagName    = "parallel-process"

	exitErrorCode = 1

	// Default for checking the existence of the file.
	checkFileExistsConst = true

	// Batch size for deleting from sqlite3.
	// This is to prevent problem with sqlite3.
	sqliteDeleteBatchSize = 1000
)

var (
	// Variable for determining history db format: file or sqlite db.
	// By default, true - sqlite db.
	historyDB = true
	// Timestamp to delete all backups before.
	beforeTimestamp string
)
