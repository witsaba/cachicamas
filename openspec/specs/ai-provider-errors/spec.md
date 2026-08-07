# Spec — the provider error taxonomy and the terminal error event

> **Milestone**: AI-19 — Wave 2 "Stream" **keystone**, deliberately before AI-20 · **Nodes**: AI-19.1 `[leaf]` terminal error event, AI-19.2 `[leaf]` category vocabulary, AI-19.3 `[leaf]` retry hints + safe metadata, AI-19.4 `[leaf]` partial-output discriminator, AI-19.5 `[leaf]` one vocabulary, two paths
> **Introduced by**: `openspec/changes/archive/2026-08-01-cachicamas-ai-provider-errors/`, merged to `main` in PR #104 (commit `37898c7`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: **C4** by construction — the terminal error event's payload is constructible from another package, which is exactly the property whose absence defined the defect. It is also the Layer 1 half of **G8**, the typed taxonomy with a partial-output discriminator (AI-19.2 … AI-19.5); the remaining halves are AI-32.2, AI-32.3 and AI-35.1, with suite case AI-23.4
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-AIP-0NN` · **Scenario IDs**: `S-AIP-0NN`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md) — `V-FAIL-05` provider/transport failure, `V-FAIL-06` failure category, `V-FAIL-07` retryability, `V-FAIL-08` partial output, `V-FAIL-09` partial-output discriminator, `V-FAIL-10` terminal error event, `V-FAIL-11` pre-stream delivery, `V-FAIL-12` mid-stream delivery, `V-FAIL-13` safe metadata. Cited by identifier, never redefined
> **Binding predecessor**: [the event envelope](../ai-event-envelope/spec.md) (AI-14) — `V-STR-18` terminal-event container invariants stay AI-14.4's and are not restated here; the failure payload obtains terminality by registration
> **Binding predecessor**: [the validation error taxonomy](../ai-validation-errors/spec.md) (AI-04) — `R-AIE-006` content-free rendering, `R-AIE-010` caller-contract-only reservation. This taxonomy is `Violation`'s complement, never its extension
> **Binding neighbours**: `V-FAIL-15` Layer 1 retry clause (AI-35) · `V-OUT-11` retry/failover policy (Layer 2)
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-19--define-the-provider-error-taxonomy-and-the-terminal-error-event) §§ AI-19.1 … AI-19.5 (lines 1035–1097) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) § 3.1 **C4**, § 4.3 invariant 4, § 7 **G8** · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

---

## Purpose

This spec constrains **one** inspectable vocabulary for everything that goes wrong after a valid request leaves the process (`V-FAIL-05`), and requires it to be reachable by **both** delivery paths — returned directly when a request never becomes a stream (`V-FAIL-11`), and carried as the stream's terminal error event payload when a stream dies mid-flight (`V-FAIL-12`).

Two properties are load-bearing and are stated as requirements rather than notes, because each records a defect this milestone exists to close:

1. **C4** — a terminal error event whose payload no adapter could construct. `R-AIP-001` requires construction from **another package**.
2. **G8** — a consumer that cannot tell *"did any output precede this failure?"* from the failure value alone. `R-AIP-010` … `R-AIP-012` require the discriminator to stay perpendicular to the delivery path, so the naive predicate "retry if nothing completed" cannot be re-expressed accidentally.

What this spec does **not** constrain: which vendor wire failure maps to which category (AI-32), backoff execution or retry scheduling (AI-35, Layer 2), the adversarial redaction sweep (AI-36), terminal-event container invariants (AI-14.4), and cross-package conformance iteration (AI-23.4).

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The nine-category vocabulary, the retryability and retry-after signals, the partial-output discriminator, the two delivery paths and the single concrete failure type therefore live here, in their own text, and not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-01-cachicamas-ai-provider-errors/`](../../changes/archive/2026-08-01-cachicamas-ai-provider-errors/) is the historical record of how AI-19 was explored, proposed, designed, applied and verified.

**The category vocabulary is append-only, and it grows here.** `R-AIP-005` fixes the discipline: a later milestone that meets an unmodelled provider failure appends a member in the pull request that needs it, together with its sentinel and its enumeration entry, and does so **in this file**. It does not widen the meaning of `unknown`, and it does not add a second vocabulary. AI-32 assigns wire failures to these categories; it does not define new ones locally.

