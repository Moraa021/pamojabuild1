package http

import (
	"time"

	"pamojabuild1/backend/internal/apihttp"
)

type CreateTaskRequest struct {
	Title          string `json:"title" binding:"required,max=255"`
	Description    string `json:"description" binding:"required"`
	Category       string `json:"category" binding:"required,max=100"`
	Region         string `json:"region" binding:"required,max=100"`
	LocationDetail string `json:"location_detail,omitempty" binding:"max=255"`
	GoalSats       int64  `json:"goal_sats,omitempty"`
	MaxVolunteers  int64  `json:"max_volunteers" binding:"min=0"`
	VolunteerMode  string `json:"volunteer_mode" binding:"required,oneof=open approval_required"`
}

type TaskListResponse struct {
	Tasks      []TaskResponse             `json:"tasks"`
	Pagination apihttp.PaginationResponse `json:"pagination"`
}

type TaskResponse struct {
	ID                    int64     `json:"id"`
	Slug                  string    `json:"slug"`
	CreatorID             int64     `json:"creator_id"`
	Title                 string    `json:"title"`
	Description           string    `json:"description"`
	Category              string    `json:"category"`
	Region                string    `json:"region"`
	LocationDetail        string    `json:"location_detail,omitempty"`
	Status                string    `json:"status"`          // "open", "in_progress", "pending_verification", "completed"
	FinancialState        string    `json:"financial_state"` // "ACTIVE", "LIQUIDATING", "READY_FOR_PAYOUT", "PAYOUT_PROCESSING", "ARCHIVED", "SYSTEM_LOCKDOWN"
	WorkStateVersion      int64     `json:"work_state_version"`
	FinancialStateVersion int64     `json:"financial_state_version"`
	GoalSats              int64     `json:"goal_sats,omitempty"`
	MaxVolunteers         int64     `json:"max_volunteers"`
	VolunteerMode         string    `json:"volunteer_mode"`
	ImagePath             string    `json:"image_path,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

type StateActionRequest struct {
	Reason string `json:"reason,omitempty" binding:"max=500"`
}

type StateTransitionResponse struct {
	ID          int64     `json:"id"`
	StateKind   string    `json:"state_kind"`
	FromState   *string   `json:"from_state"`
	ToState     string    `json:"to_state"`
	Version     int64     `json:"version"`
	ActorUserID *int64    `json:"actor_user_id,omitempty"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"created_at"`
}

type StateActionResponse struct {
	Transitions []StateTransitionResponse `json:"transitions"`
	Replayed    bool                      `json:"replayed"`
}

type StateHistoryResponse struct {
	Transitions []StateTransitionResponse  `json:"transitions"`
	Pagination  apihttp.PaginationResponse `json:"pagination"`
}
