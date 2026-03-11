package recommendation

import "testing"

func TestRecoveryStatus_ByACWR(t *testing.T) {
	if got := CalcRecoveryStatus(1.6, 1.0); got != "red" {
		t.Fatalf("expected red")
	}
	if got := CalcRecoveryStatus(1.4, 1.0); got != "yellow" {
		t.Fatalf("expected yellow")
	}
	if got := CalcRecoveryStatus(1.0, 1.0); got != "green" {
		t.Fatalf("expected green")
	}
}

func TestRecoveryStatus_ByMonotony(t *testing.T) {
	if got := CalcRecoveryStatus(1.0, 2.3); got != "red" {
		t.Fatalf("expected red")
	}
	if got := CalcRecoveryStatus(1.0, 2.1); got != "yellow" {
		t.Fatalf("expected yellow")
	}
}
