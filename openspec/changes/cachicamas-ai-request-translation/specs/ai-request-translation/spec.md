# Spec — request translation to the OpenAI-compatible wire shape

> **Milestone**: AI-26 — Translate normalized requests to wire requests (doc 0002, Wave 4 "Build") · **Nodes**: AI-26.1 … AI-26.7 `[leaf]`, AI-26.8 (`26.8.1 [decision]`, `26.8.2 [leaf]`)
> **Introduced by**: `openspec/changes/cachicamas-ai-request-translation/`
> **Status**: **change-folder delta** — promoted to `openspec/specs/ai-request-translation/spec.md` at archive
> **Project**: cachicamas (witsaba) · **Target**: the OpenAI-compatible adapter package AI-25 creates under `backend/agent/src/ai/`. *Its file decomposition, type shapes, the inventory mechanism's implementation and the fixture harness structure are **design's**; this spec constrains only observable behaviour.*
> **Requirement IDs**: `R-ART-0NN` · **Scenario IDs**: `S-ART-0NN`
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable, either by a test runnable under `go test -race` or as an explicit **review obligation** — a claim discharged by reading landed prose, a diff or a citation rather than by executing code. Review obligations are marked `*(review)*` on the scenario line. The distinction is operational, not cosmetic: `NFR-ART-F`'s red→green evidence rule applies to the **runnable** set only, because a prose-inspection scenario has no red phase to record. **Counts: 21 requirements, 89 scenarios — 72 runnable, 17 `*(review)*`.**
> **Binding predecessors, cited by identifier and exercised, never modified**:
> [`ai-cache-breakpoints`](../../../../specs/ai-cache-breakpoints/spec.md) (AI-11.3) — the advisory-marker contract this spec supplies the wire-level proof of;
> [`ai-content-parts`](../../../../specs/ai-content-parts/spec.md) (AI-06/AI-07) — the sealed, externally readable part surface;
> [`ai-reasoning-content`](../../../../specs/ai-reasoning-content/spec.md) (AI-07) — the three reasoning states;
> [`ai-model-request`](../../../../specs/ai-model-request/spec.md) and [`ai-request-extension-points`](../../../../specs/ai-request-extension-points/spec.md) (AI-10/AI-12) — the neutral request surface and the escape hatch;
> [`ai-tool-declarations`](../../../../specs/ai-tool-declarations/spec.md), [`ai-tool-messages`](../../../../specs/ai-tool-messages/spec.md) (AI-16 … AI-18);
> [`ai-provider-errors`](../../../../specs/ai-provider-errors/spec.md) (AI-19) — the taxonomy every refusal in this spec uses;
> [`ai-validation-errors`](../../../../specs/ai-validation-errors/spec.md) (AI-04) — deliberately **not** used here (see `NFR-ART-E`);
> [`ai-provider-client`](../ai-provider-client/spec.md) (AI-25) — the constructed value this translation is built alongside; its construction contract is not touched.
> **Depends on**: AI-24's merged decision (§§ 6, 7, 8, 12, 13.1), AI-25 (in flight) · **Blocks**: AI-28 … AI-32, AI-38

---

## Purpose

This is the single place where Layer 1's neutral vocabulary meets one vendor's schema, and therefore the single place where a caller-expressible feature can disappear without anyone noticing. Three failure modes this spec exists to prevent are not hypothetical:

1. **Silent loss is the default.** A translator meeting an input it has no wire field for will, unless designed otherwise, emit a body missing it and succeed. The caller then gets a plausible answer computed from less than it asked for.
2. **Determinism cannot be proven the obvious way.** Go re-randomizes map-range start per range-statement *execution*, so "translate twice, compare bytes" can pass by luck. A map-ordering leak here produces **no failure at all** — only a broken vendor cache prefix and a roughly tenfold input bill.
3. **A legal-shaped omission.** Leaving `stream_options.include_usage` unset yields an empty-but-legal `usage` that misreads as `CAP-R-03 = absent` — illegal for a required capability, and an adapter defect wearing a vendor limitation's clothes (AI-24 § 13.1).

## Definitions used by this spec

- **Translation** — the pure act of producing wire bytes from a neutral request. It takes no client, opens no socket and reads nothing ambient.
- **A wire body** — the complete serialized request payload translation produces.
- **An expectation** — the exact expected wire bytes, **checked into source as an inline literal** and compared verbatim. This adapter uses no golden-file tree and no `-update` flag; see `NFR-ART-D`.
- **A twin comparison** — translating a request and an otherwise-identical request differing only in the feature under test, and asserting the two wire bodies are **byte-identical**. It is how "dropped whole" and "ignored whole" are proven.
- **A cross-run proof** — comparing translation output against a checked-in expectation under **independent `go test` process invocations**, each with an independently seeded map hasher. A same-process double-translate is a **pin**, never the proof.
- **A refusal** — translation failing with AI-19's `PreStreamFailure` carrying `ErrUnsupportedCapability` (`FailureCategoryUnsupportedCapability`, V-FAIL-08). No new sentinel is introduced.
- **A wire-shape claim** — a statement about the vendor's accepted request schema. Four are uncited or partially cited; `R-ART-001` governs them.
- **The feature inventory** — the enumeration of every neutral request feature a caller can express, mechanically derived rather than hand-listed.
- **A policy entry** — an inventory feature's recorded disposition: **translate**, **deliberately drop**, or **refuse**. There is no fourth value, and no feature may lack one.
- **A review obligation** — a scenario discharged by reading landed prose, a diff or a citation. Marked `*(review)*`; it carries no red phase.

