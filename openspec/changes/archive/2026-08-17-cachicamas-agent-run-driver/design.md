# Design: AG-13 — Drive the multi-turn run

> Change `cachicamas-agent-run-driver` · Resolves proposal decisions 1 and 2 and the R-LSK-006 reconciliation; re-litigates nothing else.
> New capability `agent-run-driver` (prefix `RUN`). Production code in `backend/agent/src/agent/harness.go` plus continuation seams in `loop.go`, `scheduler.go`, `tool.go`.

## Technical Approach

A value-form `Harness` (the name `agent-loop-skeleton/spec.md:105` and `agent-pre-request-hook/spec.md:103` already reserve) — an exported-fields struct with no constructor, mirroring `Scheduler` (`tool.go:229`): zero surface beyond two methods, `Run` and `Steer`. It drives `Turn` N times through one nil-default `TurnOptions` extension, `Continuation`, that carries the four things a multi-turn run must share: the run identity, the `LaneStamper`, the `*Scheduler`, and the `History`. Nil `Continuation` is byte-stable AG-07..AG-12 behavior on every path. The harness emits the single `run_start`/`run_end` pair itself through the public constructors (`NewRunStart`, `NewRunEnd`) and the shared stamper; `Turn` keeps ownership of the turn brackets. Everything the run's History holds is either appended by the harness (user-side messages) or by the continuation-mode loop (the turn's assistant message and its tool results) — exactly one appender per message class, so nothing is double-appended or dropped.

## The Reconciliation — `R-LSK-006` vs AG-13.1 scenario 3 (named, binding on `sdd-spec`)

The two statements in tension:

- `agent-loop-skeleton/spec.md:83` (R-LSK-006): "The scheduler's `Schedule` function is the seam: callable from `Turn` (AG-09) or from `Harness` (AG-13), but `Turn` MUST call it at most once per invocation."
- Doc 0003:1330-1333 (AG-13.1 scenario 3): "the harness holds no privileged channel into the loop … it goes through the same public one-turn surface the skeleton's external tests use."

**Resolution: the charter wins; the harness never calls `Schedule`.** Reasons: (a) doc 0003 is the task-graph authority; R-LSK-006's parenthetical was an AG-09-era forecast about *who might* call the seam, written before AG-13.1's scenario was weighed. (b) `doc.go:31`'s package contract `L2C-03` — "callers observe the stream, they never reach into the loop" — already encodes the charter's reading package-wide. (c) A harness-side `Schedule` call (proposal options 1D/2D) would force the harness to re-accumulate tool calls and re-emit tool events outside a turn bracket, reproducing exactly the provenance destruction that disqualified option 1C.

**What stays true, explicitly:** R-LSK-006's operative clause — "`Turn` MUST call `Schedule` at most once per invocation" — remains TRUE under AG-13. The harness iterates by calling `Turn` repeatedly; each `Turn` invocation still calls `Schedule` at most once (`S-LSK-008`/`S-LSK-008a` green untouched).

**What the harness does own:** the `*Scheduler` *value* — it constructs (or receives) it, injects it via `Continuation`, and calls `WakeParked` on it. `WakeParked` is not the `Schedule` seam and not a channel into the loop: it is the upward-path wake surface D-A assigns to the loop's caller, and `agent-permission-protocol/spec.md:144` explicitly reserves its wiring for AG-13.

**Back-annotation `sdd-spec` MUST make:** amend R-LSK-006's seam sentence to: `Schedule` is invoked only by `Turn`; the AG-13 `Harness` owns the `Scheduler` value and its wake surface (`WakeParked`) but does not call `Schedule`. The "callable … from `Harness` (AG-13)" phrase is the losing statement and must be re-scoped, not deleted silently.

## Decision 1 — run/turn identity continuity: Option A, verified

| Option | Verdict |
|---|---|
| **A — nil-default `TurnOptions.Continuation`** | **Chosen.** Verified implementable below. |
| B — change `Turn`'s positional signature | Rejected: breaks `loop.go:11-14`'s documented contract and `S-LSK-011`. |
| C — driver-side event rewriting | Rejected: `Event` fields unexported; destroys provenance; contradicts scenario 3. |
| D — harness calls `Schedule` directly | Rejected by the reconciliation above. |

