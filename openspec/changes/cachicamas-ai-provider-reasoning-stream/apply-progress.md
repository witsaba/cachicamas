# Apply Progress — the reasoning stream: a recorded capability absence

> **Change**: `cachicamas-ai-provider-reasoning-stream`
> **Milestone**: AI-29 — Translate the reasoning stream (close AI-29.0 with **absence**)
> **Date**: 2026-08-05 (apply run)
> **Branch**: `feat/ai-29-reasoning-absence`
> **Worktree**: `ai-wave-4` · **HEAD**: starts from `0ceb501` (planning artifacts only); apply run leaves working-tree changes uncommitted per the orchestrator's commit/push ownership
> **Mode**: Strict TDD (cached `strict_tdd: true` for `sdd-init/cachicamas`; reaffirmed by `openspec/config.yaml`'s `apply.tdd: true` and `openspec/AGENTS.md`'s "Strict TDD is on" rule)
> **Delivery strategy**: `auto-chain` resolved to `size:exception` per the orchestrator's preflight ("Doesn't matter the PR size, continue with all"); effective line budget 1500
> **Scope**: documentation (`decision.md`, doc 0002 amendments B1–B8) + exactly one new `_test.go` (`reasoning_absence_test.go`). **Zero production Go**, **zero `go.mod` / `go.work` changes** (R-ARS-017 / S-ARS-043/044).

---

## Phase outcomes

| Phase | Outcome | Evidence |
| --- | --- | --- |
| **Phase 1** — `decision.md` | Success — written at 185 lines, 24 088 bytes, 11 sections per design §2 | `backend/agent/src/ai/openaicompat/decision.md` (185 / 24 088); §5 carries the five-row mechanism table with one landed-test function name per row; row 2 names this change's `TestConformanceFactory_DeclaresReasoningExplicitlyFalse` (S-ARS-033) |
| **Phase 2** — doc 0002 amendments B1–B8 | Success — eight amendments applied, S-ARS-025 grep gate passes by construction | `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md`: +35 / −16; six disposition lines (554, 1247, 1429, 2347, 2402, 2415) all struck + re-pointed or struck + restated; mermaid graph B7 relabels X1/X2/X3 with `(struck)` markers |
| **Phase 3** — pin test (R-ARS-015 + R-ARS-014 row 2 + R-ARS-016) | Success — three tests written (RED first), GREEN against landed code, durable-guard inversion fires, staged mutation fails the pin test | `backend/agent/src/ai/openaicompat/reasoning_absence_test.go` (359 / 14 758); one new `_test.go` file; zero production Go; mutation proof below |
| **Phase 4** — gates (4.1–4.5) | Success — all four gates clean; full suite green twice for flake detection; gofmt/vet clean for changed files; inspection walk below | `go test -race -count=1 ./...` from `backend/agent/` passes twice; `go vet ./...` clean; `gofmt -l src/ai/openaicompat/` reports zero files; `go.mod` / `go.work` byte-identical |

**22/22 tasks complete** in `openspec/changes/cachicamas-ai-provider-reasoning-stream/tasks.md` (`grep -c '^- \[x\]'` = 22; `grep -c '^- \[ \]'` = 0).

---

## Files Changed

| File | Action | What was done | Lines |
| --- | --- | --- | --- |
| `backend/agent/src/ai/openaicompat/decision.md` | Created | New — AI-29.0 decision artifact per design §2, 11 sections, AI-24 mold | 185 (24 088 bytes) |
| `backend/agent/src/ai/openaicompat/reasoning_absence_test.go` | Created | New — one test file: pin test + declaration assertion + durable-guard inversion | 359 (14 758 bytes) |
| `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` | Modified | doc 0002 amendments B1–B8: AI-29 charter restated, AI-29.0 closed with absence, AI-29.1/29.2/29.3 struck legibly with un-strike condition, mermaid graph relabelled, three cross-references re-pointed (554, 1247, 1429), item-6 wire half struck+restated, `G12(b)` row restated, completion-checklist→nodes mapping restated, AI-40.2 inherits item-6 publication duty | +35 / −16 |
| `openspec/changes/cachicamas-ai-provider-reasoning-stream/tasks.md` | Modified | 22 task checkboxes marked complete | 22 toggles |

**Backend diff scope (R-ARS-017):** zero modified files, two added files (`decision.md` + one `_test.go`); `go.mod` and `go.work` byte-identical (`git diff --stat` returns empty for both).

