# Design — tool calls and tool results

> **Change**: `cachicamas-ai-tool-messages`
> **Milestone**: AI-09 · **Nodes**: AI-09.1, AI-09.2, AI-09.3 — all `[leaf]`
> **Phase**: design
> **Date**: 2026-07-31
> **Binding input**: [`proposal.md`](./proposal.md) · [`specs/ai-tool-messages/spec.md`](./specs/ai-tool-messages/spec.md) · [`explore.md`](./explore.md) · [AI-06's `decision.md`](../cachicamas-ai-content-parts/decision.md) — **inherited whole** · [AI-08's `design.md` § 8](../cachicamas-ai-tool-declarations/design.md) — read before § 4 below

---

## 1. What this design does not decide

AI-06.1 is the keystone and this milestone is a consumer of it, not a peer. The part strategy, the
derived kind, the `(T, bool)` accessor, the placement of rules on the payload, the "no new sentinel"
posture and the five-step registration procedure are all cited from `decision.md` § 9 and are **not
re-argued here**. This document decides the three things AI-06 § 12 explicitly left open for AI-09
— *"the shape of structured payloads"* — plus the two questions doc 0002 aimed at this milestone by
name: where the ordinal lives, and whether argument bytes are checked for syntax.

## 2. The Go surface, whole

```go
// tool_call.go
type ToolCall struct{ id, name, arguments string }        // exported, opaque, comparable

func NewToolCall(id, name string, arguments []byte) (Part, error)
func (p Part) ToolCall() (ToolCall, bool)
func (c ToolCall) ID() string
func (c ToolCall) Name() string
func (c ToolCall) Arguments() []byte
func (c ToolCall) String() string
func (c ToolCall) GoString() string

func ToolCalls(content []Part) []ToolCall                  // the ordinal derivation

const emptyToolArguments = "{}"                            // unexported: the canonical empty form

// tool_result.go
type ToolResult struct {                                   // exported, opaque, comparable
	callID, content string
	failed          bool
}

func NewToolResult(callID, content string) (Part, error)   // the tool answered
func NewToolFailure(callID, content string) (Part, error)  // the tool failed, and that is content
func (p Part) ToolResult() (ToolResult, bool)
func (r ToolResult) CallID() string
func (r ToolResult) Content() string
func (r ToolResult) Failed() bool
func (r ToolResult) String() string
func (r ToolResult) GoString() string

// content_part.go — appended at three points, nothing existing touched
PartKindToolCall
PartKindToolResult
```

### 2.1 The payload type *is* the exported value type

`textPayload` is unexported because `Part.Text()` returns a `string`. A tool call carries three
values, so AI-06 `decision.md` § 6.3 applies: *"an exported payload type is an opaque value type with
unexported fields, or it is a defect."*

The design goes one step further than § 6.3 requires and makes `ToolCall` **itself** the payload —
it is the type that implements `kind()` and `validate(at Path)`. The alternative, an unexported
`toolCallPayload` wrapped by an exported `ToolCall`, would mean two types holding the same three
fields and a conversion on the accessor path. The exported type is safe as the payload for the same
reason `Part` is safe as a struct: `partPayload`'s methods are unexported, so no outside type can
satisfy it, and `ToolCall`'s fields are unexported, so no outside package can assemble one. The seal
is on `Part`, and it does not move.

### 2.2 Why the argument bytes are stored as a `string`

`Part` documents that *"Equality with == is defined and compares payloads."* An interface value
holding a struct with a slice field **panics** when compared, so a payload holding `[]byte` would
turn a documented property of AI-06's contract into a runtime panic reachable from any consumer that
compares two parts. Storing a `string` keeps `ToolCall` comparable, keeps the stored value immutable,
and makes the copy-in and copy-out non-aliasing for free rather than by discipline:

- **in** — `string(arguments)` copies the caller's bytes;
- **out** — `[]byte(c.arguments)` copies them back.

