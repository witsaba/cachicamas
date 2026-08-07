# Tasks: Define retry and idempotency policy (AI-35)

> **Change**: `cachicamas-ai-retry-policy` · **Milestone**: AI-35 (doc 0002 lines 2077–2144)
> **Wave**: 5 — Harden · **Module**: `backend/agent/` (layered per ADR 0005 § D1, NOT hexagonal — `cd backend/agent && make test`)
> **Strategy**: single-pr · strict TDD on · stdlib-only · review budget **1000 changed lines** (doc 0002 amendment line 2093)
> **Inputs**: [`proposal.md`](./proposal.md) (Flag 1 + Flag 2 closed), [`specs/ai-stream-lifecycle/spec.md`](./specs/ai-stream-lifecycle/spec.md) (R-AIS-041..044), [`specs/ai-provider-conformance-suite/spec.md`](./specs/ai-provider-conformance-suite/spec.md) (R-CNF-019 + R-CNF-017 modified + CapRetry), [`design.md`](./design.md) (943 lines, 10 decisions).
> **Reference work**: Engram obs `#2641` (decision), `#2642` (explore), `#2643` (proposal), `#2644` (spec), `#2645` (design), `#2640` (AI-34 close + workflow).

---

## § 1 Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-retry-policy` |
| **Milestone** | AI-35 (doc 0002 lines 2077–2144; charter test list at 2113–2144) |
| **Scope (in)** | `internal/retry/` helper package (NEW); `stream.go:218–227` wires `retry.Loop`; 3 NEW real-producer test files; 1 NEW conformance case; `Factory.Retry *bool`; `CapRetry` enum member (9th) |
| **Scope (out)** | Layer 2 harness retry (AG-15.1/15.2); model failover (AG-15.3); mid-stream retry; new top-level Go dependencies; configurability of `defaultMaxAttempts` |
| **Review budget** | **1000 changed lines** (per doc 0002 amendment line 2093) |
| **Test runner** | `cd backend/agent && make test` (`go test -race -v ./...`); `make lint` (`go vet` + `golangci-lint`); `make fmt` |
| **Module** | `backend/agent` — layered (`src/ai` ← `src/agent` ← `src/coding` ← `src/cmd/cachicamas`), per ADR 0005 § D1 |
| **Conformance amended** | `ai-provider-conformance-suite` — `R-CNF-019` added; `R-CNF-017` modified (totality over `[9]` entries); `CapRetry` (CAP-O-04) added |
| **Spec amended (archive)** | `ai-stream-lifecycle` — `R-AIS-041..044` (4 requirements, 14 scenarios) |
| **Forward guard** | `backend/agent/src/ai/import_boundary_test.go:107–162` — stdlib + own-module only; helper is stdlib + `ai` |

---

## § 2 Forecast (precise, per-file)

For each file, the table lists the design's estimate, my refinement (with reasoning), and the source / precedent that calibrates it.

### § 2.1 Production-code files

| File | Status | Design estimate | Refined estimate | Refinement rationale |
| --- | --- | --- | --- | --- |
| `backend/agent/src/ai/internal/retry/retry.go` | NEW | ~140 | **~110** | Trim redundant inline comments (Loop body walks in § 6 of design — only the helper signatures + algorithm skeleton + private helpers are needed). |
| `backend/agent/src/ai/internal/retry/doc.go` | NEW | ~25 | **~25** | Required: cross-layer citation of `defaultMaxAttempts = 3` + composed-bound formula. |
| `backend/agent/src/ai/internal/retry/retry_test.go` | NEW (RED-first per strict TDD) | (not in design) | **~60** | Unit tests for `computeBackoff` (pure math), `AttemptReport.Error()`/`Unwrap()` (error chain), `applyDefaults` (zero-value substitution). `internal` package test (same `package retry`). Pattern: `capture_proof_test.go` style. |
| `backend/agent/src/ai/openaicompat/stream.go` | MODIFIED | +8 / −2 | **+8 / −2** (10 net) | One import added (`internal/retry`); `Do + mapResponse` segment replaced with `retry.Loop` call. The `c.executeOnce` closure is a new method on `*Client` (~14 new lines). Total file change: +22 / −2 in absolute terms; net +20 attributable to AI-35. |
| `backend/agent/src/ai/openaicompat/execute_once.go` (new method file) | NEW | (rolled into stream.go) | **~14** | `executeOnce` closure defined as a `*Client` method (separated per Go file-per-method convention used elsewhere in `openaicompat/`); package-private. |
| `backend/agent/src/agenttest/conformance_suite.go` | MODIFIED | +12 / −0 | **+12 / −0** (12 net) | `CapRetry` enum member (1), `capabilityNames` map entry (1), `Optional()` switch case (1), `Factory.Retry *bool` field (1), `factoryDefect` nil-check (3 lines), `declaredOffered` switch case (3 lines), doc comment update for `Factory.Retry` (2 lines). |
| `backend/agent/src/agenttest/conformance_record.go` | MODIFIED | +1 / −1 | **+1 / −1** (2 net) | `entries [8]CapabilityRecordEntry` → `[9]CapabilityRecordEntry` (one array-length change). |
| `backend/agent/src/agenttest/doc.go` | MODIFIED | (not in design) | **+3 / −0** (3 net) | Note the new `openaicompat` dependency that `conformance_retry.go` introduces (per design D-R4). |

