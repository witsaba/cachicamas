# Tasks — roles and message identity

> **Change**: `cachicamas-ai-message-roles`
> **Milestone**: AI-05 · **Nodes**: AI-05.1 `[leaf]`, AI-05.2 `[leaf]`, AI-05.3 `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-message-roles/spec.md`, `design.md`
> **Branch**: `feat/ai-05-message-roles`
> **Depends on**: AI-00, AI-01, AI-02, AI-03, AI-04
> **Blocks**: AI-06 … AI-10
> **Evidence gate**: recorded green `make test` in `backend/agent/` (`go test -race -v ./...`)

---

## Node types and what they mean for this list

| Node | Type | Closes on |
| --- | --- | --- |
| AI-05.1 | `[leaf]` | Four test-list items taken red → green → refactored, **in order**, with item 4 exempt from red-first as a `*(pin)*` and **shown to bite** |
| AI-05.2 | `[leaf]` | Three test-list items, same discipline |
| AI-05.3 | `[leaf]` | Two test-list items, plus one appended during the change |

AI-05 has **no `[decision]` node**, so there is no `decision.md`. Every choice the milestone makes is made in `design.md` and is answerable to a scenario in `spec.md`.

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so every red step below follows `design.md` § 8: land the narrowest stub that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

**A note on the recorded output.** Every block below is verbatim `go test -race`. The **line numbers** are those of the run that produced the block, and each test file grew by an import line or two as later items needed one, so a line cited in an early block sits one or two lines earlier than the same assertion does in the merged file. The assertion text is what identifies it.

---

## Review Workload Forecast

Recorded after the fact, with the forecast that preceded it, because the difference is the point.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `specs/ai-message-roles/spec.md`, `design.md`, `tasks.md` | ~800 prose | **888 prose + this file** | Low | 30 min |
| AI-05.1 | `src/ai/role.go`, `src/ai/role_test.go` (+ the walking-skeleton half of `message.go`) | ~290 Go | **~451 Go** | **Medium** — the vocabulary is append-only and the pattern is copied three more times | 30 min |
| AI-05.2 | `src/ai/message.go`, `src/ai/message_test.go` | ~250 Go | **~380 Go** | **Medium** — the seam is inherited by AI-06, the keystone of wave 1 | 30 min |
| AI-05.3 | same two files | ~160 Go | **~304 Go** | Low — properties over the landed surface | 20 min |
| **Total** | 9 files (5 new markdown, 4 new Go) | ~700 Go | **1,135 Go (643 non-comment), 888+ prose** | **Medium** | **~110 min** |

### Budget reassessment — trigger 4 fired, and this is the record of it

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when "its projected diff, tests included, pushes the milestone past the review budget". **It did**, and it was forecast to in `explore.md` § 7 rather than discovered afterwards. The reassessment, rather than a silent overrun:

- **Production code is 108 non-comment lines** across two files — a closed vocabulary of three members, a minted identity, and a constructor with two rules. The surface itself is well inside the budget. What is large is the 535 non-comment lines of tests and the 267 lines of contract documentation on the declarations.
- **The ratio is the intended one for this milestone.** AI-05's deliverable is a set of properties that AI-06 … AI-10 stand on, and one of them — copy semantics — is the property doc 0002 singles out as producing "the most confusing class of test failure in a streaming package". Proving it takes two mechanisms, four mutation shapes and a copied-by-assignment case; none of those is assertable in fewer cases than are written.
- **No other split trigger fires.** The three leaves are one publicly observable behavior (what a message is); they are strictly ordered, so two agents could not work them concurrently; both test lists are inside the 7-item limit; and — the one that mattered — none needed a seam that did not exist. `explore.md` § 3.4 works that question in full, because the seam AI-06 inherits was the milestone's real risk.
- **The chain boundary is already cut.** Trigger 4 says "the node boundary is the PR-chain boundary", and the commits fall exactly there: one per leaf, plus the planning commit and this record. If the reviewer wants three PRs, no rework is required — the split already exists in history.
- **The argument against actually splitting.** Merging AI-05.1 and AI-05.2 alone lands a message whose content can be rewritten from outside by anyone who kept the slice they passed. That is a working type with a silent aliasing defect, which is exactly the partial-contract state defects **C1** and **C2** came from. The recommendation is one PR reviewed in three passes, in commit order.

---

