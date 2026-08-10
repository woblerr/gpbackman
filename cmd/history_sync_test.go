package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
)

func TestHistorySyncCommandRegistration(t *testing.T) {
	command, _, err := rootCmd.Find([]string{"history-sync"})
	if err != nil {
		t.Fatalf("Expected history-sync command to be registered, got: %v", err)
	}
	if command != historySyncCmd {
		t.Fatalf("Unexpected command registered for history-sync: %p", command)
	}
	if historySyncCmd.Use != "history-sync" {
		t.Fatalf("Unexpected command use: %s", historySyncCmd.Use)
	}
	if err := historySyncCmd.Args(historySyncCmd, []string{"unexpected"}); err == nil {
		t.Fatal("Expected history-sync to reject positional arguments")
	}
	for _, flagName := range []string{historyDBFlagName, autoLoadHistoryDBFlagName} {
		if flag := historySyncCmd.InheritedFlags().Lookup(flagName); flag == nil {
			t.Fatalf("Expected history-sync to inherit --%s", flagName)
		}
	}
}

func TestHistorySyncTimeoutFlagRegisteredOnlyOnSyncCommands(t *testing.T) {
	syncCommands := map[string]bool{
		"history-sync":  true,
		"backup-delete": true,
		"backup-clean":  true,
		"history-clean": true,
	}

	for _, command := range rootCmd.Commands() {
		flag := command.PersistentFlags().Lookup(historySyncStandbyTimeoutFlagName)
		if syncCommands[command.Name()] {
			if flag == nil {
				t.Fatalf("Expected --%s on %s", historySyncStandbyTimeoutFlagName, command.Name())
			}
			if flag.DefValue != "300" {
				t.Fatalf("Unexpected timeout default on %s: %s", command.Name(), flag.DefValue)
			}
			delete(syncCommands, command.Name())
			continue
		}
		if flag != nil {
			t.Fatalf("Did not expect --%s on %s", historySyncStandbyTimeoutFlagName, command.Name())
		}
	}
	if len(syncCommands) != 0 {
		t.Fatalf("Expected timeout flag on commands: %v", syncCommands)
	}
	if flag := rootCmd.PersistentFlags().Lookup(historySyncStandbyTimeoutFlagName); flag != nil {
		t.Fatalf("Did not expect --%s as a root flag", historySyncStandbyTimeoutFlagName)
	}
}

func TestHistorySyncTimeoutFlagAcceptsIntegerSeconds(t *testing.T) {
	flag := historySyncCmd.PersistentFlags().Lookup(historySyncStandbyTimeoutFlagName)
	if flag == nil {
		t.Fatalf("Expected --%s flag", historySyncStandbyTimeoutFlagName)
	}
	oldTimeoutSeconds := historyStandbySyncTimeoutSeconds
	oldChanged := flag.Changed
	t.Cleanup(func() {
		historyStandbySyncTimeoutSeconds = oldTimeoutSeconds
		flag.Changed = oldChanged
	})

	if err := historySyncCmd.PersistentFlags().Set(historySyncStandbyTimeoutFlagName, "600"); err != nil {
		t.Fatalf("Expected integer timeout to be accepted: %v", err)
	}
	if historyStandbySyncTimeoutSeconds != 600 {
		t.Fatalf("Unexpected timeout value: %d", historyStandbySyncTimeoutSeconds)
	}
	for _, value := range []string{"1.5", "5m"} {
		if err := flag.Value.Set(value); err == nil {
			t.Fatalf("Expected timeout value %q to be rejected", value)
		}
	}
}

