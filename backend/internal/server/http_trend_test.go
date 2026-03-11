package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stridewise/backend/internal/trend"
)

type trendServiceStub struct{}

func (trendServiceStub) GetRollingTrend(_ context.Context, _ string, _ string, _ time.Time) (trend.TrendResult, error) {
	return trend.TrendResult{}, nil
}

func TestGetTrends_RejectInvalidWindow(t *testing.T) {
	trend := &trendServiceStub{}
	srv := NewHTTPServer(":0", "token", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, trend)

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/trends?user_id=u1&window=14d", nil)
	req.Header.Set("X-Internal-Token", "token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
}
