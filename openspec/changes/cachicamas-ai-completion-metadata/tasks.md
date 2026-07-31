# Tasks — finish reasons and usage

> **Change**: `cachicamas-ai-completion-metadata`
> **Milestone**: AI-13 · **Nodes**: AI-13.1, AI-13.2, AI-13.3, AI-13.4 — all `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-completion-metadata/spec.md`, `design.md`
> **Branch**: `feat/ai-13-completion-metadata`
> **Depends on**: AI-00 … AI-03 merged; AI-04 merged on the Wave 1 branch
> **Blocks**: AI-15.2, AI-31
> **Evidence gate**: recorded green `make test` in `backend/agent/` (`go test -race -v ./...`)

---

## Node types and what they mean for this list

All four nodes are `[leaf]`: each closes only when every test-list item is taken red → green → refactored, **in order**. Two items are `*(pin)*` — AI-13.1 item 5 and AI-13.4 item 2 — which doc 0002 exempts from red-first and does **not** exempt from being mechanical. Each must be recorded biting against a deliberate scratch violation.

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so every red step follows `design.md` § 8: land the narrowest stub that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

All output blocks below are verbatim from `go test -race`, pasted after the run. Nothing in this file is written before the run it records.

---

## Order of the two chains, and why

doc 0002's AI-13 graph has two independent chains: `AI-13.1 → AI-13.2` (finish reasons) and `AI-13.3 → AI-13.4` (usage). AI-13.3 is explicitly marked *parallel with* AI-13.1, so either order is legal. **The finish-reason chain runs first.**

The defence, so it is a choice and not an accident:

1. **It is the milestone's headline.** AI-13 sits where it does because it absorbs **G12(c)** — doc 0001 § 3.2's row and § 7's `G12` entry both name refusal and pause as the reason the milestone cannot wait. Landing the charter's own reason first means the riskiest downstream commitment (the vocabulary AI-31.1 must map every vendor onto, and AI-40 freezes) is settled while the review budget is untouched.
2. **It is the walking skeleton.** doc 0002 § "Ordering inside a milestone" asks for the thinnest end-to-end path first. A provider string entering `NormalizeFinishReason` and leaving as a validated vocabulary value is that path; the usage record is a data structure with no path through it.
3. **The usage chain carries the decision that most needs a fresh reviewer.** `design.md` § 5.3's inclusive-versus-exclusive call is the one a wrong answer makes expensive and silent. Putting it second puts it in its own commit, against a diff a reviewer reaches with the vocabulary already understood.

No dependency runs the other way, so this ordering costs nothing if it is ever disputed.

---

## Review Workload Forecast

Forecast recorded **before** the first test. The actual column is filled after the last one, because the difference is the point.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `specs/ai-completion-metadata/spec.md`, `design.md`, `tasks.md` | ~600 prose | _pending_ | Low | 25 min |
| AI-13.1 + AI-13.2 | `src/ai/finish_reason.go`, `src/ai/finish_reason_test.go` | ~380 Go | _pending_ | **Medium** — AI-31.1 maps every vendor onto it; AI-40 freezes it | 30 min |
| AI-13.3 + AI-13.4 | `src/ai/usage.go`, `src/ai/usage_test.go` | ~370 Go | _pending_ | **Medium** — the inclusive/exclusive decision | 30 min |
| **Total** | 4 new Go files, 5 new markdown files | ~750 Go | _pending_ | **Medium** | **~85 min** |

### Budget reassessment — trigger 4 fires, and this is the record of it

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when the projected diff pushes the milestone past the review budget. **The forecast already exceeds it**, so this is the reassessment made in advance rather than a silent overrun discovered afterwards. It is revisited against the actual figures in the verification pass.

