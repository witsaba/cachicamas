# Tasks — the validation error taxonomy

> **Change**: `cachicamas-ai-validation-errors`
> **Milestone**: AI-04 · **Nodes**: AI-04.1 `[decision]`, AI-04.2 `[leaf]`, AI-04.3 `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `decision.md`, `specs/ai-validation-errors/spec.md`, `design.md`
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-1`
> **Depends on**: AI-00, AI-01, AI-02, AI-03 merged
> **Blocks**: AI-05 … AI-13, AI-19
> **Evidence gate**: recorded green `make test` in `backend/agent/` (`go test -race -v ./...`)

---

## Node types and what they mean for this list

| Node | Type | Closes on |
| --- | --- | --- |
| AI-04.1 | `[decision]` | The decision artifact answers every closing-checklist item and is merged. **No production code.** |
| AI-04.2 | `[leaf]` | Every test-list item taken red → green → refactored, **in order** |
| AI-04.3 | `[leaf]` | Same, with item 3 exempt from red-first as a `*(pin)*` and still fully mechanical |

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so every red step below follows `design.md` § 6: land the narrowest stub that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

---

## Review Workload Forecast

Recorded after the fact, with the forecast that preceded it, because the difference is the point.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `specs/ai-validation-errors/spec.md`, `design.md`, `tasks.md` | ~700 prose | **718 prose** | Low | 25 min |
| AI-04.1 decision + register amendment | `decision.md`, `openspec/specs/ai-contract-vocabulary/spec.md` | ~230 prose | **248 prose, 2 appended rows** | **Medium** — nine milestones inherit it | 25 min |
| AI-04.2 | `src/ai/validation.go`, `src/ai/validation_test.go` | ~250 Go | **~560 Go** | **Medium** — Layer 1's first non-guard surface | 35 min |
| AI-04.3 | same two files | ~180 Go | **~345 Go** | Low — properties over the landed surface | 20 min |
| **Total** | 7 files (6 new markdown, 2 new Go, 1 amended markdown) | ~430 Go | **905 Go (538 non-comment), 966 prose** | **Medium** | **~105 min** |

### Budget reassessment — trigger 4 fired, and this is the record of it

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when "its projected diff, tests included, pushes the milestone past the review budget". **It did.** The Go diff is 905 lines, of which 538 are non-comment: 127 lines of production code under 210 lines of contract documentation, and 411 lines of tests. The reassessment, rather than a silent overrun:

- **Production code is 127 lines.** The surface itself is well inside the budget. What is large is the test file and the GoDoc, in a milestone whose entire deliverable is a set of properties that nine later milestones stand on. doc 0002's own milestone rule is that "tests travel with the behavior they prove", and the properties here — matchability through wrapping, structural extraction, content-free rendering, determinism under a scheduler, totality — are not assertable in fewer cases than are written.
- **No other split trigger fires.** The three nodes are one publicly observable behavior (how a rule violation is reported); they are strictly ordered, so two agents could not work them concurrently; none needed a seam that did not exist; and both test lists are 3 items against a limit of 7.
- **The chain boundary is already cut.** Trigger 4 says "the node boundary is the PR-chain boundary", and the commits are split exactly there: `feat(ai): add inspectable validation sentinels (AI-04.2)` and `test(ai): prove validation failures are deterministic and content-free (AI-04.3)`. If the reviewer wants two PRs, no rework is required — the split already exists in history.
- **The argument against actually splitting is unchanged.** Merging AI-04.2 alone lands a failure type whose determinism and totality are unproven, which is the partial-contract state defect **C4** came from. The recommendation is one PR reviewed in two passes, in commit order.

---

## Phase AI-04.1 — Taxonomy boundary `[decision]`

**Deliverable:** `openspec/changes/cachicamas-ai-validation-errors/decision.md`.

### T-AIE-1 — The boundary, applied to cases the register did not reach

