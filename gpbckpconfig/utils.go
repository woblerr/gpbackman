package gpbckpconfig

import (
	"path/filepath"
	"regexp"

	"github.com/woblerr/gpbackman/textmsg"
)

// CheckTimestamp Returns error if timestamp is not valid.
func CheckTimestamp(timestamp string) error {
	timestampFormat := regexp.MustCompile(`^([0-9]{14})$`)
	if !timestampFormat.MatchString(timestamp) {
		return textmsg.ErrorValidationTimestamp()
	}
	return nil
}

// CheckFullPath Returns error if path is not full path.
func CheckFullPath(path string) error {
	if !filepath.IsAbs(path) {
		return textmsg.ErrorValidationFullPath()
	}
	return nil
}

// IsBackupActive Returns true if backup is active (not deleted).
func IsBackupActive(dateDeleted string) bool {
	return (dateDeleted == "" ||
		dateDeleted == DateDeletedPluginFailed ||
		dateDeleted == DateDeletedLocalFailed)
}
