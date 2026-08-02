# Spec — indexed text block events

> **Milestone**: AI-16 — Add text delta events (Wave 2 "Stream") · **Nodes**: AI-16.1 `[leaf]` lifecycle, AI-16.2 `[leaf]` byte fidelity, AI-16.3 `[leaf]` zero-delta blocks
> **Introduced by**: `openspec/changes/archive/2026-08-01-cachicamas-ai-text-events/`, merged to `main` in PR #104 (commit `37898c7`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: no tracked gap row of its own. `R-ATE-003` and `R-ATE-004` are the wave's coordination ground: they fix the shared, stream-wide, 1-based block-index space that binds AI-17 and AI-18, and they are the reason a stream mixing three content families reads one indexing convention rather than three. Wire translation is AI-28
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-ATE-0NN` · **Scenario IDs**: `S-ATE-0NN`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md) — `V-STR-10` event, `V-STR-11` event kind, `V-STR-12` payload, `V-STR-13` sequence, `V-STR-14` block, `V-STR-15` block index, `V-STR-16` delta, `V-STR-17` ordering invariant
> **Binding predecessor**: [the event envelope](../ai-event-envelope/spec.md) (AI-14) — the kind registry, the `BlockIndex` type and the descriptor-driven ordering checker these three kinds register into
> **Binding predecessor**: [response lifecycle events](../ai-response-events/spec.md) (AI-15) — the first registrants, whose eager-validating constructor shape these three follow
> **Binding predecessor**: [the validation error taxonomy](../ai-validation-errors/spec.md) (AI-04) — every rule violation here reports through AI-04's failure value and its landed sentinels
> **Binds**: AI-17 and AI-18, which MUST NOT introduce a per-family block-index space (`R-ATE-004`)
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-16--add-text-delta-events) §§ AI-16.1 … AI-16.3 (lines 937–966) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) § 4.3 invariant 1 · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

---

## Purpose

Layer 1 can describe a *finished* text content part (AI-06). It cannot describe text **arriving**. This spec constrains the three events a streamed text block produces — start, delta, end — so that a consumer can attribute every fragment to its own block while blocks interleave, and reconstruct the exact bytes the model produced.

Two things this spec is careful *not* to constrain: how a consumer accumulates (doc 0001 § 4.3 invariant 1 — accumulation is the consumer's, and Layer 1 ships no helper for it), and runtime enforcement of event ordering, which AI-14.4 states and AI-22.3/AI-23 package.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The three text-block kinds, the block-index rules, the byte-fidelity guarantee and the zero-delta legality therefore live here, in their own text, and not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-01-cachicamas-ai-text-events/`](../../changes/archive/2026-08-01-cachicamas-ai-text-events/) is the historical record of how AI-16 was explored, proposed, designed, applied and verified.

`R-ATE-004` is the requirement with the widest reach in this file: it binds every later content family to one stream-wide block-index space. Wave 2 satisfied it at the type level rather than by convention — exactly one `BlockIndex` type exists, declared by AI-14, with one unexported `blockPayload` interface implemented by nine payloads across three families and no second index type, no per-family counter and no family tag anywhere. A later family that needs a block index adopts that type; introducing a second one reopens this requirement.

`R-ATE-009`'s `[provisional]` marker is **resolved**. See the requirement itself for the settled rule and the recorded rationale.

## Definitions used by this spec

- **A text block** — `V-STR-14` narrowed to text: one contiguous run of streamed text, delimited by a start event and an end event, with zero or more deltas between them.
- **The block index** — `V-STR-15`. The value attributing an event to its own block.
- **A fragment** — the *new* bytes a delta carries; `V-STR-16`'s "only the new fragment", never a snapshot.
- **Reconstruction** — ordered concatenation of one block's delta fragments, producing the block's complete bytes.

---

## AI-16.1 — Text block lifecycle

### R-ATE-001 — The family is three separately registered event kinds

