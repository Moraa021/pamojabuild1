package http

import "time"

type VolunteerProfileRequest struct {
	Bio              string   `json:"bio" binding:"max=2000"`
	Skills           []string `json:"skills" binding:"max=50,dive,min=1,max=100"`
	LightningAddress string   `json:"lightning_address" binding:"max=255"`
	OnchainAddress   string   `json:"onchain_address" binding:"max=255"`
}

type VolunteerProfileResponse struct {
	UserID           int64     `json:"user_id"`
	DisplayName      string    `json:"display_name"`
	Bio              string    `json:"bio"`
	Skills           []string  `json:"skills"`
	LightningAddress string    `json:"lightning_address"`
	OnchainAddress   string    `json:"onchain_address"`
	ReputationScore  int       `json:"reputation_score"`
	Tier             string    `json:"tier"`
	CompletedTasks   int       `json:"completed_tasks"`
	TotalEarnedSats  int64     `json:"total_earned_sats"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type TaskApplicationRequest struct {
	Message string `json:"message" binding:"required,max=2000"`
}

type TaskApplicationResponse struct {
	ID          int64      `json:"id"`
	TaskSlug    string     `json:"task_slug"`
	VolunteerID int64      `json:"volunteer_id"`
	Message     string     `json:"message"`
	Status      string     `json:"status"` // "pending", "approved", "rejected"
	AppliedAt   time.Time  `json:"applied_at"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
}

type TaskSubmissionRequest struct {
	Description  string   `json:"description" binding:"required,max=5000"`
	EvidenceURLs []string `json:"evidence_urls" binding:"required,min=1,max=20,dive,required,max=2048"`
}

type TaskSubmissionResponse struct {
	ID           int64      `json:"id"`
	TaskSlug     string     `json:"task_slug"`
	VolunteerID  int64      `json:"volunteer_id"`
	Description  string     `json:"description"`
	EvidenceURLs []string   `json:"evidence_urls"`
	Status       string     `json:"status"`
	SubmittedAt  time.Time  `json:"submitted_at"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
}

type VolunteerPaymentResponse struct {
	ID              int64      `json:"id"`
	TaskSlug        string     `json:"task_slug"`
	AmountSats      int64      `json:"amount_sats"`
	PaymentMethod   string     `json:"payment_method"` // "lightning", "onchain"
	Status          string     `json:"status"`
	TransactionHash string     `json:"transaction_hash,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
}

type PaymentProfileRequest struct {
	LightningAddress string `json:"lightning_address" binding:"max=255"`
	OnchainAddress   string `json:"onchain_address" binding:"max=255"`
}

type PaymentProfileResponse struct {
	LightningAddress string `json:"lightning_address"`
	OnchainAddress   string `json:"onchain_address"`
}

type ReputationResponse struct {
	UserID          int64   `json:"user_id"`
	Score           int     `json:"score"`
	Tier            string  `json:"tier"`
	CompletedTasks  int     `json:"completed_tasks"`
	TotalEarnedSats int64   `json:"total_earned_sats"`
	SuccessRate     float64 `json:"success_rate"`
}

type TaskApplicationListResponse struct {
	Applications []TaskApplicationResponse `json:"applications"`
}

type TaskSubmissionListResponse struct {
	Submissions []TaskSubmissionResponse `json:"submissions"`
}

type VolunteerPaymentListResponse struct {
	Payments []VolunteerPaymentResponse `json:"payments"`
}
