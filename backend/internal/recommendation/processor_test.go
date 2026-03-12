package recommendation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/weather"
)

type fakeStore struct {
	created        bool
	lastRec        storage.Recommendation
	lastRecovery   storage.RecoveryScore
	profile        storage.UserProfile
	baseline       storage.BaselineCurrent
	loadSummary    storage.TrainingLoadSummary
	hasDiscomfort  bool
	latestFeedback storage.TrainingFeedback
	latestSummary  storage.TrainingSummary
	personalParams storage.UserPersonalizationParams
}

func (f *fakeStore) CreateRecommendation(_ context.Context, rec storage.Recommendation) error {
	f.created = true
	f.lastRec = rec
	return nil
}

func (f *fakeStore) CreateRecoveryScore(_ context.Context, score storage.RecoveryScore) error {
	f.lastRecovery = score
	return nil
}

func (f *fakeStore) GetLatestRecommendation(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{}, nil
}

func (f *fakeStore) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) {
	if f.profile.UserID != "" {
		return f.profile, nil
	}
	return storage.UserProfile{UserID: "u1", LocationLat: 1, LocationLng: 2, Country: "CN", Province: "SH", City: "SH", AbilityLevel: "beginner"}, nil
}

func (f *fakeStore) GetBaselineCurrent(_ context.Context, _ string) (storage.BaselineCurrent, error) {
	if f.baseline.UserID != "" {
		return f.baseline, nil
	}
	return storage.BaselineCurrent{
		UserID:       "u1",
		ACWRSRPE:     1.6,
		ACWRDistance: 1.2,
		Monotony:     1.0,
	}, nil
}

func (f *fakeStore) CreateWeatherSnapshot(_ context.Context, _ storage.WeatherSnapshot) error {
	return nil
}
func (f *fakeStore) GetLatestWeatherSnapshot(_ context.Context, _ string) (storage.WeatherSnapshot, error) {
	return storage.WeatherSnapshot{}, nil
}

func (f *fakeStore) UpsertWeatherForecasts(_ context.Context, _ []storage.WeatherForecast) error {
	return nil
}

func (f *fakeStore) GetRecentTrainingSummary(_ context.Context, _ string, _ time.Time, _ time.Time) (storage.TrainingLoadSummary, error) {
	if f.loadSummary.Sessions != 0 || f.loadSummary.Distance != 0 || f.loadSummary.Duration != 0 {
		return f.loadSummary, nil
	}
	return storage.TrainingLoadSummary{Sessions: 3, Distance: 10, Duration: 3600}, nil
}

func (f *fakeStore) GetLatestTrainingDiscomfort(_ context.Context, _ string) (bool, error) {
	return f.hasDiscomfort, nil
}

func (f *fakeStore) CreateRecommendationFeedback(_ context.Context, _ storage.RecommendationFeedback) error {
	return nil
}

func (f *fakeStore) GetLatestTrainingFeedback(_ context.Context, _ string) (storage.TrainingFeedback, error) {
	return f.latestFeedback, nil
}

func (f *fakeStore) GetTrainingSummaryBySource(_ context.Context, _ string, _ string) (storage.TrainingSummary, error) {
	return f.latestSummary, nil
}

func (f *fakeStore) GetUserPersonalizationParams(_ context.Context, _ string) (storage.UserPersonalizationParams, error) {
	return f.personalParams, nil
}

type fakeAI struct{}

func (fakeAI) Recommend(_ context.Context, _ ai.RecommendationInput) (ai.RecommendationOutput, error) {
	return ai.RecommendationOutput{
		ShouldRun:           true,
		WorkoutType:         "easy",
		IntensityRange:      "low",
		TargetVolume:        "5k",
		SuggestedTimeWindow: "morning",
		RiskLevel:           "green",
		HydrationTip:        "water",
		ClothingTip:         "light",
		Explanation:         []string{"a", "b"},
		AlternativeWorkouts: []ai.RecommendationAlternativeWorkout{{
			Type:        "treadmill",
			Title:       "室内跑步机轻松跑",
			DurationMin: 30,
			Intensity:   "low",
		}},
	}, nil
}

type fakeWeather struct{}

func (fakeWeather) GetSnapshot(_ context.Context, _ weather.Location) (weather.SnapshotInput, error) {
	return weather.SnapshotInput{TemperatureC: 20, FeelsLikeC: 41}, nil
}

func (fakeWeather) GetForecast(_ context.Context, _ weather.Location) ([]weather.ForecastInput, error) {
	tempMax := 25.0
	aqiLocal := 80
	aqiSource := "local"
	return []weather.ForecastInput{{
		Date:      time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
		TempMaxC:  &tempMax,
		AQILocal:  &aqiLocal,
		AQISource: &aqiSource,
	}}, nil
}

