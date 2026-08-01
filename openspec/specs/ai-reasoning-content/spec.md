# Spec — reasoning content and its round-trip token

> **Milestone**: AI-07 — Define reasoning content with its round-trip token · **Nodes**: AI-07.1 `[leaf]`, AI-07.2 `[leaf]`, AI-07.3 `[leaf]`, AI-07.4 `[leaf]`
> **Introduced by**: `openspec/changes/archive/2026-08-01-cachicamas-ai-reasoning-content/`, in PR #101 (open for review at archive time, base `main` @ `efdedc4`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: the Layer 1 half of **G12(b)** — the reasoning round-trip token. AI-12.1 extends it with rebuild; the wire halves are AI-26.6 and AI-29.2 (Wave 2+)
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-ARC-0NN` · **Scenario IDs**: `S-ARC-0NN`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md) — `V-REQ-09`, `V-REQ-10`, `V-REQ-11`, and `V-REQ-04` … `V-REQ-07` inherited; every Layer 1 noun below is one of its rows, cited by identifier
> **Binding predecessor**: [content parts, readable and sealed](../ai-content-parts/spec.md) and its [`decision.md`](../../changes/archive/2026-08-01-cachicamas-ai-content-parts/decision.md) — reasoning adds a kind to AI-06's closed vocabulary and inherits its one part type, cited rather than re-decided
> **Binding predecessor**: [the validation error taxonomy](../ai-validation-errors/spec.md) — every rule violation here reports through AI-04's failure value and its landed rule classes
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-07--define-reasoning-content-with-its-round-trip-token) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

---

## Purpose

Constrain the runtime behavior of Layer 1's **reasoning content**: what shapes it can take, how a consumer in another package reads them, and — the requirement the milestone exists for — that an opaque provider token attached to it comes back **byte-identical**, whatever bytes it holds.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The reasoning kind, its three-member state vocabulary and the round-trip token's byte-exactness therefore live here, in their own text, and not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-01-cachicamas-ai-reasoning-content/`](../../changes/archive/2026-08-01-cachicamas-ai-reasoning-content/) is the historical record of how AI-07 was explored, proposed, designed, applied and verified.

`R-ARC-006`'s prohibition — *nothing parses, validates, normalizes, re-encodes or verifies a round-trip token* — binds every layer, not only AI-07. AI-12.1 (rebuild), AI-26.6 (outbound wire) and AI-29.2 (inbound wire) are the milestones that must re-prove byte-exactness at their own boundary; none of them may relax this file. Requirement and scenario identifiers are stable and append-only.

## Relationship to AI-06's spec

This spec **adds a kind**; it changes nothing AI-06 landed. `R-ACP-001` … `R-ACP-012` continue to hold for reasoning parts without restatement: one part type, construction as the only door, the kind derived from the payload, a closed and exhaustively registered vocabulary, a diagnostic rendering that carries no payload, and the three legs AI-06.4's guard requires. Where a requirement below overlaps one of AI-06's, it is because reasoning makes a **new** claim about it — never because the claim is re-derived.

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **Reasoning content** — `V-REQ-09`. The content-part kind carrying a model's intermediate reasoning: a state, optional text, and an opaque round-trip token. Distinct from text content at every layer.
- **A reasoning state** — `V-REQ-10`. The value distinguishing the shapes reasoning can take. It exists so that "no reasoning text" is a recorded state rather than an empty string.
- **A round-trip token** — `V-REQ-11`. An opaque provider-supplied value which cachicamas stores and returns byte-identically and never parses, reformats, re-encodes or interprets.
- **A content part**, **a content-part kind**, **readability**, **sealing** — `V-REQ-04` … `V-REQ-07`, as landed by AI-06.
- **A caller-contract failure**, **a validation sentinel**, **a position**, **the rule order** — `V-FAIL-01` … `V-FAIL-04`, as landed by AI-04.

---

## R-ARC-001 — Reasoning is a content part, through the same strategy as text

Reasoning content MUST be represented as a value of the **one** content-part type, discovered through the same discriminator and read through an accessor of the same shape as every other kind. A second part type, a parallel interface, a per-kind exported part value, or an accessor shape that differs from text's is a violation of this requirement.

WHEN a consumer in another package holds a message containing a reasoning part, THEN it MUST discover the part's kind and obtain the reasoning payload through the exported surface alone, with no type switch over unexported types.

### Scenarios

