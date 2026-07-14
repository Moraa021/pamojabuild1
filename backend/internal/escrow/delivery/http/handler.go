package http

import (
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
		TrusteePublicKeyHex:  req.TrusteePublicKeyHex,
		L1SignatureFragment:  req.Layer1PsbtSignatureFragment,
		L2WebCryptoSignature: req.Layer2WebCryptoSignature,
	}
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
	})
}
