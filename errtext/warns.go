package errtext

import "fmt"

func WarnTextBackupAlreadyDeleted(backupName string) string {
	return fmt.Sprintf("Backup %s has already been deleted.", backupName)
}
