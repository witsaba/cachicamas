# Explore — Layer 1 contract vocabulary

> **Change**: `cachicamas-ai-contract-vocabulary`
> **Milestone**: AI-01 — Record the Layer 1 contract vocabulary
> **Node**: AI-01.1 — The vocabulary `[decision]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Target module**: `backend/agent/` — **no code is written by this change**
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005 §§ D1, D3, D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Predecessor**: AI-00 (`cachicamas-agent-module-scaffold`) — the module and both import guards
> **Authoring constraint**: doc 0002's authoring constraint binds this whole change. The vocabulary is **conceptual**. No Go type name, field name, or signature appears in any artifact of this change.

---

## 1. What this milestone is, in one paragraph

AI-01 is the cheapest milestone in Layer 1 and the only one that can prevent an entire class of later argument. It ships zero Go. Its deliverable is a recorded vocabulary: for every concept Layer 1 exposes, **one** definition, **one** owning milestone, and — for the concepts Layer 1 deliberately does *not* own — the layer that does. Every subsequent milestone charter, from AI-02 to AI-40, must be writable using only these terms. A term discovered missing is appended by amendment; it is never invented inside a pull request.

## 2. Why naming late is the failure mode this milestone exists to prevent

The retired 2026-07-30 plan defined its vocabulary milestone *first in identifier order* but let the contracts that use those nouns be built before the nouns were settled. The consequences are recorded, not hypothetical, and all four are naming failures before they are code failures:

| Defect | The naming failure underneath it | doc 0001 |
| --- | --- | --- |
| **C1** — an exported text value satisfied the content-part interface directly, so its zero value passed validation | "content part" was never defined as *a value that cannot exist without passing its construction rules*. Sealing was designed after the type, so the type's identity did not include it | § 3.1 |
| **C2** — content parts were unreadable from another package, so request translation was structurally impossible | "content part" was never defined as *a value whose payload is readable from outside the package that owns it*. Readability was not part of the noun, so nothing enforced it | § 3.1 |
| **C3** — the sequence counter was a process-global atomic, so only the first stream in a process satisfied the documented contract | "sequence" was defined without first defining "stream". A counter written before anything owned a stream necessarily lives at process scope | § 3.1 |
| **C4** — the provider interface declared a mandatory terminal error event whose payload no adapter could construct | "terminal event" and "provider/transport failure" were named in the interface's documentation before either had a definition with a constructible referent | § 3.1 |

Four additional gaps — **G4** (cache breakpoints), **G9** (per-request options and the escape hatch), **G12(b)** (the reasoning round-trip token) and **G12(c)** (refusal and pause finish reasons) — each became a *breaking change to a frozen surface* for the same reason: the noun that would have carried them ("system-instruction segment", "provider escape hatch", "round-trip token", "finish reason") did not exist when the surface froze. doc 0002 folds all four into their defining milestones (AI-11, AI-12, AI-07, AI-13) precisely because a term named at birth costs nothing and a term retrofitted costs a migration of every adapter.

**The generalization this milestone records:** every one of the eight is a case where a contract was frozen around a noun whose definition was still open. The countermeasure is not more review; it is to close the nouns before the first contract milestone starts. That is why AI-01 depends only on AI-00 and blocks AI-02, AI-03, and every contract milestone.

## 3. What vocabulary Layer 1 actually needs

Derived by walking doc 0002's milestone charters AI-02 … AI-40 and collecting every concept a charter, closing checklist, or test list names. Five categories emerge, and they map exactly onto AI-01.1's closing checklist items 1–5.

### 3.1 Request-side — what goes to the model

Sources: doc 0001 § 3.2 (what must become expressible), § 3.3 rows 1–9 (the leakage register), § 6 seams 9, 10, 11. Owning milestones AI-05 … AI-12.

The concepts, and where each is pinned:

- **role**, **message**, **message identity** — AI-05's charter: "the smallest addressable unit of a transcript: a message with a role and ordered content." Note the phrasing: a message is Layer 1's; *transcript* is not.
- **content part**, its **kind** discriminator, and the two properties that must be defined together — **readability** from another package and **sealing** against unconstructed values — AI-06. doc 0002 calls AI-06 "the keystone of wave 1" and notes the two properties are in tension: any strategy satisfying one but not the other has failed the milestone.
- **text content** — AI-06's first subject.
- **reasoning content**, its **state**, and the **round-trip token** — AI-07. doc 0001 § 3.3 row 2 and § 6 seam 11: the token is a correctness requirement, not metadata.
- **tool declaration**, **schema bytes**, **tool set**, **tool choice** — AI-08, whose charter is literally "the provider-neutral transport representation of the tools a model may call."
- **tool call**, **argument bytes**, **call ordinal**, **tool result** — AI-09. The ordinal is the Layer 1 obligation for **G5** (doc 0002's forward-requirements table: "met — the call ordinal survives normalization").
- **normalized request**, **model identity**, **system-instruction segment**, **request validation**, **generation option** — AI-10. The system instruction is segmented from birth (doc 0001 § 3.2, § 3.3 row 8, **G4**), which is exactly what the retired plan had to break its surface to add.
- **cache-boundary marker**, **breakpoint cap**, **invalidation cascade** — AI-11 (**G4**, seam 10).
- **per-request option override**, **provider escape hatch**, **request rebuild** — AI-12 (**G9**, seams 1 and 9).

### 3.2 Stream-side — what comes back, and what carries it

Sources: doc 0001 § 4.3 (the four envelope invariants, applied at Layer 1), § 6, **G13**. Owning milestones AI-02 and AI-14 … AI-20.

AI-02 owns the *container* nouns — **stream**, **carrier**, **producer**, **consumer**, **stream ownership**, **cancellation**, **abandonment**, **bounded buffer**, **sanctioned loss path** — and it is scheduled after AI-01 precisely so that it can argue about behavior using settled nouns. AI-14 owns the *content* nouns — **event**, **event kind**, **payload**, **sequence**, **block**, **block index**, **delta**, **ordering invariant**, **terminal event**. AI-15 owns the two lifecycle instances, **response-start event** and **completion event**.

The C3 lesson is structural here: **"stream" must be defined before "sequence"**, because the sequence is a property *of* a stream. Defining them in the other order is what produced a process-global counter. This exploration therefore fixes the definition order inside the stream category, and the artifact records it.

### 3.3 Metadata — what a response says about itself

Source: doc 0001 § 3.2 (refusal and pause), § 3.3 row 3, § 7 **G10** and **G12(c)**. Owning milestone AI-13.

**finish reason** with its closed vocabulary — natural stop, length, tool calls, content filter, **refusal**, **pause-turn**, unknown — plus **usage**, **token-count field**, and the distinction **absence versus zero**. doc 0001 § 7 records that G10 needs no Layer 1 work because usage already carries the cache and reasoning counts; what it *does* need is that an absent count is distinguishable from a zero count, or a cost formula built on top is silently wrong.

### 3.4 Failure — two vocabularies, deliberately separated

Sources: doc 0001 § 3.1 C4, § 4.3 invariant 4, § 6 seam 7, § 7 **G8**. Owning milestones AI-04 and AI-19.

This is the split with the highest cost of confusion, and doc 0002 schedules the two milestones at opposite ends of the plan to enforce it:

- **AI-04 — caller-contract failure.** An invalid request. The caller's bug. Knowable without any I/O. Scheduled **first** in wave 1, deliberately, because the retired plan defined it seventeenth after ten milestones had each invented their own sentinels.
- **AI-19 — provider/transport failure.** Anything that happens after a request leaves the process. Scheduled **before** AI-20, deliberately, because the retired plan's reverse ordering produced C4.

Crossing the two is the error the artifact must make hard. AI-04.1's own closing checklist demands "at least one borderline case resolved"; the vocabulary supplies the boundary that makes such a resolution possible.

The second split, orthogonal to the first, is **delivery**: a request that never becomes a stream (**pre-stream failure delivery**) versus a stream that dies mid-flight (**mid-stream failure delivery**). AI-02.1 item 5 decides the observable shape; AI-19.5 implements it as "one vocabulary, two delivery paths." doc 0001 § 7 G8 names the mid-stream case with partial output as "the most common real failure and the one naive retry logic excludes."

### 3.5 Provider surface and the proving apparatus

Sources: doc 0001 § 3.1, § 5.1 seam 6, § 7 G3. Owning milestones AI-03, AI-20 … AI-24.

**provider**, **adapter**, **provider interface**, **pre-stream contract**, **mid-stream contract** (AI-20); **required capability**, **optional capability**, **capability discovery**, **capability record** (AI-03); **fake provider** (AI-21), **stream test kit** (AI-22), **conformance suite** (AI-23); **transport**, **wire representation**, **frame** (AI-24, AI-26, AI-27).

This category is beyond the literal text of AI-01.1's checklist items 1–4, and it is included anyway, because AI-01's **acceptance** criterion is stronger than its checklist: *every subsequent milestone charter must be writable using only these terms*. AI-03's, AI-21's, AI-22's and AI-23's charters are not writable without them.

## 4. The trap terms — where a plausible reading is wrong

Six places where the obvious definition is the one that has already cost this project a milestone or a decision.

| # | Trap | The plausible-but-wrong reading | What the vocabulary must say |
| --- | --- | --- | --- |
| 1 | **"Layer 1 does not know what a tool is"** | Layer 1 must not mention tools at all | Too broad. Layer 1 owns the provider-neutral **transport representation** of tool declarations and tool calls, because those cross the model API. It must not execute tools, resolve tool names, or own application behavior. This wording has already caused one wrong decision (doc 0002, *Layer boundary*) |
| 2 | **"Provider swap is a config change"** | A new vendor is a configuration edit | Applies **only after adapters exist**. Switching between already-supported providers can be configuration-only; adding a new vendor still requires a new adapter unless it is genuinely compatible with an existing transport. Also already caused one wrong decision |
| 3 | **transcript** | Layer 1 holds the conversation | Layer 1 owns **message**, the smallest addressable unit *of* a transcript (AI-05 charter). The transcript itself — ordering across turns, the call/result pairing invariant, interruption repair — is Layer 2's harness (doc 0001 § 4.2) |
| 4 | **content part is "just data"** | A struct with public fields | Two properties are constitutive and in tension: readable from another package (**C2**) and impossible to produce without construction (**C1**). A definition carrying only one of them re-creates a shipped defect |
| 5 | **sequence** | A counter | A per-**stream** property. Defining it before "stream" is exactly what produced **C3**. Cross-stream comparison is permitted and meaningless |
| 6 | **error** | One notion of failure | Two vocabularies with two owners (AI-04, AI-19) and, inside the second, two delivery paths. Four things, one English word. The vocabulary must not let the word stand alone |

## 5. Where each category's terms come from — provenance summary

| Category | Primary doc 0001 sections | Defect / gap identifiers closed | Owning milestones |
| --- | --- | --- | --- |
| Request-side | § 3.2, § 3.3 rows 1–9, § 6 seams 9–11 | C1, C2, G4, G5, G9, G12(a), G12(b) | AI-05 … AI-12 |
| Stream-side | § 4.3 invariants 1–4, § 6, § 7 G13 | C3, G13 | AI-02, AI-14 … AI-18 |
| Metadata | § 3.2, § 3.3 row 3, § 7 G10 | G12(c), G10 (Layer 1 half already met) | AI-13 |
| Failure | § 3.1 C4, § 4.3 invariant 4, § 6 seam 7, § 7 G8 | C4, G8 | AI-04, AI-19, AI-35, AI-36 |
| Provider surface | § 3.1, § 5.1, § 6 seam 6, § 7 G3 | G3 (Layer 1 half) | AI-03, AI-20 … AI-24 |
| Excluded | § 4.1, § 4.2, § 4.3, § 5.1, § 5.2, § 7 G1–G7, G10, G11 | G1, G2, G3, G5, G6, G7, G10, G11 | Layer 2, Layer 3, composition root |

## 6. Structural options considered

| Option | Shape | Verdict |
| --- | --- | --- |
| **A — glossary only** | Alphabetical term → definition | Rejected. An alphabetical list cannot express ownership, and ownership is the property that stops two milestones from both defining a noun |
| **B — per-milestone appendix** | Each milestone's SDD defines its own terms | Rejected. This *is* the retired plan's failure mode: a term defined by whoever needed it first, in the shape convenient at that moment |
| **C — categorized register with stable term identifiers, one owner per row** | Six categories; each term has an identifier, one definition, one owning milestone, and a provenance citation | **Selected.** Ownership is a column, so double-ownership is visible on inspection. Stable identifiers make amendment append-only, matching doc 0002's identifier discipline |
| **D — machine-checkable schema (YAML/JSON) plus rendered prose** | Definitions in structured data, prose generated | Rejected for v1. There is no CI (doc 0002's repo-wide caveat), so the check would never run, and the artifact's audience is humans writing charters. Reconsider if CI ever exists |

Option C also makes the *excluded* list a first-class part of the same register rather than an afterthought, which is what AI-01.1 checklist item 5 requires.

## 7. Out of scope for this change

1. **Any Go identifier.** Type names, field names, method names, package names. Each milestone's SDD chooses spellings (AI-01 charter, *Out of scope*).
2. **Any production code, test, or `go.mod` change.** The module stays dependency-free until AI-24 (doc 0002, *Rules for every future SDD milestone*).
3. **Deciding the stream carrier.** AI-02 owns it, and doc 0002 states the choice is genuinely free there. This change defines the noun **carrier** so AI-02 can argue about it; it does not choose.
4. **Deciding the capability matrix.** AI-03 owns required/optional/excluded and the discovery mechanism. This change defines **capability**, **capability discovery** and **capability record** as nouns only.
5. **Deciding the content-part strategy.** AI-06.1 owns it. This change records that readability and sealing are both constitutive; it does not pick the mechanism that delivers them.
6. **Deciding validation-error granularity, aggregation, or positional-context shape.** AI-04.1 owns all three.
7. **Amending doc 0001 or ADR 0005.** doc 0002 already records that ADR 0005's Context and Migration narrative is stale and that amending it is a separate change. This change cites both documents; it edits neither.
8. **Layer 2 and Layer 3 vocabulary.** Named only in the exclusion register, with an owner, never defined.

## 8. Open questions carried into the proposal

1. Should the vocabulary live only under `openspec/changes/…/decision.md`, or also be mirrored into the module's documentation once `backend/agent/` exists? *(Proposal answers: the change artifact is canonical for v1; a mirror is an AI-40 concern at the freeze.)*
2. Do the proving-apparatus terms (§ 3.5) belong in the vocabulary, given they are not in checklist items 1–4? *(Proposal answers: yes — the acceptance criterion is stronger than the checklist.)*
3. How is an amendment recorded? *(Proposal answers: doc 0002's convention — a dated blockquote under the touched heading, struck-through superseded text, never a silent edit.)*

## 9. Evidence gate for this milestone

AI-01.1 is a `[decision]` leaf. Per doc 0002's node grammar, it produces **no production code** and closes when "the decision artifact answers every listed question and is merged." There is no `make test` gate, because there is nothing in `backend/agent/` for this change to test. The gate is the closing checklist, item by item, verified against `decision.md` in review.
