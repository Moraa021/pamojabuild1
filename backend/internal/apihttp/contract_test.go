package apihttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type contractRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
}

func runBindJSON(body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/", func(c *gin.Context) {
		var request contractRequest
		if !BindJSON(c, &request) {
			return
		}
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestBindJSONUsesSnakeCaseValidationFields(t *testing.T) {
	response := runBindJSON(`{}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}

	var body ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != CodeValidation || body.Fields["display_name"] != "is required" {
		t.Fatalf("unexpected validation response: %#v", body)
	}
}

func TestBindJSONRejectsUnknownFields(t *testing.T) {
	response := runBindJSON(`{"display_name":"Amina","user_id":99}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}

	var body ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Fields["user_id"] != "is not allowed" {
		t.Fatalf("expected unknown actor field rejection, got %#v", body)
	}
}
