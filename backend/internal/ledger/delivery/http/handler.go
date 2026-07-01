package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/ledger"
)

type LedgerHandler struct {
	service ledger.SecurityService
}

func NewLedgerHandler(service ledger.SecurityService) *LedgerHandler {
	return &LedgerHandler{service: service}
}

func (h *LedgerHandler) GetTaskBalance(c *gin.Context) {
	taskSlug := c.Param("slug")

	balance, err := h.service.GetTaskBalance(c.Request.Context(), taskSlug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}

func (h *LedgerHandler) VerifyChainIntegrity(c *gin.Context) {
	taskSlug := c.Param("slug")

	valid, err := h.service.VerifyEntireChainIntegrity(c.Request.Context(), taskSlug, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_slug": taskSlug,
		"valid":     valid,
	})
}

func (h *LedgerHandler) RecordTransaction(c *gin.Context) {
	taskSlug := c.Param("slug")

	var req struct {
		EntryType   string `json:"entry_type" binding:"required"`
		AmountSats  int64  `json:"amount_sats" binding:"required"`
		ReferenceID string `json:"reference_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RecordValidatedTransaction(
		c.Request.Context(),
		taskSlug,
		req.EntryType,
		req.AmountSats,
		req.ReferenceID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Transaction recorded"})
}