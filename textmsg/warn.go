package textmsg

import "fmt"

func WarnTextBackupAlreadyDeleted(backupName string) string {
	return fmt.Sprintf("Backup %s has already been deleted", backupName)
}

func WarnTextBackupFailedStatus(backupName string) string {
	return fmt.Sprintf("Backup %s has failed status. Nothing to do", backupName)
}

func WarnTextBackupUnableGetReport(backupName string) string {
	return fmt.Sprintf("Unable to get report for backup %s. Check if backup is active", backupName)
}
