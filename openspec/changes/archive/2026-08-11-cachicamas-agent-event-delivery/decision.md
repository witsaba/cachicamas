# Decision — Layer 2 event delivery and the observer model

> **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 — Decide event delivery and the observer model
> **Node**: AG-01.1 — The delivery decision `[decision]`
> **Status**: decided
> **Date**: 2026-08-11
> **Project**: cachicamas (witsaba) · **Target module**: `backend/agent/` (Layer 2) — this change touches none of it
> **Closes**: doc 0003's AG-01.1 closing checklist, five items
> **Sources**: [doc 0003 — Layer 2 task graph](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) §§ 2.2, 2.3, 4, 6, 7 · the live [Layer 1 stream-lifecycle contract](../../specs/ai-stream-lifecycle/spec.md) · the archived [AI-02 decision](../archive/2026-07-31-cachicamas-ai-stream-lifecycle/decision.md) · the archived [AI-34 backpressure decision](../archive/2026-08-07-cachicamas-ai-backpressure/decision.md)
> **Binding vocabulary**: [AG-00's Layer 2 register](../cachicamas-agent-contract-vocabulary/specs/agent-contract-vocabulary/spec.md) — every Layer 2 noun below is one of its rows, cited by identifier, never redefined here

> [!IMPORTANT]
> **This artifact decides behavior, not code.** No Go type name, field name, method name, or package identifier appears here — doc 0003's authoring constraint. Later milestones choose spellings. Where a language or standard-library shape has to be discussed, it is named descriptively rather than spelled, so that the constraint holds even inside the argument that most tempts a reader to break it. "Channel", "goroutine", "context" and "buffer" appear as ordinary vocabulary for concurrency primitives — never as the name of a declared Layer 2 symbol, because none exists yet.

---

## 1. How to use this document

**If you are writing AG-04, AG-10, AG-13, AG-14, AG-19 or AG-20:** § 10 tells you what your milestone inherits, in your milestone's own terms. Start there, then read the one decision section (§ 3 … § 7) it points at. You should not need the other four decisions.

- **AG-04** (the envelope and its lifecycle brackets): read § 3 (the carrier) and § 6 (ownership and the terminal discipline).
- **AG-10** (the permission protocol): read § 5 (the observer decoupling, for why a suspension never stalls delivery) and § 7 (the upward path, for the surface and the typed-rejection discipline).
- **AG-13** (the multi-turn run driver): read § 7 (the surface for steering input, the ended-run rejection, the pause-resumption carve-out).
- **AG-14** (the cancellation tree): read § 4 (the two-boundary backpressure rule your bounded wind-down must satisfy) and § 7 (the interrupt-during-wind-down carve-out).
- **AG-19** (delegation and re-entrancy): read § 6 (leaf-first close order) and § 7 (the upward-path recursion).
- **AG-20** (the hook taxonomy): read § 5 (the decoupling mechanism your stalled-observer test exercises).

**If you are reviewing this artifact:** § 12 walks AG-01.1's closing checklist against it, item by item, with the evidence location for each. § 4 and § 5 are where a defect is most expensive — § 4's *the case that breaks naive inheritance* and § 5's *the mechanism, and where the stalled trace terminates* are the two places a reader should look first for a hole in the argument.

**If you disagree with a decision:** each of § 3 … § 7 carries a *What this excludes* part naming the alternatives and why each lost, and § 11 rule 4 says what to do if your objection is not there.

**Every Layer 2 noun below resolves to a row in [AG-00's register](../cachicamas-agent-contract-vocabulary/specs/agent-contract-vocabulary/spec.md)**, cited by identifier rather than paraphrased. Two rows the register needed — `VL2-EVT-18` **loop-internal turn-scoped loss** and `VL2-EVT-19` **harness-facing history-guarded truncation** — were appended to it by AG-00 on this node's behalf, at this node's own recorded request, so that this change's amendment duty under the register's `R-AGV-013` is satisfied by citation rather than by a second append. § 4 cites both; it does not define them, and it does not touch AG-00's register file.

**Section order is AG-01.1's closing-checklist order**, so a reviewer can walk doc 0003 and this document in parallel: § 3 is item 1, § 4 is item 2, § 5 is item 3, § 6 is item 4, § 7 is item 5.

---

## 2. What was decided

Five conclusions, before any argument, for the reader who came for one of them.

| # | Question | Decision |
| --- | --- | --- |
| 1 | **Carrier** (`VL2-EVT-17`) | A **receive-only carrier** at the Layer 2 package boundary, argued independently at Layer 2 against `VL2-LOOP-07` and `VL2-COR-13`'s suspension requirement and against the run/turn bracket discipline — not accepted by symmetry with Layer 1. Iterator-view ergonomics, if ever built, are deferred to **AG-23.2** and placed outside the Layer 2 package |
| 2 | **Backpressure** (two boundaries) | Agent events are **lossless**, with exactly two named exceptions. The loop-internal boundary keeps Layer 1's sanctioned loss path unchanged (`VL2-EVT-18`). The harness-facing boundary is strictly narrower (`VL2-EVT-19`): it may never discard an event describing a fact already committed to `VL2-HAR-01` history. No numeric capacity is named at either boundary; the posture is decided, and `AG-21.2`'s own correctness scenarios — not a published number — are what close the deferral |
| 3 | **Observer model** (`VL2-SEAM-11`, `VL2-SEAM-12`) | One canonical internal stream with exactly one receiver; per attached consumer, an independently owned carrier fed by its own forwarding activity, absorbing the gap between producer and consumer progress for the **run's own extent** and nothing longer. More than one attached consumer is supported, with no consumer privileged at the mechanism level |
| 4 | **Close and ownership** | Three nested scopes — turn (the loop), run (the harness), delegated run (the child harness) — each with exactly one owner and exactly one closer. Delegation closes **leaf-first**: the child's stream fully closes before the parent's re-emission of it |
| 5 | **The upward path** (`VL2-COR-09`) | One harness-level inbound surface carrying three typed payload kinds — permission decision, `VL2-COR-12` steering, interrupt. A message addressed to a call or a run that can no longer receive it is a **typed rejection**, never a silent drop, at two granularities. Pause-resumption is carved out as not an instance of this path at all |

---

## 3. Decision 1 — the carrier

**Closing-checklist item 1.** What this decides: the shape of the thing a Layer 2 consumer holds at the package boundary, and the liveness rule that governs every send on it. What this excludes: what rides on that carrier (`VL2-EVT-01` … `VL2-EVT-16`, AG-04's), and whether a second, ergonomic view over it is ever built (AG-23.2's).

### Decision

The carrier (`VL2-EVT-17`) at the Layer 2 package boundary is a **receive-only carrier of agent events**. The harness (`VL2-COR-03`) hands an attached consumer something it may only receive from; the party that owns a given send-side (the distribution step's per-consumer forwarding activity, § 5) sends and closes it; the consumer receives until it is closed. This is decided **independently at Layer 2**, against Layer 2 sources, not accepted because Layer 1 (`V-STR-02`) decided the same for its own, structurally different boundary.

### Why

AI-02's own decision forbids the shortcut this section might otherwise take: *"Layer 2 decides its own carrier … the symmetry is a recommendation with reasons, never a substitute."* A carrier section whose only support is "Layer 1 decided the same" fails `S-AGE-001` outright, so each of AI-02's four grounds is reproduced below and re-tested against a **named Layer 2 source** — not against Layer 1's problem shape, which no longer applies once the runtime has its own suspension mechanism stacked on delivery.

#### First, the case for an iterator, at its strongest — restated at Layer 2

The reasons an iterator tempts a reader do not weaken by moving up a layer; if anything the second suspension mechanism sharpens two of them. They deserve a hearing before the rebuttal:

1. The stranded-producer hazard would stop being a hazard: production would run on the consumer's own concurrent unit of work, one event at a time, and a consumer that stops iterating simply unwinds.
2. Abandonment would become a supported operation rather than a violation with no test to prove it terminates.
3. The backpressure question would dissolve: no capacity to name, nothing to saturate, no loss path to defend before AG-21.2 tests anything.
4. Ergonomics: an iteration-shaped call site composes cleanly with the standard library and reads better than a receive loop.

That is the case, restated honestly. It loses on the same four grounds, each now tested against Layer 2's own charter rather than Layer 1's.

#### Ground 1 — Layer 2 must wait on several things at once, and an iterator cannot be waited on

**This ground is stronger at Layer 2, not merely inherited, and it cites a Layer 2 source: `AG-10.3`.** `AG-10.3`'s own scenario requires that while one call is suspended, sibling calls schedule, execute, and emit events, and message deltas already in flight keep flowing — proved with synchronization points holding one call suspended while others complete. A consumer sitting inside a loop over an iterator is not waiting on anything; it *is* the producer, executing it. To wait on this stream **and** on a permission decision **and** on an interrupt at once, such a consumer would have to convert the iterator into something waitable — spawning a concurrent task and a queue, reintroducing the exact leak class the iterator claims to abolish, relocated from one producer to every consumer, uncounted. Layer 2 stacks a **second** suspension mechanism — `VL2-LOOP-07`'s ask–suspend–resume protocol — directly on top of message and tool delivery, which Layer 1 never had to do. A receive-only carrier is waitable by construction; the multiplexing `VL2-COR-13` requires is one language construct away, with nothing additional to leak.

#### Ground 2 — the consumer must not be the transport reader, by analogy

At Layer 2 there is no socket; the analogous resource is the loop's own scheduling and permission work. A consumer that pauses to drive `VL2-LOOP-07`'s protocol, schedule tool execution, or aggregate cost must not stall the loop's own progress toward the next event. The honest repair under an iterator is the same hidden goroutine-and-queue Layer 1 rejected, moved somewhere no contract names it and no milestone can measure it — worse than a declared one for the identical reason AI-02 gave.

#### Ground 3 — the terminal event has nowhere clean to go

**This ground applies unchanged, and it cites a Layer 2 source: `AG-04.2`.** `AG-04.2` requires exactly one run-start preceding everything, exactly one run-end following everything with a typed outcome, and nothing after the terminal — the identical discipline Layer 1's `V-STR-18` states one layer down. A pair-yielding iterator shape invites a per-element failure check that contradicts "exactly one terminal"; a single-value shape plus an after-the-loop accessor leaves a consumer that exited early unable to distinguish "the run failed" from "I stopped receiving". A carrier of events needs neither: the terminal event is an element like any other, and closure is the end.

#### Ground 4 — the shape carries the wrong connotation, unchanged

An iteration shape connotes repeatable, in-memory, cheap. A Layer 2 delivery is single-use, side-effecting — it drives tool execution and a live permission suspension — and cancellable. Handing one back in a collection-walking shape teaches a consumer a physics the runtime does not have.

#### What is conceded

Ground 1's rebuttal is not that a stranded producer becomes impossible under a receive-only carrier — it does not. The hazard is closed by the **send discipline** (below), not by the carrier's shape: every send waits on both the destination and cancellation, and the caller owns the cancellable signal. The residual — a consumer that abandons and never cancels — remains, is a documented contract violation, is not preventable by any type, and is not testable to termination. This is the same shape of untestable obligation AI-40.3 restates at the Layer 1 freeze; § 9 below carries the equivalent statement into the Layer 2 package contract.

### The caller-owned liveness rule, adopted unweakened

Layer 1's four clauses, mapped one for one, with no clause weakened and no third ending admitted:

| Layer 1 clause | Layer 2 counterpart |
| --- | --- |
| The caller supplies a cancellable signal on the call that creates the stream; its lifetime is bounded by that signal | Unchanged. Every attached consumer's carrier is bounded by a cancellable signal supplied on the call that attaches it |
| Every send waits on both the destination and cancellation; no send is unconditional | Unchanged, and it binds the terminal send too: run-end and turn-end wait on the same discipline as every other event |
| The two legal consumer endings are drain-to-close and cancel; anything else is abandonment | Unchanged. A third ending is not introduced anywhere in this decision |
| Cancellation closes within bounded time, with backoff waiting on the signal rather than sleeping | Unchanged, and it is what makes `VL2-HAR-05` bounded wind-down's time bound achievable at the delivery layer rather than merely at the harness's control logic |

### Choice — the carrier-view convenience owner

`R-AGE-003` requires a named owner for the question of whether an ergonomic view over the carrier is ever built. **The answer: any such convenience lives outside the Layer 2 runtime package, in a test-support sibling — never inside the package Layer 2 itself ships.** It never owns and never closes the carrier it views (receive-only forecloses that at the type level for the party holding the carrier itself; a view adds no send-side). It is never a second contract: it does not change what an attached consumer may observe, only how convenient it is to observe it. **Whether such a view is built at all is `AG-23.2`'s call** (test kit and examples, wave 6 of doc 0003) — its charter already commits to shipping "a scripted-harness kit... importable... sibling to the Layer 1 kit conventions" and "runnable package examples covering... consuming events", which is exactly the surface a carrier view would sit inside if AG-23.2 decides to build one. This decision does not decide that it will; it decides where the question is asked and answered, and it is the same placement rule Layer 1 already exercises for its own carrier view. **This routing is not invented here either: AG-01's own charter entry in doc 0003 already states it** — "The iterator-view ergonomics live in the test kit" — so `AG-23.2`'s ownership matches what AG-01's own governing milestone note commits to, independently of the `AG-23.2` charter language cited above.

### What this excludes

| Excluded | Why |
| --- | --- |
| A receive-and-callback-driven iteration shape at the boundary | Grounds 1–4 above, each now tested at Layer 2 |
| A pair-yielding shape carrying a failure beside every element | Contradicts `AG-04.2`'s exactly-one-terminal discipline (Ground 3) |
| A single-value shape plus an after-the-loop failure accessor | Ambiguous after an early exit (Ground 3) |
| Both a receive-only carrier and an iteration shape offered at the boundary | Not a compromise — the union of both costs: two dialects for every consumer, every conformance property proved twice, an example written against one that does not work against the other |
| A send-capable handle given to the consumer | The consumer never sends and never closes. Receive-only is the type-level half of § 6's ownership rule, foreclosing a whole class of "nothing else closes" violations for free |
| A push-callback registered per observer | Reintroduces synchronous-observer coupling — the exact thing envelope invariant 3 (`VL2-EVT-14`) forbids — unless every callback gets its own decoupling, which relocates Ground 2's hidden buffer to every observer, uncounted. § 5 makes the correct decoupling structural instead |
| A carrier-view convenience decided here, rather than deferred | Would answer a question `AG-23.2`'s charter is better positioned to answer once a test substrate exists to build it against |

### Consequences

1. A distribution step's forwarding activity exists per attached consumer (§ 5), which is what makes this section's ownership rule and § 4's send discipline load-bearing at the observer attachment point, not merely at the package boundary.
2. `AG-23.2` owns the ergonomics question outright; this decision states the placement rule and stops.
3. No amendment is required anywhere in doc 0003's wave sequence as a result of this decision — the documented default (a receive-only carrier resembling Layer 1's) is confirmed, but confirmed **on its own argument**, which is what makes it citable rather than assumed by later milestones.

### Who inherits it

`AG-04` (the envelope travels on it), `AG-10` (the harness-level surface of § 7 is itself an attached consumer's counterpart on the way up — the carrier's discipline is what the suspension lookup's answer eventually rides on), `AG-23.2` (the carrier-view question, named above).

---

## 4. Decision 2 — backpressure, two boundaries

**Closing-checklist item 2.** What this decides: exactly which circumstances may ever lose an agent event, stated per internal boundary, with the fact that makes the harness-facing boundary different from Layer 1's problem named and traced. What this excludes: the numeric capacity at either boundary (`AG-21.2`'s), and any third circumstance in which an event may be lost.

### Decision

Agent events are **lossless**: every message event and every tool event arrives, in order, on every path other than the two boundary rules stated here. No third loss circumstance exists anywhere in this decision.

**At the loop-internal boundary** — between the loop (`VL2-COR-02`) and the harness (`VL2-COR-03`) — the rule is Layer 1's, unchanged: on the harness's own cancellation of a turn with a send in flight, late turn events drop and the turn's stream closes without its terminal. This posture is named `VL2-EVT-18` **loop-internal turn-scoped loss**.

**At the harness-facing boundary** — between the harness and each attached consumer (`VL2-SEAM-11`) — the rule is Layer 1's mechanism **plus one constraint Layer 1 does not need**: the loss path may never discard an event describing a fact already committed to `VL2-HAR-01` history. Bounded wind-down (`VL2-HAR-05`) finishes delivering every such event as part of, or immediately preceding, the terminal run-end event. This posture is named `VL2-EVT-19` **harness-facing history-guarded truncation**.

### Why

#### What "saturated" actually means, before any rule is argued from it

Layer 1's live capacity is **`0`** — an unbuffered rendezvous, measured and fixed by AI-34.1, with the superseded starting hypothesis of 64 struck through in the live contract and its "why 64" rationale annotated historical. At capacity zero, a buffer is saturated whenever no receiver is already waiting: **saturation is the ordinary condition at essentially every send**, not a load state reached only under pressure. Every argument in this section reasons from that frequency, not from the superseded hypothesis. A reader who finds this decision reasoning from "saturation is rare" has found a defect (`S-AGE-008`).

#### The case that breaks naive inheritance

```
t0  a tool executes; it writes a file — a real-world side effect
t1  the harness appends the result to history            (history now knows)
t2  the result's event is offered toward a consumer's carrier;
    no receiver is waiting at that instant                (ordinary at capacity 0)
t3  the user interrupts; the run's cancellation fires

Layer 1's rule, copied unchanged:  cancellation + saturated ⇒ drop late events,
                                    close without a terminal
Result of the unchanged copy:      the stream omits a side effect history holds,
                                    and omits run-end itself — a consumer sees a
                                    bare close and cannot even enumerate its loss
```

Layer 1's clause "a missing terminal after your own cancellation makes you the party in error" assigns blame; it does not tell the consumer *what* is missing — and at Layer 2, what is missing can be the only external record that a real-world effect happened. A session log built from the unmodified rule disagrees with reality, silently, after every interrupted run. That is the defeat of naive inheritance, and it is why the harness-facing rule must be strictly narrower than the loop-internal one.

#### Why the loop-internal boundary may inherit Layer 1's rule unchanged

The loss fires only on the harness's own cancellation of the turn it is itself consuming — the party that would lose events has already decided to stop caring, at the one place Layer 1's argument was built to cover: a single consumer, at the boundary it owns, choosing to stop. Nothing about Layer 2's own extra suspension machinery changes that shape. The hazard this leaves open is recorded, not hidden: the two boundaries now carry **different** loss postures, and a downstream milestone that conflates `VL2-EVT-18` with `VL2-EVT-19` has made exactly the mistake this decision exists to prevent. Both terms are cited by identifier everywhere below, never paraphrased, for that reason.

#### The two-boundary rule, and how the rules compose

| Boundary | Term | Rule | What may be lost | What may never be lost |
| --- | --- | --- | --- | --- |
| Loop-internal | `VL2-EVT-18` | Layer 1's, unchanged: on the harness's own cancellation of a turn with a send in flight, late turn events drop and the turn stream closes without its terminal | Turn events the harness had not yet received | Nothing more is promised — the harness is the only consumer at this boundary, and the loss fires only on its own cancellation |
| Harness-facing | `VL2-EVT-19` | Layer 1's mechanism **plus**: the loss path never discards an event describing a fact already committed to `VL2-HAR-01` history; bounded wind-down (`VL2-HAR-05`) finishes delivering everything history holds, as part of or immediately preceding run-end | Facts the harness never learned before cancellation: in-flight deltas of a turn cut short, progress of tools that never completed, everything of turns that never began | Any event describing a committed history fact, and run-end itself |

**History is the watershed that makes the two rules compose without a gap.** An event dropped at the loop-internal boundary was, by construction, never appended to history — the harness cannot append what it never received — so the harness-facing guarantee owes nothing about *that* event. There is no path by which a committed fact leaks out of both rules, because "committed" is defined by the harness's own append, which happens strictly after the loop-internal receive that `VL2-EVT-18` governs.

**The composed worst case — cancelled mid-tool-result.** A turn is cancelled while a tool result is in flight at the loop-internal boundary, and the event carrying it drops there under `VL2-EVT-18`. The tool's side effect is real, but history never saw the result event itself. `VL2-HAR-02` orphan synthesis now fires, as `VL2-HAR-01`'s own boundary requires: interruption synthesises a result for the orphaned call, typed as an interruption artifact, before history closes. **The synthesised result is itself a committed history fact** — it is what `VL2-HAR-01`'s validation admits into history for that call — so `VL2-EVT-19`'s rule requires *its* delivery before run-end, even though the original in-flight event never arrives. The guarantee is anchored on history, not on any one event instance: the original event may drop; the fact it would have carried may not, once history holds a record of it.

#### The zero-capacity objection, and where it is answered

Layer 1's own rejected zero-capacity argument warned that a rendezvous tolerates no consumer that pauses at all, and Layer 2's consumer pauses by design to drive the permission protocol. The objection is answered, not waived — and answered in full exactly once, in § 5's *the rendezvous objection, answered on its merits*, which this section's own claim depends on: **a rendezvous is intolerant of a pause on the receive step, and indifferent to a pause on work performed after hand-off.** This section states that conclusion so a reader of the backpressure rule alone is not left with an unanswered objection on the page, without repeating here the full proof — the verbatim Layer 1 warning, and why `AG-10.3` makes the suspension's placement a tested property rather than a hope — that § 5 alone carries.

### What remains droppable, and why the narrowing is not vacuous

`R-AGE-007` requires the droppable set to be stated positively so the harness-facing narrowing is checkable rather than merely asserted. **What may still be lost at the harness-facing boundary is exactly: state the harness never learned before cancellation** — in-flight deltas of a turn cut short, progress of tools that never completed, and everything of turns that never began. This set is non-empty at every interruption, because a cancelled turn always has some in-flight work that history will never hold, by the same argument that makes `VL2-HAR-02` orphan synthesis necessary at all.

**Removing the loss path entirely is unavailable at any price.** A consumer that stops reading and never cancels would make delivery wait without bound — converting `VL2-HAR-05`'s bounded wind-down into an unbounded one, in direct contradiction of its own charter. The delivery obligation this section states is owed to a consumer honouring the contract: receiving, or cancelling. A consumer that does neither within the wind-down bound is in **abandonment**, the same documented, untestable-to-termination violation Layer 1 states and AI-40.3 restates; § 9 carries the Layer 2 package contract's equivalent clause.

### Choice — the capacity-measurement owner

`R-AGE-018` requires every deferred numeric value to name its owner and its closing evidence explicitly. **No numeric capacity is named at either Layer 2 boundary — and, unlike Layer 1's channel, none is coming later either.** § 5 already decided the mechanism: each lane absorbs its own consumer's lag **without a fixed bound chosen in advance**, a deliberate, permanent choice (the trilemma's property (c), bent), not a placeholder waiting to be filled in. What remains open is narrower: whether that deliberately unbounded posture survives real concurrent pressure, or needs a cap added on top of it after all. **Owner: `AG-21.2`**, slow-consumer pressure — doc 0003 names the whole of AG-21 "the AI-33/AI-34 of this layer, over the whole assembly," but AG-21's own charter is explicit that it inherits AI-34's testing ambit only: **"Out of scope: Performance targets — correctness under pressure only."** `AG-21.2` is therefore not positioned to publish a number, and this deferral does not ask it to. Closing evidence: `AG-21.2`'s own two scenarios passing under the combined-scenario matrix — *a stalled consumer loses nothing* (the lossless posture holds when a consumer actually stalls, not merely when the argument is read) and *cancellation unblocks a stalled stream within bounds* (the wind-down bound holds even after a lane has absorbed however far its consumer fell behind). Both passing closes the deferral, permanently confirming § 5's unbounded mechanism needs no numeric cap. **If either fails**, § 11 rule 1 already governs what follows: only this artifact decides a delivery property, so the failure is the trigger for a dated amendment here (§ 11 rule 4) — the point at which a bound would first be considered, not a measurement for `AG-21.2` to hand up unilaterally.

**Why no starting hypothesis is named here, unlike AI-02's 64.** AI-02's acceptance criterion required a starting figure because AI-34.1's charter demanded one to falsify against a real channel's fixed buffer parameter. `AG-21.2` has no such parameter to falsify — § 5's mechanism was chosen precisely to need none — and its own charter excludes performance targets outright. Naming a starting figure here, or asking `AG-21.2` to produce one, would manufacture a measurement neither this decision's mechanism nor `AG-21`'s charter has any use for.

### What this excludes

| Excluded | Why |
| --- | --- |
| Layer 1's loss rule, copied unchanged at both boundaries | Defeated by the worked case above: a committed side effect silently absent from the stream while history holds it; a session log cannot trust the stream after any interrupted run without an out-of-band cross-check |
| No loss path at all at the harness-facing boundary | A consumer that stops reading and never cancels makes delivery wait without bound — contradicts `VL2-HAR-05` directly. Full losslessness is unavailable at any price |
| A third, separately-named loss circumstance anywhere | `R-AGE-004`'s lossless claim is falsified by any third circumstance; none is named because none exists in this decision |
| A starting numeric capacity, named now | Would repeat AI-02's hypothesis pattern, but `AG-21.2` has no falsification machinery to test it against and no mandate to publish one — § 5's mechanism was chosen precisely so no such figure is needed |
| Reasoning from the superseded 64-event hypothesis | The hypothesis is historical; the measured, standing figure is `0`, and every argument above is built on that figure |

### Consequences

1. `VL2-HAR-05` bounded wind-down now carries a **delivery obligation**, not only a time bound: it must finish delivering everything history holds before or with run-end, within the same bound `AG-14.3` already documents.
2. A session log or any other history-adjacent consumer may trust the harness-facing stream after an interrupted run for every fact already committed to history — the property `R-AGE-012` states per outcome in § 6.
3. `AG-00`'s register carries the vocabulary that keeps `VL2-EVT-18` and `VL2-EVT-19` from being conflated by a downstream milestone; the two rows exist precisely because this decision required them nameable distinctly (`R-AGE-005`), and they are cited here by identifier rather than reproduced.

### Who inherits it

`AG-14` (the bounded wind-down obligation, and the harness-facing rule its own time bound must satisfy), `AG-04` (the run-end and turn-end terminal events this rule protects), `AG-21.2` (the capacity deferral, closed by its own two correctness scenarios rather than a published number, as stated above), `AG-13` (the run driver's steering path shares the harness-facing boundary's discipline), `AG-16` and `AG-18` (any milestone that could otherwise conflate the two loss postures the vocabulary now separates).

---

## 5. Decision 3 — the observer model

**Closing-checklist item 3.** What this decides: the mechanism that makes envelope invariant 3 (`VL2-EVT-14`) structurally true rather than conventionally hoped for, and the fork over how many consumers Layer 2 supports. What this excludes: any policy about which attached consumer matters more than another — that is Layer 3's, never a Layer 2 property.

### Decision

Invariant 3 is made structural by a **decoupling mechanism** (`VL2-SEAM-12` observer asynchrony): one canonical internal stream whose only receiver is the distribution step; per attached consumer (`VL2-SEAM-11`), a **lane** fed by its own forwarding activity that privately owns that consumer's receive-only carrier and applies the full send discipline of § 3 toward it. Each lane absorbs the gap between the canonical producer's progress and its own consumer's progress, bounded intrinsically by the **run's own extent** — memory-only, freed at run end, never persisted (`R-02`'s no-I/O closure). More than one attached consumer is supported per run; no consumer is privileged at the mechanism level.

### Why

#### The failure a convention permits

A convention-based design — the harness holds a list of attached observers and, per event, hands the event to each inline, under a documented "observers must not block" rule — looks sufficient until the first slow observer:

```
stalled session logger
  └─ blocks its receive ─► inline fan-out loop in the harness
       └─ blocks ─► the harness's own receive from the canonical stream
            └─ blocks (capacity 0) ─► the loop's send
                 └─ blocks ─► the Layer 1 transport read
                      ⇒ token delivery freezes for EVERY consumer,
                        including the primary frontend
```

One stalled cost meter freezes the screen. The party at fault wrote no code wrong — the observer merely fell behind, which every observer eventually does. That is what "conventional, not structural" looks like in practice, and it is exactly the trace `S-AGE-010` exists to forbid a merged decision from permitting.

#### The trilemma, named honestly

At any single hand-off point, three properties cannot all hold at once: **(a)** lossless delivery to every attached consumer, **(b)** a producer that never waits on any consumer, **(c)** a fixed buffer bound. Layer 1 chose (a) and (c) and gave up (b) — its backpressure **is** waiting. At the harness-facing boundary, invariant 3 forbids giving up (b) a second time — a producer waiting on a slow observer is exactly what invariant 3 exists to prevent — and `R-AGE-004`'s losslessness forbids giving up (a). So **(c) is what bends**, and this decision states so rather than hiding it: each lane absorbs the gap between the canonical producer's progress and its own consumer's progress, without a fixed bound chosen in advance. The absorption is nonetheless intrinsically bounded by the run's own extent — a run is finite, a lane drains as its consumer catches up, and everything a lane holds is freed at run end, never written anywhere (`R-02`). Whether this deliberately unbounded absorption needs a cap after all is exactly what `AG-21.2`'s two scenarios test under real pressure (§ 4) — a correctness question `AG-21.2` answers pass or fail, not a number it measures. This intrinsic bound is also what separates the chosen mechanism from the rejected pull-based replay record below: a replay record retains everything for every consumer for the run's full lifetime by design, unconditionally; a lane grows only while its own consumer lags, and only by the size of that lag.

#### The mechanism, and where the stalled trace terminates — the load-bearing trace

```
stalled consumer B
  └─ blocks its own receive ─► forwarding activity B, mid-send
       └─ lane B absorbs subsequent events            ◄── trace TERMINATES here
distribution step: offer into lane B does not wait on consumer B
  └─ its own receive from the canonical stream stays free
       └─ producer unaffected; consumer A unaffected
```

Every path from a stalled consumer's own receive ends at that consumer's own forwarding activity, and none reaches the canonical producer or any other attached consumer. This is the property `S-AGE-010` requires by inspection: given the mechanism above, and an attached consumer that stops receiving indefinitely and never cancels, every path traced from that consumer's stalled receive back toward the canonical producer terminates at the forwarding activity that privately owns that consumer's carrier. No consumer is privileged at the mechanism level — the run's primary frontend is one more lane, symmetric with a session logger or a cost meter; privilege among them is Layer 3 policy, never a Layer 2 property. `AG-20.2` inherits this as a mechanical test, and the lane is what makes "eventually reported typed" for a stalled observer *expressible at all* — the lag is observable at one owned place, rather than nowhere.

**Ownership within the lane is § 6's rule applied a second time.** Each forwarding activity closes only the carrier it privately owns, and never the canonical stream — "nothing else closes", applied recursively at the observer attachment point exactly as it is applied at the package boundary and at each of the three ownership scopes in § 6.

#### Rejected mechanisms, judged by what each makes impossible

`S-AGE-011` requires judgement by impossibility, not preference — "discourages" is convention, "makes impossible" is structure, and this closing-checklist item asked for structure:

| Mechanism | Makes impossible | Verdict |
| --- | --- | --- |
| Independent per-consumer carrier, own forwarding activity, own send discipline (chosen) | One slow consumer stalling **any** other consumer, including the run's primary frontend — no consumer is privileged at the mechanism level | **Chosen** |
| Bounded per-consumer buffer, drop on overflow | **Rejected against `R-AGE-004`.** Any consumer propagating genuine backpressure — a bounded buffer that drops instead of waiting is a second, unsanctioned loss circumstance the moment any consumer falls behind the producer, which `R-AGE-004`'s lossless claim forbids |
| Blocking synchronous multicast (the convention-based design above) | **Nothing.** This is exactly what invariant 3 forbids, and it is named here to show what "conventional, not structural" looks like when a reviewer needs the negative example |
| Pull-based in-memory replay record | Any coupling whatsoever between producer progress and observer progress — but at the cost of retaining a growing record for every consumer, for the run's entire lifetime, and leaving "how far behind may an observer fall" an unanswered bound. Rejected for this decision: the chosen mechanism subsumes its useful property (no coupling) without that cost |

#### The fork, picked openly with its rebuttal recorded

AG-01.1's own closing-checklist item 3 names a session logger and a cost meter attaching to Layer 2 directly — which presumes at least a second consumer exists. Doc 0001 § 2.2 draws exactly **one** upward emission arc from the harness, re-emitting enriched to a single Layer 3 event type, which then fans out to frontends inside Layer 3. Read literally, § 2.2 could be taken to mean Layer 2 needs only a single non-blocking hand-off per run, with all real fan-out happening entirely inside Layer 3 — which is unconstrained by `R-02`'s no-I/O rule and may use any mechanism it likes.

**Decision: Layer 2 supports more than one attached consumer per run.** The losing reading is written down rather than silently discarded, because a later reader must be able to tell a decision from an oversight without opening `explore.md`: the single-hand-off reading is coherent, and is rejected on two grounds. First, AG-01.1's own closing checklist asks how a *second* consumer attaches, which presumes one exists to be asked about — a decision that answered "there is no second consumer" would not have answered the question the checklist item poses. Second, the chosen mechanism **subsumes** the single-consumer case at no additional cost (one lane is simply what a single-consumer run looks like under this mechanism), while the reverse is not true — a design built only for one consumer would need to be rebuilt, not extended, the day a second one is asked for. Layer 3's further fan-out to N frontends remains a **second, additional stage** of the same pipeline, never an alternative to this one: a Layer 3 re-emission consuming one Layer 2 lane and fanning it out further is exactly as compatible with this decision as a Layer 3 that never fans out at all.

#### The rendezvous objection, answered on its merits

§ 4 states the conclusion this argument depends on; this is where it is proved, once, and cross-referenced from there rather than repeated. Layer 1's rejected zero-capacity argument warned, verbatim: a rendezvous has "zero tolerance for a consumer that pauses at all — and Layer 2's consumer pauses by design, to drive the permission protocol." Layer 2 is that consumer, and the objection deserves an answer on its own terms, not a citation of its own historical defeat.

**The answer:** a rendezvous is intolerant of a pause **on the receive step**, and indifferent to a pause on work performed **after** hand-off. The permission suspension `VL2-LOOP-07` drives does not sit on the receive step of any carrier — it sits in the loop's own per-call state. `AG-10.3` makes this a tested requirement, not a hope: while one call is suspended, sibling calls schedule, execute and emit, and message deltas already in flight keep flowing — achievable only if permission handling, tool scheduling and cost work all happen off the receive path. The party that genuinely pauses for the permission protocol — the frontend awaiting a human — pauses on its **own per-consumer carrier**, where this section's trace shows the pause backs up only its own lane, never the canonical producer and never a sibling consumer. The objection's premise — a consumer doing per-event work inline, on the receive step — describes a consumer Layer 2 is architecturally forbidden from being, by the same mechanism this section decides. One argument, used twice: the same off-the-receive-path fact answers the zero-capacity objection in § 4 **and** is what this section's decoupling mechanism makes structural.

### What this excludes

| Excluded | Why |
| --- | --- |
| Blocking synchronous multicast | Makes nothing impossible; the named example of conventional-not-structural, above |
| Bounded per-consumer buffer, drop on overflow | Rejected against `R-AGE-004`'s lossless claim, above |
| Pull-based replay record as the primary mechanism | Retains everything for every consumer for the run's lifetime and leaves the falling-behind bound unanswered; rejected for the reason stated above |
| Single hand-off, with all fan-out inside Layer 3 | A coherent reading of doc 0001 § 2.2, rejected on the two grounds recorded above; the losing reading and its rebuttal are both on this page |
| A privileged "primary" consumer at the mechanism level | Contradicts the trace above directly — the run's primary frontend is one more lane, no different in kind from a session logger |
| Answering the rendezvous objection by citing that it lost, without re-arguing it | Would leave the strongest recorded counter-argument unrebutted on the page, which `design.md`'s own reasoning rules forbid |

### Consequences

1. `AG-20.2`'s stalled-observer test inherits a mechanism to exercise mechanically, not a convention to hope holds.
2. "Eventually reported typed" for a stalled observer (deferred to `AG-20.2` by the proposal) becomes expressible: the lag is observable at the one place — the lane — that owns it.
3. A second, third, or Nth attached consumer costs one more lane, never a redesign of the canonical stream or of any other consumer's lane.
4. `AG-04`'s envelope invariant 3 (`VL2-EVT-14`) is satisfied structurally at the point this decision's mechanism attaches; `VL2-EVT-14`'s own register row cites this section by cross-reference rather than restating the mechanism.

### Who inherits it

`AG-20` (`AG-20.2`'s stalled-observer test, mechanically), `AG-04` (invariant 3's structural satisfaction), `AG-10` (the permission suspension's off-the-receive-path property, proved here), `AG-23.2` (any carrier view built over a lane inherits the same receive-only, never-owns discipline as § 3).

---

## 6. Decision 4 — close and ownership

**Closing-checklist item 4.** What this decides: who owns and who alone closes each of Layer 2's three nested scopes, and what a consumer may assume once it has received a terminal event. What this excludes: the contents of the terminal outcome payloads themselves (`AG-04`'s) and the shape of any bracket beyond stating who owns it.

### Decision

One rule — **the producer creates a scope, the producer alone closes it, exactly once, on every exit path, and nothing else closes it** — instantiated three times:

```
RUN SCOPE — owner and sole closer: the harness (VL2-COR-03, stateful)
┌────────────────────────────────────────────────────────────────┐
│ run-start                                              run-end │
│                                                    (typed:      │
│  TURN SCOPE — owner and sole closer: the loop        completed /│
│  (VL2-COR-02; stateless; re-instantiated per turn —  interrupted│
│   it CANNOT own the run bracket, which is why the      / failed)│
│   owners differ)                                                │
│  ┌──────────────────┐   ┌──────────────────┐                    │
│  │ turn-start…end   │   │ turn-start…end   │                    │
│  │ (typed: finished │   │ turn-end on EVERY│                    │
│  │  / aborted)      │   │ exit path)       │                    │
│  └──────────────────┘   └──────────────────┘                    │
│                                                                 │
│  DELEGATED-RUN SCOPE — the child harness (VL2-COR-14's harness) │
│  owns and closes its OWN run-scoped stream; the parent owns     │
│  ONLY the subagent bracket on its OWN stream                    │
│  ┌──────────────────────────────────────────┐                   │
│  │ subagent-started            (parent's)   │                   │
│  │   child run-start … child run-end        │                   │
│  │   (child's stream — fully closed FIRST,  │                   │
│  │    leaf-first per AG-19.2)               │                   │
│  │ subagent-ended              (parent's)   │                   │
│  └──────────────────────────────────────────┘                   │
└────────────────────────────────────────────────────────────────┘
```

**Per-turn scope.** Owner and sole closer: the loop. The loop is the sole producer of that turn's message and tool events, bracketing them in turn-start and turn-end, and it emits turn-end on **every** exit path — normal completion, a typed failure, or cancellation — never only the happy one. Turn-end distinguishes model-finished from turn-aborted by typed outcome (`VL2-EVT-11`).

**Per-run scope.** Owner and sole closer: the harness — a **different** owner from the turn scope, and necessarily so: the loop is stateless and re-instantiated fresh per turn (`VL2-COR-02`), and cannot hold a boundary that spans many turns. One run-start precedes everything on the run-scoped stream; one run-end follows everything, carrying a typed outcome (`VL2-EVT-10`) of completed, interrupted, or failed. Nothing follows the terminal.

**Per-delegated-run scope.** The child harness owns and closes its own run-scoped stream in full, exactly as any harness does. The **parent** harness separately owns only the subagent bracket on its **own** stream — subagent-started and subagent-ended (`VL2-EVT-08`, `VL2-COR-15`) — re-emitting the child's already-closed events, parent-identified per envelope invariant 2 (`VL2-EVT-13`). Ownership here is never shared: it is strictly nested and sequential. Under leaf-first cancellation (`AG-19.2`), the child's stream fully closes — its own run-end, emitted by the child harness, on the child's own stream — **before** the parent emits subagent-ended on its own stream. The parent never closes the child's stream; the child never closes the parent's. A child scope closing before its parent's representation of it closes is not an edge case this rule merely tolerates — it is the **only** order the rule permits.

**After the terminal**, per outcome (`R-AGE-012`): on **completed**, everything received is the complete, ordered story. On **interrupted** or **failed**, the received prefix is trustworthy in exactly § 4's sense: nothing already committed to `VL2-HAR-01` history is missing from it; what had not yet happened by the time of cancellation is truncated, not silently substituted. Layer 2 has **no** "sometimes no terminal at all" case at run scope — unlike Layer 1's own sanctioned loss path, which can close a stream bare. `VL2-EVT-19`'s harness-facing rule is *why*: run-end sits inside the protected set § 4 states, so a run always ends with a typed outcome even when the events describing how it got there are truncated.

### Why

**"Close a scope exactly once" is a claim about every execution, and a reader sees one text.** Stated procedurally it is unverifiable by reading, which is why it is stated structurally: a second closing site, several closers, or a consumer-side close are each a *shape*, visible in a diff, rather than a behavior a reviewer must trust. Three misreadings each produce a defect the structural form forecloses: "closed once per successful run" grows separate closing sites on the error and cancellation paths; "closed once, by whoever finishes last" turns close into a coordination problem no contract mentions; "closed once, and the consumer may also close it when done early" sends on an already-closed carrier, which § 3's receive-only carrier already forbids at the type level for the ordinary case.

**The turn-scope and run-scope closers differ because the loop cannot be the run's owner.** The loop is stateless and re-instantiated per turn (`VL2-COR-02`, `VL2-COR-17` loop statelessness); it has no memory of a run boundary spanning turns it was never called for. Only the harness — the stateful half of the runtime, the party that holds `VL2-HAR-01` history across the whole run — can own a bracket that outlives any single turn. This is not a convenience; it is the same reason a single loop invocation never issues more than one provider call while a turn may retry across several (`VL2-COR-08`) — the two scopes answer to fundamentally different lifetimes, and conflating their owners would require the loop to remember something its own charter (`VL2-COR-17`) forbids it to remember.

**Leaf-first delegation close is the recursion of "nothing else closes", not an exception to it.** If the parent closed its own subagent bracket before the child's stream had finished, the parent's re-emission of the child's events would be re-emitting events the child had not yet finished producing — which cannot be made consistent with envelope invariant 2's parent-identification (`VL2-EVT-13`) applying to a *complete* child sequence. `AG-19.2`'s own scenario states the order directly: the tree cancels leaf-first, the child's orphans synthesize and its run-end emits, and only then does the parent's wind-down complete, with both transcripts closing valid.

### What this excludes

| Excluded | Why |
| --- | --- |
| A protocol by which a consumer signals "I am done" by closing a carrier itself | A consumer that wants to stop early cancels (§ 3). That is the mechanism, and it is the only one |
| A close performed by a carrier view, a wrapper, or a test helper | Never owns what it views (§ 3); closing it would teach an attached consumer a physics the contract does not have |
| Several producers coordinating a shared close | The coordination, if any exists below the boundary, is private to whatever produces into a single owner's scope — it is never visible as more than one closing site at the scope boundary itself |
| The parent closing the child's stream, or the reverse | Directly forbidden by the delegated-scope rule; each harness closes only its own run-scoped stream |
| A "sometimes no terminal at all" case at run scope | Foreclosed by `VL2-EVT-19`'s harness-facing rule protecting run-end specifically — unlike Layer 1, which does admit a bare close on its own sanctioned loss path |
| A single undifferentiated "the stream is the complete story" claim for every outcome | `R-AGE-012` requires the assumption stated per outcome, because it differs: completed is unconditionally complete; interrupted and failed are complete-of-what-was-committed, not complete-of-everything |

### Consequences

1. `AG-04.2`'s run and turn bracket discipline — "one run-start, one run-end, turns nested and non-overlapping, nothing after the terminal" — becomes a test of a stated structure with a stated owner, rather than an aspiration with no named closer.
2. `AG-19`'s leaf-first close order is not something AG-19 invents; it is this decision's own delegated-scope rule, applied.
3. A consumer's trust in a received prefix after interruption or failure is now precisely bounded by § 4's harness-facing rule, not left to intuition — the two decisions compose exactly the way § 4's "history is the watershed" argument requires.

### Who inherits it

`AG-04` (the bracket owners for run-start/run-end and turn-start/turn-end), `AG-13` (the run driver is the harness's own implementation of the run-scope closer), `AG-14` (the interrupted outcome and the bounded-wind-down obligation this rule places on it), `AG-19` (leaf-first close order, stated here rather than left for AG-19 to invent).

---

## 7. Decision 5 — the upward path

**Closing-checklist item 5.** What this decides: the one surface through which a message from outside a live run re-enters it, and what happens when that message loses the race against the run ending. What this excludes: the shape or storage of the lookup a decision uses to find its suspension (`AG-10`'s and `AG-13`'s), and pause-resumption, which is carved out entirely.

### Decision

The harness (`VL2-COR-03`) is the **one** inbound surface (`VL2-COR-09` upward path) for messages entering a live run, carrying three typed payload kinds: a permission decision, a `VL2-COR-12` steering message, and an interrupt. This is structurally **one** surface, never three parallel paths that happen to look alike. A call-identity-to-suspension lookup (`VL2-COR-13`) exists, is harness-owned, and lives for the suspension's lifetime — its shape, structure and storage are not fixed here. An upward message addressed to a call that no longer holds a live suspension within a live run, or to a run identity that has fully ended, receives a **typed rejection**, never a silent drop, at the matching granularity.

```
frontend            parent harness              in-flight loop invocation
   │                (the one stable,                     │
   │                 addressable surface)                │
   │ decision /      │                                   │
   │ steering /      │                                   │
   │ interrupt       │                                   │
   │ (typed payload) │                                   │
   │────────────────►│                                   │
   │                 │ resolve identity, two levels:      │
   │                 │  1. run identity  — live?          │
   │                 │  2. call identity — suspension      │
   │                 │     still held? (harness-owned      │
   │                 │     lookup; shape NOT fixed here)   │
   │                 │                                    │
   │                 ├── both live ──────────────────────►│ resume that
   │                 │                                    │ one call
   │   typed         │                                    │
   │◄─ rejection ────┤ call no longer suspended,          │
   │   (call         │ run still live                     │
   │    granularity) │                                    │
   │   typed         │                                    │
   │◄─ rejection ────┤ run already ended                  │
   │   (run          │                                    │
   │    granularity) │                                    │
```

**Reconciling the two source levels.** Doc 0001 § 2.2 draws the only upward arrow into the harness: "there is an upward arrow from the frontend, and it is the only one … It is not the frontend driving the loop." § 2.3 narrates the suspension entirely at the loop level: "the turn SUSPENDS here. Nothing blocks. Other calls proceed," answered by the frontend directly in the diagram. **Both are right at their own level, and no source ties them together before this decision.** The harness is the receiving surface because it is the stable, addressable thing a frontend can hold across a whole run; the loop is stateless and re-instantiated per turn (`VL2-COR-02`), so it cannot be what a frontend holds a reference to across turn boundaries. § 2.3's loop-level drawing is the **destination** the harness's routing reaches, never the entry point a frontend calls.

**Two carve-outs**, so the rejection machinery is not over-applied: an interrupt arriving **during** bounded wind-down (`VL2-HAR-05`), before run-end has been emitted, is silently idempotent per `AG-14.1`'s own scenario — the run is already doing what the interrupt asks, and a second signal changes nothing and panics nothing. The typed-rejection machinery begins only once run-end has actually been emitted. And pause-resumption is model-initiated — a finish reason `VL2-LOOP-08` dispatches — and harness-internal: it is **not** an instance of the upward path at all, and must never be routed through the identity-resolution or typed-rejection machinery meant for frontend-originated messages. `AG-13.3` consumes pause-resumption on its own terms.

### Why

**The race that makes rejection necessary, not defensive.** The downward stream and the upward surface are asynchronous by construction — that is the entire point of § 5's decoupling. So this interleaving is inherent to the design, not an exotic edge case a careful implementation might avoid:

```
t0  decision-required event delivered to the frontend (call identity X)
t1  human deliberates …           t1' run is interrupted; wind-down
                                      resolves suspension X; run-end emitted
t2  human answers; decision for X arrives at the surface — X is gone
```

The frontend cannot know at t2 what happened at t1' — its own knowledge is only ever a prefix of the stream it has received so far. A silent drop at t2 would leave the frontend awaiting an effect that will never come, with no way to distinguish "my answer was applied" from "my answer was lost". The typed rejection at the matching granularity — call identity within a still-live run, run identity once the run has fully ended — is the only signal that tells the answerer its answer lost the race. This is one rule at two granularities, generalised from `AG-10.1`'s own call-level precedent ("a stray decision … rejected as a typed protocol error, never a silent no-op"), decided once here so `AG-10` and `AG-13` do not each reinvent it independently and risk drifting apart.

**Recursion under delegation.** A child harness has its own inbound surface, in principle — but what a suspended call inside a delegated run would need to ask about is asked on the **parent's** stream, per `VL2-SEAM-10` derived permission scope: one place a human watches, never a second permission surface. The upward path therefore recurses rather than terminating at the first harness reached: the frontend answers through the **parent's** surface, and the parent's own routing must reach into the nested child's own suspension lookup to resolve it. The obligation to route that far is stated here; the mechanism by which the parent's routing reaches the child is `AG-19`'s.

### What this excludes

| Excluded | Why |
| --- | --- |
| Three parallel upward paths, one per payload kind | `R-09` is one decided path; three lookalike paths would triple the identity-resolution, rejection and recursion machinery and invite drift between them |
| A silent drop for a stray or late upward message | Defeated by the race above: the answerer cannot distinguish "applied" from "lost". `AG-10.1`'s typed-protocol-error precedent is generalised instead of being reinvented per milestone |
| Fixing the call-identity lookup's shape, structure, or storage | Belongs to `AG-10` and `AG-13`, the milestones that build the permission protocol and the run driver — this decision settles the container, not the contents, mirroring AI-02's own standing rule |
| Routing pause-resumption through the identity or rejection machinery | Pause-resumption is model-initiated and harness-internal, not frontend-originated; conflating it with the upward path would apply typed-rejection semantics to a case that was never addressed to anyone |
| A typed rejection for an interrupt arriving during bounded wind-down | Carved out explicitly: `AG-14.1`'s own idempotence scenario requires the redundant signal to be silently tolerated, not rejected, while the run is already doing what was asked |
| Leaving the § 2.2/§ 2.3 level mismatch implicit | `S-AGE-017` requires the reconciliation written down, with its reason, rather than assumed by whichever milestone notices the mismatch first |

### Consequences

1. `AG-10`'s permission protocol receives a named, harness-owned surface and a confirmed lookup to build against, rather than inventing where a decision arrives.
2. `AG-13`'s steering input travels the identical surface, with the identical ended-run rejection, rather than a parallel mechanism that happens to resemble it.
3. `AG-14`'s two interrupt states — during wind-down versus after run-end — are distinguished on this page, so `AG-14` implements a stated distinction rather than discovering the need for one.
4. `AG-19`'s cross-harness routing has a stated destination (the child's own suspension lookup) and a stated origin (the parent's surface), rather than an unscoped design question.

### Who inherits it

`R-AGE-017` requires this decision to close with one table stating, in each blocked milestone's own terms, what it takes from this whole decision — not only from decision 5. This is that closing inheritance table. It is presented again, organised by milestone, at § 10; the two are the same information and do not diverge.

| Milestone | What it takes from this decision |
| --- | --- |
| **`AG-04`** (envelope and ordering) | The carrier the envelope travels on (§ 3); the three ownership scopes that give run-start/run-end and turn-start/turn-end their owners (§ 6); "nothing follows the terminal" as an owned obligation, not only a validator rule |
| **`AG-10`** (permission protocol) | The harness-level surface and the call-identity lookup confirmed existing and harness-owned (this section); the typed-protocol-error discipline for stray or late decisions; § 5's assurance that a suspension never stalls delivery |
| **`AG-13`** (multi-turn run driver) | The same surface for steering input; the ended-run typed rejection generalised from `AG-10.1`'s call-level precedent; the pause-resumption carve-out, so it is never routed as an upward message |
| **`AG-14`** (cancellation tree) | Interrupt as one of the three payload kinds; the distinction between an interrupt arriving during wind-down (idempotent, silently tolerated) and after the run has ended (typed rejection); the harness-facing delivery obligation § 4 places on bounded wind-down |
| **`AG-19`** (delegation and re-entrancy) | Strictly nested, never-shared ownership and leaf-first close ordering (§ 6); the upward-path recursion — the frontend answers on the parent's surface, and the parent's routing must reach into the child's own suspension lookup |
| **`AG-20`** (hook taxonomy) | The decoupling mechanism (§ 5) that makes `AG-20.2`'s stalled-observer test pass structurally, and that makes "eventually reported typed" expressible for a stalled observer at all |

**None of the six may invent its own channel, its own loss rule, or its own way back into a live run.** A downstream milestone that finds itself deciding one of the properties this decision fixes is proposing an amendment to this artifact, not exercising an ordinary judgement call — restated once more, at the document level, in § 11 rule 6.

---

## 8. The delivery topology

Two internal boundaries, one attachment point. Every section above happens on this one picture.

```
                              LAYER 2
  ┌───────────────────────────────────────────────────────────────┐
  │  Layer 1 stream (frozen contract; capacity 0, measured)       │
  │        │                                                      │
  │        ▼              LOOP-INTERNAL BOUNDARY (VL2-EVT-18)     │
  │  ┌──────────┐   per-turn stream: turn-start … turn-end        │
  │  │   loop   │ ────────────────────────────────► ┌──────────┐  │
  │  │(stateless│   Layer 1's send discipline and   │ harness  │  │
  │  │ per turn)│   loss rule, unchanged             │(stateful)│  │
  │  └──────────┘                                    │ history · │  │
  │                                                  │ brackets │  │
  │                                                  └────┬─────┘  │
  │                     canonical run-scoped stream       │        │
  │                (exactly one receiver: the             ▼        │
  │                 distribution step)              ┌──────────┐   │
  │                                                 │ distrib. │   │
  │                                                 │  step    │   │
  │                    OBSERVER ATTACHMENT POINT    └─┬──────┬─┘   │
  │                                              lane A│      │lane B
  │                                          ┌─────────▼─┐  ┌─▼─────────┐
  │                                          │ forwarding│  │ forwarding│
  │                                          │ activity A│  │ activity B│
  └──────────────────────────────────────────┴─────┬─────┴──┴─────┬─────┘
                          HARNESS-FACING BOUNDARY  │              │
                          (VL2-EVT-19)             │              │
                          per-consumer receive-only│              │
                          carriers, one each       ▼              ▼
                                             primary         session logger,
                                             consumer        cost meter, …
```

Roles: the **loop** (`VL2-COR-02`) is the sole producer and closer of each per-turn stream. The **harness** (`VL2-COR-03`) is the sole consumer of turn streams, the owner of `VL2-HAR-01` history and the run brackets, and the sole producer onto the canonical run-scoped stream. The **distribution step** is the canonical stream's only receiver; per attached consumer it feeds one **lane**, whose **forwarding activity** privately owns that consumer's receive-only carrier and applies the full send discipline of § 3 toward it.

**Where the four envelope invariants bind** (the invariants are `VL2-EVT-12` … `VL2-EVT-15`'s; this table places them on the topology, it does not restate them):

| Invariant | Binds at | Delivery's obligation |
| --- | --- | --- |
| 1 — indexed deltas (`VL2-EVT-12`) | The loop, where deltas originate | Delivery forwards payloads verbatim; no stage rewrites or accumulates on behalf of a consumer |
| 2 — explicit nesting (`VL2-EVT-13`) | The parent harness's re-emission of a child's events onto its own canonical stream | The re-emission point is the only place a parent identifier can be attached; § 6's delegated scope |
| 3 — asynchronous observers (`VL2-EVT-14`) | The observer attachment point | § 5's mechanism, in full |
| 4 — typed errors and outcomes (`VL2-EVT-15`) | The two terminal emitters: loop (turn-end outcome), harness (run-end outcome) | § 6's terminal discipline; delivery guarantees the terminal's *arrival* (§ 4); `AG-04` defines its contents |

---

## 9. What the package contract must state

The Layer 2 equivalent of AI-40.3's Layer 1 restatement: these are prose obligations, in the form they must survive into whatever documents the shipped Layer 2 boundary, not spellings.

1. **A receive-only carrier is the boundary.** A consumer receives from it; it never sends on it and never closes it. (§ 3)
2. **The caller supplies a cancellable signal on the call that attaches a carrier.** Every send waits on both the destination and cancellation, the terminal send included. (§ 3)
3. **A consumer ends its carrier in exactly one of two ways: it drains to close, or it cancels.** (§ 3)
4. **Abandoning a carrier without cancelling it is a contract violation, not a supported mode.** It is stated rather than enforced, because no test proves a concurrent task never exits. A consumer that will not drain must cancel. This is the Layer 2 restatement of the same untestable obligation AI-40.3 carries at the Layer 1 freeze — the residual `S-AGE-002` requires priced, and the price is paid here, in the only currency available: a documented statement, not a test. (§ 3)
5. **Two internal boundaries carry different, named loss postures.** `VL2-EVT-18` at the loop-internal boundary; `VL2-EVT-19`, strictly narrower, at the harness-facing boundary. Neither is the other, and no third loss circumstance is sanctioned. (§ 4)
6. **The harness-facing boundary never discards an event describing a fact already committed to history.** Bounded wind-down finishes delivering every such event before or with run-end. (§ 4)
7. **A stalled attached consumer cannot stall any other consumer or the canonical producer.** Every path from a stalled consumer's own receive terminates at that consumer's own forwarding activity. (§ 5)
8. **Three nested scopes — turn, run, delegated run — each have exactly one owner and exactly one closer.** Nothing else closes a scope it does not own; delegation closes leaf-first. (§ 6)
9. **Exactly one run-start and one run-end bracket a run; run-end always carries a typed outcome.** Unlike Layer 1, there is no "sometimes no terminal at all" case at run scope. (§ 6)
10. **One harness-level surface receives three typed payload kinds.** A message addressed to a call or a run that can no longer receive it is a typed rejection, never a silent drop, except for the two named carve-outs. (§ 7)

---

## 10. What each blocked milestone inherits

The same six milestones as § 7's closing table, restated here per milestone rather than per decision, so a reviewer checking `AG-04`, `AG-10`, `AG-13`, `AG-14`, `AG-19` or `AG-20`'s own acceptance criterion needs only this one section. **This table does not diverge from any per-decision "Who inherits it" subsection above** — it consolidates them.

### `AG-04` — the event envelope

- The carrier it travels on (§ 3) and the caller-owned liveness rule that governs every send on it.
- The three ownership scopes and their sole owners and closers (§ 6), including that the run-scope owner differs from the turn-scope owner and why.
- "Nothing follows the terminal" and the per-outcome consumer assumption (§ 6), which `AG-04.2`'s own validator now tests against a stated owner rather than an aspiration.
- **Not inherited, because it is `AG-04`'s:** event kinds, payloads, ordering invariants, the envelope's contents.

### `AG-10` — the permission protocol

- The harness-level inbound surface and the confirmed existence of a harness-owned call-identity lookup (§ 7) — its shape remains `AG-10`'s to build.
- The typed-protocol-error discipline for a stray or already-decided call identity, generalised once here rather than reinvented (§ 7).
- The assurance that a live suspension never stalls delivery to any consumer, proved structurally rather than assumed (§ 5).
- **Not inherited:** the lookup's shape, the four-outcome vocabulary's own semantics, and the policy content behind any decision — all `AG-10`'s or Layer 3's.

### `AG-13` — the multi-turn run driver

- The identical harness-level surface for steering input, and the identical ended-run typed rejection, generalised from `AG-10.1`'s precedent rather than reinvented in parallel (§ 7).
- The pause-resumption carve-out — never routed through the upward path's identity or rejection machinery (§ 7).
- The run-scope owner and closer (the harness itself), which `AG-13`'s own run driver implements directly (§ 6).

### `AG-14` — the cancellation tree

- Interrupt as one of the upward path's three typed payload kinds, and the stated distinction between an interrupt arriving during bounded wind-down (silently idempotent) and one arriving after run-end (typed rejection) (§ 7).
- The harness-facing delivery obligation bounded wind-down must satisfy: everything already committed to history delivered before or with run-end, within the same time bound `AG-14.3` documents (§ 4).
- The loop-internal loss posture that governs what may be lost when the harness itself cancels a turn in flight (§ 4).

### `AG-19` — delegation and re-entrancy

- Strictly nested, never-shared ownership across the three scopes, and leaf-first close order as this decision's own rule, not `AG-19`'s invention (§ 6).
- The upward-path recursion: the frontend answers through the parent's surface, and the parent's routing must reach into the nested child's own suspension lookup (§ 7).
- The re-emission point as the only place a parent identifier attaches, per envelope invariant 2 (§ 8's invariant-binding table).

### `AG-20` — the hook taxonomy

- The decoupling mechanism in full — one canonical stream, per-consumer lanes, run-extent-bounded absorption — that `AG-20.2`'s stalled-observer test exercises mechanically (§ 5).
- What "eventually reported typed" for a stalled observer can mean at all: the lag observable at the lane that owns it (§ 5).

---

## 11. Standing rules this decision establishes

1. **The container is settled; the contents are not.** This artifact decides how agent events travel — carrier, backpressure posture, observer decoupling, ownership, and the upward path. What an event *says* is `AG-04`'s, `AG-05`'s and `AG-06`'s. A downstream milestone that finds itself deciding a delivery property is proposing an amendment to this artifact, not exercising a judgement call.
2. **Exactly two loss circumstances exist, named and distinct.** `VL2-EVT-18` at the loop-internal boundary, `VL2-EVT-19` at the harness-facing boundary. Any other loss is a defect, not a variant, and conflating the two postures is the specific mistake AG-00's register was asked to make unnameable.
3. **An untestable obligation is marked as such and lives in the package contract.** Abandonment without cancellation is never given a test that proves something weaker, because the weaker property would then replace the stronger claim in a reader's mind. § 9 states it in the form that must survive.
4. **A reopened question is an amendment, in the same pull request.** If a later milestone's implementation disproves a decision here, the finding is recorded as a dated amendment blockquote to this artifact — or, where the disproved claim lives in `AG-00`'s register rather than here, to that register under its own `R-AGV-013` procedure — landed in the pull request that resumes work. Superseded text is struck through and left visible, never deleted, so citations from merged milestones keep resolving.
5. **The capacity is a posture, closed by `AG-21.2`'s own two correctness scenarios, not by a published number.** No starting figure is named at either Layer 2 boundary, and none is expected from `AG-21.2` — its charter excludes performance targets. Citing a number here is a misreading of § 4; a failure of either of `AG-21.2`'s scenarios triggers a dated amendment under rule 4, never a measurement to cite in its place.
6. **No downstream milestone may invent its own channel, its own loss rule, or its own way back into a live run.** Restated here, once, at the document level, having already been stated once in § 7 attached to the closing inheritance table: `AG-04`, `AG-10`, `AG-13`, `AG-14`, `AG-19` and `AG-20` each consume this decision's surface; none of them owns the properties this decision fixes.

---

## 12. Closing-checklist verification

AG-01.1's five items, each walked against this artifact, with its evidence location and its rationale-plus-*what-this-excludes* pair.

| # | Closing-checklist item | Where answered | Evidence | Status |
| --- | --- | --- | --- | --- |
| 1 | **Carrier at the boundary**, argued at Layer 2, with the same caller-owns-the-context liveness rule as Layer 1 | § 3 | A receive-only carrier (`VL2-EVT-17`), on four grounds each tested against a named Layer 2 source (`AG-10.3` for Ground 1, `AG-04.2` for Ground 3) rather than accepted by symmetry; the iterator case stated at full strength first; both alternatives priced; the abandonment concession stated as a residual cost; the caller-owned liveness rule's four clauses mapped unweakened; the carrier-view question deferred by name to `AG-23.2`. *Excludes*: an iteration-shaped carrier, both carriers offered together, a send-capable handle, a push-callback, and deciding the carrier-view question here rather than at `AG-23.2` | **answered** |
| 2 | **Backpressure posture**: lossless, with the two boundaries stated and the same-or-narrower loss path named | § 4 | The lossless claim with exactly two named exceptions; the loop-internal boundary inherited unchanged (`VL2-EVT-18`) with its safety argued from who bears the loss; the harness-facing boundary strictly narrower (`VL2-EVT-19`) with the cancelled-mid-tool-result worked case and the history-as-watershed composition argument; the measured capacity `0` cited, not the superseded hypothesis; the droppable set stated positively; the zero-capacity objection answered by the receive-step distinction, with the full proof cross-referenced to § 5's *the rendezvous objection, answered on its merits*; the capacity deferred by name to `AG-21.2` with its closing evidence stated. *Excludes*: a third loss circumstance, a starting numeric capacity, and full losslessness at the harness-facing boundary | **answered** |
| 3 | **Observer model**: envelope invariant 3 made structural by mechanism, not convention | § 5 | The convention-based failure traced end to end to a frozen screen; the trilemma named honestly with the deliberately bent property identified; the run-extent-bounded lane-absorption mechanism stated, with the terminating trace proving every path from a stalled consumer ends at its own forwarding activity; the mechanism table judged by impossibility, including the blocking-multicast row recording "makes nothing impossible"; the multi-observer-versus-single-hand-off fork decided with the losing reading and its two rejection grounds recorded; the rendezvous objection answered on its merits. *Excludes*: a privileged consumer at the mechanism level, and any of the three rejected mechanisms | **answered** |
| 4 | **Close and ownership rules**, mirroring Layer 1's exactly-one-terminal discipline at the agent level | § 6 | Three nested scopes, each with its sole owner and sole closer, with the turn/run owner split argued from the loop's own statelessness; the delegated case's leaf-first order argued as the recursion of "nothing else closes" rather than an exception to it; the exactly-one-terminal discipline stated, including that Layer 2, unlike Layer 1, has no "sometimes no terminal at all" case at run scope; the per-outcome consumer assumption stated separately for each of the three typed outcomes. *Excludes*: a shared closer, a parent or child closing the other's stream, and a single undifferentiated post-terminal assumption | **answered** |
| 5 | **The upward path**: the surface, the lookup, and the ended-run rejection | § 7 | One harness-level surface for three typed payload kinds; the § 2.2/§ 2.3 level mismatch reconciled in writing with its reason; the lookup's existence and ownership stated without fixing its shape; typed rejection required at both granularities, with the race that makes it necessary traced explicitly; the two carve-outs (wind-down idempotence, pause-resumption) stated so the machinery is not over-applied; the delegation recursion stated as an obligation rather than left for `AG-19` to invent; the closing inheritance table naming all six blocked milestones, with the no-invented-channel rule stated once there and restated once at the document level (§ 11 rule 6). *Excludes*: three parallel paths, a silent drop, and fixing the lookup's contents | **answered** |

**Register amendment.** No amendment is authored by this change's own artifacts. The two rows this decision's backpressure posture required nameable distinctly — `VL2-EVT-18` **loop-internal turn-scoped loss** and `VL2-EVT-19` **harness-facing history-guarded truncation** — were appended to [AG-00's register](../cachicamas-agent-contract-vocabulary/specs/agent-contract-vocabulary/spec.md) by AG-00 itself, at this node's own recorded request, under the register's own `R-AGV-013` amendment procedure, with a dated blockquote at the head of the register's `VL2-EVT` category stating what was appended, that AG-01 requested it, and why: so that this change's own amendment duty is satisfied by citation rather than by a second, competing append into a register this change does not own. Both rows are cited above by identifier; neither is defined here, and this change's diff does not touch the register file. This satisfies `AG-00`'s own `R-AGV-013`, `S-AGV-037` and `S-AGV-040` for exactly these two rows, and this change's own `R-AGE-005`.

**Node status.** AG-01.1 closes on merge of this artifact. Per doc 0003's node grammar, a `[decision]` leaf produces no production code and closes when "the decision answers every question in the closing checklist and is closed before AG-04 starts." No `make test` gate applies; nothing under `backend/` is touched by this change.

**Milestone acceptance, restated from doc 0003 and checked:** *"The decision answers every question in the closing checklist and is closed before AG-04 starts."* The table above answers all five items with evidence per item. This change merges in the same pull request as `AG-00` (`cachicamas-agent-contract-vocabulary`), whose register this decision cites throughout and whose two amended rows this decision depends on; the two changes revert together or not at all, per this document's own rollback statement below.

**Rollout.** A decision artifact — merged, it becomes the settled input to `AG-04` and, through it, waves 1 through 6. Nothing executes; adoption *is* citation by later milestones.

**Rollback.** `git revert` of the single commit, complete by construction: a new directory, nothing imports it. **Partial rollback is not meaningful and must not be attempted** — the five decisions are load-bearing on each other: the carrier (§ 3) gives the send discipline its object; the observer mechanism (§ 5) presumes the carrier; the harness-facing rule (§ 4) presumes the harness owns the run scope (§ 6); the upward path (§ 7) presumes run-end delivery, which § 4's harness-facing rule is what protects. A rejected decision rejects the whole change. Because `AG-00` lands in the same pull request: reverting this change alone leaves `AG-00` intact and self-consistent, since `AG-00` defines terms and depends on nothing here; reverting `AG-00` alone would strand every `VL2-*` citation this artifact makes, so the two revert together or not at all. Post-merge reversal, once `AG-04` has defined the envelope and `AG-10`, `AG-13` and `AG-14` have consumed the upward path, is priced by doc 0001 § 3.2 at roughly three times the cost of reversal now — which is why this decision is scheduled in wave 0, before any of them starts.

**Unblocked by this decision:** `AG-02` (`cachicamas-agent-v1-scope`) and `AG-04` (the event envelope) directly — and, through `AG-04`, `AG-10`, `AG-13`, `AG-14`, `AG-19` and `AG-20`, none of which may invent its own channel, its own loss rule, or its own way back into a live run.
