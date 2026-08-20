# Design: AG-19 — Prove re-entrancy and delegation readiness

> Change `cachicamas-agent-delegation-readiness` · builds on `proposal.md` (Approach A confirmed) · every `file:line` below was opened and verified in this worktree at `main@558641f3`.

## Technical Approach

One new production file, `delegation_seam.go`, exports a context-carried publishing seam; `executeCall` installs it for exactly the `tool.Run` frame and revokes it on every exit path. Everything that knows what a subagent *is* lives in `package agent_test`. The seam rides the existing `emissions` → `runDispatcher` → `sink` funnel (`scheduler.go:155,196,281-298`) — no new channel, no new loss rule (`R-AGE-017` untouched). The proposal's Q1/Q2/Q3/Q6 stand; this design decides the two questions the proposal left open (AD-1, AD-3) and tightens the admissibility rule where the proposal's predicate was not total (AD-2).

**Deviation from the 800-word budget**: the orchestrator's launch contract explicitly requires the ordering proof, the total admissibility enumeration, and the concurrency sketch; those override the size cap for this change.

## Architecture Decisions

### AD-1 — Seam revocation: a mutex-serialized latch that covers the send (the CRITICAL hazard, decided)

**Choice**: revocation is a `sync.Mutex`-guarded boolean latch on the seam value, and **the channel send happens under the same mutex**:

```go
func (s *delegationSeam) Publish(ev Event) error {
    if err := admissible(ev); err != nil { return err }
    s.mu.Lock()
    defer s.mu.Unlock()
    if s.revoked { return ErrDelegationRevoked }
    s.emissions <- emission{ev: ev}   // send under the lock — deliberate
    return nil
}
func (s *delegationSeam) revoke() { s.mu.Lock(); s.revoked = true; s.mu.Unlock() }
```

`executeCall` installs and revokes in three lines immediately around the tool run (currently `scheduler.go:492`):

```go
seam := newDelegationSeam(runID, turnID, emissions)
defer seam.revoke()
reply, detached := s.runToolWithWindDown(withDelegationSeam(ctx, seam), tool, runArgs, PolicySlot(call.ID()))
```

**Ordering proof** (the detached goroutine cannot win the race):

1. `revoke()` is deferred in `executeCall`, so it runs in the scheduler-owned call goroutine on **every** exit: the normal return, the detach return (`scheduler.go:496-500`), the re-panic path (`scheduler.go:493-494` — defers run during unwinding, and this defer is registered *after* `defer recoverCall` at `scheduler.go:424`, so LIFO runs revocation first), and every later `return`.
2. Program order in that goroutine then gives the chain: `revoke()` returns → `executeCall` returns → `scheduleRead`/`scheduleSerialized`'s `defer wg.Done()` runs (`scheduler.go:326,354`) → `wg.Wait()` returns (`scheduler.go:219`) → `close(emissions)` runs (`scheduler.go:240`).
3. Mutex dichotomy: any `Publish` — including one from the tool goroutine `runToolWithWindDown` abandoned at `scheduler.go:621-627`, whose `tool.Run` frame outlives the call — either (i) acquires `mu` **before** `revoke()` does, in which case its send completes before `revoke()` returns and therefore, by step 2, strictly before `close(emissions)`; or (ii) acquires `mu` **after**, observes `revoked == true`, and returns `ErrDelegationRevoked` without touching the channel. There is no third interleaving, because the latch write and the send are serialized by the same mutex.
4. The under-lock send cannot deadlock `revoke()`: while any call goroutine is alive, `close(emissions)` has not run and `runDispatcher` is still draining (`scheduler.go:288-297`), and `sink` is drained by the harness's live per-attempt forwarder (`harness.go:616-638`), so a blocked send always completes.
5. Why not weaker primitives: an atomic flag makes check-then-send non-atomic — the publisher can be descheduled between the load and the send while revoke + `close(emissions)` slide in between, and the send panics. A `select` on a `done` channel does not help either: Go has no non-panicking send on a closed channel, even inside `select`. The lock must cover the send itself.

