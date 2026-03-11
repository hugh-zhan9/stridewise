package trend

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"stridewise/backend/internal/storage"
)

type TrendStore interface {
	ListTrainingLogs(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.TrainingLog, error)
	ListActivities(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.Activity, error)
	ListTrainingSummaries(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.TrainingSummary, error)
	ListBaselineHistory(ctx context.Context, userID string, from time.Time, to time.Time) ([]storage.BaselineHistory, error)
}

type Processor struct {
	store TrendStore
}

func NewProcessor(store TrendStore) *Processor {
	return &Processor{store: store}
}

func (p *Processor) Aggregate(ctx context.Context, userID string, window string, asOf time.Time) (TrendResult, error) {
	if userID == "" {
		return TrendResult{}, errors.New("user_id required")
	}
	if p.store == nil {
		return TrendResult{}, errors.New("trend store not configured")
	}
	days, err := windowDays(window)
	if err != nil {
		return TrendResult{}, err
	}
	loc := asOf.Location()
	end := asOf
	start := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(days-1))

	logs, err := p.store.ListTrainingLogs(ctx, userID, start, end)
	if err != nil {
		return TrendResult{}, err
	}
	acts, err := p.store.ListActivities(ctx, userID, start, end)
	if err != nil {
		return TrendResult{}, err
	}
	summaries, err := p.store.ListTrainingSummaries(ctx, userID, start, end)
	if err != nil {
		return TrendResult{}, err
	}
	baselines, err := p.store.ListBaselineHistory(ctx, userID, start, end)
	if err != nil {
		return TrendResult{}, err
	}

	summary := buildSummary(logs, acts, summaries, baselines, end)
	series := buildSeries(logs, acts, start, end)

	return TrendResult{
		WindowStart: start.Format("2006-01-02"),
		WindowEnd:   end.In(loc).Format("2006-01-02"),
		Summary:     summary,
		Series:      series,
	}, nil
}

type dayAgg struct {
	sessions     int
	distanceKM   float64
	durationSec  int
	paceDistance float64
	rpeSum       float64
	rpeCount     int
}

func buildSummary(logs []storage.TrainingLog, acts []storage.Activity, summaries []storage.TrainingSummary, baselines []storage.BaselineHistory, asOf time.Time) TrendSummary {
	out := TrendSummary{
		CompletionRateDist: map[string]int{"low": 0, "mid": 0, "high": 0, "unknown": 0},
		IntensityMatchDist: map[string]int{"low": 0, "mid": 0, "high": 0, "unknown": 0},
		RecoveryAdviceTags: map[string]int{},
	}

	out.Sessions = len(logs) + len(acts)

	var paceDistance float64
	var rpeSum float64
	var rpeCount int
	for _, log := range logs {
		out.DistanceKM += log.DistanceKM
		out.DurationSec += log.DurationSec
		pace := log.PaceSecPerKM
		if pace <= 0 && log.DistanceKM > 0 && log.DurationSec > 0 {
			pace = int(math.Round(float64(log.DurationSec) / log.DistanceKM))
		}
		if log.DistanceKM > 0 && pace > 0 {
			paceDistance += float64(pace) * log.DistanceKM
		}
		if log.RPE > 0 {
			rpeSum += float64(log.RPE)
			rpeCount++
		}
	}

	for _, act := range acts {
		km := act.DistanceM / 1000.0
		out.DistanceKM += km
		out.DurationSec += act.MovingTimeSec
		if km > 0 && act.MovingTimeSec > 0 {
			paceDistance += (float64(act.MovingTimeSec) / km) * km
		}
	}

	if out.DistanceKM > 0 {
		out.AvgPaceSecPerKM = int(math.Round(paceDistance / out.DistanceKM))
	}
	if rpeCount > 0 {
		out.AvgRPE = rpeSum / float64(rpeCount)
	}

	out.SummaryCount = len(summaries)
	for _, summary := range summaries {
		out.CompletionRateDist[classifyLevel(summary.CompletionRate)]++
		out.IntensityMatchDist[classifyLevel(summary.IntensityMatch)]++
		for _, tag := range extractAdviceTags(summary.RecoveryAdvice) {
			out.RecoveryAdviceTags[tag]++
		}
	}

	for _, b := range baselines {
		if b.ComputedAt.After(asOf) {
			continue
		}
		acwrSRPE := b.ACWRSRPE
		acwrDistance := b.ACWRDistance
		monotony := b.Monotony
		strain := b.Strain
		out.ACWRSRPE = &acwrSRPE
		out.ACWRDistance = &acwrDistance
		out.Monotony = &monotony
		out.Strain = &strain
		break
	}

	return out
}

