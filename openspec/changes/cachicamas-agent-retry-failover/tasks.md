# Tasks: AG-15 — Retry policy and the failover seam

> Change: `cachicamas-agent-retry-failover`. Binding inputs: `proposal.md`, `design.md` (AD-1…AD-8), `specs/agent-retry-failover/spec.md` (`R-RTY-001`…`012`, `S-RTY-001`…`015`). Strict TDD: every behavior task is RED-first (`cd backend/agent && make test`), RED output recorded in `apply-progress.md` before GREEN.

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (Go) | 1110–1860 |
| Estimated changed lines (SDD markdown, already mostly authored) | 850–1300 |
| Estimated total | 1960–3160 |
| 400-line budget risk | High |
| Chained PRs recommended | No — `size:exception` pre-authorized |
| Suggested split (if ever needed) | U1 (D1) → U2 (AG-15.1) → {U3 (AG-15.2), U4 (AG-15.3)} |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| U1 | D1: pre-stream `turn_end(Aborted)` on all 3 paths | PR 1 (single-pr) | `go test -race -run TestTurn_PreStream ./...` | `make test` full suite, byte-unchanged existing files | `loop.go` 3 emission sites + `preStreamAbortFailure` helper |
| U2 | AG-15.1 predicate + inner attempt loop + G3 bite | PR 1 | `go test -race -run TestRetryDecision\|TestHarness_Retry ./...` | `make test` | `retry_policy.go`, `retry_policy_test.go`, `harness.go` attempt-loop block |
| U3 | AG-15.2 timing/backoff/ceiling + divergence bite | PR 1 | `go test -race -run TestHarness_Backoff\|TestComposedCeiling ./...` | `make test` | `retry_backoff_test.go`, `RetryTiming` in `retry_policy.go` |
| U4 | AG-15.3 failover seam + inertness pin | PR 1 | `go test -race -run TestFailover ./...` | `make test` | `failover_policy.go`, `failover_policy_test.go` |

## Exact new/modified files (obligation 1 — fixes design.md's table)

| File | Action | Hosts |
|---|---|---|
| `backend/agent/src/agent/retry_policy.go` | Create | `retryDecision`, verdict vocabulary, `defaultRetryAttempts=3`, `RetryTiming`+defaults+backoff, package doc with `H × 4 = 12` ceiling |
| `backend/agent/src/agent/failover_policy.go` | Create | `FailoverPolicy`, `FailoverPrompt`, `FailoverVerdict`, `NoOpFailoverPolicy` |
| `backend/agent/src/agent/retry_policy_test.go` | Create | `S-RTY-001` predicate table, `S-RTY-002/003/004`, D1 stream-validity, bites `S-RTY-010`/`S-RTY-011` |
| `backend/agent/src/agent/retry_backoff_test.go` | Create | `S-RTY-005`…`009`, bite `S-RTY-012` |
| `backend/agent/src/agent/failover_policy_test.go` | Create | `S-RTY-013`, `S-RTY-014` |
| `backend/agent/src/agent/loop.go` | Modify | `preStreamAbortFailure` + 3 emission sites (D1) |
| `backend/agent/src/agent/harness.go` | Modify | inner attempt loop, `typedHarnessFailureFromError`, new `RetryAttempts`/`RetryTiming`/`Failover` fields |
| `backend/agent/src/agent/loop_test.go` (`filterOutLoopFiles`) | Modify | append 5 suffixes |
| `backend/agent/src/agent/loop_hook_test.go` (`filterOutLoopHookFiles`) | Modify | append 5 suffixes, byte-in-sync |

## Substrate filter drift — verified locations, DO NOT trust proposal's cited ranges

Verified in this phase against `main@bf482b0a`:

- `loop_test.go`: `filterOutLoopFiles` starts `:831` (matches proposal). The AG-14 block's **last** suffix is `/cancellation_events_test.go` at **`:925`**, immediately followed by the closing `}` of the `skip =` expression at `:926` — **not** `:871` as the proposal cited.
- `loop_hook_test.go`: `filterOutLoopHookFiles` starts `:907` (matches proposal). The AG-14 block's **last** suffix is `/cancellation_events_test.go` at **`:996`**, closing `}` at `:997` — **not** `:943`.

