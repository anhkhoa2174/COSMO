// Package pgtest gives each test its own empty PostgreSQL schema.
//
// The repository and service tests used to run against in-memory SQLite. That
// exercised the SQL GORM generates, but not the database the product runs on:
// JSONB operators, GIN indexes, array columns, uuid and timestamptz semantics,
// and PostgreSQL's stricter typing behave differently under SQLite or not at
// all, so a test could pass on behaviour production never has.
//
// Every call creates a schema with a random name inside the test database,
// points the connection's search_path at it, and drops it when the test ends.
// Tests share one server but never one table, so packages that run in parallel
// cannot see each other's rows.
package pgtest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// EnvKey names the variable holding the test database URL. `make test` derives
// it from the development database URL by swapping in a dedicated database, so
// a test run never touches development data.
const EnvKey = "TEST_DATABASE_URL"

// Open returns a connection to a fresh schema in the test database. Its
// signature mirrors gorm.Open so that call sites keep their error handling.
//
// A missing TEST_DATABASE_URL fails the test rather than skipping it: a skipped
// suite reports success while having checked nothing, which is the one outcome
// worse than a failure.
func Open(t testing.TB, cfg *gorm.Config) (*gorm.DB, error) {
	t.Helper()

	base := os.Getenv(EnvKey)
	if base == "" {
		t.Fatalf("%s is not set. These tests run against PostgreSQL rather than an "+
			"in-memory substitute: start the development database and run `make test`, "+
			"or export %s=postgres://USER:PASS@HOST:PORT/cosmo_agents_test?sslmode=disable",
			EnvKey, EnvKey)
	}
	if cfg == nil {
		cfg = &gorm.Config{}
	}
	if cfg.Logger == nil {
		cfg.Logger = logger.Discard
	}

	admin, err := gorm.Open(postgres.Open(base), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		return nil, fmt.Errorf("pgtest: connect to %s: %w", EnvKey, err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		return nil, err
	}

	schema, err := randomSchema()
	if err != nil {
		_ = adminSQL.Close()
		return nil, err
	}
	if err := admin.Exec(`CREATE SCHEMA ` + schema).Error; err != nil {
		_ = adminSQL.Close()
		return nil, fmt.Errorf("pgtest: create schema: %w", err)
	}
	drop := func() {
		_ = admin.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`).Error
		_ = adminSQL.Close()
	}

	dsn, err := withSearchPath(base, schema)
	if err != nil {
		drop()
		return nil, err
	}
	db, err := gorm.Open(postgres.Open(dsn), cfg)
	if err != nil {
		drop()
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		drop()
		return nil, err
	}
	// A handful of connections is plenty for a unit of test code, and keeping the
	// pool small stops a parallel `go test ./...` from exhausting the server.
	sqlDB.SetMaxOpenConns(4)

	t.Cleanup(func() {
		_ = sqlDB.Close()
		drop()
	})
	return db, nil
}

// randomSchema returns an identifier that is safe to interpolate: it is built
// only from a fixed prefix and hex digits.
func randomSchema() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "t_" + hex.EncodeToString(b), nil
}

// withSearchPath points the connection at the test schema first and public
// second, so extension functions installed in public stay reachable.
func withSearchPath(base, schema string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("pgtest: %s must be a URL: %w", EnvKey, err)
	}
	q := u.Query()
	q.Set("search_path", schema+",public")
	u.RawQuery = q.Encode()
	return u.String(), nil
}
