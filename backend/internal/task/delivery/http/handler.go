package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/task"
	taskService "pamojabuild1/backend/internal/task/service"
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
// @Failure      400   {object}  apihttp.ErrorResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      409   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	creatorID := c.GetInt64("user_id")
	if creatorID <= 0 {
		apihttp.WriteError(c, http.StatusUnauthorized, apihttp.CodeUnauthenticated, "valid session cookie required")
		return
	}

	var req CreateTaskRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}

	t := &task.Task{
		// Ownership comes from the verified token. Accepting creator_id from
		// JSON would let a caller create campaigns in another user's name.
		CreatorID:      creatorID,
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
		switch {
		case errors.Is(err, taskService.ErrInvalidTask):
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, err.Error())
		case errors.Is(err, taskService.ErrTaskConflict):
			apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, err.Error())
		default:
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not create task")
		}
		return
	}

	c.JSON(http.StatusCreated, newTaskResponse(created))
}

// GetTask godoc
// @Summary      Get task details
// @Description  Retrieve detailed information for a specific task by slug.
// @Tags         Tasks
// @Produce      json
// @Param        slug  path      string  true  "Task slug"
// @Success      200   {object}  TaskResponse
// @Failure      401   {object}  apihttp.ErrorResponse
// @Failure      404   {object}  apihttp.ErrorResponse
// @Failure      500   {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	slug := c.Param("slug")

	t, err := h.service.GetTask(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, taskService.ErrTaskNotFound) {
			apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "task not found")
		} else {
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not load task")
		}
		return
	}

	c.JSON(http.StatusOK, newTaskResponse(t))
}

