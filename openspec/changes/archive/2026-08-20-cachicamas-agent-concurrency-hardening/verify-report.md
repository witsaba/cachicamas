```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:4eec0a5c7f2f6f533d4722dd15b1fedfe04d9cca48a9aba7625aadd4c77783a6
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 17/17
scenarios: 20/20
test_command: cd backend/agent && go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:ed89766c67bcad0910b87bc0effc98e812eb764f6b75f8c23177f058b81716dc
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

# ROUND 2 — AUTHORITATIVE

> Round 2 supersedes round 1. **Round 1 is preserved verbatim below** under
> "ROUND 1 — SUPERSEDED, PRESERVED FOR THE RECORD"; its envelope has been
> reproduced as a table so this file carries exactly one machine-readable
> envelope. Nothing in round 1 was deleted.

## Verification Report

**Change**: `cachicamas-agent-concurrency-hardening` (AG-21 — Harden concurrency, backpressure and leaks)
**Milestone**: 22 of 24 · Layer 2 · Wave 6
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ag21-concurrency-hardening` · branch `feat/agent-layer2-wave6-ag21` · HEAD `d11f3d8b`
**Mode**: Strict TDD
**Verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 0 MAJOR, 3 MINOR, 2 SUGGESTION.

**All three round-1 MAJORs are genuinely closed**, and each was re-proven by my own
command, not accepted on the correction round's report. MAJOR-1 was closed by
**building** the missing conjunction, not by narrowing the text that demanded it —
I confirmed the spec's GIVEN/THEN clauses are byte-unchanged and defeat-tested the
new driver three ways. The change is **ready for `sdd-archive`**.

**Size is not a finding.** Final diff excluding `openspec/**` is **1,903 counted
lines** (1,868 + / 35 −), measured from the true merge base
(`git merge-base 54476ded d11f3d8b` = `54476ded`), under the user's pre-authorized
`size:exception`.

---

### Completeness

| Metric | Value |
|---|---|
| Tasks total | 53 |
| Tasks complete | 44 |
| Tasks incomplete | 9 — all Phase 9–10, `sdd-archive`'s own rows, correctly left `[ ]` |

Task ticks match code state. Round 2 added rows R2.1–R2.7 plus updates to 2.4 / 2.5 /
6.3 / 6.4 and the new body-correction obligation on 9.7 — all verified present.

### Build & Tests Execution

**Tests**: ✅ exit 0 — every package `ok`, zero FAIL, zero panic, zero DATA RACE.

```text
cd backend/agent && go test -race -count=1 ./...
ok  src/agent 11.505s            ok  src/agenttest 2.805s
ok  src/agenttest/sweep 1.722s   ok  src/agenttest/tracetest 2.236s
ok  src/ai 4.540s                ok  src/ai/internal/retry 2.703s
ok  src/ai/openaicompat 174.009s ok  src/ai/openaicompat/conformancetest 3.584s
ok  .../openrouter 3.315s        ok  .../openrouter/conformance 6.760s
ok  .../openrouter/internal/smoke 2.882s   ok  src/handoff 2.280s
real 174.51
```

**Not a cache artifact**: `-count=1`, real wall clock **174.51s**, `openaicompat`
alone 174.0s. No `(cached)` line anywhere.

**⚠️ Disclosed in full: my FIRST full-suite run of this session exited 1.** See
**MINOR-3** — it is a pre-existing flake in a package outside AG-21's dependency
closure, proven below, not a regression. I am reporting both runs rather than only
the green one.

**Build**: ✅ `make build` → `go build -trimpath ./...`, exit 0.
**Lint**: ✅ `make lint` → `go vet ./...` + `golangci-lint run --config=.golangci.yml ./...` → `0 issues.`
**Vet**: ✅ `go vet ./...` exit 0.
**Vuln**: ✅ `make vuln-check` exit 0, **0** `"finding"` objects.
**gofmt**: ✅ `gofmt -l` over all 8 AG-21 files → **empty** (round-1 MINOR-3 closed).
`make all` was **not** run — its `fmt` step rewrites committed files and trips the substrate guards.

---

## R1–R5 adjudications

### R1 — MAJOR-1 **CLOSED BY BUILDING**. The conjunction is real by construction. The text was **not** weakened. ✅

**The unbuffered half is genuine.** `slow_consumer_pressure_test.go:378` is
`make(chan *agent.Event)` — no capacity. `cnhStalledConsumer` (`:34-59`) reads one
event at a time, and once `stopAfter` fires it `break`s, closes `reachedStall`, then
blocks on `<-resume` (`:51-52`). At the instant `<-stalled` returns, the consumer
goroutine is **not** in a receive on the sink.

**Proven by defeat, not by reading.** I removed `close(resume)` (`:414`) and ran under
`-timeout 25s`:

```text
panic: test timed out after 25s
goroutine 8 [chan send]:
goroutine 17 [chan send]:
FAIL  github.com/cachicamas/backend/agent/src/agent  25.518s
```

Producer goroutines parked in **`chan send`** — the stall genuinely blocks the
producer at its unconditional send. Structural, never temporal.

**The steering half is genuinely pending at the same instant, by construction.**
`heldTurnScript` (`harness_steering_test.go:25`) has `agenttest.Hold(gate)` as its
**first** step — nothing is emitted ahead of it. `Gate.markReached` is signalled only
by the fake provider's producer arriving at that Hold (`fake_gate.go:46-51`).
`Harness.Steer` (`harness.go:275-277`) delegates to `steeringQueue.enqueue`, which
appends under a mutex and returns — it never blocks and never touches the loop, so a
nil return means **queued and undelivered**. In the driver, `gate.Release()` happens at
`:413` and `close(resume)` at `:414`, both **after** `Steer` at `:404`. So at the
moment of `Steer`: consumer definitively not reading **and** turn one still held with
the message queued. Both halves hold simultaneously by construction, never by inferred
timing.

**The steering assertion bites.** I sabotaged `steeringQueue.enqueue` (`harness.go:206`)
to drop the message:

```text
--- FAIL: TestCombinedPressure_StalledSteering_NeverCancelled_LosesNothing
    slow_consumer_pressure_test.go:486: combined/stalled-steering/never-cancelled:
      history has 2 entries, want 4 (prompt, turn-one assistant, steered, turn-two assistant)
```

Reverted; `harness.go` sha256 re-matched its HEAD blob and `git status --porcelain`
was empty. *(Method note: the reported line 486 is the `cnhDriveCombinedStalledSteering(t, false)`
call site, not the assertion at `:473` — `t.Helper()` attributes failures to the
first non-helper frame. Not an anomaly.)*

**The text was NOT weakened — verified by diff.** `git diff 7e1561c1 d11f3d8b --
specs/agent-event-delivery/spec.md` is **one line changed**. Every GIVEN and every
THEN of `S-AGE-031` — including the load-bearing *"AND that same run is simultaneously
in one of the four combined states of `R-CNH-001`"* — is **byte-unchanged**. The single
edit **strengthened** the trailing cross-reference from *"Cross-referenced to
`R-CNH-003`/`S-CNH-007`/`S-CNH-008` and `R-CNH-001`"* to a direct discharge naming the
two new tests. **The requirement's strength did not change; only its evidence did.**
Steering is one of the scenario's own four admissible states, so "one of the four" is
satisfied literally.

