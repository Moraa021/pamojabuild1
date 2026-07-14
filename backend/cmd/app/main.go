package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := config.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := runMigrations(db, "db/migrations"); err != nil {
		log.Printf("Warning: migration error: %v", err)
	}

	router := NewRouterWithContext(ctx, db, cfg)
	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	log.Printf("Server starting on port %s", cfg.ServerPort)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Failed to start server:", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatal("Failed to stop server cleanly:", err)
		}
	}
}