### § 2.2 Real-producer test files (NEW)

| File | Design estimate | Refined estimate | Refinement rationale |
| --- | --- | --- | --- |
| `backend/agent/src/ai/openaicompat/a_i-35_1_test.go` | ~180 | **~180** | 3 RED tests (R-AIS-041 / S-1, S-2, S-4). Pattern: `a_i-33_1_test.go` (226 lines, 3 scenarios). Scenario banners + assertion bodies. |
| `backend/agent/src/ai/openaicompat/a_i-35_2_test.go` | ~200 | **~220** | 4 RED tests (R-AIS-042 / S-1..S-4). Timing-injection via `Config{SleepFunc, JitterSeed, ...}` adds ~20 lines per scenario. Pattern: `a_i-33_5a_test.go` (423 lines, 4 scenarios). |
| `backend/agent/src/ai/openaicompat/a_i-35_3_test.go` | ~220 | **~240** | 4 RED tests (R-AIS-043 / S-1..S-4; S-4 = partial-output boundary marker per Flag 2). Shared scriptable-transport helper inlined (~40 lines). Pattern: `a_i-34_2_test.go` (338 lines). |
| `backend/agent/src/ai/openaicompat/a_i-35_internal_test.go` (optional shared helper) | ~80 | **0 (skip)** | Merge the shared helper into `a_i-35_3_test.go` (the file that uses it most — `S-1` byte-recording, `S-3` drain). Saves the separate file's overhead. |

### § 2.3 Conformance case file (NEW)

| File | Design estimate | Refined estimate | Refinement rationale |
| --- | --- | --- | --- |
| `backend/agent/src/agenttest/conformance_retry.go` | ~250 | **~220** | 7 scenarios (S-CNF-069..075). 5 scripted (httptest.NewServer + *openaicompat.Client directly — not the factory pattern); 2 suite-mechanism (S-CNF-074/075 share the existing factoryDefect mechanism, ~10 lines each). Share one `scriptableTransport` helper across the 5 scripted scenarios (~30-line savings). |

### § 2.4 Aggregate forecast

| Category | Files | Net changed lines |
| --- | --- | --- |
| Production NEW | `retry.go` + `doc.go` + `retry_test.go` + `execute_once.go` | 110 + 25 + 60 + 14 = **209** |
| Production MODIFIED | `stream.go` + `conformance_suite.go` + `conformance_record.go` + `doc.go` (agenttest) | 10 + 12 + 2 + 3 = **27** |
| Test NEW | `a_i-35_1_test.go` + `a_i-35_2_test.go` + `a_i-35_3_test.go` | 180 + 220 + 240 = **640** |
| Conformance NEW | `conformance_retry.go` | **220** |
| **Total** | 10 files | **~1096 lines** (1096/1000 = +9.6% over budget) |

**Comparison to 1000-line budget**: **96 lines over** (~10%). The design's original forecast was ~1100; this refinement shaves ~4 lines. The overrun is small and concentrated in the four test files (640 of 1096, ~58%), which are exactly the files whose size reflects the rich scenario-banner documentation density that the AI-33/AI-34 precedent (and AGENTS.md's strict-TDD discipline) requires.

### § 2.5 Trim alternative (NOT recommended — see § 6)

If the user rejects `size:exception` at the apply checkpoint, the trim path is:

| Trim | Saving | Cost |
| --- | --- | --- |
| Merge `a_i-35_internal_test.go` into `a_i-35_3_test.go` | ~60 lines | Already done (§ 2.2). |
| Compact `conformance_retry.go` (single transport helper) | ~30 lines | Already done (§ 2.3). |
| Tighten `retry.go` doc comments to function-level only | ~30 lines | Loses the algorithm narration that documents the seam for Layer 2 readers. |
| Reduce scenario-banner verbosity in `a_i-35_*_test.go` | ~80 lines | Breaks the AI-33 / AI-34 precedent for review-friendly banner density; tests stay correct but lose self-documentation. |
| **Total trim** | ~200 lines | Brings aggregate to ~896, well under 1000. |

---

## § 3 Work units (commit-shaped)

Five work units. Each is a single conventional-commit PR-ready slice. Per `work-unit-commits` skill rule "Commit by work unit", each WU ships one behaviour delta + its RED tests + doc updates.

### WU-1 — Helper package foundation

