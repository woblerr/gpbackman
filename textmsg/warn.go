package textmsg

import "fmt"

func WarnTextBackupUnableGetReport(backupName string) string {
	return fmt.Sprintf("Unable to get report for backup %s. Check if backup is active", backupName)
}

func WarnTextHistoryStandbySyncFailed(err error) string {
	return fmt.Sprintf("History db sync to standby coordinator failed; standby history may be stale: %v", err)
}