- **The milestone is genuinely two contracts, and doc 0002 says so.** Its charter names two deliverables ("a closed finish-reason vocabulary … a usage record …") and its graph has two chains that share no symbol. Split trigger 5 — "two agents could work it concurrently without conflict" — is satisfied *between the chains* and by nothing else.
- **The split is cut into history rather than deferred.** One commit per leaf-chain boundary, so if a reviewer wants two PRs the cut is `finish_reason*` against `usage*` and no rework is needed.
- **The argument against actually splitting.** `CAP-R-03` makes the two values one capability: "a stream that finishes normally ends with a completion event carrying a finish reason **and** a usage record". AI-15.2 consumes both in one event. Landing half of `CAP-R-03` is the partial-contract state defect **C4** came from. The recommendation is one PR reviewed in two passes, in commit order.
- **No other split trigger fires.** Each leaf's test list is 2 to 5 items against a limit of 7.

---

## Phase AI-13.1 — The finish-reason vocabulary `[leaf]`

Five test-list items, in doc 0002's order.

### T-ACM-1 — Item 1: the vocabulary is closed and each value is constructible

- [x] **RED** — `TestFinishReason_TheVocabulary_IsClosedAndEachValueIsConstructible`, naming all seven constants, against a stub whose `String` returned `""` and whose `Validate` returned `nil` for everything.

```
--- FAIL: TestFinishReason_TheVocabulary_IsClosedAndEachValueIsConstructible (0.00s)
    --- FAIL: .../the_zero_value_names_no_finish_reason (0.00s)
        finish_reason_test.go:50: FinishReason(0).Validate() = <nil>, want a violation — the zero value must name no finish reason
    --- FAIL: .../every_value_of_the_vocabulary_is_constructible_and_valid (0.00s)
        finish_reason_test.go:73: FinishReason(1).String() = "", want a non-empty stable string form
        finish_reason_test.go:73: FinishReason(2).String() = "", want a non-empty stable string form
        finish_reason_test.go:73: FinishReason(3).String() = "", want a non-empty stable string form
        finish_reason_test.go:73: FinishReason(4).String() = "", want a non-empty stable string form
        finish_reason_test.go:73: FinishReason(5).String() = "", want a non-empty stable string form
        finish_reason_test.go:73: FinishReason(6).String() = "", want a non-empty stable string form
        finish_reason_test.go:73: FinishReason(7).String() = "", want a non-empty stable string form
    --- FAIL: .../the_seven_values_and_their_string_forms_are_pairwise_distinct (0.00s)
        finish_reason_test.go:91: string form "" is shared by two values of the vocabulary
        (× 6)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.311s
```

- [x] **GREEN** — the constant block with the blank `iota` zero, `finishReasonLimit` derived from the last constant, the name array sized by that bound, and `Validate` reporting `ErrNotInVocabulary` through `FirstFailure`.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.326s
```

- [x] **REFACTOR** — none needed; the implementation is a constant block, an array and a bounds check. One speculative `strings` import written by reflex was removed before the green run was recorded.

**Proves:** `R-ACM-001` (`S-ACM-001`, `S-ACM-002`), `R-ACM-002` (`S-ACM-003`, `S-ACM-004`). The zero-value subtest asserts the class **and** the position, so a `Validate` that rejected everything with a bare error would still fail.

---

### T-ACM-2 — Item 2: refusal and content filter are distinct, and the line is documented

- [x] **RED** — `TestFinishReason_RefusalAndContentFilter_AreDistinctValues`, against a stub `NormalizeFinishReason` returning `FinishReasonUnknown` for every input.

```
--- FAIL: TestFinishReason_RefusalAndContentFilter_AreDistinctValues (0.00s)
    --- FAIL: .../the_refusal_family (0.00s)
        finish_reason_test.go:129: NormalizeFinishReason("refusal") = unknown, want refusal
        finish_reason_test.go:129: NormalizeFinishReason("refused") = unknown, want refusal
    --- FAIL: .../the_content-filter_family (0.00s)
        finish_reason_test.go:139: NormalizeFinishReason("content_filter") = unknown, want content_filter
        finish_reason_test.go:139: NormalizeFinishReason("safety") = unknown, want content_filter
        finish_reason_test.go:139: NormalizeFinishReason("recitation") = unknown, want content_filter
        finish_reason_test.go:139: NormalizeFinishReason("prohibited_content") = unknown, want content_filter
