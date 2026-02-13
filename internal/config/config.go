package config

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	// Listen address for the tunnel server
	ListenAddr string
	// Upstream URL for the Sentry server (e.g., https://sentry.example.com)
	SentryUpstreamURL string
	// Optional: List of allowed projects to forward events to (if empty, all projects are allowed)
	AllowedProjects map[string]struct{}
	// Optional: Maximum body size for incoming requests (in bytes)
	MaxBodySize int64
	// Optional: Whether to trust X-Forwarded-For headers for client IP (default: false)
	TrustProxy bool
	// Optional: Custom User-Agent header for upstream requests (default: "sentry-tunnel/1.0")
	UserAgent string
}

func Load() Config {
	cfg := Config{
		ListenAddr:        getEnv("LISTEN_ADDR", ":8100"),
		SentryUpstreamURL: getEnv("SENTRY_UPSTREAM", ""),
		AllowedProjects:   make(map[string]struct{}),
		MaxBodySize:       5 * 1024 * 1024, // Default to 5 MB
		TrustProxy:        getEnv("TRUST_PROXY", "false") == "true",
		UserAgent:         getEnv("USER_AGENT", "sentry-tunnel/1.0"),
	}

	if cfg.SentryUpstreamURL == "" {
		slog.Error("SENTRY_UPSTREAM is required")
		os.Exit(1)
	}

	cfg.SentryUpstreamURL = strings.TrimRight(cfg.SentryUpstreamURL, "/")

	if projects := os.Getenv("ALLOWED_PROJECTS"); projects != "" {
		for _, p := range strings.Split(projects, ",") {
			if id := strings.TrimSpace(p); id != "" {
				cfg.AllowedProjects[id] = struct{}{}
			}
		}
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
