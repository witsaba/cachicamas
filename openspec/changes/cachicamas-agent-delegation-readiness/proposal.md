# Proposal: AG-19 — Prove re-entrancy and delegation readiness

> **Change**: `cachicamas-agent-delegation-readiness` · **Milestone**: AG-19 (Layer 2 Wave 5, milestone 19 of 24; doc `0003:1793-1862`)
> **Worktree**: `cachicamas-worktrees/ag19` · base `main@558641f3` (PR #180, AG-18)
> **Artifact store**: hybrid (Engram + filesystem) · **Execution mode**: `auto`
> **Delivery**: `exception-ok` — a **single PR** carrying the change, the doc 0003 milestone-document update and the OpenSpec archive together (the AG-16/AG-17/AG-18 house pattern)
> **Review budget**: 1000 changed lines, counted **excluding everything under `openspec/`** — the user designated `openspec/` a working folder. `sdd-tasks` and `sdd-apply` inherit this counting rule verbatim.
> **TDD**: strict, RED-first. Canonical runner: `cd backend/agent && make test` (`go test -race -v ./...`); **evidence runs MUST force `-count=1`** — the real uncached agent-module suite is ~170s, so a sub-second "pass" is a cache artifact, not evidence.
> **Closes**: **G7**'s structural half (**R-14**); seam 12 — re-entrancy cannot be added later.
> **Depends on**: AG-02, AG-06, AG-10, AG-13, AG-14, AG-16 (all merged).
> **Exploration**: `explore.md` · Engram `sdd/cachicamas-agent-delegation-readiness/explore`
> **ID prefix**: `R-DEL-` / `S-DEL-` / `NFR-DEL-` — **verified free**: zero occurrences anywhere in the worktree.

---

## Intent

The harness is documented as re-entrant. Nothing proves it, and one thing quietly forbids it: **`Tool.Run(ctx, args, policy PolicySlot)` has no sink, no stamper and no scheduler handle** (`tool.go:182-186`). A tool can build a child `Harness` value today — `Harness` is value-form with "zero surface beyond two methods" (`harness.go:12-13`) — and that child's entire conversation would be **invisible** to the parent's consumer. There is no door.

Meanwhile the substrate for nesting has been paid for across five milestones and has **zero production callers**:

| Built | Where | Callers today |
|---|---|---|
| The envelope's parent field, reserved so "explicit nesting cannot be retrofitted" | `event.go:454-462`, `Event.Parent()` `:492` (AG-04.1) | 3 constructors |
| `NewSubagentStarted` / `NewSubagentEnded`, which set it | `delegation_events.go:62,134` (AG-06.3) | **zero** |
| A run-scoped cancel-cause tree tools already inherit | `harness.go:434` (AG-14) | — |
| A run-scoped cost bracket that deliberately refuses a parent scope | `R-CST-004` (AG-16) | — |

**Seam 12's argument is not "build the door later".** It is that the *shape* the door needs — a parent identifier on the envelope from birth, a cancel-cause chain that is already transitive, a cost bracket that is run-scoped rather than harness-scoped — cannot be retrofitted once consumers exist. That shape is built. AG-19 is the milestone that **proves it holds under a real nested run** and, in doing so, opens the smallest possible door. If the proof is deferred, the four specs that explicitly named AG-19 as their closer (`agent-cost-events:196`, `agent-cancellation-tree:181`, `agent-permission-protocol:169`, `agent-protocol-events:157`) stay open indefinitely, and Layer 3's subagent work would begin by *designing* the seam rather than *using* it.

---

## Resolved questions — the six ambiguities, decided here

`sdd-design` may overturn any of these on stated evidence; it may not resolve them by silence.

### Q1 — Approach A (context-carried seam reusing `Scheduler`'s `emissions` funnel) or Approach B (an optional `DelegatingTool` interface)?

**Decided: Approach A.** The exploration recommended it; this proposal re-argues it rather than inheriting it, because the naive objection — "a `context.Value`-smuggled handle is an anti-pattern" — is a real one.

**Why the objection does not hold here.** The anti-pattern is smuggling data that *should have been a parameter*. Making it a parameter here means changing `Tool.Run`'s signature, which breaks every existing implementation (`importPathGuard` at `tool.go:201`, `ScriptedTool`, `scriptedBiteTool`, `scriptedReadTool`) and is a change AG-19's dependency set does not authorise. The value is genuinely per-call and per-invocation — precisely the case Go's own stdlib reserves `context.Value` for. **The package already does this at exactly this boundary**: `loop_test.go:475-484` wraps a `ctxMarkerKey{}` value onto the context specifically to prove it reaches `tool.Run`. A ctx-carried seam is idiomatic here, not novel.

**The two guards it might trip, checked rather than assumed.**

| Guard | Verdict |
|---|---|
| `TestScheduler_SourceGuard_PolicySlotNoTypeAssertion` (`scheduler_test.go:616-646`) scans `scheduler.go` for `.(PolicySlot)` / `.(*PolicySlot)` | **Untouched.** Approach A adds `context.WithValue` to `scheduler.go` and **zero type assertions**; the seam's own assertion lives in a new file, on a different type. Crucially it does **not** overload `PolicySlot` — seam 3's single stated meaning survives intact. |
| `R-AGE-017` (`agent-event-delivery:241`): downstream milestones "may not invent [their] own channel, [their] own loss rule, or [their] own way back into a live run" | **Satisfied by A, jeopardised by B.** A reuses `Schedule`'s existing `emissions` → `runDispatcher` → `sink` funnel verbatim (`scheduler.go:155,196,281-298`) — the same single-stamping-goroutine pattern concurrent sibling tool calls already use. Inventing nothing is the point. |

**Why B loses.** B's only honest form is a second `Run` method (`DelegatingTool`), because a stateful `AcceptSeam(...)` setter races two concurrent `Schedule` calls sharing a tool value. A second `Run` method forces a branch through `runToolWithWindDown` — the wind-down, detach, panic-recovery and modify-input logic in the most safety-critical function in the scheduler (`scheduler.go:588-630`) — duplicated or conditionally shared. Approach A adds **no new call path at all**: `tool.Run` remains the single entry, and the seam rides the *same* `ctx` the child harness will derive its cancel-cause context from, which is what makes Q5's cancellation propagation free.

**B's one genuine advantage is discoverability**, and it is answered cheaply: the seam is a **named exported interface** with a doc comment, reachable through a named exported accessor, in its own file — not an unexported context key a reader must find by accident.

**Ruled out entirely:** reusing `PolicySlot` as the carrier. It would overload a seam whose single documented meaning is the permission slot (`scheduler.go:466-471`), and the source guard exists to keep that meaning single.

### Q2 — What does "child events parent-identified" mean?

**Confirmed transitive. The acceptance clause is satisfied by a walkable tree, not by stamping every child event.**

Only **three** constructors in a 19+-kind vocabulary accept a parent: `NewDelegatedRunStart`, `NewSubagentStarted`, `NewSubagentEnded`. Every message, tool, permission, cost and turn constructor takes no parent parameter and never sets `hasParent` (`permission_events.go:138,242,326`; `cost_events.go:193,296`). Making the clause literal means rewriting the entire construction surface — far outside AG-19's declared dependency set, and it would falsify `R-AEV-003`'s "a top-level event reports no parent as a distinguishable false".

The Gherkin states its own mechanism in its second clause: *"a consumer separates the two conversations by walking parents."* The normative reading, which `sdd-spec` MUST record in these terms so no later reviewer re-reads it as a stamping obligation:

> Every child event carries `Run()` = the child run identity. Exactly one `subagent_started` on the parent's stream carries `Run()` = that same child identity and `Parent()` = the parent run identity. Therefore **every** child event is attributable to the parent in one hop, with no per-event parent field.

This is what `agent-event-envelope:257`'s invariant-2 row ("AG-04.1 **+ AG-19.1**") has meant since AG-04.

### Q3 — What does "child cost aggregates into the parent's cumulative" mean?

**A consumer-side reconstruction. No Layer-2 fold occurs, and the seam makes the fold impossible rather than merely discouraged.**

`R-CST-004` (`agent-cost-events:102-119`) is shipped and machine-tested: the cumulative is "the sum over every `cost_turn` event emitted **within that run bracket**", it "MUST NOT be carried on the harness value", and "Accumulation MUST NOT introduce a second writer to the run's event path." A child's `cost_turn` is emitted inside the *child's* bracket. Folding it into the parent's `cost_session(Final)` would ship a violation of a requirement `S-CST-008`/`S-CST-021` already defend.

The charter's own qualifier agrees: *"a frontend can show both 'this subagent cost X' and 'the run cost Y'"* — **two displayed numbers, not one merged Layer-2 event.**

**The hazard is live, not hypothetical.** The parent's per-attempt forwarder folds by payload type with **no run-identity filter**: `if ct, ok := ev.CostTurn(); ok { total.add(ct) }` (`harness.go:633-635`). And under Approach A the funnel terminates in exactly that channel: `emissions` → `runDispatcher` → `sink`, where `sink` **is** the parent's per-attempt `turnSink` (`harness.go:614,648`). Any child `cost_turn` reaching the seam would be folded **silently**.

**The guard**, which is production behaviour and not test discipline: the seam **refuses** any event carrying a `cost_turn` payload, returning a typed refusal. Proven two-sidedly —

1. a direct unit scenario: publishing a `cost_turn` through the seam returns the typed refusal and reaches no sink;
2. an end-to-end scenario where the child provably spends **non-zero** tokens: the parent's terminal `cost_session` equals the sum over the `cost_turn` events **observed on the parent's own stream**, and is **strictly less than** the parent+child sum, while a consumer walking both streams by parent identity recovers the combined figure. The inequality is what makes the assertion non-vacuous.

### Q4 — What crosses onto the parent's physically delivered stream?

`CheckStream` never reads `Event.Run()`; its run bracket, `turnOpen` flag, `seenAtMostOnce` map and terminal latch are **global over the validated slice** (`stream_check.go:92-176`). So the crossing rule is not a matter of taste — it is derivable from the existing registry, and it needs no change to `stream_check.go` or `event.go` (both on `R-LSK-004`'s byte-unchanged list).

> **Admissibility rule.** An event may cross **iff** its registered descriptor has `Bracket == BracketRoleNone` **and** `Cardinality != CardinalityAtMostOne` **and** `Terminal == false`, **and** its payload is not a `cost_turn`.

| Crosses | Why it is safe |
|---|---|
| `subagent_started`, `subagent_ended` | `PlacementTurn`, `BracketRoleNone`, `CardinalityAny` (`event.go:340-347`); the parent's turn is open across the tool call |
| the child's message and tool events | same descriptor shape; this is what makes the Gherkin's "the child's events" literal rather than reduced |
| the child's `permission_decision_required` **and** its answering `permission_decision_made` | `PlacementTurn`, `CardinalityAny` (`event.go:312-319`). The pair crosses together — a mirrored ask with no mirrored answer is unreadable to the human watching |
| **Does not cross** | **Why it cannot** |
| the child's `run_start` / `run_end` | `BracketRoleOpensRun` → second open is `ErrDuplicate` (`:122-127`); `run_end` is also `Terminal: true` (`event.go:249`) → everything after it on the parent's slice is rejected |
| the child's `turn_start` / `turn_end` | `BracketRoleOpensTurn` inside an already-open turn is `ErrMisplaced` (`:137-155`) |
| the child's `permission_resolution_remembered` | `CardinalityAtMostOne` (`event.go:320-323`) — a second occurrence anywhere in one slice is `ErrDuplicate` |
| the child's `cost_turn` | Q3 — the one carve-out `CheckStream` would happily accept but `R-CST-004` forbids. **State this asymmetry loudly**: it is a requirement-level rule, not a stream-legality rule |

**Re-stamping is what makes mirroring safe.** `LaneStamper.Stamp` takes a value and returns a copy with a fresh sequence, discarding the prior one (`sequence.go:50-58`). The mirrored event is therefore a *distinct value* contiguous on the parent's lane, while the child's own value stays contiguous on the child's lane. Two lanes, no merge.

**The reconstruction algorithm** `sdd-spec` must make normative: validate the parent's delivered stream with `CheckStream` as today; **separately** capture and independently `CheckStream`-validate the child's own stream from the child `Harness.Run` call. Never concatenate them.

**The permission crossing closes a pre-stated obligation.** `S-AGE-022` (`agent-event-delivery:233-237`) already requires that a decision for a nested call "is sent to the parent's surface and arrives at the child's suspension, AND the routing obligation is stated rather than left to the delegation milestone to invent." AG-19 exercises it with **zero production routing surface**: the test-only tool owns the child `Scheduler` value and calls the existing `WakeParked` (`scheduler.go:264`) directly. Nothing new is invented; `R-AGE-017` holds unamended.

### Q5 — The exact production surface

Three exported identifiers, one new file, two lines inside `executeCall`.

| Identifier | Shape | Why it is minimal |
|---|---|---|
| `DelegationSeam` | `interface { Parent() (RunID, TurnID); Publish(ev Event) error }` | `Parent()` is unavoidable: `Tool.Run` receives no run or turn identity, so without it the tool cannot construct `NewSubagentStarted(childRun, parentRun, parentTurn, id)` at all. It exposes only what `Event.Run()`/`Event.Turn()` already expose on every event. |
| `DelegationSeamFrom(ctx) (DelegationSeam, bool)` | accessor | The named door. Lives in `delegation_seam.go`, not `scheduler.go`. |
| one typed refusal (e.g. `ErrDelegationRefused`, or a small typed error with a reason discriminator — design picks) | error | Carries both refusal reasons: inadmissible event (Q4) and revoked seam (below). |

Plus, in `executeCall`: one `context.WithValue` before `runToolWithWindDown`, and one deferred **revocation**.

**Revocation is not optional, and it is a finding this proposal adds.** `Schedule` does `close(emissions)` after `wg.Wait()` (`scheduler.go:240`). On the ordinary path a tool cannot publish after that, because `wg.Done()` is deferred past `tool.Run`. **On the detached path it can**: `runToolWithWindDown` abandons the tool's goroutine after the wind-down bound (`:621-627`) while `Schedule` proceeds to close the channel — a late `Publish` would **panic on send to a closed channel**. The seam MUST therefore be revoked when the call completes or detaches, and a revoked `Publish` MUST return the typed refusal rather than send. Approach B shares this hazard identically.

**Why this is a seam and not a subagent feature.** It names no subagent concept in any type; `Publish` publishes *events*, not subagents. It has no notion of depth, configuration, or child lifecycle. Everything that knows what a subagent *is* lives in a `_test.go` file in `package agent_test` — the `ScriptedTool` precedent (`scripted_tool_test.go:1-55`) — which production code **cannot import**. The charter's "**No subagent tool ships in v1**" is enforced by the compiler, not by a convention.

### Q6 — The nested wind-down bound

**Decided: a generous `Scheduler.WindDownBound` on the parent, not an assertion of the `DetachedCallError` framing.**

AG-19.2's scenario reads: *"the child winds down first — its orphans synthesized, its run-end emitted — and then the parent's wind-down completes, with both transcripts closing valid."* A detached call produces a **typed execution failure** on the parent's stream (`typedDetachedCallFailure`, `scheduler.go:1126-1134`) — a legal stream, but a *different story* from the one the acceptance line describes. Asserting the detach framing would prove a claim the charter did not make.

**This does not violate the no-wall-clock rule.** `NFR-CAN-002`-style discipline forbids *synchronising* by sleep or timeout. The bound is a ceiling, not a synchronisation point: the test still synchronises on channel closes and `agenttest.Gate`, and asserts ordering by observed event order on the two captured streams — never by elapsed time. Precedent exists in the opposite direction (AG-14's leak test injects a small value).

**The detach interaction is recorded as a named non-requirement, not silently dropped**: it is already owned and tested by `R-CAN-006` / `R-TLS-014`, and the structural coupling — a child wind-down slower than the parent's bound detaches the parent's tool call — is stated in the spec so it is inherited knowingly.

---

## What changes

### Production seam (small, additive, and the whole of it)

- **`delegation_seam.go`** (new): `DelegationSeam`, `DelegationSeamFrom`, the typed refusal, the admissibility filter derived from the existing registry, and the revocation state.
- **`scheduler.go`** (modified): install the seam on the tool's `ctx` in `executeCall`; revoke it on return or detach. **Zero type assertions added.**

### Test-only surface (where every subagent concept lives)

- A **delegating tool** in `package agent_test`: hosts a child `Harness`/`Scheduler`, constructs `subagent_started`/`subagent_ended`, mirrors admissible child events through the seam, and — for AG-19.3 — routes the human's decision down via the child's `WakeParked`.
- A **derived permission scope** as an ordinary Go value composing the parent's `PermissionPolicy` (`permission_protocol.go:80-94`). No new Layer 2 type: "scope" is not a Layer 2 concept and this change does not make it one.
- The **two-stream reconstruction helper** and the consumer-side cost sum.

### Not changed, and this is assertable

`event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `go.mod`, `go.sum`, and **everything under `backend/agent/src/ai/`**. No new `EventKind`; the every-kind guard passes at its committed count. `Tool`, `PolicySlot`, `Harness` and `PermissionPolicy` keep their signatures.

---

## Scope

### In

- **AG-19.1** — the seam; the first production emission of `subagent_started`/`subagent_ended`; the mirroring rule of Q4; a nested run whose events are attributable by walking parents; two sibling children interleaving through the one `runDispatcher` without cross-talk.
- **AG-19.2** — nested cancellation inherited through the existing context tree, proven leaf-first, both transcripts `CheckStream`-valid.
- **AG-19.3** — consumer-side cost reconstruction with the seam's `cost_turn` refusal as its guard; a derived permission scope by composition; the child's ask surfacing on the parent's stream and the decision reaching the child's suspension.
- The spec deltas below, the doc 0003 milestone-document update, and the OpenSpec archive — **same PR**.

### Out — quoting the charter verbatim

> "**Out of scope:** A production subagent tool, subagent configuration, and depth limits — post-v1, on this proven substrate." (`0003:1803`)

| Also deferred | Owner and why the deferral is safe |
|---|---|
| Any Layer-2 fold of child cost into a parent figure | **Never.** `R-CST-004` forbids it; Q3 makes it a typed refusal |
| A per-event parent stamp on the other 16+ event kinds | **Not built.** Q2 — the tree is walkable without it, and stamping would falsify `R-AEV-003` |
| A "scope" type, rule set, or mode flag in Layer 2 | **Layer 3 / CO-03**, restated at `permission_protocol.go:77-79` |
| Concurrent `Run` calls on **one** `Harness` value | **Not this milestone** (`agent-cancellation-tree:188`). AG-19's siblings use **two distinct `Harness` values** — noted so a reviewer does not flag a false collision |
| Subagent-scoped retry | **Post-v1** (`agent-retry-failover:227`) |
| A narrower nested-cancellation signal | **Not built.** The child participates in the existing tree; it does not get a branch of its own |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2** (`R-RUN-012`) |

---

## Capabilities and delta-spec plan

`sdd-spec` finalises the requirement text; this proposal fixes the shape.

### New

- **`agent-delegation-readiness`** — kebab-case, matching the `agent-*` convention; directory does not exist. IDs `R-DEL-0NN` / `S-DEL-0NN` / `NFR-DEL-0NN`, **prefix verified free** (zero occurrences worktree-wide). **No total requirement count is stated** — a count goes silently false the moment a later milestone appends (this repo's known count-assertion drift class).

Expected families: `R-DEL-001` the seam surface and admissibility rule · `R-DEL-002` seam lifetime and revocation · `R-DEL-003` parent identity as a walkable tree · `R-DEL-004` the two-stream split-then-validate reconstruction · `R-DEL-005` cancellation inherited leaf-first · `R-DEL-006` cost as consumer-side reconstruction, with the refusal guard · `R-DEL-007` derived permission scope, ask up / decision down · `R-DEL-008` the scope fence.

### Modified

| Capability | What changes | Certainty |
|---|---|---|
| `agent-cost-events` | Non-requirement row `:196` and the `:171` clause — the deferral **closes as consumer-side reconstruction**. `R-CST-004` is **unamended** and must be shown still true | **Certain** |
| `agent-cancellation-tree` | Non-requirement row `:181` "Subagent cancellation inheritance \| AG-19.2" closes | **Certain** |
| `agent-permission-protocol` | Non-requirement row `:169`, plus the `Blocks` lines `:9` and `:177`, close | **Certain** |
| `agent-event-envelope` | Invariant-2 row `:257` ("AG-04.1 + AG-19.1") and the `:57` clause — nesting semantics land | **Certain** |
| `agent-protocol-events` | `:154` / `:157` — the delegation family's live-emission half closes. `R-APE-006` is **unamended**, already satisfied by `delegation_events.go` | **Certain** |
| `agent-event-delivery` | Back-annotation: `R-AGE-017`'s inheritance is **exercised without amendment** (no invented channel), and `S-AGE-022`'s routing obligation is discharged | **Certain** |
| `agent-tool-scheduler` | The seam installed in `executeCall`; seam 3 / `R-TLS-002` **unamended** and the `PolicySlot` source guard still green | **Certain** |
| `agent-v1-scope` | AG-19's inheritance statement (`S-AGS-022`, `S-AGS-050`); **G7's structural half / R-14** discharged | **Certain** |
| `agent-retry-failover` | `:227` row wording, back-annotation only | **Likely** |
| `agent-loop-skeleton` | `R-LSK-004` exact-filename release for **`doc.go`** — required **only if** design edits the invariant-2 narrative at `doc.go:45-50`. **Verified: that sentence does not go false when AG-19 ships** (it says AG-04 co-closes *with AG-19.1*), so the edit is optional and the release is conditional on it | **Conditional** |
| `agent-compaction` `:320` | Row remains true; **no delta** | — |

---

## Approach

1. **Author the new capability spec and the deltas first.** Unlike AG-18, AG-19 falsifies no shipped requirement — but Q3 and Q4 are exactly where an implementation could silently violate one, so the constraints are normative before any code.
2. **The seam** (`delegation_seam.go`) with its admissibility filter and revocation, tested in isolation: an admissible kind publishes; each inadmissible class refuses typed; a revoked seam refuses rather than panics.
3. **The `executeCall` install + revoke** — two lines, then re-run the `PolicySlot` source guard and the ambient-authority / import-boundary guards **unchanged**.
4. **The delegating test tool** and AG-19.1's two scenarios, with both streams `CheckStream`-validated separately.
5. **AG-19.2** — cancellation, generous parent bound, ordering asserted by observed event order.
6. **AG-19.3** — the cost reconstruction with its strict inequality, then the derived scope with ask-up/decision-down.
7. **Doc 0003 update + archive**, same PR.

**Bites are not optional.** At minimum: (a) let the seam admit `cost_turn` → the cost scenario must FAIL showing the parent's cumulative inflated by child spend; (b) let the seam admit the child's `run_start` → the parent's `CheckStream` must FAIL with `ErrDuplicate`; (c) drop the revocation → the detach scenario must FAIL (panic or refusal missing); (d) give the child an independent context instead of the tool's `ctx` → the nested-cancellation scenario must FAIL. Each RED-recorded **before** its GREEN, with `-count=1`.

---

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/delegation_seam.go` (new) | **New** | `DelegationSeam`, `DelegationSeamFrom`, typed refusal, admissibility filter, revocation |
| `backend/agent/src/agent/scheduler.go` | **Modified** | Install seam on the tool's `ctx` in `executeCall`; revoke on return/detach |
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | **New** | Delegating tool, derived scope, reconstruction helper, three leaves' scenarios, four bites |
| `backend/agent/src/agent/doc.go` | **Conditional** | Invariant-2 narrative only if design elects it — triggers the `R-LSK-004` release |
| `openspec/specs/agent-delegation-readiness/` | **New capability** | Via the change's delta folder |
| `openspec/specs/{agent-cost-events, agent-cancellation-tree, agent-permission-protocol, agent-event-envelope, agent-protocol-events, agent-event-delivery, agent-tool-scheduler, agent-v1-scope, agent-retry-failover}` | **Delta** | Nine deltas; `agent-loop-skeleton` conditional |
| `docs/architecture/milestones/0003-…md` | **Modified** | AG-19 status, delivery table, checklist; counter to 19/24 |
| `event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `go.mod`, `go.sum`, **all of `src/ai/`** | **NOT TOUCHED** | No new kind, no Layer 1 edit, no new dependency |

---

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | **A design folds child `cost_turn` into the parent's `total`**, violating shipped `R-CST-004` — and it happens *by default*, because the seam's funnel terminates in the very channel whose forwarder folds by payload type with no run filter (`harness.go:633-635`) | **High** | Q3. The refusal is **production behaviour**, not test discipline, plus bite (a). `sdd-verify` MUST open `agent-cost-events:102-119` against the shipped code — a citation is not evidence |
| 2 | A verbatim merge of parent+child events is attempted and `CheckStream` rejects it late, in `sdd-apply` | **High** | Q4's admissibility rule is derived from the registry and stated normatively before any code; bite (b) |
| 3 | **The detached-call path publishes after `close(emissions)` and panics** — newly identified in this phase, not in `explore.md` | **Med-High** | Q5's revocation; bite (c). `sdd-design` MUST decide *where* revocation is sequenced relative to `runToolWithWindDown`'s detach arm |
| 4 | The seam is designed as a subagent-shaped door and the change reads as shipping the thing the charter forbids | Med-High | Q5's surface table; the subagent concept exists only in `package agent_test`, compiler-enforced |
| 5 | AG-19.1's acceptance is read as "stamp every child event", producing a rewrite of the construction surface | Med | Q2 is normative and stated in the spec's own words, not left to inference |
| 6 | The nested-cancellation test synchronises on wall-clock and flakes under `-race` | Med | Q6: the bound is a ceiling; synchronisation is by channel close and `agenttest.Gate`; ordering asserted by observed event order |
| 7 | One of nine-to-ten deltas is missed — **no Go test enforces a back-annotation** | **Med-High** | Enumerated with file and line above. `agent-event-delivery`, `agent-tool-scheduler`, `agent-retry-failover` and `agent-v1-scope` were **not** in `explore.md` and were found by grep in this phase |
| 8 | Evidence recorded from a **cached** run | Med | `-count=1` mandated in the header and every acceptance line; record wall-clock duration |
| 9 | Child spend is zero in the fixture, making the cost inequality vacuous | Med | Q3's assertion is two-sided and requires **non-zero** child spend and a strict inequality |
| 10 | `doc.go` is edited without the `R-LSK-004` release, or the release is requested for a file never touched | Low-Med | Stated as conditional, with the verified reason the sentence does not go false |

---

## Rollback plan

**Single revert of the AG-19 merge commit.** `delegation_seam.go` and all new test files are deleted; `scheduler.go` loses two lines in `executeCall`; the deltas are dropped; doc 0003's AG-19 line un-ticks and the counter returns to 18/24.

**The revert is clean, and the reason is structural.** Nothing persists across processes. No data migrates. No `go.mod`/`go.sum` change. No Layer 1 file touched. No `EventKind` added or removed — the delegation family simply returns to having **zero production callers**, exactly its pre-AG-19 state. No consumer outside the test surface can hold a `DelegationSeam`, because no production code constructs one.

**Forward-looking cost**: reverting re-opens G7's structural half / R-14, returns four non-requirement rows to deferred, and removes the substrate Layer 3's subagent work is meant to stand on. Scheduling consequence, not correctness.

---

## Review-workload forecast

**Counting rule**: additions + deletions, **excluding every path under `openspec/`**. SDD markdown still counts toward the **attempt budget** — a different budget; `sdd-tasks` must not conflate them.

| Component | Estimate (authored, non-`openspec`) |
|---|---|
| `delegation_seam.go` — seam, filter, revocation | 130–220 |
| `scheduler.go` — install + revoke | 15–35 |
| Delegating test tool + derived scope + reconstruction helper | 250–400 |
| Three leaves' scenarios + four bites | 600–950 |
| `doc.go` (conditional) | 0–20 |
| Doc 0003 status line, delivery table, checklist | 25–45 |
| **Counted total** | **1020–1670** |
| *Uncounted (attempt-budget relevant)*: proposal, new capability spec, 9–10 deltas, design, tasks, apply-progress, verify-report, archive report | *1100–1800* |

`Decision needed before apply: No` — `exception-ok`, single PR, `size:exception` pre-accepted at 1000 counted lines with a user-pre-accepted extension if the milestone genuinely needs it.
`Chained PRs recommended: No` — held in reserve; the slicing boundary is the Approach ordering: **U1** = seam + install + its isolated tests · **U2** = AG-19.1 · **U3** = AG-19.2 + AG-19.3.
`400-line budget risk: High`

---

## Dependencies

- **AG-02** (archived) — the v1 scope verdict placing the subagent tool post-v1 (`agent-v1-scope:135`).
- **AG-06.3** (archived) — `NewSubagentStarted`/`NewSubagentEnded` and their registry rows (`delegation_events.go:62,134`; `event.go:340-347`); `R-APE-006`.
- **AG-10** (archived) — `PermissionPolicy`, the gate, `WakeParked` (`scheduler.go:264`), `S-AGE-022`'s routing obligation.
- **AG-13** (archived) — `Harness.Run`'s loop, the per-attempt forwarder, `LeaveSinkOpen`.
- **AG-14** (merged) — `context.WithCancelCause` at `harness.go:434`, `windDownRun` (`:317-350`), `WindDownBound` and the detach path (`scheduler.go:588-630`, `:1126-1134`).
- **AG-16** (merged) — `R-CST-004`, `costAccumulator`, the unconditional fold at `harness.go:633-635`.
- **`agenttest`** — `NewProvider`, `Gate` (the mandated no-sleep sync primitive), the stream assertion kit. **Byte-unchanged** unless design proves otherwise.
- **doc 0003:1793-1862** — the AG-19 charter and its three Gherkin leaves.

---

## Success criteria — restated as verifiable checks

- [ ] `cd backend/agent && make test` green under `-race` with **`-count=1`**; wall-clock duration recorded as part of the evidence
- [ ] **AG-19.1 / nested run** — a child harness runs inside a tool; the parent's stream carries `subagent_started`, admissible child events, and `subagent_ended`; `CheckStream` accepts the parent's stream **and** the child's own stream, each validated separately, with `stream_check.go` **byte-unchanged**
- [ ] **AG-19.1 / walkable tree** — every child event on the parent's stream resolves to the parent in one hop via `subagent_started`'s `Parent()`; **no** event constructor gains a parent parameter
- [ ] **AG-19.1 / siblings** — two sibling tools each hosting a child run interleave with no cross-talk; both parent-lane sequences remain contiguous; green under `-race`
- [ ] **AG-19.2 / leaf-first** — the child's orphans are synthesized and its `run_end` emitted **before** the parent's wind-down completes; both transcripts close valid; the parent's tool call is **not** detached; ordering asserted by observed event order, never elapsed time
- [ ] **AG-19.3 / cost** — the parent's terminal `cost_session` equals the sum over the `cost_turn` events on the **parent's own** stream and is **strictly less than** the parent+child sum (child spend non-zero); a consumer walking both streams by parent identity recovers the combined figure; `R-CST-004` shown still true against the shipped code
- [ ] **AG-19.3 / seam refusal** — publishing a `cost_turn`, a run-bracket kind, a `CardinalityAtMostOne` kind, or publishing through a revoked seam each returns the typed refusal and reaches no sink
- [ ] **AG-19.3 / permission** — the child's policy is a composition of the parent's with no new Layer 2 type; what the parent allowed flows down; the child's ask and its answer both appear on the parent's stream; the decision reaches the child's suspension via the existing `WakeParked`
- [ ] **Bites RED-recorded before GREEN**: `cost_turn` admitted; `run_start` admitted; revocation dropped; independent child context
- [ ] **Scope fence** — no new `EventKind`; `Tool`, `PolicySlot`, `Harness`, `PermissionPolicy` signatures unchanged; `event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `go.mod`, `go.sum` and all of `src/ai/` **byte-unchanged**; the every-kind guard passes at its committed count
- [ ] **Guards green unchanged** — `TestScheduler_SourceGuard_PolicySlotNoTypeAssertion`, the ambient-authority guard, the import-boundary guard, the doc-contract guard
- [ ] **Deltas shipped** — nine certain, `agent-loop-skeleton` iff `doc.go` is edited; `sdd-verify` opens each cited line
- [ ] `make lint` (after `golangci-lint cache clean`), `make build`, `make vuln-check` all clean (`vuln-check` is **not** in `make all`; **do not run `make all`** — its fmt step rewrites committed files)

---

## Proposal question round

Execution mode is `auto`, so these were not asked interactively. Each changes the shape of the product, not the harness. **Answering any of them before `sdd-design` moves the recommendation above.**

1. **Is a walkable tree an acceptable delivery of "each child event parent-identified"?** Assumed **yes** (Q2). If a Layer 3 consumer needs a literal parent field on every event, that is a vocabulary-wide change and belongs to a different milestone — it cannot be smuggled in here.
2. **Should the child's own `cost_session(Final)` also cross onto the parent's stream?** It is `PlacementRun`, so `CheckStream` accepts it inside the parent's turn, and it **cannot** reach the parent's `total` (the forwarder folds only `CostTurn`) — making it the natural carrier of the frontend's "this subagent cost X". Assumed **deferred to `sdd-design`**, leaning yes; the proposal deliberately does not fix it, because a mirrored `cost_session` could confuse a naive consumer summing labels.
3. **Do the child's message and tool events belong on the parent's stream at all?** The Gherkin says "the child's events", so assumed **yes**. If the product wants a quieter parent stream, the answer is a consumer-side filter, not a narrower seam — but say so now rather than after the tests are written.
4. **Is a tool publishing arbitrary admissible events onto the parent's stream an acceptable capability?** The seam is not subagent-shaped by design (Q5), which means any tool holding it can publish. Assumed **acceptable**, bounded by the admissibility rule. A narrower, subagent-shaped door would be smaller in blast radius and larger in charter risk.
5. **Should the detach-under-nested-wind-down behaviour be a tested scenario rather than a documented consequence?** Assumed **documented only** (Q6), since `R-CAN-006` / `R-TLS-014` already own and test the detach path. If the product wants nested cancellation to be robust against slow children, that is a bound-negotiation feature and it is not in this charter.
