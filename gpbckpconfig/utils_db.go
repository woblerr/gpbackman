package gpbckpconfig

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/greenplum-db/gpbackup/history"
	"github.com/woblerr/gpbackman/textmsg"
)

// OpenHistoryDB opens an existing gpbackup_history.db SQLite database.
func OpenHistoryDB(historyDBPath string) (*sql.DB, error) {
	if _, err := os.Stat(historyDBPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, textmsg.ErrorHistoryDBFileNotFound(historyDBPath)
		}
		return nil, textmsg.ErrorUnableStatHistoryDB(historyDBPath, err)
	}
	db, err := sql.Open("sqlite3", historyDBSQLiteURI(historyDBPath))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func historyDBSQLiteURI(historyDBPath string) string {
	if filepath.IsAbs(historyDBPath) {
		dbURI := url.URL{Scheme: "file", Path: historyDBPath, RawQuery: "mode=rw"}
		return dbURI.String()
	}
	return fmt.Sprintf("file:%s?mode=rw", historyDBPath)
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

func GetBackupNamesBeforeTimestamp(timestamp, databaseName string, historyDB *sql.DB) ([]string, error) {
	return execBackupNamesQuery(getBackupNameBeforeTimestampQuery(timestamp, databaseName), databaseName, historyDB)
}

func GetBackupNamesAfterTimestamp(timestamp, databaseName string, historyDB *sql.DB) ([]string, error) {
	return execBackupNamesQuery(getBackupNameAfterTimestampQuery(timestamp, databaseName), databaseName, historyDB)
}

func GetBackupNamesForCleanBeforeTimestamp(timestamp, databaseName string, historyDB *sql.DB) ([]string, error) {
	return execBackupNamesQuery(getBackupNameForCleanBeforeTimestampQuery(timestamp, databaseName), databaseName, historyDB)
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

// Only active backups, "In progress", deleted and failed statuses - hidden.
func getBackupNameBeforeTimestampQuery(timestamp, databaseName string) string {
	query := fmt.Sprintf(`
SELECT timestamp 
FROM backups 
WHERE timestamp < '%s' 
	AND status != '%s' 
	AND date_deleted IN ('', '%s', '%s') 
`, timestamp, BackupStatusInProgress, DateDeletedPluginFailed, DateDeletedLocalFailed)
	return addDatabaseNamePredicate(query, databaseName) + "ORDER BY timestamp DESC;\n"
}

// Only active backups, "In progress", deleted and failed statuses - hidden.
func getBackupNameAfterTimestampQuery(timestamp, databaseName string) string {
	query := fmt.Sprintf(`
SELECT timestamp 
FROM backups 
WHERE timestamp > '%s' 
	AND status != '%s' 
	AND date_deleted IN ('', '%s', '%s') 
`, timestamp, BackupStatusInProgress, DateDeletedPluginFailed, DateDeletedLocalFailed)
	return addDatabaseNamePredicate(query, databaseName) + "ORDER BY timestamp DESC;\n"
}

// Only deleted backups.
func getBackupNameForCleanBeforeTimestampQuery(timestamp, databaseName string) string {
	query := fmt.Sprintf(`
SELECT timestamp 
FROM backups 
WHERE timestamp < '%s' 
	AND date_deleted NOT IN ('', '%s', '%s', '%s') 
`, timestamp, DateDeletedPluginFailed, DateDeletedLocalFailed, DateDeletedInProgress)
	return addDatabaseNamePredicate(query, databaseName) + "ORDER BY timestamp DESC;\n"
}

func addDatabaseNamePredicate(query, databaseName string) string {
	if databaseName == "" {
		return query
	}
	return query + "\tAND database_name = ?\n"
}

func execBackupNamesQuery(query, databaseName string, historyDB *sql.DB) ([]string, error) {
	if databaseName == "" {
		return execQueryFunc(query, historyDB)
	}
	return execQueryFunc(query, historyDB, databaseName)
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
func CleanBackupsDB(list []string, batchSize int, historyDB *sql.DB) error {
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
	return nil
}

func deleteBackupsFormTableQuery(db, value string) string {
	return fmt.Sprintf(`DELETE FROM %s WHERE timestamp IN (%s);`, db, value)
}

func updateDeleteStatusQuery(timestamp, status string) string {
	return fmt.Sprintf(`UPDATE backups SET date_deleted = '%s' WHERE timestamp = '%s';`, status, timestamp)
}

// Execute a query that returns rows.
func execQueryFunc(query string, historyDB *sql.DB, args ...any) ([]string, error) {
	sqlRow, err := historyDB.Query(query, args...)
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
