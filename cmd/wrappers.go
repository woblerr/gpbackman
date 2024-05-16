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
func checkBackupCanBeUsed(deleteForce, skipLocalBackup bool, backupData gpbckpconfig.BackupConfig) (bool, error) {
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
	err = checkLocalBackupStatus(skipLocalBackup, backupData.IsLocal())
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableWorkBackup(backupData.Timestamp, err))
		return result, err

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
func checkBackupType(backupType string) error {
	var validVType = map[string]bool{
		gpbckpconfig.BackupTypeFull:         true,
		gpbckpconfig.BackupTypeIncremental:  true,
		gpbckpconfig.BackupTypeMetadataOnly: true,
		gpbckpconfig.BackupTypeDataOnly:     true,
	}
	if !validVType[backupType] {
		return textmsg.ErrorInvalidValueError()
	}
	return nil
}

// Check skip flag and local backup status.
// SkipLocalBackup - true, local backup - true, returns "is a local backup" error.
// SkipLocalBackup - false,local backup - false, returns "is not a local backup" error.
func checkLocalBackupStatus(skipLocalBackup, isLocalBackup bool) error {
	if skipLocalBackup && isLocalBackup {
		return textmsg.ErrorBackupLocalStorageError()
	}
	if !skipLocalBackup && !isLocalBackup {
		return textmsg.ErrorBackupNotLocalStorageError()
	}
	return nil
}

func getBackupMasterDir(backupDir, backupDataBackupDir, backupDataDBName string) (string, string, error) {
	if backupDir != "" {
		return gpbckpconfig.CheckSingleBackupDir(backupDir)
	}
	if backupDataBackupDir != "" {
		return gpbckpconfig.CheckSingleBackupDir(backupDataBackupDir)
	}
	// Try to get the backup directory from the cluster configuration.
	// If the script executed not on the master host, the backup directory will not be found.
	// And we return "value not set" error.
	backupDirClusterInfo := getBackupMasterDirClusterInfo(backupDataDBName)
	if backupDirClusterInfo != "" {
		return backupDirClusterInfo, gpbckpconfig.GetSegPrefix(filepath.Join(backupDirClusterInfo, "backups")), nil
	}
	return "", "", textmsg.ErrorValidationValue()
}

func getBackupMasterDirClusterInfo(dbName string) string {
	db, err := gpbckpconfig.NewClusterLocalClusterConn(dbName)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableConnectLocalCluster(err))
		return ""
	}
	defer db.Close()
	sqlQuery := "SELECT datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'p';"
	backupDir, err := gpbckpconfig.ExecuteQueryLocalClusterConn(db, sqlQuery)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupDirLocalClusterConn(err))
		return ""
	}
	gplog.Debug("Master data directory: %s", backupDir)
	return backupDir
}
