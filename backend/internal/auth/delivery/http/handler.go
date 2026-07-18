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
	service      auth.Service
	cookieName   string
	cookieSecure bool
}

func NewAuthHandler(service auth.Service, cookieName string, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		service:      service,
		cookieName:   cookieName,
		cookieSecure: cookieSecure,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a user account and start an HttpOnly cookie session.
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

	session, err := h.service.Register(c.Request.Context(), req.PhoneNumber, req.Password, req.DisplayName)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, authService.ErrUserExists) {
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: "registration_failed", Message: err.Error()})
		return
	}

	h.setSessionCookie(c, session)
	c.JSON(http.StatusCreated, AuthResponse{
		UserID:      session.User.ID,
		IsAdmin:     session.User.IsAdmin,
		DisplayName: session.User.DisplayName,
		ExpiresAt:   session.ExpiresAt,
	})
}

// SignIn godoc
// @Summary      Sign in a user
// @Description  Authenticate with phone number and password and start an HttpOnly cookie session.
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

	session, err := h.service.SignIn(c.Request.Context(), req.PhoneNumber, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "auth_failed", Message: "Invalid credentials"})
		return
	}

	h.setSessionCookie(c, session)
	c.JSON(http.StatusOK, AuthResponse{
		UserID:      session.User.ID,
		IsAdmin:     session.User.IsAdmin,
		DisplayName: session.User.DisplayName,
		ExpiresAt:   session.ExpiresAt,
	})
}

// SignOut godoc
// @Summary      Sign out the current user
// @Description  Revoke the current server-side session and clear its cookie.
// @Tags         Auth
// @Produce      json
// @Success      204
// @Failure      500  {object}  ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/auth/signout [post]
func (h *AuthHandler) SignOut(c *gin.Context) {
	token := c.GetString("session_token")
	if err := h.service.SignOut(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "signout_failed", Message: "Could not revoke session"})
		return
	}
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, session *auth.Session) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
