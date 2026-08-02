# Spec — event envelope with per-stream sequencing

> **ARCHIVED DELTA — historical record.** The live contract is [`openspec/specs/ai-event-envelope/spec.md`](../../../../../specs/ai-event-envelope/spec.md), promoted from this file on 2026-08-01. This copy is preserved as the artifact AI-14 was verified against. Relative links below were re-resolved for the archive path; nothing else was changed.

> **Change**: `cachicamas-ai-event-envelope` · **Milestone**: AI-14 — Define the event envelope with per-stream sequencing (Wave 2 "Stream")
> **Nodes**: AI-14.1 `[leaf]` envelope skeleton · AI-14.2 `[leaf]` per-stream sequence · AI-14.3 `[guard]` no process-global sequence state · AI-14.4 `[leaf]` ordering invariants
> **Capability**: `ai-event-envelope` — **new**. Promoted to `openspec/specs/ai-event-envelope/spec.md` at archive.
> **Predecessor**: [`proposal.md`](../../proposal.md) · [`explore.md`](../../explore.md) · Engram `sdd/cachicamas-ai-event-envelope/proposal`
> **Requirement IDs**: `R-AEE-0NN` · **Scenario IDs**: `S-AEE-0NN` — prefix verified unused at spec time across `openspec/specs/` and `openspec/changes/` (in use and avoided: `AIV, AIC, AIS, AIE, ACP, ACB, ACM, AMR, AMSG, ATD, ATM, ARC, REX, AGM, ATE` (AI-16), `AIP` (AI-19), plus the non-AI `WS, WSY, HP, PR, DBMIG`)
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../../../../../specs/ai-contract-vocabulary/spec.md) § 4 — `V-STR-10` event, `V-STR-11` event kind, `V-STR-12` payload, `V-STR-13` sequence, `V-STR-14` block, `V-STR-15` block index, `V-STR-16` delta, `V-STR-17` ordering invariant, `V-STR-18` terminal event. All nine are **owned** by AI-14; this spec realizes them and MUST NOT re-define, re-paraphrase or narrow them.
> **Binding predecessors**: [stream lifecycle](../../../../../specs/ai-stream-lifecycle/spec.md) — § 4 single-producer ownership, § 10 the inherited/owned split · [validation error taxonomy](../../../../../specs/ai-validation-errors/spec.md) — every rejection here reports through AI-04's landed failure value and its **closed, append-only** sentinel set
> **Blocks**: AI-15 … AI-20. **Depends on**: AI-02, AI-04.
> **Sources**: doc 0002 §§ AI-14.1 … AI-14.4 (lines 855–909) · doc 0001 § 3.1 defects **C3** and **C4**, § 4.3, § 9

---

## Purpose

Layer 1 can describe what goes *to* a model. It has no type for anything coming *back*: `backend/agent/src/ai` holds no event, no sequence, no ordering rule, and Wave 2 (AI-15 … AI-20) cannot start without one.

This spec constrains three things and deliberately not a fourth. It constrains the **envelope** — a kind that cannot disagree with its payload — the **sequence** — 1-based and contiguous *per stream*, which is the retired design's defect **C3** made structurally unwritable — and the **ordering invariants**, stated as something a consumer runs rather than prose a reader nods at. It does not constrain any concrete event kind: AI-14 registers **zero** production kinds, and every requirement below is written so that AI-15 (response start, completion), AI-16 … AI-18 (content blocks) and AI-19 (terminal error) register their own kinds without amending this capability.

## Delta status against existing specs

`openspec/specs/` was re-read at spec time. It contains **no event capability**: this is a full new capability spec with **no MODIFIED, REMOVED or RENAMED requirement**. AI-02 (`ai-stream-lifecycle`) and AI-04 (`ai-validation-errors`) are cited as binding predecessors and are **not reopened**. The vocabulary register is cited by row identifier and is **not amended** — every `V-STR-1x` row this milestone owns already exists.

One coordination note is recorded rather than resolved here: AI-16's landed spec (`R-ATE-003`) fixes the block index as **1-based with 0 as the rejected sentinel**, and `R-AEE-014` below is written to be consistent with it.

## Definitions used by this spec

