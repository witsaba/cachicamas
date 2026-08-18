```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:c11f2d85e4cba01d0c7b2283d998a517dd2d7e90dd09c9a47531892cba11ab43
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 12/12
scenarios: 15/15
test_command: go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:9c356fe2bc27a4070cbb4fca08976d53a8c3be5295d11703bb59f2a6b59b4735
build_command: go build -trimpath ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: `cachicamas-agent-retry-failover` (AG-15 — retry policy and the failover seam, Layer 2 Wave 3, milestone 15 of 24)
**Version**: `specs/agent-retry-failover/spec.md` — `R-RTY-001`…`012`, `S-RTY-001`…`015`
**Mode**: Strict TDD
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ag-15-retry-failover`, branch `feat/agent-layer2-wave3-ag15`, merge base `main@bf482b0a`
**Scope**: read-only verification. Zero code changes were made by this phase.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 35 |
| Tasks complete | 35 |
| Tasks incomplete | 0 |

Verified directly: `grep -c '^- \[x\]' tasks.md` → 35, `grep -c '^- \[ \]' tasks.md` → 0.

### Build & Tests Execution

**Build**: PASSED

```text
$ cd backend/agent && go build -trimpath ./...
(no output, exit 0)
```

**Tests**: PASSED — 13 packages, 12 `ok`, 1 `[no test files]`, zero `FAIL`.

```text
$ cd backend/agent && go test -race -count=1 ./...
ok  	github.com/cachicamas/backend/agent/src/agent	9.425s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.511s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep	1.963s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest	2.201s
ok  	github.com/cachicamas/backend/agent/src/ai	4.847s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	3.005s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	174.313s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest	3.280s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	3.477s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	6.836s
?   	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures	[no test files]
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke	2.510s
ok  	github.com/cachicamas/backend/agent/src/handoff	2.211s
exit 0
```

`src/agent` — the package this change modifies — is `ok`. The `TestAI33_1_RaceCancelMidDo` flake the orchestrator flagged did **not** reproduce in this run; `src/ai/openaicompat` passed.

**Coverage** (`go test -covermode=atomic -coverprofile`, `src/agent`):

| File | Statement coverage | NFR |
|---|---|---|
| `loop.go` | **89.1 %** (277/311 statements) | `NFR-RTY-005` requires ≥ 80 % — **satisfied** |
| `retry_policy.go` `retryDecision` | 100.0 % | — |
| `retry_policy.go` `applyRetryTimingDefaults` | 100.0 % | — |
| `retry_policy.go` `newRetryJitter` | 100.0 % | — |
| `retry_policy.go` `computeRetryBackoff` | 93.3 % | — |
| `retry_policy.go` `retryWaitDelay` | 77.8 % | clamp arms (`hinted < 0`, `hinted > MaxDelay`) uncovered |
| `retry_policy.go` `DefaultRetrySleep` | 77.8 % | the timer-fires arm uncovered (by design — no test waits) |
| `harness.go` `typedHarnessFailureFromError` | 100.0 % | — |
| `loop.go` `preStreamAbortFailure` | 85.7 % | constructor-error arm uncovered (defensive) |
| `loop.go` `emitPreStreamAbort` | 87.5 % | constructor-error arms uncovered (defensive) |
| Package total | 78.9 % | no package-level threshold is specified |

### Quality Metrics

**Linter**: `golangci-lint cache clean` then `make lint` → **`0 issues.`** (exit 0) — re-run fresh by this phase, not inherited.
**Vet**: `go vet ./...` → clean (exit 0).
**Build**: `make build` → clean (exit 0).

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | PASS | `apply-progress.md` carries a complete TDD Cycle Evidence table (19 rows) |
| All tasks have tests | PASS | 19 test functions + 3 bites across 4 new test files |
| RED confirmed (tests exist) | PASS | Every named test file exists and every named test function is present |
| RED confirmed (evidence is real) | PASS | See "RED authenticity" below — every cited line number resolves exactly against the commit that recorded it |
| GREEN confirmed (tests pass) | PASS | Re-run by this phase: `src/agent` `ok` |
| Triangulation adequate | PASS | Predicate table = 7 cases + 2-case companion; bound = 2 values; retry-after = present/absent; non-retryable = 2 delivery positions; D1 = 3 paths × 2 continuation modes |
| Safety Net for modified files | PASS | Both modified pre-existing files (`loop.go`, `harness.go`) had a full package-green pre-edit run recorded, and both pre-existing observers of the pre-stream paths stay green byte-unchanged |

**TDD Compliance**: 7/7 checks passed.

#### RED authenticity — spot-checked against the commits, not trusted

Strict TDD was mandated, so the recorded RED output was re-derived rather than believed. Every cited line number was resolved against the exact blob in the commit that recorded it:

| Recorded RED | Cited location | Resolved against | Result |
|---|---|---|---|
| 0.1 — six `TestTurn_PreStream*EmitsTurnEnd` subtests | `retry_policy_test.go:146, 169, 203, 226, 259, 282` | `git show 288aa154:.../retry_policy_test.go` | **Exact** — all six are `assertPreStreamAbortSequence(t, got, …)` call sites, and the `nilContinuation=true/false` argument at each line matches the value in the recorded message. The helper calls `t.Helper()` (`retry_policy_test.go:111`), so Go attributes the failure to the caller's line, which is exactly what was recorded |
| 1.7 — bite `S-RTY-010` | `retry_policy_test.go:526` | `git show cc1cab86:…` line 526 | **Exact** — `t.Fatal("Run returned err = nil, want the mid-stream failure (G3 forbids retry; the run must end failed)")` |
| 1.8 — bite `S-RTY-011` | `retry_policy_test.go:443, 446, 457` | `git show cc1cab86:…` lines 443/446/457 | **Exact** — the turn-bracket-count `Errorf`, the aborted-count `Errorf`, and the `CheckStream` rejection `Errorf` respectively |
| 4.1 — `TestHarness_ExhaustedRetryPreservesTrueCategory` | `retry_policy_test.go:697, 700, 706, 715, 718` | HEAD (Phase 4 appended; no later shift) | **Exact** — Category, Retryable, Delivery, unwrap-identity and RetryAfter assertions, in that order |

