# Tasks — request extension points

> **Change**: `cachicamas-ai-request-extension-points`
> **Milestone**: AI-12 · **Nodes**: AI-12.1 … AI-12.4, all `[leaf]`
> **Phase**: tasks · **Project**: cachicamas (witsaba) · **Date**: 2026-08-01
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-request-extension-points/spec.md`, `design.md`
> **Worktree**: `cachicamas-worktrees/ai-12` · **Branch**: `feat/ai-12-request-extension-points` (planned against `07d2027`; **rebased onto finished Wave 1 head `1c4171e`**)
> **Depends on**: AI-10, AI-11 · **Blocks**: AI-24, AI-26.7, Layer 2's pre-request hook
> **Evidence gate**: recorded green `make test` **and** clean `make lint` in `backend/agent/` (`make test` is `go test -race -v ./...`)

---

## Entry point for the resuming agent

**Nothing is implemented.** No Go file has been touched; every leaf box below is unchecked.

**§ Phase 0 is partly done.** It ran on 2026-08-01 against finished Wave 1 head `1c4171e`. **Rows 1–6 of `design.md` § 13 are resolved** — read them there, not here, and do **not** re-derive them. **Rows 7 and 8 remain open** because AI-11 is still being implemented in `../ai-11`; both carry a decided branch and a two-step mechanical decision procedure (`design.md` § 13.2). **Task 0.4 — the pre-edit baseline — has not run and is yours.**

Read `design.md` § 13 first. It is now a resolved register rather than a checklist, and the three surprises it records — the parameter-reading rules, the swapped rule ordinals, and the `agenttest` round-trip blind spot — each moved work into a leaf below.

Then work **§ Phase AI-12.1 → AI-12.2 → AI-12.3 → AI-12.4**, in that order. AI-12.2 and AI-12.3 are declared parallel by doc 0002 and are genuinely disjoint — one touches the generation-option draft fields, the other adds a new file — so a single agent may swap them, but neither may start before AI-12.1 is green.

Every decision the phases need is already made in `design.md` §§ 4–8 and marked `[provisional]`. **A `[provisional]` decision may be changed only by recording the reason in `design.md` before writing the test**; absent a recorded reason, implement what is written.

The first thing AI-12.1 does is the § 2.2 extraction — moving `NewRequest`'s rule list to `draft.rules()`. That is the one place this milestone touches lines AI-10.3 is writing. **Rebase onto the advancing Wave 1 head; never merge, never push, never `git stash`** (the stash stack is shared across worktrees — Engram #2292).

## Node types and what they close on

| Node | Type | Closes on |
| --- | --- | --- |
| AI-12.1 … AI-12.4 | `[leaf]` | Every test-list item taken red → green → refactored, **in order**, with both outputs recorded in this file |

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so `design.md` § 11 applies: land the narrowest thing that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

Test lists below are **doc 0002 lines 748–797 reproduced faithfully**. Discovered cases are **appended** to the owning leaf's list, never substituted for a planned one and never pruned to fit the budget. Items marked *(appended)* were added by this plan; items marked *(pin)* are green-from-birth regression assertions, exempt from red-first and still fully mechanical.

Commit per leaf (Engram #2292: per-leaf commits are what make an agent death cheap). `make lint` runs **before every commit** — AI-06.2 landed with it failing, which is the recorded reason this line exists.

---

## Review Workload Forecast

**Revised 2026-08-01 by the Phase 0 re-verification gate.** Three resolved facts moved the forecast: the `draft.freeze()` extraction (§ 2.2.1), the `agenttest` round-trip obligation (§ 7.2, one new file), and the four-rows-not-one totality table (`R-REX-002`). Nothing was removed. The delivery strategy is unchanged.

| Field | Value |
| --- | --- |
| Estimated changed lines | **~1155 Go** (~325 production, ~830 test) + ~1600 markdown planning artifacts | 
| Previous estimate | ~1090 Go + ~1250 markdown |
| 400-line budget risk | High *(unchanged)* |
| Chained PRs recommended | No *(unchanged)* |
| Suggested split | Single PR — four commits, one per leaf |
| Delivery strategy | `exception-ok` *(unchanged; accepted up front at wave level)* |
| Chain strategy | `size-exception` *(unchanged)* |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Per-slice forecast

| Slice | Files | Forecast prod | Forecast test | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `spec.md`, `design.md`, `tasks.md` | — | ~1600 prose | Low | 50 min |
| AI-12.1 rebuild | `request.go`, `request_test.go` | ~145 *(+15: `draft.freeze()`)* | ~355 *(+25: four option rows not one, plus `S-REX-053`)* | **High** — every later request milestone and Layer 2's hook inherit the derive shape | 50 min |
| AI-12.2 per-request options | `request.go`, `request_test.go` | ~15 | ~135 | Low — the mechanism is AI-10's last-wins, reached through AI-12.1 | 15 min |
| AI-12.3 pass-through | `request_extension.go`, `request_extension_test.go`, `request.go`, **`agenttest/request_test.go`** | ~165 | ~270 *(+20: the two `agenttest` walks, `S-REX-054`)* | Medium-High — the shape AI-24's first adapter reads, and the leaf that repairs the round-trip pin's blind spot | 45 min |
| AI-12.4 determinism | `request_extension_test.go`, `request_test.go` | 0 | ~70 | Low | 10 min |

**Two contingencies still open** (AI-11, `design.md` § 13.2), neither changing the strategy: row 7 branch B adds ~15 prod + ~20 test to AI-12.1; row 8 branch A adds ~15 test to AI-12.2, branch B adds one recorded line.

### Budget reassessment — split trigger 4 fires, and it is stated rather than acted on

doc 0002's milestone rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when a milestone's projected diff, tests included, pushes it past the review budget. **It does, by roughly 2.7×, and `explore.md` § 9 recorded it before a line of Go existed.** The reassessment:

- **The wave-level PR budget is 5000+ lines and the user accepted it up front (`exception-ok`).** A forecast over budget therefore states a fact; it does not stop the work and does not require a new decision before apply.
- **Split trigger 1 does not fire.** Each leaf is one publicly observable behavior with a 1–4 item list. The milestone is four leaves, not one oversized one.
- **The mitigation is the leaf boundary as the commit boundary**, which is where doc 0002 puts the PR-chain boundary anyway. The chain exists in history and needs no rework to be reviewed in four slices.
- **Production is small; documentation and tests carry the weight.** The exported surface added is two option constructors, one method, one value type and two accessors. What is large is the contract documentation — where the reasoning is put in front of the person who adds the next region — and the tests.
- **Nothing is cut to fit.** Cases discovered during implementation are appended.

### Suggested work units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
| --- | --- | --- | --- | --- | --- |
| 1 | AI-12.1 — rebuild is total and copy-on-write | wave PR, commit 1 | `go test -race -run 'TestRequest_(Rebuild\|Derive)' ./src/ai/` | N/A — Layer 1 is a pure in-memory contract with no runnable surface until AI-24's transport | `request.go` additions + `request_test.go` additions; revert the commit |
| 2 | AI-12.2 — per-request overrides and one validation path | wave PR, commit 2 | `go test -race -run 'TestRequest_Override' ./src/ai/` | N/A — same reason | Same two files; independent of unit 3 |
| 3 | AI-12.3 — the typed-opaque namespaced pass-through | wave PR, commit 3 | `go test -race -run 'TestProviderExtension' ./src/ai/` | N/A — proven with fake translators in `ai_test`; a real adapter is AI-24's | `request_extension.go` + its test, deletable whole; one field and one rule row in `request.go` |
| 4 | AI-12.4 — read-back determinism | wave PR, commit 4 | `go test -race -count=1 -run 'Determinis' ./src/ai/` | N/A — same reason | Test-only; revert the commit |

Full gate before every commit, in `backend/agent/`: `make lint && make test`.

---

## Phase 0 — re-verification gate *(not a leaf; no test, no commit of its own)*

- [x] 0.1 Rebased onto the finished Wave 1 head **`1c4171e`** (AI-04 … AI-10 complete and green). AI-10 is landed; there is no `../ai-wave-1` work left to wait for.
- [x] 0.2 Walked all eight rows of `design.md` § 13. **Rows 1–6 resolved**; outcomes recorded in the table below and, with the signatures read, in `design.md` § 13.1 and `explore.md` § 6.1.
- [x] 0.3 **Resolved 2026-08-01, apply agent.** This worktree is rebased onto Wave 1 head `66e960d`, which carries AI-11 complete and green. Ran `design.md` § 13.2's two-step decision procedure against the landed surface:
  1. Read `system_instruction.go`, `tool.go`, `message.go` for a marker field: each carries an unexported `cacheBoundary bool` field plus `MarkCacheBoundary()`/`IsCacheBoundary()` methods (`system_instruction.go:15,66–78`; `tool.go:24,174–196`; `message.go:116,180–204`). **Row 7 → branch A.** Markers ride on the values and are reachable transitively through `WithSystemInstruction`, `WithTools` and `WithMessages`. AI-12 adds no option and no production line for markers; task 1.7 records "transitive" against the § 8 totality table.
  2. Read `NewRequest`'s `FirstFailure` argument list (`request.go:180–234`) for a cap rule: it now holds **eleven** positional rules, with `draft.cacheBoundaryCapRule(messages)` at position **10** and `draft.boundsRule()` at position **11** — `cache_boundary.go:156–158` documents `cacheBoundaryCapRule` itself as "rule 10 of NewRequest's documented order". **Row 8 → branch A.** The cap is a `Rule` in the list; the rebuild re-runs it for free. AI-12.2 appends one derive-path assertion (item 5 / task 2.5) and adds no rule for `V-REQ-24`, which stays AI-11's.
  3. The two rows are independent, as § 13.2 predicted, and both took branch A. Neither branch removed a planned item.
  - **Ordinal correction, made here and in `design.md` § 3**: because the cap already occupies row 10 and `boundsRule` occupies row 11, AI-12.3's own extension rule appends as row **12** — not "11 or 12" as `design.md` § 3 read before this update, and not row 11 as this file's own task 1.1 annotation assumed before AI-11 landed. `design.md` § 3's table and its "Correction 2" note are corrected accordingly; task 3.2 below already anticipated row 12 and needed no change.
- [x] 0.4 **Baseline recorded 2026-08-01, apply agent**, on the rebased tree, before any AI-12 edit:
  - `make test` (from `backend/agent/`, `go test -race -v ./...`): `ok  	github.com/cachicamas/backend/agent/src/agenttest` and `ok  	github.com/cachicamas/backend/agent/src/ai` — full suite green.
  - `make lint` (from `backend/agent/`): `go vet ./...` clean; `golangci-lint run --config=.golangci.yml ./...` → `0 issues.`
  - This is the safety net every leaf below is measured against, so the first RED in AI-12.1 is unambiguous.

### 0.2 outcome table

| Row | Check | Outcome |
| --- | --- | --- |
| 1 | `tools` / `toolChoice` are `RequestOption`s | **Matches** — `WithTools` (`request.go:334`), `WithToolChoice` (`:344`). No AI-12 work |
| 2 | cross-region rules read draft state | **Differs — branch taken.** Seven of ten rules read a `NewRequest` parameter. The three cross-region free functions need no edit; only two call sites move. `design.md` § 2.2 revised |
| 3 | rule order is a literal slice | **Differs in form, holds in substance.** A variadic argument list. **Rows 7/8 of the order were recorded backwards** and **there is no row 11**. `design.md` § 3 corrected twice |
| 4 | `Request.Equal` is exported | **Matches** (`request.go:426`). The "compare in `ai_test`, export nothing" branch is **closed, not taken** — task 3.6 keeps its wording |
| 5 | extend `Equal` to the extension region | **Confirmed, and insufficient.** The exact block is `design.md` § 7.1. The recorded rationale was wrong: AI-10.5's round-trip pin never calls `Equal`. `design.md` § 7.2 adds the `agenttest` obligation → new task 3.6a |
| 6 | `Reasoning.Token()` is `([]byte, bool)`, byte-exact | **Matches** (`reasoning_content.go:306`). No AI-12 work |
| 7 | where markers live | **Resolved 2026-08-01 by task 0.3 — branch A.** Markers ride on `Segment`/`Tool`/`Message`, reached transitively. No AI-12 work |
| 8 | breakpoint cap is a `Rule` in the list | **Resolved 2026-08-01 by task 0.3 — branch A.** `cacheBoundaryCapRule` is row 10; the rebuild re-runs it for free. AI-12.2 item 5 / task 2.5 asserts it |

**Three facts verified outside the eight rows:**

- **The `WithModel`/`WithMessages` central decision survives.** `requestDraft` carries no `model` and no `messages` field, so a `RequestOption` provably cannot reach either region (`design.md` § 13.1).
- **The system region is stored twice on `Request`**, and a careless `With` reverts it. Mitigated structurally by extracting `draft.freeze()` — new task 1.1a, `design.md` § 2.2.1.
- **AI-10.4's dependency-closure guard is confirmed and AI-12 needs nothing new.** The closure holds no `fmt`, no `os`, no `encoding/json`; everything `ProviderExtension` needs is already in it. Two obligations follow, in `design.md` § 5.1: no encoder is ever required to carry the payload, and every rendering in `request_extension.go` uses `strings.Builder`/`strconv` and **never** `fmt.Sprintf` — one `fmt` call in a non-test file of this package turns a landed guard red.

### 0.5 Finding recorded for the **Wave 1 verify phase** — not fixed here

**`S-AMR-046` in AI-10's delta spec is wrong by one word.** `openspec/changes/cachicamas-ai-model-request/specs/ai-model-request/spec.md` says *"Given each of the **four** permitted cells in turn […] twelve cells total"*. `R-AMR-011`'s own table is 4 kinds × 3 roles = 12 cells with **five** permitted: `text`/user, `text`/assistant, `reasoning`/assistant, `tool_call`/assistant, `tool_result`/tool. The landed `rolePermittedKinds` (`request.go:30–34`) agrees — 1 + 3 + 1 = **five**.

**The discrepancy is real**, cosmetic, and confined to scenario prose; the table, the "twelve cells total" count and the implementation are all correct. **AI-12 does not edit it.** It belongs to `cachicamas-ai-model-request`, which this change is forbidden to touch, and fixing another change's spec from here would land the correction outside the review that owns it. Carry it to Wave 1 verify.

---

## Phase AI-12.1 — copy-on-write rebuild `[leaf]`

**Deliverable:** `backend/agent/src/ai/request.go`, `backend/agent/src/ai/request_test.go`.
**Spec:** `R-REX-001`, `R-REX-002`, `R-REX-003`. **Design:** §§ 1, 2.1, 2.2, 4, 8.
**Depends on:** AI-11 (doc 0002) — via the marker row of item 2.

- [x] **Item 1** — WHEN a caller derives a modified request from an existing one THEN the original is observably unmodified (deep comparison before and after) and the derived request validates independently. `TestRequest_DeriveWithChangedOption_LeavesTheOriginalUnmodified`.
- [x] **Item 2** — Deriving is **total**: every region — system segments, messages, tools, tool choice, options, markers — is reachable by the rebuild path. `TestRequest_TotalityOfTheRebuildPath_EveryRegionIsReachable`.
- [x] **Item 3** *(pin)* — A request carrying reasoning round-trip tokens rebuilds with every token byte-identical. `TestRequest_DeriveWithUnrelatedChange_PreservesOpaquePayloadsByteIdentically` (extension-value sibling deferred to AI-12.3, task 3.10).
- [x] **Item 4** *(appended)* — The model identity is reachable by the rebuild path, and `WithModel` is last-wins over the parameter. `TestNewRequest_WithModelBesideTheParameter_LastApplicationWins`, `TestRequest_DeriveReplacingModelOrMessages_TheDerivedRequestCarriesTheReplacement`.
- [x] **Item 5** *(appended)* — A failed derivation returns the zero request and leaves the source unmodified; no-option derive equals the source. `TestRequest_DeriveEdgeCases_NoOptionsSucceedsAndAFailedDeriveLeavesTheSourceUnmodified`.
- [x] **Item 6** *(appended)* — Copies on the way in. `TestRequest_DeriveOptionInputMutatedAfterDeriving_LeavesTheDerivedRequestUnchanged`.

### Ordered work

- [x] 1.1 **Extraction first.** *(Combined with 1.1a and 1.2 into one behaviour-preserving refactor — see the note below the table.)* Moved `NewRequest`'s `FirstFailure` argument list to `func (d requestDraft) rules() []Rule`, order preserved exactly. **Correction to this line's own annotation**: the landed list at the rebased head is **eleven** rule expressions, not "nine inline closures plus `draft.boundsRule()`" — AI-11 had already inserted `draft.cacheBoundaryCapRule(messages)` as row 10 (task 0.3). `FirstFailure(d.rules()...)` is still a drop-in.
- [x] 1.1a **Extract the freeze too.** Added `func (d requestDraft) freeze() Request`; `NewRequest` and `With` both end `FirstFailure(draft.rules()...); return draft.freeze(), nil`.
- [x] 1.2 **Widen the draft.** Added `model string` and `messages []Message` to `requestDraft`; `NewRequest` seeds them from its parameters; every rule that read a parameter now reads `d.model`/`d.messages`. The three free functions needed no edit, only their two call sites moved.
- [x] 1.3 RED → GREEN **item 1**: `TestRequest_DeriveWithChangedOption_LeavesTheOriginalUnmodified`, two sub-cases (scalar option, slice-typed option) — capture every region before, derive, compare after (`S-REX-001`, `S-REX-002`).
- [x] 1.4 RED → GREEN `Request.With(opts ...RequestOption) (Request, error)` — copy `r.options`, apply, `FirstFailure(d.rules()...)`, freeze.
- [x] 1.5 RED → GREEN **item 4**: `WithModel`, `WithMessages`, and the last-wins pin `TestNewRequest_WithModelBesideTheParameter_LastApplicationWins` (`S-REX-007`), plus `TestRequest_DeriveReplacingModelOrMessages_TheDerivedRequestCarriesTheReplacement` (`S-REX-007`, `S-REX-008`).
- [x] 1.6 RED → GREEN **item 2**: `TestRequest_TotalityOfTheRebuildPath_EveryRegionIsReachable`, 10 rows (`design.md` § 8.1's landed set: model, messages, system, tools, tool choice, and the four generation options as separate rows, plus markers). Table length asserted against the documented count, matching `TestNewRequest_RoleVersusContentKind...`'s `if len(cases) != 12` idiom.
- [x] 1.7 Wrote the **markers** row per task 0.3's resolved branch A: `derive` is `WithMessages` with a marked message, `changed` reads `IsCacheBoundary()` off the derived message. No option added, no production line beyond what 1.1–1.5 already landed.
- [x] 1.8 **Item 3** *(pin)*: `TestRequest_DeriveWithUnrelatedChange_PreservesOpaquePayloadsByteIdentically`, two sub-cases: reasoning round-trip token with non-UTF-8 bytes (`S-REX-012`), tool-call argument bytes with non-canonical whitespace/key order (`S-REX-013`). Extension-value sibling (`S-REX-014`) deferred to AI-12.3 task 3.10, noted there.
- [x] 1.9 RED → GREEN **item 5** (`S-REX-003`, `S-REX-004`, `TestRequest_DeriveEdgeCases_...`) and **item 6** (`S-REX-005`, `S-REX-006`, `TestRequest_DeriveOptionInputMutatedAfterDeriving_...`).
- [x] 1.10 `make lint && make test` — both green, recorded below. Commit `feat(ai): derive a request by copy-on-write rebuild (AI-12.1)`.

### AI-12.1 evidence

**Note on 1.1/1.1a/1.2 sequencing.** Go's static typing makes `func (d requestDraft) rules() []Rule` (zero args, per `design.md` § 2.2) impossible to compile until the draft already carries `model`/`messages` (task 1.2's widening) — so 1.1, 1.1a and 1.2 were implemented as **one** atomic, behaviour-preserving edit and verified together, rather than as three independently-compiling increments. This is a sequencing note, not a design deviation: the end state matches `design.md` § 2.2/§ 2.2.1 exactly. The pre-existing test suite is the approval-test safety net (strict-tdd.md's "Approval Testing" pattern): zero test files were edited for this step.

```
$ go build ./... 
(no output — clean)
$ make test 2>&1 | grep -E "^(ok|FAIL)"
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.552s
ok  	github.com/cachicamas/backend/agent/src/ai	2.451s
$ git diff --stat -- backend/agent/src/ai/
 backend/agent/src/ai/request.go | 155 ++++++++++++++++++++++++++++++++++------
 1 file changed, 133 insertions(+), 22 deletions(-)
