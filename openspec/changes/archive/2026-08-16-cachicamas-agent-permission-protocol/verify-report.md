```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:d3e7cd770696ee7dc71d06d1c0ed31286f295ecd4abcd2d330bf42656331d21a
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 12/12
scenarios: 13/13
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:448443e8b46f31818b8d2a9251c33f3cf501a9ba1b1a556175dc3a522b72a972
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report (TERMINAL — supersedes the `c0fcc6e5` report)

**Change**: `cachicamas-agent-permission-protocol` (AG-10, Layer 2 Wave 2, milestone 10/24)
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag10` @ `ecfebcd4` (clean), merge-base `main` `6de08335`
**Mode**: Strict TDD · **Store**: hybrid · **RDD**: off (no `gentle-ai review` operations performed)
**Scope**: focused re-verification of the 5-commit remediation round `c0fcc6e5..ecfebcd4`, plus a full regression sweep
**Verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 4 WARNING, 3 SUGGESTION
**Prior verdict**: PASS WITH WARNINGS — 0 CRITICAL, 8 WARNING, 5 SUGGESTION

---

## Remediation round — disposition of every prior finding

| Prior | Title | Status | Proof |
|---|---|---|---|
| **W1** | Concurrent-remember race emits a `CardinalityAtMostOne`-violating stream | **CLOSED** | `rememberedSet.rememberIfAbsent` is a true CAS under the existing mutex. Defeat test (ignore the CAS return value): `TestPermission_RememberedCardinality_ConcurrentRace_AtMostOneEmission` FAILS with `count = 2, want exactly 1`. |
| **W2** | The S-PPB-004 bite hard-coded the W1 defect as required behaviour | **CLOSED** | The `want 2` assertion is gone. The name `TestPermission_RememberedCardinality_SecondEmissionRejected` now holds the genuinely restored hand-built `CheckStream` validator test (constructs two events → asserts `ai.ErrDuplicate`); the real-`Schedule` forced race moved to a new, differently-named test asserting `want exactly 1`. Both exist — this is a restore, not a rename. |
| **W3** | Structurally guaranteed lost wakeup on the wake surface | **CLOSED** | `parked.park(callID)` now precedes the `decision_required` emission. Defeat test (move registration back after the ack), full package, 20 separate processes: `TestPermission_DeferEmitsBeforePark` FAILS **20/20** deterministically. |
| **W4** | `<-reqAck` not cancellation-aware | **CLOSED (as scoped)** | Now `select { case <-reqAck: ; case <-ctx.Done(): parked.remove(...); <typed abort> }`. Defeat test (bare `<-reqAck`, keeping W3's reorder): `TestPermission_AbandonedAckWithCancel_GateDeregistersPromptly` FAILS. One documented residual survives — see R1 below. |
| **W5** | Delta spec is structurally unpromotable; target capability dir absent | **OPEN** — archive-phase work, unchanged | Re-confirmed by command below. |
| **W6** | The delta spec's "Given/When/Then" format claim is false | **OPEN** — archive-phase work, unchanged | `grep` → 0 Given/When/Then lines. |
| **W7** | `make vuln-check` fails on 5 pre-existing Go stdlib advisories | **OPEN** — accepted, out of scope, unchanged | Re-run: exit 2, same 5 IDs, `AGENT_TRACES=0`. |
| **W8** | design.md's E2E row and `agenttest` decorator undelivered | **CLOSED, with two verified deviations** | Both deviation claims independently proven by command — see D1/D2 below. |
| **S1** | S-PPB-001 survives deletion of the whole gate | **CLOSED** | `policy.resolveInvocations() == 1` added. Defeat test (delete the entire gate): FAILS with `invocation count = 0, want 1`. |
| **S2** | Last swallowing `if rerr == nil { emit }` site un-cross-referenced | **CLOSED** | 18-line rationale added at the site, explicitly naming defect 6 and why this one site stays best-effort. Behaviour intentionally unchanged. |
| **S3** | Add the `design.md:76` E2E or amend the design | **CLOSED via W8** — residual carried as S7 | `TestTurn_PermissionPolicy_E2E_DeferDenyModify` added; `design.md` itself was not back-annotated. |
| **S4** | Add the `agenttest` no-op decorator or strike the line | **CLOSED** | `NoOpPermissionPolicy` added in `permission_policy_helpers_test.go`, with its own wiring test. Placement deviation proven forced — see D2. |
| **S5** | `apply-progress` 1176 PASS vs verify 3138 PASS | **CLOSED — mechanism identified** | Pure counting-method difference on one output: `grep -c '^--- PASS'` → **1181** (top-level), `grep -c -- '--- PASS'` → **3143** (top-level + subtests), `grep -c '^ok '` → **12** packages. Both prior numbers measured the same kind of run correctly under different greps. |
| **NEW W9** | The R-APP-002/D4 ack lost its only non-vacuous guard | **OPEN** | See W9 below. |

**Prior warnings closed: 5 of 8 (W1, W2, W3, W4, W8). Prior suggestions closed: 5 of 5.
One new WARNING (W9) and three new SUGGESTIONS (S6, S7, S8) introduced by this round.**

---

## Gates re-run by this phase (verbatim)

```text
$ cd backend/agent && ./bin/golangci-lint cache clean && make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
LINT_EXIT=0

$ cd backend/agent && make build
go build -trimpath ./...
BUILD_EXIT=0

$ cd backend/agent && make test          # go test -race -v ./...
TEST_EXIT=0
PASS_LINES=3143       # grep -c -- '--- PASS'   (top-level + subtests)
TOP_LEVEL_PASS=1181   # grep -c '^--- PASS'
FAIL_LINES=0
OK_PKGS=12
RACES=0

$ go test -race -count=15 ./src/agent/
ok  	github.com/cachicamas/backend/agent/src/agent	6.040s
COUNT15_EXIT=0

$ go test -race -count=10 -run 'TestPermission|TestTurn_Permission|TestNoOp' ./src/agent/
ok  	github.com/cachicamas/backend/agent/src/agent	3.514s

$ cd backend/agent && make vuln-check
VULN_EXIT=2
GO-2026-5026 GO-2026-5972 GO-2026-6089 GO-2026-6090 GO-2026-6218
AGENT_TRACES=0

$ git diff --name-only $(git merge-base HEAD origin/main) -- backend/agent/src/agent/
backend/agent/src/agent/loop.go
backend/agent/src/agent/loop_hook_test.go
backend/agent/src/agent/loop_permission_e2e_test.go
backend/agent/src/agent/loop_test.go
backend/agent/src/agent/loop_tool_dispatch_test.go
backend/agent/src/agent/permission_policy_helpers_test.go
backend/agent/src/agent/permission_protocol.go
backend/agent/src/agent/permission_protocol_test.go
backend/agent/src/agent/scheduler.go
backend/agent/src/agent/scheduler_test.go
backend/agent/src/agent/tool.go
# 11 changed of 44 .go files; all 11 are allowlisted. 33 files byte-unchanged.

$ git diff --stat $(git merge-base HEAD origin/main) -- backend/agent/go.mod backend/agent/go.sum
(no output)

$ go test -count=1 -v -run 'TestTurn_SubstrateUntouched|TestTurn_PreRequestHook_SubstrateUntouched' ./src/agent/
--- PASS: TestTurn_PreRequestHook_SubstrateUntouched (0.06s)
--- PASS: TestTurn_SubstrateUntouched (0.03s)
ok  	github.com/cachicamas/backend/agent/src/agent	0.565s
```

**Coverage**: skipped — no coverage target wired into `make`; `test/cover` exists but is not part of the milestone's gate set.

---

## Adversarial verification of the remediation

### R1 — W3: the park-registration reorder

**R-APP-002 / D4 happens-before is preserved.** Registration (`parked.park`) is a mutex-guarded map insert that returns immediately; it is not the parked wait. The parked **wait** — `select { case <-parkCh: ; case <-ctx.Done(): }` — still runs strictly after `<-reqAck`, and `runDispatcher` still closes `em.ack` only after `sink <- &stamped` returns. The code preserves the distinction and the ack still gates the park. **Verified by reading `scheduler.go` `runPermissionGate` and `runDispatcher`.**

**A wake arriving between registration and the park is not lost.** `parkedSet.wake` does lookup → delete → `close(ch)` under one lock acquisition. A `select` on an already-closed channel returns immediately, so a wake that lands during the ack wait is observed, not dropped.

**Every exit path deregisters. No leaking path found.**

| Exit | Deregistered by | Site |
|---|---|---|
| Upward-path wake | `parkedSet.wake` (lookup-then-delete before `close`) | `permission_protocol.go` |
| Cancel while waiting for the ack | `parked.remove(call.ID())` | ack `select`, `ctx.Done()` arm |
| Cancel while genuinely parked | `parked.remove(call.ID())` | park `select`, `ctx.Done()` arm |
| Re-park after a wake (Defer twice) | `park()` overwrites the map entry with a fresh channel | tail recursion |
| Simultaneous wake + cancel | `remove` is an unconditional no-op-safe delete | by construction |

One residual, pre-existing and **not** a new leak class: the `emissions <- emission{ev: reqEv, ack: reqAck}` send now sits *between* registration and the ack wait, and that send is not cancellation-aware. If a consumer abandons `sink` **and** the `len(calls)*2` emissions buffer is already full, a gate goroutine blocks there with its parked entry registered. This is the same hazard class the prior report recorded as pre-AG-10.

**`wakeParkedWithRetry` is deleted** (`grep` → absent) and no caller reintroduced an equivalent. All five wake sites are now single direct `sched.WakeParked(id)` calls. The one remaining poll loop is inside `TestPermission_DeferEmitsBeforePark` itself, where polling-without-reading-`sink` *is* the assertion mechanism, not a masking retry.

### R2 — W4: cancellation-aware ack

`<-reqAck` is now a two-arm `select` with `ctx.Done()`. The abort arm produces `typedExecutionFailureFromError(call.ID(), ctx.Err())` — **byte-identical in shape to the pre-existing mid-park cancellation arm** — writes `results[ordinal]`, calls `emitExecutionFailure`, and returns `abort.Failure`.

**No blocking *receive* remains ahead of the cancellation select.** The only blocking operations ahead of it are `policy.Resolve(ctx, call)` (which takes `ctx`) and the `emissions <-` **send** described in R1.

**Honest scoping of "genuinely returns rather than hanging".** Cancellation now releases the *gate goroutine* — proven by the defeat test — and therefore lets `wg.Wait()` return. It does **not** make `Schedule` return: `Schedule` then does `close(emissions)` followed by `<-dispatcherDone`, and the dispatcher is still blocked on its `sink <- &stamped` send. An abandoned `sink` still hangs `Schedule` until a reader appears. The remediation's own test comment states this explicitly and the test drains `sink` before asserting end-to-end completion. **W4 is closed for the narrowing AG-10 introduced (hang on the first deferred call, unreleasable by cancel); it is not a fix for the pre-existing abandoned-sink hazard, and the artifacts do not claim it is.**

### R3 — W1/W2: concurrent-remember CAS

- **True CAS**: `rememberIfAbsent` takes `r.mu`, checks membership, inserts, returns whether this caller inserted — one critical section, `defer`-unlocked.
- **Read-class calls are not serialized.** The mutex covers only a map lookup and insert. `policy.Resolve`, the read semaphore (`scheduler.go` bounded fan-out), and tool execution are all outside it. AG-09.2 semantics are preserved, and the new test asserts it positively: `fs.write invocations = 2` (the CAS gates the *event*, not execution). `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot` remains green.
- **The new bite is not vacuous.** Defeat test — replace `if !remembered.rememberIfAbsent(...) { return }` with a discarded return value — makes `TestPermission_RememberedCardinality_ConcurrentRace_AtMostOneEmission` FAIL: `resolution_remembered count = 2, want exactly 1`. It also fails on `count == 0`, so it asserts the invariant in both directions rather than the defect.
- **The hand-built `CheckStream` validator test was genuinely restored, not renamed.** The file now contains two distinct tests: `..._SecondEmissionRejected` builds `run_start / turn_start / remembered / remembered / turn_end / run_end` by hand and asserts `errors.Is(report.Violation(), ai.ErrDuplicate)`; `..._ConcurrentRace_AtMostOneEmission` is a separate function driving a real `Schedule`. Direct `CardinalityAtMostOne` coverage did not move — it came back.

### D1 — W8 deviation 1: the E2E aborts its Defer leg by ctx-cancel

**Claim verified.** `grep -n "WakeParked" src/agent/loop.go` returns only two comment lines (124, 264); there is no call and no exported handle. The `Scheduler` is constructed as an unexported local (`sched := &Scheduler{MaxConcurrentReads: maxReadFanOutDefault}`), `func Turn(...)` returns `(ai.Message, ai.FinishReason, error)`, and `TurnOptions` exposes no scheduler or wake field. **No caller can reach `WakeParked` for a call parked inside a `Turn()` invocation.** Building design.md's literal wake-resume E2E would require widening `Turn`'s public surface — correctly deferred to AG-13, and recorded as a deviation rather than silently dropped.

### D2 — W8/S4 deviation 2: the no-op policy lives in `src/agent`, not `agenttest`

**Claim verified empirically, not just by reading.** `src/ai/import_boundary_test.go` scans `src/ai/...`, `src/agenttest/...` and `src/handoff/...` as one `go list -deps -test` closure and lists `<module>/src/agent` in `forbiddenPrefixes` with rule *"ADR 0005 § D1 row 1: Layer 1 must not import Layer 2"*.

Defeat test — an isolated copy of the module with one added file `src/agenttest/zz_probe.go` importing `src/agent`:

```text
BASELINE (no probe):
ok  	github.com/cachicamas/backend/agent/src/ai	0.567s

DEFEAT (agenttest imports src/agent):
--- FAIL: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault (0.08s)
    import_boundary_test.go:192: Layer 1 must not import "github.com/cachicamas/backend/agent/src/agent"
      rule: ADR 0005 § D1 row 1: Layer 1 must not import Layer 2
```

`agenttest` genuinely cannot hold a `PermissionPolicy` implementation, and an import-boundary test enforces it as a hard build/test failure. The placement is forced, follows `scripted_tool_test.go`'s recorded AG-09 precedent, and is not drift.

### R5 — the substrate-filter widening (`b378cc76`)

Both `filterOutLoopFiles` (`loop_test.go`) and `filterOutLoopHookFiles` (`loop_hook_test.go`) gained exactly **two** entries each: `strings.HasSuffix(path, "/loop_permission_e2e_test.go")` and `strings.HasSuffix(path, "/permission_policy_helpers_test.go")`. These are exact filename suffixes — no wildcard, no directory-level or prefix widening — so they admit only those two files and nothing else.

The guards are *allowlists of files permitted to change*; the substrate invariant is everything else in `backend/agent/src/agent/`. The empirical check confirms the invariant is intact: 11 of 44 `.go` files differ from the merge-base, all 11 are allowlisted, the remaining **33 files are byte-unchanged**, and `go.mod`/`go.sum` show a zero-line diff. Both substrate tests PASS. Neither modified file is itself one of the 10 protected substrate files.

### R6 — regression sweep

Everything the prior report certified still holds. All 12 requirements implemented, all 13 scenarios covered by a passing test. The prior round's non-vacuous bites remain non-vacuous (S-PPB-003 stray-decision typed rejection and S-PPB-004's validator clause were re-checked; S-PPB-001 went from weak to non-vacuous). The goroutine-leak machinery is untouched — `awaitGoroutineBaseline` still polls-until-settled, and exactly the same two tests (`..._PopulatesRejoinAndNoLeak`, `..._RTLS008_SourceGuard_...`) remain serial while every other test keeps `t.Parallel()`. `-race -count=15` on the whole package: clean.

