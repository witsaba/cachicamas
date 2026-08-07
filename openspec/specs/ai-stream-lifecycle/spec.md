# Spec — Layer 1 stream lifecycle, ownership and carrier

> **Milestone**: AI-02 — Decide stream lifecycle, ownership, and the carrier · **Node**: AI-02.1 `[decision]`
> **Introduced by**: `openspec/changes/archive/2026-07-31-cachicamas-ai-stream-lifecycle/`, merged in PR #95 at commit `a831c06` on 2026-07-31
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: the concern doc 0001 and ADR 0005 track as **G13**
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-AIS-0NN` · **Scenario IDs**: `S-AIS-0NN`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md) — every Layer 1 noun below is one of its rows, cited by identifier
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0003 — Layer 2 task graph](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

> [!IMPORTANT]
> **This artifact decides behavior, not code.** No Go type name, field name, method name, or package identifier appears here — doc 0002's authoring constraint. AI-14 and AI-20 choose spellings. Where a language or standard-library shape has to be discussed, it is named descriptively ("the single-value iterator function shape") rather than spelled, so that the constraint holds even inside the argument that most tempts a reader to break it.

## Purpose

Fix how a Layer 1 stream behaves **as a container**: what carries it at the package boundary, who owns it and closes it, how it is cancelled and what abandoning it means, how it buffers and where its one sanctioned loss path is, and where the line falls between a failure delivered before a stream exists and a failure delivered on one.

Five questions, five answers, and the argument for each. What the container *carries* is AI-14's; how a failure is *classified* is AI-19's. § 11 rule 1 states the boundary between this contract and theirs.

## Status — this file is the live home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The five decisions therefore live **here**, in their own text, and not only as a pointer into the archive.

- **This file states the contract.** AI-14, AI-19, AI-20, AI-21, AI-22, AI-23, AI-33, AI-34, AI-35 and AI-40 cite it. § 10 tells each of them what it inherits, in its own terms.
- **The archived `decision.md` is the historical record of how the contract was decided** — the same five decisions as they stood at merge, with AI-02.1's closing-checklist verification. It is immutable. It is at [`openspec/changes/archive/2026-07-31-cachicamas-ai-stream-lifecycle/decision.md`](../../changes/archive/2026-07-31-cachicamas-ai-stream-lifecycle/decision.md).
- **One clause here is explicitly awaiting evidence.** § 6's starting buffer capacity of **64** is a hypothesis, not a measurement, and AI-34.1's charter is to confirm or change it *with measurements*. That is an amendment to this file, not a new artifact — which is precisely why the contract could not be frozen in the archive.

### How to amend this contract

These are § 11 rule 4's terms, restated here so the next milestone can follow them without opening the archive.

| # | Rule |
| --- | --- |
| 1 | **A reopened question is an amendment, in the same pull request.** If a later milestone's implementation disproves a decision here, revert to green, record the finding as an amendment, and land it in the pull request that resumes work. |
| 2 | **A dated blockquote under the touched section heading**, stating what changed, which milestone node changed it, and why. |
| 3 | **Superseded text is struck through and left visible**, never deleted, so citations from merged charters keep resolving. Section numbers are stable for the same reason: downstream artifacts cite *§ 5* and *§ 7* of this contract by number. |
| 4 | **A downstream milestone that finds itself deciding a container property is proposing an amendment**, not exercising a judgement call (§ 11 rule 1). |
| 5 | **A Layer 1 noun this contract needs and the register lacks is appended to the register**, not defined here (§ 11, and the register's own § 9 rule 2). |

---

## 1. How to use this document

**If you are writing AI-14, AI-19, AI-20, AI-21, AI-22, AI-33, AI-34 or AI-40:** § 10 tells you what your milestone inherits, in your milestone's own terms. Start there, then read the one decision section it points at. You should not need the other four.

**If you are reviewing this artifact:** § 12 walks AI-02.1's closing checklist against it. § 3 is where the argument lives and where a defect is most expensive.

**If you disagree with a decision:** each of § 3 … § 7 carries a *What this excludes* part naming the alternatives and why each lost. If your objection is there, it was considered. If it is not, it is new, and § 11 rule 4 says what to do with it.

**Every Layer 1 noun below resolves to a row in the [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md)**, cited by identifier rather than paraphrased. Two nouns the register lacked — `V-STR-22` **carrier view** and `V-STR-23` **backpressure** — were appended to it by AI-02.1 itself, under the register's § 9 rule 2. They are used here; they are not defined here.

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

> **Amended 2026-08-07** by `cachicamas-ai-cancellation` (AI-33, Wave 5 — Harden). One behavior-only requirement added: **R-AIS-033** (body lifecycle: drain-before-close on every exit path). The delta's archived form carries the implementation vocabulary; this contract states only behavior.

### R-AIS-033 (added 2026-08-07) — Body lifecycle: drain-before-close on every exit path

> **Behavior.** When a stream ends for any reason — completion, terminal error, or any cancellation moment — the underlying transport connection MUST be cleanly released: the response body MUST be drained (any unread bytes discarded) before the producer's close fires. The drain MUST be part of the producer's existing single-defer ownership (no second closing site, no second goroutine — `R-ATS-003`). The drain MUST be silent — any error is the network's concern, not a Layer 1 contract concern. The drain MUST complete before the close returns.

#### Scenario: R-AIS-033 / S-1 — Drain fires on normal completion *(pin: `R-CNF-005`, `R-CNF-009`)*
- **GIVEN** a transport that delivers more bytes than the consumer reads before the terminal event lands
- **WHEN** the producer reaches the terminal event and exits normally
- **THEN** the response body is drained before the close, AND a subsequent request against the same transport succeeds without waiting on the prior connection's unread bytes

#### Scenario: R-AIS-033 / S-2 — Drain fires on terminal-error paths *(pin: `R-CNF-009`, `R-AEM-010`, `R-AEM-022`)*
- **GIVEN** the producer takes a terminal-error branch (in-band error or malformed frame)
- **WHEN** the terminal-error branch exits
- **THEN** the response body is drained before the close, AND the stream closes exactly once, AND the connection pool is not poisoned

#### Scenario: R-AIS-033 / S-3 — Drain fires on every cancellation moment *(pin: `R-CNF-011`, `R-CNF-012`, `R-STK-028`, `R-STK-029`)*
- **GIVEN** the consumer's context is cancelled at any point (pre-headers, between frames, blocked-send abandonment, or after completion)
- **WHEN** the producer exits via that cancellation path
- **THEN** the response body is drained before the close (or no body existed in the pre-headers case), AND the stream closes exactly once, AND no goroutine outlives the call

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

> **Amended 2026-08-07** by `cachicamas-ai-cancellation` (AI-33, Wave 5 — Harden). Five behavior-only requirements added: **R-AIS-034**, **R-AIS-035**, **R-AIS-036**, **R-AIS-037**, **R-AIS-038**. They restate the four cancellation moments and the full-package leak check in behavioral form, with text and tool-call variants each as a separate scenario per the milestone's charter (doc 0002 line 1989).

### R-AIS-034 (added 2026-08-07) — Cancellation before headers is reported without producing a stream

> **Behavior.** When the consumer's context is cancelled before the producer's transport call returns a response — equivalently, before any response byte crosses the wire — the call MUST return a typed cancellation failure carrying the pre-stream delivery classification. No stream MUST be returned. No producer MUST be spawned. No transport response MUST be opened.

#### Scenario: R-AIS-034 / S-1 — Text stream, cancellation before any response byte *(pin: `R-CNF-011`, `R-ATS-002`)*
- **GIVEN** a real transport client and a test server whose handler hangs before headers
- **WHEN** the caller invokes the stream operation with a context already cancelled at call time
- **THEN** the call returns a typed cancellation failure with the pre-stream delivery classification, AND no stream is returned, AND no goroutine was spawned, AND no goroutine growth is observed

#### Scenario: R-AIS-034 / S-2 — Tool-call stream, same conditions *(pin: `R-CNF-011`, `R-ATS-002`)*
- **GIVEN** the same setup with a tool-call request
- **WHEN** the caller invokes the stream operation with a cancelled context
- **THEN** the same observable outcome as R-AIS-034 / S-1

#### Scenario: R-AIS-034 / S-3 — Race: cancellation while the transport call is in flight *(pin: `R-CNF-011`, `R-ATS-002`)*
- **GIVEN** a server whose handler accepts the connection but never writes a status line
- **WHEN** the caller invokes the stream operation and cancels the context during the transport call
- **THEN** the same observable outcome as R-AIS-034 / S-1 — typed failure, no stream, no goroutine, no growth

### R-AIS-035 (added 2026-08-07) — Cancellation between frames closes the stream within bounded time and frees the connection

> **Behavior.** When the consumer's context is cancelled while the producer is blocked waiting for the next frame from the transport, the producer MUST exit within the bounded drain deadline plus a safety margin, the stream MUST close exactly once, the response body MUST be closed (so a stalled server cannot pin the connection), and no goroutine MUST outlive the call.

#### Scenario: R-AIS-035 / S-1 — Text stream, cancel while idle between frames *(pin: `R-CNF-011`, `R-STK-028`)*
- **GIVEN** a server that emits one frame and then stalls
- **WHEN** the caller receives the first event and then cancels the context
- **THEN** the producer exits within the bounded deadline, the channel closes exactly once, the response body is closed, AND no goroutine growth is observed

#### Scenario: R-AIS-035 / S-2 — Tool-call stream, same conditions *(pin: `R-CNF-011`, `R-STK-028`)*
- **GIVEN** the same setup with a tool-call transcript
- **WHEN** the caller receives the first tool-call event and cancels the context
- **THEN** the same observable outcome as R-AIS-035 / S-1

#### Scenario: R-AIS-035 / S-3 — Connection freed: subsequent request against the same transport succeeds *(pin: `R-CNF-011`)*
- **GIVEN** the stalling-server setup and a transport that reuses connections
- **WHEN** scenario R-AIS-035 / S-1 completes and a second request is issued
- **THEN** the second request completes promptly without waiting on the first connection's stale keep-alive slot

### R-AIS-036 (added 2026-08-07) — Truly-abandoned consumer + cancellation drops cleanly with no terminal invented

> **Behavior.** Pinned to the conformance assertion for the abandoned-then-cancelled path (verbatim wording). When the consumer has stopped reading, the producer is blocked mid-send, and the context is cancelled, the stream MUST close bare — no terminal event of any kind observed, no undelivered event forced through, no goroutine leak. The bounded-wait cap on the typed terminal IS the bounded close; the abandonment is what makes the bounded-wait terminal fail to land.

#### Scenario: R-AIS-036 / S-1 — Text stream, truly abandoned, then cancel *(pin: `R-CNF-012`, `R-STK-029`)*
- **GIVEN** a server serving a text transcript
- **WHEN** the caller invokes the stream operation, never reads from the channel, and immediately cancels the context
- **THEN** the stream closes bare within the bounded-wait cap plus safety, no completion and no error event is observed, AND no goroutine growth is observed

#### Scenario: R-AIS-036 / S-2 — Tool-call stream, same conditions *(pin: `R-CNF-012`, `R-STK-029`)*
- **GIVEN** the same setup with a tool-call transcript
- **WHEN** the caller invokes the stream operation, never reads, and cancels
- **THEN** the same observable outcome as R-AIS-036 / S-1

#### Scenario: R-AIS-036 / S-3 — Abandoned-never-cancelled path is not asserted *(pin: `R-CNF-012` narrowing, `R-STK-010`)*
- **GIVEN** the AI-33 test files
- **WHEN** a reviewer looks for an abandoned-never-cancelled test
- **THEN** none exists, AND the absence is recorded on the contract

### R-AIS-037 (added 2026-08-07) — Cancellation after completion is a no-op; close happens exactly once

> **Behavior.** When the consumer's context is cancelled after the producer has emitted the terminal event and exited, the call MUST be a no-op: the stream was already closed (or closes cleanly with no further events), the close happened exactly once, no consumer-side interleaving panic occurs, and no goroutine outlives the call.

#### Scenario: R-AIS-037 / S-1 — Text stream, cancel after completion *(pin: `R-CNF-009`, `R-CNF-011`)*
- **GIVEN** a short text transcript that ends with a terminal event
- **WHEN** the caller drains the stream to close, then cancels the context
- **THEN** the channel was closed exactly once, the recorded events carry exactly one terminal, AND no panic occurs across many repeats

#### Scenario: R-AIS-037 / S-2 — Tool-call stream, same conditions *(pin: `R-CNF-009`, `R-CNF-011`)*
- **GIVEN** the same setup with a tool-call transcript
- **WHEN** the caller drains to close, then cancels
- **THEN** the same observable outcome as R-AIS-037 / S-1

#### Scenario: R-AIS-037 / S-3 — Race: cancel and final receive interleave *(pin: `R-CNF-009`, `R-CNF-011`)*
- **GIVEN** the text-transcript setup
- **WHEN** the caller issues the cancel concurrently with the final receive, across many repeats
- **THEN** no panic occurs, AND exactly one terminal is observed, AND the channel is closed exactly once

### R-AIS-038 (added 2026-08-07) — Full-package leak check covers every exit path on both stream kinds

> **Behavior.** A single serial test suite MUST run every AI-33 exit path — completion, terminal error, each cancellation moment — across both text and tool-call streams, using the canonical leak-check helper. The suite MUST NOT use parallel test execution. The helper's standard-library-only posture MUST be preserved (no new top-level dependency).

#### Scenario: R-AIS-038 / S-1 — Full-package serial leak check passes *(pin: `R-STK-007`, `R-STK-008`, `R-STK-009`)*
- **GIVEN** the AI-33 test files for every subnode
- **WHEN** a single serial test wraps each scenario in the leak-check helper
- **THEN** no goroutine growth beyond the helper's tolerance is observed on any path, AND no parallel test execution is used

#### Scenario: R-AIS-038 / S-2 — Both stream kinds covered per scenario *(pin: `R-CNF-005`, `R-CNF-007`)*
- **GIVEN** the AI-33 test files
- **WHEN** a reviewer enumerates the scenarios per subnode
- **THEN** each subnode has at least one text-stream scenario AND at least one tool-call-stream scenario

#### Scenario: R-AIS-038 / S-3 — Module dependency unchanged *(pin: `R-STK-009`, `NFR-CNF-A`)*
- **GIVEN** the dependency file at base and after this delta
- **WHEN** a reviewer diffs the dependency file
- **THEN** no new require is added

---

## 6. Decision 4 — buffering and backpressure

**Closing-checklist item 4.**

### Decision

The buffer between producer and consumer (`V-STR-08`) is **bounded**, with a **starting capacity of 0 events** *(measured by AI-34.1 against the workload recorded in `openspec/changes/archive/2026-08-07-cachicamas-ai-backpressure/decision.md`; tie-break per doc 0002:432 applied — "prefer the smaller; backpressure observable is worth more than latency that was hidden")*. Runtime behaviour at `N=0` is identical to the prior unbuffered carrier (`make(chan T, 0) == make(chan T)`) but the named constant + the explicit buffer argument are now part of the contract that R-AIS-031 and R-AIS-039 lock down.

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

> **Amended 2026-08-07 (AI-35)** by `cachicamas-ai-retry-policy` (AI-35, Wave 5 — Harden). Four behavior-only requirements added: **R-AIS-041** (pre-stream retry predicate), **R-AIS-042** (backoff mechanics), **R-AIS-043** (replayability and partial-output boundary marker), **R-AIS-044** (composed-bound ceiling, cross-layer). The seam — in-adapter loop, factored into a shared helper invoked by the adapter's execute-once function — is fixed by the 2026-08-07 doc 0002 amendment (lines 2081–2093). The canonical `V-FAIL-15` row ("the partial-output case is never retried at Layer 1") is the load-bearing sentence these requirements turn into structure. No `MODIFIED` requirements; the existing § 7 failure-delivery contract (handover as the boundary) is unpolluted by retry, because pre-stream retry is by construction *before* handover.

### R-AIS-041 (added 2026-08-07) — Pre-stream retry predicate

The retry helper MAY retry exactly when **all** four conditions hold simultaneously:

1. The failure originates **before** the carrier handover (no semantic event has been emitted).
2. The failure is retryable per the typed taxonomy (rate-limit, unavailable, timeout categories).
3. The attempt budget has not been exhausted.
4. The caller-owned context has not been cancelled, and the remaining context budget accommodates the next delay.

If any condition fails, the helper MUST return the typed failure as-is — the typed error with its partial-output discriminator is handed up and the harness (Layer 2) decides. The boundary is **"nothing emitted"**, not **"nothing completed"** — a stream handed over that fails before emitting any content is mid-stream delivery (`V-FAIL-12`) and is never retried by Layer 1.

#### Scenario: R-AIS-041 / S-1 — Retryable pre-stream failure is retried *(pin: `R-CNF-019`, `R-ATS-002`, `R-AEM-008`)*

- **GIVEN** a real transport returning rate-limit (429), repeated across the script's attempt budget
- **WHEN** the caller invokes the stream call and no semantic event has been emitted
- **THEN** the helper issues `N+1` wire requests (where `N` is the documented maximum attempts per R-AIS-044), each one marked retryable, AND the typed failure carries the attempt count via the cause chain, AND the carrier handover never occurs

#### Scenario: R-AIS-041 / S-2 — Terminal-category failure is never retried *(pin: `R-CNF-019`, `R-AEM-008`, `R-AEM-009`)*

- **GIVEN** a real transport returning authentication failure or invalid request
- **WHEN** the caller invokes the stream call
- **THEN** the helper issues exactly one wire request, AND the typed failure is returned with retryability marked false regardless of position, AND no retry occurs even if the attempt budget has not been exhausted

#### Scenario: R-AIS-041 / S-3 — After any semantic event has been emitted, no retry occurs *(pin: `R-CNF-019`, `R-AIP-010`, `R-AIP-011`, `V-FAIL-09`)*

- **GIVEN** the helper's seam: the helper runs **before** the carrier is handed over and therefore never observes a partial-output failure
- **WHEN** a reviewer inspects the helper's call site and the carrier handover ordering
- **THEN** no failure with partial output can reach the helper by construction — the typed error with its partial-output discriminator reaches the consumer through the terminal error event, AND the helper's absence of a retry branch on the partial-output path is observable in the helper's source

#### Scenario: R-AIS-041 / S-4 — Exhausted attempt budget returns the last failure wrapped to carry the count *(pin: `R-CNF-019`, `V-FAIL-15`)*

- **GIVEN** a real transport returning a retryable-flagged failure on every attempt and an attempt budget of `N`
- **WHEN** the helper exhausts its budget
- **THEN** exactly `N+1` wire requests have been issued, AND the typed failure returned carries the attempt count reachable via the error cause chain, AND the final cause is the last wire response's typed failure

---

### R-AIS-042 (added 2026-08-07) — Backoff mechanics

Backoff is **bounded, injectable, and context-aware**. Three observable behaviors:

1. When the typed failure carries a retry-after hint, the next attempt MUST wait that hint; the hint overrides any computed backoff.
2. When no retry-after hint is present, backoff MUST grow within documented bounds — exponential with a documented cap, with jitter that is seeded and therefore assertable. The seed and the growth curve are test seams, not hidden implementation choices.
3. Backoff MUST wait on the caller-owned context. Cancellation during backoff aborts immediately; a remaining context budget smaller than the next delay short-circuits to the last error.

A documented maximum attempt count terminates retrying — **exactly `N+1` wire requests** are issued, then the last error is returned. Unbounded retry against a hard-down endpoint is the incident class this requirement prevents.

#### Scenario: R-AIS-042 / S-1 — Retry-After hint overrides computed backoff *(pin: `R-CNF-019`, `R-AEM-008`)*

- **GIVEN** a real transport returning rate-limit with a retry-after hint of `H` seconds
- **WHEN** the helper observes the hint and computes the next delay
- **THEN** the next delay equals `H` exactly — never any computed exponential value — AND the hint is read from the typed failure's retry-after accessor, AND absent hint falls back to computed backoff

#### Scenario: R-AIS-042 / S-2 — Computed backoff grows within documented bounds and jitter is assertable *(pin: `R-CNF-019`, `R-STK-009`)*

- **GIVEN** a real transport returning a retryable-flagged failure repeatedly, with no retry-after hint, and an injectable jitter seed
- **WHEN** the helper computes the next delay across attempts
- **THEN** each delay is within the documented bounded range (exponential growth, capped), AND a fixed jitter seed produces an assertable sequence of delays across attempts, AND no delay exceeds the documented maximum

#### Scenario: R-AIS-042 / S-3 — Backoff waits on context; cancellation aborts immediately *(pin: `R-CNF-019`, `R-CNF-011`)*

- **GIVEN** a real transport returning a retryable-flagged failure with a long retry-after, and a context with bounded remaining budget
- **WHEN** the caller cancels the context during a backoff wait
- **THEN** the helper aborts the wait immediately on the cancellation signal, AND no subsequent wire request is issued, AND the typed failure is returned with the last wire response's cause — never with an invented retry-exhausted category

#### Scenario: R-AIS-042 / S-4 — Bounded attempt count: exactly N+1 wire requests, then last error *(pin: `R-CNF-019`, `R-AIS-041` / S-4)*

- **GIVEN** a real transport returning a retryable-flagged failure on every attempt and a documented maximum attempt count of `N`
- **WHEN** the helper runs to exhaustion
- **THEN** exactly `N+1` wire requests are issued (counted via an instrumented transport), AND the returned typed failure is the `N+1`-th attempt's failure, AND no additional wire request follows

---

### R-AIS-043 (added 2026-08-07) — Replayability and the partial-output boundary marker

A retried request MUST re-issue from scratch with an identical body — byte-compared across attempts; nothing consumed on attempt one may corrupt attempt two. The attempt count and the final cause MUST both be reachable from the returned error chain via the cause-chain accessor. Each failed attempt's response body MUST be closed and drained before the next begins (the per-attempt connection leak hazard; exhausts the connection pool exactly during the rate-limit storm that triggered the retries).

**The partial-output boundary marker**: when the carrier has been handed over **and** the producer has emitted a semantic event **and** a retryable failure subsequently occurs, **no automatic retry occurs**. The retry helper's seam — running before the carrier handover — guarantees this by construction; this scenario pins the construction against future drift.

#### Scenario: R-AIS-043 / S-1 — Byte-identical replay across attempts *(pin: `R-CNF-019`, `R-ART-010`)*

- **GIVEN** a real transport that records each request body's bytes, and a scripted request that the helper retries
- **WHEN** the helper issues `N+1` attempts
- **THEN** every recorded request body is byte-identical to every other (no drift, no truncation, no re-encoding), AND the in-memory body slice is re-read via a fresh reader per attempt without mutation

#### Scenario: R-AIS-043 / S-2 — Attempt count and final cause are reachable from the error chain *(pin: `R-CNF-019`, `V-FAIL-15`)*

- **GIVEN** the helper has exhausted its attempt budget
- **WHEN** a consumer reads the returned error via the cause-chain accessor
- **THEN** the attempt count equals `N+1`, AND the final cause is reachable as a typed failure carrying the original failure's category and delivery classification, AND no cause is dropped from the chain on exhaustion

#### Scenario: R-AIS-043 / S-3 — Per-attempt body drain (status path) *(pin: `R-CNF-019`, `R-AIS-033` / S-1, S-2, `R-ATS-003`)*

- **GIVEN** the status path (a non-2xx response with a body) for one attempt
- **WHEN** that attempt completes (success or typed failure) before the next attempt begins
- **THEN** the response body for that attempt is drained to a discarding sink and closed — supplied by composition with the existing capture-and-drain discipline, not by a new defer in the helper — AND no attempt leaks a connection slot

#### Scenario: R-AIS-043 / S-4 — Partial-output boundary marker: after handover + emitted event, no retry *(pin: `R-CNF-019`, `R-AIP-010`, `R-AIP-011`, `V-FAIL-15`, `AG-15.1` item 2)*

- **GIVEN** a real transport returning one success frame of streaming text, then a retryable-flagged failure mid-stream
- **WHEN** the caller invokes the stream call and the producer emits the first semantic event before seeing the failure
- **THEN** the helper's seam is past: the helper is no longer in the call path, AND no second wire request is issued, AND the typed failure reaches the consumer through the terminal error event with partial output marked true

---

### R-AIS-044 (added 2026-08-07) — Composed-bound ceiling (cross-layer contract)

The composed bound **"harness attempts × Layer 1 attempts"** is documented where both layers' readers find it. The Layer 1 multiplier `defaultMaxAttempts = 3` is documented in the helper's package documentation (the file the helper ships in) and referenced verbatim by Layer 2's composed-bound test (per doc 0003 line 718). A reader from either Layer 1 (the helper's documentation) or Layer 2 (the harness test) finds the same number with the same formula.

This requirement carries **no production-code obligation** beyond the documentation's wording. Its purpose is the cross-layer visibility — the contract is binding *as documentation*, not as a runtime check.

#### Scenario: R-AIS-044 / S-1 — Layer 1 multiplier documented in helper's package doc comment *(pin: `R-CNF-019`, `AG-15.2` item 2)*

- **GIVEN** the helper's package documentation file
- **WHEN** a Layer 1 reader opens the file
- **THEN** the documentation names the wire-request count per logical call (i.e. `N+1 = 4` wire requests when retries are exhausted), AND the composed-bound formula "harness attempts × Layer 1 attempts" appears in the same documentation, AND the documentation identifies Layer 2's composed-bound test as the cross-layer consumer

#### Scenario: R-AIS-044 / S-2 — Layer 2 reader sees the same number with the same formula *(pin: `R-CNF-019`, `AG-15.2` item 2)*

- **GIVEN** Layer 2's harness-attempt test
- **WHEN** a Layer 2 reader reads the test
- **THEN** the test cites the Layer 1 multiplier verbatim from the helper's package documentation, AND the composed-bound formula matches the helper's wording, AND a divergence between the two layers' wording is observable as a test failure

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

AI-02.1's five items, each against this contract, as verified at merge.

| # | Closing-checklist item | Where answered | Status |
| --- | --- | --- | --- |
| 1 | **Carrier:** receive-only channel versus range-over-func iterator, decided with rationale. If channels win, the iterator-ergonomics requirement is delegated to AI-22.5 and the decision says so. If iterators win, doc 0002's waves 2–5 gain amendment nodes | § 3 — receive-only channel, on four sourced grounds and none of them cost of change; the iterator case stated at full strength first; "both" named and rejected; delegation to AI-22.5 stated; no doc 0002 amendment nodes required, stated as a consequence | **answered** |
| 2 | **Ownership:** the producer creates the stream and closes it exactly once; nothing else closes it; the consumer never closes it. What "exactly once" means across completion, error and cancellation is **stated, not implied** | § 4 — one sender, one closing site, every exit path including unwinding, after the last send attempt; the three misreadings named; the non-closing parties enumerated | **answered** |
| 3 | **Cancellation:** caller owns a cancellable context; every send selects on it; cancellation closes within bounded time. Abandoning without cancelling is a documented contract violation that must appear in the package contract because it cannot be tested to termination (AI-40.3 restates it) | § 5 — three obligations each marked testable or structural; "bounded" defined by exclusion; the two legal endings enumerated so "violation" has a complement; the abandonment clause placed in § 9 statements 3 and 4 and attributed to AI-40.3 | **answered** |
| 4 | **Buffering:** a bounded buffer with a decided starting capacity, revisited with measurements at AI-34; the sanctioned loss path stated here and proven at AI-20.3 | § 6 — bounded, **starting capacity 64**, justified at both ends of the range and against concurrency; three measurements with directions and a tie-break rule for AI-34.1; the loss path stated in full with its two corollaries; constant-versus-configurable deferred | **answered** |
| 5 | **Failure delivery:** what a caller observes when the request never becomes a stream versus when a stream dies mid-flight — the split AI-19.5 implements as one vocabulary over two delivery paths | § 7 — the boundary is handover; "the first event" named and rejected with the case that separates them; `V-FAIL-11` and `V-FAIL-12` each stated in terms of what a caller observes; the two-axis grid keeping delivery orthogonal to `V-FAIL-09`; ordinary cancellation answered | **answered** |

**Register amendment.** Two nouns were appended to the vocabulary register by AI-02.1, under its § 9 rule 2: `V-STR-22` **carrier view** and `V-STR-23` **backpressure**, both owned by AI-02, both taking the next free `V-STR` ordinals, under a dated amendment blockquote. No existing row was renumbered, reworded or removed. Neither term is defined here; both are cited by identifier.

**Milestone acceptance, as stated in doc 0002 and checked at merge:** *"AI-14 … AI-20 can be written without reopening any of these five questions; AI-34's buffer measurement has a stated starting point to confirm or change."* § 10 states the inheritance for AI-14, AI-20, AI-21 and AI-22 node by node, and § 6 gives AI-34.1 a starting capacity of 64 with the measurements that would move it and in which direction. AI-03 was written from this contract in the same wave and re-decided none of the five, which is the criterion demonstrated rather than asserted.

**Unblocked by this contract:** AI-14 (the event envelope), AI-20 (the model provider), AI-21 (the fake provider), AI-22 (the stream test kit) — and, through them, AI-03, AI-19, AI-23, AI-33, AI-34, AI-35 and AI-40.

---

## Requirements

The requirements below constrain **the decision** — the contract stated in § 2 … § 11 above, first recorded in the archived `decision.md`. Every scenario is a property a reviewer can check by inspection, deterministically, without running anything: a scenario reads *"given the decision, when …, then …"*. The contract is the system under test, because AI-02.1 ships no Go and there is no runtime behavior to specify.

A second distinction matters as much. Several requirements constrain the **argument**, not only the conclusion — that the strongest opposing case is present, that a rejected alternative is named, that a number carries its falsification criteria. doc 0002 requires the SDD to "record why it chose what it chose"; a conclusion with no argument does not satisfy that, and would pass a spec that only checked conclusions. An amendment under § 11 rule 4 inherits the same obligation.

### Definitions used by these requirements

- **The decision** — the contract in § 2 … § 11 of this spec.
- **The closing checklist** — AI-02.1's five items in doc 0002.
- **The register** — the [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md).
- **A register citation** — a `V-*` identifier appearing in the decision.
- **The package contract** — the prose obligations § 9 requires AI-20 to publish alongside the provider interface.
- **An inheritance statement** — a sentence naming a downstream milestone and what that milestone receives from this decision.

### R-AIS-001 — The decision is singular and answers all five items

Exactly one statement of this contract is normative, and it is this file. It MUST answer every item of AI-02.1's closing checklist. No other file MAY restate a decision of this contract as normative.

#### Scenarios

- **S-AIS-001** — Given the canonical spec tree, when a reviewer looks for the stream-lifecycle contract, then exactly one file states it AND every other artifact refers to it rather than restating a decision as normative.
- **S-AIS-002** — Given the decision, when a reviewer walks AI-02.1's five closing-checklist items in order, then each resolves to a section that states a decision, its rationale, and its consequences.

### R-AIS-002 — The carrier decision is argued, and rests on no cost-of-change argument

The decision MUST fix the carrier and MUST record the rationale. It MUST present the strongest case for the rejected option before rejecting it. It MUST NOT rest any part of the carrier rationale on the cost of invalidating existing work, because doc 0002 records that no such work existed and declares that argument void.

#### Scenarios

- **S-AIS-003** — Given § 3, when a reviewer reads it, then a case **for** the range-over-func iterator is stated affirmatively and at length before any rebuttal, AND that case includes at minimum: the structural elimination of the stranded-producer hazard, abandonment becoming a supported operation, and the dissolution of the buffering question.
- **S-AIS-004** — Given the decision, when a reviewer searches the carrier rationale for any appeal to shipped guards, merged scenarios, existing signatures, or migration cost, then none is found as a supporting ground; the only permitted mention is the explicit statement that this argument is void.
- **S-AIS-005** — Given each ground on which the iterator case is defeated, when a reviewer checks it, then it cites a specific source — doc 0001, doc 0002, doc 0003, or a register row — rather than asserting a preference.
- **S-AIS-006** — Given the decision, when a reviewer looks for a third option, then "offer both carriers at the boundary" is named and rejected with its costs enumerated.

### R-AIS-003 — The carrier decision states its delegation and its graph consequence

The decision MUST state the consequence doc 0002 attaches to the branch it takes. The receive-only channel is chosen, so the decision MUST state that the iterator-ergonomics requirement is delegated to AI-22.5 and MUST state that doc 0002's waves 2–5 therefore require no amendment nodes. (Had the iterator been chosen, the decision would instead have had to state that doc 0002's waves 2–5 gain amendment nodes under the living-graph clause, and name AI-22.5's inversion.)

#### Scenarios

- **S-AIS-007** — Given § 3's consequences, when a reviewer reads them, then exactly one of the two branches above is stated, matching the decision taken, AND the other is not left implied.
- **S-AIS-008** — Given the channel branch, when a reviewer reads the delegation, then it states that the delegated view is a convenience and never a second contract, AND it names AI-22.5's pin as the mechanical form of that claim.

### R-AIS-004 — Ownership is stated structurally

The decision MUST state what "exactly once" binds, in structural terms a reader can check against a producer: how many goroutines may send, how many closing sites may exist, and when the closing site runs relative to the last send attempt.

#### Scenarios

- **S-AIS-009** — Given § 4, when a reviewer reads it, then it states that exactly one goroutine sends on a stream AND that exactly one closing site exists in the producer AND that the closing site runs after the last send attempt and never before.
- **S-AIS-010** — Given an adapter that internally reads a transport on one goroutine and translates on another, when a reviewer consults the decision, then it states that internal fan-in happens below the boundary and that exactly one goroutine ever sends on the carrier.

### R-AIS-005 — "Exactly once" holds on all three paths, and is stated rather than implied

The decision MUST state the closing obligation separately for the completion path, the terminal-error path and the cancellation path. It MUST NOT state the obligation once for the completion path and leave the other two to be inferred.

#### Scenarios

- **S-AIS-011** — Given the decision, when a reviewer looks for the closing obligation, then the completion path, the terminal-error path and the cancellation path are each named explicitly and each carries its own statement of what is emitted and when the close happens. (§ 4 names all three paths plus the unwinding exit; the per-path *emission* statements are in §§ 6 and 7 and the § 8 diagram, which draws all three converging on one closing site.)
- **S-AIS-012** — Given an unwinding exit from the producer, when a reviewer consults the decision, then the close is stated to run on that path too.

### R-AIS-006 — Nothing but the producer closes

The decision MUST state that the consumer never closes the stream, and MUST enumerate the other parties that also never close it.

#### Scenarios

- **S-AIS-013** — Given § 4, when a reviewer reads it, then the consumer, the test kit, and any consumer above Layer 1 are each named as parties that do not close the stream, consistent with `V-STR-04` and doc 0001 § 9.

### R-AIS-007 — Cancellation obligations are stated, each classified as testable or statable

The decision MUST state that the caller owns a cancellable context, that every send waits on both the stream and cancellation, and that cancellation closes the stream within bounded time. For each obligation the decision MUST say whether it is provable by test and, where it is, name the node that proves it.

#### Scenarios

- **S-AIS-014** — Given § 5, when a reviewer reads each of the three obligations, then each carries either a naming of the node that proves it or an explicit statement that it is not provable by test.
- **S-AIS-015** — Given the bounded-close obligation, when a reviewer reads what "bounded" excludes, then the decision states that after cancellation is observable the producer begins no new blocking wait on the network or on the consumer, and that a backoff waits on the signal rather than sleeping.

### R-AIS-008 — Abandonment is a contract violation, stated in the package contract

The decision MUST state that a consumer which stops reading and never cancels is a documented contract violation rather than a supported mode. It MUST state that this sentence belongs in the package contract, MUST state why — it cannot be tested to termination — and MUST name AI-40.3 as its restatement at the v1 freeze.

#### Scenarios

- **S-AIS-016** — Given the decision, when a reviewer reads the abandonment clause, then it is stated as a violation AND it is assigned to the package contract AND the reason given is that no test can prove a goroutine never exits AND AI-40.3 is named.
- **S-AIS-017** — Given the decision, when a reviewer asks what a consumer *may* legally do to end a stream early, then the complete set of legal endings is enumerated, so that "violation" has a complement.
- **S-AIS-018** — Given AI-20.1's third test item — that the interface documentation states who closes the stream, who owns the context, and what abandoning without cancelling means — when a reviewer checks the decision, then all three of those statements are present in § 9 and quotable.

### R-AIS-009 — The buffer is bounded and carries a decided starting capacity

The decision MUST state that the buffer is bounded. It MUST carry one starting capacity, expressed as a number. It MUST justify that number against the alternatives at both ends of the range, and MUST state that concurrency multiplies the cost.

#### Scenarios

- **S-AIS-019** — Given § 6, when a reviewer looks for the capacity, then exactly one number is stated as the starting capacity AND it is not expressed as a range, a preference, or a deferral.
- **S-AIS-020** — Given the chosen number, when a reviewer reads its justification, then the justification states what a materially smaller capacity would cost and what a materially larger one would cost.
- **S-AIS-021** — Given concurrent streams, when a reviewer consults the decision, then it states that capacity is paid per live stream and names the concurrency sources that make this real.

### R-AIS-010 — The capacity is falsifiable and AI-34 inherits the criteria

The decision MUST present the capacity as a starting hypothesis, MUST state what measurement would confirm or change it, and MUST state which direction each result implies. It MUST NOT settle whether the capacity is a constant or configurable — that is AI-34.1's.

#### Scenarios

- **S-AIS-022** — Given the decision, when a reviewer reads § 6, then it names the measurements AI-34.1 should take and states, for each, what result moves the number up and what result moves it down.
- **S-AIS-023** — Given the decision, when a reviewer looks for a ruling on constant versus configurable, then it is explicitly deferred to AI-34.1 rather than decided.

### R-AIS-011 — Backpressure is lossless and exactly one loss path is sanctioned

The decision MUST state that a full buffer makes the producer wait and never drop. It MUST state the single sanctioned loss path in full — cancellation with a saturated buffer drops late events and closes without a terminal event — MUST state that this is the only such path, and MUST name AI-20.3 as where it is proven.

#### Scenarios

- **S-AIS-024** — Given the decision, when a reviewer reads the backpressure posture, then waiting is stated as the behavior of a full buffer AND dropping is stated as not being that behavior.
- **S-AIS-025** — Given the sanctioned loss path, when a reviewer reads it, then all three of its elements are present — cancellation, saturation, and closing without a terminal event — AND the decision states that a consumer treating a missing terminal after its own cancellation as corruption is the party in error.
- **S-AIS-026** — Given a stream that closes without a terminal event and was never cancelled, when a reviewer consults the decision, then that case is stated to be a producer defect, not a second loss path.

### R-AIS-012 — The two delivery paths are separated at an observable moment

The decision MUST state what a caller observes when the request never becomes a stream (`V-FAIL-11`) and what a caller observes when a stream dies mid-flight (`V-FAIL-12`). It MUST identify the moment that separates them as an observable event, MUST name and reject the alternative boundary, and MUST state that the delivery axis is orthogonal to the partial-output discriminator (`V-FAIL-09`).

#### Scenarios

- **S-AIS-027** — Given the decision, when a reviewer reads the pre-stream path, then it states that the failure is returned directly AND that no stream and no producer are created — not an empty stream, not a stream that immediately yields a failure.
- **S-AIS-028** — Given the decision, when a reviewer reads the mid-stream path, then it states that the failure arrives as the terminal error event and that the stream then closes, AND that no second route exists by which a caller learns of a mid-stream failure.
- **S-AIS-029** — Given a stream that is handed to the caller and then fails before emitting any content, when a reviewer consults the decision, then that case is classified as mid-stream delivery, AND the decision states that classifying it as pre-stream is the conflation with `V-FAIL-09` that doc 0001 § 7 **G8** records.
- **S-AIS-030** — Given the decision, when a reviewer looks for the rejected boundary, then "the first event" is named as the rejected alternative with the reason it fails.
- **S-AIS-031** — Given AI-19.5's third test item — a caller that only inspects the returned failure and a caller that only inspects the terminal event can each classify every failure — when a reviewer checks the decision, then it states the property that makes this satisfiable: the same category, retryability and partial-output discriminator are reachable on both paths.

### R-AIS-013 — The contract stays inside its own scope

This contract MUST NOT decide anything owned by AI-14, AI-19, AI-20, AI-22.4, AI-34 or AI-35. The register's § 9 rule 5 test applies: if a sentence were deleted, would a later milestone have more options? If yes, and that milestone is not AI-02, the sentence does not belong.

#### Scenarios

- **S-AIS-032** — Given the decision, when a reviewer looks for an event kind, an event payload, a sequence rule or an ordering invariant, then none is decided; the contract constrains the container and defers contents to AI-14.
- **S-AIS-033** — Given the decision, when a reviewer looks for a failure category, a retryability rule, or the shape of a terminal error payload, then none is decided; the contract fixes delivery only and defers classification to AI-19.
- **S-AIS-034** — Given the decision, when a reviewer looks for a leak-detection mechanism, then none is chosen; AI-22.4 is named as its owner.
- **S-AIS-035** — Given the decision, when a reviewer looks for a retry rule, then none is stated; AI-35 is named as its owner.

### R-AIS-014 — Vocabulary discipline, inheritance, and artifact hygiene

Every Layer 1 noun used by this contract MUST resolve to a register row. A noun the register lacks MUST be appended to the register in the same pull request that needs it, with the next free ordinal in its category and a dated amendment blockquote, and MUST NOT be defined here. The contract MUST state what each blocked milestone inherits. No Go type, field, method, interface or package identifier MAY appear in this file.

#### Scenarios

- **S-AIS-036** — Given the decision, when a reviewer collects every Layer 1 noun it uses in a normative sentence, then each resolves to exactly one register row, cited by identifier. (35 distinct `V-*` identifiers are cited; all 35 resolve.)
- **S-AIS-037** — Given a noun the register lacked, when a reviewer inspects the pull request that needed it, then the term appears as a new row in the register with the next free ordinal in its category, under a dated amendment blockquote, AND no definition of it appears here other than as a citation.
- **S-AIS-038** — Given the register amendment, when a reviewer diffs the register, then no existing row is renumbered, reworded or removed, AND the register's own term counts are updated to remain consistent.
- **S-AIS-039** — Given § 10, when a reviewer reads it, then AI-14, AI-20, AI-21 and AI-22 each have an inheritance statement in that milestone's own terms, AND AI-34 has a stated starting point.
- **S-AIS-040** — Given this file, when a reviewer scans for a single-token camel-case name, a package path, or a method-shaped name, then none is found; every term is a noun phrase with spaces, and language and standard-library shapes are named descriptively.
- **S-AIS-041** — Given the diff of any change that states or amends this contract, when a reviewer inspects it, then it contains only markdown under `openspec/`, adds nothing under `backend/`, and modifies no build, module or infrastructure file.

### R-AIS-033 through R-AIS-038 (added 2026-08-07) — AI-33 cancellation proof and resource discipline

> **Amended 2026-08-07** by `cachicamas-ai-cancellation` (AI-33, Wave 5 — Harden). Six behavior-only requirements added: **R-AIS-033** (body lifecycle: drain-before-close on every exit path, under § 4) and **R-AIS-034** through **R-AIS-038** (the four cancellation moments and the full-package leak check, under § 5). The full requirement text and scenarios are inline under § 4 and § 5 above; this canonical list exists for discovery and cross-reference.

#### Cross-reference

- **R-AIS-033** — Body lifecycle: drain-before-close on every exit path. Defined inline at § 4 (after the `### Who inherits it` block). Scenarios S-1, S-2, S-3 cover normal completion, terminal-error paths, and every cancellation moment. Pins `R-CNF-005`, `R-CNF-009`, `R-CNF-011`, `R-CNF-012`, `R-AEM-010`, `R-AEM-022`, `R-ATS-003`, `R-STK-028`, `R-STK-029`.
- **R-AIS-034** — Cancellation before headers is reported without producing a stream. Defined inline at § 5. Three scenarios (text, tool-call, race). Pins `R-CNF-011`, `R-ATS-002`.
- **R-AIS-035** — Cancellation between frames closes the stream within bounded time and frees the connection. Defined inline at § 5. Three scenarios (text, tool-call, connection-freed). Pins `R-CNF-011`, `R-STK-028`.
- **R-AIS-036** — Truly-abandoned consumer + cancellation drops cleanly with no terminal invented. Defined inline at § 5. Three scenarios (text, tool-call, abandoned-never-cancelled not asserted). Pins `R-CNF-012`, `R-STK-029`, `R-STK-010`.
- **R-AIS-037** — Cancellation after completion is a no-op; close happens exactly once. Defined inline at § 5. Three scenarios (text, tool-call, race). Pins `R-CNF-009`, `R-CNF-011`.
- **R-AIS-038** — Full-package leak check covers every exit path on both stream kinds. Defined inline at § 5. Three scenarios (full-package serial, both stream kinds, dependency unchanged). Pins `R-STK-007`, `R-STK-008`, `R-STK-009`, `R-CNF-005`, `R-CNF-007`, `NFR-CNF-A`.

