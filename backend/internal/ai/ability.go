package ai

import "context"

type AbilityLeveler interface {
	EvaluateAbilityLevel(ctx context.Context, input AbilityLevelInput) (AbilityLevelOutput, error)
}

type AbilityLevelInput struct {
	UserID         string                 `json:"user_id"`
	Profile        AbilityProfileSnapshot `json:"profile"`
	TrainingSummary AbilityTrainingSummary `json:"training_summary"`
}

type AbilityProfileSnapshot struct {
	Age              int    `json:"age"`
	WeightKG         int    `json:"weight_kg"`
	RunningYears     string `json:"running_years"`
	WeeklySessions   string `json:"weekly_sessions"`
	WeeklyDistanceKM string `json:"weekly_distance_km"`
	LongestRunKM     string `json:"longest_run_km"`
}

type AbilityTrainingSummary struct {
	Sessions         int     `json:"sessions"`
	TotalDistanceKM  float64 `json:"total_distance_km"`
	TotalDurationSec int     `json:"total_duration_sec"`
	AvgPaceSecPerKM  int     `json:"avg_pace_sec_per_km"`
	AvgRPE           float64 `json:"avg_rpe"`
	SRPELoad         float64 `json:"srpe_load"`
}

type AbilityLevelOutput struct {
	AbilityLevel string `json:"ability_level"`
	Reason       string `json:"reason,omitempty"`
}
