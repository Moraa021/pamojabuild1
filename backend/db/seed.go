package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Seed adds local development data without assuming fixed sequence values.
// Explicit IDs are avoided because they can make the next PostgreSQL insert
// collide with a sequence that was not advanced by the seed operation.
func Seed(ctx context.Context, database *sql.DB) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, display_name, role)
		VALUES ('alice@example.com', 'hashed', 'Alice', 'volunteer')
		ON CONFLICT (email) DO NOTHING`); err != nil {
		return fmt.Errorf("seed user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tasks (slug, creator_id, title, description)
		SELECT 'sample-task', id, 'Sample Task', 'A seeded task'
		FROM users
		WHERE email = 'alice@example.com'
		ON CONFLICT (slug) DO NOTHING`); err != nil {
		return fmt.Errorf("seed task: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO volunteer_profiles (user_id, bio, skills)
		SELECT id, 'Seeded user', '[]'::jsonb
		FROM users
		WHERE email = 'alice@example.com'
		ON CONFLICT (user_id) DO NOTHING`); err != nil {
		return fmt.Errorf("seed volunteer profile: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}
