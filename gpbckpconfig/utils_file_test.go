package gpbckpconfig

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestCheckBackupCanBeDisplayed(t *testing.T) {
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
			status:      BackupStatusFailure,
			dateDeleted: "",
			want:        false,
		},
		{
			name:        "Test show deleted (successful and active)",
			showD:       true,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test show deleted (successful and deleted)",
			showD:       true,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        true,
		},
		{
			name:        "Test show failed",
			showD:       false,
			showF:       true,
			status:      BackupStatusFailure,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test show failed (not failed)",
			showD:       false,
			showF:       true,
			status:      BackupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test default (not failed and active)",
			showD:       false,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: "",
			want:        true,
		},
		{
			name:        "Test default (not failed and deletion in progress)",
			showD:       false,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: DateDeletedInProgress,
			want:        true,
		},
		{
			name:        "Test default (not failed and not active)",
			showD:       false,
			showF:       false,
			status:      BackupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        false,
		},
		{
			name:        "Test show all backups",
			showD:       true,
			showF:       true,
			status:      BackupStatusSuccess,
			dateDeleted: "20220401102430",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckBackupCanBeDisplayed(tt.showD, tt.showF, tt.status, tt.dateDeleted); got != tt.want {
				t.Errorf("CheckBackupIsValid(%v, %v, %v, %v):\n%v\nwant:\n%v", tt.showD, tt.showF, tt.status, tt.dateDeleted, got, tt.want)
			}
		})
	}
}
func TestReadHistoryFile(t *testing.T) {
	filename := "testfile.yaml"
	expectedData := []byte("test data")

	// Mocking the os.ReadFile function
	execReadFile = func(name string) ([]byte, error) {
		if name != filename {
			return nil, fmt.Errorf("Expected filename %v, got %v", filename, name)
		}
		return expectedData, nil
	}
	defer func() { execReadFile = os.ReadFile }()

	tests := []struct {
		name    string
		file    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "Test read history file",
			file:    filename,
			data:    expectedData,
			wantErr: false,
		},
		{
			name:    "Test read history file (error)",
			file:    "testfile",
			data:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ReadHistoryFile(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadHistoryFile():\nerror:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(data, tt.data) {
				t.Errorf("ReadHistoryFile():\n%v\nwant:\n%v", data, tt.data)
			}
		})
	}
}