**`R-AIP-012` is a standing prohibition, not a one-time check.** A convenience accessor added later that returns a single three-member value flattening delivery path and partial output would reproduce **G8** exactly, and fails `S-AIP-039`.

## Definitions used by this spec

- **A failure value** — the single concrete type this spec requires, implementing `error` and `Unwrap() error`, carrying category, retryability, retry-after, partial-output discriminator, delivery path, bounded raw provider label and safe metadata.
- **The category** — `V-FAIL-06`, the closed classification a consumer switches on.
- **The delivery path** — `V-FAIL-11` / `V-FAIL-12`, *how the caller received the failure*. Fixed at construction by which constructor was used.
- **The partial-output discriminator** — `V-FAIL-09`, a single boolean fact: *did normalized output events precede this failure?* Independent of the delivery path.
- **The raw provider label** — an opaque, adapter-supplied, bounded and sanitized string identifying what the provider actually said. Not normalized, not a vocabulary member.

---

## AI-19.1 — The terminal error event

### R-AIP-001 — The terminal error event is constructible from another package

Layer 1 MUST expose a public surface sufficient for a package other than `ai` to construct a complete terminal error event — payload and event — using only exported identifiers. No unexported constructor, unexported field, in-package-only interface satisfaction, or internal helper MAY be required. This requirement is the direct negation of defect **C4** and MUST be proven from an external test package, not asserted in package-internal tests.

#### Scenarios

- **S-AIP-001** — Given the landed package surface, when a consumer in a different package constructs a provider failure and wraps it as a terminal error event through exported identifiers only, then construction succeeds and the resulting event is usable without any in-package assistance.
- **S-AIP-002** — Given that external construction, when a reviewer inspects the code path it used, then every identifier it touched is exported, and no `internal/`-scoped or unexported symbol appears.
- **S-AIP-003** — Given the ordering of this milestone, when a reviewer looks for a provider interface that declares a mandatory terminal error, then none has landed yet — the payload ships before the interface that will require it.

### R-AIP-002 — The constructed event satisfies AI-14's envelope invariants

A terminal error event MUST derive its kind from its payload rather than accepting an independently supplied kind, MUST be rejected at construction when its payload is nil or of a mismatched type, and MUST report itself as terminal per `V-STR-18`. AI-19 owns only the error instance's payload (`V-FAIL-10`); the container invariants (exactly one terminal event per stream, nothing follows it) remain AI-14.4's and MUST NOT be restated as behavior of this type.

#### Scenarios

- **S-AIP-004** — Given a constructed terminal error event, when a consumer reads its kind, then the kind is the registered error-terminal kind and it was derived from the payload, not supplied separately.
- **S-AIP-005** — Given a terminal error event construction attempt with a nil payload, and separately with a payload of a non-error kind, when each runs, then each fails and reports a landed AI-04 sentinel at a position naming the payload.
- **S-AIP-006** — Given a constructed terminal error event, when a consumer asks whether it terminates the stream, then the answer is yes, and it is readable without inspecting the payload's category.

### R-AIP-003 — Terminal exclusivity is preserved and the two payloads are unconfusable

A stream MUST end in a completion event **or** a terminal error event, never both. No accessor on the error payload MAY report completion-shaped information (finish reason, usage, completion metadata), and no accessor on the completion payload MAY report a failure category or retryability. A consumer MUST be able to tell the two terminal payloads apart without parsing any string.

#### Scenarios

- **S-AIP-007** — Given a recorded stream ending in a terminal error event, when a consumer scans it for a completion event, then none is present, and the terminal error is the last event.
- **S-AIP-008** — Given both terminal payload types, when a reviewer enumerates their exported accessors, then no accessor name or return type is shared in a way that lets a consumer read a category off a completion or a finish reason off a failure.
- **S-AIP-009** — Given a terminal error event and a completion event, when a consumer discriminates them by kind alone, then the discrimination succeeds for both with no string comparison.

---

## AI-19.2 — Category vocabulary

### R-AIP-004 — The vocabulary distinguishes at minimum nine categories

