package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"testing"
	"time"

	"pamojabuild1/backend/internal/auth"
	"pamojabuild1/backend/internal/testsupport"
)

func TestAccountAndInitialSessionAreCreatedAtomically(t *testing.T) {
	database := testsupport.NewPostgresDatabase(t)
	repository := NewAuthRepository(database)
	tokenHash := sha256.Sum256([]byte("repository-session-token"))
	user := &auth.User{
		PhoneNumber:  "+254700000101",
		PasswordHash: "test-password-hash",
		DisplayName:  "Session Test",
	}
	session := &auth.Session{
		TokenHash: tokenHash[:],
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := repository.Create(context.Background(), user, session); err != nil {
		t.Fatalf("create account and session: %v", err)
	}

	var profileCount int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM volunteer_profiles WHERE user_id = $1`,
		user.ID,
	).Scan(&profileCount); err != nil {
		t.Fatalf("count account profile: %v", err)
	}
	if profileCount != 1 {
		t.Fatalf("expected one account profile, got %d", profileCount)
	}

	authenticated, err := repository.GetBySessionHash(context.Background(), tokenHash[:])
	if err != nil {
		t.Fatalf("load active session: %v", err)
	}
	if authenticated.ID != user.ID {
		t.Fatalf("expected user %d, got %d", user.ID, authenticated.ID)
	}
}

func TestRevokedSessionCannotAuthenticate(t *testing.T) {
	database := testsupport.NewPostgresDatabase(t)
	repository := NewAuthRepository(database)
	tokenHash := sha256.Sum256([]byte("revoked-repository-session"))
	user := &auth.User{
		PhoneNumber:  "+254700000102",
		PasswordHash: "test-password-hash",
		DisplayName:  "Revocation Test",
	}
	session := &auth.Session{
		TokenHash: tokenHash[:],
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := repository.Create(context.Background(), user, session); err != nil {
		t.Fatalf("create account and session: %v", err)
	}
	if err := repository.RevokeSession(context.Background(), tokenHash[:]); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if _, err := repository.GetBySessionHash(context.Background(), tokenHash[:]); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected revoked session to be unavailable, got %v", err)
	}
}
