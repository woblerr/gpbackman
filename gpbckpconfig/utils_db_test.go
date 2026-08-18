package gpbckpconfig

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	_ "github.com/mattn/go-sqlite3"
	"github.com/woblerr/gpbackman/textmsg"
)

func TestOpenHistoryDBMissingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing_history.db")
	db, err := OpenHistoryDB(missingPath)
	if err == nil {
		t.Fatalf("Expected OpenHistoryDB to fail for a missing file")
	}
	if db != nil {
		t.Fatalf("Expected nil database handle for a missing file")
	}
	if want := textmsg.ErrorHistoryDBFileNotFound(missingPath).Error(); err.Error() != want {
		t.Errorf("Unexpected error:\n%v\nwant:\n%v", err, want)
	}
	if _, statErr := os.Stat(missingPath); !os.IsNotExist(statErr) {
		t.Fatalf("Expected missing history DB to stay absent, stat error: %v", statErr)
	}
}

func TestOpenHistoryDBExistingFile(t *testing.T) {
	historyDBPath := filepath.Join(t.TempDir(), "gpbackup_history.db")
	seedDB, err := sql.Open("sqlite3", "file:"+historyDBPath+"?mode=rwc")
	if err != nil {
		t.Fatalf("Failed to create seed SQLite DB: %v", err)
	}
	if err = seedDB.Ping(); err != nil {
		t.Fatalf("Failed to ping seed SQLite DB: %v", err)
	}
	if err = seedDB.Close(); err != nil {
		t.Fatalf("Failed to close seed SQLite DB: %v", err)
	}

	db, err := OpenHistoryDB(historyDBPath)
	if err != nil {
		t.Fatalf("Expected OpenHistoryDB to open existing DB, got: %v", err)
	}
	if db == nil {
		t.Fatalf("Expected non-nil database handle")
	}
	if err = db.Ping(); err != nil {
		t.Fatalf("Failed to ping opened SQLite DB: %v", err)
	}
	if err = db.Close(); err != nil {
		t.Fatalf("Failed to close opened SQLite DB: %v", err)
	}
}

func TestOpenHistoryDBExistingRelativeFile(t *testing.T) {
	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to read current directory: %v", err)
	}
	if err = os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Failed to switch to temp directory: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(oldWorkingDir); chdirErr != nil {
			t.Fatalf("Failed to restore current directory: %v", chdirErr)
		}
	}()

	seedDB, err := sql.Open("sqlite3", "file:gpbackup_history.db?mode=rwc")
	if err != nil {
		t.Fatalf("Failed to create seed SQLite DB: %v", err)
	}
	if err = seedDB.Ping(); err != nil {
		t.Fatalf("Failed to ping seed SQLite DB: %v", err)
	}
	if err = seedDB.Close(); err != nil {
		t.Fatalf("Failed to close seed SQLite DB: %v", err)
	}

	db, err := OpenHistoryDB("gpbackup_history.db")
	if err != nil {
		t.Fatalf("Expected OpenHistoryDB to open relative DB path, got: %v", err)
	}
	if err = db.Ping(); err != nil {
		t.Fatalf("Failed to ping opened SQLite DB: %v", err)
	}
	if err = db.Close(); err != nil {
		t.Fatalf("Failed to close opened SQLite DB: %v", err)
	}
}

func TestGetBackupNameQuery(t *testing.T) {
	tests := []struct {
		name  string
		showD bool
		showF bool
		want  string
	}{
		{
			name:  "Test show all",
			showD: true,
			showF: true,
			want:  `SELECT timestamp FROM backups ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show deleted",
			showD: true,
			showF: false,
			want:  `SELECT timestamp FROM backups WHERE status != 'Failure' ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show failed",
			showD: false,
			showF: true,
			want:  `SELECT timestamp FROM backups WHERE date_deleted IN ('', 'In progress', 'Plugin Backup Delete Failed', 'Local Delete Failed') ORDER BY timestamp DESC;`,
		},
		{
			name:  "Test show default",
			showD: false,
			showF: false,
			want:  `SELECT timestamp FROM backups WHERE status != 'Failure' AND date_deleted IN ('', 'In progress', 'Plugin Backup Delete Failed', 'Local Delete Failed') ORDER BY timestamp DESC;`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBackupNameQuery(tt.showD, tt.showF); got != tt.want {
				t.Errorf("getBackupNameQuery(%v, %v):\n%v\nwant:\n%v", tt.showD, tt.showF, got, tt.want)
			}
		})
	}
}

