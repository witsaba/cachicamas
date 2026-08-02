# Spec — streamed reasoning block events

> **Milestone**: AI-17 — Add reasoning delta events (Wave 2 "Stream") · **Nodes**: AI-17.1 `[leaf]` block lifecycle, AI-17.2 `[leaf]` token delivery, AI-17.3 `[leaf]` redacted and signature-only streams
> **Introduced by**: `openspec/changes/archive/2026-08-01-cachicamas-ai-reasoning-events/`, merged to `main` in PR #104 (commit `37898c7`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: the stream half of **G12(b)** — the reasoning round-trip token now survives the event boundary byte-exactly, arriving whole on the block-end event. AI-07 carried it on a finished part; this file carries it on the wire side. Wire proof is AI-29.2 and AI-26.6
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-ARE-0NN` · **Scenario IDs**: `S-ARE-0NN`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md) — `V-REQ-09` reasoning content, `V-REQ-10` reasoning state, `V-REQ-11` round-trip token, `V-STR-10` event, `V-STR-11` event kind, `V-STR-12` payload, `V-STR-14` block, `V-STR-15` block index, `V-STR-16` delta, `V-STR-17` ordering invariant
> **Binding predecessor**: [reasoning content](../ai-reasoning-content/spec.md) (AI-07) — `R-ARC-002`, `R-ARC-004` … `R-ARC-009`, consumed unchanged; `reasoning_content.go` is diff-free across this milestone
> **Binding predecessor**: [the event envelope](../ai-event-envelope/spec.md) (AI-14) — the kind registry, the `BlockIndex` type and the descriptor-driven ordering checker these three kinds register into
> **Binding predecessor**: [indexed text block events](../ai-text-events/spec.md) (AI-16) — `R-ATE-003` and `R-ATE-004`, adopted verbatim: one 1-based stream-wide block-index space, 0 rejected
> **Binding predecessor**: [the validation error taxonomy](../ai-validation-errors/spec.md) (AI-04) — every rule violation here reports through AI-04's failure value and its landed sentinels
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-17--add-reasoning-delta-events) §§ AI-17.1 … AI-17.3 (lines 968–998) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) § 3.1, § 3.2, § 4.3 invariant 1 · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

---

## Purpose

Layer 1 can describe *finished* reasoning content with its opaque round-trip token (AI-07). It cannot describe reasoning **arriving**. This spec constrains the three events a streamed reasoning block produces — start, delta, end — so that reasoning can never reach a consumer as assistant text, the round-trip token survives the event boundary byte-identically, and the redacted and signature-only shapes are streamable rather than only constructible.

Two things this spec is careful *not* to constrain: how a consumer accumulates (doc 0001 § 4.3 invariant 1 — Layer 1 ships no helper), and runtime enforcement of event ordering, which AI-14.4 states and AI-22.3/AI-23 package.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The three reasoning-block kinds, the whole-token-on-block-end rule, the presence-typed token accessor, and the redacted and signature-only shapes therefore live here, in their own text, and not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-01-cachicamas-ai-reasoning-events/`](../../changes/archive/2026-08-01-cachicamas-ai-reasoning-events/) is the historical record of how AI-17 was explored, proposed, designed, applied and verified.

`R-ARE-002` is the requirement this capability exists to protect: reasoning is a **structurally distinct family**, not a flag on a text event. It is satisfied at the type level — the three reasoning payload types are unrelated to the three text payload types, no conversion between them compiles, and no shared content-family flag exists — so a consumer that switches on event kind alone cannot render reasoning as assistant output. Any later milestone that introduces a shared flag reopens this requirement.

`R-ARE-013`'s `[provisional]` sentence is **resolved**. See the requirement itself for the settled rule and the recorded rationale.

## Adopted from AI-16, stated once

[`ai-text-events`](../ai-text-events/spec.md)'s `R-ATE-004` declares one block-index space per stream, shared across all content families, and states it as **binding on AI-17**. Its `R-ATE-003` fixes that space as **1-based, with 0 a rejected sentinel**. This spec adopts both verbatim: the shared space is 1-based, the first block of a stream carries index 1, and 0 is rejected at construction. No 0-based reasoning-block index exists anywhere in this capability. AI-18's tool-call *ordinal* remains a distinct concept from the block index.

