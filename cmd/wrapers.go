package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpbackup/history"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/errtext"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

var execOSExit = os.Exit

func logHeadersInfo() {
	gplog.Info("Start %s version %s", commandName, getVersion())
	gplog.Info("Use console log level: %s", rootLogLevelConsole)
	gplog.Info("Use file log level: %s", rootLogLevelFile)
}

func logHeadersDebug() {
	gplog.Verbose("%s command: %s", commandName, os.Args)
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
		return errtext.ErrorInvalidValueError()
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
		return errtext.ErrorInvalidValueError()
	}
	return nil
}

func openHistoryDB(historyDBPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", historyDBPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getHistoryFilePath(historyFilePath string) string {
	var historyFileName = historyFileNameConst
	if historyFilePath != "" {
		return historyFilePath
	}
	return historyFileName
}

func getHistoryDBPath(historyDBPath string) string {
	var historyDBName = historyDBNameConst
	if historyDBPath != "" {
		return historyDBPath
	}
	return historyDBName
}

func getBackupDataDB(backupName string, hDB *sql.DB) (gpbckpconfig.BackupConfig, error) {
	hBackupData, err := history.GetMainBackupInfo(backupName, hDB)
	if err != nil {
		return gpbckpconfig.BackupConfig{}, err
	}
	return gpbckpconfig.ConvertFromHistoryBackupConfig(hBackupData), nil
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

func isBackupActive(dateDeleted string) bool {
	return (dateDeleted != "" || dateDeleted == deleteStatusPluginFailed || dateDeleted == deleteStatusLocalFailed)
}

func checkFullPath(path string) error {
	if len(path) > 0 && !filepath.IsAbs(path) {
		return errtext.ErrorValidationFullPath()
	}
	return nil
}

func checkTimestamp(timestamp string) error {
	timestampFormat := regexp.MustCompile(`^([0-9]{14})$`)
	if !timestampFormat.MatchString(timestamp) {
		return errtext.ErrorValidationTimestamp()
	}
	return nil
}

func checkCompatibleFlags(flags *pflag.FlagSet, flagNames ...string) error {
	n := 0
	for _, name := range flagNames {
		if flags.Changed(name) {
			n++
		}
	}
	if n > 1 {
		return errtext.ErrorIncompatibleFlagsError()
	}
	return nil
}
