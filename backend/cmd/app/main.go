package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"pamojabuild1/backend/internal/config"
)

func runMigrations(db *sql.DB, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var sqlFiles []string
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)
	for _, f := range sqlFiles {
		content, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		log.Printf("Running migration: %s", f)
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("migration %s failed: %w", f, err)
		}
	}
	log.Println("Migrations complete")
	return nil
}

func main() {
	cfg := config.Load()

	db, err := config.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := runMigrations(db, "db/migrations"); err != nil {
		log.Printf("Warning: migration error: %v", err)
	}

	router := NewRouter(db, cfg)




	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
