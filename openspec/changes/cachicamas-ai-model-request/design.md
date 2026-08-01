# Design — the normalized request core

> **Change**: `cachicamas-ai-model-request`
> **Milestone**: AI-10 · **Nodes**: AI-10.1 … AI-10.6, all `[leaf]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-model-request/spec.md`, the register, AI-04 … AI-09 as landed
> **Owns**: every Go spelling, the documented rule order, the three cross-region dispositions, the generation-option admission table, and the seams left for AI-11 and AI-12

---

## 1. What this design owns, and how to read its status marks

AI-10 is worked as **two chained halves on a leaf boundary**, because it is the largest composition in Layer 1 and two earlier attempts died on transcript size. This document plans all six leaves so the resuming agent inherits decisions rather than re-deriving them.

Every section carries one of two marks:

- **`[landed]`** — implemented, tested, and green in this branch. The shape below is the shape in the tree.
- **`[provisional]`** — decided here, **not implemented**. The next agent may only change it by recording why, in this file, before writing the test. Absent a recorded reason, it implements what is written.

| Section | Leaf | Mark |
| --- | --- | --- |
| § 2 the request shape | AI-10.1 | `[landed]` |
| § 3 the system instruction shape | AI-10.2 | `[landed]` |
| § 4 the documented rule order | AI-10.1/.2 slots landed; .3/.4 slots reserved | mixed |
| § 5 role versus content kind | AI-10.3 | `[provisional]` |
| § 6 orphan tool results | AI-10.3 | `[provisional]` |
| § 7 duplicate tool-call identities | AI-10.3 | `[provisional]` |
| § 8 generation-option admission | AI-10.1 | `[landed]` |
| § 9 validation before I/O | AI-10.4 | `[provisional]` |
| § 10 whole-request round trip | AI-10.5 | `[provisional]` |
| § 11 immutability and equality | AI-10.6 | `[provisional]` |
| § 12 the seams for AI-11 and AI-12 | — | `[provisional]` |

**No file outside `backend/agent/src/ai/` and this change's directory is edited by the landed half.** `openspec/specs/ai-contract-vocabulary/spec.md` is read and cited, never modified; the one gap found is reported in `proposal.md`.

## 2. The request shape `[landed]` — AI-10.1

File: `backend/agent/src/ai/request.go`. Tests: `backend/agent/src/ai/request_test.go`, package `ai_test`.

```go
type Request struct {
    model    string
    messages []Message
    system   SystemInstruction
    hasSystem bool
    // one field pair per generation option; see § 8
}

func NewRequest(model string, messages []Message, opts ...RequestOption) (Request, error)

func (r Request) Model() string
func (r Request) Messages() []Message
func (r Request) SystemInstruction() (SystemInstruction, bool)
func (r Request) MaxOutputTokens() (int, bool)
func (r Request) Temperature() (float64, bool)
func (r Request) TopP() (float64, bool)
func (r Request) StopSequences() ([]string, bool)
func (r Request) String() string
func (r Request) GoString() string
```

### 2.1 Why required regions are parameters and optional regions are options

Three candidates were weighed in `explore.md` § 3.1. The selected one is functional options, and the two properties that selected it are worth restating where the shape is:

1. **Absence is structural.** An option that was never applied leaves its `has…` companion false. There is no sentinel value meaning "unset" anywhere in this file, so "temperature unset" and "temperature zero" are different requests without anyone remembering that `-1` means something. `V-MET-11`'s distinction, applied on the request side.
2. **It is AI-12's seam, cut without AI-12 being built.** A per-request override is the same `RequestOption` applied again; copy-on-write rebuild is one added method, `Request.With(...RequestOption) (Request, error)`, which reshapes no existing signature. § 12.2.

### 2.2 The option type is sealed

```go
// RequestOption is one optional region or generation option of a request.
type RequestOption func(*requestDraft)
```

`requestDraft` is unexported. A function type whose parameter type is unexported cannot be written by a consumer in another package, so the set of options is exactly this package's constructors — a closed vocabulary enforced by the compiler rather than by review. This is the same sealing move AI-06 made with `partPayload`, one dimension smaller.

`NewRequest` builds a zero `requestDraft`, applies every option in the order given, then validates the draft and freezes it into a `Request`. Options themselves are **total**: they never fail, never validate, and never allocate a violation. Every rule runs once, in one place, in the documented order (§ 4). An option that could fail would be a second validation site, which is precisely the "constructor that checks and a boundary that does not" failure `decision.md` § 7.2 names.

