package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpbackup/history"
	"github.com/jmoiron/sqlx"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

func TestSyncHistoryStandbyOrchestratesDiscoverySnapshotTransferAndInstall(t *testing.T) {
	testhelper.SetupTestLogger()

	rawTmpDir := t.TempDir()
	tmpDir, err := filepath.EvalSymlinks(rawTmpDir)
	if err != nil {
		tmpDir = rawTmpDir
	}
	primaryDataDir := filepath.Join(tmpDir, "primary")
	if err := os.Mkdir(primaryDataDir, 0o700); err != nil {
		t.Fatalf("Failed to create primary data dir: %v", err)
	}
	sourceDBPath := filepath.Join(primaryDataDir, historyDBNameConst)
	createHistoryStandbySyncTempSQLiteDB(t, sourceDBPath)
	standbyDataDir := filepath.Join(tmpDir, "standby")
	snapshotPath := setHistoryStandbySyncSnapshotPathHooks(t, tmpDir)

	db, mock := createHistoryStandbySyncMockDB(t)
	defer db.Close()
	expectHistoryStandbySyncPrimaryDataDirQuery(mock, primaryDataDir)
	expectHistoryStandbySyncStandbyQuery(mock, standbyDataDir)

	setHistoryStandbySyncRootFlags(t, sourceDBPath, false)
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		if clusterDBName != "testdb" {
			t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", clusterDBName, "testdb")
		}
		return db, nil
	})
	setHistoryStandbySyncCurrentUser(t)
	calls := setHistoryStandbySyncExecCommand(t, []historyStandbySyncExecResponse{
		{exitCode: 0},
		{exitCode: 0},
	})

	if err := syncHistoryStandby("testdb"); err != nil {
		t.Fatalf("Expected sync orchestration to succeed, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Unmet SQL expectations: %v", err)
	}
	if _, err := os.Stat(snapshotPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Expected local snapshot to be cleaned up, got: %v", err)
	}
	assertHistoryStandbySyncExecCall(t, (*calls)[0], "rsync", []string{
		"-e",
		"ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30",
		"--",
		snapshotPath,
		"gpadmin@sdw-standby:" + quoteHistoryStandbySyncRemoteShellArg(filepath.Join(standbyDataDir, filepath.Base(snapshotPath))),
	})
	assertHistoryStandbySyncExecCall(t, (*calls)[1], "ssh", nil)
}

func TestSyncHistoryStandbySkipsWhenDiscoveryFindsNoTarget(t *testing.T) {
	testhelper.SetupTestLogger()

	primaryDataDir := t.TempDir()
	sourceDBPath := filepath.Join(primaryDataDir, historyDBNameConst)
	createHistoryStandbySyncTempSQLiteDB(t, sourceDBPath)

	db, mock := createHistoryStandbySyncMockDB(t)
	defer db.Close()
	expectHistoryStandbySyncPrimaryDataDirQuery(mock, primaryDataDir)
	expectHistoryStandbySyncStandbyError(mock, sql.ErrNoRows)

	setHistoryStandbySyncRootFlags(t, sourceDBPath, false)
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		return db, nil
	})
	setHistoryStandbySyncExecCommand(t, nil)

	if err := syncHistoryStandby(""); err != nil {
		t.Fatalf("Expected skipped sync to succeed, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Unmet SQL expectations: %v", err)
	}
}

func TestSyncHistoryStandbyReturnsSnapshotError(t *testing.T) {
	testhelper.SetupTestLogger()

	primaryDataDir := t.TempDir()
	sourceDBPath := filepath.Join(primaryDataDir, historyDBNameConst)

	db, mock := createHistoryStandbySyncMockDB(t)
	defer db.Close()
	expectHistoryStandbySyncPrimaryDataDirQuery(mock, primaryDataDir)
	expectHistoryStandbySyncStandbyQuery(mock, filepath.Join(t.TempDir(), "standby"))

	setHistoryStandbySyncRootFlags(t, sourceDBPath, false)
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		return db, nil
	})
	setHistoryStandbySyncExecCommand(t, nil)

	err := syncHistoryStandby("")
	if err == nil {
		t.Fatalf("Expected sync orchestration to fail")
	}
	if !containsErrorText(err, "stat source history db") {
		t.Fatalf("Expected snapshot stat error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Unmet SQL expectations: %v", err)
	}
}

func TestSyncHistoryStandbyReturnsTransferError(t *testing.T) {
	testhelper.SetupTestLogger()

	primaryDataDir := t.TempDir()
	sourceDBPath := filepath.Join(primaryDataDir, historyDBNameConst)
	createHistoryStandbySyncTempSQLiteDB(t, sourceDBPath)
	setHistoryStandbySyncSnapshotPathHooks(t, t.TempDir())

	db, mock := createHistoryStandbySyncMockDB(t)
	defer db.Close()
	expectHistoryStandbySyncPrimaryDataDirQuery(mock, primaryDataDir)
	expectHistoryStandbySyncStandbyQuery(mock, filepath.Join(t.TempDir(), "standby"))

	setHistoryStandbySyncRootFlags(t, sourceDBPath, false)
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		return db, nil
	})
	setHistoryStandbySyncCurrentUser(t)
	setHistoryStandbySyncExecCommand(t, []historyStandbySyncExecResponse{
		{exitCode: 1, output: "rsync failed"},
		{exitCode: 0},
	})

	err := syncHistoryStandby("")
	if err == nil {
		t.Fatalf("Expected sync orchestration to fail")
	}
	if !containsErrorText(err, "rsync standby history snapshot") {
		t.Fatalf("Expected rsync error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Unmet SQL expectations: %v", err)
	}
}

func TestSyncHistoryStandbyBestEffortSkipsWhenDisabled(t *testing.T) {
	testhelper.SetupTestLogger()

	setHistoryStandbySyncRootFlags(t, "/primary/gpbackup_history.db", false)
	clusterConnCalls := 0
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		clusterConnCalls++
		return nil, errors.New("cluster connection should not run")
	})

	syncHistoryStandbyBestEffort("", true)
	if clusterConnCalls != 0 {
		t.Fatalf("\nCluster connection call count does not match:\n%v\nwant:\n%v", clusterConnCalls, 0)
	}
}

func TestSyncHistoryStandbyBestEffortWarnsAndDoesNotExit(t *testing.T) {
	testhelper.SetupTestLogger()

	setHistoryStandbySyncRootFlags(t, "/primary/gpbackup_history.db", false)
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		return nil, errors.New("discovery failed")
	})
	exitCalled := false
	setHistoryStandbySyncExecOSExit(t, func(code int) {
		exitCalled = true
	})

	syncHistoryStandbyBestEffort("", false)
	if exitCalled {
		t.Fatalf("Expected standby sync failure not to exit the process")
	}
}

