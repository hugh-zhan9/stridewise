package trend

import (
	"context"
	"testing"
	"time"

	"stridewise/backend/internal/storage"
)

type fakeTrendStore struct {
	logs      []storage.TrainingLog
	acts      []storage.Activity
	summaries []storage.TrainingSummary
	baselines []storage.BaselineHistory
}

func (f *fakeTrendStore) ListTrainingLogs(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingLog, error) {
	return f.logs, nil
}

func (f *fakeTrendStore) ListActivities(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.Activity, error) {
	return f.acts, nil
}

func (f *fakeTrendStore) ListTrainingSummaries(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingSummary, error) {
	return f.summaries, nil
}

func (f *fakeTrendStore) ListBaselineHistory(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.BaselineHistory, error) {
	return f.baselines, nil
}

func TestAggregateRollingTrend(t *testing.T) {
	store := &fakeTrendStore{
		logs: []storage.TrainingLog{{
			UserID:       "u1",
			StartTime:    time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC),
			DurationSec:  1800,
			DistanceKM:   5,
			PaceSecPerKM: 360,
			RPE:          6,
		}},
		acts: []storage.Activity{{
			UserID:         "u1",
			StartTimeLocal: time.Date(2026, 3, 9, 8, 0, 0, 0, time.UTC),
			MovingTimeSec:  1500,
			DistanceM:      4000,
		}},
		summaries: []storage.TrainingSummary{{
			UserID:         "u1",
			SourceType:     "log",
			SourceID:       "log-1",
			CompletionRate: "适中",
			IntensityMatch: "偏高",
			RecoveryAdvice: "补水",
		}},
		baselines: []storage.BaselineHistory{{
			UserID:       "u1",
			ComputedAt:   time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC),
			ACWRSRPE:     1.2,
			ACWRDistance: 1.1,
			Monotony:     1.8,
			Strain:       120,
		}},
	}
	p := NewProcessor(store)

	out, err := p.GetRollingTrend(context.Background(), "u1", "7d", time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Summary.Sessions != 2 {
		t.Fatalf("expected sessions 2, got %d", out.Summary.Sessions)
	}
	if out.Summary.AvgRPE <= 0 {
		t.Fatalf("expected avg rpe")
	}
	if out.Summary.CompletionRateDist["mid"] != 1 {
		t.Fatalf("expected completion_rate mid count")
	}
	if out.Summary.RecoveryAdviceTags["补水"] != 1 {
		t.Fatalf("expected recovery tag count")
	}
	if out.Summary.ACWRSRPE == nil {
		t.Fatalf("expected baseline acwr")
	}
}
