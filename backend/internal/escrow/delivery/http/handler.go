package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/escrow"
	escrowService "pamojabuild1/backend/internal/escrow/service"
)

type EscrowHandler struct {
	service escrow.PayoutOrchestrator
}

func NewEscrowHandler(service escrow.PayoutOrchestrator) *EscrowHandler {
	return &EscrowHandler{service: service}
}

// GetPayoutReviewManifest godoc
// @Summary      Get payout review scaffold
// @Description  Return the current placeholder payout review contract. Destinations are not accepted from query parameters; a later payout-intent workflow must derive and freeze them server-side.
// @Tags         Escrow
// @Produce      json
// @Param        slug  path  string  true  "Task slug"
// @Success      200   {object}  PayoutReviewResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      403   {object}  apihttp.ErrorResponse
// @Failure      409   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/trustees/payouts/{slug} [get]
func (h *EscrowHandler) GetPayoutReviewManifest(c *gin.Context) {
	manifest, err := h.service.PreparePayoutManifest(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writePayoutError(c, err)
		return
	}
	c.JSON(http.StatusOK, PayoutReviewResponse{
		TaskSlug:         manifest.TaskSlug,
		UnsignedPsbtHex:  manifest.UnsignedPSBTHex,
		VolunteerInvoice: manifest.VolunteerInvoice,
		L1AmountSats:     manifest.L1AmountSats,
		L2AmountSats:     manifest.L2AmountSats,
	})
}

// SubmitCoSignatures godoc
// @Summary      Submit trustee co-signature scaffold
// @Description  Store placeholder signature fragments under the authenticated trustee's registered key. Signer identity/public key cannot be selected in JSON; cryptographic validation remains deferred to the PSBT implementation.
// @Tags         Escrow
// @Accept       json
// @Produce      json
// @Param        slug  path  string               true  "Task slug"
// @Param        body  body  CoSignPayoutRequest  true  "Co-signature payload"
// @Success      200   {object}  CoSignPayoutResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      403   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/trustees/payouts/{slug}/sign [post]
func (h *EscrowHandler) SubmitCoSignatures(c *gin.Context) {
	var req CoSignPayoutRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	thresholdReached, err := h.service.SubmitTrusteeSignature(
		c.Request.Context(),
		c.Param("slug"),
		c.GetInt64("user_id"),
		req.Layer1PsbtSignatureFragment,
		req.Layer2WebCryptoSignature,
	)
	if err != nil {
		writePayoutError(c, err)
		return
	}
	c.JSON(http.StatusOK, CoSignPayoutResponse{
		ThresholdReached: thresholdReached,
		Message:          "signature submitted",
	})
}

func writePayoutError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, escrowService.ErrInvalidPayoutRequest):
		apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, "payout request is invalid")
	case errors.Is(err, escrowService.ErrTrusteeNotAssigned):
		apihttp.WriteError(c, http.StatusForbidden, apihttp.CodeUnauthorized, err.Error())
	case errors.Is(err, escrowService.ErrPayoutNotReady):
		apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, "all five trustee scaffold slots must be filled")
	default:
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not process payout review request")
	}
}
