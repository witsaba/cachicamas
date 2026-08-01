# Spec — roles and message identity

> **Change**: `cachicamas-ai-message-roles`
> **Milestone**: AI-05 · **Nodes**: AI-05.1 `[leaf]`, AI-05.2 `[leaf]`, AI-05.3 `[leaf]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-message-roles/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-AMSG-0NN` · **Scenario IDs**: `S-AMSG-0NN`
> **Binding input**: [the register](../../../../specs/ai-contract-vocabulary/spec.md) — `V-REQ-01`, `V-REQ-02`, `V-REQ-03`, `V-REQ-04`, `V-REQ-22`, `V-FAIL-01` … `V-FAIL-04`, `V-OUT-02` · [AI-04's spec](../../../cachicamas-ai-validation-errors/specs/ai-validation-errors/spec.md)

---

## Purpose

Constrain the runtime behavior of Layer 1's message: its role vocabulary, its identity, its ordered content, and the copy semantics that make it safe to hand to a consumer. Every scenario below is verifiable by a test that runs.

One requirement — `R-AMSG-009` — constrains the **shape of the exported surface** rather than its behavior, because the property it protects (that AI-06.1 inherits an undecided question) is not observable at runtime. It is marked where it appears and is verified by inspection in `tasks.md`'s verification pass.

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A role** — `V-REQ-01`. The closed, provider-neutral vocabulary value naming who a message is attributed to.
- **The role vocabulary** — the closed set a role is drawn from. Closed, not advisory.
- **A message** — `V-REQ-02`. One role plus ordered content.
- **A message identity** — `V-REQ-03`. The stable handle by which one message is distinguished from another.
- **Content** — the ordered sequence of `V-REQ-04` content parts a message carries. What a part *is* belongs to AI-06.
- **The content seam** — the named position in this contract that a content part occupies. An implementation position, not a Layer 1 concept.
- **A caller-contract failure**, **a validation sentinel**, **a position**, **the rule list** — `V-FAIL-01` … `V-FAIL-04`, as landed by AI-04.

---

## R-AMSG-001 — The role vocabulary is closed and enumerable

Layer 1 MUST expose a closed role vocabulary. Its members MUST be enumerable in a stable declaration order, and the enumeration MUST NOT be mutable by a consumer. A value that is not a member MUST NOT be usable anywhere a role is accepted.

### Scenarios

- **S-AMSG-001** — Given the role vocabulary, when a consumer enumerates it, then the members come back in a stable order that is identical on every call.
- **S-AMSG-002** — Given a consumer that mutates the enumeration it received, when it enumerates again, then the vocabulary is unchanged.
- **S-AMSG-003** — Given a role value that is not a member of the vocabulary, when it is offered anywhere a role is accepted, then it is rejected.

---

## R-AMSG-002 — A message carries each vocabulary role, and the role reads back exactly

WHEN a message is constructed with any member of the role vocabulary and at least one content element, THEN construction MUST succeed and the role MUST read back equal to the one supplied. This is AI-05's walking skeleton: the thinnest end-to-end path through the public surface.

### Scenarios

- **S-AMSG-004** — Given each member of the role vocabulary in turn, when a message is constructed with it, then construction succeeds AND the role read back equals the one supplied.
- **S-AMSG-005** — Given a message constructed from an external package, when its role is read, then it is readable through the public surface with no access to unexported state.

---

## R-AMSG-003 — A role outside the vocabulary is a caller-contract failure

WHEN a message is constructed with a role that is not a member of the vocabulary, THEN construction MUST fail with the caller-contract failure whose rule class is *value is outside a closed vocabulary*, positioned at the role. The failure MUST be matchable through wrapping and MUST NOT be a distinct error type of this contract's own. The vocabulary is closed, not advisory: an unrecognised role MUST NOT be passed through, defaulted, or coerced.

### Scenarios

- **S-AMSG-006** — Given a role value outside the vocabulary, when a message is constructed with it, then construction fails AND the failure matches the closed-vocabulary sentinel AND does not match any other sentinel of the set.
- **S-AMSG-007** — Given that failure, when a consumer extracts its position, then the position names the role field.
- **S-AMSG-008** — Given the zero value of the role type, when a message is constructed with it, then construction fails the same way — a role that was never set is not a default.
- **S-AMSG-009** — Given a failed construction, when the returned message is inspected, then it carries no identity, so a caller that ignored the failure cannot mistake it for a constructed message.

---

## R-AMSG-004 — Rendering is stable and lowercase; parsing is exact

Each member of the role vocabulary MUST render to a stable, lowercase, non-empty string, and that rendering MUST parse back to the same member. Parsing MUST accept **only** the exact rendering: not a different case, not a padded form, not an alias, not a provider's own role string. A rendering of a non-member MUST NOT parse.

### Scenarios

- **S-AMSG-010** — Given each member of the vocabulary, when it is rendered and the result is parsed, then the parse yields the same member AND the rendering is non-empty and equal to its own lowercase form.
- **S-AMSG-011** — Given a string that differs from a member's rendering only by case or surrounding whitespace, when it is parsed, then parsing fails with the closed-vocabulary sentinel.
- **S-AMSG-012** — Given an empty string, when it is parsed, then parsing fails with the *required value is empty* sentinel, in the documented rule order — empty and not-in-vocabulary are two different facts.
- **S-AMSG-013** — Given a role value outside the vocabulary, when it is rendered and the rendering is parsed, then the rendering identifies the value for a diagnostic reader AND parsing it fails.
- **S-AMSG-014** — Given a parse failure, when its rendered message is inspected, then the string that was offered appears nowhere in it.

---

## R-AMSG-005 — Vocabulary registration is exhaustive *(pin)*

Every member of the role vocabulary MUST have a table entry that supplies its rendering, its parse mapping and its validity. A member declared without one MUST fail a mechanical assertion over the vocabulary. The assertion MUST enumerate the **declared** members, not the table, or it cannot detect the omission it exists to detect.

This is the pattern every later closed vocabulary in this package reuses — AI-07 reasoning state, AI-08 tool choice, AI-13 finish reasons — and `design.md` § 3 states it as four rules.

### Scenarios

- **S-AMSG-015** — Given the vocabulary as declared, when the assertion walks every member, then each renders non-empty, round-trips through parse, and is accepted by message construction.
- **S-AMSG-016** — Given a scratch member declared without a table entry, when the assertion runs, then it fails. *(Bite proof: recorded, then the scratch member is removed.)*

---

## R-AMSG-006 — A message carries a stable identity

A message MUST carry an identity that is comparable, that is minted by construction rather than supplied by the caller, and that does not change across reads. Two messages MUST NOT share an identity, including two constructed from identical role and content. An identity MUST NOT be derivable from a message's contents, because a content-derived identity is not stable under a design where two messages may legitimately be identical.

### Scenarios

- **S-AMSG-017** — Given a constructed message, when its identity is read repeatedly, then every read yields the same value.
- **S-AMSG-018** — Given two messages constructed from identical role and content, when their identities are compared, then they differ.
- **S-AMSG-019** — Given a message value that was never constructed, when its identity is inspected, then the identity reports that it is unset — a consumer can tell a constructed message from a zero one.
- **S-AMSG-020** — Given many messages constructed concurrently, when their identities are collected, then all are distinct AND the race detector reports nothing.
- **S-AMSG-021** — Given an identity, when a consumer renders it, then the rendering is a diagnostic form only; the identity is not constructible from it.

---

## R-AMSG-007 — Content order round-trips exactly

A message MUST carry its content as an ordered sequence, and reading it back MUST yield exactly the elements supplied, in exactly the order supplied. This MUST hold when the sequence contains the same element more than once: a message MUST NOT deduplicate, reorder, coalesce or normalize its content.

### Scenarios

- **S-AMSG-022** — Given a message constructed from an ordered sequence of distinct elements, when its content is read, then the elements come back in the same order, with the same length.
- **S-AMSG-023** — Given a message constructed from a sequence containing the same element several times, when its content is read, then every repetition is present, in position — the length and the order both match.
- **S-AMSG-024** — Given a message constructed with exactly one element, when its content is read, then it holds exactly that one element.

---

## R-AMSG-008 — A message with no content is a caller-contract failure

WHEN a message is constructed with no content, THEN construction MUST fail with the *required value is empty* sentinel, positioned at the content. This MUST hold whether the absence is expressed as no arguments, an empty sequence, or a nil sequence — the three MUST NOT be distinguishable outcomes. The rule order MUST be documented and deterministic: a message that is both wrongly-roled and empty MUST report the same failure on every run.

### Scenarios

- **S-AMSG-025** — Given a construction with no content, when it is attempted, then it fails with the empty sentinel AND the position names the content field.
- **S-AMSG-026** — Given a construction with an empty sequence and one with a nil sequence, when both are attempted, then both fail identically.
- **S-AMSG-027** — Given a construction that violates both the role rule and the content rule, when it is attempted many times, then every attempt reports the same failure, which is the first rule in the documented order.

---

## R-AMSG-009 — The content seam leaves AI-06's decision open *(constrains the exported surface)*

The named position through which a message holds a content part MUST expose no payload, no discriminator, no accessor, no constructor and no rendering. It MUST NOT validate a content element, and it MUST NOT prevent an external package from satisfying it. `V-REQ-06` content-part readability and `V-REQ-07` content-part sealing are AI-06.1's to decide **together, before any code exists**, and nothing landed by this milestone may constitute either.

### Scenarios

- **S-AMSG-028** — Given the landed surface, when a reviewer reads every declaration related to content, then the seam declares exactly one method, that method is unexported, and it returns nothing.
- **S-AMSG-029** — Given a content element supplied by an external package, when a message is constructed with it, then construction succeeds — this milestone rejects no content element, so AI-06.3 item 1 remains an assertion that can fail before it passes.
- **S-AMSG-030** — Given the seam's own documentation, when a reviewer reads it, then it names AI-06 as the owner of what a content part is and records that an external package can satisfy the seam by embedding it.

---

## R-AMSG-010 — Construction copies, and reads copy

A constructed message MUST be observably unchanged by any subsequent operation a caller performs on the values it passed to the constructor, including replacing an element of the sequence, appending to it, truncating it, or reusing its backing array. A read of a message's content MUST return a value whose mutation does not change the message. The same MUST hold for the enumeration of the role vocabulary.

### Scenarios

- **S-AMSG-031** — Given a message constructed from a caller's sequence, when the caller replaces an element of that sequence in place, then the message's content is unchanged.
- **S-AMSG-032** — Given a message constructed from a caller's sequence, when the caller appends to it, truncates it, or reuses its backing array for a second construction, then neither message's content is affected.
- **S-AMSG-033** — Given a consumer that read a message's content, when it replaces an element of what it received, then re-reading the message yields the original content.
- **S-AMSG-034** — Given two consumers that each read the same message's content, when one mutates its copy, then the other's is unaffected.
- **S-AMSG-035** — Given a message copied by assignment, when the copy's content is read and mutated, then the original message is unaffected.

---

## R-AMSG-011 — Totality: no input causes a panic

Constructing, reading, rendering and parsing MUST NOT panic for any input, including a nil content sequence, a nil content element, a role far outside the declared range, an empty string, a very long string, and a very long content sequence.

### Scenarios

- **S-AMSG-036** — Given a nil content sequence, a nil content element, and a sequence of many thousands of elements, when each is constructed and read, then nothing panics.
- **S-AMSG-037** — Given a role value at the extremes of its underlying type, when it is rendered, parsed and used in construction, then nothing panics.
- **S-AMSG-038** — Given an empty and a very long string, when each is parsed as a role, then nothing panics AND the failure carries none of the string.

---

## Non-functional requirements

### NFR-AMSG-A — Dependency purity

- The implementation MUST use only the Go standard library. `backend/agent/go.mod` MUST still carry zero requires.
- Both AI-00 import guards MUST still pass unchanged.

### NFR-AMSG-B — One failure vocabulary

- Every rule violation MUST report through AI-04's failure value and one of its landed rule classes. This change MUST NOT declare a sentinel, an error type, or an error variable of its own.

### NFR-AMSG-C — Evidence

- Every test-list item of AI-05.1, AI-05.2 and AI-05.3 MUST be taken red before green, and both outputs MUST be recorded in `tasks.md`. The one exception is AI-05.1's pin, which doc 0002 marks exempt from red-first, which MUST be fully mechanical, and which MUST be **shown to bite** against a scratch violation that is then removed.
- The milestone closes only on recorded green `make test` in `backend/agent/`, which is `go test -race -v ./...`.

### NFR-AMSG-D — Reviewability

- Test functions MUST follow `Test<Subject>_<Behavior>_<Expectation>` and MUST carry a banner comment citing the leaf ID.
- No file owned by another concurrent milestone — `validation.go`, `validation_test.go` — may be modified.

---

## Acceptance criteria

The contract holds when:

1. `R-AMSG-001` … `R-AMSG-011` hold, each verified by its scenarios.
2. Every test-list item of AI-05.1, AI-05.2 and AI-05.3 is closed with recorded red and green output, in order, and the pin's bite is recorded.
3. `make test` in `backend/agent/` is green under `-race`, and the output is recorded; `make lint` is clean.
4. **doc 0002's own acceptance criterion:** a message is constructible only through the rules, its content order round-trips, and a caller cannot mutate a constructed message from outside.
