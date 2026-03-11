package ability

import (
	"context"
	"errors"
	"testing"
	"time"

	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/storage"
)

type fakeStore struct {
	profile storage.UserProfile
	logs    []storage.TrainingLog
	acts    []storage.Activity
	status  string
	updated bool
	ability string
}

func (f *fakeStore) UpdateAsyncJobStatus(_ context.Context, _ string, status string, _ int, _ string) error {
	f.status = status
	return nil
}

func (f *fakeStore) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) {
	return f.profile, nil
}

func (f *fakeStore) ListTrainingLogs(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingLog, error) {
	return f.logs, nil
}

func (f *fakeStore) ListActivities(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.Activity, error) {
	return f.acts, nil
}

func (f *fakeStore) UpdateAbilityLevel(_ context.Context, _ string, level string, _ string, _ time.Time) error {
	f.updated = true
	f.ability = level
	return nil
}

type fakeAI struct {
	level string
	err   error
}

func (f fakeAI) EvaluateAbilityLevel(_ context.Context, _ ai.AbilityLevelInput) (ai.AbilityLevelOutput, error) {
	if f.err != nil {
		return ai.AbilityLevelOutput{}, f.err
	}
	return ai.AbilityLevelOutput{AbilityLevel: f.level, Reason: "ok"}, nil
}

func TestProcessAbilityLevel_Success(t *testing.T) {
	store := &fakeStore{profile: storage.UserProfile{
		UserID:           "u1",
		Age:              28,
		WeightKG:         65,
		RunningYears:     "1-3",
		WeeklySessions:   "2-3",
		WeeklyDistanceKM: "5-15",
		LongestRunKM:     "10",
	}}
	p := NewProcessor(store, fakeAI{level: "intermediate"})
	if err := p.ProcessAbilityLevel(context.Background(), "job-1", "u1", "profile", "u1", 0); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if store.status != "success" {
		t.Fatalf("expected status success, got %s", store.status)
	}
	if !store.updated || store.ability != "intermediate" {
		t.Fatalf("expected ability updated")
	}
}

func TestProcessAbilityLevel_FailureDoesNotOverride(t *testing.T) {
	store := &fakeStore{profile: storage.UserProfile{UserID: "u1", AbilityLevel: "beginner"}}
	p := NewProcessor(store, fakeAI{err: errors.New("boom")})
	if err := p.ProcessAbilityLevel(context.Background(), "job-1", "u1", "profile", "u1", 1); err == nil {
		t.Fatalf("expected err")
	}
	if store.status != "failed" {
		t.Fatalf("expected status failed, got %s", store.status)
	}
	if store.updated {
		t.Fatalf("expected ability not updated")
	}
}
