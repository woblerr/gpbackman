package gpbckpconfig

import "testing"

func TestGetBackupNameFile(t *testing.T) {
	tests := []struct {
		name        string
		showD       bool
		showF       bool
		sAll        bool
		status      string
		dateDeleted string
		want        bool
	}{
		{
			name:        "Test show all (the other values are not important)",
			showD:       false,
			showF:       false,
			sAll:        true,
			status:      backupStatusFailure,
			dateDeleted: "test",
			want:        true,
		},
		{
			name:        "Test show deleted (failed)",
			showD:       true,
			showF:       false,
			sAll:        false,
			status:      backupStatusFailure,
			dateDeleted: "",
			want:        false,
		},
		{
			name:        "Test show deleted (successful and active)",
			showD:       true,
			showF:       false,
			sAll:        false,
			status:      backupStatusSuccess,
			dateDeleted: "",
			want:        false,
		},
		{
			name:        "Test show deleted (successful and deleted)",
			showD:       true,
			showF:       false,
			sAll:        false,
			status:      backupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        true,
		},
		{
			name:        "Test show failed",
			showD:       false,
			showF:       true,
			sAll:        false,
			status:      backupStatusFailure,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test show failed (not failed)",
			showD:       false,
			showF:       true,
			sAll:        false,
			status:      backupStatusSuccess,
			dateDeleted: "",
			want:        false,
		},
		{
			name:        "Test default (not failed and active)",
			showD:       false,
			showF:       false,
			sAll:        false,
			status:      backupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test default (not failed and not active)",
			showD:       false,
			showF:       false,
			sAll:        false,
			status:      backupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBackupNameFile(tt.showD, tt.showF, tt.sAll, tt.status, tt.dateDeleted); got != tt.want {
				t.Errorf("GetBackupNameFile(%v, %v, %v, %v, %v):\n%v\nwant:\n%v", tt.showD, tt.showF, tt.sAll, tt.status, tt.dateDeleted, got, tt.want)
			}
		})
	}
}