## Phase AI-05.1 — Walking skeleton: a message with a role `[leaf]`

Four test-list items, in doc 0002's order. All output below is verbatim from `go test -race`.

### T-AMR-1 — Item 1: a message is constructible with each vocabulary role

- [x] **RED** — `TestMessage_EachVocabularyRole_ConstructsAndReadsBackExactly` against a `Message` with no fields, a `NewMessage` returning the zero value, and a `Role()` returning `0`.

```
--- FAIL: TestMessage_EachVocabularyRole_ConstructsAndReadsBackExactly (0.00s)
    --- FAIL: TestMessage_EachVocabularyRole_ConstructsAndReadsBackExactly/user (0.00s)
        role_test.go:60: msg.Role() = 0, want 1
    --- FAIL: TestMessage_EachVocabularyRole_ConstructsAndReadsBackExactly/assistant (0.00s)
        role_test.go:60: msg.Role() = 0, want 2
    --- FAIL: TestMessage_EachVocabularyRole_ConstructsAndReadsBackExactly/tool (0.00s)
        role_test.go:60: msg.Role() = 0, want 3
FAIL	github.com/cachicamas/backend/agent/src/ai	0.483s
```

- [x] **GREEN** — `Message` stores the role; `NewMessage` sets it; `Role()` returns it. All three subtests `--- PASS`.
- [x] **REFACTOR** — none needed; the implementation is three lines.

**Proves:** `R-AMSG-002`, `S-AMSG-004`, `S-AMSG-005`.

**This is the milestone's walking skeleton**, and its cost is worth recording: it forced the `Content` seam into existence at item 1 rather than at AI-05.2, because a message with no content is not a message. The external test package cannot implement an interface whose only method is unexported, so the test's content helper satisfies it by **embedding**:

```go
type part struct {
	ai.Content
	label string
}
```

That is the bypass `message.go`'s own GoDoc records and AI-06.3 item 2 closes. It compiled first time, which is the evidence that the seam is usable from another package without exposing anything.

---

### T-AMR-2 — Item 2: a role outside the vocabulary fails with an AI-04 sentinel

- [x] **RED** — `TestMessage_RoleOutsideTheVocabulary_FailsWithTheClosedVocabularySentinel` against the item-1 implementation, which validates nothing.

```
--- FAIL: TestMessage_RoleOutsideTheVocabulary_FailsWithTheClosedVocabularySentinel (0.00s)
    --- FAIL: .../the_zero_role (0.00s)
        role_test.go:94: NewMessage(0) returned no failure, want one
    --- FAIL: .../one_past_the_last_declared_role (0.00s)
        role_test.go:94: NewMessage(4) returned no failure, want one
    --- FAIL: .../far_outside_the_declared_range (0.00s)
        role_test.go:94: NewMessage(200) returned no failure, want one
    --- FAIL: .../the_maximum_of_the_underlying_type (0.00s)
        role_test.go:94: NewMessage(255) returned no failure, want one
FAIL	github.com/cachicamas/backend/agent/src/ai	0.276s
```

- [x] **GREEN** — the `roleNames` table lands, indexed by the constant, with `roleName` as its bounds-checked lookup; `NewMessage` composes one rule through `FirstFailure`, reporting `ErrNotInVocabulary` at `At("role")`. All four subtests `--- PASS`.
- [x] **REFACTOR** — none. The table is deliberately introduced *here*, driven by validity, and only widened by item 3 to serve rendering and parsing as well.

**Proves:** `R-AMSG-003`, `S-AMSG-006` … `S-AMSG-009`. The test also asserts that a *different* sentinel does **not** match, so a fail-everything implementation fails; that the position is `role`; and that the returned message is the zero value.

**The zero role is a case, not a courtesy.** `iota + 1` is what makes `Role(0)` — a struct field nobody set — indistinguishable from a wild value at the boundary. A vocabulary starting at `iota` would have made the first subtest pass for the wrong reason.

---

### T-AMR-3 — Item 3: the string form is stable, lowercase, and round-trips

This item took the state-before-red first, exactly as `design.md` § 8 predicts.

- [x] **BEFORE RED** — the test did not compile: `want.String undefined`, `undefined: ai.ParseRole`. That is not the red step.
- [x] **RED** — against the narrowest stubs that compile and fail: `String()` returning `""`, `ParseRole` returning `(0, nil)`.