- **The envelope** — the `V-STR-10` event as a single value type: kind, payload, sequence.
- **A payload** — `V-STR-12`. The typed contents. There is no such thing as an event without one.
- **A descriptor** — the payload-independent ordering metadata a registered kind declares: `{blockRole: none|start|delta|end, cardinality: any|at-most-one, terminal: bool}`. Introduced by this spec as the mechanism that makes `V-STR-17` extensible.
- **A recorded stream** — a finite, ordered sequence of events captured from one stream, offered to the checker after the fact. Not a live channel.
- **The stamper** — the per-stream state that assigns `V-STR-13`.

---

## AI-14.1 — Envelope skeleton

### R-AEE-001 — The kind is derived from the payload and is never stored

An event's kind (`V-STR-11`) MUST be derived from the payload it carries, computed on read, and MUST NOT be stored in a field beside the payload. No exported field, constructor parameter, setter or literal form MUST allow a caller to state a kind independently of the payload, so a kind and its contents cannot disagree. This mirrors the landed `content_part.go` precedent for `V-REQ-05`.

#### Scenarios

- **S-AEE-001** — Given a constructed event, when a consumer in another package reads its kind, then the value reported is the kind the payload declares, and re-reading it yields the same value.
- **S-AEE-002** — Given the landed surface, when a reviewer enumerates every exported way to obtain an event, then none accepts a kind argument and none exposes a settable kind field.
- **S-AEE-003** — Given an event value, when a reviewer inspects its declaration, then it holds no kind field, and the kind accessor reads only the payload.

### R-AEE-002 — A payload-less event is rejected with `ErrNotInVocabulary`

An event carrying no payload MUST NOT be treated as valid. Validation MUST reject it with AI-04's landed `ErrNotInVocabulary` sentinel at a position naming the offending event, because a value that skipped construction has no payload, therefore no kind, therefore no membership of the closed kind vocabulary. **No new sentinel MUST be introduced** — AI-04's set is closed and append-only (`R-AIE-003`).

#### Scenarios

- **S-AEE-004** — Given the zero value of the event type obtained from another package, when it is validated, then it fails, `errors.Is` reports `ErrNotInVocabulary`, and the position renders the event.
- **S-AEE-005** — Given a payload whose type is not registered in the kind table, when an event carrying it is validated, then it fails with `ErrNotInVocabulary` rather than being passed through as an unknown kind.
- **S-AEE-006** — Given the change merged, when AI-04's sentinel set is enumerated, then it is unchanged — no sentinel was added, removed or restated.

### R-AEE-003 — The payload contract is sealed against outside implementation

The payload contract MUST be expressed so that no type declared outside `package ai` can satisfy it. Registering a kind MUST therefore be possible only inside this package. A consumer MUST NOT be able to fabricate an event that reports a registered kind while carrying contents this package never produced.

#### Scenarios

- **S-AEE-007** — Given a type declared in `package ai_test` that attempts to satisfy the payload contract, when the package is compiled, then compilation fails.
- **S-AEE-008** — Given the landed surface, when a reviewer enumerates the payload contract's members, then every member is unexported.
- **S-AEE-009** — Given an event obtained by any exported route from another package, when its payload is inspected, then it is either absent (the zero value) or a payload this package constructed — no third possibility exists.

### R-AEE-004 — Every registered kind has a constructible payload, asserted as a table

The kind registry MUST be covered by a mechanical exhaustiveness assertion, in the shape AI-06.4 established for content parts: the assertion enumerates the declared kind constants, cross-checks them against the registration table, and requires a witness leg per kind — a constructor reachable from another package, an accessor returning the typed payload, and a validation path. A kind declared without a complete witness MUST fail the assertion. This is the test that makes doc 0001's defect **C4** — a registered kind whose payload cannot be constructed — fail on the day it is reintroduced.

Because AI-14 registers zero production kinds (`R-AEE-006`), the table's subject at this milestone MUST be a **test-only witness payload**, registered for tests alone and bridged into `package ai_test` through `export_test.go`. The witness MUST NOT be reachable from a non-test build.

#### Scenarios

- **S-AEE-010** — Given the exhaustiveness assertion and the test-only witness kind, when the suite runs, then the assertion is non-vacuous: it reports at least one covered kind and every witness leg is exercised.
- **S-AEE-011** — Given a scratch kind constant declared without a registration-table entry, when the suite runs, then the assertion fails and names the missing kind; recorded, then dropped.
- **S-AEE-012** — Given a scratch kind with a table entry but no reachable constructor, when the suite runs, then the assertion fails and names the missing witness leg; recorded, then dropped.
- **S-AEE-013** — Given a non-test build of the package, when its exported surface is enumerated, then the witness payload and its kind are absent.

