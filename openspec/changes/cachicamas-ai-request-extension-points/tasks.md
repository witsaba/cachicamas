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
- [ ] 0.3 **Open — yours.** Rows 7 and 8 depend on AI-11, still being implemented in `../ai-11` and unreadable from here. Run `design.md` § 13.2's two-step decision procedure against AI-11's landed surface and record the branch taken for each row. The branches are already decided; you execute, you do not re-analyse.
- [ ] 0.4 **Open — yours.** Record the baseline: `make lint` clean and `make test` green on the rebased tree **before** any AI-12 edit, so the first red is unambiguous. This gate did not run tests.

### 0.2 outcome table

| Row | Check | Outcome |
| --- | --- | --- |
| 1 | `tools` / `toolChoice` are `RequestOption`s | **Matches** — `WithTools` (`request.go:334`), `WithToolChoice` (`:344`). No AI-12 work |
| 2 | cross-region rules read draft state | **Differs — branch taken.** Seven of ten rules read a `NewRequest` parameter. The three cross-region free functions need no edit; only two call sites move. `design.md` § 2.2 revised |
| 3 | rule order is a literal slice | **Differs in form, holds in substance.** A variadic argument list. **Rows 7/8 of the order were recorded backwards** and **there is no row 11**. `design.md` § 3 corrected twice |
| 4 | `Request.Equal` is exported | **Matches** (`request.go:426`). The "compare in `ai_test`, export nothing" branch is **closed, not taken** — task 3.6 keeps its wording |
| 5 | extend `Equal` to the extension region | **Confirmed, and insufficient.** The exact block is `design.md` § 7.1. The recorded rationale was wrong: AI-10.5's round-trip pin never calls `Equal`. `design.md` § 7.2 adds the `agenttest` obligation → new task 3.6a |
| 6 | `Reasoning.Token()` is `([]byte, bool)`, byte-exact | **Matches** (`reasoning_content.go:306`). No AI-12 work |
| 7 | where markers live | **Unresolved** — AI-11 landing concurrently. Both branches in `design.md` § 13.2 |
| 8 | breakpoint cap is a `Rule` in the list | **Unresolved** — no cap rule exists at `1c4171e`. Both branches in `design.md` § 13.2 |

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