Applying the same option twice is **last-wins**, deliberately and documented: it is what makes AI-12's override work by re-application rather than by a new mechanism.

### 2.3 The option constructors landed by AI-10.1 and AI-10.2

```go
func WithSystemInstruction(system SystemInstruction) RequestOption   // AI-10.2
func WithMaxOutputTokens(tokens int) RequestOption                   // AI-10.1
func WithTemperature(temperature float64) RequestOption              // AI-10.1
func WithTopP(topP float64) RequestOption                            // AI-10.1
func WithStopSequences(sequences ...string) RequestOption            // AI-10.1
```

There is deliberately **no** `WithSystemText(string)`. The ergonomic single-segment path is `NewSystemText`, which returns a value and an error, because a segment has construction rules (`R-AMR-007`) and an option that cannot fail must not carry rules. Two options for one region would also have made `R-AMR-006`'s indistinguishability a property of which option a caller picked, rather than of the value.

### 2.4 Copy in, copy out

`NewRequest` clones the message slice and the stop-sequence slice on the way in; `Messages()` and `StopSequences()` clone on the way out. Two mechanisms, not one — `message.go` records why, and the Go-specific trap it names (a variadic call with a slice spread does not copy) applies identically here.

### 2.5 Redaction `[landed]` — `R-AMR-017`

```go
func (r Request) String() string {
    // request(model, 2 messages, system[1], maxOutputTokens, temperature)
}
func (r Request) GoString() string { return r.String() }
```

The rendering names **which regions are present and how many elements each ordered region holds**, and nothing else. Not the model identity — a model name is caller data reaching a log, and the register places recognition outside this layer anyway, so there is no diagnostic value in it that a caller holding the request cannot get from `Model()`.

`GoString` is not optional and not a nicety. Without it, `%#v` falls back to reflection and prints every unexported field, which is exactly the leak AI-07 found on `Reasoning` and AI-09 landed against on `ToolCall` and `ToolResult`. The request carries all of those payloads at once.

The leak test is a table over **four verbs × six regions**: a distinct secret is planted in the model identity, a system segment, a message's text, a tool name, tool-call arguments and a stop sequence, and no secret may appear in any output. It runs against `Request`, `SystemInstruction` and `Segment` alike.

## 3. The system instruction shape `[landed]` — AI-10.2

File: `backend/agent/src/ai/system_instruction.go`. Tests: `system_instruction_test.go`, package `ai_test`.

```go
type Segment struct{ text string }

func NewSegment(text string) (Segment, error)
func (s Segment) Text() string
func (s Segment) IsZero() bool
func (s Segment) String() string     // "segment"
func (s Segment) GoString() string

type SystemInstruction struct{ segments []Segment }

func NewSystemInstruction(segments ...Segment) (SystemInstruction, error)
func NewSystemText(text string) (SystemInstruction, error)
func (i SystemInstruction) Segments() []Segment
func (i SystemInstruction) Len() int
func (i SystemInstruction) IsZero() bool
func (i SystemInstruction) String() string   // "system(2 segments)"
func (i SystemInstruction) GoString() string
```

### 3.1 Why segments exist at birth

Doc 0001 § 3.2 lists the flat system instruction **first** among the things that must change before an adapter exists, and gives the reason in one line: *a flat system instruction has nowhere to put a cache boundary*. In the retired plan, adding one was a **breaking change** — the field's type changed under every caller. Doc 0002 prices the alternative at zero: "a single-segment request is the common case, and the ergonomic constructor for it is part of this milestone's deliverable."

So the cost of segments is one convenience constructor, paid now, against a breaking change paid later. That is the whole argument, and it is why `NewSystemText` exists in the same commit as `NewSystemInstruction`.

The second thing segments dissolve is subtler and is not about caching at all. A flat string makes `""` mean both *absent* and *empty*, so a request that meant to carry no instruction and one that carries an empty one are the same value. Segments make absence structural (`hasSystem` false) and make an empty segment **unrepresentable** (`NewSegment` rejects it). "One empty segment" is therefore not merely equal to absence — it cannot be constructed.

### 3.2 Why zero segments is a construction failure

`NewSystemInstruction()` with no arguments fails with `ErrEmpty` at `system`. It would have been easy to let it produce an empty instruction and treat that as absence, and that is the trap: it gives the package **two** spellings of absence — the option unapplied, and the option applied with an empty value — and every later reader has to know both. One spelling, enforced at construction.

