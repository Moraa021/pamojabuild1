package db

import (
	"errors"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
)

func TestIgnoreNoChange(t *testing.T) {
	t.Parallel()

	if err := IgnoreNoChange(migrate.ErrNoChange); err != nil {
		t.Fatalf("expected ErrNoChange to be ignored, got %v", err)
	}

	expected := errors.New("migration failed")
	if err := IgnoreNoChange(expected); !errors.Is(err, expected) {
		t.Fatalf("expected real migration error to be preserved, got %v", err)
	}
}

func TestMigrationFilesUsePairedUTCTimestamps(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	pattern := regexp.MustCompile(`^([0-9]{14})_([a-z0-9_]+)\.(up|down)\.sql$`)
	pairs := make(map[string]map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		match := pattern.FindStringSubmatch(entry.Name())
		if match == nil {
			t.Fatalf("migration %q must use YYYYMMDDHHMMSS_name.(up|down).sql", entry.Name())
		}
		if _, err := time.Parse("20060102150405", match[1]); err != nil {
			t.Fatalf("migration %q has an invalid UTC timestamp: %v", entry.Name(), err)
		}

		key := match[1] + "_" + match[2]
		if pairs[key] == nil {
			pairs[key] = make(map[string]bool)
		}
		pairs[key][match[3]] = true
	}

	if len(pairs) == 0 {
		t.Fatal("expected at least one migration pair")
	}
	for migration, directions := range pairs {
		if !directions["up"] || !directions["down"] {
			t.Fatalf("migration %q must include both up and down files", migration)
		}
	}
}