func buildSeries(logs []storage.TrainingLog, acts []storage.Activity, start time.Time, end time.Time) []TrendPoint {
	loc := start.Location()
	seriesMap := make(map[string]*dayAgg)

	for _, log := range logs {
		day := log.StartTime.In(loc).Format("2006-01-02")
		bucket := ensureDayAgg(seriesMap, day)
		bucket.sessions++
		bucket.distanceKM += log.DistanceKM
		bucket.durationSec += log.DurationSec
		pace := log.PaceSecPerKM
		if pace <= 0 && log.DistanceKM > 0 && log.DurationSec > 0 {
			pace = int(math.Round(float64(log.DurationSec) / log.DistanceKM))
		}
		if log.DistanceKM > 0 && pace > 0 {
			bucket.paceDistance += float64(pace) * log.DistanceKM
		}
		if log.RPE > 0 {
			bucket.rpeSum += float64(log.RPE)
			bucket.rpeCount++
		}
	}

	for _, act := range acts {
		day := act.StartTimeLocal.In(loc).Format("2006-01-02")
		bucket := ensureDayAgg(seriesMap, day)
		bucket.sessions++
		km := act.DistanceM / 1000.0
		bucket.distanceKM += km
		bucket.durationSec += act.MovingTimeSec
		if km > 0 && act.MovingTimeSec > 0 {
			bucket.paceDistance += float64(act.MovingTimeSec)
		}
	}

	series := make([]TrendPoint, 0)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		key := day.In(loc).Format("2006-01-02")
		bucket := seriesMap[key]
		point := TrendPoint{Date: key}
		if bucket != nil {
			point.Sessions = bucket.sessions
			point.DistanceKM = bucket.distanceKM
			point.DurationSec = bucket.durationSec
			if bucket.distanceKM > 0 {
				point.AvgPaceSecPerKM = int(math.Round(bucket.paceDistance / bucket.distanceKM))
			}
			if bucket.rpeCount > 0 {
				point.AvgRPE = bucket.rpeSum / float64(bucket.rpeCount)
			}
		}
		series = append(series, point)
	}
	return series
}

func ensureDayAgg(m map[string]*dayAgg, key string) *dayAgg {
	bucket, ok := m[key]
	if !ok {
		bucket = &dayAgg{}
		m[key] = bucket
	}
	return bucket
}

func windowDays(window string) (int, error) {
	switch window {
	case "7d":
		return 7, nil
	case "30d":
		return 30, nil
	default:
		return 0, errors.New("window invalid")
	}
}

func classifyLevel(text string) string {
	v := strings.TrimSpace(text)
	if v == "" {
		return "unknown"
	}
	if containsAny(v, []string{"高", "过强", "偏高"}) {
		return "high"
	}
	if containsAny(v, []string{"适中", "匹配", "正常"}) {
		return "mid"
	}
	if containsAny(v, []string{"低", "不足", "偏低"}) {
		return "low"
	}
	return "unknown"
}

func extractAdviceTags(text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	tags := []string{"补水", "拉伸", "休息", "睡眠", "热身", "放松", "冰敷", "按摩"}
	out := make([]string, 0)
	for _, tag := range tags {
		if strings.Contains(trimmed, tag) {
			out = append(out, tag)
		}
	}
	return out
}

func containsAny(input string, keywords []string) bool {
	for _, k := range keywords {
		if strings.Contains(input, k) {
			return true
		}
	}
	return false
}
