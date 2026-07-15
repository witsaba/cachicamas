# Proposal — `2026-07-15-prompt-storage-table`

> **Change**: `2026-07-15-prompt-storage-table`
> **Project**: `cachicamas`
> **Artifact store**: `openspec/changes/2026-07-15-prompt-storage-table/`
> **Strict TDD**: ACTIVE — every code task starts with a failing test (RED → GREEN → REFACTOR).
> **Inherits**: `explore.md` of this change; `openspec/config.yaml`; `openspec/AGENTS.md`.

---

## 0. Decision log (locked)

These decisions were confirmed with the user before this proposal was written. They are binding for `spec.md`, `design.md`, and `tasks.md`.

| ID | Decision | Source |
| --- | --- | --- |
| D1 | Two tables: `prompt` (current definitive row) + `prompt_revision` (append-only history). | User — refined from "Con versionado desde v1": "Es mejor mantener una tabla aparte con el historial de versiones. Es mejor, en la tabla principal estarán los definitivos y en el historial, todo el cambio, así, regresarse a una versión va a ser transparente y la tabla principal sólo tendrá la información que necesita tener." |
| D2 | `prompt.body` is `TEXT` (UTF-8 markdown), not `BYTEA` nor `JSONB`. | User (industry standard; full-text search available later; readable in psql). |
| D3 | `prompt` is global (no `organization_id` / `workspace_id` FK). Slug is unique across active rows only. | User (simplest v1; multi-tenant can be added later via additive migration). |
| D4 | Soft-delete via `deleted_at TIMESTAMPTZ NULL`. Partial `UNIQUE INDEX` on `slug WHERE deleted_at IS NULL` reuses the slug on delete. | User (consistent with `workspace` table; allows reuse after delete). |
| D5 | Versioning model: `prompt.body` always holds the latest content; every `UPDATE` to body/description appends a `prompt_revision` row; `revision_number` is monotonic per `prompt_id`. | User's D1 + D2 + this proposal §3. |
| D6 | Minimal CRUD HTTP surface in v1: `GET /prompts`, `GET /prompts/:slug`, `POST /prompts`, `PATCH /prompts/:slug`, `DELETE /prompts/:slug`, `GET /prompts/:slug/revisions`, `POST /prompts/:slug/revisions/:n/restore`. | This proposal §5. |
| D7 | Forward-only migration (empty `Down` block), consistent with `sync_job.sql` and `drop_workspace_repository.sql`. | Project norm (`openspec/AGENTS.md`). |
| D8 | Use `pgx` only inside `src/infrastructure/postgres/`. Domain has no pgx import. | Project hexagonal rule. |
| D9 | Errors map: SQLSTATE 23505 → `*domain.ConflictError`; "no rows" → `*domain.NotFoundError`. | Established by `workspace_repo.go`. |
| D10 | Strict TDD: every Go task starts with a failing test; `go test -race -v ./...` passes at each task boundary. | `openspec/config.yaml` + `openspec/AGENTS.md`. |

---

## 1. Problem

The cachicamas backend has no persistent storage for LLM prompts. Today, prompts either:

- Live as constants in Go source code (no versioning, no audit, hard to update),
- Are scattered across the frontend (no source of truth),
- Or are not stored at all and are reconstructed ad-hoc per request.

This change introduces a **durable, versioned, slug-addressable prompt store** backed by PostgreSQL so that prompts become:

- A first-class data asset (findable, auditable, reversible),
- Editable without redeploying the Go binary,
- Safe to evolve (every change is preserved as a revision),
- Reusable across workspaces and organizations (no tenancy in v1, by design).

---

## 2. Target users and situations

