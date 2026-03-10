package fit

import (
	"testing"

	syncjob "stridewise/backend/internal/sync"
)

func TestConnector_EmptyDataFile(t *testing.T) {
	c := New("")
	_, err := c.FetchActivities(nil, "u1", syncjob.Checkpoint{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "fit data_file is empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}