// ListTasks godoc
// @Summary      List tasks
// @Description  List tasks with optional category, region, and status filters.
// @Tags         Tasks
// @Produce      json
// @Param        category  query     string  false  "Task category"
// @Param        region    query     string  false  "Task region"
// @Param        status    query     string  false  "Task status"
// @Param        page      query     int     false  "Page number (default 1)" minimum(1)
// @Param        page_size query     int     false  "Items per page (default 20, maximum 100)" minimum(1) maximum(100)
// @Success      200       {object}  TaskListResponse
// @Failure      400       {object}  apihttp.ErrorResponse
// @Failure      401       {object}  apihttp.ErrorResponse
// @Failure      500       {object}  apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks [get]
func (h *TaskHandler) ListTasks(c *gin.Context) {
	page, ok := queryPositiveInt(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := queryPositiveInt(c, "page_size", 20)
	if !ok {
		return
	}

	result, err := h.service.ListTasks(c.Request.Context(), task.ListOptions{
		Category: c.Query("category"),
		Region:   c.Query("region"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		if errors.Is(err, taskService.ErrInvalidTaskFilter) {
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, "invalid task list filters or pagination")
		} else {
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not list tasks")
		}
		return
	}

	tasks := make([]TaskResponse, 0, len(result.Tasks))
	for i := range result.Tasks {
		tasks = append(tasks, newTaskResponse(&result.Tasks[i]))
	}
	totalPages := 0
	if result.Total > 0 {
		totalPages = (result.Total + pageSize - 1) / pageSize
	}
	c.JSON(http.StatusOK, TaskListResponse{
		Tasks: tasks,
		Pagination: apihttp.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: result.Total,
			TotalPages: totalPages,
		},
	})
}

// StartTask godoc
// @Summary      Start task work
// @Description  Creator-only transition from open to in_progress. At least one approved volunteer is required.
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        slug             path    string              true  "Task slug"
// @Param        Idempotency-Key  header  string              true  "Unique retry key (maximum 128 characters)"
// @Param        body             body    StateActionRequest  true  "Optional audit reason; send an empty object when omitted"
// @Success      200              {object} StateActionResponse
// @Failure      400              {object} apihttp.ErrorResponse
// @Failure      401              {object} apihttp.ErrorResponse
// @Failure      403              {object} apihttp.ErrorResponse
// @Failure      404              {object} apihttp.ErrorResponse
// @Failure      409              {object} apihttp.ErrorResponse
// @Failure      500              {object} apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug}/start [post]
func (h *TaskHandler) StartTask(c *gin.Context) {
	h.runStateAction(c, h.service.StartTask)
}

// SubmitForVerification godoc
// @Summary      Submit task for verification
// @Description  Creator-only transition from in_progress to pending_verification after every approved volunteer has submitted work.
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        slug             path    string              true  "Task slug"
// @Param        Idempotency-Key  header  string              true  "Unique retry key (maximum 128 characters)"
// @Param        body             body    StateActionRequest  true  "Optional audit reason; send an empty object when omitted"
// @Success      200              {object} StateActionResponse
// @Failure      400              {object} apihttp.ErrorResponse
// @Failure      401              {object} apihttp.ErrorResponse
// @Failure      403              {object} apihttp.ErrorResponse
// @Failure      404              {object} apihttp.ErrorResponse
// @Failure      409              {object} apihttp.ErrorResponse
// @Failure      500              {object} apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug}/submit-for-verification [post]
func (h *TaskHandler) SubmitForVerification(c *gin.Context) {
	h.runStateAction(c, h.service.SubmitForVerification)
}

// VerifyTask godoc
// @Summary      Verify completed task work
// @Description  An independent task trustee verifies pending work. The atomic transition completes work and closes donations by moving the financial state to LIQUIDATING.
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        slug             path    string              true  "Task slug"
// @Param        Idempotency-Key  header  string              true  "Unique retry key (maximum 118 characters)"
// @Param        body             body    StateActionRequest  true  "Optional audit reason; send an empty object when omitted"
// @Success      200              {object} StateActionResponse
// @Failure      400              {object} apihttp.ErrorResponse
// @Failure      401              {object} apihttp.ErrorResponse
// @Failure      403              {object} apihttp.ErrorResponse
// @Failure      404              {object} apihttp.ErrorResponse
// @Failure      409              {object} apihttp.ErrorResponse
// @Failure      500              {object} apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug}/verify [post]
func (h *TaskHandler) VerifyTask(c *gin.Context) {
	h.runStateAction(c, h.service.VerifyTask)
}

type stateAction func(context.Context, string, int64, string, string) (*task.StateTransitionResult, error)

func (h *TaskHandler) runStateAction(c *gin.Context, action stateAction) {
	actorUserID := c.GetInt64("user_id")
	if actorUserID <= 0 {
		apihttp.WriteError(c, http.StatusUnauthorized, apihttp.CodeUnauthenticated, "valid session cookie required")
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		apihttp.WriteValidationError(c, map[string]string{"idempotency_key": "Idempotency-Key header is required"})
		return
	}
	var req StateActionRequest
	if !apihttp.BindJSON(c, &req) {
		return
	}
	result, err := action(c.Request.Context(), c.Param("slug"), actorUserID, req.Reason, idempotencyKey)
	if err != nil {
		writeStateActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, newStateActionResponse(result))
}

// ListStateHistory godoc
// @Summary      List task state history
// @Description  Returns the immutable work and financial transition audit history, newest first.
// @Tags         Tasks
// @Produce      json
// @Param        slug       path   string true  "Task slug"
// @Param        state_kind query  string false "State machine filter" Enums(work, financial)
// @Param        page       query  int    false "Page number (default 1)" minimum(1)
// @Param        page_size  query  int    false "Items per page (default 20, maximum 100)" minimum(1) maximum(100)
// @Success      200        {object} StateHistoryResponse
// @Failure      400        {object} apihttp.ErrorResponse
// @Failure      401        {object} apihttp.ErrorResponse
// @Failure      404        {object} apihttp.ErrorResponse
// @Failure      500        {object} apihttp.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/tasks/{slug}/state-history [get]
func (h *TaskHandler) ListStateHistory(c *gin.Context) {
	page, ok := queryPositiveInt(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := queryPositiveInt(c, "page_size", 20)
	if !ok {
		return
	}
	result, err := h.service.ListStateTransitions(c.Request.Context(), c.Param("slug"), task.StateHistoryOptions{
		StateKind: c.Query("state_kind"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		switch {
		case errors.Is(err, taskService.ErrInvalidStateHistory):
			apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, "invalid state history filters or pagination")
		case errors.Is(err, taskService.ErrTaskNotFound):
			apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "task not found")
		default:
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not load task state history")
		}
		return
	}
	totalPages := 0
	if result.Total > 0 {
		totalPages = (result.Total + pageSize - 1) / pageSize
	}
	c.JSON(http.StatusOK, StateHistoryResponse{
		Transitions: newTransitionResponses(result.Transitions),
		Pagination: apihttp.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: result.Total,
			TotalPages: totalPages,
		},
	})
}

func writeStateActionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, taskService.ErrInvalidStateAction):
		apihttp.WriteError(c, http.StatusBadRequest, apihttp.CodeValidation, "invalid state action")
	case errors.Is(err, taskService.ErrStateActionForbidden),
		errors.Is(err, taskService.ErrIndependentVerifier):
		apihttp.WriteError(c, http.StatusForbidden, apihttp.CodeUnauthorized, "account is not allowed to perform this state action")
	case errors.Is(err, taskService.ErrTaskNotFound):
		apihttp.WriteError(c, http.StatusNotFound, apihttp.CodeNotFound, "task not found")
	case errors.Is(err, taskService.ErrStateConflict):
		apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, "task state changed or action was already completed differently")
	case errors.Is(err, taskService.ErrNoApprovedVolunteers):
		apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, "task requires at least one approved volunteer")
	case errors.Is(err, taskService.ErrSubmissionsIncomplete):
		apihttp.WriteError(c, http.StatusConflict, apihttp.CodeConflict, "all approved volunteers must submit work before verification")
	default:
		apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "could not transition task state")
	}
}

