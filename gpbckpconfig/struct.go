package gpbckpconfig

import (
	"errors"
	"time"

	"github.com/greenplum-db/gp-common-go-libs/operating"
	"github.com/greenplum-db/gpbackup/utils"
	"github.com/nightlyone/lockfile"
	"gopkg.in/yaml.v2"
)

type History struct {
	BackupConfigs []BackupConfig `yaml:"backupconfigs"`
}

type BackupConfig struct {
	BackupDir             string             `yaml:"backupdir"`
	BackupVersion         string             `yaml:"backupversion"`
	Compressed            bool               `yaml:"compressed"`
	CompressionType       string             `yaml:"compressiontype"`
	DatabaseName          string             `yaml:"databasename"`
	DatabaseVersion       string             `yaml:"databaseversion"`
	DataOnly              bool               `yaml:"dataonly"`
	DateDeleted           string             `yaml:"datedeleted"`
	ExcludeRelations      []string           `yaml:"excluderelations"`
	ExcludeSchemaFiltered bool               `yaml:"excludeschemafiltered"`
	ExcludeSchemas        []string           `yaml:"excludeschemas"`
	ExcludeTableFiltered  bool               `yaml:"excludetablefiltered"`
	IncludeRelations      []string           `yaml:"includerelations"`
	IncludeSchemaFiltered bool               `yaml:"includeschemafiltered"`
	IncludeSchemas        []string           `yaml:"includeschemas"`
	IncludeTableFiltered  bool               `yaml:"includetablefiltered"`
	Incremental           bool               `yaml:"incremental"`
	LeafPartitionData     bool               `yaml:"leafpartitiondata"`
	MetadataOnly          bool               `yaml:"metadataonly"`
	Plugin                string             `yaml:"plugin"`
	PluginVersion         string             `yaml:"pluginversion"`
	RestorePlan           []RestorePlanEntry `yaml:"restoreplan"`
	SingleDataFile        bool               `yaml:"singledatafile"`
	Timestamp             string             `yaml:"timestamp"`
	EndTime               string             `yaml:"endtime"`
	WithoutGlobals        bool               `yaml:"withoutgoals"`
	WithStatistics        bool               `yaml:"withstatistics"`
	Status                string             `yaml:"status"`
}

type RestorePlanEntry struct {
	Timestamp string   `yaml:"timestamp"`
	TableFQNs []string `yaml:"tablefqdn"`
}

const (
	Layout     = "20060102150405"
	DateFormat = "Mon Jan 02 2006 15:04:05"
	// Backup types.
	BackupTypeFull         = "full"
	BackupTypeIncremental  = "incremental"
	BackupTypeDataOnly     = "data-only"
	BackupTypeMetadataOnly = "metadata-only"
	// Backup statuses.
	BackupStatusSuccess = "Success"
	BackupStatusFailure = "Failure"
	// Object filtering types.
	objectFilteringIncludeSchema = "include-schema"
	objectFilteringExcludeSchema = "exclude-schema"
	objectFilteringIncludeTable  = "include-table"
	objectFilteringExcludeTable  = "exclude-table"
	// Date deleted types.
	DateDeletedInProgress   = "In progress"
	DateDeletedPluginFailed = "Plugin Backup Delete Failed"
	DateDeletedLocalFailed  = "Local Delete Failed"
	// BackupS3Plugin S3 plugin names.
	BackupS3Plugin = "gpbackup_s3_plugin"
)

// GetBackupType Get backup type.
// The value is calculated, based on:
//   - full - contains user data, all global and local metadata for the database;
//   - incremental – contains user data, all global and local metadata changed since a previous full backup;
//   - metadata-only – contains only global and local metadata for the database;
//   - data-only – contains only user data from the database.
//
// For gpbackup you cannot combine --data-only or --metadata-only with --incremental (see docs).
// So these options cannot be set at the same time.
// If not one of the --data-only, --metadata-only and --incremental flags is not set,
// the full value is returned.
// But if the --data-only flag set, or it's full backup, but there are no tables in backup set contain data,
// the metadata-only value is returned.
// See https://github.com/greenplum-db/gpbackup/blob/b061a47b673238439442340e66ca57d896edacd5/backup/backup.go#L127-L129
// In all other cases, an error is returned.
func (backupConfig BackupConfig) GetBackupType() (string, error) {
	switch {
	case !(backupConfig.Incremental || backupConfig.DataOnly || backupConfig.MetadataOnly):
		return BackupTypeFull, nil
	case backupConfig.Incremental && !(backupConfig.DataOnly || backupConfig.MetadataOnly):
		return BackupTypeIncremental, nil
	case backupConfig.DataOnly && !(backupConfig.Incremental || backupConfig.MetadataOnly):
		return BackupTypeDataOnly, nil
	case backupConfig.MetadataOnly && !(backupConfig.Incremental):
		return BackupTypeMetadataOnly, nil
	default:
		return "", errors.New("backup type does not match any of the available values")
	}
}

