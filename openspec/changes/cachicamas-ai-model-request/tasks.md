# Tasks — the normalized request core

> **Change**: `cachicamas-ai-model-request`
> **Milestone**: AI-10 · **Nodes**: AI-10.1 … AI-10.6, all `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-model-request/spec.md`, `design.md`
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-1`
> **Depends on**: AI-00 … AI-09 merged
> **Blocks**: AI-11, AI-12, AI-20, AI-26
> **Evidence gate**: recorded green `make test` and clean `make lint` in `backend/agent/` (`go test -race -v ./...`)

---

## Entry point for the resuming agent

**AI-10 is complete.** All six leaves — AI-10.1 … AI-10.6 — are implemented and green. There is nothing left to resume in this milestone.

If this change is reopened, start by re-running the evidence gate below to confirm the tree is still exactly as this file records, then consult `openspec/AGENTS.md`/the orchestrator for the next milestone (AI-11 or AI-12 per the dependency graph — both are unblocked by AI-10's completion; AI-20 and AI-26 also depend on AI-10 and are now unblocked).

**`Request.Equal` is exported** (`request.go`), with matching `Message.Equal` (`message.go`) and `SystemInstruction.Equal` (`system_instruction.go`). `backend/agent/src/agenttest/request_test.go`'s round trip (AI-10.5) still carries its own local `requireRequestsEqual` rather than calling `Request.Equal` — that refactor was noted as optional when AI-10.5 landed and remains optional now; nothing depends on it.

**One cross-cutting fix landed inside this milestone, in a file AI-10 does not own:** `backend/agent/src/ai/tool_call.go` (AI-09.1) no longer imports `encoding/json`; its JSON-syntax rule now calls `isWellFormedJSON` (`backend/agent/src/ai/json_syntax.go`, new, AI-10.4's addition). Full account under "AI-10.4 item 3" above. Any later milestone needing JSON-syntax checking should reuse `json_syntax.go`, not reintroduce `encoding/json` anywhere reachable from `src/ai`'s production path.

AI-10.3's rule insertion order matters for what follows: `NewRequest`'s `FirstFailure` list is, in order, model, messages, messages-constructed, content (AI-06's), system, role-vs-kind (`ErrMisplaced`), duplicate-tool-call (`ErrDuplicate`), unresolved-tool-result (`ErrUnresolvedReference`), tool-choice-vs-tool-set (AI-08.3's three), bounds — exactly `design.md` § 4's documented order, rules 1–10. AI-10.4 item 1 proved this order deterministic across runs, by construction, with a bite proof.

**AI-10.4 item 3 found and fixed a real pre-existing defect, not scoped to AI-10.**`tool_call.go` (AI-09.1) used `encoding/json.Valid`, which transitively imports `os`/`io/fs` via `fmt`. AI-10.4's new no-I/O guard caught it correctly. The fix is `json_syntax.go` — a hand-rolled, stdlib-only, differentially-tested (20,000 generated inputs against `encoding/json.Valid` as an oracle, 0 disagreements) JSON syntax scanner — now called from `tool_call.go` in place of `encoding/json.Valid`. `tool_call.go` no longer imports `encoding/json`; the request path's dependency closure is clean. Full detail, including why editing an AI-09 file was in scope, is under "AI-10.4 item 3" below. **If AI-10.5 or AI-10.6 needs JSON-syntax logic again, it already exists in `json_syntax.go` — do not reintroduce `encoding/json`.**

## Node types and what they close on

| Node | Type | Closes on |
| --- | --- | --- |
| AI-10.1 … AI-10.6 | `[leaf]` | Every test-list item taken red → green → refactored, **in order**, with both outputs recorded here |

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so `design.md` § 13 applies: land the narrowest thing that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

Discovered cases are **appended** to the owning leaf's test list, never substituted for a planned one and never pruned to fit the budget.

### Evidence provenance

Every transcript in this file is **observed** — pasted from a command run while implementing the item it sits under. Where a transcript is trimmed for length that is said at the point of the trim. Nothing here is reconstructed from a commit after the fact, and nothing is written before the command that produced it was run.

---

## Review Workload Forecast

Recorded before the work, with actuals filled in as each phase closed.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `spec.md`, `design.md`, `tasks.md` (on top of the landed `explore.md`, `proposal.md`) | ~900 prose | **~950 prose** | Low | 35 min |
| AI-10.1 walking skeleton | `request.go`, `request_test.go`, `request_internal_test.go` | ~200 Go prod, ~400 Go test | **~250 prod / 535 test**, of which 3 files | **High** — every later request milestone and every adapter inherits the shape | 45 min |
| AI-10.2 segmented system instruction | `system_instruction.go`, `system_instruction_test.go`, `request.go` | ~120 Go prod, ~300 Go test | **~280 prod (both files, cumulative) / 302 test** | Medium — the shape AI-11.1 attaches markers to | 30 min |
| AI-10.3 … AI-10.6 | `request.go`, `message.go`, `system_instruction.go`, `validation.go` (+ mirror), `tool_call.go`, `json_syntax.go` (new), `import_boundary_test.go`, `request_test.go`, `json_syntax_internal_test.go` (new), `json_syntax_differential_internal_test.go` (new), `agenttest/request_test.go` (new) | ~350 Go prod, ~900 Go test | **612 prod / 1692 test** (`+612 −3` / `+1692 −0`, `git diff --numstat` over `src/ai` and `src/agenttest`) | **High**, realized — the JSON-syntax fix (AI-10.4 item 3) touched a file outside this milestone's own leaves | ~150 min across 9 per-item/per-leaf commits (`7de9698`…`598974f`) |

Totals across the **whole milestone** (both halves, `git diff --stat` from the pre-AI-10.1 commit to AI-10.6's close): `request.go` ~640 lines, `system_instruction.go` ~180, `message.go` +20, `validation.go` +40, `json_syntax.go` 252 (new), `tool_call.go` net −3 — production is concentrated in `request.go`, and the majority of it is still contract documentation, matching the AI-10.1/.2 pattern. Test code is the larger share throughout: `request_test.go` alone reached beyond 1300 lines by AI-10.6's close, `agenttest/request_test.go` added 486, and the two `json_syntax` test files added 288 between them.

**The forecast under-called AI-10.3 … AI-10.6 by roughly 1.85×** (`~1250` forecast vs. `2307` actual, insertions plus deletions). The gap is almost entirely AI-10.4 item 3: `proposal.md` and `design.md` both scoped this phase as "extend `import_boundary_test.go`", and neither anticipated that the guard, written correctly, would fail on arrival and require a from-scratch RFC 8259 scanner plus two new test files to close honestly. That addition — `json_syntax.go` (252 lines) and its two test files (288 lines) — is roughly half of the 1057-line gap on its own. The remainder is ordinary estimation slack across four leaves worked as nine separate commits.

### Budget reassessment — split trigger 4 fired before the first test was written

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when a milestone's projected diff, tests included, pushes it past the review budget. **It does, by a wide margin, and `explore.md` § 7 recorded it before a line of Go was written.** The reassessment:

- **Split trigger 1 does not fire on the milestone as a whole**, but the milestone *is* six leaves rather than one behavior, and the leaves are individually shippable. So the mitigation is not to cut the test list but to **make the leaf boundary the commit boundary**, which is where doc 0002 puts the PR-chain boundary anyway. The chain exists in history and needs no rework to be reviewed in slices.
- **The milestone is worked as two chained halves at the .2/.3 line.** `proposal.md` forecast the split at .3/.4; it was moved one leaf earlier during execution because AI-10.3 alone carries three cross-region dispositions plus an appended rule class with two registry mirrors, and two prior agents on this milestone died on transcript size. The boundary moved to protect the evidence, not to cut scope: AI-10.3's plan is unchanged and complete in `design.md` §§ 5–7.
- **Production is small; documentation and tests are where the weight is.** The exported surface the rest of the layer inherits is one request type, one system-instruction type, one segment type and five option constructors. What is large is the contract documentation — where the reasoning is put in front of the person who adds the sixth generation option — and the tests.
- **Nothing is cut to fit.** Cases discovered during implementation are appended, never pruned.

---

## Phase AI-10.1 — walking skeleton: a minimal valid request `[leaf]`

**Deliverable:** `backend/agent/src/ai/request.go`, `backend/agent/src/ai/request_test.go`.
**Spec:** `R-AMR-001` … `R-AMR-004`, `R-AMR-017`. **Design:** §§ 2, 4, 8.

- [x] **Item 1** — WHEN a request is constructed with a model identity and one user text message THEN it validates, and both read back exactly **from an external package**.
- [x] **Item 2** — WHEN the model identity is empty, or there are no messages, THEN validation fails with AI-04 sentinels.
- [x] **Item 3** — Generation options carry through construction and read back — the neutral vocabulary decided in AI-01, no more.
- [x] **Item 4** *(appended)* — Option bounds decidable from the request alone are enforced, and temperature deliberately carries no upper bound.
- [x] **Item 5** *(appended)* — The message region is copied on the way out, so a reader cannot rewrite the request it was handed.
- [x] **Item 6** *(appended)* — The content rules are AI-06's, **called** at request depth: a failing part reports the composed position.
- [x] **Item 7** *(appended)* — A request renders no region's payload through any of the four fmt verbs, and still names its shape.

### AI-10.1 item 1 — the skeleton

**Red.** `request_test.go` written first, against a `NewRequest` that returns the zero request and no error.

```
--- FAIL: TestRequest_ModelAndOneUserTextMessage_ReadBackFromAnExternalPackage (0.00s)
    request_test.go:50: request.Model() = "", want "cachicamas-neutral-model-1"
    request_test.go:55: request.Messages() has 0 elements, want 1
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.447s
FAIL
```

**Green.** `NewRequest` keeps the model and clones the message slice; `Model()` and `Messages()` read them back.

```
=== RUN   TestRequest_ModelAndOneUserTextMessage_ReadBackFromAnExternalPackage
=== PAUSE TestRequest_ModelAndOneUserTextMessage_ReadBackFromAnExternalPackage
=== CONT  TestRequest_ModelAndOneUserTextMessage_ReadBackFromAnExternalPackage
--- PASS: TestRequest_ModelAndOneUserTextMessage_ReadBackFromAnExternalPackage (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.324s
```

**Refactor.** None. The test reads the message's role and its content part's text through the exported surface only, which is the readability property doc 0002 makes constitutive.

### AI-10.1 item 2 — the required regions

**Red.** Six sub-cases in one table, against a constructor holding no rules. Trimmed to the failing lines; every sub-case failed.

```
--- FAIL: TestNewRequest_MissingRequiredRegion_FailsWithAnAI04Sentinel (0.00s)
    --- FAIL: .../an_empty_message_sequence (0.00s)
        request_test.go:131: got no failure, want required value is empty at "messages"
    --- FAIL: .../a_nil_message_sequence (0.00s)
        request_test.go:131: got no failure, want required value is empty at "messages"
    --- FAIL: .../a_whitespace-only_model_identity (0.00s)
        request_test.go:131: got no failure, want required value is empty at "model"
    --- FAIL: .../a_skipped_message_after_a_valid_one (0.00s)
        request_test.go:131: got no failure, want required value is empty at "messages[1]"
    --- FAIL: .../a_message_that_skipped_its_constructor (0.00s)
        request_test.go:131: got no failure, want required value is empty at "messages[0]"
    --- FAIL: .../an_empty_model_identity (0.00s)
        request_test.go:131: got no failure, want required value is empty at "model"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.327s
FAIL
```

**Green.** Rules 1–3 of `design.md` § 4 landed as a `FirstFailure` rule slice.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.346s
```

**Refactor.** The unconstructed-message detector reads `Message.ID().IsZero()` rather than inspecting role or content. `message.go` documents that accessor as existing for exactly this question, and role and content are what a forged value would set.

The companion test `TestNewRequest_UnrecognisedModelIdentity_ConstructsBecauseRecognitionIsNotDecidableHere` was written in the same step, green from the start, and is kept: it is the pin that stops the emptiness rule becoming a catalog check.

### AI-10.1 item 3 — the generation options carry through

**Before red** — the compile failure, which is the state before red, not red:

```
src/ai/request_test.go:175:6: undefined: ai.WithMaxOutputTokens
src/ai/request_test.go:176:6: undefined: ai.WithTemperature
src/ai/request_test.go:177:6: undefined: ai.WithTopP
src/ai/request_test.go:178:6: undefined: ai.WithStopSequences
src/ai/request_test.go:184:25: request.MaxOutputTokens undefined (type ai.Request has no field or method MaxOutputTokens)
FAIL	github.com/cachicamas/backend/agent/src/ai [build failed]
```

**Red**, with the option constructors landed and the accessors returning their zero values:

```
--- FAIL: TestRequest_ReappliedGenerationOption_IsLastWins (0.00s)
    request_test.go:258: request.Temperature() = (0, false), want (0.1, true)
--- FAIL: TestRequest_UnappliedGenerationOption_IsDistinguishableFromOneSetToItsZeroValue (0.00s)
    request_test.go:236: zeroed.Temperature() = (0, false), want (0, true)
--- FAIL: TestRequest_GenerationOptions_CarryThroughConstructionAndReadBack (0.00s)
    request_test.go:185: request.MaxOutputTokens() = (0, false), want (4096, true)
    request_test.go:188: request.Temperature() = (0, false), want (0.2, true)
    request_test.go:191: request.TopP() = (0, false), want (0.9, true)
    request_test.go:195: request.StopSequences() reported unset, want set
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.500s
FAIL
```

**Green.** The draft is frozen into the request and each accessor returns its value with its presence flag.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.354s
```

**Refactor.** The four presence flags stayed as separate fields rather than becoming a bitset: a bitset is one indirection between the reader and the fact, and the fact — absence is not zero — is the whole of `V-MET-11`'s distinction.

### AI-10.1 item 4 *(appended)* — the option bounds

**Discovered while writing item 3.** A request carrying `WithMaxOutputTokens(-1)` constructed happily. That is a value no provider accepts and it is decidable without I/O, which is the register's own definition of a caller-contract failure. Appended rather than deferred: adding a bound later is the **tightening** direction, which `proposal.md` records as the expensive one.

**Red.** Trimmed to the failing lines; all eight sub-cases failed.

```
--- FAIL: TestNewRequest_OutOfBoundGenerationOption_FailsAtItsOwnPosition (0.00s)
    --- FAIL: .../zero_maximum_output_tokens (0.00s)
        request_test.go:295: got no failure, want value is outside a documented bound at "maxOutputTokens"
    --- FAIL: .../negative_maximum_output_tokens (0.00s)
        request_test.go:295: got no failure, want value is outside a documented bound at "maxOutputTokens"
    --- FAIL: .../a_top-p_above_one (0.00s)
        request_test.go:295: got no failure, want value is outside a documented bound at "topP"
    --- FAIL: .../the_stop-sequence_option_applied_with_no_sequences (0.00s)
        request_test.go:295: got no failure, want required value is empty at "stopSequences"
    --- FAIL: .../a_negative_top-p (0.00s)
        request_test.go:295: got no failure, want value is outside a documented bound at "topP"
    --- FAIL: .../an_empty_stop_sequence (0.00s)
        request_test.go:295: got no failure, want required value is empty at "stopSequences[1]"
    --- FAIL: .../a_top-p_of_zero (0.00s)
        request_test.go:295: got no failure, want value is outside a documented bound at "topP"
    --- FAIL: .../a_negative_temperature (0.00s)
        request_test.go:295: got no failure, want value is outside a documented bound at "temperature"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.356s
FAIL
```

**Green.** Rule 10 of `design.md` § 4 landed as `requestDraft.boundsRule`, four sub-rules in the documented order, each skipped when its option was not applied.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.357s
```

**Refactor.** The companion `TestNewRequest_BoundaryGenerationOptions_ConstructBecauseNoRuleForbidsThem` pins the bounds that are deliberately **absent** — chiefly a temperature of `4`, which constructs. Without it, `design.md` § 8.3's missing cap reads as an oversight and the next reader closes it. A whitespace-only stop sequence is legal in the same test, because unlike a system-instruction segment it is a real string to match on.

### AI-10.1 item 5 *(appended)* — copy out on the message region

**Discovered while implementing item 3's `StopSequences` accessor**, which clones. `Messages()` did not, so a reader could rewrite the request it was handed.

**AI-10.6 still owns immutability as a whole** — both directions, the constructor side, and equality. This appended case covers only the leak that was landing here, because carrying a known aliasing bug across a commit boundary is how the failures `message.go` warns about survive.

**Red.**

```
--- FAIL: TestRequest_Messages_AreNotMutableThroughWhatAReaderReceived (0.00s)
    request_test.go:359: request.Messages()[0] changed after a reader rewrote the slice it received
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.326s
FAIL
```

**Green.** `Messages()` returns `slices.Clone(r.messages)`, matching `Message.Content()` and `ToolSet.Tools()`.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.331s
```

**Refactor.** None.

### AI-10.1 item 6 *(appended)* — the `validateContent` prefix, consumed

**The seam this milestone exists to cash.** `content_part.go` names this caller by name, and AI-06.3 item 3 pinned the request-shaped prefix `messages[2].content[0]` before a request existed.

The tests live in `request_internal_test.go`, and **internally by necessity rather than preference**. A constructed message cannot hold an invalid content part — `NewMessage` validates its content and `Part` is sealed — so no consumer in another package can assemble the value the test needs. That is itself the argument for landing the rule: it is what keeps `request.go` free of per-kind logic on the day a path that is not `NewMessage` produces a message. `content_part_internal_test.go` is the precedent for reaching inside to pin a seam whose failure mode is unreachable from outside.

**Red.**

```
--- FAIL: TestNewRequest_InvalidContentPart_ReportsAI06sRuleAtTheComposedPosition (0.00s)
    request_internal_test.go:48: NewRequest returned no failure, want ErrNotInVocabulary at messages[1].content[0]
--- FAIL: TestNewRequest_OverlongTextPart_ReportsAI06sDeepPositionBeneathTheRequestPrefix (0.00s)
    request_internal_test.go:77: NewRequest returned no failure, want a bound failure at messages[0].content[0].text
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.474s
FAIL
```

**Green.** Rule 4 landed as one call:

```go
validateContent(Path{AtIndex("messages", i)}, message.Content())
```

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.344s
```

**Refactor.** None — and the absence of a refactor is the deliverable. `request.go` holds **no** per-kind content logic: no text bound, no reasoning-token rule, no tool-call argument rule. The second test asserts the three-level position `messages[0].content[0].text`, of which AI-10 supplies one level and AI-06 supplies two; a request that had reimplemented the text rule would have had to know the third level exists.

### AI-10.1 item 7 *(appended)* — the redaction posture

**Red.** The compile failure first (`request.String undefined`), then, with `String` landed returning `"request()"`:

```
--- FAIL: TestRequest_Formatting_NamesThePresentRegionsAndTheirElementCounts (0.00s)
    request_test.go:436: request.String() = "request()", want it to contain "model"
    request_test.go:436: request.String() = "request()", want it to contain "2 messages"
    request_test.go:436: request.String() = "request()", want it to contain "temperature"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.369s
FAIL
```

Note which half of the pair was red. The **leak** half passed against `"request()"`, because a rendering of nothing leaks nothing — which is precisely why the second test exists. Redaction that renders nothing is not a posture, it is a deleted method, and a leak test alone would have accepted one.

**Green.** `String` renders the present regions and the message count; `GoString` delegates to it. An unapplied option is omitted rather than named as absent.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.349s
```

**Refactor.** The applied-option names moved to `requestDraft.appliedNames()`, a slice built in the documented order, so the rendering of one request is identical on every call and across processes — the same reason `ruleClasses` is a slice and not a map.

### AI-10.1 evidence gate

```
$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.

$ go test -race ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.301s
ok  	github.com/cachicamas/backend/agent/src/ai	2.381s
```

Both AI-00 import guards pass (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires`), and `go.mod` carries zero `require` lines.

