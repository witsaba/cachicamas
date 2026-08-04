# Tasks: Conformance suite accepts the mandated response lifecycle

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~380-420 (11 agenttest files: 5 amended fixtures + 6 new incl. tests) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain (cached; unused — single slice) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: feature-branch-chain
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Whole amendment (lifecycle prefix + scoped entry point + introspection) | PR 1 | `go test -race -count=1 ./...` (backend/agent) | `RunConformance(t, FakeFactory())` drivers, ll.124/1355 | Single-commit revert restores shipped counts (design) |

> **HARD CONSTRAINT (NFR-CNF-D)**: `fake_*.go` and `stream_kit_*.go` are NEVER edited. All new surface lands in new files only: `conformance_lifecycle.go`(+test), `conformance_scoped.go`(+test), `script_introspect.go`(+test). Any task that adds a `Script`/`Step` method MUST land it in `script_introspect.go`, not in `fake_script.go`.

## Phase 1: Lifecycle check foundation (D1/D2/D3)

- [x] 1.1 RED: create `conformance_lifecycle_test.go` with `TestLifecyclePrefixGuard_StartlessSlice_FailsNamingAbsentResponseStart` (input `[NewTextBlockStart(1)]`), `TestLifecyclePrefixGuard_MismatchedIdentity_FailsNamingField` (id + model subtests), and one `_Passes` positive — against not-yet-existing `checkLifecyclePrefix`. Compile-fail RED evidence. [S-CLA-015/016]
  - Evidence: `go test -race -count=1 -run TestLifecyclePrefixGuard ./src/agenttest/...` → `src/agenttest/conformance_lifecycle_test.go:26:8: undefined: checkLifecyclePrefix` (+3 more identical-shape errors) — `FAIL	github.com/cachicamas/backend/agent/src/agenttest [build failed]`.
- [x] 1.2 GREEN: create `conformance_lifecycle.go` with `checkLifecyclePrefix(events []ai.Event, wantResponseID, wantServedModel string) error` (D1) and `requireDrainedKinds(t, events, want []ai.EventKind)` (D3). New file only — do not touch `fake_*.go`. Guard tests from 1.1 pass.
  - Evidence: same command → `--- PASS: TestLifecyclePrefixGuard_StartlessSlice_FailsNamingAbsentResponseStart`, `--- PASS: TestLifecyclePrefixGuard_MismatchedIdentity_FailsNamingField` (both subtests), `--- PASS: TestLifecyclePrefixGuard_Passes`; full package `go test -race -count=1 ./...` still green (agenttest, ai, ai/openaicompat).

## Phase 2: Amend fixtures — inverted TDD, assertions before script, per file

- [ ] 2.1 RED `conformance_text.go`: amend assertions only for `order_contiguity…` (exact 6-kind list, identity at pos 0) and `empty_completion…` (exact 2-kind list, Completion at 1, identity). Run `go test -run TestConformance` — current fixtures fail these assertions. [S-CLA-001/002/003/004/005]
- [ ] 2.2 GREEN `conformance_text.go`: prepend `NewResponseStart`; append `NewCompletion(FinishReasonStop, Usage{})` to `order_contiguity…`. Re-run — passes.
- [ ] 2.3 RED `conformance_tool_call.go`: amend assertions only for `zero_delta…` (exact 4-kind list, no finish-reason equality — D6/Risk-2) and `mixed_text…` (count 7). Run and capture failure. [S-CLA-006/007]
- [ ] 2.4 GREEN `conformance_tool_call.go`: prepend RS; append `NewCompletion(FinishReasonToolCalls, Usage{})` to `zero_delta…`; prepend RS to `mixed_text…` (terminal already shipped). Re-run — passes.
- [ ] 2.5 RED `conformance_terminal.go`: amend assertions only for `normal_finish` (`rec.Len()==2`, positional) and `mid_stream_failure` (positional, error at index 1). Run and capture failure. [S-CLA-008/009]
- [ ] 2.6 GREEN `conformance_terminal.go`: prepend RS to `normal_finish`; prepend RS to `mid_stream_failure` with **no completion added** — error remains the terminal. `pre_stream_failure` untouched. Re-run — passes.
- [ ] 2.7 RED `conformance_cancellation.go`: amend assertion only — first received kind becomes `EventKindResponseStart`. No count assertion (window-scoped, stated in spec). Run and capture failure. [S-CLA-010]
- [ ] 2.8 GREEN `conformance_cancellation.go`: prepend RS to `bounded_close…` script; **no terminal added** — cancelled stream still closes bare (AI-20.3). Re-run — passes.
- [ ] 2.9 RED `conformance_capabilities.go`: amend assertions only for `reasoning` redacted subtest (exact 4-kind list, reasoning start at index 1), `cache_boundary` (count 2, Completion at 1), `finish_reason` ×7 subtests (count 2, Completion at 1 each), `usage/absent_vs_zero` (count 2, Completion at 1). Run and capture failure. [S-CLA-011/012/013/014]
- [ ] 2.10 GREEN `conformance_capabilities.go`: prepend RS to redacted-reasoning script (append `NewCompletion(FinishReasonStop, Usage{})`); prepend RS to `cache_boundary`, each of the 7 `finish_reason` subtests, and `usage/absent_vs_zero`. Re-run — all pass; drift-guard arithmetic unaffected.

