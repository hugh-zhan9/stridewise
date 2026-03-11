package recommendation

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/weather"
)

type fakeStore struct {
	created bool
	lastRec storage.Recommendation
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
	return storage.UserProfile{UserID: "u1", LocationLat: 1, LocationLng: 2, Country: "CN", Province: "SH", City: "SH"}, nil
}

func (f *fakeStore) GetBaselineCurrent(_ context.Context, _ string) (storage.BaselineCurrent, error) {
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
	return storage.TrainingLoadSummary{Sessions: 3, Distance: 10, Duration: 3600}, nil
}

func (f *fakeStore) GetLatestTrainingDiscomfort(_ context.Context, _ string) (bool, error) {
	return false, nil
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