## Definitions used by this spec

- **A reasoning block** — `V-STR-14` narrowed to reasoning: one contiguous run of streamed reasoning, delimited by a reasoning block start and a reasoning block end, with zero or more reasoning deltas between them.
- **The block index** — `V-STR-15`, in the single stream-wide space of `R-ATE-004`.
- **A fragment** — the *new* reasoning-**text** bytes a delta carries; `V-STR-16`'s "only the new fragment", never a snapshot, never token bytes.
- **Reconstruction** — ordered concatenation of one block's delta fragments, plus the block's redacted signal from its start event and its token from its end event, producing the block's complete reasoning shape.
- **The derived state** — `ReasoningState` computed from a reconstructed block by exactly the derivation `Reasoning.State()` applies (`R-ARC-004`): redacted wins, else non-empty text is `ReasoningStateText`, else `ReasoningStateTokenOnly`.

---

## AI-17.1 — Reasoning block lifecycle

### R-ARE-001 — The family is three separately registered event kinds

Layer 1 MUST expose exactly three reasoning-block event kinds — reasoning block start, reasoning delta, reasoning block end — each registered independently in AI-14's event-kind registry, each with its own constructible payload (`V-STR-12`), each kind derived from its payload (`V-STR-11`). The family MUST NOT be modelled as one kind carrying an internal phase discriminator.

#### Scenarios

- **S-ARE-001** — Given the landed registry, when a consumer in another package names each of the three kinds, then each compiles, all three are distinct values, and each is a member of the registry's closed kind set.
- **S-ARE-002** — Given each of the three kinds, when a consumer constructs its payload and reads the resulting event's kind, then the kind matches the payload and no separate phase field is exposed on any of the three.
- **S-ARE-003** — Given AI-14's exhaustiveness guard, when the suite runs, then the guard covers all three new kinds; and given a scratch unregistered kind, when the suite runs, then the guard fails and names it.

### R-ARE-002 — Reasoning events are a structurally distinct family, not a flag on a text event

The three reasoning kinds MUST be distinct from AI-16's three text kinds at the type level. Layer 1 MUST NOT expose any field, parameter, option or constructor by which a text block start, text delta or text block end can be made to carry reasoning, and MUST NOT expose a shared "content family" flag that a consumer must check to tell the two apart. A consumer that omits any check MUST NOT be able to render reasoning as assistant output: the event kind alone MUST decide the family. This is `R-ARC-002` one level down, on the wire side of the contract.

#### Scenarios

- **S-ARE-004** — Given the landed package surface, when a reviewer enumerates every exported constructor and field of the three text kinds, then none accepts reasoning content, a redacted signal, or a round-trip token.
- **S-ARE-005** — Given a reasoning delta and a text delta carrying identical fragment bytes, when a consumer compares them, then their event kinds differ, their payload types differ, and no assignment or conversion between the two payload types compiles.
- **S-ARE-006** — Given a consumer that switches only on event kind and handles no flag, when a stream carrying both families is processed, then every reasoning event is classified as reasoning and no reasoning byte reaches the text path.

### R-ARE-003 — The producer stamps a 1-based block index on all three events; 0 is rejected

Every reasoning block start, reasoning delta and reasoning block end MUST carry a block index stamped by the producer at emission time, readable from an external package without parsing content. The index MUST NOT be derived from position in a sequence. The first block of a stream MUST carry block index **1**. A block index of **0** MUST be rejected at construction with AI-04's out-of-range sentinel at a position naming the index field, so that an event whose index was never stamped is never indistinguishable from a legal block. (Adopted from `R-ATE-003`.)

#### Scenarios

- **S-ARE-007** — Given a producer emitting a start, two deltas and an end for one reasoning block, when a consumer in another package reads each of the four events, then each reports the same block index.
- **S-ARE-008** — Given a reasoning block start, delta or end constructed with block index 0, when construction runs, then it fails, `errors.Is` reports the out-of-range sentinel, and the position renders the block-index field.
- **S-ARE-009** — Given each of the three kinds in turn constructed from its zero-value payload, when construction runs, then it fails rather than yielding an event carrying block index 0.