---

## Phase AI-10.2 — segmented system instruction `[leaf]`

**Deliverable:** `backend/agent/src/ai/system_instruction.go`, `system_instruction_test.go`; one option constructor and one accessor added to `request.go`.
**Spec:** `R-AMR-005`, `R-AMR-006`, `R-AMR-007`. **Design:** § 3, § 12.1.

- [x] **Item 1** — WHEN a request carries a system instruction as **ordered segments** THEN segment order and content round-trip exactly.
- [x] **Item 2** — The single-segment convenience path produces a request **indistinguishable** from one built segment-by-segment with one segment.
- [x] **Item 3** — An absent system instruction is legal and **distinguishable from one empty segment**.
- [x] **Item 4** — Segment construction rules fail through AI-04 sentinels (empty segment, whitespace-only segment).
- [x] **Item 5** *(appended)* — The system region renders no segment text through any of the four fmt verbs, and still names its segment count.

### AI-10.2 item 1 — ordered segments round-trip

**Before red** — the compile failure, which is the state before red:

```
src/ai/system_instruction_test.go:19:44: undefined: ai.Segment
src/ai/system_instruction_test.go:22:15: undefined: ai.NewSegment
src/ai/system_instruction_test.go:50:25: undefined: ai.NewSystemInstruction
src/ai/system_instruction_test.go:57:6: undefined: ai.WithSystemInstruction
src/ai/system_instruction_test.go:63:22: request.SystemInstruction undefined (type ai.Request has no field or method SystemInstruction)
FAIL	github.com/cachicamas/backend/agent/src/ai [build failed]
```