- **Subject**: `feat(agent): add internal/retry helper package (Loop, Config, AttemptReport, DefaultMaxAttempts)`
- **Files added**:
  - `backend/agent/src/ai/internal/retry/retry.go` (~110 lines)
  - `backend/agent/src/ai/internal/retry/doc.go` (~25 lines)
  - `backend/agent/src/ai/internal/retry/retry_test.go` (~60 lines)
- **Files modified**: none
- **Strict TDD sequence (RED → GREEN → REFACTOR)**:
  - **RED**: write `internal/retry/retry_test.go` with three unit tests: `TestAttemptReport_ErrorAndUnwrap`, `TestApplyDefaults_SubstitutesZeroValues`, `TestComputeBackoff_BoundedExponentialWithSeededJitter`. All fail because `retry` package doesn't exist yet.
  - **GREEN**: lift the design's `Loop`, `Config`, `AttemptReport`, `DefaultMaxAttempts`, `applyDefaults`, `computeBackoff`, `wrapTransportError`, `defaultSleep`, `defaultRetryAfterReader` from `design.md` § 5 and § 6 verbatim. Tests pass.
  - **REFACTOR**: extract magic numbers (`baseDelayDefault = 100ms`, `maxDelayDefault = 30s`, `defaultJitterFraction = 10`) as package-private constants. `make test` green.
- **Test list**: 3 internal unit tests (AttemptReport error-chain, applyDefaults zero-value, computeBackoff bounded).
- **Dependencies**: none.
- **Focused test command**: `cd backend/agent && go test -race -v ./src/ai/internal/retry/...`
- **Runtime harness**: N/A (pure unit tests, no HTTP).
- **Rollback boundary**: `git rm -r backend/agent/src/ai/internal/retry/` removes the helper entirely; no other code references it yet.
- **Commit message**:
  ```
  feat(agent): add internal/retry helper package (Loop, Config, AttemptReport)
  
  Layer 1 pre-stream retry helper. Loop wraps an execute-once closure
  (per-attempt HTTP round-trip + wire-side status mapping) and retries
  only on retryable-flagged pre-stream failures up to Config.MaxAttempts.
  AttemptReport is the typed cause attached to the last wire failure on
  exhaustion; FinalCause is reachable via errors.As.
  
  No caller yet — WU-2 wires it into openaicompat.Stream. Stdlib-only
  imports; forward guard stays green.
  ```

### WU-2 — Stream integration + AI-35.1 predicate (RED/GREEN/REFACTOR)

- **Subject**: `feat(agent): wire retry.Loop into openaicompat Stream + AI-35.1 predicate tests`
- **Files added**:
  - `backend/agent/src/ai/openaicompat/execute_once.go` (~14 lines)
  - `backend/agent/src/ai/openaicompat/a_i-35_1_test.go` (~180 lines)
- **Files modified**:
  - `backend/agent/src/ai/openaicompat/stream.go` (+22 / −2 = 20 net)
- **Strict TDD sequence (RED → GREEN → REFACTOR)**:
  - **RED**: write `a_i-35_1_test.go` with three failing tests:
    - `TestAI35_1_RetryablePreStream_RetriesUpToBound` — scripted 429 then 2xx; expect `Do` count == `DefaultMaxAttempts + 1`.
    - `TestAI35_1_TerminalCategory_NeverRetries` — scripted 401; expect `Do` count == 1.
    - `TestAI35_1_ExhaustedBudget_ReturnsLastFailureWrappedWithAttemptCount` — scripted 429 always; expect `errors.As` over the returned error yields `*retry.AttemptReport{Attempts == N+1}` AND `*ai.Failure{Category == RateLimit}`.
    - All three fail because `stream.go` does not call `retry.Loop` yet.
  - **GREEN**: add `retry.Loop` import to `stream.go`; replace the `Do + mapResponse` segment (lines 218–227) with `retry.Loop(ctx, body, retry.Config{}, c.executeOnce)`; author `c.executeOnce` as a `*Client` method (separated file). Tests pass.
  - **REFACTOR**: extract `preStreamTransportFailure` call site to share with the helper's `wrapTransportError` shape (they should match; if they don't, document why). `make test` green.
- **Test list**: 3 RED tests above (R-AIS-041 / S-1, S-2, S-4).
- **Dependencies**: WU-1.
- **Focused test command**: `cd backend/agent && go test -race -v -run 'TestAI35_1' ./src/ai/openaicompat/...`
- **Runtime harness**: N/A (test files drive the scriptable transport).
- **Rollback boundary**: `git revert` this commit removes `retry.Loop` from `stream.go`; `stream.go` reverts to the pre-AI-35 218–227 segment; the test files drop with the code change.
- **Commit message**:
  ```
  feat(agent): wire retry.Loop into openaicompat Stream + AI-35.1 tests
  
  stream.go:218–227 (the Do + mapResponse segment) now wraps the
  per-attempt operation in retry.Loop. c.executeOnce is the per-attempt
  closure; byte-identical replay is structural because newRequest wraps
  bytes.NewReader per call (request.go:28). PartialOutput() == false is
  unconditional for every failure the helper observes (carrier handover
  at line 237 is after the helper returns).
  
  AI-35.1 predicate scenarios (R-AIS-041 / S-1, S-2, S-4): retryable
  pre-stream retried up to bound; terminal category (401) never retried;
  exhausted budget returns last failure wrapped with AttemptReport.
  ```