### R7 — empirical honesty: the "100%" vs "~27%" question

Both figures were measured, and **both are right for different consumer shapes**; neither should stand unqualified.

```text
Pre-fix (c0fcc6e5, unmodified), UNBUFFERED sink held by a synchronous consumer
  — the exact shape the original W3 finding described:
  20/20 runs, an early WakeParked was rejected for the entire 300 ms window
  => early-wake failure rate 100%.  The prior report's "100%" is CORRECT
     for a consumer that holds the event off an unbuffered sink.

Pre-fix ordering (W3 defeat), BUFFERED sink + drain goroutine + <-ready handoff
  — the shape the remediation's own probe used, 20 full-package runs:
    TestPermission_DeferEmitsBeforePark ............................ 20/20 FAIL
    TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_...    2/20 FAIL
    TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin .....   2/20 FAIL
    TestPermission_WakeParked_SynchronousOnDecisionRequired_NoRetry .  0/20 FAIL
  => 10% for the racing shapes, same order as the remediation's ~27%.
```

**Resolution**: the pre-fix lost-wakeup rate is a function of sink buffering and consumer topology, not a single constant. It is deterministic (100%) when the consumer receives from an unbuffered `sink` and wakes on receipt; it drops to the order of 10–30% when a buffered sink plus an extra goroutine handoff gives the gate time to register first. **The precise figure is not the point** — the defect was real and structural in the deterministic shape, which is precisely the shape AG-13's consumer is expected to take.

