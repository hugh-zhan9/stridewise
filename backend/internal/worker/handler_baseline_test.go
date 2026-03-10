package worker

import (
	"context"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"

	"stridewise/backend/internal/baseline"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/task"
)

type fakeBaselineStore struct {
	called bool
}

func (f *fakeBaselineStore) UpdateAsyncJobStatus(_ context.Context, _ string, _ string, _ int, _ string) error {
	f.called = true
	return nil
}

func (f *fakeBaselineStore) ListTrainingLogs(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingLog, error) {
	return nil, nil
}

func (f *fakeBaselineStore) ListActivities(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.Activity, error) {
	return nil, nil
}

func (f *fakeBaselineStore) ListActivitiesBySyncJob(_ context.Context, _ string) ([]storage.Activity, error) {
	return nil, nil
}

func (f *fakeBaselineStore) UpsertBaselineCurrent(_ context.Context, _ storage.BaselineCurrent) error {
	return nil
}

func (f *fakeBaselineStore) CreateBaselineHistory(_ context.Context, _ storage.BaselineHistory) error {
	return nil
}

func (f *fakeBaselineStore) GetTrainingLog(_ context.Context, _ string) (storage.TrainingLog, error) {
	return storage.TrainingLog{}, pgx.ErrNoRows
}

func (f *fakeBaselineStore) UpsertTrainingSummary(_ context.Context, _ storage.TrainingSummary) error {
	return nil
}

func (f *fakeBaselineStore) GetTrainingSummary(_ context.Context, _ string) (storage.TrainingSummary, error) {
	return storage.TrainingSummary{}, pgx.ErrNoRows
}

func (f *fakeBaselineStore) GetTrainingSummaryBySource(_ context.Context, _ string, _ string) (storage.TrainingSummary, error) {
	return storage.TrainingSummary{}, pgx.ErrNoRows
}

func (f *fakeBaselineStore) SoftDeleteTrainingSummaryBySource(_ context.Context, _ string, _ string) error {
	return nil
}

func (f *fakeBaselineStore) SoftDeleteTrainingFeedbackBySource(_ context.Context, _ string, _ string) error {
	return nil
}

func TestHandleBaselineRecalc_RequiresProcessor(t *testing.T) {
	SetBaselineProcessor(nil)
	payload, _ := task.EncodeBaselineRecalcPayload(task.BaselineRecalcPayload{JobID: "job-1", UserID: "u1", TriggerType: "training_create", TriggerRef: "log-1"})
	asynqTask := asynq.NewTask(task.TypeBaselineRecalc, payload)
	if err := HandleBaselineRecalc(context.Background(), asynqTask); err == nil {
		t.Fatalf("expected error")
	}
}

func TestHandleBaselineRecalc_UpdatesStatus(t *testing.T) {
	store := &fakeBaselineStore{}
	processor := baseline.NewProcessor(store)
	SetBaselineProcessor(processor)

	payload, _ := task.EncodeBaselineRecalcPayload(task.BaselineRecalcPayload{JobID: "job-1", UserID: "u1", TriggerType: "sync", TriggerRef: "job-1"})
	asynqTask := asynq.NewTask(task.TypeBaselineRecalc, payload)
	if err := HandleBaselineRecalc(context.Background(), asynqTask); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.called {
		t.Fatalf("expected async job update")
	}
}
