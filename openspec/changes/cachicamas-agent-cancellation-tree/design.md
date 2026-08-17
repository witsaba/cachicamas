# Design: AG-14 — Build the cancellation tree

> Change `cachicamas-agent-cancellation-tree` · Closes proposal Decisions 1, 2, 3 (all three sub-decisions) and the `L2C-08` question; re-litigates nothing else.
> New capability `agent-cancellation-tree` (prefix `CAN`). Production code: new `backend/agent/src/agent/cancellation.go` plus `harness.go`, `loop.go`, `scheduler.go`, `tool.go`, and — under this design's recorded substrate release — `run_events.go`, `doc.go`, `doc_contract_guard_test.go`.

## Technical Approach

`context.WithCancelCause` (stdlib, Go 1.20+; satisfies the R-01 import guard) is the propagation mechanism, exactly as the exploration recommended. `Run` derives the run's own context once at entry from the caller's `ctx` and holds the `context.CancelCauseFunc` behind a harness mutex. `Interrupt()` and `Shutdown()` sit beside `Steer` (`harness.go:149-151`) on the same non-privileged upward path (R-RUN-006): they flip a context the loop already observes, invoking the cancel func with a typed sentinel (`ErrInterrupted` / `ErrShutdown`). Every site that today reads `ctx.Err()` reads `context.Cause(ctx)` instead. Wind-down is: orphan synthesis (`History.SynthesizeOrphans`, `history.go:270-300`, first production caller), `CloseTurn`, then `run_end` with the signal's own outcome and no `*Failure`. Zero new parameters on `Turn`, `Schedule`, `executeCall` or `tool.Run` — each already receives a `ctx`.

**Scope line, recorded:** only the two harness sentinels get typed outcomes. A bare caller-context cancellation (a `ctx` the *caller* cancels without going through `Interrupt`/`Shutdown`) keeps AG-13's routing (`RunOutcomeFailed`/`Unavailable`) — the charter's third noun, deadline, is explicitly "not this milestone" (`0003:1373`), and R-RUN-010's path (b) stays intact for external cancellation.

## Decision 1 — substrate release: **Option B adopted**, new member `RunOutcomeShutdown`

Option A fails the charter as written: "Both distinguishable in run-end **outcomes and error chains**" (`0003:1379`) and AG-14.2's "the run-end outcome **says shutdown**" (`0003:1423`) require a stream-level discriminator a Layer-3 consumer can read without a Go error. Option C is rejected: loosening `RunEnd.validate` re-litigates AG-04's frozen vocabulary for a payload nobody needs — the discriminator rides the outcome member itself.

`run_events.go` has **never been released before** — AG-11's release covered only `turn_events.go` and `failure.go` (`agent-loop-skeleton/spec.md:86`) — so the structural argument stands on its own, verified against the file: the `iota` block (`run_events.go:100-118`), the `runOutcomeLimit` bound check inside `RunEnd.validate` (`:153`) and the `String()` switch (`:123-136`) are all local to `run_events.go`; a member declared elsewhere is rejected as `ai.ErrNotInVocabulary` and renders as the `runoutcome(N)` placeholder. Inserting `RunOutcomeShutdown` between `RunOutcomeFailed` (`:113`) and `runOutcomeLimit` (`:117`) leaves `Completed`/`Interrupted`/`Failed` at 1/2/3. **No `validate` change**: the failure-iff-`Failed` rule (`:156-161`) forbids a `*Failure` on every non-`Failed` outcome; both `Interrupted` and `Shutdown` are non-`Failed`, so both run-ends carry `nil` failure and pass unchanged. `NewRunEnd`'s signature (`:172`) is untouched.

**The release paragraph `sdd-spec` MUST place into `R-LSK-004`, verbatim** (widened to three files by the `L2C-08` decision below):