```
--- FAIL: TestRole_StringForm_RoundTripsThroughParseAndRender (0.00s)
    --- FAIL: .../every_member_renders_lowercase_and_parses_back (0.00s)
        role_test.go:138: Role(1).String() = "", want a non-empty rendering
        role_test.go:138: Role(2).String() = "", want a non-empty rendering
        role_test.go:138: Role(3).String() = "", want a non-empty rendering
    --- FAIL: .../only_the_exact_rendering_parses (0.00s)
        role_test.go:169: ParseRole("User") =  with no failure, want a failure
        role_test.go:169: ParseRole(" user") =  with no failure, want a failure
        role_test.go:169: ParseRole("system") =  with no failure, want a failure
        … 16 rejected forms …
    --- FAIL: .../the_empty_string_is_empty,_not_unrecognised (0.00s)
        role_test.go:186: ParseRole(""): errors.Is(err, ErrEmpty) = false, want true (err = <nil>)
    --- FAIL: .../a_non-member_renders_diagnostically_and_does_not_parse_back (0.00s)
        role_test.go:199: Role(4).String() = "", want it to identify the value
    --- FAIL: .../the_offered_string_never_reaches_the_failure (0.00s)
        role_test.go:214: ParseRole("CACHICAMA-SENTINEL-BODY-a71c") returned no failure, want one
FAIL	github.com/cachicamas/backend/agent/src/ai	0.481s
```

- [x] **GREEN** — `String` renders the table entry or `role(N)`; `ParseRole` scans the ordered table and composes two rules through `FirstFailure` — `ErrEmpty` first, `ErrNotInVocabulary` second. All five subtests `--- PASS`.
- [x] **REFACTOR** — none in this step; the file's contract documentation was written at the end of the phase.

**Proves:** `R-AMSG-004`, `S-AMSG-010` … `S-AMSG-014`.

**Sixteen rejected forms, and each class is deliberate.** Case variants and padded forms, because a parser that folds case is a vocabulary that is advisory. Provider spellings — `system`, `developer`, `human`, `model` — because `V-REQ-01`'s last clause puts that mapping in an adapter. And `role(4)`, because a diagnostic rendering that round-tripped would be a second, undeclared way into the vocabulary.

**The redaction inheritance is exercised here for the first time by a real caller.** `ParseRole` is given a string, and the string is caller data; the failure carries only `At("role")`, a constant this package wrote. The subtest feeds a distinctive sentinel body and asserts its absence — AI-04's posture costing one line at the call site rather than a policy.

---

### T-AMR-4 — Item 4 `*(pin)*`: registration is exhaustive

- [x] **PIN** — `TestRole_DeclaredVocabulary_IsExhaustivelyTabulated`, green from birth and exempt from red-first per doc 0002's leaf anatomy. It enumerates `Roles()` and asserts, per member: a non-empty rendering that is not the `role(N)` diagnostic form, lowercase, a parse round-trip, and acceptance by `NewMessage`. It also asserts two calls to `Roles()` agree, so the declaration order is stable.

- [x] **Shown to bite.** A scratch member declared without a table entry, its failure recorded, then removed:

```
    role_test.go:258: Role(4) has no table entry: String() = "role(4)"
    role_test.go:266: Role(4): ParseRole("role(4)") returned role: value is outside a closed vocabulary, want it to round-trip
    role_test.go:272: Role(4) is declared but NewMessage rejects it: role: value is outside a closed vocabulary
--- FAIL: TestRole_DeclaredVocabulary_IsExhaustivelyTabulated (0.00s)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.309s
```

The scratch violation was two lines — `RoleScratch` appended to the constant block and `roleEnd` moved past it — which is exactly the shape of a real half-finished append. Restored to `--- PASS` immediately afterwards.

**The bite was strengthened during this step.** The first attempt used `continue` after the missing-entry check, so the scratch member produced **one** failure. It was removed so the parse and construction assertions also run, and the bite now reports all three symptoms — the tabulation gap, the broken round trip, and the member its own constructor rejects. A pin that reports one of three is a weaker claim than the pin makes.

**Proves:** `R-AMSG-005`, `S-AMSG-015`, `S-AMSG-016`.