### R-ARE-004 — Reasoning blocks share one stream-wide block-index space with every other family

A stream MUST have exactly one block-index space (`R-ATE-004`). A reasoning block's index MUST NOT collide with a text block's or a tool-call block's index on the same stream. Layer 1 MUST NOT introduce a per-family numbering space for reasoning, and MUST NOT require a family tag to disambiguate an index.

#### Scenarios

- **S-ARE-010** — Given a stream carrying two interleaved reasoning blocks whose events arrive out of block order, when a consumer partitions events by block index alone, then every event lands in its own block and none is misattributed.
- **S-ARE-011** — Given a stream in which a reasoning block and a text block are both open, when a consumer reads both indices, then the two differ, and the consumer separates the blocks without consulting the event kind.
- **S-ARE-012** — Given the landed surface, when a reviewer looks for a reasoning-only index space, a 0-based reasoning index, or a family tag needed to disambiguate an index, then none exists.

### R-ARE-005 — A reasoning delta carries a text fragment only

A reasoning delta MUST carry only the reasoning-**text** bytes new since the previous delta of its block. It MUST NOT carry the block's accumulated reasoning text, MUST NOT carry round-trip token bytes, and MUST NOT carry a redacted signal or a `ReasoningState`. Layer 1 MUST NOT expose an accessor on any of the three kinds returning accumulated block content.

#### Scenarios

- **S-ARE-013** — Given a block whose deltas carry `"a"`, `"b"` and `"c"`, when a consumer reads each delta's fragment, then it reads exactly `"a"`, `"b"` and `"c"` — never `"a"`, `"ab"`, `"abc"`.
- **S-ARE-014** — Given the landed surface of the three kinds, when a reviewer enumerates their exported accessors, then none returns accumulated content, and none on the delta kind returns a token, a redacted signal or a state.

### R-ARE-006 — Concatenated deltas reconstruct the reasoning text byte-exactly

For any reasoning block, the ordered concatenation of its delta fragments MUST equal the block's complete reasoning text byte-for-byte, with no normalization, trimming or re-encoding required of the consumer. A fragment MUST be defined and documented as a sequence of raw **bytes**; an individual fragment MAY be invalid UTF-8, and construction MUST NOT reject, repair, replace or re-encode it for that reason.

#### Scenarios

- **S-ARE-015** — Given a known reasoning text split into an arbitrary number of fragments, when a test-local concatenator joins the block's deltas in arrival order, then the result is byte-identical to the original text, including leading and trailing whitespace and interior newlines.
- **S-ARE-016** — Given a multi-byte rune split across a delta boundary, when both deltas are constructed, then both succeed, neither fragment's bytes are altered, their concatenation decodes to the original rune, and no replacement character appears in the reconstruction.
- **S-ARE-017** — Given the declaration of the fragment, when a reader looks for its encoding statement, then it uses the word **byte** and records that a single fragment may not be well-formed UTF-8 on its own.

### R-ARE-007 — No complete-part rule applies to a fragment, and a zero-delta block is legal

A delta fragment MUST NOT be validated by `Reasoning.validate`'s rules for a *complete* reasoning part: a whitespace-only fragment MUST be accepted and a zero-length fragment MUST be accepted, with no `ErrEmpty`. A fragment MUST be rejected with AI-04's out-of-range sentinel only when it exceeds the existing `MaxTextLen` bytes; no new bound is invented. A reasoning block start immediately followed by its matching block end, with zero deltas, MUST be legal and MUST reconstruct to empty reasoning text — it is not a failure, a truncation, or a distinguishable error shape.

#### Scenarios

