package worker

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"

	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/task"
	"stridewise/backend/internal/training"
)

type fakeAsyncStore struct {
	called bool
}

func (f *fakeAsyncStore) UpdateAsyncJobStatus(_ context.Context, _ string, _ string, _ int, _ string) error {
	f.called = true
	return nil
}

type baselineStub struct{}

func (baselineStub) RecalcForTrigger(_ context.Context, _ string, _ string, _ string) (error, error) {
	return nil, nil
}

type recStub struct{}

func (recStub) Generate(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{}, nil
}

func TestHandleTrainingRecalc_RequiresProcessor(t *testing.T) {
	SetTrainingProcessor(nil)
	payload, _ := task.EncodeTrainingRecalcPayload(task.TrainingRecalcPayload{JobID: "job-1", UserID: "u1", LogID: "log-1", Operation: "create"})
	asynqTask := asynq.NewTask(task.TypeTrainingRecalc, payload)
	if err := HandleTrainingRecalc(context.Background(), asynqTask); err == nil {
		t.Fatalf("expected error")
	}
}

func TestHandleTrainingRecalc_UpdatesStatus(t *testing.T) {
	store := &fakeAsyncStore{}
	processor := training.NewProcessor(store, baselineStub{}, recStub{})
	SetTrainingProcessor(processor)

	payload, _ := task.EncodeTrainingRecalcPayload(task.TrainingRecalcPayload{JobID: "job-1", UserID: "u1", LogID: "log-1", Operation: "create"})
	asynqTask := asynq.NewTask(task.TypeTrainingRecalc, payload)
	if err := HandleTrainingRecalc(context.Background(), asynqTask); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.called {
		t.Fatalf("expected async job update")
	}
}
