package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/volunteer"
	"pamojabuild1/backend/internal/volunteer/service"
)

type VolunteerHandler struct {
	volunteerService   *service.VolunteerService
	applicationService *service.ApplicationService
	submissionService  *service.SubmissionService
	reputationService  *service.ReputationService
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

// GetProfile godoc
// @Summary      Get volunteer profile
// @Description  Retrieve the current volunteer profile for the authenticated user.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  VolunteerProfileResponse
// @Failure      404  {object}  map[string]string
// @Router       /api/v1/volunteers/profile [get]
func (h *VolunteerHandler) GetProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")

	profile, err := h.volunteerService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile godoc
// @Summary      Update volunteer profile
// @Description  Update the authenticated volunteer's profile data.
// @Tags         Volunteers
// @Accept       json
// @Produce      json
// @Param        body  body  VolunteerProfileRequest  true  "Profile update payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Router       /api/v1/volunteers/profile [put]
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

// ApplyForTask godoc
// @Summary      Apply to participate in a task
// @Description  Submit a volunteer application for a task.
// @Tags         Volunteers
// @Accept       json
// @Produce      json
// @Param        slug  path    string                  true  "Task slug"
// @Param        body  body    TaskApplicationRequest  true  "Application payload"
// @Success      201   {object}  TaskApplicationResponse
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Router       /api/v1/tasks/{slug}/apply [post]
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

// GetApplications godoc
// @Summary      Get volunteer applications
// @Description  List all applications submitted by the authenticated volunteer.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  map[string][]TaskApplicationResponse
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/volunteers/applications [get]
func (h *VolunteerHandler) GetApplications(c *gin.Context) {
	userID := c.GetInt64("user_id")

	applications, err := h.applicationService.GetApplications(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"applications": applications})
}

// SubmitWork godoc
// @Summary      Submit volunteer work
// @Description  Submit completed work for a specific task.
// @Tags         Volunteers
// @Accept       json
// @Produce      json
// @Param        slug  path    string                 true  "Task slug"
// @Param        body  body    TaskSubmissionRequest  true  "Submission payload"
// @Success      201   {object}  TaskSubmissionResponse
// @Failure      400   {object}  map[string]string
// @Router       /api/v1/tasks/{slug}/submissions [post]
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

// GetSubmissions godoc
// @Summary      Get volunteer submissions
// @Description  List submissions created by the authenticated volunteer.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  map[string][]TaskSubmissionResponse
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/volunteers/submissions [get]
func (h *VolunteerHandler) GetSubmissions(c *gin.Context) {
	userID := c.GetInt64("user_id")

	submissions, err := h.submissionService.GetSubmissions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"submissions": submissions})
}

// GetPayments godoc
// @Summary      Get volunteer earnings summary
// @Description  Retrieve the authenticated volunteer's earned sats and completed task count.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  VolunteerPaymentsSummaryResponse
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/volunteers/payments [get]
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

// UpdatePaymentProfile godoc
// @Summary      Update volunteer payment profile
// @Description  Update the volunteer's preferred Lightning or on-chain payment address.
// @Tags         Volunteers
// @Accept       json
// @Produce      json
// @Param        body  body  PaymentProfileRequest  true  "Payment profile payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Router       /api/v1/volunteers/payment-profile [put]
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

// GetReputation godoc
// @Summary      Get volunteer reputation
// @Description  Get the reputation score and tier for the authenticated volunteer.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  ReputationResponse
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/volunteers/reputation [get]
func (h *VolunteerHandler) GetReputation(c *gin.Context) {
	userID := c.GetInt64("user_id")

	reputation, err := h.reputationService.CalculateReputation(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reputation)
}
