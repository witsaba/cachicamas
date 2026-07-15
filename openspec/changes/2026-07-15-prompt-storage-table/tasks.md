# Tasks — `2026-07-15-prompt-storage-table`

> **Change**: `2026-07-15-prompt-storage-table`
> **Project**: `cachicamas`
> **Artifact store**: `openspec/changes/2026-07-15-prompt-storage-table/`
> **Strict TDD**: ACTIVE — every code task starts with a failing test (RED → GREEN → REFACTOR).
> **Inherits**: `proposal.md` (D1–D10), `spec.md` (S-PR-1..27, S-PR-X1..X5), `design.md` (§0 test map, §3 DDL, §4–6 contracts, §8 ADRs).

---

## 0. Review Workload Forecast (MANDATORY)

Decision needed before apply: **Yes** (chained PR recommendation + budget risk).

| Field | Value |
| --- | --- |
| Estimated changed lines | **~1,630** (Go ~1,500 + SQL ~80 + minor wiring +30) |
| 450-line budget risk | **High** (~3.6× budget per single PR) |
| Chained PRs recommended | **Yes** |
| Delivery strategy | **ask-on-risk** (orchestrator will pause for user confirmation per `gentle-ai/SKILL.md` preflight rule) |
| Chain strategy | **stacked-to-main** (each PR merges to main in order) |

**Why 4 PRs, not 1 or 2**:

| PR | Files | ~Lines | Standalone testable? |
| --- | --- | ---: | :---: |
| PR 1 — Migration + Domain | 1 SQL + 7 Go files (4 prod + 3 test) | ~430 | ✅ `go test ./src/domain/...` passes without DB; migration testable via `make test` integration tag |
| PR 2 — Repo (pgx adapters) | 4 Go files (2 prod + 2 test) | ~400 | ✅ `go test ./src/infrastructure/postgres/...` with `integration` tag against compose DB |
| PR 3 — Application service | 2 Go files (1 prod + 1 test) | ~300 | ✅ `go test ./src/application/...` with `integration` tag |
| PR 4 — HTTP handler + wiring | 2 Go files (1 prod + 1 test) + `cmd/server/main.go` (+30) | ~500 | ✅ `go test ./src/interfaces/http/...` + `make test` |

A single PR would be ~3.6× the 450-line budget — too large to review safely. A 2-PR split (e.g., PR1 = "data layer" + PR2 = "API") would put both PRs at ~800 lines each, still ~2× over budget.

A 5th PR (e.g., separating PR4 into "handlers" and "wiring") would not justify its own review cycle — wiring is 30 lines.

**Merge order**: PR1 → PR2 → PR3 → PR4. Strict serialization because each layer depends on the previous.

**Rollback per PR**: each PR is small enough that a `git revert <merge-sha>` cleanly removes it. No data migration is needed for any single PR because the change is purely additive.

---

## 1. Work-unit commit style

Per `openspec/AGENTS.md`, every commit is a **conventional commit** scoped to one logical unit. Suggested prefixes for this change:

- `feat(db):` — migrations
- `feat(domain):` — domain types / validation
- `feat(repo):` — pgx adapters
- `feat(svc):` — application service
- `feat(http):` — handlers
- `feat(wire):` — main.go wiring
- `test(domain):`, `test(repo):`, etc. — test-only commits where it helps reviewers (optional; the RED step is usually committed with the GREEN step in one commit per task)

---

## 2. Per-Task Breakdown

### PR 1 — "Migration + Domain"

**Goal**: The schema and the domain layer exist and are tested. The `prompt` and `prompt_revision` tables land in the DB. Domain validation, error vocabulary, and the repo port interface are in place. No pgx, no HTTP.

**Spec coverage**: INV-1, INV-2, INV-3 (DDL level), S-PR-10..19 (validation), S-PR-X5 (no pgx in domain).

---

#### Task 1.1 — Write the goose migration `20260715120000_prompts.sql`