FAIL	github.com/cachicamas/backend/agent/src/ai	0.438s
```

- [x] **GREEN** — the lookup table, carrying **only** the two families this item demands. `ok ... 1.309s`.
- [x] **REFACTOR** — none.

**Proves:** `R-ACM-003` (`S-ACM-005`, `S-ACM-006`). Note which assertion is load-bearing: `FinishReasonRefusal != FinishReasonContentFilter` is nearly vacuous, so the weight is on the family assertions — no provider string about a filter may land on the model's decision to decline, and none about declining may land on the filter. The line itself is documented on the two declarations (`design.md` § 4.4).

---

### T-ACM-3 — Item 3: provider strings normalize after trimming and lowering

- [x] **RED** — `TestNormalizeFinishReason_ProviderStrings_MapIntoTheVocabulary`: 21 cases across all seven families plus casing and whitespace variants, against the two-family table from item 2.

```
--- FAIL: TestNormalizeFinishReason_ProviderStrings_MapIntoTheVocabulary (0.00s)
    finish_reason_test.go:199: NormalizeFinishReason("stop") = unknown, want stop
    finish_reason_test.go:199: NormalizeFinishReason("end_turn") = unknown, want stop
    finish_reason_test.go:199: NormalizeFinishReason("stop_sequence") = unknown, want stop
    finish_reason_test.go:199: NormalizeFinishReason("complete") = unknown, want stop
    finish_reason_test.go:199: NormalizeFinishReason("length") = unknown, want length
    finish_reason_test.go:199: NormalizeFinishReason("max_tokens") = unknown, want length
    finish_reason_test.go:199: NormalizeFinishReason("max_output_tokens") = unknown, want length
    finish_reason_test.go:199: NormalizeFinishReason("tool_calls") = unknown, want tool_calls
    finish_reason_test.go:199: NormalizeFinishReason("tool_use") = unknown, want tool_calls
    finish_reason_test.go:199: NormalizeFinishReason("function_call") = unknown, want tool_calls
    finish_reason_test.go:199: NormalizeFinishReason("pause_turn") = unknown, want pause_turn
    finish_reason_test.go:199: NormalizeFinishReason("pause") = unknown, want pause_turn
    finish_reason_test.go:199: NormalizeFinishReason("  stop  ") = unknown, want stop
    finish_reason_test.go:199: NormalizeFinishReason("STOP") = unknown, want stop
    finish_reason_test.go:199: NormalizeFinishReason("MAX_TOKENS") = unknown, want length
    finish_reason_test.go:199: NormalizeFinishReason("\tSAFETY\n") = unknown, want content_filter
    finish_reason_test.go:199: NormalizeFinishReason(" Tool_Use ") = unknown, want tool_calls
    --- FAIL: .../every_string_form_of_the_vocabulary_round-trips (0.00s)
        finish_reason_test.go:208: NormalizeFinishReason("stop") = unknown, want stop
        finish_reason_test.go:208: NormalizeFinishReason("length") = unknown, want length
        finish_reason_test.go:208: NormalizeFinishReason("tool_calls") = unknown, want tool_calls
        finish_reason_test.go:208: NormalizeFinishReason("pause_turn") = unknown, want pause_turn
