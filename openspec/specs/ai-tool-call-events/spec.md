# Spec — streamed tool-call events

> **Milestone**: AI-18 — Add tool-call delta events (Wave 2 "Stream") · **Nodes**: AI-18.1 `[leaf]` call lifecycle, AI-18.2 `[leaf]` delta optionality, AI-18.3 `[leaf]` interleaving and ordinal
> **Introduced by**: `openspec/changes/archive/2026-08-01-cachicamas-ai-tool-call-events/`, merged to `main` in PR #104 (commit `37898c7`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: the Layer 1 half of **G5** — the tool-call ordinal survives normalization on the stream side (AI-18.3), completing what AI-09.2 began on the request side; wire proof is AI-30.5. It also closes **G12(a)**, delta-optional tool calls (`R-ATC-009`, `R-ATC-010`), exercised later by AI-21.2 and AI-30.2 and pinned by suite case AI-23.3
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-ATC-0NN` · **Scenario IDs**: `S-ATC-0NN`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md) — `V-STR-10` event, `V-STR-11` event kind, `V-STR-12` payload, `V-STR-14` block, `V-STR-15` block index, `V-STR-16` delta, `V-STR-17` ordering invariant, `V-STR-21` call ordinal, `V-FAIL-13` redaction
> **Binding predecessor**: [tool messages](../ai-tool-messages/spec.md) (AI-09) — `R-ATM-002`/`R-ATM-003` byte exactness, `R-ATM-005` one rule set, `R-ATM-006` derived ordinal; all restated here, never re-defined
> **Binding predecessor**: [the event envelope](../ai-event-envelope/spec.md) (AI-14) — the kind registry, the `BlockIndex` type and the descriptor-driven ordering checker these three kinds register into
> **Binding predecessor**: [indexed text block events](../ai-text-events/spec.md) (AI-16) — `R-ATE-003`/`R-ATE-004` fix the shared block-index space as **1-based with 0 as a rejected sentinel**. This spec adopts that convention verbatim; it does not restate a competing base
> **Binding predecessor**: [the validation error taxonomy](../ai-validation-errors/spec.md) (AI-04) — every rule violation reports through AI-04's failure value and its landed sentinels
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-18--add-tool-call-delta-events) §§ AI-18.1 … AI-18.3 · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) § 3.3 **G12(a)**, § 4.3 invariant 1 · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

---

## Purpose

Layer 1 can describe a *finished* tool call (AI-09 `ToolCall`). It cannot describe one **arriving**. This spec constrains the three events a streamed tool call produces — call-start, call-delta, call-end — so that a consumer sees a tool name before any argument byte, attributes every argument fragment to its own call while calls interleave, and reconstructs the exact argument bytes the model produced.

Two things this spec is careful *not* to constrain: whether the reconstructed bytes are well-formed JSON (**D2** — deferred to AI-30, the single validation entry point), and how a consumer accumulates fragments (doc 0001 § 4.3 invariant 1 — accumulation is the consumer's, and Layer 1 ships no helper).

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The three tool-call kinds, the name-before-arguments rule, the byte-exactness guarantee, the delta-optionality rule and the derived call ordinal therefore live here, in their own text, and not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-01-cachicamas-ai-tool-call-events/`](../../changes/archive/2026-08-01-cachicamas-ai-tool-call-events/) is the historical record of how AI-18 was explored, proposed, designed, applied and verified.

`R-ATC-007` and `R-ATC-012` are the two requirements most likely to be eroded by a well-meaning later change. `R-ATC-007` keeps JSON well-formedness out of the event path entirely: [`ai-tool-messages`](../ai-tool-messages/spec.md)'s `R-ATM-005` promises "one rule set, two entry points", and the event boundary must not become a third. `R-ATC-012` keeps the call ordinal derived rather than stored — no payload carries it, and no accessor reads like a call-scoped counter. Both are guarded by assertions over the landed surface, not by convention.

## Definitions used by this spec

- **A call block** — `V-STR-14` narrowed to a tool call: one streamed call, delimited by a call-start event and a call-end event, with zero or more call-deltas between them.
- **The block index** — `V-STR-15`. The shared, stream-wide value attributing an event to its own block (**D1**).
- **A fragment** — the *new* argument bytes a delta carries; `V-STR-16`'s "only the new fragment", never a snapshot.
- **The call ordinal** — `V-STR-21`. The observable position of a call among the calls of one response. A **distinct concept** from the block index, and derived, never stored.
- **Reconstruction** — ordered concatenation of one call block's delta fragments, producing the call's complete argument bytes.

---

## AI-18.1 — Call lifecycle

### R-ATC-001 — The family is three separately registered event kinds

