# Explore — the v1 capability set and optional-capability discovery

> **Change**: `cachicamas-ai-minimum-capabilities`
> **Milestone**: AI-03 — Decide the v1 capability set and optional-capability discovery
> **Node**: AI-03.1 — The capability matrix `[decision]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Target module**: `backend/agent/` — **no code is written by this change**
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Binding inputs**: [AI-01's register](../cachicamas-ai-contract-vocabulary/decision.md) — `V-PRV-06` … `V-PRV-09` are the four rows this milestone was pre-assigned · [AI-02's decision](../cachicamas-ai-stream-lifecycle/decision.md) — cancellation and failure delivery already have fixed observable shapes
> **Predecessors**: AI-01 (`cachicamas-ai-contract-vocabulary`), AI-02 (`cachicamas-ai-stream-lifecycle`)
> **Authoring constraint**: doc 0002's authoring constraint binds every artifact of this change. **No Go type name, field name, method name, or package identifier appears anywhere.** Where a language mechanism has to be discussed it is named descriptively — "an additional contract asserted on the provider value", never spelled.

---

## 1. What this milestone is, in one paragraph

AI-03 draws the line between what every adapter **must** do, what an adapter **may** do, and what v1 does not model at all — and it decides how a consumer finds out which side of the second line a given provider falls on. It writes no code. Its output is consumed as data by three later milestones: AI-20.5 implements the discovery mechanism, AI-23 marks every conformance case required or optional from this list, and AI-24.1 records the expected capability outcome for the chosen vendor before an adapter exists. AI-01 named the nouns (`V-PRV-06` … `V-PRV-09`) and refused the substance; AI-02 fixed the observable shape of two of the behaviors this milestone will call required. This milestone assigns the standings.

The milestone is scheduled in wave 0 for the reason every wave-0 milestone is: the required list is what the conformance suite *is*. A capability promoted from optional to required after an adapter exists invalidates that adapter's conformance; a capability demoted from required to optional after the suite exists invalidates the suite's own claim to be the definition of correctness.

## 2. Why "capability" is the right unit — and the three things it is not

The word does real work here, and three near-neighbours get mistaken for it. Separating them first is what keeps the optional list from bloating, because every one of the three has a plausible claim to be "something a provider may or may not do".

**A capability is a behavior a consumer can ask a provider for, and whose presence or absence the consumer can observe.** Two clauses, both load-bearing: it must be *askable* (there is something a consumer wants from it) and its absence must be *observable* (the consumer can tell). Everything else on this list fails one clause or the other.

| Near-neighbour | Example | Why it is not a capability |
| --- | --- | --- |
| **A contract obligation** | Exactly one terminal event per stream; a per-stream sequence starting at 1; validation before I/O | It binds every adapter identically. There is nothing to discover, because the answer is always yes, and a provider that fails it is not lacking a capability — it is non-conformant |
| **An adapter-local mapping obligation** | Merging consecutive same-role messages for a provider that enforces alternation; minting tool-call identifiers for a provider that assigns none; supplying a default output-token limit where one is mandatory | It is invisible above Layer 1 by construction. The adapter's whole job is to make the vendor's shape disappear. A consumer never asks "does this adapter merge messages?", because a conformant adapter always produces a request the vendor accepts |
| **A contract property that is optional for everyone** | Tool-call deltas: a call may arrive whole, with zero deltas | Optionality here is a property of the *contract*, not of a provider. No consumer may require a delta from **any** provider, so there is nothing provider-specific to advertise. Turning this into a discoverable capability would license a consumer to demand deltas from a provider that advertises them — which is exactly the assumption doc 0001 § 3.3 row 1 exists to forbid |

The third row is the one most likely to be got wrong, because doc 0001 § 3.3 row 1 literally uses the word *optional*. It means "optional in the contract for all adapters", not "an optional capability". § 7 below runs the whole nine-row register through this test.

## 3. What the record already says

Four sources carry a position, and unlike AI-02's carrier question none of them has expired.

| Source | What it records | Status |
| --- | --- | --- |
| doc 0001 § 6 seam 6 | Token counting is "an *optional* provider capability, discovered by type assertion", whose trivial v1 implementation is "absent, fall back to an estimate". The reason: "Making this optional rather than part of the provider contract is what stops it from forcing every adapter to implement it" | Binding. The mechanism family and the first optional capability are both given |
| doc 0001 § 7 **G3** | Compaction is Layer 2's; the Layer 1 impact is "an *optional* token-counting capability, discovered by type assertion — it does **not** widen the provider contract" | Binding |
| ADR 0005 § D4 row G3 | An **optional** provider token-counting capability, opt-in by assertion on the provider value, which does **not** reopen the provider interface | Binding as a verdict. The row names the interface by a retired spelling, which is not reproduced here — doc 0002's authoring constraint |
| doc 0002 AI-03's charter note | The same placement, with the consequence stated: "Layer 2's compaction needs a real token count and must degrade to an estimate when there is none, and making counting mandatory would force every future adapter to implement it or lie" | Binding, and it is the milestone's central argument |

**So the top-level mechanism is inherited, not chosen.** This is worth stating plainly, because AI-02's freedom on the carrier invites the assumption that AI-03 has the same latitude on discovery. It does not: doc 0002's closing-checklist item 4 states the mechanism family as a requirement — "an optional capability is an additional, separately-asserted contract on the provider value, so the core provider interface never widens". What is genuinely open is everything *inside* that family, and § 9 argues those:

1. One separate contract per capability, or one aggregate contract carrying all optional behavior?
2. Where is the optional contract declared — in Layer 1 beside the provider contract, or by each consumer at its own call site?
3. What exactly does "advertise" mean, and may an adapter advertise a capability it then declines to honor?
4. What does a consumer observe on absence, and may Layer 1 supply a substitute?
5. What happens to an optional capability when a provider value is wrapped?

Question 5 is not in any source document and is the one this exploration is least comfortable leaving unanswered — see § 9.5.

## 4. The admission tests

A list of capabilities assembled by intuition is a list nobody can extend consistently. Four tests, applied in order, decide where a candidate lands. They are stated here so that AI-23 and any later milestone proposing a fourth optional capability apply the same reasoning rather than re-deriving it.

**Test 0 — is it a capability at all?** Apply § 2. If it is a contract obligation, an adapter-local mapping obligation, or a contract property optional for everyone, it is not a capability and does not appear in any of the three lists.

**Test 1 — required.** A capability is required when **both** hold:
- **(a) Loop-necessity.** A correct minimal agent loop above Layer 1 cannot be written without it. Not "is inconvenient without it" — cannot be written correctly.
- **(b) Universality without a lie.** Every provider v1 would plausibly adapt already offers it, so requiring it forces no adapter either to fabricate the behavior or to fail conformance for an honest reason.

Both clauses matter, and clause (b) is the one that does the real filtering. Clause (a) alone would make token counting required: compaction genuinely needs a count. Clause (b) is what stops that conclusion, and it is the whole of the milestone's central argument.

**Test 2 — optional.** A capability is optional when **both** hold:
- **(a) Legitimate absence.** At least one real provider does not offer it, or declining to implement it is a legitimate v1 adapter choice rather than a defect.
- **(b) Correct behavior on a recorded absence.** A consumer that learns the capability is absent can still behave correctly — by degrading, by not asking, or by doing without. If a consumer cannot proceed correctly without it, the capability cannot be optional; it is required or the whole feature is excluded.

**Test 3 — excluded.** Neither of the above holds, because v1 has no contract for the capability's *presence*. This is the crisp line between excluded and optional: **an optional capability has a defined absence; an excluded one has no defined presence.** There is nothing to advertise, nothing to discover, and nothing to record, because the neutral vocabulary does not model what a positive answer would mean.

## 5. The required candidates, one at a time

doc 0002's checklist names five. Each is checked against test 1 rather than accepted, because an unexamined requirement is how a suite acquires a case no adapter can pass.

### 5.1 Streaming text

**(a)** A loop that cannot receive text cannot render a response or decide anything from it. **(b)** Every candidate vendor streams text; there is no provider worth adapting that does not. Both hold — **required**.

One precision the decision must carry, or an adapter will be failed for the wrong reason: "streaming" here obliges the *shape* — a block delimited by a start and an end, with zero or more deltas between them (`V-STR-14`, `V-STR-16`) — not a minimum number of deltas. A provider that delivers a whole run of text in one fragment is conformant. That is the same rule as § 3.3 row 1 for tool calls, and it belongs in the required capability's statement rather than being left to AI-23.2 to discover.

### 5.2 Tool calls

**(a)** doc 0001 § 4.1's loop *is* a tool loop; without the transport representation of a call there is nothing for Layer 2 to schedule. **(b)** Every candidate vendor emits tool calls. Both hold — **required**.

The precision here is where the adapter-local rows sit. Requiring the capability requires the adapter to deliver a call with an identity, a name, exact argument bytes (`V-REQ-17`) and an observable call ordinal (`V-STR-21`). It does **not** require the vendor to supply any of those directly: a vendor that assigns no identifier is adapted by an adapter that mints one (§ 3.3 row 7). That is a mapping obligation inside a required capability, not a second capability and not an excuse for an absence.

### 5.3 Completion metadata — finish reason and usage

**(a)** Loop termination (`V-OUT-10`) is Layer 2's decision and `V-MET-01` is its input. A loop with no finish reason guesses, and doc 0001's **G12(c)** records what guessing costs: refusal and pause collapsed into a fallback is a loop-termination defect. **(b)** Every vendor reports a stop condition and a token accounting. Both hold — **required**.

Two precisions, and both are places where a careless requirement would force a lie:

- **The finish-reason requirement is that the closed vocabulary is reachable, not that every value is emitted.** An adapter maps its vendor's stop conditions into `V-MET-01`'s vocabulary, and `V-MET-08` **unknown finish reason** is a legal, conformant mapping for a string the vocabulary does not recognise. A vendor that never pauses simply never produces the pause value. Requiring emission of every value would require inventing stop conditions.
- **The usage requirement is that the record is carried and `V-MET-11` absence-versus-zero is honored, not that every count is populated.** `V-MET-10` says each token-count field is independently present or absent precisely because providers report different subsets, and AI-13.3 makes a usage record with any subset valid. An adapter that reports only input and output tokens is fully conformant. **Requiring a populated count is requiring a fabricated one**, which is the same defect as a mandatory token count, one level down.

### 5.4 Cancellation

**(a)** A stream that cannot be ended early cannot be bounded, and Layer 2's permission protocol, delegation and interrupt handling all depend on it (doc 0001 § 6 seams 2, 7, 12). **(b)** Cancellation is not a vendor feature; it is a property of the local stream contract. Both hold — **required**.

This one is required *by construction* rather than by policy, and the decision should say so. AI-02 § 5 put the cancellable signal on the call that creates the stream, and AI-20.4's signature guard sees it. An adapter cannot be built that lacks it, so "absent" is not even expressible. Its semantics — every send waits on the signal, bounded close, one sanctioned loss path — are AI-02's and are cited, not re-decided.

### 5.5 Typed failures with the partial-output distinction

**(a)** `V-FAIL-09` is the single fact on which a correct retry decision turns, and doc 0001 § 7 **G8** names the defect a consumer writes without it: "retry if nothing completed" is precisely the predicate that gets this wrong. A loop without it duplicates observable output. **(b)** Every transport fails; nothing about the taxonomy asks a vendor for anything it does not already produce. Both hold — **required**.

The precision: the requirement covers both delivery paths AI-02 § 7 fixed. A provider whose pre-stream failures are typed but whose mid-stream failures arrive as an untyped closure is not partially conformant; it fails the capability. `V-FAIL-12`'s "one vocabulary, two delivery paths" is what makes the capability a single thing rather than two.

### 5.6 Is anything missing from the five?

Two candidates were considered and both fail test 0 rather than test 1.

- **Stream lifecycle conformance** — ownership, exactly-one-terminal, per-stream sequence, the sanctioned loss path. Required, certainly, but it is a contract obligation (`V-PRV-05`), proven by AI-20.3, not a capability. Listing it would make the capability list a duplicate of the contract, and a duplicated contract drifts.
- **Request translation totality** — that no expressible request feature is silently dropped. Also required, also a contract obligation; AI-26.8 proves it and AI-19's *unsupported capability* failure category is how it reports. § 10.3 explains why that category is not an absent optional capability.

This produces a structural finding that shapes the whole decision: **the required list is not the whole contract, and it must not pretend to be.** Which means the required list cannot be what AI-23 marks cases against — see § 8.

## 6. The optional candidates

doc 0002's checklist names three and invites more: "reasoning content, token counting, honoring cache-boundary markers, and anything else v1 admits".

### 6.1 Reasoning content

**Test 2(a)** — doc 0001 § 3.3 row 2 records three irreconcilable provider behaviors: signed blocks, an opaque token count only, and thought signatures. And doc 0002 makes an absence explicitly legal for the first adapter: AI-29.0 is a decision node whose first checklist item is "record whether v1 emits reasoning events for the first provider **or documents a capability absence**", with the note that "AI-03.1 makes both legal". Holds.

**Test 2(b)** — a consumer that receives no reasoning events loses nothing it needs for correctness. Reasoning is never rendered as, merged into, or substituted for text (`V-REQ-09`), so its absence cannot silently corrupt anything. Holds.

**Optional.** The precision that matters: what is optional is **emission**, not the neutral shape. AI-07 builds reasoning content, the reasoning state and the round-trip token regardless of whether any v1 adapter emits them — those are contract items (§ 3.3 row 2, an L1 contract row). An adapter that advertises reasoning inherits `V-REQ-11` whole: the round-trip token is a correctness requirement, byte-exact, and an adapter that emits reasoning while dropping signatures is non-conformant rather than partially capable. Sub-dividing reasoning into "emits reasoning" and "round-trips the token" would license exactly that adapter, and doc 0001 records what it breaks: multi-turn extended thinking with tool use.

### 6.2 Token counting

**Test 2(a)** — not every vendor exposes a counting endpoint, and an adapter for one that does not has nothing to implement. Holds.

**Test 2(b)** — a consumer that learns counting is absent can proceed: Layer 2's compaction degrades to an estimate. Holds — **but only because the absence is recorded.** A consumer that cannot tell the difference between "no count available" and "the count is zero" cannot degrade, it can only be wrong. This is why the capability and the record are one decision and not two.

**Optional, and this is the load-bearing case.** The argument in full, because it is the one a later reader is most likely to try to overturn:

Test 1(a) is satisfied — compaction genuinely needs a real count, and doc 0001 § 6 seam 6 says so: "Compaction that estimates by character count is wrong by enough to matter." A first reading therefore concludes that counting should be required. Test 1(b) is what defeats that reading. Requiring counting leaves an adapter whose vendor offers no count exactly two options:

1. **Fail conformance** — for a reason that is not its fault, and which would exclude a provider v1 might want. This makes the required list a filter on vendors rather than a definition of correctness.
2. **Fabricate** — implement the count by estimating, and return the estimate as though it were a count.

Option 2 is what "or lie" means in doc 0002's note, and its cost is worth stating precisely rather than as an epithet. Compaction consumes the count to decide what to remove from a transcript. Given an honest absence, it estimates, knows it is estimating, and can be conservative. Given a fabricated count, it believes the figure, and a figure that is wrong in the unsafe direction removes content that should have been kept — silently, with no signal at any layer that anything was estimated. **A fabricated count is worse than an absent one, because the absent one degrades visibly and the fabricated one corrupts invisibly.**

There is a corollary the decision must carry, because it is the same argument applied to Layer 1 itself: **Layer 1 must never supply a fallback estimate for an absent capability.** A default implementation that estimates is a fabrication with a nicer provenance. Layer 1 states the absence; the consumer owns the fallback, because the consumer is the only party that knows what it can tolerate.

### 6.3 Honoring cache-boundary markers

This is the candidate whose classification is genuinely arguable, and § 7's cross-check pushes against it, so it needs deciding rather than accepting.

**The case against calling it a capability.** `V-REQ-23` says markers are advisory *by contract*: "an adapter for an auto-caching provider ignores markers wholesale; an adapter for an explicit one honours them", and AI-11.3 proves that ignoring every marker leaves the request "fully translatable and semantically unchanged". If both behaviors are conformant and neither changes what the consumer receives, the difference looks exactly like the adapter-local mapping obligations of § 2's second row — invisible above Layer 1, nothing to ask.

**What defeats that case: `V-REQ-24`, the breakpoint cap.** A provider that honors markers enforces a small hard cap on their number, and AI-11.2 makes exceeding it a caller-contract failure raised before any I/O. A provider that caches automatically has no cap. So the two behaviors are *not* consumer-invisible: they differ in **what a consumer may legally construct**. Layer 2 stabilises cache prefixes (**G4**) and will place markers programmatically; a consumer placing markers in a loop needs to know whether a cap binds it. That is an askable question with an observable answer, which is test 0 satisfied and the § 2 second row escaped.

**Test 2(a)** — doc 0001 § 3.3 row 9: two providers cache prefixes automatically, one requires explicit annotation. Absence is not merely legitimate, it is the majority case. Holds.
**Test 2(b)** — a consumer that learns markers are not honored places none, or places them anyway and loses nothing, because markers never participate in validity (AI-11.1). Holds.

**Optional.** With one precision: what is discovered is *honoring*, never the markers themselves. Expressing markers on a request is an L1 contract item owned by AI-11 and available to every consumer against every provider. A reader who concludes from this row that markers are optional on the request has inverted it.

### 6.4 What else v1 admits — nothing, and the list is closed

"Anything else v1 admits" invites a fourth entry, and the honest answer after running every candidate through the tests is that there is none. The candidates considered, each with the test it fails:

| Candidate | Verdict |
| --- | --- |
| Round-tripping the reasoning signature | **Not separate.** It is inside `CAP-O-01`; splitting it licenses an adapter that emits reasoning and drops signatures (§ 6.1) |
| Reporting cache-read and cache-write token counts | **Not a capability.** It is a usage field, already covered by required completion metadata, where `V-MET-11` absence-versus-zero is the whole answer. A provider that reports no cache tokens records absent *fields*, not an absent capability |
| Honoring tool choice — forcing a named tool | **Not a capability.** A request feature the vendor cannot receive is a translation failure, typed and named at AI-26.8.2, reported through AI-19's *unsupported capability* category. That is a failure at request time, not a discoverable standing (§ 10.3) |
| Carrying a provider escape hatch | **Not a capability.** `V-REQ-28` is designed so that every adapter but one ignores a given hatch without knowing it exists. There is nothing to ask |
| Server-reported request identity, retry-after durations | **Not a capability.** Safe metadata on a failure (`V-FAIL-13`), present when the provider supplies it |

So the optional list is **exactly three, and closed**. Closing it is not tidiness: every optional capability costs AI-23 a suite case, AI-38 a record entry, and AI-24.1 a recorded expectation. A speculative fourth is three milestones of work for a capability nobody has asked for. A genuine fourth arrives as an amendment under doc 0002's living-graph clause, in the pull request that needs it — the same discipline AI-01 § 9 rule 2 applies to nouns.

## 7. The nine-row leakage cross-check

doc 0001 § 3.3 lists nine known divergences. Running all nine through test 0 is the single most useful check available on the optional list, because every row is *a way providers differ* and therefore a candidate optional capability by superficial reading.

| # | Divergence | Where doc 0001 puts it | Capability? |
| --- | --- | --- | --- |
| 1 | Tool-call arguments streamed incrementally versus whole | **L1 contract** | **No** — a contract property optional for *everyone*. No consumer may require a delta from any provider |
| 2 | Reasoning as signed blocks / opaque count / thought signatures | **L1 contract** | The *neutral shape* is contract (AI-07). **Emission** is `CAP-O-01`. The only row of the nine with a capability adjacent to it |
| 3 | Refusal and pause-turn stop reasons | **L1 contract** | **No** — vocabulary completeness (AI-13.1). Reachability is required; emission of any particular value is not |
| 4 | Tool-result placement — block in a user message / distinct role / nested object | Adapter-local | **No** — mapping obligation |
| 5 | Strict user/assistant alternation on one provider | Adapter-local | **No** — mapping obligation. The adapter merges; nobody asks |
| 6 | One provider requires an explicit output-token limit | Adapter-local | **No** — mapping obligation, with a documented default |
| 7 | One provider assigns no tool-call identifiers | Adapter-local (+ L3) | **No** — mapping obligation. The adapter mints and keeps the mapping |
| 8 | System instruction as a top-level field / differently-named field / nested object | L1 contract (segments) **+** adapter-local rendering | **No** — the segmented shape is contract; rendering is a mapping obligation |
| 9 | Two providers cache automatically, one requires annotation | L1 contract (markers) **+** adapter-local honoring | Markers are contract; **honoring** is `CAP-O-03` (§ 6.3) |

**The finding: nine documented divergences produce zero optional capabilities on their own.** Two of them (rows 2 and 9) have a capability *adjacent* to a contract item, and in both cases the capability is the emitting or honoring half, never the neutral shape. That is a strong result and it is worth recording as a rule, because it is the shape of the mistake a later reader will make: **a documented provider divergence is evidence for an adapter's mapping table, not for the optional list.**

**One discrepancy in the source, recorded because a later reader will hit it.** doc 0001 § 3.3's preamble says "Three require a contract change; the rest are absorbed inside an adapter", but its own table marks rows 8 and 9 as carrying an L1 contract half. Both are true once the attribution is right: rows 1, 2 and 3 are **G12**'s three contract items, while rows 8 and 9's contract halves belong to **G4** (structured system instruction, breakpoint markers) and are scheduled separately, at AI-10 and AI-11. The preamble counts G12's rows only. Reading "the rest are adapter-local" as covering rows 8 and 9 entirely would delete AI-11 from the plan.

## 8. The marking rule — the acceptance criterion's real requirement

doc 0002's acceptance criterion is precise and easy to under-read: *"AI-23's suite can mark each case required or optional **from this list alone**."*

§ 5.6 found that the required list is five capability-shaped entries while the contract has many more required cases — every ordering invariant, every ownership rule, the pre-stream contract, redaction. If the suite marked cases by looking for them in the required list, every one of those would be unmarked, and the natural default for an unmarked case is the wrong one.

So the criterion cannot be satisfied by the required list. It is satisfied by the **optional** list, plus a default:

> A conformance case is **optional if and only if** the capability it exercises appears in this decision's closed optional list. Every other case is **required**.

This works precisely because the optional list is closed and short. It also produces the right failure mode: a capability nobody has classified defaults to required, so an unclassified case fails loudly rather than being skipped silently — which is the same posture as item 5's, one level up.

## 9. Discovery — what is genuinely open, argued

The mechanism family is inherited (§ 3). Five sub-questions are open.

### 9.1 One contract per capability, or one aggregate?

An aggregate — a single optional contract carrying every optional behavior — is simpler to declare once and worse in every other respect. It is **all-or-none**: an adapter that can count tokens but emits no reasoning either satisfies the aggregate dishonestly or forfeits a capability it has. It also cannot grow: adding a fourth optional capability changes a contract that existing adapters already satisfy, which is a breaking change to something the decision promised would never widen.

**One separate contract per capability.** The cost is more declarations; the benefit is that each capability's presence is independently true or false, which is exactly what the record needs to be total.

### 9.2 Where is the optional contract declared?

If each consumer declares its own version of "the thing that can count tokens" at its own call site, discovery still works — but nothing can enumerate the capabilities, because there is no list. AI-23's suite could not iterate them, AI-38's record could not be total, and two consumers would inevitably declare subtly different questions.

**Declared in Layer 1, beside the provider contract, one per capability, enumerable.** Enumerability is what makes the record total and the suite exhaustive.

### 9.3 What does "advertise" mean, and may an advertised capability decline?

Advertising is **not** a flag, a registration call, a declared list, or a configuration entry. It is the fact that the provider value satisfies the additional contract. An adapter that does not offer the capability writes nothing at all — no field, no method returning a negative, no entry saying false. That asymmetry is the mechanism's main virtue: **absence costs an adapter zero lines**, which is what makes the optional list cheap to grow later and what stops any adapter being tempted to fabricate.

The corollary must be stated or it will be discovered by a consumer: **advertising binds.** A provider that satisfies the optional contract and then declines to answer is non-conformant, not absent. Without this rule an adapter would satisfy every optional contract and answer "unsupported" from inside each — reproducing the widened interface the decision exists to prevent, one level down.

### 9.4 What does a consumer observe on absence, and does it ask a value or a name?

AI-20.5 item 1 gives the answer's shape verbatim: "a clean absence, not an error and not a zero." Two things follow.

- **The question is asked of the provider value**, never of a model identity, a configuration entry, or a catalog. A catalog-driven capability table is `V-OUT-14` reaching into Layer 1, it goes stale the first time a vendor ships a change, and Layer 1 reads no configuration by rule.
- **Layer 1 supplies no substitute.** The consumer owns its fallback. § 6.2's argument applied to Layer 1 itself.

### 9.5 Wrapping — the question no source document answers

A provider value can be wrapped. AI-37 adds observability, and Layer 2 may decorate a provider for its own reasons. A wrapper that satisfies the core provider contract and forwards nothing else **silently removes every optional capability of the value it wraps** — and the removal is invisible, because absence is legitimately silent by § 9.3's asymmetry. A consumer sees a conformant provider that simply cannot count tokens, and nothing anywhere reports that something was lost.

This is the mechanism's one sharp edge, it is a direct consequence of choosing silent absence, and it is not mentioned in doc 0001, ADR 0005 or doc 0002. It belongs in the decision as a stated rule constraining every future wrapper, because the alternative is discovering it in Layer 2 after AI-37 has shipped.

## 10. The capability record

### 10.1 Why "absent" and "not exercised" must be different values

`V-PRV-09` states the requirement — "**'Absent' is a recorded outcome, not an unrun test** — a silently skipped case and a recorded absence are different facts" — and doc 0002 restates it at AI-23.8 item 1, AI-23.1 item 2, AI-23.6 item 2 and AI-38.2. The reason it needs restating four times is that in an ordinary test report the two are typographically identical: both appear as a case that did not run.

But they are opposite facts. **Absent** is a conclusion: the provider was asked, it does not advertise the capability, the cases were deliberately not exercised, and the provider is fully conformant. **Not exercised** is the absence of a conclusion: the case did not run because a transcript was missing, a harness errored, or a run was interrupted. One is a pass; the other means the record cannot be trusted to say whether the run passed.

Therefore the record's outcome set must contain both as distinct values, and the record's verdict rule must treat a single *not exercised* entry as making the whole record inconclusive rather than passing. That is the mechanical form of `V-PRV-09`.

### 10.2 What makes the record total

A record that lists only what happened cannot distinguish a missing entry from an absent capability. The record must therefore be **total over this decision's closed list**: one entry per capability, required and optional alike, in every record. A capability with no entry is a defect in the run.

Totality is only possible because the lists are closed, which is the second reason § 6.4 closes the optional list rather than leaving it open-ended.

Two more properties fall out of the consumers doc 0002 already names:

- **Standing comes from this decision, not from the run.** AI-23.6 item 2 requires that a provider failing a required case cannot pass. If the run supplied the standing, a run could demote a required capability by recording it optional.
- **Records must be comparable entry by entry.** AI-24.1 item 4 records an *expected* record before an adapter exists; AI-38.2 asserts the generated one against it. A difference in either direction is a finding — an unexpected absence is a regression, and an unexpected presence means the adapter grew a capability nobody reviewed.

### 10.3 One distinction the record must not blur

AI-19.2's failure taxonomy contains an **unsupported capability** category, and AI-26.8.2 raises it "WHEN a request expresses something the vendor cannot receive". That is a *failure*, at request time, about a request feature. An absent optional capability is a *recorded outcome*, discovered before any request is built, about the provider.

Confusing them makes a consumer treat absence as an error — precisely what AI-20.5 item 1 forbids. The decision should name the distinction, because the two share a word and nothing else.

## 11. Vocabulary check against AI-01

| Concept this decision needs | Register row | Present? |
| --- | --- | --- |
| The bare unit being classified | — | **missing** |
| Behavior every adapter must exhibit | `V-PRV-06` required capability | yes |
| Behavior an adapter may lack | `V-PRV-07` optional capability | yes |
| How a consumer learns which | `V-PRV-08` capability discovery | yes |
| What the suite emits | `V-PRV-09` capability record | yes |
| The value one record entry carries | — | **missing** |
| Counting a request's tokens before sending | — | **missing** |
| Who is judged | `V-PRV-01` provider, `V-PRV-02` adapter, `V-PRV-03` provider interface | yes |
| What judges | `V-PRV-12` conformance suite, `V-PRV-10` fake provider | yes |
| The required behaviors | `V-STR-14`, `V-STR-16`, `V-REQ-16`, `V-REQ-17`, `V-STR-21`, `V-MET-01`, `V-MET-09`, `V-MET-10`, `V-MET-11`, `V-STR-06`, `V-FAIL-05` … `V-FAIL-12` | yes |
| The optional behaviors | `V-REQ-09` … `V-REQ-11`, `V-REQ-23`, `V-REQ-24` | yes |
| The exclusions | `V-REQ-05` (image and audio named, no producer), `V-OUT-04`, `V-OUT-05`, `V-OUT-16` | yes |

Three gaps, each with the same shape as the two AI-02 found:

1. **capability** — AI-01's own § 7 preamble names it: "AI-03's charter is not writable without `capability`, `required capability`, `optional capability`, `capability discovery` and `capability record`." The preamble promises five; the table delivers four. Without the bare noun there is no way to say what § 2's three near-neighbours are *not*.
2. **capability outcome** — the word "outcome" appears inside `V-PRV-09`'s own definition, undefined. The distinction that row exists to protect is a distinction *between outcome values*, so it cannot be stated without the noun. This is `V-STR-23` **backpressure**'s situation exactly.
3. **token counting** — the phrase appears inside `V-OUT-06`'s definition ("Layer 1's only obligation is that token counting is discoverable and optional"), undefined, and it names the milestone's central capability. Undefined, it collapses into `V-MET-09` **usage** — which is a report about an output where this is a question about an input.

All three are provider-surface terms and take the next free `V-PRV` ordinals. Per AI-01 § 9 rule 2 they are appended to the register in this pull request, and each definition defers its substance to AI-03 the way `V-PRV-08` defers the mechanism.

## 12. Out of scope for this change

- **The declaration of anything.** AI-20 declares the provider contract and the optional contracts; this decision states what they must express and what their documentation must say.
- **Suite cases.** AI-23 writes them. This decision supplies the marking rule and the list they are marked against.
- **Which optional capabilities the first vendor has.** AI-24.1 item 4 records that, after the vendor is chosen. This decision cannot know it and must not guess.
- **Whether the first adapter emits reasoning.** AI-29.0's, explicitly. This decision makes both answers legal, which is the whole of its contribution there.
- **The failure taxonomy.** AI-19's. This decision requires typed failures; it defines no category.
- **Compaction, estimates, and what a consumer does with an absence.** Layer 2's (`V-OUT-06`). Layer 1 states the absence and supplies no substitute.
- **The record's serialised form.** AI-23.6 emits it and AI-40.2 publishes it; this decision sketches the shape and the outcome set, not a format.

## 13. Open questions carried into the proposal

1. Are the five required capabilities exactly right, and what precision does each need so that no adapter is failed for the wrong reason? → § 5; each carries one.
2. Is the optional list three or more? → § 6; exactly three, closed, with a stated amendment route.
3. Is honoring cache-boundary markers a capability or an adapter-local mapping obligation? → § 6.3; a capability, and the breakpoint cap is why.
4. How does AI-23 mark a case that exercises no listed capability? → § 8; the default-required rule.
5. What are the five open sub-questions inside the inherited discovery mechanism? → § 9, including the wrapping hazard no source document mentions.
6. What is in the record, and what makes it total? → § 10.
7. Which terms are appended to AI-01? → § 11; three.

## 14. Evidence gate for this milestone

AI-03.1 is a `[decision]` leaf. doc 0002's global evidence gate — recorded green `make test` in `backend/agent/` — binds behavior and guard leaves. A decision leaf closes when "the decision artifact answers every listed question and is merged". This change writes no Go, touches nothing under `backend/`, and runs no test command. Its verification is inspection against the closing checklist, recorded in `tasks.md`.
