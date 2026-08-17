```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:abc5b3120f0b7177f17ed414d171d694dbb043768d9183a3d14c8c2cc4a6eec7
verdict: fail
blockers: 3
critical_findings: 0
requirements: 17/17
scenarios: 36/41
test_command: cd backend/agent && go clean -testcache && go test -race -v -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:4ec6ab3658131d6e77a0bcd0e01d3faddcf1e480abbaf6bf967a3d62aed24bf2
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report

**Change**: `cachicamas-agent-cancellation-tree` (AG-14 — Build the cancellation tree, Layer 2 Wave 3, `0003:1371-1442`)
**Branch / worktree**: `feat/agent-layer2-wave3-ag14` @ `1ec63ccb`, merge base `52701436`
**Version**: 6 commits ahead of `origin/main`; working tree clean
**Mode**: Strict TDD

Every claim below was re-run by this phase. `apply-progress.md` was treated as a hypothesis, not as evidence.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 57 |
| Tasks complete | 56 |
| Tasks incomplete | 1 (`13.5`, explicitly reserved for `sdd-archive`) |

Verified by `grep -c "^- \[x\]" tasks.md` → 56; the single `^- \[ \]` is line 152, the archive hand-off note. Not a defect.

### Build & Tests Execution

**Build**: PASS — `cd backend/agent && make build` (`go build -trimpath ./...`), exit 0, empty output.

**Tests**: PASS — `go clean -testcache && go test -race -v -count=1 ./...`, exit 0.

```text
--- PASS lines: 3266
--- FAIL lines: 0
DATA RACE occurrences: 0
all 12 module packages report ok
```

The first `make test` invocation returned entirely from the Go build cache (`ok ... (cached)`). It was discarded and re-run with `go clean -testcache` + `-count=1`, so the green above is a genuine uncached `-race` execution.

**Lint**: PASS — `golangci-lint cache clean && make lint` → `go vet ./...` clean, `golangci-lint run --config=.golangci.yml ./...` → `0 issues.` The cache was cleaned first, per this repo's known phantom-finding hazard.

**Vulnerabilities**: PASS — `make vuln-check` (`govulncheck -json ./...`) exit 0, zero `"finding"` entries.

**Coverage**: `loop.go` = **88.01%** (771/876 statements), threshold 80% → ABOVE.
Computed independently from `go test -race -coverprofile=... -covermode=atomic ./src/agent/...` and summing the profile's `loop.go` blocks. The new cancellation branch is genuinely exercised: profile blocks `loop.go:442.2`, `442.100`, `443.67`, `444.92`, `448.3` all carry non-zero hit counts (4–6 hits each).

`TestTurn_CoverageGate` reports `--- SKIP`. That skip is pre-existing (`loop_test.go:1236-1243`, S-LSK-007, "coverage gate is enforced by `make test/cover`"), not introduced by AG-14. The gate was therefore enforced here by direct measurement rather than by that test.

### Hard pins — each verified by command

| Pin | Command | Result |
|---|---|---|
| `S-LSK-018` file set | `git diff --name-only origin/main -- backend/agent/src/agent/ \| grep -v _test.go` | Exactly `{cancellation.go, doc.go, harness.go, loop.go, run_events.go, scheduler.go, tool.go}` — the six pinned pre-existing files plus the one new file. No seventh. PASS |
| `run_events.go` minimality | `git diff origin/main -- .../run_events.go` | Adds only the `RunOutcomeShutdown` member (+7 lines of doc comment) and its `String()` case (+2). `RunEnd.validate` and `NewRunEnd` byte-unchanged. PASS |
| Forbidden substrate byte-unchanged | per-file `git diff --stat origin/main` | `turn_events.go`, `failure.go`, `stream_check.go`, `stream_check_test.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `reconstruction_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `history.go`, `permission_protocol_test.go` — all zero diff. PASS |
| `go.mod` / `go.sum` | `git diff --stat origin/main -- backend/agent/go.mod backend/agent/go.sum` | Empty. PASS |
| `backend/agent/src/ai/**` | `git diff --stat origin/main -- backend/agent/src/ai/` | Empty. PASS |
| `S-LSK-019` filters | extracted every `strings.HasSuffix(path, "...")` from `filterOutLoopFiles` and `filterOutLoopHookFiles`, `diff`'d | Identical, 35 entries each, every entry an exact filename suffix beginning `/`; no wildcard, prefix or directory pattern; `stream_check_test.go` absent from both. PASS |
| `permission_protocol_test.go` identity + green | zero diff (above) and `--- PASS` for `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` in the fresh run | PASS |
| `L2C-08` row pair | `git diff origin/main -- doc.go doc_contract_guard_test.go` | One `//\tL2C-08\t...` row in `doc.go` and one byte-matching `{id: "L2C-08", text: "..."}` entry in `expectedLayer2ContractRows`; every pre-existing row untouched; the guard passes in the fresh run. PASS |
| `S-APP-018` additive-only | `git diff origin/main -- scheduler_test.go \| grep -c "^-[^-]"` | `0` — purely additive, zero pre-existing lines touched. PASS |

### Pre-existing `gofmt` drift — confirmed NOT an AG-14 defect

Extracted all 55 `.go` files of `src/agent/` **from `origin/main`** via `git show` into a scratch directory and ran `gofmt -l` on them in isolation: **18 of 55 are gofmt-dirty at the merge base**, including `loop.go`, `scheduler.go`, `tool.go`, `permission_protocol_test.go`, `reconstruction_test.go` and `event_registry_test.go` — files AG-14 either does not touch or must keep byte-identical. The worktree count is also **18**. Every file AG-14 authored (`cancellation.go`, `cancellation_events_test.go`, `cancellation_interrupt_test.go`, `cancellation_shutdown_test.go`, `cancellation_winddown_test.go`) is gofmt-**clean**. Go 1.26.6. AG-14 introduced zero new drift and `make lint` passes regardless. Not reported as a defect.

### Charter conformance — judged against `0003:1391-1442` directly, not against the spec's restatement

| Charter Gherkin | Implementation evidence | Verdict |
|---|---|---|
| **AG-14.1 #1** "provider call cancelled per the fake's contract, in-flight tools observe cancellation, orphan synthesis runs, run ends interrupted" | `loop.go:442-449` cause check before the `turn.finish == 0` normalization; `scheduler.go:601` `tool.Run(ctx, args, policy)`; `harness.go` `windDownRun` synthesize→close→`run_end(Interrupted, nil)`. Both mechanisms proven, in two arms. | MET (split, see MAJOR-2/3) |
| **AG-14.1 #1** "a new prompt on the same harness works afterward" | `steeringQueue.reopen()` at `Run` entry; `TestHarness_Interrupt_SameHarnessRunsNextPrompt` proves run 2 completes, `Steer` returns nil, the steered message reaches `requests[2]`, and two turn brackets are emitted. | MET |
| **AG-14.1 #2** "suspension aborts typed, history closes cleanly" | `scheduler.go:756,789` both abort arms now build from `context.Cause(ctx)`; `TestHarness_Interrupt_DuringSuspensionAbortsTyped` asserts category `Cancellation`, unwrap to `ErrInterrupted`, `WakeParked` → `ErrStrayDecision`, `CloseTurn()` nil. | MET |
| **AG-14.1 #3** "nothing changes and nothing panics" | `signalMu`-guarded state; no channel is closed by a signal. `TestHarness_Interrupt_SecondInterruptIsNoOp` fires two `Interrupt()` from independent goroutines under `-race`; 0 races in the fresh run. | MET |
| **AG-14.2** "wind-down as interrupt, outcome says shutdown, subsequent prompts fail typed, distinguishable through every layer" | `RunOutcomeShutdown` on the stream; `ErrPromptAfterShutdown` wrapping `ErrShutdown` in the error chain. `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts` asserts both nouns and asserts the second `Run` emits **zero** events. | MET |
| **AG-14.3** "it still ends within the documented bound" | `runToolWithWindDown` timer arm; the test observes the **return** on `resultCh`, never a wall clock. | MET |
| **AG-14.3** "the offending call is reported typed — which tool, still running — detached and named" | `typedDetachedCallFailure` → `tool_end_execution_failure` carrying `*DetachedCallError{Tool, CallID}`, `errors.As`-extractable; asserted on tool name AND call id. No new `EventKind`, no new `Result` outcome. | MET |
| **AG-14.3** "no task belonging to the harness itself remains after the wind-down" | `RequireNoGoroutineLeak` over 50 repeats, each releasing its own tool **after** the run returned. | MET with a caveat — see MINOR-1 and MINOR-5 |

**Judgment on AG-14.3's three `Then`s.** All three are true, and the "disjoint sets" prose is *mostly* an honest reconciliation rather than a cover. The leak proof genuinely **accounts for** the detached goroutine rather than excluding it: `cancellation_winddown_test.go:189-192` closes `release` after `<-resultCh`, so the third-party goroutine exits and is inside the final count; a plain exclusion (never releasing) would have been the dishonest version and was not taken. Two honest caveats remain, both recorded below as MINORs, not as a gap being papered over.

### Spec Compliance Matrix

| Requirement | Scenario | Test | Result |
|---|---|---|---|
| R-CAN-001 / R-CAN-002 | S-CAN-001 Arm A | `cancellation_interrupt_test.go` > `TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized/provider cancelled mid-stream…` | PARTIAL — see MAJOR-3 |
| R-CAN-001 / R-CAN-002 | S-CAN-001 Arm B | same file > `…/in-flight tool call observes cancellation through the harness` | PARTIAL — see MAJOR-2 |
| R-CAN-001 | S-CAN-010 (bite) | recorded RED, reverted; mechanism present at `loop.go:442-449` | COMPLIANT |
| R-CAN-001 vocabulary | (R-CAN-001 clause) | `cancellation_events_test.go` > `TestCancellationVocabulary_SentinelsDistinctRefusalWrapsCarrierNames` (4 subtests) | COMPLIANT |
| R-CAN-002 | S-CAN-002 | `TestHarness_Interrupt_SameHarnessRunsNextPrompt` | COMPLIANT |
| R-CAN-002 | S-CAN-011 (bite) | recorded RED, reverted; mechanism present at `harness.go:451-453` | COMPLIANT |
| R-CAN-003 | S-CAN-003 | `TestHarness_Interrupt_DuringSuspensionAbortsTyped` | COMPLIANT |
| R-CAN-004 | S-CAN-004 | `TestHarness_Interrupt_SecondInterruptIsNoOp` (under `-race`) | COMPLIANT |
| R-CAN-005 | S-CAN-005 | `cancellation_shutdown_test.go` > `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts` | COMPLIANT |
| R-CAN-006 | S-CAN-006 Thens 1–2 | `cancellation_winddown_test.go` > `TestHarness_WindDown_DeafToolCannotHoldRunHostage` | COMPLIANT |
| R-CAN-006 | S-CAN-006 Then 3 | same file > `TestHarness_WindDown_NoHarnessGoroutineRemains` (serial-only) | COMPLIANT (caveat: MINOR-5) |
| R-CAN-006 | S-CAN-012 (bite) | recorded RED, reverted; timer arm present at `scheduler.go:610-620` | COMPLIANT |
| R-CAN-007 | S-CAN-007 | `cancellation_events_test.go` > `TestRunOutcomeShutdown_VocabularyAndStreamAcceptance` (4 subtests) | COMPLIANT |
| R-CAN-008 | S-CAN-008 | `doc_contract_guard_test.go` guard, green in the fresh run | COMPLIANT |
| R-RUN-001 (delta) | S-RUN-001, S-RUN-003 | `harness_test.go` four-name reflection set; `TestHarness_Interrupt_SameHarnessRunsNextPrompt` | COMPLIANT |
| R-RUN-010 (delta) | S-RUN-092 | `TestPermission_WakeParked_…_NoDeadline` byte-unchanged and green | COMPLIANT |
| R-RUN-011 (delta) | S-RUN-102 | S-CAN-001 Arm A (nil-failure interrupted close, `provider.Requests()` bound) | COMPLIANT |
| R-HIS-007 (delta) | S-HIS-097 | S-CAN-001 Arm A's synthesized-origin assertions | PARTIAL — see MAJOR-2 |
| R-HIS-007 (delta) | S-HIS-098 | `history.go` zero diff; `history_surface_guard_test.go` green | COMPLIANT |
| R-APP-009 (delta) | S-APP-017 | interrupt half + `cancellation_shutdown_test.go` > `TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal` (assertions swap, both directions) | COMPLIANT |
| R-APP-009 (delta) | S-APP-018 | `permission_protocol_test.go` zero diff; `scheduler_test.go` zero deletions | COMPLIANT |
| R-TLS-013 (delta) | S-TLS-016 | `scheduler_test.go` > `TestSchedule_ToolReceivesRunContext_EarlyReturnOnCancel` | COMPLIANT |
| R-TLS-013 (delta) | S-TLS-017 | all pre-existing scheduler tests green with zero deletions | COMPLIANT |
| R-TLS-014 (delta) | S-TLS-018 | `TestSchedule_UncancelledZeroBoundDoesNotArmTimer` | PARTIAL — see MINOR-2 |
| R-TLS-010 (restated) | S-TLS-019 | `TestSchedule_MixedCancellationBatch_DisjointResultsPerCall` | PARTIAL — see MAJOR-1 |
| R-LSK-004 (delta) | S-LSK-018 | git-diff evidence above | COMPLIANT |
| R-LSK-004 (delta) | S-LSK-019 | filter entry-set diff above | PARTIAL — see MINOR-6 |
| R-LSK-004 (delta) | S-LSK-020 | archive-time header edit; delta states this coherently | DEFERRED (correct) |
| NFR-CAN-001..005 | — | `package agent_test` throughout; `-race` green; ambient/import guards byte-unchanged and green; coverage 88.01%; single PR under `size:exception` | COMPLIANT |

**Compliance summary**: 36/41 scenarios COMPLIANT, 5 PARTIAL, 0 UNTESTED, 0 FAILING.

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | PASS | Three "TDD Cycle Evidence" tables (batches A/B/C) in `apply-progress.md` |
| All tasks have tests | PASS | 13 top-level test functions across 4 new files + 3 additions to existing files |
| RED confirmed (tests exist) | PASS | Every named test file exists and every named test function was located |
| GREEN confirmed (tests pass) | PASS | All 13 pass in the fresh uncached `-race` run |
| Triangulation adequate | PASS with note | 3 tables report `➖ Single`; each maps to a scenario with one Then-group. `S-CAN-007` and the vocabulary test carry 4 subtests each |
| Safety Net for modified files | PASS | Baselines recorded per batch; re-verified green here |

**RED evidence is genuine, not narrated.** Line-number corroboration against the current sources:
- `cancellation_interrupt_test.go:104/107/110/119` land exactly on `run_end outcome = … want RunOutcomeInterrupted`, `… want never Failed`, `run_end carries a *Failure`, `history has %d entries, want 3` — the four assertions the `S-CAN-011` bite output quotes verbatim.
- `cancellation_shutdown_test.go:192/198/201` land exactly on the corresponding shutdown assertions.
- `cancellation_winddown_test.go:100` is the `drainSink(t, sink)` call site — consistent with a `t.Helper()`-attributed `drainSink: sink did not close within 1s` failure, and with the same symptom being reported for both the 9.2 RED and the `S-CAN-012` bite.
- `cancellation_shutdown_test.go:221/224/234` land exactly on the three second-run-refusal assertions the 7.1 RED quotes.
Batch A's citations (`:82/:97/:112`) predate batch B's insertion of `readUntilDecisionRequired`, and `scheduler_test.go:1430` predates the recorded 4-tool→3-tool narrowing — both shifts are consistent with the file histories rather than with fabrication.

**Bite reverts confirmed byte-clean.** `git status --porcelain` is empty (no `.bak` residue), and all three bitten mechanisms are present in the current tree: `loop.go:442-449` (S-CAN-010), `harness.go:451-453` (S-CAN-011), `scheduler.go:610-620`'s `case <-ctx.Done():` + `time.NewTimer(bound)` (S-CAN-012).

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | 2 top-level (8 subtests) | 1 (`cancellation_events_test.go`) | `go test` |
| Integration (`agenttest`-scripted `Harness`/`Scheduler`) | 11 top-level | 4 | `go test -race` |
| E2E | 0 | 0 | not applicable to this package |
| **Total** | **13** | **5** | |

### Changed File Coverage

| File | Line % | Uncovered highlights | Rating |
|------|--------|---------------------|--------|
| `loop.go` | 88.01% | `cancellationTurnFailure` 75% (the `ferr != nil` defensive arm) | Acceptable |
| `harness.go`, `scheduler.go`, `tool.go`, `cancellation.go` | not individually gated by any spec clause | — | informational |

### Assertion Quality

No tautologies, no ghost loops, no assertion without a production-code call, no smoke-test-only tests, no mock-heavy files. Two observations rather than violations:

| File | Line | Assertion | Issue | Severity |
|------|------|-----------|-------|----------|
| `scheduler_test.go` | in `TestSchedule_UncancelledZeroBoundDoesNotArmTimer` | releases the tool immediately after `<-started` | Does not distinguish "bound unarmed" from "bound armed but never reached" | WARNING (MINOR-2) |
| `cancellation_interrupt_test.go` | 209-213 | `for i, e := range entries { … want EntryOriginAppended }` | Loop over `entries`, but `len(entries) == 0` is `t.Fatal`'d at :206 first, so the loop cannot be a ghost loop | OK |

**Assertion quality**: 0 CRITICAL, 1 WARNING.

### Quality Metrics

**Linter**: PASS — `golangci-lint run --config=.golangci.yml ./...` → `0 issues.` (cache cleaned first)
**Vet**: PASS — `go vet ./...` clean
**Vulnerabilities**: PASS — `govulncheck` exit 0, zero findings

### Issues Found

**CRITICAL**: None.

**MAJOR**

**MAJOR-1 — `S-TLS-019`'s scenario still carries the exact defect its own requirement claims to have corrected, and its passing test contradicts it.**
`openspec/changes/cachicamas-agent-cancellation-tree/specs/agent-tool-scheduler/spec.md:59` reads:
> "…the cancellation-caused slots report category `Cancellation` while any ordinary execution error still reports `Execution`…"

Two problems, both provable:
1. `ai.FailureCategoryExecution` **does not exist**. `grep -n "FailureCategory[A-Z]" backend/agent/src/ai/provider_failure.go` lists exactly nine members: `Authentication, Authorization, RateLimit, Unavailable, Timeout, Cancellation, MalformedResponse, UnsupportedCapability, Unknown`. This is the very name `R-TLS-010`'s new correction paragraph at `spec.md:47` says it removed.
2. The scenario asserts `Cancellation` for "the cancellation-caused slots", which includes the cancellation-**observing** tool's slot — directly forbidden by the same requirement three lines earlier (`spec.md:49`: "a scenario MUST NOT assert `Cancellation` on a tool's own self-reported error"). The passing test agrees with the requirement, not the scenario: `scheduler_test.go` asserts `results[0].Failure.Category() == ai.FailureCategoryUnavailable`, and `scheduler.go:1062-1066` confirms `typedFailureFromError` hardcodes `ai.FailureCategoryUnavailable`.

Provenance, by command: `git show 3ba3ab70:…/agent-tool-scheduler/spec.md | grep "S-TLS-019"` diffed against the same line at HEAD → **byte-identical**. The requirement was rewritten in `1ec63ccb`; the scenario that certifies it was not. This is the "a fix can re-encode the defect" shape — the correction was measured against the requirement's prose, not against the population of scenarios that prose governs.
**Impact**: `sdd-archive` promotes this delta verbatim into `openspec/specs/agent-tool-scheduler/spec.md`, re-introducing an unresolvable enum name into the canonical spec base and leaving a scenario that its own green test falsifies.

**MAJOR-2 — `S-CAN-001` Arm B and `S-HIS-097` both require synthesized-origin entries that the implementation provably cannot produce, and the test asserts the opposite.**
`specs/agent-cancellation-tree/spec.md:31` (Arm B) requires:
> "…orphan synthesis has closed every open call in the transcript with entries whose origin discriminator reads `synthesized` (`R-HIS-007`)…"

The covering test asserts the negation. `cancellation_interrupt_test.go:209-213`:
```go
for i, e := range entries {
    if e.Origin() != agent.EntryOriginAppended {
        t.Errorf("entries[%d].Origin() = %v, want EntryOriginAppended — this turn's own call reached a real result, it was never orphaned", i, e.Origin())
    }
}
```
The reason is structural and correct: `finishContinuationTurn` (`loop.go:500-514`) commits the assistant message and every rejoin result synchronously in the same invocation, so a call that reached `Schedule` — even with a cancellation-caused failure result — is never left open for synthesis. The reconciliation commit `e6673951` split the scenario into two arms but carried this clause into Arm B unchanged; only Arm A (whose Given is a **prior turn's** seeded open call) produces synthesized entries, and Arm A's own text no longer mentions synthesis at all.
`specs/agent-history/spec.md:33` inherits the same drift: `S-HIS-097`'s Given is "a harness-driven run interrupted **while a tool call is in flight**" — which is Arm B's shape — yet its Then demands `synthesized` origins, which only Arm A's shape yields. Its own cross-reference points at `S-CAN-001` generically.
**Impact**: two canonical-spec scenarios that no test satisfies and one test explicitly refutes, promoted at archive.

**MAJOR-3 — `S-CAN-001` Arm A's turn-bracket clause has no covering assertion anywhere in the package.**
`specs/agent-cancellation-tree/spec.md:30` requires "the turn closes aborted with a cancellation-category failure", and `R-CAN-002`'s new paragraph (`spec.md:41`) makes it normative: `TurnOutcomeAborted` plus a `*Failure` of category `ai.FailureCategoryCancellation`. The production code is `loop.go:442-449`.
Evidence of the gap: `grep -rn "TurnEnd()" *_test.go` returns 19 hits, none in any `cancellation_*_test.go`; `grep "TurnOutcomeAborted\|FailureCategoryCancellation" cancellation_*_test.go` returns only three `Category()` assertions, all on `tool_end_execution_failure` payloads, never on a `turn_end`. No test in the package reads a `turn_end`'s failure category.
What is and is not pinned: `CheckStream`'s `BracketRoleClosesRun` rule means the `turn_end` must *exist*, and `NewTurnEnd`'s failure-iff-aborted validation means a non-`Aborted` outcome carrying a failure would fail construction and cascade into a `CheckStream` violation. But a regression that emitted `TurnOutcomeAborted` with `ai.FailureCategoryUnavailable` — or dropped the failure and used a non-aborted outcome — would leave every current assertion green. The clause added by the reconciliation commit is asserted by nothing.

**MINOR**

**MINOR-1 — `R-CAN-006`'s "disjoint sets" table is not literally disjoint; the new wrapper goroutine falls in the wrong row.**
`specs/agent-cancellation-tree/spec.md:95` places in the *harness-owned, MUST exit within the bound* set: "…**every per-call goroutine the scheduler starts**…". The goroutine `scheduler.go:594-603` starts inside `runToolWithWindDown` is exactly a per-call goroutine the scheduler starts, and on the detached path it does **not** exit within the bound — it is parked inside `tool.Run`. The intended classification is clearly the third-party row (its only remaining frame is third-party code, and the buffered `resCh` lets it complete its send and exit without touching anything the harness closed — precisely the third-party row's own obligation). The behavior is right and Go offers no alternative; the table's enumeration is what is wrong. This is the one place where the reconciliation prose is doing more work than its own words support.

**MINOR-2 — `S-TLS-018`'s test is vacuous with respect to the two clauses that distinguish it.**
The scenario (`agent-tool-scheduler/spec.md:38`) requires "the release happens after an **arbitrary delay**" and "**no timer was created** — proving the bound is unarmed rather than merely generous". The test (`scheduler_test.go`, `TestSchedule_UncancelledZeroBoundDoesNotArmTimer`) does `<-started` then `close(release)` immediately — microseconds, against a 100 ms default. An unconditionally armed timer would never have fired, so the test would pass identically. The property does hold, but by a **structural** argument (`time.NewTimer` lives only inside `case <-ctx.Done():`, and `context.Background().Done()` is nil, so that arm is unselectable) that the test does not observe. The pinned `TestPermission_WakeParked_…_NoDeadline` does not close the gap either: its `EchoScriptedTool` also returns immediately. This is a scenario falsifiable only by an observation nothing external can make.

**MINOR-3 — `defer close(sink)` runs before the cancel registration is cleared, so a stream-only consumer can silently deregister the next run's signal.**
`harness.go:362-370`: the signal-clearing defer is registered first and `defer close(sink)` second, so LIFO order closes `sink` **before** `h.cancelRun = nil` executes. A consumer that treats sink-close as "the run finished" and immediately starts run #2 — exactly the Layer 3 stream-only consumer `R-CAN-005` explicitly contemplates ("a consumer that reads only the stream, such as a Layer 3 TUI holding no Go error") — can have run #2 register its cancel func and then have run #1's defer null it out. `Interrupt()` on run #2 would then be a silent no-op and the run un-cancellable. Both mutations are under `signalMu`, so this is a logical race, not a data race, and `-race` will never see it. Concurrent runs are declared out of scope (`spec.md:147`), and a caller that waits on `Run`'s **return** is safe, so this is not a defect of a stated requirement — but it is a one-line hazard (swap the two `defer` registrations) sitting on the boundary of a consumer shape the spec itself names.

**MINOR-4 — `windDownRun` discards both transcript errors, so a wind-down that corrupts history is unobservable.**
`harness.go:274-275`: `_, _ = hist.SynthesizeOrphans()` and `_ = hist.CloseTurn()`. The charter's goal line (`0003:1377`) names "history integrity after either" as a deliverable; `R-CAN-002` mandates the two calls but says nothing about their failure. If `SynthesizeOrphans` fails, the run still emits `run_end(Interrupted, nil)` and returns the sentinel, reporting a clean interrupt over a partially-synthesized transcript. This mirrors `failRun`'s pre-existing best-effort posture, so it is consistent rather than novel — recorded because it is the one exit path where shared state can end wrong and nothing says so.

**MINOR-5 — the leak proof's measurement point is after third-party release, not after wind-down.**
`TestHarness_WindDown_NoHarnessGoroutineRemains` closes `release` inside the scenario, after `<-resultCh`, so `RequireNoGoroutineLeak`'s "after" snapshot is taken once every third-party goroutine has already been let go. The charter's third `Then` is "no task belonging to the harness itself remains **after the wind-down**". The test therefore cannot, on its own, falsify "a harness-owned goroutine survived the wind-down but exits when the tool returns" — it can only falsify a permanent leak. Today nothing but the wrapper goroutine blocks on `release`, so the two coincide; a future change that parks another harness goroutine on tool completion would slip through. This is the honest reading of "accounted for rather than excluded": the accounting is real, the measurement is coarser than the sentence.

**MINOR-6 — `S-LSK-019`'s final clause has no covering test.**
"…and when an unnamed new file is placed in the package, then both substrate guards fail on it" (`agent-loop-skeleton/spec.md:49`). No test places an unnamed file. The clause describes the pre-existing guard mechanism and is true by inspection of the filter code, but nothing defends it.

**MINOR-7 — `apply-progress.md`'s final test count does not reproduce.**
`apply-progress.md:346` reports "1262 `--- PASS` / 0 `--- FAIL`" for the final `make test`. The fresh uncached run yields **3266** `--- PASS` / 0 `--- FAIL`. The FAIL count is what matters and it reproduces exactly; the PASS figure appears to have been counted under a different filter (likely top-level only, or a single package). Recorded because a count assertion in an evidence artifact that does not reproduce is the drift class this repo has already been bitten by.

**SUGGESTION**

- **S-1 — the bound is per call, so a serialized batch of deaf calls costs N × bound.** `Schedule` routes non-`EffectClassRead` calls through `scheduleSerialized` behind a capacity-1 channel (`scheduler.go:190,216`). Two cancellation-deaf mutating calls therefore pay the bound twice, sequentially. `R-CAN-006` says "wind-down MUST complete within a documented bound" without quantifying over call count. The per-call design is correct (a bound around `wg.Wait()` would race the slot write, exactly as `runToolWithWindDown`'s doc comment argues), but the requirement's phrasing should say *per call* so a future reader does not read it as a run-level guarantee.
- **S-2 — `S-CAN-001`'s two arms should be renumbered as two scenario IDs.** Keeping them as sub-bullets of one ID means the compliance matrix cannot record one arm COMPLIANT and the other PARTIAL without prose. `S-CAN-009` is free.
- **S-3 — `S-LSK-020`'s deferral is stated coherently.** The delta says the header edit "happens at ARCHIVE" and the promoted spec is correctly not edited yet. Verified: `openspec/specs/agent-loop-skeleton/spec.md` is untouched in this branch. No action needed; recorded so archive does not skip it.

### Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| `context.WithCancelCause` derived once at `Run` entry, signals invoke the cancel func | Yes | `harness.go`; no new parameter on `Turn`/`Schedule`/`executeCall`/`tool.Run` |
| Cause read, never bare `ctx.Err()`, at all three named sites | Yes | `loop.go:442`, `scheduler.go:760`, `scheduler.go:789`, plus `harness.go`'s iteration boundary and `terr` re-check |
| Bound armed only by cancellation, per call, via a zero-default `Scheduler` field | Yes | `tool.go` `WindDownBound`; timer strictly inside `case <-ctx.Done():` |
| `runToolWithWindDown` extracted as a named sub-method rather than inlined | Deviation, benign | `design.md` sketches it inlined; extraction matches the file's `scheduleRead`/`scheduleSerialized`/`runPermissionGate` sibling pattern. Recorded as design decision 9 |
| `turn_end(Aborted)` on the cause-check path | Addition, spec-backed | Not in `design.md`; required by `CheckStream`, added to `R-CAN-002` by the reconciliation commit. See MAJOR-3 for its missing assertion |
| No `fmt` in `cancellation.go` | Yes | Hand-rolled `Error()`/`Unwrap()` + `strconv.Quote`; import-boundary guard green with zero closure change |
| Panic containment across detachment | Yes | Inner goroutine recovers and carries `panicVal` on the buffered channel; `executeCall` re-panics into the unchanged `recoverCall` |

### Verdict

**FAIL — on artifact conformance, not on implementation.**

Stated plainly so the headline cannot be misread: **no code is wrong, no gate is red, and nothing here sends the change back to `sdd-apply` for production work.** Every command this phase re-ran is green — 3266 tests passing under an uncached `-race` run with zero failures and zero races, `make build` exit 0, `golangci-lint` 0 issues on a cleaned cache, `govulncheck` 0 findings, `loop.go` at 88.01% against an 80% floor. Every byte-identity pin holds exactly, including the seven-file `S-LSK-018` set, the twelve forbidden substrate files, `permission_protocol_test.go`, and the 35-entry byte-identical filter pair. All three bites are real, reverted, line-number-corroborated REDs. All six charter Gherkin scenarios are met by the production code.

The verdict is FAIL because **36 of 41 scenarios are compliant, and three of the five that are not are spec sentences that this change's own passing tests falsify** — and `sdd-archive`'s job is to promote those sentences verbatim into `openspec/specs/`. Archiving now would write into the canonical spec base:

1. a scenario naming `ai.FailureCategoryExecution`, a member that does not exist in `ai.FailureCategory`, in the very requirement whose correction paragraph says that name was removed (MAJOR-1);
2. two scenarios demanding `synthesized`-origin transcript entries under a Given whose covering test asserts, with an explicit comment, that every entry is `appended` (MAJOR-2);
3. a normative turn-bracket clause — `TurnOutcomeAborted` carrying `ai.FailureCategoryCancellation` — that no assertion in the package reads (MAJOR-3).

This is the failure mode this repository has been bitten by before: a requirement and the scenario certifying it edited out of step, so the pair certifies itself while the code says otherwise. It is cheap to close. MAJOR-1 and MAJOR-2 are **docs-only** edits to three sentences in two delta files; MAJOR-3 closes with roughly three lines of assertion added to the existing Arm A subtest, or by narrowing the clause to what `CheckStream` already pins. None of it touches production code, and none of it invalidates any evidence recorded above.

**Recommended next step**: a bounded correction pass over `specs/agent-tool-scheduler/spec.md:59`, `specs/agent-cancellation-tree/spec.md:31`, `specs/agent-history/spec.md:33`, and either `specs/agent-cancellation-tree/spec.md:30` or `cancellation_interrupt_test.go`'s Arm A subtest — then re-run this verification and proceed to `sdd-archive`. The seven MINOR and three SUGGESTION items do not block; MINOR-1, MINOR-2 and MINOR-3 are worth folding into the same pass while the context is loaded.
