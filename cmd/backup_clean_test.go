package cmd

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/spf13/pflag"
)

func TestBackupCleanDatabaseFlag(t *testing.T) {
	flag := backupCleanCmd.Flags().Lookup(databaseFlagName)
	if flag == nil {
		t.Fatalf("Expected %s flag to be registered", databaseFlagName)
	}
	if flag.DefValue != "" {
		t.Errorf("Unexpected default value: %q", flag.DefValue)
	}
	if !strings.Contains(flag.Usage, "specified database") {
		t.Errorf("Flag usage does not describe database filtering: %s", flag.Usage)
	}

	if !strings.Contains(backupCleanCmd.Long, "Without --database") {
		t.Errorf("Command help does not warn about cluster-wide cleanup without --database")
	}
}

func TestBackupCleanDatabaseFlagRequiresValue(t *testing.T) {
	rootCmd.SetArgs([]string{"backup-clean", "--" + databaseFlagName})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected --database without a value to return an error")
	}
	if !strings.Contains(err.Error(), "flag needs an argument") {
		t.Fatalf("Unexpected flag error: %v", err)
	}
}

func TestDoCleanBackupDatabaseFlagValidation(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name        string
		database    string
		setDatabase bool
		wantExit    bool
	}{
		{name: "Absent database flag"},
		{name: "Non-empty database", database: `"Customer's DB"`, setDatabase: true},
		{name: "Explicit empty database", database: "", setDatabase: true, wantExit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDatabase := backupCleanDatabase
			oldCleanBeforeTimestamp := backupCleanBeforeTimestamp
			oldBeforeTimestamp := beforeTimestamp
			oldAfterTimestamp := afterTimestamp
			oldExecOSExit := execOSExit
			backupCleanDatabase = tt.database
			backupCleanBeforeTimestamp = "20240101120000"
			beforeTimestamp = ""
			afterTimestamp = ""
			t.Cleanup(func() {
				backupCleanDatabase = oldDatabase
				backupCleanBeforeTimestamp = oldCleanBeforeTimestamp
				beforeTimestamp = oldBeforeTimestamp
				afterTimestamp = oldAfterTimestamp
				execOSExit = oldExecOSExit
			})

			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flags.String(beforeTimestampFlagName, "", "")
			flags.String(databaseFlagName, "", "")
			if err := flags.Set(beforeTimestampFlagName, "20240101120000"); err != nil {
				t.Fatalf("Failed to set %s: %v", beforeTimestampFlagName, err)
			}
			if tt.setDatabase {
				if err := flags.Set(databaseFlagName, tt.database); err != nil {
					t.Fatalf("Failed to set %s: %v", databaseFlagName, err)
				}
			}

			exited := false
			execOSExit = func(code int) {
				if code != exitErrorCode {
					t.Errorf("Unexpected exit code: %d", code)
				}
				exited = true
			}
			doCleanBackupFlagValidation(flags)
			if exited != tt.wantExit {
				t.Errorf("Exit status = %v, want %v", exited, tt.wantExit)
			}
		})
	}
}

func TestFetchBackupNamesForDeletionFiltersDatabase(t *testing.T) {
	db := createBackupCleanTestDB(t)
	for _, backup := range []struct {
		timestamp string
		database  string
	}{
		{"20240101090000", "demo"},
		{"20240101100000", `"Customer's DB"`},
		{"20240101110000", `"customer's db"`},
		{"20240101130000", `"Customer's DB"`},
	} {
		if _, err := db.Exec(`INSERT INTO backups (timestamp, database_name, status, date_deleted) VALUES (?, ?, 'Success', '')`, backup.timestamp, backup.database); err != nil {
			t.Fatalf("Failed to seed backup %s: %v", backup.timestamp, err)
		}
	}

	tests := []struct {
		name            string
		beforeTimestamp string
		afterTimestamp  string
		database        string
		want            []string
	}{
		{
			name:            "Before timestamp exact database",
			beforeTimestamp: "20240101120000",
			database:        `"Customer's DB"`,
			want:            []string{"20240101100000"},
		},
		{
			name:           "After timestamp exact database",
			afterTimestamp: "20240101120000",
			database:       `"Customer's DB"`,
			want:           []string{"20240101130000"},
		},
		{
			name:            "Quoted database name remains distinct",
			beforeTimestamp: "20240101120000",
			database:        `"customer's db"`,
			want:            []string{"20240101110000"},
		},
		{
			name:            "No database filter includes all databases",
			beforeTimestamp: "20240101120000",
			want:            []string{"20240101110000", "20240101100000", "20240101090000"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchBackupNamesForDeletion(tt.beforeTimestamp, tt.afterTimestamp, tt.database, db)
			if err != nil {
				t.Fatalf("Expected backup selection to succeed, got: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Backup names = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackupCleanDatabaseFilterUnknownDatabaseIsNoOp(t *testing.T) {
	testhelper.SetupTestLogger()
	db := createBackupCleanTestDB(t)

	if _, err := backupCleanDBLocal(false, "20240101120000", "", "unknown", "", 1, db); err != nil {
		t.Fatalf("Expected local cleanup no-op to succeed, got: %v", err)
	}
	if _, err := backupCleanDBPlugin(false, "20240101120000", "", "unknown", "", nil, db); err != nil {
		t.Fatalf("Expected plugin cleanup no-op to succeed, got: %v", err)
	}
}

func createBackupCleanTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "gpbackup_history.db")
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=rwc")
	if err != nil {
		t.Fatalf("Failed to open temporary history database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE backups (timestamp TEXT, database_name TEXT, status TEXT, date_deleted TEXT)`); err != nil {
		t.Fatalf("Failed to create backups table: %v", err)
	}
	return db
}
