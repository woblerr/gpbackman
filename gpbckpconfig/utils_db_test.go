package gpbckpconfig

import "testing"

func TestGetBackupNameQuery(t *testing.T) {
	tests := []struct {
		name  string
		showD bool
		showF bool
		want  string
	}{
		{
			name:  "Test show all",
			showD: true,
			showF: true,
			want:  `SELECT timestamp FROM backups ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show deleted",
			showD: true,
			showF: false,
			want:  `SELECT timestamp FROM backups WHERE status != 'Failure' ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show failed",
			showD: false,
			showF: true,
			want:  `SELECT timestamp FROM backups WHERE date_deleted IN ('', 'In progress', 'Plugin Backup Delete Failed', 'Local Delete Failed') ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show default",
			showD: false,
			showF: false,
			want:  `SELECT timestamp FROM backups WHERE status != 'Failure' AND date_deleted IN ('', 'In progress', 'Plugin Backup Delete Failed', 'Local Delete Failed') ORDER BY timestamp DESC;`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBackupNameQuery(tt.showD, tt.showF); got != tt.want {
				t.Errorf("getBackupNameQuery(%v, %v):\n%v\nwant:\n%v", tt.showD, tt.showF, got, tt.want)
			}
		})
	}
}

func TestGetBackupDependenciesQuery(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "Test valid result",
			value: "TestBackup",
			want:  `SELECT timestamp from restore_plans WHERE timestamp != 'TestBackup' AND restore_plan_timestamp = 'TestBackup' ORDER BY timestamp DESC;`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBackupDependenciesQuery(tt.value); got != tt.want {
				t.Errorf("getBackupDependenciesQuery(%v):\n%v\nwant:\n%v", tt.value, got, tt.want)
			}
		})
	}
}