The failure category vocabulary (`V-FAIL-06`) MUST contain at minimum, as distinct members: **authentication**, **authorization**, **rate limit**, **unavailable/overloaded**, **timeout**, **cancellation**, **malformed response**, **unsupported capability**, and **unknown**. Each MUST be constructible and each MUST be distinguishable from every other. **Cancellation is required, not optional** — AI-02.1's producer wording depends on a cancellation category existing. The vocabulary MAY carry more members; it MUST NOT carry fewer.

#### Scenarios

- **S-AIP-010** — Given the landed vocabulary, when a test names each of the nine charter categories in turn and constructs a failure carrying it, then all nine constructions succeed.
- **S-AIP-011** — Given those nine failures, when their categories are compared pairwise, then all nine values are mutually distinct, and no two share a rendered classification.
- **S-AIP-012** — Given a producer that reports cancellation without waiting for the transport to unwind, when it constructs its terminal error, then a cancellation category exists to carry it and no fallback to unknown is required.

### R-AIP-005 — Category membership is closed and enumerable

The vocabulary MUST be closed: a value outside the defined members MUST be rejected as not in vocabulary, reporting through AI-04's existing not-in-vocabulary sentinel rather than a new one. The zero value of the category type MUST NOT be a member, so that an unset category is never mistaken for a legal one. Membership MUST be **enumerable in a stable order** from another package, so AI-23.4 can iterate it exhaustively instead of listing cases by hand. The vocabulary is **append-only**: a later milestone that meets an unmodelled category appends a member in the pull request that needs it, following `R-AIE-003`'s discipline.

#### Scenarios

- **S-AIP-013** — Given the category type's zero value, when it is validated, then validation fails and reports AI-04's not-in-vocabulary sentinel.
- **S-AIP-014** — Given a category value numerically beyond the last defined member, when it is validated, then validation fails with the same sentinel and no panic occurs.
- **S-AIP-015** — Given the enumeration surface, when a consumer in another package iterates every category and validates each, then the iteration yields at least the nine charter members, every yielded value validates successfully, and repeated iterations yield the same order.
- **S-AIP-016** — Given a new member appended to the vocabulary, when the enumeration is iterated again, then the new member appears without any call site being edited by hand.

### R-AIP-006 — An unmodelled failure maps to unknown with its raw label preserved

WHEN a provider reports something the vocabulary does not model, the failure MUST carry category **unknown** and MUST preserve the adapter-supplied **raw provider label** for diagnostics. The label MUST be bounded in length and sanitized per `V-FAIL-13`. The label MUST NOT be interpreted, normalized, or mapped to a category by Layer 1 — AI-19 ships **no cross-vendor normalizer**; category assignment is the adapter's job (AI-32). The label MUST remain readable after the failure has been wrapped at least once, and MUST NOT be dropped when the category is unknown.

#### Scenarios

- **S-AIP-017** — Given a failure constructed with category unknown and a raw provider label, when a consumer reads the label, then it is exactly the sanitized label supplied at construction.
- **S-AIP-018** — Given that failure wrapped in one further error layer, when a consumer reaches the failure value and reads the label, then the label is still present and unchanged.
- **S-AIP-019** — Given a raw provider label longer than the type's bound, when the failure is constructed, then the retained label is bounded — truncated or rejected per `design.md` — and the type never carries an unbounded provider string.
- **S-AIP-020** — Given the landed surface, when a reviewer looks for a function mapping raw provider labels to categories, then none exists in package `ai`.
- **S-AIP-021** — Given a failure whose category is a modelled member and whose raw label is empty, when it is constructed, then construction succeeds — the label is diagnostic, not mandatory.

---

## AI-19.3 — Retry hints and safe metadata

### R-AIP-007 — Every failure carries a machine-readable retryability signal

Every provider failure MUST expose retryability (`V-FAIL-07`) as a machine-readable boolean, readable without parsing any message text. Layer 1 owns the **classification**, because it is where the wire evidence is; Layer 1 MUST NOT own the **decision** to retry, which is Layer 2's (`V-OUT-11`, doc 0001 § 6 seam 7). No backoff schedule, attempt counter, or failover hook MAY appear on this type.

#### Scenarios

- **S-AIP-022** — Given a failure of each charter category, when a consumer reads its retryability, then a boolean is returned for every one of them with no error and no string inspection.
- **S-AIP-023** — Given the landed surface, when a reviewer enumerates its exported identifiers, then none schedules a retry, computes a backoff interval, counts attempts, or selects a fallback model.

