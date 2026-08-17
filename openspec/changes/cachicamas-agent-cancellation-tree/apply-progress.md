# Apply Progress: AG-14 — Build the cancellation tree (`cachicamas-agent-cancellation-tree`)

> This file is extended by later batches, not overwritten. Each batch appends its own dated section below, preserving every prior batch's evidence unchanged.

## Batch A — Phases 0–4 (23 tasks: 0.1–0.7, 1.1–1.4, 2.1–2.8, 3.1, 4.1–4.3)

**Status**: All 23 assigned tasks complete. Full `./src/agent/...` suite green (`go test -race`) at every phase checkpoint. One task from Phase 12 (`12.1`) was also applied — see "Deliberate out-of-scope exception" below.

**Mode**: Strict TDD. Every behavior RED-recorded (real command run, real failure captured) before its GREEN. Staged-RED sequencing honored exactly as `tasks.md`'s own Risks section requires (2.2→2.4→2.6 each introduced as its own mechanism, with intermediate RED confirmations at 2.3/2.5 before the next mechanism landed).

### Files changed

| File | Action | What |
|---|---|---|
| `backend/agent/src/agent/loop_test.go` | Modify | `filterOutLoopFiles` widened +6 suffixes (0.1) |
| `backend/agent/src/agent/loop_hook_test.go` | Modify | `filterOutLoopHookFiles` widened +6 suffixes, byte-in-sync with the above (0.1) |
| `backend/agent/src/agent/run_events.go` | Modify — released substrate | `RunOutcomeShutdown` + `String()` case only (0.3) |
| `backend/agent/src/agent/doc.go` | Modify — released substrate | `L2C-08` row (0.5) |
| `backend/agent/src/agent/doc_contract_guard_test.go` | Modify — released substrate | Matching `expectedLayer2ContractRows` entry (0.6) |
| `backend/agent/src/agent/cancellation.go` | Create | Sentinels, `ErrPromptAfterShutdown`, `DetachedCallError`, `defaultWindDownBound` (0.7) |
| `backend/agent/src/agent/cancellation_events_test.go` | Create | `TestRunOutcomeShutdown_VocabularyAndStreamAcceptance` (S-CAN-007) + `TestCancellationVocabulary_SentinelsDistinctRefusalWrapsCarrierNames` (R-CAN-001 vocabulary proof) |
| `backend/agent/src/agent/scripted_tool_test.go` | Modify | `CancellationObservingScriptedTool` fixture + `ErrScriptedToolCancelled` sentinel |
| `backend/agent/src/agent/scheduler_test.go` | Modify | `TestSchedule_ToolReceivesRunContext_EarlyReturnOnCancel` (S-TLS-016) |
| `backend/agent/src/agent/scheduler.go` | Modify | `:462` `tool.Run(context.Background(), …)` → `tool.Run(ctx, …)` (1.2) |
| `backend/agent/src/agent/harness.go` | Modify | `signalMu`, `cancelRun`, `shutdown` fields; `Interrupt()`/`Shutdown()`; `Run` derives+uses `runCtx`; `windDownRun`; iteration-boundary check; queue `reopen()` wired at `Run` entry |
| `backend/agent/src/agent/loop.go` | Modify | Closed-channel cause check at `:417-427` (task 2.4's own location), + `turn_end(Aborted)` emission + `cancellationTurnFailure` helper |
| `backend/agent/src/agent/cancellation_interrupt_test.go` | Create | `TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized` (S-CAN-001, two subtests) + `TestHarness_Interrupt_SameHarnessRunsNextPrompt` (S-CAN-002/S-RUN-003) |
| `backend/agent/src/agent/harness_test.go` | Modify | "exactly two exported methods" → four-name set (task 12.1, applied early — see below) |

**Forbidden-list compliance** (verified via `git diff --stat` per file, and `go test -run TestTurn_SubstrateUntouched\|TestTurn_PreRequestHook_SubstrateUntouched`, both PASS): `turn_events.go`, `failure.go`, `stream_check.go`, `stream_check_test.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `reconstruction_test.go`, `history.go`, `history_surface_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `sequence.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `tool_test.go`, `permission_protocol_test.go`, `go.mod`, `go.sum` — all confirmed byte-unchanged (zero diff).

### TDD Cycle Evidence

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 0.2–0.4 | `TestRunOutcomeShutdown_VocabularyAndStreamAcceptance` | Unit | ✅ 225/225 baseline | ✅ Written, real compile-fail captured | ✅ Passed | ✅ 4 subtests (String, construct-ok, construct-reject, CheckStream) | ➖ None needed |
| 0.7 | `TestCancellationVocabulary_SentinelsDistinctRefusalWrapsCarrierNames` | Unit | ✅ (new file) | ✅ Written, real compile-fail captured | ✅ Passed | ✅ 4 subtests | ✅ Removed `fmt`, hand-rolled Error()/Unwrap() |
| 1.1–1.3 | `TestSchedule_ToolReceivesRunContext_EarlyReturnOnCancel` | Integration (Scheduler) | ✅ 225/225 | ✅ Written, real 8s timeout+goroutine-stack captured | ✅ Passed | ➖ Single scenario (S-TLS-016 has one Then) | ➖ None needed |
| 2.1–2.7 | `TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized` | Integration (Harness) | ✅ 225/225 | ✅ Written, real compile-fail + 2 staged intermediate REDs captured | ✅ Passed, stable ×10 under `-race` | ✅ 2 subtests, independent mechanisms | ➖ None needed |
| 4.1–4.3 | `TestHarness_Interrupt_SameHarnessRunsNextPrompt` | Integration (Harness) | ✅ full suite green | ✅ Written, real failure captured | ✅ Passed, stable ×10 under `-race` | ➖ Single scenario | ➖ None needed |

### Test Summary
- **Total tests written**: 5 top-level test functions (2 with subtests: 4 + 2), covering S-CAN-007, R-CAN-001 vocabulary, S-TLS-016, S-CAN-001 (2 halves), S-CAN-002/S-RUN-003.
- **Total tests passing**: full `./src/agent/...` suite green at every checkpoint (baseline 225 pre-AG-14; not re-counted exactly post-AG-14, but zero `--- FAIL` at any checkpoint including the final one).
- **Layers used**: Unit (2), Integration via `agenttest`-scripted `Harness`/`Scheduler` (3), E2E (0 — not applicable to this package).
- **Bites (RED-record → revert)**: 1 (task 3.1 / S-CAN-010).
- **Pure functions created**: `cancellationTurnFailure` (loop.go, wraps a cause as a typed Failure — deterministic, no side effects).

### RED/bite evidence (verbatim, captured)

**Task 0.2 RED** (`go test -run TestRunOutcomeShutdown_VocabularyAndStreamAcceptance`):
```
src/agent/cancellation_events_test.go:33:19: undefined: agent.RunOutcomeShutdown
... (5 occurrences)
FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
```

**Task 0.7 RED** (`go test -run TestCancellationVocabulary_SentinelsDistinctRefusalWrapsCarrierNames`):
```
src/agent/cancellation_events_test.go:119:23: undefined: agent.ErrInterrupted
... (10 occurrences: ErrInterrupted, ErrShutdown, ErrPromptAfterShutdown)
FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
```

**Task 1.1 RED** (`go test -race -timeout 8s -run TestSchedule_ToolReceivesRunContext_EarlyReturnOnCancel`) — a real, bounded hang, not a build failure:
```
panic: test timed out after 8s
...
goroutine 34 [select]:
github.com/cachicamas/backend/agent/src/agent_test.TestSchedule_ToolReceivesRunContext_EarlyReturnOnCancel.CancellationObservingScriptedTool.func3(...)
	.../scripted_tool_test.go:116 +0xf4
github.com/cachicamas/backend/agent/src/agent_test.(*ScriptedTool).Run(...)
	.../scripted_tool_test.go:158 +0x424
github.com/cachicamas/backend/agent/src/agent.(*Scheduler).executeCall(...)
	.../scheduler.go:462 +0x6f4
```
The stack trace itself names `scheduler.go:462` (`context.Background()`) as the block point — direct, captured proof.

**Task 2.1 RED** (`go test -run TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized`):
```
src/agent/cancellation_interrupt_test.go:73:5: h.Interrupt undefined (type agent.Harness has no field or method Interrupt)
src/agent/cancellation_interrupt_test.go:160:5: h.Interrupt undefined (type agent.Harness has no field or method Interrupt)
FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
```

**Task 2.3 intermediate RED** (after 2.2 landed — compiles, runs, still fails):
```
subtest "provider cancelled mid-stream...":
    cancellation_interrupt_test.go:82: Run error = messages[0].content[0].result: required value is empty, want errors.Is(err, agent.ErrInterrupted)
    cancellation_interrupt_test.go:97: run_end outcome = failed, want RunOutcomeInterrupted
    cancellation_interrupt_test.go:112: history has 2 entries, want 3

subtest "in-flight tool call observes cancellation...":
    cancellation_interrupt_test.go:169: Run error = provider failure: cancellation, want errors.Is(err, agent.ErrInterrupted)
    cancellation_interrupt_test.go:184: run_end outcome = failed, want RunOutcomeInterrupted
```
**Root-cause note** (recorded so the divergence from `tasks.md`'s own prose is not silent): the task's prose predicts "reports completed, not interrupted." My subtest 1's seeded open call makes `hist.CloseTurn()` reject instead (`messages[0].content[0].result: required value is empty`, traced to `history.go`'s `commitCloseTurnOp`), reported as **failed**, not **completed** — the underlying causality (the bare provider-stream close is never observed as a cancellation at all) is identical; only the *symptom* differs because this test's own construction seeds a pre-existing open call (see "Design decisions" below for why). Subtest 2's failure mode ("provider failure: cancellation") is a *futile second provider call* hitting the fake's own pre-stream cancellation check — this is the causality proof that the **iteration-boundary check** (2.6), not the closed-channel check (2.4) alone, is what subtest 2 needs; confirmed by task 2.5's evidence below, which shows 2.4 alone leaves subtest 2 unchanged.

**Task 2.5 intermediate RED** (after 2.4 landed):
```
subtest "provider cancelled mid-stream...":
    cancellation_interrupt_test.go:97: run_end outcome = failed, want RunOutcomeInterrupted
    (errors.Is assertion at :82 now PASSES — Turn's cause check fired)
    cancellation_interrupt_test.go:107: agent.CheckStream(...) = event[2]: value is not well-formed for its documented encoding

subtest "in-flight tool call observes cancellation...": UNCHANGED from 2.3 — "provider failure: cancellation" still appears (confirms 2.4 alone does not touch this subtest's own mechanism, exactly as architecturally predicted).
```
**Second discovery, resolved within 2.4** (not a separate task, folded into the same GREEN): `CheckStream` rejected `event[2]` because `windDownRun`'s `run_end` closed the run bracket while the turn bracket was still open (no `turn_end` had ever been emitted on the new cause-check path). Fixed by emitting `turn_end(TurnOutcomeAborted, failure)` in the same `loop.go` branch, mirroring the pre-existing `turn.fatal != nil` branch's own precedent exactly. `failure` is built via a new small helper `cancellationTurnFailure` using `ai.FailureCategoryCancellation` (confirmed pre-existing in Layer 1 via `grep` in `provider_failure.go` before use — not a new vocabulary member). This does not violate R-CAN-002's "no `*Failure`" rule: that rule governs the **run's** `run_end` (still nil failure, unchanged), not the **turn's** own `turn_end`.

**Task 2.7 GREEN**: both subtests pass; stable under `go test -race -count=10`.

**Task 3.1 bite (S-CAN-010) — scratch-delete evidence**: with `loop.go`'s cause-check block (and its now-dead `"errors"` import) scratch-removed:
```
subtest "provider cancelled mid-stream...": FAIL
    cancellation_interrupt_test.go:82: Run error = messages[0].content[0].result: required value is empty, want errors.Is(err, agent.ErrInterrupted)
    cancellation_interrupt_test.go:97: run_end outcome = failed, want RunOutcomeInterrupted
    cancellation_interrupt_test.go:112: history has 2 entries, want 3

subtest "in-flight tool call observes cancellation...": PASS (unaffected — confirms this bite isolates exactly the closed-channel mechanism, not the iteration-boundary one)
```
Reverted via a pre-mutation file backup (byte-diff confirmed identical to the pre-bite state); `go build` and the full test re-run both green afterward.

**Task 4.1 RED**:
```
cancellation_interrupt_test.go:287: Steer during the second run returned steering: value is not permitted where it appears, want nil — the steering queue must reopen at Run entry
--- FAIL: TestHarness_Interrupt_SameHarnessRunsNextPrompt
```

**Task 4.3 GREEN**: passes; stable under `go test -race -count=10`.

### Design decisions (recorded, not silent)

1. **S-CAN-001's two-subtest structure.** The scenario's Given ("a harness-driven run whose provider stream is held mid-stream and which has a cancellation-observing tool call in flight") cannot be literally simultaneous within one turn: `Schedule` only runs after that turn's own provider stream reaches `Completion` (`Harness.Run`'s loop is strictly sequential — one `Turn()` call at a time). Also, `finishContinuationTurn` commits a dispatched call's assistant-message and its result-message together, synchronously, in the same invocation — so a call that reaches `Schedule` (even with a cancellation-caused failure result) is never left "open" for orphan synthesis to repair. Resolution, verified empirically before committing to it: split into two independent, non-vacuous subtests — (a) seed `History` via `agent.NewSeededHistory` with one pre-existing open call (literally "a prior tool-calling turn"), hold *this* run's own provider stream, interrupt — proves `loop.go`'s closed-channel cause check + genuinely non-vacuous orphan synthesis (checked via `Entries()[2].Origin()==EntryOriginSynthesized` and its `CallID`); (b) dispatch the cancellation-observing tool for real via a normal (non-held) tool-calling turn, interrupt while it blocks — proves `R-TLS-013` propagates end-to-end through `Harness.Interrupt → runCtx → Turn → Schedule → tool.Run`, and exercises the **iteration-boundary check** specifically (turn 1 completes cleanly; the signal is caught before a futile turn 2). Both subtests independently satisfy every Then clause of S-CAN-001; between them, both of AG-14's cancellation-observation mechanisms (2.4 and 2.6) get their own dedicated, non-overlapping proof — confirmed by the bite (3.1), which shows subtest 1 alone is sensitive to 2.4's removal and subtest 2 is unaffected.
2. **`turn_end(Aborted)` emission on the cause-check path** (loop.go). Not explicit in `design.md`'s prose description of the cause check or of `windDownRun`, but required by `CheckStream`'s own `BracketRoleClosesRun` rule (a run bracket cannot close while a turn bracket is open) — discovered empirically via the task's own required `CheckStream` acceptance assertion, not asserted speculatively. Uses the existing `TurnOutcomeAborted` member (no new `TurnOutcome`, per the hard constraint) with a `*Failure` built from the pre-existing `ai.FailureCategoryCancellation` category. This is scoped to the **turn's** own bracket; the **run's** `run_end` still carries `nil` failure, unchanged, satisfying R-CAN-002 literally.
3. **No `fmt` in `cancellation.go`.** `fmt` transitively imports `os`/`io/fs` (`go list -deps fmt`), tripping the production-closure network/filesystem guard (`import_boundary_test.go`'s check 3) — the exact same issue `src/ai/json_syntax.go`'s own package comment already documents for `encoding/json`. Fixed by hand-rolling `Error()`/`Unwrap()` on small unexported types and using `strconv.Quote` in place of `%q`, matching `scheduler.go`'s own established idiom for this package (none of its existing sentinel-error helpers import `fmt` either). All four import/ambient-authority guards re-verified green with zero import-closure change beyond `time`/`strconv`/`errors` (all stdlib-clean).
4. **Task 12.1 applied early** (out of formal Phase 4 scope, in Phase 2). Adding `Interrupt`/`Shutdown` mechanically breaks the pre-existing `"exactly two exported methods"` pin — `design.md`'s own text calls this out explicitly ("a consequence stated so sdd-apply is not read as a silent test rewrite"). Applying the exact four-name fix immediately (rather than leaving `make test` red until a later batch reaches Phase 12) keeps the safety net intact for the rest of this milestone. No other Phase 12 item (`S-RUN-092`, `S-APP-018`, `S-LSK-018`, `S-LSK-019` confirmations, docs, final gates) was touched.

### Review budget

Batch A diff: ~401 changed lines across 11 modified tracked files + 3 new files (627 lines) ≈ 1028 lines. Well inside the pre-authorized `size:exception` (1800–2900 total estimate across all 14 phases); no budget concern for this batch, remaining budget is ample for Phases 5–13.

## Batch B — Phases 5–8 (11 tasks: 5.1–5.5, 6.1–6.2, 7.1–7.4, 8.1)

**Status**: All 11 assigned tasks complete. Full `./src/agent/...` suite green (`go test -race`, and `make test`) at every checkpoint including the final one (234 `--- PASS` / 0 `--- FAIL`).

**Mode**: Strict TDD. Every behavior RED-recorded (real command run, real failure captured) before its GREEN, with two documented, task-flagged exceptions (5.4 and 6.1) where the task list itself predicts a GREEN-on-first-run composition proof rather than a fresh RED — both confirmed genuinely by real command output, not assumed.

### Files changed

| File | Action | What |
|---|---|---|
| `backend/agent/src/agent/scheduler.go` | Modify | `typedCancellationFailureFromError`/`typedCancellationFailure` (sibling of `typedFailureFromError`); both parked-wait abort arms (the ack-wait `ctx.Done()` and the parked-wait `ctx.Done()`) now build their typed abort from `context.Cause(ctx)` through the new helper (5.2) |
| `backend/agent/src/agent/harness.go` | Modify | `Run` entry: under `signalMu`, a set `shutdown` flag returns `ErrPromptAfterShutdown` immediately, closes `sink`, emits nothing (7.2) |
| `backend/agent/src/agent/cancellation_interrupt_test.go` | Modify | `readUntilDecisionRequired` helper (shared with the shutdown test file); `TestHarness_Interrupt_DuringSuspensionAbortsTyped` (S-CAN-003, 5.1/5.3); `TestHarness_Interrupt_SecondInterruptIsNoOp` (S-CAN-004, 6.1/6.2) |
| `backend/agent/src/agent/cancellation_shutdown_test.go` | Create | `TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal` (S-APP-017 shutdown half, 5.4/5.5); `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts` (S-CAN-005, 7.1/7.3/7.4) |

**Forbidden-list compliance** (re-verified against merge-base `52701436`, via `git diff --name-only` filtered to non-test files): the set of differing PRE-EXISTING non-test files is `{doc.go, harness.go, loop.go, run_events.go, scheduler.go}` — a strict subset of `S-LSK-018`'s pinned six (`tool.go` untouched, correctly deferred to Phase 9, out of this batch's scope); `cancellation.go` is a new file (confirmed absent at merge-base via `git show`), not counted as "pre-existing". Every file on the forbidden list (`turn_events.go`, `failure.go`, `stream_check.go`, `stream_check_test.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `reconstruction_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `go.mod`, `go.sum`) re-verified byte-unchanged (`git diff --stat` per file against merge-base, zero output for each). `run_events.go`/`doc.go`/`doc_contract_guard_test.go` (the released substrate) untouched this batch — no edit needed for Phases 5–8. `permission_protocol_test.go`, `scheduler_test.go`, `tool_test.go` re-verified byte-unchanged and all green (`S-APP-018`).

### TDD Cycle Evidence

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 5.1–5.3 | `TestHarness_Interrupt_DuringSuspensionAbortsTyped` | Integration (Harness+Scheduler) | ✅ full suite green pre-edit | ✅ Written, real failure captured (category `unavailable`, unwrap mismatch) | ✅ Passed, stable ×10 under `-race` | ➖ Single scenario (S-CAN-003 has one Then group) | ➖ None needed |
| 5.4–5.5 | `TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal` | Integration (Harness+Scheduler) | ✅ full suite green | ⚠️ Composition proof, GREEN-on-first-run (task's own documented exception — 5.2's mechanism already covers both abort arms symmetrically) | ✅ Passed, stable ×10 under `-race` | ➖ Single scenario | ➖ None needed |
| 6.1–6.2 | `TestHarness_Interrupt_SecondInterruptIsNoOp` | Integration (Harness) | ✅ full suite green | ⚠️ Composition proof, GREEN-on-first-run (no new production code — 2.2's `signalMu` plus Go's own cancel-func idempotency) | ✅ Passed, stable ×10 under `-race -count=10` | ➖ Single scenario | ➖ None needed |
| 7.1–7.3 | `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts` | Integration (Harness) | ✅ full suite green | ✅ Written, real failure captured (second-run refusal missing) | ✅ Passed, stable ×10 under `-race` | ➖ Single scenario (S-CAN-005 has one Then group spanning two acts) | ➖ None needed |
| 8.1 | bite `S-CAN-011` (scratch-revert `harness.go`'s terr-based cause check) | Integration (Harness) | ✅ full suite green before/after | ✅ Real failure captured: both `S-CAN-001`'s Arm A and `S-CAN-005` FAIL reporting `RunOutcomeFailed`/`Unavailable`; Arm B unaffected (isolates the bitten mechanism, mirrors `S-CAN-010`'s own precedent) | ➖ (bite, reverted, not landed) | ➖ N/A | ➖ N/A |

### Test Summary
- **Total tests written this batch**: 4 top-level test functions (`TestHarness_Interrupt_DuringSuspensionAbortsTyped`, `TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal`, `TestHarness_Interrupt_SecondInterruptIsNoOp`, `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts`), covering `S-CAN-003`, both halves of `S-APP-017`, `S-CAN-004`, `S-CAN-005`.
- **Total tests passing**: full `./src/agent/...` suite green (`go test -race`, and `make test`) at every checkpoint; 234 `--- PASS` / 0 `--- FAIL` in the final full-package run.
- **Layers used**: Integration via `agenttest`-scripted `Harness`/`Scheduler` (4 new top-level tests, all this layer; no unit or E2E layer needed this batch).
- **Bites (RED-record → revert)**: 1 (task 8.1 / `S-CAN-011`).
- **Pure functions created**: `typedCancellationFailureFromError`, `typedCancellationFailure` (scheduler.go — deterministic, no side effects, siblings of the pre-existing `typedFailureFromError`).

### RED/bite evidence (verbatim, captured)

**Task 5.1 RED** (`go test -race -run TestHarness_Interrupt_DuringSuspensionAbortsTyped -v ./src/agent/...`):
```
    cancellation_interrupt_test.go:487: failure.Category() = unavailable, want ai.FailureCategoryCancellation
    cancellation_interrupt_test.go:490: failure does not unwrap to errors.Is-match agent.ErrInterrupted
--- FAIL: TestHarness_Interrupt_DuringSuspensionAbortsTyped (0.00s)
FAIL
```
Every other assertion in the same run (Run error `errors.Is(ErrInterrupted)`, `WakeParked`→`ErrStrayDecision`, `hist.CloseTurn()`, `CheckStream`) already passed pre-5.2 — only the failure-shape assertions this task targets were red, precisely isolating `scheduler.go`'s two abort arms as the missing mechanism.

**Task 5.1 GREEN** (same command, after 5.2 landed): `--- PASS`, stable ×10 under `-race -count=10`.

**Task 5.4 first run** (`go test -race -run TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal -v ./src/agent/...`, written and run only after 5.2 had already landed): `--- PASS` immediately — a genuine composition proof, not a fresh RED, exactly as the task's own text predicts ("Expect FAIL until 5.2 lands ... already green by this point"). Stable ×10 under `-race -count=10`.

**Task 6.1 first run** (`go test -race -count=10 -run TestHarness_Interrupt_SecondInterruptIsNoOp -v ./src/agent/...`): 10/10 `--- PASS`, no panic, no data race — no new production code required; both `Interrupt()` calls are fired from independent goroutines synchronized only by a `sync.WaitGroup` (never sequenced relative to each other), so either can race the other and the `Run` goroutine's own wind-down.

**Task 7.1 RED** (`go test -race -run TestHarness_Shutdown_WindsDownAndRefusesNewPrompts -v ./src/agent/...`, before harness.go's 7.2 edit):
```
    cancellation_shutdown_test.go:221: second Run error = agenttest: Stream call 2 of 1 scripted: agenttest: script queue exhausted, want errors.Is to match agent.ErrPromptAfterShutdown
    cancellation_shutdown_test.go:224: second Run error = agenttest: Stream call 2 of 1 scripted: agenttest: script queue exhausted, want errors.Is to match agent.ErrShutdown (the refusal wraps it)
    cancellation_shutdown_test.go:234: second Run emitted 3 event(s), want zero
--- FAIL: TestHarness_Shutdown_WindsDownAndRefusesNewPrompts (0.00s)
FAIL
```
**Root-cause note** (divergence from `tasks.md`'s own prose, recorded so it is not silent, mirroring batch A's own precedent for task 2.3): the task predicted "`RunOutcomeShutdown` not yet reachable from `Shutdown()` (routes as interrupted...)" as part of the expected failure. Empirically, that half of the assertion set (first run's `RunOutcomeShutdown` outcome, `errors.Is(err, ErrShutdown)`, synthesized-origin entries, `CheckStream`) already PASSED before 7.2 landed, because task 2.6's own `windDownRun` outcome-mapping (`if errors.Is(cause, ErrShutdown) { outcome = RunOutcomeShutdown }`) already existed from batch A. The ONLY genuinely red assertions were the second-run-refusal ones — and the concrete failure mode was the second `Run` actually proceeding (nothing yet refused it) and hitting the fake provider's exhausted script queue (only one script was configured for this test), not a "successful but wrong" completion. The underlying causality this test isolates — the missing typed refusal at `Run` entry — is identical either way; only the specific surfaced error text differs from the task's own prediction.

**Task 7.1 GREEN** (same command, after 7.2 landed): `--- PASS`, stable ×10 under `-race -count=10`.

**Task 8.1 bite (S-CAN-011) — scratch-mutation evidence**: with `harness.go`'s terr-based cause check (`if cause := context.Cause(runCtx); errors.Is(cause, ErrInterrupted) || errors.Is(cause, ErrShutdown) { return h.windDownRun(...) }`) reverted to an unconditional `return h.failRun(sink, stamper, runID, terr)`:
```
=== NAME  TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized/provider_cancelled_mid-stream,_a_genuinely_pre-existing_open_call_is_synthesized
    cancellation_interrupt_test.go:104: run_end outcome = failed, want RunOutcomeInterrupted
    cancellation_interrupt_test.go:107: run_end outcome = RunOutcomeFailed, want never Failed for an interrupted run
    cancellation_interrupt_test.go:110: run_end carries a *Failure, want none for an interrupted run
    cancellation_interrupt_test.go:119: history has 2 entries, want 3 (seeded call, run's own prompt, synthesized result)
=== NAME  TestHarness_Shutdown_WindsDownAndRefusesNewPrompts
    cancellation_shutdown_test.go:192: run_end outcome = failed, want RunOutcomeShutdown
    cancellation_shutdown_test.go:198: run_end outcome = RunOutcomeFailed, want never Failed for a shut-down run
    cancellation_shutdown_test.go:201: run_end carries a *Failure, want none for a shut-down run
    cancellation_shutdown_test.go:211: history has 2 entries, want 3 (seeded call, run's own prompt, synthesized result)
--- FAIL: TestHarness_Shutdown_WindsDownAndRefusesNewPrompts (0.00s)
--- FAIL: TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized (0.00s)
    --- PASS: TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized/in-flight_tool_call_observes_cancellation_through_the_harness (0.00s)
    --- FAIL: TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized/provider_cancelled_mid-stream,_a_genuinely_pre-existing_open_call_is_synthesized (0.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/agent	0.527s
```
Empirically confirmed (via a temporary, since-removed diagnostic `t.Logf`) that the run_end failure's category is `unavailable` in both cases — matching `wrapHarnessFailure`'s hardcoded `ai.FailureCategoryUnavailable`. `S-CAN-001`'s Arm B ("in-flight tool call observes cancellation") stays PASS, unaffected — proving this bite isolates exactly the terr-based cause check (the mechanism this task targets), not the iteration-boundary check (a separate mechanism, task 2.6's other half, which Arm B relies on and which this bite deliberately leaves untouched — mirroring `S-CAN-010`'s own established isolation precedent from batch A). Reverted via a pre-mutation file backup (`cp harness.go harness.go.bak` before mutating, `cp harness.go.bak harness.go` after, `diff` confirmed byte-identical, backup removed); `go build`, `go vet`, and the full test re-run (`go test -race -count=1 ./src/agent/...`, `make test`) both green afterward.

### Design decisions (recorded, not silent)

5. **`S-CAN-011`'s bite scope: the terr-based routing only, not the iteration-boundary check.** Task 2.6 landed both mechanisms in one commit (batch A), and task 8.1's own text says "route every non-nil `Turn` error to `failRun` unconditionally" — a phrase that describes only the `terr != nil` branch, not the iteration-boundary check (which fires before calling `Turn` again, unconditionally, independent of any `Turn` error). Reverting only the terr-based branch leaves `S-CAN-001`'s Arm B (which relies exclusively on the iteration-boundary check, since its own `Turn()` call returns a nil error — the scheduler-level cancellation failure is absorbed into a normal tool-result rejoin) correctly unaffected — an isolation outcome verified empirically, not assumed, and consistent with `S-CAN-010`'s own already-established bite discipline from batch A (one arm fails, one arm passes = the bite is precisely scoped, not overbroad).
6. **`typedCancellationFailureFromError`'s fallback to `typedExecutionFailureFromError`.** Rather than unconditionally building a `Cancellation`-category failure whenever this helper is called, it first checks whether `cause` actually matches one of the two harness sentinels; if not (a bare caller-context cancellation never routed through `Interrupt`/`Shutdown`), it falls through to the pre-existing `typedExecutionFailureFromError` path unchanged — preserving `R-CAN-001`'s scope line ("a bare cancellation of the caller's own context ... MUST NOT be reported as interrupted or as shutdown") at the per-call abort level, not just the run level. Not spelled out as an explicit branch in `design.md`'s one-line description of the helper, but required by `R-CAN-001`'s own normative scope line, which binds every cancellation-observation site the requirement names — the permission gate's two abort arms are named explicitly there.
7. **Task 7.1's RED diverged from `tasks.md`'s own prediction in its concrete failure mode, not in its underlying causality.** See the "Root-cause note" above — recorded per this project's own established discipline (batch A's task 2.3 precedent) of not silently treating a task's predictive prose as ground truth once the real command output is in hand.

### Review budget

Batch B diff: 289 insertions + 5 deletions across 3 modified tracked files, plus 1 new file (236 lines) ≈ 530 changed lines. Cumulative with batch A (~1028) ≈ 1558 lines, still within the pre-authorized `size:exception` (1800–2900 total estimate across all 14 phases); ample budget remains for Phases 9–13.

### Remaining (Phases 9–13, NOT this batch)

Phase 9 (wind-down bound, detach select, `Scheduler.WindDownBound`, `R-CAN-006`/`R-TLS-014`), Phase 10 (goroutine-leak proof), Phase 11 (bite S-CAN-012), Phase 12 (remaining guards/cross-cut confirmations — `12.1` already done, see batch A above), Phase 13 (final gates, docs, promotion prep).

## Batch C — Phases 9–13 (23 tasks: 9.1–9.8, 10.1–10.2, 11.1, 12.2–12.6, 13.1–13.4) — COMPLETE

**Status**: All 23 assigned tasks complete (task 13.5 is explicitly `sdd-archive`'s job, left `[ ]`). Full `./src/agent/...` suite green (`go test -race`, and `make test`) at every checkpoint including the final one (1262 `--- PASS` / 0 `--- FAIL` across all 12 module packages in the final `make test`).

**Mode**: Strict TDD. Every behavior RED-recorded (real command run, real failure captured) before its GREEN, with two documented, task-flagged composition-proof exceptions (9.6, 10.1) and one genuine, unpredicted RED (9.7) whose real failure mode diverged from — and corrected — the design/spec's own prose, recorded rather than silently reconciled.

### Files changed

| File | Action | What |
|---|---|---|
| `backend/agent/src/agent/tool.go` | Modify | `Scheduler.WindDownBound time.Duration` field (zero-default, `LeaveSinkOpen` precedent), `"time"` import (9.1) |
| `backend/agent/src/agent/scheduler.go` | Modify | `toolRunReply` type; `runToolWithWindDown` (new sub-method: inner-goroutine detach select, bound armed only on `ctx.Done()`); `executeCall`'s tool-invocation section rewritten to call it; `typedDetachedCallFailure` (new helper, siblings of `typedCancellationFailure`); `"time"` import (9.3–9.4) |
| `backend/agent/src/agent/cancellation_winddown_test.go` | Create | `smallWindDownBound` const, `readUntilToolStart` helper, `TestHarness_WindDown_DeafToolCannotHoldRunHostage` (S-CAN-006 Thens 1–2, 9.2/9.5), `TestHarness_WindDown_NoHarnessGoroutineRemains` (R-CAN-006 Then 3, serial-only, 10.1/10.2) |
| `backend/agent/src/agent/scheduler_test.go` | Modify | `TestSchedule_UncancelledZeroBoundDoesNotArmTimer` (S-TLS-018, 9.6), `TestSchedule_MixedCancellationBatch_DisjointResultsPerCall` (S-TLS-019, 9.7/9.8) — both purely additive, zero pre-existing lines touched |
| `backend/agent/src/agent/cancellation.go` | Modify | Package comment reworded to the file's own established `"Package agent is Layer 2..."` convention (real `make lint` finding on AG-14's own batch-A file, not pre-existing drift — task 13.4) |
| `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` | Modify | `:2171` exit-gate row ticked `[x]`; status paragraph at `:3` bumped "13 of 24" → "14 of 24", "AG-12…AG-13" → "AG-12…AG-14", new AG-14 sentence appended mirroring AG-13's own shape (task 13.2) |
| `openspec/changes/cachicamas-agent-cancellation-tree/tasks.md` | Modify | Tasks 9.1–9.8, 10.1–10.2, 11.1, 12.2–12.6, 13.1–13.4 marked `[x]` with apply notes |

**Forbidden-list compliance** (re-verified against merge-base `52701436` via `git diff --name-only`/`--stat` and `git status`, combining tracked-diff and untracked-new-file views since `cancellation_winddown_test.go` is untracked): the complete changed-file set under `backend/agent/src/agent/` is 17 files — `{doc.go, harness.go, loop.go, run_events.go, scheduler.go, tool.go}` (6, pre-existing non-test, **exactly** `S-LSK-018`'s pinned set — no seventh), `cancellation.go` (1, new non-test, confirmed absent at merge-base via `git show`), `{doc_contract_guard_test.go, harness_test.go, loop_hook_test.go, loop_test.go, scheduler_test.go, scripted_tool_test.go}` (6, pre-existing test, modified — `scheduler_test.go`'s own diff independently confirmed purely additive, zero `^-` lines), `{cancellation_events_test.go, cancellation_interrupt_test.go, cancellation_shutdown_test.go, cancellation_winddown_test.go}` (4, new test). Every forbidden-list file (`turn_events.go`, `failure.go`, `stream_check.go`, `stream_check_test.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `reconstruction_test.go`, `history.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `go.mod`, `go.sum`) re-verified byte-unchanged (0-line diff vs `origin/main`, each checked individually). Every file under `backend/agent/src/ai/**` byte-unchanged (empty `git diff --stat` over the whole subtree). `permission_protocol_test.go` byte-unchanged (0-line diff) and its pinned `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` passes 5/5 under `-race`. `run_events.go`'s diff (unchanged from batch A, re-verified) adds only `RunOutcomeShutdown` + its `String()` case — `RunEnd.validate`/`NewRunEnd` untouched.

### TDD Cycle Evidence

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 9.1–9.5 | `TestHarness_WindDown_DeafToolCannotHoldRunHostage` | Integration (Harness+Scheduler) | ✅ full suite green pre-edit | ✅ Written, real bounded-hang FAIL captured (`drainSink`'s own 1s guard, not the outer `-timeout`) | ✅ Passed, stable ×10 under `-race` | ➖ Single scenario (S-CAN-006 Thens 1–2 share one Given) | ➖ None needed |
| 9.6 | `TestSchedule_UncancelledZeroBoundDoesNotArmTimer` | Integration (Scheduler) | ✅ full suite green | ⚠️ Composition proof, GREEN-on-first-run (9.1–9.4's mechanism already correctly leaves `ctx.Done()`'s nil channel unreachable for `context.Background()`) | ✅ Passed | ➖ Single scenario | ➖ None needed |
| 9.7–9.8 | `TestSchedule_MixedCancellationBatch_DisjointResultsPerCall` | Integration (Scheduler) | ✅ full suite green pre-edit | ✅ Written, real failure captured — genuinely unpredicted (category mismatch, not the task's own silence about which failure mode); test corrected, re-run | ✅ Passed, stable ×10 under `-race` | ➖ Single scenario (3 ordinal slots inspected together) | ➖ None needed |
| 10.1–10.2 | `TestHarness_WindDown_NoHarnessGoroutineRemains` | Integration (Harness, serial-only) | ✅ full suite green | ⚠️ Composition proof, GREEN-on-first-run (Phase 9's mechanism, confirmed correct by 9.5, composes with `RequireNoGoroutineLeak`'s amplitude check without new production code) | ✅ Passed, stable ×3 under `-race` | ➖ Single scenario | ➖ None needed |
| 11.1 | bite `S-CAN-012` (scratch-remove the armed bound in `runToolWithWindDown`) | Integration (Harness+Scheduler) | ✅ full suite green before/after | ✅ Real failure captured: `TestHarness_WindDown_DeafToolCannotHoldRunHostage` FAILs via `drainSink`'s 1s guard, same symptom as 9.2's own original RED | ➖ (bite, reverted, not landed) | ➖ N/A | ➖ N/A |

### Test Summary
- **Total tests written this batch**: 4 top-level test functions (`TestHarness_WindDown_DeafToolCannotHoldRunHostage`, `TestHarness_WindDown_NoHarnessGoroutineRemains`, `TestSchedule_UncancelledZeroBoundDoesNotArmTimer`, `TestSchedule_MixedCancellationBatch_DisjointResultsPerCall`), covering `S-CAN-006` (both Thens 1–2 and Then 3), `S-TLS-018`, `S-TLS-019`/`R-TLS-010` restated.
- **Total tests passing**: full `./src/agent/...` suite green (`go test -race`, and `make test`) at every checkpoint; 1262 `--- PASS` / 0 `--- FAIL` in the final `make test` run across all 12 module packages.
- **Layers used**: Integration via `agenttest`-scripted `Harness`/`Scheduler` (4 new top-level tests, all this layer; no unit or E2E layer needed this batch).
- **Bites (RED-record → revert)**: 1 (task 11.1 / `S-CAN-012`).
- **Pure functions created**: `typedDetachedCallFailure` (scheduler.go — deterministic, no side effects, sibling of `typedCancellationFailure`).
- **New production mechanism**: `runToolWithWindDown` (scheduler.go) — the per-call detach select, extracted as a named sub-method rather than inlined into `executeCall` (design decision 9 below).

### RED/bite evidence (verbatim, captured)

**Task 9.2 RED** (`go test -race -timeout 8s -run TestHarness_WindDown_DeafToolCannotHoldRunHostage -v ./src/agent/...`, before scheduler.go's 9.3/9.4 detach select landed):
```
=== RUN   TestHarness_WindDown_DeafToolCannotHoldRunHostage
=== PAUSE TestHarness_WindDown_DeafToolCannotHoldRunHostage
=== CONT  TestHarness_WindDown_DeafToolCannotHoldRunHostage
    cancellation_winddown_test.go:100: drainSink: sink did not close within 1s (0 event(s) received so far) — the loop is not closing the sink it owns
--- FAIL: TestHarness_WindDown_DeafToolCannotHoldRunHostage (1.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/agent	1.580s
```
**Root-cause note** (divergence from `tasks.md`'s own prediction — "compile error, `WindDownBound` unread" — recorded per this project's established discipline of not silently treating a task's predictive prose as ground truth): task 9.1 (adding the field) landed first per the task's own ordering, so there was no compile error; the genuine RED is behavioral — `executeCall` still calls `tool.Run(ctx, …)` **synchronously**, so the never-closed `release` channel blocks the call goroutine forever, `wg.Wait()` in `Schedule` never returns, and the test's own `drainSink` helper's pre-existing 1-second bounded guard (not the outer `-timeout 8s`) is what actually catches the hang — a real, deterministic, captured FAIL either way.

**Task 9.2 GREEN** (same command, after 9.3/9.4 landed): `--- PASS`, stable ×10 under `-race -count=10`.

**Task 9.7 RED** (`go test -race -timeout 20s -run TestSchedule_MixedCancellationBatch_DisjointResultsPerCall -v ./src/agent/...`, first draft, before correction):
```
=== RUN   TestSchedule_MixedCancellationBatch_DisjointResultsPerCall
=== PAUSE TestSchedule_MixedCancellationBatch_DisjointResultsPerCall
=== CONT  TestSchedule_MixedCancellationBatch_DisjointResultsPerCall
    scheduler_test.go:1430: results[0].Failure.Category() = unavailable, want ai.FailureCategoryCancellation
--- FAIL: TestSchedule_MixedCancellationBatch_DisjointResultsPerCall (0.02s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/agent	0.313s
```
**Root-cause note (a genuine finding, not a production bug)**: the first draft asserted `ai.FailureCategoryCancellation` for BOTH the cancellation-observing tool's own slot AND the cancellation-deaf tool's slot. Empirically only the DEAF tool's slot is scheduler-detected cancellation (`runToolWithWindDown`'s timer arm → `typedDetachedCallFailure`, a structurally different code path from `runErr != nil`). The OBSERVING tool's own self-reported early-return error (`ErrScriptedToolCancelled`) reaches the scheduler as an ordinary non-nil `error` from `tool.Run` — the exact same pre-existing, unmodified `runErr != nil` branch any tool error takes, wrapped by `typedExecutionFailureFromError`/`typedFailureFromError`, which hardcodes `ai.FailureCategoryUnavailable`. This is not new to AG-14: `TestSchedule_ToolReceivesRunContext_EarlyReturnOnCancel` (S-TLS-016, batch A, unmodified) already asserts only `errors.Is` against the tool's own sentinel for this exact call — it never checks Category, precisely because the scheduler cannot distinguish "this tool errored because it observed cancellation" from "this tool errored for any other reason." **Second finding, also recorded**: the spec's restated `R-TLS-010` (`agent-tool-scheduler/spec.md`) names `ai.FailureCategoryExecution` as the category for "an ordinary tool error" — this member **does not exist** anywhere in `ai.FailureCategory`'s nine-member vocabulary (`Authentication, Authorization, RateLimit, Unavailable, Timeout, Cancellation, MalformedResponse, UnsupportedCapability, Unknown`; verified directly against `provider_failure.go`, not merely cited) — a spec-authoring defect distinct from `R-TLS-014`'s own unambiguous, correctly-implemented text (independently confirmed by task 9.2/9.5's own test). `backend/agent/src/ai/**` being off-limits this milestone forbids "fixing" this by inventing a tenth vocabulary member; the correction is to the test's own expectation, matching the codebase's actual, correct, pre-existing behavior. The test was also narrowed from 4 tools to the scenario's literal 3-tool Given (observing, deaf, succeeding) — the fourth "ordinary failing" tool was never named in `S-TLS-019`'s own Given clause.

**Task 9.7 GREEN** (corrected test, same command): `--- PASS`, stable ×10 under `-race -count=10`.

**Task 11.1 bite (S-CAN-012) — scratch-mutation evidence**: `runToolWithWindDown`'s `select` block (the `case <-ctx.Done(): ... timer := time.NewTimer(bound) ...` arm) reverted to an unconditional `reply = <-resCh; return reply, false` — no `ctx.Done()` arm at all, matching the removal of the now-unused `"time"` import (mirrors task 3.1's own precedent of removing a now-dead import as part of the scratch mutation):
```
go build ./... → BUILD_DONE (compiles cleanly — the bite is a behavioral regression, not a build failure)

go test -race -timeout 8s -run TestHarness_WindDown_DeafToolCannotHoldRunHostage -v ./src/agent/...
=== RUN   TestHarness_WindDown_DeafToolCannotHoldRunHostage
=== PAUSE TestHarness_WindDown_DeafToolCannotHoldRunHostage
=== CONT  TestHarness_WindDown_DeafToolCannotHoldRunHostage
    cancellation_winddown_test.go:100: drainSink: sink did not close within 1s (0 event(s) received so far) — the loop is not closing the sink it owns
--- FAIL: TestHarness_WindDown_DeafToolCannotHoldRunHostage (1.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/agent	1.458s
```
Same divergence from the task's own prediction as 9.2's RED: `drainSink`'s own 1s bounded guard catches the hang before the outer `go test -timeout 8s` ever would — a real, deterministic, captured FAIL that proves the armed bound (not some other mechanism) is what ends the run, exactly what the bite exists to isolate. Reverted via `cp scheduler.go scheduler.go.bak` before mutating, `cp scheduler.go.bak scheduler.go` after, `diff` confirmed byte-identical, backup removed; `git diff -- scheduler.go` against HEAD afterward shows only Phase 9's real, permanent changes (128 insertions, 1 deletion) — zero bite residue. `go build`, and the full test re-run (`go test -race ./src/agent/...`) both green afterward.

### Design decisions (recorded, not silent) — continuing batches A/B's numbering

8. **`typedDetachedCallFailure` calls `typedCancellationFailure` directly, never `typedCancellationFailureFromError`.** The latter checks `errors.Is(cause, ErrInterrupted) || errors.Is(cause, ErrShutdown)` and falls back to the pre-AG-14 `Unavailable` wrap when neither matches — correct for the permission-gate's abort arms (whose cause genuinely is one of the two harness sentinels), but wrong here: a `*DetachedCallError` is its own cause, never wrapping either sentinel, so routing it through the sentinel-checking wrapper would silently mis-categorize every bound-overrun report as `Unavailable` instead of `Cancellation` — directly contradicting `R-CAN-006`'s own unambiguous text. Caught during implementation (not by a failing test — this is a design choice made correctly the first time, recorded so a future reader does not "simplify" it into the wrong helper).
9. **`runToolWithWindDown` is a named, documented sub-method, not an inline block inside `executeCall`.** Design.md's own code sketch shows the mechanism inlined; extracting it keeps `executeCall`'s existing structure (gate → start event → run → disjoint-channel handling) readable, matches the file's own established sub-method pattern (`scheduleRead`, `scheduleSerialized`, `runPermissionGate` are all named siblings), and gives the "own RED-first task" framing (task 9.3) a single, isolable unit whose own doc comment carries the full rationale (why per-call, why not `wg.Wait()`, why the buffered channel, why panic re-propagation) rather than scattering it through `executeCall`'s body.
10. **`S-TLS-019`'s test scope corrected to the scenario's own literal 3-tool Given, and its category split corrected to match verified, unmodified production behavior — not the spec's own (defective) category name.** Both findings are recorded verbatim in the RED evidence above rather than silently reconciled; they are spec-text/task-prose imprecisions, not production defects, and no `ai/**` edit was made or considered to "fix" the nonexistent category name.
11. **`docs/.../0003-...md:2207`'s R-09 row left untouched.** It already cites `AG-14.1` (Interrupt) as consuming `R-09`'s upward path; `AG-14.2` (Shutdown) uses the exact same mechanism (`signalMu`/`cancelRun`) and is arguably equally-valid evidence, but the table's own citation granularity varies elsewhere (some rows cite a whole milestone, others cite specific sub-nodes), so omitting `AG-14.2` here is a stylistic choice, not a factual error — left as-is per the batch's own conservative instruction ("back-annotate only if a discrepancy is found — do not rewrite a correct row"). `:2257`'s traceability row (`AG-14 | R-08 (wind-down), v2 § 4.2`) was also verified correct: `R-08`'s "typed mid-stream failure path" step is exactly what `loop.go:442-449`'s cause check extends to the cancellation case, mirroring the pre-existing mid-stream-fatal branch's own precedent.
12. **`gofmt -l` drift (`loop.go:704`, `scheduler.go:635+`, plus a dozen-plus files this milestone never touched) independently re-verified pre-existing at `origin/main`**, not by trusting the brief's own claim but by extracting standalone copies of five representative files (including three on the hard forbidden list) directly from `origin/main` via `git show` and running `gofmt -l` on them in isolation — all five flagged, confirming the drift predates AG-14 entirely. `make fmt`/`gofmt -w` were never run, per the hard constraint; this is recorded as a maintainer decision for a separate change, not fixed here.
13. **`cancellation.go`'s package comment fixed** (batch A's own file, not this batch's new code) — a real, in-scope `golangci-lint` `revive` finding (`package-comments`), not pre-existing repo-wide drift: the file's opening comment read `"AG-14 — the cancellation vocabulary..."` where every sibling production file (`tool.go`, `harness.go`, `loop.go`, `scheduler.go`) opens with `"Package agent is Layer 2 of the cachicamas agent stack. This file (X.go) hosts ..."`. Fixed via a targeted, minimal reword of the opening lines only — the rest of the comment block, and every other line in the file, is untouched. This is not the `gofmt -w` reformatting the hard constraint forbids; it is a single, deliberate content edit to satisfy a real lint rule on a file within this change's own scope.

### Confirmations against the specs (task 13.3, evidence beyond what's already in Phases 9–12 above)

- **`R-TLS-013`** ("MUST pass the context it was called with down into `tool.Run`. MUST NOT substitute a background or otherwise detached context"): `scheduler.go`'s `runToolWithWindDown` receives `ctx` from `executeCall` (itself receiving the real run `ctx`, Phase 1) and passes it unmodified to `tool.Run(ctx, args, policy)` inside the inner goroutine — confirmed by direct code reading, not inference.
- **`R-TLS-014`**: every normative clause (zero-default field resolving to the package default; not a new `Schedule` parameter; armed only by cancellation, no timer on the uncancelled path; per-call detachment, not around the join; the third-party frame's freedom to complete its send after detachment; the typed report via the existing execution-failure path with no new `EventKind`/`Result` outcome; `Schedule`'s remaining steps unchanged in order; panic containment on both sides of the bound) verified directly against `runToolWithWindDown`'s and `typedDetachedCallFailure`'s own implementation, each independently confirmed by a passing test (9.5, 9.6, or the panic-containment re-panic path, which the pre-existing `S-TLS-011` panic test — file-unchanged — continues to exercise through the now-detached-but-unchanged-in-contract `recoverCall`).
- **`R-TLS-010` restated**: confirmed via `S-TLS-019`'s corrected test (task 9.7/9.8) — every populated slot's `Outcome`/`Failure` pairing verified disjoint; the category split (`Cancellation` only for scheduler-detected bound overrun, `Unavailable` for a tool's own self-reported error, matching pre-existing behavior) is the ONE place this batch found the restated spec prose imprecise (design decision 10 above).
- **`R-RUN-010`** ("the wind-down bound of `R-CAN-006` is NOT the third path... part of path (b)... no timer on the uncancelled path"): both of the requirement's own "two independent reasons, both checkable" are directly confirmed by this batch's own work — reason 1 (part of path (b)) by `runToolWithWindDown`'s own structure (the timer only ever exists inside the `ctx.Done()` arm); reason 2 (no timer on the uncancelled path) by `S-TLS-018`'s own test (task 9.6) AND by the unmodified `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` (task 12.2).
- **`R-LSK-004`, `R-RUN-001`, `R-RUN-011`, `R-APP-009`, `R-HIS-007`**: closed in batches A/B; this batch's own changes never touch `harness.go`'s signal/queue/wind-down logic, `loop.go`'s cause check, or `scheduler.go`'s permission-gate abort arms — confirmed unaffected by the full green suite (every S-CAN-00x/S-APP-01x test from batches A/B still passing unmodified this batch) and by the byte-unchanged/purely-additive file-diff checks above.

### Review budget

Batch C diff: `tool.go` +16/-0, `scheduler.go` +129/-2 (net across the whole file's cumulative diff, includes the executeCall rewrite), `scheduler_test.go` +192/-0 (purely additive), `cancellation.go` +4/-2 (comment fix), plus 1 new file `cancellation_winddown_test.go` (196 lines), plus doc updates (`0003-....md` +2/-2, `tasks.md` +20/-20 as `[x]` marks + apply notes) ≈ 579 changed lines. Cumulative with batch A (~1028) and batch B (~530) ≈ 2137 lines — within the pre-authorized `size:exception` (1800–2900 total estimate across all 14 phases).

### Final gates (task 13.4, full evidence)

- **`make test`** (`go test -race -v ./...`): 1262 `--- PASS` / 0 `--- FAIL`, all 12 module packages report `ok`.
- **`loop.go` coverage** (task 13.1): 88.01% (257/292 statements) via `go test -race -coverprofile=... -covermode=atomic ./src/agent/...` then `go tool cover -func`, cross-checked against the raw profile's per-statement hit counts for the new cancellation branch (`loop.go:442-449`) — all non-zero.
- **`golangci-lint cache clean && make lint`**: found and fixed one real issue (design decision 13 above); re-run after the fix: `go vet ./...` clean, `golangci-lint run --config=.golangci.yml ./...` → `0 issues.`
- **`make build`** (`go build -trimpath ./...`): clean, no output.
- **`make vuln-check`** (`govulncheck -json ./...`, explicit — not part of `make all`): exit code 0; JSON stream parsed (via `jq`) for `finding` entries — **zero** findings; ~180 `osv` entries were scanned against the module's stdlib-only dependency closure (this module carries zero third-party `require`s per its own `go.mod`/Makefile documentation), none reachable.
- **`make fmt`/`gofmt -w`**: never run, per the hard constraint — the existing `gofmt -l` drift (design decision 12 above) is independently re-confirmed pre-existing at `origin/main`, not AG-14-introduced.

### Remaining (Phase 13.5 only, NOT this batch, explicitly `sdd-archive`'s)

Promote `agent-cancellation-tree/spec.md` to `openspec/specs/agent-cancellation-tree/spec.md`; apply the five deltas into their canonical specs; archive the change folder after `sdd-verify` passes, per AG-09..AG-13 precedent. **All 57 tasks across Phases 0–13 are otherwise complete (56 `[x]`, only 13.5 `[ ]`).**

## Remediation — sdd-verify findings MAJOR-3 and MINOR-3 (bounded pass, post-verify)

**Scope**: exactly the two findings named below, per the orchestrator's bounded remediation brief. No other content in this file was touched; batches A/B/C above are preserved verbatim.

### MAJOR-3 — turn-close failure category untested, now pinned

**Finding** (verify-report.md): `R-CAN-002`'s turn-bracket clause requires the cancelled turn's own `turn_end` to carry `TurnOutcomeAborted` with a `*Failure` of category `ai.FailureCategoryCancellation` (`loop.go:442-449`), but no test in the package read a `turn_end`'s outcome or failure category — `CheckStream` only pinned the bracket's existence/well-formedness. A regression to `ai.FailureCategoryUnavailable`, or a different outcome, would have stayed green.

**Fix**: added assertions locating the `turn_end` event in the captured stream and checking outcome == `agent.TurnOutcomeAborted`, `*Failure` non-nil, category == `ai.FailureCategoryCancellation`, to:
- `cancellation_interrupt_test.go`, `TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized`'s Arm A subtest ("provider cancelled mid-stream...") — the subtest whose own doc comment identifies it as the one that fires loop.go's closed-channel cause check.
- `cancellation_shutdown_test.go`, `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts` — verified first (by tracing `heldTurnScript`+gate+`Hold`'s own "blocks until released or ctx cancelled" contract) to share Arm A's exact shape, hence to exercise the SAME `loop.go:442-449` path with the shutdown sentinel instead of the interrupt one. `TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal` was checked and found NOT to exercise this path (its abort routes through the scheduler's suspension mechanism, reporting via `tool_end_execution_failure`, never producing a `turn_end(Aborted)`) — skipped, not force-covered, per the brief's own instruction.

**RED discipline (real, captured)**: mutated `loop.go:568` (`cancellationTurnFailure`)'s `Category: ai.FailureCategoryCancellation` to `ai.FailureCategoryUnavailable`. Ran the two focused tests:

```
$ go test -race -v -count=1 -run 'TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized|TestHarness_Shutdown_WindsDownAndRefusesNewPrompts' ./src/agent/...
cancellation_interrupt_test.go:142: turn_end failure.Category() = unavailable, want ai.FailureCategoryCancellation
cancellation_shutdown_test.go:232: turn_end failure.Category() = unavailable, want ai.FailureCategoryCancellation
--- FAIL: TestHarness_Shutdown_WindsDownAndRefusesNewPrompts (0.00s)
--- FAIL: TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized (0.00s)
    --- PASS: .../in-flight_tool_call_observes_cancellation_through_the_harness (0.00s)
    --- FAIL: .../provider_cancelled_mid-stream,_a_genuinely_pre-existing_open_call_is_synthesized (0.00s)
FAIL
```

Arm B ("in-flight tool call observes...") correctly stayed PASS — independent confirmation it does not exercise this branch (matches the design's own routing: Arm B winds down through the iteration-boundary check, never loop.go's closed-channel cause check). Reverted the mutation; `git diff -- backend/agent/src/agent/loop.go` produced zero output (byte-identical revert, confirmed both immediately after reverting and again at the very end of this remediation pass). Re-ran the same focused command: both tests GREEN.

### MINOR-3 — harness.go defer-order fix (`Run`'s cancelRun/close(sink) ordering)

**Finding** (verify-report.md): `harness.go:362-370` registered `defer func(){ cancelRun=nil; cancel(nil) }()` BEFORE `defer close(sink)`. LIFO execution therefore closed `sink` FIRST and cleared `h.cancelRun` SECOND. A stream-only consumer (R-CAN-005's own shape) that starts run #2 the instant it observes run #1's sink close could have run #2 register its own `cancelRun`, then have run #1's still-pending defer null it back to `nil` — silently making a subsequent `Interrupt()` on run #2 a no-op. Both writes are `signalMu`-guarded, so `-race` reports nothing (a scheduling-dependent ordering defect, not a data race).

**Investigation before forcing anything**: traced every exit path in `Run`. The post-shutdown early return (`harness.go:346-350`) explicitly `close(sink)`s itself and returns BEFORE either defer is registered (registration happens later, at what are now lines 372/374) — confirmed the reorder cannot double-close it. `defer h.queue.close()` (unaffected, left in its original registration position, still LIFO-executes first both before and after the fix, since it was already registered after both other defers) has no coupling to `sink`/`cancelRun`. No unsafe interaction found — proceeded with the fix.

**Fix** (`harness.go`): swapped registration order — `defer close(sink)` now registers BEFORE the `cancelRun`-clearing defer, so LIFO clears `cancelRun` FIRST and closes `sink` SECOND. A stream-only consumer starting run #2 from the sink-close observation can therefore never observe the close before `cancelRun` has already been cleared. Added a doc comment at the swap explaining why the order is load-bearing (this exact class of regression is otherwise invisible to `-race`). Net diff: +13/-2 lines (comment + reorder).

**RED discipline — empirical calibration (recorded honestly, including what did NOT work)**: this is a genuine scheduling race, not a deterministic ordering bug — run #1's remaining cleanup (a mutex lock, a nil-assignment, an unlock, an already-cancelled `cancel(nil)` no-op) is structurally far cheaper than run #2's path back into contention (wake, get scheduled, run all of `Run`'s own preamble including a heap allocation for the new `context.WithCancelCause`). Naive constructions, tried first, did NOT reproduce it:
- Tight non-blocking spin-poll of `sink1`'s closure, launching run #2 in a fresh goroutine on detection: **0/500 and separately 0/20000 attempts**.
- Pre-spawned, already-running run #2 launcher goroutine (removing goroutine-creation latency from the critical path): **0/20000 attempts**.
- Widening `close(sink1)`'s own synchronous wake-loop alone, at default `GOMAXPROCS`, by parking up to 100,000 extra "noise" receivers ahead of nothing: **0/20 and 0/5 attempts**, even at 100,000 noise waiters.

Reproducing it reliably required TWO techniques together, calibrated empirically before being written into the real test:
1. `runtime.GOMAXPROCS` temporarily lowered (tried 3, then 2) so run #1's goroutine has real scheduling competition instead of an idle core it runs straight through uninterrupted.
2. A dedicated, front-of-queue real observer parked FIRST on `sink1`, with many (2000) extra "noise" receivers parked immediately after it. `sink1` is provably empty until `Interrupt()` fires (`heldTurnScript`'s `Hold` is the producer's first script step), so every one of these calls is guaranteed to park in `sink1`'s FIFO `recvq` rather than race a real event — widening `close(sink1)`'s own wake loop, which runs synchronously on run #1's own goroutine, before it can reach its next deferred call.

Calibration results against the THEN-CURRENT buggy order (`GOMAXPROCS=2`, 2000 noise waiters, 300 attempts per run, three separate runs): 2/300, 3/300, 6/300 — small but consistently non-zero. Against a temporarily-applied fix (identical harness and technique): 0/300 three times running (900/900 zero). This confirmed the technique discriminates correctly BEFORE it was written into the real, permanent test.

**Real test**: `cancellation_interrupt_test.go`, new `TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration` (2000-attempt budget, fails fast via `t.Fatalf` on the first attempt that reproduces the bug; deliberately NOT `t.Parallel()` since it mutates process-wide `GOMAXPROCS` for its own duration and must not race unrelated tests). No sleep anywhere — synchronized entirely on channel reads (`sink1`/`sink2`/gates) plus one bounded `select`+`time.After(500ms)` deadline guard for detecting "the interrupt was silently swallowed" without hanging forever, mirroring this same package's own `drainSink`/`readUntilDecisionRequired` idiom (`time.After` as a deadline guard, not a pacing sleep).

**RED — real captured output, against the (then still) buggy defer order**, 3 separate runs:

```
cancellation_interrupt_test.go:720: attempt 90: run #2 did not end within 500ms of Interrupt() — the interrupt was silently swallowed, run #1's still-pending cancelRun-clearing defer clobbered run #2's registration
--- FAIL: TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration (0.55s)
```
```
cancellation_interrupt_test.go:720: attempt 26: run #2 did not end within 500ms of Interrupt() — ...
--- FAIL: TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration (0.52s)
```
```
cancellation_interrupt_test.go:720: attempt 81: run #2 did not end within 500ms of Interrupt() — ...
--- FAIL: TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration (0.54s)
```

All three failures reproduce the EXACT predicted failure mode (Interrupt() silently doing nothing on run #2), well inside the 2000-attempt budget (hit at attempt 26-90 each time).

**GREEN — after the one-line-plus-comment fix**, 3 plain runs plus 1 run under `-race`, all PASS:

```
--- PASS: TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration (0.79s)
--- PASS: TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration (0.80s)
--- PASS: TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration (0.80s)
--- PASS: TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration (4.26s)   [-race]
```

### Substrate/pin compliance (re-verified at the end of this remediation pass)

- Files touched this pass: `cancellation_interrupt_test.go` (MAJOR-3 assertions + the new MINOR-3 test), `cancellation_shutdown_test.go` (MAJOR-3 mirrored assertions), `harness.go` (MINOR-3 fix). `git diff --stat`: **180 insertions(+), 2 deletions(-)** across the three files.
- `loop.go`: byte-identical (`git diff` empty) — the MAJOR-3 RED mutation was fully reverted before this pass's real test-writing began, and re-confirmed empty at the end.
- `permission_protocol_test.go` and every file on `R-LSK-004`'s forbidden list (`turn_events.go`, `failure.go`, `stream_check.go`, `stream_check_test.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `reconstruction_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `history.go`, `go.mod`, `go.sum`): all byte-identical (`git diff` empty for every one, checked together in one command).
- `S-LSK-018` pin: diffing the full branch against `origin/main` (merge-base `52701436`), the pre-existing non-test `.go` files that differ under `src/agent/` are EXACTLY `{run_events.go, doc.go, harness.go, loop.go, scheduler.go, tool.go}` — the six pinned files, no seventh — plus the one already-known new file `cancellation.go`. No new test file was added this pass; both edits extended existing, already-substrate-listed test files.
- `make fmt`/`gofmt -w`: never run.
- A throwaway scratch prototype file (`zzz_scratch_race_test.go`) was used to calibrate the MINOR-3 racing technique outside the deliverable; deleted before the real test was written, confirmed absent from `git status` (no untracked files).

### Closing gates (this remediation pass, real output)

- **`make test`** equivalent (`go clean -testcache && go test -race -v -count=1 ./...`): exit 0, all 12 module packages `ok` (`ai/openaicompat` alone accounts for ~175s of the ~2m56s wall-clock total — pre-existing, unrelated to this pass). Verbose count across the WHOLE suite: **3267 `--- PASS`** (3266 baseline + 1 new top-level test — MAJOR-3 only added assertions to existing subtests, contributing no new count) / **0 `--- FAIL`** / **0 `DATA RACE`**. The `src/agent` package alone, isolated: **383 `--- PASS`** / **0 `--- FAIL`**, exit 0.
- **`golangci-lint cache clean && make lint`**: `go vet ./...` clean, `golangci-lint run --config=.golangci.yml ./...` → `0 issues.`
- **`make build`** (`go build -trimpath ./...`): exit 0, no output.
- **`make vuln-check`** (`govulncheck -json ./...`, `jq`-parsed): exit 0; 170 `osv` entries scanned (module's dependency set drifted slightly since the original apply's ~180-entry count — expected over time, not itself a finding); **0** entries carry any populated `finding.trace` (i.e., zero reachable vulnerabilities).

### Status

Both findings closed, scope held to exactly the two named items — no other code, spec, or task content touched. `openspec/changes/cachicamas-agent-cancellation-tree/tasks.md` unchanged (this was a post-verify remediation pass, not a tasks-phase batch; all 57 tasks' `[x]`/`[ ]` state from batch C stands as-is, 56/57 with only 13.5 reserved for archive).