**Red**, with the two types, their constructors, the option and the accessor landed as stubs:

```
--- FAIL: TestRequest_SystemInstruction_RoundTripsSegmentOrderAndContentExactly (0.00s)
    system_instruction_test.go:62: request.SystemInstruction() reported absent, want present
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.821s
FAIL
```

**Green.** The draft carries the instruction and its presence flag; the request freezes both.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.343s
```

**Refactor.** None. `Segments()` clones on the way out from the start, matching `Message.Content()` and `ToolSet.Tools()`. The test's fixture texts include surrounding whitespace and non-ASCII on purpose, so the byte-exact round trip is asserted rather than assumed.

### AI-10.2 item 2 — the convenience path is indistinguishable

**Before red** — `undefined: ai.NewSystemText`. **Red**, with the declaration landed returning the zero value:

```
--- FAIL: TestNewSystemText_SingleSegmentPath_IsIndistinguishableFromTheSegmentBySegmentBuild (0.00s)
    system_instruction_test.go:99: text-built instruction has 0 segments, segment-built has 1
--- FAIL: TestNewSystemText_TextThatNoSegmentMayCarry_FailsWithTheSegmentsOwnRule (0.00s)
    system_instruction_test.go:117: ai.NewSystemText("   ") returned <nil>, want ErrEmpty
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.326s
FAIL
```

**The second test was written in this step and parked, not deleted.** It asserts a rule item 4 had not landed yet, and driving it green here would have meant implementing item 4's rule during item 2. It was moved to item 4's section unchanged, where it went green with **no further edit to `NewSystemText`** — which is the observable proof that the two construction paths share one rule set.

**Green**, on the first test alone:

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.333s
```

**Refactor.** `NewSystemText` was rewritten to **call** `NewSegment` and then `NewSystemInstruction` rather than to build the value directly. Building it directly passed the identical test and would have let the two paths diverge on the next rule added to a segment — AI-06's "one rule set, two callers" failure mode, one dimension smaller. Item 4's parked test is what turns that from a comment into an assertion.

### AI-10.2 item 3 — absence is structural

**Red.**

```
--- FAIL: TestRequest_AbsentSystemInstruction_IsLegalAndUnrepresentableAsAnEmptySegment (0.00s)
    --- FAIL: .../an_instruction_that_skipped_its_constructor_is_rejected_by_the_request (0.00s)
        system_instruction_test.go:142: got no failure, want required value is empty at "system"
    --- FAIL: .../a_zero-segment_instruction_cannot_be_constructed (0.00s)
        system_instruction_test.go:135: got no failure, want required value is empty at "system"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.329s
FAIL
```

**Green.** `NewSystemInstruction` rejects zero segments and unconstructed segments; `NewRequest` gains rule 5, rejecting an applied-but-unconstructed instruction.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.338s
```

**Refactor.** `SystemInstruction.IsZero` and `Segment.IsZero` were added rather than inlining a length or emptiness check, matching `MessageID.IsZero`, which `message.go` documents as existing for exactly the question "was this constructed?".

The design point this item lands: rule 1 of `NewSystemInstruction` could have been dropped, letting a zero-segment instruction stand for absence. That is the trap it exists to close — it would give the package **two** spellings of absence, the option unapplied and the option applied with an empty value, and every later reader would have to know both.

### AI-10.2 item 4 — segment construction rules

**Red**, five whitespace cases plus item 2's parked test:

```
--- FAIL: TestNewSystemText_TextThatNoSegmentMayCarry_FailsWithTheSegmentsOwnRule (0.00s)
    system_instruction_test.go:222: got no failure, want required value is empty at "text"
--- FAIL: TestNewSegment_EmptyOrWhitespaceOnlyText_FailsWithErrEmpty (0.00s)
    --- FAIL: .../a_non-breaking_space_only (0.00s)
        system_instruction_test.go:188: got no failure, want required value is empty at "text"
    --- FAIL: .../empty_text (0.00s)
        system_instruction_test.go:188: got no failure, want required value is empty at "text"
    --- FAIL: .../spaces_only (0.00s)
        system_instruction_test.go:188: got no failure, want required value is empty at "text"
    --- FAIL: .../an_ideographic_space_only (0.00s)
        system_instruction_test.go:188: got no failure, want required value is empty at "text"
    --- FAIL: .../tabs_and_newlines_only (0.00s)
        system_instruction_test.go:188: got no failure, want required value is empty at "text"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.337s
FAIL
```

**Green.** One rule: `strings.TrimSpace(text) == ""`, reporting `ErrEmpty` at `text`. `TrimSpace` is Unicode-aware, so the non-breaking and ideographic spaces fall under it without a second rule.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.327s
```

**Refactor.** The companion `TestNewSegment_TextWithSurroundingWhitespace_IsNotTrimmed` pins that whitespace-only is a **rejection criterion and never a normalization**. Without it, a later "helpful" trim would pass every other test in the file and silently rewrite callers' prompts — and prompt text is the one thing in this package that must survive byte-exact.

No upper bound was landed on a segment's length, and the omission is documented on the constructor: a text content part has one because it is model-visible content with a documented encoding, while a system instruction's size limit is a provider's, decidable only by asking, and belongs to AI-19.

