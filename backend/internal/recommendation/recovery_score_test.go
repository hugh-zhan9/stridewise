package recommendation

import "testing"

func TestBuildRecoveryScore_RedByHighLoad(t *testing.T) {
	score := BuildRecoveryScore(1.6, 2.2, 520, false, 0)
	if score.RecoveryStatus != "red" {
		t.Fatalf("expected red status, got %s", score.RecoveryStatus)
	}
	if score.OverallScore >= 60 {
		t.Fatalf("expected lower overall score")
	}
}

func TestBuildRecoveryScore_YellowByMildLoad(t *testing.T) {
	score := BuildRecoveryScore(1.35, 1.9, 350, false, 0)
	if score.RecoveryStatus != "yellow" {
		t.Fatalf("expected yellow status, got %s", score.RecoveryStatus)
	}
}