### 3.3 Why whitespace-only text is rejected but padding is preserved

A segment of spaces places no text into the instruction while occupying an ordinal that `V-REQ-19` says is individually markable — a caller could mark a cache boundary on nothing. So whitespace-only fails with `ErrEmpty` at `text`.

That is a **rejection criterion, never a normalization**. `NewSegment("  be terse.  ")` succeeds and `Text()` returns the padding intact. An adapter concatenating segments needs the caller's own separators; a constructor that trimmed would silently change the prompt, and prompt text is the one thing in this package that must survive byte-exact.

### 3.4 `Segment` is comparable; `SystemInstruction` and `Request` are not

`Segment` holds one string, so `==` is defined for it and is the natural way a test asserts round-trip equality of a segment. `SystemInstruction` and `Request` hold slices, so Go defines no `==` for them. This is not an oversight to fix in AI-10.6 — it is the reason AI-10.6's equality is **region-wise readback equality** (§ 11.2) rather than a comparison operator.

## 4. The documented rule order

`V-FAIL-04` requires a documented order, and AI-04 makes the order **data** — a slice of `Rule` values passed to `FirstFailure` — rather than control flow, so a reviewer reads which rule wins instead of tracing it. Request validation follows suit.

The full intended order, with the leaf that lands each slot. Slots marked `[provisional]` are **inserted at their numbered position** by AI-10.3, which is why the landed half leaves the generation options last rather than adjacent to the regions they belong to.

| # | Rule | Class | Position | Leaf |
| --- | --- | --- | --- | --- |
| 1 | model identity is non-empty and not whitespace-only | `ErrEmpty` | `model` | AI-10.1 `[landed]` |
| 2 | at least one message | `ErrEmpty` | `messages` | AI-10.1 `[landed]` |
| 3 | each message was constructed | `ErrEmpty` | `messages[i]` | AI-10.1 `[landed]` |
| 4 | each message's content, via AI-06's validator | AI-06's | `messages[i].content[j]…` | AI-10.1 `[landed]` |
| 5 | an applied system instruction was constructed | `ErrEmpty` | `system` | AI-10.2 `[landed]` |
| 6 | role versus content kind | `ErrMisplaced` | `messages[i].content[j]` | AI-10.3 `[provisional]` |
| 7 | tool-call identities are unique across the request | `ErrDuplicate` | `messages[i].content[j]` | AI-10.3 `[provisional]` |
| 8 | every tool result correlates to a call in the request | `ErrUnresolvedReference` | `messages[i].content[j]` | AI-10.3 `[provisional]` |
| 9 | an applied tool choice validates against the tool set | AI-08.3's three | AI-08.3's three | AI-10.3 `[provisional]` |
| 10 | generation option bounds | `ErrOutOfRange` / `ErrEmpty` | § 8.1's positions | AI-10.1 `[landed]` |

### 4.1 Three ordering decisions, each with a defensible alternative

- **Structure before parameters (rules 1–9 before rule 10).** A caller who has both a malformed message list and a negative temperature is told about the message list. The shape of the call is the more fundamental fact, and a caller who fixes it will re-run and be told about the temperature; the converse ordering makes them fix a number in a request that could never have been sent. This is the same reasoning AI-08.3 used to put its rule 2 before its rule 3.
- **Content rules (4) before cross-region rules (6–8).** A part that never passed its own constructor has no kind and no identity, so a cross-region rule reading either would be reading a zero value. Rules 6–8 may therefore assume every part is individually valid, which is what keeps them short.
- **Uniqueness (7) before correlation (8).** § 7 records why this is a precondition and not a preference: rule 8 resolves a result to a call *by identity*, and resolution against a non-unique key is not resolution.

### 4.2 The `validateContent` prefix — the seam, consumed

Rule 4 is one line:

```go
validateContent(Path{AtIndex("messages", i)}, messages[i].Content())
```

`content_part.go` documents that exact call — *"AI-10 holds a request, whose content sits one level deeper, and will pass `AtIndex("messages", i)` so the same failure renders as `messages[2].content[0]`"* — and AI-06.3 item 3 pinned the request-shaped prefix before a request existed. `decision.md` § 7.2 states the principle as **"one rule set, two callers"** and names the failure mode of the alternative: *a constructor that checks and a boundary that does not*.

