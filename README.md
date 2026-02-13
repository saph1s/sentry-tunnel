# Sentry Tunnel

A lightweight standalone proxy service that tunnels [Sentry](https://sentry.io) envelope requests from browser SDKs,
bypassing ad blockers that block direct Sentry ingestion endpoints.

## Why

Browser ad blockers (uBlock Origin, EasyList, etc.) block requests to Sentry by matching against known domains (
`sentry.io`, `*.ingest.sentry.io`) and paths (`/api/*/envelope/`, `/api/*/store/`). The Sentry SDK's [
`tunnel`](https://docs.sentry.io/platforms/javascript/troubleshooting/#using-the-tunnel-option) option allows sending
envelopes to a custom URL instead, which this service handles.

## How it works

1. SDK sends the envelope to your custom domain and path (e.g. `https://t.example.com/tunnel`)
2. Sentry Tunnel parses the envelope header, extracts the project ID from the DSN
3. Validates the project ID against an optional allowlist
4. Forwards the envelope to your self-hosted Sentry instance

Ad blockers see a POST request to `t.example.com/tunnel` — no suspicious domains or paths.

## Quick start

```yaml
# docker-compose.yml
services:
  sentry-tunnel:
    image: your-registry/sentry-tunnel:latest
    restart: unless-stopped
    ports:
      - "8090:8090"
    environment:
      SENTRY_UPSTREAM: "https://sentry.example.com"
      ALLOWED_PROJECTS: "1,2,3"
      TRUST_PROXY: "true"
```

## Configuration

| Variable           | Required | Default             | Description                                             |
|--------------------|----------|---------------------|---------------------------------------------------------|
| `SENTRY_UPSTREAM`  | Yes      | —                   | Sentry base URL (e.g. `https://sentry.example.com`)     |
| `ALLOWED_PROJECTS` | No       | *(allow all)*       | Comma-separated list of allowed project IDs             |
| `TRUST_PROXY`      | No       | `false`             | Trust `X-Forwarded-For` header for client IP resolution |
| `LISTEN_ADDR`      | No       | `:8100`             | Server listen address                                   |
| `USER_AGENT`       | No       | `sentry-tunnel/1.0` | User-Agent header for upstream requests                 |

## SDK configuration

```ts
import * as Sentry from "@sentry/nuxt";

Sentry.init({
    dsn: "https://key@sentry.example.com/1",
    tunnel: "https://t.example.com/tunnel",
});
```

The `dsn` still points to your Sentry instance (used for project/key identification), while `tunnel` points to this
service.

## Endpoints

| Method | Path      | Description                                         |
|--------|-----------|-----------------------------------------------------|
| `POST` | `/tunnel` | Accepts Sentry envelopes and forwards them upstream |
| `GET`  | `/health` | Health check, returns build info                    |

Health check response:

```json
{
  "status": "ok",
  "version": "1.0.0",
  "commit": "abc1234",
  "date": "2025-02-14T12:00:00Z"
}
```

## Building

### Docker / Podman

```bash
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  -t sentry-tunnel .
```

### Local

```bash
go build \
  -ldflags="-s -w \
    -X sentry-tunnel/internal/build.Version=1.0.0 \
    -X sentry-tunnel/internal/build.Commit=$(git rev-parse --short HEAD) \
    -X 'sentry-tunnel/internal/build.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
  -o sentry-tunnel ./cmd/sentry-tunnel
```

## Reverse proxy

Place nginx in front of the tunnel on a clean domain:

```nginx
server {
    listen 443 ssl;
    server_name t.example.com;

    location /tunnel {
        proxy_pass http://sentry-tunnel:8090;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

Set `TRUST_PROXY=true` so the tunnel uses `X-Forwarded-For` for client IP forwarding to Sentry.