## Delta status against existing specs

`openspec/specs/` was re-read at spec time. It carries no request-translation, wire-shape or feature-policy capability. `ai-request-translation` is **new**: this delta contains **ADDED requirements only** — no MODIFIED, no REMOVED and no RENAMED requirement. In particular `ai-cache-breakpoints` is **exercised, not modified**: AI-11.3 already states that an adapter for an auto-caching provider is conformant while ignoring markers entirely; `R-ART-006` supplies the wire-level proof of that existing claim.

## Amendments already merged upstream — not re-specified here

AI-24's apply landed three dated blockquotes in doc 0002. Each removes a branch's **subject** without superseding its text; each remains in force for a future adapter. This spec states only the behaviour that remains: AI-26.2 item 3 (marker-cap refusal → no cap exists), AI-26.5 item 2 (synthetic identifier minting → the vendor assigns), AI-26.7 item 2 (mandatory output limit → optional). A never-firing conditional is not a superseded claim.

---

## Cross-cutting — the wire-shape citation gate

### R-ART-001 — *(non-negotiable)* An uncited wire-shape claim BLOCKS its node; it is a defect, not an assumption

Four wire-shape claims are uncited or only partially cited. Each MUST be backed by **authoritative vendor documentation, cited in the landed artifact by title and locator**, **before** any node whose observable behaviour depends on it specifies that behaviour as fact. A node built on an uncited claim is a **defect**, not a tolerated inference: a confidently wrong wire detail propagates into every downstream milestone's fixtures, which is why this is the milestone's highest risk.

The four claims, with their provenance status recorded so a later reader cannot mistake inference for fact:

| # | Claim | Status carried in from the proposal |
|---|---|---|
| 1 | System instruction is a `messages` entry versus a top-level field, **and** whether its role is `system` or `developer` | **Fully uncited.** AI-24 §§ 6, 7, 8, 12 do not address it |
| 2 | Tool-call arguments are a JSON-encoded **string**, not a nested object | **Partially confirmed.** AI-24 § 7 settles the **response** side (partial JSON *string* fragments concatenated in `index` order); **request-side symmetry is still inference** and needs its own citation |
| 3 | `tools[].function.parameters` carries the neutral schema **verbatim** | **Fully uncited.** Load-bearing: the neutral schema bytes are byte-exact and MUST never be re-marshalled |
| 4 | The dialect does **not** enforce strict user/assistant role alternation | **Uncited.** AI-24 § 6's distinct `role: "tool"` row is *indicative against* strict alternation but **not dispositive** — it does not establish that two **consecutive same-role** messages are accepted, which is exactly what `R-ART-009` turns on |

The charter's gate map — claim 1 → AI-26.3, claim 2 → AI-26.2, claim 3 → AI-26.5, claim 4 → AI-26.4 — MUST be honoured as the minimum. It is **not** a ceiling: a claim additionally blocks **every** node whose observable behaviour depends on it, which for claims 1–4 is at least AI-26.2, AI-26.5, AI-26.4 and AI-26.3 respectively by dependency. Where the charter map and the dependency map disagree, the **earlier** of the two blocks.

Any claim that documentation **contradicts** MUST halt the node and be escalated, not reconciled locally.

#### Scenarios

- **S-ART-001** *(review)* — Given each of the four claims, when the landed artifact is read before its node's slice merges, then each carries an authoritative vendor citation with title and locator, and none is discharged by an in-repo restatement or by another SDD artifact quoting itself.
- **S-ART-002** *(review)* — Given claim 2, when its citation is read, then it establishes the **request-side** shape specifically; a response-side citation alone does not discharge it.
- **S-ART-003** *(review)* — Given claim 4, when its citation is read, then it establishes that two **consecutive same-role** messages are accepted; the distinct `role: "tool"` row is recorded as indicative and explicitly **not** accepted as the citation.
- **S-ART-004** *(review)* — Given a node whose claim is still uncited at slice-open time, when its slice is inspected, then it did not proceed — no expectation encoding that claim was authored, and the blockage is recorded rather than silently waived.
- **S-ART-005** *(review)* — Given a claim that vendor documentation contradicts, when the contradiction is found, then the node halted and the contradiction was escalated, rather than being resolved inside the slice.

---

