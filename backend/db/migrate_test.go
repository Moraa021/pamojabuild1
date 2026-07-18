package db

import (
	"errors"
	"testing"

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
