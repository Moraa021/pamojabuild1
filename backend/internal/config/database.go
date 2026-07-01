package config

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func NewDatabase(databaseURL string) (*sql.DB, error) {
	driver := "postgres"
	if len(databaseURL) > 7 && databaseURL[:7] == "sqlite:" {
		driver = "sqlite3"
	}

	db, err := sql.Open(driver, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connection established (%s)", driver)
	return db, nil
}
