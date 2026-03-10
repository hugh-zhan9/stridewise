package keep

import (
	"os"
	"testing"

	syncjob "stridewise/backend/internal/sync"
)

func TestConnector_FetchActivities_FallbackStartDate(t *testing.T) {
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

	c := New(tmp.Name())
	res, err := c.FetchActivities(nil, "u1", syncjob.Checkpoint{})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(res.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(res.Activities))
	}
}
