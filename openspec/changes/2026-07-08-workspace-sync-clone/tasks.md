# Tasks — Workspace Sync Card & Repository Cloning

> **Change**: `2026-07-08-workspace-sync-clone`
> **Phase**: tasks
> **Project**: cachicamas
> **Date**: 2026-07-08
> **Inputs**: `explore.md`, `proposal.md`, `specs/workspaces/spec.md`, `specs/workspace-syncer/spec.md`, `design.md`
> **Forecast**: 5-7 chained PRs (refined from the 4-PR plan in the proposal; see §PR Splitting)
> **Review budget**: 400 changed lines per PR (locked by the session preflight)
> **Skill resolution**: `paths-injected` (no project skills required for tasks; orchestrator has the registry indexed for the apply phase)

---

## PR Splitting (forecast)

The design phase proposed a 4-PR split. After re-forecasting the per-PR diff, the actual implementation needs more PRs because PR-2 (workspace_syncer skeleton) and PR-3 (database_administrator extension) each exceed 400 changed lines when treated as a single PR. The refined split is:

| PR | Goal | Forecast (added lines) | Depends on | Branch |
| --- | --- | ---: | --- | --- |
| 1 | Migration: `sync_job` table + ALTER `workspace` | ~150 | — | `feat/2026-07-08-workspace-sync-clone-pr1` |
| 2a | workspace_syncer scaffolding + middleware | ~380 | PR-1 | `feat/2026-07-08-workspace-sync-clone-pr2a` |
| 2b | workspace_syncer clone + handler | ~400 | PR-2a | `feat/2026-07-08-workspace-sync-clone-pr2b` |
| 2c | workspace_syncer sweep + redaction + OTel + compose | ~280 | PR-2b | `feat/2026-07-08-workspace-sync-clone-pr2c` |
| 3a | database_administrator: `sync_job` repo + service + domain | ~400 | PR-1 | `feat/2026-07-08-workspace-sync-clone-pr3a` |
| 3b | database_administrator: handlers + main.go + github/client + callback | ~500 | PR-3a, PR-2c | `feat/2026-07-08-workspace-sync-clone-pr3b` |
| 4 | Frontend: `WorkspaceSyncCard` + polling + API client | ~600 | PR-3b | `feat/2026-07-08-workspace-sync-clone-pr4` |

**Total: 7 PRs, ~2,710 changed lines** (including tests and the new service directory).

**Forecast risk:** PR-3b is at the upper bound of the budget (500 lines). The apply phase may need to split it further into PR-3b-1 (handlers + main.go) and PR-3b-2 (github client + callback) if the actual diff exceeds 400. The tasks are written to be cuttable at the handler/client boundary.

**Forecast risk:** PR-4 is over the budget at 600 lines. The apply phase may need to split it into PR-4-1 (card + polling hook + spec) and PR-4-2 (API client + mount + route-guard spec update) if the actual diff exceeds 400. The tasks are written to be cuttable at the API client boundary.

The chained PR strategy is the auto-forecast preflight choice; the apply phase is allowed to chain PRs without re-asking the user.

---

## Cross-PR prerequisites

### T-WSY-0-001 — Persist ADR `adr/workspace-syncer-internal-auth`

Record the cross-service auth posture decision (static bearer + docker network trust v1; HMAC JWT v2) in Engram with `topic_key: adr/workspace-syncer-internal-auth`, `type: decision`, `scope: project`, `project: cachicamas`. The ADR content is in `design.md` §9.

**Evidence:** `mem_save` call returns an id; the observation is referenceable from the proposal and the apply-progress.

**Strict TDD:** N/A (no test code).

**Estimated LoC:** ~20 (the ADR content, which mostly duplicates §9 of the design).

**Depends on:** — (can be done in parallel with any other task).

### T-WSY-0-002 — Persist ADR `adr/workspace-syncer-git-impl`

Record the git-implementation decision (os/exec + system `git` over `go-git`) in Engram with `topic_key: adr/workspace-syncer-git-impl`, `type: decision`, `scope: project`, `project: cachicamas`. The ADR content is in `design.md` §9.

**Evidence:** `mem_save` call returns an id; the observation is referenceable from the proposal and the apply-progress.

**Strict TDD:** N/A (no test code).

**Estimated LoC:** ~15.

**Depends on:** — (can be done in parallel).

---

## PR-1 — Migration (forecast: ~150 LoC)

**Branch:** `feat/2026-07-08-workspace-sync-clone-pr1`
**Target:** `main`
**PR description prefix:** `feat(db): sync_job table + workspace sync columns`

### T-WSY-1-001 — Write the `sync_job` migration (RED → GREEN)

Write the migration file `backend/database_administrator/src/migration/sql/20260708120000_sync_job.sql` that creates the `sync_job` table with BIGSERIAL id, FK to workspace, status CHECK, triggered_by CHECK, started_at, finished_at, commit_sha_after, error_message, error_code, attempts, created_at. Plus ALTER on `workspace` adding the 4 columns (`last_synced_at`, `last_synced_commit_sha`, `default_branch`, `last_sync_job_id`). Plus indexes (single-flight partial unique, secondary on workspace_id and status). Plus the `Down` section.

**Strict TDD evidence (in `apply-progress.md`):**

- **RED:** Write the migration test FIRST in `backend/database_administrator/src/migration/runner_test.go` extension. The test asserts the new table exists and has the locked columns. Before the migration file exists, the test compiles but the migration runner fails to apply the missing file (or the test is skipped with a clear note that the migration is absent).
- **GREEN:** Write the migration file. Re-run the test. Passes.
- **TRIANGULATE:** Add a test for the partial unique index — try to insert two `pending` rows for the same `workspace_id`; the second must fail with SQLSTATE 23505. Add a test for the FK `ON DELETE SET NULL` on `workspace.last_sync_job_id` — delete a sync_job referenced by a workspace, the workspace's `last_sync_job_id` is set to NULL.
- **REFACTOR:** Consolidate the migration into one transaction. Add `IF NOT EXISTS` / `IF EXISTS` guards for idempotency. Re-run all tests; no regression.

**Estimated LoC:** 80 (migration file) + 50 (test extension).

### T-WSY-1-002 — Write the optional cleanup migration

