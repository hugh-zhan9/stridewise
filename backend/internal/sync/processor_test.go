package sync

import (
	"context"
	"testing"
	"time"
)

type fakeConnector struct{}

func (fakeConnector) FetchActivities(_ context.Context, _ string, _ Checkpoint) (FetchResult, error) {
	return FetchResult{
		Activities: []RawActivity{{
			SourceActivityID: "k-1",
			Name:             "晨跑",
			DistanceM:        5000,
			MovingTimeSec:    1500,
			StartTime:        time.Date(2026, 3, 9, 6, 0, 0, 0, time.UTC),
			SummaryPolyline:  "abc",
		}},
		LastSyncedAt: time.Date(2026, 3, 9, 6, 0, 0, 0, time.UTC),
	}, nil
}

type fakeStore struct {
	runningCalled bool
	successCalled bool
	failedCalled  bool
	saved         []CanonicalActivity
}

func (f *fakeStore) MarkRunning(_ context.Context, _ string) error {
	f.runningCalled = true
	return nil
}

func (f *fakeStore) SaveRawAndCanonical(_ context.Context, _ string, _ string, _ string, activities []CanonicalActivity) error {
	f.saved = activities
	return nil
}

func (f *fakeStore) MarkSuccess(_ context.Context, _ string, _ int) error {
	f.successCalled = true
	return nil
}

func (f *fakeStore) MarkFailed(_ context.Context, _ string, _ int, _ string) error {
	f.failedCalled = true
	return nil
}

func (f *fakeStore) GetCheckpoint(_ context.Context, _, _ string) (Checkpoint, error) {
	return Checkpoint{}, nil
}

func (f *fakeStore) UpsertCheckpoint(_ context.Context, _, _ string, _ Checkpoint) error {
	return nil
}

func (f *fakeStore) AppendSyncError(_ context.Context, _, _, _ string, _ bool) error {
	return nil
}

func TestProcessor_ProcessSyncJob_KeepSource(t *testing.T) {
	store := &fakeStore{}
	p := NewProcessor(store, map[string]Connector{
		"keep": fakeConnector{},
	})

	err := p.ProcessSyncJob(context.Background(), "job-1", "u1", "keep", 0)
	if err != nil {
		t.Fatalf("process failed: %v", err)
	}

	if !store.runningCalled {
		t.Fatal("expected mark running called")
	}
	if !store.successCalled {
		t.Fatal("expected mark success called")
	}
	if store.failedCalled {
		t.Fatal("did not expect failed called")
	}
	if len(store.saved) != 1 {
		t.Fatalf("expected 1 saved activity, got %d", len(store.saved))
	}
	if store.saved[0].Source != "keep" || store.saved[0].UserID != "u1" {
		t.Fatalf("unexpected canonical activity: %+v", store.saved[0])
	}
}
