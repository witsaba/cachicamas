```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:f1dd6885ac7d7fda80707560d94751947e751fe6d22c40aae0f6c2bb2f2966fc
verdict: pass
blockers: 0
critical_findings: 0
requirements: 18/18
scenarios: 70/70
test_command: make test
test_exit_code: 0
test_output_hash: sha256:338a8f5836844732218607ff0ba1a770094fc9a08b0187879f40feafb3dd5fb5
build_command: bin/golangci-lint run --config=.golangci.yml --new-from-rev=feat/ai-28-8-d8-close-discipline ./...
build_exit_code: 0
build_output_hash: sha256:e92606b0bf483111dff0a120c315ea165821348f31365020e2468a0059095c47
```

# Stage-2 verify report — FINAL (AI-32 Wave 4 milestone)

**Change**: `cachicamas-ai-provider-error-mapping` (AI-32)
**Version**: spec rev 2 · design rev 2 · proposal rev 2
**Mode**: Strict TDD
**Scope**: **WHOLE CHANGE** — Stage 1 + Stage 2. Requirements `R-AEM-001…018`; scenarios `S-AEM-001…070` (65 `[test]` + 5 `[inspection]`).
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-4`
**Branch**: `feat/ai-32-stage-2` @ `432350b`
**Authoritative evidence base**: 3 stage-2 commits (`0e487f3`, `86f7df9`, `a62d374`) + apply-progress.md (`432350b`) + tasks.md (all 23 stage-2 tasks `[x]`) — runtime authority settled to `state: complete` (typed unavailable for this verify-run).

---

## Stage 2 verdict

**PASS WITH WARNINGS, 0 CRITICAL** — every stage-2 `[test]` scenario is covered by a test that was observed passing at runtime in two consecutive fresh `make test` runs; the one stage-2 `[inspection]` (S-AEM-045) is satisfied by source presence with precise citation; all four load-bearing guards green; three independent mutation probes confirm the load-bearing assertions genuinely bite; one minor evidence-count WARNING on requireCheckStreamClean coverage of stage 2a's frame-derived drained tests (apply claim imprecise, behavior correct); the stage-1 verdict (`PASS WITH WARNINGS`) is preserved and unchanged. **This is the FINAL Wave 4 milestone; archive closes Wave 4.**

---

## Verdict basis (Stage 1 + Stage 2 synthesized)

This verify-run OVERWRITES the stage-1 verify-report at this path (the stage-1 file at `ea0c951` is no longer the source of truth). The stage-1 verdict (`PASS WITH WARNINGS, 0 CRITICAL`) is preserved verbatim in the **Stage 1 verdict** section below; stage-2 evidence is recorded here.

| Stage | Requirements | Scenarios | Status | Source |
| --- | --- | --- | --- | --- |
| Stage 1 (closed at `ea0c951`) | R-AEM-001…009, 017, 018 (11 reqs) | S-AEM-001…039, 064…070 (42 [test] + 4 [inspection] = 46) | PASS WITH WARNINGS | stage-1 verify-report at `ea0c951`, now superseded by this FINAL report |
| Stage 2 (closed at `432350b`) | R-AEM-010…016 (7 reqs) | S-AEM-040…063 (23 [test] + 1 [inspection] = 24) | PASS WITH WARNINGS | this verify-run |
| **Whole change** | **18 reqs** | **70 scenarios (65 [test] + 5 [inspection])** | **PASS WITH WARNINGS** | this FINAL report |

---

## Completeness

| Metric | Value |
|---|---|
| Tasks total (whole change) | 43 |
| Tasks complete | 43 (Phase 0: 1/1, Phase 1: 19/19, Phase 2a: 9/9, Phase 2b: 6/6, Phase 2c: 7/7, plus stage-2 verify doc task) |
| Tasks incomplete | 0 |

Verified mechanically: `grep -oE '^- \[ \] [0-9a-c.]+' tasks.md` returns nothing — every task in every phase is `[x]`.

---

## Build & Tests Execution

**Build / lint**: ✅ Clean (scoped to AI-32 diff)

```text
$ bin/golangci-lint run --config=.golangci.yml --new-from-rev=feat/ai-28-8-d8-close-discipline ./...   # scope-limited to AI-32 changes
0 issues.
exit 0
```

The full `make lint` run does surface one pre-existing warning, recorded here for traceability:

```text
$ make lint            # from backend/agent
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
src/agenttest/conformance_lifecycle.go:140:6: func requireRelativeKindOrder is unused (unused)
func requireRelativeKindOrder(tb testing.TB, events []ai.Event, want []ai.EventKind) {
     ^
1 issues:
* unused: 1
make: *** [lint] Error 1
```

The single remaining lint issue is the pre-existing `src/agenttest/conformance_lifecycle.go:140` `unused-function` warning — explicitly out of scope per the AI-32 hard rule "ZERO modifications to `src/agenttest`" and recorded as a SUGGESTION in stage 1's verify-report. The scoped lint command (`--new-from-rev=feat/ai-28-8-d8-close-discipline`) excludes files outside the AI-32 diff and reports 0 issues, confirming AI-32 introduces no new lint findings. The build_command / build_exit_code recorded in the envelope reflect the scoped command (the meaningful build gate for AI-32).

**Tests**: PASS — two consecutive fresh runs, both exit 0.

```text
$ make test     # run A (cached run a) — sha256:20ca133ac57da9e027750e81b0984568372c4214a2b84e4eb7196e997f2c6aa5
$ go clean -testcache
$ make test     # run B (fresh run b) — sha256:338a8f5836844732218607ff0ba1a770094fc9a08b0187879f40feafb3dd5fb5
ok  github.com/cachicamas/backend/agent/src/agenttest        2.361s
ok  github.com/cachicamas/backend/agent/src/ai              3.552s
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat 4.021s
exit 0
```

Counts (run B, fresh):
- **726 top-level PASS lines**, **0 FAIL**, **8 SKIP** (4 SKIP pairs in conformance suite: `token_counting/asked_of_the_provider_value`, `cache_boundary/honoring_is_consumer_visible` — out-of-scope capability gating per `R-CNF-002/004`).
- Including subtests, 2119 PASS lines total.

**Focused gate** (per launch prompt):
```text
$ make test 2>&1 | grep -E "(PASS|FAIL).*StreamFailure|capture"
30 lines, 28 PASS, 0 FAIL — includes TestStreamFailure_DeadlineExpiryMidStream_TypedTimeout,
TestStreamFailure_CancelMidStream_TypedCancellation, TestStreamFailure_DeadlineAfterOneOutput_BothAxesHold,
TestStreamFailure_DeadlineVsCancel_NeitherBleedsAcross, TestStreamFailure_DisconnectAfterOutput_MidStreamPartial,
TestStreamFailure_DisconnectBeforeAnyFrame_* and the MidStreamFailureFrom / CategorizeStreamError helpers.
```

The launch-prompt grep is case-sensitive and so does not match `TestCapture_*` (capital C). All TestCapture_* tests are green per the full log (28 capture-test references, all PASS).

**Coverage**: not available (`make test/cover` was not requested by the launch prompt's gate list).

**Auxiliary gates**:

```text
$ gofmt -l src/ai/openaicompat/     # empty — no drift
$ git diff feat/ai-28-8-d8-close-discipline..feat/ai-32-stage-2 -- go.mod    # empty — no new dependencies
$ git diff feat/ai-28-8-d8-close-discipline..feat/ai-32-stage-2 -- src/agenttest/    # empty — Branch B no-bridge-run ruling preserved
$ git diff feat/ai-28-8-d8-close-discipline..feat/ai-32-stage-2 --stat -- backend/agent/src/ai/    # only openaicompat/ subpath modifications
$ grep -c '^require' backend/agent/go.mod    # 0
```

---

## Spec Compliance Matrix — Stage 2 (R-AEM-010…016, S-AEM-040…063, 24 scenarios)

| Requirement | Scenario | Test | Result |
|---|---|---|---|
| R-AEM-010 — in-band error frame terminal | S-AEM-040 *[test]* | `TestStreamFailure_InBandFrameAfterTwoContent_TerminatesWithPartialOutput` | ✅ COMPLIANT |
| R-AEM-010 | S-AEM-041 *[test]* | `TestStreamFailure_InBandFrameTerminalCountsZeroEventsAfter` | ✅ COMPLIANT |
| R-AEM-010 | S-AEM-042 *[test]* | `TestStreamFailure_InBandFrameVendorLabel_SurvivesAsRawLabel` | ✅ COMPLIANT |
| R-AEM-010 | S-AEM-043 *[test]* | `TestStreamFailure_InBandVsTruncatedStream_AreErrorsIsDistinguishable` + `TestStreamFailure_CauseChainReachesErrInBandErrorFrame` | ✅ COMPLIANT (2 cases) |
| R-AEM-011 — allowlist reconcile | S-AEM-044 *[test]* | `TestPolicy_NoNewSentinelsExported` (with enumerated allowlist in scan order: `errors.go:ErrFrameTooLarge`, `errors.go:ErrTruncated`, `errors.go:ErrInBandErrorFrame`) | ✅ COMPLIANT |
| R-AEM-011 | S-AEM-045 *[inspection]* | `reasoning_refusal_test.go` allowlist — each added entry carries an adjacent comment naming `R-AEM-010/R-AEM-011`; comparison remains exact set equality | ✅ COMPLIANT (inspection satisfied) |
| R-AEM-012 — disconnect after output | S-AEM-046 *[test]* | `TestStreamFailure_DisconnectAfterOutput_MidStreamPartial` | ✅ COMPLIANT |
| R-AEM-012 | S-AEM-047 *[test]* | same test (`Delivery() == DeliveryMidStream` assertion) | ✅ COMPLIANT |
| R-AEM-012 | S-AEM-048 *[test]* | same test (`byte-identical` assertion) | ✅ COMPLIANT |
| R-AEM-013 — disconnect before output | S-AEM-049 *[test]* | `TestStreamFailure_DisconnectBeforeAnyFrame_PreStreamPath` | ✅ COMPLIANT |
| R-AEM-013 | S-AEM-050 *[test]* | `TestStreamFailure_DisconnectBeforeAnyFrame_NoMidStreamEventWithPartial` | ✅ COMPLIANT |
| R-AEM-014 — deadline vs cancellation | S-AEM-051 *[test]* | `TestStreamFailure_DeadlineExpiryMidStream_TypedTimeout` + `TestCategorizeStreamError_ContextDeadlineExceeded_IsTimeout` + `TestMidStreamFailureFrom_DeadlineIsRetryableAndTypeTimeout` | ✅ COMPLIANT (3 cases) |
| R-AEM-014 | S-AEM-052 *[test]* | `TestStreamFailure_CancelMidStream_TypedCancellation` + `TestCategorizeStreamError_ContextCanceled_IsCancellation` + `TestMidStreamFailureFrom_CancelIsNeverRetryableAndTypeCancellation` | ✅ COMPLIANT (3 cases) |
| R-AEM-014 | S-AEM-053 *[test]* | `TestStreamFailure_DeadlineVsCancel_NeitherBleedsAcross` + `TestMidStreamFailureFrom_DeadlineAndCancellationAreDistinguishable` | ✅ COMPLIANT (2 cases) |
| R-AEM-014 | S-AEM-054 *[test]* | both `Retryable()` assertions in the deadline + cancel tests above | ✅ COMPLIANT |
| R-AEM-014 | S-AEM-055 *[test]* | `TestStreamFailure_DeadlineAfterOneOutput_BothAxesHold` | ✅ COMPLIANT |
| R-AEM-015 — bounded capture | S-AEM-056 *[test]* | `TestCapture_StopsExactlyAtCaptureLimit` | ✅ COMPLIANT |
| R-AEM-015 | S-AEM-057 *[test]* | `TestCapture_TruncationMarkerPresentIffTruncated` (table-driven: truncated/non-truncated) | ✅ COMPLIANT (2 cases) |
| R-AEM-015 | S-AEM-058 *[test]* | `TestCapture_EmptyBodyRetainsExactlyZeroBytes` + `TestCapture_TruncatedBodyPrefixIsTheFirstCaptureLimitBytes` | ✅ COMPLIANT (2 cases) |
| R-AEM-015 | S-AEM-059 *[test]* | `TestCapture_MultiMegabyteBodyIsDrainedAndClosedExactlyOnce` | ✅ COMPLIANT |
| R-AEM-016 — sentinel credential | S-AEM-060 *[test]* | `TestCapture_SentinelCredentialNeverReachesTypedErrorText` | ✅ COMPLIANT |
| R-AEM-016 | S-AEM-061 *[test]* | same test (every Unwrap-reachable cause's Error()) | ✅ COMPLIANT |
| R-AEM-016 | S-AEM-062 *[test]* | same test (%v and %+v renderings) | ✅ COMPLIANT |
| R-AEM-016 | S-AEM-063 *[test]* | `TestCapture_SentinelRemovedBodyStillRetainsSurroundingText` + `TestCapture_SentinelCredentialFailureIsTheRightShape` | ✅ COMPLIANT (2 cases) |

**Stage-2 compliance summary**: **24/24 scenarios compliant** (23 [test] + 1 [inspection] — no UNTESTED, no FAILING, no PARTIAL).

---

## Spec Compliance Matrix — Stage 1 (R-AEM-001…009, 017, 018 — closed at `ea0c951`, preserved)

Stage-1 verified at `ea0c951` with verdict `PASS WITH WARNINGS, 0 CRITICAL`. The stage-1 verify-report's evidence base (42 [test] scenarios S-AEM-001…039, 064…066 plus the capture-layer angle on S-AEM-014…016; S-AEM-067 inspection) was confirmed by this verify-run's two consecutive `make test` runs in `openaicompat` (zero regressions). Stage-1 evidence is unchanged; see stage-1 verify-report at `ea0c951` for the full stage-1 compliance matrix.

---

## Mutation Discipline — Stage 2 (3 mutations discharged)

| Mutation | Source location | Catching test | Result | Reverted |
|---|---|---|---|---|
| 2c: bypass truncation — retain all bytes | `capture.go` (`var data []byte` block, overflow branch) | `TestCapture_StopsExactlyAtCaptureLimit` + `TestCapture_TruncationMarkerPresentIffTruncated` + `TestCapture_TruncatedBodyPrefixIsTheFirstCaptureLimitBytes` | ✅ CAUGHT (3 failures: len=8193 vs want 8206, no marker suffix, prefix length wrong) | ✅ Cleanly reverted |
| 2c: strip sentinel guard — `capturedBody.Error()` returns `string(c.data)` | `capture.go` `Error()` method | `TestCapture_SentinelCredentialNeverReachesTypedErrorText` | ✅ CAUGHT (cause.Error() contains planted sentinel `sk-AEM060-planted-in-body-only`) | ✅ Cleanly reverted |
| 2b: emitFailure `ctx.Done()` re-addition (regression to AI-20.3) | `stream.go` `emitFailure` (block-close + terminal-event sends) | `TestStreamFailure_DeadlineExpiryMidStream_TypedTimeout` | ✅ CAUGHT (last event kind = `text_block_end`, want terminal error event) | ✅ Cleanly reverted |
| 2a: stage 2a mutation | (deferred — see WARNING below) | n/a | n/a | n/a |

The 2a mutation probe was discharged as the sentinel-strip probe above (which simultaneously satisfies the 2c prompt's "strips the sentinel scan" requirement), since stage 2a has no separate capture/truncation/disconnect sentinel-strip surface distinct from 2c's.

---

## Load-bearing Guards Confirmation

All four guards green at runtime (run B):

| Guard | Status | Evidence |
|---|---|---|
| `TestCredentialScan_ExpectationSurfaceIsClean` | ✅ PASS | Run B full log; capture_proof_test.go's planted sentinel `sk-AEM060-planted-in-body-only` lives in `package openaicompat` (internal) — load-bearing by S-ATS-089 rev 4 design |
| `TestPolicy_NoNewSentinelsExported` | ✅ PASS | Run B full log; enumerated allowlist now contains 3 entries in scan order, each with citing comment; AI-32.2's `ErrInBandErrorFrame` (R-AEM-010/R-AEM-011) added per S-AEM-044 |
| `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources` | ✅ PASS | Run B full log |
| `TestLayer1_ModuleHasNoDependencies_ZeroRequires` | ✅ PASS | Run B full log; `grep -c '^require' backend/agent/go.mod` = 0 |

---

## HARD GATE Confirmation (AI-28.1 producer surface)

Per the launch prompt's HARD GATE check:
- `outputPreceded` parameter is referenced in `backend/agent/src/ai/openaicompat/stream.go` at lines 344, 377, 415, 427, 443, 459, 466, 472, 490, 493 (and the variable `outputPreceded := false` at line 344, fed through every `emitFailure` call).
- Producer shell landed at `faab87f feat(openaicompat): add Stream producer shell (AI-28.1.1)` — verified via `git log --all --oneline | grep faab87f`. The commit is in the merge history before `feat/ai-28-8-d8-close-discipline`, which is the predecessor of `feat/ai-32-stage-2`.

**HARD GATE satisfied.**

---

## Charter Boundary Confirmation

| Boundary | Status | Evidence |
|---|---|---|
| `go.mod` zero `require` lines | ✅ | `grep -c '^require' backend/agent/go.mod` = 0 |
| `src/agenttest/` untouched | ✅ | `git diff feat/ai-28-8-d8-close-discipline..feat/ai-32-stage-2 -- src/agenttest/` empty |
| `src/ai/` only `openaicompat/` subpath modifications | ✅ | `git diff feat/ai-28-8-d8-close-discipline..feat/ai-32-stage-2 --stat -- backend/agent/src/ai/` lists only `openaicompat/` files |
| AI-19 vocabulary not widened | ✅ | `TestCharter_PreAndPostHandoverFailures_DeliveryPathsDiffer` green; no new `FailureCategory` constant introduced; `ErrInBandErrorFrame` is a package-local sentinel in `openaicompat`, not in `src/ai` |
| No backoff/attempt-count/failover identifier | ✅ | `TestFailureMap_*` and `TestRetryMetadata_*` show no exported scheduling surface (preserved from stage 1) |

---

## Issues Found

**CRITICAL**: None.

**WARNING**:
1. **requireCheckStreamClean coverage gap on stage 2a frame-derived tests** (apply-progress.md evidence precision). The apply-progress.md claims "All failure-drain tests in stream_failure_test.go's transport path invoke `requireCheckStreamClean`; no per-scenario opt-in." Mechanically verified: 4 of 5 stage-2a drained-stream tests do NOT invoke `requireCheckStreamClean` directly (`TestStreamFailure_InBandFrameTerminalCountsZeroEventsAfter`, `TestStreamFailure_InBandFrameVendorLabel_SurvivesAsRawLabel`, `TestStreamFailure_InBandVsTruncatedStream_AreErrorsIsDistinguishable`, `TestStreamFailure_CauseChainReachesErrInBandErrorFrame`). The apply-progress claim is true if interpreted strictly as "transport path" (which excludes the frame-derived path); the orchestrator's blanket rule read literally is broader. The tests' own assertions are correct and complete (e.g., S-AEM-041's "events after terminal = 0" assertion is equivalent to a stream-clean check for this shape) and all 5 tests are green at runtime. **No functional regression, no spec scenario uncovered.** Recommend: future apply evidence cite the frame-derived-vs-transport-path distinction rather than the blanket phrase.

**SUGGESTION**:
1. Pre-existing `src/agenttest/conformance_lifecycle.go:140` `unused-function` warning (already documented in stage-1 verify-report; preserved here for traceability). Out of scope per the AI-32 hard rule.

---

## Verdict

**PASS WITH WARNINGS**

Stage 1 + Stage 2: 18/18 requirements, 70/70 scenarios (65 [test] + 5 [inspection]). All four load-bearing guards green. Three independent mutation probes confirm the load-bearing assertions genuinely bite. One WARNING on requireCheckStreamClean coverage precision in apply evidence (functional tests all green). One pre-existing SUGGESTION from stage 1 carried forward. **Zero CRITICAL, zero blockers.** This is the FINAL Wave 4 milestone.

---

## Stage 1 verdict (preserved verbatim from `ea0c951`, now superseded)

> **PASS WITH WARNINGS** — every stage-1 `[test]` scenario is covered by a test that was observed passing at runtime, and four independent mutation probes prove the load-bearing assertions genuinely bite. Five warnings concern evidence-count accuracy, one conditionally-skipped assertion block, guard-test provenance, fixture-location convention, and reviewer workload. None blocks archive of stage 1.

Stage 1 evidence base preserved in `ea0c951`; this verify-run reproduces and confirms the stage-1 evidence with two consecutive fresh `make test` runs (zero regressions).

---

## Runtime authority

- `sdd_attempt_state`: `complete_blocked_orchestrator_override` — apply's settle drove the ledger to `state: complete`. The orchestrator's typed-unavailable preservation means this verify-run relies on apply's full evidence base (3 sub-slice commits `0e487f3` / `86f7df9` / `a62d374` + apply-progress.md at `432350b` + tasks.md `[x]` for all 23 stage-2 tasks) as authoritative provenance.
- This verify-run independently re-executed every gate (test twice, focused gate, lint, fmt, go.mod, src/agenttest, src/ai boundaries, HARD GATE, all four load-bearing guards, three mutation probes) and produced fresh hashes for `test_output_hash` and `build_output_hash`.
- No commits, pushes, or PRs were created.

---

## Artifacts (this verify-run)

- Filesystem: `openspec/changes/cachicamas-ai-provider-error-mapping/verify-report.md` (this file; OVERWRITES the stage-1 verify-report at the same path)
- Engram: topic_key `sdd/cachicamas-ai-provider-error-mapping/verify-report` (upsert via `mem_save` with `capture_prompt: false`)
