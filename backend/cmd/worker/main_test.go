package main

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"stridewise/backend/internal/storage"
)

func TestNextNightlyDelay(t *testing.T) {
	now := time.Date(2026, 3, 11, 1, 0, 0, 0, time.Local)
	d := nextNightlyDelay(now)
	if d <= 0 {
		t.Fatalf("expected positive delay")
	}
}

func TestNextNightlyDelayAfterTarget(t *testing.T) {
	now := time.Date(2026, 3, 11, 3, 0, 0, 0, time.Local)
	d := nextNightlyDelay(now)
	if d <= 0 {
		t.Fatalf("expected positive delay")
	}
}

type nightlyRunStoreStub struct {
	run     storage.NightlyBaselineRun
	runErr  error
	users   []string
	upserts []storage.NightlyBaselineRun
}

func (s *nightlyRunStoreStub) GetNightlyBaselineRun(_ context.Context, _ time.Time) (storage.NightlyBaselineRun, error) {
	if s.runErr != nil {
		return storage.NightlyBaselineRun{}, s.runErr
	}
	return s.run, nil
}

func (s *nightlyRunStoreStub) UpsertNightlyBaselineRun(_ context.Context, run storage.NightlyBaselineRun) error {
	s.upserts = append(s.upserts, run)
	return nil
}

func (s *nightlyRunStoreStub) ListActiveUsersSince(_ context.Context, _ time.Time) ([]string, error) {
	return s.users, nil
}

type nightlyEnqueuerStub struct {
	calls int
}

func (e *nightlyEnqueuerStub) EnqueueBaselineRecalc(_ context.Context, _ string, _ string, _ string) error {
	e.calls++
	return nil
}

func TestRunNightlyIfNeeded_Catchup(t *testing.T) {
	now := time.Date(2026, 3, 11, 3, 0, 0, 0, time.Local)
	store := &nightlyRunStoreStub{runErr: pgx.ErrNoRows, users: []string{"u1"}}
	enq := &nightlyEnqueuerStub{}
	runNightlyIfNeeded(context.Background(), store, enq, func() time.Time { return now })
	if enq.calls != 1 {
		t.Fatalf("expected enqueue on catchup")
	}
	if len(store.upserts) < 2 {
		t.Fatalf("expected run status upserts")
	}
	if store.upserts[len(store.upserts)-1].Status != "success" {
		t.Fatalf("expected success status")
	}
}

func TestRunNightlyIfNeeded_SkipBeforeTarget(t *testing.T) {
	now := time.Date(2026, 3, 11, 1, 0, 0, 0, time.Local)
	store := &nightlyRunStoreStub{runErr: pgx.ErrNoRows, users: []string{"u1"}}
	enq := &nightlyEnqueuerStub{}
	runNightlyIfNeeded(context.Background(), store, enq, func() time.Time { return now })
	if enq.calls != 0 {
		t.Fatalf("expected no enqueue before target")
	}
	if len(store.upserts) != 0 {
		t.Fatalf("expected no upserts before target")
	}
}

func TestRunNightlyIfNeeded_SkipAlreadySuccess(t *testing.T) {
	now := time.Date(2026, 3, 11, 3, 0, 0, 0, time.Local)
	store := &nightlyRunStoreStub{run: storage.NightlyBaselineRun{Status: "success"}, users: []string{"u1"}}
	enq := &nightlyEnqueuerStub{}
	runNightlyIfNeeded(context.Background(), store, enq, func() time.Time { return now })
	if enq.calls != 0 {
		t.Fatalf("expected no enqueue when already success")
	}
	if len(store.upserts) != 0 {
		t.Fatalf("expected no upserts when already success")
	}
}
