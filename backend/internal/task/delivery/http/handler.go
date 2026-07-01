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

// CreateTask godoc
// @Summary      Create a new task campaign
// @Description  Create a new task with goal, category, region, and volunteer settings.
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        body  body      CreateTaskRequest  true  "Task creation payload"
// @Success      201   {object}  TaskResponse
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/tasks [post]
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

// GetTask godoc
// @Summary      Get task details
// @Description  Retrieve detailed information for a specific task by slug.
// @Tags         Tasks
// @Produce      json
// @Param        slug  path      string  true  "Task slug"
// @Success      200   {object}  TaskResponse
// @Failure      404   {object}  map[string]string
// @Router       /api/v1/tasks/{slug} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	slug := c.Param("slug")

	t, err := h.service.GetTask(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, t)
}

// ListTasks godoc
// @Summary      List tasks
// @Description  List tasks with optional category, region, and status filters.
// @Tags         Tasks
// @Produce      json
// @Param        category  query     string  false  "Task category"
// @Param        region    query     string  false  "Task region"
// @Param        status    query     string  false  "Task status"
// @Success      200       {object}  map[string][]TaskResponse
// @Failure      500       {object}  map[string]string
// @Router       /api/v1/tasks [get]
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

// UpdateTaskStatus godoc
// @Summary      Update task status
// @Description  Transition the status of an existing task.
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        slug   path      string  true  "Task slug"
// @Param        body   body      object  true  "Status update payload"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  map[string]string
// @Failure      500    {object}  map[string]string
// @Router       /api/v1/tasks/{slug}/status [put]
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
