# Spec — cache-boundary markers

> **Change**: `cachicamas-ai-cache-breakpoints`
> **Milestone**: AI-11 · **Nodes**: AI-11.1, AI-11.2, AI-11.3, all `[leaf]`
> **Phase**: spec (delta — new capability)
> **Canonical spec**: `openspec/specs/ai-cache-breakpoints/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-08-01
> **Requirement IDs**: `R-ACB-0NN` · **Scenario IDs**: `S-ACB-0NN`
> **Binding input**: [the register](../../../../../specs/ai-contract-vocabulary/spec.md) — `V-REQ-14`, `V-REQ-19`, `V-REQ-20`, `V-REQ-22`, `V-REQ-23`, `V-REQ-24`, `V-REQ-25`, `V-MET-09`, `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-13`, `V-FAIL-17` · [AI-10's spec](../../../2026-08-01-cachicamas-ai-model-request/specs/ai-model-request/spec.md) and its [`design.md`](../../../2026-08-01-cachicamas-ai-model-request/design.md) §§ 4, 11, 12 · [`explore.md`](../../explore.md) · [`proposal.md`](../../proposal.md)

---

## ADDED Requirements

## Purpose

Constrain the runtime behavior of `V-REQ-23` **cache-boundary marker**, `V-REQ-24` **breakpoint cap** and `V-REQ-25` **invalidation cascade** on the normalized request. This spec covers all three leaves of AI-11.

Requirement count: **10**. Implementation status is tracked in `tasks.md`, not here.

## Requirement ownership by leaf

| Leaf | Requirements |
| --- | --- |
| AI-11.1 — markers on segments, tools and messages | `R-ACB-001`, `R-ACB-002`, `R-ACB-003`, `R-ACB-004`, `R-ACB-005` |
| AI-11.2 — cap and ordering invariants | `R-ACB-006`, `R-ACB-007` |
| AI-11.3 — advisory semantics | `R-ACB-008`, `R-ACB-009` |
| all three | `R-ACB-010` |

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A cache-boundary marker** — `V-REQ-23`. An **advisory** mark on a system-instruction segment, a tool declaration, or a message, indicating a point at which a provider may cache the preceding prefix. Never a correctness requirement.
- **A carrier** — this spec's term for one of the three things a marker may be placed on: a system-instruction segment (`V-REQ-19`), a tool declaration (`V-REQ-12`), a message (`V-REQ-02`).
- **The breakpoint cap** — `V-REQ-24`. The documented maximum number of markers one request may carry.
- **The invalidation cascade** — `V-REQ-25`. The ordering tools → system instruction → messages, in which a change invalidates cached prefixes downstream of it.
- **A cache region** — this spec's term for one of the three ordered regions of the cascade.
- **A translator** — any consumer that reads a request through its exported surface and produces something else from it. An adapter is one; a test double is another.
- **A caller-contract failure**, **a validation sentinel**, **a position**, **the rule order** — `V-FAIL-01` … `V-FAIL-04`, as landed by AI-04.

---

## R-ACB-001 — Every carrier can hold a marker, and the marker round-trips

Layer 1 MUST let a caller place a cache-boundary marker on a system-instruction segment, on a tool declaration, and on a message. It MUST expose exactly one spelling for placing a marker, identical in name across the three carriers, and exactly one spelling for reading it back.

Placing a marker MUST NOT mutate the carrier it is applied to. It MUST yield a value carrying the marker, leaving the original observably unmarked, because every carrier in this package is a value that a caller may already have shared.

Placing a marker MUST NOT change any other property of the carrier: text, name, description, schema bytes, role, content, and message identity MUST read back exactly as before.

A carrier that was never marked MUST report that it is not a cache boundary. Marking a value that already carries a marker MUST be idempotent.

### Scenarios

- **S-ACB-001** — Given a constructed system-instruction segment, when a marker is placed on it, then the result reports that it is a cache boundary and its text reads back byte-equal to the original's.
- **S-ACB-002** — Given a constructed tool declaration, when a marker is placed on it, then the result reports that it is a cache boundary and its name, description and schema bytes read back unchanged.
- **S-ACB-003** — Given a constructed message, when a marker is placed on it, then the result reports that it is a cache boundary and its role, its ordered content and its identity read back unchanged.
- **S-ACB-004** — Given a marked carrier of each kind, when the value it was derived from is read, then that original reports it is **not** a cache boundary, so placement produced a copy.
- **S-ACB-005** — Given a constructed carrier of each kind that was never marked, when it is read, then it reports it is not a cache boundary.
- **S-ACB-006** — Given an already-marked carrier of each kind, when a marker is placed on it again, then the result is indistinguishable from the value it was applied to.

---

## R-ACB-002 — A marker cannot make an unconstructed value usable

WHEN a marker is placed on a carrier that never passed its constructor, THEN the result MUST still be detectable as unconstructed by the same test that detected the original, and it MUST NOT report that it is a cache boundary.

This requirement exists because each carrier's zero-value detector is derived from its payload — a segment's empty text, a declaration's empty name, a message's unminted identity — and a mechanism that set a flag on a zero value would create a value that is simultaneously unconstructed and marked. `V-REQ-23` places a marker on a member of the request; a value that was never constructed is not one.

### Scenarios

- **S-ACB-007** — Given the zero segment, when a marker is placed on it, then the result still reports itself unset and is still rejected by system-instruction construction with `ErrEmpty` at its index.
- **S-ACB-008** — Given the zero tool declaration, when a marker is placed on it, then tool-set construction still rejects it with `ErrEmpty` at its index.
- **S-ACB-009** — Given the zero message, when a marker is placed on it, then request construction still rejects it with `ErrEmpty` at `messages[0]`.
- **S-ACB-010** — Given a marker placed on any of the three zero values, when the result is asked whether it is a cache boundary, then it reports that it is not.

---

## R-ACB-003 — A marker is readable from another package wherever it can be set

WHEN a consumer in a package other than the one that owns the contract holds a carrier, or holds a request containing carriers, THEN it MUST be able to both place and read a marker through the exported surface alone, with no access to unexported state.

Reading a marker MUST require no accessor that does not already exist on the enclosing region: a marker on a segment is reachable through the system instruction's existing segment accessor, a marker on a declaration through the tool set's existing declaration accessor, a marker on a message through the request's existing message accessor.

This requirement is constitutive rather than incidental: the only consumer a marker exists for is an adapter, and every adapter lives in another package.

### Scenarios

- **S-ACB-011** — Given a request built in an external-package test from marked segments, marked declarations and marked messages, when each region is read back through its existing accessor, then every marker placed is observed at the ordinal it was placed on.
- **S-ACB-012** — Given that request, when the same regions are read a second time, then the observed markers are identical, because each read returns a fresh copy and no consumer can rewrite another's view.

---

## R-ACB-004 — Markers participate in request equality

WHEN two requests are compared under the documented request equality, THEN marker state MUST be one of the compared facts: two requests identical in every payload but differing in the placement of a single marker MUST NOT compare equal.

A marker expresses a request for different provider behavior and therefore describes a different request. Excluding markers from equality would let a whole-request round trip drop every marker and still report success — a defect with no error, no wrong answer, and an input cost roughly ten times the intended one.

### Scenarios

- **S-ACB-013** — Given two requests built from identical inputs where one carries a marker on its first system segment, when they are compared under the documented equality, then they are not equal.
- **S-ACB-014** — Given two requests built from identical inputs with identical markers on identical ordinals, when they are compared, then they are equal.
- **S-ACB-015** — Given a marked request read region by region through the exported surface and rebuilt from what was read, when the rebuild is compared to the original, then it is equal, so no marker was lost in the round trip.

---

## R-ACB-005 — A marker never decides whether a request is valid

WHEN a request is validated, THEN no marker MAY be the reason it fails, and no marker MAY be the reason it succeeds. A marked request and the otherwise identical unmarked request MUST produce the same validation outcome: the same success, or the same rule class at the same position.

The single exception is the breakpoint cap of `R-ACB-006`, which is a property of the marker **count** across the whole request and not of any individual marker's placement.

Marker placement MUST NOT be constrained by role, by content kind, by ordinal, by adjacency to another marker, or by the length of the prefix it bounds. Whether a marked prefix is long enough for a provider to cache is decidable only by asking that provider and is a provider failure (AI-19), the same treatment the request gives an unrecognised model identity and an over-high temperature.

### Scenarios

- **S-ACB-016** — Given a valid request and its twin carrying markers on every carrier within the cap, when both are constructed, then both succeed.
- **S-ACB-017** — Given an invalid request and its twin carrying the same markers, when both are constructed, then both fail with the same rule class at the same rendered position.
- **S-ACB-018** — Given requests marking a segment, a declaration and a message in every combination within the cap, when each is constructed, then every one succeeds, because no placement rule exists.
- **S-ACB-019** — Given a request whose marked segment carries a single character, when it is constructed, then it succeeds, because minimum cacheable prefix length is not decidable from the request alone.

---

## R-ACB-006 — The breakpoint cap is documented, exported, and enforced before I/O

Layer 1 MUST document a maximum number of cache-boundary markers one request may carry, and MUST expose that maximum as a readable constant, so a caller that generates markers can consult the ceiling before constructing rather than after failing.

WHEN a request's total marker count across all three regions exceeds that maximum, THEN request validation MUST fail with `ErrOutOfRange` at a position naming the marker set, and it MUST fail **before any I/O**, per `V-REQ-22`.

The count MUST be the total across tool declarations, system-instruction segments and messages. Validation MUST NOT truncate, MUST NOT drop the excess markers, and MUST NOT succeed with a warning: `V-REQ-24` names silent truncation as the failure mode this rule exists to prevent.

A request carrying exactly the maximum MUST be valid. A request carrying no marker MUST be valid.

The cap rule MUST occupy a documented position in the request's rule order and MUST run after every rule that admits a region it counts, so that a count is never taken over a region a later rule rejects.

### Scenarios

- **S-ACB-020** — Given a request whose marker count is one above the maximum, when it is constructed, then it fails, `errors.Is` reports `ErrOutOfRange`, and the position renders the marker set's name.
- **S-ACB-021** — Given a request whose marker count is exactly the maximum, when it is constructed, then it succeeds.
- **S-ACB-022** — Given a request carrying no markers at all, when it is constructed, then it succeeds and reports an empty marker set.
- **S-ACB-023** — Given a request that exceeds the cap using markers drawn from all three regions rather than one, when it is constructed, then it fails identically, because the cap is a total and not a per-region budget.
- **S-ACB-024** — Given a request that both exceeds the cap and violates an earlier structural rule, when it is constructed, then the earlier rule's failure is the one reported, per `V-FAIL-04`.
- **S-ACB-025** — Given the exported maximum, when a caller reads it, then it is available as a constant without constructing anything.
- **S-ACB-026** — Given the dependency closure of the request-validation path, when it is inspected, then it contains no network and no filesystem package, so the cap is enforced before any I/O.

---

## R-ACB-007 — Markers are readable in tools → system → messages order

Layer 1 MUST expose the markers a request carries as an ordered sequence, and that order MUST be the invalidation cascade of `V-REQ-25`: every marker in the tool region first, then every marker in the system-instruction region, then every marker in the message region.

Within a region the order MUST be the region's own ordinal order, ascending.

The order MUST NOT depend on the order in which markers were placed, on the order in which the request's options were applied, or on any iteration whose order is not fixed by the contract.

Each element of the sequence MUST name the region it belongs to and its ordinal within that region, so a consumer can resolve it back to the carrier it marks in one step. The regions MUST form a closed, ordered vocabulary whose numeric order is the cascade order, so the ordering is a property of the vocabulary rather than of a comparison written at each call site.

The reported count MUST agree with the count the cap rule enforces, for every request.

### Scenarios

- **S-ACB-027** — Given a request whose markers were placed on a message first, then on a system segment, then on a tool declaration, when its marker sequence is read, then the sequence is the tool marker, then the system marker, then the message marker.
- **S-ACB-028** — Given a request carrying two markers in one region, when its marker sequence is read, then those two appear in ascending ordinal order.
- **S-ACB-029** — Given a marker in the sequence, when its region and ordinal are read, then indexing that region's accessor at that ordinal yields a carrier that reports it is a cache boundary.
- **S-ACB-030** — Given the same request constructed many times in one process, when its marker sequence is read each time, then the sequence is identical every time.
- **S-ACB-031** — *(pin)* Given the cache-region vocabulary, when it is enumerated, then it holds exactly three members whose numeric order is tools, then system, then messages, and a member added without an entry in the enumeration fails this pin.
- **S-ACB-032** — Given any request at the cap, when its marker sequence length is compared with the count the cap rule enforced, then the two agree.

---

## R-ACB-008 — Markers are advisory: a translator may ignore every one of them

WHEN a translator reads a request through the exported surface and never consults any marker, THEN the request MUST remain fully translatable, and the result MUST be identical to the result of translating the otherwise identical unmarked request.

No region of a request MUST be reachable only by way of a marker. No accessor MUST require a marker to have been placed. Ignoring markers wholesale MUST be a conformant adapter strategy, because two target providers cache prefixes automatically and honouring a marker there would be meaningless (doc 0001 § 3.3 row 9).

Conversely, a translator that **does** consult markers MUST be able to observe a difference between a marked request and its unmarked twin, so the identity above is a statement about the marker-blind translator and not about a translator that observes nothing.

### Scenarios

- **S-ACB-033** — Given a marked request and its unmarked twin, when each is rendered by an external-package translator that reads every region and never consults a marker, then the two renderings are identical.
- **S-ACB-034** — Given the same pair, when each is rendered by a translator that does consult markers, then the two renderings differ — the control that proves `S-ACB-033` is not vacuous.
- **S-ACB-035** — Given a marked request, when the marker-blind translator reads it, then every region it reads is complete: no text, declaration, message, content part or generation option is missing relative to the unmarked twin.

---

## R-ACB-009 — The usage record is untouched by this milestone

The response-side record of what a call consumed MUST continue to carry its cache-read and cache-write token counts exactly as landed, and this milestone MUST add, remove, rename or re-validate nothing there.

Layer 1 MUST NOT measure, aggregate, or report a cache hit rate. `V-REQ-25` places measurement outside this layer, and `V-OUT-07` places pricing above it.

### Scenarios

- **S-ACB-036** — *(pin)* Given the usage record, when its cache-read and cache-write counts are read and validated, then they behave exactly as before this milestone, and the position their validation reports is unchanged.
- **S-ACB-037** — *(pin)* Given the exported surface of this layer, when it is inspected for a hit-rate, cache-efficiency or cache-statistics accessor, then none exists.

---

## R-ACB-010 — Marker state renders as structure, never as payload

The renderings a carrier and a request produce MAY name marker state, because a marker is a structural fact about the request rather than a caller's payload.

WHEN marker state is rendered, THEN the rendering MUST NOT reproduce any payload it neighbours: not a segment's text, not a declaration's name or schema, not a message's content, not a model identity, not an option value. The unmarked rendering of every carrier and of the request MUST be unchanged by this milestone, so no existing rendering assertion changes meaning.

Both a string rendering and a Go-syntax rendering MUST continue to be defined wherever they were, per `V-FAIL-13`, so the posture stays a property of the type rather than of which verb a caller reached for.

### Scenarios

- **S-ACB-038** — Given a marked system-instruction segment holding a secret, when it is formatted through the default, string, extended and Go-syntax verbs, then every output names that it is a cache boundary and none reproduces the secret.
- **S-ACB-039** — Given an unmarked segment, when it is formatted, then its rendering is byte-identical to what it was before this milestone.
- **S-ACB-040** — Given a request carrying markers and a secret in every region, when it is formatted through all four verbs, then the rendering names how many cache boundaries the request carries and reproduces no secret.
