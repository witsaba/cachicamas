# Spec — tool declarations

> **Milestone**: AI-08 — Define tool declarations · **Nodes**: AI-08.1 `[leaf]`, AI-08.2 `[leaf]`, AI-08.3 `[leaf]`
> **Introduced by**: `openspec/changes/archive/2026-08-01-cachicamas-ai-tool-declarations/`, in PR #101 (open for review at archive time, base `main` @ `efdedc4`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-ATD-0NN` · **Scenario IDs**: `S-ATD-0NN`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md) — `V-REQ-12` … `V-REQ-15`, `V-REQ-25`, `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-17`, `V-OUT-04`, `V-OUT-05`, `V-OUT-17`; every Layer 1 noun below is one of its rows, cited by identifier
> **Binding predecessor**: [the validation error taxonomy](../ai-validation-errors/spec.md) — every rule violation here reports through AI-04's failure value and its landed rule classes, cited rather than re-decided
> **Binding predecessor**: [roles and message identity](../ai-message-roles/spec.md) — the closed-vocabulary pattern this contract reuses for tool choice is AI-05's, extended rather than re-derived
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-08--define-tool-declarations) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

---

## Purpose

Constrain the runtime behavior of Layer 1's tool transport representation: what a tool declaration holds and how faithfully it holds it, what a set of declarations guarantees about its own order, and how a tool choice is validated against the set it names. Every scenario below is verifiable by a test that runs.

The register's first trap governs the whole file. Layer 1 owns the representation because it crosses the model API; it owns nothing about what a tool *does*. No requirement below executes a tool, resolves a name, or judges a schema.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The tool declaration, the tool set and the tool-choice vocabulary therefore live here, in their own text, and not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-01-cachicamas-ai-tool-declarations/`](../../changes/archive/2026-08-01-cachicamas-ai-tool-declarations/) is the historical record of how AI-08 was explored, proposed, designed, applied and verified.

AI-08 is also the milestone that exposed two gaps in [the validation error taxonomy](../ai-validation-errors/spec.md): it met the first violation no landed rule class described — duplicate tool names in one set — and reported it through the nearest-fitting class before the class was appended. `R-AIE-003`'s second paragraph, `S-AIE-034` and `S-AIE-035` were appended there as a result. That amendment is recorded in AI-04's file, not restated as normative here.

AI-09 adds tool calls and tool results; AI-10 composes the tool set and the tool choice into the normalized request core and owns their cross-validation position prefix. Requirement and scenario identifiers are stable and append-only.

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A tool declaration** — `V-REQ-12`. The provider-neutral transport representation of one tool a model may call: its name, its description, and its schema bytes.
- **Schema bytes** — `V-REQ-13`. The declaration's argument schema, carried byte-faithfully and never validated against a meta-schema.
- **A tool set** — `V-REQ-14`. The ordered, deterministically iterable collection of declarations offered on one request.
- **A tool choice** — `V-REQ-15`. The request-level instruction constraining whether and which tool the model may call.
- **The tool-choice vocabulary** — the closed set a tool choice's member is drawn from: *automatic*, *none*, *required*, *specific*. Closed, not advisory.
- **The name rule** — the documented character and length constraint a tool name must satisfy. Its exact form is `design.md` § 3's; this spec constrains only that one exists, is applied wherever a tool name is accepted, and reports through the taxonomy.
- **A caller-contract failure**, **a validation sentinel**, **a position**, **the rule list**, **a rule class** — `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-17`, as landed by AI-04.

---

## R-ATD-001 — A declaration carries a name, a description and schema bytes, all readable from another package

WHEN a tool declaration is constructed with a name satisfying the name rule, a description, and non-empty schema bytes, THEN construction MUST succeed and all three MUST read back through the public surface, from a package that is not the declaring one, equal to what was supplied. This is AI-08's walking skeleton: the thinnest end-to-end path through the public surface.

An empty description MUST be legal. Every provider a v1 adapter could target treats a tool's description as optional; a value that is merely *recommended* is not a caller-contract failure.

### Scenarios

