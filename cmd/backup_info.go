package cmd

import (
	"database/sql"
	"os"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpbackup/history"
	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/errtext"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

const (
	backupInfoShowDeletedFlagName = "show-deleted"
	backupInfoShowFailedFlagName  = "show-failed"
	backupInfoShowAllFlagName     = "show-all"
)

// Flags for the gpbackman backup-info command (backupInfoCmd)
var (
	backupInfoShowDeleted bool
	backupInfoShowFailed  bool
	backupInfoShowAll     bool
)

var backupInfoCmd = &cobra.Command{
	Use:   "backup-info",
	Short: "Display a list of backups",
	Long: `Display a list of backups.

By default, only active backups are displayed.

To display only deleted backups, use the --show-deleted flag.
To display only failed backups, use the --show-failed flag.
To display all backups, including deleted and failed, use the --show-all flag.

The gpbackup_history.db file location can be set using the --history-db option.
Can be specified only once. The full path to the file is required.

The gpbackup_history.yaml file location can be set using the --history-file option.
Can only be specified multiple times. The full path to the file is required.

If no --history-file or --history-db options are specified, the history database will be searched in the current directory.

Only --history-file or --history-db option can be specified, not both.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doRootFlagValidation(cmd.Flags())
		doRootBackupFlagValidation(cmd.Flags())
		doBackupInfoFlagValidation(cmd.Flags())
		doBackupInfo()
	},
}

func init() {
	rootCmd.AddCommand(backupInfoCmd)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowDeleted,
		backupInfoShowDeletedFlagName,
		false,
		"show only deleted backups",
	)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowFailed,
		backupInfoShowFailedFlagName,
		false,
		"show only failed backups",
	)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowAll,
		backupInfoShowAllFlagName,
		false,
		"show all backups, including deleted and failed",
	)
}

// These flag checks are applied only for backup-info commands.
func doBackupInfoFlagValidation(flags *pflag.FlagSet) {
	// show-deleted, show-failed and show-all flags cannot be set together for backup-info command.
	err := checkCompatibleFlags(flags, backupInfoShowDeletedFlagName, backupInfoShowFailedFlagName, backupInfoShowAllFlagName)
	if err != nil {
		gplog.Error(errtext.ErrorTextUnableCompatibleFlags(
			err,
			backupInfoShowDeletedFlagName,
			backupInfoShowFailedFlagName,
			backupInfoShowAllFlagName))
		execOSExit(exitErrorCode)
	}

}

func doBackupInfo() {
	logHeadersDebug()
	if historyDB {
		backupInfoDB()
	} else {
		backupInfoFile()
	}
}

func backupInfoDB() {
	hDB, err := openHistoryDB(getHistoryDBPath(rootHistoryDB))
	if err != nil {
		gplog.Error(errtext.ErrorTextUnableOpenHistoryDB(err))
	}
	backupList, err := getBackupNames(backupInfoShowDeleted, backupInfoShowFailed, backupInfoShowAll, hDB)
	if err != nil {
		gplog.Error(errtext.ErrorTextUnableReadHistoryDB(err))
	}
	t := table.NewWriter()
	initTable(t)
	for _, backupName := range backupList {
		hBackupData, err := history.GetMainBackupInfo(backupName, hDB)
		if err != nil {
			gplog.Error(errtext.ErrorTextUnableGetBackupInfo(backupName, err))
			continue
		}
		backupData := gpbckpconfig.ConvertFromHistoryBackupConfig(hBackupData)
		backupDate, err := backupData.GetBackupDate()
		if err != nil {
			gplog.Error(errtext.ErrorTextUnableGetBackupDate(backupName, err))
		}
		backupDuration, err := backupData.GetBackupDuration()
		if err != nil {
			gplog.Error(errtext.ErrorTextUnableGetBackupDuration(backupName, err))
		}
		backupDateDeleted, err := backupData.GetBackupDateDeleted()
		if err != nil {
			gplog.Error(errtext.ErrorTextUnableGetBackupDateDeletion(backupName, err))
		}
		t.AppendRow([]interface{}{
			backupData.Timestamp,
			backupDate,
			backupData.Status,
			backupData.DatabaseName,
			backupData.GetBackupType(),
			backupData.GetObjectFilteringInfo(),
			backupData.Plugin,
			backupDuration,
			backupDateDeleted,
		})
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
			gplog.Error(errtext.ErrorTextUnableReadHistoryFile(err))
			continue
		}
		parseHData, err := gpbckpconfig.ParseResult(historyData)
		if err != nil {
			gplog.Error(errtext.ErrorTextUnableParseHistoryFile(err))
		}
		if len(parseHData.BackupConfigs) != 0 {
			for _, backupData := range parseHData.BackupConfigs {
				backupName := backupData.Timestamp
				backupDateDeleted, err := backupData.GetBackupDateDeleted()
				if err != nil {
					gplog.Error(errtext.ErrorTextUnableGetBackupDateDeletion(backupName, err))
				}
				validBackup := getBackupNameFile(backupInfoShowDeleted, backupInfoShowFailed, backupInfoShowAll, backupData.Status, backupDateDeleted)
				if validBackup {
					backupDate, err := backupData.GetBackupDate()
					if err != nil {
						gplog.Error(errtext.ErrorTextUnableGetBackupDate(backupName, err))
					}
					backupDuration, err := backupData.GetBackupDuration()
					if err != nil {
						gplog.Error(errtext.ErrorTextUnableGetBackupDuration(backupName, err))
					}
					t.AppendRow([]interface{}{
						backupData.Timestamp,
						backupDate,
						backupData.Status,
						backupData.DatabaseName,
						backupData.GetBackupType(),
						backupData.GetObjectFilteringInfo(),
						backupData.Plugin,
						backupDuration,
						backupDateDeleted,
					})
				}
			}
		}
	}
	t.Render()
}

func getBackupNames(showD, showF, sAll bool, historyDB *sql.DB) ([]string, error) {
	orderBy := " ORDER BY timestamp DESC;"
	getBackupsQuery := "SELECT timestamp FROM backups"
	switch {
	case sAll:
		getBackupsQuery += orderBy
	case showD:
		getBackupsQuery += " WHERE status != '" + failureStatus + "'" +
			" AND date_deleted NOT IN ('', '" +
			deleteStatusInProgress + "', '" +
			deleteStatusPluginFailed + "', '" +
			deleteStatusLocalFailed + "')" + orderBy

	case showF:
		getBackupsQuery += " WHERE status = '" + failureStatus + "'" + orderBy
	default:
		getBackupsQuery += " WHERE status != '" + failureStatus + "'" +
			" AND date_deleted IN ('', '" +
			deleteStatusInProgress + "', '" +
			deleteStatusPluginFailed + "', '" +
			deleteStatusLocalFailed + "')" + orderBy
	}
	backupListRow, err := historyDB.Query(getBackupsQuery)
	if err != nil {
		return nil, err
	}
	defer backupListRow.Close()
	var backupList []string
	for backupListRow.Next() {
		var b string
		err := backupListRow.Scan(&b)
		if err != nil {
			return nil, err
		}
		backupList = append(backupList, b)
	}
	if err := backupListRow.Err(); err != nil {
		return nil, err
	}
	return backupList, nil
}

func getBackupNameFile(showD, showF, sAll bool, status, dateDeleted string) bool {
	switch {
	case sAll:
		return true
	case showD:
		if dateDeleted != "" {
			return true
		}
	case showF:
		if status == failureStatus {
			return true
		}
	default:
		if status != failureStatus && dateDeleted == "" {
			return true
		}
	}
	return false
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
