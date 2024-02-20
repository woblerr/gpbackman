package gpbckpconfig

import (
	"reflect"
	"testing"

	"github.com/greenplum-db/gpbackup/history"
)

func TestConvertToHistoryBackupConfig(t *testing.T) {
	tests := []struct {
		name   string
		config BackupConfig
		want   history.BackupConfig
	}{
		{
			name: "Test correct converting",
			config: BackupConfig{
				BackupDir:             "/data/backups",
				BackupVersion:         "1.26.0",
				Compressed:            true,
				CompressionType:       "gzip",
				DatabaseName:          "test",
				DatabaseVersion:       "6.23.0",
				DataOnly:              false,
				DateDeleted:           "",
				ExcludeRelations:      []string{},
				ExcludeSchemaFiltered: false,
				ExcludeSchemas:        []string{},
				ExcludeTableFiltered:  false,
				IncludeRelations:      []string{},
				IncludeSchemaFiltered: false,
				IncludeSchemas:        []string{},
				IncludeTableFiltered:  false,
				Incremental:           false,
				LeafPartitionData:     false,
				MetadataOnly:          false,
				Plugin:                "",
				PluginVersion:         "",
				RestorePlan:           []RestorePlanEntry{},
				SingleDataFile:        false,
				Timestamp:             "20230118152654",
				EndTime:               "20230118152656",
				WithoutGlobals:        false,
				WithStatistics:        false,
				Status:                "Success",
			},
			want: history.BackupConfig{
				BackupDir:             "/data/backups",
				BackupVersion:         "1.26.0",
				Compressed:            true,
				CompressionType:       "gzip",
				DatabaseName:          "test",
				DatabaseVersion:       "6.23.0",
				DataOnly:              false,
				DateDeleted:           "",
				ExcludeRelations:      []string{},
				ExcludeSchemaFiltered: false,
				ExcludeSchemas:        []string{},
				ExcludeTableFiltered:  false,
				IncludeRelations:      []string{},
				IncludeSchemaFiltered: false,
				IncludeSchemas:        []string{},
				IncludeTableFiltered:  false,
				Incremental:           false,
				LeafPartitionData:     false,
				MetadataOnly:          false,
				Plugin:                "",
				PluginVersion:         "",
				RestorePlan:           []history.RestorePlanEntry{},
				SingleDataFile:        false,
				Timestamp:             "20230118152654",
				EndTime:               "20230118152656",
				WithoutGlobals:        false,
				WithStatistics:        false,
				Status:                "Success",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConvertToHistoryBackupConfig(tt.config)
			if !reflect.DeepEqual(config, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", config, tt.want)
			}
		})
	}
}

