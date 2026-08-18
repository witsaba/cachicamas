# Design: AG-15 — Retry policy and the failover seam

> Change: `cachicamas-agent-retry-failover` · Phase: sdd-design · Binds `sdd-spec` and `sdd-tasks`.
> Inputs: proposal.md (normative), explore.md, doc 0003:1444-1523. Every `file:line` below was opened in this phase against `main@bf482b0a` in the worktree. Exceeds the generic 800-word budget deliberately: the orchestrator mandated eight closure obligations (test enumeration, H convention, seam shapes, scenario mapping) that are themselves the artifact.

## Technical Approach

Close the pre-stream bracket gap in `Turn` first (D1), then insert a pure, ordered-gate predicate at `Harness.Run`'s failure branch between the AG-14 cancellation carve-out (`harness.go:460-462`) and `failRun` (`:469`), driven by an **inner attempt loop** around the `Turn` invocation. Backoff runs through a fresh timing seam mirroring `retry.Config`'s shape (`retry.go:25-45`) — never importing `ai/internal/retry` (internal-visibility rule; sibling roots). The failover seam is an interface with a typed zero-value-declines verdict plus a shipped declining implementation. The exhausted-retry `run_end` preserves the full typed evidence via a conditionally-routed sibling beside `wrapHarnessFailure`.

## Architecture Decisions

### AD-1 — D1 mechanics: pre-stream `turn_end(Aborted, failure)` (closes proposal Decision 1's three sub-questions)

**Choice**: All three paths after `turn_start` (`loop.go:304-308` build, `:317-328` hook, `:332-338` streamErr) emit `turn_end(TurnOutcomeAborted, failure)` before `closeSink`, mirroring the mid-stream block (`:387-405`). Scope starts at the turn bracket: the `NewRunStart`/`NewTurnStart` constructor-error returns (`:287-300`) precede an open turn and stay unchanged.

1. **Plain-error wrap (sub-question 1)**: new loop-side helper `preStreamAbortFailure(err error) (*Failure, error)`: `errors.As(err, &f)` for an `*ai.Failure` → `NewFailure(f)` (`failure.go:33-38`); otherwise `ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, Cause: err})` then `NewFailure`. This follows `typedFailureFromError`'s category precedent (`scheduler.go:1062-1072`) but deliberately diverges on delivery: a pre-stream abort is truthfully `DeliveryPreStream` (`provider_failure.go:610-612`), whereas the scheduler's sibling is mid-stream because a tool failure always is. The hook path's existing `*ai.PreStreamFailure(UnsupportedCapability)` (`loop.go:319-322`) and the streamErr path's provider-typed failure are reached by the same `errors.As` arm — one helper, three call sites. `errors.As` over a bare type assertion: `agenttest` wraps `ErrScriptsExhausted` with `fmt.Errorf` (`fake_provider.go:101`), and `failureFromResult` (`retry.go:119-125`) is the Layer 1 precedent. On a wrap/constructor error the emission is skipped best-effort and the original error still returned — the mid-stream block's own `ferr == nil` guard posture (`loop.go:388-404`).
2. **No substrate release (sub-question 2)**: the emission uses the existing `NewTurnEnd` + `TurnOutcomeAborted` and existing `NewRunEnd`; `turn_events.go`, `failure.go`, `run_events.go`, `stream_check.go` stay byte-unchanged. AG-15 requests **no** `R-LSK-004` release; the fix is emitter-side only (`R-RUN-003` pins `stream_check.go`).
3. **Nil vs continuation path (sub-question 3)**: mirror `R-ATT-005`'s mid-stream shape exactly — `turn_end(Aborted, failure)` on **both** paths; `run_end(RunOutcomeFailed, failure)` additionally on the **nil path only** (`loop.go:401-405`; R-LSK-001 point 2 — the caller's run owns the run bracket on the continuation path). Proof no existing test changes: see the enumeration section below — the two existing observers assert properties, not sequences.