### R-AEE-005 — An event is readable from an external package without a type switch over unexported types

A consumer in another package MUST be able to read an event's kind, its sequence, and its typed payload. Obtaining the typed payload MUST NOT require a type switch or type assertion over an unexported type, and MUST NOT require reflection or text parsing.

#### Scenarios

- **S-AEE-014** — Given a constructed event, when a consumer in `package ai_test` reads its kind and sequence, then both compile and report the producer's values.
- **S-AEE-015** — Given that event, when the consumer obtains the typed payload for its kind, then it does so through an exported accessor that reports success, with no assertion over an unexported type anywhere in the call.
- **S-AEE-016** — Given an event whose kind differs from the accessor being called, when the accessor runs, then it reports failure rather than panicking and rather than returning a partially populated value.

### R-AEE-006 — AI-14 registers zero production event kinds

The production kind vocabulary landed by this milestone MUST be empty. Response start and completion (AI-15), text, reasoning and tool-call blocks (AI-16 … AI-18) and the terminal error (AI-19) MUST NOT be declared here, even as placeholders. The registry MUST be structured so that a later milestone adds a kind by appending — a constant, a table entry, a payload, a constructor, an accessor and a descriptor — **without editing any requirement of this capability**.

#### Scenarios

- **S-AEE-017** — Given a non-test build of the landed package, when its event-kind vocabulary is enumerated, then it is empty.
- **S-AEE-018** — Given the landed registry and descriptor table, when a reviewer walks the documented steps for adding a kind, then each step is an append to an existing list, and none requires changing existing kind entries.
- **S-AEE-019** — Given the landed surface, when a reviewer searches for a kind named for a response, a completion, a text, reasoning or tool block, or a terminal error, then none exists.

---

## AI-14.2 — Per-stream sequence

### R-AEE-007 — The sequence is 1-based and contiguous within one stream

For any stream emitting N events, the sequences carried by those events, in emission order, MUST be exactly 1…N: the first carries 1, each subsequent event carries exactly one more than its predecessor, with no gap and no repetition. The sequence MUST be assigned by the producer (`V-STR-13`), never by the consumer and never by the caller.

#### Scenarios

- **S-AEE-020** — Given a stamper for one stream, when it stamps N events in order, then their sequences read 1, 2, … N with no gap or repeat.
- **S-AEE-021** — Given the landed surface, when a reviewer looks for an exported way to supply a sequence value from outside, then none exists.
- **S-AEE-022** — Given a single stream's stamper, when it is used again after N events, then the next event carries N+1 — the stamper has no reset that a producer can reach.

### R-AEE-008 — Sequence state belongs to the stream, not the process

The stamper MUST be per-stream state with no state shared between streams. Two streams running concurrently MUST each begin at sequence 1 and MUST each be independently contiguous, and a stream started after other streams have completed MUST also begin at 1 — there is no residual process state to inherit. Because AI-02 § 4 guarantees exactly one producer goroutine per stream, the stamper MUST NOT require atomic or lock-based coordination for its own correctness; its race-freedom under concurrent streams MUST follow from the absence of shared state, not from synchronization.

#### Scenarios

- **S-AEE-023** — Given two streams stamped concurrently under `-race`, when both complete, then each stream's first event carries 1, each is independently contiguous, and the race detector reports nothing.
- **S-AEE-024** — Given two streams that have completed, when a third stream starts, then its first event carries 1.
- **S-AEE-025** — Given the landed stamper, when a reviewer inspects its declaration, then it holds no package-level state and uses no atomic or mutex for its own counter.

### R-AEE-009 — Cross-stream comparison is permitted and meaningless, and the contract says so

Comparing sequences across two different streams MUST be permitted by the type system and MUST NOT be given any meaning by this contract. The package documentation MUST state this explicitly, so a consumer that orders a merged multi-stream log by sequence is contradicting a written rule rather than an unwritten assumption.

#### Scenarios

- **S-AEE-026** — Given the landed package documentation, when a reader looks for the cross-stream rule, then it states that overlap between streams is expected and that cross-stream ordering carries no meaning.
- **S-AEE-027** — Given two independent streams, when their sequences are compared, then equal values occur and no contract rule is violated by their occurrence.

### R-AEE-010 — The never-stamped sequence has a documented sentinel value, rejected at a stated boundary

