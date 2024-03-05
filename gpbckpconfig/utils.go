package gpbckpconfig

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/woblerr/gpbackman/textmsg"
)

// CheckTimestamp Returns error if timestamp is not valid.
func CheckTimestamp(timestamp string) error {
	timestampFormat := regexp.MustCompile(`^(\d{14})$`)
	if !timestampFormat.MatchString(timestamp) {
		return textmsg.ErrorValidationTimestamp()
	}
	return nil
}

func GetTimestampOlderThen(value uint) string {
	return time.Now().AddDate(0, 0, -int(value)).Format(Layout)
}

// CheckFullPath Returns error if path is not an absolute path or
// file does not exist.
func CheckFullPath(path string, checkFileExists bool) error {
	if !filepath.IsAbs(path) {
		return textmsg.ErrorValidationFullPath()
	}
	// In most cases this check should be mandatory.
	// But there are commands, that allows the history db file to be missing.
	if checkFileExists {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return textmsg.ErrorFileNotExist()
		}
	}
	return nil
}

// CheckTableFQN Returns error if table FQN is not in the format <schema.table>.
func CheckTableFQN(table string) error {
	format := regexp.MustCompile(`^.+\..+$`)
	if !format.MatchString(table) {
		return textmsg.ErrorValidationTableFQN()
	}
	return nil
}

// IsBackupActive Returns true if backup is active (not deleted).
func IsBackupActive(dateDeleted string) bool {
	return (dateDeleted == "" ||
		dateDeleted == DateDeletedPluginFailed ||
		dateDeleted == DateDeletedLocalFailed)
}

// backupPluginCustomReportPath Returns custom report path:
//
//	<folder>/gpbackup_<YYYYMMDDHHMMSS>_report
func backupPluginCustomReportPath(timestamp, folderValue string) string {
	return filepath.Join("/", folderValue, reportFileName(timestamp))
}

// backupS3PluginReportPath Returns path to report file name for gpbackup_s3_plugin plugin.
// Basic path for s3 plugin format:
//
//	<folder>/backups/<YYYYMMDD>/<YYYYMMDDHHMMSS>/gpbackup_<YYYYMMDDHHMMSS>_report
//
// See GetS3Path() func in https://github.com/greenplum-db/gpbackup-s3-plugin.
// If folder option is not specified or it is empty, the error will be returned.
func backupS3PluginReportPath(timestamp string, pluginOptions map[string]string) (string, error) {
	pathOption := "folder"
	// Timestamp validation is done on flags validation.
	// We assume, that is the correct value coming from.
	reportPathBasic := "backups/" + timestamp[0:8] + "/" + timestamp
	folderValue, exists := pluginOptions[pathOption]
	if !exists || folderValue == "" {
		return "", textmsg.ErrorValidationPluginOption(pathOption, BackupS3Plugin)
	}
	// It's necessary to return full path to report file with leading '/'.
	// But in config file folder value could be with leading '/' or without.
	// So we need to remove leading '/' and add it back.
	folderValue = strings.TrimPrefix(folderValue, "/")
	folderValue = strings.TrimSuffix(folderValue, "/")
	return filepath.Join("/", folderValue, reportPathBasic, reportFileName(timestamp)), nil
}

// reportFileName Returns report file name for specific timestamp.
// Report file name format: gpbackup_<YYYYMMDDHHMMSS>_report.
func reportFileName(timestamp string) string {
	return "gpbackup_" + timestamp + "_report"
}

// searchFilter returns true if the value is present in the list
func searchFilter(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
