# Spec — finish reasons and usage

> **Change**: `cachicamas-ai-completion-metadata`
> **Milestone**: AI-13 · **Nodes**: AI-13.1 `[leaf]`, AI-13.2 `[leaf]`, AI-13.3 `[leaf]`, AI-13.4 `[leaf]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-completion-metadata/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-ACM-0NN` · **Scenario IDs**: `S-ACM-0NN`
> **Binding input**: [the register](../../../../specs/ai-contract-vocabulary/spec.md) — `V-MET-01` … `V-MET-12`, `V-OUT-07`, `V-OUT-08`, `V-OUT-10`; [`CAP-R-03`](../../../../specs/ai-minimum-capabilities/spec.md); the AI-04 taxonomy

---

## Purpose

AI-13 ships runtime behavior, and this spec constrains that behavior. Every scenario below is verifiable by one test that runs, in the external test package `ai_test`.

Two things this spec is careful *not* to constrain, because they belong to another layer: what a consumer does about a finish reason (`V-OUT-10` — loop termination is Layer 2's) and what a token costs (`V-OUT-07`, `V-OUT-08` — money is Layer 2's and Layer 3's).

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A finish reason** — `V-MET-01`. The closed vocabulary value stating why generation stopped.
- **The vocabulary** — the seven values `V-MET-02` … `V-MET-08`: natural stop, length, tool calls, content filter, refusal, pause-turn, unknown.
- **A provider stop value** — the raw string a vendor reports for its stop condition. `V-MET-08` calls this "a provider stop condition"; the register defines no term for the string itself (`explore.md` § 5).
- **Normalization** — the total mapping from a provider stop value to a finish reason.
- **Usage** — `V-MET-09`. The record of what a response consumed.
- **A token count** — `V-MET-10`. One counted quantity within usage, independently present or absent.
- **Absent** and **zero** — `V-MET-11`. "Not reported" and "reported as nought": two different facts.
- **The cost formula** — `V-MET-12`. The unambiguous arithmetic a consumer can write over the fields without guessing whether a count is inclusive of another.

---

## R-ACM-001 — The finish-reason vocabulary is closed and complete from birth

Layer 1 MUST expose exactly seven finish reasons: natural stop, length, tool calls, content filter, refusal, pause-turn and unknown. Each MUST be independently constructible by a consumer in another package. Refusal and pause-turn MUST be present in the first landed version of the vocabulary, not added to it later.

### Scenarios

- **S-ACM-001** — Given the landed vocabulary, when a consumer in another package names each of the seven values, then each compiles, each is a distinct value, and no eighth value is in the vocabulary.
- **S-ACM-002** — Given the vocabulary, when a consumer asks whether refusal and pause-turn are members, then both are — `V-MET-06` and `V-MET-07`, and doc 0001 § 3.2's row.

---

## R-ACM-002 — The zero value names no finish reason

The zero value of the finish-reason type MUST NOT be a member of the vocabulary, and specifically MUST NOT be the unknown value. A finish reason that was never set MUST be reportable as a caller-contract failure rather than passing as a recorded outcome.

### Scenarios

- **S-ACM-003** — Given a finish reason nobody assigned, when it is validated, then it is rejected as outside the closed vocabulary, using the AI-04 vocabulary sentinel and a position.
- **S-ACM-004** — Given the unknown value, when it is validated, then it is accepted — "I do not recognise this provider string" is a recorded outcome (`V-MET-08`), not a fault.

---

## R-ACM-003 — Refusal and content filter are distinct, and the line is documented

The refusal value and the content-filter value MUST be two values. The documentation of each MUST state the line between them: a refusal is the model's own decision, a content-filter finish is a provider-side intervention, and the correct response differs.

### Scenarios

- **S-ACM-005** — Given the refusal value and the content-filter value, when a consumer compares them, then they are different values with different string forms.
- **S-ACM-006** — Given a provider stop value from the content-filter family, when it is normalized, then it becomes the content-filter value and never the refusal value; and the converse holds for the refusal family.

---

## R-ACM-004 — Normalization is total and case-insensitive

Normalization of a provider stop value MUST trim surrounding whitespace and lower the case before matching. It MUST return a finish reason for **every** input string, including the empty string, and MUST NOT return an error and MUST NOT panic. An unrecognised string MUST become the unknown value.

### Scenarios

- **S-ACM-007** — Given a provider stop value in any casing, with or without surrounding whitespace, when it is normalized, then it yields the same finish reason as its trimmed, lowered form.
- **S-ACM-008** — Given each known synonym family, when every member is normalized, then every member of a family yields that family's one value.
- **S-ACM-009** — Given a provider stop value the vocabulary does not recognise, when it is normalized, then it yields the unknown value, no error is returned, and no panic occurs.
- **S-ACM-010** — Given the empty string, an all-whitespace string and an over-long string, when each is normalized, then each yields the unknown value without error.

---

## R-ACM-005 — Every vocabulary value has a stable string form that round-trips

Each of the seven values MUST have a stable, package-authored string form. That string form MUST normalize back to the value it came from. A value outside the vocabulary MUST render as a form that is not a vocabulary word and MUST NOT normalize back to a vocabulary member other than unknown.

### Scenarios

- **S-ACM-011** — Given each of the seven values, when its string form is normalized, then the original value is recovered.
- **S-ACM-012** — Given a value outside the vocabulary, when its string form is read, then it is a fixed placeholder distinguishable from every vocabulary word, and no panic occurs.

---

## R-ACM-006 — Adding a value without extending the table and the string form fails a test *(pin)*

A test MUST fail when a value is added to the vocabulary without being given a string form and a normalization entry. The mechanism MUST be mechanical — it MUST NOT depend on a reviewer noticing — and MUST be demonstrated to fail against a deliberate scratch value before it lands.

### Scenarios

- **S-ACM-013** — Given a scratch value added to the vocabulary and nothing else changed, when the suite runs, then it fails, naming the value that has no string form and no round-trip.
- **S-ACM-014** — Given the landed vocabulary, when the suite runs, then the set of values the package treats as members is exactly the set of seven the test names by hand.

---

## R-ACM-007 — Refusal, pause-turn and unknown remain three states

The three values MUST remain three. A consumer MUST be able to write an exhaustive decision over the vocabulary in which those three are handled separately. Collapsing any two of them MUST require a change that is visible at compile time to every consumer, rather than a change of behavior behind an unchanged surface.

### Scenarios

- **S-ACM-015** — Given the three values, when a consumer compares them pairwise, then all three are distinct and all three have distinct string forms.
- **S-ACM-016** — Given a consumer's exhaustive decision over the vocabulary, when each of the seven values is passed through it, then each reaches its own branch and no value reaches a fallback branch.
- **S-ACM-017** — Given a proposal to collapse two of the three, when the change is attempted, then it removes a named constant, and every consumer that names that constant fails to compile.

---

## R-ACM-008 — The obligation attached to each value is recorded

The documentation of each of the three values MUST state the obligation it places on a consumer. In particular, the pause-turn value MUST record that resumption replays the received content verbatim. Layer 1 MUST NOT implement or decide that obligation — `V-OUT-10` reserves loop termination for Layer 2 — and MUST NOT expose a method that answers "should the loop continue?".

### Scenarios

- **S-ACM-018** — Given the pause-turn value's documentation, when a reader looks for the consumer's obligation, then it states that the correct response is to resume with the received content replayed verbatim, and cites the nodes that honor it.
- **S-ACM-019** — Given the landed surface, when a reviewer looks for a method that decides whether a turn continues, then none exists — the finish reason is the input to that decision, never the decision.

---

## R-ACM-009 — A usage record distinguishes absent from zero on every count

Usage MUST carry input, output, cache-read, cache-write and reasoning token counts. On **every** one of them, a count that was not reported MUST be distinguishable from a count reported as nought, by a consumer in another package, without parsing text.

### Scenarios

- **S-ACM-020** — Given a usage record in which one count is absent and another is present with the value zero, when a consumer reads both, then the first reports absence and the second reports a present value of zero.
- **S-ACM-021** — Given each of the five counts in turn, when it is left absent, then it reports absence; and when it is set to zero, then it reports a present zero. The distinction holds for all five, not only for the first.
- **S-ACM-022** — Given an absent count and a present zero count, when each is rendered as text, then the two renderings differ.

---

## R-ACM-010 — A usage record is constructible with any subset present

Usage MUST be constructible with any subset of its counts present, including none. A record with no count present MUST be a valid record — `CAP-R-03` clause 2 — and MUST NOT be a record of zeros. Validation MUST accept it.

### Scenarios

- **S-ACM-023** — Given a provider that reports only input and output, when its usage record is built, then it is valid, its input and output counts are present, and its three remaining counts report absence rather than zero.
- **S-ACM-024** — Given a provider that reports nothing, when its usage record is built, then it is valid and all five counts report absence.
- **S-ACM-025** — Given a usage record carrying a negative count, when it is validated, then it is rejected using the AI-04 out-of-range sentinel at a position naming the offending field, and the first violated field in the documented order is the one reported.

---

## R-ACM-011 — The input count excludes cached tokens, and the reasoning count is inside the output count

The input count MUST exclude both the cache-read count and the cache-write count: no token is counted in more than one of those three fields. The reasoning count MUST be a breakdown of the output count, not a term beside it. Both semantics MUST be documented on the declaration and MUST be pinned by a test over a constructed cache-hit record.

### Scenarios

- **S-ACM-026** — Given a cache-hit record whose input, cache-read and cache-write counts are all present, when the test computes the total input as the documented sum of the three, then the result exceeds the input count alone, and the record's input count is shown to be the uncached portion only.
- **S-ACM-027** — Given a record whose reasoning count is present, when the test computes the total output, then it equals the output count alone, and the test asserts that adding the reasoning count would exceed the tokens the response actually produced.
- **S-ACM-028** — Given the declarations of the five counts, when a reader looks for the inclusive-or-exclusive statement, then each count states which side it is on and why.

---

## R-ACM-012 — The cost formula is asserted, and its term list is pinned to the field set *(pin)*

The documented cost formula over the usage fields MUST be expressed as an assertion in the test. The record's field set MUST be pinned so that adding, removing, renaming or reordering a field fails a test, forcing the author of a later field to state which side of `R-ACM-011` it lands on.

### Scenarios

- **S-ACM-029** — Given the documented formula, when the test evaluates it over a constructed record, then the asserted result is the one the documentation states.
- **S-ACM-030** — Given a sixth field added to the record, when the suite runs, then it fails, naming the field set it expected.
- **S-ACM-031** — Given the landed surface, when a reviewer looks for an exported derivation over the counts, then none exists: the formula's terms are the record's own fields, so a consumer computing it must read each field and meet its absence rather than receive a total that silently treated absence as zero.

---

## Non-functional requirements

### NFR-ACM-A — Dependency purity

The change MUST add no module dependency. `go.mod` MUST still carry zero requires, and both AI-00 import guards MUST still pass.

- **S-ACM-032** — Given the change merged, when `go.mod` is read, then it declares no require; and when both import guards run, then both pass.

### NFR-ACM-B — Totality

No exported function or method of this change MUST panic for any input, including the zero value, a value outside the vocabulary, an empty string, an all-whitespace string and an over-long string.

- **S-ACM-033** — Given a table of extreme inputs, when each is passed through every exported entry point, then none panics.

### NFR-ACM-C — Evidence

Every test-list item of all four leaves MUST be taken red → green → refactored in order, with both outputs recorded in `tasks.md`; both pins MUST be recorded biting against a scratch violation; and the milestone MUST close on recorded green `make test` (`go test -race -v ./...`) in `backend/agent/`.

- **S-ACM-034** — Given `tasks.md`, when a reviewer walks the twelve test-list items, then each carries recorded red output, recorded green output, and the refactor note.

---

## Acceptance criteria

1. Seven finish reasons, refusal and pause-turn among them, none of them the zero value.
2. Every vendor string in `explore.md` § 4 normalizes to its value; every unrecognised string becomes unknown without error.
3. Refusal, pause-turn and unknown are three states a consumer's exhaustive decision handles separately.
4. The pause-turn obligation — resume, replaying received content verbatim — is recorded on the declaration for AI-31.1 and Layer 2.
5. All five token counts distinguish absent from zero, and a record with none present is valid.
6. Cached tokens are excluded from the input count; reasoning tokens are inside the output count; both are documented and pinned.
7. The cost formula is asserted in the test and pinned to the field set.
8. Both pins are shown to bite.
9. `make test` green with `-race`; `make lint` clean; zero requires; both import guards passing.