Write the optional housekeeping migration `backend/database_administrator/src/migration/sql/20260708120100_workspace_sync_cleanup_metadata.sql`. The migration is a no-op stub: it documents the decision to NOT backfill `default_branch` for pre-existing workspaces and leaves them NULL (the next sync populates the value). The migration file exists for traceability but its `Up` is a no-op SELECT that returns 1.

**Strict TDD evidence:** N/A (the migration is a no-op; no test code).

**Estimated LoC:** 20.

**Optional:** The tasks phase marks this as optional. The apply phase may skip it if the team prefers to keep the change surface small.

### T-WSY-1-003 — Grant CRUD on `sync_job` to the `queen` role

Modify `infra/postgres/init/01-init.sql` to add `GRANT SELECT, INSERT, UPDATE, DELETE ON sync_job TO queen;`. (The migration test exercises this grant; the change is required for the API service to access the table in production.)

**Strict TDD evidence:** The migration test in T-WSY-1-001 asserts the grant is present (the test runs as the `queen` role and can read/insert the table).

**Estimated LoC:** 5 (1 line + comment).

### PR-1 review focus

- Schema correctness: the `sync_job` columns are exactly the ones in `design.md` §3.
- The partial unique index clause `WHERE status IN ('pending','running')` is correct.
- The FK `ON DELETE SET NULL` on `workspace.last_sync_job_id` is correct.
- All new IDs are `BIGSERIAL`, not `UUID`.
- The `Down` section drops the new table and columns in the correct order.
- The optional cleanup migration is honest (no-op, not a backfill).

### PR-1 acceptance criteria

- [ ] `cd backend/database_administrator && make test` is green; the new migration test passes.
- [ ] The migration runs on a fresh database; the new table and columns exist.
- [ ] Re-running the migration is a no-op (the `IF NOT EXISTS` guards work).
- [ ] The diff is under 400 added lines.

---

## PR-2a — workspace_syncer scaffolding + middleware (forecast: ~380 LoC)

**Branch:** `feat/2026-07-08-workspace-sync-clone-pr2a`
**Target:** `main`
**PR description prefix:** `feat(workspace_syncer): scaffold service + token middleware + domain`
**Depends on:** PR-1

### T-WSY-2a-001 — Initialize the `backend/workspace_syncer` module

Create `backend/workspace_syncer/go.mod` with `module github.com/cachicamas/backend/workspace_syncer` and `go 1.26.3`. The dependencies are: `github.com/labstack/echo/v5 v5.2.1`, `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/trace`, `go.opentelemetry.io/otel/trace/noop`, `golang.org/x/crypto` (for `subtle.ConstantTimeCompare`). Run `go mod tidy` to produce `go.sum`.

**Strict TDD evidence:** The `go.mod` and `go.sum` are not test-driven; they are infrastructure. The first test in this PR is T-WSY-2a-002.

**Estimated LoC:** 30.

### T-WSY-2a-002 — Domain types: `CloneRequest`, `CloneResult`, validation, errors (RED → GREEN)

Create `backend/workspace_syncer/src/domain/clone.go` with the `CloneRequest` struct (job_id, workspace_id, owner, repo, default_branch, oauth_token), the `CloneResult` struct (status, commit_sha_after, error_code, error_message), and the `ValidateCloneRequest` function (rejects empty fields, negative workspace_id, non-alphanumeric owner/repo). Create `backend/workspace_syncer/src/domain/errors.go` with the locked error vocabulary (`ErrInvalidRequest`, `ErrUnauthorized`, `ErrTokenExpired`, `ErrInsufficientPermissions`, `ErrBranchNotFound`, `ErrWorktreeProbeFailed`, `ErrCloneTimeout`, `ErrCloneFailed`). Write tests FIRST in `clone_test.go` and `errors_test.go`.

**Strict TDD evidence:**

- **RED:** Write the test that asserts `ValidateCloneRequest` rejects an empty `owner`. The test fails because the function doesn't exist.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for each rejection case (empty repo, negative workspace_id, non-alphanumeric owner, non-alphanumeric repo, empty default_branch, empty oauth_token).
- **REFACTOR:** Extract the alphanumeric check as a helper. Re-run tests.

**Estimated LoC:** 60 (domain) + 40 (test) + 40 (errors) + 30 (errors test).

### T-WSY-2a-003 — Bearer-token middleware (RED → GREEN)

Create `backend/workspace_syncer/src/infrastructure/token/middleware.go` with the `ServiceTokenMiddleware(expected string) echo.MiddlewareFunc` function. The middleware reads `Authorization: Bearer <token>`, compares the value with `subtle.ConstantTimeCompare` against the expected token, and returns 401 on mismatch. Write tests FIRST in `middleware_test.go`.

**Strict TDD evidence:**

- **RED:** Write the test that asserts a request with no `Authorization` header returns 401. The test fails because the middleware doesn't exist.
- **GREEN:** Write the minimum middleware. Passes.
- **TRIANGULATE:** Add tests for: wrong token, correct token, missing `Bearer` prefix, case-sensitivity of `Bearer` (should be case-insensitive per RFC 7235).
- **REFACTOR:** Extract the `expectedBytes` slice once at construction. Re-run tests.

**Estimated LoC:** 30 (middleware) + 40 (test).

### T-WSY-2a-004 — `cmd/server/main.go` composition root (boot-only, returns 501)

Create `backend/workspace_syncer/src/cmd/server/main.go` with the composition root: read `INTERNAL_SERVICE_TOKEN` env var, fail to boot if empty, init the Echo instance, register a single placeholder `POST /internal/clone-and-validate` route that returns `501 Not Implemented` (the route exists for PR-2a, the implementation comes in PR-2b), register the `ServiceTokenMiddleware`, init the OTel no-op tracer, bind to `:8081` (or `:${WORKSPACE_SYNCER_PORT}`). Write a test that asserts the service fails to boot with an empty `INTERNAL_SERVICE_TOKEN`.

**Strict TDD evidence:**

- **RED:** Write the test that calls `main()` (or a `boot()` helper) with an empty env var. The test fails because the function doesn't exist.
- **GREEN:** Write the minimum boot logic. Passes.
- **TRIANGULATE:** Add tests for: the placeholder route returns 501 with the locked envelope, the middleware is applied (a request without the token returns 401).
- **REFACTOR:** Extract the boot logic from `main()` into a testable `boot()` function. Re-run tests.

**Estimated LoC:** 90 (main.go) + 30 (test).

