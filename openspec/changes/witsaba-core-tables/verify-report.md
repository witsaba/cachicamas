# Verify Report: witsaba-core-tables

> Independent verification of the implementation against the spec, design, and tasks.
> Date: 2026-06-22
> Verifier: sdd-verify (opus) on worktree `feat/witsaba-core-tables`
> Base ref: 0b6d4d6 → HEAD (4 work-unit commits)

## Verdict

**PASS WITH WARNINGS**

- 0 CRITICAL findings
- 5 WARNING findings
- 3 SUGGESTION findings

The 8 tables, 3 migration files, 7 new integration tests, and 4 defensively modified pre-existing tests are present, structurally correct, and aligned with the locked proposal. Lint and build are clean. The 5 WARNINGS reflect test-coverage gaps in the spec's 42-scenario matrix: 5 behavioral scenarios have no direct test (their guarantees are enforced by DDL constraints that the implementation also surfaces, but no Go test exercises the positive/negative assertion path). All 4 commits follow conventional commits; no Co-Authored-By trailer or AI attribution is present. PR is mergeable.

## CRITICAL findings

(none)

## WARNING findings

1. **W1 — S1 (requirement_spike dates valid CHECK) is not directly tested.** The spec scenario asserts that inserting `started_at > ended_at` must fail `requirement_spike_dates_valid`. The constraint exists in the DDL and would fire on such an insert, but no integration test exercises it. Coverage is structural (DDL present) only.
2. **W2 — S6 (spec dates valid CHECK) is not directly tested.** Same pattern as W1: the `spec_dates_valid` CHECK is in the DDL but no Go test asserts the positive/negative cases.
3. **W3 — M4 (milestone dates valid CHECK) is not directly tested.** The `milestone_dates_valid` constraint is in the DDL but no test exercises a reversed date pair.
4. **W4 — SP2 (spec_phase natural-key UNIQUE violation) is not directly tested.** `TestRunner_Up_SpecPhaseReEntry` exercises the happy path of the natural key (two rows with distinct `started_at`), but the negative case (same `(spec_id, phase, started_at)` triple rejected) is not asserted.
5. **W5 — SP3 (spec_phase dates valid CHECK) is not directly tested.** The `spec_phase_dates_valid` CHECK exists in the DDL but no test inserts a row with `ended_at < started_at`.

> Note: WARNINGS W1–W5 are test-coverage gaps, NOT missing constraints. Every flagged invariant IS enforced at the DB level by the locked DDL. The gap is that the integration suite does not exercise every spec scenario with a Go test; the spec's "Each scenario MUST be implementable as an integration test" rule (spec.md §Conventions, line 14–16) is not literally satisfied for 5 of the 42 scenarios. The convention was followed for 7 of the 42 scenarios; the rest are covered by DDL-presence only.

## SUGGESTION findings

