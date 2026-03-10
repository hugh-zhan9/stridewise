package task

import "testing"

func TestEncodeDecodeBaselineRecalcPayload(t *testing.T) {
	p := BaselineRecalcPayload{
		JobID:       "job-1",
		UserID:      "u1",
		TriggerType: "training_create",
		TriggerRef:  "log-1",
	}
	b, err := EncodeBaselineRecalcPayload(p)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out, err := DecodeBaselineRecalcPayload(b)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.JobID != "job-1" {
		t.Fatalf("unexpected payload")
	}
}
