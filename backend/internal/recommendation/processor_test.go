package recommendation

import (
	"context"
	"testing"
	"time"

	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/weather"
)

type fakeStore struct {
	created bool
}

func (f *fakeStore) CreateRecommendation(_ context.Context, _ storage.Recommendation) error {
	f.created = true
	return nil
}

func (f *fakeStore) GetLatestRecommendation(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{}, nil
}

func (f *fakeStore) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) {
	return storage.UserProfile{UserID: "u1", LocationLat: 1, LocationLng: 2, Country: "CN", Province: "SH", City: "SH"}, nil
}

func (f *fakeStore) GetBaselineCurrent(_ context.Context, _ string) (storage.BaselineCurrent, error) {
	return storage.BaselineCurrent{UserID: "u1"}, nil
}

func (f *fakeStore) CreateWeatherSnapshot(_ context.Context, _ storage.WeatherSnapshot) error { return nil }
func (f *fakeStore) GetLatestWeatherSnapshot(_ context.Context, _ string) (storage.WeatherSnapshot, error) {
	return storage.WeatherSnapshot{}, nil
}

func (f *fakeStore) GetRecentTrainingSummary(_ context.Context, _ string, _ time.Time, _ time.Time) (storage.TrainingLoadSummary, error) {
	return storage.TrainingLoadSummary{Sessions: 3, Distance: 10, Duration: 3600}, nil
}

func (f *fakeStore) GetLatestTrainingDiscomfort(_ context.Context, _ string) (bool, error) {
	return false, nil
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
	}, nil
}

type fakeWeather struct{}

func (fakeWeather) GetSnapshot(_ context.Context, _ weather.Location) (weather.SnapshotInput, error) {
	return weather.SnapshotInput{TemperatureC: 20}, nil
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
}
