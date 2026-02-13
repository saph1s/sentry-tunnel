package main

import (
	"log/slog"
	"net/http"
	"os"
	"sentry-tunnel/internal/build"
	"sentry-tunnel/internal/config"
	"sentry-tunnel/internal/tunnel"
	"time"
)

func main() {
	cfg := config.Load()

	handler := tunnel.NewHandler(cfg, &http.Client{
		Timeout: 10 * time.Second,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.HealthCheck)
	mux.HandleFunc("POST /tunnel", handler.Tunnel)

	slog.Info("starting sentry tunnel server",
		"version", build.Version,
		"commit", build.Commit,
		"date", build.Date,
	)

	slog.Info("configuration",
		"listen_addr", cfg.ListenAddr,
		"sentry_upstream_url", cfg.SentryUpstreamURL,
		"allowed_projects", len(cfg.AllowedProjects),
		"max_body_size", cfg.MaxBodySize,
		"trust_proxy", cfg.TrustProxy,
		"user_agent", cfg.UserAgent,
	)

	s := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := s.ListenAndServe(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
