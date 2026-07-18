// Package apihttp contains transport-level contracts shared by every HTTP
// domain. Keeping errors and request decoding here prevents individual handlers
// from exposing repository errors or inventing incompatible response shapes.
package apihttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

const (
	CodeValidation      = "validation_error"
	CodeUnauthenticated = "unauthenticated"
	CodeUnauthorized    = "unauthorized"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeInternal        = "internal_error"
)

// ErrorResponse is the only error envelope exposed by the API.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// MessageResponse is used only when an endpoint has useful confirmation data
// but no resource representation to return.
type MessageResponse struct {
	Message string `json:"message"`
}

// PaginationResponse describes page-number pagination used by list endpoints.
type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

func WriteError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: code, Message: message})
}

func WriteValidationError(c *gin.Context, fields map[string]string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
		Error:   CodeValidation,
		Message: "request validation failed",
		Fields:  fields,
	})
}

// BindJSON decodes exactly one JSON object, rejects unknown fields, and applies
// Gin's configured struct validator. Raw decoder and validator errors are never
// returned to clients because they can expose Go implementation details.
func BindJSON(c *gin.Context, destination any) bool {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		WriteValidationError(c, decodeErrorFields(err))
		return false
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		WriteValidationError(c, map[string]string{"body": "must contain exactly one JSON object"})
		return false
	}

	if err := binding.Validator.ValidateStruct(destination); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			WriteValidationError(c, validationFields(destination, validationErrors))
			return false
		}
		WriteValidationError(c, map[string]string{"body": "is invalid"})
		return false
	}

	return true
}

func decodeErrorFields(err error) map[string]string {
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	switch {
	case errors.Is(err, io.EOF):
		return map[string]string{"body": "is required"}
	case errors.As(err, &syntaxError):
		return map[string]string{"body": "must be valid JSON"}
	case errors.As(err, &typeError):
		field := typeError.Field
		if field == "" {
			field = "body"
		}
		return map[string]string{field: "has an invalid type"}
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.Trim(err.Error()[len("json: unknown field "):], `"`)
		return map[string]string{field: "is not allowed"}
	default:
		return map[string]string{"body": "must be a valid JSON object"}
	}
}

func validationFields(destination any, validationErrors validator.ValidationErrors) map[string]string {
	fields := make(map[string]string, len(validationErrors))
	valueType := reflect.TypeOf(destination)
	if valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}

	for _, fieldError := range validationErrors {
		name := fieldError.Field()
		if structField, ok := valueType.FieldByName(fieldError.StructField()); ok {
			if jsonName := strings.Split(structField.Tag.Get("json"), ",")[0]; jsonName != "" && jsonName != "-" {
				name = jsonName
			}
		}

		switch fieldError.Tag() {
		case "required":
			fields[name] = "is required"
		case "min":
			fields[name] = "is below the minimum allowed value or length"
		case "max":
			fields[name] = "exceeds the maximum allowed value or length"
		case "gt":
			fields[name] = "must be greater than " + fieldError.Param()
		case "oneof":
			fields[name] = "must be one of: " + strings.ReplaceAll(fieldError.Param(), " ", ", ")
		default:
			fields[name] = "is invalid"
		}
	}
	return fields
}
