# Explore — Workspace Sync Card & Repository Cloning

> **Change**: `2026-07-08-workspace-sync-clone`
> **Phase**: explore
> **Project**: cachicamas
> **Date**: 2026-07-08
> **Artifact store**: engram (topic `sdd/2026-07-08-workspace-sync-clone/explore`, id 1831) + filesystem mirror (this file)
> **Skill resolution**: `paths-injected` (no project skills required for explore; registry at `.atl/skill-registry.md`)
> **Predecessor**: implements the deferred item in `openspec/changes/archive/2026-07-06-workspaces/proposal.md` ("Actual repo cloning (uses persisted access_token)")

## 1. Feature shape

The workspace system gains a server-side GitHub repository synchronization capability. When a workspace is created, the system auto-enqueues a sync job that (a) clones the repository inside the new `workspace_syncer` service, and (b) validates that the persisted OAuth token grants `repo` scope with `permissions.push === true` — the precondition for both `git worktree` operations and pull-request creation. The workspace detail page (`/workspaces/:id`) gains a new card that surfaces the latest sync date, the latest commit hash captured at the moment of sync, a "Sync" button that triggers a pull from the default branch, and useful repository metadata (default branch, last commit, primary language, last push date, visibility). Sync operations are processed asynchronously: `POST /workspaces/:id/sync` returns HTTP 202 with a `job_id`; clients poll `GET /workspaces/:id/sync` for status (`pending` / `running` / `done` / `failed`).

## 2. Affected surfaces

| Surface | Kind | Scope | Description |
| --- | --- | --- | --- |
| `backend/database_administrator` | modified | domain, application, infrastructure (postgres + github), interfaces/http | Add `last_synced_at`, `last_synced_commit_sha`, `default_branch`, `last_sync_job_id` on `workspace`; add `sync_job` table; extend `WorkspaceService` with `EnqueueSync`, `GetSyncJob`; add `POST /workspaces/:id/sync` and `GET /workspaces/:id/sync`; auto-enqueue first sync inside `POST /workspaces`. |
| `backend/workspace_syncer` | **new service** | full hexagonal service (go.mod, cmd/, src/{domain,application,infrastructure,interfaces}) | Echo v5 service that owns git, the local filesystem layout, and the actual `git clone` / `git fetch` / `git worktree add` invocations. Exposes `POST /internal/clone-and-validate` and reports results back. |
| `frontend` | modified | `routes/workspaces/[id]/`, new `components/workspace-sync-card`, `adapters/api-client/` | New card on the workspace detail page; polling for job status; Sync button; metadata display. No badge changes to the workspace list page. |
| `docker-compose.yaml`, `docker-compose.vps.yaml` | modified | services + networks + volumes | Add `workspace_syncer` service, network link to `database_administrator`, shared volume for cloned repos. |
| `infra/postgres/init/01-init.sql` | modified | DDL + GRANTs | Add `sync_job` table; grant CRUD to the existing service role. |
| `openspec/specs/workspaces/spec.md` | modified | scenarios | New "Workspace sync" capability; scenarios "User triggers sync", "Auto-sync runs on workspace creation", "Sync surfaces permissions failure". Update `POST /workspaces` and `GET /workspaces/:id` response shapes. |
| `openspec/specs/workspace-syncer/spec.md` | **new spec** | new capability | Define the internal HTTP contract owned by `workspace_syncer`. |

## 3. Out of scope

1. Webhook ingestion from GitHub for incremental sync.
2. PR auto-creation. We only validate the token has permission to create PRs; we do not create any.
3. Worktree merge/rebase logic — `git worktree add` is the precondition for a future "Create PR" feature, itself out of scope here.
4. Encryption-at-rest for the persisted OAuth token.
5. OAuth token refresh cron / background expiry handling.
6. Multi-tenant keying of the new `workspace_syncer` service (single shared identity for v1).
7. Restore of soft-deleted workspaces (existing deferred item).
8. Hard delete of soft-deleted workspaces (existing deferred item).
9. Anything already declared out-of-scope by `openspec/changes/archive/2026-07-06-workspaces/proposal.md` and not strictly required by this change (badge on list cards, etc.).

## 4. Data model delta (preliminary, refined in design)

### `workspace` (modified)

> Type note: `workspace.id` is `BIGSERIAL` (int64) per `migration/sql/20260706120002_workspaces.sql` and `20260708120100_rename_workspace_primary_repo_columns.sql`. `sync_job.id` and `last_sync_job_id` must follow the same convention — **not UUID**.

```sql
ALTER TABLE workspace
  ADD COLUMN last_synced_at         TIMESTAMPTZ NULL,
  ADD COLUMN last_synced_commit_sha TEXT        NULL,
  ADD COLUMN default_branch         TEXT        NULL,
  ADD COLUMN last_sync_job_id       BIGINT      NULL REFERENCES sync_job(id) ON DELETE SET NULL;
```

