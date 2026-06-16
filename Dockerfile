# syntax=docker/dockerfile:1
#
# RELAY — relay CLI image.
#
# Build:
#   docker build -t relay .
#
# Run:
#   docker run --rm ghcr.io/soundmatt/relay version
#   docker run --rm -v "$(pwd)":/project ghcr.io/soundmatt/relay conform <binary>

# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS build

WORKDIR /build

# Dependency manifest first for layer-cache efficiency.
COPY go.mod ./

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags=-static" \
    -o /bin/relay \
    ./cmd/relay

# ── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM alpine:3.20

# git for VCS provenance; ca-certificates for TLS (relay conform reaches out to
# protocol binaries that may need network access).
RUN apk add --no-cache git ca-certificates

COPY --from=build /bin/relay /usr/local/bin/relay

LABEL org.opencontainers.image.title="RELAY" \
      org.opencontainers.image.description="Real-time Embedded Link Abstraction Yoke — conformance and observability CLI" \
      org.opencontainers.image.version="0.1.0" \
      org.opencontainers.image.source="https://github.com/SoundMatt/RELAY" \
      org.opencontainers.image.licenses="MPL-2.0" \
      io.relay.tool="RELAY" \
      io.relay.language="go" \
      io.relay.binary="relay" \
      io.relay.spec-version="0.1"

# Mount your project at /project for relay conform and relay trace.
WORKDIR /project

ENTRYPOINT ["relay"]
CMD ["help"]
