# Design — roles and message identity

> **Change**: `cachicamas-ai-message-roles`
> **Milestone**: AI-05 · **Nodes**: AI-05.1 `[leaf]`, AI-05.2 `[leaf]`, AI-05.3 `[leaf]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-message-roles/spec.md`
> **Target**: `backend/agent/src/ai/role.go` + `role_test.go`, `backend/agent/src/ai/message.go` + `message_test.go`
> **Language**: Go 1.26.3, standard library only

---

## 1. What this document owns

AI-05 has no `[decision]` node, so this document carries both halves: *what* is decided and *how it is spelled in Go*. doc 0002 names no identifier for AI-05 and the register is forbidden from carrying one, so every type, constant, function and method name below is chosen here, once, with its reasoning.

Two of the choices are load-bearing beyond this milestone and are stated as reusable patterns rather than as local decisions: **§ 3** the closed-vocabulary pattern, which AI-07, AI-08 and AI-13 each reuse, and **§ 4** the content seam, which AI-06 inherits.

---

## 2. The surface

Complete. Nothing else is exported by this milestone.

```go
package ai

// ---- role.go — the closed role vocabulary (V-REQ-01) ----

type Role uint8

const (
	RoleUser Role = iota + 1
	RoleAssistant
	RoleTool
)

func Roles() []Role                    // the vocabulary, declaration order, fresh slice
func (r Role) String() string          // "user" | "assistant" | "tool" | "role(N)"
func ParseRole(name string) (Role, error)

// ---- message.go — the message (V-REQ-02) and its identity (V-REQ-03) ----

// The seam. AI-06 decides what a content part is.
type Content interface{ /* one unexported method */ }

type MessageID struct{ /* unexported */ }

func (id MessageID) IsZero() bool
func (id MessageID) String() string

type Message struct{ /* unexported */ }

func NewMessage(role Role, content ...Content) (Message, error)

func (m Message) ID() MessageID
func (m Message) Role() Role
func (m Message) Content() []Content
```

Three constants, four types, eight functions and methods. A consumer builds a message with one call and reads it with three.

### 2.1 Why each name

| Name | Chosen because | Rejected alternative, and why |
| --- | --- | --- |
| `Role`, `RoleUser`, … | `Role` is `V-REQ-01`'s own noun. The `Role` prefix on each constant is the standard-library shape (`http.StateNew`, `time.January` by package) and it keeps the constants readable at a call site in another package: `ai.RoleAssistant` says what it is without an import alias | A nested `role.User` package. It would put a domain noun in its own package before the domain has three of them, and ADR 0005 § D2 fixes `src/ai` as the Layer 1 package |
| `Roles()` | A function, not a variable, because a package-level `var Roles []Role` is mutable by any consumer — the exact defect `R-AMSG-001`'s second scenario tests for. The plural reads as the set | `AllRoles()`, `RoleValues()`. Both are longer and neither says more. A `var` is rejected on mutability, not on style |
| `ParseRole` | `Parse<Type>` is the standard library's own convention for the string→value direction (`time.ParseDuration`, `netip.ParseAddr`, `strconv.ParseInt`), and pairing it with `String()` makes the round-trip obvious | `RoleFromString`, `NewRole`. `NewRole` reads like a constructor for a *new* role, which is exactly what a closed vocabulary does not have |
| `Content` | It is the word `V-REQ-02` uses — "one role plus ordered **content**" — and it is deliberately **not** `ContentPart`. `V-REQ-04` **content part** is AI-06's term, and naming the seam after it would claim the thing AI-06 defines | `Part`, `ContentPart`, `Element`. `ContentPart` claims `V-REQ-04`; `Part` is the name AI-06 may well want for the real thing and a collision inside package `ai` is a rename across four milestones; `Element` says nothing |
| `MessageID` | The register's term is "message identity"; `MessageIdentity` is the literal spelling and nobody writes it twice. `ID` is Go's initialism and `revive` requires the capitalisation | `MessageIdentity` (long), `MsgID` (abbreviated for no gain), a bare `uint64` or `string` alias — rejected in § 5.2, because a caller can forge, reuse or compute any of them |
| `Message`, `NewMessage` | The register's noun, and the one constructor. `New<Type>` is the convention AI-04's `Invalid` deliberately broke for a call site written hundreds of times; a message is constructed once per message and reads better spelled out | `Msg`, `NewMsg`. Abbreviating the single most-cited noun in Layer 1 saves four characters and costs a lookup |
| `ID()`, `Role()`, `Content()` | Accessors named for what they return, which is the shape AI-10's request will mirror when it walks a message | Exported fields. An exported `Content []Content` field is mutable from outside and would make `R-AMSG-010` unimplementable — the whole of AI-05.3 |
| `IsZero()` | `time.Time.IsZero`'s exact shape, and the only way a consumer can tell a constructed message from a zero-value one | `Valid()`, `Set()`. `Valid` invites the reading that an identity can be invalid for some other reason; there is only one |

