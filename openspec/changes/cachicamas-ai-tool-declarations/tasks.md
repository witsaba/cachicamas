# Tasks — tool declarations

> **Change**: `cachicamas-ai-tool-declarations`
> **Milestone**: AI-08 · **Nodes**: AI-08.1 `[leaf]`, AI-08.2 `[leaf]`, AI-08.3 `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-tool-declarations/spec.md`, `design.md`
> **Branch**: `feat/ai-08-tool-declarations`
> **Depends on**: AI-04, AI-05
> **Blocks**: AI-10, AI-18, AI-26
> **Evidence gate**: recorded green `make test` in `backend/agent/` (`go test -race -v ./...`)

---

## Node types and what they mean for this list

| Node | Type | Closes on |
| --- | --- | --- |
| AI-08.1 | `[leaf]` | Three test-list items taken red → green → refactored, **in order** |
| AI-08.2 | `[leaf]` | Three test-list items, plus one appended during the change |
| AI-08.3 | `[leaf]` | Three test-list items, plus a `*(pin)*` **shown to bite**, plus one appended pin |

AI-08 has **no `[decision]` node**, so there is no `decision.md`. Every choice the milestone makes is made in `design.md` and is answerable to a scenario in `spec.md`.

Strict TDD is on. Go's compile-time typing means a red step needs the declaration to exist, so every red step below follows `design.md` § 9: land the narrowest stub that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

**A note on the recorded output.** Every block below is verbatim `go test -race`, trimmed to the failing assertions where a block ran long. Line numbers are those of the run that produced the block; each test file grew as later items were appended, so a line cited in an early block sits earlier than the same assertion does in the merged file. The assertion text is what identifies it.

---

## Review Workload Forecast

Recorded after the fact, with the forecast that preceded it, because the difference is the point.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `specs/ai-tool-declarations/spec.md`, `design.md`, `tasks.md` | ~700 prose | **953 prose + this file** | Low | 35 min |
| AI-08.1 | `src/ai/tool.go`, `src/ai/tool_test.go` | ~320 Go | **553 Go (60 non-comment prod)** | **Medium** — byte fidelity is inherited by AI-26.4 and cannot be added later | 30 min |
| AI-08.2 | `src/ai/tool_set.go`, `src/ai/tool_set_test.go` | ~295 Go | **509 Go (28 non-comment prod)** | **Medium** — an order defect here is invisible except on a bill | 25 min |
| AI-08.3 | `src/ai/tool_choice.go`, `src/ai/tool_choice_test.go` | ~415 Go | **928 Go (126 non-comment prod)** | Medium — the pattern extension is inherited by AI-13 | 35 min |
| **Total** | 11 files (5 new markdown, 6 new Go) | ~1,030 Go | **1,990 Go (214 non-comment prod), 953+ prose** | **Medium** | **~125 min** |

### Budget reassessment — trigger 4 fired, and this is the record of it

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when "its projected diff, tests included, pushes the milestone past the review budget". **It did**, and it was forecast to in `explore.md` § 7 rather than discovered afterwards. The reassessment, rather than a silent overrun:

- **Production code is 214 non-comment lines** across three files — a three-field value with four rules, an ordered collection with two rules, and a four-member closed vocabulary with a three-rule cross-validator. The surface itself is well inside the budget. What is large is the 1,388 lines of tests and the ~600 lines of contract documentation on the declarations.
- **The ratio is the intended one for this milestone**, and for a reason narrower than "tests are good". Two of AI-08's three properties are *silent* when broken. Schema-byte fidelity fails as a cache miss, not as an error. Iteration order fails as a bill, not as a crash. Neither can be proven by one assertion: fidelity needs a fixture designed to survive nothing, plus mutation from both sides; determinism needs a set large enough for map randomization to show, read many times, compared against the caller's order rather than against itself. Those are the two blocks that dominate the count, and neither is assertable in fewer cases than are written.
- **No other split trigger fires.** The three leaves are one publicly observable behavior (what a tool declaration is on the wire); they are strictly ordered, so two agents could not work them concurrently; each test list is inside the 7-item limit even after the appends; and — the one that mattered — none needed a seam that did not exist.
- **The chain boundary is already cut.** Trigger 4 says "the node boundary is the PR-chain boundary", and the commits fall exactly there: one per leaf, plus the planning commit and this record. If the reviewer wants three PRs, no rework is required.
- **The argument against actually splitting.** Landing AI-08.1 and AI-08.2 without AI-08.3 ships a tool set nothing validates a choice against, which is a working type whose only consumer is a milestone that does not exist. Landing AI-08.1 alone ships a declaration with no collection. The recommendation is one PR reviewed in three passes, in commit order.

---

## Phase AI-08.1 — A declaration is constructible and readable `[leaf]`

Three test-list items in doc 0002's order, plus one discovered case appended to the third.

### T-ATD-1 — Item 1: name, description and schema bytes read back from an external package

- [x] **RED** — `TestTool_NameDescriptionAndSchema_ReadBackExactlyFromAnExternalPackage` against a `Tool` with no fields, a `NewTool` returning the zero value, and three accessors returning `""` / `nil`.

