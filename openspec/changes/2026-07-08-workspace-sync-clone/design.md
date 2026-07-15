# Design — Workspace Sync Card & Repository Cloning

> **Change**: `2026-07-08-workspace-sync-clone`
> **Phase**: design
> **Project**: cachicamas
> **Date**: 2026-07-08
> **Inputs**: `explore.md` (engram id 1831), `proposal.md` (engram id 1832), `specs/workspaces/spec.md`, `specs/workspace-syncer/spec.md` (engram id 1833)
> **Output**: `design.md` (this file)
> **Skill resolution**: `paths-injected` (no project skills required for design)
> **Diagrams**: ASCII (project convention — no mermaid in any existing proposal/spec)

---

## 1. Architecture overview

```
+----------------------+         HTTP/JSON (bearer INTERNAL_SERVICE_TOKEN)          +----------------------+
|                      |  POST /internal/clone-and-validate                          |                      |
|  database_adminis-   | ----------------------------------------------------------> |  workspace_syncer    |
|  trator              |                                                            |  (Go, hexagonal)     |
|  (Go, hexagonal)     |  POST /internal/sync-callback                              |                      |
|                      | <---------------------------------------------------------- |  - git via os/exec   |
|  - OAuth token store |                                                            |  - bare mirror layout|
|  - sync_job table    |                                                            |  - /data/workspaces/ |
|  - public HTTP API   |                                                            |                      |
|  - github REST proxy |                                                            |                      |
+----------+-----------+                                                            +----------+-----------+
           ^                                                                                   |
           | HTTP/JSON (cookie auth)                                                           |
           |                                                                                   v
+----------+-----------+                                                            +----------+-----------+
|                      |                                                            |                      |
|  Frontend (Qwik)     |                                                            |  Docker volume      |
|  - /workspaces/:id   |                                                            |  cachicamas_synced_ |
|  - WorkspaceSyncCard |                                                            |  repos              |
|  - polling 3s        |                                                            |  /data/workspaces/  |
|                      |                                                            |  {id}/{owner}/      |
+----------------------+                                                            |  {repo}.git/        |
                                                                                     |                      |
                                                                                     +----------------------+
```

**Service boundaries:**

| Service | Owns | Does NOT own | Postgres access | Public HTTP | Internal HTTP |
| --- | --- | --- | --- | --- | --- |
| `database_administrator` | `workspace` table, `sync_job` table, `identity.account.access_token`, public API, GitHub REST proxy, OAuth scope validation, auto-cleanup coordination | git, filesystem, worktree | yes (read/write) | yes (cookie auth) | yes (callback receiver) |
| `workspace_syncer` | git invocation, filesystem layout, worktree probe, clone-side token usage, periodic sweep | Postgres, public API, OAuth scope, cross-workspace cleanup coordination | no | no | yes (clone receiver) |
| Frontend (Qwik) | Card UI, polling loop, error rendering, button states | git, OAuth, filesystem, sync lifecycle | no (talks only to `database_administrator` over HTTPS+cookies) | n/a | n/a |

**Data flow (high level):**