## AI-26.1 — Wire skeleton `[leaf]`

### R-ART-002 — A minimal request translates to an exactly specified wire body

A neutral request carrying one user text message MUST translate to a wire body that equals its checked-in expectation **byte for byte**. The body MUST carry the model identity taken from the neutral request and MUST NOT introduce any field the policy of `R-ART-021` does not account for.

#### Scenarios

- **S-ART-006** — Given a request with one user text message, when it is translated, then the wire bytes equal the checked-in expectation exactly, with no additional, reordered or renamed field.
- **S-ART-007** — Given two requests differing only in model identity, when each is translated, then each body carries its own model value and the bodies differ in that field alone.
- **S-ART-008** — Given a deliberately mutated implementation that injects one extra wire field, when `S-ART-006` runs, then it fails — the comparison is byte-exact, not subset-shaped.

### R-ART-003 — Determinism is proven across process runs, not within one

Translating the same neutral request MUST produce **identical bytes across independent `go test` process invocations**. The proof MUST be comparison against a checked-in expectation. A same-process double-translate MUST be present as a **pin** and MUST NOT be presented, described or documented as the proof, because Go re-randomizes map-range start per range-statement execution and a same-process check can therefore pass by chance.

Serialization-time map-key ordering MUST NOT be relied on as the guarantee: a nondeterminism leak occurs during **slice construction**, before serialization, where key sorting cannot reach it.

#### Scenarios

- **S-ART-009** — Given the full expectation set, when the suite is invoked as an independent `go test` process, then every expectation matches, and this holds on repeated independent invocations.
- **S-ART-010** — Given one request, when it is translated twice within one process, then the two bodies are identical — recorded as a pin.
- **S-ART-011** — Given a deliberately mutated implementation that builds an intermediate map before flattening to a slice, when the suite runs across repeated independent invocations, then at least one invocation fails — the cross-run proof catches what the in-process pin can miss.
- **S-ART-012** *(review)* — Given the landed determinism test and its documentation, when a reader looks for which check is the proof, then the cross-run expectation is named as the proof and the in-process double-translate is explicitly labelled a pin.

### R-ART-004 — No credential appears anywhere in the expectation surface, and coverage grows automatically

No credential-shaped value MUST appear anywhere in the adapter package's expectation surface. Enforcement MUST be **one mechanical scan** over that surface as raw bytes, structured so that expectations added by **later nodes are covered without editing the scan**.

The scan MUST match **credential-shaped sentinels only**. Endpoint hosts and model identifiers are legitimate content and MUST NOT be flagged: a scan that reports them trains reviewers to ignore it, which is worse than no scan.

#### Scenarios

- **S-ART-013** — Given the landed expectation surface, when the scan runs as part of the suite, then it reports no credential-shaped value.
- **S-ART-014** — Given a credential-shaped sentinel added to a new expectation authored after the scan was written, when the suite runs, then the scan fails **without** having been edited to know about that expectation.
- **S-ART-015** — Given a credential-shaped sentinel placed in test setup rather than inside an expectation literal, when the scan runs, then it still fails — the scan's subject is the whole surface, not only expectation literals.
- **S-ART-016** — Given expectations containing an endpoint host and a model identifier, when the scan runs, then it stays green — legitimate content is not flagged.
- **S-ART-017** — Given a scan failure, when its message is read, then it names the file and the matched sentinel class, so a reviewer can act without re-deriving it.

---

## AI-26.2 — System segments and cache markers `[leaf]`

### R-ART-005 — Ordered system segments render preserving order and content

A system instruction's ordered segments MUST render on the wire preserving both their **relative order** and their **textual content**, unmodified. No segment MUST be dropped, merged, reordered, trimmed or re-encoded.

#### Scenarios

- **S-ART-018** — Given a system instruction of three distinct ordered segments, when the request is translated, then the wire body equals its expectation, carrying all three in the caller's order with content unchanged.
- **S-ART-019** — Given two requests whose segments are identical in content but reversed in order, when each is translated, then the two bodies differ — order is genuinely carried, not incidentally equal.
- **S-ART-020** — Given a request with **no** system instruction, when it is translated, then the wire body equals its expectation and contains no empty or placeholder system entry.

### R-ART-006 — Cache-boundary markers are dropped whole, proven by a byte-identical twin

The vendor caches automatically and exposes no client-supplied cache annotation. Every cache-boundary marker on a request — in **any** region of the neutral cascade — MUST therefore be **dropped whole**: it MUST NOT render as any wire field, any hint, any comment or any ordering difference. This exercises AI-11.3's advisory contract; it does not modify it.

Proof MUST be a **twin comparison**: a marked request and its unmarked twin, built by omitting every marker call and changing nothing else, MUST translate to **byte-identical** bodies.

#### Scenarios

