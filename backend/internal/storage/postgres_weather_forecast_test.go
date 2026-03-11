package storage

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWeatherForecastsMigration(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	migrationPath := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations", "008_weather_forecasts.sql"))
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration failed: %v", err)
	}

	dsn := os.Getenv("STRIDEWISE_TEST_DSN")
	if dsn == "" {
		t.Skip("STRIDEWISE_TEST_DSN not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer pool.Close()

	statements := strings.Split(string(content), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("exec migration failed: %v", err)
		}
	}

	var regclass *string
	if err := pool.QueryRow(context.Background(), "SELECT to_regclass('public.weather_forecasts')").Scan(&regclass); err != nil {
		t.Fatalf("check table failed: %v", err)
	}
	if regclass == nil || *regclass == "" {
		t.Fatalf("weather_forecasts table not found")
	}
}

func TestWeatherForecastStore(t *testing.T) {
	dsn := os.Getenv("STRIDEWISE_TEST_DSN")
	if dsn == "" {
		t.Skip("STRIDEWISE_TEST_DSN not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer pool.Close()

	store := NewPostgresStore(pool)
	ctx := context.Background()

	_, _ = pool.Exec(ctx, "DELETE FROM weather_forecasts WHERE user_id=$1", "u1")

	tempMax := 25.0
	tempMin := 12.0
	humidity := 55.0
	precip := 0.0
	pressure := 1012.0
	visibility := 10.0
	cloud := 20.0
	uv := 5.0
	textDay := "多云"
	textNight := "晴"
	iconDay := "101"
	iconNight := "150"
	wind360Day := 90
	windDirDay := "东风"
	windScaleDay := "3"
	windSpeedDay := 12.0
	wind360Night := 270
	windDirNight := "西风"
	windScaleNight := "2"
	windSpeedNight := 8.0
	sunrise := time.Date(2026, 3, 11, 6, 30, 0, 0, time.UTC)
	sunset := time.Date(2026, 3, 11, 18, 20, 0, 0, time.UTC)

	forecasts := []WeatherForecast{
		{
			ForecastID:       "f1",
			UserID:           "u1",
			ForecastDate:     time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
			TempMaxC:         &tempMax,
			TempMinC:         &tempMin,
			Humidity:         &humidity,
			PrecipMM:         &precip,
			PressureHPA:      &pressure,
			VisibilityKM:     &visibility,
			CloudPct:         &cloud,
			UVIndex:          &uv,
			TextDay:          &textDay,
			TextNight:        &textNight,
			IconDay:          &iconDay,
			IconNight:        &iconNight,
			Wind360Day:       &wind360Day,
			WindDirDay:       &windDirDay,
			WindScaleDay:     &windScaleDay,
			WindSpeedDayMS:   &windSpeedDay,
			Wind360Night:     &wind360Night,
			WindDirNight:     &windDirNight,
			WindScaleNight:   &windScaleNight,
			WindSpeedNightMS: &windSpeedNight,
			SunriseTime:      &sunrise,
			SunsetTime:       &sunset,
		},
		{
			ForecastID:   "f2",
			UserID:       "u1",
			ForecastDate: time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC),
		},
	}

	if err := store.UpsertWeatherForecasts(ctx, forecasts); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	got, err := store.GetWeatherForecasts(ctx, "u1", time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].TempMaxC == nil || *got[0].TempMaxC != 25.0 {
		t.Fatalf("expected temp_max 25")
	}
	if got[1].TempMaxC != nil {
		t.Fatalf("expected nil temp_max for second forecast")
	}
}

func TestWeatherForecastAQIColumns(t *testing.T) {
	dsn := os.Getenv("STRIDEWISE_TEST_DSN")
	if dsn == "" {
		t.Skip("STRIDEWISE_TEST_DSN not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer pool.Close()

	var col string
	err = pool.QueryRow(context.Background(),
		"SELECT column_name FROM information_schema.columns WHERE table_name='weather_forecasts' AND column_name='aqi_local'").Scan(&col)
	if err != nil || col != "aqi_local" {
		t.Fatalf("aqi_local column missing")
	}
}
