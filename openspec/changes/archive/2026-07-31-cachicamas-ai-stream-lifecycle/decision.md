# Decision — the Layer 1 stream lifecycle, ownership and carrier

> **Change**: `cachicamas-ai-stream-lifecycle`
> **Milestone**: AI-02 — Decide stream lifecycle, ownership, and the carrier
> **Node**: AI-02.1 — Lifecycle, ownership and carrier `[decision]`
> **Status**: decided
> **Date**: 2026-07-31
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: the concern doc 0001 and ADR 0005 track as **G13**
> **Sources**: [doc 0002 — Layer 1 task graph](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0003 — Layer 2 task graph](../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) · [doc 0001 — agent stack v2](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005 § D4](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Binding vocabulary**: [AI-01's register](../2026-07-31-cachicamas-ai-contract-vocabulary/decision.md) — every noun below is one of its rows, cited by identifier

> [!IMPORTANT]
> **This artifact decides behavior, not code.** No Go type name, field name, method name, or package identifier appears here — doc 0002's authoring constraint. AI-14 and AI-20 choose spellings. Where a language or standard-library shape has to be discussed, it is named descriptively ("the single-value iterator function shape") rather than spelled, so that the constraint holds even inside the argument that most tempts a reader to break it.

---

## 1. How to use this document

**If you are writing AI-14, AI-19, AI-20, AI-21, AI-22, AI-33, AI-34 or AI-40:** § 10 tells you what your milestone inherits, in your milestone's own terms. Start there, then read the one decision section it points at. You should not need the other four.

**If you are reviewing this artifact:** § 12 walks AI-02.1's closing checklist against it. § 3 is where the argument lives and where a defect is most expensive.

**If you disagree with a decision:** each of § 3 … § 7 carries a *What this excludes* part naming the alternatives and why each lost. If your objection is there, it was considered. If it is not, it is new, and § 11 rule 4 says what to do with it.

**Every Layer 1 noun below resolves to an AI-01 register row**, cited by identifier rather than paraphrased. Two nouns the register lacked — `V-STR-22` **carrier view** and `V-STR-23` **backpressure** — were appended to it by this same change, under AI-01 § 9 rule 2. They are used here; they are not defined here.

**Section order is AI-02.1's closing-checklist order**, so a reviewer can walk doc 0002 and this document in parallel.

---

## 2. What was decided

Five conclusions, before any argument, for the reader who came for one of them.

| # | Question | Decision |
| --- | --- | --- |
| 1 | **Carrier** (`V-STR-02`) | A **receive-only channel** at the package boundary. Iterator ergonomics are delegated to AI-22.5 as a `V-STR-22` **carrier view**. doc 0002 needs no amendment nodes |
| 2 | **Ownership** (`V-STR-05`) | **One sending goroutine, one closing site, every exit path, after the last send attempt.** Nothing else closes |
| 3 | **Cancellation** (`V-STR-06`, `V-STR-07`) | Caller-owned signal; every send waits on it; bounded close. The legal consumer endings are **drain to close** and **cancel**. Anything else is **abandonment** — a documented contract violation, stated in the package contract because it cannot be tested to termination |
| 4 | **Buffering** (`V-STR-08`, `V-STR-09`, `V-STR-23`) | Bounded, **starting capacity 64**, falsifiable at AI-34.1. Backpressure is waiting, never dropping. Exactly one sanctioned loss path: cancellation with a saturated buffer drops late events and closes without a terminal |
| 5 | **Failure delivery** (`V-FAIL-11`, `V-FAIL-12`) | The boundary is **the handover of the carrier**, not the first event. Before it, the failure is returned directly and no stream and no producer exist. After it, every failure arrives as the terminal error event |

---

## 3. Decision 1 — the carrier

**Closing-checklist item 1.**

### Decision

The carrier (`V-STR-02`) at the Layer 1 package boundary is a **receive-only channel of events**. A provider hands the consumer a stream (`V-STR-01`) it may only receive from; the producer (`V-STR-03`) sends and closes; the consumer (`V-STR-04`) receives until the stream is closed.

The range-over-func iterator was considered as a genuine alternative and rejected. Because channels won, doc 0002's branch applies: **the iterator-ergonomics requirement is delegated to AI-22.5**, which exposes a `V-STR-22` **carrier view** from the stream test kit (`V-PRV-11`). doc 0002's waves 2–5 therefore gain **no** amendment nodes under the living-graph clause.

### Why

doc 0002 says the carrier choice is free "here, and only here", and it deletes half of the argument that previously supported the default:

> **The carrier decision is genuinely reopened.** The retired plan's recommendation to keep channels rested partly on "switching now would invalidate a shipped signature guard and behavioural scenarios merged days ago". Nothing is shipped, so that argument is void.

**No ground below is a cost-of-change ground.** The sunk-cost argument that doc 0001 § 7 and ADR 0005 § D4 both carried is treated as void, because it is void: there is no signature guard, there are no merged scenarios, and there is no adapter. This decision is made as though the module were empty, because it is.

#### First, the iterator case, at its strongest

A reader who arrived believing the boundary should hand back an iterator should recognise their own argument here before reading a rebuttal. Four things are genuinely true of that option, and three of them are structural advantages that no amount of discipline buys back on the channel side:

1. **The stranded-producer hazard stops being a hazard and becomes an impossibility.** doc 0001 § 7 names it as the canonical objection to channels: "a consumer who stops reading strands the producer goroutine forever, and goroutines are not collected." Under an iterator there is no separate producer goroutine to strand — production runs on the consumer's own goroutine, one event at a time. If the consumer stops looping, the production body unwinds and returns. The class of defect that AI-22.4, AI-33's leak assertions and every leak-detection helper in this plan exist to police does not arise.
2. **Abandonment becomes a supported operation rather than a violation.** Leaving the loop early is ordinary and correct. Closing-checklist item 3's uncomfortable clause — that abandonment is a violation which "cannot be tested to termination" — would simply not need writing. A contract clause no test enforces is a liability, and an option that deletes the clause deserves a hearing.
3. **The buffering question dissolves.** With production driven by consumption there is no queue: no capacity to choose, nothing to saturate, no `V-STR-09` sanctioned loss path, and no arbitrary constant to defend until AI-34 measures it. Closing-checklist item 4 would become a two-line answer.
4. **Ergonomics and language direction.** The iteration shape is first-class in Go 1.26, the standard library composes over it, and the call site reads better than a receive loop. Preferring the newer, more constrained construct is usually the right instinct.

That is the case. It loses on four grounds, and each cites a document.

#### Ground 1 — Layer 2 must wait on several things at once, and an iterator cannot be waited on

This is the decisive one, and it is documented rather than anticipated.

doc 0001 § 4.1 makes the requirement explicit: the loop drives the permission protocol by "emitting the request, suspending that call, resuming on the decision", and "suspension must not block the other concurrent calls, **and must not block event delivery**." doc 0003 turns it into a test, AG-10.3 item 1:

> WHEN one call is suspended THEN sibling calls schedule, execute, and emit events; **message deltas already in flight keep flowing** — proven with synchronization points holding one call suspended while others complete.

A consumer that is inside a loop over an iterator is not waiting on anything. It *is* the producer, executing it. To wait on this stream and on a permission decision and on an interrupt at the same time, that consumer must convert the iterator into something waitable — which is done by spawning a goroutine and a queue. The leak class from the iterator's first advantage returns, relocated from one producer to every consumer, uncounted and undocumented.

A channel is waitable by construction. The multiplexing that Layer 2's whole permission, delegation and cancellation design assumes (doc 0001 § 6 seams 2, 7, 12) is one language construct away, with no additional goroutine and nothing to leak.

#### Ground 2 — the consumer must not be the socket reader

Under an iterator, every moment the consumer spends handling an event is a moment nobody is reading the provider connection. Layer 2's per-event work is not trivial: it accumulates deltas (`V-STR-16`), drives the permission protocol, schedules tool execution with a concurrency policy, and aggregates cost (doc 0001 § 4.1). A consumer that pauses for a tool execution stalls the transport read for the duration.

The honest repair under an iterator is to add a goroutine and a queue *inside* the iterator. That is the channel design, with the queue moved somewhere no contract mentions it, no capacity is stated, and AI-34 cannot measure it. **A hidden buffer is worse than a declared one** — this plan's entire posture on `V-STR-08` is that boundedness must be a contract rather than an implementation detail, precisely because "an unbounded buffer converts backpressure into memory growth" invisibly.

#### Ground 3 — the terminal event has nowhere clean to go

`V-STR-18` is emphatic: exactly one terminal event per stream, nothing follows it, and its two instances are the completion event (`V-STR-20`) and the terminal error event (`V-FAIL-10`). A mid-flight failure *is* an event, by `V-FAIL-12`.

An iterator has two shapes available and both fight that contract. A pair-yielding shape surfaces a failure alongside every element, which invites a per-element check and quietly contradicts "exactly one terminal". A single-value shape plus an after-the-loop failure accessor puts the failure out of band — and then a consumer that left the loop early cannot distinguish "it failed" from "I stopped", which is the same ambiguity `V-FAIL-09` exists to eliminate one axis over.

A channel of events needs neither. The terminal event is an element like any other, and the closed channel is the end. One mechanism, no second channel, no accessor.

#### Ground 4 — the shape carries the wrong connotation

The iteration shape in Go is built for walking collections: in-memory, repeatable, side-effect-free, cheap. A Layer 1 stream is single-use, side-effecting, cancellable and network-backed. Handing one back in a collection-walking shape teaches a reader an expectation the contract does not honour — most concretely, that iterating twice means something.

`V-PRV-10` records the same hazard at a smaller radius, for the fake provider: "a fake that closes cleanly where the real contract drops events teaches consumers the wrong physics, and they build on what the fake does." A carrier that teaches the wrong physics has the identical defect with every consumer in its blast radius instead of every test.

#### What is conceded

Ground 1's answer to the iterator's first advantage is *not* that the stranded producer is impossible under channels. It is that the hazard is closed by the **send discipline** rather than by the carrier: every send waits on the stream and on cancellation, and the caller owns the context (§ 5). doc 0002 states this directly, and it is the reason the hazard does not decide the question:

> The stranded-producer objection to channels … is answerable by the send discipline this decision adopts (every send selects on cancellation, the caller owns the context), not by the carrier.

The residual — a consumer who abandons *and* never cancels — remains. It is a contract violation, it is not preventable by the type, and it is not testable to termination. **That is a real cost of this decision**, it is the iterator's strongest single advantage, and § 5 pays it in the only currency available: a statement in the package contract, restated at the freeze by AI-40.3.

### What this excludes

| Excluded | Why |
| --- | --- |
| **A range-over-func iterator at the boundary** | Grounds 1–4 above |
| **A pair-yielding iterator** carrying a failure beside each element | Contradicts `V-STR-18`'s exactly-one-terminal discipline (ground 3) |
| **A single-value iterator plus an after-the-loop failure accessor** | Ambiguous after an early exit (ground 3) |
| **Both carriers offered at the boundary** | Not a compromise, the union of both costs: the conformance suite (`V-PRV-12`) proves every contract twice; the fake provider (`V-PRV-10`) implements both faithfully including each one's ownership and loss behavior; AI-20.4's signature guard pins two shapes; Layer 2 splits into two dialects, so an example written against one does not compile against the other |
| **A send-capable stream handed to the consumer** | The consumer never sends and never closes (`V-STR-04`). Receive-only is the type-level half of § 4's ownership rule |

### Consequences

1. **AI-22.5 owns the ergonomics.** The `V-STR-22` carrier view it exposes is a convenience over a stream the consumer already holds — **never a second contract**. AI-22.5's second test item is the mechanical form of that claim: AI-20.4's signature guard passes unmodified.
2. **doc 0002 requires no amendment.** The waves 2–5 amendment nodes that the iterator branch would have triggered under the living-graph clause are not needed. This is a positive outcome and is recorded as one, so nobody later reads the absence as an omission.
3. **doc 0003 AG-01.1 keeps its documented default.** Its charter already names channels "matching AI-02", for the same reasons. Layer 2's carrier decision remains its own, but it now mirrors a decision that was argued rather than inherited.
4. **A producer goroutine exists per live stream**, which is what makes § 4's ownership rule and § 5's cancellation obligations load-bearing rather than ceremonial.

### Who inherits it

AI-14 (the envelope travels on it), AI-20.1 and AI-20.4 (the signature and its guard), AI-21 (the fake sends on it), AI-22.5 (the view over it), doc 0003 AG-01.1 (symmetry).

---

## 4. Decision 2 — ownership

**Closing-checklist item 2.**

### Decision

`V-STR-05` says the producer creates the stream and closes it **exactly once**, across the completion, error and cancellation paths alike, and nothing else closes it. This decision states what that binds, structurally:

1. **Exactly one goroutine ever sends on a stream.** An adapter that internally reads a transport on one goroutine and translates on another performs its fan-in **below** the boundary; the boundary sees one sender.
2. **Exactly one closing site exists in the producer.** Not one per path — one, total.
3. **That site runs on every exit path**, including completion, terminal error, cancellation, and an unwinding exit.
4. **It runs after the last send attempt and never before**, which is why "no send after close" needs no separate rule: with a single sender whose final act is the close, a send after close is unreachable.
5. **Nothing else closes.** Not the consumer (`V-STR-04`), not the stream test kit, not Layer 2's harness, not a frontend. doc 0001 § 9 states the general rule this instantiates: "Nothing closes a channel it does not own."

### Why

"Close the stream exactly once" is a claim about all executions; a reader sees one text. Stated procedurally it is unverifiable by reading, and doc 0002 anticipated the consequence by requiring the meaning to be "**stated, not implied**".

Three readings of "exactly once" are available and each produces a defect. They are recorded because each is a shape a reviewer can look for:

| Misreading | Defect it produces |
| --- | --- |
| "Closed once per successful run" | The error and cancellation paths acquire their own closing sites. Double close is one refactor away |
| "Closed once, by whoever finishes last" | Several senders, and close becomes a coordination problem the contract never mentions |
| "Closed once, and the consumer may also close it when it is done early" | A send on a closed stream. Forbidden by `V-STR-04` and doc 0001 § 9, and the most common way the rule is broken in good faith |

The structural form forecloses all three, because each is a *shape* — a second closing site, several senders, a consumer-side close — rather than a behavior, and shapes are visible in a diff.

### What this excludes

- **Any protocol by which a consumer signals "I am done" by closing something.** A consumer that wants to end a stream early cancels (§ 5). That is the mechanism, and it is the only one.
- **A close performed by a wrapper, a decorator, or a test helper.** A `V-STR-22` carrier view does not own the stream it views and does not close it.
- **Several producing goroutines coordinating a close.** The coordination is moved below the boundary, where it is an adapter's private business.

### Consequences

- AI-20.3 item 1 — "the producer creates the stream and closes it exactly once, on every path: completion, terminal error, and cancellation" — becomes a test of a stated structure rather than an aspiration.
- AI-20.1 item 3's obligation to document "who closes the stream" is satisfiable verbatim from § 9.
- AI-21's fake provider inherits the same rule, and `V-PRV-10`'s contract-faithfulness clause makes it binding: a fake with a second closing site would teach Layer 2 a physics Layer 1 does not have.

### Who inherits it

AI-20.1 (documentation), AI-20.3 (proof), AI-21 (the fake obeys it), AI-24 onward (every adapter's internal fan-in rule), doc 0003 AG-01.1 item 4 (the agent-level mirror).

---

## 5. Decision 3 — cancellation, and abandonment

**Closing-checklist item 3.**

### Decision

**Cancellation** (`V-STR-06`) is the caller-owned signal that ends a stream early. Three obligations, each marked with what proves it:

| Obligation | Kind | Proven by |
| --- | --- | --- |
| The caller owns a cancellable context, supplied on the call that creates the stream, and the stream's lifetime is bounded by it | structural | AI-20.4's signature guard sees the parameter |
| **Every** send waits on both the stream and cancellation. No send is unconditional | testable | AI-20.3 item 2 against the fake; AI-33 against a transport |
| Cancellation closes the stream within bounded time | testable | AI-20.3 item 4, with AI-22.1's timeout-safe drain making the deadline cheap |

**"Bounded" is defined by what it excludes.** Once cancellation is observable, the producer begins no new blocking wait on the network and no new blocking wait on the consumer. It finishes its closing sequence and exits. A backoff waits on the signal rather than sleeping — doc 0001 § 9, restated here because a sleeping backoff is the most common way a bounded close becomes an unbounded one.

**The legal ways for a consumer to end a stream are exactly two:**

1. **Drain to close** — receive until the stream is closed.
2. **Cancel** — signal, then drain what remains or stop.

**Anything else is abandonment** (`V-STR-07`): a consumer that stops reading and never cancels.

> **Abandoning a stream without cancelling it is a documented contract violation, not a supported mode.**

This sentence belongs in the package contract. It is not testable to termination — no test proves a goroutine never exits, and a bounded observation that it has not exited *yet* is a strictly weaker claim that would be mistaken for the stronger one. doc 0001's defect **C3** is what that mistake looks like when it ships: "A shipped test documents the resulting gaps as expected." A rule that no test enforces survives only if it is written where the next reader will find it, which is why doc 0002 requires it in the contract and why **AI-40.3 restates it at the v1 freeze**.

### Why

Enumerating the legal endings is what gives "violation" a complement. Without the enumeration, "abandonment is a violation" is a prohibition with no stated alternative, and a consumer with a legitimate need to stop early has to guess. With it, the answer is one word: cancel.

Marking each obligation testable or statable is the second half. The failure this prevents is an untestable clause acquiring a test that proves something weaker — the test then stands in for the clause, the clause gets deleted as redundant, and the weaker property is all that remains. Keeping the abandonment clause explicitly *outside* the testable set is what protects it.

### What this excludes

- **Cancellation as a polled flag.** It is a context. doc 0001 § 9.
- **A producer that treats cancellation as advisory** and finishes its work first. Bounded means bounded.
- **An unconditional send anywhere in a producer**, including the terminal event (see § 7).
- **A supported "fire and forget" consumption mode.** There is none. A consumer that will not drain must cancel.
- **A finaliser, a timeout, or a sweeper that closes an abandoned stream.** Nothing but the producer closes (§ 4), and adding a rescuer would legitimise the violation while hiding it.

### Consequences

- AI-20.1 item 3's second and third documentation obligations — who owns the context, and what abandoning without cancelling means — are satisfiable verbatim from § 9.
- AI-20.3 items 2 and 4, AI-21.5 (cancellation fidelity in the fake), AI-22.4 (where leak assertions apply), AI-33 and AI-40.3 all inherit stated obligations rather than inferred ones.
- The abandoned-**then-cancelled** path is testable and is where the leak assertions go; the abandoned-**never-cancelled** path is the documented violation and gets no test pretending otherwise.

### Who inherits it

AI-20.1, AI-20.2 (a context already cancelled at call time — see § 7), AI-20.3, AI-21.5, AI-22.4, AI-33, AI-40.3.

---

## 6. Decision 4 — buffering and backpressure

**Closing-checklist item 4.**

### Decision

The buffer between producer and consumer (`V-STR-08`) is **bounded**, with a **starting capacity of 64 events**.

**Backpressure** (`V-STR-23`) means **waiting, never dropping**: when the buffer is full the producer waits — and that wait, like every send, also waits on cancellation (§ 5), so a full buffer never makes a stream uncancellable.

**Exactly one loss path is sanctioned** (`V-STR-09`), and it is the only circumstance in which a Layer 1 stream may lose an event:

> On **cancellation** with a **saturated** buffer, late events are **dropped** and the stream **closes without a terminal event**.

Three corollaries, each stated because each is a place a reader goes wrong:

1. A consumer that treats a missing terminal event **after its own cancellation** as corruption is the party in error. The contract says this so that the consumer, not the producer, carries the burden.
2. A stream that closes without a terminal event and was **never cancelled** is a producer defect. It is not a second loss path.
3. Every other path is lossless. Slow consumers, bursty consumers, pause-and-resume consumers all receive every event in order — the producer waits.

**Whether the capacity is a constant or configurable is not decided here.** That is AI-34.1's, deliberately.

### Why the bound

An unbounded buffer converts backpressure into memory growth (`V-STR-08`) — silently, and worst under exactly the conditions where memory is already tight. Boundedness has to be a contract rather than an implementation choice, because an implementation choice is reversible by one well-meaning change.

### Why 64

The buffer's job is to absorb a burst on the order of **one streamed block** (`V-STR-14`), so that a consumer's brief per-block work never reaches the transport read. A run of streamed text arrives as tens of deltas (`V-STR-16`), bracketed by a start and an end event, with metadata interleaved. 64 covers a typical block with headroom. It is a power of two so that no reader mistakes it for a measured figure — it is not one, and § "What would change it" says so.

Both ends of the range, and what each costs:

| Capacity | What it costs |
| --- | --- |
| **Zero** (a rendezvous) | Every event is a handshake; the transport read stalls on every consumer step. Maximum backpressure fidelity, zero tolerance for a consumer that pauses at all — and Layer 2's consumer pauses by design, to drive the permission protocol |
| **Single digits** | Absorbs jitter between adjacent events and nothing more. A consumer that pauses for one unit of real work still stalls the read |
| **64** | Absorbs one block's burst. A consumer that falls more than a block behind is genuinely slow and *should* feel backpressure rather than have it hidden |
| **Hundreds or more** | Hides genuine slowness, so the very property AI-34 must measure — does the producer ever wait? — becomes unobservable. Memory grows per live stream. And the drop window widens: the events lost on the sanctioned path are at most the events resident, so a bigger buffer means a bigger silent loss |

**Capacity is paid per live stream, and streams are concurrent.** Parallel tool-driven calls (doc 0001 § 6 seam 5 / **G5**), compaction calls with their own provider and cancellation (seam 5), and nested subagent runs (seam 12 / **G7**) all mean several streams are live at once. At 64 small events per stream, a dozen concurrent streams is still kilobytes — which is why memory is *not* the ground this number stands on. It stands on burst absorption, and the drop window is the constraint that keeps it from growing.

### What would change it — the experiment AI-34.1 inherits

The number is a **hypothesis**, not a result. There is no adapter, no transport and no workload today, so there cannot be a measured answer. AI-34.1's charter is to confirm or change it "**with measurements** — the decision records the workload measured and the numbers, not a preference." Three measurements, with the direction each result implies:

| # | Measurement | Result → direction |
| --- | --- | --- |
| **M1** | Buffer occupancy high-water mark across representative streams, with a consumer doing realistic per-event work | Never approaches 64 → **shrink** toward the observed high-water mark plus headroom. Regularly saturates → the burst assumption is wrong; investigate before growing |
| **M2** | Whether the producer ever waits, and for how long | Never waits → backpressure is unobservable and the number is too high to measure anything; **shrink**. Waits often, and the cause is consumer latency rather than genuine slowness → evidence to **grow** |
| **M3** | Resident memory per live stream at the concurrency the harness actually reaches — parallel tools, compaction, subagents | Material at realistic concurrency → **shrink** |

**Tie-break rule, so that AI-34.1 is not left arbitrating:** when two capacities are indistinguishable on the measurements, prefer the smaller. Backpressure that can be observed is worth more than latency that was hidden, and the drop window is smaller.

### What this excludes

- **An unbounded buffer**, in any disguise, including an auxiliary queue beyond the declared capacity. AI-34.2 item 3 asserts its absence.
- **Dropping under pressure.** Backpressure is waiting. AI-34's charter puts this beyond doubt: dropping text or tool-call events is out of scope for it, and "the single sanctioned loss path stays exactly where AI-20.3 put it."
- **A second loss path**, for any reason. AI-34.3 tests exhaustively for one.
- **A decision on constant versus configurable.** AI-34.1's, and deciding it now would remove an option from the milestone that will hold the evidence.
- **Any claim that 64 is measured.** It is not.

### Consequences

- AI-20.3 item 3 proves the sanctioned loss path, including the clause that makes a consumer treating a missing terminal as corruption the party in error.
- AI-21.4 can script a producer that emits faster than an unread consumer drains, which is what makes the saturated path deterministically exercisable.
- AI-34.1 inherits an experiment with a tie-break rule rather than a constant to overturn.
- doc 0003 AG-01.1 item 2 asks whether Layer 2 may apply "the same single sanctioned loss path as Layer 1 (cancellation on a saturated channel) and no other". This decision is what that question refers to; the answer remains Layer 2's.

### Who inherits it

AI-20.3, AI-21.4, AI-22.1, AI-33.3, AI-34.1, AI-34.2, AI-34.3, doc 0003 AG-01.1.

---

## 7. Decision 5 — failure delivery

**Closing-checklist item 5.**

### Decision

**The boundary between the two delivery paths is the handover of the carrier**, not the first event.

**Before handover — `V-FAIL-11`, pre-stream failure delivery.** The call reports the failure directly and **no stream and no producer are ever created**. Not an empty stream. Not a stream that immediately yields a failure. Nothing to drain, nothing to close, no goroutine. Two families arrive here:

- Caller-contract failures (`V-FAIL-01`), which are decidable from the request alone and are found by the single validation pass that runs once, before any I/O (`V-REQ-22`).
- Provider/transport failures (`V-FAIL-05`) that occur before a stream exists: authentication rejected, connection failed, request rejected — and a context already cancelled at call time, which reports the category AI-19 assigns to cancellation.

**After handover — `V-FAIL-12`, mid-stream failure delivery.** The failure arrives as the single terminal error event (`V-FAIL-10`), and the stream then closes. **There is no second route.** A consumer never learns of a mid-stream failure by re-inspecting a returned value, from a side channel, or from an accessor.

**A stream handed over that fails before emitting any content is mid-stream.** This is the case that separates the correct boundary from the intuitive one, and it is stated explicitly because it is where the intuition fails.

**One vocabulary, two paths.** The same closed classification (`V-FAIL-06`), the same retryability (`V-FAIL-07`) and the same partial-output discriminator (`V-FAIL-09`) are reachable on both paths. That is the property that makes AI-19.5 item 3 satisfiable: a caller that only ever inspects the returned failure, and a caller that only ever inspects the terminal event, can each classify every failure the taxonomy defines.

**What a caller observes on ordinary cancellation.** Stated here because AI-20.2 item 2 already assumes an answer, and leaving it unstated would force AI-19 or AI-20 to invent one:

> A stream ends with a terminal event whenever the producer can deliver one **without waiting**. On cancellation, the producer offers a terminal error event carrying the category AI-19 assigns to cancellation, without waiting; whether or not the offer lands, it closes. The single exception is the sanctioned loss path of § 6 — a saturated buffer — where the offer cannot land and the stream closes bare.

The offer must not wait, because waiting would violate § 5's bounded close. This is why the sanctioned loss path exists at all: it is the intersection of a bounded close and a full buffer, and nothing else.

### Why the boundary is handover

Two axes, and they are orthogonal. AI-01 § 6 drew the first pair (owner versus delivery); this is the second pair, and the confusion between them has a recorded cost.

```
                      call returns a carrier
                              |
              pre-stream      |      mid-stream            ← DELIVERY axis
              V-FAIL-11       |      V-FAIL-12               decided by THIS moment
        ----------------------+---------------------------
                              |
              no content yet  |  content already emitted    ← PARTIAL-OUTPUT axis
                              |                               V-FAIL-09 — a DIFFERENT fact
```

The rejected alternative is **"the first event."** It is intuitive, and it is wrong in exactly one case: a stream that has been handed over and fails before emitting content. Under the rejected boundary that case is classified pre-stream, which fuses the delivery axis onto the partial-output axis — and doc 0001 § 7 **G8** names the resulting defect: *"a stream that dies after emitting output is the most common real failure and the one naive retry logic excludes"*, with `V-FAIL-09` recording that **"retry if nothing completed" is precisely the predicate that gets this wrong**.

Handover survives the case. It is observable, it has no grey zone, and it keeps the axes independent — which is what lets AI-19 classify and AI-35 decide retries without either reaching for the other's fact.

### What this excludes

| Excluded | Why |
| --- | --- |
| **"The first event" as the boundary** | Fuses the delivery axis onto `V-FAIL-09`; rebuilds the **G8** defect |
| **A pre-stream failure delivered as a stream that immediately fails** | Would mean a producer exists for a request that never reached a provider, and would make AI-20.2 item 1 — "no goroutine started, no carrier returned" — untestable |
| **A pre-stream failure delivered as an empty stream that simply closes** | Indistinguishable from a provider that returned nothing successfully |
| **A mid-stream failure delivered anywhere but the terminal error event** | `V-STR-18`; a second route means a consumer can miss one |
| **Two vocabularies, one per path** | `V-FAIL-12` is explicit: one vocabulary, two delivery paths |
| **Deciding categories, retryability, or the terminal payload's shape** | AI-19's. This decision fixes *where* a failure appears, never *what it says* |

### Consequences

- AI-19.5 implements the split; all three of its test items are derivable from this section, and none of them requires reopening it.
- AI-20.2 item 1 — an invalid request fails "before any stream exists: no goroutine started, no carrier returned" — is the pre-stream path stated as a test. `V-PRV-04`'s pre-stream contract is this section's obligation, named.
- AI-20.2 item 2's cancelled-at-call-time case has a stated answer, including its ordering relative to validation: validation runs first (`V-REQ-22` requires it to run once before any I/O), and the cancellation category is reported only if the request is valid.
- AI-35 inherits a clean `V-FAIL-09`, unpolluted by delivery, which is what its "the partial-output case is never retried at Layer 1" clause (`V-FAIL-15`) stands on.

### Who inherits it

AI-19.5, AI-20.2, AI-20.3, AI-21.3 (the fake scripts a terminal error), AI-23 (the suite asserts both paths), AI-35.

---

## 8. The complete lifecycle, in one picture

```
  caller ──── request + cancellable context ────► provider
                                                     │
                        ┌────────────────────────────┴───────────────┐
                        │ validation runs once, before any I/O        │
                        │ (V-REQ-22)                                  │
                        └────────────────────────────┬───────────────┘
                                                     │
          invalid, or failed before a stream exists  │  valid
                        ┌────────────────────────────┴───────────────┐
                        ▼                                            ▼
              PRE-STREAM (V-FAIL-11)                    ── HANDOVER ──────────────
              failure returned directly                 carrier handed to caller
              no stream, no producer,                    one producer goroutine
              no goroutine                               bounded buffer, capacity 64
                                                                     │
                                          ┌──────────────────────────┼──────────────────────────┐
                                          ▼                          ▼                          ▼
                                    COMPLETION                  MID-STREAM              CANCELLATION
                                    completion event            failure (V-FAIL-12)     (caller signals)
                                    (V-STR-20)                  terminal error event         │
                                          │                     (V-FAIL-10)                  │
                                          │                          │            ┌──────────┴──────────┐
                                          │                          │            ▼                     ▼
                                          │                          │      buffer has room      buffer saturated
                                          │                          │      terminal error       late events dropped
                                          │                          │      offered, lands       nothing lands
                                          │                          │            │                     │
                                          └──────────────────────────┴────────────┴─────────────────────┘
                                                                     │
                                                     ONE CLOSING SITE, ONE SENDER
                                                     runs after the last send attempt
                                                     runs on every exit path
                                                                     │
                                                                     ▼
                                                             stream closed
```

Reading the picture against the five decisions: the handover line is § 7's boundary; the single closing site at the bottom is § 4; the branch inside cancellation is § 6's sanctioned loss path; the producer goroutine and the buffer exist because of § 3; and every arrow into the closing site is a send that waited on cancellation, which is § 5.

---

## 9. What the package contract must state

AI-20.1 item 3 requires the interface's documentation to state "the ownership rules AI-02.1 decided: who closes the stream, who owns the context, and what abandoning without cancelling means." These are those statements, in the form they must survive into the package contract. They are prose obligations, not spellings.

1. **The producer creates the stream and closes it exactly once.** One goroutine sends; one closing site exists; it runs on every exit path, after the last send attempt. Nothing else closes the stream — not the consumer, not a test helper, not a consumer above Layer 1.
2. **The caller owns the cancellable context.** The stream's lifetime is bounded by it. Every send waits on both the stream and cancellation, so a stream is always cancellable, including when its buffer is full.
3. **A consumer ends a stream in exactly one of two ways: it drains to close, or it cancels.**
4. **Abandoning a stream without cancelling it is a contract violation, not a supported mode.** It is stated rather than enforced, because no test proves a goroutine never exits. A consumer that will not drain must cancel.
5. **Cancellation closes the stream within bounded time.** After cancellation is observable the producer begins no new blocking wait on the network or on the consumer, and a backoff waits on the signal rather than sleeping.
6. **The buffer is bounded. Backpressure means waiting, never dropping.**
7. **Exactly one loss path is sanctioned:** on cancellation with a saturated buffer, late events are dropped and the stream closes without a terminal event. A consumer that treats a missing terminal after its own cancellation as corruption is the party in error. A stream that closes without a terminal and was never cancelled is a producer defect.
8. **A request that never becomes a stream reports its failure directly; no stream and no producer are created.** A stream that fails after it has been handed over reports through its terminal error event, including when no content preceded the failure.

**AI-40.3 restates statements 3 and 4 at the v1 freeze**, per doc 0002's closing-checklist item 3.

---

## 10. What each blocked milestone inherits

doc 0002: *"Blocks: AI-14, AI-20, AI-21, AI-22."* Each entry is written in that milestone's own terms, so its author can check the acceptance criterion — writable without reopening — from this table alone.

### AI-14 — the event envelope

- The envelope travels on a receive-only channel (§ 3). The carrier is settled; AI-14 decides what rides on it.
- The producer stamps sequence and owns the stream's state, which is what makes `V-STR-13` a per-stream property rather than a process-wide one. § 4's single-sender rule is the structural reason AI-14.2's "two concurrent streams each start at 1" is achievable without coordination — and the reason AI-14.3's guard against a package-global has something to guard.
- `V-STR-18`'s exactly-one-terminal discipline is assumed by §§ 4, 6 and 7, and defined by AI-14.4. This decision cites it; it does not restate it as its own rule.
- **Not inherited, because it is AI-14's:** event kinds, payloads, sequence semantics, ordering invariants.

### AI-20 — the provider interface

- **AI-20.1**: the carrier in the signature (§ 3), and the three documentation obligations satisfied verbatim by § 9.
- **AI-20.2**: the pre-stream contract (`V-PRV-04`) is § 7's pre-stream path — no goroutine, no carrier, a typed failure — plus the stated ordering of validation before the cancelled-at-call-time case.
- **AI-20.3**: the mid-stream contract (`V-PRV-05`) is §§ 4, 5 and 6 together. Item 1 is § 4's ownership; item 2 is § 5's send discipline; item 3 is § 6's sanctioned loss path with its consumer-is-in-error clause; item 4 is § 5's bounded close.
- **AI-20.4**: the guard pins the carrier § 3 chose. Its bite proof includes "changing the carrier", which is meaningful only because a carrier was decided.
- **Not inherited:** the declaration itself, and the optional-capability mechanism (AI-03).

### AI-21 — the scripted fake provider

- Every rule in § 9 binds the fake, and `V-PRV-10`'s contract-faithfulness clause makes that binding rather than advisory: a fake that closes cleanly where the real contract drops events teaches Layer 2 the wrong physics.
- **AI-21.4** can script a producer that outruns an unread consumer because § 6 declares a finite capacity — a fake cannot deterministically exercise a saturation path that no contract bounds.
- **AI-21.5**'s cancellation fidelity is § 5 plus § 6: bounded close, late events dropped, no terminal on the saturated path.
- **AI-21.3**'s scripted terminal error is § 7's mid-stream path.

### AI-22 — the stream test kit

- **AI-22.5** receives the delegation § 3 made: a `V-STR-22` carrier view, ergonomic, and **never a second contract** — with the pin (AI-20.4's guard passes unmodified) as the mechanical form of that clause. AI-22.5's note about inverting if iterators had won does not apply; channels won.
- **AI-22.1**'s timeout-safe drain is what makes § 5's "bounded time" cheap to assert.
- **AI-22.4** chooses the leak-detection mechanism; § 5 tells it where leak assertions belong — the abandoned-then-**cancelled** path, which is testable, and not the abandoned-never-cancelled path, which is not.

### Further downstream, stated because each has a dependency on a specific line here

| Milestone | What it receives |
| --- | --- |
| **AI-03** | Cancellation and typed failure delivery are required capabilities whose observable shape is already fixed by §§ 5 and 7, so the matrix can mark them without re-deciding them |
| **AI-19.5** | § 7 entire: one vocabulary, two delivery paths, and the handover boundary that makes its third test item satisfiable |
| **AI-33** | §§ 5 and 6 proven against a real transport instead of a fake |
| **AI-34.1** | § 6's starting capacity of **64**, its three measurements, the direction each result implies, and the tie-break rule — an experiment, not an opinion. Constant-versus-configurable is left open for it |
| **AI-35** | A `V-FAIL-09` unpolluted by delivery, which is what `V-FAIL-15`'s never-retry-partial-output clause stands on |
| **AI-40.3** | Statements 3 and 4 of § 9, restated at the freeze |
| **doc 0003 AG-01.1** | A carrier decision argued rather than inherited, and ownership and cancellation rules its item 4 can mirror instead of re-deriving |

---

## 11. Standing rules this decision establishes

1. **The container is settled; the contents are not.** This artifact decides how a stream behaves as a container — carrier, ownership, cancellation, buffering, failure delivery. What the container carries is AI-14's, and how a failure is classified is AI-19's. A downstream milestone that finds itself deciding a container property is proposing an amendment to this artifact, not exercising a judgement call.
2. **Exactly one loss path exists.** Any other loss is a defect, not a variant. AI-34.3 tests for the absence of a second one.
3. **An untestable obligation is marked as such and lives in the package contract.** It is never given a test that proves something weaker, because the weaker property then replaces it. Defect **C3** is what that substitution looks like when it ships.
4. **A reopened question is an amendment, in the same pull request.** If a later milestone's implementation disproves a decision here, doc 0002's living-graph clause applies: revert to green, record the finding as an amendment to this artifact with a dated blockquote, and land the amendment in the pull request that resumes work. Superseded text is struck through and left visible, so citations from merged charters keep resolving.
5. **The capacity is a hypothesis until AI-34 measures it.** Citing 64 as a settled value is a misreading of § 6, and citing it after AI-34.1 has published a measurement is a stale citation.
6. **Layer 2 decides its own carrier.** doc 0003 AG-01.1 owns the agent-level decision. This artifact is an input to it and never a substitute — the symmetry is a recommendation with reasons, not an inheritance.

---

## 12. Closing-checklist verification

AI-02.1's five items, each against this artifact.

| # | Closing-checklist item | Where answered | Status |
| --- | --- | --- | --- |
| 1 | **Carrier:** receive-only channel versus range-over-func iterator, decided with rationale. If channels win, the iterator-ergonomics requirement is delegated to AI-22.5 and the decision says so. If iterators win, doc 0002's waves 2–5 gain amendment nodes | § 3 — receive-only channel, on four sourced grounds and none of them cost of change; the iterator case stated at full strength first; "both" named and rejected; delegation to AI-22.5 stated; no doc 0002 amendment nodes required, stated as a consequence | **answered** |
| 2 | **Ownership:** the producer creates the stream and closes it exactly once; nothing else closes it; the consumer never closes it. What "exactly once" means across completion, error and cancellation is **stated, not implied** | § 4 — one sender, one closing site, every exit path including unwinding, after the last send attempt; the three misreadings named; the non-closing parties enumerated | **answered** |
| 3 | **Cancellation:** caller owns a cancellable context; every send selects on it; cancellation closes within bounded time. Abandoning without cancelling is a documented contract violation that must appear in the package contract because it cannot be tested to termination (AI-40.3 restates it) | § 5 — three obligations each marked testable or structural; "bounded" defined by exclusion; the two legal endings enumerated so "violation" has a complement; the abandonment clause placed in § 9 statements 3 and 4 and attributed to AI-40.3 | **answered** |
| 4 | **Buffering:** a bounded buffer with a decided starting capacity, revisited with measurements at AI-34; the sanctioned loss path stated here and proven at AI-20.3 | § 6 — bounded, **starting capacity 64**, justified at both ends of the range and against concurrency; three measurements with directions and a tie-break rule for AI-34.1; the loss path stated in full with its two corollaries; constant-versus-configurable deferred | **answered** |
| 5 | **Failure delivery:** what a caller observes when the request never becomes a stream versus when a stream dies mid-flight — the split AI-19.5 implements as one vocabulary over two delivery paths | § 7 — the boundary is handover; "the first event" named and rejected with the case that separates them; `V-FAIL-11` and `V-FAIL-12` each stated in terms of what a caller observes; the two-axis grid keeping delivery orthogonal to `V-FAIL-09`; ordinary cancellation answered | **answered** |

**Register amendment.** Two nouns were appended to AI-01's register by this same change, under its § 9 rule 2: `V-STR-22` **carrier view** and `V-STR-23` **backpressure**, both owned by AI-02, both taking the next free `V-STR` ordinals, under a dated amendment blockquote. No existing row was renumbered, reworded or removed. Neither term is defined in this change's artifacts; both are cited by identifier.

**Node status.** AI-02.1 closes on merge of this artifact. Per doc 0002's node grammar, a `[decision]` leaf produces no production code and closes when "the decision artifact answers every listed question and is merged." No `make test` gate applies; there is nothing in `backend/agent/` that this change touches.

**Milestone acceptance, restated from doc 0002 and checked:** *"AI-14 … AI-20 can be written without reopening any of these five questions; AI-34's buffer measurement has a stated starting point to confirm or change."* § 10 states the inheritance for AI-14, AI-20, AI-21 and AI-22 node by node, and § 6 gives AI-34.1 a starting capacity of 64 with the measurements that would move it and in which direction.

**Unblocked by this decision:** AI-03 (`cachicamas-ai-minimum-capabilities`), AI-14 (`cachicamas-ai-event-envelope`), AI-20 (`cachicamas-ai-model-provider`), AI-21 (`cachicamas-ai-fake-provider`), AI-22 (`cachicamas-ai-stream-testkit`) — and, through them, AI-19, AI-23, AI-33, AI-34, AI-35 and AI-40.