### AI-10.2 item 5 *(appended)* — the system region's redaction

AI-10.1's leak table could not cover this region because it did not exist yet, and the system instruction is the region most likely to carry a proprietary prompt.

**Before red** — `instruction.String undefined`, `(ai.Segment{}).String undefined`. **Red**, with both renderings landed returning placeholders:

```
--- FAIL: TestSystemInstruction_Formatting_NamesTheSegmentCountAndTheRequestsSystemRegion (0.00s)
    system_instruction_test.go:283: instruction.String() = "system(placeholder)", want "system(2 segments)"
    system_instruction_test.go:286: ai.Segment{}.String() = "segment(placeholder)", want "segment(unset)"
    system_instruction_test.go:289: segment.String() = "segment(placeholder)", want "segment"
    system_instruction_test.go:300: request.String() = "request(model, 1 messages)", want it to contain "system(2 segments)"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.313s
FAIL
```

**Green.** `Segment.String` names the segment and never its text or its length — `Part.String` renders on the same rule, and a prefix of a secret is still a secret. `SystemInstruction.String` names the count, which is shape rather than payload. `Request.String` now includes the system region.

```
ok  	github.com/cachicamas/backend/agent/src/ai	1.333s
```

**Refactor.** None. `GoString` delegates to `String` on both types, matching every payload-carrying type in the package.

### AI-10.2 evidence gate

```
$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.

$ go test -race ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.543s
ok  	github.com/cachicamas/backend/agent/src/ai	2.146s
```

`go.mod` carries zero `require` lines; both AI-00 import guards pass.

---

## Phase AI-10.3 — messages, tools and tool choice on the request `[leaf]` — **IMPLEMENTED**

**Deliverable:** `request.go` extended; `validation.go` plus **both** registry mirrors; `request_test.go` extended.
**Spec:** `R-AMR-008` … `R-AMR-012`. **Design:** §§ 5, 6, 7, and § 4 rows 6–9.

- [x] ~~**Item 0** *(prerequisite, do this first)* — Append `ErrMisplaced`~~ to `validation.go`'s class set **and** to `ruleClasses`, **and** update `validation_registry_internal_test.go` and `validation_test.go` in the **same commit**. The internal guard fails and says so if only one mirror moves. `design.md` § 5.3 carries the doc comment to write.
- [x] ~~**Item 1** — Message order and intra-message content order are preserved exactly through construction and readback.~~
- [x] ~~**Item 2** — The tool set and tool choice attach to the request, and AI-08.3's cross-validation runs at the request boundary too. **Call `ToolChoice.ValidateAgainst(ToolSet)`; reimplement none of its three rules.**~~
- [x] ~~**Item 3** — Role-versus-content-kind rules are enforced from `design.md` § 5.1's table, all twelve cells, in both directions, reporting `ErrMisplaced`.~~
- [x] ~~**Item 4** — An orphan tool result fails with `ErrUnresolvedReference`; an orphan tool **call** succeeds; a duplicate call identity fails with `ErrDuplicate` at the second occurrence; a result appearing before its call **succeeds**, pinning `design.md` § 6.3's deliberate non-decision.~~

### AI-10.3 item 0 — `ErrMisplaced` appended, three files or none

**Red (a).** The append was driven from the mirror first, so the first failure is the one a consumer would hit:

```
# github.com/cachicamas/backend/agent/src/ai_test [github.com/cachicamas/backend/agent/src/ai.test]
src/ai/validation_test.go:104:5: undefined: ai.ErrMisplaced
FAIL	github.com/cachicamas/backend/agent/src/ai [build failed]
FAIL
```

**Red (b).** With the sentinel declared in `validation.go` but not yet registered, AI-04's guard bit in **both** directions — the declared-but-unregistered hole and the mirror-drift hole — which is the guard working as doc 0002 defines a guard leaf:

```
--- FAIL: TestRuleClasses_EverySentinelDeclaredInTheSource_IsInTheRegistry (0.00s)
    validation_registry_internal_test.go:116: validation.go declares the sentinel ErrMisplaced, which ruleClasses does not list.
          rule: the rule-class set is closed, and a class is appended in the pull request that needs it (design.md § 3.1) — declaring the sentinel is half of that append.
          A sentinel outside ruleClasses renders as "value violates an unregistered rule", and no test of this package covers it.
--- FAIL: TestRuleClasses_TheExternalTestMirror_MatchesTheRegistryExactly (0.01s)
    validation_registry_internal_test.go:184: the mirror in validation_test.go lists ErrMisplaced, which validation.go does not register.
          rule: the mirror names the closed set, and a name outside it is either a sentinel that was never appended to ruleClasses or one that was removed from it.
    validation_registry_internal_test.go:192: the mirror is not the registry.
          validation.go: ErrEmpty, ErrNotInVocabulary, ErrOutOfRange, ErrMalformed, ErrUnresolvedReference, ErrDuplicate
          validation_test.go: ErrEmpty, ErrNotInVocabulary, ErrOutOfRange, ErrMalformed, ErrUnresolvedReference, ErrDuplicate, ErrMisplaced
          rule: same members, same order. ruleClasses is ordered on purpose, and a mirror free to reorder it is a second answer to which class comes first.
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.524s
FAIL
```

**Green.** `ErrMisplaced` appended to `ruleClasses`. All three files moved in one commit, which is what the guard was demanding.

```
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.276s
ok  	github.com/cachicamas/backend/agent/src/ai	2.223s
```

**Refactor.** None. The GoDoc follows AI-04's `ErrDuplicate` precedent — a citable case plus the difference from the nearest neighbour — and the neighbour named is `ErrNotInVocabulary`: a value outside a vocabulary is not a member and must be *renamed*; a misplaced value **is** a member, valid against every rule its own type carries, and must be *moved or dropped*.

### AI-10.3 item 1 *(pin)* — message and content order

**Green from birth.** AI-10.1 copies the message sequence and AI-05 copies a message's content, so `S-AMR-033`, `S-AMR-034` and `S-AMR-035` all held before the test was written. Recorded as a pin rather than a red, in `validation_test.go`'s established idiom.

It is pinned rather than assumed because **every cross-region rule this leaf lands reports by index** — item 4's duplicate rule reports "the second occurrence", its correlation rule scans "the whole request in order" — and a position by index says nothing the moment order is not preserved. The pin is the precondition of the three items after it.

**Closed by showing it bite**, doc 0002's rule for a check that is green on arrival. Scratch violation A, `Messages()` returning `r.messages` instead of a clone:

```
--- FAIL: TestRequest_MessageAndContentOrder_ArePreservedThroughConstructionAndReadback (0.00s)
    request_test.go:590: after the reader reversed the slice it was handed, request.Messages()[0] is not the message built at position 0 — the request handed out its own storage
    request_test.go:590: after the reader reversed the slice it was handed, request.Messages()[1] is not the message built at position 1 — the request handed out its own storage
    request_test.go:590: after the reader reversed the slice it was handed, request.Messages()[3] is not the message built at position 3 — the request handed out its own storage
    request_test.go:590: after the reader reversed the slice it was handed, request.Messages()[4] is not the message built at position 4 — the request handed out its own storage
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.352s
FAIL
```

Scratch violation B, `Messages()` returning a reversed clone — order, content order and the copy pin all bite:

```
--- FAIL: TestRequest_MessageAndContentOrder_ArePreservedThroughConstructionAndReadback (0.00s)
    request_test.go:571: request.Messages()[0] is not the message built at position 0 — the sequence was reordered between construction and readback
    request_test.go:571: request.Messages()[1] is not the message built at position 1 — the sequence was reordered between construction and readback
    request_test.go:571: request.Messages()[3] is not the message built at position 3 — the sequence was reordered between construction and readback
    request_test.go:571: request.Messages()[4] is not the message built at position 4 — the sequence was reordered between construction and readback
    request_test.go:578: the second message's content kinds = [text], want [text reasoning tool_call]
    request_test.go:581: the second message's first part = ("four", true), want ("first", true)
    request_test.go:590: after the reader reversed the slice it was handed, request.Messages()[0] is not the message built at position 0 — the request handed out its own storage
    request_test.go:590: after the reader reversed the slice it was handed, request.Messages()[1] is not the message built at position 1 — the request handed out its own storage
    request_test.go:590: after the reader reversed the slice it was handed, request.Messages()[3] is not the message built at position 3 — the request handed out its own storage
    request_test.go:590: after the reader reversed the slice it was handed, request.Messages()[4] is not the message built at position 4 — the request handed out its own storage
FAIL
```

**Green**, with `request.go` restored:

```
ok  	github.com/cachicamas/backend/agent/src/agenttest	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai	2.141s
```

**Refactor.** The mixed-kind message is built with `RoleAssistant`, which `design.md` § 5.1 makes the only role permitted to carry text, reasoning and a tool call at once. Using a forbidden cell here would have made an order test start failing on item 3 for a reason that had nothing to do with order. No production code changed in this item.