**Task 0.1**: `sdd-apply` MUST re-grep both functions for their own AG-14 tail (`strings.HasSuffix(path, "/cancellation_events_test.go")`) at apply time rather than trust any line number cited anywhere, since AG-16 (parallel) may also touch these files. Append the 5 new suffixes (`/retry_policy.go`, `/failover_policy.go`, `/retry_policy_test.go`, `/retry_backoff_test.go`, `/failover_policy_test.go`) as a new commented block ("AG-15 widening") directly after the located AG-14 tail, in **both** files, byte-in-sync, in the same commit as each new file's first introduction (AG-13 precedent, `loop_test.go:898-901`). No wildcard/prefix/directory pattern.

## Phase 0 — D1: close the pre-stream bracket gap (BLOCKER — must land before any retry test)

- [x] 0.1 RED: write `TestTurn_PreStreamBuildErrorEmitsTurnEnd`, `TestTurn_PreStreamHookErrorEmitsTurnEnd`, `TestTurn_PreStreamProviderErrorEmitsTurnEnd` in `retry_policy_test.go` — nil-path asserts `turn_end(Aborted)` + `run_end(Failed)`; continuation-path variant asserts `turn_end(Aborted)` only, no `run_end`. Record RED.
- [x] 0.2 GREEN: add `preStreamAbortFailure(err error) (*Failure, error)` to `loop.go` (`errors.As` → `NewFailure`; else `ai.PreStreamFailure(FailureCategoryUnavailable)` → `NewFailure`); call it + emit `turn_end`/`run_end` at all 3 sites (`:304-308`, `:317-328`, `:332-338`) before `closeSink`.
- [x] 0.3 Confirm byte-unchanged: `TestTurn_PreRequestHook_FailureAbortsBeforeStream` (`loop_hook_test.go:399-441`) and `TestTurn_ProviderPreStreamFailureSurfacesOnReturn` (`loop_test.go:1436-1461`) stay green with zero edits (design's enumeration table).
- [x] 0.4 Widen both substrate filters per "Substrate filter drift" above, adding only `/retry_policy.go` + `/retry_policy_test.go` in this commit (first files introduced).
- [x] 0.5 Bite `S-RTY-011` (RED-first, then revert): temporarily revert 0.2's emission, run `S-RTY-002` (added in Phase 1, so this bite is executed logically after 1.2 exists — sequence note below), confirm `CheckStream` rejects with `ai.ErrMisplaced` at `stream_check.go:141-143`. Record RED, revert. **Executed at task 1.8** per this note; see apply-progress.md for the RED evidence.

Commit 1 (`feat(agent): emit turn_end on Turn's pre-stream failure paths`): 0.1–0.4.

## Phase 1 — AG-15.1: the retry predicate and inner attempt loop

- [x] 1.1 RED: `TestRetryDecision` table test in `retry_policy_test.go` covering G0–G5 first-match-wins, incl. retryable+partial → G3 (not G4), non-retryable+partial → G2 (not G3), plain error → G1 (`S-RTY-001`). Record RED. **Amendment**: lives in new `retry_decision_internal_test.go` (`package agent`), not `retry_policy_test.go` — see apply-progress.md's "Deviations" section for why (NFR-RTY-001's own carve-out, forced by Go's unexported-symbol visibility rule).
- [x] 1.2 GREEN: add `retryDecision(terr error, attempt, bound int) retryVerdict` + `verdictSurface`/`verdictRetry`/`verdictExhausted` to `retry_policy.go`.
- [x] 1.3 RED: `TestHarness_RetryVisibleAttempts` (`S-RTY-002`, charter AG-15.1 sc.1) using a test-local `preStreamFailingProvider` wrapper (see Phase 1a). Record RED.
- [x] 1.4 GREEN: wrap `harness.go:447`'s `Turn` call in an inner attempt loop (fresh `turnSink`+forwarder per attempt); add `Harness.RetryAttempts int` (default 3 via `<=0` check); insert the gate between `:462` and `:469`, G0 unchanged.
- [x] 1.5 RED: `TestHarness_RetryPartialOutputNotRetried` (`S-RTY-003`, sc.2) and `TestHarness_RetryNonRetryableSurfacesImmediately` (`S-RTY-004`, sc.3, both positions). **No RED observed** — both passed immediately (see 1.6).
- [x] 1.6 GREEN: confirmed G3/G2 already satisfy 1.5 with no further production change; no gap found.
- [x] 1.7 Bite `S-RTY-010` (RED-first, then revert): delete G3 from `retryDecision`, re-run `S-RTY-003`, confirm it fails with `provider.Requests()` length `> 1`. Record RED, revert. Done — see apply-progress.md.
- [x] 1.8 Run Phase 0's `S-RTY-011` bite now that `S-RTY-002` exists: revert 0.2's emission, confirm `S-RTY-002` fails with `ai.ErrMisplaced`. Record RED, revert. Done — see apply-progress.md.
- [x] 1.9 Widen both substrate filters, adding `/retry_backoff_test.go`'s siblings is deferred to Phase 2; this commit needs no further filter change for `retry_policy.go`/`retry_policy_test.go` (already listed in 0.4). **Additional amendment**: widened both filters for `retry_decision_internal_test.go` (task 1.1's amendment), byte-in-sync, same commit.
- [x] 1.10 Confirmed `S-RUN-100` (`harness_test.go:1870-1919`) stays green, file byte-unchanged — G2 catches its unset `Retryable`, verified against the shipped tree.