// GetObjectFilteringInfo Get object filtering information.
// The value is calculated, base on whether at least one of the flags was specified:
//   - include-schema – at least one "--include-schema" option was specified;
//   - exclude-schema – at least one "--exclude-schema" option was specified;
//   - include-table – at least one "--include-table" option was specified;
//   - exclude-table – at least one "--exclude-table" option was specified;
//   - "" - no options was specified.
//
// For gpbackup only one type of filters can be used (see docs).
// So these options cannot be set at the same time.
// If not one of these flags is not set,
// the "" value is returned.
// In all other cases, an error is returned.
func (backupConfig BackupConfig) GetObjectFilteringInfo() (string, error) {
	switch {
	case backupConfig.IncludeSchemaFiltered && !(backupConfig.ExcludeSchemaFiltered ||
		backupConfig.IncludeTableFiltered ||
		backupConfig.ExcludeTableFiltered):
		return objectFilteringIncludeSchema, nil
	case backupConfig.ExcludeSchemaFiltered && !(backupConfig.IncludeSchemaFiltered ||
		backupConfig.IncludeTableFiltered ||
		backupConfig.ExcludeTableFiltered):
		return objectFilteringExcludeSchema, nil
	case backupConfig.IncludeTableFiltered && !(backupConfig.IncludeSchemaFiltered ||
		backupConfig.ExcludeSchemaFiltered ||
		backupConfig.ExcludeTableFiltered):
		return objectFilteringIncludeTable, nil
	case backupConfig.ExcludeTableFiltered && !(backupConfig.IncludeSchemaFiltered ||
		backupConfig.ExcludeSchemaFiltered ||
		backupConfig.IncludeTableFiltered):
		return objectFilteringExcludeTable, nil
	case !(backupConfig.ExcludeTableFiltered ||
		backupConfig.IncludeSchemaFiltered ||
		backupConfig.ExcludeSchemaFiltered ||
		backupConfig.IncludeTableFiltered):
		return "", nil
	default:
		return "", errors.New("backup filtering type does not match any of the available values")
	}
}

// GetBackupDate Get backup date.
// If an error occurs when parsing the date, the empty string and error are returned.
func (backupConfig BackupConfig) GetBackupDate() (string, error) {
	var date string
	t, err := time.Parse(Layout, backupConfig.Timestamp)
	if err != nil {
		return date, err
	}
	date = t.Format(DateFormat)
	return date, nil
}

// GetBackupDuration Get backup duration in seconds.
// If an error occurs when parsing the date, the zero duration and error are returned.
func (backupConfig BackupConfig) GetBackupDuration() (float64, error) {
	var zeroDuration float64
	startTime, err := time.Parse(Layout, backupConfig.Timestamp)
	if err != nil {
		return zeroDuration, err
	}
	endTime, err := time.Parse(Layout, backupConfig.EndTime)
	if err != nil {
		return zeroDuration, err
	}
	return endTime.Sub(startTime).Seconds(), nil
}

// GetBackupDateDeleted Get backup deletion date or backup deletion status.
// The possible values are:
//   - In progress - if the value is set to "In progress";
//   - Plugin Backup Delete Failed - if the value is set to "Plugin Backup Delete Failed";
//   - Local Delete Failed - if the value is set to "Local Delete Failed";
//   - "" - if backup is active;
//   - date  in format "Mon Jan 02 2006 15:04:05" - if backup is deleted and deletion timestamp is set.
//
// In all other cases, an error is returned.
func (backupConfig BackupConfig) GetBackupDateDeleted() (string, error) {
	switch backupConfig.DateDeleted {
	case "", DateDeletedInProgress, DateDeletedPluginFailed, DateDeletedLocalFailed:
		return backupConfig.DateDeleted, nil
	default:
		t, err := time.Parse(Layout, backupConfig.DateDeleted)
		if err != nil {
			return backupConfig.DateDeleted, err
		}
		return t.Format(DateFormat), nil
	}
}

// IsSuccess Check backup status.
// Returns:
//   - true  - if backup is successful,
//   - false - false if backup is not successful.
//
// In all other cases, an error is returned.
func (backupConfig BackupConfig) IsSuccess() (bool, error) {
	switch backupConfig.Status {
	case BackupStatusSuccess:
		return true, nil
	case BackupStatusFailure:
		return false, nil
	default:
		return false, errors.New("backup status does not match any of the available values")
	}
}

// IsLocal Check if the backup in local or in plugin storage.
// Returns:
//   - true  - if the backup in local storage (plugin field is empty);
//   - false - if the backup in plugin storage (plugin field is not empty).
func (backupConfig BackupConfig) IsLocal() bool {
	return backupConfig.Plugin == ""
}

