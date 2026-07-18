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

	"pamojabuild1/backend/internal/auth"
)

type stubAuthService struct {
	session *auth.Session
}

func (s stubAuthService) Register(context.Context, string, string, string) (*auth.Session, error) {
	return s.session, nil
}

func (s stubAuthService) SignIn(context.Context, string, string) (*auth.Session, error) {
	return s.session, nil
}

func (s stubAuthService) SignOut(context.Context, string) error {
	return nil
}

func (s stubAuthService) Authenticate(context.Context, string) (*auth.User, error) {
	return s.session.User, nil
}

func TestRegisterSetsSecureHttpOnlySessionCookieWithoutReturningToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	session := &auth.Session{
		User: &auth.User{
			ID:          42,
			DisplayName: "Amina",
		},
		Token:     "opaque-session-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	handler := NewAuthHandler(stubAuthService{session: session}, "pamojabuild_session", true)
	router := gin.New()
	router.POST("/register", handler.Register)
	request := httptest.NewRequest(
		nethttp.MethodPost,
		"/register",
		strings.NewReader(`{"phone_number":"+254700000000","password":"password123","display_name":"Amina"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusCreated {
		t.Fatalf("expected %d, got %d: %s", nethttp.StatusCreated, response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "pamojabuild_session" || cookie.Value != session.Token {
		t.Fatalf("unexpected session cookie: %#v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != nethttp.SameSiteLaxMode {
		t.Fatalf("expected HttpOnly, Secure, SameSite=Lax cookie: %#v", cookie)
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	if _, exposed := body["token"]; exposed {
		t.Fatal("opaque session token must not be exposed to JavaScript")
	}
}
