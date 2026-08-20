# Apply Progress: AG-19 — Prove re-entrancy and delegation readiness

**Change**: `cachicamas-agent-delegation-readiness` · **Mode**: Strict TDD · **Status**: 30/32 tasks complete (5.2 and 5.4 explicitly deferred to `sdd-archive`, not blocked — see below)

## Completed Tasks

- [x] 1.1–1.5 — Phase 1 (U1): the delegation seam, its installation, isolated revocation tests, S-DEL-022 bite.
- [x] 2.1–2.6 — Phase 2 (U2): the delegating-tool fixture, nested run, walkable tree, siblings under `-race`, S-DEL-021 bite.
- [x] 3.1–3.10 — Phase 3 (U3): nested cancellation, cost reconstruction, permission scope, S-DEL-023 and S-DEL-020 bites.
- [x] 4.1–4.7 — Phase 4: scope fence, inert path, substrate filter widening (+ new automated byte-in-sync test), v1-scope audit, full evidence run.
- [x] 5.1, 5.3 — Phase 5: citation re-resolution (this change's own 13 spec files only), doc 0003 update.
- [ ] 5.2, 5.4 — **NOT DONE, deliberately**: both are `sdd-archive`'s job per the orchestrator's explicit instruction ("Do not archive — that is sdd-archive's job later in this same PR") and per their own "at archive" wording.

## Files Changed

| File | Action | Lines | What |
|---|---|---|---|
| `backend/agent/src/agent/delegation_seam.go` | Created | 178 | `DelegationSeam`, `DelegationSeamFrom`, 2 sentinels, `admissible()`, mutex-latched `Publish`/`revoke` |
| `backend/agent/src/agent/scheduler.go` | Modified | +7/-1 | 3-line seam install/revoke in `executeCall`, zero new type assertions |
| `backend/agent/src/agent/delegation_seam_test.go` | Created | 550 | S-DEL-002, S-DEL-004 (25-kind table), S-DEL-005, S-AEV-125 |
| `backend/agent/src/agent/revocation_test.go` | Created | 271 | S-DEL-006/007/008/009 |
| `backend/agent/src/agent/delegating_tool_test.go` | Created | 269 | `delegatingTool` fixture + reconstruction helpers |
| `backend/agent/src/agent/nested_run_test.go` | Created | 92 | S-DEL-010 |
| `backend/agent/src/agent/walkable_tree_test.go` | Created | 204 | S-DEL-011/012 |
| `backend/agent/src/agent/siblings_test.go` | Created | 251 | S-DEL-013, S-AGE-028/029 |
| `backend/agent/src/agent/cancellation_test.go` | Created | 249 | S-DEL-014/015, S-CAN-015 |
| `backend/agent/src/agent/cost_test.go` | Created | 336 | S-DEL-016/017, S-CST-023/024 |
| `backend/agent/src/agent/permission_scope_test.go` | Created | 308 | S-DEL-018/019, `derivedScope` |
| `backend/agent/src/agent/scope_fence_test.go` | Created | 241 | S-DEL-024, S-LSK-031 (new byte-in-sync automated test) |
| `backend/agent/src/agent/inert_path_test.go` | Created | 126 | S-DEL-025 |
| `backend/agent/src/agent/v1_scope_test.go` | Created | 197 | S-AGS-062/063/064 |
| `backend/agent/src/agent/loop_test.go` | Modified | +26/-1 | Substrate filter widening (13 new entries) |
| `backend/agent/src/agent/loop_hook_test.go` | Modified | +26/-1 | Substrate filter widening, byte-in-sync |
| `docs/architecture/milestones/0003-...md` | Modified | +2/-2 | Ticked re-entrancy row, 18→19 of 24, AG-19 narrative |
| `openspec/changes/.../specs/*.md` (5 files) | Modified | citation fixes | `scheduler.go`/`scheduler_test.go`/`loop_test.go`/`loop_hook_test.go` line-range corrections |

## TDD Cycle Evidence (Strict TDD)

| Unit | RED | GREEN | REFACTOR |
|---|---|---|---|
| Seam admissibility/zero-event/sentinels | Compile-fail: `undefined: agent.DelegationSeam` at `delegation_seam_test.go:260` | `go test -race -count=1 -run TestDelegationSeam` — 8/8 PASS (after 1.2+1.3+1.4 together; external tests genuinely need installation, see task 1.2 note) | gofmt-fixed alignment |
| Revocation (normal/detach/re-panic) | Same compile-fail as above (single new-surface RED covers the whole file) | Same 8/8 PASS run | Redesigned detached fixture mid-implementation (see Key Learning) |
| S-DEL-021 bite | `CheckStream` reported `event[4]: value repeats another the collection already carries` | Reverted, `TestNestedRun_...` PASS | n/a |
| S-DEL-022 bite | Isolated `go test -run ... -count=1`: `panic: send on closed channel` at `delegation_seam.go:126`, exit 1, no `--- FAIL` line | Reverted, exit 0 | n/a |
| Nested run / walkable tree / siblings | Written directly against existing seam+fixture (no separate compile-fail RED recorded beyond the file's own first `go vet`) | `go test -race -count=1 -run 'TestNestedRun\|TestWalkableTree\|TestSiblings'` — all PASS | — |
| S-DEL-023 bite | `drainSink`'s own 1s timeout fired (child never cancels) — evidence differs from literal "FAILS on assertion 1" but proves the same claim more strongly | Reverted, PASS | n/a |
| Cancellation / cost / permission scope | Written directly; `go test -race -count=1 -run 'TestCancellation\|TestCost\|TestPermissionScope'` — all PASS | — | — |
| S-DEL-020 bite | `event[7] is a cost_turn carrying the CHILD's run identity` fired (the strict-inequality comparison itself did not flip at chosen token counts — noted honestly) | Reverted, PASS | n/a |
| Scope fence / inert path / v1-scope / S-LSK-031 | `*agent.Harness` method count assumption (4) was wrong (actual 5, missing `Compact`) — caught by running the test, not by review | Fixed after grepping all production files; PASS | — |
| Full evidence | — | `go test -race -v -count=1 ./...` from `backend/agent/`: exit 0, wall-clock 173s (17:50:06→17:52:59), zero `--- FAIL`/`FAIL` anywhere in 12 packages | `make lint` found 3 real issues (package-comment convention, 2× De Morgan), fixed, re-ran to `0 issues` |

## Work Unit Evidence (all units)

| Evidence | Value |
|---|---|
| Focused test command | `cd backend/agent && go test -race -count=1 -run '<TestPrefix>' ./src/agent/...` per phase (see commands above); all green |
| Runtime harness | Full suite: `go test -race -v -count=1 ./...` — exit 0, 173s wall-clock, zero failures across 12 packages, 529 top-level `--- PASS` in `src/agent` |
| Rollback boundary | Each phase is its own commit (6 total): `870969f8` (openspec planning), `6578e581` (seam), `a23bb92d` (nested run/walkable/siblings), `52e2ce97` (cancellation/cost/permission), `8a49b362` (scope fence/inert path/filters/v1-scope), `06eae1bc` (doc 0003 + tasks.md) — any commit reverts cleanly without touching the ones before it |

## Honest Overage Flag

The Review Workload Forecast estimated 1020–1650 counted lines (excluding `openspec/`). **Actual counted total is 3338 lines** (3333 insertions + 5 deletions, backend/ + docs/, 17 files) — roughly 2–3.3× the forecast, well beyond even the pre-accepted `size:exception`. Root cause: this codebase's own house style (every file carries extensive requirement/design-citing doc comments, consistently since AG-03) was matched throughout, and 32 tasks covering 36 scenarios genuinely needed 12 distinct test files rather than fewer, denser ones. `openspec/` total is 1687 insertions (excluded from budget, reported separately). Grand total: 34 files, 5020 insertions, 5 deletions. Per the orchestrator's explicit "do not split, do not stop on budget" instruction, work proceeded to completion; this overage is flagged prominently here and in the return envelope for the PR description to state explicitly.

## Deviations from Design

None in substance. One design-level gap discovered and corrected during implementation (see Key Learnings in the return envelope): the original mental model for S-DEL-022's crash mechanism was wrong and the fixture needed a genuine redesign to produce the real hazard.

## Issues Found

- Two pre-existing stale citations (`loop_test.go:831-871`, `loop_hook_test.go:907-943`) had been wrong since AG-11 and were never corrected through AG-15/16/17/18's own widenings — fixed now since they fall inside this change's own spec files' scope. The SAME stale citation also persists, unfixed, in the PROMOTED `openspec/specs/agent-loop-skeleton/spec.md` and `openspec/specs/agent-turn-termination/spec.md`, and in 6 archived change folders — explicitly left untouched as `sdd-archive`/out-of-scope territory, not apply's to fix.
- `gofmt -l` found 15 pre-existing baseline files (never touched by this change) that are not gofmt-clean under the current toolchain — left untouched per the no-mutating-fmt rule; only my own 4 authored/modified files were fixed.

## Correction Round — `sdd-verify` FAIL closure

**Trigger**: `sdd-verify` returned **FAIL** against `85344247` — 6 CRITICAL blockers (3 independently confirmed by the orchestrator reading the code), 6 WARNINGs. This round closes all 6 CRITICALs plus WARNING-1 (fold-blind cost equality), the only WARNING explicitly required. Full disposition table, per-blocker RED-proof commands and the exact spec-text corrections are in `tasks.md`'s own "Correction Round" section (added alongside this one) — not duplicated here in full to avoid the two documents drifting; this section is the TDD-evidence-table view of the same work.

### TDD Cycle Evidence (Correction Round)

| Fix | RED | GREEN | REFACTOR |
|---|---|---|---|
| S-DEL-015 real assertion | Scratch `context.WithCancel` inserted into `delegation_seam.go`'s `Publish` → `--- FAIL: TestCancellation_NestedRunCancelsLeafFirst` ("the production diff ... introduces \"context.WithCancel\"") | Reverted (`git checkout HEAD --`), `go test -race -count=1 -run TestCancellation_NestedRunCancelsLeafFirst` PASS | n/a — guard is new, no prior implementation to refactor |
| 3× `errors.Is(x,x)` tautologies deleted/replaced | n/a (stdlib-guaranteed always-true/always-false, not bitable) | `go test -race -count=1 -run 'TestV1Scope\|TestCancellation\|TestCost'` PASS after edit | Renamed the surviving `:104` subtest to match what it now tests (`admissibility_rule_distinguishes_inadmissible_from_revoked_by_errors_Is`) |
| Empty subtest deleted | n/a (deletion) | Confirmed absent from `go test -v` subtest list; enclosing `TestV1Scope_S_AGS_063_Seam12MechanismsAuditable` still PASS with its remaining 6 subtests | n/a |
| WARNING-1 fold-blind fix | (pre-condition for CRITICAL-4's re-derivation, not independently bitten — its effect IS the CRITICAL-4 RED below) | `go test -race -count=1 -run TestCost_` PASS after adding the `run agent.RunID` filter param | n/a |
| S-DEL-020 spec correction | Scratch-disabled gate 5 in `admissible()` (AFTER the WARNING-1 fix) → `--- FAIL: TestCost_ParentAloneStrictlyLessThanCombined`, both `cost_test.go:167` (92≠15 equality) and `:179` (leaked child cost_turn) fire; strict-inequality line does not | Reverted, PASS | n/a |
| S-DEL-023 spec correction | Scratch `childCtxOverride` → `context.Background()` via the fixture's own existing bite hook → `--- FAIL: TestCancellation_NestedRunCancelsLeafFirst`: `cancellation_test.go:148: drainSink: sink did not close within 1s (6 event(s) received so far)` | Reverted, PASS | n/a |
| S-DEL-001 new test | Scratch-changed `Parent()` to `return "ag19-bite-scratch-wrong-run-id", s.turnID` → `--- FAIL: TestDelegationSeam_S_DEL_001_...`, both the run-identity and the turn_start-Run() comparisons report the mismatch (`delegation_seam_test.go:363,373`). An earlier attempt (empty RunID) cascaded into a `t.Fatalf` from a non-test goroutine inside `beforeReturn` and hung the process (killed manually) — redesigned to a non-empty-but-wrong value, which the OUTER `t.Errorf` comparisons catch cleanly after `h.Run()` returns normally | Reverted, `go test -race -count=1 -run TestDelegationSeam_S_DEL_001` PASS | Bite redesign (see RED column) — a real discovery, not a cosmetic retry |
| S-TLS-020 new test | First draft used `strings.Contains(line, "DelegationSeam")`, which false-positived on the legitimate `newDelegationSeam`/`withDelegationSeam` calls — caught by running the test, not by review. Fixed with a word-boundary regex. Then scratch-inserted `var agBiteSeam DelegationSeam = seam` + `any(agBiteSeam).(*delegationSeam)` into `scheduler.go`'s diff region → both detectors fire in `TestScopeFence_S_TLS_020_...` | Reverted, PASS | Word-boundary regex fix (see RED column) |
| Full evidence | — | `go test -race -v -count=1 ./...` from `backend/agent/`: exit 0, wall-clock **176s** (18:59:11→19:02:07, final run after all 7 scratch-edit RED/revert cycles), zero `--- FAIL` anywhere in 12 packages, 362 top-level `--- PASS` in `src/agent` | `go vet ./...` clean throughout; `gofmt -l` clean on all 5 touched files; `delegation_seam.go`/`scheduler.go` confirmed byte-identical to the committed baseline (`git diff --stat` empty) before this final run |

### Self-correction caught by running tests, not by review (Correction Round)

Same discipline as the original apply's own two self-corrections (above). Two more this round:

1. The `S-TLS-020` word-boundary regex bug was caught only because the test was actually RUN against the real diff — `newDelegationSeam`/`withDelegationSeam` are legitimate calls this change's own wiring must make, and a naive substring check would have permanently false-positived every future `git diff` scan of this file, silently breaking the guard's usefulness rather than proving anything.
2. An early `S-DEL-001` bite attempt (`Parent()` scratch-changed to return an EMPTY `RunID`) hung the test process instead of failing cleanly: `agent.NewMessageStartText("", ...)` legitimately rejects an empty run, `t.Fatalf` fired from inside the `beforeReturn` callback — which runs on a scheduler-owned tool-call goroutine, not the test's own goroutine — and Go's `testing` package documents that `FailNow`/`Fatalf` "must be called from the goroutine running the test ... not from other goroutines"; calling it elsewhere terminates only that one goroutine via `runtime.Goexit()`, so the scheduler's own `WaitGroup` never got its completion signal and `Harness.Run()` blocked forever. Diagnosed by reading the hang, not guessed at: killed the process, redesigned the bite to return a non-empty-but-WRONG `RunID` instead (no inner error path triggered), and the outer `t.Errorf` comparisons — which run on the test's own goroutine after `h.Run()` returns normally — caught the mismatch cleanly. This is a generalizable caution for this whole file's `beforeReturn`-based fixtures: they are only safe to `t.Fatalf` from when the fatal branch is provably unreachable in a healthy run, which is true for every PRE-EXISTING use but was not true for my own bite.

### Files changed this round

| File | Action | Lines | What |
|---|---|---|---|
| `backend/agent/src/agent/cancellation_test.go` | Modified | +32/-7 | S-DEL-015 real git-diff guard (replaces dangling comment) |
| `backend/agent/src/agent/cost_test.go` | Modified | +19/-9 | WARNING-1: `sumCostTurnInputTokens` filtered by run identity |
| `backend/agent/src/agent/v1_scope_test.go` | Modified | +27/-16 | 3 tautologies deleted/replaced, 1 empty subtest deleted |
| `backend/agent/src/agent/delegation_seam_test.go` | Modified | +103/-5 | New `S-DEL-001` test; `seamCapturingTool` gained `Policy()` |
| `backend/agent/src/agent/scope_fence_test.go` | Modified | +81/-0 | New `S-TLS-020`/`S-DEL-003` raw-byte + distinctness test |
| `openspec/changes/.../specs/agent-delegation-readiness/spec.md` | Modified | +2/-2 | Corrected `S-DEL-020`/`S-DEL-023` failure-mechanism sentences |
| `openspec/changes/.../tasks.md` | Modified | — | Appended Correction Round disposition table |
| `openspec/changes/.../apply-progress.md` | Modified | — | This section |

**Changed-line total this round**: 299 lines in `backend/agent/` (authored risk, counted against the review budget); 4 lines in `openspec/specs/` (excluded, reported only). Within the PR's own already-accepted `size:exception` — no new workload decision needed; this closes defects in already-in-scope work, not new scope.

### Grep sweep for restated phrases (per "a corrected requirement leaves its scenarios wrong")

Before editing, grepped the whole change directory for `strict inequality broken`, `FAILS on assertion 1`, `runs on past the parent's interrupt`, `inflated by exactly the child's spend`. Hits outside `spec.md:192,213` (the two corrected lines): `tasks.md:76`, `apply-progress.md:46` (both already correctly recorded the divergence as apply-time observations, contrasting against — not asserting — the old literal spec text; left as historical record) and `specs/agent-cost-events/spec.md:14,62` (references only the general, still-true "cumulative inflates" claim, never the false "strict inequality broken" specific — no correction needed). `verify-report.md` cites the old text as its own finding evidence and is not this phase's artifact to edit — it remains the historical record of what triggered this round; a fresh `sdd-verify` pass produces the next one.

### Disposition summary

All 6 CRITICAL blockers closed (2 by real new assertions replacing dead code, 2 by real assertions replacing tautologies, 1 by honest deletion, 2 by spec re-scoping to the actually-observed mechanism, 2 by new dedicated tests). WARNING-1 closed. No blocker was a false positive — the orchestrator's independent confirmation of blockers 1-3 held, and this phase's own re-derivation confirmed 4-6 and W1 by real command, not by re-reading the report.