### `sync_job` (new)

```sql
CREATE TABLE sync_job (
  id                BIGSERIAL    PRIMARY KEY,
  workspace_id      BIGINT       NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
  status            TEXT         NOT NULL CHECK (status IN ('pending','running','done','failed')),
  triggered_by      TEXT         NOT NULL CHECK (triggered_by IN ('auto_on_create','manual')),
  started_at        TIMESTAMPTZ  NULL,
  finished_at       TIMESTAMPTZ  NULL,
  commit_sha_after  TEXT         NULL,
  error_message     TEXT         NULL,
  error_code        TEXT         NULL,
  attempts          INT          NOT NULL DEFAULT 0,
  created_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX sync_job_workspace_id_idx ON sync_job(workspace_id);
CREATE INDEX sync_job_status_idx       ON sync_job(status);

-- Single-flight: at most one pending/running job per workspace
CREATE UNIQUE INDEX sync_job_single_flight_uidx
  ON sync_job(workspace_id)
  WHERE status IN ('pending','running');
```

## 5. Cross-service contract draft

### Authentication

- A static service-to-service token, identical to the model currently used by `database_administrator` for outbound calls (env var `INTERNAL_SERVICE_TOKEN`), is presented via `Authorization: Bearer <token>` and validated by Echo middleware on `workspace_syncer`.
- For v1, docker network trust is the primary mitigation; the shared token is defense-in-depth and enables the same code path to run in CI without docker.

### `POST /internal/clone-and-validate` (database_administrator → workspace_syncer)

Request:

```json
{
  "job_id":         "42",
  "workspace_id":   "7",
  "owner":          "octocat",
  "repo":           "hello-world",
  "default_branch": "main",
  "oauth_token":    "gho_..."
}
```

Response (200):

```json
{
  "job_id":          "42",
  "status":          "done",
  "commit_sha_after": "abc1234...",
  "started_at":      "2026-07-08T12:00:00Z",
  "finished_at":     "2026-07-08T12:00:42Z"
}
```

Error envelope (consistent with `database_administrator`):

```json
{ "error": "WORKSPACE_PERMISSIONS_INSUFFICIENT", "message": "Token lacks push permission" }
```

### Ownership

- `database_administrator` owns: OAuth token store, `sync_job` table, public HTTP API (`POST /workspaces`, `POST /workspaces/:id/sync`, `GET /workspaces/:id/sync`).
- `workspace_syncer` owns: local filesystem layout (`/data/workspaces/{workspace_id}/{owner}/{repo}.git`), git CLI invocation, token redaction in logs. No Postgres access.

## 6. Permissions validation (preliminary)

A single GitHub REST endpoint validates both capabilities:

- **Endpoint**: `GET https://api.github.com/repos/{owner}/{repo}` with `Authorization: Bearer {token}`.
- **Required response fields**:
  - `permissions.push === true` → required for both `git worktree` (token-holder for git operations) and PR creation (`POST /repos/{owner}/{repo}/pulls` requires push).
- **Scope check (best-effort)**: read `X-OAuth-Scopes` response header (documented as deprecated by GitHub but still supported); fall back to probing another repo write surface.
- **Notes**:
  - The actual `git worktree add` and any future PR creation happen in `workspace_syncer`. Explore phase only validates the token up front; `sdd-proposal` will codify this as a reusable `ports.GitHubPermissionsChecker` interface.
  - When `default_branch` is known, additionally verify it exists via `GET /repos/{owner}/{repo}/branches/{default_branch}` to avoid surfacing "branch missing" only after a 60s clone.

## 7. Risk register (initial)

| # | Risk | Likelihood | Mitigation |
| --- | --- | --- | --- |
| 1 | Clone timeout on large repos | medium | 60–120s HTTP timeout on `workspace_syncer`; async lets the frontend poll; log `started_at` / `finished_at` for diagnosis. |
| 2 | Disk pressure when many workspaces are cloned | medium | Cap per-workspace disk usage (~5 GiB); document layout under `/data/workspaces/{workspace_id}/...`; remove cloned data on workspace soft-delete (cleanup hook in `WorkspaceService.Delete`). |
| 3 | Token expiry mid-clone | low | Set `error_message` with code `TOKEN_EXPIRED`; surface to UI banner; user re-syncs after reconnecting. |
| 4 | Race between two sync jobs for the same workspace | medium (double-clicks) | Single-flight via partial unique index `sync_job_single_flight_uidx`; handler returns 409 `SYNC_ALREADY_RUNNING`. |
| 5 | Schema migration introduces a new Postgres role | low | Extend `infra/postgres/init/01-init.sql` to grant CRUD on `sync_job` to the existing `database_administrator` role. |
| 6 | Cross-service auth posture | medium | Static shared service token + docker network trust for v1; document v2 move to HMAC-signed short-lived JWT. |
| 7 | Chained PR strategy for apply | high | Apply is split into 4 chained PRs each <400 lines: (a) `sync_job` table + ALTER, (b) `workspace_syncer` skeleton + clone logic, (c) `database_administrator` HTTP + service additions, (d) frontend sync card. |

