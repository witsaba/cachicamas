# Tasks: AG-13 — Drive the multi-turn run (`cachicamas-agent-run-driver`)

> **Scenario count** (per `spec.md`'s Coverage table, MUST match `apply-progress.md`/`verify-report.md`): `agent-run-driver` itself is **12 requirements → 25 scenarios, of which 2 are bites** (`S-RUN-061`, `S-RUN-091`). The five cross-cut deltas add **18 further scenarios** this change is responsible for closing: `agent-loop-skeleton` +5 (`S-LSK-013..017`), `agent-permission-protocol` +2 incl. 1 bite (`S-APP-015/016`, jointly owned with `S-RUN-090/091`), `agent-history` +7 (`S-HIS-090..096`), `agent-turn-termination` +1 (`S-ATT-013`, jointly owned with `S-RUN-009`/`S-RUN-080/081`), `agent-tool-scheduler` +3 (`S-TLS-013..015`). **Total new/changed evidence obligations: 43.** Design decisions in `design.md` are binding; this file does not re-derive them.

## Substrate Filter Closure (authoritative — closes `R-LSK-004`/`R-RUN-012`)

`filterOutLoopFiles` (`loop_test.go:831`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907`) MUST widen with exactly these five exact-filename suffixes, byte-in-sync, no wildcard/prefix/directory pattern:

```
/harness.go
/harness_test.go
/harness_steering_test.go
/harness_pause_test.go
/harness_suspension_test.go
```

`loop.go`, `tool.go` and `scheduler.go` are **already** in both filters — no addition needed for them. Land each new-file suffix pair in the SAME commit as the file that first needs it (AG-11/AG-12 discipline).

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1630–2660 (proposal forecast: production+test Go 930–1560 — harness.go 250–400, loop.go delta 80–150, scheduler.go/tool.go 0–60, steering queue 60–100, harness test files 500–750, loop_test.go filters ~40–100; SDD markdown incl. 5 deltas 700–1100) |
| 400-line budget risk | High — exceeds even the raised 1000-line pre-authorized ceiling |
| Chained PRs recommended | No |
| Suggested split | single PR — AG-13 only |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

ONE PR (`size:exception` pre-authorized by the user for this milestone). Runtime harness is N/A throughout: no real provider, no real tool, no socket/file/wall-clock — `agenttest` scripts and `agenttest.Gate` only (Threat Matrix: N/A, `design.md`).

| Unit | Goal | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | Seams: `TurnContinuation`, `Continuation` field, `Scheduler.LeaveSinkOpen` (Phase 0) | `go test -run "TestTurn_Continuation|TestSchedule_LeaveSinkOpen" -race ./src/agent/...` | N/A | revert `loop.go`/`tool.go`/`scheduler.go` field diffs |
| 2 | Loop continuation path: brackets, reorder, transcript commits (Phase 1) | `go test -run "TestTurn_Continuation" -race ./src/agent/...` | N/A | revert `loop.go` continuation-mode branches |
| 3 | Harness run-to-completion, run-scope reconstruction, no-privileged-channel (Phase 2) | `go test -run "TestHarness_TwoTurnRun|TestHarness_RunStream|TestHarness_LoopAccess|TestHarness_RunIdentity|TestHarness_History|TestHarness_TurnError|TestHarness_Steer|TestHarness_EachTerminal" -race ./src/agent/...` | N/A | revert `harness.go`/`harness_test.go` |
| 4 | Steering queue (Phase 3) | `go test -run "TestHarness_.*Steer" -race ./src/agent/...` | N/A | revert `harness_steering_test.go` + queue diffs |
| 5 | Pause resumption (Phase 4) | `go test -run "TestHarness_PauseFinish" -race ./src/agent/...` | N/A | revert `harness_pause_test.go` |
| 6 | Permission suspension + parked-wait bite (Phase 5) | `go test -race -count=15 -run "TestHarness_PermissionDefer" ./src/agent/...` | N/A | revert `harness_suspension_test.go` |
| 7 | Cross-cut, spec deltas, docs, final gates (Phase 6) | `cd backend/agent && make test && make lint && make build && make vuln-check` | N/A | revert promotion/docs commits |

## Phase 0: Seams, RED-first (`R-LSK-001`, `R-TLS-012`)

- [x] 0.1 RED — create `harness_test.go` (`package agent_test`): `TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission` (`S-LSK-014`). Land in the SAME commit as widening both substrate filters with `/harness.go` and `/harness_test.go`. Expect FAIL: `agent.TurnContinuation` undefined.
- [x] 0.2 GREEN — `loop.go`: add exported `TurnContinuation{Run RunID; Stamper *LaneStamper; Scheduler *Scheduler; History *History}` and `TurnOptions.Continuation *TurnContinuation`; validate all-or-nothing at `Turn` entry, before any emission — half-configured returns `ai.Invalid(ai.ErrEmpty, ai.At("continuation", …))` naming the absent member, sink left as given, zero events emitted.
- [x] 0.3 GREEN confirm — `S-LSK-014` passes; sink observably received nothing.
- [x] 0.4 RED — `harness_test.go`: `TestSchedule_LeaveSinkOpenZeroDefault_ClosesSinkUnchanged` (`S-TLS-013`), `TestSchedule_LeaveSinkOpenSet_CallerOwnsClose` (`S-TLS-014`). Expect FAIL: `Scheduler.LeaveSinkOpen` undefined.
- [x] 0.5 GREEN — `tool.go`: add `Scheduler.LeaveSinkOpen bool` (zero = false = AG-09 behavior) to the struct at `:229`. `scheduler.go`: make the close-third-step at `:219` conditional on `!LeaveSinkOpen`; every other `Schedule` step (parked-set clear, emissions close, dispatcher join, ordered rejoin) unchanged in behavior and order.
- [x] 0.6 GREEN confirm — `S-TLS-013`/`S-TLS-014` pass; keyed-literal AG-09/AG-10 scheduler tests still construct `Scheduler{...}` and stay green file-unchanged (early check for `S-TLS-015`, re-confirmed at 6.6).

## Phase 1: Loop continuation path (`R-LSK-001`, `R-HIS-010`)

- [x] 1.1 RED — `harness_test.go`: `TestTurn_ContinuationNonNil_NoRunBracketsSharedIdentityAndLane` (`S-LSK-013`, non-nil half: text-only turn). Expect FAIL: continuation is validated but ignored — brackets still emitted, identity still minted fresh.
- [x] 1.2 GREEN — `loop.go`: gated on `opts.Continuation != nil` — join `cont.Run` as `runID` instead of minting, use `cont.Stamper` instead of a fresh one (`loop.go:187-189` locals); emit **no** `run_start`/`run_end` on any path including the mid-stream fatal path (`turn_end(Aborted)` still emitted); `TurnID` still minted fresh per call.
- [x] 1.3 GREEN confirm — `S-LSK-013` non-nil half passes; nil-continuation half (zero-value `TurnOptions`) still emits brackets and mints fresh identity, byte-stable.
- [x] 1.4 RED — own task, per design's flagged reorder risk: `TestTurn_ContinuationToolCall_ScheduleBeforeFinalize_EventsInsideTurnBracket` (`S-LSK-013` tool-calling half) — asserts tool/permission events land **inside** the open turn bracket and **before** `turn_end`, and `CheckStream` accepts. Expect FAIL: today's finalize-first ordering puts tool events after `turn_end`, and `CheckStream` rejects a `PlacementTurn` event outside an open turn / after a Terminal `run_end`.
- [x] 1.5 GREEN — `loop.go`, continuation path only: call `results := sched.Schedule(...)` **before** `finalize()` (capturing the rejoin instead of the `_ =` discard at `:265`); `reconstructMessage` (continuation only) additionally appends `ai.ToolCall` parts from `t.toolCalls` with provider-exact bytes. Nil path keeps finalize-first byte-stable.
- [x] 1.6 GREEN confirm — `S-LSK-013` tool-calling half passes; `CheckStream` accepts the continuation-path event slice unmodified.
- [x] 1.7 RED — `harness_test.go`: `TestTurn_ContinuationCommitsAssistantAndToolResults_OpenSetEmptyAtClose` (`S-HIS-090`). Expect FAIL: no transcript commit exists yet — `CloseTurn` would reject an open call.
- [x] 1.8 GREEN — `loop.go`, continuation path, after finalize+rejoin: append the turn's assistant message (skip if zero content); append one `ai.RoleTool` result message per rejoin result in call order (`ai.NewToolResult` success / `ai.NewToolFailure` both failure outcomes). `Turn` does NOT call `CloseTurn` — that stays the run driver's.
- [x] 1.9 GREEN confirm — `S-HIS-090`: store holds assistant message with tool-call args + matching result; a subsequent `CloseTurn` on that history succeeds because no call is open.
- [x] 1.10 RED — `TestTurn_ContinuationEmptyContent_AppendsNothing` (`S-HIS-091`).
- [x] 1.11 GREEN confirm — covered by 1.8's skip-if-zero-content branch; store unchanged, `Turn` returns no error.
- [x] 1.12 RED — `TestTurn_ContinuationMixedOutcomes_OneResultPerCallInOrder` (`S-HIS-092`, success + both failure outcomes mixed).
- [x] 1.13 GREEN confirm — covered by 1.8; each failure outcome carried by the Layer 1 failure form, none by a content sentinel.
- [x] 1.14 RED — `TestTurn_ContinuationAppendFailure_TypedErrorReturned` (`S-HIS-093`) — construct a continuation `History` already holding an unresolved state so the commit append is rejected.
- [x] 1.15 GREEN — `loop.go`: an append failure on the continuation path returns a non-nil typed error from `Turn`; it MUST NOT be swallowed and the turn MUST NOT be reported successful.
- [x] 1.16 GREEN confirm — `S-HIS-093` passes; the caller's run terminates through `R-RUN-011`'s failure path (cross-checked again at 2.29-2.31).
- [x] 1.17 RED — `TestTurn_ContinuationNil_HistorySurfaceGuardStaysGreen` (`S-HIS-094`).
- [x] 1.18 GREEN confirm — nil continuation touches no transcript store; `history_surface_guard_test.go` passes source-unchanged.

## Phase 2: AG-13.1 — Harness run to completion (`R-RUN-001..007`, `R-RUN-011`)

- [ ] 2.1 RED — `TestHarness_StructLiteralRun_NoConstructorFieldsUnchanged` (`S-RUN-001`). Expect FAIL: compile error, `agent.Harness` undefined.
- [ ] 2.2 GREEN — create `harness.go`: `Harness{Provider ai.ModelProvider; System string; Turn TurnOptions; Scheduler *Scheduler; History *History; queue steeringQueue}`; method stubs `Run(ctx, prompt, sink) (ai.Message, ai.FinishReason, error)` / `Steer(msg) error`. Minimal `Run`: resolve nil defaults into locals (construct `Scheduler` with `LeaveSinkOpen: true` if nil, `NewHistory()` if nil), mint `run-hrn-<n>` via a package-local atomic counter, emit `run_start` via `NewRunStart` + shared `stamper.Stamp`, append `prompt`, call `Turn` once with continuation, emit `run_end`, return — single-turn happy path only at this increment.
- [ ] 2.3 GREEN confirm — `S-RUN-001`: no constructor call, caller fields unchanged except the recorded `LeaveSinkOpen` exception, exactly two exported methods.
- [ ] 2.4 RED — `TestHarness_SteerAfterTerminal_TypedRejectionNoSilentDrop` (`S-RUN-002`).
- [ ] 2.5 GREEN — implement `steeringQueue` closed-check + typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`.
- [ ] 2.6 GREEN confirm — `S-RUN-002`.
- [ ] 2.7 RED — **charter AG-13.1 sc.1**: `TestHarness_TwoTurnRun_CompletesToTerminal` (`S-RUN-010`) — two-script fake provider (tool call, then final text); consumer drains sink; asserts full kind order + `Run`'s returned values.
- [ ] 2.8 GREEN — extend `Run` to iterate: loop calling `Turn` with continuation; per-turn channel + **mandatory** forwarder goroutine relaying events to the consumer sink while the turn is in flight (so `permission_decision_required` is observable before the turn returns); dispatch on finish reason (`ToolCalls`/`PauseTurn` iterate; other members terminal-candidate); `CloseTurn` at each boundary.
- [ ] 2.9 GREEN confirm — `S-RUN-010` full kind order, `Run` returns turn-two's message/finish/nil error.
- [ ] 2.10 RED — `TestHarness_EachTerminalCandidateFinishReason_EndsAfterOneTurn` (`S-RUN-011`).
- [ ] 2.11 GREEN confirm — dispatch is total over `ai.FinishReason`'s vocabulary; each terminal-candidate member ends the run after exactly one turn with an empty queue.
- [ ] 2.12 RED — `TestHarness_SteerNearTerminal_AtomicQueueCheckYieldsAdditionalTurn` (`S-RUN-012`, Gate-held final turn).
- [ ] 2.13 GREEN — implement the atomic terminal-decision: under the queue mutex, non-empty → take + iterate; empty → mark queue closed in the same critical section, then terminate. No check-then-close race.
- [ ] 2.14 GREEN confirm — `S-RUN-012`: `Steer` returned nil, additional turn bracket appears, run terminates only after it.
- [ ] 2.15 RED — `TestHarness_EventStream_OneRunBracketContiguousLane_CheckStreamAccepts` (`S-RUN-020`).
- [ ] 2.16 GREEN confirm — one run-open/run-close pair, N turn brackets nested, sequence 1..N with no gap/repeat/restart; `CheckStream` accepts unmodified (no new production code expected — pins the composition of Phases 0–1 + 2.8).
- [ ] 2.17 RED — `TestHarness_RunIdentity_ConsistentAcrossEventsAndProvenanceDistinct` (`S-RUN-030`, `S-RUN-031`).
- [ ] 2.18 GREEN confirm/fix — every event in one run carries the same run identity, asserted by this test, not `CheckStream`; harness-driven identity carries `run-hrn-` prefix, distinct from the loop's `run-lsk-` prefix.
- [ ] 2.19 GREEN confirm — `S-RUN-030`/`S-RUN-031`.
- [ ] 2.20 RED — `TestHarness_History_AlternatingTranscriptEveryPairMatched` (`S-RUN-040`).
- [ ] 2.21 GREEN confirm — read-back holds prompt, turn-one assistant+call, matching result, turn-two assistant, in order; no duplicate/missing; `S-RUN-041` cross-checked at 6.4/6.8 (closed-route + import/ambient guards, source-unchanged).
- [ ] 2.22 RED — `TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard` (`S-RUN-050`).
- [ ] 2.23 GREEN — write the source-scan guard (`scheduler_test.go` regex precedent): `harness.go` contains no reference to any enumerated loop internal (`turnAccumulator`, loop identity minters, `emitStamped`, `closeSink`, the request builder, the finish-reason mapper) and no `.Schedule(` call site.
- [ ] 2.24 GREEN confirm — `S-RUN-050`; every capability test declares `package agent_test`; folds in `S-LSK-017`'s per-tool schedule-invocation-count == 1 assertion, attributable to `Turn` only, never the harness.
- [ ] 2.25 RED — **charter AG-13.1 sc.2**, written together per the mirror-of-`S-LSK-003a/b` pattern: `TestHarness_RunStream_ReconstructsHistoryAtRunScope` (`S-RUN-060`) AND `TestHarness_RunStream_ReconstructionBiteDropsTurnTwoEvent` (`S-RUN-061`, bite). Expect FAIL: no run-scope reconstruction helper exists.
- [ ] 2.26 GREEN — implement run-scope reconstruction: partition the replayed events by turn bracket, reconstruct each turn's messages + tool outcomes, deep-equal against `history.Entries()`.
- [ ] 2.27 GREEN confirm — `S-RUN-060` deep-equal holds; `S-RUN-061` PASSES because, with exactly one turn-two event dropped from a copy of the slice, the comparator REPORTS divergence — record the divergence output as evidence (non-vacuity proof, mirrors `S-LSK-003a`/`S-LSK-003b`).
- [ ] 2.28 RED — `TestHarness_TurnError_RunEndsTypedNoAppendNoCloseNoRetry` (`S-RUN-100`).
- [ ] 2.29 GREEN — implement the failure path: on non-nil `Turn` error, emit `run_end(RunOutcomeFailed, failure)` via `ai.MidStreamFailure` + `NewFailure`, close the sink, return the error; no append, no `CloseTurn`, no retry/fallback.
- [ ] 2.30 GREEN confirm — `S-RUN-100`: typed closing brackets then `run_end(Failed)` then sink close; transcript holds only what was committed before failure; provider recorded exactly one request.

## Phase 3: AG-13.2 — Steering queue (`R-RUN-008`)

- [ ] 3.1 RED — **charter AG-13.2 sc.1**: `TestHarness_MidTurnSteer_EntersAtBoundaryBeforeNextProviderCall` (`S-RUN-070`) in new `harness_steering_test.go`. Land in the SAME commit as widening both filters with `/harness_steering_test.go`. Uses `agenttest.Gate` + a request-recording provider wrapper (the `newCtxRecordingProvider` precedent).
- [ ] 3.2 GREEN — implement `steeringQueue` (mutex + FIFO slice + closed flag) and the drain-at-boundary step ahead of building the next request transcript.
- [ ] 3.3 GREEN confirm — `S-RUN-070`: in-flight turn's events unchanged from the un-steered baseline; steered message between turn-1 and turn-2 messages; turn-2's recorded request transcript contains it.
- [ ] 3.4 RED — **charter AG-13.2 sc.2, first Then**: `TestHarness_SteerBurst_ArrivalOrderZeroDrops` (`S-RUN-071`) — N `Steer` calls from a second goroutine in a test-determined order.
- [ ] 3.5 GREEN confirm — all N appear in arrival order, none missing, none duplicated (via 3.2's FIFO-under-mutex).
- [ ] 3.6 RED — **charter AG-13.2 sc.2, second Then**: `TestHarness_FinalTurnSteer_YieldsNewTurn` (`S-RUN-072`) — Gate-held final turn, steered before release.
- [ ] 3.7 GREEN confirm — via 2.13's atomic terminal decision: `Steer` returns nil, an additional turn bracket appears, run terminates only after it.
- [ ] 3.8 RED — `TestHarness_SteerAfterTermination_QueueClosedTypedRejection` (`S-RUN-073`).
- [ ] 3.9 GREEN confirm — typed rejection of `R-RUN-001`/2.5, transcript unchanged; queue is closed, not merely empty.

## Phase 4: AG-13.3 — Pause resumption (`R-RUN-009`)

- [ ] 4.1 RED — **charter AG-13.3, both Thens**: `TestHarness_PauseFinish_ResumesVerbatimToRealTerminal` (`S-RUN-080`, `S-RUN-081`) in new `harness_pause_test.go`. Land in the SAME commit as widening both filters with `/harness_pause_test.go`. Script 1: partial text + reasoning with a non-empty round-trip token + `PauseTurn`; script 2: final text + terminal reason.
- [ ] 4.2 GREEN confirm — via Phase 1's continuation-mode partial append (1.8) and Phase 2's `PauseTurn`-iterates dispatch (2.8): transcript entry for the paused turn deep-equal to the returned message including round-trip token bytes; turn-two's recorded request contains it byte-verbatim; run ends `Completed`.
- [ ] 4.3 GREEN confirm — `turn_end` carries `TurnOutcomePaused`, forwarded unrewritten; outcome differs from both finished and aborted; no assertion reads an `ai.FinishReason` to establish it. Jointly closes `S-ATT-013`.

## Phase 5: Permission-suspension acceptance clause + `R-APP-002` parked-wait bite (`R-RUN-010`)

- [ ] 5.1 RED — **fourth acceptance clause**: `TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake` (`S-RUN-090`, jointly `S-APP-015`) in new `harness_suspension_test.go`. Land in the SAME commit as widening both filters with `/harness_suspension_test.go`. Policy defers resolution #1, allows #2; test reads the consumer sink event by event.
- [ ] 5.2 GREEN confirm — via Phase 0's injected `*Scheduler` and Phase 2's forwarder goroutine: on reading `decision_required`, a non-blocking read of the run's completion channel shows not-returned and zero tool invocations; `WakeParked(callID)` returns nil; run completes; suspension events sit inside turn one's bracket; `CheckStream` accepts.
- [ ] 5.3 GREEN confirm — record evidence; no third resolution path, no timeout (context cancellation propagated unmodified is the only other exit).
- [ ] 5.4 RED bite — `TestHarness_PermissionDefer_ParkedWaitObservedBite` (`S-RUN-091`, jointly `S-APP-016`): policy's Resolve#2 asserts a `wake-issued` flag set immediately before `WakeParked` runs. Scratch-replace the parked-wait select with an immediate re-resolution; rerun under `go test -race -count=15 ./src/agent/`; confirm FAILS because the flag is unset at re-resolution (proves the assertion observes the parked **wait**, not the registration); record the `-count=15` output in `apply-progress.md`; revert the scratch.
- [ ] 5.5 GREEN confirm — after revert, `git diff` for `scheduler.go`/`harness.go` is byte-empty; `S-RUN-091`/`S-APP-016` pass at `-count=15`.
- [ ] 5.6 **Settle the `agent-permission-protocol/spec.md:172` staleness question by running it, not asserting it.** Scratch-delete the AG-10 acknowledgement wait in `Schedule` (`spec.md:172`'s named gap); run the full `agent/` package suite; record the OBSERVED result — whether `TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery` (AG-10 remediation) already fails it (making `:172`'s "leaves the package green" claim stale) or whether the package genuinely stays green. Record the result in `apply-progress.md` either way; revert the scratch.

## Phase 6: Substrate, spec/code twins, docs, final gates

- [ ] 6.1 Confirm both substrate filters (`S-RUN-111`, `S-LSK-016`): identical entry sets, exactly the five `/harness*.go` suffixes, no wildcard/prefix/directory pattern, `loop.go`/`tool.go`/`scheduler.go` already present and untouched in the filter lists.
- [ ] 6.2 **`ToolSource` spec/code twin re-home, SAME PR.** Confirm the promoted `agent-tool-scheduler` delta re-homes `agent-tool-scheduler/spec.md:138` to AG-20 (already drafted in `specs/agent-tool-scheduler/spec.md`); re-home the twin code comment at `backend/agent/src/agent/tool.go:239-240` ("`ToolSource` port (G6) is AG-13's widening" → AG-20) in the same commit. Verify `grep -n "AG-13" backend/agent/src/agent/tool.go` returns nothing about a `ToolSource` widening.
- [ ] 6.3 **Prove `ai.NewRequest` accepts a `RoleTool`-result transcript.** Run a direct check (or confirm via `S-RUN-070`'s request-recorder evidence) that a transcript built from `history.Entries()` — call message, then `RoleTool` result message — round-trips through `ai.NewRequest` without rejection. If it does NOT: this is a design-reopening finding, record it explicitly and do not silently paper over it.
- [ ] 6.4 Confirm the three enumerated existing tests (`TestTurn_WalkingSkeleton_EmitsContractEventOrder`, `TestTurn_TwoSequentialTurnsShareNothing`, `TestTurn_ReasoningPassThroughByteExact`) pass with source **byte-unchanged** on the nil-continuation path (`S-LSK-015`), plus every AG-09/AG-10 scheduler test and every AG-10/AG-11 permission/termination test (`S-RUN-112`).
- [ ] 6.5 **Coverage gate.** `TestTurn_CoverageGate` — confirm `loop.go` line coverage ≥ 80% under `make test`, including every new continuation branch (`NFR-RUN-004`); extend harness tests if any branch is unexercised.
- [ ] 6.6 Confirm every AG-09/AG-10 scheduler test passes file-unchanged (`S-TLS-015`, final check after Phase 0's early pass at 0.6).
- [ ] 6.7 Confirm import guard and ambient-authority guard pass with zero changes over the production and test closures (`harness.go` imports stdlib + `ai` only; no clock, no I/O — the forwarder syncs by channel close, never a sleep); confirm the stamper is touched only between turns (`NFR-RUN-003`), race-free under `-race`.
- [ ] 6.8 Confirm the merge-base diff (`S-RUN-110`, `S-HIS-095`/`096`): only allowlisted files differ under `backend/agent/src/agent/`; every `R-LSK-004` file byte-unchanged; `history.go` byte-unchanged; `go.mod`/`go.sum` diff empty; every-kind-constructible guard at its committed count (AG-13 adds zero); `history_surface_guard_test.go` passes source-unchanged, route set equal in both directions.
- [ ] 6.9 Confirm the five spec deltas' cited lines land verbatim in the promoted specs (already drafted under `specs/`): `agent-loop-skeleton` `R-LSK-001/002/004/006`; `agent-permission-protocol` `R-APP-002`; `agent-history` `R-HIS-010` + `NFR-HIS-003`; `agent-turn-termination` `R-ATT-004`; `agent-tool-scheduler` `R-TLS-012`.
- [ ] 6.10 **Docs.** Tick `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:2170` — "A multi-turn run completes with steering, pause resumption, and a complete event story — closed by AG-13"; bump the Layer 2 wave/milestone status header counters to **13/24** (status-header prose only — no count in any requirement or spec text).
- [ ] 6.11 **Final gates.** `cd backend/agent && make test` green under `-race` — all 25 `agent-run-driver` scenarios + 2 bites + 18 delta scenarios recorded; `golangci-lint cache clean && make lint` clean; `make build` clean; `make vuln-check` clean (explicit — not in `make all`).
- [ ] 6.12 Note for `sdd-archive` (not this phase): promote `agent-run-driver/spec.md` to `openspec/specs/agent-run-driver/spec.md`; apply the five deltas into their respective canonical specs; archive the change folder after `sdd-verify` passes, per AG-09..AG-12 precedent.

## Coverage Table

| Requirement | Scenario(s) | Task(s) |
|---|---|---|
| R-RUN-001 | S-RUN-001, S-RUN-002 | 2.1–2.6 |
| R-RUN-002 | S-RUN-010, S-RUN-011, S-RUN-012 | 2.7–2.14 |
| R-RUN-003 | S-RUN-020 | 2.15–2.16 |
| R-RUN-004 | S-RUN-030, S-RUN-031 | 2.17–2.19 |
| R-RUN-005 | S-RUN-040, S-RUN-041 | 2.20–2.21, 6.4, 6.8 |
| R-RUN-006 | S-RUN-050 | 2.22–2.24 |
| R-RUN-007 | S-RUN-060, S-RUN-061 (bite) | 2.25–2.27 |
| R-RUN-008 | S-RUN-070, S-RUN-071, S-RUN-072, S-RUN-073 | 3.1–3.9 |
| R-RUN-009 | S-RUN-080, S-RUN-081 | 4.1–4.3 |
| R-RUN-010 | S-RUN-090, S-RUN-091 (bite) | 5.1–5.6 |
| R-RUN-011 | S-RUN-100 | 2.28–2.30 |
| R-RUN-012 | S-RUN-110, S-RUN-111, S-RUN-112 | 6.1, 6.4, 6.8 |
| R-LSK-001 | S-LSK-013, S-LSK-014 | 0.1–0.3, 1.1–1.6 |
| R-LSK-002 | S-LSK-015 | 6.4 |
| R-LSK-004 | S-LSK-016 | 6.1, 6.8 |
| R-LSK-006 | S-LSK-017 | 2.22–2.24 |
| R-APP-002 | S-APP-015, S-APP-016 (bite) | 5.1–5.5 |
| R-HIS-010 | S-HIS-090..094 | 1.7–1.18 |
| NFR-HIS-003 | S-HIS-095, S-HIS-096 | 6.8 |
| R-TLS-012 | S-TLS-013, S-TLS-014, S-TLS-015 | 0.4–0.6, 6.6 |
| R-ATT-004 | S-ATT-013 | 4.1–4.3 |
| agent-permission-protocol `:172` staleness | (finding, not a scenario) | 5.6 |
| `ai.NewRequest`/`RoleTool` proof | (finding, not a scenario) | 6.3 |
| `ToolSource` spec/code twin | (finding, not a scenario) | 6.2 |

## Constraints restated (binding on every task above)

No `R-LSK-004` substrate edit — `stream_check.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `turn_events.go`, `doc.go`, `doc_contract_guard_test.go`, `event_registry_test.go`, `reconstruction_test.go`, `sequence.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `go.mod`, `go.sum`. No new `EventKind`. No new exported method on `History`. No `L2C-08` doc-contract row. No retry, no compaction, no cancellation vocabulary beyond unmodified `ctx` propagation. No Layer 1 edit (`backend/agent/src/ai/**` untouched). No new Go dependency. No real provider, tool, socket, file, or wall-clock sleep — synchronize with `agenttest.Gate` and channel reads only.

## Risks

- **Ordering dependency, Phase 0 before Phase 1 before Phase 2**: the continuation seam and `LeaveSinkOpen` must exist before the reorder task (1.4) can even compile its RED, and the reorder must exist before the harness can drive a tool-calling turn to completion (2.7) — apply MUST NOT reorder these phases.
- **Filter-widening landmine**: every RED test added under `backend/agent/src/agent/` trips both substrate guards until its filter suffix lands in the SAME commit as the file — repeat the AG-11/AG-12 discipline exactly, five times.
- **The schedule-before-finalize reorder is continuation-path only**: a change that accidentally reorders the nil path too would silently break `S-LSK-015`'s file-unchanged guarantee on three locked-in tests; task 1.5 MUST gate strictly on `opts.Continuation != nil`.
- **`S-RUN-091`/`S-APP-016` is a scratch-and-revert bite, `S-RUN-061` is not** — do not conflate the two bite mechanics. `S-RUN-061` permanently mutates a copied event slice as its own test input; `S-RUN-091` temporarily mutates production code and must be reverted with a verified-empty `git diff`.
- **`:172` staleness must be RUN, not inferred** — task 5.6 is a distinct obligation from the bite in 5.4; skipping the actual scratch-delete-and-run step and asserting either outcome from prose is a spec violation (`agent-permission-protocol/spec.md`'s Evidence discipline).
- **`ai.NewRequest`/`RoleTool` compatibility is unproven until run** — both `design.md` and the spec flag this; task 6.3 is not optional bookkeeping, a rejection here reopens Decision 1's design.
