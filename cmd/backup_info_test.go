package cmd

import (
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/spf13/pflag"
)

func TestBackupInfoDatabaseFlag(t *testing.T) {
	flag := backupInfoCmd.Flags().Lookup(databaseFlagName)
	if flag == nil {
		t.Fatalf("Expected %s flag to be registered", databaseFlagName)
	}
	if flag.DefValue != "" {
		t.Errorf("Unexpected default value: %q", flag.DefValue)
	}
	if !strings.Contains(flag.Usage, "specified database") {
		t.Errorf("Flag usage does not describe database filtering: %s", flag.Usage)
	}

	helpText := backupInfoCmd.UsageString()
	if !strings.Contains(helpText, "--"+databaseFlagName+" string") {
		t.Errorf("Command help does not include --%s string:\n%s", databaseFlagName, helpText)
	}
	if !strings.Contains(helpText, flag.Usage) {
		t.Errorf("Command help does not include database flag description:\n%s", helpText)
	}
}

func TestBackupInfoDatabaseFlagRequiresValue(t *testing.T) {
	rootCmd.SetArgs([]string{"backup-info", "--" + databaseFlagName})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected --database without a value to return an error")
	}
	if !strings.Contains(err.Error(), "flag needs an argument") {
		t.Fatalf("Unexpected flag error: %v", err)
	}
}

func TestDoBackupInfoDatabaseFlagValidation(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name        string
		database    string
		setDatabase bool
		wantExit    bool
	}{
		{name: "Absent database flag"},
		{name: "Non-empty database", database: `"Customer's DB"`, setDatabase: true},
		{name: "Explicit empty database", setDatabase: true, wantExit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDatabase := backupInfoDatabase
			oldTimestamp := backupInfoTimestamp
			oldExecOSExit := execOSExit
			backupInfoDatabase = tt.database
			backupInfoTimestamp = ""
			t.Cleanup(func() {
				backupInfoDatabase = oldDatabase
				backupInfoTimestamp = oldTimestamp
				execOSExit = oldExecOSExit
			})

			flags := pflag.NewFlagSet("backup-info", pflag.ContinueOnError)
			flags.String(databaseFlagName, "", "")
			if tt.setDatabase {
				if err := flags.Set(databaseFlagName, tt.database); err != nil {
					t.Fatalf("Failed to set %s: %v", databaseFlagName, err)
				}
			}

			exited := false
			exitCode := 0
			execOSExit = func(code int) {
				exited = true
				exitCode = code
			}
			doBackupInfoFlagValidation(flags)

			if exited != tt.wantExit {
				t.Errorf("Exit status = %v, want %v", exited, tt.wantExit)
			}
			if tt.wantExit && exitCode != exitErrorCode {
				t.Errorf("Exit code = %d, want %d", exitCode, exitErrorCode)
			}
		})
	}
}

func TestBackupInfoDatabaseFlagCompatibleWithTimestamp(t *testing.T) {
	testhelper.SetupTestLogger()

	oldDatabase := backupInfoDatabase
	oldTimestamp := backupInfoTimestamp
	oldExecOSExit := execOSExit
	backupInfoDatabase = `"Customer's DB"`
	backupInfoTimestamp = "20240101120000"
	t.Cleanup(func() {
		backupInfoDatabase = oldDatabase
		backupInfoTimestamp = oldTimestamp
		execOSExit = oldExecOSExit
	})

	flags := pflag.NewFlagSet("backup-info", pflag.ContinueOnError)
	flags.String(databaseFlagName, "", "")
	flags.String(timestampFlagName, "", "")
	if err := flags.Set(databaseFlagName, backupInfoDatabase); err != nil {
		t.Fatalf("Failed to set %s: %v", databaseFlagName, err)
	}
	if err := flags.Set(timestampFlagName, backupInfoTimestamp); err != nil {
		t.Fatalf("Failed to set %s: %v", timestampFlagName, err)
	}

	execOSExit = func(code int) {
		t.Fatalf("doBackupInfoFlagValidation unexpectedly exited with code %d", code)
	}
	doBackupInfoFlagValidation(flags)
}