- **S-ART-021** — Given a request carrying cache-boundary markers and its unmarked twin, when both are translated, then the two wire bodies are byte-identical.
- **S-ART-022** — Given markers placed across every region the neutral cascade admits, when the twin comparison runs for each, then every pair is byte-identical — no region leaks a marker.
- **S-ART-023** — Given a deliberately mutated implementation that renders a marker as any wire artefact, when `S-ART-021` runs, then it fails.
- **S-ART-024** *(review)* — Given the landed package documentation, when a reader looks for why markers vanish, then automatic vendor caching is stated as the reason and AI-11.3's advisory contract is cited, so the drop reads as conformance rather than as loss.

---

## AI-26.3 — Messages and content parts `[leaf]`

### R-ART-007 — Every readable content-part variant translates, and none translates by accident

Every content-part variant a caller can express and this vendor can receive MUST have a specified wire rendering, each proven against its own checked-in expectation. A variant the vendor cannot receive MUST take the refusal path of `R-ART-015`/`R-ART-021`, never a silent omission.

#### Scenarios

- **S-ART-025** — Given one request per translatable part variant, when each is translated, then each body equals its own expectation exactly.
- **S-ART-026** — Given a message mixing several part variants, when it is translated, then every part appears, each rendered as its single-variant expectation renders it.
- **S-ART-027** — Given the enumerated part-kind vocabulary, when the variant coverage set is compared against it, then every kind is accounted for by either an expectation or a recorded refusal — none is unaddressed.

### R-ART-008 — Message order and intra-message part order are preserved exactly

The order of messages within the request, and the order of parts within each message, MUST be preserved exactly on the wire. No reordering, stable-sorting, grouping or normalization MUST occur.

#### Scenarios

- **S-ART-028** — Given a multi-turn request whose messages are in a distinctive order, when it is translated, then the wire messages appear in that same order, matching the expectation.
- **S-ART-029** — Given a message whose parts are in a distinctive order, when it is translated, then the wire parts appear in that same order.
- **S-ART-030** — Given two requests differing only by a swap of two adjacent messages, when both are translated, then their bodies differ — order preservation is observable, not vacuously satisfied.

### R-ART-009 — Consecutive same-role messages are NOT merged, and this is a decision, not an omission

Two consecutive messages carrying the same role MUST render as **two distinct wire message objects**, in the caller's order. Translation MUST NOT coalesce, concatenate or otherwise rewrite the caller's message structure. The dialect does not require alternation, so merging would be unrequested rewriting of the caller's input.

This requirement is **gated on claim 4 of `R-ART-001`** and MUST NOT be specified as fact before that claim is cited.

#### Scenarios

- **S-ART-031** — Given two consecutive same-role messages, when the request is translated, then the wire body carries two distinct message objects in order, matching the expectation.
- **S-ART-032** — Given three consecutive same-role messages with distinguishable content, when translated, then three distinct wire messages appear and no content is concatenated across them.
- **S-ART-033** — Given a deliberately mutated implementation that merges consecutive same-role messages, when `S-ART-031` runs, then it fails.
- **S-ART-034** *(review)* — Given the landed package documentation, when a reader looks for why no merging happens, then it is stated as a deliberate decision with its reason, not left as an unexplained absence.

---

## AI-26.4 — Tools, deterministically `[leaf]`

### R-ART-010 — Tool declarations translate with name, description and schema byte-faithful

Each declared tool MUST translate carrying its name, its description and its schema. The schema bytes MUST pass through **unmodified** — never re-marshalled, re-indented, key-reordered or otherwise normalized, because re-marshalling silently changes bytes the vendor's cache prefix depends on.

This requirement is **gated on claim 3 of `R-ART-001`**.

#### Scenarios

- **S-ART-035** — Given a tool whose schema carries a distinctive key order and distinctive whitespace, when the request is translated, then those schema bytes appear on the wire unchanged, matching the expectation.
- **S-ART-036** — Given a tool with name and description set, when translated, then both appear verbatim in the expectation's positions.
- **S-ART-037** — Given a deliberately mutated implementation that round-trips the schema through decode and re-encode, when `S-ART-035` runs, then it fails.

### R-ART-011 — Tool ordering is deterministic across process runs, proven in a fresh process

Tool declarations MUST appear on the wire in the **caller's declaration order**, and that ordering MUST be identical **across independent process invocations**. A same-process double-translate is **insufficient** as proof and MUST NOT be presented as one.

The failure mode this closes is silent: a map-iteration leak here produces a valid body, no error and no test failure, while invalidating the vendor's cache prefix on every request.

#### Scenarios

- **S-ART-038** — Given a request declaring several tools in a distinctive order, when the suite runs as an independent process, then the wire order matches the checked-in expectation, and this holds across repeated independent invocations.
- **S-ART-039** — Given two requests declaring the same tools in different orders, when each is translated, then the wire orders differ accordingly — caller order is carried, not sorted.
- **S-ART-040** — Given a deliberately mutated implementation that builds tools through an intermediate map keyed by name, when the suite runs across repeated independent invocations, then at least one fails.
- **S-ART-041** *(review)* — Given the landed tool-ordering test and its documentation, when a reader looks for the proof, then the fresh-process expectation comparison is named as the proof and the silent cost of a leak (a broken cache prefix, not an error) is stated.