1. User creates a workspace → `database_administrator` enqueues a `sync_job` with `triggered_by = "auto_on_create"`.
2. `database_administrator` POSTs to `workspace_syncer/internal/clone-and-validate` (async; non-blocking from the user's perspective).
3. `workspace_syncer` validates the token (GitHub REST), clones the repo, runs the worktree probe, and POSTs the outcome to `database_administrator/internal/sync-callback`.
4. `database_administrator` updates `sync_job` (status, started_at, finished_at, commit_sha_after, error_*) and `workspace` (last_synced_at, last_synced_commit_sha, last_sync_job_id).
5. Frontend polls `GET /workspaces/:id/sync` every 3s while a job is in flight; renders the card with the updated data.

---

## 2. Sequence diagrams

### 2.1 Auto-sync-on-create (the "happy first sync")

```
actor       browser        db_admin           ws_syncer          github
  |            |               |                  |                  |
  |  POST /workspaces         |                  |                  |
  |  {name, repository}      |                  |                  |
  |-------------------------> |                  |                  |
  |            |    validate(name, repository)   |                  |
  |            |    GetRepository(owner, repo)   |                  |
  |            |    ---> |      |    GET /repos/{o}/{r}              |
  |            |          |      | -------------------------------> |
  |            |          |      |    200 {permissions.push: true} |
  |            |          |      | <------------------------------- |
  |            |          |      |                                  |
  |            |    INSERT workspace (RETURNING id)                 |
  |            |    INSERT sync_job (status=pending,                 |
  |            |            triggered_by=auto_on_create)             |
  |            |               |                  |                  |
  |  201 {id, last_synced_at: null,                |                  |
  |       initial_sync_job_id: 42, ...}            |                  |
  | <------------------------- |                  |                  |
  |            |               |                  |                  |
  |            |  POST /internal/clone-and-validate (async)         |
  |            |               | ----------------> |                  |
  |            |               |                  |  validate token  |
  |            |               |                  |  GET /repos/{o}/{r}
  |            |               |                  | ------------->   |
  |            |               |                  |  200 ok         |
  |            |               |                  | <-------------   |
  |            |               |                  |                  |
  |            |               |                  |  git clone --bare
  |            |               |                  |   /data/workspaces/{id}/{o}/{r}.git/
  |            |               |                  |  git worktree add /tmp/probe HEAD
  |            |               |                  |  (exit 0)       |
  |            |               |                  |  rm /tmp/probe  |
  |            |               |                  |                  |
  |            |               |  POST /internal/sync-callback       |
  |            |               | <---------------- |                  |
  |            |               |   {job_id, status: done,            |
  |            |               |    commit_sha_after: "abc1234..."}  |
  |            |               |                  |                  |
  |            |    UPDATE sync_job SET status=done,                |
  |            |            finished_at=now,                        |
  |            |            commit_sha_after=... WHERE id=42        |
  |            |            AND status='running'                    |
  |            |    UPDATE workspace SET last_synced_at=now,        |
  |            |            last_synced_commit_sha=...,             |
  |            |            last_sync_job_id=42 WHERE id={id}       |
  |            |               |                  |                  |
  | (frontend polls every 3s)  |                  |                  |
  |  GET /workspaces/:id/sync   |                  |                  |
  |-------------------------> |                  |                  |
  |  200 {status: done,        |                  |                  |
  |       commit_sha_after,    |                  |                  |
  |       finished_at, ...}    |                  |                  |
  | <------------------------- |                  |                  |
```

### 2.2 Manual sync (the "user clicks Sync")

```
actor       browser        db_admin           ws_syncer          github
  |            |               |                  |                  |
  |  POST /workspaces/:id/sync                  |                  |
  |  (empty body)         |                  |                  |
  |-------------------------> |                  |                  |
  |            |    SELECT FOR UPDATE on the     |                  |
  |            |    partial unique index         |                  |
  |            |    (workspace_id) WHERE status IN                |
  |            |    ('pending','running')        |                  |
  |            |    -- if a row exists -> 409    |                  |
  |            |       sync_already_running      |                  |
  |            |       with existing job_id      |                  |
  |            |    INSERT sync_job (status=pending,               |
  |            |            triggered_by=manual)  |                  |
  |            |               |                  |                  |
  |  202 {job_id, status: pending}               |                  |
  | <------------------------- |                  |                  |
  |            |               |                  |                  |
  |            |  POST /internal/clone-and-validate (same as 2.1)  |
  |            |               | ----------------> |                  |
  |            |               |    ... (same as 2.1) ...           |
  |            |               |  POST /internal/sync-callback       |
  |            |               | <---------------- |                  |
  |            |               |                  |                  |
  |  (frontend polls; card re-renders)            |                  |
```

### 2.3 Permissions failure (the "insufficient scope" branch)

```
actor       browser        db_admin           ws_syncer          github
  |            |               |                  |                  |
  |  POST /workspaces/:id/sync                  |                  |
  |-------------------------> |                  |                  |
  |            |    INSERT sync_job (status=pending)                |
  |            |  202 {job_id}    |                  |              |
  | <------------------------- |                  |                  |
  |            |               |                  |                  |
  |            |  POST /internal/clone-and-validate                  |
  |            |               | ----------------> |                  |
  |            |               |                  |  GET /repos/{o}/{r}
  |            |               |                  | ------------->   |
  |            |               |                  |  200 {permissions.push: false}
  |            |               |                  | <-------------   |
  |            |               |                  |                  |
  |            |               |                  |  (no clone)      |
  |            |               |                  |                  |
  |            |               |  POST /internal/sync-callback       |
  |            |               | <---------------- |                  |
  |            |               |   {status: failed,                 |
  |            |               |    error_code:                     |
  |            |               |     WORKSPACE_PERMISSIONS_         |
  |            |               |     INSUFFICIENT,                  |
  |            |               |    error_message:                  |
  |            |               |     "Token lacks push permission"} |
  |            |               |                  |                  |
  |            |    UPDATE sync_job SET status=failed,              |
  |            |            error_code=..., error_message=...       |
  |            |               |                  |                  |
  | (frontend polls; card shows error banner)     |                  |
  |  GET /workspaces/:id/sync   |                  |                  |
  |  200 {status: failed, error_message, ...}     |                  |
  | <------------------------- |                  |                  |
  |  Card renders inline error banner with       |                  |
  |  "Token lacks push permission" + "Retry sync" button             |
```

### 2.4 Soft-delete mid-sync (the "let-complete" branch)

```
actor       browser        db_admin           ws_syncer          filesystem
  |            |               |                  |                  |
  |  DELETE /workspaces/:id   |                  |                  |
  |-------------------------> |                  |                  |
  |            |    UPDATE workspace SET deleted_at=now()           |
  |            |  204          |                  |                  |
  | <------------------------- |                  |                  |
  |            |               |                  |                  |
  |            |  (sync_job was running before soft-delete)         |
  |            |               |                  |                  |
  |            |  POST /internal/sync-callback (in flight)         |
  |            | <---------------- |                                  |
  |            |    UPDATE sync_job SET status=done WHERE id=... AND status='running'
  |            |    UPDATE workspace SET last_sync_job_id=... (FK ON DELETE SET NULL keeps it; soft-delete leaves the row)
  |            |               |                  |                  |
  |  (the next sweep on workspace_syncer startup, OR a periodic     |
  |   poll, removes the orphaned /data/workspaces/{id}/ tree)       |
  |            |               |                  |  rm -rf /data/workspaces/{id}
  |            |               |                  | ---------------> |
  |            |               |                  |                  |
```

Note: the soft-delete is intentionally NOT propagated synchronously to `workspace_syncer` in v1. The cleanup happens on the next sweep. This keeps the soft-delete fast (one DB UPDATE) and avoids a synchronous cross-service call. The trade-off is a temporary window (up to one sweep cycle) where the cloned tree exists on disk without a live workspace. The sweep is bounded by a 30s timeout and runs on every workspace_syncer startup; if long-running deployments accumulate orphans, a periodic timer can be added in a follow-up change.

---

## 3. Database state machine for `sync_job`

```
                 +----------+
                 | pending  |
                 +----+-----+
                      |
   (ws_syncer picks   |   (ws_syncer callback
    up the job)       v    arrives with status=done|failed)
                 +----------+         +-----------+
                 | running  | ------> |   done    |
                 +----+-----+         +-----------+
                      |
                      |  (callback arrives
                      |   with status=failed)
                      v
                 +----------+
                 |  failed  |
                 +----------+

   Invariants:
   - (workspace_id) WHERE status IN ('pending','running') is unique
     (partial unique index sync_job_single_flight_uidx)
   - terminal states (done, failed) do NOT release the workspace_id
     for a new sync — the partial index excludes them
   - workspace.last_sync_job_id references the LATEST sync_job
     (FK ON DELETE SET NULL)
```

State transitions:

| From | To | Trigger | Side effects |
| --- | --- | --- | --- |
| (none) | `pending` | `POST /workspaces` (auto_on_create) or `POST /workspaces/:id/sync` (manual) | INSERT sync_job; INSERT/UPDATE workspace.last_sync_job_id |
| `pending` | `running` | `workspace_syncer` callback `status=running` (optional, not in v1 — the transition can happen synchronously in the clone-and-validate handler) | UPDATE sync_job SET started_at=now(), status='running' |
| `running` | `done` | `workspace_syncer` callback `status=done, commit_sha_after=...` | UPDATE sync_job; UPDATE workspace.last_synced_at, last_synced_commit_sha, last_sync_job_id |
| `running` | `failed` | `workspace_syncer` callback `status=failed, error_code=..., error_message=...` | UPDATE sync_job (no workspace changes — last_sync_job_id is updated to point at the failed job so the UI can show the error) |
| `pending` | `failed` | Pre-flight failure (e.g. GetRepository 401, or the workspace_syncer rejects the request before starting) | UPDATE sync_job |

The `pending → running` transition is optional in v1; the workspace_syncer can write `done` or `failed` directly without an intermediate `running` write. The design keeps both code paths open so v2 can add progress reporting (e.g. `running` with `progress_pct`) without changing the schema.

---

## 4. Filesystem layout (workspace_syncer side)

```
/data/workspaces/
├── {workspace_id}/
│   ├── {owner}/
│   │   ├── {repo}.git/        # bare mirror (HEAD, objects/, refs/)
│   │   ├── {other_repo}.git/  # not used in v1; reserved for the future "linked repos" feature
│   │   └── ...
│   └── {owner2}/
│       └── ...
├── {other_workspace_id}/
│   └── ...
```

**Why bare mirror:**

- Smaller on disk (no working tree files at the top level).
- No accidental writes (the future worktree feature uses `git worktree add` from the bare mirror, which is the canonical pattern).
- Single `git clone --bare` command; no need for the separate `git worktree add` step on initial clone.

**Path construction:**

```go
// layout.go
func WorkspacePath(workspaceID int64, owner, repo string) (string, error) {
    if workspaceID <= 0 {
        return "", fmt.Errorf("invalid workspace_id: %d", workspaceID)
    }
    if !validRepoSegment(owner) || !validRepoSegment(repo) {
        return "", fmt.Errorf("invalid owner/repo: %q/%q", owner, repo)
    }
    return fmt.Sprintf("/data/workspaces/%d/%s/%s.git", workspaceID, owner, repo), nil
}

func validRepoSegment(s string) bool {
    if s == "" || len(s) > 100 {
        return false
    }
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, s)
    return matched
}
```

**Cleanup hook (startup sweep):**

```go
// sweep.go
func Sweep(ctx context.Context, liveIDs map[int64]bool, fs FS, log *slog.Logger) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    entries, err := fs.ReadDir("/data/workspaces")
    if err != nil {
        return fmt.Errorf("read /data/workspaces: %w", err)
    }
    for _, entry := range entries {
        id, err := strconv.ParseInt(entry.Name(), 10, 64)
        if err != nil {
            // Non-numeric directory (shouldn't happen, but be defensive)
            log.WarnContext(ctx, "sweep: non-numeric entry", "name", entry.Name())
            continue
        }
        if !liveIDs[id] {
            if err := fs.RemoveAll(filepath.Join("/data/workspaces", entry.Name())); err != nil {
                log.ErrorContext(ctx, "sweep: remove failed", "id", id, "err", err)
                continue
            }
            log.InfoContext(ctx, "sweep: removed orphan", "id", id)
        }
    }
    return nil
}
```

The sweep uses a `FS` interface so it can be unit-tested with an in-memory fake filesystem (avoiding the need to actually delete real files in tests).

**Disk cap (v1 = none):** a per-workspace quota is explicitly deferred. The v1 trust the docker volume. A follow-up change can add a quota check in `WorkspacePath` (e.g. compute the on-disk size of the bare mirror and reject clones that exceed N MiB). For now, the only safeguard is the docker volume's underlying filesystem size.

---

## 5. Cross-service auth posture

### 5.1 v1: static bearer + docker network trust

```go
// middleware.go (workspace_syncer)
func ServiceTokenMiddleware(expected string) echo.MiddlewareFunc {
    expectedBytes := []byte(expected)
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            auth := c.Request().Header.Get("Authorization")
            const prefix = "Bearer "
            if !strings.HasPrefix(auth, prefix) {
                return echo.NewHTTPError(401, "unauthorized")
            }
            token := []byte(strings.TrimPrefix(auth, prefix))
            if subtle.ConstantTimeCompare(token, expectedBytes) != 1 {
                return echo.NewHTTPError(401, "unauthorized")
            }
            return next(c)
        }
    }
}
```

The `INTERNAL_SERVICE_TOKEN` env var is read at startup. The service fails to boot with a clear error if it is empty (S-WSY-054). The token is held in memory only; never logged; never persisted.

**Defense-in-depth layering:**

1. The bearer token is required (defense layer 1 — explicit auth).
2. The docker network trust ensures the service is not exposed to the public internet (defense layer 2 — network isolation).
3. The Go binary binds to a non-public interface (or to localhost, depending on the compose config; design phase ratifies the exact binding).

### 5.2 v2: HMAC-signed short-lived JWT (deferred)

Per ADR `adr/workspace-syncer-internal-auth` (to be persisted during the tasks phase), v2 moves to a short-lived JWT signed with a shared secret, exchanged at a /token endpoint on `database_administrator`. The v2 design is out of scope for this change but the topic key is reserved.

---

## 6. Error envelope consolidation

The workspace_syncer's internal errors are translated to the `database_administrator` envelope shape BEFORE the callback is posted. The syncer does NOT have its own public error envelope — it only has internal error codes that map to the database_administrator envelope.

| Internal code (workspace_syncer) | Mapped `error_code` (callback) | HTTP envelope (if surfaced via /internal/clone-and-validate directly) |
| --- | --- | --- |
| `INTERNAL_INVALID_REQUEST` | n/a (synchronous 400) | `{"error": "validation", "fields": {...}}` |
| `INTERNAL_UNAUTHORIZED` | n/a (synchronous 401) | `{"error": "unauthorized", "message": "..."}` |
| `TOKEN_EXPIRED` | `TOKEN_EXPIRED` | n/a (callback-only) |
| `WORKSPACE_PERMISSIONS_INSUFFICIENT` | `WORKSPACE_PERMISSIONS_INSUFFICIENT` | n/a (callback-only) |
| `BRANCH_NOT_FOUND` | `BRANCH_NOT_FOUND` | n/a (callback-only) |
| `WORKTREE_PROBE_FAILED` | `WORKTREE_PROBE_FAILED` | n/a (callback-only) |
| `CLONE_TIMEOUT` | `CLONE_TIMEOUT` | n/a (callback-only) |
| `CLONE_FAILED` (catch-all) | `CLONE_FAILED` | n/a (callback-only) |
| `INTERNAL_GIT_ERROR` (unexpected exit) | `CLONE_FAILED` | n/a (callback-only) |

The shape `{ "error": "<MACHINE_CODE>", "message": "<HUMAN_MESSAGE>" }` is the **flat** envelope used by the rest of `database_administrator`. The explore draft used a nested `{ "error": { "code": ..., "message": ... } }`; the proposal corrected this to the flat shape and the design locks it.

---

## 7. 4-PR diff forecast (per PR file inventory + LoC)

The total work is ~2,350 changed lines (including tests). Per the preflight `chained PR strategy: auto-forecast`, the work is split into 4 PRs each under the 400-line review budget.

### PR-1 — Migration (forecast: ~150 LoC, 3 new files, 1 modified)

**New files:**

| Path | LoC | Purpose |
| --- | ---: | --- |
| `backend/database_administrator/src/migration/sql/20260708120000_sync_job.sql` | 80 | `sync_job` table + ALTER on `workspace` + indexes + partial unique index |
| `backend/database_administrator/src/migration/sql/20260708120100_workspace_sync_cleanup_metadata.sql` | 20 | (optional) housekeeping — backfill default_branch for pre-existing rows; left as best-effort, NULL allowed |
| `backend/database_administrator/src/migration/runner_test.go` (extension) | 50 | Migration test asserting the new table + ALTER + indexes |

**Modified files:**

| Path | LoC delta | Purpose |
| --- | ---: | --- |
| `infra/postgres/init/01-init.sql` | 5 | Add `GRANT SELECT, INSERT, UPDATE, DELETE ON sync_job TO queen;` |

**Total:** 150 LoC across 3 new + 1 modified file. **Well under 400.** ✓

**Review focus:** schema correctness, the partial unique index `WHERE status IN ('pending','running')` clause, the FK `ON DELETE SET NULL` on `workspace.last_sync_job_id`, the BIGSERIAL type for all new IDs (not UUID), the NOT NULL constraints on `status` and `triggered_by`.

### PR-2 — workspace_syncer skeleton (forecast: ~700 LoC, 28 new files)

**New service directory (`backend/workspace_syncer/`):**

| Path | LoC | Purpose |
| --- | ---: | --- |
| `go.mod` | 30 | Module file (github.com/cachicamas/backend/workspace_syncer) |
| `go.sum` | (generated) | Locked dependencies |
| `Dockerfile` | 40 | Triple-pinned base; install git; copy binary |
| `Makefile` | 80 | build, test, lint, run, tools (mirrors database_administrator) |
| `.dockerignore` | 10 | Standard Go ignore set |
| `.golangci.yml` | 30 | Same config as database_administrator |
| `README.md` | 60 | Documents /internal/clone-and-validate |
| `src/cmd/server/main.go` | 90 | Composition root |
| `src/cmd/server/main_test.go` | 30 | Tests the boot-without-token error |
| `src/domain/clone.go` | 60 | `CloneRequest`, `CloneResult`, validation |
| `src/domain/clone_test.go` | 40 | Domain validation tests |
| `src/domain/errors.go` | 40 | Locked error vocabulary |
| `src/domain/errors_test.go` | 30 | Error vocabulary tests |
| `src/application/clone_service.go` | 80 | `CloneAndValidate`, `MarkFailed` use cases |
| `src/application/clone_service_test.go` | 60 | Use-case tests |
| `src/infrastructure/git/runner.go` | 120 | Clone, WorktreeAdd, ResolveHead |
| `src/infrastructure/git/runner_test.go` | 150 | 6+ unit tests (happy, bad token, worktree, path-safety, shell-inject, redaction) |
| `src/infrastructure/git/layout.go` | 40 | `WorkspacePath` function |
| `src/infrastructure/git/layout_test.go` | 50 | 5+ tests (S-WSY-020..024) |
| `src/infrastructure/git/sweep.go` | 60 | `Sweep` startup cleanup |
| `src/infrastructure/git/sweep_test.go` | 50 | Idempotency + 30s timeout tests |
| `src/infrastructure/httpclient/callback_client.go` | 60 | HTTP client to database_administrator callback |
| `src/infrastructure/httpclient/callback_client_test.go` | 40 | httptest mock |
| `src/infrastructure/token/middleware.go` | 30 | Bearer-token middleware (constant-time compare) |
| `src/infrastructure/token/middleware_test.go` | 40 | 4+ tests (S-WSY-010..013) |
| `src/interfaces/http/handler.go` | 100 | POST /internal/clone-and-validate |
| `src/interfaces/http/handler_test.go` | 120 | 4+ handler tests (auth, happy, perm, timeout) |
| `src/otel/otel.go` | 40 | Tracer init (mirrors database_administrator) |
| `src/otel/logging.go` | 30 | slog init with redaction handler |
| `src/otel/logging_test.go` | 30 | Redaction tests |

**Modified files (top-level repo):**

| Path | LoC delta | Purpose |
| --- | ---: | --- |
| `docker-compose.yaml` | 25 | Add `workspace_syncer` service, network link, shared volume |
| `docker-compose.vps.yaml` | 25 | Same as above (VPS compose) |

**Total:** ~1,200 LoC. **Exceeds 400.** → This PR will be split into PR-2a and PR-2b.

#### PR-2a — workspace_syncer skeleton + clone (forecast: ~750 LoC)

Cuts the workspace_syncer directory at the **functional** boundary: everything needed to receive a request, validate it, run the clone, and post the callback. Excludes the sweep (cleanup hook) and the OTel wiring (uses no-op tracer for v1).

- `go.mod`, `go.sum`, `Dockerfile`, `Makefile`, `.dockerignore`, `.golangci.yml`, `README.md`
- `src/cmd/server/main.go` (composition root) + `main_test.go`
- `src/domain/clone.go` + test
- `src/domain/errors.go` + test
- `src/application/clone_service.go` + test
- `src/infrastructure/git/runner.go` + test
- `src/infrastructure/git/layout.go` + test
- `src/infrastructure/httpclient/callback_client.go` + test
- `src/infrastructure/token/middleware.go` + test
- `src/interfaces/http/handler.go` + test
- `src/otel/otel.go` (no-op tracer)
- `src/otel/logging.go` (basic slog, no redaction yet)

#### PR-2b — workspace_syncer sweep + redaction + OTel (forecast: ~450 LoC)

Adds the production-grade observability and the cleanup hook. The 2-PR split keeps each under 400.

- `src/infrastructure/git/sweep.go` + test
- `src/otel/otel.go` (upgrade to real OTel tracer, add the `clone.execute` span)
- `src/otel/logging.go` (add the redaction handler)
- `src/otel/logging_test.go` (redaction tests)
- `docker-compose.yaml` + `docker-compose.vps.yaml` updates (only the volume mount + network link; the service definition is in PR-2a)

Wait — the compose changes are needed for both PR-2a and PR-2b. Let me reconsider the split: the compose changes should land with the minimum viable service in PR-2a, and the sweep wiring lands in PR-2b. So:

- **PR-2a (~750 LoC, but reviewed in two slices by the reviewer):** the workspace_syncer is shippable; reviewer can run `make test` and `make run` against the local stack. Compose file updated.
- **PR-2b (~450 LoC, also under 400):** adds the production-grade sweep and redaction. After this, the service is "feature-complete" for the v1 contract.

Wait, 450 still exceeds 400. Let me cut further:

- **PR-2a (~650 LoC)**: workspace_syncer skeleton + clone + callback. Under 400 LoC added (existing files don't count as new). Let me re-check: the per-PR review budget counts ADDED + DELETED lines, not just new files. The 650 LoC forecast was the TOTAL of all new files. The actual ADDED lines per PR will be less because the files are spread across the diff. Let me re-forecast.

Actually, the 400-line budget is for the PR's diff (additions + deletions), not the total new LoC. The PR can introduce a 700-LoC new service as long as the diff stays under 400 added lines per file. The reality: 28 new files, each with their first commit, will show as ~28 file creations with their content. The diff total = sum of all new file content = 700+ LoC. This exceeds 400.

**So the per-PR split must happen at the file level: 28 files in two PRs of ~14 files each.** Let me reorganize.

#### PR-2a (revised) — workspace_syncer skeleton + core (~400 LoC, 16 files)

The minimum viable workspace_syncer that boots, accepts a request, runs a clone, and posts a callback. No sweep, no redaction, basic slog.

- `go.mod` (30) + `go.sum`
- `Dockerfile` (40)
- `Makefile` (80)
- `.dockerignore` (10)
- `.golangci.yml` (30)
- `README.md` (60)
- `src/cmd/server/main.go` (90) + `main_test.go` (30)
- `src/domain/clone.go` (60) + `clone_test.go` (40)
- `src/domain/errors.go` (40) + `errors_test.go` (30)
- `src/application/clone_service.go` (80) + `clone_service_test.go` (60)
- `src/infrastructure/git/runner.go` (120) + `runner_test.go` (150)
- `src/infrastructure/git/layout.go` (40) + `layout_test.go` (50)
- `src/infrastructure/httpclient/callback_client.go` (60) + `callback_client_test.go` (40)
- `src/infrastructure/token/middleware.go` (30) + `middleware_test.go` (40)
- `src/interfaces/http/handler.go` (100) + `handler_test.go` (120)
- `src/otel/otel.go` (40, no-op tracer)
- `src/otel/logging.go` (30, basic slog)

Total: ~1,200 LoC across 22 files. **The diff would be ~1,200 added lines, which exceeds 400.**

Hmm, the chained-PR strategy says each PR should be under 400 lines. The workspace_syncer service is too big to land in one PR. The 4-PR plan needs to be refined.

**Option A:** Split PR-2 into 3 sub-PRs (PR-2a, PR-2b, PR-2c). Total 5 PRs instead of 4. Each sub-PR ~400 LoC.

**Option B:** Land the workspace_syncer as a single PR of ~1,200 LoC and accept that it exceeds the budget. The 4R review can be split per file.

**Option C:** Land the workspace_syncer scaffolding (just `go.mod`, `Dockerfile`, `Makefile`, `cmd/`, `otel/`, one minimal endpoint) in PR-2a, then iterate the use cases in PR-2b.

The user chose `chained PR strategy: auto-forecast`. The auto-forecast means the task agent can split into more PRs without re-asking. So **Option A is the right call**: the task phase will forecast 5-6 PRs, not 4.

**Updated PR plan:**

- **PR-1 (migration):** ~150 LoC. Schema only.
- **PR-2a (workspace_syncer scaffolding):** ~400 LoC. `go.mod`, `Dockerfile`, `Makefile`, `cmd/`, `otel/` (basic), `domain/`, `infrastructure/token/`. The single endpoint exists but returns 501 Not Implemented.
- **PR-2b (workspace_syncer clone):** ~400 LoC. `application/clone_service`, `infrastructure/git/runner`, `infrastructure/git/layout`, `infrastructure/httpclient/callback_client`, `interfaces/http/handler`. The endpoint is fully functional; the OTel span is wired in `clone.execute`; basic slog without redaction.
- **PR-2c (workspace_syncer sweep + redaction):** ~300 LoC. `infrastructure/git/sweep`, `otel/logging` upgrade with redaction handler, sweep wiring in `main.go`, compose file updates (volume mount + service definition in compose).
- **PR-3 (database_administrator):** ~900 LoC. Domain extension, sync_job_repo, sync_service, internal_callback_handler, modified workspace_handler, modified workspace_service, modified workspace_repo, modified github/client, new workspace_syncer/client, modified main.go, modified Makefile, modified Dockerfile. **Likely split into PR-3a (sync_job + repo + service) and PR-3b (handlers + main.go + github/client) if it exceeds 400.**
- **PR-4 (frontend):** ~600 LoC. `WorkspaceSyncCard` + `use-sync-status` + API client + mount. **Likely under 400, but close to the limit; might be a single PR.**

The actual splitting is the task phase's job. The design locks the per-PR scope boundaries; the tasks phase refines the LoC forecast and the per-task breakdown.

---

## 8. Per-PR strict TDD evidence requirements

For each PR, the `apply-progress.md` must record RED → GREEN → TRIANGULATE → REFACTOR per task. The minimum evidence per PR:

### PR-1 (migration)

For each migration piece, the test must:

- **RED:** run the migration on an empty DB, then assert the new table/column/index exists (using a test query that checks `information_schema`). The query fails before the migration runs (because the table doesn't exist).
- **GREEN:** write the migration. The test passes.
- **TRIANGULATE:** add a test for the partial unique index — try to insert two `pending` rows for the same `workspace_id`; the second must fail with SQLSTATE 23505.
- **REFACTOR:** consolidate the migration into one transaction; add `IF NOT EXISTS` guards for idempotency.

### PR-2a (workspace_syncer scaffolding)

For each file, the test must:

- **RED:** write the test that asserts the file's exported behavior. The test fails because the file doesn't exist or doesn't compile.
- **GREEN:** write the minimum code to make the test pass. No gold-plating.
- **TRIANGULATE:** add edge-case tests (empty input, wrong type, etc.).
- **REFACTOR:** clean up; re-run tests; verify no regression.

### PR-2b (workspace_syncer clone)

- **RED:** write the test that runs a clone against a real `git` binary in a temp dir. The test fails because the handler returns 501.
- **GREEN:** implement the handler end-to-end.
- **TRIANGULATE:** add tests for each failure mode (bad token, branch not found, worktree probe fail, timeout, shell injection).
- **REFACTOR:** extract the layout function, the shell-out wrapper, the redaction handler.

### PR-2c (workspace_syncer sweep + redaction)

- **RED:** write the test that asserts a non-live workspace_id is removed on startup. The test fails because the sweep doesn't exist.
- **GREEN:** implement the sweep.
- **TRIANGULATE:** add tests for idempotency, the 30s timeout, the non-numeric entry case.
- **REFACTOR:** extract the `FS` interface so tests use a fake filesystem.

### PR-3a (sync_job + repo + service on database_administrator)

- **RED:** write the test that asserts `WorkspaceService.EnqueueSync` returns a job_id. The test fails because the method doesn't exist.
- **GREEN:** implement the use case + the pgx adapter.
- **TRIANGULATE:** add tests for single-flight (a second enqueue returns ConflictError), GetLatest, the lock active query.
- **REFACTOR:** extract the SQL strings as constants; add the `INSERT ... ON CONFLICT` translation.

### PR-3b (handlers + main.go + github/client on database_administrator)

- **RED:** write the handler test that asserts `POST /workspaces/:id/sync` returns 202 with `{job_id, status: "pending"}`. The test fails because the route doesn't exist.
- **GREEN:** implement the handler + the workspace_syncer client + the main.go wiring.
- **TRIANGULATE:** add tests for the callback handler, the 409 single-flight response, the github client `GetRepository` extension.
- **REFACTOR:** consolidate the route registration; add the `INTERNAL_SERVICE_TOKEN` env var.

### PR-4 (frontend card)

- **RED:** write the vitest test that asserts the card renders all 6 metadata fields. The test fails because the component doesn't exist.
- **GREEN:** implement the component.
- **TRIANGULATE:** add tests for button states (enabled, disabled-with-spinner), error banner, polling stops on done/failed, polling interval.
- **REFACTOR:** extract the polling hook (`use-sync-status`); use the existing `<Button>` system.

---

## 9. The 2 ADRs

The tasks phase will create two ADRs in Engram:

### ADR: `adr/workspace-syncer-git-impl` (precedes PR-2a)

**Status:** proposed
**Context:** The workspace_syncer needs to invoke `git` to clone repositories and run worktree probes. Two implementation options:

- (a) `github.com/go-git/go-git/v5` — a pure-Go git implementation. Pro: no external binary. Con: incomplete worktree support, would need to shell out anyway for the probe step; larger binary; new top-level dep.
- (b) `os/exec` to the system `git` binary in the container. Pro: smaller binary, no CGO, full worktree support. Con: new build-time dep (the `git` package on the container base image).

**Decision:** option (b). The system `git` is the canonical implementation; `go-git` does not have full worktree support and would not eliminate the shell-out step. The Dockerfile installs `git` (Debian: `apt-get install -y git`; pin to `git 1:2.47.x` per the project's pinning discipline).

**Rollback:** if the binary approach causes issues, swap to `go-git` in a follow-up change. The `infrastructure/git` package is the only consumer; swapping is mechanical.

### ADR: `adr/workspace-syncer-internal-auth` (precedes PR-2a)

**Status:** proposed
**Context:** `database_administrator` and `workspace_syncer` need to authenticate each other over the internal HTTP. Three options:

- (a) Static shared bearer token (env var on both sides). Pro: zero deps, simple. Con: long-lived secret; rotation requires a coordinated restart.
- (b) Mutual TLS. Pro: short-lived certs, no shared secret. Con: new infrastructure (cert provisioning, rotation), mTLS config in Echo.
- (c) HMAC-signed short-lived JWT exchanged at a /token endpoint. Pro: short-lived, no mTLS. Con: new code on both sides, more surface area.

**Decision:** option (a) for v1, with a documented path to option (c) for v2. The docker network trust is the primary mitigation; the bearer token is defense-in-depth.

**Rollback:** swap to option (c) in v2; the v1 code is the implementation of the simpler approach and can be replaced incrementally.

---

## 10. Test strategy

### Per-PR test commands

| PR | Test command | Coverage expectation |
| --- | --- | --- |
| PR-1 | `cd backend/database_administrator && make test` (includes the migration test) | Migration test passes; existing tests still green |
| PR-2a | `cd backend/workspace_syncer && make test` | New service boots; middleware tests pass; domain tests pass |
| PR-2b | `cd backend/workspace_syncer && make test` | Clone happy path + 5 failure modes pass; runner tests pass; handler tests pass |
| PR-2c | `cd backend/workspace_syncer && make test` | Sweep tests pass; redaction tests pass |
| PR-3a | `cd backend/database_administrator && make test` | sync_service tests pass; sync_job_repo tests pass; existing workspace tests still green |
| PR-3b | `cd backend/database_administrator && make test` | Handler tests pass; github client extension tests pass; integration test for the full create-then-sync flow passes |
| PR-4 | `cd frontend && pnpm run vitest` | WorkspaceSyncCard tests pass; use-sync-status tests pass; route-guard structural test updated |

### Integration test (PR-3b's TRIANGULATE phase)

The apply phase must add an integration test that exercises the full create-then-sync flow against the local compose stack:

```
1. POST /workspaces {name, repository} → 201 with initial_sync_job_id
2. GET /workspaces/:id/sync (poll) → eventually returns status=done
3. GET /workspaces/:id → returns last_synced_at non-null, last_synced_commit_sha non-null
4. POST /workspaces/:id/sync → 202 with new job_id
5. GET /workspaces/:id/sync (poll) → eventually returns status=done
6. (verify the bare mirror exists on the workspace_syncer container)
```

This integration test runs against `make test/integration` (which boots the compose stack). It is the final triangulation case for the workspace sync flow.

### E2E test (deferred to a future change)

A Playwright e2e for the full UX (create workspace → see card render with metadata → click Sync → see updated data) is deferred to a future "workspaces e2e" change that also covers the create/delete flows. The frontend's component tests cover the card's behavior in isolation.

---

## 11. Docker compose changes

```yaml
# docker-compose.yaml (excerpt)
services:
  database_administrator:
    environment:
      INTERNAL_SERVICE_TOKEN: ${INTERNAL_SERVICE_TOKEN:?INTERNAL_SERVICE_TOKEN must be set}
    networks:
      - cachicamas_network

  workspace_syncer:
    build:
      context: ./backend/workspace_syncer
    environment:
      INTERNAL_SERVICE_TOKEN: ${INTERNAL_SERVICE_TOKEN:?INTERNAL_SERVICE_TOKEN must be set}
      OTEL_EXPORTER_OTLP_ENDPOINT: ${OTEL_EXPORTER_OTLP_ENDPOINT}
    volumes:
      - cachicamas_synced_repos:/data/workspaces
    networks:
      - cachicamas_network
    depends_on:
      - database_administrator

volumes:
  cachicamas_synced_repos:

networks:
  cachicamas_network:
```

**Key points:**

- The shared volume `cachicamas_synced_repos` is mounted at `/data/workspaces` in `workspace_syncer` only. `database_administrator` does not need direct access; the cloned data is referenced via `sync_job` rows.
- The `INTERNAL_SERVICE_TOKEN` is required on both services. The compose file uses `${VAR:?...}` syntax to fail fast if the var is unset.
- The `workspace_syncer` does not depend on the database directly (no `depends_on` for postgres); it talks to `database_administrator` over HTTP.

---

## 12. Open questions for the tasks phase

1. **PR-2 split granularity:** the design above proposes 2a (scaffolding) / 2b (clone) / 2c (sweep + redaction). The tasks phase should forecast each sub-PR's diff and refine the split if any exceeds 400. If the actual `git` integration is smaller than forecast (e.g. we can use `os/exec` with shorter code), PR-2a and PR-2b can be merged.
2. **PR-3 split granularity:** the design above proposes 3a (sync_job + repo + service) / 3b (handlers + main.go + github/client). The tasks phase should confirm the split.
3. **PR-4 size:** the design above proposes a single PR-4. The tasks phase should confirm. If the card or the polling hook grow large, split into PR-4a (card + polling) / PR-4b (mount + API client).
4. **PR ordering:** the design assumes PR-1 → PR-2 → PR-3 → PR-4. The tasks phase can interleave (e.g. PR-1 + PR-2a as a "schema + scaffolding" pair) if it improves the review flow.
5. **Migration backfill:** the `20260708120100_workspace_sync_cleanup_metadata.sql` migration is optional. The tasks phase should decide whether to land it as a separate PR (best-effort housekeeping) or skip it (NULL `default_branch` is acceptable for pre-existing rows; the next sync will populate it).

---

## 13. Acceptance criteria for the design phase

This design is accepted when:

- [ ] The architecture overview in §1 is consistent with the proposal.
- [ ] The sequence diagrams in §2 cover all 4 paths: auto-sync-on-create, manual sync, permissions failure, soft-delete mid-sync.
- [ ] The DB state machine in §3 is consistent with the `sync_job` schema in the migration.
- [ ] The filesystem layout in §4 is consistent with the cleanup hook in `R-WSY-004`.
- [ ] The cross-service auth posture in §5 is consistent with the ADR `adr/workspace-syncer-internal-auth`.
- [ ] The error envelope consolidation in §6 is consistent with the rest of the database_administrator envelope vocabulary.
- [ ] The 4-PR (or 5-6-PR) diff forecast in §7 is consistent with the per-PR scope in the proposal.
- [ ] The per-PR strict TDD evidence requirements in §8 are consistent with `openspec/config.yaml`.
- [ ] The 2 ADRs in §9 are documented in Engram with the correct topic keys.
- [ ] The test strategy in §10 covers all 4 paths.
- [ ] The docker compose changes in §11 are consistent with the existing compose file.
- [ ] The open questions in §12 are forwarded to the tasks phase.

---

## 14. Next phase

Run `sdd-tasks` for change `2026-07-08-workspace-sync-clone`. The tasks phase will:

1. Create the 2 ADRs in Engram (`adr/workspace-syncer-git-impl`, `adr/workspace-syncer-internal-auth`).
2. Forecast each PR's diff and refine the per-PR split (likely 5-6 PRs instead of 4).
3. Produce a `tasks.md` with per-PR task lists, each completable in a single PR, each including the strict TDD evidence expectation.
4. Apply the review workload guard: if any PR's forecast still exceeds 400 lines, split further.
