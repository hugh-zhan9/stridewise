package weather

import (
	"testing"
	"time"
)

func TestParseQWeatherNow(t *testing.T) {
	payload := `{"code":"200","now":{"temp":"20","feelsLike":"18","humidity":"60","windSpeed":"12","precip":"1.2","uvIndex":"3"}}`
	got, err := parseQWeatherNow([]byte(payload))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if got.TemperatureC != 20 {
		t.Fatalf("expected temp 20")
	}
	if got.FeelsLikeC != 18 {
		t.Fatalf("expected feels 18")
	}
	if got.Humidity != 60 {
		t.Fatalf("expected humidity 60")
	}
	if got.WindSpeedMS < 3.3 || got.WindSpeedMS > 3.4 {
		t.Fatalf("expected wind speed ~3.33 m/s")
	}
	if got.PrecipitationProb != 0.4 {
		t.Fatalf("expected precip prob 0.4")
	}
	if got.UVIndex != 3 {
		t.Fatalf("expected uv 3")
	}
}

func TestParseQWeatherAir(t *testing.T) {
	payload := `{"code":"200","now":{"aqi":"55"}}`
	aqi, err := parseQWeatherAir([]byte(payload))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if aqi != 55 {
		t.Fatalf("expected aqi 55")
	}
}

func TestParseQWeatherForecasts(t *testing.T) {
	payload := `{"code":"200","daily":[{"fxDate":"2026-03-11","tempMax":"25","tempMin":"12","humidity":"55","precip":"0.0","pressure":"1012","vis":"10","cloud":"20","uvIndex":"5","textDay":"多云","textNight":"晴","iconDay":"101","iconNight":"150","wind360Day":"90","windDirDay":"东风","windScaleDay":"3","windSpeedDay":"12","wind360Night":"270","windDirNight":"西风","windScaleNight":"2","windSpeedNight":"8","sunrise":"06:30","sunset":"18:20","moonrise":"20:10","moonset":"07:15","moonPhase":"盈凸月","moonPhaseIcon":"804"}]}`
	got, err := parseQWeatherForecasts([]byte(payload))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 forecast")
	}
	if got[0].TempMaxC == nil || *got[0].TempMaxC != 25 {
		t.Fatalf("expected temp_max 25")
	}
	if got[0].WindSpeedDayMS == nil || *got[0].WindSpeedDayMS < 3.3 || *got[0].WindSpeedDayMS > 3.4 {
		t.Fatalf("expected day wind speed ~3.33 m/s")
	}
	if got[0].SunriseTime == nil || got[0].SunriseTime.Hour() != 6 {
		t.Fatalf("expected sunrise hour 6")
	}
	if got[0].Date.Format("2006-01-02") != "2026-03-11" {
		t.Fatalf("expected forecast date 2026-03-11")
	}
	if got[0].SunsetTime == nil || got[0].SunsetTime.Location() != time.UTC {
		t.Fatalf("expected time in UTC")
	}
}