func TestSyncHistoryStandbySkipsDefaultSourceSelectionBeforeDiscovery(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name     string
		autoLoad bool
		setupEnv func(*testing.T, string)
	}{
		{
			name: "Default Working Directory In Primary Data Directory",
			setupEnv: func(t *testing.T, primaryDataDir string) {
				oldWorkingDir, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get working dir: %v", err)
				}
				t.Cleanup(func() {
					if chdirErr := os.Chdir(oldWorkingDir); chdirErr != nil {
						t.Fatalf("Failed to restore working dir: %v", chdirErr)
					}
				})
				if chdirErr := os.Chdir(primaryDataDir); chdirErr != nil {
					t.Fatalf("Failed to change working dir: %v", chdirErr)
				}
			},
		},
		{
			name:     "Auto Load Without Env Vars",
			autoLoad: true,
			setupEnv: func(t *testing.T, primaryDataDir string) {
				t.Setenv("MASTER_DATA_DIRECTORY", "")
				t.Setenv("COORDINATOR_DATA_DIRECTORY", "")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primaryDataDir := t.TempDir()
			tt.setupEnv(t, primaryDataDir)
			setHistoryStandbySyncRootFlags(t, "", tt.autoLoad)

			clusterConnCalls := 0
			setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
				clusterConnCalls++
				return nil, errors.New("cluster connection should not run")
			})

			if err := syncHistoryStandby(""); err != nil {
				t.Fatalf("Expected skipped sync to succeed, got: %v", err)
			}
			if clusterConnCalls != 0 {
				t.Fatalf("\nCluster connection call count does not match:\n%v\nwant:\n%v", clusterConnCalls, 0)
			}
		})
	}
}

func TestSyncHistoryStandbySkipsIneligibleExplicitSourceBeforeDiscovery(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name     string
		source   func(string) string
		setupEnv func(*testing.T, string)
	}{
		{
			name: "Different File Name Without Data Directory Env",
			source: func(tmpDir string) string {
				return filepath.Join(tmpDir, "custom-history.db")
			},
			setupEnv: func(t *testing.T, tmpDir string) {
				t.Setenv("MASTER_DATA_DIRECTORY", "")
				t.Setenv("COORDINATOR_DATA_DIRECTORY", "")
			},
		},
		{
			name: "Explicit History DB Mismatches Data Directory Env",
			source: func(tmpDir string) string {
				return filepath.Join(tmpDir, "custom", historyDBNameConst)
			},
			setupEnv: func(t *testing.T, tmpDir string) {
				t.Setenv("MASTER_DATA_DIRECTORY", filepath.Join(tmpDir, "primary"))
				t.Setenv("COORDINATOR_DATA_DIRECTORY", "")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupEnv(t, tmpDir)
			setHistoryStandbySyncRootFlags(t, tt.source(tmpDir), false)

			clusterConnCalls := 0
			setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
				clusterConnCalls++
				return nil, errors.New("cluster connection should not run")
			})

			if err := syncHistoryStandby(""); err != nil {
				t.Fatalf("Expected skipped sync to succeed, got: %v", err)
			}
			if clusterConnCalls != 0 {
				t.Fatalf("\nCluster connection call count does not match:\n%v\nwant:\n%v", clusterConnCalls, 0)
			}
		})
	}
}

func TestGetHistoryStandbySyncSourceDBPath(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name               string
		rootHistoryDB      string
		autoLoadHistoryDB  bool
		masterDataDir      string
		coordinatorDataDir string
		want               string
		wantShouldDiscover bool
		wantReason         string
	}{
		{
			name:               "Default Path",
			want:               historyDBNameConst,
			wantShouldDiscover: false,
			wantReason:         "using default working-directory history db",
		},
		{
			name:               "Explicit History DB",
			rootHistoryDB:      "path/to/" + historyDBNameConst,
			want:               "path/to/" + historyDBNameConst,
			wantShouldDiscover: true,
		},
		{
			name:          "Explicit History DB With Different File Name Without Env Vars",
			rootHistoryDB: "path/to/custom_history.db",
			want:          "path/to/custom_history.db",
			wantReason:    "source history db path/to/custom_history.db is not named gpbackup_history.db",
		},
		{
			name:               "Explicit History DB Matches Master Data Directory",
			rootHistoryDB:      filepath.Join("/master/data", historyDBNameConst),
			masterDataDir:      "/master/data",
			want:               filepath.Join("/master/data", historyDBNameConst),
			wantShouldDiscover: true,
		},
		{
			name:               "Explicit History DB Matches Coordinator Data Directory",
			rootHistoryDB:      filepath.Join("/coordinator/data", historyDBNameConst),
			coordinatorDataDir: "/coordinator/data",
			want:               filepath.Join("/coordinator/data", historyDBNameConst),
			wantShouldDiscover: true,
		},
		{
			name:          "Explicit History DB Mismatches Data Directory",
			rootHistoryDB: filepath.Join("/custom/data", historyDBNameConst),
			masterDataDir: "/master/data",
			want:          filepath.Join("/custom/data", historyDBNameConst),
			wantReason:    "source history db /custom/data/gpbackup_history.db is not cluster history db /master/data/gpbackup_history.db",
		},
		{
			name:               "Auto Load History DB Without Env Vars",
			autoLoadHistoryDB:  true,
			want:               historyDBNameConst,
			wantShouldDiscover: false,
			wantReason:         "--auto-load-history-db did not find MASTER_DATA_DIRECTORY or COORDINATOR_DATA_DIRECTORY",
		},
		{
			name:               "Auto Load History DB",
			autoLoadHistoryDB:  true,
			masterDataDir:      "/master/data",
			want:               filepath.Join("/master/data", historyDBNameConst),
			wantShouldDiscover: true,
		},
		{
			name:               "Coordinator Data Directory Fallback",
			autoLoadHistoryDB:  true,
			coordinatorDataDir: "/coordinator/data",
			want:               filepath.Join("/coordinator/data", historyDBNameConst),
			wantShouldDiscover: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MASTER_DATA_DIRECTORY", tt.masterDataDir)
			t.Setenv("COORDINATOR_DATA_DIRECTORY", tt.coordinatorDataDir)
			setHistoryStandbySyncRootFlags(t, tt.rootHistoryDB, tt.autoLoadHistoryDB)

			got, shouldDiscover, reason := getHistoryStandbySyncSourceDBPath()
			if got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
			if shouldDiscover != tt.wantShouldDiscover {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", shouldDiscover, tt.wantShouldDiscover)
			}
			if reason != tt.wantReason {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", reason, tt.wantReason)
			}
		})
	}
}

