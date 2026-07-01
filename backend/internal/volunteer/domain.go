package volunteer

import (
	"context"
	"time"
)

type VolunteerProfile struct {
	UserID           int64
	Bio              string
	Skills           []string
	LightningAddress string
	OnchainAddress   string
	ReputationScore  int
	Tier             string
	CompletedTasks   int
	TotalEarnedSats  int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TaskApplication struct {
	ID          int64
	TaskSlug    string
	VolunteerID int64
	Message     string
	Status      string
	AppliedAt   time.Time
	ReviewedAt  *time.Time
}

type TaskSubmission struct {
	ID           int64
	TaskSlug     string
	VolunteerID  int64
	Description  string
	EvidenceURLs []string
	Status       string
	SubmittedAt  time.Time
	ReviewedAt   *time.Time
}

type Payment struct {
	ID              int64
	TaskSlug        string
	VolunteerID     int64
	AmountSats      int64
	PaymentMethod   string
	Status          string
	TransactionHash string
	PaidAt          *time.Time
}

type ReputationResponse struct {
	UserID          int64   `json:"user_id"`
	Score           int     `json:"score"`
	Tier            string  `json:"tier"`
	CompletedTasks  int     `json:"completed_tasks"`
	TotalEarnedSats int64   `json:"total_earned_sats"`
	SuccessRate     float64 `json:"success_rate"`
}

type ProfileRepository interface {
	Create(ctx context.Context, profile *VolunteerProfile) error
	GetByUserID(ctx context.Context, userID int64) (*VolunteerProfile, error)
	Update(ctx context.Context, profile *VolunteerProfile) error
}

type ApplicationRepository interface {
	CreateApplication(ctx context.Context, app *TaskApplication) error
	GetApplicationsByVolunteerID(ctx context.Context, volunteerID int64) ([]TaskApplication, error)
	GetApplicationByTaskSlug(ctx context.Context, taskSlug string, volunteerID int64) (*TaskApplication, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}

type SubmissionRepository interface {
	CreateSubmission(ctx context.Context, sub *TaskSubmission) error
	GetSubmissionsByVolunteerID(ctx context.Context, volunteerID int64) ([]TaskSubmission, error)
	GetSubmissionByTaskSlug(ctx context.Context, taskSlug string, volunteerID int64) (*TaskSubmission, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *Payment) error
	GetPaymentsByVolunteerID(ctx context.Context, volunteerID int64) ([]Payment, error)
	UpdatePaymentStatus(ctx context.Context, id int64, status, txHash string) error
}
