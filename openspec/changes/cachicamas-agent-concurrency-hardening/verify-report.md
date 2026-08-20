```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:67752b65ccf6cc1f79902cd1006c25e5938086c10930e0fe752adb6f3375c096
verdict: fail
blockers: 3
critical_findings: 0
requirements: 14/17
scenarios: 17/20
test_command: cd backend/agent && go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:493212514a6244e6af2438da39b64df2757ff5d14907ed256167ee65f7b70b66
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

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
