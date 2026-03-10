package baseline

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/storage"
)

type Store interface {
	UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error
	ListTrainingLogs(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.TrainingLog, error)
	ListActivities(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.Activity, error)
	UpsertBaselineCurrent(ctx context.Context, b storage.BaselineCurrent) error
	CreateBaselineHistory(ctx context.Context, b storage.BaselineHistory) error
	GetTrainingLog(ctx context.Context, logID string) (storage.TrainingLog, error)
	UpsertTrainingSummary(ctx context.Context, summary storage.TrainingSummary) error
	GetTrainingSummary(ctx context.Context, logID string) (storage.TrainingSummary, error)
}

type Processor struct {
	store      Store
	now        func() time.Time
	summarizer ai.Summarizer
}

func NewProcessor(store Store) *Processor {
	return &Processor{store: store, now: time.Now}
}

func (p *Processor) SetSummarizer(s ai.Summarizer) {
	p.summarizer = s
}

func (p *Processor) ProcessBaselineRecalc(ctx context.Context, jobID, userID, triggerType, triggerRef string, retryCount int) error {
	if p.store == nil {
		return errors.New("baseline store is not configured")
	}
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "running", retryCount, ""); err != nil {
		return err
	}
	summaryErr, err := p.recalc(ctx, userID, triggerType, triggerRef)
	if err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	errMsg := ""
	if summaryErr != nil {
		errMsg = summaryErr.Error()
	}
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "success", retryCount, errMsg); err != nil {
		return err
	}
	return nil
}

func (p *Processor) recalc(ctx context.Context, userID, triggerType, triggerRef string) (error, error) {
	now := p.now()
	from := now.Add(-28 * 24 * time.Hour)

	logs, err := p.store.ListTrainingLogs(ctx, userID, from, now)
	if err != nil {
		return nil, err
	}
	activities, err := p.store.ListActivities(ctx, userID, from, now)
	if err != nil {
		return nil, err
	}

	sessions := make([]SessionInput, 0, len(logs)+len(activities))
	sessions7d := 0
	for _, log := range logs {
		idx := dayIndex(now, log.StartTime)
		if idx < 0 || idx > 27 {
			continue
		}
		rpePtr := (*int)(nil)
		if log.RPE > 0 {
			r := log.RPE
			rpePtr = &r
		}
		pace := log.PaceSecPerKM
		if pace == 0 && log.DistanceKM > 0 && log.DurationSec > 0 {
			pace = int(math.Round(float64(log.DurationSec) / log.DistanceKM))
		}
		sessions = append(sessions, SessionInput{
			DurationMin:   float64(log.DurationSec) / 60.0,
			DistanceKM:    log.DistanceKM,
			RPE:           rpePtr,
			PaceSecPerKM:  pace,
			StartDayIndex: idx,
		})
		if idx < 7 {
			sessions7d++
		}
	}
	for _, act := range activities {
		idx := dayIndex(now, act.StartTimeLocal)
		if idx < 0 || idx > 27 {
			continue
		}
		distanceKM := act.DistanceM / 1000.0
		pace := 0
		if distanceKM > 0 && act.MovingTimeSec > 0 {
			pace = int(math.Round(float64(act.MovingTimeSec) / distanceKM))
		}
		sessions = append(sessions, SessionInput{
			DurationMin:   float64(act.MovingTimeSec) / 60.0,
			DistanceKM:    distanceKM,
			RPE:           nil,
			PaceSecPerKM:  pace,
			StartDayIndex: idx,
		})
		if idx < 7 {
			sessions7d++
		}
	}

	metrics := CalcMetrics(sessions, sessions7d)
	computedAt := now.UTC()
	current := storage.BaselineCurrent{
		UserID:              userID,
		ComputedAt:          computedAt,
		DataSessions7d:      metrics.DataSessions7d,
		AcuteLoadSRPE:       metrics.AcuteSRPE,
		ChronicLoadSRPE:     metrics.ChronicSRPE,
		ACWRSRPE:            metrics.ACWRSRPE,
		AcuteLoadDistance:   metrics.AcuteDistance,
		ChronicLoadDistance: metrics.ChronicDistance,
		ACWRDistance:        metrics.ACWRDistance,
		Monotony:            metrics.Monotony,
		Strain:              metrics.Strain,
		PaceAvgSecPerKM:     metrics.PaceAvgSecPerKM,
		PaceLowSecPerKM:     metrics.PaceLowSecPerKM,
		PaceHighSecPerKM:    metrics.PaceHighSecPerKM,
		Status:              metrics.Status,
	}
	if err := p.store.UpsertBaselineCurrent(ctx, current); err != nil {
		return nil, err
	}

	history := storage.BaselineHistory{
		BaselineID:           uuid.NewString(),
		UserID:               userID,
		ComputedAt:           computedAt,
		TriggerType:          triggerType,
		TriggerRef:           triggerRef,
		DataSessions7d:       metrics.DataSessions7d,
		AcuteLoadSRPE:        metrics.AcuteSRPE,
		ChronicLoadSRPE:      metrics.ChronicSRPE,
		ACWRSRPE:             metrics.ACWRSRPE,
		AcuteLoadDistance:    metrics.AcuteDistance,
		ChronicLoadDistance:  metrics.ChronicDistance,
		ACWRDistance:         metrics.ACWRDistance,
		Monotony:             metrics.Monotony,
		Strain:               metrics.Strain,
		PaceAvgSecPerKM:      metrics.PaceAvgSecPerKM,
		PaceLowSecPerKM:      metrics.PaceLowSecPerKM,
		PaceHighSecPerKM:     metrics.PaceHighSecPerKM,
		Status:               metrics.Status,
	}
	if err := p.store.CreateBaselineHistory(ctx, history); err != nil {
		return nil, err
	}
	summaryErr := p.updateTrainingSummary(ctx, userID, triggerType, triggerRef, metrics)
	return summaryErr, nil
}

