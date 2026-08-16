package cmd

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpbackup/history"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

func TestHistoryCleanDatabaseFlag(t *testing.T) {
	flag := historyCleanCmd.Flags().Lookup(databaseFlagName)
	if flag == nil {
		t.Fatalf("Expected %s flag to be registered", databaseFlagName)
	}
	if flag.DefValue != "" {
		t.Errorf("Unexpected default value: %q", flag.DefValue)
	}
	if !strings.Contains(flag.Usage, "specified database") {
		t.Errorf("Flag usage does not describe database filtering: %s", flag.Usage)
	}
	if !strings.Contains(historyCleanCmd.Long, "Without --database") {
		t.Errorf("Command help does not warn about cluster-wide cleanup without --database")
	}
}

func TestHistoryCleanDatabaseFlagRequiresValue(t *testing.T) {
	rootCmd.SetArgs([]string{"history-clean", "--" + databaseFlagName})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected --database without a value to return an error")
	}
	if !strings.Contains(err.Error(), "flag needs an argument") {
		t.Fatalf("Unexpected flag error: %v", err)
	}
}

func TestDoCleanHistoryDatabaseFlagValidation(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name        string
		database    string
		setDatabase bool
		olderThan   bool
		wantExit    bool
	}{
		{name: "Absent database flag"},
		{name: "Non-empty database", database: `"Customer's DB"`, setDatabase: true},
		{name: "Explicit empty database", setDatabase: true, wantExit: true},
		{name: "Older than days", database: "demo", setDatabase: true, olderThan: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDatabase := historyCleanDatabase
			oldCleanBeforeTimestamp := historyCleanBeforeTimestamp
			oldCleanOlderThanDays := historyCleanOlderThanDays
			oldBeforeTimestamp := beforeTimestamp
			oldExecOSExit := execOSExit
			historyCleanDatabase = tt.database
			historyCleanBeforeTimestamp = "20240101120000"
			historyCleanOlderThanDays = 1
			beforeTimestamp = ""
			t.Cleanup(func() {
				historyCleanDatabase = oldDatabase
				historyCleanBeforeTimestamp = oldCleanBeforeTimestamp
				historyCleanOlderThanDays = oldCleanOlderThanDays
				beforeTimestamp = oldBeforeTimestamp
				execOSExit = oldExecOSExit
			})

			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flags.String(beforeTimestampFlagName, "", "")
			flags.Uint(olderThanDaysFlagName, 0, "")
			flags.String(databaseFlagName, "", "")
			if tt.olderThan {
				if err := flags.Set(olderThanDaysFlagName, "1"); err != nil {
					t.Fatalf("Failed to set %s: %v", olderThanDaysFlagName, err)
				}
			} else if err := flags.Set(beforeTimestampFlagName, "20240101120000"); err != nil {
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
			doCleanHistoryFlagValidation(flags)
			if exited != tt.wantExit {
				t.Errorf("Exit status = %v, want %v", exited, tt.wantExit)
			}
			if tt.olderThan && beforeTimestamp == "" {
				t.Error("Expected --older-than-days to calculate a cutoff timestamp")
			}
		})
	}
}

func TestHistoryCleanDBFiltersDatabaseAndDeletesAuxiliaryRows(t *testing.T) {
	db := createHistoryCleanTestDB(t)
	selectedDeleted := historyCleanTestConfig("20240101000000", `"Customer's DB"`, "20240102000000", gpbckpconfig.BackupStatusSuccess)
	otherDeleted := historyCleanTestConfig("20240101010000", `"customer's db"`, "20240102000000", gpbckpconfig.BackupStatusSuccess)
	selectedNew := historyCleanTestConfig("20240103000000", `"Customer's DB"`, "20240104000000", gpbckpconfig.BackupStatusSuccess)
	selectedActive := historyCleanTestConfig("20240101020000", `"Customer's DB"`, "", gpbckpconfig.BackupStatusSuccess)
	selectedFailed := historyCleanTestConfig("20240101030000", `"Customer's DB"`, "", gpbckpconfig.BackupStatusFailure)
	storeHistoryCleanConfigs(t, db, selectedDeleted, otherDeleted, selectedNew, selectedActive, selectedFailed)

	if err := historyCleanDB("20240102000000", `"Customer's DB"`, db); err != nil {
		t.Fatalf("Expected history cleanup to succeed, got: %v", err)
	}

	assertHistoryCleanTimestampExists(t, db, selectedDeleted.Timestamp, false)
	for _, backup := range []history.BackupConfig{otherDeleted, selectedNew, selectedActive, selectedFailed} {
		assertHistoryCleanTimestampExists(t, db, backup.Timestamp, true)
	}
	for _, table := range historyCleanAuxiliaryTables {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE timestamp = ?", selectedDeleted.Timestamp).Scan(&count); err != nil {
			t.Fatalf("Failed to count %s rows: %v", table, err)
		}
		if count != 0 {
			t.Errorf("Expected selected rows to be deleted from %s, found %d", table, count)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE timestamp = ?", otherDeleted.Timestamp).Scan(&count); err != nil {
			t.Fatalf("Failed to count other database %s rows: %v", table, err)
		}
		if count == 0 {
			t.Errorf("Expected other database rows to remain in %s", table)
		}
	}
}

