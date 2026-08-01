# Tasks — tool calls and tool results

> **Change**: `cachicamas-ai-tool-messages`
> **Milestone**: AI-09 · **Nodes**: AI-09.1 `[leaf]`, AI-09.2 `[leaf]`, AI-09.3 `[leaf]`
> **Phase**: tasks · **Status**: complete
> **Date**: 2026-07-31
> **Mode**: strict TDD (`openspec/config.yaml` → `apply.tdd: true`)
> **Evidence gate**: recorded green `make test` in `backend/agent/` (`go test -race -v ./...`), plus clean `make lint` before every commit

---

## Review Workload Forecast

| Phase | Node | Files touched | Δ lines | Risk | Chainable? |
| --- | --- | --- | --- | --- | --- |
| 0 | planning | 4 SDD artifacts | +1040 | Low | — (docs) |
| 1 | AI-09.1 | `tool_call.go`, `tool_call_test.go`, `content_part.go`, `content_part_registry_test.go`, `content_part_test.go` | +609 | **Medium** — first exported payload type; first kind added under AI-06's procedure by a milestone other than AI-06 | yes, commit 2 |
| 2 | AI-09.2 | `tool_call.go`, `tool_call_test.go` | +199 | Low | yes, commit 3 |
| 3 | AI-09.3 | `tool_result.go`, `tool_result_test.go`, `content_part.go`, `content_part_registry_test.go`, `content_part_test.go` | +556 | Low — inherits every decision phase 1 took | yes, commit 4 |
| | **Total production + tests** | **7 files** | **+1364 / −2** | | **4 commits** |

**Budget reassessment, recorded rather than resolved by deletion.** Doc 0002 prefers < 250 changed
lines and asks for a reassessment before 400. This change lands **1364** lines under
`backend/agent/`, of which **863 are tests** and **403 are production code including its GoDoc**.

The reassessment, honestly:

- **Split trigger 4 fired, and the split was taken** — at the leaf boundary, which doc 0002 calls the
  PR-chain boundary. Four commits, each independently reviewable, each closing on a recorded gate.
  Commits 2 and 4 are the two kinds; commit 3 is one function.
- **It was not split into two changes.** Doc 0002 put both kinds in one milestone, they share the
  `content_part.go` registration edit and the guard's witness table, and `tool_result.go` cannot
  compile as a change of its own without the constant block edit that commit 2 makes. Two changes
  would have produced one that does not build.
- **Tests were not cut to fit.** 863 test lines against 403 of production is the ratio a contract
  milestone should have, and the byte-fidelity, ordinal-derivation and redaction assertions are the
  three this milestone exists for.
- **GoDoc is a deliberate share of the production count.** Both files carry the reasoning a later
  adapter author would otherwise re-derive — why the identity is a supplied string and not a minted
  handle, why the ordinal is not stored, why a failing tool is not an error. Doc 0002's own house
  style makes that documentation load-bearing rather than decorative.

Comparable landed milestones in this wave: AI-06 landed ~1400 lines under `backend/agent/`, AI-08
forecast ~1000. The wave runs over the stated preference and this milestone does not pretend
otherwise.

---

## Phase 1 — AI-09.1 Tool call `[leaf]`

- [x] **1.1** — An external-package test reads a constructed tool call's identity, name and exact
      argument bytes back out of a message.
- [x] **1.2** — Argument bytes survive unmodified: no re-marshalling, no key reordering, no
      whitespace normalization; byte-equality asserted, and neither the caller's buffer nor the
      returned slice aliases the part.
- [x] **1.3** — Construction rules fail through AI-04 sentinels: empty identity, empty name, argument
      bytes that are not syntactically well-formed for the documented encoding.
- [x] **1.4** — A call with empty arguments is constructible and normalizes to one canonical empty
      form that decodes.
- [x] **1.5** *(appended during implementation)* — Neither the exported payload type nor the part
      reproduces the payload under any fmt verb.
- [x] **1.6** — The five-step procedure is completed: constant, payload, constructor, accessor,
      registration table and documented list, plus the AI-06.4 witness with all three legs.

### 1.1 — RED

```
=== RUN   TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage
=== PAUSE TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage
=== CONT  TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage
    tool_call_test.go:39: ai.NewMessage returned content[0]: value is outside a closed vocabulary, want no failure
--- FAIL: TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage (0.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.494s
```

The stub returned the zero `Part`, which carries no payload, therefore no kind, therefore no
membership of the closed kind vocabulary. AI-06's seal rejecting the stub *is* the red step.

### 1.1 — GREEN