FAIL	github.com/cachicamas/backend/agent/src/ai	0.322s
```

- [x] **GREEN** — the full table, and `strings.TrimSpace` then `strings.ToLower` before the lookup. `ok ... 1.314s`.
- [x] **REFACTOR** — none. No hyphen folding and no prefix matching were added: `design.md` § 4.2 records why the normalizer stays exactly as clever as the spec says.

**Proves:** `R-ACM-004` (`S-ACM-007`, `S-ACM-008`), `R-ACM-005` (`S-ACM-011`).

---

### T-ACM-4 — Item 4: an unrecognized string maps to unknown without error

**A TDD slip, recorded rather than hidden.** Item 3's green step was written with a miss branch returning `FinishReasonUnknown` — production behavior that no failing test had asked for, and precisely what this item exists to drive in. It was **reverted to the minimal form that still passes items 1 to 3** (`return finishReasonBySpelling[spelling]`, a bare map index) so that this item's red could be taken honestly.

The revert produced a better red than the one forecast. A Go map returns the zero value on a miss, so the minimal implementation handed a consumer `FinishReason(0)` — a value outside the closed vocabulary, rendering as the placeholder — for every string it did not know. The bug class doc 0002 names is a crash; this is its quieter sibling, and it is the one that would actually have shipped.

- [x] **RED** — `TestNormalizeFinishReason_UnrecognizedString_MapsToUnknownWithoutError`, eight cases.

```
--- FAIL: TestNormalizeFinishReason_UnrecognizedString_MapsToUnknownWithoutError (0.00s)
    --- FAIL: .../a_vendor_value_added_after_this_table (0.00s)
        finish_reason_test.go:252: NormalizeFinishReason("model_context_window_exceeded") = invalid, want unknown
    --- FAIL: .../a_real_spelling_with_a_suffix (0.00s)
        finish_reason_test.go:252: NormalizeFinishReason("stop_reason") = invalid, want unknown
    --- FAIL: .../the_placeholder_for_a_value_outside_the_vocabulary (0.00s)
        finish_reason_test.go:252: NormalizeFinishReason("invalid") = invalid, want unknown
    --- FAIL: .../the_empty_string (0.00s)
        finish_reason_test.go:252: NormalizeFinishReason("") = invalid, want unknown
    --- FAIL: .../an_enormous_value (0.00s)
        finish_reason_test.go:252: NormalizeFinishReason("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx") = invalid, want unknown
    --- FAIL: .../whitespace_only (0.00s)
        finish_reason_test.go:252: NormalizeFinishReason("   \t\n  ") = invalid, want unknown
    --- FAIL: .../control_bytes (0.00s)
        finish_reason_test.go:252: NormalizeFinishReason("\x00\x01\x02") = invalid, want unknown
    --- FAIL: .../a_near-miss_of_a_real_spelling (0.00s)
        finish_reason_test.go:252: NormalizeFinishReason("end-turn") = invalid, want unknown
FAIL	github.com/cachicamas/backend/agent/src/ai	0.333s
```

- [x] **GREEN** — the comma-ok lookup with the `FinishReasonUnknown` fallback restored. `ok ... 1.312s`.
- [x] **REFACTOR** — none.

**Proves:** `R-ACM-004` (`S-ACM-009`, `S-ACM-010`), `R-ACM-005` (`S-ACM-012`), `NFR-ACM-B`. Each subtest recovers from a panic and fails on it, so the crash class is covered as well as the leak class.

---

### T-ACM-5 — Item 5 *(pin)*: exhaustiveness

- [x] **PIN LANDED GREEN** — `TestFinishReason_AddingAValue_FailsWithoutATableAndAStringForm`. `ok ... 1.318s`. Exempt from red-first by doc 0002's leaf anatomy.
- [x] **REFACTOR** — the first bite proof exposed a weakness and the refactor closed it. `finishReasonLimit` was originally declared *outside* the constant block as `FinishReasonUnknown + 1`, so appending a value required a second, separate edit to the bound before the pin would notice it. The bound now lives inside the constant block as its last member, so it moves with the vocabulary and no second edit exists to be forgotten. Re-run after the refactor: `ok ... 1.371s`.
- [x] **SHOWN TO BITE** — one scratch constant appended above the bound, nothing else changed:

```
--- FAIL: TestFinishReason_AddingAValue_FailsWithoutATableAndAStringForm (0.00s)
    finish_reason_test.go:296: the package validates FinishReason(8), which this test does not name — add it to theVocabulary and give it an obligation
    finish_reason_test.go:300: FinishReason(8) validates but renders as the placeholder "invalid" — add it to finishReasonNames
    finish_reason_test.go:304: FinishReason(8): NormalizeFinishReason("invalid") = unknown, want the value back — add it to finishReasonBySpelling
    finish_reason_test.go:310: the package validates 8 values, want exactly the 7 named in this test
