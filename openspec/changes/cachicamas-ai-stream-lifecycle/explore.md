# Explore — Layer 1 stream lifecycle, ownership and carrier

> **Change**: `cachicamas-ai-stream-lifecycle`
> **Milestone**: AI-02 — Decide stream lifecycle, ownership, and the carrier
> **Node**: AI-02.1 — Lifecycle, ownership and carrier `[decision]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Target module**: `backend/agent/` — **no code is written by this change**
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0003 — Layer 2 task graph](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Binding input**: [AI-01's vocabulary](../cachicamas-ai-contract-vocabulary/decision.md) — `V-STR-01` … `V-STR-09` and `V-FAIL-11` / `V-FAIL-12` are the exact concepts this milestone decides
> **Predecessor**: AI-01 (`cachicamas-ai-contract-vocabulary`)
> **Authoring constraint**: doc 0002's authoring constraint binds every artifact of this change. **No Go type name, field name, method name, or package identifier appears anywhere.** Language and standard-library shapes are named descriptively, not as identifiers — "the single-value iterator function shape", never a spelled type.

---

## 1. What this milestone is, in one paragraph

AI-02 settles the physics of a Layer 1 stream before a single event type exists: what carries events across the package boundary, who creates and destroys the stream, what cancellation obliges of both parties, how much slack sits between producer and consumer, and how a failure reaches a caller on each of the two paths a failure can take. It writes no code. It constrains every stream milestone that follows, which is exactly why it is scheduled in wave 0 rather than discovered in wave 2. AI-01 already named the nouns; this milestone assigns them behavior.

## 2. Why five questions are one decision and not five

They look separable. They are not, and the coupling runs in both directions:

- The **carrier** determines whether a producer goroutine exists at all. If it does, ownership needs a closing rule; if it does not, the ownership question changes shape entirely.
- **Ownership** determines what cancellation must guarantee. "Closes exactly once on every path" is only checkable once "every path" is enumerated, and cancellation is one of the three.
- **Cancellation** plus a **bounded buffer** is what produces the one loss path the contract sanctions. Neither alone produces it: an unbounded buffer never saturates, and a stream that is never cancelled never drops.
- The **buffer** is only meaningful if the carrier has a queue. A carrier without one turns question 4 into a non-question — and, less obviously, turns AI-34 into a milestone with nothing to measure.
- **Failure delivery** hangs off the moment the carrier is handed to the caller. That moment is defined by the carrier choice, so the pre-stream/mid-stream boundary cannot be drawn before question 1 is answered.

Deciding them in five separate places is how a plan ends up with a mandatory terminal event that no adapter can construct. That is defect **C4** in doc 0001 § 3.1, and its cause is recorded there as "the interface shipped before the taxonomy it depended on". Splitting a coupled decision across milestones is the same failure with a different subject.

## 3. What the record already says, and which part of it expired

Three documents carry a position on the carrier, and they do not carry the same one any more.

| Source | What it recorded | Status today |
| --- | --- | --- |
| doc 0001 § 3.2 (settled stream carrier row) | Decision only; documented default is to keep channels; the canonical stranded-producer hazard is already closed by selecting on cancellation for every send; closing it before an adapter exists is worth doing because reopening afterwards is far more expensive | The reasoning stands; the milestone number (retired AI-47) is stale and maps to AI-02 |
| doc 0001 § 7 **G13** + *Two dispositions worth their reasoning* | Same, plus: "switching carriers today would invalidate the interface signature guard and the behavioural scenarios that merged days ago" | **The second clause is void.** Nothing has merged. There is no signature guard and there are no behavioural scenarios |
| ADR 0005 § D4 row G13 | Decision only; default = keep channels; Layer 1 impact: none. Its reasoning repeats the same two arguments and names an AST signature guard "merged days ago" | Same split: the hazard argument stands, the sunk-cost argument is void |

doc 0002 says this in one sentence and it is the sentence this exploration exists to honour:

> **The carrier decision is genuinely reopened.** The retired plan's recommendation to keep channels rested partly on "switching now would invalidate a shipped signature guard and behavioural scenarios merged days ago". Nothing is shipped, so that argument is void. AI-02 must decide the carrier on its merits alone.

So the question is: with the sunk-cost half of the argument deleted, does the remaining half still carry the decision? That is the whole of § 4.

## 4. The carrier — the option space, argued honestly

`V-STR-02` defines a **carrier** as the mechanism by which a consumer receives a stream's events at the package boundary. AI-01 deliberately fixed the concept and refused the choice. Three options exist.

### 4.1 Option A — a receive-only channel

The producer runs on its own goroutine, sends events, and closes the carrier. The consumer receives until closed.

**What it gives:**

- The carrier is **composable with waiting**. A consumer that must wait on this stream *and* on something else — a second stream, a deadline, a permission answer arriving from a frontend, an interrupt — can do so with one language construct and no extra goroutine.
- The producer is **decoupled from consumer work**. The network read and the consumer's per-event processing proceed independently, up to the buffer's capacity.
- The **terminal event is an ordinary element**. `V-STR-18` says a stream ends with exactly one terminal event, and `V-FAIL-10` says a mid-flight failure *is* that event. A carrier of events transports it with no second mechanism.
- Backpressure is real and observable: a full buffer makes the producer wait. That is a property `V-STR-08` requires and AI-34 measures.

**What it costs:**

- A goroutine per live stream, whose exit must be guaranteed by discipline rather than by structure.
- **Abandonment** (`V-STR-07`) — a consumer that stops reading and never cancels — cannot be prevented by the type. It becomes a documented contract violation.
- A buffer capacity must be chosen, and a wrong choice is invisible until it is measured.

### 4.2 Option B — a range-over-func iterator

The package boundary hands back an iterator function. The consumer loops over it; the production body runs on the consumer's own goroutine, one event at a time.

This is the option the record never argued against, only against its cost of adoption. With that cost gone it deserves its strongest form, so here it is:

- **The stranded-producer hazard stops being a hazard and becomes an impossibility.** There is no separate producer goroutine to strand. If the consumer stops looping, the production body unwinds and returns. The entire class of leak that AI-22.4, AI-33 and every leak assertion in this plan exist to police simply does not arise.
- **Abandonment becomes a supported operation rather than a violation.** Leaving the loop early is ordinary, idiomatic, and correct. Closing-checklist item 3's uncomfortable clause — "a documented contract violation … because it cannot be tested to termination" — disappears. A contract clause that cannot be tested is a liability; an option that deletes it is worth serious consideration.
- **The buffering question dissolves.** With production driven by consumption there is no queue, so there is no capacity to pick, nothing to saturate, and no sanctioned loss path. Question 4 evaporates, AI-34 shrinks to a measurement of transport behavior, and `V-STR-09` describes a circumstance that cannot occur.
- **Ergonomics at the call site are better**, and the language has moved: the iteration shape is a first-class part of Go 1.26, the standard library composes over it, and a consumer that wants a slice of everything has a one-call answer.
- **It is the newer, more constrained shape**, and "prefer the more constrained construct" is usually the right instinct.

That is a strong case. Four things defeat it.

**Defeat 1 — Layer 2 must wait on several things at once, and an iterator cannot be waited on.**
This is decisive and it is documented, not speculative. doc 0001 § 4.1 requires the loop to suspend a tool call for a permission decision while "suspension must not block the other concurrent calls, and must not block event delivery". doc 0003 turns that into a test: AG-10.3 item 1 — "WHEN one call is suspended THEN sibling calls schedule, execute, and emit events; **message deltas already in flight keep flowing**". A consumer that is inside a loop over an iterator is not waiting on anything else; it is executing the producer. To wait on a stream *and* a permission answer *and* a cancellation, that consumer must convert the iterator back into something waitable — which the standard library does by spawning a goroutine. The leak class returns, relocated from one producer to every consumer, and now uncounted.

**Defeat 2 — the consumer must not be the socket reader.**
Under Option B, the time a consumer spends handling an event is time nobody is reading the provider connection. Layer 2's per-event work is not trivial: it accumulates deltas, drives the permission protocol, schedules tool execution, aggregates cost. Provider streaming connections have idle expectations, and a consumer that pauses for a long tool execution stalls the read. The honest answer under Option B is to add a goroutine and a queue *inside* the iterator — at which point the iterator is a façade over Option A, with the buffer hidden where no contract mentions it and no test can reach it. Hiding a queue is worse than declaring one.

**Defeat 3 — the terminal event has nowhere clean to go.**
`V-STR-18` is emphatic: exactly one terminal event, nothing after it, and a failure is delivered *as* that event (`V-FAIL-10`, `V-FAIL-12`). Under Option B there are two shapes available. A pair-yielding iterator surfaces an error alongside every element, which invites a per-element error check and quietly contradicts "exactly one terminal". A single-value iterator plus an after-the-loop error accessor puts the failure out of band, and then a consumer that left the loop early cannot tell "it failed" from "I stopped". Option A needs neither: the terminal event is an element, and the closed carrier is the end.

**Defeat 4 — the shape carries the wrong connotation.**
The iteration shape in Go is built for walking collections: in-memory, repeatable, side-effect-free. A Layer 1 stream is single-use, side-effecting, cancellable and network-backed. Handing one back in a collection-walking shape teaches a reader an expectation the contract does not honour — most concretely, that ranging twice is meaningful. `V-PRV-10` records the equivalent hazard for the fake provider: a fake that teaches consumers the wrong physics is worse than no fake, because they build on what it does. A carrier that teaches the wrong physics has the same defect at a wider blast radius.

### 4.3 Option C — offer both at the boundary

Rejected quickly, and worth recording so nobody proposes it as a compromise. Two carriers at the boundary means: the conformance suite (`V-PRV-12`) proves every contract twice; the fake provider (`V-PRV-10`) implements both faithfully, including the ownership and loss behavior of each; AI-20.4's signature guard pins two shapes; and Layer 2 splits into two dialects, so an example written against one does not compile against the other. Two carriers is not a compromise between the options, it is the union of their costs.

### 4.4 What the ergonomic loss is worth, and where it is recovered

Option B's genuine win is call-site ergonomics, and it is not nothing. doc 0002 already places the recovery: AI-22.5 exposes an iterator-shaped view from the test kit, and pins it so that the package boundary still speaks the decided carrier. The view is a convenience over a stream a consumer already holds; it is not a second contract. That is where the ergonomics go, and the cost of putting them there is one node in wave 3.

The one thing that is genuinely **not** recovered is Option B's structural answer to abandonment. Under Option A, abandonment stays a documented violation. That is a real cost of this decision and § 6 states it rather than files it away.

## 5. Ownership — what "exactly once" can mean, and three readings that are wrong

`V-STR-05` says the producer creates and closes the stream exactly once, across completion, error and cancellation alike, and nothing else closes it. The phrase admits several readings, three of which produce defects:

1. **"Closed once per successful run."** Leaves the error and cancellation paths unstated and invites a second closing site on each, which is how a double-close arrives. doc 0002's checklist anticipates this exactly: "what 'exactly once' means across the completion, error and cancellation paths is **stated, not implied**".
2. **"Closed once, by whoever finishes last."** Legitimises a producer with several sending goroutines and turns close into a coordination problem. The plan needs one owner, not a consensus.
3. **"Closed once, and the consumer may also close it when it is done early."** The direct route to sending on a closed carrier. doc 0001 § 9 already forbids it — "Nothing closes a channel it does not own" — and `V-STR-04` says the consumer never closes.

The reading this milestone must adopt is the structural one: **one owning goroutine, one closing site, executed on every exit path, always after the last send attempt.** Everything else follows, including the property AI-20.3 tests — no send after close — which is a consequence of single ownership rather than an extra rule.

An adjacent question this milestone must answer, because AI-24's adapter will otherwise answer it silently: an adapter that internally reads a transport on one goroutine and translates on another. The rule that keeps it honest is that internal fan-in happens **below** the boundary; exactly one goroutine ever sends on the carrier.

## 6. Cancellation — what is testable, and what is only statable

Three obligations, and they are not the same kind of claim:

- **The caller owns a cancellable context.** Structural: it is a parameter, and AI-20.4's signature guard can see it.
- **Every send waits on both the stream and cancellation.** Testable: a producer whose consumer stops reading exits when the context ends, and AI-20.3 item 2 asserts it.
- **Cancellation closes the stream within bounded time.** Testable with a deadline, which is what AI-22.1's timeout-safe drain exists to make cheap.

And one obligation that is none of these: **abandonment without cancellation** (`V-STR-07`). No test proves a goroutine never exits — a bounded observation that it has not exited yet is a different and weaker claim. This is why doc 0002 requires the statement to live in the package contract, and why AI-40.3 restates it at the v1 freeze. A rule that no test enforces survives only if it is written where the next reader will find it.

There is a second-order consequence worth surfacing now: if abandonment is a violation, then **the legal ways for a consumer to end a stream must be enumerated**, or "violation" has no complement. Draining to close, and cancelling, are the two. Nothing else is legal, and nothing else needs to be.

## 7. Buffering — a capacity is a hypothesis, and it should be labelled as one

`V-STR-08` requires the buffer to be bounded and leaves the number to this milestone; AI-34 revisits it with measurements. The framing that keeps this from becoming an arbitrary constant defended forever is to treat the number as a **falsifiable starting hypothesis** and to say, here, what would falsify it.

The candidate range, and what each end buys:

| Capacity | Consequence |
| --- | --- |
| Zero (a rendezvous) | Every event is a handshake. The transport read stalls on each consumer step. Maximal backpressure fidelity, minimal tolerance for a consumer that pauses at all |
| Small (single digits) | Absorbs jitter between adjacent events, nothing more. A consumer that pauses for one unit of real work still stalls the read |
| Tens | Absorbs a burst on the order of one streamed block — a run of deltas — so a brief per-block pause never reaches the transport |
| Hundreds or more | Hides genuine slowness, grows memory per concurrent stream, and widens the cancellation drop window. Backpressure stops being observable, which is the property AI-34 needs to measure |

Two constraints narrow it further. Streams are **concurrent** in the real system — parallel tool-driven calls, compaction calls, and nested subagent runs (doc 0001 § 6 seams 5 and 12) — so capacity is paid per live stream. And the capacity bounds the sanctioned loss path: on cancellation with a saturated buffer, the events dropped are at most the events resident, so a larger buffer means a larger silent loss.

## 8. Failure delivery — two axes that get conflated, and the conflation has a name

AI-01 already drew the grid in its § 6: the **owner** axis (caller-contract versus provider/transport) and the **delivery** axis (pre-stream versus mid-stream) are orthogonal. This milestone owns the delivery axis only: `V-FAIL-11` and `V-FAIL-12`.

The trap is a third distinction that looks like the delivery axis and is not: **whether any content was already emitted**. That is `V-FAIL-09`, the partial-output discriminator, and doc 0001 § 7 **G8** records what happens when it is confused with delivery — "retry if nothing completed" is precisely the predicate that gets this wrong, and it is called out as the most common real-world failure.

So the exploration's finding is that the decision must state **what event separates the two delivery paths**, and must state it as something a reader can point at rather than a feeling about earliness. Two candidates:

- *The first event.* Attractive and wrong: it makes a stream that fails before emitting anything a pre-stream failure, which is precisely the conflation with `V-FAIL-09` above.
- *The handover of the carrier.* Once the call has returned a carrier, everything after is mid-stream, including a failure that arrives before any event. Before that, nothing exists to deliver on. This one is observable, has no grey zone, and keeps the two axes orthogonal.

The second is the only one that survives contact with `V-FAIL-09`, and AI-19.5's third test item — a caller that only inspects the returned failure and a caller that only inspects the terminal event can each classify every failure — is only satisfiable if the boundary is crisp.

One residual: **what a caller observes on ordinary cancellation.** It is not in doc 0002's checklist as a separate item, but it falls out of items 3, 4 and 5 together and AI-20.2 item 2 already assumes an answer (a context cancelled at call time "reports the cancellation category"). Leaving it unstated would force AI-19 or AI-20 to invent it, which is the failure this milestone exists to prevent. It belongs in the decision.

## 9. Vocabulary check against AI-01

Every noun this decision needs, checked against the register:

| Concept this decision assigns behavior to | Register row | Present? |
| --- | --- | --- |
| The thing being decided | `V-STR-01` stream, `V-STR-02` carrier | yes |
| Who creates, who receives | `V-STR-03` producer, `V-STR-04` consumer | yes |
| Closing discipline | `V-STR-05` stream ownership | yes |
| The caller-owned signal | `V-STR-06` cancellation | yes |
| Reading without cancelling | `V-STR-07` abandonment | yes |
| The queue and its bound | `V-STR-08` bounded buffer | yes |
| The one place events may be lost | `V-STR-09` sanctioned loss path | yes |
| How a stream ends | `V-STR-18` terminal event, `V-STR-20` completion event, `V-FAIL-10` terminal error event | yes |
| The two delivery paths | `V-FAIL-11`, `V-FAIL-12` | yes |
| Content already delivered, and the fact of it | `V-FAIL-08` partial output, `V-FAIL-09` partial-output discriminator | yes |
| The ergonomic adaptation AI-22.5 provides | — | **missing** |
| Waiting rather than dropping, as a named posture | — | **missing** (the word is used inside `V-STR-08`'s definition, never defined) |

Two gaps. Per AI-01 § 9 rule 2, a missing term is **appended to AI-01's register in the same pull request that needs it**, never invented locally. Both are container-side, so both take the next free `V-STR` ordinals. They are carried into the proposal as a scoped amendment.

## 10. Out of scope for this change

- **Any event type, kind, payload or sequence rule.** AI-14 owns the envelope; this milestone must not pre-empt it, and states behavior of the container only.
- **The provider interface itself.** AI-20 declares it; this decision constrains what it must document.
- **The error taxonomy.** AI-19 owns categories, retryability and the constructible terminal payload. This milestone decides *delivery*, not classification.
- **Buffer sizing by measurement.** AI-34.1 confirms or changes the number, and decides whether it is a constant or configurable. This milestone deliberately does not decide configurability — deciding it now would remove an option from a milestone whose whole purpose is to have measurements this one does not.
- **Leak detection mechanism.** AI-22.4 chooses it, under its own dependency constraint.
- **Layer 2's carrier.** doc 0003's AG-01.1 decides it. This decision is an input to that one, not a substitute for it.

## 11. Open questions carried into the proposal

1. Does the carrier decision hold once the sunk-cost argument is removed? → § 4; the answer is yes, on four grounds none of which is cost of change.
2. What exactly does "exactly once" bind? → § 5; one owning goroutine, one closing site, every exit path.
3. What separates pre-stream from mid-stream delivery? → § 8; the handover of the carrier, not the first event.
4. What capacity, and what would falsify it? → § 7; the number and its falsification criteria are the proposal's to fix.
5. What does a caller observe on ordinary cancellation? → § 8 residual; must be answered here or AI-19 invents it.
6. Which two terms are appended to AI-01? → § 9.

## 12. Evidence gate for this milestone

AI-02.1 is a `[decision]` leaf. doc 0002's global evidence gate — recorded green `make test` in `backend/agent/` — binds behavior and guard leaves. A decision leaf closes when "the decision artifact answers every listed question and is merged". This change writes no Go, touches nothing under `backend/`, and runs no test command. Its verification is inspection against the closing checklist, recorded in `tasks.md`.