```
--- FAIL: TestTool_NameDescriptionAndSchema_ReadBackExactlyFromAnExternalPackage (0.00s)
    --- FAIL: .../an_empty_description_is_legal (0.00s)
        tool_test.go:65: tool.Name() = "", want "list_files"
        tool_test.go:71: tool.Schema() = "", want "{\"type\":\"object\"}"
    --- FAIL: .../a_name_exactly_at_the_documented_ceiling (0.00s)
        tool_test.go:65: tool.Name() = "", want "abbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        tool_test.go:68: tool.Description() = "", want "Exactly at the ceiling."
    --- FAIL: .../a_fully_populated_declaration (0.00s)
        tool_test.go:65: tool.Name() = "", want "get_weather"
        tool_test.go:68: tool.Description() = "", want "Get the current weather for a location."
FAIL	github.com/cachicamas/backend/agent/src/ai	0.449s
```

- [x] **GREEN** — `Tool` stores the three values; `NewTool` sets them; the accessors return them. All three subtests `--- PASS`.
- [x] **REFACTOR** — one, and it is worth recording as a **discipline correction**. The first green implementation used `slices.Clone` on both the constructor and the reader. That is the *right* end state but it was not what item 1 demanded, and it would have made item 2's red impossible to take honestly. The clones were removed, item 1 re-run green against the aliasing implementation, and the clones re-introduced by item 2 — which is exactly what `design.md` § 9 predicted for this item and what AI-05's `design.md` § 8 established as the convention for copy semantics.

**Proves:** `R-ATD-001`, `S-ATD-001`, `S-ATD-002`.

**The name at the ceiling is a case, not decoration.** It is built as `"a" + strings.Repeat("b", ai.MaxToolNameLen-1)` rather than written out, so the boundary case cannot drift away from the constant it is testing — and it is the reason `MaxToolNameLen` is exported at all.

---

### T-ATD-2 — Item 2: schema bytes pass through byte-identically

- [x] **RED** — `TestTool_SchemaBytes_PassThroughByteIdentically` against the aliasing implementation left by item 1's refactor. The recorded failure is the aliasing symptom itself, not a missing return.

```
--- FAIL: TestTool_SchemaBytes_PassThroughByteIdentically (0.00s)
    --- FAIL: .../the_caller_may_mutate_the_slice_it_passed (0.00s)
        tool_test.go:126: tool.Schema() = "ZZZZ…ZZZ" after the caller rewrote its own slice,
            want "{\"type\":\"object\", \"properties\":{\"zulu\":{\"type\":\"string\"},\n\t\"alpha\":…}"
    --- FAIL: .../a_consumer_may_mutate_what_a_read_returned (0.00s)
        tool_test.go:144: tool.Schema() = "ZZZZ…ZZZ" after a consumer rewrote what it read,
            want "{\"type\":\"object\", \"properties\":{\"zulu\":{\"type\":\"string\"},\n\t\"alpha\":…}"
        tool_test.go:150: two reads share a backing array; one consumer can rewrite another's view
FAIL	github.com/cachicamas/backend/agent/src/ai	0.287s
```

- [x] **GREEN** — `slices.Clone` on the way in and on the way out. All five subtests `--- PASS`.
- [x] **REFACTOR** — none needed. The GoDoc on `Schema` was written here rather than later, because the list of things it must *not* do is the contract.

**Proves:** `R-ATD-002`, `R-ATD-004`, `S-ATD-004` … `S-ATD-006`, `S-ATD-014`, `S-ATD-015`.

**The fixture is the test.** Canonical JSON would have made this pass against exactly the implementation it exists to reject, because a marshal round trip is idempotent *after the first pass*. The fixture therefore has non-alphabetical object keys (`zulu` before `alpha`), runs of interior spaces, a tab and a newline — so a normalizing implementation differs on read one.

Two subtests came free with the fixture and pin `R-ATD-004`: bytes that no meta-schema would accept (`not json at all`, `{`, raw `0x00 0xff 0xfe`) construct successfully and read back unchanged, and two schemas differing only in whitespace stay distinct.

---

### T-ATD-3 — Item 3: construction rules fail through AI-04 sentinels

*Includes the discovered case appended under the living-graph clause — see below.*

- [x] **RED** — `TestNewTool_BrokenConstructionRules_FailWithTheDocumentedSentinels` against the item-2 implementation, which validates nothing.

```
--- FAIL: TestNewTool_BrokenConstructionRules_FailWithTheDocumentedSentinels (0.00s)
    --- FAIL: .../an_empty_name (0.00s)
        tool_test.go:300: NewTool("", ...) returned no failure, want one
    --- FAIL: .../a_name_one_byte_past_the_ceiling (0.00s)
        tool_test.go:300: NewTool("abbb…bb", ...) returned no failure, want one
    --- FAIL: .../a_name_containing_a_dot (0.00s)
        tool_test.go:300: NewTool("github.list_prs", ...) returned no failure, want one
    --- FAIL: .../a_name_containing_a_space (0.00s)
        tool_test.go:300: NewTool("get weather", ...) returned no failure, want one
    --- FAIL: .../a_name_that_is_not_ASCII (0.00s)
        tool_test.go:300: NewTool("obtener_clima_ñ", ...) returned no failure, want one
    --- FAIL: .../a_name_beginning_with_a_digit (0.00s)
        tool_test.go:300: NewTool("1st_tool", ...) returned no failure, want one
    --- FAIL: .../a_name_beginning_with_a_hyphen (0.00s)
        tool_test.go:300: NewTool("-tool", ...) returned no failure, want one
    --- FAIL: .../empty_schema_bytes (0.00s)
        tool_test.go:300: NewTool("get_weather", ...) returned no failure, want one
    --- FAIL: .../the_first_failure_in_the_documented_order_wins,_on_every_run (0.00s)
        tool_test.go:338: errors.As(err, *ai.Violation) = false, want true (err = <nil>)
    --- FAIL: .../no_offered_value_reaches_the_failure (0.00s)
        tool_test.go:363: NewTool("CACHICAMA-SENTINEL-NAME-9f2b!", ...) returned no failure, want one
FAIL	github.com/cachicamas/backend/agent/src/ai	0.499s
```