> **AG-14's release scope, recorded here rather than assumed:** `run_events.go`, `doc.go` and `doc_contract_guard_test.go` are released **for AG-14 only**. `run_events.go`, never before released, opens for a structural reason recorded rather than assumed: `RunOutcome`'s `iota` const block (`run_events.go:100-118`), its `runOutcomeLimit` bound-check inside `RunEnd.validate` (`:153`) and its `String()` switch (`:123-136`) are all local to `run_events.go`, so a member declared elsewhere would be rejected by the validator and would render as the `runoutcome(N)` placeholder. AG-14's edit to `run_events.go` MUST be confined to exactly one new member, `RunOutcomeShutdown`, inserted between `RunOutcomeFailed` and `runOutcomeLimit` so `RunOutcomeCompleted`/`RunOutcomeInterrupted`/`RunOutcomeFailed` keep the values 1/2/3, plus its `String()` case `"shutdown"`. `RunEnd.validate`'s failure-iff-`Failed` rule (`:156-161`) and `NewRunEnd`'s signature MUST NOT change — both `RunOutcomeInterrupted` and `RunOutcomeShutdown` are non-`Failed` outcomes carrying no `*Failure`; the interrupt-vs-shutdown discriminator is the outcome member itself. `doc.go` and `doc_contract_guard_test.go` open for exactly one `L2C-08` row and its matching `expectedLayer2ContractRows` entry, landing in the same pull request per the guard's own rule (`doc_contract_guard_test.go:19-22`). `turn_events.go`, `failure.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go` and `reconstruction_test.go` stay byte-unchanged, every other file named forbidden above remains forbidden, and this release does not extend to any milestone after AG-14 without its own recorded delta.

**`CheckStream`/`String()` coverage without touching `stream_check.go`:** `CheckStream` delegates outcome membership to `RunEnd.validate`'s own bound check, which admits the new member automatically once `runOutcomeLimit` moves — `stream_check.go` stays byte-unchanged (its test at `stream_check_test.go:147-153` already proves the validator side accepts `RunOutcomeInterrupted`, and `stream_check_test.go` is not in the substrate filters, so it must NOT be edited). Coverage for the new member lives in AG-14's own `cancellation_events_test.go`: `String() == "shutdown"`, `NewRunEnd(run, RunOutcomeShutdown, nil)` constructs, `NewRunEnd(run, RunOutcomeShutdown, failure)` is rejected `ai.ErrMisplaced`, and `CheckStream` accepts a full run ending `run_end(shutdown)` unmodified.

## Decision 2 — `tool.Run`'s context source: **thread the real run `ctx` (Option A adopted)**

`executeCall` passes `context.Background()` today (`scheduler.go:462`). The charter demands propagation "down through loop, provider **and tools**" (`0003:1377`) and AG-14.1's first Then ("in-flight tools observe cancellation", `0003:1399`) is unreachable by construction under B. It also makes AG-14.3 non-vacuous: a cancellation-deaf tool is only a meaningful special fixture when the ordinary tool is cancellation-aware.

**The behavioral consequence, in writing:** a tool that reads its `ctx` now returns early on cancellation where it previously ran to completion — that is the intent. **R-TLS-010's disjoint-return-channel rule stays true unchanged**: a ctx-observing tool returns a non-nil `err`, which routes through the existing `runErr != nil` branch (`scheduler.go:469-473`) into `Result{Outcome: ExecutionFailure, Failure: non-nil}` with the tool's own `Result` discarded — no site ever returns both channels populated. Cancellation adds callers to an existing branch; it adds no new return shape. Blast radius re-verified: the `context.Background()` occurrences in the test closure are callers passing it *into* `Schedule`/`Turn` (`scheduler_test.go:536`, `permission_protocol_test.go:1599`) and direct `tool.Run` unit calls bypassing the scheduler — none asserts the identity of `executeCall`'s context. `sdd-tasks` re-runs that grep against the final tree before apply (proposal binding).

## Decision 3 — the bound: **armed-by-cancellation (3B semantics), implemented per-call inside `executeCall`, not around `wg.Wait()`**