func TestHistorySyncTimeoutValidation(t *testing.T) {
	tests := []struct {
		name      string
		timeout   int
		wantExits int
	}{
		{name: "Minimum", timeout: 1},
		{name: "Maximum", timeout: 86400},
		{name: "Zero", timeout: 0, wantExits: 1},
		{name: "Negative", timeout: -1, wantExits: 1},
		{name: "Above Maximum", timeout: 86401, wantExits: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testhelper.SetupTestLogger()
			oldTimeoutSeconds := historyStandbySyncTimeoutSeconds
			historyStandbySyncTimeoutSeconds = tt.timeout
			t.Cleanup(func() {
				historyStandbySyncTimeoutSeconds = oldTimeoutSeconds
			})
			exitCalls := 0
			setHistoryStandbySyncExecOSExit(t, func(code int) {
				if code != exitErrorCode {
					t.Fatalf("Unexpected exit code: %d", code)
				}
				exitCalls++
			})

			doRootFlagValidation(historySyncCmd.Flags(), false)

			if exitCalls != tt.wantExits {
				t.Fatalf("Unexpected validation exit count: %d, want: %d", exitCalls, tt.wantExits)
			}
		})
	}
}

func TestHistorySyncCommandSucceedsAndPassesEmptyClusterDBName(t *testing.T) {
	testhelper.SetupTestLogger()

	syncCalls := 0
	setHistorySyncStandbyHook(t, func(clusterDBName string) (string, error) {
		syncCalls++
		if clusterDBName != "" {
			t.Fatalf("Unexpected cluster database name: %q", clusterDBName)
		}
		return "", nil
	})
	exitCalls := 0
	setHistoryStandbySyncExecOSExit(t, func(code int) {
		exitCalls++
	})

	executeHistorySyncCommand(t)

	if exitCalls != 0 {
		t.Fatalf("history-sync unexpectedly exited %d times", exitCalls)
	}
	if syncCalls != 1 {
		t.Fatalf("Expected one history sync call, got: %d", syncCalls)
	}
}

func TestHistorySyncCommandFailsForSyncErrorsAndSkipReasons(t *testing.T) {
	tests := []struct {
		name       string
		skipReason string
		syncErr    error
		wantCause  string
	}{
		{
			name:      "Sync Error",
			syncErr:   errors.New("discovery failed"),
			wantCause: "discovery failed",
		},
		{
			name:       "No Up Standby",
			skipReason: "no up standby coordinator found",
			wantCause:  "no up standby coordinator found",
		},
		{
			name:       "Ineligible Source",
			skipReason: "custom history db is not cluster history db",
			wantCause:  "is not cluster history db",
		},
		{
			name:       "Default Working Directory Source",
			skipReason: "using default working-directory history db",
			wantCause:  "using default working-directory history db",
		},
		{
			name:      "Transport Timeout",
			syncErr:   fmt.Errorf("standby history sync transport timed out after 300 seconds: %w", context.DeadlineExceeded),
			wantCause: "timed out after 300 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, testStderr, _ := testhelper.SetupTestLogger()
			syncCalls := 0
			setHistorySyncStandbyHook(t, func(clusterDBName string) (string, error) {
				syncCalls++
				if clusterDBName != "" {
					t.Fatalf("Unexpected cluster database name: %q", clusterDBName)
				}
				return tt.skipReason, tt.syncErr
			})
			exitCalls := 0
			setHistoryStandbySyncExecOSExit(t, func(code int) {
				if code != exitErrorCode {
					t.Fatalf("Unexpected exit code: %d", code)
				}
				exitCalls++
			})

			executeHistorySyncCommand(t)

			if exitCalls != 1 {
				t.Fatalf("Expected one error exit, got: %d", exitCalls)
			}
			if syncCalls != 1 {
				t.Fatalf("Expected one history sync call, got: %d", syncCalls)
			}
			logOutput := string(testStderr.Contents())
			if !strings.Contains(logOutput, "Unable to synchronize history db. Error:") {
				t.Fatalf("Expected history-sync error message, got: %s", logOutput)
			}
			if !strings.Contains(logOutput, tt.wantCause) {
				t.Fatalf("Expected history-sync error to contain %q, got: %s", tt.wantCause, logOutput)
			}
		})
	}
}