### AI-10.3 item 2 — the tool set and tool choice attach, cross-validation reused

**Deliverable:** `request.go` gains `WithTools`, `WithToolChoice`, `Tools()`, `ToolChoice()`, `requestDraft.tools`/`hasTools`/`toolChoice`/`hasToolChoice`, rule 9 of `design.md` § 4's table, and the two regions' line in `String()`. `validation.go` gains `violationOf`, the converter named in `design.md` § 4.2 and `decision.md` § 7.2's "one rule set, two callers" applied to a rule that is itself a method call rather than a check written in this file. `request_test.go` gains three tests.
**Spec:** `R-AMR-010`. **Design:** § 4 row 9, § 4.2.

**Before red.** `request_test.go`'s three new tests written first, against `request.go` unchanged from the AI-10.3 item 1 commit:

```
# github.com/cachicamas/backend/agent/src/ai_test [github.com/cachicamas/backend/agent/src/ai.test]
src/ai/request_test.go:633:6: undefined: ai.WithTools
src/ai/request_test.go:633:27: undefined: ai.WithToolChoice
src/ai/request_test.go:639:33: request.Tools undefined (type ai.Request has no field or method Tools)
src/ai/request_test.go:651:35: request.ToolChoice undefined (type ai.Request has no field or method ToolChoice)
src/ai/request_test.go:664:51: undefined: ai.WithTools
src/ai/request_test.go:669:35: withoutChoice.ToolChoice undefined (type ai.Request has no field or method ToolChoice)
src/ai/request_test.go:697:35: undefined: ai.WithToolChoice
src/ai/request_test.go:705:35: undefined: ai.WithToolChoice
src/ai/request_test.go:712:7: undefined: ai.WithTools
src/ai/request_test.go:713:7: undefined: ai.WithToolChoice
src/ai/request_test.go:713:7: too many errors
FAIL	github.com/cachicamas/backend/agent/src/ai [build failed]
FAIL
```

**Red.** `requestDraft`'s two field pairs, `WithTools` and `WithToolChoice` landed — the option constructors are total per § 2.2, so there is nothing to stub in them — but `Tools()` and `ToolChoice()` deliberately stubbed to report absence unconditionally, and rule 9 not yet inserted into `NewRequest`'s `FirstFailure` list:

```
=== NAME  TestRequest_ToolSetAndToolChoice_AttachAndReadBack
    request_test.go:641: request.Tools() reported no tool set, want the one applied
--- FAIL: TestRequest_ToolSetAndToolChoice_AttachAndReadBack (0.00s)
=== NAME  TestRequest_Formatting_NamesTheToolRegionsWithoutNamingAnyTool
    request_test.go:754: request.String() = "request(model, 1 messages)", want it to contain "2 tools"
    request_test.go:754: request.String() = "request(model, 1 messages)", want it to contain "toolChoice(specific)"
--- FAIL: TestRequest_Formatting_NamesTheToolRegionsWithoutNamingAnyTool (0.00s)
=== NAME  TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions/a_specific_tool_choice_and_no_tool_set_at_all
    request_test.go:724: got no failure, want required value is empty at "tools"
=== NAME  TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions/a_tool_choice_that_was_never_constructed
    request_test.go:724: got no failure, want value is outside a closed vocabulary at "toolChoice"
=== NAME  TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions/a_specific_tool_choice_naming_an_undeclared_tool
    request_test.go:724: got no failure, want value names something the request does not declare at "toolChoice.name"
--- FAIL: TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions (0.00s)
    --- FAIL: TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions/a_specific_tool_choice_and_no_tool_set_at_all (0.00s)
    --- FAIL: TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions/a_tool_choice_that_was_never_constructed (0.00s)
    --- FAIL: TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions/a_specific_tool_choice_naming_an_undeclared_tool (0.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.501s
FAIL
```

**Green.** `Tools()` and `ToolChoice()` wired to `r.options`; rule 9 inserted at its documented position — `func() *Violation { if !draft.hasToolChoice { return nil }; return violationOf(draft.toolChoice.ValidateAgainst(draft.tools)) }` — calling AI-08.3's own method rather than reimplementing its three rules; `String()` gains the two regions' rendering.

```
--- PASS: TestRequest_ToolSetAndToolChoice_AttachAndReadBack (0.00s)
--- PASS: TestRequest_Formatting_NamesTheToolRegionsWithoutNamingAnyTool (0.00s)
--- PASS: TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions (0.00s)
    --- PASS: .../a_specific_tool_choice_and_no_tool_set_at_all (0.00s)
    --- PASS: .../a_specific_tool_choice_naming_an_undeclared_tool (0.00s)
    --- PASS: .../a_tool_choice_that_was_never_constructed (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	(cached)
```

**Refactor — folded into the same commit as an appended case, not a separate item.** `TestRequest_Formatting_NamesTheToolRegionsWithoutNamingAnyTool` extends AI-10.1 item 7's leak table to the two new regions, which could not exist when that table was written. The assertion is on the position, not only the class: `TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions` checks `"toolChoice"`, `"tools"` and `"toolChoice.name"` — AI-08.3's own positions — rather than a request-shaped prefix a reimplementation would have produced, which is the observable proof that `ValidateAgainst` was *called* and not rewritten.

### AI-10.3 item 3 — role versus content kind, all twelve cells

**Deliverable:** `request.go` gains `rolePermittedKinds`, `roleAllowsKind`, and rule 6 of `design.md` § 4's table, inserted between rule 5 (system) and the existing rule 9 (tool choice against the tool set) — no rule needed to move, because that rule was already the last content rule before `boundsRule()` and rules 6–8 slot in ahead of it by construction. `request_test.go` gains `toolResultPart`, the fourth per-kind part builder, and one table-driven test.
**Spec:** `R-AMR-011`. **Design:** § 5.

**Red.** Twelve sub-cases in one table, against `NewRequest` holding no role/kind rule. Exactly the seven forbidden cells failed; the five permitted cells — including the tool_result cell, which carries a companion tool call so item 4's not-yet-landed orphan rule cannot affect it — passed from the start, which is the expected shape of a red step for a rule that does not exist yet:

```
--- FAIL: TestNewRequest_RoleVersusContentKind_EnforcesTheDocumentedTable (0.00s)
    --- PASS: .../tool_call_under_assistant (0.00s)
    --- PASS: .../text_under_assistant (0.00s)
    --- FAIL: .../tool_call_under_user (0.00s)
        request_test.go:837: got no failure, want value is not permitted where it appears at "messages[0].content[0]"
    --- FAIL: .../reasoning_under_user (0.00s)
        request_test.go:837: got no failure, want value is not permitted where it appears at "messages[0].content[0]"
    --- PASS: .../reasoning_under_assistant (0.00s)
    --- FAIL: .../tool_call_under_tool (0.00s)
        request_test.go:837: got no failure, want value is not permitted where it appears at "messages[0].content[0]"
    --- FAIL: .../reasoning_under_tool (0.00s)
        request_test.go:837: got no failure, want value is not permitted where it appears at "messages[0].content[0]"
    --- PASS: .../text_under_user (0.00s)
    --- PASS: .../tool_result_under_tool (0.00s)
    --- FAIL: .../tool_result_under_assistant (0.00s)
        request_test.go:837: got no failure, want value is not permitted where it appears at "messages[0].content[0]"
    --- FAIL: .../tool_result_under_user (0.00s)
        request_test.go:837: got no failure, want value is not permitted where it appears at "messages[0].content[0]"
    --- FAIL: .../text_under_tool (0.00s)
        request_test.go:837: got no failure, want value is not permitted where it appears at "messages[0].content[0]"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.522s
FAIL
```

**Green.** `rolePermittedKinds` landed as a slice indexed by `Role`, `roleAllowsKind` reads it with `slices.Contains`, and rule 6 walks every message and every content part in order, reporting `ErrMisplaced` at the first cell the table forbids.

```
--- PASS: TestNewRequest_RoleVersusContentKind_EnforcesTheDocumentedTable (0.00s)
    --- PASS: .../reasoning_under_assistant (0.00s)
    --- PASS: .../reasoning_under_user (0.00s)
    --- PASS: .../text_under_tool (0.00s)
    --- PASS: .../tool_result_under_tool (0.00s)
    --- PASS: .../tool_result_under_user (0.00s)
    --- PASS: .../text_under_assistant (0.00s)
    --- PASS: .../tool_result_under_assistant (0.00s)
    --- PASS: .../tool_call_under_assistant (0.00s)
    --- PASS: .../tool_call_under_tool (0.00s)
    --- PASS: .../reasoning_under_tool (0.00s)
    --- PASS: .../text_under_user (0.00s)
    --- PASS: .../tool_call_under_user (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.474s
```

**Refactor.** None. The table is a slice rather than a map, matching `roleNames`, `partKindNames` and `toolChoiceModes` — AI-04's reason applies verbatim: nothing in this package may let an unordered iteration decide anything, and `slices.Contains` over a three-or-fewer-element slice is a bounded scan whose answer cannot depend on order.

