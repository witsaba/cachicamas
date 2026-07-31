# Spec — tool calls and tool results

> **Change**: `cachicamas-ai-tool-messages`
> **Milestone**: AI-09 · **Nodes**: AI-09.1 `[leaf]`, AI-09.2 `[leaf]`, AI-09.3 `[leaf]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-tool-messages/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-ATM-0NN` · **Scenario IDs**: `S-ATM-0NN`
> **Binding input**: [the register](../../../../specs/ai-contract-vocabulary/spec.md) — `V-REQ-16`, `V-REQ-17`, `V-REQ-18`, `V-STR-21`, `V-REQ-04` … `V-REQ-07`, `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-13`, `V-OUT-02`, `V-OUT-04` · [AI-06's `decision.md`](../../../cachicamas-ai-content-parts/decision.md) — inherited whole · [AI-04's spec](../../../cachicamas-ai-validation-errors/specs/ai-validation-errors/spec.md) · [`design.md`](../../design.md)

---

## Purpose

Constrain the runtime behavior of the two content-part kinds that cross the model API in both
directions: the model's request to invoke a tool, and the answer that comes back to it. Every
scenario below is verifiable by a test that runs.

## Relationship to AI-06's spec

This spec **adds two kinds** under `R-ACP-001` … `R-ACP-012` and restates none of them. Where AI-06
already constrains a property — one part type, derived kind, `(T, bool)` accessor, rules on the
payload, exhaustive registration — the requirement below cites it and adds only what is specific to
a tool call or a tool result. `R-ACP-004`'s mechanical assertion over the declared kind set is what
proves the two new kinds are fully registered; this spec adds no second guard.

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A tool call** — `V-REQ-16`. The content-part kind carrying a model's request to invoke a tool: its
  identity, the tool name, and exact argument bytes. An *intent to invoke*; Layer 1 never acts on it.
- **Argument bytes** — `V-REQ-17`. The call's arguments, carried exactly as received or supplied.
- **A tool result** — `V-REQ-18`. The kind carrying the answer to a tool call: its correlation to the
  originating call, its content, and an indication of whether the tool reported failure.
- **A call ordinal** — `V-STR-21`. The observable position of a tool call among the calls of one
  response.
- **A content sequence** — the ordered content of one message, as `V-REQ-02` defines it.
- **A caller-contract failure**, **a validation sentinel**, **a position**, **the rule order** —
  `V-FAIL-01` … `V-FAIL-04`, as landed by AI-04.

---

## R-ATM-001 — A tool call is a content part carrying identity, name and argument bytes

Layer 1 MUST represent a model's request to invoke a tool as a content part of a registered kind,
constructed only through a function of the owning package, carrying exactly three values: the call's
identity, the tool's name, and the argument bytes.

A consumer in another package MUST be able to discover the kind and read all three back through the
exported surface alone.

The identity MUST be an opaque value a caller can **supply**, because an adapter for a provider that
assigns no tool-call identifiers mints synthetic ones (leakage register row 7) and a session one
layer up reloads them. Layer 1 MUST NOT mint, rewrite, parse or constrain the shape of the identity
beyond requiring that it is not empty.

### Scenarios

- **S-ATM-001** — Given an external package that constructs a tool call and places it in a message,
  when it reads the message's content back, discovers the part's kind and calls the matching
  accessor, then it obtains the identity, the name and the argument bytes it supplied.
- **S-ATM-002** — Given a tool call constructed with an identity of a shape no provider would emit —
  an adapter-minted synthetic identifier — when the identity is read back, then it is byte-identical
  to the supplied string, with no prefix, suffix, normalization or re-encoding.
- **S-ATM-003** — Given the tool-call accessor called on a part of a different kind, or on a part
  that was never constructed, when the caller inspects the second result, then it reports that the
  part carries no tool call, and the first result is the zero value rather than data.

---

## R-ATM-002 — Argument bytes are carried byte-exactly

WHEN argument bytes are supplied at construction and read back, THEN the bytes read MUST be equal,
byte for byte, to the bytes supplied. Re-marshalling, key reordering, whitespace normalization,
escaping, case folding and any other canonicalization of supplied bytes are forbidden (`V-REQ-17`).

The read MUST NOT alias the caller's buffer, and the construction MUST NOT alias it either: a caller
that mutates the slice it passed, and a consumer that mutates the slice it received, MUST both leave
every other holder's view unchanged.

### Scenarios

- **S-ATM-004** — Given argument bytes whose object keys are in an order a canonicalizing encoder
  would change and whose interior whitespace such an encoder would remove, when the bytes are read
  back from a message in another package, then they are byte-equal to the bytes supplied.
- **S-ATM-005** — Given a constructed tool call, when the caller mutates the slice it passed to the
  constructor, then the call's argument bytes are unchanged on re-read.
- **S-ATM-006** — Given a constructed tool call, when a consumer mutates the slice returned to it,
  then a second read returns the original bytes, and a second consumer is unaffected.

---

## R-ATM-003 — Absent arguments are legal and have exactly one canonical form

A tool call with no arguments MUST be constructible: a tool that takes no arguments is routine.

