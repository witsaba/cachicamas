# Explore — Layer 2 event delivery and the observer model

> **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 — Decide event delivery and the observer model
> **Node**: AG-01.1 — The delivery decision `[decision]`
> **Phase**: explore (read-only; no production code)
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Target module**: `backend/agent/` — **no code is written by this change**
> **Sources**: [doc 0003 — Layer 2 task graph](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) (Wave 0, Wave 2, Wave 3, AG-19/AG-20) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) (§ 2.2, § 2.3, § 4, § 6, § 7) · [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) (AI-02, AI-03 charters) · [ADR 0005](../../../docs/adr/0005-promote-agent-stack-to-own-module.md) (D1, D3, D4, Enforcement) · [ADR 0007](../../../docs/adr/0007-adopt-dag-convention-for-task-graphs.md) · the archived [AI-02 decision](../archive/2026-07-31-cachicamas-ai-stream-lifecycle/decision.md) · the archived [AI-34 backpressure decision](../archive/2026-08-07-cachicamas-ai-backpressure/decision.md) · the live [Layer 1 stream-lifecycle contract](../../specs/ai-stream-lifecycle/spec.md) · the shipped Layer 1 provider contract and its documentation comments · the shipped Layer 1 test-support sibling package · `openspec/project.md` · `openspec/AGENTS.md` · `openspec/config.yaml`
> **Checked and contributing nothing**: ADR 0004 — permission, upward path, suspension and the G1 concern are v2-only additions and do not appear in it.
> **Authoring constraint**: doc 0003's authoring constraint binds every artifact of this change. **No Go type name, field name, method name, or package identifier appears anywhere.** Language and standard-library shapes are named descriptively, never spelled.

---

## 1. Carrier at the boundary

**Documented default (doc 0003, AG-01 charter note):** channels, matching AI-02, "for symmetry with the Layer 1 carrier decision (AI-02) and for the same reasons." That one line understates the actual argument. AI-02's real argument rests on four grounds, none of them cost-of-change — the sunk-cost argument was explicitly voided because nothing was shipped when AI-02 decided.

| Ground | Argument at Layer 1 | Force at Layer 2 |
| --- | --- | --- |
| 1 — must wait on several things at once | An iterator **is** the producer, running on the consumer's own goroutine. To wait on the stream *and* a permission decision *and* an interrupt at the same time, the consumer must convert the iterator into something waitable — spawning a goroutine and a queue, and reintroducing the exact leak class the iterator claimed to abolish, relocated from one producer to every consumer, uncounted. | **Yes, more forcefully.** AG-10.3's own scenario ("one call is suspended … sibling calls schedule, execute, and emit events; message deltas already in flight keep flowing") is Ground 1 restated as a Layer 2 test. Layer 2 stacks a *second* suspension mechanism — ask, suspend, resume — on top of message and tool delivery. |
| 2 — the consumer must not be the socket reader | Every moment spent handling an event is a moment nobody reads the transport. The honest repair is a goroutine and a queue *inside* the iterator: a hidden buffer, and a hidden buffer is worse than a declared one. | **Yes, by analogy.** Layer 2's "transport" is the loop's own scheduling and permission work. A consumer that pauses to schedule tools or drive the permission protocol must not stall the loop's internal producer. |
| 3 — the terminal event has nowhere clean to go | A pair-yielding shape invites a per-element failure check, contradicting exactly-one-terminal. A single-value shape plus an after-the-loop accessor is ambiguous after an early exit — a consumer that left the loop cannot tell "it failed" from "I stopped". | **Yes.** AG-04.2 defines the identical discipline for the run and turn brackets: one run-start precedes everything, one run-end follows everything, "nothing follows the terminal". |
| 4 — the shape carries the wrong connotation | Iteration connotes repeatable, in-memory, cheap. A stream is single-use, side-effecting, cancellable. | **Yes**, unchanged. Layer 2's stream is single-use, cancellable, and side-effecting: it drives tool execution and permission suspension. |

