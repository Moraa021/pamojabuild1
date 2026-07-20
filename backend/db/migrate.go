package db

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// NewMigrator creates the versioned migration runner used by the deployment
// command and PostgreSQL integration tests. Keeping migration execution outside
// API startup prevents one application replica from racing another over DDL.
func NewMigrator(databaseURL, migrationsPath string) (*migrate.Migrate, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if migrationsPath == "" {
		return nil, errors.New("migrations path is required")
	}

	absolutePath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("resolve migrations path: %w", err)
	}

	runner, err := migrate.New("file://"+filepath.ToSlash(absolutePath), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create migration runner: %w", err)
	}
	return runner, nil
}

// IgnoreNoChange turns golang-migrate's normal "already at this version"
// result into success without hiding real migration failures.
func IgnoreNoChange(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}
