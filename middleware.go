package main

import (
	"log/slog"
	"net/http"
	"time"
)

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next(w, r)

		duration := time.Since(start)
		slog.Info("request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", duration,
			"remote_addr", r.RemoteAddr,
		)
	}
}
