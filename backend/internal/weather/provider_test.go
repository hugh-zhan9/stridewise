package weather

import (
	"context"
	"testing"
	"time"
)

func TestMockProvider_ReturnsFixedSnapshot(t *testing.T) {
	mock := NewMockProvider(SnapshotInput{TemperatureC: 18})
	got, err := mock.GetSnapshot(context.Background(), Location{Lat: 1, Lng: 2})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.TemperatureC != 18 {
		t.Fatalf("expected 18, got %v", got.TemperatureC)
	}
}

func TestMockProviderForecast(t *testing.T) {
	mock := NewMockProvider(SnapshotInput{TemperatureC: 20}, []ForecastInput{{
		Date: time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
	}})
	out, err := mock.GetForecast(context.Background(), Location{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 forecast")
	}
}
