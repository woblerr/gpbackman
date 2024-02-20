package gpbckpconfig

import (
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
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "Test full path",
			value:   "/som/path/test.txt",
			wantErr: false,
		},
		{
			name:    "Test zero length path",
			value:   "",
			wantErr: true,
		},
		{
			name:    "Test not full path",
			value:   "test.txt",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckFullPath(tt.value)
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
			if got := reportFileName(tt.timestamp); got != tt.want {
				t.Errorf("\nreportFileName():\n%v\nwant:\n%v", got, tt.want)
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