- **S-ARE-018** — Given a reasoning delta whose fragment is a single space, and one whose fragment is zero-length, when each is constructed, then both succeed and no emptiness sentinel is reported.
- **S-ARE-019** — Given a fragment of exactly `MaxTextLen` bytes and one of `MaxTextLen + 1` bytes, when each is constructed, then the first succeeds and the second fails with the out-of-range sentinel at a position naming the fragment.
- **S-ARE-020** — Given a stream containing one zero-delta reasoning block and one multi-delta reasoning block, when both are validated and reconstructed by index alone, then both are valid, the zero-delta block reconstructs to empty text, and it is not confused with an unterminated block.

### R-ARE-008 — Layer 1 ships no public accumulation or reconstruction helper

Layer 1 MUST NOT export an accumulator, transcript rebuilder, or any function reducing a reasoning block's deltas to its complete text or its derived state. Doc 0001 § 4.3 invariant 1 reserves accumulation for the consumer. Byte-exactness MUST be proven by this milestone's own tests using a test-local concatenator.

#### Scenarios

- **S-ARE-021** — Given the landed package surface, when a reviewer enumerates its exported identifiers, then none accumulates, joins or reconstructs a block's deltas.
- **S-ARE-022** — Given the tests proving `R-ARE-006` and `R-ARE-007`, when a reader looks for the concatenator they use, then it is defined inside the test package and is not exported from the contract.

---

## AI-17.2 — Token delivery

### R-ARE-009 — The round-trip token arrives whole on the block-end event, and only there

The opaque round-trip token (`V-REQ-11`) MUST be carried by the reasoning **block-end** event and by no other event of this family. It MUST arrive whole: Layer 1 MUST NOT fragment a token across deltas, MUST NOT expose a token field on the block-start or delta kinds, and MUST NOT require or permit reassembly of token bytes from more than one event. (Decision D2.)

#### Scenarios

- **S-ARE-023** — Given the landed surface of the three kinds, when a reviewer enumerates their exported fields and accessors, then a token appears on the block-end kind only.
- **S-ARE-024** — Given a block whose token is longer than any single delta fragment in the same block, when the block is streamed, then the token is still delivered in one block-end event and no delta carries any token byte.

### R-ARE-010 — The token is byte-exact across the event boundary and is never interpreted

The token bytes a consumer reads from a block-end event MUST be byte-identical to the bytes the producer supplied, for every byte class AI-07.2 covers via `opaqueTokens()` — including empty, NUL-bearing, invalid-UTF-8, binary and maximally long values. Nothing on the construction or read path MUST parse, verify, decrypt, normalize, trim, re-encode or measure the token for meaning. The token MUST be bounded only by the existing `MaxReasoningTokenLen`, reported with AI-04's out-of-range sentinel; no second bound and no new constant MUST be introduced. Aliasing MUST NOT be observable: mutating the caller's buffer after construction, or a second consumer's copy of the read result, MUST NOT change the token an event reports.

#### Scenarios

- **S-ARE-025** — Given each byte class in AI-07's existing `opaqueTokens()` fixture, when it is carried on a reasoning block-end event and read back in another package, then the bytes are identical to the input, byte for byte and length for length.
- **S-ARE-026** — Given a token supplied as a caller-owned slice, when the caller mutates that slice after construction, and separately when one reader mutates the slice it received, then the event's token is unchanged in both cases.
- **S-ARE-027** — Given a token of exactly `MaxReasoningTokenLen` bytes and one of `MaxReasoningTokenLen + 1`, when each block-end event is constructed, then the first succeeds and the second fails with the out-of-range sentinel at a position naming the token field.

### R-ARE-011 — Absent token and empty token are distinguishable

The block-end event MUST report its token through a two-result accessor whose second result states **presence**, mirroring `Reasoning.Token() ([]byte, bool)` and `R-ARC-005` verbatim. A reasoning block with **no** token MUST be valid, and MUST be distinguishable from a block carrying a **zero-length** token. Presence MUST be stored beside the bytes rather than inferred from their length, so that no copy, clone or conversion can collapse the two.

#### Scenarios

