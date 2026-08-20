# Apply progress: AG-20 — Complete the hook taxonomy

**Status**: 51/54 tasks complete. The remaining 3 (7.2, 7.3, 7.4) are explicitly
`sdd-archive`/`sdd-verify` obligations per their own task text, not apply's.

**Mode**: Strict TDD. Runner: `cd backend/agent && go test -race -count=1 ./...`.
Final evidence run: **174.14s real** (`2:54.14`), `-count=1`, uncached — matches
the ~170s baseline this project's memory records for a genuine (non-cached) run.

**Commits** (`feat/agent-layer2-wave5-ag20`):
1. `83362a26` `feat(agent): AG-20 -- complete the hook taxonomy production surface`
2. `dac4fc3f` `test(agent): AG-20 -- hook taxonomy scenario suite, all 49 scenarios`
3. `779a4c4b` `docs(0003): AG-20 -- tick hook taxonomy milestone, Wave 5 complete`
4. `591ce82d` `fix(agent): AG-20 -- S-LSK-032's own guard must allow hooks.go's diff`

## Deviation from the suggested U1/U2/U3 commit split

tasks.md's own "Suggested Work Units" table proposes three commits (U1 hooks.go
+ chain, U2 session-start + post-turn, U3 pre-compact + asynchrony). Discovered
during implementation: `hooks.go`'s observer-lane mechanics (`newObserverLane`,
`enqueue`, `drain`, `invokeRecovered`, `report`, `reportOutstanding`) are used by
BOTH session-start (U2) and post-turn (U2) AND the asynchrony suite (U3) — they
are not separable without leaving an intermediate commit non-compiling, which
violates work-unit-commits' own first checklist item ("the repo still makes
sense after applying only this commit"). Likewise `hooks_test.go`,
`hooks_harness_test.go` and `hooks_compaction_test.go` share helpers defined in
the first of the three (`noopPreRequestHook`, `wgObserverHold`, `kindsOfHKS`,
`mustNonRetryablePreStreamFailure`, etc.), so the three test files are not
independently committable either. Committed instead as: one whole production
commit, one whole test-suite commit, one docs commit, plus one small follow-up
fix commit for a bug the S-LSK-032 guard itself had (see below). Delivery
strategy is single-PR `exception-ok`, so this does not affect PR boundaries —
only the internal commit story.

## Size versus estimate

tasks.md's Review Workload Forecast estimated 1295–1755 counted lines
(excluding `openspec/`). Actual: **4641 lines** across
`compaction.go`(+52) `compaction_surgery_test.go`(+57) `harness.go`(+149)
`loop.go`(+96) `loop_hook_test.go`(+14) `loop_test.go`(+14) `hooks.go`(477)
`hooks_compaction_test.go`(505) `hooks_harness_test.go`(1071)
`hooks_test.go`(2206, later +7/-4 for the follow-up fix). The test files are
roughly 3.7× the estimate — driven by 49 independently-verifiable scenarios
each needing its own deterministic (no-wall-clock) fixture, several with their
own baseline-comparison run. `size:exception` was pre-accepted with "extend if
needed" language in the proposal/tasks headers; flagged here for visibility
since the actual size is well beyond even the pessimistic estimate.

## TDD Cycle Evidence (RED → GREEN, mandated bites)

All five bites were sabotaged, run to a captured failing transcript, then
reverted — verified via `go vet` + a focused `go test -race -count=1` run
before and after each sabotage.

| Bite | Sabotage | Command | Captured RED | Verdict |
|---|---|---|---|---|
| **S-HKS-050** (a) anti-vacuity | `drain()`: after every successful `invokeRecovered`, unconditionally call `l.report(StallOutstanding)` | `go test -race -count=1 -run TestHooks_AntiVacuity_NoStallEmptyReportSet` | `hooks_test.go:1260: report set has 1 entr(y/ies), want 0` | RED confirmed, reverted, GREEN confirmed |
| **S-HKS-051** (b) synchronous dispatch | `enqueuePostTurn`: call `obs(context.Background(), report)` directly at the fire site instead of enqueueing | `go test -race -count=1 -run TestHooks_Asynchrony_GoroutinePlacement_NoRunFrame` | Captured stack shows `(*Harness).Run` directly calling `(*observerLane).enqueuePostTurn` calling the observer closure — **fails as an assertion, not a hang**, exactly as design AD-9 predicts | RED confirmed, reverted, GREEN confirmed |
| **S-HKS-052** (c) reverse dispatch | `enqueuePostTurn`: iterate `for i := len(observers)-1; i >= 0; i--` | `go test -race -count=1 -run TestHooks_Ordering_RegistrationOrderAtAllFourPoints` | `hooks_test.go:959: post-turn order = [3 2 1 0], want [0 1 2 3]` | RED confirmed, reverted, GREEN confirmed |
| **S-HKS-053** (d) skip re-resolution | `compaction.go`: `cut = plan.Cut()` instead of `cut = resolveCut(hist, plan.Cut())` | `go test -race -count=1 -run TestHooksCompaction_ForwardAdjustment_StaysInvariantSafe` | First attempt (weak assertion) passed vacuously — **discovered a real gap in the GREEN test itself**: a harness-level `Run()` error is always nil regardless of the compaction's own outcome (R-CMP-010), so an *untouched* history is indistinguishable from a *safely-committed* one by a "pair stays together" check alone. Strengthened the GREEN test to assert `compaction_finished` count on the stream; re-sabotaged: `hooks_compaction_test.go:287: compaction_finished count = 0, want exactly 1` (`hist.markSpan(3)` reports `!spanOK` since 3 is not a recorded mark, so the compaction bracket takes its own existing failure arm) | RED confirmed (after correcting the test's own vacuity gap), reverted, GREEN confirmed |
| **S-HKS-054** (e) per-Run session-start | `harness.go`: `fireSessionStart := true` unconditionally, ignoring the latch | `go test -race -count=1 -run TestHooksHarness_SessionStart_OnceAcrossTwoSerialRuns` | `hooks_harness_test.go:86: session-start observer received 2 report(s), want exactly 1` | RED confirmed, reverted, GREEN confirmed |