## 8. Open questions for `sdd-proposal`

1. **Sync button affordance during a running job** — disabled, hidden, or "Running… (x/y)" with progress?
2. **"Useful metadata" definition** — pick v1 fields: default branch, primary language, last push date, stars, visibility (private/public), repo size? The frontend contract depends on this.
3. **In-flight sync when a workspace is soft-deleted** — soft-cancel with `error_message = "workspace deleted"`, or let it complete and ignore the result?
4. **Sync from a specific ref** — v1 is `default_branch` only; does the user want a ref picker now or defer to v2?
5. **Error-state card placement** — banner above the card, replaces the metadata block, or surfaces an inline icon?

## 9. Related code paths and existing helpers to leverage

Files to touch in `backend/database_administrator/`:

- `src/domain/workspace.go` — extend the `Workspace` aggregate with `LastSyncedAt *time.Time`, `LastSyncedCommitSha *string`, `DefaultBranch *string`; add the new `SyncJob` aggregate (`Status`, `StartedAt`, `FinishedAt`, `CommitShaAfter`, `ErrorMessage`, `Attempts`, `TriggeredBy`).
- `src/domain/errors.go` — add `ErrSyncAlreadyRunning`, `ErrTokenExpired`, `ErrInsufficientPermissions`.
- `src/application/workspace_service.go` — add `EnqueueSync(ctx, workspaceID, triggeredBy) (jobID, error)`, `GetLatestSyncJob(ctx, workspaceID) (SyncJob, error)`, `MarkWorkspaceSynced(workspaceID, commitSHA)`. The create-workspace use case calls `EnqueueSync(..., "auto_on_create")` on success.
- `src/infrastructure/postgres/workspace_repo.go` — extend queries for new columns.
- `src/infrastructure/postgres/sync_job_repo.go` — **new file**: `Insert`, `Update`, `GetLatestForWorkspace`, `LockActiveForWorkspace`.
- `src/infrastructure/github/client.go` — extend with `GetRepository(ctx, owner, repo) (*Repository, error)` returning `permissions`, `default_branch`, `language`, `pushed_at`, `visibility`, etc.
- `src/infrastructure/github/cache.go` — already in use; reuse with new resource keys.
- `src/interfaces/http/workspace_handler.go` — add `SyncHandler.Post`, `SyncHandler.Get`. Modify `CreateHandler` to return `last_synced_at: null` plus `initial_sync_job_id`.
- `src/interfaces/http/tokenctx/` — already extracts the OAuth token from the user session; reuse it server-side.
- `src/cmd/server/main.go` — register new routes; wire `workspace_syncer` HTTP client.
- `Dockerfile`, `Makefile` — extend build/test targets if needed (no new build deps expected for v1).

Frontend files:

- `frontend/src/routes/workspaces/[id]/index.tsx` (or equivalent) — mount the new card.
- **New**: `frontend/src/components/workspace-sync-card/` — owns the polling loop and the Sync button.
- `frontend/src/components/workspace-card/` — unchanged.
- `frontend/src/components/workspace-form/` — unchanged.
- `frontend/src/components/home-workspaces-section/` — unchanged.
- `frontend/src/adapters/api-client/` — add `startWorkspaceSync(id)`, `getWorkspaceSyncStatus(id)`.

Other:

- `infra/postgres/init/01-init.sql` — add `sync_job` DDL + GRANTs.
- `docker-compose.yaml`, `docker-compose.vps.yaml` — add `workspace_syncer` service.
- `openspec/specs/workspaces/spec.md`, `openspec/AGENTS.md` — update the spec; any new dep requires an ADR per `openspec/AGENTS.md`.

## 10. Skill paths to load in `sdd-proposal` and `sdd-apply`

These five registry skills will be required for the apply chain; the orchestrator has them indexed (registry at `cachicamas/.atl/skill-registry.md`):

- `go-testing` — table-driven Given/When/Then tests for `workspace_syncer` and `database_administrator`.
- `test-driven-development` — strict TDD per `openspec/config.yaml`.
- `work-unit-commits` — each work unit on its own commit/PR within the chained apply.
- `chained-pr` — the 4-PR apply plan in §7 is executed as a chain.
- `cognitive-doc-design` — keeps this `explore.md` and the proposal doc navigable.

No project-user skills were required for the explore phase itself; this document was constructed entirely from the task brief plus knowledge already encoded in `openspec/AGENTS.md` and the deferred `2026-07-06-workspaces` proposal.
