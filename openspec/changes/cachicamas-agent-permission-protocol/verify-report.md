```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:7861db98a3c7ad09579cd7f622b94f7e746c24400bc69cfcc5dc963fd9f75ec7
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 12/12
scenarios: 13/13
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:5bb9d3b2f1d2b27bd8048448392695b01858e2ff7c5b9ddf82473dcd0ba9e708
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report

**Change**: `cachicamas-agent-permission-protocol` (AG-10, Layer 2 Wave 2, milestone 10/24)
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag10` @ `c0fcc6e5` (clean), base `main` `6de08335`
**Mode**: Strict TDD · **Store**: hybrid
**Verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 8 WARNING, 5 SUGGESTION

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 20 |
| Tasks complete (`[x]`) | 19 |
| Tasks incomplete (`[ ]`) | 1 (5.4 — `make vuln-check`, out-of-scope with why-note) |
| Requirements | 12 (`R-APP-001..012`) |
| Scenarios | 13 (`S-APP-001..013`) + 4 bites (`S-PPB-001..004`) |

Every checkbox state is justified by code re-verified by command. **No checkbox needs flipping.**
Task 5.4 is correctly left `[ ]`: `make vuln-check` genuinely fails, and the annotation is accurate.

### Build & Tests Execution (verbatim, re-run by this phase)

**Tests**: PASS — 3138 `--- PASS`, 0 `FAIL`, 12/12 packages `ok`, 0 `DATA RACE`

```text
$ cd backend/agent && make test
TEST_EXIT=0
PASS_LINES=3138
FAIL_LINES=0
OK_PKGS=12
RACES=0
```

**Build**: PASS

```text
$ cd backend/agent && make build
go build -trimpath ./...
BUILD_EXIT=0
```

**Lint**: PASS (after `./bin/golangci-lint cache clean`)

```text
$ cd backend/agent && make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
LINT_EXIT=0
```

**Vuln-check**: FAIL — accepted, out of AG-10 scope (see W7)

```text
$ cd backend/agent && make vuln-check
VULN_EXIT=2
GO-2026-5026 GO-2026-5972 GO-2026-6089 GO-2026-6090 GO-2026-6218
AGENT_TRACES=0        # zero traces through src/agent/**
```

**Load / flakiness** (orchestrator-requested):

```text
$ go test -race -count=10 -run TestPermission ./src/agent/
ok  	github.com/cachicamas/backend/agent/src/agent	4.539s
EXIT=0
```

**Substrate (NFR-TLS-003 7th carry)**:

```text
$ git diff --stat main...HEAD -- <the 10 substrate files>
(no output — zero substrate edits)

$ go test -run TestTurn_SubstrateUntouched -v ./src/agent/
--- PASS: TestTurn_SubstrateUntouched (0.04s)
ok  	github.com/cachicamas/backend/agent/src/agent	0.486s
```

**Coverage**: skipped — no coverage target configured in `backend/agent/Makefile`.

### Spec Compliance Matrix