- [x] State the caller-contract / provider-transport line by citing register § 6.3 rather than paraphrasing it, give examples on both sides, and resolve at least one borderline case beyond the register's own.

**Required by the checklist:** item 1.

**Decided:** the rule is inherited verbatim. Two clauses of it are made load-bearing because they are the ones dropped on recall — *"from the request alone"* and *"not who is to blame"*. Ten examples split across the two sides. **Four** borderline cases: **A** (unrecognised model identity) cited from the register; **B** a provider cap smaller than `V-REQ-24`'s documented breakpoint cap — caller-contract for the documented cap, provider/transport for the provider's own, with the corollary that a later provider rejection is evidence the documented cap is stale rather than that the boundary is wrong; **C** a tool result reporting the tool failed — **neither** vocabulary, per `V-REQ-18`, which proves the taxonomy is not a partition of everything unpleasant; **D** an already-expired call signal — decidable without I/O yet **not** caller-contract, because it is not a property of the request value, which sharpens the rule into "necessary but not sufficient".

**Evidence:** `decision.md` § 2 (`R-AIE-001`, `S-AIE-003`). Cases B and D are also stated on the package surface, in `validation.go`'s file documentation, so a reader who never opens the SDD still meets the boundary.

---

### T-AIE-2 — Granularity, with the `errors.Is` consequence

- [x] Decide what a sentinel is one of, argue the rejected granularity at full strength first, and state what the choice does to `errors.Is`.

**Required by the checklist:** item 2. Documented default: one sentinel per rule class, reusable across types.

**Decided:** the default, taken — but not because it is the middle option. The per-instance case is argued at full strength (five affirmative grounds) and defeated on four: it *is* the recorded defect rather than resembling it; the exported surface grows without bound and freezes at AI-40; the common query degenerates into an enumeration, which is `V-FAIL-06`'s recorded failure mode; and the information is not lost, because a rule instance is the pair *(class, position)* and the position is carried anyway. The per-category extreme is rejected on `V-FAIL-06`'s own reasoning. **Consequence:** `errors.Is` answers *which class*, `errors.As` answers *where*, and a consumer asking the compound question writes both — the conceded cost.

**Landed as five classes**, each with a citable case: `ErrEmpty`, `ErrNotInVocabulary`, `ErrOutOfRange`, `ErrMalformed`, `ErrUnresolvedReference`. A sixth for conflicting values is plausible, has no citable case, and is deliberately not landed.

**Evidence:** `decision.md` § 3, two-axis table in § 3.4 (`R-AIE-003`, `R-AIE-004`).

---

### T-AIE-3 — Aggregate versus short-circuit

- [x] Decide, argue the rejected policy at full strength first, and state the consequence and the reversal path.

**Required by the checklist:** item 3. Documented default: ordered first-failure.

**Decided:** the default, taken — and explicitly **not** on doc 0002's allocation argument, which the artifact concedes is the weakest of the three offered. Aggregation is argued at full strength (five grounds, including that first-failure is recoverable from an aggregate but not the reverse) and defeated on four: `errors.Join` matches a sentinel when *any* member matches, which takes back everything decision 2 bought; `errors.As` then yields whichever member traversal reaches first, making order contract by accident; determinism stops being structural and becomes a convention nine milestones must keep; and Layer 1's caller is code, not a form. The one-way door is recorded: an "every violation" sibling remains addable without changing the failure type, while narrowing an aggregate after AI-40 breaks every consumer.

**Evidence:** `decision.md` § 4 (`R-AIE-007`, `R-AIE-008`).

---

### T-AIE-4 — Positional context without a second error type

- [x] Decide how position attaches, argue the composed-wrapper alternative first, and settle the redaction posture that the position's shape implies.

**Required by the checklist:** item 4.

