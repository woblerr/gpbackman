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
	backupTypeFull         = "full"
	backupTypeIncremental  = "incremental"
	backupTypeDataOnly     = "data-only"
	backupTypeMetadataOnly = "metadata-only"
	// Backup statuses.
	backupStatusSucceed = "Success"
	backupStatusFailed  = "Failure"
	// Object filtering types.
	objectFilteringIncludeSchema = "include-schema"
	objectFilteringExcludeSchema = "exclude-schema"
	objectFilteringIncludeTable  = "include-table"
	objectFilteringExcludeTable  = "exclude-table"
	// Date deleted types.
	dateDeletedInProgress   = "In progress"
	dateDeletedPluginFailed = "Plugin Backup Delete Failed"
	dateDeletedLocalFailed  = "Local Delete Failed"
)

// GetBackupType Get backup type.
// The value is calculated, based on:
//   - full - contains user data, all global and local metadata for the database;
//   - incremental – contains user data, all global and local metadata changed since a previous full backup;
//   - metadata-only – contains only global and local metadata for the database;
//   - data-only – contains only user data from the database.
func (backupConfig BackupConfig) GetBackupType() string {
	var backupType string
	// For gpbackup you cannot combine --data-only or --metadata-only with --incremental (see docs).
	// So these flags cannot be set at the same time.
	switch {
	case backupConfig.Incremental:
		backupType = backupTypeIncremental
	case backupConfig.DataOnly:
		backupType = backupTypeDataOnly
	case backupConfig.MetadataOnly:
		backupType = backupTypeMetadataOnly
	default:
		backupType = backupTypeFull
	}
	return backupType
}

// GetObjectFilteringInfo Get object filtering information.
// The value is calculated, base on whether at least one of the flags was specified:
//   - include-schema – at least one "--include-schema" option was specified;
//   - exclude-schema – at least one "--exclude-schema" option was specified;
//   - include-table – at least one "--include-table" option was specified;
//   - exclude-table – at least one "--exclude-table" option was specified;
//   - "" - no options was specified.
func (backupConfig BackupConfig) GetObjectFilteringInfo() string {
	var objectFiltering string
	switch {
	case backupConfig.IncludeSchemaFiltered:
		objectFiltering = objectFilteringIncludeSchema
	case backupConfig.ExcludeSchemaFiltered:
		objectFiltering = objectFilteringExcludeSchema
	case backupConfig.IncludeTableFiltered:
		objectFiltering = objectFilteringIncludeTable
	case backupConfig.ExcludeTableFiltered:
		objectFiltering = objectFilteringExcludeTable
	default:
		objectFiltering = ""
	}
	return objectFiltering
}

// GetBackupDate Get backup date.
// If an error occurs when parsing the date, the empty string is returned.
func (backupConfig BackupConfig) GetBackupDate() (string, error) {
	var date string
	t, err := time.Parse(Layout, backupConfig.Timestamp)
	if err != nil {
		return date, err
	}
	date = t.Format(DateFormat)
	return date, nil
}

// GetBackupDuration Get backup duration.
// If an error occurs when parsing the date, the zero duration is returned.
func (backupConfig BackupConfig) GetBackupDuration() (float64, error) {
	var zeroDuration float64 = 0
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
func (backupConfig BackupConfig) GetBackupDateDeleted() (string, error) {
	switch backupConfig.DateDeleted {
	case "":
		return backupConfig.DateDeleted, nil
	case dateDeletedInProgress:
		return backupConfig.DateDeleted, nil
	case dateDeletedPluginFailed:
		return backupConfig.DateDeleted, nil
	case dateDeletedLocalFailed:
		return backupConfig.DateDeleted, nil
	default:
		t, err := time.Parse(Layout, backupConfig.DateDeleted)
		if err != nil {
			return backupConfig.DateDeleted, err
		}
		return t.Format(DateFormat), nil
	}
}

func (backupConfig BackupConfig) Failed() bool {
	return backupConfig.Status == backupStatusFailed
}

func (history *History) FindBackupConfig(timestamp string) (BackupConfig, error) {
	for _, backupConfig := range history.BackupConfigs {
		if backupConfig.Timestamp == timestamp && !backupConfig.Failed() {
			return backupConfig, nil
		}
	}
	return BackupConfig{}, errors.New("timestamp doesn't match any existing backups")
}

func (history *History) UpdateBackupConfigDateDeleted(timestamp string, dataDeleted string) error {
	for idx, backupConfig := range history.BackupConfigs {
		if backupConfig.Timestamp == timestamp && !backupConfig.Failed() {
			history.BackupConfigs[idx].DateDeleted = dataDeleted
			return nil
		}
	}
	return errors.New("timestamp doesn't match any existing backups")
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
