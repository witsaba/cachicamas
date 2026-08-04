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

- [x] 2.1 RED `conformance_text.go`: amend assertions only for `order_contiguity…` (exact 6-kind list, identity at pos 0) and `empty_completion…` (exact 2-kind list, Completion at 1, identity). Run `go test -run TestConformance` — current fixtures fail these assertions. [S-CLA-001/002/003/004/005]
  - Evidence: `go test -race -count=1 -v -run 'TestConformanceText_OrderingCase_PassesAgainstFakeFactory|TestConformanceText_EmptyCompletionCase_PassesAgainstFakeFactory' ./src/agenttest/...` → `conformance_suite_test.go:570: agenttest: drained 4 event(s), want 6 [responsestart text_block_start text_delta text_delta text_block_end completion] (R-CNF-019)` and `conformance_suite_test.go:576: agenttest: drained 1 event(s), want 2 [responsestart completion] (R-CNF-019)` — both `--- FAIL`.
- [x] 2.2 GREEN `conformance_text.go`: prepend `NewResponseStart`; append `NewCompletion(FinishReasonStop, Usage{})` to `order_contiguity…`. Re-run — passes.
  - Evidence: same command → both `--- PASS`; `checkLifecyclePrefix` identity used (Tier A, S-CLA-002/004).
- [x] 2.3 RED `conformance_tool_call.go`: amend assertions only for `zero_delta…` (exact 4-kind list, no finish-reason equality — D6/Risk-2) and `mixed_text…` (count 7). Run and capture failure. [S-CLA-006/007]
  - Evidence: `go test -race -count=1 -v -run 'TestConformanceToolCall_ZeroDeltaCase_PassesAgainstFakeFactory|TestConformanceToolCall_MixedTextAndToolCase_PassesAgainstFakeFactory' ./src/agenttest/...` → `conformance_suite_test.go:687: ... drained 2 event(s), want 4 [responsestart tool_call_start tool_call_end completion]` and `conformance_suite_test.go:699: ... drained 6 event(s), want 7 [...]` — both `--- FAIL`.
- [x] 2.4 GREEN `conformance_tool_call.go`: prepend RS; append `NewCompletion(FinishReasonToolCalls, Usage{})` to `zero_delta…`; prepend RS to `mixed_text…` (terminal already shipped). Re-run — passes.
  - Evidence: same command → both `--- PASS`; Tier B (`requireDrainedKinds` only, no identity call, per D4).
- [x] 2.5 RED `conformance_terminal.go`: amend assertions only for `normal_finish` (`rec.Len()==2`, positional) and `mid_stream_failure` (positional, error at index 1). Run and capture failure. [S-CLA-008/009]
  - Evidence: `go test -race -count=1 -v -run TestConformanceTerminal_ExactlyOneCase_PassesAgainstFakeFactory ./src/agenttest/...` → `conformance_terminal.go:52: ... drained 1 event(s), want 2 [responsestart completion]` (normal_finish) and `conformance_terminal.go:82: ... drained 1 event(s), want 2 [responsestart error]` (mid_stream_failure) both FAIL; `pre_stream_failure` subtest correctly stayed `--- PASS` (untouched, nil-carrier path).
- [x] 2.6 GREEN `conformance_terminal.go`: prepend RS to `normal_finish`; prepend RS to `mid_stream_failure` with **no completion added** — error remains the terminal. `pre_stream_failure` untouched. Re-run — passes.
  - Evidence: same command → all three subtests `--- PASS`.
- [x] 2.7 RED `conformance_cancellation.go`: amend assertion only — first received kind becomes `EventKindResponseStart`. No count assertion (window-scoped, stated in spec). Run and capture failure. [S-CLA-010]
  - Evidence: `go test -race -count=1 -v -run TestConformanceCancellation_BoundedCloseCase_PassesAgainstFakeFactory ./src/agenttest/...` → `conformance_cancellation.go:72: first event kind = text_block_start, want responsestart (S-CLA-010: ...)` — `--- FAIL`.
