package task

import "testing"

func TestSyncJobPayload_RoundTrip(t *testing.T) {
	p := SyncJobPayload{JobID: "job-1", UserID: "u1", Source: "keep"}
	b, err := EncodeSyncJobPayload(p)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	got, err := DecodeSyncJobPayload(b)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if got.UserID != p.UserID || got.Source != p.Source || got.JobID != p.JobID {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestSyncJobPayload_RejectsUnsupportedSource(t *testing.T) {
	_, err := EncodeSyncJobPayload(SyncJobPayload{JobID: "job-1", UserID: "u1", Source: "apple_health"})
	if err == nil {
		t.Fatal("expected error for unsupported source")
	}
}

func TestSyncJobPayload_RequiresJobID(t *testing.T) {
	_, err := EncodeSyncJobPayload(SyncJobPayload{UserID: "u1", Source: "keep"})
	if err == nil {
		t.Fatal("expected error for missing job_id")
	}
}

func TestAbilityLevelPayload_RoundTrip(t *testing.T) {
	p := AbilityLevelPayload{JobID: "job-1", UserID: "u1", TriggerType: "profile", TriggerRef: "u1"}
	b, err := EncodeAbilityLevelPayload(p)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	got, err := DecodeAbilityLevelPayload(b)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if got.JobID != p.JobID || got.UserID != p.UserID || got.TriggerType != p.TriggerType || got.TriggerRef != p.TriggerRef {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestAbilityLevelPayload_RequiresJobID(t *testing.T) {
	_, err := EncodeAbilityLevelPayload(AbilityLevelPayload{UserID: "u1", TriggerType: "profile", TriggerRef: "u1"})
	if err == nil {
		t.Fatal("expected error for missing job_id")
	}
}