- **S-ARE-028** — Given a block-end event constructed with no token, when a consumer reads the accessor, then the second result is false; and given one constructed with a zero-length token, then the second result is true and the byte slice has length 0.
- **S-ARE-029** — Given those two block-end events, when a consumer compares them, then they are not equal, and the difference is readable without inspecting byte length.
- **S-ARE-030** — Given a reasoning block whose start, deltas and end carry no token at all, when the block is validated and reconstructed, then it is valid and its derived state is `ReasoningStateText`.

---

## AI-17.3 — Redacted and signature-only streams

### R-ARE-012 — Redaction is signalled once, on the block-start event

The redacted signal MUST be carried by the reasoning **block-start** event and by no other event of this family. No event of this family MUST carry, store or expose a `ReasoningState`: the block's state MUST be **derived** from the reconstructed block by the same derivation `Reasoning.State()` applies (`R-ARC-004`), so no two events can disagree about it. A consumer MUST be able to learn that a block's plaintext is withheld from the start event alone, before any delta arrives. (Decision D1.)

#### Scenarios

- **S-ARE-031** — Given the landed surface of the three kinds, when a reviewer enumerates their exported fields and accessors, then the redacted signal appears on the block-start kind only, and no kind exposes a `ReasoningState` field or accessor.
- **S-ARE-032** — Given a redacted block's start event, when a consumer reads it before any further event arrives, then it reports that the block is redacted.
- **S-ARE-033** — Given a reconstructed block, when its state is derived, then the derivation consults the start event's redacted signal, the concatenated fragments and the end event's token, and returns the same state `Reasoning.State()` returns for the equivalent constructed part.

### R-ARE-013 — A redacted block streams its opaque payload verbatim and derives the redacted state

A block whose start event signals redaction MUST reconstruct to `ReasoningStateRedacted`, and its opaque payload MUST be preserved verbatim under `R-ARE-010`. A redacted block MUST carry a token: a redacted block whose end event carries no token MUST be rejected with `ErrEmpty` at a position naming the token, mirroring `Reasoning.validate` rule 3. A redacted block MUST NOT carry reasoning text: a **non-empty** delta on a block whose start signalled redaction MUST be rejected with AI-04's `ErrMisplaced` at a position naming the fragment. A **zero-length** fragment on a redacted block is accepted, because it carries nothing.

**Why rejection and not carry-and-report.** Three reasons, recorded so the rule is not relitigated. First, AI-07 parity: `Reasoning` has no redacted-plus-text shape — `NewRedactedReasoning(token []byte)` accepts no text, and `State()` makes redacted win — so a wire shape the part model cannot represent would have no "equivalent constructed part", making `S-ARE-033` unsatisfiable and re-creating doc 0001 § 3.1's two-strategies defect one level up. Second, redaction means plaintext *withheld* (`V-REQ-10`); text beside it is two contradictory signals, and passing both through violates the package's rule against a second copy that two writers can disagree about. Third, it is decidable at construction, matching AI-04's caller-contract posture — the delta constructor takes the typed block-start payload, so the block's redaction bit is visible where the rejection has to happen.

#### Scenarios

- **S-ARE-034** — Given a redacted reasoning block whose end event carries an opaque payload from `opaqueTokens()`, when the block is reconstructed, then its derived state is `ReasoningStateRedacted` and the payload is byte-identical to the input.
- **S-ARE-035** — Given a redacted block whose end event carries no token, when the block is validated, then it fails, `errors.Is` reports `ErrEmpty`, and the position names the token field.
- **S-ARE-036** — Given a block whose start event signalled redaction, when a delta carrying a non-empty fragment is constructed for that block, then it is rejected and the failure names the offending field.

### R-ARE-014 — A token-and-no-text block streams as valid signature-only reasoning

A reasoning block with zero deltas (or only zero-length fragments), no redacted signal, and a token on its end event MUST be valid and MUST reconstruct to `ReasoningStateTokenOnly`. It MUST be distinguishable from a redacted block carrying the same token bytes, and from a text block with no reasoning at all. This is AI-07.4's shape on the wire side of the contract.

#### Scenarios

