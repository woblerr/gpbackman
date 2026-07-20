package textmsg

import (
	"errors"
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

func TestWarnTextHistoryStandbySyncFailed(t *testing.T) {
	got := WarnTextHistoryStandbySyncFailed(errors.New("rsync failed"))
	want := "History db sync to standby coordinator failed; standby history may be stale: rsync failed"
	if got != want {
		t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, want)
	}
}
