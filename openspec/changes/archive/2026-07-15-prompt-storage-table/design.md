# Design — `2026-07-15-prompt-storage-table`

> **Change**: `2026-07-15-prompt-storage-table`
> **Strict TDD**: ACTIVE — every Go file lands after its test lands (RED → GREEN → REFACTOR).
> **Inherits**: `proposal.md` (D1–D10), `spec.md` (S-PR-1..27, S-PR-X1..X5). All design decisions cite the proposing spec scenario or proposal decision.

---

## 0. Test map (concrete, 1:1 with spec)

The design test map is the executable manifest. Each spec scenario gets exactly one named test function. Tests are listed in the order tasks in `tasks.md` will be implemented.

| Spec | Test function | File | Layer |
| --- | --- | --- | --- |
| INV-1 | `TestPromptRepo_SchemaMatchesSpec` (build tag `integration`) | `infrastructure/postgres/prompt_repo_test.go` | DB |
| INV-2 | `TestPromptRevisionRepo_SchemaMatchesSpec` (build tag `integration`) | `infrastructure/postgres/prompt_revision_repo_test.go` | DB |
| INV-3 | `TestPromptRepo_CascadeDeleteRemovesRevisions` (integration) | `infrastructure/postgres/prompt_revision_repo_test.go` | DB |
| INV-4 | `TestPromptRepo_RevisionNumbersAreMonotonic` (integration) | `infrastructure/postgres/prompt_repo_test.go` | DB |
| S-PR-1 | `TestPromptService_Create_WritesRevisionOne` | `application/prompt_service_test.go` | service |
| S-PR-2 | `TestPromptService_Update_AppendsNextRevision` | `application/prompt_service_test.go` | service |
| S-PR-3 | `TestPromptService_UpdateDescriptionOnly_AppendsRevision` | `application/prompt_service_test.go` | service |
| S-PR-4 | `TestPromptService_Restore_AppendsNewRevisionWithHistoricalBody` | `application/prompt_service_test.go` | service |
| S-PR-5 | `TestPromptService_RestoreOnDeletedPrompt_Returns410` | `application/prompt_service_test.go` | service |
| S-PR-6 | `TestPromptHandler_Delete_SoftDeleteFreesSlug` | `interfaces/http/prompt_handler_test.go` | handler |
| S-PR-7 | `TestPromptService_Create_AllowsSoftDeletedSlug` | `application/prompt_service_test.go` | service |
| S-PR-8 | `TestPromptHandler_UpdateDeleted_Returns410` | `interfaces/http/prompt_handler_test.go` | handler |
| S-PR-9 | `TestPromptHandler_List_ExcludesDeleted` | `interfaces/http/prompt_handler_test.go` | handler |
| S-PR-10..14 | `TestPrompt_ValidateSlug_*` (table-driven) | `domain/prompt_test.go` | domain |
| S-PR-15,16 | `TestPrompt_ValidateDescription_*` | `domain/prompt_test.go` | domain |
| S-PR-17..19 | `TestPrompt_ValidateBody_*` | `domain/prompt_test.go` | domain |
| S-PR-20 | `TestPromptService_ConcurrentCreate_OneSucceedsOneConflicts` | `application/prompt_service_test.go` | service (race) |
| S-PR-21 | `TestPromptService_ConcurrentUpdate_ProducesMonotonicRevisions` | `application/prompt_service_test.go` | service (race) |
| S-PR-22..24 | `TestPromptHandler_GetBySlug_*` | `interfaces/http/prompt_handler_test.go` | handler |
| S-PR-25 | `TestPromptHandler_List_OrderByUpdatedAtDesc` | `interfaces/http/prompt_handler_test.go` | handler |
| S-PR-26 | `TestPromptHandler_ListRevisions_NewestFirst` | `interfaces/http/prompt_handler_test.go` | handler |
| S-PR-27 | `TestPromptHandler_ListRevisionsOnMissingPrompt_Returns404` | `interfaces/http/prompt_handler_test.go` | handler |
| S-PR-X1 | `TestPromptService_ConcurrentRestoreAndUpdate_NoLostUpdate` | `application/prompt_service_test.go` | service (race) |
| S-PR-X2 | `TestPromptService_DeleteThenRecreate_PreservesRevisionChains` | `application/prompt_service_test.go` | service |
| S-PR-X3 | `TestPromptHandler_NoPIITokenInLogs` | `interfaces/http/prompt_handler_test.go` | handler (capture slog) |
| S-PR-X4 | `TestPromptHandler_ErrorEnvelopeShape` | `interfaces/http/prompt_handler_test.go` | handler |
| S-PR-X5 | `TestDomainLayer_DoesNotImportPgx` (shell + `go list`) | `domain/imports_test.go` | shell + go |

