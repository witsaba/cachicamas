# Design — finish reasons and usage

> **Change**: `cachicamas-ai-completion-metadata`
> **Milestone**: AI-13 · **Nodes**: AI-13.1, AI-13.2, AI-13.3, AI-13.4 — all `[leaf]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-completion-metadata/spec.md`
> **Target**: `backend/agent/src/ai/finish_reason.go` + `finish_reason_test.go`, `backend/agent/src/ai/usage.go` + `usage_test.go`
> **Language**: Go 1.26.3, standard library only

---

## 1. What this document owns

`spec.md` says what must be true. This document decides how it is spelled in Go, and settles the two semantic questions the register hands down: whether the input count includes cached tokens, and whether the reasoning count is inside the output count. doc 0002 names no identifier for AI-13 and the register is forbidden from carrying one (`R-AIV-009`), so every name below is chosen here, once, with its reasoning.

It also owns two rejections that would otherwise be re-litigated by every reviewer: no `Resumable()`-style method (§ 4.3), and no exported derived totals (§ 5.4).

---

## 2. The surface

Complete. Nothing else is exported by this milestone.

```go
package ai

// ---- finish_reason.go — the closed vocabulary (V-MET-01 … V-MET-08) ----

type FinishReason uint8

const (
	_ FinishReason = iota // the zero value names no finish reason
	FinishReasonStop
	FinishReasonLength
	FinishReasonToolCalls
	FinishReasonContentFilter
	FinishReasonRefusal
	FinishReasonPauseTurn
	FinishReasonUnknown
)

func NormalizeFinishReason(providerStopValue string) FinishReason

func (r FinishReason) String() string
func (r FinishReason) Validate(at ...Step) error

// ---- usage.go — the consumption record (V-MET-09 … V-MET-12) ----

type TokenCount struct{ /* unexported */ }

func Tokens(n int64) TokenCount

func (t TokenCount) Count() (int64, bool)
func (t TokenCount) String() string

type Usage struct {
	Input      TokenCount
	Output     TokenCount
	CacheRead  TokenCount
	CacheWrite TokenCount
	Reasoning  TokenCount
}

func (u Usage) Validate(at ...Step) error
```

Two types, seven constants, one constructor, five methods. A Layer 2 consumer switches on a `FinishReason` and reads five fields of a `Usage`.

---

## 3. Why each name

| Name | Chosen because | Rejected alternative, and why |
| --- | --- | --- |
| `FinishReason` | It is `V-MET-01`'s own term, and the register is the naming authority for Layer 1 nouns | `StopReason`. That is one vendor's wire name, and adopting a vendor's spelling for the neutral vocabulary is the leakage § 3.3 exists to prevent |
| `FinishReasonStop` … `FinishReasonUnknown` | Prefixed constants, because Go constants share the package namespace: a bare `Unknown` in package `ai` is a name three later milestones will each want | A nested type or a `Reason` sub-package. Both cost an import or a qualifier at every call site for a value that appears in every switch Layer 2 writes |
| `NormalizeFinishReason` | It names what it does — `explore.md`'s "normalization", the total mapping from a provider stop value into the vocabulary — and its lack of an error return is then unsurprising | `ParseFinishReason`. `Parse*` sets the expectation of an `(T, error)` pair, and an error return here would be a lie: `V-MET-08` makes an unrecognised string a *recorded outcome*, not a fault |
| `Validate(at ...Step) error` | The variadic position is how a parent contract says *where* this value sits, so AI-15.2 can report "completion.finish_reason" without this file knowing the event exists. Returning `error` rather than `*Violation` is AI-04's own instruction: `FirstFailure` "converts once, here", so `if err := …; err != nil` is correct at every call site | `IsValid() bool`. It throws away the position and the rule class, which is the whole of AI-04 |
| `TokenCount` | `V-MET-10`'s term is "token-count field"; the value is the count | `Tokens` as the type name, freeing `Tokens` for nothing. The constructor is the name callers write most |
| `Tokens(n)` | The call site reads as data: `ai.Usage{Input: ai.Tokens(12), Output: ai.Tokens(3)}` | `NewTokenCount(12)`, `Present(12)`. Both are noise in a struct literal that will be written by every adapter for every response |
| `Count() (int64, bool)` | The two-result shape is already this package's idiom for "a value that may not be there" — `Step.Index() (int, bool)`, landed by AI-04. A second idiom for the same idea is the parallel vocabulary `V-FAIL-03` forbids | `Value() int64` plus `Present() bool`. Two calls, and the first one is usable without the second, which is exactly the mistake `V-MET-11` describes |
| `Input`, `Output`, `CacheRead`, `CacheWrite`, `Reasoning` | `V-MET-09`'s own list, in its own order | `PromptTokens` / `CompletionTokens`. Vendor spellings again, and they disagree with each other |
| Exported fields on `Usage` | The record must be "constructible with any subset present" and "readable from an external package, field by field". A struct literal is the shortest expression of both, and the zero value of the field type is already the correct default — absent | A constructor with five arguments, or five `With*` builders. The first forces every adapter to pass placeholders for what it does not know; the second is five methods to avoid one literal |

