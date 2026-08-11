# Project Context — cachicamas

> Bootstrap artifact for SDD. Authored by `sdd-init`. Do not edit casually — downstream
> phases (`sdd-explore`, `sdd-propose`, `sdd-apply`, `sdd-verify`, `sdd-archive`) read
> this file to align on stack, conventions, and testing discipline.

## Identity

| Field        | Value                                                                |
| ------------ | -------------------------------------------------------------------- |
| Project      | `cachicamas`                                                         |
| Repo         | `https://github.com/witsaba/cachicamas`                              |
| Primary branch | `main`                                                             |
| Default branch strategy | feature branches → PR to `main`                          |
| Language preference | bilingual: English frontmatter, Spanish body acceptable for org-facing docs |

## Tech Stack (detected)

- **Runtime**: Go 1.26.3
- **HTTP framework**: `github.com/labstack/echo/v5` v5.2.1
- **Database**: PostgreSQL 18 (`postgres:18-alpine3.24`)
- **Telemetry**: OpenTelemetry (`otel`, `otelslog`, `sdk/log`, `sdk/trace`)
  - OTLP/gRPC exporters for both traces and logs
  - Gateway pattern: apps → OTel Collector → Jaeger
- **Tracing backend**: Jaeger v2 (`jaegertracing/jaeger:2.19.0`)
- **Orchestration**: docker-compose v2, single-node local dev stack on `cachicamas_network`
- **Frontend**: `frontend/` directory present (not yet inspected by SDD)

## Architecture

- **Modules**: three Go modules under `backend/` — `database_administrator` (API + migrations),
  `workspace_syncer` (git clone/validate worker), and `agent` (the 3-layer agentic stack, from
  milestone AI-39). No module imports another; `go.work` at the repo root is for editor ergonomics
  only.
- **Style**: Hexagonal (ports & adapters) under `backend/database_administrator/src/`
  and `backend/workspace_syncer/src/`. **`backend/agent` is layered, not hexagonal** —
  `src/ai` ← `src/agent` ← `src/coding` ← `src/cmd/cachicamas`, per
  [ADR 0005](../docs/adr/0005-promote-agent-stack-to-own-module.md).
  - `cmd/server/` — entrypoint (`main.go`)
  - `application/` — use cases (e.g., `health_service.go`)
  - `domain/` — entities and contracts (e.g., `health.go`)
  - `interfaces/` — adapters (`interfaces/http/` with `health_handler.go` + `health_handler_test.go`)
  - `otel/` — observability wiring (`logging.go`, `otel.go`)
- **Module path**: `github.com/cachicamas/backend/database_administrator`
- **Binary name**: `database_administrator`, built into `./bin/`
- **Database user**: `queen` (NOSUPERUSER, CREATEROLE, CREATEDB, REPLICATION) — provisioned via `infra/postgres/init/01-init.sql`
- **Init scripts**: mounted at `/docker-entrypoint-initdb.d` in `docker-compose.yaml` (one-shot, first-boot only)
- **Pinning discipline**: every base image is triple-pinned (image:tag@digest)

## Conventions

- Conventional commits (no `Co-Authored-By` trailer, no AI attribution)
- Tools installed into `./bin/` (`LOCALBIN`) so `$GOPATH/bin` stays clean
- `golangci-lint` pinned to `v2.9.0`; config at `backend/database_administrator/.golangci.yml`
- Linters enabled: `govet`, `errcheck`, `staticcheck`, `unused`, `revive`
- Logging: `slog` + `otelslog`, OTLP via env vars (no hardcoded endpoints)
- New top-level dependencies require an ADR — per module. For `backend/agent`, the OpenTelemetry
  **API** modules are pre-authorised by [ADR 0005 § D3](../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary)
  and the OTel **SDK** and exporters are restricted to `src/cmd/`; anything else still needs its own ADR

## Testing

| Capability         | Value                                                              |
| ------------------ | ------------------------------------------------------------------ |
| Test runner        | `go test ./...` (Makefile target: `make test` runs with `-race -v`) |
| Coverage command   | `make test/cover` (writes `coverage.out`, atomic mode)             |
| Linter command     | `make lint` (auto-installs `golangci-lint` if missing)             |
| Vet command        | `make vet`                                                         |
| Format command     | `make fmt` (`gofmt` + `goimports`)                                 |
| Build command      | `make build`                                                       |
| Coverage threshold | 0 (no enforcement at project level yet)                            |
| Strict TDD         | **enabled** (RED-GREEN-REFACTOR; tests-first)                      |

`openspec/config.yaml` already declares `apply.tdd: true`. Tests live next to the code
they exercise (e.g., `interfaces/http/health_handler_test.go`). Integration tests against
the compose stack are expected to grow as services stabilize.

## Tooling Versions (pinned)

| Tool            | Version                              |
| --------------- | ------------------------------------ |
| Go              | 1.26.3                               |
| golangci-lint   | v2.9.0                               |
| goimports       | `latest` (re-resolved at install)    |
| Echo            | v5.2.1                               |
| PostgreSQL      | 18-alpine3.24                        |
| Jaeger          | 2.19.0                               |
| OTel Collector  | contrib 0.137.0                      |
| uv              | 0.11.17 (host runtime for `scripts/*.py` shebangs) |

## Active Changes

- `cachicamas-deep-healthcheck` — proposal only (no spec/design/tasks yet)
- `cachicamas-tail-sampling` — proposal + spec + design + tasks + verify-report (in flight)

## SDD Configuration

- Artifact store: **hybrid** (Engram + filesystem)
- Persistence: Engram topic_key `sdd-init/cachicamas`; filesystem under `openspec/`
- Strict TDD: enabled
- Default PR review budget: 400 changed lines; chained PRs when `sdd-tasks` forecasts High risk

## Review Checklist (for reviewers)

- [ ] reviewer can confirm the detected Go version matches `go.mod`
- [ ] reviewer can confirm the test command `go test ./...` succeeds against `backend/database_administrator`
- [ ] reviewer can confirm `openspec/config.yaml` already had `tdd: true` before this init
- [ ] reviewer can confirm no source files under `backend/database_administrator/src/` were modified by init
- [ ] reviewer can confirm `openspec/project.md` reflects the current hexagonal layout (cmd/application/domain/interfaces/otel)
- [ ] for any change touching `backend/agent`: no package of it imports another backend module, and nothing outside `application/` and `cmd/server` imports it — both directions are covered by tests, not by convention