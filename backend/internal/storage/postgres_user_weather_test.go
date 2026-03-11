package storage

import (
	"context"
	"os"
	"reflect"
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
	setStringField(t, &profile, "AbilityLevel", "beginner")
	setStringField(t, &profile, "RunningYears", "1-3")
	setStringField(t, &profile, "WeeklySessions", "2-3")
	setStringField(t, &profile, "WeeklyDistanceKM", "5-15")
	setStringField(t, &profile, "LongestRunKM", "10")
	setStringField(t, &profile, "RecentDiscomfort", "no")
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
	assertStringField(t, got, "AbilityLevel", "beginner")
	assertStringField(t, got, "RunningYears", "1-3")
	assertStringField(t, got, "WeeklySessions", "2-3")
	assertStringField(t, got, "WeeklyDistanceKM", "5-15")
	assertStringField(t, got, "LongestRunKM", "10")
	assertStringField(t, got, "RecentDiscomfort", "no")

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

func setStringField(t *testing.T, target any, name string, value string) {
	t.Helper()
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		t.Fatalf("target must be pointer to struct")
	}
	field := v.Elem().FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("missing field %s", name)
	}
	if field.Kind() != reflect.String {
		t.Fatalf("field %s not string", name)
	}
	if !field.CanSet() {
		t.Fatalf("field %s not settable", name)
	}
	field.SetString(value)
}

func assertStringField(t *testing.T, target any, name string, expected string) {
	t.Helper()
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		t.Fatalf("target must be struct")
	}
	field := v.FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("missing field %s", name)
	}
	if field.Kind() != reflect.String {
		t.Fatalf("field %s not string", name)
	}
	if field.String() != expected {
		t.Fatalf("expected %s %s, got %s", name, expected, field.String())
	}
}
