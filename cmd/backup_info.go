package cmd

import (
	"database/sql"
	"os"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/jedib0t/go-pretty/table"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

// Flags for the gpbackman backup-info command (backupInfoCmd)
var (
	backupInfoShowDeleted      bool
	backupInfoShowFailed       bool
	backupInfoBackupTypeFilter string
	backupInfoTableNameFilter  string
	backupInfoSchemaNameFilter string
	backupInfoExcludeFilter    bool
	backupInfoTimestamp        string
)

// Options for the backup-info command.
type BackupInfoOptions struct {
	ShowDeleted      bool
	ShowFailed       bool
	BackupTypeFilter string
	TableNameFilter  string
	SchemaNameFilter string
	ExcludeFilter    bool
	Timestamp        string
}

var backupInfoCmd = &cobra.Command{
	Use:   "backup-info",
	Short: "Display information about backups",
	Long: `Display information about backups.

By default, only active backups or backups with deletion status "In progress" from gpbackup_history.db are displayed.

To additional display deleted backups, use the --deleted option.
To additional display failed backups, use the --failed option.
To display all backups, use --deleted and --failed options together.

To display backups of a specific type, use the --type option.

To display backups that include the specified table, use the --table option. 
The formatting rules for <schema>.<table> match those of the --include-table option in gpbackup.

To display backups that include the specified schema, use the --schema option. 
The formatting rules for <schema> match those of the --include-schema option in gpbackup.

To display backups that exclude the specified table, use the --table and --exclude options. 
The formatting rules for <schema>.<table> match those of the --exclude-table option in gpbackup.

To display backups that exclude the specified schema, use the --schema and --exclude options. 
The formatting rules for <schema>match those of the --exclude-schema option in gpbackup.

The gpbackup_history.db file location can be set using the --history-db option.
Can be specified only once. The full path to the file is required.
If the --history-db option is not specified, the history database will be searched in the current directory.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		doRootFlagValidation(cmd.Flags(), checkFileExistsConst)
		doBackupInfoFlagValidation(cmd.Flags())
		doBackupInfo()
	},
}

func init() {
	rootCmd.AddCommand(backupInfoCmd)
	backupInfoCmd.Flags().StringVar(
		&backupInfoTimestamp,
		timestampFlagName,
		"",
		"show dependent backups for the specified backup timestamp",
	)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowDeleted,
		deletedFlagName,
		false,
		"show deleted backups",
	)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoShowFailed,
		failedFlagName,
		false,
		"show failed backups",
	)
	backupInfoCmd.Flags().StringVar(
		&backupInfoBackupTypeFilter,
		typeFlagName,
		"",
		"backup type filter (full, incremental, data-only, metadata-only)",
	)
	backupInfoCmd.Flags().StringVar(
		&backupInfoTableNameFilter,
		tableFlagName,
		"",
		"show backups that include the specified table (format <schema>.<table>)",
	)
	backupInfoCmd.Flags().StringVar(
		&backupInfoSchemaNameFilter,
		schemaFlagName,
		"",
		"show backups that include the specified schema",
	)
	backupInfoCmd.Flags().BoolVar(
		&backupInfoExcludeFilter,
		excludeFlagName,
		false,
		"show backups that exclude the specific table (format <schema>.<table>) or schema",
	)
}

// These flag checks are applied only for backup-info commands.
func doBackupInfoFlagValidation(flags *pflag.FlagSet) {
	var err error
	if flags.Changed(timestampFlagName) {
		err = gpbckpconfig.CheckTimestamp(backupInfoTimestamp)
		if err != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableValidateFlag(backupInfoTimestamp, timestampFlagName, err))
			execOSExit(exitErrorCode)
		}
		// --timestamp is not compatible with --type, --table, --schema, --exclude, --failed, --deleted
		err = checkCompatibleFlags(flags, timestampFlagName,
			typeFlagName, tableFlagName, schemaFlagName, excludeFlagName, failedFlagName, deletedFlagName)
		if err != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableCompatibleFlags(err, timestampFlagName, typeFlagName, tableFlagName, schemaFlagName, excludeFlagName))
			execOSExit(exitErrorCode)
		}
	}
	// If type is specified and have correct values.
	if flags.Changed(typeFlagName) {
		err = checkBackupType(backupInfoBackupTypeFilter)
		if err != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableValidateFlag(backupInfoBackupTypeFilter, typeFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
	// table flag and schema flags cannot be used together.
	err = checkCompatibleFlags(flags, tableFlagName, schemaFlagName)
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableCompatibleFlags(err, tableFlagName, schemaFlagName))
		execOSExit(exitErrorCode)
	}
	// If table is specified and have correct values.
	if flags.Changed(tableFlagName) {
		err = gpbckpconfig.CheckTableFQN(backupInfoTableNameFilter)
		if err != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableValidateFlag(backupInfoTableNameFilter, tableFlagName, err))
			execOSExit(exitErrorCode)
		}
	}
	// If exclude flag is specified, but table or schema flag is not.
	if flags.Changed(excludeFlagName) && !flags.Changed(tableFlagName) && !flags.Changed(schemaFlagName) {
		gplog.Error("%s", textmsg.ErrorTextUnableValidateValue(textmsg.ErrorNotIndependentFlagsError(), tableFlagName, schemaFlagName))
		execOSExit(exitErrorCode)
	}
}

func doBackupInfo() {
	logHeadersDebug()
	err := backupInfo()
	if err != nil {
		execOSExit(exitErrorCode)
	}
}

func backupInfo() error {
	t := table.NewWriter()
	initTable(t)
	hDB, err := gpbckpconfig.OpenHistoryDB(getHistoryDBPath(rootHistoryDB))
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableActionHistoryDB("open", err))
		return err
	}
	defer func() {
		closeErr := hDB.Close()
		if closeErr != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableActionHistoryDB("close", closeErr))
		}
	}()
	opts := BackupInfoOptions{
		ShowDeleted:      backupInfoShowDeleted,
		ShowFailed:       backupInfoShowFailed,
		BackupTypeFilter: backupInfoBackupTypeFilter,
		TableNameFilter:  backupInfoTableNameFilter,
		SchemaNameFilter: backupInfoSchemaNameFilter,
		ExcludeFilter:    backupInfoExcludeFilter,
		Timestamp:        backupInfoTimestamp,
	}
	err = backupInfoDB(opts, hDB, t)
	if err != nil {
		return err
	}
	t.Render()
	return nil
}

func backupInfoDB(opts BackupInfoOptions, hDB *sql.DB, t table.Writer) error {
	// List all according to showDeleted/showFailed
	if opts.Timestamp == "" {
		backupList, err := gpbckpconfig.GetBackupNamesDB(opts.ShowDeleted, opts.ShowFailed, hDB)
		if err != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableReadHistoryDB(err))
			return err
		}
		for _, backupName := range backupList {
			backupData, err := gpbckpconfig.GetBackupDataDB(backupName, hDB)
			if err != nil {
				gplog.Error("%s", textmsg.ErrorTextUnableGetBackupInfo(backupName, err))
				return err
			}
			addBackupToTable(opts.BackupTypeFilter, opts.TableNameFilter, opts.SchemaNameFilter, opts.ExcludeFilter, backupData, t)
		}
		return nil
	}
	// Timestamp mode: show base backup and only its dependent backups
	// Verify base backup exists
	baseBackupData, err := gpbckpconfig.GetBackupDataDB(opts.Timestamp, hDB)
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableGetBackupInfo(opts.Timestamp, err))
		return err
	}
	addBackupToTable("", "", "", false, baseBackupData, t)
	backupDependenciesList, err := gpbckpconfig.GetBackupDependencies(opts.Timestamp, hDB)
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableReadHistoryDB(err))
		return err
	}
	for _, depTimestamp := range backupDependenciesList {
		backupData, err := gpbckpconfig.GetBackupDataDB(depTimestamp, hDB)
		if err != nil {
			gplog.Error("%s", textmsg.ErrorTextUnableGetBackupInfo(depTimestamp, err))
			return err
		}
		addBackupToTable("", "", "", false, backupData, t)
	}
	return nil
}

func initTable(t table.Writer) {
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDefault)
	t.Style().Options.DrawBorder = false
	t.AppendHeader(table.Row{
		"timestamp",
		"date",
		"status",
		"database",
		"type",
		"object filtering",
		"plugin",
		"duration",
		"date deleted",
	})
	t.SortBy([]table.SortBy{{Name: "timestamp", Mode: table.Dsc}})
}

// addBackupToTable adds a backup to the table for displaying.
//
// If errors occur, they are logged, but they are not returned.
// The main idea is to show the maximum available information and display all errors that occur.
// But do not fall when errors occur. So, display anyway.
func addBackupToTable(backupTypeFilter, backupTableFilter, backupSchemaFilter string, backupExcludeFilter bool, backupData gpbckpconfig.BackupConfig, t table.Writer) {
	var matchToObjectFilter bool
	backupDate, err := backupData.GetBackupDate()
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableGetBackupValue("date", backupData.Timestamp, err))
	}
	backupType, err := backupData.GetBackupType()
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableGetBackupValue("type", backupData.Timestamp, err))
	}
	backupFilter, err := backupData.GetObjectFilteringInfo()
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableGetBackupValue("object filtering", backupData.Timestamp, err))
	}
	backupDuration, err := backupData.GetBackupDuration()
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableGetBackupValue("duration", backupData.Timestamp, err))
	}
	backupDateDeleted, err := backupData.GetBackupDateDeleted()
	if err != nil {
		gplog.Error("%s", textmsg.ErrorTextUnableGetBackupValue("date deletion", backupData.Timestamp, err))
	}
	matchToObjectFilter = backupData.CheckObjectFilteringExists(backupTableFilter, backupSchemaFilter, backupFilter, backupExcludeFilter)
	if (backupTypeFilter == "" || backupTypeFilter == backupType) && matchToObjectFilter {
		t.AppendRow([]interface{}{
			backupData.Timestamp,
			backupDate,
			backupData.Status,
			backupData.DatabaseName,
			backupType,
			backupFilter,
			backupData.Plugin,
			formatBackupDuration(backupDuration),
			backupDateDeleted,
		})
	}
}
