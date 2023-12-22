package textmsg

import (
	"testing"
)

func TestWarnTextFunctionsWarnAndArg(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		function func(string) string
		want     string
	}{
		{
			name:     "Test WarnTextBackupAlreadyDeleted",
			value:    "TestBackup",
			function: WarnTextBackupAlreadyDeleted,
			want:     "Backup TestBackup has already been deleted",
		},
		{
			name:     "Test WarnTextBackupUnableDeleteFailed",
			value:    "TestBackup",
			function: WarnTextBackupFailedStatus,
			want:     "Backup TestBackup has failed status. Nothing to do",
		},
		{
			name:     "Test WarnTextBackupUnableGetReport",
			value:    "TestBackup",
			function: WarnTextBackupUnableGetReport,
			want:     "Unable to get report for backup TestBackup. Check if backup is active",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
