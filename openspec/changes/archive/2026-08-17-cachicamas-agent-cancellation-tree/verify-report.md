```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:b3184a09fbb296a13edfece12632048243dd3d7c44639c094692e847f4cdba56
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 17/17
scenarios: 41/41
test_command: cd backend/agent && go clean -testcache && go test -race -v -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:08689287513ee3f8e9fc743b9eb9c80c8c0ac6fcd7decff2d5951de5a496865b
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report — RE-VERIFY PASS (supersedes the `fail` verdict below)

**Change**: `cachicamas-agent-cancellation-tree` (AG-14 — Build the cancellation tree, Layer 2 Wave 3, `0003:1371-1442`)
**Branch / worktree**: `feat/agent-layer2-wave3-ag14` @ `e3dd0cd7`, merge base `52701436`
**Version**: 9 commits ahead of `origin/main`; working tree clean
**Mode**: Strict TDD
**Supersedes**: the `fail` verdict recorded by the first verification pass @ `1ec63ccb` (3 MAJOR / 0 CRITICAL). That pass's findings are preserved verbatim in *§ Original findings — disposition* below and each is marked CLOSED with the command that proves it.

`evidence_revision` is the sha256 of `git diff origin/main -- . ':(exclude)…/verify-report.md'` — the candidate diff with this self-referential file excluded, so the value is reproducible from the tree.

### Verdict

**PASS WITH WARNINGS. The change is ready for `sdd-archive`.**

Zero CRITICAL. Zero MAJOR. Zero blockers. All three MAJORs from the first pass are closed, and each closure was proven *by mutation* in this pass rather than accepted from the remediation's own recorded output. Every gate is green on a cold cache: 3267 tests pass under an uncached `-race` run with zero failures and zero data races across all 12 module packages, `make build` exit 0, `make lint` reports `0 issues.`, `govulncheck` reports zero findings, and `loop.go` covers 88.01% against an 80% floor. Every byte-identity pin holds exactly.

What remains is 8 MINOR and 4 SUGGESTION items. Two of them are one-sentence documentation edits worth folding into the archive commit because `sdd-archive` promotes the sentences verbatim into `openspec/specs/`; the rest are correctly recorded as known limitations. **None of them blocks archive, and none sends the change back to `sdd-apply`.**

### Closure evidence — each MAJOR re-proven independently

#### MAJOR-1 — CLOSED. `S-TLS-019` now matches its own requirement and its own passing test.

`specs/agent-tool-scheduler/spec.md:59` was rewritten in `5b118d2c`. The nonexistent `ai.FailureCategoryExecution` is gone from the scenario, and the category split is now stated the way the code behaves: the **deaf** tool's scheduler-synthesized bound-overrun failure reports `ai.FailureCategoryCancellation`; the **observing** tool's self-reported error reports `ai.FailureCategoryUnavailable`.

| Spec clause | Test assertion | Result |
|---|---|---|
| observing slot → `Unavailable` | `scheduler_test.go:1449-1450` `results[0].Failure.Category() != ai.FailureCategoryUnavailable` → error | MATCH |
| deaf slot → `Cancellation` | `scheduler_test.go:1465-1466` `results[1].Failure.Category() != ai.FailureCategoryCancellation` → error | MATCH |
| sibling unaffected | `scheduler_test.go:1478-1485` | MATCH |
| disjoint channels | `scheduler_test.go:1436-1440` | MATCH |

`grep -rn "FailureCategoryExecution" openspec/changes/… backend/agent/src/` now returns the name only in three legitimate historical contexts — `spec.md:47` (the correction paragraph that names the removed defect), `spec.md:53` (the `(Previously: …)` provenance note), and `scheduler_test.go:1380` (a comment recording the same finding). Zero normative uses remain. `grep -n "FailureCategory[A-Z][a-zA-Z]* " backend/agent/src/ai/provider_failure.go` re-confirms the nine-member vocabulary contains no `Execution`.

#### MAJOR-2 — CLOSED. The synthesized-origin clause now sits in the arm that produces it, and it is non-vacuous.

`specs/agent-cancellation-tree/spec.md:30` (Arm A) now carries the orphan-synthesis clause **and** states the seeded open call that makes it non-vacuous. `spec.md:31` (Arm B) now asserts "no open call left behind" instead of demanding synthesized origins. `specs/agent-history/spec.md:33` (`S-HIS-097`) was re-worded from "in flight" to "left **open** by an earlier turn", with the structural reason recorded at `spec.md:35`.

Both now match the tests:

| Clause | Test | Result |
|---|---|---|
| Arm A: seeded open call repaired, entry origin `synthesized` | `cancellation_interrupt_test.go:157-159` asserts `entries[2].Origin() == agent.EntryOriginSynthesized`; `:160-166` asserts the synthesized `ToolResult.CallID() == "call-can-001-seed"` | MATCH |
| Arm A: every pre-signal entry still `appended` | `cancellation_interrupt_test.go:151-152,154-155` | MATCH |
| Arm A: turn closed | `cancellation_interrupt_test.go:168-170` `seeded.CloseTurn()` → nil | MATCH |
| Arm B: entries all `appended` | `cancellation_interrupt_test.go:239-243` | MATCH |
| `S-HIS-097` "open by an earlier turn", interrupt **or** shutdown | Arm A (interrupt) plus `cancellation_shutdown_test.go:146,235-237` — the shutdown test uses the same `NewSeededHistory` shape | MATCH |

**Non-vacuity proven by mutation, not by reading.** Removing transcript repair entirely — `windDownRun`'s `SynthesizeOrphans()` + `CloseTurn()` deleted via `go test -overlay` — makes Arm A fail (`cancellation_interrupt_test.go:149: history has 2 entries, want 3`) while Arm B correctly passes. The synthesis assertion therefore reads a real repair, not an empty set.

#### MAJOR-3 — CLOSED. The turn-close outcome and failure category are now pinned, and the pins are live.

Assertions were added to `S-CAN-001` Arm A (`cancellation_interrupt_test.go:126-145`) and to the shutdown wind-down test (`cancellation_shutdown_test.go:214-233`): locate the `turn_end`, assert `TurnOutcomeAborted`, assert the `*Failure` is present, assert `Category() == ai.FailureCategoryCancellation`.

**Revert verified byte-identical, not narrated.** `git rev-parse 1ec63ccb:…/loop.go HEAD:…/loop.go` → both `11a7d54051c340eae2810c15e14b3319ecdd74a1`. `git diff --stat 1ec63ccb..HEAD -- …/loop.go` is empty. The RED mutation left no residue.

**Both halves of the pin proven live by independent mutation** (`go test -overlay`, no git write, production tree untouched):

| Mutation applied to `loop.go` | Observed | Verdict |
|---|---|---|
| `cancellationTurnFailure`: `Category: ai.FailureCategoryCancellation` → `ai.FailureCategoryUnavailable` | `cancellation_interrupt_test.go:144: turn_end failure.Category() = unavailable, want ai.FailureCategoryCancellation` **and** `cancellation_shutdown_test.go:232:` the same | Category pin is live in **both** tests |
| `NewTurnEnd(…, TurnOutcomeAborted, failure)` → `NewTurnEnd(…, TurnOutcomeUnknown, nil)` | `:137 turn_end outcome = unknown, want agent.TurnOutcomeAborted` **and** `:141 turn_end carries no *Failure, want one`; the same pair at `cancellation_shutdown_test.go:225,229` | Outcome and failure-presence pins are live in **both** tests |

The remediation's own recorded RED (only the category half) understated the coverage actually achieved; the outcome half is pinned too.

#### MINOR-3 — CLOSED. The defer-ordering defect is really fixed, and the regression test really detects it.

**(a) Is the fix correct and complete for every `Run` exit path?** Yes.

`harness.go:372` now registers `defer close(sink)` *before* the `cancelRun`-clearing defer at `harness.go:374-380`. LIFO therefore executes `h.queue.close()` → `cancelRun = nil; cancel(nil)` → `close(sink)`. Because deferred calls run sequentially on the returning goroutine, the nil-assignment strictly happens-before the channel close, and a consumer that starts run #2 on observing the close can no longer have its registration overwritten. Run #2's own registration takes `signalMu`, so the ordering is total.

Exit-path audit (`grep -n "return " harness.go` over `Run`, read in full):
- The **post-shutdown early return** (`harness.go:346-349`) closes `sink` itself and returns *before either defer is registered* — no double close, and no `cancelRun` is ever registered on that path. This is the one path the brief asked about explicitly; it is safe.
- Every other return (`NewRunStart` error, `failRun` call sites, `windDownRun`, the terminal-decision success path) is inside the deferred region and gets the new order.
- `defer h.queue.close()` (`harness.go:391`) is registered last and so still runs first, unchanged before and after the fix; it has no coupling to `sink` or `cancelRun`.
- The new order also means `cancel(nil)` now runs *before* `close(sink)`. The per-turn forwarder goroutine is already joined at `<-forwarderDone` before any return, so no goroutine observes `runCtx` after the swap that did not before. No new hazard.

**(b) Is `TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration` a genuine regression detector or decorative?** Genuine, and stronger than its author claimed.

Inverting the two `defer` registrations back to the pre-fix order via `go test -overlay` (production tree untouched) and running the test three times:

```
attempt 17: run #2 did not end within 500ms of Interrupt() — the interrupt was silently swallowed …
attempt 76: …
attempt 21: …
```

3 RED out of 3 runs, first reproduction at attempt 17/76/21 of a 2000-attempt budget, ~0.6 s per detection. The author's calibration ("~2–6 hits per 300 attempts") is consistent with this and, if anything, pessimistic. The coverage profile independently corroborates that all 2000 attempts really execute: `loop.go:442.2` carries 4006 hits in a run where every other cancellation test contributes single digits.

**(c) Does it risk flaking green-side or destabilising the suite?** Low risk today; one cheap change would remove the residual risk.

- **Green-side stability**: `go test -race -count=10` → `ok … 120.122s`, 0 failures — 20,000 attempts with zero false positives. Plus a separate single run and the full-suite run: 0/22,000 total.
- **Cost**: the `src/agent` package goes from **2.853 s** to **7.109 s** under `-race` (`go test -race -count=1 -skip '…SinkClose…'` vs without the skip). The whole module suite is 2 m 53 s, dominated by `openaicompat` at 172 s. A 4.3 s addition is real but not a destabiliser.
- **`GOMAXPROCS(2)` is safe as written.** The test is deliberately not `t.Parallel()`. Go resumes paused top-level parallel tests only after every serial top-level test has finished, so no unrelated test executes while the process-wide value is lowered, and the original is restored by `defer`.
- **Residual flake vector — see SUGGESTION S-4.** The only failure trigger is a 500 ms `time.After` deadline (`cancellation_interrupt_test.go:717`). Under the defect, run #2's `Interrupt()` is a *permanent* no-op and run #2 never ends, so the deadline is a hang detector, not a latency measurement. On a heavily loaded shared runner a legitimate run #2 could conceivably exceed 500 ms and produce a false RED. Raising the deadline to a few seconds costs nothing on the green side and removes the vector entirely.

**Net judgment: the test is net-positive.** It is the only assertion in the package that defends an ordering defect `-race` is structurally blind to, it discriminates 3/3 against the defect and 0/22,000 against the fix, and its cost is 4.3 s in a 173 s suite.

### Build & Tests Execution — all re-run in this pass

**Tests**: PASS — `go clean -testcache && go test -race -v -count=1 ./...`, exit 0.

```text
--- PASS lines: 3267
--- FAIL lines: 0
DATA RACE occurrences: 0
all 12 module packages report ok
wall clock: 2m53s
```

The count moves from the first pass's 3266 to 3267 — exactly the one `--- PASS` line the new MINOR-3 regression test adds. The delta reproduces the arithmetic rather than merely resembling it.

**Build**: PASS — `cd backend/agent && make build` (`go build -trimpath ./...`), exit 0, 25-byte output.

**Lint**: PASS — `./bin/golangci-lint cache clean && make lint` → `go vet ./...` clean, `bin/golangci-lint run --config=.golangci.yml ./...` → `0 issues.`, exit 0.

> **Recorded so the next pass does not lose an hour to it.** An initial lint attempt in this pass reported `src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go:17:9: var-naming: don't use an underscore in package name (revive)` — reproducibly, three times, on a file byte-identical to `origin/main`. It was **not** a real finding. The repo pins `bin/golangci-lint` at **2.9.0**; `$HOME/go/bin/golangci-lint` is **2.12.2**. Cleaning the shared cache with the 2.12.2 binary and then running the 2.9.0 binary produces this phantom. Proof: `git archive origin/main` extracted to a scratch tree and linted with the *same* 2.9.0 binary and config yields `0 issues.`; after `./bin/golangci-lint cache clean` (2.9.0), both the scratch base tree and this worktree yield `0 issues.` cold. **Always clean the cache with `./bin/golangci-lint`, never with whatever is on `PATH`.**

