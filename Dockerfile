# syntax=docker/dockerfile:1

# ============================================================
# PHOENIX PANEL — multi-stage build
# Stage 1 builds a static Go binary; stage 2 is a minimal runtime.
# ============================================================

# ---- Build stage ----
FROM golang:1.22-alpine AS build

# git is needed for VCS-stamped builds; build-base for cgo-free sqlite driver
# (we use the pure-Go modernc sqlite driver, so cgo stays disabled).
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy source. (go.sum is generated during build via -mod=mod so the repo can
# be built without a committed go.sum; CI should commit one via `go mod tidy`.)
COPY . .

# Resolve and lock dependencies, then build a static binary.
ENV GOFLAGS=-mod=mod
RUN go mod tidy

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/phoenix ./cmd/phoenix

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S phoenix && adduser -S -G phoenix phoenix

WORKDIR /app

# Data dir for the sqlite database (mounted as a volume in compose).
RUN mkdir -p /app/data && chown -R phoenix:phoenix /app

COPY --from=build /out/phoenix /app/phoenix

USER phoenix

EXPOSE 8080

# Liveness probe hits the unauthenticated health endpoint.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/app/phoenix"]
