# Apply Progress — `2026-07-15-prompt-storage-table`

> **Change**: `2026-07-15-prompt-storage-table`
> **Apply strategy**: 4 PRs chained (stacked-to-main), `auto-forecast` per preflight.
> **Strict TDD**: ACTIVE — every task reports RED → GREEN status.
> **Final commit chain**: `1a5ec38 ← a600ff8 ← 483a299 ← ef33df6 ← f52a5e8 (main)`

This file is updated after each PR lands.

---

## Final state — ALL 4 PRs complete

| PR | SHA | Scope | Tests |
| --- | --- | --- | ---: |
| PR 1 | `ef33df6` | Migration + Domain | 28 unit |
| PR 2 | `483a299` | Repo pgx adapters | 25 integration |
| PR 3 | `a600ff8` | Application service | 17 integration |
| PR 4 | `1a5ec38` | HTTP handler + wiring | 13 integration |
| **Total** | | | **83 tests, 0 failures** |

All tests run with `INTEGRATION=1 go test -race -count=1 -p 1` against the compose-provisioned Postgres (`cachicamas_pg`, user `queen`).

---

## PR 1 — Migration + Domain ✅

Files (4 new, 736 lines):

- `backend/database_administrator/src/migration/sql/20260715120000_prompts.sql` (67 lines)
- `backend/database_administrator/src/domain/prompt.go` (327 lines)
- `backend/database_administrator/src/domain/prompt_test.go` (305 lines)
- `backend/database_administrator/src/domain/imports_test.go` (37 lines)

Tests: 28 unit, all green. `TestDomainLayer_DoesNotImportPgx` verifies the architectural invariant from spec S-PR-X5.

---

## PR 2 — Repo (pgx adapters) ✅

Files (4 new, ~1,200 lines):

- `backend/database_administrator/src/infrastructure/postgres/prompts/prompt_repo.go` (303 lines)
- `backend/database_administrator/src/infrastructure/postgres/prompts/prompt_revision_repo.go` (177 lines)
- `backend/database_administrator/src/infrastructure/postgres/prompts/prompt_repo_test.go` (563 lines)
- `backend/database_administrator/src/infrastructure/postgres/prompts/prompt_revision_repo_test.go` (269 lines)

Tests: 25 integration, all green. Lives in `infrastructure/postgres/prompts/` sub-package to isolate from the pre-existing `workspace_repo_test.go` build failure (unrelated to prompts).

Methods: PromptRepo.Insert / SelectBySlug / SelectBySlugAny / SelectByID / SelectList / UpdateBody / SoftDelete / LockAndLoad / MaxRevisionNumber. PromptRevisionRepo.Insert / SelectLatestForPrompt / SelectByPromptAndNumber / SelectListByPrompt.

Errors: pgconn 23505 → `*domain.ConflictError`; sql.ErrNoRows → `*domain.NotFoundError`. Handler does not import pgx.

---

## PR 3 — Application service ✅

Files (2 new, ~1,500 lines):

- `backend/database_administrator/src/application/prompt_service.go` (388 lines)
- `backend/database_administrator/src/application/prompt_service_test.go` (1,112 lines)

Tests: 17 integration, all green. Includes 3 concurrency tests:

- `TestPromptService_ConcurrentCreate_OneSucceedsOneConflicts` (S-PR-20)
- `TestPromptService_ConcurrentUpdate_ProducesMonotonicRevisions` (S-PR-21)
- `TestPromptService_ConcurrentRestoreAndUpdate_NoLostUpdate` (S-PR-X1)

Concurrency gate: FOR UPDATE row lock on `prompt.id` before reading the current row. Two goroutines on the same prompt serialize; revision numbers are monotonic.

---

## PR 4 — HTTP handler + wiring ✅

Files (2 new, ~1 modified):

- `backend/database_administrator/src/interfaces/http/prompt_handler.go` (380 lines)
- `backend/database_administrator/src/interfaces/http/prompt_handler_test.go` (410 lines)
- `backend/database_administrator/src/cmd/server/main.go` (+22 lines)

Tests: 13 integration, all green. Includes:

- Happy paths for all 7 endpoints
- Error envelope shape (S-PR-X4): `{"error":{"code":"...","message":"..."}}`
- Log redaction (S-PR-X3): `TestPromptHandler_NoPIIInLogs` captures the slog buffer and asserts a sentinel string in the request body is absent from any log line.