### 2.2 What is deliberately not on the surface

Each is plausible, each has no citable case, and AI-04 set the precedent for refusing them.

- **`RoleSystem`.** `V-REQ-19` gives the system instruction its own region of the request, segmented from birth and owned by AI-10.2; doc 0001 § 3.3 row 8 records that every provider takes it as a top-level shape an adapter renders. A `system` role would be a second way to say one thing. Adding a constant later is additive and the pin (§ 3) forces the table entry in the same commit; removing one after AI-40's freeze is not free.
- **`Role.IsValid()`.** Validity is checked at the one boundary a role enters through — `NewMessage` — and AI-10 will validate a message it was given, not a bare role. An exported predicate invites a caller to check and then not check, which is how a closed vocabulary becomes advisory.
- **A `Message.ContentLen()` or an index accessor.** `len(m.Content())` answers it, and a second read path is a second place for the copy contract to be got wrong.
- **Message equality.** AI-10.6 item 3 owns "the documented equality", defined over a request's regions. Two identities are comparable today, which is all `V-REQ-03` asks.
- **Anything on `Content` beyond the marker.** § 4.

---

## 3. The closed-vocabulary pattern — stated once, reused four times

AI-05 lands the first closed vocabulary in Layer 1. Three more follow: **AI-07** reasoning state, **AI-08** tool choice, **AI-13** finish reasons. doc 0002 says so directly in AI-05.1 item 4 — "the pattern every later closed vocabulary in this package reuses" — so this section is written as a procedure, not as a description of `Role`.

### 3.1 The four rules

**Rule 1 — the vocabulary is an integer defined type whose zero value is not a member.**

```go
type Role uint8

const (
	RoleUser Role = iota + 1   // iota + 1, never iota
	RoleAssistant
	RoleTool

	roleEnd // one past the last declared member; not a member, never exported
)
```

`iota + 1` is the whole of `S-AMSG-008`: a struct field of type `Role` that nobody set holds `Role(0)`, and `Role(0)` must be rejected exactly like `Role(99)`. A vocabulary starting at `iota` makes "never set" indistinguishable from its first member, which is the defect that makes a zero value look valid — `V-REQ-04`'s constitutive complaint, one contract over.

**Rule 2 — the table is an indexed slice, and it is the single source of rendering, parsing and validity.**

```go
var roleNames = []string{
	RoleUser:      "user",
	RoleAssistant: "assistant",
	RoleTool:      "tool",
}
```

Indexed by the constant itself, so a reader sees the pairing without counting positions. A **slice, not a map** — AI-04's `ruleClasses` established the rule and its reason: nothing in this package may let an unordered iteration decide anything, and a registry is where that temptation is strongest.

**Rule 3 — the enumeration walks the constant space, not the table.**

```go
func Roles() []Role {
	out := make([]Role, 0, int(roleEnd-roleFirst))
	for r := roleFirst; r < roleEnd; r++ {
		out = append(out, r)
	}
	return out
}
```

This is the rule that makes the pin bite, and it is the one that is easy to get backwards. An enumeration derived from `roleNames` would list exactly the members that have table entries, so a member declared without one would be *invisible* to every assertion over the enumeration — the pin would pass forever and would be decorative. Walking `roleFirst … roleEnd` derives the enumeration from the **declaration**, so a new constant appears in it the moment it is declared and takes its table entry with it or fails.

**Rule 4 — the pin asserts the round trip over the enumeration, and it is shown to bite.**

For every member the enumeration yields: its rendering is non-empty and equal to its own lowercase form, parsing that rendering returns the same member, and the member is accepted where the vocabulary is consumed. A member declared without a table entry renders as `role(N)`, which parses back to nothing, and fails two of the three.

