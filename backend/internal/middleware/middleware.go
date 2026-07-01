package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorResponse is the standard JSON error payload returned by middleware.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// ErrorHandler recovers panics and converts binding/validation errors into JSON responses.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "unexpected server error",
				})
			}
		}()

		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		for _, ginErr := range c.Errors {
			if ginErr.Err == nil {
				continue
			}

			if validationErr, ok := ginErr.Err.(validator.ValidationErrors); ok {
				c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
					Error:   "validation_error",
					Message: "request validation failed",
					Fields:  formatValidationFields(validationErr),
				})
				return
			}

			if ginErr.Type == gin.ErrorTypeBind {
				c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
					Error:   "validation_error",
					Message: ginErr.Error(),
				})
				return
			}
		}
	}
}

// ValidationMiddleware checks JSON request shape for write operations.
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
			contentType := strings.ToLower(c.GetHeader("Content-Type"))
			if contentType == "" || !strings.Contains(contentType, "application/json") {
				c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, ErrorResponse{
					Error:   "unsupported_media_type",
					Message: "Content-Type must be application/json",
				})
				return
			}

			if c.Request.ContentLength == 0 {
				c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
					Error:   "validation_error",
					Message: "request body is required",
				})
				return
			}
		}

		c.Next()
	}
}

func formatValidationFields(errs validator.ValidationErrors) map[string]string {
	fields := make(map[string]string, len(errs))
	for _, err := range errs {
		fieldName := err.Field()
		switch err.Tag() {
		case "required":
			fields[fieldName] = "is required"
		case "email":
			fields[fieldName] = "must be a valid email"
		case "min":
			fields[fieldName] = "must meet minimum length/value"
		case "gt":
			fields[fieldName] = "must be greater than the required value"
		default:
			fields[fieldName] = "invalid value"
		}
	}
	return fields
}

// RateLimiter is a simple stub for rate limiting
func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