func TestRunHistoryMutationWithStandbySyncTriggersAfterSuccess(t *testing.T) {
	testhelper.SetupTestLogger()

	setHistoryStandbySyncRootFlags(t, "/primary/gpbackup_history.db", false)
	clusterConnCalls := 0
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		clusterConnCalls++
		if clusterDBName != "testdb" {
			t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", clusterDBName, "testdb")
		}
		return nil, errors.New("stop after sync trigger")
	})
	setHistoryStandbySyncExecOSExit(t, func(code int) {
		t.Fatalf("Command unexpectedly exited with code %d", code)
	})

	runHistoryMutationWithStandbySync(func() (string, error) {
		return "testdb", nil
	}, false)
	if clusterConnCalls != 1 {
		t.Fatalf("\nCluster connection call count does not match:\n%v\nwant:\n%v", clusterConnCalls, 1)
	}
}

func TestRunHistoryMutationWithStandbySyncDoesNotTriggerAfterFailure(t *testing.T) {
	testhelper.SetupTestLogger()

	setHistoryStandbySyncRootFlags(t, "/primary/gpbackup_history.db", false)
	clusterConnCalls := 0
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		clusterConnCalls++
		return nil, errors.New("cluster connection should not run")
	})
	exitCalls := 0
	setHistoryStandbySyncExecOSExit(t, func(code int) {
		exitCalls++
	})

	runHistoryMutationWithStandbySync(func() (string, error) {
		return "", errors.New("primary command failed")
	}, false)
	if exitCalls != 1 {
		t.Fatalf("\nExit call count does not match:\n%v\nwant:\n%v", exitCalls, 1)
	}
	if clusterConnCalls != 0 {
		t.Fatalf("\nCluster connection call count does not match:\n%v\nwant:\n%v", clusterConnCalls, 0)
	}
}

func TestNoHistorySyncStandbyFlagDisablesSyncAfterMutationSuccess(t *testing.T) {
	testhelper.SetupTestLogger()

	setHistoryStandbySyncRootFlags(t, "/primary/gpbackup_history.db", false)
	clusterConnCalls := 0
	setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
		clusterConnCalls++
		return nil, errors.New("cluster connection should not run")
	})
	setHistoryStandbySyncExecOSExit(t, func(code int) {
		t.Fatalf("Command unexpectedly exited with code %d", code)
	})

	runHistoryMutationWithStandbySync(func() (string, error) {
		return "testdb", nil
	}, true)
	if clusterConnCalls != 0 {
		t.Fatalf("\nCluster connection call count does not match:\n%v\nwant:\n%v", clusterConnCalls, 0)
	}
}

func TestReadOnlyCommandsDoNotTriggerHistoryStandbySync(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name  string
		setup func(*testing.T)
		run   func()
	}{
		{
			name: "Backup Info",
			setup: func(t *testing.T) {
				historyDBPath := createInitializedHistoryDB(t, t.TempDir())
				setHistoryStandbySyncRootFlags(t, historyDBPath, false)
			},
			run: doBackupInfo,
		},
		{
			name: "Report Info",
			setup: func(t *testing.T) {
				historyDBPath, timestamp, backupDir := createHistoryDBWithLocalReport(t, t.TempDir())
				setHistoryStandbySyncRootFlags(t, historyDBPath, false)
				oldReportInfoTimestamp := reportInfoTimestamp
				oldReportInfoBackupDir := reportInfoBackupDir
				reportInfoTimestamp = timestamp
				reportInfoBackupDir = backupDir
				t.Cleanup(func() {
					reportInfoTimestamp = oldReportInfoTimestamp
					reportInfoBackupDir = oldReportInfoBackupDir
				})
			},
			run: doReportInfo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterConnCalls := 0
			setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
				clusterConnCalls++
				return nil, errors.New("cluster connection should not run")
			})
			setHistoryStandbySyncExecOSExit(t, func(code int) {
				t.Fatalf("Command unexpectedly exited with code %d", code)
			})
			tt.setup(t)

			tt.run()
			if clusterConnCalls != 0 {
				t.Fatalf("\nCluster connection call count does not match:\n%v\nwant:\n%v", clusterConnCalls, 0)
			}
		})
	}
}

func TestCheckHistoryStandbySyncSourcePathEligible(t *testing.T) {
	testhelper.SetupTestLogger()

	tmpDir := t.TempDir()
	primaryDataDir := filepath.Join(tmpDir, "primary")
	if err := os.Mkdir(primaryDataDir, 0o700); err != nil {
		t.Fatalf("Failed to create primary data dir: %v", err)
	}
	clusterHistoryDBPath := filepath.Join(primaryDataDir, historyDBNameConst)
	if err := os.WriteFile(clusterHistoryDBPath, []byte("history"), 0o600); err != nil {
		t.Fatalf("Failed to create history db file: %v", err)
	}
	symlinkPath := filepath.Join(tmpDir, "history-link.db")
	if err := os.Symlink(clusterHistoryDBPath, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	tests := []struct {
		name      string
		source    string
		wantMatch bool
	}{
		{
			name:      "Cluster History DB",
			source:    clusterHistoryDBPath,
			wantMatch: true,
		},
		{
			name:      "Symlink To Cluster History DB",
			source:    symlinkPath,
			wantMatch: true,
		},
		{
			name:      "Different History DB",
			source:    filepath.Join(tmpDir, "custom", historyDBNameConst),
			wantMatch: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, _, err := checkHistoryStandbySyncSourcePathEligible(tt.source, primaryDataDir)
			if err != nil {
				t.Fatalf("Expected source path eligibility check to succeed, got: %v", err)
			}
			if got != tt.wantMatch {
				t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.wantMatch)
			}
		})
	}
}

