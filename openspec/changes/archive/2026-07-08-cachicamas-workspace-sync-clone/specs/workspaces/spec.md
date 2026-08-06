# Spec delta — Workspaces (workspace sync card)

> **Change**: `2026-07-08-workspace-sync-clone`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/workspaces/spec.md` (this file describes the deltas only; sdd-sync consolidates)
> **Format**: Given/When/Then + RFC 2119 keywords per `openspec/config.yaml`
> **Type convention**: all new IDs are `BIGSERIAL` (int64) per the type correction in the explore phase. NO UUIDs.

## Purpose

Add a new requirement group `R-WS-019` (Workspace sync) to the existing Workspaces spec. Update the wire shape of the existing `POST /workspaces` (R-WS-001) and `GET /workspaces/:id` (R-WS-003) to carry the new sync metadata fields. Lock the behavior of the new endpoints `POST /workspaces/:id/sync` and `GET /workspaces/:id/sync`. Lock the card's UX affordances (button states, error banner) on the workspace detail page. Lock the auto-sync-on-create timing. Lock the single-flight invariant. Lock the cleanup-on-soft-delete invariant.

The motivation is the deferred tech-debt item in `openspec/changes/archive/2026-07-06-workspaces/proposal.md` ("Actual repo cloning (uses persisted access_token)"). This spec is the contract that `sdd-apply` implements across 4 chained PRs and that `sdd-verify` checks against.

## Type convention (locked)

- `workspace.id` is `BIGSERIAL` (int64). All new identifiers follow the same convention.
- `sync_job.id` is `BIGSERIAL`.
- `workspace.last_sync_job_id` is `BIGINT NULL REFERENCES sync_job(id) ON DELETE SET NULL`.
- **No UUIDs are introduced in this change.** If a future change needs UUIDs, it is its own decision and its own ADR.

## Decisions carried forward from the proposal (locked)

These were decided in `proposal.md` §"Decisions taken in this proposal" and are ratifiable in review but not negotiable in this spec phase:

1. Sync button during a running job MUST render as **disabled with a spinner + "Syncing…" text**. Re-enables on `done` or `failed`. No progress percentage in v1.
2. Metadata v1 field set MUST be `default_branch`, `primary_language`, `pushed_at`, `visibility`, `size_kb`. No `stars`. No `description`. No `topics`. No `license`.
3. Soft-delete mid-sync MUST let the sync complete, then drop the result. The `ON DELETE SET NULL` FK on `workspace.last_sync_job_id` handles the row reference; the workspace_syncer's cleanup hook removes the cloned data on soft-delete.
4. Sync target ref MUST be the workspace's `default_branch` only in v1. The internal API contract documents a future `ref` field; the public `POST /workspaces/:id/sync` body does NOT carry `ref` yet.
5. Error-state UI MUST be an inline banner inside the card with the `error_message` and a "Retry sync" button. No top-of-page banner. No metadata-block replacement.

## New requirement group

### R-WS-019 — Workspace sync

A workspace can be synced against its primary GitHub repository. Sync validates the persisted OAuth token's `permissions.push === true` (the precondition for both `git worktree` and PR creation), clones the repository into the new `workspace_syncer` service, runs a worktree probe, and stores the outcome. A workspace is created with an in-flight sync job in v1; the user can also trigger a manual sync at any time. The card on `/workspaces/:id` surfaces the latest sync date, the commit SHA at the moment of sync, useful repository metadata, and a Sync button.

#### Scenarios

- **S-WS-190** — Given a signed-in, ownboarded user POSTs `/workspaces` with a valid `{name, repository}` payload and the GitHub token grants `permissions.push === true`, when the workspace is created, then the response is `201 Created` with `Location: /workspaces/{id}` AND the body carries the new field `initial_sync_job_id` (BIGSERIAL) AND the body carries `last_synced_at: null` AND the body carries `last_sync_job_id: null` AND the new `sync_job` row has `status = "pending"` AND `triggered_by = "auto_on_create"`.
- **S-WS-191** — Given a live workspace with id X, when a signed-in user POSTs `/workspaces/X/sync` with an empty body, then the response is `202 Accepted` AND the body is `{"job_id": <BIGSERIAL>, "status": "pending"}` AND the `sync_job` row has `status = "pending"` AND `triggered_by = "manual"`.
- **S-WS-192** — Given a sync_job with id J for workspace X and the user has not refreshed the page, when a signed-in user GETs `/workspaces/X/sync`, then the response is `200 OK` AND the body is `{"job_id": J, "status": <"pending"|"running"|"done"|"failed">, "started_at": <TIMESTAMPTZ|null>, "finished_at": <TIMESTAMPTZ|null>, "commit_sha_after": <TEXT|null>, "error_message": <TEXT|null>, "error_code": <TEXT|null>, "attempts": <INT>}`.
- **S-WS-193** — Given a sync_job is in `pending` or `running` for workspace X (job_id J), when a second signed-in user POSTs `/workspaces/X/sync`, then the response is `409 Conflict` AND the body is `{"error": "sync_already_running", "message": "A sync is already in progress for this workspace.", "job_id": J}`.
- **S-WS-194** — Given a sync_job is in `running`, when the workspace_syncer reports back via the internal callback with `status = "done"` AND `commit_sha_after = "abc1234..."`, then the `sync_job` row is updated to `status = "done"`, `finished_at = now()`, `commit_sha_after = "abc1234..."` AND the `workspace` row is updated to `last_synced_at = now()`, `last_synced_commit_sha = "abc1234..."`, `last_sync_job_id = J` AND a subsequent GET `/workspaces/:id` returns the updated fields.
- **S-WS-195** — Given the OAuth token does NOT grant `permissions.push === true` on the workspace's primary repo, when the `GetRepository` preflight runs, then the `sync_job` row is updated to `status = "failed"`, `error_code = "WORKSPACE_PERMISSIONS_INSUFFICIENT"`, `error_message = "Token lacks push permission on this repository."` AND the workspace row's `last_synced_at` is NOT updated AND the card surfaces the error banner with the error_message and a "Retry sync" button.
- **S-WS-196** — Given a workspace is soft-deleted (`deleted_at` set to a non-NULL value) AND a sync_job is in `running` for that workspace, when the workspace_syncer finishes the clone, then the syncer's cleanup hook removes the cloned data under `/data/workspaces/{workspace_id}/...` (idempotent) AND the `sync_job` row's `status` and `commit_sha_after` are written BUT the `workspace` row's `last_sync_job_id` is set to NULL by the `ON DELETE SET NULL` FK (when the FK is materialized; for soft-delete, the FK keeps the value and a future sweeper purges the orphaned job after a retention window). The card on the workspace is gone (the workspace is soft-deleted); no UX impact.
- **S-WS-197** — Given a sync_job is in `pending` or `running`, when the user views `/workspaces/:id`, then the Sync button is rendered as `<button disabled aria-busy="true">` with a spinner icon and the text "Syncing…" AND the metadata block is NOT replaced — it shows the last-known-good data (or "Not synced yet." if no prior sync exists).
- **S-WS-198** — Given a sync_job is in `failed` with a non-NULL `error_message`, when the user views `/workspaces/:id`, then the card renders an inline banner with the `error_message` text AND a "Retry sync" button that, on click, calls `POST /workspaces/:id/sync` (which will 409 if another job is pending/running).
- **S-WS-199** — Given the OAuth token is `NULL` for the user (legacy pre-PR-1a row), when the user POSTs `/workspaces` or `POST /workspaces/:id/sync`, then the response is `401 Unauthorized` AND the body is `{"error": "github_not_connected", "message": "Reconnect GitHub to enable sync."}` (the existing `github_not_connected` envelope from R-WS-017).
- **S-WS-200** — Given a live workspace with id X, when a signed-in user GETs `/workspaces/X/sync` AND no sync_job exists for X, then the response is `200 OK` AND the body is `{"job_id": null, "status": null, ...}` (all sync fields NULL) AND the card on the detail page falls back to "Not synced yet." copy.

### Wire shape updates to existing requirements

The following updates are appended to the canonical spec during `sdd-sync`. They are listed here for traceability.

#### R-WS-001 (POST /workspaces) — wire shape delta

- Response body now includes:
  - `initial_sync_job_id: BIGSERIAL` — the id of the auto-on-create sync_job, or `null` if the token lacks permissions and the job was never enqueued.
  - `last_synced_at: TIMESTAMPTZ | null` — `null` on a fresh workspace.
  - `last_sync_job_id: BIGSERIAL | null` — `null` on a fresh workspace.
- Existing fields unchanged: `id`, `name`, `organization_id`, `owner_user_id`, `repository` (with `github_id`, `full_name`, `owner`, `name`), `created_at`, `updated_at`, `deleted_at`.

#### R-WS-003 (GET /workspaces/:id) — wire shape delta

- Response body now includes:
  - `last_synced_at: TIMESTAMPTZ | null`
  - `last_synced_commit_sha: TEXT | null` — the short SHA (7 chars) is enough for the card; the full SHA is stored on the row.
  - `default_branch: TEXT | null` — the GitHub default branch (e.g. `"main"`, `"master"`).
  - `last_sync_job_id: BIGSERIAL | null`
- Existing fields unchanged.

#### R-WS-001 through R-WS-018 — invariants preserved

- The `unique_index workspace_org_name_live_key` partial unique index on `(organization_id, name) WHERE deleted_at IS NULL` is preserved. The new `last_synced_at`, `last_synced_commit_sha`, `default_branch`, `last_sync_job_id` columns are NULLable and do not affect the uniqueness invariant.
- The 1:1 model (workspace = one repo) from the 2026-07-08-workspaces-simplify change is preserved. No `workspace_repository` table; the sync operates on the single primary repository.

## Strict TDD posture

`openspec/config.yaml` declares `apply.tdd: true` and `apply.test_command: "go test ./..."`. The apply phase MUST record RED → GREEN → TRIANGULATE → REFACTOR evidence in `apply-progress.md` per PR, with the following minimum coverage per PR:

- **PR-1 (migration)**: at least 3 migration tests — the `sync_job` table exists with the locked columns, the partial unique index is present, the `workspace` ALTER added the 4 columns with the right types.
- **PR-2 (workspace_syncer)**: at least 6 tests in `infrastructure/git/runner_test.go` (clone happy path, clone with bad token, worktree probe, path-safety reject, shell injection reject, token redaction in logs); at least 4 tests in `interfaces/http/handler_test.go` (auth, happy path, error envelope, missing fields); at least 2 tests in `application/clone_service_test.go` (use-case happy path, use-case error mapping).
- **PR-3 (database_administrator)**: at least 4 tests in `application/sync_service_test.go` (EnqueueSync, GetLatestSyncJob, ProcessSyncCallback happy, ProcessSyncCallback failed); at least 4 tests in `infrastructure/postgres/sync_job_repo_test.go` (Insert, GetLatest, LockActive, single-flight unique violation); at least 3 tests in `interfaces/http/internal_callback_handler_test.go` (callback idempotency, callback auth, callback 404); at least 2 tests in `interfaces/http/workspace_handler_test.go` for the new sync endpoints.
- **PR-4 (frontend)**: at least 6 tests in `workspace-sync-card.spec.tsx` (renders metadata, button enabled, button disabled during pending/running, error banner, polling stops on done, polling stops on failed); at least 3 tests in `use-sync-status.spec.ts` (initial fetch, polling interval, polling stops on terminal state); the route-guard structural spec for `/workspaces/:id` is updated to assert the card is mounted.

## Out of scope (per proposal, re-locked here for spec phase)

- Webhook ingestion from GitHub.
- PR auto-creation (only the permission check is in scope; no PRs are created).
- Worktree merge/rebase or any worktree-based feature beyond the probe.
- Token encryption-at-rest.
- OAuth token refresh cron.
- Multi-tenant keying of the new `workspace_syncer` service.
- Restore of soft-deleted workspaces.
- Hard delete of soft-deleted workspaces.
- Badge / status indicator on the workspace list cards.
- Sync from a specific ref (v1 is `default_branch` only).
- Per-workspace disk usage quota (deferred until production shows pressure).

## Non-functional requirements

### NFR-WS-019-A — Performance

- `POST /workspaces/:id/sync` response time < 50ms p95 (the work happens in workspace_syncer; this endpoint only inserts the job row).
- `GET /workspaces/:id/sync` response time < 30ms p95 (indexed query on `sync_job(workspace_id)`).
- The card on the workspace detail page MUST render the cached metadata within 100ms p95 of the route loader finishing.

### NFR-WS-019-B — Security

- The OAuth token MUST NOT be returned in any HTTP response. The card's `error_message` MUST NOT contain the token.
- The internal service-to-service bearer token MUST be validated on every request to `workspace_syncer` AND every callback to `database_administrator`. Missing or wrong token → 401.
- The `workspace_id` used as a filesystem path component MUST be validated as ASCII digits only (BIGSERIAL is always digits; the validation is defense-in-depth against future code paths).
- The OAuth token MUST be redacted from any log line emitted by `workspace_syncer`. A custom slog handler enforces this.

### NFR-WS-019-C — Reliability

- A single-flight invariant is enforced at the database level via the partial unique index `(workspace_id) WHERE status IN ('pending','running')`. A concurrent `INSERT` of a second job for the same workspace fails with `pgconn.PgError.Code = "23505"`; the adapter translates this to `*domain.ConflictError` with code `sync_already_running`.
- The `workspace_syncer` callback to `database_administrator` MUST be idempotent: a re-post of the same `done` callback MUST NOT double-update `workspace.last_synced_at`. The handler uses `WHERE id = $1 AND status = 'running'` in the UPDATE; a second `done` callback matches no rows and is logged as a no-op.
- The clone timeout on `workspace_syncer` is 90s. A timeout produces `error_code: "CLONE_TIMEOUT"`, `error_message: "Repository clone took longer than 90 seconds."`.

### NFR-WS-019-D — Observability

- `database_administrator` emits a new OTel span `sync.enqueue` (HTTP 201) and `sync.status` (HTTP 200) for the new endpoints. Attributes: `http.method`, `http.route`, `http.status_code`, `workspace.id`, `sync_job.id`, `sync_job.status`.
- `workspace_syncer` emits a new OTel span `clone.execute` with attributes: `workspace.id`, `owner`, `repo`, `default_branch`, `clone.duration_ms`, `worktree.probe.exit_code`, `worktree.probe.duration_ms`.
- Every sync lifecycle event (`enqueue`, `start`, `done`, `failed`, `timeout`) emits a structured slog line with the `job_id` and `workspace_id`.

### NFR-WS-019-E — Migration safety

- The new migration `20260708120000_sync_job.sql` is purely additive: new table, ALTER TABLE with nullable columns. No backfill is required.
- The new `workspace.last_sync_job_id` column has `ON DELETE SET NULL` so a `sync_job` row can be hard-deleted (e.g. in a future retention sweeper) without orphaning the workspace row.
- The `Down` section of the migration drops the new columns and the new table. In production, the Down is documented but not run; forward-fix is the policy.

## Acceptance criteria

The change is accepted when:

1. All scenarios in this delta spec are implemented and tested (RED → GREEN → TRIANGULATE → REFACTOR per task).
2. The full SDD lifecycle is followed: explore → proposal → spec → design → tasks → apply (4 PRs) → verify (per PR) → sync (per PR) → archive (after the last PR).
3. The 4 chained PRs are under 400 changed lines each.
4. The full test suite passes:
   - `cd backend/database_administrator && make test` (race-clean)
   - `cd backend/workspace_syncer && make test` (race-clean)
   - `cd frontend && pnpm run vitest`
   - `cd frontend && pnpm run lint` and `pnpm run fmt.check`
5. The two ADRs (`adr/workspace-syncer-git-impl`, `adr/workspace-syncer-internal-auth`) are persisted to Engram and referenced from the proposal.
6. The canonical `openspec/specs/workspaces/spec.md` is updated (during `sdd-sync`) with R-WS-019 and the wire-shape deltas to R-WS-001 and R-WS-003.
7. The new canonical `openspec/specs/workspace-syncer/spec.md` is written (during `sdd-sync`) from the workspace-syncer spec in this change.
