package worker

import (
	"context"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"stridewise/backend/internal/personalization"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/task"
)

type fakePersonalizationStore struct {
	called bool
}

func (f *fakePersonalizationStore) UpdateAsyncJobStatus(_ context.Context, _ string, _ string, _ int, _ string) error {
	f.called = true
	return nil
}

func (f *fakePersonalizationStore) ListTrainingSummaries(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingSummary, error) {
	return nil, nil
}

func (f *fakePersonalizationStore) ListRecentRecommendationFeedbackSignals(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.RecommendationFeedbackSignal, error) {
	return nil, nil
}

func (f *fakePersonalizationStore) UpsertUserPersonalizationParams(_ context.Context, _ storage.UserPersonalizationParams) error {
	return nil
}

func TestHandlePersonalizationRecalc_RequiresProcessor(t *testing.T) {
	SetPersonalizationProcessor(nil)
	payload, _ := task.EncodePersonalizationRecalcPayload(task.PersonalizationRecalcPayload{
		JobID: "job-1", UserID: "u1", TriggerType: "feedback", TriggerRef: "f1",
	})
	asynqTask := asynq.NewTask(task.TypePersonalizationRecalc, payload)
	if err := HandlePersonalizationRecalc(context.Background(), asynqTask); err == nil {
		t.Fatalf("expected error")
	}
}

func TestHandlePersonalizationRecalc_UpdatesStatus(t *testing.T) {
	store := &fakePersonalizationStore{}
	processor := personalization.NewProcessor(store)
	SetPersonalizationProcessor(processor)

	payload, _ := task.EncodePersonalizationRecalcPayload(task.PersonalizationRecalcPayload{
		JobID: "job-1", UserID: "u1", TriggerType: "feedback", TriggerRef: "f1",
	})
	asynqTask := asynq.NewTask(task.TypePersonalizationRecalc, payload)
	if err := HandlePersonalizationRecalc(context.Background(), asynqTask); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.called {
		t.Fatalf("expected async job update")
	}
}

