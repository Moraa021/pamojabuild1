package config

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func NewDatabase(databaseURL string) (*sql.DB, error) {
	driver := "postgres"
	dsn := databaseURL

	if len(databaseURL) < 7 || databaseURL[:7] != "postgre" {
		driver = "sqlite"
		dsn = databaseURL + "?_journal_mode=WAL&_foreign_keys=on"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connection established (%s)", driver)
	return db, nil
}