func TestConvertToHistoryRestorePlan(t *testing.T) {
	tests := []struct {
		name string
		plan []RestorePlanEntry
		want []history.RestorePlanEntry
	}{
		{
			name: "Test correct converting",
			plan: []RestorePlanEntry{
				{
					Timestamp: "20220401101430",
					TableFQNs: []string{"table1", "table2"},
				},
				{
					Timestamp: "20220401102430",
					TableFQNs: []string{"table1", "table2"},
				},
			},
			want: []history.RestorePlanEntry{
				{
					Timestamp: "20220401101430",
					TableFQNs: []string{"table1", "table2"},
				},
				{
					Timestamp: "20220401102430",
					TableFQNs: []string{"table1", "table2"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := convertToHistoryRestorePlan(tt.plan)
			if !reflect.DeepEqual(plan, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", plan, tt.want)
			}
		})
	}
}

func TestConvertToHistoryRestorePlanEntry(t *testing.T) {
	tests := []struct {
		name      string
		planEntry RestorePlanEntry
		want      history.RestorePlanEntry
	}{
		{
			name: "Test correct converting",
			planEntry: RestorePlanEntry{
				Timestamp: "20220401102430",
				TableFQNs: []string{"table1", "table2"},
			},
			want: history.RestorePlanEntry{
				Timestamp: "20220401102430",
				TableFQNs: []string{"table1", "table2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planEntry := convertToHistoryRestorePlanEntry(tt.planEntry)
			if !reflect.DeepEqual(planEntry, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", planEntry, tt.want)
			}
		})
	}
}

func TestConvertFromHistoryBackupConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *history.BackupConfig
		want   BackupConfig
	}{
		{
			name: "Test correct converting",
			config: &history.BackupConfig{
				BackupDir:             "/data/backups",
				BackupVersion:         "1.26.0",
				Compressed:            true,
				CompressionType:       "gzip",
				DatabaseName:          "test",
				DatabaseVersion:       "6.23.0",
				DataOnly:              false,
				DateDeleted:           "",
				ExcludeRelations:      []string{},
				ExcludeSchemaFiltered: false,
				ExcludeSchemas:        []string{},
				ExcludeTableFiltered:  false,
				IncludeRelations:      []string{},
				IncludeSchemaFiltered: false,
				IncludeSchemas:        []string{},
				IncludeTableFiltered:  false,
				Incremental:           false,
				LeafPartitionData:     false,
				MetadataOnly:          false,
				Plugin:                "",
				PluginVersion:         "",
				RestorePlan:           []history.RestorePlanEntry{},
				SingleDataFile:        false,
				Timestamp:             "20230118152654",
				EndTime:               "20230118152656",
				WithoutGlobals:        false,
				WithStatistics:        false,
				Status:                "Success",
			},
			want: BackupConfig{
				BackupDir:             "/data/backups",
				BackupVersion:         "1.26.0",
				Compressed:            true,
				CompressionType:       "gzip",
				DatabaseName:          "test",
				DatabaseVersion:       "6.23.0",
				DataOnly:              false,
				DateDeleted:           "",
				ExcludeRelations:      []string{},
				ExcludeSchemaFiltered: false,
				ExcludeSchemas:        []string{},
				ExcludeTableFiltered:  false,
				IncludeRelations:      []string{},
				IncludeSchemaFiltered: false,
				IncludeSchemas:        []string{},
				IncludeTableFiltered:  false,
				Incremental:           false,
				LeafPartitionData:     false,
				MetadataOnly:          false,
				Plugin:                "",
				PluginVersion:         "",
				RestorePlan:           []RestorePlanEntry{},
				SingleDataFile:        false,
				Timestamp:             "20230118152654",
				EndTime:               "20230118152656",
				WithoutGlobals:        false,
				WithStatistics:        false,
				Status:                "Success",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConvertFromHistoryBackupConfig(tt.config)
			if !reflect.DeepEqual(config, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", config, tt.want)
			}
		})
	}
}

func TestConvertFromHistoryRestorePlan(t *testing.T) {
	tests := []struct {
		name string
		plan []history.RestorePlanEntry
		want []RestorePlanEntry
	}{
		{
			name: "Test correct converting",
			plan: []history.RestorePlanEntry{
				{
					Timestamp: "20220401101430",
					TableFQNs: []string{"table1", "table2"},
				},
				{
					Timestamp: "20220401102430",
					TableFQNs: []string{"table1", "table2"},
				},
			},
			want: []RestorePlanEntry{
				{
					Timestamp: "20220401101430",
					TableFQNs: []string{"table1", "table2"},
				},
				{
					Timestamp: "20220401102430",
					TableFQNs: []string{"table1", "table2"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := convertFromHistoryRestorePlan(tt.plan)
			if !reflect.DeepEqual(plan, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", plan, tt.want)
			}
		})
	}
}

func TestConvertFromHistoryRestorePlanEntry(t *testing.T) {
	tests := []struct {
		name      string
		planEntry history.RestorePlanEntry
		want      RestorePlanEntry
	}{
		{
			name: "Test correct converting",
			planEntry: history.RestorePlanEntry{
				Timestamp: "20220401102430",
				TableFQNs: []string{"table1", "table2"},
			},
			want: RestorePlanEntry{
				Timestamp: "20220401102430",
				TableFQNs: []string{"table1", "table2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planEntry := convertFromHistoryRestorePlanEntry(tt.planEntry)
			if !reflect.DeepEqual(planEntry, tt.want) {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", planEntry, tt.want)
			}
		})
	}
}
