# Spec — response lifecycle events

> **Change**: `cachicamas-ai-response-events` · **Milestone**: AI-15 — Add response lifecycle events (Wave 2 "Stream")
> **Nodes**: AI-15.1 `[leaf]` response start · AI-15.2 `[leaf]` completion
> **Capability**: `ai-response-events` — **new**. Promoted to `openspec/specs/ai-response-events/spec.md` at archive.
> **Predecessor**: [`proposal.md`](../../proposal.md) · [`explore.md`](../../explore.md) · Engram `sdd/cachicamas-ai-response-events/proposal` (obs #2162)
> **Requirement IDs**: `R-ARP-0NN` · **Scenario IDs**: `S-ARP-0NN` — prefix re-verified unused across `openspec/specs/` and `openspec/changes/` at spec time (0 matches for `R-ARP-`/`S-ARP-`)
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../../../../specs/ai-contract-vocabulary/spec.md) — `V-STR-01` stream, `V-STR-10` event, `V-STR-11` event kind, `V-STR-12` payload, `V-STR-17` ordering invariant, `V-STR-18` terminal event, `V-STR-19` response-start event, `V-STR-20` completion event, `V-MET-*` finish reason / usage / absence-versus-zero, `V-REQ-21` model identity (request-side)
> **Binding predecessors**: [validation error taxonomy](../../../../specs/ai-validation-errors/spec.md) (AI-04) · [completion metadata](../../../../specs/ai-completion-metadata/spec.md) (AI-13)
> **Depends on**: AI-13 (merged), AI-14 (concurrent — envelope, kind registry, per-stream sequence, invariant checker). **Blocks**: AI-16 … AI-19
> **Sources**: doc 0002 §§ AI-15.1 … AI-15.2 (lines 911–935) · doc 0001 § 3.1 **C4**, § 4.3, § 7 **G10**

---

## Purpose

A stream has no way to say "the provider started answering" or "the provider finished normally". Without the first, a consumer cannot attribute a stream to a provider response and cannot learn which model actually served it. Without the second, a normal end is indistinguishable from a failed one — which is exactly the shape defect **C4** produced.

This spec constrains the two events that open and close a response, so that an **empty but successful** response — response start, then completion, no content between them — is legal, ordered, and distinguishable from every failure shape available before AI-19.

What this spec deliberately does not constrain: the envelope, sequence stamping, kind registry and invariant-checker mechanics themselves (AI-14); any content-family block event (AI-16 … AI-18); the terminal error event and failure taxonomy (AI-19); any new finish-reason value or usage field (AI-13, consumed unchanged); vendor mapping of response ids and served model names (AI-31).

## Delta status against existing specs

`openspec/specs/` was re-read at spec time. It contains **no event capability**: AI-14's `ai-event-envelope` change folder holds only `proposal.md` and `explore.md`, with no `specs/` directory, so AI-14's spec had not landed when this artifact was written. This spec is therefore a **full new capability spec**, written against AI-14's proposal — specifically its decisions **D1** (invariants read only `(kind, block index, descriptor)`, where a descriptor is `{blockRole, cardinality: any|at-most-one, terminal: bool}`) and **D2** ("for AI-15, read this: the checker generalizes" — `at-most-one` and `terminal` are reusable descriptor-driven primitives, requiring no AI-14 change).

**Reconciliation gate.** Every descriptor-level assumption below is stated as behavior, never as a Go signature. If AI-14's landed spec or code diverges from D1/D2 — in particular if it ships block-triple-only invariants without the generic `at-most-one` and `terminal` primitives — `design.md` MUST reconcile before `sdd-apply`, by **generalizing AI-14's checker**, never by special-casing AI-15's two kinds inside it. That reconciliation is a spec-level change to `ai-event-envelope` and MUST be raised there, not silently absorbed here.

One `MODIFIED` capability is carried: `ai-contract-vocabulary`, as an append-only register amendment (`R-ARP-011`).

## Definitions used by this spec

- **Response start** — `V-STR-19`, the event announcing that a provider has begun responding.
- **Completion** — `V-STR-20`, the terminal event of a stream that finished normally.
- **Provider response identity** — the provider's own opaque handle for the response this stream carries. Provider-supplied, never Layer-1-minted. Register noun appended by `R-ARP-011`.
- **Served model** — the model that actually produced the response, as reported by the provider. Distinct from `V-REQ-21` model identity, which is what the request *asked for*. Register noun appended by `R-ARP-011`.
- **Absence** — AI-13's `TokenCount` property: an absent count is not the count zero.

