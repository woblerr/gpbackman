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

// GetBackupNamesDB Returns a list of backup names that are active (completed successfully and not deleted).
func GetBackupNamesDB(showD, showF, sAll bool, historyDB *sql.DB) ([]string, error) {
	backupListRow, err := historyDB.Query(getBackupNameQuery(showD, showF, sAll))
	if err != nil {
		return nil, err
	}
	defer backupListRow.Close()
	var backupList []string
	for backupListRow.Next() {
		var b string
		err := backupListRow.Scan(&b)
		if err != nil {
			return nil, err
		}
		backupList = append(backupList, b)
	}
	if err := backupListRow.Err(); err != nil {
		return nil, err
	}
	return backupList, nil
}

func getBackupNameQuery(showD, showF, sAll bool) string {
	orderBy := " ORDER BY timestamp DESC;"
	getBackupsQuery := "SELECT timestamp FROM backups"
	switch {
	case sAll:
		getBackupsQuery += orderBy
	case showD:
		getBackupsQuery += " WHERE status != '" + backupStatusFailure + "'" +
			" AND date_deleted NOT IN ('', '" +
			DateDeletedInProgress + "', '" +
			DateDeletedPluginFailed + "', '" +
			DateDeletedLocalFailed + "')" + orderBy

	case showF:
		getBackupsQuery += " WHERE status = '" + backupStatusFailure + "'" + orderBy
	default:
		getBackupsQuery += " WHERE status != '" + backupStatusFailure + "'" +
			" AND date_deleted IN ('', '" +
			DateDeletedInProgress + "', '" +
			DateDeletedPluginFailed + "', '" +
			DateDeletedLocalFailed + "')" + orderBy
	}
	return getBackupsQuery
}
