```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:005107c1145d18fabb1f7ad6ccfc58e79c428d2a18da78570c52b62cd6191515
verdict: fail
blockers: 2
critical_findings: 2
requirements: 17/25
scenarios: 52/61
test_command: "cd backend/agent && go test -race -count=1 ./..."
test_exit_code: 0
test_output_hash: sha256:c39afd4b42bd7bd94d0a345745873c83ef70afcfd45e4ac43ab8a4f1c3374d36
build_command: "cd backend/agent && go build ./..."
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

# Verification Report — AG-20 `cachicamas-agent-hook-taxonomy`

**Branch** `feat/agent-layer2-wave5-ag20` · **HEAD** `d6dae07426961761005fc98c99fce0f5c82149cf` · **merge-base** `2a138b5997c5e6a3f5e13fcfdc873833ed8975fa`
(`evidence_revision` above is `sha256(HEAD commit id)`, the envelope's required digest form.)

**Verdict: FAIL** — 2 CRITICAL, 8 MAJOR, 5 MINOR.

The implementation is, on the whole, of high quality: the production surface is
correct, the suite is genuinely green on an uncached run, the invariants hold by
measurement, and the substrate discipline is exact. **Every CRITICAL and MAJOR
finding below is a spec-versus-code contradiction or an untested claim, not a
code defect.** Nine scenarios will archive asserting something that is false of
the shipped code, and one regression guard that the change's own correction round
was built around does not fire.

Size is not a finding (`exception-ok`, pre-authorised).

---

## 1. Evidence run — re-executed independently, not taken from the report

```
$ cd backend/agent && go test -race -count=1 ./...
ok   .../src/agent                                    9.599s
ok   .../src/agenttest                                2.524s
ok   .../src/agenttest/sweep                          1.440s
ok   .../src/agenttest/tracetest                      2.180s
ok   .../src/ai                                       4.698s
ok   .../src/ai/internal/retry                        2.647s
ok   .../src/ai/openaicompat                        170.488s
ok   .../src/ai/openaicompat/conformancetest          3.174s
ok   .../src/ai/openaicompat/openrouter               3.457s
ok   .../src/ai/openaicompat/openrouter/conformance   6.654s
?    .../conformance/fixtures                         [no test files]
ok   .../openrouter/internal/smoke                    2.900s
ok   .../src/handoff                                  2.085s
real 171.00   user 63.77   sys 7.49
EXIT=0
```

**171.00s real, exit 0, `-count=1`, zero `FAIL`, no `(cached)` marker anywhere.**
This independently corroborates apply's 172.57s claim. `go build ./...` exits 0
with empty output.

Caveat recorded for honesty: a stray `go test` process from an earlier session
was running concurrently for part of this run, so 171.00s may be marginally
inflated. It does not affect the exit code or the pass/fail result.

---

## 2. Claim-by-claim re-verification

| # | Claim re-run | Method | Outcome |
|---|---|---|---|
| 1 | Full suite green, uncached, ~170s | ran it | **CONFIRMED** — 171.00s, exit 0 |
| 2a | Three anti-vacuity sites converted to skip | `git diff` merge-base | **CONFIRMED** — all three present |
| 2b | `S-TLS-020` / `S-DEL-015` now pass on branch | targeted `-race -count=1 -v` | **CONFIRMED** — both SKIP with the stated messages |
| 2c | `TestHooks_AntiVacuityFloorsSkipRatherThanFail` bites | **planted the anti-pattern** | **REFUTED — CRITICAL-1** (below) |
| 3 | 49-scenario coverage matrix | read every test body | **8 scenarios mis-state or under-prove**; count itself is stale (50 new, not 49) |
| 4 | Five bites RED for the right reason | reproduced `S-HKS-053` | 4 sound; `S-HKS-053`'s stated failure mode is **unreachable** — MAJOR-6 |
| 5 | `S-HKS-011` structural proof | counted call sites | **Proof is REAL** (exactly 4 and 1, none elsewhere) but the scenario demands 8 runs — MAJOR-2 |
| 6 | 6 ADDED-not-MODIFIED requirements (task 7.4) | opened each target | **Deferral is DEFENSIBLE** — see §5. One unrelated falsification found (CRITICAL-2) |
| 7 | Non-negotiable invariants | runtime reflection + grep | **ALL CONFIRMED** — see §3 |
| 8 | Substrate filters byte-in-sync, 4 exact-suffix entries | extracted and diffed both | **CONFIRMED** — 74 entries each, identical and same order |
| 9 | Recorded release bounded; no stale byte-unchanged claims | grepped whole change dir | **One stale claim found** — MAJOR-4 |
| 10 | Doc 0003 truthfully updated | read diff, verified prose vs code | Ticks and narrative correct; counter carries a pre-existing off-by-one — MINOR-4 |

---

## 3. Non-negotiable invariants — all hold (measured, not read)

| Invariant | Method | Result |
|---|---|---|
| `len(agent.EventKinds()) == 25` | runtime reflection probe | **25** ✅ |
| `*agent.Harness` has exactly 5 methods | runtime reflection probe | **5**: `Compact, Interrupt, Run, Shutdown, Steer` ✅ |
| No wall clock in production | grep `hooks.go` | no `"time"` import, no `After/Sleep/Timer/Ticker/WithTimeout/WithDeadline` ✅ |
| No poll in the lane | read `observerLane.drain` | `sync.Cond.Wait()` — blocking, not polling ✅ |
| No clock in assertion paths | grep 4 AG-20 files | `time` used only for `time.Duration` in injected `SleepFunc` signatures ✅ |
| **Observers get `context.Background()`, NOT `WithoutCancel`** | `hooks.go:338,357` | **`obs(context.Background(), report)`** ✅ |
| Observer cannot reach `DelegationSeamFrom` | structural + behavioural | `Background()` carries **no values**, so the exported value lookup returns nothing. Behaviourally confirmed by `S-DEL-026` and `S-AGE-030` child-run fixtures ✅ |
| 20 named files byte-unchanged | `git diff` merge-base, file by file | all 20 unchanged ✅ |
| `src/ai/`, `src/agenttest/`, `go.mod`, `go.sum` | `git diff` | all **empty** ✅ |
| Non-test files changed | `git diff --name-only` | exactly `{loop.go, harness.go, compaction.go, hooks.go}` ✅ |

This is the load-bearing one and it holds: `context.WithoutCancel` would have
preserved values and handed an observer the parent's publishing seam. The code
uses a freshly rooted `Background()` context, closing the path by construction.

---

## 4. Findings

### CRITICAL-1 — `TestHooks_AntiVacuityFloorsSkipRatherThanFail` is a dead guard

I planted the exact anti-pattern back into `scope_fence_test.go` (in-place
`t.Skip` → `t.Fatal` at the empty-diff branch) and ran the guard:

```
--- PASS: TestHooks_AntiVacuityFloorsSkipRatherThanFail/scope_fence_test.go (0.01s)
```

**It passed with the mechanism fully defeated.** Root cause — the regex at
`hooks_test.go:2164`:

```go
emptyDiffThenFatal := regexp.MustCompile(`(?s)[Dd]iff\s*==\s*""\s*\{[^}]{0,300}t\.Fatal`)
```

`[^}]{0,300}` allows at most 300 characters between the opening brace and
`t.Fatal`. At the planted site the explanatory comment makes that gap **717
characters**. I then simulated an in-place `t.Skip`→`t.Fatal` reversion at all
three protected sites:

| Site | In-place reversion caught? |
|---|---|
| `scope_fence_test.go` | **No** |
| `cancellation_test.go` | **No** |
| `hooks_test.go` | **No** |

The guard fires only on a *short-form* reintroduction (< 300 chars), which is not
the shape any of the three sites has — each carries a long explanatory comment
precisely because of this milestone's own fix. The single most likely regression
(someone "restores the strict check" in place) is invisible to it.

This directly falsifies `S-LSK-033`'s closing clause — *"a future edit that
reintroduces a fatal on an empty diff is **caught mechanically** rather than
discovered on the next descendant branch"* — and defeats the guard's stated
purpose of protecting AG-21/22/23. It is the project's own recorded
"a fix can re-encode the defect" pattern: the correction round closed the
original defect and shipped a guard that cannot detect its return.

*Worktree restored; `git status` clean; guard re-confirmed green afterwards.*

### CRITICAL-2 — AG-20 falsifies the shipped `L2C-08` package contract row, with no delta anywhere

`doc.go:36` (byte-unchanged, and *required* to be by `R-LSK-008` consequence 4)
states as a package-wide guarantee:

> `L2C-08` … after the documented wind-down bound only third-party tool code may
> remain running — reported typed by tool and call identity — **and every
> goroutine the package itself owns has exited**.

AG-20 ships a per-run observer lane goroutine that is package-owned and, by
design, does **not** exit when an observer stalls. `hooks.go`'s own comment says
so plainly:

> *"A permanently stalled observer (this goroutine blocked inside
> `invokeRecovered`) never reaches that check again: the goroutine **leaks by
> design**, the caller's leak (R-HKS-008 consequence 1)."*

And `S-HKS-017`'s passing test asserts exactly this: `Run` **returns while the
hold is still held**. `NFR-HKS-005` hands "the permanently-stalled leak" to AG-21.

`R-CAN-006`'s "Harness-owned tasks" row enumerates the goroutines that MUST exit
within the wind-down bound; the observer lane is a new member of that class and
appears in no enumeration. Yet:

- `agent-cancellation-tree` is **not** among this change's eleven deltas.
- `R-HKS-011`'s closed-sequence table — which exists precisely to catch *"a new
  producer or a new goroutine landing inside a requirement that never mentions
  it"* — has a row for `R-CAN-002`/`S-CAN-013` (the **event tail**) but **no row
  for `R-CAN-006`/`R-CAN-008`'s goroutine-exit claim**. The table missed its own
  case.

Archive would promote a package-wide contract row that the shipped code makes
false. Needs either an `agent-cancellation-tree` delta scoping the claim, or an
`R-HKS-011` row recording it explicitly.

### MAJOR-1 — `S-HKS-026` (the central inertness proof) compares the change against itself

`TestHooks_Inertness_ByteIdenticalOnEveryArm` has 7 arms. **Not one sets `Hooks`
at all**, and none references the merge base. Each arm runs the *same hookless
configuration twice on the same branch* and compares only
`kindsEqualHKS(first, second)` — the sequence of `EventKind` values.

The scenario claims: *"six paired runs … **one of each pair on the merge base**
and one on this change with a zero-value `Hooks` … each pair's event streams are
byte-identical …, each pair's **history read-backs** are byte-identical, each
pair's **captured provider requests** are byte-identical."*

What is actually proven: the hookless path is *deterministic across two runs*,
and `runtime.NumGoroutine()` returns to baseline (real and valuable — no lane on
the inert path). What is **not** proven: anything relative to pre-AG-20, history
read-backs, or provider requests.

AG-19's `inert_path_test.go` uses the identical shortcut but **justifies it
explicitly** (`inert_path_test.go:86-95`): its diff installs a seam that is
*"never read"*, gated behind an opt-in accessor, so two-runs-on-this-change is
"the same experiment". **That argument does not transfer to AG-20**, which adds
code that executes on the hookless path — `applyPreRequestHook` widened to chain
composition in `loop.go`, the `turnCost`/`capturedOutcome` capture in
`harness.go`, the `len(chain) > 0` guard in `compaction.go`. AG-20 inherits the
technique without restating or re-earning the justification.

### MAJOR-2 — `S-HKS-011` demands eight runs; two shipped

Scenario text: *"Given **eight runs**, one driving each of rows 7–14, when each
run returns, then the observer's recorded report count is zero and the stall
reporter's report set is empty."*

Delivered: rows 13 and 7 behaviorally; rows 8–12 and 14 rest on a structural
count. **I verified the structural proof is real** — `harness.go` contains
exactly 4 `lane.enqueuePostTurn(` sites and exactly 1 `lane.enqueueSessionStart(`
site, and no other production file calls either. That bounds *where* firing can
happen; it does not show those six exit paths avoid the four known sites.

Honestly disclosed in `apply-progress.md` and in the test's own comment — but the
scenario text was never amended to match, so it archives claiming eight runs.

### MAJOR-3 — `S-HKS-023` and `S-PRH-009` assert an insertion property the code does not have

Both require: *"given the failing element … with **two additional elements
inserted before it** so its composed position moves … the recorded attribution
string is **unchanged**."*

Attribution is `Hooks.PreRequest[i]` — the element's **slice index**. Inserting
two elements before index 1 moves the failing function to index 3, so attribution
becomes `Hooks.PreRequest[3]`: **changed**. The test declines to test this and
documents why (`hooks_test.go:744-757`): inserting *within* `Hooks.PreRequest` is
"a DIFFERENT registration … so THAT case is not what this scenario tests". It
instead compares with/without the **singular field**.

The reasoning is correct and `R-HKS-009`'s own prose is right (*"stable under
insertion **ELSEWHERE**"*). The two **scenarios** overreach and will archive false.

### MAJOR-4 — `S-HKS-001` still claims `scope_fence_test.go` is byte-unchanged

`agent-hook-taxonomy/spec.md:63`:

> *"…`scope_fence_test.go` and `harness_test.go`'s named-method assertion are
> **byte-unchanged** and green…"*

`scope_fence_test.go` is **not** byte-unchanged — it carries the recorded release
(13 changed lines). `R-HKS-010` consequence 4 removed it from the byte-unchanged
list and says so explicitly; `S-HKS-001`'s text was not updated with it. The test
is correct (`hksScopeFenceByteUnchangedFiles()` lists 20 files and **excludes**
`scope_fence_test.go`). This is the project's recorded *"correcting a requirement
leaves its scenarios wrong"* pattern.

### MAJOR-5 — `S-HKS-012`'s cost clauses are unachievable

Scenario requires *"each attempt carrying a **distinct, non-zero** usage figure"*
and *"that sum is **strictly greater** than the last attempt's own figure on at
least one reported figure"*.

Retried attempts never reach `finalize()`, the only site emitting `cost_turn`, so
they contribute **zero** cost events. The test correctly asserts
`costTurnCount == 1` and `report.Cost() == streamCost` — i.e. the sum **equals**
the single (last) attempt's figure. Both clauses are false; the test documents
the impossibility at `hooks_harness_test.go:713-727`. The attempt count (3) is
genuinely proven.

### MAJOR-6 — bite `S-HKS-053`'s stated failure mode is unreachable (orchestrator asked for an explicit judgment)

I reproduced the sabotage (`cut = resolveCut(hist, plan.Cut())` → `cut = plan.Cut()`):

```
hooks_compaction_test.go:287: compaction_finished count = 0, want exactly 1
--- FAIL: TestHooksCompaction_ForwardAdjustment_StaysInvariantSafe
```

Exactly as apply reported. The scenario says it *"FAILS **with the committed
prefix splitting a call/result pair**"*. It does not — `hist.markSpan(3)` reports
`!spanOK` and the bracket takes its typed failure arm.

I probed further: I **also** neutralised the post-hook `markSpan` guard. It
*still* failed with zero `compaction_finished` and no split prefix —
`buildLoopRequest` refuses a pairing-broken prefix. So pairing safety is
**three-layered**, and the split the scenario describes is **not reachable** by
this sabotage at all.

**Judgment: the bite still proves the property** — removing the re-resolution
turns a successful compaction into a failed one, so the step is genuinely
load-bearing. **But the scenario's stated observable is wrong and must be
corrected**, not archived. Apply was right to flag it and right not to decide it
alone.

### MAJOR-7 — `S-RUN-113` asserts a four-method set; the set is five

Scenario: *"its method set is exactly the **four** names `R-RUN-001` enumerates"*.
Measured: **five** (`Compact, Interrupt, Run, Shutdown, Steer`). The test is
correct — it asserts all five. Root cause is a **pre-existing** defect:
`R-RUN-001`'s table still lists four and was never back-annotated when AG-18
added `Compact`. AG-20 did not create it, but `S-RUN-113` **restates it as a new
claim**, which makes it this change's to fix.

### MAJOR-8 — `tasks.md`'s scenario count is stale (count-assertion drift)

`tasks.md:60`: *"**49 scenarios** … each discharged once, none doubled."* The
change now declares **50** new scenarios. `S-LSK-033`, added by the correction
round, is absent from the coverage table and cited by **no test by ID**. This is
exactly the recorded *"count assertions are a drift class"* failure — a later
append silently falsified the count and nothing failed.

Authoritative counts I measured across the 11 delta files: **25 requirements, 61
scenario declarations** (61 = 50 new + 11 pre-existing restatements).

---

## 5. Task 7.4 discharged — the ADDED-not-MODIFIED judgment

I opened each targeted original rather than accepting the deferral claim.

**Verdict: the ADDED-not-MODIFIED choice is DEFENSIBLE.** Two independent reasons:

1. **The change does use `MODIFIED` where genuine amendment occurs** — `R-CMP-004`
   (forward-adjustment scoping), `R-PRH-002/003/005/007` (chain composition,
   source attribution, nil-default condition, determinism), `R-AGE-008`
   (invariant-3 back-annotation), plus MODIFIED non-requirement/dependency
   blocks. So the ADDED requirements are not being used to dodge amendment.
2. **None of the targeted originals is falsified by AG-20:**

| Targeted original | Falsified? | Evidence |
|---|---|---|
| `R-RUN-001` (named method surface) | **No** | AG-20 adds a **field**, not a method; measured method set unchanged at 5. (Its own four-vs-five staleness is AG-18's, pre-existing) |
| `R-RUN-003` (one bracket, one lane) | **No** | lane emits nothing, stamps nothing, holds no sink |
| `R-RUN-004` / `R-RUN-010` (no third path, no timeout) | **No** | no clock added; no-deadline pin file-unchanged (asserted and passing) |
| `R-ATT-001` (outcome vocabulary) | **No** | no member added; `S-ATT-015` asserts reported outcome == streamed outcome |
| `R-CMP-001/013/014` | **No** | failures land on the **existing** arm; run continues; verified in `compaction.go` |
| `R-DEL-009` (scope fence) | **No** | no kind, no method, no signature change — all measured |
| `R-DEL-010` (closed sequence) | **No** | every row concerns **event emission**; AG-20 emits nothing |
| `R-AGS-007/012/013`, `R-AIV-007` | **No** | Layer 1 untouched (`src/ai/` diff empty); no concrete hook ships |

The `MODIFIED`-block-truncation argument (a `MODIFIED` block must reproduce the
requirement in full or archive silently drops the remainder) is a real, recorded
hazard in this repository, and `R-LSK-004` at ~65 lines is a genuine instance.

**However** — the one genuine falsification I did find (CRITICAL-2, `L2C-08` /
`R-CAN-006`) is recorded in **no delta at all**. That is a *missing delta*, not a
MODIFIED-vs-ADDED error, and it is the blocker.

---

## 6. Scenario compliance matrix

Legend: ✅ covered and truthful · ⚠️ covered but text over-claims · ❌ not proven as written

### New `S-HKS-*` scenarios (31, incl. 5 bites)

| Scenario | Covering test | Verdict |
|---|---|---|
| S-HKS-001 | `TestHooks_OneSurface_HarnessMethodSetUnwidened` | ⚠️ MAJOR-4 (stale byte-unchanged clause) |
| S-HKS-002 | `TestHooks_TransportRefusesRatherThanOverwrites` | ✅ |
| S-HKS-003 | `TestHooks_ObservingUnconstructible_DirectTurnInert` | ✅ |
| S-HKS-004 | `TestHooks_PreRequestChain_CompositionOrderSingularFirst` | ✅ |
| S-HKS-005 | `TestHooks_PreRequestChain_ZeroValueByteIdenticalToSkeleton` | ✅ (AG-08 suite byte-unchanged, verified) |
| S-HKS-006 | `TestHooksCompaction_Splice_BothDoors_PreProviderOrdering` | ✅ |
| S-HKS-007 | `TestHooksCompaction_Idempotence_IdenticalPlanByteIdenticalToNoHook` | ⚠️ "byte-identical" asserted as kind-sequence (MINOR-5) |
| S-HKS-008 | `TestHooksCompaction_ForwardAdjustment_StaysInvariantSafe` | ✅ (strengthened to check stream counts) |
| S-HKS-009 | `TestHooksCompaction_ChainElementFailure_LandsOnExistingArm_RunContinues` | ✅ |
| S-HKS-010 | `TestHooksHarness_PostTurn_EveryYesRowFiresExactlyOnce` | ✅ |
| S-HKS-011 | `TestHooksHarness_PostTurn_NoRowFiresNothing` | ❌ MAJOR-2 (2 of 8 rows) |
| S-HKS-012 | `TestHooksHarness_PostTurn_CostIsSumOverAttempts` | ❌ MAJOR-5 (two unachievable clauses) |
| S-HKS-013 | `TestHooksHarness_SessionStart_OnceAcrossTwoSerialRuns` | ✅ |
| S-HKS-014 | `TestHooksHarness_SessionStart_DelegatedChildFiresOwn` | ✅ |
| S-HKS-015 | `TestHooksHarness_SessionStart_ShutdownFiresNone_DegenerateRunStillFires` | ✅ |
| S-HKS-016 | `TestHooks_Ordering_RegistrationOrderAtAllFourPoints` | ✅ |
| S-HKS-017 | `TestHooks_Asynchrony_DeliveryUnimpeded_RunReturnsWithHoldHeld` | ⚠️ MINOR-5 (kind-sequence, not bytes) |
| S-HKS-018 | `TestHooks_Asynchrony_GoroutinePlacement_NoRunFrame` | ✅ strong — real stack capture, no clock |
| S-HKS-019 | `TestHooks_Asynchrony_EventuallyReported_OutstandingAndQueued` | ✅ |
| S-HKS-020 | `TestHooks_Panic_ReportedOnLane_LaneContinues_ProcessSurvives` | ✅ (isolated-process RED recorded) |
| S-HKS-021 | `TestHooks_Reporter_NilReportsNothing_StallingStallsBothObservables` | ✅ |
| S-HKS-022 | `TestHooks_PreRequestFailure_AttributesBySourceName_AbortsBeforeIO` | ✅ strong |
| S-HKS-023 | `TestHooks_PreRequestFailure_SingularNamedDistinctly_InsertionDoesNotRenumber` | ❌ MAJOR-3 |
| S-HKS-024 | `TestHooks_ScopeFence_ByteUnchangedFilesAndNoNewKind` | ✅ (see MINOR-1 on skip placement) |
| S-HKS-025 | `TestHooks_ClosedSequenceTable_HoldsAmendedAndClarifiedRowsResolve` | ⚠️ table itself is missing the `R-CAN-006` row (CRITICAL-2) |
| S-HKS-026 | `TestHooks_Inertness_ByteIdenticalOnEveryArm` | ❌ MAJOR-1 |
| S-HKS-050 (bite a) | `TestHooks_AntiVacuity_NoStallEmptyReportSet` | ✅ RED reproduced by apply, shape correct |
| S-HKS-051 (bite b) | `TestHooks_Asynchrony_GoroutinePlacement_NoRunFrame` | ✅ fails as assertion, not hang — as designed |
| S-HKS-052 (bite c) | `TestHooks_Ordering_RegistrationOrderAtAllFourPoints` | ✅ `[3 2 1 0]` vs `[0 1 2 3]` |
| S-HKS-053 (bite d) | `TestHooksCompaction_ForwardAdjustment_StaysInvariantSafe` | ❌ MAJOR-6 — **I reproduced it**; property proven, stated observable wrong |
| S-HKS-054 (bite e) | `TestHooksHarness_SessionStart_OnceAcrossTwoSerialRuns` | ✅ count 2 vs 1 |

### Amendment-delta scenarios (19 new)

| Scenario | Covering test | Verdict |
|---|---|---|
| S-PRH-008 | `TestHooks_PreRequestChain_FinalOutputOnly_...` | ✅ |
| S-PRH-009 | `TestHooks_PreRequestFailure_SingularNamedDistinctly_...` (no ID citation) | ❌ MAJOR-3 |
| S-PRH-010 | `TestHooks_PreRequestChain_NilSingularDoesNotSkipNonEmptyChain` | ✅ |
| S-PRH-011 | `TestHooks_PreRequestChain_DeterministicAndOrderSensitive` | ✅ order-sensitivity genuinely asserted |
| S-CMP-038 | `TestResolveCut_FixedPointOnItsOwnOutput` + compaction tests | ✅ |
| S-CMP-039 | `TestHooksCompaction_ChainElementFailure_...` | ✅ |
| S-CMP-040 | `TestResolveCut_FixedPointOnItsOwnOutput` | ✅ |
| S-CMP-041 | `TestHooksCompaction_ForwardAdjustment_...` | ✅ |
| S-CMP-042 | `TestHooksCompaction_MisplacedOptionsRejection_TotalForHooksField` | ✅ member-by-member |
| S-LSK-032 | `TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters` | ✅ verified independently (§2 #8) |
| S-LSK-033 | `TestHooks_AntiVacuityFloorsSkipRatherThanFail` (no ID citation) | ❌ **CRITICAL-1**; first clause untested |
| S-AEV-126 | `TestHooks_S_AEV_126_Invariant3ClosureCheckedAgainstShippedMechanism` | ✅ |
| S-AGE-030 | `TestHooks_S_AGE_030_StalledHookTraceHasNoPathBack_ChildObserverFindsNoSeam` | ✅ seam-absence genuinely asserted |
| S-RUN-113 | `TestHooksHarness_S_RUN_113_AdditionsExactlyFourAbsencesAsserted` | ❌ MAJOR-7 |
| S-ATT-015 | `TestHooksHarness_S_ATT_015_ReportedOutcomeIsStreamedOutcome` | ✅ |
| S-DEL-026 | `TestHooksHarness_S_DEL_026_ChildObserverFindsNoSeam_FenceGreenUnedited` | ✅ |
| S-AGS-065 | `TestHooks_S_AGS_065_DischargeIsAuditableNotAsserted` | ✅ |
| S-AGS-066 | `TestHooks_S_AGS_066_Layer3HalfCheckedAgainstShippedSurface` | ✅ |
| S-AIV-032 | `TestHooks_S_AIV_032_ExclusionCheckedAgainstShippedCode` | ✅ |

Pre-existing restatements (`S-AGE-010/011`, `S-CMP-010/011/012/013/030`,
`S-PRH-001/002/003/006`) are covered by their existing green owning tests; each
delta's "AG-20 update" note is accurate.

---

## 7. MINOR findings

1. **`hooks_test.go:1560` skip placement.** The `t.Skip` precedes four
   *diff-independent* assertions in the same function (kind count, `Harness`
   method count, `Turn` and `Run` signatures). On any future branch that does not
   touch `loop.go`, all four silently stop running. Mitigated: `scope_fence_test.go:103`
   (`TestScopeFence_S_DEL_024_…`, which cannot skip) asserts the same invariants
   unconditionally. Recommend moving the skip below them.
2. **`S-LSK-033`'s bounding clause is inaccurate for `cancellation_test.go`.** It
   says the only change converts one branch *"to a call to `t.Skip`"*. Actually
   that file converts the branch to `continue`, adds a `scannedAny` flag and two
   assignments, and adds a **separate** terminal `t.Skip`. `apply-progress.md`
   describes this correctly; the scenario does not.
3. **`TestCancellation_NestedRunCancelsLeafFirst` now reports `SKIP` although its
   entire behavioural core ran** (the skip is terminal). Sound, but misleading to
   anyone scanning for skipped coverage — and inconsistent with the same file's
   `t.Logf` + `return` precedent 12 lines earlier.
4. **Doc 0003 counter.** The wave table sums to 24 milestones (AG-00…AG-23);
   shipped through AG-20 is **21**, but the status line says **"20 of 24"** — a
   pre-existing off-by-one AG-20 carried forward (prior value was 19). The
   rewritten status line also re-publishes the now-stale *"5 of 7 wave-2
   milestones shipped"* (the table lists Wave 2 as 5 milestones, all shipped).
   The completion checklist additionally leaves the AG-03/04/05/06/07/08/09 rows
   unticked — all pre-existing. **AG-20's own two ticks and its narrative
   sentence are accurate**: I checked each clause against the code.
5. **"Byte-identical" is implemented as `EventKind`-sequence equality**
   (`kindsEqualHKS`) in `S-HKS-007/017/026` and `S-RUN-113`. Payload fields,
   history read-backs and captured provider requests are not compared. Weaker
   than the scenario text in every case.

---

## 8. TDD compliance

| Check | Result | Detail |
|---|---|---|
| TDD Evidence reported | ✅ | Full table in `apply-progress.md` |
| All tasks have tests | ✅ | 51/54; 7.2/7.3 are archive obligations, 7.4 discharged here |
| RED confirmed (test files exist) | ✅ | all four AG-20 files present |
| GREEN confirmed (tests pass) | ✅ | independently re-run, exit 0 |
| Bites RED for the right reason | ⚠️ 4/5 | `S-HKS-053`'s observable is misdescribed (MAJOR-6) |
| Regression guard bites | ❌ | CRITICAL-1 |
| Triangulation | ✅ | member-by-member, both-doors, six-arm and four-point fixtures throughout |

**Assertion quality:** no tautologies, no ghost loops, no orphan empty-checks, no
mock-heavy tests — every scenario drives a real `Harness`/`Turn`/`Compact` call
through `agenttest.Provider`. The defects are *over-claiming scenario prose* and
*two under-powered comparisons* (MAJOR-1, MINOR-5), not trivial assertions.

**Quality gates** (re-run): `go build ./...` exit 0, empty output. `gofmt`, `go vet`,
`golangci-lint`, `make vuln-check` as reported by apply — not independently re-run.

---

## 9. Known items deliberately NOT re-raised

- `agent-pre-request-hook`'s header count (*"6 charter → 7 spec + 2 bites = 9
  total"*, `:7`, `:11-15`, `:147`) — **confirmed recorded** as an `sdd-archive`
  promotion obligation at `tasks.md:133` (task 7.2). Not a new finding.
- `S-AGS-064` permanently unreused — recorded at task 7.3.
- PR size — pre-authorised `size:exception`.

---

## 10. Required before archive

**Blockers (2):**

1. **CRITICAL-1** — repair `TestHooks_AntiVacuityFloorsSkipRatherThanFail` so it
   detects an in-place reversion at all three sites (e.g. widen/remove the 300-char
   window, or match on the enclosing block rather than a character budget), and
   **prove it bites** by planting the anti-pattern. Then reconcile `S-LSK-033`.
2. **CRITICAL-2** — record the observer-lane goroutine against `R-CAN-006` /
   `R-CAN-008` / `L2C-08`: either an `agent-cancellation-tree` delta scoping the
   "every goroutine the package itself owns has exited" clause, or a new
   `R-HKS-011` row. Do not archive a package contract row the code falsifies.

**Scenario-text corrections (MAJOR 2–8):** `S-HKS-001`, `S-HKS-011`, `S-HKS-012`,
`S-HKS-023`, `S-HKS-026`, `S-HKS-053`, `S-PRH-009`, `S-RUN-113`, plus `tasks.md`'s
scenario count and coverage table. Each will otherwise archive as a false claim.
`S-HKS-011` and `S-HKS-026` additionally need either stronger fixtures or an
explicit, justified narrowing.

**Next phase:** `sdd-apply`.