- **Spec ref**: INV-1, INV-2 (DDL only).
- **Output**: NEW `backend/database_administrator/src/migration/sql/20260715120000_prompts.sql`. Verbatim from `design.md` §3.
- **Test**: `go test ./src/migration/... -tags=integration` (runs the goose runner against a test DB; the existing migration tests already cover this pattern — see `migration/runner_test.go`).
- **Reviewable in**: 10 minutes.
- **Work-unit commit**: `feat(db): add prompt + prompt_revision tables with partial unique slug`.

---

#### Task 1.2 — Write `domain/prompt.go` and `domain/prompt_revision.go` (types + zero-value constructors) **RED first**

- **Spec ref**: spec INV-3 (column shape contract).
- **Output**: NEW `domain/prompt.go` with the `Prompt` struct + `db`/`json` tags. NEW `domain/prompt_revision.go` with the `PromptRevision` struct.
- **Test (RED)**: `domain/prompt_test.go` — `TestPrompt_TypeShape` asserts `Prompt.ID` is `int64`, `Description` is `string`, etc. Compiles only after the types exist.
- **Test (GREEN)**: same test passes.
- **Reviewable in**: 10 minutes.
- **Work-unit commit**: `feat(domain): add Prompt and PromptRevision value types`.

---

#### Task 1.3 — Add slug + description + body validators **RED first**

- **Spec ref**: S-PR-10..19.
- **Output**: NEW `domain/prompt_validation.go` with `ValidateSlug`, `ValidateDescription`, `ValidateBody`, and the `slugRe` regex.
- **Test (RED)**: `domain/prompt_validation_test.go` — table-driven tests:
  - `TestValidateSlug_RejectsUppercase` (S-PR-10)
  - `TestValidateSlug_Accepts2Chars` (S-PR-12, boundary)
  - `TestValidateSlug_Accepts100Chars` (S-PR-12, boundary upper)
  - `TestValidateSlug_RejectsLeadingHyphen` (S-PR-13)
  - `TestValidateSlug_RejectsTrailingHyphen` (S-PR-14)
  - `TestValidateSlug_AcceptsHyphenInMiddle` (sanity)
  - `TestValidateDescription_RejectsEmpty` (S-PR-15)
  - `TestValidateDescription_Accepts280Chars` (S-PR-16)
  - `TestValidateDescription_Rejects281Chars` (S-PR-16 boundary)
  - `TestValidateBody_RejectsEmpty` (S-PR-19)
  - `TestValidateBody_Accepts524288Chars` (S-PR-17 boundary)
  - `TestValidateBody_Rejects524289Chars` (S-PR-18)
- **Reviewable in**: 20 minutes.
- **Work-unit commit**: `feat(domain): add prompt slug/description/body validation`.

---

#### Task 1.4 — Add locked error vocabulary **RED first**

- **Spec ref**: §4.2 error codes + S-PR-X4 envelope.
- **Output**: NEW `domain/prompt_errors.go` with:

  ```go
  type PromptError struct {
      Code    string
      Message string
      cause   error
  }
  func (e *PromptError) Error() string  { return e.Message }
  func (e *PromptError) Code() string   { return e.Code }
  func (e *PromptError) Unwrap() error  { return e.cause }
  ```

  Plus constructors: `NewPromptSlugTaken()`, `NewPromptNotFound(slug)`, `NewPromptDeleted()`, `NewPromptRevisionNotFound(n)`, `NewPromptSlugInvalid(reason)`, `NewPromptDescriptionInvalid(reason)`, `NewPromptBodyTooLarge(got, max)`.
- **Test (RED → GREEN)**: `domain/prompt_errors_test.go`:
  - `TestPromptError_Code_ReturnsLockedCode`
  - `TestPromptError_Unwrap_PreservesCause`
  - `TestNewPromptSlugTaken_ProducesConflictError`
- **Reviewable in**: 15 minutes.
- **Work-unit commit**: `feat(domain): add locked prompt error vocabulary`.

---

#### Task 1.5 — Add the repo ports (interfaces only) **RED first**

- **Spec ref**: design §4.3.
- **Output**: NEW `domain/prompt_repo_port.go` with `PromptRepository`, `PromptRevisionRepository`, and the `sqlExecutor` interface.
- **Test (RED)**: `domain/prompt_repo_port_test.go` is a compile-time check that the interface signatures exist (`var _ PromptRepository = nil` — fails to compile until the interface is defined).
- **Reviewable in**: 10 minutes.
- **Work-unit commit**: `feat(domain): add prompt repository ports`.

