package training

import (
	"context"
	"errors"
	"testing"

	"stridewise/backend/internal/storage"
)

type fakeAsyncStore struct {
	lastStatus string
	lastRetry  int
	lastErr    string
}

func (f *fakeAsyncStore) UpdateAsyncJobStatus(_ context.Context, _ string, status string, retryCount int, errMsg string) error {
	f.lastStatus = status
	f.lastRetry = retryCount
	f.lastErr = errMsg
	return nil
}

type fakeBaseline struct {
	called     bool
	summaryErr error
	err        error
}

func (f *fakeBaseline) RecalcForTrigger(_ context.Context, _ string, _ string, _ string) (error, error) {
	f.called = true
	return f.summaryErr, f.err
}

type fakeRecommender struct {
	called bool
	err    error
}

func (f *fakeRecommender) Generate(_ context.Context, _ string) (storage.Recommendation, error) {
	f.called = true
	return storage.Recommendation{}, f.err
}

func TestProcessor_ProcessTrainingRecalc(t *testing.T) {
	store := &fakeAsyncStore{}
	p := NewProcessor(store, &fakeBaseline{}, &fakeRecommender{})

	if err := p.ProcessTrainingRecalc(context.Background(), "job-1", "u1", "log-1", "create", 0); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if store.lastStatus != "success" {
		t.Fatalf("expected success, got %s", store.lastStatus)
	}
}

func TestProcessor_ProcessTrainingRecalc_InvokesDeps(t *testing.T) {
	store := &fakeAsyncStore{}
	baseline := &fakeBaseline{summaryErr: errors.New("summary failed")}
	rec := &fakeRecommender{}
	p := NewProcessor(store, baseline, rec)

	if err := p.ProcessTrainingRecalc(context.Background(), "job-1", "u1", "log-1", "update", 2); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !baseline.called {
		t.Fatalf("expected baseline recalc called")
	}
	if !rec.called {
		t.Fatalf("expected recommendation generate called")
	}
	if store.lastStatus != "success" {
		t.Fatalf("expected success, got %s", store.lastStatus)
	}
	if store.lastErr == "" {
		t.Fatalf("expected err msg recorded")
	}
}
