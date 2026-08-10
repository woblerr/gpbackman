package cmd

import (
	"errors"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/spf13/cobra"
	"github.com/woblerr/gpbackman/textmsg"
)

var syncHistoryStandbyForCommand = syncHistoryStandby

var historySyncCmd = &cobra.Command{
	Use:   "history-sync",
	Short: "Synchronize the cluster history database to the standby coordinator",
	Long: `Synchronize the cluster gpbackup_history.db to an up standby coordinator.

The source history database must be the cluster history database.
Use --history-db to select an explicit source file, or --auto-load-history-db to resolve it from the coordinator data directory.
The command fails when no standby coordinator is available or when the source database is not eligible for synchronization.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doRootFlagValidation(cmd.Flags(), checkFileExistsConst)
		doHistorySync()
	},
}

func init() {
	rootCmd.AddCommand(historySyncCmd)
	historySyncCmd.PersistentFlags().IntVar(
		&historyStandbySyncTimeoutSeconds,
		historySyncStandbyTimeoutFlagName,
		historySyncStandbyTimeoutDefault,
		"shared rsync and remote install timeout in seconds; must be an integer between 1 and 86400",
	)
}

func doHistorySync() {
	logHeadersDebug()
	skipReason, err := syncHistoryStandbyForCommand("")
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableActionHistoryDB("synchronize", err))
		execOSExit(exitErrorCode)
		return
	}
	if skipReason != "" {
		gplog.Error("%s", textmsg.ErrorTextUnableActionHistoryDB("synchronize", errors.New(skipReason)))
		execOSExit(exitErrorCode)
		return
	}
}