---

#### Task 1.6 — Enforce "domain must not import pgx" **RED first**

- **Spec ref**: S-PR-X5.
- **Output**: NEW `domain/imports_test.go` that shells out to `go list -deps -f '{{.ImportPath}}' ./src/domain` and asserts the output does not contain `github.com/jackc/pgx`.
- **Test**: runs in the `make test` target (no integration tag needed).
- **Reviewable in**: 5 minutes.
- **Work-unit commit**: `test(domain): enforce no pgx import in domain package`.

---

**PR 1 task count: 6.** Total ~430 lines.

---

### PR 2 — "Repo (pgx adapters)"

**Goal**: The pgx adapters implement the ports and translate SQLSTATE 23505 / "no rows" into domain errors. Integration tests run against the compose DB.

**Spec coverage**: INV-1..4 at the DB layer, EC1 (unique violation), EC9 (concurrent INSERT).

**Files**:

- NEW `infrastructure/postgres/prompt_repo.go` (~180 lines prod + 10 lines package doc).
- NEW `infrastructure/postgres/prompt_repo_test.go` (~150 lines integration tests).
- NEW `infrastructure/postgres/prompt_revision_repo.go` (~140 lines prod).
- NEW `infrastructure/postgres/prompt_revision_repo_test.go` (~130 lines integration tests).

---

#### Task 2.1 — Implement `PromptRepo` skeleton (constructor + `Insert`)

- **Spec ref**: design §5.1 step 4.
- **Output**: `PromptRepo` struct + `NewPromptRepo(db *sql.DB) *PromptRepo`. `Insert(ctx, sqlExecutor, *Prompt) error` that executes the INSERT and returns `*sql.DB`-style errors. Compile-time `var _ domain.PromptRepository = (*PromptRepo)(nil)`.
- **Test (RED)**: `TestPromptRepo_Insert_ReturnsNoErrorOnHappyPath`.
- **Reviewable in**: 15 minutes.
- **Work-unit commit**: `feat(repo): add PromptRepo.Insert skeleton`.

---

#### Task 2.2 — Implement `SelectBySlug`, `SelectByID`, `SelectList`

- **Spec ref**: S-PR-22, S-PR-25.
- **Output**: three methods. `SelectBySlug` and `SelectByID` translate `sql.ErrNoRows` → `domain.NewPromptNotFound(slug)`. `SelectList` filters `WHERE deleted_at IS NULL ORDER BY updated_at DESC LIMIT $1 OFFSET $2`.
- **Test (RED → GREEN)**:
  - `TestPromptRepo_SelectBySlug_HappyPath` (integration)
  - `TestPromptRepo_SelectBySlug_NotFound` (integration; asserts `*domain.PromptError` with `Code() == "PROMPT_NOT_FOUND"`)
  - `TestPromptRepo_SelectBySlug_DeletedPrompt_ReturnsNotFound`
  - `TestPromptRepo_SelectList_OrderByUpdatedAtDesc`
  - `TestPromptRepo_SelectList_ExcludesDeleted`
- **Reviewable in**: 25 minutes.
- **Work-unit commit**: `feat(repo): add PromptRepo select methods with not-found translation`.

---

#### Task 2.3 — Implement `UpdateBody`, `SoftDelete`, `LockAndLoad`

- **Spec ref**: S-PR-2, S-PR-6, design §5.2 step 3.
- **Output**: `UpdateBody` issues `UPDATE prompt SET body=$2, description=$3, updated_at=now() WHERE id=$1`. `SoftDelete` issues `UPDATE prompt SET deleted_at=now() WHERE id=$1`. `LockAndLoad` issues `SELECT … FOR UPDATE`.
- **Test (RED → GREEN)**:
  - `TestPromptRepo_UpdateBody_UpdatesRow`
  - `TestPromptRepo_SoftDelete_SetsDeletedAt`
  - `TestPromptRepo_LockAndLoad_AcquiresRowLock` (verify by attempting a concurrent UPDATE in a separate TX and timing out)
  - `TestPromptRepo_LockAndLoad_NotFound` (asserts `PromptError(Code: PROMPT_NOT_FOUND)`)