```
=== RUN   TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage
=== PAUSE TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage
=== CONT  TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage
--- PASS: TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.464s
```

### 1.1a — The AI-06.4 guard bites, kind one of two

Declaring the constant without a witness is step 3/4/5 left undone, and the guard says so by name.
Recorded verbatim — this is the bite proof AI-06 `decision.md` § 8 requires, taken against a real
missing step rather than a scratch one:

```
--- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.01s)
    --- FAIL: …/the_declared_constants,_PartKinds()_and_partKindNames_agree (0.00s)
        content_part_registry_test.go:166: PartKinds() enumerates 2, which has no witness
    --- FAIL: …/every_declared_constant_has_a_witness,_and_every_witness_a_constant (0.00s)
        content_part_registry_test.go:143: PartKindToolCall is declared in the source but has no entry in partKindWitnesses.
              Adding a kind is the five-step procedure in decision.md § 8, and this guard is what fails when a step is missing.
              Add a witness with all three legs: a constructor with rules, a payload accessor, and a validation path.
FAIL
```

The guard stays red from here until item 1.3 lands the rules its leg 1 demands, and green from the
moment the witness is added. That window is the guard doing its job: it names the missing step
rather than waiting for a reviewer to notice it.

### 1.2 — RED (bite proof by deliberate scratch defect)

The implementation that closed 1.1 already stores the bytes verbatim, so a stub returning `nil`
would prove only that the assertion runs. The stronger red is the **defect this item exists to
catch**: `NewToolCall` was temporarily made to decode and re-encode the caller's bytes, exactly the
canonicalization `V-REQ-17` forbids. All four assertions fired:

```
    tool_call_test.go:113: after the caller overwrote its own buffer, call.Arguments() = "{\"alpha\":{\"nested\":[1,2,3]},\"beta\":\"  spaced  \",\"zeta\":1}", want the bytes supplied at construction
    tool_call_test.go:132: after a consumer overwrote the slice it received, a second read returned "{\"alpha\":{\"nested\":[1,2,3]},\"beta\":\"  spaced  \",\"zeta\":1}", want the constructed bytes
    tool_call_test.go:93: call.Arguments() = "{\"alpha\":{\"nested\":[1,2,3]},\"beta\":\"  spaced  \",\"zeta\":1}",
                        want "{ \"zeta\": 1,\n  \"alpha\":   { \"nested\": [ 1,  2 , 3 ] } ,\n  \"beta\" : \"  spaced  \" }"
          V-REQ-17: no re-marshalling, no key reordering, no whitespace normalization.
    tool_call_test.go:140: a second consumer observed "{\"alpha\":{\"nested\":[1,2,3]},\"beta\":\"  spaced  \",\"zeta\":1}", want the constructed bytes — two holders of one part must not observe each other
--- FAIL: TestToolCall_ArgumentBytes_PassThroughByteIdentically (0.00s)
    --- FAIL: …/the_caller_may_reuse_the_slice_it_passed (0.00s)
    --- FAIL: …/bytes_a_canonicalizing_encoder_would_rewrite_survive_a_message_round_trip (0.00s)
    --- FAIL: …/a_consumer_that_rewrites_what_it_received_cannot_rewrite_the_part (0.00s)
FAIL
```

Note the diff in the first assertion: the keys were sorted **and** the interior whitespace removed —
the fixture was chosen so both halves of the defect are visible in one message.

### 1.2 — GREEN (scratch defect removed)

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.324s
```

### 1.3 — RED

```
--- FAIL: TestNewToolCall_BrokenConstructionRules_FailWithTheDocumentedSentinels (0.00s)
    --- FAIL: …/an_empty_identity (0.00s)
        tool_call_test.go:181: ai.NewToolCall accepted an empty identity, want a caller-contract failure
    --- FAIL: …/the_failure_never_reproduces_the_values_it_was_given (0.00s)
        tool_call_test.go:213: ai.NewToolCall accepted an empty identity, want a caller-contract failure
    --- FAIL: …/argument_bytes_that_are_not_the_documented_encoding_at_all (0.00s)
        tool_call_test.go:181: ai.NewToolCall accepted argument bytes that are not the documented encoding at all, want a caller-contract failure
    --- FAIL: …/argument_bytes_that_are_a_truncated_object (0.00s)
        tool_call_test.go:181: ai.NewToolCall accepted argument bytes that are a truncated object, want a caller-contract failure
    --- FAIL: …/more_than_one_rule_broken_reports_the_first_in_the_documented_order (0.00s)
        tool_call_test.go:198: want a failure carrying required value is empty, got none
    --- FAIL: …/argument_bytes_that_are_whitespace_only (0.00s)
        tool_call_test.go:181: ai.NewToolCall accepted argument bytes that are whitespace only, want a caller-contract failure
    --- FAIL: …/argument_bytes_carrying_two_values_rather_than_one (0.00s)
        tool_call_test.go:181: ai.NewToolCall accepted argument bytes carrying two values rather than one, want a caller-contract failure
    --- FAIL: …/an_empty_tool_name (0.00s)
        tool_call_test.go:181: ai.NewToolCall accepted an empty tool name, want a caller-contract failure
