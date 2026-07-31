# Spec — content parts: readable and sealed

> **Change**: `cachicamas-ai-content-parts`
> **Milestone**: AI-06 · **Nodes**: AI-06.1 `[decision]`, AI-06.2 `[leaf]`, AI-06.3 `[leaf]`, AI-06.4 `[guard]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-content-parts/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-ACP-0NN` · **Scenario IDs**: `S-ACP-0NN`
> **Binding input**: [the register](../../../../specs/ai-contract-vocabulary/spec.md) — `V-REQ-04` … `V-REQ-08`, `V-REQ-02`, `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-13` · [AI-04's spec](../../../cachicamas-ai-validation-errors/specs/ai-validation-errors/spec.md) · [AI-05's spec](../../../cachicamas-ai-message-roles/specs/ai-message-roles/spec.md) · [`decision.md`](../../decision.md)

---

## Purpose

Constrain the runtime behavior of Layer 1's content part: what it is, how a consumer in another package reads its payload, why nothing that skipped construction can be valid, and how the kind set stays exhaustively registered. Every scenario below is verifiable by a test that runs, with two marked exceptions that are verifiable by a mechanical scan.

## Relationship to AI-05's spec

`R-AMR-009` — *"the content seam decides nothing about a content part"* — was written by AI-05 as a deliberate placeholder, and its second scenario pinned that `NewMessage` rejects no content element so that AI-06.3 item 1 could fail before it passed. **This spec supersedes that scenario.** `R-ACP-006` is its replacement, and AI-05's requirement is otherwise unaffected: the seam still names the position a part occupies, it is simply no longer an interface. AI-05's own artifacts are not edited; the supersession is recorded here and in `design.md` § 3.

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A content part** — `V-REQ-04`. One element of a message's ordered content, carrying exactly one kind of payload. Both of its properties are constitutive: readable from outside the owning package, and invalid if it skipped construction.
- **A content-part kind** — `V-REQ-05`. The discriminator naming which payload a part carries. The set is closed and exhaustively registered.
- **Content-part readability** — `V-REQ-06`. **Content-part sealing** — `V-REQ-07`.
- **Text content** — `V-REQ-08`. The kind carrying model-visible natural-language text.
- **A message**, **content** — `V-REQ-02`, as landed by AI-05.
- **A caller-contract failure**, **a validation sentinel**, **a position**, **the rule order** — `V-FAIL-01` … `V-FAIL-04`, as landed by AI-04.
- **A registered kind** — a kind that has a table entry supplying its rendering and its validity.

---

## R-ACP-001 — A content part is one opaque value type, and construction is its only door

Layer 1 MUST represent every content-part kind as values of **one** exported type. That type MUST NOT expose any field to another package, and it MUST NOT be an interface. A value of it MUST be obtainable from another package only by calling a construction function of this package.

Later kinds — reasoning, tool call, tool result — MUST use this same type. A second part type, a parallel part interface, or a per-kind exported value type is a violation of this requirement.

### Scenarios

- **S-ACP-001** — Given the content-part type, when its exported surface is inspected, then it declares no exported field, so no consumer in another package can assemble one by literal.
- **S-ACP-002** — Given a message's content, when the element type is inspected, then it is that one concrete type rather than an interface, so no other type can be offered in its place.

---

## R-ACP-002 — A part's payload is readable from another package

WHEN a consumer in a package other than the one that owns the contract holds a constructed message, THEN it MUST be able to discover each part's kind and obtain that part's payload through the exported surface alone, with no access to unexported state and **no type switch over unexported types**.

The payload MUST be returned **byte-equal** to the value supplied at construction. No normalization, trimming, re-encoding, escaping or case folding is permitted on the read path or the construction path.

### Scenarios

- **S-ACP-003** — Given an external package that constructs a text part and places it in a message, when it reads the message's content back and obtains the payload, then the payload is byte-equal to the string supplied.
- **S-ACP-004** — Given that same external package, when it discovers the part's kind, then the kind is an exported comparable value it can compare against an exported constant.
- **S-ACP-005** — Given a payload accessor called on a part of a **different** kind, when the caller inspects the second result, then it reports that the part does not carry that payload, and the first result is the zero value rather than data.

---

## R-ACP-003 — The kind is derived from the payload, so the two cannot disagree

A content part MUST NOT store its kind independently of its payload. The kind MUST be computed from the payload the part carries.

WHEN a part reports a kind, THEN the accessor for that kind MUST yield a payload, and every accessor for another kind MUST report that it does not.

### Scenarios