func TestCheckHistoryStandbySyncSourcePathEligibleDefaultWorkingDirectory(t *testing.T) {
	testhelper.SetupTestLogger()

	// Resolve symlinks on tmpDir so that paths built from it match the
	// normalized paths returned by normalizeHistoryStandbySyncPath
	// (needed on macOS where /var -> /private/var).
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to resolve tmpDir symlinks: %v", err)
	}
	primaryDataDir := filepath.Join(tmpDir, "primary")
	workingDir := filepath.Join(tmpDir, "work")
	if mkdirErr := os.Mkdir(primaryDataDir, 0o700); mkdirErr != nil {
		t.Fatalf("Failed to create primary data dir: %v", mkdirErr)
	}
	if mkdirErr := os.Mkdir(workingDir, 0o700); mkdirErr != nil {
		t.Fatalf("Failed to create working dir: %v", mkdirErr)
	}
	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working dir: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(oldWorkingDir); chdirErr != nil {
			t.Fatalf("Failed to restore working dir: %v", chdirErr)
		}
	})
	if chdirErr := os.Chdir(workingDir); chdirErr != nil {
		t.Fatalf("Failed to change working dir: %v", chdirErr)
	}

	got, normalizedSource, normalizedCluster, err := checkHistoryStandbySyncSourcePathEligible(historyDBNameConst, primaryDataDir)
	if err != nil {
		t.Fatalf("Expected source path eligibility check to succeed, got: %v", err)
	}
	if got {
		t.Fatalf("Expected default working-directory history db to be ineligible")
	}
	if normalizedSource != filepath.Join(workingDir, historyDBNameConst) {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", normalizedSource, filepath.Join(workingDir, historyDBNameConst))
	}
	if normalizedCluster != filepath.Join(primaryDataDir, historyDBNameConst) {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", normalizedCluster, filepath.Join(primaryDataDir, historyDBNameConst))
	}
}

func TestCheckHistoryStandbySyncSourcePathEligibleResolvesExistingParentSymlinkForMissingLeaf(t *testing.T) {
	testhelper.SetupTestLogger()

	tmpDir := t.TempDir()
	realPrimaryDataDir := filepath.Join(tmpDir, "real-primary")
	if err := os.Mkdir(realPrimaryDataDir, 0o700); err != nil {
		t.Fatalf("Failed to create real primary data dir: %v", err)
	}
	sourceDBPath := filepath.Join(realPrimaryDataDir, historyDBNameConst)
	if err := os.WriteFile(sourceDBPath, []byte("history"), 0o600); err != nil {
		t.Fatalf("Failed to create source history db file: %v", err)
	}
	linkedPrimaryDataDir := filepath.Join(tmpDir, "linked-primary")
	if err := os.Symlink(realPrimaryDataDir, linkedPrimaryDataDir); err != nil {
		t.Fatalf("Failed to create primary data dir symlink: %v", err)
	}

	got, normalizedSource, normalizedCluster, err := checkHistoryStandbySyncSourcePathEligible(sourceDBPath, linkedPrimaryDataDir)
	if err != nil {
		t.Fatalf("Expected source path eligibility check to succeed, got: %v", err)
	}
	if !got {
		t.Fatalf("Expected source path to match cluster history db")
	}
	if normalizedSource != normalizedCluster {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", normalizedSource, normalizedCluster)
	}
	wantSourceDBPath, err := filepath.EvalSymlinks(sourceDBPath)
	if err != nil {
		t.Fatalf("Failed to resolve source history db path: %v", err)
	}
	if normalizedCluster != wantSourceDBPath {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", normalizedCluster, wantSourceDBPath)
	}
}

func TestDiscoverHistoryStandbySyncTarget(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name             string
		sourcePath       func(string, string) string
		rootHistoryDB    string
		autoLoad         bool
		queryStandbyErr  error
		wantTarget       bool
		wantStandbyQuery bool
		wantSkipReason   string
		wantErr          bool
	}{
		{
			name: "Source Path Match",
			sourcePath: func(primaryDataDir, _ string) string {
				return filepath.Join(primaryDataDir, historyDBNameConst)
			},
			wantTarget:       true,
			wantStandbyQuery: true,
		},
		{
			name: "Source Path Mismatch",
			sourcePath: func(_, tmpDir string) string {
				return filepath.Join(tmpDir, "custom", historyDBNameConst)
			},
			wantStandbyQuery: false,
			wantSkipReason:   "is not cluster history db",
		},
		{
			name:          "Explicit History DB Match",
			rootHistoryDB: "cluster",
			sourcePath: func(primaryDataDir, _ string) string {
				return filepath.Join(primaryDataDir, historyDBNameConst)
			},
			wantTarget:       true,
			wantStandbyQuery: true,
		},
		{
			name:     "Auto Load History DB Match",
			autoLoad: true,
			sourcePath: func(primaryDataDir, _ string) string {
				return filepath.Join(primaryDataDir, historyDBNameConst)
			},
			wantTarget:       true,
			wantStandbyQuery: true,
		},
		{
			name: "No Standby Row",
			sourcePath: func(primaryDataDir, _ string) string {
				return filepath.Join(primaryDataDir, historyDBNameConst)
			},
			queryStandbyErr:  sql.ErrNoRows,
			wantStandbyQuery: true,
			wantSkipReason:   "no up standby coordinator found",
		},
		{
			name: "Standby Query Error",
			sourcePath: func(primaryDataDir, _ string) string {
				return filepath.Join(primaryDataDir, historyDBNameConst)
			},
			queryStandbyErr:  errors.New("query failed"),
			wantStandbyQuery: true,
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawTmpDir := t.TempDir()
			tmpDir, err := filepath.EvalSymlinks(rawTmpDir)
			if err != nil {
				tmpDir = rawTmpDir
			}
			primaryDataDir := filepath.Join(tmpDir, "primary")
			if mkdirErr := os.Mkdir(primaryDataDir, 0o700); mkdirErr != nil {
				t.Fatalf("Failed to create primary data dir: %v", mkdirErr)
			}
			t.Setenv("MASTER_DATA_DIRECTORY", "")
			if tt.autoLoad {
				t.Setenv("MASTER_DATA_DIRECTORY", primaryDataDir)
			}

			historyDBPath := ""
			if tt.rootHistoryDB == "cluster" {
				historyDBPath = filepath.Join(primaryDataDir, historyDBNameConst)
			}
			setHistoryStandbySyncRootFlags(t, historyDBPath, tt.autoLoad)

			db, mock := createHistoryStandbySyncMockDB(t)
			defer db.Close()
			expectHistoryStandbySyncPrimaryDataDirQuery(mock, primaryDataDir)
			if tt.wantStandbyQuery {
				if tt.queryStandbyErr != nil {
					expectHistoryStandbySyncStandbyError(mock, tt.queryStandbyErr)
				} else {
					expectHistoryStandbySyncStandbyQuery(mock, filepath.Join(tmpDir, "standby"))
				}
			}
			setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
				if clusterDBName != "testdb" {
					t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", clusterDBName, "testdb")
				}
				return db, nil
			})

			target, skipReason, err := discoverHistoryStandbySyncTarget(tt.sourcePath(primaryDataDir, tmpDir), "testdb")
			if (err != nil) != tt.wantErr {
				t.Fatalf("\ndiscoverHistoryStandbySyncTarget() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("Unmet SQL expectations: %v", err)
			}
			if tt.wantSkipReason == "" && skipReason != "" {
				t.Fatalf("Expected empty skip reason, got: %q", skipReason)
			}
			if tt.wantSkipReason != "" && !strings.Contains(skipReason, tt.wantSkipReason) {
				t.Fatalf("Expected skip reason containing %q, got: %q", tt.wantSkipReason, skipReason)
			}
			if (target != nil) != tt.wantTarget {
				t.Fatalf("\nVariables do not match:\n%v\nwant target:\n%v", target, tt.wantTarget)
			}
			if target != nil {
				wantSourcePath := filepath.Join(primaryDataDir, historyDBNameConst)
				if target.sourceDBPath != wantSourcePath {
					t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", target.sourceDBPath, wantSourcePath)
				}
				if target.standbyHost != "sdw-standby" {
					t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", target.standbyHost, "sdw-standby")
				}
				if target.standbyHistoryDBPath != filepath.Join(tmpDir, "standby", historyDBNameConst) {
					t.Fatalf(
						"\nVariables do not match:\n%v\nwant:\n%v",
						target.standbyHistoryDBPath,
						filepath.Join(tmpDir, "standby", historyDBNameConst),
					)
				}
			}
		})
	}
}