| Requirement | Scenario | Covering test | Result |
|---|---|---|---|
| R-APP-001 per-call gate | S-APP-001 sync AllowOnce → no event, executes | `permission_protocol_test.go:202` `TestPermission_ImmediateAllow_NoEvent` | COMPLIANT |
| R-APP-001 | S-APP-002 Defer A + AllowOnce B → A parks, B completes | `:762` `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot` | COMPLIANT |
| R-APP-002 emit before park | S-APP-003 decision_required reaches sink BEFORE the parked wait blocks | `:272` `TestPermission_DeferEmitsBeforePark` | COMPLIANT |
| R-APP-003 typed rejection | S-APP-004 wake unknown callID → typed rejection, no parked call touched | `:391` `TestPermission_StrayDecisionIsTypedError` + `:1137` `..._UnknownCallID_TypedRejection_NoTouch` | COMPLIANT |
| R-APP-004 AllowOnce | S-APP-005 executes AND decision_made{AllowOnce} | `:566` `TestPermission_FourOutcomes/AllowOnce_EmitsDecisionMade` | COMPLIANT |
| R-APP-005 Deny typed | S-APP-006 Deny (sync OR wake) → ExecutionFailure + typed `*Failure` | `:566` `.../Deny_TypedResultAndDecisionMade` (sync) + `:1554` RTLS-008 slot `rtls_2` (wake) | COMPLIANT |
| R-APP-006 ModifyInput transparent | S-APP-007 `tool_start.Arguments()` byte-equals `decision_made.ModifiedArguments()` | `:566` `.../ModifyInput_SubstitutesArguments` + RTLS-008 `rtls_3`/`rtls_4` | COMPLIANT |
| R-APP-007 AllowAlways + Remember gate | S-APP-008 Remember=true → 1 emission; false → 0 | `:1281` `TestPermission_AllowAlways_Remember_Branches` (2 subtests) | COMPLIANT |
| R-APP-008 sibling isolation | S-APP-009 A parked, B read-class AllowOnce → B's Result in ordinal slot | `:762` `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot` | COMPLIANT |
| R-APP-009 cancel mid-park | S-APP-010 two parked, cancel → both typed abort, Schedule returns, no leak | `:892` `TestPermission_CancellationMidPark_PopulatesRejoinAndNoLeak` | COMPLIANT |
| R-APP-010 remembered suppresses asks | S-APP-011 second identical call NOT consulted, no decision_required | `:1377` `TestPermission_RememberedSuppressesSubsequentAsk` | PARTIAL — sequential lane only (see W1) |
| R-APP-011 Layer 2 owns protocol | S-APP-012 L3 impl consumed without naming the type | `:535` `var _ agent.PermissionPolicy = (*scriptedPermissionPolicy)(nil)`; `loop_tool_dispatch_test.go` `wiringTestPolicy` + `TestTurn_PermissionPolicy_WiredToSchedule` | COMPLIANT |
| R-APP-012 substrate preservation | S-APP-013 10 files byte-unchanged; filter widened for the 2 new files | `loop_test.go` `TestTurn_SubstrateUntouched` + `git diff` = 0 lines | COMPLIANT |

**Compliance summary**: 13/13 scenarios have a passing covering test; 1 (S-APP-011) is PARTIAL for the concurrent read-class lane.

### RED-bite non-vacuity audit (the audit-#3038 discriminating question)

The re-validation audit found 3 of 4 original bites vacuous under *"would this still pass if the implementation it guards were deleted?"*. Re-applying that test to the replacements:

| Bite | Guards | Survives deletion of what it guards? | Verdict |
|---|---|---|---|
| S-PPB-001 `TestPermission_ImmediateAllow_NoEvent` | AllowOnce bypasses the gate | **Yes** — deleting the whole gate (nil-policy bypass) also yields 0 `decision_required`, 1 invocation, Success. It never asserts `policy.Resolve` was called. | WEAK (see S1) — but the property is separately pinned by `TestPermission_FourOutcomes/AllowOnce_...` (asserts `decision_made`) and `TestTurn_PermissionPolicy_WiredToSchedule` (Deny policy → tool not invoked). No scenario loses coverage. |
| S-PPB-002 `TestPermission_DeferEmitsBeforePark` | the R-APP-002 ack ordering | **No** — with the ack removed the gate parks while the dispatcher is still blocked on the unbuffered `sink`, so the 300 ms `WakeParked` poll succeeds and `t.Fatal` fires (`:308`). | NON-VACUOUS |
| S-PPB-003 `TestPermission_StrayDecisionIsTypedError` | `Scheduler.WakeParked` typed rejection | **No** — re-pointed at the real production surface (`scheduler.go:235`); the deleted `ResolveStrayDecision` stub is gone. Its sibling `..._NoTouch` proves the second clause (a genuinely parked sibling stays parked). | NON-VACUOUS (defect 3 fixed) |
| S-PPB-004 `TestPermission_RememberedCardinality_SecondEmissionRejected` | the `CardinalityAtMostOne` descriptor | **No** for the validator — deleting `Cardinality: CardinalityAtMostOne` (`event.go:322`) makes `CheckStream` accept and `t.Fatal` fires (`:525`). | NON-VACUOUS, but see W2 — it also asserts the *scheduler defect* still happens. |

Defect 4 (`policy.Remember` never called) is fixed: `scheduler.go:658` is a real production call site, guarded by `TestPermission_AllowAlways_Remember_Branches` asserting `rememberInvocations() == 1` and the tool name/outcome arguments.

