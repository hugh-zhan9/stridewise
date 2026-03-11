package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stridewise/backend/internal/storage"
)

type recServiceStub struct{}

func (recServiceStub) Generate(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{RecID: "r1"}, nil
}

func (recServiceStub) GetLatest(_ context.Context, _ string) (storage.Recommendation, error) {
	return storage.Recommendation{RecID: "r1"}, nil
}

func (recServiceStub) Feedback(_ context.Context, _ string, _ string, _ string, _ string) error {
	return nil
}

type profileStoreStub struct{}

func (profileStoreStub) UpsertUserProfile(_ context.Context, _ storage.UserProfile) error {
	return nil
}

func (profileStoreStub) GetUserProfile(_ context.Context, _ string) (storage.UserProfile, error) {
	return storage.UserProfile{UserID: "u1", AbilityLevel: "beginner"}, nil
}

func TestGenerateRecommendation(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, profileStoreStub{}, nil, nil, nil, nil, nil, nil, recServiceStub{}, nil)
	body := map[string]any{"user_id": "u1"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/recommendations/generate", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