FAIL
```

### 1.3 — GREEN

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.331s
```

**Sentinel walk, recorded.** `ErrEmpty` for the identity and the name, `ErrMalformed` for the
argument bytes. **No class was appended** — `ErrMalformed`'s own GoDoc was written about this exact
case one milestone earlier. A uniqueness rule over call identities was considered and **not**
stretched onto `ErrMalformed`; `design.md` § 3.2 records the deferral to AI-10.3 and names the
duplicate class as the one to use if that boundary decides otherwise.

### 1.4 — RED

```
=== RUN   TestNewToolCall_AbsentArguments_NormalizeToOneCanonicalDecodableForm
=== PAUSE TestNewToolCall_AbsentArguments_NormalizeToOneCanonicalDecodableForm
=== CONT  TestNewToolCall_AbsentArguments_NormalizeToOneCanonicalDecodableForm
    tool_call_test.go:250: ai.NewToolCall with nil arguments returned arguments: value is not well-formed for its documented encoding; a tool that takes no arguments is routine
--- FAIL: TestNewToolCall_AbsentArguments_NormalizeToOneCanonicalDecodableForm (0.00s)
FAIL
```

1.3's rule made the absent case fail, which is exactly the state 1.4 exists to correct: the rule is
right and the *absent* input is not a supplied value it should be judging.

### 1.4 — GREEN

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.321s
```

The invariant it buys, stated once: **every constructible tool call's argument bytes are well-formed
JSON**, including the no-argument case, so AI-26 and AI-30 decode without a guard clause.

### 1.5 — RED (appended case)

```
    tool_call_test.go:330: %v rendered "{call-sk-live-4f2a9c read_sk-live-4f2a9c {\"token\":\"sk-live-4f2a9c\"}}", which reproduces the payload it carries
    tool_call_test.go:330: %s rendered "{call-sk-live-4f2a9c read_sk-live-4f2a9c {\"token\":\"sk-live-4f2a9c\"}}", which reproduces the payload it carries
    tool_call_test.go:330: %#v rendered "ai.ToolCall{id:\"call-sk-live-4f2a9c\", name:\"read_sk-live-4f2a9c\", arguments:\"{\\\"token\\\":\\\"sk-live-4f2a9c\\\"}\"}", which reproduces the payload it carries
--- FAIL: TestToolCall_Rendering_NeverReproducesItsPayload (0.00s)
```

**Why it was appended.** AI-06 put the redaction posture on `Part`, which was complete while every
payload was a string behind an accessor. This milestone is the first to export a payload *type*, and
that reopens it: the argument bytes are model output derived from a user's prompt. `design.md` § 8
records the reasoning and the deliberate divergence from AI-08's `Tool`, which carries no renderers
because a schema is a caller's own literal.

### 1.5 — GREEN

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.318s
```

### 1.6 — Two discovered edits outside the planned file set

1. **`content_part_test.go`** carries AI-06's hand-written exhaustive switch over `PartKinds()`, whose
   `default` branch errors on a kind it does not exercise. Adding a kind requires a branch. It is an
   append at the end of the switch and it is the pin doing its job:
   `kind tool_call is registered but this test does not exercise it — AI-06.4 mechanizes what this default catches by hand`.
   Reported to the orchestrator: AI-07 will hit the identical assertion, and the merge is two
   independent case branches appended to one switch.
2. **A lint failure caught before commit**, exactly the class `design.md` § 9 warned about:
   `src/ai/tool_call_test.go:324:10: S1025: should use String() instead of fmt.Sprintf (staticcheck)`.
   Fixed by calling `String()` directly. `make lint` ran before every commit of this change.

### Phase 1 gate

```
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	2.170s
=== LINT ===
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

Commit: `feat(ai): add the tool call, byte-exact and registered (AI-09.1)`

---

## Phase 2 — AI-09.2 Ordinal position `[leaf]`

- [x] **2.1** — WHEN a message carries several tool calls THEN each call's ordinal position is
      observable and stable across reads.
- [x] **2.2** — The ordinal survives message copy and readback, follows position rather than
      construction order, and the derivation is total.

### 2.1 — RED

```
    tool_call_test.go:375: ai.ToolCalls returned 0 calls, want 3 — a part of another kind is skipped, not counted
    tool_call_test.go:382: ai.ToolCalls returned 0 calls, want 3 — a part of another kind is skipped, not counted