- [x] **GREEN** — `toolNameRules` lands as an ordered `[]Rule`, composed with the schema rule through `FirstFailure`. All ten subtests `--- PASS`.
- [x] **REFACTOR** — `toolNameRules` was extracted as a function returning rules rather than a boolean predicate, because AI-08.3 needs the *same* classes at the *same* position in the *same* order. One name rule in the package, not two that could drift.

**Proves:** `R-ATD-003`, `S-ATD-007` … `S-ATD-013`.

#### The discovered case, and the clause it lands under

Doc 0002's AI-08.1 item 3 names three rules: "empty name, a name outside the documented character rules, empty schema bytes". Implementing it produced a fourth: **a name longer than the ceiling**.

Under doc 0002's living-graph clause this is a discovered **test case**, not a discovered prerequisite — clause 3's last sentence: "Newly discovered *test cases* (not prerequisites) are appended to the owning leaf's test list instead." **No doc 0002 amendment is required, and none was made.** It is recorded here because the reason it is a separate case is a design decision:

> Length and shape report **different rule classes** — `ErrOutOfRange` and `ErrMalformed`. `V-FAIL-17` makes a rule class "the kind of thing a rule checks, independent of what it checks it on", and length and shape are different kinds with different fixes: "shorten it" versus "spell it differently". Folding them into one class would tell a caller with a 200-character name to change its spelling, and a caller with `github.list_prs` to shorten it. Both messages would be wrong.

The documented order is **empty name → over-long name → malformed name → empty schema**, and the ordering is asserted rather than assumed: a name that is both over-long and malformed (65 dots) reports `ErrOutOfRange` and explicitly does **not** match `ErrMalformed`.

#### The name rule, and which half is verified

`design.md` § 3.2 carries the argument; the evidence trail is recorded here because it is the milestone's one externally-sourced claim.

- **Verified from a primary source, on 2026-07-31 for this change:** Anthropic's tool-definition reference states a tool `name` **"Must match the regex `^[a-zA-Z0-9_-]{1,64}$`"**. That is the rule's length ceiling and its alphabet, exactly.
- **A deliberate narrowing, not inherited:** the leading-character restriction (first byte an ASCII letter or `_`). It is added because at least one major vendor additionally requires a function name to begin with a letter or underscore, and because loosening later is additive while tightening is breaking. A reviewer should read it as a choice this change made.

---

## Phase AI-08.2 — Tool-set rules `[leaf]`

Three test-list items in doc 0002's order, plus one appended.

### T-ATD-4 — Item 1: duplicate names are rejected

- [x] **RED** — `TestNewToolSet_DuplicateNames_FailWithASentinelAtTheSecondOccurrence` against a `ToolSet` with no fields and a `NewToolSet` returning the zero value.

```
--- FAIL: TestNewToolSet_DuplicateNames_FailWithASentinelAtTheSecondOccurrence (0.00s)
    --- FAIL: .../an_adjacent_pair (0.00s)
        tool_set_test.go:66: NewToolSet([read read]) returned no failure, want one
    --- FAIL: .../a_separated_pair (0.00s)
        tool_set_test.go:66: NewToolSet([read write edit read]) returned no failure, want one
    --- FAIL: .../the_first_of_several_duplicate_pairs (0.00s)
        tool_set_test.go:66: NewToolSet([a b b a]) returned no failure, want one
    --- FAIL: .../three_of_the_same_name (0.00s)
        tool_set_test.go:66: NewToolSet([x x x]) returned no failure, want one
    --- FAIL: .../the_same_occurrence_is_reported_on_every_run (0.00s)
        tool_set_test.go:95: errors.As(err, *ai.Violation) = false, want true (err = <nil>)
    --- FAIL: .../the_offered_name_never_reaches_the_failure (0.00s)
        tool_set_test.go:119: NewToolSet returned no failure, want one
FAIL	github.com/cachicamas/backend/agent/src/ai	0.305s
```

- [x] **GREEN** — a linear scan over the ordered sequence, reporting `Invalid(ErrDuplicate, AtIndex("tools", i))` at the second occurrence. All six subtests `--- PASS`. *(Shipped as `ErrMalformed`; corrected to `ErrDuplicate` on 2026-07-31 — the second red/green cycle is recorded below.)*
- [x] **REFACTOR** — none. The scan is deliberately not a map lookup; see below.

**Proves:** `R-ATD-007`, `S-ATD-023` … `S-ATD-025`, and — from the correction below — `R-AIE-003`/`S-AIE-034` of the AI-04 delta spec.

