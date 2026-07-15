# Spec — `prompts` (delta / initial canonical)

> **Change**: `2026-07-15-prompt-storage-table`
> **Domain**: `prompts`
> **Strict TDD**: ACTIVE — every scenario is independently verifiable by a named test function (see `design.md` test map).
> **Inherits**: `proposal.md` of this change (decisions D1–D10); `openspec/config.yaml`; `openspec/AGENTS.md`.

This spec is the **initial canonical** for the `prompts` domain. No prior canonical exists. After `sdd-archive`, it moves into `openspec/specs/prompts/spec.md`.

---

## 1. Scope

This spec covers:

1. The `prompt` and `prompt_revision` tables (DDL invariants).
2. The `domain.Prompt`, `domain.PromptRevision`, and `domain.PromptRepository` contracts.
3. The HTTP API under `/prompts/*`.
4. The versioning model (two tables, snapshot semantics).
5. The soft-delete + slug-reuse semantics.
6. The error envelope and locked error codes.

Out of scope: vector embeddings, prompt templating, UI, multi-tenancy (see proposal §9).

---

## 2. Conventions

- RFC 2119 keywords: **MUST**, **SHALL**, **SHOULD**, **MAY**.
- Each scenario is **independently verifiable** (one test function per scenario).
- Scenarios are named `S-PR-<n>` for atomic, `S-PR-X<n>` for cross-cutting.
- Error envelope: `{"error":{"code":"…","message":"…"}}` (locked, matches existing handlers).
- All timestamps are `TIMESTAMPTZ` in UTC.

---

## 3. Storage invariants (DB layer)

### INV-1 — `prompt` row

> The `prompt` table MUST have the following columns and constraints. (DDL exact in `design.md` §3.)

- `id BIGSERIAL PRIMARY KEY`.
- `description TEXT NOT NULL` with `CHECK (length(description) BETWEEN 1 AND 280)`.
- `slug TEXT NOT NULL` with `CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$')`.
- `body TEXT NOT NULL` with `CHECK (length(body) BETWEEN 1 AND 524288)`.
- `deleted_at TIMESTAMPTZ NULL` (default `NULL`).
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- Table owner: `queen`.
- Partial unique index `prompt_slug_active_uidx` on `(slug) WHERE deleted_at IS NULL`.

### INV-2 — `prompt_revision` row

> The `prompt_revision` table MUST be append-only and MUST snapshot the body and description at the time of the change.

- `id BIGSERIAL PRIMARY KEY`.
- `prompt_id BIGINT NOT NULL REFERENCES prompt(id) ON DELETE CASCADE`.
- `revision_number INT NOT NULL` with `CHECK (revision_number > 0)`.
- `description TEXT NOT NULL`.
- `body TEXT NOT NULL` with `CHECK (length(body) BETWEEN 1 AND 524288)`.
- `change_note TEXT NULL`.
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- `created_by TEXT NULL`.
- `UNIQUE (prompt_id, revision_number)`.
- Index `prompt_revision_prompt_id_idx` on `(prompt_id, revision_number DESC)`.
- Table owner: `queen`.

### INV-3 — Cardinality

```
prompt  1 ── N  prompt_revision
```

- Each `prompt` MUST have **at least one** `prompt_revision` row (the initial creation writes revision `1`).
- Each `prompt_revision` MUST belong to exactly one `prompt` (FK `ON DELETE CASCADE`).

### INV-4 — Revision monotonicity

- `revision_number` MUST be a strictly increasing positive integer, unique per `prompt_id`.
- The application layer assigns `revision_number` as `(current_max + 1)` under a transaction that holds a row-level lock on `prompt.id`. (See `design.md` §5.2 for the locking strategy.)

---

## 4. Versioning scenarios

### S-PR-1 — Create produces revision 1

> **Given** an empty database
> **When** `POST /prompts` with valid body
> **Then** the response is `201 Created`
> **And** the response body has `id`, `slug`, `description`, `body`, `revision_number = 1`, `created_at`, `updated_at`
> **And** exactly one row exists in `prompt` with `slug = <slug>`
> **And** exactly one row exists in `prompt_revision` with `prompt_id = <id>`, `revision_number = 1`, `body = <body>`, `description = <description>`.

### S-PR-2 — Update produces the next revision

> **Given** an existing prompt with revision_number = 3
> **When** `PATCH /prompts/:slug` with a new body
> **Then** the response is `200 OK` with the updated prompt
> **And** `prompt_revision` has a new row with `revision_number = 4` and the new body
> **And** `prompt_revision` rows with `revision_number IN (1, 2, 3)` are unchanged.

### S-PR-3 — Update of description also creates a revision

> **Given** an existing prompt
> **When** `PATCH /prompts/:slug` with a new description only
> **Then** a new revision is created with the new description and the existing body.

### S-PR-4 — Restore appends a new revision (does not mutate history)