### Phase 1a — test-local provider wrapper (design-owned fixture)

- [x] 1a.1 In `retry_policy_test.go`, define `preStreamFailingProvider` (the `errorProvider` precedent, `loop_test.go:1408-1421`): fails its first N `Stream` calls with a scripted retryable `*ai.PreStreamFailure`, captures its own requests, then delegates to an inner `agenttest.Provider`. **Test-local only — do not widen `agenttest`.**

Commit 2 (`feat(agent): retry a pre-output failure across an inner attempt loop`): 1.1–1.8, 1a.1.

## Phase 2 — AG-15.2: timing seam, backoff, retry-after, ceiling

- [x] 2.1 RED: `TestRetryTiming_BackoffJitterAndClamp`, `TestHarness_RetryAfterOverridesBackoff` (`S-RTY-007`, sc.1 first Then), `TestHarness_InterruptAbortsBackoff` (`S-RTY-008`, sc.1 second Then), `TestHarness_BoundHoldsAboveH` (`S-RTY-005`, sc.2 first Then). All in `retry_backoff_test.go`, all synchronized via injected `SleepFunc`/channels, zero `time.Sleep`. Record RED.
- [x] 2.2 GREEN: add `RetryTiming{NowFunc, SleepFunc, BaseDelay, MaxDelay, JitterSeed}` + `applyDefaults` + `computeBackoff` to `retry_policy.go`; add `Harness.RetryTiming` field; wire backoff wait on `runCtx` at G4, re-check `context.Cause(runCtx)` on sleep error → `windDownRun` for sentinels, fall through otherwise.
- [x] 2.3 RED: `TestComposedCeiling_MatchesLayer1Wording` (`S-RTY-009`, sc.2 second Then) reading `ai/internal/retry/doc.go` at a repo-relative path, asserting both the Layer 1 sentences and `retry_policy.go`'s doc wording. Record RED.
- [x] 2.4 GREEN: write `retry_policy.go`'s package doc stating `H` counts total attempts, `H × 4 = 12` at defaults, citing `ai/internal/retry/doc.go` verbatim.
- [x] 2.5 Bite `S-RTY-012` (RED-first, then revert): perturb the cited wording in a scratch copy / via test-only string substitution proving detection — **never edit the real `ai/internal/retry/doc.go`** — confirm `S-RTY-009` fails. Record RED, revert (leave `ai/internal/retry/doc.go` byte-unchanged against merge base).
- [x] 2.6 Widen both substrate filters, adding `/retry_backoff_test.go`.
- [x] 2.7 Grep new test files for `time.Sleep`; confirm zero hits (`NFR-RTY-002`).

Commit 3 (`feat(agent): inject retry timing, honor retry-after, state the composed ceiling`): 2.1–2.6.

## Phase 3 — AG-15.3: the failover seam (parallel with Phase 2 after Phase 1 lands)

- [x] 3.1 RED: `TestFailover_ConsultedOnceAtExhaustion` (`S-RTY-013`, sc.1) with a recording `FailoverPolicy`; `TestFailover_InertnessNilVsNoOp` (`S-RTY-014`, sc.2) running one fixture twice. Record RED.
- [x] 3.2 GREEN: add `FailoverPolicy` interface, `FailoverPrompt{Attempts int; Failure *ai.Failure}`, `FailoverVerdict{}` (zero-value declines), `NoOpFailoverPolicy{}` to `failover_policy.go`; add `Harness.Failover FailoverPolicy` (nil-default); consult at G5 before the terminal report.
- [x] 3.3 Widen both substrate filters, adding `/failover_policy.go` + `/failover_policy_test.go`.

Commit 4 (`feat(agent): consult the failover seam once at retry exhaustion`): 3.1–3.3.