FAIL	github.com/cachicamas/backend/agent/src/ai	0.297s
```

Scratch removed; the suite returns to `ok`. Four failures for one omission, each naming the file to edit.

**Proves:** `R-ACM-006` (`S-ACM-013`, `S-ACM-014`). The walk covers all 256 values of the underlying `uint8`, so there is no bound in the test to keep in step with the package and the pin cannot rot. The placeholder is read from `ai.FinishReason(0).String()` rather than written out, so renaming it does not silently disarm the assertion.

---

## Phase AI-13.2 — Three-way distinguishability `[leaf]`

Two test-list items — **and a finding about the plan, reported rather than papered over.**

Neither item drives production code. AI-13.1's landed constants already make refusal, pause-turn and unknown three distinct values, so both of AI-13.2's tests were green the first time they ran. doc 0002 marks neither item `*(pin)*`, but in practice both behave as pins: they assert a property an earlier leaf established rather than driving a new one. That is not a defect in the plan — the property is worth its own leaf precisely because it is the one G12(c) says gets lost — but "red first" was not available, and inventing a stub to manufacture one would have been theatre.

Both are therefore recorded the way doc 0002 records a guard leaf: **landed green, then shown to bite** against a deliberate reintroduction of the defect. The revert-and-record clause did not fire; no graph amendment is proposed. What is proposed is a note for whoever next reads AI-13.2: its two items would be more honestly marked `*(pin)*`.

### T-ACM-6 — Item 1: three states, and collapsing them is compile-visible

- [x] **LANDED GREEN** — `TestFinishReason_RefusalPauseAndUnknown_AreThreeSeparateStates`, with `consumerResponse` — an exhaustive Layer 2-shaped switch whose fallback branch fails the test. `ok ... 1.323s`. No red was available; see the phase note above.

- [x] **SHOWN TO BITE (1) — the collapse does not compile.** The G12(c) defect written out literally: `FinishReasonRefusal` and `FinishReasonPauseTurn` removed from the `iota` block and re-declared as `= FinishReasonUnknown`, exactly the state doc 0001 § 3.2 describes as "both currently collapse into the unknown fallback".

```
# github.com/cachicamas/backend/agent/src/ai_test [github.com/cachicamas/backend/agent/src/ai.test]
src/ai/finish_reason_test.go:351:7: duplicate case ai.FinishReasonPauseTurn (constant 5 of uint8 type ai.FinishReason) in expression switch
	src/ai/finish_reason_test.go:349:9: previous case
src/ai/finish_reason_test.go:353:7: duplicate case ai.FinishReasonUnknown (constant 5 of uint8 type ai.FinishReason) in expression switch
	src/ai/finish_reason_test.go:349:9: previous case
FAIL	github.com/cachicamas/backend/agent/src/ai [build failed]
FAIL
```

This is the strongest available evidence for `S-ACM-017`, and it is stronger than a failing assertion: the **compiler** rejects the collapse in every consumer that switches exhaustively, because two collapsed constants are a duplicate case. "Reintroducing the collapse requires a compile-visible change" is not asserted here — it is enforced by the language.

- [x] **SHOWN TO BITE (2) — a partial collapse that still compiles.** The values kept distinct but their string forms collapsed onto `"unknown"`, which is how the distinction dies at the boundary where a consumer logs or serialises it:

```
--- FAIL: TestFinishReason_RefusalPauseAndUnknown_AreThreeSeparateStates (0.00s)
    --- FAIL: .../the_three_are_distinct_values_with_distinct_string_forms (0.00s)
        finish_reason_test.go:384: unknown and unknown share the string form "unknown", want three
        finish_reason_test.go:384: unknown and unknown share the string form "unknown", want three
        finish_reason_test.go:384: unknown and unknown share the string form "unknown", want three