Layer 1 MUST expose exactly three tool-call event kinds — call-start, call-delta, call-end — each registered independently in AI-14's event-kind registry, each with its own constructible payload (`V-STR-12`). The family MUST NOT be modelled as one kind carrying an internal phase discriminator, and MUST NOT be folded into the text or reasoning families by a family tag.

#### Scenarios

- **S-ATC-001** — Given the landed registry, when a consumer in another package names each of the three kinds, then each compiles, all three are distinct values, and each is a member of the registry's closed kind set.
- **S-ATC-002** — Given each of the three kinds, when a consumer constructs its payload and reads the resulting event's kind, then the kind is derived from the payload, matches, and no phase field or family tag is exposed on any of the three.
- **S-ATC-003** — Given AI-14's exhaustiveness guard, when the suite runs, then the guard covers all three new kinds; and given a scratch unregistered kind, when the suite runs, then the guard fails and names it.

### R-ATC-002 — Call-start carries identity and tool name, validated non-empty at construction

A call-start event MUST carry the call's identity and its tool name, both readable from an external package. Construction MUST reject an empty identity and MUST reject an empty tool name with AI-04's `ErrEmpty` sentinel at a position naming the offending field (**D3**, by direct analogy to `NewToolCall`'s first two rules). The tool name MUST be available from the call-start event alone, before any argument byte has been streamed.

#### Scenarios

- **S-ATC-004** — Given a call-start constructed with a non-empty identity and tool name, when a consumer in another package reads both, then it reads exactly the supplied values, and it has read the tool name without having observed any delta or the call-end.
- **S-ATC-005** — Given a call-start constructed with an empty identity, when construction runs, then it fails, `errors.Is` reports `ErrEmpty`, and the position renders the identity field.
- **S-ATC-006** — Given a call-start constructed with an empty tool name, when construction runs, then it fails, `errors.Is` reports `ErrEmpty`, and the position renders the name field.
- **S-ATC-007** — Given the zero value of the call-start payload, when it is validated, then it fails rather than yielding an event carrying an empty identity or name.

### R-ATC-003 — All three events carry a shared, stream-wide block index

Every call-start, call-delta and call-end event MUST carry a block index stamped by the producer at emission time (**D1**). That index MUST live in the **one block-index space per stream** that `R-ATE-004` fixes; a call block's index MUST NOT collide with a text or reasoning block's index on the same stream, and the index alone MUST be sufficient to attribute an event to its block with no family tag. Layer 1 MUST NOT introduce a call-scoped counter as a second index space.

#### Scenarios

- **S-ATC-008** — Given a producer emitting a call-start, two deltas and a call-end for one call, when a consumer in another package reads each of the four events, then each reports the same block index.
- **S-ATC-009** — Given a stream in which a call block and a text block are both open, when a consumer reads both blocks' indices, then the two indices differ, and the consumer separates them without consulting the event kind.
- **S-ATC-010** — Given the landed surface of the three kinds, when a reviewer looks for a call-scoped index, a per-family numbering space, or a family tag required to disambiguate an index, then none exists.

### R-ATC-004 — The block index is 1-based and 0 is a rejected sentinel

The first block of a stream MUST carry block index 1. A block index of 0 MUST be rejected at construction with AI-04's out-of-range sentinel at a position naming the index field, so that an event whose index was never stamped is never indistinguishable from a legal block. This is `R-ATE-003` adopted unchanged; this spec MUST NOT define a different base or a different sentinel value.

#### Scenarios

- **S-ATC-011** — Given a call-start, call-delta or call-end constructed with block index 0, when construction runs, then it fails, `errors.Is` reports the out-of-range sentinel, and the position renders the block-index field.
- **S-ATC-012** — Given a stream whose first block is a call block, when its events are read, then their block index is 1, not 0.

### R-ATC-005 — A delta carries an index and only the new argument fragment

A call-delta MUST carry the block index of its call and only the argument bytes new since that call's previous delta. It MUST NOT carry accumulated argument content, and Layer 1 MUST NOT expose any field, method or accessor on the three kinds that returns accumulated call arguments. A fragment MUST be treated as raw bytes: a zero-length, whitespace-only, or individually invalid-UTF-8 fragment MUST be accepted, unaltered.

#### Scenarios

- **S-ATC-013** — Given a call whose deltas carry `{"a"`, `:1`, `}`, when a consumer reads each delta's fragment, then it reads exactly those three fragments — never a growing prefix chain.
- **S-ATC-014** — Given a delta whose fragment is zero-length, and one whose fragment is a single space, when each is constructed, then both succeed, no emptiness sentinel is reported, and each fragment's bytes are unaltered.
- **S-ATC-015** — Given the landed surface of the three kinds, when a reviewer enumerates their exported accessors, then none returns accumulated or snapshot argument content.

### R-ATC-006 — Call-end carries the exact argument bytes, byte-equal and never re-marshalled

A call-end event MUST carry the call's complete argument bytes such that they are byte-for-byte equal to the ordered concatenation of that call's delta fragments. No component on the event path MAY re-marshal, canonicalize, re-order keys, re-indent, trim, or re-encode those bytes. Every accessor returning argument bytes or a fragment MUST return a fresh copy, so that a caller mutating the returned bytes cannot affect the event, and two successive reads MUST return equal values.

#### Scenarios

- **S-ATC-016** — Given a known argument byte sequence split into an arbitrary number of fragments, when a test-local concatenator joins the call's deltas in arrival order, then the result is byte-identical to the call-end's argument bytes.
- **S-ATC-017** — Given argument bytes containing redundant whitespace, a non-canonical key order and interior newlines, when the call-end is constructed and read back, then every one of those bytes is preserved with no normalization.
- **S-ATC-018** — Given a constructed call-end, when a caller mutates the byte slice returned by its argument accessor and reads the accessor again, then the second read is unaffected and equals the original bytes.

### R-ATC-007 — Call-end MUST NOT validate argument well-formedness

Construction of a call-end MUST NOT parse, validate, or reject its argument bytes for JSON well-formedness, and MUST NOT report `ErrMalformed` for them (**D2**). AI-09's rule set runs exactly once, at AI-30's reassembly — `R-ATM-005`'s "one rule set, two entry points" MUST NOT gain a third entry point at the event boundary. Layer 1 MUST NOT canonicalize empty arguments to `{}` on the event path; that canonicalization is `NewToolCall`'s, at reassembly.

#### Scenarios

- **S-ATC-019** — Given a call-end constructed with argument bytes that are not well-formed JSON, when construction runs, then it succeeds and its bytes are readable back byte-equal.
- **S-ATC-020** — Given a call-end constructed with zero-length argument bytes, when its arguments are read, then they are zero-length and have not been replaced by `{}`.
- **S-ATC-021** — Given the landed event path, when a reviewer traces it, then no call to AI-09's well-formedness check appears on it, and the deferral to AI-30 is stated in the contract text.

### R-ATC-008 — Identity, name and argument bytes are redacted from rendering

The `String()` and `GoString()` renderings of all three payloads MUST NOT reveal the call identity, the tool name, an argument fragment, or the argument bytes (`V-FAIL-13`), matching `ToolCall`'s shipped behavior.

#### Scenarios

- **S-ATC-022** — Given each of the three payloads populated with a distinctive identity, name and fragment/arguments, when `String()` and `GoString()` are rendered, then neither rendering contains any of those values.

---

## AI-18.2 — Delta optionality

### R-ATC-009 — A call with zero deltas is legal and complete

A stream carrying a call-start immediately followed by that call's call-end, with zero deltas between them, MUST be legal and MUST be treated as a complete call (**G12(a)**). Layer 1 MUST NOT require at least one delta per call, and MUST NOT treat a zero-delta call as a failure, a truncation, or an unterminated block. No consumer contract in this spec MAY require a delta to have been observed.

#### Scenarios

- **S-ATC-023** — Given a stream containing a call-start and its matching call-end with no delta between them, when the events are validated, then both are valid and no rule violation is reported.
- **S-ATC-024** — Given a stream containing one zero-delta call and one multi-delta call, when a consumer processes both by block index alone, then both close normally and the zero-delta call is not confused with an unterminated block.
- **S-ATC-025** — Given the contract text, when a reader looks for a statement of delta optionality, then it states explicitly that no consumer may require a delta.

### R-ATC-010 — Whole and fragmented deliveries are indistinguishable after reconstruction

For the same tool call, a delivery whose arguments arrive entirely on the call-end with zero deltas and a delivery whose identical arguments arrive across any number of deltas MUST produce the same reconstruction: the same identity, the same tool name, and byte-equal argument bytes. Layer 1 MUST NOT expose delta count, fragment boundaries, or any other signal that lets a consumer distinguish the two after reconstruction.

#### Scenarios

- **S-ATC-026** — Given one argument byte sequence delivered twice — once as a zero-delta call, once split across five deltas — when both calls are reconstructed, then the two reconstructions are equal in identity, name and argument bytes.
- **S-ATC-027** — Given those two reconstructions, when a consumer inspects the landed surface for a delta count, a fragment-boundary list, or a "was fragmented" indicator, then none exists.
- **S-ATC-028** — Given a call whose arguments are split at a boundary that divides a multi-byte rune, when its deltas are concatenated, then the reconstruction decodes to the original rune and no replacement character appears.

---

## AI-18.3 — Interleaving and the call ordinal

### R-ATC-011 — Interleaved calls reconstruct independently

A stream MAY interleave the events of two or more open call blocks in any order permitted by `V-STR-17`. A consumer MUST be able to partition every call event to its own call using the block index alone, with no reliance on arrival adjacency and no family tag. Reconstruction of each call MUST be free of cross-contamination: no fragment of one call MAY appear in another call's reconstruction.

#### Scenarios

- **S-ATC-029** — Given two calls whose start, delta and end events are emitted interleaved, when a consumer partitions the events by block index alone, then every event lands in its own call and no event is misattributed.
- **S-ATC-030** — Given that interleaved stream, when both calls are reconstructed, then each call's identity, name and argument bytes match exactly what its own producer emitted, and neither contains a byte of the other.
- **S-ATC-031** — Given the interleaving test, when the suite runs under `-race`, then it passes with no data race reported.

### R-ATC-012 — The call ordinal is observable from the events and stored nowhere

Each call's ordinal (`V-STR-21`) MUST be observable from the stream's events, obtained by filtering the reconstructed block sequence to call-kind blocks and re-counting — the stream-side continuation of `ToolCalls()`'s derivation. The ordinal MUST NOT be stored as a field on any of the three payloads, restating `R-ATM-006` rather than re-defining it. The ordinal MUST remain a distinct concept from the block index: a call's block index MAY differ from its ordinal whenever text or reasoning blocks share the stream.

#### Scenarios

- **S-ATC-032** — Given a stream containing a text block, then a call block, then a second call block, when a consumer filters to call-kind blocks and counts, then the two calls yield ordinals in emission order and each ordinal differs from that call's block index.
- **S-ATC-033** — Given the landed surface of the three payloads, when a reviewer enumerates their fields and exported accessors, then no ordinal, index-among-calls, or call-sequence-number field exists on any of them.
- **S-ATC-034** — Given two calls whose events interleave, when their ordinals are derived, then the ordinals follow the order of the calls' start events and are stable across repeated derivation from the same recorded stream.

---

## Non-functional requirements

### NFR-ATC-A — Dependency purity

The change MUST add no module dependency. `backend/agent/go.mod` MUST still carry zero requires, and both AI-00 import guards MUST still pass.

- **S-ATC-035** — Given the change merged, when `go.mod` is read, then it declares no require; and when both import guards run, then both pass.

### NFR-ATC-B — Totality

No exported function or method of this contract MUST panic for any input, including each payload's zero value, block index 0, an empty identity or name, a zero-length fragment, an invalid-UTF-8 fragment, and malformed argument bytes.

- **S-ATC-036** — Given a table of those extreme inputs, when each is passed through every exported entry point of the three kinds, then none panics.

### NFR-ATC-C — Failure reporting

Every rule violation in this spec MUST be reported through AI-04's existing failure value and its landed sentinels (`ErrEmpty`, the out-of-range sentinel). No new sentinel and no second failure type MUST be introduced.

- **S-ATC-037** — Given each rejecting scenario above, when its failure is inspected, then it is AI-04's failure value, `errors.Is` matches a landed sentinel, and its position names a field of the offending payload.

### NFR-ATC-D — Evidence

Every test-list item of AI-18.1, AI-18.2 and AI-18.3 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) in `backend/agent/` and clean `make lint`.

