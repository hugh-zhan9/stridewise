package weather

import "time"

type ForecastInput struct {
	Date             time.Time
	TempMaxC         *float64
	TempMinC         *float64
	Humidity         *float64
	PrecipMM         *float64
	PressureHPA      *float64
	VisibilityKM     *float64
	CloudPct         *float64
	UVIndex          *float64
	AQILocal         *int
	AQIQAQI          *int
	AQISource        *string
	TextDay          *string
	TextNight        *string
	IconDay          *string
	IconNight        *string
	Wind360Day       *int
	WindDirDay       *string
	WindScaleDay     *string
	WindSpeedDayMS   *float64
	Wind360Night     *int
	WindDirNight     *string
	WindScaleNight   *string
	WindSpeedNightMS *float64
	SunriseTime      *time.Time
	SunsetTime       *time.Time
	MoonriseTime     *time.Time
	MoonsetTime      *time.Time
	MoonPhase        *string
	MoonPhaseIcon    *string
}