- **S-ARC-001** — Given an external package that constructs a reasoning part and places it in a message, when it reads the content back, discovers the kind and calls the matching accessor, then it obtains the payload — the identical loop it writes for text.
- **S-ARC-002** — Given the reasoning kind, when the kind vocabulary is enumerated, then it is a member, in a stable declaration order, and the enumeration is unchanged by a consumer that rewrites the slice it received.
- **S-ARC-003** — Given a reasoning part, when its diagnostic rendering is inspected, then it names the kind and reproduces neither the reasoning text nor the token.

---

## R-ARC-002 — Reasoning and text are structurally distinct

Reasoning content MUST NOT be readable as text content, and text content MUST NOT be readable as reasoning. No accessor may yield one as the other, and a consumer switching on the kind MUST NOT be able to conflate them.

### Scenarios

- **S-ARC-004** — Given a reasoning part carrying reasoning text, when the text accessor is called on it, then it reports that the part carries no text content, and the first result is the zero value rather than the reasoning text.
- **S-ARC-005** — Given a text part, when the reasoning accessor is called on it, then it reports that the part carries no reasoning.
- **S-ARC-006** — Given a reasoning part and a text part built from the identical string, when their kinds are compared, then they differ — so a consumer switching on the kind reaches a different branch for each.

---

## R-ARC-003 — The reasoning state vocabulary is closed, enumerable and complete

Layer 1 MUST expose the reasoning states as a **closed** vocabulary, enumerable in a stable declaration order, not mutable by a consumer, and each member MUST be constructible. The members are exactly three:

1. **with text** — the model emitted reasoning text;
2. **redacted** — the provider withheld the plaintext and shipped an opaque payload that must replay verbatim;
3. **token-only** — the provider emitted **no reasoning text at all**, and what remains is the token. This single state is both `V-REQ-10`'s *signature-only* shape and its *"a provider that emitted no reasoning text"* shape.

A value that is not a declared state MUST render in a form that identifies it for a diagnostic reader and MUST NOT be accepted anywhere a state is required.

### Scenarios

- **S-ARC-007** — Given each of the three states in turn, when a reasoning part of that state is constructed, then construction succeeds and the part reports that state.
- **S-ARC-008** — Given the state vocabulary, when a consumer enumerates it, then it returns exactly three members in a stable order identical on every call, and a consumer that rewrites the slice it received does not change the vocabulary.
- **S-ARC-009** — Given the zero value of the state type, when it is rendered, then the rendering marks it as unset AND it is not a member of the enumeration.
- **S-ARC-010** — Given a constructed reasoning part, when its state is read, then the state is a member of the vocabulary — no constructed part can report a state that is not.

---

## R-ARC-004 — The state agrees with the payload because it is derived from it

A reasoning payload MUST NOT store a state independently of the content it describes. The state MUST be computed from that content, so that a payload whose state contradicts its own text is not a state the type can be in.

WHEN a reasoning part reports the *with text* state, THEN it MUST carry reasoning text. WHEN it reports any other state, THEN it MUST carry no reasoning text.

### Scenarios

- **S-ARC-011** — Given a reasoning part reporting the *with text* state, when its text is read, then the text is non-empty and contains at least one non-whitespace character.
- **S-ARC-012** — Given a reasoning part reporting the redacted or token-only state, when its text is read, then the text is empty.
- **S-ARC-013** — Given reasoning constructed with no text and a token, when its state is read, then the state is *token-only* — the "no reasoning text" fact is a recorded state, never an empty string a consumer has to interpret.

---

## R-ARC-005 — A reasoning part carries an opaque token whose absence is distinguishable from an empty token

Reasoning content MUST be able to carry a provider-supplied token alongside its state and text. The exported surface MUST let a consumer distinguish **a token that is absent** from **a token that is present and zero bytes long**, and MUST let a producer express each of the two.

The distinction MUST survive placement in a message, readback, and copying.

### Scenarios

- **S-ARC-014** — Given reasoning constructed with no token, when the token accessor is called, then it reports that no token is present.
- **S-ARC-015** — Given reasoning constructed with a token of zero length, when the token accessor is called, then it reports that a token **is** present, and the token is zero bytes long.
- **S-ARC-016** — Given those two parts placed in messages and read back, when their token accessors are called, then each still reports what it was constructed with — the distinction is not collapsed by the round trip.
- **S-ARC-017** — Given a reasoning part with no reasoning text and no token, when it is constructed, then construction fails with the *required value is empty* sentinel positioned at the token: a part with neither carries nothing.

---

## R-ARC-006 — Nothing interprets the token

Layer 1 MUST NOT parse, validate, normalize, re-encode, escape, trim, case-fold, decode or verify a round-trip token. The only rule it MAY apply is a **documented sanity bound** on the token's length, exported so a caller can check before constructing, and that bound MUST NOT be presented as a provider limit.

