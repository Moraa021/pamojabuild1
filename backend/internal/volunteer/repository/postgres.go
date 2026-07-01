package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"pamojabuild1/backend/internal/volunteer"
)

type VolunteerRepository struct {
	db *sql.DB
}

func NewVolunteerRepository(db *sql.DB) *VolunteerRepository {
	return &VolunteerRepository{db: db}
}

// Profile operations
func (r *VolunteerRepository) CreateProfile(ctx context.Context, profile *volunteer.VolunteerProfile) error {
	skillsJSON, _ := json.Marshal(profile.Skills)
	query := `
		INSERT INTO volunteer_profiles (user_id, bio, skills, lightning_address, onchain_address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`
	
	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	
	return r.db.QueryRowContext(ctx, query,
		profile.UserID, profile.Bio, skillsJSON, profile.LightningAddress, profile.OnchainAddress, now, now,
	).Err()
}

func (r *VolunteerRepository) GetProfileByUserID(ctx context.Context, userID int64) (*volunteer.VolunteerProfile, error) {
	profile := &volunteer.VolunteerProfile{}
	var skillsJSON []byte
	
	query := `
		SELECT user_id, bio, skills, lightning_address, onchain_address, 
		       reputation_score, tier, completed_tasks, total_earned_sats, created_at, updated_at
		FROM volunteer_profiles WHERE user_id = $1`
	
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.UserID, &profile.Bio, &skillsJSON, &profile.LightningAddress,
		&profile.OnchainAddress, &profile.ReputationScore, &profile.Tier,
		&profile.CompletedTasks, &profile.TotalEarnedSats, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(skillsJSON, &profile.Skills)
	return profile, nil
}

func (r *VolunteerRepository) UpdateProfile(ctx context.Context, profile *volunteer.VolunteerProfile) error {
	skillsJSON, _ := json.Marshal(profile.Skills)
	query := `
		UPDATE volunteer_profiles 
		SET bio = $1, skills = $2, lightning_address = $3, onchain_address = $4, updated_at = $5
		WHERE user_id = $6`
	
	return r.db.QueryRowContext(ctx, query,
		profile.Bio, skillsJSON, profile.LightningAddress, profile.OnchainAddress,
		time.Now(), profile.UserID,
	).Err()
}

// Application operations
func (r *VolunteerRepository) CreateApplication(ctx context.Context, app *volunteer.TaskApplication) error {
	query := `
		INSERT INTO task_applications (task_slug, volunteer_id, message, status, applied_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	
	app.AppliedAt = time.Now()
	app.Status = "pending"
	
	return r.db.QueryRowContext(ctx, query,
		app.TaskSlug, app.VolunteerID, app.Message, app.Status, app.AppliedAt,
	).Scan(&app.ID)
}

func (r *VolunteerRepository) GetApplicationsByVolunteerID(ctx context.Context, volunteerID int64) ([]volunteer.TaskApplication, error) {
	query := `
		SELECT id, task_slug, volunteer_id, message, status, applied_at, reviewed_at
		FROM task_applications WHERE volunteer_id = $1
		ORDER BY applied_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query, volunteerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var applications []volunteer.TaskApplication
	for rows.Next() {
		var app volunteer.TaskApplication
		if err := rows.Scan(&app.ID, &app.TaskSlug, &app.VolunteerID, &app.Message,
			&app.Status, &app.AppliedAt, &app.ReviewedAt); err != nil {
			return nil, err
		}
		applications = append(applications, app)
	}
	
	return applications, nil
}

// Submission operations
func (r *VolunteerRepository) CreateSubmission(ctx context.Context, sub *volunteer.TaskSubmission) error {
	evidenceJSON, _ := json.Marshal(sub.EvidenceURLs)
	query := `
		INSERT INTO task_submissions (task_slug, volunteer_id, description, evidence_urls, status, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`
	
	sub.SubmittedAt = time.Now()
	sub.Status = "submitted"
	
	return r.db.QueryRowContext(ctx, query,
		sub.TaskSlug, sub.VolunteerID, sub.Description, evidenceJSON, sub.Status, sub.SubmittedAt,
	).Scan(&sub.ID)
}

func (r *VolunteerRepository) GetSubmissionsByVolunteerID(ctx context.Context, volunteerID int64) ([]volunteer.TaskSubmission, error) {
	query := `
		SELECT id, task_slug, volunteer_id, description, evidence_urls, status, submitted_at, reviewed_at
		FROM task_submissions WHERE volunteer_id = $1
		ORDER BY submitted_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query, volunteerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var submissions []volunteer.TaskSubmission
	for rows.Next() {
		var sub volunteer.TaskSubmission
		var evidenceJSON []byte
		if err := rows.Scan(&sub.ID, &sub.TaskSlug, &sub.VolunteerID, &sub.Description,
			&evidenceJSON, &sub.Status, &sub.SubmittedAt, &sub.ReviewedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(evidenceJSON, &sub.EvidenceURLs)
		submissions = append(submissions, sub)
	}
	
	return submissions, nil
}

// Payment operations
func (r *VolunteerRepository) GetPaymentsByVolunteerID(ctx context.Context, volunteerID int64) ([]volunteer.Payment, error) {
	query := `
		SELECT id, task_slug, volunteer_id, amount_sats, payment_method, status, transaction_hash, paid_at
		FROM volunteer_payments WHERE volunteer_id = $1
		ORDER BY paid_at DESC NULLS LAST`
	
	rows, err := r.db.QueryContext(ctx, query, volunteerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var payments []volunteer.Payment
	for rows.Next() {
		var p volunteer.Payment
		if err := rows.Scan(&p.ID, &p.TaskSlug, &p.VolunteerID, &p.AmountSats,
			&p.PaymentMethod, &p.Status, &p.TransactionHash, &p.PaidAt); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	
	return payments, nil
}