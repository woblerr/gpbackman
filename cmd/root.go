package cmd

import (
	"fmt"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/errtext"
)

const (
	rootHistoryDBFlagName       = "history-db"
	rootHistoryFilesFlagName    = "history-file"
	rootLogFileFlagName         = "log-file"
	rootLogLevelConsoleFlagName = "log-level-console"
	rootLogLevelFileFlagName    = "log-level-file"
)

// Flags for the gpbackman command (rootCmd)
var (
	rootHistoryFiles    []string
	rootHistoryDB       string
	rootLogFile         string
	rootLogLevelConsole string
	rootLogLevelFile    string
)

var rootCmd = &cobra.Command{
	Use:   commandName,
	Short: "gpBackMan - utility for managing backups created by gpbackup",
	Args:  cobra.NoArgs,
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&rootHistoryDB,
		rootHistoryDBFlagName,
		"",
		"full path to the gpbackup_history.db file",
	)
	rootCmd.PersistentFlags().StringArrayVar(
		&rootHistoryFiles,
		rootHistoryFilesFlagName,
		[]string{""},
		"full path to the gpbackup_history.yaml file, could be specified multiple times",
	)
	rootCmd.PersistentFlags().StringVar(
		&rootLogFile,
		rootLogFileFlagName,
		"",
		"full path to log file directory, if not specified, the log file will be created in the $HOME/gpAdminLogs directory",
	)
	rootCmd.PersistentFlags().StringVar(
		&rootLogLevelConsole,
		rootLogLevelConsoleFlagName,
		"info",
		"level for console logging (error, info, debug, verbose)",
	)
	rootCmd.PersistentFlags().StringVar(
		&rootLogLevelFile,
		rootLogLevelFileFlagName,
		"info",
		"level for file logging (error, info, debug, verbose)",
	)
}

func doInit(version string) {
	setVersion(version)
	// If log-file flag is specified the log file will be created in the specified directory
	gplog.InitializeLogging(commandName, rootLogFile)
}

func setVersion(version string) {
	rootCmd.Version = version
}

func getVersion() string {
	return rootCmd.Version
}

// These flag checks are applied for all commands:
func doRootFlagValidation(flags *pflag.FlagSet) {
	var err error
	// If history-db flag is specified and full path.
	if flags.Changed(rootHistoryDBFlagName) {
		err = checkFullPath(rootHistoryDB)
		if err != nil {
			gplog.Error(errtext.ErrorTextUnableValidateFlag(rootHistoryDB, rootHistoryDBFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
	// If history-file flag is specified and full path.
	if flags.Changed(rootHistoryFilesFlagName) {
		for _, hFile := range rootHistoryFiles {
			err = checkFullPath(hFile)
			if err != nil {
				gplog.Error(errtext.ErrorTextUnableValidateFlag(hFile, rootHistoryFilesFlagName, err))
				execOSExit(exitErrorCode)
			}
		}
	}
	// Check, that the log level is correct.
	err = setLogLevelConsole(rootLogLevelConsole)
	if err != nil {
		gplog.Error(errtext.ErrorTextUnableValidateFlag(rootLogLevelConsole, rootLogLevelConsoleFlagName, err))
		execOSExit(exitErrorCode)
	}
	err = setLogLevelFile(rootLogLevelFile)
	if err != nil {
		gplog.Error(errtext.ErrorTextUnableValidateFlag(rootLogLevelFile, rootLogLevelFileFlagName, err))
		execOSExit(exitErrorCode)
	}
}

// These flag checks are applied only to commands:
// - backup-info
// - backup-delete
func doRootBackupFlagValidation(flags *pflag.FlagSet) {
	// history-file flag and history-db flags cannot be used together for backup-info and backup-delete commands.
	err := checkCompatibleFlags(flags, rootHistoryDBFlagName, rootHistoryFilesFlagName)
	if err != nil {
		gplog.Error(errtext.ErrorTextUnableCompatibleFlags(err, rootHistoryDBFlagName, rootHistoryFilesFlagName))
		execOSExit(exitErrorCode)
	}
	// If history-files flag is specified, set historyDB = false.
	// It's file format for history database.
	if flags.Changed(rootHistoryFilesFlagName) && !flags.Changed(rootHistoryDBFlagName) {
		historyDB = false
	}
}

func Execute(version string) {
	doInit(version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		execOSExit(exitErrorCode)
	}
}