FAIL	github.com/cachicamas/backend/agent/src/ai	0.304s
```

Both mutations reverted; the suite returns to `ok`.

- [x] **REFACTOR** — none.

**Proves:** `R-ACM-007` (`S-ACM-015`, `S-ACM-016`, `S-ACM-017`).

---

### T-ACM-7 — Item 2: the obligation attached to each value

The obligation is documentation, because `V-OUT-10` keeps the decision in Layer 2. doc 0002 nevertheless puts it in a **test list** rather than in a verify-report checklist, and its own leaf anatomy says prose claims with no objective check do not belong in a test list. So the claim is made objective: the test parses `finish_reason.go` with `go/parser` and asserts the required clauses on each constant's documentation comment. doc 0002 already names AST scans as a guard mechanism, and `go/ast` is standard library, so the zero-require rule holds.

- [x] **LANDED GREEN** — `TestFinishReason_EachValue_CarriesItsDocumentedObligation`, seven constants plus a whole-vocabulary response-distinctness subtest. `ok ... 1.310s`.

- [x] **SHOWN TO BITE** — the pause-turn obligation replaced with the plausible-sounding but wrong "the consumer should send the request again", which is the defect: re-sending a request is not resuming a turn.

```
--- FAIL: TestFinishReason_EachValue_CarriesItsDocumentedObligation (0.00s)
    --- FAIL: .../FinishReasonPauseTurn (0.00s)
        finish_reason_test.go:500: FinishReasonPauseTurn documentation does not state "obligation"
        finish_reason_test.go:500: FinishReasonPauseTurn documentation does not state "verbatim"
        finish_reason_test.go:500: FinishReasonPauseTurn documentation does not state "AI-31.1"
FAIL	github.com/cachicamas/backend/agent/src/ai	0.305s
```

Reverted; the suite returns to `ok`.

- [x] **REFACTOR** — none.

**Proves:** `R-ACM-008` (`S-ACM-018`, `S-ACM-019`). The recorded pause-turn obligation is *resume the turn, replaying the content already received verbatim*, and the test requires the citation of **AI-31.1** to survive alongside it, so the node that must honor it cannot be edited out of the comment quietly. `S-ACM-019` — that no method answers "should the loop continue?" — is held by `design.md` § 4.3 and by the surface itself: the package exports no such method.

---

## Phase AI-13.3 — Usage: absent is not zero `[leaf]`

Three test-list items.

### T-ACM-8 — Item 1: absent is distinguishable from zero on every count

- [x] **RED** — `TestUsage_AnAbsentCount_IsDistinguishableFromZero`, against a `TokenCount` carrying only an `int64` and a `Count` returning `(t.count, true)`.

```
--- FAIL: TestUsage_AnAbsentCount_IsDistinguishableFromZero (0.00s)
    --- FAIL: .../cache_write (0.00s)
        usage_test.go:55: absent cache_write: Count() = (0, true), want (0, false) — an unreported count must not report as present
        usage_test.go:72: absent cache_write and cache_write reported as nought both render as "0", want two renderings
    --- FAIL: .../reasoning (0.00s)
        usage_test.go:55: absent reasoning: Count() = (0, true), want (0, false) — an unreported count must not report as present
        usage_test.go:72: absent reasoning and reasoning reported as nought both render as "0", want two renderings
    --- FAIL: .../input (0.00s)
        usage_test.go:55: absent input: Count() = (0, true), want (0, false) — an unreported count must not report as present
        usage_test.go:72: absent input and input reported as nought both render as "0", want two renderings
    --- FAIL: .../cache_read (0.00s)
        usage_test.go:55: absent cache_read: Count() = (0, true), want (0, false) — an unreported count must not report as present
        usage_test.go:72: absent cache_read and cache_read reported as nought both render as "0", want two renderings
    --- FAIL: .../output (0.00s)
        usage_test.go:55: absent output: Count() = (0, true), want (0, false) — an unreported count must not report as present
        usage_test.go:72: absent output and output reported as nought both render as "0", want two renderings
