package gpbckpconfig

import (
	"testing"
)

func TestCheckBackupCanBeDisplayed(t *testing.T) {
	tests := []struct {
		name        string
		showD       bool
		showF       bool
		status      string
		dateDeleted string
		want        bool
	}{
		{
			name:        "Test show deleted (failed)",
			showD:       true,
			showF:       false,
			status:      BackupStatusFailure,
			dateDeleted: "",
			want:        false,
		},
		{
			name:        "Test show deleted (successful and active)",
			showD:       true,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test show deleted (successful and deleted)",
			showD:       true,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        true,
		},
		{
			name:        "Test show failed",
			showD:       false,
			showF:       true,
			status:      BackupStatusFailure,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test show failed (not failed)",
			showD:       false,
			showF:       true,
			status:      BackupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test default (not failed and active)",
			showD:       false,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test default (not failed and deletion in progress)",
			showD:       false,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: DateDeletedInProgress,
			want:        true,
		},
		{
			name:        "Test default (not failed and not active)",
			showD:       false,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        false,
		},
		{
			name:        "Test show all backups",
			showD:       true,
			showF:       true,
			status:      BackupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckBackupCanBeDisplayed(tt.showD, tt.showF, tt.status, tt.dateDeleted); got != tt.want {
				t.Errorf("CheckBackupIsValid(%v, %v, %v, %v):\n%v\nwant:\n%v", tt.showD, tt.showF, tt.status, tt.dateDeleted, got, tt.want)
			}
		})
	}
}
