# Proposal — Layer 2 event delivery and the observer model

> **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 — Decide event delivery and the observer model
> **Node**: AG-01.1 — The delivery decision `[decision]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Driver**: braejan
> **Wave**: 0 of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) — decided before any event exists
> **Scope**: documentation only — `openspec/changes/cachicamas-agent-event-delivery/`. **Zero Go. Zero `go.mod` change. Zero files under `backend/`.**
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AG-00 (`cachicamas-agent-contract-vocabulary`) — decided in parallel, same wave, same pull request
> **Blocks**: AG-02, AG-04 — and, through them, AG-10, AG-13, AG-14, AG-19, AG-20
> **Authoring constraint**: doc 0003's authoring constraint binds every artifact of this change. No Go identifier appears anywhere.

---

## Intent

Close AG-01.1's five-item closing checklist in one merged artifact, so that AG-04 can define the envelope and AG-10, AG-13, AG-14, AG-19 and AG-20 can be written without any of them inventing its own channel, its own loss rule, or its own way back into a live run.

The milestone exists because the delivery contract is the layer's only contract with everything above it. Doc 0001 § 4.3 states the test that decides what belongs on the stream: *"If a thing is not on the stream, no frontend can render it and no session log can reconstruct it."* A contract that carries that much weight, settled *after* producers exist, is settled against five merged consumers instead of against a blank page — the same multiplier doc 0001 § 3.2 records for every wave-0 contract change, and the reason G13's Layer 1 analogue was scheduled at AI-02 rather than discovered at AI-24.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can separate the inherited from the decided.

1. **The vocabulary is AG-00's.** Run, turn, suspension, steering message, delegation and the parent relationship are cited from its register by term identifier, never redefined here.
2. **No Go identifiers.** Doc 0003's authoring constraint. Later milestones choose spellings.
3. **No production code.** A `[decision]` leaf produces none, by doc 0003's node grammar.
4. **Layer 2 performs no I/O of its own** (R-02) and imports only Layer 1, the standard library, and the OpenTelemetry API (R-01). Every mechanism proposed here lives inside that.
5. **Envelope invariant 3 is doc 0001's, not this decision's**: observers are never synchronous on the streaming path. This decision makes it structural; it does not get to weaken it.
6. **The run and turn bracketing discipline is AG-04.2's** — one run-start, one run-end, turns nested and non-overlapping, nothing after the terminal. This decision states who owns each bracket, not what the brackets are.
7. **Layer 1's contract is frozen and is an input.** Its carrier, its send discipline, its ownership rule, and its one sanctioned loss path are cited from the live Layer 1 stream-lifecycle contract, never paraphrased.

## Scope

### In scope — one pull request

| Artifact | Content |
| --- | --- |
| `explore.md` | The option space, argued: the four grounds tested at Layer 2, the mechanism table, the source conflicts, the open questions |
| `proposal.md` | This file |
| `specs/agent-event-delivery/spec.md` | Requirements, each a checkable property of the decision artifact |
| `design.md` | The structure `decision.md` implements and the reasoning methods it applies |
| `tasks.md` | The single leaf AG-01.1 — one task per closing-checklist item, plus the verification pass |
| `decision.md` | **The deliverable.** Five decisions, each with rationale, *what this excludes*, and a *who inherits it* table |

### Out of scope — explicit, including deferred-but-related

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Event kinds, payloads, ordering invariants, the envelope's contents | AG-04, AG-05, AG-06 | This decision fixes how events *travel*, never what they *say* |
| Whether Layer 2's envelope wraps Layer 1 payloads or reuses Layer 1 identities | AG-00.1 item 2 | Vocabulary's, and being decided in parallel |
| Whether the permission concern, delegation, hooks, compaction or parallel tools ship in v1 | AG-02 | A stubbed seam needs the same delivery rules the moment it is exercised once, including in a test. AG-01's answers do not move on AG-02's verdict |
| The concrete shape of the call-identity-to-suspension lookup | AG-10, AG-13 | The container is settled here; the contents are not — AI-02's standing rule, applied |
| A numeric buffer capacity for any Layer 2 boundary | **Deferred, deliberately** | Layer 1's own capacity was a hypothesis until AI-34.1 measured it. Naming a number here without a workload would repeat the mistake with none of the falsification machinery. This decision fixes the *posture*; a later milestone measures |
| A Layer 2 carrier-view convenience, and where it would live | **Deferred, deliberately** | R-02 and the production/test closure split (AG-03.2) already place it outside the Layer 2 package. Whether it is wanted at all is scope, not delivery |
| What "eventually reported typed" means for a stalled observer | AG-20.2 | This decision supplies the mechanism that makes the report possible |
| Layer 3's fan-out to N frontends, and any policy about which consumer matters | Layer 3 | There is no privileged consumer at the mechanism level. Privilege is policy |

## The five decisions this proposal commits to

Stated in one line each so a reviewer can accept or reject the substance before reading the full artifact. Each is argued in `decision.md`.

### 1. Carrier — a receive-only carrier, argued at Layer 2, not inherited by symmetry

AI-02's four grounds are reproduced and tested **at Layer 2**, because AI-02's own decision forbids closure by citation: *"Layer 2 decides its own carrier … the symmetry is a recommendation with reasons, never a substitute."* Ground 1 (a consumer must wait on the stream, a permission decision, and an interrupt at once — an iterator cannot be waited on) applies **more forcefully** here than at Layer 1, because AG-10.3 turns it into a Layer 2 test and Layer 2 stacks a second suspension mechanism on top of delivery. Ground 3 (the terminal event has nowhere clean to go) applies unchanged against AG-04.2's bracketing. Grounds 2 and 4 apply by analogy, with the loop's own scheduling and permission work standing in for the transport read.

**Alternatives and what each costs:** a push-callback re-creates synchronous-observer coupling that violates invariant 3 unless every callback is wrapped in its own decoupling — Ground 2's hidden buffer, relocated. A sink handle with an explicit close hands a non-owner the ability to close, which a receive-only carrier forecloses at the type-system level for free.

**Conceded, explicitly:** the stranded-producer hazard is closed by the **send discipline**, not by the carrier. The residual — a consumer that abandons *and* never cancels — is a documented contract violation, not testable to termination. This is the same untestable obligation AI-40.3 restates at the Layer 1 freeze, and it goes into the Layer 2 package contract in the same terms.

**The caller-owns-the-context liveness rule is adopted as Layer 1 states it**: the caller supplies the cancellable context on the creating call; every send waits on both the stream and cancellation, the terminal send included; the two legal endings are drain-to-close and cancel; cancellation closes within bounded time, with a backoff that waits on the signal rather than sleeping.

**Iterator ergonomics belong outside the package.** Layer 1 put them in its test-support sibling package as a view that never owns and never closes the carrier, "never a second contract". Layer 2 inherits that placement rule; whether it ever builds one is deferred above.

### 2. Backpressure — a hybrid, with the harness-facing rule strictly narrower than Layer 1's

Agent events are **lossless**: message and tool events all arrive, in order.

**What "saturated" actually means today.** Doc 0003's item 2 cites the mechanism and names no capacity, and needs no correction. The load-bearing fact is one level down: Layer 1's capacity was a hypothesis of 64 until AI-34.1 measured it and fixed it at **`0`** — an unbuffered rendezvous — with the superseded figure struck through and the "why 64" prose annotated as historical. At capacity zero, a buffer is saturated whenever no receiver is already waiting. The sanctioned loss path is therefore reachable at essentially every send, not only under load. That strengthens the case for narrowing it here.

**The sharpest objection, answered rather than waived.** Layer 1's rejected zero-capacity argument warned, verbatim, that a rendezvous has *"zero tolerance for a consumer that pauses at all — and Layer 2's consumer pauses by design, to drive the permission protocol."* Layer 2 is that consumer. The argument lost to measurement, and the reason it lost is what makes item 3 structural: a rendezvous is intolerant of a consumer that pauses **on the receive step**, and indifferent to one that pauses on work performed *after* handing the event off. AG-10.3 already forbids the first — a suspension must not block sibling calls, and deltas in flight keep flowing — which is only achievable if permission, scheduling and cost work happen off the receive path. The warning assumed a consumer doing per-event work inline. Layer 2 is architecturally forbidden from being that consumer.

**The decision, at two boundaries:**

| Boundary | Rule | Hazard it leaves open |
| --- | --- | --- |
| Loop-internal | Layer 1's rule unchanged: on cancellation with a saturated buffer, late events drop and the stream closes without a terminal. It fires only on the consumer's own cancellation — only when the party that would lose events already stopped caring | The turn-scoped and run-scoped loss postures differ, and AG-00 must name them distinctly so AG-13, AG-16 and AG-18 never conflate them |
| Harness-facing | Layer 1's mechanism **plus one constraint Layer 1 does not need**: the loss path may never discard an event describing a fact **already committed to history**. Cancellation still truncates what has not yet happened; bounded wind-down must finish delivering what history already holds, as part of or immediately preceding run-end | Bounded wind-down (AG-14.3) now carries a delivery obligation as well as a time bound, and AG-14.3's bound is what keeps it bounded |

**Why not the two alternatives.** Inheriting unchanged at both boundaries would let the stream silently omit an already-committed side effect — a tool that already ran and wrote a file — while the harness's history (AG-12) still holds it; a session log could then not trust the stream after any interrupted run without an out-of-band cross-check, and Layer 1's "you cancelled, so you are the party in error" clause does not tell a consumer *what* it lost. Removing the loss path entirely is unavailable at any price: a consumer that stops reading and never cancels would make the producer wait forever, converting a bounded wind-down into an unbounded one, in direct contradiction of AG-14.3.

### 3. Observer model — one canonical stream, an independently-fed carrier per attached consumer

Invariant 3 becomes structural by **decoupling mechanism**: each attached consumer receives its own carrier, fed by its own forwarding task that reads one canonical internal stream and applies the same send discipline (wait on both the destination and cancellation). Each forwarding task closes only the carrier it privately owns and never the canonical stream — "nothing else closes", applied recursively.

**What each candidate makes impossible** (not what it discourages):

| Mechanism | Makes impossible | Verdict |
| --- | --- | --- |
| Independent per-consumer carrier with its own send discipline | One slow consumer stalling **any** other, including the run's primary — no consumer is privileged at the mechanism level | **Chosen** |
| Bounded per-consumer buffer, drop on overflow | Any consumer propagating backpressure upstream | Rejected — conflicts with item 2's lossless requirement for any consumer slower than the producer |
| Blocking synchronous multicast | **Nothing** | Rejected — this is exactly what invariant 3 forbids; named to show what "conventional, not structural" looks like |
| Pull-based in-memory replay record | Any coupling between producer progress and observer progress | Rejected for v1 — requires retaining a growing record for the run's lifetime and leaves "how far behind may an observer fall" unanswered |

**The fork, picked openly with its rebuttal recorded.** AG-01.1 item 3 names a session logger and a cost meter attaching to Layer 2 directly. Doc 0001 § 2.2 draws exactly one upward emission arc from the harness, which then fans out inside Layer 3. **Decision: Layer 2 supports more than one attached consumer per run.** *Rebuttal, recorded rather than buried:* § 2.2 read literally needs only a single non-blocking hand-off, with all real fan-out inside Layer 3, which is unconstrained by R-02 and may use any mechanism it likes. That reading is coherent and is rejected on two grounds — AG-01.1's own checklist asks how a *second consumer* attaches, which presumes one exists; and the chosen mechanism subsumes the single-consumer case at no cost, whereas the reverse is not true. Layer 3's further fan-out to N frontends remains a **second, additional stage** of the same pipeline, not an alternative to this one.

### 4. Close and ownership — three nested scopes, one sole closer each

| Scope | Sole owner and closer | Terminal discipline |
| --- | --- | --- |
| Per-turn | The loop (stateless) — sole producer of that turn's message and tool events | It brackets them in turn-start and turn-end, and emits turn-end on **every** exit path: normal, typed failure, cancellation. Turn-end distinguishes model-finished from turn-aborted by typed outcome |
| Per-run | The harness (stateful) — a **different owner**, because the loop is re-instantiated per turn and does not know the run boundary | One run-start precedes everything, one run-end follows everything, and run-end carries a typed outcome: completed, interrupted, or failed. Nothing follows the terminal |
| Per-delegated-run | The child harness owns its own run-scoped stream; the parent separately owns the subagent bracket on its **own** stream, re-emitting the child's already-closed events, parent-identified | Ownership is never shared — strictly nested and sequential. Leaf-first cancellation (AG-19.2) means the child's stream fully closes before the parent's representation of it closes |

**What a consumer may assume after a terminal event.** A consumer that receives run-end may treat everything it saw as the complete, ordered story **unless** run-end's typed outcome says interrupted or failed — in which case decision 2's harness-facing guarantee is what still makes the received prefix trustworthy: nothing already committed to history is missing from it. Layer 2 has no "sometimes no terminal at all" case at run scope, precisely because decision 2 protects run-end delivery.

### 5. The upward path — one harness-level surface, three payload kinds, typed rejection at two granularities

**Reconciling doc 0001 § 2.2 with § 2.3.** § 2.2 draws the only upward arrow into the harness; § 2.3 describes the suspension at loop level. Both are right at their own level, and no source ties them together. **The harness is the receiving surface** because it is the stable, addressable thing a frontend can hold across a whole run — the loop is stateless and re-instantiated per turn. The harness then routes the message down to whichever in-flight loop invocation holds the matching suspended call. § 2.3's loop-level drawing is the *destination*, not the entry point.

1. **The surface a frontend calls:** the harness — one inbound surface carrying three typed payload kinds (permission decision, steering input, interrupt). R-09's "one decided upward path" is structurally one surface, not three parallel paths that happen to look alike.
2. **How a decision finds its suspended call:** by the call identity the original decision-required event carried. This decision fixes **that** a call-identity-to-suspension lookup exists and is harness-owned, for the suspension's lifetime. It does not fix what that lookup is.
3. **An upward message addressed to a run that already ended:** a **typed rejection, never a silent drop**. AG-10.1 already states this at call-identity granularity; it is generalised here as one rule at two granularities — call identity within a live run, and run identity itself.

**Carve-out.** Pause-resumption is model-initiated (a finish reason) and harness-internal. **It is not an instance of R-09's upward path**, and must not be routed through the typed-rejection or call-identity machinery meant for frontend-originated messages.

**Recursion under delegation.** A child harness has its own inbound surface, but what a child's policy scope would ask about is asked on the **parent's** stream — one place a human watches. The upward path therefore recurses: the frontend answers through the parent's surface, and the parent's routing must reach a nested child's own suspension lookup.

## Who inherits this decision

None of these may invent its own channel.

| Milestone | What it takes from this decision |
| --- | --- |
| **AG-04** (envelope and ordering) | The carrier the envelope travels on; the three ownership scopes that give run-start/run-end and turn-start/turn-end their owners; "nothing follows the terminal" as an owned obligation rather than a validator rule alone |
| **AG-10** (permission protocol) | The harness-level surface; the call-identity lookup confirmed as existing and harness-owned; the typed-protocol-error discipline for stray or late decisions; item 2's assurance that a suspension never stalls delivery |
| **AG-13** (multi-turn run driver) | The same surface for steering input; the ended-run typed rejection generalised from AG-10.1's call-level precedent; the pause-resumption carve-out, so it is not routed as an upward message |
| **AG-14** (cancellation tree) | Interrupt as one of the three payload kinds; the distinction between an interrupt arriving **during** wind-down (idempotent, silently tolerated) and **after** the run ended (typed rejection); the harness-facing delivery obligation that bounded wind-down must satisfy |
| **AG-19** (delegation and re-entrancy) | Strictly nested, never shared ownership; leaf-first close ordering; the upward-path recursion — the frontend answers on the parent's surface, the parent routes into the child's suspension lookup |
| **AG-20** (hook taxonomy) | The decoupling mechanism that makes AG-20.2's stalled-observer test pass structurally, and that makes "eventually reported typed" expressible at all |

## Capabilities

> Contract between this proposal and `sdd-spec`.

### New capabilities

- `agent-event-delivery`: how Layer 2 events reach consumers — carrier, backpressure posture at both internal boundaries, the observer decoupling mechanism, close and ownership across three nested scopes, and the upward path into a live run.

### Modified capabilities

- None. `ai-stream-lifecycle` is **cited as a frozen input** and is not amended. Layer 1's contract is unchanged by this decision.

## Approach

1. **Argue the carrier at Layer 2, not by symmetry.** Reproduce AI-02's four grounds and test each against a Layer 2 source — AG-10.3 for Ground 1, AG-04.2 for Ground 3 — because AI-02 itself forbids closure by citation.
2. **Correct the intuition about "saturated" before using it.** Cite the measured capacity `0` and its consequence directly from the live Layer 1 contract, so the loss path is reasoned about at its real frequency.
3. **Answer the zero-capacity objection instead of noting that it lost.** The reason it lost — pauses must not sit on the receive step — is the same reason the observer model must decouple. One argument, used twice.
4. **Tabulate mechanisms by what each makes impossible.** "Discourages" is convention; "makes impossible" is structure, and AG-01.1 asked for structure.
5. **Pick the forks openly and record the rebuttals.** Both genuine source ambiguities — multi-observer versus single hand-off, and the § 2.2/§ 2.3 level mismatch — are decided in the artifact with the losing reading written down, so a later reader can tell a decision from an oversight.
6. **Apply Layer 1's ownership rule recursively rather than restating it three times.** One rule, three scopes, one table.
7. **Close by inheritance.** The artifact ends with what each blocked milestone receives, in that milestone's own terms, so the acceptance criterion is checkable by reading one table.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-agent-event-delivery/` | Six new markdown files | None — new directory |
| `openspec/specs/` | One new capability spec at promotion time | None until archive |
| `docs/architecture/` | **None.** Doc 0003 needs no amendment: its item 2 cites the mechanism, not a capacity | — |
| `openspec/specs/ai-stream-lifecycle/spec.md` | **None.** Cited as a frozen input | — |
| `backend/agent/` | **None** | — |
| `go.mod`, `go.work`, `docker-compose.yaml`, `infra/` | **None** | — |

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| The carrier reads as a restated default rather than an argued decision | Medium | High — AI-02 explicitly forbids closure by citation | Each of the four grounds is tested against a named Layer 2 source; the alternatives and their costs are stated; a spec requirement makes "the argument cites a Layer 2 source, not only AI-02" checkable |
| A later reader reasons from the superseded capacity hypothesis and concludes the loss path is rare | Medium | High — it would justify inheriting Layer 1's rule unchanged | The decision cites the measured `0` and its consequence explicitly, and states that "saturated" is the ordinary condition, not the exceptional one |
| The two-boundary loss posture is conflated into one | Medium | High — a run-scoped drop of a committed side effect | AG-00 is asked to name the two postures distinctly; the decision states them in one table with the hazard each leaves open |
| The multi-observer fork is picked wrong | Low | Medium | The rebuttal is recorded in the artifact. The chosen mechanism subsumes the single-consumer case, so the cost of being wrong is an unused capability rather than a rework |
| Vocabulary minted here collides with AG-00's parallel register | Medium | Low | Every Layer 2 noun is cited from AG-00 by term identifier. Any noun AG-00 lacks is flagged for appending to its register in this same pull request, never defined here |
| Over-reach into AG-04, AG-10 or AG-13 | Medium | Medium | AI-02's test applied literally: if a sentence were deleted, would a later milestone have more options? If yes and the milestone is not AG-01, the sentence is cut. The call-identity lookup is the worked example — its existence is decided, its shape is not |
| A consumer above Layer 2 closes something it does not own | Low | High | "Nothing else closes" is stated per scope and applied recursively through delegation; the receive-only carrier makes it a type-level impossibility for the ordinary case |

