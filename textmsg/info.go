package textmsg

import (
	"fmt"
	"strings"
)

func InfoTextBackupDeleteStart(backupName string) string {
	return fmt.Sprintf("Start deleting backup %s", backupName)
}

func InfoTextBackupAlreadyDeleted(backupName string) string {
	return fmt.Sprintf("Backup %s has already been deleted.", backupName)
}

func InfoTextBackupFailedStatus(backupName string) string {
	return fmt.Sprintf("Backup %s has failed status.", backupName)
}

func InfoTextBackupDeleteSuccess(backupName string) string {
	return fmt.Sprintf("Backup %s successfully deleted", backupName)
}

func InfoTextBackupDependenciesList(backupName string, list []string) string {
	return fmt.Sprintf("Backup %s has dependent backups: %s", backupName, strings.Join(list, ", "))
}

func InfoTextBackupDeleteList(list []string) string {
	return fmt.Sprintf("The following backups will be deleted: %s", strings.Join(list, ", "))
}

func InfoTextBackupDeleteListFromHistory(list []string) string {
	return fmt.Sprintf("The following backups will be deleted from history: %s", strings.Join(list, ", "))
}

func InfoTextCommandExecution(list ...string) string {
	return fmt.Sprintf("Executing command: %s", strings.Join(list, " "))
}

func InfoTextNothingToDo() string {
	return "Nothing to do"
}
