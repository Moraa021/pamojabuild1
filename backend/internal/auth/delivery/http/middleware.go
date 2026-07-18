package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/auth"
)

func AuthMiddleware(authService auth.Service, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			apihttp.WriteError(c, http.StatusUnauthorized, apihttp.CodeUnauthenticated, "valid session cookie required")
			return
		}

		user, err := authService.Authenticate(c.Request.Context(), token)
		if err != nil {
			apihttp.WriteError(c, http.StatusUnauthorized, apihttp.CodeUnauthenticated, "invalid or expired session")
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("session_token", token)
		c.Next()
	}
}