func TestGetBackupDependenciesQuery(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		function func(string) string
		want     string
	}{
		{
			name:     "Test getBackupDependenciesQuery",
			value:    "TestBackup",
			function: getBackupDependenciesQuery,
			want: `
SELECT timestamp 
FROM restore_plans
WHERE timestamp != 'TestBackup'
	AND restore_plan_timestamp = 'TestBackup'
ORDER BY timestamp DESC;
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.value); got != tt.want {
				t.Errorf("getBackupDependenciesQuery(%v):\n%v\nwant:\n%v", tt.value, got, tt.want)
			}
		})
	}
}

func TestBackupNameTimestampQueries(t *testing.T) {
	const timestamp = "20240101120000"
	tests := []struct {
		name     string
		function func(string, string) string
		want     string
	}{
		{
			name:     "Before timestamp without database filter",
			function: getBackupNameBeforeTimestampQuery,
			want: "\nSELECT timestamp \n" +
				"FROM backups \n" +
				"WHERE timestamp < '20240101120000' \n" +
				"\tAND status != 'In Progress' \n" +
				"\tAND date_deleted IN ('', 'Plugin Backup Delete Failed', 'Local Delete Failed') \n" +
				"ORDER BY timestamp DESC;\n",
		},
		{
			name:     "After timestamp without database filter",
			function: getBackupNameAfterTimestampQuery,
			want: "\nSELECT timestamp \n" +
				"FROM backups \n" +
				"WHERE timestamp > '20240101120000' \n" +
				"\tAND status != 'In Progress' \n" +
				"\tAND date_deleted IN ('', 'Plugin Backup Delete Failed', 'Local Delete Failed') \n" +
				"ORDER BY timestamp DESC;\n",
		},
		{
			name:     "History clean without database filter",
			function: getBackupNameForCleanBeforeTimestampQuery,
			want: "\nSELECT timestamp \n" +
				"FROM backups \n" +
				"WHERE timestamp < '20240101120000' \n" +
				"\tAND date_deleted NOT IN ('', 'Plugin Backup Delete Failed', 'Local Delete Failed', 'In progress') \n" +
				"ORDER BY timestamp DESC;\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(timestamp, ""); got != tt.want {
				t.Fatalf("Unexpected query:\n%s\nwant:\n%s", got, tt.want)
			}
			if got := tt.function(timestamp, `"customer's db"`); got != tt.want[:len(tt.want)-len("ORDER BY timestamp DESC;\n")]+"\tAND database_name = ?\nORDER BY timestamp DESC;\n" {
				t.Fatalf("Unexpected filtered query:\n%s", got)
			}
		})
	}
}

func TestGetBackupNamesTimestampQueriesBindDatabaseName(t *testing.T) {
	const (
		timestamp    = "20240101120000"
		databaseName = `"customer's db"`
	)
	tests := []struct {
		name     string
		query    string
		function func(string, string, *sql.DB) ([]string, error)
	}{
		{
			name:     "Before timestamp",
			query:    getBackupNameBeforeTimestampQuery(timestamp, databaseName),
			function: GetBackupNamesBeforeTimestamp,
		},
		{
			name:     "After timestamp",
			query:    getBackupNameAfterTimestampQuery(timestamp, databaseName),
			function: GetBackupNamesAfterTimestamp,
		},
		{
			name:     "History clean",
			query:    getBackupNameForCleanBeforeTimestampQuery(timestamp, databaseName),
			function: GetBackupNamesForCleanBeforeTimestamp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create SQL mock: %v", err)
			}
			t.Cleanup(func() { _ = db.Close() })
			mock.ExpectQuery(regexp.QuoteMeta(tt.query)).WithArgs(databaseName).
				WillReturnRows(sqlmock.NewRows([]string{"timestamp"}).AddRow("20240101110000"))

			got, err := tt.function(timestamp, databaseName, db)
			if err != nil {
				t.Fatalf("Expected query to succeed, got: %v", err)
			}
			if len(got) != 1 || got[0] != "20240101110000" {
				t.Fatalf("Unexpected backup names: %v", got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("Unmet SQL expectations: %v", err)
			}
		})
	}
}

func TestGetBackupNamesBeforeTimestampFiltersDatabaseExactly(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "history.db")
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=rwc")
	if err != nil {
		t.Fatalf("Failed to open temporary SQLite DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, execErr := db.Exec(`CREATE TABLE backups (timestamp TEXT, database_name TEXT, status TEXT, date_deleted TEXT)`); execErr != nil {
		t.Fatalf("Failed to create backups table: %v", execErr)
	}
	for _, backup := range [][]string{
		{"20240101110000", `"Customer"`, "Success", ""},
		{"20240101100000", "customer", "Success", ""},
		{"20240101090000", `"customer's db"`, "Success", ""},
	} {
		if _, execErr := db.Exec(`INSERT INTO backups (timestamp, database_name, status, date_deleted) VALUES (?, ?, ?, ?)`, backup[0], backup[1], backup[2], backup[3]); execErr != nil {
			t.Fatalf("Failed to insert backup %q: %v", backup[0], execErr)
		}
	}

	got, err := GetBackupNamesBeforeTimestamp("20240101120000", "customer", db)
	if err != nil {
		t.Fatalf("Expected filtered query to succeed, got: %v", err)
	}
	if len(got) != 1 || got[0] != "20240101100000" {
		t.Fatalf("Expected exact case-sensitive database match, got: %v", got)
	}
}

func TestTextQueryFunctionsArg(t *testing.T) {
	testBackupName := "TestBackup"
	tests := []struct {
		name     string
		value1   string
		value2   string
		function func(string, string) string
		want     string
	}{
		{
			name:     "Test deleteBackupsFormTableQuery",
			value1:   testBackupName,
			value2:   "'20220401102430', '20220401102430'",
			function: deleteBackupsFormTableQuery,
			want:     "DELETE FROM TestBackup WHERE timestamp IN ('20220401102430', '20220401102430');",
		},
		{
			name:     "Test updateDeleteStatusQuery",
			value1:   testBackupName,
			value2:   "20220401102430",
			function: updateDeleteStatusQuery,
			want:     "UPDATE backups SET date_deleted = '20220401102430' WHERE timestamp = 'TestBackup';",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.function(tt.value1, tt.value2); got != tt.want {
				t.Errorf("\nVariables do not match:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