Layer 1 MUST expose exactly three text-block event kinds — text block start, text delta, text block end — each registered independently in AI-14's event-kind registry, each with its own constructible payload (`V-STR-12`). The family MUST NOT be modelled as one kind carrying an internal phase discriminator, because that would make the kind disagree with its payload and hide the family from AI-14's exhaustiveness guard.

#### Scenarios

- **S-ATE-001** — Given the landed registry, when a consumer in another package names each of the three kinds, then each compiles, all three are distinct values, and each is a member of the registry's closed kind set.
- **S-ATE-002** — Given each of the three kinds, when a consumer constructs its payload and reads the resulting event's kind, then the kind is derived from the payload and matches, and no separate phase field is exposed on any of the three.
- **S-ATE-003** — Given AI-14's exhaustiveness guard, when the suite runs, then the guard covers all three new kinds; and given a scratch unregistered kind, when the suite runs, then the guard fails and names it.

### R-ATE-002 — The producer stamps a block index on all three events

Every text block start, text delta and text block end event MUST carry a block index stamped by the producer at emission time. The index MUST NOT be derived from position in a sequence, because at emission time no finished sequence exists to derive from. The index MUST be readable from an external package without parsing text.

#### Scenarios

- **S-ATE-004** — Given a producer emitting a start, two deltas and an end for one block, when a consumer in another package reads each of the four events, then each reports the same block index.
- **S-ATE-005** — Given a recorded stream, when a consumer reads a text event's block index, then the value it reads is the value the producer stamped, and it does not change if the events are re-read in a different order.

### R-ATE-003 — The block index is 1-based and 0 is a rejected sentinel

The first block of a stream MUST carry block index 1. A block index of 0 MUST be rejected at construction with AI-04's out-of-range sentinel at a position naming the index, so that an event whose index was never stamped is never indistinguishable from a legal block.

#### Scenarios

- **S-ATE-006** — Given a text block start, delta or end constructed with block index 0, when construction runs, then it fails, `errors.Is` reports the out-of-range sentinel, and the position renders the block-index field.
- **S-ATE-007** — Given a stream's first text block, when its events are read, then their block index is 1, not 0.
- **S-ATE-008** — Given each of the three kinds in turn, when it is constructed with a zero-value payload, then construction fails rather than yielding an event carrying block index 0.

### R-ATE-004 — One block-index space per stream, shared across all content families

A stream MUST have exactly one block-index space. A text block's index MUST NOT collide with a reasoning block's or a tool-call block's index on the same stream; the index alone MUST be sufficient to attribute an event to its block, with no family tag required. This requirement is **binding on AI-17 and AI-18**, which MUST NOT introduce a per-family numbering space.

*Cross-check performed at spec time*: AI-17's proposal decision **D3** and AI-18's proposal decision **D1** independently reach the same shared, stream-wide conclusion and both flag it as coordination ground. One phrasing divergence is recorded rather than relitigated: AI-17's D3 illustrates the collision with "text block 0" / "reasoning block 0", which is 0-based wording. `R-ATE-003` is authoritative — the space is 1-based and 0 is the rejected sentinel. AI-17's spec must adopt that, and AI-18's call ordinal remains a distinct concept from the block index.

#### Scenarios

- **S-ATE-009** — Given a stream carrying two interleaved text blocks whose events arrive out of block order, when a consumer partitions the events by block index alone, then every event lands in its own block and no event is misattributed.
- **S-ATE-010** — Given a stream in which a text block and a non-text block are both open, when a consumer reads both blocks' indices, then the two indices differ, and the consumer separates them without consulting the event kind.
- **S-ATE-011** — Given the landed surface, when a reviewer looks for a per-family index space or a family tag required to disambiguate an index, then none exists.

### R-ATE-005 — A delta carries a fragment, never accumulated text

A text delta MUST carry only the bytes new since the previous delta of its block. It MUST NOT carry the block's accumulated content. Layer 1 MUST NOT expose a field, method or accessor on any of the three kinds that returns accumulated block content.

#### Scenarios

