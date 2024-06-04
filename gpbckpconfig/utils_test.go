package gpbckpconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "Test valid timestamp",
			value:   "20230822120000",
			wantErr: false,
		},
		{
			name:    "Test invalid timestamp",
			value:   "invalid",
			wantErr: true,
		},
		{
			name:    "Test invalid timestamp (wrong length)",
			value:   "2023082212000",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTimestamp(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nVariables do not match:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckFullPath(t *testing.T) {
	// Create a temporary file to simulate an existing file
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	// Clean up
	defer os.Remove(tempFile.Name())

	tests := []struct {
		name            string
		value           string
		checkFileExists bool
		wantErr         bool
	}{
		{
			name:            "Test exist file and full path",
			value:           tempFile.Name(),
			checkFileExists: true,
			wantErr:         false,
		},
		{
			name:            "Test full path and not exist file",
			value:           "/some/path/test.txt",
			checkFileExists: true,
			wantErr:         true,
		},
		{
			name:            "Test zero length path",
			value:           "",
			checkFileExists: false,
			wantErr:         true,
		},
		{
			name:            "Test not full path",
			value:           "test.txt",
			checkFileExists: false,
			wantErr:         true,
		},
	}
	fmt.Print(tempFile.Name())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckFullPath(tt.value, tt.checkFileExists)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nVariables do not match:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestIsBackupActive(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "Test empty delete date",
			value: "",
			want:  true,
		},
		{
			name:  "Test plugin error",
			value: DateDeletedPluginFailed,
			want:  true,
		},
		{
			name:  "Test local error",
			value: DateDeletedLocalFailed,
			want:  true,
		},
		{
			name:  "Test deletion in progress",
			value: DateDeletedInProgress,
			want:  false,
		},
		{
			name:  "Test deleted",
			value: "20220401102430",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBackupActive(tt.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestBackupS3PluginReportPath(t *testing.T) {
	tests := []struct {
		name          string
		timestamp     string
		pluginOptions map[string]string
		want          string
		wantErr       bool
	}{
		{
			name:          "valid timestamp and options",
			timestamp:     "20230112131415",
			pluginOptions: map[string]string{"folder": "/path/to/folder"},
			want:          "/path/to/folder/backups/20230112/20230112131415/gpbackup_20230112131415_report",
			wantErr:       false,
		},
		{
			name:      "missing options",
			timestamp: "20230112131415",
			wantErr:   true,
		},
		{
			name:          "wrong options",
			timestamp:     "20230112131415",
			pluginOptions: map[string]string{"wrong_key": "/path/to/folder"},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backupS3PluginReportPath(tt.timestamp, tt.pluginOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nbackupS3PluginReportPath() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("\nbackupS3PluginReportPath() got:%v\nwant:%v\n", got, tt.want)
			}
		})
	}
}

func TestReportFileName(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		want      string
	}{
		{
			name:      "Valid timestamp",
			timestamp: "202301011234",
			want:      "gpbackup_202301011234_report",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReportFileName(tt.timestamp); got != tt.want {
				t.Errorf("\nReportFileName():\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestBackupPluginCustomReportPath(t *testing.T) {
	type args struct {
		timestamp   string
		folderValue string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "basic",
			args: args{
				timestamp:   "20230101123456",
				folderValue: "/backup/folder",
			},
			want: "/backup/folder/gpbackup_20230101123456_report",
		},
		{
			name: "folder with leading and trailing slashes",
			args: args{
				timestamp:   "20230101123456",
				folderValue: "/backup//folder//",
			},
			want: "/backup/folder/gpbackup_20230101123456_report",
		},
		{
			name: "folder with spaces",
			args: args{
				timestamp:   "20230101123456",
				folderValue: "/backup/folder with spaces",
			},
			want: "/backup/folder with spaces/gpbackup_20230101123456_report",
		},
		{
			name: "folder without leading /",
			args: args{
				timestamp:   "20230101123456",
				folderValue: "backup/folder with spaces",
			},
			want: "/backup/folder with spaces/gpbackup_20230101123456_report",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backupPluginCustomReportPath(tt.args.timestamp, tt.args.folderValue); got != tt.want {
				t.Errorf("\nbackupPluginCustomReportPath():\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestGetTimestampOlderThen(t *testing.T) {
	// Call the function with a known input
	input := uint(1)
	got := GetTimestampOlderThen(input)

	// Parse the returned timestamp
	parsedTime, err := time.Parse(Layout, got)
	if err != nil {
		t.Errorf("Failed to parse timestamp: %v", err)
	}

	// Check if the returned timestamp is within the expected range
	now := time.Now()
	if !parsedTime.Before(now.Add(-time.Duration(input)*24*time.Hour)) || parsedTime.After(now) {
		t.Errorf("Returned timestamp is not within the expected range")
	}
}

func TestCheckTableFQN(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "Test valid table name",
			value:   "public.table_1",
			wantErr: false,
		},
		{
			name:    "Test invalid table name",
			value:   "invalid_table",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTableFQN(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nCheckTableFQN():\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}
func TestReportFilePath(t *testing.T) {
	backupDir := "/path/to/backup"
	timestamp := "20230101123456"
	want := "/path/to/backup/backups/20230101/20230101123456/gpbackup_20230101123456_report"
	got := ReportFilePath(backupDir, timestamp)
	if got != want {
		t.Errorf("\nReportFilePath():\n%v\nwant:\n%v", got, want)
	}
}

func TestGetSegPrefix(t *testing.T) {
	backupDir := "/path/to/backup/segment-1/backups"
	want := "segment"
	got := GetSegPrefix(backupDir)
	if got != want {
		t.Errorf("\nGetSegPrefix():\n%v\nwant:\n%v", got, want)
	}
}

func TestCheckMasterBackupDir(t *testing.T) {
	tempDir := os.TempDir()

	tests := []struct {
		name                string
		testDir             string
		backupDir           string
		wantDir             string
		wantPrefix          string
		wantSingleBackupDir bool
		wantErr             bool
	}{
		{
			name:                "Valid single backup dir",
			testDir:             filepath.Join(tempDir, "noSegPrefix", "backups"),
			backupDir:           filepath.Join(tempDir, "noSegPrefix"),
			wantDir:             filepath.Join(tempDir, "noSegPrefix"),
			wantPrefix:          "",
			wantSingleBackupDir: true,
			wantErr:             false,
		},
		{
			name:                "Valid single backup dir with segment prefix",
			testDir:             filepath.Join(tempDir, "segPrefix", "segment-1", "backups"),
			backupDir:           filepath.Join(tempDir, "segPrefix"),
			wantDir:             filepath.Join(tempDir, "segPrefix", "segment-1"),
			wantPrefix:          "segment",
			wantSingleBackupDir: false,
			wantErr:             false,
		},
		{
			name:                "Invalid backup dir",
			testDir:             filepath.Join(tempDir, "invalid"),
			backupDir:           filepath.Join(tempDir, "invalid"),
			wantDir:             "",
			wantPrefix:          "",
			wantSingleBackupDir: false,
			wantErr:             true,
		},
		{
			name:                "Multiple backup dirs",
			testDir:             tempDir,
			backupDir:           "some/path",
			wantDir:             "",
			wantPrefix:          "",
			wantSingleBackupDir: false,
			wantErr:             true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.MkdirAll(tt.testDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.Remove(tt.testDir)
			gotDir, gotPrefix, gotIsSingleBackupDir, err := CheckMasterBackupDir(tt.backupDir)
			// Check for unexpected error
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckMasterBackupDir() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}

			// Check the returned directory
			if gotDir != tt.wantDir {
				t.Errorf("CheckMasterBackupDir() gotDir:\n%v\nwantDir:\n%v", gotDir, tt.wantDir)
			}

			// Check the returned prefix
			if gotPrefix != tt.wantPrefix {
				t.Errorf("CheckMasterBackupDir() gotPrefix:\n%v\nwantPrefix:\n%v", gotPrefix, tt.wantPrefix)
			}

			// Check if the returned value for single backup directory is correct
			if gotIsSingleBackupDir != tt.wantSingleBackupDir {
				t.Errorf("CheckMasterBackupDir() gotIsSingleBackupDir:\n%v\nwantSingleBackupDir:\n%v", gotIsSingleBackupDir, tt.wantSingleBackupDir)
			}
		})
	}
}