| Persona | Use case |
| --- | --- |
| **Backend dev** writing prompt logic | `SELECT body FROM prompt WHERE slug = 'welcome-email'` — the LLM driver reads the current body. |
| **Operator / SRE** promoting a prompt | `INSERT INTO prompt (description, slug, body) VALUES (...)` — single statement, immediately visible. |
| **Product owner reviewing changes** | `SELECT … FROM prompt_revision WHERE prompt_id = X ORDER BY revision_number DESC` — full history. |
| **AI / automation reverting a regression** | `POST /prompts/:slug/revisions/:n/restore` — atomic revert by re-inserting a past revision as the new latest. |
| **Frontend (deferred)** | Read-only consumer of the API in a follow-up change. |

Non-users (out of scope for v1):

- End users who would edit prompts through a UI (no UI in v1).
- Multi-tenant SaaS customers who would expect `organization_id` on every row (deferred to v1.1).

---

## 3. Product outcome

After this change ships:

1. `prompt` and `prompt_revision` exist in the main database, owned by `queen`.
2. The Go service exposes CRUD + revision endpoints under `/prompts`.
3. Any LLM driver in the codebase can do `repo.SelectBySlug(ctx, "welcome-email")` and get the current body.
4. Every modification is preserved as a `prompt_revision` row — `revision_number` is monotonic per `prompt`.
5. Soft-deleting a prompt frees the slug for reuse.
6. Reverting to a past revision is one API call and is itself recorded as a new revision.

---

## 4. Current-state gap

| Today's reality | After this change |
| --- | --- |
| Prompts are inline strings in Go code | Prompts are first-class rows in `prompt` |
| No audit trail of what changed | `prompt_revision` has the full change log |
| Updating a prompt = re-deploy | Updating = `PATCH /prompts/:slug` |
| No way to revert a bad prompt | `POST /prompts/:slug/revisions/:n/restore` |
| Slug collisions not protected | Partial UNIQUE on `slug WHERE deleted_at IS NULL` |

---

## 5. Product decisions locked in this proposal

### 5.1 Two-table model (D1, D5)

```text
prompt                    prompt_revision
─────                     ───────────────
id (PK)            1───N  id (PK)
description                prompt_id (FK → prompt.id, ON DELETE CASCADE)
slug                       revision_number
body                       description (snapshot)
deleted_at                 body (snapshot)
created_at                 change_note
updated_at                 created_at
                           created_by
```

- `prompt` always reflects the **current, definitive** content.
- Every `INSERT` or `UPDATE` of body/description appends a `prompt_revision` row.
- `revision_number` is monotonic per `prompt_id` (1, 2, 3, …). Enforced by `UNIQUE (prompt_id, revision_number)` + a small transaction in the application layer.
- Deleting a `prompt` cascades to its `prompt_revision` rows (the parent row carries the canonical state).

### 5.2 Slug rules (D3, D4)

- Regex: `^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$` (lowercase alnum + hyphen, 2–100 chars, no leading/trailing hyphen).
- Uniqueness enforced **only across active rows** via partial UNIQUE index:

  ```sql
  CREATE UNIQUE INDEX prompt_slug_active_uidx ON prompt(slug) WHERE deleted_at IS NULL;
  ```

- After soft-delete, the slug is free for reuse.

### 5.3 Description rules (R3)

- `description TEXT NOT NULL CHECK (length(description) BETWEEN 1 AND 280)`.
- 280 chars matches Twitter's classic limit; forces concise, human-meaningful labels.

### 5.4 Body rules (D2, R5)

- `body TEXT NOT NULL CHECK (length(body) BETWEEN 1 AND 524288)` — 512 KB cap. A typical markdown prompt is 2–20 KB; the cap leaves headroom for long system prompts.
- Markdown subset is not validated at the DB layer (the app trusts the input). Validation lives in the LLM driver.

### 5.5 Versioning rules (D1, D5)

- `revision_number` starts at `1` for the first revision.
- `INSERT` into `prompt` creates revision `1`.
- `UPDATE` of body **or** description creates the next revision; the old one is preserved verbatim (snapshot semantics — no shared row in `prompt_revision` is mutated).
- `DELETE` (soft-delete) does **not** create a revision; the `deleted_at` column captures it on the parent.
- Restore = INSERT a new revision with the historical body and description; `prompt.body` / `prompt.description` are updated to match; `prompt.updated_at` is bumped.

