package textmsg

import "testing"

func TestInfoTextFunctionAndArg(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		function func(string) string
		want     string
	}{
		{
			name:     "Test InfoTextBackupDeleteStart",
			value:    "TestBackup",
			function: InfoTextBackupDeleteStart,
			want:     "Start deleting backup TestBackup",
		},
		{
			name:     "Test InfoTextBackupDeleteSuccess",
			value:    "TestBackup",
			function: InfoTextBackupDeleteSuccess,
			want:     "Backup TestBackup successfully deleted",
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

func TestInfoTextFunctionAndMultipleArgs(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		valueList []string
		function  func(string, []string) string
		want      string
	}{
		{
			name:      "Test InfoTextBackupDependenciesList",
			value:     "TestBackup1",
			valueList: []string{"TestBackup2", "TestBackup3"},
			function:  InfoTextBackupDependenciesList,
			want:      "Backup TestBackup1 has dependent backups: TestBackup2, TestBackup3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.value, tt.valueList); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