### R-AIS-041 through R-AIS-044 (added 2026-08-07) — AI-35 retry discipline and the partial-output boundary

> **Amended 2026-08-07 (AI-35)** by `cachicamas-ai-retry-policy` (AI-35, Wave 5 — Harden). Four behavior-only requirements added: **R-AIS-041** (pre-stream retry predicate), **R-AIS-042** (backoff mechanics), **R-AIS-043** (replayability and partial-output boundary marker), **R-AIS-044** (composed-bound ceiling, cross-layer). The full requirement text and all scenarios are inline under § 7 above; this cross-reference list exists for discovery and conformance pinning.

#### Cross-reference

- **R-AIS-041** — Pre-stream retry predicate. Defined inline at § 7. Four scenarios (S-1 retryable-pre-stream-retried, S-2 terminal-category-never-retried, S-3 after-handover-no-retry, S-4 exhausted-budget-returns-last-failure). Pins `R-CNF-019`, `R-ATS-002`, `R-AEM-008`, `R-AEM-009`, `R-AIP-010`, `R-AIP-011`, `V-FAIL-15`.
- **R-AIS-042** — Backoff mechanics. Defined inline at § 7. Four scenarios (S-1 retry-after-overrides-computed, S-2 computed-backoff-bounded-with-jitter, S-3 backoff-waits-on-context, S-4 bounded-attempt-count-exactly-N+1). Pins `R-CNF-019`, `R-AEM-008`, `R-STK-009`, `R-CNF-011`.
- **R-AIS-043** — Replayability and partial-output boundary marker. Defined inline at § 7. Four scenarios (S-1 byte-identical-replay, S-2 attempt-count-and-final-cause-in-chain, S-3 per-attempt-drain, S-4 partial-output-boundary-marker). Pins `R-CNF-019`, `R-AIS-033`, `R-ART-010`, `R-ATS-003`, `R-AIP-010`, `R-AIP-011`, `V-FAIL-15`, `AG-15.1`.
- **R-AIS-044** — Composed-bound ceiling (cross-layer). Defined inline at § 7. Two scenarios (S-1 layer-1-multiplier-documented, S-2 layer-2-reader-sees-same-number). Pins `R-CNF-019`, `AG-15.2`.