### Orchestrator-flagged scrutiny items

#### 1. The deliberately-unfixed concurrent-remember race — the apply agent's argument only half-holds

**R-APP-010's literal wording**: *"When `Policy.Remember` returned `true` for a `toolName`, identical **subsequent** calls in the same run MUST NOT be asked."*

`subsequent` is temporal and predicated on `Remember` having already returned `true`. In the forced-race
(`permission_protocol_test.go:430-454`) both goroutines pass `remembered.remembered()` (`scheduler.go:563`)
*before* either `Remember` returns, so neither call is "subsequent" to the other's remembering.
**On R-APP-010 alone, the apply agent is right: not a violation.** Likewise R-APP-007 (`scheduler.go:658-673`)
literally says "`true` emits" and the code does exactly that.

**But the `CardinalityAtMostOne` backstop claim is false in production.** Verified by command:

```text
$ grep -rn "CheckStream(" src/ | grep -v "_test.go"
src/agenttest/fake_provider.go:105:     ...ai.CheckStream(stepEvents(steps))...
src/agenttest/stream_kit_ordering.go:25: ...ai.CheckStream(events)...
src/agent/stream_check.go:92:func CheckStream(events []Event) StreamReport {
src/ai/stream_check.go:64:func CheckStream(events []Event) StreamReport {
```

Both non-definition call sites are inside `src/agenttest/**` — the **test harness**. `agent.CheckStream`
has **zero production invocations**. So in a real run the invalid stream is not rejected, not logged, and
not surfaced: it is emitted straight to the consumer. The backstop fires only when a test happens to call
`CheckStream`. See W1.

#### 2. Defect 6 — all four branches confirmed fixed by reading

| Outcome | Site | Shape |
|---|---|---|
| `AllowOnce` | `scheduler.go:626-633` | `err != nil` → `typedExecutionFailureFromError` → `results[ordinal] = abort` → `emitExecutionFailure` → `return false` |
| `AllowAlways` | `scheduler.go:645-656` | identical, and it returns **before** `policy.Remember` is invoked |
| `Deny` | `scheduler.go:684-695` | identical |
| `ModifyInput` | `scheduler.go:720-735` | identical — the one with a RED test (`:1720`) |

All four are structurally identical five-line blocks. **The three untested branches are equivalent enough
to be safe**: they share the same constructor (`NewPermissionDecisionMade`), the same error variable, and
the same recovery statements in the same order. The apply agent's stated reason for not RED-testing them
is also correct — only `ModifyInput` has a *policy-controlled* way to force the constructor to fail
(empty `ModifiedArgs`); the others would need an empty `runID`/`turnID` from the `Schedule` **caller**,
which no policy can inject. One residual asymmetry remains at `scheduler.go:669-672` (see S2).

#### 3. The R-APP-002 ack — happens-before holds; **no deadlock is reachable**

Mechanism: `emission.ack` (`scheduler.go:82`); dispatcher closes it only after `sink <- &stamped` returns
(`scheduler.go:261-267`); the gate blocks on `<-reqAck` (`scheduler.go:592-594`) before `parked.park()`
(`scheduler.go:596`).

*Happens-before*: R-APP-002 requires "emission **reaches `sink`** before the parked wait blocks". A
completed `sink <- &stamped` means the value is in `sink` (delivered for an unbuffered channel, enqueued
for a buffered one). `close(ack)` is sequenced after it, and `<-reqAck` after that, and `park()` after
that. **The required happens-before is genuinely established.**

*Deadlock*: the complete wait graph for the new edge is

```
gate goroutine G  --(<-reqAck)-->  dispatcher D  --(sink <- ev)-->  external consumer C
```

`D` never waits on `G` (it only ranges over `emissions`, which it exclusively reads, and sends on `sink`).
`G`'s earlier `emissions <- ...` can block only when the `len(calls)*2` buffer is full, which again waits
on `D`, not the reverse. **There is no internal cycle, so no internal deadlock is reachable.** A hang
requires `C` to stop consuming `sink` — a caller-contract violation that already hung `Schedule`
pre-AG-10 (any emission would block `D` once the buffer filled). One genuine narrowing does exist: see W4.

