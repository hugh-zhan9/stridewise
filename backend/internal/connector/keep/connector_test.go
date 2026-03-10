package keep

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	syncjob "stridewise/backend/internal/sync"
)

func TestConnector_FetchActivities_Live(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.1/users/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"token":"t1"}}`))
	})
	mux.HandleFunc("/pd/v3/stats/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"records":[{"logs":[{"stats":{"id":"run_1","isDoubtful":false}}]}],"lastTimestamp":0}}`))
	})
	mux.HandleFunc("/pd/v3/runninglog/run_1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"abc_1","startTime":1700000000000,"endTime":1700000600000,"duration":600,"distance":1000,"dataType":"outdoorRunning","timezone":"Asia/Shanghai","geoPoints":null,"heartRate":null}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewLive("13000000000", "pass", srv.URL, srv.Client())
	res, err := c.FetchActivities(context.Background(), "u1", syncjob.Checkpoint{})
	if err != nil {
		t.Fatalf("fetch err: %v", err)
	}
	if len(res.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(res.Activities))
	}
	if res.Activities[0].SourceActivityID != "1" {
		t.Fatalf("unexpected id: %s", res.Activities[0].SourceActivityID)
	}
	if res.Activities[0].MovingTimeSec != 600 {
		t.Fatalf("unexpected moving time: %d", res.Activities[0].MovingTimeSec)
	}
	if res.LastSyncedAt.IsZero() {
		t.Fatal("expected last synced at")
	}
	if res.Activities[0].StartTime.After(time.Now().Add(24 * time.Hour)) {
		t.Fatal("unexpected start time")
	}
}

func TestConnector_EmptyCredentials(t *testing.T) {
	c := NewLive("", "", "http://example.com", nil)
	_, err := c.FetchActivities(context.Background(), "u1", syncjob.Checkpoint{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "keep credential is empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConnector_FetchActivities_WritesRawDecoded(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.1/users/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"token":"t1"}}`))
	})
	mux.HandleFunc("/pd/v3/stats/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"records":[{"logs":[{"stats":{"id":"run_1","isDoubtful":false}}]}],"lastTimestamp":0}}`))
	})

	geoPayload, _ := encodeRunmapForTestGeo([]map[string]any{{"latitude": 30.0, "longitude": 120.0, "timestamp": 1}})
	hrPayload, _ := encodeRunmapForTest([]map[string]any{{"timestamp": 1, "hr": 90}})

	mux.HandleFunc("/pd/v3/runninglog/run_1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"abc_1","startTime":1700000000000,"endTime":1700000600000,"duration":600,"distance":1000,"dataType":"outdoorRunning","timezone":"Asia/Shanghai","geoPoints":"` + geoPayload + `","heartRate":{"averageHeartRate":88,"heartRates":"` + hrPayload + `"}}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewLive("13000000000", "pass", srv.URL, srv.Client())
	res, err := c.FetchActivities(context.Background(), "u1", syncjob.Checkpoint{})
	if err != nil {
		t.Fatalf("fetch err: %v", err)
	}
	if len(res.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(res.Activities))
	}
	raw := res.Activities[0].Raw
	if raw["geo_points_raw"] == "" {
		t.Fatal("expected geo_points_raw")
	}
	if raw["geo_points_decoded"] == nil {
		t.Fatal("expected geo_points_decoded")
	}
	if raw["heart_rate_raw"] == nil {
		t.Fatal("expected heart_rate_raw")
	}
	if raw["heart_rates_decoded"] == nil {
		t.Fatal("expected heart_rates_decoded")
	}
}
