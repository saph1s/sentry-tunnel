package tunnel

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sentry-tunnel/internal/build"
	"sentry-tunnel/internal/config"
	"sentry-tunnel/internal/envelope"
	"strings"
)

type Handler struct {
	cfg    config.Config
	client *http.Client
}

func NewHandler(cfg config.Config, client *http.Client) *Handler {
	return &Handler{
		cfg:    cfg,
		client: client,
	}
}

// clientIP extracts the client's IP address from the request, considering proxy headers if configured.
func (h *Handler) clientIP(r *http.Request) string {
	if h.cfg.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// First IP in the chain is the original client
			if ip := strings.SplitN(xff, ",", 2)[0]; ip != "" {
				return strings.TrimSpace(ip)
			}
		}
	}

	// Strip port from RemoteAddr (format: "ip:port" or "[ipv6]:port")
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// HealthCheck provides a simple endpoint to verify the tunnel is running.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": build.Version,
		"commit":  build.Commit,
		"date":    build.Date,
	})
}

// Tunnel handles incoming Sentry envelope requests, validates them, and forwards to the upstream Sentry server.
func (h *Handler) Tunnel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, h.cfg.MaxBodySize))
	if err != nil {
		slog.Warn("failed to read body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	projectID, err := envelope.ParseProjectID(body)
	if err != nil {
		slog.Warn("invalid envelope", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate project against allowlist
	if len(h.cfg.AllowedProjects) > 0 {
		if _, ok := h.cfg.AllowedProjects[projectID]; !ok {
			slog.Warn("blocked project", "project_id", projectID)
			http.Error(w, "project not allowed", http.StatusForbidden)
			return
		}
	}

	// Forward to Sentry
	upstreamURL := fmt.Sprintf("%s/api/%s/envelope/", h.cfg.SentryUpstreamURL, projectID)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, strings.NewReader(string(body)))
	if err != nil {
		slog.Error("failed to create upstream request", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/x-sentry-envelope")
	req.Header.Set("User-Agent", h.cfg.UserAgent)

	// Forward real client IP
	clientIP := h.clientIP(r)
	req.Header.Set("X-Forwarded-For", clientIP)
	req.Header.Set("X-Real-IP", clientIP)

	resp, err := h.client.Do(req)
	if err != nil {
		slog.Error("upstream request failed", "error", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