#### 4. The bounded retry — sound as a test primitive, but it documents a production hazard

`wakeParkedWithRetry` (`permission_protocol_test.go:1011-1023`) is a **test helper**, bounded at 1 s, and
it calls `t.Fatalf` if it never succeeds. It therefore cannot mask a genuine hang and is a legitimate
test-side synchronization for an asynchronous registration — **not sleep-and-hope**.

But it exists because of a real, and *structurally guaranteed*, production gap. Because `close(ack)`
is sequenced **after** `sink <- &stamped`, at the instant a synchronous consumer holds the
`decision_required` event the gate goroutine has provably **not yet** executed `parked.park()`. So a
consumer that wakes immediately on receipt does not merely *sometimes* lose the wake — it loses it
**every time**. No production retry exists; only the test has one. See W3.

#### 5. The goroutine-leak baseline — not flaky

Two tests drop `t.Parallel()` deliberately (`TestPermission_CancellationMidPark_PopulatesRejoinAndNoLeak:892`,
`TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin:1554`); every other test in the file keeps it.
The reasoning at `:1546-1553` is correct: Go parks top-level `t.Parallel()` tests until all sequential
top-level tests have finished, so a serial test does run without sibling goroutine churn.
`awaitGoroutineBaseline` (`:1520`) polls-until-settled rather than sampling once, which is the specific
fix for what AG-09 reverted. Empirically confirmed: `go test -race -count=10 -run TestPermission ./src/agent/`
→ `ok ... 4.539s`, exit 0, zero flakes. **The AG-09 raciness has not returned.**

#### 6. The two design deviations — both acceptable, neither is drift

| Deviation | Judgement |
|---|---|
| `Scheduler.parkedMu` / `parked` in `tool.go:230-233` rather than `scheduler.go` | **Acceptable.** `design.md`'s File Changes table assigns *behaviour* to `scheduler.go`, and `WakeParked` (the behaviour) is there (`scheduler.go:235`). The `Scheduler` **struct** is declared in `tool.go` (AG-09.1's file); Go requires fields to live with the declaration. Following the table literally was impossible. |
| `openspec/AGENTS.md` had no AG-07/08/09 pointer precedent | **Acceptable, and the claim is verified.** `git show main:openspec/AGENTS.md \| grep "AG-0[6789]\|AG-1[0-9]"` returns nothing. The added section (18 lines) states honestly that AG-10 is the first such pointer rather than fabricating continuity. Task 1.3 is genuinely done. |

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|---|---|---|
| R-APP-001 | Implemented | `runPermissionGate` `scheduler.go:542`; called from `executeCall:401` for every scheduled call; nil-policy bypass at `:555` |
| R-APP-002 | Implemented | ack channel `scheduler.go:82`, `:262-267`, `:592-596` |
| R-APP-003 | Implemented | `ErrStrayDecision` `permission_protocol.go:58`; `WakeParked` `scheduler.go:235-243`; `parkedSet.wake` lookup-then-delete under one lock `permission_protocol.go:184-195` |
| R-APP-004 | Implemented | `scheduler.go:625-636` |
| R-APP-005 | Implemented | `scheduler.go:683-713`, incl. a defensive `errPermissionDeniedWithoutFailure` default at `:707` |
| R-APP-006 | Implemented | `scheduler.go:719-738` + `executeCall:424-426, :443-446` (both `ToolStart` and `Run` input rewritten from the same `modifiedArgs`) |
| R-APP-007 | Implemented | `scheduler.go:644-675` |
| R-APP-008 | Implemented | bounded read semaphore is not exclusive (`scheduler.go:298`); park holds a slot but siblings acquire others |
| R-APP-009 | Implemented | `select { <-parkCh; <-ctx.Done() }` `scheduler.go:597-617`; abort populates the ordinal slot at `:613-615` |
| R-APP-010 | Implemented (sequential) | pre-`Resolve` suppression `scheduler.go:563-565`; `rememberedSet` `permission_protocol.go:208-234`. Concurrent lane: W1 |
| R-APP-011 | Implemented | interface at `permission_protocol.go:80-94`; zero rule sets / mode flags in Layer 2; two external-package implementations consume it |
| R-APP-012 | Implemented | zero-line diff on all ten files; filter widened in `loop_test.go` |