**This change writes no second content validator.** `request.go` holds no per-kind logic at all: no text-length rule, no reasoning-token rule, no tool-call argument rule. Every one of those runs because rule 4 called AI-06's function, and every kind added after this milestone is validated at request depth on the day it is added, with no edit here. `request_test.go` asserts the **composed position**, not merely that a failure occurred, because the position is the only observable proof the prefix was passed rather than dropped.

## 5. Disposition 1 `[provisional]` — role versus content kind (AI-10.3)

`explore.md` § 2.3 lists this as one of three questions handed forward by name. It is decidable only with a request in hand, because it relates a message's role to its content's kind, and nothing below AI-10 holds both under a rule.

### 5.1 The table

| Kind | `user` | `assistant` | `tool` |
| --- | --- | --- | --- |
| `text` | permitted | permitted | forbidden |
| `reasoning` | forbidden | permitted | forbidden |
| `tool_call` | forbidden | permitted | forbidden |
| `tool_result` | forbidden | forbidden | permitted |

Implemented as one `var` — a table keyed by role holding the permitted kinds — so a later loosening is one cell, and the test is one row per cell, twelve rows, asserted in **both** directions.

### 5.2 Why strict rather than permissive

The argument is doc 0001 § 3.3 row 4: **an adapter maps *from* the neutral shape.** A neutral shape with two legal placements for one kind makes every adapter, forever, handle both — and the ones that handle only the placement their author happened to see are the ones that break on the second caller. One legal placement per kind is one branch per adapter.

The rollback asymmetry decides the strictness: **loosening a cell later is additive** (a request that was rejected starts being accepted, and no caller notices), while **tightening one is breaking** (a request a caller shipped stops constructing). Deciding strictly now is the reversible direction.

The specific cells worth a sentence each:

- **`text` forbidden in `tool`.** A tool message's payload is a tool result; `role.go` says so, and a text part there is a caller who meant `NewToolResult`.
- **`reasoning` forbidden outside `assistant`.** Reasoning is the model's deliberation. A user did not produce it, and a tool did not.
- **`tool_result` forbidden outside `tool`.** This is the cell `role.go`'s `RoleTool` GoDoc hands here by name.

### 5.3 Why this needs an appended rule class

A `Reasoning` part in a user message is a **perfectly constructed part**. What fails is its placement. `explore.md` § 3.4 walks all six landed classes and shows each fails the fit: something is present (not `ErrEmpty`); the kind *is* a vocabulary member (not `ErrNotInVocabulary`); no bound is crossed (not `ErrOutOfRange`); the value is well-formed for its encoding (not `ErrMalformed`); it names nothing (not `ErrUnresolvedReference`); nothing repeats (not `ErrDuplicate`).

AI-04's own rule governs: the set *"is extended by appending a class in the pull request that needs it, never by a milestone defining a sentinel of its own"*. So AI-10.3 appends:

```go
// ErrMisplaced is the class for a value that is valid in itself and not
// permitted where it appears.
ErrMisplaced = errors.New("value is not permitted where it appears")
```

…and adds it to `ruleClasses`. The criterion is the one AI-04 used for `ErrDuplicate`: **the fix the consumer is being told to make is different** — "move it, or drop it", not "spell it differently" — and `errors.Is` is the only place a consumer can read that difference.

**Obligation this creates.** `validation_registry_internal_test.go` requires the external mirror in `validation_test.go` to be updated **in the same commit**. The guard says so when it fails. AI-10.3's first commit therefore touches three files or none.

## 6. Disposition 2 `[provisional]` — orphan tool results (AI-10.3)

### 6.1 The decision

**A tool result whose call identity appears in no tool-call part anywhere in the request is rejected** with `ErrUnresolvedReference` at the result's composed position.

The class fits verbatim: *"a value naming something the request does not declare"*. A tool result names a call. If the request declares no such call, the reference is unresolved. This is the same shape as AI-08.3's rule 3 — a tool choice naming an undeclared tool — one region over.

### 6.2 The converse is deliberately legal

**A tool call with no matching result constructs successfully.** A call awaiting its result is the ordinary mid-turn state: it is what the request looks like between the model asking for a tool and the caller running it. Rejecting it would make the most common intermediate request unrepresentable.

Repairing *orphaned calls* is a transcript invariant one layer up (`V-OUT-02`), and doc 0002's AI-09 out-of-scope clause scopes this milestone to the request-level fragment only.

