package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/trustee"
	trusteeService "pamojabuild1/backend/internal/trustee/service"
)

type TrusteeHandler struct{ service trustee.Service }

func NewTrusteeHandler(service trustee.Service) *TrusteeHandler {
	return &TrusteeHandler{service: service}
}

// NominateTrustee godoc
// @Summary Nominate a task trustee
// @Description Creator-only nomination into one of five task trustee slots.
// @Tags Trustees
// @Accept json
// @Produce json
// @Param slug path string true "Task slug"
// @Param body body NominateTrusteeRequest true "Nomination"
// @Success 201 {object} AssignmentResponse
// @Failure 400 {object} apihttp.ErrorResponse
// @Failure 401 {object} apihttp.ErrorResponse
// @Failure 403 {object} apihttp.ErrorResponse
// @Failure 404 {object} apihttp.ErrorResponse
// @Failure 409 {object} apihttp.ErrorResponse
// @Security CookieAuth
// @Router /api/v1/tasks/{slug}/trustees/nominations [post]
func (h *TrusteeHandler) NominateTrustee(c *gin.Context) {
	var req NominateTrusteeRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	a, err := h.service.Nominate(c.Request.Context(), c.Param("slug"), c.GetInt64("user_id"), req.TrusteeIndex, req.UserID, req.Message)
	if err != nil {
		writeServiceError(c, err, "could not nominate trustee")
		return
	}
	c.JSON(http.StatusCreated, toAssignmentResponse(a, false))
}

// AcceptNomination godoc
// @Summary Accept a trustee nomination
// @Description Accepts the authenticated account's invitation and returns the one-time proof challenge.
// @Tags Trustees
// @Produce json
// @Param slug path string true "Task slug"
// @Success 200 {object} AssignmentResponse
// @Failure 400 {object} apihttp.ErrorResponse
// @Failure 401 {object} apihttp.ErrorResponse
// @Failure 409 {object} apihttp.ErrorResponse
// @Failure 500 {object} apihttp.ErrorResponse
// @Security CookieAuth
// @Router /api/v1/tasks/{slug}/trustees/accept [post]
func (h *TrusteeHandler) AcceptNomination(c *gin.Context) {
	a, err := h.service.Accept(c.Request.Context(), c.Param("slug"), c.GetInt64("user_id"))
	if err != nil {
		writeServiceError(c, err, "could not accept trustee nomination")
		return
	}
	c.JSON(http.StatusOK, toAssignmentResponse(a, true))
}

// RegisterTrusteeKeys godoc
// @Summary Prove and register trustee keys
// @Description Activates an accepted trustee after validating xpub network and ownership proofs for both keys.
// @Tags Trustees
// @Accept json
// @Produce json
// @Param slug path string true "Task slug"
// @Param body body RegisterTrusteeKeysRequest true "Key proofs"
// @Success 201 {object} AssignmentResponse
// @Failure 400 {object} apihttp.ErrorResponse
// @Failure 401 {object} apihttp.ErrorResponse
// @Failure 409 {object} apihttp.ErrorResponse
// @Failure 500 {object} apihttp.ErrorResponse
// @Security CookieAuth
// @Router /api/v1/tasks/{slug}/trustees/keys [post]
func (h *TrusteeHandler) RegisterTrusteeKeys(c *gin.Context) {
	var req RegisterTrusteeKeysRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	a, err := h.service.RegisterKeys(c.Request.Context(), c.Param("slug"), c.GetInt64("user_id"), keyRegistration(req, ""))
	if err != nil {
		writeServiceError(c, err, "could not register trustee keys")
		return
	}
	c.JSON(http.StatusCreated, toAssignmentResponse(a, false))
}

// PrepareKeyRotation godoc
// @Summary Create a trustee key-rotation challenge
// @Tags Trustees
// @Produce json
// @Param slug path string true "Task slug"
// @Success 200 {object} ChallengeResponse
// @Failure 400 {object} apihttp.ErrorResponse
// @Failure 401 {object} apihttp.ErrorResponse
// @Failure 409 {object} apihttp.ErrorResponse
// @Failure 500 {object} apihttp.ErrorResponse
// @Security CookieAuth
// @Router /api/v1/tasks/{slug}/trustees/keys/rotation-challenge [post]
func (h *TrusteeHandler) PrepareKeyRotation(c *gin.Context) {
	challenge, err := h.service.PrepareKeyRotation(c.Request.Context(), c.Param("slug"), c.GetInt64("user_id"))
	if err != nil {
		writeServiceError(c, err, "could not prepare trustee key rotation")
		return
	}
	c.JSON(http.StatusOK, ChallengeResponse{ProofChallenge: challenge})
}