- **S-ATE-012** — Given a block whose deltas carry `"a"`, `"b"` and `"c"`, when a consumer reads each delta's fragment, then it reads exactly `"a"`, `"b"` and `"c"` — never `"a"`, `"ab"`, `"abc"`.
- **S-ATE-013** — Given the landed surface of the three kinds, when a reviewer enumerates their exported accessors, then none returns accumulated or snapshot content for the block.

---

## AI-16.2 — Byte fidelity across deltas

### R-ATE-006 — Concatenated deltas reconstruct the text byte-exactly

For any text block, the ordered concatenation of its delta fragments MUST equal the block's complete text byte-for-byte. Reconstruction MUST NOT require normalization, trimming, re-encoding, or any transformation by the consumer.

#### Scenarios

- **S-ATE-014** — Given a known text split into an arbitrary number of fragments, when a test-local concatenator joins the block's deltas in arrival order, then the result is byte-identical to the original text.
- **S-ATE-015** — Given a text containing leading whitespace, trailing whitespace and interior newlines, when its deltas are concatenated, then the result preserves every one of those bytes with no trimming.

### R-ATE-007 — A fragment is a byte fragment, not a string fragment

A delta fragment MUST be defined and documented as a sequence of **raw bytes**. An individual fragment MAY be invalid UTF-8 on its own. Construction MUST NOT reject a fragment for not being well-formed UTF-8, and MUST NOT repair, replace or re-encode any byte of it.

#### Scenarios

- **S-ATE-016** — Given a multi-byte rune split across a delta boundary, so that one fragment ends mid-rune and the next begins mid-rune, when both deltas are constructed, then both succeed and neither fragment's bytes are altered.
- **S-ATE-017** — Given that same pair of deltas, when their fragments are concatenated, then the result decodes to the original rune, and no replacement character appears anywhere in the reconstruction.
- **S-ATE-018** — Given the declaration of the fragment, when a reader looks for its encoding statement, then it uses the word **byte** and records that a single fragment may not be well-formed UTF-8 on its own.

### R-ATE-008 — No text-content emptiness rule applies to a fragment

A delta fragment MUST NOT be validated by the whitespace-emptiness rule that AI-06's complete text content applies. A whitespace-only fragment MUST be accepted, and a zero-length fragment MUST be accepted. Construction MUST NOT return `ErrEmpty` for either.

#### Scenarios

- **S-ATE-019** — Given a delta whose fragment is a single space, when it is constructed, then it succeeds and its fragment is exactly one space byte.
- **S-ATE-020** — Given a delta whose fragment is zero-length, when it is constructed, then it succeeds and no emptiness sentinel is reported.
- **S-ATE-021** — Given a block whose deltas include a whitespace-only fragment among non-empty ones, when they are concatenated, then the whitespace byte appears in the reconstruction at its original position.

### R-ATE-009 — A fragment is bounded by the existing text ceiling

A delta fragment MUST be rejected with AI-04's out-of-range sentinel when it exceeds `MaxTextLen` bytes, reusing that existing constant. No new bound is invented at Layer 1.

**Why this bound and not another.** `MaxTextLen`'s own rationale — make an unbounded value decidable from the request alone (`V-REQ-24`) — applies verbatim to a fragment, which would otherwise be the only unbounded caller-supplied byte carrier in Layer 1. The cap is unobservable to a correct producer, because a fragment of a legal text is never longer than the text and `NewText` already bounds that at `MaxTextLen`; it costs one length comparison; and it keeps a stream-to-`Part` materialization path from emitting fragments the `NewText` boundary would later reject. A new, smaller per-fragment constant was rejected: any smaller cap could reject a fragment that a legal complete text can contain.

**Recorded asymmetry.** Only the single fragment is bounded. The block's *reconstructed total* is not, because bounding it requires accumulation, which `R-ATE-011` forbids at Layer 1. Enforcement of a total, if ever wanted, belongs to AI-23.

#### Scenarios

- **S-ATE-022** — Given a delta whose fragment exceeds `MaxTextLen` bytes, when it is constructed, then it fails, `errors.Is` reports the out-of-range sentinel, and the position names the fragment.
- **S-ATE-023** — Given a fragment of exactly `MaxTextLen` bytes, when it is constructed, then it succeeds — the cap cannot reject any fragment a legal complete text could produce.

