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
		doDeleteBackupFlagValidation(cmd.Flags())
		doDeleteBackup()
	},
}

var execCommand = exec.Command

func init() {
	rootCmd.AddCommand(backupDeleteCmd)
	backupDeleteCmd.PersistentFlags().StringArrayVar(
		&backupDeleteTimestamp,
		timestampFlagName,
		[]string{""},
		"the backup timestamp for deleting, could be specified multiple times",
	)
	backupDeleteCmd.PersistentFlags().StringVar(
		&backupDeletePluginConfigFile,
		pluginConfigFileFlagName,
		"",
		"the full path to plugin config file",
	)
	backupDeleteCmd.PersistentFlags().BoolVar(
		&backupDeleteCascade,
		cascadeFlagName,
		false,
		"delete all dependent backups for the specified backup timestamp",
	)
	backupDeleteCmd.PersistentFlags().BoolVar(
		&backupDeleteForce,
		forceFlagName,
		false,
		"try to delete, even if the backup already mark as deleted",
	)
	backupDeleteCmd.MarkPersistentFlagRequired(timestampFlagName)
}

// These flag checks are applied only for backup-delete command.
func doDeleteBackupFlagValidation(flags *pflag.FlagSet) {
	var err error
	// If timestamps are specified and have correct values.
	if flags.Changed(timestampFlagName) {
		for _, timestamp := range backupDeleteTimestamp {
			err = gpbckpconfig.CheckTimestamp(timestamp)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableValidateFlag(timestamp, timestampFlagName, err))
				execOSExit(exitErrorCode)
			}
		}
	}
	// If plugin-config flag is specified and full path.
	if flags.Changed(pluginConfigFileFlagName) {
		err = gpbckpconfig.CheckFullPath(backupDeletePluginConfigFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(backupDeletePluginConfigFile, pluginConfigFileFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
}

func doDeleteBackup() {
	logHeadersDebug()
	err := deleteBackup()
	if err != nil {
		execOSExit(exitErrorCode)
	}
}

func deleteBackup() error {
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
		if len(backupDeletePluginConfigFile) > 0 {
			pluginConfig, err := utils.ReadPluginConfig(backupDeletePluginConfigFile)
			if err != nil {
				return err
			}
			err = backupDeleteDBPlugin(backupDeleteTimestamp, backupDeleteCascade, pluginConfig, hDB)
			if err != nil {
				return err
			}
		} else {
			err := backupDeleteDBLocal()
			if err != nil {
				return err
			}
		}
	} else {
		if len(backupDeletePluginConfigFile) > 0 {
			pluginConfig, err := utils.ReadPluginConfig(backupDeletePluginConfigFile)
			if err != nil {
				return err
			}
			err = backupDeleteFilePlugin(backupDeleteTimestamp, backupDeleteCascade, pluginConfig)
			if err != nil {
				return err
			}
		} else {
			err := backupDeleteFileLocal()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func backupDeleteDBPlugin(backupListForDeletion []string, deleteCascade bool, pluginConfig *utils.PluginConfig, hDB *sql.DB) error {
	for _, backupName := range backupListForDeletion {
		backupData, err := gpbckpconfig.GetBackupDataDB(backupName, hDB)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
			return err
		}
		gplog.Info(textmsg.InfoTextBackupDeleteStart(backupName))
		canBeDeleted, err := checkBackupCanBeDeleted(backupData)
		if err != nil {
			return err
		}
		if canBeDeleted {
			backupDependencies, err := gpbckpconfig.GetBackupDependencies(backupName, hDB)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableGetBackupValue("dependencies", backupName, err))
				return err
			}
			if len(backupDependencies) > 0 {
				gplog.Info(textmsg.InfoTextBackupDependenciesList(backupName, backupDependencies))
				if deleteCascade {
					gplog.Debug(textmsg.InfoTextBackupDeleteList(backupDependencies))
					// If the deletion of at least one dependent backup fails, we fail full entire chain.
					err = backupDeleteDBCascade(backupDependencies, hDB, pluginConfig)
					if err != nil {
						gplog.Error(textmsg.ErrorTextUnableDeleteBackupCascade(backupName, err))
						return err
					}
				} else {
					gplog.Error(textmsg.ErrorTextUnableDeleteBackupUseCascade(backupName, textmsg.ErrorBackupDeleteCascadeOptionError()))
					return textmsg.ErrorBackupDeleteCascadeOptionError()
				}
			}
			err = backupDeleteDBPluginFunc(backupName, pluginConfig, hDB)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func backupDeleteDBCascade(backupList []string, hDB *sql.DB, pluginConfig *utils.PluginConfig) error {
	for _, backup := range backupList {
		gplog.Info(textmsg.InfoTextBackupDeleteStart(backup))
		backupData, err := gpbckpconfig.GetBackupDataDB(backup, hDB)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backup, err))
			return err
		}
		canBeDeleted, err := checkBackupCanBeDeleted(backupData)
		if err != nil {
			return err
		}
		if canBeDeleted {
			err = backupDeleteDBPluginFunc(backup, pluginConfig, hDB)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func backupDeleteDBPluginFunc(backupName string, pluginConfig *utils.PluginConfig, hDB *sql.DB) error {
	var err error
	dateDeleted := getCurrentTimestamp()
	err = updateDeleteStatus(backupName, gpbckpconfig.DateDeletedInProgress, hDB)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableSetBackupStatus(gpbckpconfig.DateDeletedInProgress, backupName, err))
		return err
	}
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
		return err
	}
	gplog.Info(textmsg.InfoTextBackupDeleteSuccess(backupName))
	return nil
}

func backupDeleteFilePlugin(backupListForDeletion []string, deleteCascade bool, pluginConfig *utils.PluginConfig) error {
	for _, historyFile := range rootHistoryFiles {
		hFile := getHistoryFilePath(historyFile)
		historyData, err := gpbckpconfig.ReadHistoryFile(hFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("read", err))
			return err
		}
		parseHData, err := gpbckpconfig.ParseResult(historyData)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("parse", err))
			return err
		}
		if len(parseHData.BackupConfigs) != 0 {
			for _, backupName := range backupListForDeletion {
				backupPositionInHistoryFile, backupData, err := parseHData.FindBackupConfig(backupName)
				if err != nil {
					gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
					return err
				}
				gplog.Info(textmsg.InfoTextBackupDeleteStart(backupName))
				canBeDeleted, err := checkBackupCanBeDeleted(backupData)
				if err != nil {
					return err
				}
				if canBeDeleted {
					backupDependencies := parseHData.FindBackupConfigDependencies(backupName, backupPositionInHistoryFile)
					if len(backupDependencies) > 0 {
						gplog.Info(textmsg.InfoTextBackupDependenciesList(backupName, backupDependencies))
						if deleteCascade {
							gplog.Debug(textmsg.InfoTextBackupDeleteList(backupDependencies))
							// If the deletion of at least one dependent backup fails, we fail full entire chain.
							err = backupDeleteFileCascade(backupDependencies, &parseHData, pluginConfig)
							if err != nil {
								gplog.Error(textmsg.ErrorTextUnableDeleteBackupCascade(backupName, err))
								return err
							}
						} else {
							gplog.Error(textmsg.ErrorTextUnableDeleteBackupUseCascade(backupName, textmsg.ErrorBackupDeleteCascadeOptionError()))
							return textmsg.ErrorBackupDeleteCascadeOptionError()
						}
					}
					err = backupDeleteFilePluginFunc(backupData, &parseHData, pluginConfig)
					if err != nil {
						return err
					}
				}
			}
		}
		errUpdateHFile := parseHData.UpdateHistoryFile(hFile)
		if errUpdateHFile != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("update", errUpdateHFile))
			return errUpdateHFile
		}
	}
	return nil
}

