package textmsg

import "testing"

func TestInfoTextFunctionaAndArg(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		function func(string) string
		want     string
	}{
		{
			name:     "Test InfoTextBackupDeleteSuccess",
			value:    "TestBackup",
			function: InfoTextBackupDeleteSuccess,
			want:     "Backup TestBackup successfully deleted.",
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