```go
// TurnContinuation — AG-13's run-continuation seam. Nil = pre-AG-13 behavior byte-stable.
type TurnContinuation struct {
    Run       RunID        // caller-minted; Turn joins it instead of minting (loop.go:132's forecast)
    Stamper   *LaneStamper // caller-owned; ONE contiguous 1-based lane for the whole run
    Scheduler *Scheduler   // caller-owned; WakeParked reachable while Turn is blocked (Decision 2)
    History   *History     // caller-owned; Turn appends this turn's assistant message + tool results
}
// TurnOptions gains: Continuation *TurnContinuation
```

All four fields are required non-nil/non-empty when `Continuation != nil`; a half-configured continuation is rejected typed (`ai.Invalid(ai.ErrEmpty, ai.At("continuation", …))`) before any emission. `TurnID` is still minted fresh per call — N distinct turn brackets need N identities.

Continuation-mode deltas inside `Turn`, each gated on `opts.Continuation != nil`:

1. `runID = cont.Run`; `stamper = cont.Stamper` — the existing locals at `loop.go:187-189`. Every emission site (`emitStamped`, `newTurnAccumulator`, `Schedule`) already threads these two locals, so the injection is one assignment each; `emitStamped` honors the injected stamper with zero change (verified `loop.go:355-358`).
2. **No `run_start`, no `run_end`, on any path** — including the mid-stream fatal path (`loop.go:294` run-end suppressed; `turn_end(Aborted)` still emitted). The harness owns the run bracket because only it knows the run boundary: a terminal finish with a non-empty steering queue must NOT close the run.
3. **Schedule-before-finalize.** Verified against the descriptor table: all five tool kinds and all permission kinds are `PlacementTurn` (`event.go:285-318`), and `CheckStream` rejects a `PlacementTurn` event outside an open turn (`stream_check.go:161-163`) and anything after the `Terminal: true` `run_end` (`:108-110`, `:249`). Today's nil path runs finalize-first (`loop.go:255-266`), so tool events land after `run_end` — a latent misordering that no existing test observes (`CheckStream` is called nowhere in `loop_test.go`/`loop_hook_test.go`/`loop_tool_dispatch_test.go`; the dispatch tests count kinds only). The continuation path reorders: `results := sched.Schedule(...)` first (tool/permission events inside the open turn), `finalize()` second (`turn_end` only). The nil path keeps finalize-first byte-stable.
4. **The rejoin discard ends here** (continuation only): the `_ = sched.Schedule(...)` at `loop.go:265` — the gap `agent-turn-termination/spec.md:148` names — becomes `results :=` on the continuation path. `reconstructMessage` (continuation only) additionally appends the turn's `ai.ToolCall` parts from `t.toolCalls` (provider-exact bytes), so a pure-tool turn yields a constructible assistant message and History's pairing invariant has calls to match.
5. **History wiring** (the `agent-history/spec.md:185/:201` deferral coming due): after finalize, `Turn` appends the assistant message (skipped if zero — a ran-to-empty turn appends nothing), then one `ai.RoleTool` message per rejoin result in call order (`ai.NewToolResult(callID, content)` for `ToolOutcomeSuccess`, `ai.NewToolFailure(callID, content)` for both failure outcomes — verified `ai/tool_result.go:77,105`; the SynthesizeOrphans one-message-per-result precedent). Append failure is a typed error return. `CloseTurn` stays the harness's call.

**Bracket validity for N turns in one run** — verified against `stream_check.go`: `BracketRoleOpensRun` rejects a second `run_start` (`:122-126`) so the harness emits exactly one; `turn_start`/`turn_end` carry no `CardinalityAtMostOne` and the state machine re-admits a new turn after the previous closes (`:137-155`); only `run_end` is `Terminal: true` (`event.go:249`). One shared stamper gives the one contiguous 1-based lane rule 1 demands.