---

## Spec Compliance Matrix

| Requirement | Scenario | Covering test | Result |
|---|---|---|---|
| R-APP-001 per-call gate | S-APP-001 sync `AllowOnce` → no event, executes | `TestPermission_ImmediateAllow_NoEvent` (now also asserts `resolveInvocations()==1`) | COMPLIANT |
| R-APP-001 | S-APP-002 `Defer` A + `AllowOnce` B → A parks, B completes | `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot` | COMPLIANT |
| R-APP-002 emit before park | S-APP-003 `decision_required` reaches sink BEFORE the parked wait blocks | `TestPermission_DeferEmitsBeforePark` | **PARTIAL** — proves delivery + registration-before-emission; no longer proves the ack ordering (W9) |
| R-APP-003 typed rejection | S-APP-004 wake unknown callID → typed rejection, no parked call touched | `TestPermission_StrayDecisionIsTypedError` + `..._UnknownCallID_TypedRejection_NoTouch` | COMPLIANT |
| R-APP-004 AllowOnce | S-APP-005 executes AND `decision_made{AllowOnce}` | `TestPermission_FourOutcomes/AllowOnce_...`; also `TestNoOpPermissionPolicy_AllowsEverySynchronously` | COMPLIANT |
| R-APP-005 Deny typed | S-APP-006 Deny → `ExecutionFailure` + typed `*Failure` | `TestPermission_FourOutcomes/Deny_...`, RTLS-008 `rtls_2`, and now `TestTurn_PermissionPolicy_E2E_DeferDenyModify` at `Turn()` level | COMPLIANT |
| R-APP-006 ModifyInput transparent | S-APP-007 `tool_start.Arguments()` byte-equals `decision_made.ModifiedArguments()` | `TestPermission_FourOutcomes/ModifyInput_...`, RTLS-008 `rtls_3`/`rtls_4`, and the new E2E (asserts both the stream bytes and `RecordedArgs()[0]`) | COMPLIANT |
| R-APP-007 AllowAlways + Remember | S-APP-008 `Remember=true` → 1 emission; `false` → 0 | `TestPermission_AllowAlways_Remember_Branches` (2 subtests) | COMPLIANT |
| R-APP-008 sibling isolation | S-APP-009 A parked, B read-class → B's Result in ordinal slot | `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot`; the new E2E extends it to `Turn()` | COMPLIANT |
| R-APP-009 cancel mid-park | S-APP-010 parked + cancel → typed abort, Schedule returns, no leak | `TestPermission_CancellationMidPark_PopulatesRejoinAndNoLeak`; `..._AbandonedAckWithCancel_GateDeregistersPromptly` adds the ack-path arm | COMPLIANT — strengthened |
| R-APP-010 remembered suppresses asks | S-APP-011 second identical call NOT consulted | `TestPermission_RememberedSuppressesSubsequentAsk` + `..._ConcurrentRace_AtMostOneEmission` | **COMPLIANT — upgraded from PARTIAL** (the concurrent lane is now correct by construction) |
| R-APP-011 Layer 2 owns protocol | S-APP-012 L3 impl consumed without naming the type | compile guards on `scriptedPermissionPolicy`, `wiringTestPolicy`, `NoOpPermissionPolicy`; enforced from the other side by `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` | COMPLIANT — strengthened |
| R-APP-012 substrate preservation | S-APP-013 files byte-unchanged; filter widened for new files | `TestTurn_SubstrateUntouched` + `TestTurn_PreRequestHook_SubstrateUntouched` + merge-base diff = 0 | COMPLIANT |

