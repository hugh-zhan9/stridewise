package task

import "testing"

func TestSyncJobPayload_WithRetryCountRoundTrip(t *testing.T) {
	p := SyncJobPayload{JobID: "j1", UserID: "u1", Source: "keep", RetryCount: 2}
	b, err := EncodeSyncJobPayload(p)
	if err != nil {
		t.Fatalf("encode err: %v", err)
	}
	got, err := DecodeSyncJobPayload(b)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if got.RetryCount != 2 {
		t.Fatalf("expected retry_count 2, got %d", got.RetryCount)
	}
}