### 3.1 Why the zero value of `FinishReason` names nothing

`explore.md` T1. The constant block starts at `iota` with a blank identifier, so `FinishReason(0)` is not a member of the vocabulary and `Validate` rejects it.

This is the one place this milestone spends a name to buy a property. The property is that **"nobody set this" and "the provider said something I do not recognise" stay two facts**, which is `V-MET-11`'s distinction applied to the finish reason instead of to a token count. If unknown were the zero value, a completion event constructed without a finish reason would report a deliberate, recorded "unrecognised provider string" — a fabricated observation, and precisely the class of lie `CAP-R-03`'s precision clauses exist to forbid one level down.

It also matches the repository's own posture: retired defect **C1** was a type whose zero value passed validation, and doc 0002 answers it at AI-06.1 with "zero-value-invalid".

### 3.2 A note on AI-05, running concurrently

AI-05 lands roles, and a role is also a closed vocabulary that will also want an exhaustiveness pin. The two were built without sight of each other, so **a reconciliation pass is expected after both land** — either extracting a shared shape or, more likely, recording that two small enums do not justify one shared abstraction. Nothing here anticipates that pass: no name is reserved for it, and no helper is exported in the hope of being reused. Whichever pattern reads better after both are visible is the one that should survive.

---

## 4. The finish-reason chain

### 4.1 Closure, and the pin that keeps it closed

The vocabulary's extent is expressed once, as an unexported bound declared immediately after the last constant:

```go
const finishReasonLimit = FinishReasonUnknown + 1 // the first value outside the vocabulary
```

`Validate` accepts `0 < r < finishReasonLimit`. The string forms live in an array sized by that bound, indexed by the value:

```go
var finishReasonNames = [finishReasonLimit]string{
	FinishReasonStop: "stop",
	// …
}
```

Adding a constant therefore does three things at once, and two of them are failures. The bound grows, so the new value validates. The array grows with an empty entry, so `String` returns the placeholder. And the round-trip `NormalizeFinishReason(r.String()) == r` fails for it. That is the mechanism `R-ACM-006` asks for, built from the language rather than from a linter the module may not have (`explore.md` T5).

The test-side half walks every value the underlying `uint8` can hold — 256 iterations, no bound to guess — asks the package which of them validate, and compares that set against the seven the test names by hand. So a new constant fails the count assertion as well. Both halves are recorded biting against a scratch value in `tasks.md`.

### 4.2 Normalization is a map lookup after trimming and lowering

`strings.TrimSpace` then `strings.ToLower`, then one lookup in `finishReasonBySpelling`. Nothing else: no hyphen folding, no prefix matching, no fuzzy fallback. `R-ACM-004` says "trims and lowers", and every additional cleverness is a rule some vendor's next release will trip over in a way nobody tests.

A map is used here, and AI-04's warning — "nothing in this package may let an unordered iteration decide anything" — is respected: the map is only ever *looked up*, never iterated. The set of synonyms per family comes from `explore.md` § 4 and is deliberately short. Per-vendor mapping is AI-31.1's; this table exists so that the common spellings do not each need an adapter special case, not to be exhaustive over vendors that do not exist yet.

The miss branch returns `FinishReasonUnknown`. There is no error return and no panic path, for any string, including the empty one — `R-ACM-004`, and doc 0002's "the normalizer-crash bug class, pinned from the first day".