**Note on scenario S-AMR-046's count.** `spec.md` describes this scenario as covering "the four permitted cells", but the table in `design.md` § 5.1 and this spec's own requirement text yield **five**: `text` is permitted under both `user` and `assistant`. The test implements the requirement text and `design.md`'s table — five permitted cells, seven forbidden, twelve total, both directions — rather than the scenario prose's count, which undercounts by one. Not a `[provisional]` decision changed, since the requirement and the table already said five; the prose miscount is left as-is rather than edited, because `spec.md` is this change's own delta and out of scope for the apply phase to rewrite.

### AI-10.3 item 4 — orphan results, orphan calls, duplicate identities, and the ordering non-decision

**Deliverable:** `request.go` gains `toolCallLocation`, `duplicateToolCallRule`, `unresolvedToolResultRule`, `anyToolCallHasID`, and rules 7 and 8 of `design.md` § 4's table, inserted between rule 6 (role-vs-kind) and rule 9 (tool choice against the tool set). `request_test.go` gains one test with four sub-cases.
**Spec:** `R-AMR-012`. **Design:** §§ 6, 7.

**Red.** Four sub-cases against `NewRequest` holding neither rule. The two rejection cases failed exactly as expected; the two "legal" cases already passed, because nothing yet rejected them — the correct red shape for a rule that does not exist yet:

```
--- FAIL: TestNewRequest_ToolCallAndResultCorrelation_ChecksAcrossTheRequest (0.00s)
    --- PASS: .../an_orphan_tool_call_is_legal (0.00s)
    --- PASS: .../a_result_before_its_call_is_legal (0.00s)
    --- FAIL: .../an_orphan_tool_result_is_rejected (0.00s)
        request_test.go:859: got no failure, want value names something the request does not declare at "messages[0].content[0]"
    --- FAIL: .../a_duplicate_call_identity_fails_at_the_second_occurrence (0.00s)
        request_test.go:886: got no failure, want value repeats another the collection already carries at "messages[1].content[0]"
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.493s
FAIL
```

**Green.** `duplicateToolCallRule` collects every tool call into one ordered slice (message index, content index, identity) and scans it exactly as `NewToolSet` scans a tool set — linear and nested, reporting the lowest index whose identity repeats an earlier one. `unresolvedToolResultRule` walks every tool result and asks `anyToolCallHasID`, which scans the whole request rather than a prefix, cashing `design.md` § 6.3's "anywhere in the request" wording literally.

```
--- PASS: TestNewRequest_ToolCallAndResultCorrelation_ChecksAcrossTheRequest (0.00s)
    --- PASS: .../a_result_before_its_call_is_legal (0.00s)
    --- PASS: .../an_orphan_tool_result_is_rejected (0.00s)
    --- PASS: .../a_duplicate_call_identity_fails_at_the_second_occurrence (0.00s)
    --- PASS: .../an_orphan_tool_call_is_legal (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.340s
```

**Refactor.** None. Re-running item 3's `tool_result_under_tool` sub-test alone confirmed the companion-message design anticipated this rule correctly: it still passes now that `unresolvedToolResultRule` is live.

### AI-10.3 evidence gate