**Why it can bite at all** is `design.md` § 3 rule 3, and it is the part worth copying: `Roles()` enumerates the **constant space**, not the name table. An enumeration derived from `roleNames` would have listed exactly the members that have an entry, so the omission would have been invisible and the pin decorative. AI-07, AI-08 and AI-13 each land a closed vocabulary; the rule is stated in `role.go`'s own file comment so it travels with the code rather than with this document.

---

## Phase AI-05.2 — Identity and content ordering `[leaf]`

### T-AMR-5 — Item 1: a stable, comparable identity

- [x] **RED** — `TestMessage_Identity_IsStableComparableAndDistinct` against stubs: `MessageID` with an unexported counter, `IsZero` real, `String()` returning `""`, and `ID()` returning the zero value.

```
--- FAIL: TestMessage_Identity_IsStableComparableAndDistinct (0.00s)
    --- FAIL: .../repeated_reads_agree (0.00s)
        message_test.go:40: msg.ID() = , want a minted identity
    --- FAIL: .../identity_renders_diagnostically (0.00s)
        message_test.go:128: MessageID.String() = "", want a diagnostic rendering
        message_test.go:132: the zero MessageID renders as "", want a diagnostic rendering
    --- FAIL: .../identical_messages_have_different_identities (0.00s)
        message_test.go:64: two messages share the identity , want distinct identities
    --- FAIL: .../concurrent_construction_mints_distinct_identities (0.00s)
        message_test.go:109: goroutine 0 minted an unset identity
        … 64 goroutines …
FAIL	github.com/cachicamas/backend/agent/src/ai	0.288s
```

Note which subtest is **absent** from that list: *an unconstructed message has no identity* passed against the zero-returning stub, as it must — it is the assertion pinning `S-AMSG-009` and `S-AMSG-019`, and a stub that minted eagerly would have failed it.

- [x] **GREEN** — `lastMessageID atomic.Uint64`, `mintMessageID` on the success path of `NewMessage` only, `String` rendering `msg-N` and `msg-unset`. All five subtests `--- PASS`, with the concurrent one under `-race`.
- [x] **REFACTOR** — none in this step.

**Proves:** `R-AMSG-006`, `S-AMSG-017` … `S-AMSG-021`.

**The C3 question, answered in the code rather than only here.** A package-level counter is what defect **C3** was, so `lastMessageID`'s GoDoc carries the distinction: C3's contract was a statement about the counter's *value* — "every stream's first event carries 1, every stream is independently contiguous" — which a process-global cannot satisfy for the second stream in a process. `V-REQ-03` states no property of the value at all. The observable contract is that two messages differ, and nothing would change if the counter were replaced by random bytes tomorrow.

---

### T-AMR-6 — Item 2: content order round-trips exactly

- [x] **RED** — `TestMessage_ContentOrder_RoundTripsExactly` against a `Content()` stub returning `nil`, per `design.md` § 8's note that returning the stored slice would pass trivially.

```
--- FAIL: TestMessage_ContentOrder_RoundTripsExactly (0.00s)
    --- FAIL: .../one_element (0.00s)
        message_test.go:170: len(msg.Content()) = 0, want 1
    --- FAIL: .../three_distinct_elements (0.00s)
        message_test.go:170: len(msg.Content()) = 0, want 3
    --- FAIL: .../the_same_element_repeated (0.00s)
        message_test.go:170: len(msg.Content()) = 0, want 3
    --- FAIL: .../repetitions_interleaved_with_distinct_elements (0.00s)
        message_test.go:170: len(msg.Content()) = 0, want 6
FAIL	github.com/cachicamas/backend/agent/src/ai	0.317s
```

- [x] **GREEN** — `Message` gains a `content []Content` field, stored and returned. All four subtests `--- PASS`.
- [x] **REFACTOR** — none, and one thing was deliberately **not** done: the sequence is stored by reference and returned by reference. Nothing yet demands a copy, and AI-05.3 is the leaf that drives both copies in. Introducing them here would have cost AI-05.3 its red.

**Proves:** `R-AMSG-007`, `S-AMSG-022` … `S-AMSG-024`.

**The repeated-element case is the one that carries the item.** Three distinct elements round-trip through a set, a map keyed by the element, or anything that deduplicates. `[a, a, a]` and `[a, b, a, a, c, b]` do not, and the failure would otherwise be a repeated tool call or two identical text parts silently disappearing from a request.

---

### T-AMR-7 — Item 3: a message with no content fails with an AI-04 sentinel

