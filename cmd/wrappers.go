package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
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
		return errtext.ErrorIncompatibleFlagsError()
	}
	return nil
}

func formatBackupDuration(value float64) string {
	hours := int(value / 3600)
	minutes := (int(value) % 3600) / 60
	seconds := int(value) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