func TestDiscoverHistoryStandbySyncTargetDiscoveryFailures(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name       string
		setupHooks func(*testing.T, string)
		wantError  string
	}{
		{
			name: "Cluster Connection Failure",
			setupHooks: func(t *testing.T, primaryDataDir string) {
				setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
					return nil, errors.New("connect failed")
				})
			},
			wantError: "connect failed",
		},
		{
			name: "Primary Query Failure",
			setupHooks: func(t *testing.T, primaryDataDir string) {
				db, mock := createHistoryStandbySyncMockDB(t)
				expectHistoryStandbySyncPrimaryDataDirError(mock, errors.New("primary query failed"))
				setHistoryStandbySyncNewClusterConnHook(t, func(clusterDBName string) (*sqlx.DB, error) {
					return db, nil
				})
			},
			wantError: "primary query failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			primaryDataDir := filepath.Join(tmpDir, "primary")
			if err := os.Mkdir(primaryDataDir, 0o700); err != nil {
				t.Fatalf("Failed to create primary data dir: %v", err)
			}

			tt.setupHooks(t, primaryDataDir)

			_, _, err := discoverHistoryStandbySyncTarget(filepath.Join(primaryDataDir, historyDBNameConst), "")
			if err == nil {
				t.Fatalf("Expected discovery to fail")
			}
			if !containsErrorText(err, tt.wantError) {
				t.Fatalf("Expected error %q, got: %v", tt.wantError, err)
			}
		})
	}
}

func TestCreateHistoryStandbySyncSnapshotCreatesValidVacuumIntoSnapshot(t *testing.T) {
	testhelper.SetupTestLogger()

	tmpDir := t.TempDir()
	sourceDBPath := filepath.Join(tmpDir, "source.db")
	createHistoryStandbySyncTempSQLiteDB(t, sourceDBPath)
	if err := os.Chmod(sourceDBPath, 0o640); err != nil {
		t.Fatalf("Failed to set source db permissions: %v", err)
	}

	setHistoryStandbySyncSnapshotPathHooks(t, tmpDir)

	snapshotPath, err := createHistoryStandbySyncSnapshot(sourceDBPath)
	if err != nil {
		t.Fatalf("Expected snapshot creation to succeed, got: %v", err)
	}
	defer cleanupHistoryStandbySyncSnapshot(snapshotPath)

	if filepath.Dir(snapshotPath) != tmpDir {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", filepath.Dir(snapshotPath), tmpDir)
	}
	snapshotFileName := filepath.Base(snapshotPath)
	if !strings.HasPrefix(snapshotFileName, historyFileNameBaseConst+"_") || !strings.HasSuffix(snapshotFileName, historyFileDBSuffixConst+".snap") {
		t.Fatalf("Unexpected snapshot file name: %s", snapshotFileName)
	}
	fileInfo, err := os.Stat(snapshotPath)
	if err != nil {
		t.Fatalf("Expected snapshot file to exist, got: %v", err)
	}
	sourceInfo, err := os.Stat(sourceDBPath)
	if err != nil {
		t.Fatalf("Expected source history db file to exist, got: %v", err)
	}
	if fileInfo.Mode().Perm() != sourceInfo.Mode().Perm() {
		t.Fatalf("\nSnapshot file permissions do not match:\n%v\nwant:\n%v", fileInfo.Mode().Perm(), sourceInfo.Mode().Perm())
	}

	rowCount := queryHistoryStandbySyncSnapshotRowCount(t, snapshotPath)
	if rowCount != 1 {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", rowCount, 1)
	}
	quickCheckResults, err := runHistoryStandbySyncQuickCheck(snapshotPath)
	if err != nil {
		t.Fatalf("Expected quick_check to succeed, got: %v", err)
	}
	if err := validateHistoryStandbySyncQuickCheckResults(quickCheckResults); err != nil {
		t.Fatalf("Expected quick_check result to be valid, got: %v", err)
	}
}

func TestCreateHistoryStandbySyncSnapshotFailsIfTargetExists(t *testing.T) {
	testhelper.SetupTestLogger()

	tmpDir := t.TempDir()
	sourceDBPath := filepath.Join(tmpDir, "source.db")
	createHistoryStandbySyncTempSQLiteDB(t, sourceDBPath)

	snapshotPath := setHistoryStandbySyncSnapshotPathHooks(t, tmpDir)
	if err := os.WriteFile(snapshotPath, []byte("existing snapshot"), 0o600); err != nil {
		t.Fatalf("Failed to create existing snapshot file: %v", err)
	}

	_, err := createHistoryStandbySyncSnapshot(sourceDBPath)
	if err == nil {
		t.Fatalf("Expected snapshot creation to fail when target exists")
	}
	if !containsErrorText(err, "target already exists") {
		t.Fatalf("Expected target exists error, got: %v", err)
	}

	contents, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("Expected existing snapshot file to remain, got: %v", err)
	}
	if string(contents) != "existing snapshot" {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", string(contents), "existing snapshot")
	}
}