func TestHistorySyncCommandRootValidationRejectsMissingOrIncompatibleSourceFlags(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name string
		args func(*testing.T) []string
	}{
		{
			name: "Missing Explicit Source",
			args: func(t *testing.T) []string {
				return []string{"--" + historyDBFlagName, filepath.Join(t.TempDir(), historyDBNameConst)}
			},
		},
		{
			name: "Incompatible Source Flags",
			args: func(t *testing.T) []string {
				path := filepath.Join(t.TempDir(), historyDBNameConst)
				createHistoryStandbySyncTempSQLiteDB(t, path)
				return []string{"--" + historyDBFlagName, path, "--" + autoLoadHistoryDBFlagName}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHistorySyncStandbyHook(t, func(string) (string, error) {
				t.Fatal("History sync should not run when root validation fails")
				return "", nil
			})
			type exitPanic struct{ code int }
			setHistoryStandbySyncExecOSExit(t, func(code int) {
				panic(exitPanic{code: code})
			})

			defer func() {
				value := recover()
				if value == nil {
					t.Fatal("Expected root validation to exit")
				}
				exit, ok := value.(exitPanic)
				if !ok {
					panic(value)
				}
				if exit.code != exitErrorCode {
					t.Fatalf("Unexpected exit code: %d", exit.code)
				}
			}()

			executeHistorySyncCommand(t, tt.args(t)...)
		})
	}
}

func executeHistorySyncCommand(t *testing.T, args ...string) {
	t.Helper()

	resetHistorySyncSourceFlags(t)
	rootCmd.SetArgs(append([]string{"history-sync"}, args...))
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Expected history-sync command execution to return no error, got: %v", err)
	}
}

func resetHistorySyncSourceFlags(t *testing.T) {
	t.Helper()

	flags := rootCmd.PersistentFlags()
	historyDBFlag := flags.Lookup(historyDBFlagName)
	autoLoadHistoryDBFlag := flags.Lookup(autoLoadHistoryDBFlagName)
	oldRootHistoryDB := rootHistoryDB
	oldRootAutoLoadHistoryDB := rootAutoLoadHistoryDB
	oldHistoryDBChanged := historyDBFlag.Changed
	oldAutoLoadHistoryDBChanged := autoLoadHistoryDBFlag.Changed

	rootHistoryDB = ""
	rootAutoLoadHistoryDB = false
	historyDBFlag.Changed = false
	autoLoadHistoryDBFlag.Changed = false

	t.Cleanup(func() {
		rootHistoryDB = oldRootHistoryDB
		rootAutoLoadHistoryDB = oldRootAutoLoadHistoryDB
		historyDBFlag.Changed = oldHistoryDBChanged
		autoLoadHistoryDBFlag.Changed = oldAutoLoadHistoryDBChanged
	})
}

func setHistorySyncStandbyHook(t *testing.T, hook func(string) (string, error)) {
	t.Helper()

	oldHook := syncHistoryStandbyForCommand
	syncHistoryStandbyForCommand = hook
	t.Cleanup(func() {
		syncHistoryStandbyForCommand = oldHook
	})
}

func TestHistorySyncCommandHasNoLocalNoHistorySyncStandbyFlag(t *testing.T) {
	if flag := historySyncCmd.PersistentFlags().Lookup(noHistorySyncStandbyFlagName); flag != nil {
		t.Fatalf("Expected history-sync not to define --%s", noHistorySyncStandbyFlagName)
	}
	if flag := historySyncCmd.LocalFlags().Lookup(noHistorySyncStandbyFlagName); flag != nil {
		t.Fatalf("Expected history-sync not to define a local --%s", noHistorySyncStandbyFlagName)
	}
}

func TestHistorySyncCommandHelpContainsSourceFlagsWithoutNoHistorySyncStandby(t *testing.T) {
	var output bytes.Buffer
	historySyncCmd.SetOut(&output)
	t.Cleanup(func() {
		historySyncCmd.SetOut(nil)
	})

	if err := historySyncCmd.Help(); err != nil {
		t.Fatalf("Expected history-sync help to succeed, got: %v", err)
	}

	help := output.String()
	for _, flagName := range []string{historyDBFlagName, autoLoadHistoryDBFlagName} {
		if !strings.Contains(help, "--"+flagName) {
			t.Fatalf("Expected history-sync help to include --%s", flagName)
		}
	}
	if strings.Contains(help, "--"+noHistorySyncStandbyFlagName) {
		t.Fatalf("Expected history-sync help not to include --%s", noHistorySyncStandbyFlagName)
	}
}
