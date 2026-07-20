package http

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
	"pamojabuild1/backend/internal/task"
)

type stubTaskService struct {
	created     *task.Task
	list        *task.ListResult
	stateResult *task.StateTransitionResult
	stateErr    error
	actorID     *int64
}

func (s stubTaskService) CreateCampaign(context.Context, *task.Task) (*task.Task, error) {
	return s.created, nil
}
func (s stubTaskService) GetTask(context.Context, string) (*task.Task, error) {
	return s.created, nil
}
func (s stubTaskService) ListTasks(context.Context, task.ListOptions) (*task.ListResult, error) {
	return s.list, nil
}
func (s *stubTaskService) StartTask(_ context.Context, _ string, actorID int64, _, _ string) (*task.StateTransitionResult, error) {
	if s.actorID != nil {
		*s.actorID = actorID
	}
	return s.stateResult, s.stateErr
}
func (s *stubTaskService) SubmitForVerification(context.Context, string, int64, string, string) (*task.StateTransitionResult, error) {
	return &task.StateTransitionResult{}, nil
}
func (s *stubTaskService) VerifyTask(context.Context, string, int64, string, string) (*task.StateTransitionResult, error) {
	return &task.StateTransitionResult{}, nil
}
func (s *stubTaskService) AdvanceFinancialState(context.Context, string, string, string, string) (*task.StateTransitionResult, error) {
	return &task.StateTransitionResult{}, nil
}
func (s *stubTaskService) ListStateTransitions(context.Context, string, task.StateHistoryOptions) (*task.StateHistoryResult, error) {
	return &task.StateHistoryResult{}, nil
}

func taskHandlerRouter(handler *TaskHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/tasks", func(c *gin.Context) {
		c.Set("user_id", int64(42))
		handler.CreateTask(c)
	})
	router.GET("/tasks", handler.ListTasks)
	router.POST("/tasks/:slug/start", func(c *gin.Context) {
		c.Set("user_id", int64(42))
		handler.StartTask(c)
	})
	return router
}

func TestCreateTaskReturnsExplicitSnakeCaseDTO(t *testing.T) {
	createdAt := time.Now().UTC()
	router := taskHandlerRouter(NewTaskHandler(&stubTaskService{created: &task.Task{
		ID:             7,
		Slug:           "clean-water",
		CreatorID:      42,
		Title:          "Clean Water",
		Status:         "open",
		FinancialState: "ACTIVE",
		CreatedAt:      createdAt,
	}}))
	request := httptest.NewRequest(nethttp.MethodPost, "/tasks", strings.NewReader(
		`{"title":"Clean Water","description":"Repair pump","category":"community","region":"Kisumu","max_volunteers":2,"volunteer_mode":"open"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["creator_id"] != float64(42) || body["financial_state"] != "ACTIVE" {
		t.Fatalf("unexpected task response: %#v", body)
	}
	if _, leaked := body["CreatorID"]; leaked {
		t.Fatal("domain struct field names must not leak into JSON")
	}
}

func TestCreateTaskRejectsClientSelectedCreatorID(t *testing.T) {
	router := taskHandlerRouter(NewTaskHandler(&stubTaskService{}))
	request := httptest.NewRequest(nethttp.MethodPost, "/tasks", strings.NewReader(
		`{"creator_id":99,"title":"Clean Water","description":"Repair pump","category":"community","region":"Kisumu","max_volunteers":2,"volunteer_mode":"open"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
	var body apihttp.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != apihttp.CodeValidation || body.Fields["creator_id"] != "is not allowed" {
		t.Fatalf("unexpected error response: %#v", body)
	}
}

func TestListTasksReturnsPaginationMetadata(t *testing.T) {
	router := taskHandlerRouter(NewTaskHandler(&stubTaskService{list: &task.ListResult{
		Tasks: []task.Task{},
		Total: 45,
	}}))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(nethttp.MethodGet, "/tasks?page=2&page_size=20", nil))

	if response.Code != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body TaskListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Pagination.Page != 2 || body.Pagination.TotalPages != 3 || body.Tasks == nil {
		t.Fatalf("unexpected list response: %#v", body)
	}
}

func TestStartTaskRequiresIdempotencyHeader(t *testing.T) {
	router := taskHandlerRouter(NewTaskHandler(&stubTaskService{}))
	request := httptest.NewRequest(nethttp.MethodPost, "/tasks/community-garden/start", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	var body apihttp.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != apihttp.CodeValidation || body.Fields["idempotency_key"] == "" {
		t.Fatalf("unexpected missing header response: %#v", body)
	}
}

func TestStartTaskUsesSessionActorAndReturnsReplayMetadata(t *testing.T) {
	var capturedActor int64
	service := &stubTaskService{
		actorID: &capturedActor,
		stateResult: &task.StateTransitionResult{
			Replayed: true,
			Transitions: []task.StateTransition{{
				ID:        8,
				StateKind: task.StateKindWork,
				ToState:   task.WorkStateInProgress,
				Version:   2,
			}},
		},
	}
	router := taskHandlerRouter(NewTaskHandler(service))
	request := httptest.NewRequest(nethttp.MethodPost, "/tasks/community-garden/start", strings.NewReader(`{"reason":"ready"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "frontend-retry-1")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if capturedActor != 42 {
		t.Fatalf("expected session actor 42, got %d", capturedActor)
	}
	var body StateActionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Replayed || len(body.Transitions) != 1 || body.Transitions[0].ToState != task.WorkStateInProgress {
		t.Fatalf("unexpected action response: %#v", body)
	}
}