- [x] **RED** — `TestMessage_NoContent_FailsWithTheEmptySentinel` against `NewMessage`'s single role rule.

```
--- FAIL: TestMessage_NoContent_FailsWithTheEmptySentinel (0.00s)
    --- FAIL: .../the_three_ways_of_saying_nothing_fail_identically (0.00s)
        message_test.go:209: no arguments: NewMessage returned no failure, want one
        message_test.go:209: an empty slice: NewMessage returned no failure, want one
        message_test.go:209: a nil slice: NewMessage returned no failure, want one
FAIL	github.com/cachicamas/backend/agent/src/ai	0.307s
```

The second subtest — *both rules violated reports the first in the documented order* — passed against the stub, correctly: the role rule already existed and already won. It becomes a real assertion the moment a second rule exists, which is the next line of production code.

- [x] **GREEN** — a second rule in `FirstFailure`, reporting `ErrEmpty` at `At("content")`. Both subtests `--- PASS`.
- [x] **REFACTOR** — none.

**Proves:** `R-AMSG-008`, `S-AMSG-025` … `S-AMSG-027`. The order subtest runs 256 times and asserts an identical rendered failure on every run.

---

## Phase AI-05.3 — Copy semantics `[leaf]`

doc 0002 attaches an unusual note to this leaf: it "is the one whose absence produces the most confusing class of test failure in a streaming package", and it is what lets AI-21.6's request capture assert on history without defensive copying at the call site. Both reds below are the confusing failure itself, reproduced deliberately.

### T-AMR-8 — Item 1: construction copies, it does not alias

- [x] **RED** — `TestMessage_CallerMutatesWhatItPassed_MessageIsUnchanged` against AI-05.2's implementation, which stores the caller's slice.

```
--- FAIL: TestMessage_CallerMutatesWhatItPassed_MessageIsUnchanged (0.00s)
    --- FAIL: .../replacing_an_element_in_place (0.00s)
        message_test.go:282: msg.Content()[0] = {<nil> c}, want {<nil> a}
        message_test.go:282: msg.Content()[1] = {<nil> d}, want {<nil> b}
    --- FAIL: .../reusing_the_backing_array_for_a_second_message (0.00s)
        message_test.go:304: msg.Content()[0] = {<nil> c}, want {<nil> a}
        message_test.go:304: msg.Content()[1] = {<nil> d}, want {<nil> b}
FAIL	github.com/cachicamas/backend/agent/src/ai	0.311s
```

**That output is the whole argument for the item.** Nobody wrote to `msg`. A variadic parameter called with a slice spread does not copy, so `NewMessage(role, parts...)` aliased the caller's backing array, and the message's content changed underneath it.

- [x] **GREEN** — `slices.Clone(content)` in `NewMessage`. Both subtests `--- PASS`.
- [x] **REFACTOR** — none.

**Proves:** `R-AMSG-010`, `S-AMSG-031`, `S-AMSG-032`.

**Neither mutation shape reallocates**, on purpose. An append beyond capacity moves the caller's slice to a new array and the defect hides itself; in-place replacement and backing-array reuse are the two shapes that expose it.

---

### T-AMR-9 — Item 2: reads return copies

- [x] **RED** — `TestMessage_CallerMutatesWhatItRead_MessageIsUnchanged` against a `Content()` returning the stored slice.

```
--- FAIL: TestMessage_CallerMutatesWhatItRead_MessageIsUnchanged (0.00s)
    --- FAIL: .../a_reader_cannot_rewrite_the_message (0.00s)
        message_test.go:357: msg.Content()[0] = {<nil> c}, want {<nil> a}
    --- FAIL: .../two_readers_are_independent (0.00s)
        message_test.go:369: the second reader saw {<nil> c} at index 0, want {<nil> a}
        message_test.go:371: msg.Content()[0] = {<nil> c}, want {<nil> a}
    --- FAIL: .../a_message_copied_by_assignment_is_independent (0.00s)
        message_test.go:382: msg.Content()[0] = {<nil> c}, want {<nil> a}
        message_test.go:383: msg.Content()[0] = {<nil> c}, want {<nil> a}
FAIL	github.com/cachicamas/backend/agent/src/ai	0.317s
```