**Rejected**: emitting only `turn_end` on the nil path (leaves the nil-path run bracket dangling, diverging from the mid-stream precedent for no reason); fixing `CheckStream` (forbidden, `agent-run-driver/spec.md:125`).

### AD-2 — Predicate placement and the inner attempt loop (proves "identical transcript")

**Choice**: pure function `retryDecision(terr error, attempt, bound int) retryVerdict` implementing gates **G1–G5** exactly per the proposal's table, first match wins; verdict vocabulary `verdictSurface` / `verdictRetry` / `verdictExhausted`. G0 stays the existing inline cause check (`harness.go:460-462`), untouched and ahead. The evidence is reached by `errors.As(terr, &aiFailure)` — `Turn` returns its fatal raw (`loop.go:411`, `:414`, `:338`, `:327`).

The retry re-invocation is an **inner attempt loop** wrapped around `harness.go:447`'s `Turn` call (fresh `turnSink` + forwarder goroutine per attempt, since `Turn` closes its sink), NOT a `continue` of the outer run loop. This is what makes AG-15.1 scenario 1's byte-identity provable rather than assumed:

- `transcript` is built once per logical turn at `harness.go:428` and reused by reference across attempts.
- No failure path writes `History`: `Turn`'s only history writer is `finishContinuationTurn`, called solely on the completion arm (`loop.go:350`); the fatal branch returns at `:411`/`:414` before it; pre-stream paths return before the accumulator exists; `newTurnAccumulator` holds no `History` reference (`loop.go:341`, `:742`); `Run`'s failure branch appends nothing and never calls `CloseTurn` (`harness.go:463-469`; `CloseTurn` only after `terr == nil`, `:472`).
- The steering drain (`harness.go:422-426`) sits outside the attempt loop, so no steered message can interleave two attempts of one logical turn.

Therefore attempt n+1's request is built from identical inputs; the test asserts it with `ai.Request.Equal` (precedent `loop_hook_test.go:447`), mirroring `R-AIS-043 / S-1` at harness scope. Plain errors from `hist.Append` (`harness.go:402-404`, `:423-425`, `:485-487`) keep their direct `failRun` calls — the predicate governs only `terr` from `Turn`.

### AD-3 — The attempt bound `H`: convention, value, carrier

**Choice**: `H` counts **total attempts** — `Turn` invocations for one logical turn — **consciously differing** from Layer 1's retries-after-the-first (`retry.go:15-18`). Written reasons: (1) the charter's observable is `len(provider.Requests()) == H` exactly — a total-attempts count; (2) the composed ceiling multiplies two totals: Layer 1's own doc states its per-call total as `N+1 = 4` wire requests (`ai/internal/retry/doc.go:3-6`), so `H × 4` needs `H` total — under a budget convention the ceiling would be `(H+1) × 4`, corrupting the arithmetic; (3) re-importing "3 means 4" at a second layer doubles the off-by-one trap Layer 1's doc already has to disclaim. The divergence is stated in `retry_policy.go`'s package doc adjacent to the ceiling.

**Value and carrier**: `defaultRetryAttempts = 3` (3 `Turn` invocations = 1 initial + 2 retries; worst-case composed ceiling **3 × 4 = 12 wire requests**), carried as a zero-default `Harness.RetryAttempts int` field — the `Scheduler.WindDownBound` idiom (`scheduler.go:611-613`), `<= 0` selects the default. G4 fires while `attempt < H`; G5 at `attempt == H`.

### AD-4 — `wrapHarnessFailure` sibling: preserve **everything**, by not reconstructing

