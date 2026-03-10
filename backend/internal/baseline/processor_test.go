package baseline

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"stridewise/backend/internal/storage"
)

type fakeBaselineStore struct {
	statuses       []string
	current        storage.BaselineCurrent
	history        storage.BaselineHistory
	trainingLogs   []storage.TrainingLog
	activities     []storage.Activity
	summaryUpdated bool
}

func (f *fakeBaselineStore) UpdateAsyncJobStatus(_ context.Context, _ string, status string, _ int, _ string) error {
	f.statuses = append(f.statuses, status)
	return nil
}

func (f *fakeBaselineStore) ListTrainingLogs(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingLog, error) {
	return f.trainingLogs, nil
}

func (f *fakeBaselineStore) ListActivities(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.Activity, error) {
	return f.activities, nil
}

func (f *fakeBaselineStore) UpsertBaselineCurrent(_ context.Context, b storage.BaselineCurrent) error {
	f.current = b
	return nil
}

func (f *fakeBaselineStore) CreateBaselineHistory(_ context.Context, b storage.BaselineHistory) error {
	f.history = b
	return nil
}

func (f *fakeBaselineStore) GetTrainingLog(_ context.Context, logID string) (storage.TrainingLog, error) {
	for _, log := range f.trainingLogs {
		if log.LogID == logID {
			return log, nil
		}
	}
	return storage.TrainingLog{}, pgx.ErrNoRows
}

func (f *fakeBaselineStore) UpsertTrainingSummary(_ context.Context, _ storage.TrainingSummary) error {
	f.summaryUpdated = true
	return nil
}

func (f *fakeBaselineStore) GetTrainingSummary(_ context.Context, _ string) (storage.TrainingSummary, error) {
	return storage.TrainingSummary{}, pgx.ErrNoRows
}

func TestProcessBaselineRecalc_UpdatesStatus(t *testing.T) {
	now := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	store := &fakeBaselineStore{
		trainingLogs: []storage.TrainingLog{
			{LogID: "log-1", UserID: "u1", StartTime: now.Add(-2 * time.Hour), DurationSec: 1800, DistanceKM: 5, RPE: 6, PaceSecPerKM: 360},
			{LogID: "log-2", UserID: "u1", StartTime: now.Add(-24 * time.Hour), DurationSec: 2000, DistanceKM: 5, RPE: 6, PaceSecPerKM: 360},
			{LogID: "log-3", UserID: "u1", StartTime: now.Add(-48 * time.Hour), DurationSec: 2100, DistanceKM: 6, RPE: 6, PaceSecPerKM: 350},
		},
	}
	p := NewProcessor(store)
	p.now = func() time.Time { return now }

	if err := p.ProcessBaselineRecalc(context.Background(), "job-1", "u1", "training_create", "log-1", 0); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(store.statuses) < 2 || store.statuses[0] != "running" || store.statuses[len(store.statuses)-1] != "success" {
		t.Fatalf("expected running->success, got %v", store.statuses)
	}
	if store.current.UserID != "u1" || store.current.Status != "ok" {
		t.Fatalf("unexpected baseline current: %+v", store.current)
	}
	if store.history.TriggerType != "training_create" || store.history.TriggerRef != "log-1" {
		t.Fatalf("unexpected baseline history: %+v", store.history)
	}
	if !store.summaryUpdated {
		t.Fatalf("expected summary updated")
	}
}