**Decided:** exactly **one** concrete failure type, always carrying a position that may be empty. The composed-wrapper alternative is defeated on `V-FAIL-03`'s own words — two `errors.As` targets *is* the second parallel vocabulary, and the wrapping order becomes observable contract that nothing enforces. The position is **structural only**: filtered names and integer indices, no caller-supplied value ever. "Which tool" is its index in the ordered tool set (`V-REQ-14`). Two constraints make this construction rather than discipline: a name failing the filter is replaced *whole* (a prefix of a secret is still a secret), and the rendered message reproduces the **registered class's** text, never the supplied error's.

**Evidence:** `decision.md` § 5, conceded costs in § 6 (`R-AIE-002`, `R-AIE-005`, `R-AIE-006`).

---

### T-AIE-5 — Register amendment (AI-01), append-only

- [x] Append the two nouns AI-04 needs, with the next free `V-FAIL` ordinals and a dated amendment blockquote; update the register's counts.

**Appended:** `V-FAIL-16` **validation rule** and `V-FAIL-17` **rule class**, both owned by AI-04, both in § 6.1.

**Discipline applied** (register § 9 rules 2 and 3): appended in the same pull request that needs them; not defined locally; no existing row renumbered, reworded, reordered or removed; the § 10 term counts updated from 114 to 116, the failure-category figure from 15 to 17, and the checklist row 4 identifier range from `V-FAIL-15` to `V-FAIL-17`.

**Evidence:** `openspec/specs/ai-contract-vocabulary/spec.md` § 6 and § 10 — a 10-line diff, entirely additive except the two count figures and the range (`R-AIE-012`, `S-AIE-033`).

---

## Phase AI-04.2 — Sentinels are inspectable `[leaf]`

Three test-list items, in doc 0002's order. One red → green → refactor cycle each. All output below is verbatim from `go test -race`.

### T-AIE-6 — Item 1: a wrapped failure still matches its sentinel

- [x] **RED** — `TestViolation_WrappedFailure_StillMatchesItsSentinel` against a stub `Invalid` that stores nothing and a `Violation` with no fields.

```
=== NAME  TestViolation_WrappedFailure_StillMatchesItsSentinel/wrapped_once
    validation_test.go:49: errors.Is(err, ErrEmpty) = false, want true
=== NAME  TestViolation_WrappedFailure_StillMatchesItsSentinel/unwrapped
    validation_test.go:49: errors.Is(err, ErrEmpty) = false, want true
=== NAME  TestViolation_WrappedFailure_StillMatchesItsSentinel/wrapped_three_times
    validation_test.go:49: errors.Is(err, ErrEmpty) = false, want true
--- FAIL: TestViolation_WrappedFailure_StillMatchesItsSentinel (0.00s)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.520s
```

- [x] **GREEN** — `Violation` stores the rule; `Unwrap` returns it. All three subtests `--- PASS`.
- [x] **REFACTOR** — none needed; the implementation is three lines.

**Proves:** `R-AIE-004`, `S-AIE-009`, `S-AIE-010`. The test also asserts a *different* sentinel does **not** match, so a match-everything implementation fails.