**Choice**: new sibling `typedHarnessFailureFromError(cause error) (*Failure, error)`: `errors.As` reaches an `*ai.Failure` → return `NewFailure(f)` **wrapping the identical value**; otherwise route to the untouched `wrapHarnessFailure` (`harness.go:250-259`), byte-identical `Unavailable` behavior for plain errors. `failRun` calls the sibling. This is AG-14's conditionally-routed-sibling shape (`scheduler.go:1088-1099`) taken one step further: the `*ai.Failure` arm does not rebuild a report at all — it reuses the loop's own `turn_end` precedent (`loop.go:387-388`).

**Extent (the design question, closed)**: ALL fields are preserved — Category, Retryable, RetryAfter, PartialOutput, Delivery, plus RawLabel/StatusClass/RequestID and the cause chain — because the wrapped pointer is the original. Justified from what the exhausted-retry consumer must conclude: *why* it failed (Category), *whether re-prompting can help and when* (Retryable, RetryAfter), *whether output was already rendered* — the same question G3 answers, which a consumer needs to avoid duplicating content on a manual re-prompt (PartialOutput), and *whether anything crossed the carrier at all* (Delivery — pre-stream means nothing was shown). Verified legal: `RunEnd.validate` checks only outcome-vocabulary and failure-iff-Failed (`run_events.go:161-172`) — it never inspects delivery; `ErrorEvent`'s pre-stream rejection (`provider_failure.go:626-637`) governs Layer 1's stream-terminal event only, which `run_end` is not. Bonus identity property, testable: on the exhausted report, `runEnd.Failure().Unwrap()` is pointer-identical to the final attempt's `*ai.Failure` — the `turn_failure_test.go:65` pin shape at harness scope. **Grep obligation answered**: zero existing tests assert the wrapped `run_end` category — `harness_test.go:1898` asserts presence only; `cancellation_interrupt_test.go:111,:223,:607` and `cancellation_shutdown_test.go:200` assert absence on wind-down paths; `turn_failure_test.go:111` is loop-level. Nothing pins `Unavailable` on the harness `run_end`.

### AD-5 — The timing seam: `RetryTiming`, mirrored fresh

**Choice**: new exported struct in the `agent` package, shape-mirroring `retry.Config` (`retry.go:25-45`) with the harness-irrelevant `AfterReader` dropped (the harness reads `(*ai.Failure).RetryAfter()` directly, `provider_failure.go:471-476`):

```go
type RetryTiming struct {
    NowFunc    func() time.Time                                  // seeds jitter
    SleepFunc  func(ctx context.Context, d time.Duration) error  // MUST return promptly on ctx cancellation
    BaseDelay  time.Duration
    MaxDelay   time.Duration
    JitterSeed int64
}
```

Carried as `Harness.RetryTiming RetryTiming` (value; zero value fully sane). Production defaults via an `applyDefaults` mirror (`retry.go:138-158`): `NowFunc = time.Now`, `SleepFunc` = the package's own context-aware sleep, `BaseDelay = 100ms`, `MaxDelay = 30s` (`retry.go:21-22`), `JitterSeed = 0` → seeded from `NowFunc` (`retry.go:160-165`). Backoff computation mirrors `computeBackoff` (`retry.go:169-190`); retry-after overrides the computed delay when present, clamped to `[0, MaxDelay]` (`retry.go:102-110`). **Hard constraint honored**: shape mirrored, package never imported (`ai/internal/retry` is internal to `.../src/ai`; `.../src/agent` is a sibling — compile-time forbidden).

### AD-6 — The backoff wait's cancellation arm

**Choice**: the wait receives **`runCtx`** (the `context.WithCancelCause` context, `harness.go:360-362`), never `context.Background()`. The default sleep mirrors `defaultSleep` (`retry.go:192-207`): early `ctx.Err()` check, `select { case <-timer.C; case <-ctx.Done() }` — the `Scheduler.WindDownBound` timer discipline (`scheduler.go:605-628`). On a non-nil sleep error, `Run` re-checks `context.Cause(runCtx)`: `ErrInterrupted`/`ErrShutdown` → `windDownRun` (`harness.go:273-285`), exactly like an iteration-boundary signal; a bare non-sentinel cancellation falls through to the failure surface — parity with the existing scope rule (`harness.go:456-462`, R-CAN-001).

