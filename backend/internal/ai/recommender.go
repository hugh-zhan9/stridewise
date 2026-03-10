package ai

import (
	"context"
	"time"
)

type Recommender interface {
	Recommend(ctx context.Context, input RecommendationInput) (RecommendationOutput, error)
}

type RecommendationInput struct {
	RequestID string `json:"request_id"`
	UserProfile RecommendationUserProfile `json:"user_profile"`
	Baseline   RecommendationBaseline     `json:"baseline"`
	Weather    RecommendationWeather      `json:"weather"`
	TrainingLoad7D TrainingLoadSummary    `json:"training_load_7d"`
	Constraints RecommendationConstraints `json:"constraints"`
	CurrentTime time.Time                 `json:"current_time"`
}

type RecommendationUserProfile struct {
	UserID       string `json:"user_id"`
	AbilityLevel string `json:"ability_level"`
	GoalType     string `json:"goal_type"`
	Age          int    `json:"age"`
	WeightKG     int    `json:"weight_kg"`
	Country      string `json:"country"`
	Province     string `json:"province"`
	City         string `json:"city"`
}

type RecommendationBaseline struct {
	Status           string  `json:"status"`
	AcuteLoadSRPE    float64 `json:"acute_load_srpe"`
	ChronicLoadSRPE  float64 `json:"chronic_load_srpe"`
	ACWRSRPE         float64 `json:"acwr_srpe"`
	AcuteLoadDistance float64 `json:"acute_load_distance"`
	ChronicLoadDistance float64 `json:"chronic_load_distance"`
	ACWRDistance     float64 `json:"acwr_distance"`
	Monotony         float64 `json:"monotony"`
	Strain           float64 `json:"strain"`
	PaceAvgSecPerKM  int     `json:"pace_avg_sec_per_km"`
	PaceLowSecPerKM  int     `json:"pace_low_sec_per_km"`
	PaceHighSecPerKM int     `json:"pace_high_sec_per_km"`
}

type RecommendationWeather struct {
	TemperatureC      float64 `json:"temperature_c"`
	FeelsLikeC        float64 `json:"feels_like_c"`
	Humidity          float64 `json:"humidity"`
	WindSpeedMS       float64 `json:"wind_speed_ms"`
	PrecipitationProb float64 `json:"precipitation_prob"`
	AQI               int     `json:"aqi"`
	UVIndex           float64 `json:"uv_index"`
	RiskLevel         string  `json:"risk_level"`
}

type TrainingLoadSummary struct {
	Sessions int     `json:"sessions"`
	Distance float64 `json:"distance_km"`
	Duration int     `json:"duration_sec"`
}

type RecommendationConstraints struct {
	WeatherRisk   string `json:"weather_risk"`
	HasDiscomfort bool   `json:"has_discomfort"`
	HighLoad      bool   `json:"high_load"`
}

type RecommendationOutput struct {
	ShouldRun          bool     `json:"should_run"`
	WorkoutType        string   `json:"workout_type"`
	IntensityRange     string   `json:"intensity_range"`
	TargetVolume       string   `json:"target_volume"`
	SuggestedTimeWindow string  `json:"suggested_time_window"`
	RiskLevel          string   `json:"risk_level"`
	HydrationTip       string   `json:"hydration_tip"`
	ClothingTip        string   `json:"clothing_tip"`
	Explanation        []string `json:"explanation"`
}
