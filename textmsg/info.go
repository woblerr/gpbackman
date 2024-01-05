package textmsg

import (
	"fmt"
	"strings"
)

func InfoTextBackupDeleteStart(backupName string) string {
	return fmt.Sprintf("Start deleting backup %s", backupName)
}

func InfoTextBackupAlreadyDeleted(backupName string) string {
	return fmt.Sprintf("Backup %s has already been deleted. Nothing to do", backupName)
}

func InfoTextBackupFailedStatus(backupName string) string {
	return fmt.Sprintf("Backup %s has failed status. Nothing to do", backupName)
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

func InfoTextPluginCommandExecution(list ...string) string {
	return fmt.Sprintf("Executing Plugin command: %s", strings.Join(list, " "))
}
