package authorization

import (
	"context"
	"database/sql"
	"testing"

	"pamojabuild1/backend/internal/testsupport"
)

func TestTaskRelationshipQueries(t *testing.T) {
	database := testsupport.NewPostgresDatabase(t)
	repository := NewRepository(database)
	creatorID := insertRelationshipUser(t, database, "+254700002001")
	volunteerID := insertRelationshipUser(t, database, "+254700002002")
	trusteeID := insertRelationshipUser(t, database, "+254700002003")

	if _, err := database.Exec(`
		INSERT INTO tasks (
			slug, creator_id, title, status, financial_state,
			work_state_version, financial_state_version
		)
		VALUES ('relationship-task', $1, 'Relationship Task', 'open', 'ACTIVE', 1, 1)`,
		creatorID,
	); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO task_applications (task_slug, volunteer_id, status)
		VALUES ('relationship-task', $1, 'approved')`,
		volunteerID,
	); err != nil {
		t.Fatalf("insert approved application: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO trustee_keys (
			task_slug, trustee_index, user_id, xpub, web_crypto_pubkey_hex
		)
		VALUES ('relationship-task', 0, $1, 'test-xpub', 'test-browser-key')`,
		trusteeID,
	); err != nil {
		t.Fatalf("insert trustee: %v", err)
	}

	hasApproved, err := repository.HasApprovedVolunteer(context.Background(), "relationship-task")
	if err != nil || !hasApproved {
		t.Fatalf("expected approved volunteer, found=%v err=%v", hasApproved, err)
	}
	approved, submitted, err := repository.ApprovedSubmissionCounts(context.Background(), "relationship-task")
	if err != nil || approved != 1 || submitted != 0 {
		t.Fatalf("unexpected initial submission counts: approved=%d submitted=%d err=%v", approved, submitted, err)
	}

	if _, err := database.Exec(`
		INSERT INTO task_submissions (task_slug, volunteer_id, description)
		VALUES ('relationship-task', $1, 'completed work')`,
		volunteerID,
	); err != nil {
		t.Fatalf("insert submission: %v", err)
	}
	approved, submitted, err = repository.ApprovedSubmissionCounts(context.Background(), "relationship-task")
	if err != nil || approved != 1 || submitted != 1 {
		t.Fatalf("unexpected completed submission counts: approved=%d submitted=%d err=%v", approved, submitted, err)
	}

	isVolunteer, err := repository.IsTaskVolunteer(context.Background(), "relationship-task", volunteerID)
	if err != nil || !isVolunteer {
		t.Fatalf("expected task volunteer, found=%v err=%v", isVolunteer, err)
	}
	isTrustee, err := repository.IsTaskTrustee(context.Background(), "relationship-task", trusteeID)
	if err != nil || !isTrustee {
		t.Fatalf("expected task trustee, found=%v err=%v", isTrustee, err)
	}
}

func insertRelationshipUser(t *testing.T, database *sql.DB, phone string) int64 {
	t.Helper()
	var userID int64
	if err := database.QueryRow(`
		INSERT INTO users (phone_number, password_hash, display_name)
		VALUES ($1, 'test-password-hash', 'Relationship Test')
		RETURNING id`,
		phone,
	).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return userID
}
