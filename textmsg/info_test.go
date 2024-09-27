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
		{
			name:     "Test InfoTextBackupAlreadyDeleted",
			value:    "TestBackup",
			function: InfoTextBackupAlreadyDeleted,
			want:     "Backup TestBackup has already been deleted",
		},
		{
			name:     "Test InfoTextBackupDirPath",
			value:    "/test/path",
			function: InfoTextBackupDirPath,
			want:     "Path to backup directory: /test/path",
		},
		{
			name:     "Test InfoTextSegmentPrefix",
			value:    "TestValue",
			function: InfoTextSegmentPrefix,
			want:     "Segment Prefix: TestValue",
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

func TestInfoTextFunctionAndTwoArgs(t *testing.T) {
	tests := []struct {
		name     string
		value1   string
		value2   string
		function func(string, string) string
		want     string
	}{
		{
			name:     "Test InfoTextBackupUnableDeleteFailed",
			value1:   "TestBackup",
			value2:   "In Progress",
			function: InfoTextBackupStatus,
			want:     "Backup TestBackup has status: In Progress",
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

func TestInfoTextFunctionAndMultipleSeparateArgs(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		function func(...string) string
		want     string
	}{
		{
			name:     "Test InfoTextCommandExecution",
			values:   []string{"execution_command", "some_argument"},
			function: InfoTextCommandExecution,
			want:     "Executing command: execution_command some_argument",
		},
		{
			name:     "Test InfoTextCommandExecutionSucceeded",
			values:   []string{"execution_command", "some_argument"},
			function: InfoTextCommandExecutionSucceeded,
			want:     "Command succeeded: execution_command some_argument",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.values...); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestInfoTextFunctionAndMultipleListArgs(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		function func([]string) string
		want     string
	}{
		{
			name:     "Test InfoTextBackupDeleteList",
			values:   []string{"TestBackup1", "TestBackup2"},
			function: InfoTextBackupDeleteList,
			want:     "The following backups will be deleted: TestBackup1, TestBackup2",
		},
		{
			name:     "Test InfoTextBackupDeleteListFromHistory",
			values:   []string{"TestBackup1", "TestBackup2"},
			function: InfoTextBackupDeleteListFromHistory,
			want:     "The following backups will be deleted from history: TestBackup1, TestBackup2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.values); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestInfoTextFunction(t *testing.T) {
	tests := []struct {
		name     string
		function func() string
		want     string
	}{
		{
			name:     "Test InfoTextNothingToDo",
			function: InfoTextNothingToDo,
			want:     "Nothing to do",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
