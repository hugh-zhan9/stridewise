package training

import "testing"

func TestParseDuration(t *testing.T) {
	sec, err := ParseDuration("01:02:03")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sec != 3723 {
		t.Fatalf("expected 3723, got %d", sec)
	}
}

func TestParsePace(t *testing.T) {
	sec, err := ParsePace("05'30''")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sec != 330 {
		t.Fatalf("expected 330, got %d", sec)
	}
}

func TestNormalizeTrainingType_Custom(t *testing.T) {
	tp, custom, err := NormalizeTrainingType("自由跑")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if tp != "custom" || custom != "自由跑" {
		t.Fatalf("unexpected result: %s %s", tp, custom)
	}
}