**No requirement in the spec is silently unimplemented.** Every `R-APP-NNN` has a production code path
and at least one passing test. The dead code the audit found (`ResolveStrayDecision`, `errParkedSetShutdown`,
`closeAll()`, the `wakeErr != nil` branch, the literal `nil` policy at `loop.go:247`) is all gone —
`make lint`'s `unused` checker reporting `0 issues.` is independent confirmation.

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| D1 `PermissionPolicy` interface (`Resolve`+`Remember`) | Yes | `permission_protocol.go:80-94`, exact signatures |
| D2 park locus inside scheduler | Yes | `parkedSet` per `Schedule` call |
| D3 `SetCallID` before park (R-TLS-009) | Yes | `executeCall:389`, before the gate at `:401` |
| D4 emit before park | Yes | strengthened with the ack; see item 3 |
| D5 ModifyInput transparency | Yes | one `modifiedArgs` drives both `ToolStart` and `Run` |
| D6 Deny → `Result{ExecutionFailure, typedDenial}`, not a Go error | Yes | `scheduler.go:697-712` |
| D7 per-call abort on the threaded `ctx` | Yes | `ctx` is now a real parameter, not `_` |
| D8 zero substrate edits | Yes | verified by `git diff` and by test |
| Data flow "wake: re-evaluate verdict" | Yes | tail-recursion into `runPermissionGate` `scheduler.go:608` |
| File Changes table (`scheduler.go` for the Scheduler fields) | Deviated | acceptable — Go declaration constraint, see item 6 |
| Testing Strategy: E2E "scripted fake provider drives a turn" | Partially | `TestTurn_PermissionPolicy_WiredToSchedule` drives one Deny through a real `Turn` + `agenttest` provider, but not the "one each: deferred, denied, modified" full-stream E2E the table specifies. See S3. |
| Migration: "a no-op pass-through decorator goes in `agenttest`" | Not done | no `agenttest` decorator was added; nil-policy bypass serves the same purpose. See S4. |

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | Pass | "TDD Cycle Evidence" present in Engram `#3039` and in `apply-progress.md` |
| All tasks have tests | Pass | 19/19 completed tasks map to a test file or a verified gate |
| RED confirmed (test files exist) | Pass | `permission_protocol.go` (235 L), `permission_protocol_test.go` (1761 L), `loop_tool_dispatch_test.go` (+124 L) all present |
| GREEN confirmed (tests pass now) | Pass | 3138 `--- PASS`, 0 `FAIL`, re-run by this phase |
| Triangulation adequate | Pass | `TestPermission_FourOutcomes` 4 subtests; `..._Remember_Branches` 2 subtests; RTLS-008 5 calls x 4 outcomes |
| Safety net for modified files | Pass | `scheduler.go`/`loop.go`/`tool.go` modified with AG-09's existing suites green throughout |
| RED-bite non-vacuity | Pass with 1 weak | 3/4 bites fail on deletion of what they guard; S-PPB-001 does not (S1) |

**TDD Compliance**: 6/7 checks fully passed, 1 partial.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|---|---|---|---|
| Unit (`Schedule`-level, `package agent_test`) | 14 top-level (`TestPermission_*`), 8 subtests | `permission_protocol_test.go` | `go test -race` |
| Integration (full `Turn` through `agenttest` provider) | 1 (`TestTurn_PermissionPolicy_WiredToSchedule`) | `loop_tool_dispatch_test.go` | `agenttest` |
| E2E (real process/browser) | 0 | — | N/A for a Go library layer |
| **Total (this change)** | **15 top-level** | **2** | |

### Assertion Quality

| File | Line | Assertion | Issue | Severity |
|---|---|---|---|---|
| `permission_protocol_test.go` | 225-242 | 0 `decision_required` + 1 invocation + Success | Bite does not assert `policy.Resolve` was reached; survives deletion of the whole gate | SUGGESTION (S1) |
| `permission_protocol_test.go` | 503-505 | `rememberedCount != 2 → t.Fatalf` | Asserts the *defect* occurs; fixing W1 makes this test fail | WARNING (W2) |

No tautologies, no ghost loops, no orphan-empty assertions, no mock-heavy tests, no smoke-only tests found.

