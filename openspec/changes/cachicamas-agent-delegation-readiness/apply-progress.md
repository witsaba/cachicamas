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
