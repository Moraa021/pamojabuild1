package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/volunteer"
)

type VolunteerHandler struct {
	service volunteer.Service
}

func NewVolunteerHandler(service volunteer.Service) *VolunteerHandler {
	return &VolunteerHandler{service: service}
}

func (h *VolunteerHandler) GetProfile(c *gin.Context) {
	userID := c.GetInt64("user_id") // from auth middleware
	
	profile, err := h.service.GetProfile(c.Request.Context(), userID)
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

	if err := h.service.UpdateProfile(c.Request.Context(), userID, profile); err != nil {
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

	app, err := h.service.ApplyForTask(c.Request.Context(), taskSlug, userID, req.Message)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, app)
}

func (h *VolunteerHandler) GetApplications(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	// Note: This would need to be added to the service interface
	// For now, we'll just return a placeholder
	c.JSON(http.StatusOK, gin.H{"applications": []interface{}{}})
}

func (h *VolunteerHandler) SubmitWork(c *gin.Context) {
	taskSlug := c.Param("slug")
	userID := c.GetInt64("user_id")
	
	var req TaskSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.service.SubmitWork(c.Request.Context(), taskSlug, userID, req.Description, req.EvidenceURLs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

func (h *VolunteerHandler) GetPayments(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	payments, err := h.service.GetPayments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

func (h *VolunteerHandler) UpdatePaymentProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	
	var req PaymentProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the volunteer profile with payment info
	profile := &volunteer.VolunteerProfile{
		LightningAddress: req.LightningAddress,
		OnchainAddress:   req.OnchainAddress,
	}

	if err := h.service.UpdateProfile(c.Request.Context(), userID, profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment profile updated"})
}