### 6.3 What is deliberately **not** decided, and is pinned anyway

**Whether a call must appear *before* the result that correlates to it is unlanded.** Doc 0002 item 4 scopes the rule to existence "anywhere in the request"; ordering repair is `V-OUT-02`'s.

The non-decision is pinned by a test (`S-AMR-050`): a request whose result appears in an earlier message than its call **constructs successfully**. A deliberate non-decision that is not asserted is indistinguishable from an oversight, and the next person to read the code would close it by reflex.

## 7. Disposition 3 `[provisional]` — duplicate tool-call identities (AI-10.3)

**Tool-call identities are unique across the whole request. A repeat is rejected with `ErrDuplicate`, positioned at the second occurrence.**

AI-09's `design.md` § 3.2 deferred this question here and named `ErrDuplicate` as the class if the answer was "reject". It is.

The reason is not tidiness. **§ 6.1's rule resolves a result to a call by identity.** A resolution rule over a non-unique key resolves to a set, and "the result correlates to one of two calls" is not a fact an adapter can act on. Uniqueness is therefore a *precondition of a rule this milestone lands*, which is why it is decided here rather than left to a hardening milestone.

Position at the **second** occurrence, by index, matching `NewToolSet`'s precedent exactly — including its reason: the reported duplicate is always the lowest index whose identity repeats an earlier one, so the answer is deterministic, and the position is an index rather than the identity itself because an identity is caller data and a position reaches a log.

The scan is linear over the ordered messages and their ordered content, for `tool_set.go`'s reason: nothing in this package may let an unordered iteration decide anything.

## 8. Generation options — the admission table `[landed]`

`V-REQ-26` defines a generation option and states the admission test — *"an option that exists to satisfy one provider belongs in the escape hatch instead"* — but the register enumerates no members. AI-01's closing checklist required the **term**, not the list. So AI-10 owns the list, under a test the register already wrote. `proposal.md` records the recommended register amendment; **this change does not edit the register**.

### 8.1 Admitted — the v1 set

| Option | Go spelling | Why it is neutral | Bound checked here |
| --- | --- | --- | --- |
| maximum output tokens | `WithMaxOutputTokens(int)` / `MaxOutputTokens() (int, bool)` | Every provider caps generation length, under some name. The concept survives translation without loss. | strictly positive, else `ErrOutOfRange` at `maxOutputTokens` |
| temperature | `WithTemperature(float64)` / `Temperature() (float64, bool)` | Universal sampling parameter; every provider has it and means the same thing by it. | not negative, else `ErrOutOfRange` at `temperature`. **No upper bound** — § 8.3 |
| top-p | `WithTopP(float64)` / `TopP() (float64, bool)` | Nucleus sampling, universal, and universally on the same `(0, 1]` scale. | greater than 0 and not greater than 1, else `ErrOutOfRange` at `topP` |
| stop sequences | `WithStopSequences(...string)` / `StopSequences() ([]string, bool)` | Every provider supports stopping on caller-supplied strings. Providers differ on *how many* they accept — a per-provider cap, not a concept difference. | applied with at least one sequence, else `ErrEmpty` at `stopSequences`; every sequence non-empty, else `ErrEmpty` at `stopSequences[i]` |

Four options, each individually optional, each with a `has…` companion, so `Temperature() (0, true)` and `Temperature() (0, false)` are different requests.

### 8.2 Rejected candidates, each with the provider that fails the test

Named so the list is a decision rather than an omission. Every one of these is AI-12's escape hatch (`V-REQ-28`), and the recorded reason is what a future milestone must overturn to admit it.

| Candidate | Fails because | Owner |
| --- | --- | --- |
| `top_k` | Not universal: some providers expose it, others do not, and the ones that do disagree on whether it composes with top-p. An option a caller sets and half the adapters silently drop is worse than one they cannot set. | AI-12 escape hatch |
| frequency / presence penalty | One provider's pair of knobs with one provider's semantics. Neutral only in name. | AI-12 escape hatch |
| `seed` / deterministic sampling | Some providers accept it, none guarantee it, and the guarantee is the entire value of the parameter. A neutral option that promises nothing is a lie in the vocabulary. | AI-12 escape hatch |
| response format / JSON mode | Not a generation parameter but a structured-output contract, with per-provider shapes ranging from a flag to a schema. Modelling it as an option would smuggle a schema into `V-REQ-26`. | Deliberately unowned in Layer 1 |
| thinking / reasoning budget | Provider-specific in both name and unit, and it interacts with `V-REQ-09` reasoning content, which AI-07 already models on the response side. | AI-12 escape hatch |
| `n` / multiple candidates | Changes the *shape of the response*, not the generation. A request option that turns one response into many is not an option, it is a different call. | Deliberately unowned in Layer 1 |
| streaming toggle | Not a property of the request at all. Whether a call streams is which method is invoked, and `V-STR-*` owns it. | AI-13 … AI-18 |
| user / metadata identifiers | Abuse-tracking metadata, not generation. Provider-specific field names and retention semantics. | AI-12 escape hatch |

