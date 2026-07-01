package main

import (
    "log"

    _ "modernc.org/sqlite"
    _ "github.com/lib/pq"

    "pamojabuild1/backend/db"
    "pamojabuild1/backend/internal/config"
)

func main() {
    cfg := config.Load()
    database, err := config.NewDatabase(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("db open: %v", err)
    }
    defer database.Close()

    if err := db.RunMigrations(database, "db/migrations"); err != nil {
        log.Fatalf("migrations failed: %v", err)
    }
    log.Println("migrations applied")
}
