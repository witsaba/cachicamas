# Tasks — reasoning content and its round-trip token

> **Change**: `cachicamas-ai-reasoning-content`
> **Milestone**: AI-07 · **Nodes**: AI-07.1 `[leaf]`, AI-07.2 `[leaf]`, AI-07.3 `[leaf]`, AI-07.4 `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-reasoning-content/spec.md`, `design.md`, and [AI-06's `decision.md`](../2026-08-01-cachicamas-ai-content-parts/decision.md) inherited whole
> **Branch**: `feat/ai-07-reasoning-content`
> **Depends on**: AI-06 merged
> **Blocks**: AI-17, AI-29
> **Evidence gate**: recorded green `make test` and `make lint` in `backend/agent/` (`go test -race -v ./...`)

---

## Node types and what they mean for this list

| Node | Type | Closes on |
| --- | --- | --- |
| AI-07.1 | `[leaf]` | Three test-list items taken red → green → refactored, **in order**, plus one case discovered mid-leaf |
| AI-07.2 | `[leaf]` | Two items, plus one appended case for the rule order |
| AI-07.3 | `[leaf]` | Two items — one a declared **pin**, one a real aliasing bug — plus one appended case |
| AI-07.4 | `[leaf]` | Three items — one a re-observed red, two declared **pins** — plus one appended case |

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so every red below follows AI-05's rule as AI-06 inherited it: **a compile error is the state before red, not red.** Where that rule forced a stub, `design.md` § 7 named the stub in advance and the transcript below is of that stub failing.

### Provenance — every transcript here was observed

Unlike AI-06, this milestone ran in one session and **no transcript below is reconstructed.** Every red and green block is pasted verbatim from the run that produced it. Two reds are marked **re-observed**: the item's implementation was already present because an earlier item's minimal step had landed it, so the implementation was reverted to `design.md` § 7's named stub, the failure recorded, and the implementation restored. Both are labelled at the point of use, and neither is presented as a first-time red.

---

## Review Workload Forecast

Recorded with the forecast that preceded it, because the difference is the point.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `specs/ai-reasoning-content/spec.md`, `design.md`, `tasks.md` | ~700 prose | **~740 prose + this file** | Low | 25 min |
| The kind | `reasoning_content.go` | ~200 Go | **374 Go (107 non-comment)** | **Medium** — a new payload shape that AI-17 and AI-29 inherit | 30 min |
| The registration | `content_part.go`, `content_part_registry_test.go`, `content_part_test.go` | ~30 Go | **50 Go added / 2 removed** | Low — three appends and a witness | 10 min |
| The proofs | `reasoning_content_test.go` | ~400 Go | **1214 Go (915 non-comment)** | **Medium** — byte-class properties over a value nothing may touch | 45 min |
| **Total** | 5 files (1 new Go, 3 modified Go, 1 new test, 5 new markdown) | ~630 Go, ~700 prose | **1636 Go added / 2 removed**, ~740 prose + this file | **Medium** | **~110 min** |

### Budget reassessment — trigger 4 fired, and `explore.md` § 5 forecast it

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when the projected diff pushes the milestone past the review budget. **It did.** The reassessment, rather than a silent overrun or a cut test list:

- **Production is 107 non-comment lines in one file, plus 12 changed lines in another.** The surface AI-17 and AI-29 inherit is small; two thirds of `reasoning_content.go` is the reasoning behind it, put in front of the person who adds the third kind.
- **The weight is in the tests, and it is structural.** Four of the twelve test-list items are byte-class properties over a value this package promises never to touch, and a property of that kind is proven by enumerating classes — the fixture table alone covers ten. The alternative is asserting the property once on ASCII, which is how a token that survives `"abc"` and dies on an embedded NUL ships.
- **Split trigger 1 does not fire, and it is the one that would matter.** The four leaves are one contract: the token has no meaning without the state it hangs on, and the state has no purpose without the token that makes "no reasoning text" worth recording. AI-07.3 and AI-07.4 are declared parallel by doc 0002 precisely because they are two proofs of one landed shape, not two shapes.
- **Split triggers 2, 3 and 5 do not fire.** Every leaf went green-to-green in one sitting; no leaf needed a seam that did not exist — AI-06 built them all; and the leaves are strictly ordered, so no two agents could have worked them concurrently.
- **The chain boundary is already cut.** Five commits on node boundaries; the pull request is reviewable in five passes in commit order.
- **Nothing was cut to fit.** Four cases were appended during implementation and all four stayed. Two of them found real defects.

---

## Phase AI-07.1 — reasoning as a content part `[leaf]`

**Deliverable:** `src/ai/reasoning_content.go`, the three appends to `content_part.go`, and the witness in AI-06.4's guard.

### Test list

1. - [x] A reasoning part is constructed and read back through the AI-06.1 strategy, exactly like text — no second strategy, no special case.
2. - [x] Reasoning and text are structurally distinct: no accessor yields reasoning content as text, and a consumer switching on kind cannot conflate them.
3. - [x] The reasoning state vocabulary is closed and each state is constructible: with text, redacted, and token-only.
4. - [x] *(appended)* AI-06.2's by-hand exhaustiveness pin needs a branch for the new kind.

---

### Item 1 — a reasoning part, through AI-06's strategy

**The stub taken red**, per `design.md` § 7: `PartKindReasoning` declared with its table entry and its documented line, `Reasoning` with no fields, `NewReasoning` returning the zero `Part`.

**RED.** Two failures in one run, and the second is the point: declaring a kind constant makes **AI-06.4's guard bite immediately**, before any behavior is written, which is exactly what a guard over a five-step procedure is for.

```
--- FAIL: TestReasoning_ConstructedAndReadBack_UsesTheContentPartStrategy (0.00s)
    reasoning_content_test.go:34: part.Kind() = unset, want reasoning
--- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.01s)
    --- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath/every_declared_constant_has_a_witness,_and_every_witness_a_constant (0.00s)
        content_part_registry_test.go:143: PartKindReasoning is declared in the source but has no entry in partKindWitnesses.
              Adding a kind is the five-step procedure in decision.md § 8, and this guard is what fails when a step is missing.
              Add a witness with all three legs: a constructor with rules, a payload accessor, and a validation path.
    --- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath/the_declared_constants,_PartKinds()_and_partKindNames_agree (0.00s)
        content_part_registry_test.go:166: PartKinds() enumerates 2, which has no witness
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.485s
```

**Minimal code:** `Reasoning{text string}` with `kind()`, one rule (`ErrEmpty` at `text` for text carrying no non-whitespace character), `NewReasoning` storing the text, `Part.Reasoning()`, and the guard witness with all three legs.

**GREEN**

```
--- PASS: TestReasoning_ConstructedAndReadBack_UsesTheContentPartStrategy (0.00s)
--- PASS: TestPartKindDocumentation_MatchesTheRegistrationTable (0.00s)
--- PASS: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.01s)
    --- PASS: …/every_declared_constant_has_a_witness,_and_every_witness_a_constant (0.00s)
    --- PASS: …/the_declared_constants,_PartKinds()_and_partKindNames_agree (0.00s)
    --- PASS: …/PartKindReasoning (0.00s)
        --- PASS: …/PartKindReasoning/leg_1_—_a_constructor,_with_rules (0.00s)
        --- PASS: …/PartKindReasoning/leg_2_—_a_payload_accessor (0.00s)
        --- PASS: …/PartKindReasoning/leg_3_—_a_validation_path (0.00s)
    --- PASS: …/PartKindText (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.344s
```

**What it establishes.** The loop AI-26 will write is the identical loop for reasoning and for text, with one identifier changed. And AI-06.4's guard, written one milestone ago against a vocabulary of one, bit correctly on the first kind that was ever added after it.

---

### Item 2 — reasoning and text are structurally distinct

**RED — re-observed.** `Part.Reasoning` was written with its type assertion under item 1's pressure, so the failure state had to be recreated: the accessor was reverted to `design.md` § 7's named stub — `return payload, true`, unconditional — which item 1 alone accepts and item 2 does not.

```
--- FAIL: TestReasoning_AndText_AreStructurallyDistinct (0.00s)
    --- FAIL: TestReasoning_AndText_AreStructurallyDistinct/a_part_that_was_never_constructed_yields_neither (0.00s)
        reasoning_content_test.go:111: the zero Part reported reasoning
    --- FAIL: TestReasoning_AndText_AreStructurallyDistinct/no_accessor_yields_text_content_as_reasoning (0.00s)
        reasoning_content_test.go:98: textPart.Reasoning() reported reasoning on a text part
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.338s
```

**Minimal code:** the type assertion restored.

**GREEN**

```
--- PASS: TestReasoning_AndText_AreStructurallyDistinct (0.00s)
    --- PASS: …/no_accessor_yields_reasoning_content_as_text (0.00s)
    --- PASS: …/no_accessor_yields_text_content_as_reasoning (0.00s)
    --- PASS: …/a_part_that_was_never_constructed_yields_neither (0.00s)
    --- PASS: …/a_consumer_switching_on_kind_cannot_conflate_them (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.330s
```

**What it establishes.** The reasoning part and the text part are built from the **identical string**, so nothing passes because the payloads happen to differ. `V-REQ-09`'s "distinct from text content at every layer" is a property of the accessors, not of the data.

---

### Item 3 — the state vocabulary, closed and wholly constructible

**The stub taken red**: the three constants, the table and `ReasoningStates()` landed mechanically from `role.go`'s pattern; `State()` returning `ReasoningStateText` unconditionally; `NewRedactedReasoning` delegating to `NewReasoning("", token)`.

**RED**

```
--- FAIL: TestReasoningStates_Vocabulary_IsClosedStableAndConstructible (0.00s)
    --- FAIL: …/each_state_is_constructible_and_reports_itself (0.00s)
        --- FAIL: …/each_state_is_constructible_and_reports_itself/token-only (0.00s)
            reasoning_content_test.go:193: construction returned text: required value is empty, want no failure
        --- FAIL: …/each_state_is_constructible_and_reports_itself/redacted (0.00s)
            reasoning_content_test.go:193: construction returned text: required value is empty, want no failure
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.341s
```

**Minimal code:** `Reasoning` gains `token`, `hasToken` and `redacted`; `State()` becomes derived; `NewRedactedReasoning` sets `redacted`; the empty-text rule is narrowed to the text state and joined by "a state that carries no text carries a token".

**GREEN**

```
--- PASS: TestReasoningStates_Vocabulary_IsClosedStableAndConstructible (0.00s)
    --- PASS: …/each_state_is_constructible_and_reports_itself (0.00s)
        --- PASS: …/each_state_is_constructible_and_reports_itself/with_text (0.00s)
        --- PASS: …/each_state_is_constructible_and_reports_itself/redacted (0.00s)
        --- PASS: …/each_state_is_constructible_and_reports_itself/token-only (0.00s)
    --- PASS: …/the_vocabulary_is_closed,_stable_and_not_rewritable_by_a_consumer (0.00s)
    --- PASS: …/the_zero_value_is_not_a_member_and_renders_as_unset (0.00s)
    --- PASS: …/a_member_renders_as_its_registered_name_and_a_non-member_identifiably (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.336s
```

**The vocabulary is three, and the register names four shapes.** `V-REQ-10` lists "full text, redacted, signature-only, or a provider that emitted no reasoning text at all"; doc 0002's AI-07.1 item 3 asks for three. They agree, and AI-07.4 item 2 is the sentence that reconciles them: *"a reasoning part with a token but no text … the signature-only shape, **and** the 'this provider emitted no reasoning text' state"* — one shape, two vendors' words for it. `ReasoningStateTokenOnly` is named for what it carries rather than for either word.

**Refactor.** The file banner and the vocabulary's GoDoc, written terse during the stub phase, were filled in: why the token is correctness rather than metadata, why the milestone invents nothing, and why the state is derived.

---

### Item 4 — *(appended)* AI-06.2's by-hand pin needs a branch

**Discovered** when the full suite ran after item 3. `TestPart_KindAndPayload_Agree` exercises every registered kind by hand and fails on any kind it does not know:

```
--- FAIL: TestPart_KindAndPayload_Agree (0.00s)
    --- FAIL: TestPart_KindAndPayload_Agree/every_registered_kind_reports_itself_and_yields_its_payload (0.00s)
        content_part_test.go:54: kind reasoning is registered but this test does not exercise it — AI-06.4 mechanizes what this default catches by hand
```

**This is the sixth step of a five-step procedure**, and it is worth recording as such: `decision.md` § 8 lists five, and AI-06's own by-hand pin adds a sixth that the procedure does not mention. `content_part_test.go` is otherwise untouched by this change; the branch appended to it mirrors the text branch exactly. **AI-09 will hit the identical failure**, and the two appends land in the same `switch` — an append at the end of a switch, which is the merge shape every other edit in this change also takes.

**GREEN:** full suite clean; `make lint` 0 issues.

---

## Phase AI-07.2 — opaque token storage `[leaf]`

**Deliverable:** `Reasoning.Token`, `MaxReasoningTokenLen`, and the rules.

### Test list

1. - [x] A reasoning part can carry an opaque provider token alongside its state and text; **absence is distinguishable from an empty token**.
2. - [x] Nothing in the package interprets, validates, normalizes or length-caps the token beyond a documented sanity bound — proven with tokens that are not valid UTF-8, not valid JSON, and not printable.
3. - [x] *(appended)* The construction rules fail through AI-04's sentinels, in the documented order.

---

### Item 1 — absent is not empty

**The stub taken red**, per `design.md` § 7 — the collapse made concrete: `Token()` inferring presence from `len(token) > 0`.

**RED**

```
--- FAIL: TestReasoning_AbsentToken_IsDistinguishableFromAnEmptyToken (0.00s)
    --- FAIL: …/a_redacted_part_whose_payload_is_present_and_zero_bytes_long (0.00s)
        reasoning_content_test.go:338: as constructed: reasoning.Token() reported present = false, want true
        reasoning_content_test.go:338: read back out of a message: reasoning.Token() reported present = false, want true
    --- FAIL: …/a_token_that_is_present_and_zero_bytes_long (0.00s)
        reasoning_content_test.go:338: as constructed: reasoning.Token() reported present = false, want true
        reasoning_content_test.go:338: read back out of a message: reasoning.Token() reported present = false, want true
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.489s
```

**Minimal code:** `Token()` returns the stored `hasToken` rather than a length test.

**GREEN**

```
--- PASS: TestReasoning_AbsentToken_IsDistinguishableFromAnEmptyToken (0.00s)
    --- PASS: …/no_token_at_all (0.00s)
    --- PASS: …/a_token_that_is_present_and_zero_bytes_long (0.00s)
    --- PASS: …/a_token_with_bytes_in_it (0.00s)
    --- PASS: …/a_redacted_part_whose_payload_is_present_and_zero_bytes_long (0.00s)
    --- PASS: …/a_part_with_neither_text_nor_token_carries_nothing_and_is_rejected (0.00s)
        --- PASS: …/no_text_and_no_token (0.00s)
        --- PASS: …/redacted_with_no_payload_to_replay (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.315s
```

**How absence is represented, and why.** Presence is a `bool` stored beside the bytes; `nil` is only the spelling at the door. The distinction is asserted twice per case — as constructed and after a message round trip — because the failure mode is a collapse somewhere on the path, not at the door.

---

### Item 2 — nothing interprets the token

The declaration of `MaxReasoningTokenLen` was needed for the test to compile, which is the state before red:

```
src/ai/reasoning_content_test.go:491:9: undefined: ai.MaxReasoningTokenLen
FAIL	github.com/cachicamas/backend/agent/src/ai [build failed]
```

**The stub taken red**, per `design.md` § 7: the constant declared, the bound rule absent.

**RED**

```
--- FAIL: TestReasoning_OpaqueToken_IsNeitherInterpretedNorNormalized (0.00s)
    --- FAIL: …/the_documented_sanity_bound_is_exact_and_exported (0.00s)
        reasoning_content_test.go:505: a token one byte over the bound returned no failure, want one
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.422s
```

**Minimal code:** the bound rule, `ErrOutOfRange` at `token`.

**GREEN** — ten byte classes, plus the encoding and bound subtests:

```
--- PASS: TestReasoning_OpaqueToken_IsNeitherInterpretedNorNormalized (0.00s)
    --- PASS: …/every_byte_class_survives_construction_and_readback_unaltered (0.00s)
        --- PASS: …/every_byte_value_from_0x00_to_0xff (0.00s)
        --- PASS: …/not_well-formed_UTF-8 (0.00s)
        --- PASS: …/a_lone_UTF-16_surrogate_half,_encoded_as_bytes (0.00s)
        --- PASS: …/not_valid_JSON (0.00s)
        --- PASS: …/not_printable (0.00s)
        --- PASS: …/an_embedded_zero_byte_between_printable_bytes (0.00s)
        --- PASS: …/bidirectional_control_characters,_built_from_escapes (0.00s)
        --- PASS: …/something_that_looks_like_base64_and_must_not_be_decoded (0.00s)
        --- PASS: …/something_that_looks_like_a_rendered_violation (0.00s)
        --- PASS: …/a_single_zero_byte (0.00s)
    --- PASS: …/a_token_that_is_not_well-formed_UTF-8_is_stored,_not_repaired (0.00s)
    --- PASS: …/the_documented_sanity_bound_is_exact_and_exported (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.317s
```

**Lint bit here, and it bit correctly.** The bidi fixture was written with **literal** control characters despite being named "built from escapes", and staticcheck caught it:

```
src/ai/reasoning_content_test.go:414:67: ST1018: string literal contains Unicode format characters, consider using escape sequences instead (staticcheck)
		{"bidirectional control characters, built from escapes", []byte("start<U+202E>reversed<U+202C>end")},
1 issues:
* staticcheck: 1
```

Rewritten so the control characters are `\u202E` and `\u202C` escapes rather than literal bytes. This is the Trojan Source hazard, and a token fixture is exactly where one would hide.

---

### Item 3 — *(appended)* the documented rule order

**Appended while writing item 2**: the bound on the *token* had a home and the bound on the reasoning *text* had none, and the ordering between the four rules is contract under `V-FAIL-04`.

**RED**

```
--- FAIL: TestNewReasoning_RuleViolations_FailWithTheDocumentedSentinels (0.00s)
    --- FAIL: …/reasoning_text_one_byte_over_the_documented_bound (0.00s)
        reasoning_content_test.go:592: construction returned no failure, want one
    --- FAIL: …/the_failure_never_reproduces_the_reasoning_it_was_given (0.00s)
        reasoning_content_test.go:626: construction returned no failure, want one
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.313s
```

**Minimal code:** the text bound, `ErrOutOfRange` at `text`, against the package's existing `MaxTextLen`.

**GREEN**

```
--- PASS: TestNewReasoning_RuleViolations_FailWithTheDocumentedSentinels (0.00s)
    --- PASS: …/reasoning_text_of_only_whitespace (0.00s)
    --- PASS: …/reasoning_text_of_a_single_space,_with_no_token_either (0.00s)
    --- PASS: …/reasoning_text_one_byte_over_the_documented_bound (0.00s)
    --- PASS: …/no_text_and_no_token (0.00s)
    --- PASS: …/a_token_one_byte_over_the_documented_bound (0.00s)
    --- PASS: …/text_and_token_both_broken_reports_the_first_in_the_documented_order (0.00s)
    --- PASS: …/the_failure_never_reproduces_the_reasoning_it_was_given (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.312s
```

**No new rule class, and the one that was nearly needed.** Every rule reports through `ErrEmpty` or `ErrOutOfRange`. The violation that would have needed a sixth — a state contradicting the payload it describes — is designed out by deriving the state rather than reported. `validation.go` is untouched. `design.md` § 4.4 states plainly what the honest response would have been otherwise: **append a class in this pull request**, not stretch `ErrNotInVocabulary` (the state *is* in the vocabulary) or `ErrMalformed` (nothing is malformed for its encoding).

---

## Phase AI-07.3 — byte-exact round trip `[leaf]`

### Test list

1. - [x] *(pin)* A reasoning part with a token, placed in a message, read back and re-attached, keeps the token **byte-identical** — every byte class, including a token longer than any plausible buffer boundary.
2. - [x] The property survives AI-05.3's copy semantics.
3. - [x] *(appended)* A reasoning part compares with `==` like every other part.

---

### Item 1 — the two-hop round trip *(pin)*

**Green from birth, declared as a pin** under doc 0002's leaf anatomy: a regression assertion over a property AI-07.2's storage established. `design.md` § 7 predicted this and it is recorded rather than dressed up as a red. The trip is deliberately **two hops** — out of one message and into another — because one hop would pass for an implementation that returned the caller's own slice.

```
--- PASS: TestReasoning_TokenThroughAMessage_RoundTripsByteIdentical (0.00s)
    --- PASS: …/every_byte_value_from_0x00_to_0xff (0.00s)
    --- PASS: …/not_well-formed_UTF-8 (0.00s)
    --- PASS: …/a_lone_UTF-16_surrogate_half,_encoded_as_bytes (0.00s)
    --- PASS: …/not_valid_JSON (0.00s)
    --- PASS: …/not_printable (0.00s)
    --- PASS: …/an_embedded_zero_byte_between_printable_bytes (0.00s)
    --- PASS: …/bidirectional_control_characters,_built_from_escapes (0.00s)
    --- PASS: …/something_that_looks_like_base64_and_must_not_be_decoded (0.00s)
    --- PASS: …/something_that_looks_like_a_rendered_violation (0.00s)
    --- PASS: …/a_single_zero_byte (0.00s)
    --- PASS: …/high_Unicode_encoded_as_bytes (0.00s)
    --- PASS: …/longer_than_any_plausible_buffer_boundary (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.308s
```

Byte classes covered: **binary (all 256 values), high Unicode encoded as bytes, an embedded NUL, a lone surrogate half, malformed UTF-8, non-printable control bytes, bidi controls, invalid JSON, base64-shaped bytes that must not be decoded, and 139 264 bytes** — past 4 KiB, 32 KiB, 64 KiB and 128 KiB boundaries. Each is asserted at three points: as constructed, out of the first message, and out of the second.

---

### Item 2 — the copy semantics, and a real aliasing bug

**RED.** Four subtests failed and the race detector fired three times. This is not a hypothetical: the stored slice shares its backing array with the caller's buffer, and the accessor hands that array straight back out.

```
WARNING: DATA RACE
Read at 0x00c000298000 by goroutine 15:
  bytes.Equal()
  …ai_test.TestReasoning_TokenAcrossCopies_IsUnaffectedByTheCallersSlice.func4.1()
      reasoning_content_test.go:831
Previous write at 0x00c000298004 by goroutine 18:
  …ai_test.TestReasoning_TokenAcrossCopies_IsUnaffectedByTheCallersSlice.func4.1()
      reasoning_content_test.go:835
```

```
--- FAIL: TestReasoning_TokenAcrossCopies_IsUnaffectedByTheCallersSlice (0.00s)
    --- FAIL: …/mutating_the_slice_the_caller_supplied_does_not_change_the_token (0.00s)
        reasoning_content_test.go:759: the token became 2a2a…2a after the caller overwrote its own slice, want 00ff7468652070726f76696465722773207369676e6174757265fe01
    --- FAIL: …/mutating_the_bytes_an_accessor_returned_does_not_change_the_token (0.00s)
        reasoning_content_test.go:779: the token became 2a2a…2a after a consumer overwrote the bytes it received, want 00ff…fe01
    --- FAIL: …/copying_a_message_copies_the_token_exactly (0.00s)
        reasoning_content_test.go:807: the copy's token is 2a2a…2a, want 00ff…fe01 — two holders of a copy observed each other
    --- FAIL: …/two_consumers_reading_in_parallel_observe_the_same_bytes (0.00s)
        reasoning_content_test.go:832: a parallel reader observed 2a2a…2a, want 00ff…fe01
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.271s
```

**The red was taken honestly**, as instructed: the token was stored as the caller's slice, and no clone was written before the test demanded one. AI-08 hit the identical bug on schema bytes.

**Minimal code:** `bytes.Clone` on the construction path **and** on the read path. Two clones, because a clone in the constructor fixes exactly half of it — the accessor hands `Reasoning` out by value and the slice inside travels with it.

**GREEN**

```
--- PASS: TestReasoning_TokenAcrossCopies_IsUnaffectedByTheCallersSlice (0.00s)
    --- PASS: …/mutating_the_slice_the_caller_supplied_does_not_change_the_token (0.00s)
    --- PASS: …/mutating_the_bytes_an_accessor_returned_does_not_change_the_token (0.00s)
    --- PASS: …/copying_a_message_copies_the_token_exactly (0.00s)
    --- PASS: …/two_consumers_reading_in_parallel_observe_the_same_bytes (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.323s
```

---

### Item 3 — *(appended)* `Part` equality, and the better storage it drove

**Discovered while fixing item 2.** `content_part.go` states the property as landed contract: *"Part is a value… Equality with `==` is defined and compares payloads."* A payload containing a slice makes `==` **panic**, so the arrival of a second kind would have silently removed a property of the first.

**RED**

```
--- FAIL: TestReasoningPart_Equality_ComparesPayloadsWithoutPanicking (0.00s)
    --- FAIL: …/parts_differing_in_payload,_kind_or_presence_are_not_equal (0.00s)
panic: runtime error: comparing uncomparable type ai.Reasoning [recovered, repanicked]
FAIL	github.com/cachicamas/backend/agent/src/ai	0.307s
```

**Minimal code:** the token is stored as a **`string`** rather than a `[]byte`, which is `design.md` § 3.3's destination reached by the route § 3.3 said it would be reached by. A string is immutable, so both halves of the aliasing hazard close by the type rather than by two remembered clones, and both clones were **removed** in the same step. A Go string is a byte container and not a text type — `text_content.go` already relies on that.

**GREEN**, with every earlier assertion still holding:

```
--- PASS: TestReasoning_ConstructedAndReadBack_UsesTheContentPartStrategy (0.00s)
--- PASS: TestReasoningPart_Equality_ComparesPayloadsWithoutPanicking (0.00s)
--- PASS: TestReasoning_AndText_AreStructurallyDistinct (0.00s)
--- PASS: TestReasoning_TokenAcrossCopies_IsUnaffectedByTheCallersSlice (0.00s)
--- PASS: TestReasoning_AbsentToken_IsDistinguishableFromAnEmptyToken (0.00s)
--- PASS: TestReasoningStates_Vocabulary_IsClosedStableAndConstructible (0.00s)
--- PASS: TestReasoning_TokenThroughAMessage_RoundTripsByteIdentical (0.00s)
--- PASS: TestReasoning_OpaqueToken_IsNeitherInterpretedNorNormalized (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.324s
```

Equality also preserves the absent/empty distinction: a part with no token is not equal to one with an empty token, which is AI-07.2 item 1's fact reached through a second door.

---

## Phase AI-07.4 — redacted and signature-only variants `[leaf]`

### Test list

1. - [x] A **redacted** reasoning part carries its opaque payload byte-exact and is distinguishable from a part that merely has no text.
2. - [x] *(pin)* A reasoning part with a token but **no text** is constructible and valid.
3. - [x] *(pin)* Neither variant can be confused with a text part or with an absent part by any accessor.
4. - [x] *(appended)* The reasoning payload renders without its payload.

---

### Item 1 — the redacted variant

**RED — re-observed**, and this is `design.md` § 7's named stub exactly: `NewRedactedReasoning` delegating to `NewReasoning("", token)`, so its state derives to *token-only* and the two shapes become indistinguishable.

```
--- FAIL: TestReasoning_RedactedVariant_ReplaysItsPayloadVerbatim (0.00s)
    --- FAIL: …/redacted_is_distinguishable_from_a_part_that_merely_has_no_text (0.00s)
        reasoning_content_test.go:992: both parts report state token-only — "the provider withheld the plaintext" and "the provider emitted no reasoning text" collapsed into one
    --- FAIL: …/the_opaque_payload_survives_a_message_trip_byte_for_byte (0.00s)
        --- FAIL: …/not_printable (0.00s)
            reasoning_content_test.go:957: reasoning.State() = token-only, want redacted
        --- FAIL: …/a_lone_UTF-16_surrogate_half,_encoded_as_bytes (0.00s)
            reasoning_content_test.go:957: reasoning.State() = token-only, want redacted
        … eight further byte classes, identically
```

**Minimal code:** the redacted door sets `redacted: true`.

**GREEN**

```
--- PASS: TestReasoning_RedactedVariant_ReplaysItsPayloadVerbatim (0.00s)
    --- PASS: …/the_opaque_payload_survives_a_message_trip_byte_for_byte (0.00s)
    --- PASS: …/redacted_is_distinguishable_from_a_part_that_merely_has_no_text (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.300s
```

The distinguishability subtest builds the two shapes from the **identical bytes** and asserts that too, so the state comparison cannot pass because the payloads differed.

---

### Items 2 and 3 — the signature-only shape, and confusability *(pins)*

**Green from birth, declared as pins** over AI-07.1 item 3 and AI-07.2 item 1. What each adds is one word the earlier items did not assert: item 2 adds **valid** — the shape must survive message construction rather than merely be constructible, because `doc 0001` § 3.3 row 2 needs it to reach a request rather than be filtered at the boundary; item 3 adds **absent** — "no text" and "no payload" look alike from outside and are not alike at all.

```
--- PASS: TestReasoning_SignatureOnlyVariant_IsConstructibleAndValid (0.00s)
--- PASS: TestReasoning_RedactedAndSignatureOnly_AreConfusableWithNothing (0.00s)
    --- PASS: …/redacted (0.00s)
    --- PASS: …/signature-only (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.425s
```

Item 2 also puts a signature-only part **beside a text part in one message**, which is the shape an assistant turn actually takes.

---

### Item 4 — *(appended)* the payload rendered its own payload

**Discovered while writing item 3, and it was a real leak.** `content_part.go` records why `Part` has a `String` method: *"fmt prints the unexported fields of a struct it has no String method for, so `%v` on a part would print the prompt."* `Reasoning` is the **first exported payload struct** in the package, so it is the first value that sentence applies to — and it carries the two most sensitive things Layer 1 holds.

**RED**, with the stub `String` in place so the failure is behavioral rather than a build error:

```
--- FAIL: TestReasoning_DiagnosticRendering_CarriesNoPayload (0.00s)
    --- FAIL: …/a_redacted_part (0.00s)
        reasoning_content_test.go:1184: the payload rendered with %#v reproduces the token: ai.Reasoning{text:"", token:"SENSITIVE-SIGNATURE-BYTES", hasToken:true, redacted:true}
        reasoning_content_test.go:1192: reasoning.String() = "", want "reasoning(redacted)"
    --- FAIL: …/reasoning_with_text_and_a_token (0.00s)
        reasoning_content_test.go:1181: the payload rendered with %v reproduces the reasoning text: SENSITIVE-DELIBERATION
        reasoning_content_test.go:1181: the payload rendered with %s reproduces the reasoning text: SENSITIVE-DELIBERATION
        reasoning_content_test.go:1181: the payload rendered with %+v reproduces the reasoning text: SENSITIVE-DELIBERATION
        reasoning_content_test.go:1181: the payload rendered with %#v reproduces the reasoning text: ai.Reasoning{text:"SENSITIVE-DELIBERATION", token:"SENSITIVE-SIGNATURE-BYTES", hasToken:true, redacted:false}
        reasoning_content_test.go:1184: the payload rendered with %#v reproduces the token: ai.Reasoning{…}
        reasoning_content_test.go:1181: the payload rendered with %q reproduces the reasoning text: "SENSITIVE-DELIBERATION"
        reasoning_content_test.go:1192: reasoning.String() = "SENSITIVE-DELIBERATION", want "reasoning(text)"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.384s
```

**Minimal code:** `Reasoning.String()` naming the state alone, and `Reasoning.GoString()` so `%#v` cannot fall back to reflection — `V-FAIL-13` puts the posture on the type rather than on which verb someone reached for.

**GREEN**

```
--- PASS: TestReasoning_DiagnosticRendering_CarriesNoPayload (0.00s)
    --- PASS: …/reasoning_with_text_and_a_token (0.00s)
    --- PASS: …/a_redacted_part (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.283s
```

**Not the length, either.** The rendering names the state and nothing else: a length is caller-derived, and a prefix of a secret is still a secret.

---

## Evidence gate

### `make test` — `go test -race -v ./...`, cache cleared

```
--- PASS: TestViolation_PositionalContext_IsExtractedByErrorsAs (0.00s)
--- PASS: TestReasoning_TokenThroughAMessage_RoundTripsByteIdentical (0.00s)
--- PASS: TestMessage_UnconstructedContentElement_IsRejected (0.00s)
--- PASS: TestReasoningStates_Vocabulary_IsClosedStableAndConstructible (0.00s)
--- PASS: TestReasoning_RedactedVariant_ReplaysItsPayloadVerbatim (0.00s)
--- PASS: TestReasoning_OpaqueToken_IsNeitherInterpretedNorNormalized (0.00s)
--- PASS: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.03s)
--- PASS: TestViolation_ExtremeInputs_NeverPanics (0.00s)
--- PASS: TestPart_HandRolledFromAnotherPackage_DoesNotCompile (0.00s)
--- PASS: TestNewText_RuleViolations_FailWithTheDocumentedSentinels (0.01s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	2.304s
```

Both packages, cache cleared: **44 top-level tests, 254 passing assertions counting subtests, 0 failures.**

```
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.252s
ok  	github.com/cachicamas/backend/agent/src/ai	2.260s
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
--- PASS: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault (0.04s)
--- PASS: TestLayer1_ModuleHasNoDependencies_ZeroRequires (0.04s)
```

```
module github.com/cachicamas/backend/agent

go 1.26.3
```

`go.mod` is unchanged and carries **zero requires**. Everything this milestone added is standard library: `strconv`, `strings` in production; `bytes`, `errors`, `fmt`, `strings`, `sync`, `unicode/utf8` in tests.

---

## Commits

| # | SHA | Subject | Node |
| --- | --- | --- | --- |
| 1 | `82339b1` | `docs(sdd): plan reasoning content and its round-trip token (AI-07)` | planning |
| 2 | `37cfac1` | `feat(ai): add reasoning as a content part with a derived state vocabulary (AI-07.1)` | AI-07.1 |
| 3 | `a65941b` | `feat(ai): carry an opaque round-trip token, absence distinct from empty (AI-07.2)` | AI-07.2 |
| 4 | `99facd8` | `test(ai): prove the round-trip token survives byte-exact across copies (AI-07.3)` | AI-07.3 |
| 5 | `2602cda` | `feat(ai): prove the redacted and signature-only reasoning shapes (AI-07.4)` | AI-07.4 |
| 6 | *this file* | `docs(sdd): record the AI-07 task list with its red-green evidence` | — |

Five of the six fall on node boundaries, so the pull request is reviewable in passes or chainable without rework.

---

## Register amendment

**None.** `V-REQ-09`, `V-REQ-10` and `V-REQ-11` already define reasoning content, its state and its round-trip token, and each names AI-07 as owner. `explore.md` § 6 records the two candidate terms considered and rejected — *"token presence"* and *"redaction"* — and `design.md` § 4.4 records why no rule class was appended to AI-04's five.

`openspec/specs/ai-contract-vocabulary/spec.md` is **not modified by this change**, and neither is `validation.go`.

---

## Doc 0002 amendment

**None proposed.** The revert-and-record clause did not fire: no leaf's first red needed a seam that did not exist, and nothing in AI-07's charter or test lists was disproved by implementation. Four cases were **appended** to leaves — which the clause explicitly routes to the test list rather than to the graph — and they are recorded at their leaves above.

One observation that is not an amendment but belongs on the record for the next kind: `decision.md` § 8's procedure has **five** steps, and adding a kind actually takes **six**. The sixth is a branch in AI-06.2's by-hand exhaustiveness pin, `TestPart_KindAndPayload_Agree`, whose `default` case fails on any kind it does not know. It is not a defect — the pin is doing exactly what it was written to do — but the procedure does not mention it, and AI-09 will meet it too.

---

## What the next milestones inherit

| Milestone | What it can now cite instead of re-deriving |
| --- | --- |
| AI-09 (tool call / tool result) | A worked second application of the five-step procedure, the sixth step above, and `decision.md` § 6.3's rule for an exported payload type with structure — including that such a type needs its own `String`/`GoString` |
| AI-17 (reasoning blocks on the stream) | `Reasoning`, its three states, and a token whose byte-exactness is proven at the model layer, so AI-17 proves it on the *event* path rather than from scratch |
| AI-29 (wire normalization) | The same, from the wire side, plus `ReasoningStateTokenOnly` as the landed answer to "this provider emitted no reasoning text" |
| AI-12.1, AI-26.6 | The round-trip property stated in a form they extend: two hops through a message today, through a rebuild and a wire there |