---

## AI-26.5 — Tool results and identifiers `[leaf]`

### R-ART-012 — A tool result translates into the vendor's distinct tool-role message shape

A neutral tool result MUST translate into **one distinct wire message carrying the vendor's tool role** — never a block nested inside a user-role message and never a nested object on another message. This shape is **given** by AI-24 § 6 and is not re-derived here.

#### Scenarios

- **S-ART-042** — Given a request carrying one tool result, when translated, then the wire body contains exactly one message in the vendor's tool-role shape, matching the expectation.
- **S-ART-043** — Given a request carrying several tool results, when translated, then one distinct tool-role message appears per result, in the caller's order.
- **S-ART-044** — Given a tool result recorded as failed, when translated, then the failure disposition is carried per the expectation and is not silently rendered as a success.

### R-ART-013 — The wire tool-call identifier is always exactly the neutral identifier; none is ever minted

The identifier on every wire tool call and every wire tool result MUST be **byte-identical** to the neutral call's identifier. Translation MUST NEVER generate, derive, normalize, truncate or otherwise invent an identifier. The synthetic-minting branch has **no subject** for this vendor, and this requirement is its recorded no-op pin rather than a separate empty test.

#### Scenarios

- **S-ART-045** — Given tool calls with distinctive identifiers, when the request is translated, then every wire identifier equals its neutral identifier byte for byte.
- **S-ART-046** — Given an identifier with an unusual but valid shape, when translated, then it survives verbatim — no normalization is applied.
- **S-ART-047** — Given a deliberately mutated implementation that regenerates identifiers, when `S-ART-045` runs, then it fails.

### R-ART-014 — Result-to-call correlation survives an interleaved multi-call turn

For a turn issuing multiple tool calls whose results arrive interleaved, each wire tool-result message MUST correlate to its originating call by identifier, and the correlation MUST survive translation intact. No positional, index-based or ordinal-based correlation MUST be substituted for identity.

This requirement is **gated on claim 2 of `R-ART-001`** where argument encoding is observable in its expectations.

#### Scenarios

- **S-ART-048** — Given a turn with three tool calls and their three results supplied in an order different from the call order, when translated, then each wire result names its own call's identifier, matching the expectation.
- **S-ART-049** — Given a deliberately mutated implementation that correlates by position, when `S-ART-048` runs, then it fails — interleaving is what makes the substitution detectable.
- **S-ART-050** — Given two calls to the **same** tool name with different identifiers, when translated, then their results remain distinguishable and each carries its own identifier.

---

## AI-26.6 — Reasoning replay refuses `[leaf]`

### R-ART-015 — *(non-negotiable)* A request carrying reasoning content REFUSES, in every state and at every position

WHEN a neutral request carries a reasoning content part — in **any** of its reasoning states, at **any** position, in **any** message — translation MUST **fail** with a **refusal** as defined above: AI-19's `PreStreamFailure` carrying `ErrUnsupportedCapability`. It MUST NOT drop the part, MUST NOT render it as text, and MUST NOT route it through the caller-owned provider-extension channel.

**The request is neutrally valid.** AI-10 and AI-12 accept it; what fails is the **provider's expressiveness**, not the caller's contract. This is why the refusal uses AI-19's taxonomy and **not** AI-04's validation taxonomy, which AI-25 correctly uses for construction-time faults. AI-03 § 10.4 settles the classification: *unsupported capability is a request-time failure and is not an absent optional capability.* The two are different failure classes and MUST NOT be read as drift; `NFR-ART-E` records the distinction so a later reader cannot mistake it for one.

**Consequence, stated so it is not later misread as a defect: Layer 2 MUST strip reasoning parts before replaying multi-turn history to this adapter.** That is the consumer-owned fallback AI-03 standing rule 4 anticipates — not a gap, not a defect, and not this milestone's work. A transcript captured from a reasoning-emitting provider and replayed here **fails hard**, which is acceptable for v1 **only because it is written down**.

This requirement does **not** decide the capability verdict; AI-29.0 owns it.

#### Scenarios

- **S-ART-051** — Given a request whose assistant message carries a reasoning part, for each reasoning state in turn, when it is translated, then translation fails with a `PreStreamFailure` carrying `ErrUnsupportedCapability`, and no wire body is produced.
- **S-ART-052** — Given reasoning parts placed first, last and mid-message across several messages, when each is translated, then every position refuses identically — position does not change the outcome.
- **S-ART-053** — Given a refusal, when the error is inspected, then it names reasoning content as the unsupported feature, so a caller can act without reading this spec.
- **S-ART-054** — Given the merged change, when the module's exported sentinel set is enumerated and compared against the set before the change, then it is identical — **zero new sentinels**.
- **S-ART-055** — Given a deliberately mutated implementation that drops the reasoning part and succeeds, when `S-ART-051` runs, then it fails.
- **S-ART-056** *(review)* — Given the landed package documentation, when a reader looks for reasoning replay, then it states plainly that a reasoning-bearing transcript fails hard here, names consumer-side stripping as the remedy, and records the duty as routed to AI-40 rather than as an unfixed defect.

