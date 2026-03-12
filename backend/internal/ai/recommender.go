package ai

import (
	"context"
	"time"
)

type Recommender interface {
	Recommend(ctx context.Context, input RecommendationInput) (RecommendationOutput, error)
}

type RecommendationInput struct {
	RequestID              string                          `json:"request_id"`
	UserProfile            RecommendationUserProfile       `json:"user_profile"`
	Baseline               RecommendationBaseline          `json:"baseline"`
	Weather                RecommendationWeather           `json:"weather"`
	TrainingLoad7D         TrainingLoadSummary             `json:"training_load_7d"`
	Constraints            RecommendationConstraints       `json:"constraints"`
	CurrentTime            time.Time                       `json:"current_time"`
	RecoveryStatus         string                          `json:"recovery_status"`
	RecoveryScore          *RecommendationRecoveryScore    `json:"recovery_score,omitempty"`
	LatestTrainingFeedback *RecommendationTrainingFeedback `json:"latest_training_feedback,omitempty"`
	Personalization        *RecommendationPersonalization  `json:"personalization,omitempty"`
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
	Status              string  `json:"status"`
	AcuteLoadSRPE       float64 `json:"acute_load_srpe"`
	ChronicLoadSRPE     float64 `json:"chronic_load_srpe"`
	ACWRSRPE            float64 `json:"acwr_srpe"`
	AcuteLoadDistance   float64 `json:"acute_load_distance"`
	ChronicLoadDistance float64 `json:"chronic_load_distance"`
	ACWRDistance        float64 `json:"acwr_distance"`
	Monotony            float64 `json:"monotony"`
	Strain              float64 `json:"strain"`
	PaceAvgSecPerKM     int     `json:"pace_avg_sec_per_km"`
	PaceLowSecPerKM     int     `json:"pace_low_sec_per_km"`
	PaceHighSecPerKM    int     `json:"pace_high_sec_per_km"`
}

type RecommendationWeather struct {
	TemperatureC      float64                  `json:"temperature_c"`
	FeelsLikeC        float64                  `json:"feels_like_c"`
	Humidity          float64                  `json:"humidity"`
	WindSpeedMS       float64                  `json:"wind_speed_ms"`
	PrecipitationProb float64                  `json:"precipitation_prob"`
	AQI               int                      `json:"aqi"`
	UVIndex           float64                  `json:"uv_index"`
	RiskLevel         string                   `json:"risk_level"`
	Forecasts         []RecommendationForecast `json:"forecasts"`
}

type RecommendationForecast struct {
	Date             string   `json:"date"`
	TempMaxC         *float64 `json:"temp_max_c"`
	TempMinC         *float64 `json:"temp_min_c"`
	Humidity         *float64 `json:"humidity"`
	PrecipMM         *float64 `json:"precip_mm"`
	PressureHPA      *float64 `json:"pressure_hpa"`
	VisibilityKM     *float64 `json:"visibility_km"`
	CloudPct         *float64 `json:"cloud_pct"`
	UVIndex          *float64 `json:"uv_index"`
	AQI              int      `json:"aqi"`
	AQISource        string   `json:"aqi_source"`
	TextDay          *string  `json:"text_day"`
	TextNight        *string  `json:"text_night"`
	IconDay          *string  `json:"icon_day"`
	IconNight        *string  `json:"icon_night"`
	Wind360Day       *int     `json:"wind360_day"`
	WindDirDay       *string  `json:"wind_dir_day"`
	WindScaleDay     *string  `json:"wind_scale_day"`
	WindSpeedDayMS   *float64 `json:"wind_speed_day_ms"`
	Wind360Night     *int     `json:"wind360_night"`
	WindDirNight     *string  `json:"wind_dir_night"`
	WindScaleNight   *string  `json:"wind_scale_night"`
	WindSpeedNightMS *float64 `json:"wind_speed_night_ms"`
	SunriseTime      *string  `json:"sunrise_time"`
	SunsetTime       *string  `json:"sunset_time"`
	MoonriseTime     *string  `json:"moonrise_time"`
	MoonsetTime      *string  `json:"moonset_time"`
	MoonPhase        *string  `json:"moon_phase"`
	MoonPhaseIcon    *string  `json:"moon_phase_icon"`
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

type RecommendationTrainingFeedback struct {
	SourceType string                         `json:"source_type"`
	SourceID   string                         `json:"source_id"`
	CreatedAt  string                         `json:"created_at"`
	Content    string                         `json:"content"`
	Summary    *RecommendationTrainingSummary `json:"summary"`
}

type RecommendationTrainingSummary struct {
	CompletionRate   string `json:"completion_rate"`
	IntensityMatch   string `json:"intensity_match"`
	RecoveryAdvice   string `json:"recovery_advice"`
	AnomalyNotes     string `json:"anomaly_notes"`
	PerformanceNotes string `json:"performance_notes"`
	NextSuggestion   string `json:"next_suggestion"`
}

type RecommendationPersonalization struct {
	IntensityBias    float64            `json:"intensity_bias"`
	VolumeMultiplier float64            `json:"volume_multiplier"`
	TypePreference   map[string]float64 `json:"type_preference"`
}

type RecommendationRecoveryScore struct {
	OverallScore        float64 `json:"overall_score"`
	FatigueScore        float64 `json:"fatigue_score"`
	RecoveryScore       float64 `json:"recovery_score"`
	ACWRComponent       float64 `json:"acwr_component"`
	MonotonyComponent   float64 `json:"monotony_component"`
	StrainComponent     float64 `json:"strain_component"`
	DiscomfortPenalty   float64 `json:"discomfort_penalty"`
	RestingHRPenalty    float64 `json:"resting_hr_penalty"`
	RecoveryStatus      string  `json:"recovery_status"`
}

type RecommendationOutput struct {
	ShouldRun           bool                               `json:"should_run"`
	WorkoutType         string                             `json:"workout_type"`
	IntensityRange      string                             `json:"intensity_range"`
	TargetVolume        string                             `json:"target_volume"`
	SuggestedTimeWindow string                             `json:"suggested_time_window"`
	RiskLevel           string                             `json:"risk_level"`
	HydrationTip        string                             `json:"hydration_tip"`
	ClothingTip         string                             `json:"clothing_tip"`
	Explanation         []string                           `json:"explanation"`
	AlternativeWorkouts []RecommendationAlternativeWorkout `json:"alternative_workouts"`
}

type RecommendationAlternativeWorkout struct {
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	DurationMin int      `json:"duration_min"`
	Intensity   string   `json:"intensity"`
	Tips        []string `json:"tips"`
}
