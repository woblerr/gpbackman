package cmd

import (
	"fmt"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

// Flags for the gpbackman command (rootCmd)
var (
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
		historyDBFlagName,
		"",
		"full path to the gpbackup_history.db file",
	)
	rootCmd.PersistentFlags().StringVar(
		&rootLogFile,
		logFileFlagName,
		"",
		"full path to log file directory, if not specified, the log file will be created in the $HOME/gpAdminLogs directory",
	)
	rootCmd.PersistentFlags().StringVar(
		&rootLogLevelConsole,
		logLevelConsoleFlagName,
		"info",
		"level for console logging (error, info, debug, verbose)",
	)
	rootCmd.PersistentFlags().StringVar(
		&rootLogLevelFile,
		logLevelFileFlagName,
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
func doRootFlagValidation(flags *pflag.FlagSet, checkFileExists bool) {
	var err error
	// If history-db flag is specified and full path.
	// The existence of the file is checked by condition from each specific command.
	// Not all commands (see history-migrate command, report-info command flags) require a history db file to exist.
	if flags.Changed(historyDBFlagName) {
		err = gpbckpconfig.CheckFullPath(rootHistoryDB, checkFileExists)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(rootHistoryDB, historyDBFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
	// Check, that the log level is correct.
	err = setLogLevelConsole(rootLogLevelConsole)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableValidateFlag(rootLogLevelConsole, logLevelConsoleFlagName, err))
		execOSExit(exitErrorCode)
	}
	err = setLogLevelFile(rootLogLevelFile)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableValidateFlag(rootLogLevelFile, logLevelFileFlagName, err))
		execOSExit(exitErrorCode)
	}
}

func Execute(version string) {
	doInit(version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		execOSExit(exitErrorCode)
	}
}
