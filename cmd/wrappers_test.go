package cmd

import "testing"

func TestGetHistoryFilePath(t *testing.T) {
	tests := []struct {
		name            string
		historyFilePath string
		want            string
	}{
		{"Empty Path", "", historyFileNameConst},
		{"Non Empty Path", "path/to/" + historyFileNameConst, "path/to/" + historyFileNameConst},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHistoryFilePath(tt.historyFilePath); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestGetHistoryDBPath(t *testing.T) {
	tests := []struct {
		name            string
		historyFilePath string
		want            string
	}{
		{"Empty Path", "", historyDBNameConst},
		{"Non Empty Path", "path/to/" + historyDBNameConst, "path/to/" + historyDBNameConst},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHistoryDBPath(tt.historyFilePath); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestIsBackupActive(t *testing.T) {
	tests := []struct {
		name            string
		historyFilePath string
		want            string
	}{
		{"Empty Path", "", historyDBNameConst},
		{"Non Empty Path", "path/to/" + historyDBNameConst, "path/to/" + historyDBNameConst},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHistoryDBPath(tt.historyFilePath); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}
