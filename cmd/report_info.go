package cmd

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpbackup/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

// Flags for the gpbackman report-info command (reportInfoCmd)
var (
	reportInfoTimestamp            string
	reportInfoPluginConfigFile     string
	reportInfoReportFilePluginPath string
)

var reportInfoCmd = &cobra.Command{
	Use:   "report-info",
	Short: "Display the report for a specific backup",
	Long: `Display the report for a specific backup.

The --timestamp option must be specified.

The report could be displayed only for active backups.

The storage plugin config file location can be set using the --plugin-config option.
The full path to the file is required.

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
		timestampFlagName,
		"",
		"the backup timestamp for report displaying",
	)
	reportInfoCmd.PersistentFlags().StringVar(
		&reportInfoPluginConfigFile,
		pluginConfigFileFlagName,
		"",
		"the full path to plugin config file",
	)
	reportInfoCmd.PersistentFlags().StringVar(
		&reportInfoReportFilePluginPath,
		reportFilePluginPathFlagName,
		"",
		"the full path to plugin report file",
	)
	_ = reportInfoCmd.MarkPersistentFlagRequired(timestampFlagName)
}

// These flag checks are applied only for report-info command.
func doReportInfoFlagValidation(flags *pflag.FlagSet) {
	var err error
	// If timestamps are specified and have correct values.
	if flags.Changed(timestampFlagName) {
		err = gpbckpconfig.CheckTimestamp(reportInfoTimestamp)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(reportInfoTimestamp, timestampFlagName, err))
			execOSExit(exitErrorCode)
		}

	}
	// If plugin-config flag is specified and full path.
	if flags.Changed(pluginConfigFileFlagName) {
		err = gpbckpconfig.CheckFullPath(reportInfoPluginConfigFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(reportInfoPluginConfigFile, pluginConfigFileFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
	// If plugin-report-file-pat flag is specified and full path.
	if flags.Changed(reportFilePluginPathFlagName) {
		err = gpbckpconfig.CheckFullPath(reportInfoReportFilePluginPath)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableValidateFlag(reportInfoReportFilePluginPath, reportFilePluginPathFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
}

func doReportInfo() {
	logHeadersDebug()
	err := reportInfo()
	if err != nil {
		execOSExit(exitErrorCode)
	}
}

func reportInfo() error {
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
		if reportInfoPluginConfigFile != "" {
			pluginConfig, err := utils.ReadPluginConfig(reportInfoPluginConfigFile)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableReadPluginConfigFile(err))
				return err
			}
			err = reportInfoDBPlugin(reportInfoTimestamp, reportInfoPluginConfigFile, pluginConfig, hDB)
			if err != nil {
				return err
			}
		} else {
			err := reportInfoDBLocal()
			if err != nil {
				return err
			}
		}
	} else {
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
				if reportInfoPluginConfigFile != "" {
					pluginConfig, err := utils.ReadPluginConfig(reportInfoPluginConfigFile)
					if err != nil {
						return err
					}
					err = reportInfoFilePlugin(reportInfoTimestamp, reportInfoPluginConfigFile, pluginConfig, parseHData)
					if err != nil {
						return err
					}
				} else {
					err := reportInfoFileLocal()
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func reportInfoDBPlugin(backupName, pluginConfigPath string, pluginConfig *utils.PluginConfig, hDB *sql.DB) error {
	backupData, err := gpbckpconfig.GetBackupDataDB(backupName, hDB)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
		return err
	}
	canGetReport, err := checkBackupCanBeUsed(false, backupData)
	if err != nil {
		return err
	}
	if canGetReport {
		err = reportInfoPluginFunc(backupData, pluginConfigPath, pluginConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func reportInfoFilePlugin(backupName, pluginConfigPath string, pluginConfig *utils.PluginConfig, parseHData gpbckpconfig.History) error {
	_, backupData, err := parseHData.FindBackupConfig(backupName)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
		return err
	}
	canGetReport, err := checkBackupCanBeUsed(false, backupData)
	if err != nil {
		return err
	}
	if canGetReport {
		err = reportInfoPluginFunc(backupData, pluginConfigPath, pluginConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func reportInfoPluginFunc(backupData gpbckpconfig.BackupConfig, pluginConfigPath string, pluginConfig *utils.PluginConfig) error {
	reportFile, err := backupData.GetReportFilePathPlugin(reportInfoReportFilePluginPath, pluginConfig.Options)
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupReportPath(backupData.Timestamp, err))
		return err
	}
	gplog.Debug(textmsg.InfoTextPluginCommandExecution(pluginConfig.ExecutablePath, restoreDataPluginCommand, pluginConfigPath, reportFile))
	stdout, stderr, err := execReportInfo(pluginConfig.ExecutablePath, restoreDataPluginCommand, pluginConfigPath, reportFile)
	if stderr != "" {
		gplog.Error(stderr)
	}
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableGetBackupReport(backupData.Timestamp, err))
		return err
	}
	// Display the report.
	fmt.Println(stdout)
	return nil
}

// TODO
func reportInfoDBLocal() error {
	gplog.Warn("The functionality is still in development")
	return nil

}

// TODO
func reportInfoFileLocal() error {
	gplog.Warn("The functionality is still in development")
	return nil
}

func execReportInfo(executablePath, reportInfoPluginCommand, pluginConfigFile, file string) (string, string, error) {
	cmd := execCommand(executablePath, reportInfoPluginCommand, pluginConfigFile, file)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