An event that was never stamped MUST carry a documented sentinel sequence value, and that value MUST be 0 — the zero value of the sequence, so that the unstamped state cannot be forged as a legal one. An event carrying the sentinel MUST be rejected with AI-04's `ErrOutOfRange` at a position naming the sequence. The **boundary at which the rejection happens MUST be named in the contract text**, not left to be inferred: it is the producer's emission boundary, the same boundary the stamper serves.

#### Scenarios

- **S-AEE-028** — Given a hand-constructed event that was never stamped, when it is offered at the producer's emission boundary, then it is rejected, `errors.Is` reports `ErrOutOfRange`, and the position renders the sequence.
- **S-AEE-029** — Given the landed documentation, when a reader looks for the sentinel, then it names the value 0 and names the boundary that rejects it.
- **S-AEE-030** — Given a stamped event carrying sequence 1, when it is offered at that same boundary, then it is accepted — the rejection rule cannot reject the first legal event of a stream.

---

## AI-14.3 — No process-global sequence state (guard)

### R-AEE-011 — A package-wide mechanical scan forbids package-level mutable sequence state

A guard MUST scan the **entire non-test source of `backend/agent/src/ai`** — every `.go` file that is not `_test.go`, whether or not it was added by this milestone — by parsing its AST, not by matching text. The guard MUST fail when it finds a package-level `var` whose type is, or recursively contains, an integer type (`int`, any sized `int*`/`uint*`, `uintptr`) or a `sync/atomic` type, unless that declaration is allowlisted per `R-AEE-012`. "Recursively contains" MUST cover at least: named types whose underlying type qualifies, struct types with a qualifying field at any depth, and array types of a qualifying element — so a counter wrapped in one unexported struct is still found. The guard MUST also fail on any package-level function that resets such state.

#### Scenarios

- **S-AEE-031** — Given the landed package, when the guard runs, then it passes.
- **S-AEE-032** — Given a scratch package-level `var scratchSeq uint64` added to a non-test file, when the guard runs, then it fails and names the file and identifier; recorded red, then dropped.
- **S-AEE-033** — Given a scratch package-level var of an unexported struct type whose only field is a `uint64`, when the guard runs, then it still fails and names it; recorded red, then dropped.
- **S-AEE-034** — Given a scratch package-level reset function for sequence state, when the guard runs, then it fails and names it; recorded red, then dropped.
- **S-AEE-035** — Given a package-level counter added to a `_test.go` file, when the guard runs, then it passes — the guard's subject is the shipped contract, not its tests.

### R-AEE-012 — The allowlist is an explicit, reasoned, non-stale table

The guard MUST carve out non-sequence state through an explicit allowlist and MUST NOT be softened into a blanket exemption. Each allowlist entry MUST carry three fields: the **file** (path relative to the package directory), the **identifier**, and a **non-empty rationale** stating why that state is not `V-STR-13` sequence state. The allowlist MUST contain exactly one entry at this milestone — `message.go` / `lastMessageID` — whose rationale MUST name defect **C3** and MUST record the distinction already written in `message.go`: `V-REQ-03` message identity needs only "two messages are distinguishable", which any process-wide monotonic counter satisfies, whereas `V-STR-13` needs "every stream's first event carries 1, contiguous", which a process-wide counter structurally cannot give.

The allowlist MUST be **non-stale**: the guard MUST fail if an entry names a file or identifier that no longer exists, and MUST fail if an entry's rationale is empty. Adding an entry MUST therefore be a deliberate, reviewable act rather than a silent way to pass the guard.

#### Scenarios

- **S-AEE-036** — Given the landed allowlist, when it is enumerated, then it has exactly one entry, naming `message.go` and `lastMessageID`, with a non-empty rationale that names C3.
- **S-AEE-037** — Given the allowlist with the `lastMessageID` entry removed, when the guard runs, then it fails and names `lastMessageID`; recorded red, then restored.
- **S-AEE-038** — Given an allowlist entry naming an identifier that does not exist in the package, when the guard runs, then it fails as stale and names that entry; recorded red, then dropped.
- **S-AEE-039** — Given an allowlist entry with an empty rationale, when the guard runs, then it fails and names that entry; recorded red, then dropped.
- **S-AEE-040** — Given the change merged, when `message.go` is diffed, then `lastMessageID` is unchanged — the allowlist accommodates the shipped counter rather than the milestone rewriting it.

### R-AEE-013 — The guard records its own reason in source