---

## AI-26.7 — Options, limits and the escape hatch `[leaf]`

### R-ART-016 — Every neutral generation option either maps or fails explicitly

Every neutral generation option a caller can set MUST either map to a specified wire field or take the refusal path of `R-ART-021`. **Silently absent is forbidden.** An option's presence flag MUST be respected: an unset option MUST NOT render as a zero value, and a deliberately zero-valued option MUST NOT be treated as unset.

#### Scenarios

- **S-ART-057** — Given one request per settable generation option, when each is translated, then each body equals its expectation with that option's wire field present and carrying the caller's value.
- **S-ART-058** — Given an option set to a zero value and the same option left unset, when both are translated, then the two bodies differ — absence and zero are distinguished, not conflated.

### R-ART-017 — *(non-negotiable)* Every request sets the in-stream usage opt-in, asserted positively

Every wire body this adapter produces MUST set the dialect's in-stream usage opt-in (`stream_options.include_usage: true`). This obligation is assigned to this node **by name** in AI-24 §§ 8 and 13.1.

It MUST be asserted **positively**, on **every** expectation, not inferred from the absence of an error: left unset, the vendor returns an empty-but-legal `usage` that misreads as `CAP-R-03 = absent` — illegal for a required capability, and an adapter defect wearing a vendor limitation's clothes. There is no request shape exempt from this.

#### Scenarios

- **S-ART-059** — Given any request this spec's expectations cover, when it is translated, then the wire body carries the usage opt-in set to true.
- **S-ART-060** — Given the complete expectation set, when every expectation is walked, then **each one** carries the opt-in — the assertion is total, not sampled on one representative case.
- **S-ART-061** — Given a deliberately mutated implementation that omits the opt-in, when the suite runs, then it fails on the positive assertion rather than passing on the absence of an error.

### R-ART-018 — Omitting the output-token limit leaves the field explicitly ABSENT

The dialect's output-token limit is optional for this vendor, so the mandatory-default branch is a **deliberate no-op**. WHEN a request omits the neutral max-output-tokens option, the wire body MUST **omit** the field entirely. Absence MUST be asserted as **explicit absence**, not as equality against some default value — the honest pin for a dead branch.

#### Scenarios

- **S-ART-062** — Given a request omitting the max-output-tokens option, when it is translated, then the wire body contains no output-limit field at all, asserted as absence.
- **S-ART-063** — Given a request setting that option, when it is translated, then the field is present and carries the caller's value.
- **S-ART-064** *(review)* — Given the landed package documentation, when a reader looks for the output-limit posture, then it states that the mandatory-default branch is intentionally dead for this vendor and why, rather than leaving an unexplained absence a later contributor would "fix".

### R-ART-019 — The adapter's own extension namespace merges; every foreign namespace is ignored whole

Provider-extension values in the adapter's **own reserved namespace** MUST merge into the wire body. Values in **every other namespace** MUST be enumerated but never inspected and never rendered — **ignored whole**.

"Ignored whole" MUST be proven by the same twin-comparison shape as `R-ART-006`: attaching a foreign-namespace extension MUST leave the wire body **byte-identical**. The reserved namespace value MUST be read from AI-25's landed artifact, never re-invented here.

#### Scenarios

- **S-ART-065** — Given a request carrying an extension in the adapter's own namespace and its twin without it, when both are translated, then the bodies differ and the first matches its expectation with the extension's bytes merged.
- **S-ART-066** — Given a request carrying a foreign-namespace extension and its twin without it, when both are translated, then the two bodies are **byte-identical**.
- **S-ART-067** — Given several foreign namespaces attached at once, when translated, then the body remains byte-identical to the twin — the rule is per-namespace and total, not first-match.
- **S-ART-068** — Given a deliberately mutated implementation that renders any foreign namespace, when `S-ART-066` runs, then it fails.

---

## AI-26.8 — Unsupported-feature policy

### R-ART-020 — The feature inventory is mechanically derived and stays honest as the surface grows `[decision]`

The inventory MUST enumerate **every** neutral request feature a caller can express. It MUST be derived **mechanically**, and MUST NOT be a hand-maintained list, because a hand-maintained list silently omits whatever is added after it was written.

Two derivation surfaces exist and both MUST be covered:

1. Neutral **closed vocabularies** that are already exported and runtime-walkable — part kinds, reasoning states, tool-choice modes, cache regions and **roles** — MUST be enumerated through their own enumerators. Reflection MUST NOT be used; the sealed design already provides a first-class registry and a reflective walk would be strictly worse.
2. The neutral **generation-option and extension surface**, which is plain fields rather than a closed vocabulary, MUST be covered by the repository's existing source-scan-against-a-policy-table idiom, cross-checking the declared option constructors against the policy table.

