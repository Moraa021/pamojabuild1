package http

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/trustee"
)

type stubTrusteeService struct {
	actorID      int64
	nomineeID    int64
	registration *trustee.KeyRegistration
	roster       []trustee.Assignment
}

func (s *stubTrusteeService) Nominate(_ context.Context, slug string, actorID int64, index int32, userID int64, message string) (*trustee.Assignment, error) {
	s.actorID = actorID
	s.nomineeID = userID
	return &trustee.Assignment{TaskSlug: slug, TrusteeIndex: index, UserID: userID, DisplayName: "Nominee", Status: trustee.StatusInvited, NominationMessage: message, InvitedAt: time.Now().UTC()}, nil
}
func (s *stubTrusteeService) Accept(context.Context, string, int64) (*trustee.Assignment, error) {
	return nil, nil
}
func (s *stubTrusteeService) RegisterKeys(_ context.Context, slug string, actorID int64, r *trustee.KeyRegistration) (*trustee.Assignment, error) {
	s.actorID = actorID
	s.registration = r
	return &trustee.Assignment{TaskSlug: slug, TrusteeIndex: 1, UserID: actorID, Status: trustee.StatusActive, InvitedAt: time.Now().UTC()}, nil
}
func (s *stubTrusteeService) PrepareKeyRotation(context.Context, string, int64) (string, error) {
	return "challenge", nil
}
func (s *stubTrusteeService) RotateKeys(context.Context, string, int64, *trustee.KeyRegistration) error {
	return nil
}
func (s *stubTrusteeService) Replace(context.Context, string, int64, int32, int64, string, string) error {
	return nil
}
func (s *stubTrusteeService) GetTaskTrustees(context.Context, string) ([]trustee.Assignment, error) {
	return s.roster, nil
}

func trusteeHandlerRouter(service trustee.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewTrusteeHandler(service)
	router := gin.New()
	withActor := func(action gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) { c.Set("user_id", int64(42)); action(c) }
	}
	router.POST("/tasks/:slug/trustees/nominations", withActor(handler.NominateTrustee))
	router.POST("/tasks/:slug/trustees/keys", withActor(handler.RegisterTrusteeKeys))
	router.GET("/tasks/:slug/trustees", withActor(handler.ListTrustees))
	return router
}

func TestNominationUsesSessionActorAndExplicitNominee(t *testing.T) {
	service := &stubTrusteeService{}
	router := trusteeHandlerRouter(service)
	request := httptest.NewRequest(nethttp.MethodPost, "/tasks/community/trustees/nominations", strings.NewReader(`{"trustee_index":1,"user_id":99,"message":"local elder"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != nethttp.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if service.actorID != 42 || service.nomineeID != 99 {
		t.Fatalf("actor or nominee was not bound correctly: actor=%d nominee=%d", service.actorID, service.nomineeID)
	}
}

func TestKeyRegistrationRejectsClientSelectedActor(t *testing.T) {
	service := &stubTrusteeService{}
	router := trusteeHandlerRouter(service)
	request := httptest.NewRequest(nethttp.MethodPost, "/tasks/community/trustees/keys", strings.NewReader(`{"user_id":99,"xpub":"x","web_crypto_pubkey_hex":"04aa","xpub_proof_signature_hex":"aa","web_crypto_proof_signature_hex":"bb","proof_challenge":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
	var body apihttp.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Fields["user_id"] != "is not allowed" {
		t.Fatalf("unexpected response: %#v", body)
	}
}

func TestRosterDoesNotExposeTrusteeKeyMaterial(t *testing.T) {
	service := &stubTrusteeService{roster: []trustee.Assignment{{TaskSlug: "community", TrusteeIndex: 0, UserID: 7, DisplayName: "Amina", Status: trustee.StatusActive, InvitedAt: time.Now().UTC()}}}
	router := trusteeHandlerRouter(service)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(nethttp.MethodGet, "/tasks/community/trustees", nil))
	if response.Code != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, secretField := range []string{"xpub", "web_crypto_pubkey_hex", "proof_signature"} {
		if strings.Contains(body, secretField) {
			t.Fatalf("roster leaked %q: %s", secretField, body)
		}
	}
}