### R-AIP-008 — A provider-supplied retry-after is carried typed, with absence distinguishable from zero

WHEN a provider supplies a retry-after duration, the failure MUST carry it as a typed duration, never as text a caller re-parses. The accessor MUST report **presence separately from value**, following the two-result presence idiom already established by `usage.go`'s `TokenCount`, so that "no retry-after was supplied" and "a retry-after of zero was supplied" remain distinguishable. Retry-after MUST be readable **independently of** retryability: the two are separate facts and neither MAY be derived from the other.

#### Scenarios

- **S-AIP-024** — Given a failure constructed with a retry-after of 30 seconds, when a consumer reads it, then the value is a typed 30-second duration and presence is reported true.
- **S-AIP-025** — Given a failure constructed with no retry-after, when a consumer reads it, then presence is reported false, and the returned duration is not treated as meaningful.
- **S-AIP-026** — Given a failure constructed with an explicit retry-after of zero, when a consumer reads it, then presence is reported true and the value is zero — distinguishable from the absent case of `S-AIP-025`.
- **S-AIP-027** — Given a failure that is not retryable but carries a retry-after, and a failure that is retryable and carries none, when both are read, then both constructions succeed and neither accessor derives its answer from the other.

### R-AIP-009 — Human text is useful without a credential or a response body

The rendered string of a failure MUST be built **only** from the category's own registered text, bounded safe metadata, and fixed punctuation. It MUST NOT render the wrapped cause's text, following `R-AIE-006`'s precedent exactly. Machine-readable fields (category, retryability, retry-after, status class, provider request identity where one exists) MUST be separate accessors, not substrings of the message. `Unwrap()` MUST still expose the cause, so wrapped causes remain inspectable for any caller that asks. This makes redaction a **property of the type** rather than of every caller's discipline; the adversarial sweep remains AI-36's (`V-FAIL-14`).

#### Scenarios

- **S-AIP-028** — Given a failure wrapping a cause whose own message contains a distinctive planted sentinel resembling a credential and a raw response body, when the failure's rendered string is read, then the sentinel does not appear anywhere in it.
- **S-AIP-029** — Given that same failure, when a consumer calls `Unwrap()` and reads the cause's own message, then the planted sentinel is present — inspectability is preserved while default rendering stays clean.
- **S-AIP-030** — Given a failure of each charter category, when its rendered string is read, then it names the category in human-readable form and is non-empty, so the message is useful without any body or credential.
- **S-AIP-031** — Given a failure carrying a status class and a provider request identity, when a consumer needs those values, then it reads them from dedicated accessors, and the test does not need to parse the rendered string to obtain them.

> **Carried forward.** `R-AIP-009`'s guarantee covers `%v`, `%s` and `%+v`, which were reproduced clean. `*Failure` is the only Wave 2 payload without a `GoString()`, so `%#v` falls back to reflection and reproduces two provider-supplied fields — the sanitized, bounded `rawLabel` and `requestID`. The wrapped cause's text does **not** leak (it renders as a pointer address), so `R-AIP-009`'s literal claim holds; but the inconsistency with the wave's own four-verb rendering convention is real. Recorded as `W2` in the Wave 2 verify report and owned by Wave 3. The fix is one `GoString()` method plus one four-verb canary test.

---

## AI-19.4 — Partial-output discriminator

### R-AIP-010 — Delivery path and partial output are two perpendicular discriminating inputs

A failure MUST expose **two independent** inspectable facts:

1. the **partial-output discriminator** (`V-FAIL-09`) — a single boolean answering *did normalized output events precede this failure?*; and
2. the **delivery path** (`V-FAIL-11` / `V-FAIL-12`) — a separate accessor stating whether the caller received the failure pre-stream or mid-stream.

These MUST NOT be collapsed into one field, one enum, or one derived value. The delivery boundary is **carrier handover**, already decided by AI-02.1: a stream handed over that fails before emitting any content is **mid-stream** (`V-FAIL-12`), not pre-stream. AI-19.4.1's three distinguishable shapes are the two facts read **together**, and all three MUST be answerable **from the failure value alone**, without replaying the stream:

| Shape | Delivery path | Partial output |
| --- | --- | --- |
| Pre-stream failure | pre-stream | `false` |
| Mid-stream failure, zero output | mid-stream | `false` |
| Mid-stream failure after output | mid-stream | `true` |

The fourth cell (pre-stream × `true`) MUST be **unconstructible**: the pre-stream construction path MUST NOT accept an output flag at all.

#### Scenarios

- **S-AIP-032** — Given one failure of each of the three shapes, when a consumer reads both the delivery path and the partial-output discriminator from each value alone, then the three resulting pairs are `(pre-stream, false)`, `(mid-stream, false)` and `(mid-stream, true)`, and all three are mutually distinguishable.
- **S-AIP-033** — Given the pre-stream construction path, when a reviewer inspects its parameters, then it takes no partial-output argument, and no exported operation can set the discriminator on a pre-stream failure afterwards.
- **S-AIP-034** — Given a stream that was handed over to the caller and then failed before emitting any content, when its failure's delivery path is read, then it is mid-stream — not pre-stream — and its discriminator is `false`.
- **S-AIP-035** — Given the three shapes, when a consumer reads only the delivery path, then pre-stream and mid-stream are distinguishable but the two mid-stream shapes are not — confirming the delivery path alone is insufficient and the second input is load-bearing.

### R-AIP-011 — "Is a naive retry safe?" is answerable from the discriminator alone

A consumer holding a failure MUST be able to decide whether re-issuing could duplicate already-observed output using the partial-output discriminator **alone**, without reading the delivery path, without replaying the stream, and without counting events. This is the binding clause of `V-FAIL-15`: the partial-output case is never retried at Layer 1, because re-issuing after any semantic event has been emitted can duplicate observable output.

#### Scenarios

- **S-AIP-036** — Given a failure whose discriminator is `true`, when a consumer evaluates whether a naive retry could duplicate output, then the answer is yes from that single field, regardless of category, retryability, or delivery path.
- **S-AIP-037** — Given a failure whose discriminator is `false`, when the same evaluation runs, then the answer is no from that single field, and it is identical for the pre-stream and the mid-stream-zero-output shapes.
- **S-AIP-038** — Given a retryable failure carrying partial output, when the Layer 1 rule of `V-FAIL-15` is applied, then retryability being `true` does not override the never-retry-after-partial-output conclusion — the two facts are read separately.

### R-AIP-012 — No accessor may re-conflate the two axes

The surface MUST NOT expose any accessor, predicate, or convenience helper that returns a single value encoding both the delivery path and the partial-output fact, nor one that derives either from the other. A three-member enum flattening the product of the two axes is explicitly prohibited, because it reproduces defect **G8** by making the naive-retry predicate read two facts out of one field. This is a requirement, not a note: a later convenience accessor that collapses the axes fails this scenario.

#### Scenarios

- **S-AIP-039** — Given the landed exported surface, when a reviewer enumerates its accessors, then none returns a value whose members are pre-stream / mid-stream-no-output / mid-stream-after-output, and none named as a retry-safety verdict combines the two axes.
- **S-AIP-040** — Given the partial-output discriminator's declared type, when a reader inspects it, then it is a single boolean fact and carries no delivery information in its name, type, or documented meaning.

---

## AI-19.5 — One vocabulary, two delivery paths

### R-AIP-013 — One concrete failure type serves both delivery paths

The failure returned directly when a request never becomes a stream (`V-FAIL-11`) and the payload of the terminal error event when a stream dies mid-flight (`V-FAIL-12`) MUST be **the same concrete type**, exposing the same accessors with the same meanings. Layer 1 MUST NOT ship a second failure type, a parallel category vocabulary, or a converter between two shapes. AI-19 MUST NOT add categories to AI-04's rule-class registry and MUST NOT reuse AI-04's `Violation` for provider failures — this taxonomy is `Violation`'s complement, not its extension (`R-AIE-010`).

#### Scenarios

- **S-AIP-041** — Given a pre-stream failure and a mid-stream terminal error event payload, when a consumer inspects the dynamic type of each, then both are the same concrete type.
- **S-AIP-042** — Given both values, when a consumer reads category, retryability, retry-after, raw label and partial-output discriminator from each, then every accessor exists on both and behaves identically.
- **S-AIP-043** — Given the landed surface, when a reviewer looks for a second provider-failure type, a second category vocabulary, or a converter between two failure shapes, then none exists.
- **S-AIP-044** — Given AI-04's rule-class registry, when it is compared against its pre-AI-19 contents, then no provider category has been added to it, and no provider failure is reported as a `Violation`.