- **S-ATD-001** — Given a name, a description and schema bytes, when a declaration is constructed from an external package, then construction succeeds AND the name, the description and the schema bytes each read back equal to what was supplied, with no access to unexported state.
- **S-ATD-002** — Given a declaration constructed with an empty description, when it is validated and read, then construction succeeds AND the description reads back empty.
- **S-ATD-003** — Given a failed construction, when the returned declaration is inspected, then it carries no name, so a caller that ignored the failure cannot mistake it for a constructed declaration.

---

## R-ATD-002 — Schema bytes pass through byte-identically

Schema bytes MUST be carried exactly as supplied: no re-marshalling, no key reordering, no whitespace normalization, no canonicalization of any kind (`V-REQ-13`). WHEN schema bytes are read back, THEN they MUST be **byte-equal** to those supplied, including for input whose key order and whitespace a marshal-and-unmarshal round trip would rewrite.

This is a correctness requirement, not a convenience. `V-REQ-25`'s invalidation cascade makes the tool set the first region of a cached prefix; a byte that changes between two otherwise identical requests invalidates that prefix silently, with no error and no wrong answer.

The declaration MUST NOT alias the caller's buffer, in either direction: mutating the slice passed to the constructor MUST NOT change the declaration, and mutating the slice returned by a read MUST NOT change it either.

### Scenarios

- **S-ATD-004** — Given schema bytes whose object keys are not in alphabetical order and whose whitespace is irregular, when they are read back from an external package, then the bytes are equal to the supplied bytes, byte for byte.
- **S-ATD-005** — Given a constructed declaration, when the caller mutates the slice it passed to the constructor, then the declaration's schema bytes are unchanged on re-read.
- **S-ATD-006** — Given a constructed declaration, when a consumer mutates the slice a read returned, then the declaration's schema bytes are unchanged on re-read AND a second consumer observes no change.

---

## R-ATD-003 — Declaration construction rules fail through the taxonomy, in a documented order

Every rule violation in declaration construction MUST be reported as a caller-contract failure through AI-04's taxonomy, positioned, with no error type or sentinel of this contract's own. The rules MUST be evaluated in a documented order and MUST report the first failure identically across runs (`V-FAIL-04`).

The rules, and the rule class each violation belongs to:

1. An empty name — *a required value is empty*.
2. A name longer than the documented ceiling — *a value outside a documented bound*.
3. A name containing a character the name rule excludes, or beginning with a character the name rule excludes from that position — *a value not well-formed for its documented encoding*.
4. Empty schema bytes — *a required value is empty*.

Rules 2 and 3 are distinct classes because they are distinct facts with distinct fixes, which is `V-FAIL-17`'s own criterion: the class is the kind of thing a rule checks, not the field it checks it on.

### Scenarios

- **S-ATD-007** — Given an empty name, when a declaration is constructed, then construction fails with the empty sentinel AND the position names the name field.
- **S-ATD-008** — Given a name longer than the documented ceiling, when a declaration is constructed, then construction fails with the out-of-range sentinel AND does not match the malformed sentinel.
- **S-ATD-009** — Given a name containing a character outside the documented alphabet, when a declaration is constructed, then construction fails with the malformed sentinel AND does not match the out-of-range sentinel.
- **S-ATD-010** — Given a name whose first character is excluded from that position by the name rule, when a declaration is constructed, then construction fails with the malformed sentinel.
- **S-ATD-011** — Given empty schema bytes, when a declaration is constructed, then construction fails with the empty sentinel AND the position names the schema field.
- **S-ATD-012** — Given an input violating several rules at once, when a declaration is constructed, then the failure reported is the first in the documented order, identically on every run.
- **S-ATD-013** — Given any failed construction, when the failure is rendered, then the rendering carries none of the offered name or schema bytes.

---

## R-ATD-004 — Layer 1 does not judge the schema

Schema bytes MUST NOT be validated against a schema meta-schema, and MUST NOT be interpreted, parsed for meaning, or rejected on the grounds of what they describe. WHEN schema bytes are non-empty, THEN their content MUST NOT affect whether construction succeeds.