Two mechanisms and not one, which is `message.go`'s own stated rule: *"a constructor that clones and
a reader that does not passes every construction test and fails the moment two consumers hold the
same declaration."* `Arguments()` returns `[]byte` rather than `string` because the contract's noun
is *argument bytes* and because AI-26 and AI-30 both hold bytes; the conversion is where the copy is.

### 2.3 Why the call identity is a plain `string`

`explore.md` § 6 has the table. The one-sentence version: `MessageID` is opaque and minted because a
caller supplying one is the defect; a tool-call identity is opaque and **supplied** because an
adapter minting a synthetic one is the *requirement* (leakage row 7), and because a session reload
one layer up has to put the same string back. A minted handle here would make register row 7
unimplementable.

`ToolResult.CallID()` returns the same `string` type and compares to `ToolCall.ID()` with `==`. No
correlation helper is exported: a consumer pairing calls to results is doing Layer 2's job
(`V-OUT-02`), and giving it a function of ours would be an invitation.

---

## 3. Rules, and their documented order

### 3.1 `NewToolCall`

| # | Rule | Class | Position |
| --- | --- | --- | --- |
| 1 | the identity is not empty | `ErrEmpty` | `id` |
| 2 | the tool name is not empty | `ErrEmpty` | `name` |
| 3 | the argument bytes are well-formed JSON | `ErrMalformed` | `arguments` |

Identity before name because a call with neither is more usefully reported as "you gave me no
identity": the identity is what makes the call addressable at all, and a caller fixing one failure at
a time must make progress in a predictable direction — `text_content.go`'s stated reason, applied to
three rules instead of two.

**The tool name has no shape rule here.** AI-08 lands `toolNameRules` — length, first byte, character
set — on a tool *declaration*, and this milestone does not import it, for two reasons. It is landing
concurrently in a sibling worktree and is not on this branch, so depending on it would be depending
on a merge. And doc 0002's AI-09.1 item 3 lists exactly three rules, of which the name's is
"empty name". Whether a call's name must satisfy the same shape rule as a declaration's is a
*cross-region* question — the same class as "does this tool choice name a declared tool?" — and
AI-10.3 owns cross-region validation by name. Recorded here so the absence reads as a decision.

### 3.2 `NewToolResult` and `NewToolFailure`

| # | Rule | Class | Position |
| --- | --- | --- | --- |
| 1 | the correlation is not empty | `ErrEmpty` | `call` |

**Content has no rule, and the emptiness of it is legal.** A tool that produced no output — a search
with no matches, a write that printed nothing — is routine, and Layer 1 rejecting it would force
every caller to invent a placeholder string. Which placeholder, and whether a provider needs one, is
an application decision: `V-OUT-04` puts tool execution above this layer, and the two wording traps
in `doc.go` put "application behavior" outside it. This is the one place where a tool result
deliberately differs from `text_content.go`, whose emptiness rule exists because `V-REQ-08` says
*model-visible natural-language text* and a string of separators is none.

**No duplicate rule.** Uniqueness of call identities across a message is a collection rule; each
identity is well-formed on its own and what would fail is uniqueness across a collection. Stretching
`ErrMalformed` onto it would be the defect AI-08 shipped and had corrected. It is deferred to AI-10.3
with the rest of cross-region validation. If the reviewing integration decides it belongs at message
scope after all, the class to use is the duplicate class, never `ErrMalformed`.

### 3.3 One rule set, two entry points

Both payloads implement `validate(at Path) *Violation`, and both constructors call it. AI-06
`decision.md` § 7.2 is the reason this is one function called twice rather than two implementations,
and AI-06.4's leg 3 is what proves the message boundary reaches it.

---

## 4. Decision — argument bytes are checked for syntax, and the asymmetry with AI-08 is right

Doc 0002 asks for this check on AI-09.1's argument bytes and pointedly not on AI-08.1's schema bytes.
AI-08's `design.md` § 8 recorded both sides and declined to land it there, so this milestone reads
that paragraph before deciding rather than after.