The bite proof is not optional and not a thought experiment: declare a scratch member, record the failure output, remove it. doc 0002's guard grammar requires exactly this, and the pin's value is entirely in having done it.

### 3.2 What a later vocabulary changes, and what it must not

AI-07's reasoning state, AI-08's tool choice and AI-13's finish reason each substitute their own noun for `Role` and their own members for the three below. **Rules 1 and 3 are the load-bearing ones** and neither has a degree of freedom: `iota + 1` and an enumeration over the constant space. Rule 2's table may carry more than a name — AI-13's finish reasons may want a retryability disposition beside the rendering — and that is a widening of the row, not a change of shape.

One member of AI-13's vocabulary is a real exception worth flagging in advance: `V-MET-08` **unknown finish reason** is "the recorded outcome for a provider stop condition the vocabulary does not recognise", which means AI-13 has a member that is *reached* by a failed lookup rather than only by a successful one. That is a difference in what parsing does on failure, not in the pattern — the member is still declared, still tabulated, and still round-trips.

### 3.3 Rendering and parsing

```go
func (r Role) String() string {
	if name := roleName(r); name != "" {
		return name
	}
	return "role(" + strconv.FormatUint(uint64(r), 10) + ")"
}
```

`time.Month` renders an out-of-range value as `%!Month(42)` — deliberately ugly, so nothing downstream mistakes it for a name. `role(42)` is the same idea in this package's register: it identifies the value for a diagnostic reader and it does not parse back, which is `S-AMSG-013`. It is not caller data — a `Role` is an integer, and the only string in the failure path is the one `ParseRole` was given, which never reaches a message.

`ParseRole` is **exact**: no case folding, no trimming, no aliases, no provider strings. `V-REQ-01`'s last clause is "Not a provider's own role string — an adapter maps between them", so an adapter that receives `"Assistant"` from a vendor maps it; this package does not guess. Two rules, in AI-04's documented order:

```go
func ParseRole(name string) (Role, error) {
	parsed := roleWithName(name)
	if err := FirstFailure(
		func() *Violation { if name == "" { return Invalid(ErrEmpty, At("role")) }; return nil },
		func() *Violation { if parsed == 0 { return Invalid(ErrNotInVocabulary, At("role")) }; return nil },
	); err != nil {
		return 0, err
	}
	return parsed, nil
}
```

Empty first, because "you gave me nothing" and "you gave me something I do not know" are different facts with different fixes — and because it is the first place in Layer 1 where `FirstFailure`'s ordering is observable to a consumer rather than to a test. The offered string appears in neither failure: the position is `At("role")`, a constant this package wrote, which is what AI-04's redaction posture buys at every call site that inherits it.

---

## 4. The content seam

### 4.1 The shape

```go
// Content is the seam through which a message holds one element of its ordered
// content. AI-06 decides what a content part is; this milestone decides only
// that a message holds a sequence of them, in order.
type Content interface {
	isContent()
}
```

One method, unexported, returning nothing. That is the entire declaration, and every omission is deliberate.

### 4.2 What it leaves open, clause by clause

AI-06.1's closing checklist has four items. The seam must touch none of them.

| AI-06.1 item | Left open by |
| --- | --- |
| 1(a) — an adapter in another package can read a part's payload | The seam exposes nothing readable. Accessors may land on the interface, on concrete variants, or on neither with a walk; all three remain available |
| 1(b) — no value that skipped the constructor can validate | The seam validates nothing, and this milestone rejects no content element. `R-AMSG-009`'s second scenario pins that deliberately: AI-06.3 item 1 must be able to fail before it passes |
| 2 — the kind is derived from the payload | Neither exists here |
| 3 — the procedure for adding a kind, and its guard | Nothing is registered here, so AI-06.4 registers against an empty table |
| 4 — the accessor shape for kind and payload | No accessor exists |

### 4.3 The bypass, recorded rather than closed

An interface whose only method is unexported cannot be *implemented* from another package — but it can be **satisfied** from one, by embedding the interface:

```go
type part struct {
	ai.Content        // embedded, nil; promotes the unexported method
	label      string
}
```