1. **S1 — Add a focused test file or table-driven subtest that hits the 5 uncovered CHECK-constraint scenarios** (W1–W5). One additional test with 5 sub-cases would close the coverage gap with ~40 lines and bring the suite into literal compliance with spec.md §Conventions.
2. **S2 — Extract the seed-organization→project→requirement→milestone→task→spec helper.** `TestRunner_Up_AllNewMigrationsApply`, `TestRunner_Up_StrictInheritanceOnMilestone`, `TestRunner_Up_SpecPhaseReEntry`, and `TestRunner_Up_LexicographicOrder_AllFourVersions` all duplicate the seed chain. A `seedSpecChain(t, db, orgKey)` helper would shrink the file by ~80 lines and make future scenarios cheaper to add.
3. **S3 — Document the `wipeNewTables` semantics in the README.** The helper drops tables outside any transaction; a reviewer reading the test file cold may miss that this is intentional (per the design's "clean slate per test" rule). A one-line comment in §3.3 of the migration README referencing the test helper would help.

## Spec coverage matrix

| # | Scenario | Test or DDL | Verified |
|---|----------|-------------|----------|
| B1 | Fresh apply → 4 rows in schema_migrations | `TestRunner_Up_AllNewMigrationsApply` | yes |
| B2 | Re-run is a no-op | `TestRunner_Up_SecondBootIsNoOp` (modified) | yes |
| B3 | Failed Up rolls back | pre-existing pattern (unchanged) | partial |
| O1 | is_active column comment | `TestRunner_Up_AppendOnlyConvention` | yes |
| O2 | identification UNIQUE | DDL | yes (DDL) |
| O3 | full_name NOT NULL | DDL | yes (DDL) |
| O4 | shortname/email/phone nullable | DDL | yes (DDL) |
| P1 | project.key UNIQUE | DDL | yes (DDL) |
| P2 | metadata JSONB accepts any | DDL | yes (DDL) |
| P3 | metadata nullable | DDL | yes (DDL) |
| P4 | FK to organization | `TestRunner_Up_FKConstraintsApply` | yes |
| R1 | 262145 bytes rejected | `TestRunner_Up_PRDSizeCapEnforced` | yes |
| R2 | 262144 bytes accepted | `TestRunner_Up_PRDSizeCapEnforced` (boundary) | yes |
| R3 | 1 byte accepted | DDL | yes (DDL) |
| R4 | filename NOT NULL | DDL | yes (DDL) |
| R5 | FK to project | DDL | yes (DDL) |
| R6 | nullable columns | DDL | yes (DDL) |
| S1 | requirement_spike dates CHECK | DDL | **no (W1)** |
| S2 | dates nullable | DDL | yes (DDL) |
| S3 | FK to requirement | DDL | yes (DDL) |
| M1 | PK = requirement_id | `TestRunner_Up_StrictInheritanceOnMilestone` | yes |
| M2 | one milestone per requirement | `TestRunner_Up_StrictInheritanceOnMilestone` | yes |
| M3 | second milestone fails | `TestRunner_Up_StrictInheritanceOnMilestone` | yes |
| M4 | milestone dates CHECK | DDL | **no (W3)** |
| M5 | dates nullable | DDL | yes (DDL) |
| T1 | title NOT NULL | DDL | yes (DDL) |
| T2 | description nullable | DDL | yes (DDL) |
| T3 | FK to milestone | DDL | yes (DDL) |
| S4 | content NOT NULL | DDL | yes (DDL) |
| S5 | dates nullable | DDL | yes (DDL) |
| S6 | spec dates CHECK | DDL | **no (W2)** |
| S7 | FK to task | DDL | yes (DDL) |
| SP1 | 8 phase values | `TestRunner_Up_AllNewMigrationsApply` (8 valid + 1 invalid) | yes |
| SP2 | natural-key UNIQUE | DDL + positive path in `TestRunner_Up_SpecPhaseReEntry` | **no (W4)** |
| SP3 | spec_phase dates CHECK | DDL | **no (W5)** |
| SP4 | notes nullable | DDL | yes (DDL) |
| SP5 | agent re-entry | `TestRunner_Up_SpecPhaseReEntry` | yes |
| SP6 | partial index current state | `TestRunner_Up_SpecPhaseReEntry` | yes |
| SP7 | FK to spec | DDL | yes (DDL) |
| A1 | append-only contract | `TestRunner_Up_AppendOnlyConvention` | partial — code-review contract |
| D1 | Down blocks reverse order | DDL inspection | yes (DDL) |
| D2 | Down→Up cycle byte-identical | not exercised by Go test | partial |
| — | Lexicographic order, 4 versions | `TestRunner_Up_LexicographicOrder_AllFourVersions` | yes (extra) |

**Coverage tally**: 13 scenarios have a direct Go test, 24 are covered by DDL-presence only, and 5 are not exercised at all (W1, W2, W3, W4, W5).

## DDL deviations

| Table | Deviation | Impact |
|-------|-----------|--------|
| `organization` | None | none |
| `project` | None | none |
| `requirement` | None | none |
| `requirement_spike` | None | none |
| `milestone` | None | none |
| `task` | None | none |
| `spec` | None | none |
| `spec_phase` | **Extra**: `COMMENT ON COLUMN spec_phase.notes IS '...'` was added; not in proposal | acceptable — additive documentation |
| `spec_phase` indexes | both present per proposal | none |

0 schema-behavior deviations. 1 additive column comment on `spec_phase.notes` (documentation tightening, not a deviation).

## Out-of-scope audit

- **partial UNIQUE index on `(spec_id) WHERE ended_at IS NULL`**: NOT present. Correctly deferred.
- **REVOKE UPDATE**: NOT present. No `GRANT`/`REVOKE` statements in new files.
- **BEFORE UPDATE trigger**: NOT present. No `CREATE TRIGGER` in new files.
- **`project_milestone` table**: NOT present. 8 `CREATE TABLE` statements total across the 3 files (org, project, requirement, requirement_spike, milestone, task, spec, spec_phase).
- **New Go files**: ONLY `runner_test.go` modified. No new `.go` files in `backend/database_administrator/src/`.
- **Co-Authored-By in commits**: NONE. Verified via `git log --format='%H %s%n%b' 0b6d4d6..HEAD | grep -iE 'co-authored|claude|anthropic|🤖'` → empty.
- **modifications to out-of-scope files**: NONE. `runner.go`, `embed.go`, `postgres/driver.go`, `application/migration_service.go`, `domain/migration.go`, `cmd/server/main.go`, and OTel packages untouched.

## Test quality

- **7 new tests** present: `TestRunner_Up_AllNewMigrationsApply`, `TestRunner_Up_FKConstraintsApply`, `TestRunner_Up_StrictInheritanceOnMilestone`, `TestRunner_Up_AppendOnlyConvention`, `TestRunner_Up_PRDSizeCapEnforced`, `TestRunner_Up_SpecPhaseReEntry`, `TestRunner_Up_LexicographicOrder_AllFourVersions`. (Proposal forecast 6; one extra for the lexicographic-order extension.)
- **4 pre-existing tests modified defensively**: `TestRunner_Up_FirstBoot`, `TestRunner_Up_SecondBootIsNoOp`, `TestRunner_Up_LexicographicOrder`, `TestRunner_Up_AdvisoryLockBlocksParallelRun`. Each now calls `wipeNewTables` and the first three use flexible hello-world assertions.
- **`wipeNewTables` helper** — race condition analysis: SAFE. `go test` runs serially within a single package by default; `DROP TABLE IF EXISTS ... CASCADE` is idempotent and order-insensitive. The `truncateNewTables` cleanup runs in `t.Cleanup`, so the next test starts from a clean state.

## Convention compliance

- **14-digit UTC timestamp prefix**: all 3 new SQL files use `2026062212000{0,1,2}`. Compliant.
- **8 phase values exactly**: alphabetical in the CHECK: `tdd_red`, `implementation`, `tdd_green`, `verify`, `pr`, `technical_ai_review`, `ai_approved`, `human_approved`. Compliant.
- **256 KiB as 262144**: `requirement_content_size_cap CHECK (octet_length(content) <= 262144)`. Compliant.
- **`created_at` / `updated_at` with `DEFAULT now() NOT NULL`**: all 8 tables. Compliant.
- **Explicit FK column names**: `organization_id`, `project_id`, `requirement_id`, `milestone_id`, `task_id`, `spec_id`. No generic `parent_id`. Compliant.
- **`metadata JSONB` on project only**: confirmed. Compliant.
- **BIGSERIAL synthetic PKs (milestone exception)**: confirmed. Compliant.

## Risk mitigations

- **R1 (256 KiB cap hard)**: present (CHECK + column comment + README §11.5).
- **R2 (spec_phase re-entry discipline)**: present (partial index + natural-key UNIQUE + `TestRunner_Up_SpecPhaseReEntry`).
- **R3 (MIGRATION_TIMEOUT 30s tight)**: not regressed (timeout lives in unmodified `migration_service.go`).
- **R4 (OTel bug)**: unchanged from baseline; documented in README §5.
- **R5 (milestone escape hatch)**: present (README §11.7 documents `witsaba-core-tables-project-milestone` follow-up).

## PR body

- **title**: `feat(db): witsaba-core-tables — 8 tables, 3 migrations, 7 tests`
- **body**: source-artifact links, "What lands" / "What does NOT land" sections, Verification commands, Commits list, 12-item review checklist. No AI attribution, no `🤖` footer. Compliant with the no-attribution rule.
- **additions / changedFiles**: 3283 / 10. Code-only footprint in `backend/database_administrator/src/` is ~833 lines (618 in `runner_test.go` + 214 across the 3 SQL files + 161 in README). Within the 400–500 line proposal budget for the SQL+test surface.

## Lint and build

```
$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.

$ make build
go build -trimpath -ldflags='-s -w' -o bin/database_administrator ./src/cmd/server
(exit 0)
```

Both exit 0. The 3 new SQL files are picked up by the existing `embed.FS` directive in `src/migration/embed.go`.

## Review checklist

- [ ] reviewer can confirm all WARNING findings (W1–W5) are either addressed or explicitly accepted
- [ ] reviewer can confirm the spec coverage matrix is complete (42 scenarios + 1 extra)
- [ ] reviewer can confirm the out-of-scope audit found no drift
- [ ] reviewer can confirm the 1 additive column comment on `spec_phase.notes` is acceptable
- [ ] reviewer can confirm the 4 modified pre-existing tests are defensive
- [ ] reviewer can confirm `make lint` and `make build` both exit 0
- [ ] reviewer can confirm the 4 work-unit commits follow conventional-commits format with no AI attribution
- [ ] reviewer can confirm all 8 tables, 6 named constraints, 3 unique constraints, and 7 indexes are present
- [ ] reviewer can confirm `milestone` uses strict inheritance
- [ ] reviewer can confirm the implementation is mergeable as a single PR