Both new tests PASS under `-race -count=1`.

### R2 — MAJOR-2's six-site correction is complete and internally consistent. ✅ (one residual comment → MINOR-1)

Requirement and scenarios now say the **same** thing:

| Site | Current text | ✓ |
|---|---|---|
| `specs/agent-concurrency-hardening/spec.md:35` | heading: "the **two cells** the charter's uniform Then cannot describe" | ✅ |
| `…:42` (assertion 4) | "**except the compaction mid-run and child-harness-active cells**" | ✅ |
| `…:47-49` (table) | compaction×failure = adjusted; **child×failure = adjusted + containment**; suspension×failure = **uniform Then, plus a park-boundary check** | ✅ |
| `…:51-55` (deviation) | the two bullets are compaction and **child-harness-active** | ✅ |
| `…:56` | explicit: "A suspension pending × provider failure is **NOT** one of the two exceptions" | ✅ |
| `…` `S-CNH-005` heading | "**uniform in outcome and adjusted only in timing**"; body: "the uniform Then, **not** one of the two recorded exceptions above" | ✅ |
| `design.md:36` (AD-2) | "**Suspension×failure is uniform, not adjusted**" | ✅ |
| `design.md:46` (AD-3) | "except compaction×failure **and child-harness-active×failure**" | ✅ |
| `design.md:164` | amendment record carries a `(Previously: …)` note | ✅ |
| `docs/…/0003-…md:3` | "two cells (a failed compaction, **and a child harness's contained provider failure**) … (sdd-verify round-2, MAJOR-2: a suspended call's provider failure is uniform in outcome, adjusted only in timing, and is not one of these two)" | ✅ |
| `combined_matrix_test.go:17-21` | Go comment names the correct pair and disclaims suspension | ✅ |

**I re-ran the zero-survivors sweep myself** with a regex the correction round's own
sweep could not have used (`suspension` within 140 chars of `adjusted Then` / `second
exception`, in either order, across the whole change scope plus `src/agent`). Two hits,
**both corrective text explicitly denying the old pairing** — `spec.md:56` and
`tasks.md:56`. **Zero genuine survivors in the normative text.**

The mechanism itself I re-derived from production source: `loop.go` has exactly **two**
`Schedule` call sites — `:411` (`_ = sched.Schedule(...)`, discarded, the
nil-continuation path) and `:542` (`results = ...Schedule(...)`, captured). `Schedule`
returns `[]Result` and no error, so a contained child failure can never become a
Turn-level error on the parent. The corrected spec matches the shipped mechanism.

One residual Go doc comment still carries the old claim → **MINOR-1** below.

### R3 — MAJOR-3 closed; the corrected clause is **TRUE**, measured directly. ✅

`S-LSK-034` now reads *"neither names `cost_events_test.go`, `stream_check_test.go` or
`reconstruction_test.go` (`failure.go` and `cost_events.go` ARE named in both,
legitimately, released by AG-11 and AG-16 respectively…)"*. I measured both filters
directly rather than trusting either report:

| Entry | `filterOutLoopFiles` | `filterOutLoopHookFiles` | Clause |
|---|---|---|---|
| `/failure.go` | **1** | **1** | present — correctly no longer claimed absent ✅ |
| `/cost_events.go` | **1** | **1** | present — correctly no longer claimed absent ✅ |
| `/cost_events_test.go` | 0 | 0 | genuinely absent ✅ |
| `/stream_check_test.go` | 0 | 0 | genuinely absent ✅ |
| `/reconstruction_test.go` | 0 | 0 | genuinely absent ✅ |
| AG-21's 5 new files | 1 each | 1 each | present in **both** filters ✅ |

- The inherited-falsity correction **did** land as a `(Previously: …)` bullet inside
  `S-LSK-032`'s own AG-21-amendment section in **AG-21's delta**
  (`specs/agent-loop-skeleton/spec.md`, new bullet) ✅
- A **promotion obligation** was added, and it is explicitly flagged as a **body** edit
  distinct from the two header edits, both at the delta's header-maintenance block and
  in **task 9.7** ✅
- The already-promoted `openspec/specs/agent-loop-skeleton/spec.md` was **correctly left
  untouched**. `git diff 54476ded d11f3d8b -- openspec/specs/` lists **only**
  `agent-hook-taxonomy/spec.md`, which is AG-20's truncation repair from `c46b696b`, not
  this correction ✅

### R4 — the correction's two honesty flags, adjudicated

**(a) SUGGESTION-1 is genuinely proven — I isolated what apply could not.** Apply
honestly reported that its RED caught a broader failure (CheckStream + event-kind
mismatch) rather than isolating the new line. I closed that gap with a scalpel mutation
of **only** the strengthened comparison, one cell at a time:

```text
# mutate combined_matrix_test.go:100  RunOutcomeCompleted -> RunOutcomeInterrupted
--- FAIL: TestCombinedMatrix/compaction_mid_run/provider_failure
    compaction_mid_run/provider_failure: run_end outcome = completed, want RunOutcomeCompleted

# mutate combined_matrix_test.go:163  RunOutcomeCompleted -> RunOutcomeInterrupted
--- FAIL: TestCombinedMatrix/child_harness_active/provider_failure
    child_harness_active/provider_failure: run_end outcome = completed, want RunOutcomeCompleted
```

Each fires alone, with **no** CheckStream or event-kind noise, and each reports the real
observed outcome `completed`. That proves the line executes, reads the true outcome, and
that the true outcome is `RunOutcomeCompleted`. **Verdict: genuinely proven, not
unproven.** Both mutations reverted; tree byte-identical.

**(b) The leak-sweep omission is a real but benign text gap → MINOR-2.** I did not take
the scope call on trust — I tested it. Adding both arms of the new driver to
`combined_leak_sweep_test.go`'s scenario and running the sweep:

```text
ok  github.com/cachicamas/backend/agent/src/agent  4.099s
```

**The sweep passes with the new driver included, in 4.1s.** So the omission hides **no
leak** — and the driver already honours `NFR-HKS-005`/`NFR-CNH-003` in full: its gate is
released **inline** at `:413` before the driver returns **and** carries a cleanup release
registered at construction (`:369`). What remains is a text gap, graded MINOR-2 below,
with a proven two-line remedy. Reverted; tree byte-identical.

**(c) My own round-1 citation was wrong, and I am correcting it here.** Round 1 cited
`loop.go:265` as the `Schedule` discard. That is **false**: `loop.go:265` is the
`opts TurnOptions` **parameter declaration**. `grep -n "Schedule("` over `loop.go`
returns exactly two sites — **`:411`** (`_ = sched.Schedule(...)`, the real discard) and
`:542` (captured). **The correction round is right and round 1 was imprecise.** The
substance of round-1 MINOR-1 stands; only its line number was wrong.

### R5 — no regressions, no weakening, no scope escape. ✅

I inspected **every** non-comment removed line in the round-2 Go diff. There are exactly
six, and **not one is a weakening**:

- 4 lines: the two weak `== Interrupted || == Shutdown` negative checks, **replaced by
  strictly stronger positive `!= RunOutcomeCompleted` checks** (SUGGESTION-1).
- 2 lines: `gofmt` struct-field and trailing-comment realignment.

