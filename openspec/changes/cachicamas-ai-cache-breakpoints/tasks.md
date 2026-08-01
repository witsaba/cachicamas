# Tasks — cache-boundary markers

> **Change**: `cachicamas-ai-cache-breakpoints`
> **Milestone**: AI-11 · **Nodes**: AI-11.1, AI-11.2, AI-11.3, all `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-01
> **Worktree**: `cachicamas-worktrees/ai-11` · **Branch**: `feat/ai-11-cache-breakpoints` · **Base**: `07d2027`
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-cache-breakpoints/spec.md`, `design.md`
> **Depends on**: AI-10 · **Blocks**: AI-24, AI-26.2
> **Evidence gate**: recorded green `make test` **and** clean `make lint` in `backend/agent/` (`make test` is `go test -race -v ./...`)

---

## Entry point for the resuming agent

**Nothing in this milestone is implemented.** Every box below is unchecked. All five planning artifacts are written and committed.

Start here, in this order:

1. **Run `explore.md` § 8 — the re-verification register.** Eight rows, each naming a file and a symbol from AI-10 that this plan assumes. AI-10.3 items 2–4, AI-10.4, AI-10.5 and AI-10.6 were landing concurrently in `../ai-wave-1` while this plan was written, so those rows are promises, not observations. **Do this before writing the first test.** A row that failed is a `revert-and-record` event under doc 0002's living-graph clause: record the discovered prerequisite in `explore.md` § 8 and in `design.md` § 10, then proceed against what is actually in the tree.
2. **Rebase or merge the completed AI-10** into this branch, and confirm `make test` and `make lint` are green **before** adding anything. A red baseline makes every later RED observation ambiguous.
3. Start at **§ Phase AI-11.1 item 1**, the walking skeleton.

Every decision this milestone needs is already made in `design.md` §§ 2–6. A decision may be changed only by recording the reason in `design.md` **before** writing the test; absent a recorded reason, implement what is written.

**This milestone appends no rule class.** `ErrOutOfRange` already cites `V-REQ-24` in its own GoDoc (`validation.go:81`), so `ruleClasses` and both of its registry mirrors stay untouched — unlike AI-08.2 and AI-10.3. If you find yourself editing `validation.go`'s sentinel block, stop and re-read `explore.md` § 3.3.

**Never edit `openspec/specs/ai-contract-vocabulary/spec.md`.** `V-REQ-23`, `V-REQ-24` and `V-REQ-25` are already written there and are this milestone's binding input. If a new term turns out to be needed, report it upward; the orchestrator appends it once.

## `explore.md` § 8 re-verification register — resolved

Run at apply time against the merged tree (base `1c4171e`, AI-10 complete) via `codegraph_explore`, cross-checked against direct `Read` of `request.go`, `message.go`, `tool.go`, `system_instruction.go`, `tool_set.go`, `tool_choice.go`, `validation.go`, `content_part.go`, `role.go`, `role_test.go`, `import_boundary_test.go`, `agenttest/request_test.go`.

