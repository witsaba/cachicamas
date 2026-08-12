# Design — Layer 2 event delivery and the observer model

> **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 · **Node**: AG-01.1 `[decision]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Inputs**: `explore.md`, `proposal.md`, `specs/agent-event-delivery/spec.md` (R-AGE-001 … R-AGE-019)
> **Output**: the structure `decision.md` implements, the worked cases that prove the five decisions survive contact, and the choices the spec forces where the proposal left an opening
> **Precedent**: the archived AI-02 design (`openspec/changes/archive/2026-07-31-cachicamas-ai-stream-lifecycle/design.md`) — same node type, same artifact kind, same discipline
> **Diagrams**: ASCII (project convention — the AI-02 design records it; every diagram labels roles and messages, never signatures)
> **Authoring constraint**: doc 0003's — no Go type, field, method or package identifier anywhere in this change

---

## 1. What is being designed

Not a delivery mechanism. A **document** — `decision.md` — that six later milestones (AG-04, AG-10, AG-13, AG-14, AG-19, AG-20) will read as the settled physics of how Layer 2 events travel, each consulting it once, for one question, with its own charter open and nothing else. The AI-02 design established the three properties that follow from that reading pattern, and they are inherited here unchanged: each decision answerable without the other four; consequences co-located with conclusions; the argument present, not merely the verdict.

What is new at Layer 2 is that the decisions are not equally *provable* on the page. The carrier is an argument; the backpressure narrowing and the observer mechanism are claims that a specific case cannot happen. For those, this design does the work a decision artifact cannot skip: it traces the case that would break each claim and shows where the trace terminates. Sections 3–7 carry those traces; `decision.md` restates them as decided.

## 2. The failure modes this design targets

### 2.1 The restated default

AI-02's own decision forbids closure by citation: "Layer 2 decides its own carrier … the symmetry is a recommendation with reasons, never a substitute." A carrier section whose only support is "Layer 1 decided the same" is indistinguishable on the page from one that argued — until wave 4, when someone asks *why* and the question genuinely reopens at doc 0001 § 3.2's triple cost. Countermeasure, inherited from the AI-02 design and made checkable by `S-AGE-001`: **every ground cites a Layer 2 source by identifier** — AG-10.3 for the multiplexing ground, AG-04.2 for the terminal ground — and the strongest-opposing-case rule applies: the rejected option stated affirmatively, at its strongest, before any rebuttal.

### 2.2 The naive inheritance

Layer 1's sanctioned loss path is correct at Layer 1 and **wrong at one of Layer 2's two boundaries**, for a reason Layer 1 cannot have: the harness's history (AG-12) is an independent record of the same facts the stream carries. The failure mode is copying a correct rule across a boundary where its premise fails. Countermeasure: the rule is stated *per boundary* (§ 4), and the worked case that breaks the naive copy is on the page — a reviewer who would inherit unchanged must first defeat the trace in § 4.2.

### 2.3 Convention posing as structure

Envelope invariant 3 ("observers are never synchronous on the streaming path") can be satisfied by a sentence or by a mechanism, and the two are indistinguishable until a consumer stalls in production. AG-01.1 item 3 demands mechanism; `S-AGE-010` makes it checkable as a trace property. Countermeasure: § 5 draws the failure a convention-based design permits, end to end, and then the mechanism under which every path from a stalled consumer terminates before reaching the producer.

## 3. The delivery topology

Two internal boundaries, one attachment point. Everything in later sections happens on this picture.

```
                              LAYER 2
  ┌───────────────────────────────────────────────────────────────┐
  │  Layer 1 stream (frozen contract; capacity 0, measured)       │
  │        │                                                      │
  │        ▼              LOOP-INTERNAL BOUNDARY                  │
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
                          per-consumer receive-only│              │
                          carriers, one each       ▼              ▼
                                             primary         session logger,
                                             consumer        cost meter, …
