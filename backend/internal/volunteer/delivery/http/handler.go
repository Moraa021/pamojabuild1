package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
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
// @Description  Retrieve the volunteer profile attached to the authenticated general account.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  VolunteerProfileResponse
// @Failure      401  {object}  apihttp.ErrorResponse
// @Failure      404  {object}  apihttp.ErrorResponse
// @Failure      500  {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/volunteers/profile [get]
func (h *VolunteerHandler) GetProfile(c *gin.Context) {
	profile, err := h.volunteerService.GetProfile(c.Request.Context(), c.GetInt64("user_id"))
	if err != nil {
		if errors.Is(err, service.ErrProfileNotFound) {
			apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "volunteer profile not found")
		} else {
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not load volunteer profile")
		}
		return
	}
	c.JSON(http.StatusOK, newVolunteerProfileResponse(profile))
}

// UpdateProfile godoc
// @Summary      Update volunteer profile
// @Description  Replace editable profile fields for the authenticated account.
// @Tags         Volunteers
// @Accept       json
// @Produce      json
// @Param        body  body  VolunteerProfileRequest  true  "Profile replacement payload"
// @Success      200   {object}  VolunteerProfileResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/volunteers/profile [put]
func (h *VolunteerHandler) UpdateProfile(c *gin.Context) {
	var req VolunteerProfileRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	userID := c.GetInt64("user_id")
	profile := &volunteer.VolunteerProfile{
		Bio:              req.Bio,
		Skills:           req.Skills,
		LightningAddress: req.LightningAddress,
		OnchainAddress:   req.OnchainAddress,
	}
	if err := h.volunteerService.UpdateProfile(c.Request.Context(), userID, profile); err != nil {
		if errors.Is(err, service.ErrInvalidProfile) {
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, err.Error())
		} else {
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not update volunteer profile")
		}
		return
	}
	updated, err := h.volunteerService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "profile updated but could not be reloaded")
		return
	}
	c.JSON(http.StatusOK, newVolunteerProfileResponse(updated))
}

// ApplyForTask godoc
// @Summary      Apply to participate in a task
// @Description  Submit an application for the authenticated account; volunteer_id is derived from the cookie session.
// @Tags         Volunteers
// @Accept       json
// @Produce      json
// @Param        slug  path    string                  true  "Task slug"
// @Param        body  body    TaskApplicationRequest  true  "Application payload"
// @Success      201   {object}  TaskApplicationResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      404   {object}  apihttp.ErrorResponse
// @Failure      409   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug}/apply [post]
func (h *VolunteerHandler) ApplyForTask(c *gin.Context) {
	var req TaskApplicationRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	app, err := h.applicationService.ApplyForTask(
		c.Request.Context(),
		c.Param("slug"),
		c.GetInt64("user_id"),
		req.Message,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidApplication):
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, "application message and task slug are required")
		case errors.Is(err, service.ErrApplicationTaskNotFound):
			apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "task not found")
		case errors.Is(err, service.ErrAlreadyApplied):
			apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, "an application already exists for this account and task")
		case errors.Is(err, service.ErrRelationshipConflict):
			apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, err.Error())
		default:
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not create task application")
		}
		return
	}
	c.JSON(http.StatusCreated, newTaskApplicationResponse(app))
}

// GetApplications godoc
// @Summary      Get volunteer applications
// @Description  List applications submitted by the authenticated account.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  TaskApplicationListResponse
// @Failure      401  {object}  apihttp.ErrorResponse
// @Failure      500  {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/volunteers/applications [get]
func (h *VolunteerHandler) GetApplications(c *gin.Context) {
	applications, err := h.applicationService.GetApplications(c.Request.Context(), c.GetInt64("user_id"))
	if err != nil {
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not list task applications")
		return
	}
	response := make([]TaskApplicationResponse, 0, len(applications))
	for index := range applications {
		response = append(response, newTaskApplicationResponse(&applications[index]))
	}
	c.JSON(http.StatusOK, TaskApplicationListResponse{Applications: response})
}

