package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/config"
)

func contractTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return NewRouterWithLightningNode(nil, &config.Config{
		SessionCookieName: "test_session",
	}, nil)
}

func TestRegisteredAPIContractRoutes(t *testing.T) {
	router := contractTestRouter()
	expected := map[string]bool{
		"GET /health":                              false,
		"POST /api/v1/auth/register":               false,
		"POST /api/v1/auth/signin":                 false,
		"GET /api/v1/auth/me":                      false,
		"POST /api/v1/auth/signout":                false,
		"GET /api/v1/tasks":                        false,
		"POST /api/v1/tasks":                       false,
		"GET /api/v1/tasks/:slug":                  false,
		"POST /api/v1/tasks/:slug/apply":           false,
		"POST /api/v1/tasks/:slug/submissions":     false,
		"POST /api/v1/tasks/:slug/trustees":        false,
		"POST /api/v1/tasks/:slug/donate":          false,
		"GET /api/v1/volunteers/profile":           false,
		"PUT /api/v1/volunteers/profile":           false,
		"GET /api/v1/volunteers/applications":      false,
		"GET /api/v1/volunteers/submissions":       false,
		"GET /api/v1/volunteers/payments":          false,
		"GET /api/v1/volunteers/payment-profile":   false,
		"PUT /api/v1/volunteers/payment-profile":   false,
		"GET /api/v1/volunteers/reputation":        false,
		"GET /api/v1/trustees/payouts/:slug":       false,
		"POST /api/v1/trustees/payouts/:slug/sign": false,
		"GET /api/v1/ledger/tasks/:slug":           false,
		"GET /api/v1/ledger/tasks/:slug/verify":    false,
		"GET /api/v1/lightning/invoices/status":    false,
	}

	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, tracked := expected[key]; tracked {
			expected[key] = true
		}
	}
	for route, found := range expected {
		if !found {
			t.Errorf("expected registered route %s", route)
		}
	}
}

func TestProtectedRouteReturnsStandardUnauthenticatedError(t *testing.T) {
	router := contractTestRouter()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	var body apihttp.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != apihttp.CodeUnauthenticated || body.Message == "" {
		t.Fatalf("unexpected error response: %#v", body)
	}
}

func TestMissingRouteReturnsStandardNotFoundError(t *testing.T) {
	router := contractTestRouter()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
	var body apihttp.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != apihttp.CodeNotFound {
		t.Fatalf("unexpected error response: %#v", body)
	}
}