**Total test functions: ~30** (exact count will lock once tasks land).

---

## 1. Architecture (no new cross-service seams)

This change is entirely inside the existing `database_administrator` service. No new container, no new OTel spans, no new env vars.

```text
HTTP layer (interfaces/http/prompt_handler.go)
        │
        ▼
Application (application/prompt_service.go)
        │
        ▼
Domain port (domain.PromptRepository, domain.PromptRevisionRepository)
        │
        ▼
Postgres adapter (infrastructure/postgres/prompt_repo.go,
                  infrastructure/postgres/prompt_revision_repo.go)
        │
        ▼
prompt + prompt_revision tables (migration/sql/<ts>_prompts.sql)
```

Two distinct repo structs (not one) because their concerns differ:

- `PromptRepo` — `Insert`, `SelectBySlug`, `SelectByID`, `SelectList` (active only), `UpdateBody` (current row), `SoftDelete` (sets `deleted_at`), and the **next-revision lock** helper.
- `PromptRevisionRepo` — `Insert`, `SelectLatestForPrompt`, `SelectByPromptAndNumber`, `SelectListByPrompt`.

The service composes both. The service holds the revision-assignment logic under a transaction.

---

## 2. Sequence diagrams

### 2.1 `POST /prompts` — create + revision 1

```text
Client                Handler             PromptService        PromptRepo         PromptRevisionRepo       DB
  │                     │                       │                  │                    │                  │
  │ POST /prompts       │                       │                  │                    │                  │
  │ (slug, desc, body)  │                       │                  │                    │                  │
  ├────────────────────▶│                       │                  │                    │                  │
  │                     │ CreatePrompt(req)     │                  │                    │                  │
  │                     ├──────────────────────▶│                  │                    │                  │
  │                     │                       │ ValidateSlug(s)  │                    │                  │
  │                     │                       │ ValidateDesc(d)  │                    │                  │
  │                     │                       │ ValidateBody(b)  │                    │                  │
  │                     │                       │ INSERT prompt+rev1 in TX              │                  │
  │                     │                       ├─────────────────▶│───────────────────▶│                  │
  │                     │                       │                  │  BEGIN             │                  │
  │                     │                       │                  │  INSERT prompt     │                  │
  │                     │                       │                  ├───────────────────▶│─────────────────▶│
  │                     │                       │                  │  INSERT rev 1      │                  │
  │                     │                       │                  │  COMMIT            │                  │
  │                     │                       │◀─────────────────┤◀───────────────────┤                  │
  │                     │ ◀─────────────────────┤                  │                    │                  │
  │ 201 Created         │                       │                  │                    │                  │
  │ {id, slug, ...,     │                       │                  │                    │                  │
  │  revision_number=1} │                       │                  │                    │                  │
  ◀─────────────────────┤                       │                  │                    │                  │
```

On unique violation (SQLSTATE 23505 on `prompt_slug_active_uidx`):

```text
Service
  │   pgconn.PgError code 23505
  │   ─▶ translate via errors.As → *domain.ConflictError(Code: PROMPT_SLUG_TAKEN)
  │   ─▶ Handler maps → 409 with envelope
```

### 2.2 `POST /prompts/:slug/revisions/:n/restore`

```text
Handler          PromptService        PromptRepo              PromptRevisionRepo                  DB
  │                  │                    │                          │                              │
  │ Restore(slug,n)  │                    │                          │                              │
  ├─────────────────▶│                    │                          │                              │
  │                  │ BEGIN TX           │                          │                              │
  │                  │ Select prompt by slug (FOR UPDATE)            │                              │
  │                  ├───────────────────▶│                          │                              │
  │                  │◀──── prompt ───────┤                          │                              │
  │                  │ If deleted → 410                              │                              │
  │                  │ Select revision n (by prompt_id, n)           │                              │
  │                  ├───────────────────┼─────────────────────────▶│                              │
  │                  │◀─── revision ─────┼──────────────────────────┤                              │
  │                  │ Assign next revision number = (MAX+1)         │                              │
  │                  │ INSERT new revision with historical body     │                              │
  │                  ├───────────────────┼─────────────────────────▶│                              │
  │                  │ UPDATE prompt SET body=, desc=, updated_at=now│                              │
  │                  ├───────────────────▶│                          │                              │
  │                  │ COMMIT             │                          │                              │
  │                  │◀───────────────────┤                          │                              │
  │ 200 OK           │                    │                          │                              │
  ◀──────────────────┤                    │                          │                              │
```

