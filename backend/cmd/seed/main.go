package main

import (
    "log"

    "pamojabuild1/backend/db"
    "pamojabuild1/backend/internal/config"
)

func main() {
    cfg := config.Load()
    d, err := config.NewDatabase(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("failed to open db: %v", err)
    }
    defer d.Close()

    if err := db.Seed(d); err != nil {
        log.Fatalf("seed failed: %v", err)
    }
}