**Decision: `NewToolCall` requires non-empty argument bytes to be syntactically well-formed JSON,
and reports `ErrMalformed` at `arguments` when they are not.** The documented encoding is JSON, and
it is documented on the constructor rather than implied.

**Four reasons, in the order that carries.**

1. **AI-04 already committed the package to it, in prose.** `ErrMalformed`'s GoDoc reads: *"Register
   § 6.3: argument bytes that are not well-formed are decidable from the request alone. Bytes that
   are well-formed but do not satisfy a tool's schema are neither this nor a provider failure."* The
   class was landed with this exact case as its citable example. Declining the check would leave
   `ErrMalformed` with a documented justification no code uses.
2. **The provenance argument is AI-08's own, and it points here.** Its § 8 says argument bytes are
   *"produced by a model and reassembled from a stream, so a malformed reassembly is a real and
   recurring defect class that AI-30 must catch"*, while schema bytes are *"supplied by the caller as
   a literal in their own source"*. Those are different provenances with different failure rates. A
   malformed schema fails once, on the developer's machine, on the first run. A malformed argument
   payload fails in production, on a stream that dropped a frame, and the value that reaches the
   contract is the one place a truncation is still cheap to find. **The asymmetry is not an
   inconsistency; it is the two provenances being treated differently on purpose.**
3. **The canonicalization in § 5 needs the encoding anyway.** `R-ATM-003` requires one canonical
   empty form that a consumer can decode. Choosing that form is choosing an encoding. Having chosen
   it, refusing to check the same encoding on the neighbouring input would be an inconsistency
   *within one constructor*, which is worse than one across two milestones.
4. **It makes an invariant stateable.** With the rule, *every* constructible tool call's argument
   bytes are well-formed JSON — including the no-argument case. AI-26 and AI-30 can decode without a
   guard clause, and the "parse failure on empty input" bug class doc 0002 names is unreachable
   rather than merely unlikely.

**The conceded cost, stated plainly.** The contract now names an encoding, so a future provider whose
tool arguments are not JSON would be a breaking change rather than an additive one. That is the cost
AI-08 declined to pay for schemas and it is real. It is accepted because every provider in doc 0001's
register carries tool arguments as JSON, because `V-REQ-17`'s own noun is *argument bytes* with
byte-equality as the contract — a statement about fidelity, not about opacity to syntax — and because
the alternative buys generality for a provider nobody has named at the price of shipping the defect
class AI-30 is required to catch.

**Where the boundary sits, restated so no adapter author re-litigates it.** Syntax is checked;
*meaning* is not. Well-formed bytes that no declared tool would accept construct successfully
(`S-ATM-015`). That is the register's trap-1 line — bytes versus meaning — and this decision does not
move it.

---

## 5. Decision — the ordinal is derived from position, and lives nowhere else

**Where it lives: nowhere.** There is no ordinal field on `ToolCall`. The ordinal of a call is its
index among the tool calls of a content sequence, computed on read by:

```go
// ToolCalls returns the tool calls of a content sequence in content order.
// The index of a call in the result is its ordinal.
func ToolCalls(content []Part) []ToolCall
```

`explore.md` § 5 walks the three candidates. The argument that decides it is AI-06's own, one level
up: *"[Part] holds no kind field. [Part.Kind] asks the payload, so the discriminator and the payload
cannot disagree — there is only one of them."* An ordinal stored at construction is a second copy of
a fact the sequence already holds, and a part is placed in a message **after** it is built, so the
two copies are guaranteed to be writable independently. Doc 0002 warned about exactly this and the
answer is the one the package already made for the kind.

### 5.1 Consequence 1 — a lone call has no ordinal, and that is correct

`ToolCall` on its own answers `ID()`, `Name()` and `Arguments()` and nothing about position, because
position is not a property it has. A consumer that needs the ordinal needs the sequence, which is
true of the thing being modelled: `V-STR-21` defines the ordinal as *"the position of a tool call
among the calls of one response"*.