```
Only `request.go` changed; no `_test.go` file touched; full suite green.

**RED transcripts** (`With` landed as a compiling stub returning `Request{}, nil` first, per `design.md` § 11's "narrowest thing that compiles and fails"; `WithModel`/`WithMessages` landed as compiling no-ops first):
```
$ go test -race -run 'TestRequest_DeriveWithChangedOption_LeavesTheOriginalUnmodified' -v ./src/ai/...
    request_test.go:1423: derived.Temperature() = (0, false), want (0.99, true)
--- FAIL: TestRequest_DeriveWithChangedOption_LeavesTheOriginalUnmodified (0.00s)

$ go test -race -run 'TestNewRequest_WithModelBesideTheParameter_LastApplicationWins|TestRequest_DeriveReplacingModelOrMessages...' -v ./src/ai/...
    request_test.go:1478: request.Model() = "a", want "b" — ai.WithModel must win over the constructor's own "a" parameter
    request_test.go:1497: derived.Model() = "m-source", want "m-derived"
    request_test.go:1518: derived.Messages() carries 1 messages in the wrong order, want [first, second] as supplied
--- FAIL (both)
```

**GREEN, after implementing `With`/`WithModel`/`WithMessages` for real:** all of the above pass; full package `go test -race -count=1 ./src/ai/...` green.

**Bite proof 1 — item 2's totality table, § 2.2.1's exact trap.** Scratch-reverted `freeze()` to omit `system`/`hasSystem`:
```
--- FAIL: TestRequest_TotalityOfTheRebuildPath_EveryRegionIsReachable (0.00s)
    request_test.go:1737: region "system_instruction": the rebuild path did not reach it — the derived request does not observe the supplied change
    --- PASS: .../tool_choice, .../model, .../messages, .../tools, .../temperature, .../top_p, .../max_output_tokens, .../stop_sequences, .../cache_boundary_markers  (all nine other rows unaffected)