### 5.6 API surface (D6)

| Method | Path | Purpose | Status codes |
| --- | --- | --- | --- |
| `GET` | `/prompts` | List active prompts (pagination later). | 200 |
| `GET` | `/prompts/:slug` | Get the current prompt body. | 200, 404 |
| `POST` | `/prompts` | Create a new prompt (revision 1). | 201, 409 (slug taken), 400 (invalid) |
| `PATCH` | `/prompts/:slug` | Update body/description (creates next revision). | 200, 404, 409, 400 |
| `DELETE` | `/prompts/:slug` | Soft-delete. | 204, 404 |
| `GET` | `/prompts/:slug/revisions` | List revisions (newest first). | 200, 404 |
| `POST` | `/prompts/:slug/revisions/:n/restore` | Restore revision `n` as the new latest. | 200, 404, 409 |

Locked error envelope (mirrors `workspace`):

```json
{ "error": { "code": "PROMPT_SLUG_TAKEN", "message": "..." } }
```

Error codes (locked, RFC 2119 SHALL apply):

- `PROMPT_NOT_FOUND` (404)
- `PROMPT_SLUG_TAKEN` (409)
- `PROMPT_SLUG_INVALID` (400)
- `PROMPT_DESCRIPTION_INVALID` (400)
- `PROMPT_BODY_TOO_LARGE` (400)
- `PROMPT_REVISION_NOT_FOUND` (404)
- `PROMPT_DELETED` (410 — gone, no longer addressable by slug)

### 5.7 Observability

- Every handler logs request start/end with `slog` + `otelslog` (existing pattern).
- No new OTel spans in v1; the existing trace context propagates.
- No new metrics in v1.

---

## 6. Business rules

R1. A prompt's `slug` MUST be unique among active rows. (D3, D4)
R2. A prompt's `body` MUST NOT be empty and MUST NOT exceed 512 KB. (D2)
R3. A prompt's `description` MUST be 1–280 characters. (R3)
R4. Every modification (create or update) MUST produce exactly one `prompt_revision` row. (D1, D5)
R5. `revision_number` MUST be a strictly increasing positive integer, unique per `prompt_id`. (D5)
R6. Soft-delete MUST free the `slug` for reuse. (D4)
R7. Hard-delete (cascade) MUST remove all `prompt_revision` rows for that `prompt`. (D5)
R8. Restore MUST append a new revision; it MUST NOT mutate or delete prior revisions. (D5)
R9. The DB user for all DDL/DML on these tables MUST be `queen`. (Project norm.)
R10. The Go service MUST NOT log full prompt bodies (PII / IP risk). (Sensible default.)

---

## 7. Implications and impact

| Area | Impact |
| --- | --- |
| Schema | Two new tables; one new partial unique index; one new regular index on `prompt(updated_at DESC)`. No destructive change to existing tables. |
| Go code | New `domain/prompt.go` + `domain/prompt_revision.go`; new `infrastructure/postgres/prompt_repo.go`; new `application/prompt_service.go`; new `interfaces/http/prompt_handler.go`; minor wiring in `cmd/server/main.go`. |
| Tests | New domain tests, repo tests, service tests, handler tests (full TDD coverage). |
| Frontend | None in v1. API is read/written by future change. |
| Operations | One forward-only migration. No new env vars. No new infra. |
| Review workload | Forecast ~1,000–1,200 lines of new code across all layers. **Chained PRs recommended** (auto-forecast triggered; user to confirm at apply-pause gate). |

---

## 8. Edge cases (must be spec'd)

