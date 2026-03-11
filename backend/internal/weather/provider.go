package weather

import "context"

type Location struct {
	Lat      float64
	Lng      float64
	Country  string
	Province string
	City     string
}

type Provider interface {
	GetSnapshot(ctx context.Context, location Location) (SnapshotInput, error)
	GetForecast(ctx context.Context, location Location) ([]ForecastInput, error)
}

type MockProvider struct {
	fixed     SnapshotInput
	forecasts []ForecastInput
}

func NewMockProvider(fixed SnapshotInput, forecasts ...[]ForecastInput) MockProvider {
	if len(forecasts) > 0 {
		return MockProvider{fixed: fixed, forecasts: forecasts[0]}
	}
	return MockProvider{fixed: fixed}
}

func (m MockProvider) GetSnapshot(_ context.Context, _ Location) (SnapshotInput, error) {
	return m.fixed, nil
}

func (m MockProvider) GetForecast(_ context.Context, _ Location) ([]ForecastInput, error) {
	return m.forecasts, nil
}