- [ ] **Item 1** — WHEN a caller derives a modified request from an existing one THEN the original is observably unmodified (deep comparison before and after) and the derived request validates independently.
- [ ] **Item 2** — Deriving is **total**: every region — system segments, messages, tools, tool choice, options, markers — is reachable by the rebuild path. A region the hook cannot reach is a region a cache breakpoint or injected context can never be applied to.
- [ ] **Item 3** *(pin)* — A request carrying reasoning round-trip tokens rebuilds with every token byte-identical (AI-07.3's property, extended across the path session persistence will later travel).
- [ ] **Item 4** *(appended)* — The **model identity** is reachable by the rebuild path too, and `WithModel` inside `NewRequest` is last-wins over the parameter — the disposition pinned rather than left to inference (`design.md` § 4).
- [ ] **Item 5** *(appended)* — A failed derivation returns the zero request and leaves the source observably unmodified; a derivation with no options succeeds and equals the source.
- [ ] **Item 6** *(appended)* — The rebuild copies on the way **in**: mutating the slice passed to `WithMessages` or `WithStopSequences` after the derivation leaves the derived request unchanged, matching `NewRequest`'s landed discipline.

### Ordered work

- [ ] 1.1 **Extraction first.** Move `NewRequest`'s `FirstFailure` argument list to `func (d requestDraft) rules() []Rule`, with the order preserved exactly; `NewRequest` calls `FirstFailure(d.rules()...)`. Behaviour-preserving — the existing suite must stay green with no test edited. Record the green run. *(Verified: the landed list is a **variadic argument list**, nine inline closures plus `draft.boundsRule()`; `FirstFailure(rules ...Rule)` makes `d.rules()...` a drop-in. `design.md` § 3 carries the real order — note that **duplicate identities precede orphan results**, the reverse of what this plan first recorded.)*
- [ ] 1.1a **Extract the freeze too** *(added by Phase 0)*. Add `func (d requestDraft) freeze() Request`, and make `NewRequest` and `With` both end `return d.freeze(), nil`. **Not cosmetic**: `Request` stores the system region twice — `Request.system`/`hasSystem` *and* `requestDraft.system`/`hasSystem` — and `SystemInstruction()` reads the top-level pair, so a `With` that freezes without re-deriving it silently reverts the region (`design.md` § 2.2.1). One draft, one rule slice, one freeze, two seeds. Behaviour-preserving on its own; record the green run.
- [ ] 1.2 **Widen the draft.** Add `model string` and `messages []Message` to `requestDraft`; `NewRequest` seeds them from its parameters before applying options; every rule that read a parameter now reads draft state (`design.md` § 2.2). *(Verified: **seven of ten** rules read a parameter — rule 1 reads `model`, rules 2, 3, 4, 6 iterate `messages`, rules 7 and 8 pass `messages` to a free function. The three free functions `duplicateToolCallRule` / `unresolvedToolResultRule` / `anyToolCallHasID` take `[]Message` as a parameter and need **no edit** — only their two call sites move to `d.messages`. The whole diff stays inside `NewRequest`'s body.)*
- [ ] 1.3 RED → GREEN **item 1**: `TestRequest_DeriveWithChangedOption_LeavesTheOriginalUnmodified` — capture every region of the source before, derive, compare after (`S-REX-001`, `S-REX-002`).
- [ ] 1.4 RED → GREEN `Request.With(opts ...RequestOption) (Request, error)` — copy the draft, apply, `FirstFailure(d.rules()...)`, freeze.
- [ ] 1.5 RED → GREEN **item 4**: `WithModel`, `WithMessages`, and the last-wins pin `TestNewRequest_WithModelBesideTheParameter_LastApplicationWins` (`S-REX-007`).
- [ ] 1.6 RED → GREEN **item 2**: the region-enumeration table of `design.md` § 8 — one row per region, each with a `derive` and a `changed` observer; the failure message names the missing region (`S-REX-007` … `S-REX-011`). **The complete list is `design.md` § 8.1: nine landed regions plus markers plus extensions.** The four generation options get **four rows, not one** — `spec.md` `R-REX-002` now requires it, because a single "generation options" row is a row a fifth option can be added past. The system-instruction row must observe through `SystemInstruction()` on the derived request, which is what catches a broken freeze (`S-REX-053`).
- [ ] 1.7 Write the **markers** row against whatever Phase 0.3 recorded, using `design.md` § 13.2 row 7's decision procedure. Transitive → assert reachability through the region-level option, add nothing. Request-level → append one option and one row, and note the appended case here.
- [ ] 1.8 **Item 3** *(pin)*: `TestRequest_RebuildWithReasoningToken_PreservesEveryTokenByte`, plus the tool-call argument-bytes and extension-value siblings (`S-REX-012` … `S-REX-014`). Extension row lands after AI-12.3; note the dependency and revisit.
- [ ] 1.9 RED → GREEN **item 5** (`S-REX-003`, `S-REX-004`) and **item 6** (`S-REX-005`, `S-REX-006`).
- [ ] 1.10 `make lint && make test`, record both, commit `feat(ai): derive a request by copy-on-write rebuild (AI-12.1)`.

---

## Phase AI-12.2 — per-request options `[leaf]`

**Deliverable:** `backend/agent/src/ai/request.go`, `backend/agent/src/ai/request_test.go`.
**Spec:** `R-REX-004`, `R-REX-005`. **Design:** §§ 1, 3.
**Depends on:** AI-12.1.

- [ ] **Item 1** — WHEN a per-request option overrides a construction-time option THEN the effective value is the override, observable via readback; absent overrides fall through to the constructed value.
- [ ] **Item 2** — Option validation runs at derive time with the same AI-04 sentinels as construction — there is no second, weaker validation path.
- [ ] **Item 3** *(appended)* — A **cross-region** rule that the source satisfied and the derivation breaks fails at derive time, so no rule is silently construction-only.
- [ ] **Item 4** *(appended)* — Derive-time first-failure is deterministic and follows the same documented order as construction — same class, same position, on every run. Assert equivalence **between the two doors**, never against an absolute ordinal in the rule order: the order is shared with milestones that append to it, and an absolute ordinal would fail on an append that changed no behavior (`spec.md` `R-REX-005`).
- [ ] **Item 5** *(appended by Phase 0, conditional on § 13 row 8)* — If AI-11 landed the breakpoint cap `V-REQ-24` as a `Rule` in the request's rule list, then a derivation that pushes the request past the cap fails at derive time with `ErrOutOfRange` at the position construction reports — the rebuild re-runs it for free, with no AI-12 production code. If AI-11 enforced the cap elsewhere, **assert nothing**: record the gap in the closing checklist for the Wave 1 verify phase, and do **not** add the rule here — `V-REQ-24` is AI-11's and a second owner on one register term is worse than a missing assertion.

### Ordered work

- [ ] 2.1 RED → GREEN **item 1**: override wins; absent override falls through with the source's value **and** its presence flag; a newly supplied option flips absent → present on the derived request only (`S-REX-015` … `S-REX-019`).
- [ ] 2.2 RED → GREEN **item 2**: for each bounded option, assert the derive-path failure is indistinguishable from the construction failure by `errors.Is` class **and** rendered position (`S-REX-020` … `S-REX-022`). Table-driven, one row per rule.
- [ ] 2.3 RED → GREEN **item 3**: replace the tool set so a landed tool choice no longer resolves; assert `ErrUnresolvedReference` at the choice's position (`S-REX-023`). Replace the messages with content that violates an AI-06 rule; assert the composed position (`S-REX-024`).
- [ ] 2.4 RED → GREEN **item 4**: a derivation violating two rules at once, run many times, reports the identical class and position, and it is the one construction reports first (`S-REX-025`). Use two rules whose landed relative order is known — **duplicate tool-call identities (rule 7) precedes orphan tool results (rule 8)**, deliberately, because uniqueness is a precondition of resolution (`design.md` § 3).
- [ ] 2.5 **Item 5**, per Phase 0.3's row-8 branch: assert the cap on the derive path, or record the gap. One line either way.
- [ ] 2.6 `make lint && make test`, record both, commit `feat(ai): override generation options per request (AI-12.2)`.

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