This is the register's trap 1 applied to `V-REQ-13`: the bytes cross the API and are Layer 1's to carry; what they mean is not.

### Scenarios

- **S-ATD-014** — Given schema bytes that are non-empty but describe nothing a meta-schema would accept, when a declaration is constructed, then construction succeeds and the bytes read back unchanged.
- **S-ATD-015** — Given two declarations whose schema bytes differ only in whitespace, when both are constructed, then both succeed and each reads back its own bytes exactly.

---

## R-ATD-005 — A tool set preserves the caller's order, deterministically

A tool set MUST expose its declarations in **the order the caller supplied**, and that order MUST be identical on every read (`V-REQ-14`). No operation of the tool set may allow an unordered iteration to determine an observable result.

The requirement is stated as the caller's order rather than merely a stable order because a stable-but-arbitrary order satisfies self-comparison and still breaks the property `V-REQ-25` needs: a caller that builds the same set twice must obtain the same prefix.

### Scenarios

- **S-ATD-016** — Given a tool set built from a sequence of declarations, when it is read twice, then both reads yield the same sequence.
- **S-ATD-017** — Given a tool set large enough that an unordered iteration would be observable, when it is read many times, then every read yields exactly the caller's order.
- **S-ATD-018** — Given two tool sets built from the same declarations in the same order, when each is read, then the two sequences are equal element for element.
- **S-ATD-019** — Given a consumer that mutates the sequence a read returned, when the set is read again, then the set is unchanged.
- **S-ATD-020** — Given a caller that mutates the arguments it passed to the constructor, when the set is read, then the set is unchanged.

---

## R-ATD-006 — An empty tool set is legal

A tool set containing no declarations MUST be constructible and MUST NOT be a caller-contract failure. A request offering no tools is the common case, not an error. The zero value of the tool-set type MUST be that same empty set, so a set nobody built behaves as a set with nothing in it rather than as an unconstructed value.

### Scenarios

- **S-ATD-021** — Given no declarations, when a tool set is constructed, then construction succeeds AND the set reads back as empty.
- **S-ATD-022** — Given the zero value of the tool-set type, when it is read, then it yields an empty sequence and reports a size of zero.

---

## R-ATD-007 — Duplicate tool names in one set are a caller-contract failure

WHEN a tool set is constructed from declarations of which two or more carry the same name, THEN construction MUST fail with a caller-contract failure through AI-04's taxonomy, positioned at the **second** occurrence by index. Which occurrence is reported MUST be identical across runs.

Positioning by index rather than by name is `V-FAIL-03`'s posture as AI-04 landed it: `V-REQ-14` makes the set ordered and deterministically iterable, so an index identifies one declaration unambiguously, and the caller — which holds the set — resolves it to a name in one step without a caller value reaching a message that will be logged.

### Scenarios

- **S-ATD-023** — Given two declarations with the same name, when a tool set is constructed from them, then construction fails with a caller-contract failure of the taxonomy.
- **S-ATD-024** — Given that failure, when its position is read, then it names the tool-set field indexed by the position of the second occurrence.
- **S-ATD-025** — Given a set with several duplicate pairs, when it is constructed repeatedly, then the same occurrence is reported every time.
- **S-ATD-026** — Given a declaration that never passed construction, when it is offered to a tool set, then the set rejects it rather than carrying an unconstructed value to the wire.

---

## R-ATD-008 — The tool-choice vocabulary is closed and enumerable

Layer 1 MUST expose a closed tool-choice vocabulary with exactly four members: *automatic*, *none*, *required*, and *specific*. Its members MUST be enumerable in a stable declaration order, the enumeration MUST NOT be mutable by a consumer, and a value that is not a member MUST NOT be usable anywhere a tool choice is accepted. Each member MUST render to a stable, lowercase, non-empty string that parses back to the same member and only to it.

The zero value of the vocabulary type MUST NOT be a member, so a field nobody set is rejected exactly like a wild value.

### Scenarios

