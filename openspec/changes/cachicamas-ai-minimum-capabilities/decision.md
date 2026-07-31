# Decision — the v1 capability set and optional-capability discovery

> **Change**: `cachicamas-ai-minimum-capabilities`
> **Milestone**: AI-03 — Decide the v1 capability set and optional-capability discovery
> **Node**: AI-03.1 — The capability matrix `[decision]`
> **Status**: decided
> **Date**: 2026-07-31
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: the Layer 1 half of the concern doc 0001 and ADR 0005 track as **G3**
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Binding vocabulary**: [AI-01's register](../cachicamas-ai-contract-vocabulary/decision.md) — every noun below is one of its rows, cited by identifier
> **Binding predecessor**: [AI-02's decision](../cachicamas-ai-stream-lifecycle/decision.md) — the observable shapes of cancellation and failure delivery are already fixed there, and are cited rather than re-decided

> [!IMPORTANT]
> **This artifact decides standings, not code.** No Go type name, field name, method name, or package identifier appears here — doc 0002's authoring constraint. AI-20 chooses spellings. Where a language mechanism must be discussed it is named descriptively ("an additional contract asserted on the provider value"), so the constraint holds inside § 9, which is where it is most tempting to break.

---

## 1. How to use this document

**If you are writing AI-20.5, AI-23, AI-24, AI-29 or AI-37:** § 12 tells you what your milestone inherits, in your milestone's own terms. Start there, then read the one section it points at.

**If you are marking a conformance case required or optional:** § 11 is the rule, and it is one sentence. You do not need the rest.

**If you are proposing a new capability, in either list:** § 4 is the admission test, and § 8 is the guard that stops the most common wrong answer. Both were written for you.

**If you are reviewing this artifact:** § 14 walks AI-03.1's closing checklist against it. § 6.2 is where the argument is load-bearing and where a defect is most expensive.

**Every Layer 1 noun below resolves to an AI-01 register row**, cited by identifier rather than paraphrased. Three nouns the register lacked — `V-PRV-16` **capability**, `V-PRV-17` **token counting** and `V-PRV-18` **capability outcome** — were appended to it by this same change, under AI-01 § 9 rule 2. They are used here; they are not defined here.

**Sections 5 … 10 follow AI-03.1's closing-checklist order**, so a reviewer can walk doc 0002 and this document in parallel. Four sections sit outside that spine: § 3 and § 4 come first because they are the method that produces all three lists; § 8 comes after the lists because it is a check on them, not a source of them; § 11 comes after the mechanism because it is what makes doc 0002's acceptance criterion satisfiable.

---

## 2. What was decided

The matrix, before any argument, for the reader who came for one row.

### Required — a conformant adapter exhibits every one

| Id | Capability | One-line obligation |
| --- | --- | --- |
| `CAP-R-01` | **streaming text** | Text arrives as delimited blocks on the stream, reconstructible exactly |
| `CAP-R-02` | **tool calls** | A model's request to invoke a tool arrives with identity, name, exact argument bytes and an observable ordinal |
| `CAP-R-03` | **completion metadata** | A normally-finished stream ends carrying a finish reason from the closed vocabulary and a usage record |
| `CAP-R-04` | **cancellation** | The caller-owned signal ends the stream within bounded time, with AI-02 § 5's discipline |
| `CAP-R-05` | **typed failures with the partial-output distinction** | Every failure is classifiable through one vocabulary on both delivery paths, and says whether content preceded it |

### Optional — an adapter that lacks one is **fully conformant**

| Id | Capability | Why optional rather than required |
| --- | --- | --- |
| `CAP-O-01` | **reasoning content** | Providers differ irreconcilably in what they emit, and a consumer that receives none loses nothing it needs for correctness |
| `CAP-O-02` | **token counting** | A required count forces an adapter whose vendor has none either to fail conformance or to fabricate — and a fabricated count corrupts a compaction decision **silently**, where an absent one degrades to a **visible** estimate |
| `CAP-O-03` | **honoring cache-boundary markers** | Markers are advisory by contract; an adapter for an auto-caching provider is conformant while ignoring every one of them |

### Excluded for v1 — not a capability at all, in either list

| Id | Excluded | Why |
| --- | --- | --- |
| `CAP-X-01` | **multimodal content beyond text** | Needs per-provider capability detection v1 does not model |
| `CAP-X-02` | **embeddings** | Returns something other than a normalized event stream; admitting it widens the very contract § 9 promises never to widen |
| `CAP-X-03` | **batch APIs** | No stream to cancel, no bounded close, no terminal event — incompatible with the lifecycle AI-02 fixed |
| `CAP-X-04` | **server-side tool execution** | Places tool execution below Layer 1, routing around the permission protocol and the sandbox. A trap-1 violation, not a scope choice |

### The four rules

1. **Discovery** (§ 9) — an optional capability is an additional, separately-asserted contract on the provider value. One contract per capability. The core provider interface (`V-PRV-03`) never widens.
2. **Marking** (§ 11) — a conformance case is optional **if and only if** the capability it exercises appears in the optional list above. Every other case is required.
3. **The record** (§ 10) — total over both closed lists, with a four-value outcome set in which **absent** and **not exercised** are different values.
4. **No substitutes** (§ 6.2) — Layer 1 never supplies a fallback for an absent capability. The consumer owns its fallback, because the consumer is the only party that knows what it can tolerate.

---

## 3. What a capability is — and the three things it is not

`V-PRV-16` names the unit. Its content, for this decision's purposes:

> **A capability is a behavior a consumer can ask a provider for, and whose presence or absence the consumer can observe.**

Both clauses are load-bearing. It must be *askable* — there is something a consumer wants from it — and its absence must be *observable* — the consumer can tell. Three near-neighbours fail one clause or the other, and each has a plausible claim to be "something a provider may or may not do". Separating them first is what keeps the optional list from absorbing every documented difference between vendors.

| Near-neighbour | Example from the record | Why it is not a capability |
| --- | --- | --- |
| **A contract obligation** | Exactly one terminal event per stream (`V-STR-18`); a per-stream sequence starting at 1 (`V-STR-13`); validation once, before any I/O (`V-REQ-22`) | It binds every adapter identically. Nothing to discover — the answer is always yes — and a provider that fails it is not lacking a capability, it is non-conformant |
| **An adapter-local mapping obligation** | Merging consecutive same-role messages for a provider that enforces alternation (doc 0001 § 3.3 row 5); minting tool-call identifiers for a provider that assigns none (row 7); supplying a default output-token limit where one is mandatory (row 6) | Invisible above Layer 1 by construction. The adapter's whole job is to make the vendor's shape disappear. No consumer asks "does this adapter merge messages?", because a conformant adapter always produces a request the vendor accepts |
| **A contract property optional for everyone** | Tool-call deltas: a call may arrive whole, with zero deltas (`V-STR-16`, doc 0001 § 3.3 row 1) | The optionality is a property of the *contract*, not of a provider. **No consumer may require a delta from any provider**, so there is nothing provider-specific to advertise. Making this discoverable would license a consumer to demand deltas from a provider that advertised them — the exact assumption row 1 exists to forbid |

The third row is the one most often got wrong, because doc 0001 § 3.3 row 1 uses the word *optional*. It means "optional in the contract, for all adapters", not "an optional capability". § 8 runs all nine of the register's rows through this test.

---

## 4. The admission tests

A list assembled by intuition is a list nobody can extend consistently — and this list *will* be extended, by whoever proposes a fourth optional capability. The tests are therefore part of the deliverable, not scaffolding behind it. Applied in order:

**Test 0 — is it a capability at all?** Apply § 3. A contract obligation, an adapter-local mapping obligation, or a contract property optional for everyone is not a capability and appears in no list.

**Test 1 — required** (`V-PRV-06`). Both clauses must hold:

- **(a) Loop-necessity.** A correct minimal agent loop above Layer 1 cannot be written without it. Not "is inconvenient without it" — **cannot be written correctly**.
- **(b) Universality without a lie.** Every provider v1 would plausibly adapt already offers it, so requiring it forces no honest adapter either to fabricate the behavior or to fail conformance for something that is not a defect.

Clause (b) does the real filtering, and § 6.2 is the case that proves it: token counting satisfies (a) and fails (b), and (a) alone would have made it required.

**Test 2 — optional** (`V-PRV-07`). Both clauses must hold:

- **(a) Legitimate absence.** At least one real provider does not offer it, or declining to implement it is a legitimate v1 adapter choice rather than a defect.
- **(b) Correct behavior on a recorded absence.** A consumer that learns the capability is absent can still behave correctly — by degrading, by not asking, or by doing without. If a consumer cannot proceed correctly, the capability is required or the feature is excluded; it cannot be optional.

Clause (b) says *recorded* absence, not *absence*. That word is why § 10 is part of the same decision and not a separate one: optionality is only survivable if the consumer can tell.

**Test 3 — excluded.** Neither of the above holds, because v1 has no contract for the capability's **presence**. This is the crisp line:

> **An optional capability has a defined absence. An excluded one has no defined presence.**

There is nothing to advertise, nothing to discover, and nothing to record, because the neutral vocabulary does not model what a positive answer would mean.

---

## 5. Required capabilities

**Closing-checklist item 1.**

Five entries, closed. Each carries what it obliges, what it does **not** oblige, and the test clauses that give it its standing.

The negative clause is not editorial. The expensive defect in a required list is not an omission — it is an entry that some honest adapter cannot satisfy, which leaves it two options: fail conformance for something that is not a defect, or fabricate the behavior. **A fabricated answer is indistinguishable from a real one at every layer above.** Two such readings hide inside `CAP-R-03` alone.

### `CAP-R-01` — streaming text

**Obliges.** Model-visible text (`V-REQ-08`) arrives on the stream as one or more blocks (`V-STR-14`), each delimited by a start and an end event, with zero or more deltas between them (`V-STR-16`), such that a consumer accumulating deltas reconstructs the text exactly.

**Does not oblige.** A minimum number of deltas. **A block delivered whole, with zero deltas, is conformant** and — after reconstruction — indistinguishable from a fragmented one, which is `V-STR-16`'s own clause. Nor does it oblige any particular fragment boundary: a delta may split a multi-byte character, and the consumer's reconstruction is what must survive it, not the producer's chunking.

**Why required.** (a) A loop that cannot receive text cannot render a response or decide anything from it. (b) Every candidate vendor streams text; requiring it excludes no adapter v1 wants.

### `CAP-R-02` — tool calls

**Obliges.** A model's request to invoke a tool (`V-REQ-16`) arrives carrying its identity, the tool name, exact argument bytes (`V-REQ-17`) and an observable call ordinal (`V-STR-21`) — the last so that results can rejoin in call order above Layer 1.

**Does not oblige.** That the *vendor* supply any of those directly. A vendor that assigns no identifier is adapted by an adapter that mints one and keeps the mapping (doc 0001 § 3.3 row 7): that adapter is **satisfying** this capability, not lacking it. Nor does it oblige incremental delivery, for the reason `CAP-R-01` gives. Nor does it oblige anything about executing, resolving or permitting a call — trap 1, and `V-OUT-04`.

**Why required.** (a) doc 0001 § 4.1's loop *is* a tool loop; without the transport representation of a call there is nothing for Layer 2 to schedule. (b) Every candidate vendor emits tool calls.

### `CAP-R-03` — completion metadata

**Obliges.** A stream that finishes normally ends with a completion event (`V-STR-20`) carrying a finish reason (`V-MET-01`) drawn from the closed vocabulary, and a usage record (`V-MET-09`).

**Does not oblige — two clauses, and both matter.**

1. **Not every finish-reason value must be emitted.** The obligation is that the vocabulary is *reachable*: an adapter maps its vendor's stop conditions into it, and `V-MET-08` **unknown finish reason** is a legal, conformant mapping for a string the vocabulary does not recognise. A vendor that never pauses simply never produces the pause value. Requiring emission of every value would require inventing stop conditions.
2. **Not every token count must be populated.** `V-MET-10` makes each count independently present or absent precisely because providers report different subsets, and `V-MET-11` makes "not reported" and "reported as nought" different facts. An adapter that reports only input and output tokens produces a valid usage record, not a deficient one. **Requiring a populated count is requiring a fabricated one** — the same defect as a mandatory token count, one level down, and harder to see because it hides inside a capability everyone agrees is required.

**Why required.** (a) Loop termination (`V-OUT-10`) is Layer 2's decision and the finish reason is its input; doc 0001's **G12(c)** records what guessing costs — refusal and pause collapsed into a fallback is a loop-termination defect, not a cosmetic gap. (b) Every vendor reports a stop condition and a token accounting.

### `CAP-R-04` — cancellation

**Obliges.** Exactly what AI-02 § 5 decided, cited and not restated: the caller owns the cancellable signal (`V-STR-06`), every send waits on it, and cancellation closes the stream within bounded time, with the one sanctioned loss path (`V-STR-09`) as its only exception.

**Does not oblige.** Anything of the vendor. Cancellation is a property of the local stream contract, not a provider feature.

**Why required, and why the standing is unusual.** This one is required **by construction** rather than by policy: AI-02 put the signal on the call that creates the stream, and AI-20.4's signature guard sees it. An adapter that lacks it cannot be built, so "absent" is not expressible. It appears in the list anyway, because the list must be total for `CAP-R-05`'s sake — a consumer asking "is this stream cancellable?" must find the answer here rather than infer it from a signature.

### `CAP-R-05` — typed failures with the partial-output distinction

**Obliges.** Every failure is classifiable through one vocabulary (`V-FAIL-05`, `V-FAIL-06`), carries its retryability (`V-FAIL-07`) and states whether any semantic content preceded it (`V-FAIL-09`) — **on both delivery paths** AI-02 § 7 fixed: returned directly before handover (`V-FAIL-11`), and as the terminal error event after it (`V-FAIL-12`).

**Does not oblige.** Any particular category to be producible by any particular vendor — that is AI-19's closed vocabulary and an adapter maps into it. Nor does it oblige a retry: whether to act on retryability is a policy decision one layer up (`V-OUT-11`, `V-FAIL-15`).

**Why required.** (a) `V-FAIL-09` is the single fact on which a correct retry decision turns, and doc 0001 § 7 **G8** names the defect a consumer writes without it — "retry if nothing completed" is precisely the predicate that gets this wrong, on the most common real-world failure. A loop without it duplicates observable output. (b) Every transport fails; nothing here asks a vendor for anything it does not already produce.

**One precision.** A provider whose pre-stream failures are typed but whose mid-stream failures arrive as an untyped closure is not partially conformant — it fails this capability. `V-FAIL-12`'s "one vocabulary, two delivery paths" is what makes this one capability rather than two.

### What is deliberately not on this list

Two behaviors are required of every adapter and are **not** capabilities, so listing them would make this list a duplicate of the contract — and a duplicated contract drifts.

- **The stream lifecycle** — ownership, exactly-one-terminal, per-stream sequence, the sanctioned loss path. Required, and it is `V-PRV-05`, proven by AI-20.3.
- **Request-translation totality** — that no expressible request feature is silently dropped. Required, and it is AI-26.8's, reported through AI-19's *unsupported capability* category. § 10.4 explains why that category is not an absent optional capability.

This is why the required list cannot be what AI-23 marks cases against. § 11.

---

## 6. Optional capabilities

**Closing-checklist item 2.**

Three entries, **closed**. Each carries the reason it is optional rather than required, and what a consumer does on a recorded absence — because test 2(b) is only satisfied if optionality is survivable, not merely permitted.

> **A provider that lacks every one of these is fully conformant.** That is `V-PRV-07`'s clause and it is restated here because it is the sentence a conformance suite is most likely to contradict in practice: an adapter implementing only the required surface passes, completely, with three recorded absences.

### `CAP-O-01` — reasoning content

**What it is.** Emitting a model's intermediate reasoning as reasoning content (`V-REQ-09`), with its state (`V-REQ-10`) and its opaque round-trip token (`V-REQ-11`).

**Why optional rather than required.** Test 2(a): doc 0001 § 3.3 row 2 records three irreconcilable provider behaviors — signed blocks, an opaque token count only, and thought signatures — and doc 0002 makes an absence explicitly legal for the first adapter: AI-29.0's first checklist item is to "record whether v1 emits reasoning events for the first provider **or documents a capability absence**", noting that "AI-03.1 makes both legal". Test 2(b): a consumer that receives no reasoning events loses nothing it needs for correctness, and cannot be silently corrupted, because reasoning is never rendered as, merged into, or substituted for text (`V-REQ-09`).

**On a recorded absence.** The consumer proceeds unchanged. There is no degraded mode, because nothing above Layer 1 depends on reasoning being present.

**What is optional is emission, not the neutral shape.** AI-07 builds reasoning content, the reasoning state and the round-trip token regardless of whether any v1 adapter emits them — those are Layer 1 contract items (doc 0001 § 3.3 row 2). A reader who concludes from this row that AI-07 is conditional has inverted it.

**Not sub-dividable.** An adapter that advertises this capability inherits `V-REQ-11` whole: the round-trip token is byte-exact, always. Splitting the capability into "emits reasoning" and "round-trips the token" would license an adapter that emits reasoning and drops signatures — and doc 0001 § 3.2 records what that breaks: multi-turn extended thinking with tool use fails, because at least one provider signs reasoning cryptographically. An advertised capability is advertised whole.

### `CAP-O-02` — token counting

**What it is.** `V-PRV-17`: answering how many tokens a given request would consume, asked of the provider **before** the request is sent. Distinct from `V-MET-09` **usage**, which reports what a response consumed after the fact.

**This is the load-bearing entry**, so the argument is given in full, including the reading that says it should be required — because that reading is correct as far as it goes, and an artifact that does not concede it will not survive the first person who makes it.

**The case for requiring it, at its strongest.** Test 1(a) is satisfied, and not marginally. Layer 2's compaction (`V-OUT-06`) must decide what to remove from a transcript, and that decision needs a real number. doc 0001 § 6 seam 6 says so directly: "Compaction that estimates by character count is wrong by enough to matter." A capability that a core Layer 2 feature genuinely needs looks exactly like a required one, and the first reading concludes that it is.

**Why it loses.** Test 1(b). Requiring counting leaves an adapter whose vendor exposes no count exactly two options:

1. **Fail conformance** — for something that is not a defect, which turns the required list into a filter on which vendors may be adapted rather than a definition of correctness. doc 0001 § 8's v1 target is one adapter proven against a suite; a required list that pre-excludes vendors is deciding AI-24's question from wave 0.
2. **Fabricate** — implement the count by estimating, and return the estimate as though it were a count. This is what "or lie" means in doc 0002's charter note.

The second option's cost is the whole argument, and it is worth stating precisely rather than as an epithet:

> Given an honest **absence**, compaction estimates, **knows it is estimating**, and can be conservative — it can protect more recent turns, or refuse to compact, or surface the uncertainty. Given a **fabricated count**, compaction believes the figure. A figure wrong in the unsafe direction removes content that should have been kept, silently, with no signal at any layer that anything was ever estimated.
>
> **A fabricated count is worse than an absent one, because the absent one degrades visibly and the fabricated one corrupts invisibly.**

Test 2 then holds trivially: (a) not every vendor exposes a counting endpoint; (b) a consumer that learns counting is absent proceeds by estimating — **because the absence was recorded**, which is § 10's job and the reason it is part of this decision.

**On a recorded absence.** The consumer estimates, and knows it is estimating. Layer 1 supplies nothing.

**The corollary, promoted to a standing rule** (§ 13 rule 4), because it is the same argument turned on Layer 1 itself: **Layer 1 must never supply a fallback estimate for an absent capability.** A default implementation that estimates is a fabrication with better provenance — the consumer cannot tell it apart from a real count, which is precisely the harm. Layer 1 states the absence; the consumer owns the fallback, because the consumer is the only party that knows what it can tolerate.

### `CAP-O-03` — honoring cache-boundary markers

**What it is.** An adapter rendering the cache-boundary markers a request carries (`V-REQ-23`) into its vendor's explicit cache annotation, and enforcing the vendor's breakpoint cap (`V-REQ-24`).

**The case for calling this an adapter-local mapping obligation, stated first because it is genuinely strong.** `V-REQ-23` makes markers **advisory by contract**: "an adapter for an auto-caching provider ignores markers wholesale; an adapter for an explicit one honours them", and AI-11.3 proves that ignoring every marker leaves the request "fully translatable and semantically unchanged". If both behaviors are conformant and neither changes what the consumer receives, the difference looks exactly like § 3's second row — invisible above Layer 1, nothing to ask.

**Why it is nonetheless a capability: `V-REQ-24`, the breakpoint cap.** A provider that honors markers enforces a small hard cap on their number, and AI-11.2 makes exceeding it a caller-contract failure (`V-FAIL-01`) raised before any I/O. A provider that caches automatically has no cap. So the two behaviors differ in **what a consumer may legally construct** — and Layer 2, which stabilises cache prefixes under **G4**, places markers programmatically. A consumer placing markers in a loop needs to know whether a cap binds it. That is an askable question with an observable answer: test 0 satisfied, § 3's second row escaped.

**Why optional rather than required.** Test 2(a): doc 0001 § 3.3 row 9 — two providers cache prefixes automatically, one requires explicit annotation. Absence is not merely legitimate, it is the majority case. Test 2(b): a consumer that learns markers are not honored places none, or places them anyway and loses nothing, because markers never participate in validity (AI-11.1).

**On a recorded absence.** The consumer places markers or does not; either way its request is valid and its meaning unchanged. No cap binds it.

**What is discovered is honoring, never the markers.** Expressing cache-boundary markers on a request is a Layer 1 contract item owned by AI-11, available to every consumer against every provider. A reader who concludes from this row that markers are conditional has inverted it, exactly as with `CAP-O-01`.

### 6.4 Why the list is exactly three

doc 0002's checklist says "and anything else v1 admits". The honest answer, after running every candidate through § 4's tests, is that nothing else is admitted. The candidates and the clause each fails:

| Candidate | Verdict |
| --- | --- |
| Round-tripping the reasoning signature | **Not separate.** Inside `CAP-O-01`; splitting it licenses an adapter that emits reasoning and drops signatures |
| Reporting cache-read and cache-write token counts | **Not a capability.** A usage field, already inside `CAP-R-03`, where `V-MET-11` absence-versus-zero is the whole answer. A provider reporting no cache tokens has absent *fields*, not an absent capability |
| Honoring tool choice — forcing a named tool | **Not a capability.** A request feature a vendor cannot receive is a translation failure, typed and named at AI-26.8.2 through AI-19's *unsupported capability* category. A failure at request time, not a standing (§ 10.4) |
| Carrying a provider escape hatch | **Not a capability.** `V-REQ-28` is designed so every adapter but one ignores a given hatch "without needing to know it exists". Nothing to ask |
| Server-supplied request identity, retry-after durations | **Not a capability.** Safe metadata on a failure (`V-FAIL-13`), present when the provider supplies it |

**The list is closed, and closing it is structural rather than tidy.** Every optional capability costs AI-23 a suite case, AI-38 a record entry and AI-24.1 a recorded expectation — and § 10's totality property is only available over a closed list. A speculative fourth is three milestones of work for a capability nobody asked for.

**How a fourth is admitted.** By amendment to this artifact, in the pull request that needs it, under doc 0002's living-graph clause — the same discipline AI-01 § 9 rule 2 applies to nouns. The proposer applies § 4's test 2 and § 8's divergence rule, and states which clause of each is satisfied.

---

## 7. Excluded for v1

**Closing-checklist item 3.**

Four entries. The separating rule, restated from § 4 because this is where it is applied:

> **An optional capability has a defined absence. An excluded one has no defined presence.**

An exclusion is not "an optional capability that no provider happens to offer". It is a capability the neutral vocabulary does not model, so there is nothing to advertise, nothing to discover, and nothing to record.

### `CAP-X-01` — multimodal content beyond text

**Why excluded.** doc 0001 § 8 states it as a v1 non-goal: "Image and audio discriminators exist in the content vocabulary and no constructor produces them. **Enabling them requires per-provider capability detection that v1 does not model.**" `V-REQ-05` records the same fact from the vocabulary side: image and audio are named in the content vocabulary and deliberately have no producer.

**Why not optional.** This is the exclusion most likely to be re-proposed as an optional capability, so the reason is worth spelling out. Optionality requires a defined presence, and "supports images" is not one question — it is a family: which modalities, in which direction, in which formats, at which size limits, with which transcoding. That family *is* the per-provider capability-detection model doc 0001 says v1 does not have. Admitting it as an optional capability would require building that model, which is the work being deferred. And `V-REQ-05` requires every registered content-part kind to have a constructible payload, so a positive answer would also be contract work, not capability work.

### `CAP-X-02` — embeddings

**Why excluded.** `V-PRV-01` defines a provider as "anything that accepts a normalized request and returns a normalized event stream". An embeddings call returns a vector. It has no stream, no blocks, no finish reason, no usage in the same sense, and no terminal event.

**Why not optional.** Admitting it would require the provider surface to express a second call shape, which is exactly the widening § 9 promises never to do. An optional *contract* asserted on the provider value could carry it — and that is precisely why the exclusion has to be explicit rather than left to the mechanism's flexibility: the mechanism would permit it, and the layer boundary does not. Embeddings belong to a different port, which nothing in v1 defines.

### `CAP-X-03` — batch APIs

**Why excluded.** A batch submission is asynchronous, its result arrives after an unbounded delay, and it has no live stream. Every clause of AI-02's lifecycle assumes otherwise: a stream is "the bounded, ordered delivery of events for exactly one model call" (`V-STR-01`), it ends with exactly one terminal event, and it closes within bounded time on cancellation. **There is nothing to cancel in a batch job and nothing to close.**

**Why not optional.** Same failure as embeddings: it is not a behavior added to a stream, it is the absence of a stream. And Layer 2's loop is interactive by construction — doc 0001 § 4.1's permission protocol suspends and resumes *during* a response.

### `CAP-X-04` — server-side tool execution

**Why excluded.** The provider executes a tool itself and returns its result. This is the exclusion with the most principled reason, and it is not scope:

- It is **trap 1** directly. Layer 1 carries the transport representation of a tool call and "never acts on it" (`V-REQ-16`); tool execution is `V-OUT-04`, owned by Layer 2 for scheduling and Layer 3 for confinement.
- It would place execution **below** Layer 1, where it is invisible to the permission protocol (`V-OUT-05`, doc 0001 § 6 seam 2, **G1**) and to the sandbox (`V-OUT-16`, seam 3, **G2**). doc 0001 § 6 seam 2 states the consequence of execution happening out of band: "the event stream stops being a complete description of the session. Every frontend then reimplements it, differently."

**Why not optional.** An optional capability is a behavior a consumer may use or not. This one *removes a decision the architecture assigns to another layer*, and it does so whether or not any consumer asks. There is no version of "absent" that makes its presence safe.

### Exclusion is not permanent, and it is not a gap

Each entry above is recorded so that its absence "reads as a decision rather than an oversight" (doc 0001 § 8's own framing). Admitting one after v1 is a new milestone with its own decision — most obviously `CAP-X-01`, which needs the capability-detection model first. None of them is admitted by this decision's amendment route, which is for optional capabilities the vocabulary already models.

---

## 8. The nine-row leakage cross-check

doc 0001 § 3.3 lists nine known provider divergences. Running all nine through § 4's test 0 is the single most useful guard available on § 6, because every row is *a way providers differ* and is therefore a candidate optional capability on a fast reading.

| # | Divergence | doc 0001 places it | Capability? |
| --- | --- | --- | --- |
| 1 | Tool-call arguments streamed incrementally versus whole | **L1 contract** | **No** — a contract property optional for *everyone*. No consumer may require a delta from any provider |
| 2 | Reasoning as signed blocks / opaque count only / thought signatures | **L1 contract** | The neutral shape is contract (AI-07). **Emission** is `CAP-O-01` |
| 3 | Refusal and pause-turn stop reasons with no neutral equivalent | **L1 contract** | **No** — vocabulary completeness (AI-13.1). Reachability is required; emitting any particular value is not (`CAP-R-03`) |
| 4 | Tool-result placement — block in a user message / distinct role / nested object | Adapter-local | **No** — mapping obligation |
| 5 | Strict user/assistant alternation on one provider | Adapter-local | **No** — mapping obligation. The adapter merges; nobody asks |
| 6 | One provider requires an explicit output-token limit | Adapter-local | **No** — mapping obligation, with a documented default |
| 7 | One provider assigns no tool-call identifiers | Adapter-local (+ L3) | **No** — mapping obligation. The adapter mints and keeps the mapping; that is `CAP-R-02` satisfied |
| 8 | System instruction as top-level field / differently-named field / nested object | L1 contract (segments) **+** adapter-local rendering | **No** — the segmented shape is contract (`V-REQ-19`); rendering is a mapping obligation |
| 9 | Two providers cache automatically, one requires annotation | L1 contract (markers) **+** adapter-local honoring | Markers are contract (`V-REQ-23`); **honoring** is `CAP-O-03` (§ 6.3) |

**The finding: nine documented divergences produce zero optional capabilities on their own.** Two rows have a capability *adjacent* to a contract item — rows 2 and 9 — and in both cases the capability is the emitting or honoring half, never the neutral shape. Promoted to a rule (§ 13 rule 3):

> **The divergence rule.** A documented provider divergence is evidence for an adapter's mapping table, not for the optional list. Promoting one to a capability requires showing that the difference is **consumer-visible** — that a consumer would behave differently knowing it. For row 9, `V-REQ-24`'s breakpoint cap is that showing. For rows 4 … 8, there is none.

**One discrepancy in the source, recorded because a later reader will hit it.** doc 0001 § 3.3's preamble says "Three require a contract change; the rest are absorbed inside an adapter", while its own table marks rows 8 and 9 as carrying a Layer 1 contract half. Both are accurate once the attribution is right: rows 1, 2 and 3 are **G12**'s three contract items, and rows 8 and 9's contract halves belong to **G4** — the structured system instruction and the breakpoint markers — scheduled separately at AI-10 and AI-11. The preamble counts G12's rows only. Reading "the rest are adapter-local" as covering rows 8 and 9 entirely would delete AI-11 from the plan.

---

## 9. The discovery mechanism

**Closing-checklist item 4.**

### What was inherited, and what was decided

Unlike AI-02's carrier, the mechanism *family* here is not a free choice. doc 0002's checklist item 4 states it as a requirement: an optional capability is an additional, separately-asserted contract on the provider value, so the core provider interface never widens. doc 0001 § 6 seam 6 and § 7 **G3**, and ADR 0005 § D4 row G3, all say the same thing in the same words. **This decision does not argue that choice; it would be dishonest to present an inherited constraint as an argued conclusion.** What was open is everything inside the family, and five sub-questions are decided below.

### The decision

**An optional capability is an additional contract, declared in Layer 1 beside the provider contract, one per capability, asserted on the provider value at the point of use. The provider interface (`V-PRV-03`) never widens.**

#### How an adapter advertises

**By satisfying the additional contract, and by no other means.** Advertising is not a flag, not a registration call, not a declared list, not a configuration entry, and not a constructor argument. The provider value either satisfies the extra contract or it does not, and that fact *is* the advertisement.

**An adapter that does not offer the capability declares nothing at all** — no field, no negative answer, no entry saying it is unsupported. This asymmetry is the mechanism's central virtue and is worth naming: **absence costs an adapter zero lines**, which is what makes the optional list cheap to extend and what removes every incentive to fabricate.

Three rules bind the advertising side:

1. **One contract per capability.** Never a single aggregate contract carrying all optional behavior — see the exclusions below.
2. **Advertising binds.** A provider that satisfies an optional contract and then declines to answer is **non-conformant**, not absent. Without this rule an adapter could satisfy every optional contract and refuse from inside each one, reproducing the widened interface this mechanism exists to prevent, one level down and harder to see.
3. **An advertised capability is advertised whole.** No partial satisfaction: `CAP-O-01` carries `V-REQ-11`'s byte-exact round trip with it (§ 6.1).

#### How a consumer asks

**At the point of use, by asking the provider value whether it also satisfies the optional contract.** The result is either the capability, usable immediately, or a **clean absence** — AI-20.5 item 1's words: "not an error and not a zero".

Three rules bind the asking side:

1. **The question is asked of the provider value.** Never of a model identity (`V-REQ-21`), a configuration entry, or a provider catalog. A catalog-driven capability table is `V-OUT-14` reaching into Layer 1; it goes stale the first time a vendor ships a change, and Layer 1 reads no configuration by rule (ADR 0005 § D1 row 4).
2. **Absence is not a failure.** It is not reported through `V-FAIL-01` or `V-FAIL-05`, and it is not the *unsupported capability* category — § 10.4.
3. **Layer 1 supplies no substitute.** § 6.2's corollary, and § 13 rule 4.

#### Wrapping — the rule no source document states

A provider value can be wrapped: AI-37 adds observability, and Layer 2 may decorate a provider for its own reasons. A wrapper that satisfies the core provider contract and forwards nothing else **silently removes every optional capability of the value it wraps** — and the removal is invisible, precisely because absence is legitimately silent by the asymmetry above. A consumer sees a conformant provider that simply cannot count tokens, and nothing anywhere reports that something was lost.

> **The forwarding rule.** A value that wraps a provider either forwards every optional contract the wrapped value satisfies, or documents which it removes and why. Silently narrowing a provider's capability set is a defect, not a design choice.

This is the mechanism's one sharp edge, it is a direct consequence of choosing silent absence, and it appears in no source document. **AI-37 is the first milestone it binds**; every Layer 2 decorator inherits it.

### What this excludes

| Excluded | Why |
| --- | --- |
| **Widening the core provider contract** with an optional behavior every adapter must implement | The thing the mechanism exists to prevent. Every adapter then implements it or refuses from inside it, and the honest implementations differ from the fabricated ones only by a comment. AI-20.5 item 3's pin is the mechanical form of this prohibition |
| **A provider-returned capability list** — the provider declares what it supports | Two sources of truth, the list and the behavior, which drift. A provider can claim a capability it does not implement and nothing checks it. And the consumer still needs a way to *use* the capability, so the list buys nothing it does not also risk |
| **A configuration- or catalog-driven capability table** | `V-OUT-14`. Stale by construction, and Layer 1 reads no configuration |
| **A single aggregate optional contract** carrying every optional behavior | All-or-none: an adapter that can count tokens but emits no reasoning must either satisfy it dishonestly or forfeit a capability it has. And it cannot grow — adding a fourth optional capability would change a contract existing adapters already satisfy, which is a breaking change to something this decision promised would never widen |
| **Per-consumer contracts declared at each call site** | Discovery would still work, but nothing could enumerate the capabilities: AI-23 could not iterate them and AI-38's record could not be total. Two consumers would declare subtly different versions of the same question |
| **Any Layer 1 default implementation of an optional capability** | § 6.2. A default that estimates is a fabrication with better provenance |

### Consequences

1. **AI-20.5's three test items are derivable from this section** and none requires reopening it: item 1 is the ask and the clean absence, item 2 is the fully-conformant minimal provider, item 3 is the pin that the core contract does not widen.
2. **AI-20.1's documentation obligation grows by one clause**: the interface's documentation states that optional capabilities are separately asserted and enumerable, so a consumer knows where to look.
3. **The optional contracts are enumerable**, which is what makes § 10's record total and AI-23's optional-capability cases exhaustive rather than a hand-maintained list.
4. **AI-37 inherits the forwarding rule** before it writes its first wrapper.

---

## 10. The capability record

**Closing-checklist item 5.**

`V-PRV-09` states the requirement: the record is "the emitted, recorded outcome stating which capabilities a provider offers and which it does not. **'Absent' is a recorded outcome, not an unrun test** — a silently skipped case and a recorded absence are different facts."

doc 0002 restates that clause four times — AI-23.1 item 2, AI-23.8 item 1, AI-23.6 item 2, AI-38.2 — which is a reliable signal that prose is not enough. In an ordinary test report the two are typographically identical: a case with no result. **So the distinction is made structural rather than editorial**: it is a difference between two values of a closed set, not a difference in how a report is worded.

### The shape

```
  capability record
    ├── subject
    │     ├── which provider was judged
    │     └── which run produced this record
    │           (so two records are comparable, and a stale one is recognisable)
    │
    └── entries  — ONE PER CAPABILITY IN THE CLOSED LISTS, required and optional alike
          │
          ├── capability   the identifier from § 5 or § 6
          ├── standing     required | optional     ← from THIS decision, never from the run
          └── outcome      satisfied | absent | failed | not exercised
                                        ▲                        ▲
                                        │                        │
                              a CONCLUSION              the ABSENCE of a conclusion
                              (conformant)              (the record is inconclusive)
```

### The four outcome values

`V-PRV-18` names the unit. The set is **closed**, and it is closed for the same reason `V-MET-01` and `V-FAIL-06` are: an open one pushes classification back into prose.

| Outcome | Meaning | Legal for |
| --- | --- | --- |
| **satisfied** | The cases were exercised and the provider met them | required, optional |
| **absent** | The provider was asked, does not advertise the capability, and its cases were **deliberately** not exercised. A conclusion, and a conformant one | **optional only** |
| **failed** | The cases were exercised and the provider did not meet them | required, optional |
| **not exercised** | The cases did not run — a missing transcript, a harness error, an interrupted run. **Never a conformance outcome** | neither, as a result |

Three consequences of the table, each stated because each is a place a reader goes wrong:

1. **`absent` is illegal for a required capability.** A required capability that is not satisfied is `failed`. There is no third state, and there are no waivers — AI-38.1 item 2 says the same thing from the other end: "There are no waivers; that is what 'required' means."
2. **An advertised optional capability that fails its cases is `failed`, not `absent`.** § 9's advertising-binds rule, in the record. An adapter cannot escape a broken capability by having offered it.
3. **`not exercised` is not a fourth kind of result.** It is the record saying it does not know. It is in the set precisely so that it cannot be written as `absent`.

### The verdict rule

> A capability record is a **pass** when every required entry is `satisfied`, every optional entry is `satisfied` or `absent`, and **no** entry is `not exercised`.
>
> A record containing any `not exercised` entry is **inconclusive** — neither a pass nor a failure. It reports that the run cannot say whether the provider conforms.

The third clause is the whole of checklist item 5, made mechanical. Without it, a four-value set collapses to three the moment a verdict is computed, and a skipped case reads as a pass again — the failure `V-PRV-09` exists to prevent, defeated at the first use of the thing built to prevent it.

### What makes the record total

**One entry per capability in this decision's closed lists, in every record.** A capability with no entry is a **defect in the run**, not an absence.

Totality is only available because § 5 and § 6 are closed, which is the second structural reason § 6.4 closes the optional list rather than leaving it open-ended.

Two further properties, each required by a consumer doc 0002 already names:

- **Standing comes from this decision, not from the run.** AI-23.6 item 2 requires that a provider failing a required case cannot pass. If the run supplied the standing, a run could demote a required capability by recording it optional, and the suite's own verdict would become negotiable.
- **Records are comparable entry by entry.** AI-24.1 item 4 records an **expected** record for the chosen vendor *before* an adapter exists; AI-38.2 asserts the generated one against it. A difference in either direction is a finding: an unexpected `absent` is a regression, and an unexpected `satisfied` means the adapter grew a capability nobody reviewed.

### What the record does not carry

- **No capability-specific detail** — not a breakpoint cap, not a counting method, not timings. The record answers "what did this run observe about this capability", and every additional field is one more thing to keep comparable between AI-24.1's expectation and AI-38.2's generated record.
- **No model content, no credential, no raw provider text.** AI-40.2 publishes this record into package documentation, so `V-FAIL-13`'s safe-metadata posture and `V-FAIL-14`'s redaction discipline bind it. A record is published by construction; it is not a debugging artifact.

### 10.4 One distinction the record must not blur

AI-19.2's failure taxonomy contains an **unsupported capability** category, and AI-26.8.2 raises it "WHEN a request expresses something the vendor cannot receive". It shares a word with this section and nothing else:

| | Absent optional capability | The *unsupported capability* failure |
| --- | --- | --- |
| **What it is about** | The provider | One request's features |
| **When it is known** | Before any request is built | At translation time, per request |
| **How it is observed** | A clean absence from the ask (§ 9) | A typed failure (`V-FAIL-05`) naming the feature |
| **Is it an error?** | **No** — AI-20.5 item 1 is explicit | Yes |
| **Where it is recorded** | A record entry, outcome `absent` | Not in the record at all |

Confusing them makes a consumer treat absence as a failure, which is exactly what AI-20.5 item 1 forbids, and makes an adapter report a translation failure as a missing capability, which would hide a real defect behind a conformant-looking record.

---

## 11. The marking rule

doc 0002's acceptance criterion for this milestone is precise and easy to under-read:

> **"AI-23's suite can mark each case required or optional from this list alone."**

§ 5 found that the required list is five capability-shaped entries while the contract has many more required cases — every ordering invariant, every ownership rule, the pre-stream contract, redaction, translation totality. If the suite marked cases by looking them up in the required list, all of those would be **unmarked**, and the natural default for an unmarked case is the dangerous one.

So the criterion is not satisfiable from the required list. It is satisfiable from the **optional** list plus a default:

> **The marking rule.** A conformance case is **optional if and only if** the capability it exercises appears in this decision's optional list (`CAP-O-01` … `CAP-O-03`). **Every other case is required.**

This works because the optional list is closed and short, and it produces the right failure mode: a capability nobody has classified defaults to **required**, so an unclassified case fails loudly rather than being skipped silently. That is § 10's posture — an unrun case is never a pass — applied one level up, in the suite instead of in the record.

**Worked cases**, because the rule is easier to state than to apply:

| Conformance case | Marked | Why |
| --- | --- | --- |
| Concatenated deltas reconstruct the text exactly (AI-23.2) | required | `CAP-R-01` |
| A whole tool call with zero deltas (AI-23.3 item 2) | required | `CAP-R-02`, whose negative clause makes the zero-delta shape mandatory to support, not optional to receive |
| Exactly one terminal per stream (AI-23.4 item 1) | required | Exercises no listed optional capability → required by default. It is `V-PRV-05`, and the default is what marks it |
| A planted secret appears in no event or error string (AI-23.7) | required | Same: no listed optional capability → required |
| Streamed reasoning never leaks into text (AI-23.8 item 1) | optional | `CAP-O-01` |
| A provider advertising counting answers a count (AI-23.8 item 4) | optional | `CAP-O-02` |
| Absent-versus-zero is honored in usage (AI-23.8 item 3) | **required** | Grouped under AI-23.8's optional-capability node, but it exercises `CAP-R-03`. **The node it lives in does not decide its standing; the capability it exercises does** |

The last row is the rule's sharpest edge and the reason it is stated as a biconditional over *the capability the case exercises* rather than over the suite's node structure.

---

## 12. What each blocked milestone inherits

doc 0002: *"Blocks: AI-20.5, AI-23, AI-24."* Each entry is written in that milestone's own terms, so its author can check the acceptance criterion — writable without reopening — from this table alone.

### AI-20.5 — optional capabilities are discovered, not required

- **Item 1** (a consumer discovers the token-counting capability and uses it; otherwise observes a clean absence) is § 9 entire: the ask is made of the provider value, and absence is "not an error and not a zero" — this decision adopts AI-20.5's own words as the contract.
- **Item 2** (a provider implementing only the required surface is fully conformant) is § 6's opening clause, and § 5 is the complete definition of "the required surface" it must exercise.
- **Item 3** (*pin*: the core interface does not widen) is the mechanical form of § 9's decision. The four rejected alternatives are what the pin protects against.
- **Also inherited by AI-20.1**: the interface documentation states that optional capabilities are separately asserted and enumerable.
- **Not inherited, because it is AI-20's:** every declaration — the provider contract, the optional contracts, and their spellings.

### AI-23 — the provider conformance suite

- **AI-23.1 item 2** (a case can be marked required or optional-capability; a skipped optional case is reported, never silent) is § 11's marking rule plus § 10's outcome set. The rule is total, so no case is unmarkable.
- **AI-23.8** (optional-capability cases) has exactly three subjects: `CAP-O-01`, `CAP-O-02`, `CAP-O-03`. Its item 3 — absent-versus-zero in usage — exercises `CAP-R-03` and is **required** despite living in that node (§ 11's last worked case).
- **AI-23.6** (the required/optional matrix) emits the record of § 10: total over both closed lists, four-value outcome set, standing taken from this decision, and the verdict rule that makes any `not exercised` entry inconclusive. Its item 2 — a provider failing a required case cannot pass, a provider skipping an optional case passes with the skip recorded — is the verdict rule stated from the suite's side.
- **AI-23.4 and AI-23.7** are marked required by the default, not by an entry. That is the rule working as intended.
- **Not inherited:** every assertion. This decision supplies the list and the marking rule, not the cases.

### AI-24 — select first provider and transport

- **AI-24.1 item 4** (which optional capabilities the vendor supports, recorded as the expected capability report) is written in § 10's vocabulary: one entry per capability in the closed lists, with an expected outcome from the four-value set. Only `satisfied` and `absent` are legitimate *expectations*; `failed` and `not exercised` are run results, not predictions.
- **AI-24.1 items 2 and 3** feed specific entries: whether the vendor expresses cache breakpoints or caches automatically decides the expected outcome for `CAP-O-03`; whether it signs reasoning blocks decides part of `CAP-O-01`'s.
- **The five required capabilities are a floor on vendor selection.** A candidate vendor that cannot support all five cannot be the first adapter — which is exactly why § 5's negative clauses matter: an over-stated requirement would exclude a vendor for something that is not a defect.
- **Not inherited, and deliberately not pre-empted:** which vendor. This decision names none and guesses at none.

### Further downstream, stated because each has a dependency on a specific line here

| Milestone | What it receives |
| --- | --- |
| **AI-11** | `CAP-O-03` depends on `V-REQ-24`'s cap being real and enforced before I/O. AI-11.2 is what makes honoring consumer-visible, and therefore what makes it a capability at all rather than an adapter-local mapping |
| **AI-13** | `CAP-R-03`'s two negative clauses stand on AI-13.1's unknown value and AI-13.3's absent-versus-zero. Neither is loosened here; both are required to exist |
| **AI-19** | The *unsupported capability* category is a request-time failure and is **not** an absent optional capability — § 10.4. AI-19.2 defines the category; this decision only keeps the two apart |
| **AI-29.0** | Its decision stands: emit reasoning, or record a capability absence. **Both are legal**, and that is this decision's entire contribution to it. If absence wins, AI-23.8 records `absent` for `CAP-O-01` — a result, not a gap |
| **AI-37** | § 9's forwarding rule, before the first wrapper exists |
| **AI-38.2** | § 10's record, generated and asserted against AI-24.1's expectation — "not implemented is a recorded result, never an unrun test" is the `absent`-versus-`not exercised` distinction, in that node's own words |
| **AI-40.2** | The record is published into package documentation, which is why § 10 forbids it carrying model content, credentials or raw provider text |

---

## 13. Standing rules this decision establishes

1. **The lists are closed, and a new entry is an amendment.** Required, optional and excluded are complete as of this artifact. A fourth optional capability arrives by amendment to this document, in the pull request that needs it, having applied § 4's test 2 and § 8's divergence rule — never by a downstream milestone deciding locally that a behavior is optional.
2. **The negative clause is part of a required entry.** A required capability stated without what it does *not* oblige is incomplete, because the expensive defect in a required list is an entry that forces an honest adapter to invent data.
3. **The divergence rule.** A documented provider divergence is evidence for an adapter's mapping table, not for the optional list. Promotion requires showing the difference is consumer-visible.
4. **Layer 1 never substitutes for an absent capability.** No default implementation, no estimate, no synthesised figure. Layer 1 states the absence; the consumer owns the fallback. A substitute is a fabrication with better provenance, and the consumer cannot tell it from a real answer — which is the harm.
5. **Advertising binds, and absence is silent.** A provider that offers a capability honors it whole; a provider that lacks one declares nothing. A wrapper forwards or documents (§ 9).
6. **An unrun case is never a pass.** In the record it is `not exercised` and makes the record inconclusive; in the suite an unclassified case defaults to required. The two are the same rule at two levels.
7. **This decision judges no provider.** It defines standings and the mechanism for observing them. Which capabilities a given vendor has is AI-24.1's, and whether the first adapter emits reasoning is AI-29.0's. A sentence here that answered either would delete a decision node.

---

## 14. Closing-checklist verification

AI-03.1's five items, each against this artifact.

| # | Closing-checklist item | Where answered | Status |
| --- | --- | --- | --- |
| 1 | **Required capabilities enumerated:** streaming text, tool calls, completion metadata (finish reason and usage), cancellation, typed failures with the partial-output distinction | § 5 — `CAP-R-01` … `CAP-R-05`, all five present, each with what it obliges, what it does **not** oblige, and its test clauses. The two clauses that would otherwise force a fabrication — a populated token count and an emitted finish-reason value — are stated explicitly. § 5's closing note records the two required behaviors that are contract obligations rather than capabilities, and why listing them would be wrong | **answered** |
| 2 | **Optional capabilities enumerated:** reasoning content, token counting, honoring cache-boundary markers, and anything else v1 admits — **each with the reason it is optional rather than required** | § 6 — `CAP-O-01` … `CAP-O-03`, each with its reason and with what a consumer does on a recorded absence. "Anything else" resolves to nothing, with five candidates recorded and the clause each fails (§ 6.4). `CAP-O-02` carries the full argument including the opposing reading; `CAP-O-03` carries the case against its own classification before answering it | **answered** |
| 3 | **Explicitly excluded for v1, with the reason:** multimodal content beyond text, embeddings, batch APIs, server-side tool execution | § 7 — `CAP-X-01` … `CAP-X-04`, each with its reason and with why it is not merely an optional capability nobody offers, under the stated rule: an optional capability has a defined absence, an excluded one has no defined presence | **answered** |
| 4 | **The discovery mechanism is decided:** an optional capability is an additional, separately-asserted contract on the provider value, so the core provider interface never widens. **How an adapter advertises it, and how a consumer asks, are both stated** | § 9 — both halves stated, with the family's inheritance acknowledged rather than presented as an argued conclusion. Advertise: by satisfying the contract and by no other means; an adapter without it declares nothing. Ask: of the provider value, at the point of use, observing a clean absence. Plus three advertising rules, three asking rules, the wrapper forwarding rule, and six rejected alternatives | **answered** |
| 5 | **"Absent" is a recorded outcome, not an unrun test:** the shape of the capability record AI-23.6 emits and AI-38.2 asserts is sketched here | § 10 — the record sketched with its subject and entries; total over both closed lists; each entry carrying capability, standing (from this decision) and one outcome from a closed four-value set in which **absent** and **not exercised** are distinct; the verdict rule that makes any `not exercised` entry inconclusive; what the record must not carry; and § 10.4's separation from the *unsupported capability* failure | **answered** |

**Beyond the checklist, and required by the milestone's acceptance criterion.** § 11's marking rule. doc 0002's criterion — "AI-23's suite can mark each case required or optional **from this list alone**" — is not satisfiable from the required list, because that list is a subset of what is required. It is satisfiable from the optional list plus a required default, stated as a biconditional over the capability a case exercises, with seven worked cases including the one that lives in an optional node while exercising a required capability.

**Register amendment.** Three nouns were appended to AI-01's register by this same change, under its § 9 rule 2: `V-PRV-16` **capability**, `V-PRV-17` **token counting** and `V-PRV-18` **capability outcome**, all owned by AI-03, all taking the next free `V-PRV` ordinals, under a dated amendment blockquote. No existing row was renumbered, reworded or removed. Each definition defers its substance to AI-03 by name, so the register still settles words rather than behavior (AI-01 § 9 rule 5). None of the three is defined in this change's artifacts; all three are cited by identifier. One of them, `V-PRV-16`, closes a gap AI-01 identified in its own § 7 preamble, which names five terms "AI-03's charter is not writable without" against a table that delivered four.

**Node status.** AI-03.1 closes on merge of this artifact. Per doc 0002's node grammar, a `[decision]` leaf produces no production code and closes when "the decision artifact answers every listed question and is merged." No `make test` gate applies; there is nothing in `backend/agent/` that this change touches.

**Milestone acceptance, restated from doc 0002 and checked:** *"AI-23's suite can mark each case required or optional from this list alone; a provider lacking an optional capability is fully conformant and records 'absent' rather than skipping silently."* The first clause is § 11. The second is § 6's opening statement — a provider lacking all three optional capabilities is fully conformant — together with § 10's outcome set, in which `absent` is a legal conformant outcome and `not exercised` is a different value that no record may use in its place.

**Wave 0 closes here.** doc 0002's exit condition for wave 0 — "The module exists, both import directions bite, and vocabulary, stream lifecycle, carrier and capability scope are recorded decisions" — holds on this merge.

**Unblocked by this decision:** AI-20 (`cachicamas-ai-model-provider`, node AI-20.5), AI-23 (`cachicamas-ai-conformance-suite`), AI-24 (`cachicamas-ai-first-provider-decision`) — and, through them, AI-29.0, AI-37, AI-38 and AI-40. Wave 1 (AI-04 … AI-13) begins in parallel and depends on nothing decided here.
