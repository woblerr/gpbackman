package gpbckpconfig

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/greenplum-db/gp-common-go-libs/operating"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// NewClusterLocalClusterConn creates a new connection to the local postgres database
// Returns an error if the connection could not be established.
func NewClusterLocalClusterConn(dbName string) (*sqlx.DB, error) {
	if dbName == "" {
		return nil, errors.New("Database name cannot be empty")
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

// ExecuteQueryLocalClusterConn executes a query  and returns result.
func ExecuteQueryLocalClusterConn(conn *sqlx.DB, query string) (string, error) {
	var result string
	err := conn.Get(&result, query)
	return result, err
}