func TestValidateHistoryStandbySyncQuickCheckResults(t *testing.T) {
	tests := []struct {
		name    string
		results []string
		wantErr bool
	}{
		{
			name:    "Single OK",
			results: []string{"ok"},
		},
		{
			name:    "No Rows",
			results: []string{},
			wantErr: true,
		},
		{
			name:    "Multiple OK Rows",
			results: []string{"ok", "ok"},
			wantErr: true,
		},
		{
			name:    "Invalid Result",
			results: []string{"database disk image is malformed"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHistoryStandbySyncQuickCheckResults(tt.results)
			if (err != nil) != tt.wantErr {
				t.Fatalf("\nvalidateHistoryStandbySyncQuickCheckResults() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestWithHistoryStandbySyncSnapshotCleansUpAfterSyncSuccessOrFailure(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name    string
		syncErr error
	}{
		{
			name: "Success",
		},
		{
			name:    "Failure",
			syncErr: errors.New("sync failed"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sourceDBPath := filepath.Join(tmpDir, "source.db")
			createHistoryStandbySyncTempSQLiteDB(t, sourceDBPath)
			setHistoryStandbySyncSnapshotPathHooks(t, tmpDir)

			var snapshotPath string
			err := withHistoryStandbySyncSnapshot(sourceDBPath, func(path string) error {
				snapshotPath = path
				if _, statErr := os.Stat(snapshotPath); statErr != nil {
					t.Fatalf("Expected snapshot to exist during sync callback, got: %v", statErr)
				}
				return tt.syncErr
			})
			if tt.syncErr != nil {
				if !errors.Is(err, tt.syncErr) {
					t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", err, tt.syncErr)
				}
			} else if err != nil {
				t.Fatalf("Expected sync callback to succeed, got: %v", err)
			}
			if snapshotPath == "" {
				t.Fatalf("Expected sync callback to receive snapshot path")
			}
			if _, err := os.Stat(snapshotPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Expected snapshot to be cleaned up after sync callback, got: %v", err)
			}
		})
	}
}

func TestSyncHistoryStandbySnapshotToStandbyTransfersAndInstalls(t *testing.T) {
	testhelper.SetupTestLogger()

	setHistoryStandbySyncCurrentUser(t)
	snapshotPath := "/tmp/gpbackup_history_20260703000000_000000000_42.db.snap"
	target := &historyStandbySyncTarget{
		standbyHost:          "sdw-standby",
		standbyDataDir:       "/data/standby",
		standbyHistoryDBPath: "/data/standby/gpbackup_history.db",
	}
	calls := setHistoryStandbySyncExecCommand(t, []historyStandbySyncExecResponse{
		{exitCode: 0},
		{exitCode: 0},
	})

	err := syncHistoryStandbySnapshotToStandby(target, snapshotPath)
	if err != nil {
		t.Fatalf("Expected standby sync to succeed, got: %v", err)
	}

	remoteTempPath := "/data/standby/gpbackup_history_20260703000000_000000000_42.db.snap"
	if filepath.Dir(remoteTempPath) != target.standbyDataDir {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", filepath.Dir(remoteTempPath), target.standbyDataDir)
	}
	assertHistoryStandbySyncExecCall(t, (*calls)[0], "rsync", []string{
		"-e",
		"ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30",
		"--",
		snapshotPath,
		"gpadmin@sdw-standby:'/data/standby/gpbackup_history_20260703000000_000000000_42.db.snap'",
	})
	assertHistoryStandbySyncExecCall(t, (*calls)[1], "ssh", []string{
		"-o",
		"StrictHostKeyChecking=no",
		"-o",
		"ConnectTimeout=30",
		"gpadmin@sdw-standby",
		"if [ -e '/data/standby/gpbackup_history.db' ]; then chown --reference='/data/standby/gpbackup_history.db' -- '/data/standby/gpbackup_history_20260703000000_000000000_42.db.snap' 2>/dev/null || true; chmod --reference='/data/standby/gpbackup_history.db' -- '/data/standby/gpbackup_history_20260703000000_000000000_42.db.snap' 2>/dev/null || true; fi; mv -f -- '/data/standby/gpbackup_history_20260703000000_000000000_42.db.snap' '/data/standby/gpbackup_history.db'",
	})
}

func TestSyncHistoryStandbySnapshotToStandbyCleansRemoteTempAfterInstallFailure(t *testing.T) {
	testhelper.SetupTestLogger()

	setHistoryStandbySyncCurrentUser(t)
	snapshotPath := "/tmp/gpbackup_history_20260703000000_000000000_42.db.snap"
	target := &historyStandbySyncTarget{
		standbyHost:          "sdw-standby",
		standbyDataDir:       "/data/standby",
		standbyHistoryDBPath: "/data/standby/gpbackup_history.db",
	}
	calls := setHistoryStandbySyncExecCommand(t, []historyStandbySyncExecResponse{
		{exitCode: 0},
		{exitCode: 1, output: "mv failed"},
		{exitCode: 0},
	})

	err := syncHistoryStandbySnapshotToStandby(target, snapshotPath)
	if err == nil {
		t.Fatalf("Expected standby sync to fail")
	}
	if !containsErrorText(err, "install standby history snapshot") {
		t.Fatalf("Expected install error, got: %v", err)
	}

	assertHistoryStandbySyncExecCall(t, (*calls)[0], "rsync", nil)
	assertHistoryStandbySyncExecCall(t, (*calls)[1], "ssh", nil)
	assertHistoryStandbySyncExecCall(t, (*calls)[2], "ssh", []string{
		"-o",
		"StrictHostKeyChecking=no",
		"-o",
		"ConnectTimeout=30",
		"gpadmin@sdw-standby",
		"rm -f -- '/data/standby/gpbackup_history_20260703000000_000000000_42.db.snap'",
	})
}

func TestSyncHistoryStandbySnapshotToStandbyFailuresReturnErrorsAndDoNotExit(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name      string
		responses []historyStandbySyncExecResponse
		wantError string
	}{
		{
			name: "Rsync Failure",
			responses: []historyStandbySyncExecResponse{
				{exitCode: 1, output: "rsync failed"},
				{exitCode: 0},
			},
			wantError: "rsync standby history snapshot",
		},
		{
			name: "Remote Install Failure",
			responses: []historyStandbySyncExecResponse{
				{exitCode: 0},
				{exitCode: 1, output: "mv failed"},
				{exitCode: 0},
			},
			wantError: "install standby history snapshot",
		},
		{
			name: "Remote Cleanup Failure",
			responses: []historyStandbySyncExecResponse{
				{exitCode: 1, output: "rsync failed"},
				{exitCode: 1, output: "rm failed"},
			},
			wantError: "additionally failed to clean up remote temp file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHistoryStandbySyncCurrentUser(t)
			setHistoryStandbySyncExecCommand(t, tt.responses)
			exitCalled := false
			setHistoryStandbySyncExecOSExit(t, func(code int) {
				exitCalled = true
			})

			err := syncHistoryStandbySnapshotToStandby(&historyStandbySyncTarget{
				standbyHost:          "sdw-standby",
				standbyDataDir:       "/data/standby",
				standbyHistoryDBPath: "/data/standby/gpbackup_history.db",
			}, "/tmp/gpbackup_history_20260703000000_000000000_42.db.snap")
			if err == nil {
				t.Fatalf("Expected standby sync to fail")
			}
			if !containsErrorText(err, tt.wantError) {
				t.Fatalf("Expected error %q, got: %v", tt.wantError, err)
			}
			if exitCalled {
				t.Fatalf("Expected standby sync failure not to exit the process")
			}
		})
	}
}

func TestSyncHistoryStandbySnapshotToStandbyCurrentUserFailureDoesNotRunCommands(t *testing.T) {
	testhelper.SetupTestLogger()

	oldCurrentUser := historyStandbySyncCurrentUser
	historyStandbySyncCurrentUser = func() (string, error) {
		return "", errors.New("current user failed")
	}
	t.Cleanup(func() {
		historyStandbySyncCurrentUser = oldCurrentUser
	})
	calls := setHistoryStandbySyncExecCommand(t, nil)

	err := syncHistoryStandbySnapshotToStandby(&historyStandbySyncTarget{
		standbyHost:          "sdw-standby",
		standbyDataDir:       "/data/standby",
		standbyHistoryDBPath: "/data/standby/gpbackup_history.db",
	}, "/tmp/gpbackup_history_20260703000000_000000000_42.db.snap")
	if err == nil {
		t.Fatalf("Expected standby sync to fail")
	}
	if !containsErrorText(err, "resolve current OS user") {
		t.Fatalf("Expected current user error, got: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("\nCommand call count does not match:\n%v\nwant:\n%v", len(*calls), 0)
	}
}

func TestHistoryStandbySyncCommandBuildersQuotePaths(t *testing.T) {
	remotePath := "/data/standby dir/owner's/gpbackup_history.db"
	quotedPath := "'/data/standby dir/owner'\"'\"'s/gpbackup_history.db'"

	if got := quoteHistoryStandbySyncRemoteShellArg(remotePath); got != quotedPath {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got, quotedPath)
	}
	if got := buildHistoryStandbySyncRsyncArgs("/tmp/local snapshot.db", "sdw-standby", "gpadmin", remotePath); got[4] != "gpadmin@sdw-standby:"+quotedPath {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got[4], "gpadmin@sdw-standby:"+quotedPath)
	}
	if got := buildHistoryStandbySyncRemoteCleanupCommand(remotePath); got != "rm -f -- "+quotedPath {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got, "rm -f -- "+quotedPath)
	}
	installCommand := buildHistoryStandbySyncRemoteInstallCommand(remotePath, remotePath)
	if !strings.Contains(installCommand, "chown --reference="+quotedPath+" -- "+quotedPath) {
		t.Fatalf("Expected install command to quote chown paths, got: %s", installCommand)
	}
	if !strings.Contains(installCommand, "chmod --reference="+quotedPath+" -- "+quotedPath) {
		t.Fatalf("Expected install command to quote chmod paths, got: %s", installCommand)
	}
	if !strings.Contains(installCommand, "mv -f -- "+quotedPath+" "+quotedPath) {
		t.Fatalf("Expected install command to quote mv paths, got: %s", installCommand)
	}
}