**`Scheduler.LeaveSinkOpen`** (new field on `tool.go:229`'s struct, zero-value false = AG-09 behavior): `Schedule` today closes `sink` unconditionally after the rejoin (`scheduler.go:219`), which would make the continuation path's post-`Schedule` `turn_end` a send on a closed channel. The harness sets `LeaveSinkOpen: true` on the scheduler it owns; `Schedule`'s close-third step becomes conditional. Every existing direct-`Schedule` test constructs `&Scheduler{MaxConcurrentReads: …}` keyed, so the new field is invisible to them. `tool.go` is not in `R-LSK-004`'s forbidden list (verified `agent-loop-skeleton/spec.md:60`) and is already in both substrate filters (`loop_test.go:853`, `loop_hook_test.go:925`). This is the one file the proposal's Affected Areas missed; recorded here as a conscious addition.

### Existing tests asserting unconditional brackets / sequence-restart — exhaustive enumeration

Grep-verified: `EventKindRunStart|EventKindRunEnd|Sequence(` appears in exactly these tests.

| Test | What it pins | Under AG-13 |
|---|---|---|
| `TestTurn_WalkingSkeleton_EmitsContractEventOrder` (`loop_test.go:326`; brackets `:351,:359`, sequence `:372`, run-end payload `:418`) | run_start first, run_end last, 1-based lane | `TurnOptions{}` → nil path → **literally green, file-unchanged** |
| `TestTurn_TwoSequentialTurnsShareNothing` (`loop_test.go:960`; sequence-restarts-at-1 `:1028-1037`) | two invocations with **fresh opts** share nothing | Nil path → **literally green**. Its Given ("fresh slices, fresh opts, fresh sink, fresh provider script", `:952-953`) scopes R-LSK-002 to independent invocations; a continuation-opts invocation is a different scenario, not an amendment. This reading is recorded here as binding; if `sdd-verify` disagrees, that is a conscious sign-off point, not a quiet edit. |
| `TestTurn_ReasoningPassThroughByteExact` (`loop_test.go:1058`; brackets `:1088,:1098`) | full kind order incl. run brackets | Nil path → **literally green** |
| All 10 `loop_hook_test.go` tests | zero `RunStart`/`RunEnd`/`Sequence` assertions (grep-verified) | Nil path (only `PreRequestHook`/`Model` set) → green |
| All 3 `loop_tool_dispatch_test.go` tests | tool-event **counts** only, no order/bracket assertions | Nil path → green. NB: the `:137-139` comment claims tool events are "bracketed by the loop's own brackets" — false today (they follow `run_end`); comment-only, no assertion, left untouched |
| AG-11's `turn_termination_test.go` / `turn_failure_test.go`, AG-10's `loop_permission_e2e_test.go` | fatal-path brackets, permission e2e | All nil-continuation → green file-unchanged |

**No existing test file changes except the two substrate-filter widenings.** The proposal's "re-scoped bracket assertions" row turns out to be unnecessary: conditional emission never fires on the nil path.

## Decision 2 — the wake seam: Option A (inject `*Scheduler`), not C

| Option | Verdict |
|---|---|
| **A — `Continuation.Scheduler`** | **Chosen.** |
| B — return the scheduler from `Turn` | Useless: handle needed while `Turn` is blocked. |
| C — wake channel/callback via `TurnOptions` | Rejected on surface and semantics: `WakeParked` already exists with exact per-callID semantics (`ErrStrayDecision`, W3 registration-before-emission, D-B two-release-paths — `scheduler.go:235-243`, `:196-216`) proven by the AG-10 suite. A callback would be a second wake vocabulary that either duplicates the parked-set keying or delegates to it anyway, and it would still need an object to live on. One pointer field adds no new concept. |
| D — harness calls `Schedule` | Rejected by the reconciliation. |

**Resolution contract (proposal risk 6), stated:** `parkedSet` lives exactly one `Schedule` call (`scheduler.go:147-150`, cleared at `:214-216`). A `PermissionDefer` suspension inside a harness-driven turn resolves by exactly one of: (a) an external `Scheduler.WakeParked(callID)` on the injected scheduler; (b) cancellation of the run `ctx`, which the harness propagates unmodified into every `Turn`. The harness adds **no third path and no timeout**: a run whose policy defers and whose owner neither wakes nor cancels does not terminate — by design. That is AG-10's own contract (R-APP-009 is the safety net), and richer cancellation vocabulary is AG-14's. Post-W3, registration precedes the `decision_required` emission and the emission's ack precedes the parked wait, so a consumer that has read `decision_required` off the run stream may call `WakeParked` with a guaranteed-live entry — the tests' synchronization is the stream itself, no wall clock.

## `Harness` surface and file layout

One production file, `harness.go` (steering queue inline as unexported `steeringQueue` — one file, one substrate-filter entry, AG-12's one-file precedent):

```go
type Harness struct {
    Provider  ai.ModelProvider // required
    System    string
    Turn      TurnOptions   // per-turn base (Model, MaxTokens, hooks, Tools, PermissionPolicy)
    Scheduler *Scheduler    // caller-owned wake handle; nil → harness constructs one
    History   *History      // caller-owned transcript; nil → NewHistory()
    queue     steeringQueue // zero-value ready; mutex + FIFO slice + closed flag
}
func (h *Harness) Run(ctx context.Context, prompt ai.Message, sink chan<- *Event) (ai.Message, ai.FinishReason, error)
func (h *Harness) Steer(msg ai.Message) error
```

Value-form, no constructor, nil-defaults resolved at `Run` entry into locals (caller fields never mutated, except `LeaveSinkOpen` set once on the scheduler before any turn). One `Run` per `Harness` value; cross-run state is AG-21. `Steer` contract: **nil return ⇒ the message enters History before a subsequent provider call (zero drops); after the run's terminal decision it returns `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`.** Run identity: harness-minted `run-hrn-<n>` package atomic counter (distinct provenance prefix from `run-lsk-`; `loop.go:132`'s "caller-supplied groups" forecast honored without touching the loop's minters).

## The run algorithm

1. Resolve defaults; validate `prompt`; `stamper := &LaneStamper{}`.
2. Emit `run_start` (public `NewRunStart` + `stamper.Stamp`, sequence 1) on the consumer sink.
3. `history.Append(prompt)`.
4. **Loop:** (a) drain the steering queue FIFO under its mutex, `Append` each; (b) `transcript := ` messages from `history.Entries()`; (c) make a per-turn channel `turnSink` and start one forwarder goroutine relaying `*Event` pointers to the consumer sink — mandatory, so `decision_required` is observable while `Turn` is still blocked; (d) `Turn(ctx, provider, system, transcript, optsWithContinuation, turnSink)`; (e) wait for the forwarder (turnSink is closed by `Turn`; `Scheduler` leaves it open per `LeaveSinkOpen`); (f) on error: emit `run_end(RunOutcomeFailed, failure)` (wrapping via public `ai.MidStreamFailure` + `NewFailure`; `NewRunEnd` enforces failure-iff-Failed, `run_events.go:148-176`), close sink, return — no append, no `CloseTurn`, retry is AG-15's; (g) `history.CloseTurn()` — succeeds because the loop appended calls-then-results, open set empty.
5. **Dispatch on finish:** `ToolCalls` → iterate (results already in History). `PauseTurn` → iterate (the partial is already appended; next transcript replays it verbatim). Every other vocabulary member (`Stop`, `Length`, `ContentFilter`, `Refusal`, `Unknown`) → terminal candidate: atomically, under the queue mutex — if non-empty, take the messages and iterate (**a message queued during the final turn yields a new turn**); if empty, mark the queue closed (later `Steer` rejected typed) and terminate.
6. Terminate: emit `run_end(RunOutcomeCompleted)`, close the consumer sink, return the last turn's `(msg, finish, nil)`.

Stamper single-writer discipline: the harness touches the stamper only between turns (steps 2 and 6); during a turn only `Turn` and the scheduler's dispatcher do — strictly sequential phases, race-free under `-race`.

Pause semantics rendered concretely: the returned `ai.Message` is `reconstructMessage`'s partial (text + reasoning with byte-exact round-trip token); "replayed verbatim" = that exact value appended to History and re-included in the next request's transcript — never re-synthesized. The pause stays visible because `finalize` already emits `turn_end` with `TurnOutcomePaused` via `outcomeForFinish` (`loop.go:343-344`; `turn_events.go:106-110`) and the harness forwards, never rewrites.

```mermaid
sequenceDiagram
    participant C as Consumer (sink)
    participant H as Harness.Run
    participant T as Turn (continuation)
    participant S as Scheduler (injected)
    participant P as Policy owner
    H->>C: run_start (seq 1)
    H->>H: Append(prompt); drain steer queue
    H->>T: Turn #1 (transcript, cont)
    T->>C: turn_start, message events
    T->>S: Schedule(calls) — before finalize
    S->>C: permission_decision_required (parked)
    P->>S: WakeParked(callID)
    S->>C: decision_made, tool_start, tool_end_success
    T->>C: turn_end(ToolCalls)
    T->>T: Append(assistant+calls), Append(RoleTool results)
    H->>H: CloseTurn; drain steer queue (steered msg → Append)
    H->>T: Turn #2
    T->>C: turn_start, partial text, turn_end(Paused)
    T->>T: Append(partial)
    H->>H: CloseTurn; iterate (pause)
    H->>T: Turn #3 (transcript incl. partial verbatim)
    T->>C: turn_start, text, turn_end(Finished)
    H->>C: run_end(Completed) — sink closed
```

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/harness.go` | Create | `Harness`, `steeringQueue`, run algorithm, run-bracket emission, run-ID minting |
| `backend/agent/src/agent/harness_test.go` | Create | AG-13.1: two-turn run, run-scope reconstruction + bite, RunID-value consistency, no-privileged-channel source guard (`package agent_test`) |
| `backend/agent/src/agent/harness_steering_test.go` | Create | AG-13.2 scenarios (`package agent_test`) |
| `backend/agent/src/agent/harness_pause_test.go` | Create | AG-13.3 scenario (`package agent_test`) |
| `backend/agent/src/agent/harness_suspension_test.go` | Create | Permission-suspension clause + the R-APP-002 parked-wait bite (`package agent_test`) |
| `backend/agent/src/agent/loop.go` | Modify | `TurnContinuation`; conditional identity/brackets; schedule-before-finalize + rejoin capture + History append (continuation only) |
| `backend/agent/src/agent/tool.go` | Modify | `Scheduler.LeaveSinkOpen bool` (zero = AG-09 behavior) — conscious addition vs the proposal's Affected Areas |
| `backend/agent/src/agent/scheduler.go` | Modify | close-third step conditional on `LeaveSinkOpen` (`:219`); doc update |
| `backend/agent/src/agent/loop_test.go` | Modify | `filterOutLoopFiles` widened — five exact suffixes below |
| `backend/agent/src/agent/loop_hook_test.go` | Modify | `filterOutLoopHookFiles` — same five, byte-in-sync |
| Five spec deltas + doc 0003 `:2170` tick, counters 13/24 | Delta | Per proposal; plus `agent-turn-termination/spec.md:148` (the `_ =` discard line AG-13 now changes) and the stale claim at `agent-permission-protocol/spec.md:172` (see bite) |

Substrate-filter widening (NFR-HIS-004 discipline — exact filenames, no wildcards, both filters byte-in-sync): `/harness.go`, `/harness_test.go`, `/harness_steering_test.go`, `/harness_pause_test.go`, `/harness_suspension_test.go`. `loop.go`, `scheduler.go`, `tool.go` are already listed in both filters. Every `R-LSK-004` file — including `stream_check.go`, `event.go`, `run_events.go`, `turn_events.go`, `doc.go`, `reconstruction_test.go`, `event_registry_test.go` — stays byte-unchanged; `history.go` untouched (no new exported route; `history_surface_guard_test.go` green untouched).

**No `L2C-08` row, confirmed:** one-run/N-turn bracketing is behavior of the run/turn families that `CheckStream`'s existing state machine already validates — not a new package-wide guarantee on every future event family. "No privileged channel" is `L2C-03` verbatim (`doc.go:31`); run-scope reconstruction is `L2C-04`/`L2C-05` at wider scope. Overturn criterion: if `sdd-spec`'s deltas end up binding *future* event families to the one-run bracket (they do not — AG-19's child runs re-open the question), add the row and clear the `doc.go` substrate bar explicitly.

## Architecture Decisions

| Decision | Alternatives rejected | Rationale |
|---|---|---|
| One grouped `Continuation *TurnContinuation` pointer, all-or-nothing validated | Four independent `TurnOptions` fields | A half-configured continuation (RunID without stamper) silently breaks lane contiguity; one nil-check gates every conditional site |
| Harness owns run brackets; `Turn` keeps turn brackets | `Turn` conditionally emits `run_end` on last turn | `Turn` cannot know a turn is last: a terminal finish with a queued steering message continues the run |
| Schedule-before-finalize, continuation only | Reorder both paths | Nil-path reorder needs the sink-close change on the nil path too and alters a byte-stable ordering existing tests never assert but AG-09's docs pin; continuation path is where `CheckStream` acceptance is load-bearing |
| Loop appends assistant + results; harness appends user-side; harness closes turns | Harness rebuilds messages from observed events | Deny/orphan results emit no `tool_start`, so event-side rebuild loses the call's name/args; the rejoin slice is fully populated by construction (R-TLS-009) and `agent-history/spec.md:185` explicitly forecast wiring "into `Turn`/`Schedule`" |
| `LeaveSinkOpen` field on `Scheduler` | New `Schedule` parameter (breaks public signature); private forwarding channel inside `Turn` (second goroutine, ordering risk) | Keyed-literal construction everywhere keeps existing tests untouched; zero value preserves AG-09 |
| Value-form struct, no constructor, no interface | `NewHarness`; a `Harness` interface | `Scheduler` precedent; no second implementation exists (AG-04..AG-12 rule) |
| Harness mints `run-hrn-<n>` itself | Reuse `mintLoopRunID` | Keeps the harness off loop internals; provenance-distinct prefix |
| Terminal decision atomic with queue close | Check-then-close without the lock | The gap would drop a racing `Steer` accepted with nil — violates zero-drops |

## Testing Strategy (strict TDD, `cd backend/agent && make test`, all files `package agent_test`)

Names `Test<Subject>_<Behavior>_<Expectation>`; banners cite the leaf (`// AG-13.1 — …`); RED recorded before GREEN per scenario; synchronization by `agenttest.Gate` and channel reads — never wall-clock ordering.

| Test | Scenario / sync | Assertions |
|---|---|---|
| `TestHarness_TwoTurnRun_CompletesToTerminal` | AG-13.1 sc.1. `agenttest.NewProvider(scriptToolCall, scriptFinalText)` — multi-script, one per `Turn`; drain sink to close | Kind order run_start → turn_start₁ → … tool_start/tool_end_success … → turn_end₁(ToolCalls) → turn_start₂ → … → turn_end₂(Finished) → run_end(Completed); exactly one run bracket; `CheckStream(events)` accepts unmodified; every event's `Run()` equals the one run ID (AG-13's own assertion — `CheckStream` never checks identity values, deferred to AG-19); History alternates with every pair matched (`Entries()` walk) |
| `TestHarness_RunStream_ReconstructsHistoryAtRunScope` + `TestHarness_RunStream_ReconstructionBiteDropsTurnTwoEvent` | AG-13.1 sc.2 + non-vacuity bite (mirrors S-LSK-003a/b) | Partition replayed events by turn bracket; reconstruct messages + tool outcomes; deep-equal against `history.Entries()` at run scope. Bite: copy the slice, drop one turn-two event, assert the comparator REPORTS divergence — RED-recorded first |
| `TestHarness_LoopAccess_PublicOneTurnSurfaceOnly` | AG-13.1 sc.3 | Source-scan guard (the `scheduler_test.go` regex precedent): `harness.go` contains no reference to loop internals (`turnAccumulator`, `mintLoop`, `emitStamped`, `closeSink`, `buildLoopRequest`, `outcomeForFinish`) and no `.Schedule(` call site; plus the behavioral half — every harness test lives in external `agent_test` and observes only the stream/History |
| `TestHarness_MidTurnSteer_EntersAtBoundaryBeforeNextProviderCall` | AG-13.2 sc.1. Turn-one script holds at a `Gate`; on `Reached()` → `Steer(msg)` → `Release()` | In-flight turn's events unchanged; steered message in History between turn-1 and turn-2 messages; a test-side request-recording provider wrapper (the `newCtxRecordingProvider` precedent, `loop_test.go:476-485`) proves turn-2's `ai.Request` transcript contains it — "before the next provider call", by evidence; consecutive same-role entries accepted |
| `TestHarness_SteerBurst_ArrivalOrderZeroDrops` and `TestHarness_FinalTurnSteer_YieldsNewTurn` | AG-13.2 sc.2. Gate-held final turn; N `Steer`s from a second goroutine (ordered by the test, concurrent with the run) | All N in History in arrival order, zero drops; a message steered during the FINAL turn produces an additional turn bracket (script provides response N+1) rather than a drop; `Steer` after run end returns the typed rejection |
| `TestHarness_PauseFinish_ResumesVerbatimToRealTerminal` | AG-13.3. Script 1: partial text + reasoning-with-binary-token + `Completion(PauseTurn)`; script 2: final text + Stop | Stream shows `turn_end` outcome `TurnOutcomePaused` (visible, not absorbed); History's paused-turn entry content deep-equal to the returned partial including the round-trip token bytes; request recorder proves the next transcript replays it verbatim; run ends `Completed` |
| `TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake` | Fourth acceptance clause. Policy: Resolve#1 → `Defer`, Resolve#2 → `AllowOnce`; harness in a goroutine with `runDone` channel; test reads the consumer sink event-by-event | On reading `decision_required`: `runDone` not closed (non-blocking select — deterministic: the call goroutine is parked) and tool invocations == 0; `WakeParked(callID)` returns nil (post-W3 the entry is guaranteed live once the emission is read — the stream IS the synchronization, no wall clock, no timeout); run then completes; suspension events sit INSIDE turn one's bracket; `CheckStream` accepts |
| `TestHarness_PermissionDefer_ParkedWaitObservedBite` | The inherited R-APP-002 bite. `agent-permission-protocol/spec.md:172`, verbatim: "**Known gap (carried to AG-13)**: the `R-APP-002` acknowledgement itself currently has no non-vacuous guard — deleting the acknowledgement leaves the package green. The behaviour is present and correct in production; the missing bite must observe the parked **wait**, not the registration." | Mechanism: the policy's Resolve#2 asserts a `wake-issued` atomic flag the test sets immediately before `WakeParked` — under correct code the flag read is happens-after the wake via `parkCh` close, guaranteed true; combined with the run-in-flight and zero-invocations checks above, the WAIT (not the registration) is what is observed. Bite evidence: scratch-replace the parked-wait select with an immediate re-resolve → the flag assertion fires; RED recorded over `go test -race -count=15 ./src/agent/` (NFR-APP-002's repeated-run discipline; S-PPB-002's 20/20 evidence precedent), then reverted. Honesty note for `sdd-spec`: `TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery` (AG-10 remediation) already guards the ack at `Schedule` level, so `:172`'s "leaves the package green" is stale — the delta must record that test as the existing ack guard and this bite as the loop-level parked-wait observation, verified by actually running the scratch at apply time |
| Guards | — | `TestTurn_TwoSequentialTurnsShareNothing` and every AG-09/AG-10 scheduler test pass **file-unchanged**; both substrate filters byte-in-sync; import guard (harness imports stdlib + `ai` only), ambient-authority guard (no clock/I/O in `harness.go` — forwarder syncs by channel close), `TestTurn_CoverageGate` ≥ 80% on `loop.go` including the new continuation branches |

`make test` (`-race`), `make lint` (after `golangci-lint cache clean`), `make build`, `make vuln-check` (not in `make all`).

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The harness is a pure in-memory driver under L2C-02.

## Migration / Rollout

No migration. Rollback per proposal — single revert; every seam is nil-default/zero-default additive (`Continuation`, `LeaveSinkOpen`), so the parent commit's behavior is exactly the nil path. One addition to the proposal's rollback inventory: `tool.go` returns to its AG-09 field set.

## Open Questions

None blocking. Three obligations forwarded: (1) `sdd-spec` carries the R-LSK-006 back-annotation exactly as the Reconciliation section words it; (2) `sdd-spec` re-homes `agent-tool-scheduler/spec.md:138`'s "`ToolSource` port (G6) — AG-13 widening" to AG-20 — the adjacent line `:137` already names AG-20 and the AG-13 charter never mentions tool sources — or records why it stays; (3) `sdd-spec` records the `:172` staleness finding from the bite row. `sdd-tasks` must pin the schedule-before-finalize reorder and the `LeaveSinkOpen` seam as their own RED-first tasks, and verify at apply time that `ai.NewRequest` accepts a transcript containing `RoleTool` result messages (expected — History's pairing guarantees calls precede results — but unproven until run).
