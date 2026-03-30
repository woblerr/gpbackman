package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

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
