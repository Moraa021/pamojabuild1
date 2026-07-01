package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/trustee"
)

type TrusteeHandler struct {
	service trustee.Service
}

func NewTrusteeHandler(service trustee.Service) *TrusteeHandler {
	return &TrusteeHandler{service: service}
}

func (h *TrusteeHandler) RegisterTrusteeKeys(c *gin.Context) {
	taskSlug := c.Param("slug")

	var req RegisterTrusteeKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := &trustee.TrusteeKey{
		UserID:             req.UserID,
		TrusteeIndex:       req.TrusteeIndex,
		Xpub:               req.Xpub,
		WebCryptoPubkeyHex: req.WebCryptoPubkeyHex,
	}

	if err := h.service.AssignTrusteeSlot(c.Request.Context(), taskSlug, key); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Trustee keys registered"})
}

func (h *TrusteeHandler) GetTrustees(c *gin.Context) {
	taskSlug := c.Param("slug")

	keys, err := h.service.GetTaskTrustees(c.Request.Context(), taskSlug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"trustees": keys})
}

func (h *TrusteeHandler) VerifySignature(c *gin.Context) {
	var req struct {
		PublicKeyHex string `json:"public_key_hex" binding:"required"`
		Message      string `json:"message" binding:"required"`
		SignatureHex string `json:"signature_hex" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := h.service.VerifyWebCryptoSignature(
		c.Request.Context(),
		req.PublicKeyHex,
		[]byte(req.Message),
		req.SignatureHex,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": valid})
}