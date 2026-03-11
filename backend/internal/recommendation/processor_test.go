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
	created bool
	lastRec storage.Recommendation
	profile storage.UserProfile
	baseline storage.BaselineCurrent
	loadSummary storage.TrainingLoadSummary
	hasDiscomfort bool
}

func (f *fakeStore) CreateRecommendation(_ context.Context, rec storage.Recommendation) error {
	f.created = true
	f.lastRec = rec
	return nil
}

func (f *fakeStore) GetLatestRecommendation(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{}, nil
}

func (f *fakeStore) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) {
	if f.profile.UserID != "" {
		return f.profile, nil
	}
	return storage.UserProfile{UserID: "u1", LocationLat: 1, LocationLng: 2, Country: "CN", Province: "SH", City: "SH"}, nil
}

func (f *fakeStore) GetBaselineCurrent(_ context.Context, _ string) (storage.BaselineCurrent, error) {
	if f.baseline.UserID != "" {
		return f.baseline, nil
	}
	return storage.BaselineCurrent{
		UserID:          "u1",
		ACWRSRPE:        1.6,
		ACWRDistance:    1.2,
		Monotony:        1.0,
	}, nil
}

func (f *fakeStore) CreateWeatherSnapshot(_ context.Context, _ storage.WeatherSnapshot) error { return nil }
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
	return []weather.ForecastInput{{
		Date:     time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
		TempMaxC: &tempMax,
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
	return nil, nil
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

func TestGenerateRecommendation_ConservativeTemplate(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID: "u1",
			LocationLat: 1,
			LocationLng: 2,
			Country: "CN",
			Province: "SH",
			City: "SH",
			RunningYears: "1-3",
			WeeklySessions: "2-3",
			WeeklyDistanceKM: "5-15",
			LongestRunKM: "10",
			RecentDiscomfort: "no",
		},
		baseline: storage.BaselineCurrent{
			UserID: "u1",
			Status: "insufficient_data",
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

func TestGenerateRecommendation_ConservativeTemplateDiscomfort(t *testing.T) {
	store := &fakeStore{
		profile: storage.UserProfile{
			UserID: "u1",
			LocationLat: 1,
			LocationLng: 2,
			Country: "CN",
			Province: "SH",
			City: "SH",
			RunningYears: "1-3",
			WeeklySessions: "2-3",
			WeeklyDistanceKM: "5-15",
			LongestRunKM: "10",
			RecentDiscomfort: "yes",
		},
		baseline: storage.BaselineCurrent{
			UserID: "u1",
			Status: "insufficient_data",
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
