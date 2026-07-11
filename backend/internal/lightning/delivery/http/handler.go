package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/lightning"
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
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/tasks/{slug}/donate [post]
func (h *LightningHandler) RequestDonationInvoice(c *gin.Context) {
	taskSlug := c.Param("slug")

	var req DonationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoice, err := h.service.RequestDonationInvoice(c.Request.Context(), taskSlug, req.AmountSats)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Failure      400           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Router       /api/v1/lightning/invoices/status [get]
func (h *LightningHandler) CheckInvoiceStatus(c *gin.Context) {
	paymentHash := c.Query("payment_hash")
	if paymentHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_hash is required"})
		return
	}

	invoice, err := h.service.GetInvoiceStatus(c.Request.Context(), paymentHash)
	if err != nil {
		if errors.Is(err, lightning.ErrInvoiceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, lightning.ErrInvalidPaymentHash) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