> **Amended 2026-07-31 — the sentinel choice was wrong, and correcting it exercised AI-04's append rule.**
>
> **What was recorded here originally:** that a duplicate name is cleanly none of AI-04's five classes (true), and that a sixth sentinel was therefore unavailable because "AI-04's decision record is explicit that the set grows by appending a class in the pull request that needs it, and this change does not need one" (an inversion of that rule). `decision.md` § 3.5 says a milestone appends a class **in the pull request that needs it** precisely so that no milestone defines a local sentinel or stretches a neighbouring class. AI-08 met the "two values conflict" case AI-04 explicitly forecast, so AI-08 *was* the pull request that needed it.
>
> **Why the wrong class is a contract defect and not a cosmetic one:** a duplicate tool name is not malformed. Each name in the set is perfectly well-formed — `NewTool` already accepted every one of them — and what fails is uniqueness *across* the set, which is not a property any single value has. An adapter author reading `ErrMalformed` goes looking for the badly spelled name and finds none, and `errors.Is` cannot separate *spell it differently* from *you have two of these*, which are the two different fixes a consumer must make.
>
> **The correction, in two commits.** First, `ErrDuplicate` was appended to `validation.go`'s rule-class registry and to the hand-written mirror in `validation_test.go`, with no behaviour change and the whole suite green. Then the behaviour change, red first:
>
> ```
> --- FAIL: TestNewToolSet_DuplicateNames_FailWithASentinelAtTheSecondOccurrence (0.00s)
>     --- FAIL: .../three_of_the_same_name (0.00s)
>         tool_set_test.go:73: errors.Is(err, ErrDuplicate) = false, want true (err = tools[1]: value is not well-formed for its documented encoding)
>         tool_set_test.go:76: errors.Is(err, ErrMalformed) = true, want false — a duplicate name is well-formed (err = tools[1]: value is not well-formed for its documented encoding)
>     --- FAIL: .../an_adjacent_pair (0.00s)
>         tool_set_test.go:73: errors.Is(err, ErrDuplicate) = false, want true (err = tools[1]: value is not well-formed for its documented encoding)
>         tool_set_test.go:76: errors.Is(err, ErrMalformed) = true, want false — a duplicate name is well-formed (err = tools[1]: value is not well-formed for its documented encoding)
>     --- FAIL: .../the_first_of_several_duplicate_pairs (0.00s)
>         tool_set_test.go:73: errors.Is(err, ErrDuplicate) = false, want true (err = tools[2]: value is not well-formed for its documented encoding)
>         tool_set_test.go:76: errors.Is(err, ErrMalformed) = true, want false — a duplicate name is well-formed (err = tools[2]: value is not well-formed for its documented encoding)
>     --- FAIL: .../the_offered_name_never_reaches_the_failure (0.00s)
>         tool_set_test.go:129: Error() = "tools[1]: value is not well-formed for its documented encoding", want it composed only of the position and the rule class
>     --- FAIL: .../a_separated_pair (0.00s)
>         tool_set_test.go:73: errors.Is(err, ErrDuplicate) = false, want true (err = tools[3]: value is not well-formed for its documented encoding)
>         tool_set_test.go:76: errors.Is(err, ErrMalformed) = true, want false — a duplicate name is well-formed (err = tools[3]: value is not well-formed for its documented encoding)
> FAIL	github.com/cachicamas/backend/agent/src/ai	0.467s
> ```
>
> Green followed from one changed argument in `NewToolSet`. The assertion is now two-sided — it requires `ErrDuplicate` **and** requires the failure not to match `ErrMalformed` — so the two readings can never be silently merged again. T-ATD-9's ordering subtest was updated in the same cycle: "emptiness is reported before duplication" now asserts the absence of `ErrDuplicate`, which is the class that subtest was always about.
>
> **What the cost of the mistake actually was:** one line at one call site, plus its assertions. That is the append rule's whole argument. Had AI-08 defined a local `errDuplicateToolName` in `tool_set.go` instead — the alternative § 3.5 forbids — the same correction would have been a change to every consumer that had learned to match it.

**The sentinel choice was a real finding, and the finding was recorded correctly while the conclusion drawn from it was not.** A duplicate name is cleanly **none** of AI-04's five landed classes. `ErrOutOfRange` is wrong (not a bound); `ErrNotInVocabulary` is wrong (a tool set is not a closed vocabulary); `ErrMalformed` is wrong (every name in the set is well-formed). The right move on a violation no landed class describes is to append one, which is what `decision.md` § 3.5 asks for and what this milestone now does. The reasoning is in `NewToolSet`'s own GoDoc, because a reader who hits the failure will look at the failure, not in a change directory.

**The linear scan buys determinism, not just simplicity.** A map-based duplicate check would be `O(n)`, over a collection whose realistic size makes that irrelevant, and it would put an unordered structure in the one place AI-04 warns about. The ordered scan guarantees the *reported* duplicate is always the lowest index whose name repeats an earlier one — asserted over 64 repeated runs.

---

### T-ATD-5 — Item 2: an empty tool set is legal

- [x] **RED** — `TestNewToolSet_EmptySet_IsLegal` against `Tools()` returning `nil`, `Len()` returning `0` and `Declares` returning `false`.

```
--- FAIL: TestNewToolSet_EmptySet_IsLegal (0.00s)
    --- FAIL: .../a_non-empty_set_reports_what_it_holds (0.00s)
        tool_set_test.go:193: set.Len() = 0, want 2
        tool_set_test.go:196: set.Declares reported false for a name the set holds
FAIL	github.com/cachicamas/backend/agent/src/ai	0.274s
```

- [x] **GREEN** — the set stores the sequence; `Len` and `Declares` read it. All three subtests `--- PASS`.
- [x] **REFACTOR** — none.

**Proves:** `R-ATD-006`, `S-ATD-021`, `S-ATD-022`.

**The red is narrow on purpose, and it is the honest one.** The stubs already satisfied every *empty* assertion, because an empty set and an unimplemented set are indistinguishable from the outside. The only assertion that could fail was the non-empty control — which is precisely why the control is in this test rather than in a later one. A test that only asserted emptiness would have passed against a type that stores nothing at all.

