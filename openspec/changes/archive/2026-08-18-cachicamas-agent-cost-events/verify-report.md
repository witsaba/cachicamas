```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:f6523aac23a201e8b70c75973419394379c05277835e77057204537071a9d845
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 15/15
scenarios: 58/58
test_command: cd backend/agent && go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:0e0e3bcd97be6b523c5465b56a6ebdfdc92cac4141a20401e82386fdf27bfdbe
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report

**Change**: `cachicamas-agent-cost-events` (AG-16 — Emit cost and usage events)
**Worktree**: `cachicamas-worktrees/ag16`, branch `feat/agent-layer2-wave3-ag16`, HEAD `8e90d9cf`, base `origin/main@09bb30e1`
**Mode**: Strict TDD
**Verdict**: **PASS WITH FINDINGS** — 0 CRITICAL, 3 WARNING, 3 SUGGESTION

Every claim below is marked `verified-by-command` with its command and result, or `unverified` with the reason.

### Completeness

| Metric | Value | Evidence |
|---|---|---|
| Tasks total | 58 | `grep -c '^\s*-\s*\[x\]' tasks.md` = 58, `[ ]` = 0 — verified-by-command |
| Tasks complete | 58 | same; `apply-progress.md` independently = 58/0 — verified-by-command |
| Tasks incomplete | 0 | verified-by-command |

### Build & Tests Execution

**Tests**: PASS — 12/12 packages `ok`, exit 0, uncached, under `-race`.

The apply phase's `make test` was served from the Go build cache (0.97s wall, `(cached)` markers). I discarded that and re-ran uncached:

```text
cd backend/agent && go test -race -count=1 ./...
ok  .../src/agent                                    9.623s
ok  .../src/agenttest                                2.233s
ok  .../src/agenttest/sweep                          2.180s
ok  .../src/agenttest/tracetest                      1.933s
ok  .../src/ai                                       4.719s
ok  .../src/ai/internal/retry                        2.605s
ok  .../src/ai/openaicompat                        171.614s
ok  .../src/ai/openaicompat/conformancetest          2.877s
ok  .../src/ai/openaicompat/openrouter               3.389s
ok  .../src/ai/openaicompat/openrouter/conformance   6.839s
ok  .../src/ai/openaicompat/openrouter/internal/smoke 2.468s
ok  .../src/handoff                                  2.100s
EXIT=0
```

**Build**: PASS — `make build` exit 0 — verified-by-command.
**Lint**: PASS — `golangci-lint cache clean && make lint` → `go vet` clean, `0 issues.` — verified-by-command (cache cleared first, per the known lint-cache-artifact hazard).
**gofmt**: PASS — `gofmt -l` run per-file over all 10 changed `.go` files: empty output — verified-by-command.
**Vulnerabilities**: PASS — `./bin/govulncheck ./...` → `No vulnerabilities found.`, exit 0 — verified-by-command.
**Coverage**: not run — not required by the acceptance criteria and not part of the milestone's evidence gate.

### The three charter AG-16.1 scenarios — do the tests actually falsify the claim?

I did not accept "a covering test exists". For each I read the test, identified the discriminating assertion, and where the spec names a bite I re-ran the mutation myself with `go test -overlay` (the tree was never modified; `git status --short` empty afterwards — verified-by-command).

#### Scenario 1 — "per-turn cost is exact and honest about absence"

| Check | Result | Evidence |
|---|---|---|
| Mixed-record case covered (the exact named defect) | **YES** | `cost_turn_emission_test.go:479` `TestTurn_CostTurn_MixedRecordPartialPresence` scripts `ai.Usage{Input: ai.Tokens(100)}` and asserts `InputTokens()==(100,true)` while output/cacheRead/cacheWrite/reasoning each assert `ok==false`. Reads the discriminator per figure — verified-by-command (source read + test run) |
| Distinguishes `ai.Tokens(0)` from absent | **YES, non-vacuously** | `:422` `TestTurn_CostTurn_AbsenceVsReportedZero` runs two turns whose counts are **both 0** and asserts `okAbsent != okZero`. Two events with equal counts cannot satisfy it — the spec's own anti-vacuity rule is met — verified-by-command |
| Bite `S-CST-020` genuinely RED | **REPRODUCED** | I collapsed `costFromUsage`'s per-figure presence to a whole-record `anyReported` flag via overlay. Result: `TestTurn_CostTurn_MixedRecordPartialPresence` FAILED with `OutputTokens() = (0, true), want (0, false)` (and the same for the other three) — byte-matching apply-progress task 1.8. Critically, `TestTurn_CostTurn_AbsenceVsReportedZero` **still passed** under the mutation, confirming the spec's claim that S-CST-005 alone would have shipped the defect — verified-by-command |

#### Scenario 2 — "cumulative counts every attempt"

| Check | Result | Evidence |
|---|---|---|
| Retry-inclusive by assertion, not by assumption | **YES** | `cost_session_test.go:248`. `wantSum := sumUsageFromCostTurns(events)` is computed from the `cost_turn` events on the *same recorded stream*, then compared to the terminal `cost_session` — an observed-event equality, never a hand-computed literal (the repo's known count-assertion drift class is avoided). A non-vacuity guard at `:324` requires the summed input to be positive and reported |
| Counterfactual: would it pass if the retried attempt's tokens were dropped? | **NO — it fails** | I ran **both** mutations. (a) apply's `+=`→`=` overwrite: FAILED `InputTokens() = (19, true), want (719, true)` — byte-matching apply-progress task 3.5. (b) **the spec's literal Given**, an accumulator that skips a retried attempt's `cost_turn` (`total.add(ct)` guarded by `attempt == 1` in `harness.go`): FAILED with the identical five-figure output. So `S-CST-021` archives as true under its own stated mechanism — verified-by-command |

#### Scenario 3 — "late usage corrects an estimate"

| Check | Result | Evidence |
|---|---|---|
| Observes a real scripted sequence, not an unconditional label | **YES** | `cost_session_test.go:395` drives a genuine two-logical-turn run (`PauseTurn` then `Stop`), asserts ≥1 `Estimate` positioned strictly between the turn brackets, and that the last `cost_session` is `Final`, sits immediately before `run_end`, and equals the cumulative. `:471` `TestHarness_SingleTurnRun_FinalOnly` covers the conditional half (no fabricated estimate) |
| Bite `S-CST-022` genuinely RED | **REPRODUCED** | I mutated the success-close emission at `harness.go:676` from `CostLabelFinal` to `CostLabelEstimate`. Both partners FAILED: `cost_session immediately before run_end has label estimate, want Final` and `sole cost_session label = estimate, want Final` — byte-matching apply-progress task 3.8. The assertion reads the terminal event's own label, not a cost-event count — verified-by-command |

**All three bites reproduced independently. None was a paper claim.**

### Wall-clock reachers

**NONE** — verified-by-command, by runtime and by reading, not by grepping `time.Sleep`.

- Grep for `time.`/`Sleep`/`Deadline`/`Timeout`/`After(` across the three new test files returns exactly one hit: `RetryTiming{SleepFunc: instantSleep}` (`cost_session_test.go:284`) — the injected no-op seam AG-15 introduced (`retry_backoff_test.go:33`).
- Per-test runtimes under `-race`, uncached: all 17 new tests report `0.00s` except `TestCost_ScopeFence` at `0.05s` (it shells out to `git diff`). The retry-bearing test — the one AG-15 was bitten on — runs at `0.00s`, proving it never reaches the real clock.

### Spec Compliance Matrix (new capability `agent-cost-events`)

| Requirement | Scenario | Test | Result |
|---|---|---|---|
| R-CST-001 | S-CST-001 | `TestTurn_CostTurn_FiguresExactPerTurn` | COMPLIANT |
| R-CST-001 | S-CST-002 | `TestTurn_NoCompletion_CostTurnAllAbsent` | COMPLIANT |
| R-CST-001 | S-CST-003 | `TestTurn_AbortedTurn_NoCostTurn` (5 subtests) | COMPLIANT |
| R-CST-001 | S-CST-004 | `TestTurn_CostTurn_LabelAlwaysFinal` + inline extension over the retry stream | COMPLIANT |
| R-CST-002 | S-CST-005 | `TestTurn_CostTurn_AbsenceVsReportedZero` | COMPLIANT |
| R-CST-002 | S-CST-006 | `TestTurn_CostTurn_MixedRecordPartialPresence` | COMPLIANT |
| R-CST-002 | S-CST-020 (bite) | reproduced by verify via overlay | COMPLIANT |
| R-CST-003 | S-CST-007 | `TestCostFromUsage_TableDriven` (+ zero-value payload test) | COMPLIANT |
| R-CST-004 | S-CST-008 | `TestHarness_CostSession_CumulativeEqualsEmittedCostTurns` | COMPLIANT |
| R-CST-004 | S-CST-021 (bite) | reproduced by verify, both mechanisms | COMPLIANT |
| R-CST-005 | S-CST-009 | `TestHarness_CostSession_EstimateThenFinal` | COMPLIANT |
| R-CST-005 | S-CST-010 | `TestHarness_SingleTurnRun_FinalOnly` | COMPLIANT |
| R-CST-005 | S-CST-011 | `TestTurn_Standalone_NoCostSession` | COMPLIANT |
| R-CST-005 | S-CST-022 (bite) | reproduced by verify via overlay | COMPLIANT |
| R-CST-006 | S-CST-012 | `TestHarness_CostSession_FinalOnFailedRun` (2 subtests) | COMPLIANT |
| R-CST-006 | S-CST-013 | `..._FinalOnInterruptedRun` + `..._FinalOnShutdownRun` | COMPLIANT |
| R-CST-007 | S-CST-014 | `TestCost_ScopeFence` + unedited `cost_events_test.go` / `event_registry_test.go` | COMPLIANT |

**Compliance summary**: 17/17 scenarios compliant for the new capability; 58/58 across all seven spec files.

### Substrate and scope pins

| Pin | Result | Evidence |
|---|---|---|
| `cost_events_test.go` byte-unchanged | **HELD** | `git diff --stat origin/main -- cost_events_test.go` empty — verified-by-command |
| `CostFigures` byte-unchanged | **HELD** | struct extracted from both revisions and `diff`ed: identical, 21 lines — verified-by-command |
| `R-LSK-004` bounded release honoured | **HELD** | All 7 diff hunks in `cost_events.go` fall inside the two permitted edits. Every removed line is one of: the 2 released const doc comments, or the 2 payload literals (unavoidable initialisation of the released presence field). No signature changed — verified-by-command |
| `S-LSK-001` amended sequence | **CORRECT** | Promoted `agent-loop-skeleton/spec.md:65` inserts **`cost_turn`** immediately before `turn_end`; the implementing test's `wantKinds` matches — verified-by-command |
| `R-CAN-002` / `R-CAN-005` wind-down order | **CORRECT** | Promoted text states the order with "emit the run-scoped final cost figure" as the third step, immediately before the run-close; `R-CAN-005` explicitly inherits it and re-states the post-shutdown silence as absolute — verified-by-command |
| Scenario-ID renumbering `S-CAN-012`→`013`/`014` | **FULLY CONSISTENT** | `S-CAN-012` remains solely R-CAN-006's AG-14 bite; `S-CAN-013`/`S-CAN-014` each defined once with an ID note; the header explains the collision; and the cross-references *inside* `S-CAN-002` and `S-CAN-005` were also updated. No duplicate `(S|R)-XXX-NNN` definition introduced in any promoted spec — verified-by-command |
| No wildcard in substrate filters | **HELD** | Both filters widened by exact filename suffix only; `cost_events_test.go` and `stream_check_test.go` deliberately absent — verified-by-command (read of both filter tails) |
| Test amendments enumerated (NFR-CST-005) | **HONEST** | `harness_steering_test.go` carries exactly the recorded `EventKindCostTurn` insertion with its written determination; `loop_hook_test.go`'s diff is filter widening + the disclosed pre-existing gofmt reflow + an EOF-newline fix, with **no assertion change** — verified-by-command |

### Promoted specs vs their deltas

Machine-checked: every substantive line (≥25 chars, non-quote) under a `## MODIFIED` heading in each of the seven deltas was searched verbatim in its promoted spec — verified-by-command.

| Capability | Lines checked | Unmatched | Adjudication |
|---|---|---|---|
| agent-cost-events | full-file diff | 2 | Only the header "Becomes … at archive" framing and the Sources line's relative links (which would dangle post-archive). Matches the `agent-cancellation-tree` precedent. **Correct.** |
| agent-cancellation-tree | 34 | 5 | 4 are the deliberate `S-CAN-012/013`→`013/014` renumbering (including its two cross-references); 1 is a delta-authoring instruction. **Correct.** |
| agent-loop-skeleton | 64 | 1 | A delta-authoring instruction ("At archive it MUST read …"); the instruction was in fact executed — the promoted header reads `S-LSK-001` through `S-LSK-026`. **Correct.** |
| agent-protocol-events | 36 | 1 | Authoring instruction only. **Correct.** |
| agent-retry-failover | 13 | 1 | Authoring instruction only. **Correct.** |
| agent-run-driver | 56 | 1 | Authoring instruction only. **Correct.** |
| agent-turn-termination | 8 | 1 | Authoring instruction only. **Correct.** |

The three transcription slips apply reported catching (paraphrased link label, dropped "Two exits emit no cost event at all…" sentence, reverted "this"/"that") are all absent from the promoted text — none survived. I found **one** defect apply did not catch, recorded as WARNING-1 below.

### Doc 0003

Whole-file diff is exactly 3 lines (2 modified, 1 added) — verified-by-command.

| Check | Result |
|---|---|
| Reconciliation note in the AG-06 established form | **CORRECT** — inserted immediately after the Out-of-scope line, opens "**Note — two v2 conflicts, reconciled here (the AG-00 reconcile-or-flag duty, executed):**", and names both reconciliations (sum-over-emitted-events; run-scoped estimate/final axis) |
| Status line | **CORRECT** — "15 of 24"→"**16 of 24**", "AG-12…AG-15"→"AG-12…**AG-16**", plus an AG-16 narrative sentence following the AG-13/14/15 pattern |
| Shared checklist row | **CORRECT AND STILL UNTICKED** — `:2174` `- [ ] Every turn emits a cost event; … — closed by AG-16.1, AG-18.1.` remains unticked because AG-18 has not landed. It is the only checklist row naming AG-16.1 (the other 10 mentions are register/dependency/traceability rows) |

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | YES | Table present in `apply-progress.md` with 9 rows |
| All tasks have tests | YES | 58/58 tasks; 16 new top-level test functions, all present in the tree |
| RED confirmed | YES | All 3 named bites reproduced by verify via `-overlay`, output byte-matching the recorded RED |
| GREEN confirmed | YES | 17/17 new tests pass on an uncached `-race` execution |
| Triangulation adequate | YES | 9-case conversion table; 5 aborted-path subtests; 2 failed-close subtests; estimate/final covered on both the multi-turn and single-turn arms |
| Safety net | YES | Full-package baseline recorded before Phase 1; every non-amended blast-radius site re-run green and confirmed unmodified |

**TDD Compliance**: 6/6 checks passed.

One honest qualification, carried from apply rather than hidden: task 1.4 (`S-CST-003`, aborted paths emit no `cost_turn`) was **vacuously true before GREEN** — no emission existed anywhere at that point. Apply recorded this explicitly rather than claiming a RED. It is non-vacuous now, since S-CST-001's positive proof and bite S-CST-020 both exercise the emission path. Accepted, not flagged.

### Assertion Quality

No tautologies, no orphan empty-collection checks, no ghost loops, no type-only assertions, no smoke tests, no mock-heavy files. Two patterns deserve positive note:

- `TestTurn_CostTurn_AbsenceVsReportedZero` asserts `okAbsent != okZero` on two events whose **counts are both 0** — it cannot pass by reading counts alone, exactly as `R-CST-002`'s evidence-discipline clause demands.
- `TestHarness_CostSession_CumulativeEqualsEmittedCostTurns` compares against a sum derived from the same recorded stream plus a positivity guard, avoiding both the hand-computed-literal trap and the all-absent-coincidence trap.

**Assertion quality**: All assertions verify real behavior. 0 CRITICAL, 0 WARNING.

### Test Layer Distribution

| Layer | Tests | Files |
|---|---|---|
| Unit (internal, `package agent`) | 2 | `cost_usage_test.go` |
| Integration (external `package agent_test`, Turn/Harness-driven) | 14 | `cost_turn_emission_test.go`, `cost_session_test.go` |
| **Total new** | **16** | **3** |

Satisfies `NFR-CST-001`: every behavioral claim is observable from the external test package; the internal test only additionally exercises the pure conversion.

### Issues Found

**CRITICAL**: None.

**WARNING**:

1. **A false amendment claim was introduced into a normative promoted spec.** `openspec/specs/agent-loop-skeleton/spec.md:70` — S-LSK-009's new AG-16 note ends: "*the closed-order assertion at `loop_test.go:1152-1165` gains the kind at that position as an enumerated amendment*". Three facts falsify it, each verified-by-command: (a) `loop_test.go:1152` is inside `TestTurn_ReasoningPassThroughByteExact`, declared at `:1148` — that is S-LSK-005's test; (b) S-LSK-009's real test lives in `loop_tool_dispatch_test.go`, which has an **empty diff vs `origin/main`** — it was never amended; (c) the sentence is **new text added by this change** (confirmed in the promoted-spec diff), so AG-16 authored it rather than inheriting it. The open-items list assumed this citation was "cosmetic-in-archived-docs"; it is not — it leaked into shipped normative text. The note's *substantive* claim (the `cost_turn` lands after the rejoin-ordered tool events and before `turn_end`) is true and tested; only the citation and the "gains the kind … as an enumerated amendment" clause are false. Fix: drop that trailing clause, or repoint it at S-LSK-005.

2. **`R-CST-002`'s "A count MUST NOT be readable without its discriminator" is falsified by the shipped code, and by a file the same requirement pins byte-unchanged.** `CostTurn.Figures()` (`cost_events.go:227`) and `CostSession.Figures()` (`:328`) remain public and return five bare `uint64`s with no discriminator, and `cost_events_test.go:60-67` reads exactly that way from an external test package — a file `R-CST-002` itself requires to stay byte-unchanged and green. The two obligations cannot both hold. The clause propagated to **two** promoted specs: `agent-cost-events/spec.md:69` and `agent-protocol-events/spec.md:60`. No scenario asserts it, which is why nothing failed — it is an unverified normative sentence that archives as false. The real, tested guarantee is the weaker and correct one: a presence-*paired* surface exists and absence is never conflated with a reported nought. Concrete risk: a later milestone reading the clause literally would delete `Figures()` and break a pinned file. Fix: soften to "a presence-paired reading MUST be available for every figure", or explicitly carve out the legacy `Figures()` accessor.

3. **`CostLabel`'s type doc comment now contradicts the two constant doc comments immediately beneath it.** `cost_events.go:85-89` still reads "*distinguishes figures emitted before the stream's final usage update (Estimate) from the figure emitted on the final update (Final)*" — verified byte-unchanged vs `origin/main`. Two lines later, the released const docs state the post-AG-16 truth: `Estimate` is the **run-scoped** running total, and a `cost_turn` is **never** `Estimate`. A reader of the shipped file gets two incompatible definitions of the same enum. Apply's reasoning for leaving it is sound and I endorse it — the `R-LSK-004` release named only `:96-103`, and widening it unilaterally would have exceeded the recorded scope. But the outcome is a self-contradicting source file today. Fix belongs in a future milestone's own recorded release, not here.

**SUGGESTION**:

1. **Bite `S-CST-021`'s recorded mechanism differs from the scenario's stated Given.** The promoted scenario says the scratch tree "*skips the `cost_turn` of any logical turn that was retried*"; `apply-progress.md` task 3.5 records mutating `costAccumulator.add` from `+=` to `=`. I executed **both** and they produce byte-identical failure output, so the scenario archives as **true** — but the record and the spec name different mutations, which is the "a bite can prove the right thing by the wrong mechanism" shape. Align apply-progress's wording with the scenario, or vice versa.

2. **`windDownRun` call-site citation — confirmed cosmetic.** The shipped code has **3** call sites (`harness.go:489`, `:578`, `:604`), not the 2 that `design.md`/`tasks.md` cite. Verified that **no promoted spec mentions `windDownRun` at all**, so the error is confined to archived planning documents and cannot mislead a future reader of normative text. All three were widened; the Go compiler made an undercount unshippable. No action needed.

3. **Pre-existing duplicate scenario IDs in unrelated promoted specs.** A repo-wide sweep found duplicate definitions of `S-PRH-001` (`agent-pre-request-hook/spec.md`) and `S-ATS-098`…`S-ATS-101` (`ai-provider-text-stream/spec.md`). Both files have an **empty diff vs `origin/main`** — not AG-16's doing. Flagged for separate hygiene work.

### Verdict

**PASS WITH FINDINGS** — 0 CRITICAL, 3 WARNING, 3 SUGGESTION.

The implementation is correct, genuinely test-driven, and archive-ready. All three charter scenarios are proven by tests that actually falsify their claims — I reproduced every named bite myself rather than trusting the report, including the one whose recorded mechanism differed from its spec text. The suite is green uncached under `-race`, lint and vulnerability checks are clean, every substrate pin holds, and the scenario-ID collision was resolved consistently across definitions, cross-references and the header.

All three warnings are **spec-and-comment text defects, not behavior defects**. Nothing blocks archive. Findings 1 and 2 are the repository's known spec-staleness class — a false sentence archived in normative text — and warrant a follow-up correction change; finding 3 needs a future milestone's own recorded substrate release.