---

## AI-15.1 — Response start

### R-ARP-001 — Response start is one separately registered event kind with its own payload

Layer 1 MUST expose response start as a single event kind registered independently in AI-14's kind registry, with its own constructible payload (`V-STR-12`). Its kind MUST be **derived from** its payload (`V-STR-11`), never stored as a field and never carried as a lifecycle discriminator shared with completion. Response start and completion MUST NOT be modelled as one payload type carrying a start/end discriminant, because that would make one type answer for two registered kinds and break AI-14.1's "every registered kind has a constructible payload" table.

#### Scenarios

- **S-ARP-001** — Given the landed registry, when a consumer in another package names the response-start kind, then it compiles, is a distinct value from every other registered kind, and is a member of the registry's closed kind set.
- **S-ARP-002** — Given the response-start payload, when a consumer constructs it and reads the resulting event's kind, then the kind is derived from the payload and matches, and no lifecycle-phase field is exposed on it.
- **S-ARP-003** — Given AI-14's exhaustiveness guard, when the suite runs, then the guard covers the response-start kind; and given a scratch unregistered kind, when the suite runs, then the guard fails and names it.

### R-ARP-002 — Response start carries provider response identity and served model, both externally readable

A response-start event MUST carry exactly two normalized fields: the **provider response identity** and the **served model**. Both MUST be readable from a package outside `ai` without a type switch over unexported types and without parsing any other field. Both MUST be plain byte-exact strings following the `ToolCall.ID()` / `Request.Model()` precedent — Layer 1 MUST NOT mint, parse, canonicalize, case-fold, trim or otherwise transform either value, and MUST NOT apply the `MessageID` minted-identity pattern to either.

#### Scenarios

- **S-ARP-004** — Given a response start constructed with a known response identity and a known served model, when a consumer in an external test package reads both, then it reads exactly the bytes supplied, with no type switch over unexported types.
- **S-ARP-005** — Given a response identity containing punctuation, mixed case and a vendor prefix (for example `msg_01AbC…`), when it is read back, then it is byte-identical to the input — not trimmed, not lower-cased, not re-formatted.
- **S-ARP-006** — Given the landed surface, when a reviewer enumerates the response-start payload's exported accessors, then exactly the two fields above are readable and no minting, parsing or catalog-lookup entry point exists.

### R-ARP-003 — The served model is a distinct concept from the requested model

The served model MUST be defined and documented as *the model that actually produced the response*, distinct from `V-REQ-21` model identity. Layer 1 MUST NOT assume, assert, or validate that the served model equals the requested model, and MUST NOT reject a response start whose served model differs from the model named by the originating request.

#### Scenarios

- **S-ARP-007** — Given a request naming model `A` and a response start reporting served model `B`, when the response start is constructed and validated, then it succeeds and reports `B` unchanged.
- **S-ARP-008** — Given the landed surface, when a reviewer looks for any comparison, cross-check or coupling between the response-start served model and a request's model identity, then none exists.
- **S-ARP-009** — Given the register, when a reader resolves "served model", then it resolves to the appended stream-side noun of `R-ARP-011`, not to `V-REQ-21`.

### R-ARP-004 — Both fields are required and non-empty

Construction of a response start MUST fail when either the provider response identity or the served model is empty, reporting AI-04's `ErrEmpty` sentinel through the existing failure value, at a position naming the offending field. The first failure only MUST be reported, via the package's `FirstFailure` idiom. Neither field is absent-capable: a provider that ships no response id obliges its **adapter** to synthesize one, and a provider that omits the served model obliges its adapter to fill it — an unfilled field yields an invalid event rather than a silently empty one.

*(This resolves the proposal's open questions 1 and 2 as assumed: non-empty for both, adapter's obligation.)*

#### Scenarios

- **S-ARP-010** — Given a response start constructed with an empty response identity, when construction runs, then it fails, `errors.Is` reports `ErrEmpty`, and the position renders the response-identity field.
- **S-ARP-011** — Given a response start constructed with an empty served model, when construction runs, then it fails, `errors.Is` reports `ErrEmpty`, and the position renders the served-model field.
- **S-ARP-012** — Given a response start constructed with both fields empty, when construction runs, then exactly one failure is reported and it names the first offending field.
- **S-ARP-013** — Given the zero value of the response-start payload, when it is validated, then it fails rather than yielding an event carrying two empty strings.