| Scope check | Result |
|---|---|
| Production Go changed | **zero** — every changed `.go` is `_test.go` ✅ |
| `src/ai/` diff | **empty** ✅ |
| `src/agenttest/` diff | **empty** — no widening ✅ |
| `go.mod` / `go.sum` diff | **empty** — no new dependency ✅ |
| New `EventKind` / `TurnOutcome` / `RunOutcome` declared | **none** ✅ |
| Wall clock in the 4 correction-touched files | **none** — `time.Sleep\|time.Since\|time.Now\|time.Tick\|runtime.NumGoroutine` returns nothing ✅ |
| Remaining `time.` use | one defensive `time.After(5s)` + `t.Fatal` deadline (`:396`) — never a synchronization point, never an assertion ✅ |
| Substrate-filter entries for new symbols | **not required** — the correction added **no new file**; all 5 AG-21 files already present in **both** filters ✅ |
| `gofmt -l` over all 8 AG-21 files | **empty** ✅ |
| Worktree / base checkout | worktree clean; base untouched at `54476ded`, `git status` empty ✅ |

---

### Spec Compliance Matrix

| Requirement | Scenario | Covering evidence | Result |
|---|---|---|---|
| `R-CNH-001` | `S-CNH-001` (12 cells) | `TestCombinedMatrix` 12/12 PASS; requirement now matches shipped behaviour | ✅ COMPLIANT *(was PARTIAL)* |
| `R-CNH-001` | `S-CNH-002` | `containmentChildFailure`, `containmentLeafFirst` | ✅ COMPLIANT |
| `R-CNH-001` | `S-CNH-003` (bite) | round-1 re-plant → RED, reverted | ✅ COMPLIANT |
| `R-CNH-001` | `S-CNH-004` | `adjustedThenCompactionFailure`; positive-outcome line mutation-proven (R4a) | ✅ COMPLIANT |
| `R-CNH-001` | `S-CNH-005` | `adjustedThenSuspensionFailure` → `assertUniformOutcome` + `requestsAtPark == 1`; spec now agrees | ✅ COMPLIANT |
| `R-CNH-002` | `S-CNH-006` | `agenttest/` diff empty; all files `package agent_test`; one `Run` at a time | ✅ COMPLIANT |
| `R-CNH-003` | `S-CNH-007` | `TestSlowConsumerPressure_NeverCancelled_LosesNothing` | ✅ COMPLIANT |
| `R-CNH-003` | `S-CNH-008` (bite) | apply-recorded RED | ✅ COMPLIANT |
| `R-CNH-004` | `S-CNH-009` | `TestSlowConsumerPressure_CancelledUnblocksWithinBound` | ✅ COMPLIANT |
| `R-CNH-004` | `S-CNH-010` (bite) | apply-recorded hang under `-timeout` | ✅ COMPLIANT |
| `R-CNH-005` | `S-CNH-011` | `TestConcurrencyHardening_PackageLeakSweep`, no `t.Parallel()` | ⚠️ COMPLIANT with **MINOR-2** (one driver omitted) |
| `R-CNH-005` | `S-CNH-012` (bite) | round-1 re-plant → growth 2→52, tolerance 25 | ✅ COMPLIANT |
| `R-CNH-006` | `S-CNH-013` | pressure-2 driver in sweep; release after run returns | ✅ COMPLIANT |
| `R-CNH-007` | `S-CNH-014` | `TestCrossRunState_NilHistory_…`; both defeats re-proven round 1; wording fixed (SUGGESTION-2) | ✅ COMPLIANT |
| `R-CNH-007` | `S-CNH-015` | `TestCrossRunState_SharedHistory_…` | ✅ COMPLIANT |
| `R-CNH-007` | `S-CNH-016` (bite) | round-1 re-plant → RED, count 2 | ✅ COMPLIANT |
| `R-CNH-008` | `S-CNH-017` | scope fence re-verified by command (R5 table) | ✅ COMPLIANT |
| `R-AGE-008` | `S-AGE-031` | **`TestCombinedPressure_StalledSteering_{NeverCancelled_LosesNothing,Interrupted}`** — conjunction built and defeat-proven (R1) | ✅ COMPLIANT *(was PARTIAL)* |
| `R-LSK-008` | `S-LSK-032` amendment | `TestHooks_…NoReleaseExactWideningInBothFilters` PASS with `t.Logf` fall-through | ✅ COMPLIANT |
| `R-LSK-009` | `S-LSK-034` | clause now TRUE (measured, R3); bite (f) two independent REDs | ✅ COMPLIANT *(was PARTIAL)* — see SUGGESTION-1 |
| `R-AGS-016` | `S-AGS-067` | four postures verified by construction | ✅ COMPLIANT |
| `R-AGS-009` | `S-AGS-035` parenthetical | doc `0003:2204` left naming `AG-01.1, AG-20.2` | ✅ COMPLIANT |
| `R-CAN-002`/`005`/`008`, `R-RUN-001`, turn-termination non-req | (no new scenario minted) | discharged via `S-CNH-011`–`016` | ✅ COMPLIANT |

**Compliance summary**: **20/20 scenarios COMPLIANT**, 0 PARTIAL, 0 FAILING, 0 UNTESTED.
**Requirements**: 17/17 — 8 `R-CNH` + `R-AGE-008` + `R-RUN-001` + `R-CAN-002`/`005`/`008` + `R-LSK-008`/`009` + `R-AGS-016`/`009`.

**Reverse traceability — all 7 delta specs walked again, scenario by scenario. No
scenario is unclaimed.** The seven scenarios with no literal ID string in code are
accounted for and none is orphaned: `S-CNH-008`/`010`/`012`/`016` are the transient
`(bite)` convention (RED-then-revert, recorded in `tasks.md`/`apply-progress`, four of
which I or round 1 re-planted and watched FAIL); `S-CNH-006` and `S-CNH-017` are
construction/scope claims discharged by the git-level commands in the R5 table; and
`S-AGS-067`'s four postures are discharged by construction.

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD evidence reported | ✅ | `apply-progress` + per-row evidence in `tasks.md`, incl. R2.1–R2.7 |
| All behavioural rows have tests | ✅ | 5 files; round 2 added 1 driver + 2 tests, no new file |
| RED confirmed | ✅ | all test files present and compiling |
| GREEN confirmed | ✅ | full uncached `-race -count=1` suite exit 0 |
| Bites RED-first | ✅ | round 1 re-planted 4 of 7; **round 2 added 4 further independent defeats** (stall, steering, and the two isolated outcome mutations) |
| Triangulation | ✅ | 12-cell table + 2 pressure arms + **2 new combined arms** + 2 cross-run branches |
| Safety net for modified files | ✅ | round-2 edits confined to released regions; guards re-run green |

**TDD compliance**: 7/7.

### Assertion Quality

No tautology, no orphan-empty check, no type-only assertion, no ghost loop, no
smoke-test-only, no mock-heavy file. The change's two absence-shaped assertions were
**defeat-tested**: the cross-run needle twice in round 1, and in round 2 I defeat-tested
the new combined driver's stall and steering halves independently. The two strengthened
outcome assertions were **mutation-proven in isolation**.

