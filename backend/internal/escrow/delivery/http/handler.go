package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/escrow"
)

type EscrowHandler struct {
	service escrow.PayoutOrchestrator
}

func NewEscrowHandler(service escrow.PayoutOrchestrator) *EscrowHandler {
	return &EscrowHandler{service: service}
}

func (h *EscrowHandler) GetPayoutReviewManifest(c *gin.Context) {
	taskSlug := c.Param("slug")

	// Get destination address and volunteer invoice from request
	destinationAddress := c.Query("destination_address")
	volunteerInvoice := c.Query("volunteer_invoice")

	manifest, err := h.service.PreparePayoutManifest(
		c.Request.Context(),
		taskSlug,
		destinationAddress,
		volunteerInvoice,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, PayoutReviewResponse{
		TaskSlug:         manifest.TaskSlug,
		UnsignedPsbtHex:  "unsigned_psbt_placeholder",
		VolunteerInvoice: "volunteer_invoice_placeholder",
		L1AmountSats:     0,
		L2AmountSats:     0,
	})
}

func (h *EscrowHandler) SubmitCoSignatures(c *gin.Context) {
	taskSlug := c.Param("slug")

	var req CoSignPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payload := &escrow.SignatureCollection{
		TrusteePublicKeyHex:   req.TrusteePublicKeyHex,
		L1SignatureFragment:   req.Layer1PsbtSignatureFragment,
		L2WebCryptoSignature: req.Layer2WebCryptoSignature,
	}

	thresholdReached, err := h.service.SubmitTrusteeSignature(c.Request.Context(), taskSlug, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if thresholdReached {
		if err := h.service.FinalizeAndBroadcastPayout(c.Request.Context(), taskSlug); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"threshold_reached": thresholdReached,
		"message":          "Signature submitted",
	})
}