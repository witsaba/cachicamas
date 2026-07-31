# Design — the validation error taxonomy

> **Change**: `cachicamas-ai-validation-errors`
> **Milestone**: AI-04 · **Nodes**: AI-04.2 `[leaf]`, AI-04.3 `[leaf]` (AI-04.1's deliverable is `decision.md`)
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `decision.md`, `specs/ai-validation-errors/spec.md`
> **Target**: `backend/agent/src/ai/validation.go` + `validation_test.go`
> **Language**: Go 1.26.3, standard library only

---

## 1. What this document owns

`decision.md` decided *what* the taxonomy is. This document decides *how it is spelled in Go*. doc 0002 deliberately names no identifier for AI-04, and the register is forbidden from carrying one (`R-AIV-009`), so every type, field, method and variable name in Layer 1's first non-guard file is chosen here — once, with its reasoning, so nine downstream milestones inherit a settled surface instead of a convention.

---

## 2. The surface

Complete. Nothing else is exported by this milestone.

```go
package ai

// Rule classes — the sentinel set (V-FAIL-02, V-FAIL-17).
var (
	ErrEmpty               error // a required value is empty
	ErrNotInVocabulary     error // a value is outside a closed vocabulary
	ErrOutOfRange          error // a value is outside a documented bound
	ErrMalformed           error // a value is not well-formed for its documented encoding
	ErrUnresolvedReference error // a value names something the request does not declare
	ErrDuplicate           error // a value repeats another the collection already carries
	//                             (appended 2026-07-31 by AI-08 — see § 3.1)
)

// Positional context (V-FAIL-03).
type Step struct{ /* unexported */ }

func At(name string) Step                  // a named field, no index
func AtIndex(name string, index int) Step  // a named field, one element of it

func (s Step) Name() string
func (s Step) Index() (int, bool)
func (s Step) String() string

type Path []Step

func (p Path) String() string

// The one caller-contract failure type (V-FAIL-01).
type Violation struct{ /* unexported */ }

func Invalid(rule error, at ...Step) *Violation

func (v *Violation) Error() string
func (v *Violation) Unwrap() error
func (v *Violation) Path() Path

// Ordered first-failure (V-FAIL-04).
type Rule func() *Violation

func FirstFailure(rules ...Rule) error
```

Five sentinels at AI-04 — six from 2026-07-31, see § 3.1 — four types, nine functions and methods. A Layer 2 consumer writes at most two calls against it: `errors.Is` for the class and `errors.As` for the position. The sentinel count is the only part of this surface the append rule may move; the types, functions and methods are what AI-40 freezes.

---

## 3. Why each name

| Name | Chosen because | Rejected alternative, and why |
| --- | --- | --- |
| `Violation` | It names the *thing that happened* — a rule was violated — and reads as a noun at the call site: `var v *ai.Violation; errors.As(err, &v)`. The sentinel already carries which class and the path carries where, so the type is literally the record of one violation | `ValidationError`. Idiomatic in the standard library's sense, but it invites the pairing `ValidationError` / `ProviderError` at AI-19 and with it the false symmetry that both are errors of the same kind. They are not: one is the caller's bug and is knowable without I/O, the other is neither. The register keeps the two vocabularies apart; the names should too |
| `Invalid` | The call site reads as a sentence: `return ai.Invalid(ai.ErrEmpty, ai.At("model"))` — "invalid: empty, at model" | `NewViolation`. Correct and noisier. Nine milestones write this call hundreds of times |
| `ErrEmpty`, `ErrNotInVocabulary`, `ErrOutOfRange`, `ErrMalformed`, `ErrUnresolvedReference`, `ErrDuplicate` | Each names a **rule class**, never a rule instance or a type. `revive`'s `error-naming` rule requires the `Err` prefix; the rest of each name is the class | `ErrEmptyModelIdentity` and its kin — per-instance sentinels, rejected in `decision.md` § 3.2. For `ErrDuplicate` specifically: `ErrDuplicateToolName`, which is the same per-instance mistake wearing the new class's clothes — the class is uniqueness, and a tool set is only the first collection to need it |
| `At`, `AtIndex` | They read as location at the call site and they are unlikely to collide with a domain noun a later milestone wants | `Field` and `Element`. Both are words AI-10 may well want for the request's own vocabulary, and a collision in package `ai` is a rename across nine milestones |
| `Step`, `Path` | A path is a sequence of steps; both are the ordinary words and neither is a domain noun of Layer 1 | `Location`, `Position`. Longer, and "position" is already the register's conceptual term — reusing it as a Go name would blur the artifact/code line the register draws |
| `Rule`, `FirstFailure` | `Rule` is `V-FAIL-16`'s noun. `FirstFailure` names the policy in the call, so a reader of AI-05's validator sees which policy is in force without opening this file | `Validate`, `Check`, `All`. `Validate` says nothing about ordering, and ordering is the entire point |
| `Path()` on the failure | The accessor is the extraction `V-FAIL-03` requires | Exporting the field. An exported slice field is mutable from outside and would make the failure's position rewritable by a consumer |

### 3.1 Why the sentinels are five, and not four or eight

> **Amended 2026-07-31** — six, not five. `ErrDuplicate` was appended by **AI-08** under `decision.md` § 3.5's append rule, and the row for it is in the table below with the citable case the rule requires: AI-08.2 item 1, two tool declarations carrying the same name in one tool set. The heading is left as it was written rather than renumbered, because the question it asks — *why this many* — is answered by the citable-case rule and not by the number, and a heading rewritten each time the set grows would make the append look like a revision of the reasoning instead of an application of it. § 2's surface listing and § 3's name table carry the same amendment. `ruleClasses` in § 4.2 is the enumeration in code; it grew by one entry, at the end.
>
> The reason it is `ErrDuplicate` and not `ErrMalformed` stretched one notch is stated where a reader meets it, in `NewToolSet`'s own GoDoc: each name in a set with a repeated entry is perfectly well-formed, and what fails is uniqueness across the set — a property no single value can be inspected for. The two classes name the two different fixes a consumer must make, which is the only thing `errors.Is` is for.

Each has a citable case, per `R-AIE-003` and `S-AIE-007`:

| Sentinel | Citable case |
| --- | --- |
| `ErrEmpty` | Register § 6.3: "Had the model identity been *empty*, that is decidable without I/O and is a caller-contract failure." Also doc 0002 AI-05.2 item 3 (a message with no content) |
| `ErrNotInVocabulary` | `V-REQ-01`: "a value outside the vocabulary is a caller-contract failure, not an unknown passed through." Also doc 0002 AI-05.1 item 2 |
| `ErrOutOfRange` | `V-REQ-24` breakpoint cap: "a request that exceeds it is a caller-contract failure rather than a silent truncation" |
| `ErrMalformed` | Register § 6.3: "Argument bytes that are not well-formed for the documented encoding are decidable from the request alone → caller-contract" |
| `ErrUnresolvedReference` | Register § 6.3: "a tool choice naming a tool absent from the declared set is decidable from the request alone → caller-contract" |
| `ErrDuplicate` *(appended 2026-07-31 by AI-08)* | doc 0002 AI-08.2 item 1: two tool declarations with the same name in one tool set. Both halves of the fact are in the request, so it is decidable from the request alone |

~~A sixth class for conflicting values is plausible and has no citable case; it is not landed.~~ The append rule (`decision.md` § 3.5) is what makes that cheap to correct — and it did: the sixth class was landed by AI-08 when the case arrived, at the cost of one line at one call site. The struck sentence is left visible rather than deleted, because a prediction that came true is worth more as evidence than as a tidy paragraph.

---

## 4. The three mechanisms that carry the decisions

### 4.1 Class via `Unwrap`, position via the type

```go
type Violation struct {
	rule error
	at   Path
}

func (v *Violation) Unwrap() error { return v.rule }
```

`errors.Is(err, ai.ErrEmpty)` traverses any number of wrappers, reaches the `*Violation`, unwraps once more and matches the sentinel. `errors.As(err, &v)` reaches the same value and yields the position. Two axes, one type, no ordering dependency between wrappers — `decision.md` § 5.2.

`Unwrap` returning the rule rather than a wrapped chain is deliberate: it makes the sentinel the failure's *cause*, which is what `errors.Is` was designed to traverse, and it keeps the failure a leaf of the chain in every other respect.

### 4.2 Redaction as construction, in two places

`decision.md` § 5.3 requires redaction to be a property of the type. Two mechanisms, both mechanical:

**Structural names are filtered at construction and stored filtered.** The filter admits ASCII letters and underscore, up to 32 bytes, and rejects everything else — a name that fails is replaced *whole* by `"?"`, never trimmed, because a prefix of a secret is still a secret. Digits are excluded as well as punctuation: a structural field name never needs one, and their absence removes the shape most credentials and opaque identifiers take. Filtering at construction rather than at rendering means the value itself never holds caller data, so a future milestone that logs `Step.Name()` directly is still safe.

The filter is a **backstop, not the contract**. The contract is that a position is built from names this package wrote; the filter is what makes a value that is not one unable to render. A caller string that happens to be identifier-shaped and short would survive it, and that is the stated limit of the mechanism rather than a gap in it — the pin is what keeps the contract honest.

**The message renders the registered class's text, never the supplied error's.**

```go
func ruleText(rule error) string {
	if rule == nil {
		return "value violates an unnamed rule"
	}
	for _, class := range ruleClasses {
		if errors.Is(rule, class) {
			return class.Error()
		}
	}
	return "value violates an unregistered rule"
}
```

`ruleClasses` is an ordered slice — not a map — of the sentinels (five at AI-04; `ErrDuplicate` appended at the end on 2026-07-31, see § 3.1). `errors.Is` is used so a *wrapped* sentinel is still recognised as its class, and the text printed is the class's own, so a wrapper whose message carries caller data contributes nothing to the rendering while remaining fully matchable. Nothing in the standard library does this; it is the difference between "we agreed not to put values in messages" and "there is nowhere to put them".

> **Amended 2026-07-31** — `ruleClasses` is now **enforced**, by `backend/agent/src/ai/validation_registry_internal_test.go` (`T-AIE-12`). AI-08's `ErrDuplicate` append is what exposed the gap: the registry is unexported on purpose — a consumer never iterates the set, and exporting it to satisfy a test would widen the surface AI-40 freezes — so `validation_test.go` carried a hand-written copy, and its comment argued that being hand-written was the point. Nothing compared the two. A class appended here stayed outside the "a failure matches exactly one rule class" pin until somebody remembered to edit the copy, and a sentinel declared in the `var` block but never added to the slice was caught by nothing at all: `ruleText` renders it as `"value violates an unregistered rule"`, silently, and no test covered it. The guard is an internal test — package `ai`, so it reads the unexported slice without exporting anything — that scans both sources with `go/ast` and asserts both directions: every declared sentinel is registered, and the mirror is the registry with the same members in the same order. It binds the source it reads to the running slice by comparing each `ruleClasses[i].Error()` against the string literal its declaration was given, so it is a claim about the registry `ruleText` iterates and not about two lists agreeing in one file. Both directions are recorded biting, verbatim, in `tasks.md` § `T-AIE-12`; no drift was present when it landed.
>
> The general form matters more than the instance. This section argues that redaction is a *property of the type rather than of anyone's discipline*; the same standard applies to the registry the redaction reads from, and a hand-written mirror was discipline wearing a mechanism's clothes. AI-10 and AI-12 face the same shape — a closed set with an unexported enumeration — and the guard, not the comment, is what they should copy.

The rendered form is therefore composed only of: the class's own fixed text, filtered names, decimal integers, `.`, `[`, `]` and `: `.

```
model: required value is empty
messages[2].content[0]: value is outside a closed vocabulary
tools[3]: value names something the request does not declare
```

### 4.3 Ordered first-failure, evaluated lazily

```go
type Rule func() *Violation

func FirstFailure(rules ...Rule) error {
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if v := rule(); v != nil {
			return v
		}
	}
	return nil
}
```

Three properties, each load-bearing:

- **Lazy.** A rule is a function, so a rule after the first violation is never evaluated (`S-AIE-018`). A variadic list of already-constructed violations would evaluate everything and would allocate on the success path, which is the aggregation shape wearing a first-failure hat.
- **Ordered by a slice.** The thing deciding which rule fires is a slice index. There is nowhere for a map to hide (`S-AIE-023`), which is `decision.md` § 4.2's "impossible to violate, not merely required".
- **Returns `error`, not `*Violation`.** This is the typed-nil trap, and it would otherwise be reproduced at every call site in nine milestones: a nil `*Violation` assigned to an `error` is a non-nil interface. Returning `error` and converting exactly once, here, means `if err := ai.FirstFailure(...); err != nil` is correct everywhere. `S-AIE-019` is the test that pins it.

### 4.4 Totality

`R-AIE-009` requires no input to panic. The mechanisms:

| Input | Behavior |
| --- | --- |
| A nil `*Violation` receiver | `Error()` returns a fixed string; the method checks its receiver |
| `Invalid(nil, …)` | Renders "value violates an unnamed rule"; `Unwrap` returns nil, which ends the chain cleanly for `errors.Is` |
| A zero-value `Step{}` | Renders as `?` — the empty name takes the same path as a rejected one |
| `AtIndex(name, i)` with `i < 0` | Treated as no index. A negative index names no element, and rendering `[-1]` would be a message a reader has to decode |
| A path of thousands of steps | Rendered with a `strings.Builder` in a loop. Nothing recurses, so depth costs memory and never the stack |
| A structural name at or beyond the bound | Replaced whole at construction |
| A nil entry in a rule list | Skipped by `FirstFailure` |

### 4.5 `Path()` returns a copy

`v.Path()` clones the slice before returning it. A failure is a value that gets passed upward and logged; a consumer that appends to the returned slice must not be able to rewrite the failure's own position. The cost is one allocation on a cold path, which is the correct side of that trade.

---

## 5. File layout

| File | Contents | Approximate size |
| --- | --- | --- |
| `backend/agent/src/ai/validation.go` | The whole surface, in the order: file documentation for the taxonomy, sentinels and registry, `Step`/`Path`, `Violation`, `Rule`/`FirstFailure`, unexported helpers | ~190 lines including GoDoc *(landed at 337: 127 lines of code under 210 of contract documentation)* |
| `backend/agent/src/ai/validation_test.go` | `package ai_test`. Six test functions, one per test-list item, each with a leaf-ID banner | ~230 lines *(landed at 568)* |

The file documentation block is separated from `package ai` by a blank line on purpose: `doc.go` owns the package comment, and a second one would be a duplicate package doc. This block is a file banner, which is the same shape AI-00's two guard files use.

One production file, because the surface is one concept and splitting it would make a reader open two files to answer one question. `doc.go` is not touched: it describes the package, and a milestone's own contract documentation belongs on its declarations.

**The test package is `ai_test`, not `ai`.** It is a genuinely external package — it cannot see an unexported field — so the tests exercise the same surface Layer 2 will. `src/agenttest/` is deliberately not touched: its own comment reserves the first real cross-package readability proof for AI-06.2, on the first content part that exists, and duplicating that proof here would make the AI-06.2 claim false the day it is written.

---

## 6. How a red step is taken in a statically typed language

`openspec/AGENTS.md` requires the red test to **compile and run**, with the assertion failing for the right reason. Go will not compile a test against a symbol that does not exist, so "write the test first" needs one stated convention, applied to every item of AI-04.2 and AI-04.3:

1. Write the test.
2. Add the **narrowest declaration** that makes it compile and makes the assertion fail — a function returning the zero value, a method returning a fixed empty string. Never one that could pass.
3. Run, and record the failing output. A compile error is *not* the red step; it is the state before it.
4. Implement minimally, run, record green.
5. Refactor while green.

The rule that keeps this honest: a red step whose stub could plausibly satisfy any part of the assertion is not a red step. Where a stub would be indistinguishable from an implementation — as it would for a one-line accessor — the item is proven by an assertion that the stub cannot satisfy, not by a weaker assertion.

The pin (AI-04.3 item 3) is exempt from red-first by doc 0002's leaf anatomy, and is still fully mechanical: it feeds a distinctive sentinel body through **both** dynamic channels of the message — the structural name and the rule's own text — and asserts its absence.

---

## 7. Test plan

Six functions, in the order they are written. Each carries a banner comment citing its leaf ID.

| # | Leaf item | Test function | What makes it fail if the behavior regresses |
| --- | --- | --- | --- |
| 1 | AI-04.2.1 | `TestViolation_WrappedFailure_StillMatchesItsSentinel` | Wraps once and twice; asserts the sentinel matches and that a *different* sentinel does not, so a match-everything implementation fails |
| 2 | AI-04.2.2 | `TestViolation_PositionalContext_IsExtractedByErrorsAs` | Extracts through a wrapper; reads name and index of every step programmatically; covers a message/content path, a tool index, and a field with no index |
| 3 | AI-04.2.3 | `TestViolation_RenderedMessage_CarriesNoCallerContent` | Feeds a distinctive body as a structural name and inside a wrapping rule error; asserts absence of the body and of every prefix of it |
| 4 | AI-04.3.1 | `TestFirstFailure_SeveralViolatedRules_ReportsTheFirstInOrder` | Asserts the reported rule, that later rules never ran, that an all-passing list yields a truly nil error, and that repeated and concurrent evaluation agree |
| 5 | AI-04.3.2 | `TestViolation_ExtremeInputs_NeverPanics` | Table of nil, zero-value, empty, deeply nested and over-long inputs; the test fails on any panic |
| 6 | AI-04.3.3 *(pin)* | `TestViolation_SentinelBody_IsNeverRetainedInTheMessage` | The regression pin. A later refactor that embeds payloads fails here |

Determinism (item 4) is proven by repetition and by concurrency rather than by inspection, because "no map iteration decides which rule fires" is a claim about behavior under a scheduler. The concurrent half runs under `-race`, which the evidence gate already requires.

---

## 8. What this design deliberately does not do

- **No `Is` or `As` methods on `Violation`.** `Unwrap` is sufficient and custom traversal is where surprises live.
- **No error code, no string key, no `Kind` field.** Two representations of one fact drift; `errors.Is` is the single query.
- **No aggregation type.** `decision.md` § 4.3 records that it remains addable without changing anything landed here.
- **No formatting verbs.** `Violation` implements `error` and nothing else; a `Format` method would be a second rendering path and therefore a second place for caller data to appear.
- **No constructor validation that rejects an unregistered rule.** Rejecting would need an error return from an error constructor. Instead the *rendering* is safe for any rule, and an unregistered class renders generically — visible in test output, harmless in a log.