## Phase 4 — Decision 4: preserve the true category on the exhausted-retry report

- [ ] 4.1 RED: `TestHarness_ExhaustedRetryPreservesTrueCategory` and `TestHarness_ExhaustedRetryPlainErrorStaysUnavailable` (`S-RTY-015`), asserting pointer-identity via `runEnd.Failure().Unwrap()`. Record RED.
- [ ] 4.2 GREEN: add `typedHarnessFailureFromError(cause error) (*Failure, error)` sibling beside `wrapHarnessFailure` (`harness.go:250-259`, AG-14 sibling shape); route `failRun` through it; plain-error arm calls the untouched `wrapHarnessFailure` byte-identically.
- [ ] 4.3 Grep confirms no existing test pins `Unavailable` on the harness `run_end` (per design's enumeration: `harness_test.go:1898` presence-only; cancellation tests assert absence).

Commit 5 (`fix(agent): preserve the true failure category on an exhausted-retry run_end`): 4.1–4.3.

## Phase 5 — Documentation (part of the deliverable)

- [ ] 5.1 `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:2172` — tick `- [x] Partial-output failures are never silently retried; retry attempts are visible events — closed by AG-15.1.`
- [ ] 5.2 Same doc, line 3 status header: extend "Wave 3 opens with AG-12…AG-14" prose to include AG-15, following AG-13/AG-14's narrative pattern (what AG-15 closes: G8's Layer 2 half, the retry predicate, backoff, the failover seam).
- [ ] 5.3 Same doc: bump the milestone counter from **14 of 24** to **15 of 24** in the status line.
- [ ] 5.4 R-15/G8 back-annotation: no separate checkbox exists for R-15 (line 69 is a static row); the checklist tick at 5.1 is the closing annotation — confirm no other `- [ ]` in the doc names AG-15 as its closer besides `:2172`.
- [ ] 5.5 Composed-ceiling documentation (H × 4 = 12) lives in `retry_policy.go`'s package doc (task 2.4) — this is "where both layers' readers find it" per the charter's `0003:1455` note, which is already satisfied by the shipped `ai/internal/retry/doc.go` half; no charter text edit needed.
- [ ] 5.6 Flag for `sdd-verify`/archive: `specs/agent-loop-skeleton/spec.md` carries an unusual `## MODIFIED Header — the allocated scenario range` section (`:10-12`, extending `S-LSK-020` → `S-LSK-023`) — confirm the archive step applies this header edit, not only the `MODIFIED Requirements` blocks.

Commit 6 (`docs(0003): tick AG-15, bump milestone counter to 15/24`): 5.1–5.3, 5.6.

## Coverage table — every scenario has a task

| Scenario | Task(s) |
|---|---|
| S-RTY-001 | 1.1–1.2 |
| S-RTY-002 | 1.3–1.4, 1a.1 |
| S-RTY-003 | 1.5–1.6 |
| S-RTY-004 | 1.5–1.6 |
| S-RTY-005 | 2.1–2.2 |
| S-RTY-006 | 2.1–2.2, 2.7 |
| S-RTY-007 | 2.1–2.2 |
| S-RTY-008 | 2.1–2.2 |
| S-RTY-009 | 2.3–2.4 |
| S-RTY-010 (bite) | 1.7 |
| S-RTY-011 (bite) | 0.5 / 1.8 |
| S-RTY-012 (bite) | 2.5 |
| S-RTY-013 | 3.1–3.2 |
| S-RTY-014 | 3.1–3.2 |
| S-RTY-015 | 4.1–4.2 |
| D1 (pre-stream `turn_end`, both paths) | 0.1–0.3 |
| Both substrate filters, exact-filename, byte-in-sync | 0.4, 1.9, 2.6, 3.3 |
| Composed ceiling + divergence detection | 2.3–2.5 |
| Doc 0003 checklist / counter / MODIFIED-header flag | 5.1–5.6 |

## Known risks carried forward

- `S-RUN-100`'s "exactly 1 request" pin: `sdd-verify` MUST re-check `Retryable` stays unset in `loop_test.go:1266-1268` against the shipped tree (task 1.10), not trust this sentence.
- `agent-loop-skeleton`'s delta has a `## MODIFIED Header` section outside the usual `MODIFIED Requirements` shape (task 5.6) — archive must not drop it.
- AG-16 runs parallel and also edits `harness.go`'s turn-invocation block (the inner attempt loop this change adds) — the merging orchestrator resolves; flagged in proposal risk 11.
