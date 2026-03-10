package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBaselineStore(t *testing.T) {
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
	current := BaselineCurrent{
		UserID:         "u1",
		ComputedAt:     time.Now().UTC(),
		DataSessions7d: 3,
		Status:         "ok",
	}
	if err := store.UpsertBaselineCurrent(context.Background(), current); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	history := BaselineHistory{
		BaselineID:  "b1",
		UserID:      "u1",
		ComputedAt:  time.Now().UTC(),
		TriggerType: "training_create",
		TriggerRef:  "log-1",
		Status:      "ok",
	}
	if err := store.CreateBaselineHistory(context.Background(), history); err != nil {
		t.Fatalf("history failed: %v", err)
	}
}

func TestGetTrainingSummary(t *testing.T) {
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
	summary := TrainingSummary{
		SummaryID:        "s1",
		UserID:           "u1",
		LogID:            "log-1",
		CompletionRate:   "ok",
		IntensityMatch:   "ok",
		RecoveryAdvice:   "ok",
		AnomalyNotes:     "ok",
		PerformanceNotes: "ok",
		NextSuggestion:   "ok",
	}
	if err := store.UpsertTrainingSummary(context.Background(), summary); err != nil {
		t.Fatalf("upsert summary failed: %v", err)
	}
	if _, err := store.GetTrainingSummary(context.Background(), "log-1"); err != nil {
		t.Fatalf("get summary failed: %v", err)
	}
}