- **Reviewable in**: 25 minutes.
- **Work-unit commit**: `feat(repo): add PromptRepo update, soft-delete, and FOR UPDATE lock`.

---

#### Task 2.4 — Implement `MaxRevisionNumber` and translate 23505 → `PROMPT_SLUG_TAKEN`

- **Spec ref**: EC1, S-PR-20, design §5.1 step 4 (unique violation handling).
- **Output**: helper `translateInsertError(err) error` that detects `*pgconn.PgError` with code `23505` and returns `domain.NewPromptSlugTaken()`. Apply it in `Insert` and `UpdateBody` (UPDATE on slug is not used, but UpdateBody can race with delete + recreate with the same slug — same partial-index collision is possible). Add `MaxRevisionNumber(ctx, sqlExecutor, promptID) (int, error)` returning `COALESCE(MAX(revision_number), 0)`.
- **Test (RED → GREEN)**:
  - `TestPromptRepo_Insert_DuplicateSlug_ReturnsConflictError` (integration; inserts twice with same slug; second returns `PROMPT_SLUG_TAKEN`)
  - `TestPromptRepo_MaxRevisionNumber_ReturnsZeroOnEmpty` (integration)
  - `TestPromptRepo_MaxRevisionNumber_ReturnsLatestOnExisting` (integration)
- **Reviewable in**: 25 minutes.
- **Work-unit commit**: `feat(repo): translate 23505 to PROMPT_SLUG_TAKEN and add MaxRevisionNumber`.

---

#### Task 2.5 — Implement `PromptRevisionRepo`

- **Spec ref**: INV-2, S-PR-26.
- **Output**: `PromptRevisionRepo` with `Insert`, `SelectLatestForPrompt`, `SelectByPromptAndNumber`, `SelectListByPrompt`. `SelectListByPrompt` orders `revision_number DESC`. Compile-time check `var _ domain.PromptRevisionRepository = (*PromptRevisionRepo)(nil)`.
- **Test (RED → GREEN)**:
  - `TestPromptRevisionRepo_Insert_StoresSnapshot`
  - `TestPromptRevisionRepo_SelectByPromptAndNumber_NotFound` (asserts `PROMPT_REVISION_NOT_FOUND`)
  - `TestPromptRevisionRepo_SelectListByPrompt_OrderDesc`
  - `TestPromptRevisionRepo_CascadeDeleteRemovesRevisions` (deletes the parent prompt; asserts all revisions are gone)
- **Reviewable in**: 25 minutes.
- **Work-unit commit**: `feat(repo): add PromptRevisionRepo with snapshot semantics`.

---

**PR 2 task count: 5.** Total ~400 lines.

---

### PR 3 — "Application service"

**Goal**: The `PromptService` orchestrates create / update / restore / soft-delete / list / get / list-revisions. Concurrency invariants from S-PR-20, S-PR-21, S-PR-X1 hold.

**Spec coverage**: S-PR-1..5, S-PR-7, S-PR-20, S-PR-21, S-PR-X1, S-PR-X2.

**Files**:

- NEW `application/prompt_service.go` (~200 lines prod).
- NEW `application/prompt_service_test.go` (~250 lines integration tests, including race tests).

---

#### Task 3.1 — Implement `Create(ctx, in)` (service)

- **Spec ref**: S-PR-1.
- **Output**: `Create` runs validation, opens a TX, inserts the prompt, inserts revision `1`, commits, returns both hydrated rows.
- **Test (RED → GREEN)**: `TestPromptService_Create_WritesRevisionOne` (integration; asserts the prompt exists, the revision exists with `revision_number=1`, the bodies match).
- **Reviewable in**: 20 minutes.
- **Work-unit commit**: `feat(svc): add PromptService.Create with TX-bound revision 1`.

---

#### Task 3.2 — Implement `Update(ctx, slug, in)`

