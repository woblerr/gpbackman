package cmd

const (
	commandName = "gpbackman"

	deleteBackupPluginCommand = "delete_backup"

	historyFileNameBaseConst           = "gpbackup_history"
	historyFileNameSuffixConst         = ".yaml"
	historyFileNameMigratedSuffixConst = ".migrated"
	historyFileDBSuffixConst           = ".db"
	historyFileNameConst               = historyFileNameBaseConst + historyFileNameSuffixConst
	historyDBNameConst                 = historyFileNameBaseConst + historyFileDBSuffixConst

	exitErrorCode = 1
)

var (
	// Variable for determining history db format: file or sqlite db.
	// By default, true - sqlite db.
	historyDB bool = true
)
