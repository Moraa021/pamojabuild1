package db

import (
    "database/sql"
    "log"
    "time"
)

// Seed populates the database with initial, idempotent seed data.
func Seed(database *sql.DB) error {
    _, err := database.Exec(`INSERT OR IGNORE INTO users (id, email, password_hash, display_name, role, created_at, updated_at) VALUES (1, 'alice@example.com', 'hashed', 'Alice', 'volunteer', datetime('now'), datetime('now'))`)
    if err != nil {
        log.Printf("seed user insert failed: %v", err)
        return err
    }

    _, err = database.Exec(`INSERT OR IGNORE INTO tasks (id, slug, creator_id, title, description, created_at) VALUES (1, 'sample-task', 1, 'Sample Task', 'A seeded task', datetime('now'))`)
    if err != nil {
        log.Printf("seed task insert failed: %v", err)
        return err
    }

    _, err = database.Exec(`INSERT OR IGNORE INTO volunteer_profiles (user_id, bio, skills, created_at, updated_at) VALUES (1, 'Seeded user', '[]', datetime('now'), datetime('now'))`)
    if err != nil {
        log.Printf("seed profile insert failed: %v", err)
        return err
    }

    log.Printf("Seeding complete at %s", time.Now().Format(time.RFC3339))
    return nil
}
