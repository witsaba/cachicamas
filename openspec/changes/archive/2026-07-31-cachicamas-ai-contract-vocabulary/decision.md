# Decision — the Layer 1 contract vocabulary

> **Change**: `cachicamas-ai-contract-vocabulary`
> **Milestone**: AI-01 — Record the Layer 1 contract vocabulary
> **Node**: AI-01.1 — The vocabulary `[decision]`
> **Status**: decided
> **Date**: 2026-07-31
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Sources**: [doc 0002 — Layer 1 task graph](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0001 — agent stack v2](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md) · [ADR 0006](../../../../docs/adr/0006-resolve-skill-and-prompt-source-of-truth.md)

> [!IMPORTANT]
> **This artifact names concepts, not code.** No Go type name, field name, method name, or package identifier appears here. Each owning milestone's own SDD cycle chooses spellings — that is doc 0002's authoring constraint and AI-01's explicit out-of-scope clause. A term here is a noun phrase and a definition; it is never an API.

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

Both are quoted verbatim from doc 0002's [Layer boundary](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#layer-boundary) section. They are restated here rather than paraphrased **because each has already caused one wrong decision**, and paraphrase is how the first one went wrong in the first place.

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

### 6.1 Caller-contract failures — AI-04

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `V-FAIL-01` | **caller-contract failure** | A violation of a construction or composition rule by the caller. **The caller's bug, knowable without any I/O.** Every rule violation in every Layer 1 contract reports through this one vocabulary. Deliberately defined before the first validating contract, because the retired plan defined it seventeenth and spent a milestone rationalizing ten milestones' worth of ad-hoc sentinels. | AI-04 | 0001 § 3.1 |
| `V-FAIL-02` | **validation sentinel** | The stable, matchable value identifying which rule a caller-contract failure violated. Matchable through at least one layer of wrapping, so a caller can classify a failure it received indirectly. | AI-04 | 0001 § 9 ("Errors are typed and inspectable") |
| `V-FAIL-03` | **positional context** | The attached information naming *where* a violation occurred — which message, which content index, which tool — without becoming a second, parallel failure vocabulary. | AI-04 | 0001 § 9 |
| `V-FAIL-04` | **first-failure ordering** | The documented, deterministic order in which rules are checked, so that a value violating several rules reports the same one on every run. Determinism is the property; whether to report the first violation or all of them is AI-04.1's decision. | AI-04 | 0001 § 9 |

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

## 10. Closing-checklist verification

AI-01.1's six items, each against this artifact.

| # | Closing-checklist item | Where answered | Status |
| --- | --- | --- | --- |
| 1 | Request-side terms: role, message, content part and its kinds, tool declaration, tool choice, tool call, tool result, system-instruction segment, cache-boundary marker, generation option, provider escape hatch | § 3 — `V-REQ-01` … `V-REQ-29`; all eleven named terms present | **answered** |
| 2 | Stream-side terms: event, event kind, payload, sequence, stream, terminal event, delta, block index, call ordinal | § 4 — `V-STR-01` … `V-STR-23`; all nine named terms present, with `call ordinal` grouped here and owned by AI-09 | **answered** |
| 3 | Metadata terms: finish reason, usage, token-count field, absence versus zero | § 5 — `V-MET-01` … `V-MET-12`; all four named terms present, plus the complete closed finish-reason vocabulary | **answered** |
| 4 | Failure terms defined and separated: caller-contract (AI-04) versus provider/transport (AI-19), and the pre-stream versus mid-stream delivery split | § 6 — the two-axis diagram, `V-FAIL-01` … `V-FAIL-15`, the decidability-without-I/O rule, and four worked borderline cases | **answered** |
| 5 | Terms explicitly excluded, with their owner named: agent turn, transcript, session, tool execution, permission, compaction, cost, price, frontend | § 8.1 — `V-OUT-01` … `V-OUT-09`, all nine with a non-Layer-1 owner and the Layer 1 neighbour each is confused with; § 8.2 appends eight more | **answered** |
| 6 | The two wording traps from the layer boundary restated, because both have already caused one wrong decision each | § 2 — both quoted verbatim, each with the record of its wrong decision and its operational consequence for this register | **answered** |

**Term count:** 29 request-side · 23 stream-side · 12 metadata · 15 failure · 18 provider-surface · 17 excluded = **114 terms**. *(Amended 2026-07-31: `V-STR-22`, `V-STR-23` appended by AI-02.1 — see the amendment blockquote in § 4. Amended 2026-07-31: `V-PRV-16`, `V-PRV-17`, `V-PRV-18` appended by AI-03.1 — see the amendment blockquote in § 7.)*

**Node status.** AI-01.1 closes on merge of this artifact. Per doc 0002's node grammar, a `[decision]` leaf produces no production code and closes when "the decision artifact answers every listed question and is merged." No `make test` gate applies; there is nothing in `backend/agent/` that this change touches.

**Unblocked by this decision:** AI-02 (`cachicamas-ai-stream-lifecycle`), AI-03 (`cachicamas-ai-minimum-capabilities`), and every contract milestone AI-04 … AI-40.
