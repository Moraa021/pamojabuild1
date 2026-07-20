package authorization

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubTrusteeReader struct {
	allowed bool
	err     error
}

func (s stubTrusteeReader) IsTaskTrustee(context.Context, string, int64) (bool, error) {
	return s.allowed, s.err
}

func runTrusteeMiddleware(t *testing.T, reader TrusteeRelationshipReader, userID int64) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/trustees/payouts/:slug",
		func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		},
		RequireTaskTrustee(reader),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/trustees/payouts/a-task", nil))
	return recorder
}

func TestRequireTaskTrusteeAllowsAssignedTrustee(t *testing.T) {
	recorder := runTrusteeMiddleware(t, stubTrusteeReader{allowed: true}, 42)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestRequireTaskTrusteeRejectsUnassignedAccount(t *testing.T) {
	recorder := runTrusteeMiddleware(t, stubTrusteeReader{}, 42)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, recorder.Code)
	}
}

func TestRequireTaskTrusteeFailsClosedOnRepositoryError(t *testing.T) {
	recorder := runTrusteeMiddleware(t, stubTrusteeReader{err: errors.New("database unavailable")}, 42)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}