The guard MUST carry a source comment stating why it exists, naming defect **C3** and the retired design's process-global atomic counter, and stating that the fix "is not a smaller counter; it is putting the counter where the stream is." A guard whose reason is not written down is deleted by the first person who finds it inconvenient.

#### Scenarios

- **S-AEE-041** — Given the guard's source, when a reader looks for its rationale, then a comment names C3, names the retired process-global counter, and states the fix.
- **S-AEE-042** — Given the guard's failure message, when it fires, then the message names the offending file and identifier and points at the recorded rationale, so a future contributor learns the rule from the failure alone.

---

## AI-14.4 — Ordering invariants

### R-AEE-014 — Every registered kind declares a payload-independent ordering descriptor

Each registered event kind MUST declare a descriptor with exactly three components:

| Component | Domain | Meaning |
|---|---|---|
| `blockRole` | `none` \| `start` \| `delta` \| `end` | The kind's position in a block's (`V-STR-14`) lifecycle. `none` means the kind is not part of a block. |
| `cardinality` | `any` \| `at-most-one` | Whether more than one event of this kind may appear on one stream. |
| `terminal` | `true` \| `false` | Whether this kind ends the stream (`V-STR-18`). |

A kind whose `blockRole` is `start`, `delta` or `end` MUST carry a block index (`V-STR-15`), which is 1-based with 0 rejected — consistent with AI-16's `R-ATE-003`. A kind whose `blockRole` is `none` MUST NOT be required to carry one. The descriptor table MUST be registered beside the kind table and MUST be covered by the same exhaustiveness assertion as `R-AEE-004`, so a kind cannot be registered without a descriptor.

#### Scenarios

- **S-AEE-043** — Given the descriptor table and the test-only witness kind, when the exhaustiveness assertion runs, then every registered kind has a descriptor and each component holds a value from its stated domain.
- **S-AEE-044** — Given a scratch kind registered without a descriptor entry, when the suite runs, then the assertion fails and names it; recorded red, then dropped.
- **S-AEE-045** — Given a descriptor whose `blockRole` is `start`, `delta` or `end`, when its kind's payload is inspected, then it exposes a readable block index; and given a `none` descriptor, then no block index is required.

### R-AEE-015 — The checker is payload-independent

The ordering checker MUST read only each event's `(kind, descriptor, block index, sequence)`. It MUST NOT reference, type-switch over, or import knowledge of any concrete payload type, and MUST NOT special-case any kind by name. Consequently, registering a new kind with a descriptor MUST make the checker apply to it **with no change to the checker's own source**.

#### Scenarios

- **S-AEE-046** — Given the checker's source, when a reviewer reads it, then it names no concrete payload type and no individual kind constant.
- **S-AEE-047** — Given a second test-only witness kind registered with a descriptor, when the checker runs against a stream containing it, then the checker constrains it correctly without any edit to the checker.
- **S-AEE-048** — Given a recorded stream, when the checker runs, then it reaches its verdict without reading any payload field other than the block index.

### R-AEE-016 — Block ordering: start precedes deltas which precede end, per block index

For every block index appearing on a stream, the checker MUST require that the `start` event for that index precedes every `delta` for that index, and that every such `delta` precedes the `end` for that index. Events belonging to different block indices MAY interleave arbitrarily; interleaving MUST NOT be reported as a violation. A `delta` or `end` with no preceding `start` for its index MUST be reported. A `start` with no matching `end` before the stream finishes MUST be reported as an unterminated block, distinguishably from an out-of-order event.

#### Scenarios

- **S-AEE-049** — Given a recorded stream with one block's start, two deltas and end in order, when the checker runs, then it reports no violation.
- **S-AEE-050** — Given a stream where a delta for block 2 appears before block 2's start, when the checker runs, then it reports a violation naming block index 2 and the offending event's sequence.
- **S-AEE-051** — Given a stream with two blocks whose events interleave, when the checker runs, then it reports no violation.
- **S-AEE-052** — Given a stream whose block 1 start has no matching end, when the checker runs, then it reports an unterminated block, distinguishable from an ordering violation.
- **S-AEE-053** — Given a stream with a block start immediately followed by its end and zero deltas, when the checker runs, then it reports no violation — a zero-delta block is legal (`V-STR-16`).

### R-AEE-017 — `at-most-one` is a reusable, descriptor-driven primitive

