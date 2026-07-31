# Design — content parts: readable and sealed

> **Change**: `cachicamas-ai-content-parts`
> **Milestone**: AI-06 · **Nodes**: AI-06.1 `[decision]`, AI-06.2 `[leaf]`, AI-06.3 `[leaf]`, AI-06.4 `[guard]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `decision.md`, `proposal.md`, `specs/ai-content-parts/spec.md`
> **Target**: `backend/agent/src/ai/`, `backend/agent/src/agenttest/`
> **Constraint**: Go 1.26.3, **standard library only**, zero requires

---

## 1. What this document owns

`decision.md` chose the strategy and demonstrated both properties against it. This document is the Go: every type, function, constant and file name, the validation order, the migration of AI-05's tests, and how a red step is taken against a statically typed language. Where the two disagree, `decision.md` wins and this file is the bug.

---

## 2. The surface

```go
// content_part.go

type Part struct{ payload partPayload }

func (p Part) Kind() PartKind
func (p Part) String() string

type PartKind uint8

const (
	PartKindText PartKind = iota + 1

	partKindFirst = PartKindText
	partKindEnd   = PartKindText + 1
)

func PartKinds() []PartKind
func (k PartKind) String() string

// unexported
type partPayload interface {
	kind() PartKind
	validate(at Path) *Violation
}

var partKindNames = []string{PartKindText: "text"}

func partKindName(k PartKind) string
func (p Part) validate(at Path) *Violation
func validateContent(prefix Path, content []Part) *Violation
func under(prefix Path, steps ...Step) []Step
```

```go
// text_content.go

const MaxTextLen = 1 << 20

func NewText(text string) (Part, error)
func (p Part) Text() (string, bool)

// unexported
type textPayload struct{ text string }

func (textPayload) kind() PartKind
func (t textPayload) validate(at Path) *Violation
```

```go
// message.go — changed

func NewMessage(role Role, content ...Part) (Message, error)
func (m Message) Content() []Part
// type Content — removed
```

### 2.1 Why each name

| Name | Why |
| --- | --- |
| `Part` | `V-REQ-04` is "content part"; the package is `ai`, so `ai.Part` reads as the register's term without stuttering. `ContentPart` would repeat what `Message.Content()` already says |
| `PartKind`, `PartKindText` | `V-REQ-05` is "content-part kind". The constants are prefixed with their type, as `RoleUser` is, so a bare `Text` cannot collide with the accessor |
| `NewText` | The kind's constructor, named for the kind and not for the type it returns. AI-07's `NewReasoning` and AI-09's `NewToolCall` are the same shape, which is what makes the procedure in `decision.md` § 8 a procedure |
| `Part.Text() (string, bool)` | The accessor is named for the kind it reads, matching the constructor. `decision.md` § 6.2 works the alternatives |
| `MaxTextLen` | Exported so a caller can check before constructing. `Len` and not `Size` because it bounds `len(text)` exactly — bytes, not runes |
| `partPayload` | Unexported, with unexported methods: the type that makes `Part` a sum, invisible from outside. Naming it `payload` alone would collide with the field |
| `validateContent` | Named for what it validates, not for who calls it. Its `prefix` parameter is the pinned reuse point for AI-10 |
| `under` | A three-line helper that concatenates a prefix with steps. Named as a preposition because it reads at the call site: `Invalid(ErrEmpty, under(at, At("text"))...)` |

### 2.2 What is deliberately not on the surface

| Absent | Why |
| --- | --- |
| `ParsePartKind` | No register clause asks a kind to be parsed. `ParseRole` exists because `V-REQ-01` puts provider-role mapping in an adapter; nothing says the same of a kind |
| An exported `TextPart` type | Its zero value would be **C1** |
| `Part.IsZero()` | `Kind()` already answers it against an exported vocabulary, and a second predicate would be a second way to ask one question |
| `Part.MustText()` | A panic reachable by asking the wrong question of a valid value |
| Any exported field, on any type | `decision.md` § 3.3 bypasses 2 and 3 |
| A `Content` alias for compatibility | It would keep the embedding bypass alive, which is the defect this milestone closes |

---

## 3. The seam closure

AI-05 landed `type Content interface{ isContent() }` and documented it as **not a seal on purpose**, with the embedding bypass written into its GoDoc and its `design.md` § 4.3. This change removes it.

| Before (AI-05) | After (AI-06) |
| --- | --- |
| `type Content interface{ isContent() }` | *removed* |
| `NewMessage(role Role, content ...Content)` | `NewMessage(role Role, content ...Part)` |
| `Message.Content() []Content` | `Message.Content() []Part` |
| elements not inspected; a nil element accepted | every element validated in index order |

**Why removal rather than an alias.** An alias `type Content = Part` would compile and would keep AI-05's call sites working, but nothing would be gained: the bypass AI-05 documented is the *interface*, and an alias to a struct does not preserve it. An alias to the old interface, kept alongside, would preserve exactly the defect. So the choice is between removing a name and keeping a name that means something else — and doc 0002's rule "a milestone may refine later names, but it must not violate ADR 0004" plus the v1 freeze living at AI-38 make removal the cheap option today and the expensive one later.

**What AI-05's tests become.** `role_test.go` declares the content helper; it changes from a struct embedding the interface to a call through the new constructor:

```go
// before
type part struct {
	ai.Content
	label string
}
func newPart(label string) part { return part{label: label} }

