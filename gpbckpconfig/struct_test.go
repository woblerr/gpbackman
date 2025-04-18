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
			want:    BackupTypeIncremental,
			wantErr: false,
		},
		{
			name: "Test data-only backup",
			config: BackupConfig{
				Incremental:  false,
				DataOnly:     true,
				MetadataOnly: false,
			},
			want:    BackupTypeDataOnly,
			wantErr: false,
		},
		{
			name: "Test metadata-only backup",
			config: BackupConfig{
				Incremental:  false,
				DataOnly:     false,
				MetadataOnly: true,
			},
			want:    BackupTypeMetadataOnly,
			wantErr: false,
		},
		{
			name: "Test metadata-only when no data in regular backup",
			config: BackupConfig{
				Incremental:  false,
				DataOnly:     true,
				MetadataOnly: true,
			},
			want:    BackupTypeMetadataOnly,
			wantErr: false,
		},
		{
			name: "Test metadata-only when no data in regular incremental backup",
			config: BackupConfig{
				Incremental:  true,
				DataOnly:     false,
				MetadataOnly: true,
			},
			want:    BackupTypeMetadataOnly,
			wantErr: false,
		},
		{
			name: "Test full backup",
			config: BackupConfig{
				Incremental:  false,
				DataOnly:     false,
				MetadataOnly: false,
			},
			want:    BackupTypeFull,
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
			config:  BackupConfig{Status: BackupStatusSuccess},
			want:    true,
			wantErr: false,
		},
		{
			name:    "Test Failure status",
			config:  BackupConfig{Status: BackupStatusFailure},
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

func TestBackupConfigGetReportFilePathPlugin(t *testing.T) {

	type args struct {
		customReportPath string
		pluginOptions    map[string]string
	}
	tests := []struct {
		name    string
		config  BackupConfig
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test custom report path",
			config: BackupConfig{
				Timestamp: "20220401102430",
				Plugin:    BackupS3Plugin,
			},
			args: args{
				customReportPath: "/path/to/report",
				pluginOptions:    make(map[string]string),
			},
			want:    "/path/to/report/gpbackup_20220401102430_report",
			wantErr: false,
		},
		{
			name: "Test s3 plugin report path if custom report path is not set and folder is absent",
			config: BackupConfig{
				Timestamp: "20220401102430",
				Plugin:    BackupS3Plugin,
			},
			args: args{
				customReportPath: "",
				pluginOptions: map[string]string{
					"bucket": "bucket",
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Test s3 plugin report path if custom report path is not set and folder is empty",
			config: BackupConfig{
				Timestamp: "20220401102430",
				Plugin:    BackupS3Plugin,
			},
			args: args{
				customReportPath: "",
				pluginOptions: map[string]string{
					"folder": "",
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Test s3 plugin report path if custom report path is not set and folder is ok",
			config: BackupConfig{
				Timestamp: "20220401102430",
				Plugin:    BackupS3Plugin,
			},

			args: args{
				customReportPath: "",
				pluginOptions: map[string]string{
					"folder": "/path/to/report",
				},
			},
			want:    "/path/to/report/backups/20220401/20220401102430/gpbackup_20220401102430_report",
			wantErr: false,
		},
		{
			name: "Test some plugin report path if custom report path is not set",
			config: BackupConfig{
				Timestamp: "20220401102430",
				Plugin:    "some_plugin",
			},

			args: args{
				customReportPath: "",
				pluginOptions: map[string]string{
					"folder": "/path/to/report",
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.GetReportFilePathPlugin(tt.args.customReportPath, tt.args.pluginOptions)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nBackupConfig.GetReportFilePathPlugin()error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("\nBackupConfig.GetReportFilePathPlugin() got:\n%v\nwant:%v\ns", got, tt.want)
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

func TestCheckObjectFilteringExists(t *testing.T) {
	tests := []struct {
		name          string
		tableFilter   string
		schemaFilter  string
		objectFilter  string
		excludeFilter bool
		want          bool
		BackupConfig  BackupConfig
	}{
		{
			name:          "no filters specified",
			tableFilter:   "",
			schemaFilter:  "",
			objectFilter:  "",
			excludeFilter: false,
			want:          true,
			BackupConfig:  BackupConfig{},
		},
		{
			name:          "table filter is specified and backup has included table",
			tableFilter:   "test.table1",
			schemaFilter:  "",
			objectFilter:  "include-table",
			excludeFilter: false,
			want:          true,
			BackupConfig: BackupConfig{
				IncludeRelations: []string{"test.table1", "test.table2"},
			},
		},
		{
			name:          "table filter is specified and backup hasn't included table",
			tableFilter:   "test.table1",
			schemaFilter:  "",
			objectFilter:  "include-table",
			excludeFilter: false,
			want:          false,
			BackupConfig: BackupConfig{
				IncludeRelations: []string{"test.table2", "test.table3"},
			},
		},
		{
			name:          "table filter is specified and backup hasn't object filter",
			tableFilter:   "test.table1",
			schemaFilter:  "",
			objectFilter:  "",
			excludeFilter: false,
			want:          false,
			BackupConfig:  BackupConfig{},
		},
		{
			name:          "table filter is specified and backup has another object filter",
			tableFilter:   "test.table1",
			schemaFilter:  "",
			objectFilter:  "include-schema",
			excludeFilter: false,
			want:          false,
			BackupConfig: BackupConfig{
				IncludeSchemas: []string{"test"},
			},
		},
		{
			name:          "table filter is specified and exclude filter is specified, backup has exclude table",
			tableFilter:   "test.table1",
			schemaFilter:  "",
			objectFilter:  "exclude-table",
			excludeFilter: true,
			want:          true,
			BackupConfig: BackupConfig{
				ExcludeRelations: []string{"test.table1", "test.table2"},
			},
		},
		{
			name:          "table filter is specified and exclude filter is specified, backup hasn't object filter",
			tableFilter:   "test.table1",
			schemaFilter:  "",
			objectFilter:  "",
			excludeFilter: true,
			want:          false,
			BackupConfig:  BackupConfig{},
		},
		{
			name:          "schema filter is specified and backup has included schema",
			tableFilter:   "",
			schemaFilter:  "test",
			objectFilter:  "include-schema",
			excludeFilter: false,
			want:          true,
			BackupConfig: BackupConfig{
				IncludeSchemas: []string{"test"},
			},
		},
		{
			name:          "schema filter is specified and backup hasn't object filter",
			tableFilter:   "",
			schemaFilter:  "test",
			objectFilter:  "",
			excludeFilter: false,
			want:          false,
			BackupConfig:  BackupConfig{},
		},
		{
			name:          "schema filter is specified and exclude filter is specified, backup has exclude schema",
			tableFilter:   "",
			schemaFilter:  "test",
			objectFilter:  "exclude-schema",
			excludeFilter: true,
			want:          true,
			BackupConfig: BackupConfig{
				ExcludeSchemas: []string{"test"},
			},
		},
		{
			name:          "schema filter is specified and exclude filter is specified, backup hasn't object filter",
			tableFilter:   "",
			schemaFilter:  "test",
			objectFilter:  "",
			excludeFilter: true,
			want:          false,
			BackupConfig:  BackupConfig{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.BackupConfig.CheckObjectFilteringExists(tt.tableFilter, tt.schemaFilter, tt.objectFilter, tt.excludeFilter)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestIsInProgress(t *testing.T) {
	tests := []struct {
		name   string
		config BackupConfig
		want   bool
	}{
		{
			name:   "Test in progress status",
			config: BackupConfig{Status: BackupStatusInProgress},
			want:   true,
		},
		{
			name:   "Test success status",
			config: BackupConfig{Status: BackupStatusSuccess},
			want:   false,
		},
		{
			name:   "Test failure status",
			config: BackupConfig{Status: BackupStatusFailure},
			want:   false,
		},
		{
			name:   "Test empty status",
			config: BackupConfig{Status: ""},
			want:   false,
		},
		{
			name:   "Test unknown status",
			config: BackupConfig{Status: "unknown"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsInProgress()
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}
