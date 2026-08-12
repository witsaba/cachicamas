# Spec — Layer 2 agent event delivery and the observer model

> **Capability**: `agent-event-delivery` (new) · **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 — Decide event delivery and the observer model · **Node**: AG-01.1 `[decision]`
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-AGE-0NN` · **Scenario IDs**: `S-AGE-0NN`
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

#### Scenario: S-AGE-010 — The stalled-observer trace has no path back to the producer

- **GIVEN** the decision's named mechanism, and an attached consumer that stops receiving indefinitely and never cancels
- **WHEN** a reviewer traces every path from that consumer's stalled receive back toward the canonical producer
- **THEN** every path terminates at the forwarding activity that privately owns that consumer's carrier, and none reaches the canonical producer or any other consumer
- **AND** a decision whose only defence is a stated obligation, a convention, or a documented "must not block" rule fails this scenario

#### Scenario: S-AGE-011 — Rejected mechanisms are judged by impossibility, not preference

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the mechanism table
- **THEN** each row states what that mechanism makes impossible, AND the blocking synchronous multicast row records that it makes nothing impossible, AND the drop-on-overflow row is rejected against `R-AGE-004`

### R-AGE-009 — Layer 2 supports more than one attached consumer, with no privileged consumer

The decision MUST settle the single-hand-off versus multi-observer fork explicitly. It MUST decide that Layer 2 supports more than one attached consumer per run, MUST state that no consumer is privileged at the mechanism level (privilege is policy, and policy is Layer 3's), and MUST record the losing reading — a single non-blocking hand-off with all fan-out inside Layer 3 — together with the grounds on which it was rejected. Layer 3's further fan-out MUST be described as a second, additional stage of the same pipeline, not an alternative to this one.

#### Scenario: S-AGE-012 — The fork is decided with its rebuttal on the page

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the observer section
- **THEN** the multi-consumer verdict is stated, AND the single-hand-off reading is written down as a coherent reading that was rejected, with its grounds
- **AND** a later reader can distinguish a decision from an oversight without opening the exploration artifact

### R-AGE-010 — Three nested ownership scopes, each with exactly one closer

The decision MUST state, for each of the per-turn, per-run and per-delegated-run scopes, its sole owner and sole closer: the loop for the turn scope, the harness for the run scope, and the child harness for a delegated run with the parent separately owning the subagent bracket on its own stream. It MUST state that ownership is never shared, that nesting is strict and sequential, and that nothing else closes — not a consumer, not a test helper, not a party above the layer.

#### Scenario: S-AGE-013 — Every scope has exactly one closer

- **GIVEN** the merged decision
- **WHEN** a reviewer enumerates the closers per scope
- **THEN** each of the three scopes has exactly one, AND the turn-scope closer differs from the run-scope closer with the reason stated (the loop is re-instantiated per turn and does not know the run boundary)

#### Scenario: S-AGE-014 — Delegation does not break the ownership rule

- **GIVEN** a delegated run cancelled leaf-first
- **WHEN** a reviewer traces the close order under the decision's rules
- **THEN** the child's stream fully closes before the parent's representation of it closes, AND the parent never closes the child's stream, AND the child never closes the parent's

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

#### Scenario: S-AGE-022 — A decision for a nested call reaches the child

- **GIVEN** a call suspended inside a delegated run
- **WHEN** a reviewer traces where the frontend sends the decision and where it must arrive
- **THEN** it is sent to the parent's surface and arrives at the child's suspension, AND the routing obligation is stated rather than left to the delegation milestone to invent

### R-AGE-017 — Downstream milestones consume this surface rather than inventing their own

The decision MUST close with an inheritance table stating, in each milestone's own terms, what AG-04, AG-10, AG-13, AG-14, AG-19 and AG-20 take from it. It MUST state that none of them may invent its own channel, its own loss rule, or its own way back into a live run; a downstream milestone that finds itself deciding one of these properties is proposing an amendment to this decision, not exercising a judgement call.

#### Scenario: S-AGE-023 — Every blocked milestone has a row

- **GIVEN** the merged decision
- **WHEN** a reviewer checks the inheritance table against AG-04, AG-10, AG-13, AG-14, AG-19 and AG-20
- **THEN** each has a row stating what it inherits, AND the no-invented-channel rule is stated once for all of them
- **AND** the acceptance criterion is checkable by reading that one table

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
