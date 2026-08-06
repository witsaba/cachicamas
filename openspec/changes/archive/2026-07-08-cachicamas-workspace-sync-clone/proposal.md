# Proposal: Workspace Sync Card & Repository Cloning

> **Change**: `2026-07-08-workspace-sync-clone`
> **Status**: proposed
> **Created**: 2026-07-08
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Scope**: `@cachicamas/backend` (database_administrator + new workspace_syncer) + `@cachicamas/frontend` + compose stack + OpenSpec specs
> **Persistence**: engram `sdd/2026-07-08-workspace-sync-clone/proposal` + filesystem mirror
> **Predecessor artifact**: `openspec/changes/2026-07-08-workspace-sync-clone/explore.md` (id 1831) + `openspec/changes/archive/2026-07-06-workspaces/proposal.md` (the deferred item this change implements)
> **Stack**: Go 1.26.3 + Echo v5 + Postgres 18 + OTel; Qwik 1.20 + Tailwind 4 frontend; same Go module pattern as `database_administrator` for the new service.

---

## Intent

Cachicamas landed the Workspaces feature on 2026-07-08 (PR #44). A workspace is a logical container scoped to the install's single organization that maps 1:1 to a GitHub repository and shows up in `/workspaces` and `/workspaces/:id`. Today, that row is a metadata stub: it knows the repo's `owner/name` and `github_id`, but the server has never actually fetched the repo. The OAuth `access_token` is persisted (PR #44), the `repo` scope is requested at sign-in, the `IsRepoAccessible` check works — and then nothing happens. The 2026-07-06 proposal explicitly deferred "Actual repo cloning (uses persisted access_token)" to a follow-up change because the first slice's scope was workspaces + connection, not server-side fetches.

This change implements that deferred item. It adds a new `workspace_syncer` Go service that consumes the persisted `access_token` over HTTP, runs `git clone` + permission checks + a worktree probe, and persists the outcome. The `database_administrator` service grows a `sync_job` table, two new public endpoints, and a `last_synced_at` / `last_synced_commit_sha` / `default_branch` triple on the workspace row. The workspace detail page (`/workspaces/:id`) gains a new "Sync" card that surfaces the latest sync date, the commit SHA captured at the moment of sync, a Sync button that pulls the default branch, and useful repository metadata. The first sync fires automatically when a workspace is created so the card never shows "never synced" for a fresh workspace.

Why now. The workspaces surface is half-functional without clone: the user creates a workspace against a repo, and the system has not verified the OAuth token grants `repo` scope with `permissions.push === true`, has not downloaded the code, has not run any git-level validation. Every future feature that depends on the cloned tree (PR authoring, worktree per branch, webhook ingestion, code-aware previews) is blocked on this slice. The cost of deferring grows with every new feature added on top.

Why split into a new service. `database_administrator` is a thin Go service that owns the database_administrator schema, the auth middleware, the GitHub REST client, and the HTTP API. Adding `git clone` to it would conflate the role of a "control plane" (CRUD, validation, REST surface) with a "data plane" (filesystem-bound long-running work). The new `workspace_syncer` owns the data plane: it has zero Postgres access, mounts the same docker volume, talks only to git and to `database_administrator` over HTTP. This is a one-time split that future heavy features (clone-for-PR, worktree-per-branch, partial clone, etc.) can build on without rewriting the existing service.

---

## Locked decisions (user-confirmed 2026-07-08)

These are NOT proposals. They were decided in a 4-question round before `sdd-explore` started. The proposal documents them here for traceability; the next phases do NOT re-litigate them.

1. **Service split**: a new Go service `workspace_syncer` is created as a separate container. `database_administrator` delegates sync work over HTTP. The new service owns git, the local filesystem layout, and the actual `git clone` / `git fetch` / `git worktree add` invocations. `database_administrator` keeps owning the OAuth token store, the `sync_job` table, and the public HTTP API.
2. **Transport & sync model**: HTTP/JSON, lowest footprint (no gRPC, no WebSockets). `POST /workspaces/:id/sync` returns HTTP 202 with a `job_id`. `GET /workspaces/:id/sync` returns the job status (`pending` / `running` / `done` / `failed`). The choice of HTTP/JSON was made on footprint grounds: same Echo stack, no new deps, request/poll pattern fits the use case without streaming or push. A future "push notifications" feature can land as a separate change without breaking the current contract.
3. **First sync timing**: when `POST /workspaces` creates a workspace, the create use case auto-enqueues a sync job in the same request and returns 201 with `last_synced_at: null` plus the `initial_sync_job_id`. The card never shows "never synced" for a workspace that was just created.
4. **Card placement**: a new card on the workspace detail page (`/workspaces/:id`) only. The list cards on `/workspaces` are unchanged in this slice. (A future "status on list" change is explicitly out of scope.)

**Type correction carried forward**: the explore draft used UUID for `sync_job.id` and `last_sync_job_id`. The existing `workspace.id` is `BIGSERIAL` (int64) per `migration/sql/20260706120002_workspaces.sql`. All new IDs in this change MUST be `BIGSERIAL` (int64) to match. **No UUIDs are introduced.**

---

## Scope

### In scope (delivered as 4 chained PRs)

- **New `backend/workspace_syncer` Go service** with the same hexagonal layout as `database_administrator` (`cmd/`, `src/{domain,application,infrastructure,interfaces,otel}/`, `Dockerfile`, `Makefile`, `go.mod`). Echo v5, OTel (`otelslog` + `sdk/trace`), pinned Go 1.26.3. Registered in `docker-compose.yaml` and `docker-compose.vps.yaml` with a shared volume for cloned repos.
- **New `sync_job` table** on the existing `database_administrator` schema (DDL: `backend/database_administrator/src/migration/sql/20260708120000_sync_job.sql`). Plus ALTER on `workspace` adding `last_synced_at TIMESTAMPTZ NULL`, `last_synced_commit_sha TEXT NULL`, `default_branch TEXT NULL`, `last_sync_job_id BIGINT NULL REFERENCES sync_job(id) ON DELETE SET NULL`. All new IDs are `BIGSERIAL`. Single-flight enforced by a partial unique index `(workspace_id) WHERE status IN ('pending','running')`.
- **Two new public endpoints on `database_administrator`**:
  - `POST /workspaces/:id/sync` → 202 with `{job_id, status: "pending"}`.
  - `GET /workspaces/:id/sync` → 200 with `{job_id, status, started_at, finished_at, commit_sha_after, error_message, error_code, attempts}`.
  - `POST /workspaces` is modified to enqueue the first sync on success and return `{initial_sync_job_id}` in the response.
- **One new internal endpoint on `workspace_syncer`**:
  - `POST /internal/clone-and-validate` (database_administrator → workspace_syncer) — receives a `job_id`, `workspace_id`, `owner`, `repo`, `default_branch`, and `oauth_token`; runs the clone; runs the worktree probe; reports back via a callback. Authenticated with a static service-to-service token (env var `INTERNAL_SERVICE_TOKEN`) — defense-in-depth on top of docker network trust.
- **Extension to the GitHub client** in `backend/database_administrator/src/infrastructure/github/client.go` to add `GetRepository(ctx, owner, repo)` that returns `permissions.push`, `default_branch`, `primary_language`, `pushed_at`, `visibility`, `size_kb`. Used by `WorkspaceService` before enqueuing the sync job to fail fast on insufficient permissions.
- **Frontend `WorkspaceSyncCard` component** mounted on `/workspaces/:id`. Card surfaces:
  - Last sync date (relative, e.g. "2 minutes ago").
  - Last commit SHA captured at the moment of sync (short form, 7 chars + tooltip with full).
  - Default branch (with ref icon).
  - Primary language.
  - Last push date (relative).
  - Visibility (Public / Private badge).
  - Repository size in KB.
  - Sync button.
- **API client methods** in `frontend/src/lib/api.ts`: `startWorkspaceSync(id)` and `getWorkspaceSyncStatus(id)`.
- **Auto-cleanup of cloned data on workspace soft-delete**: the `workspace_syncer` listens for a workspace-soft-deleted notification (the simplest mechanism is a poll on `GET /workspaces/:id` plus a periodic sweeper, but the design phase should ratify the cleanest mechanism — webhook from database_administrator vs. periodic sweeper).
- **Single-flight on sync jobs per workspace**: the partial unique index plus a 409 `SYNC_ALREADY_RUNNING` response on the second `POST /workspaces/:id/sync` call while a job is pending or running.
- **OpenSpec specs updated**:
  - `openspec/specs/workspaces/spec.md` — modified with a new "R-WS-019 — Workspace sync" requirement group.
  - `openspec/specs/workspace-syncer/spec.md` — new (the first spec for the new service).

### Out of scope (deferred)

- Webhook ingestion from GitHub for incremental sync.
- PR auto-creation. The change validates that the token has permission to create PRs (`permissions.push === true`) but does not create any.
- Worktree merge/rebase / actual `git worktree add` execution (the precondition is met; the future feature is its own change).
- Encryption-at-rest for the persisted OAuth token.
- OAuth token refresh cron / background expiry handling.
- Multi-tenant keying of the new `workspace_syncer` service (single shared identity in v1).
- Restore of soft-deleted workspaces (existing deferred item).
- Hard delete of soft-deleted workspaces (existing deferred item).
- Badge / status indicator on the workspace list cards (deferred to a future "status on list" change).
- Sync from a specific ref — v1 is `default_branch` only. The internal API contract documents the extension point so v2 can add a `ref` param without breaking the v1 contract.
- Per-workspace disk usage cap (the first slice trusts the docker volume; a follow-up change can add a quota if production shows pressure).

---

## Decisions taken in this proposal (ratifiable or correctable)

These are decisions the proposal phase would normally ask in a question round. In auto-mode they're documented as explicit assumptions so the user can ratify or correct in review. Five open questions from the explore, each with a recommended default:

1. **Sync button during a running job — disabled with a spinner + "Syncing…" text.** No progress percentage in v1 (we don't have meaningful granularity — `git clone` doesn't expose per-step progress over its exit). The button is re-enabled when `GET /workspaces/:id/sync` returns `done` or `failed`.
2. **Metadata v1 field set — `default_branch`, `primary_language`, `pushed_at`, `visibility`, `size_kb`.** No `stars` (not a sync concern, drifts noisily; would force a separate "refresh metadata" affordance). No `description` (not in the user's brief; can be added later). No `topics` or `license` (same rationale).
3. **In-flight sync when a workspace is soft-deleted — let it complete, then drop the result.** The FK `ON DELETE SET NULL` on `workspace.last_sync_job_id` already handles the "row should not reference a non-existent job" case. The syncer's cleanup hook removes the cloned data on soft-delete even mid-sync — the clone is orphaned and the next sync of the same repo reuses the path or garbage-collects it.
4. **Ref picker — default branch only in v1.** The internal API documents a future `ref` field; the public `POST /workspaces/:id/sync` request body does NOT carry a `ref` yet. v2 adds it.
5. **Error-state UI placement — inline banner inside the card** with the `error_message` text and a "Retry sync" button. No top-of-page banner (would feel disproportionate for a single workspace); no replace-the-metadata block (would lose the last-known-good data).

---

## Affected areas

| Area | Kind | File | Change |
| --- | --- | --- | --- |
| Migration | new | `backend/database_administrator/src/migration/sql/20260708120000_sync_job.sql` | `sync_job` table + ALTER on `workspace` (4 columns) + partial unique index for single-flight + secondary indexes on `(workspace_id)` and `(status)` |
| Migration | new | `backend/database_administrator/src/migration/sql/20260708120100_workspace_sync_cleanup_metadata.sql` | (Optional) housekeeping if PR-1 needs to backfill default_branch for pre-existing workspaces — left as best-effort, NULL allowed for pre-existing rows |
| Domain | modified | `backend/database_administrator/src/domain/workspace.go` | Extend `Workspace` with `LastSyncedAt *time.Time`, `LastSyncedCommitSha *string`, `DefaultBranch *string`; add the new `SyncJob` aggregate |
| Domain | modified | `backend/database_administrator/src/domain/errors.go` | Add `ErrSyncAlreadyRunning`, `ErrTokenExpired`, `ErrInsufficientPermissions`, `ErrCloneFailed` |
| Application | modified | `backend/database_administrator/src/application/workspace_service.go` | Add `EnqueueSync`, `GetLatestSyncJob`, `MarkWorkspaceSynced` use cases. Modify `Create` to call `EnqueueSync(..., "auto_on_create")` on success |
| Application | new | `backend/database_administrator/src/application/sync_service.go` | Use cases: `StartSync`, `GetSyncStatus`, `ProcessSyncCallback` (called by the workspace_syncer's HTTP callback), `MarkFailed` |
| Infrastructure | modified | `backend/database_administrator/src/infrastructure/postgres/workspace_repo.go` | Add 4 columns to the column list, the INSERT column list, and the SELECT column list. Scan helper updates |
| Infrastructure | new | `backend/database_administrator/src/infrastructure/postgres/sync_job_repo.go` | pgx adapter for `SyncJobRepository` — Insert, Update, GetLatestForWorkspace, LockActiveForWorkspace |
| Infrastructure | modified | `backend/database_administrator/src/infrastructure/github/client.go` | Add `GetRepository(ctx, owner, repo) (*Repository, error)`; new `Repository` type with the v1 metadata fields |
| Infrastructure | modified | `backend/database_administrator/src/infrastructure/github/cache.go` | Reuse the 5-min in-memory cache for `GetRepository` keyed by `owner/name` |
| Infrastructure | new | `backend/database_administrator/src/infrastructure/workspace_syncer/client.go` | HTTP client for the workspace_syncer internal endpoint. Bearer-token auth. `internalClientTimeout = 90s` |
| HTTP | modified | `backend/database_administrator/src/interfaces/http/workspace_handler.go` | Add `SyncHandler.Post` (POST /workspaces/:id/sync) and `SyncHandler.Get` (GET /workspaces/:id/sync); modify `CreateHandler` to return `initial_sync_job_id` |
| HTTP | modified | `backend/database_administrator/src/interfaces/http/internal_callback_handler.go` (new) | Receives the workspace_syncer's callback reporting `done`/`failed`; updates `sync_job` and `workspace.last_synced_*` |
| HTTP | modified | `backend/database_administrator/src/cmd/server/main.go` | Register new routes, wire `SyncService`, `SyncJobRepo`, `workspace_syncerClient` |
| HTTP | modified | `backend/database_administrator/Dockerfile`, `Makefile` | No new build deps expected for the database_administrator slice; only the `INTERNAL_SERVICE_TOKEN` env var is added to the service config |
| Frontend component | new | `frontend/src/components/workspace-sync-card/` | New component: `index.tsx`, `workspace-sync-card.spec.tsx`, `use-sync-status.ts` (polling hook) |
| Frontend route | modified | `frontend/src/routes/workspaces/[id]/index.tsx` | Mount the new card below the existing workspace info |
| Frontend route | modified | `frontend/src/routes/workspaces/[id]/route-guard.spec.ts` | (if exists) Update structural test to expect the new card |
| Frontend API client | modified | `frontend/src/lib/api.ts` | Add `startWorkspaceSync(id)`, `getWorkspaceSyncStatus(id)` |
| New service | new | `backend/workspace_syncer/` (entire directory) | Full new Go service. See "Apply delivery" §PR-2 for the file inventory |
| Docker | modified | `docker-compose.yaml`, `docker-compose.vps.yaml` | Add `workspace_syncer` service; network link to `database_administrator`; shared volume `cachicamas_synced_repos` mounted at `/data/workspaces` in workspace_syncer |
| Infra | modified | `infra/postgres/init/01-init.sql` | No new table here (sync_job lives on the database_administrator schema, not the queen role's DDL). However, add `GRANT CRUD` on `sync_job` to the `queen` role for the API service to access |
| Specs | modified | `openspec/specs/workspaces/spec.md` | Add R-WS-019 (Workspace sync) with Given/When/Then scenarios |
| Specs | new | `openspec/specs/workspace-syncer/spec.md` | First spec for the new service |
| Docs | modified | `backend/database_administrator/README.md` | Document the 2 new public endpoints + the callback endpoint + the `INTERNAL_SERVICE_TOKEN` env var |
| Docs | modified | `frontend/src/components/README.md` (or equivalent) | Add the new card to the component inventory |

---

## Approach (top-level shape)

**Directory layout for the new service:**

```
backend/workspace_syncer/
├── go.mod                        # github.com/cachicamas/backend/workspace_syncer
├── go.sum
├── Dockerfile                    # triple-pinned base; copies the binary built by Makefile
├── Makefile                      # build, test, lint, run, tools (mirrors database_administrator/Makefile)
├── .dockerignore
├── .golangci.yml                 # same config as database_administrator
├── README.md                     # documents /internal/clone-and-validate
└── src/
    ├── cmd/server/main.go        # composition root
    ├── domain/                   # CloneRequest, CloneResult, validation, errors
    │   ├── clone.go
    │   └── clone_test.go
    ├── application/              # use cases
    │   ├── clone_service.go      # CloneAndValidate, CleanupWorkspace, MarkFailed
    │   └── clone_service_test.go
    ├── infrastructure/
    │   ├── git/                  # shell-out to git binary, fs layout
    │   │   ├── runner.go         # Clone, Fetch, WorktreeAdd, ResolveHead
    │   │   ├── runner_test.go
    │   │   └── layout.go         # /data/workspaces/{workspace_id}/{owner}/{repo}.git
    │   ├── httpclient/           # HTTP client to call back to database_administrator
    │   │   └── callback_client.go
    │   └── token/                # bearer-token middleware
    │       └── middleware.go
    ├── interfaces/http/
    │   ├── handler.go            # POST /internal/clone-and-validate
    │   ├── handler_test.go
    │   └── routes.go
    └── otel/                     # tracing + logging, mirrors database_administrator/src/otel/
        ├── otel.go
        └── logging.go
```

**Filesystem layout (workspace_syncer side):**

```
/data/workspaces/{workspace_id}/{owner}/{repo}.git/
    # bare mirror, OR
    # working tree with `.git/` and the files at the head
```

The design phase ratifies whether each workspace is a bare mirror (faster for read-only operations, smaller on disk) or a full working tree (simpler, supports future `git worktree add` directly). Initial recommendation: bare mirror (smaller on disk, no accidental writes, future worktree feature can use `git worktree add` from the bare mirror).

**Cross-service auth:**

- Static service-to-service token, identical to the model currently used by `database_administrator` for outbound calls (env var `INTERNAL_SERVICE_TOKEN`), is presented via `Authorization: Bearer <token>` and validated by Echo middleware on `workspace_syncer`.
- For v1, docker network trust is the primary mitigation; the shared token is defense-in-depth and enables the same code path to run in CI without docker.
- The ADR for this lives at Engram topic `adr/workspace-syncer-internal-auth`.

**Cross-service contract (final):**

```http
POST /internal/clone-and-validate
Authorization: Bearer <INTERNAL_SERVICE_TOKEN>
Content-Type: application/json

{
  "job_id":         42,
  "workspace_id":   7,
  "owner":          "octocat",
  "repo":           "hello-world",
  "default_branch": "main",
  "oauth_token":    "gho_..."
}
```

Response on acceptance (the syncer does the work synchronously and posts the result back to database_administrator's callback endpoint; this 200 just means "we accepted and started"):

```http
HTTP 202 Accepted
{ "job_id": 42, "status": "running" }
```

Callback from workspace_syncer to database_administrator (separate URL, posted by the syncer on completion):

```http
POST /internal/sync-callback
Authorization: Bearer <INTERNAL_SERVICE_TOKEN>
Content-Type: application/json

{
  "job_id":           42,
  "workspace_id":     7,
  "status":           "done" | "failed",
  "commit_sha_after": "abc1234...",        // only if status=done
  "error_code":       "WORKSPACE_PERMISSIONS_INSUFFICIENT",  // only if status=failed
  "error_message":    "Token lacks push permission"          // only if status=failed
}
```

Error envelope (consistent with the rest of database_administrator):

```json
{ "error": "WORKSPACE_PERMISSIONS_INSUFFICIENT", "message": "Token lacks push permission" }
```

**Type note**: the IDs in the JSON are `int64` (BIGSERIAL), serialized as JSON numbers. The Go struct on both sides is `int64`. NOT UUID.

---

## Apply delivery (4-PR chain)

The total work is too large for a single PR (forecast ~2,400 changed lines including tests). Per the preflight `chained PR strategy: auto-forecast`, the change is split into 4 chained PRs, each under the 400-line review budget.

### PR-1 — Migration only

- Files: `backend/database_administrator/src/migration/sql/20260708120000_sync_job.sql` (new), `infra/postgres/init/01-init.sql` (1 line of GRANT).
- Estimated LoC: ~150.
- Tests: `backend/database_administrator/src/migration/runner_test.go` extension to assert the new table + ALTER + indexes + partial unique index.
- Review focus: schema correctness, single-flight index partial clause, FK ON DELETE SET NULL on `workspace.last_sync_job_id`.

### PR-2 — `workspace_syncer` skeleton

- Files: the entire `backend/workspace_syncer/` tree (see "Approach" §"Directory layout"); compose file updates.
- Estimated LoC: ~700 (including tests).
- Tests: domain + application + infrastructure + handler unit tests; one end-to-end handler test using a temp dir + a real `git` binary.
- Review focus: hexagonal boundaries (no Postgres import), token redaction in logs, shell-out safety (no shell injection — args are passed as a slice), filesystem path safety (workspace_id is BIGSERIAL, sanitized before joining paths).
- Dependencies: PR-1 must be merged first (so the workspace_id is known valid by the time the syncer runs against the real DB).

### PR-3 — `database_administrator` extension

- Files: domain extension, new `sync_job_repo.go`, `sync_service.go`, new `internal_callback_handler.go`, modified `workspace_handler.go`, modified `workspace_service.go`, modified `workspace_repo.go`, modified `github/client.go`, new `workspace_syncer/client.go`, modified `main.go`, modified `Makefile` and `Dockerfile` (no new deps).
- Estimated LoC: ~900.
- Tests: pgx adapter tests, service tests, handler tests, internal callback handler tests, GitHub client extension test.
- Review focus: single-flight enforcement, error envelope consistency, service-to-service token middleware, callback idempotency (re-posting the same callback does not double-update `workspace.last_synced_at`).
- Dependencies: PR-1 + PR-2 must be merged first (the syncer must exist; the table must exist).

### PR-4 — Frontend card

- Files: new `frontend/src/components/workspace-sync-card/`, modified `frontend/src/routes/workspaces/[id]/index.tsx`, modified `frontend/src/lib/api.ts`, modified `frontend/src/components/README.md` (or equivalent).
- Estimated LoC: ~600.
- Tests: component spec, polling hook spec, route-guard spec.
- Review focus: polling interval (locked at 3s in v1, stops on `done`/`failed`), error banner inside the card, button disabled state, no auto-retry on failed sync (user must click "Retry").
- Dependencies: PR-3 must be merged first (the API must exist for the card to call).

**Forecast vs budget**: each PR is under 400 changed lines. The chain as a whole (~2,350 LoC) exceeds the 400-line single-PR budget; chained PRs are the explicitly approved strategy per the session preflight.

---

## Risks

| # | Risk | Likelihood | Mitigation |
| --- | --- | --- | --- |
| 1 | Clone timeout on large repos (60s+ on 1 GiB repos) | medium | Async job model + 90s timeout on the workspace_syncer HTTP client; UI polls every 3s; `error_code: "CLONE_TIMEOUT"` surfaced with a clear "Retry" affordance |
| 2 | Disk pressure across many cloned workspaces | medium | Trust the docker volume in v1; document the per-workspace path; the design phase ratifies a per-workspace quota + sweeper (deferred to a follow-up if production shows pressure) |
| 3 | Token expiry mid-clone | low | `error_code: "TOKEN_EXPIRED"`; UI surfaces "Reconnect GitHub" banner using the existing `ReconnectGitHub` component; the next sync after re-auth picks up the new token |
| 4 | Race between two sync jobs for the same workspace (double-click) | medium | Partial unique index `sync_job_single_flight_uidx`; handler returns 409 `SYNC_ALREADY_RUNNING` with the existing job_id so the UI can poll it instead of starting a new one |
| 5 | Cross-service auth posture | medium | Static shared service token + docker network trust for v1; ADR `adr/workspace-syncer-internal-auth` documents the v2 move to HMAC-signed short-lived JWT |
| 6 | Shell injection via `os/exec` if args aren't sanitized | medium | Always pass args as a `[]string` to `exec.Command`; never use `sh -c` with interpolated input; workspace_id is BIGSERIAL and sanitized to ASCII digits before joining paths |
| 7 | Workspace ID collision or path traversal in the workspace_syncer filesystem | low | The workspace_id is BIGSERIAL (digits only) and validated before joining; the layout function rejects any id with non-digit characters |
| 8 | Token leaked via logs | low | The syncer's `runner.go` redacts `oauth_token` from any log line via a custom slog handler; reviewed by `review-risk` in the apply phase |
| 9 | Soft-delete mid-sync leaves orphan data | low | The workspace_syncer's cleanup hook runs on every sync start (idempotent: removes `/data/workspaces/{id}/...` if no live `sync_job` references it) and on every soft-delete event (polled via a short interval OR pushed via a database_administrator → syncer notification) |
| 10 | Chained PRs drift in the middle (one PR merged in an unexpected order) | low | The apply phase enforces the chain: each PR's branch is `feat/2026-07-08-workspace-sync-clone-prN` and merges to main in order. The `sdd-verify` for PR-N checks the prior PRs are on main |
| 11 | Frontend polling overloads database_administrator with status requests | low | UI polls every 3s while a job is `pending` or `running`; stops on `done`/`failed`. The status endpoint is a cheap indexed query; p95 target < 30ms |
| 12 | New top-level Go dep for `workspace_syncer` (the `git` binary) requires an ADR | low (already known) | ADR at `adr/workspace-syncer-git-impl` recommends `os/exec` + system `git` over `go-git`; no new top-level Go module dep is added |
| 13 | Strict TDD posture skipped in any PR | low | Each `apply-progress.md` records RED → GREEN → TRIANGULATE → REFACTOR per task; `sdd-verify` checks for the evidence |

---

## Rollback plan (3 levels)

1. **Revert any single PR** cleanly (each is forward-only additive; the migrations have a `Down` that drops the new table / columns; the workspace_syncer is a separate container that can be stopped without affecting database_administrator).
2. **Revert the chain**: drop `workspace_syncer` from `docker-compose.yaml`, drop the `sync_job` migration, remove the card from the frontend. Server state remains valid (no other code references the new tables or endpoints).
3. **Operational fallback**: if production shows the card UX wrong, the API can be feature-flagged off by removing the new routes from `database_administrator/src/cmd/server/main.go`; the card then 404s on poll and degrades gracefully to a "Sync unavailable" inline message. No data is lost.

No migrations require a down-migration in the happy path. If a `sync_job` row exists and the migration is rolled back, the DB-level rollback drops the `sync_job` table — destroying data. Mitigation: never roll back a migration that has data; forward-fix instead. Documented in PR-1's description.

---

## Dependencies

- **No new top-level Go deps in `database_administrator`**. Reuses `labstack/echo/v5`, `jackc/pgx/v5`, `go.opentelemetry.io/*` (all already vendored).
- **New `workspace_syncer` service** needs:
  - `github.com/labstack/echo/v5` (already vendored)
  - `go.opentelemetry.io/*` (already vendored)
  - **`git` binary in the container** (no Go dep — uses `os/exec`). Requires the Dockerfile to install `git` (Debian: `apt-get install -y git`; pin to `git 1:2.47.x` per the project's pinning discipline).
  - **An ADR at `adr/workspace-syncer-git-impl`** documents the `os/exec` + system `git` choice over `github.com/go-git/go-git/v5`. Rationale: `go-git` does not have full worktree support; shelling out is unavoidable anyway for the worktree probe; the system `git` binary is the smallest-footprint path. Persisted to Engram with topic `adr/workspace-syncer-git-impl` and referenced from this proposal.
- **No new npm deps in the frontend** (Qwik 1.20 + Tailwind 4 + the existing `<Button>` system handle the card).
- **No changes to docker compose networking model** beyond adding the new service and the shared volume.
- **No changes to the database user/role**; the existing `queen` role already has CRUD on tables in the `database_administrator` schema.

---

## OpenSpec spec impact

- `openspec/specs/workspaces/spec.md` — **modified**. Add a new "R-WS-019 — Workspace sync" requirement group with Given/When/Then scenarios for: auto-sync on create, manual sync trigger, sync job status poll, single-flight 409, permissions-failure UX, soft-delete cleans up cloned data, sync with insufficient token. Update the `POST /workspaces` and `GET /workspaces/:id` response shapes to include the new fields (`last_synced_at`, `last_synced_commit_sha`, `default_branch`, `initial_sync_job_id`, `last_sync_job_id`).
- `openspec/specs/workspace-syncer/spec.md` — **new**. First spec for the new service. Document the internal `POST /internal/clone-and-validate` endpoint contract, the worktree probe requirement, the disk layout, the soft-delete cleanup behavior, the bearer-token middleware, and the callback shape. Use Given/When/Then + RFC 2119 keywords per `openspec/config.yaml`. All IDs are BIGSERIAL.

---

## Success criteria

PR-1 is shippable when:

- [ ] `sync_job` table exists with the locked columns, FKs, the partial unique index `(workspace_id) WHERE status IN ('pending','running')`, and the secondary indexes.
- [ ] `workspace` has the 4 new columns with the right types and the `last_sync_job_id` FK.
- [ ] `go test ./...` is green; `migration_test.go` extension passes.

PR-2 is shippable when:

- [ ] `workspace_syncer` boots with `make run` and exposes `POST /internal/clone-and-validate`.
- [ ] The endpoint accepts a real request from a `curl` against a local Postgres-emptied docker compose stack; clones a real GitHub repo (use `mocks-github-oauth` for the token); the worktree probe exits 0.
- [ ] The token-redaction slog handler unit test passes.
- [ ] The path-safety unit test rejects non-digit workspace_ids.
- [ ] `go test ./...` is green; `golangci-lint` is clean.

PR-3 is shippable when:

- [ ] `POST /workspaces` returns `{initial_sync_job_id, last_synced_at: null}` on success.
- [ ] `POST /workspaces/:id/sync` returns 202 with `{job_id, status: "pending"}`.
- [ ] A second `POST /workspaces/:id/sync` while a job is pending returns 409 `SYNC_ALREADY_RUNNING` with the existing `job_id`.
- [ ] `GET /workspaces/:id/sync` returns the current job state.
- [ ] The internal callback handler updates `sync_job.status` and `workspace.last_synced_*` on `done` / `failed`.
- [ ] `go test ./...` is green; `golangci-lint` is clean.

PR-4 is shippable when:

- [ ] `/workspaces/:id` shows the new card with all 6 metadata fields and the Sync button.
- [ ] Clicking Sync on a fresh workspace triggers the auto-job; the card shows "Syncing…" with a spinner; on completion the metadata refreshes without a page reload.
- [ ] On a `failed` job, the inline error banner shows the `error_message` and a "Retry sync" button.
- [ ] All new Vitest specs pass; `pnpm run vitest`, `pnpm run lint`, `pnpm run fmt.check` are green.

Cross-PR:

- [ ] Total diff across all 4 PRs is under 3,000 changed lines (including tests).
- [ ] Each individual PR is under the 400-line review budget.
- [ ] The full SDD lifecycle is followed: explore → proposal → spec → design → tasks → apply (per PR) → verify (per PR) → sync (per PR) → archive (after the last PR).
- [ ] Strict TDD evidence per PR is captured in `apply-progress.md` with RED → GREEN → TRIANGULATE → REFACTOR per task.
- [ ] The `adr/workspace-syncer-git-impl` ADR is persisted to Engram and referenced from the proposal.
- [ ] No file outside the listed Affected areas is modified.

---

## Notes for the `sdd-spec` phase

The spec phase must produce two specs:

1. **`openspec/changes/2026-07-08-workspace-sync-clone/specs/workspaces/spec.md`** (delta) — appends R-WS-019 (Workspace sync) to the canonical `openspec/specs/workspaces/spec.md`. Use Given/When/Then + RFC 2119 keywords per `openspec/config.yaml`. Each scenario independently verifiable. All new IDs are BIGSERIAL (NOT UUID). Carry forward the 5 locked Q&A decisions from the proposal: button disabled with spinner, v1 metadata set, soft-delete let-complete, default branch only, inline error banner. Cover at minimum: auto-sync on create, manual sync trigger, single-flight 409, poll status, permissions-failure UX, soft-delete cleanup, and the API wire shapes for `POST /workspaces`, `POST /workspaces/:id/sync`, `GET /workspaces/:id/sync`, and the `GET /workspaces/:id` response.

2. **`openspec/changes/2026-07-08-workspace-sync-clone/specs/workspace-syncer/spec.md`** (new) — the first spec for the new service. Document the internal endpoint contract, the worktree probe requirement, the disk layout, the soft-delete cleanup behavior, the bearer-token middleware, and the callback shape. Use the same Given/When/Then format. The first requirement group could be "R-WSY-001 — Clone and validate" with sub-scenarios for: happy path, insufficient permissions, default branch not found, worktree probe failure, token expired, soft-delete mid-sync.

## Notes for the `sdd-design` phase

The design phase must produce a `design.md` that includes:

- **Two sequence diagrams** (the 4R reviewers in `sdd-apply` need them):
  1. Auto-sync-on-create path: client → database_administrator → workspace_syncer → git → workspace_syncer → database_administrator callback → database_administrator updates → response to client.
  2. Manual sync path: client → database_administrator → workspace_syncer → git → workspace_syncer → database_administrator callback → database_administrator updates → client polls.
- **Database state diagram** for the `sync_job` row (pending → running → done | failed; with the partial unique index enforcement on the pending|running side).
- **Filesystem layout** for `/data/workspaces/{workspace_id}/{owner}/{repo}.git/` — bare mirror vs working tree (initial recommendation: bare mirror). Show the cleanup-on-soft-delete path.
- **The 4-PR diff forecast** with per-PR file inventory and LoC estimate (the tasks phase uses this).
- **The cross-service auth posture** including the v1 static-token + v2-JWT ADR cross-reference.
- **Strict TDD evidence requirements** for each PR (the apply phase must record RED → GREEN → TRIANGULATE → REFACTOR per task in `apply-progress.md`).
- **The error envelope consolidation** — the design must codify that the workspace_syncer's errors are translated to the database_administrator envelope shape before the callback is posted. The syncer does NOT have its own public error envelope; only the internal contract.

## Notes for the `sdd-tasks` phase

The tasks phase must produce a `tasks.md` with:

- Per-PR task list (T-WSY-1-001, T-WSY-2-001, etc.).
- Each task completable in a single PR (per `openspec/config.yaml`).
- Forecast per PR's changed-line count and stay under 400 per PR. Auto-forecast per preflight means the task agent is allowed to chain PRs without re-asking.
- Each task includes its strict TDD evidence expectation (what test fails first, what production code is the minimum GREEN, what TRIANGULATE cases are added, what REFACTOR cleanup is done).
- The cross-service auth ADR is recorded as T-WSY-0-001 (precedes PR-1) with topic `adr/workspace-syncer-internal-auth`.
- The git-impl ADR is recorded as T-WSY-0-002 (precedes PR-2) with topic `adr/workspace-syncer-git-impl`.
