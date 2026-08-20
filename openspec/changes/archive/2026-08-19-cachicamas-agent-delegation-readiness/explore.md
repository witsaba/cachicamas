# Exploration — `cachicamas-agent-delegation-readiness` (AG-19)

Milestone AG-19 (Layer 2 Wave 5, milestone 19 of 24), doc 0003 lines 1793-1862. Worktree: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ag19`. Store: hybrid — this file is the OpenSpec half; the Engram half is topic key `sdd/cachicamas-agent-delegation-readiness/explore`.

## 1. AG-06.3 — the delegation event family (constructible, unemitted)

`backend/agent/src/agent/delegation_events.go`:
- `SubagentStarted struct{ subagentID string }` (line 40), kind `EventKindSubagentStarted`.
- `SubagentEnded struct{ subagentID string }` (line 114), kind `EventKindSubagentEnded`.
- `func NewSubagentStarted(run, parent RunID, turn TurnID, subagentID string) (Event, error)` (line 62): `run` = the subagent's OWN run identity; `parent` = the delegating run's identity. Sets `Event{payload, run, turn, hasTurn:true, parent, hasParent:true}`.
- `func NewSubagentEnded(run, parent RunID, turn TurnID, subagentID string) (Event, error)` (line 134): identical shape.
- Both accessors: `Event.SubagentStarted() (SubagentStarted, bool)` (line 93), `Event.SubagentEnded() (SubagentEnded, bool)` (line 165).
- Registry rows (`event.go:340-347`): both kinds are `Placement: PlacementTurn`, `Bracket: BracketRoleNone` (zero value, implicit), `Cardinality: CardinalityAny`, `Terminal: false`.
- Doc comment (delegation_events.go:1-26) states AG-06.3 is "the FIRST non-[NewDelegatedRunStart] consumer" of the envelope's parent field.
- Spec: `openspec/specs/agent-protocol-events/spec.md` R-APE-006 (line 102-110): "A `subagent_started` event and a `subagent_ended` event MUST set the envelope's parent identifier... making the delegation tree walkable." Scenarios S-APE-050/051/052.
- No emission mechanism exists anywhere in the codebase — grep for callers outside `delegation_events.go`/its own test file returns zero production call sites. AG-06.3's charter (doc 0003:670-681) states "Out of scope: Emission (AG-19...)".

## 2. Parent identification on the envelope

`backend/agent/src/agent/event.go`:
- `Event` struct (line 454-462): `{payload eventPayload; seq Sequence; run RunID; turn TurnID; hasTurn bool; parent RunID; hasParent bool}`.
- `func (e Event) Parent() (RunID, bool) { return e.parent, e.hasParent }` (line 492): "reports the run identity this event was delegated from... A top-level event reports 'no parent' as this distinguishable false, never an ambiguous zero value (R-AEV-003, S-AEV-021)."
- **The envelope already has the parent field — this is NOT an amendment AG-19 needs to make.** It was reserved at AG-04.1 (`run_events.go:60-76`, `NewDelegatedRunStart`) specifically so nesting "cannot be retrofitted" (S-AEV-022).
- **Only THREE constructors in the entire event vocabulary (19+ kinds) ever set `.parent`**: `NewDelegatedRunStart` (run_events.go:68), `NewSubagentStarted` (delegation_events.go:62), `NewSubagentEnded` (delegation_events.go:134). Every other constructor — message, tool, permission (`NewPermissionDecisionRequired(run, turn, callID, name, arguments)` at `permission_events.go:138`, `NewPermissionDecisionMade` at :242, `NewPermissionResolutionRemembered` at :326), cost (`NewCostTurn`/`NewCostSession`, `cost_events.go:193,296`), turn events — accepts NO parent parameter and never sets `hasParent`.
- **Consequence**: "each child event parent-identified" (the AG-19.1 Gherkin phrase) can only be a DERIVED/transitive property (a consumer walks: event's `Run()` == childRunID → find the `subagent_started` event whose `Run()` == childRunID → read ITS `Parent()` == parentRunID), never a literal per-event `.parent` field, without rewriting every other event constructor in the package (far outside AG-19's declared dependency set: AG-02, AG-06, AG-10, AG-13, AG-14, AG-16). This matches the Gherkin's own second clause: "a consumer separates the two conversations by walking parents."
- `doc.go:47-50` (the machine-checked layer-contract paragraph) already names this precisely: "AG-04 co-closes... invariant 2 with AG-19.1" — and the pre-v2 archive (`docs/architecture/milestones/archive/0003-...-v1.md:1071`) labels invariant 2 "explicit nesting," test-listed at v1:375 and v1:870 with the identical "walking parents" phrasing. AG-19.1 is the pre-planned closer of an already-declared envelope invariant, not new scope.

## 3. Lane / ordering law — `CheckStream` is a single-bracket, flat-slice validator

`backend/agent/src/agent/stream_check.go` (`CheckStream`, lines 92-185):
- Tracks exactly ONE `runBracketState` (unopened/open/closed) and ONE `turnOpen bool` — **never reads `Event.Run()` anywhere in the function body.** It validates a flat, single-conversation slice.
- `seqExpect` is a single running counter over the WHOLE slice (line 93, 103-106): contiguous 1-based, checked before any kind-specific logic.
- `BracketRoleOpensRun`/`BracketRoleClosesRun` are checked GLOBALLY: "opening the run bracket twice... are all rejected" (doc comment lines 79-82; code lines 122-135) — a SECOND `EventKindRunStart`-shaped event anywhere in the slice, even one belonging to a different `run` id, is `ErrDuplicate`.
- `CardinalityAtMostOne` (lines 166-171) is also tracked globally by `EventKind` alone, via `seenAtMostOnce map[EventKind]bool` — not scoped by run. `EventKindPermissionResolutionRemembered` is registered `CardinalityAtMostOne` (event.go:174-177, R-APE-003) — a SECOND occurrence of that kind anywhere in one validated slice fails, regardless of which run it belongs to.
- `sequence.go:1-58` (`LaneStamper`) documents that "two lanes' sequences overlap by construction (both start at 1)" and explicitly anticipates multiple independent lanes: "If a later milestone observes a lane fed by more than one writer, that finding amends AG-01 § 5 first" (sequence.go:20-24).
- **Conclusion**: `CheckStream` cannot be called, unmodified, against a single flat slice that interleaves a parent's own bracket verbatim with a child's own raw `RunStart`/`RunEnd` (two `BracketRoleOpensRun` events → `ErrDuplicate`) or with two independent `permission_resolution_remembered` events. The validator needs NO code change, but the **reconstruction algorithm** a consumer/test must use is: (a) validate the PARENT's own physical/delivered stream as today — `subagent_started`/`subagent_ended` sit inside it as ordinary `PlacementTurn`, `BracketRoleNone` events (their own `Run()` reports the CHILD's id, which `CheckStream` ignores, so this is harmless to the parent's own bracket); (b) separately capture and independently `CheckStream`-validate the CHILD's own emitted stream (its OWN real `RunStart`→...→`RunEnd`, unmodified, from the child `Harness.Run()` call), which must never be merged verbatim onto the parent's own validated slice. "The consumer separates the two conversations by walking parents" is therefore the load-bearing mechanism, not an incidental nicety — CheckStream's existing invariants make any other interpretation invalid.

## 4. Constructibility from inside a tool — the concrete gap

`backend/agent/src/agent/tool.go:182-186`:
```go
type Tool interface {
    Name() string
    EffectClass() EffectClass
    Run(ctx context.Context, args []byte, policy PolicySlot) (Result, error)
}
```
- `Tool.Run` receives ONLY `ctx`, `args`, `policy PolicySlot` (`PolicySlot = any`, tool.go:117). **No sink, no stamper, no scheduler handle, no way to write an event or reach the parent's stream at all.**
- `Harness` is value-form, "zero surface beyond two methods, `Run` and `Steer`" (harness.go:12-13) — trivially constructible from anywhere (`&Harness{...}`), so a tool CAN build a child harness value with zero new plumbing. The gap is entirely about **emitting the child's activity onto anything the parent's own consumer observes.**
- The scheduler DOES already have the sink/stamper in scope at the exact call site: `Scheduler.Schedule(ctx, calls, reg, runID, turnID, policy PermissionPolicy, stamper *LaneStamper, sink chan<- *Event)` (scheduler.go:131-140) receives them directly; `executeCall` (scheduler.go:401-413) receives `emissions chan<- emission` (an unexported per-Schedule channel) and uses it to enqueue `ToolStart`/`ToolEnd*` (e.g. line 464 `emissions <- emission{ev: startEv}`); a single `runDispatcher` goroutine (scheduler.go:281-298) is the ONLY writer that stamps (`stamper.Stamp(em.ev)`, line 289) and sends onto `sink` (line 290) — the existing "many call goroutines funnel through one channel, one stamping goroutine" pattern Schedule already uses for concurrent sibling tool calls.
- `executeCall` calls `tool.Run(ctx, args, PolicySlot(call.ID()))` (scheduler.go:492) — **in the current wiring, `PolicySlot`'s concrete value is just `call.ID()` as a string**, not the `PermissionPolicy`; "Layer 2 does not interpret it; the tool or a Layer 3 sandbox does" (scheduler.go:466-471 comment).
- `scheduler.go`'s own file-header (lines 27-28) documents a source-guard test: "`scheduler.go` contains zero type assertions on `PolicySlot` (the seam 3 promise — Layer 2 never reads the value)." **`PolicySlot` is therefore the wrong vehicle for a delegation handle** — reusing it for a second, unrelated purpose (event forwarding) would overload a seam whose single stated meaning is the permission slot, and risks tripping the spirit (if not the letter) of that guard.
- **No `context.WithValue` injection of any kind exists in `scheduler.go` today** (grep confirms zero matches) — there is no existing "reach the parent from inside a tool" precedent to reuse; AG-19 would be the first to add one.

## 5. Cancellation tree — propagation is automatic; wind-down ordering is a genuine nested race with a documented bound

`backend/agent/src/agent/harness.go:434` (`Run`): `runCtx, cancel := context.WithCancelCause(ctx)` — derives the run's own cancel-cause context from the CALLER's `ctx` (the same `ctx` a tool receives via `tool.Run(ctx, ...)`, per scheduler.go:492, `runToolWithWindDown(ctx, tool, runArgs, ...)` at scheduler.go:588-601, itself invoked with the `ctx` threaded down from `Schedule`→`Turn`→`Harness.Run`'s own `runCtx`).
- **Consequence: if a test-only tool constructs a child `Harness` and calls `child.Run(ctx, prompt, childSink)` passing the EXACT `ctx` it received from `tool.Run`, Go's context tree gives cancellation propagation for free** — no new plumbing needed. When the parent's `Interrupt()`/`Shutdown()` cancels the parent's own `runCtx` (via `h.cancelRun`, harness.go:411-436), that cancellation is the ancestor of the tool's `ctx`, which is the ancestor of the child's own `context.WithCancelCause(ctx)` — `context.Cause` propagates transitively down that whole derived chain.
- `windDownRun` (harness.go:317-350): synthesizes orphans (`hist.SynthesizeOrphans()`), closes the turn, emits the run-scoped final `cost_session`, then `run_end(RunOutcomeInterrupted|RunOutcomeShutdown)`.
- The nested race: the PARENT's own call to the tool is itself bounded by `runToolWithWindDown`'s wind-down bound (`scheduler.go:588-630`, default `defaultWindDownBound = 100 * time.Millisecond`, `cancellation.go:86-92`, overridable via `Scheduler.WindDownBound`, `tool.go:243-256`). If the CHILD's own wind-down (`windDownRun` — orphan synthesis, cost emission, run_end) takes longer than the PARENT's wind-down bound, the parent scheduler detaches the tool call (`typedDetachedCallFailure`, scheduler.go:1126-1134) — the child's wind-down would still complete, but the tool's own call is reported as a `DetachedCallError` on the parent's stream rather than as an ordinary tool result. **This is a structural coupling the design/test must account for** (e.g., set a generous `Scheduler.WindDownBound` on the parent for the nested-cancellation scenario, or assert the detached-call framing explicitly as a documented edge behavior).
- `openspec/specs/agent-cancellation-tree/spec.md:181` (Explicit non-requirements table): "Subagent cancellation inheritance | **AG-19.2.**" — the AG-14 spec deliberately did not claim this; AG-19.2 is its designated closer, confirming this is in-scope, expected work, not scope creep.
- `openspec/specs/agent-cancellation-tree/spec.md:188`: "Concurrent runs on one harness value | **Not this milestone.**" — AG-19's "two sibling tools each hosting a child run" scenario uses TWO DISTINCT `Harness` VALUES, never two concurrent `Run()` calls on the SAME value, so it does not collide with this non-requirement.

## 6. Cost aggregation — does NOT fold automatically; folding it would violate a shipped requirement

`backend/agent/src/agent/harness.go:478`: `var total costAccumulator` — **a local variable in `Run`'s own stack frame**, "never a Harness field (R-CAN-002)" (comment at 470-478).
`backend/agent/src/agent/cost_usage.go:101-108`: `costAccumulator` doc comment: "a local in `Harness.Run`'s own stack frame, never a Harness field (R-CAN-002) — Harness is value-form, serially reused, and carries no cross-run state."
`harness.go:614-638` (the per-attempt forwarder goroutine): drains a fresh `turnSink` created per attempt (line 614), and for every event carrying a `CostTurn` payload — **checked purely by payload type via `ev.CostTurn()`, with NO run-identity filter at all** — calls `total.add(ct)` (line 634) before forwarding the event onward with `sink <- ev` (line 636).
- **A CHILD harness's `total` is a wholly separate local variable in the CHILD's own `Run` stack frame — it never touches the PARENT's `total` by any existing mechanism.** Verified: this reproduces exactly the memory's prior finding ("the `total.add(ct)` fold runs in a goroutine draining `turnSink`... spend does NOT fold automatically for anything outside that path") and confirms it is STILL true at this commit.
- **Requirement collision (the load-bearing finding for AG-19.3)**: `openspec/specs/agent-cost-events/spec.md` R-CST-004 (lines 102-119): "The run-scoped cumulative figure a `cost_session` carries MUST be defined as the sum over every `cost_turn` event emitted **within that run bracket**... The cumulative state MUST be **run-scoped**: it MUST NOT outlive one run, and it MUST NOT be carried on the harness value... **Accumulation MUST NOT introduce a second writer to the run's event path.**" Line 171: "Aggregating a delegated or subagent run's cost into a parent run is **AG-19**'s, by the charter's own Goal line (0003:1533). **Nothing in this capability may be written as if a parent scope exists.**" Line 196 (Explicit non-requirements): "Delegated / subagent cost aggregation into a parent run | **AG-19**...".
- **Consequence**: any design that literally folds a child's `cost_turn` figures into the PARENT Harness's own `total`/`cost_session` (i.e., makes the parent's own `cost_session(Final)` event's figures include child spend) would violate R-CST-004's already-shipped, machine-tested "sum over every `cost_turn` emitted WITHIN THAT RUN BRACKET" constraint, since the child's `cost_turn` events are emitted within the CHILD's own run bracket, not the parent's. **"Child cost aggregates into the parent's cumulative figures" (the AG-19.3 acceptance line) can only be satisfied as a CONSUMER-SIDE reconstruction** — a test (or Layer 3 frontend) walks both streams by parent identity and computes the sum itself — never as a Layer-2-internal fold. This exactly matches the milestone's own qualifying clause: "a frontend can show both 'this subagent cost X' and 'the run cost Y'" (two DISPLAYED numbers, not one merged Layer-2 event).
- **Second, sharper risk**: if a future design chooses to forward the child's raw `cost_turn` events through the PARENT's own per-attempt `turnSink` (rather than through `Schedule`'s `emissions` channel or a wholly separate channel), they would be captured by `harness.go:633-635`'s `ev.CostTurn()` check UNCONDITIONALLY (it does not check `ev.Run()`), silently and automatically folding the child's tokens into the parent's own `total` — directly violating R-CST-004. **The chosen forwarding channel/point is therefore not a free implementation choice: it must bypass this specific capture point for `cost_turn`-kind payloads.**

## 7. Permission scope derivation — "scope" is not a Layer 2 type; PermissionPolicy is a plain interface

`backend/agent/src/agent/permission_protocol.go:80-94`:
```go
type PermissionPolicy interface {
    Resolve(ctx context.Context, call ai.ToolCall) PermissionVerdict
    Remember(ctx context.Context, toolName string, outcome PermissionOutcome) bool
}
```
- No "scope" struct, interface, or derivation mechanism exists anywhere in Layer 2. "Layer 2 MUST NOT define rule sets or mode flags — those are out of scope (AG-10.0 non-reqs)" (comment, lines 77-79). Policy content itself is Layer 3's (doc 0004, CO-03).
- `openspec/specs/agent-permission-protocol/spec.md:169`: "Subagent tool scope | **AG-19.3.**" (Explicit non-requirements table) — AG-10's charter itself deliberately left this open (doc 0003:1015) and the spec confirms it.
- Permission event constructors (`NewPermissionDecisionRequired(run, turn, callID, name, arguments)`, permission_events.go:138; `NewPermissionDecisionMade`, :242; `NewPermissionResolutionRemembered`, :326) accept NO parent parameter — same gap as §2 above: a child's own `permission_decision_required` cannot be literally parent-stamped; it is only transitively attributable via the `subagent_started`/`subagent_ended` bracket, same as every other child event.
- **Consequence**: "a scope derived from the parent's policy" is achievable ENTIRELY IN TEST CODE with zero Layer 2 production changes — a test constructs an ordinary Go value implementing `PermissionPolicy` that wraps/narrows the parent's own `PermissionPolicy` (composition, not a new Layer 2 type), and passes it as the CHILD `Scheduler.Schedule`'s own `policy PermissionPolicy` argument. "What it would ask about is asked on the parent's stream — one place a human watches" (the second AG-19.3 scenario) requires the CHILD's `permission_decision_required` events to be forwarded onto whatever merged/observed stream a human watches — the SAME forwarding mechanism §3/§4 need for ordinary events, no new mechanism specific to permission.

## 8. Test-only tool precedent (AG-07/AG-09 substrate)

`backend/agent/src/agent/scripted_tool_test.go:1-55`:
```go
// The tool lives in `package agent_test` (external posture, NFR-TLS-001).
// Layer 1's `agenttest` package cannot import the `agent` package
// (ADR 0005 § D1 row 1 — Layer 1 must not import Layer 2), so the
// scripted tool is a test-only file in the agent package's external
// test surface.
package agent_test