**The zero value is a member of the legal domain here**, unlike every closed vocabulary in this package. `ToolSet{}` reports length zero, yields an empty sequence and declares nothing — correct rather than convenient, because `R-ATD-006` makes the empty set legal.

---

### T-ATD-6 — Item 3: iteration yields the caller's order, every time

- [x] **RED** — `TestToolSet_Iteration_YieldsTheCallersOrderEveryTime` against the item-2 implementation, which aliases in both directions. The race detector reported the aliasing directly, which is a stronger red than the assertion alone.

```
WARNING: DATA RACE
Write at 0x00c00024a008 by goroutine 12:
  slices.Reverse[...ai.Tool...]()
  ai_test.TestToolSet_Iteration_YieldsTheCallersOrderEveryTime.func6()
      tool_set_test.go:314
Previous read at 0x00c00024a008 by goroutine 9:
  ai.NewToolSet()
      tool_set.go:56
  ai_test.TestToolSet_Iteration_YieldsTheCallersOrderEveryTime.func3()
      tool_set_test.go:265
[8 × WARNING: DATA RACE]

--- FAIL: TestToolSet_Iteration_YieldsTheCallersOrderEveryTime (0.00s)
    --- FAIL: .../every_read_yields_the_caller's_order (0.00s)
        tool_set_test.go:257: read 0 yielded [tool_-51_63 tool_51_1 …], want the caller's order [tool_0_0 tool_51_1 …]
    --- FAIL: .../two_sets_built_the_same_way_are_element-for-element_equal (0.00s)
    --- FAIL: .../concurrent_readers_each_observe_the_caller's_order (0.00s)
        tool_set_test.go:284: a concurrent reader observed an order other than the caller's  [× 5]
    --- FAIL: .../the_caller_may_mutate_what_it_passed (0.00s)
        tool_set_test.go:306: the set changed after the caller rewrote its own slice: [intruder tool_-38_62 …]
FAIL	github.com/cachicamas/backend/agent/src/ai	0.290s
```

- [x] **GREEN** — `slices.Clone` on the way in and on the way out. All five subtests `--- PASS`, no races.
- [x] **REFACTOR** — none.

**Proves:** `R-ATD-005`, `S-ATD-016` … `S-ATD-020`.

**The assertion is against the caller's order, not against a previous read**, and that is the whole design of the test. A stable-but-arbitrary order — which a map-backed implementation with a sorted key list would give — satisfies self-comparison and still breaks the property `V-REQ-25` needs: a caller who builds the same set twice must obtain the same cache prefix. The set is 64 elements read 100 times, plus 8 concurrent readers under `-race`, so map randomization shows on read one rather than eventually.

**The names are generated so that lexical, insertion and hash orders all differ** (`tool_<(448-13i) mod 64>_<i>`), which is why several carry a hyphen — incidentally exercising the name rule's non-leading hyphen.

---

### T-ATD-7 — Item 4 *(appended)*: an unconstructed declaration is rejected

Discovered while implementing item 1 and appended to AI-08.2's list under the living-graph clause. **No doc 0002 amendment required** — a discovered case, not a discovered prerequisite.

- [x] **RED** — `TestNewToolSet_AnUnconstructedDeclaration_IsRejected`.

```
--- FAIL: TestNewToolSet_AnUnconstructedDeclaration_IsRejected (0.00s)
    --- FAIL: .../the_zero_declaration_alone (0.00s)
        tool_set_test.go:358: NewToolSet returned no failure, want one — an unconstructed declaration must not reach the wire
    --- FAIL: .../a_zero_declaration_after_a_valid_one (0.00s)
        tool_set_test.go:358: NewToolSet returned no failure, want one — …
    --- FAIL: .../a_zero_declaration_before_a_valid_one (0.00s)
        tool_set_test.go:358: NewToolSet returned no failure, want one — …
    --- FAIL: .../emptiness_is_reported_before_duplication (0.00s)
        tool_set_test.go:386: errors.Is(err, ErrEmpty) = false, want true
            (err = tools[1]: value is not well-formed for its documented encoding)
        tool_set_test.go:389: errors.Is(err, ErrMalformed) = true, want false
        tool_set_test.go:397: violation.Path() = "tools[1]", want "tools[0]"
FAIL	github.com/cachicamas/backend/agent/src/ai	0.297s
```

- [x] **GREEN** — an emptiness check per element, **before** the duplicate scan. All four subtests `--- PASS`.
- [x] **REFACTOR** — none.

> **Amended 2026-07-31** — the transcript above is left verbatim, as recorded. Its `ErrMalformed` lines are the duplicate class *as it was then spelled*; T-ATD-4's amendment replaced it with `ErrDuplicate`, and the `emptiness_is_reported_before_duplication` subtest now asserts the absence of `ErrDuplicate`. What the subtest pins is unchanged — that the unconstructed fact wins over the duplicate fact — and it is a stronger pin under the new class, because "not the duplicate class" now means exactly that instead of doubling as "not the malformed class".

**Why it exists.** A `Tool{}` that skipped `NewTool` has an empty name, which no constructed declaration's is — so emptiness is a sound detector without a separate `constructed` flag. Without the check, a value that skipped its constructor reaches the wire carrying a name no provider accepts. This is AI-06.3 item 1's shape applied one contract over, and the last subtest pins the ordering: two zero declarations are *both* unconstructed and duplicates of one another, and the unconstructed fact — the more fundamental one — is reported first, at the earlier index.

