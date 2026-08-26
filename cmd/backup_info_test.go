package cmd

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greenplum-db/gp-common-go-libs/testhelper"
	"github.com/greenplum-db/gpbackup/history"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/pflag"
	"github.com/woblerr/gpbackman/gpbckpconfig"
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

	for _, text := range []string{
		"Without the --database option, backups for all databases are displayed.",
		"Database names are matched exactly and case-sensitively against backup history.",
		"include the double quotes in the flag value",
		"The --database value is used without transformation.",
		"The --detail option can be used with --timestamp to display object filtering details in this mode.",
	} {
		if !strings.Contains(backupInfoCmd.Long, text) {
			t.Errorf("Command help does not include %q:\n%s", text, backupInfoCmd.Long)
		}
	}
	if !strings.Contains(backupInfoCmd.Long, "--database, --type, --table, --schema, --exclude, --failed, --deleted") {
		t.Errorf("Command help does not preserve timestamp incompatibilities:\n%s", backupInfoCmd.Long)
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

func TestBackupInfoFlagCompatibility(t *testing.T) {
	testhelper.SetupTestLogger()

	tests := []struct {
		name         string
		setDatabase  bool
		setTimestamp bool
		setDetail    bool
		wantExit     bool
	}{
		{name: "database with detail", setDatabase: true, setDetail: true},
		{name: "timestamp with detail", setTimestamp: true, setDetail: true},
		{name: "database with timestamp", setDatabase: true, setTimestamp: true, wantExit: true},
		{name: "database with timestamp and detail", setDatabase: true, setTimestamp: true, setDetail: true, wantExit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			flags.Bool(detailFlagName, false, "")
			if tt.setDatabase {
				if err := flags.Set(databaseFlagName, backupInfoDatabase); err != nil {
					t.Fatalf("Failed to set %s: %v", databaseFlagName, err)
				}
			}
			if tt.setTimestamp {
				if err := flags.Set(timestampFlagName, backupInfoTimestamp); err != nil {
					t.Fatalf("Failed to set %s: %v", timestampFlagName, err)
				}
			}
			if tt.setDetail {
				if err := flags.Set(detailFlagName, "true"); err != nil {
					t.Fatalf("Failed to set %s: %v", detailFlagName, err)
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

func TestAddBackupToTableDatabaseFilter(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		database string
		wantRows int
	}{
		{name: "absent filter", database: "demo", wantRows: 1},
		{name: "exact match", filter: "demo", database: "demo", wantRows: 1},
		{name: "case sensitive mismatch", filter: "Demo", database: "demo"},
		{name: "quoted database name", filter: `"Customer's DB"`, database: `"Customer's DB"`, wantRows: 1},
		{name: "unknown database", filter: "unknown", database: "demo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupTable := table.NewWriter()
			addBackupToTable("", "", "", tt.filter, false, false, backupInfoTestConfig("20240101000000", tt.database), backupTable)
			if got := backupTable.Length(); got != tt.wantRows {
				t.Errorf("Rows = %d, want %d", got, tt.wantRows)
			}
		})
	}
}

func TestAddBackupToTableDatabaseFilterCombinesWithOtherFilters(t *testing.T) {
	matchingTable := backupInfoTestConfig("20240101000000", "demo")
	matchingTable.IncludeTableFiltered = true
	matchingTable.IncludeRelations = []string{"public.orders"}

	matchingSchema := backupInfoTestConfig("20240101000001", "demo")
	matchingSchema.IncludeSchemaFiltered = true
	matchingSchema.IncludeSchemas = []string{"public"}

	wrongType := backupInfoTestConfig("20240101000002", "demo")
	wrongType.Incremental = true
	wrongObject := backupInfoTestConfig("20240101000003", "demo")
	wrongObject.IncludeTableFiltered = true
	wrongObject.IncludeRelations = []string{"public.customers"}
	otherDatabase := backupInfoTestConfig("20240101000004", "other")
	otherDatabase.IncludeTableFiltered = true
	otherDatabase.IncludeRelations = []string{"public.orders"}

	tests := []struct {
		name           string
		databaseFilter string
		typeFilter     string
		tableFilter    string
		schemaFilter   string
		includeDetail  bool
		wantRows       int
	}{
		{name: "type filter", typeFilter: "full", wantRows: 4},
		{name: "table filter", tableFilter: "public.orders", wantRows: 2},
		{name: "schema filter", schemaFilter: "public", wantRows: 1},
		{name: "database and type filters", databaseFilter: "demo", typeFilter: "full", wantRows: 3},
		{name: "database and table filters", databaseFilter: "demo", tableFilter: "public.orders", wantRows: 1},
		{name: "database and schema filters", databaseFilter: "demo", schemaFilter: "public", wantRows: 1},
		{name: "detail does not change rows", databaseFilter: "demo", typeFilter: "full", includeDetail: true, wantRows: 3},
	}

	configs := []gpbckpconfig.BackupConfig{matchingTable, matchingSchema, wrongType, wrongObject, otherDatabase}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupTable := table.NewWriter()
			for _, backupData := range configs {
				addBackupToTable(tt.typeFilter, tt.tableFilter, tt.schemaFilter, tt.databaseFilter, false, tt.includeDetail, backupData, backupTable)
			}
			if got := backupTable.Length(); got != tt.wantRows {
				t.Errorf("Rows = %d, want %d", got, tt.wantRows)
			}
		})
	}

	withoutDetails := table.NewWriter()
	withDetails := table.NewWriter()
	for _, backupData := range configs {
		addBackupToTable("full", "", "", "demo", false, false, backupData, withoutDetails)
		addBackupToTable("full", "", "", "demo", false, true, backupData, withDetails)
	}
	if withoutDetails.Length() != withDetails.Length() {
		t.Errorf("Rows with details = %d, without details = %d", withDetails.Length(), withoutDetails.Length())
	}
}

func TestBackupInfoDBDatabaseFilter(t *testing.T) {
	historyDB := createBackupInfoTestDB(t,
		backupInfoHistoryConfig("20240101000000", "demo"),
		backupInfoHistoryConfig("20240101000001", "other"),
	)

	tests := []struct {
		name     string
		filter   string
		wantRows int
	}{
		{name: "absent filter", wantRows: 2},
		{name: "exact filter", filter: "demo", wantRows: 1},
		{name: "unknown filter", filter: "unknown", wantRows: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupTable := table.NewWriter()
			err := backupInfoDB(BackupInfoOptions{DatabaseFilter: tt.filter}, historyDB, backupTable)
			if err != nil {
				t.Fatalf("backupInfoDB() error = %v", err)
			}
			if got := backupTable.Length(); got != tt.wantRows {
				t.Errorf("Rows = %d, want %d", got, tt.wantRows)
			}
		})
	}
}

