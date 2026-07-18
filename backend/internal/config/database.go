package config

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewDatabase(databaseURL string) (*sql.DB, error) {
	if err := validatePostgresURL(databaseURL); err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// PostgreSQL coordinates concurrency and transactional row locks. A real
	// pool is required once ledger and payout workers run alongside HTTP calls.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Print("Database connection established (postgres)")
	return db, nil
}

func validatePostgresURL(databaseURL string) error {
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL must use postgres:// or postgresql://, got %q", parsed.Scheme)
	}
	if parsed.Host == "" || parsed.Path == "" || parsed.Path == "/" {
		return errors.New("DATABASE_URL must include a PostgreSQL host and database name")
	}
	return nil
}
