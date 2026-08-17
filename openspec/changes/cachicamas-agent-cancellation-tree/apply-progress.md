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