```
Reverted; re-ran `-count=1`: full table green again.

**Bite proof 2 — item 3's pin.** Scratch-inserted a lossy reasoning-token strip inside `With`:
```
--- FAIL: TestRequest_DeriveWithUnrelatedChange_PreservesOpaquePayloadsByteIdentically (0.00s)
    request_test.go:1783: derived reasoning token = ("", false), want ("\xff\xfe\x00\x80ok", true) — byte-identical to the source's
    --- PASS: .../tool_call_argument_bytes (unaffected — proves the scratch bug is surgical to reasoning content)
```
Reverted; re-ran: both sub-tests green again.

**Full gate:**
```
$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.

$ make test 2>&1 | grep -E "^(ok|FAIL)"
ok  	github.com/cachicamas/backend/agent/src/agenttest
ok  	github.com/cachicamas/backend/agent/src/ai
```

---

## Phase AI-12.2 — per-request options `[leaf]`

**Deliverable:** `backend/agent/src/ai/request.go`, `backend/agent/src/ai/request_test.go`.
**Spec:** `R-REX-004`, `R-REX-005`. **Design:** §§ 1, 3.
**Depends on:** AI-12.1.

- [x] **Item 1** — WHEN a per-request option overrides a construction-time option THEN the effective value is the override, observable via readback; absent overrides fall through to the constructed value. `TestRequest_OverrideGenerationOption_ReplacesTheConstructionTimeValueOrFallsThrough`.
- [x] **Item 2** — Option validation runs at derive time with the same AI-04 sentinels as construction — there is no second, weaker validation path. `TestRequest_DeriveTimeValidation_MatchesConstructionsClassAndPosition`.
- [x] **Item 3** *(appended)* — A **cross-region** rule that the source satisfied and the derivation breaks fails at derive time. `TestRequestWith_ToolSetReplacedSoTheChoiceNoLongerResolves_FailsAtDeriveTime` + `TestRequestWith_MessagesReplacedWithInvalidContent_ReportsAI06sRuleAtTheComposedPosition` (internal test).
- [x] **Item 4** *(appended)* — Derive-time first-failure is deterministic, same class/position as construction, every run. `TestRequestWith_TwoRulesViolatedAtOnce_ReportsTheDocumentedOrderFirstAcrossManyRuns`.
- [x] **Item 5** *(appended by Phase 0, § 13 row 8 = branch A, task 0.3)* — the cap re-runs on the derive path for free. `TestRequestWith_DerivationPastTheCacheBoundaryCap_FailsAtDeriveTime`. No AI-12 production code; no rule added for `V-REQ-24`.

### Ordered work

- [x] 2.1 RED → GREEN **item 1**: override wins; absent override falls through with the source's value **and** its presence flag; a newly supplied option flips absent → present on the derived request only (`S-REX-015` … `S-REX-019`).
- [x] 2.2 RED → GREEN **item 2**: for each bounded option, assert the derive-path failure is indistinguishable from the construction failure by `errors.Is` class **and** rendered position (`S-REX-020` … `S-REX-022`). Table-driven, one row per rule, six cases (model, messages, and the four `boundsRule` sub-rules).
- [x] 2.3 RED → GREEN **item 3**: replace the tool set so a landed tool choice no longer resolves; assert `ErrUnresolvedReference` at the choice's position (`S-REX-023`). Replace the messages with content that violates an AI-06 rule (assembled internally, matching `TestNewRequest_InvalidContentPart_...`'s precedent — content cannot be smuggled from `ai_test`); assert the composed position (`S-REX-024`).
- [x] 2.4 RED → GREEN **item 4**: a derivation violating two rules at once, run 100 times, reports the identical class and position, and it is the one construction reports first (`S-REX-025`). Used duplicate tool-call identities (rule 7) and orphan tool results (rule 8).
- [x] 2.5 **Item 5**, branch A (task 0.3): asserted the cap on the derive path — a construction anchor plus a `With` call, both reporting `ErrOutOfRange` at `cacheBoundaries`.
- [x] 2.6 `make lint && make test` — both green, recorded below. Commit `feat(ai): override generation options per request (AI-12.2)`.

### AI-12.2 evidence

**All items green from birth**, structurally: `With` and `NewRequest` already share `draft.rules()` (AI-12.1), so "derive-time validation is construction's validation" (R-REX-005) held the moment the tests were written, with no new production code. Every test was still run and its real output recorded, per strict TDD's evidence requirement.

```
$ go test -race -count=1 -run 'TestRequest_OverrideGenerationOption...|TestRequest_DeriveTimeValidation...|TestRequestWith_ToolSetReplaced...|TestRequestWith_TwoRulesViolated...|TestRequestWith_DerivationPast...|TestRequestWith_MessagesReplacedWithInvalidContent...' -v ./src/ai/...
--- PASS: TestRequestWith_MessagesReplacedWithInvalidContent_ReportsAI06sRuleAtTheComposedPosition (0.00s)
--- PASS: TestRequestWith_ToolSetReplacedSoTheChoiceNoLongerResolves_FailsAtDeriveTime (0.00s)
--- PASS: TestRequestWith_DerivationPastTheCacheBoundaryCap_FailsAtDeriveTime (0.00s)
--- PASS: TestRequestWith_TwoRulesViolatedAtOnce_ReportsTheDocumentedOrderFirstAcrossManyRuns (0.00s)
--- PASS: TestRequest_OverrideGenerationOption_ReplacesTheConstructionTimeValueOrFallsThrough (0.00s)  (5 sub-cases)
--- PASS: TestRequest_DeriveTimeValidation_MatchesConstructionsClassAndPosition (0.00s)  (6 sub-cases)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	1.577s
```

**Bite proof — item 2's central claim, "there is no second, weaker validation path".** Scratch-changed `With` to call `FirstFailure(scratchRules[:len(scratchRules)-1]...)` — a hand-duplicated rule list missing `boundsRule`, simulating exactly the defect R-REX-005 exists to make structurally impossible:
```
--- FAIL: TestRequest_DeriveTimeValidation_MatchesConstructionsClassAndPosition (0.00s)
    request_test.go:2073: got no failure, want required value is empty at "stopSequences[1]"
    request_test.go:2073: got no failure, want value is outside a documented bound at "maxOutputTokens"
    request_test.go:2073: got no failure, want value is outside a documented bound at "topP"
    request_test.go:2073: got no failure, want value is outside a documented bound at "temperature"
    --- PASS: .../an_empty_model_identity, .../an_empty_message_sequence  (unaffected — proves the scratch is surgical to boundsRule)
