package textmsg

import (
	"errors"
	"fmt"
	"strings"
)

// Collection of possible error texts.

// Errors that occur when working with a history db.

func ErrorTextUnableOpenHistoryDB(err error) string {
	return fmt.Sprintf("Unable to open history db. Error: %v", err)
}

func ErrorTextUnableReadHistoryDB(err error) string {
	return fmt.Sprintf("Unable to read data from history db. Error: %v", err)
}

func ErrorTextUnableWriteIntoHistoryDB(err error) string {
	return fmt.Sprintf("Unable to write into history db. Error: %v", err)
}

func ErrorTextUnableInitHistoryDB(err error) string {
	return fmt.Sprintf("Unable to initialize history db. Error: %v", err)
}

// Errors that occur when working with a history db.

func ErrorTextUnableActionHistoryFile(value string, err error) string {
	return fmt.Sprintf("Unable to %s history file. Error: %v", value, err)
}

// Errors that occur when working with a backup data.

func ErrorTextUnableGetBackupInfo(backupName string, err error) string {
	return fmt.Sprintf("Unable to get info for backup %s. Error: %v", backupName, err)
}

func ErrorTextUnableGetBackupValue(value, backupName string, err error) string {
	return fmt.Sprintf("Unable to get backup %s for backup %s. Error: %v", value, backupName, err)
}

func ErrorTextUnableSetBackupStatus(value, backupName string, err error) string {
	return fmt.Sprintf("Unable to set %s status for backup %s. Error: %v", value, backupName, err)
}

func ErrorTextUnableDeleteBackup(backupName string, err error) string {
	return fmt.Sprintf("Unable to delete backup %s. Error: %v", backupName, err)
}

func ErrorTextUnableDeleteBackupCascade(backupName string, err error) string {
	return fmt.Sprintf("Unable to delete dependent backups for backup %s. Error: %v", backupName, err)
}

func ErrorTextUnableDeleteBackupUseCascade(backupName string, err error) string {
	return fmt.Sprintf("Backup %s has dependent backups. Use --cascade option. Error: %v", backupName, err)
}

func ErrorTextBackupInProgress(backupName string, err error) string {
	return fmt.Sprintf("Backup %s in progress. Wait for the backup to finish or check actual status manually. Error: %v", backupName, err)
}

// Errors that occur when working with a backup plugin.

func ErrorTextUnableReadPluginConfigFile(err error) string {
	return fmt.Sprintf("Unable to read plugin config file. Error: %v", err)
}

// Errors that occur during flags validation.

func ErrorTextUnableValidateFlag(value, flag string, err error) string {
	return fmt.Sprintf(
		"Unable to validate value %s for flag %s. Error: %v", value, flag, err)
}

func ErrorTextUnableCompatibleFlags(err error, values ...string) string {
	return fmt.Sprintf(
		"Unable to use the following flags together: %s. Error: %v",
		strings.Join(values, ", "), err)
}

func ErrorTextUnableCompatibleFlagsValues(err error, values ...string) string {
	var result []string
	for i := 0; i < len(values); i += 2 {
		result = append(result, values[i]+"="+values[i+1])
	}
	return fmt.Sprintf(
		"Unable to use the provided values for these flags together: %s. Error: %v",
		strings.Join(result, ", "), err)
}

// Error that is returned when flags validation not passed.

func ErrorInvalidValueError() error {
	return errors.New("invalid flag value")
}

func ErrorIncompatibleValuesError() error {
	return errors.New("incompatible flags values")
}

func ErrorIncompatibleFlagsError() error {
	return errors.New("incompatible flags")
}

// Error that is returned when backup deletion fails.

func ErrorBackupDeleteCascadeError() error {
	return errors.New("delete cascade is failed")
}

func ErrorBackupDeleteInProgressError() error {
	return errors.New("backup in progress")
}

func ErrorBackupDeleteCascadeOptionError() error {
	return errors.New("use cascade option")
}

// Error that is returned when some validation fails.

func ErrorValidationFullPath() error {
	return errors.New("not an absolute path")
}

func ErrorValidationTimestamp() error {
	return errors.New("not a timestamp")
}
