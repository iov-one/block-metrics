package store

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/iov-one/block-metrics/utils"
)

// ensureDB connects to a Postgres instance creates a database and returns a
// connection to it. If the connection to Postres cannot be established, the
// test is skipped.
//
// Each database is initialized with the schema.
//
// Unless an option is provided, defaults are used:
//   * Database name: test_database_<creation time in unix ns>
//   * Host: localhost
//   * Port: 5432
//   * SSLMode: disable
//   * User: postgres
//
//
// Function connects to the 'postgres' database first to create a new database.
func EnsureDB(t *testing.T) (testdb *sql.DB, cleanup func()) {
	t.Helper()

	var opts = struct {
		User     string
		Password string
		Port     string
		Host     string
		SSLMode  string
		DBName   string
	}{
		User:     utils.Env("POSTGRES_TEST_USER", "postgres"),
		Password: utils.Env("POSTGRES_TEST_PASSWORD", "postgres"),
		Port:     utils.Env("POSTGRES_TEST_PORT", "5432"),
		Host:     utils.Env("POSTGRES_TEST_HOST", "localhost"),
		SSLMode:  utils.Env("POSTGRES_TEST_SSLMODE", "disable"),
		DBName: utils.Env("POSTGRES_TEST_DATABASE",
			fmt.Sprintf("test_database_%d", time.Now().UnixNano())),
	}

	rootDsn := fmt.Sprintf(
		"host='%s' port='%s' user='%s' password=%s dbname='postgres' sslmode='%s'",
		opts.Host, opts.Port, opts.User, opts.Password, opts.SSLMode)
	rootdb, err := sql.Open("postgres", rootDsn)
	if err != nil {
		t.Skipf("cannot connect to postgres: %s", err)
	}
	if err := rootdb.Ping(); err != nil {
		t.Skipf("cannot ping postgres: %s", err)
	}
	if _, err := rootdb.Exec("CREATE DATABASE " + opts.DBName); err != nil {
		t.Fatalf("cannot create database: %s", err)
		rootdb.Close()
	}

	testDsn := fmt.Sprintf(
		"host='%s' port='%s' user='%s' password='%s' dbname='%s' sslmode='%s'",
		opts.Host, opts.Port, opts.User, opts.Password, opts.DBName, opts.SSLMode)
	testdb, err = sql.Open("postgres", testDsn)
	if err != nil {
		t.Fatalf("cannot connect to created database: %s", err)
	}
	if err := testdb.Ping(); err != nil {
		t.Fatalf("cannot ping test database: %s", err)
	}

	if err := EnsureSchema(testdb); err != nil {
		t.Fatalf("cannot ensure schema: %s", err)
	}

	cleanup = func() {
		testdb.Close()
		if _, err := rootdb.Exec("DROP DATABASE " + opts.DBName); err != nil {
			t.Logf("cannot delete test database %q: %s", opts.DBName, err)
		}
		rootdb.Close()
	}
	return testdb, cleanup
}