**Assertion quality**: ✅ all assertions verify real behaviour.

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| AD-1 shared driver core | ✅ | one `runCombinedCell` |
| AD-2 per-cell drive plan | ✅ | **corrected** — deviation now names compaction + child-harness-active |
| AD-3 assertion core | ✅ | **corrected and strengthened** — assertion 4 carve-out matches shipped code; both adjusted cells assert `RunOutcomeCompleted` positively |
| AD-4 structural stall | ✅ | unbuffered sink, no clock; **defeat-proven** |
| AD-5 leak sweep | ⚠️ | passes; one new driver not enrolled — **MINOR-2** |
| AD-6 minted needle + defeat | ✅ | defeat-proven both halves |
| AD-7 seven bites | ✅ | 4 re-planted round 1, 4 new defeats round 2 |
| AD-9 branch-scoping, both layers | ✅ | `t.Logf` fall-through, PASS not SKIP |
| AD-10 exact-suffix widening, both filters | ✅ | measured: 5/5 new files in both; the 3 named-absent files 0/0 |
| D1 no production code | ✅ | zero production Go changed |
| D5 no `agenttest` widening | ✅ | diff empty |
| `NFR-CNH-005` review budget | ✅ | 1,903 counted lines under **pre-authorized** `size:exception` — not a finding |

---

### Issues Found

**CRITICAL**: None.

**MAJOR**: None. All three round-1 MAJORs independently re-verified as closed.

**MINOR**

