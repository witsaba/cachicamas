# Spec — Workspace Syncer

> **Change**: `2026-07-08-workspace-sync-clone`
> **Phase**: spec (new)
> **Canonical path** (after `sdd-sync`): `openspec/specs/workspace-syncer/spec.md`
> **Format**: Given/When/Then + RFC 2119 keywords per `openspec/config.yaml`
> **Type convention**: all identifiers are `BIGSERIAL`/`BIGINT` (int64). The new service does NOT have a Postgres database; the IDs flow through as JSON numbers from `database_administrator`. No UUIDs.

## Purpose

Define the acceptance criteria for the new `workspace_syncer` Go service. This service is a thin internal data-plane that consumes the persisted GitHub `access_token` over HTTP, runs `git clone` + permission checks + a worktree probe, persists cloned data on a shared docker volume, and reports the outcome back to `database_administrator` via a callback. It has no public HTTP surface (no `GET /workspaces`, no `/health` exposed to the public internet, no auth middleware on a user-facing route). The single internal endpoint and the callback contract are the entire API.

The motivation is the deferred tech-debt item in `openspec/changes/archive/2026-07-06-workspaces/proposal.md`. Without this service, the workspaces surface is half-functional: the workspace row is a metadata stub and no validation of the OAuth token's `permissions.push === true` happens server-side.

## Architecture invariants

- **No Postgres access.** The new service has zero `database/sql` / pgx imports. It receives the `job_id` and `workspace_id` as int64 in the request body and reports outcomes via HTTP callback to `database_administrator`.
- **No public HTTP surface.** The Echo instance only listens on the docker network; no route is exposed to the public internet. The `/health` endpoint (if any) binds to a separate port or to `localhost` only.
- **System `git` via `os/exec`.** The service shells out to the `git` binary in the container. No `github.com/go-git/go-git` import. ADR at Engram `adr/workspace-syncer-git-impl` documents the choice.
- **Filesystem layout is canonical**: `/data/workspaces/{workspace_id}/{owner}/{repo}.git/` is the only path pattern the service uses. No arbitrary paths. The layout function is the single source of truth.
- **Token redaction.** A custom slog handler redacts any field matching `oauth_token`, `authorization`, or `token` (case-insensitive) from any log line.

## Requirements

### R-WSY-001 — Clone and validate

The service exposes a single internal endpoint `POST /internal/clone-and-validate` that accepts a `job_id`, `workspace_id`, `owner`, `repo`, `default_branch`, and `oauth_token`. It validates the token's permissions via `GET /repos/{owner}/{repo}` (checking `permissions.push === true`), clones the repository into the canonical filesystem path, runs a worktree probe, posts the outcome back to `database_administrator` via the callback, and returns `202 Accepted` to the caller.

#### Scenarios