WHEN a token is stored and read back, THEN the bytes MUST be identical for **every** byte class, including bytes that are not well-formed UTF-8, bytes that are not valid JSON, non-printable bytes, and an embedded zero byte.

### Scenarios

- **S-ARC-018** — Given a token containing every byte value from 0x00 to 0xFF, when the part is constructed and read back, then the bytes are identical and the length is unchanged.
- **S-ARC-019** — Given tokens that are not well-formed UTF-8, not valid JSON, and not printable, when each is constructed and read back, then each is byte-identical — no encoding is validated and no replacement character appears.
- **S-ARC-020** — Given a token exactly at the documented bound, when the part is constructed, then construction succeeds; given one byte more, then construction fails with the *value is outside a documented bound* sentinel positioned at the token.
- **S-ARC-021** — Given a construction failure caused by a token, when the rendered failure is inspected, then no byte of the token appears in it.

---

## R-ARC-007 — The token round-trips byte-identically through a message

WHEN a reasoning part carrying a token is placed in a message, read back, and re-attached to another message, THEN the token MUST be byte-identical to the bytes supplied at construction, for every byte class, and for a token longer than any plausible buffer boundary.

### Scenarios

- **S-ARC-022** — Given a reasoning part carrying a binary token, when it is placed in a message, read back, and placed in a second message and read again, then the token is byte-identical after every hop.
- **S-ARC-023** — Given tokens containing high Unicode encoded as bytes, an embedded zero byte, and a length larger than a typical I/O buffer, when each makes the same trip, then each is byte-identical.
- **S-ARC-024** — Given a reasoning part carrying reasoning text and a token, when it makes the trip, then the text is byte-identical too — the round trip is a property of the payload, not of the token alone.

---

## R-ARC-008 — The token survives the copy semantics of a message

A reasoning part MUST NOT share mutable state with the caller that constructed it, nor with a consumer that read it.

WHEN a caller mutates the byte slice it supplied at construction, THEN the stored token MUST NOT change. WHEN a consumer mutates the bytes an accessor returned, THEN neither the stored token nor another consumer's view of it MUST change. WHEN a message is copied, THEN the copy's token MUST be byte-identical to the original's.

### Scenarios

- **S-ARC-025** — Given a reasoning part constructed from a byte slice, when the caller overwrites every byte of that slice afterwards, then the token read back is the value supplied at construction.
- **S-ARC-026** — Given a consumer that reads the token and overwrites the bytes it received, when the token is read again, then it is unchanged, and a second consumer reading in parallel observes the original.
- **S-ARC-027** — Given a message containing a reasoning part, when the message is copied and both copies are read, then both yield byte-identical tokens.

---

## R-ARC-009 — The redacted and signature-only shapes are constructible, valid, and confusable with nothing

A **redacted** reasoning part MUST be constructible, MUST carry its opaque payload byte-exact, and MUST be distinguishable from a part that merely has no reasoning text. A reasoning part carrying a token and **no** text MUST be constructible and valid.

Neither shape MUST be readable as text content, and neither MUST be indistinguishable from a part that carries no payload at all.

### Scenarios

- **S-ARC-028** — Given a redacted reasoning part, when its state and payload are read, then the state is *redacted* and the payload is byte-identical to the bytes supplied — encrypted redacted blocks must replay verbatim.
- **S-ARC-029** — Given a redacted part and a token-only part carrying the identical bytes, when their states are compared, then they differ: "the provider withheld the plaintext" and "the provider emitted no reasoning text" are two facts, not one.
- **S-ARC-030** — Given a reasoning part with a token and no text, when it is offered as message content, then the message accepts it — the signature-only shape is valid, not a degenerate one.
- **S-ARC-031** — Given each of the two shapes, when the text-content accessor is called on it and when it is compared with a part carrying no payload, then it yields no text content AND it is not the unconstructed part: its kind is a member of the vocabulary and message construction accepts it.

---

## Out of scope for this spec

| Behavior | Owner |
| --- | --- |
| The token's survival through request rebuild | AI-12.1 |
| The token's survival through the wire, outbound and inbound | AI-26.6, AI-29.2 |
| Reasoning blocks and deltas on a stream | AI-17 |
| Reasoning **token counts** in usage | AI-13.3 |
| Which role may carry a reasoning part | AI-10.3 |
| Tool call and tool result kinds | AI-09 |
| Any verification, decryption or interpretation of a token | nobody — `V-REQ-11` forbids it at every layer |
