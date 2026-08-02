# Spec — Layer 1 contract vocabulary

> **Milestone**: AI-01 — Record the Layer 1 contract vocabulary · **Node**: AI-01.1 `[decision]`
> **Introduced by**: `openspec/changes/archive/2026-07-31-cachicamas-ai-contract-vocabulary/`, merged in PR #95 at commit `a831c06` on 2026-07-31
> **Status**: **live** — this file *is* the register, and later milestones append to it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-AIV-0NN` · **Scenario IDs**: `S-AIV-0NN`
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005](../../../docs/adr/0005-promote-agent-stack-to-own-module.md) · [ADR 0006](../../../docs/adr/0006-resolve-skill-and-prompt-source-of-truth.md)

> [!IMPORTANT]
> **This artifact names concepts, not code.** No Go type name, field name, method name, or package identifier appears here. Each owning milestone's own SDD cycle chooses spellings — that is doc 0002's authoring constraint and AI-01's explicit out-of-scope clause. A term here is a noun phrase and a definition; it is never an API.

## Purpose

Hold the single, definitive vocabulary of Layer 1 of the cachicamas agent stack: every noun a milestone charter, an SDD, or a pull-request description from AI-02 through AI-40 may use about Layer 1, with exactly one definition, exactly one owning milestone, and its provenance.

The register is **114 terms in six categories**. It settles words, not behavior: it names the stream carrier without choosing it, names the capability standings without assigning them, names the failure vocabularies without classifying into them. Each of those is a later milestone's decision, and § 9 rule 5 states the test that keeps this file out of them.

## Status — this file is the live register, and it is appendable

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The register therefore lives **here**, in the canonical tree, and not in the archive. Three consequences, stated because the distinction is load-bearing for this artifact in a way it is not for most:

1. **This file is the register.** It carries the definitive text of every Layer 1 term. A milestone that needs a Layer 1 noun resolves it here, cites the row identifier, and — when the noun is missing — appends it here.
2. **The archived `decision.md` is the historical record of how the register was first decided**, not a second copy of it. It holds AI-01.1's argument, its closing-checklist verification, and the state of the register on the day AI-01.1 merged. It is immutable; nothing is ever added to it. It is at [`openspec/changes/archive/2026-07-31-cachicamas-ai-contract-vocabulary/decision.md`](../../changes/archive/2026-07-31-cachicamas-ai-contract-vocabulary/decision.md).
3. **Archiving the register would have frozen an artifact that AI-04 … AI-40 must still write to.** The append rule is not theoretical: it was exercised twice inside wave 0 itself, by the two milestones AI-01 blocks — AI-02.1 appended `V-STR-22` and `V-STR-23`, AI-03.1 appended `V-PRV-16`, `V-PRV-17` and `V-PRV-18`, taking the register from 109 terms to 111 to 114.

### How to amend this register

These are § 9's rules 2 and 3 in operational form, restated here so the next milestone can follow them without opening the archive.

| # | Rule |
| --- | --- |
| 1 | **Append, never invent.** A Layer 1 noun this register lacks is added *here*. It is never defined locally in a downstream milestone's SDD. |
| 2 | **Next free ordinal in its category.** `V-<CAT>-nn`, taking the next unused `nn` for that `<CAT>`. Identifiers are append-only: never renumbered, never reused, never reordered. |
| 3 | **A dated amendment blockquote under the touched category heading.** It states what was appended, by which milestone node, and *why the register lacked the term* — the last part is what makes the rule self-enforcing. |
| 4 | **No silent edit.** No existing row may be renumbered, reworded, reordered or removed by an amendment. A superseded definition keeps its identifier with its old text struck through and visible, so citations from merged charters keep resolving. |
| 5 | **Update the counts.** The per-category figures and the sum in § 10, and any identifier range that the append extends, are part of the amendment. |
| 6 | **In the same pull request that needs it.** The amendment lands with the milestone that discovered the gap, not afterwards. |

A row placed under the category heading it *belongs to* may sit physically above rows with lower ordinals. That is not a numbering error; see the second paragraph of § 4's amendment blockquote.

---

## 1. How to use this document

**If you are writing a milestone charter, an SDD, or a pull-request description for AI-02 through AI-40:** every Layer 1 noun you use should resolve to a row below. Cite the row identifier (`V-REQ-04`, `V-STR-13`, …) rather than re-paraphrasing the definition — a paraphrase is how a definition drifts.

**If the noun you need is not here:** it is appended by amendment, in the same pull request that needs it, with the next free ordinal in its category and a dated blockquote under the category heading. It is never invented locally in a downstream SDD. That rule is the whole point of the milestone; see § 9.

**Row shape.** Every term carries five fields:

| Field | Meaning |
| --- | --- |
| **Id** | `V-<CAT>-nn`. Stable and append-only. Never renumbered, never reused. |
| **Term** | A conceptual noun phrase, always with spaces. Never a Go identifier. |
| **Definition** | What the term **is**, and — where a recorded defect came from an unstated negative — what it **is not**. |
| **Owner** | Exactly one **current** doc 0002 milestone identifier. Never a range, never two. |
| **Provenance** | The doc 0001 section the concept derives from, plus the `C1`–`C4` / `G1`–`G13` identifier where the term exists to close a specific defect or gap. |

**Six categories**, ordered to match AI-01.1's closing checklist:

| Code | Category | Checklist item |
| --- | --- | --- |
| `V-REQ` | Request-side — what goes to the model | 1 |
| `V-STR` | Stream-side — what comes back, and what carries it | 2 |
| `V-MET` | Metadata — what a response says about itself | 3 |
| `V-FAIL` | Failure — two vocabularies, two delivery paths | 4 |
| `V-PRV` | Provider surface and proving apparatus | beyond the checklist, required by AI-01's acceptance criterion |
| `V-OUT` | Excluded — named, attributed, never defined | 5 |

**A note on citations.** doc 0001 and ADR 0005 were written against the retired milestone numbering and still carry identifiers that no longer mean what they say — retired AI-43 is cache breakpoints, now **AI-11**; retired AI-45 is the reasoning round-trip token, now **AI-07**; retired AI-47 is the stream carrier, now **AI-02**. Every ownership and provenance citation in this document is a **current** doc 0002 identifier, translated through doc 0002's identifier map; the three retired numbers just named are the only ones this document mentions, and each is named beside its replacement.

The hazard is not theoretical, and one case is worse than stale: doc 0001 § 3.1 assigns defect **C4** to "AI-18", and AI-18 exists today as a **different, real** milestone (tool-call delta events). A copied citation therefore points at a wrong milestone that resolves. C4 is closed by **AI-19**.

---

## 2. The two wording traps

Both are quoted verbatim from doc 0002's [Layer boundary](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#layer-boundary) section. They are restated here rather than paraphrased **because each has already caused one wrong decision**, and paraphrase is how the first one went wrong in the first place.

### Trap 1 — the tool trap

> **"Layer 1 does not know what a tool is" is too broad** — Layer 1 must understand the provider-neutral *transport representation* of tool declarations and tool calls because those cross the model API; it must not execute tools, resolve tool names, or own application behavior.

**Both halves bind this register, and they bind it in opposite directions.**

The first half is why `V-REQ-12` … `V-REQ-18` exist at all. A tool declaration and a tool call cross the model API; they are wire-facing content, and Layer 1 must be able to carry them faithfully. AI-08's charter states it directly: its goal is "the provider-neutral transport representation of the tools a model may call."

The second half is why `V-OUT-04`, `V-OUT-05`, `V-OUT-16` and `V-OUT-17` exist. Layer 1 carries the representation and does nothing with it: it does not execute a tool, does not resolve a name to an implementation, does not decide whether a call is permitted, does not confine execution, and does not know where the tool set came from. AI-08's charter puts the same fence in its out-of-scope clause: "Validating a tool's JSON schema against a schema meta-schema, executing tools, and resolving tool names — none of which is Layer 1's."

**Applying the trap to a new case.** Ask whether the thing crosses the model API. Argument bytes do — they are sent. An argument *value*, parsed and interpreted, does not: interpreting them is executing the tool. Layer 1 therefore carries argument bytes byte-exactly (`V-REQ-17`) and never parses them.

### Trap 2 — the provider-swap trap

> **"Provider swap is a config change" applies only after adapters exist** — switching between already-supported providers can be configuration-only; adding a new vendor still requires a new adapter unless it is genuinely compatible with an existing transport.

**What this binds.** The neutral vocabulary in this register is what *makes* an adapter writable; it is not a substitute for one. `V-PRV-02` (adapter) is a term precisely so that no charter can promise a vendor without promising the milestone that writes its adapter. doc 0001 § 8 records the corollary as a v1 non-goal: one adapter proven against a conformance suite is the target, and "the suite is what makes the second adapter cheap."

The escape hatch (`V-REQ-28`) is where this trap most often reappears in disguise. An escape hatch lets a provider-specific value survive to *its* adapter. It does not let a provider without an adapter be used. doc 0001 § 3.3 states the design principle the hatch encodes: "the correct response to leakage is a typed pass-through, not a wider neutral vocabulary."

---

## 3. Request-side terms — `V-REQ`

What goes to the model. Owners AI-05 … AI-12. **Closing-checklist item 1.**

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `V-REQ-01` | **role** | The closed, provider-neutral vocabulary value naming who a message is attributed to. Closed, not advisory: a value outside the vocabulary is a caller-contract failure, not an unknown passed through. Not a provider's own role string — an adapter maps between them. | AI-05 | 0001 § 3.3 rows 4, 5 |
| `V-REQ-02` | **message** | The smallest addressable unit of a transcript: one role plus ordered content. Layer 1's unit of attribution and ordering *within* a request. Not a turn, not a transcript, not a history — see `V-OUT-01`, `V-OUT-02`. | AI-05 | 0001 § 4.2 |
| `V-REQ-03` | **message identity** | The stable handle by which one message is distinguished from another within a request. Identity is Layer 1's; correlating identity across turns or across a persisted session is not. | AI-05 | 0001 § 5.2 |
| `V-REQ-04` | **content part** | One element of a message's ordered content, carrying exactly one kind of payload. Two properties are **constitutive, not incidental**: its payload is readable from outside the package that owns it, **and** no value that skipped construction — zero value, hand-rolled implementation, literal — can be valid. A definition carrying only one of the two re-creates a shipped defect. | AI-06 | 0001 § 3.1 **C1**, **C2** |
| `V-REQ-05` | **content-part kind** | The discriminator naming which payload a content part carries. The kind set is closed and exhaustively registered; every registered kind has a constructible payload. Registered kinds in v1: text, reasoning, tool call, tool result. Image and audio are named in the content vocabulary and deliberately have no producer — see `V-PRV-07` and doc 0001 § 8. | AI-06 | 0001 § 3.1 **C1**, § 8 |
| `V-REQ-06` | **content-part readability** | The property that an adapter or consumer in another package can read a part's payload back out of a constructed message. Without it, request translation is structurally impossible, not merely inconvenient. | AI-06 | 0001 § 3.1 **C2** |
| `V-REQ-07` | **content-part sealing** | The property that a value which did not pass the construction rules cannot be valid, cannot reach a message, and cannot satisfy the part contract by accident. | AI-06 | 0001 § 3.1 **C1** |
| `V-REQ-08` | **text content** | The content-part kind carrying model-visible natural-language text. The first subject the part contract is proven against. | AI-06 | 0001 § 3.1 |
| `V-REQ-09` | **reasoning content** | The content-part kind carrying a model's intermediate reasoning: a state, optional text, and an opaque round-trip token. Distinct from text content at every layer — reasoning is never rendered as, merged into, or substituted for text. | AI-07 | 0001 § 3.2, § 3.3 row 2 · **G12(b)** |
| `V-REQ-10` | **reasoning state** | The value distinguishing the shapes reasoning can take: full text, redacted, signature-only, or a provider that emitted no reasoning text at all. Exists so that "no reasoning text" is a recorded state rather than an empty string. | AI-07 | 0001 § 3.3 row 2 |
| `V-REQ-11` | **round-trip token** | An opaque provider-supplied value attached to reasoning content, which cachicamas stores and returns **byte-identically** and never parses, reformats, re-encodes, or interprets. A correctness requirement, not metadata: at least one provider signs reasoning cryptographically, and an inexact return fails multi-turn extended thinking with tool use. | AI-07 | 0001 § 3.2, § 3.3 row 2, § 6 seam 11 · **G12(b)** |
| `V-REQ-12` | **tool declaration** | The provider-neutral transport representation of one tool a model may call: its name, its description, and its schema bytes. A description of a capability being offered to the model. **Not** an executable, not a resolution target, not a permission subject — trap 1. | AI-08 | 0001 § 3.3 (register), § 7 **G6** |
| `V-REQ-13` | **schema bytes** | The tool declaration's argument schema, carried byte-faithfully. Layer 1 transports it and never validates it against a meta-schema. | AI-08 | 0001 § 3.3 |
| `V-REQ-14` | **tool set** | The ordered, deterministically iterable collection of tool declarations offered on one request. Determinism matters because the set is a cache prefix (`V-REQ-25`); a non-deterministic order silently invalidates a cache. | AI-08 | 0001 § 3.2, § 5.1 (tool-source note) · **G4** |
| `V-REQ-15` | **tool choice** | The request-level instruction constraining whether and which tool the model may call, validated against the declared tool set. A constraint expressed to the model — not a decision about whether a resulting call is allowed to run, which is `V-OUT-05`. | AI-08 | 0001 § 3.3 |
| `V-REQ-16` | **tool call** | The content-part kind carrying a model's request to invoke a tool: its identity, the tool name, and exact argument bytes. Represents an *intent to invoke*. Layer 1 never acts on it — trap 1. Some providers assign no identifier of their own, in which case an adapter mints one and keeps the mapping. | AI-09 | 0001 § 3.3 rows 1, 7 |
| `V-REQ-17` | **argument bytes** | The tool call's arguments, carried exactly as received or supplied: no re-marshalling, no key reordering, no whitespace normalization. Byte-equality is the contract, because translation and stream reassembly both depend on it. | AI-09 | 0001 § 3.3 row 1 |
| `V-REQ-18` | **tool result** | The content-part kind carrying the answer to a tool call: its correlation to the originating call, its content, and an indication of whether the tool reported failure. A *result that reports failure* is not a `V-FAIL-01` and not a `V-FAIL-05`; it is ordinary content. Providers place results differently — a block in a user-role message, a distinct role, a nested object — and each adapter maps that on the way out. | AI-09 | 0001 § 3.3 row 4 |
| `V-REQ-19` | **system-instruction segment** | One ordered, individually markable piece of the system instruction. Segmented **from birth**: a flat system instruction has nowhere to put a cache boundary, and providers place the instruction as a top-level field, a differently-named field, or a nested object. A single-segment request is the common case and stays ergonomic. | AI-10 | 0001 § 3.2, § 3.3 row 8 · **G4** |
| `V-REQ-20` | **normalized request** | The complete provider-neutral description of one model call: model identity, ordered system-instruction segments, ordered messages, the tool set and tool choice, and generation options. The one thing a provider receives. Everything it holds is readable from another package; nothing in it is a vendor wire shape. | AI-10 | 0001 § 3.1, § 4.1 |
| `V-REQ-21` | **model identity** | The neutral name of the model a request targets. A name carried to an adapter — not a catalog entry, not a price-table key, not a selection policy; those are `V-OUT-14` and `V-OUT-08`. | AI-10 | 0001 § 5.2 |
| `V-REQ-22` | **request validation** | The single check that a request satisfies every construction and composition rule, performed **once, before any I/O**, reporting through the caller-contract failure vocabulary. Validation happening after a socket is open is a different concept and a defect. | AI-10 | 0001 § 3.1 |
| `V-REQ-23` | **cache-boundary marker** | An **advisory** mark on a system-instruction segment, a tool declaration, or a message, indicating a point at which a provider may cache the preceding prefix. Advisory by contract: an adapter for an auto-caching provider ignores markers wholesale; an adapter for an explicit one honours them. A marker is never a correctness requirement. | AI-11 | 0001 § 3.2, § 3.3 row 9, § 6 seam 10 · **G4** |
| `V-REQ-24` | **breakpoint cap** | The documented maximum number of cache-boundary markers a request may carry. Exists because at least one provider enforces a small hard cap, and a request that exceeds it is a caller-contract failure rather than a silent truncation. | AI-11 | 0001 § 3.2 · **G4** |
| `V-REQ-25` | **invalidation cascade** | The stated ordering — tools, then system instruction, then messages — in which a change invalidates cached prefixes downstream of it. Readable by an adapter so it can render markers correctly; Layer 1 never measures a hit rate. | AI-11 | 0001 § 3.2 · **G4** |
| `V-REQ-26` | **generation option** | A neutral, provider-independent parameter shaping generation, carried on the request. Neutrality is the admission test: an option that exists to satisfy one provider belongs in the escape hatch instead. | AI-10 | 0001 § 3.2, § 3.3 (conclusion) · **G9** |
| `V-REQ-27` | **per-request option override** | A generation option supplied or replaced for one call without rebuilding the caller's own defaults. The mechanism by which a caller varies one call; not a session setting and not a preference store. | AI-12 | 0001 § 3.2 · **G9** |
| `V-REQ-28` | **provider escape hatch** | A typed-but-opaque, namespaced pass-through carrying a provider-specific value the neutral vocabulary deliberately does not model. It survives to its own adapter; every other adapter ignores it without needing to know it exists. Its existence is a design position: growing the neutral vocabulary once per provider quirk grows it without bound. **Not** a way to use a provider that has no adapter — trap 2. | AI-12 | 0001 § 3.2, § 3.3 (conclusion), § 6 seam 9 · **G9** |
| `V-REQ-29` | **request rebuild** | Deriving a modified request from an existing one such that the original is observably unmodified and the derived request validates independently. The mechanism a pre-request hook stands on: the hook is the last moment the outgoing request exists as data, and a hook cannot modify a request it cannot rebuild. Layer 1 supplies rebuildability; the hook itself is `V-OUT-13`. | AI-12 | 0001 § 6 seam 1 · **G9**, **G11** |

---

## 4. Stream-side terms — `V-STR`

What comes back, and what carries it. Owners AI-02, AI-09, AI-14 … AI-18. **Closing-checklist item 2.**

> **Definition order is load-bearing here.** The container nouns are defined before the content nouns, and `sequence` is defined **after** and **as a property of** `stream`. Defect **C3** — a process-global counter that made "the first event of every stream carries sequence 1" achievable only for the first stream in a process — is a direct consequence of the reverse order. Encoding the order in the vocabulary excludes the defect at the definitional level, where no test can reach it.

> **Amended 2026-07-31** — appended `V-STR-22` **carrier view** and `V-STR-23` **backpressure** to § 4.1, both owned by **AI-02**, in the pull request for `cachicamas-ai-stream-lifecycle` (AI-02.1) that needed them. Per § 9 rule 2, a milestone that needs a Layer 1 noun this register lacks appends it here rather than defining it locally. `carrier view` was needed because AI-02.1 delegates iterator ergonomics to AI-22.5 and had no noun for the delegated thing; without one, every downstream restatement says "iterator view", a phrase welded to one carrier choice. `backpressure` was needed because AI-02.1's buffering decision turns on "backpressure means waiting, never dropping", and the word already appeared *inside* `V-STR-08`'s definition without being defined — exactly the drift this register exists to prevent. No existing row was renumbered, reworded or removed; § 9 rule 3 holds. Term counts in § 10 updated accordingly.
>
> **On where appended rows sit.** Both land in § 4.1 with the container terms they belong to, which places `V-STR-22` and `V-STR-23` physically *above* `V-STR-10` … `V-STR-21` in § 4.2. That is not a numbering error. Identifiers are append-only (§ 9 rule 3) while rows are grouped by meaning, so the two orders diverge the first time a category is extended and will diverge further as later milestones append. A range written `V-STR-01 … V-STR-23` therefore denotes the identifier span, never a reading order — the alternative, filing a container term under content terms to keep the page in numeric order, would trade a real property for a cosmetic one.

> **Amended 2026-08-01** — appended `V-STR-24` **provider response identity** and `V-STR-25` **served model** to § 4.2, both owned by **AI-15**, in the pull request for `cachicamas-ai-response-events` (AI-15.1) that needed them. Per § 9 rule 2: `V-STR-19` response-start event named the event but defined neither of its two fields, and `V-REQ-21` model identity is explicitly request-side and does not cover the model that actually served a response — no stream-side row defined either noun. Both are needed so a response-start event can carry a provider's own response handle and the model that actually produced the response, kept distinct from what the request asked for. No existing row was renumbered, reworded, reordered or removed; § 9 rule 3 holds. Term counts in § 10 updated accordingly (23 → 25 stream-side, 116 → 118 total).

### 4.1 Container terms — the stream itself

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `V-STR-01` | **stream** | One provider response in progress: the bounded, ordered delivery of events for exactly one model call, from its first event to its single terminal event. A stream is a *thing that owns state* — its sequence, its buffer, its lifetime — which is what makes `V-STR-13` a property of it rather than of the process. | AI-02 | 0001 § 3.1 **C3**, § 7 **G13** |
| `V-STR-02` | **carrier** | The mechanism by which a consumer receives a stream's events at the package boundary. The concept is fixed here so that AI-02 can decide **which** carrier on its merits; doc 0002 records that the choice is genuinely free at AI-02 and only there. | AI-02 | 0001 § 3.2, § 7 **G13** |
| `V-STR-03` | **producer** | The party that creates a stream, emits its events, stamps their sequence, and closes the stream exactly once. Always the provider side. | AI-02 | 0001 § 9 (streams checklist) |
| `V-STR-04` | **consumer** | The party that receives events. It never closes the stream and never stamps a sequence. | AI-02 | 0001 § 9 |
| `V-STR-05` | **stream ownership** | The rule that the producer creates and closes the stream **exactly once**, across the completion, error and cancellation paths alike; nothing else closes it. "Exactly once" is stated for all three paths, not implied from the happy one. | AI-02 | 0001 § 9 ("Nothing closes a channel it does not own") |
| `V-STR-06` | **cancellation** | The caller-owned signal that ends a stream early. Every send selects on it, and cancellation closes the stream within bounded time. A signal, never a polled flag; a backoff waits on it rather than sleeping. | AI-02 | 0001 § 6, § 9 |
| `V-STR-07` | **abandonment** | A consumer that stops reading a stream **and never cancels it**. A documented contract violation rather than a supported mode, stated because it cannot be tested to termination. It is restated at the v1 freeze. | AI-02 | 0001 § 7 **G13** |
| `V-STR-08` | **bounded buffer** | The finite capacity between producer and consumer. Bounded by contract: an unbounded buffer converts backpressure into memory growth. Its starting capacity is AI-02's decision and is revisited with measurements later. | AI-02 | 0001 § 6, § 9 |
| `V-STR-09` | **sanctioned loss path** | The single, named, documented circumstance in which events may be dropped — a saturated buffer during cancellation drops late events and closes without a terminal. Exactly one such path exists; any other loss is a defect. | AI-02 | 0001 § 9 |
| `V-STR-22` | **carrier view** | A convenience adaptation of a stream into an iteration shape other than the decided carrier, offered **outside** the package boundary. A view over a stream the consumer already holds; it does not own the stream, never closes it, and is **never a second contract** — the boundary keeps speaking the decided carrier. *(Appended 2026-07-31 by AI-02.1.)* | AI-02 | 0001 § 7 **G13** ("expose an iterator view from the test kit for ergonomics") |
| `V-STR-23` | **backpressure** | The posture a bounded buffer takes when it is full: the producer **waits**. Waiting, never dropping — a full buffer slows a stream and never shortens it. Distinct from the `V-STR-09` sanctioned loss path, which is not backpressure but its one documented exception. Defined because the word already appeared inside `V-STR-08` undefined, and an undefined word is how "backpressure" comes to mean discarding. *(Appended 2026-07-31 by AI-02.1.)* | AI-02 | 0001 § 6, § 9 |

### 4.2 Content terms — what a stream carries

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `V-STR-10` | **event** | One indivisible observation delivered on a stream: a kind, a payload, and a sequence. The unit a consumer receives. Layer 1's event, not Layer 2's — see `V-OUT-01`. | AI-14 | 0001 § 4.3 (invariants applied at Layer 1) |
| `V-STR-11` | **event kind** | The discriminator naming what an event carries. **Derived from** the payload rather than set alongside it, so a kind cannot disagree with its contents. The kind set is closed, and every registered kind has a constructible payload. | AI-14 | 0001 § 3.1 **C4**, § 9 ("every event kind … has a payload that can actually be constructed") |
| `V-STR-12` | **payload** | The typed contents of an event. A payload-less event does not exist. A registered kind whose payload cannot be constructed by an adapter in another package is defect **C4** reintroduced. | AI-14 | 0001 § 3.1 **C4** |
| `V-STR-13` | **sequence** | The 1-based, contiguous ordinal an event carries, assigned by the producer, **per stream**. Every stream's first event carries 1; every stream is independently contiguous; concurrent streams each start at 1. Cross-stream comparison is permitted and meaningless. **Not** a process-wide counter — that is exactly defect **C3**, whose fix "is not a smaller counter; it is putting the counter where the stream is." | AI-14 | 0001 § 3.1 **C3** |
| `V-STR-14` | **block** | One contiguous unit of streamed content within a response — a run of text, a reasoning passage, a single tool call — delimited by a start event and an end event, with zero or more deltas between them. | AI-14 | 0001 § 4.3 |
| `V-STR-15` | **block index** | The value attributing an event to its own block, so that events for concurrently interleaved blocks are separable by index alone. | AI-14 | 0001 § 4.3 invariant 1 |
| `V-STR-16` | **delta** | An incremental fragment of a block's content, carrying **an index and only the new fragment** — never a snapshot of accumulated content. Consumers accumulate. A full copy per token is quadratic allocation if copied and a data race if shared. Deltas are optional in the general case: a block delivered whole with zero deltas is legal and, after reconstruction, indistinguishable from a fragmented one. | AI-14 | 0001 § 3.3 row 1, § 4.3 invariant 1 · **G12(a)** |
| `V-STR-17` | **ordering invariant** | A stated, checkable rule about legal event order on any stream: a block's start precedes its deltas which precede its end; exactly one terminal event per stream; nothing follows a terminal. Expressed as something a consumer can run against a recorded stream, not only as prose. | AI-14 | 0001 § 4.3 |
| `V-STR-18` | **terminal event** | The single event that ends a stream. Exactly one per stream, and nothing follows it. Its two instances are the completion event (`V-STR-20`) and the terminal error event (`V-FAIL-10`) — a stream ends with one **or** the other, never both and never neither, except on the sanctioned loss path (`V-STR-09`). | AI-14 | 0001 § 3.1 **C4**, § 4.3 |
| `V-STR-19` | **response-start event** | The event announcing that a provider has begun responding. The stream's lifecycle opening. | AI-15 | 0001 § 2.3 |
| `V-STR-20` | **completion event** | The terminal event of a stream that finished normally, carrying the response's finish reason and usage. | AI-15 | 0001 § 4.3, § 7 **G10** |
| `V-STR-21` | **call ordinal** | The observable position of a tool call among the calls of one response, preserved unchanged through normalization, streaming, and adapter translation. Exists because several providers reject tool results that do not correspond positionally to their calls, which makes a deterministic call-ordered re-join possible above Layer 1. **Grouped here per AI-01.1's checklist item 2; owned by AI-09**, where the concept originates in the tool-call content contract. Restated — never re-defined — at AI-18.3 (streaming) and AI-30.5 (adapter). | AI-09 | 0001 § 2.3 item 5, § 7 **G5** |
| `V-STR-24` | **provider response identity** | The provider's own opaque handle for the response a stream carries. Provider-supplied and byte-exact — never minted, parsed or canonicalized by Layer 1. Distinct from `V-STR-01` stream, which is the delivery in progress, and from any Layer-1-minted identity. *(Appended 2026-08-01 by AI-15.1.)* | AI-15 | 0001 § 2.3 |
| `V-STR-25` | **served model** | The model that actually produced the response, as reported by the provider. Distinct from `V-REQ-21` **model identity**, which is what the request asked for; the two may legitimately differ, and conflating them is how a routed or aliased model becomes invisible. *(Appended 2026-08-01 by AI-15.1.)* | AI-15 | 0001 § 2.3 |

---

## 5. Metadata terms — `V-MET`

What a response says about itself. Owner AI-13. **Closing-checklist item 3.**

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `V-MET-01` | **finish reason** | The closed vocabulary value stating why generation stopped. Closed and complete from birth, because collapsing distinct stop conditions into a fallback is a loop-termination defect above Layer 1, not a cosmetic gap. | AI-13 | 0001 § 3.2, § 3.3 row 3 · **G12(c)** |
| `V-MET-02` | **natural-stop finish reason** | Generation ended because the model finished. | AI-13 | 0001 § 3.2 |
| `V-MET-03` | **length finish reason** | Generation ended because an output limit was reached. | AI-13 | 0001 § 3.2 |
| `V-MET-04` | **tool-calls finish reason** | Generation ended because the model requested one or more tool calls. | AI-13 | 0001 § 3.2 |
| `V-MET-05` | **content-filter finish reason** | Generation ended because a provider-side content filter intervened. | AI-13 | 0001 § 3.2 |
| `V-MET-06` | **refusal finish reason** | Generation ended because the model **declined**. Distinct from unknown and from pause: a decline is a final answer. | AI-13 | 0001 § 3.2, § 3.3 row 3 · **G12(c)** |
| `V-MET-07` | **pause-turn finish reason** | Generation **paused** and can be resumed. Distinct from a decline and from unknown: the correct response is to resume, not to stop or to guess. | AI-13 | 0001 § 3.2, § 3.3 row 3 · **G12(c)** |
| `V-MET-08` | **unknown finish reason** | The recorded outcome for a provider stop condition the vocabulary does not recognise. A recorded state — "I do not recognise this string" — never a silent substitute for refusal or pause. Those three are three states with three different correct responses. | AI-13 | 0001 § 3.2 · **G12(c)** |
| `V-MET-09` | **usage** | The record of what a response consumed: input, output, cache-read, cache-write and reasoning token counts. Layer 1 reports usage; it never prices it — see `V-OUT-07`, `V-OUT-08`. | AI-13 | 0001 § 7 **G10** |
| `V-MET-10` | **token-count field** | One counted quantity within usage. Each is independently present or absent, because providers report different subsets. | AI-13 | 0001 § 7 **G10** |
| `V-MET-11` | **absence versus zero** | The definitional distinction between a count a provider did not report and a count a provider reported as nought. "Not reported" and "reported as none" are different facts, and a consumer that cannot tell them apart writes a wrong cost formula and a wrong compaction estimate. | AI-13 | 0001 § 7 **G10** |
| `V-MET-12` | **cost formula** | The unambiguous arithmetic a consumer can write over usage fields without guessing whether a count is inclusive of another. Layer 1 owns the field semantics that make such a formula writable; the money is not Layer 1's. One nuance the register records because it is easy to get wrong: on models with adaptive reasoning, the reasoning token count arrives only on the final usage update, so any mid-stream figure is structurally an estimate. | AI-13 | 0001 § 7 **G10** ("Two dispositions worth their reasoning") |

---

## 6. Failure terms — `V-FAIL`

**Closing-checklist item 4.** Two separations, and they are **orthogonal**. Reading them as one is the confusion this section exists to prevent.

```
                       who owns the failure
                 caller-contract  |  provider / transport
                    (AI-04)       |        (AI-19)
              +-------------------+-----------------------+
  pre-stream  |  invalid request  |  auth rejected,       |
  delivery    |  never leaves     |  connect failed,      |
              |  the process      |  request rejected     |
              +-------------------+-----------------------+
  mid-stream  |  (empty by        |  stream dies after    |
  delivery    |  construction —   |  emitting output;     |
              |  validation runs  |  the partial-output   |
              |  before any I/O)  |  case                 |
              +-------------------+-----------------------+