### 4.3 The three-way distinction, and the method that is deliberately absent

`R-ACM-007` asks that collapsing refusal, pause-turn and unknown require a compile-visible change. In Go that is a property of naming them as separate constants: a consumer's exhaustive switch names each one, so deleting a constant breaks every consumer's build. The test asserts this by naming all seven constants individually and by running a consumer-shaped exhaustive switch whose `default` branch fails the test.

**Rejected: a `Resumable() bool` or `NextAction()` method.** The case for it is real and worth stating at full strength: doc 0001 says the three values have "three different correct responses", so a type that knows the values could carry the responses; every consumer would otherwise re-derive the same three-branch mapping; and a single source for it is exactly what a neutral contract is for.

It loses on `V-OUT-10`, which is unambiguous: "loop termination — Layer 2 owns it; `V-MET-01` finish reason is the **input** to that decision, not the decision. A different termination rule is a different harness, not a different Layer 1." A method answering "should the loop continue?" makes every harness that disagrees a fork of Layer 1 rather than a policy above it, and it does so while looking like a convenience. So the obligation is **recorded, not implemented**: it is stated in the GoDoc of each of the three values, and `R-ACM-008` makes the recording checkable. AI-13.2's test list says "testable where it is testable" — this is the clause that says which half is which.

The pause-turn obligation, recorded verbatim on the declaration: *the correct response is to resume, replaying the received content verbatim.* doc 0002 assigns two nodes to honor it — AI-31.1 for the mapping, and Layer 2 for the loop — and notes at AI-31.1 that a paused response containing block types normalization skips makes resume lossy, which is AI-24's decision to record.

### 4.4 Refusal against content filter, on the declarations

`R-ACM-003` asks for the line to be documented where it will be read. Both constants carry it:

- **Refusal** — the model itself declined. The decision is the model's, it is final, and the response is a complete answer of the form "no". Retrying the same request produces the same answer.
- **Content filter** — a provider-side filter intervened, before or after the model. The decision is not the model's, and the correct response may well differ: a different provider, a different model, or a surfaced policy message.

Collapsing them would produce a harness that retries refusals forever and one that surfaces filter interventions as model opinions. They are `V-MET-05` and `V-MET-06`, two rows, for that reason.

---

## 5. The usage chain

### 5.1 Absence is the zero value, and it is a value type

```go
type TokenCount struct {
	count   int64
	present bool
}
```

`explore.md` T2 rejects both alternatives. A `*int64` is aliasable — two usage records can be made to share a count, and a copy of a record is not a copy of its data — and it invites `*u.Input` at call sites, where an absent count becomes a panic. A sentinel integer puts "absent" inside the domain of the number.

The value type makes the **zero value already correct**: `Usage{}` is a record whose five counts are all absent, which is exactly what `CAP-R-03` clause 2 requires of an adapter that reports nothing. There is no constructor to remember and no way to build a record of fabricated zeros by accident.

`int64` rather than `int` because a token count arrives from a wire number and leaves toward an arithmetic that Layer 3 will multiply by a price; a width that varies by platform in that chain is a defect waiting for a 32-bit build.

`Tokens(-1)` is constructible on purpose. AI-04's design records the same choice for `Invalid`: a constructor that validates needs an error return, and an error return on a value constructor is paid for at every call site. Rejection happens where rejection belongs — in `Validate`, through `ErrOutOfRange`.

### 5.2 Validation is ordered first-failure over the five fields

```go
func (u Usage) Validate(at ...Step) error {
	return FirstFailure(
		u.rule(u.Input, at, "input"),
		u.rule(u.Output, at, "output"),
		u.rule(u.CacheRead, at, "cache_read"),
		u.rule(u.CacheWrite, at, "cache_write"),
		u.rule(u.Reasoning, at, "reasoning"),
	)
}
```

The order of the argument list is the documented order (`V-FAIL-04`, inherited from AI-04.1 decision 3), and it is the same order as `V-MET-09`'s list and the struct's own field order. Three orders that agree is not an accident worth leaving to chance — `R-ACM-012`'s field-set pin holds all three together.

