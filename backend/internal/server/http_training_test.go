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
)

type trainingStoreStub struct {
	conflict bool
}

func (s *trainingStoreStub) HasTrainingConflict(_ context.Context, _ string, _ time.Time, _ time.Time, _ string) (bool, error) {
	return s.conflict, nil
}

func (s *trainingStoreStub) CreateTrainingLog(_ context.Context, _ storage.TrainingLog) error { return nil }
func (s *trainingStoreStub) UpdateTrainingLog(_ context.Context, _ storage.TrainingLog) error { return nil }
func (s *trainingStoreStub) SoftDeleteTrainingLog(_ context.Context, _ string) error { return nil }
func (s *trainingStoreStub) ListTrainingLogs(_ context.Context, _ string, _ time.Time, _ time.Time) ([]storage.TrainingLog, error) {
	return nil, nil
}
func (s *trainingStoreStub) GetTrainingLog(_ context.Context, _ string) (storage.TrainingLog, error) {
	return storage.TrainingLog{}, nil
}

func TestCreateTrainingLog_Conflict(t *testing.T) {
	store := &trainingStoreStub{conflict: true}

	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, store, nil, nil)

	body := map[string]any{
		"user_id": "u1",
		"training_type": "轻松跑",
		"start_time": "2026-03-10 07:00:00",
		"duration": "00:30:00",
		"distance_km": 5.0,
		"pace": "06'00''",
		"rpe": 5,
		"discomfort": false,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/training/logs", bytes.NewReader(b))
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}