WHEN no argument bytes are supplied — an absent, nil or zero-length value — THEN the call MUST carry
one canonical empty form, and every such call MUST carry the **same** form. The canonical form MUST
be syntactically well-formed for the documented argument encoding, so that a consumer decoding a
call's arguments never meets an input that parses only by special case.

This normalization applies to the **absent** case alone. It MUST NOT be read as licence to normalize
supplied bytes, which `R-ATM-002` forbids.

### Scenarios

- **S-ATM-007** — Given a tool call constructed with nil arguments and a second constructed with a
  zero-length slice, when both are read back, then their argument bytes are equal to each other and
  are the documented canonical empty form.
- **S-ATM-008** — Given a tool call constructed with no arguments, when its argument bytes are
  decoded by a consumer using the documented encoding, then decoding succeeds.
- **S-ATM-009** — Given a tool call constructed with argument bytes that are well-formed and
  non-empty, when they are read back, then they are the supplied bytes and **not** the canonical
  empty form, even when the supplied bytes are semantically an empty argument object written
  differently.

---

## R-ATM-004 — Tool-call construction rules fail with the landed sentinels, in a documented order

Construction MUST fail with a caller-contract failure carrying one of AI-04's landed rule classes,
at a position, when any rule is broken. The rules and their order are part of the contract:

1. the identity is not empty, else `ErrEmpty` at the identity's position;
2. the tool name is not empty, else `ErrEmpty` at the name's position;
3. the argument bytes are syntactically well-formed for the documented encoding, else `ErrMalformed`
   at the arguments' position.

Bytes that are well-formed but do not satisfy the declared tool's schema MUST NOT fail construction:
this layer does not validate against schemas (`V-REQ-13`).

On failure the constructor MUST return the zero content part, so a caller that ignored the error
cannot mistake the result for a constructed one.

The failure message MUST NOT reproduce any byte of the identity, the name or the argument bytes
(`V-FAIL-13`).

### Scenarios

- **S-ATM-010** — Given an empty identity, when a tool call is constructed, then construction fails,
  the failure matches the empty-value rule class, and the position names the identity.
- **S-ATM-011** — Given an empty tool name, when a tool call is constructed, then construction fails
  with the empty-value rule class at the name's position.
- **S-ATM-012** — Given argument bytes that are not well-formed for the documented encoding —
  truncated, whitespace-only, or a bare fragment — when a tool call is constructed, then construction
  fails with the malformed-value rule class at the arguments' position.
- **S-ATM-013** — Given inputs breaking more than one rule at once, when a tool call is constructed
  repeatedly, then the same rule is reported every time, and it is the first in the documented order.
- **S-ATM-014** — Given any failing construction, when the rendered failure is inspected, then it
  contains no substring of the identity, the name or the argument bytes it was given.
- **S-ATM-015** — Given argument bytes that are well-formed but describe arguments no declared tool
  would accept, when a tool call is constructed, then construction succeeds.

---

## R-ATM-005 — A tool call that skipped its rules is rejected at the message boundary

WHEN a tool-call part reaches message construction carrying a payload its own rules reject, THEN
message construction MUST fail with a registered rule class at the offending element's position.

The constructor and the message boundary MUST run **one** rule set from two entry points, per AI-06
`decision.md` § 7.2.

### Scenarios

- **S-ATM-016** — Given a tool-call part assembled inside the owning package without going through
  the constructor and holding a value its rules reject, when it is offered as message content, then
  message construction fails with a registered rule class.
- **S-ATM-017** — Given a properly constructed tool-call part, when it is offered as message content,
  then message construction succeeds.

---

## R-ATM-006 — The call ordinal is the call's position among the calls of a content sequence

Layer 1 MUST make a tool call's ordinal observable from another package (`V-STR-21`). The ordinal
MUST be **derived from the call's position** among the tool calls of a content sequence and MUST NOT
be stored on the call at construction time, so that no stored value can disagree with the position a
message actually holds.

WHEN a content sequence is read for its tool calls, THEN the result MUST list them in content order,
skipping every part of another kind, and the position of a call in that result IS its ordinal.

The derivation MUST be total: a sequence with no tool calls, an empty sequence, and a sequence
containing parts that were never constructed MUST all be answered without failure.

### Scenarios

- **S-ATM-018** — Given a message whose content interleaves several tool calls with parts of other
  kinds, when its tool calls are read, then they appear in content order, other kinds are absent, and
  each call's ordinal is its index in that result.
- **S-ATM-019** — Given that same message, when its tool calls are read repeatedly, then every read
  yields the same calls in the same order with the same ordinals.
- **S-ATM-020** — Given that same message copied by value, when the copy's tool calls are read, then
  the ordinals are identical to the original's.
- **S-ATM-021** — Given a consumer that mutates the sequence returned to it, when the tool calls are
  read again from the message, then the ordinals are unchanged.
- **S-ATM-022** — Given a content sequence that is empty, that holds no tool call, or that holds a
  part which was never constructed, when its tool calls are read, then the result is empty or omits
  the unconstructed element, and no failure and no panic occurs.