> **Given** a prompt with revisions `[1: "v1 body", 2: "v2 body", 3: "v3 body"]` and `prompt.body = "v3 body"`
> **When** `POST /prompts/:slug/revisions/1/restore`
> **Then** the response is `200 OK` with `prompt.body = "v1 body"`
> **And** `prompt_revision` has a new row with `revision_number = 4`, `body = "v1 body"`, `description = <description from rev 1>`, `change_note = "restored from revision 1"`
> **And** revisions 1, 2, 3 are unchanged.

### S-PR-5 — Restore is rejected if prompt is soft-deleted

> **Given** a soft-deleted prompt
> **When** `POST /prompts/:slug/revisions/:n/restore`
> **Then** the response is `410 Gone` with code `PROMPT_DELETED`.

---

## 5. Soft-delete scenarios

### S-PR-6 — Soft-delete frees the slug

> **Given** an active prompt with slug `welcome`
> **When** `DELETE /prompts/welcome`
> **Then** the response is `204 No Content`
> **And** `prompt.deleted_at IS NOT NULL`
> **And** `GET /prompts/welcome` returns `404`.

### S-PR-7 — Create with a soft-deleted slug is allowed

> **Given** a soft-deleted prompt with slug `welcome`
> **When** `POST /prompts` with slug `welcome`
> **Then** the response is `201 Created`
> **And** the new prompt is active.

### S-PR-8 — Update on a soft-deleted prompt is rejected

> **Given** a soft-deleted prompt with slug `welcome`
> **When** `PATCH /prompts/welcome`
> **Then** the response is `410 Gone` with code `PROMPT_DELETED`.

### S-PR-9 — List excludes soft-deleted

> **Given** two prompts: `welcome` (active) and `archived` (soft-deleted)
> **When** `GET /prompts`
> **Then** the response lists only `welcome`.

---

## 6. Slug and description validation

### S-PR-10 — Slug must match the regex

> **When** `POST /prompts` with slug `"Welcome"` (uppercase)
> **Then** the response is `400` with code `PROMPT_SLUG_INVALID`.

### S-PR-11 — Slug boundary: 2 chars minimum

> **When** `POST /prompts` with slug `"a"` (1 char)
> **Then** the response is `400` with code `PROMPT_SLUG_INVALID`.

### S-PR-12 — Slug boundary: 100 chars maximum

> **When** `POST /prompts` with a 100-char slug
> **Then** the response is `201` (boundary inclusive).

### S-PR-13 — Slug with leading hyphen is rejected

> **When** `POST /prompts` with slug `"-welcome"`
> **Then** the response is `400` with code `PROMPT_SLUG_INVALID`.

### S-PR-14 — Slug with trailing hyphen is rejected

> **When** `POST /prompts` with slug `"welcome-"`
> **Then** the response is `400` with code `PROMPT_SLUG_INVALID`.

### S-PR-15 — Description must be 1–280 chars

> **When** `POST /prompts` with `description = ""`
> **Then** the response is `400` with code `PROMPT_DESCRIPTION_INVALID`.

### S-PR-16 — Description at 280 chars is accepted

> **When** `POST /prompts` with a 280-char description
> **Then** the response is `201` (boundary inclusive).

---

## 7. Body validation

### S-PR-17 — Body at 512 KB is accepted

> **When** `POST /prompts` with a 524288-byte body
> **Then** the response is `201` (boundary inclusive).

### S-PR-18 — Body over 512 KB is rejected

> **When** `POST /prompts` with a 524289-byte body
> **Then** the response is `400` with code `PROMPT_BODY_TOO_LARGE`.

### S-PR-19 — Empty body is rejected

> **When** `POST /prompts` with `body = ""`
> **Then** the response is `400` with code `PROMPT_BODY_TOO_LARGE` (the `> 0` clause).

---

## 8. Concurrency scenarios

### S-PR-20 — Concurrent create with same slug: one wins

> **Given** no active prompt with slug `welcome`
> **When** two `POST /prompts` requests with the same slug arrive within microseconds
> **Then** exactly one response is `201 Created`
> **And** the other response is `409` with code `PROMPT_SLUG_TAKEN`
> **And** the DB has exactly one active prompt with that slug
> **And** each successful create wrote exactly one `prompt_revision` row.

### S-PR-21 — Concurrent update produces two revisions in order

> **Given** a prompt at revision 5
> **When** two `PATCH /prompts/:slug` requests arrive concurrently
> **Then** both responses are `200 OK`
> **And** `prompt_revision` has revisions 5 (existing), 6 (from one PATCH), 7 (from the other PATCH)
> **And** `prompt.body` matches the body of the second-PATCH-wins revision (deterministic: lock-then-update).
> **And** both `prompt.updated_at` are later than the original.

---

## 9. Read scenarios

### S-PR-22 — GET by slug returns current body

