package server

import (
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"

	"stridewise/backend/internal/middleware"
)

func NewHTTPServer(addr string, internalToken string) *kratoshttp.Server {
	if addr == "" {
		addr = ":8000"
	}

	srv := kratoshttp.NewServer(
		kratoshttp.Address(addr),
		kratoshttp.Filter(middleware.InternalTokenFilter(internalToken)),
	)

	srv.Handle("/internal/health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	srv.Handle("/internal/metrics", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("metrics_placeholder 1\n"))
	}))

	return srv
}
