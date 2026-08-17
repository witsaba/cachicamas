# Apply Progress: AG-13 — Drive the multi-turn run (`cachicamas-agent-run-driver`)

> This batch's scope: **Phase 0 and Phase 1 ONLY** (tasks 0.1–0.6, 1.1–1.18). Phases 2–6
> (the `Harness` type, steering, pause resumption, permission-suspension bite, and the
> cross-cut/docs/final-gates phase) are explicitly out of scope for this batch and are
> untouched.

## Status: 24/24 assigned tasks complete (Phase 0: 6/6, Phase 1: 18/18)

Mode: **Strict TDD**. Test runner: `cd backend/agent && make test` (`go test -race -v ./...`).

## Commits (work units, in order)

1. `51d44678` — `feat(agent): AG-13 continuation seam — TurnOptions.Continuation, nil-default` (tasks 0.1–0.3)
2. `5260dd24` — `feat(agent): AG-13 scheduler sink-ownership seam — LeaveSinkOpen` (tasks 0.4–0.6)
3. `8db37082` — `feat(agent): AG-13 continuation identity/brackets — no run_start/run_end` (tasks 1.1–1.3, plus the mid-stream-fatal half of 1.2)
4. `49c6b679` — `feat(agent): AG-13 schedule-before-finalize reorder — continuation path` (tasks 1.4–1.6; bundled the History-append production code from 1.8/1.15 — see note below)
5. `ccffeac7` — `test(agent): AG-13 History-wiring evidence — S-HIS-090..093` (tasks 1.7, 1.10, 1.12, 1.14–1.16 test evidence, plus the `ai.NewRequest`/`RoleTool` proof)
6. `5a146a72` — `test(agent): AG-13 S-HIS-094 — nil continuation touches no transcript store` (tasks 1.17–1.18)