> **Given** a prompt with `body = "hello"`
> **When** `GET /prompts/welcome`
> **Then** the response is `200` with `body = "hello"`.

### S-PR-23 — GET by slug returns 404 when missing

> **When** `GET /prompts/nonexistent`
> **Then** the response is `404` with code `PROMPT_NOT_FOUND`.

### S-PR-24 — GET by slug returns 404 when soft-deleted

> **Given** a soft-deleted prompt with slug `welcome`
> **When** `GET /prompts/welcome`
> **Then** the response is `404` with code `PROMPT_NOT_FOUND`.

### S-PR-25 — List returns active prompts ordered by updated_at desc

> **Given** three prompts with `updated_at` at `t1`, `t2`, `t3` (`t3` newest)
> **When** `GET /prompts`
> **Then** the response lists them in order `[t3, t2, t1]`.

### S-PR-26 — Revisions are returned newest-first

> **Given** a prompt with 4 revisions
> **When** `GET /prompts/:slug/revisions`
> **Then** the response lists `[4, 3, 2, 1]`.

### S-PR-27 — Revisions endpoint returns 404 if prompt is missing or deleted

> **When** `GET /prompts/nonexistent/revisions`
> **Then** the response is `404`.

---

## 10. Cross-cutting scenarios

### S-PR-X1 — Concurrent restore + update is serialized

> **Given** a prompt at revision 5
> **When** `POST /prompts/:slug/revisions/3/restore` and `PATCH /prompts/:slug` arrive concurrently
> **Then** both complete successfully
> **And** `prompt_revision` has 6 or 7 new rows (one per logical operation), with monotonic numbering.

### S-PR-X2 — Soft-delete + restore round-trip

> **Given** an active prompt with 3 revisions
> **When** `DELETE /prompts/:slug` then `POST /prompts/:slug/revisions/2/restore` on a fresh, re-created prompt with the same slug
> **Then** the recreate succeeds (slug is free)
> **And** restore produces revision 4 with the historical body
> **And** the chain of revisions `[1, 2, 3]` is preserved on the original (deleted) prompt and `[4]` on the new one.

### S-PR-X3 — No PII leakage in logs

> **Given** a prompt with `body = "secret instructions"`
> **When** any handler logs the request
> **Then** the captured log output MUST NOT contain the literal `"secret instructions"` substring.
> **Verification**: `TestLogsDoNotLeakPromptBody` captures `slog` output and asserts the substring is absent.

### S-PR-X4 — Error envelope shape is locked

> **When** any handler returns an error
> **Then** the response body MUST match `{"error":{"code":"<UPPER_SNAKE>","message":"<human-readable>"}}`.
> **And** the `Content-Type` MUST be `application/json; charset=utf-8`.

### S-PR-X5 — Domain layer does not import pgx

> **Given** the package `domain`
> **When** the test runs `go list -deps -f '{{.ImportPath}}' ./src/domain`
> **Then** the output MUST NOT contain `github.com/jackc/pgx`.

---

## 11. Edge cases (mapped from proposal §8)

| EC | Scenario ref | Expected |
| --- | --- | --- |
| EC1 | S-PR-20 | 409 `PROMPT_SLUG_TAKEN` |
| EC2 | S-PR-7 | 201 |
| EC3 | S-PR-8 | 410 `PROMPT_DELETED` |
| EC4 | S-PR-5 | 410 `PROMPT_DELETED` |
| EC5 | S-PR-17 | 201 |
| EC6 | S-PR-18 | 400 `PROMPT_BODY_TOO_LARGE` |
| EC7 | S-PR-13 | 400 `PROMPT_SLUG_INVALID` |
| EC8 | S-PR-10 | 400 `PROMPT_SLUG_INVALID` |
| EC9 | S-PR-20 | 409 + DB invariant |
| EC10 | S-PR-21 | monotonic numbering, no lost update |

---

## 12. Test map (handoff to `design.md`)

For each scenario above, `design.md` MUST name one test function in the appropriate Go file:

- DB-layer invariants (INV-1..4) → `infrastructure/postgres/prompt_repo_test.go` (integration tag).
- Domain validation (S-PR-10..19, S-PR-22..24) → `domain/prompt_test.go`.
- Service / revision assignment (S-PR-1..5, S-PR-20, S-PR-21, S-PR-X1, S-PR-X2) → `application/prompt_service_test.go` (with pgx fixture).
- HTTP handlers (S-PR-6..9, S-PR-22..27, S-PR-X4) → `interfaces/http/prompt_handler_test.go`.
- Cross-cutting (S-PR-X3, S-PR-X5) → `application/prompt_service_test.go` + a `go list` shell test in the CI hook.

---

## 13. Status

- ✅ Scenarios enumerated; Given/When/Then written; cross-cutting covered.
- ⏭ Next: `design.md` (DDL exact, sequence diagrams, test map, ADRs).
