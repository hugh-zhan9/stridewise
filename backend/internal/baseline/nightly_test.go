package baseline

import (
	"context"
	"testing"
	"time"
)

type nightlyStoreStub struct {
	users []string
	err   error
}

func (s nightlyStoreStub) ListActiveUsersSince(_ context.Context, _ time.Time) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.users, nil
}

type nightlyEnqueuerStub struct {
	calls       int
	lastUser    string
	lastTrigger string
	lastRef     string
}

func (e *nightlyEnqueuerStub) EnqueueBaselineRecalc(_ context.Context, userID, triggerType, triggerRef string) error {
	e.calls++
	e.lastUser = userID
	e.lastTrigger = triggerType
	e.lastRef = triggerRef
	return nil
}

func TestRunNightlyBaselineRecalc(t *testing.T) {
	store := nightlyStoreStub{users: []string{"u1"}}
	enq := &nightlyEnqueuerStub{}
	now := time.Date(2026, 3, 11, 2, 0, 0, 0, time.Local)
	RunNightlyBaselineRecalc(context.Background(), store, enq, func() time.Time { return now })
	if enq.calls != 1 {
		t.Fatalf("expected 1 enqueue, got %d", enq.calls)
	}
	if enq.lastUser != "u1" || enq.lastTrigger != "nightly" {
		t.Fatalf("unexpected enqueue: user=%s trigger=%s", enq.lastUser, enq.lastTrigger)
	}
	if enq.lastRef != "nightly-20260311" {
		t.Fatalf("unexpected trigger ref: %s", enq.lastRef)
	}
}

func TestRunNightlyBaselineRecalc_EmptyUsers(t *testing.T) {
	store := nightlyStoreStub{users: []string{}}
	enq := &nightlyEnqueuerStub{}
	RunNightlyBaselineRecalc(context.Background(), store, enq, time.Now)
	if enq.calls != 0 {
		t.Fatalf("expected 0 enqueue, got %d", enq.calls)
	}
}

func TestRunNightlyBaselineRecalc_ErrorIgnored(t *testing.T) {
	store := nightlyStoreStub{err: context.Canceled}
	enq := &nightlyEnqueuerStub{}
	RunNightlyBaselineRecalc(context.Background(), store, enq, time.Now)
	if enq.calls != 0 {
		t.Fatalf("expected 0 enqueue, got %d", enq.calls)
	}
}
