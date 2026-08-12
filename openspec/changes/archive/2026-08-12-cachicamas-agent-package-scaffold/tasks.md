# Tasks: `cachicamas-agent-package-scaffold` (AG-03)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~750–950 (4 new guard/doc files ~550–650 lines + L1 fix ~40 lines + spec delta ~90 lines already written + evidence/PR notes) |
| 400-line budget risk | High (against default) / Medium (against this run's 1000-line budget) |
| Chained PRs recommended | No — single PR per proposal/design; nodes are not independently mergeable (L1 fix must land with `src/agent/`'s creation) |
| Suggested split | Single PR, `size:exception` |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Medium

Rationale: proposal Risk #7 and design's own scope (three guards + one cross-cutting L1 edit) make this inherently one PR — `src/agent/` and the L1 self-reference fix are causally coupled (design.md File Changes: "must land before AG-03.2's first red … or `make test` breaks mid-stack"), so no chain boundary is safe. The user pre-authorized `size:exception` against a 1000-line budget for this exact reason; this run's delivery context confirms `exception-ok`, so no further decision is needed before apply.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 (whole change) | Create `src/agent/`, its doc contract, both guards, and the coupled L1 fix | PR 1 (only) | `cd backend/agent && go test -race -v ./src/agent/... ./src/ai/...` | `cd backend/agent && make test && make lint` | Delete `backend/agent/src/agent/`; revert `backend/agent/src/ai/import_boundary_test.go`; per proposal Rollback plan levels 1–3 |

## Phase 0: Sequencing Gate — L1 self-reference fix (AD-1)

> MUST complete and pass in isolation before any task in Phase 1 leaves `src/agent/` in a tested state. This is a correctness requirement (design.md, File Changes table; proposal settled decision 2), not ordering preference.

- [x] 0.1 In `backend/agent/src/ai/import_boundary_test.go`, replace `const layer1Pattern = modulePath + "/..."` with `var layer1Patterns = []string{modulePath + "/src/ai/...", modulePath + "/src/agenttest/...", modulePath + "/src/handoff/..."}` (AD-1).
- [x] 0.2 Make `listNonStdlibDeps` variadic (`patterns ...string`) and update both call sites (`:154`, `:323`) to pass `layer1Patterns...`; join the vacuous-pass fatal message across the slice.
- [x] 0.3 Rewrite the `KNOWN HAZARD` comment block (lines 82–92) to record the chosen mechanism (narrowing, not exemption), the reason (AD-1's rejected-alternative rationale — a member exemption would also exempt the genuine violation), and the date. Preserve the `{modulePath}/src/agent` forbidden-prefix row unchanged.
- [x] 0.4 **GREEN (pre-existence baseline)**: with `src/agent/` still absent, run `cd backend/agent && go test ./src/ai/...`; confirm `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` and `TestLayer1_DependencySet_ExactRequiresAndClosure` both pass, the latter with `wantGoModRequires`/`wantExternalClosure` byte-unchanged. This proves the narrowed-pattern fix is neutral before `src/agent/` exists (S-AGP-064 partial evidence).

> The genuine-violation re-bite (S-AGP-062) needs `src/agent/` to exist as a real importable package first — it runs as task 1.7/1.8 below, after Phase 1. Do not attempt it here.

## Phase 1: AG-03.1 — Package scaffold and doc contract (mechanical + guard)

- [x] 1.1 Create `backend/agent/src/agent/doc.go`: `package agent`, doc comment only, three `L2C-NN` tab-delimited rows per design AD-4 (L2C-01 imports rule, L2C-02 no-I/O rule, L2C-03 event-stream-only rule) — no type, no constant, no function beyond the package clause (R-AGP-001, R-AGP-002).
- [x] 1.2 Create `backend/agent/src/agent/doc_contract_guard_test.go`: `layer2ContractRowPattern`, `contractRow`, `expectedLayer2ContractRows` (committed table mirroring 1.1's three rows), `docGoPath`, `parseLayer2ContractRows`, `TestLayer2DocContract_MatchesTheCommittedTable` — byte-level raw-file read via `runtime.Caller(0)`, count-then-order diff, mirrors `doc_matrix_guard_test.go` (AD-4). Header comment states the substitution for doc 0003's cited convention (S-AGP-015).
- [x] 1.3 **RED** (S-AGP-013): scratch-edit one `L2C-NN` row's text in `doc.go` without updating `expectedLayer2ContractRows`; run `go test ./src/agent/...`; confirm the guard FAILS naming the divergent row, expected text, and found text. Record output, then revert the scratch edit.
- [x] 1.4 **RED** (S-AGP-014): scratch-append a new contract row to `doc.go` without adding it to the table; run the guard; confirm it FAILS naming the unexpected row (proves closed comparison). Record output, then revert.
- [x] 1.5 **GREEN**: confirm `go test ./src/agent/...` passes with no scratch edits present (S-AGP-012).
- [x] 1.6 Confirm `test -e backend/agent/src/coding` and `test -e backend/agent/src/cmd` both exit non-zero (S-AGP-003) and `go doc github.com/cachicamas/backend/agent/src/agent` exports nothing beyond the package clause (S-AGP-002).
- [x] 1.7 **Sequencing gate payoff**: with `src/agent/` now existing, run `cd backend/agent && go test ./src/ai/...`; confirm `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` passes **for real reasons** (narrowed pattern excludes `src/agent` from the scanned set, no vacuous self-reference) and `TestLayer1_DependencySet_ExactRequiresAndClosure` still passes unchanged (S-AGP-060, S-AGP-063, S-AGP-064). (First run showed Go's test cache reusing task 0.4's pre-existence result — `-count=1` was required for genuine fresh evidence; see apply-progress.md.)
- [x] 1.8 **RED — S-AGP-062 (Layer 1 re-bite)**: plant a scratch file in `backend/agent/src/ai/` that genuinely imports `github.com/cachicamas/backend/agent/src/agent`; run `go test ./src/ai/...`; confirm `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` FAILS naming the offending import path and the `.../src/agent` forbidden-prefix rule — proving the fix is a fix, not a silencing. Record the failing output, then delete the scratch file and re-confirm green.

## Phase 2: AG-03.2 — Forward import guard, both closures (guard, depends on Phase 0 + Phase 1)

- [x] 2.1 Run and record the fresh closure measurement (AD-6, R-AGP-004) from `backend/agent/`: the three `go list -deps[-test]` commands over `src/ai` / `src/ai/...` listed in design.md; capture full output, date, and exact commands as evidence. Compare against doc 0003's 2026-08-11 claim; report any divergence explicitly (S-AGP-040..043). If a network-reaching path appears unexpectedly in the production closure, STOP and report as a blocking finding before continuing. **Finding**: no network-reaching path (not blocking); but the fresh measurement DISPROVES design.md's stated "src/ai's measured forced closure" rationale for the 5 OTel/xxhash allowlist entries — bare `src/ai`'s own production closure is empty of non-stdlib entries; OTel lives only in the forbidden `openaicompat` subtree. Per the explicit instruction ("never as a speculative grant"), those 5 entries are omitted from the allowlist actually written in 2.2. Full evidence in apply-progress.md.
- [x] 2.2 Create `backend/agent/src/agent/import_boundary_test.go` with the exact tables from design.md Interfaces/Contracts: `modulePath`, `layer2Pattern`, `forbiddenPrefixes` (8 rows, checked before allowlist), `allowedProductionPrefixes` (own package, `src/ai`, five closure-forced OTel/xxhash entries per AD-2/AD-6 — **deviation**: the 5 OTel/xxhash entries omitted per 2.1's finding, recorded in the file's own package comment), `allowedTestPrefixes` (production ∪ `src/agenttest`), `networkOrFilesystemPackages` (`net`, `net/http`, `os`, `io/fs`).
- [x] 2.3 Port `normalizeListedPackage` verbatim from L1; implement `listNonStdlibDeps`, `listAllProductionDeps`, `matchForbidden`, `isAllowed`. **Extension beyond design.md**: `isAllowed` also trims a trailing `_test` and retries — `go list -deps -test`'s synthesized `<pkg>_test` external-test-package self-reference is invisible to a curated per-layer allowlist (unlike L1's single bare-modulePath entry, which covers it by accident); found via a real GREEN-baseline failure, fixed, documented in source.
- [x] 2.4 Implement the three checks as separate tests: `TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault`, `TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction`, `TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage` (AD-3). Each fatals on an empty `go list` result.
- [x] 2.5 **GREEN baseline**: `go test ./src/agent/...` passes with no violation present (S-AGP-020, S-AGP-025, S-AGP-026).
- [x] 2.6 **RED — B1/S-AGP-027**: plant `backend/agent/src/coding/doc.go` (scratch `package coding`) and `src/agent/scratch_violation.go` blank-importing it; confirm forward guard FAILS on the `src/coding` forbidden row naming path + rule. Record, then delete both scratch files.
- [x] 2.7 **RED — B2/S-AGP-028**: plant `src/agent/scratch_violation.go` blank-importing `net/http`; confirm check 3 FAILS naming `net/http`. Record, then delete.
- [x] 2.8 **RED — B3/S-AGP-029**: plant `src/agent/scratch_violation.go` blank-importing `.../src/ai/openaicompat`; confirm the deny-by-name forbidden row fires (not a network side effect, not deny-by-default). Record, then delete. (Bonus confirmation: this bite is the only run in the whole session where the 5 OTel/xxhash paths ever appeared, and only via the forbidden import — empirically vindicating their omission from the allowlist, task 2.1/2.2.)
- [x] 2.9 **RED/GREEN pair — B4/S-AGP-026**: red half — plant `src/agent/scratch_violation.go` (production) blank-importing `.../src/agenttest`; confirm check 1 FAILS on the deny-by-default allowlist rule. Green half — plant `src/agent/scratch_widen_test.go` with the same import; confirm checks 1 and 3 stay green (production never sees it) and check 2 passes admitting it. Record both, then delete both. (Green half is direct empirical proof the OTel-omission decision is correct: it passes with zero OTel allowlist entries.)
- [x] 2.10 Confirm `git status` clean and `go.mod` gained no `require` entry (S-AGP-030).

## Phase 3: AG-03.3 — No-ambient-authority guard (guard, depends on Phase 1; parallel with Phase 2)

- [x] 3.1 Create `backend/agent/src/agent/ambient_authority_test.go`, lifting `openrouter/ambient_authority_test.go` verbatim per AD-5: `forbiddenAmbientAuthorityPackage` set (`os`, `os/exec`, `syscall`, `io/ioutil`), `isLayer2SourceFile`, `scanNonTestSourcesForAmbientAuthority`, `scanFileForAmbientAuthority` — alias/dot-import-aware, uniform `_test.go` exclusion, recorded no-type-information limitation (S-AGP-051..054, S-AGP-056).
- [x] 3.2 Implement tests: `..._NonTestSourcesCarryNoForbiddenCallSite` (scans `"."`), `..._ForbiddenSetIsPackageScopedDenyByDefault`, `..._FailsOnStagedMutation` (permanent `t.TempDir()` bite), `..._FileSelectionIsUniform`, `..._TestSourcesStayGreenEvenWithForbiddenCalls`.
- [x] 3.3 **GREEN baseline**: `go test ./src/agent/...` passes, reports having inspected at least one source file (S-AGP-050). (Openrouter's precedent didn't explicitly fence/report this; added an explicit `isLayer2SourceFile`-count fatal + `t.Logf` in the top-level test to satisfy S-AGP-050's literal wording — see apply-progress.md.)
- [x] 3.4 **RED — B5/S-AGP-055**: plant `src/agent/scratch_violation.go` (`package agent`) calling `os.Getenv("SCRATCH_ONLY")`; confirm the ambient scan FAILS naming file, line, and offending package. Record, then delete (the TempDir staged-mutation test in 3.2 keeps this falsifiability proof permanent).

## Phase 4: Cross-check and evidence close-out

- [x] 4.1 Run `cd backend/agent && make test` (`go test -race -v ./...`) and `make lint`; confirm both green/clean; record output (S-AGP-004). All 12 packages `ok`, zero FAIL/panic/DATA RACE; golangci-lint "0 issues."
- [x] 4.2 Confirm `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` are byte-unchanged (`git diff --stat` shows no entry) (S-AGP-005).
- [x] 4.3 Confirm the changed-file set under `backend/agent/src/ai/` is exactly one entry (`import_boundary_test.go`) with no `require` added and no forbidden-prefix row removed (S-AGP-066).
- [x] 4.4 Assemble the seven recorded reds (S-AGP-013, S-AGP-014, S-AGP-027, S-AGP-028, S-AGP-029, S-AGP-055, S-AGP-062) plus B4's paired assertion into the PR evidence section, alongside the fresh closure measurement from 2.1.
- [x] 4.5 Write the PR description stating why the change exceeds the default 400-line budget (NFR-AGP-003) and citing the pre-authorized `size:exception` against the 1000-line budget for this run. Final: 1005 changed lines (5 over the ~1000 figure; flagged in apply-progress.md and the final report, not silently rounded).

## Phase 5: Spec delta application (already drafted, apply at archive)

- [ ] 5.1 At archive, merge the `agent-module-scaffold` delta's `R-AGM-004`/`S-AGM-035` and `R-AGM-005`/`S-AGM-041` full MODIFIED restatements into `openspec/specs/agent-module-scaffold/spec.md`, adding the "Amended 2026-08-12 (AG-03)" header note per that spec's promotion discipline.
- [ ] 5.2 Promote `openspec/changes/cachicamas-agent-package-scaffold/specs/agent-package-scaffold/spec.md` verbatim as the new `openspec/specs/agent-package-scaffold/spec.md` (new capability, no prior spec to merge against).
