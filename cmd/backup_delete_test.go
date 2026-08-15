package cmd

import (
	"database/sql"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpbackup/history"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

type recordingBackupDeleter struct {
	deleted []string
}

func (d *recordingBackupDeleter) backupDeleteDB(backupName string, _ *sql.DB, _ bool) error {
	d.deleted = append(d.deleted, backupName)
	return nil
}

// fakeExecCommand returns a function that mocks execCommand.
// If failOnRmRf is true, SSH commands containing "rm -rf" will fail.
func fakeExecCommand(failOnRmRf bool) func(string, ...string) *exec.Cmd {
	return func(command string, args ...string) *exec.Cmd {
		if command == "ssh" && len(args) > 0 {
			remoteCmd := args[len(args)-1]
			if failOnRmRf && strings.HasPrefix(remoteCmd, "rm -rf") {
				return exec.Command("false")
			}
		}
		return exec.Command("true")
	}
}

func TestExecuteDeleteBackupOnSegments(t *testing.T) {
	testhelper.SetupTestLogger()
	tests := []struct {
		name         string
		failOnRmRf   bool
		ignoreErrors bool
		wantErr      bool
	}{
		{
			name:         "Delete phase error causes error",
			failOnRmRf:   true,
			ignoreErrors: false,
			wantErr:      true,
		},
		{
			name:         "All phases succeed",
			failOnRmRf:   false,
			ignoreErrors: false,
			wantErr:      false,
		},
		{
			name:         "Delete phase error ignored",
			failOnRmRf:   true,
			ignoreErrors: true,
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origExecCommand := execCommand
			defer func() { execCommand = origExecCommand }()
			execCommand = fakeExecCommand(tt.failOnRmRf)
			configs := []gpbckpconfig.SegmentConfig{
				{ContentID: "0", Hostname: "host1", DataDir: "/data/seg0"},
				{ContentID: "1", Hostname: "host2", DataDir: "/data/seg1"},
			}
			err := executeDeleteBackupOnSegments(
				"/tmp",
				"",
				"20230101010101",
				"gpseg",
				true,
				tt.ignoreErrors,
				configs,
				2,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nexecuteDeleteBackupOnSegments() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestBackupDeleteDBCascadeDeletesSameDatabaseDependency(t *testing.T) {
	db := createBackupDeleteTestDB(t,
		backupDeleteTestConfig("20240101010101", "demo"),
		backupDeleteTestConfig("20240102020202", "demo", "20240101010101"),
	)
	deleter := &recordingBackupDeleter{}

	_, err := backupDeleteDB([]string{"20240101010101"}, true, false, false, false, deleter, db)
	if err != nil {
		t.Fatalf("Expected cascade deletion to succeed, got: %v", err)
	}
	if want := []string{"20240102020202", "20240101010101"}; !reflect.DeepEqual(deleter.deleted, want) {
		t.Errorf("Deleted backups = %v, want %v", deleter.deleted, want)
	}
}

func TestBackupDeleteDBCascadeRejectsCrossDatabaseDependency(t *testing.T) {
	db := createBackupDeleteTestDB(t,
		backupDeleteTestConfig("20240101010101", "demo"),
		backupDeleteTestConfig("20240102020202", "other", "20240101010101"),
	)
	deleter := &recordingBackupDeleter{}

	_, err := backupDeleteDB([]string{"20240101010101"}, true, false, false, false, deleter, db)
	wantErr := textmsg.ErrorBackupDependencyDatabaseMismatch("20240101010101", "demo", "20240102020202", "other")
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("Error = %v, want %v", err, wantErr)
	}
	if len(deleter.deleted) != 0 {
		t.Errorf("Cross-database dependency was passed to deleter: %v", deleter.deleted)
	}
}

func TestBackupDeleteDBCascadeUsesSourceDatabaseForEachBackup(t *testing.T) {
	db := createBackupDeleteTestDB(t,
		backupDeleteTestConfig("20240101010101", "demo"),
		backupDeleteTestConfig("20240102020202", "demo", "20240101010101"),
		backupDeleteTestConfig("20240103030303", "other"),
		backupDeleteTestConfig("20240104040404", "other", "20240103030303"),
	)
	deleter := &recordingBackupDeleter{}

	_, err := backupDeleteDB([]string{"20240101010101", "20240103030303"}, true, false, false, false, deleter, db)
	if err != nil {
		t.Fatalf("Expected independent cascade chains to succeed, got: %v", err)
	}
	if want := []string{"20240102020202", "20240101010101", "20240104040404", "20240103030303"}; !reflect.DeepEqual(deleter.deleted, want) {
		t.Errorf("Deleted backups = %v, want %v", deleter.deleted, want)
	}
}

func TestBackupDeleteDBDependenciesRequireCascade(t *testing.T) {
	db := createBackupDeleteTestDB(t,
		backupDeleteTestConfig("20240101010101", "demo"),
		backupDeleteTestConfig("20240102020202", "demo", "20240101010101"),
	)
	deleter := &recordingBackupDeleter{}

	_, err := backupDeleteDB([]string{"20240101010101"}, false, false, false, false, deleter, db)
	if err == nil || err.Error() != textmsg.ErrorBackupDeleteCascadeOptionError().Error() {
		t.Fatalf("Error = %v, want %v", err, textmsg.ErrorBackupDeleteCascadeOptionError())
	}
	if len(deleter.deleted) != 0 {
		t.Errorf("Backups were deleted without cascade: %v", deleter.deleted)
	}
}

func TestBackupCleanCascadeRejectsCrossDatabaseDependency(t *testing.T) {
	db := createBackupDeleteTestDB(t,
		backupDeleteTestConfig("20240102020202", "demo"),
		backupDeleteTestConfig("20240101010101", "other", "20240102020202"),
	)

	_, err := backupCleanDBLocal(true, "", "20240101500000", "", "", 1, db)
	wantErr := textmsg.ErrorBackupDependencyDatabaseMismatch("20240102020202", "demo", "20240101010101", "other")
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("Error = %v, want %v", err, wantErr)
	}
}

func createBackupDeleteTestDB(t *testing.T, configs ...history.BackupConfig) *sql.DB {
	t.Helper()
	testhelper.SetupTestLogger()

	db, err := history.InitializeHistoryDatabase(filepath.Join(t.TempDir(), "gpbackup_history.db"))
	if err != nil {
		t.Fatalf("Failed to initialize history database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for index := range configs {
		if err := history.StoreBackupHistory(db, &configs[index]); err != nil {
			t.Fatalf("Failed to store backup %s: %v", configs[index].Timestamp, err)
		}
	}
	return db
}

func backupDeleteTestConfig(timestamp, databaseName string, restorePlanTimestamp ...string) history.BackupConfig {
	config := history.BackupConfig{
		Timestamp:    timestamp,
		DatabaseName: databaseName,
		DateDeleted:  "",
		Status:       gpbckpconfig.BackupStatusSuccess,
	}
	for _, dependency := range restorePlanTimestamp {
		config.RestorePlan = append(config.RestorePlan, history.RestorePlanEntry{Timestamp: dependency})
	}
	return config
}
