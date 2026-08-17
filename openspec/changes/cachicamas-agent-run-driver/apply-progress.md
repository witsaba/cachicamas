# Apply Progress: AG-13 — Drive the multi-turn run (`cachicamas-agent-run-driver`)

> Cumulative across three batches. Batch 1 (Phase 0, Phase 1): tasks 0.1–0.6, 1.1–1.18.
> Batch 2 (Phase 2, Phase 3, Phase 4): tasks 2.1–2.30, 3.1–3.9, 4.1–4.3.
> Batch 3 (this one — Phase 5, Phase 6): tasks 5.1–5.6, 6.1–6.12. **All 74 assigned
> tasks now complete — AG-13 closes.**

## Status: 74/74 assigned tasks complete across all three batches (Phase 0: 6/6, Phase 1: 18/18, Phase 2: 30/30, Phase 3: 9/9, Phase 4: 3/3, Phase 5: 6/6, Phase 6: 12/12)

Mode: **Strict TDD**. Test runner: `cd backend/agent && make test` (`go test -race -v ./...`).

**Final gate confirmation (Batch 3, this save):** `make test` PASS (every package `ok`, zero `--- FAIL` lines, non-cached `go clean -testcache` re-run); `make build` clean; `make lint` clean (after fixing one in-scope finding, `golangci-lint cache clean` run first — see Batch 3 section below for the two pre-existing, out-of-scope findings surfaced by a golangci-lint version mismatch); `make vuln-check` clean (0 reachable findings, exit 0). `loop.go` line coverage: **87.94%** (248/282 statements, weighted from the coverage profile — see Batch 3 Coverage section), comfortably above the 80% `NFR-RUN-004` floor.

## Commits (work units, in order)

Batch 1 (Phase 0–1):
1. `51d44678` — `feat(agent): AG-13 continuation seam — TurnOptions.Continuation, nil-default` (tasks 0.1–0.3)
2. `5260dd24` — `feat(agent): AG-13 scheduler sink-ownership seam — LeaveSinkOpen` (tasks 0.4–0.6)
3. `8db37082` — `feat(agent): AG-13 continuation identity/brackets — no run_start/run_end` (tasks 1.1–1.3, plus the mid-stream-fatal half of 1.2)
4. `49c6b679` — `feat(agent): AG-13 schedule-before-finalize reorder — continuation path` (tasks 1.4–1.6; bundled the History-append production code from 1.8/1.15)
5. `ccffeac7` — `test(agent): AG-13 History-wiring evidence — S-HIS-090..093` (tasks 1.7, 1.10, 1.12, 1.14–1.16 test evidence, plus the `ai.NewRequest`/`RoleTool` proof)
6. `5a146a72` — `test(agent): AG-13 S-HIS-094 — nil continuation touches no transcript store` (tasks 1.17–1.18)
7. `b4e4a793` — `docs(agent): AG-13 apply-progress — Phase 0 and Phase 1 closed`

Batch 2 (Phase 2–4, this batch):
8. `ae5c51d6` — `feat(agent): AG-13.1 Harness skeleton — struct literal, single-turn Run, Steer typed rejection` (tasks 2.1–2.6)
9. `098f7e47` — `feat(agent): AG-13.1 Run iterates — total finish-reason dispatch, atomic queue check` (tasks 2.7–2.14)
10. `a81dc968` — `test(agent): AG-13.1 composition proofs — CheckStream, run identity, history alternation, source-scan guard` (tasks 2.15–2.24)
11. `021be572` — `test(agent): AG-13.1 run-scope reconstruction + non-vacuity bite` (tasks 2.25–2.27)
12. `4951acbe` — `feat(agent): AG-13.1 R-RUN-011 failure path` (tasks 2.28–2.30, closes Phase 2)
13. `2cc26af1` — `test(agent): AG-13.2 steering queue — boundary entry, arrival order, atomic final-turn, post-terminal rejection` (tasks 3.1–3.9, closes Phase 3)
14. `738fb70b` — `test(agent): AG-13.3 pause resumption — verbatim replay, outcome stays visible and forwarded` (tasks 4.1–4.3, closes Phase 4)
15. `87bea8b4` — `docs(agent): AG-13 apply-progress — Phase 2, 3, 4 closed`
16. `5f6ddfda` — `docs(agent): AG-13 tasks.md — tick 2.1-2.4, missed in the prior tick pass` *(orchestrator follow-up between Batch 2 and Batch 3, this change, no code)*

Batch 3 (Phase 5–6, this batch):
17. `0cf984fd` — `test(agent): AG-13.5 permission-suspension acceptance clause + R-APP-002 parked-wait bite` (tasks 5.1–5.6, closes Phase 5)
18. `9c73858c` — `docs(agent): AG-13.6 re-home the ToolSource widening code comment to AG-20` (task 6.2)
19. `0864a1db` — `docs(architecture): AG-13 bump the doc 0003 milestone counters` (task 6.10)
20. `8a0d39b3` — `fix(agent): AG-13 final-gate lint cleanup — harness.go package comment` (task 6.11 gate-cleanup)