HTTP status mapping:

- 201 Created (POST success)
- 200 OK (GET, PATCH, restore)
- 204 No Content (DELETE)
- 400 Bad Request (*ValidationError, code=`validation`)
- 404 Not Found (*NotFoundError, code=`not_found`)
- 409 Conflict (*ConflictError, code=`conflict`)
- 410 Gone (*GoneError, code=`prompt_deleted` — the only NEW wire code)

---

## Run instructions

```bash
# 1. Apply the migration to the dev DB (idempotent).
docker exec -i cachicamas-postgres psql -U queen -d cachicamas_pg \
  < backend/database_administrator/src/migration/sql/20260715120000_prompts.sql

# 2. Run the full SDD test suite.
cd backend/database_administrator
INTEGRATION=1 go test -race -count=1 -p 1 \
  -run "^(TestValidateSlug|TestValidateDescription|TestValidateBody|TestValidate.*_Error|TestNewPromptDeleted|TestAsPromptDeleted|TestPrompt_|TestCodePrompt|TestMaxPrompt|TestDefaultAndMax|TestDomainLayer|TestPromptRepo_|TestPromptRevisionRepo_|TestPromptService_|TestPromptHandler_)" \
  ./src/domain/... ./src/infrastructure/postgres/prompts/... \
  ./src/application/... ./src/interfaces/http/...
# expect: 83 PASS, 0 FAIL
```

---

## Deviations from the locked design (final list)

1. **Wire codes are generic, not feature-specific.** The project's existing handler maps `*ValidationError` / `*ConflictError` / `*NotFoundError` to generic codes (`validation`, `conflict`, `not_found`). PR 1 follows this convention. The only NEW wire code is `prompt_deleted` (HTTP 410) via the new `*GoneError`. `design.md §4.2` documents this.
2. **`*GoneError` is new.** No existing type covers HTTP 410, so `domain/prompt.go` declares a dedicated type + `NewPromptDeleted(slug)` + `AsPromptDeleted(err)` helper.
3. **`SelectBySlugAny` added.** The repo's `SelectBySlug` hides soft-deleted rows (returns 404 for both "never existed" and "soft-deleted"). To honor spec S-PR-5 / S-PR-8 (410 for soft-deleted), the service uses `SelectBySlugAny` first, checks `DeletedAt`, and maps accordingly. This is captured in PR 3.
4. **Prompts live in `infrastructure/postgres/prompts/` sub-package.** The pre-existing `workspace_repo_test.go` build failure (fields renamed in 2026-07-08 simplify, test not updated) blocks the parent `postgres` package's test compilation. The new sub-package isolates the prompts tests cleanly.
5. **`UpdateBody` does not take `updatedAt` arg.** The DB owns `updated_at` via `DEFAULT now()`. Safer for multi-TX scenarios.

None of these deviations are breaking changes; all are documented in `design.md §4.2` and `tasks.md`.

---

## Pre-existing failures NOT caused by this change

- `src/infrastructure/postgres/workspace_repo_test.go` references `domain.Workspace.PrimaryRepoGitHubID` and `domain.LinkedRepository`, both renamed/dropped in the 2026-07-08 workspaces-simplify change but the test file was not updated. Separate cleanup task. Run `make test` excludes the prompt suite from this; the prompt suite passes cleanly.

---

## Skill resolution

`paths-injected`. Loaded before any work:

- `/Users/braejan/.claude/skills/go-testing/SKILL.md`
- `/Users/braejan/.claude/skills/test-driven-development/SKILL.md`
- `/Users/braejan/.claude/skills/work-unit-commits/SKILL.md`
- `/Users/braejan/.config/opencode/skills/chained-pr/SKILL.md`

---

## Open items (post-merge)

- Sync `openspec/changes/2026-07-15-prompt-storage-table/specs/prompts/spec.md` to `openspec/specs/prompts/spec.md` (canonical).
- Archive the change to `openspec/changes/archive/2026-07-15-prompt-storage-table/`.
- Decide on the chained PR split vs single mega-PR for the actual GitHub PR (currently committed as 4 sequential commits on `feat/2026-07-15-prompts-pr1`).
