package cmd

import (
	"bytes"
	"fmt"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpbackup/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

const (
	reportInfoTimestampFlagName            = "timestamp"
	reportInfoPluginConfigFileFlagName     = "plugin-config"
	reportInfoReportFilePluginPathFlagName = "plugin-report-file-path"
)

// Flags for the gpbackman report-info command (reportInfoCmd)
var (
	reportInfoTimestamp            string
	reportInfoPluginConfigFile     string
	reportInfoReportFilePluginPath string
)

var reportInfoCmd = &cobra.Command{
	Use:   "report-info",
	Short: "Display the report for specific backup set",
	Long: `Display the report for specific backup set.

The --timestamp option must be specified.

The report could be displayed only for active backups.

The storage plugin config file location can be set using the --plugin-config option.
The full path to the file is required. In this case, the deletion will be performed using the storage plugin.

If a custom plugin is used, it is required to specify the path to the directory with the repo file using the --plugin-report-file-path option.
It is not necessary to use the --plugin-report-file-path flag for the following plugins (the path is generated automatically):
  * gpbackup_s3_plugin,

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
		doReportInfoFlagValidation(cmd.Flags())
		doReportInfo()
	},
}

func init() {
	rootCmd.AddCommand(reportInfoCmd)
	reportInfoCmd.PersistentFlags().StringVar(
		&reportInfoTimestamp,
		reportInfoTimestampFlagName,
		"",
		"the backup timestamp for report displaying",
	)
	reportInfoCmd.PersistentFlags().StringVar(
		&reportInfoPluginConfigFile,
		reportInfoPluginConfigFileFlagName,
		"",
		"the full path to plugin config file",
	)
	reportInfoCmd.PersistentFlags().StringVar(
		&reportInfoReportFilePluginPath,
		reportInfoReportFilePluginPathFlagName,
		"",
		"the full path to plugin report file",
	)
	reportInfoCmd.MarkPersistentFlagRequired(reportInfoTimestampFlagName)
}

// These flag checks are applied only for report-info command.
func doReportInfoFlagValidation(flags *pflag.FlagSet) {
	var err error
	// If timestamps are specified and have correct values.
	if flags.Changed(reportInfoTimestampFlagName) {
		err = gpbckpconfig.CheckTimestamp(reportInfoTimestamp)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(reportInfoTimestamp, reportInfoTimestampFlagName, err))
			execOSExit(exitErrorCode)
		}

	}
	// If plugin-config flag is specified and full path.
	if flags.Changed(reportInfoPluginConfigFileFlagName) {
		err = gpbckpconfig.CheckFullPath(reportInfoPluginConfigFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(reportInfoPluginConfigFile, reportInfoPluginConfigFileFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
	// If plugin-report-file-pat flag is specified and full path.
	if flags.Changed(reportInfoReportFilePluginPathFlagName) {
		err = gpbckpconfig.CheckFullPath(reportInfoReportFilePluginPath)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(reportInfoReportFilePluginPath, reportInfoReportFilePluginPathFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
}

func doReportInfo() {
	logHeadersDebug()
	if len(reportInfoPluginConfigFile) > 0 {
		pluginConfig, err := utils.ReadPluginConfig(reportInfoPluginConfigFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableReadPluginConfigFile(err))
			execOSExit(exitErrorCode)
		}
		if historyDB {
			reportInfoDBPlugin(pluginConfig)
		} else {
			reportInfoFilePlugin(pluginConfig)
		}
	} else {
		if historyDB {
			reportInfoDBLocal()
		} else {
			reportInfoFileLocal()
		}
	}
}

func reportInfoDBPlugin(pluginConfig *utils.PluginConfig) {
	hDB, err := gpbckpconfig.OpenHistoryDB(getHistoryDBPath(rootHistoryDB))
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableOpenHistoryDB(err))
		execOSExit(exitErrorCode)
	}
	backupName := reportInfoTimestamp
	backupData, err := gpbckpconfig.GetBackupDataDB(backupName, hDB)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
	}
	if checkBackupCanGetReport(backupData) {
		reportInfoPluginFunc(backupData, pluginConfig)
	}
	hDB.Close()
}

// TODO
func reportInfoFilePlugin(pluginConfig *utils.PluginConfig) {
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
			backupName := reportInfoTimestamp
			_, backupData, err := parseHData.FindBackupConfig(backupName)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
				continue
			}
			if checkBackupCanGetReport(backupData) {
				reportInfoPluginFunc(backupData, pluginConfig)
			}
		}
	}
}

func reportInfoPluginFunc(backupData gpbckpconfig.BackupConfig, pluginConfig *utils.PluginConfig) {
	reportFile, err := backupData.GetReportFilePathPlugin(reportInfoReportFilePluginPath, pluginConfig.Options)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupReportPath(backupData.Timestamp, err))
	}
	stdout, stderr, err := execReportInfo(pluginConfig.ExecutablePath, restoreDataPluginCommand, reportInfoPluginConfigFile, reportFile)
	if len(stderr) > 0 {
		gplog.Error(stderr)
	}
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupReport(backupData.Timestamp, err))
	}
	// Display the report.
	fmt.Println(stdout)
}

// TODO
func reportInfoDBLocal() {
	gplog.Warn("The functionality is still in development")
}

// TODO
func reportInfoFileLocal() {
	gplog.Warn("The functionality is still in development")
}

// Report could be displayed only for active backups:
// - backup has success status and backup is active
// Returns:
// - true, if report can be displayed;
// - false, if report can't be displayed.
// Errors and warnings will also be logged.
func checkBackupCanGetReport(backupData gpbckpconfig.BackupConfig) bool {
	result := false
	backupSuccessStatus, err := backupData.IsSuccess()
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupValue("status", backupData.Timestamp, err))
		return result
	}
	if !backupSuccessStatus {
		gplog.Warn(textmsg.WarnTextBackupFailedStatus(backupData.Timestamp))
		return result
	}
	// Checks, if this is local backup.
	if backupData.IsLocal() {
		gplog.Error(textmsg.ErrorTextUnableGetBackupReport(backupData.Timestamp, textmsg.ErrorBackupLocalStorageError()))
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
			gplog.Warn(textmsg.ErrorTextBackupDeleteInProgress(backupData.Timestamp, textmsg.ErrorBackupDeleteInProgressError()))
		} else {
			gplog.Warn(textmsg.WarnTextBackupAlreadyDeleted(backupData.Timestamp))
		}
	}
	return result
}

func execReportInfo(executablePath, reportInfoPluginCommand, pluginConfigFile, file string) (string, string, error) {
	cmd := execCommand(executablePath, reportInfoPluginCommand, pluginConfigFile, file)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