func TestHistoryCleanDBWithoutFilterAndUnknownDatabase(t *testing.T) {
	db := createHistoryCleanTestDB(t)
	first := historyCleanTestConfig("20240101000000", "demo", "20240102000000", gpbckpconfig.BackupStatusSuccess)
	second := historyCleanTestConfig("20240101010000", "other", "20240102000000", gpbckpconfig.BackupStatusSuccess)
	storeHistoryCleanConfigs(t, db, first, second)

	if err := historyCleanDB("20240102000000", "unknown", db); err != nil {
		t.Fatalf("Expected unknown database cleanup to be a no-op, got: %v", err)
	}
	assertHistoryCleanTimestampExists(t, db, first.Timestamp, true)
	assertHistoryCleanTimestampExists(t, db, second.Timestamp, true)

	if err := historyCleanDB("20240102000000", "", db); err != nil {
		t.Fatalf("Expected unfiltered history cleanup to succeed, got: %v", err)
	}
	assertHistoryCleanTimestampExists(t, db, first.Timestamp, false)
	assertHistoryCleanTimestampExists(t, db, second.Timestamp, false)
}

var historyCleanAuxiliaryTables = []string{
	"backups",
	"restore_plans",
	"restore_plan_tables",
	"exclude_relations",
	"exclude_schemas",
	"include_relations",
	"include_schemas",
}

func createHistoryCleanTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testhelper.SetupTestLogger()

	db, err := history.InitializeHistoryDatabase(filepath.Join(t.TempDir(), "gpbackup_history.db"))
	if err != nil {
		t.Fatalf("Failed to initialize history database: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = OFF;"); err != nil {
		t.Fatalf("Failed to disable foreign keys for cleanup test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func historyCleanTestConfig(timestamp, databaseName, dateDeleted, status string) history.BackupConfig {
	return history.BackupConfig{
		Timestamp:        timestamp,
		DatabaseName:     databaseName,
		DateDeleted:      dateDeleted,
		Status:           status,
		ExcludeRelations: []string{"public.excluded_relation"},
		ExcludeSchemas:   []string{"excluded_schema"},
		IncludeRelations: []string{"public.included_relation"},
		IncludeSchemas:   []string{"included_schema"},
		RestorePlan: []history.RestorePlanEntry{{
			Timestamp: "20230101000000",
			TableFQNs: []string{"public.restored_table"},
		}},
	}
}

func storeHistoryCleanConfigs(t *testing.T, db *sql.DB, configs ...history.BackupConfig) {
	t.Helper()
	for i := range configs {
		if err := history.StoreBackupHistory(db, &configs[i]); err != nil {
			t.Fatalf("Failed to store backup %s: %v", configs[i].Timestamp, err)
		}
	}
}

func assertHistoryCleanTimestampExists(t *testing.T, db *sql.DB, timestamp string, want bool) {
	t.Helper()
	for _, table := range historyCleanAuxiliaryTables {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE timestamp = ?", timestamp).Scan(&count); err != nil {
			t.Fatalf("Failed to count %s rows: %v", table, err)
		}
		if (count > 0) != want {
			t.Errorf("Rows for %s in %s exist = %v, want %v", timestamp, table, count > 0, want)
		}
	}
}