**Compliance summary**: 13/13 scenarios have a passing covering test. One is PARTIAL (S-APP-003, ordering half unguarded — W9). The prior round's PARTIAL (S-APP-011) is now fully COMPLIANT.

---

## RED-bite non-vacuity audit (defeat-tested by command, not by reading)

| Bite | Guards | Defeat test applied | Outcome |
|---|---|---|---|
| S-PPB-001 `TestPermission_ImmediateAllow_NoEvent` | the gate runs at all | delete the whole gate (`return true,nil,nil` at entry) | **FAIL** — `invocation count = 0, want 1`. **NON-VACUOUS** (was WEAK) |
| S-PPB-002 `TestPermission_DeferEmitsBeforePark` | registration-before-emission | move `park()` back after the ack | **FAIL 20/20** deterministically. **NON-VACUOUS** for the reorder |
| S-PPB-002 (same test) | the R-APP-002/D4 **ack** | delete `reqAck` + the dispatcher's `close(em.ack)` dependency | **PASSES** — 0/15 separate full-package runs, `-race -count=10` clean. **VACUOUS for the ack → W9** |
| S-PPB-003 `TestPermission_StrayDecisionIsTypedError` | `WakeParked` typed rejection | (unchanged this round) | NON-VACUOUS |
| S-PPB-004 `..._SecondEmissionRejected` | the `CardinalityAtMostOne` descriptor | (restored hand-built form) | NON-VACUOUS |
| NEW `..._ConcurrentRace_AtMostOneEmission` | the `rememberIfAbsent` CAS | ignore the CAS return value | **FAIL** — `count = 2, want exactly 1`. **NON-VACUOUS** |
| NEW `..._AbandonedAckWithCancel_GateDeregistersPromptly` | the cancellation-aware ack select | revert to bare `<-reqAck` | **FAIL** — spurious `nil` instead of `ErrStrayDecision`. **NON-VACUOUS** |
| NEW `..._SynchronousOnDecisionRequired_NoRetry` | (claims) the W3 reorder | move `park()` back after the ack | **PASSES 0/40 isolated, 0/20 full-package.** **VACUOUS → S6** |

