package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/task"
)

type TaskHandler struct {
	service task.Service
}

func NewTaskHandler(service task.Service) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t := &task.Task{
		CreatorID:      req.CreatorID,
		Title:          req.Title,
		Description:    req.Description,
		Category:       req.Category,
		Region:         req.Region,
		LocationDetail: req.LocationDetail,
		GoalSats:       req.GoalSats,
		MaxVolunteers:  req.MaxVolunteers,
		VolunteerMode:  req.VolunteerMode,
	}

	created, err := h.service.CreateCampaign(c.Request.Context(), t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	slug := c.Param("slug")

	t, err := h.service.GetTask(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, t)
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	category := c.Query("category")
	region := c.Query("region")
	status := c.Query("status")

	tasks, err := h.service.ListTasks(c.Request.Context(), category, region, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (h *TaskHandler) UpdateTaskStatus(c *gin.Context) {
	slug := c.Param("slug")
	
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.TransitionVolunteerStatus(c.Request.Context(), slug, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}