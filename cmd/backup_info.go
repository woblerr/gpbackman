package cmd

import (
	"os"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

const (
	backupInfoShowDeletedFlagName = "show-deleted"
	backupInfoShowFailedFlagName  = "show-failed"
)

// Flags for the gpbackman backup-info command (backupInfoCmd)
var (
	backupInfoShowDeleted bool
	backupInfoShowFailed  bool
)

var backupInfoCmd = &cobra.Command{
	Use:   "backup-info",
	Short: "Display a list of backups",
	Long: `Display a list of backups.

By default, only active backups or backups with deletion status "In progress" from gpbackup_history.db are displayed.

To additional display deleted backups, use the --show-deleted option.
To additional display failed backups, use the --show-failed option.
To display all backups, use --show-deleted  and --show-failed options together.

The gpbackup_history.db file location can be set using the --history-db option.
Can be specified only once. The full path to the file is required.

The gpbackup_history.yaml file location can be set using the --history-file option.
Can be specified multiple times. The full path to the file is required.

If no --history-file or --history-db options are specified, the history database will be searched in the current directory.

Only --history-file or --history-db option can be specified, not both.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doRootFlagValidation(cmd.Flags())
		doRootBackupFlagValidation(cmd.Flags())
		// doBackupInfoFlagValidation(cmd.Flags())
		doBackupInfo()
	},
}

func init() {
	rootCmd.AddCommand(backupInfoCmd)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowDeleted,
		backupInfoShowDeletedFlagName,
		false,
		"show deleted backups",
	)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowFailed,
		backupInfoShowFailedFlagName,
		false,
		"show failed backups",
	)
}

// These flag checks are applied only for backup-info commands.
// func doBackupInfoFlagValidation(flags *pflag.FlagSet) {
// }

func doBackupInfo() {
	logHeadersDebug()
	if historyDB {
		backupInfoDB()
	} else {
		backupInfoFile()
	}
}

func backupInfoDB() {
	hDB, err := gpbckpconfig.OpenHistoryDB(getHistoryDBPath(rootHistoryDB))
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableOpenHistoryDB(err))
	}
	backupList, err := gpbckpconfig.GetBackupNamesDB(backupInfoShowDeleted, backupInfoShowFailed, hDB)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableReadHistoryDB(err))
	}
	t := table.NewWriter()
	initTable(t)
	for _, backupName := range backupList {
		backupData, err := gpbckpconfig.GetBackupDataDB(backupName, hDB)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
			continue
		}
		addBackupToTable(backupData, t)

	}
	hDB.Close()
	t.Render()
}

func backupInfoFile() {
	t := table.NewWriter()
	initTable(t)
	for _, historyFile := range rootHistoryFiles {
		hFile := getHistoryFilePath(historyFile)
		historyData, err := gpbckpconfig.ReadHistoryFile(hFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("read", err))
			continue
		}
		parseHData, err := gpbckpconfig.ParseResult(historyData)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("parse", err))
			continue
		}
		if len(parseHData.BackupConfigs) != 0 {
			for _, backupData := range parseHData.BackupConfigs {
				backupDateDeleted, err := backupData.GetBackupDateDeleted()
				if err != nil {
					gplog.Error(textmsg.ErrorTextUnableGetBackupValue("date deletion", backupData.Timestamp, err))
				}
				validBackup := gpbckpconfig.GetBackupNameFile(backupInfoShowDeleted, backupInfoShowFailed, backupData.Status, backupDateDeleted)
				if validBackup {
					addBackupToTable(backupData, t)
				}
			}
		}
	}
	t.Render()
}

func initTable(t table.Writer) {
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDefault)
	t.Style().Options.DrawBorder = false
	t.AppendHeader(table.Row{
		"timestamp",
		"date",
		"status",
		"database",
		"type",
		"object filtering",
		"plugin",
		"duration",
		"date deleted",
	})
	t.SortBy([]table.SortBy{{Name: "timestamp", Mode: table.Dsc}})
}

func addBackupToTable(backupData gpbckpconfig.BackupConfig, t table.Writer) {
	backupDate, err := backupData.GetBackupDate()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("date", backupData.Timestamp, err))
	}
	backupType, err := backupData.GetBackupType()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("type", backupData.Timestamp, err))
	}
	backupFilter, err := backupData.GetObjectFilteringInfo()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("object filtering", backupData.Timestamp, err))
	}
	backupDuration, err := backupData.GetBackupDuration()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("duration", backupData.Timestamp, err))
	}
	backupDateDeleted, err := backupData.GetBackupDateDeleted()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("date deletion", backupData.Timestamp, err))
	}
	t.AppendRow([]interface{}{
		backupData.Timestamp,
		backupDate,
		backupData.Status,
		backupData.DatabaseName,
		backupType,
		backupFilter,
		backupData.Plugin,
		formatBackupDuration(backupDuration),
		backupDateDeleted,
	})
}