- **Spec ref**: S-PR-2, S-PR-3, design §5.2.
- **Output**: `Update` opens a TX, `LockAndLoad` the prompt, compute `next = MaxRevisionNumber + 1`, insert the new revision, `UpdateBody`, commit, re-read, return.
- **Test (RED → GREEN)**:
  - `TestPromptService_Update_AppendsNextRevision`
  - `TestPromptService_Update_DescriptionOnly_AppendsRevision`
  - `TestPromptService_Update_NotFound_ReturnsPromptNotFound`
  - `TestPromptService_Update_DeletedPrompt_Returns410`
- **Reviewable in**: 25 minutes.
- **Work-unit commit**: `feat(svc): add PromptService.Update with FOR UPDATE serialization`.

---

#### Task 3.3 — Implement `Restore(ctx, slug, n)`

- **Spec ref**: S-PR-4, S-PR-5.
- **Output**: `Restore` opens a TX, `LockAndLoad` the prompt (reject if deleted), read revision `n`, compute `next`, insert a new revision copying historical body and description, set `change_note = "restored from revision <n>"`, `UpdateBody` on the parent, commit.
- **Test (RED → GREEN)**:
  - `TestPromptService_Restore_AppendsNewRevisionWithHistoricalBody`
  - `TestPromptService_Restore_OnDeletedPrompt_ReturnsDeletedError`
  - `TestPromptService_Restore_RevisionNotFound_Returns404`
- **Reviewable in**: 20 minutes.
- **Work-unit commit**: `feat(svc): add PromptService.Restore with snapshot of historical revision`.

---

#### Task 3.4 — Implement `SoftDelete`, `GetBySlug`, `List`, `ListRevisions`

- **Spec ref**: S-PR-6, S-PR-9, S-PR-22..27.
- **Output**: four more methods. `SoftDelete` is idempotent (returns nil on already-deleted). `List` defaults `limit=50, offset=0`, hard cap `limit <= 200`. `ListRevisions` requires the prompt to exist and not be soft-deleted.
- **Test (RED → GREEN)**:
  - `TestPromptService_SoftDelete_IdempotentOnSecondCall`
  - `TestPromptService_GetBySlug_DeletedPrompt_ReturnsNotFound`
  - `TestPromptService_List_ExcludesDeleted`
  - `TestPromptService_List_LimitCapEnforced` (asserts `limit=300` becomes `limit=200`)
  - `TestPromptService_ListRevisions_MissingPrompt_Returns404`
  - `TestPromptService_ListRevisions_NewestFirst`
- **Reviewable in**: 30 minutes.
- **Work-unit commit**: `feat(svc): add SoftDelete, GetBySlug, List, ListRevisions`.

---

#### Task 3.5 — Concurrency race tests (S-PR-20, S-PR-21, S-PR-X1)

- **Spec ref**: S-PR-20, S-PR-21, S-PR-X1.
- **Output**: three new tests in `prompt_service_test.go`:
  - `TestPromptService_ConcurrentCreate_OneSucceedsOneConflicts` — two goroutines POST the same slug; one gets 201, the other `PROMPT_SLUG_TAKEN`. Uses `sync.WaitGroup`.
  - `TestPromptService_ConcurrentUpdate_ProducesMonotonicRevisions` — two goroutines PATCH the same prompt; both succeed; final `revision_number` is exactly `prev + 2`; both new revisions are present.
  - `TestPromptService_ConcurrentRestoreAndUpdate_NoLostUpdate` — one goroutine restores rev 1, another PATCHes; both complete; revision numbers are monotonic.
- **Test (RED → GREEN)**: each test runs `t.Parallel()` and uses a fresh DB transaction isolation; passes after Tasks 3.1–3.4 are in.
- **Reviewable in**: 30 minutes.
- **Work-unit commit**: `test(svc): add concurrency invariants for create, update, and restore`.

---

**PR 3 task count: 5.** Total ~300 lines (production smaller; tests heavier).

---

### PR 4 — "HTTP handler + wiring"

**Goal**: The `/prompts/*` routes work end-to-end. The error envelope shape is locked.

