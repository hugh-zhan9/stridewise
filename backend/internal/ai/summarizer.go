package ai

import (
	"context"
	"time"
)

type Summarizer interface {
	Summarize(ctx context.Context, input SummaryInput) (SummaryOutput, error)
}

type SummaryInput struct {
	UserID             string           `json:"user_id"`
	LogID              string           `json:"log_id"`
	TrainingType       string           `json:"training_type"`
	TrainingTypeCustom string           `json:"training_type_custom"`
	StartTime          time.Time        `json:"start_time"`
	DurationSec        int              `json:"duration_sec"`
	DistanceKM         float64          `json:"distance_km"`
	PaceSecPerKM       int              `json:"pace_sec_per_km"`
	RPE                int              `json:"rpe"`
	Discomfort         bool             `json:"discomfort"`
	Baseline           BaselineSnapshot `json:"baseline"`
}

type BaselineSnapshot struct {
	DataSessions7d    int     `json:"data_sessions_7d"`
	AcuteLoadSRPE     float64 `json:"acute_load_srpe"`
	ChronicLoadSRPE   float64 `json:"chronic_load_srpe"`
	ACWRSRPE          float64 `json:"acwr_srpe"`
	AcuteLoadDistance float64 `json:"acute_load_distance"`
	ChronicLoadDistance float64 `json:"chronic_load_distance"`
	ACWRDistance      float64 `json:"acwr_distance"`
	Monotony          float64 `json:"monotony"`
	Strain            float64 `json:"strain"`
	PaceAvgSecPerKM   int     `json:"pace_avg_sec_per_km"`
	PaceLowSecPerKM   int     `json:"pace_low_sec_per_km"`
	PaceHighSecPerKM  int     `json:"pace_high_sec_per_km"`
	Status            string  `json:"status"`
}

type SummaryOutput struct {
	CompletionRate   string `json:"completion_rate"`
	IntensityMatch   string `json:"intensity_match"`
	RecoveryAdvice   string `json:"recovery_advice"`
	AnomalyNotes     string `json:"anomaly_notes"`
	PerformanceNotes string `json:"performance_notes"`
	NextSuggestion   string `json:"next_suggestion"`
}