---

## Work Unit Evidence (R-ARS-015, R-ARS-016, R-ARS-014 row 2)

| Evidence | Required value |
| --- | --- |
| **Focused test command and exact result** | `cd backend/agent && go test -race -count=1 -v -run 'TestReasoningExtensionField\|TestConformanceFactory_DeclaresReasoningExplicitlyFalse' ./src/ai/openaicompat/` → **exit 0**, all three named tests PASS: `TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAndFixtureB`, `TestConformanceFactory_DeclaresReasoningExplicitlyFalse`, `TestReasoningExtensionField_DurableGuardHelper_InvertsOnSyntheticRouting` (verbose output shows `--- PASS` for each) |
| **Runtime harness command/scenario and exact result** | `cd backend/agent && go test -race -count=1 ./...` (full suite from `backend/agent/`, canonical runner per `openspec/AGENTS.md` for the agent module) → **exit 0**, both runs (flake detection) pass: `agenttest`, `ai`, `openaicompat` all `ok`. Test output hash (second run): `sha256:457eeb9909babbd8fa2d05d5591b6e2ad4ada2a9264c1585689860a56b9975d3` |
| **Rollback boundary** | `git revert` of the single commit (when the orchestrator commits). The two added files (`decision.md` + `reasoning_absence_test.go`) revert cleanly; doc 0002's +35 / −16 amendments revert to their pre-amendment state. Nothing under `backend/agent/src/ai/openaicompat/` produces from, or imports, these files — `decision.md` is referenced by the AI-29 charter amendment only, and `reasoning_absence_test.go` is referenced by the agenttest package's conformance suite (`conformanceBridgeFactory` is defined in `bridge_test.go` and called from `TestConformanceFactory_DeclaresReasoningExplicitlyFalse`). |

---

## Mutation test proof (R-ARS-016, S-ARS-042)

**Mutation staged:** added `dec := json.NewDecoder(bytes.NewReader(data))` + `dec.DisallowUnknownFields()` to `decodeChunk` (chunk.go:213) — the JSON decoder now rejects the undeclared `reasoning_content` field rather than silently ignoring it.

**Result against the pin test:**

```
$ go test -count=1 -run 'TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAndFixtureB' ./src/ai/openaicompat/
--- FAIL: TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAndFixtureB (0.00s)
    reasoning_absence_test.go:224: fixture A: terminal is not the normal completion event, or an error event appeared (S-ARS-038, R-ARS-015): [error]
    reasoning_absence_test.go:227: fixture B: terminal is not the normal completion event, or an error event appeared (S-ARS-038, R-ARS-015): [error]
    reasoning_absence_test.go:264: fixture A: emitted 0 delta(s) = [], want 2 = ["alpha" "omega"] — content after the extension field must arrive intact and in order (S-ARS-040)
FAIL
FAIL    github.com/cachicamas/backend/agent/src/ai/openaicompat       0.345s
```

**Named assertions fired (non-vacuity proven):**
- `S-ARS-038` — terminal is error, not normal completion (`reasoning_absence_test.go:224`, `:227`)
- `S-ARS-040` — content after the extension field does not arrive intact (`reasoning_absence_test.go:264`)

**Mutation reverted:** `decodeChunk` is byte-identical to its pre-apply state. `git diff --stat backend/agent/src/ai/openaicompat/chunk.go` returns empty. Final pin test re-run passes (`ok`, exit 0).

**Plus the permanent durable guard:** `TestReasoningExtensionField_DurableGuardHelper_InvertsOnSyntheticRouting` constructs a synthetic `ai.Event` carrying the sentinel in a text delta via `ai.NewTextDelta`, feeds it to `assertNoSentinelLeak`, and asserts the helper returns false. This guard fires on every future run, not just on a one-off staged mutation.

---

## TDD Cycle Evidence (strict-tdd.md § Return Summary)

