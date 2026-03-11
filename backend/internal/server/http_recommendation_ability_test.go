package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"stridewise/backend/internal/storage"
)

type recStoreStub struct {
	profile storage.UserProfile
	jobID   string
	created bool
}

func (r *recStoreStub) UpsertUserProfile(_ context.Context, p storage.UserProfile) error {
	r.profile = p
	return nil
}

func (r *recStoreStub) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) {
	return r.profile, nil
}

func (r *recStoreStub) CreateAsyncJob(_ context.Context, job storage.AsyncJob) error {
	r.jobID = job.JobID
	r.created = true
	return nil
}

func (r *recStoreStub) UpdateAsyncJobStatus(_ context.Context, _ string, _ string, _ int, _ string) error {
	return nil
}

func (r *recStoreStub) FindActiveAsyncJob(_ context.Context, _ string, _ string) (storage.AsyncJob, error) {
	return storage.AsyncJob{}, errors.New("not found")
}

type recServiceStubAbility struct{}

func (recServiceStubAbility) Generate(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{}, nil
}

func (recServiceStubAbility) GetLatest(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{}, nil
}

func (recServiceStubAbility) Feedback(_ context.Context, _ string, _ string, _ string, _ string) error {
	return nil
}

type recServiceFallbackStub struct{}

func (recServiceFallbackStub) Generate(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{
		RecID:        "r1",
		UserID:       "u1",
		OverrideJSON: []byte(`{}`),
		IsFallback:   true,
	}, nil
}

func (recServiceFallbackStub) GetLatest(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{
		RecID:        "r1",
		UserID:       "u1",
		OverrideJSON: []byte(`{}`),
		IsFallback:   true,
	}, nil
}

func (recServiceFallbackStub) Feedback(_ context.Context, _ string, _ string, _ string, _ string) error {
	return nil
}

func TestGenerateRecommendation_BlocksWithoutAbilityLevel(t *testing.T) {
	store := &recStoreStub{profile: storage.UserProfile{UserID: "u1"}}
	enqueuer := &abilityEnqueuerStub{}

	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, store, nil, nil, nil, store, nil, enqueuer, recServiceStubAbility{}, nil)

	body := map[string]any{"user_id": "u1"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/recommendations/generate", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if !enqueuer.called {
		t.Fatalf("expected ability level job enqueued")
	}
	if !strings.Contains(rec.Body.String(), "ability_level_not_ready") {
		t.Fatalf("expected ability_level_not_ready message")
	}
}

func TestGenerateRecommendationFallbackMeta(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, profileStoreStub{}, nil, nil, nil, nil, nil, nil, recServiceFallbackStub{}, nil)
	body := map[string]any{"user_id": "u1"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/recommendations/generate", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	meta, ok := resp["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta in response")
	}
	if meta["fallback_reason"] != "ai_unavailable" {
		t.Fatalf("expected ai_unavailable fallback, got %v", meta["fallback_reason"])
	}
	if meta["confidence"] != 0.4 {
		t.Fatalf("expected confidence 0.4, got %v", meta["confidence"])
	}
}