This is **not** in tension with AI-18 and AI-30 carrying an ordinal as a field. `V-STR-21` names both
of those as **restatements** of this term rather than definitions of it. A stream event is not inside
a sequence a reader can index — it arrives alone, over time — so the ordinal has to travel with it,
and the value it carries is the one this derivation would compute over the assembled content. The
place where the two could drift is the adapter, and AI-30.5 is where that is proven.

### 5.2 Consequence 2 — the derivation is a package-level function, not a method

Doc 0002's charter phrases every test in terms of a **message**, and the obvious shape would be
`func (m Message) ToolCalls() []ToolCall`. This design does not use it, for two reasons.

- **Ownership.** `message.go` is AI-05's, closed, and this wave is merging three milestones against
  neighbouring files. AI-09 owns no line of it.
- **Generality, which is the better reason.** AI-10 will walk a request's messages, AI-30 will
  assemble content from a stream, and neither holds a `Message` at the point it needs the ordinal.
  Both hold `[]Part`. A function over a content sequence serves all three call sites; a method on
  `Message` serves one.

AI-06 `decision.md` § 4.3 objected to free functions — `ai.TextOf(p)` — and the objection does not
reach here. It was about **two call shapes for one question about one part**: a reader who has a
`Part` should have exactly one way to ask what it holds. A sequence is not a `Part`, has no method
set of ours, and there is no competing shape for the question. `ai.ToolCalls(msg.Content())` reads as
what it is.

### 5.3 Totality

The derivation never fails and never panics. A part of another kind is skipped; a part that was never
constructed carries no payload, so the accessor answers `false` and it is skipped too. An empty or
nil sequence yields an empty result. `R-ATM-006`'s last clause pins all three, because a derivation
that panicked on the zero part would be a new way to reach the defect AI-06 closed.

---

## 6. Decision — a tool failure is a state, not an error, and two constructors say so

`V-REQ-18` is unusually explicit: *"A result that reports failure is not a `V-FAIL-01` and not a
`V-FAIL-05`; it is ordinary content."* So the failure indication is a field on a successfully
constructed value, and the only question is how a caller sets it.

| Shape | Verdict |
| --- | --- |
| `NewToolResult(callID, content string, failed bool)` | Rejected. `NewToolResult(id, out, true)` at a call site is unreadable, and a bare boolean parameter is the classic place a caller inverts a meaning silently |
| `NewToolResult(...)` plus `result.WithFailure()` | Rejected. A second construction step means a valid-looking value exists between them, which is the shape AI-06 spent a milestone removing |
| A failure *reason* vocabulary | Rejected as out of scope. Layer 1 cannot know why a tool failed; the tool's own output is the content, and a taxonomy here would be Layer 2's policy wearing a Layer 1 type |
| **Two constructors: `NewToolResult` and `NewToolFailure`** | **Chosen.** The call site names the outcome, both run the identical rule, and `Failed()` reads the state back |

`Failed()` rather than `IsError()` or `Err()`: the word *error* in Go names a value a caller must
handle, and this is precisely the thing that must **not** be handled — it is content to be sent to
the model. `V-REQ-18`'s own word is "failure".

**Nothing about this reaches AI-04.** No sentinel is defined, matched, or wrapped for it, and
`NewToolFailure` returns a nil error on the happy path exactly like its sibling.

---

## 7. Deliberate omissions