## Rollback plan

The change is additive documentation in a new directory. **Rollback is `git revert` of the single commit.** Nothing is generated from these files, nothing imports them, no build depends on them, and no existing file is modified — so the revert is complete by construction and cannot leave a dangling citation.

Two qualifications matter:

- **Partial rollback is not meaningful and must not be attempted.** The five decisions are coupled: the carrier determines whether a producer exists and therefore what ownership must guarantee; the observer mechanism depends on the carrier; item 2's harness-facing rule depends on item 4's run-scope owner being the harness. Reverting one section would leave the others citing an answer that no longer exists. If any decision is rejected in review, reject the change and re-propose.
- **Post-merge reversal is a different cost and is why this is scheduled in wave 0.** Once AG-04 defines the envelope and AG-10, AG-13 and AG-14 consume the upward path, the carrier and the loss posture are embedded in an envelope contract, a permission protocol, a run driver and a cancellation tree. Doc 0001 § 3.2 prices a wave-0 contract change reopened later at roughly three times the original.

Because AG-00 lands in the same pull request, reverting this change alone leaves AG-00 intact and self-consistent — AG-00 defines terms and does not depend on this decision. The reverse is not true: reverting AG-00 alone would strip the terms this artifact cites, so the two revert together or not at all.

## Dependencies