One trap this design must not fall into: each rule appends a step to the caller's position, and five closures appending to one slice would share a backing array and overwrite each other's last step. The position is therefore extended with `append(slices.Clone(at), At(name))`, never with a bare `append` on the variadic. The test for `S-ACM-025` positions the record inside a parent path precisely so this is exercised rather than assumed.

### 5.3 The two semantic decisions

**Decision 1 — the input count EXCLUDES cache-read and cache-write.** The three are mutually exclusive; no token is counted twice; the total input is their sum.

The case for the inclusive reading, at full strength: one of the two vendors already reports it that way, so an inclusive contract needs no adapter arithmetic for it; "input tokens" in plain English means all the tokens of the input, and a field that excludes most of a cached prompt is surprising; and a consumer that only wants a rough figure can read one field instead of three.

It loses on three grounds.

1. **The three have three different prices.** A cached read costs a fraction of fresh input and a cache write costs more than it. No consumer can ever sum them at one rate, so the "read one field" convenience is unavailable in the only computation that matters. Once every term is multiplied by its own rate, disjoint fields make the formula a plain sum and inclusive fields make it a sum with a subtraction inside it — and the subtraction is the step that gets forgotten.
2. **Absence degrades gracefully one way only.** Under exclusive semantics, an absent cache-read leaves the input field still exactly true of what it names, and the consumer knows only that it cannot compute a total. Under inclusive semantics an absent cache-read makes the input field's *meaning* unknown — it may or may not contain cached tokens — so a single missing field corrupts a figure that is present. `V-MET-11` says a consumer who confuses absence with zero "writes a wrong cost formula"; the inclusive reading makes that outcome reachable even for a consumer who is careful.
3. **Both conversions are available to the adapter, and only one is available to the consumer.** An adapter holds both numbers and can subtract or add at will. A consumer holding an inclusive input and an absent cache figure holds nothing it can correct.

The cost, conceded: the adapter for the inclusive vendor must subtract, and if it forgets, every cached token is counted twice. doc 0002 already schedules the catch — AI-31.3 verifies these semantics "against a real cache-hit transcript".

**Decision 2 — the reasoning count is INSIDE the output count.** It is a breakdown of a billed quantity, not a quantity of its own.

This is the opposite rule on the opposite side of the record, which needs its reason stated or it reads as inconsistency. Both vendors bill reasoning tokens **at the output rate**: they are output tokens that happen to be attributable to reasoning. By the rule that decides decision 1 — *a count priced differently from every other is a term; a count priced the same as another is a breakdown of it* — reasoning is a breakdown. Making it a separate term would create a field with no rate of its own, and a consumer summing the terms would double-count every thinking token.

It has a second benefit that `V-MET-12` names: "on models with adaptive reasoning, the reasoning token count arrives only on the final usage update, so any mid-stream figure is structurally an estimate". Under this rule the **output count is complete at every update** and only the attribution fills in later. Under the alternative, every mid-stream output figure would be short by an unknown amount.

**The rule, stated once for the next field that arrives:** a count priced differently from every other is a term of the formula; a count priced the same as another is a breakdown of it. `R-ACM-012`'s pin exists so the author of that field has to come back and read this paragraph.

### 5.4 Rejected: exported derived totals

The obvious way to make the formula executable is to export it:

```go
func (u Usage) TotalInput() TokenCount  // Input + CacheRead + CacheWrite
func (u Usage) TotalOutput() TokenCount // Output
```

The case for it, at full strength: the formula would then live in production code rather than only in documentation and a test; a field added later would have to be routed through it, so the pin would bite on the arithmetic instead of on a field list; and every consumer would otherwise write the same three-term sum.

It loses on absence, and the way it loses is instructive. A derived total has to decide what to do when a component is absent, and both answers are wrong:

- **Sum the present components.** This is treating absent as zero inside the one function whose entire purpose is the fields where that distinction matters. It is `V-MET-11`'s defect, wearing a convenience method as a disguise.
- **Report absent unless every component is present.** Honest, but it makes the total absent for any provider that does not report all three — and doc 0002 explicitly requires an adapter to leave a field absent when the transcript never carried it (AI-31.2 item 1: "usage fields never present in the transcript are **absent** in the normalized usage, not zero"). So the honest version returns absent for exactly the providers a consumer would ask about.

