# syntax=docker/dockerfile:1.6

# Triple-pinned: Go 1.26.3 + alpine 3.22 + sha256 digest. The ':1.26.3-alpine'
# (or ':latest') tag would silently roll forward to whatever alpine the
# upstream maintainers publish next. Pin explicitly so security upgrades
# are a visible commit.
#
# PR-2a installs the `git` binary in the runtime stage; PR-2b uses it for
# the actual clone work.
FROM golang:1.26.3-alpine3.22@sha256:REPLACE_AT_BUILD_TIME AS build

WORKDIR /src

# Cache go.mod / go.sum separately from the source so a source-only edit
# doesn't bust the module download layer.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the rest of the source.
COPY . .

# Build a static binary. -trimpath strips the local filesystem paths from
# the binary so the build is reproducible. -ldflags '-s -w' strips the
# symbol table and DWARF info for a smaller image.
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/workspace_syncer ./src/cmd/server

# Runtime stage: minimal alpine with `git` installed. The `git` binary is
# REQUIRED for the clone + worktree probe (see ADR adr/workspace-syncer-git-impl).
# Pin the git package to a specific version for reproducibility.
FROM alpine:3.22@sha256:REPLACE_AT_BUILD_TIME AS runtime

# Install git at a pinned version. The version specifier (`=`) ensures
# reproducible builds across `docker build` invocations.
RUN apk add --no-cache git=2.47.3-r0 ca-certificates=20241121-r1 \
    && git --version

# Copy the static binary from the build stage.
COPY --from=build /out/workspace_syncer /usr/local/bin/workspace_syncer

# Run as a non-root user (the `git` operations need read/write to the
# workspace data dir; the non-root user is created by the alpine `adduser`
# helper).
RUN adduser -D -u 10001 syncer
USER syncer

# The data directory is mounted from the host (or a named volume) by
# docker-compose. The non-root user needs write access.
RUN mkdir -p /data/workspaces && chown syncer:syncer /data/workspaces
VOLUME ["/data/workspaces"]

EXPOSE 8081

# The entrypoint reads the INTERNAL_SERVICE_TOKEN env var and fails to
# boot with a clear error if it is empty (see src/cmd/server/main.go).
ENTRYPOINT ["/usr/local/bin/workspace_syncer"]