- **MINOR-1 — a residual wrong-pair claim survives in a Go doc comment, at the exact
  site a maintainer will consult.**
  `backend/agent/src/agent/combined_matrix_test.go:108-109` still reads: *"//
  adjustedThenSuspensionFailure — S-CNH-005. The suspension x provider-failure cell, **at
  its R-CNH-001-recorded adjusted Then**."* The corrected `R-CNH-001` records **no**
  adjusted Then for suspension — `specs/agent-concurrency-hardening/spec.md:56` denies it
  in terms. The comment even contradicts itself three lines later (`:112`: *"then the
  **uniform** provider-failure Then holds"*). A weaker sibling sits at
  `combined_state_fixtures_test.go:574` (*"unless this cell carries R-CNH-001's recorded
  adjusted Then"*, while the suspension cell does carry a non-nil `adjustedThen` hook).
  *Why the correction round missed it*: its sweep regex required `compaction` and
  `suspension` within 80 characters of each other; this line names neither `compaction`
  nor `exception`, so the sweep **structurally could not** catch it — the "zero
  survivors" claim is therefore true of the normative text but not of the Go comments.
  *Failure scenario*: a maintainer touching the suspension cell reads its doc comment,
  is told `R-CNH-001` records an adjusted Then for suspension, cross-checks
  `R-CNH-001`, finds the opposite, and "reconciles" the spec back to the wrong pairing —
  reintroducing MAJOR-2 verbatim. Two-line comment fix; **not blocking** (Go comments are
  not promoted into `openspec/specs/` by `sdd-archive`, and the code is correct).

- **MINOR-2 — the leak sweep omits one of "the drivers this change adds", and the
  requirement's own enumeration is now an undercount.**
  `R-CNH-005` (`specs/agent-concurrency-hardening/spec.md`) requires the sweep run *"the
  drivers this change adds — the twelve cells, **the two pressure scenarios** and the
  cross-run driver"*. The change now adds a **third** pressure-file driver,
  `cnhDriveCombinedStalledSteering` (`slow_consumer_pressure_test.go:365`), which is
  absent from `combined_leak_sweep_test.go:41-49`. The general clause ("the drivers this
  change adds") is therefore falsified by the shipped sweep, and the appositive count
  ("the two pressure scenarios") went stale the moment a third was appended — this
  repository's recorded count-drift failure mode, and no test fails for it.
  *Adjudication*: **acceptable scope call on substance, real gap on text.** I proved it
  hides no leak — enrolling both arms and re-running the sweep gives `ok … 4.099s` — and
  the driver already satisfies `NFR-HKS-005` (inline gate release at `:413` **plus**
  cleanup release at `:369`). *Failure scenario*: a future edit that makes the combined
  driver leak a goroutine is not caught by the one sweep whose stated job is to catch
  exactly that, while `R-CNH-005` is archived claiming it covers every driver the change
  adds. Remedy is two lines plus one appositive edit, both proven to pass. **Not
  blocking.**

- **MINOR-3 — a pre-existing, unrelated flake makes the declared full-suite command
  non-deterministic.**
  My **first** `go test -race -count=1 ./...` of this session exited **1**:
  `src/ai/openaicompat/a_i-33_1_test.go:181: Category() = unavailable, want
  FailureCategoryCancellation` in `TestAI33_1_RaceCancelMidDo`. **Blame settled by
  reachability, not by re-running**: `go list -deps -test ./src/ai/openaicompat` does
  **not** contain `backend/agent/src/agent`, and AG-21's diff under `src/ai/` is
  **empty** — the failure is outside this change's dependency closure and cannot be
  caused by it. The test is untouched by AG-21 (last modified in `07e4d0c2`) and is
  inherently racy by construction: it launches `go cancel()` against a **real HTTP dial**
  and asserts the failure category is `cancellation`, which loses to `unavailable`
  whenever the dial fails first. Re-run **20×** in isolation: green. Second full suite:
  exit 0. *Failure scenario*: any gate that treats the full suite as a binary signal —
  including `sdd-archive`'s own re-run — can fail on a coin flip unrelated to the change
  under review. **Follow-up for the `openaicompat` owner, not a blocker for AG-21.**

**SUGGESTION**

- **SUGGESTION-1** — `S-LSK-034`'s absence clause is now **true** but still has **no
  permanent covering assertion**. I verified it by direct measurement, and the nearest
  shipped assertion (`hooks_test.go:2079-2084`) checks `countString(...) > 1` — a
  *duplicate* guard, which defends no absence claim. A three-name absence assertion
  beside the existing filter checks would make the clause self-defending and would have
  caught the original MAJOR-3 at authoring time.
- **SUGGESTION-2** — the new combined driver's committed-fact check is thinner than its
  standalone siblings'. `cnhDriveCombinedStalledSteering` counts only
  `cost_session(Final) == 1` (`:439-447`), where `cnhAssertCommittedFactsPresent`
  (`:91-123`) additionally pins a tool-end event by **call identity**. That is a
  reasonable consequence of choosing a steering script with no tool call, and
  `S-AGE-031`'s identity-set clause is carried by the standalone halves — but a combined
  arm driving a tool call would exercise the identity set under the conjunction too.

---

### Verdict

**PASS WITH WARNINGS** — 0 CRITICAL, 0 MAJOR, 3 MINOR, 2 SUGGESTION.

Every round-1 MAJOR is closed, and closed the honest way: MAJOR-1 by **building** the
conjunction rather than narrowing the claim that demanded it, with the spec's GIVEN/THEN
clauses proven byte-unchanged; MAJOR-2 by correcting all six sites into agreement with
the shipped mechanism, which I re-derived from `loop.go` rather than inherited; MAJOR-3
by replacing a false clause with one I then measured to be true. Nothing was weakened —
the only assertions removed in the correction round were replaced by strictly stronger
ones. The three MINORs are a self-contradicting Go comment, a two-line sweep enrolment
proven to hide no leak, and a pre-existing flake in a package this change cannot reach.
None of them can be falsified into a blocker.

**The change is READY FOR `sdd-archive`.**

`sdd-archive` should carry forward: the `S-LSK-032` **body** correction obligation
(task 9.7, not just the two header edits), and — if it takes cheap follow-ups — MINOR-1
and MINOR-2, both of which are text-only and proven safe.

*(Headline/risks consistency check: this report's verdict, its counts, its matrix and
its risks all agree — 0 CRITICAL, 0 MAJOR, nothing blocking archive. No truncation.)*

---
---

# ROUND 1 — SUPERSEDED, PRESERVED FOR THE RECORD

> Preserved verbatim below except that its YAML envelope is reproduced as a table, so
> this file carries exactly one machine-readable envelope. Round 1's verdict of **FAIL**
> was correct **at commit `7e1561c1`** and was answered by the correction round
> (`17d7edf4`..`c68e307c`). Its **only** factual error is the `loop.go:265` citation in
> MINOR-1, corrected in round 2 §R4(c) to `loop.go:411`.

**Round-1 envelope (reproduced as a table):**

| Field | Value |
|---|---|
| schema | `gentle-ai.verify-result/v1` |
| evidence_revision | `sha256:67752b65ccf6cc1f79902cd1006c25e5938086c10930e0fe752adb6f3375c096` |
| verdict | `fail` |
| blockers | 3 |
| critical_findings | 0 |
| requirements | 14/17 |
| scenarios | 17/20 |
| test_command | `cd backend/agent && go test -race -count=1 ./...` |
| test_exit_code | 0 |
| test_output_hash | `sha256:493212514a6244e6af2438da39b64df2757ff5d14907ed256167ee65f7b70b66` |
| build_command | `cd backend/agent && make build` |
| build_exit_code | 0 |
| build_output_hash | `sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495` |


## Verification Report

**Change**: `cachicamas-agent-concurrency-hardening` (AG-21 — Harden concurrency, backpressure and leaks)
**Milestone**: 22 of 24 · Layer 2 · Wave 6
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ag21-concurrency-hardening` · branch `feat/agent-layer2-wave6-ag21` · HEAD `7e1561c1`
**Mode**: Strict TDD
**Verdict**: **FAIL** — 0 CRITICAL, 3 MAJOR, 4 MINOR, 2 SUGGESTION.

The implementation is sound and the suite is genuinely green. Every MAJOR is a **normative-text defect**: spec prose that the shipped code falsifies. Because `sdd-archive` promotes these files verbatim into `openspec/specs/`, shipping them as-is would archive three false claims that no green suite will ever surface — this repository's own recorded failure mode. The verdict is FAIL to force a scoped text correction, not because anything is broken.

---

### Completeness

| Metric | Value |
|---|---|
| Tasks total | 53 |
| Tasks complete | 44 |
| Tasks incomplete | 9 — all Phase 9–10, `sdd-archive`'s own rows (`tasks.md:111-122`), correctly left `[ ]` |

Task ticks match code state. Verified independently: no Phase 0–8 row is unticked, and no ticked row lacks a shipped artifact.

### Build & Tests Execution

**Tests**: ✅ all packages `ok`, exit 0, **zero** FAIL / panic / DATA RACE.

```text
cd backend/agent && go test -race -count=1 ./...
ok  src/agent 11.670s   ok  src/agenttest 2.430s   ok  src/agenttest/sweep 1.840s
ok  src/agenttest/tracetest 2.031s   ok  src/ai 4.689s   ok  src/ai/internal/retry 2.223s
ok  src/ai/openaicompat 174.594s     ok  src/ai/openaicompat/conformancetest 2.700s
ok  src/ai/openaicompat/openrouter 3.079s   .../openrouter/conformance 6.499s
.../openrouter/internal/smoke 2.130s      ok  src/handoff 1.815s
real 2:55.39
```

**Not a cache artifact**: `-count=1`, real wall clock **2:55.39**, `openaicompat` alone 174.6s — matching the documented ~170s uncached baseline. No `(cached)` line anywhere.

**Build**: ✅ `make build` → `go build -trimpath ./...`, exit 0.
**Lint**: ✅ `make lint` → `go vet ./...` + `golangci-lint run --config=.golangci.yml ./...` → `0 issues.`
**Vuln**: ✅ `make vuln-check` exit 0, **0** `"finding"` objects.
**Vet**: ✅ exit 0.
`make all` was **not** run (its `fmt` step rewrites committed files and trips the substrate guards).

---

### Adjudications requested by the orchestrator

#### V1 — `S-AGE-031` is **NOT** discharged. Apply's escalation was correct. → **MAJOR-1**

The promoted delta text (`specs/agent-event-delivery/spec.md:81-82`) makes the claim a **conjunction**:

> **GIVEN** a run whose consumer sink is **unbuffered** … **AND** that same run is simultaneously in one of the four combined states of `R-CNH-001` — a suspension pending, a steer queued and undelivered, a compaction call in flight, or a child harness active …

Proved by command that the conjunction is exercised **nowhere** in the change:

| Driver | Sink | Combined state |
|---|---|---|
| `arrangeSuspensionPending` … `arrangeChildHarnessActive` (all 12 cells) | `make(chan *agent.Event, 512)` — **buffered** (`combined_state_fixtures_test.go:242,307,407,494`) | yes |
| `cnhDrivePressureNeverCancelled` | `make(chan *agent.Event)` — **unbuffered** (`slow_consumer_pressure_test.go:159`) | **none** |
| `cnhDrivePressureCancelledUnblocks` | `make(chan *agent.Event)` — **unbuffered** (`slow_consumer_pressure_test.go:247`) | **none** |

No shipped driver has both halves. The stalled-consumer half is genuinely proven and the combined-state half is genuinely proven, but **never together**, which is precisely what `S-AGE-031` asserts. Status: **PARTIAL**, not COMPLIANT.

`tasks.md:92` (row 6.4) claims the scenario is "discharged by the cross-referenced `S-CNH-007`/`S-CNH-008` evidence". That discharge claim is not supported by the shipped tests.

**What closes it — either is acceptable, the first is cheaper:**
- **(a) Correct the delta text.** Rewrite `S-AGE-031`'s second GIVEN so the combined-state clause is an *adjacent, separately-proven* claim rather than a conjunction on the same run, and correct `tasks.md:92`'s discharge sentence to match. Zero new test code.
- **(b) Build the combined variant.** One new driver: an unbuffered stalled sink over (say) the steering-queued arrangement. Larger, and it re-opens the diff.

Do **not** archive `S-AGE-031` as written.

#### V2 — `R-CNH-001`'s prose understates the exceptions, and it names the **wrong pair**. → **MAJOR-2**

Mechanism verified first-hand in production source, not taken from the report:

- `finishContinuationTurn` (`loop.go:525-566`) calls `opts.Continuation.Scheduler.Schedule(...)` at `:542`. `Schedule` returns `[]Result` and **no error** — a contained tool-execution failure becomes a `Result` with `ToolOutcomeExecutionFailure`, converted by `toolResultMessage` (`:578-599`) into an `ai.NewToolFailure` message and appended to history. It can never become a Turn-level error, so it can never become `RunOutcomeFailed`.
- Confirmed empirically: `adjustedThenChildFailure` (`combined_matrix_test.go:133-150`) asserts the parent reaches its own terminal with `got.err == nil`, never `RunOutcomeFailed`.

Now the contradiction. The shipped table carries **three** `adjustedThen` cells (`combined_matrix_test.go:25,36,43`), and inspection of what each actually asserts sharpens the finding further:

| Cell | Shipped behaviour | `R-CNH-001` says |
|---|---|---|
| suspension × failure | `adjustedThenSuspensionFailure` **calls `assertUniformOutcome`** (`combined_matrix_test.go:115`) — uniform assertion 4 **is** carried | "**adjusted Then**" |
| compaction × failure | assertion 4 **replaced** (`:75-99`) | "**adjusted Then**" ✔ |
| child × failure | assertion 4 **replaced** (`:133-150`) | "**uniform Then**, plus containment" ✘ |

So the spec does not merely undercount — it names **suspension** (which does carry the uniform Then) and omits **child** (which does not). Every site below is falsified by the shipped code:

| Site | Text |
|---|---|
| `specs/agent-concurrency-hardening/spec.md:42` | assertion 4: "a provider failure yields the **failed run outcome** with a typed `*Failure`" — uniform, no child carve-out |
| `specs/agent-concurrency-hardening/spec.md:49` | table row: `a child harness active │ … │ uniform Then, plus containment` |
| `specs/agent-concurrency-hardening/spec.md:51` | "**Recorded charter deviation — two cells**" |
| `specs/agent-concurrency-hardening/spec.md:56` | "Neither adjustment weakens the cell: **both cells** still carry all four uniform assertions" |
| `design.md:46` | AD-3 assertion 4: "…**except compaction×failure** whose Then is AD-2's" — names only **one** |
| `docs/architecture/milestones/0003-…md:3` | "with **two cells** (a failed compaction, and a provider failure while a call is parked) carrying a recorded, evidence-cited adjusted Then" |

`S-CNH-002`'s own scenario text ("the parent reaches its own terminal") is non-committal and stays true, so apply was right to trust it over the table — but the requirement and its table must now be corrected to agree, or AG-21 archives a normative table its own passing test contradicts. **A scoped text correction closes this; no code change is needed.** All six sites must change together — correcting the requirement without the table, the design and the doc paragraph is the exact half-fix this repository has already paid for.

#### V3 — Apply's mechanism claim is **correct**; its stated *reason* is wrong. → **MINOR-1**

Verified in `loop.go:525-566`: `finishContinuationTurn` appends the assistant message (`:547-552`) and then **every** call's result (`:553-562`) — success, result-failure and execution-failure alike — before returning, and `Harness.Run`'s wind-down check is only reached later. So a detached call is indeed never left open for `SynthesizeOrphans` (`history.go:371-401`, which repairs only `h.open`). The redesign to `NewSeededHistory` is correct and `S-CNH-015` is genuinely discharged: `cross_run_state_test.go:255-270` asserts an `EntryOriginSynthesized` tool-result entry matching the seeded call, and `:271-273` asserts `CloseTurn()` returns nil.

**But** the shipped comment at `combined_matrix_test.go:123-125` says "*Schedule's own return value is discarded by the loop (the documented R-LSK-… gap)*". That is false on this path: `results` **is** captured at `loop.go:542`. The `_ = sched.Schedule(...)` discard is the **nil-continuation** path (`loop.go:265`), which `Harness.Run` never takes. The correct reason is simply that `Schedule` has no error return. Left as-is, this comment teaches the wrong mechanism at the exact site a future reader will consult — and it carries a dangling `R-LSK-...` placeholder.

#### V4 — The absence assertion is **real, not vacuous**. Proven by defeat, twice. → **PASS**

Both halves were re-planted and watched to FAIL:

```text
# Defeat 1 — the absence clause, against a deliberately caller-shared *agent.History:
cross_run_state_test.go:78: run 2's captured request(s) reference run 1's minted
  call identity 2 time(s), want 0 — nil History: nothing carries over
--- FAIL: TestCrossRunState_NilHistory_AbsenceWithAntiGhostFloor (0.02s)

# Defeat 2 — the anti-ghost floor, needle replaced by a never-minted identity:
cross_run_state_test.go:78: anti-ghost floor: run 1's own tool_start for the minted
  call identity was not observed — the needle is not findable at all
--- FAIL: TestCrossRunState_NilHistory_AbsenceWithAntiGhostFloor (0.02s)
```

- **The needle is a minted identity, not a shape.** `cnhMintNonce` (`cross_run_state_test.go:29-33`) draws from a process-wide `atomic.Uint64`, giving `call-cnh-nonce-<N>` — unique per execution, unproducible by any other fixture. `cnhCountCallIDInRequests` (`:38-53`) matches on `tc.ID()` / `tr.CallID()` **equality**, never a substring.
- **The anti-ghost floor is present and load-bearing** (`:128-136`, `t.Fatal`), and defeat 2 proves it bites.
- **Non-vacuity of the absence measure is proven positively**: defeat 1 returns a count of **2** on the identical code path, so `run2Provider.Requests()` is demonstrably non-empty and the green `!= 0` check is a real measurement, not an empty-collection artifact.

This is the strongest item in the change. `-overlay` was sound here — this test shells out to nothing.

---

### Bites re-proven by command (4 of 7 re-planted, ≥3 required)

| Bite | Re-planted sabotage | Observed RED |
|---|---|---|
| **(a)** `S-CNH-003` | deleted `h.Interrupt()` from the steering×interrupt fire closure | `Run error = <nil>, want errors.Is(_, agent.ErrInterrupted)` + `run_end outcome = completed, want RunOutcomeInterrupted` — clean mismatch in **0.489s**, not a hang, under `-timeout 120s` ✅ |
| **(c)** `S-CNH-012` | `go func(){ <-neverClosed }()` once per sweep iteration | `RequireNoGoroutineLeak: goroutine count grew from 2 to 52 across 50 repeats (tolerance 25)` — growth exactly 50 ✅ |
| **(d)** `S-CNH-016` | nil-`History` → caller-shared `*agent.History` | absence clause RED, count 2 (see V4) ✅ |
| **(f)** `S-LSK-034` | deleted `/combined_matrix_test.go` from `filterOutLoopFiles` **only** | **TWO independent REDs**: `loop_test.go:1359: substrate was edited (R-LSK-004 violated)` **and** `scope_fence_test.go:153: filterOutLoopFiles has 78 entries, filterOutLoopHookFiles has 79` ✅ |

**Method note, and a live overlay trap.** Bite (f) run under `-overlay` produced only **one** RED — `S-LSK-031` reported a **FALSE GREEN**, because `extractHasSuffixEntries` (`scope_fence_test.go:117-132`) calls `os.ReadFile(path)`, which reads the real tree and is unaffected by the Go build overlay. Re-run with a **real edit** in the worktree, both REDs appeared exactly as apply recorded. The worktree was then restored and confirmed **byte-identical** (`git checkout --`, sha256 match) and `git status --porcelain` empty. Bites (b), (e), (g) were not re-planted; (g)'s GREEN half was observed live (see below).

---

### Spec Compliance Matrix

| Requirement | Scenario | Covering test / evidence | Result |
|---|---|---|---|
| `R-CNH-001` | `S-CNH-001` (12 cells) | `TestCombinedMatrix` — 12/12 PASS | ⚠️ **PARTIAL** — child×failure diverges from the requirement's own table (**MAJOR-2**) |
| `R-CNH-001` | `S-CNH-002` | `containmentChildFailure` + `containmentLeafFirst` | ✅ COMPLIANT |
| `R-CNH-001` | `S-CNH-003` (bite a) | re-planted → RED ✅, reverted | ✅ COMPLIANT |
| `R-CNH-001` | `S-CNH-004` | `adjustedThenCompactionFailure` | ✅ COMPLIANT |
| `R-CNH-001` | `S-CNH-005` | `adjustedThenSuspensionFailure` (`requestsAtPark == 1`) | ✅ COMPLIANT |
| `R-CNH-002` | `S-CNH-006` | `agenttest/` diff **empty**; all 5 files `package agent_test`; one `Run` at a time | ✅ COMPLIANT |
| `R-CNH-003` | `S-CNH-007` | `TestSlowConsumerPressure_NeverCancelled_LosesNothing` | ✅ COMPLIANT |
| `R-CNH-003` | `S-CNH-008` (bite b) | apply-recorded RED; not re-planted | ✅ COMPLIANT (accepted) |
| `R-CNH-004` | `S-CNH-009` | `TestSlowConsumerPressure_CancelledUnblocksWithinBound` | ✅ COMPLIANT |
| `R-CNH-004` | `S-CNH-010` (bite e) | apply-recorded hang under `-timeout`; not re-planted | ✅ COMPLIANT (accepted) |
| `R-CNH-005` | `S-CNH-011` | `TestConcurrencyHardening_PackageLeakSweep`, no `t.Parallel()` | ✅ COMPLIANT |
| `R-CNH-005` | `S-CNH-012` (bite c) | re-planted → RED ✅, reverted | ✅ COMPLIANT |
| `R-CNH-006` | `S-CNH-013` | pressure-2 driver inside sweep; `close(release)` after run returns | ✅ COMPLIANT |
| `R-CNH-007` | `S-CNH-014` | `TestCrossRunState_NilHistory_…` + **both defeats re-proven** | ✅ COMPLIANT |
| `R-CNH-007` | `S-CNH-015` | `TestCrossRunState_SharedHistory_…` | ✅ COMPLIANT |
| `R-CNH-007` | `S-CNH-016` (bite d) | re-planted → RED ✅ | ✅ COMPLIANT |
| `R-CNH-008` | `S-CNH-017` | scope fence re-verified by command (below) | ✅ COMPLIANT |
| `R-AGE-008` | `S-AGE-031` | pressure drivers cover the stall half only | ❌ **PARTIAL** — combined-state conjunction unexercised (**MAJOR-1**) |
| `R-LSK-008` | `S-LSK-032` amendment | `TestHooks_…NoReleaseExactWideningInBothFilters` PASS with `t.Logf` fall-through | ✅ COMPLIANT |
| `R-LSK-009` | `S-LSK-034` | bite (f) two REDs ✅ | ⚠️ **PARTIAL** — false clause (**MAJOR-3**) |
| `R-AGS-016` | `S-AGS-067` | four postures verified by construction | ✅ COMPLIANT |
| `R-AGS-009` | `S-AGS-035` parenthetical | doc `0003:2204` left naming `AG-01.1, AG-20.2` | ✅ COMPLIANT |
| `R-CAN-002`/`005`/`008`, `R-RUN-001`, turn-termination non-req | (no new scenario minted) | discharged via `S-CNH-014/015/016`, `S-CNH-011/012/013` | ✅ COMPLIANT |

**Compliance summary**: 17/20 scenarios COMPLIANT, 3 PARTIAL, 0 FAILING, 0 UNTESTED.
**Reverse traceability**: all 7 delta specs walked scenario-by-scenario. **No scenario is unclaimed.** Three are claimed but under-supported (above).

### Scope fence — `S-CNH-017` / `R-CNH-008`, re-verified by command

| Check | Result |
|---|---|
| `git diff main...HEAD -- src/ai/` | **empty** ✅ |
| `git diff main...HEAD -- src/agenttest/` | **empty** ✅ (no widening) |
| `git diff main...HEAD -- go.mod go.sum` | **empty** ✅ (no third-party dep; `goleak` rejection stands) |
| Changed files under `src/agent/` | exactly 8, **all `_test.go`** — **zero production Go** ✅ |
| New `EventKind` / `TurnOutcome` / `CostLabel` | **none declared**; registry guard holds at 25 (`compaction_stream_test.go:98`) ✅ |
| Wall-clock assertion in new tests | **none** — `grep 'time.Sleep|time.Since|time.Now|time.Tick|runtime.NumGoroutine'` over all 5 new files returns **nothing** ✅ |
| Remaining `time.` uses | 3 defensive `time.After(5s)` `t.Fatal` deadlines + 2 `WindDownBound` ceilings — never a synchronization point or an assertion ✅ |
| D1 contingency | **did not fire** ✅ |

### AD-10 filter widening — independently re-extracted

Both filters carry **79** entries; sorted sets **identical**; AG-21's 5 files present in **both**; **no** wildcard, prefix or directory pattern — every entry is an exact `/<name>.go` suffix (`strings.Count(e,"/") == 1`). Confirmed twice: by my own extraction and by the repo's own live guards (`TestScopeFence_S_LSK_031_…` PASS, `TestTurn_SubstrateUntouched` PASS).

### AD-9 branch-scoping — both layers confirmed shipped

`hooks_test.go` carries **both** required guards (`case len(nonTestChanged) == 0:` and `case !containsString(nonTestChanged, "hooks.go"):`), each `t.Logf` + fall-through, never `t.Skip`. Observed live:

```text
hooks_test.go:1991: AG-21: nonTestChanged is empty on this branch — … skipping it here
  and falling through to the diff-independent filter-entry assertions below
--- PASS: TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters (0.04s)
```

PASS, not SKIP — so the filter-entry half ran, which is exactly what design AD-9 promised and what bite (g) part 2 reserved. Design's Open Question 3 is answered: the `hooks.go`-signature layer **exists** in the shipped fix.

### `NFR-HKS-005` / `NFR-CNH-003` compliance

| Obligation | Result |
|---|---|
| No driver samples `runtime.NumGoroutine()` | ✅ zero occurrences across all 5 new files |
| Every gate released **inline** before its driver returns | ✅ `combined_state_fixtures_test.go:343,345,347,428,430,432,514,516,518`; `cross_run_state_test.go:241,315` |
| Every gate additionally carries a cleanup release at construction | ✅ `:293,377,457`; `cross_run_state_test.go:228,293` |
| Tool-release channels closed inline (never `t.Cleanup`) | ✅ `slow_consumer_pressure_test.go:268`, `cross_run_state_test.go:120` — correct, since a `t.Cleanup` would not fire until the 50-repeat sweep ends |
| No goroutine baseline asserted in a gated test | ✅ the only sampler is `RequireNoGoroutineLeak` itself |
| Stalled-observer leak excluded from the sweep | ✅ no driver gates an observing hook; the single `SessionStart` observer is never held |

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD evidence reported | ✅ | `apply-progress` + per-row evidence lines in `tasks.md` |
| All behavioural rows have tests | ✅ | 5 new files, 1,547 lines |
| RED confirmed (test files exist) | ✅ | all 5 present and compiling |
| GREEN confirmed (tests pass) | ✅ | full uncached `-race -count=1` suite, exit 0 |
| Bites RED-first | ✅ | 4 of 7 independently re-planted and watched to FAIL |
| Triangulation | ✅ | 12-cell table + 2 pressure arms + 2 cross-run branches |
| Safety net for modified files | ✅ | `hooks_test.go` / `loop_test.go` / `loop_hook_test.go` edits confined to the released regions; guards re-run green |

**TDD compliance**: 7/7.

### Test Layer Distribution

| Layer | Tests | Files |
|---|---|---|
| Integration (in-process, external `package agent_test`) | 5 top-level + 12 subtests | 5 |
| Unit | 0 | 0 |
| E2E | 0 | 0 |

Appropriate: AG-21's whole subject is assembly behaviour, which is unreachable from a unit boundary.

### Assertion Quality

No tautology, no orphan-empty check, no type-only assertion, no ghost loop, no smoke-test-only, no mock-heavy file. The one absence assertion in the change was **defeat-tested twice** and bites. Coverage was not measured per-file: this change adds only tests and changes no production code, so changed-file coverage is not a meaningful metric here.

**Assertion quality**: ✅ all assertions verify real behaviour.

---

### Issues Found

**CRITICAL**: None.

**MAJOR**

- **MAJOR-1 — `S-AGE-031` is archived as discharged but its central conjunction is never exercised.**
  `specs/agent-event-delivery/spec.md:81-82` requires the structurally stalled consumer **and** one of `R-CNH-001`'s four combined states *on the same run*. All four combined-state arrangers use **buffered** sinks (`combined_state_fixtures_test.go:242,307,407,494`); both unbuffered drivers are standalone (`slow_consumer_pressure_test.go:159,247`). *Failure scenario*: a future regression that breaks decoupling only under the combination — a stalled consumer while a compaction is in flight — ships green, because the claim covering it was archived on evidence that never tested it. `tasks.md:92` must be corrected in the same edit.

- **MAJOR-2 — `R-CNH-001`'s deviation record names the wrong pair of cells and its table is falsified by its own passing test.**
  Six sites listed in V2 above. *Failure scenario*: a reader implementing against `R-CNH-001` builds a child×failure cell asserting `RunOutcomeFailed` (as assertion 4 and the table both require), it fails against correct shipped behaviour, and the reader "fixes" production code to satisfy a requirement that was wrong. All six sites must change together.

- **MAJOR-3 — `S-LSK-034` mints a new clause that is demonstrably false, and no test defends it.**
  The scenario asserts both filters name *"neither `failure.go`, `cost_events.go`, `cost_events_test.go`, `stream_check_test.go` or `reconstruction_test.go`"* (`specs/agent-loop-skeleton/spec.md:88`). Measured directly: `/failure.go` **is present once in each filter** (`loop_test.go:877`) and `/cost_events.go` **is present once in each** — both legitimately released by AG-11 and AG-16 respectively. Only the other three are absent. The nearest implementing assertion (`hooks_test.go:2079-2084`) checks `countString(...) > 1` — i.e. **no duplicate**, not absence — so the clause is defended by nothing. *Failure scenario*: the false clause archives, and the next milestone that legitimately needs `cost_events_test.go` in a filter is told by the spec that a rule exists which the tree already violates. Note this clause is **inherited verbatim from the already-promoted `S-LSK-032`**, so the pre-existing text carries the same defect — but `S-LSK-034` is newly authored by this change and is therefore this change's to get right. Narrow it to the three files that are genuinely absent.

**MINOR**

- **MINOR-1 (V3)** — `combined_matrix_test.go:123-125` states the wrong mechanism: `Schedule`'s return is **captured** at `loop.go:542` on the continuation path (the discard is `loop.go:265`, the nil-continuation path `Harness.Run` never takes). The real reason is that `Schedule` has no error return. Also contains a dangling `R-LSK-...` placeholder.
- **MINOR-2** — `tasks.md:91` (row 6.3) cites the event-kind guards as living in `hooks_test.go`; the `len(kinds) != 25` guards are in `compaction_stream_test.go:98,173`. `hooks_test.go` does reference `EventKinds()`, so the claim is not empty, but the citation does not resolve where it points.
- **MINOR-3** — two of AG-21's own new files are not `gofmt`-clean: `combined_state_fixtures_test.go` (struct-field alignment in `cellCase`) and `slow_consumer_pressure_test.go` (trailing-comment alignment). Whitespace only; `make lint` passes because no gofmt linter is enabled. 15 further files are dirty **pre-existing** and are not this change's.
- **MINOR-4** — `cnhDriveCrossRunNilHistory`'s `release` channel (`cross_run_state_test.go:94`) has no cleanup fallback; a `t.Fatal` inside `readUntilToolStart` before `:120` would leave the deaf tool's goroutine parked. Deliberate and correct for the sweep (a `t.Cleanup` would misreport all 50 iterations), so the tradeoff is right — but it is unrecorded.

**SUGGESTION**

- **SUGGESTION-1** — `adjustedThenChildFailure` and `adjustedThenCompactionFailure` assert only *"not interrupted and not shutdown"*. Asserting `RunOutcomeCompleted` positively would be strictly stronger and would have surfaced MAJOR-2 at authoring time.
- **SUGGESTION-2** — `S-CNH-014`'s spec text says *"the defeat test is **part of this scenario**"*, but the shipped test carries it only as a comment (`cross_run_state_test.go:70-75`) pointing at `apply-progress`. That matches the repo's transient-bite convention and `S-CNH-016` owns it, so it is not a defect — but the wording invites a future reader to look for code that is not there.

---

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| AD-1 shared driver core | ✅ | one `runCombinedCell`, no per-cell assertion copies |
| AD-2 per-cell drive plan | ⚠️ | followed, but the child×failure Then diverged in reality — **MAJOR-2** |
| AD-3 assertion core | ⚠️ | assertions 1–3 uniform across all 12; assertion 4 replaced in 2 cells, one of them unrecorded — **MAJOR-2** |
| AD-4 structural stall | ✅ | unbuffered sink, no clock |
| AD-5 leak sweep, 60s fallback | ✅ | measured ~2.6s; fallback correctly **not** applied |
| AD-6 minted needle + defeat | ✅ | **independently defeat-proven, both halves** |
| AD-7 seven bites | ✅ | 4 re-planted here, 3 accepted on apply's record |
| AD-9 branch-scoping, both layers | ✅ | confirmed live, `t.Logf` fall-through, PASS not SKIP |
| AD-10 exact-suffix widening, both filters | ✅ | 79 == 79, identical, exact; bite (f) two REDs |
| D1 no production code | ✅ | zero production Go changed |
| D5 no `agenttest` widening | ✅ | diff empty |
| `NFR-CNH-005` review budget | ✅ | 1,684 counted lines under the **pre-authorized** `size:exception` — **not a finding** |

### Verdict

**FAIL** — 0 CRITICAL, 3 MAJOR, 4 MINOR, 2 SUGGESTION.

The engineering is correct, the suite is honestly green, and the highest-risk item in the change — the cross-run absence assertion — was defeat-proven to bite. The change is blocked only on **normative text that its own passing tests falsify**. All three MAJOR findings are closable by a **text-only scoped correction** with no new test code and no production change; MAJOR-1 also permits the alternative of building the combined driver.

**Not archive-ready.** Route back to `sdd-apply` for the scoped text correction, then re-verify the corrected artifacts.
