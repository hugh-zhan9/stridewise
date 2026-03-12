package personalization

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"stridewise/backend/internal/storage"
)

type Store interface {
	UpdateAsyncJobStatus(ctx context.Context, jobID, status string, retryCount int, errMsg string) error
	ListTrainingSummaries(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.TrainingSummary, error)
	ListRecentRecommendationFeedbackSignals(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.RecommendationFeedbackSignal, error)
	UpsertUserPersonalizationParams(ctx context.Context, params storage.UserPersonalizationParams) error
}

type Processor struct {
	store Store
	now   func() time.Time
}

func NewProcessor(store Store) *Processor {
	return &Processor{store: store, now: time.Now}
}

func (p *Processor) ProcessPersonalizationRecalc(ctx context.Context, jobID, userID, triggerType, triggerRef string, retryCount int) error {
	if p.store == nil {
		return errors.New("personalization store not configured")
	}
	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "running", retryCount, ""); err != nil {
		return err
	}

	to := p.now()
	from := to.Add(-28 * 24 * time.Hour)
	summaries, err := p.store.ListTrainingSummaries(ctx, userID, from, to)
	if err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}
	signals, err := p.store.ListRecentRecommendationFeedbackSignals(ctx, userID, from, to)
	if err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}

	params, reason := deriveParams(userID, summaries, signals)
	params.ReasonJSON = reason
	if err := p.store.UpsertUserPersonalizationParams(ctx, params); err != nil {
		_ = p.store.UpdateAsyncJobStatus(ctx, jobID, "failed", retryCount, err.Error())
		return err
	}

	if err := p.store.UpdateAsyncJobStatus(ctx, jobID, "success", retryCount, ""); err != nil {
		return err
	}
	return nil
}

func deriveParams(userID string, summaries []storage.TrainingSummary, signals []storage.RecommendationFeedbackSignal) (storage.UserPersonalizationParams, []byte) {
	intensityBias := 0.0
	volumeMultiplier := 1.0
	typePreference := map[string]float64{
		"轻松跑": 1.0,
		"有氧跑": 1.0,
		"间歇跑": 1.0,
		"长距离": 1.0,
	}

	lowCompletion := 0
	highIntensity := 0
	discomfortCount := 0
	for _, s := range summaries {
		switch classifyLevel(s.IntensityMatch) {
		case "low":
			intensityBias += 0.08
		case "high":
			intensityBias -= 0.08
			highIntensity++
		}
		switch classifyLevel(s.CompletionRate) {
		case "high":
			volumeMultiplier += 0.03
		case "low":
			volumeMultiplier -= 0.06
			lowCompletion++
		}
		if hasDiscomfort(s.AnomalyNotes) || hasDiscomfort(s.RecoveryAdvice) {
			intensityBias -= 0.04
			volumeMultiplier -= 0.04
			discomfortCount++
		}
	}

	feedbackStats := map[string]int{"yes": 0, "neutral": 0, "no": 0}
	for _, sig := range signals {
		trainingType := normalizeTrainingType(sig.WorkoutType)
		if trainingType == "" {
			continue
		}
		useful := normalizeUseful(sig.Useful)
		feedbackStats[useful]++
		switch useful {
		case "yes":
			typePreference[trainingType] += 0.15
		case "no":
			typePreference[trainingType] -= 0.15
		}
	}

	intensityBias = clamp(intensityBias, -0.30, 0.30)
	volumeMultiplier = clamp(volumeMultiplier, 0.70, 1.20)
	for k, v := range typePreference {
		typePreference[k] = clamp(v, 0.50, 1.50)
	}

	reason := map[string]any{
		"summary_count":         len(summaries),
		"feedback_count":        len(signals),
		"low_completion_count":  lowCompletion,
		"high_intensity_count":  highIntensity,
		"discomfort_count":      discomfortCount,
		"feedback_useful_stats": feedbackStats,
	}
	reasonJSON, _ := json.Marshal(reason)

	return storage.UserPersonalizationParams{
		UserID:           userID,
		IntensityBias:    intensityBias,
		VolumeMultiplier: volumeMultiplier,
		TypePreference:   typePreference,
		Version:          1,
	}, reasonJSON
}

func classifyLevel(raw string) string {
	x := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(x, "低"), strings.Contains(x, "不足"), strings.Contains(x, "偏低"):
		return "low"
	case strings.Contains(x, "高"), strings.Contains(x, "过强"), strings.Contains(x, "偏高"):
		return "high"
	case strings.Contains(x, "匹配"), strings.Contains(x, "适中"), strings.Contains(x, "正常"), strings.Contains(x, "完成"):
		return "mid"
	default:
		return "unknown"
	}
}

func hasDiscomfort(raw string) bool {
	x := strings.ToLower(strings.TrimSpace(raw))
	if x == "" {
		return false
	}
	return strings.Contains(x, "不适") || strings.Contains(x, "疼") || strings.Contains(x, "痛") || strings.Contains(x, "异常")
}

func normalizeTrainingType(raw string) string {
	x := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(x, "轻松"), strings.Contains(x, "easy"):
		return "轻松跑"
	case strings.Contains(x, "有氧"), strings.Contains(x, "aerobic"), strings.Contains(x, "endurance"):
		return "有氧跑"
	case strings.Contains(x, "间歇"), strings.Contains(x, "interval"):
		return "间歇跑"
	case strings.Contains(x, "长距离"), strings.Contains(x, "long"):
		return "长距离"
	default:
		return ""
	}
}

func normalizeUseful(raw string) string {
	x := strings.ToLower(strings.TrimSpace(raw))
	switch x {
	case "yes", "有用", "useful":
		return "yes"
	case "no", "无用", "not_useful":
		return "no"
	default:
		return "neutral"
	}
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return math.Round(v*1000) / 1000
}