---

## Correctness (Static Evidence)

| Requirement | Status | Notes |
|---|---|---|
| R-APP-001 | Implemented | `runPermissionGate`; nil-policy bypass preserved |
| R-APP-002 | Implemented | ack channel + `close(em.ack)` after `sink <- &stamped`; park **wait** still strictly after the ack |
| R-APP-003 | Implemented | `ErrStrayDecision`; `parkedSet.wake` lookup-then-delete under one lock |
| R-APP-004 | Implemented | unchanged |
| R-APP-005 | Implemented | unchanged, incl. the `errPermissionDeniedWithoutFailure` default |
| R-APP-006 | Implemented | one `modifiedArgs` drives both `ToolStart` and `Run`; now also proven at `Turn()` level |
| R-APP-007 | Implemented | `policy.Remember` → CAS-gated emission |
| R-APP-008 | Implemented | bounded read semaphore untouched by the CAS |
| R-APP-009 | Implemented | **strengthened** — cancellation now honoured on the ack path too; `parkedSet.remove` on every non-wake exit |
| R-APP-010 | Implemented | **upgraded** — concurrent lane correct by construction, not by validator backstop |
| R-APP-011 | Implemented | zero rule sets / mode flags in Layer 2; three external-package implementations |
| R-APP-012 | Implemented | 33 of 44 `src/agent` files byte-unchanged; `go.mod`/`go.sum` zero-diff |

