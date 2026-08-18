# Apply Progress: AG-15 — Retry policy and the failover seam

> Change: `cachicamas-agent-retry-failover`. Worktree `cachicamas-worktrees/ag-15-retry-failover`, branch `feat/agent-layer2-wave3-ag15`, base `main@bf482b0a`.
> **Batch 1 (this batch): Phase 0 + Phase 1 + Phase 1a only.** Phases 2–5 NOT started.
> Strict TDD active. Every RED below is the actual observed output of a real `go test` run against the shipped tree at that moment — none are predicted or paraphrased.

## Commits this batch

| SHA | Message | Tasks |
|---|---|---|
| `288aa154` | `feat(agent): emit turn_end on Turn's pre-stream failure paths` | 0.1–0.4 |
| `cc1cab86` | `feat(agent): retry a pre-output failure across an inner attempt loop` | 1.1–1.10, 1a.1 (and 0.5's bite, executed at 1.8 per tasks.md's own sequencing note) |

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
- [x] **0.4** Widened both substrate filters (`filterOutLoopFiles` at `loop_test.go:831`, `filterOutLoopHookFiles` at `loop_hook_test.go:907`) with `/retry_policy.go` and `/retry_policy_test.go`, appended immediately after the re-grepped AG-14 tail (verified at `loop_test.go:925`/`:926` and `loop_hook_test.go:996`/`:997` — matching the orchestrator's own re-verified locations, not the proposal's stale citation). Byte-in-sync between both filters.
- [x] **0.5** Bite `S-RTY-011` — **deferred and executed at task 1.8** per tasks.md's own sequencing note ("this bite is executed logically after 1.2 exists"). See 1.8 below for the RED evidence and revert.

## Phase 1 — AG-15.1: the retry predicate and inner attempt loop

- [x] **1.1** RED: `TestRetryDecision` (table test, G1–G5) + `TestRetryDecision_SameEvidenceRetriesThenExhausts` in **`retry_decision_internal_test.go`** (see "Deviation: predicate test file split" below). RED observed (compile failure — `retryDecision`/`retryVerdict`/verdict constants did not exist yet):
  ```
  src/agent/retry_decision_internal_test.go:72:11: undefined: retryVerdict
  src/agent/retry_decision_internal_test.go:78:13: undefined: verdictSurface
  ...
  src/agent/retry_decision_internal_test.go:121:14: undefined: retryDecision
  FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
  ```
- [x] **1.2** GREEN: added `retryDecision(terr error, attempt, bound int) retryVerdict` + `verdictSurface`/`verdictRetry`/`verdictExhausted` + `defaultRetryAttempts = 3` to new `retry_policy.go`. Re-run: `TestRetryDecision` (7 subtests) + the same-evidence test all PASS.
- [x] **1.3** RED: `TestHarness_RetryVisibleAttempts` (`S-RTY-002`) written using the new `preStreamFailingProvider` wrapper (1a.1). RED observed (compile failure against the pre-1.4 `harness.go`, deliberately reverted first so this RED is genuine — see "TDD ordering correction" below):
  ```
  src/agent/retry_policy_test.go:374:3: unknown field RetryAttempts in struct literal of type agent.Harness
  FAIL	github.com/cachicamas/backend/agent/src/agent [build failed]
  ```
- [x] **1.4** GREEN: wrapped `harness.go`'s `Turn` call in an inner attempt loop (fresh `turnSink`+forwarder per attempt, same `transcript` slice reused by reference); added `Harness.RetryAttempts int` (`<=0` → `defaultRetryAttempts`); inserted the `retryDecision` gate between the existing G0 cancellation check and the fall-through to `failRun`, G0 unchanged. Re-run: `TestHarness_RetryVisibleAttempts` PASSES.
- [x] **1.5** RED/confirm: `TestHarness_RetryPartialOutputNotRetried` (`S-RTY-003`) and `TestHarness_RetryNonRetryableSurfacesImmediately` (`S-RTY-004`, both positions: pre-stream via wrapper, mid-stream via script) written and run. **No RED observed** — both PASSED on first run. This matches task 1.6's own framing exactly (see 1.6).
- [x] **1.6** Confirmed: G2/G3 (already implemented at 1.2, already exercised end-to-end once the inner loop landed at 1.4) satisfy 1.5's scenarios with **zero further production change**. No gap found.
- [x] **1.7** Bite `S-RTY-010` — RED-first, then reverted. Mutation: deleted the `if failure.PartialOutput() { return verdictSurface }` line from `retryDecision` (`retry_policy.go`). Re-ran `TestHarness_RetryPartialOutputNotRetried`. RED observed:
  ```
  --- FAIL: TestHarness_RetryPartialOutputNotRetried (0.00s)
      retry_policy_test.go:526: Run returned err = nil, want the mid-stream failure (G3 forbids retry; the run must end failed)
  FAIL
  ```
  Root cause confirmed by code trace: with G3 deleted, the retryable+partial failure falls to G4 (`attempt=1 < bound=3`) → retries → the queued second (normally-completing) script is consumed → `Run` succeeds. A successful `Run` here is only reachable by consuming **both** queued scripts, i.e. `provider.Requests()` grows to 2 — the exact "more than one request" defect `S-RTY-010` targets, reached via the test's `err == nil` assertion (the test's first check) rather than its later explicit count check, since `t.Fatal` stops there. **Reverted**: `git diff` on `retry_policy.go` after revert is empty (confirmed byte-identical to the GREEN state). Re-ran: PASS.