### T-WSY-2a-005 — `Makefile`, `Dockerfile`, `.dockerignore`, `.golangci.yml`, `README.md`

Create the supporting files. The `Makefile` mirrors `backend/database_administrator/Makefile` (build, test, lint, run, tools targets). The `Dockerfile` triple-pins the base image (e.g. `golang:1.26.3-alpine3.22@sha256:...`), runs `go mod download`, builds the binary, and produces a slim runtime image with the binary only. The `git` binary is NOT installed in this PR (it comes with the next PR-2b). The `.dockerignore` and `.golangci.yml` match the database_administrator versions. The `README.md` documents the `/internal/clone-and-validate` endpoint (the contract is locked; the implementation is in PR-2b).

**Strict TDD evidence:** N/A (these are infrastructure files).

**Estimated LoC:** 40 (Dockerfile) + 80 (Makefile) + 10 (.dockerignore) + 30 (.golangci.yml) + 60 (README.md).

### PR-2a review focus

- The `go.mod` module path is `github.com/cachicamas/backend/workspace_syncer` (not `database_administrator`).
- The `INTERNAL_SERVICE_TOKEN` env var is required at boot; the service fails fast with a clear error.
- The `ServiceTokenMiddleware` uses `subtle.ConstantTimeCompare` (no early-exit on length mismatch).
- The `cmd/server/main.go` composition root is testable; the test asserts the boot-without-token error.
- The Makefile and Dockerfile match the existing project's pattern.
- The diff is under 400 added lines.

### PR-2a acceptance criteria

- [ ] `cd backend/workspace_syncer && make test` is green.
- [ ] `cd backend/workspace_syncer && make lint` is clean.
- [ ] The service fails to boot with an empty `INTERNAL_SERVICE_TOKEN`.
- [ ] The placeholder `POST /internal/clone-and-validate` returns 501.
- [ ] The middleware rejects requests without the token.
- [ ] The diff is under 400 added lines.

---

## PR-2b — workspace_syncer clone + handler (forecast: ~400 LoC)

**Branch:** `feat/2026-07-08-workspace-sync-clone-pr2b`
**Target:** `main`
**PR description prefix:** `feat(workspace_syncer): clone + worktree probe + handler`
**Depends on:** PR-2a

### T-WSY-2b-001 — `infrastructure/git/layout.go` — canonical filesystem path

Create `backend/workspace_syncer/src/infrastructure/git/layout.go` with the `WorkspacePath(workspaceID int64, owner, repo string) (string, error)` function (per `design.md` §4). Write tests FIRST in `layout_test.go`.

**Strict TDD evidence:**

- **RED:** Test that `WorkspacePath(42, "octocat", "hello-world")` returns `/data/workspaces/42/octocat/hello-world.git`. Fails.
- **GREEN:** Write the function. Passes.
- **TRIANGULATE:** Add tests for negative workspace_id, empty owner, empty repo, non-alphanumeric owner (e.g. `"octo cat"`), non-alphanumeric repo.
- **REFACTOR:** Extract `validRepoSegment` as a private helper with the `^[a-zA-Z0-9._-]+$` regex. Re-run tests.

**Estimated LoC:** 40 (layout) + 50 (test).

### T-WSY-2b-002 — `infrastructure/git/runner.go` — git clone + worktree probe (RED → GREEN)

Create `backend/workspace_syncer/src/infrastructure/git/runner.go` with the `Runner` struct (fields: `gitPath string` (default `git`), `fs fs.FS`, `exec Executor`). The `Runner` has methods: `Clone(ctx, workspaceID, owner, repo, oauthToken) (path, error)`, `WorktreeProbe(ctx, path) (headSHA, error)`, `ResolveHead(ctx, path) (sha, error)`. The `Clone` method runs `git clone --bare https://x-access-token:<token>@github.com/{owner}/{repo}.git {path}` with a 90s `context.WithTimeout`. The `WorktreeProbe` method runs `git -C {path} worktree add /tmp/probe-{id} HEAD` and checks the exit code. Write tests FIRST in `runner_test.go` using a temp dir and a real `git` binary (the test environment has `git` available; the test skips if `git` is not present).

**Strict TDD evidence:**

- **RED:** Test that `Clone` runs the expected `git` command and creates the expected path. Fails because the function doesn't exist.
- **GREEN:** Write the minimum `Clone` using `os/exec.CommandContext`. Passes.
- **TRIANGULATE:** Add tests for: bad token (the `git clone` command exits non-zero), timeout (the 90s fires), worktree probe failure (the probe command exits non-zero), shell injection (an owner value like `octo;rm -rf /` is rejected by `validRepoSegment`).
- **REFACTOR:** Extract the `os/exec` wrapper into a small `Executor` interface so the test can use a fake. Re-run tests.

**Estimated LoC:** 120 (runner) + 150 (test).

### T-WSY-2b-003 — `infrastructure/httpclient/callback_client.go` — database_administrator callback

Create `backend/workspace_syncer/src/infrastructure/httpclient/callback_client.go` with the `CallbackClient` struct (fields: `baseURL string`, `token string`, `httpClient *http.Client`). The `CallbackClient` has one method: `Post(ctx, path string, body any) error`. The method POSTs JSON to the database_administrator's callback URL with `Authorization: Bearer <token>`. Write tests FIRST using `httptest.NewServer` to mock the database_administrator.

**Strict TDD evidence:**

- **RED:** Test that `Post` sends a POST with the body and the bearer token. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: 4xx response (returns an error), 5xx response (returns an error with a retry hint), network error (returns the error).
- **REFACTOR:** Extract the JSON encoding into a helper. Re-run tests.

**Estimated LoC:** 60 (client) + 40 (test).

### T-WSY-2b-004 — `application/clone_service.go` — use case

Create `backend/workspace_syncer/src/application/clone_service.go` with the `CloneService` struct (fields: `runner *git.Runner`, `callback *httpclient.CallbackClient`, `logger *slog.Logger`). The `CloneService` has one method: `CloneAndValidate(ctx, req domain.CloneRequest) error`. The method validates the request, calls `runner.Clone`, calls `runner.WorktreeProbe`, calls `runner.ResolveHead`, and posts the result via `callback.Post`. Errors are translated to the database_administrator envelope codes. Write tests FIRST using fakes for `runner` and `callback`.

**Strict TDD evidence:**