**Mechanism (a conscious refinement of 3B's letter, same contract):** `executeCall` runs `tool.Run(ctx, …)` in an inner goroutine sending its `(Result, error)` on a **buffered capacity-1 channel**; the harness-owned call goroutine then selects:

```go
resCh := make(chan toolRunReply, 1)
go func() {
    defer func() { if r := recover(); r != nil { resCh <- toolRunReply{panicVal: r} } }()
    res, err := tool.Run(ctx, runArgs, PolicySlot(call.ID()))
    resCh <- toolRunReply{res: res, err: err}
}()
select {
case reply := <-resCh:            // uncancelled path: the ONLY arm that can fire
    // existing disjoint-channel handling; a panicVal re-panics into
    // the already-deferred recoverCall (scheduler.go:410) unchanged
case <-ctx.Done():                // bound armed ONLY here
    timer := time.NewTimer(bound) // the one legitimate clock use
    select {
    case reply := <-resCh:        // tool honored cancellation within the bound
    case <-timer.C:               // overrun: typed DetachedCallError report
    }
}
```

Why not a bound directly on `wg.Wait()` (`scheduler.go:205`): abandoning the join races the still-running call goroutine's later writes — `results[ordinal]` (data race under `-race`) and `emissions <-` after `close(emissions)` (`scheduler.go:226`; panic: send on closed channel). Per-call detach keeps **every** writer of `results`/`emissions` a harness-owned goroutine that provably exits within the bound; the only detached frame is the tool's own `tool.Run`, which communicates through a buffered channel it can always complete its send on and then exits whenever the third-party code returns — even a post-detach panic is recovered inside the inner goroutine and parked harmlessly on the buffered channel, never crashing the process. `wg.Wait()`, D-B's comment (`scheduler.go:206-216`), the dispatcher lifecycle and the parked-set clearing all stay **byte-unchanged**. On the uncancelled path the select blocks on `resCh` alone — no timer is ever created, so `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` (`permission_protocol_test.go:1573-1600`, driving `Schedule` with bare `context.Background()`) passes **file-unchanged**, and R-RUN-010's "no third path and no timeout" (`agent-run-driver/spec.md:200`) stays literally true for the non-cancelled case: the bound is part of wind-down, R-RUN-010's own path (b). Panic containment (R-TLS-011/S-TLS-011) is preserved by re-panicking a received `panicVal` in the call goroutine so the existing `defer recoverCall(ordinal, results)` (`scheduler.go:410`) handles it identically.

**3.1 — value and configurability:** a new zero-default field **`Scheduler.WindDownBound time.Duration`** (on `tool.go`'s `Scheduler` struct — the `LeaveSinkOpen` precedent, a conscious addition to the proposal's Affected Areas exactly as AG-13's design added `tool.go`), resolving to package constant **`defaultWindDownBound = 100 * time.Millisecond`** (declared in `cancellation.go`, named in the spec — that is the "documented bound"). Why a field, not a constant alone: the leak test repeats its scenario `leakRepeats = 50` times (`stream_kit_leak.go:61`); at a fixed 100 ms it would spend 5 s in bounds alone. Tests inject 20–50 ms through the caller-owned scheduler AG-13's continuation seam already provides — no new plumbing, no harness field, and no second caller-field mutation (R-RUN-001:68 permits exactly one, `LeaveSinkOpen`). Why 100 ms: an order of magnitude above goroutine-scheduling jitter under `-race` (agenttest's own settle window is 50 ms, `stream_kit_leak.go:77`), and armed only after cancellation, so a healthy run never pays it. `time` is not in the ambient-authority guard's forbidden set (`ambient_authority_test.go:73-94`: `os`, process-execution, `syscall`, legacy I/O only) — verified before committing to a timer in production code.

**3.2 — the typed report's carrier: the existing execution-failure path; no new event kind.** The overrun call's ordinal slot gets `Result{Outcome: ToolOutcomeExecutionFailure, Failure: typed}` whose `*Failure` is built with `ai.FailureCategoryCancellation` and cause **`*DetachedCallError{Tool: call.Name(), CallID: call.ID()}`** (new exported error type in `cancellation.go`; message states the call is still running past the wind-down bound; `errors.As`-extractable through `Failure`'s preserved unwrap chain, `scheduler.go:936-937`). On the stream it is the existing `tool_end_execution_failure` kind via `emitExecutionFailure` — `event.go`/`event_registry_test.go` untouched, exactly as Decision 1's release requires. A new helper `typedCancellationFailureFromError` (sibling of `typedFailureFromError`, `scheduler.go:938`, which hardcodes `Unavailable`) builds the cancellation-category wrap; cancellation-caused tool errors and parked aborts route through it, other tool errors keep `Unavailable`.

**3.3 — "detached and named", the scope distinction written down:** *the harness's own tasks* are `runDispatcher` (`scheduler.go:267`), the per-turn forwarder (`harness.go:274-279`), every `scheduleRead`/`scheduleSerialized` call goroutine, and `Run`'s control flow — all MUST exit within the wind-down, and do: call goroutines exit at the bound, `wg.Wait()` returns, `emissions` closes, the dispatcher exits, `Turn` closes `turnSink`, the forwarder exits. *The one surviving goroutine* is the tool's own `tool.Run` frame — third-party code whose lifetime the harness cannot end (Go has no goroutine-kill primitive). It is **named** by the typed report (tool name + call ID + still-running), not by any runtime mechanism. AG-14.3's second Then quantifies over that tool-owned frame; its third Then quantifies over harness-owned tasks only — the two Thens are simultaneously true because they range over disjoint sets, and the spec states this rather than leaving it implicit.

## Signal vocabulary and harness state (`cancellation.go` + `harness.go`)

```go
// cancellation.go — new production file
var ErrInterrupted = errors.New("agent: run interrupted")           // errors.Is sentinel
var ErrShutdown    = errors.New("agent: harness shut down")          // errors.Is sentinel
var ErrPromptAfterShutdown error  // wraps ErrShutdown; Run's post-shutdown typed refusal
type DetachedCallError struct{ Tool, CallID string }                 // errors.As carrier
const defaultWindDownBound = 100 * time.Millisecond
```

**Idempotency mechanism: one `sync.Mutex` (`signalMu`) guarding `{cancelRun context.CancelCauseFunc; shutdown bool}` — not `sync.Once`, not atomics.** `sync.Once` is per-object one-shot and cannot reset — wrong for a harness that must accept a *new* run (with a fresh cancel registration) after an interrupt (AG-14.1's "new prompt works afterward"). Atomics cannot couple "read the flag" with "invoke the cancel func" atomically. Under the mutex, a second `Interrupt()`/`Shutdown()` during the first's cleanup observes either a live cancel func — calling it is a **documented Go no-op on an already-cancelled context** (first cause wins; `context.Cause` keeps returning it) — or `nil` (between runs): no path panics, and no channel is ever closed by a signal, so there is no double-close to race. `Run` entry (under `signalMu`): if `shutdown` → return `ErrPromptAfterShutdown` (typed, `errors.Is(err, ErrShutdown)`), emit nothing, close the sink; else derive `runCtx, cancel := context.WithCancelCause(ctx)`, store `cancelRun = cancel`, **reopen the steering queue** (see the R-RUN-001 delta below). `Run` exit: clear `cancelRun` under the mutex, then `cancel(nil)` deferred. `Shutdown()` sets the one-way `shutdown` flag and cancels; `Interrupt()` only cancels. Shutdown racing an interrupt's wind-down: the first cause wins on the context (outcome reflects the first signal), the flag still latches, so the harness refuses afterwards — stated, not accidental.

**AG-21 boundary (proposal risk 7), stated:** the shutdown flag is *terminal and one-way*, never resumes a run, and holds no transcript; the queue reopen is per-run bookkeeping. Neither pre-empts AG-21's cross-run state. The authoritative reservation is `agent-run-driver/spec.md:72` ("One `Run` per harness value. Cross-run state is **AG-21**'s") — **not** `agent-loop-skeleton/spec.md:106`, which is R-LSK-005 coverage text; the proposal's risk row 7 citation is wrong, and `agent-run-driver/spec.md:275` carries the same stale pointer as a pre-existing defect AG-14's delta MAY back-annotate but MUST NOT propagate.

## Cancellation-aware routing changes, site by site

| Site | Today | Under AG-14 |
|---|---|---|
| `loop.go:417-427` closed-channel branch | bare close normalizes to `FinishReasonStop` — an interrupt reads as success | before the `turn.finish == 0` normalization: if `cause := context.Cause(ctx)` matches `ErrInterrupted`/`ErrShutdown`, return a typed error carrying the cause instead of finalizing — `Turn`'s first-ever cancellation check |
| `scheduler.go:639`, `:665` parked-wait aborts | `typedExecutionFailureFromError(call.ID(), ctx.Err())` — raw `context.Canceled`, category `Unavailable` | `typedCancellationFailureFromError(call.ID(), context.Cause(ctx))` — the typed value names *which signal*; `parked.remove` unchanged |
| `scheduler.go:462` | `tool.Run(context.Background(), …)` | `tool.Run(ctx, …)` inside the detach select (Decisions 2+3) |
| `harness.go:200-207` `failRun` | every `Turn` error → `RunOutcomeFailed`/`Unavailable` | unchanged for genuine failures; a `Turn` error (or iteration-boundary check) whose cause matches a sentinel routes to a **new `windDownRun` path** instead |
| `harness.go` run loop | no cancellation check between turns | at every iteration boundary, `context.Cause(runCtx)` is consulted; a pending signal triggers wind-down rather than another `Turn` (no futile provider call on a cancelled context) |

**`windDownRun`** (new, beside `failRun`): (1) `hist.SynthesizeOrphans()` — first production caller; idempotent, so a turn whose results were already appended synthesizes zero; (2) `hist.CloseTurn()`; (3) emit `run_end(RunOutcomeInterrupted | RunOutcomeShutdown, nil)` via the public constructor; (4) return an error satisfying `errors.Is` on the matching sentinel. This requires the **R-RUN-011 carve-out delta**: on the cancellation path the harness MAY synthesize orphans and MUST `CloseTurn` — the "no append, no close" rule is re-scoped to genuine failures; the no-retry rule extends verbatim to cancellations.

## Data Flow

    Interrupt()/Shutdown() ──signalMu──▶ cancelRun(sentinel)      Shutdown also latches the one-way flag
                                             │ context.WithCancelCause (derived once at Run entry)
              ┌──────────────────────────────┼──────────────────────────────┐
              ▼                              ▼                              ▼
      provider fake closes bare      tool.Run(ctx) returns err       parked-wait arms fire
      loop.go:417 cause check        (or stays deaf → armed bound    scheduler.go:639/:665
      → typed Turn error              → DetachedCallError report)    context.Cause → typed abort
              └──────────────────────────────┼──────────────────────────────┘
                                             ▼
        Run: windDownRun → SynthesizeOrphans → CloseTurn → run_end(interrupted|shutdown, nil)
                          → Run returns err, errors.Is-matchable on the sentinel

## Cross-cutting — `L2C-08`: **YES, a new row**

AG-12's discriminator, applied fresh: does AG-14 declare a new package-wide guarantee, or implement behavior inside one already declared? AG-13's bracketing stayed inside `CheckStream`'s existing state machine — no row. AG-14 does not: bounded wind-down and two-signal distinguishability are a **liveness and control guarantee no existing row covers** — `L2C-03`/`L2C-04` govern the upward stream, `L2C-07` the transcript; nothing governs downward control or goroutine lifetime. The guarantee binds every *future* tool and turn family (AG-19's subagents inherit it, AG-21's leak sweep depends on the report being precise — `0003:1439`'s own words), which is exactly the overturn criterion AG-13's design recorded for adding a row. Consequence, taken: Decision 1's release names `doc.go` and `doc_contract_guard_test.go`; the row and its `expectedLayer2ContractRows` entry land in the same PR (`doc_contract_guard_test.go:19-22`). Draft row (grammar `^//\tL2C-\d\d\t`, `sdd-spec` may refine wording, not scope):

> `L2C-08` Cancellation is a bounded, typed, two-signal tree: interrupt aborts the run and keeps the harness; shutdown aborts the run and terminally refuses new prompts; both propagate down through loop, provider and tools as one context cancellation cause, stay `errors.Is`-distinguishable in the error chain and distinct in the run-end outcome; after the documented wind-down bound only third-party tool code may remain running — reported typed by tool and call identity — and every goroutine the package itself owns has exited (doc 0003 AG-14 acceptance; agent-cancellation-tree).

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/cancellation.go` | Create | Sentinels, `ErrPromptAfterShutdown`, `DetachedCallError`, `defaultWindDownBound` |
| `backend/agent/src/agent/harness.go` | Modify | `Interrupt`/`Shutdown`, `signalMu` state, cancel-cause derivation, queue reopen, `windDownRun`, iteration-boundary cause check, post-shutdown refusal |
| `backend/agent/src/agent/loop.go` | Modify | Closed-channel-branch cause check (`:417-427`) |
| `backend/agent/src/agent/scheduler.go` | Modify | `executeCall` detach + armed bound; `ctx` into `tool.Run`; cause-aware parked aborts; `typedCancellationFailureFromError` |
| `backend/agent/src/agent/tool.go` | Modify | `Scheduler.WindDownBound time.Duration` (zero-default; conscious addition, `LeaveSinkOpen` precedent) |
| `backend/agent/src/agent/run_events.go` | Modify — **released substrate** | `RunOutcomeShutdown` + `String()` case only |
| `backend/agent/src/agent/doc.go` | Modify — **released substrate** | `L2C-08` row |
| `backend/agent/src/agent/doc_contract_guard_test.go` | Modify — **released substrate** | Matching `expectedLayer2ContractRows` entry, same PR |
| `cancellation_interrupt_test.go`, `cancellation_shutdown_test.go`, `cancellation_winddown_test.go`, `cancellation_events_test.go` | Create | `package agent_test`; the wind-down file is serial-only |
| `backend/agent/src/agent/harness_test.go` | Modify | The `"exactly two exported methods"` subtest (`harness_test.go:1018-1024`) widens to the four-method set — a **conscious, delta-backed** edit (R-RUN-001/S-RUN-001), found by this design, not named in the proposal |
| `loop_test.go` / `loop_hook_test.go` | Modify | Filter widening below, byte-in-sync |
| Five spec deltas + doc 0003 `:2171`/`:2207`/`:2257` | Delta | Per proposal, plus the R-RUN-001 items below |

**Substrate-filter widening** (`filterOutLoopFiles` `loop_test.go:831`, `filterOutLoopHookFiles` `loop_hook_test.go:907` — exact filename suffixes, no wildcard/prefix/directory pattern, byte-in-sync): add `/run_events.go`, `/cancellation.go`, `/cancellation_interrupt_test.go`, `/cancellation_shutdown_test.go`, `/cancellation_winddown_test.go`, `/cancellation_events_test.go`. `doc.go`, `doc_contract_guard_test.go`, `harness_test.go`, `loop.go`, `scheduler.go`, `tool.go` are already listed. Every other `R-LSK-004` file stays byte-unchanged; `go.mod`/`go.sum` unchanged; no `ai/**` edit; no new exported `History` method (`history_surface_guard_test.go` green untouched).

**`agent-run-driver` delta items this design adds to the proposal's list (R-RUN-001):** (a) the public surface widens from two methods to four (`Run`, `Steer`, `Interrupt`, `Shutdown`) — S-RUN-001's "no third exported method" wording re-scoped, not silently broken; (b) "One `Run` per harness value" re-scoped to *one run at a time; a run that ended interrupted or completed may be followed by another `Run` on the same value (AG-14.1), the steering queue reopening at `Run` entry; concurrent runs stay out of scope and cross-run transcript state beyond the terminal shutdown flag is AG-21's*; (c) `:275`'s stale `agent-loop-skeleton/spec.md:106` pointer recorded as pre-existing, back-annotated to `:72`.

**`agenttest/tracetest` (`backend/agent/src/agenttest/tracetest/tracetest.go`), read directly as required: AG-14 does not use it.** It is a hand-rolled recording OTel `TracerProvider`/`Tracer`/`Span` built for Layer 1's AI-37 observability proofs (span lifecycle, attribute mapping, denylist absence — its package comment, `:1-47`). AG-14 makes no span-shaped assertion; its observability surface is the event stream and the error chain. Recorded either-way per the brief.

## Architecture Decisions (summary)

| Decision | Rejected | Rationale |
|---|---|---|
| `RunOutcomeShutdown` member (release B) | A: error-chain only; C: loosen `validate` | Charter's plural "outcomes and error chains"; validator locality; no payload rule change |
| Real `ctx` into `tool.Run` | Defer; bound-only protection | AG-14.1 Then unreachable under B; R-TLS-010 disjointness preserved by the existing `runErr` branch |
| Per-call detach select, bound armed on `ctx.Done()` | Bound on `wg.Wait()`; harness-level timer (3C) | `wg.Wait()` abandonment races `results` writes and panics on post-close `emissions` sends; 3C leaks dispatcher+forwarder — contradicts AG-14.3 Then 3 |
| `Scheduler.WindDownBound`, default const 100 ms | Constant only; `Harness` field | Leak test needs a small injected bound; no second caller-field mutation on `Harness` (R-RUN-001:68) |
| Report via existing execution-failure path + `DetachedCallError` cause | New event kind; new `Result` outcome | `event.go`/`event_registry_test.go` are unreleased substrate; `errors.As` gives tool+callID typed |
| One `signalMu` mutex | `sync.Once`; atomics | Once cannot re-register a second run's cancel func; atomics cannot couple flag-read with cancel-invoke; cancel-func itself is idempotent per Go docs |
| Sentinels + `context.Cause` | Second channel/context threaded alongside `ctx` | Zero signature changes; AG-13's no-signature-change precedent |

## Testing Strategy (strict TDD, `cd backend/agent && make test`, all `package agent_test`; every behavior RED-recorded before GREEN; sync by `agenttest.Gate`/channel reads — the bound is the only clock use)

| # | Test (file) | Scenario → ID | Sync / assertions |
|---|---|---|---|
| 1 | `TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized` (`cancellation_interrupt_test.go`) | AG-14.1 sc.1 → **S-CAN-001** | Provider `Hold` gate mid-stream + ctx-aware blocking tool from a prior turn; `Interrupt()` at `Reached`; fake closes bare per R-CNF-011/012; assert `run_end(interrupted)` with no `*Failure`, `errors.Is(err, ErrInterrupted)`, never `Failed`/`Unavailable`, synthesized-origin entries close every open call, `CheckStream` accepts unmodified |
| 2 | `TestHarness_Interrupt_SameHarnessRunsNextPrompt` (same file) | AG-14.1 sc.1 And → **S-CAN-002** | After S-CAN-001's wind-down, a second `Run` on the same value completes `run_end(completed)`; steering works in run 2 (queue reopened) |
| 3 | `TestHarness_Interrupt_DuringSuspensionAbortsTyped` (same file) | AG-14.1 sc.2 → **S-CAN-003** | Defer policy; read `decision_required` off the stream (the stream is the sync); `Interrupt()`; parked abort carries `FailureCategoryCancellation` and `errors.Is(…, ErrInterrupted)` through the failure's unwrap chain (`scheduler.go:639`/`:665` → `context.Cause`); history closes cleanly via `SynthesizeOrphans`+`CloseTurn` |
| 4 | `TestHarness_Interrupt_SecondInterruptIsNoOp` (same file) | AG-14.1 sc.3 → **S-CAN-004** | Second `Interrupt()` fired from a separate goroutine concurrent with the first's wind-down, under `-race`; identical stream, no panic, no double close (no channel is signal-closed by design) |
| 5 | `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts` (`cancellation_shutdown_test.go`) | AG-14.2 → **S-CAN-005** | Same wind-down shape; `run_end(shutdown)` — a value distinct from interrupted and failed; `errors.Is(err, ErrShutdown)` and NOT `ErrInterrupted`; a second `Run` returns `ErrPromptAfterShutdown` (`errors.Is` → `ErrShutdown`), emits nothing |
| 6 | `TestHarness_WindDown_DeafToolCannotHoldRunHostage` (`cancellation_winddown_test.go`) | AG-14.3 → **S-CAN-006** | `BlockingScriptedTool` (`scripted_tool_test.go:70-83`) with a never-closed `release`; injected scheduler with small `WindDownBound`; `Interrupt()`; run *returns* (completion observed by channel, not a wall-clock assertion); stream carries `tool_end_execution_failure` whose failure `errors.As`-extracts `DetachedCallError{Tool, CallID}`; `t.Cleanup(close(release))` |
| 7 | `TestHarness_WindDown_NoHarnessGoroutineRemains` (same file, **serial-only — no `t.Parallel()`**, enforced by `RequireNoGoroutineLeak`'s `tb.Setenv` sentinel, `stream_kit_leak.go:110`) | AG-14.3 Then 3 → part of **S-CAN-006** | `RequireNoGoroutineLeak(t, scenario)` where each iteration runs the deaf-tool interrupt run *then closes that iteration's `release`* — the detached goroutine is thereby **accounted for** (proven alive past wind-down by the typed report, proven exiting once third-party code returns), not merely excluded; small injected bound keeps 50 repeats fast |
| 8 | `TestRunOutcomeShutdown_VocabularyAndStreamAcceptance` (`cancellation_events_test.go`) | Decision 1 coverage → **S-CAN-007** | `String()=="shutdown"`; constructor accepts `(shutdown, nil)`, rejects `(shutdown, failure)` typed; `CheckStream` accepts a `run_end(shutdown)` stream with `stream_check.go` byte-unchanged |
| Bites | scratch → RED-record → revert | **S-CAN-010**: delete `loop.go`'s cause check → interrupt recorded as a normal completed turn (test 1 fails). **S-CAN-011**: revert `failRun`/`windDownRun` routing → cancellation reappears failed/unavailable (tests 1/5 fail). **S-CAN-012**: remove the armed bound → test 6 hangs; RED evidenced via `go test -timeout` failure output, then reverted | Per NFR-APP-002's repeated-run discipline where raciness applies |
| Guards | — | `TestPermission_WakeParked_..._NoDeadline` and every AG-09/AG-10 scheduler test and `TestTurn_TwoSequentialTurnsShareNothing` pass **file-unchanged** (`sdd-verify` asserts file identity, not just green); both filters byte-in-sync; import + ambient-authority guards zero-change over both closures; coverage gate on `loop.go` ≥ 80% including the new branch |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Cancellation is in-memory control flow under `L2C-02`.

## Migration / Rollout

No migration. Rollback per proposal — single revert; every seam is additive and zero-default (`WindDownBound`, the new outcome member constructed by no external consumer, the sentinels). One addition to the proposal's rollback inventory: `tool.go` returns to its AG-13 field set, and `harness_test.go`'s method-count subtest returns to two.

## Open Questions

None blocking. Forwarded obligations: (1) `sdd-spec` places the release paragraph verbatim and carries the R-RUN-001 delta items (four methods; serial-reuse re-scope; the `:275` stale-pointer back-annotation to `:72` — do not propagate `agent-loop-skeleton/spec.md:106`); (2) `sdd-tasks` re-runs the `context.Background()` identity grep against the final tree before apply; (3) `sdd-tasks` sequences the `doc.go` row + guard row + `run_events.go` member into the same commit as their first consumer, and pins the detach select + `windDownRun` + loop cause-check as their own RED-first tasks; (4) apply verifies at runtime that `NewRunEnd(run, RunOutcomeShutdown, nil)` streams through `CheckStream` before any harness wiring (expected by the bound-check argument, unproven until run).
