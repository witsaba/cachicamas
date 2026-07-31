# ============================================================================
# Cachicamas — database_administrator microservice
#
# Multi-stage build:
#   1. builder   — golang:1.26.5-alpine3.24. Compiles a fully-static binary
#                  (CGO disabled — the service has no C dependencies: only
#                  net/http + Echo v5, both pure Go).
#   2. runtime   — gcr.io/distroless/static-debian12:nonroot. Smallest
#                  possible image (no shell, no libc, no busybox, no
#                  package manager). Works because the binary is fully
#                  static and does not need the dynamic linker.
#
# Pinning policy: the builder image is triple-pinned
# (Go major.minor.patch + alpine3.minor). Floating tags like
# 'golang:alpine3.24' or 'golang:1.26-alpine' would silently pull
# new Go releases / new alpine versions on the next build. We want
# every upgrade to be an explicit commit that we can review.
#
# Note on healthchecks: this image has NO shell, NO wget, NO curl, NO nc.
# The `healthcheck:` block in docker-compose.yaml for this service is
# disabled for that reason — verify the service from the host with
# `curl http://localhost:8080/health`. If a real in-container healthcheck
# becomes necessary later, add a tiny static Go binary that does a single
# TCP connect to localhost:8080 and COPY it into the image.
#
# Build context is the directory containing this Dockerfile
# (backend/database_administrator/), so COPY paths are relative to it.
# ============================================================================

# ---------- 1. Build stage -------------------------------------------------
FROM golang:1.26.5-alpine3.24 AS builder

# Rebuild-triggerable build args. Bumping these busts the cache.
ARG BUILD_VERSION=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_TIME=unknown

# Tooling + reproducibility knobs.
# CGO_ENABLED=0 produces a fully static binary (no glibc dependency), which
# is the simplest match for distroless/static and avoids the CGO+dynamic-
# linker footgun (see the README if you ever need to re-enable CGO).
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOFLAGS="-mod=readonly" \
    GIT_TERMINAL_PROMPT=0

WORKDIR /src

# Cache go.mod/go.sum separately so dependency downloads are reusable.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download -x && \
    go mod verify

# Source. Copy AFTER deps so this layer only rebuilds when code changes.
COPY src/ ./src/

# Fully-static binary with build metadata baked in.
# -trimpath strips local paths for reproducibility.
# -ldflags '-s -w' drops the symbol table (smaller binary).
# CGO_ENABLED=0 (set above) makes this static — no extldflags needed.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
      -trimpath \
      -buildvcs=false \
      -ldflags="-s -w \
        -X 'main.version=${BUILD_VERSION}' \
        -X 'main.commit=${BUILD_COMMIT}' \
        -X 'main.buildTime=${BUILD_TIME}'" \
      -o /out/database_administrator \
      ./src/cmd/server

# ---------- 2. Runtime stage ----------------------------------------------
# distroless/static-debian12:nonroot — smallest distroless variant. No
# shell, no busybox, no /usr/bin/wget. Combined with the static binary
# above, this is the simplest possible runtime for a pure-Go HTTP service.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# OTel collectors / Postgres TLS need CA roots.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary. The build process produces it with +x already,
# so no --chmod is needed.
COPY --from=builder /out/database_administrator /app/database_administrator

# The nonroot image ships with a `nonroot` user (UID 65532).
USER nonroot:nonroot
WORKDIR /app

# The microservice listens on :8080 by default (matches SERVICE_PORT).
EXPOSE 8080

# The service is a long-running HTTP server. The compose healthcheck is
# disabled because this image has no shell/wget/curl (see header comment).
ENTRYPOINT ["/app/database_administrator"]
CMD []

# Image metadata.
LABEL org.opencontainers.image.title="cachicamas/database_administrator" \
      org.opencontainers.image.description="Cachicamas database administrator microservice" \
      org.opencontainers.image.source="https://github.com/cachicamas/backend/database_administrator" \
      org.opencontainers.image.licenses="MIT"