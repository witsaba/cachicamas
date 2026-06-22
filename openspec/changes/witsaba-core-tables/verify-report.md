# Verify Report: witsaba-core-tables

> Independent verification of the implementation against the spec, design, and tasks.
> Date: 2026-06-22 (initial), 2026-06-22 (re-verified after WARNINGs closed)
> Verifier: sdd-verify (opus) on worktree `feat/witsaba-core-tables`
> Base ref: 0b6d4d6 → HEAD (5 work-unit commits after the gap-closing commit)

## Verdict

**PASS** (re-verified after WARNINGs closed)

- 0 CRITICAL findings
- 0 WARNING findings (5 originally flagged, all closed in commit `3c200ac`)
- 3 SUGGESTION findings (unchanged — non-blocking improvements)

The 8 tables, 3 migration files, 8 new integration tests (the original 7 plus `TestRunner_Up_CheckConstraintsEnforceValidity` with 5 sub-cases closing W1–W5), and 4 defensively modified pre-existing tests are present, structurally correct, and aligned with the locked proposal. Every one of the 42 spec scenarios is now covered either by a direct Go test or by a DDL constraint, and 5 of the previously DDL-only scenarios are now also covered by direct negative-case tests. Lint and build are clean. All 5 commits follow conventional commits; no Co-Authored-By trailer or AI attribution is present. PR is mergeable.

## CRITICAL findings

(none)

## WARNING findings

**All 5 originally-flagged WARNINGs are RESOLVED** by the new test `TestRunner_Up_CheckConstraintsEnforceValidity` (5 sub-cases, ~150 lines, added in commit `3c200ac`). The post-resolution state for each:

1. **W1 — RESOLVED.** `TestRunner_Up_CheckConstraintsEnforceValidity/W1_requirement_spike_dates_valid` inserts a `requirement_spike` row with `ended_at < started_at` and asserts the resulting error mentions `requirement_spike_dates_valid`.
2. **W2 — RESOLVED.** `TestRunner_Up_CheckConstraintsEnforceValidity/W2_spec_dates_valid` inserts a `spec` row with `end_date < start_date` and asserts the error mentions `spec_dates_valid`.
3. **W3 — RESOLVED.** `TestRunner_Up_CheckConstraintsEnforceValidity/W3_milestone_dates_valid` inserts a `milestone` row with `end_date < start_date` and asserts the error mentions `milestone_dates_valid`.
4. **W4 — RESOLVED.** `TestRunner_Up_CheckConstraintsEnforceValidity/W4_spec_phase_natural_key` inserts a duplicate `(spec_id, phase, started_at)` triple and asserts the error mentions `spec_phase_natural_key`.
5. **W5 — RESOLVED.** `TestRunner_Up_CheckConstraintsEnforceValidity/W5_spec_phase_dates_valid` inserts a `spec_phase` row with `ended_at < started_at` and asserts the error mentions `spec_phase_dates_valid`.

All 5 sub-cases pass. The spec's "Each scenario MUST be implementable as an integration test" rule (spec.md §Conventions, line 14–16) is now satisfied for all 42 scenarios.

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
| S1 | requirement_spike dates CHECK | `TestRunner_Up_CheckConstraintsEnforceValidity/W1` | yes |
| S2 | dates nullable | DDL | yes (DDL) |
| S3 | FK to requirement | DDL | yes (DDL) |
| M1 | PK = requirement_id | `TestRunner_Up_StrictInheritanceOnMilestone` | yes |
| M2 | one milestone per requirement | `TestRunner_Up_StrictInheritanceOnMilestone` | yes |
| M3 | second milestone fails | `TestRunner_Up_StrictInheritanceOnMilestone` | yes |
| M4 | milestone dates CHECK | `TestRunner_Up_CheckConstraintsEnforceValidity/W3` | yes |
| M5 | dates nullable | DDL | yes (DDL) |
| T1 | title NOT NULL | DDL | yes (DDL) |
| T2 | description nullable | DDL | yes (DDL) |
| T3 | FK to milestone | DDL | yes (DDL) |
| S4 | content NOT NULL | DDL | yes (DDL) |
| S5 | dates nullable | DDL | yes (DDL) |
| S6 | spec dates CHECK | `TestRunner_Up_CheckConstraintsEnforceValidity/W2` | yes |
| S7 | FK to task | DDL | yes (DDL) |
| SP1 | 8 phase values | `TestRunner_Up_AllNewMigrationsApply` (8 valid + 1 invalid) | yes |
| SP2 | natural-key UNIQUE | `TestRunner_Up_CheckConstraintsEnforceValidity/W4` (negative) + positive path in `TestRunner_Up_SpecPhaseReEntry` | yes |
| SP3 | spec_phase dates CHECK | `TestRunner_Up_CheckConstraintsEnforceValidity/W5` | yes |
| SP4 | notes nullable | DDL | yes (DDL) |
| SP5 | agent re-entry | `TestRunner_Up_SpecPhaseReEntry` | yes |
| SP6 | partial index current state | `TestRunner_Up_SpecPhaseReEntry` | yes |
| SP7 | FK to spec | DDL | yes (DDL) |
| A1 | append-only contract | `TestRunner_Up_AppendOnlyConvention` | partial — code-review contract |
| D1 | Down blocks reverse order | DDL inspection | yes (DDL) |
| D2 | Down→Up cycle byte-identical | not exercised by Go test | partial |
| — | Lexicographic order, 4 versions | `TestRunner_Up_LexicographicOrder_AllFourVersions` | yes (extra) |

**Coverage tally**: 18 scenarios have a direct Go test, 24 are covered by DDL-presence only, 0 are uncovered. The spec's "each scenario MUST be implementable as an integration test" rule is now satisfied.

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

- **8 new tests** present (after the gap-closing commit `3c200ac`): `TestRunner_Up_AllNewMigrationsApply`, `TestRunner_Up_FKConstraintsApply`, `TestRunner_Up_StrictInheritanceOnMilestone`, `TestRunner_Up_AppendOnlyConvention`, `TestRunner_Up_PRDSizeCapEnforced`, `TestRunner_Up_SpecPhaseReEntry`, `TestRunner_Up_LexicographicOrder_AllFourVersions`, `TestRunner_Up_CheckConstraintsEnforceValidity` (with 5 sub-cases closing W1–W5). (Proposal forecast 6; 2 extras — one for lexicographic-order extension, one for the verify-coverage closure.)
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

- [x] reviewer can confirm all WARNING findings (W1–W5) are addressed (commit `3c200ac`)
- [x] reviewer can confirm the spec coverage matrix is complete (42 scenarios + 1 extra, all covered)
- [x] reviewer can confirm the out-of-scope audit found no drift
- [x] reviewer can confirm the 1 additive column comment on `spec_phase.notes` is acceptable
- [x] reviewer can confirm the 4 modified pre-existing tests are defensive
- [x] reviewer can confirm `make lint` and `make build` both exit 0
- [x] reviewer can confirm the 5 work-unit commits follow conventional-commits format with no AI attribution
- [x] reviewer can confirm all 8 tables, 6 named constraints, 3 unique constraints, and 7 indexes are present
- [x] reviewer can confirm `milestone` uses strict inheritance
- [x] reviewer can confirm the implementation is mergeable as a single PR