AI-05's own external tests use exactly this, in four lines, and it is how `ai_test` obtains content elements without this package exporting a constructor for something it has not defined.

This is not a leak in the seam; it is the seam declining to answer a question that is not its. AI-06.3 item 2 says the seal may be "the compiler or validation, **whichever the AI-06.1 strategy chose**". Landing an unexported method that stopped the embedding route would take that choice; landing one that does not, and saying so in the seam's own GoDoc, hands AI-06 a known-open door with its dimensions written on it. A hand-rolled part built this way has a nil embedded interface, so any method AI-06 later calls on it panics — which is precisely why AI-06.3 item 1 must reject it at validation rather than trusting the compiler.

### 4.4 Why not the alternatives

| Alternative | Why not |
| --- | --- |
| `type Content any` | Accepts `42`. It is not a seam, it is the absence of one, and `NewMessage` would have no type to name |
| An interface with an exported method (`Kind()`, `String()`) | Decides AI-06.1 item 4's accessor shape, and an exported-method interface is implementable from anywhere — the retired defect **C1**, landed one milestone early |
| An opaque struct with unexported fields | Needs an exported constructor for the external tests to build two distinguishable values, and that constructor **is** AI-06.2's construction rules |
| Making `Message` generic in its content type | Viral: AI-10's request, AI-14's events and the provider interface would all carry the parameter, to defer one decision by one milestone |
| No content at all until AI-06 | AI-05.2 item 2 and the whole of AI-05.3 are unwritable, and AI-05 stops being a walking skeleton |

If a reviewer disagrees with this seam, the disagreement is with doc 0002's ordering of AI-05 before AI-06, and the response is its revert-and-record clause: a new node, its edge, and a dated amendment in this same PR.

---

## 5. The message

### 5.1 The shape

```go
type Message struct {
	id      MessageID
	role    Role
	content []Content
}

func NewMessage(role Role, content ...Content) (Message, error) {
	if err := FirstFailure(
		func() *Violation { if roleName(role) == "" { return Invalid(ErrNotInVocabulary, At("role")) }; return nil },
		func() *Violation { if len(content) == 0 { return Invalid(ErrEmpty, At("content")) }; return nil },
	); err != nil {
		return Message{}, err
	}
	return Message{id: mintMessageID(), role: role, content: slices.Clone(content)}, nil
}
```

Four properties, each answering a scenario:

- **A value, not a pointer.** A `Message` is data. Returning `*Message` would make a nil message a thing that exists, and it would put the aliasing hazard AI-05.3 exists to close back at every call site that copies one.
- **The zero `Message` on failure** (`S-AMSG-009`). A caller that ignores the error gets a message whose identity reports unset, rather than one that looks constructed.
- **Variadic content**, so `NewMessage(ai.RoleUser, part)` is the common case and `NewMessage(role, parts...)` is the general one. The second form is exactly the aliasing hazard § 6 covers.
- **The rule order is `role` then `content`**, documented on `NewMessage` and asserted by `S-AMSG-027`. It matches the declaration order of the parameters, which is the only ordering a reader will guess.

Nil content **elements** are accepted. That is not an oversight: AI-06.3 item 1 is the assertion that rejects an unconstructed part, and it needs AI-06.1's strategy to know what one is. `R-AMSG-009`'s second scenario pins the acceptance so it cannot be quietly tightened here.

### 5.2 Identity: minted, opaque, comparable

```go
type MessageID struct{ n uint64 }

var lastMessageID atomic.Uint64

func mintMessageID() MessageID { return MessageID{n: lastMessageID.Add(1)} }
```

**Minted, not supplied.** `V-REQ-03` calls identity "the stable handle by which one message is **distinguished from** another". Distinctness is not a property a caller-supplied handle has: two messages built from the same string are indistinguishable and the type cannot notice. Minting makes it structural. It also makes `S-AMSG-017` a real assertion — an identity computed on read, from the content say, would satisfy "comparable" and fail "does not change across reads".

**Opaque and comparable.** A struct with one unexported field. A `uint64` or `string` alias would be comparable too, and would also be forgeable, reusable and arithmetic — a caller could write `id + 1` and get another message's identity. `netip.Addr` and `time.Time` take the same shape for the same reason.

