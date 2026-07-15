# Spec — Prompts

## Purpose

Define the acceptance criteria for the Prompts feature delivered by change `2026-07-15-prompt-storage-table`. Prompts are first-class, versioned, slug-addressable LLM prompt bodies stored in PostgreSQL with full audit (append-only `prompt_revision` history) and soft-delete semantics (the `slug` is freed for reuse after delete via the partial UNIQUE index `prompt_slug_active_uidx`).

This spec is the contract that `sdd-apply` implemented (4 chained commits, PR1 → PR2 → PR3 → PR4, landed as PR #48) and that `sdd-verify` checked against (83 tests passing on `-race` against compose-provisioned Postgres).

## Architecture (recap)

- Two tables: `prompt` (current definitive row) + `prompt_revision` (append-only history).
- Hexagonal Go layout: `domain/prompt.go` defines the `PromptRepository` and `PromptRevisionRepository` ports; `infrastructure/postgres/prompts/` implements them as pgx adapters; `application/prompt_service.go` orchestrates use cases; `interfaces/http/prompt_handler.go` exposes 7 HTTP endpoints.
- Concurrency gate: every write (Update / Restore / SoftDelete) acquires a row-level `FOR UPDATE` lock on `prompt.id` before reading the current row. Two goroutines on the same prompt serialize; revision numbers are monotonic.
- Wire codes are generic (project convention: `validation` → 400, `conflict` → 409, `not_found` → 404). The only feature-specific code is `prompt_deleted` → 410, surfaced by the new `*domain.GoneError` type (no existing type covered 410).

## Requirements

### R-PR-001 — Two tables, append-only history

The `prompt` table is the current definitive row; the `prompt_revision` table is an append-only journal of every change. Every `INSERT` into `prompt` writes revision `1`. Every `UPDATE` of body or description appends a `prompt_revision` row whose `revision_number` is `MAX(revision_number) + 1` for that prompt.

#### Scenarios

S-PR-1 — Create produces revision 1.

> **Given** an empty database
> **When** `POST /prompts` with valid body
> **Then** the response is `201 Created`
> **And** the response body has `id`, `slug`, `description`, `body`, `revision_number = 1`, `created_at`, `updated_at`
> **And** exactly one row exists in `prompt` with `slug = <slug>`
> **And** exactly one row exists in `prompt_revision` with `prompt_id = <id>`, `revision_number = 1`, `body = <body>`, `description = <description>`.

S-PR-2 — Update produces the next revision.

> **Given** an existing prompt with revision_number = 3
> **When** `PATCH /prompts/:slug` with a new body
> **Then** the response is `200 OK` with the updated prompt
> **And** `prompt_revision` has a new row with `revision_number = 4` and the new body
> **And** `prompt_revision` rows with `revision_number IN (1, 2, 3)` are unchanged.

S-PR-3 — Update of description also creates a revision.

> **Given** an existing prompt
> **When** `PATCH /prompts/:slug` with a new description only
> **Then** a new revision is created with the new description and the existing body.

S-PR-4 — Restore appends a new revision (does not mutate history).

> **Given** a prompt with revisions `[1: "v1 body", 2: "v2 body", 3: "v3 body"]` and `prompt.body = "v3 body"`
> **When** `POST /prompts/:slug/revisions/1/restore`
> **Then** the response is `200 OK` with `prompt.body = "v1 body"`
> **And** `prompt_revision` has a new row with `revision_number = 4`, `body = "v1 body"`, `description = <description from rev 1>`, `change_note = "restored from revision 1"`
> **And** revisions 1, 2, 3 are unchanged.

S-PR-5 — Restore is rejected if prompt is soft-deleted.

> **Given** a soft-deleted prompt
> **When** `POST /prompts/:slug/revisions/:n/restore`
> **Then** the response is `410 Gone` with code `prompt_deleted`.

### R-PR-002 — Soft-delete + slug reuse

The `deleted_at TIMESTAMPTZ` column marks a prompt as soft-deleted. A partial UNIQUE index on `slug WHERE deleted_at IS NULL` keeps slug uniqueness scoped to live rows; soft-deleted slugs are reusable.

#### Scenarios

S-PR-6 — Soft-delete frees the slug.

> **Given** an active prompt with slug `welcome`
> **When** `DELETE /prompts/welcome`
> **Then** the response is `204 No Content`
> **And** `prompt.deleted_at IS NOT NULL`
> **And** `GET /prompts/welcome` returns `404`.

S-PR-7 — Create with a soft-deleted slug is allowed.

> **Given** a soft-deleted prompt with slug `welcome`
> **When** `POST /prompts` with slug `welcome`
> **Then** the response is `201 Created`
> **And** the new prompt is active.

S-PR-8 — Update on a soft-deleted prompt is rejected.

> **Given** a soft-deleted prompt with slug `welcome`
> **When** `PATCH /prompts/welcome`
> **Then** the response is `410 Gone` with code `prompt_deleted`.

S-PR-9 — List excludes soft-deleted.

> **Given** two prompts: `welcome` (active) and `archived` (soft-deleted)
> **When** `GET /prompts`
> **Then** the response lists only `welcome`.

### R-PR-003 — Validation

#### Scenarios

S-PR-10 — Slug must match the regex.

> **When** `POST /prompts` with slug `"Welcome"` (uppercase)
> **Then** the response is `400` with code `validation`, field `slug`.

S-PR-11 — Slug boundary: 2 chars minimum.

S-PR-12 — Slug boundary: 100 chars maximum (inclusive).

S-PR-13 — Slug with leading hyphen is rejected.

S-PR-14 — Slug with trailing hyphen is rejected.

S-PR-15 — Description must be 1–280 chars.

S-PR-16 — Description at 280 chars is accepted (boundary inclusive).

S-PR-17 — Body at 524288 bytes is accepted (boundary inclusive).

S-PR-18 — Body over 524288 bytes is rejected with code `validation`, field `body`.

S-PR-19 — Empty body is rejected with code `validation`, field `body`.

### R-PR-004 — Concurrency

#### Scenarios

S-PR-20 — Concurrent create with same slug: one wins.

> **Given** no active prompt with slug `welcome`
> **When** two `POST /prompts` requests with the same slug arrive within microseconds
> **Then** exactly one response is `201 Created`
> **And** the other response is `409` with code `conflict`
> **And** the DB has exactly one active prompt with that slug
> **And** each successful create wrote exactly one `prompt_revision` row.

S-PR-21 — Concurrent update produces two revisions in order.

> **Given** a prompt at revision 5
> **When** two `PATCH /prompts/:slug` requests arrive concurrently
> **Then** both responses are `200 OK`
> **And** `prompt_revision` has revisions 5 (existing), 6 (from one PATCH), 7 (from the other PATCH)
> **And** `prompt.body` matches the body of the second-PATCH-wins revision
> **And** both `prompt.updated_at` are later than the original.

### R-PR-005 — Reads

#### Scenarios

S-PR-22 — GET by slug returns current body.

S-PR-23 — GET by slug returns 404 when missing.

S-PR-24 — GET by slug returns 404 when soft-deleted.

S-PR-25 — List returns active prompts ordered by updated_at desc.

S-PR-26 — Revisions are returned newest-first.

S-PR-27 — Revisions endpoint returns 404 if prompt is missing or deleted.

### R-PR-006 — Cross-cutting

#### Scenarios

S-PR-X1 — Concurrent restore + update is serialized; both succeed; revision numbers are monotonic.

S-PR-X2 — Soft-delete + recreate preserves revision chains (the chain `[1, 2, 3]` stays on the original deleted prompt; the new prompt gets its own revision 1).

S-PR-X3 — No PII leakage in logs: the request body and the prompt body MUST NOT appear in any log line.

S-PR-X4 — Error envelope shape is locked: `{"error":{"code":"…","message":"…"}}` with `Content-Type: application/json; charset=utf-8`.

S-PR-X5 — Domain layer MUST NOT import `github.com/jackc/pgx` (enforced by `TestDomainLayer_DoesNotImportPgx`).

### R-PR-007 — HTTP API surface

| Method | Path | Purpose | Status codes |
| --- | --- | --- | --- |
| GET | `/prompts` | List active prompts (default `limit=50`, hard cap `200`) | 200 |
| GET | `/prompts/:slug` | Get current prompt body | 200, 404 |
| POST | `/prompts` | Create a new prompt (revision 1) | 201, 409, 400 |
| PATCH | `/prompts/:slug` | Update body/description (creates next revision) | 200, 404, 409, 410, 400 |
| DELETE | `/prompts/:slug` | Soft-delete (idempotent) | 204, 404 |
| GET | `/prompts/:slug/revisions` | List revisions (newest first) | 200, 404 |
| POST | `/prompts/:slug/revisions/:n/restore` | Restore revision `n` as the new latest | 200, 404, 410 |

### R-PR-008 — Locked error vocabulary

| Wire code | HTTP | Triggered by |
| --- | --- | --- |
| `validation` | 400 | `*domain.ValidationError` (slug regex / description length / body length) |
| `not_found` | 404 | `*domain.NotFoundError` (slug missing or soft-deleted; revision missing) |
| `conflict` | 409 | `*domain.ConflictError` (slug collision on partial UNIQUE index) |
| `prompt_deleted` | 410 | `*domain.GoneError` (UPDATE / RESTORE on a soft-deleted prompt) |
| `server` | 500 | `*domain.InternalError` (catch-all) |

## Edge cases (from proposal §8)

| EC | Scenario ref | Expected |
| --- | --- | --- |
| EC1 | S-PR-20 | 409 `conflict` |
| EC2 | S-PR-7 | 201 |
| EC3 | S-PR-8 | 410 `prompt_deleted` |
| EC4 | S-PR-5 | 410 `prompt_deleted` |
| EC5 | S-PR-17 | 201 |
| EC6 | S-PR-18 | 400 `validation` field `body` |
| EC7 | S-PR-13 | 400 `validation` field `slug` |
| EC8 | S-PR-10 | 400 `validation` field `slug` |
| EC9 | S-PR-20 | 409 + DB invariant |
| EC10 | S-PR-21 | monotonic numbering, no lost update |

## Out of scope (deferred to later changes)

- Vector embeddings of prompts (pgvector)
- Prompt templating / variable substitution engine
- UI surface for editing prompts
- Multi-tenant scoping (`organization_id` FK)
- Prompt history retention sweeper
- Encryption at rest of body content
- Bulk import / export
- A/B testing hooks
- Rate limiting on write endpoints

## Implementation reference

- Migration: `backend/database_administrator/src/migration/sql/20260715120000_prompts.sql`
- Domain: `backend/database_administrator/src/domain/prompt.go`
- Repo: `backend/database_administrator/src/infrastructure/postgres/prompts/prompt_repo.go` + `prompt_revision_repo.go`
- Service: `backend/database_administrator/src/application/prompt_service.go`
- Handler: `backend/database_administrator/src/interfaces/http/prompt_handler.go`
- Wiring: `backend/database_administrator/src/cmd/server/main.go`
- Tests: 83 named test functions across `domain/`, `infrastructure/postgres/prompts/`, `application/`, `interfaces/http/`
