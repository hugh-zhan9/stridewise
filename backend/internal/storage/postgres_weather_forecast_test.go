package storage

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