// SubmitWork godoc
// @Summary      Submit volunteer work
// @Description  Submit evidence for the authenticated account after its task application has been approved.
// @Tags         Volunteers
// @Accept       json
// @Produce      json
// @Param        slug  path    string                 true  "Task slug"
// @Param        body  body    TaskSubmissionRequest  true  "Submission payload"
// @Success      201   {object}  TaskSubmissionResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      403   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug}/submissions [post]
func (h *VolunteerHandler) SubmitWork(c *gin.Context) {
	var req TaskSubmissionRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	submission, err := h.submissionService.SubmitWork(
		c.Request.Context(),
		c.Param("slug"),
		c.GetInt64("user_id"),
		req.Description,
		req.EvidenceURLs,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidSubmission):
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, err.Error())
		case errors.Is(err, service.ErrNotApproved):
			apihttp.WriteError(c, http.StatusForbidden, apihttp.CodeUnauthorized, "an approved volunteer application is required")
		default:
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not create work submission")
		}
		return
	}
	c.JSON(http.StatusCreated, newTaskSubmissionResponse(submission))
}

// GetSubmissions godoc
// @Summary      Get volunteer submissions
// @Description  List submissions created by the authenticated account.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  TaskSubmissionListResponse
// @Failure      401  {object}  apihttp.ErrorResponse
// @Failure      500  {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/volunteers/submissions [get]
func (h *VolunteerHandler) GetSubmissions(c *gin.Context) {
	submissions, err := h.submissionService.GetSubmissions(c.Request.Context(), c.GetInt64("user_id"))
	if err != nil {
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not list work submissions")
		return
	}
	response := make([]TaskSubmissionResponse, 0, len(submissions))
	for index := range submissions {
		response = append(response, newTaskSubmissionResponse(&submissions[index]))
	}
	c.JSON(http.StatusOK, TaskSubmissionListResponse{Submissions: response})
}

// GetPayments godoc
// @Summary      Get volunteer payment records
// @Description  List payment records belonging to the authenticated account. This remains empty until the later payout workflow writes records.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  VolunteerPaymentListResponse
// @Failure      401  {object}  apihttp.ErrorResponse
// @Failure      500  {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/volunteers/payments [get]
func (h *VolunteerHandler) GetPayments(c *gin.Context) {
	payments, err := h.volunteerService.GetPayments(c.Request.Context(), c.GetInt64("user_id"))
	if err != nil {
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not list volunteer payments")
		return
	}
	response := make([]VolunteerPaymentResponse, 0, len(payments))
	for index := range payments {
		response = append(response, newVolunteerPaymentResponse(&payments[index]))
	}
	c.JSON(http.StatusOK, VolunteerPaymentListResponse{Payments: response})
}

// GetPaymentProfile godoc
// @Summary      Get volunteer payment profile
// @Description  Return saved Lightning and on-chain payout addresses for the authenticated account.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  PaymentProfileResponse
// @Failure      401  {object}  apihttp.ErrorResponse
// @Failure      404  {object}  apihttp.ErrorResponse
// @Failure      500  {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/volunteers/payment-profile [get]
func (h *VolunteerHandler) GetPaymentProfile(c *gin.Context) {
	profile, err := h.volunteerService.GetProfile(c.Request.Context(), c.GetInt64("user_id"))
	if err != nil {
		if errors.Is(err, service.ErrProfileNotFound) {
			apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "volunteer profile not found")
		} else {
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not load payment profile")
		}
		return
	}
	c.JSON(http.StatusOK, newPaymentProfileResponse(profile))
}

// UpdatePaymentProfile godoc
// @Summary      Update volunteer payment profile
// @Description  Replace saved payout addresses without overwriting unrelated volunteer profile fields.
// @Tags         Volunteers
// @Accept       json
// @Produce      json
// @Param        body  body  PaymentProfileRequest  true  "Payment profile payload"
// @Success      200   {object}  PaymentProfileResponse
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/volunteers/payment-profile [put]
func (h *VolunteerHandler) UpdatePaymentProfile(c *gin.Context) {
	var req PaymentProfileRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	userID := c.GetInt64("user_id")
	if err := h.volunteerService.UpdatePaymentProfile(
		c.Request.Context(),
		userID,
		req.LightningAddress,
		req.OnchainAddress,
	); err != nil {
		if errors.Is(err, service.ErrInvalidProfile) {
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, err.Error())
		} else {
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not update payment profile")
		}
		return
	}
	profile, err := h.volunteerService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "payment profile updated but could not be reloaded")
		return
	}
	c.JSON(http.StatusOK, newPaymentProfileResponse(profile))
}

