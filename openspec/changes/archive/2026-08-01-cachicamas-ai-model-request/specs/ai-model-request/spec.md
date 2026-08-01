# Spec — the normalized request core

> **Change**: `cachicamas-ai-model-request`
> **Milestone**: AI-10 · **Nodes**: AI-10.1 … AI-10.6, all `[leaf]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-model-request/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-AMR-0NN` · **Scenario IDs**: `S-AMR-0NN`
> **Binding input**: [the register](../../../../../specs/ai-contract-vocabulary/spec.md) — `V-REQ-01` … `V-REQ-22`, `V-REQ-26`, `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-13`, `V-FAIL-17` · [AI-04's spec](../../../2026-08-01-cachicamas-ai-validation-errors/specs/ai-validation-errors/spec.md) · [AI-05's spec](../../../2026-08-01-cachicamas-ai-message-roles/specs/ai-message-roles/spec.md) · [AI-06's spec](../../../2026-08-01-cachicamas-ai-content-parts/specs/ai-content-parts/spec.md) and its [`decision.md`](../../../2026-08-01-cachicamas-ai-content-parts/decision.md) § 7.2 · [`explore.md`](../../explore.md) · [`proposal.md`](../../proposal.md)

---

## Purpose

Constrain the runtime behavior of `V-REQ-20` — the normalized request: the one value a provider receives. This spec covers all six leaves of AI-10. **Implementation status is tracked in `tasks.md`, not here**: at the time this file was written, AI-10.1 and AI-10.2 are implemented and AI-10.3 … AI-10.6 are planned. A requirement being written here is a statement about the finished milestone, not a claim that it is green.

Requirement count: **17**. `proposal.md` forecast 16; the redaction posture was split out as `R-AMR-017` rather than folded into the readability requirement, because it is the opposite property and a reader must be able to cite one without the other.

## Requirement ownership by leaf

| Leaf | Requirements | Status |
| --- | --- | --- |
| AI-10.1 | `R-AMR-001`, `R-AMR-002`, `R-AMR-003`, `R-AMR-004`, `R-AMR-017` | **implemented** |
| AI-10.2 | `R-AMR-005`, `R-AMR-006`, `R-AMR-007` | **implemented** |
| AI-10.3 | `R-AMR-008`, `R-AMR-009`, `R-AMR-010`, `R-AMR-011`, `R-AMR-012` | planned |
| AI-10.4 | `R-AMR-013`, `R-AMR-014` | planned |
| AI-10.5 | `R-AMR-015` | planned |
| AI-10.6 | `R-AMR-016` | planned |

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A normalized request** — `V-REQ-20`. The complete provider-neutral description of one model call: model identity, ordered system-instruction segments, ordered messages, the tool set and tool choice, and generation options.
- **A model identity** — `V-REQ-21`. The neutral name of the model a request targets. Not a catalog entry.
- **A system-instruction segment** — `V-REQ-19`. One ordered, individually markable piece of the system instruction.
- **Request validation** — `V-REQ-22`. The single check that a request satisfies every construction and composition rule, performed **once, before any I/O**.
- **A generation option** — `V-REQ-26`. A neutral, provider-independent parameter shaping generation. Neutrality is the admission test.
- **A region** — this spec's term for one of the six addressable parts of a request: model identity, system instruction, messages, tool set, tool choice, generation options.
- **A caller-contract failure**, **a validation sentinel**, **a position**, **the rule order** — `V-FAIL-01` … `V-FAIL-04`, as landed by AI-04.
- **A message**, **a role**, **content** — `V-REQ-01` … `V-REQ-03`, as landed by AI-05. There is **no system role**.
- **A content part**, **a content-part kind** — `V-REQ-04`, `V-REQ-05`, as landed by AI-06.

---

## R-AMR-001 — A request is one sealed value, and construction is its only door

Layer 1 MUST represent a normalized request as **one** exported value type. That type MUST NOT expose any field to another package, and it MUST NOT be an interface. A value of it MUST be obtainable from another package only by calling this package's construction function.

The construction function MUST take the **required** regions as parameters and every **optional** region through an option value this package alone can produce. An option that was not applied MUST be structurally distinguishable from one applied with a zero value; a sentinel value meaning "unset" is a violation of this requirement.

Construction MUST return the zero request on failure, so a caller that ignored the error cannot mistake the result for a constructed request.

### Scenarios

- **S-AMR-001** — Given the request type, when its exported surface is inspected, then it declares no exported field, so no consumer in another package can assemble one by literal.
- **S-AMR-002** — Given the option type, when a consumer in another package attempts to produce a value of it other than by calling this package's option constructors, then the program does not compile.
- **S-AMR-003** — Given a construction call that violates a rule, when the caller inspects the returned request, then it is the zero value and reports no model identity.

---

## R-AMR-002 — Model identity and messages are required, and their absence is a caller-contract failure

A request MUST carry a non-empty model identity and at least one message.

WHEN the model identity is empty, or consists only of whitespace, THEN construction MUST fail with `ErrEmpty` at `model`.

WHEN no message is supplied, THEN construction MUST fail with `ErrEmpty` at `messages`.

WHEN a supplied message never passed message construction, THEN construction MUST fail with `ErrEmpty` at `messages[i]`, where `i` is its index.

The model identity MUST NOT be checked against any catalog, price table or selection policy. Only emptiness is decidable in this layer; an unrecognised model is a provider failure (register § 6.3).

### Scenarios

- **S-AMR-004** — Given a model identity and one user text message, when the request is constructed, then construction succeeds.
- **S-AMR-005** — Given an empty model identity, when the request is constructed, then it fails, `errors.Is` reports `ErrEmpty`, and the position renders `model`.
- **S-AMR-006** — Given a whitespace-only model identity, when the request is constructed, then it fails identically to the empty one, because a name made of spaces names nothing.
- **S-AMR-007** — Given no messages, when the request is constructed, then it fails, `errors.Is` reports `ErrEmpty`, and the position renders `messages`.
- **S-AMR-008** — Given a message value that skipped its constructor, when the request is constructed, then it fails with `ErrEmpty` at `messages[0]`.
- **S-AMR-009** — Given a model identity naming a model no provider offers, when the request is constructed, then construction **succeeds**, because recognition is not decidable from the request alone.

---

## R-AMR-003 — Every region a request holds is readable from another package

WHEN a consumer in a package other than the one that owns the contract holds a constructed request, THEN it MUST be able to read every region through the exported surface alone, with no access to unexported state.

Every region MUST read back **exactly** as supplied: byte-equal for text, order-preserving for sequences. No normalization, trimming, re-encoding or case folding is permitted on either the construction path or the read path.

A reader of an optional region MUST be able to tell "absent" from "present" without knowing that a zero value means absence.

### Scenarios

- **S-AMR-010** — Given an external package holding a request built from a model identity and one user text message, when it reads the model identity and the message content, then both are byte-equal to what was supplied.
- **S-AMR-011** — Given a model identity carrying punctuation and non-ASCII bytes, when it is read back, then it is byte-equal, because this layer carries a name rather than parsing one.
- **S-AMR-012** — Given a request with no system instruction, when a reader asks for it, then the second result reports absence rather than a zero value the reader must interpret.

---

## R-AMR-004 — Generation options are the neutral set, individually optional, and bounded

A request MUST carry generation options drawn from a closed, neutral set admitted by `V-REQ-26`'s test. The v1 set is: **maximum output tokens, temperature, top-p, stop sequences**. An option that exists to satisfy one provider MUST NOT be added here; it belongs to the provider escape hatch (`V-REQ-28`, AI-12).

Each option MUST be **individually** optional. A request with an option unset MUST be distinguishable from one with the same option set to its zero value.

Bounds that are decidable from the request alone MUST be enforced at construction:

| Option | Rule | Class | Position |
| --- | --- | --- | --- |
| maximum output tokens | strictly positive | `ErrOutOfRange` | `maxOutputTokens` |
| temperature | not negative | `ErrOutOfRange` | `temperature` |
| top-p | greater than 0 and not greater than 1 | `ErrOutOfRange` | `topP` |
| stop sequences | applied with at least one sequence | `ErrEmpty` | `stopSequences` |
| stop sequences | every sequence non-empty | `ErrEmpty` | `stopSequences[i]` |

Temperature MUST NOT carry an upper bound. Providers disagree on it (1.0 versus 2.0), and a bound correct for one is a caller-contract failure this package invented for the other.

### Scenarios

- **S-AMR-013** — Given a request with all four options applied, when each is read back from an external package, then each reports its value and reports that it is set.
- **S-AMR-014** — Given a request with no option applied, when each is read back, then each reports that it is not set, and the first result is the zero value rather than data.
- **S-AMR-015** — Given a request with temperature applied as `0`, when it is read back, then it reports `0` **and** reports that it is set — distinguishable from S-AMR-014's unset temperature.
- **S-AMR-016** — Given maximum output tokens applied as `0` or a negative value, when the request is constructed, then it fails with `ErrOutOfRange` at `maxOutputTokens`.
- **S-AMR-017** — Given a negative temperature, when the request is constructed, then it fails with `ErrOutOfRange` at `temperature`.
- **S-AMR-018** — Given a top-p of `0`, or above `1`, when the request is constructed, then it fails with `ErrOutOfRange` at `topP`; given a top-p of exactly `1`, construction succeeds.
- **S-AMR-019** — Given the stop-sequence option applied with no sequences, when the request is constructed, then it fails with `ErrEmpty` at `stopSequences`.
- **S-AMR-020** — Given a stop-sequence list whose second element is empty, when the request is constructed, then it fails with `ErrEmpty` at `stopSequences[1]`.
- **S-AMR-021** — Given a request whose stop sequences were read back, when the reader mutates the returned slice and re-reads, then the request is unchanged.

---

## R-AMR-005 — The system instruction is ordered segments, and the order round-trips

A system instruction MUST be modelled as an **ordered sequence of segments** rather than as a single string. Segment order and each segment's text MUST round-trip exactly through construction and readback from another package.

The reason is recorded rather than assumed: doc 0001 § 3.2 lists a flat system instruction first among the things that must change before an adapter exists, because a flat string has nowhere to put a cache boundary. Retrofitting segments was a breaking change in the retired plan. They cost nothing at birth.

### Scenarios

- **S-AMR-022** — Given a system instruction built from three segments, when an external package reads the request's system instruction and walks its segments, then it obtains three segments whose texts are byte-equal to those supplied, in the order supplied.
- **S-AMR-023** — Given the same instruction, when the reader mutates the segment slice it received and re-reads, then the request's segments are unchanged.
- **S-AMR-024** — Given a segment whose text carries punctuation, newlines and non-ASCII bytes, when it is read back, then it is byte-equal.

---

## R-AMR-006 — The single-segment path is indistinguishable, and absence is structural

The ergonomic single-segment construction path MUST produce a system instruction **indistinguishable** from one built segment-by-segment with one segment: same segment count, same segment text, and equal under this spec's documented equality (`R-AMR-016`).

An absent system instruction MUST be legal, and MUST be expressed **structurally** — by the option not being applied — not by a present-but-empty value.

"One empty segment" MUST be unrepresentable rather than merely equivalent to absence. A system instruction MUST NOT be constructible with zero segments, and a segment MUST NOT be constructible with empty text (`R-AMR-007`). Together these make absence the only zero-content state.

### Scenarios

- **S-AMR-025** — Given a system instruction built by the single-segment path and one built by the segment-by-segment path from the same text, when both are read back, then their segment counts and segment texts are equal.
- **S-AMR-026** — Given a request with no system instruction applied, when it is constructed, then construction succeeds and the reader is told the region is absent.
- **S-AMR-027** — Given a request whose system instruction holds one segment, when a reader compares it against a request with no system instruction, then the two are distinguishable by the presence flag alone, without inspecting segment text.
- **S-AMR-028** — Given an attempt to construct a system instruction with zero segments, when it is made, then it fails with `ErrEmpty` at `system`, so the value "an instruction that says nothing" does not exist.
- **S-AMR-029** — Given the system-instruction option applied with a value that skipped its constructor, when the request is constructed, then it fails with `ErrEmpty` at `system`.

---

## R-AMR-007 — Segment construction rules report through AI-04's vocabulary

A segment MUST carry non-empty text. WHEN a segment is constructed with empty text, or with text consisting only of whitespace, THEN construction MUST fail with `ErrEmpty` at `text`.

Whitespace-only is rejected for the same reason emptiness is: a segment exists to place text into the instruction, and a segment of spaces places none while occupying an ordinal that `V-REQ-19` says is individually markable. Accepting it would let a caller mark a cache boundary on nothing.

The text MUST NOT be trimmed. Whitespace-only is a rejection criterion, never a normalization: a segment whose text is `"  be terse.  "` MUST round-trip with its padding intact, because an adapter concatenating segments needs the caller's own separators.

### Scenarios

- **S-AMR-030** — Given empty segment text, when the segment is constructed, then it fails, `errors.Is` reports `ErrEmpty`, and the position renders `text`.
- **S-AMR-031** — Given segment text made only of spaces, tabs and newlines, when the segment is constructed, then it fails identically.
- **S-AMR-032** — Given segment text with leading and trailing whitespace around real content, when the segment is constructed and read back, then construction succeeds and the text is byte-equal including its padding.

---

## R-AMR-008 — Message order and intra-message content order are preserved *(AI-10.3, planned)*

A request MUST preserve the order of the messages supplied to it, and each message MUST preserve the order of its content parts. Both MUST be observable from another package.

### Scenarios

- **S-AMR-033** — Given a request built from five messages in a known order, when an external package reads them, then it obtains five messages in that order.
- **S-AMR-034** — Given a message holding text, reasoning and a tool call in a known order, when it is read back through the request, then the parts appear in that order.
- **S-AMR-035** — Given a request whose message slice was read back, when the reader reorders the returned slice and re-reads, then the request's order is unchanged.

---

## R-AMR-009 — The content rules are AI-06's, called at request depth *(AI-10.3, planned)*

Request validation MUST validate message content by calling AI-06's content validator with a request-shaped position prefix. It MUST NOT reimplement any per-kind content rule.

WHEN a content part inside a request violates one of AI-06's rules, THEN the reported position MUST compose the request depth with the message depth — for example `messages[2].content[0]`.

This is `decision.md` § 7.2's "one rule set, two callers", cashed. AI-06.3 item 3 pinned the request-shaped prefix before a request existed; this requirement is the other end of that seam.

### Scenarios

- **S-AMR-036** — Given a request whose third message's first content part skipped its constructor, when the request is constructed, then it fails with the class AI-06 reports for that part, at position `messages[2].content[0]`.
- **S-AMR-037** — Given a request whose second message holds a text part exceeding the documented text bound, when the request is constructed, then it fails with the class AI-06's text rule reports, at `messages[1].content[0].text`, proving the rule ran rather than being reimplemented.

---

## R-AMR-010 — Tool set and tool choice attach, and are cross-validated at the request boundary *(AI-10.3, planned)*

A request MUST be able to carry a tool set and a tool choice, both optional and independently so.

WHEN a request carries a tool choice, THEN request validation MUST run AI-08.3's cross-validation of that choice against the request's tool set, with its three rules and its positions unchanged. Request validation MUST NOT reimplement them.

### Scenarios

- **S-AMR-038** — Given a request carrying three tool declarations and a specific tool choice naming the second, when it is constructed, then construction succeeds and both regions read back from another package.
- **S-AMR-039** — Given a request carrying a specific tool choice and no tool set, when it is constructed, then it fails with `ErrEmpty` at `tools` — AI-08.3's rule 2, at its own position.
- **S-AMR-040** — Given a request carrying tools and a specific tool choice naming an undeclared tool, when it is constructed, then it fails with `ErrUnresolvedReference` at `toolChoice.name`.
- **S-AMR-041** — Given a request carrying a tool choice that skipped its constructor, when it is constructed, then it fails with `ErrNotInVocabulary` at `toolChoice`.
- **S-AMR-042** — Given a request carrying tools and no tool choice, when it is constructed, then construction succeeds, because omitting the choice is not the same as choosing none.

---

## R-AMR-011 — Role versus content kind is enforced from a documented table *(AI-10.3, planned)*

A content part that is individually valid MUST still be rejected when it appears under a role that may not carry its kind. The table is:

| Kind | `user` | `assistant` | `tool` |
| --- | --- | --- | --- |
| `text` | permitted | permitted | forbidden |
| `reasoning` | forbidden | permitted | forbidden |
| `tool_call` | forbidden | permitted | forbidden |
| `tool_result` | forbidden | forbidden | permitted |

WHEN a part appears under a role the table forbids, THEN validation MUST fail with `ErrMisplaced` at `messages[i].content[j]`.

`ErrMisplaced` is a rule class **appended** to AI-04's set under AI-04's own append rule, because none of the six landed classes describes a value that is well-formed and wrongly placed. Appending it obliges this change to update both registry mirrors in the same commit.

The reason the table is strict rather than permissive: an adapter maps **from** the neutral shape (doc 0001 § 3.3 row 4), and a neutral shape with two legal placements for one kind makes every adapter handle both forever. Loosening a cell later is additive; tightening one breaks a caller who shipped.

### Scenarios

- **S-AMR-043** — Given a reasoning part in a user message, when the request is constructed, then it fails with `ErrMisplaced` at that part's composed position.
- **S-AMR-044** — Given a tool-result part in an assistant message, when the request is constructed, then it fails with `ErrMisplaced`.
- **S-AMR-045** — Given a text part in a tool message, when the request is constructed, then it fails with `ErrMisplaced`.
- **S-AMR-046** — Given each of the five permitted cells in turn, when a request is built holding it, then construction succeeds — so the table is proven in both directions, twelve cells total.

---

## R-AMR-012 — Tool-call and tool-result correlation is checked across the request *(AI-10.3, planned)*

WHEN a tool-result part correlates to a call identity that appears in **no** tool-call part anywhere in the request, THEN validation MUST fail with `ErrUnresolvedReference` at the result's composed position.

WHEN a tool-call part carries an identity another tool-call part in the request already carries, THEN validation MUST fail with `ErrDuplicate` at the **second** occurrence's composed position.

A tool call with no matching result MUST be legal. A call awaiting its result is the ordinary mid-turn state, and repairing orphaned calls is a transcript invariant one layer up (`V-OUT-02`).

Whether a call must appear **before** the result that correlates to it is deliberately unlanded. Doc 0002 scopes the rule to existence "anywhere in the request"; ordering repair is `V-OUT-02`'s. The disposition MUST be pinned by test so it is asserted rather than assumed.

Uniqueness is a precondition rather than a preference: the orphan rule resolves a result to a call **by identity**, and a rule that resolves by a non-unique key is not a rule.

### Scenarios

- **S-AMR-047** — Given a request whose tool message carries a result correlating to an identity no call in the request holds, when it is constructed, then it fails with `ErrUnresolvedReference` at `messages[i].content[j]`.
- **S-AMR-048** — Given a request whose assistant message holds a tool call with no matching result, when it is constructed, then construction **succeeds**.
- **S-AMR-049** — Given a request holding two tool calls with the same identity, when it is constructed, then it fails with `ErrDuplicate` positioned at the second one.
- **S-AMR-050** — Given a request whose result appears in an **earlier** message than the call it correlates to, when it is constructed, then construction succeeds, pinning the deliberate non-decision about ordering.

---

## R-AMR-013 — Validation happens once, before any I/O *(AI-10.4, planned)*

Request validation MUST run entirely at construction, MUST complete before any transport is contacted, and MUST perform no I/O of its own.

The absence of I/O MUST be asserted **mechanically**, in the style of the AI-00.3 import guard: the dependency closure of the request path contains no network and no filesystem package.

Validation MUST be **total** over the regions: a request that passes MUST NOT be able to contain an unconstructed part, a duplicate tool name, or an unresolvable tool choice.

### Scenarios

- **S-AMR-051** — Given the request path's package, when its dependency closure is computed, then it contains no `net`, `net/http`, `os` or `io/fs` package.
- **S-AMR-052** — Given a request that passed construction, when every region is re-examined, then no region holds a value its own contract would reject.

---

## R-AMR-014 — The first failure is deterministic and follows a documented order *(AI-10.4, planned)*

WHEN a request violates more than one rule, THEN validation MUST report the violation that comes first in the documented rule order, and MUST report the same one on every run.

The order MUST be documented in `design.md` and MUST be data rather than control flow, so a reviewer reads which rule wins instead of tracing it. This is AI-04.3's property, at request scope.

### Scenarios

- **S-AMR-053** — Given a request with an empty model identity **and** no messages, when it is constructed, then it reports the model failure, repeatedly across many runs.
- **S-AMR-054** — Given a request violating one rule from each of four regions at once, when it is constructed a hundred times, then the identical failure — same class, same position — is reported every time.

---

## R-AMR-015 — A whole request round-trips, exhaustively over the registered kinds *(AI-10.5, planned)*

WHEN an external-package test walks a request holding every registered content-part kind — text, reasoning with its round-trip token, tool call, tool result — plus system segments, tools, a tool choice and every generation option, THEN it MUST be able to reconstruct an **equal** request from what it read.

The walk's kind handling MUST be **exhaustive over the registered kinds**, consuming AI-06.4's registration: adding a kind without a readable accessor MUST fail this pin.

This is the property AI-26's translator consumes. It is proven before AI-26 exists, which is the whole point of proving it here.

### Scenarios

- **S-AMR-055** — Given a request holding all four part kinds plus every optional region, when an external package reads every region and rebuilds a request from what it read, then the rebuilt request is equal to the original under `R-AMR-016`'s equality.
- **S-AMR-056** — *(pin)* Given the registered kind set, when the round-trip walk is checked against it, then every registered kind is handled, and a kind added without an accessor fails the pin.

---

## R-AMR-016 — A request is immutable, and equality is documented *(AI-10.6, planned)*

WHEN a caller mutates anything a reader returned from a request, THEN the request MUST be observably unchanged on re-read.

WHEN a caller mutates the values it passed to the constructor, THEN the constructed request MUST be observably unchanged. This is AI-05.3's property at request scope.

Equality MUST be documented. Two requests constructed from identical inputs MUST compare equal under that documented equality, and operations on one MUST NOT affect the other.

Because a request holds slices, `==` MUST NOT be the documented equality: Go does not define it for the type, and a comparison a consumer cannot write is not a contract. Equality is **region-wise readback equality** — each region compared by the values its accessors return — with message identity excluded, because `V-REQ-03` makes two messages built from identical inputs deliberately distinguishable.

### Scenarios

- **S-AMR-057** — Given a request whose message slice a reader mutated, when the request is re-read, then it yields the original messages.
- **S-AMR-058** — Given a caller that mutates the message slice it passed to the constructor, when the request is re-read, then it yields what was passed at construction time.
- **S-AMR-059** — Given two requests constructed from identical inputs, when they are compared under the documented equality, then they are equal.
- **S-AMR-060** — Given those two requests, when one's readbacks are mutated, then the other is unchanged.

---

## R-AMR-017 — A request renders no payload through any formatting verb

A request holds every payload in the package at once — the prompt, the model's deliberation, tool arguments, tool results and the system instruction — which makes it the highest-value leak in Layer 1.

The request type, the system-instruction type and the segment type MUST each define both a string rendering and a Go-syntax rendering, and both MUST name **structural facts only**: which regions are present, and how many elements each ordered region holds. Neither MUST reproduce a model identity, a segment's text, a message's content, a tool's name or schema, a stop sequence, or any option value.

WHEN a request holding a secret in every region is formatted through any verb, THEN the secret MUST NOT appear in the output.

The Go-syntax rendering is required explicitly and not by implication: without it, that verb falls back to reflection and prints the unexported fields, which would make the posture a property of which verb a caller reached for. `V-FAIL-13` puts it on the type instead.

### Scenarios

- **S-AMR-061** — Given a request whose model identity, system segment, message text, tool name, tool arguments and stop sequence each carry a distinct secret, when the request is formatted through the default, string, extended and Go-syntax verbs, then none of the six secrets appears in any of the four outputs.
- **S-AMR-062** — Given the same request, when it is formatted, then the rendering names the present regions and their element counts, so a diagnostic reader still learns the shape.
- **S-AMR-063** — Given a system instruction and a segment formatted directly through the same four verbs, when the outputs are inspected, then neither reproduces the segment's text.
