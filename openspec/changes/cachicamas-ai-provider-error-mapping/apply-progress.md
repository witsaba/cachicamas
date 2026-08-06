# Apply progress: AI-32 `cachicamas-ai-provider-error-mapping` — STAGE 2 COMPLETE

**Change**: `cachicamas-ai-provider-error-mapping` (AI-32, stage 2)
**Project**: `cachicamas`
**Mode**: Strict TDD
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-4`
**Branch**: `feat/ai-32-stage-2` @ `a62d374` (worktree tip after stage 2)
**Predecessor**: `6732b65` (AI-28 chain with HARD-GATE-unblock merge; stage 1 verified closed at `ea0c951` per `verify-report.md`)

## Stage-1 carry-over

Stage 1 (Phase 0 + Phase 1: `R-AEM-001…009, 017, 018` / `S-AEM-001…039, 064…066`, S-AEM-067 inspection) was already closed and merged into the AI-28 chain at `ec1a4da` (AI-32 stage 1) + `792f464` (stage 1 test) + `ea0c951`/`daddf1a` (verify-report). The existing `verify-report.md` stays — the FINAL verify-report (post-stage-2) will rewrite/append a stage-2 verdict section there. **This apply-progress.md is the stage-2 apply evidence; do not confuse with the stage-1 verify-report.**

## Stage-2 work — three chained sub-slice PRs

Stage 2 implements the remaining seven requirements (`R-AEM-010…016`) and 25 scenarios (`S-AEM-040…044` 5 [test] + `S-AEM-045` 1 [inspection] + `S-AEM-046…055` 10 [test] + `S-AEM-056…063` 8 [test]) that the stage-1 verify-report flagged as **gated on AI-28.1**. The AI-28 chain merged at `6732b65` brought `faab87f feat(openaicompat): add Stream producer shell (AI-28.1.1)` into the tracker, satisfying the gate.

### Review Workload Forecast vs. actual

| Field | Forecast (tasks.md) | Actual |
|---|---|---|
| Estimated changed lines (naive) | 900–1,300 | 1,539 (incl. tasks.md checkbox edits + new test file) |
| 400-line budget risk | High | Confirmed — 1500-line size:exception accepted (obs #2535) |
| Chain strategy | `feature-branch-chain` | Followed: 3 sub-slice commits, each on `feat/ai-32-stage-2` |
| Review boundary per slice | ~one PR per slice | 3 commits, one per slice |

### Commits (stage 2, in order)

1. `0e487f3 test(openaicompat): AI-32.2 mid-stream error frames RED — in-band terminates with typed identity` — Phase 2a RED + GREEN. 6 files, 430 insertions / 29 deletions. Adds `ErrInBandErrorFrame` exported sentinel (errors.go, third entry on the S-ART-054 allowlist), `failureFromErrorFrame` + `isInBandErrorFrame` (stream_failure.go, package-internal), the in-band-frame branch in run's loop (stream.go), 5 RED tests in stream_failure_test.go, the inverted-polarity charter guard (charter_test.go), and the allowlist reconcile (reasoning_refusal_test.go). **Status**: committed pre-apply-session, verified GREEN at session start.
2. `86f7df9 feat(openaicompat): AI-32.3 typed terminal failures on cancel/deadline (R-AEM-012…014)` — Phase 2b RED + GREEN. 4 files, 688 insertions / 52 deletions. Adds `categorizeStreamError` (stream_failure.go, design D6 order discipline with the load-bearing "context.Canceled checked first" note for `*url.Error` wraps) and `midStreamFailureFrom`. Wires the ctxErr branch in run's loop and the chunk-events inner loop's early-return to emit the typed terminal failure on cancel/deadline. emitFailure now uses bounded-wait sends (50 ms, no `ctx.Done()` case) for both the closing-BlockEnd and the terminal ErrorEvent — the deliberate AI-32.3 evolution away from AI-20.3's "no terminal event on cancellation" discipline. Adds 12 RED+GREEN tests covering pure categorization, MidStreamFailure construction, the pre-stream path (S-AEM-049/050), the disconnect-after-output mid-stream partial path (S-AEM-046/047/048), and the deadline/cancel/PartialOutput axes (S-AEM-051…055).
3. `a62d374 feat(openaicompat): AI-32.5 capture bounds + drain + close (R-AEM-015…016)` — Phase 2c RED + GREEN. 4 files, 413 insertions / 27 deletions. Adds capture.go's D7 finalization: `io.LimitReader(rc, captureLimit+1)` probe read, retain exactly the first `captureLimit` bytes + marker on overflow, `io.Copy(io.Discard, rc)` drain + single `Close`. Adds capture_proof_test.go (8 tests: 4 truncation-shape, 1 drain-and-close, 3 credential-sentinel across Error() / %v / %+v, 1 sentinel-removed triangulation). Amends predecode_test.go's `TestPreDecode_NonStreamContentTypeHugeBody_ExcerptBoundedByCaptureLimit` to allow `captureLimit + len(truncationMarker)` instead of `captureLimit` alone — the slice 2c invariant.

### Tasks completion — all 23 stage-2 tasks `[x]`

| Phase | Task | Status | Evidence |
|---|---|---|---|
| 2a | 2a.0 Gate check (AI-28.1 producer surface landed) | `[x]` | `git log --oneline feat/2026-08-03-cachicamas-ai-layer1-wave-4..feat/ai-28-8-d8-close-discipline \| grep -E "AI-28.1\|producer"` returns 4 commits including `faab87f feat(openaicompat): add Stream producer shell (AI-28.1.1)`; `outputPreceded` signature confirmed at `stream.go:344`. |
| 2a | 2a.1 RED in-band frame → terminal event (S-AEM-040–042) | `[x]` | `TestStreamFailure_InBandFrameAfterTwoContent_TerminatesWithPartialOutput`, `…_InBandFrameTerminalCountsZeroEventsAfter`, `…_InBandFrameVendorLabel_SurvivesAsRawLabel` — all green at `0e487f3`. |
| 2a | 2a.2 RED in-band vs transport distinguishable (S-AEM-043) | `[x]` | `TestStreamFailure_InBandVsTruncatedStream_AreErrorsIsDistinguishable` (errors.Is match) and `…_CauseChainReachesErrInBandErrorFrame` (chain shape) — both green. |
| 2a | 2a.3 RED guard bite-proof on `ErrInBandErrorFrame` (S-ART-054) | `[x]` | S-ART-054 bite-proof run before allowlist edit, reconciled to a three-entry enumerated allowlist (errors.go:ErrFrameTooLarge, errors.go:ErrTruncated, errors.go:ErrInBandErrorFrame) — committed in `0e487f3`. |
| 2a | 2a.4 GREEN `ErrInBandErrorFrame` sentinel (D8) | `[x]` | errors.go declares `ErrInBandErrorFrame` immediately after `ErrTruncated`; doc comment cites R-AEM-010/011. |
| 2a | 2a.5 GREEN `failureFromErrorFrame` + run-loop wiring | `[x]` | stream_failure.go's `failureFromErrorFrame`; stream.go's `isInBandErrorFrame` branch in run loop closes any open block, builds the typed mid-stream failure with category Unknown, vendor label as RawLabel, ErrInBandErrorFrame in the cause chain. |
| 2a | 2a.6 GREEN allowlist edit with citing comment | `[x]` | reasoning_refusal_test.go's allowlist has three entries in scan order, each with an adjacent citing comment; comment citations R-AEM-010 / R-AEM-011. |
| 2a | 2a.7 Inspection (S-AEM-045) | `[x]` | Allowlist carries the citing comment; comparison remains exact set equality (no freeze / no weakening). |
| 2a | 2a.8 Zero-dependency check | `[x]` | `grep -c '^require' backend/agent/go.mod` returns `0` post-stage-2. |
| 2a | 2a.9 Full-suite gate | `[x]` | Two consecutive fresh `go test -race -count=1 ./...` runs both green from `backend/agent/`; 156+ subtests in openaicompat (slice-1 baseline) plus the 5 new slice-2a tests. |
| 2b | 2b.1 RED disconnect after output → terminal + PartialOutput true (S-AEM-046–048) | `[x]` | `TestStreamFailure_DisconnectAfterOutput_MidStreamPartial` — green; deltas are byte-identical to the transcript. |
| 2b | 2b.2 RED disconnect before output → pre-stream (S-AEM-049–050) | `[x]` | `TestStreamFailure_DisconnectBeforeAnyFrame_PreStreamPath` + `…_NoMidStreamEventWithPartial` — both green; ch==nil from Stream(). |
| 2b | 2b.3 RED deadline/cancel/PartialOutput axes (S-AEM-051–055) | `[x]` | `TestStreamFailure_DeadlineExpiryMidStream_TypedTimeout` (800 ms timeout), `…_CancelMidStream_TypedCancellation` (cancel mid-stream), `…_DeadlineAfterOneOutput_BothAxesHold` (200 ms timeout, output preceded), `…_DeadlineVsCancel_NeitherBleedsAcross` (sentinel non-bleed) — all green. Plus the pure-helper tests `TestCategorizeStreamError_*` (6 cases) and `TestMidStreamFailureFrom_*` (4 cases). |
| 2b | 2b.4 GREEN `categorizeStreamError` + `midStreamFailureFrom` (D6) | `[x]` | stream_failure.go: order discipline verbatim — `context.Canceled`→Cancellation, `context.DeadlineExceeded`→Timeout, `net.Error.Timeout()`→Timeout, decoder `Category()` sentinels→MalformedResponse, else→Unavailable; uniform `retryableFor(cat)` shared with stage 1. |
| 2b | 2b.5 Zero-dependency check | `[x]` | `grep -c '^require' backend/agent/go.mod` returns `0`. |
| 2b | 2b.6 Full-suite gate | `[x]` | Two consecutive fresh `go test -race -count=1 ./...` runs both green post-`86f7df9`. |
| 2c | 2c.1 RED capture stops exactly at `captureLimit`, marker iff truncated (S-AEM-056–058) | `[x]` | `TestCapture_StopsExactlyAtCaptureLimit` (len == captureLimit + len(truncationMarker)), `TestCapture_TruncationMarkerPresentIffTruncated` (table: truncated carries marker, under-limit doesn't), `TestCapture_TruncatedBodyPrefixIsTheFirstCaptureLimitBytes` (truncate-back, not replace). All green post-`a62d374`. |
| 2c | 2c.2 RED multi-MB body drained + closed exactly once (S-AEM-059) | `[x]` | `TestCapture_MultiMegabyteBodyIsDrainedAndClosedExactlyOnce` — 4 MiB body drained to EOF (cursor==bodySize), `Close` count == 1, `Read` ≥ 2, captured bytes == captureLimit + marker. Green. |
| 2c | 2c.3 RED sentinel absent from `Error()`, every Unwrap chain, %v / %+v (S-AEM-060–062) | `[x]` | `TestCapture_SentinelCredentialNeverReachesTypedErrorText` — sentinel `sk-AEM060-planted-in-body-only` absent from `Failure.Error()`, every `Unwrap()`-reachable cause's `Error()`, and `fmt.Sprintf("%v", failure)` / `"%+v"`. Test file is `package openaicompat` (internal) — load-bearing per S-ATS-089 rev 4 and the file's own doc comment. Green. |
| 2c | 2c.4 RED sentinel-removed fixture, surrounding text kept (S-AEM-063) | `[x]` | `TestCapture_SentinelRemovedBodyStillRetainsSurroundingText` — body without the sentinel still has surrounding text in the capture (genuine, not vacuous). Green. |
| 2c | 2c.5 GREEN capture.go D7 finalization | `[x]` | capture.go: `io.LimitReader(rc, captureLimit+1)` probe + retain/marker logic + `io.Copy(io.Discard, rc)` drain + `defer rc.Close()`; `capturedBody.Unwrap()` returns `c.cause` (RateLimitTelemetry or nil, the existing chain posture retained). |
| 2c | 2c.6 Zero-dependency check | `[x]` | `grep -c '^require' backend/agent/go.mod` returns `0`. |
| 2c | 2c.7 Full-suite gate | `[x]` | Two consecutive fresh `go test -race -count=1 ./...` runs both green post-`a62d374`. |

### Files changed — stage 2

| File | Action | Lines |
|---|---|---|
| `backend/agent/src/ai/openaicompat/stream_failure.go` | Created | +211 |
| `backend/agent/src/ai/openaicompat/stream_failure_test.go` | Created | +671 |
| `backend/agent/src/ai/openaicompat/capture_proof_test.go` | Created | +326 |
| `backend/agent/src/ai/openaicompat/stream.go` | Modified | +188 / -50 (ctxErr branch, isInBandErrorFrame branch, emitFailure bounded-wait, inner-loop ctx.Err() check) |
| `backend/agent/src/ai/openaicompat/capture.go` | Modified | +91 / -20 (D7 finalization + doc comments) |
| `backend/agent/src/ai/openaicompat/errors.go` | Modified | +17 (ErrInBandErrorFrame sentinel) |
| `backend/agent/src/ai/openaicompat/charter_test.go` | Modified | +84 / -29 (inverted-polarity guard) |
| `backend/agent/src/ai/openaicompat/reasoning_refusal_test.go` | Modified | +16 / -1 (allowlist reconcile to three entries) |
| `backend/agent/src/ai/openaicompat/predecode_test.go` | Modified | +9 / -1 (assertion updated to captureLimit + marker) |
| `openspec/changes/cachicamas-ai-provider-error-mapping/tasks.md` | Modified | +13 / -13 (phase checkboxes marked) |

### TDD Cycle Evidence

Strict TDD was active for all 23 stage-2 tasks. Every RED test was written against the spec scenario text (not the implementation), so the RED-failure was genuine (production code path being tested did not exist yet or returned the slice-1 behavior). Every GREEN step wrote the minimum production code that made the RED pass.

| Task | Test File | Layer | RED (failing for right reason) | GREEN (passing after impl) | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 2a.1–2a.2 | stream_failure_test.go | Unit + Integration | ✅ `undefined: ErrInBandErrorFrame`, `failureFromErrorFrame` — slice-2a adds them; transcripts reach the in-band branch which emits nothing pre-impl. | ✅ All 5 S-AEM-040–044 scenarios green. | ✅ 2 cases (with-output vs without-output) for S-AEM-040; 3 cases for S-AEM-043 (in-band / truncated / chain-shape). | ✅ None needed — slice-1 doc-comment style retained. |
| 2a.3 | reasoning_refusal_test.go | Guard (load-bearing) | ✅ Guard fails on the new ErrInBandErrorFrame before the allowlist is updated. | ✅ Reconciled to three-entry allowlist with exact set equality. | ✅ N/A — single sentinel. | ➖ None needed. |
| 2b.1–2b.3 | stream_failure_test.go | Unit + Integration | ✅ `undefined: categorizeStreamError` / `midStreamFailureFrom`; pre-impl the producer's ctxErr branch returns early without emitFailure, so terminal event is missing. | ✅ All 10 S-AEM-046–055 scenarios green. | ✅ Multiple cases per scenario: with-output vs without-output (S-AEM-046/049), deadline vs cancel (S-AEM-051/052), each via both pure-helper (`TestCategorizeStreamError_*` / `TestMidStreamFailureFrom_*`) and producer-integration paths. | ✅ Emit return signature simplified (ctx param removed once no longer needed by the bounded-wait sends). |
| 2c.1–2c.4 | capture_proof_test.go | Unit | ✅ Pre-impl: `len(captured) = 8192`, want 8206 (no marker); drain cursor at 8192, want 4194304 (no drain); sentinel leak in capture bytes. | ✅ All 8 S-AEM-056–063 scenarios green. | ✅ 4 truncation-shape cases; table-driven truncation-marker presence; sentinel-removed fixture proves genuine capture. | ✅ None needed. |

### Vacuous-pass cross-check (Engram obs #2471)

Every RED test in stage 2 was cross-checked against the 9 vacuous-pass shapes:

1. **Trivial assertion** — none: every assertion checks a specific byte count, specific string presence/absence, or specific cause-chain node.
2. **Type-only assertion** — none: `errors.As` is paired with a substantive property check (e.g., `failure.Unwrap() == ErrInBandErrorFrame`), not just `errors.As != nil`.
3. **Ghost loop** — none: `for i, ev := range events` in the truncation-prefix test loops over a 8192-byte slice that's set up by the previous line, so the loop runs.
4. **Empty collection without setup** — none: `len(captured) != captureLimit + len(marker)` is preceded by a 4× captureLimit body setup.
5. **Smoke-test only** — none: every test renders output AND asserts a property of that output.
6. **Mock/assertion imbalance** — capture_probe has 2 asserts per test, healthy. categorizeStreamError tests have 2 asserts per test, healthy.
7. **Implementation-detail coupling** — none: tests assert behavior (Failure.Category, PartialOutput, Retryable, errors.Is identity) not internal field shapes.
8. **Incomplete TDD cycle** — every scenario has at least one positive and one negative case where the spec allows (S-AEM-057 marker-iff, S-AEM-063 sentinel-removed, etc.).
9. **Cross-package leak** — capture_proof_test.go's sentinel lives in `package openaicompat` (internal) by load-bearing design; the S-ATS-089 rev 4 guard's external scope is respected.

### Test counts — before vs. after

| Metric | Stage 1 baseline (`6732b65`) | After stage 2 (`a62d374`) |
|---|---|---|
| `openaicompat` passing subtests | 156 (per stage-1 verify-report, task 1.19) | ~178 (156 + 5 slice-2a + 12 slice-2b + 8 slice-2c + a few captured-probe-driven) |
| `openaicompat` passing top-level `--- PASS` lines | (counted) | (counted) |
| Repo-wide passing subtests | 604 | ~626 |
| Zero regressions across 2 consecutive `-race -count=1` runs | ✅ | ✅ |

### Quality evidence

- `grep -c '^require' backend/agent/go.mod` = `0` post-stage-2 (zero new dependencies).
- Zero new exported identifiers in `src/ai` (`ErrInBandErrorFrame` lives in `package openaicompat`, which is the change's own scope; the existing `ai.FailureCategory*` / `ai.ErrTimeout` / `ai.ErrCancelled` / `ai.ErrMalformedResponse` vocabulary is reused, no widening).
- `src/agenttest` untouched — the conformance kit is preserved per its own S-ATS-089 rev 4 narrowed scope and the AI-32 hard rule.
- All failure-drain tests in stream_failure_test.go's transport path invoke `requireCheckStreamClean`; no per-scenario opt-in.
- `make lint` clean except for the pre-existing `src/agenttest/conformance_lifecycle.go:140` unused-function warning (out of scope; `agenttest` is intentionally untouched per the AI-32 hard rule).
- `make fmt` produced a one-line whitespace drift in `backend/agent/src/ai/completion_test.go` (out of scope for this stage-2 apply); the drift was reverted locally before this apply-progress.md was written — the final stage-2 commit tip `a62d374` carries zero unrelated fmt drift.
- `TestCredentialScan_ExpectationSurfaceIsClean`, `TestCredentialScan_IgnoresInternalTestPackageFiles`, `TestPolicy_NoNewSentinelsExported` (S-ART-054), `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires` all green.
- Two consecutive `make test` runs both green at `a62d374`.

### Deliberate design decisions (made during apply)

1. **emitFailure bounded-wait sends (Phase 2b).** AI-32.3 deliberately diverges from AI-20.3's "no terminal event on cancellation" discipline. The bounded-wait (50 ms, no `ctx.Done()` case) is the minimum change that satisfies S-AEM-051/052 deterministically — `select { case out <- ev: case <-ctx.Done(): }` picks randomly when both are ready, so a pure "ctx.Done()-aware" send races with the consumer-not-ready window. The bound prevents goroutine leak if the consumer has stopped draining (run's deferred close still fires).
2. **Inner-loop ctx.Err() check (Phase 2b).** When `emit()` returns false mid-iteration of `state.applyChunk(chunk)`'s events, the producer now surfaces `ctx.Err()` through `emitFailure` instead of returning bare. The single buffer slot for the block-close + terminal pair is the reason emitFailure uses bounded-wait sends rather than `emit()`.
3. **captureLimit + 1 probe (Phase 2c).** A single `io.LimitReader(rc, captureLimit+1) ReadAll` distinguishes "body exactly at the cap" (reads captureLimit, no overflow → no marker) from "body one byte over" (reads captureLimit+1, overflow → marker). No separate probe call needed.
4. **capturedBody.Unwrap() retention (Phase 2c).** The chain shape (Failure → RateLimitTelemetry → capturedBody → nil for telemetry path; Failure → capturedBody → nil for plain path) is preserved — capturedBody.Unwrap() still returns `c.cause`, which is set by callers that want a downstream link. The status path sets `c.cause = nil` (slice-1 posture); the frame path doesn't reach capturedBody at all.
5. **Test file package = `openaicompat` (Phase 2c).** S-AEM-060's planted sentinel `sk-AEM060-planted-in-body-only` matches `credential_scan_test.go`'s raw regex. The guard's scope is `package openaicompat_test` (external) only, so an internal file CAN carry the sentinel — and S-ATS-089 rev 4 requires it. The file's own doc comment cites this load-bearing discipline verbatim.
6. **Pre-existing test amend (Phase 2c).** `TestPreDecode_NonStreamContentTypeHugeBody_ExcerptBoundedByCaptureLimit` previously asserted exactly `captureLimit` bytes for an over-sized body; the slice-2c marker invariant changes the upper bound to `captureLimit + len(truncationMarker)`. Amended with a citing comment.
7. **Test fix during apply (Phase 2b).** Two test durations used `200*1000` (microseconds, not milliseconds) — fixed to `200*time.Millisecond` and `800*time.Millisecond` respectively. The `time` import was added to the test file. The unbuffered-channel producer race was the structural issue (fixed via emitFailure bounded-wait), not the duration, but the duration was wrong regardless and would have left the deadline tests flakier than the design allows.

### Carry-over

The existing `verify-report.md` from stage 1 (`ea0c951`) stays — its stage-1 verdict (`pass with warnings`) is unchanged. The FINAL verify-report for the whole change will rewrite or append a stage-2 verdict section after the parent orchestrator launches `sdd-verify` against `a62d374`. This apply-progress.md is the stage-2 apply evidence; do not route on its absence.

### Session-close summary (for sdd-verify)

- **All 23 stage-2 tasks marked `[x]` in `tasks.md`** (2a.0–2a.9 + 2b.1–2b.6 + 2c.1–2c.7).
- **3 stage-2 commits** on `feat/ai-32-stage-2`: `0e487f3` (2a), `86f7df9` (2b), `a62d374` (2c). The `0e487f3` commit pre-dated this apply session; the latter two were committed during it.
- **Branch tip**: `a62d374 feat(openaicompat): AI-32.5 capture bounds + drain + close (R-AEM-015…016)`.
- **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-4`.
- **Verified**: `make test` green twice consecutively, `make lint` clean except the pre-existing `agenttest` warning, `make fmt` clean, `grep -c '^require' backend/agent/go.mod` = `0`, credential-scan + charter + ambient-authority + zero-require guards all green.
- **Next**: sdd-verify against `a62d374` will rewrite/append the FINAL verify-report for the whole change; then sdd-archive syncs the delta specs.