- **AG-00** (`cachicamas-agent-contract-vocabulary`) — **hard**, decided in parallel in this wave and pull request. Every Layer 2 noun used here is cited from its register by term identifier.
- **AI-02 / AI-34** — the frozen Layer 1 stream contract, cited as an input. Not amended.
- No new Go dependency, and no ADR required: this change adds no code and no module requirement.

## Success criteria

Restated from AG-01's charter — *"the decision answers every question in the closing checklist and is closed before AG-04 starts"* — so `sdd-spec` can turn each into a requirement.

1. All five closing-checklist items are answered in `decision.md`, each with rationale and a *what this excludes* part.
2. The carrier decision reproduces AI-02's four grounds and argues each **at Layer 2** against a named Layer 2 source, not by symmetry alone; the alternatives and their costs are stated; the abandonment concession is present.
3. The caller-owns-the-context liveness rule appears as Layer 1 states it, and iterator-view ergonomics are placed outside the Layer 2 package with a stated reason.
4. The backpressure posture is stated per boundary, with the hazard each option leaves open, and cites the measured Layer 1 capacity rather than the superseded hypothesis.
5. The zero-capacity objection — "Layer 2's consumer pauses by design" — is addressed head-on, and its answer is the same argument that makes the observer model structural.
6. The observer mechanisms are tabulated by what each makes **impossible**; the chosen one decouples by mechanism, not convention; the multi-observer fork is picked with its rebuttal recorded.
7. Close and ownership are stated for all three nested scopes, each with a sole closer, and what a consumer may assume after a terminal event is stated for each run outcome.
8. The upward path names one surface, states that the call-identity lookup exists and is harness-owned without fixing its shape, requires a typed rejection at both granularities, records the pause-resumption carve-out, and states the delegation recursion.
9. Doc 0001 § 2.2 and § 2.3 are explicitly reconciled, and the reconciliation is written down rather than assumed.
10. AG-04, AG-10, AG-13, AG-14, AG-19 and AG-20 each have a stated inheritance in one table.
11. No Go identifier appears in any artifact of this change.
12. No file outside `openspec/changes/cachicamas-agent-event-delivery/` is modified.

