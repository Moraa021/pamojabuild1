package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/auth"
)

func AuthMiddleware(authService auth.Service, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "auth_required", Message: "Valid session cookie required"})
			c.Abort()
			return
		}

		user, err := authService.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid_session", Message: "Invalid or expired session"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("session_token", token)
		c.Next()
	}
}