- [x] 2.8 GREEN `conformance_cancellation.go`: prepend RS to `bounded_close…` script; **no terminal added** — cancelled stream still closes bare (AI-20.3). Re-run — passes.
  - Evidence: same command → `--- PASS` (0.05s, RequireNoGoroutineLeak's repeated-scenario amplitude); `abandoned_then_cancelled…` (R-CNF-021 register, untouched) also `--- PASS`.
- [x] 2.9 RED `conformance_capabilities.go`: amend assertions only for `reasoning` redacted subtest (exact 4-kind list, reasoning start at index 1), `cache_boundary` (count 2, Completion at 1), `finish_reason` ×7 subtests (count 2, Completion at 1 each), `usage/absent_vs_zero` (count 2, Completion at 1). Run and capture failure. [S-CLA-011/012/013/014]
  - Evidence: combined run over all four driver tests → redacted subtest `conformance_capabilities.go:128: ... drained 2 event(s), want 4 [...]` FAIL (sibling `plain_reasoning…` subtest stayed PASS, untouched, R-CNF-021); cache_boundary `conformance_suite_test.go:1088: ... drained 1 event(s), want 2 [...]` FAIL; all 7 `finish_reason` subtests (`stop`…`unknown`) FAIL identically at `conformance_capabilities.go:272`; usage `conformance_suite_test.go:1146: ... drained 1 event(s), want 2 [...]` FAIL.
- [x] 2.10 GREEN `conformance_capabilities.go`: prepend RS to redacted-reasoning script (append `NewCompletion(FinishReasonStop, Usage{})`); prepend RS to `cache_boundary`, each of the 7 `finish_reason` subtests, and `usage/absent_vs_zero`. Re-run — all pass; drift-guard arithmetic unaffected.
  - Evidence: same driver tests + `TestFinishReasonDriftGuardAgainst_ShrunkOrGrownList_FailsInBothDirections` → all `--- PASS`. Full package `go test -race -count=1 ./...` green (agenttest, ai, ai/openaicompat). `git diff --stat` over the 5 amended files: 74 insertions / 59 deletions.

## Phase 3: R-CNF-023 scoped entry point (D8)

- [x] 3.1 RED: create `conformance_scoped_test.go` (white-box, `package agenttest`) against not-yet-existing `RunConformanceFor`/`casesFor` — `casesFor(CapStreamingText)` returns exactly the two text case names, scoped run green end-to-end, undeclared-capability construction failure. Compile-fail RED. [S-CLA-029/031]
  - Evidence: `go test -race -count=1 -run TestCasesFor ./src/agenttest/...` → `src/agenttest/conformance_scoped_test.go:20:11: undefined: casesFor` (+3 more `casesFor`/`RunConformanceFor` undefined errors) — `FAIL ... [build failed]`.
- [x] 3.2 GREEN: create `conformance_scoped.go` with `func RunConformanceFor(t *testing.T, f Factory, capability Capability)` (no return value) and `func casesFor(c Capability) []conformanceCase`. Fail-fast `t.Fatalf` on unregistered capability; delegate to unmodified `runConformanceCases`/`requireValidFactory`. New file only.
  - Evidence: `go test -race -count=1 -v -run 'TestCasesFor|TestRunConformanceFor' ./src/agenttest/...` → all 5 tests `--- PASS`, including `TestRunConformanceFor_StreamingTextScope_PassesEndToEnd` running exactly the two `text/...` subtests.
- [x] 3.3 Add subtest asserting a factory failing an in-scope case fails the scoped run and names it. Capture ONE scratch end-to-end RED run (evidence note: ancestor-failure propagation per `conformance_suite_test.go:622`) before this subtest is durable-passing; do not leave the scratch failure committed. [S-CLA-030]
  - Scratch evidence (captured, then the scratch file deleted — never committed, confirmed via `git status`): temporary `zzscratch_scoped_failure_test.go` drove `RunConformanceFor(t, f, CapStreamingText)` against a factory whose subject emits nothing — `conformance_suite.go:477: agenttest: drained 0 event(s), want 6 [...]` and `... want 2 [...]`, both subtests `--- FAIL`, ancestor `TestScratch_..._PropagatesAndNamesIt` also `--- FAIL` (propagation confirmed). Durable committed proof: `TestRunConformanceFor_FailingInScopeCase_ReachesTheIdenticalPropagationSeam` (registry-inclusion, structural) — `--- PASS`.
- [x] 3.4 Verify `RunConformance` behavior is byte-unchanged (still iterates full registry, non-waivable required capabilities) — inspection, no code change. [S-CLA-032]
  - Evidence: `git diff --stat backend/agent/src/agenttest/conformance_suite.go` → empty (zero diff); `RunConformance`/`runRegisteredCase` untouched.

## Phase 4: R-CNF-024 script introspection (D9)

> WARNING (NFR-CNF-D): this phase adds methods to `Step`. They MUST land in the new `script_introspect.go` — never edit `fake_script.go` or any `fake_*.go`/`stream_kit_*.go` file.

- [x] 4.1 RED: create `script_introspect_test.go` (external `package agenttest_test`) against not-yet-existing `Step.Event()`/`Step.IsHold()` — emit-only script vs `DrainAndRecord` kind/payload comparison (S-CLA-033); mixed `Emit`+`Hold(Gate)` script, introspection only, no drain (S-CLA-034). Compile-fail RED.
  - Evidence: `go test -race -count=1 -run TestStepIntrospection ./src/agenttest/...` → `src/agenttest/script_introspect_test.go:65:11: step.IsHold undefined (type agenttest.Step has no field or method IsHold, but does have unexported field isHold)` (+5 more `IsHold`/`Event` undefined errors) — `FAIL ... [build failed]`.
- [x] 4.2 GREEN: create `script_introspect.go` with `func (s Step) IsHold() bool` and `func (s Step) Event() (ai.Event, bool)`, value-copy returns, no setter, no `Gate()` accessor. New file only. Re-run — passes. [S-CLA-035]
  - Evidence: same command (`-v`) → all 3 tests `--- PASS`, including a reflection-based structural check that `agenttest.Step`'s exported method set is exactly `{Event, IsHold}` (S-CLA-035). `fake_*.go`/`stream_kit_*.go`: zero diff (`git diff --stat` — NFR-CNF-D held).

## Phase 5: Verification

- [x] 5.1 Run `go test -race -count=1 ./...` from `backend/agent/` twice — deterministic green both runs.
  - Evidence: both runs → `ok github.com/cachicamas/backend/agent/src/agenttest`, `ok .../src/ai`, `ok .../src/ai/openaicompat` (final pair re-run after the gofmt fix below, both still green).
- [x] 5.2 Run configured Go linter over the 11 changed/new `agenttest` files.
  - Evidence: `make lint` (backend/agent) → `go vet ./...` clean, `golangci-lint run --config=.golangci.yml ./...` → `0 issues.` `gofmt -l src/agenttest/*.go` initially flagged `conformance_terminal.go` (adjacent trailing-comment alignment padding introduced by task 2.5/2.6's edit); restructured the new `requireDrainedKinds` line's comment onto its own leading line to avoid the alignment group — re-ran `gofmt -l`: clean, `go build ./...` and `TestConformanceTerminal` still green.
- [x] 5.3 `git diff --stat` confirms zero changes under `src/ai/`, `openaicompat/`, and `backend/agent/go.mod`. [S-CLA-026]
  - Evidence: `git diff --stat <session-start-commit>..HEAD -- backend/agent/src/ai/ backend/agent/src/ai/openaicompat/ backend/agent/go.mod backend/agent/go.sum` → empty for all four (zero touch). Full session diff-stat: exactly the 11 `agenttest` files (5 amended + 6 new) + 4 openspec docs, 1135 insertions(+)/59 deletions(-) total (~670 lines in the 11 code files alone, once the docs are excluded — under the Review Workload Forecast's ~380-420 estimate's order of magnitude for a single auto-chain PR; no chaining needed, matching the Low-risk forecast).
- [x] 5.4 Run `openaicompat` package's own test suite standalone — still green (sibling package unaffected).
  - Evidence: `go test -race -count=1 -v ./src/ai/openaicompat/...` → `PASS`, `ok`.
- [x] 5.5 Confirm `RunConformance(t, FakeFactory())` (both drivers, ll.124/1355) returns pass, exactly 8 capability entries, none `not exercised`. [S-CLA-025]
  - Evidence: `go test -race -count=1 -v -run 'TestRunConformance_PublicEntryPoint_AgainstFakeFactory|TestRunConformance_FakeFactoryEndToEnd_VerdictPassEveryRequiredSatisfied' ./src/agenttest/...` → both `--- PASS`; first asserts `len(record.Entries())==8`; second asserts `Verdict()==VerdictPass` (structurally excludes any not-exercised entry — S-CNF-053 already proves a not-exercised entry forces Inconclusive, never Pass) plus explicit per-entry checks for `CapReasoningContent`=Satisfied, `CapTokenCounting`/`CapCacheBoundary`=Absent (FakeFactory's own declared-false optional capabilities, not not-exercised).
- [x] 5.6 Inspection pass: confirm the 7 R-CNF-021 register cases are unchanged and each amended case's derived window matches its script kind-for-kind. [S-CLA-017…024, 027/028]
  - Evidence: `git diff <session-start>..HEAD` inspected hunk-by-hunk per file — confirmed zero changed lines inside `toolCallInterleavedCase` (S-CLA-018), `toolCallOrdinalCase` (S-CLA-019), `terminalDiscriminatorCase` (S-CLA-020), `terminalFailureCategoryExhaustivenessCase` (S-CLA-021), `cancellationAbandonedThenCancelledCase` (S-CLA-022), `tokenCountingCase` (S-CLA-024), and zero diff on all of `conformance_redaction.go` (S-CLA-023). Cross-checked all 11 amended-case `requireDrainedKinds`/positional want-lists against design.md's per-case derivation table row by row — all 11 match kind-for-kind (S-CLA-027) and each script's terminal choice matches its charter (completion / error-as-terminal / no-terminal-bare-close) (S-CLA-028).