For any kind whose descriptor declares `cardinality: at-most-one`, the checker MUST report a violation when a stream carries a second event of that kind, naming the kind and the offending event's sequence. This rule MUST be derived entirely from the descriptor and MUST NOT be written as logic specific to any particular kind, so that **AI-15 obtains "exactly one response start per stream" by registering its kind as `at-most-one`, with no change to AI-14** — as AI-15.1 test 2 requires.

#### Scenarios

- **S-AEE-054** — Given a test-only kind registered `at-most-one` and a stream carrying it twice, when the checker runs, then it reports a violation naming that kind and the second event's sequence.
- **S-AEE-055** — Given the same kind and a stream carrying it once, or not at all, when the checker runs, then it reports no cardinality violation.
- **S-AEE-056** — Given a kind registered `any` appearing many times, when the checker runs, then no cardinality violation is reported.

### R-AEE-018 — `terminal` is a reusable, descriptor-driven primitive

For any kind whose descriptor declares `terminal: true`, the checker MUST report a violation when any event follows an event of that kind on the same stream, and MUST report a violation when a stream carries more than one terminal event — because `V-STR-18` admits exactly one. Because AI-14 registers no terminal production kind, "a stream with no terminal event" MUST be reported as a **distinct, separately identifiable outcome** rather than folded into the ordering violations, so AI-15 and AI-19 can require its presence without AI-14 asserting a kind it does not own. AI-19 MUST obtain "nothing follows the terminal error" by registering `terminal: true`, with no change to AI-14 — as AI-15.2 test 2 requires of the completion event.

#### Scenarios

- **S-AEE-057** — Given a test-only kind registered `terminal: true` and a stream carrying one event after it, when the checker runs, then it reports a violation naming the following event's sequence.
- **S-AEE-058** — Given a stream carrying two events of terminal kinds, when the checker runs, then it reports a violation naming the second.
- **S-AEE-059** — Given a stream carrying no terminal event at all, when the checker runs, then it reports the "no terminal event" outcome distinguishably, and reports it as informational rather than as an ordering violation.
- **S-AEE-060** — Given a stream whose last event is its single terminal event, when the checker runs, then it reports no violation.

### R-AEE-019 — The checker runs against a recorded stream and reports through AI-04

The invariants MUST be expressed as executable code a consumer can run against a recorded stream, not as prose alone — this is the form AI-22.3 packages and AI-23 enforces. The checker MUST accept a finite ordered slice of events, MUST NOT consume a channel, and MUST NOT mutate its input. It MUST report the **first** violation in stream order, per `V-FAIL-04`, through AI-04's landed failure value with a position naming the offending event's sequence, and MUST introduce no new sentinel and no second failure type.

#### Scenarios

- **S-AEE-061** — Given a recorded stream containing two violations, when the checker runs, then it reports the one with the lower sequence.
- **S-AEE-062** — Given any reported violation, when it is inspected, then it is AI-04's failure value, `errors.Is` matches a landed sentinel, and the position renders the offending event's sequence.
- **S-AEE-063** — Given a recorded stream, when the checker runs twice over the same slice, then both runs report the same verdict and the slice is unchanged.

### R-AEE-020 — Deltas carry a fragment, and the contract forbids a snapshot

A kind registered with `blockRole: delta` MUST carry an index and only the new fragment, never a snapshot of accumulated content (`V-STR-16`). This capability MUST NOT export an accumulator, transcript rebuilder or any function reducing a block's deltas to its complete content — doc 0001 § 4.3 invariant 1 reserves accumulation for the consumer, and a full copy per token is quadratic allocation if copied and a data race if shared. This requirement is **binding on AI-16, AI-17 and AI-18**.

#### Scenarios

- **S-AEE-064** — Given the landed package surface, when a reviewer enumerates its exported identifiers, then none accumulates, joins or reconstructs a block's deltas.
- **S-AEE-065** — Given the documented steps for registering a `delta` kind, when a reader follows them, then they state that the payload carries only the new fragment and that a snapshot is forbidden.

---

## Non-functional requirements

### NFR-AEE-A — Dependency purity

The change MUST add no module dependency. `backend/agent/go.mod` MUST still declare zero requires, and both AI-00 import guards MUST still pass.

- **S-AEE-066** — Given the change merged, when `go.mod` is read, then it declares no require; and when both import guards run, then both pass.

### NFR-AEE-B — Totality

