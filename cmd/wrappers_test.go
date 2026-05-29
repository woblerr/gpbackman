package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

func TestGetHistoryFilePath(t *testing.T) {
	tests := []struct {
		name            string
		historyFilePath string
		want            string
	}{
		{"Empty Path", "", historyFileNameConst},
		{"Non Empty Path", "path/to/" + historyFileNameConst, "path/to/" + historyFileNameConst},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHistoryFilePath(tt.historyFilePath); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestGetHistoryDBPath(t *testing.T) {
	tests := []struct {
		name               string
		historyDBPath      string
		autoLoadHistoryDB  bool
		masterDataDir      string
		coordinatorDataDir string
		want               string
	}{
		{
			name:          "Empty Path",
			historyDBPath: "",
			want:          historyDBNameConst,
		},
		{
			name:          "Non Empty Path",
			historyDBPath: "path/to/" + historyDBNameConst,
			want:          "path/to/" + historyDBNameConst,
		},
		{
			name:               "Env Vars Ignored Without Auto Load",
			autoLoadHistoryDB:  false,
			masterDataDir:      "/master/data",
			coordinatorDataDir: "/coordinator/data",
			want:               historyDBNameConst,
		},
		{
			name:              "Master Data Directory",
			autoLoadHistoryDB: true,
			masterDataDir:     "/master/data",
			want:              filepath.Join("/master/data", historyDBNameConst),
		},
		{
			name:               "Coordinator Data Directory",
			autoLoadHistoryDB:  true,
			coordinatorDataDir: "/coordinator/data",
			want:               filepath.Join("/coordinator/data", historyDBNameConst),
		},
		{
			name:               "Master Data Directory Has Priority",
			autoLoadHistoryDB:  true,
			masterDataDir:      "/master/data",
			coordinatorDataDir: "/coordinator/data",
			want:               filepath.Join("/master/data", historyDBNameConst),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MASTER_DATA_DIRECTORY", tt.masterDataDir)
			t.Setenv("COORDINATOR_DATA_DIRECTORY", tt.coordinatorDataDir)
			if got := getHistoryDBPath(tt.historyDBPath, tt.autoLoadHistoryDB); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestRootAutoLoadHistoryDBFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup(autoLoadHistoryDBFlagName)
	if flag == nil {
		t.Fatalf("Expected %s flag to be registered", autoLoadHistoryDBFlagName)
	}
	if flag.DefValue != "false" {
		t.Errorf("\nDefault value does not match:\n%v\nwant:\n%v", flag.DefValue, "false")
	}
	if !strings.Contains(flag.Usage, "MASTER_DATA_DIRECTORY") ||
		!strings.Contains(flag.Usage, "COORDINATOR_DATA_DIRECTORY") {
		t.Errorf("Flag usage does not mention both data directory environment variables: %s", flag.Usage)
	}
}

func TestRootHelpIncludesAutoLoadHistoryDBFlag(t *testing.T) {
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	defer rootCmd.SetOut(os.Stdout)
	defer rootCmd.SetErr(os.Stderr)

	if err := rootCmd.Help(); err != nil {
		t.Fatalf("Expected root help to render, got: %v", err)
	}
	helpText := output.String()
	for _, value := range []string{
		"--auto-load-history-db",
		"MASTER_DATA_DIRECTORY",
		"COORDINATOR_DATA_DIRECTORY",
	} {
		if !strings.Contains(helpText, value) {
			t.Errorf("Expected help output to contain %q, got:\n%s", value, helpText)
		}
	}
}

func TestFormatBackupDuration(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{"01:00:00", 3600, "01:00:00"},
		{"01:01:01", 3661, "01:01:01"},
		{"00:00:00", 0, "00:00:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBackupDuration(tt.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestRenameHistoryFile(t *testing.T) {
	// Create temp dir.
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	// Create temp file.
	tmpFile := filepath.Join(tmpDir, "testHistoryfile")
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	f.Close()
	// Rename History file.
	err = renameHistoryFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to rename file: %v", err)
	}
	// Check that old file does not exist.
	if _, err := os.Stat(tmpFile); err == nil {
		t.Errorf("Old file still exists")
	} else if !os.IsNotExist(err) {
		t.Errorf("Failed to check if old file exists: %v", err)
	}
	// Check that new file does exist.
	newFile := tmpFile + historyFileNameMigratedSuffixConst
	if _, err := os.Stat(newFile); err != nil {
		if os.IsNotExist(err) {
			t.Errorf("New file does not exist")
		} else {
			t.Errorf("Failed to check if new file exists: %v", err)
		}
	}
}

func TestGetCurrentTimestamp(t *testing.T) {
	result := getCurrentTimestamp()
	_, err := time.Parse(gpbckpconfig.Layout, result)
	if err != nil {
		t.Errorf("Got an error: %v", err)
	}
}

func TestCheckCompatibleFlags(t *testing.T) {
	testCases := []struct {
		name      string
		flagNames []string
		wantErr   bool
	}{
		{
			name:      "No flags changed",
			flagNames: []string{},
			wantErr:   false,
		},
		{
			name:      "One flag changed",
			flagNames: []string{"flag1"},
			wantErr:   false,
		},
		{
			name:      "Multiple flags changed",
			flagNames: []string{"flag1", "flag2"},
			wantErr:   true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			for _, name := range tt.flagNames {
				flags.String(name, "", "")
				flags.Set(name, "")
			}
			err := checkCompatibleFlags(flags, tt.flagNames...)
			if (err != nil) != tt.wantErr {
				t.Errorf("\ncheckCompatibleFlags() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDoRootFlagValidationRejectsHistoryDBWithAutoLoadHistoryDB(t *testing.T) {
	testhelper.SetupTestLogger()

	oldExecOSExit := execOSExit
	oldRootHistoryDB := rootHistoryDB
	defer func() {
		execOSExit = oldExecOSExit
		rootHistoryDB = oldRootHistoryDB
	}()

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String(historyDBFlagName, "", "")
	flags.Bool(autoLoadHistoryDBFlagName, false, "")

	rootHistoryDB = "/tmp/gpbackup_history.db"
	if err := flags.Set(historyDBFlagName, rootHistoryDB); err != nil {
		t.Fatalf("Failed to set %s flag: %v", historyDBFlagName, err)
	}
	if err := flags.Set(autoLoadHistoryDBFlagName, "true"); err != nil {
		t.Fatalf("Failed to set %s flag: %v", autoLoadHistoryDBFlagName, err)
	}

	type exitPanic struct {
		code int
	}
	execOSExit = func(code int) {
		panic(exitPanic{code: code})
	}

	defer func() {
		value := recover()
		if value == nil {
			t.Fatalf("Expected doRootFlagValidation to exit")
		}
		exit, ok := value.(exitPanic)
		if !ok {
			panic(value)
		}
		if exit.code != exitErrorCode {
			t.Fatalf("Unexpected exit code:\n%v\nwant:\n%v", exit.code, exitErrorCode)
		}
	}()

	doRootFlagValidation(flags, false)
}

func TestCheckBackupCanBeUsed(t *testing.T) {
	// Initializes gplog,
	testhelper.SetupTestLogger()
	testCases := []struct {
		name            string
		deleteForce     bool
		skipLocalBackup bool
		backupConfig    gpbckpconfig.BackupConfig
		want            bool
		wantErr         bool
	}{
		{
			name:            "Successful backup with plugin and force, skipLocalBackup true",
			deleteForce:     true,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Successful backup with plugin and without force",
			deleteForce:     false,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Failed backup with plugin and force",
			deleteForce:     true,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusFailure,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Failed backup with plugin and without force",
			deleteForce:     false,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusFailure,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Successful backup without plugin and force",
			deleteForce:     true,
			skipLocalBackup: false,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      "",
				DateDeleted: "",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Successful backup without plugin and without force",
			deleteForce:     false,
			skipLocalBackup: false,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      "",
				DateDeleted: "",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Successful deleted backup with plugin and force",
			deleteForce:     true,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "20240113210000",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Successful deleted backup with plugin and without force",
			deleteForce:     false,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "20240113210000",
			},
			want:    false,
			wantErr: false,
		},
		{
			name:            "Invalid backup status with plugin and without force",
			deleteForce:     false,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      "some_status",
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Successful backup with plugin with deletion in progress and force",
			deleteForce:     true,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: gpbckpconfig.DateDeletedInProgress,
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Successful backup with plugin with deletion in progress and without force",
			deleteForce:     false,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: gpbckpconfig.DateDeletedInProgress,
			},
			want:    false,
			wantErr: false,
		},
		{
			name:            "Successful backup with plugin with invalid deletion date and without force",
			deleteForce:     false,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "some date",
			},
			want:    true,
			wantErr: false,
		},
		{
			name:            "Successful backup with plugin with invalid skipLocalBackup variable",
			deleteForce:     false,
			skipLocalBackup: false,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      gpbckpconfig.BackupS3Plugin,
				DateDeleted: "some date",
			},
			want:    false,
			wantErr: true,
		},
		{
			name:            "Successful backup without plugin with invalid skipLocalBackup variable",
			deleteForce:     false,
			skipLocalBackup: true,
			backupConfig: gpbckpconfig.BackupConfig{
				Status:      gpbckpconfig.BackupStatusSuccess,
				Plugin:      "",
				DateDeleted: "some date",
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkBackupCanBeUsed(tt.deleteForce, tt.skipLocalBackup, tt.backupConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("\ncheckBackupCanBeUsed() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("\ncheckBackupCanBeUsed got:\n%v\nwant:\n%v\n", got, tt.want)
			}
		})
	}
}

func TestCheckBackupType(t *testing.T) {
	tests := []struct {
		name      string
		inputType string
		wantErr   bool
	}{
		{
			name:      "Valid Backup Type",
			inputType: gpbckpconfig.BackupTypeFull,
			wantErr:   false,
		},
		{
			name:      "Invalid Backup Type",
			inputType: "InvalidType",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkBackupType(tt.inputType); (err != nil) != tt.wantErr {
				t.Errorf("checkBackupType() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestGetBackupMasterDir(t *testing.T) {
	// Create a unique temp directory for this test to avoid conflicts with other tests
	tempDir, err := os.MkdirTemp("", "gpbackman-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	// Cleanup the entire test directory at the end, not individual subdirectories
	defer os.RemoveAll(tempDir)
	tests := []struct {
		name                  string
		testDir               string
		backupDir             string
		backupDataBackupDir   string
		backupDataDBName      string
		wantBackupMasterDir   string
		wantSegPrefix         string
		wantIsSingleBackupDir bool
		wantErr               bool
	}{
		{
			name:                  "BackupDir is set and valid",
			testDir:               filepath.Join(tempDir, "segPrefix", "segment-1", "backups"),
			backupDir:             filepath.Join(tempDir, "segPrefix"),
			backupDataBackupDir:   "",
			backupDataDBName:      "",
			wantBackupMasterDir:   filepath.Join(tempDir, "segPrefix", "segment-1"),
			wantSegPrefix:         "segment",
			wantIsSingleBackupDir: false,
			wantErr:               false,
		},
		{
			name:                  "BackupDataBackupDir is set and valid",
			testDir:               filepath.Join(tempDir, "segPrefix", "segment-1", "backups"),
			backupDir:             "",
			backupDataBackupDir:   filepath.Join(tempDir, "segPrefix"),
			backupDataDBName:      "",
			wantBackupMasterDir:   filepath.Join(tempDir, "segPrefix", "segment-1"),
			wantSegPrefix:         "segment",
			wantIsSingleBackupDir: false,
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.MkdirAll(tt.testDir, 0o755)
			if err != nil {
				t.Fatalf("Failed to create test directory structure: %v", err)
			}
			gotBackupMasterDir, gotSegPrefix, gotIsSingleBackupDir, err := getBackupMasterDir(tt.backupDir, tt.backupDataBackupDir, tt.backupDataDBName)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckMasterBackupDir() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
			if gotBackupMasterDir != tt.wantBackupMasterDir {
				t.Errorf("getBackupMasterDir() gotMasterDir:\n%v\nwantBackupMasterDir\n%v", gotBackupMasterDir, tt.wantBackupMasterDir)
			}
			if gotSegPrefix != tt.wantSegPrefix {
				t.Errorf("getBackupMasterDir() gotSegPrefix\n%v\nwantSegPrefix\n%v", gotSegPrefix, tt.wantSegPrefix)
			}
			if gotIsSingleBackupDir != tt.wantIsSingleBackupDir {
				t.Errorf("getBackupMasterDir() gotIsSingleBackupDir\n%v\nwantIsSingleBackupDir\n%v", gotIsSingleBackupDir, tt.wantIsSingleBackupDir)
			}
		})
	}
}

func TestCheckSingleBackupDir(t *testing.T) {
	backupDir := "/path/to/backup"
	segPrefix := "seg"
	segID := "1"

	tests := []struct {
		name              string
		isSingleBackupDir bool
		want              string
	}{
		{
			name:              "Is single backup dir",
			isSingleBackupDir: true,
			want:              backupDir,
		},
		{
			name:              "Is not single backup dir",
			isSingleBackupDir: false,
			want:              filepath.Join(backupDir, fmt.Sprintf("%s%s", segPrefix, segID)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkSingleBackupDir(backupDir, segPrefix, segID, tt.isSingleBackupDir); got != tt.want {
				t.Errorf("checkSingleBackupDir()\n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

func TestGetBackupSegmentDir(t *testing.T) {
	segPrefix := "seg"
	segID := "1"

	tests := []struct {
		name                string
		backupDir           string
		backupDataBackupDir string
		backupDataDir       string
		isSingleBackupDir   bool
		want                string
		wantErr             bool
	}{
		{
			name:                "Test when backupDir is not empty",
			backupDir:           "/path/to/backupDir",
			backupDataBackupDir: "",
			backupDataDir:       "",
			isSingleBackupDir:   true,
			want:                "/path/to/backupDir",
			wantErr:             false,
		},
		{
			name:                "Test when backupDataBackupDir is not empty",
			backupDir:           "",
			backupDataBackupDir: "/path/to/backupDataBackupDir",
			backupDataDir:       "",
			isSingleBackupDir:   true,
			want:                "/path/to/backupDataBackupDir",
			wantErr:             false,
		},
		{
			name:                "Test when backupDataDir is not empty",
			backupDir:           "",
			backupDataBackupDir: "",
			backupDataDir:       "/path/to/backupDataDir",
			isSingleBackupDir:   true,
			want:                "/path/to/backupDataDir",
			wantErr:             false,
		},
		{
			name:                "Test error when all backup directories are empty",
			backupDir:           "",
			backupDataBackupDir: "",
			backupDataDir:       "",
			isSingleBackupDir:   true,
			want:                "",
			wantErr:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBackupSegmentDir(tt.backupDir, tt.backupDataBackupDir, tt.backupDataDir, segPrefix, segID, tt.isSingleBackupDir)
			if got != tt.want {
				t.Errorf("getBackupSegmentDir() got:\n%v\nwant:\n%v", got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("getBackupSegmentDir() error:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckLocalBackupStatus(t *testing.T) {
	tests := []struct {
		name            string
		skipLocalBackup bool
		isLocalBackup   bool
		wantErr         bool
	}{
		{
			name:            "Skip local and local backup",
			skipLocalBackup: true,
			isLocalBackup:   true,
			wantErr:         true,
		},
		{
			name:            "Skip local and plugin backup",
			skipLocalBackup: true,
			isLocalBackup:   false,
			wantErr:         false,
		},
		{
			name:            "Do not skip local and local backup",
			skipLocalBackup: false,
			isLocalBackup:   true,
			wantErr:         false,
		},
		{
			name:            "Do not skip local and plugin backup",
			skipLocalBackup: false,
			isLocalBackup:   false,
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkLocalBackupStatus(tt.skipLocalBackup, tt.isLocalBackup)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkLocalBackupStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