func setHistoryStandbySyncRootFlags(t *testing.T, historyDBPath string, autoLoadHistoryDB bool) {
	t.Helper()

	oldRootHistoryDB := rootHistoryDB
	oldRootAutoLoadHistoryDB := rootAutoLoadHistoryDB
	rootHistoryDB = historyDBPath
	rootAutoLoadHistoryDB = autoLoadHistoryDB
	t.Cleanup(func() {
		rootHistoryDB = oldRootHistoryDB
		rootAutoLoadHistoryDB = oldRootAutoLoadHistoryDB
	})
}

func setHistoryStandbySyncNewClusterConnHook(t *testing.T, hook func(string) (*sqlx.DB, error)) {
	t.Helper()

	oldHook := historyStandbySyncNewClusterConn
	historyStandbySyncNewClusterConn = hook
	t.Cleanup(func() {
		historyStandbySyncNewClusterConn = oldHook
	})
}

func setHistoryStandbySyncExecOSExit(t *testing.T, hook func(int)) {
	t.Helper()

	oldHook := execOSExit
	execOSExit = hook
	t.Cleanup(func() {
		execOSExit = oldHook
	})
}

func createInitializedHistoryDB(t *testing.T, tmpDir string) string {
	t.Helper()

	historyDBPath := filepath.Join(tmpDir, historyDBNameConst)
	db, err := history.InitializeHistoryDatabase(historyDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize history db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close initialized history db: %v", err)
	}
	return historyDBPath
}

func createHistoryDBWithLocalReport(t *testing.T, tmpDir string) (string, string, string) {
	t.Helper()

	historyDBPath := filepath.Join(tmpDir, historyDBNameConst)
	db, err := history.InitializeHistoryDatabase(historyDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize history db: %v", err)
	}
	defer db.Close()

	timestamp := "20260703010101"
	backupDir := filepath.Join(tmpDir, "backup-root")
	reportPath := gpbckpconfig.ReportFilePath(backupDir, timestamp)
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o700); err != nil {
		t.Fatalf("Failed to create report directory: %v", err)
	}
	if err := os.WriteFile(reportPath, []byte("report contents"), 0o600); err != nil {
		t.Fatalf("Failed to write report file: %v", err)
	}
	if err := history.StoreBackupHistory(db, &history.BackupConfig{
		Timestamp:    timestamp,
		BackupDir:    backupDir,
		DatabaseName: "postgres",
		DateDeleted:  "",
		Status:       gpbckpconfig.BackupStatusSuccess,
	}); err != nil {
		t.Fatalf("Failed to store backup history: %v", err)
	}
	return historyDBPath, timestamp, backupDir
}

