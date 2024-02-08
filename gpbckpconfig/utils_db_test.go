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
		name     string
		value    string
		function func(string) string
		want     string
	}{
		{
			name:     "Test getBackupDependenciesQuery",
			value:    "TestBackup",
			function: getBackupDependenciesQuery,
			want: `
SELECT timestamp 
FROM restore_plans
WHERE timestamp != 'TestBackup'
	AND restore_plan_timestamp = 'TestBackup'
ORDER BY timestamp DESC;
`},
		{
			name:     "Test getBackupNameBeforeTimestampQuery",
			value:    "20240101120000",
			function: getBackupNameBeforeTimestampQuery,
			want: `
SELECT timestamp 
FROM backups 
WHERE timestamp < '20240101120000' 
	AND status != 'Failure' 
	AND date_deleted IN ('', 'Plugin Backup Delete Failed', 'Local Delete Failed') 
ORDER BY timestamp DESC;
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.value); got != tt.want {
				t.Errorf("getBackupDependenciesQuery(%v):\n%v\nwant:\n%v", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetBackupNameForCleanBeforeTimestampQuery(t *testing.T) {
	tests := []struct {
		name  string
		value string
		showD bool
		want  string
	}{
		{
			name:  "Show deleted and failed backups",
			value: "20240101120000",
			showD: true,
			want:  `SELECT timestamp FROM backups WHERE timestamp < '20240101120000' AND (status = 'Failure' OR date_deleted NOT IN ('', 'Plugin Backup Delete Failed', 'Local Delete Failed', 'In progress')) ORDER BY timestamp DESC;`,
		},
		{
			name:  "Show only failed backups",
			value: "20240101120000",
			showD: false,
			want:  `SELECT timestamp FROM backups WHERE timestamp < '20240101120000' AND status = 'Failure' ORDER BY timestamp DESC;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBackupNameForCleanBeforeTimestampQuery(tt.value, tt.showD); got != tt.want {
				t.Errorf("getBackupNameForCleanBeforeTimestampQuery(%v, %v):\n%v\nwant:\n%v", tt.value, tt.showD, got, tt.want)
			}
		})
	}
}

func TestTextQueryFunctionsArg(t *testing.T) {
	testBackupName := "TestBackup"
	tests := []struct {
		name     string
		value1   string
		value2   string
		function func(string, string) string
		want     string
	}{
		{
			name:     "Test deleteBackupsFormTableQuery",
			value1:   testBackupName,
			value2:   "'20220401102430', '20220401102430'",
			function: deleteBackupsFormTableQuery,
			want:     "DELETE FROM TestBackup WHERE timestamp IN ('20220401102430', '20220401102430');",
		},
		{
			name:     "Test updateDeleteStatusQuery",
			value1:   testBackupName,
			value2:   "20220401102430",
			function: updateDeleteStatusQuery,
			want:     "UPDATE backups SET date_deleted = '20220401102430' WHERE timestamp = 'TestBackup';",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.value1, tt.value2); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