- **RED:** Test that `CloneAndValidate` returns nil on the happy path. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for each error path (bad token → `WORKSPACE_PERMISSIONS_INSUFFICIENT`, worktree probe fail → `WORKTREE_PROBE_FAILED`, etc.).
- **REFACTOR:** Extract the error-to-code mapping as a `map[error]string` lookup. Re-run tests.

**Estimated LoC:** 80 (service) + 60 (test).

### T-WSY-2b-005 — `interfaces/http/handler.go` — replace the placeholder route

Modify `backend/workspace_syncer/src/interfaces/http/handler.go` (create in this PR) to replace the 501 placeholder from PR-2a with the full implementation: parse the JSON body, validate the request, call `CloneService.CloneAndValidate` in a goroutine, return `202 Accepted` with `{"job_id": <id>, "status": "running"}`. The goroutine handles the long-running clone; the response is immediate. Write tests FIRST in `handler_test.go` using a fake `CloneService`.

**Strict TDD evidence:**

- **RED:** Test that the handler returns 202 with the locked body on a valid request. Fails (the placeholder returns 501).
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: missing field (400), invalid workspace_id (400), wrong token (401), empty body (400).
- **REFACTOR:** Extract the body parsing into a helper. Re-run tests.

**Estimated LoC:** 100 (handler) + 120 (test).

### T-WSY-2b-006 — `cmd/server/main.go` — wire the new dependencies

Modify `backend/workspace_syncer/src/cmd/server/main.go` to construct the `Runner`, the `CallbackClient`, the `CloneService`, and the handler. Replace the placeholder route registration with the new handler. Write a test that asserts the full request flow (a valid request returns 202; the goroutine eventually posts the callback).

**Strict TDD evidence:**

- **RED:** The integration test in `cmd/server/main_test.go` fails because the dependencies are not wired.
- **GREEN:** Wire them. Passes.
- **TRIANGULATE:** Add tests for the env var defaults (`WORKSPACE_SYNCER_PORT` defaulting to 8081, `DATABASE_ADMINISTRATOR_URL` defaulting to `http://database_administrator:8080`).
- **REFACTOR:** Extract the wiring into a `newServer(cfg Config) *echo.Echo` helper. Re-run tests.

**Estimated LoC:** 30 (diff to main.go) + 30 (test extension).

### PR-2b review focus

- `os/exec` args are always passed as a `[]string`; no `sh -c` with interpolated input.
- The `git` URL is constructed from sanitized `owner` and `repo` (already validated by `validRepoSegment`).
- The token is held in memory only for the duration of the clone.
- The worktree probe runs in a separate temp dir, not the bare mirror path.
- The 90s timeout is enforced via `context.WithTimeout` and `cmd.Cancel`.
- The handler returns 202 immediately; the clone runs in a goroutine; the callback is posted on completion or failure.
- The diff is under 400 added lines.

### PR-2b acceptance criteria

- [ ] `cd backend/workspace_syncer && make test` is green.
- [ ] `cd backend/workspace_syncer && make lint` is clean.
- [ ] The handler returns 202 with `{"job_id": <id>, "status": "running"}` on a valid request.
- [ ] The handler returns 401 without a valid bearer token.
- [ ] The handler returns 400 on a missing or malformed body.
- [ ] The runner tests pass (clone happy path, bad token, timeout, worktree probe fail, shell injection reject).
- [ ] The diff is under 400 added lines.

---

## PR-2c — workspace_syncer sweep + redaction + OTel + compose (forecast: ~280 LoC)

**Branch:** `feat/2026-07-08-workspace-sync-clone-pr2c`
**Target:** `main`
**PR description prefix:** `feat(workspace_syncer): sweep + token redaction + OTel + compose`
**Depends on:** PR-2b

### T-WSY-2c-001 — `infrastructure/git/sweep.go` — startup cleanup

Create `backend/workspace_syncer/src/infrastructure/git/sweep.go` with the `Sweep(ctx, liveIDs, fs, log) error` function (per `design.md` §4). The sweep walks `/data/workspaces/`, parses each entry name as an int64, removes it if not in `liveIDs`, and is bounded by a 30s timeout. The `FS` interface is small (`ReadDir`, `RemoveAll`) so the test can use a fake in-memory filesystem. Write tests FIRST in `sweep_test.go`.

**Strict TDD evidence:**

- **RED:** Test that `Sweep` removes an entry not in `liveIDs`. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: idempotency (running twice produces the same state), the 30s timeout, the non-numeric entry case (the sweep logs a warning and continues).
- **REFACTOR:** Extract the `FS` interface. Re-run tests.

**Estimated LoC:** 60 (sweep) + 50 (test).

### T-WSY-2c-002 — `otel/logging.go` — token redaction slog handler

Modify `backend/workspace_administrator/src/otel/logging.go` (created in PR-2a as basic slog) to add a custom slog handler that redacts any field matching `oauth_token`, `authorization`, or `token` (case-insensitive). Write tests FIRST in `logging_test.go`.

**Strict TDD evidence:**

- **RED:** Test that a log line containing `oauth_token = "gho_..."` is redacted to `oauth_token = "[REDACTED]"`. Fails.
- **GREEN:** Write the custom handler. Passes.
- **TRIANGULATE:** Add tests for: the `Authorization` header field, the `access_token` field, nested fields (e.g. `request.headers.authorization`), case-insensitive matching.
- **REFACTOR:** Extract the redaction logic as a helper function. Re-run tests.

**Estimated LoC:** 30 (diff to logging.go) + 30 (test).

### T-WSY-2c-003 — `otel/otel.go` — real OTel tracer

Modify `backend/workspace_syncer/src/otel/otel.go` to upgrade from the no-op tracer to a real OTel tracer using the OTLP exporter. The `clone.execute` span is opened in `application/clone_service.go` with the locked attributes (workspace.id, owner, repo, default_branch, clone.duration_ms, worktree.probe.exit_code, worktree.probe.duration_ms). The `OTEL_EXPORTER_OTLP_ENDPOINT` env var is read at startup (no fallback to no-op; the service logs a warning if the env var is empty and falls back to no-op).

**Strict TDD evidence:**