**Appended test case** (doc 0002 allows discovered cases to be appended to the owning leaf's list): a `*(pin)*` subtest, *a failure matches exactly one rule class*, iterating the five classes and asserting exactly one match each. It is green from birth and fails the day a later milestone appends a class by wrapping an existing one, which would make `errors.Is` ambiguous for every consumer. Proves `S-AIE-008`.

---

### T-AIE-7 — Item 2: positional context is extracted by `errors.As`

- [x] **RED** — `TestViolation_PositionalContext_IsExtractedByErrorsAs` against stub `At` / `AtIndex` returning a zero `Step`, and a `Path()` returning nil.

```
=== NAME  .../the_content_of_a_message
    validation_test.go:116: len(path) = 0, want 2
=== NAME  .../one_tool_of_the_declared_set
    validation_test.go:116: len(path) = 0, want 1
=== NAME  .../a_field_of_one_tool
    validation_test.go:116: len(path) = 0, want 2
=== NAME  .../a_field_with_no_index
    validation_test.go:116: len(path) = 0, want 1
--- FAIL: TestViolation_PositionalContext_IsExtractedByErrorsAs (0.00s)
    --- FAIL: .../the_returned_position_is_a_copy (0.00s)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.500s
```

- [x] **GREEN** — `Step` stores a name, an index and whether it has one; `Invalid` stores the path; both `Invalid` and `Path()` clone, which is what the copy subtest demanded.
- [x] **REFACTOR** — none. The name filter is deliberately **not** introduced here: nothing yet demands it, and item 3 is what drives it in.

**Proves:** `R-AIE-005`, `S-AIE-011` … `S-AIE-013`. The position is read **programmatically** — name and index per step — not by parsing the message. The four cases cover a field, a message's content part, a tool by index, and a field of one tool.

---

### T-AIE-8 — Item 3: the rendered message carries no caller content

This item took **two** red steps, and the second is the one that matters.

- [x] **RED 1** — the test asserts both halves at once: the message must name its rule class and its position, *and* must carry no caller content. Against the stub `Error()` returning `"invalid"`, the first half fails and the second passes vacuously — which is exactly why both halves are in one test.

```
=== NAME  .../the_message_names_its_rule_class_and_its_position
    validation_test.go:175: Error() = "invalid", want it to contain "required value is empty"
    validation_test.go:175: Error() = "invalid", want it to contain "messages[2]"
    validation_test.go:175: Error() = "invalid", want it to contain "content"
--- FAIL: TestViolation_RenderedMessage_CarriesNoCallerContent (0.00s)
```

- [x] **RED 2** — the *obvious* implementation, the one anybody writes: concatenate the position and the supplied error's own text. It satisfies the first half and demonstrates the leak in four different shapes.

```
    validation_test.go:213: Error() = "CACHICAMA-SENTINEL-BODY-9f3aCACHICAMA-SENTINEL-BODY-9f3a…: required value is empty",
        want it to contain no prefix of the sentinel body (found "CACHICAMA-SENTINEL-BODY-9f3a")
    validation_test.go:213: Error() = "tools[1].{\"api_key\":\"CACHICAMA-SENTINEL-BODY-9f3a\"}: value is not well-formed for its documented encoding", …
    validation_test.go:213: Error() = "model: rejected CACHICAMA-SENTINEL-BODY-9f3a: required value is empty", …
    validation_test.go:213: Error() = "CACHICAMA-SENTINEL-BODY-9f3a: required value is empty", …
--- FAIL: TestViolation_RenderedMessage_CarriesNoCallerContent (0.00s)
    --- PASS: .../the_message_names_its_rule_class_and_its_position (0.00s)
    --- PASS: .../a_rule_error_carrying_a_body_is_still_matchable (0.00s)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.356s
```

- [x] **GREEN** — the two structural constraints of `decision.md` § 5.3: `structuralName` filters at construction and replaces a non-structural name whole; `ruleText` renders the **registered class's** own text and never the supplied error's. All six subtests pass, including the one asserting the wrapped rule is still matchable.
- [x] **REFACTOR** — the file reorganised into its documented order, contract GoDoc written on every exported declaration, and `ruleClasses` established as the single registry that `ruleText` consults. Two branches added during the refactor without a test demanding them — a negative-index guard and an empty-name placeholder — were **reverted** and left for T-AIE-10 to drive in.

**Proves:** `R-AIE-006`, `S-AIE-014` … `S-AIE-017`. This is where the redaction posture starts, per `V-FAIL-13`.

---

## Phase AI-04.3 — Deterministic and content-free `[leaf]`

### T-AIE-9 — Item 1: the first violated rule, identically across runs and under `-race`

- [x] **RED** — `TestFirstFailure_SeveralViolatedRules_ReportsTheFirstInOrder` against a stub `FirstFailure` returning nil.

```
    validation_test.go:320: FirstFailure() = <nil>, want a violation
    validation_test.go:360: run 0: FirstFailure() = <nil>, want "messages[2]: value is outside a closed vocabulary"
    validation_test.go:389: goroutine 0: FirstFailure() = "", want "messages[2]: value is outside a closed vocabulary"
    … (64 goroutines) …
--- FAIL: TestFirstFailure_SeveralViolatedRules_ReportsTheFirstInOrder (0.00s)
    --- FAIL: .../reports_the_first_violated_rule_and_stops_there (0.00s)
    --- FAIL: .../repeated_evaluation_agrees (0.00s)
    --- FAIL: .../concurrent_evaluation_agrees (0.00s)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.247s
```

Note which subtest is **absent** from that list: *no violated rule yields a genuinely absent failure* passed against the nil-returning stub, as it must — that is the assertion pinning the typed-nil trap, and a stub returning `(*Violation)(nil)` would have failed it.

- [x] **GREEN** — `FirstFailure` iterates the slice, calls each rule lazily, skips a nil entry, returns the first non-nil violation, and returns a true nil `error` otherwise. All four subtests `--- PASS`.
- [x] **REFACTOR** — none; the loop is seven lines and is the whole determinism argument.

**Proves:** `R-AIE-007`, `R-AIE-008`, `S-AIE-018` … `S-AIE-023`. Rules 2, 3 and 5 of a five-rule list are violated; the reported failure is rule 2's, and the evaluation record is exactly `[1 2]` — rules 3, 4 and 5 never ran. 1,000 sequential evaluations and 64 concurrent goroutines under `-race` all report the same sentinel and the same rendered position, with no race reported.

---

### T-AIE-10 — Item 2: no input causes a panic

- [x] **RED** — `TestViolation_ExtremeInputs_NeverPanics`, a 13-case table: a nil failure rendered, matched and read; a nil rule; an unregistered rule; a zero-value step; an empty name; a negative index; the maximum index; a name at and far beyond the length bound; and a ten-thousand-step position. Three panics and four wrong renderings.

```
    validation_test.go:492: panicked: runtime error: invalid memory address or nil pointer dereference
    validation_test.go:502: act() = ": required value is empty", want it to contain "?"
    validation_test.go:502: act() = "model: value violates an unregistered rule", want it to contain "unnamed"
    validation_test.go:492: panicked: runtime error: invalid memory address or nil pointer dereference
    validation_test.go:502: act() = "value violates an unregistered rule", want it to contain "unnamed"
    validation_test.go:505: act() = "messages[-1]: required value is empty", want it not to contain "[-"
    validation_test.go:492: panicked: runtime error: invalid memory address or nil pointer dereference
--- FAIL: TestViolation_ExtremeInputs_NeverPanics (0.00s)
```

- [x] **GREEN** — nil-receiver guards on `Error`, `Unwrap` and `Path`; a distinct text for a nil rule, so "unnamed" and "unregistered" stay different facts; `Step.Name` reports the placeholder for a step with no name; `AtIndex` treats a negative index as no index. All 13 cases `--- PASS`.
- [x] **REFACTOR** — none beyond a comment paragraph break.

**Proves:** `R-AIE-009`, `S-AIE-024` … `S-AIE-027`. Rendering a ten-thousand-step position costs 0.01s and does not recurse.

---

### T-AIE-11 — Item 3 `*(pin)*`: constructing a failure retains no reference to the content

- [x] **PIN** — green from birth, exempt from red-first per doc 0002's leaf anatomy, and fully mechanical. `TestViolation_SentinelBody_IsNeverRetainedInTheMessage` enumerates the message's dynamic channels **structurally** — a field name, an element name, a deeper segment, a name past the length bound, a rule wrapping a registered class whose own text carries the body, and a rule that is not a registered class — renders each four ways (`Error`, the position's own rendering, `fmt.Sprint`, `%v` through the `error` interface), and asserts that **no prefix** of the body appears. It then asserts the body is absent from the position *as data*, not merely from the message.

- [x] **Shown to bite.** Two scratch refactors, each reverted immediately:

| Scratch violation | Result |
| --- | --- |
| `ruleText` returns `rule.Error()` — "render what we were given, it is more informative" | `rendering "model: CACHICAMA-SENTINEL-BODY-9f3a" retains "CACHICAMA-SENTINEL-BODY-9f3a"` — 2 subtests fail |
| `structuralName` truncates an over-long name to the bound instead of replacing it | `rendering "CACHICAMA-SENTINEL-BODY-9f3aCACH: required value is empty" retains "CACHICAMA-SENTINEL-BODY-9f3a"` — 1 subtest fails |

The second bite is why the pin was extended with a *name past the length bound* channel: without it, the truncation refactor was caught only by T-AIE-8's table and the pin was weaker than its own claim.

**Why it is not a duplicate of T-AIE-8:** T-AIE-8 covers the channels as they are used today. The pin covers them structurally, through every rendering path, and asserts absence of every prefix — so the refactor it exists to catch fails here even when introduced through a route T-AIE-8's table does not list.

**Proves:** `R-AIE-006`, `S-AIE-015`, `S-AIE-016`.

---

### T-AIE-12 `*(guard)*` — the rule-class registry is enforced, not mirrored

> **Amended 2026-07-31** — this leaf was appended after AI-08's `ErrDuplicate` append exposed a hole in the way this milestone landed the closed set. `ruleClasses` in `validation.go` is unexported on purpose, so `validation_test.go` carried a hand-written copy of it, and its comment argued that being hand-written *was the point* — the append rule read a second time. It was not. **Nothing compared the two.** A class appended to the package registry stayed outside the "a failure matches exactly one rule class" pin until somebody remembered to edit the copy, and a sentinel declared in `validation.go` but never added to `ruleClasses` was caught by nothing at all — it renders as `"value violates an unregistered rule"` and no test of this package covers it. doc 0002 defines a guard leaf as a mechanical check that must keep failing forever when violated, closed only by showing it bite; a mirror that cannot fail was never that, and AI-10 and AI-12 were about to copy the shape.

- [x] **GUARD** — `backend/agent/src/ai/validation_registry_internal_test.go`, package `ai` and not `ai_test`, so it reads the unexported registry directly and nothing is exported to satisfy a test. It scans source with `go/ast`, `go/parser` and `go/token` — standard library, in the idiom of AI-00's `import_boundary_test.go` and AI-13's vocabulary pin — because a list obtained from the thing under test agrees with itself no matter what is added to it. Two directions:

  1. `TestRuleClasses_EverySentinelDeclaredInTheSource_IsInTheRegistry` collects every top-level `var X = errors.New("…")` of `validation.go` and requires each to appear in the `ruleClasses` composite literal, and each registry element to be one of those sentinels. It then binds what it read to the running slice by comparing `ruleClasses[i].Error()` against the string literal the sentinel at position `i` was declared with — without that step the guard would prove two lists in one file agree and say nothing about the registry `ruleText` iterates.
  2. `TestRuleClasses_TheExternalTestMirror_MatchesTheRegistryExactly` scans the elements of `validation_test.go`'s mirror and requires the same members in the same order. The order is asserted and not only the membership: `ruleClasses` is a slice and not a map precisely so no unordered iteration decides anything, and a mirror free to reorder would be a second, disagreeing answer to which class comes first.

- [x] **Shown to bite**, once per direction. Both scratch violations were reverted immediately, and the output below is verbatim from `go test -race -run TestRuleClasses ./src/ai/`.

**Bite 1 — a sentinel declared in `validation.go`'s rule-class block and not appended to `ruleClasses`:**

```
--- FAIL: TestRuleClasses_EverySentinelDeclaredInTheSource_IsInTheRegistry (0.00s)
    validation_registry_internal_test.go:116: validation.go declares the sentinel ErrScratch, which ruleClasses does not list.
          rule: the rule-class set is closed, and a class is appended in the pull request that needs it (design.md § 3.1) — declaring the sentinel is half of that append.
          A sentinel outside ruleClasses renders as "value violates an unregistered rule", and no test of this package covers it.
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.323s
```

**Bite 2 — one element removed from `validation_test.go`'s mirror, which is exactly the drift AI-08 could have caused:**

```
--- FAIL: TestRuleClasses_TheExternalTestMirror_MatchesTheRegistryExactly (0.01s)
    validation_registry_internal_test.go:174: validation.go registers the rule class ErrDuplicate, which the mirror in validation_test.go omits.
          rule: the mirror is what the "a failure matches exactly one rule class" pin iterates, so a class missing from it is a class that pin does not cover.
          Append it to ruleClasses in validation_test.go, at the same position it holds here.
    validation_registry_internal_test.go:192: the mirror is not the registry.
          validation.go: ErrEmpty, ErrNotInVocabulary, ErrOutOfRange, ErrMalformed, ErrUnresolvedReference, ErrDuplicate
          validation_test.go: ErrEmpty, ErrNotInVocabulary, ErrOutOfRange, ErrMalformed, ErrUnresolvedReference
          rule: same members, same order. ruleClasses is ordered on purpose, and a mirror free to reorder it is a second answer to which class comes first.
```

- [x] **No drift was already present.** The guard was green on the registry as AI-08 left it, so it records a closed hole rather than an open defect.

- [x] **The misleading comment is gone.** The doc comment above the mirror in `validation_test.go` claimed the hand-written copy was deliberate; it now points at the guard that enforces it and says to append in the same commit.

**Vacuity, closed three ways** — a guard that reads two files by name must not confuse *"the thing I was pointed at is gone"* with *"it agrees"*. A `validation.go` declaring no sentinel fails, a `ruleClasses` with no elements fails, and a file declaring no top-level `ruleClasses` at all fails with a message naming the rename.

**Proves:** `R-AIE-003`, `S-AIE-007`, `S-AIE-008`, and the new `S-AIE-035`.

---

## Verification pass (closes the milestone)

Ordered by cost of a missed defect. The first three run; the rest are inspection.

- [x] **V-1** — `make test` in `backend/agent/` green under `-race`, both packages, 45 test cases. `ok github.com/cachicamas/backend/agent/src/agenttest` · `ok github.com/cachicamas/backend/agent/src/ai 1.338s`.
- [x] **V-2** — Both AI-00 import guards pass (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires`), and `go.mod` still carries zero `require` directives (`NFR-AIE-A`). The package imports `errors`, `slices`, `strconv` and `strings` — standard library only.
- [x] **V-3** — `make lint` (`go vet ./...` then `golangci-lint run` with govet, errcheck, staticcheck, unused and revive): **0 issues**.
- [x] **V-4** — Every test function follows `Test<Subject>_<Behavior>_<Expectation>` and carries a banner citing its leaf ID (`NFR-AIE-C`).
- [x] **V-5** — No exported declaration names a role, message, content part, tool, request or option. The exported surface is 5 sentinels, `Step`, `At`, `AtIndex`, `Name`, `Index`, `String`, `Path`, `Violation`, `Invalid`, `Error`, `Unwrap`, `Path`, `Rule`, `FirstFailure` — nothing bound to a type AI-05 … AI-13 has not defined (`R-AIE-011`, `S-AIE-030`).
- [x] **V-6** — No exported declaration names a transport, status, category, retry or stream concept (`R-AIE-010`, `S-AIE-028`).
- [x] **V-7** — Each of the five sentinels carries its citable case in its own GoDoc: register § 6.3 for `ErrEmpty`, `ErrMalformed` and `ErrUnresolvedReference`, `V-REQ-01` for `ErrNotInVocabulary`, `V-REQ-24` for `ErrOutOfRange` (`S-AIE-007`).
- [x] **V-8** — Nothing in the package decides which rule fires except a slice index. The package contains no map at all; `ruleClasses` is a slice, and its GoDoc says why (`S-AIE-023`).
- [x] **V-9** — The register diff is 10 lines: two appended rows, one dated blockquote, two count figures and one identifier range. No existing row touched (`S-AIE-033`).
- [x] **V-10** — The diff touches only `openspec/` and `backend/agent/src/ai/`. Nothing under `src/agenttest/`, no build, module or infrastructure file, no other backend module.

---

## Review focus

For the reviewer, in priority order — the first three are where a defect is expensive.

1. **The redaction claim.** Read `validation.go` looking for a path by which a caller-supplied string could reach `Error()`. There are exactly two dynamic inputs — the structural name and the rule — and both are neutralised at a single point each (`structuralName`, `ruleText`). If a third exists, the pin is decorative. Note the honest limit stated in `structuralName`'s own comment: the filter is a backstop, and the contract is that positions are built from names the package wrote.
2. **The boundary applied, not asserted.** In `decision.md` § 2.3, check case D against register § 6.3's actual words. "Decidable without I/O" alone gives the wrong answer; the case turns on "from the request alone". If the artifact reaches the right answer without that clause, it reached it by intuition.
3. **A restated default.** Both decision 2 and decision 3 take doc 0002's recommendation. Read § 3.1 and § 4.1 first and ask whether the opposing case is stated well enough that a reader could adopt it. § 4.1's concession that doc 0002's own allocation argument is the weakest of the three is the marker of a real weighing rather than a rubber stamp.
4. **Speculative surface.** Any exported name mentioning a Layer 1 domain noun is design imposed on AI-05 … AI-13 from outside their milestone.
5. **The typed-nil trap.** `FirstFailure` returns `error`, not `*Violation`. If that is ever "simplified", every `err != nil` in nine milestones becomes true. The subtest *no violated rule yields a genuinely absent failure* is the only thing standing in front of it.
6. **Determinism proven by repetition, not inspection.** A test that read the source and asserted no map would not be the test doc 0002 asked for; the one written runs the same check 1,000 times sequentially and 64 times concurrently under `-race`.
7. **Register discipline.** Two rows appended to a live register. Check that nothing else in it moved and that this change defines neither term locally.

---

## Acceptance criteria for the milestone

1. All four closing-checklist items (T-AIE-1 … T-AIE-4) are answered in `decision.md`, each with the rejected alternative argued at full strength first. **Met.**
2. The register amendment (T-AIE-5) lands in the same pull request, append-only. **Met.**
3. Every test-list item of AI-04.2 and AI-04.3 closed with recorded red and green output, in order. **Met** — seven red runs across six items, one of which needed two.
4. The verification pass V-1 … V-10 recorded complete. **Met.**
5. `spec.md`'s `R-AIE-001` … `R-AIE-012` hold. **Met.**
6. **doc 0002's own acceptance criterion:** every construction and validation rule in AI-05 … AI-13 reports through this taxonomy; failures are matchable through at least one layer of wrapping; no error message carries a content body. **Met**, and the first two are the subject of T-AIE-6 and T-AIE-8 respectively.

## Next

- **AI-05** — `cachicamas-ai-message-roles`: the closed role vocabulary, message identity, ordered content and copy-on-construct semantics. It is the first consumer, and its charter's "fails with an AI-04 sentinel" (AI-05.1 item 2, AI-05.2 item 3) is satisfiable from what this milestone lands: `ErrNotInVocabulary` at `At("role")` and `ErrEmpty` at `At("content")`, composed through `FirstFailure`.
- Then AI-06, the keystone of wave 1, where the class set meets its first real test: whether a construction bypass needs a class of its own, and therefore whether the append rule is exercised for the first time.