The fourth subtest — *the role vocabulary cannot be rewritten* — passed against the stub, because `Roles()` was already a function returning a fresh slice. It is recorded here rather than dropped: it is the same property at a different scope, and a later "optimisation" of `Roles()` into a package-level `var` fails it.

- [x] **GREEN** — `slices.Clone(m.content)` in `Content()`. All four subtests `--- PASS`.
- [x] **REFACTOR** — the contract GoDoc for `message.go` written: the seam's open door, the copy contract as two mechanisms, the C3 distinction, and the rule order on `NewMessage`. Each file's banner comment was also separated from its `package` clause by a blank line — revive reads an attached banner as a second package comment, and `validation.go` already uses the separated shape.

**Proves:** `R-AMSG-010`, `S-AMSG-002`, `S-AMSG-033` … `S-AMSG-035`.

---

### T-AMR-10 — Item 3 *(appended, pin)*: no input causes a panic

Appended to AI-05.3's test list during this change, per doc 0002's rule that a newly discovered *test case* is appended to the owning leaf's list rather than chased ad hoc. It was written into `spec.md` as `R-AMSG-011` during the spec phase, so it is recorded as an append rather than presented as if doc 0002 had listed it.

- [x] **PIN** — `TestMessage_ExtremeInputs_NeverPanics`, green from birth. Six shapes: a nil content element; nil and constructed elements mixed; ten thousand elements; every one of the 256 values of the role's underlying type rendered, parsed and constructed with; the zero message read every way; and an empty and a 40,000-byte name parsed as a role. All `--- PASS`.

- [x] **Not shown to bite, and the reason is stated rather than omitted.** Unlike AI-05.1's pin there is no guard to remove: the shapes above are total by construction today, because nothing in the constructor inspects an element and nothing in the package recurses. A scratch violation would have to be an *added* defect rather than a removed guard, which proves nothing about the code that exists.

**Its value is forward-looking and specific.** AI-06.3 adds a rule that rejects an unconstructed content part and AI-10 adds rules at request scope; both touch this constructor, and both are written against inputs — a nil element, a very large sequence, a role at the end of its range — that a rule is easy to write without considering. It is the same standing AI-04's totality item has, one milestone earlier in the chain that will exercise it.

**Proves:** `R-AMSG-011`, `S-AMSG-036` … `S-AMSG-038`.

---

## Verification pass (closes the milestone)

Ordered by cost of a missed defect. The first three run; the rest are inspection.

