package keep

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeepClient_LoginAndFetchList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1.1/users/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"token":"t1"}}`))
	})
	mux.HandleFunc("/pd/v3/stats/detail", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer t1" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"records":[{"logs":[{"stats":{"id":"run_1","isDoubtful":false}}]}],"lastTimestamp":0}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := NewKeepClient(srv.URL, srv.Client())
	token, err := client.Login(context.Background(), "13000000000", "pass")
	if err != nil {
		t.Fatalf("login err: %v", err)
	}
	if token != "t1" {
		t.Fatalf("unexpected token: %s", token)
	}

	ids, last, err := client.FetchRunIDs(context.Background(), "t1", "running", 0)
	if err != nil {
		t.Fatalf("fetch ids err: %v", err)
	}
	if last != 0 {
		t.Fatalf("expected lastTimestamp 0, got %d", last)
	}
	if len(ids) != 1 || ids[0] != "run_1" {
		t.Fatalf("unexpected ids: %+v", ids)
	}
}