**S-HKS-020 panic-recovery-removed** (task 6.6, S-DEL-022 precedent, not one of
the 5 mandated bites but explicitly required): removed the `defer recover()` in
`invokeRecovered`, ran `TestHooks_Panic_ReportedOnLane_LaneContinues_ProcessSurvives`
in isolation. Result: a genuine process-crashing panic trace (`panic: hks-020
deliberate panic, recovered by the lane` — goroutine trace through
`(*observerLane).drain` → `invokeRecovered` → the observer closure), **not** a
`--- FAIL:` line — non-zero process exit, exactly the S-DEL-022 evidence shape.
Reverted; full suite confirmed green afterward.

## Work Unit Evidence

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| Production surface (commit 1) | `go build ./... && go vet ./...` clean; `go test -race -count=1 -run TestHooks ./src/agent/...` green (exercises every new production path) | `agenttest.Gate`/hand-rolled channel gates + `agenttest.Provider` drive real `Harness.Run`/`Turn`/`Compact` calls through the new code — no mocks below the provider boundary | `git revert 83362a26` deletes `hooks.go`, reverts the four filter entries, and reverts loop.go/harness.go/compaction.go to their pre-AG-20 shape |
| Test suite (commit 2) | `go test -race -count=1 ./src/agent/...` — only the 2 pre-existing, unrelated failures (see below) | Same as above; every scenario drives a real `Harness`/`Turn` call, no scenario asserts against a mock | `git revert dac4fc3f` deletes the three `hooks_*_test.go` files and the `compaction_surgery_test.go` addition |
| Docs + tasks (commit 3) | N/A — documentation only; structural readback (grep for the two ticked checkboxes and the counter) | N/A | `git revert 779a4c4b` |
| Guard fix (commit 4) | `go vet` + `go test -race -count=1 -run TestHooks_SubstrateFilters...` green | N/A | `git revert 591ce82d` |

## Real bugs found and fixed during this apply (not merely test-writing noise)

1. **`fmt` import in `loop.go`** broke `TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage` (`fmt` transitively imports `os`/`io/fs`). Fixed by a hand-rolled `Unwrap()`-carrying error type, mirroring `cancellation.go`'s own documented precedent.
2. **`ai.At("turn.hooks")`** (one dotted-string call) instead of `ai.At("turn"), ai.At("hooks")` (two variadic segments) rendered `Violation.Path().String()` as `"?"` instead of `"turn.hooks"`. Fixed; confirmed against `loop.go`'s own `validateContinuation` precedent.
3. **`attemptsMade`**: the attempt loop's own `attempt` induction variable is scoped to the `for` statement and goes out of scope after `break` — sites (i) and (ii), both outside the loop, needed a separate outer-scope variable set on every iteration's first statement.
4. **A genuine scheduling race**, not a flaky-test artifact: a "clean" (non-held) observer CAN legitimately be reported `StallQueued` by the terminal snapshot even though it completes moments later, since `Run`'s own goroutine and the lane's drain goroutine race independently with no synchronization forcing one to wait for the other (by design — R-HKS-008 forbids waiting). Verified by both a hang (an unbuffered-sink attempt deadlocked, since `Run` blocks on `run_start` long before any observer is even enqueued) and a genuine failure (a too-small buffer let the race manifest). Fixed deterministically: size the sink buffer to exactly the pre-enqueue event count (from an unhooked baseline run), forcing `Run`'s first post-enqueue send to block until the test explicitly drains — giving the lane's drain goroutine unbounded, untimed opportunity to finish before the snapshot can possibly run.
5. **Forward-adjustment fixture** (`S-HKS-008`): an open call with no result anywhere survives compaction (retraction excludes it) but then breaks the *subsequent* logical turn's own request build (the model-facing request also requires pairing-closure). Fixed by using a complete call/result pair instead of a bare open call, so the forward-adjusted-then-retracted cut leaves both protected together.
6. **`TestHooksCompaction_ChainElementFailure...`** asserted `hist.Len()` stayed unchanged after a failed compaction — wrong: the run legitimately *continues* (R-CMP-010) and appends the following logical turn's own entries, so total length grows. Fixed to assert the *pre-existing* entries stay byte-unchanged instead.
7. **`TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters`** itself: once `hooks.go` was committed (no longer untracked), it legitimately started appearing in `git diff <merge-base>` — the test's own error message already said "(plus hooks.go, which is NEW, not pre-existing)" but never actually added it to the allowed set. Fixed in the small follow-up commit.

