package task

import "testing"

func TestSyncJobPayload_RoundTrip(t *testing.T) {
	p := SyncJobPayload{UserID: "u1", Source: "keep"}
	b, err := EncodeSyncJobPayload(p)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	got, err := DecodeSyncJobPayload(b)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if got.UserID != p.UserID || got.Source != p.Source {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestSyncJobPayload_RejectsUnsupportedSource(t *testing.T) {
	_, err := EncodeSyncJobPayload(SyncJobPayload{UserID: "u1", Source: "apple_health"})
	if err == nil {
		t.Fatal("expected error for unsupported source")
	}
}