No exported function or method of this contract MUST panic for any input, including the zero value of the event, the zero sequence, a nil or empty recorded stream, a stream containing only unstamped events, and an accessor called for the wrong kind.

- **S-AEE-067** — Given a table of those extreme inputs, when each is passed through every exported entry point of this capability, then none panics.

### NFR-AEE-C — Failure reporting

Every rejection in this spec MUST be reported through AI-04's existing failure value and its landed sentinels. No new sentinel and no second failure type MUST be introduced.

- **S-AEE-068** — Given each rejecting scenario above, when its failure is inspected, then it is AI-04's failure value, `errors.Is` matches a landed sentinel, and its position names the offending field.

### NFR-AEE-D — Redaction

The event and its payloads MUST NOT render caller data through any `fmt` verb, including `%v` and `%#v`, following the `V-FAIL-13` posture already landed on `Part`. A diagnostic rendering MUST name the kind and sequence and MUST NOT name payload contents or their length.

- **S-AEE-069** — Given an event carrying a payload with known contents, when it is formatted with `%v`, `%s` and `%#v`, then no byte of the payload and no length derived from it appears in any output.

### NFR-AEE-E — Evidence

Every test-list item of AI-14.1 … AI-14.4 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. Every "recorded red, then dropped" scenario above MUST have its red output captured before the scratch subject is removed. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) in `backend/agent/` and clean `make lint`.

- **S-AEE-070** — Given `tasks.md`, when a reviewer walks the four nodes' test-list items, then each carries recorded red output, recorded green output, and a refactor note.

---

## Acceptance criteria

1. An event's kind is derived from a sealed payload; no caller can state a kind, and a payload-less event is rejected with AI-04's `ErrNotInVocabulary` at a named position.
2. A registry exhaustiveness assertion is non-vacuous at this milestone via a test-only witness payload bridged through `export_test.go`, and bites on a scratch kind missing a table entry, a constructor, or a descriptor.
3. Kind, sequence and typed payload are readable from `package ai_test` with no type switch over unexported types.
4. Zero production event kinds ship; adding one later is an append, with no requirement of this capability edited.
5. One stream's sequences read 1…N contiguously; two concurrent streams each start at 1 and are independently contiguous under `-race`; a third stream started after both finish also starts at 1.
6. The never-stamped sentinel is 0, rejected with `ErrOutOfRange` at the named producer emission boundary; cross-stream meaninglessness is stated in the contract text.
7. The AI-14.3 guard scans the whole non-test package by AST, fails on a scratch package-level counter and on one wrapped in a struct, and passes with exactly one reasoned `lastMessageID` allowlist entry; a stale or unreasoned entry fails the guard.
8. Ordering invariants run as code against a recorded stream, reading only `(kind, descriptor, block index, sequence)`.
9. `at-most-one` and `terminal` are descriptor-driven primitives proven with test-only witness kinds — AI-15 and AI-19 obtain their invariants by registration alone.
10. `make test` green under `-race`, `make lint` clean, both import guards passing, `go.mod` still zero requires.

## Open items carried into design

1. The stamper's exact type name and exported-ness are deferred to `design.md`; AI-20 (its eventual caller) has not landed, and this spec constrains only its per-stream, non-global, single-writer behaviour.
2. `design.md` owns the exact AST match rule for `R-AEE-011`'s recursive type matching — depth limit, treatment of type aliases, and behaviour on generic type parameters — and MUST record the chosen limit and its reason.
3. `design.md` owns where the `R-AEE-010` producer emission boundary lives as a callable surface at a milestone with no provider interface, and MUST record whether it is an exported validation entry point or an unexported one exercised through `export_test.go`.
4. `R-AEE-018`'s "no terminal event" informational outcome is a shape AI-15 and AI-19 will consume; `design.md` MUST state its representation so those milestones can require presence without re-specifying it.
5. The two Wave 1 learning docs (`learning/sdd-rework-patterns`, `learning/sdd-spec-inconsistency-patterns`) were unavailable to the explore executor; the orchestrator SHOULD re-pull them before `sdd-design`.

> **Deviation note**: this artifact exceeds the sdd-spec 650-word budget. The landed capability-spec precedent in this repository (`openspec/specs/ai-content-parts/spec.md`, `openspec/specs/ai-completion-metadata/spec.md`, and AI-16's `ai-text-events` spec in this same wave) carries per-requirement scenario blocks, non-functional requirements and acceptance criteria at this density; house convention wins.
