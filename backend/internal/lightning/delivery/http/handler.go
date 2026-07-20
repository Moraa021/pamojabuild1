package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/lightning"
	lightningService "pamojabuild1/backend/internal/lightning/service"
)

type LightningHandler struct {
	service lightning.Service
}

func NewLightningHandler(service lightning.Service) *LightningHandler {
	return &LightningHandler{service: service}
}

// RequestDonationInvoice godoc
// @Summary      Request a donation invoice
// @Description  Create a Lightning invoice for a task donation.
// @Tags         Lightning
// @Accept       json
// @Produce      json
// @Param        slug  path      string           true  "Task slug"
// @Param        body  body      DonationRequest  true  "Donation request payload"
// @Success      201   {object}  DonationInvoiceResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      404   {object}  apihttp.ErrorResponse
// @Failure      409   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug}/donate [post]
func (h *LightningHandler) RequestDonationInvoice(c *gin.Context) {
	taskSlug := c.Param("slug")

	var req DonationRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}

	invoice, err := h.service.RequestDonationInvoice(c.Request.Context(), taskSlug, req.AmountSats)
	if err != nil {
		switch {
		case errors.Is(err, lightningService.ErrInvalidDonation):
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, "amount_sats must be greater than zero and task slug must be valid")
		case errors.Is(err, lightningService.ErrDonationTaskNotFound):
			apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "task not found")
		case errors.Is(err, lightningService.ErrDonationsNotAllowed):
			apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, "task is not accepting donations")
		default:
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not create donation invoice")
		}
		return
	}

	c.JSON(http.StatusCreated, DonationInvoiceResponse{
		PaymentRequest: invoice.PaymentRequest,
		PaymentHash:    invoice.PaymentHash,
		ExpiresAt:      invoice.ExpiresAt.Unix(),
	})
}

// CheckInvoiceStatus godoc
// @Summary      Check Lightning invoice status
// @Description  Check the settlement status of a Lightning payment hash.
// @Tags         Lightning
// @Produce      json
// @Param        payment_hash  query  string  true  "Payment hash"
// @Success      200           {object}  InvoiceStatusResponse
// @Failure      400           {object}  apihttp.ErrorResponse
// @Failure      401           {object}  apihttp.ErrorResponse
// @Failure      404           {object}  apihttp.ErrorResponse
// @Failure      500           {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/lightning/invoices/status [get]
func (h *LightningHandler) CheckInvoiceStatus(c *gin.Context) {
	paymentHash := c.Query("payment_hash")
	if paymentHash == "" {
		apihttp.WriteValidationError(c, map[string]string{"payment_hash": "is required"})
		return
	}

	invoice, err := h.service.GetInvoiceStatus(c.Request.Context(), paymentHash)
	if err != nil {
		if errors.Is(err, lightning.ErrInvoiceNotFound) {
			apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "invoice not found")
			return
		}
		if errors.Is(err, lightning.ErrInvalidPaymentHash) {
			apihttp.WriteValidationError(c, map[string]string{"payment_hash": "must be 64 hexadecimal characters"})
			return
		}
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not load invoice status")
		return
	}

	response := InvoiceStatusResponse{
		PaymentHash: invoice.PaymentHash,
		Status:      invoice.Status,
		Settled:     invoice.Settled,
	}
	if !invoice.ExpiresAt.IsZero() {
		response.ExpiresAt = invoice.ExpiresAt.Unix()
	}
	if !invoice.SettledAt.IsZero() {
		response.SettledAt = invoice.SettledAt.Unix()
	}

	c.JSON(http.StatusOK, response)
}