```

Roles: the **loop** is the sole producer and closer of each per-turn stream. The **harness** is the sole consumer of turn streams, the owner of history and the run brackets, and the sole producer onto the canonical run-scoped stream. The **distribution step** is the canonical stream's only receiver; per attached consumer it feeds one **lane**, whose **forwarding activity** privately owns that consumer's receive-only carrier and applies the full send discipline toward it.

**Where the four envelope invariants bind** (the invariants are doc 0001 § 4.3's; this table places them, it does not restate them):

| Invariant | Binds at | Delivery's obligation |
| --- | --- | --- |
| 1 — indexed deltas | The loop, where deltas originate | Delivery forwards payloads verbatim; no stage rewrites or accumulates on behalf of a consumer |
| 2 — explicit nesting | The parent harness's re-emission of a child's events onto its own canonical stream | The re-emission point is the only place a parent identifier can be attached; § 6's delegated scope |
| 3 — asynchronous observers | The observer attachment point | § 5's mechanism; the whole of `R-AGE-008` |
| 4 — typed errors and outcomes | The two terminal emitters: loop (turn-end outcome), harness (run-end outcome) | § 6's terminal discipline; delivery guarantees the terminal's *arrival* (§ 4), AG-04 defines its contents |

## 4. Backpressure, worked through

### 4.1 What "saturated" means, before any rule is stated

Layer 1's live capacity is **0** — measured by AI-34.1, the superseded 64 struck through in the contract. At capacity zero a buffer is saturated whenever no receiver is already waiting: **saturation is the ordinary condition at essentially every send**, not a load state. Any rule triggered by "cancellation with a saturated buffer" is therefore triggered by "cancellation", full stop, at the moment of any in-flight send. Every argument below reasons from that frequency. A reviewer who finds the artifact reasoning from "saturation is rare" has found a defect (`S-AGE-008`).

### 4.2 The case that breaks naive inheritance

```
t0  a tool executes; it writes a file — a real-world side effect
t1  the harness appends the result to history       (history now knows)
t2  the result's event is offered toward a consumer carrier;
    no receiver is waiting at that instant           (ordinary at capacity 0)
t3  the user interrupts; the run's cancellation fires

Layer 1's rule, copied:  cancellation + saturated ⇒ drop late events,
                         close without a terminal
Result of the copy:      the stream omits a side effect history holds,
                         and omits run-end itself — a consumer sees a
                         bare close and cannot even enumerate its loss
```

Layer 1's clause "a missing terminal after your own cancellation makes you the party in error" assigns blame; it does not tell the consumer *what* is missing — and at Layer 2 what is missing can be the only external record that a real-world effect happened. A session log built from this stream disagrees with reality, silently, after every interrupted run. That is the defeat of the naive copy, and it is why the harness-facing rule must be strictly narrower.

### 4.3 The two-boundary rule, and how the rules compose

| Boundary | Rule | What may be lost | What may never be lost |
| --- | --- | --- | --- |
| Loop-internal | Layer 1's, unchanged: on the harness's own cancellation of a turn with a send in flight, late turn events drop and the turn stream closes without its terminal | Turn events the harness had not yet received | Nothing more is promised — the harness is the only consumer here, and the loss fires only on its own cancellation |
| Harness-facing | Layer 1's mechanism **plus**: the loss path never discards an event describing a fact already committed to history; bounded wind-down finishes delivering everything history holds, as part of or immediately preceding run-end | Facts the harness never learned before cancellation: in-flight deltas of a turn cut short, progress of tools that never completed, everything of turns that never began | Any event describing a committed history fact, and run-end itself |

**History is the watershed that makes the two rules compose.** An event dropped at the loop-internal boundary was, by construction, never appended to history — so the harness-facing guarantee owes nothing about *that event*. There is no gap through which a committed fact can leak out of both rules, because "committed" is defined by the harness's own append, which happens strictly after the loop-internal receive.

**The composed worst case — cancelled mid-tool-result.** A turn is cancelled while a tool result is in flight at the loop-internal boundary and drops there. The tool's side effect is real, but history never saw the result. The harness's own history invariant (doc 0001 § 4.2) now fires: interruption synthesises results for orphaned calls before history closes. The synthesised result **is** a committed history fact, so the harness-facing rule requires its delivery before run-end. The guarantee is anchored on **history**, not on any particular event instance — the original event may drop; the fact may not.

### 4.4 What remains droppable, and the residual

The narrowing is not vacuous (`R-AGE-007`): the droppable set is exactly *state the harness never learned before cancellation* — and it is non-empty at every interruption, because a cancelled turn always has in-flight work that history will never hold. Removing the loss path entirely is unavailable at any price: a consumer that stops reading and never cancels would make delivery wait without bound, converting AG-14.3's bounded wind-down into an unbounded one. The delivery obligation is therefore owed to a consumer honouring the contract — receiving, or cancelling. A consumer that does neither within the wind-down bound is in **abandonment**, the same documented, untestable-to-termination violation Layer 1 states and AI-40.3 restates; the Layer 2 package contract carries the equivalent clause.

### 4.5 What is deferred, and to whom

No numeric capacity is named at any Layer 2 boundary (`R-AGE-018`). The posture is decided; the number is measured. **Owner: AG-21 (specifically AG-21.2, slow-consumer pressure) — doc 0003 names it "the AI-33/AI-34 of this layer".** Closing evidence: lane-absorption high-water mark and producer-wait observations under AG-21.2's stalled-consumer scenarios, with Layer 1's tie-break inherited — when two answers are indistinguishable on the measurements, prefer the smaller. Why no starting hypothesis, where AI-02 stated one: AI-02's acceptance criterion required a starting point because AI-34's charter demanded one to falsify; AG-21's charter demands correctness under pressure, not a number to test — and Layer 1's own hypothesis survived one milestone before measurement overturned it. Stating one here would repeat the mistake with none of the falsification machinery.

## 5. The observer decoupling mechanism

### 5.1 The failure a convention permits

Convention-based design: the harness holds a list of attached observers and, per event, hands the event to each inline, under a documented "observers must not block" rule. The rule is real; the structure ignores it:

```
stalled session logger
  └─ blocks its receive ─► inline fan-out loop in the harness
       └─ blocks ─► the harness's receive from the canonical stream
            └─ blocks (capacity 0) ─► the loop's send
                 └─ blocks ─► the Layer 1 transport read
                      ⇒ token delivery freezes for EVERY consumer,
                        including the primary frontend
