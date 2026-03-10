package strava

import (
	"os"
	"testing"

	syncjob "stridewise/backend/internal/sync"
)

func TestConnector_FetchActivities(t *testing.T) {
	tmp, err := os.CreateTemp("", "activities-*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	payload := `[
      {"run_id": 1, "name": "a", "distance": 1000, "moving_time": "0:10:00", "start_date_local": "2026-01-01 08:00:00"}
    ]`
	if _, err := tmp.WriteString(payload); err != nil {
		t.Fatalf("write: %v", err)
	}

	c := New(tmp.Name())
	res, err := c.FetchActivities(nil, "u1", syncjob.Checkpoint{})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(res.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(res.Activities))
	}
}

func TestConnector_EmptyDataFile(t *testing.T) {
	c := New("")
	_, err := c.FetchActivities(nil, "u1", syncjob.Checkpoint{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "strava data_file is empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}
