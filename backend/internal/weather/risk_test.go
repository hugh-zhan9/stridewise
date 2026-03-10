package weather

import "testing"

func TestClassifyRisk_RedByAQI(t *testing.T) {
	input := SnapshotInput{
		TemperatureC:      22,
		FeelsLikeC:        22,
		Humidity:          0.5,
		WindSpeedMS:       3,
		PrecipitationProb: 0.1,
		AQI:               180,
		UVIndex:           3,
	}
	if got := ClassifyRisk(input); got != RiskRed {
		t.Fatalf("expected %s, got %s", RiskRed, got)
	}
}