--- FAIL: TestToolCalls_InterleavedContent_AreObservableInOrderWithStableOrdinals (0.00s)
    --- FAIL: …/the_calls_appear_in_content_order_and_nothing_else_does (0.00s)
    --- FAIL: …/every_read_yields_the_same_ordinals (0.00s)
FAIL
```

### 2.1 — GREEN

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.300s
```

### 2.2 — RED (bite proof by deliberate scratch defect)

`ToolCalls` was temporarily made to sort its result by call identity — a plausible wrong
implementation ("make the order deterministic") rather than an implausible one. **Item 2.1 stayed
green** and only 2.2's derivation assertion fired, which is the proof that 2.2 catches something 2.1
does not:

```
--- FAIL: TestToolCalls_MessageCopyAndReadback_PreserveEveryOrdinal (0.00s)
    --- FAIL: …/the_ordinal_follows_the_position_a_message_holds,_not_the_order_the_calls_were_built_in (0.00s)
        tool_call_test.go:428: the call at ordinal 0 is "call-alpha", want "call-gamma"
        tool_call_test.go:428: the call at ordinal 2 is "call-gamma", want "call-alpha"
FAIL
```

### 2.2 — GREEN (scratch defect removed) and phase gate

```
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	2.379s
=== LINT ===
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

**Where the ordinal lives: nowhere.** It is the index of a call in `ToolCalls(content)`. `design.md`
§ 5 and `explore.md` § 5 carry the three candidates and the argument; the short form is `Part.Kind()`
one level up — a stored ordinal is a second copy of a fact the sequence already holds, and a part is
placed in a message *after* it is built, so the two copies are writable independently.

Commit: `feat(ai): derive the tool-call ordinal from position (AI-09.2)`

---

## Phase 3 — AI-09.3 Tool result `[leaf]`

- [x] **3.1** — An external-package test reads a constructed tool result's call correlation and
      content back out of a message.
- [x] **3.2** — Correlation identity round-trips exactly, including identities an adapter minted
      synthetically; an empty correlation fails with an AI-04 sentinel; an empty content is legal.
- [x] **3.3** — A result that reports a tool failure is distinguishable from one that reports
      success, and constructing one is not an error.
- [x] **3.4** *(appended during implementation)* — Neither the exported payload type nor the part
      reproduces the payload under any fmt verb.
- [x] **3.5** — The five-step procedure is completed for the second kind, witness included.

### 3.1 — RED

```
=== RUN   TestToolResult_CorrelationAndContent_ReadBackFromAMessageInAnExternalPackage
=== PAUSE TestToolResult_CorrelationAndContent_ReadBackFromAMessageInAnExternalPackage
=== CONT  TestToolResult_CorrelationAndContent_ReadBackFromAMessageInAnExternalPackage
    tool_result_test.go:33: ai.NewMessage returned content[0]: value is outside a closed vocabulary, want no failure
--- FAIL: TestToolResult_CorrelationAndContent_ReadBackFromAMessageInAnExternalPackage (0.00s)
FAIL
```

### 3.1 — GREEN, and the guard bites for kind two of two

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.359s
=== the guard, for the second kind ===
--- FAIL: TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath (0.01s)
    --- FAIL: …/the_declared_constants,_PartKinds()_and_partKindNames_agree (0.00s)
        content_part_registry_test.go:193: PartKinds() enumerates 3, which has no witness
    --- FAIL: …/every_declared_constant_has_a_witness,_and_every_witness_a_constant (0.00s)
        content_part_registry_test.go:170: PartKindToolResult is declared in the source but has no entry in partKindWitnesses.
              Adding a kind is the five-step procedure in decision.md § 8, and this guard is what fails when a step is missing.
              Add a witness with all three legs: a constructor with rules, a payload accessor, and a validation path.
FAIL
```

The guard bit twice in this change, once per kind, on the same missing step. That is the reason it
exists: the five-step procedure is safe to hand to a later milestone only because a machine fails
when a step is skipped.

### 3.2 — RED

```
--- FAIL: TestToolResult_CorrelationIdentity_RoundTripsExactlyIncludingSyntheticOnes (0.00s)
    --- FAIL: …/an_empty_correlation_is_a_caller-contract_failure (0.00s)
        tool_result_test.go:128: ai.NewToolResult accepted an empty correlation; a result that names no call is an answer to nothing
FAIL
```

