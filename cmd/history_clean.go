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
	historyCleanBeforeTimestamp      string
	historyCleanOlderThanDays        uint
	historyCleanNoHistorySyncStandby bool
	historyCleanDatabase             string
)

var historyCleanCmd = &cobra.Command{
	Use:   "history-clean",
	Short: "Clean deleted backups from the history database",
	Long: `Clean deleted backups from the history database.
Only the database is being cleaned up.

Information is deleted only about deleted backups from gpbackup_history.db. Each backup must be deleted first.

To delete information about backups older than the given timestamp, use the --before-timestamp option. 
To delete information about backups older than the given number of days, use the --older-than-day option. 
Only --older-than-days or --before-timestamp option must be specified, not both.

Use --database to clean history only for the specified database. Without --database,
cleanup includes deleted backup history for all databases in the history database.
Database names are matched exactly and case-sensitively against backup history.

The gpbackup_history.db file location can be set using the --history-db option.
Can be specified only once. The full path to the file is required.
If the --history-db option is not specified, gpbackman uses gpbackup_history.db in the current directory.
Pass --auto-load-history-db to resolve it from MASTER_DATA_DIRECTORY first, then COORDINATOR_DATA_DIRECTORY.

After successful cleanup, gpBackMan syncs the cluster gpbackup_history.db to an up standby coordinator when sync conditions are met.
Pass --no-history-sync-standby to disable this sync.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doRootFlagValidation(cmd.Flags(), checkFileExistsConst)
		doCleanHistoryFlagValidation(cmd.Flags())
		doCleanHistory()
	},
}

func init() {
	rootCmd.AddCommand(historyCleanCmd)
	historyCleanCmd.Flags().StringVar(
		&historyCleanDatabase,
		databaseFlagName,
		"",
		"delete backup history only for the specified database",
	)
	historyCleanCmd.PersistentFlags().UintVar(
		&historyCleanOlderThanDays,
		olderThanDaysFlagName,
		0,
		"delete information about backups older than the given number of days",
	)
	historyCleanCmd.PersistentFlags().StringVar(
		&historyCleanBeforeTimestamp,
		beforeTimestampFlagName,
		"",
		"delete information about backups older than the given timestamp",
	)
	historyCleanCmd.PersistentFlags().BoolVar(
		&historyCleanNoHistorySyncStandby,
		noHistorySyncStandbyFlagName,
		false,
		"disable gpbackup_history.db sync to the standby coordinator",
	)
	historyCleanCmd.PersistentFlags().IntVar(
		&historyStandbySyncTimeoutSeconds,
		historySyncStandbyTimeoutFlagName,
		historySyncStandbyTimeoutDefault,
		"shared rsync and remote install timeout in seconds; must be an integer between 1 and 86400",
	)
	historyCleanCmd.MarkFlagsMutuallyExclusive(beforeTimestampFlagName, olderThanDaysFlagName)
}

// These flag checks are applied only for history-clean command.
func doCleanHistoryFlagValidation(flags *pflag.FlagSet) {
	var err error
	if flags.Changed(databaseFlagName) && historyCleanDatabase == "" {
		gplog.Error("%s", textmsg.ErrorTextUnableValidateFlag(historyCleanDatabase, databaseFlagName, textmsg.ErrorEmptyDatabase()))
		execOSExit(exitErrorCode)
	}
	// If before-timestamp are specified and have correct values.
	if flags.Changed(beforeTimestampFlagName) {
		err = gpbckpconfig.CheckTimestamp(historyCleanBeforeTimestamp)
		if err != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableValidateFlag(historyCleanBeforeTimestamp, beforeTimestampFlagName, err))
			execOSExit(exitErrorCode)
		}
		beforeTimestamp = historyCleanBeforeTimestamp
	}
	if flags.Changed(olderThanDaysFlagName) {
		beforeTimestamp = gpbckpconfig.GetTimestampOlderThan(historyCleanOlderThanDays)
	}
	if beforeTimestamp == "" {
		gplog.Error("%s", textmsg.ErrorTextUnableValidateValue(textmsg.ErrorValidationValue(), olderThanDaysFlagName, beforeTimestampFlagName))
		execOSExit(exitErrorCode)
	}
}

func doCleanHistory() {
	logHeadersDebug()
	runHistoryMutationWithStandbySync(cleanHistory, historyCleanNoHistorySyncStandby)
}

func cleanHistory() (string, error) {
	hDB, err := gpbckpconfig.OpenHistoryDB(getHistoryDBPath(rootHistoryDB, rootAutoLoadHistoryDB))
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableActionHistoryDB("open", err))
		return "", err
	}
	defer func() {
		closeErr := hDB.Close()
		if closeErr != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableActionHistoryDB("close", closeErr))
		}
	}()
	err = historyCleanDB(beforeTimestamp, historyCleanDatabase, hDB)
	if err != nil {
		return "", err
	}
	return "", nil
}

func historyCleanDB(cutOffTimestamp, databaseName string, hDB *sql.DB) error {
	backupList, err := gpbckpconfig.GetBackupNamesForCleanBeforeTimestamp(cutOffTimestamp, databaseName, hDB)
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableReadHistoryDB(err))
		return err
	}
	if len(backupList) > 0 {
		gplog.Debug("%s", textmsg.InfoTextBackupDeleteListFromHistory(backupList))
		err := gpbckpconfig.CleanBackupsDB(backupList, sqliteDeleteBatchSize, hDB)
		if err != nil {
			return err
		}
	} else {
		gplog.Info("%s", textmsg.InfoTextNothingToDo())
	}
	return nil
}
