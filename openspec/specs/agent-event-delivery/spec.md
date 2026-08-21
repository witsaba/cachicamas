# Spec — Layer 2 agent event delivery and the observer model

> **Capability**: `agent-event-delivery` (new) · **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 — Decide event delivery and the observer model · **Node**: AG-01.1 `[decision]`
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-AGE-0NN` · **Scenario IDs**: `S-AGE-0NN` (allocated `S-AGE-001` through `S-AGE-031`)
> **Binding vocabulary**: AG-00's Layer 2 register — every Layer 2 noun below is one of its rows, cited, never redefined here
> **Frozen input, not amended**: [Layer 1 stream lifecycle](../../../../specs/ai-stream-lifecycle/spec.md)
> **Sources**: `proposal.md` and `explore.md` of this change · doc 0003 AG-01 charter · doc 0001 § 2.2, § 2.3, § 4.3

> [!IMPORTANT]
> **The system under test is the recorded decision artifact, not running code.** AG-01.1 is a `[decision]` leaf: it produces no production code and no file under the backend module. Every scenario below is verifiable by inspecting the merged artifact, without executing anything. No Go type name, field name, method name, or package identifier appears here — doc 0003's authoring constraint. Later milestones choose spellings.

## Purpose

Fix how Layer 2 events **travel**: the carrier at the package boundary, the backpressure posture at each of the two internal boundaries, the mechanism that makes a slow observer structurally unable to stall delivery, who owns and closes what across three nested scopes, and how a message re-enters a live run from above. What events **say** is AG-04's, AG-05's and AG-06's.

## What must be true — summary

| # | Closing-checklist item | Requirements |
| --- | --- | --- |
| 1 | Carrier at the boundary | `R-AGE-001` … `R-AGE-003` |
| 2 | Backpressure posture | `R-AGE-004` … `R-AGE-007` |
| 3 | Observer model | `R-AGE-008`, `R-AGE-009` |
| 4 | Close and ownership | `R-AGE-010` … `R-AGE-012` |
| 5 | The upward path (R-09) | `R-AGE-013` … `R-AGE-017` |
| — | Deferral, acceptance, constraints | `R-AGE-018`, `R-AGE-019` |

---

## Requirements

### R-AGE-001 — The carrier is decided and argued at Layer 2, not inherited by symmetry

The decision MUST name a **receive-only carrier** at the Layer 2 package boundary: a consumer may only receive from it, and cannot send on it or close it. The decision MUST reproduce Layer 1's four carrier grounds and MUST argue each **against a named Layer 2 source**, not against Layer 1 alone. It MUST state the alternatives considered — a push-callback and a send-capable sink handle — and what each costs. It MUST concede that the stranded-producer hazard is closed by the send discipline rather than by the carrier, and that a consumer which abandons without cancelling remains an untestable documented violation.

#### Scenario: S-AGE-001 — Each ground cites a Layer 2 source

- **GIVEN** a reviewer holding the merged decision and the four Layer 1 carrier grounds
- **WHEN** they check each ground for its supporting citation
- **THEN** at least the multiplexing ground and the terminal-event ground each cite a **Layer 2** milestone or a Layer 2 source clause by identifier
- **AND** a decision whose only support for any ground is "Layer 1 decided the same" fails this scenario

#### Scenario: S-AGE-002 — Alternatives are priced, not merely listed

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the carrier section
- **THEN** the push-callback alternative is rejected with a stated cost, AND the send-capable sink alternative is rejected with a stated cost, AND the abandonment concession is present as a stated residual cost of the chosen option

### R-AGE-002 — The caller-owned liveness rule is adopted as Layer 1 states it

The decision MUST adopt, unweakened: the caller supplies a cancellable liveness signal on the call that creates the stream; **every** send waits on both the destination and cancellation, the terminal send included; the two legal consumer endings are **drain to close** and **cancel**, and anything else is abandonment; cancellation closes within bounded time, with any backoff waiting on the signal rather than sleeping.

#### Scenario: S-AGE-003 — All four clauses are present and unweakened

- **GIVEN** Layer 1's four liveness clauses
- **WHEN** a reviewer maps each onto the merged decision
- **THEN** each has a counterpart with the same or narrower force
- **AND** no counterpart admits an unconditional send, an advisory cancellation, or a third legal ending

### R-AGE-003 — Iterator-view ergonomics live outside the Layer 2 runtime package

The decision MUST place any carrier-view convenience **outside** the Layer 2 runtime package, in a test-support sibling, and MUST state that such a view never owns and never closes the carrier it views and is never a second contract. Whether such a view is built at all MUST be left open, with the milestone that owns the question named.

#### Scenario: S-AGE-004 — Placement is stated with its reason

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the carrier section
- **THEN** the view is placed outside the runtime package, AND the reason cites the no-I/O rule or the production/test closure split, AND the "never a second contract" clause is present
- **AND** the existence question is explicitly deferred to a named milestone rather than silently answered

### R-AGE-004 — Agent events are lossless and ordered

The decision MUST state that agent events are lossless: every message event and every tool event arrives, in order, on every path other than the sanctioned loss paths this decision itself names.

#### Scenario: S-AGE-005 — The lossless claim is stated with named exceptions only

- **GIVEN** the merged decision
- **WHEN** a reviewer enumerates every circumstance in which an event may be lost
- **THEN** every such circumstance is one of the two boundary rules below, AND no third loss circumstance appears anywhere in the artifact

### R-AGE-005 — The loop-internal boundary inherits Layer 1's loss path unchanged

At the loop-internal boundary the decision MUST adopt Layer 1's rule verbatim in force: on cancellation with a saturated buffer, late events are dropped and the stream closes without a terminal event. The decision MUST state why the inheritance is safe here — the loss fires only on the consumer's own cancellation — and MUST record the hazard it leaves open, namely that the two boundaries now carry different loss postures which AG-00 must name distinctly.

#### Scenario: S-AGE-006 — The boundary is named and its hazard recorded

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the backpressure section
- **THEN** the loop-internal boundary is stated separately from the harness-facing one, AND its rule is the unchanged Layer 1 rule, AND the conflation hazard is recorded with the vocabulary milestone named as its mitigation

### R-AGE-006 — The harness-facing boundary may never drop what history already knows

At the harness-facing boundary the decision MUST state a rule **strictly narrower** than Layer 1's: the loss path MUST NOT discard an event describing a fact **already committed to the harness's history**. Bounded wind-down MUST finish delivering every such event as part of, or immediately preceding, the terminal run-end event. The decision MUST cite the **measured** Layer 1 capacity and its consequence — that a buffer is saturated whenever no receiver is already waiting, so the loss path is the ordinary condition rather than the exceptional one — and MUST NOT reason from the superseded capacity hypothesis. It MUST answer the zero-capacity objection ("Layer 2's consumer pauses by design") on its merits rather than noting that it lost.

#### Scenario: S-AGE-007 — A committed side effect cannot be silently absent

- **GIVEN** a run in which a tool has already executed and its outcome is committed to history, and the run is then cancelled
- **WHEN** a reviewer traces the decision's harness-facing rule over that run
- **THEN** the rule requires the event describing that outcome to be delivered before or with run-end
- **AND** a decision that permits that event to be dropped because the buffer was saturated fails this scenario

#### Scenario: S-AGE-008 — The narrowing rests on the measured capacity

- **GIVEN** the merged decision
- **WHEN** a reviewer checks every capacity figure it cites for Layer 1
- **THEN** the cited standing figure is the measured one, AND any mention of the superseded hypothesis is marked historical
- **AND** the "consumer pauses by design" objection is answered by distinguishing a pause on the receive step from a pause on work performed after hand-off

### R-AGE-007 — What remains droppable is stated, so the narrowing is not vacuous

The decision MUST state positively what the harness-facing loss path may still discard: state the harness never learned before cancellation — facts not yet committed to history, such as in-flight events of work cut short. It MUST also state why removing the loss path entirely is unavailable: a consumer that stops reading and never cancels would make the producer wait without bound, contradicting the bounded wind-down obligation.

#### Scenario: S-AGE-009 — Both sides of the line are stated

- **GIVEN** the merged decision
- **WHEN** a reviewer looks for the droppable set
- **THEN** the artifact names at least one class of event that may still be lost, AND explains why a fully lossless harness-facing boundary is unavailable at any price
- **AND** a decision that states only what may not be dropped fails this scenario

### R-AGE-008 — A stalled observer is structurally unable to stall the streaming path

The decision MUST make envelope invariant 3 structural. It MUST name a **decoupling mechanism** — one canonical internal stream, and per attached consumer an independently owned carrier fed by its own forwarding activity applying the same send discipline — such that the canonical producer's progress does not depend on any attached consumer's receive progress. A statement of the obligation, a convention, a review rule, or documentation alone MUST NOT satisfy this requirement. The decision MUST tabulate the rejected mechanisms by what each makes **impossible**, including the blocking synchronous multicast that makes nothing impossible, named to show what "conventional, not structural" looks like.

**Back-annotation (AG-20) — the MECHANICAL half has landed, and the requirement's own standard is what it was measured against.** AG-01.1 answered this requirement for *consumers*, at a time when Layer 2 had no observing hooks and the property was therefore unfalsifiable in code. AG-20.2 lands three observing hook points and the discipline that makes them safe. Four mechanisms are recorded here as **shipped structure**, not as reassurance:

- **Enqueue is a lock-append that never blocks.** The run's fire sites append to a per-run lane under a mutex and continue; they never wait on an observer's progress. That, and not scheduling luck, is the non-blocking property, and it is the same *shape* this requirement demanded of consumers — an independently owned carrier fed by its own activity.
- **Dispatch is on the lane's own goroutine.** An observing hook is never invoked on the goroutine that drives the run, stamps events, or delivers them. AG-20 asserts this as a **goroutine-placement** property — the observer captures its own stack and the test asserts the harness run frame and the forwarder frame are **absent** — because a placement property is what asynchrony actually is, and because the alternative shape (holding a gate and waiting for an observable that a synchronous implementation never produces) is unbounded without a clock, which `R-RUN-010` and six shipped NFRs ban.
- **The observers' context is value-stripped, so `S-AGE-010`'s trace is preserved by CONSTRUCTION.** Observers run on a freshly rooted context — deliberately **not** a cancellation-stripped derivation of the run context, because such a derivation preserves context **values**, and in a hosted child run the run context carries the delegation publishing seam retrievable by a plain value lookup (`delegation_seam.go:101-104`). A value-preserving observer context would hand an **observing** hook the one sanctioned door back onto the **parent's** streaming lane, asynchronously, after `Run` returned — a path from a stalled observer to a producer, which is precisely the path this requirement's scenario forbids. Value-stripping closes it structurally rather than by convention.
- **The stall report is a Go-side typed value and NOT an event, for this requirement's own reason.** An event announcing the stall would itself be a path from the stalled observer back onto the producer's stream. The report is delivered to a nil-defaultable reporter, off the streaming path, and **no `EventKind` is registered**. A budget argument for the same conclusion could be overturned by a budget; this one cannot.

**What the back-annotation does NOT claim.** It does not discharge `R-AGE-009`: AG-20 ships no consumer fan-out, no attachment surface and no second carrier. It does not weaken the "documentation alone MUST NOT satisfy" clause — that clause is the bar AG-20.2 was written to clear. And it does not extend the mechanism to *mutating* hooks, which run inline on the caller's goroutine exactly as AG-08 shipped, by design and out of scope.

(Previously: the requirement carried no record of whether the mechanism it demanded had ever been exercised against a real observing hook. `agent-event-envelope/spec.md:268` reserved invariant 3 as closed by "AG-01.1 + AG-20.2", so a reader after the AG-20 merge could not tell from this requirement whether the second half had landed, nor by what mechanism — and would have had to re-derive from the code whether `S-AGE-010`'s trace still held once observers existed.)

**Back-annotation (AG-21) — the mechanism is now exercised UNDER PRESSURE and IN COMBINATION, and this paragraph closes nothing.** Two things are recorded, and the boundary between them is the point:

- **What AG-21 adds is falsification, not mechanism.** AG-21 ships no production code by default (`R-CNH-008`). It stalls a real consumer **structurally** — an unbuffered sink with no receiver, so the producer is genuinely blocked at its unconditional send rather than merely slow — while the same run is simultaneously suspended, steering, compacting or hosting a child, and then signals it. That combination is what *"under pressure"* means here, and it is the first time the decoupling mechanism is observed against anything other than a single-feature fixture.
- **What this paragraph explicitly does NOT claim.** It does **not** close envelope invariant 3: `agent-event-envelope/spec.md:269` records that as `AG-01.1 + AG-20.2 — CLOSED`, and AG-21 cannot close what is already closed. It does **not** discharge `R-AGE-009`, which remains decided-and-unbuilt. It does **not** satisfy this requirement by prose — the requirement's own clause above forbids exactly that, and **`S-AGE-031` below is the discharge; this paragraph merely records where to find it.** And it does **not** merge the consumer claim with the hook claim: `S-AGE-010`, `S-AGE-030` and `S-AGE-031` are three separate members of one family, and conflating any two of them is the error `agent-event-delivery/spec.md:143` already names.

(Previously, at AG-21: the requirement recorded the mechanism as shipped and asserted against single-feature fixtures only. A reader could not tell from it whether the decoupling had ever been observed while the run was simultaneously under a second kind of pressure — a pending suspension, a queued steer, an in-flight compaction or a live child run — nor whether a structurally stalled consumer, as opposed to a merely slow one, had ever been driven at all. Both were unasserted, and doc 0003's reverse table `0003:2265` traced AG-21 to R-05 with nothing in this capability recording what that trace was owed.)

#### Scenario: S-AGE-010 — The stalled-observer trace has no path back to the producer

- **GIVEN** the decision's named mechanism, and an attached consumer that stops receiving indefinitely and never cancels
- **WHEN** a reviewer traces every path from that consumer's stalled receive back toward the canonical producer
- **THEN** every path terminates at the forwarding activity that privately owns that consumer's carrier, and none reaches the canonical producer or any other consumer
- **AND** a decision whose only defence is a stated obligation, a convention, or a documented "must not block" rule fails this scenario

*(AG-20 update: the assertion this scenario makes is exactly what it was, and it is about a stalled **consumer**. `S-AGE-030` states the parallel claim for a stalled **hook**, which AG-20 makes possible for the first time; the two are separate and must not be conflated.)*

*(AG-21 update: unchanged in claim. `S-AGE-031` states the same family's claim **under pressure and in combination**; it is a third member, not a restatement, and the three must not be conflated.)*

#### Scenario: S-AGE-011 — Rejected mechanisms are judged by impossibility, not preference

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the mechanism table
- **THEN** each row states what that mechanism makes impossible, AND the blocking synchronous multicast row records that it makes nothing impossible, AND the drop-on-overflow row is rejected against `R-AGE-004`

*(AG-20 update: unchanged. AG-20 adds no mechanism to that table; it implements the one already chosen.)*

*(AG-21 update: unchanged. AG-21 adds no mechanism to that table either; it stresses the one already chosen.)*

#### Scenario: S-AGE-030 — AG-20: the stalled-HOOK trace has no path back to the producer either, and it is checked in code

- **GIVEN** the merged AG-20 change and an observing hook held open indefinitely by the module's test gate primitive
- **WHEN** a reviewer traces every path from that hook's stalled invocation back toward the canonical producer
- **THEN** every path terminates inside the observer lane's own drain activity, and **none** reaches the producer, the sink, the stamper or any other consumer
- **AND** the observer's invocation context carries **no** delegation publishing seam, so the one sanctioned door onto a parent's lane is unreachable from a hook — asserted by a hosted child run whose observer looks the seam up and finds none
- **AND** when the run carrying that gated hook is recorded with a sink buffered to the script's full event count, then its event stream is **byte-identical** to the same script with no hooks installed, modulo the freshly minted run and turn identifiers, and `CheckStream` accepts it unmodified
- **AND** `Run` returns **while the gate is still held**, and the gate is released only afterwards
- **AND** no assertion in this scenario reads elapsed time, sleeps or polls
- **AND** a defence resting on a doc comment, a convention or a review rule fails this scenario exactly as it fails `S-AGE-010`

Cross-referenced to `R-HKS-007` / `S-HKS-017` / `S-HKS-018` and `R-HKS-008` / `S-HKS-019`.

#### Scenario: S-AGE-031 — AG-21: the decoupling holds with a STRUCTURALLY stalled consumer while the run is simultaneously under a second pressure

- **GIVEN** a run whose consumer sink is **unbuffered** and whose consumer has read a prefix and then stopped receiving — so the producer is blocked at its unconditional send **by construction**, with no receiver at all, rather than by any timing arrangement
- **AND** that same run is simultaneously in one of the four combined states of `R-CNH-001` — a suspension pending, a steer queued and undelivered, a compaction call in flight, or a child harness active — with the state proven pending by a happens-before edge the production code itself provides, never by elapsed time
- **WHEN** the consumer resumes receiving, and separately when the run is signalled while still stalled
- **THEN** every event describing a fact already committed to the transcript is present on the stream once the consumer drains to completion, checked against the scripted event identity set — count, kinds and call identities — so that a single missing committed-fact event is a divergence (`R-AGE-006`)
- **AND** on the never-cancelled arm the number of events absent is **zero**: `R-AGE-005`'s sanctioned loss fires only on cancellation with a saturated buffer, so on that arm it is unreachable and any absence is unsanctioned and fails this scenario
- **AND** `CheckStream` accepts the drained stream **unmodified**, with `stream_check.go` byte-unchanged
- **AND** on the signalled arm the run **returns**, observed by a read on its completion channel and never by a wall-clock assertion, and the run-end outcome and returned error match the firing signal
- **AND** no assertion in this scenario reads elapsed time, sleeps or polls
- **AND** this scenario asserts nothing about a stalled **hook** — that is `S-AGE-030`'s — and nothing about closing envelope invariant 3, which `AG-01.1 + AG-20.2` already closed
- **AND** a defence resting on the back-annotation paragraph above, on a doc comment, or on a convention fails this scenario exactly as it fails `S-AGE-010`

Discharged directly by `TestCombinedPressure_StalledSteering_NeverCancelled_LosesNothing` and `TestCombinedPressure_StalledSteering_Interrupted` (`slow_consumer_pressure_test.go`), which drive R-CNH-001's steering-queued state and R-CNH-003's structural stall on the SAME run — the literal conjunction this scenario's GIVEN clauses require (sdd-verify round-2, MAJOR-1: the standalone `S-CNH-007`/`S-CNH-008` evidence alone does not exercise it). Cross-referenced to `R-CNH-003` / `S-CNH-007` / `S-CNH-008` and `R-CNH-001` for the single-feature halves.

### R-AGE-009 — Layer 2 supports more than one attached consumer, with no privileged consumer

The decision MUST settle the single-hand-off versus multi-observer fork explicitly. It MUST decide that Layer 2 supports more than one attached consumer per run, MUST state that no consumer is privileged at the mechanism level (privilege is policy, and policy is Layer 3's), and MUST record the losing reading — a single non-blocking hand-off with all fan-out inside Layer 3 — together with the grounds on which it was rejected. Layer 3's further fan-out MUST be described as a second, additional stage of the same pipeline, not an alternative to this one.

#### Scenario: S-AGE-012 — The fork is decided with its rebuttal on the page

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the observer section
- **THEN** the multi-consumer verdict is stated, AND the single-hand-off reading is written down as a coherent reading that was rejected, with its grounds
- **AND** a later reader can distinguish a decision from an oversight without opening the exploration artifact

### R-AGE-010 — Three nested ownership scopes, each with exactly one closer

The decision MUST state, for each of the per-turn, per-run and per-delegated-run scopes, its sole owner and sole closer: the loop for the turn scope, the harness for the run scope, and the child harness for a delegated run with the parent separately owning the subagent bracket on its own stream. It MUST state that ownership is never shared, that nesting is strict and sequential, and that nothing else closes — not a consumer, not a test helper, not a party above the layer.

**Back-annotation (AG-19) — the per-delegated-run scope now exists in fact, and the rule it was written for held without amendment.** AG-04 wrote this scope before any delegated run could exist. AG-19 creates one, and every clause survives literally:

- **The child harness is the sole closer of the child's stream.** The child's sink closes only after the child's run-close was sent (`harness.go:446`), and the parent's tool never closes it.
- **The parent separately owns the subagent bracket on its own stream.** `subagent_started` and `subagent_ended` are parent-lane events, stamped by the parent's dispatcher, and the child never emits them.
- **Nesting is strict and sequential, and the ordering is structural rather than asserted.** The tool publishes `subagent_ended` only after the child's sink closed, so on the parent's lane `index(subagent_ended) < index(parent run-close)` — which is exactly `S-AGE-014`'s "the child's stream fully closes before the parent's representation of it closes", now checkable by a test instead of by a reviewer.
- **Ownership is never shared even with N children.** Sibling tool calls each host a **distinct** `Harness` value with its own context, transcript, stamper and sink; concurrent `Run` calls on one `Harness` value remain out of scope.

#### Scenario: S-AGE-013 — Every scope has exactly one closer

- **GIVEN** the merged decision
- **WHEN** a reviewer enumerates the closers per scope
- **THEN** each of the three scopes has exactly one, AND the turn-scope closer differs from the run-scope closer with the reason stated (the loop is re-instantiated per turn and does not know the run boundary)

#### Scenario: S-AGE-014 — Delegation does not break the ownership rule

- **GIVEN** a delegated run cancelled leaf-first
- **WHEN** a reviewer traces the close order under the decision's rules
- **THEN** the child's stream fully closes before the parent's representation of it closes, AND the parent never closes the child's stream, AND the child never closes the parent's

*(AG-19 update: this scenario's claim is unchanged, and it is now discharged by execution rather than by review — `S-DEL-014` runs exactly this shape and asserts the close order by observed event index on the parent's lane, never by elapsed time.)*

#### Scenario: S-AGE-028 — AG-19: the delegated scope is exercised, and the closers are counted in a running system

- **GIVEN** a parent run whose tool hosts a child harness, and separately two sibling tools each hosting their own child
- **WHEN** the runs complete and every stream is captured
- **THEN** each child's sink is closed exactly once, by that child's own harness, and neither the parent nor a sibling closes it
- **AND** `subagent_started` and `subagent_ended` appear only on the parent's lane, and no child emits either
- **AND** on the parent's lane the index of each `subagent_ended` is strictly less than the index of the parent's run-close
- **AND** the parent's stream and each child's stream are `CheckStream`-valid **validated separately**, never concatenated

### R-AGE-011 — Exactly-one-terminal discipline at the agent level

The decision MUST state that exactly one run-start precedes everything on a run-scoped stream, exactly one run-end follows everything, run-end carries a typed outcome of completed, interrupted or failed, and nothing follows the terminal. It MUST state that the loop emits turn-end on **every** exit path — normal, typed failure and cancellation — with turn-end distinguishing model-finished from turn-aborted by typed outcome. It MUST state that, unlike Layer 1, the run scope has **no** "sometimes no terminal at all" case, and MUST identify `R-AGE-006` as what makes that true.

#### Scenario: S-AGE-015 — A cancelled run still ends with a terminal

- **GIVEN** a run interrupted mid-turn
- **WHEN** a reviewer traces the decision's obligations over that run
- **THEN** one turn-end with the aborted outcome and one run-end with the interrupted outcome are both required, AND nothing is permitted after run-end

### R-AGE-012 — What a consumer may assume after a terminal event

The decision MUST state what a consumer may assume for each run-end outcome: on the completed outcome, everything received is the complete, ordered story; on the interrupted or failed outcomes, the received prefix is trustworthy in the specific sense that nothing already committed to history is missing from it, while what had not yet happened is truncated.

#### Scenario: S-AGE-016 — The assumption is stated per outcome, not once

- **GIVEN** the merged decision
- **WHEN** a reviewer looks up each of the three run-end outcomes
- **THEN** each has a stated consumer assumption, AND the interrupted and failed cases cite the harness-facing narrowing as what makes the prefix trustworthy
- **AND** a decision that states a single undifferentiated "the stream is the complete story" fails this scenario

### R-AGE-013 — One harness-level inbound surface carrying three typed payload kinds

The decision MUST name the harness as the single receiving surface for messages entering a live run, carrying three typed payload kinds: a permission decision, a steering input, and an interrupt. It MUST state that this is structurally **one** surface rather than three parallel paths, and MUST reconcile the two source levels explicitly — the upward arrow drawn at harness level and the suspension described at loop level — recording that the loop level is the destination, not the entry point, because the harness is the stable addressable thing a frontend holds across a whole run while the loop is stateless and re-instantiated per turn.

#### Scenario: S-AGE-017 — The two source sections are reconciled in writing

- **GIVEN** the two source sections that describe the upward path at different levels
- **WHEN** a reviewer reads the upward-path section
- **THEN** the reconciliation is written down with its reason, AND all three payload kinds are stated to share one surface
- **AND** a decision that leaves the level mismatch implicit fails this scenario

### R-AGE-014 — The call-identity lookup exists, is harness-owned, and its shape is not fixed

The decision MUST state **that** a lookup from the call identity carried by the original decision-required event to the specific in-flight suspension exists, that it is harness-owned, and that it lives for the suspension's lifetime. It MUST NOT fix that lookup's shape, structure or storage; those belong to the milestones that build the permission protocol and the run driver.

#### Scenario: S-AGE-018 — Existence decided, contents left open

- **GIVEN** the merged decision
- **WHEN** a reviewer applies the over-reach test — would deleting this sentence give a later milestone more options?
- **THEN** the lookup's existence and ownership are stated, AND no sentence constrains its shape, AND the deferral names the milestones that own the shape

### R-AGE-015 — Typed rejection at two identity granularities, never a silent drop

The decision MUST require a **typed rejection** — never a silent drop and never a silent no-op — for an upward message addressed to a call identity that no longer holds a live suspension within a live run, and for an upward message addressed to a run identity that has fully ended. It MUST state the two carve-outs explicitly: an interrupt arriving **during** bounded wind-down is silently idempotent and is not a rejection case, and pause-resumption is model-initiated and harness-internal and is **not** an instance of the upward path, so it must never be routed through the typed-rejection or call-identity machinery.

#### Scenario: S-AGE-019 — A message to an ended run is rejected typed

- **GIVEN** a run whose run-end has already been emitted
- **WHEN** a frontend sends any of the three payload kinds to that run identity
- **THEN** the decision requires a typed rejection, AND a silent drop is explicitly forbidden

#### Scenario: S-AGE-020 — A redundant interrupt during wind-down is not a rejection

- **GIVEN** a run in bounded wind-down that has not yet emitted run-end
- **WHEN** a second interrupt arrives
- **THEN** the decision requires it to be tolerated idempotently, AND the two states — during wind-down versus after the run ended — are distinguished on the page rather than conflated

#### Scenario: S-AGE-021 — Pause-resumption is carved out

- **GIVEN** the merged decision
- **WHEN** a reviewer looks for the pause-resumption case
- **THEN** it is stated as model-initiated and harness-internal and explicitly excluded from the upward path, with the consuming milestone named

### R-AGE-016 — The upward path recurses under delegation

The decision MUST state that a child harness has its own inbound surface, that what a child's policy scope would ask about is asked on the **parent's** stream, and that the frontend therefore answers through the parent's surface while the parent's routing must reach the nested child's own suspension lookup.

**Back-annotation (AG-19) — `S-AGE-022`'s routing obligation is DISCHARGED, and it is discharged with ZERO new production routing surface.** The obligation this requirement stated was that the routing be *stated rather than left to the delegation milestone to invent*. AG-19 is that milestone, and it invented nothing:

- **Ask up.** The child's `permission_decision_required` is emitted on the child's own stream by the child's own scheduler and is **mirrored** onto the parent's stream through the publishing seam. Its answering `permission_decision_made` crosses **with it**: a mirrored ask with no mirrored answer is unreadable to the human watching, so the pair crosses together or not at all (`R-DEL-002`, `R-DEL-008`).
- **Decision down.** The human's verdict, given on the parent's surface, reaches the child's suspension through the **existing** wake surface (`scheduler.go:264-272`), on the child `Scheduler` value the delegating tool already owns. The gate re-enters resolution on wake (`scheduler.go:772-783`) and the child's derived scope answers with that verdict. **No new routing type, method, channel or registry ships.**
- **What "the parent's policy allowed flows down" operationally means, since the phrase admits a weaker reading.** The child's scope is an ordinary composition of the parent's policy; it may only **narrow** and MUST NOT **widen**, and AG-19 asserts **both** directions rather than only the convenient one (`S-DEL-018`). No scope type, rule set or mode flag enters Layer 2.
- **What is not carried up.** `permission_resolution_remembered` is `CardinalityAtMostOne`, so it is refused at the seam and stays on the child's own stream. A remembered rule remains scoped to one `Schedule` call, exactly as `R-APP-010`'s deferral already says.

#### Scenario: S-AGE-022 — A decision for a nested call reaches the child

- **GIVEN** a call suspended inside a delegated run
- **WHEN** a reviewer traces where the frontend sends the decision and where it must arrive
- **THEN** it is sent to the parent's surface and arrives at the child's suspension, AND the routing obligation is stated rather than left to the delegation milestone to invent

*(AG-19 update: the obligation is now discharged by execution — `S-DEL-019` suspends a child call, observes the ask and its answer on the parent's stream, answers through the existing wake surface and asserts the child's call resumes with that verdict. This scenario's own claim is unchanged; the annotation adds discharge evidence.)*

### R-AGE-017 — Downstream milestones consume this surface rather than inventing their own

The decision MUST close with an inheritance table stating, in each milestone's own terms, what AG-04, AG-10, AG-13, AG-14, AG-19 and AG-20 take from it. It MUST state that none of them may invent its own channel, its own loss rule, or its own way back into a live run; a downstream milestone that finds itself deciding one of these properties is proposing an amendment to this decision, not exercising a judgement call.

**Back-annotation (AG-19) — AG-19's row is EXERCISED UNAMENDED, and the borderline case is adjudicated in the open rather than assumed away.** AG-19 is the first milestone whose whole subject matter is "a way back into a live run", so the row deserves an explicit verdict rather than a reassurance:

- **No channel is invented.** AG-19 reuses the scheduler's existing emission funnel verbatim — the same funnel, the same single stamping dispatcher goroutine, the same sink — that concurrent sibling tool calls already use. It creates no channel, declares no buffer capacity and names no numeric value, so `R-AGE-018` is untouched too.
- **No loss rule is invented.** A mirrored event either reaches the funnel or is refused **typed**, by one of exactly two sentinels: inadmissible (the kind can never cross) or revoked (the hosting call has completed or detached). There is no third outcome, no silent drop and no panic (`R-DEL-002`, `R-DEL-003`).
- **No second stamping writer is introduced.** Each lane keeps exactly one stamping writer: the parent's stamper is touched only by the parent's dispatcher, the child's only by the child's own. Re-stamping discards the prior sequence and returns a copy, so both lanes stay contiguous and are **never merged**.
- **The borderline judgement, stated because this requirement demands it be stated.** What AG-19 adds is a **named exported accessor** onto the channel this decision already owns, reachable only for the duration of one tool call and revoked on every exit path. Read as "a new door", it is new; read as "a new way", it is the existing way given a name and a lifetime. **This delta records the reading as a judgement made in the open, not as an amendment made by silence.** A later milestone that wants a *second* funnel, a buffer, an unrevoked handle, or a publish path that survives its hosting call is proposing an amendment to this decision — and this paragraph is what it will be measured against.

#### Scenario: S-AGE-023 — Every blocked milestone has a row

- **GIVEN** the merged decision
- **WHEN** a reviewer checks the inheritance table against AG-04, AG-10, AG-13, AG-14, AG-19 and AG-20
- **THEN** each has a row stating what it inherits, AND the no-invented-channel rule is stated once for all of them
- **AND** the acceptance criterion is checkable by reading that one table

*(AG-19 update: the table's row count and its milestone list are unchanged — AG-19 was always in it. What changed is that AG-19's row is now discharged. AG-20 remains outstanding.)*

#### Scenario: S-AGE-029 — AG-19: the no-invented-channel rule is checked against code, not against a claim

- **GIVEN** the merged AG-19 change
- **WHEN** its production diff is read
- **THEN** it declares no channel type, no channel creation, no buffer capacity and no numeric capacity anywhere
- **AND** the seam's every refusal path returns one of exactly two sentinels distinguishable by `errors.Is`, with no path that drops an event silently and no path that panics
- **AND** each lane's stamper has exactly one writing goroutine, proven under `-race` with two sibling children running concurrently

### R-AGE-018 — Every deferred number names its owner and its closing evidence

Where the decision deliberately defers a numeric value — in particular any buffer capacity at any Layer 2 boundary — it MUST defer the **posture** decision explicitly rather than silently, MUST name the later milestone that owns measuring it, and MUST state what evidence closes the deferral, mirroring how Layer 1 deferred its own capacity to a measuring milestone with a stated experiment and a tie-break rule. The decision MUST NOT name a starting numeric capacity, and MUST state why naming one without a workload would repeat a mistake with none of the falsification machinery.

#### Scenario: S-AGE-024 — The capacity deferral is a decision, not an omission

- **GIVEN** the merged decision
- **WHEN** a reviewer searches for any numeric buffer capacity attributed to a Layer 2 boundary
- **THEN** none is stated, AND the deferral is recorded with the owning milestone named and the closing evidence described
- **AND** a decision that simply omits the topic fails this scenario

### R-AGE-019 — AG-01's acceptance criterion and authoring constraints hold

The change MUST satisfy AG-01's acceptance criterion: every question in AG-01.1's five-item closing checklist is answered in the recorded decision, each with rationale and a *what this excludes* part, and the decision is **closed before AG-04 starts**. Every Layer 2 noun MUST be cited from AG-00's register by term identifier; any noun the register lacks MUST be appended to that register in the same pull request rather than defined here. No Go type name, field name, method name, or package identifier MUST appear in any artifact of this change; no file under the backend module and no build or dependency file MUST be added or modified.

#### Scenario: S-AGE-025 — The checklist is walked and closed

- **GIVEN** AG-01.1's five closing-checklist items
- **WHEN** a reviewer walks the merged decision against them
- **THEN** each item has an answer with rationale and a *what this excludes* part, AND the change is merged before AG-04's first node begins

#### Scenario: S-AGE-026 — The authoring constraint holds across the whole change

- **GIVEN** every file this change adds
- **WHEN** a reviewer inspects them for Go identifiers and for files outside the change directory
- **THEN** no Go type, field, method or package identifier appears, AND every added file is inside this change's directory, AND no backend, build or dependency file is touched

#### Scenario: S-AGE-027 — Vocabulary is cited, never minted

- **GIVEN** the merged decision and AG-00's published register
- **WHEN** a reviewer resolves each Layer 2 noun the decision uses
- **THEN** each resolves to a register row by identifier, AND any noun the register lacked was appended to the register in this same pull request
