package personalization

import (
	"context"
	"testing"
	"time"

	"stridewise/backend/internal/storage"
)

type fakeStore struct {
	updated bool
	params  storage.UserPersonalizationParams
}

func (f *fakeStore) UpdateAsyncJobStatus(_ context.Context, _ string, _ string, _ int, _ string) error {
	return nil
}

func (f *fakeStore) ListTrainingSummaries(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingSummary, error) {
	return []storage.TrainingSummary{
		{
			CompletionRate: "偏低",
			IntensityMatch: "偏高",
			AnomalyNotes:   "有轻微不适",
		},
		{
			CompletionRate: "适中",
			IntensityMatch: "匹配",
		},
	}, nil
}

func (f *fakeStore) ListRecentRecommendationFeedbackSignals(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.RecommendationFeedbackSignal, error) {
	return []storage.RecommendationFeedbackSignal{
		{Useful: "yes", WorkoutType: "轻松跑"},
		{Useful: "no", WorkoutType: "间歇跑"},
	}, nil
}

func (f *fakeStore) UpsertUserPersonalizationParams(_ context.Context, p storage.UserPersonalizationParams) error {
	f.updated = true
	f.params = p
	return nil
}

func TestProcessPersonalizationRecalc(t *testing.T) {
	store := &fakeStore{}
	p := NewProcessor(store)
	p.now = func() time.Time { return time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC) }

	if err := p.ProcessPersonalizationRecalc(context.Background(), "job-1", "u1", "feedback", "f1", 0); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.updated {
		t.Fatalf("expected params updated")
	}
	if store.params.UserID != "u1" {
		t.Fatalf("expected user id")
	}
	if store.params.VolumeMultiplier >= 1.0 {
		t.Fatalf("expected volume reduced by low completion and discomfort")
	}
	if store.params.TypePreference["轻松跑"] <= store.params.TypePreference["间歇跑"] {
		t.Fatalf("expected easy run preference higher than interval")
	}
}

