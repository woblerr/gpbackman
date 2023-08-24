package textmsg

import "fmt"

func WarnTextBackupAlreadyDeleted(backupName string) string {
	return fmt.Sprintf("Backup %s has already been deleted", backupName)
}

func WarnTextBackupUnableDeleteFailed(backupName string) string {
	return fmt.Sprintf("Backup %s has failed status. Nothing to delete", backupName)
}