`make lint`'s `unused` checker reporting `0 issues.` independently confirms no dead code was left behind by the `wakeParkedWithRetry` deletion or the `remember` → `rememberIfAbsent` replacement.

## Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| D1 `PermissionPolicy` interface | Yes | unchanged |
| D2 park locus inside scheduler | Yes | `parkedSet` per `Schedule` call, now with `remove` |
| D3 `SetCallID` before park | Yes | unchanged |
| D4 emit before park | Yes (implementation) | ack intact and correct; its **test guard** was lost — W9 |
| D5 ModifyInput transparency | Yes | now also proven through `Turn()` |
| D6 Deny → typed `Result`, not a Go error | Yes | unchanged |
| D7 per-call abort on the threaded `ctx` | Yes | **strengthened** — the ack wait now honours it too |
| D8 zero substrate edits | Yes | verified against merge-base |
| File Changes table (`Scheduler` fields in `scheduler.go`) | Deviated | accepted — Go declaration constraint, unchanged from prior report |
| Testing Strategy E2E row | Yes, with a proven deviation | `TestTurn_PermissionPolicy_E2E_DeferDenyModify`; Defer leg aborts by ctx-cancel because `Turn()` exposes no wake handle (D1 above) |
| Migration "no-op decorator in `agenttest`" | Delivered, relocated | `NoOpPermissionPolicy` in `package agent_test`; `agenttest` placement proven impossible (D2 above) |
| design.md not back-annotated with either deviation | Deviated | S7 |

---

## TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | Pass | Phase 6 (R1–R7) in `tasks.md` and `apply-progress.md`, plus Engram `#3039` |
| All tasks have tests | Pass | 26/26 completed tasks map to a test or a verified gate |
| RED confirmed (test files exist) | Pass | `loop_permission_e2e_test.go` (292 L) and `permission_policy_helpers_test.go` (106 L) both present |
| GREEN confirmed (tests pass now) | Pass | 3143 `--- PASS` / 1181 top-level, 0 `FAIL`, re-run by this phase |
| Triangulation adequate | Pass | four-outcome matrix (4 subtests), Remember branches (2), RTLS-008 (5 calls × 4 outcomes), E2E (3 outcomes) |
| Safety net for modified files | Pass | AG-09's suites green throughout; `-race -count=15` clean |
| RED-bite non-vacuity | **Pass with 1 loss and 1 weak** | 6 of 8 defeat tests fire; the ack guard was lost (W9) and the new W3 bite is redundant (S6) |

**TDD Compliance**: 6/7 checks fully passed, 1 partial.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|---|---|---|---|
| Unit (`Schedule`-level, `package agent_test`) | 17 top-level | `permission_protocol_test.go` | `go test -race` |
| Integration / E2E (full `Turn` through the `agenttest` scripted provider) | 3 top-level | `loop_permission_e2e_test.go`, `permission_policy_helpers_test.go`, `loop_tool_dispatch_test.go` | `agenttest` |
| **Total (this change)** | **20 top-level** (was 15) | **4** | |

