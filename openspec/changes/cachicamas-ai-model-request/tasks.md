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

This half of the milestone implements **AI-10.1 and AI-10.2 only**. **AI-10.3, AI-10.4, AI-10.5 and AI-10.6 are not implemented.**

The resuming agent starts at **§ Phase AI-10.3**, below. Every box in phases AI-10.3 … AI-10.6 is unchecked, and every decision those phases need is already made in `design.md` §§ 5–12 and marked `[provisional]` there. A `[provisional]` decision may be changed only by recording the reason in `design.md` **before** writing the test; absent a recorded reason, implement what is written.

The first thing AI-10.3 does is append `ErrMisplaced` to `validation.go` **and both of its registry mirrors in the same commit**. `validation_registry_internal_test.go` fails and says so if only one mirror moves.

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
| SDD planning artifacts | `spec.md`, `design.md`, `tasks.md` (on top of the landed `explore.md`, `proposal.md`) | ~900 prose | _filled at close_ | Low | 35 min |
| AI-10.1 walking skeleton | `request.go`, `request_test.go` | ~200 Go prod, ~400 Go test | _filled at close_ | **High** — every later request milestone and every adapter inherits the shape | 45 min |
| AI-10.2 segmented system instruction | `system_instruction.go`, `system_instruction_test.go`, `request.go`, `request_test.go` | ~120 Go prod, ~300 Go test | _filled at close_ | Medium — the shape AI-11.1 attaches markers to | 30 min |
| AI-10.3 … AI-10.6 | not implemented in this half | ~350 Go prod, ~900 Go test | — | Medium-High | ~90 min |

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

- [ ] **Item 1** — WHEN a request carries a system instruction as **ordered segments** THEN segment order and content round-trip exactly.
- [ ] **Item 2** — The single-segment convenience path produces a request **indistinguishable** from one built segment-by-segment with one segment.
- [ ] **Item 3** — An absent system instruction is legal and **distinguishable from one empty segment**.
- [ ] **Item 4** — Segment construction rules fail through AI-04 sentinels (empty segment, whitespace-only segment).

---

## Phase AI-10.3 — messages, tools and tool choice on the request `[leaf]` — **NOT IMPLEMENTED**

**Deliverable:** `request.go` extended; `validation.go` plus **both** registry mirrors; `request_test.go` extended.
**Spec:** `R-AMR-008` … `R-AMR-012`. **Design:** §§ 5, 6, 7, and § 4 rows 6–9.

- [ ] **Item 0** *(prerequisite, do this first)* — Append `ErrMisplaced` to `validation.go`'s class set **and** to `ruleClasses`, **and** update `validation_registry_internal_test.go` and `validation_test.go` in the **same commit**. The internal guard fails and says so if only one mirror moves. `design.md` § 5.3 carries the doc comment to write.
- [ ] **Item 1** — Message order and intra-message content order are preserved exactly through construction and readback.
- [ ] **Item 2** — The tool set and tool choice attach to the request, and AI-08.3's cross-validation runs at the request boundary too. **Call `ToolChoice.ValidateAgainst(ToolSet)`; reimplement none of its three rules.**
- [ ] **Item 3** — Role-versus-content-kind rules are enforced from `design.md` § 5.1's table, all twelve cells, in both directions, reporting `ErrMisplaced`.
- [ ] **Item 4** — An orphan tool result fails with `ErrUnresolvedReference`; an orphan tool **call** succeeds; a duplicate call identity fails with `ErrDuplicate` at the second occurrence; a result appearing before its call **succeeds**, pinning `design.md` § 6.3's deliberate non-decision.

---

## Phase AI-10.4 — validation happens once, before I/O `[leaf]` — **NOT IMPLEMENTED**

**Spec:** `R-AMR-013`, `R-AMR-014`. **Design:** § 9.

- [ ] **Item 1** — The first failure in the documented order is reported, identically across runs.
- [ ] **Item 2** — Validation is total over the regions.
- [ ] **Item 3** — The request path's dependency closure contains no network and no filesystem package, asserted mechanically in the AI-00.3 guard style. Extend `import_boundary_test.go` rather than adding a parallel guard.

---

## Phase AI-10.5 — whole-request round trip `[leaf]` — **NOT IMPLEMENTED**

**Deliverable:** a new test file under `backend/agent/src/agenttest/`.
**Spec:** `R-AMR-015`. **Design:** § 10.

- [ ] **Item 1** — An external-package test walks a request holding every part variant plus segments, tools and options, and reconstructs an **equal** request from what it read.
- [ ] **Item 2** *(pin)* — The walk's kind handling is exhaustive over `PartKinds()`: a kind added without a readable accessor fails this pin.

---

## Phase AI-10.6 — immutability `[leaf]` — **NOT IMPLEMENTED**

**Spec:** `R-AMR-016`. **Design:** § 11.

- [ ] **Item 1** — Mutating anything a reader returned leaves the request observably unchanged.
- [ ] **Item 2** — Mutating the values passed to the constructor leaves the request observably unchanged.
- [ ] **Item 3** — Two requests built from identical inputs compare equal by `design.md` § 11.2's documented equality, and neither is affected by operations on the other. Decide there whether to export `Equal`; the recorded default is yes.

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