### R-ARP-005 — At most one response start per stream

A stream MUST carry **at most one** response-start event. A second response-start event on the same stream MUST violate AI-14.4's ordering invariants and MUST be detectable as such by a consumer running the checker against a recorded stream. This MUST be achieved by registering response start with AI-14's descriptor-driven `at-most-one` cardinality — Layer 1 MUST NOT special-case the response-start kind inside the invariant checker, because that would break the registry-driven extensibility the package has maintained since AI-06.4.

#### Scenarios

- **S-ARP-014** — Given a recorded stream containing one response start followed by a completion, when the invariant checker runs, then no violation is reported.
- **S-ARP-015** — Given a recorded stream containing two response-start events, when the invariant checker runs, then it reports a violation that names the response-start kind and the cardinality rule, and the violation is programmatically detectable rather than prose-only.
- **S-ARP-016** — Given the registration of the response-start kind, when a reviewer reads it, then its cardinality is the generic `at-most-one` descriptor value, and the invariant checker's source contains no branch naming the response-start kind.

---

## AI-15.2 — Completion

### R-ARP-006 — Completion is one separately registered event kind with its own payload

Layer 1 MUST expose completion as a single event kind registered independently in AI-14's kind registry, with its own constructible payload whose kind is derived, not stored — the same constraint `R-ARP-001` places on response start.

#### Scenarios

- **S-ARP-017** — Given the landed registry, when a consumer in another package names the completion kind, then it compiles, is distinct from the response-start kind and from every other registered kind, and is a member of the closed kind set.
- **S-ARP-018** — Given AI-14's exhaustiveness guard, when the suite runs, then it covers the completion kind alongside the response-start kind.

### R-ARP-007 — Completion embeds AI-13's finish reason and usage unchanged

A completion event MUST carry exactly one finish reason and exactly one usage record, both AI-13's existing types embedded **as-is**. This change MUST NOT introduce a new finish-reason value, a new usage field, a parallel finish-reason or usage type, or a re-encoding of either at the event boundary. Both MUST be readable from an external package. Completion construction MUST reject an invalid finish reason by delegating to AI-13's existing `Validate`, reporting `ErrNotInVocabulary`, and MUST reject a negative token count by delegating to `Usage.Validate`, reporting `ErrOutOfRange`.

#### Scenarios

- **S-ARP-019** — Given a completion constructed with a valid finish reason and a usage record, when a consumer in an external package reads both, then it reads the same values supplied, and their types are the ones AI-13 shipped.
- **S-ARP-020** — Given a completion constructed with the zero-value (invalid) finish reason, when construction runs, then it fails and `errors.Is` reports `ErrNotInVocabulary` at a position naming the finish-reason field.
- **S-ARP-021** — Given a completion constructed with a negative token count in its usage, when construction runs, then it fails and `errors.Is` reports `ErrOutOfRange` at a position naming the offending count.
- **S-ARP-022** — Given the landed surface, when a reviewer enumerates this change's declarations, then no new finish-reason member, no new usage field and no second finish-reason or usage type is declared.

### R-ARP-008 — Absence survives the event boundary

AI-13.3's absence-versus-zero property MUST be preserved across the event boundary. A usage record whose token counts are absent MUST still report them as absent after being carried by a completion event, and MUST NOT be normalized to zero, defaulted, or made present. Completion MUST NOT require any token count to be present — requiring a populated count is requiring a fabricated one.

#### Scenarios

- **S-ARP-023** — Given a usage record with every token count absent, when it is carried by a completion and read back from an external package, then every count still reports absent, and none reports the value zero as present.
- **S-ARP-024** — Given a usage record mixing an explicit `Tokens(0)` input count with an absent output count, when it is carried by a completion and read back, then the input count reports present with value 0 and the output count reports absent — the two remain distinguishable.
- **S-ARP-025** — Given a completion whose usage is entirely absent, when it is validated, then validation succeeds — presence is never required.

### R-ARP-009 — Completion is terminal

Completion MUST be registered as a terminal event (`V-STR-18`) using AI-14's generic descriptor-driven `terminal` property. A stream MUST carry at most one completion, and **no event of any kind** may follow it. An event emitted after a completion MUST violate AI-14.4's ordering invariants and be detectable as such. As with `R-ARP-005`, the invariant checker MUST NOT name the completion kind: terminality is a descriptor value, so AI-19's terminal error registers the same way with no further checker change.

