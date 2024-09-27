package cmd

import (
	"database/sql"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

// Flags for the gpbackman history-clean command (historyCleanCmd)
var (
	historyCleanBeforeTimestamp string
	historyCleanOlderThenDays   uint
)

var historyCleanCmd = &cobra.Command{
	Use:   "history-clean",
	Short: "Clean failed and deleted backups from the history database",
	Long: `Clean failed and deleted backups from the history database.
Only the database is being cleaned up.

Information is deleted only about deleted backups from gpbackup_history.db. Each backup must be deleted first.

To delete information about backups older than the given timestamp, use the --before-timestamp option. 
To delete information about backups older than the given number of days, use the --older-than-day option. 
Only --older-than-days or --before-timestamp option must be specified, not both.

The gpbackup_history.db file location can be set using the --history-db option.
Can be specified only once. The full path to the file is required.

The gpbackup_history.yaml file location can be set using the --history-file option.
Can be specified multiple times. The full path to the file is required.

If no --history-file or --history-db options are specified, the history database will be searched in the current directory.

Only --history-file or --history-db option can be specified, not both.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doRootFlagValidation(cmd.Flags(), checkFileExistsConst)
		doRootBackupFlagValidation(cmd.Flags())
		doCleanHistoryFlagValidation(cmd.Flags())
		doCleanHistory()
	},
}

func init() {
	rootCmd.AddCommand(historyCleanCmd)
	historyCleanCmd.PersistentFlags().UintVar(
		&historyCleanOlderThenDays,
		olderThenDaysFlagName,
		0,
		"delete information about backups older than the given number of days",
	)
	historyCleanCmd.PersistentFlags().StringVar(
		&historyCleanBeforeTimestamp,
		beforeTimestampFlagName,
		"",
		"delete information about backups older than the given timestamp",
	)
	historyCleanCmd.MarkFlagsMutuallyExclusive(beforeTimestampFlagName, olderThenDaysFlagName)
}

// These flag checks are applied only for backup-clean command.
func doCleanHistoryFlagValidation(flags *pflag.FlagSet) {
	var err error
	// If before-timestamp are specified and have correct values.
	if flags.Changed(beforeTimestampFlagName) {
		err = gpbckpconfig.CheckTimestamp(historyCleanBeforeTimestamp)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(historyCleanBeforeTimestamp, beforeTimestampFlagName, err))
			execOSExit(exitErrorCode)
		}
		beforeTimestamp = historyCleanBeforeTimestamp
	}
	if flags.Changed(olderThenDaysFlagName) {
		beforeTimestamp = gpbckpconfig.GetTimestampOlderThen(historyCleanOlderThenDays)
	}
	if beforeTimestamp == "" {
		gplog.Error(textmsg.ErrorTextUnableValidateValue(textmsg.ErrorValidationValue(), olderThenDaysFlagName, beforeTimestampFlagName))
		execOSExit(exitErrorCode)
	}
}

func doCleanHistory() {
	logHeadersDebug()
	err := cleanHistory()
	if err != nil {
		execOSExit(exitErrorCode)
	}
}

func cleanHistory() error {
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
		err = historyCleanDB(beforeTimestamp, hDB)
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
				err = historyCleanFile(beforeTimestamp, &parseHData)
				if err != nil {
					return err
				}
			}
			errUpdateHFile := parseHData.UpdateHistoryFile(hFile)
			if errUpdateHFile != nil {
				gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("update", errUpdateHFile))
				return errUpdateHFile
			}
		}
	}
	return nil
}

func historyCleanDB(cutOffTimestamp string, hDB *sql.DB) error {
	backupList, err := gpbckpconfig.GetBackupNamesForCleanBeforeTimestamp(cutOffTimestamp, hDB)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableReadHistoryDB(err))
		return err
	}
	if len(backupList) > 0 {
		gplog.Debug(textmsg.InfoTextBackupDeleteListFromHistory(backupList))
		err := gpbckpconfig.CleanBackupsDB(backupList, sqliteDeleteBatchSize, hDB)
		if err != nil {
			return err
		}
	} else {
		gplog.Info(textmsg.InfoTextNothingToDo())
	}
	return nil
}

func historyCleanFile(cutOffTimestamp string, parseHData *gpbckpconfig.History) error {
	backupIdxs := make([]int, 0)
	backupList := make([]string, 0)
	for idx, backupConfig := range parseHData.BackupConfigs {
		// In history file we have sorted timestamps by descending order.
		if backupConfig.Timestamp < cutOffTimestamp {
			backupDateDeleted, err := backupConfig.GetBackupDateDeleted()
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableGetBackupValue("date deletion", backupConfig.Timestamp, err))
				return err
			}
			if !gpbckpconfig.IsBackupActive(backupDateDeleted) && (backupDateDeleted != gpbckpconfig.DateDeletedInProgress) {
				backupIdxs = append(backupIdxs, idx)
				backupList = append(backupList, backupConfig.Timestamp)
			}
		}
	}
	if len(backupList) > 0 {
		gplog.Debug(textmsg.InfoTextBackupDeleteListFromHistory(backupList))
		parseHData.RemoveMultipleFromHistoryFile(backupIdxs)
	} else {
		gplog.Info(textmsg.InfoTextNothingToDo())
	}
	return nil
}
