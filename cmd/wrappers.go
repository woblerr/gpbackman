package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

var execOSExit = os.Exit

func logHeadersDebug() {
	gplog.Debug("Start %s version %s", commandName, getVersion())
	gplog.Debug("Use console log level: %s", rootLogLevelConsole)
	gplog.Debug("Use file log level: %s", rootLogLevelFile)
	gplog.Debug("%s command: %s", commandName, os.Args)
}

// Sets the log levels for the console and file loggers.
// Uppercase or lowercase letters are accepted.
// If an incorrect value is specified, an error is returned.
func setLogLevelConsole(level string) error {
	switch strings.ToLower(level) {
	case "info":
		gplog.SetVerbosity(gplog.LOGINFO)
	case "error":
		gplog.SetVerbosity(gplog.LOGERROR)
	case "debug":
		gplog.SetVerbosity(gplog.LOGDEBUG)
	case "verbose":
		gplog.SetVerbosity(gplog.LOGVERBOSE)
	default:
		return textmsg.ErrorInvalidValueError()
	}
	return nil
}

// Sets the log levels for the console and file loggers.
// Uppercase or lowercase letters are accepted.
// If an incorrect value is specified, an error is returned.
func setLogLevelFile(level string) error {
	switch strings.ToLower(level) {
	case "info":
		gplog.SetLogFileVerbosity(gplog.LOGINFO)
	case "error":
		gplog.SetLogFileVerbosity(gplog.LOGERROR)
	case "debug":
		gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
	case "verbose":
		gplog.SetLogFileVerbosity(gplog.LOGVERBOSE)
	default:
		return textmsg.ErrorInvalidValueError()
	}
	return nil
}

func getHistoryDBPath(historyDBPath string) string {
	var historyDBName = historyDBNameConst
	if historyDBPath != "" {
		return historyDBPath
	}
	return historyDBName
}

func getHistoryFilePath(historyFilePath string) string {
	var historyFileName = historyFileNameConst
	if historyFilePath != "" {
		return historyFilePath
	}
	return historyFileName
}

func getDataFromHistoryFile(historyFile string) (gpbckpconfig.History, error) {
	var hData gpbckpconfig.History
	historyData, err := gpbckpconfig.ReadHistoryFile(historyFile)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("read", err))
		return hData, err
	}
	hData, err = gpbckpconfig.ParseResult(historyData)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("parse", err))
		return hData, err
	}
	return hData, nil
}

func renameHistoryFile(filename string) error {
	fileDir := filepath.Dir(filename)
	fileName := filepath.Base(filename)
	newFileName := fileName + historyFileNameMigratedSuffixConst
	newPath := filepath.Join(fileDir, newFileName)
	err := os.Rename(filename, newPath)
	if err != nil {
		return err
	}
	return nil
}

func getCurrentTimestamp() string {
	return time.Now().Format(gpbckpconfig.Layout)
}

func checkCompatibleFlags(flags *pflag.FlagSet, flagNames ...string) error {
	n := 0
	for _, name := range flagNames {
		if flags.Changed(name) {
			n++
		}
	}
	if n > 1 {
		return textmsg.ErrorIncompatibleFlagsError()
	}
	return nil
}

func formatBackupDuration(value float64) string {
	hours := int(value / 3600)
	minutes := (int(value) % 3600) / 60
	seconds := int(value) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// The backup can be used in one of the cases:
// - backup has success status and backup is active
// - backup has success status, not active, but the --force flag is set.
// Returns:
// - true, if backup can be used;
// - false, if backup can't be used.
// Errors and warnings will also returned and logged.
func checkBackupCanBeUsed(deleteForce bool, backupData gpbckpconfig.BackupConfig) (bool, error) {
	result := false
	backupSuccessStatus, err := backupData.IsSuccess()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("status", backupData.Timestamp, err))
		// There is no point in performing further checks.
		return result, err
	}
	if !backupSuccessStatus {
		gplog.Warn(textmsg.InfoTextBackupFailedStatus(backupData.Timestamp))
		gplog.Info(textmsg.InfoTextNothingToDo())
		return result, nil
	}
	// Checks, if this is local backup.
	// In this case	the backup can't be deleted.
	// TODO
	// The same check for backup, which was done with the storage plugin,
	// but the storage plugin is not specified. And the deletion is set as local.
	// Now it is only necessary to check whether the backup is in the local storage.
	if backupData.IsLocal() {
		gplog.Error(textmsg.ErrorTextUnableWorkBackup(backupData.Timestamp, textmsg.ErrorBackupLocalStorageError()))
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
			// We do not return the error here,
			// because it is necessary to leave the possibility of starting the process
			// of deleting backups that are stuck in the "In Progress" status using the --force flag.
			gplog.Error(textmsg.ErrorTextBackupDeleteInProgress(backupData.Timestamp, textmsg.ErrorBackupDeleteInProgressError()))
		} else {
			gplog.Debug(textmsg.InfoTextBackupAlreadyDeleted(backupData.Timestamp))
			gplog.Debug(textmsg.InfoTextNothingToDo())
		}
	}
	// If flag --force is set.
	if deleteForce {
		result = true
	}
	return result, nil
}

// Check that specified backup type is supported.
func checkBackupType(backuType string) error {
	var validVType = map[string]bool{
		gpbckpconfig.BackupTypeFull:         true,
		gpbckpconfig.BackupTypeIncremental:  true,
		gpbckpconfig.BackupTypeMetadataOnly: true,
		gpbckpconfig.BackupTypeDataOnly:     true,
	}
	if !validVType[backuType] {
		return textmsg.ErrorInvalidValueError()
	}
	return nil
}