#### Scenarios

- **S-ARP-026** — Given the registration of the completion kind, when a reviewer reads its descriptor, then its terminal property is set, and the invariant checker's source contains no branch naming the completion kind.
- **S-ARP-027** — Given a recorded stream ending in a completion, when the invariant checker runs, then no violation is reported and the completion is identified as the stream's terminal event.
- **S-ARP-028** — Given a recorded stream containing an event of any kind after a completion, when the invariant checker runs, then it reports a violation naming the terminality rule and the offending following event.
- **S-ARP-029** — Given a recorded stream containing two completion events, when the invariant checker runs, then it reports a violation.

### R-ARP-010 — An empty response is legal and distinguishable from every failure

A stream consisting of exactly a response start followed by a completion, with no content event between them, MUST be legal: the invariant checker MUST report no violation, and Layer 1 MUST NOT treat the absence of content as a failure, a truncation, or an incomplete stream. Such a stream MUST be distinguishable from every failure shape available at this milestone: it terminates with a completion event rather than with no terminal at all, and its finish reason is a legal member of AI-13's closed vocabulary. Distinguishing it MUST NOT require inspecting content, counting events, or consulting usage.

#### Scenarios

- **S-ARP-030** — Given a recorded stream of exactly a response start then a completion, when the invariant checker runs, then it reports no violation and reports the stream as normally terminated.
- **S-ARP-031** — Given that same stream, when a consumer asks whether it succeeded, then it answers from the presence and kind of the terminal event alone, without reading content or usage.
- **S-ARP-032** — Given a stream that ends without any terminal event, when it is compared to the empty-but-complete stream, then the two are distinguishable, and the truncated one is reported as lacking a terminal.
- **S-ARP-033** — Given a completion whose finish reason is `FinishReasonRefusal` and one whose finish reason is a normal stop, when both are read, then both are legal completions and the two finish reasons remain three-way distinguishable from `FinishReasonUnknown`, exactly as AI-13 shipped them.

---

## Register amendment — `ai-contract-vocabulary` (MODIFIED capability)

### R-ARP-011 — Two stream-side nouns are appended to the register in this pull request

**The gap is confirmed still open at spec time.** `V-STR-19` names the response-start event but defines neither of its two fields; `V-STR-20` names the completion event; `V-REQ-21` **model identity** is explicitly request-side ("the neutral name of the model a request targets") and does not cover the model that actually served the response. No stream-side row defines *provider response identity* or *served model*. The highest stream-side ordinal in the live register is `V-STR-23`.

This change MUST append exactly two rows to `openspec/specs/ai-contract-vocabulary/spec.md` § 4.2 (content terms), owned by **AI-15**, in this same pull request, per register § 9 rules 2 and 6:

| Id | Term | Sense to be recorded |
| --- | --- | --- |
| `V-STR-24` | **provider response identity** | The provider's own opaque handle for the response a stream carries. Provider-supplied and byte-exact — never minted, parsed or canonicalized by Layer 1. Distinct from `V-STR-01` stream, which is the delivery in progress, and from any Layer-1-minted identity. |
| `V-STR-25` | **served model** | The model that actually produced the response, as reported by the provider. Distinct from `V-REQ-21` **model identity**, which is what the request asked for; the two may legitimately differ, and conflating them is how a routed or aliased model becomes invisible. |

The amendment MUST follow every register amendment rule: append-only ordinals (`V-STR-24`, `V-STR-25` are the next free stream-side values), a dated amendment blockquote under § 4 stating what was appended, by which milestone node, and *why the register lacked the term*; no existing row renumbered, reworded, reordered or removed; and the per-category and total counts in § 10 updated from 116 to 118 with 23 stream-side becoming 25.

#### Scenarios

- **S-ARP-034** — Given the merged register, when a reader resolves `V-STR-24` and `V-STR-25`, then both exist under § 4.2, both name AI-15 as owner, and both carry the senses above.
- **S-ARP-035** — Given the merged register, when a reviewer diffs § 3 through § 8 against the pre-amendment file, then the only changes are the two appended rows, the § 4 amendment blockquote and the § 10 counts — no existing row is renumbered, reworded, reordered or removed.
- **S-ARP-036** — Given § 10, when a reader reads the term count, then it reads 25 stream-side and **118 terms** total, and the amendment note names AI-15.
- **S-ARP-037** — Given this change's spec, design and tasks, when a reviewer looks for a locally-defined meaning of "response identity" or "served model", then every occurrence cites `V-STR-24` or `V-STR-25` rather than re-paraphrasing.