```
Reverted `With` to `FirstFailure(draft.rules()...)`; re-ran `-count=1`: all 6 sub-cases green again.

**Full gate:**
```
$ make lint
0 issues.
$ make test 2>&1 | grep -E "^(ok|FAIL)"
ok  	github.com/cachicamas/backend/agent/src/agenttest
ok  	github.com/cachicamas/backend/agent/src/ai
```

---

## Phase AI-12.3 — typed-but-opaque pass-through `[leaf]`

**Deliverable:** `backend/agent/src/ai/request_extension.go`, `backend/agent/src/ai/request_extension_test.go`, `backend/agent/src/ai/request.go`, **`backend/agent/src/agenttest/request_test.go`** *(added by Phase 0)*.
**Spec:** `R-REX-006`, `R-REX-007`, `R-REX-008`, `R-REX-010`. **Design:** §§ 2.3, 3, 5, 5.1, 6, 7, 7.1, 7.2.
**Depends on:** AI-12.1. **Parallel with:** AI-12.2.

**Hard constraint from AI-10.4, verified in Phase 0** (`design.md` § 5.1): the `ai` package's production dependency closure contains no `os`, `net`, `net/http` or `io/fs`, enforced by `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage`. `encoding/json` is banned from non-test files as a consequence — it imports `fmt`, which imports `os`. **`request_extension.go` must use `strings.Builder` and `strconv` and never `fmt.Sprintf`; a single `fmt` call in it turns a landed guard red.** Everything the region needs (`bytes`, `slices`, `strings`, `strconv`) is already in the closure, so this leaf adds no import and `go.mod` stays at zero requires.

- [ ] **Item 1** — WHEN a caller attaches a provider-namespaced opaque value THEN it survives to the adapter that claims the namespace, byte-exact.
- [ ] **Item 2** — WHEN an adapter for a *different* provider reads the same request THEN the foreign value is invisible to it and its translation is unaffected.
- [ ] **Item 3** — The pass-through is inert in validation and equality: two requests differing only in a third provider's namespace validate identically.
- [ ] **Item 4** *(appended)* — The region's own rules: an empty or whitespace-only namespace and an empty value each fail with `ErrEmpty` at `extensions[i].namespace` / `extensions[i].value`; a whitespace-only **value** is legal, because bytes are opaque (`design.md` § 6).
- [ ] **Item 5** *(appended)* — Re-applying a namespace is last-wins and keeps its **first** read-back ordinal; the region is a slice and never a map.
- [ ] **Item 6** *(appended)* — Neither a namespace nor a value renders through any of the four fmt verbs, and the rendering still names the region and its count (`R-REX-010`).
- [ ] **Item 7** *(appended by Phase 0)* — The region survives the **readback-rebuild** walk that `R-AMR-015`'s round-trip pin uses, which does **not** route through `Request.Equal` (`S-REX-054`). Extending the documented equality is necessary and not sufficient: `agenttest`'s `rebuildFromReadback` and `requireRequestsEqual` each walk a fixed nine-region list, so the pin would go green on a request that lost every extension. Both walks gain the region.

### Ordered work

- [ ] 3.1 Create `request_extension.go` with the banner `// AI-12.3 — the provider escape hatch: typed, namespaced, opaque.` and a blank line before `package ai` (revive `package-comments`).
- [ ] 3.2 RED → GREEN `ProviderExtension` (sealed, no exported constructor), `WithProviderExtension`, `Request.ProviderExtension(ns)`, `Request.ProviderExtensions()`; `requestDraft` gains `extensions []ProviderExtension`; rule row 12 appended **last** in `draft.rules()`.
- [ ] 3.3 RED → GREEN **item 1**: a fake translator claiming namespace `alpha` reproduces the value byte-exactly, including bytes that are not valid UTF-8 (`S-REX-026`, `S-REX-027`, `S-REX-034`).
- [ ] 3.4 RED → GREEN **item 2**: a second fake translator claiming `beta` finds no `alpha` extension, and its output for the request **with** the `alpha` extension is byte-identical to its output for the request **without** it (`S-REX-032`, `S-REX-033`). This pair is the milestone's acceptance clause.
- [ ] 3.5 RED → GREEN **item 3**: two requests differing only in a third namespace construct identically; and when both violate the same rule elsewhere, both fail with the same class at the same position (`S-REX-035`, `S-REX-036`).
- [ ] 3.6 RED → GREEN the equality half of item 3, per `design.md` § 7: two requests differing only in an extension **value** are **not** equal; a rebuild preserves the region (`S-REX-037`, `S-REX-038`). **Phase 0 resolved § 13 row 4: `Request.Equal` *is* exported (`request.go:426`), so extend it** — the alternative branch (compare region-by-region in `ai_test`, export no second equality) is **closed, not taken**, and is kept in this line so the choice stays visible. The exact block is `design.md` § 7.1: one `slices.EqualFunc(r.ProviderExtensions(), other.ProviderExtensions(), providerExtensionsEqual)` before `return true`, with a **free** comparator on `toolsEqual`'s precedent, no presence flag, read through the exported accessor.
- [ ] 3.6a RED → GREEN **item 7** *(added by Phase 0 — `design.md` § 7.2)*: in `backend/agent/src/agenttest/request_test.go`, add the extension region to `rebuildFromReadback` (re-apply through `WithProviderExtension` in read-back order, as the tool set and each option are re-applied) and to `requireRequestsEqual` (namespace by string equality, value by `bytes.Equal`, as tool schemas are compared). Red first: a request carrying an extension must fail `TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest` **before** the helpers are extended — that red is the proof the blind spot was real (`S-REX-054`).
- [ ] 3.7 RED → GREEN **item 4** (`S-REX-039` … `S-REX-045`), including the positional ordinal check on the second extension and the no-format-rule case.
- [ ] 3.8 RED → GREEN **item 5** (`S-REX-028` … `S-REX-031`), including copy-out on `Value()`.
- [ ] 3.9 RED → GREEN **item 6**: four verbs × two secret-bearing fields, one table (`S-REX-050` … `S-REX-052`); extend `Request.String`'s `appliedNames` with the extension region, count only. Build every rendering with `strings.Builder`/`strconv`, matching `Request.String` (`request.go:584`) — **no `fmt`**, per the leaf's hard constraint above.
- [ ] 3.10 Close AI-12.1 item 3's deferred extension row (task 1.8) now that the region exists.
- [ ] 3.11 `make lint && make test`, record both, commit `feat(ai): carry a namespaced opaque provider value (AI-12.3)`.