func containsErrorText(err error, want string) bool {
	return err != nil && strings.Contains(err.Error(), want)
}

type historyStandbySyncExecResponse struct {
	exitCode int
	output   string
}

type historyStandbySyncExecCall struct {
	command string
	args    []string
}

func setHistoryStandbySyncCurrentUser(t *testing.T) {
	t.Helper()

	oldCurrentUser := historyStandbySyncCurrentUser
	historyStandbySyncCurrentUser = func() (string, error) {
		return "gpadmin", nil
	}
	t.Cleanup(func() {
		historyStandbySyncCurrentUser = oldCurrentUser
	})
}

func setHistoryStandbySyncExecCommand(t *testing.T, responses []historyStandbySyncExecResponse) *[]historyStandbySyncExecCall {
	t.Helper()

	oldCommand := execCommand
	calls := make([]historyStandbySyncExecCall, 0, len(responses))
	responseIndex := 0
	execCommand = func(command string, args ...string) *exec.Cmd {
		if responseIndex >= len(responses) {
			t.Fatalf("Unexpected command: %s %v", command, args)
		}
		response := responses[responseIndex]
		responseIndex++
		copiedArgs := append([]string(nil), args...)
		calls = append(calls, historyStandbySyncExecCall{command: command, args: copiedArgs})

		cmd := exec.Command(os.Args[0], "-test.run=TestHistoryStandbySyncExecHelper", "--")
		cmd.Env = append(
			os.Environ(),
			"GPBACKMAN_TEST_EXEC_HELPER=1",
			fmt.Sprintf("GPBACKMAN_TEST_EXEC_EXIT_CODE=%d", response.exitCode),
			"GPBACKMAN_TEST_EXEC_OUTPUT="+response.output,
		)
		return cmd
	}
	t.Cleanup(func() {
		execCommand = oldCommand
		if responseIndex != len(responses) {
			t.Fatalf("\nCommand count does not match:\n%v\nwant:\n%v", responseIndex, len(responses))
		}
	})
	return &calls
}

func assertHistoryStandbySyncExecCall(t *testing.T, got historyStandbySyncExecCall, wantCommand string, wantArgs []string) {
	t.Helper()

	if got.command != wantCommand {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got.command, wantCommand)
	}
	if wantArgs == nil {
		return
	}
	if len(got.args) != len(wantArgs) {
		t.Fatalf("\nArgument count does not match:\n%v\nwant:\n%v\nargs:\n%v", len(got.args), len(wantArgs), got.args)
	}
	for i := range wantArgs {
		if got.args[i] != wantArgs[i] {
			t.Fatalf("\nArgument %d does not match:\n%v\nwant:\n%v\nargs:\n%v", i, got.args[i], wantArgs[i], got.args)
		}
	}
}

func TestHistoryStandbySyncExecHelper(t *testing.T) {
	if os.Getenv("GPBACKMAN_TEST_EXEC_HELPER") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, os.Getenv("GPBACKMAN_TEST_EXEC_OUTPUT"))
	exitCode, err := strconv.Atoi(os.Getenv("GPBACKMAN_TEST_EXEC_EXIT_CODE"))
	if err != nil {
		os.Exit(1)
	}
	os.Exit(exitCode)
}

func createHistoryStandbySyncMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock DB: %v", err)
	}
	return sqlx.NewDb(rawDB, "sqlmock"), mock
}

func expectHistoryStandbySyncPrimaryDataDirQuery(mock sqlmock.Sqlmock, primaryDataDir string) {
	query := "SELECT datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'p';"
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WillReturnRows(sqlmock.NewRows([]string{"datadir"}).AddRow(primaryDataDir))
}

func expectHistoryStandbySyncPrimaryDataDirError(mock sqlmock.Sqlmock, err error) {
	query := "SELECT datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'p';"
	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(err)
}

func expectHistoryStandbySyncStandbyQuery(mock sqlmock.Sqlmock, dataDir string) {
	query := "SELECT hostname, datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'm' AND status = 'u';"
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WillReturnRows(sqlmock.NewRows([]string{"hostname", "datadir"}).AddRow("sdw-standby", dataDir))
}

func expectHistoryStandbySyncStandbyError(mock sqlmock.Sqlmock, err error) {
	query := "SELECT hostname, datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'm' AND status = 'u';"
	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(err)
}

func setHistoryStandbySyncSnapshotPathHooks(t *testing.T, tempDir string) string {
	t.Helper()

	oldTempDir := historyStandbySyncTempDir
	oldNow := historyStandbySyncNow
	oldPID := historyStandbySyncPID
	historyStandbySyncTempDir = func() string {
		return tempDir
	}
	historyStandbySyncNow = func() time.Time {
		return time.Date(2026, 7, 3, 0, 0, 0, 123456789, time.UTC)
	}
	historyStandbySyncPID = func() int {
		return 42
	}
	t.Cleanup(func() {
		historyStandbySyncTempDir = oldTempDir
		historyStandbySyncNow = oldNow
		historyStandbySyncPID = oldPID
	})
	return filepath.Join(tempDir, "gpbackup_history_20260703000000_123456789_42.db.snap")
}

func createHistoryStandbySyncTempSQLiteDB(t *testing.T, dbPath string) {
	t.Helper()

	db, err := sql.Open("sqlite3", historyStandbySyncSQLiteURI(dbPath, "rwc"))
	if err != nil {
		t.Fatalf("Failed to open temp sqlite db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("CREATE TABLE backups (timestamp TEXT PRIMARY KEY, status TEXT)"); err != nil {
		t.Fatalf("Failed to create temp sqlite table: %v", err)
	}
	if _, err := db.Exec("INSERT INTO backups (timestamp, status) VALUES (?, ?)", "20260703000000", "Success"); err != nil {
		t.Fatalf("Failed to seed temp sqlite table: %v", err)
	}
}

func queryHistoryStandbySyncSnapshotRowCount(t *testing.T, snapshotPath string) int {
	t.Helper()

	db, err := sql.Open("sqlite3", historyStandbySyncSQLiteURI(snapshotPath, "ro"))
	if err != nil {
		t.Fatalf("Failed to open snapshot sqlite db: %v", err)
	}
	defer db.Close()

	var rowCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM backups").Scan(&rowCount); err != nil {
		t.Fatalf("Failed to query snapshot sqlite db: %v", err)
	}
	return rowCount
}
