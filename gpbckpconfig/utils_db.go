package gpbckpconfig

import (
	"database/sql"

	"github.com/greenplum-db/gpbackup/history"
)

// OpenHistoryDB Opens the history backup database.
func OpenHistoryDB(historyDBPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", historyDBPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// GetBackupDataDB Read backup data from history database and return BackupConfig struct.
func GetBackupDataDB(backupName string, hDB *sql.DB) (BackupConfig, error) {
	hBackupData, err := history.GetMainBackupInfo(backupName, hDB)
	if err != nil {
		return BackupConfig{}, err
	}
	return ConvertFromHistoryBackupConfig(hBackupData), nil
}

// GetBackupNamesDB Returns a list of backup names.
func GetBackupNamesDB(showD, showF bool, historyDB *sql.DB) ([]string, error) {
	return execQueryFunc(getBackupNameQuery(showD, showF), historyDB)
}

func GetBackupDependencies(backupName string, historyDB *sql.DB) ([]string, error) {
	return execQueryFunc(getBackupDependenciesQuery(backupName), historyDB)
}

func GetBackupNamesBeforeTimestamp(timestamp string, historyDB *sql.DB) ([]string, error) {
	return execQueryFunc(getBackupNameBeforeTimestampQuery(timestamp), historyDB)
}

func getBackupNameQuery(showD, showF bool) string {
	orderBy := " ORDER BY timestamp DESC;"
	getBackupsQuery := "SELECT timestamp FROM backups"
	switch {
	// Displaying all backups (active, deleted, failed)
	case showD && showF:
		getBackupsQuery += orderBy
	// Displaying only active and deleted backups; failed - hidden.
	case showD && !showF:
		getBackupsQuery += " WHERE status != '" + backupStatusFailure + "'" + orderBy
	// Displaying only active and failed backups; deleted - hidden.
	case !showD && showF:
		getBackupsQuery += " WHERE date_deleted IN ('', '" +
			DateDeletedInProgress + "', '" +
			DateDeletedPluginFailed + "', '" +
			DateDeletedLocalFailed + "')" + orderBy
	// Displaying only active backups or backups with deletion status "In progress", deleted and failed - hidden.
	default:
		getBackupsQuery += " WHERE status != '" + backupStatusFailure + "'" +
			" AND date_deleted IN ('', '" +
			DateDeletedInProgress + "', '" +
			DateDeletedPluginFailed + "', '" +
			DateDeletedLocalFailed + "')" + orderBy
	}
	return getBackupsQuery
}

func getBackupDependenciesQuery(backupName string) string {
	getDependenciesQuery := `SELECT timestamp FROM restore_plans ` +
		`WHERE timestamp != '` + backupName +
		`' AND restore_plan_timestamp = '` + backupName +
		`' ORDER BY timestamp DESC;`
	return getDependenciesQuery
}

// Only active backups,  "In progress", deleted and failed  statuses - hidden.
func getBackupNameBeforeTimestampQuery(timestamp string) string {
	getBackupBeforeTimestampQuery := `SELECT timestamp FROM backups`
	getBackupBeforeTimestampQuery += " WHERE timestamp < '" + timestamp + "'" +
		" AND status != '" + backupStatusFailure + "'" +
		" AND date_deleted IN ('', '" +
		DateDeletedPluginFailed + "', '" +
		DateDeletedLocalFailed + "') ORDER BY timestamp DESC;"
	return getBackupBeforeTimestampQuery
}

func execQueryFunc(query string, historyDB *sql.DB) ([]string, error) {
	sqlRow, err := historyDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer sqlRow.Close()
	var resultList []string
	for sqlRow.Next() {
		var b string
		err := sqlRow.Scan(&b)
		if err != nil {
			return nil, err
		}
		resultList = append(resultList, b)
	}
	if err := sqlRow.Err(); err != nil {
		return nil, err
	}
	return resultList, nil
}
