package config

import "testing"

func TestLoadUsesSecureSessionDefaults(t *testing.T) {
	t.Setenv("SESSION_COOKIE_SECURE", "")
	t.Setenv("SESSION_COOKIE_NAME", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	config := Load()

	if !config.SessionCookieSecure {
		t.Fatal("session cookies must be secure by default")
	}
	if config.SessionCookieName != "pamojabuild_session" {
		t.Fatalf("unexpected default cookie name %q", config.SessionCookieName)
	}
	if len(config.CORSAllowedOrigins) != 1 || config.CORSAllowedOrigins[0] != "http://localhost:3000" {
		t.Fatalf("unexpected default allowed origins: %#v", config.CORSAllowedOrigins)
	}
}

func TestLoadAllowsExplicitLocalCookieAndOrigins(t *testing.T) {
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, http://127.0.0.1:3000")

	config := Load()

	if config.SessionCookieSecure {
		t.Fatal("expected explicit local development cookie override")
	}
	if len(config.CORSAllowedOrigins) != 2 {
		t.Fatalf("expected two allowed origins, got %#v", config.CORSAllowedOrigins)
	}
}