| # | Assumed | Result | Detail |
| --- | --- | --- | --- |
| 1 | `Request` carries an optional tool set via `WithTools(ToolSet)` / `Request.Tools() (ToolSet, bool)` | **Held** | `request.go:334` `func WithTools(tools ToolSet) RequestOption`; `request.go:357` `func (r Request) Tools() (ToolSet, bool) { return r.options.tools, r.options.hasTools }`. Exact match. |
| 2 | `Request` carries an optional tool choice via `WithToolChoice` / `Request.ToolChoice()` | **Held** | `request.go:344,365`. Exact match. |
| 3 | The rule order is a `[]Rule` passed to `FirstFailure` in `NewRequest`, cap inserts at the last structural slot | **Held, refined** | It is a **variadic argument list of inline `func() *Violation` literals**, not a named slice — `FirstFailure(rules ...Rule) error`. Confirmed exactly 10 positional rules in this order: (1) model non-empty, (2) messages non-empty, (3) each message constructed, (4) each message's content (AI-06's), (5) system constructed-if-applied, (6) role-vs-content-kind (`ErrMisplaced`), (7) duplicate-tool-call (`ErrDuplicate`), (8) unresolved-tool-result (`ErrUnresolvedReference`), (9) tool-choice-vs-tool-set, (10) `draft.boundsRule()`. Duplicate-tool-call precedes orphan-tool-result, matching AI-10's own tasks.md. The cap rule is inserted as the **new 10th positional argument**, immediately before `draft.boundsRule()` (which becomes 11th). |
| 4 | `Request.Equal(other Request) bool` is exported | **Held** | `request.go:426`, exported, region-wise: `model` by `!=`, `Messages()` via `slices.EqualFunc(..., Message.Equal)`, `SystemInstruction()` via `SystemInstruction.Equal`, `Tools()` via `slices.EqualFunc(..., toolsEqual)` (an unexported free function in `request.go`), `ToolChoice()`, then the four generation options. |
| 5 | Equality includes marker state once markers exist | **Held only for `Segment`; corrected for `Tool` and `Message` — two escalation-shaped findings, both resolved in-scope, not escalated** | (a) `Segment` stays comparable, so `SystemInstruction.Equal`'s `slices.Equal(i.segments, other.segments)` picks up `cacheBoundary` "for free" via `==` — the orchestrator's predicted outcome, confirmed. (b) `Message.Equal` (`message.go`) and `toolsEqual` (`request.go`) are **hand-rolled, explicit field comparisons** (`m.role == other.role && slices.Equal(m.content, other.content)`; `a.Name()==b.Name() && ...`) — adding a struct field does **not** make them see it; this is not a defect in AI-10.6 (the field did not exist when it was written), it is the ordinary shape of a hand-rolled comparator. **Both are already-AI-11.1-owned files** (`message.go` per design.md's own file table; `request.go`'s `toolsEqual` is a natural extension of the same file AI-11.2 already modifies, and `R-ACB-004` is explicitly AI-11.1's requirement), so the fix landed as part of AI-11.1 rather than a workaround: `Message.Equal` now also compares `cacheBoundary`, and `toolsEqual` now also compares `IsCacheBoundary()`. Proven with a genuine RED: `TestRequest_MarkersDifferByOneMarker_AreNotEqualButIdenticalMarkersAreEqual/tool` failed before the `toolsEqual` fix and passed after. (c) **Second, independent finding, raised by the orchestrator mid-task and verified**: AI-10.5's whole-request round-trip pin (`agenttest/request_test.go`, `TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest`) never calls `Request.Equal` at all — it drives two **hand-written** helpers, `rebuildFromReadback` and `requireRequestsEqual`/`requireToolsEqual`/`requireMessagesEqual`, so "`Equal` sees markers for free" is necessary but not sufficient: that pin would stay green on a request that silently dropped every marker on rebuild. Appended to AI-11.1's test list as item 6 (below) and closed in the same leaf, editing `backend/agent/src/agenttest/request_test.go` — the shared test kit, not `openspec/changes/cachicamas-ai-model-request/`. |
| 6 | AI-10.5's round trip lives in `agenttest/`, driven from `PartKinds()` | **Held, plus the row-5(c) finding above** | `agenttest/request_test.go`; AI-11.3's translator is written independently per design.md and does not consume this walk. |
| 7 | `ErrMisplaced` and its registry mirrors are settled, `ruleClasses` stable | **Held** | `validation.go:131`: `ruleClasses = []error{ErrEmpty, ErrNotInVocabulary, ErrOutOfRange, ErrMalformed, ErrUnresolvedReference, ErrDuplicate, ErrMisplaced}` — 7 classes, `ErrDuplicate` (AI-08.2) and `ErrMisplaced` (AI-10.3) both present, `ErrOutOfRange` registered — confirms AI-11.2 appends no class. |
| 8 | `Segment` stays comparable; `Tool`/`Message` stay non-comparable with the named zero-detectors | **Held** | `Segment{text string}` (comparable); `Tool{name, description string; schema []byte}` (non-comparable, detector `Name()==""`); `Message{id MessageID; role Role; content []Part}` (non-comparable, detector `ID().IsZero()`). Adding `cacheBoundary bool` to all three keeps `Segment` comparable and does not change `Tool`/`Message`'s comparability. |

**Additional, load-bearing constraint surfaced mid-apply (not in the original register, recorded here for AI-11.2/.3):** `backend/agent/src/ai`'s production dependency closure is `errors sync sync/atomic io iter unicode unicode/utf8 bytes cmp slices strconv strings math/bits` — **no `fmt`**. Every rendering extension in this milestone (`Segment.String()`, `Request.String()`, `CacheRegion.String()`, `CacheBoundary.String()`) uses `strings.Builder`/`strconv` only, never `fmt.Sprintf`, matching `role.go`'s and `system_instruction.go`'s own idiom. `fmt` remains fine in `_test.go` files, which the dependency-closure guard excludes (`go list -deps` with no `-test`).

## Node types and what they close on

| Node | Type | Closes on |
| --- | --- | --- |
| AI-11.1, AI-11.2, AI-11.3 | `[leaf]` | Every test-list item taken red → green → refactored, **in order**, with both outputs recorded here |

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so `design.md` § 7 applies: land the narrowest thing that **compiles and fails**, record the failure, then implement. **A compile error is the state before red, not red.**

Discovered cases are **appended** to the owning leaf's test list, never substituted for a planned one and never pruned to fit the budget. The items marked *(appended)* below were discovered during planning and are already part of the contract.

**Commit per leaf.** Wave 1 recorded that per-leaf commits are what made a mid-milestone agent death cheap; a single end-of-milestone commit loses everything.

---

## Review Workload Forecast

| Field | Value |
| --- | --- |
| Estimated changed lines | ~1 000 Go (~330 production, ~670 test) plus ~1 100 prose in five SDD artifacts |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR — the wave PR, with the leaf boundary as the commit boundary |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

### Budget reassessment — split trigger 4 fires, and was forecast before any code

doc 0002's rule is *"prefer less than 250 changed lines; stop and reassess before 400"*, and split trigger 4 fires when a milestone's projected diff, tests included, pushes it past the review budget. **It does.** `explore.md` § 7 recorded it before a line of Go was written. The reassessment:

- **Split trigger 1 does not fire.** Three leaves, one publicly observable behaviour family: a marker's placement, its cap, its advisory status. Splitting them across PRs would ship a marker nobody can count and a cap over nothing.
- **The mitigation is the commit boundary, not a smaller test list.** doc 0002 puts the PR-chain boundary at the node boundary; three leaf commits make the chain reviewable in slices without rework.
- **Production is small; documentation and tests carry the weight.** The exported surface is one constant, one three-member vocabulary, one value type, three method pairs, and one accessor. What is large is the contract documentation, which is where this package puts its reasoning, and the tests.
- **The wave-level PR budget is 5 000+ lines and was accepted up front as `exception-ok`.** A forecast over budget is stated, not obeyed.
- **Nothing is cut to fit.**

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
| --- | --- | --- | --- | --- | --- |
| 1 — AI-11.1 | A marker can be placed on and read from all three carriers, and it changes nothing else | wave PR, commit 1 | `go test -race -run 'TestSegment_Cache\|TestTool_Cache\|TestMessage_Cache\|TestRequest_Marker' ./src/ai/` | N/A — a value-type contract with no runnable process; the cross-package harness is unit 3 | `git revert` commit 1: removes one field and two methods from each of three files |
| 2 — AI-11.2 | The cap is enforced and the cascade order is observable | wave PR, commit 2 | `go test -race -run 'TestRequest_CacheBoundar\|TestCacheRegion' ./src/ai/` | N/A — same reason | `git revert` commit 2: removes `cache_boundary.go` and one rule row |
| 3 — AI-11.3 | The advisory contract holds, with a control proving the assertion is not vacuous | wave PR, commit 3 | `go test -race ./src/agenttest/` | `make test` in `backend/agent/` — the whole-module run is the harness that proves the guards and both AI-00 import boundaries still hold | `git revert` commit 3: removes one test file; no production code |

---

## Phase AI-11.1 — markers on segments, tools and messages `[leaf]`

**Deliverable:** `system_instruction.go`, `tool.go`, `message.go`, `cache_boundary_test.go`.
**Spec:** `R-ACB-001` … `R-ACB-005`, `R-ACB-010`. **Design:** §§ 2, 3.1, 3.4.
**Depends on:** AI-10.

Test list — items 1–3 are doc 0002's, verbatim; 4–5 were appended during planning; item 6 was discovered and appended during apply.

- [x] **Item 1** — A system segment, a tool declaration and a message can each carry a cache-boundary marker; marker placement round-trips through construction and readback.
  - [x] 1.1 RED: `TestSegment_MarkCacheBoundary_RoundTripsThroughConstruction` in `ai_test` against a `MarkCacheBoundary` that returns the receiver unchanged. Assert marked, and text byte-equal. `S-ACB-001`. RED observed: `seg.MarkCacheBoundary undefined (type ai.Segment has no field or method MarkCacheBoundary)`.
  - [x] 1.2 GREEN: add `cacheBoundary bool` to `Segment`; implement `MarkCacheBoundary` and `IsCacheBoundary` (`design.md` § 3.1). GREEN observed: `--- PASS: TestSegment_MarkCacheBoundary_RoundTripsThroughConstruction`.
  - [x] 1.3 RED → GREEN: the same pair on `Tool` — name, description and schema bytes unchanged. `S-ACB-002`. RED: `tool.MarkCacheBoundary undefined`. GREEN: `--- PASS: TestTool_MarkCacheBoundary_RoundTripsThroughConstruction`.
  - [x] 1.4 RED → GREEN: the same pair on `Message` — role, ordered content **and identity** unchanged. `S-ACB-003`. RED: `msg.MarkCacheBoundary undefined`. GREEN: `--- PASS: TestMessage_MarkCacheBoundary_RoundTripsThroughConstruction`, plus regression `TestMessage_|TestRequest_Equal` green.
  - [x] 1.5 RED → GREEN: placement returns a **copy** — the value it was applied to still reports unmarked, on all three carriers. `S-ACB-004`. True by construction (value receivers) the moment it was written — `TestCarrier_MarkCacheBoundary_LeavesTheOriginalValueUnmarked` passed on first run. Closed by showing it bite per doc 0002's idiom: scratch violation `IsCacheBoundary() bool { return true }` on all three carriers → all three subtests failed (`original.IsCacheBoundary() = true ..., want false`) → reverted → green again.
  - [x] 1.6 RED → GREEN: a never-marked carrier reports not-a-boundary; marking twice is idempotent. `S-ACB-005`, `S-ACB-006`. `Tool` and `Message` are non-comparable, so compare field-wise (`design.md` § 3.1). Also true by construction; `TestCarrier_NeverMarkedOrMarkedTwice_ReportsCorrectly` passed on first run. Closed by showing it bite: scratch violation `s.cacheBoundary = !s.cacheBoundary` (toggle instead of set) on `Segment.MarkCacheBoundary` → `segment` subtest failed (`marking an already-marked segment again changed it`) → reverted → green again.
  - [x] 1.7 Refactor: the three method pairs are three spellings of one idea — confirmed: identical shape (`if <zero-detector> { return receiver }; receiver.cacheBoundary = true; return receiver`) and identical `IsCacheBoundary` shape on all three; each GoDoc states its own zero-value rule (`Segment.IsZero()`, `Tool.Name()==""`, `Message.ID().IsZero()`).

- [x] **Item 2** — Markers do not participate in validity: a marked request and its unmarked twin validate identically and are otherwise equal.
  - [x] 2.1 RED → GREEN: a valid request and its marked twin (within the cap) both construct successfully. `S-ACB-016`. True by construction (no rule reads `cacheBoundary` yet) — `TestRequest_MarkedTwinOfAValidRequest_BothConstructSuccessfully` green on first run.
  - [x] 2.2 RED → GREEN: an **invalid** request and its marked twin fail with the same rule class at the same rendered position. `S-ACB-017`. Same reason, green on first run: `TestRequest_MarkedTwinOfAnInvalidRequest_FailsWithTheSameRuleAtTheSamePosition`.
  - [x] 2.3 RED → GREEN: every placement combination across the three carriers, within the cap, constructs — there is no placement rule. `S-ACB-018`. `TestRequest_EveryMarkerPlacementCombination_Constructs`, all 8 combinations green on first run.
  - [x] 2.4 RED → GREEN: a marked segment of one character constructs — minimum cacheable prefix length is not decidable here. `S-ACB-019`. `TestRequest_MarkedSingleCharacterSegment_Constructs`, green on first run.
  - [x] 2.5 RED → GREEN: two requests identical but for one marker are **not** equal; two with identical markers **are**. `S-ACB-013`, `S-ACB-014`. Uses AI-10.6's `Equal` — see the resolved register row 5 above. **Genuine RED**, not a compile failure: `TestRequest_MarkersDifferByOneMarker_AreNotEqualButIdenticalMarkersAreEqual/tool` failed (`unmarked.Equal(markedOnce) = true for carrier "tool", want false`) because `toolsEqual` in `request.go` named `Name`/`Description`/`Schema` explicitly and did not see the new field. Fixed by adding `a.IsCacheBoundary() == b.IsCacheBoundary()` to `toolsEqual`. GREEN: all three carrier subtests (`segment`, `tool`, `message`) pass. `segment` and `message` subtests were green from the start — `Segment` via `==`-comparability, `Message` via the `Message.Equal` fix already landed in item 1.4's GREEN step.
  - [x] 2.6 RED → GREEN: a marked request read region by region and rebuilt is equal to the original — no marker lost in the round trip. `S-ACB-015`. `TestRequest_MarkedRequestReadAndRebuilt_IsEqualToTheOriginal`, green on first run (the local rebuild explicitly carries `.MarkCacheBoundary()` across).

- [x] **Item 3** — A marker is readable from an external package wherever it can be set — the adapter is the only consumer that will ever care.
  - [x] 3.1 RED → GREEN: an `ai_test` request built from marked segments, marked declarations and marked messages observes every marker at the ordinal it was placed on, through the **existing** region accessors only — no new accessor on `SystemInstruction` or `ToolSet`. `S-ACB-011`. `TestRequest_MarkersOnEveryOrdinal_AreObservableThroughExistingAccessors`, green on first run.
  - [x] 3.2 RED → GREEN: reading each region twice observes identical markers, because each read returns a fresh copy. `S-ACB-012`. `TestRequest_MarkersReadTwice_AreIdentical`, green on first run.

- [x] **Item 4** *(appended)* — A marker cannot make an unconstructed value usable.
  - [x] 4.1 RED → GREEN: `MarkCacheBoundary` on the zero `Segment`, zero `Tool` and zero `Message` returns the receiver; each is still rejected by its enclosing constructor with `ErrEmpty` at its index. `S-ACB-007` … `S-ACB-009`. `TestCarrier_MarkCacheBoundaryOnTheZeroValue_StillRejectedByItsEnclosingConstructor`, all 3 subtests green on first run — the `IsZero`/`Name()==""`/`ID().IsZero()` guards in item 1.2–1.4's `MarkCacheBoundary` already cover this.
  - [x] 4.2 RED → GREEN: none of the three marked zero values reports itself a cache boundary. `S-ACB-010`. `TestCarrier_MarkCacheBoundaryOnTheZeroValue_ReportsNotACacheBoundary`, green on first run.

- [x] **Item 5** *(appended)* — Marker state renders as structure and never as payload, and the unmarked rendering is unchanged.
  - [x] 5.1 RED → GREEN: a marked segment holding a secret renders through the default, string, extended and Go-syntax verbs naming the boundary and reproducing no secret. `S-ACB-038`. **Genuine RED**: `fmt.Sprintf("%v", marked) = "segment", want it to name the cache boundary` (×4 verbs) before `Segment.String()` was extended. GREEN after adding the `s.cacheBoundary` branch (using string concatenation, not `fmt` — production file).
  - [x] 5.2 RED → GREEN: an unmarked segment's rendering is byte-identical to before — `system_instruction_test.go:288` asserts `"segment"` exactly and must stay green. `S-ACB-039`. Confirmed green throughout (full-suite regression), and pinned directly under this leaf's own banner by `TestSegment_UnmarkedRendering_IsUnchangedFromBeforeThisMilestone`.

- [x] **Item 6** *(discovered during apply, appended — not substituted for any planned item)* — The AI-10.5 whole-request round-trip pin (`agenttest/request_test.go`) must itself preserve markers across a rebuild, because it never calls `Request.Equal` (register row 5(c) above): it drives hand-written `rebuildFromReadback` / `requireRequestsEqual` / `requireToolsEqual` / `requireMessagesEqual` helpers that, unextended, would report success on a request that silently dropped every marker. In scope for AI-11.1 because `R-ACB-004` (equality/round-trip preservation) is this leaf's own requirement and the file is the shared test kit (`backend/agent/src/agenttest/request_test.go`), not `openspec/changes/cachicamas-ai-model-request/` — the same reasoning AI-10.4 used reaching into `tool_call.go`'s own suite for a bite proof.
  - [x] 6.1 Marked `segmentTwo`, `bookTool` and `toolMessage` in `buildFullRequest`, one marked carrier per region; added `IsCacheBoundary()` assertions to `requireToolsEqual` and `requireMessagesEqual` (the segment leg already compares by `==` via `SystemInstruction.Equal`'s `slices.Equal`, so it needed no new assertion, only the marked fixture). **Genuine RED observed**: `messages[2].IsCacheBoundary() = false, want true`; `system instruction segments are not equal`; `tools[1].IsCacheBoundary() = false, want true` — `TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest` failed on all three carriers because `rebuildFromReadback` did not yet carry a marker across.
  - [x] 6.2 GREEN: extended `rebuildFromReadback` to call `.MarkCacheBoundary()` on the rebuilt segment/tool/message whenever the original reported `IsCacheBoundary()`. `--- PASS: TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest`; full `agenttest` package regression green.

- [x] **AI-11.1 close:** record green `make test` and clean `make lint`; commit `feat(ai): place cache-boundary markers on segments, tools and messages (AI-11.1)`.
  - `go test -race ./...` (from `backend/agent/`): `ok  	github.com/cachicamas/backend/agent/src/agenttest	1.506s` / `ok  	github.com/cachicamas/backend/agent/src/ai	2.247s`.
  - `make lint`: one `staticcheck` finding (`ST1023`, redundant explicit `ai.Segment` type on a `:=` I had added transiently to keep the import used before later tests needed it) — fixed by removing the explicit type once later tests made the import's use obvious; `make lint` → `0 issues.`
  - Both AI-00 import guards (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires`) and AI-10.4's guard (`TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage`) all green.

---

## Phase AI-11.2 — cap and ordering invariants `[leaf]`

**Deliverable:** `cache_boundary.go` (new), `request.go`, `cache_boundary_test.go`.
**Spec:** `R-ACB-006`, `R-ACB-007`, `R-ACB-010`. **Design:** §§ 2, 3.2, 3.3, 3.4, 4.
**Depends on:** AI-11.1.

Test list — items 1–2 are doc 0002's, verbatim; item 3 was appended during planning.

- [x] **Item 1** — WHEN a request's total marker count exceeds the documented cap THEN validation fails before I/O with an AI-04 sentinel naming the excess — the vendor cap is small and hard, and catching it client-side is the point of the seam.
  - [x] 1.1 RED: `TestRequest_CacheBoundariesOverCap_FailsWithOutOfRange` against a request one marker above `MaxCacheBoundaries`; assert `errors.Is(err, ErrOutOfRange)` and the position renders `cacheBoundaries`. `S-ACB-020`. RED observed: `undefined: ai.MaxCacheBoundaries`.
  - [x] 1.2 GREEN: create `cache_boundary.go` with `MaxCacheBoundaries = 4`, the shared walk `requestDraft.cacheBoundaries`, and `cacheBoundaryCapRule`; insert it as rule 10 of `NewRequest`'s slice (`design.md` § 3.3). Verify the landed rule-order table against the tree first — `explore.md` § 8 row 3. Landed the full design.md § 3.2 shape in one file (`CacheRegion`, `CacheBoundary`, `CacheRegions()`, `Request.CacheBoundaries()`) because the cap rule's own signature already needs `[]CacheBoundary`, not a bare count — items 2 and 3 below exercise the rest of that surface. GREEN: `--- PASS: TestRequest_CacheBoundariesOverCap_FailsWithOutOfRange`; full-package regression green.
  - [x] 1.3 RED → GREEN: exactly the cap is valid; no markers at all is valid and reports an empty sequence. `S-ACB-021`, `S-ACB-022`. `TestRequest_CacheBoundariesAtCapOrAbsent_IsValid`, green on first run (1.2's implementation already correct).
  - [x] 1.4 RED → GREEN: the cap is a **total**, not a per-region budget — exceeding it with markers drawn from all three regions fails identically. `S-ACB-023`. `TestRequest_CacheBoundariesOverCapAcrossRegions_FailsIdentically`, green on first run.
  - [x] 1.5 RED → GREEN: a request that both breaks an earlier structural rule and exceeds the cap reports the **earlier** rule, per `V-FAIL-04`. `S-ACB-024`. `TestRequest_EarlierStructuralViolationAndOverCap_ReportsTheEarlierRule`, green on first run.
  - [x] 1.6 RED → GREEN: `MaxCacheBoundaries` is readable as a constant without constructing anything. `S-ACB-025`. `TestMaxCacheBoundaries_IsReadableAsAConstant`, green on first run.
  - [x] 1.7 RED → GREEN: extend the landed `import_boundary_test.go` pattern so the request-validation closure is asserted to contain no network and no filesystem package — do not write a second guard. `S-ACB-026`. The existing `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` already scopes to `modulePath + "/src/ai"` as a whole, so it covers `cache_boundary.go` with **zero** new guard code, confirmed passing once the file landed. Closed by showing it bite (doc 0002's idiom): scratch violation — `import "io/fs"` + `var _ fs.FS` added to `cache_boundary.go` — → guard failed (`the request path's dependency closure imports "io/fs" ... no filesystem package`) → reverted → green again.

- [x] **Item 2** — An adapter can read markers **in tools → system → messages order** regardless of the order in which they were set, because that is the order the invalidation cascade runs in.
  - [x] 2.1 RED: `TestRequest_CacheBoundaries_YieldCascadeOrderRegardlessOfPlacementOrder` — place the message marker first, then the system marker, then the tool marker; assert the returned sequence is tools, system, messages. `S-ACB-027`. **Blocking assumption resolved**: `Request.Tools()` — register row 1, held exactly as assumed, no blocker. Green on first run (1.2 already landed the full walk).
  - [x] 2.2 GREEN: implement `CacheRegion`, its three constants from `iota + 1`, `CacheRegions()`, `String()`, `CacheBoundary` with `Region()`/`Index()`/`String()`, and `Request.CacheBoundaries()` delegating to the shared walk (`design.md` § 3.2). Landed together with 1.2, per the note there.
  - [x] 2.3 RED → GREEN: two markers in one region appear in ascending ordinal order. `S-ACB-028`. `TestRequest_TwoMarkersInOneRegion_AppearInAscendingOrdinalOrder`, green on first run.
  - [x] 2.4 RED → GREEN: indexing a boundary's region accessor at its ordinal yields a carrier that reports it is a boundary. `S-ACB-029`. `TestRequest_CacheBoundary_ResolvesBackToTheCarrierItMarks`, green on first run.
  - [x] 2.5 RED → GREEN: the same request constructed many times in one process yields an identical sequence every time — no map decides anything. `S-ACB-030`. `TestRequest_CacheBoundaries_AreDeterministicAcrossManyConstructions`, 100 runs, green on first run.

- [x] **Item 3** *(appended)* — The listing and the enforced count come from one walk, and the region vocabulary is exhaustive.
  - [x] 3.1 RED → GREEN *(pin)*: `CacheRegions()` holds exactly three members whose numeric order is tools, system, messages; a member declared without an entry in the enumeration fails the pin. Copy `role_test.go`'s constant-space walk — **not** a walk over the name table. `S-ACB-031`. `TestCacheRegions_DeclaredVocabulary_IsExhaustivelyTabulatedInCascadeOrder`, green on first run. Closed by showing it bite: scratch member `cacheRegionScratchBite` declared without a `cacheRegionNames` entry → `ai.CacheRegions() = [tools system messages cacheregion(4)]` failed the assertion → reverted → green again.
  - [x] 3.2 RED → GREEN: for a request at the cap, the reported sequence length equals the count the cap rule enforced. `S-ACB-032`. `TestRequest_AtTheCap_ReportedSequenceLengthEqualsTheEnforcedCount`, green on first run — true by construction of the one-walk design.
  - [x] 3.3 RED → GREEN: a request carrying markers renders its boundary count and no payload through all four verbs; a request with no markers renders as before. `S-ACB-040`. **Genuine RED**: `fmt.Sprintf("%v", request) = "request(model, 1 messages, system(2 segments))", want it to contain "2 cache boundaries"` (×4 verbs) before `Request.String()` was extended. GREEN after adding the `len(r.CacheBoundaries())` clause (`strings.Builder`/`strconv`, no `fmt` — production file).
  - [x] 3.4 Refactor: confirm `Request.CacheBoundaries()` and the cap rule call the **same** function and that neither re-walks a region. Confirmed by inspection: `Request.CacheBoundaries()` is `r.options.cacheBoundaries(r.messages)`; `cacheBoundaryCapRule` is `len(d.cacheBoundaries(messages)) > MaxCacheBoundaries` — identical unexported method, no second walk anywhere in `cache_boundary.go`.

- [x] **AI-11.2 close:** record green `make test` and clean `make lint`; commit `feat(ai): cap and order the request's cache boundaries (AI-11.2)`.
  - `go test -race ./...`: `ok  	github.com/cachicamas/backend/agent/src/agenttest	(cached)` / `ok  	github.com/cachicamas/backend/agent/src/ai	2.264s`.
  - `make lint`: `0 issues.`
  - Both AI-00 import guards and AI-10.4's dependency-closure guard green.

---

## Phase AI-11.3 — advisory semantics `[leaf]`

**Deliverable:** `backend/agent/src/agenttest/cache_boundary_test.go`.
**Spec:** `R-ACB-008`, `R-ACB-009`. **Design:** § 6.
**Depends on:** AI-11.2.

Test list — items 1–2 are doc 0002's, verbatim; item 3 was appended during planning.

- [ ] **Item 1** — WHEN a translator ignores every marker THEN the request is still fully translatable and semantically unchanged — an adapter for an auto-caching provider is conformant while ignoring them entirely.
  - [ ] 1.1 RED → GREEN: write the **marker-blind** translator in `agenttest` — it reads model, system segment text, declaration name/description/schema, message role and content, tool choice and every generation option through the exported surface, and never calls `IsCacheBoundary` or `CacheBoundaries`. Assert `render(marked) == render(unmarked)`. `S-ACB-033`.
  - [ ] 1.2 RED → GREEN: the blind rendering of the marked request is complete relative to the unmarked twin — no region silently dropped from both. `S-ACB-035`.

- [ ] **Item 2** *(pin)* — The usage-side surface is untouched by this milestone — the cache token fields exist from AI-13.3 and this milestone adds request-side expression only.
  - [ ] 2.1 GREEN-from-birth pin: `Usage`'s cache-read and cache-write counts read and validate exactly as before, at the same positions. `S-ACB-036`.
  - [ ] 2.2 GREEN-from-birth pin: this layer exports no hit-rate, cache-efficiency or cache-statistics accessor. `S-ACB-037`.

- [ ] **Item 3** *(appended)* — The control that proves item 1 is not vacuous.
  - [ ] 3.1 RED → GREEN: the **marker-aware** control — the same walk plus one marker read — renders the marked request and its unmarked twin **differently**. Without it, a blind translator that rendered nothing would satisfy item 1 perfectly. `S-ACB-034`. This is AI-06's `testdata/handrolled` + `testdata/constructed` lesson applied to a rendering.

- [ ] **AI-11.3 close:** record green `make test` and clean `make lint`; commit `test(ai): prove cache-boundary markers are advisory (AI-11.3)`.

---

## Milestone close

- [ ] `make test` green in `backend/agent/` — paste the transcript.
- [ ] `make lint` clean in `backend/agent/` — paste the transcript. **Run it before every commit**, not only at the end.
- [ ] `go.mod` still at **zero requires**.
- [ ] Both AI-00 import guards pass.
- [ ] `openspec/specs/ai-contract-vocabulary/spec.md` unmodified — confirm with `git diff --stat`.
- [ ] `validation.go`'s `ruleClasses` and both registry mirrors unmodified — confirm with `git diff --stat`.
- [ ] Record the **actual** changed-line count against `explore.md` § 7's forecast in the Review Workload Forecast table above.
- [ ] Record any `explore.md` § 8 row that failed re-verification, and what was done instead.
- [ ] Never push, never merge, never open a PR, never `git stash`.
