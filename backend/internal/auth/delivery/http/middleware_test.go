package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/auth"
)

type middlewareAuthService struct {
	user          *auth.User
	receivedToken string
}

func (s *middlewareAuthService) Register(context.Context, string, string, string) (*auth.Session, error) {
	panic("not used")
}

func (s *middlewareAuthService) SignIn(context.Context, string, string) (*auth.Session, error) {
	panic("not used")
}

func (s *middlewareAuthService) SignOut(context.Context, string) error {
	panic("not used")
}

func (s *middlewareAuthService) Authenticate(_ context.Context, token string) (*auth.User, error) {
	s.receivedToken = token
	return s.user, nil
}

func TestAuthMiddlewareUsesSessionCookieAndSetsActorContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &middlewareAuthService{user: &auth.User{ID: 42}}
	router := gin.New()
	router.Use(AuthMiddleware(service, "pamojabuild_session"))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(nethttp.StatusOK, gin.H{"user_id": c.GetInt64("user_id")})
	})
	request := httptest.NewRequest(nethttp.MethodGet, "/protected", nil)
	request.AddCookie(&nethttp.Cookie{Name: "pamojabuild_session", Value: "opaque-token"})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusOK {
		t.Fatalf("expected %d, got %d", nethttp.StatusOK, response.Code)
	}
	if service.receivedToken != "opaque-token" {
		t.Fatalf("expected cookie token to be authenticated, got %q", service.receivedToken)
	}
}

func TestAuthMiddlewareRejectsMissingSessionCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware(&middlewareAuthService{}, "pamojabuild_session"))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(nethttp.StatusNoContent)
	})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(nethttp.MethodGet, "/protected", nil))

	if response.Code != nethttp.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", nethttp.StatusUnauthorized, response.Code)
	}
}
