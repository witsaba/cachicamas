# ============================================================================
# Cachicamas — agent_chat microservice (chat archetype composition root)
#
# Multi-stage build:
#   1. builder   — golang:1.26.6-alpine3.24. Compiles a fully-static binary
#                  (CGO disabled — the chat binary is pure Go: Echo v5 +
#                  pgx/v5 stdlib driver, no C dependencies).
#   2. runtime   — gcr.io/distroless/static-debian12:nonroot. Smallest
#                  possible image (no shell, no libc, no busybox, no
#                  package manager). Works because the binary is fully
#                  static and does not need the dynamic linker.
#
# Pinning policy: the builder image is triple-pinned
# (Go major.minor.patch + alpine3.minor), matching the sibling services
# (backend/database_administrator/Dockerfile). Floating tags like
# 'golang:alpine3.24' would silently pull new Go releases on the next
# build. Every upgrade must be an explicit commit we can review.
#
# Outbound TLS: the model provider is hardwired to https://openrouter.ai
# (backend/agent/src/ai/openaicompat/openrouter/wrapper.go). The CA root
# bundle is copied from the builder into the distroless image — without
# it every provider request fails with an x509 "unknown authority" error.
#
# Note on healthchecks: this image has NO shell, NO wget, NO curl, NO nc.
# The `healthcheck:` block in docker-compose.yaml for this service is
# disabled for that reason — verify the service from the host instead:
#   curl -sS -o /dev/null -w '%{http_code}\n' \
#     http://localhost:8090/api/agent/conversations
#   # → 401 (the JWE cookie middleware rejects unauthenticated requests;
#          a 200 requires a valid Auth.js session cookie)
#
# Build context is the directory containing this Dockerfile
# (backend/agent/), so COPY paths are relative to it.
# ============================================================================

# ---------- 1. Build stage -------------------------------------------------
FROM golang:1.26.6-alpine3.24 AS builder

# Reproducibility knobs.
# CGO_ENABLED=0 produces a fully static binary (no glibc dependency), which
# is the simplest match for distroless/static (same rationale as the
# database_administrator Dockerfile).
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

# Fully-static binary.
# -trimpath strips local paths for reproducibility.
# -ldflags '-s -w' drops the symbol table (smaller binary).
# CGO_ENABLED=0 (set above) makes this static — no extldflags needed.
# Unlike database_administrator there is no build-metadata injection:
# cmd/chat declares no version vars to -X into.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
      -trimpath \
      -buildvcs=false \
      -ldflags="-s -w" \
      -o /out/agent_chat \
      ./src/cmd/chat

# ---------- 2. Runtime stage ----------------------------------------------
# distroless/static-debian12:nonroot — smallest distroless variant. No
# shell, no busybox, no /usr/bin/wget. Combined with the static binary
# above, this is the simplest possible runtime for a pure-Go HTTP service.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# CA roots — REQUIRED here (unlike a purely internal service): the chat
# binary dials https://openrouter.ai/api/v1 on every turn. Also covers
# OTLP/Postgres TLS if those ever move to TLS endpoints.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary. The build process produces it with +x already,
# so no --chmod is needed.
COPY --from=builder /out/agent_chat /app/agent_chat

# The nonroot image ships with a `nonroot` user (UID 65532).
USER nonroot:nonroot
WORKDIR /app

# The chat binary listens on :8080 by default (CACHICAMAS_CHAT_PORT).
EXPOSE 8080

# The service is a long-running HTTP server. The compose healthcheck is
# disabled because this image has no shell/wget/curl (see header comment).
ENTRYPOINT ["/app/agent_chat"]
CMD []

# Image metadata.
LABEL org.opencontainers.image.title="cachicamas/agent_chat" \
      org.opencontainers.image.description="Cachicamas chat agent microservice" \
      org.opencontainers.image.source="https://github.com/cachicamas/backend/agent" \
      org.opencontainers.image.licenses="MIT"
