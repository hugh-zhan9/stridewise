package weather

import (
	"context"
	"testing"
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
