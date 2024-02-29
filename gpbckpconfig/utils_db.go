package gpbckpconfig

import (
	"database/sql"
	"fmt"
	"strings"

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
	hBackupData, err := history.GetBackupConfig(backupName, hDB)
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

func GetBackupNamesForCleanBeforeTimestamp(timestamp string, cleanD bool, historyDB *sql.DB) ([]string, error) {
	return execQueryFunc(getBackupNameForCleanBeforeTimestampQuery(timestamp, cleanD), historyDB)
}

func getBackupNameQuery(showD, showF bool) string {
	orderBy := "ORDER BY timestamp DESC;"
	getBackupsQuery := "SELECT timestamp FROM backups"
	switch {
	// Displaying all backups (active, deleted, failed)
	case showD && showF:
		getBackupsQuery = fmt.Sprintf("%s %s", getBackupsQuery, orderBy)
	// Displaying only active and deleted backups; failed - hidden.
	case showD && !showF:
		getBackupsQuery = fmt.Sprintf("%s WHERE status != '%s' %s", getBackupsQuery, BackupStatusFailure, orderBy)
	// Displaying only active and failed backups; deleted - hidden.
	case !showD && showF:
		getBackupsQuery = fmt.Sprintf("%s WHERE date_deleted IN ('', '%s', '%s', '%s') %s", getBackupsQuery, DateDeletedInProgress, DateDeletedPluginFailed, DateDeletedLocalFailed, orderBy)
	// Displaying only active backups or backups with deletion status "In progress", deleted and failed - hidden.
	default:
		getBackupsQuery = fmt.Sprintf("%s WHERE status != '%s' AND date_deleted IN ('', '%s', '%s', '%s') %s", getBackupsQuery, BackupStatusFailure, DateDeletedInProgress, DateDeletedPluginFailed, DateDeletedLocalFailed, orderBy)
	}
	return getBackupsQuery
}

func getBackupDependenciesQuery(backupName string) string {
	return fmt.Sprintf(`
SELECT timestamp 
FROM restore_plans
WHERE timestamp != '%s'
	AND restore_plan_timestamp = '%s'
ORDER BY timestamp DESC;
`, backupName, backupName)
}

// Only active backups,  "In progress", deleted and failed  statuses - hidden.
func getBackupNameBeforeTimestampQuery(timestamp string) string {
	return fmt.Sprintf(`
SELECT timestamp 
FROM backups 
WHERE timestamp < '%s' 
	AND status != '%s' 
	AND date_deleted IN ('', '%s', '%s') 
ORDER BY timestamp DESC;
`, timestamp, BackupStatusFailure, DateDeletedPluginFailed, DateDeletedLocalFailed)
}

func getBackupNameForCleanBeforeTimestampQuery(timestamp string, cleanD bool) string {
	orderBy := "ORDER BY timestamp DESC;"
	getBackupsQuery := fmt.Sprintf("SELECT timestamp FROM backups WHERE timestamp < '%s'", timestamp)
	switch {
	case cleanD:
		// Return  deleted, failed backup.
		getBackupsQuery = fmt.Sprintf("%s AND (status = '%s' OR date_deleted NOT IN ('', '%s', '%s', '%s')) %s", getBackupsQuery, BackupStatusFailure, DateDeletedPluginFailed, DateDeletedLocalFailed, DateDeletedInProgress, orderBy)
	default:
		// Return failed backups.
		getBackupsQuery = fmt.Sprintf("%s AND status = '%s' %s", getBackupsQuery, BackupStatusFailure, orderBy)
	}
	return getBackupsQuery
}

// UpdateDeleteStatus Updates the date_deleted column in the history database.
func UpdateDeleteStatus(backupName, dateDeleted string, historyDB *sql.DB) error {
	err := execStatementFunc(updateDeleteStatusQuery(backupName, dateDeleted), historyDB)
	if err != nil {
		return err
	}
	return nil
}

// CleanBackupsDB cleans the backup history database by deleting backups based on the given list of backup names.
func CleanBackupsDB(list []string, batchSize int, cleanD bool, historyDB *sql.DB) error {
	for i := 0; i < len(list); i += batchSize {
		end := i + batchSize
		if end > len(list) {
			end = len(list)
		}
		batchIDs := list[i:end]
		idStr := "'" + strings.Join(batchIDs, "','") + "'"
		err := execStatementFunc(deleteBackupsFormTableQuery("backups", idStr), historyDB)
		if err != nil {
			return err
		}
		if cleanD {
			err = execStatementFunc(deleteBackupsFormTableQuery("restore_plans", idStr), historyDB)
			if err != nil {
				return err
			}
			err = execStatementFunc(deleteBackupsFormTableQuery("restore_plan_tables", idStr), historyDB)
			if err != nil {
				return err
			}
			err = execStatementFunc(deleteBackupsFormTableQuery("exclude_relations", idStr), historyDB)
			if err != nil {
				return err
			}
			err = execStatementFunc(deleteBackupsFormTableQuery("exclude_schemas", idStr), historyDB)
			if err != nil {
				return err
			}
			err = execStatementFunc(deleteBackupsFormTableQuery("include_relations", idStr), historyDB)
			if err != nil {
				return err
			}
			err = execStatementFunc(deleteBackupsFormTableQuery("include_schemas", idStr), historyDB)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteBackupsFormTableQuery(db, value string) string {
	return fmt.Sprintf(`DELETE FROM %s WHERE timestamp IN (%s);`, db, value)
}

func updateDeleteStatusQuery(timestamp, status string) string {
	return fmt.Sprintf(`UPDATE backups SET date_deleted = '%s' WHERE timestamp = '%s';`, status, timestamp)
}

// Execute a query that returns rows.
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

// Execute a query that doesn't return rows.
func execStatementFunc(query string, historyDB *sql.DB) error {
	tx, err := historyDB.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	err = tx.Commit()
	return err
}
