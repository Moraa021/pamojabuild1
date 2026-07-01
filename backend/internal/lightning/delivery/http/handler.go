package http

import (
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
		ExpiresAt:      invoice.SettledAt.Unix() + 3600, // 1 hour expiry
	})
}

func (h *LightningHandler) CheckInvoiceStatus(c *gin.Context) {
	paymentHash := c.Query("payment_hash")
	
	// This would need a GetByPaymentHash method on the service
	// For now, return a placeholder
	c.JSON(http.StatusOK, gin.H{
		"payment_hash": paymentHash,
		"settled":      false,
	})
}