- **S-WSY-001** — Given a valid request body `{job_id, workspace_id, owner, repo, default_branch, oauth_token}` AND the OAuth token grants `permissions.push === true` on the repo AND the clone completes successfully AND the worktree probe exits 0, when `POST /internal/clone-and-validate` is called, then the response is `202 Accepted` with `{"job_id": <id>, "status": "running"}` AND a `/data/workspaces/{workspace_id}/{owner}/{repo}.git/` directory exists AND a callback is posted to `database_administrator` with `status = "done"` AND `commit_sha_after = <the HEAD sha of the cloned tree>`.
- **S-WSY-002** — Given a valid request body AND the OAuth token's `permissions.push === false` (or the response from `GET /repos/{owner}/{repo}` returns 403), when `POST /internal/clone-and-validate` is called, then a callback is posted to `database_administrator` with `status = "failed"`, `error_code = "WORKSPACE_PERMISSIONS_INSUFFICIENT"`, `error_message = "Token lacks push permission on this repository."` AND the response is `202 Accepted` (the failure is reported via the callback, not the synchronous response).
- **S-WSY-003** — Given a valid request body AND the OAuth token is valid BUT the `default_branch` does not exist on the remote (e.g. the repo was force-pushed and the default branch was renamed), when `POST /internal/clone-and-validate` is called, then a callback is posted with `status = "failed"`, `error_code = "BRANCH_NOT_FOUND"`, `error_message = "Default branch '<name>' not found on remote."` AND no clone is performed.
- **S-WSY-004** — Given a valid request body AND the clone completes AND the worktree probe (`git worktree add /tmp/probe HEAD`) exits non-zero, when the probe finishes, then the cloned tree is removed (`rm -rf /data/workspaces/{workspace_id}/...`) AND a callback is posted with `status = "failed"`, `error_code = "WORKTREE_PROBE_FAILED"`, `error_message = "git worktree add exited with code <n>."`.
- **S-WSY-005** — Given a valid request body AND the clone takes longer than 90 seconds, when the timeout fires, then the in-flight `git` process is killed (`context.WithTimeout` + `cmd.Cancel`) AND the partial clone is removed AND a callback is posted with `status = "failed"`, `error_code = "CLONE_TIMEOUT"`, `error_message = "Repository clone took longer than 90 seconds."`.
- **S-WSY-006** — Given the OAuth token returns `401 Unauthorized` from GitHub (the token has been revoked or expired), when `POST /internal/clone-and-validate` is called, then the `GetRepository` preflight fails with 401 AND a callback is posted with `status = "failed"`, `error_code = "TOKEN_EXPIRED"`, `error_message = "GitHub token is no longer valid. Please reconnect GitHub."` AND no clone is attempted.
- **S-WSY-007** — Given a valid request body AND a sync_job is currently `running` for the same workspace_id (mid-flight, before the callback is posted), when a second `POST /internal/clone-and-validate` arrives for the same workspace_id, then the second call is rejected at the workspace_syncer level (idempotency by workspace_id) AND a callback is NOT posted (the original job is the source of truth) AND the response is `202 Accepted` with `{"job_id": <id>, "status": "running"}` (the existing in-flight job's id is returned, NOT a new one).

### R-WSY-002 — Bearer token middleware

Every request to `workspace_syncer` MUST be authenticated via the `INTERNAL_SERVICE_TOKEN` env var. The middleware reads `Authorization: Bearer <token>` and compares the value in constant time. Missing or wrong token → 401 with `{"error": "unauthorized", "message": "Invalid or missing service token."}`.

#### Scenarios

- **S-WSY-010** — Request with no `Authorization` header → response is `401 Unauthorized` with the locked envelope.
- **S-WSY-011** — Request with `Authorization: Bearer wrong-token` → response is `401 Unauthorized` with the locked envelope.
- **S-WSY-012** — Request with `Authorization: Bearer <INTERNAL_SERVICE_TOKEN>` → middleware passes; the handler runs.
- **S-WSY-013** — The comparison is constant-time (no early-exit on length mismatch or first-byte mismatch). Verified by a unit test that asserts `subtle.ConstantTimeCompare` is used.

### R-WSY-003 — Filesystem layout

The canonical path pattern is `/data/workspaces/{workspace_id}/{owner}/{repo}.git/`. The layout function is the single source of truth and rejects any `workspace_id` that is not ASCII digits (defense-in-depth against path traversal).

#### Scenarios

- **S-WSY-020** — `workspace_id = 42`, `owner = "octocat"`, `repo = "hello-world"` → the path is `/data/workspaces/42/octocat/hello-world.git/`.
- **S-WSY-021** — `workspace_id = "42/../etc"` → the layout function returns an error; no path is constructed.
- **S-WSY-022** — `workspace_id = ""` → the layout function returns an error.
- **S-WSY-023** — `workspace_id = -1` (negative) → the layout function returns an error.
- **S-WSY-024** — The bare mirror convention is used: the path ends in `.git/` and contains a bare git repository (no working tree files at the top level). The HEAD file, objects/ directory, and refs/ directory are present after a successful clone.

### R-WSY-004 — Cleanup hook

On startup, the service runs a sweep that removes any `/data/workspaces/{workspace_id}/...` directory whose `workspace_id` no longer has a live `sync_job` (queried via `GET /internal/live-workspaces` to `database_administrator` or via a periodic poll; the design phase ratifies the mechanism). The sweep is idempotent and bounded by a 30s timeout.

#### Scenarios

- **S-WSY-030** — On startup, a `GET /internal/live-workspaces` returns `{live_ids: [1, 2, 3]}` AND `/data/workspaces/` contains `1/`, `2/`, `3/`, `4/`, `5/`, then the sweep removes `4/` and `5/` and leaves `1/`, `2/`, `3/` intact.
- **S-WSY-031** — The sweep is idempotent: running it twice in a row produces the same filesystem state.
- **S-WSY-032** — The sweep respects a 30s timeout: if the live-workspaces lookup or the filesystem walk exceeds 30s, the sweep logs a warning and exits; it does not block startup indefinitely.

### R-WSY-005 — OTel & slog observability

Every clone execution emits an OTel span `clone.execute` with the locked attributes. Every log line is a structured slog line with `job_id`, `workspace_id`, `owner`, `repo`. The OAuth token is redacted from every log line by the custom slog handler.

#### Scenarios

- **S-WSY-040** — A successful clone emits a `clone.execute` span with `workspace.id`, `owner`, `repo`, `default_branch`, `clone.duration_ms`, `worktree.probe.exit_code = 0`, `worktree.probe.duration_ms`.
- **S-WSY-041** — A failed clone (any failure mode) emits the same span with `worktree.probe.exit_code` set to the exit code (or omitted if the clone failed before the probe).
- **S-WSY-042** — A log line containing the literal string `oauth_token = "gho_..."` is redacted to `oauth_token = "[REDACTED]"` by the custom slog handler. Verified by a unit test that asserts the redaction is applied before the line is written to stdout.
- **S-WSY-043** — A log line containing `Authorization: Bearer gho_...` is redacted to `Authorization: Bearer [REDACTED]`.

### R-WSY-006 — Service composition root

The service is a single Go binary built from `cmd/server/main.go`. The Makefile produces `./bin/workspace_syncer`. The Dockerfile triple-pins the base image and installs the `git` binary (Debian: `apt-get install -y git`; pin to `git 1:2.47.x` per the project's pinning discipline).

#### Scenarios

- **S-WSY-050** — `make build` produces `./bin/workspace_syncer` with no CGO dependencies.
- **S-WSY-051** — `make test` runs `go test -race -v ./...` and is green.
- **S-WSY-052** — `make lint` runs `go vet` and `golangci-lint` (v2.9.0) and is clean.
- **S-WSY-053** — The Dockerfile builds the binary, installs `git`, and produces a slim runtime image. The image's entrypoint is the `workspace_syncer` binary.
- **S-WSY-054** — The `INTERNAL_SERVICE_TOKEN` env var is read at startup; the service fails to boot with a clear error if it is empty.

## Non-functional requirements

### NFR-WSY-001 — Performance

- A typical clone (1 MiB repo) completes in < 5s p95.
- A large clone (100 MiB repo) completes in < 60s p95.
- The 90s timeout is a hard ceiling; a clone that exceeds it is aborted.

### NFR-WSY-002 — Security

- The OAuth token is held in memory only for the duration of a single clone. It is redacted from logs and never persisted to disk.
- The service-to-service bearer token is read from the env var at startup and held in memory; it is never logged.
- The filesystem path construction rejects any non-digit `workspace_id` to prevent path traversal.
- The `os/exec` invocation always passes args as a `[]string` to `exec.Command`; no `sh -c` with interpolated input. The git URL is constructed from sanitized `owner` and `repo` (validated against `^[a-zA-Z0-9._-]+$`).

### NFR-WSY-003 — Reliability

- The clone-and-validate pipeline is idempotent at the workspace level: a second call for the same `workspace_id` while one is in flight is rejected (R-WSY-001 S-WSY-007).
- The cleanup hook is idempotent: running it twice produces the same state.
- The service does NOT have a database; there is no DB-side state to corrupt. State is either on the filesystem (cloned trees) or in the request body (current job).

### NFR-WSY-004 — Observability

- Every clone execution emits an OTel span.
- Every log line is structured (slog) with `job_id`, `workspace_id`, `owner`, `repo`, and a `level`.
- The custom slog handler redacts `oauth_token`, `authorization`, and `token` fields (case-insensitive) from every line.

## Strict TDD posture

`openspec/config.yaml` declares `apply.tdd: true` and `apply.test_command: "go test ./..."`. The apply phase MUST record RED → GREEN → TRIANGULATE → REFACTOR evidence in `apply-progress.md` for PR-2, with the following minimum coverage:

- `infrastructure/git/runner_test.go` — at least 6 tests: clone happy path, clone with bad token, worktree probe, path-safety reject, shell injection reject, token redaction in logs.
- `infrastructure/git/layout_test.go` — at least 5 tests covering S-WSY-020 through S-WSY-024.
- `interfaces/http/handler_test.go` — at least 4 tests covering S-WSY-001 (happy path), S-WSY-002 (permissions), S-WSY-005 (timeout), S-WSY-007 (idempotency).
- `interfaces/http/middleware_test.go` — at least 4 tests covering S-WSY-010 through S-WSY-013.
- `application/clone_service_test.go` — at least 2 tests covering the use-case happy path and the use-case error mapping.
- `cmd/server/main_test.go` — at least 1 test asserting the service fails to boot with an empty `INTERNAL_SERVICE_TOKEN`.

## Out of scope (per proposal)

- Webhook ingestion from GitHub.
- PR auto-creation.
- Worktree merge/rebase or any worktree-based feature beyond the probe.
- Token encryption-at-rest.
- Multi-tenant keying of the service (single shared identity in v1).
- Web-based admin UI for the service (operators interact via docker compose logs + the database_administrator's `GET /workspaces/:id/sync` endpoint).
- HTTPS termination (the docker network is trusted; v1 runs behind the database_administrator's reverse proxy if a TLS surface is needed; the design phase ratifies).

## Acceptance criteria

The change is accepted when:

1. All scenarios in this spec are implemented and tested (RED → GREEN → TRIANGULATE → REFACTOR per task).
2. The `workspace_syncer` service boots with `make run` against a local docker compose stack.
3. The endpoint `POST /internal/clone-and-validate` accepts a request from `database_administrator` and clones a real GitHub repo (use `mocks-github-oauth` for the token).
4. The cleanup hook removes orphaned data on startup.
5. The token-redaction slog handler unit test passes.
6. The path-safety unit test rejects non-digit `workspace_id`s.
7. The shell-injection unit test rejects `owner` or `repo` values that contain shell metacharacters.
8. `make test` is green (race-clean); `make lint` is clean.
9. The Dockerfile builds the binary and installs `git`; the image runs end-to-end.
10. The ADR at `adr/workspace-syncer-git-impl` is persisted to Engram and referenced from the proposal.
