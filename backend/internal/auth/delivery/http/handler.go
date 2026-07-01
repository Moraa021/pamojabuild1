package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/auth"
)

type AuthHandler struct {
	service auth.Service
}

func NewAuthHandler(service auth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "validation_error", Message: err.Error()})
		return
	}

	user, token, err := h.service.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: "registration_failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token:       token,
		UserID:      user.ID,
		Role:        user.Role,
		DisplayName: user.DisplayName,
	})
}

func (h *AuthHandler) SignIn(c *gin.Context) {
	var req SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "validation_error", Message: err.Error()})
		return
	}

	user, token, err := h.service.SignIn(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "auth_failed", Message: "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:       token,
		UserID:      user.ID,
		Role:        user.Role,
		DisplayName: user.DisplayName,
	})
}

func (h *AuthHandler) SignOut(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Signed out successfully"})
}
