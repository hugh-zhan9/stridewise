package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTrainingLogCRUD(t *testing.T) {
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
	start := time.Date(2026, 3, 10, 7, 0, 0, 0, time.UTC)

	log := TrainingLog{
		LogID:        "log-1",
		UserID:       "u1",
		Source:       "manual",
		TrainingType: "轻松跑",
		StartTime:    start,
		DurationSec:  3600,
		DistanceKM:   10.5,
		PaceStr:      "05'30''",
		PaceSecPerKM: 330,
		RPE:          6,
		Discomfort:   false,
	}
	if err := store.CreateTrainingLog(context.Background(), log); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	log.DistanceKM = 11.0
	if err := store.UpdateTrainingLog(context.Background(), log); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := store.SoftDeleteTrainingLog(context.Background(), log.LogID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestTrainingLogConflict(t *testing.T) {
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
	start := time.Date(2026, 3, 10, 7, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	conflict, err := store.HasTrainingConflict(context.Background(), "u1", start, end, "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	_ = conflict
}

func TestListTrainingLogs(t *testing.T) {
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
	start := time.Date(2026, 3, 10, 7, 0, 0, 0, time.UTC)
	log := TrainingLog{
		LogID:        "log-list-1",
		UserID:       "u1",
		Source:       "manual",
		TrainingType: "轻松跑",
		StartTime:    start,
		DurationSec:  1800,
		DistanceKM:   5.0,
		PaceStr:      "06'00''",
		PaceSecPerKM: 360,
		RPE:          5,
		Discomfort:   false,
	}
	if err := store.CreateTrainingLog(context.Background(), log); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	items, err := store.ListTrainingLogs(context.Background(), "u1", start.Add(-time.Hour), start.Add(time.Hour))
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected logs")
	}
}

func TestGetTrainingLog(t *testing.T) {
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
	start := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	log := TrainingLog{
		LogID:        "log-get-1",
		UserID:       "u1",
		Source:       "manual",
		TrainingType: "轻松跑",
		StartTime:    start,
		DurationSec:  1800,
		DistanceKM:   5.0,
		PaceStr:      "06'00''",
		PaceSecPerKM: 360,
		RPE:          5,
		Discomfort:   false,
	}
	if err := store.CreateTrainingLog(context.Background(), log); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	got, err := store.GetTrainingLog(context.Background(), log.LogID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.LogID != log.LogID {
		t.Fatalf("unexpected log id")
	}
}