**Not defect C3, and the argument matters.** A package-level atomic counter is what **C3** was, and `V-STR-13` records the lesson: "the fix is not a smaller counter; it is putting the counter where the stream is." C3's contract was a statement about the counter's *value* — "every stream's first event carries 1, every stream is independently contiguous" — which a process-global counter cannot satisfy for the second stream in a process. `V-REQ-03` states no property of the value at all. It asks that two messages be distinguishable, and a monotonic process-wide counter satisfies that for every message in the process, not merely the first request's. The observable contract is `S-AMSG-018` (two messages differ) and `S-AMSG-021` (the rendering is diagnostic only), and neither would change if the counter were replaced by random bytes tomorrow.

The one thing the counter must be is **race-free**, because messages are constructed concurrently the moment Layer 2 exists. `atomic.Uint64` rather than a mutex, and `S-AMSG-020` proves it under `-race` rather than by inspection.

`MessageID.String()` renders `msg-7`, and `msg-unset` for the zero value. There is deliberately **no** `ParseMessageID`: an identity a consumer can reconstruct from text is a supplied identity wearing a rendering, and `S-AMSG-021` pins the absence.

---

## 6. Copy semantics — two mechanisms, not one

doc 0002 attaches an unusual note to AI-05.3: it is "the one whose absence produces the most confusing class of test failure in a streaming package", and it is what lets AI-21.6's request capture assert on history without defensive copying at the call site.

The hazard is specific, and it is Go's rather than the design's. A variadic parameter called with a slice spread does **not** copy:

```go
parts := []ai.Content{a, b}
msg, _ := ai.NewMessage(ai.RoleUser, parts...)  // msg.content aliases parts
parts[0] = c                                    // msg's first element is now c
```

Nothing in the language warns about it, every happy-path test passes, and the symptom appears later as content that changed with nobody writing to the message.

Two mechanisms close it, and they are genuinely two:

| Direction | Mechanism | Scenario |
| --- | --- | --- |
| In | `slices.Clone(content)` in `NewMessage` | `S-AMSG-031`, `S-AMSG-032` |
| Out | `slices.Clone(m.content)` in `Content()` | `S-AMSG-033`, `S-AMSG-034`, `S-AMSG-035` |

A design with only the first passes every construction test and fails the moment two consumers read the same message. A design with only the second passes every read test and fails on the spread above. The test list has two items because the design has two mechanisms, and each red step reproduces its own hazard rather than asserting immutability in the abstract.

`Roles()` allocates for the same reason (`S-AMSG-002`), which is why it is a function.

**What copy semantics does not promise, and must not be read as promising:** this milestone copies the *sequence*. Whether a content part's own payload can be mutated after construction is a property of the part, and the part is AI-06's. Saying otherwise here would be `V-REQ-07` sealing decided from outside AI-06.1 — the shortest route to the exact defect the seam exists to avoid.

---

## 7. File layout

| File | Contents | Forecast |
| --- | --- | --- |
| `backend/agent/src/ai/role.go` | File documentation for the vocabulary and the reusable pattern; `Role`; the constants and the terminator; `roleNames`; `Roles`; `String`; `ParseRole`; `roleName` / `roleWithName` | ~110 lines including GoDoc |
| `backend/agent/src/ai/role_test.go` | `package ai_test`. AI-05.1's four items | ~180 lines |
| `backend/agent/src/ai/message.go` | File documentation for the message and the seam; `Content`; `MessageID` and its minting; `Message`; `NewMessage`; the three accessors | ~140 lines including GoDoc |
| `backend/agent/src/ai/message_test.go` | `package ai_test`. AI-05.2's and AI-05.3's items | ~270 lines |

**Two production files, split on the leaf boundary.** doc 0002's split trigger 4 calls that boundary the PR-chain boundary, so the commits fall where a reviewer would cut them. The alternative — one `message.go` holding both — would put a vocabulary and a container in one file because they merged in one milestone, which is a reason that stops being true at AI-07.

**`validation.go` and `validation_test.go` are not touched.** AI-13 is being built concurrently against those files; the two milestones' file sets are disjoint by construction, and this milestone only reads them.

**`doc.go` is not touched.** It describes the package; a milestone's own contract documentation belongs on its declarations. `src/agenttest/` is not touched either: its own comment reserves the first cross-package readability proof for AI-06.2, and `ai_test` is already an external package.

---

## 8. How a red step is taken in a statically typed language