```

One stalled cost meter freezes the screen. The documented rule is violated by a party that wrote no code wrong — the observer merely fell behind. That is what "conventional, not structural" looks like, and it is the trace `S-AGE-010` exists to forbid.

### 5.2 The trilemma, named honestly

At any single hand-off point, three properties cannot all hold: **(a)** lossless delivery to every consumer, **(b)** a producer that never waits on any consumer, **(c)** a fixed buffer bound. Layer 1 chose (a) + (c) and gave up (b) — backpressure is waiting. At Layer 2's harness-facing boundary, invariant 3 forbids giving up (b), and item 2 forbids giving up (a). So (c) is what bends, and the design says so rather than hiding it: each lane absorbs the gap between producer progress and its consumer's progress. The absorption is intrinsically bounded by the **run's own extent** — a run is finite, the lane drains as its consumer catches up, and everything is freed at run end (Layer 2 performs no persistence, R-02; this is memory only, inside the no-I/O closure). Whether a tighter numeric bound is wanted is exactly AG-21.2's measurement (§ 4.5). This is also what separates the chosen mechanism from the rejected pull-based replay record: the record retains everything for every consumer for the run's lifetime by design; a lane grows only while its consumer lags and only by the lag.

### 5.3 The mechanism, and where the stalled trace terminates

Chosen (proposal decision 3): one canonical internal stream whose **only** receiver is the distribution step; per attached consumer, a lane with its own forwarding activity that privately owns that consumer's receive-only carrier and applies the full send discipline toward it — every send waits on the destination *and* the consumer's own liveness signal, and the forwarding activity closes only the carrier it owns, never the canonical stream ("nothing else closes", applied recursively).

```
stalled consumer B
  └─ blocks its receive ─► forwarding activity B, mid-send
       └─ lane B absorbs subsequent events        ◄── trace TERMINATES here
distribution step: offer into lane B does not wait on consumer B
  └─ its receive from the canonical stream stays free
       └─ producer unaffected; consumer A unaffected
