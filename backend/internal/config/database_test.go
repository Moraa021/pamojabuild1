package config

import (
	"strings"
	"testing"
)

func TestValidatePostgresURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		databaseURL string
		wantError   string
	}{
		{name: "postgres", databaseURL: "postgres://user:pass@localhost:5432/pamoja"},
		{name: "postgresql", databaseURL: "postgresql://user:pass@localhost:5432/pamoja"},
		{name: "missing", wantError: "DATABASE_URL is required"},
		{name: "sqlite path", databaseURL: "/tmp/pamoja.db", wantError: "must use postgres"},
		{name: "sqlite URL", databaseURL: "sqlite:///tmp/pamoja.db", wantError: "must use postgres"},
		{name: "missing host", databaseURL: "postgres:///pamoja", wantError: "host and database name"},
		{name: "missing database", databaseURL: "postgres://localhost", wantError: "host and database name"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validatePostgresURL(test.databaseURL)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("expected valid PostgreSQL URL, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected error containing %q, got %v", test.wantError, err)
			}
		})
	}
}
