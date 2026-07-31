# Tasks — content parts: readable and sealed

> **Change**: `cachicamas-ai-content-parts`
> **Milestone**: AI-06 · **Nodes**: AI-06.1 `[decision]`, AI-06.2 `[leaf]`, AI-06.3 `[leaf]`, AI-06.4 `[guard]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `decision.md`, `specs/ai-content-parts/spec.md`, `design.md`
> **Branch**: `feat/ai-06-content-parts`
> **Depends on**: AI-00 … AI-05 merged
> **Blocks**: AI-07, AI-09, AI-10, AI-26 — and, through AI-10, the rest of wave 2
> **Evidence gate**: recorded green `make test` and `make lint` in `backend/agent/` (`go test -race -v ./...`)

---

## Node types and what they mean for this list

| Node | Type | Closes on |
| --- | --- | --- |
| AI-06.1 | `[decision]` | The decision artifact answers every closing-checklist item and is merged, plus the seam closure it authorises. |
| AI-06.2 | `[leaf]` | Every test-list item taken red → green → refactored, **in order** |
| AI-06.3 | `[leaf]` | Same, across three items, one of which is answered by the compiler rather than by an assertion |
| AI-06.4 | `[guard]` | Green is not enough. The guard must be **recorded failing** against a deliberately broken registration, twice, and the break dropped |

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so every red step below follows `design.md` § 7: land the narrowest thing that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

### A note on provenance — what was observed and what was reconstructed

This milestone was implemented across two sessions. The first session's process died mid-AI-06.3 and **its red transcripts were not preserved**. This document is explicit about the difference everywhere it matters:

- **Observed** — the transcript below was produced by a command run while writing this document, and pasted verbatim.
- **Reconstructed** — the step is recorded from the commit that landed it and from the artifacts in the tree. Where a red transcript is missing it is said so plainly; it is never invented.

AI-06.1 and AI-06.2 are **reconstructed**, with their greens re-observed. AI-06.3 item 1 is a **re-observed red**: the rule its test drives was already landed but uncommitted, so the rule was removed, the failure recorded, and the rule restored. AI-06.3 items 2 and 3 and all of AI-06.4 are **observed** end to end.

---

## Review Workload Forecast

Recorded after the fact, with the forecast that preceded it, because the difference is the point.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `specs/ai-content-parts/spec.md`, `design.md`, `tasks.md` | ~800 prose | **853 prose + this file** | Low | 30 min |
| AI-06.1 decision + seam closure | `decision.md`, `content_part.go`, `message.go`, `text_content.go`, AI-05's two test files | ~370 prose, ~200 Go | **369 prose, 277 Go added / 87 removed** | **High** — four milestones plus AI-26 inherit the shape | 45 min |
| AI-06.2 text content | `text_content.go`, `content_part.go`, `content_part_test.go`, `agenttest/content_part_test.go` | ~350 Go | **646 Go added / 6 removed** | **Medium** — the first payload-carrying value type in Layer 1 | 40 min |
| AI-06.3 the seal | `content_part.go`, `message.go`, `content_part_test.go`, `content_part_internal_test.go`, two `testdata/` programs | ~300 Go | **656 Go** | **Medium** — one half is the compiler's, and the proof shells out | 40 min |
| AI-06.4 the guard | `content_part_registry_test.go` | ~200 Go | **543 Go** | Low — no production code, and it bites | 25 min |
| **Total** | 16 files (11 new Go, 3 modified Go, 6 new markdown) | ~850 Go, ~1170 prose | **2114 Go added / 87 removed (1219 non-comment), 1222 prose + this file** | **Medium-High** | **~180 min** |

Non-comment Go, by file, so the shape of the overrun is visible rather than asserted:

| File | Total | Non-comment | Kind |
| --- | --- | --- | --- |
| `src/ai/content_part.go` | 315 | **85** | production |
| `src/ai/text_content.go` | 121 | **35** | production |
| `src/ai/message.go` (whole file, after +26 / −48) | 177 | **46** | production |
| `src/ai/content_part_test.go` | 694 | 476 | test |
| `src/ai/content_part_registry_test.go` | 543 | 365 | test (guard) |
| `src/ai/content_part_internal_test.go` | 184 | 116 | test |
| `src/agenttest/content_part_test.go` | 84 | 51 | test |
| `src/ai/testdata/handrolled/main.go` | 66 | 21 | fixture — must not compile |
| `src/ai/testdata/constructed/main.go` | 41 | 24 | fixture — control |

### Budget reassessment — trigger 4 fired, and it was forecast before the work

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when "its projected diff, tests included, pushes the milestone past the review budget". **It did, by a wide margin, and `explore.md` § 7 said so before a line was written.** AI-04, AI-05 and AI-13 each recorded the same overrun; this is the fourth, and the largest. The reassessment, rather than a silent overrun or a cut test list:

- **Production is 166 non-comment lines across three files.** The surface AI-07, AI-09, AI-10 and AI-26 inherit is small. What is large is (a) the contract documentation, which is where the decision's reasoning is put in front of the person who will add the second kind, and (b) the tests, which are 1053 non-comment lines.
- **The test weight is structural, not padding.** The milestone's deliverable is *two constitutive properties*, and doc 0001 records that the retired Layer 1 shipped each without the other, in opposite directions. Proving "readable" needs a second package (`agenttest`). Proving "sealed" needs three separate mechanisms — a runtime rejection, a **compile failure** with its own control program, and a reflection assertion — because no single mechanism covers the eight bypasses of `decision.md` § 3.3. The guard then needs a fourth, an AST scan, because Go checks no sum's exhaustiveness.
- **Split trigger 1 does not fire, and it is the one that would matter.** AI-06.2 and AI-06.3 are not two behaviors; they are the two halves of one contract, and merging either alone ships exactly the half-contract state that produced C1 and C2. AI-06.4 is a guard over the result and cannot precede it.
- **Split triggers 2, 3 and 5 do not fire.** The nodes are strictly ordered, so no two agents could work them concurrently; no node needed a seam that did not exist; and no leaf's test list exceeded 7 items (AI-06.2 has 4 plus 2 appended, AI-06.3 has 3 plus 1 appended, AI-06.4 has 2).
- **The chain boundary is already cut.** The commits fall on node boundaries and the PR is reviewable in six passes in commit order. If a reviewer wants it chained, no rework is needed: the split exists in history.
- **Nothing was cut to fit.** Per the standing instruction, the test list was appended to during implementation, not pruned. Two cases were appended (below), and both stayed.

---

## Phase AI-06.1 — one content-part strategy `[decision]`

**Deliverable:** `openspec/changes/cachicamas-ai-content-parts/decision.md` and the seam closure it authorises.

*Reconstructed from commits `2ae3be9` and `2a09c3c` and from the artifacts in the tree. No red step applies — a `[decision]` node produces no behavior.*

### T-ACP-1 — The strategy, with both properties demonstrated

- [x] Choose one shape and demonstrate readability and sealing against it, rather than asserting the pair is achievable.

**Decided:** `type Part struct{ payload partPayload }` — one exported opaque value type with **one unexported field**. `partPayload` is unexported with unexported methods `kind() PartKind` and `validate(at Path) *Violation`. `PartKind` is `uint8`, closed at `iota + 1`. `NewText(string) (Part, error)` is the only door in; `Part.Kind()` is derived from the payload and never stored; `Part.Text() (string, bool)` is the exported accessor.

The argument in one sentence, from § 3.4: **a struct can export behavior without exporting construction; an interface cannot.** For an interface, "readable" and "implementable" are the same word. Readability is delivered by exported methods on `Part`; sealing by the unexported field those methods read; the two surfaces are disjoint, which is exactly why widening one does not widen the other — the failure mode that produced C1 and C2 as separate defects of one contract.

**Evidence:** `decision.md` §§ 3.1–3.4. Five alternatives are argued at full strength in § 4, three of which were actually shipped by the retired design or by AI-05.

### T-ACP-2 — The seam AI-05 held open is closed

- [x] Remove AI-05's `Content` interface and make `Message` hold `[]Part`.

**Decided and landed** in `2a09c3c`, as its own commit, with AI-05's tests migrated and green. `message.go`'s own documentation records the handover: AI-05 landed `Content` as "not a seal on purpose" because closing it would have taken a decision that was not AI-05's. This milestone took it. Leaving `Content` as an alias was considered and rejected in `decision.md` § 10: it would leave the embedding bypass alive, which *is* the defect.

**Cost conceded openly:** a name exported one milestone ago disappears. Nothing outside the module imports it, and ADR 0005's v1 freeze is AI-38.

### T-ACP-3 — Validation, and whether a sentinel is needed

- [x] Decide the rule order and whether "an unconstructed value" needs a rule class of its own — the question AI-04 § 8 left open for this milestone.

**Decided: no new sentinel.** Because the kind is derived from the payload, a part that skipped construction *has no kind*, and a value with no kind is a value outside a closed vocabulary — which is what `ErrNotInVocabulary` names. AI-05 set the precedent one milestone earlier with `Role(0)`. **The register is unamended**, and `decision.md` § 11 records the two candidate terms that were considered and rejected.

**Evidence:** `decision.md` §§ 7.1–7.4.

### T-ACP-4 — The procedure for adding a kind, and its guard

- [x] State the five steps and specify what AI-06.4 must assert, including where the witnesses live.

**Decided:** five steps mechanized as three legs plus a documentation claim; witnesses live **in the guard's own test file**, keyed by the declared constant; production carries only `partKindNames`, a slice indexed by the constant and never a map. The guard enumerates the **declared constant space**, not the name table.

**Evidence:** `decision.md` § 8, `design.md` §§ 5.1–5.3.

---

## Phase AI-06.2 — text content, constructible and readable `[leaf]`

**Deliverable:** `src/ai/text_content.go`, plus the tests in `src/ai/content_part_test.go` and `src/agenttest/content_part_test.go`.

*Reconstructed from commit `6a88fb1`. **The red transcripts from the first session were not preserved and are not reproduced here.** The green below was re-observed while writing this document. `design.md` § 7 records the stub each item was taken red against, and those stubs are visible in the diff of `2a09c3c` (which landed `Text()` returning `"", false` and `Kind()` returning `0`).*

### Test list

1. - [x] An external package constructs a text part, places it in a message, and reads the exact text back — byte-equal.
2. - [x] The kind a part reports always matches the payload it yields.
3. - [x] The construction rules fail through AI-04's sentinels, in the documented order.
4. - [x] Valid text survives construction unaltered.
5. - [x] *(appended)* The kind vocabulary is closed, stable and immutable.
6. - [x] *(appended)* A part's diagnostic rendering carries no payload.

Items 5 and 6 were **appended during implementation**, per doc 0002's rule that a discovered case joins the owning leaf's list rather than being chased ad hoc. Item 6's discovery is recorded on the test itself: writing item 1 showed `%v` printing the payload, because fmt prints the unexported fields of a struct with no `String` method. That is a redaction defect on the first payload-carrying value type in the package, and `Part.String` and `Part.GoString` exist because of it.

### GREEN — re-observed 2026-07-31

```
--- PASS: TestPart_String_CarriesNoPayload (0.00s)
--- PASS: TestPart_KindAndPayload_Agree (0.00s)
--- PASS: TestPartKinds_Vocabulary_IsClosedStableAndImmutable (0.00s)
--- PASS: TestNewText_ValidText_SurvivesConstructionUnaltered (0.00s)
--- PASS: TestNewText_RuleViolations_FailWithTheDocumentedSentinels (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	0.711s
--- PASS: TestContentPart_TextConstructedAndReadFromAnExternalPackage_RoundTripsByteEqual (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/agenttest	0.416s
```

### Defect found in the landed commit, and fixed here

`6a88fb1` landed with **`make lint` failing**, which the milestone's evidence gate forbids. Two findings, both in `content_part_test.go`:

```
src/ai/content_part_test.go:229:26: ST1018: string literal contains Unicode format characters, consider using escape sequences instead (staticcheck)
		{"right-to-left text", "‫مرحبا بالعالم‬"},
```

The right-to-left test case wrote U+202B and U+202C literally. That is the Trojan Source hazard: a source file whose visible order differs from its byte order, in a test whose entire subject is bytes. Rewritten as `‫` / `‬` escapes with the reason recorded inline. The second finding was an unused subtest parameter (revive). Both are fixed in commit `17f952c`; `make lint` reports `0 issues` from there on.

---

## Phase AI-06.3 — unconstructed parts cannot reach a message `[leaf]`

**Deliverable:** rule 3 of `NewMessage`, `Part.validate`, `validateContent`, the two `testdata/` programs, and the internal reuse pin.

### Test list

1. - [x] The zero value of any exported part-related type, placed directly into message content, is rejected with an AI-04 sentinel. **This is doc 0001's defect C1.**
2. - [x] An external package attempting to implement the part contract: the compiler prevents it, **and** validation rejects what the compiler still allows. Both halves proven separately.
3. - [x] The same value offered through the request path is rejected there too, in the documented validation order.
4. - [x] *(appended)* Reflection is not a way in either.

Item 4 was **appended** when writing the compile proof: bypass 7 of `decision.md` § 3.3 has a compile-time half (`p.payload = …`) and a runtime half (`reflect` refusing to `Set`), and the runtime half had no home.

---

### Item 1 — the zero part, rejected at `content[i]`

**RED** — re-observed. The rule this test drives was already written but uncommitted when this session began, so it was removed from `NewMessage` (`validateContent` left in place, unreferenced), the failure recorded, and the rule restored. The transcript is therefore of the state *immediately before* the rule existed:

```
--- FAIL: TestMessage_UnconstructedContentElement_IsRejected (0.00s)
    --- FAIL: TestMessage_UnconstructedContentElement_IsRejected/a_part_promoted_out_of_an_embedding_type_is_the_zero_part (0.00s)
        content_part_test.go:491: ai.NewMessage accepted a promoted zero part, want a rejection
    --- FAIL: TestMessage_UnconstructedContentElement_IsRejected/the_zero_part_is_rejected,_positioned_by_ordinal (0.00s)
        --- FAIL: TestMessage_UnconstructedContentElement_IsRejected/the_zero_part_is_rejected,_positioned_by_ordinal/the_only_element (0.00s)
            content_part_test.go:448: ai.NewMessage returned no failure, want one
        --- FAIL: TestMessage_UnconstructedContentElement_IsRejected/the_zero_part_is_rejected,_positioned_by_ordinal/several_unconstructed_elements_report_the_first (0.00s)
            content_part_test.go:448: ai.NewMessage returned no failure, want one
        --- FAIL: TestMessage_UnconstructedContentElement_IsRejected/the_zero_part_is_rejected,_positioned_by_ordinal/the_last_element (0.00s)
            content_part_test.go:448: ai.NewMessage returned no failure, want one
        --- FAIL: TestMessage_UnconstructedContentElement_IsRejected/the_zero_part_is_rejected,_positioned_by_ordinal/the_third_of_four (0.00s)
            content_part_test.go:448: ai.NewMessage returned no failure, want one
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.434s
```

**Minimal code:** `Part.validate` (payload present, kind registered, payload's own rules), `validateContent`, and rule 3 of `NewMessage`.

**GREEN**

```
--- PASS: TestMessage_UnconstructedContentElement_IsRejected (0.00s)
    --- PASS: TestMessage_UnconstructedContentElement_IsRejected/the_role_rule_still_wins_over_the_element_rule (0.00s)
    --- PASS: TestMessage_UnconstructedContentElement_IsRejected/a_part_promoted_out_of_an_embedding_type_is_the_zero_part (0.00s)
    --- PASS: TestMessage_UnconstructedContentElement_IsRejected/a_message_that_was_constructed_holds_only_valid_parts (0.00s)
    --- PASS: TestMessage_UnconstructedContentElement_IsRejected/the_zero_part_is_rejected,_positioned_by_ordinal (0.00s)
        --- PASS: …/the_only_element (0.00s)
        --- PASS: …/the_last_element (0.00s)
        --- PASS: …/the_third_of_four (0.00s)
        --- PASS: …/several_unconstructed_elements_report_the_first (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	0.249s
```

**What it establishes.** `ErrNotInVocabulary` at `content[i]`, never `ErrEmpty`; the *first* failing element is the one reported; the role rule still wins over the element rule on 64 consecutive runs (V-FAIL-04, order is contract); and a rejected construction returns a message whose identity reports unset.

---

### Item 2 — the part contract is unimplementable, and the compiler says so

The seal is **two mechanisms with a clean boundary**, and this item proves each half separately, exactly as `decision.md` § 3.3 divides them:

| Bypass | Attempt from another package | Answered by |
| --- | --- | --- |
| 1 | `var p ai.Part` | **validation** — item 1 above |
| 2 | `ai.Part{payload: x}` | **compiler** |
| 3 | `ai.Part{x}` | **compiler** |
| 4 | `type mine struct{ ai.Part }` offered as content | **compiler** |
| 5 | `mine{}.Part` | **validation** — item 1 above |
| 6 | implementing `partPayload` | **compiler** |
| 7 | reflection: writing the field | **compiler** (statically) and **`reflect.CanSet`** (dynamically) |
| 8 | copying a valid part and mutating it | **nothing to mutate** — `Part` is a value, `textPayload` holds a `string` |

**How the compiler half is asserted.** Not as prose, and not as a `//go:build ignore` file that nothing checks. Two scratch programs under `src/ai/testdata/`, built by the test with `go build -o os.DevNull`:

- `testdata/handrolled/main.go` — attempts every compiler-answered bypass, each in its own declaration so one refusal cannot mask another. **Must not build**, and the test requires each of six named diagnostics.
- `testdata/constructed/main.go` — the **control**. Same package, same import path, same directory, written through `ai.NewText`. **Must build.** Without it, "this does not compile" is satisfied by a misspelled import, an unresolvable module, or a broken `package ai`.

`testdata` is excluded from package patterns by the go tool and by golangci-lint, so neither program is in `make build`, `make test`'s package set, or `make lint`. Shelling out to the toolchain is precedented: AI-00's import guard runs `go list -deps -test` for the same reason — a property of the build graph is not observable from inside the test binary.

**RED.** The red state for this item is *the seal being absent*, which is simulated by replacing `testdata/handrolled/main.go` with a legal program — the only way those bypasses can be written in a world where `Part` exports its construction surface. (`design.md` § 7 had planned to take this red before `Message` held `[]Part`; commit `2a09c3c` already landed that, so the seal-absent state had to be simulated at the fixture instead. This is the substitute, and it is recorded as such.) The control subtest passes in the same run, which is what makes the failure mean something:

```
--- FAIL: TestPart_HandRolledFromAnotherPackage_DoesNotCompile (0.00s)
    --- FAIL: TestPart_HandRolledFromAnotherPackage_DoesNotCompile/the_hand-rolled_program_does_not_build (0.10s)
        content_part_test.go:600: testdata/handrolled compiled, want a build failure.
            Every bypass in decision.md § 3.3 that the compiler is supposed to answer is now open.
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.555s
```

**Minimal code:** the real `testdata/handrolled/main.go`.

**GREEN**

```
--- PASS: TestPart_ExportedSurface_ExposesNoConstructibleState (0.00s)
    --- PASS: …/the_struct_declares_exactly_one_field,_and_it_is_unexported (0.00s)
    --- PASS: …/reflection_cannot_write_the_payload (0.00s)
    --- PASS: …/a_Part_value_carries_no_exported_method_that_sets_anything (0.00s)
--- PASS: TestPart_HandRolledFromAnotherPackage_DoesNotCompile (0.00s)
    --- PASS: TestPart_HandRolledFromAnotherPackage_DoesNotCompile/the_hand-rolled_program_does_not_build (0.04s)
    --- PASS: TestPart_HandRolledFromAnotherPackage_DoesNotCompile/the_control_program_builds (0.09s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	0.374s
```

**The six diagnostics the guard requires**, as the toolchain produced them:

```
# github.com/cachicamas/backend/agent/src/ai/testdata/handrolled
testdata/handrolled/main.go:12:46: cannot refer to unexported field payload in struct literal of type ai.Part
testdata/handrolled/main.go:13:51: implicit assignment to unexported field payload in struct literal of type ai.Part
testdata/handrolled/main.go:14:78: cannot use embedsPart{} (value of struct type embedsPart) as ai.Part value in argument to ai.NewMessage
testdata/handrolled/main.go:15:35: name partPayload not exported by package ai
testdata/handrolled/main.go:15:49: cannot use myPayload{} (value of struct type myPayload) as ai.partPayload value in variable declaration: myPayload does not implement ai.partPayload (unexported method kind)
testdata/handrolled/main.go:16:41: p.payload undefined (cannot refer to unexported field payload)
```

The fifth line is worth reading twice. `myPayload` implements **both** of `partPayload`'s methods, with the right names and the right signatures, and still does not implement it — an unexported method belongs to the package that declared it. Bypass 6 is not merely inconvenient from outside the package; it is impossible.

The test matches on the message substrings, not the line numbers, so the fixture may move.

---

### Item 3 — the request path, pinned rather than speculated

**The choice, stated plainly as required.** AI-10 does not exist and this milestone has no charter to build it. Of the two routes `proposal.md` set out, this change takes the first:

> **Taken — express the rule as something AI-10 reuses, and pin the reuse point.** The element rule lives in `validateContent(prefix Path, content []Part) *Violation`, whose `prefix` parameter exists for exactly one caller that has not arrived. `NewMessage` passes none, so a failure renders as `content[0]`. AI-10 will pass `AtIndex("messages", i)`, so the same failure renders as `messages[2].content[0]`. `decision.md` § 7.2 — "one rule set, two callers" — anticipated this, and § 12 explicitly leaves *where* AI-10 calls it from to AI-10.
>
> **Not taken — revert-and-record.** That clause fires when a leaf's first red test cannot be driven green in small steps. This one can: AI-10 and AI-06 live in the same package, so "the request path" is not a missing seam, it is a caller that has not arrived. Inventing a `Request` type to satisfy the letter of the item would freeze a shape AI-10 has four leaves to decide and would break the milestone rule of one contract per milestone. **No doc 0002 amendment is proposed.**

**Why this is checkable and not a promise.** `src/ai/content_part_internal_test.go` calls `validateContent` with request-shaped prefixes and asserts the rendered position. It fails the day the prefix stops composing — which is the day AI-10 would otherwise discover, at its own boundary, that it needs a second copy of the element rule.

**RED.** Taken by making `validateContent` ignore its prefix — the pre-item-3 state, in which the reuse point is a parameter nobody honours:

```
--- FAIL: TestValidateContent_RequestShapedPrefix_ReportsTheDeeperPosition (0.00s)
    --- FAIL: …/a_request-shaped_prefix,_an_element_deeper_in_the_sequence (0.00s)
        content_part_internal_test.go:106: validateContent(messages[0], …).Path() = "content[1]", want "messages[0].content[1]"
    --- FAIL: …/the_prefix_composes_with_the_payload's_own_position (0.00s)
        content_part_internal_test.go:106: validateContent(messages[7], …).Path() = "content[0].text", want "messages[7].content[0].text"
    --- FAIL: …/a_request-shaped_prefix_—_what_AI-10_will_report (0.00s)
        content_part_internal_test.go:106: validateContent(messages[2], …).Path() = "content[0]", want "messages[2].content[0]"
    --- FAIL: …/a_prefix_of_more_than_one_step (0.00s)
        content_part_internal_test.go:106: validateContent(request.messages[1], …).Path() = "content[0]", want "request.messages[1].content[0]"
    --- FAIL: …/an_unregistered_payload_kind_makes_every_part_carrying_it_invalid (0.00s)
        content_part_internal_test.go:171: validateContent(…).Path() = "content[0]", want "messages[1].content[0]"
    --- FAIL: …/the_caller's_prefix_is_not_rewritten_by_the_rule_that_extends_it (0.00s)
        content_part_internal_test.go:140: the first call reported "content[0]", want "messages[4].content[0]"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.474s
```

**Minimal code:** `part.validate(under(prefix, AtIndex("content", i)))` — the prefix honoured, composed with `under`, which concatenates rather than appends.

**GREEN**

```
--- PASS: TestValidateContent_RequestShapedPrefix_ReportsTheDeeperPosition (0.00s)
    --- PASS: …/no_prefix,_an_unconstructed_element_—_what_NewMessage_reports_today (0.00s)
    --- PASS: …/the_prefix_composes_with_the_payload's_own_position (0.00s)
    --- PASS: …/a_request-shaped_prefix,_an_element_deeper_in_the_sequence (0.00s)
    --- PASS: …/a_request-shaped_prefix_—_what_AI-10_will_report (0.00s)
    --- PASS: …/a_prefix_of_more_than_one_step (0.00s)
    --- PASS: …/the_caller's_prefix_is_not_rewritten_by_the_rule_that_extends_it (0.00s)
    --- PASS: …/valid_content_reports_nothing,_at_either_depth (0.00s)
    --- PASS: …/no_prefix,_an_element_deeper_in_the_sequence (0.00s)
    --- PASS: …/an_unregistered_payload_kind_makes_every_part_carrying_it_invalid (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	0.310s
```

The pin covers three things beyond the headline: the aliasing hazard `under` exists for (a prefix with spare capacity is called twice, and neither the caller's slice nor the earlier violation is rewritten); that an empty sequence is *not* this rule's failure, because emptiness is the caller's rule; and that a payload whose kind has no table entry makes every part carrying it invalid, which is what keeps `partKindNames` load-bearing rather than decorative.

---

## Phase AI-06.4 — kind registration is exhaustive `[guard]`

**Deliverable:** `src/ai/content_part_registry_test.go`. **No production code.**

### Test list

1. - [x] A mechanical scan asserts every registered kind has all three legs: a constructor with rules, a payload accessor, and a validation path.
2. - [x] The package documentation's list of kinds matches the registration table — an AST scan, not prose.

### Design — and how it avoids passing by tautology

AI-13's critique of an exhaustiveness pin is that *a pin reading the package's own accessor compares the package against itself*. That critique applies here twice over, so the guard trusts **only the text a human wrote**:

- It parses the package's non-test `.go` files with `go/ast` and collects every **exported constant of type `PartKind`**, in declaration order, carrying the type forward across an `iota` run the way the language does. `partKindFirst` and `partKindEnd` state a value without a type, which ends the run — so the two bounds are excluded by the same rule that excludes them from being members.
- It then compares **three** answers against each other: what the source declares, what `PartKinds()` enumerates, and what `partKindNames` registers. Any two disagreeing is a failure, in either direction.

That is what makes the guard bite in the two directions a table-driven or accessor-driven guard cannot: a constant declared **without a table entry**, and a constant declared **without moving `partKindEnd`** — the second of which `PartKinds()` structurally cannot see. Bite 1b below records exactly that.

**The three legs**, per `decision.md` § 8, with the witnesses in the guard's own file and never in production:

| Leg | Step | What the guard requires |
| --- | --- | --- |
| 1 | a constructor with rules | The valid witness builds a part of the kind; the invalid witness is **rejected**, with a `*Violation` naming a **registered** rule class, returning the zero `Part` |
| 2 | a payload accessor | Says yes to a part of its own kind; says no to the zero `Part` and to a part of **every other** registered kind |
| 3 | a validation path | A part assembled **without** its constructor, holding a value its rules reject, fails `part.validate` **and** `NewMessage` — while a properly constructed part still passes, so the leg is not satisfied by a boundary that rejects everything |

Leg 2's "every other kind" loop is empty today and populates itself when AI-07 lands, without an edit.

### Item 1 — RED

The witness table starts empty; the source scan does not:

```
--- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.00s)
    --- FAIL: …/every_declared_constant_has_a_witness,_and_every_witness_a_constant (0.00s)
        content_part_registry_test.go:122: PartKindText is declared in the source but has no entry in partKindWitnesses.
              Adding a kind is the five-step procedure in decision.md § 8, and this guard is what fails when a step is missing.
              Add a witness with all three legs: a constructor with rules, a payload accessor, and a validation path.
    --- FAIL: …/the_declared_constants,_PartKinds()_and_partKindNames_agree (0.00s)
        content_part_registry_test.go:145: PartKinds() enumerates 1, which has no witness
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.472s
```

### Items 1 and 2 — GREEN

```
--- PASS: TestPartKindDocumentation_MatchesTheRegistrationTable (0.00s)
--- PASS: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.00s)
    --- PASS: …/every_declared_constant_has_a_witness,_and_every_witness_a_constant (0.00s)
    --- PASS: …/the_declared_constants,_PartKinds()_and_partKindNames_agree (0.00s)
    --- PASS: …/PartKindText (0.00s)
        --- PASS: …/PartKindText/leg_1_—_a_constructor,_with_rules (0.00s)
        --- PASS: …/PartKindText/leg_2_—_a_payload_accessor (0.00s)
        --- PASS: …/PartKindText/leg_3_—_a_validation_path (0.00s)
```

---

## Bite proofs — a guard leaf does not close on green

`decision.md` § 8 requires the guard to be recorded failing **twice**: once for a scratch kind whose witness is missing a leg, once for a documentation list that drifted from the table. Both are below, verbatim, plus a third that is not required but is the one that answers AI-13's tautology objection directly. **Every scratch was dropped**; the tree at commit `bde88d2` contains none of them, and `git diff` against it is empty.

### Bite 1 — a scratch kind whose witness is missing a leg

`PartKindScratch` was declared in `content_part.go` (constant, `partKindEnd` moved past it, `partKindNames` entry, GoDoc line — four of the five steps done correctly), and given a witness with `read` left nil. Because the other four steps were done, the guard fails on **exactly one** thing and nothing else:

```
--- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.00s)
    --- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath/PartKindScratch (0.00s)
        content_part_registry_test.go:212: PartKindScratch has no leg 2: step 4 of the procedure is an exported accessor func (p Part) Scratch() (T, bool)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.293s
```

The failure names the missing step and the signature the kind's author has to write. **Scratch dropped.**

### Bite 1b — the tautology check: a constant `PartKinds()` cannot see

Not required by `decision.md`, recorded because the milestone instruction asks that the guard "genuinely cannot pass by tautology". The same `PartKindScratch` constant, this time with `partKindEnd` **left where it was** and no table entry and no witness — the state in which the package's own enumeration is blind to it:

```
--- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.00s)
    --- FAIL: …/every_declared_constant_has_a_witness,_and_every_witness_a_constant (0.00s)
        content_part_registry_test.go:143: PartKindScratch is declared in the source but has no entry in partKindWitnesses.
              Adding a kind is the five-step procedure in decision.md § 8, and this guard is what fails when a step is missing.
              Add a witness with all three legs: a constructor with rules, a payload accessor, and a validation path.
    --- FAIL: …/the_declared_constants,_PartKinds()_and_partKindNames_agree (0.00s)
        content_part_registry_test.go:159: PartKinds() enumerates 1 kinds but the source declares 2 ([PartKindText PartKindScratch]).
              A constant declared without moving partKindEnd past it is invisible to every assertion over PartKinds().
        content_part_registry_test.go:187: partKindNames holds 1 entries but the source declares 2 constants ([PartKindText PartKindScratch])
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.297s
```

A guard built on `PartKinds()` or on `partKindNames` would have been **green** in this state. **Scratch dropped.**

### Bite 2 — a documentation list that drifted from the table

Recorded in **both directions**, because the guard asserts both.

**2a — a documented kind the table lacks.** A line for `image` added to `PartKind`'s GoDoc list and nowhere else:

```
--- FAIL: TestPartKindDocumentation_MatchesTheRegistrationTable (0.00s)
    content_part_registry_test.go:342: the PartKind GoDoc lists "image", which partKindNames does not register.
          documented: [text image]
          tabulated:  [text]
          A documented kind with no table entry is not a member of the vocabulary: every part carrying it would be invalid.
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.303s
```

This is the case that matters most in practice: `image` and `audio` are named in the content vocabulary and deliberately have no producer (V-PRV-07, doc 0001 § 8), so a documented list that quietly gained one would be a lie a consumer trusts.

**2b — the documented name drifted from the tabulated one.** The `text` line renamed to `prose`:

```
--- FAIL: TestPartKindDocumentation_MatchesTheRegistrationTable (0.00s)
    content_part_registry_test.go:335: partKindNames registers "text", which the PartKind GoDoc does not list.
          documented: [prose]
          tabulated:  [text]
          Step 5 of decision.md § 8 is one step: the table entry and the documented line land together.
    content_part_registry_test.go:342: the PartKind GoDoc lists "prose", which partKindNames does not register.
          documented: [prose]
          tabulated:  [text]
          A documented kind with no table entry is not a member of the vocabulary: every part carrying it would be invalid.
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.277s
```

Both directions fire from one edit, which is the correct behaviour: a rename is simultaneously an addition and a removal. **Drift reverted.**

---

## Evidence gate

### `make test` — `go test -race -v ./...`, cache cleared

```
    --- PASS: TestNewText_RuleViolations_FailWithTheDocumentedSentinels/non-ASCII_whitespace (0.00s)
    --- PASS: TestNewText_RuleViolations_FailWithTheDocumentedSentinels/tabs_and_newlines (0.00s)
    --- PASS: TestNewText_RuleViolations_FailWithTheDocumentedSentinels/a_single_space (0.00s)
    --- PASS: TestNewText_RuleViolations_FailWithTheDocumentedSentinels/the_failure_never_reproduces_the_text_it_was_given (0.00s)
    --- PASS: TestNewText_RuleViolations_FailWithTheDocumentedSentinels/the_documented_bound_is_exact_and_exported (0.01s)
    --- PASS: TestNewText_RuleViolations_FailWithTheDocumentedSentinels/both_rules_violated_reports_the_first_in_the_documented_order (0.82s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	2.317s
```

Both packages, non-verbose: 31 top-level tests, **0 failures**.

```
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.269s
ok  	github.com/cachicamas/backend/agent/src/ai	2.332s
```

### `make lint` — `go vet` then golangci-lint v2.9.0

```
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

### AI-00's guards, and the zero-requires rule

```
--- PASS: TestLayer1Package_IsImportableFromAnExternalPackage_Compiles (0.00s)
--- PASS: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault (0.06s)
--- PASS: TestLayer1_ModuleHasNoDependencies_ZeroRequires (0.06s)
```

`go.mod` is unchanged and carries **zero requires**. Everything this milestone added is standard library: `slices`, `strconv`, `strings` in production; `errors`, `fmt`, `os`, `os/exec`, `reflect`, `go/ast`, `go/doc/comment`, `go/parser`, `go/token`, `path/filepath` in tests.

---

## Commits

| # | SHA | Subject | Node |
| --- | --- | --- | --- |
| 1 | `2ae3be9` | `docs(sdd): decide one content-part strategy, readable and sealed (AI-06.1)` | AI-06.1 |
| 2 | `2a09c3c` | `refactor(ai): close the content seam on a concrete part type (AI-06.1)` | AI-06.1 |
| 3 | `6a88fb1` | `feat(ai): add text content, constructible and readable (AI-06.2)` | AI-06.2 |
| 4 | `17f952c` | `feat(ai): reject unconstructed content parts at the message boundary (AI-06.3)` | AI-06.3 item 1 |
| 5 | `79d7218` | `test(ai): prove the part contract is unimplementable and its rule reusable (AI-06.3)` | AI-06.3 items 2–4 |
| 6 | `bde88d2` | `test(ai): guard that every declared part kind is exhaustively registered (AI-06.4)` | AI-06.4 |
| 7 | *this file* | `docs(sdd): record the AI-06 task list with its red-green evidence` | — |

Six of the seven fall on node boundaries, so the PR is reviewable in passes or chainable without rework. Commit 4 also carries the two lint fixes described under AI-06.2, because leaving `make lint` red would have made the evidence gate unmeetable for every commit after it.

---

## Register amendment

**None.** `decision.md` § 11 records why: `V-REQ-04` … `V-REQ-08` already define every noun used, including both constitutive properties of a content part and the closed, exhaustively registered kind set, and § 7.3 records why no rule class is appended to AI-04's five. The two candidate terms considered and rejected were *"opaque value type"* (a Go implementation shape, which the register's own scope rule excludes) and *"kind registration"* (already carried by `V-REQ-05`'s "closed and exhaustively registered").

`openspec/specs/ai-contract-vocabulary/spec.md` is **not modified by this change**.

---

## What the next milestones inherit

| Milestone | What it can now cite instead of re-deriving |
| --- | --- |
| AI-07 (reasoning) | The five-step procedure, the guard that fails when a step is missing, and the one-file-per-kind layout. Adding a kind is one new file plus three lines in `content_part.go` |
| AI-09 (tool call / tool result) | The same, plus `decision.md` § 6.3's rule for a payload that carries structure rather than a string |
| AI-10 (the request) | `validateContent(prefix, content)` with its position contract already proven at depth. § 12 leaves *where* AI-10 calls it from to AI-10 |
| AI-26 (request translation) | The exact loop it will write, executed today from `src/agenttest` and again as `testdata/constructed/main.go` |
