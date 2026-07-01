package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/volunteer"
	"pamojabuild1/backend/internal/volunteer/service"
)

type VolunteerHandler struct {
	volunteerService    *service.VolunteerService
	applicationService  *service.ApplicationService
	submissionService   *service.SubmissionService
	reputationService   *service.ReputationService
}

func NewVolunteerHandler(
	volunteerService *service.VolunteerService,
	applicationService *service.ApplicationService,
	submissionService *service.SubmissionService,
	reputationService *service.ReputationService,
) *VolunteerHandler {
	return &VolunteerHandler{
		volunteerService:   volunteerService,
		applicationService: applicationService,
		submissionService:  submissionService,
		reputationService:  reputationService,
	}
}

func (h *VolunteerHandler) GetProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	profile, err := h.volunteerService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *VolunteerHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	var req VolunteerProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile := &volunteer.VolunteerProfile{
		Bio:              req.Bio,
		Skills:           req.Skills,
		LightningAddress: req.LightningAddress,
		OnchainAddress:   req.OnchainAddress,
	}

	if err := h.volunteerService.UpdateProfile(c.Request.Context(), userID, profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}

func (h *VolunteerHandler) ApplyForTask(c *gin.Context) {
	taskSlug := c.Param("slug")
	userID := c.GetInt64("user_id")
	
	var req TaskApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, err := h.applicationService.ApplyForTask(c.Request.Context(), taskSlug, userID, req.Message)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, app)
}

func (h *VolunteerHandler) GetApplications(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	applications, err := h.applicationService.GetApplications(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"applications": applications})
}

func (h *VolunteerHandler) SubmitWork(c *gin.Context) {
	taskSlug := c.Param("slug")
	userID := c.GetInt64("user_id")
	
	var req TaskSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.submissionService.SubmitWork(c.Request.Context(), taskSlug, userID, req.Description, req.EvidenceURLs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

func (h *VolunteerHandler) GetSubmissions(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	submissions, err := h.submissionService.GetSubmissions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"submissions": submissions})
}

func (h *VolunteerHandler) GetPayments(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	profile, err := h.volunteerService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Payments are tracked through the ledger/escrow systems
	c.JSON(http.StatusOK, gin.H{
		"total_earned_sats": profile.TotalEarnedSats,
		"completed_tasks":   profile.CompletedTasks,
	})
}

func (h *VolunteerHandler) UpdatePaymentProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	var req PaymentProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile := &volunteer.VolunteerProfile{
		LightningAddress: req.LightningAddress,
		OnchainAddress:   req.OnchainAddress,
	}

	if err := h.volunteerService.UpdateProfile(c.Request.Context(), userID, profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment profile updated"})
}

func (h *VolunteerHandler) GetReputation(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	reputation, err := h.reputationService.CalculateReputation(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reputation)
}