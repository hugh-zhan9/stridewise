package weather

type RiskLevel string

const (
	RiskGreen  RiskLevel = "green"
	RiskYellow RiskLevel = "yellow"
	RiskRed    RiskLevel = "red"
)

type SnapshotInput struct {
	TemperatureC      float64
	FeelsLikeC        float64
	Humidity          float64
	WindSpeedMS       float64
	PrecipitationProb float64
	AQI               int
	UVIndex           float64
}

func ClassifyRisk(input SnapshotInput) RiskLevel {
	if input.FeelsLikeC >= 40.6 {
		return RiskRed
	}
	if input.WindSpeedMS >= 17.9 {
		return RiskRed
	}
	if input.AQI >= 151 {
		return RiskRed
	}
	if input.UVIndex >= 8 {
		return RiskRed
	}
	if input.FeelsLikeC >= 32.2 {
		return RiskYellow
	}
	if input.WindSpeedMS >= 13.9 {
		return RiskYellow
	}
	if input.PrecipitationProb >= 0.4 {
		return RiskYellow
	}
	if input.AQI >= 101 {
		return RiskYellow
	}
	if input.UVIndex >= 3 {
		return RiskYellow
	}
	return RiskGreen
}
