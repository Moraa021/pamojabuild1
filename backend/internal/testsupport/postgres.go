package testsupport

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"pamojabuild1/backend/db"
	"pamojabuild1/backend/internal/config"
)

// NewPostgresDatabase creates an isolated schema in the PostgreSQL instance
// named by TEST_DATABASE_URL and applies the real migration set. Repository
// tests must exercise PostgreSQL semantics instead of a permissive substitute.
func NewPostgresDatabase(t *testing.T) *sql.DB {
	t.Helper()

	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}

	adminDB, err := config.NewDatabase(baseURL)
	if err != nil {
		t.Fatalf("connect to TEST_DATABASE_URL: %v", err)
	}

	schema := "test_" + randomHex(t, 8)
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + quoteIdentifier(schema)); err != nil {
		adminDB.Close()
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}

	schemaURL := withSearchPath(t, baseURL, schema)
	runner, err := db.NewMigrator(schemaURL, migrationsPath(t))
	if err != nil {
		dropTestSchema(t, adminDB, schema)
		adminDB.Close()
		t.Fatalf("create test migration runner: %v", err)
	}
	if err := db.IgnoreNoChange(runner.Up()); err != nil {
		runner.Close()
		dropTestSchema(t, adminDB, schema)
		adminDB.Close()
		t.Fatalf("apply PostgreSQL test migrations: %v", err)
	}
	if sourceErr, databaseErr := runner.Close(); sourceErr != nil || databaseErr != nil {
		dropTestSchema(t, adminDB, schema)
		adminDB.Close()
		t.Fatalf("close test migration runner: source=%v database=%v", sourceErr, databaseErr)
	}

	testDB, err := config.NewDatabase(schemaURL)
	if err != nil {
		dropTestSchema(t, adminDB, schema)
		adminDB.Close()
		t.Fatalf("connect to isolated PostgreSQL schema: %v", err)
	}

	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Errorf("close PostgreSQL test database: %v", err)
		}
		dropTestSchema(t, adminDB, schema)
		if err := adminDB.Close(); err != nil {
			t.Errorf("close PostgreSQL test admin connection: %v", err)
		}
	})

	return testDB
}

func migrationsPath(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve PostgreSQL test helper path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "..", "db", "migrations")
}

func randomHex(t *testing.T, byteCount int) string {
	t.Helper()

	value := make([]byte, byteCount)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("generate test schema suffix: %v", err)
	}
	return hex.EncodeToString(value)
}

func withSearchPath(t *testing.T, databaseURL, schema string) string {
	t.Helper()

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func quoteIdentifier(identifier string) string {
	// Identifiers are generated internally from a fixed prefix and hex bytes.
	// Quoting remains deliberate so a future format change cannot turn cleanup
	// into executable SQL.
	return `"` + identifier + `"`
}

func dropTestSchema(t *testing.T, adminDB *sql.DB, schema string) {
	t.Helper()

	if _, err := adminDB.Exec(`DROP SCHEMA IF EXISTS ` + quoteIdentifier(schema) + ` CASCADE`); err != nil {
		t.Errorf("drop PostgreSQL test schema %s: %v", schema, err)
	}
}