The five identity shapes — provider-assigned, adapter-minted synthetic, punctuated, non-ASCII, and a
single byte — round-tripped from the first green, because the value is carried verbatim. The red is
the rule that must exist beside them, and its absence is what the run reports.

### 3.2 — GREEN

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.287s
```

### 3.3 — RED

```
--- FAIL: TestToolResult_ReportedFailure_IsDistinguishableAndIsNotAnError (0.00s)
    --- FAIL: …/constructing_a_failure_produces_no_error_and_reads_back_as_failed (0.00s)
        tool_result_test.go:181: result.Failed() = false on a result built by ai.NewToolFailure
    --- FAIL: …/two_results_alike_but_for_the_outcome_are_distinguishable (0.00s)
        tool_result_test.go:215: a success and a failure with identical correlation and content report the same outcome
        tool_result_test.go:218: a success and a failure with identical correlation and content compare ==; the outcome is part of the value
FAIL
```

### 3.3 — GREEN

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.315s
```

**How a tool failure is represented without touching AI-04.** `NewToolFailure` is a second
constructor returning `(Part, nil)` on its happy path; the outcome is a field on the value, read with
`Failed()`. No sentinel is defined, matched or wrapped, and the AI-06.4 witness for this kind uses
an **empty correlation** as its invalid input, precisely so that a reviewer sees that a failing tool
is *not* the rejected case.

### 3.4 — RED (appended case)

The compile error was the state before red; the narrowest declaration that could not pass was a
`String()` returning the content:

```
--- FAIL: TestToolResult_Rendering_NeverReproducesItsPayload (0.00s)
    tool_result_test.go:272: %v rendered "the tool printed sk-live-4f2a9c", which reproduces the payload it carries
    tool_result_test.go:272: %s rendered "the tool printed sk-live-4f2a9c", which reproduces the payload it carries
    tool_result_test.go:272: %#v rendered "ai.ToolResult{callID:\"call-sk-live-4f2a9c\", content:\"the tool printed sk-live-4f2a9c\", failed:true}", which reproduces the payload it carries
FAIL
```

### 3.4 — GREEN

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.323s
```

The outcome is rendered and the payload is not: the outcome is a two-valued state this package
defines rather than caller data, and it is the one thing a log reader needs.

### 3.5 — Registration, and the phase gate

The second kind's three appended points in `content_part.go`, its witness with three legs, and the
second branch of AI-06's hand-written exhaustive switch. Final gate:

```
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	2.182s
=== LINT ===
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

Both AI-00 import guards, run explicitly:

```
=== RUN   TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault
=== RUN   TestLayer1_ModuleHasNoDependencies_ZeroRequires
--- PASS: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault
--- PASS: TestLayer1_ModuleHasNoDependencies_ZeroRequires
```

`go.mod` carries zero `require` directives; `encoding/json` is the standard library and the
deny-by-default allowlist admits it without an entry.

Commit: `feat(ai): add the tool result, with failure as content (AI-09.3)`

---

## Closing checklist

| Charter acceptance clause | Evidence |
| --- | --- |
| Argument bytes round-trip byte-equal from an external package | `TestToolCall_ArgumentBytes_PassThroughByteIdentically`, plus the recorded bite against a canonicalizing implementation |
| A result's correlation round-trips exactly, synthetic identities included | `TestToolResult_CorrelationIdentity_RoundTripsExactlyIncludingSyntheticOnes`, five identity shapes |
| Ordinal position is observable | `TestToolCalls_InterleavedContent_…` and `TestToolCalls_MessageCopyAndReadback_…`, plus the recorded bite against an imposed order |
| Construction rules fail through AI-04 sentinels; empty arguments construct | `TestNewToolCall_BrokenConstructionRules_…`, `TestNewToolCall_AbsentArguments_…` |
| A tool failure is distinguishable and is not an error | `TestToolResult_ReportedFailure_IsDistinguishableAndIsNotAnError` |
| Both kinds carry a three-leg AI-06.4 witness, and `make test` is green | `TestPartKindRegistration_…/PartKindToolCall`, `…/PartKindToolResult`, both bites recorded |
| `make lint` clean, `go.mod` zero requires | above |

**No sentinel appended. No register amendment applied.** One candidate register clarification —
`V-REQ-17` and the canonical empty form — is reported to the orchestrator in `explore.md` § 7 and
resolved locally by `R-ATM-003` rather than by editing the canonical spec.

**Nothing contradicted doc 0002.** The revert-and-record clause was not invoked: every leaf's first
red test was driven green in small steps, and no missing prerequisite seam was discovered.
