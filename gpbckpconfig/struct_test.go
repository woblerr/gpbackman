package gpbckpconfig

import (
	"reflect"
	"testing"
)

func TestGetBackupType(t *testing.T) {
	tests := []struct {
		name    string
		config  BackupConfig
		want    string
		wantErr bool
	}{
		{
			name: "Test incremental backup",
			config: BackupConfig{
				Incremental:  true,
				DataOnly:     false,
				MetadataOnly: false,
			},
			want:    backupTypeIncremental,
			wantErr: false,
		},
		{
			name: "Test data-only backup",
			config: BackupConfig{
				Incremental:  false,
				DataOnly:     true,
				MetadataOnly: false,
			},
			want:    backupTypeDataOnly,
			wantErr: false,
		},
		{
			name: "Test metadata-only backup",
			config: BackupConfig{
				Incremental:  false,
				DataOnly:     false,
				MetadataOnly: true,
			},
			want:    backupTypeMetadataOnly,
			wantErr: false,
		},
		{
			name: "Test metadata-only when no data in regular backup",
			config: BackupConfig{
				Incremental:  false,
				DataOnly:     true,
				MetadataOnly: true,
			},
			want:    backupTypeMetadataOnly,
			wantErr: false,
		},
		{
			name: "Test full backup",
			config: BackupConfig{
				Incremental:  false,
				DataOnly:     false,
				MetadataOnly: false,
			},
			want:    backupTypeFull,
			wantErr: false,
		},
		{
			name: "Test invalid backup case 1",
			config: BackupConfig{
				Incremental:  true,
				DataOnly:     true,
				MetadataOnly: false,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Test invalid backup case 2",
			config: BackupConfig{
				Incremental:  true,
				DataOnly:     true,
				MetadataOnly: true,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetBackupType()
			if (err != nil) != tt.wantErr {
				t.Errorf("\nGetBackupType() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestGetObjectFilteringInfo(t *testing.T) {
	tests := []struct {
		name    string
		config  BackupConfig
		want    string
		wantErr bool
	}{
		{
			name: "Test IncludeSchemaFiltered",
			config: BackupConfig{
				IncludeSchemaFiltered: true,
				ExcludeSchemaFiltered: false,
				IncludeTableFiltered:  false,
				ExcludeTableFiltered:  false,
			},
			want:    objectFilteringIncludeSchema,
			wantErr: false,
		},
		{
			name: "Test ExcludeSchemaFiltered",
			config: BackupConfig{
				IncludeSchemaFiltered: false,
				ExcludeSchemaFiltered: true,
				IncludeTableFiltered:  false,
				ExcludeTableFiltered:  false,
			},
			want:    objectFilteringExcludeSchema,
			wantErr: false,
		},
		{
			name: "Test IncludeTableFiltered",
			config: BackupConfig{
				IncludeSchemaFiltered: false,
				ExcludeSchemaFiltered: false,
				IncludeTableFiltered:  true,
				ExcludeTableFiltered:  false,
			},
			want:    objectFilteringIncludeTable,
			wantErr: false,
		},
		{
			name: "Test ExcludeTableFiltered",
			config: BackupConfig{
				IncludeSchemaFiltered: false,
				ExcludeSchemaFiltered: false,
				IncludeTableFiltered:  false,
				ExcludeTableFiltered:  true,
			},
			want:    objectFilteringExcludeTable,
			wantErr: false,
		},
		{
			name: "Test NoFiltering",
			config: BackupConfig{
				IncludeSchemaFiltered: false,
				ExcludeSchemaFiltered: false,
				IncludeTableFiltered:  false,
				ExcludeTableFiltered:  false,
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "Test InvalidFiltering",
			config: BackupConfig{
				IncludeSchemaFiltered: true,
				ExcludeSchemaFiltered: true,
				IncludeTableFiltered:  true,
				ExcludeTableFiltered:  true,
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetObjectFilteringInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("\nGetObjectFilteringInfo() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestGetBackupDate(t *testing.T) {
	tests := []struct {
		name    string
		config  BackupConfig
		want    string
		wantErr bool
	}{
		{
			name:    "Test valid timestamp",
			config:  BackupConfig{Timestamp: "20220401102430"},
			want:    "Fri Apr 01 2022 10:24:30",
			wantErr: false,
		},
		{
			name:    "Test invalid timestamp",
			config:  BackupConfig{Timestamp: "invalid"},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetBackupDate()
			if (err != nil) != tt.wantErr {
				t.Errorf("\nGetBackupDate() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestGetBackupDuration(t *testing.T) {
	tests := []struct {
		name    string
		config  BackupConfig
		want    float64
		wantErr bool
	}{
		{
			name: "Test valid timestamps",
			config: BackupConfig{
				Timestamp: "20220401102430",
				EndTime:   "20220401115502",
			},
			want:    5432,
			wantErr: false,
		},
		{
			name: "invalid start timestamp",
			config: BackupConfig{
				Timestamp: "invalid",
				EndTime:   "20220401115502",
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "invalid end timestamp",
			config: BackupConfig{
				Timestamp: "20220401102430",
				EndTime:   "invalid",
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetBackupDuration()
			if (err != nil) != tt.wantErr {
				t.Errorf("\nGetBackupDuration() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestGetBackupDateDeleted(t *testing.T) {
	testCases := []struct {
		name    string
		config  BackupConfig
		want    string
		wantErr bool
	}{
		{
			name:    "Test empty",
			config:  BackupConfig{DateDeleted: ""},
			want:    "",
			wantErr: false,
		},
		{
			name:    "Test in progress",
			config:  BackupConfig{DateDeleted: DateDeletedInProgress},
			want:    DateDeletedInProgress,
			wantErr: false,
		},
		{
			name:    "Test plugin backup delete failed",
			config:  BackupConfig{DateDeleted: DateDeletedPluginFailed},
			want:    DateDeletedPluginFailed,
			wantErr: false,
		},
		{
			name:    "Test local delete failed",
			config:  BackupConfig{DateDeleted: DateDeletedLocalFailed},
			want:    DateDeletedLocalFailed,
			wantErr: false,
		},
		{
			name:    "Test valid date",
			config:  BackupConfig{DateDeleted: "20220401102430"},
			want:    "Fri Apr 01 2022 10:24:30",
			wantErr: false,
		},
		{
			name:    "Test invalid date",
			config:  BackupConfig{DateDeleted: "InvalidDate"},
			want:    "InvalidDate",
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetBackupDateDeleted()
			if (err != nil) != tt.wantErr {
				t.Errorf("\nGetBackupDateDeleted() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestIsSuccess(t *testing.T) {
	tests := []struct {
		name    string
		config  BackupConfig
		want    bool
		wantErr bool
	}{
		{
			name:    "Test success status",
			config:  BackupConfig{Status: backupStatusSuccess},
			want:    true,
			wantErr: false,
		},
		{
			name:    "Test Failure status",
			config:  BackupConfig{Status: backupStatusFailure},
			want:    false,
			wantErr: false,
		},
		{
			name:    "Test Unknown status",
			config:  BackupConfig{Status: "unknown"},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.IsSuccess()
			if (err != nil) != tt.wantErr {
				t.Errorf("\nIsSuccess() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestIsLocal(t *testing.T) {
	tests := []struct {
		name   string
		config BackupConfig
		want   bool
	}{
		{
			name:   "Test local backup",
			config: BackupConfig{Plugin: ""},
			want:   true,
		},
		{
			name:   "Test plugin backup",
			config: BackupConfig{Plugin: "plugin"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsLocal()
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestFindBackupConfig(t *testing.T) {
	historyTest := &History{
		BackupConfigs: []BackupConfig{
			{Timestamp: "20220401102430"},
		},
	}
	tests := []struct {
		name    string
		history History
		value   string
		want    BackupConfig
		wantErr bool
	}{
		{
			name:    "Test existed timestamp",
			history: *historyTest,
			value:   "20220401102430",
			want:    BackupConfig{Timestamp: "20220401102430"},
			wantErr: false,
		},
		{
			name:    "Test unknown timestamp",
			history: *historyTest,
			value:   "unknown",
			want:    BackupConfig{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got, err := tt.history.FindBackupConfig(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nFindBackupConfig() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestUpdateBackupConfigDateDeleted(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		value     string
		want      History
		wantErr   bool
	}{
		{
			name:      "Test existed timestamp and date deleted",
			timestamp: "20220401102430",
			value:     "20220401112430",
			want: History{BackupConfigs: []BackupConfig{
				{Timestamp: "20220401092430", DateDeleted: ""},
				{Timestamp: "20220401102430", DateDeleted: "20220401112430"},
			},
			},
			wantErr: false,
		},
		{
			name:      "Test existed timestamp and some text",
			timestamp: "20220401102430",
			value:     DateDeletedPluginFailed,
			want: History{BackupConfigs: []BackupConfig{
				{Timestamp: "20220401092430", DateDeleted: ""},
				{Timestamp: "20220401102430", DateDeleted: DateDeletedPluginFailed},
			},
			},
			wantErr: false,
		},
		{
			name:      "Test unknown timestamp",
			timestamp: "unknown",
			value:     "unknown",
			want: History{BackupConfigs: []BackupConfig{
				{Timestamp: "20220401092430", DateDeleted: ""},
				{Timestamp: "20220401102430", DateDeleted: ""},
			},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyTest := History{
				BackupConfigs: []BackupConfig{
					{Timestamp: "20220401092430", DateDeleted: ""},
					{Timestamp: "20220401102430", DateDeleted: ""},
				},
			}
			err := historyTest.UpdateBackupConfigDateDeleted(tt.timestamp, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nUpdateBackupConfigDateDeleted() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(historyTest, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", historyTest, tt.want)
			}
		})
	}
}

func TestFindBackupConfigDependencies(t *testing.T) {
	historyTest := History{
		BackupConfigs: []BackupConfig{
			{
				Timestamp: "20220401202430",
				RestorePlan: []RestorePlanEntry{
					{Timestamp: "20220401092430"},
					{Timestamp: "20220401102430"},
					{Timestamp: "20220401202430"},
				},
			},
			{
				Timestamp: "20220401102430",
				RestorePlan: []RestorePlanEntry{
					{Timestamp: "20220401092430"},
					{Timestamp: "20220401102430"},
				},
			},
			{Timestamp: "20220401092430",
				RestorePlan: []RestorePlanEntry{
					{Timestamp: "20220401092430"},
				},
			},
			{Timestamp: "20220401082430",
				RestorePlan: []RestorePlanEntry{},
			},
			{Timestamp: "20220401072430",
				RestorePlan: []RestorePlanEntry{},
			},
		},
	}
	tests := []struct {
		name         string
		timestamp    string
		stopPosition int
		want         []string
	}{
		{
			name:         "Test with dependent backups idx 0",
			timestamp:    "20220401202430",
			stopPosition: 0,
			want:         []string{},
		},
		{
			name:         "Test with dependent backups idx 1",
			timestamp:    "20220401102430",
			stopPosition: 1,
			want:         []string{"20220401202430"},
		},
		{
			name:         "Test no dependent backups idx 2",
			timestamp:    "20220401092430",
			stopPosition: 2,
			want:         []string{"20220401202430", "20220401102430"},
		},
		{
			name:         "Test no dependent backups idx 4",
			timestamp:    "20220401072430",
			stopPosition: 4,
			want:         []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := historyTest.FindBackupConfigDependencies(tt.timestamp, tt.stopPosition)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}
