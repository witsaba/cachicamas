# Apply progress: AG-20 — Complete the hook taxonomy

**Status**: 52/54 tasks complete (7.4 discharged by `sdd-verify` itself, see
below). The remaining 2 (7.2, 7.3) are explicitly `sdd-archive` obligations
per their own task text, not apply's or verify's. `sdd-verify` round 1
returned **FAIL — 2 CRITICAL, 8 MAJOR, 5 MINOR**; all 15 findings are closed —
see "Correction round 2" below, which is this file's current, authoritative
evidence.

**Mode**: Strict TDD. Runner: `cd backend/agent && go test -race -count=1 ./...`.
Final evidence run (post round-2 correction, see below): **175.09s real**
(`2:55.09`), `-count=1`, uncached — matches the ~170s baseline this project's
memory records for a genuine (non-cached) run; exit 0, zero `FAIL`.

**Commits** (`feat/agent-layer2-wave5-ag20`):
1. `83362a26` `feat(agent): AG-20 -- complete the hook taxonomy production surface`
2. `dac4fc3f` `test(agent): AG-20 -- hook taxonomy scenario suite, all 49 scenarios`
3. `779a4c4b` `docs(0003): AG-20 -- tick hook taxonomy milestone, Wave 5 complete`
4. `591ce82d` `fix(agent): AG-20 -- S-LSK-032's own guard must allow hooks.go's diff`
5. `50e02b8f` `fix(agent): AG-20 -- close anti-vacuity floor merge blocker in scope guards`
   — see "Correction round 1" below.
6. `7d64d219` `docs(openspec): AG-20 -- record the correction commit hash in apply-progress`
7. `d6dae074` `docs(openspec): AG-20 -- annotate task 6.4 with the correction round`
8. `f5232f3a` `fix(agent): AG-20 -- repair the dead anti-vacuity regression guard (CRITICAL-1)`
9. `190d4426` `fix(agent): AG-20 -- track backtick raw strings in the shared stripGoComments`
10. `9268d007` `fix(agent): AG-20 -- scope L2C-08's goroutine-exit claim to admit the observer lane (CRITICAL-2)`
11. `f05cdccb` `docs(openspec): AG-20 -- correct four overreaching scenarios (MAJOR-1..4)`
12. `f3bc556e` `docs(openspec): AG-20 -- correct remaining MAJOR/MINOR findings (5..8, MINOR-4/5)`
    — commits 6-12 are "Correction round 2", see below.

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

## Correction round 1 (`sdd-apply` coordinator review): the two failures are a merge blocker, not pre-existing

**This section corrects an earlier, wrong claim in this same file.** The
first pass of this apply reported `TestScopeFence_S_TLS_020_...` and
`TestCancellation_NestedRunCancelsLeafFirst` as "pre-existing and unrelated
to AG-20," on the theory that they were structurally unable to pass on any
branch cut after AG-19 merged. **That claim was checked and is false.** Both
tests pass today on `main` (`2a138b59`):

```
$ git rev-parse origin/main && git merge-base HEAD origin/main
2a138b5997c5e6a3f5e13fcfdc873833ed8975fa
2a138b5997c5e6a3f5e13fcfdc873833ed8975fa
$ go test -race -count=1 -run 'TestScopeFence_S_TLS_020_SeamRidesBesidePolicySlotNotInsideIt|TestCancellation_NestedRunCancelsLeafFirst' ./src/agent/...
ok  	github.com/cachicamas/backend/agent/src/agent  (on main)
```

They fail only on AG-20's own branch. This is a genuine **merge blocker**,
not a pre-existing condition, and treating it as pre-existing would have
shipped a red suite on `main` the moment AG-20 merged.

**Root cause.** `scope_fence_test.go:236-259` (S-TLS-020) does: (1) resolve
`baseRef = merge-base(HEAD, origin/main)`; (2) `if baseRefIsHEAD(...) {
t.Skip(...) }` — this is the *only* case AG-19's own fix (`c4e4455f`)
covers, and it is why the test is green when run **on** `main` itself, where
`HEAD == baseRef`; (3) `diff := gitDiff(baseRef, ".../scheduler.go")`; then
**`if diff == "" { t.Fatal(...) }`** — an anti-vacuity floor. On a branch cut
*after* AG-19 merged, step 2 does not fire (`HEAD != baseRef` — the branch is
ahead of its own merge-base by definition), and `scheduler.go`'s diff against
that merge-base is legitimately empty because AG-20 correctly does not touch
`scheduler.go`. Step 3 then fatals. `baseRefIsHEAD`'s skip only ever covered
the literal post-merge-checkout case, never the general "this branch simply
doesn't touch the guarded file" case — so the floor fires on every descendant
branch that (correctly) leaves the guarded file alone, not only on AG-20's.

