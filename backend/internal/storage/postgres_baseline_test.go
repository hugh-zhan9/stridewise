package storage

import (
	"context"
	"fmt"
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
		SourceType:       "log",
		SourceID:         "log-1",
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
	if _, err := store.GetTrainingSummaryBySource(context.Background(), "log", "log-1"); err != nil {
		t.Fatalf("get summary failed: %v", err)
	}
}

func TestListTrainingSummariesIncludesActivity(t *testing.T) {
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
	var activityID int64
	err = pool.QueryRow(context.Background(), `
		INSERT INTO activities (user_id, source, source_activity_id, name, distance_m, moving_time_sec, start_time_utc, start_time_local, timezone)
		VALUES ('u1','keep','a1','act',5000,1500,NOW(),NOW(),'UTC')
		RETURNING id
	`).Scan(&activityID)
	if err != nil {
		t.Fatalf("insert activity failed: %v", err)
	}

	summary := TrainingSummary{
		SummaryID:        "s-act",
		UserID:           "u1",
		SourceType:       "activity",
		SourceID:         fmt.Sprintf("%d", activityID),
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

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	summaries, err := store.ListTrainingSummaries(context.Background(), "u1", from, to)
	if err != nil {
		t.Fatalf("list summaries failed: %v", err)
	}
	if len(summaries) == 0 {
		t.Fatalf("expected activity summary in list")
	}
}

func TestGetLatestTrainingFeedback(t *testing.T) {
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
	now := time.Now().UTC()
	sourceID := fmt.Sprintf("log-%d", now.UnixNano())
	feedback := TrainingFeedback{
		FeedbackID: fmt.Sprintf("f-%d", now.UnixNano()),
		UserID:     "u1",
		SourceType: "log",
		SourceID:   sourceID,
		LogID:      sourceID,
		Content:    "太累了",
		CreatedAt:  now,
	}
	if err := store.CreateTrainingFeedback(context.Background(), feedback); err != nil {
		t.Fatalf("create feedback failed: %v", err)
	}
	got, err := store.GetLatestTrainingFeedback(context.Background(), "u1")
	if err != nil {
		t.Fatalf("get latest feedback failed: %v", err)
	}
	if got.Content == "" {
		t.Fatalf("expected content")
	}
}

func TestListActiveUsersSince(t *testing.T) {
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
	now := time.Now().UTC()
	logID := fmt.Sprintf("log-%d", now.UnixNano())
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO training_logs (log_id, user_id, source, training_type, start_time, duration_sec, distance_km, pace_str, pace_sec_per_km, rpe, discomfort, created_at, updated_at)
		VALUES ($1,'u1','manual','easy',NOW(),1800,5,'05''30''',330,5,false,NOW(),NOW())
	`, logID); err != nil {
		t.Fatalf("insert training log failed: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO activities (user_id, source, source_activity_id, name, distance_m, moving_time_sec, start_time_utc, start_time_local, timezone)
		VALUES ('u2','keep','a1','run',5000,1500,NOW(),NOW(),'UTC')
	`); err != nil {
		t.Fatalf("insert activity failed: %v", err)
	}

	users, err := store.ListActiveUsersSince(context.Background(), now.Add(-28*24*time.Hour))
	if err != nil {
		t.Fatalf("list active users failed: %v", err)
	}
	if len(users) == 0 {
		t.Fatalf("expected users")
	}
}

func TestNightlyBaselineRunStore(t *testing.T) {
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
	now := time.Now().UTC()
	run := NightlyBaselineRun{
		RunDate:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
		Status:    "running",
		StartedAt: &now,
	}
	if err := store.UpsertNightlyBaselineRun(context.Background(), run); err != nil {
		t.Fatalf("upsert run failed: %v", err)
	}
	got, err := store.GetNightlyBaselineRun(context.Background(), run.RunDate)
	if err != nil {
		t.Fatalf("get run failed: %v", err)
	}
	if got.Status != "running" {
		t.Fatalf("unexpected status: %s", got.Status)
	}
}