// after
func newPart(label string) ai.Part {
	p, err := ai.NewText(label)
	if err != nil {
		panic("newPart(" + label + "): " + err.Error())
	}
	return p
}
```

`message_test.go` changes `[]ai.Content` to `[]ai.Part` in seven places and replaces its two `nil` content elements — which AI-05 accepted deliberately — with `ai.Part{}`, whose expectation flips from *accepted* to *rejected*. That flip **is** `R-AMR-009`'s supersession, made visible in the test that pinned it.

---

## 4. Validation

### 4.1 The order, and where each rule lives

```
NewMessage
 ├─ 1. role in vocabulary          ErrNotInVocabulary  at role          (AI-05)
 ├─ 2. content not empty           ErrEmpty            at content       (AI-05)
 └─ 3. validateContent(nil, content)                                    (AI-06)
       └─ for i, p := range content        first failure wins
             p.validate(under(prefix, AtIndex("content", i)))
              ├─ a. payload != nil          ErrNotInVocabulary  at content[i]
              ├─ b. kind registered         ErrNotInVocabulary  at content[i]
              └─ c. payload.validate(at)
                    textPayload:
                     ├─ i.  non-whitespace present  ErrEmpty       at content[i].text
                     └─ ii. len ≤ MaxTextLen        ErrOutOfRange  at content[i].text
