# Exploration — `cachicamas-agent-run-driver` (AG-13: drive the multi-turn run)

> Phase: sdd-explore · Change: cachicamas-agent-run-driver · Milestone AG-13, doc 0003 lines 1294-1370
> Depends on: AG-10 (permission protocol), AG-11 (turn termination), AG-12 (history). Blocks: AG-14, AG-15, AG-16, AG-17, AG-19, AG-20.
> Engram mirror: topic `sdd/cachicamas-agent-run-driver/explore` (observation #3162).

## Charter (doc 0003:1294-1370)

Goal: the harness runs the loop repeatedly — append user message, run turn, execute/append results (via the loop), repeat — until a terminal finish reason, emitting run lifecycle events, handling pause-resumption, and accepting queued steering input. Three leaves: AG-13.1 run to completion, AG-13.2 steering input, AG-13.3 pause resumption.

## Finding 1 — `Turn` is a self-contained one-turn *run*, not a composable one-turn primitive

`agent.Turn(ctx, provider, system, transcript, opts TurnOptions, sink chan<- *Event) (ai.Message, ai.FinishReason, error)` (`loop.go:179`):

- Mints a **fresh** `RunID` and `TurnID` on every call (`mintLoopRunID`/`mintLoopTurnID`, `loop.go:141-151`). No parameter accepts a caller-supplied identity.
- **Always** emits `NewRunStart` at the top (`loop.go:193-198`) and **always** emits `NewTurnEnd` + `NewRunEnd` in `turnAccumulator.finalize()` (`loop.go:666-681`) on every normal return path — regardless of finish reason, including `ToolCalls` and `PauseTurn`, which are not narrative-terminal.
- Uses a per-call `LaneStamper`, so each call's sequence restarts at 1.
- `TestTurn_TwoSequentialTurnsShareNothing` (`loop_test.go:960`) is an existing, green, locked-in test proving two `Turn()` calls share nothing. This is R-LSK-002, preserved through AG-08..AG-12.

`CheckStream` (`stream_check.go:92-185`) **does** support multiple sequential turn brackets inside one run bracket — `BracketRoleOpensTurn` only rejects "already open", never "opened before", and neither `turn_start` nor `turn_end` carries `CardinalityAtMostOne` (`event.go:253-257`). But `BracketRoleOpensRun` rejects a second run-start outright (`ErrDuplicate`), `run_end` is the only `Terminal: true` descriptor (`event.go:249`), and the sequence check demands one contiguous 1-based run.

**Consequence:** calling `Turn()` N times and forwarding each call's events verbatim into one consumer stream produces N run brackets and N sequence restarts, which `CheckStream` rejects on both counts. The driver cannot satisfy "one run, N turns, correct event stream" by simple concatenation.

Two existing comments already flag the gap and are in apparent tension: `loop.go:11-14` ("AG-13's later Harness can wrap it without changing the signature") and `loop.go:132` ("AG-13's Harness will mint them in caller-supplied groups"). Nothing in the codebase resolves it — this is AG-13's first design decision.

Likely resolution (for `sdd-design` to decide): extend `TurnOptions` — which already grew across AG-08/09/10 without changing `Turn`'s positional signature — with an optional caller-supplied run/turn identity plus continuation flags, and make `RunStart`/`RunEnd` emission conditional. Must be verified against `CheckStream`'s exact state machine and every existing `loop_test.go` scenario asserting unconditional `RunStart`/`RunEnd`.

## Finding 2 — a permission suspension spanning a turn has no external wake door

`Turn()` constructs `sched := &Scheduler{MaxConcurrentReads: maxReadFanOutDefault}` **locally** (`loop.go:259`) and calls `sched.Schedule(...)` synchronously inline; the `*Scheduler` pointer is never returned or exposed. `Scheduler.WakeParked(callID string) error` (`scheduler.go:235`) is the only production path that releases a parked call (`PermissionDefer` verdict) — the other release is `ctx.Done()`. The driver therefore has no way to wake a parked call while that `Turn()` invocation is blocked inside `Schedule()`.

`loop.go:260-264` already names this exact gap: "the loop's own upward-path wake wiring (`Scheduler.WakeParked`) is AG-13's scope."

Design must decide how the driver gets a live handle. Most likely: inject a caller-owned `*Scheduler` through a new `TurnOptions` field, with a nil default preserving today's behavior byte-stable.

## Finding 3 — `CheckStream` never validates RunID/TurnID *value* consistency

`CheckStream` tracks bracket **structure** only; it never reads or compares the `RunID` string carried by events (it reads `TurnID` only to match a closing turn to its open one). A driver could synthesize a coherent bracket structure whose events disagree on the RunID and the validator would not catch it.

Recommendation: do **not** exploit this. It would violate the spirit of R-AEV-002 / VL2-EVT-02 envelope identity even though the validator does not yet enforce it. Whether AG-13 strengthens `CheckStream` or defers is an explicit design question, not something to rely on silently.

## Finding 4 — History (AG-12) is clean and ready, zero surprises

`history.go` exports `NewHistory()`, `NewSeededHistory([]ai.Message) (*History, error)`, `(*History).Append(ai.Message) error`, `(*History).CloseTurn() error`, `(*History).SynthesizeOrphans() (int, error)`, `(*History).Entries() []Entry`, `(*History).Len() int`.

History's own `doc.go` states: "It does not wire into loop.go or scheduler.go — the run driver's consumption of this surface is AG-13's (NFR-HIS-003)", and `CloseTurn()`'s doc: "History detects nothing itself — AG-13's run driver will call it when a provider turn ends."

There is no RunID/TurnID coupling in History at all — it is pure `ai.Message` storage keyed by ordinal position, and consecutive same-role entries are legal (no role-alternation check in `commitAppendOp`). AG-13.2's "consecutive same-role entries are legal in the neutral transcript" is therefore already satisfied by History as built; the driver only needs to `Append` the queued user message at the turn boundary before building the next request's transcript.

## Finding 5 — pause vocabulary (AG-11) already exists; only the action is missing

`turn_events.go` defines `TurnOutcomeFinished | Aborted | LengthLimited | ToolCalls | ContentFiltered | Refused | Paused | Unknown`. `TurnOutcomePaused` (`turn_events.go:106-110`) documents: "the turn suspended and expected to resume, with the content already received replayed verbatim (`ai.FinishReasonPauseTurn`). Acting on the pause is AG-13's, not this milestone's."

"Verbatim replay" concretely means: the driver takes the `ai.Message` that `Turn()` returned — the partial assistant message already reconstructed by `turnAccumulator.finalize()`/`reconstructMessage` — and re-sends it as-is (append to History, then re-include it in the next request's transcript) rather than discarding or re-synthesizing it. `outcomeForFinish` (`loop.go:331-350`) already maps `ai.FinishReasonPauseTurn → TurnOutcomePaused` and is total and registered, so no new outcome value is needed.

## Finding 6 — permission protocol (AG-10) constrains suspension lifetime

`PermissionPolicy` is `Resolve(ctx, ai.ToolCall) PermissionVerdict` + `Remember(ctx, toolName, outcome) bool`; `PermissionVerdict{Outcome, ModifiedArgs, Failure}`; `PermissionDefer = PermissionOutcome(0)` is the zero value signalling suspension. `parkedSet` lives exactly one `Schedule` call. Any suspension must therefore resolve (wake or ctx-cancel) before that `Turn()` call returns, or `Turn()` blocks forever — which is precisely why the Scheduler-injection seam of Finding 2 is load-bearing for the acceptance criterion "the run survives a permission suspension spanning a turn".

## Test substrate

`agenttest.NewProvider(scripts ...Script) *Provider` takes **multiple** `Script` values; each successive `.Stream()` call (that is, each `Turn()` invocation) consumes the next script in order. This is exactly the two-call-then-answer mechanism: `agenttest.NewProvider(scriptToolCall, scriptFinalText)`.

`Script`/`Step`/`Emit(ev ai.Event) Step` (`fake_script.go`) and `Gate{Reached() <-chan struct{}, Release()}` (`fake_gate.go`) provide wall-clock-free synchronization for the steering-mid-turn scenarios: a hold step blocks the fake stream until the test goroutine calls `Release()` after queueing a steering message.

## Guards that will bite

| Guard | What it requires from AG-13 |
| --- | --- |
| `import_boundary_test.go` | New run-driver files may import stdlib + the `ai` package only, deny-by-default, over both production and test closures. |
| `ambient_authority_test.go` | No I/O and no wall-clock call sites in any new production file. |
| `doc_contract_guard_test.go` | AG-13 likely needs **no** new `L2C-07` row (unlike AG-12): it rides the existing `L2C-03` guarantee — "the event stream is the only upward contract… callers observe the stream, they never reach into the loop" — which *is* AG-13.1's "no privileged channel" claim extended to multi-turn. Confirm in `sdd-propose` rather than assume. |
| `history_surface_guard_test.go` | Closed-route guard, set-equal both directions. Do not add exported methods to `History` without updating `expectedHistoryRoutes`; prefer wiring through the existing methods only. |
| `event_registry_test.go` | Every-kind-constructible witness table. Only touch if a new `EventKind` is needed — current read: none is; AG-13 wires existing families. |
| `invariant_pin_test.go` | Pins `doc.go`'s membership-criterion and ordering-invariant prose byte-exact. If `doc.go` changes, these pins change with it. |
| `loop_test.go` substrate filter | Strips permitted-to-change files by exact filename suffix. AG-13 must widen the list with its new filenames — exact names, no wildcards, mirroring AG-11/AG-12 discipline. |

## AG-12 archived-change conventions to mirror

`openspec/changes/archive/2026-08-16-cachicamas-agent-history/` contains: `proposal.md`, `exploration.md` (**not** `explore.md`), `design.md`, `tasks.md`, `apply-progress.md`, `verify-report.md`, `archive-report.md`, plus `specs/agent-history/spec.md`. The proposal header block carries Change / Milestone / Branch / Artifact-store / Pre-authorized / TDD / Closes / Blocks / Exploration lines.

AG-12 pre-authorized `size:exception` against its budget (forecast 1100-1600 lines, of which ~450-700 was SDD markdown alone). AG-13 touches more surface — `loop.go` and `scheduler.go` wiring, a new run-driver file, a steering queue, pause-resume, and a run-scope reconstruction test — so the 1000-line budget for this change is very likely to need an explicit extension.

## Approaches considered

1. **Extend `TurnOptions` with run-continuation and Scheduler-injection seams** (both nil-default, byte-stable). Preserves the documented "no positional signature change" contract, mirrors how `TurnOptions` already grew across AG-08/09/10, and keeps `TestTurn_TwoSequentialTurnsShareNothing` valid for the nil-default case. Cost: first milestone to touch `loop.go`/`scheduler.go` since AG-10; conditional emission must be proven against every existing `CheckStream` scenario. **Effort: high.**
2. **Driver-side event rewriting** — call `Turn()` unchanged per turn, then strip inner run brackets and re-synthesize an outer one. Requires no loop changes, but `Event`'s fields are unexported with no rewrite door, so the driver would have to reconstruct every payload through its own constructors — defeating "no privileged channel into the loop" and breaking identity provenance. **Not recommended.**

## Recommendation

Approach 1. Do not treat AG-13 as pure composition of three finished pieces: the RunID/TurnID continuity gap and the `WakeParked` exposure gap are genuine `Turn()`/`Scheduler` extensions. Budget explicitly for `loop.go`, `scheduler.go`, and `loop_test.go` changes, and take an explicit design decision on the `TurnOptions` extension shape before tasks are cut.

## Risks carried into the proposal

1. R-LSK-002 (`TestTurn_TwoSequentialTurnsShareNothing`) is a locked-in, tested invariant. Any fix must preserve it for the nil-default case or consciously amend it with sign-off.
2. Scheduler injection changes `Turn()`'s internal `Schedule` call site; every existing AG-09/AG-10 scheduler test must stay green.
3. The 1000-line review budget is very likely tight; AG-12's comparable change needed `size:exception`.
4. `CheckStream`'s missing cross-event RunID-value consistency check is a latent gap. Decide explicitly whether AG-13 strengthens it or defers — do not silently rely on it.

## Ready for proposal

Yes, with the four risks above carried into the proposal's Risks table and an explicit design decision requested on the `TurnOptions` extension shape.