### Assertion Quality

| File | Line | Assertion | Issue | Severity |
|---|---|---|---|---|
| `permission_protocol_test.go` | ~1108–1120 | single `WakeParked` == nil after `<-ready` | Passes identically with the reorder reverted — proves nothing the sibling tests do not | SUGGESTION (S6) |
| `permission_protocol_test.go` | 1196, 1202 | `time.Sleep(100 * time.Millisecond)` ×2 | Wall-clock margins in a `t.Parallel()` test | SUGGESTION (S8) |

The prior round's two entries are both resolved: `:225–242` gained the `resolveInvocations` assertion (S1) and `:503–505`'s `want 2` is gone (W2). No tautologies, no ghost loops, no orphan-empty assertions, no mock-heavy or smoke-only tests.

### Quality Metrics

**Linter**: `0 issues.` (`go vet` + `golangci-lint`, after `cache clean`) · **Build**: exit 0 · **Race detector**: 0 `DATA RACE` across 12 packages, `-count=15` on `src/agent`

---

## Issues Found

**CRITICAL**: None.

**WARNING (4)**:

- **W9 — NEW REGRESSION: the R-APP-002 / D4 ack lost its only non-vacuous test guard.**
  Deleting the ack outright — remove `reqAck`, send `emission{ev: reqEv}` with no ack, delete the wait — leaves the **entire `src/agent` package green**: `-race -count=10` in-process → `ok`, and 0/15 separate full-package processes failed. At `c0fcc6e5` the identical deletion failed deterministically:
  ```text
  permission_protocol_test.go:308: WakeParked succeeded before anything read from sink —
    the call parked before decision_required could have reached sink (R-APP-002/D4 ordering violated)
  --- FAIL: TestPermission_DeferEmitsBeforePark
  ```
  The rewrite of `TestPermission_DeferEmitsBeforePark` was *necessary* — its old proxy ("an early wake must keep failing") became the wrong invariant the moment registration moved ahead of emission — but the replacement asserts only that `decision_required` is still delivered. The ack ordering itself is now asserted by nothing.
  *Not CRITICAL because*: the ack is present and correct in production (verified by reading `runDispatcher` and `runPermissionGate`), D4 still holds, and S-APP-003 retains a passing covering test for the delivery half. *Why it matters*: a future refactor can now delete the ack and every gate stays green. A replacement bite must observe the *wait*, not the registration — e.g. assert that `parkCh` is not selected on until after a `sink` reader has taken the event, or expose a test-only ordering probe.

- **W5 — the delta spec is structurally unpromotable and the target capability directory does not exist.** Unchanged; archive-phase work. Re-confirmed by command: `ls openspec/specs/agent-permission-protocol` → *No such file or directory*, and `openspec/specs/` currently holds 67 capabilities, none of them `agent-permission-protocol`. The delta at `openspec/changes/cachicamas-agent-permission-protocol/specs/agent-permission-protocol/spec.md` is 32 lines with `### R-APP-NNN` headings = **0**, `#### Scenario` blocks = **0**, `## Coverage` sections = **0**, while all **12** `R-APP-NNN` and all **13** `S-APP-NNN` IDs are present. **Archive must produce**, not copy: (1) a new `openspec/specs/agent-permission-protocol/spec.md`; (2) one `### R-APP-NNN — <title>` heading per requirement, matching `openspec/specs/agent-tool-scheduler/spec.md`'s promoted form; (3) a `#### Scenarios` block under each requirement; (4) each requirement rewritten from a one-line table cell into a normative RFC 2119 paragraph; (5) a `## Coverage` charter→spec mapping table.

- **W6 — the delta spec's own format claim is false.** Unchanged. Header line 3 states *"**Format**: RFC 2119 + Given/When/Then"*; `grep -ciE '^\s*-?\s*(given|when|then)\b'` over the file → **0**. `openspec/config.yaml` `rules.specs` mandates Given/When/Then and independently verifiable scenarios. Inherited byte-identically from the Engram spec artifact `#3034`, so it is a spec-phase defect surfacing at verify. Archive must author the Given/When/Then scenarios as part of the W5 transform.

- **W7 — `make vuln-check` fails on 5 pre-existing Go stdlib advisories.** Unchanged and correctly characterised. Re-run: exit 2; `GO-2026-5026 GO-2026-5972 GO-2026-6089 GO-2026-6090 GO-2026-6218`, all Go stdlib at go1.26.5, all fixed in go1.26.6; `grep -c 'src/agent/'` over the output → **0**; every trace runs through `src/ai/openaicompat/**` (Layer 1). Pre-existing on `main`. Task 5.4 remains correctly `[ ]` with an accurate why-note. Accepted, out of AG-10 scope.