---

## Phase AI-08.3 — Tool choice `[leaf]`

Three test-list items, plus the `*(pin)*`, plus one appended pin.

### T-ATD-8 — Item 1: the vocabulary is closed and each member is constructible

- [x] **RED** — `TestToolChoice_EachVocabularyMember_IsConstructibleAndReadsBack` against `ToolChoiceModes()` returning `nil`, `String()` returning `""`, `ParseToolChoiceMode` returning `(0, nil)`, both constructors returning the zero value, and `Name()` returning `("", false)`.

```
--- FAIL: TestToolChoice_EachVocabularyMember_IsConstructibleAndReadsBack (0.00s)
    --- FAIL: .../each_payload-free_member_constructs_and_reads_back (0.00s)
        tool_choice_test.go:41: choice.Mode() = , want   [× 3]
    --- FAIL: .../the_payload-carrying_member_constructs_and_carries_its_name (0.00s)
        tool_choice_test.go:57: choice.Mode() = , want
        tool_choice_test.go:61: choice.Name() reported no name, want the one supplied
    --- FAIL: .../the_payload-free_constructor_rejects_the_payload-carrying_member (0.00s)
        tool_choice_test.go:76: NewToolChoice(ToolChoiceSpecific) returned no failure, want one
    --- FAIL: .../a_mode_outside_the_vocabulary_is_rejected (0.00s)
        tool_choice_test.go:100: NewToolChoice(0) returned no failure, want one
        tool_choice_test.go:100: NewToolChoice(5) returned no failure, want one
        tool_choice_test.go:100: NewToolChoice(200) returned no failure, want one
        tool_choice_test.go:100: NewToolChoice(255) returned no failure, want one
    --- FAIL: .../the_named_constructor_applies_the_tool-name_rule (0.00s)
        --- FAIL: .../empty (0.00s)
            tool_choice_test.go:132: NewNamedToolChoice("") returned no failure, want one
        --- FAIL: .../containing_a_dot (0.00s)
            tool_choice_test.go:132: NewNamedToolChoice("github.list_prs") returned no failure, want one
    --- FAIL: .../the_vocabulary_enumerates_in_a_stable_order_and_cannot_be_rewritten (0.00s)
        tool_choice_test.go:157: ToolChoiceModes() returned 0 members, want 4
FAIL	github.com/cachicamas/backend/agent/src/ai	0.281s
```

- [x] **GREEN** — the vocabulary, the widened table, the enumeration over the constant space, the rendering, the exact parser, and both constructors. All six subtests `--- PASS`.
- [x] **REFACTOR** — none. `NewNamedToolChoice` calls `toolNameRules` rather than duplicating the checks, which was the point of extracting it in T-ATD-3.

**Proves:** `R-ATD-008`, `R-ATD-009`, `S-ATD-027` … `S-ATD-038`.

**The pattern extension is the milestone's one real design decision**, and `design.md` § 5 is its record. In one sentence: the vocabulary stays a payload-free integer; the payload lives in a separate value type carrying a member plus its payload; the table row widens by one arity column.

AI-05's `design.md` § 3.2 pre-authorised exactly that — "Rule 2's table may carry more than a name … that is a widening of the row, not a change of shape" — so this change spends an allowance the pattern already granted rather than bending it. The two rules AI-05 calls load-bearing are untouched: `iota + 1` still makes a mode nobody set fail like a wild one, and `ToolChoiceModes` still walks the **constant space** rather than the table, which is what lets the pin bite.

`NewToolChoice(ToolChoiceSpecific)` fails with `ErrEmpty` at `name` rather than quietly producing a nameless "specific" choice: the member requires a name and none was supplied, which is exactly what the empty class means.

---

### T-ATD-9 — Item 2: a choice naming an undeclared tool fails

- [x] **RED** — `TestToolChoice_NamingAnUndeclaredTool_FailsWithUnresolvedReference` against a `ValidateAgainst` returning `nil`.

```
--- FAIL: TestToolChoice_NamingAnUndeclaredTool_FailsWithUnresolvedReference (0.00s)
    --- FAIL: .../an_undeclared_tool_fails (0.00s)
        tool_choice_test.go:265: choice naming "delete": ValidateAgainst returned no failure, want one
        tool_choice_test.go:265: choice naming "Read": ValidateAgainst returned no failure, want one
        tool_choice_test.go:265: choice naming "read_file": ValidateAgainst returned no failure, want one
        tool_choice_test.go:265: choice naming "rea": ValidateAgainst returned no failure, want one
    --- FAIL: .../a_choice_that_is_not_a_vocabulary_member_is_rejected (0.00s)
        tool_choice_test.go:303: the zero ToolChoice validated, want a failure — a value nobody set is not a default
    --- FAIL: .../the_offered_name_never_reaches_the_failure (0.00s)
        tool_choice_test.go:329: ValidateAgainst returned no failure, want one
FAIL	github.com/cachicamas/backend/agent/src/ai	0.275s
```

- [x] **GREEN** — `ValidateAgainst` composes its rules through `FirstFailure`. All five subtests `--- PASS`.
- [x] **REFACTOR** — none. **The green over-reached**, and T-ATD-10 records the correction.

**Proves:** `R-ATD-010` in part, `S-ATD-039`, `S-ATD-040`, `S-ATD-043`.

`"Read"` is in the undeclared list on purpose: resolution is exact, so a case variant is as undeclared as a name nobody ever mentioned.

---