- **S-ATD-027** — Given the tool-choice vocabulary, when a consumer enumerates it, then the four members come back in a stable order that is identical on every call.
- **S-ATD-028** — Given a consumer that mutates the enumeration it received, when it enumerates again, then the vocabulary is unchanged.
- **S-ATD-029** — Given each member in turn, when its rendering is parsed, then it parses back to the same member.
- **S-ATD-030** — Given a rendering that is not exactly a registered one — a case variant, a padded form, or the diagnostic rendering of a non-member — when it is parsed, then parsing fails and yields no member.
- **S-ATD-031** — Given the empty string, when it is parsed as a member, then it fails with the *empty* rule class rather than the *not in vocabulary* one — different facts, different fixes.
- **S-ATD-032** — Given the zero value of the vocabulary type, when it is offered anywhere a tool choice is accepted, then it is rejected.

---

## R-ATD-009 — Each member is constructible, and the payload-carrying member carries its payload

Every member of the tool-choice vocabulary MUST be constructible as a tool-choice value. Three members carry no payload. The fourth — *specific* — names one tool, and that name MUST read back from the constructed value exactly as supplied.

A constructor for a payload-free member MUST reject the payload-carrying member, and it MUST do so through the taxonomy rather than by producing a value with no payload. A tool name supplied to the payload-carrying constructor MUST be checked against **the same name rule** a declaration's name is checked against, so a choice naming a syntactically impossible tool fails at construction rather than at cross-validation.

### Scenarios

- **S-ATD-033** — Given each payload-free member in turn, when a tool choice is constructed for it, then construction succeeds AND the member reads back equal.
- **S-ATD-034** — Given a tool name, when a tool choice is constructed for the payload-carrying member, then construction succeeds AND both the member and the name read back equal.
- **S-ATD-035** — Given a payload-free member, when its constructed value is asked for a name, then it reports that it carries none — distinguishable from carrying an empty one.
- **S-ATD-036** — Given the payload-carrying member, when it is offered to the payload-free constructor, then construction fails through the taxonomy with the *empty* rule class positioned at the name.
- **S-ATD-037** — Given a name that violates the name rule, when a tool choice is constructed for the payload-carrying member, then construction fails with the same rule class a declaration's name would have failed with.
- **S-ATD-038** — Given the zero value of the tool-choice type, when it is validated, then it fails as a non-member rather than defaulting to any behavior.

---

## R-ATD-010 — A tool choice is cross-validated against the declared set, in a documented order

WHEN a tool choice is validated against a tool set, THEN the rules MUST be evaluated in a documented order and the first failure reported, identically across runs:

1. The member is one of the vocabulary — else *value is outside a closed vocabulary*.
2. The member is anything other than *none* AND the set is empty — else *a required value is empty*, positioned at the tool set.
3. The member is *specific* AND the set declares no tool of that name — else *value names something the request does not declare*, positioned at the choice's name.

Rule 2 precedes rule 3 deliberately: with an empty set, "you have declared no tools at all" is the more fundamental fact and the more actionable message than "the tool you named is not declared".

Rule 2 exists because it is the combination every provider rejects, and catching it before any I/O is the point of the seam. *None* against an empty set MUST succeed — expressing "do not call a tool" when there are no tools is coherent, not contradictory.

### Scenarios

- **S-ATD-039** — Given a tool set declaring a tool and a *specific* choice naming that tool, when the choice is validated against the set, then validation succeeds.
- **S-ATD-040** — Given a non-empty tool set and a *specific* choice naming a tool the set does not declare, when the choice is validated, then validation fails with the unresolved-reference sentinel AND the position names the choice's name.
- **S-ATD-041** — Given an empty tool set and a choice of *automatic*, *required* or *specific*, when the choice is validated, then validation fails with the empty sentinel positioned at the tool set.
- **S-ATD-042** — Given an empty tool set and a choice of *none*, when the choice is validated, then validation succeeds.
- **S-ATD-043** — Given a non-empty tool set and a choice of *none*, *automatic* or *required*, when the choice is validated, then validation succeeds without consulting any name.
- **S-ATD-044** — Given an empty tool set and a *specific* choice naming an undeclared tool — two rules violated at once — when the choice is validated, then the failure reported is the first in the documented order, identically on every run.
- **S-ATD-045** — Given any cross-validation failure, when it is rendered, then the rendering carries none of the offered tool name.

