# Tasks: AG-14 — Build the cancellation tree (`cachicamas-agent-cancellation-tree`)

> **Scenario count** (per `agent-cancellation-tree/spec.md`'s own requirements, MUST match `apply-progress.md`/`verify-report.md`): the new capability itself is **8 requirements → 11 scenario obligations** (`S-CAN-001`…`008`, of which `S-CAN-010`/`011`/`012` are bites). The five cross-cut deltas add **13 further scenarios** this change is responsible for closing: `agent-loop-skeleton` +2 (`S-LSK-018`, `S-LSK-019`), `agent-run-driver` +3 (`S-RUN-003`, `S-RUN-092`, `S-RUN-102`), `agent-permission-protocol` +2 (`S-APP-017`, `S-APP-018`), `agent-tool-scheduler` +4 (`S-TLS-016`, `S-TLS-017`, `S-TLS-018`, `S-TLS-019`), `agent-history` +2 (`S-HIS-097`, `S-HIS-098`). **Total new/changed evidence obligations: 24.** `design.md`'s three closed decisions are binding; this file does not re-derive them.

## Forwarded obligation 2 — `context.Background()` identity grep, RE-RUN against the current tree

`grep -rn "context\.Background()" backend/agent/src/agent` (executed for this file, see full output in the phase-preflight transcript) shows exactly one production occurrence feeding `tool.Run`: **`scheduler.go:462`** inside `executeCall`. Every other hit is one of two shapes, neither of which asserts the identity of `executeCall`'s own internal context: (a) a caller passing `context.Background()` *into* `Schedule`/`Turn` as the run's own `ctx` (`scheduler_test.go:536,557,1173`; `permission_protocol_test.go:160,1267,1434,1512,1599,1980`; `loop_tool_dispatch_test.go:114,186,266`), or (b) a direct `tool.Run` unit-test call bypassing the scheduler entirely (`tool_test.go:57,82,124`). None of these captures, compares, or asserts on the `ctx` value `tool.Run` receives from inside `executeCall` — confirmed by reading each call site, not merely counting matches. **Safe to replace `scheduler.go:462`'s `context.Background()` with `ctx` (Phase 1).**

## Substrate Filter Closure (authoritative — closes `R-LSK-004`/`R-CAN-008`)

`filterOutLoopFiles` (`loop_test.go:831`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907`) MUST widen with exactly these six exact-filename suffixes, byte-in-sync, no wildcard/prefix/directory pattern:

```
/run_events.go
/cancellation.go
/cancellation_interrupt_test.go
/cancellation_shutdown_test.go
/cancellation_winddown_test.go
/cancellation_events_test.go
```

`doc.go`, `doc_contract_guard_test.go`, `harness_test.go`, `loop.go`, `scheduler.go`, `tool.go` are **already** in both filters — no addition needed. `stream_check_test.go` stays absent from both (it MUST NOT be edited). Land each new-file suffix pair in the SAME commit as the file that first needs it (AG-11/AG-12/AG-13 discipline).

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1800–2900 (production Go 350–650: `cancellation.go` ~80–150, `harness.go` delta ~150–260, `loop.go` delta ~20–40, `scheduler.go` delta ~100–180, `tool.go` ~10, `run_events.go`/`doc.go`/`doc_contract_guard_test.go` ~20; test Go 750–1250: four new `cancellation_*_test.go` files ~600–1000, `harness_test.go`/`loop_test.go`/`loop_hook_test.go`/`scheduler_test.go` diffs ~150–250; SDD markdown (design+specs already on branch, plus doc 0003 back-annotations and this file) ~700–1000) |
| 400-line budget risk | High — exceeds even the raised 1000-line pre-authorized ceiling |
| Chained PRs recommended | No |
| Suggested split | single PR — AG-14 only |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

**The `size:exception` extension is already granted by the user for AG-14, up front — `sdd-apply` MUST NOT stop to ask.** `NFR-CAN-005` requires the PR description to state why the change exceeds the default budget; that sentence is the review-budget forecast above.

### Suggested Work Units

ONE PR (`size:exception` pre-authorized for AG-14). Runtime harness is N/A throughout — no real provider, no real tool, no socket/file/wall-clock sleep; `agenttest` scripts, `agenttest.Gate`, and channel reads only (Threat Matrix: N/A, `design.md`). The wind-down bound is the one legitimate clock use.

| Unit | Goal | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | Substrate release + vocabulary (Phase 0) | `go test -run "TestRunOutcomeShutdown" -race ./src/agent/...` | N/A | revert `run_events.go`/`doc.go`/`doc_contract_guard_test.go`/`cancellation.go` |
| 2 | Real ctx into `tool.Run` (Phase 1) | `go test -run "TestSchedule_ToolReceivesRunContext" -race ./src/agent/...` | N/A | revert `scheduler.go:462` |
| 3 | Interrupt core + wind-down mechanisms (Phases 2–3) | `go test -run "TestHarness_Interrupt_MidTurn" -race ./src/agent/...` | N/A | revert `harness.go`/`loop.go` cancellation additions |
| 4 | Serial reuse, suspension abort, no-op, shutdown (Phases 4–8) | `go test -run "TestHarness_(Interrupt|Shutdown)_" -race ./src/agent/...` | N/A | revert corresponding test files |
| 5 | Wind-down bound + detach select (Phases 9–10) | `go test -run "TestHarness_WindDown" -race -count=1 ./src/agent/...` | N/A | revert `scheduler.go` detach select, `tool.go` field |
| 6 | Guards, cross-cut deltas, docs, final gates (Phases 11–12) | `cd backend/agent && make test && make lint && make build && make vuln-check` | N/A | revert promotion/docs commits |

## Phase 0: Substrate release & vocabulary (`R-CAN-007`, `R-CAN-008`, `R-LSK-004` delta)

- [x] 0.1 Widen both substrate filters with the six exact suffixes listed above; confirm the two entry sets stay identical and byte-in-sync (`S-LSK-019`).
- [x] 0.2 RED — create `cancellation_events_test.go` (`package agent_test`): `TestRunOutcomeShutdown_VocabularyAndStreamAcceptance` (`S-CAN-007`). Expect FAIL: `agent.RunOutcomeShutdown` undefined.
- [x] 0.3 GREEN — `run_events.go`: insert `RunOutcomeShutdown` between `RunOutcomeFailed` (`:113`) and `runOutcomeLimit` (`:117`), keeping `Completed`/`Interrupted`/`Failed` at 1/2/3; add its `String()` case `"shutdown"`. `RunEnd.validate` and `NewRunEnd`'s signature untouched.
- [x] 0.4 GREEN confirm — `S-CAN-007`: `String()=="shutdown"`; `NewRunEnd(run, Shutdown, nil)` constructs; `NewRunEnd(run, Shutdown, failure)` rejected `ai.ErrMisplaced`; `CheckStream` accepts a full `run_end(shutdown)` stream; `stream_check.go` byte-unchanged. **Satisfies forwarded obligation 4: `NewRunEnd(run, RunOutcomeShutdown, nil)` proven to stream through `CheckStream` BEFORE any harness wiring (Phase 2+).**
- [x] 0.5 Same commit as 0.6 — `doc.go`: append the `L2C-08` row (grammar `^//\tL2C-\d\d\t`) with `R-CAN-008`'s normative text verbatim.
- [x] 0.6 Same commit as 0.5 — `doc_contract_guard_test.go`: append the matching `expectedLayer2ContractRows` entry. Confirm the doc-contract guard passes with `L2C-08` present in both, every pre-existing row byte-unchanged, none removed or reworded (`S-CAN-008`).
- [x] 0.7 Create `cancellation.go`: sentinels `ErrInterrupted`, `ErrShutdown`; `ErrPromptAfterShutdown` (wraps `ErrShutdown`); `type DetachedCallError struct{ Tool, CallID string }` with an `Error()` method; `const defaultWindDownBound = 100 * time.Millisecond`. Pure vocabulary — no behavior wired yet. Confirm ambient-authority and import-boundary guards pass with zero change (`time` is not forbidden). **Apply note**: `fmt` cannot be used (transitively imports `os`/`io/fs`, trips the network/filesystem import guard) — implemented via hand-rolled `Error()`/`Unwrap()` + `strconv.Quote`, matching scheduler.go's own existing idiom; see apply-progress.md.

## Phase 1: `tool.Run` receives the real context (Decision 2, `R-TLS-013`)

- [x] 1.1 RED — `scheduler_test.go`: `TestSchedule_ToolReceivesRunContext_EarlyReturnOnCancel` (`S-TLS-016`) — scripted tool returns a distinguishable typed error as soon as `ctx.Done()`; run cancelled while that call executes. Expect FAIL: tool observes `context.Background()`, never returns early, sibling slots unaffected assertion is vacuous.
- [x] 1.2 GREEN — `scheduler.go:462`: `tool.Run(context.Background(), …)` → `tool.Run(ctx, …)`. No signature change on `Schedule`/`executeCall`/`tool.Run`.
- [x] 1.3 GREEN confirm — `S-TLS-016`: the ordinal slot carries an execution-failure result attributable to the tool's own early return; the tool's recorded work-completed flag is false; sibling calls still occupy their own ordinal slots.
- [x] 1.4 Confirm `S-TLS-017` — every existing AG-09/AG-10 scheduler test and every direct `tool.Run` unit call in `tool_test.go` pass with source **file-unchanged**, per the grep evidence above: none asserts context identity.

## Phase 2: Interrupt core — signals, mid-turn interrupt, orphans (`R-CAN-001`, `R-CAN-002`, `R-CAN-003` groundwork, `R-RUN-011` carve-out, `R-HIS-007` back-annotation)

- [x] 2.1 RED — create `cancellation_interrupt_test.go` (`package agent_test`): `TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized` (`S-CAN-001`) — provider `Hold`-gated mid-stream (`agenttest.Gate`, `.Reached()`) plus a new cancellation-observing scripted tool (test-only fixture, sibling of `BlockingScriptedTool`) from a prior tool-calling turn; `Interrupt()` fired after `.Reached()`. Expect FAIL: compile error, `agent.Harness` has no `Interrupt` method. **Apply note**: implemented as two t.Run subtests — the scenario's Given ("provider stream AND tools in flight") cannot be literally simultaneous in one turn (Schedule only runs after that turn's own Completion); see apply-progress.md for the full architectural rationale.
- [x] 2.2 GREEN — `harness.go`: add `signalMu sync.Mutex` guarding `{cancelRun context.CancelCauseFunc; shutdown bool}` on `Harness`; `Interrupt()`/`Shutdown()` methods that only invoke `cancelRun` with the matching sentinel under the mutex (Shutdown also latches `shutdown = true`); `Run` entry derives `runCtx, cancel := context.WithCancelCause(ctx)` under the mutex and stores `cancelRun`; `Run` exit clears `cancelRun` under the mutex then `cancel(nil)` deferred. No loop/scheduler wiring yet.
- [x] 2.3 GREEN confirm (intermediate, expected still RED) — test 2.1 now compiles and runs but FAILS: the bare provider-stream close still normalizes to `ai.FinishReasonStop` and the run reports completed, not interrupted — this is the causality proof that `loop.go`'s cause check is the next required mechanism.
- [x] 2.4 **Own RED-first task (forwarded obligation 3a) — `loop.go` closed-channel cause check.** GREEN — `loop.go:417-427`, before the `turn.finish == 0` normalization: `if cause := context.Cause(ctx); errors.Is(cause, ErrInterrupted) || errors.Is(cause, ErrShutdown) { return a typed error carrying cause }` — `Turn`'s first-ever cancellation observation. **Apply note**: also emits `turn_end(TurnOutcomeAborted, failure)` before returning — CheckStream rejects a run-close while a turn is still open; see apply-progress.md.
- [x] 2.5 GREEN confirm (intermediate, expected still RED) — `Turn` now returns a typed cancellation error to `Run`, but `Run`'s existing `failRun` still routes every non-nil `Turn` error to `RunOutcomeFailed`/`Unavailable` — proving `windDownRun` is the remaining missing mechanism.
- [x] 2.6 **Own RED-first task (forwarded obligation 3b) — the harness `windDownRun` path.** GREEN — `harness.go`: new `windDownRun(sink, stamper, runID, hist, cause error) (ai.Message, ai.FinishReason, error)`: `hist.SynthesizeOrphans()`, `hist.CloseTurn()`, `NewRunEnd(runID, outcome, nil)` where `outcome` maps `ErrInterrupted→RunOutcomeInterrupted` / `ErrShutdown→RunOutcomeShutdown`, then return `(ai.Message{}, 0, cause)`. Wire into `Run`'s turn-error handling: if `errors.Is` matches a sentinel on `context.Cause(runCtx)`, call `windDownRun` instead of `failRun`. Add the iteration-boundary check at the top of `Run`'s `for` loop: consult `context.Cause(runCtx)` before starting another `Turn`; a pending signal winds down instead of calling `Turn` again.
- [x] 2.7 GREEN confirm — `S-CAN-001` passes in full: stream cancellation follows `R-CNF-011`/`R-CNF-012` (bare close, no synthesized terminal event), the cancellation-observing tool returns early rather than completing, orphan synthesis closes every open call with `synthesized`-origin entries, `run_end(interrupted)` carries no `*Failure`, `errors.Is(err, ErrInterrupted)`, `CheckStream` accepts the stream unmodified. **Jointly closes `S-RUN-102` (carve-out), `S-HIS-097` (synthesis's first production caller), and confirms `S-TLS-016` inside the integrated harness path.**
- [x] 2.8 Confirm `S-HIS-098`: `history.go` byte-unchanged, `history_surface_guard_test.go` source byte-unchanged and green, its enumerated exported route set equal in both directions.

## Phase 3: Bite `S-CAN-010` — loop.go cause check causality proof

- [x] 3.1 Scratch-delete the `loop.go` cause check added at 2.4; rerun `S-CAN-001`; confirm it FAILS reporting a normally completed turn (bare stream close normalized to `FinishReasonStop`, interrupt recorded as success). Record the failure output in `apply-progress.md`. Revert; confirm `git diff -- loop.go` is byte-empty.

## Phase 4: Same-harness serial reuse (`R-CAN-002` reuse clause, `R-RUN-001` delta)

- [x] 4.1 RED — `cancellation_interrupt_test.go`: `TestHarness_Interrupt_SameHarnessRunsNextPrompt` (`S-CAN-002`, jointly `S-RUN-003`) — after `S-CAN-001`'s wind-down, drive a second `Run` on the same harness value against a fresh script. Expect FAIL: `Run`'s post-shutdown-style refusal or a closed steering queue rejects the second run (queue never reopens).
- [x] 4.2 GREEN — `harness.go` `Run` entry (same critical section as 2.2's cancel-cause derivation): reopen the steering queue — clear `queue.closed` under `queue.mu` — before deriving the new run context, unless the terminal shutdown flag is set (Phase 7 wires that half).
- [x] 4.3 GREEN confirm — `S-CAN-002`/`S-RUN-003`: second run emits its own complete run bracket, ends `RunOutcomeCompleted`, returns nil error, accepts a `Steer` call with nil return during the second run and the message reaches its transcript before the next provider call, `CheckStream` accepts the second run's stream unmodified.

## Phase 5: Interrupt/shutdown during permission suspension (`R-CAN-003`, `R-APP-009` restated)

- [ ] 5.1 RED — `cancellation_interrupt_test.go`: `TestHarness_Interrupt_DuringSuspensionAbortsTyped` (`S-CAN-003`) — deferring permission policy; consumer reads `decision_required` off the stream (the stream is the sync); `Interrupt()` fires. Expect FAIL: parked abort still derives from `ctx.Err()`, category `Unavailable`, not `errors.Is`-matchable against `ErrInterrupted`.
- [ ] 5.2 GREEN — `scheduler.go:639` and `:665`: replace `typedExecutionFailureFromError(call.ID(), ctx.Err())` with a new `typedCancellationFailureFromError(call.ID(), context.Cause(ctx))` (sibling of `typedFailureFromError`, `scheduler.go:938`) that builds `ai.FailureCategoryCancellation` when the cause matches a sentinel, else keeps the existing `Unavailable` wrap. `parked.remove` and the rejoin-population/goroutine-baseline guarantees stay byte-unchanged.
- [ ] 5.3 GREEN confirm — `S-CAN-003`: the parked call's ordinal slot carries `ExecutionFailure` whose `*Failure` reports `Cancellation` and `errors.Is`-matches `ErrInterrupted`; a subsequent wake for that call returns `ErrStrayDecision`; transcript reads back with no open call after wind-down; `CheckStream` accepts unmodified.
- [ ] 5.4 RED — `cancellation_shutdown_test.go`: `TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal` (`S-APP-017` shutdown half, jointly with 5.3's interrupt half) — same parked-call shape, `Shutdown()` fires instead. Expect FAIL until 5.2 lands (already green by this point, so this is a composition-proof GREEN-on-first-run — confirm genuine causality via the same category/sentinel assertions, not vacuously).
- [ ] 5.5 GREEN confirm — `S-APP-017`: the two assertions swap — category `Cancellation`, `errors.Is` matches `ErrShutdown` and fails `ErrInterrupted` — proving the failure carries the firing signal's identity, not a category constant.

## Phase 6: A second signal is a no-op (`R-CAN-004`)

- [ ] 6.1 RED — `cancellation_interrupt_test.go`: `TestHarness_Interrupt_SecondInterruptIsNoOp` (`S-CAN-004`) — a second `Interrupt()` fired from a separate goroutine concurrent with the first's wind-down, under `-race`.
- [ ] 6.2 GREEN confirm — no panic, no data race, run-end outcome and returned error identical to `S-CAN-001`'s, emitted sequence accepted by `CheckStream` unmodified. Composition proof via 2.2's `signalMu`-guarded state (Go's documented no-op on an already-cancelled context) — confirm genuinely, not assumed, by running under `-race -count=10`.

## Phase 7: Shutdown winds down and terminally refuses (`R-CAN-005`, `R-CAN-007` consumer)

- [ ] 7.1 RED — create `cancellation_shutdown_test.go`: `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts` (`S-CAN-005`) — same shape as `S-CAN-001`; `Shutdown()` fires; then a second `Run` is invoked on the same value. Expect FAIL: `RunOutcomeShutdown` not yet reachable from `Shutdown()` (routes as interrupted, since 2.6's outcome-mapping already exists) and the second `Run` is accepted rather than refused.
- [ ] 7.2 GREEN — `harness.go` `Run` entry: under `signalMu`, if `shutdown` is set, return `ErrPromptAfterShutdown` immediately, emit nothing, close the sink (leave the queue reopen from Phase 4 conditional on `!shutdown`).
- [ ] 7.3 GREEN confirm — `S-CAN-005`: wind-down produces the same synthesized-origin closures and nil-failure run-close shape as `S-CAN-001`; `run_end` outcome is `RunOutcomeShutdown`, distinct from both interrupted and failed; returned error `errors.Is`-matches `ErrShutdown` and fails `ErrInterrupted`; the second `Run` returns `ErrPromptAfterShutdown` (`errors.Is` → `ErrShutdown`), consumer observes no event, first run's stream still accepted by `CheckStream` unmodified.
- [ ] 7.4 Confirm the two signals stay distinguishable through both nouns of `0003:1379` — the run-end outcome on the stream (7.3) and the Go error chain (7.3) — so a stream-only consumer can tell them apart without a Go error.

## Phase 8: Bite `S-CAN-011` — cancellation carve-out causality proof

- [ ] 8.1 Scratch-revert the harness's cause-aware routing added at 2.6 (route every non-nil `Turn` error to `failRun` unconditionally); rerun `S-CAN-001` and `S-CAN-005`; confirm BOTH FAIL reporting `RunOutcomeFailed` with a non-nil `Unavailable` failure. Record output in `apply-progress.md`. Revert; confirm `git diff -- harness.go` is byte-empty.

## Phase 9: Wind-down bound — detach select, `Scheduler.WindDownBound`, typed report (`R-CAN-006`, `R-TLS-014`)

- [ ] 9.1 `tool.go`: add `Scheduler.WindDownBound time.Duration` (zero-default field, `LeaveSinkOpen` precedent) to the `Scheduler` struct.
- [ ] 9.2 RED — create `cancellation_winddown_test.go` (`package agent_test`): `TestHarness_WindDown_DeafToolCannotHoldRunHostage` (`S-CAN-006`, Thens 1–2) — `BlockingScriptedTool` (`scripted_tool_test.go:74-83`, already context-deaf) with a never-closed `release`; caller-owned `Scheduler` injected with a small `WindDownBound`; `Interrupt()` fires. Expect FAIL: compile error, `WindDownBound` unread; the join hangs (would need `-timeout` to observe).
- [ ] 9.3 **Own RED-first task (forwarded obligation 3c) — `executeCall`'s detach select and armed bound.** GREEN — `scheduler.go` `executeCall`: run `tool.Run(ctx, runArgs, PolicySlot(call.ID()))` in an inner goroutine sending `(Result, error)` (recovering any panic into the buffered channel) on a **buffered capacity-1** channel `resCh`; the call goroutine selects `case reply := <-resCh` (uncancelled path, only arm that can fire — no timer created) vs `case <-ctx.Done()`, which arms a `time.NewTimer(s.WindDownBound)` (resolving `0` to `defaultWindDownBound`) and selects again between `resCh` and the timer.
- [ ] 9.4 GREEN — on timer overrun: write `Result{Outcome: ToolOutcomeExecutionFailure, Failure: typed}` to the ordinal slot via `typedCancellationFailureFromError`-shaped construction with cause `&DetachedCallError{Tool: call.Name(), CallID: call.ID()}`; emit the existing `tool_end_execution_failure` via `emitExecutionFailure` — no new `EventKind`, no new `Result` outcome. `wg.Wait()`, the parked-set clear, the emissions close, the dispatcher join and the ordered rejoin stay byte-unchanged in order and behavior.
- [ ] 9.5 GREEN confirm — `S-CAN-006` Thens 1–2: the run **returns**, observed by a read on the run's completion channel, never a wall-clock assertion; the stream carries `tool_end_execution_failure` for that call whose `*Failure` `errors.As`-extracts `*DetachedCallError` naming that tool and that call identity.
- [ ] 9.6 Confirm `S-TLS-018`: a `Scheduler` at the zero-value bound with an uncancelled `Schedule` whose tool blocks until an explicit release — `Schedule` returns with a fully populated rejoin after release, no bound-derived failure in any slot, no timer created (assert via a deterministic proxy — e.g. no `DetachedCallError` present and completion latency tracks the release, not the bound).
- [ ] 9.7 RED — `scheduler_test.go`: `TestSchedule_MixedCancellationBatch_DisjointResultsPerCall` (`S-TLS-019`) — a batch mixing the cancellation-observing tool (Phase 1's fixture), the cancellation-deaf `BlockingScriptedTool`, and an ordinary succeeding tool; run cancelled mid-batch.
- [ ] 9.8 GREEN confirm — `S-TLS-019`/`R-TLS-010` restated: every ordinal slot populated; no slot carries both a tool-supplied `Result` and a scheduler-supplied `*Failure`; cancellation-caused slots report `Cancellation`, any ordinary execution error still reports `Execution`; the succeeding sibling's result is unaffected — disjointness confirmed under cancellation.

## Phase 10: Goroutine-leak proof, serial-only (`R-CAN-006` Then 3, part of `S-CAN-006`)

- [ ] 10.1 RED-then-compose — `cancellation_winddown_test.go`: `TestHarness_WindDown_NoHarnessGoroutineRemains`, **no `t.Parallel()`** (`stream_kit_leak.go:80,110` enforces this mechanically via `tb.Setenv`). Body: `agenttest.RequireNoGoroutineLeak(t, scenario)` where `scenario` runs the deaf-tool interrupt run from 9.2 and, **after the run returns**, closes that iteration's own `release` channel — so the detached goroutine is **accounted for** (proven alive past wind-down by 9.5's typed report, proven to exit once third-party code returns) rather than merely excluded from the count. A plain exclusion (never releasing, or releasing before the run returns) fails `R-CAN-006` — this is the discipline the design and spec both flag.
- [ ] 10.2 GREEN confirm — no harness-owned goroutine (`runDispatcher`, the per-turn forwarder, every per-call goroutine, `Run`'s own control flow) survives the wind-down across the repeated scenario; small injected `WindDownBound` (20–50ms) keeps the 50 repeats fast.

## Phase 11: Bite `S-CAN-012` — armed-bound causality proof

- [ ] 11.1 Scratch-remove the armed bound added at 9.3 (revert to an unconditional wait on `resCh`); rerun `S-CAN-006`; confirm it does NOT return — evidenced by `go test -timeout <bounded>` reporting the hung test, not by any passing assertion. Record the timeout output in `apply-progress.md`. Revert; confirm `git diff -- scheduler.go` is byte-empty.

## Phase 12: Guards & cross-cutting delta scenarios

- [x] 12.1 **`harness_test.go` method-count widen (`S-RUN-001` delta, design-flagged conscious edit).** Update the `"exactly two exported methods"` subtest (`:1018-1024`) to assert the sorted four-name set `{Interrupt, Run, Shutdown, Steer}`; rename the subtest so it no longer claims a count. Confirm no fifth exported method exists. **Applied early, in batch A/Phase 2**, not deferred to Phase 12: leaving it broken after Phase 2 landed `Interrupt`/`Shutdown` would leave `make test` red for the rest of this milestone. No other Phase 12 item (guards, docs, final gates) was touched.
- [ ] 12.2 Confirm `S-RUN-092` (AG-14 adds no third release path): `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` (`permission_protocol_test.go:1573-1600`) passes with source **byte-unchanged**; a run whose policy defers, driven with neither a wake nor a signal, does not terminate on its own within any bound (the bound stays unarmed — nothing cancelled the run).
- [ ] 12.3 Confirm `S-APP-018`: `permission_protocol_test.go` passes file-unchanged in full, and every AG-09/AG-10 scheduler test passes file-unchanged.
- [ ] 12.4 Confirm `S-LSK-018`: merge-base `git diff` over `backend/agent/src/agent/` shows the set of pre-existing non-test files that differ is **exactly** `{run_events.go, doc.go, harness.go, loop.go, scheduler.go, tool.go}` — treat as a hard pin, check it explicitly before hand-off; `run_events.go`'s diff adds only the outcome member and its `String()` case; `turn_events.go`, `failure.go`, `stream_check.go`, `stream_check_test.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `reconstruction_test.go`, `history.go`, every file under `src/ai/` byte-unchanged; `go.mod`/`go.sum` diff empty; every-kind-constructible guard passes at its committed kind count (AG-14 registers zero).
- [ ] 12.5 Confirm `S-LSK-019` (already begun at 0.1): both filters' entry sets are identical to each other, each of the six new suffixes present as an exact filename suffix in both, no wildcard/prefix/directory pattern, `stream_check_test.go` absent from both, and an unnamed new file placed in the package fails both guards.
- [ ] 12.6 Confirm import-boundary and ambient-authority guards pass with **zero** change over both the production and test closures; `time` remains the only new stdlib surface, used solely for the bound.

## Phase 13: Final gates

- [ ] 13.1 **Coverage gate.** `loop.go` line coverage ≥ 80% under `make test`, including the new cancellation branch (`NFR-CAN-004`); extend tests if any branch is unexercised.
- [ ] 13.2 **Docs.** `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md`: tick `:2171` — "Interrupt and shutdown are distinguishable end to end; wind-down is bounded — closed by AG-14."; verify `:2207`'s R-09 row and `:2257`'s traceability row already name AG-14 correctly (back-annotate only if a discrepancy is found — do not rewrite a correct row); bump the status paragraph at `:3` from "13 of 24" to "14 of 24" with a sentence describing AG-14's closure, mirroring AG-13's own sentence.
- [ ] 13.3 Confirm the five spec deltas' cited lines land verbatim in the promoted specs (already drafted under `specs/`): `agent-loop-skeleton` `R-LSK-004`; `agent-run-driver` `R-RUN-001`/`R-RUN-010`/`R-RUN-011`; `agent-permission-protocol` `R-APP-009`; `agent-tool-scheduler` `R-TLS-010`/`R-TLS-013`/`R-TLS-014`; `agent-history` `R-HIS-007`.
- [ ] 13.4 **Final gates.** `cd backend/agent && make test` green under `-race` — all 11 `agent-cancellation-tree` scenarios (incl. 3 bites) + 13 cross-cut delta scenarios recorded; `golangci-lint cache clean && make lint` clean; `make build` clean; `make vuln-check` clean (explicit — not in `make all`).
- [ ] 13.5 Note for `sdd-archive` (not this phase): promote `agent-cancellation-tree/spec.md` to `openspec/specs/agent-cancellation-tree/spec.md`; apply the five deltas into their canonical specs; archive the change folder after `sdd-verify` passes, per AG-09..AG-13 precedent.

## Coverage Table

| Requirement | Scenario(s) | Task(s) |
|---|---|---|
| R-CAN-001 | S-CAN-001 (vocabulary+propagation half) | 2.1–2.7 |
| R-CAN-002 | S-CAN-001, S-CAN-002, S-CAN-011 (bite) | 2.1–2.7, 4.1–4.3, 8.1 |
| R-CAN-003 | S-CAN-003 | 5.1–5.3 |
| R-CAN-004 | S-CAN-004 | 6.1–6.2 |
| R-CAN-005 | S-CAN-005 | 7.1–7.4 |
| R-CAN-006 | S-CAN-006, S-CAN-012 (bite) | 9.1–9.6, 10.1–10.2, 11.1 |
| R-CAN-007 | S-CAN-007 | 0.2–0.4 |
| R-CAN-008 | S-CAN-008 | 0.5–0.6 |
| S-CAN-010 (bite) | loop.go cause-check causality | 3.1 |
| R-LSK-004 delta | S-LSK-018, S-LSK-019 | 0.1, 12.4–12.5 |
| R-RUN-001 delta | S-RUN-003 | 4.1–4.3, 12.1 |
| R-RUN-010 delta | S-RUN-092 | 12.2 |
| R-RUN-011 delta | S-RUN-102 | 2.7 |
| R-APP-009 delta | S-APP-017, S-APP-018 | 5.4–5.5, 12.3 |
| R-TLS-010 delta | S-TLS-019 | 9.7–9.8 |
| R-TLS-013 | S-TLS-016, S-TLS-017 | 1.1–1.4 |
| R-TLS-014 | S-TLS-018 | 9.6 |
| R-HIS-007 delta | S-HIS-097, S-HIS-098 | 2.7–2.8 |

## Constraints restated (binding on every task above)

No `R-LSK-004` substrate edit beyond the recorded AG-14 release (`run_events.go`: exactly one member + `String()` case; `doc.go`/`doc_contract_guard_test.go`: exactly the `L2C-08` row + its expectation entry, same PR). `turn_events.go`, `failure.go`, `stream_check.go`, `stream_check_test.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `reconstruction_test.go`, `history.go`, `go.mod`, `go.sum` stay byte-unchanged. No new `EventKind`, no new `TurnOutcome`, no new exported `History` method. No retry, no compaction, no cancellation vocabulary beyond the two named sentinels. No Layer 1 edit (`backend/agent/src/ai/**` untouched). No new Go dependency. No real provider, tool, socket, file, or wall-clock sleep — synchronize with `agenttest.Gate` and channel reads only; the wind-down bound is the one legitimate clock use. `permission_protocol_test.go` and every AG-09/AG-10 scheduler test pass file-unchanged. `TestHarness_WindDown_NoHarnessGoroutineRemains` never calls `t.Parallel()`.

## Risks

- **Staged-RED sequencing (Phase 2) depends on order**: tasks 2.2→2.4→2.6 each introduce one mechanism and each has its own *intermediate* RED confirmation (2.3, 2.5) before the next mechanism lands — apply MUST NOT collapse these into one uninterrupted GREEN, or the forwarded obligation to pin each mechanism as its own RED-first task is not actually satisfied, only asserted.
- **Bite ordering**: `S-CAN-010` (Phase 3) needs only `S-CAN-001`; `S-CAN-011` (Phase 8) needs BOTH `S-CAN-001` and `S-CAN-005`, so it cannot run before Phase 7. `S-CAN-012` (Phase 11) needs `S-CAN-006`, so it cannot run before Phase 9. Do not conflate these scratch-and-revert bites with a permanently-mutated-test-input bite (there are none of that kind in this change).
- **The leak test's accounting discipline (10.1) is easy to get wrong by exclusion** — releasing the blocked tool BEFORE the run returns, or never releasing it, both silently break `R-CAN-006`'s "accounted for, not merely excluded" requirement without failing the raw goroutine-count assertion. The task explicitly orders release-after-return.
- **`S-APP-017` needs both signals in evidence** — a single-signal parked-abort test proves category `Cancellation` but not that the abort *names* the firing signal; Phase 5 deliberately pairs an interrupt test (5.1–5.3) with a shutdown test (5.4–5.5) against the same parked-call shape.
- **Filter-widening landmine** — every RED test added under `backend/agent/src/agent/` trips both substrate guards until its filter suffix lands in the SAME commit as the file, repeating the AG-11/AG-12/AG-13 discipline six times for this change's new files.
