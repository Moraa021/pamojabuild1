package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/trustee"
	trusteeService "pamojabuild1/backend/internal/trustee/service"
)

type TrusteeHandler struct {
	service trustee.Service
}

func NewTrusteeHandler(service trustee.Service) *TrusteeHandler {
	return &TrusteeHandler{service: service}
}

// RegisterTrusteeKeys godoc
// @Summary      Register trustee keys
// @Description  Fill an empty trustee scaffold slot using the authenticated account ID. Nomination, acceptance, and proof-of-key ownership remain deferred to trustee onboarding.
// @Tags         Trustees
// @Accept       json
// @Produce      json
// @Param        slug  path  string                      true  "Task slug"
// @Param        body  body  RegisterTrusteeKeysRequest  true  "Trustee key registration payload"
// @Success      201   {object}  TrusteeRegistrationResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      404   {object}  apihttp.ErrorResponse
// @Failure      409   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug}/trustees [post]
func (h *TrusteeHandler) RegisterTrusteeKeys(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		apihttp.WriteError(c, http.StatusUnauthorized, apihttp.CodeUnauthenticated, "valid session cookie required")
		return
	}

	var req RegisterTrusteeKeysRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}

	key := &trustee.TrusteeKey{
		// Identity is security-sensitive and comes only from the authenticated
		// session, never a caller-selected user_id field.
		UserID:             userID,
		TrusteeIndex:       req.TrusteeIndex,
		Xpub:               req.Xpub,
		WebCryptoPubkeyHex: req.WebCryptoPubkeyHex,
	}
	if err := h.service.AssignTrusteeSlot(c.Request.Context(), c.Param("slug"), key); err != nil {
		switch {
		case errors.Is(err, trusteeService.ErrInvalidTrusteeIndex),
			errors.Is(err, trusteeService.ErrInvalidTrusteeKeys):
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, err.Error())
		case errors.Is(err, trusteeService.ErrTrusteeTaskNotFound):
			apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "task not found")
		case errors.Is(err, trusteeService.ErrSlotAlreadyTaken),
			errors.Is(err, trusteeService.ErrTrusteeConflict):
			apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, err.Error())
		default:
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not register trustee keys")
		}
		return
	}

	c.JSON(http.StatusCreated, TrusteeRegistrationResponse{
		TaskSlug:     key.TaskSlug,
		TrusteeIndex: key.TrusteeIndex,
		UserID:       key.UserID,
	})
}