## TDD Cycle Evidence

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 0.1–0.3 | `TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission` (S-LSK-014) | Unit | ✅ full pkg green (baseline `e82c33e1`) | ✅ genuine — nil-pointer panic in `provider.Stream` (validation not wired; see RED Evidence below) | ✅ Passed | ✅ table-driven, 4 sub-cases (one per absent member) | ✅ none needed |
| 0.4–0.6 | `TestSchedule_LeaveSinkOpenZeroDefault_ClosesSinkUnchanged` / `...LeaveSinkOpenSet_CallerOwnsClose` (S-TLS-013/014) | Unit | ✅ | ✅ genuine — compile error `unknown field LeaveSinkOpen` | ✅ Passed | ✅ the two tests are each other's triangulation (default vs. set) | ✅ none needed |
| 1.1–1.3 | `TestTurn_ContinuationNonNil_NoRunBracketsSharedIdentityAndLane` (S-LSK-013 non-nil/text-only half) | Unit | ✅ | ✅ genuine — `run_end` present, want absent | ✅ Passed | ✅ two-turn scenario (shared identity/lane) + fresh-TurnID assertion | ✅ none needed |
| 1.2 (mid-stream-fatal clause) | `TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd` | Unit | ✅ | ✅ genuine, via scratch-revert (production code had been written ahead of this specific test — see Deviations) | ✅ Passed after restore | ➖ single scenario (not a charter scenario; evidence/coverage test) | ✅ none needed |
| 1.4–1.6 | `TestTurn_ContinuationToolCall_ScheduleBeforeFinalize_EventsInsideTurnBracket` (S-LSK-013 tool-calling half) | Unit | ✅ | ✅ genuine — tool events after `turn_end`; `CheckStream` rejected with `ErrMisplaced` | ✅ Passed; `CheckStream` accepts the framed stream | ➖ single scenario (the ordering property under test) | ✅ none needed |
| 1.7–1.9 | `TestTurn_ContinuationCommitsAssistantAndToolResults_OpenSetEmptyAtClose` (S-HIS-090) | Unit | ✅ | ✅ genuine, via scratch-revert (see Deviations) | ✅ Passed after restore | ➖ single scenario; triangulated further by 092's mixed-outcome test | ✅ none needed |
| 1.10–1.11 | `TestTurn_ContinuationEmptyContent_AppendsNothing` (S-HIS-091) | Unit | ✅ | ➖ not independently RED (see Deviations — confirmed via the same scratch-revert run: this test passes unchanged with or without the wiring, exactly as tasks.md's own "covered by 1.8" framing predicts) | ✅ Passed | ➖ single scenario | ✅ none needed |
| 1.12–1.13 | `TestTurn_ContinuationMixedOutcomes_OneResultPerCallInOrder` (S-HIS-092) | Unit | ✅ | ✅ genuine, via the same scratch-revert run | ✅ Passed after restore | ✅ 3 calls: success / ResultFailure / orphan-ExecutionFailure | ✅ none needed |
| 1.14–1.16 | `TestTurn_ContinuationAppendFailure_TypedErrorReturned` (S-HIS-093) | Unit | ✅ | ✅ genuine, via the same scratch-revert run | ✅ Passed after restore | ➖ single scenario | ✅ none needed |
| 1.17–1.18 | `TestTurn_ContinuationNil_HistorySurfaceGuardStaysGreen` (S-HIS-094) | Unit | ✅ | ➖ true by construction — the nil path never reaches `opts.Continuation` (already established by every prior nil-path guarantee); no separate scratch was meaningful here | ✅ Passed | ➖ single scenario | ✅ none needed |

### Test Summary
- **Total tests written**: 11 (`harness_test.go`, 939 lines)
- **Total tests passing**: 11/11, plus the full pre-existing suite (see Work Unit Evidence)
- **Layers used**: Unit (11)
- **Approval tests**: None — no refactoring tasks this batch
- **Pure functions created**: `validateContinuation`, `toolResultMessage` (both side-effect-free)

## RED Evidence (verbatim, selected)

**Task 0.1** (`go test -run TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission -race -v ./src/agent/...`):
```
--- FAIL: TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission/stamper_absent (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered, repanicked]
...
github.com/cachicamas/backend/agent/src/agent.Turn(...)
	.../backend/agent/src/agent/loop.go:304 +0x6f0
```
(`Turn` proceeded all the way to a nil `provider.Stream` call instead of rejecting the half-configured continuation early — proves validation was not yet wired.)

**Task 0.4** (`go test -run TestSchedule_LeaveSinkOpen -race -v ./src/agent/...`):
```
# github.com/cachicamas/backend/agent/src/agent_test [github.com/cachicamas/backend/agent/src/agent.test]
src/agent/harness_test.go:171:51: unknown field LeaveSinkOpen in struct literal of type agent.Scheduler
FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
```

**Task 1.1** (`go test -run TestTurn_ContinuationNonNil_NoRunBracketsSharedIdentityAndLane -race -v ./src/agent/...`):
```
harness_test.go:293: continuation-path event kind = run_end, want no run_start/run_end (the caller's run owns the bracket)
harness_test.go:300: last event kind = run_end, want turn_end (turn_end, no run_end)
--- FAIL: TestTurn_ContinuationNonNil_NoRunBracketsSharedIdentityAndLane (0.00s)
```

**Task 1.2 mid-stream-fatal clause** (scratch-revert run):
```
harness_test.go:396: continuation-path fatal event kind = run_end, want no run_start/run_end even on the mid-stream fatal path
harness_test.go:402: last event kind = run_end, want turn_end
--- FAIL: TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd (0.00s)
```

**Task 1.4** (`go test -run TestTurn_ContinuationToolCall_ScheduleBeforeFinalize_EventsInsideTurnBracket -race -v ./src/agent/...`):
```
harness_test.go:517: tool events at [start=2 end=3] land after turn_end at [1], want both BEFORE it (inside the open turn bracket)
harness_test.go:527: CheckStream rejected the continuation-path stream: event[3]: value is not permitted where it appears
--- FAIL: TestTurn_ContinuationToolCall_ScheduleBeforeFinalize_EventsInsideTurnBracket (0.00s)
```
(Confirms design's predicted failure mode exactly: `ErrMisplaced` on a `PlacementTurn` event after the terminal `run_end`.)

**Tasks 1.7/1.12/1.14 combined scratch-revert run** (History-append call sites temporarily removed from `finishContinuationTurn`, `go test -run 'TestTurn_ContinuationCommitsAssistantAndToolResults...|TestTurn_ContinuationEmptyContent...|TestTurn_ContinuationMixedOutcomes...|TestTurn_ContinuationAppendFailure...' -race -v ./src/agent/...`):
```
--- PASS: TestTurn_ContinuationEmptyContent_AppendsNothing (0.00s)
harness_test.go:893: Turn returned err = nil, want a non-nil typed rejection (the transcript store rejects the append)
harness_test.go:830: history has 0 entries, want 4 (assistant message + 3 tool-result messages)
harness_test.go:616: history has 0 entries, want 2 (assistant message + one tool-result message)
--- FAIL: TestTurn_ContinuationAppendFailure_TypedErrorReturned (0.00s)
--- FAIL: TestTurn_ContinuationMixedOutcomes_OneResultPerCallInOrder (0.00s)
--- FAIL: TestTurn_ContinuationCommitsAssistantAndToolResults_OpenSetEmptyAtClose (0.00s)
```
(S-HIS-091 passes either way — confirming tasks.md's own "covered by 1.8" framing for the skip-if-zero-content case is correct, not a gap.)

After each RED capture, the scratch was reverted and the corresponding GREEN was re-confirmed before moving on; `git diff` was checked to confirm each restore was byte-identical to the intended implementation (in two cases — the mid-stream-fatal clause and the History-append wiring — the restore came out byte-identical to what had already been committed, because that production code had been written together with an adjacent task in the same commit; see Deviations below).

## Work Unit Evidence

| Unit | Focused test command | Result | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| Phase 0 (seams) | `go test -run "TestTurn_ContinuationHalfConfigured|TestSchedule_LeaveSinkOpen" -race -v ./src/agent/...` | All 6 sub-tests PASS | N/A — no real provider/tool/socket/file/wall-clock; `agenttest` fakes only | `git revert` commits `51d44678`, `5260dd24` (or delete the `Continuation`/`LeaveSinkOpen` field diffs in `loop.go`/`tool.go`/`scheduler.go`) |
| Phase 1 (loop continuation path) | `go test -run "TestTurn_Continuation" -race -v ./src/agent/...` | All 9 sub-tests PASS | N/A | `git revert` commits `8db37082`, `49c6b679`, `ccffeac7`, `5a146a72` (or delete the continuation-mode branches in `loop.go` + `harness_test.go`) |
| Full package (both units together) | `cd backend/agent && go test -race ./src/agent/...` | `ok  	github.com/cachicamas/backend/agent/src/agent	1.87s` (varies run to run) | N/A | Same as above, combined |
| Full module (`make test`) | `cd backend/agent && make test` (`go test -race -v ./...`) | **PASS** — every package `ok` (`src/agent`, `src/agenttest` (+ `sweep`, `tracetest`), `src/ai` (+ subpackages), `src/handoff`); zero `FAIL` lines | N/A | N/A (verification only) |

## Critical Correctness Constraints — outcome

1. **Continuation gates strictly on `opts.Continuation != nil`; nil-default path byte-stable.** Confirmed. The three enumerated tests —
   `TestTurn_WalkingSkeleton_EmitsContractEventOrder`, `TestTurn_TwoSequentialTurnsShareNothing`, `TestTurn_ReasoningPassThroughByteExact` —
   pass and their **source files are byte-unchanged** (`git diff e82c33e1..HEAD -- backend/agent/src/agent/loop_test.go` shows only the
   substrate-filter-widening hunk, never touching these three test bodies; re-confirmed by running them explicitly after every commit in
   this batch, most recently green in the final `make test` run above).
2. **Schedule-before-finalize reorder.** Implemented, continuation-path only. Proven two ways: (a) direct event-index assertion (tool
   events land before `turn_end`); (b) `agent.CheckStream` accepts the framed stream (`run_start` + the continuation-path slice +
   `run_end`, all stamped through one shared `LaneStamper`, mirroring exactly what design.md's run algorithm step 2 says the future
   `Harness` will do) — RED showed `CheckStream` rejecting with `ErrMisplaced` before the reorder, GREEN shows it accepting after.
3. **`ai.NewRequest` accepts a `RoleTool`-result transcript — PROVEN, accepted.** `TestTurn_ContinuationCommitsAssistantAndToolResults_OpenSetEmptyAtClose`
   builds `transcript := history.Entries()` messages (assistant message with the tool call, then the `RoleTool` result message) and calls
   `ai.NewRequest("model-his-090", transcript)` directly; it returns a nil error. This is NOT a design-reopening finding — Layer 1's own
   `request.go:44` (`rolePermittedKinds[RoleTool] = {PartKindToolResult}`) and its `unresolvedToolResultRule` (resolves a result against
   any call anywhere in the request, not only an earlier one) already predicted this; the test is what turns that prediction into a run
   fact, per this batch's instruction to prove it rather than assert it from source alone.
4. **Substrate discipline.** No `R-LSK-004` file touched. `git diff e82c33e1..HEAD --stat` shows exactly: `harness_test.go` (new),
   `loop.go`, `loop_hook_test.go`, `loop_test.go`, `scheduler.go`, `tool.go` — six files, all either new or explicitly assigned to this
   batch. `go.mod`/`go.sum` untouched.
5. **No new exported method on `History`.** Confirmed — `history.go` and `history_surface_guard_test.go` are byte-unchanged
   (`git diff e82c33e1..HEAD -- backend/agent/src/agent/history.go backend/agent/src/agent/history_surface_guard_test.go` is empty). The
   continuation path uses only `Append`, `Entries`, `Len`, and `CloseTurn` (called by the *test*, standing in for the future run driver —
   `Turn` itself never calls `CloseTurn`, matching design).
6. **No new `EventKind`, no new dependency, no Layer 1 edit.** Confirmed — `run_events.go`, `turn_events.go`, and every file under
   `backend/agent/src/ai/` are byte-unchanged; `go.mod`/`go.sum` empty diff.
7. **No wall-clock sleeps, no sockets, no files in tests.** Confirmed — every new test uses `agenttest.NewProvider`/`Script`/`Emit`,
   buffered channels, and non-blocking `select`/`default` probes; the only `time` usage anywhere in the touched files is the
   pre-existing `drainSink`'s bounded `time.After(1s)` safety net (loop_test.go, unchanged) which every new test reuses rather than
   duplicating.
8. **Substrate filter widening, exact suffix, byte-in-sync.** `filterOutLoopFiles` (loop_test.go) and `filterOutLoopHookFiles`
   (loop_hook_test.go) both widened with exactly `/harness.go` and `/harness_test.go` — two suffixes, landed together in the same
   commit as `harness_test.go` itself (per task 0.1's explicit instruction), byte-in-sync between the two functions. The remaining
   three suffixes (`/harness_steering_test.go`, `/harness_pause_test.go`, `/harness_suspension_test.go`) are **not yet added** —
   correctly deferred to Phase 3/4/5, which are out of this batch's scope.
9. **Test naming and banners.** Every new test follows `Test<Subject>_<Behavior>_<Expectation>`; every banner cites the scenario id
   (S-LSK-013/014, S-TLS-013/014, S-HIS-090..094) or is explicitly marked as a non-charter coverage/evidence test.

## Files Changed

| File | Action | What Was Done |
|---|---|---|
| `backend/agent/src/agent/loop.go` | Modified | `TurnContinuation` struct + `validateContinuation`; `TurnOptions.Continuation` field; conditional identity minting/lane sharing; conditional `run_start`/`run_end` (nil-default vs. continuation) on both the normal-completion and mid-stream-fatal paths; `finishContinuationTurn` (schedule-before-finalize + History commits); `toolResultMessage`; `reconstructMessage` continuation-only `ai.ToolCall` parts; `turnAccumulator.continuation` field threaded through `newTurnAccumulator`/`finalize` |
| `backend/agent/src/agent/tool.go` | Modified | `Scheduler.LeaveSinkOpen bool` field (zero = AG-09 behavior) |
| `backend/agent/src/agent/scheduler.go` | Modified | `Schedule`'s close-third step conditional on `!LeaveSinkOpen`; doc update on the sink-ownership contract |
| `backend/agent/src/agent/harness_test.go` | Created | 11 tests across Phase 0 and Phase 1 (see TDD Cycle Evidence); `package agent_test` |
| `backend/agent/src/agent/loop_test.go` | Modified | `filterOutLoopFiles` widened with `/harness.go`, `/harness_test.go` |
| `backend/agent/src/agent/loop_hook_test.go` | Modified | `filterOutLoopHookFiles` widened identically, byte-in-sync |

No other file in `backend/agent/` differs from baseline `e82c33e1`.

## Deviations from Design / Tasks — noted honestly

1. **Two production edits landed slightly ahead of their dedicated RED test**, both later verified with a genuine scratch-and-revert RED
   cycle rather than left unproven:
   - The mid-stream-fatal `run_end` suppression (part of task 1.2's "on any path including the mid-stream fatal path" clause) was
     written in the same edit as the main `finalize()` gating, because both are the identical one-line guard pattern applied at two
     call sites in the same function. `TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd` was then written, the guard was
     scratch-removed, RED was captured (verbatim above), and the guard was restored — `git diff` after restore showed the file
     unchanged from the version already in place, confirming no unintended drift.
   - The `R-HIS-010` `History.Append` wiring (tasks 1.8/1.15) was written together with the schedule-before-finalize reorder (task 1.5)
     in commit `49c6b679`, because `finishContinuationTurn` is one coherent function and splitting the rejoin-capture from what happens
     to the rejoin felt like busywork rather than a meaningful seam. Tests for S-HIS-090/091/092/093 were then written in the next
     commit and run against the *already-passing* implementation; to get real RED (not fabricated), the two `History.Append` call sites
     were scratch-removed as one block and all four tests re-run together — 090/092/093 failed (verbatim above) and 091 passed
     unchanged, which is exactly what tasks.md's own "1.11 covered by 1.8" / "1.13 covered by 1.8" annotations predict. The block was
     then restored, and `git diff` showed `loop.go` byte-identical to the already-committed version — so this test commit (`ccffeac7`)
     touches only `harness_test.go`.
   Both deviations are process notes, not behavior deviations: every task's stated GREEN behavior is implemented exactly as specified,
   and every task's RED claim is backed by an actually-executed failing run, just not always in the same commit as the production
   line that made it pass.
2. **`toolResultMessage`'s content mapping for `ToolOutcomeExecutionFailure`** is an implementation choice the spec does not pin
   (design.md says only "`ai.NewToolFailure(callID, content)` for both failure outcomes"). `ToolOutcomeResultFailure` uses `r.Content`
   (the tool's own failure output, per `tool.go`'s `Result` doc). `ToolOutcomeExecutionFailure` has no `Content` (documented "zero for
   execution-failure"), so its typed `*Failure`'s redacted diagnostic (`r.FailureFor().Unwrap().Error()`) is used instead — still
   content the model can reason about, never a sentinel, and the outcome itself rides `Failed()` regardless of which content path was
   taken (proven by `TestTurn_ContinuationMixedOutcomes_OneResultPerCallInOrder`, which exercises exactly this branch via the orphan
   path). Recorded here as a conscious choice available for `sdd-verify`/`sdd-spec` to sign off on.
3. **`finishContinuationTurn`'s return values on the append-failure path (S-HIS-093)**: `Turn` returns `(msg, finish, err)` — the actual
   reconstructed message and finish reason from `finalize()`, plus the non-nil append error — rather than zeroing `msg`/`finish` (as the
   mid-stream-fatal path does, for a different reason: there, no valid finish was ever observed). The spec only requires "a non-nil
   typed error... the turn MUST NOT be reported successful"; it does not pin `msg`/`finish` on this path, and the harness (Phase 2,
   `R-RUN-011`) only branches on `err != nil`, so this is a low-stakes, information-preserving choice rather than a spec deviation.
4. **`TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd`** is not one of the named charter test IDs in tasks.md's task list for
   1.1–1.6, but it is required by R-LSK-001 point 2's explicit "on any path including the mid-stream fatal path" clause and by this
   batch's own critical-correctness constraint #1. Added as a non-charter coverage/evidence test, mirroring this file's own established
   precedent (`TestTurn_MidStreamErrorSurfacesOnReturn` et al. in `loop_test.go`).

None of these are design-reopening findings. No blockers were hit.

## Remaining Tasks (out of this batch's scope — Phases 2–6)

- [ ] Phase 2 (AG-13.1) — `Harness` type, `Run`/`Steer`, two-turn run to completion, run-scope reconstruction (+ bite), no-privileged-channel guard, failure path (`R-RUN-011`, where S-HIS-093's cross-check at 2.29–2.31 completes)
- [ ] Phase 3 (AG-13.2) — steering queue
- [ ] Phase 4 (AG-13.3) — pause resumption
- [ ] Phase 5 — permission-suspension acceptance clause + the `R-APP-002` parked-wait bite
- [ ] Phase 6 — substrate/spec-code twins (including the `ToolSource` re-home at `tool.go:239-240`, still reading "AG-13's widening" —
      **untouched by this batch, correctly**, since task 6.2 owns it), docs, coverage gate, final gates (`make lint`, `make build`,
      `make vuln-check`)

## `make test` Final Result

```
cd backend/agent && make test
```
**PASS.** Full module (`go test -race -v ./...`) green: `src/agent` (1.87s, not cached — genuinely re-run), `src/agenttest`,
`src/agenttest/sweep`, `src/agenttest/tracetest`, `src/ai`, `src/ai/internal/retry`, `src/ai/openaicompat` (+ subpackages),
`src/handoff` all report `ok`. Zero `FAIL` lines in the full log.