---

## AI-16.3 — Zero-delta blocks

### R-ATE-010 — A text block with no deltas is legal and reconstructs to empty

A stream carrying a text block start immediately followed by that block's text block end, with zero deltas between them, MUST be legal. Its reconstruction MUST be the empty byte sequence. Layer 1 MUST NOT require at least one delta per block, and MUST NOT treat a zero-delta block as a failure, a truncation, or a distinguishable error shape.

#### Scenarios

- **S-ATE-024** — Given a stream containing a text block start and its matching text block end with no delta between them, when the events are validated, then both are valid and no rule violation is reported.
- **S-ATE-025** — Given that block, when a test-local concatenator reconstructs it, then the result is empty and no error is produced by the empty join.
- **S-ATE-026** — Given a stream containing one zero-delta block and one multi-delta block, when a consumer processes both by index alone, then both close normally and the zero-delta block is not confused with an unterminated block.

### R-ATE-011 — Layer 1 ships no public accumulation or reconstruction helper

Layer 1 MUST NOT export an accumulator, transcript rebuilder, or any function that reduces a block's deltas to its complete text. Doc 0001 § 4.3 invariant 1 reserves accumulation for the consumer. Byte-exactness MUST be proven by this milestone's own tests using a test-local concatenator.

#### Scenarios

- **S-ATE-027** — Given the landed package surface, when a reviewer enumerates its exported identifiers, then none accumulates, joins or reconstructs a block's deltas.
- **S-ATE-028** — Given the tests that prove `R-ATE-006` and `R-ATE-010`, when a reader looks for the concatenator they use, then it is defined inside the test package and is not exported from the contract.

---

## Non-functional requirements

### NFR-ATE-A — Dependency purity

The change MUST add no module dependency. `backend/agent/go.mod` MUST still carry zero requires, and both AI-00 import guards MUST still pass.

- **S-ATE-029** — Given the change merged, when `go.mod` is read, then it declares no require; and when both import guards run, then both pass.

### NFR-ATE-B — Totality

No exported function or method of this contract MUST panic for any input, including the zero value of each payload, a zero block index, a zero-length fragment, an invalid-UTF-8 fragment, and an over-long fragment.

- **S-ATE-030** — Given a table of those extreme inputs, when each is passed through every exported entry point of the three kinds, then none panics.

### NFR-ATE-C — Failure reporting

Every rule violation in this spec MUST be reported through AI-04's existing failure value and its landed sentinels. No new sentinel and no second failure type MUST be introduced.

- **S-ATE-031** — Given each rejecting scenario above, when its failure is inspected, then it is AI-04's failure value, `errors.Is` matches a landed sentinel, and its position names a field of the offending payload.

### NFR-ATE-D — Evidence

Every test-list item of AI-16.1, AI-16.2 and AI-16.3 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) in `backend/agent/` and clean `make lint`.

- **S-ATE-032** — Given `tasks.md`, when a reviewer walks the five test-list items, then each carries recorded red output, recorded green output, and a refactor note.

---

## Acceptance criteria

1. Three separately registered text event kinds, covered by AI-14's exhaustiveness guard, shown to bite against a scratch kind.
2. All three events carry a producer-stamped block index; 1-based, with 0 rejected at construction by AI-04's out-of-range sentinel.
3. Two interleaved text blocks are attributed by index alone, with no reliance on arrival order and no family tag.
4. The block-index space is one per stream and shared across families — binding on AI-17 and AI-18, cross-checked against their proposals.
5. Deltas carry fragments only; no accessor returns accumulated content.
6. Concatenated deltas reconstruct byte-exactly, including a boundary that splits a multi-byte rune.
7. Whitespace-only and zero-length fragments are legal; no `ErrEmpty` rule from complete text content applies.
8. A start/end pair with zero deltas is legal and reconstructs to empty.
9. No public accumulator ships; byte-exactness is proven with a test-local concatenator.
10. `make test` green under `-race`, `make lint` clean, both import guards passing, `go.mod` still zero requires.