### T-ATD-10 — Item 3: only "none" is legal against an empty tool set

This item is the milestone's **second discipline correction**, and it is recorded rather than hidden.

- [x] **GREEN-FROM-BIRTH (not acceptable)** — `TestToolChoice_AgainstAnEmptyToolSet_OnlyNoneIsLegal` passed on its first run, because T-ATD-9's implementation had written all three cross-validation rules at once instead of only the one item 2 demanded. A test that has never failed has not been shown to discriminate.
- [x] **RED (recovered)** — the emptiness rule was removed from `ValidateAgainst`, the test re-run, and the failure recorded. This is the same move item 1 required, and it earns its keep: the recovered red shows the ordering assertion bites, because with rule 2 absent the specific case reports `ErrUnresolvedReference` at `toolChoice.name` instead of `ErrEmpty` at `tools`.

```
--- FAIL: TestToolChoice_AgainstAnEmptyToolSet_OnlyNoneIsLegal (0.00s)
    --- FAIL: .../every_mode_other_than_none_fails (0.00s)
        --- FAIL: .../auto (0.00s)
            tool_choice_test.go:376: auto against an empty tool set validated, want a failure
        --- FAIL: .../required (0.00s)
            tool_choice_test.go:376: required against an empty tool set validated, want a failure
        --- FAIL: .../specific (0.00s)
            tool_choice_test.go:379: errors.Is(err, ErrEmpty) = false, want true
                (err = toolChoice.name: value names something the request does not declare)
            tool_choice_test.go:387: violation.Path() = "toolChoice.name", want "tools"
    --- FAIL: .../the_emptiness_rule_precedes_the_resolution_rule,_on_every_run (0.00s)
        tool_choice_test.go:424: errors.Is(err, ErrEmpty) = false, want true
            (err = toolChoice.name: value names something the request does not declare)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.290s
```

- [x] **GREEN** — the emptiness rule restored to its documented position, second. All four subtests `--- PASS`.
- [x] **REFACTOR** — none.

**Proves:** `R-ATD-010` in full, `S-ATD-041`, `S-ATD-042`, `S-ATD-044`, `S-ATD-045`.

**The ordering decision, and why it is asserted rather than assumed.** Against an empty set, a `specific` choice violates rules 2 and 3 at once. Rule 2 wins: "you have declared no tools at all" names the thing the caller must fix, while "the tool you named is not declared" is true of every possible name and helps nobody. The assertion runs 64 times and checks both that `ErrEmpty` matches and that `ErrUnresolvedReference` explicitly does **not**, so a later reordering is a test failure rather than a silent behavior change.

**Rule 2 is the milestone's most valuable single line.** It is the combination every provider rejects, and catching it costs a comparison instead of a network round trip. `none` against an empty set is asserted legal rather than merely permitted by omission — including against the zero `ToolSet`.

---

### T-ATD-11 — Item 4 `*(pin)*`: registration is exhaustive

Green from birth and exempt from red-first per doc 0002's leaf anatomy.

- [x] **GREEN** — `TestToolChoiceMode_DeclaredVocabulary_IsExhaustivelyTabulated` passes against the landed vocabulary.
- [x] **BITE** — a scratch member `ToolChoiceScratch` was declared and `toolChoiceEnd` moved past it, **without** a `toolChoiceModes` entry. Recorded, then removed.

```
--- FAIL: TestToolChoiceMode_DeclaredVocabulary_IsExhaustivelyTabulated (0.00s)
    tool_choice_test.go:487: ToolChoiceMode(5) has no table entry: String() = "toolChoice(5)"
FAIL	github.com/cachicamas/backend/agent/src/ai	0.284s
```

- [x] **REMOVED** — the scratch member and the moved terminator were reverted; the suite is green.

**Proves:** `R-ATD-011`, `S-ATD-046` … `S-ATD-049`.

**Step 3 is this milestone's own addition to the pin**, and it is what the widened table row buys: it asserts a **biconditional** against the arity column — a member is constructible payload-free *if and only if* its row does not require a name — rather than against a hard-coded list of members. A fifth member added tomorrow is covered by whichever branch its own row selects.

**The honest limit, recorded rather than discovered later.** Step 4 maps "needs a name" onto one specific constructor. With one payload-carrying member that is exact. If a second ever lands, the arity column stops being a boolean and step 4 needs a per-member constructor mapping — a widening of the same row and the same shape. The note is in the test's own banner comment so the milestone that meets the case reads it instead of re-deriving it.

---

### T-ATD-12 — Item 5 *(appended, pin)*: no input causes a panic

Appended during the change, green from birth, mirroring AI-05's own totality pin.

- [x] **GREEN** — `TestToolContract_NoInputCausesAPanic`. Ten probes, all `--- PASS`.

**Why it matters more here than it looks.** Three of this milestone's types have a zero value a caller can obtain with no constructor — `Tool{}`, `ToolSet{}`, `ToolChoice{}` — and AI-10 will hand all three to a request that validates before any I/O. A panic there would escalate a caller-contract failure into a process failure. The probes include every one of the 256 values `ToolChoiceMode`'s underlying type can hold, a 64 KiB name, non-UTF-8 bytes through the parser, and a nil schema.

---

## Verification pass (closes the milestone)

- [x] **`make test` in `backend/agent/` is green with `-race`.** 176 passing assertions across the package.

```
--- PASS: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault (0.04s)
--- PASS: TestLayer1_ModuleHasNoDependencies_ZeroRequires (0.04s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.599s
```

