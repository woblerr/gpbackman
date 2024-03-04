package cmd

import (
	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gpbackup/history"
	"github.com/spf13/cobra"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

var historyMigrateCmd = &cobra.Command{
	Use:   "history-migrate",
	Short: "Migrate history database",
	Long: `Migrate data from gpbackup_history.yaml to gpbackup_history.db SQLite history database.

The data from the gpbackup_history.yaml file will be uploaded to gpbackup_history.db SQLite history database.
If the gpbackup_history.db file does not exist, it will be created.
The gpbackup_history.yaml file will be renamed to gpbackup_history.yaml.migrated.

The gpbackup_history.db file location can be set using the  --history-db option.
Can be specified only once. The full path to the file is required.

The gpbackup_history.yaml file location can be set using the  --history-file option.
Can be specified multiple times. The full path to the file is required.

If no --history-file and/or --history-db options are specified, the files will be searched in the current directory.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// No need to check historyDB existence.
		doRootFlagValidation(cmd.Flags(), false)
		doMigrateHistory()
	},
}

func init() {
	rootCmd.AddCommand(historyMigrateCmd)
}

func doMigrateHistory() {
	err := migrateHistory()
	if err != nil {
		execOSExit(exitErrorCode)
	}
}

func migrateHistory() error {
	hDB, err := history.InitializeHistoryDatabase(getHistoryDBPath(rootHistoryDB))
	if err != nil {
		gplog.Error(textmsg.ErrorTextUnableInitHistoryDB(err))
		return err
	}
	defer func() {
		closeErr := hDB.Close()
		if closeErr != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryDB("close", closeErr))
		}
	}()
	for _, historyFile := range rootHistoryFiles {
		hFile := getHistoryFilePath(historyFile)
		historyData, err := gpbckpconfig.ReadHistoryFile(hFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("read", err))
			return err
		}
		parseHData, err := gpbckpconfig.ParseResult(historyData)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("parse", err))
			return err
		}
		for _, backupConfig := range parseHData.BackupConfigs {
			hBackupConfig := gpbckpconfig.ConvertToHistoryBackupConfig(backupConfig)
			err = history.StoreBackupHistory(hDB, &hBackupConfig)
			if err != nil {
				gplog.Error(textmsg.ErrorTextUnableWriteIntoHistoryDB(err))
				return err
			}
		}
		err = renameHistoryFile(hFile)
		if err != nil {
			gplog.Error(textmsg.ErrorTextUnableActionHistoryFile("rename", err))
			return err
		}
	}
	return nil
}