- [x] **1.8** Bite `S-RTY-011` (Phase 0's 0.5, executed here per the sequencing note) — RED-first, then reverted. Mutation: `emitPreStreamAbort` in `loop.go` made an unconditional no-op (`return` as its first statement). Re-ran `TestHarness_RetryVisibleAttempts` (`S-RTY-002`). RED observed:
  ```
  --- FAIL: TestHarness_RetryVisibleAttempts (0.00s)
      retry_policy_test.go:443: turn brackets: 3 turn_start, 1 turn_end, want exactly 3 each (H -- one bracket per attempt)
      retry_policy_test.go:446: aborted turn_end count = 0, want 2 (H-1)
      retry_policy_test.go:457: CheckStream rejected the recorded multi-attempt stream: event[2]: value is not permitted where it appears, want it accepted unmodified
  FAIL
  ```
  `"value is not permitted where it appears"` is confirmed (via `grep` on `src/ai/validation.go:123`) to be `ai.ErrMisplaced`'s exact `.Error()` string — matching the design's precise prediction (`stream_check.go:141-143`, second `turn_start` on an already-open turn bracket). **Reverted**: `git diff` on `loop.go` after revert is empty (confirmed byte-identical to the Phase 0 commit). Re-ran: PASS.
- [x] **1.9** No further filter change needed for `retry_policy.go`/`retry_policy_test.go` (already done at 0.4). **Additional, enumerated amendment**: widened both filters again for `retry_decision_internal_test.go` (see deviation note below), landed in this same commit.
- [x] **1.10** Confirmed `S-RUN-100` (`TestHarness_TurnError_RunEndsTypedNoAppendNoCloseNoRetry`, `harness_test.go:1870`) stays green, **source byte-unchanged** (`git diff --stat main -- harness_test.go` is empty). Verified against the shipped tree, not trusted from the sentence: its fixture (`scriptTextThenMidStreamError`, `loop_test.go:1266-1268`) constructs `ai.FailureReport{Category: ai.FailureCategoryUnavailable}` with **no `Retryable` field set** — Go zero-value `false` — so G2 catches it, exactly as design claimed. Ran the test standalone: PASS.
- [x] **1a.1** `preStreamFailingProvider` defined in `retry_policy_test.go` (the `errorProvider` precedent, `loop_test.go:1408-1421`): fails its first `failCount` `Stream` calls with a scripted `*ai.Failure`, captures its own requests independently of the inner provider, delegates afterward. Test-local only — `agenttest` untouched (confirmed: zero diff against main for anything under `src/agenttest/`).

## Deviations from design (both enumerated, both forced by Go language constraints, not by choice)

1. **Predicate test file split (`retry_decision_internal_test.go`).** Design AD-2 names `retryDecision`/`retryVerdict` as unexported identifiers (lowercase), and this codebase's test convention is **100% `package agent_test`** across every one of its 33 existing test files (verified by grep — zero exceptions before this change). `NFR-RTY-001` itself anticipates the resulting tension and grants an explicit carve-out: *"The predicate's own table-driven test MAY exercise it through whatever surface sdd-design fixed, provided every behavioral claim above is also observable externally."* Since Go enforces package-per-**file** (not per-directory) and unexported symbols are categorically unreachable from `agent_test`, `S-RTY-001`'s directly-callable table test cannot physically live in the same file as `retry_policy_test.go`'s other (externally-driven) scenarios. I created one small additional file, `retry_decision_internal_test.go` (`package agent`), hosting only `TestRetryDecision` and its same-evidence companion. Every other requirement this predicate serves is independently and externally proven by `retry_policy_test.go`'s `Harness.Run`-driven scenarios, satisfying `NFR-RTY-001`'s proviso. Both substrate filters were widened with this file's exact suffix, byte-in-sync, landed in the same commit that introduces it (task 1.9's note above).
2. **`emitPreStreamAbort` helper (loop.go), beyond the one design names.** Design AD-1 names exactly one helper, `preStreamAbortFailure(err error) (*Failure, error)` — implemented verbatim under that exact name and signature. I additionally factored the (turn_end + conditional run_end) emission shared by all three call sites into one small private helper, `emitPreStreamAbort`, rather than repeating ~10 lines inline three times as the pre-existing mid-stream block does. This is a DRY implementation choice, not a behavioral deviation — `preStreamAbortFailure` exists exactly as named and is the one function `emitPreStreamAbort` delegates to.

## TDD ordering correction (recorded, not hidden)

While implementing task 1.4, I initially wrote the inner-attempt-loop production code in `harness.go` **before** running task 1.3's test against the pre-1.4 tree — a strict-TDD ordering violation caught before it produced any false evidence. Corrected by: `git checkout -- harness.go` (safe — uncommitted, and confirmed empty diff afterward against the true pre-1.4 state), writing and running `TestHarness_RetryVisibleAttempts` to get a genuine RED against the reverted file (`unknown field RetryAttempts`), then reapplying the exact same GREEN implementation. No fabricated evidence entered this record.

## TDD Cycle Evidence

| Task | Test | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 0.1–0.2 | `TestTurn_PreStreamBuildErrorEmitsTurnEnd` | External (agent_test) | N/A (new file) | Written, observed (6 subtests) | Passed | 3 paths × 2 continuation modes = 6 cases | Shared `assertPreStreamAbortSequence`/`emitPreStreamAbort` extracted |
| 0.1–0.2 | `TestTurn_PreStreamHookErrorEmitsTurnEnd` | External | N/A | Written, observed | Passed | ➖ (part of the 6-case set above) | ➖ |
| 0.1–0.2 | `TestTurn_PreStreamProviderErrorEmitsTurnEnd` | External | N/A | Written, observed | Passed | ➖ | ➖ |
| 1.1–1.2 | `TestRetryDecision` (+ same-evidence test) | Internal (package agent) | N/A (new file) | Written, observed (compile fail) | Passed (7+2 subtests) | ✅ 9 cases covering G1–G5 and first-match ordering | Clean; no refactor needed |
| 1.3–1.4 | `TestHarness_RetryVisibleAttempts` | External | ✅ full package green pre-edit | Written, observed (compile fail, post-revert) | Passed | ➖ single scenario (S-RTY-002) | ➖ |
| 1.5–1.6 | `TestHarness_RetryPartialOutputNotRetried` | External | ✅ | Written, run — **no RED** (already satisfied, per task 1.6) | Passed | ➖ single scenario (S-RTY-003) | ➖ |
| 1.5–1.6 | `TestHarness_RetryNonRetryableSurfacesImmediately` | External | ✅ | Written, run — **no RED** (already satisfied) | Passed (2 subtests: pre-stream + mid-stream positions) | ✅ 2 positions | ➖ |
| 1.7 | Bite `S-RTY-010` | External | — | Written (mutation), observed | Reverted, confirmed clean | — | — |
| 1.8 | Bite `S-RTY-011` | External | — | Written (mutation), observed | Reverted, confirmed clean | — | — |

### Test Summary
- **Total test functions added**: 8 (3 D1 + `TestRetryDecision` + same-evidence + `TestHarness_RetryVisibleAttempts` + `TestHarness_RetryPartialOutputNotRetried` + `TestHarness_RetryNonRetryableSurfacesImmediately`), 19 total (sub)test cases across them, plus 2 bites executed and reverted.
- **Total tests passing**: all, per the fresh full-suite run below.
- **Layers used**: External (agent_test) for all behavioral scenarios; Internal (package agent) for the one predicate unit test NFR-RTY-001 carves out.
- **Approval tests**: none — no refactoring-of-existing-behavior tasks in this batch.
- **Pure functions created**: `retryDecision` (the whole point), `preStreamAbortFailure`.

## Work Unit Evidence

| Evidence | Phase 0 (D1) | Phase 1 (predicate + inner loop) |
|---|---|---|
| Focused test command / result | `go test -race -run 'TestTurn_PreStream' ./src/agent/...` → PASS (6/6 subtests) | `go test -race -run 'TestRetryDecision\|TestHarness_Retry' ./src/agent/...` → PASS |
| Runtime harness command / result | `go test -race ./src/agent/...` (full package) → PASS; `make test` (`go test -race -v -count=1 ./...`, fresh, no cache) → PASS, exit 0, 0 FAILs across all 12 module packages | same |
| Rollback boundary | `loop.go`'s 3 emission sites + `preStreamAbortFailure`/`emitPreStreamAbort`; both substrate filters' new suffix lines. Commit `288aa154` is independently revertable. | `retry_policy.go`, `retry_decision_internal_test.go`, `retry_policy_test.go`, `harness.go`'s attempt-loop block + `RetryAttempts` field; both substrate filters' new suffix lines. Commit `cc1cab86` is independently revertable (depends on `288aa154`). |

## Full suite verification (batch end)

`go test -race -v -count=1 ./...` from `backend/agent/` (byte-identical command to `make test`, run with `-count=1` to force a fresh, non-cached result):

```
ok  	github.com/cachicamas/backend/agent/src/agent	9.066s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.501s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep	1.934s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest	2.131s
ok  	github.com/cachicamas/backend/agent/src/ai	4.972s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	2.341s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	171.683s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest	2.863s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	3.325s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	6.822s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke	2.318s
ok  	github.com/cachicamas/backend/agent/src/handoff	2.047s
```
Zero `--- FAIL` lines anywhere in the log. Exit code 0. `go vet ./...` clean.

## Constraint compliance (verified, not assumed)

- `backend/agent/src/ai/**`: `git diff --stat main -- backend/agent/src/ai/` is empty. Untouched.
- `ai/internal/retry` never imported from `agent` package (not applicable yet this batch — `RetryTiming`, which would need this discipline, is Phase 2).
- `stream_check.go`, `stream_check_test.go`, `turn_events.go`, `failure.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `history.go`, `go.mod`, `go.sum`, `harness_test.go`: all confirmed byte-unchanged (`git diff --stat main --` on each is empty).
- 25 registered `EventKind`s: unchanged (no new kind registered by this batch).
- Substrate filters: byte-in-sync between `loop_test.go` and `loop_hook_test.go` at every widening step this batch made.
- Both bites (`S-RTY-010`, `S-RTY-011`) reverted and independently confirmed via empty `git diff` on the mutated file.
- `S-RUN-100`'s fixture: verified unset `Retryable`, source byte-unchanged (task 1.10).
- No `time.Sleep` in any test added this batch (grep-confirmed: no timing seam exists yet — Phase 2's scope).
- Worktree discipline: every write landed under `cachicamas-worktrees/ag-15-retry-failover`; `git status` on the base checkout (`/Users/braejan/workspace/witsaba/repositories/cachicamas`) is clean.

## Remaining tasks (NOT started this batch)

- [ ] Phase 2 (2.1–2.7): `RetryTiming` (timing seam), backoff/jitter/clamp, retry-after override, interrupt-during-backoff, composed-ceiling documentation + divergence test + bite `S-RTY-012`.
- [ ] Phase 3 (3.1–3.3): the failover seam (`FailoverPolicy`, `FailoverPrompt`, `FailoverVerdict`, `NoOpFailoverPolicy`), consulted at G5, inertness pin.
- [ ] Phase 4 (4.1–4.3): `typedHarnessFailureFromError` sibling — preserve the true category on an exhausted-retry `run_end`.
- [ ] Phase 5 (5.1–5.6): doc 0003 checklist tick, milestone counter bump, `agent-loop-skeleton` delta's `## MODIFIED Header` flag for `sdd-verify`/archive.

## Coverage table status (this batch's slice)

| Scenario | Task(s) | Status |
|---|---|---|
| D1 (both paths) | 0.1–0.3 | ✅ closed |
| Both substrate filters (0.4 slice) | 0.4 | ✅ closed for this batch's 3 files |
| S-RTY-001 | 1.1–1.2 | ✅ closed |
| S-RTY-002 | 1.3–1.4, 1a.1 | ✅ closed |
| S-RTY-003 | 1.5–1.6 | ✅ closed |
| S-RTY-004 | 1.5–1.6 | ✅ closed |
| S-RTY-010 (bite) | 1.7 | ✅ closed |
| S-RTY-011 (bite) | 0.5 / 1.8 | ✅ closed |
| S-RTY-005…009, 012 | Phase 2 | not started |
| S-RTY-013, 014 | Phase 3 | not started |
| S-RTY-015 | Phase 4 | not started |
| Doc 0003 checklist | Phase 5 | not started |
