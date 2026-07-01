package service

import (
    "context"
    "errors"
    "testing"

    "pamojabuild1/backend/internal/events"
    "pamojabuild1/backend/internal/task"
)

type mockTaskRepo struct{
    created *task.Task
    err error
}
func (m *mockTaskRepo) Create(ctx context.Context, t *task.Task) error {
    if m.err != nil { return m.err }
    m.created = t
    return nil
}
func (m *mockTaskRepo) GetByID(ctx context.Context, id int64) (*task.Task, error) { return nil, nil }
func (m *mockTaskRepo) GetBySlug(ctx context.Context, slug string) (*task.Task, error) { return nil, nil }
func (m *mockTaskRepo) List(ctx context.Context, category, region, status string) ([]task.Task, error) { return nil, nil }
func (m *mockTaskRepo) UpdateStatus(ctx context.Context, slug string, status string) error { return nil }
func (m *mockTaskRepo) UpdateFinancialState(ctx context.Context, slug string, state string) error { return nil }

func TestCreateCampaignPublishesEvent(t *testing.T) {
    repo := &mockTaskRepo{}
    bus := events.NewEventBus()
    var got bool
    unsub := bus.Subscribe(events.TaskCreated, func(e events.Event) {
        if payload, ok := e.Payload.(events.TaskCreatedPayload); ok && payload.Title == "Hello" {
            got = true
        }
    })
    defer unsub()

    svc := NewTaskService(repo, bus)
    ctx := context.Background()
    tsk := &task.Task{Title: "Hello"}
    res, err := svc.CreateCampaign(ctx, tsk)
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if res.Slug == "" { t.Fatalf("expected slug set") }
    // allow goroutine to run
    // small sleep
    // but avoid import time by checking repo
    if repo.created == nil { t.Fatalf("expected repo.Create called") }
    // Replay history to ensure event published
    bus.Replay()
    if !got { t.Fatalf("expected TaskCreated event to be published") }
}

func TestCreateCampaignRepoError(t *testing.T) {
    repo := &mockTaskRepo{err: errors.New("oops")}
    bus := events.NewEventBus()
    svc := NewTaskService(repo, bus)
    _, err := svc.CreateCampaign(context.Background(), &task.Task{Title: "X"})
    if err == nil { t.Fatalf("expected error") }
}