### WU-3 — Backoff mechanics (RED/GREEN/REFACTOR)

- **Subject**: `feat(agent): backoff mechanics — retry-after precedence, ctx-aware waits, seeded jitter, bounded count`
- **Files added**:
  - `backend/agent/src/ai/openaicompat/a_i-35_2_test.go` (~220 lines)
- **Files modified**: none (Retry-After + sleep math live in `internal/retry/retry.go`, already shipped by WU-1).
- **Strict TDD sequence (RED → GREEN → REFACTOR)**:
  - **RED**: write `a_i-35_2_test.go` with four failing tests:
    - `TestAI35_2_RetryAfterHint_OverridesComputedBackoff` — scripted `429 Retry-After: H`; expect backoff between attempts == H exactly (not computed exponential).
    - `TestAI35_2_ComputedBackoff_BoundedWithSeededJitter` — scripted `429` (no Retry-After); `Config{JitterSeed: <fixed>}`; expect each delay in `[base * 2^(attempt-1), base * 2^attempt]` and assertable jitter sequence.
    - `TestAI35_2_Backoff_CtxCancellationAbortsImmediately` — scripted `429 Retry-After: 60`; ctx cancelled mid-backoff; expect helper aborts, no subsequent wire request, returned error is LAST typed failure (not ctx.Err()).
    - `TestAI35_2_BoundedAttemptCount_ExactlyNPlusOneWireRequests` — scripted 429 always; expect `Do` count == `DefaultMaxAttempts + 1` exactly.
    - All four fail because no timing-injection tests exist yet (but the helper already has `SleepFunc` and `JitterSeed` seams from WU-1).
  - **GREEN**: pass `Config{SleepFunc: spySleep, NowFunc: spyNow, JitterSeed: fixed}` from each test; tests drive the helper's already-implemented backoff math.
  - **REFACTOR**: if `computeBackoff`'s jitter math is awkward, extract to a separate `jittered(base time.Duration, j *rand.Rand) time.Duration` helper. `make test` green.
- **Test list**: 4 RED tests above (R-AIS-042 / S-1, S-2, S-3, S-4).
- **Dependencies**: WU-1, WU-2.
- **Focused test command**: `cd backend/agent && go test -race -v -run 'TestAI35_2' ./src/ai/openaicompat/...`
- **Runtime harness**: N/A (timing injection, no wall-clock).
- **Rollback boundary**: `git revert` removes `a_i-35_2_test.go` only; `retry.go`'s backoff math stays (it's an internal helper, exercised by WU-2's exhaustion test). No production code reverts.
- **Commit message**:
  ```
  feat(agent): backoff mechanics — retry-after, ctx-aware waits, jitter, bounded count
  
  AI-35.2 scenarios (R-AIS-042 / S-1..S-4): Retry-After hint overrides
  computed backoff; computed backoff bounded exponential with seeded
  assertable jitter; context cancellation aborts the wait immediately
  and returns the LAST typed failure (not ctx.Err()); exactly N+1 wire
  requests per logical call.
  
  Timing injected via Config.SleepFunc + Config.NowFunc + Config.JitterSeed.
  No wall-clock sleeps in tests.
  ```

### WU-4 — Replayability + partial-output boundary marker (RED/GREEN/REFACTOR)

- **Subject**: `feat(agent): replayability + partial-output boundary marker`
- **Files added**:
  - `backend/agent/src/ai/openaicompat/a_i-35_3_test.go` (~240 lines; includes inlined shared helper for scriptable transport + body recording)