**Spec coverage**: S-PR-6..9, S-PR-22..27, S-PR-X3, S-PR-X4.

**Files**:

- NEW `interfaces/http/prompt_handler.go` (~200 lines prod).
- NEW `interfaces/http/prompt_handler_test.go` (~250 lines tests).
- MOD `interfaces/http/router.go` (or `cmd/server/main.go`) — add 7 routes (+30 lines).

---

#### Task 4.1 — Add handler struct + constructor + envelope helpers

- **Spec ref**: S-PR-X4.
- **Output**: `PromptHandler` struct (handler depends only on `application.PromptService` + `*slog.Logger`). `writeError(c echo.Context, status int, code, message string)` helper. `writeJSON(c, status, payload)` helper.
- **Test (RED → GREEN)**: `TestPromptHandler_ErrorEnvelopeShape` (asserts the JSON keys are exactly `{"error":{"code":"…","message":"…"}}`).
- **Reviewable in**: 15 minutes.
- **Work-unit commit**: `feat(http): add PromptHandler with locked error envelope`.

---

#### Task 4.2 — Implement `Create` handler

- **Spec ref**: S-PR-1, EC1, EC5, EC6.
- **Output**: `POST /prompts` parses the request body, calls `service.Create`, maps errors:
  - `PROMPT_SLUG_TAKEN` → 409
  - `PROMPT_SLUG_INVALID` / `PROMPT_DESCRIPTION_INVALID` / `PROMPT_BODY_TOO_LARGE` → 400
- **Test (RED → GREEN)**:
  - `TestPromptHandler_Create_HappyPath_201` (asserts response body and status)
  - `TestPromptHandler_Create_DuplicateSlug_409`
  - `TestPromptHandler_Create_InvalidSlug_400`
  - `TestPromptHandler_Create_EmptyBody_400`
  - `TestPromptHandler_Create_BodyTooLarge_400` (mock the validator or use a 600 KB body)
- **Reviewable in**: 25 minutes.
- **Work-unit commit**: `feat(http): add POST /prompts handler with full validation`.

---

#### Task 4.3 — Implement `GetBySlug`, `List`, `ListRevisions`, `Delete`

- **Spec ref**: S-PR-6, S-PR-9, S-PR-22..27.
- **Output**: four more handlers. `GetBySlug` and `ListRevisions` map `PROMPT_NOT_FOUND` → 404. `Delete` is idempotent (returns 204 on already-deleted).
- **Test (RED → GREEN)**:
  - `TestPromptHandler_GetBySlug_200`
  - `TestPromptHandler_GetBySlug_404` (missing)
  - `TestPromptHandler_GetBySlug_404` (soft-deleted)
  - `TestPromptHandler_List_ExcludesDeleted`
  - `TestPromptHandler_List_OrderByUpdatedAtDesc`
  - `TestPromptHandler_ListRevisions_NewestFirst`
  - `TestPromptHandler_ListRevisions_404` (missing)
  - `TestPromptHandler_Delete_204`
  - `TestPromptHandler_Delete_Idempotent_204`
- **Reviewable in**: 30 minutes.
- **Work-unit commit**: `feat(http): add read handlers and soft-delete`.

---

#### Task 4.4 — Implement `Update` and `Restore` handlers

- **Spec ref**: S-PR-2, S-PR-3, S-PR-4, S-PR-5, S-PR-8.
- **Output**: two more handlers. `PATCH /prompts/:slug` accepts partial `{description?, body?}`; at least one must be present. `POST /prompts/:slug/revisions/:n/restore` reads `:n` as int.
- **Test (RED → GREEN)**:
  - `TestPromptHandler_Update_HappyPath_200`
  - `TestPromptHandler_Update_DescriptionOnly_200`
  - `TestPromptHandler_Update_Deleted_410`
  - `TestPromptHandler_Update_NotFound_404`
  - `TestPromptHandler_Restore_HappyPath_200`
  - `TestPromptHandler_Restore_OnDeleted_410`
  - `TestPromptHandler_Restore_RevisionNotFound_404`
- **Reviewable in**: 30 minutes.
- **Work-unit commit**: `feat(http): add PATCH /prompts/:slug and restore endpoints`.

