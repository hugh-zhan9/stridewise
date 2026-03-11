package baseline

import (
	"context"
	"time"
)

type NightlyStore interface {
	ListActiveUsersSince(ctx context.Context, since time.Time) ([]string, error)
}

type NightlyEnqueuer interface {
	EnqueueBaselineRecalc(ctx context.Context, userID, triggerType, triggerRef string) error
}

func RunNightlyBaselineRecalc(ctx context.Context, store NightlyStore, enqueuer NightlyEnqueuer, nowFn func() time.Time) {
	if store == nil || enqueuer == nil {
		return
	}
	now := nowFn()
	since := now.Add(-28 * 24 * time.Hour)
	users, err := store.ListActiveUsersSince(ctx, since)
	if err != nil {
		return
	}
	triggerRef := "nightly-" + now.Format("20060102")
	for _, userID := range users {
		_ = enqueuer.EnqueueBaselineRecalc(ctx, userID, "nightly", triggerRef)
	}
}