func dayIndex(now time.Time, start time.Time) int {
	startLocal := time.Date(start.In(now.Location()).Year(), start.In(now.Location()).Month(), start.In(now.Location()).Day(), 0, 0, 0, 0, now.Location())
	nowLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	diff := nowLocal.Sub(startLocal)
	return int(diff.Hours() / 24)
}

func (p *Processor) updateTrainingSummary(ctx context.Context, userID, triggerType, triggerRef string, metrics Metrics) error {
	if triggerType != "training_create" && triggerType != "training_update" && triggerType != "training_delete" {
		return nil
	}
	if triggerType == "training_delete" {
		return nil
	}
	log, err := p.store.GetTrainingLog(ctx, triggerRef)
	if err != nil {
		return err
	}
	output, err := p.generateSummary(ctx, log, metrics)
	if err != nil {
		if errors.Is(err, errAISummaryFailed) {
			if _, getErr := p.store.GetTrainingSummary(ctx, triggerRef); getErr != nil {
				if errors.Is(getErr, pgx.ErrNoRows) {
					fallback := fallbackSummary()
					return p.store.UpsertTrainingSummary(ctx, storage.TrainingSummary{
						SummaryID:        uuid.NewString(),
						UserID:           userID,
						LogID:            log.LogID,
						CompletionRate:   fallback.CompletionRate,
						IntensityMatch:   fallback.IntensityMatch,
						RecoveryAdvice:   fallback.RecoveryAdvice,
						AnomalyNotes:     fallback.AnomalyNotes,
						PerformanceNotes: fallback.PerformanceNotes,
						NextSuggestion:   fallback.NextSuggestion,
					})
				}
				return getErr
			}
			return err
		}
		return err
	}
	summary := storage.TrainingSummary{
		SummaryID:        uuid.NewString(),
		UserID:           userID,
		LogID:            log.LogID,
		CompletionRate:   output.CompletionRate,
		IntensityMatch:   output.IntensityMatch,
		RecoveryAdvice:   output.RecoveryAdvice,
		AnomalyNotes:     output.AnomalyNotes,
		PerformanceNotes: output.PerformanceNotes,
		NextSuggestion:   output.NextSuggestion,
	}
	return p.store.UpsertTrainingSummary(ctx, summary)
}

var errAISummaryFailed = errors.New("ai summary failed")

func (p *Processor) generateSummary(ctx context.Context, log storage.TrainingLog, metrics Metrics) (ai.SummaryOutput, error) {
	input := ai.SummaryInput{
		UserID:             log.UserID,
		LogID:              log.LogID,
		TrainingType:       log.TrainingType,
		TrainingTypeCustom: log.TrainingTypeCustom,
		StartTime:          log.StartTime,
		DurationSec:        log.DurationSec,
		DistanceKM:         log.DistanceKM,
		PaceSecPerKM:       log.PaceSecPerKM,
		RPE:                log.RPE,
		Discomfort:         log.Discomfort,
		Baseline: ai.BaselineSnapshot{
			DataSessions7d:     metrics.DataSessions7d,
			AcuteLoadSRPE:      metrics.AcuteSRPE,
			ChronicLoadSRPE:    metrics.ChronicSRPE,
			ACWRSRPE:           metrics.ACWRSRPE,
			AcuteLoadDistance:  metrics.AcuteDistance,
			ChronicLoadDistance: metrics.ChronicDistance,
			ACWRDistance:       metrics.ACWRDistance,
			Monotony:           metrics.Monotony,
			Strain:             metrics.Strain,
			PaceAvgSecPerKM:    metrics.PaceAvgSecPerKM,
			PaceLowSecPerKM:    metrics.PaceLowSecPerKM,
			PaceHighSecPerKM:   metrics.PaceHighSecPerKM,
			Status:             metrics.Status,
		},
	}
	if p.summarizer == nil {
		return ai.SummaryOutput{}, errAISummaryFailed
	}
	output, err := p.summarizer.Summarize(ctx, input)
	if err != nil {
		return ai.SummaryOutput{}, fmt.Errorf("%w: %v", errAISummaryFailed, err)
	}
	return output, nil
}

func fallbackSummary() ai.SummaryOutput {
	text := "AI 当前不可用，已使用规则占位总结"
	return ai.SummaryOutput{
		CompletionRate:   text,
		IntensityMatch:   text,
		RecoveryAdvice:   text,
		AnomalyNotes:     text,
		PerformanceNotes: text,
		NextSuggestion:   text,
	}
}