- **S-ATC-038** — Given `tasks.md`, when a reviewer walks the test-list items, then each carries recorded red output, recorded green output, and a refactor note.

---

## Acceptance criteria

1. Three separately registered tool-call event kinds, covered by AI-14's exhaustiveness guard, shown to bite against a scratch kind.
2. Call-start exposes identity and tool name before any argument byte; empty identity or name is rejected at construction with `ErrEmpty`.
3. All three events carry the shared, stream-wide block index; 1-based with 0 rejected by AI-04's out-of-range sentinel, per `R-ATE-003`/`R-ATE-004`.
4. Deltas carry index plus fragment only; no accessor returns accumulated arguments.
5. Call-end arguments are byte-equal to the concatenated fragments, never re-marshalled, returned as fresh copies.
6. Call-end performs no JSON well-formedness validation and no empty-argument canonicalization; the deferral to AI-30 is stated in the contract text.
7. A zero-delta call is legal and complete, and is indistinguishable after reconstruction from its fragmented equivalent.
8. Two interleaved calls reconstruct independently by block index alone, with no cross-contamination, green under `-race`.
9. Each call's ordinal is derived from the event sequence, distinct from the block index, and stored on no payload.
10. `make test` green under `-race`, `make lint` clean, both import guards passing, `go.mod` still zero requires.

## Deliberately unbounded, and why

No size ceiling applies to an argument fragment or to a call-end's complete argument bytes. AI-09 ships no argument-length constant, and inventing one at Layer 1 would be a new bound rather than a reused one — the same reasoning `R-ATE-009` applies in the opposite direction, where an existing ceiling already fit. If a ceiling is ever wanted, it arrives by amendment here, with the constant it reuses named.