Adding a feature to the neutral surface **without** a policy entry MUST fail the guard — the same failure mode as the repository's existing "declared but unregistered" guards. The mechanism MUST be recorded as a decision, with its ground truth named, so a later reader knows what the guard compares against and why it is not the package compared against itself.

#### Scenarios

- **S-ART-069** — Given the landed inventory, when it is enumerated, then it lists every member of each of the five closed vocabularies plus every declared generation option and the extension surface.
- **S-ART-070** — Given a new member added to any closed vocabulary without a policy entry, when the guard runs, then it fails and names the unaccounted member.
- **S-ART-071** — Given a new generation-option constructor added to the neutral surface without a policy entry, when the guard runs, then it fails and names it.
- **S-ART-072** — Given the guard's ground truth, when it is inspected, then it is derived from the neutral package's own declarations rather than from a list the adapter maintains about itself.
- **S-ART-073** *(review)* — Given the landed decision record, when a reader looks for how the inventory stays honest, then both derivation surfaces are named with their mechanisms and the reason reflection is rejected.

### R-ART-021 — *(non-negotiable)* The policy is total, and silent dropping is the one forbidden outcome

Every feature in the inventory MUST carry exactly one policy entry: **translate**, **deliberately drop**, or **refuse**. A feature with no entry MUST fail the guard. A **deliberately drop** entry MUST carry a recorded reason and MUST be proven by a twin comparison, so that dropping is always a decision with evidence rather than an omission.

WHEN a request expresses something this vendor cannot receive, translation MUST fail with a **refusal naming the feature**. Failing anonymously is insufficient; **silently dropping is forbidden outright** and is the exit check this milestone exists to enforce.

The policy MUST grow **automatically** with the inventory: the walk MUST be driven by the inventory, never by a parallel list, so a feature added to the neutral surface cannot escape it.

#### Scenarios

- **S-ART-074** — Given the inventory, when the exhaustive walk runs, then every feature resolves to exactly one of the three policy values and none is unresolved.
- **S-ART-075** — Given each feature whose policy is **refuse**, when a request expressing it is translated, then translation fails with a `PreStreamFailure` carrying `ErrUnsupportedCapability`, and the error **names that feature**.
- **S-ART-076** — Given each feature whose policy is **deliberately drop**, when the marked request and its twin are translated, then the bodies are byte-identical and the recorded reason exists.
- **S-ART-077** — Given a deliberately mutated implementation that drops a **refuse**-policy feature silently and succeeds, when the walk runs, then it fails.
- **S-ART-078** — Given a feature added to the inventory with no policy entry, when the walk runs, then it fails — the walk is inventory-driven, not list-driven.
- **S-ART-079** — Given every refusal this spec produces, when their error values are compared, then all use the same category uniformly across AI-26.6 and AI-26.8.2, and no new sentinel was added.

---

## Non-functional requirements

### NFR-ART-A — Dependency purity

This change MUST add no module dependency. `backend/agent/go.mod` MUST still declare **zero** `require` directives, and both AI-00 import guards MUST still pass. Any dependency is a **hard blocker**, not a tradeoff.

- **S-ART-080** — Given the change merged, when `go.mod` is read and both AI-00 import guards run, then it declares no require and both guards pass.

### NFR-ART-B — The neutral surface is read, never changed

This change MUST NOT alter any behaviour, declaration or signature in `package ai`. Every value translation reads MUST be reached through an already-public accessor. Every AI-00 … AI-25 test MUST pass identically with or without this change.

- **S-ART-081** *(review)* — Given the merged diff, when every path under the neutral package is inspected, then none is modified.
- **S-ART-082** — Given the merged change, when the full suite runs, then every predecessor milestone's tests pass unchanged.

### NFR-ART-C — Translation is pure

Translation MUST perform no I/O, MUST take no client, MUST open no socket, MUST read nothing ambient, and MUST NOT mutate the neutral request it is given. Every proof in this spec MUST be a value comparison; no scenario MUST be discharged by a network assertion.

- **S-ART-083** — Given a neutral request, when it is translated, then the request compares equal to an independently built copy of itself afterwards — translation mutated nothing.
- **S-ART-084** *(review)* — Given this milestone's tests, when they are enumerated, then none constructs a client, a test server or a socket.

### NFR-ART-D — Expectations are inline literals; there is no regeneration flag

Expectations MUST be **inline literals checked into source**. This adapter MUST NOT introduce a golden-file tree or an `-update`-style regeneration flag. The reason is recorded rather than assumed: a regeneration flag makes blessing a wrong change trivial, which is precisely the drift this milestone exists to prevent, and an inline expectation places the case and its expected bytes in one review hunk. What proves cross-run determinism is that the expectation is **checked into source and compared in a fresh process**, not where it lives. This convention is adapter-wide, converged with AI-27 independently.

