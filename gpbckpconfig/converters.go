package gpbckpconfig

import (
	"github.com/greenplum-db/gpbackup/history"
)

// Converters are necessary to compatibility with older gpbackup versions.
// If any field will be added or deleted in the new version, this will not lead to a breakdown of the entire process.

// ConvertFromHistoryBackupConfig converts a history.BackupConfig to a BackupConfig.
// It maps the fields from the history.BackupConfig struct to the BackupConfig struct.
func ConvertFromHistoryBackupConfig(hBackupConfig history.BackupConfig) BackupConfig {
	return BackupConfig{
		BackupDir:             hBackupConfig.BackupDir,
		BackupVersion:         hBackupConfig.BackupVersion,
		Compressed:            hBackupConfig.Compressed,
		CompressionType:       hBackupConfig.CompressionType,
		DatabaseName:          hBackupConfig.DatabaseName,
		DatabaseVersion:       hBackupConfig.DatabaseVersion,
		DataOnly:              hBackupConfig.DataOnly,
		DateDeleted:           hBackupConfig.DateDeleted,
		ExcludeRelations:      hBackupConfig.ExcludeRelations,
		ExcludeSchemaFiltered: hBackupConfig.ExcludeSchemaFiltered,
		ExcludeSchemas:        hBackupConfig.ExcludeSchemas,
		ExcludeTableFiltered:  hBackupConfig.ExcludeTableFiltered,
		IncludeRelations:      hBackupConfig.IncludeRelations,
		IncludeSchemaFiltered: hBackupConfig.IncludeSchemaFiltered,
		IncludeSchemas:        hBackupConfig.IncludeSchemas,
		IncludeTableFiltered:  hBackupConfig.IncludeTableFiltered,
		Incremental:           hBackupConfig.Incremental,
		LeafPartitionData:     hBackupConfig.LeafPartitionData,
		MetadataOnly:          hBackupConfig.MetadataOnly,
		Plugin:                hBackupConfig.Plugin,
		PluginVersion:         hBackupConfig.PluginVersion,
		RestorePlan:           convertFromHistoryRestorePlan(hBackupConfig.RestorePlan),
		SingleDataFile:        hBackupConfig.SingleDataFile,
		Timestamp:             hBackupConfig.Timestamp,
		EndTime:               hBackupConfig.EndTime,
		WithoutGlobals:        hBackupConfig.WithoutGlobals,
		WithStatistics:        hBackupConfig.WithStatistics,
		Status:                hBackupConfig.Status,
	}
}

// convertFromHistoryRestorePlan converts a slice of history.RestorePlanEntry to a slice of RestorePlanEntry.
// It iterates over the input slice and calls convertFromHistoryRestorePlanEntry for each entry.
func convertFromHistoryRestorePlan(hRestorePlan []history.RestorePlanEntry) []RestorePlanEntry {
	restorePlan := make([]RestorePlanEntry, len(hRestorePlan))
	for _, hRestorePlanEntry := range hRestorePlan {
		restorePlan = append(restorePlan, convertFromHistoryRestorePlanEntry(hRestorePlanEntry))
	}
	return restorePlan
}

// convertFromHistoryRestorePlanEntry converts a history.restorePlanEntry to a restorePlanEntry.
func convertFromHistoryRestorePlanEntry(hRestorePlanEntry history.RestorePlanEntry) RestorePlanEntry {
	return RestorePlanEntry{
		Timestamp: hRestorePlanEntry.Timestamp,
		TableFQNs: hRestorePlanEntry.TableFQNs,
	}
}

// ConvertToHistoryBackupConfig converts a BackupConfig to a history.BackupConfig.
// It maps the fields from the BackupConfig struct to the history.BackupConfig struct.
func ConvertToHistoryBackupConfig(backupConfig BackupConfig) history.BackupConfig {
	return history.BackupConfig{
		BackupDir:             backupConfig.BackupDir,
		BackupVersion:         backupConfig.BackupVersion,
		Compressed:            backupConfig.Compressed,
		CompressionType:       backupConfig.CompressionType,
		DatabaseName:          backupConfig.DatabaseName,
		DatabaseVersion:       backupConfig.DatabaseVersion,
		DataOnly:              backupConfig.DataOnly,
		DateDeleted:           backupConfig.DateDeleted,
		ExcludeRelations:      backupConfig.ExcludeRelations,
		ExcludeSchemaFiltered: backupConfig.ExcludeSchemaFiltered,
		ExcludeSchemas:        backupConfig.ExcludeSchemas,
		ExcludeTableFiltered:  backupConfig.ExcludeTableFiltered,
		IncludeRelations:      backupConfig.IncludeRelations,
		IncludeSchemaFiltered: backupConfig.IncludeSchemaFiltered,
		IncludeSchemas:        backupConfig.IncludeSchemas,
		IncludeTableFiltered:  backupConfig.IncludeTableFiltered,
		Incremental:           backupConfig.Incremental,
		LeafPartitionData:     backupConfig.LeafPartitionData,
		MetadataOnly:          backupConfig.MetadataOnly,
		Plugin:                backupConfig.Plugin,
		PluginVersion:         backupConfig.PluginVersion,
		RestorePlan:           convertToHistoryRestorePlan(backupConfig.RestorePlan),
		SingleDataFile:        backupConfig.SingleDataFile,
		Timestamp:             backupConfig.Timestamp,
		EndTime:               backupConfig.EndTime,
		WithoutGlobals:        backupConfig.WithoutGlobals,
		WithStatistics:        backupConfig.WithStatistics,
		Status:                backupConfig.Status,
	}
}

// convertToHistoryRestorePlan converts a slice of RestorePlanEntry to a slice of history.RestorePlanEntry.
// It iterates over the input slice and calls convertToHistoryRestorePlanEntry for each entry.
func convertToHistoryRestorePlan(restorePlan []RestorePlanEntry) []history.RestorePlanEntry {
	hRestorePlan := make([]history.RestorePlanEntry, len(restorePlan))
	for _, restorePlanEntry := range restorePlan {
		hRestorePlan = append(hRestorePlan, convertToHistoryRestorePlanEntry(restorePlanEntry))
	}
	return hRestorePlan
}

// convertToHistoryRestorePlanEntry converts a restorePlanEntry to a history.restorePlanEntry.
func convertToHistoryRestorePlanEntry(restorePlanEntry RestorePlanEntry) history.RestorePlanEntry {
	return history.RestorePlanEntry{
		Timestamp: restorePlanEntry.Timestamp,
		TableFQNs: restorePlanEntry.TableFQNs,
	}
}