func TestBackupInfoDBTimestamp(t *testing.T) {
	baseTimestamp := "20240101000000"
	historyDB := createBackupInfoTestDB(t,
		backupInfoHistoryConfig(baseTimestamp, "demo"),
		backupInfoHistoryConfigWithDependency("20240101000001", "demo", baseTimestamp),
		backupInfoHistoryConfigWithDependency("20240101000002", "other", baseTimestamp),
		backupInfoHistoryConfigWithDependency("20240101000003", `"Customer's DB"`, baseTimestamp),
	)

	tests := []struct {
		name          string
		includeDetail bool
		wantRows      int
	}{
		{name: "base and dependencies", wantRows: 4},
		{name: "detail keeps all rows", includeDetail: true, wantRows: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupTable := table.NewWriter()
			err := backupInfoDB(BackupInfoOptions{
				Timestamp:   baseTimestamp,
				ShowDetails: tt.includeDetail,
			}, historyDB, backupTable)
			if err != nil {
				t.Fatalf("backupInfoDB() error = %v", err)
			}
			if got := backupTable.Length(); got != tt.wantRows {
				t.Errorf("Rows = %d, want %d", got, tt.wantRows)
			}
		})
	}
}

func TestBackupInfoDBTimestampPreservesMissingTimestampError(t *testing.T) {
	historyDB := createBackupInfoTestDB(t, backupInfoHistoryConfig("20240101000000", "demo"))
	err := backupInfoDB(BackupInfoOptions{
		Timestamp: "20240101999999",
	}, historyDB, table.NewWriter())
	if err == nil {
		t.Fatal("backupInfoDB() error = nil, want missing timestamp error")
	}
	if !strings.Contains(err.Error(), "timestamp doesn't match any existing backups") {
		t.Errorf("backupInfoDB() error = %v, want missing timestamp error", err)
	}
}

func backupInfoTestConfig(timestamp, databaseName string) gpbckpconfig.BackupConfig {
	return gpbckpconfig.BackupConfig{
		Timestamp:    timestamp,
		EndTime:      timestamp,
		DatabaseName: databaseName,
		Status:       gpbckpconfig.BackupStatusSuccess,
	}
}

func backupInfoHistoryConfig(timestamp, databaseName string) history.BackupConfig {
	return history.BackupConfig{
		Timestamp:    timestamp,
		EndTime:      timestamp,
		DatabaseName: databaseName,
		Status:       history.BackupStatusSucceed,
	}
}

func backupInfoHistoryConfigWithDependency(timestamp, databaseName, baseTimestamp string) history.BackupConfig {
	config := backupInfoHistoryConfig(timestamp, databaseName)
	config.RestorePlan = []history.RestorePlanEntry{{Timestamp: baseTimestamp}}
	return config
}

func createBackupInfoTestDB(t *testing.T, configs ...history.BackupConfig) *sql.DB {
	t.Helper()
	historyDB, err := history.InitializeHistoryDatabase(filepath.Join(t.TempDir(), historyDBNameConst))
	if err != nil {
		t.Fatalf("Failed to initialize history database: %v", err)
	}
	t.Cleanup(func() {
		if err := historyDB.Close(); err != nil {
			t.Errorf("Failed to close history database: %v", err)
		}
	})
	for i := range configs {
		if err := history.StoreBackupHistory(historyDB, &configs[i]); err != nil {
			t.Fatalf("Failed to store backup %s: %v", configs[i].Timestamp, err)
		}
	}
	return historyDB
}