EC1. Create a prompt with a slug that already exists (active). → 409 `PROMPT_SLUG_TAKEN`.
EC2. Create a prompt with a slug that exists but is soft-deleted. → 201 (slug is free).
EC3. Update a prompt that is soft-deleted. → 410 `PROMPT_DELETED`.
EC4. Restore a revision whose parent prompt is soft-deleted. → 410 `PROMPT_DELETED`.
EC5. Body exactly 524 288 chars. → 201 (boundary inclusive).
EC6. Body 524 289 chars. → 400 `PROMPT_BODY_TOO_LARGE`.
EC7. Slug with leading or trailing hyphen. → 400 `PROMPT_SLUG_INVALID`.
EC8. Slug with uppercase letters. → 400 `PROMPT_SLUG_INVALID`.
EC9. Two concurrent `POST /prompts` with the same slug. → one succeeds (201), the other gets 409. Single-flight via DB unique constraint, not application locks.
EC10. Two concurrent `PATCH` on the same prompt. → Both succeed serially; both append a revision; the later one wins on `prompt.body`. `revision_number` is assigned under a row-level lock (the spec will lock this).

---

## 9. Non-goals (explicit)

- Multi-tenancy (no `organization_id` / `workspace_id` FK) — v1.1 candidate.
- Vector embeddings / semantic search — separate change (pgvector).
- Prompt templating / variable substitution engine — pure storage.
- Encryption at rest — Postgres-level TDE is a separate infra change.
- UI for editing — separate frontend change.
- Prompt A/B testing / experimentation hooks — v1.1+ candidate.
- Bulk import / export — v1.1.
- Rate limiting on `POST /prompts` — none in v1 (admin-only).

---

## 10. Rollback plan

Per `openspec/AGENTS.md`, the migration is forward-only with an empty `Down` block. To roll back operationally:

1. Stop new traffic to `/prompts/*` (drop the routes from the handler registry, or revert the wiring commit).
2. `DROP TABLE prompt_revision CASCADE;` (cascades to no other table — `prompt` is the parent).
3. `DROP TABLE prompt;`
4. Remove the Go code paths in a follow-up commit.
5. **No data migration back-out** is needed because the change is purely additive.

If the change has been live and a rollback is requested after data exists, the prompt rows must be exported as `COPY ... TO` before drop. This is documented as a runbook step but not enforced by the migration.

---

## 11. Spec/design/tasks mapping (preview)

- **`specs/prompts/spec.md`** — atomic Given/When/Then scenarios for each endpoint, each error code, each edge case in §8.
- **`design.md`** — DDL exactly as it will land in the migration file; sequence diagram for `POST /prompts/:slug/revisions/:n/restore`; error mapping table; ADR-007 for "two-table versioning vs columns on the same table".
- **`tasks.md`** — forecast + chained PR split (PR1 migration, PR2 domain+repo, PR3 application+handler+wiring) with TDD per task.

---

## 12. Open task-phase questions (will surface again at apply gate)

These do not block this proposal but must be answered before `sdd-apply`:

- **Q-A**: Confirm the chained-PR split (3 PRs) is acceptable vs single mega-PR (against budget) or different split.
- **Q-B**: Pagination for `GET /prompts` in v1 — offset/limit (simple) or keyset (forward-compatible)? **Recommended: simple offset/limit with `?limit=` default 50, max 200.**
- **Q-C**: Restore endpoint — should it be `POST /prompts/:slug/revisions/:n/restore` or `POST /prompts/:slug/restore` with body `{"revision_number": n}`? **Recommended: path-segment, RESTful.**
- **Q-D**: Authentication for `POST/PATCH/DELETE /prompts/*` — admin-only (no auth middleware for now) or require an `X-Admin` header? **Recommended: admin-only, no extra header in v1; will gate behind existing middleware when added.**

---

## 13. Status

- ✅ `explore.md` complete (industry research + project recon + risks).
- ✅ This `proposal.md` complete (locked decisions, business rules, edge cases).
- ⏭ Next: `specs/prompts/spec.md` (Given/When/Then for each scenario in §8).
