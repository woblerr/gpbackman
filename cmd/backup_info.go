package cmd

import (
	"database/sql"
	"os"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

// Flags for the gpbackman backup-info command (backupInfoCmd)
var (
	backupInfoShowDeleted      bool
	backupInfoShowFailed       bool
	backupInfoBackupTypeFilter string
)

var backupInfoCmd = &cobra.Command{
	Use:   "backup-info",
	Short: "Display information about backups",
	Long: `Display information about backups.

By default, only active backups or backups with deletion status "In progress" from gpbackup_history.db are displayed.

To additional display deleted backups, use the --deleted option.
To additional display failed backups, use the --failed option.
To display all backups, use --deleted and --failed options together.

To display backups of a specific type, use the --type option.

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
		doBackupInfoFlagValidation(cmd.Flags())
		doBackupInfo()
	},
}

func init() {
	rootCmd.AddCommand(backupInfoCmd)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowDeleted,
		deletedFlagName,
		false,
		"show deleted backups",
	)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowFailed,
		failedFlagName,
		false,
		"show failed backups",
	)
	backupInfoCmd.Flags().StringVar(
		&backupInfoBackupTypeFilter,
		typeFlagName,
		"",
		"backup type filter (full, incremental, data-only, metadata-only)",
	)
}

// These flag checks are applied only for backup-info commands.
func doBackupInfoFlagValidation(flags *pflag.FlagSet) {
	var err error
	// If type is specified and have correct values.
	if flags.Changed(typeFlagName) {
		err = checkBackupType(backupInfoBackupTypeFilter)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(backupInfoBackupTypeFilter, typeFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
}

func doBackupInfo() {
	logHeadersDebug()
	err := backupInfo()
	if err != nil {
		execOSExit(exitErrorCode)
	}
}

func backupInfo() error {
	t := table.NewWriter()
	initTable(t)
	if historyDB {
		hDB, err := gpbckpconfig.OpenHistoryDB(getHistoryDBPath(rootHistoryDB))
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryDB("open", err))
			return err
		}
		defer func() {
			closeErr := hDB.Close()
			if closeErr != nil {
				gplog.Error(textmsg.ErrorTextUnableActionHistoryDB("close", closeErr))
			}
		}()
		err = backupInfoDB(backupInfoShowDeleted, backupInfoShowFailed, backupInfoBackupTypeFilter, hDB, t)
		if err != nil {
			return err
		}
	} else {
		for _, historyFile := range rootHistoryFiles {
			hFile := getHistoryFilePath(historyFile)
			parseHData, err := getDataFromHistoryFile(hFile)
			if err != nil {
				return err
			}
			if len(parseHData.BackupConfigs) != 0 {
				err = backupInfoFile(backupInfoShowDeleted, backupInfoShowFailed, backupInfoBackupTypeFilter, parseHData, t)
				if err != nil {
					return err
				}
			}
		}
	}
	t.Render()
	return nil
}

func backupInfoDB(showDeleted, showFailed bool, backupTypeFilter string, hDB *sql.DB, t table.Writer) error {
	backupList, err := gpbckpconfig.GetBackupNamesDB(showDeleted, showFailed, hDB)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableReadHistoryDB(err))
		return err
	}
	for _, backupName := range backupList {
		backupData, err := gpbckpconfig.GetBackupDataDB(backupName, hDB)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
			return err
		}
		addBackupToTable(backupTypeFilter, backupData, t)
	}
	return nil
}

func backupInfoFile(showDeleted, showFailed bool, backupTypeFilter string, parseHData gpbckpconfig.History, t table.Writer) error {
	for _, backupData := range parseHData.BackupConfigs {
		backupDateDeleted, err := backupData.GetBackupDateDeleted()
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupValue("date deletion", backupData.Timestamp, err))
		}
		validBackup := gpbckpconfig.GetBackupNameFile(showDeleted, showFailed, backupData.Status, backupDateDeleted)
		if validBackup {
			addBackupToTable(backupTypeFilter, backupData, t)
		}
	}
	return nil
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

// If errors occur, they are logged, but they are not returned.
// The main idea is to show the maximum available information and display all errors that occur.
// But do not fall when errors occur. So, display anyway.
func addBackupToTable(backupTypeFilter string, backupData gpbckpconfig.BackupConfig, t table.Writer) {
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
	if (backupTypeFilter == "") || (backupTypeFilter != "" && backupTypeFilter == backupType) {
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
}