AI-04's convention, restated because every item below follows it and because it is the rule most easily lost:

1. Write the test.
2. Add the **narrowest declaration** that makes it compile and makes the assertion fail — a function returning the zero value, an accessor returning nil. Never one that could pass.
3. Run, and record the failing output. **A compile error is not the red step**; it is the state before it.
4. Implement minimally, run, record green.
5. Refactor while green.

A red step whose stub could plausibly satisfy any part of the assertion is not a red step. Two items here are at particular risk and are handled explicitly:

- **AI-05.2 item 2** (order round-trips). A stub `Content()` returning the stored slice passes trivially, so the red step stubs it to `nil` and the assertion is on order *and* length *and* the repeated-element case.
- **AI-05.3 items 1 and 2** (copy semantics). A stub that already copies is indistinguishable from the implementation, so the red step is taken against the *aliasing* implementation — the one anybody writes — and the recorded failure is the aliasing symptom itself, not a missing return.

The pin (AI-05.1 item 4) is exempt from red-first by doc 0002's leaf anatomy, and is still fully mechanical. Its bite proof is a scratch member declared without a table entry, its failure recorded, then removed.

---

## 9. Test plan

Nine functions, in the order they are written. Each carries a banner comment citing its leaf ID.

| # | Leaf item | Test function | What makes it fail if the behavior regresses |
| --- | --- | --- | --- |
| 1 | AI-05.1.1 | `TestMessage_EachVocabularyRole_ConstructsAndReadsBackExactly` | Walks `Roles()`, constructs one message per member, asserts the role reads back equal. The walking skeleton: role, message, content and identity all exist by the end of it |
| 2 | AI-05.1.2 | `TestMessage_RoleOutsideTheVocabulary_FailsWithTheClosedVocabularySentinel` | The zero role, one past the last, and a far-out value; asserts the sentinel, that no *other* sentinel matches, the position, and that the returned message has no identity |
| 3 | AI-05.1.3 | `TestRole_StringForm_RoundTripsThroughParseAndRender` | Round-trips every member; rejects case variants, padded forms, the empty string (with a *different* sentinel), and the `role(N)` rendering of a non-member; asserts the offered string is absent from the failure |
| 4 | AI-05.1.4 *(pin)* | `TestRole_DeclaredVocabulary_IsExhaustivelyTabulated` | Enumerates the declared members and asserts each renders, parses back, and constructs. Bites on a member declared without a table entry |
| 5 | AI-05.2.1 | `TestMessage_Identity_IsStableComparableAndDistinct` | Repeated reads agree; two identical messages differ; a zero message reports unset; 64 concurrent constructions under `-race` are all distinct |
| 6 | AI-05.2.2 | `TestMessage_ContentOrder_RoundTripsExactly` | Order, length, and the repeated-identical-element case, which a set- or map-backed implementation fails |
| 7 | AI-05.2.3 | `TestMessage_NoContent_FailsWithTheEmptySentinel` | No arguments, an empty slice and a nil slice all fail identically; the both-rules-violated case reports the first in the documented order, on every run |
| 8 | AI-05.3.1 | `TestMessage_CallerMutatesWhatItPassed_MessageIsUnchanged` | Replaces an element in place, appends, truncates, and reuses the backing array for a second message |
| 9 | AI-05.3.2 | `TestMessage_CallerMutatesWhatItRead_MessageIsUnchanged` | Mutates a returned slice; two readers are independent; a message copied by assignment is independent; `Roles()` is covered here for the same property |

---

## 10. What this design deliberately does not do

- **No `MarshalText`, `MarshalJSON` or any encoding on `Role` or `Message`.** Layer 1 returns values; wire shapes are confined to adapters (`V-PRV-15`), and a marshaller here would be a second rendering path with a second set of rules.
- **No `ParseMessageID`.** § 5.2.
- **No validation of content elements.** § 4.2, and `R-AMSG-009`'s second scenario pins it.
- **No message equality, no `Message.Equal`.** AI-10.6 item 3 owns it.
- **No role-versus-content-kind rule.** AI-05.1's own out-of-scope clause; AI-10.3 owns it, and it needs kinds.
- **No exported sentinel, error type or error variable.** `NFR-AMSG-B`. Everything reports through AI-04's five landed classes.