```

**The owner split** answers *whose failure it is*. **The delivery split** answers *how the caller observes it*. The lower-left cell being empty is not an accident: `V-REQ-22` requires validation to run once, before any I/O, which is precisely what keeps caller-contract failures out of the mid-stream path.

> **Amended 2026-07-31** — appended `V-FAIL-16` **validation rule** and `V-FAIL-17` **rule class** to § 6.1, both owned by **AI-04**, in the pull request for `cachicamas-ai-validation-errors` (AI-04.1) that needed them. Per § 9 rule 2, a milestone that needs a Layer 1 noun this register lacks appends it here rather than defining it locally. `validation rule` was needed because the word "rule" already appeared *inside* `V-FAIL-01` ("a violation of a construction or composition rule"), `V-FAIL-02` ("which rule a caller-contract failure violated") and `V-FAIL-04` ("the order in which rules are checked") without ever being defined — `V-STR-23` **backpressure**'s situation exactly, and an undefined "rule" is how a check that needs I/O comes to be called one. `rule class` was needed because doc 0002's AI-04.1 closing checklist states its recommended default as "one sentinel per rule **class**, reusable across types", and the register carried no term for the thing a sentinel is one of; without it AI-04.1 cannot state its own unit of granularity, and a decision that cannot state its unit has not made it. Both defer their substance to AI-04 the way `V-PRV-08` defers the discovery mechanism — the register says what a rule class *is*, never which classes exist — so § 9 rule 5 holds. No existing row was renumbered, reworded, reordered or removed; § 9 rule 3 holds. Term counts in § 10 and the failure-category identifier range in its checklist row 4 updated accordingly.

### 6.1 Caller-contract failures — AI-04

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `V-FAIL-01` | **caller-contract failure** | A violation of a construction or composition rule by the caller. **The caller's bug, knowable without any I/O.** Every rule violation in every Layer 1 contract reports through this one vocabulary. Deliberately defined before the first validating contract, because the retired plan defined it seventeenth and spent a milestone rationalizing ten milestones' worth of ad-hoc sentinels. | AI-04 | 0001 § 3.1 |
| `V-FAIL-02` | **validation sentinel** | The stable, matchable value identifying which rule a caller-contract failure violated. Matchable through at least one layer of wrapping, so a caller can classify a failure it received indirectly. | AI-04 | 0001 § 9 ("Errors are typed and inspectable") |
| `V-FAIL-03` | **positional context** | The attached information naming *where* a violation occurred — which message, which content index, which tool — without becoming a second, parallel failure vocabulary. | AI-04 | 0001 § 9 |
| `V-FAIL-04` | **first-failure ordering** | The documented, deterministic order in which rules are checked, so that a value violating several rules reports the same one on every run. Determinism is the property; whether to report the first violation or all of them is AI-04.1's decision. | AI-04 | 0001 § 9 |
| `V-FAIL-16` | **validation rule** | One checkable statement about a request value whose violation is a `V-FAIL-01`: a condition the value must satisfy, decidable from the request alone. A rule belongs to the contract that states it, and it occupies a stated position in that contract's `V-FAIL-04` order. **A check that needs anything the request value does not carry is not a rule** — that is the boundary of § 6.3 expressed as a property of the noun. Which rules each contract states is that contract's milestone's. *(Appended 2026-07-31 by AI-04.1.)* | AI-04 | 0001 § 3.1, § 9 |
| `V-FAIL-17` | **rule class** | The kind of thing a `V-FAIL-16` **validation rule** checks, independent of what it checks it on: "a required value is empty" is one class whether the value is a model identity, a message's content, or a tool's name. The unit a `V-FAIL-02` **validation sentinel** identifies, which is what allows one sentinel to be reusable across every Layer 1 contract instead of one per rule per type. Which classes exist, and how the set grows, are AI-04's. *(Appended 2026-07-31 by AI-04.1.)* | AI-04 | 0001 § 9 ("Errors are typed and inspectable") |

### 6.2 Provider and transport failures — AI-19

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `V-FAIL-05` | **provider/transport failure** | Any failure occurring **after a valid request leaves the process**: transport, protocol, authentication, rate limiting, provider-side error, deadline, disconnect. Not the caller's bug and not knowable without I/O. Normalized into one inspectable vocabulary so that no consumer parses a message string. | AI-19 | 0001 § 4.3 invariant 4, § 7 **G8** |
| `V-FAIL-06` | **failure category** | The closed classification of a provider/transport failure. The thing a consumer switches on. Closed, because an open one pushes classification back into message parsing. | AI-19 | 0001 § 7 **G8** |
| `V-FAIL-07` | **retryability** | The property of a provider/transport failure stating whether re-issuing the request could plausibly succeed. **Knowable only where the wire failure is** — hence Layer 1 — while whether to *act* on it is a policy decision one layer up. Collapsing the two loses both. | AI-19 | 0001 § 6 seam 7, § 7 **G8** |
| `V-FAIL-08` | **partial output** | The content a stream already delivered before it failed. A stream that dies after emitting output is the most common real-world failure and the one a naive retry predicate excludes. Partial output is preserved, never discarded. | AI-19 | 0001 § 2.3, § 4.3 invariant 4, § 7 **G8** |
| `V-FAIL-09` | **partial-output discriminator** | The inspectable property of a failure stating whether any semantic content preceded it. The single fact on which a correct retry decision turns: "retry if nothing completed" is precisely the predicate that gets this wrong. | AI-19 | 0001 § 7 **G8** |
| `V-FAIL-10` | **terminal error event** | The terminal event of a stream that failed mid-flight, carrying the failure's category, its retryability, its partial-output discriminator, and its wrapped cause — **constructible by an adapter in another package**. That last clause is the whole of defect **C4**: the retired plan declared this event mandatory in two files while no adapter could build it, because the interface shipped before the taxonomy it depended on. | AI-19 | 0001 § 3.1 **C4** |
| `V-FAIL-11` | **pre-stream failure delivery** | What a caller observes when a request **never becomes a stream**: the failure is returned directly, and no stream and no producer ever exist. | AI-19 | 0001 § 4.3 invariant 4 ("Setup failures and stream failures are distinguishable") |
| `V-FAIL-12` | **mid-stream failure delivery** | What a caller observes when a stream **dies after starting**: the failure arrives as the terminal error event, carrying whatever partial output preceded it. One vocabulary, two delivery paths: a caller that only ever inspects the returned failure and a caller that only ever inspects the terminal event can each classify every failure the vocabulary defines. AI-02 decides the observable shape of the split; AI-19 implements it. | AI-19 | 0001 § 4.3 invariant 4, § 7 **G8** |
| `V-FAIL-13` | **safe metadata** | The diagnostic information a failure may carry: status classes, categories, retry hints, bounded and sanitized excerpts. Never a credential, never a header, never a raw provider body, never model content. The posture starts at the first thing in the package that formats caller data — a validation failure — not at the hardening milestone. | AI-19 | 0001 § 9 (observability checklist) |
| `V-FAIL-14` | **redaction** | The discipline that keeps credentials, authorization headers, sensitive bodies and unbounded provider text out of failures, logs, spans, fixtures and test output, proven adversarially by planting a sentinel and asserting its absence. | AI-36 | 0001 § 9 · ADR 0005 § D3 denylist |
| `V-FAIL-15` | **retry policy (Layer 1 half)** | The stated rule governing which failures Layer 1 may re-issue automatically. One clause is promoted to structure: **the partial-output case is never retried at Layer 1**, because re-issuing after any semantic event has been emitted can duplicate observable output. Agent-turn retries and model failover are not Layer 1's — see `V-OUT-11`. | AI-35 | 0001 § 6 seams 7, 8, § 7 **G8** |

### 6.3 The boundary, as a rule and a worked case

**The rule.** A failure is a caller-contract failure if and only if it is decidable from the request alone, without contacting a provider. Everything else is a provider/transport failure. The test is *decidability without I/O*, not *severity*, not *who is to blame*, and not *where in the code it was noticed*.

**Worked borderline case — a model identity the provider does not recognise.** The request is well-formed: a model identity is present, non-empty, and structurally valid. Layer 1 holds no catalog (`V-OUT-14`), so it cannot know whether the provider supports this model. The unrecognised identity is therefore **not** decidable from the request alone, and it is a **provider/transport failure** (`V-FAIL-05`), delivered pre-stream (`V-FAIL-11`) with the appropriate category. Had the model identity been *empty*, that is decidable without I/O and is a **caller-contract failure** (`V-FAIL-01`).

The same test resolves the other recurring borderline cases: a tool choice naming a tool absent from the declared set is decidable from the request alone → caller-contract. Argument bytes that are not well-formed for the documented encoding are decidable from the request alone → caller-contract. Argument bytes that are well-formed but do not satisfy the tool's schema are **neither**: Layer 1 does not validate arguments against schemas (trap 1), so it carries them and the failure, if any, belongs above.

---

## 7. Provider-surface and proving terms — `V-PRV`

The interface Layer 2 calls, and the apparatus that judges an adapter. Owners AI-03, AI-20 … AI-24.

> **Why this category exists although AI-01.1's checklist does not name it.** AI-01's *acceptance* criterion is stronger than its checklist: "Every subsequent milestone's charter can be written using only these terms." AI-03's charter is not writable without `capability`, `required capability`, `optional capability`, `capability discovery` and `capability record`; AI-21's, AI-22's and AI-23's are not writable without `fake provider`, `stream test kit` and `conformance suite`. The charter is the normative scope, so the acceptance criterion governs.

> **Amended 2026-07-31** — appended `V-PRV-16` **capability**, `V-PRV-17` **token counting** and `V-PRV-18` **capability outcome**, all owned by **AI-03**, in the pull request for `cachicamas-ai-minimum-capabilities` (AI-03.1) that needed them. Per § 9 rule 2, a milestone that needs a Layer 1 noun this register lacks appends it here rather than defining it locally. `capability` closes a gap this section's own preamble identifies: it names five terms AI-03's charter is not writable without, and the table below delivered four — without the bare noun there is no way to say that a contract obligation, an adapter-local mapping obligation and a contract property optional for every adapter are *not* capabilities. `token counting` was needed because the phrase already appeared inside `V-OUT-06`'s definition without being defined, where it silently collapses into `V-MET-09` **usage** — a report about an output standing in for a question about an input. `capability outcome` was needed because the word "outcome" already appeared inside `V-PRV-09`'s definition undefined, and the distinction that row exists to protect — a recorded absence is not an unrun case — is a distinction *between outcome values*; this is `V-STR-23` **backpressure**'s situation exactly. Each definition defers its substance to AI-03 the way `V-PRV-08` defers the discovery mechanism, so § 9 rule 5 holds. No existing row was renumbered, reworded or removed; § 9 rule 3 holds. Term counts in § 10 updated accordingly.

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `V-PRV-01` | **provider** | Anything that accepts a normalized request and returns a normalized event stream. Layer 1's outward face and the only thing Layer 2 calls. | AI-20 | 0001 § 2.2, § 3.1 |
| `V-PRV-02` | **adapter** | A concrete provider implementation that translates between the neutral contracts and one vendor's wire representation. Everything vendor-specific lives inside one. Trap 2 binds here: a vendor without an adapter is not usable by configuration. | AI-20 | 0001 § 2.1, § 3.3 |
| `V-PRV-03` | **provider interface** | The declared surface every provider satisfies: a request in, a stream out, with cancellation carried by the caller. **No vendor type and no wire type appears on it.** Implementable from another package. | AI-20 | 0001 § 2.2, § 9 ("No vendor wire type crosses the Layer 1 method boundary") |
| `V-PRV-04` | **pre-stream contract** | The stated obligations that hold before any stream exists: an invalid request fails before any stream and before any producer is created. | AI-20 | 0001 § 4.3 invariant 4 |
| `V-PRV-05` | **mid-stream contract** | The stated obligations that hold once a stream exists: ownership, sequencing, ordering, terminal discipline, cancellation, and the sanctioned loss path. | AI-20 | 0001 § 9 |
| `V-PRV-06` | **required capability** | Behavior every conformant adapter must exhibit. Absence is non-conformance. | AI-03 | 0001 § 6 seam 6 |
| `V-PRV-07` | **optional capability** | Behavior a conformant adapter **may** lack. An adapter implementing only the required surface is fully conformant. Optionality is a deliberate design position, not laxity: making a capability required forces every future adapter either to implement it or to lie about it. | AI-03 | 0001 § 6 seam 6, § 7 **G3** |
| `V-PRV-08` | **capability discovery** | The mechanism by which a consumer learns whether a provider offers an optional capability — an additional, separately-asserted contract on the provider value, **so the core provider interface never widens**. Which mechanism, precisely, is AI-03's decision. | AI-03 | 0001 § 6 seam 6, § 7 **G3** |
| `V-PRV-09` | **capability record** | The emitted, recorded outcome stating which capabilities a provider offers and which it does not. **"Absent" is a recorded outcome, not an unrun test** — a silently skipped case and a recorded absence are different facts. | AI-03 | 0001 § 6 seam 6 |
| `V-PRV-10` | **fake provider** | A scriptable, network-free provider used to exercise Layer 1 and Layer 2 deterministically. **Contract-faithful, not convenient**: a fake that closes cleanly where the real contract drops events teaches consumers the wrong physics, and they build on what the fake does. | AI-21 | 0001 § 4.1 ("callable directly from a test with a scripted provider") |
| `V-PRV-11` | **stream test kit** | The importable helpers that drain, record, diff and assert over streams, including ordering and gap assertions and a leak-detection mechanism. Every helper exists to assert something a Layer 1 contract states. | AI-22 | 0001 § 9 (streams checklist) |
| `V-PRV-12` | **conformance suite** | The reusable body of contract tests any provider can be plugged into, defining the behavior every adapter must exhibit without copying assertions. It is what makes a second adapter cheap, and it judges the fake provider first. Deterministic by construction: no credential, no socket. | AI-23 | 0001 § 8 (v1 non-goals) |
| `V-PRV-13` | **transport** | The mechanism by which an adapter reaches a vendor over the network. A different vendor that is genuinely compatible with an existing transport may not need a new adapter — the only qualification trap 2 admits. | AI-24 | 0001 § 2.1 |
| `V-PRV-14` | **frame** | One unit of a vendor's streaming protocol as received on the wire, before translation into events. An adapter-side term: a frame is never delivered to a consumer. | AI-27 | 0001 § 3.3 |
| `V-PRV-15` | **wire representation** | A vendor's own shape for a request or a response. Confined entirely inside an adapter; it never crosses the provider interface in either direction. | AI-26 | 0001 § 9 |
| `V-PRV-16` | **capability** | A behavior a consumer can ask a provider for, and whose presence or absence the consumer can observe. Both clauses bind: askable, **and** observably absent. Distinct from a contract obligation that binds every adapter identically — there the answer is always yes and nothing is discovered — and from an adapter-local mapping obligation, which is invisible above Layer 1 by construction, and from a contract property that is optional for *every* adapter, where the optionality belongs to the contract rather than to a provider. The generic term of which `V-PRV-06` and `V-PRV-07` are the two standings. Which behaviors are capabilities, and which standing each takes, is AI-03's. *(Appended 2026-07-31 by AI-03.1.)* | AI-03 | 0001 § 6 seam 6, § 3.3 |
| `V-PRV-17` | **token counting** | Answering how many tokens a given request would consume, asked of a provider **before** the request is sent. Distinct from `V-MET-09` **usage**, which records what a response consumed after the fact: one is a question about an input, the other a report about an output, and a consumer that substitutes the second for the first has no figure at all at the moment it needs one. Defined because the phrase already appeared inside `V-OUT-06`'s definition undefined. Its standing, and what a consumer observes when it is absent, are AI-03's. *(Appended 2026-07-31 by AI-03.1.)* | AI-03 | 0001 § 6 seam 6, § 7 **G3** · ADR 0005 § D4 row G3 |
| `V-PRV-18` | **capability outcome** | The value one entry of a `V-PRV-09` capability record carries: what one run observed about one capability for one provider. Defined because the word "outcome" already appears inside `V-PRV-09` undefined, and the distinction that row exists to protect — a recorded absence is not an unrun case — is a distinction *between outcome values*, so it cannot be stated without the noun. Which values the set contains, and which of them are conformant, are AI-03's. *(Appended 2026-07-31 by AI-03.1.)* | AI-03 | 0001 § 6 seam 6 |

---

## 8. Excluded terms — `V-OUT`

**Closing-checklist item 5.** These are named and attributed, and **deliberately not defined here** — defining a Layer 2 concept inside a Layer 1 artifact is the first step toward Layer 1 implementing it. Where an exclusion is easy to misread, the row names the Layer 1 neighbour it is confused with.

### 8.1 The nine the checklist requires

| Id | Excluded term | Owner | Layer 1 neighbour it is confused with | Provenance |
| --- | --- | --- | --- | --- |
| `V-OUT-01` | **agent turn** | Layer 2 — the loop runs one turn and emits agent-level events | `V-STR-01` **stream**. One turn may involve several streams (a compaction call, a retry, the response itself). A stream is a provider response; a turn is a unit of agent work. | 0001 § 4.1, § 4.3 |
| `V-OUT-02` | **transcript** | Layer 2 — the harness owns history, including the invariant that every tool call has a matching result, and the repair of orphaned calls after interruption or compaction | `V-REQ-02` **message**. AI-05's charter defines a message as "the smallest addressable unit of a transcript" — Layer 1 owns the unit, never the collection, its ordering across turns, or its repair. | 0001 § 4.2 |
| `V-OUT-03` | **session** | Layer 3 — append-only records with parent chains, under the user's home directory | `V-REQ-20` **normalized request**. A request is one call's data; a session is the durable record across calls. Note the coupling Layer 3 must honour: adapter-minted tool-call identifiers must survive serialisation and reload. | 0001 § 5.2 |
| `V-OUT-04` | **tool execution** | Layer 2 schedules it (concurrency policy, ordered re-join); Layer 3 confines it (the sandbox port) | `V-REQ-16` **tool call** and `V-REQ-18` **tool result**. Layer 1 carries the intent and the answer as transport representations. It never runs anything — trap 1. | 0001 § 4.1, § 5.1 · seams 2, 3 · **G2**, **G5** |
| `V-OUT-05` | **permission** | Layer 2 owns the protocol (ask, suspend, resume, on the event stream); Layer 3 owns the policy | `V-REQ-15` **tool choice**. Tool choice constrains what the model may request; permission decides whether a resulting call may run. Different questions, different layers. | 0001 § 4.1, § 5.1 · seam 2 · **G1** |
| `V-OUT-06` | **compaction** | Layer 2 — a model call with its own provider, cost and cancellation, which must protect recent turns and never orphan a call/result pair | `V-PRV-07` **optional capability**. Layer 1's only obligation is that token counting is discoverable and optional; compaction that estimates by character count is wrong by enough to matter. | 0001 § 4.2, § 6 seams 5, 6 · **G3** |
| `V-OUT-07` | **cost** | Layer 2 emits it as first-class per-turn and cumulative events | `V-MET-09` **usage**. Layer 1 already carries the cache and reasoning token counts an adapter would report; doc 0001 § 7 records that the Layer 1 obligation for **G10** is **already met**. What remains is entirely above. | 0001 § 4.3, § 7 **G10** |
| `V-OUT-08` | **price** | Layer 3 — the price-table port, catalog-driven per model | `V-MET-12` **cost formula**. Layer 1 owns the field semantics that make a formula unambiguous; it never holds money. | 0001 § 5.1, § 7 **G10** |
| `V-OUT-09` | **frontend** | Above Layer 3 — consumes events only; print mode is the minimum viable one | `V-STR-04` **consumer**. Layer 1's consumer is whatever drains a stream; a frontend is a renderer of Layer 2's enriched stream and is invisible to Layer 1 entirely. | 0001 § 2.2, § 8 |

### 8.2 Appended for completeness

Not required by the checklist. Included because a charter from AI-02 through AI-40 may reach for any of them, and each is a plausible mis-attribution to Layer 1.

| Id | Excluded term | Owner | Note | Provenance |
| --- | --- | --- | --- | --- |
| `V-OUT-10` | **loop termination** | Layer 2 — the loop decides whether a turn is complete | `V-MET-01` **finish reason** is the input to that decision, not the decision. A different termination rule is a different harness, not a different Layer 1. | 0001 § 4.1, § 4.2 |
| `V-OUT-11` | **retry and failover policy** | Layer 2 — the harness decides whether to retry, back off, or switch model | Layer 1 supplies `V-FAIL-07` **retryability** and, at `V-FAIL-15`, only the retries that cannot duplicate observed output. Failover re-opens the token budget, the price table and the cache prefix; only the harness holds all three. | 0001 § 4.2, § 6 seams 7, 8 · **G8** |
| `V-OUT-12` | **delegation / subagent** | Layer 2 — a harness invoked from inside a tool, with nested cancellation, cost and permission scope | Requires re-entrancy, a structural property of the harness. No Layer 1 obligation. | 0001 § 4.2, § 6 seam 12 · **G7** |
| `V-OUT-13` | **hook** | Layer 2 and Layer 3 — pre-request, pre-compact, post-turn, session-start; observers never synchronous on the streaming path | Layer 1's obligation is `V-REQ-29` **request rebuild**, which is the mechanism the pre-request hook stands on. The hook itself is not Layer 1's. | 0001 § 4.1, § 6 seam 1 · **G11** |
| `V-OUT-14` | **provider catalog / credential resolution** | The composition root — resolves the catalog and credentials, then injects them | Layer 1 receives what it needs. Nothing below the composition root reads the environment, opens a configuration file, or decides a policy. | 0001 § 5.2 · ADR 0005 § D1 row 4 |
| `V-OUT-15` | **skill / prompt / project instruction** | Layer 3 — the skill and prompt source ports, over an ordered chain | `V-REQ-19` **system-instruction segment** is the transport shape; what fills a segment, and which source wins, is policy and is settled by ADR 0006. | 0001 § 5.1 · ADR 0006 |
| `V-OUT-16` | **sandbox** | Layer 3 — a parameter of tool *execution*, not of a tool | Confinement is a property of the call site. Entirely invisible to Layer 1 — trap 1. | 0001 § 5.1, § 6 seam 3 · **G2** |
| `V-OUT-17` | **tool source** | Layer 3 — built-ins, servers, per-session filters; dynamic between turns | `V-REQ-14` **tool set** is what one request carries. Where the declarations came from, and whether the set can change, is Layer 3's — with a consequence Layer 3 must price: a set that changes between turns invalidates the cache prefix. | 0001 § 5.1, § 6 seam 4 · **G6** |

---

## 9. Standing rules this vocabulary establishes

1. **One definition, one owner.** Every term above has exactly one definition and exactly one owning milestone. A downstream milestone that needs a different meaning is proposing an amendment, not exercising a judgement call.
2. **A missing term is appended, never invented.** When a milestone needs a Layer 1 noun this register lacks, the term is appended here — next free ordinal in its category, dated amendment blockquote under the category heading — **in the same pull request that needs it**. It is not defined locally in that milestone's SDD. This is doc 0002's living-graph clause applied to nouns instead of nodes.
3. **Identifiers are append-only.** `V-*` identifiers are never renumbered, reused, or reordered. A superseded definition keeps its identifier with its old text struck through and visible, so citations from merged charters keep resolving.
4. **Citations use current milestone identifiers.** doc 0001 and ADR 0005 still carry retired numbers. Translate through doc 0002's identifier map. The trap is live: doc 0001 attributes **C4** to "AI-18", which today is a different milestone; C4 is **AI-19**'s.
5. **This artifact settles words, not behavior.** It does not choose the stream carrier (AI-02), the capability matrix or the discovery mechanism (AI-03), validation granularity or aggregation (AI-04), or the content-part strategy (AI-06). The test for over-reach: if a sentence were deleted, would a later milestone have more options? If yes, and that milestone is not AI-01, the sentence does not belong here.
6. **The two traps bind every later reading.** Layer 1 owns the transport representation of tools and nothing else about them. A provider without an adapter is not reachable by configuration.

---

## 10. Closing-checklist verification and term count

AI-01.1's six items, each against this register, as verified at merge. The term count below is **live**: an amendment updates it.

| # | Closing-checklist item | Where answered | Status |
| --- | --- | --- | --- |
| 1 | Request-side terms: role, message, content part and its kinds, tool declaration, tool choice, tool call, tool result, system-instruction segment, cache-boundary marker, generation option, provider escape hatch | § 3 — `V-REQ-01` … `V-REQ-29`; all eleven named terms present | **answered** |
| 2 | Stream-side terms: event, event kind, payload, sequence, stream, terminal event, delta, block index, call ordinal | § 4 — `V-STR-01` … `V-STR-25`; all nine named terms present, with `call ordinal` grouped here and owned by AI-09 | **answered** |
| 3 | Metadata terms: finish reason, usage, token-count field, absence versus zero | § 5 — `V-MET-01` … `V-MET-12`; all four named terms present, plus the complete closed finish-reason vocabulary | **answered** |
| 4 | Failure terms defined and separated: caller-contract (AI-04) versus provider/transport (AI-19), and the pre-stream versus mid-stream delivery split | § 6 — the two-axis diagram, `V-FAIL-01` … `V-FAIL-17`, the decidability-without-I/O rule, and four worked borderline cases | **answered** |
| 5 | Terms explicitly excluded, with their owner named: agent turn, transcript, session, tool execution, permission, compaction, cost, price, frontend | § 8.1 — `V-OUT-01` … `V-OUT-09`, all nine with a non-Layer-1 owner and the Layer 1 neighbour each is confused with; § 8.2 appends eight more | **answered** |
| 6 | The two wording traps from the layer boundary restated, because both have already caused one wrong decision each | § 2 — both quoted verbatim, each with the record of its wrong decision and its operational consequence for this register | **answered** |

**Term count:** 29 request-side · 25 stream-side · 12 metadata · 17 failure · 18 provider-surface · 17 excluded = **118 terms**. *(Amended 2026-07-31: `V-STR-22`, `V-STR-23` appended by AI-02.1 — see the amendment blockquote in § 4. Amended 2026-07-31: `V-PRV-16`, `V-PRV-17`, `V-PRV-18` appended by AI-03.1 — see the amendment blockquote in § 7. Amended 2026-07-31: `V-FAIL-16`, `V-FAIL-17` appended by AI-04.1 — see the amendment blockquote in § 6. Amended 2026-08-01: `V-STR-24`, `V-STR-25` appended by AI-15.1 — see the amendment blockquote in § 4.)*

**Consumers of this register.** AI-02 (`ai-stream-lifecycle`) and AI-03 (`ai-minimum-capabilities`) were written from it in wave 0 and cite 35 and 60 distinct `V-*` identifiers respectively; every one resolves. AI-04 (`ai-validation-errors`) opens wave 1 as the first milestone to both consume the register and ship Go. Every contract milestone AI-04 … AI-40 consumes it in turn.

---

## Requirements

The requirements below constrain **this register**. Every scenario is a property a reviewer can check by inspection, deterministically, without running anything: a scenario reads *"given the register, when …, then …"* — the register is the system under test. That distinction is stated once, here, so no later reader mistakes a documentation contract for a runtime one.

### R-AIV-001 — The register is singular

Exactly one Layer 1 vocabulary register exists, and it is this file. No other file in the repository MAY carry a competing definition of a Layer 1 term. A downstream artifact MUST cite a row by its identifier, or quote the owning row verbatim and say that it is quoting; a restatement in other words is a second definition and is a defect. The register is the single source for Layer 1 term ownership until it is superseded by an appended decision node under AI-01.

#### Scenarios

- **S-AIV-001** — Given the repository, when a reviewer looks for the definition of a Layer 1 term, then exactly one file — this one — defines it AND no other artifact restates a term definition as normative.

### R-AIV-002 — One definition per term

Every term in the register MUST carry exactly one definition. A term MUST NOT be defined twice, in two categories, or with two wordings. Where a concept recurs in a later milestone, the register MUST express the recurrence as a cross-reference from the single owning row, never as a second definition.

#### Scenarios

- **S-AIV-002** — Given the register, when a reviewer collects every term name across all six categories, then no term name appears in more than one row AND every row's definition text is unique.
- **S-AIV-003** — Given the term `call ordinal`, which the AI-01.1 closing checklist groups stream-side while the concept originates in the tool-call content contract, when a reviewer inspects the register, then exactly one row defines it AND that row names AI-09 as owner AND the row cross-references AI-18.3 and AI-30.5 as restatements rather than defining the term again there.

### R-AIV-003 — One owning milestone per term

Every non-excluded term in the register MUST name exactly one owning milestone, expressed as a current `AI-NN` identifier from doc 0002. Two rows MUST NOT claim the same term. A row MUST NOT name two owners, name a range, or leave the owner unstated.

#### Scenarios

- **S-AIV-004** — Given the register, when a reviewer reads the owner column of every non-excluded row, then each cell holds exactly one `AI-NN` identifier AND that identifier exists as a milestone heading in doc 0002.
- **S-AIV-005** — Given any milestone `AI-NN` cited as an owner, when a reviewer opens that milestone's charter in doc 0002, then the charter's goal or deliverable covers the owned term — an owner whose charter does not cover the term is a defect in this register, not in doc 0002.

### R-AIV-004 — Stable, append-only term identifiers

Every term MUST carry a stable identifier of the form `V-<CAT>-nn`, where `<CAT>` is the category code and `nn` is an ordinal within that category. Identifiers MUST be append-only: a term added later takes the next free ordinal in its category, and existing identifiers MUST NOT be renumbered, reused, or reordered. A superseded term MUST retain its identifier with its definition struck through.

#### Scenarios

- **S-AIV-006** — Given the register, when a reviewer reads any category's identifiers in document order, then ordinals are unique within the category AND no ordinal is reused across a superseded and a live row.
- **S-AIV-007** — Given a term discovered missing, when it is added, then it receives the next free ordinal in its category AND is introduced by a dated amendment blockquote under its category heading AND no existing identifier changes.

### R-AIV-005 — Category completeness against the closing checklist

The register MUST define, at minimum, every term named by AI-01.1's closing checklist items 1 through 4:

- **Request-side** (item 1): role, message, content part and its kinds, tool declaration, tool choice, tool call, tool result, system-instruction segment, cache-boundary marker, generation option, provider escape hatch.
- **Stream-side** (item 2): event, event kind, payload, sequence, stream, terminal event, delta, block index, call ordinal.
- **Metadata** (item 3): finish reason, usage, token-count field, absence versus zero.
- **Failure** (item 4): caller-contract failure, provider/transport failure, pre-stream failure delivery, mid-stream failure delivery.

The register MAY define additional terms beyond this minimum where AI-01's acceptance criterion requires them.

#### Scenarios

- **S-AIV-008** — Given AI-01.1's closing-checklist items 1 through 4, when a reviewer checks each named term against the register, then every named term resolves to exactly one row.
- **S-AIV-009** — Given the register contains terms beyond the checklist minimum, when a reviewer asks why each exists, then the register names the downstream milestone charter that could not be written without it.

### R-AIV-006 — Failure terms are separated by owner and by delivery path

The register MUST separate **caller-contract failure** (AI-04's territory) from **provider/transport failure** (AI-19's territory), stating the line between them, and MUST separately state the **pre-stream versus mid-stream** delivery split. The two separations are orthogonal and MUST be presented as such: the owner split says *whose failure it is*, the delivery split says *how the caller observes it*.

#### Scenarios

- **S-AIV-010** — Given the failure category of the register, when a reviewer reads it, then caller-contract failure names AI-04 as owner AND provider/transport failure names AI-19 as owner AND the boundary between them is stated as a rule a reader can apply to a new case.
- **S-AIV-011** — Given the failure category, when a reviewer looks for the delivery split, then pre-stream failure delivery and mid-stream failure delivery are both defined AND both are attributed to AI-19 as the milestone that implements them as one vocabulary over two paths AND the register records that AI-02.1 decides the observable shape.
- **S-AIV-012** — Given a borderline failure case — a request the caller could not have known was invalid without contacting the provider — when a reader applies the stated boundary, then the register resolves it to exactly one side, and § 6.3 shows that resolution as a worked example.

### R-AIV-007 — Excluded terms carry a named owner

The register MUST list, at minimum, these nine terms as excluded from Layer 1, each with the layer or component that owns it: agent turn, transcript, session, tool execution, permission, compaction, cost, price, frontend. An excluded term MUST NOT be defined by this register; it MUST be named, attributed, and — where the exclusion is easy to misread — accompanied by the Layer 1 concept it is commonly confused with.

#### Scenarios

- **S-AIV-013** — Given the exclusion register, when a reviewer checks the nine terms of AI-01.1 checklist item 5, then all nine are present AND each names an owner that is a layer, a port, or the composition root — never a Layer 1 milestone.
- **S-AIV-014** — Given the excluded term `transcript`, when a reviewer reads its row, then the row names Layer 2's harness as owner AND distinguishes it from the Layer 1 term `message`, which AI-05's charter calls "the smallest addressable unit of a transcript".
- **S-AIV-015** — Given any excluded term, when a reviewer looks for a definition of it in the register, then none is present — the row states ownership and the confusable Layer 1 neighbour only.

### R-AIV-008 — Provenance and identifier translation

Where a term exists to close a specific recorded defect or gap, its row MUST cite the identifier (`C1`–`C4`, `G1`–`G13`) and the doc 0001 section it derives from. Every milestone identifier cited in the register MUST be a **current** doc 0002 identifier. Where a source document uses a retired identifier, the register MUST translate it through doc 0002's identifier map and MUST NOT reproduce the retired number.

#### Scenarios

- **S-AIV-016** — Given the term `round-trip token`, when a reviewer reads its row, then it cites doc 0001 § 3.2 and § 3.3 row 2 and gap `G12(b)` AND names AI-07 as owner — not the retired identifier AI-45 that doc 0001 § 3.2 carried.
- **S-AIV-017** — Given the term `terminal error event`, when a reviewer reads its row, then it cites defect `C4` AND names AI-19 as owner — not the retired identifier AI-18 that doc 0001 § 3.1 carried, which in current numbering is a different milestone.
- **S-AIV-018** — Given every `AI-NN` identifier in this register that **attributes ownership or provenance**, when a reviewer resolves each against doc 0002's milestone headings, then every one resolves AND none is a retired identifier. One exception, and only one: the register MAY name retired identifiers inside its explicit retired-identifier warning in § 1, where naming them is the point; such a mention MUST be adjacent to the current identifier that replaces it.

### R-AIV-009 — No Go identifiers

This register MUST NOT contain a Go type name, field name, method name, interface name, or package identifier belonging to the Layer 1 surface. Terms MUST be expressed as conceptual noun phrases. Naming the *spelling* of a term is each owning milestone's SDD decision.

#### Scenarios

- **S-AIV-019** — Given this register, when a reviewer scans for camel-case or Pascal-case single-token names, struct or interface declarations, or field lists, then none is present.
- **S-AIV-020** — Given the term `provider escape hatch`, when a reviewer reads its row, then the definition describes what the concept must be able to carry and what nothing in Layer 1 may do with it, and states no name, shape, or signature.

### R-AIV-010 — The register is a documentation contract, never code

The register MUST NOT be implemented as, mirrored into, or generated from code. AI-01.1 created no file under `backend/`, no `go.mod`, no `go.sum`, no `Makefile`, and no build or container configuration; an amendment to the register MUST likewise change nothing but markdown.

#### Scenarios

- **S-AIV-021** — Given the diff of any change that touches this register, when a reviewer lists its changed paths, then every path that belongs to the register is markdown under `openspec/specs/ai-contract-vocabulary/`.
- **S-AIV-022** — Given the repository before and after any amendment to this register, when `make build` and `make test` are run in every module, then their results are identical in outcome — an amendment is provably inert with respect to the build.

### R-AIV-011 — Growth is by amendment, never by invention

A Layer 1 term that turns out to be missing, wrong, or double-owned MUST be corrected by an amendment to this register, landed in the same pull request that needs the correction. A downstream milestone MUST NOT introduce a new Layer 1 term in its own SDD without appending it here. An amendment MUST follow doc 0002's convention: a dated blockquote under the touched heading, struck-through text for superseded claims, and no silent edit.

#### Scenarios

- **S-AIV-023** — Given a downstream milestone whose SDD needs a term the register does not carry, when that SDD is written, then the same pull request appends the term to the register with the next free ordinal in its category AND a dated amendment blockquote records the addition.
- **S-AIV-024** — Given a superseded definition, when a reviewer reads its row, then the old text is struck through and remains visible AND the replacing text is present AND the row's identifier is unchanged.
- **S-AIV-025** — Given any pull request for milestones AI-02 through AI-40, when a reviewer applies the review checklist, then a Layer 1 noun used in that PR's artifacts either resolves to a register row or is accompanied by the amendment that adds it.

### R-AIV-012 — Downstream charters are expressible in these terms

The register MUST be complete enough that every milestone charter from AI-02 through AI-40 can be written using only its terms plus ordinary English. This is AI-01's acceptance criterion, stated normatively so it is checkable rather than aspirational.

#### Scenarios

- **S-AIV-026** — Given AI-02's charter as written in doc 0002, when a reviewer maps each domain noun in it to the register, then every one resolves — specifically carrier, ownership, cancellation, buffering, and the failure-delivery split.
- **S-AIV-027** — Given AI-03's charter as written in doc 0002, when a reviewer maps each domain noun in it to the register, then every one resolves — specifically capability, required capability, optional capability, capability discovery, and capability record.
- **S-AIV-028** — Given any charter from AI-04 through AI-40, when a reviewer finds a domain noun that does not resolve, then that finding is recorded as a defect in this register and closed by amendment under `R-AIV-011` — not by inventing the term in that milestone's SDD.

### R-AIV-013 — The two wording traps are restated verbatim

The register MUST restate both wording traps from doc 0002's *Layer boundary* section, quoted verbatim rather than paraphrased, each accompanied by the record that it has already caused one wrong decision.

#### Scenarios

- **S-AIV-029** — Given § 2, when a reviewer compares its trap quotations against doc 0002's *Layer boundary* paragraph, then both sentences match character for character, including their qualifying clauses.
- **S-AIV-030** — Given the first trap, when a reviewer reads the accompanying text, then it states that Layer 1 owns the provider-neutral transport representation of tool declarations and tool calls AND that Layer 1 must not execute tools, resolve tool names, or own application behavior AND the register's tool-related rows are consistent with both halves.
- **S-AIV-031** — Given the second trap, when a reviewer reads the accompanying text, then it states that the claim applies only after adapters exist AND that adding a new vendor still requires a new adapter unless it is genuinely compatible with an existing transport.

---

## Non-functional requirements

### NFR-AIV-A — Reviewability

- The register MUST be readable in one sitting by someone who has read doc 0002's *Layer boundary* section and nothing else.
- Every row MUST fit the pattern `identifier · term · definition · owner · provenance` so a reviewer can scan one column at a time.
- The register MUST NOT require the reader to open doc 0001 to understand a definition; citations are for provenance, not for comprehension.

### NFR-AIV-B — Traceability

- Every `C1`–`C4` and every `G1`–`G13` identifier with a Layer 1 obligation MUST appear at least once in the register's provenance column, so the vocabulary is auditable against doc 0002's traceability spine. All 17 appear.
- Every citation MUST name a document and a section, not a line number, because line numbers drift.

### NFR-AIV-C — Durability

- The register MUST NOT cite shipped code as evidence. doc 0002's authoring constraint applies: every citation points at a contract document, an ADR, or the architecture reference.

---

## Acceptance criteria

The contract holds when:

1. `R-AIV-001` through `R-AIV-013` hold, each verified by its scenarios.
2. All six items of AI-01.1's closing checklist are answered in § 1 … § 10.
3. No Go identifier appears anywhere in this file.
4. Every amendment leaves the register with a correct term count in § 10, no renumbered identifier, and no reworded existing row.
5. Every `V-*` citation in every downstream milestone's artifacts resolves to a live row here.

Criteria 1 through 3 were verified at AI-01.1's merge and recorded in the archived `verify-report.md`: 114 rows, zero duplicate identifiers, zero duplicate term names, complete ordinal contiguity, one owning milestone on each of the 97 non-excluded rows, all 17 defect and gap identifiers traceable, both wording traps byte-identical to their source, and zero Go identifiers. Criteria 4 and 5 bind every amendment thereafter.
