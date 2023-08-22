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

// GetBackupNameFile Returns true if the backup satisfies the parameters.
// This function is only applicable to processing values obtained from gpbackup history file.
//
// The value is calculated, based on:
//   - if sAll is true, the function returns true, implying that all backups should be shown regardless of their status and delete status;
//   - if showD is true, the function returns true only for backups that have already been deleted;
//   - if showF is true, the function returns true only for backups with the failed status;
//   - if none of the parameters were passed, the function returns true for backups that have a successful status and have not been deleted;
//   - If none of the above conditions are met, the function returns false.
func GetBackupNameFile(showD, showF, sAll bool, status, dateDeleted string) bool {
	switch {
	case sAll:
		return true
	case showD:
		if status != backupStatusFailure && !IsBackupActive(dateDeleted) {
			return true
		}
	case showF:
		if status == backupStatusFailure {
			return true
		}
	default:
		if status != backupStatusFailure && IsBackupActive(dateDeleted) {
			return true
		}
	}
	return false
}
