package textmsg

import (
	"errors"
	"testing"
)

func TestErrorTextFunctionsErrorOnly(t *testing.T) {
	testError := errors.New("test error")
	tests := []struct {
		name     string
		testErr  error
		function func(error) string
		want     string
	}{
		{
			name:     "Test ErrorTextUnableOpenHistoryDB",
			testErr:  testError,
			function: ErrorTextUnableOpenHistoryDB,
			want:     "Unable to open history db. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableReadHistoryDB",
			testErr:  testError,
			function: ErrorTextUnableReadHistoryDB,
			want:     "Unable to read data from history db. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableWriteIntoHistoryDB",
			testErr:  testError,
			function: ErrorTextUnableWriteIntoHistoryDB,
			want:     "Unable to write into history db. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableInitHistoryDB",
			testErr:  testError,
			function: ErrorTextUnableInitHistoryDB,
			want:     "Unable to initialize history db. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableReadPluginConfigFile",
			testErr:  testError,
			function: ErrorTextUnableReadPluginConfigFile,
			want:     "Unable to read plugin config file. Error: test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.testErr); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestErrorTextFunctionsErrorAndArg(t *testing.T) {
	testError := errors.New("test error")
	testBackupName := "TestBackup"
	tests := []struct {
		name     string
		value    string
		testErr  error
		function func(string, error) string
		want     string
	}{
		{
			name:     "Test ErrorTextUnableGetBackupInfo",
			value:    testBackupName,
			testErr:  testError,
			function: ErrorTextUnableGetBackupInfo,
			want:     "Unable to get info for backup TestBackup. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableDeleteBackup",
			value:    testBackupName,
			testErr:  testError,
			function: ErrorTextUnableDeleteBackup,
			want:     "Unable to delete backup TestBackup. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableDeleteBackupCascade",
			value:    testBackupName,
			testErr:  testError,
			function: ErrorTextUnableDeleteBackupCascade,
			want:     "Unable to delete dependent backups for backup TestBackup. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableDeleteBackupUseCascade",
			value:    testBackupName,
			testErr:  testError,
			function: ErrorTextUnableDeleteBackupUseCascade,
			want:     "Backup TestBackup has dependent backups. Use --cascade option. Error: test error",
		},
		{
			name:     "Test ErrorTextBackupInProgress",
			value:    testBackupName,
			testErr:  testError,
			function: ErrorTextBackupDeleteInProgress,
			want:     "Backup TestBackup deletion in progress. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableActionHistoryFile",
			value:    "do something with",
			testErr:  testError,
			function: ErrorTextUnableActionHistoryFile,
			want:     "Unable to do something with history file. Error: test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.value, tt.testErr); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestErrorTextFunctionsErrorAndTwoArgs(t *testing.T) {
	testError := errors.New("test error")
	tests := []struct {
		name     string
		value1   string
		value2   string
		testErr  error
		function func(string, string, error) string
		want     string
	}{
		{
			name:     "Test ErrorTextUnableSetBackupStatus",
			value1:   "Test status",
			value2:   "TestBackup",
			testErr:  testError,
			function: ErrorTextUnableSetBackupStatus,
			want:     "Unable to set Test status status for backup TestBackup. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableValidateFlag",
			value1:   "TestValue",
			value2:   "TestFlag",
			testErr:  testError,
			function: ErrorTextUnableValidateFlag,
			want:     "Unable to validate value TestValue for flag TestFlag. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableGetBackupValue",
			value1:   "test parameter",
			value2:   "TestBackup",
			testErr:  testError,
			function: ErrorTextUnableGetBackupValue,
			want:     "Unable to get backup test parameter for backup TestBackup. Error: test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.value1, tt.value2, tt.testErr); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestErrorTextFunctionsErrorAndMultipleArgs(t *testing.T) {
	testError := errors.New("test error")
	tests := []struct {
		name     string
		values   []string
		testErr  error
		function func(error, ...string) string
		want     string
	}{
		{
			name:     "Test ErrorTextUnableCompatibleFlags",
			values:   []string{"TestFlag1", "TestFlag2"},
			testErr:  testError,
			function: ErrorTextUnableCompatibleFlags,
			want:     "Unable to use the following flags together: TestFlag1, TestFlag2. Error: test error",
		},
		{
			name:     "Test ErrorTextUnableCompatibleFlagsValues",
			values:   []string{"TestFlag1", "TestValue1", "TestFlag2", "TestValue2"},
			testErr:  testError,
			function: ErrorTextUnableCompatibleFlagsValues,
			want:     "Unable to use the provided values for these flags together: TestFlag1=TestValue1, TestFlag2=TestValue2. Error: test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.testErr, tt.values...); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestErrorFunctions(t *testing.T) {
	tests := []struct {
		name     string
		errFunc  func() error
		expected string
	}{
		{"ErrorInvalidValueError", ErrorInvalidValueError, "invalid flag value"},
		{"ErrorIncompatibleValuesError", ErrorIncompatibleValuesError, "incompatible flags values"},
		{"ErrorIncompatibleFlagsError", ErrorIncompatibleFlagsError, "incompatible flags"},
		{"ErrorBackupDeleteCascadeError", ErrorBackupDeleteCascadeError, "delete cascade is failed"},
		{"ErrorBackupDeleteInProgressError", ErrorBackupDeleteInProgressError, "backup deletion in progress"},
		{"ErrorBackupDeleteCascadeOptionError", ErrorBackupDeleteCascadeOptionError, "use cascade option"},
		{"ErrorValidationFullPath", ErrorValidationFullPath, "not an absolute path"},
		{"ErrorValidationTimestamp", ErrorValidationTimestamp, "not a timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errFunc()
			if err == nil || err.Error() != tt.expected {
				t.Errorf("\n%s() error:\n%v\nwant:\n%v", tt.name, err, tt.expected)
			}
		})
	}
}