- **S-ACP-006** — Given a constructed text part, when its kind is read and the matching accessor is called, then the accessor reports success and yields a non-empty payload — never an empty or zero payload.
- **S-ACP-007** — Given every registered kind in turn, when a part of that kind is built through its constructor, then the kind it reports is the kind that was built.
- **S-ACP-008** — Given a part that carries no payload, when its kind is read, then the kind is not a member of the registered kind set.

---

## R-ACP-004 — The kind vocabulary is closed, enumerable and exhaustively registered

Layer 1 MUST expose the content-part kinds as a closed vocabulary, enumerable in a stable declaration order, and the enumeration MUST NOT be mutable by a consumer. Every declared kind MUST have a table entry supplying its rendering and its validity. A kind declared without one MUST fail a mechanical assertion over the vocabulary, and that assertion MUST enumerate the **declared** kinds rather than the table.

A kind's rendering MUST be stable and lowercase. A value that is not a declared kind MUST render in a form that identifies it for a diagnostic reader and MUST NOT be accepted anywhere a kind is required.

### Scenarios

- **S-ACP-009** — Given the kind vocabulary, when a consumer enumerates it, then the members come back in a stable order that is identical on every call.
- **S-ACP-010** — Given a consumer that mutates the enumeration it received, when it enumerates again, then the vocabulary is unchanged.
- **S-ACP-011** — Given the zero value of the kind type, when it is rendered, then the rendering marks it as unset AND it is not a member of the enumeration.
- **S-ACP-012** — Given a scratch kind declared without a table entry, when the assertion runs, then it fails. *(Bite proof: recorded, then the scratch kind is removed.)*

---

## R-ACP-005 — Text content has documented construction rules that report through the taxonomy

WHEN a text part is constructed, THEN the following rules MUST be checked in this order, and the first violation MUST be reported as a caller-contract failure positioned at the text:

1. the text MUST contain at least one non-whitespace character, else the rule class *required value is empty*;
2. the text MUST NOT exceed the documented byte bound, else the rule class *value is outside a documented bound*.

The bound MUST be an exported constant, so a caller can check before constructing. A failed construction MUST return a part that is not valid, so a caller that ignored the failure cannot mistake it for a constructed one.

### Scenarios

- **S-ACP-013** — Given an empty string, when a text part is constructed from it, then construction fails with the *required value is empty* sentinel positioned at the text.
- **S-ACP-014** — Given a string of only whitespace — spaces, tabs, newlines, or non-ASCII whitespace — when a text part is constructed from it, then construction fails the same way: text carrying no model-visible characters is empty.
- **S-ACP-015** — Given a string one byte longer than the documented bound, when a text part is constructed from it, then construction fails with the *value is outside a documented bound* sentinel positioned at the text; given a string exactly at the bound, construction succeeds.
- **S-ACP-016** — Given a failed construction, when the returned part is offered to a message, then the message rejects it — a failed construction yields nothing usable.
- **S-ACP-017** — Given a construction failure, when its rendered message is inspected, then the text that was offered appears nowhere in it.

---

## R-ACP-006 — A value that skipped construction cannot reach a message

WHEN a content element that did not pass its construction rules is offered to message construction, THEN construction MUST fail with a caller-contract failure whose rule class is *value is outside a closed vocabulary*, positioned at the offending element **by ordinal**. This MUST hold for the zero value of the part type, for a part promoted out of a type that embeds it, and for any part whose payload's kind is not registered.

The rule MUST be applied to every element in index order, and the first failing element MUST be the one reported.

### Scenarios

- **S-ACP-018** — Given the zero value of the part type, when it is offered as message content, then construction fails with the closed-vocabulary sentinel AND the position names the content element by ordinal.
- **S-ACP-019** — Given a message content sequence whose third element is unconstructed and whose others are valid, when it is offered, then construction fails and the position names index 2.
- **S-ACP-020** — Given a value of a type that **embeds** the part type, when its promoted part is offered as message content, then construction fails identically — embedding yields a zero part, not a valid one.
- **S-ACP-021** — Given a message constructed only from valid parts, when its content is read back, then every element is valid, so a consumer never has to defend against an unconstructed part it received.

---

## R-ACP-007 — The part contract cannot be satisfied from another package

Another package MUST NOT be able to supply a value that a message accepts as content without calling a construction function of this package. The prevention MUST be structural rather than advisory: it MUST be a compile-time failure, verified mechanically, not a review convention.

### Scenarios

- **S-ACP-022** — Given a program in another package that declares a type embedding the part type and offers it as message content, when the program is compiled, then compilation fails.
- **S-ACP-023** — Given a program in another package that writes a composite literal of the part type with any field set, whether named or positional, when the program is compiled, then compilation fails.
- **S-ACP-024** — Given an otherwise identical program that constructs its part through the exported constructor, when it is compiled, then compilation succeeds — so the failures above are attributable to the bypass and not to the setup.