---

#### Task 4.5 — Add the route wiring + S-PR-X3 (no PII in logs)

- **Spec ref**: design §7, S-PR-X3.
- **Output**: in `cmd/server/main.go` (or wherever routes are declared), add 7 routes (Section 7 of design). The handler logs via `slog.Default()` and MUST NOT log the request body or the prompt body.
- **Test (RED → GREEN)**:
  - `TestPromptHandler_NoPIITokenInLogs` — captures `slog` output via a custom `slog.Handler`; sends a `POST /prompts` with body containing a unique sentinel string; asserts the sentinel is absent from the captured log lines.
- **Reviewable in**: 20 minutes.
- **Work-unit commit**: `feat(wire): add /prompts routes with log redaction`.

---

**PR 4 task count: 5.** Total ~500 lines (handler tests are the heaviest part).

---

## 3. Cross-PR Dependencies (DAG)

```
PR 1 (Migration + Domain)
   │
   ▼
PR 2 (Repo)
   │
   ▼
PR 3 (Service)
   │
   ▼
PR 4 (Handler + wiring)
```

- **No cycles.** Each layer depends on the previous.
- **Strict serialization**: PR 2 cannot merge before PR 1 (no port interface); PR 3 cannot merge before PR 2 (no repo); PR 4 cannot merge before PR 3 (no service).
- **PR 1 alone is shippable**: the migration runs and the domain types compile, but no HTTP surface is exposed.
- **PR 1 + PR 2 together** are shippable but inert (data layer only; nothing calls the service).
- **PR 1 + PR 2 + PR 3** ship a usable Go API but no HTTP. The service could be consumed by other internal code immediately (e.g., an LLM driver).
- **PR 1 + PR 2 + PR 3 + PR 4** ship the full v1.

---

## 4. Per-PR Review Checklists

### PR 1 reviewer checklist

- [ ] Migration `20260715120000_prompts.sql` matches `design.md` §3 verbatim.
- [ ] Domain types mirror the DDL column-for-column with `db` and `json` tags.
- [ ] `ValidateSlug` regex matches `^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$` exactly.
- [ ] All validation tests pass on the boundary cases.
- [ ] Error vocabulary matches `design.md` §4.2 exactly (codes, naming).
- [ ] `domain` package compiles without any `github.com/jackc/pgx` import (verified by `TestDomainLayer_DoesNotImportPgx`).
- [ ] `make test` passes; `make lint` passes.

### PR 2 reviewer checklist

- [ ] `PromptRepo` implements all methods of `domain.PromptRepository` (compile-time check).
- [ ] `PromptRevisionRepo` implements all methods of `domain.PromptRevisionRepository`.
- [ ] `*pgconn.PgError` translation to `PROMPT_SLUG_TAKEN` is covered by an integration test.
- [ ] `LockAndLoad` issues `SELECT … FOR UPDATE` (verified by `TestPromptRepo_LockAndLoad_AcquiresRowLock`).
- [ ] All integration tests run against the compose DB and pass on `-race`.
- [ ] No `pgx` imports outside `infrastructure/postgres/`.
- [ ] `make test` passes; `make lint` passes.

### PR 3 reviewer checklist

- [ ] `PromptService.Create` writes both prompt + revision in a single TX.
- [ ] `PromptService.Update` and `PromptService.Restore` acquire the `FOR UPDATE` lock before computing the next revision number.
- [ ] All concurrency tests pass (S-PR-20, S-PR-21, S-PR-X1).
- [ ] `List` enforces `limit <= 200`.
- [ ] `SoftDelete` is idempotent.
- [ ] All service tests pass on `-race`.

### PR 4 reviewer checklist

- [ ] All 7 routes are registered in `cmd/server/main.go` (or wherever routes live).
- [ ] Error envelope matches the locked shape (S-PR-X4).
- [ ] No prompt body or request body is logged (S-PR-X3, captured-log test).
- [ ] All HTTP status codes match `design.md` §6.
- [ ] `make test` passes; `make build` produces the binary; `make lint` passes.

---

## 5. Out-of-Scope Tasks (explicitly NOT in the apply phase)

