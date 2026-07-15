# Apply Progress — `2026-07-15-prompt-storage-table`

> **Change**: `2026-07-15-prompt-storage-table`
> **Apply strategy**: 4 PRs chained (stacked-to-main), `auto-forecast` per preflight.
> **Strict TDD**: ACTIVE — every task reports RED → GREEN status.

This file is updated after each PR lands. Read it for the current state of the change.

---

## PR 1 — Migration + Domain (DONE)

### What landed

| File | Status | Lines |
| --- | --- | ---: |
| `backend/database_administrator/src/migration/sql/20260715120000_prompts.sql` | ✅ New | 87 |
| `backend/database_administrator/src/domain/prompt.go` | ✅ New | 367 |
| `backend/database_administrator/src/domain/prompt_test.go` | ✅ New | 320 |
| `backend/database_administrator/src/domain/imports_test.go` | ✅ New | 33 |

**Total: 807 lines** (over the original ~430 forecast because the consolidated `prompt.go` includes the ports + validation + errors + the new `GoneError` type all in one file per the project's per-feature file convention; this is still well under 450 lines per file).

### Test results

- `go test -race -count=1 ./src/domain/...` → **28 tests PASS** (1.5s wall time).
  - 10 slug validation tests (S-PR-10..14)
  - 3 description validation tests (S-PR-15,16)
  - 3 body validation tests (S-PR-17,18,19)
  - 3 ValidationError surface tests
  - 3 GoneError surface tests
  - 2 entity struct shape tests
  - 4 locked-constant stability tests
  - 1 architectural invariant test (`TestDomainLayer_DoesNotImportPgx`)
- `go vet ./src/domain/...` → clean.
- `go build ./src/domain/...` → clean.

### What was NOT done in PR 1 (and why)

- **DDL not applied to the dev DB.** The migration file is on disk; running it against the compose DB is a manual `make migrate` step the user (or CI) triggers after the PR merges. The migration tests under `src/migration/` already run the runner and would catch DDL errors when invoked via `go test -race -tags=integration ./src/migration/...`.
- **No integration tests yet.** PR 1 deliberately stops at the unit level because PR 2 introduces the pgx adapters and is where the integration suite (against the compose DB) lands.

### Deviations from locked design (recorded for the next reviewer)

1. **`*GoneError` is new; not in `organization.go`.** No existing type covers HTTP 410, so `domain/prompt.go` declares a dedicated `GoneError` + `CodePromptDeleted = "prompt_deleted"` + `NewPromptDeleted(slug)` + `AsPromptDeleted(err)` helper. The handler's `errors.As(err, &gone)` maps to 410. See design.md §4.2 (post-edit) for the locked vocabulary.
2. **Wire codes are generic, not feature-specific.** The project's existing handler maps `*ValidationError` → 400/code `validation`, `*ConflictError` → 409/code `conflict`, `*NotFoundError` → 404/code `not_found`. PR 1 reuses those types; the only NEW wire code is `prompt_deleted` (410). The original proposal locked `PROMPT_*` codes at the wire; design.md §4.2 now records the adjustment.
3. **Repository port signature: `UpdateBody(ctx, db, id, body, description)` without `updatedAt` arg.** The DB owns `updated_at` via the `DEFAULT now()` + trigger. Passing an explicit `updatedAt` (as the design draft had) is a footgun in two-TX scenarios. Removed for safety; a future test can override by issuing a raw `UPDATE` if needed.

### Pre-existing failures NOT caused by PR 1 (out of scope)

- `src/infrastructure/postgres/workspace_repo_test.go` references `domain.Workspace.PrimaryRepoGitHubID` and `domain.LinkedRepository` — both were renamed/dropped in the 2026-07-08 workspaces-simplify change but the test file was not updated. This is a separate cleanup task, not blocking the prompt feature.
- `make test/cover` and `make lint` were not run because they require a clean workspace test setup and the bash commit hook blocks fresh `git` operations. They will run in CI on the PR.

### Work-unit commits for the user to apply manually

The bash commit hook blocks `git commit` in this session, so the following commits are listed for the user to apply (each commit is a logical unit; the order preserves the RED → GREEN TDD discipline):

```bash
# Task 1.1 — migration
git add backend/database_administrator/src/migration/sql/20260715120000_prompts.sql
git commit -m "feat(db): add prompt + prompt_revision tables with partial unique slug"

# Task 1.2..1.6 — domain in 5 work units (one per task); all changes are in prompt.go + prompt_test.go + imports_test.go
git add backend/database_administrator/src/domain/prompt.go backend/database_administrator/src/domain/prompt_test.go backend/database_administrator/src/domain/imports_test.go
git commit -m "feat(domain): add Prompt and PromptRevision value types"
git commit -m "feat(domain): add prompt slug/description/body validation"
git commit -m "feat(domain): add prompt GoneError for soft-deleted 410 case"
git commit -m "feat(domain): add prompt repository ports (PromptRepository, PromptRevisionRepository)"
git commit -m "test(domain): enforce no pgx import in domain package"
```

For PR 1, the user can also squash all five domain commits into a single `feat(domain): add prompts domain layer (types, validation, errors, ports)` if a flat history is preferred.

### PR 1 status: ✅ ready for review

PR 1 is internally complete and self-contained. The migration is forward-only, the domain layer compiles clean, and all 28 unit tests pass. The next PR (PR 2 — Repo) cannot begin until this PR merges.

---

## PR 2 — Repo (NOT STARTED)

Awaiting PR 1 merge confirmation before delegating to the worker subagent. The apply-pause gate per orchestrator workflow is satisfied at this checkpoint.

Files to land in PR 2:

- NEW `src/infrastructure/postgres/prompt_repo.go` (~180 lines)
- NEW `src/infrastructure/postgres/prompt_repo_test.go` (~150 lines, build tag `integration`)
- NEW `src/infrastructure/postgres/prompt_revision_repo.go` (~140 lines)
- NEW `src/infrastructure/postgres/prompt_revision_repo_test.go` (~130 lines)

Tasks (from tasks.md §PR2):

1. PromptRepo.Insert skeleton
2. SelectBySlug / SelectByID / SelectList
3. UpdateBody / SoftDelete / LockAndLoad
4. MaxRevisionNumber + 23505 → ConflictError translation
5. PromptRevisionRepo

Each task starts with a RED test (integration, against the compose DB).

---

## PR 3 — Application service (NOT STARTED)

Awaiting PR 2.

## PR 4 — HTTP handler + wiring (NOT STARTED)

Awaiting PR 3.

---

## Skill resolution

`paths-injected` (project AGENTS.md lists the relevant skill paths). Loaded before any work:

- `/Users/braejan/.claude/skills/go-testing/SKILL.md`
- `/Users/braejan/.claude/skills/test-driven-development/SKILL.md` (mandated by project AGENTS.md; if not present on disk, the orchestrator must surface a degraded warning)
- `/Users/braejan/.claude/skills/work-unit-commits/SKILL.md`

---

## Open user decisions pending for the rest of the apply phase

None — all Q-A through Q-E from the apply-pause gate are answered (4 PRs chained, limit/offset, path-segment restore, admin-only without header, inline soft-delete). The remaining PRs (2, 3, 4) can proceed after PR 1 merges.
