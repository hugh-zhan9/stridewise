package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stridewise/backend/internal/storage"
)

type baselineStoreStub struct{}

type baselineStoreStubWithSummary struct{}

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

func (b *baselineStoreStubWithSummary) GetBaselineCurrent(_ context.Context, _ string) (storage.BaselineCurrent, error) {
	return storage.BaselineCurrent{UserID: "u1", Status: "ok"}, nil
}

func (b *baselineStoreStubWithSummary) ListBaselineHistory(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.BaselineHistory, error) {
	return nil, nil
}

func (b *baselineStoreStubWithSummary) ListTrainingSummaries(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingSummary, error) {
	return []storage.TrainingSummary{{
		SummaryID:        "s1",
		UserID:           "u1",
		SourceType:       "log",
		SourceID:         "log-1",
		LogID:            "log-1",
		CompletionRate:   "ok",
		IntensityMatch:   "ok",
		RecoveryAdvice:   "ok",
		AnomalyNotes:     "ok",
		PerformanceNotes: "ok",
		NextSuggestion:   "ok",
	}}, nil
}

func (b *baselineStoreStubWithSummary) CreateTrainingFeedback(_ context.Context, _ storage.TrainingFeedback) error {
	return nil
}

func TestGetBaselineCurrent(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, &baselineStoreStub{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/baseline/current?user_id=u1", nil)
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateTrainingFeedbackWithSource(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, &baselineStoreStub{}, nil, nil, nil)

	body := `{"user_id":"u1","source_type":"activity","source_id":"1","content":"ok"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/training/feedback", strings.NewReader(body))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListTrainingSummariesResponseSnakeCase(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, &baselineStoreStubWithSummary{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/training/summaries?user_id=u1", nil)
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "summary_id") || !strings.Contains(body, "source_type") || !strings.Contains(body, "source_id") {
		t.Fatalf("expected snake_case fields, got %s", body)
	}
}

func TestResponseEnvelopeSuccess(t *testing.T) {
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, &baselineStoreStub{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/training/summaries?user_id=u1", nil)
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "\"data\"") || !strings.Contains(body, "\"meta\"") {
		t.Fatalf("expected envelope, got %s", body)
	}
}