---

## Phase AI-12.4 — read-back determinism `[leaf]`

**Deliverable:** `backend/agent/src/ai/request_extension_test.go`, `backend/agent/src/ai/request_test.go`.
**Spec:** `R-REX-009`. **Design:** §§ 2.3, 6.
**Depends on:** AI-12.2, AI-12.3.
**Out of scope:** wire-byte determinism — owned by **AI-26.1** and **AI-26.4**, where wire bytes first exist. This node only guarantees the neutral surface cannot be the source of the nondeterminism.

- [ ] **Item 1** — Reading or iterating the option set and the pass-through values of one request twice yields identical order and content — the extension surfaces expose no map-iteration nondeterminism to a future serializer or wire body.
- [ ] **Item 2** *(appended)* — Two requests built from identical inputs in the same order read back identical sequences, and the string rendering of one request is byte-identical across repeated calls.

### Ordered work

- [ ] 4.1 RED → GREEN **item 1**: read the extension set of a five-extension request 100 times and the applied-option set of a four-option request 100 times; assert identical order and byte-equal values every time (`S-REX-046`, `S-REX-048`).
- [ ] 4.2 RED → GREEN **item 2**: same-inputs-two-requests comparison, and repeated `String()` rendering (`S-REX-047`, `S-REX-049`).
- [ ] 4.3 Record in this file the one-line statement that wire-byte determinism was **not** attempted and belongs to AI-26.1/AI-26.4, so a later reader does not read the omission as an oversight.
- [ ] 4.4 `make lint && make test`, record both, commit `test(ai): pin read-back determinism of options and extensions (AI-12.4)`.

