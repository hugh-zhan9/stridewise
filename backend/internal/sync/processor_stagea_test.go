package sync

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stageAConnector struct {
	result FetchResult
	err    error
}

func (c stageAConnector) FetchActivities(_ context.Context, _ string, _ Checkpoint) (FetchResult, error) {
	if c.err != nil {
		return FetchResult{}, c.err
	}
	return c.result, nil
}

type stageAStore struct {
	checkpointSaved bool
	errorSaved      bool
	successCalled   bool
	failedCalled    bool
}

func (s *stageAStore) MarkRunning(context.Context, string) error { return nil }
func (s *stageAStore) SaveRawAndCanonical(context.Context, string, string, string, []CanonicalActivity) error {
	return nil
}
func (s *stageAStore) MarkSuccess(context.Context, string, int) error {
	s.successCalled = true
	return nil
}
func (s *stageAStore) MarkFailed(context.Context, string, int, string) error {
	s.failedCalled = true
	return nil
}
func (s *stageAStore) GetCheckpoint(context.Context, string, string) (Checkpoint, error) {
	return Checkpoint{}, nil
}
func (s *stageAStore) UpsertCheckpoint(context.Context, string, string, Checkpoint) error {
	s.checkpointSaved = true
	return nil
}
func (s *stageAStore) AppendSyncError(context.Context, string, string, string, bool) error {
	s.errorSaved = true
	return nil
}

func TestProcessor_SavesCheckpointOnSuccess(t *testing.T) {
	store := &stageAStore{}
	p := NewProcessor(store, map[string]Connector{
		"keep": stageAConnector{result: FetchResult{Activities: []RawActivity{{
			SourceActivityID: "a1",
			Name:             "run",
			DistanceM:        1000,
			MovingTimeSec:    300,
			StartTime:        time.Now(),
		}}, LastSyncedAt: time.Now()}},
	})

	err := p.ProcessSyncJob(context.Background(), "j1", "u1", "keep", 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !store.checkpointSaved {
		t.Fatal("expected checkpoint saved")
	}
	if !store.successCalled {
		t.Fatal("expected success called")
	}
}

func TestProcessor_AppendsSyncErrorOnConnectorFailure(t *testing.T) {
	store := &stageAStore{}
	p := NewProcessor(store, map[string]Connector{
		"keep": stageAConnector{err: errors.New("upstream timeout")},
	})

	err := p.ProcessSyncJob(context.Background(), "j2", "u1", "keep", 1)
	if err == nil {
		t.Fatal("expected err")
	}
	if !store.errorSaved {
		t.Fatal("expected sync error saved")
	}
	if !store.failedCalled {
		t.Fatal("expected failed called")
	}
}