// GetReputation godoc
// @Summary      Get volunteer reputation
// @Description  Get the current reputation summary for the authenticated account.
// @Tags         Volunteers
// @Produce      json
// @Success      200  {object}  ReputationResponse
// @Failure      401  {object}  apihttp.ErrorResponse
// @Failure      500  {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/volunteers/reputation [get]
func (h *VolunteerHandler) GetReputation(c *gin.Context) {
	reputation, err := h.reputationService.CalculateReputation(c.Request.Context(), c.GetInt64("user_id"))
	if err != nil {
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not calculate volunteer reputation")
		return
	}
	c.JSON(http.StatusOK, ReputationResponse{
		UserID:          reputation.UserID,
		Score:           reputation.Score,
		Tier:            reputation.Tier,
		CompletedTasks:  reputation.CompletedTasks,
		TotalEarnedSats: reputation.TotalEarnedSats,
		SuccessRate:     reputation.SuccessRate,
	})
}

func newVolunteerProfileResponse(profile *volunteer.VolunteerProfile) VolunteerProfileResponse {
	skills := append([]string(nil), profile.Skills...)
	if skills == nil {
		skills = []string{}
	}
	return VolunteerProfileResponse{
		UserID:           profile.UserID,
		DisplayName:      profile.DisplayName,
		Bio:              profile.Bio,
		Skills:           skills,
		LightningAddress: profile.LightningAddress,
		OnchainAddress:   profile.OnchainAddress,
		ReputationScore:  profile.ReputationScore,
		Tier:             profile.Tier,
		CompletedTasks:   profile.CompletedTasks,
		TotalEarnedSats:  profile.TotalEarnedSats,
		CreatedAt:        profile.CreatedAt,
		UpdatedAt:        profile.UpdatedAt,
	}
}

func newTaskApplicationResponse(application *volunteer.TaskApplication) TaskApplicationResponse {
	return TaskApplicationResponse{
		ID:          application.ID,
		TaskSlug:    application.TaskSlug,
		VolunteerID: application.VolunteerID,
		Message:     application.Message,
		Status:      application.Status,
		AppliedAt:   application.AppliedAt,
		ReviewedAt:  application.ReviewedAt,
	}
}

func newTaskSubmissionResponse(submission *volunteer.TaskSubmission) TaskSubmissionResponse {
	evidenceURLs := append([]string(nil), submission.EvidenceURLs...)
	if evidenceURLs == nil {
		evidenceURLs = []string{}
	}
	return TaskSubmissionResponse{
		ID:           submission.ID,
		TaskSlug:     submission.TaskSlug,
		VolunteerID:  submission.VolunteerID,
		Description:  submission.Description,
		EvidenceURLs: evidenceURLs,
		Status:       submission.Status,
		SubmittedAt:  submission.SubmittedAt,
		ReviewedAt:   submission.ReviewedAt,
	}
}

func newVolunteerPaymentResponse(payment *volunteer.Payment) VolunteerPaymentResponse {
	return VolunteerPaymentResponse{
		ID:              payment.ID,
		TaskSlug:        payment.TaskSlug,
		AmountSats:      payment.AmountSats,
		PaymentMethod:   payment.PaymentMethod,
		Status:          payment.Status,
		TransactionHash: payment.TransactionHash,
		PaidAt:          payment.PaidAt,
	}
}

func newPaymentProfileResponse(profile *volunteer.VolunteerProfile) PaymentProfileResponse {
	return PaymentProfileResponse{
		LightningAddress: profile.LightningAddress,
		OnchainAddress:   profile.OnchainAddress,
	}
}
