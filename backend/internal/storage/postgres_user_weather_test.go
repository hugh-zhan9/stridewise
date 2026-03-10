package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUserProfileUpsertAndGet(t *testing.T) {
	dsn := os.Getenv("STRIDEWISE_TEST_DSN")
	if dsn == "" {
		t.Skip("STRIDEWISE_TEST_DSN not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer pool.Close()

	store := NewPostgresStore(pool)
	profile := UserProfile{
		UserID:         "u1",
		Gender:         "male",
		Age:            28,
		HeightCM:       175,
		WeightKG:       65,
		GoalType:       "5k",
		GoalCycle:      "8w",
		GoalFrequency:  3,
		GoalPace:       "05:30",
		FitnessLevel:   "beginner",
		LocationLat:    31.2,
		LocationLng:    121.5,
		Country:        "CN",
		Province:       "SH",
		City:           "Shanghai",
		LocationSource: "manual",
	}
	if err := store.UpsertUserProfile(context.Background(), profile); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	got, err := store.GetUserProfile(context.Background(), "u1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.LocationLat != 31.2 {
		t.Fatalf("expected lat 31.2, got %v", got.LocationLat)
	}

	snapshot := WeatherSnapshot{
		UserID:            "u1",
		Date:              time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		TemperatureC:      18,
		FeelsLikeC:        18,
		Humidity:          0.4,
		WindSpeedMS:       2,
		PrecipitationProb: 0.1,
		AQI:               50,
		UVIndex:           2,
		RiskLevel:         "green",
	}
	if err := store.CreateWeatherSnapshot(context.Background(), snapshot); err != nil {
		t.Fatalf("create snapshot failed: %v", err)
	}
	gotSnap, err := store.GetWeatherSnapshot(context.Background(), "u1", snapshot.Date)
	if err != nil {
		t.Fatalf("get snapshot failed: %v", err)
	}
	if gotSnap.RiskLevel != "green" {
		t.Fatalf("expected green, got %s", gotSnap.RiskLevel)
	}
}