---

## Non-functional requirements

### NFR-ARP-A — Dependency purity

The change MUST add no module dependency. `backend/agent/go.mod` MUST still carry zero requires, and both AI-00 import guards MUST still pass.

- **S-ARP-038** — Given the change merged, when `go.mod` is read, then it declares no require; and when both import guards run, then both pass.

### NFR-ARP-B — Totality

No exported function or method of this contract MUST panic for any input, including the zero value of each of the two payloads, empty strings in either response-start field, an invalid finish reason, a fully absent usage record, and a usage record with a negative count.

- **S-ARP-039** — Given a table of those extreme inputs, when each is passed through every exported entry point of the two kinds, then none panics.

### NFR-ARP-C — Failure reporting

Every rule violation in this spec MUST be reported through AI-04's existing failure value and its landed sentinels (`ErrEmpty`, `ErrNotInVocabulary`, `ErrOutOfRange`). No new sentinel and no second failure type MUST be introduced.

- **S-ARP-040** — Given each rejecting scenario above, when its failure is inspected, then it is AI-04's failure value, `errors.Is` matches a landed sentinel, and its position names a field of the offending payload.

### NFR-ARP-D — Evidence

Every test-list item of AI-15.1 (two) and AI-15.2 (three) MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) in `backend/agent/` and clean lint.

- **S-ARP-041** — Given `tasks.md`, when a reviewer walks the five test-list items, then each carries recorded red output, recorded green output, and a refactor note.

---

## Acceptance criteria

1. Response start and completion are two separately registered event kinds with derived kinds and independent payloads; neither is a discriminated variant of the other.
2. Response start carries provider response identity and served model, both byte-exact, both readable from an external package with no type switch over unexported types.
3. Served model is distinct from requested model, with a passing case where the two differ.
4. Both response-start fields are required and non-empty, rejected with `ErrEmpty` at a position naming the field.
5. At most one response start per stream, enforced by AI-14's generic `at-most-one` descriptor, with no kind-specific branch in the checker.
6. Completion embeds AI-13's `FinishReason` and `Usage` unchanged, with no new value, field or parallel type.
7. Absent token counts remain absent across the event boundary, and `Tokens(0)` present remains distinguishable from absent.
8. Completion is terminal via AI-14's generic `terminal` descriptor; a second completion and any following event are both rejected.
9. A response-start-then-completion stream is legal and distinguishable from a stream with no terminal, without inspecting content or usage.
10. `V-STR-24` and `V-STR-25` appended to the register in this pull request, with the dated blockquote and § 10 counts updated to 118.
11. `make test` green under `-race`, lint clean, both import guards passing, `go.mod` still zero requires.

## Open items carried into design

1. **AI-14 reconciliation (blocking).** AI-14's spec had not landed at spec time. If its checker ships without the generic `at-most-one` and `terminal` descriptor primitives its proposal's **D2** promises, `design.md` MUST generalize AI-14's checker — never special-case AI-15's kinds — and MUST raise that as a delta on `ai-event-envelope`. `sdd-apply` stays hard-gated behind AI-14's merge.
2. **Payload naming and constructor shape** are design's, not spec's. This artifact deliberately pins no Go signature, per doc 0002's authoring constraint.
3. **Proposal questions 3 and 4** are resolved here: generalize AI-14's checker in design (item 1), and land the register amendment in this PR (`R-ARP-011`). Questions 1 and 2 are resolved by `R-ARP-004` as assumed.
4. **AI-16's `R-ATE-003`** establishes a 1-based, stream-wide block-index space. Neither AI-15 event carries a block index, so no interaction exists; design MUST confirm that response start and completion register with AI-14's `blockRole: none`.

> **Deviation note**: this artifact exceeds the sdd-spec 650-word budget. The landed capability-spec precedent in this repository (`openspec/specs/ai-completion-metadata/spec.md`, `openspec/specs/ai-cache-breakpoints/spec.md`) and the sibling AI-16 spec carry per-requirement scenario blocks, non-functional requirements and acceptance criteria at this density; house convention wins.