A `Publish` racing revocation therefore returns the typed `ErrDelegationRevoked` — never a panic, never a silent drop (the error return is the caller's signal).

**Alternatives rejected**: atomic bool (step 5); closed-`done`-channel select (step 5); revoking inside `runToolWithWindDown`'s detach arm only (misses the normal-return leak case: a tool that spawns its own goroutine holding the seam and returns).

### AD-2 — Admissibility: descriptor-derived, made total with a registry-membership gate (proposal rule tightened)

**Choice**: `Publish` admits an event iff, in order: (1) `eventRegistryEntry(ev.Kind())` exists; (2) `Bracket == BracketRoleNone`; (3) `Cardinality != CardinalityAtMostOne`; (4) `Terminal == false`; (5) `ev.Kind() != EventKindCostTurn`. Any failure returns `ErrDelegationInadmissible`.

**Correction to the proposal**: gate (1) is new, and load-bearing. A zero `Event` reports kind 0 (`event.go:469-474`), has no registry row (`event.go:372-377`), and under the proposal's pure descriptor predicate its zero-value descriptor fields (`BracketRoleNone`, `CardinalityAny`, `Terminal:false` — zero values per `event_descriptor.go:70-72,102-104,115-118`) would **admit** it — and `CheckStream` would then silently skip it (`stream_check.go:112-118`), leaving junk on a "valid" parent stream. Without gate (1) the rule is not total; with it, every possible `Event` value is decided.

**Total enumeration** over all 25 registered kinds (`event.go:100-221,242-366`):

| Kind(s) | Descriptor evidence | Verdict |
|---|---|---|
| `run_start` | `BracketRoleOpensRun` (`event.go:243-246`) | refuse (2) |
| `run_end` | `BracketRoleClosesRun, Terminal:true` (`event.go:247-250`) | refuse (2) and (4) |
| `turn_start`, `turn_end` | `OpensTurn`/`ClosesTurn` (`event.go:251-258`) | refuse (2) |
| 6 message kinds | `PlacementTurn`, rest zero (`event.go:259-282`) | **admit** |
| 5 tool kinds | `PlacementTurn` (`event.go:285-304`) | **admit** |
| `permission_decision_required`, `permission_decision_made` | `PlacementTurn, Terminal:false` (`event.go:312-319`) | **admit** (the ask/answer pair crosses together) |
| `permission_resolution_remembered` | `CardinalityAtMostOne` (`event.go:320-323`) | refuse (3) |
| `cost_turn` | `PlacementTurn, Terminal:false` (`event.go:329-332`) | refuse (5) — R-CST-004 guard |
| `cost_session` | `PlacementRun, Terminal:false` (`event.go:333-336`) | **admit** — AD-3 |
| `subagent_started`, `subagent_ended` | `PlacementTurn, Terminal:false` (`event.go:340-347`) | **admit** (the seam's raison d'être) |
| 3 compaction kinds | `PlacementTurn, Terminal:false` (`event.go:354-365`) | **admit** — a child's compaction is legitimately visible to the human |
| unregistered / zero kind | no registry row | refuse (1) |

Refused: 6 kinds + non-members. Admitted: 19 kinds. **Asymmetry stated loudly**: `cost_turn` is the one refusal `CheckStream` would happily accept — it is a requirement-level rule (`R-CST-004`, `agent-cost-events:102-119`), not a stream-legality rule, and it exists because the funnel terminates in the exact channel whose forwarder folds by payload type with no run filter (`harness.go:633-635`). Gate (4) is currently implied by gate (2) — the only `Terminal:true` kind is `run_end`, which is also `ClosesRun` — and is kept anyway as defense against a future `Terminal` `BracketRoleNone` kind.

### AD-3 — The child's `cost_session(Final)` DOES cross onto the parent's stream (open question 2, decided: yes)

Both of the proposal's claims verified directly:

1. **Stream-legal**: `cost_session` registers `{Placement: PlacementRun, Terminal: false}` with **zero-value Cardinality** (`event.go:333-336`), and the zero value is `CardinalityAny` (`event_descriptor.go:115-118`) — **not** `CardinalityAtMostOne`. So two `cost_session` events on one validated slice (the child's crossed one plus the parent's own final) trip neither the cardinality rule (`stream_check.go:166-171`) nor placement (the default branch checks only `PlacementTurn` against `turnOpen`, `stream_check.go:157-163`; `PlacementRun` inside the parent's open turn passes). The `ErrDuplicate` worry dissolves: the answer does not flip.
2. **Cannot reach the parent's `total`**: the per-attempt forwarder folds only `ev.CostTurn()` (`harness.go:633-635`); a `CostSession` payload never matches.

**How a consumer distinguishes the two**: by `Event.Run()`. `LaneStamper.Stamp` rewrites only `seq` (`sequence.go:54-58`), so the crossed child event still carries `Run() == childRunID` while the parent's own final (emitted by `windDownRun`/`failRun`/the success path via `sendStamped`, e.g. `harness.go:343-345`) carries `Run() == parentRunID`. The crossed `cost_session` is the natural carrier of the frontend's "this subagent cost X" on the one stream a human watches — labels alone are not the discriminator; run identity is, and the spec must say so.

### AD-4 — Two sentinel errors, matching house convention

**Choice**: `ErrDelegationInadmissible` and `ErrDelegationRevoked`, plain sentinels tested via `errors.Is`. **Rejected**: one struct error with a reason field. **Rationale**: a tool must distinguish a programming error (publishing a kind that can never cross — fail the test) from a timing outcome it must tolerate (the call is over — stop publishing). Sentinels match `ErrStrayDecision` (`scheduler.go:269`), `ErrInterrupted`/`ErrShutdown`, and the package's `errors.Is` discrimination idiom.

### AD-5 — The seam is an interface over an unexported, unforgeable struct

**Choice**: export `DelegationSeam` (interface) and `DelegationSeamFrom`; keep `delegationSeam`, `delegationSeamKey`, `newDelegationSeam`, `withDelegationSeam`, `admissible`, `revoke` unexported. **Rationale**: no code outside the package can construct or install a seam — only `executeCall` mints one — so a "production subagent tool" cannot even be wired up without going through the scheduler. The charter's fence is compiler-enforced twice: the delegating tool lives in `package agent_test` (the `ScriptedTool` precedent, `scripted_tool_test.go:1-55`), and the seam itself is unmintable from outside. The ctx-carried handle follows the in-package precedent `ctxMarkerKey struct{}` (`loop_test.go:473-486`). The `PolicySlot` source guard is untouched: it scans only `scheduler.go` (`scheduler_test.go:626`) for the four assertion shapes (`:637-642`); the seam's single type assertion (`ctx.Value(...).(DelegationSeam)`) lives in `delegation_seam.go`, and `scheduler.go` gains zero assertions — even the `context.WithValue` text lives behind `withDelegationSeam`.

### AD-6 — Nested wind-down: generous parent bound, ordering proven by event order (proposal Q6 confirmed)

**Choice**: the AG-19.2 test sets a generous `Scheduler.WindDownBound` on the parent (seconds — a ceiling, never a synchronization point; AG-14 precedent for injecting a bound). The `DetachedCallError` framing is not asserted; it stays a named non-requirement owned by `R-CAN-006`/`R-TLS-014`.

**Concrete assertions** for "the child winds down first … and then the parent's wind-down completes":

1. `errors.Is(childRunErr, ErrInterrupted)` — the parent's interrupt cause reached the child transitively: child `runCtx` derives from the tool's `ctx` (`harness.go:434`), which derives from the parent's `runCtx`; `context.Cause` propagates the ancestor's cause.
2. Child stream: `CheckStream`-valid; its `cost_session(Final)` precedes its terminal `run_end(RunOutcomeInterrupted)` (the `windDownRun` order, `harness.go:343-348`).
3. Parent stream: `CheckStream`-valid; `index(subagent_ended) < index(parent's run_end)`. This is load-bearing, not decorative: the tool publishes `subagent_ended` only after the child's sink closed (the child `Run`'s `defer close(sink)`, `harness.go:446`, runs only after its `run_end` was sent), so parent-lane order structurally proves the child's wind-down completed before the parent's did. The parent lane is single-stamped and contiguous, so index comparison is sound.
4. The parent's tool result is **not** a `DetachedCallError` (`errors.As` returns false) — the call completed inside the bound.

No assertion reads elapsed time; synchronization is channel closes and observed event order.

### AD-7 — Derived permission scope: a test-only narrowing composition

**Choice**: `derivedScope struct{ parent agent.PermissionPolicy; ... }` in `package agent_test`, implementing `PermissionPolicy` (`permission_protocol.go:80-94`) by delegating `Resolve`/`Remember` to the parent's policy. **Operational meaning of "what the parent's policy allowed flows down"**: the derived scope may only *narrow* (map a parent Allow to Deny/Defer for tools outside the child's grant) and never *widen* (never map a parent Deny/Defer to Allow); the test asserts both directions. **Ask up**: the child's `permission_decision_required` is emitted on the child's stream by the child scheduler and mirrored through the seam (admissible, AD-2) — the parent's stream is the one place a human watches, discharging `S-AGE-022` with zero production routing surface. **Decision down**: the test, playing the human on the parent's stream, calls the child `Scheduler.WakeParked(callID)` (`scheduler.go:264-272`); the gate re-enters `Resolve` on wake (`scheduler.go:766-777`) and the derived scope returns the human's verdict; the resulting `permission_decision_made` mirrors up too (the pair crosses together). No new Layer 2 type: "scope" stays out of Layer 2 (`permission_protocol.go:78-79`).

## Data Flow

    child Harness.Run ──childSink──▶ delegating tool (agent_test)
                                        │  translate: NewSubagentStarted/Ended;
                                        │  mirror admissible child events (AD-2)
                                        ▼
                              seam.Publish (mutex latch, AD-1)
                                        ▼
      parent executeCall ─────▶ emissions ──▶ runDispatcher ──Stamp──▶ sink ──▶ forwarder ──▶ consumer
                                              (single stamping writer,          (folds only CostTurn,
                                               scheduler.go:281-298)             harness.go:633-635)

**Sequence/lane discipline**: a mirrored event gets its parent-lane sequence from the one dispatcher goroutine (`stamper.Stamp(em.ev)`, `scheduler.go:289`); `Stamp` discards the prior stamp and returns a copy (`sequence.go:53-58`). The child's own lane is unharmed because `Event` is a value type — the tool publishes a copy; the child's captured slice keeps its own contiguous stamps from the child's own `LaneStamper`. Each lane keeps exactly one stamping writer (`sequence.go:13-24`): the parent's stamper is touched only by `runDispatcher`; the child's only by the child's own dispatcher/harness. Two lanes, both contiguous, never merged.

**Concurrency sketch (AG-19.1 siblings, `-race`)**: N read-class tool goroutines each hold a *distinct* `*delegationSeam`. The seams share no memory except the `emissions` channel; each mutex guards only its own latch; a multi-producer channel send is the synchronization point under the Go memory model; exactly one goroutine reads `emissions` and stamps. Each child harness is a distinct value with its own `runCtx`, `History`, stamper, and sink — no shared state exists that `-race` could flag beyond the already-race-proven funnel sibling tool calls use today. Cross-talk is impossible by construction: attribution is `Event.Run()`, and no seam ever carries another child's events.

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/delegation_seam.go` | Create | `DelegationSeam`, `DelegationSeamFrom`, two sentinels, `admissible`, revocation latch (~130-190 lines) |
| `backend/agent/src/agent/scheduler.go` | Modify | 3 lines + comment in `executeCall` around `runToolWithWindDown`; zero type assertions |
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | Create | delegating tool, derived scope, two-stream reconstruction helper, scenarios, bites |
| `doc.go` | **Not edited** (decision) | The invariant-2 sentence (`doc.go:45-50`) stays true when AG-19 ships (per the proposal's verification); skipping the edit keeps `R-LSK-004`'s exact-filename list untouched — no release needed |
| `event.go`, `event_descriptor.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `go.mod`, `go.sum`, all of `src/ai/` | NOT TOUCHED | Assertable byte-unchanged; no new `EventKind` |

## Interfaces / Contracts

```go
// delegation_seam.go — the WHOLE exported surface.

// DelegationSeam is the re-entrancy seam a tool reaches through its
// context: the one sanctioned door from inside tool.Run back onto the
// parent's event lane. It names no subagent concept.
type DelegationSeam interface {
    // Parent reports the identity of the run and turn hosting this
    // tool call — the identities a tool cannot otherwise obtain, and
    // without which NewSubagentStarted is unconstructible.
    Parent() (RunID, TurnID)
    // Publish offers one event to the parent's lane. It returns
    // ErrDelegationInadmissible or ErrDelegationRevoked; a nil return
    // means the event reached the dispatcher funnel.
    Publish(ev Event) error
}

// DelegationSeamFrom returns the seam installed for this tool call,
// and whether one is installed.
func DelegationSeamFrom(ctx context.Context) (DelegationSeam, bool)

var ErrDelegationInadmissible error // kind may never cross (AD-2)
var ErrDelegationRevoked error     // the hosting call has completed or detached (AD-1)
```

Every identifier's export justification: `DelegationSeam` and `DelegationSeamFrom` are consumed from `package agent_test` (an external package by construction) and later by Layer 3 tools — unexported is unreachable there. The two sentinels must be `errors.Is`-testable by those same callers. Nothing else is exported; the surface reads as a publishing seam, not a subagent contract (no lifecycle, no config, no depth, no child anything).

## Testing Strategy

| Test | Proves (Gherkin / req) | RED staged first |
|---|---|---|
| T1 seam unit: 25-kind admissibility table + zero-Event refusal + revoked refusal, driven through a `ScriptedTool` capturing the seam | R-DEL-001/002 | new-surface test; RED recorded per house overlay protocol |
| T2 nested run + walkable tree, both streams validated separately | AG-19.1 sc. 1 | seam file lands first, `executeCall` install last: `DelegationSeamFrom` returns false → no `subagent_started` on the parent stream → clean RED without overlay |
| T3 sibling children, two distinct Harness values, `-race` | AG-19.1 sc. 2 | same install-last RED |
| T4 nested cancellation, AD-6's four assertions | AG-19.2 | bite (d): child built on `context.Background()` → child never cancels → assertion 1 fails |
| T5 cost reconstruction, strict inequality, non-zero child spend | AG-19.3 sc. 1 | bite (a): admit `cost_turn` → mirrored spend folds at `harness.go:633-635` → parent cumulative inflated → inequality fails, visibly |
| T6 derived scope, ask-up/decision-down via `WakeParked` | AG-19.3 sc. 2 | without mirroring, the ask never appears on the parent stream → RED |
| Bite (b) | R-DEL-004 guard | admit `run_start` → parent `CheckStream` fails `ErrDuplicate` (`stream_check.go:122-127`) |
| Bite (c) | R-DEL-002 | drop revocation → detached tool publishes after `close(emissions)` → **panic**. Staging: tiny injected `WindDownBound`, tool ignores ctx, test gates the publish on a channel it closes only after `Schedule` returned (so `emissions` is provably closed, `scheduler.go:240` precedes `:249`). **Flagged hard**: the unrecovered panic in the detached goroutine crashes the whole test binary — record this RED from an isolated `go test -run` invocation, `-count=1` |

Evidence runs: `cd backend/agent && make test` semantics with `-count=1` forced, wall-clock recorded.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The seam moves in-process `Event` values over an existing in-process channel.

## Migration / Rollout

No migration. Rollback is the proposal's single revert: delete `delegation_seam.go` and the test files, drop 3 lines from `executeCall`.

## Open Questions

None. Both proposal-deferred questions are decided (AD-1, AD-3); the proposal's admissibility predicate is tightened, not overturned (AD-2 gate 1); Q6 is confirmed with concrete assertions (AD-6); the `doc.go` edit is declined, so the conditional `agent-loop-skeleton` delta does not fire.