### Quality Metrics

**Linter**: `0 issues.` (`go vet` + `golangci-lint`, after `cache clean`)
**Type checker / build**: `go build -trimpath ./...` exit 0
**Race detector**: 0 `DATA RACE` across 12 packages and a `-count=10` permission-suite run

### Issues Found

**CRITICAL**: None.

**WARNING**:

- **W1 — the concurrent-remember race emits a stream that violates the project's own event grammar, and nothing in production catches it.**
  `scheduler.go:563` (pre-`Resolve` suppression check) and `scheduler.go:658-659` (`policy.Remember` → `remembered.remember`) are not atomic.
  *Failure scenario*: a model returns two parallel `read_file` calls in one turn; both are read-class so both run concurrently (`scheduler.go:191`); both pass `remembered.remembered("read_file") == false`; both reach `AllowAlways`; a Layer 3 policy that persists the rule returns `true` twice; both emit `permission_resolution_remembered{tool_name:"read_file"}`. The turn's stream now carries two events of a kind declared `Cardinality: CardinalityAtMostOne` (`event.go:322`). `agent.CheckStream` has **zero production call sites**, so nothing rejects, logs, or surfaces it — the malformed stream reaches the consumer. The apply agent's claim that "the `CardinalityAtMostOne` validator backstops this" is therefore **incorrect for production**; the backstop is test-time only.
  *Note on scope*: `R-APP-010`'s "subsequent" wording does not literally cover this, and `R-APP-007` literally says "`true` emits", so this is **not** a violation of the letter of either requirement — hence WARNING, not CRITICAL. It **is** in tension with the NFR block's "`CardinalityAtMostOne` bite (R-APE-003 / S-APE-082 carry)".
  *On the stated reason for not fixing it*: the apply agent wrote that a fix "would risk changing AG-09.2's read-class concurrency semantics". That does not hold up. A compare-and-set inside the existing mutex in `rememberedSet` (`permission_protocol.go:230-234`) — return `false` when the name was already present, and emit only on `true` — is a two-line change that touches no scheduling code and no AG-09.2 lane. The real cost is W2.

- **W2 — the S-PPB-004 bite hard-codes the W1 defect as required behaviour.**
  `permission_protocol_test.go:503-505` fails with *"resolution_remembered count = %d, want 2"* whenever the count is not exactly 2. A correct scheduler that suppressed the duplicate would emit 1 and **fail this test**. The bite therefore prevents the W1 fix rather than merely documenting the gap. Any future fix must re-point this bite at hand-built events (which is what it did before commit `ba1e8777`) or at a policy-level double-emit.

- **W3 — the wake surface has a structurally guaranteed lost-wakeup window.**
  `scheduler.go:261-267` closes the ack only after `sink <- &stamped` completes, and `scheduler.go:592-596` parks only after `<-reqAck`.
  *Failure scenario*: a Layer 3 consumer reads `permission_decision_required` from `sink` and — being already-decided, e.g. an auto-approve policy or a replayed decision — immediately calls `sched.WakeParked(callID)`. At that instant the gate goroutine is still blocked on `<-reqAck`, so `s.parked.channels` has no entry, `WakeParked` returns `ErrStrayDecision` (`scheduler.go:239-241`), and the call then parks with no further wake pending. It stays parked until run-context cancellation. Because the ack ordering *forces* emission-strictly-before-park, this is not probabilistic — an immediate synchronous wake fails **100%** of the time.
  *Why it is not a spec violation*: R-APP-003 requires exactly this typed rejection for a callID "not in the parked set", and the spec never promises that a wake issued after observing `decision_required` succeeds. But AG-13 will consume this surface, and the only mitigation that exists today is a helper inside the test file (`wakeParkedWithRetry:1011`). Record it so AG-13 does not rediscover it as a hang.

