package cmd

import (
	"bytes"
	"database/sql"
	"os/exec"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpbackup/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

const (
	backupDeleteTimestampFlagName        = "timestamp"
	backupDeletePluginConfigFileFlagName = "plugin-config"
	backupDeleteCascadeFlagName          = "cascade"
	backupDeleteForceFlagName            = "force"
)

// Flags for the gpbackman backup-delete command (backupDeleteCmd)
var (
	backupDeleteTimestamp        []string
	backupDeletePluginConfigFile string
	backupDeleteCascade          bool
	backupDeleteForce            bool
)
var backupDeleteCmd = &cobra.Command{
	Use:   "backup-delete",
	Short: "Delete a specific backup set",
	Long: `Delete a specific backup set.

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
Can only be specified multiple times. The full path to the file is required.

If no --history-file or --history-db options are specified, the history database will be searched in the current directory.

Only --history-file or --history-db option can be specified, not both.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doRootFlagValidation(cmd.Flags())
		doRootBackupFlagValidation(cmd.Flags())
		doDeleteBackupFlagValidation(cmd.Flags())
		doDeleteBackup()
	},
}

var execCommand = exec.Command

func init() {
	rootCmd.AddCommand(backupDeleteCmd)
	backupDeleteCmd.PersistentFlags().StringArrayVar(
		&backupDeleteTimestamp,
		backupDeleteTimestampFlagName,
		[]string{""},
		"the backup timestamp for deleting, could be specified multiple times",
	)
	backupDeleteCmd.PersistentFlags().StringVar(
		&backupDeletePluginConfigFile,
		backupDeletePluginConfigFileFlagName,
		"",
		"the full path to plugin config file",
	)
	backupDeleteCmd.PersistentFlags().BoolVar(
		&backupDeleteCascade,
		backupDeleteCascadeFlagName,
		false,
		"delete all dependent backups for the specified backup timestamp",
	)
	backupDeleteCmd.PersistentFlags().BoolVar(
		&backupDeleteForce,
		backupDeleteForceFlagName,
		false,
		"try to delete, even if the backup already mark as deleted",
	)
	backupDeleteCmd.MarkPersistentFlagRequired(backupDeleteTimestampFlagName)
}

// These flag checks are applied only for backup-delete command.
func doDeleteBackupFlagValidation(flags *pflag.FlagSet) {
	var err error
	// If timestamps are specified and have correct values.
	if flags.Changed(backupDeleteTimestampFlagName) {
		for _, timestamp := range backupDeleteTimestamp {
			err = gpbckpconfig.CheckTimestamp(timestamp)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableValidateFlag(timestamp, backupDeleteTimestampFlagName, err))
				execOSExit(exitErrorCode)
			}
		}
	}
	// If plugin-config flag is specified and full path.
	if flags.Changed(backupDeletePluginConfigFileFlagName) {
		err = gpbckpconfig.CheckFullPath(backupDeletePluginConfigFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(backupDeletePluginConfigFile, backupDeletePluginConfigFileFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
	// history-file flag and history-db flags cannot be used together for backup-delete command.
	err = checkCompatibleFlags(flags, rootHistoryDBFlagName, rootHistoryFilesFlagName)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableCompatibleFlags(err, rootHistoryDBFlagName, rootHistoryFilesFlagName))
		execOSExit(exitErrorCode)
	}
}

func doDeleteBackup() {
	logHeadersInfo()
	logHeadersDebug()
	if len(backupDeletePluginConfigFile) > 0 {
		pluginConfig, err := utils.ReadPluginConfig(backupDeletePluginConfigFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableReadPluginConfigFile(err))
			execOSExit(exitErrorCode)
		}
		if historyDB {
			backupDeleteDBPlugin(pluginConfig)
		} else {
			backupDeleteFilePlugin(pluginConfig)
		}
	} else {
		if historyDB {
			backupDeleteDBLocal()
		} else {
			backupDeleteFileLocal()
		}
	}
}

func backupDeleteDBPlugin(pluginConfig *utils.PluginConfig) {
	hDB, err := gpbckpconfig.OpenHistoryDB(getHistoryDBPath(rootHistoryDB))
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableOpenHistoryDB(err))
		execOSExit(exitErrorCode)
	}
	for _, backupName := range backupDeleteTimestamp {
		backupData, err := gpbckpconfig.GetBackupDataDB(backupName, hDB)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
			continue
		}
		backupSuccessStatus, err := backupData.IsSuccess()
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
			continue
		}
		if backupSuccessStatus {
			if len(backupData.RestorePlan) > 0 {
				if backupDeleteCascade {
					// If the deletion of at least one dependent backup fails, we fail full entire chain.
					err = backupDeleteDBCascade(backupData.RestorePlan, hDB, pluginConfig)
					if err != nil {
						gplog.Error(textmsg.ErrorTextUnableDeleteBackupCascade(backupName, textmsg.ErrorBackupDeleteCascadeError()))
						// Skip deleting the original backup,
						// because an error occurred when deleting the dependencies.
						continue
					}
					_ = backupDeleteDBPluginFunc(backupName, backupData, hDB, pluginConfig)
				} else {
					gplog.Error(textmsg.ErrorTextUnableDeleteBackupUseCascade(backupName, textmsg.ErrorBackupDeleteCascadeOptionError()))
					continue
				}
			} else {
				_ = backupDeleteDBPluginFunc(backupName, backupData, hDB, pluginConfig)
			}
		} else {
			gplog.Warn(textmsg.WarnTextBackupUnableDeleteFailed(backupName))
		}
	}
	hDB.Close()
}

func backupDeleteDBCascade(restorePlanEntry []gpbckpconfig.RestorePlanEntry, hDB *sql.DB, pluginConfig *utils.PluginConfig) error {
	for _, restorePlanData := range restorePlanEntry {
		backupNameRestorePlan := restorePlanData.Timestamp
		backupDataRestorePlan, err := gpbckpconfig.GetBackupDataDB(backupNameRestorePlan, hDB)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupNameRestorePlan, err))
			return err
		}
		err = backupDeleteDBPluginFunc(backupNameRestorePlan, backupDataRestorePlan, hDB, pluginConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func backupDeleteDBPluginFunc(backupName string, backupData gpbckpconfig.BackupConfig, hDB *sql.DB, pluginConfig *utils.PluginConfig) error {
	var err error
	backupDateDeleted, errDateDeleted := backupData.GetBackupDateDeleted()
	if errDateDeleted != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("date deletion", backupData.Timestamp, errDateDeleted))
	}
	// If the backup date deletion has invalid value, try to delete the backup.
	if gpbckpconfig.IsBackupActive(backupDateDeleted) || errDateDeleted != nil {
		dateDeleted := getCurrentTimestamp()
		stdout, stderr, errdel := execDeleteBackup(pluginConfig.ExecutablePath, deleteBackupPluginCommand, backupDeletePluginConfigFile, backupName)
		if len(stderr) > 0 {
			gplog.Error(stderr)
		}
		if errdel != nil {
			err = updateDeleteStatus(backupName, gpbckpconfig.DateDeletedPluginFailed, hDB)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableSetBackupStatus(gpbckpconfig.DateDeletedPluginFailed, backupName, err))
			}
			gplog.Error(textmsg.ErrorTextUnableDeleteBackup(backupName, errdel))
			return errdel
		}
		gplog.Info(stdout)
		err = updateDeleteStatus(backupName, dateDeleted, hDB)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableSetBackupStatus(dateDeleted, backupName, err))
			return errdel
		}
		gplog.Info(textmsg.InfoTextBackupDeleteSuccess(backupName))
	} else {
		if backupDateDeleted == gpbckpconfig.DateDeletedInProgress {
			gplog.Warn(textmsg.ErrorTextBackupInProgress(backupName, textmsg.ErrorBackupDeleteInProgressError()))
			return textmsg.ErrorBackupDeleteInProgressError()
		} else {
			gplog.Warn(textmsg.WarnTextBackupAlreadyDeleted(backupName))
		}
	}
	return nil
}

func backupDeleteFilePlugin(pluginConfig *utils.PluginConfig) {
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
			for _, backupName := range backupDeleteTimestamp {
				backupPositionInHistoryFile, backupData, err := parseHData.FindBackupConfig(backupName)
				if err != nil {
					gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
					continue
				}
				gplog.Info(textmsg.InfoTextBackupDeleteStart(backupName))
				if checkBackupCanBeDeleted(backupData) {
					backupDependencies := parseHData.FindBackupConfigDependencies(backupName, backupPositionInHistoryFile)
					if len(backupDependencies) > 0 {
						gplog.Info(textmsg.InfoTextBackupDependenciesList(backupName, backupDependencies))
						if backupDeleteCascade {
							// If the deletion of at least one dependent backup fails, we fail full entire chain.
							err := backupDeleteFileCascade(backupDependencies, &parseHData, pluginConfig)
							if err != nil {
								gplog.Error(textmsg.ErrorTextUnableDeleteBackupCascade(backupName, textmsg.ErrorBackupDeleteCascadeError()))
								// Skip deleting the original backup,
								// because an error occurred when deleting the dependencies.
								continue
							}
						} else {
							gplog.Error(textmsg.ErrorTextUnableDeleteBackupUseCascade(backupName, textmsg.ErrorBackupDeleteCascadeOptionError()))
							continue
						}
					}
					_ = backupDeleteFilePluginFunc(backupData, &parseHData, pluginConfig)
				}
			}
		}
		errUpdateHFile := parseHData.UpdateHistoryFile(hFile)
		if errUpdateHFile != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("update", errUpdateHFile))
		}
	}
}

func backupDeleteFileCascade(backupList []string, parseHData *gpbckpconfig.History, pluginConfig *utils.PluginConfig) error {
	for _, backup := range backupList {
		gplog.Info(textmsg.InfoTextBackupDeleteStart(backup))
		_, backupData, err := parseHData.FindBackupConfig(backup)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backup, err))
			return err
		}
		if checkBackupCanBeDeleted(backupData) {
			err = backupDeleteFilePluginFunc(backupData, parseHData, pluginConfig)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func backupDeleteFilePluginFunc(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History, pluginConfig *utils.PluginConfig) error {
	var err error
	dateDeleted := getCurrentTimestamp()
	stdout, stderr, errdel := execDeleteBackup(pluginConfig.ExecutablePath, deleteBackupPluginCommand, backupDeletePluginConfigFile, backupData.Timestamp)
	if len(stderr) > 0 {
		gplog.Error(stderr)
	}
	if errdel != nil {
		err = parseHData.UpdateBackupConfigDateDeleted(backupData.Timestamp, gpbckpconfig.DateDeletedPluginFailed)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableSetBackupStatus(gpbckpconfig.DateDeletedPluginFailed, backupData.Timestamp, err))
		}
		gplog.Error(textmsg.ErrorTextUnableDeleteBackup(backupData.Timestamp, errdel))
		return errdel
	}
	gplog.Info(stdout)
	err = parseHData.UpdateBackupConfigDateDeleted(backupData.Timestamp, dateDeleted)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableSetBackupStatus(dateDeleted, backupData.Timestamp, err))
		return err
	}
	gplog.Info(textmsg.InfoTextBackupDeleteSuccess(backupData.Timestamp))
	return nil
}

// TODO
func backupDeleteDBLocal() {
	gplog.Warn("The functionality is still in development")
}

// TODO
func backupDeleteFileLocal() {
	gplog.Warn("The functionality is still in development")
}

func execDeleteBackup(executablePath, deleteBackupPluginCommand, pluginConfigFile, timestamp string) (string, string, error) {
	cmd := execCommand(executablePath, deleteBackupPluginCommand, pluginConfigFile, timestamp)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func updateDeleteStatus(timestamp, deleteStatus string, historyDB *sql.DB) error {
	tx, _ := historyDB.Begin()
	_, err := tx.Exec(`UPDATE backups
		SET date_deleted = ?
		WHERE timestamp = ?;`,
		deleteStatus, timestamp)
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Commit()
	return err
}

// The backup can be deleted in one of the cases:
// - backup has success status and backup is active
// - backup has success status, not active, but the --force flag is set.
// Returns:
// - true, if backup can be deleted;
// - false, if backup can't be deleted.
// Errors and warnings will also be logged.
func checkBackupCanBeDeleted(backupData gpbckpconfig.BackupConfig) bool {
	result := false
	backupSuccessStatus, err := backupData.IsSuccess()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupData.Timestamp, err))
		// There is no point in performing further checks.
		return result
	}
	if !backupSuccessStatus {
		gplog.Warn(textmsg.WarnTextBackupUnableDeleteFailed(backupData.Timestamp))
		return result
	}
	backupDateDeleted, errDateDeleted := backupData.GetBackupDateDeleted()
	if errDateDeleted != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("date deletion", backupData.Timestamp, errDateDeleted))
	}
	// If the backup date deletion has invalid value, try to delete the backup.
	if gpbckpconfig.IsBackupActive(backupDateDeleted) || errDateDeleted != nil {
		result = true
	} else {
		if backupDateDeleted == gpbckpconfig.DateDeletedInProgress {
			gplog.Warn(textmsg.ErrorTextBackupInProgress(backupData.Timestamp, textmsg.ErrorBackupDeleteInProgressError()))
		} else {
			gplog.Warn(textmsg.WarnTextBackupAlreadyDeleted(backupData.Timestamp))
		}
	}
	// If flag --force is set.
	if backupDeleteForce {
		result = true
	}
	return result
}