FAIL	github.com/cachicamas/backend/agent/src/ai	0.429s
```

- [x] **GREEN** — `TokenCount{count, present}`, `Tokens` setting the flag, `Count` returning both, `String` rendering `"absent"` for the zero value. `ok ... 1.462s`.
- [x] **REFACTOR** — none.

**Proves:** `R-ACM-009` (`S-ACM-020`, `S-ACM-021`, `S-ACM-022`). The table runs the assertion once per field rather than once, because `V-MET-10` makes each count independently present or absent and a property proven on `Input` alone would be a property of `Input`. The rendering assertion is not decoration: a log line is where the distinction is otherwise lost first.

---

### T-ACM-9 — Item 2: constructible with any subset present

- [x] **RED** — `TestUsage_AnySubsetOfCounts_ProducesAValidRecord`, against a stub `Validate` returning `Invalid(ErrEmpty, at...)` — a validator with an opinion about presence, which is the defect `CAP-R-03` clause 2 forbids.

```
--- FAIL: TestUsage_AnySubsetOfCounts_ProducesAValidRecord (0.00s)
    --- FAIL: .../several_negative_counts_report_the_first_in_the_documented_order (0.00s)
        usage_test.go:171: violation position = "usage", want "usage.output" — output precedes cache_read and reasoning
    --- FAIL: .../a_provider_that_reports_nothing (0.00s)
        usage_test.go:119: Validate() = usage: required value is empty, want <nil> — any subset of counts is a valid record
    --- FAIL: .../a_provider_that_reports_only_input_and_output (0.00s)
        usage_test.go:119: Validate() = usage: required value is empty, want <nil> — any subset of counts is a valid record
    --- FAIL: .../a_negative_count_is_rejected_out_of_range_at_its_own_field (0.00s)
        usage_test.go:145: errors.Is(err, ErrOutOfRange) = false, want true; err = completion.usage: required value is empty
        usage_test.go:153: violation position = "completion.usage", want "completion.usage.cache_read"
    --- FAIL: .../a_provider_that_reports_everything (0.00s)
        usage_test.go:119: Validate() = usage: required value is empty, want <nil> — any subset of counts is a valid record
    --- FAIL: .../a_provider_that_reports_only_a_reported_nought (0.00s)
        usage_test.go:119: Validate() = usage: required value is empty, want <nil> — any subset of counts is a valid record
FAIL	github.com/cachicamas/backend/agent/src/ai	0.289s
```

- [x] **GREEN** — `Validate` built from `FirstFailure` over five `Rule` closures in the documented field order, each reporting `ErrOutOfRange` at its own field. `ok ... 1.315s`.
- [x] **REFACTOR** — the five rules were extracted to one `nonNegative` helper.

**A claim withdrawn after checking it.** The helper clones the caller's position with `append(slices.Clone(at), At(name))`, and the first version of its comment — and of `design.md` § 5.2 — justified that by saying five closures appending to one variadic slice would overwrite each other's last step. **That is false**, and it was checked rather than believed: with a bare `append` the whole suite still passes, because `FirstFailure` evaluates lazily and stops at the first failure and `Invalid` copies the position it is given, so at most one rule ever builds one.

```
(bare append, no slices.Clone)
ok  	github.com/cachicamas/backend/agent/src/ai	1.292s
```

The clone is kept for the narrower reason that is true: `at` is the caller's slice, and appending to a variadic parameter can write into the caller's backing array past its length. Both the declaration comment and `design.md` § 5.2 now say that instead, and both record that the tests do not distinguish the two.

**Proves:** `R-ACM-010` (`S-ACM-023`, `S-ACM-024`, `S-ACM-025`). The nested-position case (`completion.usage.cache_read`) is load-bearing for a different reason than first supposed: it proves the caller's prefix survives into the reported position rather than being replaced by it.

---

### T-ACM-10 — Item 3: readable from an external package, field by field

Landed green: items 1 and 2 already exported the fields and surfaced absence, so no red was available. Recorded the way AI-13.2's items were, with two mutations that between them cover both halves of the item.

- [x] **LANDED GREEN** — `TestUsage_FromAnExternalPackage_IsReadableFieldByField`, written **without** the `usageFields` helper because the point of the item is the shape a real consumer writes. `ok ... 1.292s`.

- [x] **SHOWN TO BITE (1) — "readable field by field".** One field unexported:

```
# github.com/cachicamas/backend/agent/src/ai_test [github.com/cachicamas/backend/agent/src/ai.test]
src/ai/usage_test.go:36:57: u.CacheWrite undefined (type *ai.Usage has no field or method CacheWrite, but does have unexported field cacheWrite)
src/ai/usage_test.go:109:4: unknown field CacheWrite in struct literal of type ai.Usage, but does have unexported cacheWrite
src/ai/usage_test.go:205:40: usage.CacheWrite undefined (type ai.Usage has no field or method CacheWrite, but does have unexported field cacheWrite)
FAIL	github.com/cachicamas/backend/agent/src/ai [build failed]
```

- [x] **SHOWN TO BITE (2) — "with absence surfaced rather than defaulted".** `Count` returning `(t.count, true)`:

```
--- FAIL: TestUsage_FromAnExternalPackage_IsReadableFieldByField (0.00s)
    usage_test.go:212: cache_write never reported: Count() = (0, true), want (0, false)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.281s