These are listed in the proposal §9. The apply phase MUST NOT create tasks for them:

- Vector embeddings of prompts (pgvector).
- Prompt templating / variable substitution engine.
- UI surface for editing prompts.
- Multi-tenant scoping (`organization_id` FK).
- Prompt history pruning / retention sweeper.
- Encryption-at-rest of body content (Postgres-level TDE).
- Bulk import / export.
- A/B testing hooks.
- Rate limiting on write endpoints.

---

## 6. Risks and Mitigations

| Risk | Severity | Mitigation |
| --- | --- | --- |
| 4 PRs feels like a lot for ~1,630 lines | Low | Each PR is reviewable in isolation; no cross-cutting concerns. Merge order is unambiguous. |
| Migration is forward-only; if PR 1 lands in dev and the schema is wrong, the next PRs cannot proceed | Low | The DDL is reviewed in PR 1; if a typo slips through, the small migration patch is a 1-line diff in PR 1 itself before anything else merges. |
| Integration tests in PR 2 depend on the compose DB | Medium | All repo tests use the `-tags=integration` build tag and the existing `migration/postgres.Open` test helper (see `migration/postgres/driver_test.go`). The CI hook (Makefile) runs `make test` with `integration` on PRs that touch `infrastructure/`. |
| Race tests are flaky in CI | Medium | Use the same pattern as `application/sync_callback_integration_test.go` (the existing project integration test). Avoid `time.Sleep`; use channel signaling or `errgroup`. |
| `FOR UPDATE` lock can deadlock if update-then-restore race | Low | Lock order is always `prompt.id` first, then `prompt_revision` insert. The two paths in PR 3 share the same starting point; deadlock is impossible. |
| Service test setup creates real DB rows; CI DB pollution | Low | Each test uses a TX-scoped fixture helper that rolls back at test end. The existing pattern is in `application/sync_service_test.go`. |
| `make build` fails after PR 4 wiring | Low | PR 4 reviewer's checklist includes `make build`. |
| The pi-lens `MD013` advisory (50 warnings on this `tasks.md`) | None | Cosmetic; the warnings are line-length advisories on the test-map table. No action required. |

---

## 7. User Decisions (locked at the apply gate)

- **Q-A (delivery)**: **4 PRs chained, stacked-to-main**. PR1 → PR2 → PR3 → PR4, each mergeable on its own.
- **Q-B (pagination)**: **Simple `limit/offset`**. Defaults `limit=50`, hard cap `limit <= 200`. Forward-compat for keyset is a v1.1 follow-up.
- **Q-C (restore endpoint)**: **Path-segment** `POST /prompts/:slug/revisions/:n/restore`. RESTful; `:n` is the revision number.
- **Q-D (auth posture)**: **Admin-only, no extra header**. The route is bound to the internal compose network; no new middleware in v1.
- **Q-E (softdelete)**: **Inline** — the `workspace` table does it inline and there is no shared package to reuse.

---

## 8. Skill Resolution

`paths-injected` (project registry not strictly needed; project AGENTS.md already lists the relevant skill paths). Loaded before any work:

- `/Users/braejan/.claude/skills/go-testing/SKILL.md`
- `/Users/braejan/.claude/skills/test-driven-development/SKILL.md` (referenced from project AGENTS.md; if not present on disk, the orchestrator must surface a degraded warning)
- `/Users/braejan/.claude/skills/work-unit-commits/SKILL.md`
- `/Users/braejan/.config/opencode/skills/chained-pr/SKILL.md`
- `/Users/braejan/.config/opencode/skills/branch-pr/SKILL.md`

---

## 9. Status

- ✅ Forecast at the top (per project rule).
- ✅ PR split justified and merge order locked.
- ✅ Per-task breakdown with RED steps (strict TDD enforced).
- ✅ Cross-PR DAG drawn.
- ✅ Reviewer checklists per PR.
- ✅ Out-of-scope items locked.
- ✅ Risks enumerated.
- ⏭ Next: orchestrator pause for user confirmation (Q-A..Q-E) → `sdd-apply` (4 subagent runs, one per PR).