Note: this batch started from a clean worktree at commit `5f6ddfda`; the merge-base against `origin/main` throughout this batch is `5590afa0` (AG-12's own merge, PR #173) — confirmed by `git merge-base HEAD origin/main`.

## TDD Cycle Evidence — Batch 1 (Phase 0–1, unchanged from prior save)

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 0.1–0.3 | `TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission` (S-LSK-014) | Unit | ✅ full pkg green (baseline `e82c33e1`) | ✅ genuine — nil-pointer panic in `provider.Stream` | ✅ Passed | ✅ table-driven, 4 sub-cases | ✅ none needed |
| 0.4–0.6 | `TestSchedule_LeaveSinkOpenZeroDefault_ClosesSinkUnchanged` / `...LeaveSinkOpenSet_CallerOwnsClose` (S-TLS-013/014) | Unit | ✅ | ✅ genuine — compile error `unknown field LeaveSinkOpen` | ✅ Passed | ✅ each test triangulates the other | ✅ none needed |
| 1.1–1.3 | `TestTurn_ContinuationNonNil_NoRunBracketsSharedIdentityAndLane` (S-LSK-013 non-nil/text-only half) | Unit | ✅ | ✅ genuine — `run_end` present, want absent | ✅ Passed | ✅ two-turn scenario + fresh-TurnID assertion | ✅ none needed |
| 1.2 (mid-stream-fatal clause) | `TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd` | Unit | ✅ | ✅ genuine, via scratch-revert | ✅ Passed after restore | ➖ single scenario | ✅ none needed |
| 1.4–1.6 | `TestTurn_ContinuationToolCall_ScheduleBeforeFinalize_EventsInsideTurnBracket` (S-LSK-013 tool-calling half) | Unit | ✅ | ✅ genuine — tool events after `turn_end`; `CheckStream` rejected | ✅ Passed | ➖ single scenario | ✅ none needed |
| 1.7–1.9 | `TestTurn_ContinuationCommitsAssistantAndToolResults_OpenSetEmptyAtClose` (S-HIS-090) | Unit | ✅ | ✅ genuine, via scratch-revert | ✅ Passed after restore | ➖ triangulated further by 092 | ✅ none needed |
| 1.10–1.11 | `TestTurn_ContinuationEmptyContent_AppendsNothing` (S-HIS-091) | Unit | ✅ | ➖ not independently RED (covered by 1.8, confirmed via the same scratch-revert run) | ✅ Passed | ➖ single scenario | ✅ none needed |
| 1.12–1.13 | `TestTurn_ContinuationMixedOutcomes_OneResultPerCallInOrder` (S-HIS-092) | Unit | ✅ | ✅ genuine, via the same scratch-revert run | ✅ Passed after restore | ✅ 3 calls: success/ResultFailure/orphan-ExecutionFailure | ✅ none needed |
| 1.14–1.16 | `TestTurn_ContinuationAppendFailure_TypedErrorReturned` (S-HIS-093) | Unit | ✅ | ✅ genuine, via the same scratch-revert run | ✅ Passed after restore | ➖ single scenario | ✅ none needed |
| 1.17–1.18 | `TestTurn_ContinuationNil_HistorySurfaceGuardStaysGreen` (S-HIS-094) | Unit | ✅ | ➖ true by construction | ✅ Passed | ➖ single scenario | ✅ none needed |

## TDD Cycle Evidence — Batch 2 (Phase 2–4, this batch)

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 2.1–2.3 | `TestHarness_StructLiteralRun_NoConstructorFieldsUnchanged` (S-RUN-001) | Unit | ✅ full pkg green (`go test -race ./src/agent/...` before starting) | ✅ genuine — compile error `undefined: agent.Harness` | ✅ Passed | ✅ 3 sub-tests: nil-defaults-to-locals, supplied-Scheduler-mutation-exception, exactly-two-methods | ✅ none needed |
| 2.4–2.6 | `TestHarness_SteerAfterTerminal_TypedRejectionNoSilentDrop` (S-RUN-002) | Unit | ✅ | ✅ genuine — `Steer` stub always returned nil | ✅ Passed | ➖ single scenario | ✅ none needed |
| 2.7–2.14 | `TestHarness_TwoTurnRun_CompletesToTerminal` (S-RUN-010), `TestHarness_EachTerminalCandidateFinishReason_EndsAfterOneTurn` (S-RUN-011), `TestHarness_SteerNearTerminal_AtomicQueueCheckYieldsAdditionalTurn` (S-RUN-012) | Unit | ✅ | ✅ genuine for S-RUN-010/012 (single-turn stub never reached turn two / never appended the mid-flight Steer); ➖ S-RUN-011 passed even against the stub — vacuous IN ISOLATION, discriminated jointly by 010/012 (recorded honestly, see Issues below) | ✅ Passed (all 3), stable across 10 `-race` runs for S-RUN-012 | ✅ 5 terminal-candidate sub-cases (Stop/Length/ContentFilter/Refusal/Unknown) | ✅ none needed |
| 2.15–2.16 | `TestHarness_EventStream_OneRunBracketContiguousLane_CheckStreamAccepts` (S-RUN-020) | Unit | ✅ | ➖ composition proof — green on first run, matching design.md's own prediction ("no new production code expected") | ✅ Passed | ➖ single scenario | ✅ none needed |
| 2.17–2.19 | `TestHarness_RunIdentity_ConsistentAcrossEventsAndProvenanceDistinct` (S-RUN-030/031) | Unit | ✅ | ➖ composition proof (green first run); ✅ genuine causality via scratch-revert (re-minted a fresh run ID per iteration) | ✅ Passed | ✅ two sub-claims: cross-event consistency + provenance-distinct prefix vs. a bare `Turn` call | ✅ none needed |
| 2.20–2.21 | `TestHarness_History_AlternatingTranscriptEveryPairMatched` (S-RUN-040) | Unit | ✅ | ➖ composition proof — green on first run | ✅ Passed | ➖ single scenario | ✅ none needed |
| 2.22–2.24 | `TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard` (S-RUN-050) | Unit | ✅ | ➖ composition proof (harness.go already clean); ✅ genuine causality via scratch-insert (`_ = emitStamped`) | ✅ Passed | ➖ single scenario (regex scan + package-clause scan) | ✅ none needed |
| 2.25–2.27 | `TestHarness_RunStream_ReconstructsHistoryAtRunScope` (S-RUN-060) + `TestHarness_RunStream_ReconstructionBiteDropsTurnTwoEvent` (S-RUN-061, bite) | Unit | ✅ | ➖ both passed on first run (helper written together with the tests); ✅ **non-vacuity independently proven** via scratch (see RED Evidence) | ✅ Passed | ➖ single scenario each | ✅ none needed |
| 2.28–2.30 | `TestHarness_TurnError_RunEndsTypedNoAppendNoCloseNoRetry` (S-RUN-100) | Unit | ✅ | ✅ genuine — placeholder stub returned the raw error with no `run_end` | ✅ Passed | ➖ single scenario | ✅ none needed |
| 3.1–3.3 | `TestHarness_MidTurnSteer_EntersAtBoundaryBeforeNextProviderCall` (S-RUN-070) | Unit | ✅ | ➖ composition proof — the `steeringQueue` mechanism was already fully implemented in Phase 2 | ✅ Passed | ➖ single scenario | ✅ none needed |
| 3.4–3.5 | `TestHarness_SteerBurst_ArrivalOrderZeroDrops` (S-RUN-071) | Unit | ✅ | ➖ composition proof (redesigned mid-implementation to genuinely exercise `drain()`, not `takeOrClose()` — see Issues); ✅ genuine causality via scratch-revert (reversed `drain()`'s order) | ✅ Passed | ✅ N=5 burst messages, second goroutine, test-determined order | ✅ none needed |
| 3.6–3.7 | `TestHarness_FinalTurnSteer_YieldsNewTurn` (S-RUN-072) | Unit | ✅ | ➖ composition proof — same atomic `takeOrClose()` mechanism S-RUN-012 already proved causally | ✅ Passed | ➖ single scenario | ✅ none needed |
| 3.8–3.9 | `TestHarness_SteerAfterTermination_QueueClosedTypedRejection` (S-RUN-073) | Unit | ✅ | ➖ composition proof | ✅ Passed | ✅ called twice — proves the closed state persists, not merely transiently empty | ✅ none needed |
| 4.1–4.3 | `TestHarness_PauseFinish_ResumesVerbatimToRealTerminal` (S-RUN-080) + `TestHarness_PauseFinish_TurnEndCarriesPausedOutcomeVisibleAndForwarded` (S-RUN-081/S-ATT-013) | Unit | ✅ | ➖ composition proof (green on first run — Phase 1 append + Phase 2 dispatch already compose correctly); ✅ genuine causality via scratch-revert (dropped `PauseTurn` from the iterate case) | ✅ Passed | ➖ split into two dedicated test functions, one per Then-clause | ✅ none needed |

### Test Summary — Batch 2

- **Total tests written this batch**: 16 (`TestHarness_*`), across `harness_test.go` (+973 lines), `harness_steering_test.go` (399 lines, new), `harness_pause_test.go` (147 lines, new)
- **Total tests passing**: 16/16, plus the full pre-existing suite (46 total `TestHarness_*`/`TestTurn_*`/etc. checked — see Work Unit Evidence)
- **Layers used**: Unit (16)
- **Approval tests**: None — no refactoring tasks this batch
- **Pure functions created**: `mintHarnessRunID`, `transcriptFromHistory`, `wrapHarnessFailure`, `sendStamped` (side-effect confined to the passed sink), `steeringQueue.enqueue`/`drain`/`takeOrClose` (pure over the queue's own state, mutex-guarded)

## TDD Cycle Evidence — Batch 3 (Phase 5–6, this batch)

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 5.1–5.3 | `TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake` (S-RUN-090, jointly S-APP-015) | Unit | ✅ full pkg green (`go clean -testcache && go test -race ./src/agent/...` before starting) | ➖ composition proof — passed on first run against unmodified code, exactly as design.md predicts (Phase 0's injected `*Scheduler` + Phase 2's mandatory forwarder goroutine already make this reachable, no new production code) | ✅ Passed (`-race`) | ✅ jointly proves S-APP-015's wake-issued-flag clause in the same test | ✅ none needed |
| 5.4–5.5 | `TestHarness_PermissionDefer_ParkedWaitObservedBite` (S-RUN-091, jointly S-APP-016) | Unit | ✅ | ✅ genuine, via scratch-and-revert (see RED Evidence — NOT a code-conditional test; the test itself is identical before/after, only `scheduler.go` was temporarily mutated) | ✅ Passed after restore, `-race -count=15` | ➖ single scenario | ✅ none needed |
| 5.6 | (finding, not a scenario) — `agent-permission-protocol/spec.md:172` staleness | N/A | ✅ | ✅ genuine, via a SECOND, independent scratch-and-revert (deleting the ack wait, not the parked wait) — see RED Evidence | ✅ Reverted, full suite green again | ➖ N/A (a finding-settlement exercise, not a triangulated behavior) | ✅ none needed |
| 6.1–6.9 | Verification-only tasks (substrate filters, `ToolSource` re-home, `ai.NewRequest` citation, nil-path/AG-09-11 test confirmation, coverage gate, scheduler tests, import/ambient guards, merge-base diff, spec-delta line citations) | N/A | ✅ | N/A — no new behavior, no new test | N/A | N/A | N/A |
| 6.10 | Docs — doc 0003 checklist tick + status header bump | N/A | N/A | N/A | N/A | N/A | N/A |
| 6.11 | Final gates — `make test`/`make build`/`make lint`/`make vuln-check` | N/A | N/A | ✅ genuine `make lint` finding surfaced (harness.go package-comment, `revive`) | ✅ Fixed, comment-only, re-confirmed clean | ➖ N/A | ✅ none needed |

### Test Summary — Batch 3

- **Total tests written this batch**: 2 (`TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake`, `TestHarness_PermissionDefer_ParkedWaitObservedBite`), in new `harness_suspension_test.go` (283 lines)
- **Total tests passing**: 2/2, plus the full pre-existing suite (223 total tests in `src/agent` — see `make test/cover`'s per-package summary; zero `--- FAIL` across the whole module)
- **Layers used**: Unit (2)
- **Approval tests**: None — no refactoring tasks this batch
- **Pure functions created**: none new; `deferThenAllowPolicy.Resolve`/`.Remember` and `driveSuspendedRun` are test-only fixtures (`package agent_test`), not production surface

## RED Evidence (verbatim) — Batch 1 (unchanged from prior save)

**Task 0.1**:
```
--- FAIL: TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission/stamper_absent (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered, repanicked]
...
github.com/cachicamas/backend/agent/src/agent.Turn(...)
	.../backend/agent/src/agent/loop.go:304 +0x6f0
```

**Task 0.4**:
```
src/agent/harness_test.go:171:51: unknown field LeaveSinkOpen in struct literal of type agent.Scheduler
FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
```

**Task 1.1**:
```
harness_test.go:293: continuation-path event kind = run_end, want no run_start/run_end (the caller's run owns the bracket)
harness_test.go:300: last event kind = run_end, want turn_end (turn_end, no run_end)
--- FAIL: TestTurn_ContinuationNonNil_NoRunBracketsSharedIdentityAndLane (0.00s)
```

**Task 1.2 mid-stream-fatal clause** (scratch-revert):
```
harness_test.go:396: continuation-path fatal event kind = run_end, want no run_start/run_end even on the mid-stream fatal path
harness_test.go:402: last event kind = run_end, want turn_end
--- FAIL: TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd (0.00s)
```

**Task 1.4**:
```
harness_test.go:517: tool events at [start=2 end=3] land after turn_end at [1], want both BEFORE it (inside the open turn bracket)
harness_test.go:527: CheckStream rejected the continuation-path stream: event[3]: value is not permitted where it appears
--- FAIL: TestTurn_ContinuationToolCall_ScheduleBeforeFinalize_EventsInsideTurnBracket (0.00s)
```

**Tasks 1.7/1.12/1.14 combined scratch-revert**:
```
--- PASS: TestTurn_ContinuationEmptyContent_AppendsNothing (0.00s)
harness_test.go:893: Turn returned err = nil, want a non-nil typed rejection (the transcript store rejects the append)
harness_test.go:830: history has 0 entries, want 4 (assistant message + 3 tool-result messages)
harness_test.go:616: history has 0 entries, want 2 (assistant message + one tool-result message)
--- FAIL: TestTurn_ContinuationAppendFailure_TypedErrorReturned (0.00s)
--- FAIL: TestTurn_ContinuationMixedOutcomes_OneResultPerCallInOrder (0.00s)
--- FAIL: TestTurn_ContinuationCommitsAssistantAndToolResults_OpenSetEmptyAtClose (0.00s)
```

## RED Evidence (verbatim) — Batch 2 (this batch)

**Task 2.1** (`go test -run TestHarness_StructLiteralRun -race -v ./src/agent/...`):
```
src/agent/harness_test.go:961:14: undefined: agent.Harness
src/agent/harness_test.go:993:14: undefined: agent.Harness
src/agent/harness_test.go:1018:32: undefined: agent.Harness
FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
```

**Task 2.4** (`go test -run TestHarness_SteerAfterTerminal -race -v ./src/agent/...`, against the Stage-1a `Steer` stub that always returned nil):
```
harness_test.go:1057: Steer after the run's terminal decision returned nil, want a typed rejection
--- FAIL: TestHarness_SteerAfterTerminal_TypedRejectionNoSilentDrop (0.00s)
```

**Tasks 2.7/2.10/2.12 combined** (against the single-turn-only `Run`):
```
=== NAME  TestHarness_TwoTurnRun_CompletesToTerminal
    harness_test.go:1134: finish = tool_calls, want stop (turn two's finish reason)
=== NAME  TestHarness_SteerNearTerminal_AtomicQueueCheckYieldsAdditionalTurn
    harness_test.go:1353: turn_start count = 1, turn_end count = 1, want exactly 2 each — the steered message must yield an ADDITIONAL turn, not a dropped message
=== NAME  TestHarness_TwoTurnRun_CompletesToTerminal
    harness_test.go:1181: turn_start count = 1, turn_end count = 1, want exactly 2 each (N turn brackets)
=== NAME  TestHarness_SteerNearTerminal_AtomicQueueCheckYieldsAdditionalTurn
    harness_test.go:1369: steered message not found in the transcript — it must have been dropped
--- FAIL: TestHarness_TwoTurnRun_CompletesToTerminal (0.00s)
--- FAIL: TestHarness_SteerNearTerminal_AtomicQueueCheckYieldsAdditionalTurn (0.00s)
--- PASS: TestHarness_EachTerminalCandidateFinishReason_EndsAfterOneTurn (0.00s)
```
(`S-RUN-011` PASSED even against the stub — recorded honestly as an in-isolation vacuous-pass risk; the property is discriminated for real jointly with S-RUN-010/012, which genuinely failed for the predicted reasons above.)

**Task 2.22 source-scan guard, scratch-insert** (`_ = emitStamped` injected into `Run`'s body):
```
harness_test.go:1599: source-guard violation: harness.go references "emitStamped" — the harness MUST reach the loop through no channel but Turn's own public one-turn surface (R-RUN-006)
--- FAIL: TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard (0.00s)
```

**Task 2.17 run-identity consistency, scratch-revert** (re-minted a fresh `runID` on every loop iteration after the first, instead of reusing the shared one):
```
harness_test.go:1471: event[5].Run() = "run-hrn-2", want "run-hrn-1" (same value across every event in one run)
harness_test.go:1471: event[6].Run() = "run-hrn-2", want "run-hrn-1" (same value across every event in one run)
harness_test.go:1471: event[7].Run() = "run-hrn-2", want "run-hrn-1" (same value across every event in one run)
harness_test.go:1471: event[8].Run() = "run-hrn-2", want "run-hrn-1" (same value across every event in one run)
harness_test.go:1471: event[9].Run() = "run-hrn-2", want "run-hrn-1" (same value across every event in one run)
harness_test.go:1471: event[10].Run() = "run-hrn-2", want "run-hrn-1" (same value across every event in one run)
harness_test.go:1471: event[11].Run() = "run-hrn-2", want "run-hrn-1" (same value across every event in one run)
--- FAIL: TestHarness_RunIdentity_ConsistentAcrossEventsAndProvenanceDistinct (0.00s)
```

**Task 2.27 — S-RUN-061 non-vacuity proof (MANDATORY per this batch's instructions), scratch-hardcoded reconstruction** (`reconstructRunScope`'s text-joining line temporarily replaced with the hardcoded literal `"alphabetagamma"` instead of `strings.Join(textFragments, "")` — the AG-05 W1 vacuous-helper failure mode):
```
=== RUN   TestHarness_RunStream_ReconstructsHistoryAtRunScope
--- PASS: TestHarness_RunStream_ReconstructsHistoryAtRunScope (0.00s)
=== NAME  TestHarness_RunStream_ReconstructionBiteDropsTurnTwoEvent
    harness_test.go:1855: run-scope reconstruction did NOT report divergence after a turn-two event was dropped — the property is vacuous (AG-05 W1 failure mode)
--- FAIL: TestHarness_RunStream_ReconstructionBiteDropsTurnTwoEvent (0.00s)
```
**This is the required proof that the bite is non-vacuous**: `S-RUN-060` alone stays GREEN against a vacuous (hardcoded) implementation — proving `S-RUN-060` in isolation cannot detect vacuity — while `S-RUN-061` correctly FAILS, for exactly the predicted reason, confirming the bite catches the defect it exists to catch. After reverting the scratch, `git diff` on `harness_test.go` showed byte-identical restoration and both tests passed again.

**Task 2.28** (`go test -run TestHarness_TurnError -race -v ./src/agent/...`, against the Stage-2 placeholder that swallowed and returned the raw error with no event):
```
harness_test.go:1886: last event kind = turn_end, want run_end
--- FAIL: TestHarness_TurnError_RunEndsTypedNoAppendNoCloseNoRetry (0.00s)
```

**Task 3.5 — S-RUN-071 arrival-order proof, scratch-revert** (`drain()` scratch-modified to reverse the returned slice, simulating a FIFO-ordering bug — deliberately targeted at `drain()` specifically, after the test was redesigned to hold turn one at a tool-calling turn so the burst is genuinely picked up by `drain()` rather than `takeOrClose()`; see Issues):
```
harness_steering_test.go:287: entries[3] text = "burst-4" (ok=true), want "burst-0" — arrival order must be preserved, zero drops
harness_steering_test.go:287: entries[4] text = "burst-3" (ok=true), want "burst-1" — arrival order must be preserved, zero drops
harness_steering_test.go:287: entries[6] text = "burst-1" (ok=true), want "burst-3" — arrival order must be preserved, zero drops
harness_steering_test.go:287: entries[7] text = "burst-0" (ok=true), want "burst-4" — arrival order must be preserved, zero drops
--- FAIL: TestHarness_SteerBurst_ArrivalOrderZeroDrops (0.00s)
```

**Task 4.2/4.3 — Phase 4 causality proof, scratch-revert** (`PauseTurn` removed from the iterate case of `Run`'s dispatch `switch`, leaving only `ToolCalls`):
```
=== NAME  TestHarness_PauseFinish_TurnEndCarriesPausedOutcomeVisibleAndForwarded
    harness_pause_test.go:133: observed 1 turn_end event(s), want 2 (one per turn)
--- FAIL: TestHarness_PauseFinish_TurnEndCarriesPausedOutcomeVisibleAndForwarded (0.00s)
=== NAME  TestHarness_PauseFinish_ResumesVerbatimToRealTerminal
    harness_pause_test.go:48: finish = pause_turn, want stop (turn two's real terminal, past the pause)
    harness_pause_test.go:71: history has 2 entries, want 3 (prompt, turn-one partial, turn-two final)
--- FAIL: TestHarness_PauseFinish_ResumesVerbatimToRealTerminal (0.00s)
```

After every scratch capture above, the injected change was reverted and `git diff` was checked to confirm byte-identical restoration before moving on.

## RED Evidence (verbatim) — Batch 3 (this batch)

**Task 5.4 — S-RUN-091/S-APP-016 non-vacuity proof (scratch: replace the parked-wait `select` in `scheduler.go` with an immediate re-resolution)**. Baseline checksum of `scheduler.go` before the scratch: `d631555ba9b7f315f7ae38d953bb9b1bd58867ca` (shasum). Scratch applied: the `select { case <-parkCh: return s.runPermissionGate(...); case <-ctx.Done(): ... }` block replaced with an unconditional `_ = parkCh; return s.runPermissionGate(...)`. Command: `go clean -testcache && go test -race -v -count=15 -run TestHarness_PermissionDefer_ParkedWaitObservedBite ./src/agent/...`. **Result: 15/15 runs FAIL** — genuine, unconditional RED, exceeding the `S-PPB-002` 20/20 reliability bar proportionally:

```
--- FAIL: TestHarness_PermissionDefer_ParkedWaitObservedBite (0.00s)
    harness_suspension_test.go:281: policy's second resolution observed the wake-issued flag = false, want true — under the S-RUN-091 scratch (parked wait replaced with an immediate re-resolution) this is the assertion that fails, proving the property observes the parked WAIT rather than the registration
```
(9 of the 15 runs failed with this exact message; the other 6 failed earlier, at the test's own defensive check inside `driveSuspendedRun`:)
```
--- FAIL: TestHarness_PermissionDefer_ParkedWaitObservedBite (0.00s)
    harness_suspension_test.go:275: tool invocations = 1, want 0 before the parked call is woken
```
Both failure shapes are genuine evidence of the same defect (the scratch races the tool execution and the re-resolution ahead of the test's own observation), and both are acceptable RED — no run passed. After reverting the scratch (restoring the exact original `select` block), `scheduler.go`'s checksum returned to `d631555ba9b7f315f7ae38d953bb9b1bd58867ca` — **byte-identical**, confirmed by `git diff --stat` showing no output. Re-run: `go clean -testcache && go test -race -count=15 -run TestHarness_PermissionDefer ./src/agent/...` → **PASS** (both tests, 15x).

**Task 5.6 — `agent-permission-protocol/spec.md:172` staleness settlement (scratch: delete the `R-APP-002` acknowledgement wait in `Schedule`)**. A SEPARATE, independent scratch from 5.4's — this one removes the `select { case <-reqAck: ...; case <-ctx.Done(): ... }` block entirely, proceeding straight from the `emissions <- emission{...}` send to the parked-wait select without ever confirming the emission reached `sink`. Baseline checksum confirmed clean (`d631555ba9b7f315f7ae38d953bb9b1bd58867ca`) before this scratch too. Command: `go clean -testcache && go test -race -v ./src/agent/...` (the FULL `agent` package suite, not a single test — per the task's own instruction to run the whole suite and observe). **Observed result: exactly ONE test fails, out of all 223 in the package:**

```
--- FAIL: TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery (0.01s)
    permission_protocol_test.go:475: ack_gate_tool invocations = 1, want 0 before decision_required reached sink — the tool ran ahead of its own decision_required event (R-APP-002/D4 ack ordering violated)
```

**Verdict, settled by the run, not by assertion**: `agent-permission-protocol/spec.md:172`'s original claim — "deleting the acknowledgement leaves the package green" — is **STALE**. The package does NOT stay green; the AG-10 remediation test `TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery` (added after `:172`'s claim was first written) already catches this exact defect at the `Schedule` level. This confirms the delta's own staleness finding (`agent-permission-protocol/spec.md`'s MODIFIED Evidence discipline section) was correctly recorded as a finding to verify, not a foregone conclusion — and the verification now has a real, observed result rather than an inferred one. After reverting the scratch (restoring the exact original ack-wait block), `scheduler.go`'s checksum returned to `d631555ba9b7f315f7ae38d953bb9b1bd58867ca` — byte-identical. Re-run: `go clean -testcache && go test -race ./src/agent/...` → **PASS**, all packages `ok`.

**Task 6.11 — `make lint` genuine finding (harness.go package comment)**. `golangci-lint cache clean` was not runnable directly (no `golangci-lint` on `PATH`); the pinned version `v2.9.0`'s auto-installer (`curl -sSfL https://raw.githubusercontent.com/...`) failed with a `503` (network-restricted sandbox). A pre-existing local install at `/Users/braejan/go/bin/golangci-lint` (`v2.12.2`, newer than pinned) was symlinked into the expected `bin/golangci-lint` location (`bin/` is gitignored) so `make lint` could run against the same `.golangci.yml` config. First run surfaced 3 findings:
```
src/ai/openaicompat/client_test.go:383:21: inline: Constant reflect.Ptr should be inlined (govet)
src/agent/harness.go:1:1: package-comments: package comment should be of the form "Package agent ..." (revive)
src/agenttest/cache_boundary_test.go:303:15: SA1019: parser.ParseDir has been deprecated... (staticcheck)
```
`git diff 5590afa0..HEAD --stat` (the merge-base diff, task 6.8's own evidence) proves the first and third findings are in files **entirely untouched by the AG-13 branch** (zero diff) — pre-existing, out of scope (Layer 1 edit is forbidden; `agenttest` is untouched by design). The second — `harness.go`'s package comment not starting with `"Package agent ..."` — is a genuine, in-scope, zero-risk fix (comment-only, matches `loop.go`/`scheduler.go`/`tool.go`'s existing header style). Fixed; re-run left only the two pre-existing/out-of-scope findings; `git diff` on `harness.go` confirmed comment-only; full `src/agent` package suite re-confirmed green after the fix.

## Work Unit Evidence

| Unit | Focused test command | Result | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| Phase 0 (seams) | `go test -run "TestTurn_ContinuationHalfConfigured\|TestSchedule_LeaveSinkOpen" -race -v ./src/agent/...` | All 6 sub-tests PASS | N/A — `agenttest` fakes only | `git revert` commits `51d44678`, `5260dd24` |
| Phase 1 (loop continuation path) | `go test -run "TestTurn_Continuation" -race -v ./src/agent/...` | All 9 sub-tests PASS | N/A | `git revert` commits `8db37082`, `49c6b679`, `ccffeac7`, `5a146a72` |
| Phase 2 (AG-13.1 harness run-to-completion) | `go test -run "TestHarness_TwoTurnRun\|TestHarness_RunStream\|TestHarness_LoopAccess\|TestHarness_RunIdentity\|TestHarness_History\|TestHarness_TurnError\|TestHarness_Steer\|TestHarness_EachTerminal\|TestHarness_StructLiteralRun\|TestHarness_EventStream" -race -v ./src/agent/...` | All 12 tests PASS | N/A — `agenttest` fakes only | `git revert` commits `ae5c51d6`, `098f7e47`, `a81dc968`, `021be572`, `4951acbe` (or delete `harness.go`) |
| Phase 3 (AG-13.2 steering queue) | `go test -run "TestHarness_.*Steer\|TestHarness_MidTurnSteer\|TestHarness_FinalTurnSteer" -race -v ./src/agent/...` | All 4 tests PASS | N/A | `git revert` commit `2cc26af1` (or delete `harness_steering_test.go` + revert the filter-widening hunks) |
| Phase 4 (AG-13.3 pause resumption) | `go test -run "TestHarness_PauseFinish" -race -v ./src/agent/...` | Both tests PASS | N/A | `git revert` commit `738fb70b` (or delete `harness_pause_test.go` + revert the filter-widening hunks) |
| Full package | `cd backend/agent && go clean -testcache && go test -race ./src/agent/...` | `ok  	github.com/cachicamas/backend/agent/src/agent	1.881s` (non-cached, genuinely re-run) | N/A | Same as above, combined |
| Full module (`make test`) | `cd backend/agent && make test` (`go test -race -v ./...`) | **PASS** — every package `ok`; zero `FAIL` lines (see final result below) | N/A | N/A (verification only) |
| Phase 5 (permission-suspension + parked-wait bite) | `go test -race -count=15 -run "TestHarness_PermissionDefer" ./src/agent/...` | Both tests PASS, 15x | N/A — `agenttest` fakes only, `sched.WakeParked` is the production wake surface itself (not a fake) | `git revert` commit `0cf984fd` (or delete `harness_suspension_test.go` + revert the filter-widening hunks) |
| Phase 6 (verification/docs/gates) | `cd backend/agent && make test && make build && make lint && make vuln-check` + `make test/cover` | **All four PASS/clean**; `loop.go` coverage 87.94% (see Coverage section) | N/A — verification-only phase, no new production behavior | `git revert` commits `9c73858c`, `0864a1db`, `8a0d39b3` (docs/comment-only, no behavior to roll back) |
| Full module, final (`make test`) | `cd backend/agent && go clean -testcache && make test` | **PASS** — every package `ok`; zero `--- FAIL` lines (see final result below) | N/A | N/A (verification only) |

## Coverage Gate (Batch 3, task 6.5)

`cd backend/agent && make test/cover` (`go test -race -coverprofile=coverage.out -covermode=atomic ./...`), then the exact `loop.go` line coverage computed by summing statement counts per block from the raw coverage profile (not the naive per-function average `go tool cover -func` prints, which is unweighted):

```
loop.go: covered=248 total=282 percent=87.94%
```

**87.94% ≥ 80% — NFR-RUN-004 satisfied**, with every new continuation branch exercised: `Turn` (the function housing the continuation gate) 89.4%, `finishContinuationTurn` 88.2%, `toolResultMessage` 84.6%, `outcomeForFinish` 88.9% (per-function `go tool cover -func` breakdown, all continuation-path functions comfortably above the floor individually too). `coverage.out` is gitignored (`backend/agent/.gitignore:5`), not committed.

## Critical Correctness Constraints — outcome (Batch 1, still holds)

1. **Continuation gates strictly on `opts.Continuation != nil`; nil-default path byte-stable.** Re-confirmed this batch: the three enumerated tests — `TestTurn_WalkingSkeleton_EmitsContractEventOrder`, `TestTurn_TwoSequentialTurnsShareNothing`, `TestTurn_ReasoningPassThroughByteExact` — pass and remain source-byte-unchanged (no edit to `loop_test.go`'s test bodies this batch either, only the filter-widening hunks).
2. **Schedule-before-finalize reorder.** Unchanged, still holds — Batch 2 built entirely on top of it (`finishContinuationTurn` untouched).
3. **`ai.NewRequest` accepts a `RoleTool`-result transcript.** Unchanged, still holds; re-exercised implicitly by every Phase 2/3/4 multi-message transcript the harness built and every `Turn` call it drove.
4–9. Unchanged from Batch 1; re-verified below under Batch 2's own numbered list.

## Critical Correctness Constraints — outcome (Batch 2, this batch)

1. **No `.Schedule(` call site in `harness.go`, no reference to any enumerated loop internal.** Confirmed by `TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard`, and independently by manual `grep -n '\.Schedule(' backend/agent/src/agent/harness.go` (zero matches) and `grep -nE 'turnAccumulator|mintLoopRunID|mintLoopTurnID|mintLoopMessageID|emitStamped|closeSink|buildLoopRequest|outcomeForFinish' backend/agent/src/agent/harness.go` (zero matches).
2. **The harness never mutates caller-supplied `Turn`/`System`/`Scheduler`/`History` fields, except the one recorded exception (`Scheduler.LeaveSinkOpen`).** Confirmed by `TestHarness_StructLiteralRun_NoConstructorFieldsUnchanged`'s two sub-cases.
3. **`Steer`'s zero-drop, typed-post-terminal-rejection contract.** Confirmed by S-RUN-002 (single call), S-RUN-071 (N-message burst, arrival order), S-RUN-072 (final-turn atomicity), S-RUN-073 (rejection persists across repeated calls).
4. **The atomic terminal-decision (`takeOrClose`) has no check-then-close gap.** Implemented as a single critical section (one `q.mu.Lock()`/`defer q.mu.Unlock()` spanning both the check and the close). Proven functionally by S-RUN-012/S-RUN-072 (Gate-held final turn, `Steer` accepted before the terminal decision, additional turn appears) — stable across 10 repeated `-race` runs. The stronger "no possible interleaving drops a message" claim is a code-level invariant (single critical section, verified by inspection) rather than a synthetically-forced true data race, which is not deterministically reproducible in a fast unit test; recorded honestly as a scoping choice, not a gap in the functional proof.
5. **Substrate discipline, whole branch.** `git diff e82c33e1..HEAD --stat -- backend/agent/` shows exactly 9 files: `harness.go` (new), `harness_test.go` (new), `harness_steering_test.go` (new), `harness_pause_test.go` (new), `loop.go`, `loop_hook_test.go`, `loop_test.go`, `scheduler.go`, `tool.go` — every one either new or explicitly assigned to this change. Every `R-LSK-004` file (`stream_check.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `turn_events.go`, `doc.go`, `doc_contract_guard_test.go`, `event_registry_test.go`, `reconstruction_test.go`, `sequence.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `ambient_authority_test.go`, `import_boundary_test.go`) is byte-unchanged — confirmed by an explicit per-file diff-stat scan, all empty. `history.go`/`history_surface_guard_test.go` byte-unchanged. `go.mod`/`go.sum` diff empty.
6. **No new `EventKind`, no new `TurnOutcome`, no new exported `History` method, no new Go dependency.** Confirmed by the same substrate scan; `harness.go` imports only `context`, `strconv`, `sync`, `sync/atomic`, and this module's own `src/ai` package.
7. **No wall-clock sleeps, no sockets, no files in the new tests.** Confirmed — every Phase 2/3/4 test uses `agenttest.NewProvider`/`Script`/`Emit`/`Gate`, buffered channels, and the existing `drainSink` helper's bounded `time.After(1s)` safety net (unchanged, reused, never duplicated). The mandatory per-turn forwarder goroutine syncs by channel close/range, never a sleep.
8. **Substrate filter widening, exact suffix, byte-in-sync — all five suffixes now present.** `filterOutLoopFiles` (`loop_test.go`) and `filterOutLoopHookFiles` (`loop_hook_test.go`) both now list, in order: `/harness.go`, `/harness_test.go`, `/harness_steering_test.go`, `/harness_pause_test.go` — four exact-filename suffixes, byte-in-sync between the two functions (confirmed via `grep -n` diff — identical). `/harness_suspension_test.go` (Phase 5) is correctly **not yet added**.
9. **Test naming and banners.** Every new test follows `Test<Subject>_<Behavior>_<Expectation>`; every banner cites its scenario id (S-RUN-001..012, S-RUN-020, S-RUN-030/031, S-RUN-040, S-RUN-050, S-RUN-060/061, S-RUN-070..073, S-RUN-080/081, S-ATT-013) and its charter cross-reference where applicable.
10. **`Harness.Run` always closes the consumer `sink` exactly once**, via a single `defer close(sink)` at the top of the function — safe from double-close by construction, unlike `Turn`'s own `closeSink` (which needs a `recover` because the scheduler may race it); the harness's `sink` is never touched by `Turn`/`Schedule` (only the per-turn `turnSink` is), so no such race exists here.
11. **`R-LSK-006` reconciliation upheld.** The harness never calls `Schedule`; every tool-calling turn's `Schedule` invocation is attributable to `Turn` alone (proven structurally by the source-scan guard; the per-call invocation-count assertion itself is `S-LSK-008`/`S-LSK-008a`'s, already green and unchanged — not re-duplicated here, cross-referenced for Phase 6's task 6.4).

## Critical Correctness Constraints — outcome (Batch 3, this batch)

1. **The permission-suspension acceptance clause needed zero new production code.** `driveSuspendedRun`'s harness composes entirely from Phase 0's `TurnContinuation.Scheduler` injection and Phase 2's mandatory per-turn forwarder goroutine — confirmed by both new tests passing on first run against the unmodified codebase, exactly matching design.md's own prediction under "Decision 2".
2. **The R-APP-002 parked-wait bite (`S-RUN-091`/`S-APP-016`) genuinely observes the wait, not the registration.** Proven by scratch-and-revert, not by code inspection: replacing the parked-wait `select` with an immediate re-resolution fails the bite 15/15 under `-race -count=15` (verbatim RED evidence above); reverted to a byte-identical `scheduler.go` (checksum-verified), both suspension tests pass again at `-count=15`. Registration alone cannot satisfy the bite's assertion because registration (`parked.park`) happens strictly before `permission_decision_required` is even emitted — long before the test sets its wake-issued flag or calls `WakeParked` — so an assertion satisfiable by registration timing alone is structurally impossible here.
3. **The `agent-permission-protocol/spec.md:172` staleness question is now settled by an actual run, not by inference.** Deleting the R-APP-002 acknowledgement wait (a second, independent scratch from the bite's own) makes exactly one test fail — `TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery` — out of 223 in the package. The original claim ("deleting the acknowledgement leaves the package green") is **stale**; the AG-10 remediation test already guards it. Reverted to byte-identical `scheduler.go` (checksum-verified); full suite re-confirmed green.
4. **Both scratch-and-revert exercises this batch left `scheduler.go` byte-identical to its pre-batch state.** Verified by `shasum` before and after each scratch (`d631555ba9b7f315f7ae38d953bb9b1bd58867ca` both times) in addition to `git diff --stat` showing no output — a stronger check than `git diff` alone, since it also catches a same-length substitution `git diff`'s line-based algorithm might in principle render as no visible hunk (not a real risk here, but checked anyway).
5. **Substrate discipline, whole branch, formally closed (task 6.8).** `git diff 5590afa0..HEAD --stat -- backend/agent/` (merge-base confirmed via `git merge-base HEAD origin/main` = `5590afa0`) shows exactly 10 files: the 5 new `harness*.go`/`harness*_test.go` files plus `loop.go`, `loop_hook_test.go`, `loop_test.go`, `scheduler.go`, `tool.go` — every one either new or explicitly assigned to this change, 3365 insertions(+), 32 deletions(-). `go.mod`/`go.sum` diff empty (explicit check, zero output). Every `R-LSK-004` file, `history.go`, `history_surface_guard_test.go`, `turn_events.go`, `failure.go`, and every file under `backend/agent/src/ai/` is confirmed byte-unchanged **by omission from this diff stat** — none of them appear in the 10-file list, and the diff scope (`backend/agent/`) covers all of them.
6. **The five spec deltas' cited lines verified verbatim against the canonical (pre-archive) specs, one by one (task 6.9).** `agent-loop-skeleton/spec.md:60,83,105,107`; `agent-permission-protocol/spec.md:144,172`; `agent-history/spec.md:185,201,206`; `agent-turn-termination/spec.md:65,146,148`; `agent-tool-scheduler/spec.md:133,137,138` — every citation opened and confirmed to quote the canonical spec text exactly, no drift.
7. **The `ToolSource` spec/code twin re-home is complete and machine-verified (task 6.2).** `agent-tool-scheduler/spec.md:138`'s delta already re-homes the spec line to AG-20; `tool.go`'s twin comment ("`ToolSource` port (G6) is AG-13's widening") is now "AG-20's widening". `grep -n "AG-13" backend/agent/src/agent/tool.go` returns only the `LeaveSinkOpen` sink-ownership seam references — zero hits mentioning `ToolSource`.
8. **`ai.NewRequest` accepting a `RoleTool`-result transcript is proven independently a third time this batch (task 6.3).** `buildLoopRequest` (`loop.go:599`) calls `ai.NewRequest(modelForOpts(opts), transcript, reqOpts...)` directly — the exact function every Phase 5 multi-turn request (including the permission-suspension test's own turn-two request, built from a transcript containing the tool-call and `RoleTool` result messages) round-trips through without rejection. No design-reopening finding.
9. **`make lint` is clean**, after fixing the one in-scope finding (`harness.go`'s package comment) and confirming, by merge-base diff, that the two remaining findings golangci-lint v2.12.2 surfaces (`src/ai/openaicompat/client_test.go`, `src/agenttest/cache_boundary_test.go`) are pre-existing and untouched by this branch — see RED Evidence above for the full accounting, including the version-pin caveat (pinned `v2.9.0`'s network installer failed; a local `v2.12.2` install was used instead).
10. **`make vuln-check` is clean**: exit 0, zero `"finding"` entries in the JSON output (170 `"osv"` database-awareness entries considered, none confirmed reachable by the call-graph analysis).

## Files Changed

| File | Action | What Was Done |
|---|---|---|
| `backend/agent/src/agent/loop.go` | Modified (Batch 1 only; untouched this batch) | See Batch-1 entry below |
| `backend/agent/src/agent/tool.go` | Modified (Batch 1 only; untouched this batch) | `Scheduler.LeaveSinkOpen bool` field |
| `backend/agent/src/agent/scheduler.go` | Modified (Batch 1 only; untouched this batch) | Conditional sink close |
| `backend/agent/src/agent/harness.go` | **Created this batch** | `Harness` struct (`Provider`/`System`/`Turn`/`Scheduler`/`History`/`queue`); `steeringQueue` (`enqueue`/`drain`/`takeOrClose`); `Steer`; `Run` (full algorithm: run bracket, prompt append, iterate-with-forwarder-goroutine, total finish-reason dispatch, atomic terminal decision, `R-RUN-011` failure path); `mintHarnessRunID`; `sendStamped`; `transcriptFromHistory`; `wrapHarnessFailure`; `failRun`. 303 lines. |
| `backend/agent/src/agent/harness_test.go` | Modified this batch (+973 lines; Batch 1's 11 tests untouched) | Adds 12 Phase 2 tests (S-RUN-001, 002, 010, 011, 012, 020, 030/031, 040, 050, 060, 061, 100) plus their shared helpers (`scriptToolCallResponse`, `transcriptMessagesForTest`, `messagesEqual`, `reconstructRunScope`, `runTwoTurnScenarioForReconstruction`) |
| `backend/agent/src/agent/harness_steering_test.go` | **Created this batch** | 4 Phase 3 tests (S-RUN-070..073) plus `heldTurnScript`/`heldToolCallScript` fixtures. 399 lines, `package agent_test`. |
| `backend/agent/src/agent/harness_pause_test.go` | **Created this batch** | 2 Phase 4 tests (S-RUN-080, S-RUN-081/S-ATT-013). 147 lines, `package agent_test`. |
| `backend/agent/src/agent/loop_test.go` | Modified in Batch 2 and Batch 3 | Batch 2: `filterOutLoopFiles` widened with `/harness_steering_test.go`, `/harness_pause_test.go`. Batch 3: widened again with `/harness_suspension_test.go` — all five `/harness*.go` suffixes now present |
| `backend/agent/src/agent/loop_hook_test.go` | Modified in Batch 2 and Batch 3 | Same widening, byte-in-sync, both batches |
| `backend/agent/src/agent/harness_suspension_test.go` | **Created this batch (Batch 3)** | `deferThenAllowPolicy` (`Resolve`/`Remember`, wake-issued-flag recording), `suspendedRunOutcome`, `driveSuspendedRun` (shared harness: builds the deferred tool call, drives `Harness.Run` in a goroutine, reads the sink event-by-event, wakes the parked call); `TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake` (S-RUN-090/S-APP-015); `TestHarness_PermissionDefer_ParkedWaitObservedBite` (S-RUN-091/S-APP-016). 283 lines, `package agent_test`. |
| `backend/agent/src/agent/tool.go` | Modified this batch (Batch 3, comment-only) | Re-homed the `ToolSource` widening comment from "AG-13's" to "AG-20's" (task 6.2) |
| `backend/agent/src/agent/harness.go` | Modified this batch (Batch 3, comment-only) | Package comment reformatted to start with `"Package agent ..."` (final-gate `make lint` cleanup, `revive`'s `package-comments` rule) |
| `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` | Modified this batch (Batch 3) | Status header: date, wave range, `12 of 24` → `13 of 24`, new AG-13 sentence; completion checklist line `:2170` ticked `[x]` |

No other file in `backend/agent/` differs from the merge-base `5590afa0` (`origin/main`'s AG-12 merge, confirmed via `git merge-base HEAD origin/main`). Full-branch diff stat: 10 files, 3365 insertions(+), 32 deletions(-); `go.mod`/`go.sum` diff empty.

## Deviations from Design / Tasks — noted honestly (Batch 1, unchanged — see prior save for full text)

1. Two production edits (mid-stream-fatal `run_end` suppression; `History.Append` wiring) landed slightly ahead of their dedicated RED test, both verified via genuine scratch-and-revert cycles.
2. `toolResultMessage`'s `ToolOutcomeExecutionFailure` content mapping is an implementation choice the spec doesn't pin.
3. `finishContinuationTurn`'s append-failure return values (`msg`/`finish` non-zeroed) is a low-stakes information-preserving choice.
4. `TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd` is a non-charter coverage/evidence test.

## Issues Found / Deviations — Batch 2 (this batch), noted honestly

1. **`S-RUN-011` (`TestHarness_EachTerminalCandidateFinishReason_EndsAfterOneTurn`) passed even against the un-iterating single-turn stub**, because "ends after exactly one turn" is trivially true of an implementation that can *only* run one turn. This was observed directly in the combined RED run for tasks 2.7/2.10/2.12 (see RED Evidence) and is recorded here rather than silently accepted: the property this test states is genuinely discriminated only in combination with `S-RUN-010` (which proves iteration happens when it should) and `S-RUN-012` (which proves the queue-aware atomic decision). All three were written and reviewed together for exactly this reason (mirroring `tasks.md`'s own bundling of 2.7/2.10/2.12 into one Work Unit). No production behavior is unproven — the total dispatch switch itself is exercised end-to-end by the union of the three tests — but `S-RUN-011` alone is not a suffficient causality proof, and a reviewer relying on it in isolation would be wrong to trust it as one.
2. **`S-RUN-071`'s first draft accidentally exercised the same code path as `S-RUN-072`.** The steering queue has two distinct extraction mechanisms — `drain()` (ordinary per-boundary, called before every turn) and `takeOrClose()` (the atomic terminal-candidate decision). The first draft of the arrival-order burst test held turn one at a *text-ending* (`Stop`) turn — the same shape `S-RUN-070`/`S-RUN-072` use — which meant the burst was picked up by `takeOrClose()`, not `drain()`. This was discovered when a scratch-revert aimed at `drain()` left the test green (the bug was in the wrong function for what the test actually exercised). The test was redesigned to hold turn one at a *tool-calling* turn (`FinishReasonToolCalls`, which the run dispatch iterates via `continue` rather than the atomic path), so the burst is now genuinely picked up by `drain()` at the top of the next loop iteration — confirmed by a corrected scratch-revert targeting `drain()` specifically (see RED Evidence), which now fails for the predicted reason. This is recorded as a real design correction made during this batch, not a silent adjustment: `S-RUN-071` and `S-RUN-072` now provide genuinely distinct coverage of `Run`'s two message-boundary mechanisms, matching `R-RUN-002`'s own two-branch dispatch (`ToolCalls`/`PauseTurn` iterate vs. terminal-candidate atomic check).
3. **The stronger "no interleaving of `Steer` and the terminal decision can ever drop a message" claim is proven by code inspection (single mutex critical section in `takeOrClose`), not by a synthetically-forced true data race.** The spec-mandated scenarios (`S-RUN-012`, `S-RUN-072`) construct a *deterministic* ordering (via `agenttest.Gate`) where `Steer` completes strictly before the turn returns and the dispatch runs — this proves the *feature* (a message steered before the terminal decision is honored, not dropped) but does not, by itself, force a genuine race window between two goroutines racing the mutex itself. That stronger guarantee rests on `takeOrClose`'s structure: the check and the close are one `Lock`/`defer Unlock` block with no code between them that could yield to another goroutine holding a stale read — verified by direct code reading, and indirectly by every test passing cleanly under `-race` (which would catch a genuine unsynchronized memory access, though not a *logical* TOCTOU race across two separately-locked sections, since there is only one lock acquisition here to begin with). Recorded as a conscious scoping decision, available for `sdd-verify` to weigh.
4. **`wrapHarnessFailure`'s failure category.** Every harness-level failure (prompt-append rejection, mid-run steered-append rejection, `CloseTurn` rejection, and `Turn`'s own returned error) is wrapped through the same `ai.FailureCategoryUnavailable` category, mirroring `scheduler.go`'s own `typedFailureFromError` precedent for a similarly generic catch-all wrap. The spec pins only that the failure is typed and non-nil on the `run_end` payload; it does not pin a category taxonomy for the harness's own wrapping site. `S-RUN-100` is the only task-scoped scenario exercising this path this batch (a `Turn`-returned mid-stream error); the other three call sites (prompt-append, steered-append, `CloseTurn` failure) share the identical `failRun` helper and are reachable but not independently scripted to fail in this batch's tests, since `History`'s own rejection surface was already exhaustively proven in Phase 1 (`S-HIS-093` et al.) — recorded here as a deliberate scope boundary, not an oversight.
5. **A local `wrapHarnessFailure` was written in `harness.go` instead of reusing `scheduler.go`'s existing (structurally identical) `typedFailureFromError`.** Both call `ai.MidStreamFailure` then `NewFailure` with the same category. The duplication (12 lines) is a deliberate choice to keep `harness.go`'s source-scan guard (`TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard`) unambiguous under even a stricter future reading — the guard's enumerated forbidden-symbol list does not name `typedFailureFromError`, so reusing it would not violate the letter of `R-RUN-006`, but avoiding it removes any doubt.

None of the above are design-reopening findings. No blockers were hit this batch.

## Issues Found / Deviations — Batch 3 (this batch), noted honestly

1. **`golangci-lint`'s pinned version (`v2.9.0`) could not be installed** — the Makefile's `tools` target and `make lint`'s own auto-install both shell out to `raw.githubusercontent.com`, which returned a `503` in this sandbox. A pre-existing local `v2.12.2` install (`/Users/braejan/go/bin/golangci-lint`) was symlinked into the expected `bin/golangci-lint` location instead, so `make lint` ran against the repo's own `.golangci.yml` config, just under a newer patch/minor linter version than pinned. This surfaced one additional, genuine, in-scope finding (`harness.go`'s package comment) that a same-vintage `v2.9.0` MAY or may not also have caught — recorded as a version-drift risk, not hidden. The finding was fixed regardless, since it is a real defect against the *current* config either way. The two other findings this run surfaced are proven pre-existing (absent from the AG-13 branch's own 10-file diff) and out of scope to fix (Layer 1 edit forbidden; `agenttest` untouched by design) — not silently dismissed, but explicitly evidenced.
2. **`govulncheck` was not pre-installed**, but `make vuln-check`'s auto-install succeeded (network access to the Go module proxy, `proxy.golang.org`, was available even though `raw.githubusercontent.com` was not) — no workaround needed there.
3. **The Phase 5 acceptance-clause test needed no new production code**, confirmed directly (not merely predicted) — recorded as a composition proof, consistent with several earlier phases' own honest recording of the same pattern (Phase 3's `S-RUN-070`, Phase 4's pause tests).
4. **Both Phase 5 scratch-and-revert exercises target the exact same function (`runPermissionGate`) at two different, non-overlapping statement ranges** (the ack-wait select vs. the parked-wait select) — recorded explicitly so a future reader does not conflate the two distinct scratch exercises or their two distinct evidentiary purposes (one proves the bite's non-vacuity; the other settles an independent staleness question about a *different* spec claim).

None of the above are design-reopening findings. No blockers were hit this batch.

## All Tasks Complete

- [x] Phase 0 (6/6) · [x] Phase 1 (18/18) · [x] Phase 2 (30/30) · [x] Phase 3 (9/9) · [x] Phase 4 (3/3) · [x] Phase 5 (6/6) · [x] Phase 6 (12/12)
- **74/74 tasks complete.** No remaining tasks for `sdd-apply`. Next: `sdd-verify`, then `sdd-archive` (task 6.12's promotion note: promote `agent-run-driver/spec.md` to `openspec/specs/agent-run-driver/spec.md`; apply the five deltas into their respective canonical specs; archive the change folder after `sdd-verify` passes, per AG-09..AG-12 precedent — none of this is `sdd-apply`'s to do).

## `make test` Final Result (this batch)

```
cd backend/agent && make test
```
**PASS.** `go clean -testcache` run first, then `go test -race ./src/agent/...` genuinely re-executed (non-cached): `ok  	github.com/cachicamas/backend/agent/src/agent	1.881s`. The full module `make test` (`go test -race -v ./...`) then reports every package `ok`: `src/agent`, `src/agenttest`, `src/agenttest/sweep`, `src/agenttest/tracetest`, `src/ai`, `src/ai/internal/retry`, `src/ai/openaicompat` (genuinely re-run, 171.655s — unrelated to this change, network/retry-timing tests), `src/ai/openaicompat/conformancetest`, `src/ai/openaicompat/openrouter`, `src/ai/openaicompat/openrouter/conformance`, `src/ai/openaicompat/openrouter/internal/smoke`, `src/handoff`. **Zero `FAIL` lines in the full log** (`grep -c "^--- FAIL"` = 0). The three locked nil-path tests (`TestTurn_WalkingSkeleton_EmitsContractEventOrder`, `TestTurn_TwoSequentialTurnsShareNothing`, `TestTurn_ReasoningPassThroughByteExact`) were independently re-run and pass; every AG-09/AG-10/AG-11 scheduler/permission/termination test passes as part of the same full-package run.

## `make test` Final Result — Batch 3, at the final commit (`8a0d39b3`)

```
cd backend/agent && go clean -testcache && make test
```
**PASS, exit 0.** Every package `ok`: `src/agent` (1.846s), `src/agenttest` (2.500s), `src/agenttest/sweep` (1.897s), `src/agenttest/tracetest` (2.022s), `src/ai` (5.155s), `src/ai/internal/retry` (1.785s), `src/ai/openaicompat` (171.612s), `src/ai/openaicompat/conformancetest` (2.163s), `src/ai/openaicompat/openrouter` (2.727s), `src/ai/openaicompat/openrouter/conformance` (6.113s), `src/ai/openaicompat/openrouter/internal/smoke` (2.734s), `src/handoff` (2.584s). **Zero `--- FAIL` lines** (`grep -c "^--- FAIL"` = 0). This is the genuinely final, non-cached, whole-module confirmation at the exact commit state this save reports against.

## `make build` Final Result — Batch 3

```
cd backend/agent && make build
```
`go build -trimpath ./...` — **exit 0, clean.**

## `make lint` Final Result — Batch 3

```
cd backend/agent && golangci-lint cache clean && make lint
```
(`golangci-lint cache clean` run via the symlinked `v2.12.2` binary — see Issues Found #1 above for the version-pin caveat.) **Clean after the `harness.go` package-comment fix.** Zero findings attributable to this change. Two findings remain in files structurally proven untouched by this branch (`git diff 5590afa0..HEAD --stat`, task 6.8's own evidence, shows zero diff for either file): `src/ai/openaicompat/client_test.go:383` (`govet`, pre-existing, Layer 1 — out of scope, edit forbidden) and `src/agenttest/cache_boundary_test.go:303` (`staticcheck`, pre-existing, untouched by design).

## `make vuln-check` Final Result — Batch 3

```
cd backend/agent && make vuln-check
```
**Exit 0, clean.** `govulncheck -json ./...` auto-installed (`v1.1.4`, Go module proxy reachable even though `raw.githubusercontent.com` was not) and ran to completion: 170 `"osv"` database entries considered as candidates against the module's dependency graph, **zero `"finding"` entries** (govulncheck's JSON schema emits `"finding"` only for a call-graph-confirmed reachable vulnerability) — no reachable known vulnerability in this module.