---

## R-ATD-011 — Tool-choice registration is exhaustive *(pin)*

Every declared member of the tool-choice vocabulary MUST have a registry entry, MUST render and parse back through it, and MUST be constructible by the constructor its declared arity selects. A member declared without a registry entry MUST fail this requirement.

Green from birth and exempt from red-first per doc 0002's leaf anatomy. It is protective rather than diagnostic: the property it protects is that a later milestone cannot add a member without also making it renderable, parseable and constructible.

### Scenarios

- **S-ATD-046** — Given the declared tool-choice constant space, when it is enumerated, then every declared member has a registry entry with a non-empty, lowercase rendering.
- **S-ATD-047** — Given each declared member, when its rendering is parsed, then it parses back to the same member.
- **S-ATD-048** — Given each declared member, when it is offered to the constructor its declared arity selects, then construction succeeds — and when it is offered to the other constructor, it does not.
- **S-ATD-049** — Given a member declared without a registry entry, when the pin runs, then the pin fails.

---

## R-ATD-012 — This contract carries no behavior beyond transport *(constrains the exported surface)*

Layer 1's tool contract MUST NOT expose any means of executing a tool, resolving a tool name to an implementation, deciding whether a call is permitted, or reporting where a tool set came from (`V-OUT-04`, `V-OUT-05`, `V-OUT-17`). It MUST NOT define an error type or sentinel of its own, and MUST NOT define any content part, message or request type.

Constrains the shape of the exported surface rather than its behavior; verified by inspection in the archived `tasks.md` verification pass.

### Scenarios

- **S-ATD-050** — Given this contract's exported surface, when a reviewer inspects it, then it exposes no invocation, resolution, permission or provenance operation.
- **S-ATD-051** — Given this contract's exported surface, when a reviewer inspects it, then it declares no error variable and no error type, and every failure it produces matches one of AI-04's landed rule classes.
- **S-ATD-052** — Given this contract's files, when a reviewer inspects them, then they define no content part, no message type and no request type.

---

## Non-functional requirements

### NFR-ATD-A — Dependency purity

The implementation MUST use only the Go standard library, and `go.mod` MUST continue to carry zero requires. Both AI-00 import-boundary guards MUST continue to pass unchanged.

### NFR-ATD-B — One failure vocabulary

No new sentinel, error variable or error type. Every failure reports through AI-04's `Invalid`, carries one of its landed rule classes, and is matchable through wrapping by the same single extraction target a consumer already writes.

### NFR-ATD-C — No I/O, no clock, no randomness

Nothing in this contract performs I/O, reads a clock, or consumes randomness. Validation is decidable from the values alone, which is `V-FAIL-01`'s own criterion for a caller-contract failure.

### NFR-ATD-D — Evidence

Every test-list item is taken red → green → refactored in order, with both outputs recorded in `tasks.md`. The pin is exempt from red-first and is shown to bite. The milestone closes only on recorded green `make test` in `backend/agent/`.

### NFR-ATD-E — Reviewability

The contract fits three production files split on the node boundary. The name rule, the documented rule orders, and the extension to the closed-vocabulary pattern are each stated in the GoDoc of the declaration that enforces them, not only in this contract's markdown.

---

## Acceptance criteria

1. `R-ATD-001` … `R-ATD-011` each have at least one passing test, and `R-ATD-012` has a recorded inspection.
2. `make test` in `backend/agent/` is green with `-race`; `make lint` is clean.
3. Both import guards pass; `go.mod` carries zero requires.
4. Schema-byte equality is asserted against input a marshal round trip would rewrite.
5. Order determinism is asserted over a set large enough that an unordered iteration would be observable, across many reads, against the caller's order.
6. The exhaustiveness pin is shown to bite, and the failure output is recorded.
