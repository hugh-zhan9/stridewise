package config

import (
	"os"
	"testing"
)

func TestLoad_IncludesOfflineSources(t *testing.T) {
	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	payload := `
server:
  http:
    addr: ":8000"
security:
  internal_token: "token"
postgres:
  dsn: "postgres://stridewise:stridewise@localhost:5432/stridewise?sslmode=disable"
redis:
  addr: "localhost:6379"
asynq:
  concurrency: 5
keep:
  data_file: "keep.json"
strava:
  data_file: "strava.json"
garmin:
  data_file: "garmin.json"
nike:
  data_file: "nike.json"
gpx:
  data_file: "gpx.json"
tcx:
  data_file: "tcx.json"
fit:
  data_file: "fit.json"
`
	if _, err := tmp.WriteString(payload); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Keep.DataFile != "keep.json" {
		t.Fatalf("expected keep data_file")
	}
	if cfg.Strava.DataFile != "strava.json" {
		t.Fatalf("expected strava data_file")
	}
	if cfg.Garmin.DataFile != "garmin.json" {
		t.Fatalf("expected garmin data_file")
	}
	if cfg.Nike.DataFile != "nike.json" {
		t.Fatalf("expected nike data_file")
	}
	if cfg.GPX.DataFile != "gpx.json" {
		t.Fatalf("expected gpx data_file")
	}
	if cfg.TCX.DataFile != "tcx.json" {
		t.Fatalf("expected tcx data_file")
	}
	if cfg.FIT.DataFile != "fit.json" {
		t.Fatalf("expected fit data_file")
	}
}