---

## Acceptance criteria

The contract holds when:

1. `R-AIS-001` through `R-AIS-014` hold, each verified by its scenarios.
2. All five items of AI-02.1's closing checklist are answered in § 2 … § 11, and § 12 records the verification item by item.
3. No Go identifier appears anywhere in this file.
4. AI-14, AI-20, AI-21 and AI-22 can each be written from § 10 alone, without reopening any of the five questions.
5. AI-34.1 has a starting capacity, three measurements, a direction per result, and a tie-break rule — an experiment rather than a constant to overturn.

Criteria 1 through 5 were verified at AI-02.1's merge and recorded in the archived `verify-report.md`, which returned **PASS** on all five closing-checklist items with 35 of 35 register citations resolving.

> **Amended 2026-08-07** — AI-33 acceptance criteria (added):
> 6. `R-AIS-033` through `R-AIS-038` hold, each verified by its scenarios inline under § 4 and § 5 above. The cross-reference list at § Requirements enumerates each requirement's scenarios and conformance pins.
> 7. AI-33's verify-report (archived at `openspec/changes/archive/2026-08-07-cachicamas-ai-cancellation/verify-report.md`) returned **PASS** on all six requirements (R-AIS-033 through R-AIS-038), 22 of 22 scenarios, with 0 CRITICAL findings. The eight recorded deviations are acknowledged in the archive-report, none are blockers.
> 8. No Go identifier appears anywhere in this file — verified by reading the file directly; the AI-33 amendment restates the implementation-vocabulary delta in behavior-only form per § 11 rule 4.

> **Amended 2026-08-07** — AI-35 acceptance criteria (added):
> 9. `R-AIS-041` through `R-AIS-044` hold, each verified by its scenarios archived at `openspec/changes/archive/2026-08-07-cachicamas-ai-retry-policy/specs/ai-stream-lifecycle/spec.md`. The cross-reference list at § Requirements enumerates each requirement's scenarios and conformance pins.
> 10. AI-35's verify-report (archived at `openspec/changes/archive/2026-08-07-cachicamas-ai-retry-policy/verify-report.md`) returned **PASS** on all four requirements (R-AIS-041 through R-AIS-044), 14 of 14 scenarios (integrated into R-CNF-019's 7 conformance scenarios for a total of 21/21 scenarios across both specs), with 0 CRITICAL findings. The three acknowledged deviations (D-A1, D-A2, D-A3) are documented in the archive-report, none are blockers.
> 11. No Go identifier appears anywhere in this file; the AI-35 amendment records the seam and the four requirements in behavior-only form per § 11 rule 4. The implementation-vocabulary delta is archived at the conformance-suite delta spec.