### R-AIP-014 — `errors.Is` and `errors.As` reach through at least one layer of wrapping

Each category MUST have a sentinel such that `errors.Is(err, <category sentinel>)` reports true for a failure of that category **without** the caller first unwrapping to the concrete type. `errors.As` MUST reach the full failure value, giving access to retryability, retry-after, metadata and the discriminator. Both MUST hold through **at least one** further layer of wrapping, exactly as `Violation` does today. `Unwrap()` MUST expose the original cause so that `errors.Is` also reaches sentinels belonging to the cause.

#### Scenarios

- **S-AIP-045** — Given a failure of each charter category wrapped once with an additional error layer, when `errors.Is` is called with that category's sentinel, then it reports true for the matching category and false for every other category's sentinel.
- **S-AIP-046** — Given that same wrapped failure, when `errors.As` targets the concrete failure type, then it succeeds and the recovered value reports the original retryability, retry-after presence, raw label and partial-output discriminator.
- **S-AIP-047** — Given a failure wrapping a cause that itself matches a standard-library sentinel, when `errors.Is` is called with that sentinel through both layers, then it reports true — the chain is not severed by this type.

### R-AIP-015 — Either path alone is sufficient to classify every failure

A caller that **only ever** inspects the returned pre-stream error, and a caller that **only ever** inspects the terminal error event payload, MUST each be able to classify every failure the taxonomy defines. No category, retry hint, metadata field, or discriminator MAY be reachable on one delivery path and unreachable on the other.

#### Scenarios

- **S-AIP-048** — Given a consumer that reads only pre-stream returned errors, when it is exercised over every charter category, then it classifies all of them and no category is unreachable to it.
- **S-AIP-049** — Given a consumer that reads only terminal error event payloads, when it is exercised over every charter category, then it classifies all of them, including reading retryability, retry-after and the raw label.
- **S-AIP-050** — Given the accessor set reachable on each path, when the two sets are compared, then they are identical, and no accessor is available on one path only.

### R-AIP-016 — Redaction is a property of the failure payload, not of the caller's formatting verb

The provider-failure payload MUST render redacted output under **every** formatting verb a caller can reach, including the Go-syntax verb. Specifically:

1. A caller requesting a Go-syntax representation MUST NOT receive a representation derived by reflection over the payload's internal state, and MUST NOT through it reach the **wrapped underlying cause** — the one field that may carry raw provider response text, provider-body fragments, or credential-adjacent material.
2. The Go-syntax rendering MUST agree with the payload's already-redacted textual rendering, so redaction cannot drift between the two as either evolves.
3. No content excluded from the textual rendering MAY become reachable through any other verb. Redaction MUST NOT be reachable-or-not depending on the caller's formatting choice.
4. The obligation MUST hold **totally**: an absent failure payload MUST format under every one of those verbs without panicking, returning the contract's defined absent-failure rendering (`NFR-AIP-B`).

This requirement adds no accessor and no second failure type; it constrains rendering only, and therefore leaves `R-AIP-013`'s single-type rule and `R-AIP-015`'s accessor parity untouched.

#### Scenarios

- **S-AIP-056** — Given a failure payload wrapping a cause that carries a planted sentinel string standing in for raw provider text, when the payload is formatted with each of the plain, string, extended and Go-syntax verbs, then the planted sentinel appears in **none** of the four outputs, and no other fragment of the wrapped cause appears in the Go-syntax output.
- **S-AIP-057** — Given that same failure payload, when its Go-syntax rendering is compared byte-for-byte with its redacted textual rendering, then the two are identical; and given an **absent** failure payload, when it is formatted under each of those same four verbs, then each yields the contract's defined absent-failure rendering and none panics (`NFR-AIP-B`, `S-AIP-052`).

---

## Non-functional requirements

### NFR-AIP-A — Dependency purity

The change MUST add no module dependency. `backend/agent/go.mod` MUST still carry zero requires — `errors` and `time` are standard library — and both AI-00 import guards MUST still pass.