**The defective shape appears in three places — all three are fixed, not
only the two that were red:**

1. `scope_fence_test.go:259` (S-TLS-020, AG-19's, over `scheduler.go`) — was RED.
2. `cancellation_test.go:222` (S-DEL-015, AG-19's mirror, over
   `delegation_seam.go`/`scheduler.go`) — was RED.
3. `hooks_test.go:1536` (AG-20's own, new this milestone, over `loop.go`) —
   was GREEN only because AG-20 happens to modify `loop.go`; on a future
   branch that doesn't, it would fatal for the identical reason. This
   happened because task 4.3's own text (`tasks.md:108`) instructed reusing
   `scope_fence_test.go`'s pattern *including* "fail loudly on an empty diff
   (`S-TLS-020` precedent)" — the defective shape was in the plan, not just
   copied by accident, and this apply propagated it into new code
   unmodified. Left as-authored in `tasks.md` (task 4.3 was in fact
   completed as specified); the correction is recorded here instead.

**The fix.** For a guard asserting an absence in a file's *added* lines, an
empty diff means there are no added lines, so the asserted absence holds
trivially and correctly — that is a **skip**, not a failure. All three sites
now do `t.Skip(...)` on an empty diff, each with a message stating that the
subject file is unchanged on this branch (so the absence holds vacuously)
and that the guard bites on the branch that actually introduces the hunk
(where the diff is non-empty — verified true for S-TLS-020 on AG-19's own
branch history). `cancellation_test.go`'s two-path loop now skips only if
**neither** of its two paths had anything to scan, preserving a real scan on
either one alone. A new regression test,
`TestHooks_AntiVacuityFloorsSkipRatherThanFail`, source-scans all three files
for the `diff == "" { ... t.Fatal }` anti-pattern and fails if it reappears
anywhere, so this cannot silently regress on AG-21/22/23.

**Spec documentation.** `scope_fence_test.go` was previously listed as
byte-unchanged in three delta files; it now carries one narrow, explicit
release (bounded to exactly this fatal-to-skip conversion, nothing else),
recorded following the AG-11 (`turn_events.go`/`failure.go`) and AG-18
(`doc.go`/`doc_contract_guard_test.go`) precedents:
- `specs/agent-loop-skeleton/spec.md` — `R-LSK-008` gains a new consequence
  (6) describing the release in full, plus a new scenario `S-LSK-033`
  asserting the fixed property (skip, not fail, on an empty diff; the
  anti-pattern is absent from all three files).
- `specs/agent-hook-taxonomy/spec.md` — `R-HKS-010`'s byte-unchanged list no
  longer names `scope_fence_test.go`; a fourth consequence cross-references
  the release.
- `specs/agent-delegation-readiness/spec.md` — the header note, the
  `R-DEL-009` "not modified" table row, and `S-DEL-026`'s scenario text are
  all updated to the same accurate framing.
`agent-run-driver/spec.md`'s two mentions (`:13`, `:44`) were reviewed and
left unchanged: both are narrowly scoped to the `NumMethod()`/named-method
assertion specifically, which remains byte-unchanged and accurate.
`design.md`/`proposal.md` (planning artifacts, not deltas) were left
unchanged, consistent with SDD practice of recording apply-time deviations
in this file rather than retroactively editing frozen planning docs.

**Verification after the fix** (focused, `-race -count=1 -v`):

```
--- PASS: TestHooks_AntiVacuityFloorsSkipRatherThanFail (0.02s)
    --- PASS: TestHooks_AntiVacuityFloorsSkipRatherThanFail/cancellation_test.go (0.00s)
    --- PASS: TestHooks_AntiVacuityFloorsSkipRatherThanFail/scope_fence_test.go (0.01s)
    --- PASS: TestHooks_AntiVacuityFloorsSkipRatherThanFail/hooks_test.go (0.03s)
--- SKIP: TestScopeFence_S_TLS_020_SeamRidesBesidePolicySlotNotInsideIt (0.06s)
    scope_fence_test.go:270: S-TLS-020: scheduler.go's diff against the merge base is empty on this branch — the absence this guard asserts holds vacuously (nothing was added to violate it); the guard bites on the branch that actually introduces the seam-install hunk
--- SKIP: TestCancellation_NestedRunCancelsLeafFirst (0.07s)
    cancellation_test.go:240: S-DEL-015: neither delegation_seam.go nor scheduler.go changed against the merge base on this branch — the absence this guard asserts holds vacuously; it bites on the branch that actually introduces the seam-install hunk, mirroring S-TLS-020's identical fix at scope_fence_test.go
--- PASS: TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters (0.07s)
--- PASS: TestHooks_ScopeFence_ByteUnchangedFilesAndNoNewKind (0.22s)
PASS
ok  	github.com/cachicamas/backend/agent/src/agent	1.753s
```

The earlier Engram memory entry titled
`bug/ag-19-anti-vacuity-floor-breaks-for-every-later-milestone`, which
repeated the "pre-existing, not fixed here" framing, is superseded by this
section and by a corrected Engram save made in this same correction round.

## Correction round 2 (`sdd-verify`): FAIL — 2 CRITICAL, 8 MAJOR, 5 MINOR, all closed

`sdd-verify` independently re-ran the suite (**171.00s real**, exit 0),
re-derived the coverage matrix from the test bodies rather than trusting this
file's own claims, planted the anti-vacuity anti-pattern back into the guard
round 1 shipped, and opened every cited requirement. Verdict: the production
code is correct, every non-negotiable invariant holds by measurement, and task
7.4's ADDED-not-MODIFIED judgment was discharged in this change's favour — but
9 scenarios would have archived asserting something the shipped tests do not
prove, and the correction-round-1 regression guard did not fire. Every finding
below is a spec-versus-code contradiction or an untested claim, never a code
defect; **no test was weakened and no coverage was deleted** to close any of
them.

| # | Finding | What changed | Test or scenario? |
|---|---|---|---|
| CRITICAL-1 | `TestHooks_AntiVacuityFloorsSkipRatherThanFail`'s regex bounded the scan to 300 chars between `{` and `t.Fatal`; every protected site's own explanatory comment (717 chars at the reproduced site) exceeded it, so planting the anti-pattern back in at any of the three sites passed silently | Widened `[^}]{0,300}` to unbounded `[^}]*` (safe: `[^}]` cannot cross the guarded branch's own `}`) and ran the match against comment-stripped source, reusing this package's own `stripGoComments` (`scheduler_test.go`) rather than adding a competing implementation. Discovered `stripGoComments` itself had a latent bug — no backtick-raw-string tracking, desynced by `hasSuffixLiteralPattern`'s own 3-quote regex literal — and fixed that at its root, benefiting its two pre-existing callers too. **Proved the repair bites**: planted the historical anti-pattern back at all three sites in turn, captured the real `--- FAIL` output for each, reverted via `git checkout --` | **Test** (the regression guard itself, plus its shared dependency) |
| CRITICAL-2 | `L2C-08`'s package contract row claimed "every goroutine the package itself owns has exited" past the wind-down bound; AG-20's own observer lane goroutine may legitimately still be running, by design, while blocked inside a permanently stalled observer — falsifying the row, with no delta recording it | Widened `doc.go`'s `L2C-08` row and `doc_contract_guard_test.go`'s mirrored `expectedLayer2ContractRows` entry to name a second carve-out (a permanently stalled observing-hook invocation, reported typed by hook point and index), following the AG-18/`L2C-07` precedent exactly; added `TestDocContract_L2C08CarveOutClause` mirroring `TestDocContract_L2C07BothClausesTogether`; added a new `agent-cancellation-tree` delta amending `R-CAN-006` (a third disjoint-set table row) and `R-CAN-008` in full, with two new scenarios (`S-CAN-016`, proven by existing `S-HKS-017`/`S-HKS-019` fixtures without a new Go test; `S-CAN-017`, proven by the new doc-contract test); added the missing row to `R-HKS-011`'s own closed-sequence table; recorded the narrow release of `doc.go`/`doc_contract_guard_test.go` (no new filter entry needed — both pre-existing in both substrate filters since AG-18) | **Production doc + spec + one new test** |
| MAJOR-1 | `S-HKS-026` claimed a merge-base comparison and history-read-back/provider-request byte-identity the test never performs — it runs the same hookless config twice on this branch and compares only event-kind sequences. AG-19's identical shortcut earned its justification (its own diff is never read on the hookless path); AG-20 does not re-earn it, since `loop.go`'s chain composition, `harness.go`'s capture locals and `compaction.go`'s `len(chain) > 0` guard all execute unconditionally on that path too | Corrected the scenario (and the test's own Go doc comment) to claim only what two-runs-on-this-branch proves: determinism under repetition and goroutine-count return-to-baseline, with equivalence to pre-AG-20 behavior stated as a structural (code-inspection) claim, not a behavioral one this test measures. Also fixed a stale "six paired runs" to the actual seven arms | **Scenario** (chose narrowing, the option `sdd-verify` explicitly offered, over building an impractical merge-base-checkout comparison this codebase has no precedent for) |
| MAJOR-2 | `S-HKS-011` demanded eight behavioral runs; two shipped (rows 13, 7) plus a structural call-site-count proof for the remaining six, exactly as this file's "Known limitation" section above and the Go test's own comment already disclosed honestly | Corrected the scenario text to state the actual, achieved method (2 behavioral + structural count proof) instead of the unshipped 8-run claim | **Scenario** (the Go test and this file's own prior prose were already honest; only the spec scenario overreached) |
| MAJOR-3 | `S-HKS-023` / `S-PRH-009` both claimed attribution is unchanged when elements are inserted INTO the chain ahead of the failing one — false and must be false: attribution is that element's own slice index, so insertion ahead of it correctly renumbers it. The test proves a different, true claim: the SINGULAR field's own presence/absence does not renumber the chain | Corrected both scenarios to state the true claim (stability under the singular field's own insertion/removal, explicitly NOT under insertion into the chain) | **Scenario** (×2 — `agent-hook-taxonomy` and `agent-pre-request-hook` deltas) |
| MAJOR-4 | `S-HKS-001` still claimed `scope_fence_test.go` (the whole file) is byte-unchanged; `R-HKS-010` consequence 4 had already retracted that claim for the file but `S-HKS-001` was never updated to match | Corrected `S-HKS-001` to name the specific assertions that stay byte-unchanged (not the whole file); swept the entire change directory for the same stale pattern and confirmed every remaining mention is narrowly scoped to a still-accurate, genuinely-unedited assertion (e.g. `:102-105`'s `NumMethod()` check) | **Scenario** |
| MAJOR-5 | `S-HKS-012` claimed each retried attempt carries a distinct nonzero usage figure and the sum strictly exceeds the last attempt's own — both structurally unachievable: `retryDecision`'s G1-G5 gates only retry attempts that never reach `finalize()`, the sole `cost_turn` emission site, so a retried attempt always contributes zero | Corrected the scenario to the achievable claim the fixture proves: the sum equals the one completing attempt's own figure; attempt count is genuinely 3 | **Scenario** |
| MAJOR-6 | Bite `S-HKS-053` claimed its RED shows "the committed prefix splitting a call/result pair"; the actual, reproduced RED is "compaction fails typed" (`hist.markSpan` rejects the cut, zero `compaction_finished`) — `sdd-verify` additionally neutralised the `markSpan` guard too and it STILL could not reach a split, since `buildLoopRequest` independently refuses a pairing-broken prefix | Corrected the scenario and the bite table row to the actual, reproduced observable, while keeping the verdict the property IS proven (removing re-resolution turns success into typed failure) | **Scenario** (the actual RED shape was already recorded accurately in this file's own "TDD Cycle Evidence" table above, `S-HKS-053` row — only the spec's own scenario text and bite table had drifted from it; `sdd-verify` independently reproduced the same RED, including the further probe that shows the split is unreachable even with a second guard also removed) |
| MAJOR-7 | `S-RUN-113` claimed a four-method set; the shipped set is five (`Compact, Interrupt, Run, Shutdown, Steer`) — `R-RUN-001`'s own pre-existing (AG-18) staleness, which AG-20 did not create but which `S-RUN-113` restated as a fresh claim | Corrected the scenario to state the true, current five names directly rather than delegating to `R-RUN-001`'s stale count; renamed the covering test from `...AdditionsExactlyFourAbsencesAsserted` to `...AdditionsExactlyFiveAbsencesAsserted` (identifier only — the test's own assertion logic was already correct and unchanged) | **Scenario**, plus a test **identifier** rename (no logic change) |
| MAJOR-8 | `tasks.md`'s coverage table stated a total of "49 scenarios," now stale (52 new scenarios after `S-LSK-033`, `S-CAN-016`, `S-CAN-017`); `S-LSK-033` was additionally absent from the coverage table entirely | Replaced the stated total with the `S-LSK-020` no-total treatment (allocated ranges per delta instead of a count that goes silently stale on the next append); added the three missing scenario rows | **Task artifact** (`tasks.md`, not a Go test or an OpenSpec scenario) |
| MINOR-1 | `hooks_test.go`'s S-HKS-024 test ran its loop.go anti-vacuity `t.Skip` BEFORE four diff-independent reflection assertions (event-kind count, method count, `Turn`/`Run` signatures); on a future branch not touching `loop.go`, all four would silently stop running with this suite | Moved the skip to the end of the function, after all four diff-independent assertions | **Test** |
| MINOR-2 | `S-LSK-033`'s bounding clause said the release "converts one branch to a call to `t.Skip`" — true for `scope_fence_test.go`, inaccurate for `cancellation_test.go`, which converts to `continue` plus a `scannedAny` flag plus a separate terminal branch | Corrected the scenario to describe each file's actual, different shape precisely | **Scenario** |
| MINOR-3 | `TestCancellation_NestedRunCancelsLeafFirst` reported `SKIP` even though its entire behavioral core had already run and passed — misleading to anyone scanning for skipped coverage, and inconsistent with this same file's own pre-existing `baseRefIsHEAD` branch, which uses `t.Logf` + `return` a few lines above | Converted the terminal "nothing to scan" branch from `t.Skip` to `t.Logf` + `return`, matching the file's own established idiom; the test now reports `PASS` | **Test** |
| MINOR-4 | Doc 0003's status line said "20 of 24" (the document's own Wave 0-6 table of contents sums to 21 shipped) and "5 of 7 wave-2 milestones shipped" (the same table lists exactly 5 Wave-2 milestones, all shipped — the "7" was never correct) — both pre-existing, carried forward rather than introduced by AG-20 (prior counter value was 19); AG-20's own two ticks and narrative sentence were independently confirmed accurate | Corrected both counts using the document's own authoritative table of contents as evidence | **Architecture doc** (not a Go test or OpenSpec scenario) |
| MINOR-5 | "Byte-identical" in `S-HKS-007`, `S-HKS-017` and `S-RUN-113` is implemented as `kindsEqualHKS` (event-KIND sequence equality) — payload fields, full events and (except `S-HKS-007`'s own history-read-back half, which genuinely is compared entry-by-entry) provider requests are not compared | Added a precision parenthetical to each scenario naming the actual mechanism, without touching `S-HKS-007`'s accurate history-read-back claim | **Scenario** (×3) |

No finding required deleting or weakening a test, and no finding required
adding new behavioral coverage beyond CRITICAL-1's own regression guard and
CRITICAL-2's one new doc-contract test — every other fix corrected prose to
match code that was already correct.

## Final full-suite evidence (round 2, post-`sdd-verify`-correction, uncached)

Round 1's own evidence (172.57s, immediately above) is superseded by this
run, taken after every CRITICAL/MAJOR/MINOR fix above landed:

```
$ cd backend/agent && go test -race -count=1 ./...
ok  	github.com/cachicamas/backend/agent/src/agent	9.462s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.209s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep	1.792s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest	1.900s
ok  	github.com/cachicamas/backend/agent/src/ai	4.441s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	2.385s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	174.486s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest	2.618s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	3.151s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	6.203s
?   	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures	[no test files]
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke	2.360s
ok  	github.com/cachicamas/backend/agent/src/handoff	2.073s
$ echo $?
0
```

Wall-clock, measured with `time`: **`( go test -race -count=1 ./... )  65.18s
user 7.86s system 41% cpu 2:55.09 total`** — **175.09s real**, `-count=1`,
uncached (started `13:02:39`, finished `13:05:34`). Zero `FAIL` anywhere in
the module, zero `(cached)` marker. `go vet ./...`: clean, no output.
`gofmt -l` on every AG-20-owned and correction-round file (`hooks.go`,
`hooks_test.go`, `hooks_harness_test.go`, `hooks_compaction_test.go`,
`loop.go`, `harness.go`, `compaction.go`, `loop_test.go`, `loop_hook_test.go`,
`compaction_surgery_test.go`, `scope_fence_test.go`, `cancellation_test.go`,
`scheduler_test.go`, `doc.go`, `doc_contract_guard_test.go`): clean —
`scheduler_test.go` alone shows 3 gofmt hunks (lines 35, 77, 1028), all
confirmed pre-existing and outside this correction round's own edit (which
sits at ~lines 677-745) via `gofmt -d`; none touches this round's own lines.

**Anti-vacuity guard proof (CRITICAL-1), captured transcripts:**

```
$ # site 1: scope_fence_test.go, t.Skip -> t.Fatal planted
$ go test -count=1 -v -run 'TestHooks_AntiVacuityFloorsSkipRatherThanFail' ./src/agent/...
    hooks_test.go:2222: scope_fence_test.go contains an empty-diff branch that
    calls t.Fatal(f) instead of t.Skip ...:
        diff == "" {
        		t.Fatal
--- FAIL: TestHooks_AntiVacuityFloorsSkipRatherThanFail (0.02s)
    --- FAIL: TestHooks_AntiVacuityFloorsSkipRatherThanFail/scope_fence_test.go (0.01s)
FAIL   (reverted via git checkout --; re-confirmed PASS afterward)

$ # site 2: cancellation_test.go, continue -> t.Fatal planted
$ go test -count=1 -v -run 'TestHooks_AntiVacuityFloorsSkipRatherThanFail' ./src/agent/...
    hooks_test.go:2222: cancellation_test.go contains an empty-diff branch
    that calls t.Fatal(f) instead of t.Skip ...
--- FAIL: TestHooks_AntiVacuityFloorsSkipRatherThanFail (0.02s)
    --- FAIL: TestHooks_AntiVacuityFloorsSkipRatherThanFail/cancellation_test.go (0.00s)
FAIL   (reverted via git checkout --; re-confirmed PASS afterward)

$ # site 3: hooks_test.go's own S-HKS-024 site, t.Skip -> t.Fatal planted
$ go test -count=1 -v -run 'TestHooks_AntiVacuityFloorsSkipRatherThanFail' ./src/agent/...
--- FAIL: TestHooks_AntiVacuityFloorsSkipRatherThanFail (0.02s)
    --- FAIL: TestHooks_AntiVacuityFloorsSkipRatherThanFail/hooks_test.go (0.02s)
FAIL   (reverted via git checkout --; re-confirmed PASS afterward)
```

All three plants caught, all three reverted cleanly (`git status` clean after
each), `go build`/`go vet` confirmed clean after the final revert.

## Gates (Phase 6)

- `gofmt -l` on all 10 AG-20 files plus the round-2 correction files (see
  above): clean.
- `go vet ./...`: clean (re-confirmed round 2).
- `golangci-lint run --config=.golangci.yml ./...` (after `make lint`
  auto-installed golangci-lint v2.9.0): found 4 issues (3 comment-form
  findings on `StallOutstanding`/`StallQueued`/`StallPanicked`, 1 De Morgan's
  law suggestion) — fixed; re-run: **0 issues**. Not independently re-run in
  round 2 (`sdd-verify` noted this and did not re-run it either; no round-2
  edit touched a linted construct these findings were about).
- `make build`: clean.
- `make vuln-check`: exit 0, **zero reachable findings** — `govulncheck -json`
  reports several OSV database entries for dependencies in the build list
  (e.g. `go.opentelemetry.io/otel` `GO-2026-5506`), but zero `"finding"`/trace
  entries, meaning none is actually called from the reachable call graph.