- **S-ART-085** — Given the landed change, when the adapter package is inspected for a golden tree or a regeneration flag, then neither exists.
- **S-ART-086** *(review)* — Given the landed test documentation, when a reader looks for why, then the reason above is stated in place.

### NFR-ART-E — The refusal taxonomy differs from AI-25's, deliberately, and the reason is recorded

Refusals here use AI-19's `PreStreamFailure` + `ErrUnsupportedCapability`; AI-25's construction faults use AI-04's `Violation`. This is **not drift**. The distinction MUST be recorded in the landed artifact so a later reader does not "harmonize" them:

| | AI-25 construction fault | AI-26 refusal |
|---|---|---|
| Example | malformed endpoint, empty credential | request carries a reasoning part |
| Is the request valid? | no request exists yet | **yes** — neutrally valid |
| What failed | the **caller's** contract | the **provider's** expressiveness |
| Class | validation fault | capability failure |

- **S-ART-087** *(review)* — Given the landed artifact, when a reader asks why two taxonomies appear across AI-25 and AI-26, then the class distinction is stated with AI-03 § 10.4 cited, rather than left to be inferred.

### NFR-ART-F — Evidence, under the race detector, per slice

Every behaviour this spec requires MUST hold under `go test -race`. Every **runnable** scenario MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. **A scenario that cannot be made to fail first is not a scenario.** Scenarios marked `*(review)*` are review obligations with no red phase; each MUST be discharged by a recorded reviewer confirmation naming what was read, and MUST NOT be reported as a red→green item. Each slice MUST close on recorded green `make test` (`go test -race -v ./...`) from `backend/agent/` and clean `make lint`.

- **S-ART-088** *(review)* — Given `tasks.md`, when a reviewer walks the runnable scenarios, then each carries recorded red output, recorded green output and a refactor note, and each `*(review)*` scenario instead carries a recorded confirmation naming what was read.
- **S-ART-089** — Given this milestone's whole test set run repeatedly under `-race`, when results are compared, then they are identical across runs and the race detector reports nothing.

---

## Acceptance criteria

1. All four wire-shape claims carry an authoritative vendor citation **before** their node is built; an uncited claim **blocked** its node rather than proceeding on inference (`R-ART-001`).
2. A minimal request matches its byte-exact expectation, and translating the same request yields identical bytes **across independent `go test` invocations**; the in-process double-translate is present as a pin and is not presented as the proof (`R-ART-002`, `R-ART-003`).
3. No credential appears anywhere in the expectation surface, proven by one scan that automatically covers later nodes' additions; hosts and model identifiers are not flagged (`R-ART-004`).
4. Ordered system segments preserve order and content, and a cache-marked request and its unmarked twin produce **byte-identical** bodies (`R-ART-005`, `R-ART-006`).
5. Every readable part variant translates, message and part order are preserved exactly, and two consecutive same-role messages render as **two distinct wire messages** (`R-ART-007` … `R-ART-009`).
6. Tool declarations carry name, description and byte-faithful schema, and tool ordering is proven deterministic **in a fresh process** (`R-ART-010`, `R-ART-011`).
7. Tool results translate into the vendor's distinct tool-role shape, every wire identifier is byte-identical to the neutral one, none is ever minted, and correlation survives an interleaved multi-call turn (`R-ART-012` … `R-ART-014`).
8. A reasoning-bearing request **refuses** in all three states and at any position, with a `PreStreamFailure` carrying `ErrUnsupportedCapability`, **zero new sentinels**, and the Layer 2 stripping duty written down and routed to AI-40 (`R-ART-015`).
9. Every request sets the in-stream usage opt-in, asserted **positively on every expectation**; omitting the output-token option leaves the field **explicitly absent**; the adapter's own extension namespace merges and every foreign namespace leaves the body byte-identical (`R-ART-016` … `R-ART-019`).
10. The inventory is mechanically derived from both surfaces and fails when a feature gains no policy entry; the walk is inventory-driven, total, and **no expressible request feature is silently dropped** (`R-ART-020`, `R-ART-021`).
11. `go.mod` still declares zero requires, the neutral package is unmodified, translation is pure, and expectations are inline with no regeneration flag (`NFR-ART-A` … `NFR-ART-D`).
12. The refusal-taxonomy distinction against AI-25 is recorded in place (`NFR-ART-E`).
13. `make test` green under `-race` and `make lint` clean for **every slice**, with red/green evidence recorded per runnable scenario and a recorded confirmation per `*(review)*` scenario (`NFR-ART-F`).

## Left to design, deliberately

This spec constrains observable behaviour only. File decomposition, type and function shapes, the inventory guard's implementation, the expectation harness's structure, the chain slicing, and the concrete wire field names beyond those AI-24 settled are **design's**, and are not constrained here beyond the behaviour each must produce.
