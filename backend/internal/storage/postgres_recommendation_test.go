package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecommendationStore(t *testing.T) {
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
	rec := Recommendation{
		RecID:              "r1",
		UserID:             "u1",
		RecommendationDate: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		InputJSON:          []byte(`{"a":1}`),
		OutputJSON:         []byte(`{"b":2}`),
		RiskLevel:          "green",
		OverrideJSON:       []byte(`{}`),
		IsFallback:         false,
		AIProvider:         "openai",
		AIModel:            "gpt-4.1-mini",
		PromptVersion:      "v1",
		EngineVersion:      "v1",
	}
	if err := store.CreateRecommendation(context.Background(), rec); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := store.GetLatestRecommendation(context.Background(), "u1"); err != nil {
		t.Fatalf("get latest failed: %v", err)
	}
	feedback := RecommendationFeedback{
		FeedbackID: "f1",
		RecID:      "r1",
		UserID:     "u1",
		Useful:     "yes",
		Reason:     "ok",
	}
	if err := store.CreateRecommendationFeedback(context.Background(), feedback); err != nil {
		t.Fatalf("feedback failed: %v", err)
	}
}
