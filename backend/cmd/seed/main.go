package main

import (
	"context"
	"log"

	"pamojabuild1/backend/db"
	"pamojabuild1/backend/internal/config"
)

func main() {
	cfg := config.Load()
	database, err := config.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	if err := db.Seed(context.Background(), database); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
}
