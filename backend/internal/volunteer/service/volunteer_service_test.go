package service

import (
    "context"
    "errors"
    "testing"

    "pamojabuild1/backend/internal/volunteer"
)

type mockVolunteerProfileRepo struct {
    profile   *volunteer.VolunteerProfile
    updateErr error
}

func (m *mockVolunteerProfileRepo) Create(ctx context.Context, profile *volunteer.VolunteerProfile) error {
    m.profile = profile
    return nil
}

func (m *mockVolunteerProfileRepo) GetByUserID(ctx context.Context, userID int64) (*volunteer.VolunteerProfile, error) {
    if m.profile != nil && m.profile.UserID == userID {
        return m.profile, nil
    }
    return nil, errors.New("not found")
}

func (m *mockVolunteerProfileRepo) Update(ctx context.Context, profile *volunteer.VolunteerProfile) error {
    if m.updateErr != nil {
        return m.updateErr
    }
    m.profile = profile
    return nil
}

func TestGetProfileSuccess(t *testing.T) {
    repo := &mockVolunteerProfileRepo{profile: &volunteer.VolunteerProfile{UserID: 1}}
    svc := NewVolunteerService(repo)

    profile, err := svc.GetProfile(context.Background(), 1)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if profile.UserID != 1 {
        t.Fatalf("expected profile for user 1, got %#v", profile)
    }
}

func TestGetProfileNotFound(t *testing.T) {
    repo := &mockVolunteerProfileRepo{}
    svc := NewVolunteerService(repo)

    _, err := svc.GetProfile(context.Background(), 2)
    if err == nil {
        t.Fatal("expected error when profile is missing")
    }
}

func TestUpdateProfile(t *testing.T) {
    repo := &mockVolunteerProfileRepo{profile: &volunteer.VolunteerProfile{UserID: 1}}
    svc := NewVolunteerService(repo)

    profile := &volunteer.VolunteerProfile{UserID: 1, Bio: "Updated"}
    err := svc.UpdateProfile(context.Background(), 1, profile)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if repo.profile.Bio != "Updated" {
        t.Fatalf("expected profile update, got %#v", repo.profile)
    }
}
