package baseline

import "testing"

func TestCalcPaceAverage(t *testing.T) {
	input := []SessionInput{
		{DistanceKM: 5, PaceSecPerKM: 360},
		{DistanceKM: 10, PaceSecPerKM: 330},
	}
	avg := CalcPaceAverage(input)
	if avg != 340 {
		t.Fatalf("expected 340, got %d", avg)
	}
}

func TestCalcACWRDistance(t *testing.T) {
	items := []SessionInput{
		{DistanceKM: 5, StartDayIndex: 0},
		{DistanceKM: 5, StartDayIndex: 1},
	}
	m := CalcMetrics(items, 2)
	if m.ACWRDistance <= 0 {
		t.Fatalf("expected acwr distance")
	}
}
