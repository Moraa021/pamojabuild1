package db

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
)

func RunMigrations(db *sql.DB, migrationsPath string) error {
	files, err := ioutil.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	for _, fileName := range migrationFiles {
		filePath := filepath.Join(migrationsPath, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fileName, err)
		}

		log.Printf("Running migration: %s", fileName)
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", fileName, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("rollback failed after migration error: %v", rollbackErr)
			}
			return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", fileName, err)
		}
	}

	log.Println("All migrations completed successfully")
	return nil
}