- [x] **V-1** — `make test` in `backend/agent/` green under `-race`, both packages. `ok github.com/cachicamas/backend/agent/src/agenttest 1.281s` · `ok github.com/cachicamas/backend/agent/src/ai 1.534s`. 92 passing cases in total, **45 of them this milestone's**.
- [x] **V-2** — Both AI-00 import guards pass (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires`), and `go.mod` still carries zero `require` directives (`NFR-AMSG-A`). The two new files import `slices`, `strconv` and `sync/atomic` — standard library only.
- [x] **V-3** — `make lint` (`go vet ./...` then `golangci-lint run` with govet, errcheck, staticcheck, unused and revive): **0 issues**.
- [x] **V-4** — Every test function follows `Test<Subject>_<Behavior>_<Expectation>` and carries a banner citing its leaf ID (`NFR-AMSG-D`).
- [x] **V-5** — This change declares **no error variable, no error type and no sentinel** (`NFR-AMSG-B`). Every failure is `Invalid(ErrNotInVocabulary | ErrEmpty, At(…))` composed through `FirstFailure` — exactly what AI-04's `tasks.md` § Next predicted AI-05 would need, and nothing more.
- [x] **V-6** — The content seam exposes one unexported method and nothing else: no payload, no kind, no accessor, no constructor, no rendering, and no validation of an element (`R-AMSG-009`, `S-AMSG-028` … `S-AMSG-030`). Both of AI-06.1's properties are untouched, and the embedding bypass is documented in the seam's own GoDoc rather than left to be discovered.
- [x] **V-7** — Each vocabulary member carries its citable case in `role.go`'s file comment: doc 0001 § 3.3 row 5 for `RoleUser` and `RoleAssistant`, doc 0002 AI-10.3 item 3 for `RoleTool`. The absence of `RoleSystem` carries its reason and its append path.
- [x] **V-8** — Nothing in the two new files decides anything by unordered iteration. Neither contains a map; `roleNames` is an indexed slice and its GoDoc says why, following AI-04's `ruleClasses`.
- [x] **V-9** — No exported declaration names a content-part kind, a payload, a request, a tool, a stream or a provider concept. The exported surface is `Role`, `RoleUser`, `RoleAssistant`, `RoleTool`, `Roles`, `String`, `ParseRole`, `Content`, `MessageID`, `IsZero`, `String`, `Message`, `NewMessage`, `ID`, `Role`, `Content`.
- [x] **V-10** — The register is **unamended**, and `explore.md` § 6 records the check plus the three near-misses that were considered and refused.
- [x] **V-11** — The diff touches only `openspec/changes/cachicamas-ai-message-roles/` and four new files under `backend/agent/src/ai/`. Nothing under `src/agenttest/`, no `validation.go` or `validation_test.go` change (AI-13 is concurrent on those), no `doc.go`, no build, module or infrastructure file, no other backend module.
- [x] **V-12** — doc 0002 needed **no amendment**. The revert-and-record clause was not invoked: no leaf's first red failed to drive green in small steps, and the one node whose seam might not have existed — AI-05.2's ordering against a content type AI-06 has not defined — is worked in `explore.md` § 3.4 and `design.md` § 4, which show AI-06.1's four checklist items each remaining open.

---

## Review focus

For the reviewer, in priority order — the first two are where a defect is expensive.

1. **The seam, against AI-06.1's actual words.** Read `design.md` § 4.2's table beside doc 0002's AI-06.1 item 1, and ask whether any of the four checklist items has been narrowed. The specific test is item 1(b): can AI-06 still choose *either* a compile-time seal *or* a validation seal? `message.go`'s GoDoc argues it can, because the embedding route is open and documented. If it cannot, this milestone handed the keystone a conclusion.
2. **Copy semantics as two mechanisms.** `NewMessage` clones and `Content()` clones. Removing either leaves a type that passes half its tests. T-AMR-8's and T-AMR-9's recorded reds are the symptom each one prevents; check that they are the *aliasing* symptom and not a missing return.
3. **The absent `system` role.** `role.go`'s file comment argues it from `V-REQ-19` and doc 0001 § 3.3 row 8. If a reviewer believes a system-role message is needed, the correct move is an appended constant with its table entry — the pin makes a half-append fail — not a reopening of the vocabulary's shape.
4. **The pin's enumeration source.** `Roles()` walks the constants, not `roleNames`. If that is ever "simplified" to range over the table, the pin passes forever and stops being able to detect anything. It is the single line on which AI-07's, AI-08's and AI-13's pins also depend.
5. **Minted identity and defect C3.** `lastMessageID`'s GoDoc makes the argument. Check it against `V-STR-13`'s actual wording rather than against the memory of "a package-level counter was the bug".
6. **No second failure vocabulary.** Grep the two new files for `errors.New`, `fmt.Errorf` and `Err` — there is none. Every failure is AI-04's, which is the property nine milestones were sequenced to get.

---

## Acceptance criteria for the milestone

1. Every test-list item of AI-05.1, AI-05.2 and AI-05.3 closed with recorded red and green output, in order. **Met** — nine reds across nine items, one of which needed the state-before-red recorded first.
2. The exhaustiveness pin is shown to bite and the scratch violation removed. **Met** — three recorded failures, then `--- PASS`.
3. The verification pass V-1 … V-12 recorded complete. **Met.**
4. `spec.md`'s `R-AMSG-001` … `R-AMSG-011` hold. **Met.**
5. **doc 0002's own acceptance criterion:** a message is constructible only through the rules, its content order round-trips, and a caller cannot mutate a constructed message from outside. **Met**, by T-AMR-2/T-AMR-7, T-AMR-6, and T-AMR-8/T-AMR-9 respectively.

## Next

- **AI-06** — `cachicamas-ai-content-parts`, the keystone of wave 1. It inherits a seam and not a conclusion: `Content` declares one unexported method, exposes nothing readable, validates nothing, and does not close the embedding route. AI-06.1's four checklist items are each open, and `design.md` § 4.2 maps them to the reason each one is. The first thing AI-06 should check is that mapping.
- Then AI-07 and AI-08, which are the first two reuses of the closed-vocabulary pattern in `role.go`'s file comment. Neither should re-derive it; both should copy it and record which of the four rules they varied.
