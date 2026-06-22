# Tasks: witsaba-core-tables

> Ordered, verifiable work items for `sdd-apply`. Each task is one work-unit commit (or one logical commit-group within a work unit). The change lands as a single PR per cached `delivery_strategy: single-pr`, but the four work units below correspond 1:1 to the four commits in `design.md §7`.

## Conventions

- **Strict TDD is ON** (per `openspec/AGENTS.md` lines 42-57 and `sdd-init/cachicamas`): tests are written FIRST, then the minimum production code (SQL) to make them GREEN, then REFACTOR. The red step is mandatory.
- Each task ends with a `Verification` block: a concrete command, expected exit code, and expected output shape.
- Each task ends with a `Commit` block: the conventional-commit message to use. **No `Co-Authored-By` trailer. No AI attribution.** (`openspec/AGENTS.md` line 19 + line 84.)
- Each task ends with a `Spec` block: the spec.md Given/When/Then scenarios the task satisfies. Traceability is mandatory.
- Working directory for every `cd`: `backend/database_administrator`.
- `INTEGRATION=1` gates every new test (reuses `integrationRunnerDB(t)` at `runner_test.go:42-71`). Without it, the new tests SKIP — that is correct behavior.
- The test cleanup recipe is **TRUNCATE-the-new-tables CASCADE in `t.Cleanup`** (per `design.md §3.2`). Do NOT delete from `public.schema_migrations`; do NOT wrap the test in BEGIN/ROLLBACK (deadlocks against the runner's own transactions).
- A Go developer with zero prior context must be able to execute each task by reading only the task body, this Conventions section, and the file paths cited.

## Work-unit map

The four commits below are the work units from `design.md §7`. Each commit is independently green and independently revertable.

| Commit | Files | Tests added/extended |
|---|---|---|
| C1 | `sql/20260622120000_orgs_and_projects.sql` | T1 + T2 |
| C2 | `sql/20260622120001_requirements_and_milestones.sql` | T3 + T5 |
| C3 | `sql/20260622120002_tasks_and_specs.sql` | T4 + T6 + extend `TestRunner_Up_LexicographicOrder` |
| C4 | `src/migration/README.md` (docs only) | none (refactor / docs) |

Tasks 1-3 land the SQL + tests in three strict TDD pairs (each commit is one RED-GREEN-REFACTOR cycle). Task 4 is the docs commit (refactor / polish).

---

## Task 1: RED — add failing tests for organization + project behavior

**File**: `backend/database_administrator/src/migration/runner_test.go` (append two new `TestRunner_Up_*` functions; no other edits).

**Depends on**: Task 0 (branch + compose stack running). Run `make test/integration` once before this task to confirm the baseline is green: `INTEGRATION=1 go test -race -v ./...` passes the four existing tests (`TestRunner_Up_FirstBoot`, `TestRunner_Up_SecondBootIsNoOp`, `TestRunner_Up_LexicographicOrder`, `TestRunner_Up_AdvisoryLockBlocksParallelRun`).

**Adds**:

1. `TestRunner_Up_AllNewMigrationsApply` — Given a wiped `public.schema_migrations`. When `runner.Up(ctx)`. Then:
   - The applied slice has length 4 (versions `20260621120000`, `20260622120000`, `20260622120001`, `20260622120002`).
   - The applied slice is in lexicographic order.
   - Every new table exists in `public` — query `information_schema.tables WHERE table_schema = 'public' AND table_name IN ('organization','project','requirement','requirement_spike','milestone','task','spec','spec_phase')` and assert exactly 8 rows.
   - The `spec_phase` CHECK accepts exactly the 8 locked phase values: insert each of the 8 successfully (use 8 separate `INSERT INTO spec_phase (spec_id, phase) VALUES (?, ?)` after seeding a `spec` row), then assert an INSERT with `phase = 'NOT_A_PHASE'` fails with the `spec_phase_phase_check` constraint violation.

2. `TestRunner_Up_FKConstraintsApply` — Given the runner just applied all 4 migrations. When `INSERT INTO project (organization_id, key, full_name) VALUES (99999, 'orphan', 'orphan')` is attempted. Then the insert returns an error whose message contains `project_organization_id_fkey` (substring match; pgx error wording can vary).

Both tests use the TRUNCATE-CASCADE cleanup recipe from `design.md §3.2`:

```go
t.Cleanup(func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _, _ = db.ExecContext(ctx,
        "TRUNCATE TABLE spec_phase, spec, task, milestone, requirement_spike, requirement, project, organization CASCADE")
})
```

Both tests call `resetSchemaMigrations(t, db)` at the top, then `runner.Up(ctx)` with a 30 s timeout (same as existing tests at `runner_test.go:134`).

**Spec**: satisfies B1 (fresh apply has 4 rows in order), P4 (FK to organization enforced), SP1 (phase check accepts the 8 values, rejects others).

**Verification** (RED step):

```bash
cd backend/database_administrator
INTEGRATION=1 go test -race -v -run 'TestRunner_Up_AllNewMigrationsApply|TestRunner_Up_FKConstraintsApply' ./src/migration/...
```

Expected: both tests FAIL. `TestRunner_Up_AllNewMigrationsApply` fails because `len(applied) != 4` (only the hello-world row exists; `20260622120000_orgs_and_projects.sql` is not in the embed.FS yet). `TestRunner_Up_FKConstraintsApply` fails because `project` does not exist yet (`relation "project" does not exist`).

`make test` (no `INTEGRATION=1`) must still pass — the new tests are skipped via `integrationRunnerDB`.

**Commit**: do **NOT** commit yet. Task 1 is the RED step of one RED-GREEN-REFACTOR cycle. The GREEN step is Task 2; the commit happens after Task 2 lands.

---

## Task 2: GREEN — write migration file 1 (orgs + projects)

**File**: `backend/database_administrator/src/migration/sql/20260622120000_orgs_and_projects.sql` (new).

**Depends on**: Task 1 (tests are written and currently FAIL). Compose stack still running.

**Adds** — single SQL file with the goose v3 single-file idiom (`design.md §2.1` + `migration/README.md:51-88`):

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE organization (
    id              BIGSERIAL    PRIMARY KEY,
    shortname       TEXT,
    full_name       TEXT         NOT NULL,
    identification  TEXT         NOT NULL UNIQUE,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    email           TEXT,
    phone           TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
COMMENT ON COLUMN organization.is_active IS
    'UPDATE-in-place: this column is the ONLY mutation allowed on organization rows.';

CREATE TABLE project (
    id              BIGSERIAL    PRIMARY KEY,
    organization_id BIGINT       NOT NULL REFERENCES organization(id),
    key             TEXT         NOT NULL UNIQUE,
    full_name       TEXT         NOT NULL,
    start_date      DATE,
    end_date        DATE,
    metadata        JSONB,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_project_organization_id ON project(organization_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE project;
DROP TABLE organization;
-- +goose StatementEnd
```

**Spec**: O1 (column comment on `is_active`), O2 (`identification UNIQUE`), O3 (`full_name NOT NULL`), O4 (nullable columns), P1 (`key UNIQUE`), P2 (`metadata` accepts any JSON), P3 (`metadata NULL` allowed), P4 (FK enforced).

**Verification** (GREEN step):

```bash
cd backend/database_administrator
make build                                      # confirms embed.FS compiles
INTEGRATION=1 go test -race -v -run 'TestRunner_Up_AllNewMigrationsApply|TestRunner_Up_FKConstraintsApply' ./src/migration/...
INTEGRATION=1 go test -race -v ./src/migration/...   # full migration package, no other test regresses
```

Expected: both new tests PASS. The four existing integration tests still PASS. `make test` (no `INTEGRATION=1`) still passes (integration tests skip).

**Refactor pass**: re-read the test body; confirm assertion names are stable; confirm the TRUNCATE CASCADE list contains only the tables this commit owns (`organization`, `project`); trim the cleanup list to just those two tables to keep the test surface honest (per `design.md §3.2` last paragraph: only the tables the test populates need to be in the TRUNCATE list, but defensive full CASCADE is acceptable too — pick one and stick with it across all four commits for consistency).

Re-run `INTEGRATION=1 go test -race -v ./src/migration/...` after the refactor — must still be green.

**Commit** (work-unit commit C1 from `design.md §7`):

```bash
cd backend/database_administrator
git add src/migration/sql/20260622120000_orgs_and_projects.sql src/migration/runner_test.go
git commit -m "feat(db): add orgs and projects tables with FK integration test"
```

Conventional commit. No `Co-Authored-By`. No AI attribution. Single commit for the whole RED-GREEN-REFACTOR cycle of work unit 1.

---

## Task 3: RED + GREEN — requirement + spike + strict-inherited milestone

Two sub-tasks, one work-unit commit (C2).

### 3a. RED — add failing tests

**File**: `backend/database_administrator/src/migration/runner_test.go` (append two more tests).

**Adds**:

1. `TestRunner_Up_StrictInheritanceOnMilestone` — Given the runner just applied all 4 migrations and a seeded `project` + `requirement`. When:
   - `SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey) WHERE i.indrelid = 'milestone'::regclass AND i.indisprimary` is run. Then the result has exactly one row whose `attname = 'requirement_id'` (no synthetic `id`).
   - `INSERT INTO milestone (requirement_id, title) VALUES (1, 'M1')` succeeds.
   - `INSERT INTO milestone (requirement_id, title) VALUES (1, 'M1 dup')` fails with a `milestone_pkey` violation.

2. `TestRunner_Up_PRDSizeCapEnforced` — Given the runner just applied all 4 migrations and a seeded `project` + `requirement` parent row. When:
   - `INSERT INTO requirement (project_id, filename, content) VALUES (1, 'edge.md', repeat('x', 262144))` is run. Then it succeeds (boundary).
   - `INSERT INTO requirement (project_id, filename, content) VALUES (1, 'big.md', repeat('x', 262145))` is run. Then it fails with `requirement_content_size_cap` in the error message.

Use the TRUNCATE CASCADE cleanup recipe.

### 3b. GREEN — write migration file 2

**File**: `backend/database_administrator/src/migration/sql/20260622120001_requirements_and_milestones.sql` (new).

**Depends on**: Task 2 (commit C1 is on the branch).

**Adds** — single SQL file with the goose v3 idiom. Per `design.md §2.2`:

- `requirement` first (`project_id BIGINT NOT NULL REFERENCES project(id)`, `filename TEXT NOT NULL`, `content TEXT NOT NULL`, plus nullable `git_repository_url`, `analysis_result`, `is_technically_viable`; CHECK `requirement_content_size_cap CHECK (octet_length(content) <= 262144)`; COMMENT ON COLUMN `requirement.content IS 'Max PRD size: 256 KiB (262144 bytes), enforced via CHECK constraint.'`; `idx_requirement_project_id`).
- `requirement_spike` second (`requirement_id BIGINT NOT NULL REFERENCES requirement(id)`, nullable `started_at`/`ended_at`/`outcome`/`findings`; CHECK `requirement_spike_dates_valid CHECK (ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at)`; `idx_requirement_spike_requirement_id`).
- `milestone` last, strict-inherited: `requirement_id BIGINT PRIMARY KEY REFERENCES requirement(id)` (PK = FK, no synthetic `id`); `title TEXT NOT NULL`; nullable `description`, `start_date`, `end_date`; CHECK `milestone_dates_valid CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)`.

Down block in reverse: `DROP TABLE milestone; DROP TABLE requirement_spike; DROP TABLE requirement;` (children-first; strict-inheritance table first within a level).

Wrap Up and Down each in `-- +goose StatementBegin` / `-- +goose StatementEnd`.

**Spec**: R1, R2, R3, R4, R5, R6 (requirement + cap + FK + nullables), S1, S2, S3 (spike + FK + multi-cardinality), M1, M2, M3, M4, M5 (strict inheritance + 1:1 + date CHECK + nullables).

**Verification**:

```bash
cd backend/database_administrator
make build
INTEGRATION=1 go test -race -v -run 'TestRunner_Up_StrictInheritanceOnMilestone|TestRunner_Up_PRDSizeCapEnforced' ./src/migration/...
INTEGRATION=1 go test -race -v ./src/migration/...
make test    # no INTEGRATION; all four existing tests must still pass
```

Expected: the two new tests PASS. The four prior new tests still PASS. The four pre-existing tests still PASS. `make test` is green.

**Commit** (work-unit commit C2):

```bash
cd backend/database_administrator
git add src/migration/sql/20260622120001_requirements_and_milestones.sql src/migration/runner_test.go
git commit -m "feat(db): add requirements, spikes, and strict-inherited milestone"
```

Conventional commit. No `Co-Authored-By`. No AI attribution.

---

## Task 4: RED + GREEN — task + spec + spec_phase + lexicographic-order extension

Three sub-tasks, one work-unit commit (C3). This is the largest task because it covers three tables and the agent-first re-entry pattern.

### 4a. RED — add failing tests + extend the lexicographic-order test

**File**: `backend/database_administrator/src/migration/runner_test.go` (append two tests, extend one).

**Adds**:

1. `TestRunner_Up_AppendOnlyConvention` — Given the runner just applied all 4 migrations. When `SELECT col_description('organization'::regclass, attnum) FROM pg_attribute WHERE attrelid = 'organization'::regclass AND attname = 'is_active'` is run. Then the returned comment is exactly `'UPDATE-in-place: this column is the ONLY mutation allowed on organization rows.'` (verbatim; locked decision from `proposal.md §1`).

2. `TestRunner_Up_SpecPhaseReEntry` — Given the runner just applied all 4 migrations and a seeded `project`, `requirement`, `milestone`, `task`, `spec`. When the agent runs the canonical re-entry pattern from `proposal.md §"Agent-first re-entry pattern"`:
   - `INSERT INTO spec_phase (spec_id, phase, notes) VALUES (?, 'tdd_red', 'initial red')`.
   - `UPDATE spec_phase SET ended_at = now() WHERE spec_id = ? AND ended_at IS NULL`.
   - `INSERT INTO spec_phase (spec_id, phase, notes) VALUES (?, 'technical_ai_review', 'reviewing')`.
   - `UPDATE spec_phase SET ended_at = now() WHERE spec_id = ? AND ended_at IS NULL`.
   - `time.Sleep(10 * time.Millisecond)` (force non-equal `started_at` microseconds between the two `tdd_red` rows).
   - `INSERT INTO spec_phase (spec_id, phase, notes) VALUES (?, 'tdd_red', 're-entering after AI review found X')`.

   Then:
   - All four inserts succeed.
   - `SELECT count(*) FROM spec_phase WHERE spec_id = ?` returns 3.
   - `SELECT count(*) FROM spec_phase WHERE spec_id = ? AND ended_at IS NULL` returns 1 (partial index `idx_spec_phase_current_state` serves it — assert via `EXPLAIN` if cheap, otherwise just assert the count).
   - The two `tdd_red` rows have distinct `started_at` (proves the natural-key UNIQUE did not block the re-entry).

3. **Extend `TestRunner_Up_LexicographicOrder`** at `runner_test.go:217-271`. After the existing assertions, also seed `version_id = 20260622120000`, `20260622120001`, `20260622120002` as already-applied bookkeeping rows (mirroring the existing `20260101000000` pattern at lines 239-242), then call `runner.Up(ctx)` again and assert the applied slice is empty (all 4 new versions are already recorded). This proves the runner does not re-apply the 3 new files regardless of lexicographic ordering among them.

   Alternative simpler extension: after the existing `Up()` succeeds, query `public.schema_migrations` for `version_id IN (20260621120000, 20260622120000, 20260622120001, 20260622120002)` and assert all 4 are present and ordered. Pick whichever is easier to reason about; both satisfy `design.md §3.3` extension requirement.

Use the TRUNCATE CASCADE cleanup recipe.

### 4b. GREEN — write migration file 3

**File**: `backend/database_administrator/src/migration/sql/20260622120002_tasks_and_specs.sql` (new).

**Depends on**: Task 3 (commit C2 is on the branch).

**Adds** — single SQL file with the goose v3 idiom. Per `design.md §2.3`:

- `task` first (`milestone_id BIGINT NOT NULL REFERENCES milestone(id)`, `title TEXT NOT NULL`, nullable `description`; `idx_task_milestone_id`).
- `spec` second (`task_id BIGINT NOT NULL REFERENCES task(id)`, `content TEXT NOT NULL`, nullable `start_date`, `end_date`; CHECK `spec_dates_valid CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)`; `idx_spec_task_id`).
- `spec_phase` last (`spec_id BIGINT NOT NULL REFERENCES spec(id)`, `phase TEXT NOT NULL`, `started_at TIMESTAMPTZ NOT NULL DEFAULT now()`, nullable `ended_at`/`notes`; CHECK `spec_phase_phase_check CHECK (phase IN ('tdd_red','implementation','tdd_green','verify','pr','technical_ai_review','ai_approved','human_approved'))` — exactly 8 values; UNIQUE `spec_phase_natural_key UNIQUE (spec_id, phase, started_at)`; CHECK `spec_phase_dates_valid CHECK (ended_at IS NULL OR ended_at >= started_at)`; `idx_spec_phase_spec_id`; partial `idx_spec_phase_current_state ON spec_phase(spec_id) WHERE ended_at IS NULL`).

Down block in reverse: `DROP TABLE spec_phase; DROP TABLE spec; DROP TABLE task;` (children-first).

Wrap Up and Down each in `-- +goose StatementBegin` / `-- +goose StatementEnd`.

**CRITICAL — what NOT to add**: the partial UNIQUE index `CREATE UNIQUE INDEX idx_spec_phase_one_open_per_spec ON spec_phase(spec_id) WHERE ended_at IS NULL;` is **NOT** in scope for v1 (`design.md §6.2` decision is explicit and the rationale cites locked proposal A2). Do NOT add it.

**Spec**: T1, T2, T3 (task constraints), S4, S5, S6, S7 (spec constraints), SP1, SP2, SP3, SP4, SP5, SP6, SP7 (spec_phase + agent re-entry + partial index).

**Verification**:

```bash
cd backend/database_administrator
make build
INTEGRATION=1 go test -race -v -run 'TestRunner_Up_AppendOnlyConvention|TestRunner_Up_SpecPhaseReEntry|TestRunner_Up_LexicographicOrder' ./src/migration/...
INTEGRATION=1 go test -race -v ./src/migration/...
make test    # no INTEGRATION; existing tests still pass
```

Expected: the two new tests PASS; the extended lexicographic test PASSES. All previously added tests still PASS. `make test` is green.

**Refactor pass**: skim all four new test functions for consistent naming, consistent TRUNCATE list, consistent assertion style; tighten any obvious smells; re-run the full suite.

**Commit** (work-unit commit C3):

```bash
cd backend/database_administrator
git add src/migration/sql/20260622120002_tasks_and_specs.sql src/migration/runner_test.go
git commit -m "feat(db): add tasks, specs, and append-only spec_phase history"
```

Conventional commit. No `Co-Authored-By`. No AI attribution.

---

## Task 5: REFACTOR — migration README documentation

**File**: `backend/database_administrator/src/migration/README.md` (append a new section; do NOT edit existing sections).

**Depends on**: Tasks 1-4 (the schema is in place; the docs reflect it).

**Adds** — append a `## 11. Schema overview` section that documents:

- The 8 new tables in dependency order: `organization`, `project`, `requirement`, `requirement_spike`, `milestone` (strict inheritance — PK = FK = `requirement_id`, 1:1), `task`, `spec`, `spec_phase` (synthetic PK + UNIQUE natural key + partial index).
- Cardinality map (1:1 for `requirement ↔ milestone`; 1:N for everything else).
- Append-only convention: in-place `UPDATE` is allowed ONLY on `organization.is_active`; the column comment on `is_active` is the documented contract; DB-level enforcement (REVOKE / trigger) is deferred to follow-up changes `witsaba-core-tables-append-only-enforcement` and `witsaba-core-tables-project-milestone` (cite the `design.md §6.2` and §6.5 decisions).
- 256 KiB PRD cap: documented at `requirement.content`; enforced via CHECK `requirement_content_size_cap`; escape hatch is `ALTER TABLE ... DROP CONSTRAINT ... ADD CONSTRAINT ... CHECK (octet_length(content) <= <new_cap>)`.
- The agent-first re-entry SQL pattern (the canonical 4-statement transition from `proposal.md §"Agent-first re-entry pattern"`).

This is a docs-only commit. No behavior change. No new tests required (the behavior is already covered by Tasks 1-4).

**Spec**: A1 (append-only documentation contract). The README is the human-facing companion to the spec scenario A1.

**Verification**:

```bash
cd backend/database_administrator
make test      # still green; docs change does not affect tests
make lint      # still green
INTEGRATION=1 make test/integration   # full integration suite still green
ls src/migration/sql/                 # confirms 4 SQL files present
ls src/migration/                     # confirms runner.go, runner_test.go, embed.go unchanged
```

Expected: every command exits 0. The 3 new SQL files are in dependency order. The 6 new test functions are present. No new top-level Go dependencies.

**Commit** (work-unit commit C4):

```bash
cd backend/database_administrator
git add src/migration/README.md
git commit -m "chore(db): document witsaba-core-tables migration in migration README"
```

Conventional commit. No `Co-Authored-By`. No AI attribution.

---

## Task 6: Local CI — full quality gates

**Depends on**: Tasks 1-5 (all four commits are on the branch).

**Runs** (in this order; each MUST exit 0):

```bash
cd backend/database_administrator
make fmt          # gofmt + goimports; auto-fixes style issues in place
make lint         # go vet + golangci-lint v2.9.0
make test         # race-enabled unit tests, no INTEGRATION (integration tests skip)
make build        # compiles ./bin/database_administrator
make test/integration   # boots compose postgres, runs full integration suite with INTEGRATION=1
```

Expected: every target exits 0. The `migration.up applied_count=4` slog line appears once during the integration suite. No `panic`, no `DATA RACE`, no `lint` warnings, no `vet` findings.

If `make fmt` rewrites anything, stage and amend into the relevant commit (do NOT create a new "formatting" commit — that violates work-unit discipline). If `make lint` surfaces a real issue, fix the issue in the relevant commit (not a new commit) and re-run all gates.

**Verification**: paste the full `make test` + `make test/integration` output into the PR description.

**Commit**: no new commit. This task may produce amended commits if `make fmt` rewrites anything.

---

## Task 7: Deliver — push branch and open PR

**Depends on**: Task 6 (all gates green).

**Runs**:

```bash
cd /Users/braejan/workspace/witsaba/repositories/cachicamas/.worktrees/witsaba-core-tables
git push -u origin feat/witsaba-core-tables
gh pr create --base main --title "feat(db): witsaba-core-tables — 8 tables, 3 migrations, 6 tests" --body-file /tmp/witsaba-core-tables-pr-body.md
```

**PR body** must include (template follows; replace `<URL>` with the proposal / spec / design / tasks URLs after commit):

```markdown
## Summary

This PR lands the witsaba framework's 8-table core schema as 3 goose
migration files plus 6 new integration tests. Schema is agent-first
and append-mostly by default.

## Source artifacts

- Proposal: openspec/changes/witsaba-core-tables/proposal/proposal.md
- Spec:     openspec/changes/witsaba-core-tables/specs/witsaba-core-tables/spec.md
- Design:   openspec/changes/witsaba-core-tables/design.md
- Tasks:    openspec/changes/witsaba-core-tables/tasks.md

## What lands

- 3 new SQL files in `backend/database_administrator/src/migration/sql/`:
  - `20260622120000_orgs_and_projects.sql` (organization, project)
  - `20260622120001_requirements_and_milestones.sql` (requirement, requirement_spike, milestone)
  - `20260622120002_tasks_and_specs.sql` (task, spec, spec_phase)
- 6 new integration tests in `runner_test.go`:
  - TestRunner_Up_AllNewMigrationsApply
  - TestRunner_Up_FKConstraintsApply
  - TestRunner_Up_StrictInheritanceOnMilestone
  - TestRunner_Up_AppendOnlyConvention
  - TestRunner_Up_PRDSizeCapEnforced
  - TestRunner_Up_SpecPhaseReEntry
- Extended TestRunner_Up_LexicographicOrder to cover all 4 versions.
- Docs: §11 Schema overview in `src/migration/README.md`.

## What does NOT land

- No new Go files. No edits to runner.go, embed.go, postgres/driver.go,
  or any application code.
- No new top-level Go dependencies.
- No DB-level append-only enforcement (REVOKE / trigger) — deferred to
  follow-up change `witsaba-core-tables-append-only-enforcement`.
- No partial UNIQUE index on `(spec_id) WHERE ended_at IS NULL` —
  deferred to the same follow-up change.
- No `project_milestone` table — deferred to
  `witsaba-core-tables-project-milestone`.

## Verification

```
$ make test
$ make test/integration
$ make lint
$ make build
```

All four exit 0 on the latest commit.

## Review checklist

- [ ] reviewer can confirm 8 tables exist with the columns and constraints listed in proposal.md §"The schema"
- [ ] reviewer can confirm `milestone` uses strict inheritance (PK = FK = `requirement_id`, no synthetic `id`)
- [ ] reviewer can confirm `requirement_content_size_cap` is `octet_length(content) <= 262144` AND the column comment is on `requirement.content`
- [ ] reviewer can confirm `spec_phase` has the 3 constraints (phase_check with 8 values, natural_key UNIQUE, dates_valid) and the 2 indexes
- [ ] reviewer can confirm the 3 SQL files are in dependency order: 20260622120000, 20260622120001, 20260622120002
- [ ] reviewer can confirm Down blocks reverse in dependency order
- [ ] reviewer can confirm 6 new test names match design.md §3.3
- [ ] reviewer can confirm the partial UNIQUE index on `(spec_id) WHERE ended_at IS NULL` is NOT added
- [ ] reviewer can confirm no source Go file is modified (runner.go, embed.go, driver.go, application/migration_service.go, domain/migration.go, otel/*, cmd/server/main.go)
- [ ] reviewer can confirm no Co-Authored-By trailer and no AI attribution in any commit
- [ ] reviewer can confirm PR size is within the 400-line default review budget (~400-500 lines total per proposal.md §"PR size forecast")
```

**Verification**:

```bash
git log --oneline main..HEAD           # 4 work-unit commits
gh pr view --web                       # PR is open and renders correctly
gh pr checks                           # all CI checks pending or green
```

Expected: 4 commits ahead of `main`. PR is open. CI starts.

**Commit**: no new commit. This task is delivery only.

---

## Out of scope (explicit follow-up changes)

These are NOT in this PR. Each is a separate `sdd-new` invocation when the time comes.

- **`witsaba-core-tables-append-only-enforcement`** — adds the partial UNIQUE index on `(spec_id) WHERE ended_at IS NULL`, REVOKE UPDATE on append-only tables from `queen`, and a unit test asserting the agent re-entry pattern fails when the prior row is not closed. Tracked at `design.md §6.2`.
- **`witsaba-core-tables-project-milestone`** — adds a `project_milestone` table for the future "deliverable per release channel" cardinality escape hatch. Tracked at `design.md §6.5`.

If a wiki `Incompleteness-Log.md` exists at `wiki/Incompleteness-Log.md`, add entries for these two follow-ups in the same PR commit as Task 5 (the docs commit):

```
- [YYYY-MM-DD] witsaba-core-tables follow-ups [missing] DB-level append-only enforcement
  (REVOKE UPDATE, partial UNIQUE on spec_phase current_state) deferred — proposed:
  witsaba-core-tables-append-only-enforcement.
- [YYYY-MM-DD] witsaba-core-tables follow-ups [missing] project_milestone escape hatch
  for multi-milestone projects deferred — proposed:
  witsaba-core-tables-project-milestone.
```

(Per `~/.claude/CLAUDE.md` "Wiki & Documentation Discipline" rule 1.)

---

## Review checklist

- [ ] reviewer can confirm each task has a Verification section with a concrete command
- [ ] reviewer can confirm the RED-GREEN-REFACTOR ordering is preserved (Task 1 RED, Task 2 GREEN, Task 3 RED+GREEN, Task 4 RED+GREEN, Task 5 REFACTOR/docs)
- [ ] reviewer can confirm each task has a Commit section with a conventional commit message and NO `Co-Authored-By`
- [ ] reviewer can confirm each task has a Spec section linking it to Given/When/Then scenarios
- [ ] reviewer can confirm the work-unit map (C1-C4) matches `design.md §7`
- [ ] reviewer can confirm the task count is 7 (small enough for a single PR with 4 work-unit commits)
- [ ] reviewer can confirm no source Go file is modified (Tasks 1-5 only touch `runner_test.go`, the 3 new SQL files, and `README.md`)
- [ ] reviewer can confirm the partial UNIQUE index on `(spec_id) WHERE ended_at IS NULL` is explicitly listed as out of scope
- [ ] reviewer can confirm the agent-first re-entry pattern from `proposal.md §"Agent-first re-entry pattern"` is exercised in Task 4
- [ ] reviewer can confirm `wiki/Incompleteness-Log.md` (if it exists) gets two follow-up entries per the wiki discipline rule