doc 0002 asks for the formula "expressed as an assertion in the test", and names the failure it wants caught as "a later field addition that changes the formula". Both are delivered without the method: the formula is asserted over constructed records in `usage_test.go`, and the field-set pin (§ 6) fails on any field addition. The surface stays at its minimum, which is also AI-04's stated posture — no speculative surface, and no exported helper landed for a consumer that does not exist.

If Layer 2 later wants a total, it can write one against fields it can see, with an absence policy that is its own layer's business.

---

## 6. The two pins

Both are `*(pin)*` items: exempt from red-first, mechanical, and recorded biting against a deliberate scratch violation before landing.

| Pin | Mechanism | Bites when |
| --- | --- | --- |
| **Vocabulary exhaustiveness** (AI-13.1 item 5) | Walk every value of the underlying `uint8`; collect those the package validates; assert that set equals the seven the test names; assert every member round-trips through `String` and `NormalizeFinishReason` | A constant is added without a string form, without a normalization entry, or without the test's hand-named list being updated — three independent failures for one omission |
| **Cost-formula term list** (AI-13.4 item 2) | `reflect` over `Usage`: assert exactly five fields, with the documented names, types and order | A field is added, removed, renamed or reordered — forcing its author to state which side of § 5.3's rule it lands on |

`reflect` is standard library, so the zero-require rule holds. It is used only in a test and only over a type this package defines.

---

## 7. File layout

| File | Contents | Forecast |
| --- | --- | --- |
| `backend/agent/src/ai/finish_reason.go` | File banner, the constant block with per-value documentation, the bound, the string array, the normalization table, `NormalizeFinishReason`, `String`, `Validate` | ~150 lines including GoDoc |
| `backend/agent/src/ai/finish_reason_test.go` | `package ai_test`. Seven test functions — five for AI-13.1, two for AI-13.2 | ~230 lines |
| `backend/agent/src/ai/usage.go` | File banner, `TokenCount` with its constructor and two methods, `Usage` with its documented field semantics and the cost formula, `Validate` | ~140 lines including GoDoc |
| `backend/agent/src/ai/usage_test.go` | `package ai_test`. Five test functions — three for AI-13.3, two for AI-13.4 | ~230 lines |

Four files rather than two, split along the graph's two chains. They share no symbol, so a reviewer reading the cost-formula decision never has to scroll past the stop vocabulary. `doc.go` is not touched — it describes the package, and a milestone's contract documentation belongs on its declarations. `validation.go` is not touched at all; this change is a **consumer** of AI-04, which is the point of AI-04 having landed first.

The test package is `ai_test`, external, so every assertion is written against exactly the surface Layer 2 sees. `src/agenttest/` is deliberately untouched: its own comment reserves the first cross-package readability proof for AI-06.2.

---

## 8. How a red step is taken in a statically typed language

Inherited verbatim from AI-04's `design.md` § 6, because the rule has to be the same in every milestone of this package:

1. Write the test.
2. Add the **narrowest declaration** that makes it compile and makes the assertion fail — a constant block with the right names but the wrong closure, a method returning a fixed string, a validator returning nil.
3. Run, and record the failing output. A compile error is *not* the red step; it is the state before it.
4. Implement minimally, run, record green.
5. Refactor while green.

A red step whose stub could plausibly satisfy any part of the assertion is not a red step.

---

## 9. What this design deliberately does not do

- **No `Resumable()`, `IsTerminal()` or `NextAction()`** — § 4.3. `V-OUT-10`.
- **No exported derived totals** — § 5.4.
- **No `MarshalJSON`, no wire tags.** Layer 1 returns values; rendering belongs to whoever renders. A tag here would also quietly decide the absent-versus-zero question for a wire format nobody has specified.
- **No `error` return on `NormalizeFinishReason`** — § 4.2. `V-MET-08`.
- **No new validation sentinel.** Two rule classes already cover this milestone: `ErrNotInVocabulary` for a finish reason outside the vocabulary, `ErrOutOfRange` for a negative count. AI-04.1's append discipline is what makes waiting for a real case cheap.
- **No completion event, no vendor mapping, no price.** AI-15.2, AI-31.1, Layer 3.