```

Every path from B's stalled receive ends at B's own forwarding activity. No consumer is privileged at the mechanism level — the run's primary frontend is one more lane; privilege is policy, and policy is Layer 3's. AG-20.2 inherits this as a mechanical test, and the lane is also what makes "eventually reported typed" for a stalled observer *expressible*: the lag is observable at one owned place.

### 5.4 The rendezvous objection, answered on its merits

Layer 1's rejected zero-capacity argument warned, verbatim: a rendezvous has *"zero tolerance for a consumer that pauses at all — and Layer 2's consumer pauses by design, to drive the permission protocol."* Layer 2 is that consumer, and the objection deserves an answer, not a citation of its defeat. The answer: a rendezvous is intolerant of a pause **on the receive step**, and indifferent to a pause on work performed after hand-off. The permission suspension does not sit on the receive step — it sits in the loop's per-call state. AG-10.3 makes that a requirement, not a hope: while one call is suspended, sibling calls schedule, execute and emit, and *message deltas already in flight keep flowing* — which is only achievable if permission, tool scheduling and cost work happen off the receive path. The party that genuinely pauses for the permission protocol — the frontend awaiting a human — pauses on a **per-consumer carrier**, where § 5.3's trace shows the pause backs up only its own lane. The objection's premise (a consumer doing per-event work inline on the receive step) describes a consumer Layer 2 is architecturally forbidden from being. One argument, used twice: the same off-the-receive-path fact answers the zero-capacity objection *and* is what the observer mechanism makes structural.

## 6. Ownership and closing — three nested scopes

One rule — the producer creates, the producer closes, exactly once, on every exit path, and nothing else closes — instantiated three times:

```
RUN SCOPE — owner and sole closer: the harness (stateful)
┌────────────────────────────────────────────────────────────────┐
│ run-start                                              run-end │
│                                                    (typed:     │
│  TURN SCOPE — owner and sole closer: the loop       completed /│
│  (stateless; re-instantiated per turn — it CANNOT   interrupted│
│   own the run bracket, which is why the owners       / failed) │
│   differ)                                                      │
│  ┌──────────────────┐   ┌──────────────────┐                   │
│  │ turn-start…end   │   │ turn-start…end   │                   │
│  │ (typed: finished │   │ turn-end on EVERY│                   │
│  │  / aborted)      │   │ exit path)       │                   │
│  └──────────────────┘   └──────────────────┘                   │
│                                                                │
│  DELEGATED-RUN SCOPE — the child harness owns and closes its   │
│  OWN run-scoped stream; the parent owns ONLY the subagent      │
│  bracket on its own stream                                     │
│  ┌──────────────────────────────────────────┐                  │
│  │ subagent-started            (parent's)   │                  │
│  │   child run-start … child run-end        │                  │
│  │   (child's stream — fully closed FIRST,  │                  │
│  │    leaf-first per AG-19.2)               │                  │
│  │ subagent-ended              (parent's)   │                  │
│  └──────────────────────────────────────────┘                  │
└────────────────────────────────────────────────────────────────┘
```

The delegated case is where "nothing else closes" earns its recursion: under leaf-first cancellation the child's stream fully closes — child run-end emitted by the child harness, on the child's stream — *before* the parent emits subagent-ended on its own stream, re-emitting the child's already-closed events parent-identified (invariant 2's binding point). The parent never closes the child's stream; the child never closes the parent's; ownership is nested and sequential, never shared. A child scope closing before its parent is therefore not an edge case the rule tolerates — it is the only order the rule permits.

**After the terminal**, per outcome (`R-AGE-012`): on *completed*, everything received is the complete ordered story. On *interrupted* or *failed*, the received prefix is trustworthy in exactly § 4.3's sense — nothing committed to history is missing from it; what had not yet happened is truncated. Layer 2 has no "sometimes no terminal at all" case at run scope, and § 4.3's harness-facing rule is *why* — run-end is inside the protected set.

## 7. The upward path, end to end

```
frontend            parent harness              in-flight loop invocation
   │                (the one stable,                     │
   │                 addressable surface)                │
   │ decision /      │                                   │
   │ steering /      │                                   │
   │ interrupt       │                                   │
   │ (typed payload) │                                   │
   │────────────────►│                                   │
   │                 │ resolve identity, two levels:     │
   │                 │  1. run identity  — live?         │
   │                 │  2. call identity — suspension    │
   │                 │     still held? (harness-owned    │
   │                 │     lookup; shape NOT fixed here) │
   │                 │                                   │
   │                 ├── both live ─────────────────────►│ resume that
   │                 │                                   │ one call
   │   typed         │                                   │
   │◄─ rejection ────┤ call no longer suspended,         │
   │   (call         │ run still live                    │
   │    granularity) │                                   │
   │   typed         │                                   │
   │◄─ rejection ────┤ run already ended                 │
   │   (run          │                                   │
   │    granularity) │                                   │
```

**Why the harness is the surface** (`S-AGE-017`, reconciling doc 0001 § 2.2 with § 2.3): § 2.2 draws the only upward arrow into the harness; § 2.3 narrates the suspension at loop level. Both are right at their own level — the harness is the stable, addressable thing a frontend can hold across a whole run, while the loop is stateless and re-instantiated per turn; the loop level is the *destination* of the routing, not the entry point. One surface, three typed payload kinds — not three paths that happen to look alike.

**The race that makes the rejection necessary, not defensive.** The downward stream and the upward surface are asynchronous by construction — that is the whole point of § 5. So this interleaving is *inherent*, not exotic:

```
t0  decision-required event delivered to the frontend (call identity X)
t1  human deliberates …           t1' run is interrupted; wind-down
                                      resolves suspension X; run-end emitted
t2  human answers; decision for X arrives at the surface — X is gone
```

The frontend cannot know at t2 what happened at t1' — its knowledge is a prefix of the stream. A silent drop would leave it awaiting an effect that will never come, unable to distinguish "applied" from "lost". The typed rejection at the matching granularity — call identity within a live run, run identity once the run has ended — is the only signal that tells the answerer its answer lost the race. This is one rule at two granularities, generalised from AG-10.1's call-level precedent, decided once here so AG-10 and AG-13 do not reinvent it independently.

**Two carve-outs**, so the machinery is not over-applied: an interrupt arriving *during* bounded wind-down (run not yet ended) is silently idempotent per AG-14.1 — the run is already doing what the interrupt asks; the typed rejection begins only once run-end has been emitted. And pause-resumption is model-initiated (a finish reason) and harness-internal — not an instance of the upward path at all, never routed through the identity or rejection machinery (consumed by AG-13).

**Recursion under delegation** (`S-AGE-022`): a child harness has its own inbound surface, but what a child's policy scope would ask about is asked on the **parent's** stream — one place a human watches. The frontend answers through the parent's surface; the parent's routing must reach the nested child's own suspension lookup. The obligation is stated here; the routing's shape is AG-19's.

## 8. Structure of `decision.md`

The AI-02 skeleton, re-instantiated — section order is AG-01.1's closing-checklist order, so a reviewer walking doc 0003 walks the artifact in parallel:

```
  §1  How to use this document      ← per-milestone entry points
  §2  What was decided              ← five one-line conclusions, before any argument
  §3  Decision 1 — the carrier              (checklist item 1)
  §4  Decision 2 — backpressure, two boundaries  (item 2)
  §5  Decision 3 — the observer model       (item 3)
  §6  Decision 4 — close and ownership      (item 4)
  §7  Decision 5 — the upward path          (item 5)
  §8  The delivery topology                 ← § 3's picture, as one settled diagram
  §9  What the package contract must state  ← abandonment clause and its kin
  §10 What each blocked milestone inherits  ← AG-04 · AG-10 · AG-13 · AG-14 · AG-19 · AG-20
  §11 Standing rules                        ← amendment terms; no-invented-channel rule
  §12 Closing-checklist verification
```

Every decision section carries the AI-02 five-part shape, in order: **Decision** (one paragraph) · **Why** (the argument, with sources) · **What this excludes** (alternatives, named and rejected) · **Consequences** (what becomes true elsewhere) · **Who inherits it** (named nodes, in their own terms). Uniformity is what turns five consultations into five successes.

**Where the argument's weight goes** — the five decisions are not equally contested, and the artifact must not pretend they are:

```
 contested                                                       settled
 |-----------------------------------------------------------------|
 backpressure     carrier        observer       upward     ownership
 (the narrowing   (genuinely     (a genuine     path       (precision,
  is new; § 4.2's  reopened —     source fork;  (a level    not persua-
  worked case      AI-02 forbids  rebuttal on   mismatch    sion; nobody
  must be on       closure by     the page)     to recon-   disputes it)
  the page)        citation)                    cile)
```

## 9. Reasoning rules the artifact applies

Five, each with the failure it prevents:

1. **Argue-at-this-layer** (§ 2.1) — prevents the restated default; every carrier ground cites a Layer 2 source by identifier.
2. **Strongest-opposing-case** — inherited from AI-02 verbatim; the rejected reading stated at its strongest before rebuttal. Applied to the carrier *and* to both source forks.
3. **Made-impossible-not-discouraged** (§ 5) — mechanisms tabulated by what each makes impossible; the blocking multicast row kept precisely because it makes nothing impossible. Prevents convention posing as structure.
4. **Pick-the-fork-and-record-the-rebuttal** — both genuine source ambiguities (multi-observer versus single hand-off; § 2.2 versus § 2.3 level) decided on the page with the losing reading written down, so a later reader can tell a decision from an oversight.
5. **Named-owner deferral** (§ 4.5) — the Layer 2 successor to AI-02's falsifiable-hypothesis rule: where AI-02 stated a number to falsify, AG-01 states **no number**, and instead a posture, a named measuring owner, and the evidence that closes the deferral. Prevents both the unearned constant and the silent omission (`S-AGE-024`).

The deletion test (AI-01 § 9 rule 5) governs scope throughout: if deleting a sentence gives a later milestone more options and that milestone is not AG-01, the sentence is cut. The suspension lookup is the worked example — its existence and ownership are decided; its shape, structure and storage are AG-10's and AG-13's.

## 10. Choices the spec forces where the proposal left an opening

Three gaps the spec flags, each closed here so `decision.md` inherits an answer, not a question:

| Gap | Spec hook | Choice |
| --- | --- | --- |
| The carrier-view convenience needs a **named** owning milestone | `R-AGE-003` | **AG-23.2** (test kit + examples, wave 6) — doc 0003's node that packages Layer 2's scripted-harness test kit, the exact analogue of Layer 1's AI-22 placement. The view, if ever built, lives in the test-support sibling, never owns or closes what it views, and is never a second contract. Whether it is built at all is AG-23.2's call |
| The two loss postures must be **nameable distinctly** | `R-AGE-005` | Two rows are appended to **AG-00's register in this same pull request**, following the AI-02 register-amendment procedure (append-only, next free ordinals, dated amendment blockquote, owner AG-01): one term for the loop-internal posture — Layer 1's sanctioned loss at turn scope — and one for the harness-facing posture — history-guarded truncation. `decision.md` cites them by identifier; it does not define them |
| What remains droppable must be **stated positively** | `R-AGE-007` | § 4.4's droppable set — facts the harness never learned before cancellation — plus the why-not-fully-lossless argument, both carried into decision § 4 verbatim in force |

A fourth choice the spec requires and the proposal deferred namelessly: the capacity-measurement owner is **AG-21.2** with the closing evidence of § 4.5 (`R-AGE-018`).

## 11. Alternatives considered and rejected

Consolidated across the five decisions; each row appears in the owning decision section's *What this excludes*:

| Alternative | Why it lost |
| --- | --- |
| Iterator at the boundary | Cannot be waited on alongside a permission decision and an interrupt — Ground 1, *stronger* at Layer 2 because AG-10.3 makes multiplexing a tested requirement, and Layer 2 stacks a second suspension mechanism on delivery |
| Push-callback observer registration | Re-creates synchronous-observer coupling (invariant 3 violated) unless every callback gets its own decoupling — Ground 2's hidden buffer, relocated to every observer, uncounted |
| Send-capable sink handle with explicit close | Hands a non-owner the ability to close; receive-only forecloses it at the type level for free |
| Layer 1's loss rule copied unchanged at both boundaries | Defeated by § 4.2's trace: a committed side effect silently absent from the stream while history holds it; the session log cannot trust the stream after any interrupted run |
| No loss path at all at the harness boundary | A consumer that stops reading and never cancels makes delivery wait without bound — contradicts AG-14.3. Full losslessness is unavailable at any price |
| Blocking synchronous multicast | Makes nothing impossible — the named example of conventional-not-structural |
| Bounded per-consumer buffer, drop on overflow | Makes backpressure propagation impossible, but by violating item 2's losslessness for any consumer slower than the producer |
| Pull-based replay record as the primary mechanism | Retains everything, for every consumer, for the run's lifetime, and leaves "how far behind" unanswered; the lane grows only by an actual lag and drains on catch-up |
| Single hand-off, all fan-out in Layer 3 | A coherent reading of doc 0001 § 2.2, rejected: AG-01.1's own checklist presumes a second consumer attaches at Layer 2, and the chosen mechanism subsumes the single-consumer case at no cost while the reverse is not true. Rebuttal recorded in decision § 5 |
| Three parallel upward paths, one per payload kind | R-09 is one decided path; three lookalike paths triple the identity, rejection and recursion machinery and invite drift |
| Silent drop for stray or late upward messages | Defeated by § 7's race: the answerer cannot distinguish "applied" from "lost"; AG-10.1's typed-protocol-error precedent generalised instead |

## 12. Rollout, rollback, and the seams later milestones extend through

**Rollout**: a decision artifact — merged, it becomes the settled input to AG-04 and, through it, waves 1–6. Nothing executes; adoption *is* citation by later milestones.

**Rollback**: `git revert` of the single commit, complete by construction (new directory, nothing imports it). Partial rollback is not meaningful and must not be attempted — the five decisions are load-bearing on each other (the carrier gives the send discipline its object; the observer mechanism presumes the carrier; the harness-facing rule presumes the harness owns the run scope; the upward path presumes run-end delivery). A rejected decision rejects the change. AG-00 lands in the same pull request: reverting this change alone leaves AG-00 intact; reverting AG-00 alone would strand this artifact's citations — so AG-00 reverts together with this or not at all. Post-merge reversal is priced by doc 0001 § 3.2 at roughly three times, which is why this is wave 0.

**Named seams** — where each later milestone extends without reopening this decision:

| Seam | Fixed here | Extended by |
| --- | --- | --- |
| The envelope | The carrier it travels on; the brackets' owners | AG-04 (contents, ordering, validation) |
| The suspension lookup | Exists, harness-owned, lives for the suspension | AG-10, AG-13 (shape, structure, storage) |
| Wind-down delivery | The obligation: committed history delivered before run-end | AG-14.3 (the time bound that keeps it bounded) |
| The delegation bracket | Ownership recursion; leaf-first close; parent-identified re-emission | AG-19 (cross-harness routing shape) |
| The observer attachment | The lane mechanism; no privileged consumer | AG-20 (hook taxonomy; the stalled-observer test), AG-23.2 (the optional view) |
| The capacity | Posture only; no number | AG-21.2 (measurement, with the inherited tie-break) |

## 13. Verification approach

Every requirement in `spec.md` is checkable by inspection; nothing runs, as a property of the node type (`openspec/config.yaml` sets `apply.tdd` for Go service code, and this change writes none). Threat matrix: N/A — no routing, shell, subprocess, VCS automation, executable-file classification, or process-integration boundary exists in a documentation-only decision change. The verification pass in `tasks.md` is ordered by cost of a missed defect:

| Rank | Check | Cost if missed |
| --- | --- | --- |
| 1 | The harness-facing narrowing is present with § 4.2's worked case, and the droppable set is stated | A session log that silently disagrees with reality after every interrupted run — the G8 of this layer |
| 2 | Each carrier ground cites a Layer 2 source by identifier | The decision reopens in wave 4 at triple cost |
| 3 | The stalled-observer trace terminates at the lane, and the multicast row records "makes nothing impossible" | Invariant 3 ships as a convention and fails at the first slow consumer |
| 4 | The rendezvous objection is answered by the receive-step distinction | The strongest recorded counter-argument stands unrebutted on the page |
| 5 | Three scopes, one closer each; leaf-first order in the delegated case | A double close or a shared-ownership drift the first time delegation is exercised |
| 6 | One surface, two rejection granularities, both carve-outs, the recursion | AG-10, AG-13 and AG-14 each invent a channel; the race of § 7 ships as a silent drop |
| 7 | Deferrals are named: AG-23.2, AG-21.2, the two AG-00 register rows | An omission is indistinguishable from a decision; vocabulary drifts one milestone in |
| 8 | The deletion test over every normative sentence; no Go identifiers; vocabulary cited by identifier | Over-reach into AG-04/AG-10/AG-13; the authoring constraint breaks |

## 14. Acceptance criteria for the design phase

1. `decision.md`'s section order matches AG-01.1's closing-checklist order, with the § 8 skeleton.
2. Every decision section carries the five-part shape of § 8, in order.
3. § 4.2's worked case and § 4.3's composition table appear in decision § 4; the droppable set is positive, not implied.
4. § 5.1's convention trace and § 5.3's terminating trace both appear in decision § 5, with the mechanism table judged by impossibility.
5. The rendezvous objection is answered by the receive-step/after-hand-off distinction, in decision § 4 or § 5, once, and cross-referenced rather than repeated.
6. The three-scope diagram of § 6 and the upward-path sequence of § 7 appear in the artifact, roles-and-messages only.
7. Decision § 10 states an inheritance for each of AG-04, AG-10, AG-13, AG-14, AG-19, AG-20, and the no-invented-channel rule once for all.
8. The AG-00 register amendment follows § 10's procedure; AG-23.2 and AG-21.2 are named where § 10 places them.

## 15. Next phase

`tasks.md` — five tasks, one per closing-checklist item, plus the AG-00 vocabulary reconciliation and the verification pass of § 13. Then `decision.md`, the deliverable.