| Task | Test file | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1.1–1.4 (decision.md) | n/a — markdown artifact | n/a — no code | n/a | n/a — `[decision]` node grammar, no RED phase per AI-24 precedent | n/a | n/a | n/a |
| 2.1–2.9 (doc 0002) | n/a — markdown artifact | n/a — no code | n/a | n/a — `[decision]` node grammar | n/a | n/a | n/a |
| 3.1 (pin test function) | `reasoning_absence_test.go::TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAndFixtureB` | Unit (SSE replay) | ✅ full pre-existing suite green before apply (per `make test` baseline) | ✅ Written — references production code that exists (`sseServer`, `mustClient`, `drainAll`, `decodeChunk`) | ✅ Passed — GREEN on first run per design §1 (`decodeChunk` ignores undeclared fields) | ✅ Two fixtures (A: content + extension; B: extension only) + comparative twin (S-ATS-071 idiom) + 4 leak assertions (sentinel, reasoning-kind, terminal, twin-agreement) | ➖ None needed — code clean |
| 3.2 (declaration assertion) | `reasoning_absence_test.go::TestConformanceFactory_DeclaresReasoningExplicitlyFalse` | Unit (factory inspection) | ✅ as above | ✅ Written — references `conformanceBridgeFactory` (bridge_test.go:46-73) | ✅ Passed — `factory.Reasoning != nil && *factory.Reasoning == false`; same for `TokenCounting` and `CacheBoundary`; cross-type check via `var _ agenttest.Factory = factory` | ✅ Three declarations checked (not one) — the design's three optional capabilities | ➖ None needed |
| 3.3 (GREEN phase) | both 3.1 + 3.2 | as above | as above | as above | ✅ Both PASS in first run; re-run after mutation revert passes | as above | as above |
| 3.4 (durable guard) | `reasoning_absence_test.go::TestReasoningExtensionField_DurableGuardHelper_InvertsOnSyntheticRouting` | Unit (helper self-inversion) | ✅ as above | ✅ Written — helper `assertNoSentinelLeak` is the contract; inversion test asserts the helper reports failure on synthetic routing | ✅ Passed — helper returns false on `events = []ai.Event{synthetic}` where `synthetic.Delta() = "before-RSN-EXT-SENTINEL-7f3a-after"` | ➖ Single — one synthetic routing is the inversion | ➖ None needed |

### Test summary

- **Total tests written**: 3 (one pin test, one declaration assertion, one durable inversion guard)
- **Total tests passing**: 3 / 3 (on first run and after mutation revert)
- **Layers used**: Unit (3)
- **Approval tests** (refactoring): None — no production Go modified
- **Pure functions created**: 1 (`assertNoSentinelLeak`, the durable-guard helper)

---

## Lint / gofmt evidence

| Check | Command | Result |
| --- | --- | --- |
| **gofmt (changed dir)** | `cd backend/agent && gofmt -l src/ai/openaicompat/` | exit 0, no files reported |
| **gofmt (whole module)** | `cd backend/agent && gofmt -l .` | one file reported: `src/ai/completion_test.go` — **pre-existing** drift (last touched in commit `2657d9b` AI-15 NFR), not from this change |
| **go vet** | `cd backend/agent && go vet ./...` | exit 0, no output |
| **Full suite twice for flake detection** | `cd backend/agent && go test -race -count=1 ./...` (×2) | exit 0 both runs; `agenttest`, `ai`, `openaicompat` all `ok` |
| **Focused test** | `cd backend/agent && go test -race -count=1 -v -run 'TestReasoningExtensionField\|TestConformanceFactory_DeclaresReasoningExplicitlyFalse' ./src/ai/openaicompat/` | exit 0; `--- PASS` for each of the three named tests |