## Known limitation, reported honestly: S-HKS-011's no-row coverage

Design's R-HKS-004 table has 8 "no-fire" rows (7–14). Two are exercised
behaviorally end-to-end from `package agent_test`:
- **Row 13** (shutdown refusal at entry) — trivial, direct.
- **Row 7** (iteration-boundary signal, no turn ran this iteration) — via a
  test-local `interruptingTool` whose `Run` method calls `h.Interrupt()` as a
  side effect, so the signal is set *after* the tool-calling turn's own
  provider stream has already completed successfully but *before* the next
  iteration's own boundary check.

The other six (8: post-compaction signal; 9: steering-drain append error; 10:
prompt append error pre-loop; 11: terminal-decision steer append error; 12:
`NewRunStart` construction error; 14: compaction bracket) were judged
impractical to construct as independent behavioral fixtures from
`package agent_test` within this session's time budget — several require
forcing an unexported constructor or an internal validation path to fail in a
way this package cannot reach externally. Their shared mechanical basis is
instead verified structurally: `TestHooksHarness_PostTurn_NoRowFiresNothing`'s
own subtest asserts `harness.go` contains **exactly four**
`lane.enqueuePostTurn(` call sites and **exactly one**
`lane.enqueueSessionStart(` call site — meaning there is no fifth, hidden fire
site any of the eight no-rows could reach. This is a real, if narrower, proof
than eight independent behavioral runs; recorded here rather than silently
claimed as complete. Scenario ID `S-HKS-011` is still mapped to task 2.5 per
the coverage table, since the join of (2 behavioral rows) + (structural count
proof for the remaining 6) is what task 2.5 actually produced.

## Final full-suite evidence

```
$ cd backend/agent && go test -race -count=1 ./...
--- FAIL: TestCancellation_NestedRunCancelsLeafFirst (0.09s)
    cancellation_test.go:222: ...S-DEL-015 has nothing to scan (expected the AG-19 seam-install hunk)...
--- FAIL: TestScopeFence_S_TLS_020_SeamRidesBesidePolicySlotNotInsideIt (0.07s)
    scope_fence_test.go:259: scheduler.go's diff against the merge base is empty...
FAIL	github.com/cachicamas/backend/agent/src/agent	8.204s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.501s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep	1.885s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest	2.015s
ok  	github.com/cachicamas/backend/agent/src/ai	4.490s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	2.565s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	173.513s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest	3.152s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	3.433s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	6.618s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke	2.604s
ok  	github.com/cachicamas/backend/agent/src/handoff	2.272s
FAIL
real: 2:54.14 (174.14s), -count=1, uncached
```

The two failures are **pre-existing and unrelated to AG-20**: both are
anti-vacuity floors (`S-TLS-020`, `S-DEL-015`) that require a non-empty
`git diff <merge-base> -- scheduler.go|delegation_seam.go`. Since AG-19 is
already merged into `origin/main` (`2a138b59`) and AG-20 never touches either
file (design requires both byte-unchanged; confirmed via
`git diff --stat -- scheduler.go delegation_seam.go` on the AG-20 branch
showing zero lines), the diff is genuinely empty for AG-20 and for any future
milestone that also correctly leaves those two files alone. `baseRefIsHEAD`'s
own escape hatch only covers the literal post-merge-checkout case (`HEAD ==
baseRef`), not a later branch several commits ahead of that same merge-base.
Recorded separately: `bug/ag-19-anti-vacuity-floor-breaks-for-every-later-milestone`
(Engram). Not fixed here — both files are on AG-20's explicit do-not-touch
list, and `scope_fence_test.go` is on the byte-unchanged list too.

## Gates (Phase 6)

- `gofmt -l` on all 10 AG-20 files: clean (two files needed one `gofmt -w`
  pass for pure whitespace after initial authoring; verified clean after).
- `go vet ./...`: clean.
- `golangci-lint run --config=.golangci.yml ./...` (after `make lint`
  auto-installed golangci-lint v2.9.0): found 4 issues (3 comment-form
  findings on `StallOutstanding`/`StallQueued`/`StallPanicked`, 1 De Morgan's
  law suggestion) — fixed; re-run: **0 issues**.
- `make build`: clean.
- `make vuln-check`: exit 0, **zero reachable findings** — `govulncheck -json`
  reports several OSV database entries for dependencies in the build list
  (e.g. `go.opentelemetry.io/otel` `GO-2026-5506`), but zero `"finding"`/trace
  entries, meaning none is actually called from the reachable call graph.
