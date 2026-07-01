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

// GetTaskBalance godoc
// @Summary      Get ledger balance for a task
// @Description  Retrieve the current ledger balance summary for a task.
// @Tags         Ledger
// @Produce      json
// @Param        slug  path      string  true  "Task slug"
// @Success      200   {object}  ledger.BalanceSummary
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/ledger/tasks/{slug} [get]
func (h *LedgerHandler) GetTaskBalance(c *gin.Context) {
	taskSlug := c.Param("slug")

	balance, err := h.service.GetTaskBalance(c.Request.Context(), taskSlug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}

// VerifyChainIntegrity godoc
// @Summary      Verify ledger chain integrity
// @Description  Validate the cryptographic chain integrity of a task ledger.
// @Tags         Ledger
// @Produce      json
// @Param        slug  path      string  true  "Task slug"
// @Success      200   {object}  VerifyChainResponse
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/ledger/tasks/{slug}/verify [get]
func (h *LedgerHandler) VerifyChainIntegrity(c *gin.Context) {
	taskSlug := c.Param("slug")

	valid, err := h.service.VerifyEntireChainIntegrity(c.Request.Context(), taskSlug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_slug": taskSlug,
		"valid":     valid,
	})
}

// RecordTransaction godoc
// @Summary      Record a ledger transaction
// @Description  Append a validated ledger transaction for a task.
// @Tags         Ledger
// @Accept       json
// @Produce      json
// @Param        slug  path      string                    true  "Task slug"
// @Param        body  body      LedgerTransactionRequest  true  "Transaction payload"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/ledger/tasks/{slug} [post]
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