- **S-ARE-037** — Given a reasoning block start with no redacted signal, no deltas, and a block end carrying a token, when the block is validated and reconstructed, then it is valid, its text is empty, and its derived state is `ReasoningStateTokenOnly`.
- **S-ARE-038** — Given that block and a redacted block carrying byte-identical token bytes, when both are reconstructed, then their derived states are `ReasoningStateTokenOnly` and `ReasoningStateRedacted` respectively, and the two are not equal.
- **S-ARE-039** — Given a reasoning block with neither text, nor a token, nor a redacted signal, when it is validated, then it reconstructs to a legal empty reasoning block under `R-ARE-007` and is never reported as redacted or signature-only.

---

## Non-functional requirements

### NFR-ARE-A — Dependency purity and non-modification of AI-07

The change MUST add no module dependency: `backend/agent/go.mod` MUST still carry zero requires and both AI-00 import guards MUST still pass. `reasoning_content.go` and `reasoning_content_test.go` MUST be unchanged; `ReasoningState` and `MaxReasoningTokenLen` MUST NOT be edited, and `opaqueTokens()` MUST be reused rather than reinvented.

- **S-ARE-040** — Given the change merged, when `go.mod` is read and both import guards run, then no require is declared and both guards pass; and when the diff is inspected, then neither reasoning-content file appears in it.

### NFR-ARE-B — Totality

No exported function or method of this contract MUST panic for any input, including each payload's zero value, block index 0, a zero-length fragment, an invalid-UTF-8 fragment, an over-long fragment, a nil token, a zero-length token, and an over-long token.

- **S-ARE-041** — Given a table of those extreme inputs, when each is passed through every exported entry point of the three kinds, then none panics.

### NFR-ARE-C — Failure reporting

Every rule violation in this spec MUST be reported through AI-04's existing failure value and its landed sentinels. No new sentinel and no second failure type MUST be introduced.

- **S-ARE-042** — Given each rejecting scenario above, when its failure is inspected, then it is AI-04's failure value, `errors.Is` matches a landed sentinel, and its position names a field of the offending payload.

### NFR-ARE-D — Evidence

Every test-list item of AI-17.1, AI-17.2 and AI-17.3 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) in `backend/agent/` and clean `make lint`.

- **S-ARE-043** — Given `tasks.md`, when a reviewer walks the six test-list items, then each carries recorded red output, recorded green output, and a refactor note.

---

## Acceptance criteria

1. Three separately registered reasoning event kinds, covered by AI-14's exhaustiveness guard, shown to bite against a scratch kind.
2. Reasoning is a structurally distinct family: no flag on a text event, no text constructor accepting reasoning, no shared payload type.
3. All three events carry a producer-stamped block index, **1-based**, with **0** rejected at construction by AI-04's out-of-range sentinel.
4. The block-index space is one per stream and shared with text (AI-16) and tool calls (AI-18); reasoning and text indices never collide and need no family tag.
5. Deltas carry text fragments only — never accumulated text, never token bytes, never a state.
6. Concatenated deltas reconstruct byte-exactly, including a boundary that splits a multi-byte rune; whitespace-only and zero-length fragments are legal; a zero-delta block reconstructs to empty.
7. The token arrives whole on the block-end event only, byte-identical for every `opaqueTokens()` class, bounded solely by `MaxReasoningTokenLen`, with no observable aliasing.
8. A block with no token is valid and distinguishable from one with an empty token, through a two-result presence accessor.
9. Redaction is signalled on the block-start event only; no event carries a `ReasoningState`; the state is derived exactly as `Reasoning.State()` derives it.
10. A redacted block preserves its opaque payload verbatim and derives `ReasoningStateRedacted`; a token-and-no-text block derives `ReasoningStateTokenOnly` and is distinguishable from it.
11. No public accumulator ships; byte-exactness is proven with a test-local concatenator.
12. `make test` green under `-race`, `make lint` clean, both import guards passing, `go.mod` still zero requires, `reasoning_content.go` untouched.

## Deliberately out of scope

Whether a consumer may *display* redacted reasoning at all is a Layer 2 rendering policy. This spec takes no position beyond `R-ARE-012` making the redacted signal available before the first delta arrives, which is what a rendering policy needs in order to exist.
