package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
)

// ErrorHandler recovers panics and converts binding/validation errors into JSON responses.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "unexpected server error")
			}
		}()

		c.Next()
	}
}

// ValidationMiddleware checks the media type when a write request contains a
// body. Bodyless operations such as session sign-out are valid and must not be
// forced to send meaningless JSON merely because they use POST.
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		hasBody := c.Request.ContentLength != 0
		if hasBody && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
			contentType := strings.ToLower(c.GetHeader("Content-Type"))
			if contentType == "" || !strings.Contains(contentType, "application/json") {
				apihttp.WriteError(c, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
				return
			}
		}

		c.Next()
	}
}

// RateLimiter is a simple stub for rate limiting
func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
