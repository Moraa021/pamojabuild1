package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/auth"
	authService "pamojabuild1/backend/internal/auth/service"
)

type AuthHandler struct {
	service auth.Service
}

func NewAuthHandler(service auth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account using phone number and password, and return a JWT token.
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

	user, token, err := h.service.Register(c.Request.Context(), req.PhoneNumber, req.Password, req.DisplayName)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, authService.ErrUserExists) {
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: "registration_failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token:       token,
		UserID:      user.ID,
		IsAdmin:     user.IsAdmin,
		DisplayName: user.DisplayName,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	})
}

// SignIn godoc
// @Summary      Sign in a user
// @Description  Authenticate with phone number and password to receive a JWT token.
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

	user, token, err := h.service.SignIn(c.Request.Context(), req.PhoneNumber, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "auth_failed", Message: "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:       token,
		UserID:      user.ID,
		IsAdmin:     user.IsAdmin,
		DisplayName: user.DisplayName,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	})
}

// SignOut godoc
// @Summary      Sign out the current user
// @Description  Invalidate the current session token or clear authentication state.
// @Tags         Auth
// @Produce      json
// @Success      204
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/auth/signout [post]
func (h *AuthHandler) SignOut(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "auth_required", Message: "Authenticated account required"})
		return
	}
	if err := h.service.SignOut(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "signout_failed", Message: "Could not revoke session"})
		return
	}
	c.Status(http.StatusNoContent)
}
