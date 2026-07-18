package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/ledger"
	ledgerService "pamojabuild1/backend/internal/ledger/service"
)

type LedgerHandler struct {
	service ledger.SecurityService
}

func NewLedgerHandler(service ledger.SecurityService) *LedgerHandler {
	return &LedgerHandler{service: service}
}

// GetTaskBalance godoc
// @Summary      Get ledger balance for a task
// @Description  Retrieve the current ledger balance summary for an existing task.
// @Tags         Ledger
// @Produce      json
// @Param        slug  path      string  true  "Task slug"
// @Success      200   {object}  TaskBalanceResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      404   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/ledger/tasks/{slug} [get]
func (h *LedgerHandler) GetTaskBalance(c *gin.Context) {
	balance, err := h.service.GetTaskBalance(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeLedgerError(c, err)
		return
	}
	c.JSON(http.StatusOK, TaskBalanceResponse{
		L2BalanceSats: balance.L2BalanceSats,
		L1BalanceSats: balance.L1BalanceSats,
		CurrentIndex:  balance.CurrentIndex,
	})
}

// VerifyChainIntegrity godoc
// @Summary      Verify ledger chain integrity
// @Description  Validate the cryptographic chain integrity for an existing task ledger.
// @Tags         Ledger
// @Produce      json
// @Param        slug  path      string  true  "Task slug"
// @Success      200   {object}  VerifyChainResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      404   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/ledger/tasks/{slug}/verify [get]
func (h *LedgerHandler) VerifyChainIntegrity(c *gin.Context) {
	taskSlug := c.Param("slug")
	valid, err := h.service.VerifyEntireChainIntegrity(c.Request.Context(), taskSlug)
	if err != nil {
		writeLedgerError(c, err)
		return
	}
	c.JSON(http.StatusOK, VerifyChainResponse{TaskSlug: taskSlug, Valid: valid})
}

func writeLedgerError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ledgerService.ErrInvalidTaskSlug):
		apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, "task slug is invalid")
	case errors.Is(err, ledgerService.ErrLedgerTaskNotFound):
		apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "task not found")
	default:
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not read task ledger")
	}
}