- **Files modified**: none (the production code for replay + drain is structural — already in `Translate`/`newRequest`/`captureBody` per WU-1's helper design).
- **Strict TDD sequence (RED → GREEN → REFACTOR)**:
  - **RED**: write `a_i-35_3_test.go` with four failing tests:
    - `TestAI35_3_ByteIdenticalReplay_AcrossAttempts` — scripted 429 then 200; transport records every body; expect every recorded body byte-identical via `bytes.Equal`.
    - `TestAI35_3_AttemptCountAndFinalCause_ReachableFromErrorChain` — scripted 429 always; expect `errors.As(err, &report)` finds `*retry.AttemptReport{Attempts == N+1}` AND `errors.As(err, &failure)` finds `*ai.Failure{Category == RateLimit, Delivery == DeliveryPreStream}`.
    - `TestAI35_3_PerAttemptDrain_StatusPath` — scripted 429 with trailing garbage; expect `Do` count == N+1, connection-reuse count == 1 (supplied by composition with `captureBody`).
    - `TestAI35_3_PartialOutputBoundaryMarker_AfterHandoverNoRetry` (the Flag 2 test moved from AI-35.1) — scripted `200 text/event-stream` with one text delta then a retryable mid-stream failure; expect `Do` count == 1 (helper seam past); terminal error event has `PartialOutput() == true`.
    - All four fail because the test infrastructure doesn't exist yet.
  - **GREEN**: tests pass by composition — replay is structural (fresh `bytes.NewReader` per attempt), drain is composition with `captureBody`, boundary marker is the seam by construction. No production code changes.
  - **REFACTOR**: extract `ai35ScriptableTransport` + `ai35Counters` as inlined package-private helpers within the test file (NOT a separate shared file — saves ~60 lines per § 2.2).
- **Test list**: 4 RED tests above (R-AIS-043 / S-1, S-2, S-3, S-4).
- **Dependencies**: WU-1, WU-2.
- **Focused test command**: `cd backend/agent && go test -race -v -run 'TestAI35_3' ./src/ai/openaicompat/...`
- **Runtime harness**: N/A (scripted httptest.NewServer).
- **Rollback boundary**: `git revert` removes `a_i-35_3_test.go` only; the production code (already shipped in WU-1+2) is unaffected. No production code reverts.
- **Commit message**:
  ```
  feat(agent): replayability + partial-output boundary marker
  
  AI-35.3 scenarios (R-AIS-043 / S-1..S-4): byte-identical replay across
  attempts (verified via recorded body bytes); attempt count + final
  cause reachable from the error chain via errors.As (AttemptReport +
  *ai.Failure); per-attempt drain supplied by composition with
  captureBody (no new defer in the helper); partial-output boundary
  marker — after a semantic event has been emitted, no automatic retry
  (the helper's seam is past; Do count == 1 after a mid-stream failure).
  
  The boundary marker test was moved from AI-35.1 per proposal Flag 2.
  ```

### WU-5 — Conformance capability + case body (RED/GREEN/REFACTOR)

- **Subject**: `feat(agent): CapRetry conformance capability + retry/auto_retry_up_to_documented_bound case body`
- **Files added**:
  - `backend/agent/src/agenttest/conformance_retry.go` (~220 lines)
- **Files modified**:
  - `backend/agent/src/agenttest/conformance_suite.go` (+12 / −0 = 12 net)
  - `backend/agent/src/agenttest/conformance_record.go` (+1 / −1 = 2 net)
  - `backend/agent/src/agenttest/doc.go` (+3 / −0 = 3 net)
- **Strict TDD sequence (RED → GREEN → REFACTOR)**:
  - **RED**: write `conformance_retry.go` with seven sub-tests:
    - `retryable_pre_stream_retried_up_to_bound` (S-CNF-069) — scripted `429 Retry-After: 0` × N+1 then `200`; expect `Do` count == N+1; final stream is the 2xx.
    - `terminal_category_never_retried` (S-CNF-070) — scripted 401; expect `Do` count == 1; `Retryable() == false`.
    - `partial_output_boundary_no_retry` (S-CNF-071) — scripted 200 with one text delta then mid-stream 429; expect `Do` count == 1; terminal event has `PartialOutput() == true`.
    - `byte_identical_replay` (S-CNF-072) — recorded bodies; expect every body byte-identical.
    - `attempt_count_and_final_cause_in_chain` (S-CNF-073) — exhaustion path; expect `errors.As` finds both `*retry.AttemptReport` and `*ai.Failure`.
    - `cap_retry_absent_reported_not_silent` (S-CNF-074) — `Factory.Retry = false`; expect record entry `absent`.
    - `factory_nil_retry_defect` (S-CNF-075) — `Factory.Retry = nil`; expect construction fails.
    - All seven fail because `CapRetry` doesn't exist; `Factory.Retry` field doesn't exist.
  - **GREEN**: edit `conformance_suite.go` to add `CapRetry` enum member, `capabilityNames` map entry, `Optional()` case, `Factory.Retry *bool` field, `factoryDefect` nil-check, `declaredOffered` case. Edit `conformance_record.go` to grow `entries [8]` → `[9]`. Edit `agenttest/doc.go` to note the new `openaicompat` dependency. All seven tests pass.
  - **REFACTOR**: share a single `scriptableTransport(t, responses)` helper across the 5 scripted scenarios (saves ~30 lines per § 2.3).
- **Test list**: 7 conformance sub-tests above (S-CNF-069..075).
- **Dependencies**: WU-1, WU-2 (helper + Stream wiring must exist for the case body to construct a real `*openaicompat.Client`).
- **Focused test command**: `cd backend/agent && go test -race -v -run 'TestConformance/retry' ./src/agenttest/...`
- **Runtime harness**: N/A (scripted `httptest.NewServer`).
- **Rollback boundary**: `git revert` removes `conformance_retry.go` + reverts the four `conformance_suite.go` / `conformance_record.go` / `agenttest/doc.go` edits; the helper + Stream wiring stays. The suite reverts to `[8]` capabilities.
- **Commit message**:
  ```
  feat(agent): CapRetry conformance capability + retry/auto_retry_up_to_documented_bound case body
  
  Capability enum grows from [8] to [9]; R-CNF-017 totality rebuild is
  mechanical (one new line). Factory.Retry *bool mirrors Reasoning /
  TokenCounting / CacheBoundary (nil-fails-construction per R-CNF-002 /
  S-CNF-006). Conformance case body registers under CapRetry and asserts
  S-CNF-069..075 against a scripted httptest.NewServer + real
  *openaicompat.Client.
  
  The OpenRouter wrapper inherits the helper transparently (its
  *openaicompat.Client embed delegates Stream verbatim), so AI-38's
  conformance roll-up extends the same case body without rewriting
  assertions.
  ```

### § 3.1 Work-unit summary table

| Unit | Subject | Files (added / modified) | Net lines | Likely PR | Focused test command | Runtime harness | Rollback boundary |
| --- | --- | --- | --- | --- | --- | --- | --- |
| WU-1 | Helper package foundation | +3 / −0 | ~195 | PR-1 (part) | `go test -race -v ./src/ai/internal/retry/...` | N/A | `git rm -r src/ai/internal/retry/` |
| WU-2 | Stream integration + AI-35.1 | +2 / +1 | ~214 | PR-1 (part) | `go test -race -v -run 'TestAI35_1' ./src/ai/openaicompat/...` | N/A | revert `stream.go:218–227` + drop test files |
| WU-3 | Backoff mechanics | +1 / −0 | ~220 | PR-1 (part) | `go test -race -v -run 'TestAI35_2' ./src/ai/openaicompat/...` | N/A | drop `a_i-35_2_test.go` only |
| WU-4 | Replayability + boundary | +1 / −0 | ~240 | PR-1 (part) | `go test -race -v -run 'TestAI35_3' ./src/ai/openaicompat/...` | N/A | drop `a_i-35_3_test.go` only |
| WU-5 | Conformance CapRetry + case | +1 / +3 | ~237 | PR-1 (part) | `go test -race -v -run 'TestConformance/retry' ./src/agenttest/...` | N/A | revert conformance_suite.go + drop case file |
| **Total** | single-PR | +8 / +4 | **~1106** | **1 PR** | — | — | — |

---

## § 4 Strict TDD ordering per work unit

Per `openspec/AGENTS.md` rule 3 (strict TDD). Each WU's RED → GREEN → REFACTOR sequence is documented inline in § 3. Consolidated summary:

| WU | RED test | GREEN (minimum production) | REFACTOR |
| --- | --- | --- | --- |
| WU-1 | `internal/retry/retry_test.go` — 3 unit tests (`AttemptReport`, `applyDefaults`, `computeBackoff`) | Lift design § 5 + § 6 verbatim into `retry.go` + `doc.go` | Extract magic numbers as constants |
| WU-2 | `a_i-35_1_test.go` — 3 scenario tests (R-AIS-041 / S-1, S-2, S-4) | `stream.go:218–227` calls `retry.Loop`; `c.executeOnce` method | Inline `preStreamTransportFailure` consistently |
| WU-3 | `a_i-35_2_test.go` — 4 scenario tests (R-AIS-042 / S-1..S-4) | Pass `Config{SleepFunc, JitterSeed, ...}` from tests (helper already has seams) | Extract `jittered()` from `computeBackoff` |
| WU-4 | `a_i-35_3_test.go` — 4 scenario tests (R-AIS-043 / S-1..S-4) | Tests pass by composition (no production change) | Extract `ai35ScriptableTransport` + `ai35Counters` as inlined helpers |
| WU-5 | `conformance_retry.go` — 7 scenario sub-tests (S-CNF-069..075) | Add `CapRetry` + `Factory.Retry` + case body; grow `entries [8]` → `[9]` | Share `scriptableTransport` across scenarios |

**Invariant**: every WU's RED test compiles + runs + fails for the right reason BEFORE the production code lands. No WU skips straight to implementation.

---

## § 5 Risk per work unit

| WU | Risk class | Description | Mitigation |
| --- | --- | --- | --- |
| WU-1 | SUGGESTION | `math/rand/v2` requires Go 1.22+; project is Go 1.26.3 — fine, but downgrade-blocked | Forward guard catches at `go build` time |
| WU-1 | WARNING | Helper file is the only new production code beyond Stream-line edits; reviewer needs to understand the seam before reading Stream integration | Helper's `doc.go` opens with the seam description verbatim from design § 2 |
| WU-2 | CRITICAL | Stream.go modification is the integration point; a wrong edit could break all existing AI-33/AI-34 tests | RED tests in `a_i-35_1_test.go` ARE the regression guard; existing AI-33 / AI-34 tests must continue to pass (`cd backend/agent && make test`) |
| WU-2 | SUGGESTION | `c.executeOnce` method name is awkward; apply phase may pick a different name | Closure is package-private; renaming has no blast radius |
| WU-3 | WARNING | Timing injection via `Config{SleepFunc}` must NOT introduce wall-clock sleeps (per AGENTS.md) | Tests inject a `spySleep` that records calls and returns immediately; no `time.Sleep` |
| WU-4 | CRITICAL | Partial-output boundary marker test (S-4) is structurally subtle — a mid-stream retryable failure must NOT trigger retry | Test scripts a `200 text/event-stream` with one text delta THEN a 429 mid-stream; asserts `Do` count == 1 AND terminal event has `PartialOutput() == true` |
| WU-4 | SUGGESTION | Shared helper inlining may make `a_i-35_3_test.go` slightly longer than ideal | Test file is internal `package openaicompat`; reviewers see it as one cohesive test set |
| WU-5 | WARNING | Conformance case body uses `httptest.NewServer` + `*openaicompat.Client` directly, adding `openaicompat` as a dependency of `agenttest` (breaks adapter-agnostic posture of other conformance cases per design D-R4) | `agenttest/doc.go` is updated to note the new dependency (3 lines); forward guard stays green |
| WU-5 | SUGGESTION | `Factory.Retry` field name choice is the design's; spec deferred to design phase (per `specs/ai-provider-conformance-suite/spec.md:28`) | Name mirrors `Reasoning` / `TokenCounting` / `CacheBoundary` precedent |
| **Cross-WU** | WARNING | Aggregate ~1106 lines (~106 over the 1000-line budget); see § 6 | Recommended: single-PR with `size:exception`; alternative trim identified in § 2.5 |
| **Cross-WU** | WARNING | Tooling fragility (Engram #2636/#2639) — runtime ledger `complete` blocks subsequent `sdd-attempt acquire` after `passed` settle; archive may need manual | Anticipate at apply time; AI-34 manual-archive pattern (cherry-pick doc 0002 to branch) ready |

**No CRITICAL risks outside the cross-WU budget overrun and WU-2/WU-4 test correctness.**

---

## § 6 Aggregate review budget check

### § 6.1 Total forecast

| Category | Lines |
| --- | --- |
| Production NEW (`retry.go` + `doc.go` + `retry_test.go` + `execute_once.go`) | 209 |
| Production MODIFIED (`stream.go` + `conformance_suite.go` + `conformance_record.go` + `agenttest/doc.go`) | 27 |
| Test NEW (`a_i-35_1_test.go` + `a_i-35_2_test.go` + `a_i-35_3_test.go`) | 640 |
| Conformance NEW (`conformance_retry.go`) | 220 |
| **Total** | **~1096** |

(Design's original forecast was ~1100; my refinement is 4 lines lower.)

### § 6.2 Comparison to 1000-line budget

- **Budget**: 1000 changed lines (per doc 0002 amendment line 2093).
- **Forecast**: ~1096 changed lines.
- **Overrun**: **+96 lines (~10% over)**.
- **Concentration**: 640 of 1096 (58%) is in the three test files. These are exactly the files whose size reflects the rich scenario-banner documentation density that the AI-33/AI-34 precedent (and AGENTS.md's strict-TDD discipline) requires for reviewer cognitive load.

### § 6.3 Recommendation

**SINGLE-PR with `size:exception`** — the orchestrator asks the user at the apply checkpoint to approve the 96-line overrun.

**Reasoning**:

1. **Natural cohesion**. AI-35 is one retry mechanism with four tightly-coupled subnodes (predicate + backoff + replayability + boundary) plus one conformance case. The helper is incomplete without tests; the conformance depends on the helper. Splitting into chained PRs would create artificial seams (helper-without-tests is un-reviewable; conformance-without-helper is un-buildable).
2. **The 1000-line budget was explicitly pre-authorized** at session preflight per doc 0002 amendment line 2093. The proposal's Risk 7.6 carries the same disposition: "Single-PR strategy approved at session preflight; review budget raised to 1000 lines per doc 0002 amendment line 2093."
3. **The overrun is small** (~10%) and concentrated in test files whose density is mandated by the AI-33 / AI-34 precedent and the project's strict-TDD discipline. Trimming test verbosity sacrifices the review surface (scenario banners, in-test documentation of the seam construction) for a marginal line-count reduction.
4. **The trim alternative exists** (§ 2.5) if the user rejects `size:exception`, but it costs test documentation density. Chained PRs are explicitly NOT recommended — the natural unit is one PR.

### § 6.4 Question wording for the apply checkpoint

The orchestrator should relay the following question to the user at the apply checkpoint:

> **Question** (relayed by orchestrator at the sdd-apply checkpoint):
>
> AI-35's aggregate forecast is **~1096 lines** — about **96 lines over the 1000-line review budget** that doc 0002 amendment line 2093 authorized. The change is naturally cohesive (one retry mechanism: helper + 3 test files + conformance case), so chained PRs would create artificial seams. Three options:
>
> 1. **ACCEPT `size:exception`** (recommended) — proceed as a single PR; the ~96-line overrun reflects the natural scope of the helper + test files + conformance case body. The trim would compromise the scenario-banner documentation density that the AI-33 / AI-34 precedent and the project's strict-TDD discipline require.
> 2. **TRIM to under 1000** — tighten `retry.go` doc comments (~30 lines), merge the shared test helper into `a_i-35_3_test.go` (already done), reduce scenario-banner verbosity in `a_i-35_*_test.go` (~80 lines). Brings aggregate to ~890, well under 1000. Cost: ~30 minutes of trimming; test documentation density reduced.
> 3. **CHAINED PRs** (NOT recommended) — split into PR-1 (helper + smoke test + stream integration + AI-35.1 tests) and PR-2 (AI-35.2 / AI-35.3 tests + conformance). Cost: ~2 PRs, ~2 review cycles, and artificial seams in the helper-tests-review split.
>
> **Recommendation: option 1 (accept `size:exception`).** Which path do you want?

---

## § 7 Out-of-scope (for this change)

Re-stated from `proposal.md` § 3 (Non-goals) and `design.md` § 1 (Scope).

| Item | Owner | Reason |
| --- | --- | --- |
| Harness-level (turn-level) retry predicate | **Layer 2 / AG-15.1** — doc 0003 lines 706–712 | The Layer 2 half of `V-FAIL-15`. This change is the Layer 1 half. |
| Bounded backoff at the harness (composed-bound test body) | **Layer 2 / AG-15.2** — doc 0003 lines 714–719 | Same Layer 2 responsibility; reads `defaultMaxAttempts = 3` from `internal/retry/doc.go` verbatim (R-AIS-044). |
| Model failover seam | **Layer 2 / AG-15.3** — doc 0003 lines 721–726 | Re-budgets tokens, prices, cache prefix. Out of scope at any layer that doesn't own the budget. |
| Mid-stream retry (after a semantic event has been emitted) | **Forever** | Helper's seam is pre-stream only by construction; a mid-stream retryable failure reaches the harness via the terminal error event with `PartialOutput() == true`. R-AIS-043 / S-4 pins the seam by instrumented assertion. |
| Configuration surface for max attempts (`defaultMaxAttempts` promoted to `Config`) | **None** | Package-private; overridable in tests via `Config.MaxAttempts`; stays constant until a workload demands configurability (AI-34 precedent — Q3). |
| New top-level Go dependencies | **Blocked** | Forward guard at `import_boundary_test.go:107–162` is mechanical; helper is stdlib-only (`context`, `errors`, `math/rand/v2`, `time`). `backend/agent/go.mod` zero requires. |
| OpenRouter conformance bridge extension | **AI-38** — doc 0002 line 2118 | OpenRouter wrapper inherits helper transparently (`openrouter/wrapper.go` embeds `*openaicompat.Client` and delegates `Stream` verbatim). AI-35 lands the case body for `openaicompat`; AI-38 extends it for the wrapper. |
| Spec canonicalisation (`openspec/specs/ai-stream-lifecycle/spec.md` and `openspec/specs/ai-provider-conformance-suite/spec.md` gain inline blockquotes) | **sdd-archive** | The delta specs are reversible without code change. The canonical specs keep their pre-AI-35 wording until the archive phase lands. |
| `conformance_retry.go` companion test in `openrouter/conformance/bridge_test.go` | **AI-38** | Same as the OpenRouter extension above. |
| Helper signature shape refinements (rename `c.executeOnce`, fine-tune `Config` zero-value behaviour) | **sdd-apply** | The apply phase lifts the design § 5 signatures verbatim; refinements (renames, ordering) are the apply phase's per its commit-shape discipline. |
| Detailed conformance case body for S-CNF-074 and S-CNF-075 (suite-mechanism assertions) | **sdd-apply** | These are already enforced by the existing `factoryDefect` mechanism (`conformance_suite.go:266–280`); the case body's sub-tests just document the expected behaviour. |
| The `DefaultMaxAttempts = 3` value (constant's exact value) | **Design-locked** (proposal § 6 Q5 / explore § 5 D5 / design § 3 D10) | Confirmed at design time. Archive phase confirms wording match between helper doc and AG-15.2 test. |