The single line that generalises the table: **an option is admitted when its meaning survives translation to every adapter unchanged.** Everything above fails that either by not existing everywhere, or by existing everywhere with different meanings.

### 8.3 Why temperature has no upper bound

Providers disagree: some cap it at `1.0`, others at `2.0`. A bound that is right for one is **a caller-contract failure this package invented** for the other — the request would be rejected here for a value that provider accepts, and the caller would have no way to express a legal call.

So the lower bound is landed (a negative temperature is meaningless everywhere) and the upper bound is deliberately absent. A too-high temperature is a **provider failure** and reports through AI-19, which is the same reasoning the register applies to an unrecognised model identity in its § 6.3 worked example. This is recorded in `proposal.md` as deliberately unlanded rather than forgotten.

## 9. Validation before I/O `[provisional]` — AI-10.4

### 9.1 Determinism

`R-AMR-014` at request scope is already structurally satisfied by § 4: the order is a `[]Rule` handed to `FirstFailure`, evaluated in slice order, with no map iteration anywhere in the path. AI-10.4's work is to **assert** it, not to build it — a request violating one rule from each of four regions, constructed many times, reports the identical class and position every time.

The one place determinism could still be lost is the cross-region scans of §§ 6–7. Both walk ordered messages and ordered content; neither may introduce a map for lookup speed. If a future profile demands one, the map may index but must not decide: the *reported* duplicate stays the lowest index in the ordered walk.

### 9.2 Totality

"A request that passes cannot contain an unconstructed part, a duplicate tool name, or an unresolvable tool choice" is a claim about *coverage of the region set*, and the honest way to assert it is one test per region asserting that region's own rule ran through the request path — proving the call happened, not re-testing the rule. The duplicate tool name is `NewToolSet`'s, which runs before a set can be handed to a request at all; that asymmetry is worth a comment in the test rather than a redundant rule.

### 9.3 The no-I/O guard

In the AI-00.3 style: compute the dependency closure of the request path's package and assert it contains no network and no filesystem package. `import_boundary_test.go` is the landed pattern to copy; it already exists in this package and needs extending rather than replacing.

The guard is what makes `V-REQ-22`'s *"validation happening after a socket is open is a different concept and a defect"* mechanical rather than aspirational.

## 10. Whole-request round trip `[provisional]` — AI-10.5

Lives in `backend/agent/src/agenttest/`, not in `ai_test`, because the milestone's acceptance is *readability from another package* and `agenttest` is the package that already proves it for AI-06.

The walk builds a request holding every registered part kind — text, reasoning **with its round-trip token**, tool call, tool result — plus system segments, tools, a tool choice and all four generation options; reads every region back through the exported surface alone; rebuilds a request from what it read; and asserts the rebuild equals the original under § 11.2's equality.

The **pin** is the load-bearing half: the walk's kind handling is driven from `PartKinds()`, so a kind registered without a readable accessor fails the pin. This consumes AI-06.4's registration rather than duplicating it, and it is the mechanism that keeps `V-REQ-06` true for kinds nobody has written yet.

Note for the implementer: `Reasoning`'s token is `([]byte, bool)` and must be compared by bytes, not by `%v`; and the rebuilt request's `MessageID`s **will differ**, which is why § 11.2 excludes identity from equality.

## 11. Immutability and equality `[provisional]` — AI-10.6

### 11.1 Immutability

Two directions, and they are two mechanisms rather than one:

- **Mutating what a reader returned.** `Messages()`, `StopSequences()`, `SystemInstruction().Segments()` each return a fresh slice per call. Assert by mutating the returned slice and re-reading.
- **Mutating what the constructor was passed.** `NewRequest` clones on the way in. Assert by mutating the caller's own slice after construction and re-reading. This is AI-05.3's property at request scope, and it is the one Go hides: the variadic-spread trap `message.go` documents.

