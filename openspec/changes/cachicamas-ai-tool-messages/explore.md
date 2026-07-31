# Explore — tool calls and tool results

> **Change**: `cachicamas-ai-tool-messages`
> **Milestone**: AI-09 — Define tool calls and tool results
> **Nodes**: AI-09.1 `[leaf]` · AI-09.2 `[leaf]` · AI-09.3 `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Depends on**: AI-04, AI-05, AI-06 (landed) · **Parallel with**: AI-07, AI-08 · **Blocks**: AI-10, AI-18, AI-26

---

## 1. What already exists, and what this milestone may not re-decide

AI-06 landed the content-part strategy as a **keystone**, and its `decision.md` § 9 was written so that
AI-09 can cite a row rather than re-derive an answer. Everything in that table is inherited here
without argument:

| Question | Inherited answer | Where it binds this milestone |
| --- | --- | --- |
| What type does a variant produce? | `ai.Part` | `NewToolCall` and `NewToolResult` return `(Part, error)` |
| How is a variant distinguished? | An unexported-method payload answering `kind()` | Two payload types, two `kind()` methods |
| How does a consumer read it? | `p.<Kind>() (T, bool)` | `Part.ToolCall()`, `Part.ToolResult()` |
| Where do the rules live? | `validate(at Path) *Violation` on the payload | One rule set, two entry points |
| Which sentinels? | AI-04's, appended only for a genuinely new class | § 4 below concludes: none appended |
| Where does it register? | `partKindNames` + the guarded GoDoc list | Two entries each, plus two witnesses |
| May a payload type be exported? | Yes, when it carries structure — opaque, unexported fields, package-owned constructor | `ToolCall` and `ToolResult` are exported; their fields are not |

The last row is the one AI-06 wrote *for this milestone*. Text got away with returning a `string`;
a tool call carries three values and a tool result carries three, so both need a value type.

## 2. What the register already names

The vocabulary register is unamended by this change. Four rows already carry every noun:

- **`V-REQ-16` tool call** — "its identity, the tool name, and exact argument bytes … Represents an
  *intent to invoke*. Layer 1 never acts on it — trap 1. Some providers assign no identifier of
  their own, in which case an adapter mints one and keeps the mapping."
- **`V-REQ-17` argument bytes** — "carried exactly as received or supplied: no re-marshalling, no key
  reordering, no whitespace normalization. Byte-equality is the contract."
- **`V-REQ-18` tool result** — "its correlation to the originating call, its content, and an
  indication of whether the tool reported failure. A *result that reports failure* is not a
  `V-FAIL-01` and not a `V-FAIL-05`; it is ordinary content."
- **`V-STR-21` call ordinal** — "The observable position of a tool call among the calls of one
  response, preserved unchanged through normalization, streaming, and adapter translation …
  **owned by AI-09**."

`V-STR-21` is the row that decides AI-09.2, and § 5 below reads it closely, because *"the position
of a tool call among the calls"* is a statement about a sequence, not about a call.

One candidate clarification is recorded rather than applied — see § 7.

## 3. The four architectural facts this milestone is built from

1. **Leakage-register row 4.** "Tool results are a block inside a user-role message on one provider,
   a distinct role on another, and a nested response object on a third … The normalized tool-result
   content part already models this. Each adapter maps it on the way out." Verdict: **adapter-local**.
   So this milestone owns the neutral shape and owes the adapters nothing else — in particular it
   does **not** decide which role may carry a tool result. That is AI-10.3's, by name.
2. **Leakage-register row 7.** "One provider assigns no tool-call identifiers. The adapter mints
   synthetic identifiers and keeps the mapping — which must survive session serialisation and
   reload." Verdict: **adapter-local + L3**. So the identity this contract carries must be an
   ordinary opaque string that survives round trips unchanged — not a minted, unforgeable handle
   like `MessageID`. An adapter must be able to *supply* it. This is the single most consequential
   shape difference between AI-05's message identity and AI-09's call identity, and § 6 records it.
3. **§ 7 G5.** "Parallel tool execution with deterministic, call-ordered re-join … Layer 1 impact:
   the tool-call ordinal must survive normalisation." Seam now.
4. **doc 0001 § 2.3 item 5.** "the ordering is not cosmetic" — several providers reject tool results
   that do not correspond positionally to their calls.

## 4. The sentinel question, answered before any code

Every rule this milestone could want, matched against AI-04's five landed classes:

| Rule | Class | Already cited by AI-04? |
| --- | --- | --- |
| The call identity is empty | `ErrEmpty` | Yes — "a required value that is empty" |
| The tool name is empty | `ErrEmpty` | Yes |
| The argument bytes are not well-formed for their documented encoding | `ErrMalformed` | **Yes, by name.** `ErrMalformed`'s own GoDoc reads: *"Register § 6.3: argument bytes that are not well-formed are decidable from the request alone. Bytes that are well-formed but do not satisfy a tool's schema are neither this nor a provider failure — this package does not validate against schemas."* |
| The result's correlation is empty | `ErrEmpty` | Yes |

**No class is appended.** The one rule that looked new — malformed argument bytes — is the case
AI-04 wrote `ErrMalformed`'s GoDoc *about*, one milestone before this one existed. AI-04 anticipated
AI-09 and this milestone consumes the anticipation rather than adding to it.

Two rules were considered and rejected, both recorded in `design.md`:

- **Uniqueness of call identities within one message.** This is a *collection* rule, not a value
  rule, and its natural home is the boundary that owns the collection. That boundary is
  `NewMessage` (AI-05, closed) at message scope and the request at AI-10.3's scope. Layer 1's own
  charter puts the transcript-wide version above this layer (`V-OUT-02`). Deferred to AI-10.3, and
  explicitly **not** stretched onto `ErrMalformed` — each repeated identity is well-formed on its
  own and what would fail is uniqueness across a collection, which is a different class of fact.
- **A documented byte ceiling on argument bytes and result content.** AI-08 bounds a tool *name* and
  does not bound schema bytes; AI-06 bounds text because `V-REQ-24`'s reasoning applies to a value a
  human typed. Argument bytes are model-produced and their size is the model's business.
  `design.md` § 7 records the deferral rather than landing an arbitrary constant.

## 5. The ordinal: three candidate homes, walked

Doc 0002 hands this milestone a warning rather than an answer: *"a value stored inside the part at
construction time can disagree with the part's actual position in the message, and a part is placed
in a message after it is built."*

| Candidate | How it would work | Why it fails, or what it costs |
| --- | --- | --- |
| **A — a field on the call** | `NewToolCall(id, name, args, ordinal)` | The constructor cannot know the position: a part is built, *then* placed. Two calls built with ordinal 0 can both reach one message, and a call built with ordinal 7 can be the only call in it. The stored value and the observable position are two things that can disagree — which is **exactly** the shape AI-06 § 5 removed when it made the kind derived rather than stored |
| **B — assigned by the message** | `NewMessage` rewrites each tool-call part with its index | `Message` would have to mutate parts it was handed, which contradicts "the sequence is copied, not the parts in it", and it puts a tool-specific rule inside the general message boundary. It also cannot serve a *request*-level ordinal without AI-10 re-doing it |
| **C — derived from position** | The ordinal *is* the index of a call among the tool-call parts of a content sequence | Nothing to disagree with, because there is only one of it. Costs one exported derivation function. This is AI-06 § 5's answer transposed one level up |

**C.** The precedent is not an analogy, it is the same argument in the same package one milestone
earlier: *"[Part] holds no kind field. [Part.Kind] asks the payload, so the discriminator and the
payload cannot disagree — there is only one of them."* Replace *kind* with *ordinal* and *payload*
with *position* and the sentence still holds.

`V-STR-21`'s own wording supports it: the ordinal is *"the observable position of a tool call among
the calls of one response"* — a property of a call's place in a sequence, which is precisely what a
derivation reads and what a stored field can only mirror.

Two consequences, both accepted and both recorded in `design.md` § 5:

1. A `ToolCall` **on its own** has no ordinal. It gets one from a sequence. AI-18's streaming events
   and AI-30's adapter will carry an ordinal *field*, because a stream event is not inside a
   sequence a reader can index — and `V-STR-21` calls those **restatements** of this term, not
   redefinitions, so the two are not in tension.
2. The derivation must be reachable from another package without touching `Message`. AI-09 owns no
   part of `message.go` (AI-05's, closed, and being merged against concurrently), so the derivation
   is a package-level function over a content sequence rather than a method on `Message` — which is
   also the more general shape, since AI-10's request walk and a stream reassembly both hold
   `[]Part` and neither holds a `Message`.

## 6. Identity: why a tool call's is a plain string and a message's is not

AI-05 made `MessageID` an opaque minted struct precisely so a caller *cannot* supply one. Copying
that here would be the wrong lesson learned well:

| | `MessageID` (AI-05) | tool-call identity (AI-09) |
| --- | --- | --- |
| Who produces it | This package, by minting | The **model**, or an adapter minting a synthetic one (register row 7) |
| Must a caller be able to supply it? | No — supply is the defect | **Yes** — an adapter has to, and so does a session reload one layer up |
| Must it survive serialisation? | Not stated | **Yes**, explicitly — doc 0001 § 5.2 |
| Shape | Opaque struct, no parser | Opaque **string**, carried verbatim |

So the identity is `string`, stored and returned unchanged, and the only rule on it is that it is not
empty. A synthetic identifier an adapter minted is textually indistinguishable from one a provider
sent, which is the property AI-26.5 needs and the reason there is deliberately no validation of its
shape.

## 7. The argument-byte encoding question — read against AI-08 § 8 first

AI-08's `design.md` § 8 recorded *both sides* of the schema-syntax argument and declined to land the
check, stating the reason as an asymmetry: *"doc 0002 asks for syntactic validation explicitly on
**AI-09.1's argument bytes** … and pointedly does not on **AI-08.1's schema bytes**"*, because
*"argument bytes are produced by a model and reassembled from a stream, so a malformed reassembly is
a real and recurring defect class that AI-30 must catch; schema bytes are supplied by the caller as a
literal in their own source."*

This milestone reads that paragraph as instructed and **agrees with it**, landing the check on
arguments. `design.md` § 4 carries the full reasoning and the conceded cost. The short version:
AI-08's "against" paragraph is an argument *for* the asymmetry, not against the check, and AI-04's
`ErrMalformed` GoDoc had already committed the package to it in prose.

**Candidate register clarification, reported and not applied.** `V-REQ-17` says argument bytes are
carried *"exactly as received or supplied"*. Doc 0002's AI-09.1 item 4 asks that a call with **empty**
arguments *"normalizes to one canonical empty form"*. Those two sentences are in tension for exactly
one input — the empty one — and this change resolves it as: byte fidelity is a property of *supplied*
bytes, and *absent* arguments are not supplied bytes. A one-clause addition to `V-REQ-17` would make
that explicit. This change does not edit the canonical register; `R-ATM-003` states the resolution
locally and the orchestrator holds the register amendment.

## 8. Budget forecast

| Artifact | Forecast |
| --- | --- |
| `src/ai/tool_call.go` | ~135 lines including GoDoc |
| `src/ai/tool_result.go` | ~110 lines including GoDoc |
| `src/ai/content_part.go` | +12 lines, appended in three places only |
| `src/ai/content_part_registry_test.go` | +34 lines, two witnesses appended |
| `src/ai/tool_call_test.go` | ~250 lines (`ai_test`) |
| `src/ai/tool_result_test.go` | ~170 lines (`ai_test`) |
| **Total** | **~710 changed lines** |

Over doc 0002's 400-line reassessment threshold, and the reassessment is recorded rather than
resolved by deletion. The milestone is **two content kinds plus a derivation**, which doc 0002 chose
to keep in one milestone; splitting the file set further would put `tool_result.go` in a second PR
that cannot compile without the first, and the two share the registration edit and the guard table.
`tasks.md` § Review Workload Forecast states the chain a reviewer can use instead: three commits on
the leaf boundary, which is doc 0002's own PR-chain boundary. Tests are not cut to fit.

## 9. What is deliberately not explored

Executing anything, resolving a tool name to an implementation, validating arguments against a
schema, pairing calls to results across a transcript, and any wire shape. The first four are
`V-OUT-04` and `V-OUT-02`; the last is AI-24 onward.