```

Rules 1–3 compose through `FirstFailure`, so the order is data and a reviewer reads which rule wins instead of tracing it. Within rule 3 the loop is index-ordered, so the *first failing element* is reported — `V-FAIL-04` applied to a sequence.

### 4.2 One rule set, two entry points

`NewText` and `textPayload.validate` are not two implementations. `NewText` is:

```go
func NewText(text string) (Part, error) {
	payload := textPayload{text: text}
	if err := FirstFailure(func() *Violation { return payload.validate(nil) }); err != nil {
		return Part{}, err
	}
	return Part{payload: payload}, nil
}
```

The constructor builds the payload, runs the payload's own rules, and wraps it. The message boundary runs the same rules on whatever it was handed. From outside the package the second path is unreachable — that is the seal — and from inside it is the exact mistake a future variant author makes. `decision.md` § 7.2 records why this satisfies AI-06.4's two separate legs without two rule sets.

Note the `nil` prefix in the constructor: a construction failure is positioned at `text`, because the caller is looking at a `NewText` call and not at a message. The message boundary supplies `content[i]`, so the same rule renders as `content[0].text` there.

### 4.3 The emptiness rule inspects, it does not rewrite

```go
if strings.TrimSpace(t.text) == "" { … }
```

`TrimSpace` is used as a **test**, and its result is discarded. The stored text is the caller's string, byte for byte. Writing `textPayload{text: strings.TrimSpace(text)}` would pass every rule test and fail `R-ACP-008` — which is why `R-ACP-008`'s first scenario asserts that surrounding whitespace *survives*.

`TrimSpace` cuts on `unicode.IsSpace`, so U+00A0, U+2028 and the ideographic space are all "whitespace" for the rule. That is the intended reading of `V-REQ-08`'s "model-visible": a part carrying only separators carries nothing a model reads.

### 4.4 `MaxTextLen` = 1 MiB, in bytes

It is a **sanity bound, not a provider limit**. Layer 1 holds no model catalog (`V-OUT-14`), so it cannot know any provider's context window, and a bound that pretended to would be wrong for every provider on the day after it was written. What the bound is for: making an unbounded value a caller-contract failure decidable from the request alone, which is `ErrOutOfRange`'s registered case (`V-REQ-24` uses the same reasoning for the breakpoint cap).

Bytes, not runes: the cost on the wire is bytes, and counting runes would require decoding the string — a re-encoding, on the path whose whole promise is that nothing re-encodes.

1 MiB because it is far above any plausible single text part and far below any memory concern, and because a power of two makes "one byte over the bound" a case a test can write without arithmetic.

### 4.5 `Part.String()` — the redaction posture, extended

`validation.go`'s file comment says *"a validation failure is the first thing in this package that formats caller data"*. A content part is the second, and the default is worse: `fmt.Sprintf("%v", part)` on a struct with unexported fields prints them. So:

```go
func (p Part) String() string {
	if p.payload == nil {
		return "part(unset)"
	}
	return "part(" + p.Kind().String() + ")"
}
```

It names the kind and nothing else — not the payload, not its length. A test that wants the text calls `Text()`, which is what an accessor is for.

---

## 5. The registration table and its guard

### 5.1 Production carries names; the guard carries witnesses

Production's registry is one slice, indexed by the constant, exactly like AI-05's `roleNames` and for AI-04's stated reason that a Layer 1 registry is never a map:

```go
var partKindNames = []string{PartKindText: "text"}
```

It is load-bearing in production, not decoration: `partKindName` is the bounds-checked lookup behind `PartKind.String()` **and** behind validation rule 3(b), so a payload type whose `kind()` returns an unregistered constant makes every part carrying it invalid.

The guard's witnesses — how to build a valid part of the kind, how to build one its rules reject, how to read it, and how to assemble one that skipped its rules — live in `content_part_registry_test.go`, keyed by the declared constant. Putting four closures per kind into production so a test could call them would be production carrying test scaffolding. The bite is identical: the guard enumerates the **declared constant space**, so a kind declared without a witness entry fails the first assertion.

### 5.2 The declared-constant-space rule, inherited

`PartKinds()` walks `partKindFirst … partKindEnd`, never `partKindNames`. AI-05's `role.go` states the reason and it applies verbatim: an enumeration derived from the table lists exactly the members that have an entry, so a member declared without one would be invisible to every assertion over the enumeration, and the exhaustiveness guard would be decorative.

### 5.3 The documentation claim, and how a scan checks it

`R-ACP-011` requires the documented kind list to match the table. The list lives in the GoDoc of `PartKind`, under a heading, in a fixed list shape:

```go
// # Registered kinds
//
//   - text — model-visible natural-language text (V-REQ-08).
type PartKind uint8
```

The scan parses `content_part.go` with `go/parser` (stdlib), finds the `PartKind` type spec, reads its doc comment, takes every list item under the `Registered kinds` heading, and compares the first word of each against `partKindNames`. Two failure directions, both asserted: a documented kind the table lacks, and a tabulated kind the documentation lacks.

`go/ast`, `go/parser` and `go/token` are standard library, so the guard stays inside the zero-requires rule.

### 5.4 The compile-time half of the seal

`R-ACP-007` is about what does **not** compile, which no in-process assertion can prove. Two scratch programs under `src/ai/testdata/` are built by the test with `go build`:

| Program | Contains | Expected |
| --- | --- | --- |
| `testdata/handrolled/main.go` | a type embedding `ai.Part` offered as content; a named composite literal `ai.Part{payload: nil}`; a positional literal `ai.Part{nil}` | **build fails**, and the output names each attempt |
| `testdata/constructed/main.go` | the same program written through `ai.NewText` | **build succeeds** |

The second exists so the first cannot pass for an unrelated reason. `testdata` is excluded from `./...` by the go tool and by golangci-lint, so neither program is part of `make build`, `make test`'s package set or `make lint`. Shelling out to the toolchain from a test is precedented in this module: AI-00's import guard runs `go list -deps -test` for the same reason — a property of the build graph is not observable from inside the binary.

---

## 6. File layout

| File | Node | Package | Content |
| --- | --- | --- | --- |
| `src/ai/content_part.go` | AI-06.1 | `ai` | `Part`, `PartKind`, the vocabulary, the table, `PartKinds`, both `String`s, `partPayload`, `Part.validate`, `validateContent`, `under` |
| `src/ai/text_content.go` | AI-06.2 | `ai` | `MaxTextLen`, `textPayload`, `NewText`, `Part.Text` |
| `src/ai/message.go` | AI-06.3 | `ai` | *modified*: seam closure, rule 3 |
| `src/ai/content_part_test.go` | AI-06.2, AI-06.3 | `ai_test` | Items 2–4 of AI-06.2; items 1–2 of AI-06.3 |
| `src/ai/content_part_internal_test.go` | AI-06.3 | `ai` | Item 3 — the pinned reuse point |
| `src/ai/content_part_registry_test.go` | AI-06.4 | `ai` | The guard, its witnesses, and the AST scan |
| `src/ai/testdata/handrolled/main.go` | AI-06.3 | `main` | Must not compile |
| `src/ai/testdata/constructed/main.go` | AI-06.3 | `main` | Must compile |
| `src/agenttest/content_part_test.go` | AI-06.2 | `agenttest_test` | Item 1 — the external round trip |
| `src/ai/role_test.go`, `src/ai/message_test.go` | — | `ai_test` | *migrated*, § 3 |

**One file per kind** is the layout AI-07 and AI-09 inherit: `text_content.go` holds the payload type, its rules, its constructor and its accessor, so adding a kind is one new file plus three lines in `content_part.go`. Methods on `Part` therefore live in more than one file, which is unusual and deliberate — the alternative is one file that grows a section per milestone and conflicts on every one.

**Two internal test files** (`package ai`) alongside the external ones. The guard needs `partKindNames` and needs to assemble a part that skipped its rules; the reuse pin needs `validateContent`. Neither is expressible from `ai_test`, and exporting either for a test's benefit would widen the surface the milestone exists to narrow. Everything a consumer can do is still tested from `ai_test` and from `agenttest_test`.

---

## 7. How a red step is taken against a compiler

AI-05's `design.md` § 8 established the rule and this milestone follows it: **a compile error is the state before red, not red.** Each item lands the narrowest stub that compiles and fails.

| Item | The stub that makes it fail rather than not build |
| --- | --- |
| AI-06.2.1 — external round trip | `Part`, `PartKind`, `NewText` storing the text, `Text()` returning `"", false`. The test reads a payload that is not yielded |
| AI-06.2.2 — kind agrees with payload | `Kind()` returning `0` |
| AI-06.2.3 — construction rules | `NewText` returning no error |
| AI-06.3.1 — zero value rejected | `NewMessage` not validating elements |
| AI-06.3.2 — compile-time seal | The scratch programs exist; the test asserts the build fails before `Message` holds `[]Part` |
| AI-06.4 — the guard | The witness table exists with one leg's assertion unwritten |

The seam closure (§ 3) is scaffolding, not a red step: it changes types so a test can be *written*, and it lands with AI-05's tests green. It is its own commit for exactly that reason.

---

## 8. Test plan

| Node | Test | Where |
| --- | --- | --- |
| AI-06.2.1 | `TestContentPart_TextConstructedAndReadFromAnExternalPackage_RoundTripsByteEqual` | `agenttest_test` |
| AI-06.2.2 | `TestPart_KindAndPayload_Agree` | `ai_test` |
| AI-06.2.3 | `TestNewText_RuleViolations_FailWithTheDocumentedSentinels` | `ai_test` |
| AI-06.2.4 | `TestNewText_ValidText_SurvivesConstructionUnaltered` | `ai_test` |
| AI-06.3.1 | `TestMessage_UnconstructedContentElement_IsRejected` | `ai_test` |
| AI-06.3.2 | `TestPart_HandRolledFromAnotherPackage_DoesNotCompile` · `TestPart_ExportedSurface_ExposesNoConstructibleState` | `ai_test` |
| AI-06.3.3 | `TestValidateContent_RequestShapedPrefix_ReportsTheDeeperPosition` | `ai` |
| AI-06.4.1 | `TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath` | `ai` |
| AI-06.4.2 | `TestPartKindDocumentation_MatchesTheRegistrationTable` | `ai` |
| — | `TestPartKinds_Vocabulary_IsClosedStableAndImmutable` · `TestPart_String_CarriesNoPayload` | `ai_test` |

Every name follows `Test<Subject>_<Behavior>_<Expectation>`; every scenario banner cites its leaf ID.

---

## 9. What this design deliberately does not do

- **No `init()`.** The table is a composite literal, evaluated once, mutable by nobody.
- **No reflection in production.** Reflection appears only in one test, to assert that `Part` declares no exported field.
- **No caching of `Kind()`.** Caching would be the kind field this milestone removed, wearing a performance argument.
- **No `error` return from an accessor.** `decision.md` § 6.2.
- **No exported payload type.** AI-07 and AI-09 will export payload *values* under `decision.md` § 6.3's rule; AI-06 needs none, because a text payload is a `string`.
- **No change to `validation.go`, `role.go`, or any tool or finish-reason file.** AI-08 and AI-13 are in flight in other worktrees.