- **RED:** Test that the `clone.execute` span is opened with the expected attributes. Fails (the no-op tracer doesn't emit anything).
- **GREEN:** Wire the real OTel tracer. Passes.
- **TRIANGULATE:** Add tests for: the env-var-empty fallback to no-op, the span attribute values, the duration measurement.
- **REFACTOR:** Extract the tracer init as a `NewTracer(ctx) (trace.Tracer, error)` helper. Re-run tests.

**Estimated LoC:** 40 (diff to otel.go) + 30 (test).

### T-WSY-2c-004 — `cmd/server/main.go` — wire the sweep

Modify `backend/workspace_syncer/src/cmd/server/main.go` to call the sweep on startup. The sweep runs in a goroutine after the Echo server starts. The sweep fetches the live workspace IDs from `database_administrator` via a new internal endpoint `GET /internal/live-workspaces` (added in PR-3b; for PR-2c, the sweep uses an empty `liveIDs` map, which is acceptable for the v1 behavior — the next sync of a soft-deleted workspace will fail to find the cloned data and re-clone).

**Strict TDD evidence:**

- **RED:** Test that `main()` calls the sweep. Fails.
- **GREEN:** Wire the sweep. Passes.
- **TRIANGULATE:** Add tests for: the sweep error is logged but does not abort startup, the sweep runs in a goroutine (the test does not block).
- **REFACTOR:** Extract the sweep call into a `startupSweep(ctx, ...)` helper. Re-run tests.

**Estimated LoC:** 20 (diff to main.go) + 20 (test).

### T-WSY-2c-005 — `docker-compose.yaml` and `docker-compose.vps.yaml`

Modify the top-level `docker-compose.yaml` and `docker-compose.vps.yaml` to add the `workspace_syncer` service, the network link to `database_administrator`, and the shared volume `cachicamas_synced_repos`. The `INTERNAL_SERVICE_TOKEN` env var is required on both services (use `${VAR:?...}` syntax for fail-fast).

**Strict TDD evidence:** N/A (compose files are not test-driven; the integration test in PR-3b's triangulation phase exercises the compose stack).

**Estimated LoC:** 25 (per compose file) = 50 total.

### T-WSY-2c-006 — `Dockerfile` — install `git`

Modify the `Dockerfile` to install the `git` binary in the runtime image (Debian: `apt-get install -y git`; pin to `git 1:2.47.x` per the project's pinning discipline).

**Strict TDD evidence:** The Dockerfile is not test-driven. The integration test in PR-3b asserts the `git` binary is present in the running container.

**Estimated LoC:** 10 (diff to Dockerfile).

### PR-2c review focus

- The sweep is idempotent and bounded by 30s.
- The redaction handler covers `oauth_token`, `authorization`, `access_token`, and `token` (case-insensitive).
- The OTel tracer falls back to no-op if the env var is empty (does not crash).
- The sweep error does not abort startup.
- The compose file uses `${VAR:?...}` syntax for the shared token.
- The Dockerfile installs the `git` binary in the runtime image.
- The diff is under 400 added lines.

### PR-2c acceptance criteria

- [ ] `cd backend/workspace_syncer && make test` is green.
- [ ] `cd backend/workspace_syncer && make lint` is clean.
- [ ] The sweep removes entries not in `liveIDs` on startup.
- [ ] The redaction handler redacts the expected fields.
- [ ] The OTel tracer emits the `clone.execute` span with the expected attributes.
- [ ] The compose stack boots with `docker compose up workspace_syncer`; the service binds to `:8081` and accepts requests.
- [ ] The diff is under 400 added lines.

---

## PR-3a — database_administrator: `sync_job` repo + service + domain (forecast: ~400 LoC)

**Branch:** `feat/2026-07-08-workspace-sync-clone-pr3a`
**Target:** `main`
**PR description prefix:** `feat(db_admin): sync_job domain + repo + service`
**Depends on:** PR-1

### T-WSY-3a-001 — Domain extension: `SyncJob`, errors, port

Modify `backend/database_administrator/src/domain/workspace.go` (or create a new `sync.go` file in the same package) to add the `SyncJob` aggregate (`ID int64`, `WorkspaceID int64`, `Status string`, `TriggeredBy string`, `StartedAt *time.Time`, `FinishedAt *time.Time`, `CommitShaAfter *string`, `ErrorMessage *string`, `ErrorCode *string`, `Attempts int`, `CreatedAt time.Time`). Add the `SyncJobRepository` port (Insert, Update, GetLatestForWorkspace, LockActiveForWorkspace). Add the new errors (`ErrSyncAlreadyRunning`, `ErrTokenExpired`, `ErrInsufficientPermissions`).

**Strict TDD evidence:**

- **RED:** Test that the new `SyncJob` type's JSON tags are correct. Fails.
- **GREEN:** Write the type. Passes.
- **TRIANGULATE:** Add tests for nullable fields (the `*time.Time` and `*string` types).
- **REFACTOR:** Move the `SyncJob` and `SyncJobRepository` to a new `sync.go` file. Re-run tests.

**Estimated LoC:** 100 (domain).

### T-WSY-3a-002 — `infrastructure/postgres/sync_job_repo.go` (RED → GREEN)

Create the pgx adapter for the `SyncJobRepository` port. The adapter implements `Insert` (with the partial unique index returning a `ConflictError` on `SQLSTATE 23505`), `Update` (status, started_at, finished_at, commit_sha_after, error_code, error_message), `GetLatestForWorkspace` (the most recent job, ordered by id DESC), `LockActiveForWorkspace` (a SELECT FOR UPDATE on the partial index). Write tests FIRST using a real Postgres (the existing test infrastructure).

**Strict TDD evidence:**

- **RED:** Test that `Insert` returns a `ConflictError` when a second `pending` row is inserted for the same workspace_id. Fails.
- **GREEN:** Write the adapter with the SQLSTATE translation. Passes.
- **TRIANGULATE:** Add tests for: `GetLatestForWorkspace` returns the most recent job, `LockActiveForWorkspace` returns the active row (and blocks on a concurrent transaction), `Update` with `WHERE id = $1 AND status = 'running'` matches no rows after a second `done` callback (idempotency).
- **REFACTOR:** Extract the column list as a constant. Re-run tests.

**Estimated LoC:** 120 (adapter) + 150 (test).

### T-WSY-3a-003 — `application/sync_service.go` (RED → GREEN)

Create the `SyncService` struct with the use cases: `EnqueueSync(ctx, workspaceID, triggeredBy) (jobID int64, error)`, `GetLatestSyncJob(ctx, workspaceID) (*SyncJob, error)`, `ProcessSyncCallback(ctx, jobID, result) error`. The `EnqueueSync` method calls `repo.Insert` and returns the new job id; on `ConflictError` (single-flight), it returns the existing job id. The `ProcessSyncCallback` method updates the job and the workspace's `last_synced_*` fields. Write tests FIRST using a fake repo.

**Strict TDD evidence:**

- **RED:** Test that `EnqueueSync` returns a new job id on the first call and the existing job id on the second call. Fails.
- **GREEN:** Write the use case. Passes.
- **TRIANGULATE:** Add tests for: `GetLatestSyncJob` returns nil when no job exists, `ProcessSyncCallback` updates the workspace's `last_synced_at` only when the job is `done` (not `failed`).
- **REFACTOR:** Extract the single-flight translation as a helper. Re-run tests.

**Estimated LoC:** 100 (service) + 100 (test).

### T-WSY-3a-004 — Modify `WorkspaceService.Create` to enqueue the first sync

Modify `backend/database_administrator/src/application/workspace_service.go` to call `SyncService.EnqueueSync(workspaceID, "auto_on_create")` on the success path of `Create`. The first-sync enqueue is best-effort: if it fails (e.g. the workspace was created but the sync_job insert failed for any reason), the workspace is still returned with `last_synced_at: null` and a warning is logged. Write tests FIRST.

**Strict TDD evidence:**

- **RED:** Test that `Create` returns a workspace with `initial_sync_job_id` set on success. Fails.
- **GREEN:** Wire the enqueue call. Passes.
- **TRIANGULATE:** Add tests for: the enqueue failure does not roll back the workspace create (best-effort).
- **REFACTOR:** Extract the enqueue call as a `enqueueFirstSync(ctx, workspaceID)` helper. Re-run tests.

**Estimated LoC:** 30 (diff) + 30 (test).

### PR-3a review focus

- The `SyncJob` domain type uses BIGSERIAL (matches the migration).
- The pgx adapter translates `SQLSTATE 23505` to `*domain.ConflictError` (not a custom error).
- The `EnqueueSync` use case is single-flight-safe.
- The `ProcessSyncCallback` is idempotent (re-posting the same callback does not double-update).
- The first-sync enqueue is best-effort (does not roll back the create on failure).
- The diff is under 400 added lines.

### PR-3a acceptance criteria

- [ ] `cd backend/database_administrator && make test` is green.
- [ ] `cd backend/database_administrator && make lint` is clean.
- [ ] The `SyncJob` and `SyncJobRepository` are exposed in the domain.
- [ ] The pgx adapter's `Insert` returns a `ConflictError` on single-flight violation.
- [ ] The `EnqueueSync` use case returns the existing job id on a second call.
- [ ] `WorkspaceService.Create` returns `initial_sync_job_id` on success.
- [ ] The diff is under 400 added lines.

---

## PR-3b — database_administrator: handlers + main.go + github/client + callback (forecast: ~500 LoC, may split into PR-3b-1 and PR-3b-2)

**Branch:** `feat/2026-07-08-workspace-sync-clone-pr3b`
**Target:** `main`
**PR description prefix:** `feat(db_admin): sync endpoints + workspace_syncer client + callback receiver`
**Depends on:** PR-3a, PR-2c

### T-WSY-3b-001 — `infrastructure/workspace_syncer/client.go` — HTTP client to workspace_syncer

Create the HTTP client that the service uses to call `workspace_syncer`. The client has a `CloneAndValidate(ctx, jobID, workspaceID, owner, repo, defaultBranch, oauthToken) error` method. The method POSTs to `http://workspace_syncer:8081/internal/clone-and-validate` with the `INTERNAL_SERVICE_TOKEN` bearer. The client has a 90s timeout (the syncer does the work asynchronously; the client just sends the request). Write tests FIRST using `httptest.NewServer`.

**Strict TDD evidence:**

- **RED:** Test that the client posts the expected body and bearer token. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: the 90s timeout, the network error path, the 4xx/5xx response.
- **REFACTOR:** Extract the JSON encoding as a helper. Re-run tests.

**Estimated LoC:** 60 (client) + 40 (test).

### T-WSY-3b-002 — `interfaces/http/workspace_handler.go` — sync endpoints (RED → GREEN)

Modify `backend/database_administrator/src/interfaces/http/workspace_handler.go` to add `SyncHandler.Post` and `SyncHandler.Get`. The `POST /workspaces/:id/sync` handler validates the workspace exists, calls `SyncService.EnqueueSync(..., "manual")`, calls `workspace_syncerClient.CloneAndValidate(...)` in a goroutine, and returns `202 Accepted` with `{"job_id": <id>, "status": "pending"}`. The `GET /workspaces/:id/sync` handler calls `SyncService.GetLatestSyncJob(...)` and returns the job state. Write tests FIRST using fakes for the service and the client.

**Strict TDD evidence:**

- **RED:** Test that `POST /workspaces/:id/sync` returns 202 with the locked body on a valid request. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: 404 when the workspace does not exist, 409 when a sync is already running (returns the existing job id), 401 when the user is not authenticated.
- **REFACTOR:** Extract the route registration into a helper. Re-run tests.

**Estimated LoC:** 100 (handler) + 100 (test).

### T-WSY-3b-003 — `interfaces/http/internal_callback_handler.go` — callback receiver

Create the internal callback handler that receives the workspace_syncer's callback. The `POST /internal/sync-callback` handler authenticates with the `INTERNAL_SERVICE_TOKEN` (using the same middleware from `workspace_handler.go`), parses the body, calls `SyncService.ProcessSyncCallback(...)`, and returns 204. Write tests FIRST.

**Strict TDD evidence:**

- **RED:** Test that the callback handler updates the `sync_job` and `workspace` on a `done` callback. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: 401 without the bearer token, 404 when the job does not exist, idempotency (re-posting the same `done` callback returns 204 and does not double-update).
- **REFACTOR:** Extract the body parsing as a helper. Re-run tests.

**Estimated LoC:** 60 (handler) + 80 (test).

### T-WSY-3b-004 — `infrastructure/github/client.go` — `GetRepository` extension

Modify `backend/database_administrator/src/infrastructure/github/client.go` to add `GetRepository(ctx, owner, repo) (*Repository, error)`. The method calls `GET https://api.github.com/repos/{owner}/{repo}` and returns a `Repository` struct with `DefaultBranch`, `PrimaryLanguage`, `PushedAt`, `Visibility`, `SizeKB`, `Permissions.Push`. The method uses the existing 5-min in-memory cache (extend the cache key to include `repo` in addition to the user). Write tests FIRST using `httptest.NewServer`.

**Strict TDD evidence:**

- **RED:** Test that `GetRepository` returns the parsed response. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: 401 (token expired), 403 (rate-limited), 404 (repo not found), the cache hit/miss path.
- **REFACTOR:** Extract the response parsing as a helper. Re-run tests.

**Estimated LoC:** 80 (client) + 100 (test).

### T-WSY-3b-005 — `WorkspaceService.Create` — preflight permission check

Modify `backend/database_administrator/src/application/workspace_service.go` to call `githubClient.GetRepository(...)` before enqueuing the first sync. If `permissions.push === false`, the first sync is still enqueued (the `WORKSPACE_PERMISSIONS_INSUFFICIENT` error will be set by the workspace_syncer), but a warning is logged. If the call fails (network error), the first sync is still enqueued (the `CLONE_FAILED` error will be set by the workspace_syncer). This makes the preflight best-effort. Write tests FIRST.

**Strict TDD evidence:**

- **RED:** Test that `Create` still enqueues the first sync even when `GetRepository` fails. Fails.
- **GREEN:** Wire the preflight call. Passes.
- **TRIANGULATE:** Add tests for: the preflight failure logs a warning, the preflight success (push === true) does not change the flow.
- **REFACTOR:** Extract the preflight as a `preflightPermissions(ctx, owner, repo)` helper. Re-run tests.

**Estimated LoC:** 30 (diff) + 30 (test).

### T-WSY-3b-006 — `cmd/server/main.go` — wire the new routes and clients

Modify `backend/database_administrator/src/cmd/server/main.go` to register the new routes (`POST /workspaces/:id/sync`, `GET /workspaces/:id/sync`, `POST /internal/sync-callback`), wire the new dependencies (`SyncService`, `SyncJobRepo`, `workspace_syncerClient`), and add the `INTERNAL_SERVICE_TOKEN` env var to the config. Write a test that asserts the routes are registered.

**Strict TDD evidence:**

- **RED:** Test that the routes are registered. Fails.
- **GREEN:** Wire the routes. Passes.
- **TRIANGULATE:** Add tests for: the env var defaults, the `WORKSPACE_SYNCER_URL` env var (default `http://workspace_syncer:8081`).
- **REFACTOR:** Extract the wiring into a `newServer(cfg Config) *echo.Echo` helper. Re-run tests.

**Estimated LoC:** 40 (diff) + 30 (test).

### T-WSY-3b-007 — Integration test: full create-then-sync flow

Add an integration test in `backend/database_administrator/src/integration/sync_flow_test.go` that exercises the full create-then-sync flow against the local compose stack. The test creates a workspace, polls for the sync to complete, asserts the workspace's `last_synced_at` is non-null, triggers a manual sync, and asserts it completes. The test runs as part of `make test/integration`.

**Strict TDD evidence:**

- **RED:** Test the full flow. Fails.
- **GREEN:** Implement the test using the existing integration test infrastructure.
- **TRIANGULATE:** Add edge cases: the permission failure flow (the workspace is created but `last_synced_at` is null after the failure), the single-flight 409 flow.
- **REFACTOR:** Extract the polling helper as a reusable function. Re-run tests.

**Estimated LoC:** 100 (test).

### PR-3b review focus

- The `INTERNAL_SERVICE_TOKEN` middleware is reused from PR-3a (or added in this PR if PR-3a didn't add it).
- The handler returns 202 immediately; the workspace_syncer call is async.
- The single-flight 409 returns the existing job id.
- The callback handler is idempotent.
- The `GetRepository` extension uses the existing 5-min cache.
- The preflight permission check is best-effort (does not block the create).
- The integration test runs against the compose stack.
- The diff may exceed 400 lines; if so, split into PR-3b-1 (handlers + main.go) and PR-3b-2 (github client + callback).

### PR-3b acceptance criteria

- [ ] `cd backend/database_administrator && make test` is green.
- [ ] `cd backend/database_administrator && make lint` is clean.
- [ ] `cd backend/database_administrator && make test/integration` is green (the full flow runs end-to-end).
- [ ] `POST /workspaces/:id/sync` returns 202 with `{"job_id", "status": "pending"}`.
- [ ] A second `POST /workspaces/:id/sync` while a job is pending returns 409 with the existing job id.
- [ ] The internal callback handler updates the workspace on `done` and `failed`.
- [ ] The diff is under 400 lines if kept as a single PR; if split into PR-3b-1 and PR-3b-2, each is under 400.

---

## PR-4 — Frontend: `WorkspaceSyncCard` + polling + API client (forecast: ~600 LoC, may split into PR-4-1 and PR-4-2)

**Branch:** `feat/2026-07-08-workspace-sync-clone-pr4`
**Target:** `main`
**PR description prefix:** `feat(frontend): workspace sync card + polling`
**Depends on:** PR-3b

### T-WSY-4-001 — `adapters/api-client/` — add `startWorkspaceSync` and `getWorkspaceSyncStatus`

Modify `frontend/src/lib/api.ts` (or wherever the API client lives) to add two new methods. Write tests FIRST using a fetch mock.

**Strict TDD evidence:**

- **RED:** Test that `startWorkspaceSync(42)` POSTs to `/workspaces/42/sync` and returns the parsed JSON. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: the 409 response, the 401 response, the network error.
- **REFACTOR:** Extract the response parsing as a helper. Re-run tests.

**Estimated LoC:** 40 (diff) + 40 (test).

### T-WSY-4-002 — `components/workspace-sync-card/use-sync-status.ts` — polling hook

Create the polling hook. The hook takes a `workspaceId` and a `trigger$` signal, fetches the initial status, and polls every 3s while the status is `pending` or `running`. The hook stops polling when the status is `done` or `failed`. The hook returns the current status, an `isPolling` flag, and a `trigger()` function to manually re-fetch. Write tests FIRST using a `fetch` mock.

**Strict TDD evidence:**

- **RED:** Test that the hook returns the initial status. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: the polling interval, the polling stops on `done`/`failed`, the manual trigger.
- **REFACTOR:** Extract the polling loop as a helper. Re-run tests.

**Estimated LoC:** 60 (hook) + 80 (test).

### T-WSY-4-003 — `components/workspace-sync-card/workspace-sync-card.tsx` — the card component

Create the card component. The card renders:

- The last sync date (relative, e.g. "2 minutes ago") from `last_synced_at`.
- The last commit SHA (short, 7 chars + tooltip with full) from `last_synced_commit_sha`.
- The default branch (with a ref icon) from `default_branch`.
- The primary language.
- The last push date (relative) from the metadata.
- The visibility (Public / Private badge).
- The repository size in KB.
- The Sync button.

The card uses the `use-sync-status` hook to poll for the latest job state. When a job is `pending` or `running`, the button is disabled with a spinner + "Syncing…". When a job is `failed`, the card renders an inline error banner with the `error_message` and a "Retry sync" button. When no job exists, the card shows "Not synced yet." copy.

The card uses the existing `<Button>` system (no new design tokens). Write tests FIRST using vitest.

**Strict TDD evidence:**

- **RED:** Test that the card renders the metadata fields. Fails.
- **GREEN:** Write the minimum code. Passes.
- **TRIANGULATE:** Add tests for: the button disabled state, the error banner, the "Not synced yet." fallback, the polling stops on terminal states.
- **REFACTOR:** Extract the metadata block as a sub-component. Re-run tests.

**Estimated LoC:** 200 (component) + 200 (test).

### T-WSY-4-004 — `routes/workspaces/[id]/index.tsx` — mount the card

Modify the workspace detail page to mount the new card below the existing workspace info. The card receives the workspace and the initial sync state from the route loader. The route's `useTask$` (or `useResource$`) calls the new API client methods to get the initial status. Write tests FIRST.

**Strict TDD evidence:**

- **RED:** Test that the card is mounted on the detail page. Fails.
- **GREEN:** Wire the card. Passes.
- **TRIANGULATE:** Add tests for: the card is not mounted on the list page, the card renders correctly for a soft-deleted workspace (the route is not accessible).
- **REFACTOR:** Extract the route loader as a helper. Re-run tests.

**Estimated LoC:** 40 (diff) + 30 (test).

### T-WSY-4-005 — `routes/workspaces/[id]/route-guard.spec.ts` — update the structural test

Update the existing route-guard structural test (if it exists) to assert that the new card is imported and mounted. If the test does not exist, create it. Write the test FIRST.

**Strict TDD evidence:**

- **RED:** Test that the source file imports `WorkspaceSyncCard`. Fails.
- **GREEN:** Add the import and the JSX. Passes.
- **TRIANGULATE:** Add tests for: the card is imported once, the card is rendered with the correct props.
- **REFACTOR:** N/A (the test is small).

**Estimated LoC:** 20 (test).

### T-WSY-4-006 — `components/README.md` (or equivalent) — update the component inventory

Document the new `WorkspaceSyncCard` component in the project's component inventory. The doc explains the props, the polling behavior, the error states, and the design tokens used.

**Strict TDD evidence:** N/A (documentation).

**Estimated LoC:** 30.

### PR-4 review focus

- The card uses the existing `<Button>` system (no new design tokens).
- The polling hook stops on terminal states (no zombie polls).
- The button is disabled with a spinner + "Syncing…" during `pending`/`running`.
- The error banner is inline (not a top-of-page banner).
- The "Retry sync" button re-triggers the sync (and respects the single-flight 409).
- The route-guard spec asserts the card is mounted.
- The diff may exceed 400 lines; if so, split into PR-4-1 (API client + polling hook + spec) and PR-4-2 (card component + mount + route-guard spec).

### PR-4 acceptance criteria

- [ ] `cd frontend && pnpm run vitest` is green.
- [ ] `cd frontend && pnpm run lint` is clean.
- [ ] `cd frontend && pnpm run fmt.check` is clean.
- [ ] The card renders on the workspace detail page.
- [ ] The card renders all 6 metadata fields.
- [ ] The button is disabled with a spinner during `pending`/`running`.
- [ ] The error banner is inline.
- [ ] The polling stops on terminal states.
- [ ] The diff is under 400 lines if kept as a single PR; if split into PR-4-1 and PR-4-2, each is under 400.

---

## Cross-PR review workload guard

The orchestrator's review workload guard runs after the tasks phase and before the apply phase. Per the session preflight `chained PR strategy: auto-forecast`, the apply phase is allowed to chain PRs without re-asking the user. The forecast above is the apply phase's starting point; the apply phase can refine the split if any PR exceeds 400 lines in practice.

The forecast above has 7 PRs (1 + 2a + 2b + 2c + 3a + 3b + 4). PR-3b and PR-4 are at the upper bound of the budget; the apply phase may split them further. The total forecast is ~2,710 lines across 7 PRs, with each PR under 500 lines (most under 400).

If the apply phase finds that any PR exceeds 400 lines in practice, the orchestrator pauses and asks the user to approve the split. The user has pre-approved chained PRs, so the pause is short.

---

## Cross-PR acceptance criteria

The change is accepted when:

1. All 7 (or 8 if PR-3b and PR-4 are split) PRs are merged to `main`.
2. The full test suite passes at every stage:
   - `cd backend/database_administrator && make test`
   - `cd backend/database_administrator && make test/integration`
   - `cd backend/workspace_syncer && make test`
   - `cd frontend && pnpm run vitest`
   - `cd frontend && pnpm run lint` and `pnpm run fmt.check`
3. The integration test in PR-3b's triangulation phase exercises the full create-then-sync flow against the compose stack.
4. The 2 ADRs are persisted in Engram with the correct topic keys.
5. The canonical `openspec/specs/workspaces/spec.md` is updated (during `sdd-sync`) with R-WS-019 and the wire-shape deltas to R-WS-001 and R-WS-003.
6. The new canonical `openspec/specs/workspace-syncer/spec.md` is written (during `sdd-sync`) from the workspace-syncer spec in this change.
7. The OpenSpec change is archived (during `sdd-archive`) after all PRs are merged.

---

## Next phase

**PAUSE before `sdd-apply`**: per the orchestrator's review workload guard, the orchestrator must ask the user to approve the delivery strategy before launching the apply phase. The user has pre-approved chained PRs, so the pause is short: confirm the 5-7 PR forecast, confirm the apply phase can run them in this session (or split across sessions), and confirm any open questions.