The line numbers drift between commits in exactly the way real edits produce (Phase 1's `preStreamFailingProvider` fixture inserted ~57 lines above the Phase 0 tests; Phase 4's two added imports shifted later lines by 2), and every recorded number matches its own commit rather than HEAD. **No fabricated RED was found.** This is positive evidence of a real TDD cycle, not merely an absence of contradiction.

One recorded RED deviates from its scenario's literal prediction, in a way that *increases* credibility rather than reducing it: `S-RTY-010` predicts a failure "reporting that the provider recorded more than one request", but the observed RED is the earlier `err == nil` `t.Fatal`. That is exactly what a real run produces — with G3 deleted, the queued second script succeeds, `Run` returns nil, and the `t.Fatal` at line 526 fires before the request-count check at line 533 is ever reached. A fabricated RED would have matched the spec's prediction.

#### The three bites — reverted and residue-free

| Bite | Mutation executed | Revert verified |
|---|---|---|
| `S-RTY-010` (delete G3) | `retryDecision`'s `if failure.PartialOutput() { return verdictSurface }` deleted | `retry_policy.go:119-121` present at HEAD; predicate ordering intact |
| `S-RTY-011` (revert pre-stream emission) | `emitPreStreamAbort` made an unconditional no-op | `loop.go:635-655` present and wired at all three sites (`loop.go:306`, `:340`, `:358`) |
| `S-RTY-012` (perturb cited Layer 1 wording) | See WARNING-3 — the **test's own expectation slice** was perturbed, not the Layer 1 file | `git diff bf482b0a..HEAD -- backend/agent/src/ai/` is **empty**; `grep -rn "BITE-" backend/agent/src/agent/` → zero hits in the tree |

`git log -p bf482b0a..HEAD | grep "BITE-S-RTY"` returns 3 hits, **all three inside `apply-progress.md`'s own prose describing the bite** (`git log -S "BITE-S-RTY" --name-only` names only `apply-progress.md`). No bite mutation survives in any Go source at any commit on the branch.

### Predicate ordering — read, not assumed (obligation 5)

`retry_policy.go:111-126` implements the gates in exactly the spec's order, first match wins:

| Gate | Line | Code | Spec order |
|---|---|---|---|
| G0 | `harness.go:492-494` | `context.Cause(runCtx)` sentinel check, **before** `retryDecision` is called | ahead of G1–G5 — **correct**, and unchanged from AG-14's site and shape |
| G1 | `retry_policy.go:113-115` | `if !errors.As(terr, &failure) { return verdictSurface }` | 1st |
| G2 | `retry_policy.go:116-118` | `if !failure.Retryable() { return verdictSurface }` | 2nd |
| G3 | `retry_policy.go:119-121` | `if failure.PartialOutput() { return verdictSurface }` | 3rd — **fires after G2, which is the point** |
| G4 | `retry_policy.go:122-124` | `if attempt < bound { return verdictRetry }` | 4th |
| G5 | `retry_policy.go:125` | `return verdictExhausted` | 5th |

G3 after G2 is proven non-vacuous by two independent means: the table case `"G2 evaluated before G3: non-retryable AND partial output still surfaces at G2"` (`retry_decision_internal_test.go:86-91`), and the `S-RTY-010` bite. An inverted order would flip the G2-before-G3 table row.

`Delivery()` appears nowhere in `retryDecision` — confirmed by reading the whole function. G0 is not duplicated: `grep` finds the sentinel check at `harness.go:492-494` (the pre-existing AG-14 site, moved inside the attempt loop but with its condition byte-identical) and at `harness.go:521-523` (the new backoff-abort re-consult mandated by `R-RTY-008`/AD-6). No third check exists.

### Backoff wait selects on the run context (obligation 6)

`harness.go:530` — `if serr := timing.SleepFunc(runCtx, delay); serr != nil {` — passes **`runCtx`**, the `context.WithCancelCause` context the run derives at entry, never `context.Background()` and never a bare timer. On a non-nil sleep error the harness re-reads `context.Cause(runCtx)` (`harness.go:521-523`) and routes `ErrInterrupted`/`ErrShutdown` to `windDownRun`; anything else `break`s to the ordinary failure surface with the original `terr` intact.

`DefaultRetrySleep` (`retry_policy.go:266-281`) checks `ctx.Err()` first, then `select { case <-timer.C; case <-ctx.Done() }` — a real cancellation arm, not a bare timer. `TestHarness_InterruptAbortsBackoff` (`retry_backoff_test.go:296-375`) proves the end-to-end path with a `SleepFunc` that signals then blocks on `ctx.Done()`, and asserts the interrupted run-close carries a **nil** `*Failure`, that no failed run-close appears anywhere, that the provider recorded exactly 1 request, `errors.Is(err, agent.ErrInterrupted)`, and that `CheckStream` accepts the stream. A backoff waiting on a bare timer would hang this test forever, not pass it.

### `H`'s counting convention stated adjacent to every number (obligation 7)

`R-RTY-005` makes this itself falsifiable. Every production site that states `H`, the default, or the ceiling was opened:

| Site | Number stated | Convention stated adjacent? |
|---|---|---|
| `retry_policy.go:21-33` (package doc, composed ceiling) | `H × 4 = 12` | Yes — "H counts TOTAL attempts (R-RTY-005) — this is stated here, adjacent to the number, because a reader who does not already know the convention cannot otherwise tell whether '3' means 3 or 4" |
| `retry_policy.go:46-56` (`defaultRetryAttempts`) | `3` | Yes — "three Turn invocations for one logical turn — one initial attempt plus at most two retries… H counts TOTAL attempts, consciously diverging from Layer 1's own retries-after-the-first convention" |
| `harness.go:71-77` (`Harness.RetryAttempts`) | `H` | Yes — "up to H **total** Turn invocations (one initial attempt plus at most H-1 retries)" |
| `failover_policy.go:45-47` (`FailoverPrompt.Attempts`) | `H` | Yes — "the number of **Turn invocations** made for the logical turn" |

