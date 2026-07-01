package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
    "pamojabuild1/backend/internal/config"
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

    tmpFile, err := os.CreateTemp("", "pamoja-integration-*.db")
    if err != nil {
        t.Fatalf("failed to create temp db: %v", err)
    }
    dbPath := tmpFile.Name()
    tmpFile.Close()
    t.Cleanup(func() {
        os.Remove(dbPath)
    })

    cfg := &config.Config{
        ServerPort:   "0",
        DatabaseURL:  dbPath,
        JWTSecret:    "integration-secret",
        ServerSecret: "ledger-test-secret",
    }

    db, err := config.NewDatabase(cfg.DatabaseURL)
    if err != nil {
        t.Fatalf("failed to open database: %v", err)
    }
    t.Cleanup(func() {
        db.Close()
    })

    _, filename, _, ok := runtime.Caller(0)
    if !ok {
        t.Fatalf("failed to resolve test file path")
    }
    migrationsPath := filepath.Join(filepath.Dir(filename), "..", "..", "db", "migrations")

    if err := runMigrations(db, migrationsPath); err != nil {
        t.Fatalf("failed to run migrations: %v", err)
    }

    router := NewRouter(db, cfg)
    return router, db
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
    _, err := db.Exec(`UPDATE task_applications SET status = 'approved', reviewed_at = CURRENT_TIMESTAMP WHERE task_slug = ? AND volunteer_id = ?`, slug, volunteerID)
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

    taskBody := fmt.Sprintf(`{"creator_id":%d,"title":"Integration Task","description":"Complete a full flow test","category":"testing","region":"earth","goal_sats":1000,"max_volunteers":1,"volunteer_mode":"open"}`, signInResp.UserID)
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

    applyBody := fmt.Sprintf(`{"volunteer_id":%d,"message":"I can help"}`, signInResp.UserID)
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
    err := db.QueryRow(`SELECT COUNT(1) FROM task_submissions WHERE task_slug = ? AND volunteer_id = ?`, taskResp.Slug, signInResp.UserID).Scan(&count)
    if err != nil {
        t.Fatalf("failed to query submissions: %v", err)
    }
    if count != 1 {
        t.Fatalf("expected one submission row, got %d", count)
    }
}
