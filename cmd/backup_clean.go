package cmd

import (
	"database/sql"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpbackup/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

// Flags for the gpbackman backup-clean command (backupCleanCmd)
var (
	backupCleanBeforeTimestamp  string
	backupCleanPluginConfigFile string
	backupCleanOlderThenDays    uint
	backupCleanCascade          bool
)

var backupCleanCmd = &cobra.Command{
	Use:   "backup-clean",
	Short: "Delete backups by condition",
	Long: `Delete backups by condition.

To delete backup sets older than the given timestamp, use the --before-timestamp option. 
To delete backup sets older than the given number of days, use the --older-than-day option. 
Only --older-than-days or --before-timestamp option must be specified, not both.

By default, the existence of dependent backups is checked and deletion process is not performed,
unless the --cascade option is passed in.

By default, the deletion will be performed for local backup (in development).

The storage plugin config file location can be set using the --plugin-config option.
The full path to the file is required. In this case, the deletion will be performed using the storage plugin.

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
		doCleanBackupFlagValidation(cmd.Flags())
		doCleanBackup()
	},
}

func init() {
	rootCmd.AddCommand(backupCleanCmd)
	backupCleanCmd.PersistentFlags().StringVar(
		&backupCleanPluginConfigFile,
		pluginConfigFileFlagName,
		"",
		"the full path to plugin config file",
	)
	backupCleanCmd.PersistentFlags().BoolVar(
		&backupCleanCascade,
		cascadeFlagName,
		false,
		"delete all dependent backups",
	)
	backupCleanCmd.PersistentFlags().UintVar(
		&backupCleanOlderThenDays,
		olderThenDaysFlagName,
		0,
		"delete backup sets older than the given number of days",
	)
	backupCleanCmd.PersistentFlags().StringVar(
		&backupCleanBeforeTimestamp,
		beforeTimestampFlagName,
		"",
		"delete backup sets older than the given timestamp",
	)
	backupCleanCmd.MarkFlagsMutuallyExclusive(beforeTimestampFlagName, olderThenDaysFlagName)
}

// These flag checks are applied only for backup-clean command.
func doCleanBackupFlagValidation(flags *pflag.FlagSet) {
	var err error
	// If before-timestamp are specified and have correct values.
	if flags.Changed(beforeTimestampFlagName) {
		err = gpbckpconfig.CheckTimestamp(backupCleanBeforeTimestamp)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(backupCleanBeforeTimestamp, beforeTimestampFlagName, err))
			execOSExit(exitErrorCode)
		}
		beforeTimestamp = backupCleanBeforeTimestamp
	}
	if flags.Changed(olderThenDaysFlagName) {
		beforeTimestamp = gpbckpconfig.GetTimestampOlderThen(backupCleanOlderThenDays)
	}
	// If plugin-config flag is specified and full path.
	if flags.Changed(pluginConfigFileFlagName) {
		err = gpbckpconfig.CheckFullPath(backupCleanPluginConfigFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(backupCleanPluginConfigFile, pluginConfigFileFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
	if beforeTimestamp == "" {
		gplog.Error(textmsg.ErrorTextUnableValidateValue(textmsg.ErrorValidationValue(), olderThenDaysFlagName, beforeTimestampFlagName))
		execOSExit(exitErrorCode)
	}
}

func doCleanBackup() {
	logHeadersDebug()
	err := cleanBackup()
	if err != nil {
		execOSExit(exitErrorCode)
	}
}

func cleanBackup() error {
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
		if len(backupCleanPluginConfigFile) > 0 {
			pluginConfig, err := utils.ReadPluginConfig(backupCleanPluginConfigFile)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableReadPluginConfigFile(err))
				return err
			}
			err = backupCleanDBPlugin(backupCleanCascade, pluginConfig, hDB)
			if err != nil {
				return err
			}
		} else {
			err := backupCleanDBLocal()
			if err != nil {
				return err
			}
		}
	} else {
		if len(backupCleanPluginConfigFile) > 0 {
			pluginConfig, err := utils.ReadPluginConfig(backupCleanPluginConfigFile)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableReadPluginConfigFile(err))
				return err
			}
			err = backupCleanFilePlugin(backupCleanCascade, pluginConfig)
			if err != nil {
				return err
			}
		} else {
			err := backupCleanFileLocal()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func backupCleanDBPlugin(deleteCascade bool, pluginConfig *utils.PluginConfig, hDB *sql.DB) error {
	backupList, err := gpbckpconfig.GetBackupNamesBeforeTimestamp(beforeTimestamp, hDB)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableReadHistoryDB(err))
		return err
	}
	gplog.Debug(textmsg.InfoTextBackupDeleteList(backupList))
	// Execute deletion for each backup.
	// Use backupDeleteDBPlugin function from backup-delete command.
	err = backupDeleteDBPlugin(backupList, deleteCascade, pluginConfig, hDB)
	if err != nil {
		return err
	}
	return nil
}

func backupCleanFilePlugin(deleteCascade bool, pluginConfig *utils.PluginConfig) error {
	return nil
}

// TODO
func backupCleanDBLocal() error {
	gplog.Warn("The functionality is still in development")
	return nil
}

// TODO
func backupCleanFileLocal() error {
	gplog.Warn("The functionality is still in development")
	return nil
}