- **W4 — `<-reqAck` is not cancellation-aware, narrowing R-APP-009's "no goroutine waits forever".**
  `scheduler.go:594` is a bare `<-reqAck` with no `case <-ctx.Done()`; the `select` that honours cancellation only begins at `:597`, *after* the ack.
  *Failure scenario*: a consumer abandons `sink` (e.g. its own context is cancelled and it returns from its read loop) while a `Defer` verdict is in flight. The dispatcher blocks on `sink <- &stamped` (`:261`), the gate blocks on `<-reqAck` (`:594`), `wg.Wait()` (`:196`) never returns, `Schedule` never returns, `Turn` never returns. Cancelling the run context does **not** release it, because the goroutine is not yet in the `select`. Pre-AG-10 an abandoned `sink` also hung `Schedule` once the `len(calls)*2` buffer filled, so this narrows an existing hazard rather than creating a new class — but it now triggers on the **first** deferred call instead of after `2*len(calls)` emissions.

- **W5 — the on-disk delta spec is not in the form this repo promotes, and the target capability directory does not exist.**
  `openspec/changes/cachicamas-agent-permission-protocol/specs/agent-permission-protocol/spec.md` is 32 lines; `openspec/specs/agent-permission-protocol/` does not exist (`ls` → *No such file or directory*).
  Repo convention, verified against AG-09: commit `b2ab3867` ("AG-09 spec+design committed") wrote the **full 204-line capability spec directly to `openspec/specs/agent-tool-scheduler/spec.md`**, and the change folder's `specs/` held only the *cross-cut delta into a pre-existing capability* (`specs/agent-loop-skeleton/spec.md`, 45 lines). AG-06.1 followed the same shape (`specs/agent-event-envelope/spec.md` only).
  *Content-wise the delta is complete*: all 12 `R-APP-NNN` and all 13 `S-APP-NNN` IDs are present. *Structurally it is not promotable.* Exactly what is missing for archive:
  1. `openspec/specs/agent-permission-protocol/spec.md` does not exist and must be authored.
  2. Zero `### R-APP-NNN — <title>` requirement headings (main specs use one per requirement — `specs/agent-tool-scheduler/spec.md:25`).
  3. Zero `#### Scenarios` blocks (`grep -c "^#### Scenario"` → 0; AG-09's main spec has 11+).
  4. Requirements are compressed into one markdown table row each rather than a normative RFC 2119 paragraph.
  5. No `## Coverage` charter→spec mapping table (present in every promoted spec).
  The archive phase cannot mechanically copy this file; it must perform a real transform.

- **W6 — the delta spec's own format claim is false.**
  Its header (line 3) states **"Format: RFC 2119 + Given/When/Then"**, but the body contains no Given/When/Then scenario anywhere (`grep -ci "given\|when\|then"` → 3, all incidental prose). `openspec/config.yaml` `rules.specs` mandates *"Use Given/When/Then for scenarios"* and *"Each scenario MUST be independently verifiable"*. Scenario cells such as *"S-APP-001 sync `AllowOnce` A → no event, executes."* are shorthand, not verifiable Given/When/Then. Note this is inherited from the Engram spec artifact `#3034` (byte-identical content), so it is a spec-phase defect surfacing at verify, not an apply-phase regression.

- **W7 — `make vuln-check` fails (accepted, out of scope).**
  Exit 2 with 5 Go **stdlib** advisories at go1.26.5 — `GO-2026-6218`, `GO-2026-6090`, `GO-2026-6089`, `GO-2026-5972`, `GO-2026-5026`, all fixed in go1.26.6. Independently confirmed: `grep -c "src/agent/"` over the vuln output → **0**; every trace runs through `src/ai/openaicompat/**` (Layer 1). Pre-existing on `main`, not introduced by AG-10, and the spec's NFR "Verify" line does list `make vuln-check` green — so this is a genuine, knowingly-unmet acceptance item, correctly recorded as `[ ]` on task 5.4. A toolchain bump is out of AG-10's scope per explicit instruction.

- **W8 — the design's E2E testing row and its `agenttest` decorator are not delivered.**
  `design.md:76` specifies an E2E driving "one each: deferred, denied, modified" through a scripted fake provider and asserting the full stream `decision_required → decision_made → tool_start (modified args) → tool_end_*`. What exists is `TestTurn_PermissionPolicy_WiredToSchedule`, a single-Deny wiring proof. `design.md:84`'s "no-op pass-through decorator goes in `agenttest`" was also not added. Neither breaks a spec scenario (RTLS-008 covers the mixed-outcome matrix at `Schedule` level), hence WARNING not CRITICAL.

**SUGGESTION**:

- **S1** — Strengthen S-PPB-001 (`permission_protocol_test.go:202-243`) with `if n := policy.resolveInvocations(); n != 1 { ... }`. One line makes the bite fail when the gate is deleted. Today it does not.
- **S2** — `scheduler.go:669-672` still uses the swallowing `if rerr == nil { emit }` shape that defect 6 removed from the other four sites. The inline rationale (best-effort telemetry, decision already made) is defensible, but it is the last instance of a pattern the milestone otherwise eliminated; a short comment cross-referencing defect 6 or a `slog` warn would close the inconsistency.
- **S3** — Add the `design.md:76` E2E as a follow-up (or amend the design to record that RTLS-008 subsumes it), so the design table stops describing untaken work.
- **S4** — Either add the `agenttest` no-op decorator `design.md:84` promises or strike the line; the nil-policy bypass already serves the purpose.
- **S5** — `apply-progress` reports "1176 `--- PASS` lines"; this phase measured **3138** on a full uncached `make test`. Both runs exit 0 with 0 `FAIL`, so the discrepancy is almost certainly package caching during the apply run. Worth recording the counting method alongside the number in future artifacts.

### Task Checkbox Audit

All 20 checkbox states are supported by the code. **Flip nothing.**

| Task | State | Code support |
|---|---|---|
| 1.1 | `[x]` | `permission_protocol.go` exists with interface, verdict, `parkedSet`, `rememberedSet` |
| 1.2 | `[x]` | `TestPermission_FourOutcomes` + `..._Remember_Branches` are both table-driven `t.Run` |
| 1.3 | `[x]` | `openspec/AGENTS.md` +18 lines; the "no prior precedent" note is verified true |
| 2.1 | `[x]` | test exists and passes (weak — S1, but present and honest) |
| 2.2 | `[x]` | `TestPermission_DeferEmitsBeforePark`, non-vacuous |
| 2.3 | `[x]` | re-pointed at real `WakeParked`; stub deleted |
| 2.4 | `[x]` | re-pointed at a real `Schedule` run (but see W2) |
| 3.1 | `[x]` | `runPermissionGate` immediate + defer + recursion on wake |
| 3.2 | `[x]` | four outcomes implemented; defect 6 fixed on all four |
| 3.3 | `[x]` | `select { <-parkCh; <-ctx.Done() }`; dead sweep removed |
| 3.4 | `[x]` | `policy.Remember` called at `scheduler.go:658`; suppression at `:563` |
| 4.1 | `[x]` | `Schedule` signature carries `policy`; AG-09 sub-paths byte-clean |
| 4.2 | `[x]` | `loop.go:265` forwards `opts.PermissionPolicy`; the literal `nil` is gone |
| 4.3 | `[x]` | `TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin`, `-count=10` clean |
| 4.4 | `[x]` | `TestTurn_SubstrateUntouched` PASS with the widened filter |
| 5.1 | `[x]` | re-run: exit 0, 3138 PASS, 0 FAIL, `-race` |
| 5.2 | `[x]` | re-run after `cache clean`: `0 issues.` |
| 5.3 | `[x]` | re-run: exit 0 |
| 5.4 | `[ ]` | **correctly unchecked** — `make vuln-check` exit 2; the why-note is accurate |
| 5.5 | `[x]` | `git diff main...HEAD` over the 10 files: 0 lines |

### Verdict

**PASS WITH WARNINGS** — 0 CRITICAL, 8 WARNING, 5 SUGGESTION.

Every one of the 12 requirements is implemented in production code and covered by a passing test; all
four gates the milestone owns (`test`, `lint`, `build`, substrate) are green and were re-run by this
phase; every defect the re-validation audit `#3038` listed is fixed and the three vacuous bites are now
non-vacuous. **Nothing blocks archive.**

The two warnings the archive phase must act on before promoting are **W5** (author
`openspec/specs/agent-permission-protocol/spec.md` in the AG-09 promoted-spec form) and **W6**
(supply the Given/When/Then scenarios `openspec/config.yaml` requires). **W1**, **W3**, and **W4** are
correctness gaps that do not violate the letter of any `R-APP-NNN` requirement but should be carried
forward as explicit inputs to AG-13, which owns the consumer side of this protocol.
