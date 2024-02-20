package gpbckpconfig

import (
	"os"

	"gopkg.in/yaml.v2"
)

var execReadFile = os.ReadFile

// ReadHistoryFile Read history file.
func ReadHistoryFile(filename string) ([]byte, error) {
	data, err := execReadFile(filename)
	return data, err
}

// ParseResult Parse result to History struct.
func ParseResult(output []byte) (History, error) {
	var hData History
	err := yaml.Unmarshal(output, &hData)
	return hData, err
}

// CheckBackupCanBeDisplayed Returns true if the backup satisfies the parameters.
// This function is only applicable to processing values obtained from gpbackup history file.
//
// The value is calculated, based on:
//   - if showD is true, the function returns true only for backups that active or already deleted;
//   - if showF is true, the function returns true only for backups that active or failed;
//   - if none of the parameters were passed, the function returns true for backups that have a successful status and have not been deleted;
//   - If none of the above conditions are met, the function returns false.
func CheckBackupCanBeDisplayed(showD, showF bool, status, dateDeleted string) bool {
	switch {
	// Displaying all backups (active, deleted, failed)
	case showD && showF:
		return true
	// Displaying only active and deleted backups; failed - hidden.
	case showD && !showF:
		if status != BackupStatusFailure {
			return true
		}
	// Displaying only active and failed backups; deleted - hidden.
	case !showD && showF:
		if IsBackupActive(dateDeleted) {
			return true
		}
	// Displaying only active backups or backups with deletion status "In progress", deleted and failed - hidden.
	default:
		if status != BackupStatusFailure && (IsBackupActive(dateDeleted) || dateDeleted == DateDeletedInProgress) {
			return true
		}
	}
	return false
}
