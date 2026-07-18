package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newBrowserSecurityRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(BrowserSecurity([]string{"http://localhost:3000"}))
	router.POST("/change", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return router
}

func TestBrowserSecurityAllowsConfiguredCredentialedOrigin(t *testing.T) {
	router := newBrowserSecurityRouter()
	request := httptest.NewRequest(http.MethodPost, "/change", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Sec-Fetch-Site", "same-site")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, response.Code)
	}
	if value := response.Header().Get("Access-Control-Allow-Origin"); value != "http://localhost:3000" {
		t.Fatalf("expected exact allowed origin, got %q", value)
	}
	if value := response.Header().Get("Access-Control-Allow-Credentials"); value != "true" {
		t.Fatalf("expected credentialed CORS response, got %q", value)
	}
}

func TestBrowserSecurityRejectsCrossSiteMutation(t *testing.T) {
	router := newBrowserSecurityRouter()
	request := httptest.NewRequest(http.MethodPost, "/change", nil)
	request.Header.Set("Origin", "https://attacker.example")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestBrowserSecurityRejectsUnlistedOriginWithoutFetchMetadata(t *testing.T) {
	router := newBrowserSecurityRouter()
	request := httptest.NewRequest(http.MethodPost, "/change", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, response.Code)
	}
}