---

## R-ACP-008 — Valid text survives construction unaltered

WHEN a text part is constructed from a string that satisfies its rules, THEN the payload read back MUST be byte-identical to the string supplied, for every byte class the string may contain.

This MUST hold for embedded newlines and carriage returns, leading and trailing whitespace around non-whitespace content, high Unicode including astral-plane runes and combining marks, right-to-left text, content that looks like markup or a template, and strings containing the package's own diagnostic punctuation.

### Scenarios

- **S-ACP-025** — Given text containing embedded newlines and surrounding whitespace, when the part is constructed and read back, then the payload is byte-identical, including the surrounding whitespace — the emptiness rule inspects the text, it does not rewrite it.
- **S-ACP-026** — Given text containing astral-plane runes, combining marks, right-to-left runs and invalid UTF-8 byte sequences, when the part is constructed and read back, then the payload is byte-identical.
- **S-ACP-027** — Given text that looks like markup, a template, or a rendered validation failure, when the part is constructed and read back, then the payload is byte-identical and nothing in the package has interpreted it.

---

## R-ACP-009 — The element rule is reusable at request scope *(reuse point)*

The rule that rejects an unconstructed content element MUST be expressed once, in a form that accepts a position prefix, so that a later contract validating a request applies the identical rule at a deeper position rather than re-implementing it.

WHEN the rule is applied with a prefix, THEN the reported position MUST be the prefix followed by the element's own position, and the rule class MUST be unchanged by the prefix.

### Scenarios

- **S-ACP-028** — Given an unconstructed element and no prefix, when the rule is applied, then the position names the content element by ordinal.
- **S-ACP-029** — Given the same element and a request-shaped prefix naming a message by ordinal, when the rule is applied, then the position is the prefix followed by the element's own position, and the rule class is identical to the unprefixed case.
- **S-ACP-030** — Given message construction, when it validates content, then it applies that same rule rather than a second implementation of it.

---

## R-ACP-010 — Every registered kind has a constructor with rules, a payload accessor, and a validation path *(guard)*

Every registered content-part kind MUST have all three of:

1. a **construction function** whose rules reject at least one input, reporting a registered rule class;
2. a **payload accessor** that yields the payload for a part of that kind and reports failure for a part of another kind;
3. a **validation path** that rejects a part of that kind which skipped its construction rules, at the message boundary.

A mechanical assertion MUST enumerate the declared kinds and MUST fail when a kind is missing any of the three.

### Scenarios

- **S-ACP-031** — Given every declared kind, when the assertion runs, then each has all three, and each is exercised rather than merely present.
- **S-ACP-032** — Given a scratch kind registered with only two of the three, when the assertion runs, then it fails and names the missing one. *(Bite proof: recorded, then the scratch kind is removed.)*

---

## R-ACP-011 — The documented kind list matches the registration table *(guard)*

The package documentation of the kind vocabulary MUST list the registered kinds, and a mechanical scan MUST compare that list to the registration table. A kind present in one and absent from the other MUST fail the scan.

### Scenarios

- **S-ACP-033** — Given the documented list and the registration table, when the scan compares them, then they name the same kinds.
- **S-ACP-034** — Given a documented list that names a kind the table does not, or omits one the table has, when the scan runs, then it fails. *(Bite proof: recorded, then the drift is reverted.)*

---

## R-ACP-012 — A part's diagnostic rendering carries no payload

A content part MUST render, for a diagnostic reader, in a form that names its kind and **never** reproduces its payload. This extends `V-FAIL-13`'s redaction posture from validation failures to the first payload-carrying value type in the package.

### Scenarios

- **S-ACP-035** — Given a text part carrying a distinctive string, when the part is rendered by any standard formatting verb, then the distinctive string appears nowhere in the rendering.
- **S-ACP-036** — Given a part that carries no payload, when it is rendered, then the rendering marks it as unset rather than naming a kind.

---

## Out of scope for this spec

| Behavior | Owner |
| --- | --- |
| Reasoning content, its state vocabulary, its round-trip token | AI-07 |
| Tool call and tool result payloads | AI-09 |
| Whether a given role may carry a given kind | AI-10.3 |
| Where a request runs its single validation pass | AI-10.4, `V-REQ-22` |
| Cache-boundary markers | AI-11 |
| Any wire encoding | AI-24 onward |
| Image and audio kinds | deliberately producerless — `V-PRV-07` |