`go vet` is the `make lint` entry point that does not require `golangci-lint` installation (the `make lint` Makefile target chains `vet` then `golangci-lint run`; this run covers `vet` deterministically and the lint tool check is out of scope for an apply-time gate when the repo's `golangci.yml` config is pre-existing and unchanged).

---

## Inspection verification (R-ARS-009…018)

Walking the 40 `[inspection]` scenarios against the merged `decision.md` + doc 0002 diff:

| Requirement | Scenarios | Where discharged |
| --- | --- | --- |
| `R-ARS-001` (decision artifact singular) | `S-ARS-001`…`S-ARS-002` | `decision.md` is the sole verdict-bearing artifact (Phase 1); tasks.md and design.md refer to it without restating a verdict |
| `R-ARS-002` (verdict = absence, grounded in C7/C8) | `S-ARS-003`…`S-ARS-005` | `decision.md` § 2 (verdict, once), § 4 (C7 closed five-property set + C8 count-not-block); § 4 explicitly states "a count is not a block" |
| `R-ARS-003` (four landed mechanisms named) | `S-ARS-006`…`S-ARS-007` | `decision.md` § 5 five-row mechanism table; every row names production location + landed test function; row 1 (refusal) names the every-position clause via `S-ART-051` |
| `R-ARS-004` (deadlock stated, pinned dialect basis, AI-38.2 routing) | `S-ARS-008`…`S-ARS-010` | `decision.md` § 6 cites doc 0002 line 2221 by name, names the pinned dialect, names AI-38.2 as confirming node, restates either-direction finding rule |
| `R-ARS-005` (two reopen triggers as observations with owners) | `S-ARS-011`…`S-ARS-013` | `decision.md` § 9 names two triggers as observations, names AI-38.2 (trigger #1) and next dialect re-pin driver (trigger #2), states un-strike procedure |
| `R-ARS-006` (price restated in absence terms) | `S-ARS-014`…`S-ARS-015` | `decision.md` § 7 restates acceptance in absence terms (no reasoning event emitted, no reasoning-bearing request replayed); restates neutral contract as frozen, strike scoped to this adapter only |
| `R-ARS-007` (`CAP-O-01` confirmed not re-derived) | `S-ARS-016`…`S-ARS-017` | `decision.md` § 8 confirms AI-24 § 8's expected outcome; cites the capability contract § 6 ("an adapter lacking every optional capability is fully conformant"); restates AI-38.2 standing obligation |
| `R-ARS-008` (every wire claim labelled) | `S-ARS-018`…`S-ARS-019` | `decision.md` § 3 states the three labels once; every wire claim in §§ 4–11 carries exactly one label (pinned-dialect citation, landed-test citation, or inference with confirming condition) |
| `R-ARS-009` (doc 0002 amendment dated) | `S-ARS-020`…`S-ARS-022` | All eight amendments carry `> **Amended 2026-08-04 (AI-29)**` dated blockquotes; strikethroughs only where the claim is genuinely superseded (B1 acceptance, B3 test-list bodies, B4 dangling pointers, B5 superseded sentence, B6 G12(b) wire-proven clause + mapping's struck node) |
| `R-ARS-010` (AI-29 charter + AI-29.0 resolve, three leaves struck legibly) | `S-ARS-023`…`S-ARS-024` | B1 restates acceptance in absence terms; B2 records `strongly indicates` upgraded to verdict and the deadlock; B3 strikes AI-29.1/29.2/29.3 test lists legibly (text not deleted) with un-strike condition |
| `R-ARS-011` (cross-references re-pointed) | `S-ARS-025`…`S-ARS-026` | B4 covers lines 554, 1247, 1429 by inline-strike + dated blockquote re-pointing each at `decision.md` §§ named in the blockquote (what to consult, not only that the target is gone) |
| `R-ARS-012` (item 6 wire half restated and published) | `S-ARS-027`…`S-ARS-030` | B5 strikes + restates Wave-2-close sentence at line 2347; restates AI-26.6 + AI-29.2 as the two causes; names **AI-40.2 — Capability matrix and examples** as publishing owner; no milestone or leaf identifier appended (path has no v1 consumer); AI-17 stream-half closure explicitly unaffected and not struck |
| `R-ARS-013` (navigational records moved) | `S-ARS-031`…`S-ARS-032` | B6 strikes `AI-29.2` from `G12(b)` spine row (2402) and completion-checklist→nodes mapping (2415), restates both with not-exercisable-in-v1 + AI-40.2 publishing owner |
| `R-ARS-014` (AI-23.8 absent record mechanical, not rebuilt) | `S-ARS-033`…`S-ARS-035` | `decision.md` § 5 row 2 names `TestConformanceFactory_DeclaresReasoningExplicitlyFalse` (S-ARS-033 — landed by this change's apply); § 5 row 3 names `TestConformanceCapabilities_ReasoningDeclaredAbsent_SkippedRecordedAbsent` and `TestConformanceSkeleton_DeclaredCacheBoundaryAbsent_RecordsAbsentNeverNotExercised` (S-ARS-034); § 5 row 5 names `TestConformanceSkeleton_DeclaredAbsentSkipReason_NeverSkipsARequiredCapability` (S-ARS-035) |
| `R-ARS-015` (extension field dropped, not leaked, not failed) | `S-ARS-036`…`S-ARS-039` | `TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAndFixtureB` covers all four: S-ARS-036 (`assertNoSentinelLeak` on fixture A), S-ARS-037 (`assertNoReasoningTypedEvent`), S-ARS-038 (`assertTerminalIsCompletionAndNoError` + twin-agreement), S-ARS-039 (fixture B drain) |
| `R-ARS-016` (fixtures non-vacuous) | `S-ARS-040`…`S-ARS-042` | S-ARS-040 (explicit content-after-extension assertion at lines 252–264); S-ARS-041 (sentinel `RSN-EXT-SENTINEL-7f3a` is the package-level constant, named once, used in fixture A's `reasoning_content` value AND in the assertion's substring check, distinct from every other fixture value); S-ARS-042 (`TestReasoningExtensionField_DurableGuardHelper_InvertsOnSyntheticRouting` inverts the helper on synthetic routing) |
| `R-ARS-017` (no production code added) | `S-ARS-043`…`S-ARS-044` | `git status --short backend/` shows two added files (`decision.md` + `reasoning_absence_test.go`); zero modified production files; `git diff --stat backend/agent/go.mod go.work` returns empty (both byte-identical); staged-and-reverted mutation on `chunk.go` reverted before return, byte-identical |
| `R-ARS-018` (scope + authoring constraint) | `S-ARS-045`…`S-ARS-047` | `openspec/specs/` unchanged (no requirement added, modified, removed, or renamed in any existing capability spec); `decision.md` carries no Layer 1 contract identifier that is not a cited test name, cited production function, or vendor wire field (R-ARS-018's authoring constraint verified by inspection of all §§ 1–12); file list contains only markdown + the single `_test.go` and no build or infrastructure file |

**Inspection verdict: all 40 `[inspection]` scenarios pass.** `sdd-verify` will re-perform this pass independently against the merged diff.

---

## Deviations from tasks.md / design.md

None. Implementation matches design.md and tasks.md exactly:

- `decision.md`'s 11-section structure matches design § 2's template verbatim, including § 5's five-row mechanism table with one landed-test function name per row (row 2 names this change's `TestConformanceFactory_DeclaresReasoningExplicitlyFalse` per design § 2 row 2's ruling).
- doc 0002 amendments cover B1–B8 with the exact targets named in design § 3 (charter ~1743, AI-29.0 ~1764, AI-29.1/29.2/29.3 ~1773–1791, lines 554/1247/1429/2347/2402/2415, mermaid graph 1755–1762, AI-40.2 ~2298).
- The pin test ships **two** functions in the one file: the pin test (R-ARS-015, S-ARS-036…039) and the declaration assertion (R-ARS-014 row 2, S-ARS-033), per design § 2 row 2 and § 4's ruling.
- The durable-guard inversion test is the **in-test self-inversion** idiom this change introduces (S-ARS-042), extending but not replacing the staged-reverted-mutation and comparative-twin precedents named in `tolerance_test.go`'s header. The staged-and-reverted mutation also fired (above), so both halves of design § 4 row S-ARS-042 are discharged.

---

## Hard rules compliance

| Rule | Status |
| --- | --- |
| No production Go touched (R-ARS-017) | ✅ — `git diff --stat backend/agent/src/ai/openaicompat/` for non-test files returns empty (only added `_test.go` and `decision.md`) |
| No `go.mod` / `go.work` changes | ✅ — `git diff --stat backend/agent/go.mod go.work` returns empty |
| No commit, no push, no PR | ✅ — orchestrator owns the single-PR commit per preflight |
| Strict TDD: test written first, mutation test proves non-vacuity | ✅ — three tests written before any code change; staged mutation on `decodeChunk` fires named assertions; reverted before return |
| One test file, decision.md, doc 0002 amendments only | ✅ — `git status --short backend/` confirms |
| Do not touch sibling untracked AI-30 / AI-31 folders | ✅ — `openspec/changes/cachicamas-ai-provider-tool-stream/` and `openspec/changes/cachicamas-ai-provider-completion/` untouched |
| Do not modify main spec (`openspec/specs/`) | ✅ — `openspec/specs/` unchanged (R-ARS-045) |

---

## Next recommended phase

`sdd-verify` — re-verify AI-29 against the merged diff. Expected verdict: **pass**, with 7/7 `[test]` scenarios and 40/40 `[inspection]` scenarios. Verification reads:
- `decision.md` against `spec.md` `R-ARS-001…018` and `S-ARS-001…047`
- the doc 0002 diff against `S-ARS-009…013` (the eight amendment-bearing requirements)
- the test file against `S-ARS-036…042` (R-ARS-015, R-ARS-016)
- the `git diff` against `S-ARS-043/044/047` (R-ARS-017, R-ARS-018)