type ScriptedTool struct {
    toolName string
    Effect   agent.EffectClass
    Script   func(ctx context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error)
    // ...recorded invocation state...
}
func (s *ScriptedTool) Name() string { return s.toolName }
func NewScriptedTool(name string, effect agent.EffectClass, result agent.Result) *ScriptedTool { ... }
```
- **Precedent for AG-19's own "test-only tool hosting a child harness"**: a NEW `_test.go` file in `package agent_test` (same package as `scripted_tool_test.go`), implementing `agent.Tool`, constructed with `agent.Harness`/`agent.Scheduler` values directly. This keeps it entirely inside the external test surface — it can NEVER be imported by production code, satisfying "no subagent tool ships in v1" by construction (the compiler enforces it, not a convention).
- `agenttest.Provider`/`agenttest.NewProvider(scripts ...Script)` (agenttest/fake_provider.go:64) is the scripted-conversation substrate a child `Harness.Provider` would use — same fake used throughout AG-13+ harness tests.

## 9. Package boundary guards (AG-03)

`backend/agent/src/agent/doc.go` — the machine-checked layer contract (rows parsed back and diffed against `doc_contract_guard_test.go`):
- L2C-01/02: import allowlist + no-ambient-authority — AG-19's mechanism (context propagation, channel forwarding, event construction) is pure Go stdlib + existing `agent`/`ai` types; nothing it plausibly needs (no I/O, no new external import) risks tripping these.
- L2C-06: "Permission, cost, delegation, and compaction families are constructible on the event stream... the parent identifier envelope field (R-APE-006)... belong in this package's prose and per-family files, not in the guarded row" — AG-19 EMITS what AG-06.3 already made constructible; it does not need to touch this row's text.
- **doc.go:47-50 already pre-declares AG-19.1 as invariant 2's co-closer** — meaning AG-19.1 IS expected to touch this specific paragraph (adding the proof, not new vocabulary) once emission ships; this is anticipated maintenance, not scope creep, and should be listed as an explicit task.
- No `L2C-*` row needs a NEW row for AG-19 under the current wording — the existing L2C-06 row already covers the delegation family's constructibility; AG-19 only needs to update doc.go's *narrative* (not the guarded table) to record the invariant-2 closure, exactly as prior milestones amended surrounding prose without new rows.

## 10. Requirement-collision sweep

| id | file:line | clause | AG-19 interaction |
|---|---|---|---|
| CheckStream bracket engine | `stream_check.go:92-185` | single global run bracket, global `CardinalityAtMostOne`, single flat sequence counter | Forbids literal verbatim merge of a child's own `RunStart`/`RunEnd` onto the parent's own `CheckStream`-validated slice (§3). No spec-text change needed; a per-run reconstruction step is mandatory in the consumer/test, and should be documented. |
| R-APE-006 | `openspec/specs/agent-protocol-events/spec.md:102-110` | "A `subagent_started` event and a `subagent_ended` event MUST set the envelope's parent identifier" | Already satisfied by delegation_events.go; AG-19 only calls the existing constructors, does not need to amend this requirement. |
| R-CST-004 | `openspec/specs/agent-cost-events/spec.md:102-119` | "MUST be run-scoped... MUST NOT be carried on the harness value... MUST NOT introduce a second writer to the run's event path" | Directly forecloses folding child cost into the parent's own `cost_session`/`total` (§6) — the single most consequential collision found. AG-19.3's "aggregates into the parent's cumulative figures" must be read/implemented as consumer-side reconstruction, and the proposal should say so explicitly to avoid a design that silently violates a shipped requirement. |
| agent-cost-events non-req row | `spec.md:171,196` | "Aggregating a delegated or subagent run's cost into a parent run is AG-19's... Nothing in this capability may be written as if a parent scope exists" | Explicitly hands this exact question to AG-19 — confirms the ambiguity is intentional and AG-19's proposal is where it must be resolved, not silently assumed. |
| agent-cancellation-tree non-req row | `spec.md:181` | "Subagent cancellation inheritance \| AG-19.2." | Confirms in-scope, pre-planned. |
| agent-cancellation-tree non-req row | `spec.md:188` | "Concurrent runs on one harness value \| Not this milestone." | Does not block AG-19 (distinct Harness values, not concurrent Run on one value) — worth a one-line note in the proposal so a reviewer doesn't flag it as a false collision. |
| agent-permission-protocol non-req row | `spec.md:169` | "Subagent tool scope \| AG-19.3." | Confirms in-scope, pre-planned; no existing "scope" type to amend. |
| `EventKindPermissionResolutionRemembered` cardinality | `event.go:174-177` (R-APE-003) | `CardinalityAtMostOne` | A second occurrence on ONE `CheckStream`-validated slice fails — reinforces §3's "split before validate" conclusion; a naive same-stream merge of parent+child `permission_resolution_remembered` events would break this even without a run-bracket collision. |
| doc.go invariant-2 pre-declaration | `doc.go:47-50` | "AG-04 co-closes... invariant 2 with AG-19.1" | Not a collision — a pre-planned narrative update AG-19.1 must make (§9). |

No count assertions ("contains N rows/kinds") were found scoped to the delegation family specifically beyond the registry's own kind count comments (event.go:190-199, "2 kinds"), which AG-19 does not change (it emits existing kinds, adds none).

## The hardest open question: how does a child's events reach the parent stream with parent identity?

Two viable approaches, compared against the evidence above:

**Approach A — `emissions`-channel seam (reuse the existing multi-writer funnel)**
`Scheduler.executeCall` already has `emissions chan<- emission` in scope (scheduler.go:401-464) and is the SAME mechanism concurrent sibling tool calls already use to reach `sink` safely through one stamping goroutine (`runDispatcher`, scheduler.go:281-298). A minimal seam: `executeCall` wraps `ctx` with a small, exported, package-external-usable handle (NOT the unexported `emission` type itself, since the test-only tool lives in `package agent_test` — needs an exported interface, e.g. a `DelegationSink`-shaped door) before calling `tool.Run(ctx, args, policy)`. A delegation-aware tool retrieves it, runs the child `Harness` on its OWN private sink, and a forwarding goroutine translates the child's `RunStart`/`RunEnd` into `NewSubagentStarted`/`NewSubagentEnded` (enqueued via the handle) while letting the child's OTHER events (message/tool/cost/permission) flow to wherever the child's own captured stream lives (NOT verbatim onto the parent's `CheckStream`-governed slice, per §3/§6's collisions).
- Pros: reuses an already-race-proven, already-tested single-writer/multi-producer pattern (`runDispatcher`); avoids any change to `Tool`'s exported signature; naturally serializes with sibling tool calls' own start/end events; keeps `CheckStream` and `R-CST-004` intact by construction if the forwarder is careful to exclude `cost_turn` payloads from any channel `harness.go:618-637`'s `total.add` filter can see.
- Cons: needs a NEW, currently-nonexistent exported "delegation door" type/accessor (small, additive, but real production code — contradicts a naive "tests only" hypothesis for AG-19.1); the forwarder/translation logic is genuinely delicate (§3, §6) and needs its own dedicated tests.

**Approach B — Tool interface extension (explicit delegation parameter)**
Add a second, optional interface (e.g. `DelegatingTool` with a richer `Run` accepting an explicit `DelegationContext{Sink, Stamper, ParentRun, ParentPolicy}`), and have `executeCall` type-assert `tool.(DelegatingTool)` before falling back to the plain `Tool.Run`. Non-breaking to every existing `Tool` implementation.
- Pros: explicit, discoverable, no `context.Value` indirection; matches "seam 12: re-entrancy cannot be added later" (doc 0003:1795) as a first-class, named door rather than a smuggled context key.
- Cons: larger surface than Approach A (a whole new interface + dispatch branch in `executeCall`); still needs the SAME translation/exclusion care for `CheckStream`/`R-CST-004`; risks becoming a de facto "subagent tool contract" that reads as shipping more than the charter's explicit "no subagent tool ships in v1" (doc 0003:1795, 1803) intends.

**Recommendation for `sdd-propose`**: Approach A, scoped to the smallest possible exported surface (one accessor + one narrow interface), with the translation/forwarding logic living in a `_test.go`-only helper alongside the test-only tool wherever feasible, and a production seam limited to "make `emissions` (or an equivalent) reachable from inside `tool.Run`" — nothing else. Both approaches require the SAME two non-negotiable constraints discovered above: (1) never let the child's raw `RunStart`/`RunEnd` or `permission_resolution_remembered` land verbatim on the parent's own `CheckStream`-validated slice; (2) never let the child's `cost_turn` payloads reach `harness.go:618-637`'s `total.add` capture point.

## Effort estimate

Given the evidence, AG-19 is **NOT** "mostly tests" in the naive sense — it needs one small, real production seam (reaching the parent's event-forwarding door from inside `tool.Run`) plus the test-only tool and its scripted-conversation scenarios. Everything else (child Harness construction, cancellation propagation, PermissionPolicy derivation, cost reconstruction) is achievable in test code with zero production change, given the evidence in §5-§7.

## Risks (CRITICAL flagged)

1. **CRITICAL** — R-CST-004 (`agent-cost-events/spec.md:102-119`) forbids folding child cost into the parent's own cumulative; a design that does this anyway would ship a requirement violation. The proposal MUST state the "aggregation" is consumer-side reconstruction, not a Layer-2 fold.
2. **CRITICAL** — `CheckStream`'s global (non-run-scoped) bracket/cardinality state (`stream_check.go`) makes a verbatim single-stream merge of parent+child events structurally invalid; the proposal MUST specify the per-run split-then-validate reconstruction algorithm explicitly, or a reviewer will reasonably ask "how does this pass CheckStream" and get no answer.
3. **HIGH** — no existing seam lets a tool reach the parent's event-forwarding path (`Tool.Run` has no sink/stamper/scheduler handle); AG-19.1 needs a small, real, additive production change, contradicting a "tests only" assumption for this leaf specifically.
4. **MEDIUM** — nested wind-down timing: the child's own wind-down (`windDownRun`) must complete inside the PARENT's `Scheduler.WindDownBound` (default 100ms) or the parent's own tool call gets reported as a `DetachedCallError` rather than a clean nested cancellation — AG-19.2's test needs either a generous bound or an explicit assertion of this edge behavior.
5. **LOW** — `doc.go`'s machine-checked layer contract already pre-declares AG-19.1 as invariant 2's co-closer (`doc.go:47-50`); forgetting to update that narrative paragraph (not a new L2C-* row) would leave a stale citation.