**What is conceded, explicitly.** The stranded-producer hazard is not literally impossible under channels. It is closed by the **send discipline**, not by the carrier: every send waits on both the stream and cancellation, and the caller owns the context. The residual — a consumer that abandons *and* never cancels — remains a documented contract violation, not testable to termination, because no test proves a goroutine never exits. AI-40.3 restates this at the Layer 1 freeze; AG-01 must carry the equivalent statement into the Layer 2 package contract, because it is the same *shape* of untestable obligation.

**The caller-owns-the-context liveness rule, as Layer 1 actually states it** (the eight ownership rules restated on the shipped stream's own documentation comment; also the archived AI-02 decision § 5):

- The caller supplies a cancellable context on the call that creates the stream; the stream's lifetime is bounded by it.
- Every send waits on both the stream and cancellation — no send is unconditional, the terminal send included.
- A consumer ends a stream in exactly one of two legal ways: **drain to close**, or **cancel**. Anything else is **abandonment**: a documented contract violation, not a supported mode.
- Cancellation closes the stream within bounded time. Once cancellation is observable the producer begins no new blocking wait on the network or on the consumer, and a backoff waits on the signal rather than sleeping.

**Iterator ergonomics, concretely, as shipped.** Layer 1 put them in its test-support sibling package: a small view built over a carrier the caller already holds. It never owns the stream and never closes it — a receive-only carrier cannot even compile a close — and it is explicitly "never a second contract". Its only role is ergonomics over an already-decided carrier.

**Alternatives beyond the iterator, with what each costs:**

- **Push-callback** (an observer registers a callback invoked per event): reintroduces synchronous-observer coupling that directly violates envelope invariant 3, unless every callback is wrapped in its own goroutine and queue — which is Ground 2's hidden buffer, relocated to the harness-to-observer boundary.
- **A sink handle with an explicit close operation**: hands a consumer something that, unless carefully guarded, a non-owner can close — violating "nothing else closes" unless made asymmetric by construction. A receive-only carrier gives that asymmetry for free at the type-system level; a sink adds a manual invariant a carrier enforces mechanically.

**Recommendation.** Keep AI-02's documented default — a receive-only carrier at the Layer 2 package boundary — because Grounds 1 and 3 apply with *equal or greater* force at Layer 2, not merely by symmetry. Iterator ergonomics, if wanted at all, belong in a Layer 2 test-support sibling package, never in the Layer 2 package itself, because R-02 (no I/O of its own) and the production/test closure split (AG-03.2) already draw that exact line. Whether such a convenience is needed in v1 is an AG-02 question, not an AG-01 one — open question 3 below.

## 2. Backpressure posture

**Requirement (doc 0003, AG-01.1 item 2):** agent events are **lossless** — message and tool events must all arrive, in order. This is a materially *stronger* claim than anything Layer 1 owes, because Layer 2's event stream is the **only** contract with everything above it. Doc 0001 § 4.3: "If a thing is not on the stream, no frontend can render it and no session log can reconstruct it."

**What Layer 1's sanctioned loss path actually is today — verified, and load-bearing.** Doc 0003's item 2 cites the *mechanism* ("cancellation on a saturated channel") and names no capacity at all; there is nothing stale in doc 0003 on this point. The load-bearing fact is one level down, in Layer 1's own live contract:

- AI-02 chose a bounded buffer and recorded a **starting capacity of 64 as an explicit hypothesis**, with the measurements that would falsify it.
- AI-34.1 **measured** buffer occupancy, wait frequency, and per-stream memory across three workloads and changed the answer. The live, decided capacity is **`0`** — an unbuffered rendezvous. The Layer 1 contract's summary row carries the superseded 64 struck through, and its "why 64" rationale prose is annotated as *historical, not the standing rationale*.
- **Consequence for the word "saturated".** At capacity zero, a buffer is saturated whenever there is no receiver already waiting. "Saturated" therefore does not describe a rare, load-induced condition — it is the ordinary condition at essentially every send. The sanctioned loss path is far easier to reach than the word suggests, which *strengthens* the case for narrowing it at Layer 2 rather than inheriting it unchanged.

**The sharpest question in this decision, stated in Layer 1's own words.** Layer 1's rejected zero-capacity argument warned, verbatim, that a rendezvous has *"zero tolerance for a consumer that pauses at all — and Layer 2's consumer pauses by design, to drive the permission protocol."* Layer 2 is that consumer. That argument lost to measurement — but the proposal must say *why* it lost rather than noting that it did, because the reason is exactly what makes item 3 structural:

> A rendezvous is intolerant of a consumer that pauses **on the receive step**. It is indifferent to a consumer that pauses on work it performs *after* handing the event off. Layer 2's own design already forbids the first: AG-10.3 requires that a suspended call not block sibling calls and that "message deltas already in flight keep flowing", which is only achievable if the permission protocol, tool scheduling, and cost aggregation happen off the receive path. The zero-capacity warning assumed a consumer doing its per-event work inline on the receive step. Layer 2 is architecturally forbidden from being that consumer. The warning is answered by the loop's own structure, not waived.

**Whether Layer 2 may inherit the rule unchanged — the hazard analysis, at two different boundaries:**

- **Loop-internal boundary** (the loop as consumer of Layer 1's stream, and as producer of turn-scoped events). Layer 1's rule transfers cleanly. The loss only ever fires on the *consumer's own cancellation* — only when the party that would lose events has already decided to stop caring. That is not a lossless-in-order violation; it is the same carve-out Layer 1 already states, and Grounds 1–4 apply identically because this boundary faces Layer 1's exact problem shape.
- **Harness-facing boundary** — the one AG-01.1 item 2 actually names. Inheriting the identical rule is riskier here, because of a fact Layer 1 does not have: **the harness's history (AG-12) is an independent, in-memory source of truth for the same facts the stream carries.** AG-13.1 asserts the event stream is "the complete story" at run scope; AG-14.1 promises that even an interrupted run "ends with the interrupted outcome", which needs a terminal event and orphan-synthesised history to be provable. If the sanctioned loss path silently drops an event describing an **already-committed side effect** — a tool that already ran and wrote a file, not merely a message delta the model can be asked to regenerate — the loss is qualitatively worse than Layer 1's. Layer 1's droppable events are provider deltas that are safely irrelevant once cancelled. Layer 2's could be the only external record that a real-world effect happened.

**Recommendation — a hybrid, not a blind inheritance.** Reuse Layer 1's exact mechanism at the loop-internal boundary, where Layer 1's argument transfers whole. At the harness-facing boundary add one constraint Layer 1 does not need: **the loss path may never discard an event describing a fact already committed to history.** Cancellation may still stop a run early and truncate what has not yet happened, but bounded wind-down (AG-14.3) must finish delivering everything already committed to history as part of, or immediately preceding, the terminal run-end event, rather than leaving that delivery subject to buffer capacity. This still bounds time — AG-14.3's own bound governs it — and does not reopen the waiting-never-dropping discipline. It narrows *what may ever be a candidate for loss* to "state the harness never learned before cancellation", which by construction cannot be smaller than zero.

**Hazards named per option (required by the checklist):**

| Option | Hazard it leaves open |
| --- | --- |
| Inherit Layer 1's rule unchanged at both boundaries | A session log cannot trust the stream after any interrupted run without cross-checking history out of band — silently, because Layer 1's "a missing terminal after your own cancellation makes you the party in error" clause does not tell a consumer *what* it is missing. Layer 2 would possess ground truth its own primary contract can silently fail to convey. |
| The hybrid | Two different loss postures at two internal boundaries. The vocabulary (AG-00) must name them distinctly so AG-13, AG-16 and AG-18 do not conflate turn-scoped Layer-1-style loss with run-scoped history-guarded loss. |
| No loss path at all at the harness boundary | A consumer that stops reading and never cancels makes the producer wait forever, which converts a bounded wind-down into an unbounded one and contradicts AG-14.3 directly. Full losslessness is not available at any price. |

## 3. The observer model

**Requirement (doc 0001 § 4.3 invariant 3, restated by AG-01.1 item 3):** observers are never synchronous on the streaming path; a slow listener must not stall token delivery. AG-01.1 requires this be made **structural**, by decoupling **mechanism**, not convention. AG-20.2 is the mechanical test of it: "a deliberately stalled observing hook … delivery is unimpeded."

**Candidate mechanisms, by what each makes impossible:**

| # | Mechanism | Makes impossible | Cost / caveat |
| --- | --- | --- | --- |
| 1 | Independent per-consumer carrier, each fed by its own forwarding task reading one canonical internal stream, each applying its own send discipline | One slow consumer stalling **any** other, including the run's "primary" consumer — there is no privileged primary at the mechanism level; every attached consumer is symmetrically one more observer of the canonical stream | Mirrors AI-02's send discipline at a second boundary. Each forwarding task closes only the carrier it privately owns, never the canonical stream: "nothing else closes", applied recursively |
| 2 | Bounded per-consumer buffer with drop-on-overflow instead of waiting | Any consumer propagating backpressure upstream — there is nowhere for the pressure to go | Conflicts directly with the lossless requirement for any consumer slower than the producer. Tolerable only for a genuinely secondary, non-authoritative observer (a cost meter), never for one whose correctness the run depends on |
| 3 | Blocking synchronous multicast (every observer must accept before the stream advances) | **Nothing.** This is exactly what invariant 3 forbids | Named only to show what "conventional, not structural" looks like — the rejected default |
| 4 | Pull-based replay record (observers read an append-only in-memory record at their own pace; the producer never touches an observer's carrier) | Any coupling whatsoever between producer progress and observer progress | Requires retaining a growing in-memory record for the run's lifetime. Layer 2 performs no persistence (R-02), so it is memory-only and must be freed at run end. "How far behind may an observer fall" is an open bound this mechanism does not answer for free |

**Recommendation:** mechanism 1, with an explicit statement that there is no privileged consumer at the mechanism level — only at the policy level, which is Layer 3's.

**A genuine, unresolved fork the decision must settle openly.** AG-01.1 item 3 names "a session logger, a cost meter" attaching to Layer 2 directly, implying Layer 2 is itself a multi-consumer boundary. Doc 0001 § 2.2 draws exactly **one** upward emission arc from the harness — the harness re-emits, enriched, to a single Layer 3 event type, which then fans out to the frontends. Read literally, § 2.2 suggests Layer 2 needs only **one** non-blocking hand-off per run, with the actual multi-observer fan-out happening entirely inside Layer 3's re-emission, which is unconstrained by Layer 2's no-I/O rule. The two readings are not automatically reconcilable from the sources as written. Given that "how a second consumer attaches" sits in AG-01's own closing checklist, the safer reading is that Layer 2 *does* support at least a second attached consumer per run — mechanism 1 answers that — and Layer 3's further fan-out to N frontends is a *second, additional* stage layered on top. They are not mutually exclusive; they are two stages of one pipeline. This must be recorded as an explicit decision with its rebuttal, not left implicit.

## 4. Close and ownership rules

Layer 1's shipped rule: the producer creates the stream and closes it exactly once; one goroutine sends; one closing site exists; it runs on every exit path, after the last send attempt; nothing else closes it — not the consumer, not a test helper, not a consumer above the layer. Applied to Layer 2's three nested scopes:

| Scope | Owner and sole closer | Terminal discipline |
| --- | --- | --- |
| Per-turn | The loop (stateless) — sole producer of that turn's message and tool events, bracketing them in turn-start and turn-end | "Turns nest strictly inside the run and never overlap" (AG-04.2). Turn-end distinguishes model-finished from turn-aborted by typed outcome. On every exit path — normal, typed failure, cancellation — the loop is the one party that emits turn-end |
| Per-run | The harness (stateful) — a **different owner** from the turn scope, because the loop is re-instantiated per turn and does not know the run boundary | Exactly one run-start precedes everything, exactly one run-end follows everything, and run-end carries a typed outcome: completed, interrupted, or failed (AG-04.2). "Nothing follows the terminal" is pinned by the same validator the turn scope uses |
| Per-delegated-run (nested) | The child harness owns its own run-scoped stream. The **parent** harness separately owns the subagent-started and subagent-ended bracket on its **own** stream, re-emitting the child's already-closed events, parent-identified | Ownership is never shared — it is strictly nested and sequential. Leaf-first cancellation (AG-19.2) means the child's stream fully closes before the parent's representation of it closes. This is "nothing else closes" applied recursively, not violated by delegation |

**What a consumer may assume after a terminal event**, as the direct analogue of Layer 1's own clause: a consumer that receives run-end may treat everything it saw as the complete, ordered story **unless** run-end's typed outcome says interrupted or failed — in which case item 2's hybrid guarantee (nothing already committed to history is missing) is what lets it still trust what it received. Layer 2 does not have Layer 1's "sometimes no terminal at all" case at run scope, precisely because the hybrid protects run-end delivery.

## 5. The upward path (R-09)

**The only upward arrow in doc 0001 § 2.2** runs from the frontend to the **harness**, not the loop: a permission decision that resumes a suspended turn, annotated "there is an upward arrow from the frontend, and it is the only one … It is not the frontend driving the loop." The turn-sequence diagram in § 2.3, however, describes the suspension entirely at the **loop** level: the loop emits the permission request to the frontend, "the turn SUSPENDS here. Nothing blocks. Other calls proceed," and the frontend answers the loop.

These are not contradictory once reconciled, but no source document currently ties them together. Reconciling them is AG-01.1 item 5's job, and the reconciliation is: **the harness is the stable, addressable surface a frontend can hold across a whole run** — the loop is stateless and re-instantiated per turn — so the harness is architecturally the right *receiving* surface, and it routes the decision down to whichever in-flight loop invocation holds the matching suspended call.

**What item 5 must decide concretely:**

1. **The surface a frontend calls:** the harness, matching § 2.2's drawn arrow, and the natural surface for all three upward-message kinds R-09 names — permission decision, steering input, interrupt. AG-13.2 already shows steering entering via the run driver; AG-14 shows interrupt as a harness-level signal. **Recommendation: state explicitly that all three share one surface.** R-09's "one decided upward path" is not three parallel paths that happen to look similar; it is structurally one harness-level inbound surface carrying three typed payload kinds.
2. **How a decision finds its suspended call:** by the call identity the original decision-required event carried (AG-10.1). The harness must therefore hold, or reach, a lookup from call identity to the specific in-flight suspension, for that suspension's lifetime. AG-01 decides **that** this lookup exists and is harness-owned — not **what** it is, matching AI-02's standing rule that the container is settled here and the contents are not.
3. **An upward message addressed to a run that already ended:** a typed rejection, never a silent drop. AG-10.1 already states the identical discipline one level down, at call-identity granularity: "a stray decision … rejected as a typed protocol error, never a silent no-op". **Recommendation:** generalise it as one rule instantiated at two identity granularities — call identity within a live run, and run identity itself — decided once here rather than reinvented by AG-10 and AG-13 independently.

**A carve-out worth making explicit.** Pause-resumption (AG-13.3) is model-initiated — a finish reason — not frontend-initiated. It is harness-internal: the harness resumes its own driving loop after a typed turn outcome. **Recommendation: state explicitly that pause-resumption is not an instance of R-09's upward path**, even though both involve something resuming a suspended-looking state, so AG-13 does not route it through the typed-rejection and call-identity machinery meant for frontend-originated messages.

**Downstream consumers, and what each needs from this decision:**

- **AG-10 (permission protocol):** the surface, the call-identity lookup confirmed as existing and harness-owned, and the typed-protocol-error discipline for stray or late decisions. AG-10.1 currently reads as a forward reference — "the loop implements that decision, it does not invent a channel" — and AG-01 is the milestone that must name the path.
- **AG-13 (multi-turn run driver):** the same harness-level surface for steering input, the run-already-ended typed rejection generalised from AG-10.1's call-level precedent, and the explicit pause-resumption carve-out.
- **AG-14 (cancellation tree):** interrupt is one of R-09's three kinds. AG-14.1 and AG-14.2 describe what happens *after* interrupt or shutdown fires, not how it *enters* — that entry is AG-01's. Two states need distinguishing: an interrupt arriving **during** wind-down tolerates a redundant signal silently (AG-14.1's idempotence scenario); an interrupt arriving **after** the run has fully ended requires item 5's typed rejection. AG-01 must state the distinction so AG-14 does not conflate them.
- **AG-19 (delegation):** a child harness has its own harness-level surface, but AG-19.3 states that what a child's policy scope would ask about "is asked on the parent's stream — one place a human watches." The upward path therefore recurses: a decision for a call made inside a subagent still must reach the *child's* suspended call, but the frontend answers through the **parent's** surface, and the parent's routing must reach into a nested child's own suspension lookup. **Recommendation:** state this recursion explicitly, rather than leave AG-19 to invent cross-harness routing.
- **AG-20 (hook taxonomy):** AG-20.2 is the mechanical test of invariant 3 and inherits whichever mechanism item 3 picks, including what "eventually reported typed" means for a stalled observer.

## Conflicts across doc 0001, 0002 and 0003

1. **The meaning of "saturated" is not what the phrase suggests** (detailed under item 2). Doc 0003's item 2 is mechanism-accurate and names no capacity, so **doc 0003 needs no correction**. The live Layer 1 capacity is `0` — an unbuffered rendezvous, measured and fixed by AI-34.1 — which makes the sanctioned loss path reachable at essentially every send rather than only under load. Cite the Layer 1 contract's § 6 and AI-34.1's archived decision directly; do not paraphrase the superseded hypothesis.
2. **Structural level mismatch** between doc 0001 § 2.2 (permission decision arrives at the harness) and § 2.3 (suspension described at loop level, resumed by a loop-directed message). Reconciled under item 5 above; unreconciled in the sources themselves prior to this decision.
3. **Single hand-off versus multi-observer boundary** (detailed under item 3): AG-01.1 item 3's "a session logger, a cost meter" language versus doc 0001 § 2.2's single drawn re-emission arc. Not resolved silently here — flagged as an explicit fork AG-01.1 must pick, with its rebuttal recorded.
4. **Symmetry versus inheritance.** AI-02's own decision is explicit that "Layer 2 decides its own carrier … the symmetry is a recommendation with reasons, never a substitute" for AG-01's own argument. Doc 0003's charter-note phrasing ("for symmetry … and for the same reasons") could be misread as closure by citation. It is not; AG-01.1 must argue the point independently, as § 1 above does.

## Open questions the proposal must settle

1. Does Layer 2 host multiple simultaneous attached consumers per run, or exactly one, with Layer 3 responsible for all further fan-out? (item 3)
2. Does the harness-facing stream apply Layer 1's sanctioned loss path unchanged, or does it need the "never drop what history already knows" qualification? (item 2)
3. Is there a Layer 2 equivalent of Layer 1's carrier view, and if so where does it live given R-02 and the production/test closure split? Most likely a Layer 2 analogue of Layer 1's test-support sibling package, decided by whichever milestone builds Layer 2's test substrate — not by AG-01.
4. The exact shape of "how a decision finds its suspended call" is intentionally left to AG-10 and AG-13: the container is settled here, the contents are not.
5. Confirm the pause-resumption carve-out is recorded explicitly so AG-13 does not conflate it with frontend-originated messages.

## What belongs to AG-00 (vocabulary) rather than here

- Precise definitions of run, turn, suspension, steering message, delegation and the parent relationship. AG-01 uses these terms and must not redefine them.
- The loop's six must-nevers and the harness's one must-never, restated as vocabulary-level obligations (AG-00.1 item 3). AG-01 cites that restatement rather than re-deriving it.
- Whether Layer 2's envelope wraps Layer 1 payloads or reuses Layer 1 identities as-is (AG-00.1 item 2). AG-01 assumes such an envelope exists and decides how it travels, not what it contains.
- **Process note:** AG-00 is being decided in parallel in the same wave and pull request; its artifact was not available during this exploration. The proposal must confirm no naming conflict once AG-00's register is published, and cite it by term identifier — mirroring how AI-02 cites AI-01's register by row identifier throughout — rather than paraphrasing.

## What belongs to AG-02 (v1 scope) rather than here

Whether the permission concern, delegation and re-entrancy, hooks, compaction and parallel tools actually ship in v1. AG-01 decides the delivery *mechanism* assuming these exist as seams; AG-02 decides how much of each is implemented versus a trivial stub. AG-01's answers — carrier, backpressure, ownership, upward-path surface — do not change based on AG-02's verdict, because a stubbed seam still needs the same rules the moment it is exercised even once, including in a test.

## Recommendation summary

| Checklist item | Recommended answer | Confidence |
| --- | --- | --- |
| 1. Carrier | A receive-only carrier, matching AI-02, on independently-argued Grounds 1 and 3 — stronger at Layer 2, not merely symmetric | High |
| 2. Backpressure | Hybrid: Layer 1's exact rule at the loop-internal boundary; a strictly narrower loss surface ("never drop what history already knows") at the harness-facing boundary | Medium — needs AG-01.1 to state it as one explicit decision, not leave it to AG-13 and AG-16 to improvise |
| 3. Observer model | Independent per-consumer carrier with its own send discipline (mechanism 1); the single-hand-off-versus-multi-observer fork picked openly, with its rebuttal recorded | Medium — a genuine source ambiguity, flagged rather than silently resolved |
| 4. Close and ownership | Three nested scopes — turn/loop, run/harness, delegated-run/child-harness — each with its own sole owner-closer, mirroring Layer 1 recursively | High |
| 5. Upward path | One harness-level surface for all three upward-message kinds; typed rejection at both stray-call and ended-run granularity; explicit pause-resumption carve-out; explicit delegation recursion | High |

## Risks

- The "saturated means every send" finding (item 2) is the highest-value correction this exploration carries. Reasoning from the superseded capacity hypothesis would make the sanctioned loss path look rarer than it is and weaken the case for narrowing it.
- The single-hand-off-versus-multi-observer ambiguity (item 3) is a genuine fork in the sources. Picking wrong has real cost later: AG-06's delegation event family and AG-19's nested-consumer routing both assume an answer.
- AG-00's register is not yet available. Any vocabulary AG-01 mints ad hoc risks colliding with AG-00's parallel decision in the same pull request.

## Ready for proposal

**Yes.** All five checklist items have a recommended, argued answer. The open questions above are explicit decision points for the proposal to resolve. AG-01.1 is a `[decision]` node, so the deliverable follows the archived AI-02 decision's structure: a what-was-decided summary table first, then one section per checklist item with a *What this excludes* part, then a *Who inherits it* table for AG-04, AG-10, AG-13, AG-14, AG-19 and AG-20.