## Phase 3: R-CNF-023 scoped entry point (D8)

- [ ] 3.1 RED: create `conformance_scoped_test.go` (white-box, `package agenttest`) against not-yet-existing `RunConformanceFor`/`casesFor` — `casesFor(CapStreamingText)` returns exactly the two text case names, scoped run green end-to-end, undeclared-capability construction failure. Compile-fail RED. [S-CLA-029/031]
- [ ] 3.2 GREEN: create `conformance_scoped.go` with `func RunConformanceFor(t *testing.T, f Factory, capability Capability)` (no return value) and `func casesFor(c Capability) []conformanceCase`. Fail-fast `t.Fatalf` on unregistered capability; delegate to unmodified `runConformanceCases`/`requireValidFactory`. New file only.
- [ ] 3.3 Add subtest asserting a factory failing an in-scope case fails the scoped run and names it. Capture ONE scratch end-to-end RED run (evidence note: ancestor-failure propagation per `conformance_suite_test.go:622`) before this subtest is durable-passing; do not leave the scratch failure committed. [S-CLA-030]
- [ ] 3.4 Verify `RunConformance` behavior is byte-unchanged (still iterates full registry, non-waivable required capabilities) — inspection, no code change. [S-CLA-032]

## Phase 4: R-CNF-024 script introspection (D9)

> WARNING (NFR-CNF-D): this phase adds methods to `Step`. They MUST land in the new `script_introspect.go` — never edit `fake_script.go` or any `fake_*.go`/`stream_kit_*.go` file.

- [ ] 4.1 RED: create `script_introspect_test.go` (external `package agenttest_test`) against not-yet-existing `Step.Event()`/`Step.IsHold()` — emit-only script vs `DrainAndRecord` kind/payload comparison (S-CLA-033); mixed `Emit`+`Hold(Gate)` script, introspection only, no drain (S-CLA-034). Compile-fail RED.
- [ ] 4.2 GREEN: create `script_introspect.go` with `func (s Step) IsHold() bool` and `func (s Step) Event() (ai.Event, bool)`, value-copy returns, no setter, no `Gate()` accessor. New file only. Re-run — passes. [S-CLA-035]

## Phase 5: Verification

- [ ] 5.1 Run `go test -race -count=1 ./...` from `backend/agent/` twice — deterministic green both runs.
- [ ] 5.2 Run configured Go linter over the 11 changed/new `agenttest` files.
- [ ] 5.3 `git diff --stat` confirms zero changes under `src/ai/`, `openaicompat/`, and `backend/agent/go.mod`. [S-CLA-026]
- [ ] 5.4 Run `openaicompat` package's own test suite standalone — still green (sibling package unaffected).
- [ ] 5.5 Confirm `RunConformance(t, FakeFactory())` (both drivers, ll.124/1355) returns pass, exactly 8 capability entries, none `not exercised`. [S-CLA-025]
- [ ] 5.6 Inspection pass: confirm the 7 R-CNF-021 register cases are unchanged and each amended case's derived window matches its script kind-for-kind. [S-CLA-017…024, 027/028]