---

## Closing checklist for the milestone

- [ ] All four leaves' items taken red → green → refactored, in order, both outputs recorded above.
- [ ] `make test` green with `-race`; `make lint` clean; both AI-00 import guards passing; `go.mod` at zero requires.
- [ ] **AI-10.4's guard still green**: `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` passes, and `request_extension.go` imports no `fmt` and no `encoding/json`.
- [ ] **AI-10.5's round-trip pin still bites**: `agenttest`'s `rebuildFromReadback` and `requireRequestsEqual` both carry the extension region (task 3.6a), and the pre-extension red was recorded.
- [ ] `openspec/specs/ai-contract-vocabulary/spec.md` **untouched**; `openspec/changes/cachicamas-ai-model-request/` and `openspec/changes/cachicamas-ai-cache-breakpoints/` **untouched**; `validation.go` carries **no** new rule class.
- [ ] Phase 0's eight-row register complete: rows 1–6 already resolved (§ 0.2 table); rows 7–8's branches recorded once AI-11 lands.
- [ ] The `S-AMR-046` finding (§ 0.5) carried to the **Wave 1 verify phase**, and **not** fixed from this change.
- [ ] If row 8 took branch B, the "breakpoint cap is not re-run on the derive path" gap is recorded here for Wave 1 verify.
- [ ] Any discovered prerequisite appended to doc 0002 under the revert-and-record clause, **in the same PR**; any discovered *test case* appended to its owning leaf's list here.
- [ ] Actuals filled into the per-slice forecast table.
- [ ] **Region-level exhaustiveness guard — considered, deferred to Wave 2.** Three milestones in a row (AI-10.5's `agenttest` round trip, AI-11's `Message.Equal`/`toolsEqual`, now AI-12.3's `Request.Equal` and the same two `agenttest` helpers) each had a hand-rolled region walk that did not see a new region "for free". A `go/parser` AST guard over `Request`/`requestDraft`'s own field declarations, in AI-06.4's idiom, was considered and judged **out of scope for AI-12**: `design.md` § 8 already provides the enumeration-based totality proof for today's regions and explicitly calls itself "the cheap half of AI-06.4's guard idea without adding a `[guard]` node the charter does not carry"; a struct's fields are not a closed constant-space vocabulary the way `PartKind` is, so a faithful guard needs a field-pairing heuristic (value field vs. `hasX` presence flag) that is a real design decision in its own right, not a mechanical extension of the landed pattern — and this milestone is already `High` risk with `size-exception` delivery. Recorded here as a **Wave 2 proposal**: a `[guard]` node, charter-authorised, that AST-scans `requestDraft`'s field declarations against the totality-table regions and against `agenttest`'s two walks and `Request.Equal`'s region list, so a future field added without a matching entry in all three fails a test rather than a re-verification gate. See `design.md` § 14 for the same note.