```
$ go test -race ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai	2.160s

$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

`go.mod` carries zero `require` lines; both AI-00 import guards pass. `NewRequest`'s `FirstFailure` list now matches `design.md` § 4's documented order exactly, rules 1–10 — no rule needed relocating, because AI-10.3 item 2 had already landed the tool-choice-vs-tool-set rule as the last content rule before `boundsRule()`, and rules 6–8 slot in ahead of it by construction rather than by a later reorder.

---

## Phase AI-10.4 — validation happens once, before I/O `[leaf]` — **IMPLEMENTED**

**Spec:** `R-AMR-013`, `R-AMR-014`. **Design:** § 9.

- [x] ~~**Item 1** — The first failure in the documented order is reported, identically across runs.~~
- [x] ~~**Item 2** — Validation is total over the regions.~~
- [x] ~~**Item 3** — The request path's dependency closure contains no network and no filesystem package, asserted mechanically in the AI-00.3 guard style. Extend `import_boundary_test.go` rather than adding a parallel guard.~~

### On this phase's evidence shape

All three items assert properties `design.md` § 9 states are **already true** by construction — "AI-10.4's work is to assert it, not to build it" (§ 9.1). Each test is therefore a **pin**: green from birth, in the same category `TestLayer1_ModuleHasNoDependencies_ZeroRequires` already established as legitimate and exempt from red-first. Doc 0002's rule for a check that is green on arrival is to close it by **showing it bite** a scratch violation, recorded below for every pin in this phase — the same idiom AI-10.3 item 1 used.

### AI-10.4 item 1 — determinism, shown to bite

**Deliverable:** `request_test.go` gains one test, two sub-cases (S-AMR-053, S-AMR-054).
No production change: `design.md` § 4's rule list is already a `[]Rule` in slice order with no map anywhere in the path.

**Green from birth**, against the rule order landed through AI-10.3:

```
--- PASS: TestNewRequest_MultipleViolations_ReportsTheDocumentedOrderFirstAcrossManyRuns (0.00s)
    --- PASS: .../model_and_messages_both_violated_reports_the_model_failure (0.00s)
    --- PASS: .../four_regions_violated_at_once_reports_the_earliest_rule (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.532s
```

**Closed by showing it bite.** Scratch violation: rules 1 (model) and 2 (messages) swapped in `NewRequest`'s `FirstFailure` list, `request.go` otherwise untouched. Trimmed to the first 20 of 100 identical failures — the test reports every one, and all 100 disagreed the same way:

```
    request_test.go:926: violation.Path() = "messages", want "model"
    request_test.go:926: violation.Path() = "messages", want "model"
    ... (18 more, identical)
--- FAIL: TestNewRequest_MultipleViolations_ReportsTheDocumentedOrderFirstAcrossManyRuns (0.00s)
    --- FAIL: .../model_and_messages_both_violated_reports_the_model_failure (0.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.474s
```

`request.go` restored from the pre-scratch state; the test above re-ran green afterward, confirming the restore was exact (bytes identical to the pre-scratch file, diffed).

### AI-10.4 item 2 — totality, one clause shown to bite, one shown unreachable

**Deliverable:** `request_test.go` gains two tests and one helper (`requireTool`).
No production change to `request.go`: this item asserts that AI-10.1–AI-10.3's rules already cover every region, and does not add a new one.

**Green from birth** — the re-examination test (content rebuilt via `NewMessage`, tool choice re-validated via `ValidateAgainst`) and the duplicate-name unreachability test:

```
--- PASS: TestNewRequest_ThatConstructed_ReExaminesCleanUnderEveryRegionsOwnContract (0.00s)
--- PASS: TestNewRequest_DuplicateToolName_CannotReachARequestBecauseNewToolSetRefusesIt (0.00s)
```

**Closed by showing it bite, first clause (content + tool choice).** Scratch violation: rule 9 (tool-choice-vs-tool-set) removed from `request.go`, and the re-examine test's fixture temporarily pointed at an unresolvable choice (`"SCRATCH-BITE-undeclared-tool"` naming no declared tool). Construction wrongly succeeded — proving rule 9 was doing real work — and the re-examine step caught the resulting inconsistency:

```
--- FAIL: TestNewRequest_ThatConstructed_ReExaminesCleanUnderEveryRegionsOwnContract (0.00s)
    request_test.go:1017: readChoice.ValidateAgainst(readTools) returned toolChoice.name: value names something the request does not declare, want no failure — a tool choice this package attached to a request must still validate against that request's own tool set
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.338s
```

Both `request.go` and `request_test.go` restored from the pre-scratch state (diffed byte-identical); the full suite re-ran green afterward.

**Second clause (duplicate tool name), not independently bitten.** `design.md` § 9.2 names this clause's asymmetry explicitly: a request cannot hold a tool set with a repeated name because `NewToolSet` (AI-08.2) already refuses to construct one — there is no separate request-level rule to disable and re-enable. The bite proof for the underlying rule already exists in `tool_set_test.go` (AI-08's own suite, not this milestone's to touch); this test's role is only the integration fact that the unreachability is what makes `R-AMR-013`'s second clause true at the request level.

### AI-10.4 item 3 — the no-I/O guard, and what it found

**Deliverable:** `import_boundary_test.go` extended with `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage`, `requestPackagePath`, `networkOrFilesystemPackages`, `listAllDeps`.

**The guard, written exactly as `design.md` § 9.3 specifies, did not pass on arrival.** `go list -deps` (no `-test`) on `github.com/cachicamas/backend/agent/src/ai` reported `os` and `io/fs` in the closure:

```
=== CONT  TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage
    import_boundary_test.go:292: the request path's dependency closure imports "os"
          rule: V-REQ-22: validation runs entirely at construction and performs no I/O of its own — no filesystem package
    import_boundary_test.go:292: the request path's dependency closure imports "io/fs"
          rule: V-REQ-22: validation runs entirely at construction and performs no I/O of its own — no filesystem package
--- FAIL: TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage (0.04s)
FAIL
```

**Root cause, traced mechanically, not guessed:**

```
$ go list -deps -f '{{.ImportPath}}' encoding/json | grep -x -E 'os|io/fs|fmt'
io/fs
os
fmt
$ go list -f '{{.Imports}}' fmt | tr ' ' '\n' | grep -x 'os'
os
```

`tool_call.go` (AI-09.1) calls `encoding/json.Valid` to check `V-REQ-17`'s "syntactically well-formed JSON" rule on argument bytes. `encoding/json` imports `fmt` for its own error formatting, and `fmt` imports `os` for `Stdout`/`Stderr`, which imports `io/fs`. This chain is real, current (Go 1.26.3, this repository), and pre-dates AI-10 — AI-09 had no guard that would have caught it, because this specific guard is what AI-10.4 item 3 exists to add.

**This is a genuine defect the guard is designed to catch, not a false positive in the new test.** `design.md` § 9.3's own words: "The guard is what makes V-REQ-22's 'validation happening after a socket is open is a different concept and a defect' mechanical rather than aspirational." Weakening the guard to tolerate this one case would be exactly the aspirational-not-mechanical failure it exists to prevent — R-AMR-013/S-AMR-051 state the rule with no carve-out. `tool_call.go` is inside `backend/agent/src/ai/`, which `design.md` § 1 does not forbid editing (its rule is "no file **outside** `backend/agent/src/ai/`"); AI-09 is a merged, upstream dependency of AI-10, not a parallel change, so a self-contained internal fix that preserves AI-09's own tests and its documented contract is in scope.

**The fix: `json_syntax.go`, a hand-rolled RFC 8259 syntax scanner (`isWellFormedJSON`), stdlib-only and free of `os`/`io/fs`.** It decodes nothing — syntax only, matching `ErrMalformed`'s own documented boundary — and is a pure function over `[]byte`. `tool_call.go`'s rule 3 now calls it instead of `json.Valid`; the `encoding/json` import is removed from that file entirely.

**Safety net (approval-testing discipline, `strict-tdd.md`'s pattern for refactoring existing code).** `TestNewToolCall_BrokenConstructionRules_FailWithTheDocumentedSentinels` and `TestNewToolCall_AbsentArguments_NormalizeToOneCanonicalDecodableForm` — AI-09's own tests, thirteen sub-cases — ran unchanged before touching `tool_call.go`, and ran unchanged after, byte-for-byte the same assertions, same expected values:

```
$ go test -race -v -run 'TestNewToolCall' ./src/ai/...
--- PASS: TestNewToolCall_AbsentArguments_NormalizeToOneCanonicalDecodableForm (0.00s)
    (3 sub-tests PASS)
--- PASS: TestNewToolCall_BrokenConstructionRules_FailWithTheDocumentedSentinels (0.00s)
    (10 sub-tests PASS, including "well-formed bytes no declared tool would accept still construct"
     and "argument bytes carrying two values rather than one")
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.349s
```

**New coverage on the scanner itself**, beyond what any one caller needs, because a syntax checker with a silent gap is worse than the dependency it replaces:

- `json_syntax_internal_test.go` — `TestIsWellFormedJSON_RFC8259Grammar_MatchesEncodingJSONValid`, 55 named sub-cases spanning every RFC 8259 production (objects, arrays, strings with escapes and unicode escapes, numbers with fractions and exponents, the three literals) plus every case `tool_call_test.go` already carries, pinned here too. All 55 passed on the first run.
- `json_syntax_differential_internal_test.go` — `TestIsWellFormedJSON_AgreesWithEncodingJSONValid_OverManyGeneratedInputs`, a deterministic (fixed-seed) generator producing 20,000 JSON-shaped and mutated near-JSON byte strings, checked against `encoding/json.Valid` as an oracle. `encoding/json` is imported **only in this test file**: safe, because this guard's own `listAllDeps` deliberately omits `-test`, for the reason stated on the guard itself — a test file may need something production code must not. **0 of 20,000 disagreed.**

```
$ go test -race -run 'TestIsWellFormedJSON' -v ./src/ai/...
--- PASS: TestIsWellFormedJSON_RFC8259Grammar_MatchesEncodingJSONValid (0.00s)
    (55 sub-tests PASS)
--- PASS: TestIsWellFormedJSON_AgreesWithEncodingJSONValid_OverManyGeneratedInputs (0.06s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.535s
```

**Green, guard included, after the fix:**

```
$ go test -race -run 'TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage' -v ./src/ai/...
--- PASS: TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage (0.04s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.367s
```

**Closed by showing it bite.** Scratch violation: a throwaway file `zz_scratch_bite.go` (`package ai; import _ "os"`) added, guard run, removed:

```
=== CONT  TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage
    import_boundary_test.go:292: the request path's dependency closure imports "os"
          rule: V-REQ-22: ... — no filesystem package
    import_boundary_test.go:292: the request path's dependency closure imports "io/fs"
          rule: V-REQ-22: ... — no filesystem package
--- FAIL: TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage (0.04s)
FAIL
```

Removed; the guard passed again immediately afterward (shown above).

**Why not scope the guard away from this instead.** The only way to avoid touching `tool_call.go` would have been to compute a narrower, file-level or call-graph-level dependency closure — but `go list -deps` is package-granular by construction, and `design.md` § 9.3 explicitly specifies "the AI-00.3 guard style", which is package-level `go list`. A narrower mechanism would be a second, heavier guard technology this milestone has no charter to introduce, for a problem the existing style already finds correctly.

### AI-10.4 evidence gate

```
$ go test -race -v ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.487s
ok  	github.com/cachicamas/backend/agent/src/ai	2.446s

$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.

$ cat go.mod
module github.com/cachicamas/backend/agent

go 1.26.3
```

`go.mod` carries zero `require` lines (unchanged — `math/rand/v2` and everything else touched in this phase is standard library); both AI-00 import guards pass; the new AI-10.4 guard passes non-vacuously.

---

## Phase AI-10.5 — whole-request round trip `[leaf]` — **IMPLEMENTED**

**Deliverable:** `backend/agent/src/agenttest/request_test.go` (new file).
**Spec:** `R-AMR-015`. **Design:** § 10.

- [x] ~~**Item 1** — An external-package test walks a request holding every part variant plus segments, tools and options, and reconstructs an **equal** request from what it read.~~
- [x] ~~**Item 2** *(pin)* — The walk's kind handling is exhaustive over `PartKinds()`: a kind added without a readable accessor fails this pin.~~

No production change: this phase's whole deliverable is the test file. Both items are pins — the round trip and its exhaustiveness are already true of the exported surface AI-10.1–AI-10.3 landed — closed by showing them bite, in the same idiom as AI-10.4.

### AI-10.5 item 2 — the exhaustiveness pin, landed first

`partKindReaders` — a `map[ai.PartKind]func(*testing.T, ai.Part) ai.Part`, one entry per registered kind — is the single source both items share: item 1's walk calls it to rebuild content, item 2 asserts it is complete against `ai.PartKinds()`. Landed first because item 1's walk depends on it.

**Green from birth**, with all four kinds present:

```
--- PASS: TestPartKindReaders_EveryRegisteredKind_HasAReader (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.439s
```

**Closed by showing it bite.** Scratch violation: the `ai.PartKindToolResult` entry deleted from `partKindReaders`.

```
=== CONT  TestPartKindReaders_EveryRegisteredKind_HasAReader
    request_test.go:95: ai.PartKind tool_result is registered in ai.PartKinds() but this round-trip walk's partKindReaders has no reader for it — a kind added without a readable accessor here fails this pin (R-AMR-015, S-AMR-056)
--- FAIL: TestPartKindReaders_EveryRegisteredKind_HasAReader (0.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/agenttest	0.282s
FAIL
```

Restored (diffed byte-identical against the pre-scratch file); re-ran green, shown above.

### AI-10.5 item 1 — the round trip

`buildFullRequest` constructs one request holding all four content-part kinds — text (user), reasoning with a round-trip token and a tool call (assistant), a tool result correlating to that call (tool) — plus two system segments, two declared tools, a specific tool choice, and all four generation options. `rebuildFromReadback` reads every region through the exported surface alone (never touching a field) and constructs a second request from what it read. `requireRequestsEqual` compares the two region-wise, message identity excluded per `design.md` § 11.2 — `Request.Equal` does not exist yet (AI-10.6 lands it, and depends on AI-10.5 per the milestone's own dependency graph), so this comparison is local to the test, not a call to production code.

**Green from birth:**

```
--- PASS: TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.439s
```

**Closed by showing it bite.** Scratch violation: the tool-call reader in `partKindReaders` rebuilt with a corrupted name (`call.Name()+"-SCRATCH-BITE-corrupted"`), proving the comparison is real rather than trivially satisfied:

```
=== CONT  TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest
    request_test.go:143: messages[1].content[1].ToolCall() = ("call-1", "search_flights-SCRATCH-BITE-corrupted"), want ("call-1", "search_flights")
--- FAIL: TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest (0.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/agenttest	0.283s
FAIL
```

Restored (diffed byte-identical against the pre-scratch file); re-ran green, both tests together:

```
--- PASS: TestPartKindReaders_EveryRegisteredKind_HasAReader (0.00s)
--- PASS: TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/agenttest	(cached)
```

**Refactor.** None.

### AI-10.5 evidence gate

```
$ go test -race ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.268s
ok  	github.com/cachicamas/backend/agent/src/ai	(cached)

$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

`go.mod` carries zero `require` lines; both AI-00 import guards and the AI-10.4 no-I/O guard pass.

---

## Phase AI-10.6 — immutability `[leaf]` — **IMPLEMENTED**

**Spec:** `R-AMR-016`. **Design:** § 11.

- [x] ~~**Item 1** — Mutating anything a reader returned leaves the request observably unchanged.~~
- [x] ~~**Item 2** — Mutating the values passed to the constructor leaves the request observably unchanged.~~
- [x] ~~**Item 3** — Two requests built from identical inputs compare equal by `design.md` § 11.2's documented equality, and neither is affected by operations on the other. Decide there whether to export `Equal`; the recorded default is yes.~~

This closes AI-10, the milestone's last leaf.

### AI-10.6 item 1 — mutating a readback, shown to bite

**Deliverable:** `request_test.go` gains one test, three sub-cases (messages, stop sequences, system-instruction segments). No production change: `Messages()`, `StopSequences()` and `Segments()` already clone.

**Green from birth:**

```
--- PASS: TestRequest_ReadbackMutation_LeavesTheRequestUnchanged (0.00s)
    --- PASS: .../messages (0.00s)
    --- PASS: .../stop_sequences (0.00s)
    --- PASS: .../system_instruction_segments (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.401s
```

**Closed by showing it bite, all three sub-cases at once.** Scratch violation: the three clone points removed together — `Messages()` returns `r.messages` directly, `StopSequences()` returns `r.options.stopSequences` directly, `SystemInstruction.Segments()` returns `i.segments` directly:

```
=== NAME  TestRequest_ReadbackMutation_LeavesTheRequestUnchanged/system_instruction_segments
    request_test.go:1135: after the reader reversed the segment slice it received, the request's first segment changed — the request handed out its own storage
=== NAME  TestRequest_ReadbackMutation_LeavesTheRequestUnchanged/messages
    request_test.go:1078: after the reader reversed the slice it received, request.Messages()[0] changed — the request handed out its own storage
=== NAME  TestRequest_ReadbackMutation_LeavesTheRequestUnchanged/stop_sequences
    request_test.go:1099: after the reader mutated the slice it received, request.StopSequences()[0] = "SCRATCH-MUTATED", want "</a>" — the request handed out its own storage
--- FAIL: TestRequest_ReadbackMutation_LeavesTheRequestUnchanged (0.00s)
    --- FAIL: .../system_instruction_segments (0.00s)
    --- FAIL: .../messages (0.00s)
    --- FAIL: .../stop_sequences (0.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.324s
FAIL
```

`request.go` and `system_instruction.go` restored (diffed byte-identical against the pre-scratch files); re-ran green, shown above.

### AI-10.6 item 2 — mutating a constructor input, shown to bite

**Deliverable:** `request_test.go` gains one test, two sub-cases (the message sequence, the stop-sequence slice). No production change: `NewRequest` already clones `messages` on the way in, and `WithStopSequences` already clones via `slices.Clone`. **This closes a real, previously-untested gap** — neither S-AMR-058-shaped scenario (mutate what was *passed to* the constructor, as opposed to what a reader *received back*) had a dedicated test before this leaf, despite the underlying clone already being landed by AI-10.1.

**Green from birth:**

```
--- PASS: TestNewRequest_ConstructorInputMutatedAfterConstruction_LeavesTheRequestUnchanged (0.00s)
    --- PASS: .../the_message_sequence (0.00s)
    --- PASS: .../the_stop_sequence_slice (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.313s
```

**Closed by showing it bite, both sub-cases at once.** Scratch violation: `NewRequest`'s `messages: slices.Clone(messages)` changed to `messages: messages`, and `WithStopSequences`'s `slices.Clone(sequences)` changed to `sequences`:

```
=== NAME  TestNewRequest_ConstructorInputMutatedAfterConstruction_LeavesTheRequestUnchanged/the_message_sequence
    request_test.go:1169: after the caller reversed the slice it passed to ai.NewRequest, request.Messages()[0] changed — NewRequest aliased the caller's storage instead of cloning it
=== NAME  TestNewRequest_ConstructorInputMutatedAfterConstruction_LeavesTheRequestUnchanged/the_stop_sequence_slice
    request_test.go:1188: after the caller mutated the slice it passed to ai.WithStopSequences, request.StopSequences()[0] = "SCRATCH-MUTATED", want "</a>" — WithStopSequences aliased the caller's storage instead of cloning it
--- FAIL: TestNewRequest_ConstructorInputMutatedAfterConstruction_LeavesTheRequestUnchanged (0.00s)
    --- FAIL: .../the_message_sequence (0.00s)
    --- FAIL: .../the_stop_sequence_slice (0.00s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/ai	0.315s
FAIL
```

`request.go` restored (diffed byte-identical against the pre-scratch file); re-ran green together with item 1, shown above.

### AI-10.6 item 3 — `Equal`, exported

**Deliverable:** `request.go` gains `Request.Equal` and `toolsEqual`; `message.go` gains `Message.Equal`; `system_instruction.go` gains `SystemInstruction.Equal`. `request_test.go` gains `buildEquatableRequest` and one test.

**The decision.** `design.md` § 11.2 records the default as **yes** and states the reason: AI-10.5's round trip needs it, AI-26 needs it, and a comparison every consumer re-derives is a comparison every consumer gets subtly wrong. No reason was found or recorded to change it, so it is implemented exactly as written — `Message` and `SystemInstruction` gain matching `Equal` methods so `Request.Equal` composes them rather than being one large function, per § 11.2's own instruction. `ToolSet`/`Tool`/`ToolChoice` do **not** gain new `Equal` methods: § 11.2 says tools and tool choice are compared "by their own accessors", and `Tool` cannot use `==` in any case — its `schema` field is a slice.

**Before red** — the compile failure, which is the state before red per `design.md` § 13:

```
$ go vet ./...
vet: src/ai/request_test.go:1245:12: first.Equal undefined (type ai.Request has no field or method Equal)
```

**Green**, directly — `Message.Equal` (role + `slices.Equal` over comparable `Part`s, per `content_part.go`'s documented `==`), `SystemInstruction.Equal` (`slices.Equal` over comparable `Segment`s), `Request.Equal` (every region via its own accessor, message identity excluded), and `toolsEqual` (a free function, name + description + `bytes.Equal` schema) landed together:

```
--- PASS: TestRequest_Equal_IdenticalInputsCompareEqualAndOperationsDoNotLeak (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.513s
```

**Triangulation, in the same test rather than a second one.** `first.Equal(second)` (identical inputs, independently minted message identities, proven different before the equality check) is the positive case; `first.Equal(third)` (a request carrying none of the optional regions) is the negative case, and both directions of both are asserted — `Equal` must be symmetric. Without the negative case, an `Equal` that always returned `true` would have passed.

**`bytes` was checked against the AI-10.4 no-I/O guard before landing**, since `request.go` is exactly the package that guard scopes:

```
$ go list -deps -f '{{.ImportPath}}' bytes | grep -x -E 'os|io/fs|net|net/http' || echo "(none - clean)"
(none - clean)

$ go test -race -run 'TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage' -v ./src/ai/...
--- PASS: TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage (0.03s)
PASS
```

**Refactor.** None.

### AI-10.6 evidence gate

```
$ go test -race ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.307s
ok  	github.com/cachicamas/backend/agent/src/ai	2.408s

$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

`go.mod` carries zero `require` lines; both AI-00 import guards and the AI-10.4 no-I/O guard pass.

---

## Evidence

_Filled at the close of each implemented phase, from observed command output._

### Guards

- `go.mod` must carry **zero requires**.
- Both AI-00 import guards must pass.
- `validation.go` is **unchanged by this half**; `ErrMisplaced` is AI-10.3's first task.

---

## Doc 0002 amendments

_None so far._ The revert-and-record clause is triggered only when a leaf's first red cannot be driven green in small steps. If it fires, the amendment lands in this change as `> **Amended 2026-07-31** …` with the superseded text struck through, and is reported prominently.

## Register amendments

**None made.** One gap is reported in `proposal.md` § "Vocabulary check": `V-REQ-26` names the generation-option concept and its admission test but enumerates no members. `design.md` § 8 carries the table an amendment would cite. `openspec/specs/ai-contract-vocabulary/spec.md` is **not edited by this change**.
