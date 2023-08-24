package textmsg

import "fmt"

func InfoTextBackupDeleteSuccess(backupName string) string {
	return fmt.Sprintf("Backup %s successfully deleted.", backupName)
}