type safeWeather struct{}

func (safeWeather) GetSnapshot(_ context.Context, _ weather.Location) (weather.SnapshotInput, error) {
	return weather.SnapshotInput{
		TemperatureC:      20,
		FeelsLikeC:        20,
		Humidity:          0.5,
		WindSpeedMS:       1,
		PrecipitationProb: 0,
		AQI:               50,
		UVIndex:           1,
	}, nil
}

func (safeWeather) GetForecast(_ context.Context, _ weather.Location) ([]weather.ForecastInput, error) {
	aqiQAQI := 60
	aqiSource := "qaqi"
	return []weather.ForecastInput{{
		Date:      time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
		AQIQAQI:   &aqiQAQI,
		AQISource: &aqiSource,
	}}, nil
}

type missingAQIWeather struct{}

func (missingAQIWeather) GetSnapshot(_ context.Context, _ weather.Location) (weather.SnapshotInput, error) {
	return weather.SnapshotInput{TemperatureC: 20, FeelsLikeC: 41}, nil
}

func (missingAQIWeather) GetForecast(_ context.Context, _ weather.Location) ([]weather.ForecastInput, error) {
	return []weather.ForecastInput{{
		Date: time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
	}}, nil
}

func TestGenerateRecommendation(t *testing.T) {
	store := &fakeStore{}
	p := NewProcessor(store, fakeWeather{}, fakeAI{})
	p.now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.created {
		t.Fatalf("expected create recommendation")
	}

	var input ai.RecommendationInput
	if err := json.Unmarshal(store.lastRec.InputJSON, &input); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if len(input.Weather.Forecasts) != 1 {
		t.Fatalf("expected 1 forecast in input")
	}
	if input.RecoveryStatus != "red" {
		t.Fatalf("expected recovery_status red")
	}

	var output map[string]any
	if err := json.Unmarshal(store.lastRec.OutputJSON, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if _, ok := output["AlternativeWorkouts"]; !ok {
		t.Fatalf("expected AlternativeWorkouts in output")
	}
}

func TestGenerateRecommendation_ForecastAQIMissingFallback(t *testing.T) {
	store := &fakeStore{}
	p := NewProcessor(store, missingAQIWeather{}, fakeAI{})
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.lastRec.IsFallback {
		t.Fatalf("expected fallback when forecast aqi missing")
	}
}

func TestGenerateRecommendation_IncludesLatestTrainingFeedback(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID:       "u1",
			LocationLat:  1,
			LocationLng:  2,
			Country:      "CN",
			Province:     "SH",
			City:         "SH",
			AbilityLevel: "beginner",
		},
		latestFeedback: storage.TrainingFeedback{
			UserID:     "u1",
			SourceType: "log",
			SourceID:   "log-1",
			Content:    "太累了",
			CreatedAt:  time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC),
		},
		latestSummary: storage.TrainingSummary{
			SummaryID:      "s1",
			UserID:         "u1",
			SourceType:     "log",
			SourceID:       "log-1",
			CompletionRate: "ok",
		},
	}
	p := NewProcessor(store, safeWeather{}, fakeAI{})
	p.now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	var input ai.RecommendationInput
	if err := json.Unmarshal(store.lastRec.InputJSON, &input); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if input.LatestTrainingFeedback == nil {
		t.Fatalf("expected latest_training_feedback")
	}
	if input.LatestTrainingFeedback.Content == "" {
		t.Fatalf("expected feedback content")
	}
	if input.LatestTrainingFeedback.Summary == nil {
		t.Fatalf("expected summary")
	}
	if input.LatestTrainingFeedback.Summary.CompletionRate == "" {
		t.Fatalf("expected completion_rate")
	}
}

func TestGenerateRecommendation_ConservativeTemplate(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID:           "u1",
			LocationLat:      1,
			LocationLng:      2,
			Country:          "CN",
			Province:         "SH",
			City:             "SH",
			AbilityLevel:     "beginner",
			RunningYears:     "1-3",
			WeeklySessions:   "2-3",
			WeeklyDistanceKM: "5-15",
			LongestRunKM:     "10",
			RecentDiscomfort: "no",
		},
		baseline: storage.BaselineCurrent{
			UserID:         "u1",
			Status:         "insufficient_data",
			DataSessions7d: 0,
		},
		loadSummary: storage.TrainingLoadSummary{Sessions: 0},
	}
	p := NewProcessor(store, safeWeather{}, fakeAI{})
	p.now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	var output RecommendationOutput
	if err := json.Unmarshal(store.lastRec.OutputJSON, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.WorkoutType != "easy_run" {
		t.Fatalf("expected workout_type easy_run, got %s", output.WorkoutType)
	}
	if len(output.Explanation) == 0 || output.Explanation[0] == "" {
		t.Fatalf("expected explanation for conservative template")
	}
	found := false
	for _, line := range output.Explanation {
		if strings.Contains(line, "保守模板") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected explanation contains 保守模板")
	}
}

