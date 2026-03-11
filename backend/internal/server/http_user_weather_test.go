package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/weather"
)

type fakeStore struct {
	profile storage.UserProfile
	snap    storage.WeatherSnapshot
}

func (f *fakeStore) UpsertUserProfile(_ context.Context, p storage.UserProfile) error {
	f.profile = p
	return nil
}

func (f *fakeStore) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) {
	return f.profile, nil
}

func (f *fakeStore) CreateWeatherSnapshot(_ context.Context, s storage.WeatherSnapshot) error {
	f.snap = s
	return nil
}

func (f *fakeStore) GetWeatherSnapshot(_ context.Context, _ string, _ time.Time) (storage.WeatherSnapshot, error) {
	return f.snap, nil
}

func TestCreateUserProfile_RequiresLocation(t *testing.T) {
	store := &fakeStore{}
	provider := weather.NewMockProvider(weather.SnapshotInput{TemperatureC: 20})

	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, store, store, provider, nil, nil, nil, nil)

	body := map[string]any{
		"user_id": "u1",
		"gender": "male",
		"age": 20,
		"height_cm": 175,
		"weight_kg": 65,
		"goal_type": "5k",
		"goal_cycle": "8w",
		"goal_frequency": 3,
		"goal_pace": "05:30",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/user/profile", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateUserProfile_PersistsQuestionnaire(t *testing.T) {
	store := &fakeStore{}
	provider := weather.NewMockProvider(weather.SnapshotInput{TemperatureC: 20})

	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, store, store, provider, nil, nil, nil, nil)

	body := map[string]any{
		"user_id": "u1",
		"gender": "male",
		"age": 20,
		"height_cm": 175,
		"weight_kg": 65,
		"goal_type": "5k",
		"goal_cycle": "8w",
		"goal_frequency": 3,
		"goal_pace": "05:30",
		"location_lat": 31.2,
		"location_lng": 121.5,
		"country": "CN",
		"province": "SH",
		"city": "Shanghai",
		"location_source": "manual",
		"running_years": "1-3",
		"weekly_sessions": "2-3",
		"weekly_distance_km": "5-15",
		"longest_run_km": "10",
		"recent_discomfort": "no",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/user/profile", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if store.profile.RunningYears != "1-3" {
		t.Fatalf("expected running_years 1-3, got %s", store.profile.RunningYears)
	}
	if store.profile.WeeklySessions != "2-3" {
		t.Fatalf("expected weekly_sessions 2-3, got %s", store.profile.WeeklySessions)
	}
	if store.profile.WeeklyDistanceKM != "5-15" {
		t.Fatalf("expected weekly_distance_km 5-15, got %s", store.profile.WeeklyDistanceKM)
	}
	if store.profile.LongestRunKM != "10" {
		t.Fatalf("expected longest_run_km 10, got %s", store.profile.LongestRunKM)
	}
	if store.profile.RecentDiscomfort != "no" {
		t.Fatalf("expected recent_discomfort no, got %s", store.profile.RecentDiscomfort)
	}
}

func TestCreateUserProfile_RejectsManualAbilityLevel(t *testing.T) {
	store := &fakeStore{}
	provider := weather.NewMockProvider(weather.SnapshotInput{TemperatureC: 20})

	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, store, store, provider, nil, nil, nil, nil)

	body := map[string]any{
		"user_id": "u1",
		"gender": "male",
		"age": 20,
		"height_cm": 175,
		"weight_kg": 65,
		"goal_type": "5k",
		"goal_cycle": "8w",
		"goal_frequency": 3,
		"goal_pace": "05:30",
		"location_lat": 31.2,
		"location_lng": 121.5,
		"country": "CN",
		"province": "SH",
		"city": "Shanghai",
		"location_source": "manual",
		"running_years": "1-3",
		"weekly_sessions": "2-3",
		"weekly_distance_km": "5-15",
		"longest_run_km": "10",
		"recent_discomfort": "no",
		"fitness_level": "beginner",
		"ability_level": "advanced",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/user/profile", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
