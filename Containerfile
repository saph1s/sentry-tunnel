FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod ./

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
    -X sentry-tunnel/internal/build.Version=${VERSION} \
    -X sentry-tunnel/internal/build.Commit=${COMMIT} \
    -X 'sentry-tunnel/internal/build.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o sentry-tunnel ./cmd/sentry-tunnel

FROM alpine:3.23

RUN apk --no-cache add ca-certificates

COPY --from=builder /build/sentry-tunnel /sentry-tunnel

EXPOSE 8100

ENTRYPOINT ["/sentry-tunnel"]