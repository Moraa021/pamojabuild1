package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/auth"
	authsvc "pamojabuild1/backend/internal/auth/service"
)

func AuthMiddleware(service auth.Service, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			apihttp.WriteError(c, http.StatusUnauthorized, apihttp.CodeUnauthenticated, "valid session cookie required")
			return
		}

		user, err := service.Authenticate(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, authsvc.ErrInvalidSession) {
				apihttp.WriteError(c, http.StatusUnauthorized, apihttp.CodeUnauthenticated, "invalid or expired session")
			} else {
				apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not validate session")
			}
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("session_token", token)
		c.Next()
	}
}
