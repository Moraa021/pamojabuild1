package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
)

func BrowserSecurity(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimRight(origin, "/")] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		_, originAllowed := allowed[origin]

		// Sec-Fetch-Site is supplied by modern browsers and cannot be set by
		// ordinary page JavaScript. Rejecting cross-site mutations complements
		// SameSite cookies and protects endpoints even when no JSON preflight is
		// involved.
		if isStateChanging(c.Request.Method) &&
			strings.EqualFold(c.GetHeader("Sec-Fetch-Site"), "cross-site") {
			apihttp.WriteError(c, http.StatusForbidden, apihttp.CodeUnauthorized, "cross-site request rejected")
			return
		}
		if isStateChanging(c.Request.Method) && origin != "" && !originAllowed {
			apihttp.WriteError(c, http.StatusForbidden, apihttp.CodeUnauthorized, "origin not allowed")
			return
		}

		if origin != "" && originAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}

		if c.Request.Method == http.MethodOptions {
			if origin == "" || !originAllowed {
				apihttp.WriteError(c, http.StatusForbidden, apihttp.CodeUnauthorized, "origin not allowed")
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func isStateChanging(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
