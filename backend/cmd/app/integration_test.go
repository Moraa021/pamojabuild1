package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"pamojabuild1/backend/internal/config"
	"pamojabuild1/backend/internal/testsupport"
)

type authResponse struct {
	Token  string `json:"token"`
	UserID int64  `json:"user_id"`
}

type taskResponse struct {
	Slug string `json:"slug"`
}

type verifyResponse struct {
	TaskSlug string `json:"task_slug"`
	Valid    bool   `json:"valid"`
}

func newTestRouter(t *testing.T) (*gin.Engine, *sql.DB) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		ServerPort:   "0",
		JWTSecret:    "integration-secret-at-least-32-bytes",
		ServerSecret: "ledger-test-secret",
	}
	database := testsupport.NewPostgresDatabase(t)

	router := NewRouter(database, cfg)
	return router, database
}

func doJSONRequest(t *testing.T, router *gin.Engine, method, url, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func approveApplication(t *testing.T, db *sql.DB, slug string, volunteerID int64) {
	_, err := db.Exec(`UPDATE task_applications SET status = 'approved', reviewed_at = CURRENT_TIMESTAMP WHERE task_slug = $1 AND volunteer_id = $2`, slug, volunteerID)
	if err != nil {
		t.Fatalf("failed to approve application: %v", err)
	}
}

func TestFullFlow(t *testing.T) {
	router, db := newTestRouter(t)

	registerBody := `{"phone_number":"+15550000001","password":"password123","display_name":"Test User"}`
	resp := doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/register", "", registerBody)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 from register, got %d: %s", resp.Code, resp.Body.String())
	}

	var registerResp authResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &registerResp); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}

	signInBody := `{"phone_number":"+15550000001","password":"password123"}`
	resp = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/signin", "", signInBody)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 from signin, got %d: %s", resp.Code, resp.Body.String())
	}

	var signInResp authResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &signInResp); err != nil {
		t.Fatalf("failed to decode signin response: %v", err)
	}
	token := signInResp.Token
	if token == "" {
		t.Fatal("expected signin token")
	}

	taskBody := `{"title":"Integration Task","description":"Complete a full flow test","category":"testing","region":"earth","goal_sats":1000,"max_volunteers":1,"volunteer_mode":"open"}`
	resp = doJSONRequest(t, router, http.MethodPost, "/api/v1/tasks", token, taskBody)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 from create task, got %d: %s", resp.Code, resp.Body.String())
	}

	var taskResp taskResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &taskResp); err != nil {
		t.Fatalf("failed to decode task response: %v", err)
	}
	if taskResp.Slug == "" {
		t.Fatal("expected task slug")
	}

	var storedCreatorID int64
	if err := db.QueryRow(`SELECT creator_id FROM tasks WHERE slug = $1`, taskResp.Slug).Scan(&storedCreatorID); err != nil {
		t.Fatalf("read stored task creator: %v", err)
	}
	if storedCreatorID != signInResp.UserID {
		t.Fatalf("expected authenticated account %d to own task, got %d", signInResp.UserID, storedCreatorID)
	}

	trusteeRegisterBody := `{"phone_number":"+15550000002","password":"password123","display_name":"Test Trustee"}`
	resp = doJSONRequest(t, router, http.MethodPost, "/api/v1/auth/register", "", trusteeRegisterBody)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 from trustee registration, got %d: %s", resp.Code, resp.Body.String())
	}
	var trusteeAuth authResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &trusteeAuth); err != nil {
		t.Fatalf("decode trustee auth response: %v", err)
	}

	trusteeKeysBody := `{"trustee_index":0,"xpub":"integration-xpub","web_crypto_pubkey_hex":"integration-public-key"}`
	resp = doJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/trustees", taskResp.Slug), trusteeAuth.Token, trusteeKeysBody)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 from trustee key registration, got %d: %s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/apply", taskResp.Slug), trusteeAuth.Token, `{"message":"conflicting application"}`)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected trustee/volunteer conflict, got %d: %s", resp.Code, resp.Body.String())
	}

	applyBody := `{"message":"I can help"}`
	resp = doJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/apply", taskResp.Slug), token, applyBody)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 from apply, got %d: %s", resp.Code, resp.Body.String())
	}

	approveApplication(t, db, taskResp.Slug, signInResp.UserID)

	submitBody := `{"description":"Work completed","evidence_urls":["https://example.com/proof"]}`
	resp = doJSONRequest(t, router, http.MethodPost, fmt.Sprintf("/api/v1/tasks/%s/submissions", taskResp.Slug), token, submitBody)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 from submit work, got %d: %s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, fmt.Sprintf("/api/v1/ledger/tasks/%s/verify", taskResp.Slug), token, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 from verify, got %d: %s", resp.Code, resp.Body.String())
	}

	var verifyResp verifyResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("failed to decode verify response: %v", err)
	}
	if !verifyResp.Valid {
		t.Fatalf("expected ledger chain to be valid")
	}

	// Ensure the submitted record exists in database
	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM task_submissions WHERE task_slug = $1 AND volunteer_id = $2`, taskResp.Slug, signInResp.UserID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query submissions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one submission row, got %d", count)
	}
}
