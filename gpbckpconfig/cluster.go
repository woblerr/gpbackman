package gpbckpconfig

import (
	"fmt"
	"strconv"

	"github.com/greenplum-db/gp-common-go-libs/operating"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/woblerr/gpbackman/textmsg"
)

type SegmentConfig struct {
	ContentID string
	Hostname  string
	DataDir   string
}

type StandbyCoordinatorConfig struct {
	Hostname string
	DataDir  string
}

// NewClusterLocalClusterConn creates a new connection to the local postgres database
// Returns an error if the connection could not be established.
func NewClusterLocalClusterConn(dbName string) (*sqlx.DB, error) {
	if dbName == "" {
		return nil, textmsg.ErrorEmptyDatabase()
	}
	username := operating.System.Getenv("PGUSER")
	if username == "" {
		currentUser, _ := operating.System.CurrentUser()
		username = currentUser.Username
	}
	host := operating.System.Getenv("PGHOST")
	if host == "" {
		host, _ = operating.System.Hostname()
	}
	port, err := strconv.Atoi(operating.System.Getenv("PGPORT"))
	if err != nil {
		port = 5432
	}
	connStr := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=disable&connect_timeout=60", username, host, port, dbName)
	return sqlx.Connect("postgres", connStr)
}

func NewClusterLocalClusterConnWithDefault(dbName string) (*sqlx.DB, error) {
	if dbName == "" {
		dbName = GetDefaultLocalClusterDBName()
	}
	return NewClusterLocalClusterConn(dbName)
}

func GetDefaultLocalClusterDBName() string {
	if dbName := operating.System.Getenv("PGDATABASE"); dbName != "" {
		return dbName
	}
	return "postgres"
}

// ExecuteQueryLocalClusterConn executes a query on the local cluster connection and returns the result.
// The function is generic and can handle different types of results based on the type parameter T.
//
// Parameters:
// - conn: A pointer to the sqlx.DB connection object.
// - query: A string containing the SQL query to be executed.
//
// Returns:
// - T: The result of the query, which can be of any type specified by the caller.
// - error: An error object if the query execution fails or if the type is unsupported.
//
// The function supports the following types for T:
// - string: The result will be a single string value.
// - []SegmentConfig: The result will be a slice of SegmentConfig structs.
// - StandbyCoordinatorConfig: The result will be a single standby coordinator config.
//
// If the type T is not supported, the function returns an error indicating the unsupported type.
func ExecuteQueryLocalClusterConn[T any](conn *sqlx.DB, query string) (T, error) {
	var result T
	switch any(result).(type) {
	case string:
		var data string
		err := conn.Get(&data, query)
		if err != nil {
			return result, err
		}
		result = any(data).(T)
	case []SegmentConfig:
		var segConfigs []SegmentConfig
		err := conn.Select(&segConfigs, query)
		if err != nil {
			return result, err
		}
		result = any(segConfigs).(T)
	case StandbyCoordinatorConfig:
		var standbyConfig StandbyCoordinatorConfig
		err := conn.Get(&standbyConfig, query)
		if err != nil {
			return result, err
		}
		result = any(standbyConfig).(T)
	default:
		return result, fmt.Errorf("unsupported type")
	}
	return result, nil
}

func GetPrimaryCoordinatorDataDirLocalClusterConn(conn *sqlx.DB) (string, error) {
	sqlQuery := "SELECT datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'p';"
	return ExecuteQueryLocalClusterConn[string](conn, sqlQuery)
}

func GetUpStandbyCoordinatorLocalClusterConn(conn *sqlx.DB) (StandbyCoordinatorConfig, error) {
	sqlQuery := "SELECT hostname, datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'm' AND status = 'u';"
	return ExecuteQueryLocalClusterConn[StandbyCoordinatorConfig](conn, sqlQuery)
}