func TestGenerateRecommendation_IncludesPersonalizationParams(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID:       "u1",
			LocationLat:  1,
			LocationLng:  2,
			Country:      "CN",
			Province:     "SH",
			City:         "SH",
			AbilityLevel: "beginner",
		},
		personalParams: storage.UserPersonalizationParams{
			UserID:           "u1",
			IntensityBias:    -0.1,
			VolumeMultiplier: 0.9,
			TypePreference: map[string]float64{
				"轻松跑": 1.2,
				"有氧跑": 1.0,
				"间歇跑": 0.8,
				"长距离": 0.9,
			},
		},
	}
	p := NewProcessor(store, safeWeather{}, fakeAI{})
	p.now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	var input ai.RecommendationInput
	if err := json.Unmarshal(store.lastRec.InputJSON, &input); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if input.Personalization == nil {
		t.Fatalf("expected personalization in input")
	}
	if input.Personalization.VolumeMultiplier != 0.9 {
		t.Fatalf("expected volume_multiplier 0.9")
	}
}

func TestGenerateRecommendation_IncludesRecoveryScore(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID:       "u1",
			LocationLat:  1,
			LocationLng:  2,
			Country:      "CN",
			Province:     "SH",
			City:         "SH",
			AbilityLevel: "beginner",
		},
		baseline: storage.BaselineCurrent{
			UserID:       "u1",
			ACWRSRPE:     1.55,
			ACWRDistance: 1.45,
			Monotony:     2.05,
			Strain:       520,
		},
	}
	p := NewProcessor(store, safeWeather{}, fakeAI{})
	p.now = func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	var input ai.RecommendationInput
	if err := json.Unmarshal(store.lastRec.InputJSON, &input); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if input.RecoveryScore == nil {
		t.Fatalf("expected recovery_score")
	}
	if input.RecoveryScore.OverallScore <= 0 {
		t.Fatalf("expected positive overall score")
	}
	if store.lastRecovery.UserID != "u1" {
		t.Fatalf("expected recovery score persisted")
	}
}

func TestGenerateRecommendation_EngineVersionWithStrategy(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID:       "u1",
			LocationLat:  1,
			LocationLng:  2,
			Country:      "CN",
			Province:     "SH",
			City:         "SH",
			AbilityLevel: "beginner",
		},
	}
	p := NewProcessor(store, safeWeather{}, fakeAI{})
	p.SetDecisionStrategy("ai_primary")
	p.now = func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if store.lastRec.EngineVersion != "v2-ai_primary" {
		t.Fatalf("expected engine_version v2-ai_primary, got %s", store.lastRec.EngineVersion)
	}
}

func TestGenerateRecommendation_RuleOnlyStrategy(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID:       "u1",
			LocationLat:  1,
			LocationLng:  2,
			Country:      "CN",
			Province:     "SH",
			City:         "SH",
			AbilityLevel: "beginner",
		},
	}
	p := NewProcessor(store, safeWeather{}, fakeAI{})
	p.SetDecisionStrategy("rule_only")
	p.now = func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if store.lastRec.EngineVersion != "v2-rule_only" {
		t.Fatalf("expected engine_version v2-rule_only, got %s", store.lastRec.EngineVersion)
	}
	if !store.lastRec.IsFallback {
		t.Fatalf("expected fallback=true for rule_only strategy")
	}
}

func TestGenerateRecommendation_ConservativeTemplateDiscomfort(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID:           "u1",
			LocationLat:      1,
			LocationLng:      2,
			Country:          "CN",
			Province:         "SH",
			City:             "SH",
			AbilityLevel:     "beginner",
			RunningYears:     "1-3",
			WeeklySessions:   "2-3",
			WeeklyDistanceKM: "5-15",
			LongestRunKM:     "10",
			RecentDiscomfort: "yes",
		},
		baseline: storage.BaselineCurrent{
			UserID:         "u1",
			Status:         "insufficient_data",
			DataSessions7d: 0,
		},
		loadSummary: storage.TrainingLoadSummary{Sessions: 0},
	}
	p := NewProcessor(store, fakeWeather{}, fakeAI{})
	p.now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }
	if _, err := p.Generate(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	var output RecommendationOutput
	if err := json.Unmarshal(store.lastRec.OutputJSON, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.ShouldRun {
		t.Fatalf("expected should_run false when recent_discomfort yes")
	}
	if output.RiskLevel != "red" {
		t.Fatalf("expected risk_level red, got %s", output.RiskLevel)
	}
}
