package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"stridewise/backend/internal/baseline"
	"stridewise/backend/internal/storage"
)

const nightlyTargetHour = 2

type nightlyRunStore interface {
	GetNightlyBaselineRun(ctx context.Context, runDate time.Time) (storage.NightlyBaselineRun, error)
	UpsertNightlyBaselineRun(ctx context.Context, run storage.NightlyBaselineRun) error
	ListActiveUsersSince(ctx context.Context, since time.Time) ([]string, error)
}

func runNightlyScheduler(ctx context.Context, store nightlyRunStore, enqueuer baseline.NightlyEnqueuer, nowFn func() time.Time) {
	if store == nil || enqueuer == nil || nowFn == nil {
		return
	}
	runNightlyIfNeeded(ctx, store, enqueuer, nowFn)
	for {
		delay := nextNightlyDelay(nowFn())
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			runNightlyIfNeeded(ctx, store, enqueuer, nowFn)
		}
	}
}

func runNightlyIfNeeded(ctx context.Context, store nightlyRunStore, enqueuer baseline.NightlyEnqueuer, nowFn func() time.Time) {
	if store == nil || enqueuer == nil || nowFn == nil {
		return
	}
	now := nowFn()
	if !isAfterNightlyTarget(now) {
		return
	}
	runDate := nightlyRunDate(now)
	run, err := store.GetNightlyBaselineRun(ctx, runDate)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err == nil && run.Status == "success" {
		return
	}
	startedAt := now
	_ = store.UpsertNightlyBaselineRun(ctx, storage.NightlyBaselineRun{
		RunDate:      runDate,
		Status:       "running",
		ErrorMessage: "",
		StartedAt:    &startedAt,
	})
	baseline.RunNightlyBaselineRecalc(ctx, store, enqueuer, func() time.Time { return now })
	completedAt := nowFn()
	_ = store.UpsertNightlyBaselineRun(ctx, storage.NightlyBaselineRun{
		RunDate:      runDate,
		Status:       "success",
		ErrorMessage: "",
		StartedAt:    &startedAt,
		CompletedAt:  &completedAt,
	})
}

func isAfterNightlyTarget(now time.Time) bool {
	local := now.In(time.Local)
	target := time.Date(local.Year(), local.Month(), local.Day(), nightlyTargetHour, 0, 0, 0, local.Location())
	return !local.Before(target)
}

func nextNightlyDelay(now time.Time) time.Duration {
	local := now.In(time.Local)
	target := time.Date(local.Year(), local.Month(), local.Day(), nightlyTargetHour, 0, 0, 0, local.Location())
	if !local.Before(target) {
		target = target.Add(24 * time.Hour)
	}
	return target.Sub(local)
}

func nightlyRunDate(now time.Time) time.Time {
	local := now.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
}
