package gpbckpconfig

import "testing"

func TestGetBackupNameQuery(t *testing.T) {
	tests := []struct {
		name  string
		showD bool
		showF bool
		sAll  bool
		want  string
	}{
		{
			name:  "Test show all",
			showD: false,
			showF: false,
			sAll:  true,
			want:  `SELECT timestamp FROM backups ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show deleted",
			showD: true,
			showF: false,
			sAll:  false,
			want:  `SELECT timestamp FROM backups WHERE status != 'Failure' AND date_deleted NOT IN ('', 'In progress', 'Plugin Backup Delete Failed', 'Local Delete Failed') ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show failed",
			showD: false,
			showF: true,
			sAll:  false,
			want:  `SELECT timestamp FROM backups WHERE status = 'Failure' ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show default",
			showD: false,
			showF: false,
			sAll:  false,
			want:  `SELECT timestamp FROM backups WHERE status != 'Failure' AND date_deleted IN ('', 'In progress', 'Plugin Backup Delete Failed', 'Local Delete Failed') ORDER BY timestamp DESC;`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBackupNameQuery(tt.showD, tt.showF, tt.sAll); got != tt.want {
				t.Errorf("getBackupNameQuery(%v, %v, %v):\n%v\nwant:\n%v", tt.showD, tt.showF, tt.sAll, got, tt.want)
			}
		})
	}
}