**Vulnerabilities**: PASS — `make vuln-check` (`govulncheck -json ./...`) exit 0, zero `"finding"` entries.

**Coverage**: `loop.go` = **88.01%** (257/292 statements in the profile's own block accounting), threshold 80% → ABOVE. Unchanged from the first pass. The cancellation branch is genuinely exercised: profile blocks `loop.go:442.2`, `442.100`, `443.67`, `444.92`, `448.3` and `cancellationTurnFailure`'s `566.61`/`574.2` all carry 4004–4006 hits. The single zero-hit block is `571.17,573.3` — `cancellationTurnFailure`'s defensive `ferr != nil` arm.

### Hard pins — each re-verified by command in this pass

| Pin | Command | Result |
|---|---|---|
| `S-LSK-018` file set | `git diff --name-only origin/main -- backend/agent/src/agent/ \| grep -v _test.go` | Exactly `{cancellation.go, doc.go, harness.go, loop.go, run_events.go, scheduler.go, tool.go}` — the six pinned pre-existing files plus the one new file. No seventh. PASS |
| Forbidden substrate byte-unchanged | per-file `git diff --stat origin/main` | `turn_events.go`, `failure.go`, `stream_check.go`, `stream_check_test.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `reconstruction_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `history.go`, `permission_protocol_test.go` — all zero diff. PASS |
| `permission_protocol_test.go` identity + green | zero diff (above) and `--- PASS` in the fresh run | PASS |
| `go.mod` / `go.sum` / `backend/agent/src/ai/**` | `git diff --stat origin/main -- …` | Empty. PASS |
| `.golangci.yml` / `Makefile` | `git diff --stat origin/main -- …` | Empty — the lint config is not the source of the phantom above. PASS |
| `S-LSK-019` filters byte-in-sync | extracted every `strings.HasSuffix(path, "…")` from `filterOutLoopFiles` (`loop_test.go:831`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907`) and compared | Identical in order and bytes, 35 entries each; every entry an exact filename suffix beginning `/`; no wildcard, prefix or directory pattern; `stream_check_test.go` absent from both; all five `cancellation*` files present in both. PASS |
| `loop.go` RED-revert identity | `git rev-parse 1ec63ccb:…/loop.go HEAD:…/loop.go` | Both `11a7d540…`. PASS |
| Remediation blast radius | `git diff --name-status 1ec63ccb..HEAD` | Exactly 8 paths: 2 test files, `harness.go`, 3 spec deltas, `apply-progress.md`, and this report. No production file other than `harness.go`. PASS |

### Pre-existing `gofmt` drift — confirmed NOT an AG-14 defect, and confirmed not worsened

`gofmt -l` over `src/agent/` yields **18** dirty files in this worktree and **18** in the pristine `origin/main` tree extracted via `git archive`. `comm -23` over the two sorted lists is **empty** — AG-14 introduced zero new drift. Every AG-14-authored or AG-14-edited file (`cancellation.go`, the four `cancellation_*_test.go` files, and `harness.go`) is gofmt-**clean**. Go 1.26.6. Deliberately left alone because `make fmt` would destroy the byte-unchanged pins. Known and accepted; not a finding.

### Spec Compliance Matrix

Authoritative totals counted from the six delta spec files: **17 requirements** (`grep -cE '^### (R|NFR)-'`), **41 scenario bullets** (`grep -cE '^\s*- \*\*S-[A-Z]+-[0-9]+[a-z]?\*\*'`).

| Requirement | Scenario | Test | Result |
|---|---|---|---|
| R-CAN-001 / R-CAN-002 | S-CAN-001 Arm A | `TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized/provider cancelled mid-stream…` | COMPLIANT (was PARTIAL — MAJOR-3 closed) |
| R-CAN-001 / R-CAN-002 | S-CAN-001 Arm B | same file > `…/in-flight tool call observes cancellation through the harness` | COMPLIANT (was PARTIAL — MAJOR-2 closed; caveat MINOR-8) |
| R-CAN-001 | S-CAN-010 (bite) | recorded RED, reverted; mechanism present at `loop.go:442-449` | COMPLIANT |
| R-CAN-001 vocabulary | (R-CAN-001 clause) | `TestCancellationVocabulary_SentinelsDistinctRefusalWrapsCarrierNames` (4 subtests) | COMPLIANT |
| R-CAN-002 | S-CAN-002 | `TestHarness_Interrupt_SameHarnessRunsNextPrompt` | COMPLIANT |
| R-CAN-002 | S-CAN-011 (bite) | recorded RED, reverted; mechanism present at `harness.go` | COMPLIANT |
| R-CAN-003 | S-CAN-003 | `TestHarness_Interrupt_DuringSuspensionAbortsTyped` | COMPLIANT |
| R-CAN-004 | S-CAN-004 | `TestHarness_Interrupt_SecondInterruptIsNoOp` (under `-race`) | COMPLIANT |
| R-CAN-005 | S-CAN-005 | `TestHarness_Shutdown_WindsDownAndRefusesNewPrompts` (+ the new `turn_end` assertions) | COMPLIANT |
| R-CAN-005 (consumer shape) | — | `TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration` | COMPLIANT (new in remediation) |
| R-CAN-006 | S-CAN-006 Thens 1–2 | `TestHarness_WindDown_DeafToolCannotHoldRunHostage` | COMPLIANT |
| R-CAN-006 | S-CAN-006 Then 3 | `TestHarness_WindDown_NoHarnessGoroutineRemains` (serial-only) | COMPLIANT (caveat: MINOR-5) |
| R-CAN-006 | S-CAN-012 (bite) | recorded RED, reverted; timer arm present at `scheduler.go:610-620` | COMPLIANT |
| R-CAN-007 | S-CAN-007 | `TestRunOutcomeShutdown_VocabularyAndStreamAcceptance` (4 subtests) | COMPLIANT |
| R-CAN-008 | S-CAN-008 | `doc_contract_guard_test.go` guard, green in the fresh run | COMPLIANT |
| R-RUN-001 (delta) | S-RUN-001, S-RUN-003 | `harness_test.go` four-name reflection set; `TestHarness_Interrupt_SameHarnessRunsNextPrompt` | COMPLIANT |
| R-RUN-010 (delta) | S-RUN-092 | `TestPermission_WakeParked_…_NoDeadline` byte-unchanged and green | COMPLIANT |
| R-RUN-011 (delta) | S-RUN-102 | S-CAN-001 Arm A (nil-failure interrupted close, `provider.Requests()` bound) | COMPLIANT |
| R-HIS-007 (delta) | S-HIS-097 | S-CAN-001 Arm A's synthesized-origin assertions + the shutdown test's identical seeded shape | COMPLIANT (was PARTIAL — MAJOR-2 closed) |
| R-HIS-007 (delta) | S-HIS-098 | `history.go` zero diff; `history_surface_guard_test.go` green | COMPLIANT |
| R-APP-009 (delta) | S-APP-017 | interrupt half + `TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal` | COMPLIANT |
| R-APP-009 (delta) | S-APP-018 | `permission_protocol_test.go` zero diff; `scheduler_test.go` zero deletions | COMPLIANT |
| R-TLS-013 (delta) | S-TLS-016 | `TestSchedule_ToolReceivesRunContext_EarlyReturnOnCancel` | COMPLIANT |
| R-TLS-013 (delta) | S-TLS-017 | all pre-existing scheduler tests green with zero deletions | COMPLIANT |
| R-TLS-014 (delta) | S-TLS-018 | `TestSchedule_UncancelledZeroBoundDoesNotArmTimer` | PARTIAL — MINOR-2 |
| R-TLS-010 (restated) | S-TLS-019 | `TestSchedule_MixedCancellationBatch_DisjointResultsPerCall` | COMPLIANT (was PARTIAL — MAJOR-1 closed) |
| R-LSK-004 (delta) | S-LSK-018 | git-diff evidence above | COMPLIANT |
| R-LSK-004 (delta) | S-LSK-019 | filter entry-set comparison above | PARTIAL — MINOR-6 |
| R-LSK-004 (delta) | S-LSK-020 | archive-time header edit; delta states this coherently | DEFERRED (correct) |
| NFR-CAN-001..005 | — | `package agent_test` throughout; `-race` green; ambient/import guards byte-unchanged and green; coverage 88.01%; single PR under `size:exception` | COMPLIANT |

**Compliance summary**: **41/41 scenarios have a covering test that passed at runtime** in this pass — 0 UNTESTED, 0 FAILING. That is the completion criterion the envelope's `scenarios: 41/41` records.

Two of those 41 are additionally graded **PARTIAL on clause-level observability**: `S-TLS-018` and `S-LSK-019` each carry one sub-clause that their otherwise-passing covering test cannot falsify (MINOR-2 and MINOR-6). `S-LSK-020` is correctly DEFERRED to archive. Read `41/41` as "every scenario is covered and green", not as "every clause is independently falsifiable" — the two exceptions are named above and carried as warnings, which is precisely what `pass_with_warnings` denotes here.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 57 |
| Tasks complete | 56 |
| Tasks incomplete | 1 (`tasks.md:152` — task `13.5`, explicitly reserved for `sdd-archive`) |

`grep -c "^- \[x\]" tasks.md` → 56; the single `^- \[ \]` is the archive hand-off note. Not a defect. `tasks.md` is byte-unchanged by the remediation (`git diff 1ec63ccb..HEAD -- tasks.md` is empty), which is correct: the remediation closed verification findings, it did not add scope.

### Original findings — disposition

The first pass's findings are preserved here rather than deleted, per the record-keeping requirement.

| # | Original finding (first pass @ `1ec63ccb`) | Status | Closing evidence |
|---|---|---|---|
| **MAJOR-1** | `S-TLS-019` asserted `Cancellation` on the cancellation-**observing** tool's slot — forbidden by its own `R-TLS-010` three lines above — and named the nonexistent `ai.FailureCategoryExecution` | **CLOSED** in `5b118d2c` | `specs/agent-tool-scheduler/spec.md:59` rewritten; split now matches `scheduler_test.go:1449-1450` and `:1465-1466`; zero normative uses of `FailureCategoryExecution` remain |
| **MAJOR-2** | `S-CAN-001` Arm B and `S-HIS-097` demanded `synthesized` origins on a path that cannot produce them; the covering test asserted `appended` with an explicit refuting comment | **CLOSED** in `5b118d2c` | Clause moved to Arm A (`spec.md:30`) with the seeded open call; Arm B (`:31`) now asserts "no open call left behind"; `S-HIS-097` (`agent-history/spec.md:33`) re-worded "in flight" → "left **open** by an earlier turn". Non-vacuity proven by overlay mutation (removing transcript repair fails Arm A, passes Arm B) |
| **MAJOR-3** | `S-CAN-001` Arm A's `TurnOutcomeAborted` + `ai.FailureCategoryCancellation` clause had no covering assertion anywhere in the package | **CLOSED** in `e3dd0cd7` | Assertions at `cancellation_interrupt_test.go:126-145` and `cancellation_shutdown_test.go:214-233`; both halves proven live by two independent overlay mutations; `loop.go` revert byte-identical (`git rev-parse` blob match) |
| **MINOR-1** | `R-CAN-006`'s "disjoint sets" table puts `runToolWithWindDown`'s wrapper goroutine in the harness-owned row it cannot satisfy | **OPEN** | Unchanged at `specs/agent-cancellation-tree/spec.md:96`. See below |
| **MINOR-2** | `S-TLS-018`'s "arbitrary delay" and "no timer was created" are not what the test observes | **OPEN** | Unchanged at `specs/agent-tool-scheduler/spec.md:38`. See below |
| **MINOR-3** | `defer close(sink)` ran before the cancel registration was cleared | **CLOSED** in `e3dd0cd7` | `harness.go:372` swap; fix audited across every `Run` exit path including the post-shutdown early return; regression test proven 3/3 RED against the inverted order and 0/22,000 green-side |
| **MINOR-4** | `windDownRun` discards both transcript errors | **OPEN** | Unchanged at `harness.go:274-275`. See below |
| **MINOR-5** | Leak snapshot is taken after third-party release, so only a permanent leak is falsifiable | **OPEN** | Unchanged. See below |
| **MINOR-6** | `S-LSK-019`'s final clause ("an unnamed new file … both guards fail on it") has no covering test | **OPEN** | Unchanged at `specs/agent-loop-skeleton/spec.md:49`. See below |
| **MINOR-7** | `apply-progress.md`'s final `--- PASS` count does not reproduce | **OPEN** | Unchanged, and now in three places: `apply-progress.md:242`, `:272`, `:346` all say `1262`. Fresh run: 3267. See below |
| **S-1 / S-2 / S-3** | per-call bound wording; renumber `S-CAN-001`'s arms; `S-LSK-020` deferral is coherent | **OPEN / informational** | Unchanged; carried forward below |

### Issues Found in this pass

**CRITICAL**: None.

**MAJOR**: None.

**MINOR**

**MINOR-1 — OPEN, worth fixing at archive.** `specs/agent-cancellation-tree/spec.md:96` places "every per-call goroutine the scheduler starts" in the *harness-owned, MUST exit within the bound* row. The goroutine `runToolWithWindDown` starts (`scheduler.go:594-603`) is exactly that, and on the detached path it does **not** exit within the bound — it is parked inside `tool.Run`. Its correct home is the third-party row. The behavior is right and Go offers no alternative; the table's enumeration is what is wrong. Docs-only, promoted verbatim by archive.

**MINOR-2 — OPEN, record as a known limitation.** `specs/agent-tool-scheduler/spec.md:38` requires "the release happens after an **arbitrary delay**" and "**no timer was created**". `TestSchedule_UncancelledZeroBoundDoesNotArmTimer` does `<-started` then `close(release)` immediately, so an unconditionally armed 100 ms timer would never have fired and the test would pass identically. The property does hold, by the structural argument that `time.NewTimer` lives only inside `case <-ctx.Done():` and `context.Background().Done()` is nil — but nothing external can observe "no timer was created" in Go. Correctly deferred: this is a scenario clause that is true and unobservable, and the honest disposition is to record it, not to fake an observation.

**MINOR-4 — OPEN, record as a known limitation.** `harness.go:274-275`: `_, _ = hist.SynthesizeOrphans()` and `_ = hist.CloseTurn()`. A wind-down that corrupts the transcript still emits `run_end(Interrupted, nil)` and returns the sentinel. This mirrors `failRun`'s pre-existing best-effort posture, so it is consistent rather than novel. Worth a follow-up item in a later milestone, not a blocker here.

**MINOR-5 — OPEN, record as a known limitation.** `TestHarness_WindDown_NoHarnessGoroutineRemains` closes `release` after `<-resultCh`, so `RequireNoGoroutineLeak`'s "after" snapshot is taken once every third-party goroutine has been let go. It can falsify a permanent leak but not "a harness-owned goroutine survived the wind-down and exits only when the tool returns". Today nothing but the wrapper goroutine blocks on `release`, so the two coincide.

**MINOR-6 — OPEN, record as a known limitation.** `specs/agent-loop-skeleton/spec.md:49`'s final clause — "when an unnamed new file is placed in the package, then both substrate guards fail on it" — has no covering test. It describes the pre-existing guard mechanism and is true by inspection of the 35-entry filter pair, but nothing defends it.

**MINOR-7 — OPEN, worth fixing (cheap).** `apply-progress.md:242`, `:272` and `:346` all report "1262 `--- PASS` / 0 `--- FAIL`" for the final `make test`. The fresh uncached run yields **3267**. The `--- FAIL` figure is what matters and it reproduces exactly; the PASS figure was evidently counted under a different filter. Recorded because a count assertion in an evidence artifact that does not reproduce is a drift class this repository has already been bitten by.

**MINOR-8 — NEW, worth fixing at archive.** `specs/agent-cancellation-tree/spec.md:32` closes with: "The property Arm B actually owns is **no open call survives the wind-down**, which is what `CloseTurn` would reject and what the scenario asserts." The final clause is not true as written. Arm B's subtest never calls `CloseTurn` — `grep -n "CloseTurn" cancellation_*_test.go` returns exactly three sites (`cancellation_interrupt_test.go:168` in Arm A, `:528` in the suspension test, `cancellation_shutdown_test.go:113`), none in Arm B. The property itself **is** true and **is** defended, just not by the named mechanism: an overlay mutation that suppresses the rejoin commit in `finishContinuationTurn` makes Arm B fail, and a break that left the call open would be repaired into a `synthesized` entry that Arm B's all-`appended` loop (`:239-243`) would catch. This is materially weaker than the MAJOR-2 it replaced — a true normative clause with an inaccurate meta-claim about which assertion carries it, not a clause a green test refutes. The cleanest fix is one line: add `hist.CloseTurn()` to Arm B, matching the three sibling sites, which makes the prose literally accurate.

**MINOR-9 — NEW, worth fixing (cheap, code comment only).** `cancellation_interrupt_test.go:616-619` states: "harness.go's Run registers `defer close(sink)` BEFORE the `cancelRun`-clearing defer, so LIFO execution closes the sink FIRST and clears `h.cancelRun` SECOND." The premise is the **fixed** order but the conclusion is the **defect's** behavior — LIFO means a defer registered first runs *last*, so the fixed order clears `cancelRun` first and closes the sink second. The production comment at `harness.go:365-366` states this correctly, and the two now contradict each other. The rest of that paragraph correctly narrates the defect the test reproduces; only the opening sentence was half-updated. A future reader would conclude the current code still has the bug. Not promoted into any spec, so it does not reach `openspec/specs/`.

**SUGGESTION**

- **S-1 — the bound is per call, so a serialized batch of deaf calls costs N × bound.** `Schedule` routes non-`EffectClassRead` calls through `scheduleSerialized` behind a capacity-1 channel (`scheduler.go:190,216`). `R-CAN-006` says "wind-down MUST complete within a documented bound" without quantifying over call count. The per-call design is correct; the requirement's phrasing should say *per call*.
- **S-2 — `S-CAN-001`'s two arms should be renumbered as two scenario IDs.** With the arms now carrying genuinely different Thens and different Givens, one ID for two arms forces the compliance matrix to record them in prose. `S-CAN-009` is free.
- **S-3 — `S-LSK-020`'s deferral is stated coherently.** `openspec/specs/agent-loop-skeleton/spec.md` is untouched in this branch, as the delta says it should be. Recorded so archive does not skip the header edit.
- **S-4 — NEW: raise the MINOR-3 test's deadline from 500 ms.** `cancellation_interrupt_test.go:717`'s `time.After(500 * time.Millisecond)` is a hang detector, not a latency measurement: under the defect the interrupt is permanently swallowed and run #2 never ends. Raising it to 3–5 s costs nothing on the green side (0/22,000 attempts already finish well inside 500 ms) and removes the only plausible false-RED vector on a loaded CI runner.

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | PASS | Three "TDD Cycle Evidence" tables (batches A/B/C) plus a bounded "Remediation" section in `apply-progress.md` |
| All tasks have tests | PASS | 14 top-level test functions across 4 new files + additions to 3 existing files |
| RED confirmed | PASS, and independently re-proven | This pass re-derived the RED for MAJOR-3 (two mutations) and MINOR-3 (one mutation) via `go test -overlay`, without any git write. The remediation's recorded output matched in every case |
| GREEN confirmed | PASS | All 14 pass in the fresh uncached `-race` run |
| Bite reverts byte-clean | PASS | `git status --porcelain` empty; `loop.go` blob identical to `1ec63ccb`; all three bitten mechanisms present |
| Safety Net for modified files | PASS | Baselines recorded per batch; re-verified green here |

### Assertion Quality

| File | Line | Assertion | Issue | Severity |
|------|------|-----------|-------|----------|
| `scheduler_test.go` | in `TestSchedule_UncancelledZeroBoundDoesNotArmTimer` | releases the tool immediately after `<-started` | Does not distinguish "bound unarmed" from "bound armed but never reached" | WARNING (MINOR-2) |
| `cancellation_interrupt_test.go` | 239-243 | `for i, e := range entries { … want EntryOriginAppended }` | Loop over `entries`, but `len(entries) == 0` is `t.Fatal`'d first, so it cannot be a ghost loop | OK |
| `cancellation_interrupt_test.go` | 126-145 | `turn_end` outcome/failure/category | Proven live by two independent mutations | OK |
| `cancellation_interrupt_test.go` | 650-726 | `…SinkCloseObservationDoesNotLoseRun2CancelRegistration` | Proven 3/3 RED against the inverted defer order; 0/22,000 false positives | OK |

No tautologies, no ghost loops, no assertion without a production-code call, no smoke-test-only tests, no mock-heavy files. **0 CRITICAL, 1 WARNING.**

### Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| `context.WithCancelCause` derived once at `Run` entry, signals invoke the cancel func | Yes | `harness.go`; no new parameter on `Turn`/`Schedule`/`executeCall`/`tool.Run` |
| Cause read, never bare `ctx.Err()`, at all three named sites | Yes | `loop.go:442`, `scheduler.go:760`, `scheduler.go:789`, plus `harness.go`'s iteration boundary and `terr` re-check |
| Bound armed only by cancellation, per call, via a zero-default `Scheduler` field | Yes | `tool.go` `WindDownBound`; timer strictly inside `case <-ctx.Done():` |
| `runToolWithWindDown` extracted as a named sub-method rather than inlined | Deviation, benign | Matches the file's `scheduleRead`/`scheduleSerialized`/`runPermissionGate` sibling pattern. Recorded as design decision 9 |
| `turn_end(Aborted)` on the cause-check path | Addition, spec-backed **and now test-backed** | Required by `CheckStream`; added to `R-CAN-002` by the reconciliation commit; pinned by assertion since `e3dd0cd7` |
| `defer` ordering in `Run` | Corrected | `close(sink)` registered first so LIFO clears `cancelRun` first. Post-shutdown early return closes `sink` itself before either defer registers — no double close |
| No `fmt` in `cancellation.go` | Yes | Hand-rolled `Error()`/`Unwrap()` + `strconv.Quote`; import-boundary guard green |
| Panic containment across detachment | Yes | Inner goroutine recovers and carries `panicVal` on the buffered channel; `executeCall` re-panics into the unchanged `recoverCall` |

### Recommendation

**Proceed to `sdd-archive`.**

Two one-sentence documentation edits are worth folding into the archive commit, because `sdd-archive` promotes these exact sentences into `openspec/specs/`:

1. `specs/agent-cancellation-tree/spec.md:96` — move "every per-call goroutine the scheduler starts" from the harness-owned row to the third-party row (MINOR-1).
2. `specs/agent-cancellation-tree/spec.md:32` — either add `hist.CloseTurn()` to Arm B's subtest or drop the "and what the scenario asserts" clause (MINOR-8).

Two more are cheap and local but do not reach the canonical spec base: MINOR-9 (fix the inverted LIFO sentence in the test's doc comment) and MINOR-7 (correct the three stale `1262` counts in `apply-progress.md`).

MINOR-2, MINOR-4, MINOR-5 and MINOR-6 are **correctly deferred and should simply be recorded as known limitations** — each is a true property whose observation is either impossible from outside the package (MINOR-2, MINOR-6) or coarser than the sentence describing it (MINOR-5), or a best-effort posture consistent with pre-existing precedent (MINOR-4). S-4 is a cheap hardening of the new regression test's deadline.

None of these blocks archive.
