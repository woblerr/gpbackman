package gpbckpconfig

import (
	"testing"
)

func TestGetBackupNameFile(t *testing.T) {
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
			status:      backupStatusFailure,
			dateDeleted: "",
			want:        false,
		},
		{
			name:        "Test show deleted (successful and active)",
			showD:       true,
			showF:       false,
			status:      backupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test show deleted (successful and deleted)",
			showD:       true,
			showF:       false,
			status:      backupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        true,
		},
		{
			name:        "Test show failed",
			showD:       false,
			showF:       true,
			status:      backupStatusFailure,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test show failed (not failed)",
			showD:       false,
			showF:       true,
			status:      backupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test default (not failed and active)",
			showD:       false,
			showF:       false,
			status:      backupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test default (not failed and deletion in progress)",
			showD:       false,
			showF:       false,
			status:      backupStatusSuccess,
			dateDeleted: DateDeletedInProgress,
			want:        true,
		},
		{
			name:        "Test default (not failed and not active)",
			showD:       false,
			showF:       false,
			status:      backupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        false,
		},
		{
			name:        "Test show all backups",
			showD:       true,
			showF:       true,
			status:      backupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBackupNameFile(tt.showD, tt.showF, tt.status, tt.dateDeleted); got != tt.want {
				t.Errorf("GetBackupNameFile(%v, %v, %v, %v):\n%v\nwant:\n%v", tt.showD, tt.showF, tt.status, tt.dateDeleted, got, tt.want)
			}
		})
	}
}