The `FOR UPDATE` row lock on `prompt.id` is the **single concurrency gate** for:

- Update (`PATCH`),
- Restore (`POST …/restore`),
- Soft-delete (`DELETE`).

All three MUST acquire the lock before reading the current row.

---

## 3. DDL — the migration

File: `backend/database_administrator/src/migration/sql/20260715120000_prompts.sql`

```sql
-- +goose Up
-- +goose StatementBegin
--
-- 2026-07-15-prompt-storage-table
-- Lifts proposal D1, D2, D3, D4, D5 and spec INV-1..4.
--
-- Two tables: `prompt` (current definitive row) + `prompt_revision`
-- (append-only history). Soft-delete on `prompt.deleted_at`. Slug uniqueness
-- is partial over active rows so the slug can be reused after delete.
--
CREATE TABLE IF NOT EXISTS prompt (
    id           BIGSERIAL    PRIMARY KEY,
    description  TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 280),
    slug         TEXT         NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$'),
    body         TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
    deleted_at   TIMESTAMPTZ  NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

ALTER TABLE prompt OWNER TO queen;

CREATE TABLE IF NOT EXISTS prompt_revision (
    id              BIGSERIAL    PRIMARY KEY,
    prompt_id       BIGINT       NOT NULL REFERENCES prompt(id) ON DELETE CASCADE,
    revision_number INT          NOT NULL CHECK (revision_number > 0),
    description     TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 280),
    body            TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
    change_note     TEXT         NULL,
    created_by      TEXT         NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT prompt_revision_unique UNIQUE (prompt_id, revision_number)
);

ALTER TABLE prompt_revision OWNER TO queen;

COMMENT ON TABLE prompt IS
    'Current definitive row of an LLM prompt. Always reflects the latest version. Soft-delete via deleted_at reuses the slug.';

COMMENT ON COLUMN prompt.body IS
    'Markdown prompt body (utf-8 TEXT). Validated 1..524288 bytes by CHECK constraint.';

COMMENT ON COLUMN prompt.slug IS
    'URL-friendly slug. Format: ^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$. Unique among active (non-deleted) rows.';

COMMENT ON TABLE prompt_revision IS
    'Append-only history of changes to `prompt`. INSERT-only; never UPDATE, never DELETE except via CASCADE when the parent prompt is hard-deleted.';

COMMENT ON COLUMN prompt_revision.revision_number IS
    'Monotonic per prompt_id. Strictly increasing positive integer. Assigned by the application under a FOR UPDATE row lock on the parent prompt.';

-- Slug uniqueness is scoped to active rows.
CREATE UNIQUE INDEX IF NOT EXISTS prompt_slug_active_uidx
    ON prompt(slug) WHERE deleted_at IS NULL;

-- List ordering by recency (used by GET /prompts).
CREATE INDEX IF NOT EXISTS prompt_updated_at_idx
    ON prompt(updated_at DESC);

-- Revision lookup supports: latest first, exact (prompt_id, n).
CREATE INDEX IF NOT EXISTS prompt_revision_prompt_id_idx
    ON prompt_revision(prompt_id, revision_number DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Forward-only migration per openspec/AGENTS.md.
-- +goose StatementEnd
```

### 3.1 Idempotency notes

- All `CREATE` are `IF NOT EXISTS` (project norm).
- The migration must be safe to **fail and retry** on a fresh DB; the CHECK constraints are part of the table defs and cannot be conditionally added.
- For the second migration wave (if we ever add an index or column), use `DO $$ … $$;` blocks per `sync_job.sql` precedent.

---

## 4. Domain — types and validation

`backend/database_administrator/src/domain/prompt.go`:

```go
package domain

import (
    "errors"
    "fmt"
    "regexp"
    "time"
    "unicode/utf8"
)

const (
    MaxDescriptionLen = 280
    MaxBodyLen        = 524288
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$`)

