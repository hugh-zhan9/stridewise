package training

import (
	"context"
	"testing"
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

func TestProcessor_ProcessTrainingRecalc(t *testing.T) {
	store := &fakeAsyncStore{}
	p := NewProcessor(store)

	if err := p.ProcessTrainingRecalc(context.Background(), "job-1", "u1", "log-1", "create", 0); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if store.lastStatus != "success" {
		t.Fatalf("expected success, got %s", store.lastStatus)
	}
}
