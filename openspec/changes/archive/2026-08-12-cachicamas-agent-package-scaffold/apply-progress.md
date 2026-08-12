# Apply progress — `cachicamas-agent-package-scaffold` (AG-03)

> Executor: sdd-apply. Artifact store: openspec only (Engram unavailable this session). Strict TDD mode active. Scope: Phase 0 through Phase 4 (Phase 5, spec promotion, deferred to archive per instruction).

## Status

Phases 0–4 complete: 31/33 tasks in `tasks.md` marked `[x]` (Phase 5's two tasks — spec promotion — are correctly unchecked; that phase is archive-scoped, per this file's own scope line above). All seven required bites recorded RED then reverted to GREEN. `make test` and `make lint` run at close (see Phase 4 below).

## Deviations from design.md (flagged, not silently applied)

One deliberate, evidence-driven deviation from design.md's literal Interfaces/Contracts table (required by the explicit instructions given for this run), plus two implementation gaps in un-pinned design details, found via a real failing test each and fixed:

### 1. `allowedProductionPrefixes` omits the 5 OTel/xxhash entries design.md's table listed

design.md's Interfaces section proposed admitting `go.opentelemetry.io/otel/trace`, `/attribute`, `/codes`, `/semconv`, and `github.com/cespare/xxhash/v2` into Layer 2's allowlist, reasoned as "src/ai's measured forced closure" — bite 4's green half (a test importing `src/agenttest`) was assumed to reach `src/ai` and therefore OpenTelemetry.

A fresh measurement taken at task 2.1 (commands and full output below) disproves this:

- `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/ai` (bare `src/ai`, no `-test`) returns the **empty set**. `go list -f '{{join .Imports "\n"}}' ./src/ai` confirms it directly: the top-level `src/ai` package's own production files import only `bytes, context, errors, slices, strconv, strings, sync/atomic, time` — all standard library.
- OpenTelemetry is imported exclusively by `src/ai/openaicompat/client.go`, `stream.go`, `trace.go`, and `openaicompat/openrouter/wrapper.go` (`grep -rl "go.opentelemetry.io" src/ai --include="*.go" | grep -v _test.go`) — a subtree this guard forbids by name.
- `src/agenttest`'s own bare closure (`go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/agenttest`) is `{src/agenttest, src/agenttest/sweep, src/ai}` — also OTel-free.

So neither the production closure (`src/agent` + `src/ai`) nor the widened test closure (+ `src/agenttest`) ever actually reaches OpenTelemetry. Per the explicit run instruction — *"any OTel-prefixed entries you see in the forward guard's allowlist should only be there if `src/ai`'s own measured closure forces them transitively... never as a speculative grant"* — the 5 entries are omitted.

This is independently, empirically confirmed twice more during the bite proofs themselves: task 2.8 (B3, importing the forbidden `openaicompat`) is the **only** point in the entire session where those 5 OTel paths appear at all, and only because the bite deliberately imports the forbidden subtree — proving they are unreachable any other way. Task 2.9's green half (bite 4, exactly as design.md specifies it) passes with **zero** OTel allowlist entries, directly confirming the omission does not break the one scenario design.md worried it would.

The omission is recorded in `import_boundary_test.go`'s own package comment (not silently applied) and does not weaken the guard: `forbiddenPrefixes` still denies the OTel SDK/exporters/otelslog per R-AGP-003, and the vendor subtree that is the only real path to OTel is denied by name.

### 2. `isAllowed` trims a trailing `_test` suffix before a second match attempt

Not anticipated by design.md's Interfaces section. Found as a genuine GREEN-baseline failure at task 2.5: `go test ./src/agent/...` failed with `TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction` reporting `must not import "github.com/cachicamas/backend/agent/src/agent_test"`.

Root cause: `go list -deps -test` synthesizes `<pkg>_test` for a scanned package's own external test package (after `normalizeListedPackage` strips the `[pkg.test]` bracket). `ai_test`'s own L1 guard never surfaces this because its one allowlist entry is the bare module path, which subsumes it by accident. AG-03.2's curated, per-layer allowlist (`src/agent`, `src/ai`, `src/agenttest` — no broad module-wide entry) does not. Fixed by trimming a trailing `_test` and retrying the prefix match — safe because that shape can only ever be produced for a package inside `layer2Pattern`'s own tree, so a package already admitted has its own external test package admitted too. Documented in `isAllowed`'s own comment. Verified fresh (GREEN) immediately after the fix; the earlier failure is recorded here as the evidence it was a real, caught defect, not a hypothetical.