// GetReportFilePathPlugin Return path to report file name for specific plugin.
// If custom report path is set, it is returned.
// Otherwise, the path from plugin is returned.
func (backupConfig BackupConfig) GetReportFilePathPlugin(customReportPath string, pluginOptions map[string]string) (string, error) {
	if customReportPath != "" {
		return backupPluginCustomReportPath(backupConfig.Timestamp, customReportPath), nil
	}
	// In future another plugins may be added.
	switch backupConfig.Plugin {
	case BackupS3Plugin:
		return backupS3PluginReportPath(backupConfig.Timestamp, pluginOptions)
	default:
		// nothing to do
	}
	return "", errors.New("the path to the report is not specified")
}

// CheckObjectFilteringExists checks if the object filtering exists in the backup.
//
// This function is responsible for determining whether table or schema filtering exists in the backup, and if so, whether the specified filter type is being used.
// Returns:
//   - true - if table or schema filtering exists in the backup or no filters are specified;
//   - false - if table or schema filtering does not exists in the backup.
func (backupConfig BackupConfig) CheckObjectFilteringExists(tableFilter, schemaFilter, objectFilter string, excludeFilter bool) bool {
	switch {
	case tableFilter != "" && !excludeFilter:
		if objectFilter == objectFilteringIncludeTable {
			return searchFilter(backupConfig.IncludeRelations, tableFilter)
		}
		return false
	case tableFilter != "" && excludeFilter:
		if objectFilter == objectFilteringExcludeTable {
			return searchFilter(backupConfig.ExcludeRelations, tableFilter)
		}
		return false
	case schemaFilter != "" && !excludeFilter:
		if objectFilter == objectFilteringIncludeSchema {
			return searchFilter(backupConfig.IncludeSchemas, schemaFilter)
		}
		return false
	case schemaFilter != "" && excludeFilter:
		if objectFilter == objectFilteringExcludeSchema {
			return searchFilter(backupConfig.ExcludeSchemas, schemaFilter)
		}
		return false
	default:
		return true
	}
}

func (history *History) FindBackupConfig(timestamp string) (int, BackupConfig, error) {
	for idx, backupConfig := range history.BackupConfigs {
		if backupConfig.Timestamp == timestamp {
			return idx, backupConfig, nil
		}
	}
	return 0, BackupConfig{}, errors.New("backup timestamp doesn't match any existing backups")
}

func (history *History) FindBackupConfigDependencies(timestamp string, stopPosition int) []string {
	dependentBackups := make([]string, 0, stopPosition+1)
	for i := 0; i < stopPosition; i++ {
		dependentBackupConfig := history.BackupConfigs[i]
		if len(dependentBackupConfig.RestorePlan) > 0 {
			for _, ts := range dependentBackupConfig.RestorePlan {
				if ts.Timestamp == timestamp {
					dependentBackups = append(dependentBackups, dependentBackupConfig.Timestamp)
				}
			}
		} else {
			continue
		}
	}
	return dependentBackups
}

func (history *History) UpdateBackupConfigDateDeleted(timestamp, dataDeleted string) error {
	for idx, backupConfig := range history.BackupConfigs {
		if backupConfig.Timestamp == timestamp {
			history.BackupConfigs[idx].DateDeleted = dataDeleted
			return nil
		}
	}
	return errors.New("backup timestamp doesn't match any existing backups")
}

func (history *History) UpdateHistoryFile(historyFile string) error {
	lock, err := lockHistoryFile(historyFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = lock.Unlock()
	}()
	err = history.WriteToFileAndMakeReadOnly(historyFile)
	return err
}

func (history *History) WriteToFileAndMakeReadOnly(filename string) error {
	_, err := operating.System.Stat(filename)
	fileExists := err == nil
	if fileExists {
		err = operating.System.Chmod(filename, 0644)
		if err != nil {
			return err
		}
	}
	historyFileContents, err := yaml.Marshal(history)
	if err != nil {
		return err
	}
	return utils.WriteToFileAndMakeReadOnly(filename, historyFileContents)
}

// RemoveMultipleFromHistoryFile Remove multiple backups from history file.
// Idxs is a list of sorted indexes of backups to be removed.
// It iterates over the sorted indices and uses the copy function to shift the elements to the left,
// effectively removing the element at the current index. Finally, it trims the slice to the correct length.
func (history *History) RemoveMultipleFromHistoryFile(idxs []int) {
	j := 0
	for _, i := range idxs {
		if i == len(history.BackupConfigs)-1 || i != len(history.BackupConfigs)-j-1 {
			copy(history.BackupConfigs[i-j:], history.BackupConfigs[i-j+1:])
			j++
		}
	}
	history.BackupConfigs = history.BackupConfigs[:len(history.BackupConfigs)-j]
}

func lockHistoryFile(historyFile string) (lockfile.Lockfile, error) {
	lock, err := lockfile.New(historyFile + ".lck")
	if err != nil {
		return lock, err
	}
	err = lock.TryLock()
	for err != nil {
		time.Sleep(60 * time.Millisecond)
		err = lock.TryLock()
	}
	return lock, err
}