- [x] **`make lint` is clean.** `go vet ./...` then `golangci-lint run --config=.golangci.yml ./...` → `0 issues.`
- [x] **Both AI-00 import guards pass**, unchanged and untouched.
- [x] **`go.mod` carries zero requires.** Verified directly, not only by the guard.
- [x] **`R-ATD-012` inspection** — the exported surface of `tool.go`, `tool_set.go` and `tool_choice.go` exposes no invocation, resolution, permission or provenance operation; declares no error variable and no error type; and defines no content part, message or request type. Every failure it produces matches one of AI-04's five landed classes.
- [x] **File-set disjointness** — `validation.go`, `role.go`, `message.go`, `doc.go`, `import_boundary_test.go` and `src/agenttest/` are untouched, so the concurrent AI-06 and AI-13 milestones cannot conflict with this one.
- [x] **No register amendment** — `explore.md` § 6 records the check and the three near-misses.
- [x] **No doc 0002 amendment** — both discovered items are test *cases* under the living-graph clause's rule 3, appended to their owning leaves and recorded in T-ATD-3 and T-ATD-7.

---

## Review focus

Where a reviewer's attention pays for itself, in order:

1. **`design.md` § 3.2 — the name rule.** It is the milestone's one externally-sourced claim and its one narrowing. The verified half is quoted; the chosen half is labelled. If the leading-character restriction is wrong, say so now — loosening it later is additive, but the argument should be tested while it is cheap.
2. **`tool_test.go`'s byte-fidelity fixture.** Check that it would *not* survive a marshal round trip. If it would, the test is decorative and AI-26.4 inherits nothing.
3. **`tool_set_test.go`'s order assertion.** Check that it compares against the caller's order and not against a previous read. Self-comparison passes for a stable-but-wrong implementation.
4. **`design.md` § 5 and `tool_choice.go`'s file comment — the pattern extension.** AI-13 will copy whichever of these two is clearer. They must agree.
5. **`ValidateAgainst`'s rule order.** Rule 2 before rule 3 is a judgement call with a defensible alternative. The test pins it; the reviewer should decide whether it pinned the right one.
6. **`NewToolSet`'s duplicate class.** ~~A real finding: no landed class fits cleanly.~~ *(Amended 2026-07-31 — review found the conclusion inverted AI-04's append rule. `ErrMalformed` was replaced by `ErrDuplicate`, appended to AI-04's registry in this pull request; see T-ATD-4's amendment, `design.md` § 6.1, and the GoDoc. What a reviewer should still check is whether "uniqueness across a collection" is the right **class**, or whether the class is narrower than that — it is the first class appended after AI-04 and the next collection to need one will inherit its wording.)*
7. **The two discipline corrections** (T-ATD-1's over-implementation, T-ATD-10's green-from-birth). Both are recorded rather than hidden, and both produced better red evidence than the straight path would have.

---

## Acceptance criteria for the milestone

1. `R-ATD-001` … `R-ATD-011` each have at least one passing test; `R-ATD-012` has a recorded inspection. ✅
2. Every test-list item taken red → green → refactored in order, with both outputs recorded; pins exempt from red-first and the exhaustiveness pin shown to bite. ✅
3. `make test` green with `-race`; `make lint` clean; both import guards pass; `go.mod` carries zero requires. ✅
4. Schema-byte equality asserted against input a marshal round trip would rewrite. ✅
5. Order determinism asserted over 64 declarations, 100 reads plus concurrent readers, against the caller's order. ✅
6. The empty tool set is legal; the zero `ToolSet` is that empty set. ✅
7. Every tool-choice member constructible; the payload-carrying one carries its name; cross-validation runs in the documented order. ✅
8. ~~No new sentinel~~, no error type, no register amendment, no doc 0002 amendment. ✅ *(Amended 2026-07-31 — **one** sentinel was appended, `ErrDuplicate`, under `decision.md` § 3.5's append rule and with the citable case that rule requires: AI-08.2 item 1. The criterion as written treated a stable sentinel set as the goal; the goal is a set that grows only by appending, only for a violation no landed class describes, and only in the pull request that meets it. All three held. No error type was added, no register row was needed — `V-FAIL-17` states that which rule classes exist is AI-04's, not the register's — and doc 0002 is untouched.)*

---

## Next

**AI-08 is complete.** What it hands forward:

- **AI-10** receives `ToolSet` and `ToolChoice` as request regions, and `AI-10.3` re-runs `ValidateAgainst` at the request boundary — which is why it is a method a request can call rather than logic inlined in a constructor.
- **AI-09** builds tool calls and results. It inherits the byte-fidelity discipline exactly (`V-REQ-17` argument bytes) and should reuse `tool_test.go`'s fixture strategy rather than a canonical-JSON one. Note the one place it *diverges*: doc 0002 asks AI-09.1 for a syntactic well-formedness rule on argument bytes that AI-08 deliberately does not apply to schema bytes — `design.md` § 8 records both sides.
- **AI-11.1** adds a cache-boundary marker to a tool declaration. `Tool` is a value with unexported fields and no encoding, so the marker is additive.
- **AI-13** lands the third closed vocabulary. `design.md` § 5.5 states the general move it should take from here: when a member needs something a bare integer cannot carry, widen the table row and put the payload in a separate value type — never widen the vocabulary type itself.
- **AI-26.4** is the reason `R-ATD-002` exists. The cache prefix it builds depends on schema bytes surviving unchanged and on the tool set iterating in the caller's order; both are now properties of the types rather than of anyone's discipline.