## Notes for the following phases

- **`spec.md`** — the system under test is the artifact, as it was for AI-02. Every scenario must be checkable by inspection, without running anything. Requirement and scenario identifiers follow the repo's `R-`/`S-` convention with a capability-specific infix.
- **`design.md`** — owns the structure of `decision.md` and the three reasoning methods it applies: argue-at-this-layer for the carrier, made-impossible-not-discouraged for the observer mechanisms, and pick-the-fork-and-record-the-rebuttal for the two source ambiguities.
- **`tasks.md`** — five tasks, one per closing-checklist item, plus the AG-00 vocabulary reconciliation and the verification pass.
- **`decision.md`** — the deliverable. Ends with the inheritance table, because the acceptance criterion is stated in terms of what the blocked milestones can do without reopening this one.

## Proposal question round

No blocking product question was raised before writing, and none of the four items below blocks `sdd-spec` or `sdd-design`. They are recorded for the driver to confirm, correct, or expand into a second round.

| # | Question | Assumption taken |
| --- | --- | --- |
| 1 | Should Layer 2 support more than one attached consumer per run in v1, or is a single hand-off with all fan-out in Layer 3 the intended product shape? | **More than one.** AG-01.1 item 3 presumes a second consumer exists; the chosen mechanism subsumes the single-consumer case at no cost. Rebuttal recorded in decision 3 |
| 2 | Is "an already-committed side effect must never be silently absent from the stream" a real product obligation, or is an out-of-band history cross-check acceptable for session logs? | **A real obligation.** Doc 0001 § 4.3 makes the stream the only contract upward, and a session log that must cross-check history out of band contradicts it |
| 3 | Should a frontend addressing an ended run receive a typed rejection even when the message is a redundant interrupt during wind-down? | **No** — that specific case stays silently idempotent per AG-14.1; the typed rejection applies once the run has fully ended. The two states are distinguished explicitly |
| 4 | Is deferring every numeric capacity to a later measuring milestone acceptable, or is a starting hypothesis wanted now? | **Defer.** Layer 1's hypothesis survived one milestone before measurement overturned it; naming a number here without a workload repeats that with none of the falsification machinery |