func queryPositiveInt(c *gin.Context, name string, defaultValue int) (int, bool) {
	value := c.Query(name)
	if value == "" {
		return defaultValue, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		apihttp.WriteValidationError(c, map[string]string{name: "must be a positive integer"})
		return 0, false
	}
	return parsed, true
}

func newTaskResponse(t *task.Task) TaskResponse {
	return TaskResponse{
		ID:                    t.ID,
		Slug:                  t.Slug,
		CreatorID:             t.CreatorID,
		Title:                 t.Title,
		Description:           t.Description,
		Category:              t.Category,
		Region:                t.Region,
		LocationDetail:        t.LocationDetail,
		Status:                t.Status,
		FinancialState:        t.FinancialState,
		WorkStateVersion:      t.WorkStateVersion,
		FinancialStateVersion: t.FinancialStateVersion,
		GoalSats:              t.GoalSats,
		MaxVolunteers:         t.MaxVolunteers,
		VolunteerMode:         t.VolunteerMode,
		ImagePath:             t.ImagePath,
		CreatedAt:             t.CreatedAt,
	}
}

func newStateActionResponse(result *task.StateTransitionResult) StateActionResponse {
	return StateActionResponse{
		Transitions: newTransitionResponses(result.Transitions),
		Replayed:    result.Replayed,
	}
}

func newTransitionResponses(transitions []task.StateTransition) []StateTransitionResponse {
	response := make([]StateTransitionResponse, 0, len(transitions))
	for _, transition := range transitions {
		response = append(response, StateTransitionResponse{
			ID:          transition.ID,
			StateKind:   transition.StateKind,
			FromState:   transition.FromState,
			ToState:     transition.ToState,
			Version:     transition.Version,
			ActorUserID: transition.ActorUserID,
			Reason:      transition.Reason,
			CreatedAt:   transition.CreatedAt,
		})
	}
	return response
}