| Omitted | Why, and who could add it |
| --- | --- |
| A byte ceiling on argument bytes | `MaxTextLen` exists because `V-REQ-24`'s reasoning covers a value a human typed and a request must be decidable alone. Argument bytes are model-produced; their size is a function of the model's output limit, which Layer 1 cannot know (`V-OUT-14`, no model catalog). AI-08 bounds a tool *name* and does not bound schema bytes, and this follows the neighbouring precedent. A milestone with a real case may add it additively |
| A byte ceiling on result content | Same reasoning; the producer is a tool, and Layer 2 schedules tools |
| A shape rule on the call's tool name | § 3.1 — cross-region, AI-10.3 |
| Uniqueness of call identities | § 3.2 — collection rule, AI-10.3 |
| A correlation helper pairing results to calls | § 2.3 — `V-OUT-02`, Layer 2 |
| A failure-reason vocabulary | § 6 |
| A parser for either exported payload type | The register gives neither a text form. A renderer that round-trips is a supplied identity wearing a rendering — `MessageID`'s own recorded reason |

---

## 8. Redaction, and why it is in this milestone rather than AI-36

`Part.String()` names the kind and never the payload, and `Part.GoString()` exists *"so the redaction
posture covers every fmt verb rather than the three a reader thinks of"*. That posture is complete
for `Part` — and this milestone is the first to export a payload **type**, which reopens it: a
consumer holding a `ToolCall` and writing `log.Printf("%v", call)` would print the unexported fields
through reflection, and those fields are argument bytes a model produced from a user's prompt.

`ToolCall` and `ToolResult` therefore carry `String()` and `GoString()` that name the type and the
call identity's *presence* — never the identity, the name, the arguments or the content. This is
`V-FAIL-13`'s posture applied where AI-06 put it: on the type, not on anyone's discipline. It is
recorded as a case **discovered during** AI-09.1 and appended to the leaf's test list per doc 0002's
leaf anatomy, rather than smuggled in unproven.

It is a deliberate divergence from AI-08's `Tool`, which has no renderers. The difference is
provenance again: a schema and a tool name are the caller's own literals, and argument bytes are
model output derived from a prompt.

---

## 9. File layout

| File | Contents | Forecast |
| --- | --- | --- |
| `src/ai/tool_call.go` | File banner; `emptyToolArguments`; `ToolCall` with `kind`/`validate`; `NewToolCall`; `Part.ToolCall`; `ID`/`Name`/`Arguments`; `String`/`GoString`; `ToolCalls` | ~150 lines including GoDoc |
| `src/ai/tool_result.go` | File banner; `ToolResult` with `kind`/`validate`; `NewToolResult`; `NewToolFailure`; `Part.ToolResult`; `CallID`/`Content`/`Failed`; `String`/`GoString` | ~120 lines including GoDoc |
| `src/ai/content_part.go` | **Appended at three points only**: two constants + `partKindEnd`, two table entries, two documented lines | +12 lines |
| `src/ai/content_part_registry_test.go` | **Appended only**: two witnesses × three legs | +34 lines |
| `src/ai/tool_call_test.go` | `package ai_test`. AI-09.1 and AI-09.2 | ~280 lines |
| `src/ai/tool_result_test.go` | `package ai_test`. AI-09.3 | ~190 lines |

**Two production files, split on the leaf boundary**, which doc 0002's split trigger 4 calls the
PR-chain boundary. `ToolCalls` lives in `tool_call.go` rather than a third file because it is one
function over the type declared above it.

**Nothing else is touched.** `validation.go`, `role.go`, `message.go`, `text_content.go` are
read-only, and every AI-07 and AI-08 file is absent from this branch by construction. The
`content_part.go` and guard edits are pure appends so that the concurrent AI-07 merge is a
three-way append rather than a conflict — and AI-06.4's guard is what catches a merge that got it
wrong, in either direction.

**The lint gotchas, recorded because each has already cost this wave a run.**

1. revive's `package-comments` treats a comment block attached directly above `package ai` as the
   package comment and rejects a second one. Both new files separate their banner from the `package`
   clause with a blank line, the shape every landed file uses.
2. `staticcheck`'s `ST1018` (Trojan Source) fires on literal bidirectional control characters in a
   source literal. The non-ASCII fixtures in `R-ATM-008`'s round-trip table are built from escape
   sequences, never from literal bytes.
3. `make lint` runs before every commit, not before the last one.