---

## R-ATM-007 — A tool result is a content part carrying correlation, content and a failure indication

Layer 1 MUST represent the answer to a tool call as a content part of a registered kind, constructed
only through a function of the owning package, carrying its correlation to the originating call, its
content, and whether the tool reported failure (`V-REQ-18`).

A consumer in another package MUST be able to discover the kind and read all three back through the
exported surface alone.

### Scenarios

- **S-ATM-023** — Given an external package that constructs a tool result and places it in a message,
  when it reads the message's content back and calls the matching accessor, then it obtains the
  correlation, the content and the failure indication it constructed the result with.
- **S-ATM-024** — Given the tool-result accessor called on a part of a different kind, or on a part
  that was never constructed, when the caller inspects the second result, then it reports that the
  part carries no tool result.

---

## R-ATM-008 — The correlation round-trips exactly, including a synthetically minted identity

WHEN a correlation identity is supplied and read back, THEN it MUST be byte-identical to the value
supplied. Layer 1 MUST NOT mint, rewrite, parse or constrain its shape beyond requiring that it is
not empty, because an adapter for a provider that assigns no identifiers mints synthetic ones
(leakage register row 7) and a session reload one layer up depends on the mapping surviving.

A correlation MUST match the identity of the tool call it answers under ordinary string equality, so
that no consumer needs a comparison function of this package's to pair them.

### Scenarios

- **S-ATM-025** — Given a tool call constructed with an adapter-minted synthetic identity and a tool
  result constructed to correlate to it, when both are read back out of messages, then the result's
  correlation equals the call's identity under ordinary equality.
- **S-ATM-026** — Given correlation identities of several shapes — provider-assigned, synthetic,
  containing punctuation, containing non-ASCII bytes — when each round-trips, then each reads back
  byte-identical.
- **S-ATM-027** — Given an empty correlation, when a tool result is constructed, then construction
  fails with the empty-value rule class at the correlation's position, and the zero content part is
  returned.

---

## R-ATM-009 — A tool that reported failure is distinguishable, and is not routed through the failure taxonomy

Layer 1 MUST let a consumer distinguish a tool result that reports the tool failed from one that
reports it succeeded, by reading the constructed part.

Constructing a result that reports failure MUST succeed. It MUST NOT produce a caller-contract
failure (`V-FAIL-01`) and MUST NOT be represented as a provider or transport failure (`V-FAIL-05`):
a failing tool is ordinary content the model must see (`V-REQ-18`).

Two results with the same correlation and the same content but different failure indications MUST be
distinguishable.

### Scenarios

- **S-ATM-028** — Given a tool result constructed as a failure, when it is read back from a message,
  then it reports failure, its content is the content it was given, and no error was produced at
  construction.
- **S-ATM-029** — Given a tool result constructed as a success, when it is read back, then it reports
  no failure.
- **S-ATM-030** — Given two results with identical correlation and identical content, one success and
  one failure, when both are read back, then they are distinguishable by the failure indication
  alone.

---

## R-ATM-010 — Both new kinds are registered, and their exported payload types are opaque

Each new kind MUST complete AI-06 `decision.md` § 8's five-step procedure: a declared constant with
the enumeration bound moved past it, an unexported-method payload, an exported constructor whose
rules are the payload's, an exported `(T, bool)` accessor, and an entry in both the registration
table and the documented kind list.

Each exported payload type MUST be an opaque value type: no exported field, no exported way to
assemble one, and every instance obtained from a constructor of the owning package (AI-06
`decision.md` § 6.3).

Both payload types MUST remain comparable with `==`, because AI-06 documents part equality as
defined.

### Scenarios

- **S-ATM-031** — Given the declared content-part kind set, when the mechanical registration
  assertion runs, then both new kinds have a constructor with rules, a payload accessor and a
  validation path, and the declared constants, the enumeration and the registration table agree.
- **S-ATM-032** — Given the documented kind list, when it is compared against the registration table,
  then both new kinds appear in both.
- **S-ATM-033** — Given each exported payload type, when its exported surface is inspected, then it
  declares no exported field.
- **S-ATM-034** — Given two content parts of the same new kind constructed from identical inputs,
  when they are compared with `==`, then the comparison completes and reports them equal.

---

## R-ATM-011 — Neither kind renders its payload

WHEN a tool call, a tool result, or a content part carrying either is rendered by any formatting verb
of the host language, THEN the rendering MUST NOT reproduce the argument bytes, the result content,
the identity or the tool name (`V-FAIL-13`).

A consumer that wants a payload calls the accessor, which is what an accessor is for.

### Scenarios

- **S-ATM-035** — Given a tool call carrying a recognizable secret in its argument bytes, when it is
  rendered by the default, the string and the Go-syntax verbs, then no rendering contains any
  substring of the argument bytes or of the identity.
- **S-ATM-036** — Given a tool result carrying a recognizable secret in its content, when it is
  rendered by the same three verbs, then no rendering contains any substring of the content.
- **S-ATM-037** — Given content parts carrying each new kind, when each is rendered, then the
  rendering names the kind and nothing else.