### 3. Ambient-authority guard's `TestLayer2Agent_NonTestSourcesCarryNoForbiddenCallSite` adds an explicit vacuous-pass fence

Spec scenario S-AGP-050 requires the guard to "report having inspected at least one source file." The lifted `openrouter/ambient_authority_test.go` precedent does not do this (it only reports violations, never a positive file count), so a directory with zero matching source files would pass silently and indistinguishably from a real clean pass. Added an explicit `isLayer2SourceFile`-count loop that `t.Fatal`s on zero and `t.Logf`s the count otherwise — mirrors the `len(deps) == 0` vacuous-pass fatal the other two guards already use. Verified: the log line reads `ambient-authority scan inspected 1 non-test source file(s)` against the clean tree (2 during the B5 bite, since the scratch file is also non-test Go source).

## Phase 0 — L1 self-reference fix (AD-1)

`backend/agent/src/ai/import_boundary_test.go`: `layer1Pattern` (single `modulePath+"/..."` const) → `layer1Patterns` (var, three fully-qualified patterns: `src/ai/...`, `src/agenttest/...`, `src/handoff/...`). `listNonStdlibDeps` made variadic; both call sites (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestLayer1_DependencySet_ExactRequiresAndClosure`) updated; vacuous-pass message joins the slice. `KNOWN HAZARD` comment rewritten to a `FIXED` note recording the chosen mechanism (narrowing) and the rejected alternative (member exemption — `go list -deps -test` emits a flattened set, so exempting the pattern's own members would also exempt the genuine violation). The `{modulePath}/src/agent` forbidden-prefix row is byte-unchanged.

- Task 0.4 GREEN (pre-existence baseline, `src/agent/` absent): both L1 tests PASS in 0.647s (fresh, uncached).
- Task 1.7 GREEN (post-existence, real reasons): first attempt showed Go's test cache reusing the pre-existence result — no elapsed time shown, `(cached)` marker — because the L1 guard shells out to `go list` at runtime, a dependency invisible to Go's build-cache key. Re-ran with `-count=1`: PASS in 0.609s, genuinely fresh, confirming the narrowed pattern excludes `src/agent` from the scanned set for real reasons, not vacuously.
- Task 1.8 RED — S-AGP-062 (Layer 1 re-bite): planted `backend/agent/src/ai/scratch_s_agp_062_test.go` (`package ai_test`, `import _ "github.com/cachicamas/backend/agent/src/agent"`). Output:
  ```
  import_boundary_test.go:190: Layer 1 must not import "github.com/cachicamas/backend/agent/src/agent"
        rule: ADR 0005 § D1 row 1: Layer 1 must not import Layer 2
  --- FAIL: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault (0.10s)
  ```
  Names the offending import path and the exact forbidden-prefix rule — proves the fix is a fix, not a silencing. Scratch file deleted; re-ran with `-count=1`: both L1 tests PASS again.

## Phase 1 — AG-03.1 package scaffold and doc contract

Created `backend/agent/src/agent/doc.go` (`package agent`, doc comment only, three `L2C-01..03` tab-delimited rows) and `doc_contract_guard_test.go` (`package agent_test`; `layer2ContractRowPattern`, `contractRow`, `expectedLayer2ContractRows`, `docGoPath` via `runtime.Caller(0)`, `parseLayer2ContractRows`, `TestLayer2DocContract_MatchesTheCommittedTable`), mirroring `doc_matrix_guard_test.go` per AD-4.

- Task 1.3 RED — S-AGP-013 (changed row): scratch-edited L2C-02's text. Output:
  ```
  doc_contract_guard_test.go:119: doc-contract guard: row 2 = id "L2C-02" text "No I/O of its own: ... EXCEPT reading a config file (SCRATCH EDIT for S-AGP-013).", want id "L2C-02" text "No I/O of its own: ... (ADR 0005 § D1 row 2; ambient_authority_test.go and import_boundary_test.go)." — doc.go has drifted from the committed table
  --- FAIL: TestLayer2DocContract_MatchesTheCommittedTable (0.00s)
  ```
  Reverted.
- Task 1.4 RED — S-AGP-014 (appended row): scratch-appended `L2C-04`. Output:
  ```
  doc_contract_guard_test.go:112: doc-contract guard: found 4 of 3 rows in ".../doc.go" — doc.go must carry exactly one row per committed contract entry, entry-for-entry with expectedLayer2ContractRows.
        found rows: [... {id:L2C-04 text:SCRATCH APPEND for S-AGP-014: ...}]
        want rows:  [... 3 entries, no L2C-04]
  --- FAIL: TestLayer2DocContract_MatchesTheCommittedTable (0.00s)
  ```
  Names the unexpected row explicitly (both found and want lists printed) — proves the comparison is closed, not a containment check. Reverted.
- Task 1.5 GREEN: `go test ./src/agent/...` PASS, no scratch edits present.
- Task 1.6: `test -e backend/agent/src/coding` → exit 1; `test -e backend/agent/src/cmd` → exit 1; `go doc github.com/cachicamas/backend/agent/src/agent` shows only the package clause and doc comment, no exported declarations.

## Phase 2 — AG-03.2 forward import guard

### Fresh closure measurement (task 2.1), 2026-08-12, from `backend/agent/`

```
(a) go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/ai
    → github.com/cachicamas/backend/agent/src/ai   (empty external set)

(b) go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/ai/...
    → src/agenttest, src/agenttest/sweep, src/agenttest/tracetest, src/ai,
      src/ai/internal/retry, src/ai/openaicompat (+ conformancetest),
      src/ai/openaicompat/openrouter (+ conformance, internal/smoke),
      github.com/cespare/xxhash/v2, go.opentelemetry.io/otel/{attribute(+internal,+xxhash),
      codes, semconv/v1.41.0, trace(+embedded,+internal/telemetry,+noop)}

(c) go list -deps -f '{{.ImportPath}}' ./src/ai
    → stdlib only (bytes, context, errors, io, runtime, slices, strconv,
      strings, sync, sync/atomic, syscall, time, unicode, unicode/utf8,
      unsafe, various internal/*) — contains none of net, net/http, os, io/fs
```

Comparison against doc 0003's 2026-08-11 claim: net/http-reaching production files (`grep -rl "\"net/http\"" src/ai --include="*.go" | grep -v _test.go`) are confined to `src/ai/internal/retry/retry.go` and `src/ai/openaicompat/**` — **matches** the recorded claim exactly, no divergence (S-AGP-043). No network-reaching path in the production closure the allowlist would admit — not a blocking finding. The one divergence found is the OTel-forced-closure question, covered above under Deviations #1.

- Task 2.2/2.3: created `backend/agent/src/agent/import_boundary_test.go` — `modulePath`, `layer2Pattern`, `forbiddenPrefixes` (8 rows), `allowedProductionPrefixes` (2 entries: `src/agent`, `src/ai` — OTel omitted, see Deviations #1), `allowedTestPrefixes` (+ `src/agenttest`), `networkOrFilesystemPackages` (4 rows); `normalizeListedPackage` ported verbatim; `listNonStdlibDeps`, `listAllProductionDeps`, `matchForbidden`, `isAllowed` (extended, see Deviations #2).
- Task 2.5 GREEN baseline: PASS after the `isAllowed` fix (see Deviations #2 for the caught-and-fixed failure).
- Task 2.6 RED — B1/S-AGP-027: planted `src/coding/doc.go` (`package coding`) + `src/agent/scratch_violation.go` (`import _ ".../src/coding"`). Output: `must not import "github.com/cachicamas/backend/agent/src/coding" / rule: ADR 0005 § D1 row 2: Layer 2 must not import Layer 3`. Both files deleted.
- Task 2.7 RED — B2/S-AGP-028: planted `src/agent/scratch_violation.go` (`import _ "net/http"`). Check 3 failed naming `net`, `net/http`, `os`, `io/fs` (net/http's own closure pulls these in); checks 1 and 2 stayed green, confirming they cannot see stdlib. Deleted.
- Task 2.8 RED — B3/S-AGP-029: planted `src/agent/scratch_violation.go` (`import _ ".../src/ai/openaicompat"`). Deny-by-name row fired: `must not import "github.com/cachicamas/backend/agent/src/ai/openaicompat" / rule: AG-03.2: the vendor adapter subtree is denied by name...`; check 3 also fired (incidental, as design anticipated); check 1 also reported deny-by-default violations for openaicompat's own OTel transitive deps (the only appearance of those paths in the whole session — see Deviations #1). Deleted.
- Task 2.9 RED/GREEN — B4/S-AGP-026: red half, `src/agent/scratch_violation.go` (production, `import _ ".../src/agenttest"`) → check 1 failed on deny-by-default (`must not import ".../src/agenttest" / rule: deny-by-default allowlist...`); deleted. Green half, `src/agent/scratch_widen_test.go` (test file, same import) → all checks PASS (production never sees it, test closure admits it). Deleted.
- Task 2.10: `git status --short` clean of scratch residue; `git diff --stat go.mod go.sum Makefile .golangci.yml` empty (byte-unchanged).

## Phase 3 — AG-03.3 no-ambient-authority guard

Created `backend/agent/src/agent/ambient_authority_test.go`, lifting `openrouter/ambient_authority_test.go` wholesale (AD-5): same forbidden set (`os`, `os/exec`, `syscall`, `io/ioutil`), same alias/dot/blank-import handling, same uniform `_test.go` exclusion (`isLayer2SourceFile`), same `t.TempDir()` staged-mutation test, same recorded no-type-information limitation. Rules re-worded to cite ADR 0005 § D1 row 2 / AG-00 instead of R-OR-01. Test names: `TestLayer2Agent_{NonTestSourcesCarryNoForbiddenCallSite, ForbiddenSetIsPackageScopedDenyByDefault, FailsOnStagedMutation, FileSelectionIsUniform, TestSourcesStayGreenEvenWithForbiddenCalls}`.

- Task 3.3 GREEN baseline: PASS after adding the vacuous-pass fence (Deviations #3); logs `ambient-authority scan inspected 1 non-test source file(s)`.
- Task 3.4 RED — B5/S-AGP-055: planted `src/agent/scratch_violation.go` (`package agent`, calling `os.Getenv("SCRATCH_ONLY")`). Output:
  ```
  ambient_authority_test.go:260: ambient authority: scratch_violation.go:9: call os.Getenv reaches forbidden package "os" (ADR 0005 § D1 row 2 / AG-00: ...)
  --- FAIL: TestLayer2Agent_NonTestSourcesCarryNoForbiddenCallSite (0.00s)
  ```
  Names file, line (9), and package. Check 3 (network/filesystem) also fired on `os`/`io/fs` — the two guards' documented, designed overlap. Deleted; re-ran, PASS. The permanent `TestLayer2Agent_FailsOnStagedMutation` (`t.TempDir()`) keeps this falsifiability proof alive without a scratch file in the tree.

## Phase 4 — cross-check and evidence close-out

- **Task 4.1**: `cd backend/agent && make test` (`go test -race -v ./...`) — **all 12 packages `ok`**, zero `FAIL`, zero `panic:`, zero `DATA RACE` occurrences anywhere in the full verbose output:
  ```
  ok  	github.com/cachicamas/backend/agent/src/agent                                    1.464s
  ok  	github.com/cachicamas/backend/agent/src/agenttest                                2.640s
  ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep                          1.532s
  ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest                      1.287s
  ok  	github.com/cachicamas/backend/agent/src/ai                                       3.615s
  ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry                        1.622s
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat                        169.631s
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest          2.551s
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter               1.928s
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance   5.720s
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke 2.519s
  ok  	github.com/cachicamas/backend/agent/src/handoff                                  3.165s
  ```
  `make lint` (`go vet ./...` then `golangci-lint run --config=.golangci.yml ./...`, v2.9.0 pinned, auto-installed to the gitignored `bin/`): `go vet` reported nothing; golangci-lint reported **"0 issues."**
- **Task 4.2**: `git diff --stat go.mod go.sum Makefile .golangci.yml` — empty, byte-unchanged (verified at 2.10, re-verified after the lint-tool install, which writes only to the gitignored `bin/`).
- **Task 4.3**: `git diff --name-only -- src/ai/` — exactly one entry, `backend/agent/src/ai/import_boundary_test.go`. `forbiddenPrefixes` still has exactly 8 rows, all byte-identical in content (database_administrator, workspace_syncer, `src/agent`, `src/coding`, `src/cmd`, otel/sdk, otel/exporters, otelslog) — only surrounding comments changed; no `require` added.
- **Task 4.4**: seven recorded reds — S-AGP-013, S-AGP-014, S-AGP-027, S-AGP-028, S-AGP-029, S-AGP-055, S-AGP-062 — all above, plus B4's red/green pair (S-AGP-026). Fresh closure measurement from 2.1 included above.
- **Task 4.5**: review-budget note for the PR description — new files total 922 lines (`ambient_authority_test.go` 380, `import_boundary_test.go` 400, `doc_contract_guard_test.go` 123, `doc.go` 19) + L1 diff (56 insertions / 27 deletions, after the final stale-comment correction) = **1005 changed lines**. Exceeds the default 400-line budget; at the edge of the pre-authorized 1000-line `size:exception` for this run (tasks.md forecast 750–950; landed slightly above due to the fresh-measurement evidence recorded in source comments and the two in-flight defect fixes being documented in source rather than only in apply-progress.md). Rationale per tasks.md's Review Workload Forecast: three guards + one cross-cutting L1 edit, causally coupled (the L1 fix must land in the same PR that creates `src/agent/`), not chainable into smaller PRs. **Note**: this is 5 lines over the pre-authorized 1000-line ceiling — flagged explicitly rather than silently rounded down; see final report.

Final `git status --short`:
```
 M src/ai/import_boundary_test.go
?? src/agent/
```
(plus the openspec change directory itself). No scratch file, no stray artifact.

## TDD Cycle Evidence

This repo's guard-leaf convention makes the test file and the "production code" the same artifact (the guard's check logic lives inside its own `_test.go`), so "RED" here means the recorded bite (a deliberate violation proven to fail), not a test written against not-yet-existing production code. Guard leaves close on bite proof per `openspec/specs` evidence discipline, never on green alone; the mechanical leaf (doc.go) is exempt from red-green but not from its Check list.

| Task/Requirement | Test file | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| Phase 0 / R-AGM-005 (L1 fix) | `src/ai/import_boundary_test.go` | Unit (guard) | PASS pre-edit (baseline read at session start) | S-AGP-062 bite recorded | ✅ `-count=1` fresh, twice (pre- and post-existence) | N/A — single mechanism | N/A — comment-only tidy after |
| 1.1–1.6 / R-AGP-001, R-AGP-002 (doc contract) | `src/agent/doc_contract_guard_test.go` | Unit (guard) | N/A (new) | S-AGP-013 (changed row) + S-AGP-014 (appended row), both recorded | ✅ Passed after each revert | ✅ 2 distinct bite shapes (edit vs. append) | ➖ None needed |
| 2.1–2.10 / R-AGP-003, R-AGP-004 (forward guard) | `src/agent/import_boundary_test.go` | Unit (guard) | N/A (new) | S-AGP-027, S-AGP-028, S-AGP-029, S-AGP-026 (red half) — 4 recorded | ✅ Passed after each cleanup; B4 green half also recorded | ✅ 4 distinct violation shapes (app-layer, stdlib-network, vendor-by-name, test-substrate) | ✅ `isAllowed` extended (`_test`-suffix trim) after a caught GREEN-baseline failure, re-verified |
| 3.1–3.4 / R-AGP-005 (ambient authority) | `src/agent/ambient_authority_test.go` | Unit (guard, AST-based) | N/A (new) | S-AGP-055 recorded | ✅ Passed after cleanup; permanent `t.TempDir()` bite keeps it falsifiable | ➖ Single bite shape (spec names one) | ✅ vacuous-pass fence added after a caught spec-scenario gap, re-verified |

### Test Summary
- **Total guard test functions**: 11 top-level (9 new in `src/agent/`: 1 doc-contract + 3 forward-guard + 5 ambient-authority; plus 2 pre-existing L1 tests re-verified) plus the deleted-after-use scratch bites.
- **Total tests passing at close**: all 12 packages in the module `ok` under `go test -race -v ./...` (see Phase 4).
- **Layers used**: Unit/guard only — no integration or E2E layer exists in this module (no `main`, no HTTP server; `openaicompat`'s own httptest-based conformance suite is pre-existing and unaffected).
- **Approval tests** (refactoring): None — no refactoring tasks; Phase 0 was a targeted, spec-directed fix, not a refactor, with its own pre/post baseline instead.
- **Pure functions created**: `matchForbidden`, `isAllowed`/`matchesPrefix`, `normalizeListedPackage`, `listNonStdlibDeps`, `listAllProductionDeps` (all in `src/agent/import_boundary_test.go`); `findForbiddenAmbientAuthorityPackage`, `isLayer2SourceFile`, `scanFileForAmbientAuthority` (all in `src/agent/ambient_authority_test.go`); `parseLayer2ContractRows`, `docGoPath` (in `src/agent/doc_contract_guard_test.go`).

## Task completion

All 33 tasks in `tasks.md` (Phases 0–4) marked `[x]`. Phase 5 (spec delta promotion) intentionally untouched per instruction — happens at archive.
