package gpbckpconfig

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestGetPrimaryCoordinatorDataDirLocalClusterConn(t *testing.T) {
	db, mock := createClusterMockDB(t)
	defer db.Close()

	query := "SELECT datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'p';"
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WillReturnRows(sqlmock.NewRows([]string{"datadir"}).AddRow("/primary/datadir"))

	got, err := GetPrimaryCoordinatorDataDirLocalClusterConn(db)
	if err != nil {
		t.Fatalf("Expected primary coordinator datadir lookup to succeed, got: %v", err)
	}
	if got != "/primary/datadir" {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got, "/primary/datadir")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Unmet SQL expectations: %v", err)
	}
}

func TestGetDefaultLocalClusterDBName(t *testing.T) {
	tests := []struct {
		name       string
		pgDatabase string
		want       string
	}{
		{
			name: "Default",
			want: "postgres",
		},
		{
			name:       "PGDATABASE",
			pgDatabase: "appdb",
			want:       "appdb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PGDATABASE", tt.pgDatabase)
			if got := GetDefaultLocalClusterDBName(); got != tt.want {
				t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestGetPrimaryCoordinatorDataDirLocalClusterConnErrors(t *testing.T) {
	tests := []struct {
		name      string
		queryErr  error
		wantError error
	}{
		{
			name:      "No Rows",
			queryErr:  sql.ErrNoRows,
			wantError: sql.ErrNoRows,
		},
		{
			name:      "Query Error",
			queryErr:  errors.New("query failed"),
			wantError: errors.New("query failed"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := createClusterMockDB(t)
			defer db.Close()

			query := "SELECT datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'p';"
			mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(tt.queryErr)

			_, err := GetPrimaryCoordinatorDataDirLocalClusterConn(db)
			if err == nil {
				t.Fatalf("Expected primary coordinator datadir lookup to fail")
			}
			if !errors.Is(err, tt.wantError) && err.Error() != tt.wantError.Error() {
				t.Fatalf("Expected %v, got: %v", tt.wantError, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("Unmet SQL expectations: %v", err)
			}
		})
	}
}

func TestGetUpStandbyCoordinatorLocalClusterConn(t *testing.T) {
	db, mock := createClusterMockDB(t)
	defer db.Close()

	query := "SELECT hostname, datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'm' AND status = 'u';"
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WillReturnRows(sqlmock.NewRows([]string{"hostname", "datadir"}).AddRow("sdw-standby", "/standby/datadir"))

	got, err := GetUpStandbyCoordinatorLocalClusterConn(db)
	if err != nil {
		t.Fatalf("Expected standby coordinator lookup to succeed, got: %v", err)
	}
	if got.Hostname != "sdw-standby" {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got.Hostname, "sdw-standby")
	}
	if got.DataDir != "/standby/datadir" {
		t.Fatalf("\nVariables do not match:\n%v\nwant:\n%v", got.DataDir, "/standby/datadir")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Unmet SQL expectations: %v", err)
	}
}

func TestGetUpStandbyCoordinatorLocalClusterConnNoRows(t *testing.T) {
	db, mock := createClusterMockDB(t)
	defer db.Close()

	query := "SELECT hostname, datadir FROM gp_segment_configuration WHERE content = -1 AND role = 'm' AND status = 'u';"
	mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(sql.ErrNoRows)

	_, err := GetUpStandbyCoordinatorLocalClusterConn(db)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Expected sql.ErrNoRows, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Unmet SQL expectations: %v", err)
	}
}

func createClusterMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock DB: %v", err)
	}
	return sqlx.NewDb(rawDB, "sqlmock"), mock
}
