package http

import (
<<<<<<< HEAD
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

// GetPayoutReviewManifest godoc
// @Summary      Get payout review manifest
// @Description  Retrieve an escrow payout review manifest for a trustee to inspect.
// @Tags         Escrow
// @Produce      json
// @Param        slug                path      string  true  "Task slug"
// @Param        destination_address query     string  false "Payment destination address"
// @Param        volunteer_invoice   query     string  false "Volunteer invoice reference"
// @Success      200                 {object}  PayoutReviewResponse
// @Failure      500                 {object}  map[string]string
// @Router       /api/v1/trustees/payouts/{slug} [get]
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

// SubmitCoSignatures godoc
// @Summary      Submit trustee co-signatures
// @Description  Submit trustee signature fragments for a task payout.
// @Tags         Escrow
// @Accept       json
// @Produce      json
// @Param        slug  path  string             true  "Task slug"
// @Param        body  body  CoSignPayoutRequest  true  "Co-signature payload"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/trustees/payouts/{slug}/sign [post]
func (h *EscrowHandler) SubmitCoSignatures(c *gin.Context) {
	taskSlug := c.Param("slug")

	var req CoSignPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payload := &escrow.SignatureCollection{
=======
	"context"
	"encoding/json"
	"net/http"
	"pamojabuild1/internal/escrow"

	"github.com/gorilla/mux"
)

type OrchestratorService interface {
	PreparePayoutManifest(ctx context.Context, taskSlug string, destinationAddress string, volunteerInvoice string) (*escrow.SignatureCollection, error)
	SubmitTrusteeSignature(ctx context.Context, taskSlug string, payload *escrow.SignatureCollection) (bool, error)
	FinalizeAndBroadcastPayout(ctx context.Context, taskSlug string) error
}

type Handler struct {
	Service OrchestratorService
}

func NewHandler(svc OrchestratorService) *Handler {
	return &Handler{Service: svc}
}

func (h *Handler) GetPayoutReviewManifest(w http.ResponseWriter, r *http.Request) {
	taskSlug := mux.Vars(r)["task_slug"]
	if taskSlug == "" {
		http.Error(w, "task_slug is required", http.StatusBadRequest)
		return
	}
	manifest, err := h.Service.PreparePayoutManifest(r.Context(), taskSlug, "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := PayoutReviewResponse{
		TaskSlug:         manifest.TaskSlug,
		UnsignedPsbtHex:  "pending",
		VolunteerInvoice: "",
		L1AmountSats:     0,
		L2AmountSats:     0,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SubmitCoSignatures(w http.ResponseWriter, r *http.Request) {
	taskSlug := mux.Vars(r)["task_slug"]
	if taskSlug == "" {
		http.Error(w, "task_slug is required", http.StatusBadRequest)
		return
	}
	var req CoSignPayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.TrusteePublicKeyHex == "" || req.Layer1PsbtSignatureFragment == "" || req.Layer2WebCryptoSignature == "" {
		http.Error(w, "all signature fields are required", http.StatusBadRequest)
		return
	}
	payload := &escrow.SignatureCollection{
		TaskSlug:             taskSlug,
>>>>>>> feature/escrow-psbt
		TrusteePublicKeyHex:  req.TrusteePublicKeyHex,
		L1SignatureFragment:  req.Layer1PsbtSignatureFragment,
		L2WebCryptoSignature: req.Layer2WebCryptoSignature,
	}
<<<<<<< HEAD

	thresholdReached, err := h.service.SubmitTrusteeSignature(c.Request.Context(), taskSlug, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Payout finalization is driven by event subscribers on threshold reached.
	c.JSON(http.StatusOK, gin.H{
		"threshold_reached": thresholdReached,
		"message":           "Signature submitted",
=======
	thresholdMet, err := h.Service.SubmitTrusteeSignature(r.Context(), taskSlug, payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if thresholdMet {
		if err := h.Service.FinalizeAndBroadcastPayout(r.Context(), taskSlug); err != nil {
			http.Error(w, "threshold met but broadcast failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"threshold_met": true,
			"message":       "3/5 signatures collected — payout finalized and broadcast",
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"threshold_met": false,
		"message":       "signature recorded, waiting for more trustees",
>>>>>>> feature/escrow-psbt
	})
}