type Prompt struct {
    ID          int64
    Description string
    Slug        string
    Body        string
    DeletedAt   *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type PromptRevision struct {
    ID             int64
    PromptID       int64
    RevisionNumber int
    Description    string
    Body           string
    ChangeNote     *string
    CreatedBy      *string
    CreatedAt      time.Time
}

var (
    ErrSlugInvalid      = errors.New("prompt slug invalid")
    ErrDescriptionInvalid = errors.New("prompt description invalid")
    ErrBodyTooLarge     = errors.New("prompt body too large")
)

func ValidateSlug(slug string) error {
    if !slugRe.MatchString(slug) {
        return fmt.Errorf("%w: slug must match ^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$: %q", ErrSlugInvalid, slug)
    }
    return nil
}

func ValidateDescription(desc string) error {
    n := utf8.RuneCountInString(desc)
    if n < 1 || n > MaxDescriptionLen {
        return fmt.Errorf("%w: description must be 1..%d chars: got %d", ErrDescriptionInvalid, MaxDescriptionLen, n)
    }
    return nil
}

func ValidateBody(body string) error {
    n := utf8.RuneCountInString(body)
    if n < 1 || n > MaxBodyLen {
        return fmt.Errorf("%w: body must be 1..%d chars: got %d", ErrBodyTooLarge, MaxBodyLen, n)
    }
    return nil
}
```

### 4.1 Validation rules table

| Field | Rule | Source |
| --- | --- | --- |
| `slug` | Regex `^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$` | spec S-PR-10..14 |
| `description` | Rune count 1..280 | spec S-PR-15,16 |
| `body` | Rune count 1..524288 | spec S-PR-17..19 |

### 4.2 Error vocabulary (locked, adjusted to project pattern)

> **Adjustment from proposal §5.6**: the project's existing handler (see `interfaces/http/workspace_handler.go`) maps error **types** to **generic** wire codes (`validation`, `conflict`, `not_found`), not feature-specific codes. Reusing `*ValidationError` / `*ConflictError` / `*NotFoundError` from `organization.go` keeps consistency with the codebase. The only NEW wire code is `prompt_deleted` (HTTP 410), because the existing vocabulary has no 410 case.

Codes at the wire envelope:

| Type | Wire code | HTTP | When |
| --- | --- | --- | --- |
| `*ValidationError` | `validation` | 400 | `ValidateSlug` / `ValidateDescription` / `ValidateBody` fail; `Fields` map carries the per-field message |
| `*ConflictError` | `conflict` | 409 | INSERT races with another active row; SQLSTATE 23505 on `prompt_slug_active_uidx` |
| `*NotFoundError` | `not_found` | 404 | `SelectBySlug` returns no rows, or all rows have `deleted_at IS NOT NULL`; `SelectByPromptAndNumber` returns no rows |
| `*GoneError` (new) | `prompt_deleted` | 410 | Update or restore on a soft-deleted prompt |

The `*GoneError` type is new in `domain/prompt.go` because no existing type covers 410. Reusing `*NotFoundError` would force the handler to infer 410 from message text, which is brittle. The dedicated type keeps the HTTP mapping deterministic and testable with `errors.As`.

Each error implements the `AppError` interface (declared in `organization.go`):

```go
type AppError interface {
    error
    Code() string
}
```

The handler maps `Code()` to the HTTP status; the body uses the locked envelope `{"error":{"code":"…","message":"…"}}`. The prompt feature's `*GoneError.Code()` returns the new `CodePromptDeleted` constant (`"prompt_deleted"`).

### 4.3 Port (Go interface)

```go
type PromptRepository interface {
    Insert(ctx context.Context, db sqlExecutor, p *Prompt) error
    SelectBySlug(ctx context.Context, db sqlExecutor, slug string) (*Prompt, error)
    SelectByID(ctx context.Context, db sqlExecutor, id int64) (*Prompt, error)
    SelectList(ctx context.Context, db sqlExecutor, limit, offset int) ([]*Prompt, error)
    UpdateBody(ctx context.Context, db sqlExecutor, id int64, body, description string, updatedAt time.Time) error
    SoftDelete(ctx context.Context, db sqlExecutor, id int64, deletedAt time.Time) error
    LockAndLoad(ctx context.Context, db sqlExecutor, id int64) (*Prompt, error)
    MaxRevisionNumber(ctx context.Context, db sqlExecutor, promptID int64) (int, error) // used inside the locked TX
}

type PromptRevisionRepository interface {
    Insert(ctx context.Context, db sqlExecutor, r *PromptRevision) error
    SelectLatestForPrompt(ctx context.Context, db sqlExecutor, promptID int64) (*PromptRevision, error)
    SelectByPromptAndNumber(ctx context.Context, db sqlExecutor, promptID int64, n int) (*PromptRevision, error)
    SelectListByPrompt(ctx context.Context, db sqlExecutor, promptID int64) ([]*PromptRevision, error)
}

type sqlExecutor interface {
    ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
}
```

`sqlExecutor` lets the service pass either a `*sql.DB` or a `*sql.Tx` so a single multi-statement operation is atomic.

---

## 5. Service — orchestration rules

`backend/database_administrator/src/application/prompt_service.go`:

### 5.1 `Create(ctx, in CreatePromptInput) (*Prompt, *PromptRevision, error)`

```text
1. Validate in.ValidateSlug, ValidateDescription, ValidateBody. ─▶ ErrXxxInvalid on failure.
2. tx, _ := db.BeginTx(ctx, nil)
3. defer tx.Rollback()
4. INSERT INTO prompt (...) RETURNING id, created_at, updated_at.
5. INSERT INTO prompt_revision (prompt_id=id, revision_number=1, ...).
6. tx.Commit().
7. Return the hydrated Prompt + PromptRevision(rev=1).
```

If step 4 hits SQLSTATE 23505 (slug taken), translate to `*domain.ConflictError(Code: PROMPT_SLUG_TAKEN)`.

### 5.2 `Update(ctx, slug, in UpdatePromptInput) (*Prompt, *PromptRevision, error)`

```text
1. tx, _ := db.BeginTx(ctx, nil).
2. defer tx.Rollback().
3. prompt := promptRepo.LockAndLoad(ctx, tx, id-by-slug).
   - if not found → PROMPT_NOT_FOUND.
   - if DeletedAt != nil → PROMPT_DELETED.
4. newRev := promptRevRepo.MaxRevisionNumber(ctx, tx, id) + 1.
5. promptRevRepo.Insert(ctx, tx, &PromptRevision{PromptID: id, RevisionNumber: newRev, ...}).
6. promptRepo.UpdateBody(ctx, tx, id, in.Body, in.Description, now()).
7. tx.Commit().
8. Re-read prompt, return.
```

This is the **single concurrency gate**. Two concurrent `Update` calls serialize on step 3's `FOR UPDATE` lock and produce sequential revisions (S-PR-21).

### 5.3 `Restore(ctx, slug, n) (*Prompt, *PromptRevision, error)`

Same shape as `Update` but reads the historical revision and inserts it as `newRev` (S-PR-4). `change_note = "restored from revision <n>"`. `created_by` left NULL for v1.

### 5.4 `SoftDelete(ctx, slug) error`

`LockAndLoad` → check `DeletedAt != nil` (idempotent if already deleted? **decide**: returning 204 even if already deleted is the conventional idempotent semantic; spec S-PR-6 says 204. We WILL return 204 even on a second delete to keep idempotency, but the response body is empty). Mark `deleted_at = now()`.

### 5.5 `List(ctx, limit, offset) ([]*Prompt, error)`

`promptRepo.SelectList(ctx, db, limit, offset)` filtering `deleted_at IS NULL`, ordered by `updated_at DESC`. Defaults: `limit=50, offset=0`. **Hard cap: `limit <= 200`**.

### 5.6 `ListRevisions(ctx, slug) ([]*PromptRevision, error)`

`promptRepo.SelectBySlug` (must exist and not be deleted). Then `promptRevRepo.SelectListByPrompt`.

---

## 6. Handler

`backend/database_administrator/src/interfaces/http/prompt_handler.go`:

| HTTP method + path | Calls | Maps errors → |
| --- | --- | --- |
| `GET /prompts` | `service.List(ctx, parseLimit(query))` | — |
| `GET /prompts/:slug` | `service.GetBySlug(ctx, slug)` | `PROMPT_NOT_FOUND → 404` |
| `POST /prompts` | `service.Create(ctx, body)` | `PROMPT_SLUG_TAKEN → 409`, `PROMPT_*_INVALID → 400` |
| `PATCH /prompts/:slug` | `service.Update(ctx, slug, body)` | `PROMPT_NOT_FOUND → 404`, `PROMPT_DELETED → 410`, validation `→ 400`, `PROMPT_SLUG_TAKEN → 409` |
| `DELETE /prompts/:slug` | `service.SoftDelete(ctx, slug)` | `PROMPT_NOT_FOUND → 404`, idempotent on already-deleted |
| `GET /prompts/:slug/revisions` | `service.ListRevisions(ctx, slug)` | `PROMPT_NOT_FOUND → 404` |
| `POST /prompts/:slug/revisions/:n/restore` | `service.Restore(ctx, slug, n)` | `PROMPT_NOT_FOUND → 404`, `PROMPT_REVISION_NOT_FOUND → 404`, `PROMPT_DELETED → 410` |

Request body for `POST /prompts`:

```json
{ "slug": "welcome-email", "description": "...", "body": "..." }
```

Request body for `PATCH /prompts/:slug`:

```json
{ "description": "...optional...", "body": "...optional..." }
```

At least one of `description` or `body` MUST be present; otherwise `400 PROMPT_DESCRIPTION_INVALID` (or a generic `PROMPT_BODY_TOO_LARGE` — to lock in code review).

---

## 7. Wiring

`backend/database_administrator/src/cmd/server/main.go`: add one block after the existing repos:

```go
promptRepo := postgres.NewPromptRepo(db)
promptRevRepo := postgres.NewPromptRevisionRepo(db)
promptSvc := application.NewPromptService(promptRepo, promptRevRepo, slog.Default())
promptHandler := interfaces.NewPromptHandler(promptSvc, slog.Default())

e.GET("/prompts", promptHandler.List, mw...)
e.POST("/prompts", promptHandler.Create, mw...)
e.GET("/prompts/:slug", promptHandler.GetBySlug, mw...)
e.PATCH("/prompts/:slug", promptHandler.Update, mw...)
e.DELETE("/prompts/:slug", promptHandler.Delete, mw...)
e.GET("/prompts/:slug/revisions", promptHandler.ListRevisions, mw...)
e.POST("/prompts/:slug/revisions/:n/restore", promptHandler.Restore, mw...)
```

No new middleware; reuse the existing auth middleware via `mw...` spread (per project norm).

---

## 8. ADRs (architecture decision records)

### ADR-PMT-001 — `prompt.body` is `TEXT`, not `BYTEA` or `JSONB`

**Status**: Accepted (user-confirmed in proposal Q1).

**Context**: The user explicitly asked for a "binary column to store the markdown prompt". "Binary" in PostgreSQL terminology means `BYTEA`. However, the industry standard (LangChain `langchain-postgres`, LlamaIndex `PostgresChatStore`, the PostgreSQL wiki, Oracle Developers 2026-07-09 guidance) is `TEXT` for any human-readable markdown content. The user confirmed `TEXT` after seeing the tradeoff.

**Decision**: Use `TEXT` with utf-8. Validate 1..524288 chars via `CHECK`. No compression or encryption in v1.

**Consequences**:

- (+) Direct psql inspection; markdown is human-readable in dumps.
- (+) Future-proofs for `tsvector` full-text search and `LIKE`/`regex` if the LLM driver adds it.
- (+) DB-level utf-8 validation via `CHECK` rejects garbage bytes.
- (−) No support for true binary payloads. If a later change needs encrypted/compressed prompts, it adds a `prompt_payload BYTEA` column instead of overloading `body`.

### ADR-PMT-002 — Two tables (`prompt` + `prompt_revision`) instead of versioning columns on `prompt`

**Status**: Accepted (user-refined in proposal Q1, D1, D5).

**Context**: The naive versioning model puts `version INT`, `is_active BOOL`, and a `previous_id` back-pointer on the same table; reads of "the current version" become a partial-index lookup. The two-table model (proposed by the user during Q1) makes `prompt` always reflect the current state and `prompt_revision` an append-only journal of every change. Reverts are clean: insert-as-new-revision mutates `prompt` without touching history.

**Decision**: Two tables. See INV-1, INV-2 in `spec.md`.

**Consequences**:

- (+) Reads of the current prompt are a simple `SELECT FROM prompt WHERE slug = ?` — no partial-index quirk, no `is_active` filter, no `ORDER BY version DESC LIMIT 1`.
- (+) Revisions are immutable; restores preserve history (S-PR-4, S-PR-X2).
- (−) Every write spans two rows in one TX; the service must coordinate.
- (−) Stale revisions accumulate; a future retention sweeper is a separate change (out of scope for v1).

### ADR-PMT-003 — Slug uniqueness is partial over active rows

**Status**: Accepted (proposal D4).

**Context**: `UNIQUE(slug)` would prevent reusing `welcome` after a soft-delete, which the user wants. The partial-index idiom `UNIQUE(slug) WHERE deleted_at IS NULL` is well-documented (DEV.to article in explore.md §2). It also keeps reads fast because the partial index only holds live rows.

**Decision**: `CREATE UNIQUE INDEX prompt_slug_active_uidx ON prompt(slug) WHERE deleted_at IS NULL;`. The CHECK on `slug` format stays on the table.

**Consequences**:

- (+) Soft-deleted slugs are immediately reusable (S-PR-7, EC2).
- (+) The partial index stays small even with high churn.
- (−) Two soft-deleted prompts with the same slug can coexist in the DB; this is fine (they're invisible to API reads) but a future audit query must dedupe.

### ADR-PMT-004 — `revision_number` is assigned under `FOR UPDATE`, not via DB sequence

**Status**: Accepted (proposal D5 + spec INV-4).

**Context**: A naive `INSERT … revision_number = (SELECT MAX(revision_number)+1 FROM prompt_revision WHERE prompt_id = ?)` has a classic race: two concurrent inserts can pick the same `MAX+1`. Using a per-prompt `SEQUENCE` would work but would tie to the parent insert via trigger plumbing. The clean solution is row-level locking: `SELECT … FROM prompt WHERE id = ? FOR UPDATE` before reading `MAX(revision_number)`.

**Decision**: The service opens a TX, calls `promptRepo.LockAndLoad` (which issues the `FOR UPDATE` SQL), then computes `next = MaxRevisionNumber + 1`, then inserts the revision, updates the parent row, and commits. S-PR-21 confirms two concurrent updates serialize.

**Consequences**:

- (+) Correctness: monotonic numbering is guaranteed by the row lock, not by a uniqueness alone.
- (+) No trigger or sequence on the table; DDL stays simple.
- (−) Update throughput is single-flight per prompt. Multiple prompts in parallel still work; only the same prompt's updates serialize.

---

## 9. Files (delta from `main`)

```text
NEW    backend/database_administrator/src/migration/sql/20260715120000_prompts.sql
NEW    backend/database_administrator/src/domain/prompt.go
NEW    backend/database_administrator/src/domain/prompt_test.go
NEW    backend/database_administrator/src/domain/prompt_revision.go
NEW    backend/database_administrator/src/domain/prompt_errors.go
NEW    backend/database_administrator/src/domain/prompt_validation.go
NEW    backend/database_administrator/src/domain/prompt_validation_test.go
NEW    backend/database_administrator/src/domain/prompt_repo_port.go         # interface only
NEW    backend/database_administrator/src/domain/imports_test.go            # S-PR-X5
NEW    backend/database_administrator/src/infrastructure/postgres/prompt_repo.go
NEW    backend/database_administrator/src/infrastructure/postgres/prompt_repo_test.go
NEW    backend/database_administrator/src/infrastructure/postgres/prompt_revision_repo.go
NEW    backend/database_administrator/src/infrastructure/postgres/prompt_revision_repo_test.go
NEW    backend/database_administrator/src/application/prompt_service.go
NEW    backend/database_administrator/src/application/prompt_service_test.go
NEW    backend/database_administrator/src/interfaces/http/prompt_handler.go
NEW    backend/database_administrator/src/interfaces/http/prompt_handler_test.go
MOD    backend/database_administrator/src/cmd/server/main.go                 # +30 lines (wiring)
```

Estimated changed lines (rough): **~1,200** (Go ~1,100 + SQL ~100). Suggests chained PR split per `tasks.md`.

---

## 10. Out-of-scope design hooks (for future changes)

- pgvector column for prompt embeddings (out of scope per proposal §9; design leaves `prompt.body` as `TEXT` so a future `prompt_embedding` table can co-exist without DDL conflict).
- `prompt.tags TEXT[]` (out of scope; can be added with one ALTER).
- `prompt.search_vector tsvector GENERATED ALWAYS AS ...` (out of scope; deferred until a search use case appears).

---

## 11. Status

- ✅ DDL exact (will land as the migration file in tasks.md).
- ✅ Sequence diagrams for the four critical flows.
- ✅ Domain/service/handler contract written.
- ✅ ADRs 001-004 captured.
- ✅ Test map 1:1 with spec.
- ⏭ Next: `tasks.md` with chained-PR forecast + per-task RED steps.
