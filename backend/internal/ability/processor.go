package ability

import (
	"context"
	"errors"
	"time"

	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/storage"
)

type Store interface {
	UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error
	GetUserProfile(ctx context.Context, userID string) (storage.UserProfile, error)
	ListTrainingLogs(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.TrainingLog, error)
	ListActivities(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.Activity, error)
	UpdateAbilityLevel(ctx context.Context, userID, level, reason string, updatedAt time.Time) error
}

type Processor struct {
	store Store
	ai    ai.AbilityLeveler
	now   func() time.Time
}

func NewProcessor(store Store, leveler ai.AbilityLeveler) *Processor {
	return &Processor{store: store, ai: leveler, now: time.Now}
}

func (p *Processor) ProcessAbilityLevel(ctx context.Context, jobID, userID, triggerType, triggerRef string, retryCount int) error {
	if p.store == nil {
		return errors.New("ability store not configured")
	}
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "running", retryCount, ""); err != nil {
		return err
	}
	profile, err := p.store.GetUserProfile(ctx, userID)
	if err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	from := p.now().Add(-28 * 24 * time.Hour)
	to := p.now()
	logs, err := p.store.ListTrainingLogs(ctx, userID, from, to)
	if err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	acts, err := p.store.ListActivities(ctx, userID, from, to)
	if err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}

	input := ai.AbilityLevelInput{
		UserID: userID,
		Profile: ai.AbilityProfileSnapshot{
			Age:              profile.Age,
			WeightKG:         profile.WeightKG,
			RunningYears:     profile.RunningYears,
			WeeklySessions:   profile.WeeklySessions,
			WeeklyDistanceKM: profile.WeeklyDistanceKM,
			LongestRunKM:     profile.LongestRunKM,
		},
		TrainingSummary: buildSummary(logs, acts),
	}

	if p.ai == nil {
		err := errors.New("ability ai not configured")
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	out, err := p.ai.EvaluateAbilityLevel(ctx, input)
	if err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	if err := p.store.UpdateAbilityLevel(ctx, userID, out.AbilityLevel, out.Reason, p.now().UTC()); err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "success", retryCount, ""); err != nil {
		return err
	}
	return nil
}

func buildSummary(logs []storage.TrainingLog, acts []storage.Activity) ai.AbilityTrainingSummary {
	var sessions int
	var distance float64
	var duration int
	var paceSum float64
	var paceDistance float64
	var srpeSum float64

	sessions += len(logs) + len(acts)
	for _, log := range logs {
		distance += log.DistanceKM
		duration += log.DurationSec
		if log.DistanceKM > 0 && log.PaceSecPerKM > 0 {
			paceSum += float64(log.PaceSecPerKM) * log.DistanceKM
			paceDistance += log.DistanceKM
		}
		if log.RPE > 0 {
			srpeSum += float64(log.DurationSec/60) * float64(log.RPE)
		}
	}
	for _, act := range acts {
		distance += act.DistanceM / 1000.0
		duration += act.MovingTimeSec
	}

	avgPace := 0
	if paceDistance > 0 {
		avgPace = int(paceSum / paceDistance)
	}
	avgRPE := 0.0
	if len(logs) > 0 {
		avgRPE = srpeSum / float64(len(logs))
	}

	return ai.AbilityTrainingSummary{
		Sessions:         sessions,
		TotalDistanceKM:  distance,
		TotalDurationSec: duration,
		AvgPaceSecPerKM:  avgPace,
		AvgRPE:           avgRPE,
		SRPELoad:         srpeSum,
	}
}
