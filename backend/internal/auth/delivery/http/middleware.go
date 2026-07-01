package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/auth"
)

func AuthMiddleware(authService auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "auth_required", Message: "Authorization header required"})
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid_token", Message: "Invalid token format"})
			c.Abort()
			return
		}

		user, err := authService.ValidateToken(c.Request.Context(), tokenParts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid_token", Message: "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	}
}
