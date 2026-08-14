# Archive Report — AG-09 `cachicamas-agent-tool-scheduler`

> **Change**: `cachicamas-agent-tool-scheduler` · **AG-09** (Layer 2 Wave 2, milestone 9 of 24; doc 0003 lines 902–1004)
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag09`
> **Branch**: `feat/agent-layer2-wave2-ag09`
> **Base**: `e27e8411` (AG-08 merge point)
> **Artifact store**: HYBRID (filesystem + Engram)
> **Mode**: Strict TDD (per `openspec/AGENTS.md`)
> **Archived to**: `openspec/changes/archive/2026-08-13-cachicamas-agent-tool-scheduler/`
> **Closed at**: `aaf93b4a`

## Final Verdict

**`pass-with-warnings`** — 0 blockers, 0 critical findings, 6 warnings.

Per the orchestrator's final-state facts (outranking the intermediate `verify-report` snapshot at #3026):

- CRITICAL #1 (`TestScheduler_UnbufferedSink_ConcurrentConsumer` `-count=10` flake) — **FIXED** in commit `74c22be0` by dropping the racy `runtime.NumGoroutine()` baseline; the deterministic observable is the drained count + the consumer's exit.
- CRITICAL #2 (`tasks.md` 5.1 checkbox lag) — **FIXED** in commit `74c22be0` by flipping the checkbox.
- Re-run `go test -race -count=10 ./src/agent/...` → green in **3.211s** (verified at archive time, two consecutive runs).

Per `verify-report` (#3026) intermediate snapshot at verification time:

| Metric | Value |
|---|---|
| Verdict | `pass-with-warnings` |
| Blockers | 0 |
| Critical findings | 0 |
| Warnings | 6 |
| Requirements | 12/12 |
| Scenarios | 20/20 |
| `loop.go` line coverage | 84.3% (≥ 80% threshold per R-LSK-005) |
| Boundary guards | All pass untouched |
| Substrate files (21 + go.mod + go.sum + Makefile + .golangci.yml) | Byte-identical to base `e27e8411` |
| Lint / vet / build | Clean |

The intermediate `verify-report` snapshot (#3026) initially reported `verdict: fail` at first persist; per the orchestrator's launch prompt the second update committed at `aaf93b4a` carries the final `pass-with-warnings` verdict after remediation commit `74c22be0`. The archive reflects the **final** state per the Final-State Authority hierarchy (orchestrator launch prompt > intermediate snapshots).

## Engram Observation IDs (for traceability)

| Phase | Topic key | Observation ID | Persisted at |
|---|---|---|---|
| Explore | `sdd/cachicamas-agent-tool-scheduler/explore` | **#3013** | 2026-08-13 14:52:39 |
| Proposal | `sdd/cachicamas-agent-tool-scheduler/proposal` | **#3014** | 2026-08-13 14:58:27 |
| Spec | `sdd/cachicamas-agent-tool-scheduler/spec` (set memo) | **#3017** | 2026-08-13 15:04:36 |
| Design | (design memo) | **#3018** | 2026-08-13 15:06:59 |
| Tasks | `sdd/cachicamas-agent-tool-scheduler/tasks` | **#3019** | 2026-08-13 15:12:15 |
| Apply-progress | `sdd/cachicamas-agent-tool-scheduler/apply-progress` | **#3020** | 2026-08-13 15:15:53 |
| Verify-report | `sdd/cachicamas-agent-tool-scheduler/verify-report` | **#3026** | 2026-08-14 11:24:39 |
| **Archive (this report)** | `sdd/cachicamas-agent-tool-scheduler/archive-report` | **(this save)** | 2026-08-14 |

## Commit History (10 commits)

Work-unit commits (8, per `tasks.md`) + 2 remediation commits:

```
aaf93b4a chore(agent): AG-09 verify-report updated to PASS WITH WARNINGS after remediation
74c22be0 fix(agent): AG-09 verify remediation — drop racy goroutine baseline, flip tasks.md 5.1
39709f61 chore(agent): AG-09 apply-progress recorded — 7 work-unit commits, all gates clean
54323601 docs(agent): AG-09 archive prep — doc 0003 + spec reconcile
0ce7f679 test(agent): AG-09 end-to-end tool dispatch — S-LSK-008 + S-LSK-008a
c33d1a09 feat(agent): AG-09 wire-up — TurnOptions.Tools + drainSink 1-line deadline
58697852 feat(agent): AG-09.2+09.3+09.4 scheduler — hand-rolled concurrency, rejoin, panic containment
469db46f feat(agent): AG-09.1 contract — Tool interface, EffectClass, PolicySlot, Result
2be43fee test(agent): AG-09 RED bites — 6 bite tests RED-recorded
b2ab3867 chore(agent): AG-09 spec+design committed
```

## Specs Synced

| Domain | Action | Source-of-truth file | Details |
|---|---|---|---|
| `agent-tool-scheduler` | **Created** (new capability spec) | `openspec/specs/agent-tool-scheduler/spec.md` | 11 requirements (`R-TLS-001..011`) + 12 spec scenarios + 6 bites + 1 cross-cut scenario (`S-LSK-008`) referenced. Created at commit 1 (`b2ab3867`); already at canonical location. |
| `agent-loop-skeleton` | **Modified** (delta folded) | `openspec/specs/agent-loop-skeleton/spec.md` | Delta merged in commit 7 (`54323601`): ADDED `R-LSK-006` (one cycle per turn) + `S-LSK-008` + `S-LSK-008a`; MODIFIED `R-LSK-001` (`Tools map[string]Tool` field + AG-08 `PreRequestHook`); MODIFIED `R-LSK-005` (coverage includes `loop_tool_dispatch_test.go` + the one-cycle invariant). Verified at archive time: all three modifications present. |

The `openspec/specs/agent-loop-skeleton/spec.md` file at HEAD reflects the merged delta — no archive-time merge was needed.

## Coverage Summary

- **11 R-TLS requirements** (`R-TLS-001`..`R-TLS-011`) all covered.
- **12 spec scenarios** (`S-TLS-001`..`S-TLS-011` + `S-LSK-009` wire-up) — all GREEN under `-race`.
- **6 bites** (`S-TLS-002a`, `S-TLS-005a`, `S-TLS-006a`, `S-TLS-006b`, `S-TLS-010a`, `S-TLS-011a`) — all GREEN.
- **2 cross-cuts** (`S-LSK-008` + `S-LSK-008a`, the AG-09 → AG-13 wording-trap boundary) — both GREEN.
- **Total**: **20 scenarios** (matching the identical count stated in `proposal.md` / `tasks.md` / `apply-progress.md` / `verify-report.md`).
- **Substrate preservation (NFR-TLS-003)**: **21 files byte-identical to base `e27e8411`** — 6th consecutive "substrate untouched" milestone.
- **`go.mod` / `go.sum`**: byte-identical to base (no new deps; no `errgroup`).
- **`Makefile` / `.golangci.yml`**: not modified.
- **Boundary guards** (`import_boundary_test.go`, `ambient_authority_test.go`): pass untouched.
- **Size**: `size:exception` consumed at **4258 lines** (3050 code + 1208 planning, per orchestrator launch prompt). The actual `git diff --stat` against `e27e8411..HEAD` shows 19 files / 4539 insertions / 35 deletions (= 4574 lines) including the apply-progress and verify-report artifacts themselves.

## Archive Folder Contents

```
openspec/changes/archive/2026-08-13-cachicamas-agent-tool-scheduler/
├── apply-progress.md     (22931 bytes)
├── design.md             (28002 bytes)
├── exploration.md        (50897 bytes)
├── proposal.md           (23838 bytes)
├── specs/
│   └── agent-loop-skeleton/
│       └── spec.md       (delta, frozen for audit)
├── tasks.md              (6768 bytes — all 8 tasks [x])
├── verify-report.md      (24498 bytes — final verdict: pass-with-warnings)
└── archive-report.md     (this file)
```

The `openspec/changes/cachicamas-agent-tool-scheduler/` folder has been removed from the active changes directory (verified: no such directory exists at archive time).

## Task Completion Gate

`openspec/changes/cachicamas-agent-tool-scheduler/tasks.md` — all **8 task checkboxes marked `[x]`** at archive time (verified line-by-line):

- [x] 1.1 spec+design chore (`b2ab3867`)
- [x] 2.1 RED bites (`2be43fee`)
- [x] 3.1 AG-09.1 contract (`469db46f`)
- [x] 3.2 AG-09.2+09.3+09.4 scheduler (`58697852`)
- [x] 3.3 AG-09 wire-up + drainSink (`c33d1a09`)
- [x] 4.1 end-to-end dispatch test (`0ce7f679`)
- [x] 4.2 substrate-untouched widening (`54323601`, as part of doc reconcile)
- [x] 5.1 doc 0003 + spec reconcile (`54323601`, checkbox flipped by `74c22be0`)

No exceptional mechanical reconciliation was needed by `sdd-archive`; the remediation commit `74c22be0` flipped the stale checkbox before this archive ran.

## Native Review Receipt Gate

Per the orchestrator's structured status: `gentle-ai review status` reports `applicability: unrelated`, `receipt.status: not_applicable`. The kill switch is off (braejan's `gentle-ai review mode disable --scope clone`); delivery is `disabled/unmanaged`. Archive proceeds under the Native Review Receipt Gate's `not_applicable` relaxation.

## Archive Verification Checklist

- [x] Main spec `openspec/specs/agent-loop-skeleton/spec.md` reflects the merged delta (verified: `R-LSK-006` + `S-LSK-008`/`S-LSK-008a` present; `R-LSK-001` and `R-LSK-005` carry the modified bodies).
- [x] New capability spec `openspec/specs/agent-tool-scheduler/spec.md` is at the canonical location (since commit 1).
- [x] Change folder moved to archive at `openspec/changes/archive/2026-08-13-cachicamas-agent-tool-scheduler/`.
- [x] Active changes directory no longer contains `cachicamas-agent-tool-scheduler`.
- [x] Archived `tasks.md` has all 8 tasks marked `[x]`.
- [x] Archive contains all required artifacts: proposal, exploration, design, tasks, apply-progress, verify-report, specs delta.

## Risks (warnings carried forward to the archive audit trail)

Per `verify-report` (#3026) at verification time, 6 warnings, all documented and load-bearing-claim-bite-covered:

1. **Wire-up order deviates from design D8b** — finalize → Schedule → closeSink (not close → finalize → Schedule as the design stated). Reason: scheduler closes sink after rejoin; finalize must emit closing brackets BEFORE the close. Documented in `loop.go:230-238` and `apply-progress.md` Deviation #1. Spec does not contradict (S-LSK-008: "at most one cycle per `Turn`").
2. **`closeSink` has `defer/recover` wrapper** — Required because the scheduler closes the sink after rejoin and the loop's `closeSink` would otherwise panic on "send on closed channel". Documented in `loop.go:279-290`. Cleanest fix would be removing `closeSink` from the loop's wire-up path (the scheduler owns the close), but that's an API change to `Turn` — out of scope for AG-09.
3. **Timing assertions widened** — In `S-TLS-004`, `S-TLS-005a`, `S-TLS-006`, `S-TLS-006a`, `S-TLS-006b`. Original strict-ordering assertions were flaky under Go scheduler jitter. Widened to interval-overlap / non-equality / count-based assertions. The bites cover the load-bearing claims.
4. **Goroutine baseline check removed from `S-TLS-010` and `S-TLS-011` property tests** — The `runtime.NumGoroutine()` baseline was racy under heavy parallel test execution; the bite tests cover the same property with deterministic counter assertions. CRITICAL #1 (the AG-08 W1 carry test) was the last such instance and was fixed in commit `74c22be0`.
5. **`scripted_tool_test.go` location deviates from design** — Design placed `agenttest/scripted_tool.go` in Layer 1; actual implementation is `agent/scripted_tool_test.go` in `package agent_test`. Reason: ADR 0005 § D1 row 1 — Layer 1 must not import Layer 2. Documented as apply-progress Deviation #7.
6. **`agent_test_helpers_test.go` was NOT modified** — Design forecast adding `ordinalFromToolStart` helper, but the actual implementation derives call order from `ScriptedTool.Invocations()` counters + `events` slice indexing. No helper was needed.

2 SUGGESTIONS (non-blocking):

- **Replace goroutine baseline with deterministic invariant** in `TestScheduler_UnbufferedSink_ConcurrentConsumer` — already addressed by remediation commit `74c22be0`.
- **Consolidate substrate filter widening at AG-23** — Three commits (3, 5, 7) widened the `TestTurn_SubstrateUntouched` filter, once per new AG-09 file. The widening pattern is file-granularity. AG-23 is the natural consolidation point.
- **Add `MaxConcurrentReads` benchmark** — D8a mentions AG-13 may benchmark alternative defaults (4 vs 8 vs 16). Add `BenchmarkScheduler_ReadsFanOut`. Not blocking.

## Acceptance Criteria

- [x] AG-09 closes **G5 (R-13)** — parallel tool execution with deterministic call-ordered rejoin + per-tool concurrency policy.
- [x] AG-09 closes **v2 § 6 seams 2 & 3** — seam 2 (turn-typed execution surface) and seam 3 (opaque per-call `PolicySlot` Layer 2 carries but never reads).
- [x] `loop.go` line coverage ≥ 80% (R-LSK-005): **84.3%**.
- [x] 21 substrate files + `go.mod` + `go.sum` byte-unchanged (NFR-TLS-003).
- [x] AG-03 boundary guards stay green untouched.
- [x] No new top-level Go deps; no `errgroup` (source-guarded).
- [x] No `PolicySlot` type assertion in `scheduler.go` (source-guarded).
- [x] Doc 0003 lines 902–1004 milestone 9 of 24 marked shipped.
- [x] SDD cycle complete: explore → propose → spec → design → tasks → apply → verify → archive.

## Next Step

The SDD cycle for AG-09 is complete. PR open (see `branch-pr` work). Ready for AG-10 (permission protocol wraps the scheduler) per the dependency graph in `openspec/specs/agent-tool-scheduler/spec.md`.

## Key Learnings

1. **Goroutine baseline checks are racy in shared test binaries** — `runtime.NumGoroutine()` captures a process-wide count that other parallel tests' transient goroutines can push past any tolerance. The deterministic observable is `len(drainedEvents)` and `wg.Wait()`-style synchronization.
2. **Wire-up order matters when sink ownership is shared** — When the scheduler takes over the sink after the loop's `finalize`, the close-after-finalize pattern requires `finalize` to emit closing brackets BEFORE the scheduler's close. Documented in `loop.go:230-238`.
3. **`tasks.md` is the canonical completion source** — `apply-progress.md` is commentary; if its "all tasks complete" claim disagrees with `tasks.md`'s checkboxes, the verify phase must surface that as a discrepancy. Apply phases should end with a `tasks.md` flip.
4. **Layer 1 cannot import Layer 2** — ADR 0005 § D1 row 1 means test fixtures that satisfy the `agent.Tool` interface must live in `agent_test` (external) or in Layer 2's own files. Putting them in `agenttest` (Layer 1) would create a cycle.
5. **The archive report reflects final state, not intermediate snapshots** — The first `verify-report` persist reported `verdict: fail` (CRITICAL #1 + CRITICAL #2); the orchestrator-applied remediation in commit `74c22be0` fixed both, and `aaf93b4a` carried the final `pass-with-warnings` verdict. The archive's job is to record the final state, with explicit citation of where the fix landed.

---

**SDD cycle complete**. Change archived; PR open. AG-09 closes G5 (R-13) and v2 seams 2 & 3.
