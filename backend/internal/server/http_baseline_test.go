package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stridewise/backend/internal/storage"
)

type baselineStoreStub struct{}

func (b *baselineStoreStub) GetBaselineCurrent(_ context.Context, _ string) (storage.BaselineCurrent, error) {
	return storage.BaselineCurrent{UserID: "u1", Status: "ok"}, nil
}

func (b *baselineStoreStub) ListBaselineHistory(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.BaselineHistory, error) {
	return nil, nil
}

func (b *baselineStoreStub) ListTrainingSummaries(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingSummary, error) {
	return nil, nil
}

func (b *baselineStoreStub) CreateTrainingFeedback(_ context.Context, _ storage.TrainingFeedback) error {
	return nil
}

func TestGetBaselineCurrent(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, &baselineStoreStub{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/baseline/current?user_id=u1", nil)
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
