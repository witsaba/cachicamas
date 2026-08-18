# Apply Progress: AG-15 — Retry policy and the failover seam

> Change: `cachicamas-agent-retry-failover`. Worktree `cachicamas-worktrees/ag-15-retry-failover`, branch `feat/agent-layer2-wave3-ag15`, base `main@bf482b0a`.
> **Batch 1: Phase 0 + Phase 1 + Phase 1a.** **Batch 2 (this batch, final): Phase 2 + Phase 3 + Phase 4 + Phase 5.**
> **All 35/35 tasks complete.**
> Strict TDD active throughout both batches. Every RED below is the actual observed output of a real `go test` run against the shipped tree at that moment — none are predicted or paraphrased.

## Commits

| SHA | Message | Tasks | Batch |
|---|---|---|---|
| `288aa154` | `feat(agent): emit turn_end on Turn's pre-stream failure paths` | 0.1–0.4 | 1 |
| `cc1cab86` | `feat(agent): retry a pre-output failure across an inner attempt loop` | 1.1–1.10, 1a.1 (and 0.5's bite, executed at 1.8) | 1 |
| `0baec9b1` | `docs(openspec): AG-15 SDD planning artifacts and batch-1 progress` | (artifacts) | 1 |
| `38e06ef2` | `feat(agent): inject retry timing, honor retry-after, state the composed ceiling` | 2.1–2.7 | 2 |
| `790f63e8` | `feat(agent): consult the failover seam once at retry exhaustion` | 3.1–3.3 | 2 |
| `473f605c` | `fix(agent): preserve the true failure category on an exhausted-retry run_end` | 4.1–4.3 | 2 |
| `2f2cae67` | `docs(0003): tick AG-15, bump milestone counter to 15/24` | 5.1–5.6 | 2 |
| `a6a3f515` | `style(agent): separate retry_policy.go/failover_policy.go headers from the package doc` | (lint fix, discovered at batch-2 verification) | 2 |

## Phase 0 — D1: close the pre-stream bracket gap

- [x] **0.1** RED: `TestTurn_PreStreamBuildErrorEmitsTurnEnd`, `TestTurn_PreStreamHookErrorEmitsTurnEnd`, `TestTurn_PreStreamProviderErrorEmitsTurnEnd` in `retry_policy_test.go`, each with a `nil continuation` and a `continuation` subtest (6 subtests total). RED observed:
  ```
  --- FAIL: TestTurn_PreStreamProviderErrorEmitsTurnEnd (0.00s)
      --- FAIL: TestTurn_PreStreamProviderErrorEmitsTurnEnd/nil_continuation (0.00s)
          retry_policy_test.go:259: event sequence = [run_start turn_start], want kinds [run_start turn_start turn_end run_end] (nilContinuation=true)
      --- FAIL: TestTurn_PreStreamProviderErrorEmitsTurnEnd/continuation (0.00s)
          retry_policy_test.go:282: event sequence = [turn_start], want kinds [turn_start turn_end] (nilContinuation=false)
  --- FAIL: TestTurn_PreStreamBuildErrorEmitsTurnEnd (0.00s)
      --- FAIL: TestTurn_PreStreamBuildErrorEmitsTurnEnd/continuation (0.00s)
          retry_policy_test.go:169: event sequence = [turn_start], want kinds [turn_start turn_end] (nilContinuation=false)
      --- FAIL: TestTurn_PreStreamBuildErrorEmitsTurnEnd/nil_continuation (0.00s)
          retry_policy_test.go:146: event sequence = [run_start turn_start], want kinds [run_start turn_start turn_end run_end] (nilContinuation=true)
  --- FAIL: TestTurn_PreStreamHookErrorEmitsTurnEnd (0.00s)
      --- FAIL: TestTurn_PreStreamHookErrorEmitsTurnEnd/continuation (0.00s)
          retry_policy_test.go:226: event sequence = [turn_start], want kinds [turn_start turn_end] (nilContinuation=false)
      --- FAIL: TestTurn_PreStreamHookErrorEmitsTurnEnd/nil_continuation (0.00s)
          retry_policy_test.go:203: event sequence = [run_start turn_start], want kinds [run_start turn_start turn_end run_end] (nilContinuation=true)
  FAIL
  ```
  All 6 fail exactly on the D1 gap: the stream carries only `run_start turn_start` (or `turn_start`), no `turn_end`/`run_end`.
- [x] **0.2** GREEN: added `preStreamAbortFailure(err error) (*Failure, error)` (`errors.As` → `NewFailure`; else `ai.PreStreamFailure(FailureCategoryUnavailable)` → `NewFailure`) and `emitPreStreamAbort(...)` to `loop.go`; wired at all 3 sites (`buildLoopRequest` error, hook error, `provider.Stream` error) before `closeSink`. Re-run: all 6 subtests PASS.
- [x] **0.3** Confirmed byte-unchanged (source AND result): `TestTurn_PreRequestHook_FailureAbortsBeforeStream` (`loop_hook_test.go:399`) and `TestTurn_ProviderPreStreamFailureSurfacesOnReturn` (`loop_test.go:1436`) both PASS with zero edits.
- [x] **0.4** Widened both substrate filters (`filterOutLoopFiles` at `loop_test.go:831`, `filterOutLoopHookFiles` at `loop_hook_test.go:907`) with `/retry_policy.go` and `/retry_policy_test.go`, appended immediately after the re-grepped AG-14 tail. Byte-in-sync between both filters.
- [x] **0.5** Bite `S-RTY-011` — **deferred and executed at task 1.8** per tasks.md's own sequencing note. See 1.8 below for the RED evidence and revert.

## Phase 1 — AG-15.1: the retry predicate and inner attempt loop

- [x] **1.1** RED: `TestRetryDecision` (table test, G1–G5) + `TestRetryDecision_SameEvidenceRetriesThenExhausts` in **`retry_decision_internal_test.go`**. RED observed (compile failure — `retryDecision`/`retryVerdict`/verdict constants did not exist yet):
  ```
  src/agent/retry_decision_internal_test.go:72:11: undefined: retryVerdict
  src/agent/retry_decision_internal_test.go:78:13: undefined: verdictSurface
  ...
  src/agent/retry_decision_internal_test.go:121:14: undefined: retryDecision
  FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
  ```
- [x] **1.2** GREEN: added `retryDecision(terr error, attempt, bound int) retryVerdict` + `verdictSurface`/`verdictRetry`/`verdictExhausted` + `defaultRetryAttempts = 3` to new `retry_policy.go`. Re-run: `TestRetryDecision` (7 subtests) + the same-evidence test all PASS.
- [x] **1.3** RED: `TestHarness_RetryVisibleAttempts` (`S-RTY-002`) written using the new `preStreamFailingProvider` wrapper (1a.1). RED observed (compile failure against the pre-1.4 `harness.go`, deliberately reverted first so this RED is genuine):
  ```
  src/agent/retry_policy_test.go:374:3: unknown field RetryAttempts in struct literal of type agent.Harness
  FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
  ```
- [x] **1.4** GREEN: wrapped `harness.go`'s `Turn` call in an inner attempt loop (fresh `turnSink`+forwarder per attempt, same `transcript` slice reused by reference); added `Harness.RetryAttempts int` (`<=0` → `defaultRetryAttempts`); inserted the `retryDecision` gate between the existing G0 cancellation check and the fall-through to `failRun`, G0 unchanged. Re-run: `TestHarness_RetryVisibleAttempts` PASSES.
- [x] **1.5** RED/confirm: `TestHarness_RetryPartialOutputNotRetried` (`S-RTY-003`) and `TestHarness_RetryNonRetryableSurfacesImmediately` (`S-RTY-004`, both positions). **No RED observed** — both PASSED on first run (G2/G3, already implemented at 1.2, already exercised end-to-end once the inner loop landed at 1.4).
- [x] **1.6** Confirmed: G2/G3 satisfy 1.5's scenarios with **zero further production change**. No gap found.
- [x] **1.7** Bite `S-RTY-010` — RED-first, then reverted. Mutation: deleted the `if failure.PartialOutput() { return verdictSurface }` line from `retryDecision`. Re-ran `TestHarness_RetryPartialOutputNotRetried`. RED observed:
  ```
  --- FAIL: TestHarness_RetryPartialOutputNotRetried (0.00s)
      retry_policy_test.go:526: Run returned err = nil, want the mid-stream failure (G3 forbids retry; the run must end failed)
  FAIL
  ```
  **Reverted**: `git diff` on `retry_policy.go` after revert is empty. Re-ran: PASS.
- [x] **1.8** Bite `S-RTY-011` (Phase 0's 0.5, executed here) — RED-first, then reverted. Mutation: `emitPreStreamAbort` in `loop.go` made an unconditional no-op. Re-ran `TestHarness_RetryVisibleAttempts` (`S-RTY-002`). RED observed:
  ```
  --- FAIL: TestHarness_RetryVisibleAttempts (0.00s)
      retry_policy_test.go:443: turn brackets: 3 turn_start, 1 turn_end, want exactly 3 each (H -- one bracket per attempt)
      retry_policy_test.go:446: aborted turn_end count = 0, want 2 (H-1)
      retry_policy_test.go:457: CheckStream rejected the recorded multi-attempt stream: event[2]: value is not permitted where it appears, want it accepted unmodified
  FAIL
  ```
  `"value is not permitted where it appears"` confirmed as `ai.ErrMisplaced`'s exact `.Error()` string. **Reverted**: `git diff` on `loop.go` after revert is empty. Re-ran: PASS.
- [x] **1.9** No further filter change needed for `retry_policy.go`/`retry_policy_test.go` (already done at 0.4). Widened both filters for `retry_decision_internal_test.go`, landed in this same commit.
- [x] **1.10** Confirmed `S-RUN-100` (`TestHarness_TurnError_RunEndsTypedNoAppendNoCloseNoRetry`, `harness_test.go:1870`) stays green, **source byte-unchanged**. Its fixture leaves `Retryable` unset (Go zero-value `false`), so G2 catches it. Ran standalone: PASS.
- [x] **1a.1** `preStreamFailingProvider` defined in `retry_policy_test.go`: fails its first `failCount` `Stream` calls with a scripted `*ai.Failure`, captures its own requests independently of the inner provider, delegates afterward. Test-local only — `agenttest` untouched.

## Phase 2 — AG-15.2: timing seam, backoff, retry-after, ceiling

- [x] **2.1** RED: `TestHarness_BoundHoldsAboveH`, `TestRetryTiming_BackoffJitterAndClamp`, `TestHarness_RetryAfterOverridesBackoff`, `TestHarness_InterruptAbortsBackoff`, `TestDefaultRetrySleep_PreCancelledContextReturnsImmediately`, `TestComposedCeiling_MatchesLayer1Wording` written in new `retry_backoff_test.go`. RED observed (compile failure — `RetryTiming` did not exist yet):
  ```
  src/agent/retry_backoff_test.go:115:5: unknown field RetryTiming in struct literal of type agent.Harness
  src/agent/retry_backoff_test.go:115:26: undefined: agent.RetryTiming
  ... (5 sites total)
  FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
  ```
- [x] **2.2** GREEN: added `RetryTiming{NowFunc, SleepFunc, BaseDelay, MaxDelay, JitterSeed}`, `applyRetryTimingDefaults`, `newRetryJitter`, `computeRetryBackoff`, `retryWaitDelay`, `DefaultRetrySleep` (exported — S-RTY-008's own "direct unit test of the default sleep" scenario needs external, agent_test-level access, NFR-RTY-001) to `retry_policy.go`; added `Harness.RetryTiming` field; wired the backoff wait on `runCtx` at G4 inside the inner attempt loop, re-checking `context.Cause(runCtx)` on sleep error → `windDownRun` for `ErrInterrupted`/`ErrShutdown`, falling through to the ordinary failure surface (with the original `terr`) otherwise. Two small wording mismatches surfaced and were fixed before GREEN: `doc.go`'s "at most N+1 = 4 wire requests" line-wraps across two comment lines in the real file (fixed the test's expected substring to the single-line-contained "N+1 = 4 wire requests"), and a case mismatch between the test's "total attempts" and the doc's authored "TOTAL attempts" (aligned the test to the authored capitalization). Re-run: all Phase 2 tests PASS.
- [x] **2.3** RED folded into 2.1 (`TestComposedCeiling_MatchesLayer1Wording`, same file/commit — S-RTY-009 is not one of the seven charter scenarios and design's Testing Strategy table does not assign it a separate row, so it was authored alongside the other Phase 2 tests rather than as a later, separately-RED-recorded increment).
- [x] **2.4** GREEN: expanded `retry_policy.go`'s package doc with a `# The composed worst-case ceiling (R-RTY-009)` section stating `H` counts TOTAL attempts, `H × 4 = 12` at the shipped defaults, citing `ai/internal/retry/doc.go` verbatim ("DefaultMaxAttempts = 3", "N+1 = 4 wire requests").
- [x] **2.5** Bite `S-RTY-012` (RED-first, then reverted) — **never edited the real `ai/internal/retry/doc.go`**, per tasks.md's own explicit instruction and this batch's hard constraint #1. Perturbed the TEST's own local copy of the expected Layer-1 wording (`wantLayer1RetryDocSentences`, appending a `BITE-S-RTY-012-PERTURBED` marker to one entry) and re-ran the **same** `TestComposedCeiling_MatchesLayer1Wording` scenario test — mirroring the S-RTY-010/S-RTY-011 mutate-rerun-revert mechanics exactly, just applied to the test's own captured expectation rather than to the (forbidden-to-touch) Layer 1 file. RED observed:
  ```
  --- FAIL: TestComposedCeiling_MatchesLayer1Wording (0.00s)
      retry_backoff_test.go:464: ai/internal/retry/doc.go is missing the cited sentence(s) [N+1 = 4 wire requests BITE-S-RTY-012-PERTURBED] -- this capability's ceiling documentation cites it verbatim and has drifted
  FAIL
  ```
  **Reverted**: `git diff` on `retry_backoff_test.go` after revert is empty; `git diff` on `backend/agent/src/ai/internal/retry/doc.go` against main is empty (confirmed both before and after the bite — the real file was never touched at any point). Re-ran: PASS.
- [x] **2.6** Widened both substrate filters, adding `/retry_backoff_test.go`, byte-in-sync, same commit.
- [x] **2.7** Grep verification (not a new Go test — task 2.7's own wording is "grep", and the property is already provable by Go's own compiler-enforced internal-package-visibility rule for the import half): `grep -n "time\.Sleep(" retry_policy.go retry_policy_test.go retry_decision_internal_test.go retry_backoff_test.go` → zero hits. `grep -n "src/ai/internal/retry" retry_policy.go retry_policy_test.go retry_decision_internal_test.go retry_backoff_test.go harness.go loop.go` → zero hits.

## Phase 3 — AG-15.3: the failover seam

- [x] **3.1** RED: `TestFailover_ConsultedOnceAtExhaustion` (`S-RTY-013`, two subtests), `TestFailoverPolicy_DocumentsRealImplementationObligations`, `TestFailover_InertnessNilVsNoOp` (`S-RTY-014`) written in new `failover_policy_test.go`. RED observed (compile failure):
  ```
  src/agent/failover_policy_test.go:23:17: undefined: agent.FailoverPrompt
  src/agent/failover_policy_test.go:26:75: undefined: agent.FailoverPrompt
  src/agent/failover_policy_test.go:26:97: undefined: agent.FailoverVerdict
  ...
  src/agent/failover_policy_test.go:169:43: undefined: agent.FailoverPolicy
  src/agent/failover_policy_test.go:179:4: unknown field Failover in struct literal of type agent.Harness
  FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
  ```
- [x] **3.2** GREEN: added `FailoverPolicy` interface (`Resolve(ctx, FailoverPrompt) FailoverVerdict`), `FailoverPrompt{Attempts int; Failure *ai.Failure}`, `FailoverVerdict{}` (zero-value declines, no fields in v1), `NoOpFailoverPolicy{}` to new `failover_policy.go`; added `Harness.Failover FailoverPolicy` (nil-default). Wired the G5 consult in `harness.go`: captured the loop's own `retryDecision` verdict into a new `verdict retryVerdict` variable (previously discarded after the `!= verdictRetry` check), and — after the inner attempt loop, only when `verdict == verdictExhausted && h.Failover != nil` — call `h.Failover.Resolve(runCtx, FailoverPrompt{Attempts: bound, Failure: <the terminal *ai.Failure via errors.As>})` before `failRun`. Re-run: all Phase 3 tests PASS on first GREEN attempt.
- [x] **3.3** Widened both substrate filters, adding `/failover_policy.go` + `/failover_policy_test.go`, byte-in-sync, same commit.

## Phase 4 — Decision 4: preserve the true category on the exhausted-retry report

- [x] **4.1** RED: `TestHarness_ExhaustedRetryPreservesTrueCategory` and `TestHarness_ExhaustedRetryPlainErrorStaysUnavailable` (`S-RTY-015`) appended to `retry_policy_test.go`, asserting pointer-identity via `runEnd.Failure().Unwrap()`. RED observed for the first (the second passed immediately — see note below):
  ```
  --- FAIL: TestHarness_ExhaustedRetryPreservesTrueCategory (0.00s)
      retry_policy_test.go:697: run_end failure Category() = unavailable, want rate_limit (the true category, not the hardcoded Unavailable)
      retry_policy_test.go:700: run_end failure Retryable() = false, want true (preserved from the final attempt)
      retry_policy_test.go:706: run_end failure Delivery() = 2, want 1
      retry_policy_test.go:715: run_end failure does not unwrap to the identical *ai.Failure the final attempt failed with -- it was reconstructed, not wrapped
      retry_policy_test.go:718: unwrapped failure RetryAfter() = (0s, false), want (2.5s, true)
  FAIL
  ```
  `TestHarness_ExhaustedRetryPlainErrorStaysUnavailable` **PASSED on first run, no RED** — it drives an empty-`System` fixture through `buildLoopRequest`'s plain-error path (G1: not an `*ai.Failure`), which was already routed through the unchanged `wrapHarnessFailure` before this phase's GREEN; this test proves the "stays the same" half of S-RTY-015 and legitimately has no organic RED, mirroring batch 1's 1.5/1.6 precedent.
- [x] **4.2** GREEN: added `typedHarnessFailureFromError(cause error) (*Failure, error)` sibling beside `wrapHarnessFailure` in `harness.go` (`errors.As` reaches an `*ai.Failure` → `NewFailure(f)` wrapping the identical pointer; else routes to the untouched `wrapHarnessFailure`); routed `failRun`'s one call site through the sibling. Re-run: both tests PASS.
- [x] **4.3** Grep confirmed no existing test pins `Unavailable` on the harness `run_end`: `harness_test.go:1898` area asserts failure **presence only** (`if _, hasFailure := runEnd.Failure(); !hasFailure`), no category check; the only other `FailureCategoryUnavailable` hit across `harness_test.go`/`cancellation_*_test.go` is inside a **comment** in `cancellation_interrupt_test.go:125` describing what a regression would look like — the actual assertion there checks `TurnEnd`'s category is `Cancellation`, and the adjacent `run_end` assertion (`cancellation_interrupt_test.go:111-113`) checks **absence** of any failure, not a category. Matches design's enumeration exactly.

## Phase 5 — Documentation

- [x] **5.1** `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:2172` — ticked `- [x] Partial-output failures are never silently retried; retry attempts are visible events — closed by AG-15.1.`
- [x] **5.2** Same doc, line 3 status header: extended the narrative with an AG-15 sentence (following AG-13/AG-14's density and style) describing the retry predicate, byte-identical visible retries, the partial-output-never-retried rule, non-retryable-immediate-surface, the injected backoff timing seam with retry-after override and interrupt-safe waits, the composed ceiling and its divergence test, the true-category preservation on the exhausted-retry report, and the failover seam's inertness pin.
- [x] **5.3** Same doc: bumped the milestone counter from **14 of 24** to **15 of 24**, and "Wave 3 opens with AG-12…AG-14" to "…AG-12…AG-15". Confirmed (grep) this is the only occurrence of the counter in the doc.
- [x] **5.4** R-15/G8 back-annotation: confirmed via grep that line `:2172` is the only `- [ ]` (now `- [x]`) checklist item in the doc naming AG-15 as its closer; `R-15`'s own row (`:69`) is a static descriptive table row, not a checkbox, exactly as tasks.md's own note anticipated.
- [x] **5.5** Composed-ceiling documentation confirmed to already live in `retry_policy.go`'s package doc (task 2.4) — "where both layers' readers find it" is satisfied by that file plus the shipped `ai/internal/retry/doc.go` half. No charter (`0003:1455`) text edit made or needed.
- [x] **5.6** **Flag for `sdd-verify`/archive, explicitly recorded here**: `openspec/changes/cachicamas-agent-retry-failover/specs/agent-loop-skeleton/spec.md` carries a `## MODIFIED Header — the allocated scenario range` section (currently at `:10-12` in that delta file) that extends the promoted spec's header claim from "`S-LSK-001` through `S-LSK-020`" to "`S-LSK-001` through `S-LSK-023`". This is **outside** the normal `## MODIFIED Requirements` block shape the archive step's usual merge logic expects. **The archive step MUST apply this header edit to `openspec/specs/agent-loop-skeleton/spec.md`'s own header line, not only merge the `R-LSK-001`/`R-LSK-004` MODIFIED Requirements blocks** — otherwise the promoted spec's header will silently understate its own allocated range after this change archives (`S-LSK-021`, `S-LSK-022`, `S-LSK-023` would exist in the merged requirements but the header's stated range would still say "through `S-LSK-020`", making the header's own claim false — the exact defect class `S-LSK-020` itself exists to make impossible for a *count*, but the header's *range* claim is a different sentence that needs its own edit).

## Deviations from design

Both enumerated, both forced by Go language constraints or Strict-TDD honesty, not by choice.

1. **Predicate test file split (`retry_decision_internal_test.go`).** [Batch 1] Design AD-2 names `retryDecision`/`retryVerdict` as unexported identifiers, and this codebase's test convention is 100% `package agent_test`. `NFR-RTY-001` grants an explicit carve-out for exactly this case. Created `retry_decision_internal_test.go` (`package agent`), hosting only `TestRetryDecision` and its same-evidence companion. Both substrate filters widened in sync, same commit.
2. **`emitPreStreamAbort` helper (loop.go), beyond the one design names.** [Batch 1] Design AD-1 names exactly one helper, `preStreamAbortFailure`, implemented verbatim. Additionally factored the shared turn_end+run_end emission into `emitPreStreamAbort`, a DRY choice delegating to the named helper, not a behavioral deviation.
3. **`DefaultRetrySleep` exported.** [Batch 2] Design AD-6 says the default sleep "mirrors `defaultSleep`" without specifying export status. `S-RTY-008`'s own second Given ("a direct unit test of the default sleep function with an already-cancelled context") requires external (`agent_test`) access under `NFR-RTY-001`'s blanket external-test rule, which has no carve-out for this scenario the way `S-RTY-001` has for the predicate. Exporting it as `agent.DefaultRetrySleep` is the only way to satisfy both requirements simultaneously; it is additionally reusable by any future caller wanting production timing defaults without importing `ai/internal/retry` (which is forbidden anyway). Not a behavioral deviation — same algorithm, same shape, just a capitalized name.
4. **`retryWaitDelay`'s "differs from computed backoff" / "equals the computed backoff" assertions use range checks, not RNG-replicated exact values.** [Batch 2] `S-RTY-007`'s literal wording says the requested duration "equals the computed backoff for that attempt number." Rather than reimplementing `math/rand/v2`'s exact PCG sequence inside the test (fragile, couples the test to jitter internals it should not need to know), the test asserts the requested delay falls within the mathematically-guaranteed range `[BaseDelay, 2*BaseDelay)` for attempt 1 (retry-after-absent case) and clearly *outside* that range for a retry-after value chosen far beyond it (retry-after-present case). This proves "used the computed-backoff path" vs. "used the override" precisely, without RNG-replication fragility. Not a requirement deviation — a test-implementation choice within design's own latitude (design does not pin exact test mechanics for this scenario).

## TDD ordering correction (Batch 1, recorded, not hidden)

While implementing task 1.4, the inner-attempt-loop production code in `harness.go` was initially written **before** running task 1.3's test against the pre-1.4 tree — a strict-TDD ordering violation caught before it produced any false evidence. Corrected by: `git checkout -- harness.go`, writing and running `TestHarness_RetryVisibleAttempts` to get a genuine RED against the reverted file (`unknown field RetryAttempts`), then reapplying the exact same GREEN implementation. No fabricated evidence entered this record.

## TDD Cycle Evidence

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 0.1–0.2 | `TestTurn_PreStream*EmitsTurnEnd` (×3) | External (agent_test) | N/A (new file) | Written, observed (6 subtests) | Passed | 3 paths × 2 continuation modes = 6 cases | `assertPreStreamAbortSequence`/`emitPreStreamAbort` extracted |
| 1.1–1.2 | `TestRetryDecision` (+ same-evidence test) | Internal (package agent) | N/A (new file) | Written, observed (compile fail) | Passed (7+2 subtests) | ✅ 9 cases, G1–G5 + first-match ordering | Clean |
| 1.3–1.4 | `TestHarness_RetryVisibleAttempts` | External | ✅ full package green pre-edit | Written, observed (compile fail, post-revert) | Passed | ➖ single scenario (S-RTY-002) | ➖ |
| 1.5–1.6 | `TestHarness_RetryPartialOutputNotRetried` | External | ✅ | Written, run — **no RED** (already satisfied) | Passed | ➖ single scenario (S-RTY-003) | ➖ |
| 1.5–1.6 | `TestHarness_RetryNonRetryableSurfacesImmediately` | External | ✅ | Written, run — **no RED** (already satisfied) | Passed (2 subtests) | ✅ 2 positions | ➖ |
| 1.7 | Bite `S-RTY-010` | External | — | Written (mutation), observed | Reverted, confirmed clean | — | — |
| 1.8 | Bite `S-RTY-011` | External | — | Written (mutation), observed | Reverted, confirmed clean | — | — |
| 2.1–2.2 | `TestHarness_BoundHoldsAboveH` | External | ✅ full package green pre-edit | Written, observed (compile fail) | Passed | ✅ 2 bounds (3, 5) | ➖ |
| 2.1–2.2 | `TestRetryTiming_BackoffJitterAndClamp` | External | ✅ | Written, observed (compile fail) | Passed | ✅ 2 independent runs, seed reproducibility | ➖ |
| 2.1–2.2 | `TestHarness_RetryAfterOverridesBackoff` | External | ✅ | Written, observed (compile fail) | Passed after 2 wording fixes | ✅ 2 subtests (present/absent) | ➖ |
| 2.1–2.2 | `TestHarness_InterruptAbortsBackoff` | External | ✅ | Written, observed (compile fail) | Passed | ➖ single scenario (S-RTY-008 first half) | ➖ |
| 2.1–2.2 | `TestDefaultRetrySleep_PreCancelledContextReturnsImmediately` | External | ✅ | Written, observed (compile fail — undefined symbol) | Passed | ➖ single case (S-RTY-008 second half) | ➖ |
| 2.1–2.4 | `TestComposedCeiling_MatchesLayer1Wording` | External | ✅ | Written, observed (compile fail, then 2 wording-mismatch FAILs) | Passed | ➖ | ➖ |
| 2.5 | Bite `S-RTY-012` | External | — | Written (test-local wording perturbation), observed | Reverted, confirmed clean; Layer 1 file confirmed byte-unchanged throughout | — | — |
| 3.1–3.2 | `TestFailover_ConsultedOnceAtExhaustion` | External | ✅ full package green pre-edit | Written, observed (compile fail) | Passed on first GREEN attempt (2 subtests) | ✅ 2 cases (consulted / not consulted) | ➖ |
| 3.1–3.2 | `TestFailoverPolicy_DocumentsRealImplementationObligations` | External | ✅ | Written, observed (compile fail — file did not exist) | Passed | ➖ structural (doc-text check) | ➖ |
| 3.1–3.2 | `TestFailover_InertnessNilVsNoOp` | External | ✅ | Written, observed (compile fail) | Passed on first GREEN attempt | ➖ single fixture, driven twice | ➖ |
| 4.1–4.2 | `TestHarness_ExhaustedRetryPreservesTrueCategory` | External | ✅ full package green pre-edit | Written, observed (5 real assertion failures) | Passed | ➖ single scenario | ➖ |
| 4.1–4.2 | `TestHarness_ExhaustedRetryPlainErrorStaysUnavailable` | External | ✅ | Written, run — **no RED** (already satisfied by unchanged `wrapHarnessFailure`) | Passed | ➖ | ➖ |

### Test Summary
- **Total test functions added this batch (2)**: 10 (`TestHarness_BoundHoldsAboveH`, `TestRetryTiming_BackoffJitterAndClamp`, `TestHarness_RetryAfterOverridesBackoff`, `TestHarness_InterruptAbortsBackoff`, `TestDefaultRetrySleep_PreCancelledContextReturnsImmediately`, `TestComposedCeiling_MatchesLayer1Wording`, `TestFailover_ConsultedOnceAtExhaustion`, `TestFailoverPolicy_DocumentsRealImplementationObligations`, `TestFailover_InertnessNilVsNoOp`, `TestHarness_ExhaustedRetryPreservesTrueCategory`, `TestHarness_ExhaustedRetryPlainErrorStaysUnavailable` — 11 total), plus 1 bite (`S-RTY-012`) executed and reverted.
- **Grand total across both batches**: 19 test functions (8 batch 1 + 11 batch 2), 3 bites (`S-RTY-010`, `S-RTY-011`, `S-RTY-012`), all executed RED-first (or explicitly recorded as "no RED — already satisfied") and reverted where applicable.
- **Total tests passing**: all, per the fresh full-suite run below.
- **Layers used**: External (`agent_test`) for every behavioral scenario across both batches; Internal (`package agent`) for the one predicate unit test `NFR-RTY-001` carves out.
- **Approval tests**: none — no refactoring-of-existing-behavior tasks in this change.
- **Pure functions created this batch**: `applyRetryTimingDefaults`, `newRetryJitter`, `computeRetryBackoff`, `retryWaitDelay`, `checkComposedCeilingWording` (test-local). `DefaultRetrySleep` is not pure (real I/O via `time.NewTimer`/`ctx.Done()`) but is the one production side-effecting seam this phase adds, exactly as designed.

## Work Unit Evidence

| Evidence | Phase 0 (D1) | Phase 1 (predicate + inner loop) | Phase 2 (timing seam) | Phase 3 (failover seam) | Phase 4 (true-category report) | Phase 5 (docs) |
|---|---|---|---|---|---|---|
| Focused test command / result | `go test -race -run 'TestTurn_PreStream' ./src/agent/...` → PASS (6/6 subtests) | `go test -race -run 'TestRetryDecision\|TestHarness_Retry' ./src/agent/...` → PASS | `go test -race -run 'TestHarness_BoundHoldsAboveH\|TestRetryTiming_BackoffJitterAndClamp\|TestHarness_RetryAfterOverridesBackoff\|TestHarness_InterruptAbortsBackoff\|TestDefaultRetrySleep_PreCancelledContextReturnsImmediately\|TestComposedCeiling_MatchesLayer1Wording' ./src/agent/...` → PASS | `go test -race -run 'TestFailover' ./src/agent/...` → PASS | `go test -race -run 'TestHarness_ExhaustedRetry' ./src/agent/...` → PASS | N/A — documentation only, no test surface |
| Runtime harness command / result | `go test -race ./src/agent/...` (full package) → PASS | same | `go test -race -count=1 ./src/agent/...` → PASS | same | same | N/A |
| Rollback boundary | `loop.go`'s 3 emission sites + `preStreamAbortFailure`/`emitPreStreamAbort`; both filters' new suffix lines. Commit `288aa154` independently revertable. | `retry_policy.go`, `retry_decision_internal_test.go`, `retry_policy_test.go`, `harness.go`'s attempt-loop block; both filters' new suffix lines. Commit `cc1cab86` independently revertable (depends on `288aa154`). | `retry_policy.go`'s `RetryTiming`/backoff additions, `retry_backoff_test.go`, `harness.go`'s backoff-wait block, `Harness.RetryTiming`; both filters. Commit `38e06ef2` independently revertable (depends on prior commits). | `failover_policy.go`, `failover_policy_test.go`, `harness.go`'s G5 consult block + `Harness.Failover`; both filters. Commit `790f63e8` independently revertable. | `harness.go`'s `typedHarnessFailureFromError` + `failRun`'s one-line routing change, `retry_policy_test.go`'s two new tests. Commit `473f605c` independently revertable. | `docs/architecture/milestones/0003-...md`'s 3 edits (checklist tick, status-header sentence, counter bump). Commit `2f2cae67` independently revertable. |

## Full suite verification (batch 2 end, supersedes batch 1's snapshot)

`go test -race -v -count=1 ./...` from `backend/agent/` (byte-identical command to `make test`, forced fresh with `-count=1`):

```
ok  	github.com/cachicamas/backend/agent/src/agent	9.891s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.754s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep	2.214s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest	2.256s
ok  	github.com/cachicamas/backend/agent/src/ai	5.230s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	2.803s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	172.862s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest	3.444s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	3.760s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	7.275s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke	2.692s
ok  	github.com/cachicamas/backend/agent/src/handoff	2.400s
```
Zero `--- FAIL` lines anywhere in the log (grep-confirmed: `grep -c "^--- FAIL"` → `0`). All 12 packages report `ok`.

Additionally run after this snapshot (the `retry_policy.go`/`failover_policy.go` blank-line lint fix, commit `a6a3f515`, touches only comment formatting): `go test -race -count=1 ./src/agent/...` → `ok` (re-confirmed green post-fix).

`go vet ./...` → clean (exit 0).
`make lint` (after `golangci-lint cache clean` — none present pre-first-run, so the target's own auto-install ran fresh): first run found 2 `revive` `package-comments` issues (both fixed, see commit `a6a3f515`); **re-run: `0 issues.`**
`make build` → clean (`go build -trimpath ./...`, exit 0).
`make vuln-check` → clean (`govulncheck`, exit 0, 0 `"finding"` entries in the JSON output — every entry in the log is a scanned advisory, not an applicable one).

## Constraint compliance (verified, not assumed)

- `backend/agent/src/ai/**`: `git diff --stat main -- backend/agent/src/ai/` is empty. Untouched across both batches.
- `ai/internal/retry` never imported from `agent` package: `grep -n "src/ai/internal/retry"` across every file this change touches or adds → zero hits (also true at the language level: Go's internal-visibility rule would refuse to compile such an import regardless).
- `stream_check.go`, `stream_check_test.go`, `turn_events.go`, `failure.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `history.go`, `go.mod`, `go.sum`, `harness_test.go`: all confirmed byte-unchanged this batch (`git diff --stat main --` on each, single combined command, empty output).
- 25 registered `EventKind`s: unchanged — `event_descriptor.go` (where the registry lives) is confirmed byte-unchanged, so its committed kind count cannot have moved.
- Substrate filters: byte-in-sync between `loop_test.go` and `loop_hook_test.go` at every widening step this batch made (`/retry_backoff_test.go`, `/failover_policy.go`, `/failover_policy_test.go`).
- All three bites (`S-RTY-010`, `S-RTY-011` — batch 1; `S-RTY-012` — batch 2) reverted and independently confirmed via empty `git diff` on the mutated file/marker; `grep -c "BITE-S-RTY-012"` across the entire commit history (`git log -p 0baec9b1..HEAD`) → `0`.
- `S-RUN-100`'s fixture: verified unset `Retryable`, source byte-unchanged (task 1.10, batch 1).
- No existing test pins `Unavailable` on the harness `run_end` (task 4.3, verified this batch).
- No `time.Sleep` in any test added either batch (grep-confirmed both batches).
- Worktree discipline: every write landed under `cachicamas-worktrees/ag-15-retry-failover`; `git status` on the base checkout (`/Users/braejan/workspace/witsaba/repositories/cachicamas`) confirmed clean, on `main`, at `bf482b0a` — genuinely re-checked this batch in a separate command after catching a self-authored mistake where an earlier check reused the worktree's cwd under a misleading "BASE CHECKOUT" label instead of actually `cd`-ing there.
- All 35/35 tasks marked `[x]` in `tasks.md` (`grep -c "^- \[x\]"` → `35`; `grep -c "^- \[ \]"` → `0`).

## Coverage table status (final — all closed)

| Scenario | Task(s) | Status |
|---|---|---|
| D1 (both paths) | 0.1–0.3 | ✅ closed |
| Both substrate filters | 0.4, 1.9, 2.6, 3.3 | ✅ closed |
| S-RTY-001 | 1.1–1.2 | ✅ closed |
| S-RTY-002 | 1.3–1.4, 1a.1 | ✅ closed |
| S-RTY-003 | 1.5–1.6 | ✅ closed |
| S-RTY-004 | 1.5–1.6 | ✅ closed |
| S-RTY-005 | 2.1–2.2 | ✅ closed |
| S-RTY-006 | 2.1–2.2, 2.7 | ✅ closed |
| S-RTY-007 | 2.1–2.2 | ✅ closed |
| S-RTY-008 | 2.1–2.2 | ✅ closed |
| S-RTY-009 | 2.3–2.4 | ✅ closed |
| S-RTY-010 (bite) | 1.7 | ✅ closed |
| S-RTY-011 (bite) | 0.5 / 1.8 | ✅ closed |
| S-RTY-012 (bite) | 2.5 | ✅ closed |
| S-RTY-013 | 3.1–3.2 | ✅ closed |
| S-RTY-014 | 3.1–3.2 | ✅ closed |
| S-RTY-015 | 4.1–4.2 | ✅ closed |
| Doc 0003 checklist / counter / narrative | 5.1–5.6 | ✅ closed |

## Known risks / flags carried forward to sdd-verify and sdd-archive

- **`agent-loop-skeleton`'s delta has a `## MODIFIED Header` section outside the usual `MODIFIED Requirements` shape (task 5.6, re-flagged explicitly above)** — the archive step must apply this header-range edit to the promoted spec, not only merge the `MODIFIED Requirements` blocks.
- `S-RUN-100`'s "exactly 1 request" pin: re-checked this batch against the shipped tree (task 4.3's grep), still holds.
- AG-16 runs parallel and also edits `harness.go`'s turn-invocation block (the inner attempt loop this change adds) — the merging orchestrator resolves; flagged in proposal risk 11 (unchanged from batch 1).
- Two minor, non-blocking lint findings were discovered and fixed at the very end of this batch (`revive` `package-comments` on the two new production files' headers) — `sdd-verify` should not need to re-discover these, but the fix (a two-line blank-line insertion, commit `a6a3f515`) is the smallest possible diff and worth a quick confirming glance.
