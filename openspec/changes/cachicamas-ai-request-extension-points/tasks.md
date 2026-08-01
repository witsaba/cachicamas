# Tasks — request extension points

> **Change**: `cachicamas-ai-request-extension-points`
> **Milestone**: AI-12 · **Nodes**: AI-12.1 … AI-12.4, all `[leaf]`
> **Phase**: tasks · **Project**: cachicamas (witsaba) · **Date**: 2026-08-01
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-request-extension-points/spec.md`, `design.md`
> **Worktree**: `cachicamas-worktrees/ai-12` · **Branch**: `feat/ai-12-request-extension-points` (based on `07d2027`)
> **Depends on**: AI-10, AI-11 · **Blocks**: AI-24, AI-26.7, Layer 2's pre-request hook
> **Evidence gate**: recorded green `make test` **and** clean `make lint` in `backend/agent/` (`make test` is `go test -race -v ./...`)

---

## Entry point for the resuming agent

**Nothing is implemented.** This file is the plan only; every box below is unchecked.

Start at **§ Phase 0**, which is not a leaf — it is the re-verification gate. AI-10's second half and AI-11 were being written **concurrently with this plan**, in `../ai-wave-1` and `../ai-11`, and neither had landed when it was written. `design.md` § 13 is an eight-row checklist naming the exact file and symbol behind every assumption. **Run it before writing the first test.** A row that differs does not stop the milestone; each row carries its branch.

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

| Field | Value |
| --- | --- |
| Estimated changed lines | **~1090 Go** (~310 production, ~780 test) + ~1250 markdown planning artifacts |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR — four commits, one per leaf |
| Delivery strategy | `exception-ok` |
| Chain strategy | `size-exception` |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Per-slice forecast

| Slice | Files | Forecast prod | Forecast test | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `spec.md`, `design.md`, `tasks.md` | — | ~1250 prose | Low | 40 min |
| AI-12.1 rebuild | `request.go`, `request_test.go` | ~130 | ~330 | **High** — every later request milestone and Layer 2's hook inherit the derive shape | 45 min |
| AI-12.2 per-request options | `request.go`, `request_test.go` | ~15 | ~130 | Low — the mechanism is AI-10's last-wins, reached through AI-12.1 | 15 min |
| AI-12.3 pass-through | `request_extension.go`, `request_extension_test.go`, `request.go` | ~165 | ~250 | Medium-High — the shape AI-24's first adapter reads | 40 min |
| AI-12.4 determinism | `request_extension_test.go`, `request_test.go` | 0 | ~70 | Low | 10 min |

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

- [ ] 0.1 Fetch and rebase onto the current `../ai-wave-1` head; record the SHA in this file.
- [ ] 0.2 Walk all eight rows of `design.md` § 13 against the rebased tree. Record each row's outcome — **matches** or **differs, branch taken** — in a short table appended here.
- [ ] 0.3 Check `../ai-11`'s `design.md` for the marker surface. Record whether markers ride on `Segment`/`Tool`/`Message` (transitive reachability, no new option) or are a request-level region (one option appended to AI-12.1).
- [ ] 0.4 Record the baseline: `make lint` clean and `make test` green on the rebased tree **before** any AI-12 edit, so the first red is unambiguous.

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

- [ ] 1.1 **Extraction first.** Move `NewRequest`'s `FirstFailure` argument list to `func (d requestDraft) rules() []Rule`, with the order preserved exactly; `NewRequest` calls `FirstFailure(d.rules()...)`. Behaviour-preserving — the existing suite must stay green with no test edited. Record the green run.
- [ ] 1.2 **Widen the draft.** Add `model string` and `messages []Message` to `requestDraft`; `NewRequest` seeds them from its parameters before applying options; every rule that read a parameter now reads draft state (`design.md` § 2.2).
- [ ] 1.3 RED → GREEN **item 1**: `TestRequest_DeriveWithChangedOption_LeavesTheOriginalUnmodified` — capture every region of the source before, derive, compare after (`S-REX-001`, `S-REX-002`).
- [ ] 1.4 RED → GREEN `Request.With(opts ...RequestOption) (Request, error)` — copy the draft, apply, `FirstFailure(d.rules()...)`, freeze.
- [ ] 1.5 RED → GREEN **item 4**: `WithModel`, `WithMessages`, and the last-wins pin `TestNewRequest_WithModelBesideTheParameter_LastApplicationWins` (`S-REX-007`).
- [ ] 1.6 RED → GREEN **item 2**: the region-enumeration table of `design.md` § 8 — one row per region, each with a `derive` and a `changed` observer; the failure message names the missing region (`S-REX-007` … `S-REX-011`).
- [ ] 1.7 Write the **markers** row against whatever Phase 0.3 recorded. Transitive → assert reachability through the region-level option. Request-level → append one option and one row, and note the appended case here.
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
- [ ] **Item 4** *(appended)* — Derive-time first-failure is deterministic and follows the same documented order as construction — same class, same position, on every run.

### Ordered work

- [ ] 2.1 RED → GREEN **item 1**: override wins; absent override falls through with the source's value **and** its presence flag; a newly supplied option flips absent → present on the derived request only (`S-REX-015` … `S-REX-019`).
- [ ] 2.2 RED → GREEN **item 2**: for each bounded option, assert the derive-path failure is indistinguishable from the construction failure by `errors.Is` class **and** rendered position (`S-REX-020` … `S-REX-022`). Table-driven, one row per rule.
- [ ] 2.3 RED → GREEN **item 3**: replace the tool set so a landed tool choice no longer resolves; assert `ErrUnresolvedReference` at the choice's position (`S-REX-023`). Replace the messages with content that violates an AI-06 rule; assert the composed position (`S-REX-024`).
- [ ] 2.4 RED → GREEN **item 4**: a derivation violating two rules at once, run many times, reports the identical class and position, and it is the one construction reports first (`S-REX-025`).
- [ ] 2.5 `make lint && make test`, record both, commit `feat(ai): override generation options per request (AI-12.2)`.

---

## Phase AI-12.3 — typed-but-opaque pass-through `[leaf]`

**Deliverable:** `backend/agent/src/ai/request_extension.go`, `backend/agent/src/ai/request_extension_test.go`, `backend/agent/src/ai/request.go`.
**Spec:** `R-REX-006`, `R-REX-007`, `R-REX-008`, `R-REX-010`. **Design:** §§ 2.3, 3, 5, 6, 7.
**Depends on:** AI-12.1. **Parallel with:** AI-12.2.

- [ ] **Item 1** — WHEN a caller attaches a provider-namespaced opaque value THEN it survives to the adapter that claims the namespace, byte-exact.
- [ ] **Item 2** — WHEN an adapter for a *different* provider reads the same request THEN the foreign value is invisible to it and its translation is unaffected.
- [ ] **Item 3** — The pass-through is inert in validation and equality: two requests differing only in a third provider's namespace validate identically.
- [ ] **Item 4** *(appended)* — The region's own rules: an empty or whitespace-only namespace and an empty value each fail with `ErrEmpty` at `extensions[i].namespace` / `extensions[i].value`; a whitespace-only **value** is legal, because bytes are opaque (`design.md` § 6).
- [ ] **Item 5** *(appended)* — Re-applying a namespace is last-wins and keeps its **first** read-back ordinal; the region is a slice and never a map.
- [ ] **Item 6** *(appended)* — Neither a namespace nor a value renders through any of the four fmt verbs, and the rendering still names the region and its count (`R-REX-010`).

### Ordered work

- [ ] 3.1 Create `request_extension.go` with the banner `// AI-12.3 — the provider escape hatch: typed, namespaced, opaque.` and a blank line before `package ai` (revive `package-comments`).
- [ ] 3.2 RED → GREEN `ProviderExtension` (sealed, no exported constructor), `WithProviderExtension`, `Request.ProviderExtension(ns)`, `Request.ProviderExtensions()`; `requestDraft` gains `extensions []ProviderExtension`; rule row 12 appended **last** in `draft.rules()`.
- [ ] 3.3 RED → GREEN **item 1**: a fake translator claiming namespace `alpha` reproduces the value byte-exactly, including bytes that are not valid UTF-8 (`S-REX-026`, `S-REX-027`, `S-REX-034`).
- [ ] 3.4 RED → GREEN **item 2**: a second fake translator claiming `beta` finds no `alpha` extension, and its output for the request **with** the `alpha` extension is byte-identical to its output for the request **without** it (`S-REX-032`, `S-REX-033`). This pair is the milestone's acceptance clause.
- [ ] 3.5 RED → GREEN **item 3**: two requests differing only in a third namespace construct identically; and when both violate the same rule elsewhere, both fail with the same class at the same position (`S-REX-035`, `S-REX-036`).
- [ ] 3.6 RED → GREEN the equality half of item 3, per `design.md` § 7: two requests differing only in an extension **value** are **not** equal; a rebuild preserves the region (`S-REX-037`, `S-REX-038`). Per `design.md` § 13 row 4 — extend `Request.Equal` if AI-10.6 exported it, otherwise compare region-by-region in `ai_test` and **export no second equality**.
- [ ] 3.7 RED → GREEN **item 4** (`S-REX-039` … `S-REX-045`), including the positional ordinal check on the second extension and the no-format-rule case.
- [ ] 3.8 RED → GREEN **item 5** (`S-REX-028` … `S-REX-031`), including copy-out on `Value()`.
- [ ] 3.9 RED → GREEN **item 6**: four verbs × two secret-bearing fields, one table (`S-REX-050` … `S-REX-052`); extend `Request.String`'s `appliedNames` with the extension region, count only.
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
- [ ] `openspec/specs/ai-contract-vocabulary/spec.md` **untouched**; `validation.go` carries **no** new rule class.
- [ ] Phase 0's eight-row re-verification table filled in, with every branch taken recorded.
- [ ] Any discovered prerequisite appended to doc 0002 under the revert-and-record clause, **in the same PR**; any discovered *test case* appended to its owning leaf's list here.
- [ ] Actuals filled into the per-slice forecast table.