### AD-7 — The failover seam

**Choice** (fixes Decision 3's open names):

```go
type FailoverPolicy interface {
    // Resolve is consulted exactly once when the retry budget for a
    // logical turn is exhausted (G5). Contract for a real (post-v1)
    // implementation: it MUST re-count the context budget and restart
    // the cache prefix for the substitute route (AGS-D).
    Resolve(ctx context.Context, prompt FailoverPrompt) FailoverVerdict
}
type FailoverPrompt  struct { Attempts int; Failure *ai.Failure } // exported-field report, FailureReport posture
type FailoverVerdict struct{} // zero value declines; acceptance fields extend non-breakingly (AG-04 posture)
type NoOpFailoverPolicy struct{} // shipped decliner: Resolve returns FailoverVerdict{} (S-LSK-010 shape)
```

- **Declines = the zero value.** v1 ships no acceptance field at all, so an accepting verdict is unconstructible — the `PreStreamFailure` unconstructible-cell posture (`provider_failure.go:604-609`), stronger than an ignored `Accept bool`. v2 adds fields (route, re-budget) non-breakingly; existing implementations returning `FailoverVerdict{}` keep compiling as decliners.
- **Lives on `Harness`** (`Harness.Failover FailoverPolicy`, nil-default), not `TurnOptions`: exhaustion is a run-driver concept — `Turn` knows nothing of attempts; `TurnOptions.PermissionPolicy` sits where its consumer (the scheduler) is, and this seam's consumer is `Run`. Nil is never called and is behaviorally a decline; nil vs installed `NoOpFailoverPolicy` MUST produce byte-identical streams (the inertness pin).
- Method name `Resolve` mirrors `PermissionPolicy` (`permission_protocol.go:80-94`).

### AD-8 — Composed-ceiling divergence detection (`R-AIS-044 / S-2`)

**Choice**: a Layer 2 test reads `ai/internal/retry/doc.go` as a **file** via a repo-relative path (reading is not importing; the file stays byte-unchanged per `R-RUN-012`) and asserts (1) the cited Layer 1 sentences are present verbatim (`DefaultMaxAttempts = 3`, "at most N+1 = 4 wire requests"), and (2) `retry_policy.go`'s own doc states the formula in the helper's wording with `H × 4`. Perturbing either side's wording fails the test — divergence is observable as a test failure, not a comment. Precedent for repo-relative test reads: the substrate guards already shell out to git from `agent_test` (`loop_hook_test.go:812`).

## Data Flow

```
Turn fails (terr) ──► G0 cause check (:460, unchanged) ──► windDownRun
                        │ not cancelled
                        ▼
             retryDecision(terr, attempt, H)      [pure, G1–G5, first match]
        surface ◄───────┼───────► retry (attempt < H)
           │            │            │  backoff: retry-after ▷ computed, select on runCtx
           │      exhausted (==H)    │  sentinel abort ──► windDownRun
           │            │            └──► Turn again, same transcript slice
           │            ▼
           │   Failover.Resolve(ctx, prompt)  — once; verdict declines
           ▼            │
   failRun ◄────────────┘  → typedHarnessFailureFromError → run_end(Failed, TRUE evidence)
```

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/retry_policy.go` | Create | `retryDecision` + verdict vocabulary, `RetryTiming` + defaults + backoff + context sleep, `defaultRetryAttempts`, package doc with the composed ceiling citing `ai/internal/retry/doc.go` |
| `backend/agent/src/agent/failover_policy.go` | Create | `FailoverPolicy`, `FailoverPrompt`, `FailoverVerdict`, `NoOpFailoverPolicy` |
| `backend/agent/src/agent/loop.go` | Modify | AD-1: `preStreamAbortFailure` + emissions on the three paths |
| `backend/agent/src/agent/harness.go` | Modify | AD-2 inner attempt loop, AD-4 sibling, new `Harness` fields `RetryAttempts` / `RetryTiming` / `Failover` |
| `backend/agent/src/agent/retry_policy_test.go` | Create | predicate table-driven unit tests, AG-15.1 scenarios, bite (a), D1 stream-validity + bite (b) |
| `backend/agent/src/agent/retry_backoff_test.go` | Create | AG-15.2 scenarios, ceiling count test, divergence test + bite (c) |
| `backend/agent/src/agent/failover_policy_test.go` | Create | AG-15.3 scenarios, inertness pin |
| `loop_test.go` (`filterOutLoopFiles`, `:831`) / `loop_hook_test.go` (`filterOutLoopHookFiles`, `:907`) | Modify | Append **five** exact-suffix entries after the AG-14 block, byte-in-sync across both filters: `/retry_policy.go`, `/failover_policy.go`, `/retry_policy_test.go`, `/retry_backoff_test.go`, `/failover_policy_test.go`. No wildcard/prefix/directory pattern. `harness.go`/`harness_test.go` are already listed (AG-13 widening, `loop_test.go:902-903`). |

NOT touched: everything the proposal's Affected Areas table pins — all of `backend/agent/src/ai/**`, all `R-LSK-004` substrate, `go.mod`/`go.sum`; zero new `EventKind` (25 stays 25).

## Existing tests observing a pre-stream failure path (obligation 3 — exhaustive enumeration)

Method: opened both custom test providers (`grep 'func .* Stream('` over `agent/*_test.go` returns exactly `ctxRecordingProvider` at `loop_test.go:571` and `errorProvider` at `loop_test.go:1410`), all `PreStreamFailure` test references, and all `run_end`-failure assertions.

| Path | Tests driving it | What they assert | Impact of D1 |
|---|---|---|---|
| Build error (`loop.go:304-308`) | **None.** No test invokes `Turn` with inputs failing `buildLoopRequest` | — | none |
| Hook error (`loop.go:317-328`) | `TestTurn_PreRequestHook_FailureAbortsBeforeStream`, `loop_hook_test.go:399-441` (S-PRH-003) | 0 captured requests; sink non-empty; error category `UnsupportedCapability` | **green byte-unchanged** — asserts properties, not sequences; one extra event satisfies "non-empty" a fortiori |
| `provider.Stream` error (`loop.go:332-338`) | `TestTurn_ProviderPreStreamFailureSurfacesOnReturn`, `loop_test.go:1436-1461` (nil path, via `errorProvider` `:1408-1421`) | non-nil error; sink non-empty | **green byte-unchanged** — same reason; nil-path `run_end(Failed)` addition also satisfies it |
| Harness-level pre-stream failure | **None.** `errorProvider` is referenced only in `loop_test.go`; no zero-script `agenttest.NewProvider()` exists in any agent test; S-RUN-100 (`harness_test.go:1870-1919`) is the mid-stream fixture (G2 catches it, file byte-unchanged); cancellation tests reach `Turn`'s post-loop branch (`loop.go:442-450`) via mid-stream gate holds, never the pre-stream paths | — | none |

**Conclusion**: zero existing test files require amendment for D1. `sdd-verify` re-checks this table against the shipped tree; any test edit outside the three new files is an enumerated amendment needing sign-off.

## Testing Strategy (seven charter scenarios → concrete tests, plus bites)

Fixture note: `agenttest.Script` cannot fail pre-stream (`Stream` errors only on zero request / cancelled ctx / exhaustion, `fake_provider.go:74-117`). Design-owned fixture: a test-local `preStreamFailingProvider` (the `errorProvider` precedent, `loop_test.go:1408-1421`) that fails its first N calls with a scripted retryable `*ai.PreStreamFailure`, **captures its own requests**, then delegates to an inner `agenttest.Provider`. "Fail pre-stream forever" = N > H, asserting the wrapper's count `== H` exactly — the stronger proof. Zero `time.Sleep` synchronization anywhere (NFR-RUN-002/NFR-CAN-002): all waits are recorded `SleepFunc` injections or channel/Gate reads.

| # | Scenario (charter) | Test (file) | Strategy |
|---|---|---|---|
| 1 | AG-15.1 retryable-no-output retries visibly | `retry_policy_test.go` | wrapper fails H−1 pre-stream retryable, then inner success script; recording `SleepFunc` (returns nil). Assert: wrapper count == H; all captured requests `ai.Request.Equal`-identical; stream carries H turn pairs (H−1 `turn_end(Aborted)` + 1 finished) with distinct `TurnID`s on one contiguous lane + `run_end(Completed)`; **`CheckStream` accepts the recorded stream unmodified** |
| 2 | AG-15.1 partial output forbids retry | `retry_policy_test.go` | script: text delta then terminal `ErrorEvent` with `MidStreamFailure({Retryable: true}, outputPreceded=true)`; second script queued, never consumed. Assert `len(provider.Requests()) == 1`; `turn_end` failure has `PartialOutput() == true` with partial content; run ends Failed |
| 3 | AG-15.1 non-retryable surfaces immediately | `retry_policy_test.go` | terminal failure `Retryable: false` in both positions (pre-stream via wrapper, post-output via script). Assert 1 request; recording `SleepFunc` called zero times |
| 4 | AG-15.2 retry-after wins, interrupt aborts backoff | `retry_backoff_test.go` | (a) failure carrying `RetryAfter`; recording `SleepFunc` asserts requested delay == retry-after, not computed backoff. (b) `SleepFunc` that signals "sleeping" on a channel then blocks on `ctx.Done()`; test calls `h.Interrupt()`; assert `run_end(Interrupted)` with **nil** failure via `windDownRun`. Plus a direct unit test of the default sleep with a pre-cancelled ctx (immediate return) |
| 5 | AG-15.2 harness bound holds; ceiling documented | `retry_backoff_test.go` | wrapper scripted to fail H+2 times; assert wrapper count == H exactly and inner provider untouched; AD-8's divergence test pins both layers' wording |
| 6 | AG-15.3 seam consulted before giving up | `failover_policy_test.go` | recording `FailoverPolicy` installed; exhaust retries; assert `Resolve` called exactly once with `Attempts == H` and the final `*ai.Failure`; `run_end(Failed)` carries the true category (AD-4) |
| 7 | AG-15.3 inertness pin | `failover_policy_test.go` | identical failing fixture run twice — `Failover` nil vs `NoOpFailoverPolicy{}` — assert event streams identical (kind/outcome/sequence/failure evidence) and identical returned error |

Also: table-driven unit test of `retryDecision` over every gate and the first-match ordering (G2 before G3 proven by a retryable+partial case), and the AD-4 pointer-identity assertion.

**Bites (RED-recorded before GREEN, then reverted):** (a) delete G3 from `retryDecision` → scenario 2 retries (`Requests() == 2`) and fails; (b) revert AD-1's emission → scenario 1's `CheckStream` rejects with `ai.ErrMisplaced` (`stream_check.go:141-143`) and fails; (c) perturb the cited `doc.go` wording locally → AD-8's test fails.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. (AD-8's test reads one repo file read-only; the existing substrate guards already established that surface.)

## Migration / Rollout

No migration. Single-revert rollback per proposal; all three new `Harness` fields are zero-value inert, so an unconfigured harness behaves as designed by default (H = 3, production timing, no failover).

## Open Questions

None blocking. Deliberate deferrals stand as proposed: no `agent.Failure.RetryAfter()` (D5), no Layer 1 edits, no new `EventKind`.
