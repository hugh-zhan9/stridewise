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
}

type MockProvider struct {
	fixed SnapshotInput
}

func NewMockProvider(fixed SnapshotInput) MockProvider {
	return MockProvider{fixed: fixed}
}

func (m MockProvider) GetSnapshot(_ context.Context, _ Location) (SnapshotInput, error) {
	return m.fixed, nil
}