Layer 1's own `ai/internal/retry/doc.go` states its budget as retries-after-the-first ("`DefaultMaxAttempts = 3`", "`N+1 = 4` wire requests"), read verbatim and never edited. `3 × 4 = 12` is arithmetically consistent because both factors are totals. **No number appears without its convention.**

### Failover seam inertness pin (obligation 8)

`TestFailover_InertnessNilVsNoOp` (`failover_policy_test.go:163-259`) drives one fixture twice — `Failover: nil` and `Failover: agent.NoOpFailoverPolicy{}` — and compares event count, per-index kind, turn-scoping, `run_end` outcome + failure presence + category + retryability, `turn_end` outcome + failure presence + category + retryability + partial-output, `errors.Is` against both cancellation sentinels, `errors.As` to `*ai.Failure` and its category, and runs `CheckStream` on both. The production guard is `harness.go:544` — `if verdict == verdictExhausted && h.Failover != nil` — so a nil policy is never called, and `NoOpFailoverPolicy.Resolve` returns the zero `FailoverVerdict{}`, which the harness reads but never branches on (v1's verdict has no fields, so acceptance is unconstructible). Inertness holds structurally, not merely observationally.

### Charter scenario mapping — all seven have real, reachable, falsifiable tests (obligation 3)

| # | Charter scenario | Lines | Spec scenario | Concrete test | Result |
|---|---|---|---|---|---|
| 1 | retryable-with-no-output retries visibly | `0003:1470-1474` | `S-RTY-002`, bite `S-RTY-011` | `TestHarness_RetryVisibleAttempts` (`retry_policy_test.go:355`) | PARTIAL — see WARNING-1 |
| 2 | partial output forbids automatic retry | `0003:1476-1479` | `S-RTY-003`, bite `S-RTY-010` | `TestHarness_RetryPartialOutputNotRetried` (`retry_policy_test.go:510`) | COMPLIANT |
| 3 | non-retryable surfaces immediately | `0003:1481-1484` | `S-RTY-004` | `TestHarness_RetryNonRetryableSurfacesImmediately` (`retry_policy_test.go:564`, 2 subtests) | COMPLIANT |
| 4 | retry-after wins and backoff waits on the context | `0003:1494-1498` | `S-RTY-007`, `S-RTY-008` | `TestHarness_RetryAfterOverridesBackoff` (`retry_backoff_test.go:212`), `TestHarness_InterruptAbortsBackoff` (`:296`), `TestDefaultRetrySleep_PreCancelledContextReturnsImmediately` (`:384`) | COMPLIANT |
| 5 | the harness bound holds above any lower-layer retrying | `0003:1500-1504` | `S-RTY-005`, `S-RTY-009`, bite `S-RTY-012` | `TestHarness_BoundHoldsAboveH` (`retry_backoff_test.go:87`), `TestComposedCeiling_MatchesLayer1Wording` (`:459`) | PARTIAL — see WARNING-3, WARNING-4 |
| 6 | the retry path consults the failover seam before giving up | `0003:1514-1517` | `S-RTY-013` | `TestFailover_ConsultedOnceAtExhaustion` (`failover_policy_test.go:51`, 2 subtests), `TestFailoverPolicy_DocumentsRealImplementationObligations` (`:140`) | COMPLIANT |
| 7 | the none implementation changes nothing (pin) | `0003:1519-1522` | `S-RTY-014` | `TestFailover_InertnessNilVsNoOp` (`failover_policy_test.go:163`) | COMPLIANT |

**No charter leaf is orphaned, reduced, or driven by an assertion that cannot fail.** Each mapped test drives production code through the exported surface (`agent.Turn` or `agent.Harness.Run`) and asserts observable values, not shapes.

### Spec Compliance Matrix

| Requirement | Scenario | Test | Result |
|---|---|---|---|
| `R-RTY-001` | `S-RTY-001` | `retry_decision_internal_test.go > TestRetryDecision` (7 cases) + `TestRetryDecision_SameEvidenceRetriesThenExhausts` | COMPLIANT |
| `R-RTY-001` | `S-RTY-010` (bite) | task 1.7, RED verified against `cc1cab86:retry_policy_test.go:526` | COMPLIANT |
| `R-RTY-002` | `S-RTY-002` | `retry_policy_test.go > TestHarness_RetryVisibleAttempts` | **PARTIAL** — the Given's injected sleep function is absent (WARNING-1) |
| `R-RTY-002` | `S-RTY-011` (bite) | task 1.8, RED verified against `cc1cab86:retry_policy_test.go:443/446/457` | COMPLIANT |
| `R-RTY-003` | `S-RTY-003` | `retry_policy_test.go > TestHarness_RetryPartialOutputNotRetried` | COMPLIANT |
| `R-RTY-004` | `S-RTY-004` | `retry_policy_test.go > TestHarness_RetryNonRetryableSurfacesImmediately` | COMPLIANT |
| `R-RTY-005` | `S-RTY-005` | `retry_backoff_test.go > TestHarness_BoundHoldsAboveH` (bounds 3 and 5) | COMPLIANT |
| `R-RTY-006` | `S-RTY-006` | `retry_backoff_test.go > TestRetryTiming_BackoffJitterAndClamp` | **PARTIAL** — the source-scan half is a manual grep, not a test (WARNING-2), and the property it would scan for is violated (WARNING-1) |
| `R-RTY-007` | `S-RTY-007` | `retry_backoff_test.go > TestHarness_RetryAfterOverridesBackoff` (present/absent) | COMPLIANT |
| `R-RTY-008` | `S-RTY-008` | `retry_backoff_test.go > TestHarness_InterruptAbortsBackoff` + `TestDefaultRetrySleep_PreCancelledContextReturnsImmediately` | COMPLIANT |
| `R-RTY-009` | `S-RTY-009` | `retry_backoff_test.go > TestComposedCeiling_MatchesLayer1Wording` | **PARTIAL** — the "Layer 1 file byte-unchanged against the merge base" Then has no covering test (WARNING-4) |
| `R-RTY-009` | `S-RTY-012` (bite) | task 2.5 | **PARTIAL** — a different mutation than the scenario specifies (WARNING-3) |
| `R-RTY-010` | `S-RTY-013` | `failover_policy_test.go > TestFailover_ConsultedOnceAtExhaustion` + `TestFailoverPolicy_DocumentsRealImplementationObligations` | COMPLIANT |
| `R-RTY-011` | `S-RTY-014` | `failover_policy_test.go > TestFailover_InertnessNilVsNoOp` | COMPLIANT |
| `R-RTY-012` | `S-RTY-015` | `retry_policy_test.go > TestHarness_ExhaustedRetryPreservesTrueCategory` + `TestHarness_ExhaustedRetryPlainErrorStaysUnavailable` | COMPLIANT |

**Compliance summary**: 11/15 scenarios COMPLIANT, 4 PARTIAL, **0 UNTESTED, 0 FAILING**. 10/12 requirements fully closed; `R-RTY-006` and `R-RTY-009` each carry one unasserted clause.

**Reading the envelope's `12/12` and `15/15`**: the envelope's completed/total counts every requirement and scenario that has recorded, passing runtime evidence and is neither `FAILING` nor `UNTESTED`. All 12 and all 15 qualify. The four `PARTIAL` rows above are scenarios whose covering test passes but leaves one stated clause unasserted; they are recorded as WARNINGs, not as blockers, and the envelope's counts do not erase them — this section and the Issues section are the authoritative reading of coverage depth.

#### Delta-spec scenarios (this change's own additions to other capabilities)

| Scenario | Owning delta | Covering test | Result |
|---|---|---|---|
| `S-LSK-021` | `agent-loop-skeleton` | `TestTurn_PreStream{Build,Hook,Provider}ErrorEmitsTurnEnd` (6 subtests) | **PARTIAL** — the "`CheckStream` accepts each recorded stream unmodified" clause is unasserted on the nil-continuation path (WARNING-6) |
| `S-LSK-022` | `agent-loop-skeleton` | `TestTurn_SubstrateUntouched` / `TestTurn_PreRequestHook_SubstrateUntouched` | **PARTIAL** — the guards cover `src/agent/` + `go.mod`/`go.sum` only; the "every file under `backend/agent/src/ai/`" clause is untested (WARNING-4) |
| `S-LSK-023` | `agent-loop-skeleton` | both filters carry the same 6 exact-filename entries (verified by reading both) | **PARTIAL** — no test compares the two entry sets (WARNING-7; pre-existing since `S-LSK-019`) |
| `S-RUN-021` | `agent-run-driver` | `TestHarness_RetryVisibleAttempts` | COMPLIANT — one run bracket, brackets == attempts, distinct turn IDs, contiguous 1-based sequence, `CheckStream` clean |
| `S-RUN-103` | `agent-run-driver` | `TestHarness_RetryVisibleAttempts` + `TestHarness_RetryPartialOutputNotRetried` | **PARTIAL** — the "transcript holds no entry written by the failed attempt, and `CloseTurn` was not called for it" clause is unasserted (WARNING-5) |
| `S-ATT-014` | `agent-turn-termination` | `TestTurn_PreStream*ErrorEmitsTurnEnd` | **PARTIAL** — the distinguishing clause (`PartialOutput() == false` and pre-stream delivery on each aborting `*Failure`) is unasserted (WARNING-8) |
| `S-RUN-100` | `agent-run-driver` (pre-existing pin) | `harness_test.go:1870 TestHarness_TurnError_RunEndsTypedNoAppendNoCloseNoRetry` | COMPLIANT — **re-checked, not trusted**: its fixture is `ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)` at `loop_test.go:1299-1301` with `Retryable` **unset** (Go zero value `false`), so `R-RTY-001`'s **G2** catches it at the first gate that can, and `harness_test.go` is byte-unchanged vs `bf482b0a` |

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|---|---|---|
| `R-RTY-001` | Implemented | `retry_policy.go:111-126`; G0 ahead at `harness.go:492-494`, unchanged shape and site |
| `R-RTY-002` | Implemented | Inner attempt loop `harness.go:497-537`; `transcript` built once at `:470` and reused by reference; steering drain at `:463` sits outside the loop; no `CloseTurn` on a failed attempt (`harness.go:558` is reached only after `terr == nil`) |
| `R-RTY-003` | Implemented | G3 at `retry_policy.go:119-121` |
| `R-RTY-004` | Implemented | G2 at `:116-118`, reads `Retryable()` not a category allowlist |
| `R-RTY-005` | Implemented | `defaultRetryAttempts = 3` (`:56`), `Harness.RetryAttempts` zero-default (`harness.go:77`), `bound <= 0` selects the default (`harness.go:472-475`) |
| `R-RTY-006` | Implemented | `RetryTiming` (`:145-167`), `applyRetryTimingDefaults` (`:178-192`); `grep -rn "src/ai/internal/retry"` over `backend/agent/src/agent/` → zero import hits; `go list -deps ./src/agent` excludes the package |
| `R-RTY-007` | Implemented | `retryWaitDelay` (`:241-255`) reads `RetryAfter()` presence-typed; a reported zero is honoured as zero, not treated as absent |
| `R-RTY-008` | Implemented | `harness.go:521-533`; `DefaultRetrySleep` (`:266-281`) |
| `R-RTY-009` | Implemented | `retry_policy.go:21-33` package doc |
| `R-RTY-010` | Implemented | `failover_policy.go:27-79`; consult at `harness.go:544-549`, exactly once, before `failRun` |
| `R-RTY-011` | Implemented | nil guard at `harness.go:544` |
| `R-RTY-012` | Implemented | `typedHarnessFailureFromError` (`harness.go:295-300`) wraps the identical pointer; `wrapHarnessFailure` (`:272-281`) untouched and reached only on the plain-error arm |

**Substrate (`NFR-RTY-004`) — independently re-verified:**

```text
$ git diff --stat bf482b0a..HEAD -- backend/agent/src/ai/
(empty)
$ git diff --stat bf482b0a..HEAD -- backend/agent/src/agent/harness_test.go
(empty)
```

Full diffstat vs `main@bf482b0a`: 23 files, +3876/−37. Pre-existing **non-test** files that differ: exactly `{harness.go, loop.go}` — matching `S-LSK-022`'s claim. Pre-existing test files that differ: `{loop_test.go, loop_hook_test.go, loop_test.go`'s and `loop_hook_test.go`'s filter functions only`}`. `stream_check.go`, `stream_check_test.go`, `turn_events.go`, `failure.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `history.go`, `go.mod`, `go.sum` all byte-unchanged.

Both substrate filters carry the identical 6-entry exact-filename set (`/retry_policy.go`, `/retry_policy_test.go`, `/retry_decision_internal_test.go`, `/retry_backoff_test.go`, `/failover_policy.go`, `/failover_policy_test.go`) — one per file this change introduces, no wildcard, prefix or directory pattern. Verified by reading `loop_test.go:925-958` and `loop_hook_test.go:996-1019`.

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| AD-1 — pre-stream `turn_end(Aborted)` on all 3 paths | Yes | `loop.go:306`, `:340`, `:358`; `preStreamAbortFailure` at `:612-630` exactly as designed |
| AD-2 — pure predicate + inner attempt loop | Yes | `retryDecision` unexported, `harness.go:497-537` is a fresh inner loop, not a `continue` of the outer run loop |
| AD-3 — `H` counts total attempts, zero-default field | Yes | See obligation 7 above |
| AD-4 — `wrapHarnessFailure` sibling, preserve everything | Yes | `harness.go:295-300`; the identity property is asserted at `retry_policy_test.go:709-716` |
| AD-5 — `RetryTiming` mirrored fresh, never imported | Yes | Field-for-field mirror of `retry.Config` minus `AfterReader`; import absent at compile time and by `go list -deps` |
| AD-6 — wait selects on `runCtx` | Yes | See obligation 6 above |
| AD-7 — `FailoverPolicy` / `FailoverPrompt` / `FailoverVerdict` / `NoOpFailoverPolicy` | Yes | Names and shapes exactly as fixed |
| AD-8 — divergence detection by reading `doc.go` as a file | Yes, with a caveat | The test reads both files and compares; see WARNING-3 for how the bite was executed |

**Recorded deviations — assessed, not merely acknowledged:**

1. **`retry_decision_internal_test.go` (package-internal test file).** **Acceptable.** `NFR-RTY-001`'s carve-out names exactly this case ("The predicate's own table-driven test MAY exercise it through whatever surface `sdd-design` fixed"), design AD-2 fixes `retryDecision`/`retryVerdict` as unexported, and Go's export rule leaves no other route. Every requirement the predicate serves is independently proven externally by `Harness.Run`-driven scenarios. Both filters were widened in the same commit that introduced the file.
2. **`emitPreStreamAbort` private helper beyond AD-1's one named helper.** **Acceptable.** A DRY factoring of the shared `turn_end` + conditional `run_end` emission that delegates to the design-named `preStreamAbortFailure`; no behavioral surface added, nothing exported, and the design's guard posture (`ferr == nil`, skip-and-return-original) is preserved verbatim.
3. **`DefaultRetrySleep` exported.** **Acceptable, and correctly reasoned.** `S-RTY-008`'s second Given demands a direct unit test of the production default sleep, `NFR-RTY-001` requires behavioral tests to live in `package agent_test`, and its carve-out covers only `S-RTY-001`. Exporting is the only construction satisfying both. It is a pure additive export with no behavioral change, and it gives callers production timing defaults without reaching for the forbidden `ai/internal/retry`.
4. **`S-RTY-007` asserts a mathematically-guaranteed range rather than RNG-replicated exact values.** **Acceptable, and the better engineering choice.** `computeRetryBackoff` returns `base + jitter.Int64N(base)` for attempt 1, so the value is provably in `[BaseDelay, 2·BaseDelay)`. The test asserts membership for the retry-after-absent case and asserts the retry-after-present case is both exactly the reported value **and** outside that range. That is a sharper discriminator than an exact-value comparison would be, and it does not couple the test to `math/rand/v2`'s PCG internals — which the spec never asked it to know.

**One deviation was recorded inline in `apply-progress.md` task 2.5 rather than in its Deviations section, and is material** — see WARNING-3.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit (pure function, in-package) | 2 functions / 9 cases | 1 (`retry_decision_internal_test.go`) | `go test` |
| Unit (pure function, external) | 2 functions | `retry_backoff_test.go` (`TestRetryTiming_BackoffJitterAndClamp` partly, `TestDefaultRetrySleep_…`) | `go test -race` |
| Integration (drives `Turn` / `Harness.Run` end to end) | 15 functions | `retry_policy_test.go`, `retry_backoff_test.go`, `failover_policy_test.go` | `go test -race` |
| Documentation/structural (reads source as a file) | 2 functions | `retry_backoff_test.go`, `failover_policy_test.go` | `os.ReadFile` |
| **Total** | **19 functions + 3 bites** | **4 files** | |
| E2E | 0 | — | not installed / not applicable to a library module |

The distribution is correct for a library: the charter's behavioral claims are all proven through the exported `Turn`/`Harness.Run` surface, not through internal seams.

### Assertion Quality

| File | Line | Assertion | Issue | Severity |
|---|---|---|---|---|
| `failover_policy_test.go` | 193-231 | `for i := range nilEvents { … }` | The comparison loop is guarded only by `len(nilEvents) != len(noOpEvents)`; if both streams were empty the loop body would never run and the whole inertness comparison would pass vacuously. In practice both streams always carry ≥ 2 events, and the `errors.As`/category assertions after the loop are unconditional, so the test cannot silently degrade to nothing — but a `len(nilEvents) == 0` fatal would close the hole. | SUGGESTION |

No tautologies. No assertion without a production-code call. No orphan empty-collection check. No smoke-test-only pattern. No CSS/implementation-detail coupling. No mock-heavy file (zero mocking frameworks in use; the two test doubles are hand-written providers/policies that record real inputs). Every count assertion in this change is one the spec explicitly designates as *the contract* (`R-RTY-003`: "the count is the contract"; `R-RTY-005`: same wording; `R-RTY-010`: same wording).

**Assertion quality**: 0 CRITICAL, 0 WARNING, 1 SUGGESTION.

### Count-assertion drift audit (obligation 10)

Scanned the whole change directory for count phrasings that a later milestone could silently falsify:

| Location | Phrasing | Verdict |
|---|---|---|
| `specs/agent-retry-failover/spec.md` header | *"No total is stated: a count goes silently false the moment a later milestone appends (`S-LSK-020`)"* | **Correct discipline.** The spec states an allocated **range**, never a total |
| `R-RTY-003` / `S-RTY-003` | "exactly one request" | The count **is** the contract — a second request is the exact defect. Safe |
| `R-RTY-005` / `S-RTY-005` | "exactly `H`" | The count **is** the contract — it is the only observable proving the bound. Safe |
| `R-RTY-010` / `S-RTY-013` | "exactly once" | The count **is** the contract — "consulted once at exhaustion" is the charter's own claim. Safe |
| `R-RTY-009` | `3 × 4 = 12` | Guarded by `TestComposedCeiling_MatchesLayer1Wording`, which fails if either layer's wording drifts |
| `NFR-RTY-004` | "the every-kind-constructible guard MUST pass at its committed kind count; this change registers none" | Deliberately avoids restating 25. Safe |
| `specs/agent-loop-skeleton/spec.md` `S-LSK-020` | "neither states a scenario total" | Upheld by this change's own delta |
| `specs/agent-loop-skeleton/spec.md` `S-LSK-006` (verbatim, pre-existing) | "still passes at 25 kinds" | Pre-existing verbatim reproduction, not introduced here. Not a regression |
| `loop_test.go:1267` (pre-existing) | "(5 charter → 7 spec + 2 bites = 9 total)" | Pre-existing AG-07 comment, byte-unchanged by this change |

**No new count assertion was introduced where the count is not itself the contract.** This change followed the discipline correctly.

### Requirement / scenario agreement audit (obligation 11)

Grepped the change directory for phrases this change introduces and compared each requirement against its own scenarios:

| Phrase | Requirement says | Scenario(s) say | Agree? |
|---|---|---|---|
| "counts **total attempts**" | `R-RTY-005` | `S-RTY-005` observable is `len(Requests()) == H` (a total) | Yes |
| "consulted **exactly once**" | `R-RTY-010` | `S-RTY-013` asserts `len(calls) == 1` | Yes |
| composed ceiling `H × 4 = 12` | `R-RTY-009` | `S-RTY-009`; `ai-stream-lifecycle` back-annotation "3 total harness attempts × 4 wire requests = 12" | Yes |
| "`Delivery()` MUST NOT be a gate" | `R-RTY-001` | `S-RTY-001`'s table varies `PartialOutput`, never `Delivery` | Yes |
| "zero value declines" / "no acceptance field at all" | `R-RTY-010` | `S-RTY-013` "its returned zero verdict was read as a decline" | Yes |
| "**a recording sleep function that returns nil without waiting**" | `R-RTY-006`'s "no test may reach a wall clock" | `S-RTY-002`'s Given states it; the implementing test does **not** inject one | **NO — WARNING-1** |
| "a scratch tree in which the cited wording in the **Layer 1 package documentation** is perturbed" | `R-RTY-009` | `S-RTY-012`; `ai-stream-lifecycle` back-annotation repeats it; the executed bite perturbed the **test's own expectation slice** | **NO — WARNING-3** |

Two disagreements found, both between a scenario's Given/mechanism and what was actually executed. Both are recorded below.

### Documentation obligations (obligation 9)

| Obligation | Verified at | Result |
|---|---|---|
| doc 0003 AG-15 checklist tick | `docs/architecture/milestones/0003-…md:2172` — `- [ ]` → `- [x] Partial-output failures are never silently retried; retry attempts are visible events — closed by AG-15.1.` | Done |
| Milestone counter | line 3 — **14 of 24** → **15 of 24** | Done |
| Wave 3 prose | line 3 — "Wave 3 opens with AG-12…AG-14" → "…AG-12…AG-15" | Done |
| AG-15 status-header narrative | line 3 — a full AG-15 sentence in AG-13/AG-14's density and style, naming the predicate, visible byte-identical retries, the never-retry-after-output rule, immediate non-retryable surfacing, the injected timing seam with retry-after override and interrupt-safe waits, the composed ceiling and its divergence test, true-category preservation, and the failover seam's inertness pin | Done |
| Composed-ceiling documentation "where both layers' readers find it" | `retry_policy.go:21-33` (Layer 2 half) + the shipped `ai/internal/retry/doc.go` (Layer 1 half, unedited), tied together by `TestComposedCeiling_MatchesLayer1Wording` | Done |
| `R-15`/G8 back-annotation | `:2172` is the only checklist item naming AG-15 as its closer; `R-15`'s row at `:69` is a static descriptive row with no checkbox | Correct — nothing further to tick |

The doc 0003 diff is exactly 4 changed lines across 2 hunks: the status header and the one checkbox. Nothing else in the doc moved.

### Issues Found

**CRITICAL**: None.

**WARNING**:

1. **`TestHarness_RetryVisibleAttempts` reaches the production wall clock, violating `R-RTY-006` and `NFR-RTY-002`.**
   `retry_policy_test.go:372-377` constructs the harness **without a `RetryTiming` field**, so `applyRetryTimingDefaults` (`retry_policy.go:178-192`) installs `DefaultRetrySleep` with `BaseDelay = 100ms`. The test therefore performs two real backoff waits of `[100ms, 200ms)` and `[200ms, 400ms)`. Measured: `--- PASS: TestHarness_RetryVisibleAttempts (0.35s)` versus `0.00s` for every sibling that injects `instantSleep`.
   This contradicts three normative statements: `S-RTY-002`'s Given ("**and a recording sleep function that returns nil without waiting**"), `R-RTY-006` ("Backoff timing is injected, and **no test may reach a wall clock**"), and `NFR-RTY-002` ("this change introduces **no legitimate use of real clock time at all**").
   It is a genuine drift, not a nitpick: the test was authored in Phase 1 **before** the timing seam existed (Phase 2), and when Phase 2 landed a 100 ms default the pre-existing test silently began sleeping for real. Nothing failed, because the property was never test-enforced (see WARNING-2), and `apply-progress.md` task 2.7's grep for `time.Sleep(` correctly returned zero hits — the test does not call `time.Sleep`, it calls `time.NewTimer` transitively through production code, which the grep cannot see. `instantSleep`'s own doc comment at `retry_backoff_test.go:28-32` even cites this test by name as the convention it makes concrete, so the omission was noticed and not closed.
   **Fix**: add `RetryTiming: agent.RetryTiming{SleepFunc: instantSleep},` to the harness literal at `retry_policy_test.go:372-377`. One line. Not blocking — the test passes and asserts the right things — but it should not archive as-is.

2. **`S-RTY-006`'s second half has no covering test.** The scenario reads "given the full test suite for this capability, when its **sources are scanned**, then no test file in it calls a wall-clock sleep and no production file added by this change imports the Layer 1 retry package." `apply-progress.md` task 2.7 discharges this with a manual `grep` — but the spec's own preamble states "**Every scenario is independently verifiable by `cd backend/agent && make test`**", and a manual grep is not. The import half is genuinely compiler-enforced (Go's internal-visibility rule makes the import unbuildable), so only the wall-clock half is materially unguarded — and WARNING-1 is precisely the defect an enforcing test would have caught. Note that a naive `time.Sleep(` scan would **not** have caught WARNING-1 either; a meaningful guard would have to assert per-test elapsed time or scan for harness literals lacking an injected `SleepFunc`.

3. **Bite `S-RTY-012` was executed against a different target than the scenario specifies, and one delta would archive that discrepancy as fact.**
   `S-RTY-012` says: "Given a scratch tree in which the cited wording in the **Layer 1 package documentation** is perturbed, when `S-RTY-009` runs, then it FAILS reporting the divergence." `specs/ai-stream-lifecycle/spec.md`'s back-annotation repeats it: "discharged by the bite `S-RTY-012`, **which perturbs the cited Layer 1 wording in a scratch tree** and records the resulting failure before reverting."
   What was actually executed (`apply-progress.md` task 2.5): the **test's own expectation slice** `wantLayer1RetryDocSentences` (`retry_backoff_test.go:410-413`) was perturbed with a `BITE-S-RTY-012-PERTURBED` marker; the real `ai/internal/retry/doc.go` was never touched.
   **Proving power is equivalent**, and I verified why rather than assuming it: `checkComposedCeilingWording` (`retry_backoff_test.go:428-436`) is a pure `strings.Contains(content, want)` over both operands, so perturbing `want` and perturbing `content` produce the identical failure. The cross-layer contract **is** genuinely enforced by a test. The apply phase's caution was also reasonable — the alternative required temporarily editing a file the change forbids touching.
   **The problem is the artifact, not the engineering.** At archive, `S-RTY-012` and the `R-AIS-044` back-annotation are promoted verbatim into the canonical specs, where they will assert a bite mechanism that was never run. That is the "a citation is not evidence" defect class, permanently encoded. **Recommendation**: before archive, amend `S-RTY-012`'s Given and the `ai-stream-lifecycle` back-annotation sentence to describe the executed mutation (perturbing the test's own captured expectation, which the `strings.Contains` symmetry makes equivalent), or re-run the bite as literally specified. Do **not** archive both sentences unchanged.

4. **`S-RTY-009`'s third Then ("the Layer 1 file is byte-unchanged against the merge base") and `S-LSK-022`'s "every file under `backend/agent/src/ai/`" clause have no covering test.** Both existing substrate guards (`TestTurn_SubstrateUntouched` at `loop_test.go:1220-1255`, `TestTurn_PreRequestHook_SubstrateUntouched` at `loop_hook_test.go:812-850`) diff only `backend/agent/src/agent/` plus `go.mod`/`go.sum`. `grep -rn "backend/agent/src/ai" backend/agent/src/agent/*_test.go` finds no guard covering that tree. **The property itself holds** — I verified `git diff --stat bf482b0a..HEAD -- backend/agent/src/ai/` is empty — but it holds by discipline, not by a test. A future milestone could edit Layer 1 from Layer 2's branch and no test would notice.

5. **`S-RUN-103`'s "the transcript holds no entry written by the failed attempt, and `CloseTurn` was not called for it" is unasserted.** No test in this change inspects the `History` value it passes to the harness (`grep` over `retry_policy_test.go` finds `agent.NewHistory()` constructions and zero reads). Design AD-2 argues the property statically and correctly. **Material risk is low**: the byte-identity assertion (`requests[i].Equal(requests[0])`, `retry_policy_test.go:394-398`) would fail if anything were written to the transcript between attempts, since the request is rebuilt from it each time. The `CloseTurn` half has no such proxy.

6. **`S-LSK-021`'s "`CheckStream` accepts each recorded stream unmodified" is unasserted on the nil-continuation path.** `assertPreStreamAbortSequence` (`retry_policy_test.go:110-161`) checks kinds, outcomes and failure presence, but never calls `CheckStream`. The continuation-path pre-stream abort **is** `CheckStream`-validated indirectly, by `TestHarness_RetryVisibleAttempts` (`:457`), `TestHarness_InterruptAbortsBackoff` (`retry_backoff_test.go:371`) and `TestFailover_InertnessNilVsNoOp` (`failover_policy_test.go:251-258`), whose streams all contain pre-stream aborts. Only the `run_start / turn_start / turn_end / run_end` nil-continuation shape is unvalidated. One added line per subtest would close it.

7. **`S-LSK-023`'s "when their entry sets are compared, then they are identical to each other" is unasserted.** No test compares `filterOutLoopFiles`' and `filterOutLoopHookFiles`' suffix lists. I verified by reading both that the 6-entry sets are identical. This is a **pre-existing pattern** — `S-LSK-019` (AG-14) makes the same claim with the same absence — so it is not an AG-15 regression, but it is inherited debt worth naming.

8. **`S-ATT-014`'s distinguishing clause is unasserted.** The scenario states: "when each aborting `*Failure` is inspected, then it reports `PartialOutput() == false` and **pre-stream delivery**, while `S-ATT-007`'s mid-stream failure reports mid-stream delivery — so the two abort classes stay distinguishable by evidence while sharing one outcome member", and explicitly says it "exists so that a regression in it fails **this** capability's suite too". `assertPreStreamAbortSequence` asserts only that the failure is non-nil; `grep -n "DeliveryPreStream\|PartialOutput()" retry_policy_test.go` finds no such assertion on any of the six pre-stream abort subtests. **The property holds by construction** — `preStreamAbortFailure` (`loop.go:612-630`) routes plain errors through `ai.PreStreamFailure`, which is the unconstructible-partial-output cell and always reports pre-stream delivery — but the scenario's stated purpose (fail *this* suite on regression) is not achieved.

**SUGGESTION**:

9. **`TestDefaultRetrySleep_PreCancelledContextReturnsImmediately` (`retry_backoff_test.go:390-399`) reads a real clock** (`time.Now()` / `time.Since` with a 200 ms ceiling). The test's own comment pre-empts the objection ("a diagnostic ceiling proving no real timer was armed, not a synchronization mechanism"), which is a fair reading of `NFR-RTY-002`'s *synchronization* clause — but not of its absolute clause ("no legitimate use of real clock time at all"). Consider asserting on the returned error's identity to `context.Canceled` alone, which is sufficient and clock-free.

10. **`TestFailover_InertnessNilVsNoOp`'s comparison loop lacks a non-empty guard.** See the Assertion Quality table. Add `if len(nilEvents) == 0 { t.Fatal(...) }` before the loop.

11. **`tasks.md`'s coverage table omits every delta-spec scenario.** It maps `S-RTY-001`…`015`, D1, the filters, the ceiling and the docs — but `S-LSK-021`, `S-LSK-022`, `S-LSK-023`, `S-RUN-021`, `S-RUN-103` and `S-ATT-014` appear nowhere in it, which is why four of them (WARNING-5, 6, 7, 8) reached this phase only partly covered. A future `sdd-tasks` run should enumerate delta scenarios in the coverage table alongside the owning capability's own.

12. **Carry forward to archive (already flagged by apply as task 5.6, re-confirmed here as real).** `specs/agent-loop-skeleton/spec.md:10-12` carries a `## MODIFIED Header — the allocated scenario range` section outside the normal `## MODIFIED Requirements` shape. The archive step **must** apply it to the promoted spec's own header line (`S-LSK-001` through `S-LSK-020` → through `S-LSK-023`), not merely merge the `R-LSK-001`/`R-LSK-004` blocks. Verified: the delta does add `S-LSK-021`, `S-LSK-022` and `S-LSK-023`, so leaving the header at `S-LSK-020` would make the header's own range claim false — the exact defect `S-LSK-020` exists to prevent.

### Acceptance criteria (spec § Acceptance criteria)

| # | Criterion | Result |
|---|---|---|
| 1 | Every `S-RTY-001`…`015` has recorded evidence; all three bites RED-recorded before GREEN | Met — RED authenticity independently re-derived; `S-RTY-012`'s mechanism deviates (WARNING-3) |
| 2 | All seven charter Gherkin scenarios mapped and closed, none reduced | Met |
| 3 | `make test` green under `-race`; `make lint` (after cache clean), `make build`, `make vuln-check` clean | Met — test/lint/build re-run by this phase, all clean; `vuln-check` per apply's record |
| 4 | `CheckStream` accepts the multi-attempt run stream unmodified, `stream_check.go` byte-unchanged | Met — `retry_policy_test.go:457`; `stream_check.go` byte-unchanged |
| 5 | Every AG-13/AG-14 harness, cancellation and loop test passes byte-unchanged; exceptions enumerated | Met — `harness_test.go` byte-unchanged; the only test-file edits are the two filter functions, which are the enumerated widening |
| 6 | Both substrate filters carry an identical exact-filename entry set, one per introduced file, no wildcards | Met — 6 entries, identical, verified by reading both |
| 7 | All six spec deltas written, every cited line re-read against the shipped change | Met — all seven files present (six deltas + the owning spec); cited lines spot-checked and resolved |
| 8 | doc 0003 checklist ticked, R-15/G8 back-annotated, milestone counter bumped | Met |

### Verdict

**PASS WITH WARNINGS** — 0 CRITICAL, 8 WARNING, 4 SUGGESTION. The full `-race` suite is green (exit 0), lint is clean from a cleared cache, the gate ordering and the backoff's cancellation arm are correct as read in the source, all seven charter scenarios have real falsifiable tests, the three bites were genuinely executed and cleanly reverted, and the RED evidence is verifiably authentic rather than reconstructed. Nothing found blocks archive; the two findings worth acting on first are the one-line wall-clock fix (WARNING-1) and amending `S-RTY-012` / the `R-AIS-044` back-annotation so the archived specs do not assert a bite mechanism that was never run (WARNING-3).