**SUGGESTION (3)**:

- **S6 — the test the remediation designates as the W3 RED bite does not actually catch the W3 revert.** `TestPermission_WakeParked_SynchronousOnDecisionRequired_NoRetry` passed **0/40** isolated runs and **0/20** full-package runs with the reorder reverted. Its doc comment claims *"RED before the fix: fails on the first (only) attempt with ErrStrayDecision"*, and `tasks.md` R1 records *"~27% failure over 30 runs pre-fix"*; neither reproduces for the test as committed, because `drainUntilDecisionRequired`'s buffered sink plus the extra `<-ready` goroutine handoff reliably gives the gate time to register first. The reorder is nonetheless properly guarded — deterministically, 20/20 — by `TestPermission_DeferEmitsBeforePark`. Either re-shape the no-retry test to an unbuffered synchronous consumer (where the pre-fix failure is 100%) or correct its doc comment to say it is a regression guard, not the RED bite.
- **S7 — `design.md` was not back-annotated with either accepted deviation.** Both deviations are well recorded in the test files and `tasks.md` Phase 6, but `design.md:76` still specifies a wake-resume E2E and `design.md:84` still says the decorator "goes in `agenttest`". Both are now provably impossible at this milestone (D1, D2). Two edited lines would stop the design table describing untakeable work.
- **S8 — two 100 ms wall-clock sleeps in `..._AbandonedAckWithCancel_GateDeregistersPromptly`** (`:1196`, `:1202`). The defeat test proves the assertion is not masked by them, and the test's comment argues correctly that the ack cannot close regardless of timing. Still, fixed wall-clock margins in a `t.Parallel()` test are a flake surface under loaded CI; a poll-until-deregistered loop bounded by a deadline would carry the same proof without the fixed cost.

---

## Task Checkbox Audit

| Metric | Value |
|--------|-------|
| Checkbox items total | 27 (20 original + 7 Phase 6 remediation) |
| Complete `[x]` | 26 |
| Incomplete `[ ]` | 1 — task 5.4 (`make vuln-check`), correctly unchecked with an accurate why-note |

All 26 `[x]` states are supported by code and by commands re-run in this phase. The Phase 6 entries R1–R7 each map to a verified change: R1→W3 reorder, R2→W4 ack select + `parkedSet.remove`, R3→CAS + restored validator bite, R4→S1 assertion, R5→S2 rationale, R6→E2E + `NoOpPermissionPolicy`, R7→substrate filter widening. **Flip nothing.** Two claims inside those entries are overstated and are corrected in this report rather than in the tasks file: R1's "~27% failure pre-fix" for the no-retry test (S6) and the Phase 6 gate table's "1181 `--- PASS`" (correct for `^--- PASS`; the inclusive count is 3143 — S5's mechanism, now documented).

---

## Verdict

**PASS WITH WARNINGS** — 0 CRITICAL, 4 WARNING, 3 SUGGESTION.

The remediation round did what it claimed for four of five assigned items and did it soundly: W1, W2, W3 and W8 are fully closed, W4 is closed for the narrowing AG-10 introduced with its residual honestly documented, and all five prior SUGGESTIONS are resolved. Every fix was defeat-tested by command — six of the eight bites in the suite fail when the implementation they guard is reverted. Both W8/S4 deviation claims were independently proven, the `agenttest` one by reproducing the Layer 1 boundary failure in an isolated module copy. All four gates AG-10 owns are green and were re-run by this phase; the substrate invariant holds with 33 of 44 files byte-unchanged.

**Nothing blocks archive.** One new warning is worth an explicit decision before archiving: **W9** — the R-APP-002/D4 ack is still correct but is no longer guarded by any test, a capability the pre-remediation suite had. The orchestrator may accept it as a follow-up or spend one more scoped remediation on a replacement bite; it does not invalidate any requirement.

Archive must still act on **W5** (author `openspec/specs/agent-permission-protocol/spec.md` in the AG-09 promoted form — headings, `#### Scenarios`, RFC 2119 paragraphs, `## Coverage`) and **W6** (supply the Given/When/Then scenarios `openspec/config.yaml` requires). **W7** stays accepted and out of scope. **W9**, the residual abandoned-sink hazard in R1/R2, and the fact that `Turn()` exposes no wake handle are the three explicit inputs AG-13 should carry forward.