- **S-AIP-051** — Given the change merged, when `go.mod` is read, then it declares no require; and when both import guards run, then both pass.

### NFR-AIP-B — Totality

No exported function or method of this contract MUST panic for any input, including the zero value of the category type, the zero value of the failure type, an out-of-range category, an over-long raw label, a nil wrapped cause, and a nil payload.

- **S-AIP-052** — Given a table of those extreme inputs, when each is passed through every exported entry point of this contract, then none panics and every accessor returns a defined value.

### NFR-AIP-C — Failure reporting for the taxonomy's own rejections

Construction and validation rejections **within this spec** MUST report through AI-04's existing failure value and its landed sentinels. No new sentinel type and no third failure type MUST be introduced for the taxonomy's own rule violations. The per-category sentinels of `R-AIP-014` are `errors.Is` targets for classification, not a new failure type.

- **S-AIP-053** — Given each rejecting scenario in this spec, when its failure is inspected, then it is AI-04's failure value, `errors.Is` matches a landed sentinel, and its position names the offending field.

### NFR-AIP-D — Evidence

Every test-list item of AI-19.1 … AI-19.5 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md` (`openspec/config.yaml` sets `apply.tdd: true`). The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) in `backend/agent/` and clean `make lint`.

- **S-AIP-054** — Given `tasks.md`, when a reviewer walks every test-list item of the five nodes, then each carries recorded red output, recorded green output, and a refactor note.

### NFR-AIP-E — Sequencing gate

`sdd-apply` MUST NOT start until AI-14 (`cachicamas-ai-event-envelope`) and AI-15 (`cachicamas-ai-response-events`) have landed in this worktree. `R-AIP-002` and `R-AIP-003` are asserted against their landed surfaces, not against assumed spellings.

- **S-AIP-055** — Given the worktree at apply time, when a reviewer checks for AI-14's envelope and AI-15's completion event, then both are present and the sealed-payload assumption of `R-AIP-001` has been re-verified against landed code.

---

## Acceptance criteria

1. An adapter in another package constructs the terminal error event through exported identifiers only, and it succeeds — the property whose absence was **C4**.
2. The constructed event derives its kind from its payload, rejects nil and mismatched payloads, and reports itself terminal.
3. Terminal exclusivity holds and no accessor confuses the error payload with the completion payload.
4. The vocabulary distinguishes at minimum the nine charter categories, cancellation included, each constructible and mutually distinct.
5. Membership is closed, zero-value-excluded, and enumerable in a stable order from another package.
6. An unmodelled failure maps to unknown with a bounded, sanitized raw provider label preserved across at least one wrap; no cross-vendor normalizer ships.
7. Every failure carries a machine-readable retryability boolean; no backoff, attempt counter or failover appears at Layer 1.
8. Retry-after is typed with presence separate from value — absent, zero and non-zero are three distinguishable readings.
9. The rendered string never contains the wrapped cause's text; a planted sentinel body is absent from it and present through `Unwrap()`.
10. Delivery path and partial output are two separate accessors; the three shapes are distinguishable from the failure value alone; the fourth cell is unconstructible; no accessor collapses the axes.
11. "Is a naive retry safe?" is answerable from the discriminator alone, and retryability does not override `V-FAIL-15`'s never-retry-after-partial-output clause.
12. One concrete type serves both delivery paths; no second failure type, no second vocabulary, no `Violation` reuse, no addition to AI-04's rule-class registry.
13. `errors.Is` per-category sentinels and `errors.As` to the concrete type both survive at least one wrap, and the cause's own chain is not severed.
14. Either delivery path alone classifies every failure the taxonomy defines, with identical accessor sets.
15. `make test` green under `-race`, `make lint` clean, both import guards passing, `go.mod` still zero requires.

## Node AI-19.6, checked and deliberately not appended

doc 0002's split trigger for this milestone fires only if **category-specific metadata** (rate-limit reset, quota identity) grows the member list past seven items. AI-19 landed nine categories with **no category-specific metadata field**: `R-AIP-008`'s retry-after is uniform across every category, not rate-limit-specific. The trigger is about metadata-driven growth, not the base member count, so AI-19.6 was correctly not appended. Re-checked at tasks time against the landed category count and re-confirmed at wave verification. A later milestone that adds a per-category metadata field must re-evaluate this trigger before landing it.
