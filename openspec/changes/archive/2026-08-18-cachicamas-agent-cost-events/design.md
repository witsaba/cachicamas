# Design: AG-16 — Emit cost and usage events (`cachicamas-agent-cost-events`)

> Change `cachicamas-agent-cost-events` · Milestone AG-16 · Worktree `cachicamas-worktrees/ag16`, base `main@09bb30e1`
> Inputs: `proposal.md` (Decisions 1–5), `explore.md` §1–§10. Every citation below re-read against the worktree during this phase.
> Settled upstream, cited not re-litigated: `Turn`'s signature is frozen (`agent-turn-termination/spec.md:113`, `NFR-ATT-004` at `:153`); `S-APE-083` reflects only `CostFigures` (`cost_events_test.go:199`); no new `EventKind`; token-only.

## Technical Approach

Capture `completion.Usage()` in `turnAccumulator`, emit `cost_turn(CostLabelFinal)` from `finalize()` inside the turn bracket, intercept it in `Harness.Run`'s existing forwarder into a run-local accumulator, and emit `cost_session(Estimate)` before each continued logical turn plus exactly one `cost_session(Final)` immediately before the run-close on **all three** run closes (success, failure, wind-down). Presence travels beside `CostFigures` as per-figure paired accessors mirroring `TokenCount.Count() (int64, bool)` (`usage.go:63-65`). `CostFigures`, `cost_events_test.go`, `stream_check.go`, `event_descriptor.go`, and all of `src/ai/**` stay byte-unchanged.

## Architecture Decisions

### DD1 — Presence: per-figure, stored beside `CostFigures`, read through paired accessors

**Context.** `ai.Usage` carries per-field presence (`usage.go:44-47`); `CostFigures` is five plain `uint64` (`cost_events.go:137-157`), pinned by `S-APE-083`'s reflection walk (`cost_events_test.go:199`, which inspects `CostFigures` only). "Absence is reported as absence, never invented zeros" (`0003:1535`).

**Rejected.** (a) presence inside `CostFigures` — breaks the reflection pin by field count. (b) a third `CostLabel` member — collides the presence axis with DD4's estimate/final axis. (c) skip emission on absence — Scenario 1 requires an event. (d) **whole-record flag — the named defect this shape exists to prevent**: for `ai.Usage{Input: ai.Tokens(100)}` a record-level "present" reports reasoning as `(0, present)` — an invented zero for four absent figures. Per-figure or nothing.

**Decision — exact declarations.** In `cost_events.go`:

```go
// costPresence records, per figure, whether Layer 1 reported it.
// Unexported: consumers read presence only through the paired
// accessors below, so a count is never readable without its bit.
type costPresence struct {
	input, output, cacheRead, cacheWrite, reasoning bool
}

type CostTurn struct {          // was: label, figures (cost_events.go:162-165)
	label    CostLabel
	figures  CostFigures
	presence costPresence       // NEW — beside, never inside, CostFigures
}
// CostSession (:227-230) gains the identical third field.

// Five paired accessors per payload type, mirroring
// TokenCount.Count() (usage.go:63-65). (0,false) = absent;
// (0,true) = reported nought. Ten methods total.
func (c CostTurn) InputTokens() (uint64, bool)      { return c.figures.InputTokens, c.presence.input }
func (c CostTurn) OutputTokens() (uint64, bool)     { ... }
func (c CostTurn) CacheReadTokens() (uint64, bool)  { ... }
func (c CostTurn) CacheWriteTokens() (uint64, bool) { ... }
func (c CostTurn) ReasoningTokens() (uint64, bool)  { ... }
// same five on CostSession
```

**Constructors.** `NewCostTurn`/`NewCostSession` (`cost_events.go:181-193`, `:248-257`) keep their exact signatures — `cost_events_test.go` calls them and must stay byte-unchanged — and set presence **all-reported** (a caller handing five plain figures asserts five figures; `Figures()` behavior is unchanged). The presence-bearing path is package-private, in new file `cost_usage.go`:

```go
// costFromUsage is total and pure: Tokens(0) → (0, reported);
// unreported → (0, absent). int64→uint64 is safe — Layer 1
// validates non-negativity (usage.go:151-169).
func costFromUsage(u ai.Usage) (CostFigures, costPresence)

func newCostTurnFromUsage(run RunID, turn TurnID, label CostLabel, u ai.Usage) (Event, error)
func newCostSessionFromTotals(run RunID, label CostLabel, f CostFigures, p costPresence) (Event, error)
```

The conversion helper is **package-private** (closing the proposal's open sub-question): no Layer 3 exists (`0003:110`), external tests read events through the accessors, and a smaller surface keeps the rollback clean. Zero-value payloads (wrong-kind `Event.CostTurn()`, `cost_events.go:199-202`) report all-absent — coherent.

### DD2 — Emission site and ordering: `cost_turn` from `finalize()`, immediately before `turn_end`

**Decision (site — settled by R-ATT-007/NFR-ATT-004, executed here).** `turnAccumulator` (`loop.go:815-823` constructor) gains `usage ai.Usage`; the Completion case (`loop.go:923-927`) adds `t.usage = completion.Usage()` beside the existing finish capture; `finalize()` (`loop.go:1001-1025`) emits, as its **first** emission:

```go
if costTurn, cerr := newCostTurnFromUsage(t.runID, t.turnID, CostLabelFinal, t.usage); cerr == nil {
	emitStamped(t.sink, t.stamper, costTurn)
}
```

**Why before `turn_end`:** `cost_turn` is `PlacementTurn`; `CheckStream` rejects a turn-placed event outside an open turn (`stream_check.go:161-163`) and the bracket closes at `turn_end` (`:147-155`). Emitting first-in-`finalize()` holds on both call paths (`loop.go:383` nil, `finishContinuationTurn` continuation) and lands after tool/permission events on the continuation path (schedule-before-finalize, `R-LSK-001` point 3) — still inside the bracket.

**Per-path rule table:**

| Path | `cost_turn`? | Why |
|---|---|---|
| Normal completion (either call path) | Yes, `Final`, exact figures | `finalize()` reached with captured usage |
| Provider closes with no `Completion` (`S-ATT-012`) | Yes, `Final`, **all five absent** | `finalize()` runs with zero `ai.Usage{}`; a first-class DD1 absence case, not an error |
| Mid-stream fatal (`loop.go:396-430`) | **No** | never reaches `finalize()`; `Completion` is `Terminal` (`ai/event.go:167-170`) so no usage exists; the aborted brackets already carry the failure |
| Pre-stream aborts (`emitPreStreamAbort`, `loop.go:358`) | **No** | same — no data source (`provider_failure.go:320` carries no usage) |
| Cancellation mid-turn (AG-14) | **No** | routes through the fatal path; `turn_end(Aborted)`, no invented figures |

Rule stated once for `sdd-spec`: **a `cost_turn` is emitted iff the turn closes non-aborted; its figures carry per-figure presence from whatever the stream reported, possibly nothing.** Harness aggregation: the forwarder (`harness.go:507-509`) becomes `if ct, ok := ev.CostTurn(); ok { total.add(ct) }; sink <- ev` — a pure read on the existing single-forwarding path, no double-emission (the event is forwarded exactly once, unmodified). Race-free: `total` is written only by the live forwarder goroutine and read only after `<-forwarderDone` (`harness.go:521`; close-of-channel happens-before). Standalone `Turn` (nil continuation) emits `cost_turn` with no aggregator — acceptable and to be stated in the spec: `cost_session` is harness-scoped.

**Blast radius (enumerated; see the table below).** `S-LSK-001` (`agent-loop-skeleton/spec.md:56`) enumerates the nil-path happy sequence **closed** ("observes in order: run_start, turn_start, …, turn_end, run_end") and its implementing test asserts `len(got) != len(wantKinds)` (`loop_test.go:350-361`). R-LSK-001's AG-13 clause also claims nil-path "byte-stable pre-AG-13 behavior" (`spec.md:29`, `:35`). **This is a cross-capability amendment**, declared: the `agent-loop-skeleton` delta amends `R-LSK-001` (nil-path emission obligation + an AG-16 carve-out on the compatibility clause, following the AG-15 precedent that already amended it for pre-stream aborts) and appends an "Amended by AG-16" annotation to `S-LSK-001`'s sequence. `agent-turn-termination` pins **relative** order only (`S-ATT-007`: turn_end → run_end → close) on the fatal path, which gains no event — back-annotation only, no amendment.

### DD3 — Cumulative algebra: sum over every emitted `cost_turn` in the run bracket

Confirmed against code: each attempt of a retried logical turn is its own `Turn` invocation with its own forwarder (`harness.go:502-521`), so any attempt reaching a `Completion` emits its own counted `cost_turn` — retry-inclusive by construction, no retry-awareness in the accumulator. An attempt cannot fail after reaching a `Completion` and then retry: `retryDecision` G1 requires a typed `*ai.Failure` and G3 blocks on partial output (`retry_policy.go:111-126`); post-`Completion` errors (e.g. history commit) are plain errors → surface. Even if one ever became reachable, the sum-over-emitted-events rule already counts it — the question dissolves.

**Per-figure algebra** (implemented by `costAccumulator.add`, justified by "never invented zeros" — a figure no turn ever reported stays absent, not `0`):

| left | right | cumulative count | cumulative presence |
|---|---|---|---|
| absent | absent | 0 | **absent** |
| absent | present *n* | *n* | present |
| present *m* | present *n* | *m + n* | present |
| present 0 | absent | 0 | **present** (a reported nought survives) |

Absent is the additive identity in value; presence is OR. Plain `uint64` addition, no saturation: each contribution is bounded by `int64` max (Layer 1 non-negativity), overflow is unreachable in any real run and is not guarded.

```go
// cost_usage.go
type costAccumulator struct {
	figures  CostFigures
	presence costPresence
}
func (a *costAccumulator) add(c CostTurn)                                  // the algebra above, per figure
func (a *costAccumulator) sessionEvent(run RunID, label CostLabel) (Event, error)
```

`total` is a **local in `Run`'s stack frame**, never a `Harness` field (`Harness` is value-form, serially reused, carries no cross-run state — `harness.go:47-118`, `R-CAN-002`).

### DD4 — Estimate/final at run scope, closed on every run-close

`cost_turn` is **always `CostLabelFinal`**: `Completion` is the sole, at-most-once, terminal usage carrier (`completion.go:92`, `ai/event.go:167-170`), so per-turn usage is complete whenever known. The `Estimate` axis is run-scoped, on the charter's own authority (`0003:1537`).

| Emission | Label | Site | Condition |
|---|---|---|---|
| `cost_turn` | `Final` | `finalize()`, before `turn_end` | every non-aborted turn close |
| `cost_session` running total | `Estimate` | before each `continue` in `Run`'s outer loop — the ToolCalls/PauseTurn arm (`harness.go:606`) and the steering-took arm (`:615`) | only when the driver decides to run another logical turn |
| `cost_session` terminal | `Final` | immediately before the run-close event in **all three** closes: success (`harness.go:621-624`), `failRun` (`:335-342`), `windDownRun` (`:315-327`) | exactly once per run bracket |

**Non-happy closes, answered definitively.** A **failed** run (`R-RUN-011`) and a **cancelled** run (interrupt/shutdown wind-down) both emit `cost_session(Final)` carrying the cumulative at close time — tokens spent before the failure are real spend. `failRun` and `windDownRun` gain a `total *costAccumulator` parameter (unexported, five/three call sites, mechanical) and emit best-effort before their `NewRunEnd`, mirroring the existing best-effort posture. `R-RUN-011`'s no-append/no-`CloseTurn` posture is untouched — a cost emission writes no transcript. Two paths emit **nothing**: the post-shutdown refusal (`harness.go:388-392`, "MUST emit no event whatsoever", `R-CAN-005`) and the `NewRunStart` error return (`:438-441`) — no run bracket, no cost events. A single-turn success run therefore emits `Final` only (no `continue` ever ran), satisfying Scenario 3's conditional reading; `CardinalityAny` (both kinds) makes multi-emission legal; `PlacementRun` makes the between-turns and before-run-end positions legal (`stream_check.go:157-163`).

**Spec invariant for `sdd-spec`:** within any harness run bracket, the `cost_session` events are zero or more `Estimate`s followed by exactly one `Final` immediately preceding the run-close, on every outcome.

**Discovered amendment (a sixth delta, missed by the proposal's five).** `R-CAN-002` enumerates the wind-down order: "synthesize orphans; close the turn; emit the run-close event; return" (`agent-cancellation-tree/spec.md:65`), inherited by shutdown via `R-CAN-005` (`:98`). Inserting the terminal `cost_session` before the run-close preserves the enumerated order but must not be a silent edit: the **`agent-cancellation-tree` delta** records the insertion explicitly. `S-CAN-001/004/005` stay green: `CheckStream` accepts the new event, run-end remains last, and `S-CAN-004`'s same-stream equality compares two streams that both carry it.

## Emission sequence (multi-turn run, one retried turn, success close)

1. `run_start` (`harness.go:442`)
2. `turn_start` … `turn_end(Aborted)` — failed attempt; **no `cost_turn`**
3. `turn_start` … content events … `cost_turn(Final)` … `turn_end` — succeeding attempt
4. `cost_session(Estimate)` — driver decided to continue
5. `turn_start` … `cost_turn(Final)` … `turn_end` — last turn
6. `cost_session(Final)` — cumulative over events at 3 and 5
7. `run_end(Completed)`

Standalone `Turn` (nil continuation): `run_start, turn_start, …, cost_turn(Final), turn_end, run_end` — no `cost_session`.

## File Changes

| File | Action | Description |
|---|---|---|
| `src/agent/cost_usage.go` | Create | `costPresence` consumers' conversion `costFromUsage`, `newCostTurnFromUsage`, `newCostSessionFromTotals`, `costAccumulator` |
| `src/agent/cost_events.go` | Modify | `presence costPresence` field on `CostTurn`/`CostSession`; 10 paired accessors; DD4 label-axis doc comments at `:96-103`; **`CostFigures` (`:137-157`) byte-unchanged** |
| `src/agent/loop.go` | Modify | `usage ai.Usage` on `turnAccumulator`; capture at `:923-927`; emission first-in-`finalize()` (`:1001`) |
| `src/agent/harness.go` | Modify | forwarder interception (`:507-509`); `var total costAccumulator` local; `Estimate` before `:606`/`:615` continues; `total` param + `Final` emission in `failRun`/`windDownRun`; `Final` before `:621` |
| `src/agent/cost_usage_test.go`, `cost_turn_emission_test.go`, `cost_session_test.go` | Create | conversion table, Scenario 1, Scenarios 2–3 + non-happy closes (names final in `tasks.md`) |
| `loop_test.go` (`:831-871`), `loop_hook_test.go` (`:907-943`) | Modify | substrate filters widened by exact filename suffix, byte-in-sync, one entry per new file |
| **Spec deltas — SIX** | Delta | `agent-protocol-events` (R-APE-004/005), `agent-run-driver` (sites + `:326` closed), `agent-retry-failover` (`:219` closed, DD3 reading), `agent-loop-skeleton` (**amendment**: R-LSK-001 + S-LSK-001), `agent-turn-termination` (back-annotation: pin held), **`agent-cancellation-tree` (amendment: R-CAN-002 order insertion — added by this design)** |
| `docs/architecture/milestones/0003-…` | Modify | tick, counter 16/24, Decision 5 reconciliation note (AG-06 template, `0003:613`) |
| `cost_events_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `go.mod`/`go.sum`, `src/ai/**` | NOT TOUCHED | pins hold byte-unchanged |

## Test blast radius (enumerated) and remediation

| Test site | Path exercised | Effect | Remediation |
|---|---|---|---|
| `loop_test.go:350-361` (implements `S-LSK-001`, closed `wantKinds`) | nil-path happy | +`cost_turn` before `turn_end` | insert `EventKindCostTurn`; **signed-off amendment**, tied to the `agent-loop-skeleton` delta |
| `loop_test.go:1152-1165` (closed `wantOrder`) | nil-path tool dispatch | +`cost_turn` before `turn_end` | insert kind at position |
| `harness_steering_test.go:103-126` (turn-one closed `wantKinds`) | harness run | +`cost_turn`; +`Estimate` if the slice crosses `turn_end` | insert kind(s) |
| `retry_policy_test.go:110-131` `assertPreStreamAbortSequence` | pre-stream aborts | none — no `finalize()` | none |
| `turn_failure_test.go:86-125` (last-two positional) | mid-stream fatal | none | none |
| `harness_test.go:297-305` (first/last kind), `:1162-1173` (last=`run_end`) | continuation turn / run close | `turn_end`/`run_end` still last | none |
| `harness_pause_test.go:94`, `cancellation_interrupt_test.go:97-100,:212-215,:593-596`, `cancellation_shutdown_test.go:183-186` (last=`run_end`) | closes | `Final` precedes; `run_end` still last | none |
| kind-filtered scan loops (scheduler, permission, dispatch, e2e suites) | various | filtered; unaffected | none — verify at apply |
| Any live-stream `CheckStream` assertion | all | new events legally placed | none; proves DD2/DD4 with `stream_check.go` byte-unchanged |

Every edited test is a signed-off amendment recorded in `apply-progress.md` (proposal risk 3), never a quiet fix.

## Strict-TDD plan (RED-first; `cd backend/agent && make test`)

| Test (external `package agent_test`) | Covers | RED-first bite |
|---|---|---|
| `TestCostFromUsage_TableDriven` (internal table test permitted for the pure converter; else via emission) | conversion totality, `Tokens(0)` vs absent | RED: function absent |
| `TestTurn_CostTurn_FiguresExactPerTurn` | Scenario 1 exactness, all five figures, multi-turn | RED: no `cost_turn` on today's stream |
| `TestTurn_CostTurn_AbsenceVsReportedZero` | Scenario 1 absence: `ai.Usage{}` vs `ai.Usage{Input: ai.Tokens(0)}` events observably differ via accessors, never counts alone | RED: absent |
| `TestTurn_CostTurn_MixedRecordPartialPresence` | Scenario 1 mixed: `ai.Usage{Input: ai.Tokens(100)}` → four absent, input `(100,true)` | **Bite (a)**: collapse presence to one bool → this fails |
| `TestTurn_CostTurn_LabelAlwaysFinal` + `TestTurn_NoCompletion_CostTurnAllAbsent` + `TestTurn_AbortedTurn_NoCostTurn` | DD2 path table, `S-ATT-012` companion | RED: absent |
| `TestHarness_CostSession_CumulativeEqualsEmittedCostTurns` (retry via `errorProvider` wrapper precedent, `loop_test.go:1408-1421`) | Scenario 2 — equality against observed events on the same stream, never a literal | **Bite (b)**: skip retried logical turns in `add` → fails |
| `TestHarness_CostSession_EstimateThenFinal` + `TestHarness_SingleTurnRun_FinalOnly` | Scenario 3 | **Bite (c)**: label the terminal `Estimate` → fails |
| `TestHarness_CostSession_FinalOnFailedRun` + `TestHarness_CostSession_FinalOnInterruptedRun` | DD4 non-happy closes | RED: closes emit no `Final` today |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Single-PR revert per proposal Rollback Plan; the only new externally visible surface is the ten paired accessors, with no live consumer.

## Open Questions

None blocking. Two apply-time verifications flagged: (1) whether `harness_steering_test.go`'s turn-one slice crosses `turn_end` (decides if `Estimate` joins its `wantKinds`); (2) any kind-filtered suite that also asserts a total event count — none found by search, re-verified at apply.
