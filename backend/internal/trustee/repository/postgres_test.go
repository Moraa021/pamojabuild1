package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"

	"pamojabuild1/backend/internal/testsupport"
	"pamojabuild1/backend/internal/trustee"
)

func TestOnboardingPersistsActiveProjectionAndHistory(t *testing.T) {
	database := testsupport.NewPostgresDatabase(t)
	repo := NewTrusteeRepository(database)
	creatorID := insertTrusteeTestUser(t, database, 1)
	trusteeID := insertTrusteeTestUser(t, database, 2)
	insertTrusteeTestTask(t, database, "trustee-onboarding", creatorID)

	assignment := &trustee.Assignment{TaskSlug: "trustee-onboarding", TrusteeIndex: 2, UserID: trusteeID, NominatedBy: creatorID, NominationMessage: "community treasurer"}
	if err := repo.Nominate(context.Background(), assignment); err != nil {
		t.Fatalf("nominate: %v", err)
	}
	accepted, err := repo.Accept(context.Background(), assignment.TaskSlug, trusteeID, "proof-challenge")
	if err != nil || accepted.Status != trustee.StatusAccepted {
		t.Fatalf("accept: assignment=%#v err=%v", accepted, err)
	}
	active, err := repo.Activate(context.Background(), &trustee.KeyRegistration{TaskSlug: assignment.TaskSlug, UserID: trusteeID, Xpub: "xpub-version-one", WebCryptoPubkeyHex: "04abcd", XpubProofSignatureHex: "proof-a", WebCryptoProofSignatureHex: "proof-b", ProofChallenge: "proof-challenge"})
	if err != nil || active.Status != trustee.StatusActive {
		t.Fatalf("activate: assignment=%#v err=%v", active, err)
	}

	key, err := repo.GetSpecificTrustee(context.Background(), assignment.TaskSlug, 2)
	if err != nil || key.UserID != trusteeID {
		t.Fatalf("active projection: key=%#v err=%v", key, err)
	}
	var historyCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM trustee_assignment_history WHERE task_slug=$1`, assignment.TaskSlug).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 3 {
		t.Fatalf("expected nomination, acceptance and activation history, got %d", historyCount)
	}
}

func TestConcurrentNominationsHaveOneWinner(t *testing.T) {
	database := testsupport.NewPostgresDatabase(t)
	repo := NewTrusteeRepository(database)
	creatorID := insertTrusteeTestUser(t, database, 11)
	firstID := insertTrusteeTestUser(t, database, 12)
	secondID := insertTrusteeTestUser(t, database, 13)
	insertTrusteeTestTask(t, database, "trustee-race", creatorID)

	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, userID := range []int64{firstID, secondID} {
		workers.Add(1)
		go func(candidate int64) {
			defer workers.Done()
			<-start
			results <- repo.Nominate(context.Background(), &trustee.Assignment{TaskSlug: "trustee-race", TrusteeIndex: 0, UserID: candidate, NominatedBy: creatorID})
		}(userID)
	}
	close(start)
	workers.Wait()
	close(results)
	var success, conflict int
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, trustee.ErrRegistrationConflict) {
			conflict++
		} else {
			t.Fatalf("unexpected nomination result: %v", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("expected one winner and one conflict, got success=%d conflict=%d", success, conflict)
	}
}

func TestReplacementRevokesOldAuthorizationAtomically(t *testing.T) {
	database := testsupport.NewPostgresDatabase(t)
	repo := NewTrusteeRepository(database)
	creatorID := insertTrusteeTestUser(t, database, 21)
	oldID := insertTrusteeTestUser(t, database, 22)
	newID := insertTrusteeTestUser(t, database, 23)
	insertTrusteeTestTask(t, database, "trustee-replace", creatorID)
	a := &trustee.Assignment{TaskSlug: "trustee-replace", TrusteeIndex: 1, UserID: oldID, NominatedBy: creatorID}
	if err := repo.Nominate(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Accept(context.Background(), a.TaskSlug, oldID, "challenge"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Activate(context.Background(), &trustee.KeyRegistration{TaskSlug: a.TaskSlug, UserID: oldID, Xpub: "old-xpub", WebCryptoPubkeyHex: "old-web", XpubProofSignatureHex: "a", WebCryptoProofSignatureHex: "b", ProofChallenge: "challenge"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Replace(context.Background(), &trustee.Replacement{TaskSlug: a.TaskSlug, TrusteeIndex: 1, NewUserID: newID, ActorUserID: creatorID, Reason: "trustee unavailable"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetSpecificTrustee(context.Background(), a.TaskSlug, 1); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("old active key must be removed, got %v", err)
	}
	invited, err := repo.GetAssignmentForUser(context.Background(), a.TaskSlug, newID)
	if err != nil || invited.Status != trustee.StatusInvited {
		t.Fatalf("replacement invitation: %#v %v", invited, err)
	}
}

func insertTrusteeTestUser(t *testing.T, database *sql.DB, suffix int) int64 {
	t.Helper()
	var id int64
	phone := fmt.Sprintf("+254711%06d", suffix)
	if err := database.QueryRow(`INSERT INTO users(phone_number,password_hash,display_name) VALUES($1,'hash',$2) RETURNING id`, phone, fmt.Sprintf("User %d", suffix)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
func insertTrusteeTestTask(t *testing.T, database *sql.DB, slug string, creatorID int64) {
	t.Helper()
	_, err := database.Exec(`INSERT INTO tasks(slug,creator_id,title,description,category,region,status,financial_state,goal_sats,max_volunteers,volunteer_mode) VALUES($1,$2,'Trustee test','Description','community','Nairobi','open','ACTIVE',1000,1,'approval_required')`, slug, creatorID)
	if err != nil {
		t.Fatal(err)
	}
}
