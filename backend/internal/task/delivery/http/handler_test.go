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
	created *task.Task
	list    *task.ListResult
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
func (s stubTaskService) TransitionVolunteerStatus(context.Context, string, string) error {
	return nil
}
func (s stubTaskService) TransitionFinancialState(context.Context, string, string) error {
	return nil
}

func taskHandlerRouter(handler *TaskHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/tasks", func(c *gin.Context) {
		c.Set("user_id", int64(42))
		handler.CreateTask(c)
	})
	router.GET("/tasks", handler.ListTasks)
	return router
}

func TestCreateTaskReturnsExplicitSnakeCaseDTO(t *testing.T) {
	createdAt := time.Now().UTC()
	router := taskHandlerRouter(NewTaskHandler(stubTaskService{created: &task.Task{
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
	router := taskHandlerRouter(NewTaskHandler(stubTaskService{}))
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
	router := taskHandlerRouter(NewTaskHandler(stubTaskService{list: &task.ListResult{
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