```

Both reverted; the suite returns to `ok`.

- [x] **REFACTOR** — none.

**Proves:** `R-ACM-009`, `R-ACM-010`. This is retired defect **C2**'s equivalent for the usage record — a contract unreadable from another package makes translation structurally impossible — proven one milestone before AI-06.2 makes the same proof for content parts. `src/agenttest/` is untouched: its own comment reserves that file for AI-06.2.

---

## Phase AI-13.4 — Cost-formula clarity `[leaf]`

Two test-list items.

### T-ACM-11 — Item 1: the inclusive-or-exclusive semantics, pinned against a cache-hit record

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-011` (`S-ACM-026`, `S-ACM-027`, `S-ACM-028`).

---

### T-ACM-12 — Item 2 *(pin)*: the formula's term list is the field set

- [ ] **PIN LANDED GREEN**
- [ ] **SHOWN TO BITE** against a scratch field, then scratch removed
- [ ] **REFACTOR**

**Proves:** `R-ACM-012` (`S-ACM-029`, `S-ACM-030`, `S-ACM-031`).

---

## Verification pass

- [ ] `make test` in `backend/agent/` — green, `-race`, every package. Tail recorded.
- [ ] `make lint` — `go vet` plus `golangci-lint` at the pinned `v2.9.0`, clean.
- [ ] `go.mod` still declares **zero requires**.
- [ ] Both AI-00 import guards pass.
- [ ] `validation.go` and `validation_test.go` untouched — this change is a consumer of AI-04, not an editor of it.
- [ ] No file under `openspec/specs/` touched: this milestone needs no register amendment.
- [ ] Revert-and-record clause: whether it fired, and any doc 0002 amendment it required.

---

## What a later milestone inherits

| Node | Inherits |
| --- | --- |
| **AI-15.2** | Both values, ready to be carried by a completion event, and AI-13.3's absent-versus-zero property to preserve across the event boundary — which that node's test list already restates as its own item |
| **AI-23.8** | The conformance case for absent-versus-zero in usage, which `openspec/specs/ai-minimum-capabilities/spec.md` § 11 marks **required** despite its node |
| **AI-31.1** | The vocabulary every vendor stop value maps onto, and the recorded pause-turn obligation — resume, replaying received content verbatim |
| **AI-31.3** | The inclusive/exclusive semantics to verify against a real cache-hit transcript |
| **Layer 2 / Layer 3** | Honest token counts, with `V-OUT-07` cost events and `V-OUT-08` prices left entirely to them |
| **AI-05's reviewer** | Two closed vocabularies built without sight of each other. `design.md` § 3.2 records that a reconciliation pass is expected and that nothing here anticipates it |