### 11.2 Equality — region-wise readback equality

`Request` holds slices, so Go defines no `==` for it. The documented equality is therefore:

> Two requests are equal when, for every region, the values their accessors return are equal — `Model()` by string equality, `Messages()` element-wise by role and by content parts (which are comparable, per AI-06), `SystemInstruction()` by presence and then element-wise over comparable `Segment`s, tools and tool choice by their own accessors, and each generation option by value **and** by its presence flag.

**Message identity is excluded**, and the exclusion is not a convenience. `V-REQ-03` makes `MessageID` deliberately unforgeable and minted per construction, precisely so that two messages built from identical inputs are *distinguishable*. Including identity in request equality would make `R-AMR-015`'s round trip unprovable by construction, and would make `S-AMR-059` false for any two requests at all.

Whether this equality is **exported** as `func (r Request) Equal(other Request) bool` is AI-10.6's to decide, and the default recorded here is **yes**: AI-10.5's round trip needs it, AI-26 needs it, and a comparison every consumer re-derives is a comparison every consumer gets subtly wrong. If AI-10.6 exports it, `Message` and `SystemInstruction` need matching `Equal` methods for it to be readable rather than one large function.

## 12. The seams left open, by name

### 12.1 AI-11 — cache-boundary markers

**Nothing in the request shape changes when markers arrive.** A marker attaches to the things that are already ordered and individually addressable: `Segment`, `Tool` and `Message`.

The concrete seam this half leaves:

- `Segment` is a struct with one unexported field, not a string alias. Adding `marked bool` is a field, not a type change, and no signature moves.
- `NewSegment(text string) (Segment, error)` stays; AI-11 adds a sibling constructor or an option, and every existing caller compiles untouched.
- `Segments()` returns `[]Segment`, so a marker read is `segments[i].IsCacheBoundary()` and needs no new accessor on `SystemInstruction`.
- The request-level breakpoint **cap** (`V-REQ-24`, `ErrOutOfRange`) inserts into § 4's order as a new row after rule 5. The order is a slice; inserting a row is one line.
- The tools → system → messages **read order** AI-11.2 needs is already the order § 4 validates in, and already the order the regions are documented in.

`Segment.String()` renders `"segment"` with no marker state today; AI-11 may extend it to name the marker, because a marker is structural rather than payload.

### 12.2 AI-12 — overrides, rebuild, escape hatch

- **Per-request override** is the same `RequestOption` applied again. § 2.2's last-wins rule is landed and documented for exactly this.
- **Copy-on-write rebuild** is one added method, `func (r Request) With(opts ...RequestOption) (Request, error)`: read the frozen request into a `requestDraft`, apply, re-validate through the same rule slice, freeze. No existing signature changes, and validation stays single-sited.
- **The escape hatch** (`V-REQ-28`) is one more optional region and therefore one more `RequestOption` — `WithProviderExtension(namespace string, value …)`. It needs no request-shape change, and § 8.2's rejected-candidate table is the list of its first customers.

### 12.3 AI-26 — translation

AI-26 consumes `R-AMR-015`'s round-trip property, not a translation. Nothing is left for it here beyond the guarantee that every region is readable and every kind is registered.

## 13. Test strategy

- **Behavioral tests in `ai_test`.** External package, for AI-06's reason: the consumer this contract exists for is an adapter in a vendor package, and readability from outside is constitutive rather than incidental.
- **`src/agenttest/`** carries only AI-10.5's cross-package round trip. The landed half adds nothing there.
- **Naming** `Test<Subject>_<Behavior>_<Expectation>`, with a banner comment citing the leaf ID and, where the test pins a decision, the reason for the decision.
- **stdlib only.** No assertion library. Comparisons are `slices.Equal`, `bytes.Equal`, `errors.Is`, `errors.As`, and hand-written table loops.
- **Red before green, per item, recorded.** Go's compile-time typing means a red step needs the declaration to exist, so the pattern AI-06 recorded applies: land the narrowest thing that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.
- **Lint traps already paid for this wave**, and re-paid here: a file banner comment must be separated from `package ai` by a blank line (revive `package-comments`); literal bidirectional control characters in a test table trip `ST1018`, so such fixtures are built from escapes; `fmt.Sprintf("%s", x)` where `String()` exists trips `S1025`. `make lint` runs before every commit.