// RotateTrusteeKeys godoc
// @Summary Rotate trustee keys
// @Description Replaces the authenticated active trustee's keys while preserving the revoked key version.
// @Tags Trustees
// @Accept json
// @Produce json
// @Param slug path string true "Task slug"
// @Param body body RotateTrusteeKeysRequest true "Replacement key proofs"
// @Success 204
// @Failure 400 {object} apihttp.ErrorResponse
// @Failure 401 {object} apihttp.ErrorResponse
// @Failure 409 {object} apihttp.ErrorResponse
// @Failure 500 {object} apihttp.ErrorResponse
// @Security CookieAuth
// @Router /api/v1/tasks/{slug}/trustees/keys/rotate [post]
func (h *TrusteeHandler) RotateTrusteeKeys(c *gin.Context) {
	var req RotateTrusteeKeysRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	if err := h.service.RotateKeys(c.Request.Context(), c.Param("slug"), c.GetInt64("user_id"), keyRegistration(req.RegisterTrusteeKeysRequest, req.Reason)); err != nil {
		writeServiceError(c, err, "could not rotate trustee keys")
		return
	}
	c.Status(http.StatusNoContent)
}

// ReplaceTrustee godoc
// @Summary Replace a task trustee
// @Description Creator-only replacement that revokes the old trustee atomically and invites a new account without lowering the five-slot policy.
// @Tags Trustees
// @Accept json
// @Produce json
// @Param slug path string true "Task slug"
// @Param index path int true "Trustee slot (0-4)"
// @Param body body ReplaceTrusteeRequest true "Replacement"
// @Success 204
// @Failure 400 {object} apihttp.ErrorResponse
// @Failure 401 {object} apihttp.ErrorResponse
// @Failure 403 {object} apihttp.ErrorResponse
// @Failure 404 {object} apihttp.ErrorResponse
// @Failure 409 {object} apihttp.ErrorResponse
// @Failure 500 {object} apihttp.ErrorResponse
// @Security CookieAuth
// @Router /api/v1/tasks/{slug}/trustees/{index}/replace [post]
func (h *TrusteeHandler) ReplaceTrustee(c *gin.Context) {
	index, err := strconv.ParseInt(c.Param("index"), 10, 32)
	if err != nil {
		apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, "trustee index must be between 0 and 4")
		return
	}
	var req ReplaceTrusteeRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	if err := h.service.Replace(c.Request.Context(), c.Param("slug"), c.GetInt64("user_id"), int32(index), req.UserID, req.Reason, req.Message); err != nil {
		writeServiceError(c, err, "could not replace trustee")
		return
	}
	c.Status(http.StatusNoContent)
}

// ListTrustees godoc
// @Summary List a task's public trustee roster
// @Description Returns trustee identities and onboarding states without exposing xpubs, browser public keys, or proofs.
// @Tags Trustees
// @Produce json
// @Param slug path string true "Task slug"
// @Success 200 {object} RosterResponse
// @Failure 400 {object} apihttp.ErrorResponse
// @Failure 401 {object} apihttp.ErrorResponse
// @Failure 500 {object} apihttp.ErrorResponse
// @Security CookieAuth
// @Router /api/v1/tasks/{slug}/trustees [get]
func (h *TrusteeHandler) ListTrustees(c *gin.Context) {
	items, err := h.service.GetTaskTrustees(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeServiceError(c, err, "could not list trustees")
		return
	}
	out := make([]AssignmentResponse, 0, len(items))
	for i := range items {
		out = append(out, toAssignmentResponse(&items[i], false))
	}
	c.JSON(http.StatusOK, RosterResponse{Trustees: out})
}

func keyRegistration(req RegisterTrusteeKeysRequest, reason string) *trustee.KeyRegistration {
	return &trustee.KeyRegistration{Xpub: req.Xpub, WebCryptoPubkeyHex: req.WebCryptoPubkeyHex, XpubProofSignatureHex: req.XpubProofSignatureHex, WebCryptoProofSignatureHex: req.WebCryptoProofSignatureHex, ProofChallenge: req.ProofChallenge, RotationReason: reason}
}
func toAssignmentResponse(a *trustee.Assignment, includeChallenge bool) AssignmentResponse {
	r := AssignmentResponse{TaskSlug: a.TaskSlug, TrusteeIndex: a.TrusteeIndex, UserID: a.UserID, DisplayName: a.DisplayName, Status: a.Status, NominationMessage: a.NominationMessage, InvitedAt: a.InvitedAt, AcceptedAt: a.AcceptedAt, ActivatedAt: a.ActivatedAt}
	if includeChallenge {
		r.ProofChallenge = a.ProofChallenge
	}
	return r
}
func writeServiceError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, trusteeService.ErrInvalidTrustee), errors.Is(err, trusteeService.ErrInvalidTrusteeIndex), errors.Is(err, trusteeService.ErrInvalidTrusteeKeys):
		apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, err.Error())
	case errors.Is(err, trusteeService.ErrTrusteeForbidden):
		apihttp.WriteError(c, http.StatusForbidden, apihttp.CodeUnauthorized, "task creator access required")
	case errors.Is(err, trustee.ErrTaskNotFound):
		apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "task not found")
	case errors.Is(err, trustee.ErrUserNotFound):
		apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "user not found")
	case errors.Is(err, trusteeService.ErrTrusteeConflict), errors.Is(err, trusteeService.ErrTrusteeState):
		apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, err.Error())
	default:
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, fallback)
	}
}
