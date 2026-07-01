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

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account and return an authentication token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest  true  "Registration payload"
// @Success      201   {object}  AuthResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Router       /api/v1/auth/register [post]
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

// SignIn godoc
// @Summary      Sign in a user
// @Description  Authenticate with email and password to receive a JWT token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      SignInRequest  true  "Sign in payload"
// @Success      200   {object}  AuthResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Router       /api/v1/auth/signin [post]
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

// SignOut godoc
// @Summary      Sign out the current user
// @Description  Invalidate the current session token or clear authentication state.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /api/v1/auth/signout [post]
func (h *AuthHandler) SignOut(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Signed out successfully"})
}