func backupDeleteFileCascade(backupList []string, parseHData *gpbckpconfig.History, pluginConfig *utils.PluginConfig) error {
	for _, backup := range backupList {
		gplog.Info(textmsg.InfoTextBackupDeleteStart(backup))
		_, backupData, err := parseHData.FindBackupConfig(backup)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backup, err))
			return err
		}
		canBeDeleted, err := checkBackupCanBeDeleted(backupData)
		if err != nil {
			return err
		}
		if canBeDeleted {
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
	backupName := backupData.Timestamp
	dateDeleted := getCurrentTimestamp()
	err = parseHData.UpdateBackupConfigDateDeleted(backupName, gpbckpconfig.DateDeletedInProgress)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableSetBackupStatus(gpbckpconfig.DateDeletedInProgress, backupName, err))
		return err
	}
	stdout, stderr, errdel := execDeleteBackup(pluginConfig.ExecutablePath, deleteBackupPluginCommand, backupDeletePluginConfigFile, backupName)
	if len(stderr) > 0 {
		gplog.Error(stderr)
	}
	if errdel != nil {
		err = parseHData.UpdateBackupConfigDateDeleted(backupName, gpbckpconfig.DateDeletedPluginFailed)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableSetBackupStatus(gpbckpconfig.DateDeletedPluginFailed, backupName, err))
		}
		gplog.Error(textmsg.ErrorTextUnableDeleteBackup(backupName, errdel))
		return errdel
	}
	gplog.Info(stdout)
	err = parseHData.UpdateBackupConfigDateDeleted(backupName, dateDeleted)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableSetBackupStatus(dateDeleted, backupName, err))
		return err
	}
	gplog.Info(textmsg.InfoTextBackupDeleteSuccess(backupName))
	return nil
}

// TODO
func backupDeleteDBLocal() error {
	gplog.Warn("The functionality is still in development")
	return nil
}

// TODO
func backupDeleteFileLocal() error {
	gplog.Warn("The functionality is still in development")
	return nil
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
// Errors and warnings will also returned and logged.
func checkBackupCanBeDeleted(backupData gpbckpconfig.BackupConfig) (bool, error) {
	result := false
	backupSuccessStatus, err := backupData.IsSuccess()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("status", backupData.Timestamp, err))
		// There is no point in performing further checks.
		return result, err
	}
	if !backupSuccessStatus {
		gplog.Warn(textmsg.WarnTextBackupFailedStatus(backupData.Timestamp))
		return result, nil
	}
	// Checks, if this is local backup.
	// In this case	the backup can't be deleted.
	// TODO
	// The same check for backup, which was done with the storage plugin,
	// but the storage plugin is not specified. And the deletion is set as local.
	// Now it is only necessary to check whether the backup is in the local storage.
	if backupData.IsLocal() {
		gplog.Error(textmsg.ErrorTextUnableDeleteBackup(backupData.Timestamp, textmsg.ErrorBackupLocalStorageError()))
		return result, textmsg.ErrorBackupLocalStorageError()
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
			gplog.Warn(textmsg.ErrorTextBackupDeleteInProgress(backupData.Timestamp, textmsg.ErrorBackupDeleteInProgressError()))
		} else {
			gplog.Warn(textmsg.WarnTextBackupAlreadyDeleted(backupData.Timestamp))
		}
	}
	// If flag --force is set.
	if backupDeleteForce {
		result = true
	}
	return result, nil
}