---

## 10. How a red step is taken

AI-04's convention, restated because every item follows it:

1. Write the one test.
2. Add the **narrowest declaration** that makes it compile and the assertion fail — a constructor
   returning the zero value with a nil error, an accessor returning `false`, a derivation returning
   `nil`. Never one that could pass.
3. Run, and record the failing output verbatim. **A compile error is not the red step**; it is the
   state before it.
4. Implement minimally, run, record green.
5. Refactor while green.

Four items are at particular risk of a stub that passes for the wrong reason, and are handled
explicitly:

- **AI-09.1 item 2** (byte fidelity). A stub `Arguments()` returning the stored value passes
  trivially. The red step returns `nil`, and the fixture's key order and interior whitespace are
  chosen so that a *canonicalizing* implementation — the failure this item exists to catch — fails
  the same assertion.
- **AI-09.1 item 4** (canonical empty form). A stub that stores what it was given passes the
  "constructible" half and fails the "one canonical form" half, so the assertion is on the equality
  of two differently-supplied empties **and** on the form being decodable.
- **AI-09.2 item 1** (ordinal). A stub returning the stored slice would pass a self-comparison. The
  red step returns `nil`, and the assertion compares against the **caller's** interleaved order
  across repeated reads.
- **AI-09.3 item 3** (failure distinguishable). A stub `Failed()` returning `false` passes the
  success case. The red step is written on the failure case first.

## 11. Test plan

| # | Leaf item | Test function | What fails if the behavior regresses |
| --- | --- | --- | --- |
| 1 | AI-09.1.1 | `TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage` | The walking skeleton: construct, place, read back all three |
| 2 | AI-09.1.2 | `TestToolCall_ArgumentBytes_PassThroughByteIdentically` | Non-alphabetical keys, irregular interior whitespace, `bytes.Equal`; plus mutation of the caller's buffer and of the returned slice |
| 3 | AI-09.1.3 | `TestNewToolCall_BrokenConstructionRules_FailWithTheDocumentedSentinels` | Empty identity, empty name, four malformed argument shapes; asserts class, position, first-failure order, determinism, no leaked bytes, and that well-formed-but-schema-violating bytes are accepted |
| 4 | AI-09.1.4 | `TestNewToolCall_AbsentArguments_NormalizeToOneCanonicalDecodableForm` | nil and zero-length agree; the form decodes; a supplied non-empty value is not replaced |
| 5 | AI-09.1.5 *(appended)* | `TestToolCall_Rendering_NeverReproducesItsPayload` | Three fmt verbs against a call whose arguments carry a recognizable secret |
| 6 | AI-09.2.1 | `TestToolCalls_InterleavedContent_AreObservableInOrderWithStableOrdinals` | Interleaves other kinds; repeated reads; other kinds absent |
| 7 | AI-09.2.2 | `TestToolCalls_MessageCopyAndReadback_PreserveEveryOrdinal` | Value copy, mutation of the returned sequence, and the total cases |
| 8 | AI-09.3.1 | `TestToolResult_CorrelationAndContent_ReadBackFromAMessageInAnExternalPackage` | The walking skeleton for the second kind |
| 9 | AI-09.3.2 | `TestToolResult_CorrelationIdentity_RoundTripsExactlyIncludingSyntheticOnes` | Synthetic, punctuated and non-ASCII identities; equality against the originating call's identity; the empty-correlation rule |
| 10 | AI-09.3.3 | `TestToolResult_ReportedFailure_IsDistinguishableAndIsNotAnError` | Failure constructs without error, reads back as failed, and is distinguishable from a success with identical correlation and content |
| 11 | AI-09.3.4 *(appended)* | `TestToolResult_Rendering_NeverReproducesItsPayload` | As item 5, for the second kind |

The AI-06.4 guard is not a test function of this milestone; it is the landed guard, extended with two
witnesses. Its bite is recorded in `tasks.md` for each kind as the five-step procedure is completed.
