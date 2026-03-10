package common

import (
	"os"
	"testing"
	"time"

	syncjob "stridewise/backend/internal/sync"
)

func TestParseRunningPageJSON_FiltersByCheckpoint(t *testing.T) {
	tmp, err := os.CreateTemp("", "activities-*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	payload := `[
      {"run_id": 1, "name": "a", "distance": 1000, "moving_time": "0:10:00", "start_date": "2026-01-01 00:00:00+00:00", "start_date_local": "2026-01-01 08:00:00"},
      {"run_id": 2, "name": "b", "distance": 2000, "moving_time": "0:20:00", "start_date": "2026-01-02 00:00:00+00:00", "start_date_local": "2026-01-02 08:00:00"}
    ]`
	if _, err := tmp.WriteString(payload); err != nil {
		t.Fatalf("write: %v", err)
	}

	cp := syncjob.Checkpoint{LastSyncedAt: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)}
	res, err := ParseRunningPageJSON(tmp.Name(), cp)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(res.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(res.Activities))
	}
	if res.Activities[0].SourceActivityID != "2" {
		t.Fatalf("unexpected id: %s", res.Activities[0].SourceActivityID)
	}
	if res.LastSyncedAt.IsZero() {
		t.Fatal("expected last synced at")
	}
}

func TestParseRunningPageJSON_FallbackStartDate(t *testing.T) {
	tmp, err := os.CreateTemp("", "activities-*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	payload := `[
      {"run_id": "a1", "name": "x", "distance": 1000, "moving_time": "0:10:00", "start_date": "2026-01-03 00:00:00+00:00", "start_date_local": "invalid"}
    ]`
	if _, err := tmp.WriteString(payload); err != nil {
		t.Fatalf("write: %v", err)
	}

	res, err := ParseRunningPageJSON(tmp.Name(), syncjob.Checkpoint{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(res.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(res.Activities))
	}
}
