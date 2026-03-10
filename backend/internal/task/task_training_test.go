package task

import "testing"

func TestEncodeDecodeTrainingRecalcPayload(t *testing.T) {
	p := TrainingRecalcPayload{JobID: "job-1", UserID: "u1", LogID: "log-1", Operation: "create"}
	b, err := EncodeTrainingRecalcPayload(p)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out, err := DecodeTrainingRecalcPayload(b)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.JobID != "job-1" {
		t.Fatalf("unexpected payload")
	}
}
