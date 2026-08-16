```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:57b2d6ebd0a947b5d2c6da9ca1f7c3b05670a281ce3f7cbee15128886fa5dd90
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 13/13
scenarios: 22/22
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:e5c57b4701d6083aa35a19992a3345d0bae61ec74cd1dd99082839df597dde85
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

# Verification Report: AG-11 — Complete turn termination and typed failure reporting

**Change**: `cachicamas-agent-turn-termination` · Layer 2 Wave 2, milestone 11/24
**Branch**: `feat/agent-layer2-wave2-ag11` @ `44f2bf18` · base `main` `b8eb7d75`
**Mode**: Strict TDD · **Store**: hybrid (Engram + OpenSpec filesystem)
**Verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 3 WARNING, 2 SUGGESTION

## Method

Every apply-phase claim was treated as input, not evidence. All four gates were
re-run in this worktree. Every behavioral test was subjected to an independent
**defeat test** — the production code (or its upstream vocabulary) was broken
under `go test -overlay`, and the test was required to fail. `-overlay` never
writes to the tree; the single edit that could not be reached by overlay (the
AST-based signature guard reads `loop.go` from disk at runtime) was made
directly and reverted, with `git status` confirmed clean afterwards.

## Completeness

| Check | Result | Details |
|---|---|---|
| tasks.md items | ⚠️ 50/51 | Only 8.3 (openspec archive move) unchecked — deliberate, correctly reasoned deferral |
| Requirements implemented | ✅ 13/13 | `R-ATT-001..009` (ADDED) + `R-LSK-001`, `R-LSK-004`, `R-APP-012`, `R-AEV-008` (MODIFIED) |
| Scenarios covered | ✅ 22/22 | 12 `S-ATT` + 5 `S-TTB` bites + 5 new cross-cut (`S-LSK-011/012`, `S-APP-014`, `S-AEV-074/075`) |
| Spec promotion | ⚠️ content faithful | Text faithful; heading form and one relative link diverge (W1, W2) |

## Gate Evidence (re-run by verify, not quoted from apply)

| Gate | Command | Exit | Evidence |
|---|---|---|---|
| Tests | `cd backend/agent && make test` | **0** | 12 `ok` packages, 0 `FAIL`, `-race` |
| Lint | `cd backend/agent && make lint` | **0** | `0 issues.` — `go vet` + `golangci-lint` |
| Build | `cd backend/agent && make build` | **0** | `go build -trimpath ./...` |
| Vuln | `cd backend/agent && make vuln-check` | **0** | 0 `"finding"` objects, go1.26.6 |

Lint reported zero findings, so the repo's known stale-lint-cache failure mode
(which produces false *positives*) is not in play; `golangci-lint` resolves to
`bin/golangci-lint`, freshly installed in this worktree.

## Defeat-Test Matrix — proof every pin is non-vacuous

14 defeat tests were constructed; **all 14 produced the required failure**.

| # | Test under proof | Injected defect | Result |
|---|---|---|---|
| 1 | `TestTurn_ExhaustivenessPin` | `outcomeForFinish`: `Refusal` → `Finished` | ✅ FAIL `dispatch/refusal` |
| 2 | `TestTurn_ExhaustivenessPin` | `outcomeForFinish` returns a constant | ✅ FAIL 6/7 dispatch subtests |
| 3 | `TestTurn_ExhaustivenessPin` | **8th `ai.FinishReason` appended** upstream | ✅ FAIL — names the unhandled value |
| 4 | `TestTurn_FinishReasonDispatch_...` | test table aliases `Refusal` → `Finished` | ✅ FAIL + duplicate-outcome detector |
| 5 | `TestTurn_RefusalPauseFinished` | `PauseTurn` → `TurnOutcomeRefused` | ✅ FAIL pairwise-distinct |
| 6 | `TestTurn_FatalPath_EmitsTypedBrackets` | fatal branch restored to pre-AG-11 shape | ✅ FAIL |
| 7 | `TestTurn_FatalPath_...` (identity pin) | two independent `NewFailure` wraps | ✅ FAIL — pointer identity, not value equality |
| 8 | `TestTurn_PartialContentSurvives` | typed arm returns `ai.Message{}` | ✅ FAIL |
| 9 | `TestTurn_NoCompletionPath` | D2 zero-finish normalization removed | ✅ FAIL |
| 10 | `TestTurn_ExactlyOneProviderCall` | second `provider.Stream` on fatal path | ✅ FAIL both subtests |
| 11 | `TestTurn_InternalErrorArm_EmitsNothing` | internal arm emits a `run_end` | ✅ FAIL |
| 12 | `TestTurn_SignatureUnchanged` | 7th **variadic** param on `Turn` (real edit, reverted) | ✅ FAIL — catches a backward-compatible widening |
| 13 | `TestFailure_PartialOutput_...` + `TestTurn_TypedFailureFullyInspectable` | `PartialOutput()` returns constant `false` | ✅ FAIL both |
| 14 | `TestTurnOutcome_DistinctMemberPerFinishReason` | two `TurnOutcome` `String()` forms collapsed | ✅ FAIL |

### The two findings the orchestrator flagged for independent proof

**(1) The shipped exhaustiveness pin is NOT vacuous.** Apply self-reported that
its first draft cross-checked only its own table. The **shipped** version adds a
behavioral half that dispatches all seven members through a real `agent.Turn`
call. Defeat tests 1 and 2 prove it: breaking `outcomeForFinish` fails the pin.
The vacuous draft would have survived both.

**(2) The charter's "future finish reason fails the suite" clause IS proven.**
An eighth `ai.FinishReason` was simulated by inserting a constant before
`finishReasonLimit` in `src/ai/finish_reason.go` under `-overlay` (no Layer 1
edit was written to the tree). The agent-layer pin failed with:

```
turn_termination_test.go:289: ai.FinishReason(8) (invalid) validates but is absent
    from dispatchVocabulary — an eighth finish reason the dispatch does not handle
turn_termination_test.go:293: ai.FinishReason.Validate() admitted 8 candidate(s),
    dispatchVocabulary names 7 — the pin is not closed in both directions
```

This is a full demonstration, not a partial one: the pin uses `Validate()` as the
membership oracle rather than the unexported `finishReasonLimit`, so it detects an
upstream vocabulary change from an external test package.

## Scenario → Test Coverage (22/22)

| Scenario | Covering test | Runtime | Defeat-proven |
|---|---|---|---|
| S-ATT-001 | `TestTurnOutcome_DistinctMemberPerFinishReason` | ✅ PASS | ✅ #14 |
| S-ATT-002 | `TestTurnOutcome_ZeroAndFailureRuleUnchanged` | ✅ PASS | regression pin on unchanged `validate` |
| S-ATT-003 | `TestTurn_FinishReasonDispatch_EachProducesDistinctOutcome` | ✅ PASS | ✅ #4 |
| S-ATT-004 | `TestTurn_ExhaustivenessPin` | ✅ PASS | ✅ #1 #2 #3 |
| S-ATT-005 | `TestTurn_RefusalPauseFinished` | ✅ PASS | ✅ #5 |
| S-ATT-006 | `TestTurn_PauseReplaysVerbatim` | ✅ PASS | ✅ via #5 dispatch |
| S-ATT-007 | `TestTurn_FatalPath_EmitsTypedBrackets` | ✅ PASS | ✅ #6 #7 |
| S-ATT-008 | `TestFailure_PartialOutput_ReachableAsTypedValue` | ✅ PASS | ✅ #13 |
| S-ATT-009 | `TestTurn_PartialContentSurvives` | ✅ PASS | ✅ #8 |
| S-ATT-010 | `TestTurn_ExactlyOneProviderCall` | ✅ PASS | ✅ #10 |
| S-ATT-011 | `TestTurn_SubstrateUntouched` + merge-base diff | ✅ PASS | verified independently below |
| S-ATT-012 | `TestTurn_NoCompletionPath` | ✅ PASS | ✅ #9 |
| S-TTB-001 | bite on `TestTurn_FinishReasonDispatch_...` | ✅ reproduced | ✅ #4 |
| S-TTB-002 | bite on `TestTurn_ExhaustivenessPin` | ✅ reproduced | ✅ #3 |
| S-TTB-003 | bite on `TestTurn_RefusalPauseFinished` | ✅ reproduced | ✅ #5 |
| S-TTB-004 | bite on `TestTurn_FatalPath_EmitsTypedBrackets` | ✅ reproduced | ✅ #6 |
| S-TTB-005 | bite on `TestTurn_PartialContentSurvives` | ✅ reproduced | ✅ #8 |
| S-LSK-011 | `TestTurn_SignatureUnchanged` | ✅ PASS | ✅ #12 |
| S-LSK-012 | `TestTurn_SubstrateUntouched` + merge-base diff | ✅ PASS | verified independently |
| S-APP-014 | `failure.go` scoped diff + filter-identity check | ✅ verified | verified independently |
| S-AEV-074 | `TestFailure_PartialOutput_ReachableAsTypedValue` | ✅ PASS | ✅ #13 |
| S-AEV-075 | `TestTurn_TypedFailureFullyInspectable` | ✅ PASS | ✅ #13 |

Every scenario maps to an executing test or an independently re-derived
structural check. **No scenario is uncovered, and no covering test was found
that cannot fail.**

## RED-Proof Audit (apply-progress claims re-derived)

All five bites were independently reproduced. Three matched apply's captured
output **verbatim, including line numbers**. Two (S-TTB-004, S-TTB-005) showed a
uniform **+4 line offset** — apply recorded `turn_failure_test.go:90` / `:149`
where HEAD now reports `:94` / `:153`.

**This is not a false claim.** The offset is fully explained: commit `bd27581a`
(Phase 6, landing after Phase 5's `11ea09b4`) added four import lines
(`go/ast`, `go/parser`, `go/printer`, `go/token`) to `turn_failure_test.go`.
Verified by reading `git show 11ea09b4:...turn_failure_test.go` at lines 90 and
149 — both carry the exact asserted strings. The RED evidence was genuine and
captured against the tree state of its own moment.

The three "N/A — already GREEN" entries (`TestTurn_ExactlyOneProviderCall`,
`TestTurn_InternalErrorArm_EmitsNothing`, `TestTurn_NoCompletionPath`) and the
one approval-testing entry (`TestTurn_SignatureUnchanged`) were the highest
vacuity risk, since none carries a RED proof. **All four were defeat-tested
(#9, #10, #11, #12) and all four bite.** Apply's honest labeling of these as
regression/approval pins rather than fabricating RED claims is correct conduct.

## Binding Constraint Verification (design D1–D8)

| Constraint | Result | Evidence |
|---|---|---|
| `loop.go`'s `_ = sched.Schedule(...)` byte-unchanged (D8) | ✅ | Absent from `git diff b8eb7d75 -- loop.go`; still at `loop.go:265` |
| `turn_events.go` confined to outcome members + `String()` | ✅ | Diff contains only 6 members + 6 `String()` cases |
| `TurnEnd.validate` failure-iff-aborted rule unchanged | ✅ | `validate` absent from the diff entirely |
| `NewTurnEnd` signature unchanged | ✅ | `NewTurnEnd` absent from the diff entirely |
| `failure.go` confined to `PartialOutput()` | ✅ | Only addition; `NewFailure`/`Category`/`Delivery`/`Retryable`/`Unwrap` absent from diff |
| Both filters carry an identical entry set | ✅ | 18 entries each, `diff` empty |
| Exact-filename suffixes only, no widening | ✅ | `HasSuffix` only; the sole `HasPrefix` parses `diff --git`, not paths |
| `turnEnd.Failure() == runEnd.Failure()` by pointer | ✅ | `if turnFailure != runFailure` on `*agent.Failure`; defeat #7 proves it |
| Milestone doc: only line 2167 flipped | ✅ | Diff shows exactly one line; 2162 and 2168 remain `[ ]` |
| Only released non-test files differ | ✅ | `failure.go`, `loop.go`, `turn_events.go` only |
| `go.mod` / `go.sum` byte-unchanged | ✅ | Empty diff |
| 25-kind guard still passes | ✅ | `TestEventKindRegistration_...` green |

## Spec Promotion Verification (diffed, not eyeballed)

**ADDED capability** — `openspec/specs/agent-turn-termination/spec.md` vs its
change-folder delta: `diff` shows **exactly one removed line**, the
promotion-note header. Content is otherwise byte-identical. Correct.

**Three MODIFIED deltas** — each named requirement block was replaced in full
with the delta's MODIFIED text; the `(Previously: ...)` clauses, the new
scenarios (`S-LSK-011`, `S-LSK-012`, `S-APP-014`, `S-AEV-074`, `S-AEV-075`) and
the relative cross-link rewrite (`../agent-turn-termination/spec.md`, correctly
re-resolved for the canonical location) all landed. No un-back-annotated merge
and no silent divergence. Content is faithful.

Two presentational defects are recorded as W1 and W2 below.

## Issues

### CRITICAL — none

No blocker was found. Nothing prevents archive.

### WARNING

**W1 — Spec-promotion heading transform was not applied (4 headings, 3 files).**
The repo's canonical form is `### R-XXX-NNN — <Title>`; delta files use OpenSpec's
`### Requirement: <Title>` parse form. AG-09 and AG-10 both authored
`### Requirement: Loop surface: single-turn function form (D1)` in their deltas,
and the canonical file at `b8eb7d75` still correctly read
`### R-LSK-001 — Loop surface: single-turn function form (D1)` — the transform was
applied by those milestones. AG-11 promoted the delta form instead:

- `openspec/specs/agent-loop-skeleton/spec.md:23` and `:58`
- `openspec/specs/agent-permission-protocol/spec.md:116`
- `openspec/specs/agent-event-envelope/spec.md:126`

Each file now mixes both heading styles. AG-11's own new capability spec correctly
uses `### R-ATT-001 — ...`, so the divergence is confined to the three MODIFIED
files. Content is unaffected; this is a consistency defect that degrades
grep-by-requirement-ID lookups. Suggested fix: rewrite the four headings to
`### R-LSK-001 — Loop surface: single-turn function form (D1)`,
`### R-LSK-004 — Substrate untouched, with AG-11's recorded exact-filename release`,
`### R-APP-012 — Substrate preservation, 7th consecutive milestone (D8, NFR-TLS-003 carry)`,
`### R-AEV-008 — Invariant pin 4: a failure payload is a typed value, never a message string`.

**W2 — Broken relative doc-0003 link in the canonical AG-11 spec.**
`openspec/specs/agent-turn-termination/spec.md:3` uses `../../../../docs/...`
(4 levels) where the canonical location resolves at 3. Proven by path resolution:

```
ls openspec/specs/agent-turn-termination/../../../../docs/.../0003-...md  → No such file
ls openspec/specs/agent-turn-termination/../../../docs/.../0003-...md     → resolves
```

All nine sibling canonical `agent-*` specs use three levels. This is the only
canonical spec in the repository with a broken doc-0003 link. Apply recorded the
deviation knowingly (Deviations item 4), reasoning that `tasks.md` 7.5 bound the
promotion to "byte-identical minus the header note". That reasoning is defensible
for the apply phase, but the outcome is a broken link in a permanent canonical
document, which is worth one line to fix.

Secondary, lower priority: the change-folder deltas for `agent-turn-termination`
and `agent-loop-skeleton` also carry 4 levels where their (deeper) location needs
5 — broken in the delta too, and it will move again on archive.

**W3 — Task 8.3 (openspec archive move) is unchecked.**
Assessed as **correctly deferred**, not an omission. AG-10's precedent
(`verify-report.md` referenced at the non-archived path across two remediation
rounds; archive move `c4830cd7` landing as the final pre-merge chore commit)
establishes that archiving follows verification. Moving the folder mid-apply
would have stranded the path this very report is written to. The task is
annotated in `tasks.md:117` rather than silently skipped. It is a
lifecycle/cleanup task, so per the verify decision gate it is a WARNING, not a
CRITICAL. It belongs to `sdd-archive`.

### SUGGESTION

**S1 — Review-budget overage is justified work, not scope creep.**
Code-only total is **1134 lines** across the eight files `design.md` forecast
(≈940), i.e. +20.6% over forecast and +134 over the session-extended 1000-line
budget. The distribution shows why this is not bloat:

- Production code: **173 lines** (`loop.go` 116, `turn_events.go` 45,
  `failure.go` 12) against a 177-line forecast — **under** budget.
- Tests: **961 lines**, entirely the overage.

The extra test volume is three pins the design named but did not separately
line-budget: the failure-identity pointer pin, the D1 internal-error-arm
regression pin, and the AST signature guard — all three of which this
verification defeat-tested and found load-bearing (#7, #11, #12). Removing any
of them would weaken the milestone. Recommend accepting under the pre-authorized
`size:exception`.

**S2 — `apply-progress.md` states the overage against two different baselines.**
Deviations item 5 says "≈20.6% over the design estimate"; Workload/PR Boundary
says "134 lines (≈13.4%) over". Both are arithmetically correct (940 vs 1000
baselines) but reading them together is confusing. Cosmetic.

## Strict TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | ✅ | Full "TDD Cycle Evidence" table present in `apply-progress.md` |
| All tasks have tests | ✅ | 14 test functions across 3 files |
| RED confirmed | ✅ | 8 genuine RED claims; 5 bites independently reproduced |
| GREEN confirmed | ✅ | 14/14 pass under `-race -count=1` |
| Never-RED tests justified | ✅ | 4 labeled honestly as regression/approval pins; all 4 defeat-tested |
| Triangulation | ✅ | Dispatch walks all 7 members; `ExactlyOneProviderCall` has 2 subtests |
| Safety net | ✅ | Substrate guards ran before and after the filter widening |

**TDD Compliance: 7/7.** Apply's disclosure of its own vacuous first draft
(Risks item 1) — caught internally and corrected before any RED was claimed — is
exemplary conduct and was independently confirmed to have been fixed in the
shipped code.

### Assertion Quality

✅ **All assertions verify real behavior.** No tautology, no orphan empty-collection
check, no type-only assertion standing alone, no ghost loop, no smoke test. The
loop in `TestTurn_InternalErrorArm_EmitsNothing` iterates drained events and could
in principle be empty, but the test asserts message content, finish value and error
type outside that loop, so it cannot pass vacuously. Every scenario drives a real
`agent.Turn` call against a scripted `agenttest.Provider`. No assertion reads a
failure message string, satisfying `S-AEV-073`'s carry-forward.

### Test Layer Distribution

| Layer | Tests | Files |
|---|---|---|
| Unit (vocabulary/accessor pins) | 4 | 2 |
| Integration (scripted provider through real `Turn`) | 10 | 2 |
| **Total** | **14** | **3** |

## Design Coherence (D1–D8)

All eight decisions implemented as designed. Two documented sequencing
deviations, both pre-authorized:

1. **D6's `reconstructMessage()` extraction pulled into Phase 2's commit.**
   Risk 1 in both `tasks.md` and `design.md` explicitly anticipated and
   authorized this as task sequencing, not design change. ✅ Acceptable.
2. **Work Units 3/4/5 consolidated into `c1224a7b`.** Justified: a single
   exhaustive switch satisfies all three, and splitting would produce
   intermediate commits red for the other phases' tests. ✅ Acceptable.

No design deviation breaks a spec.

## Verdict

**PASS WITH WARNINGS.**

The implementation is correct, complete and — unusually well — *proven*. Every
behavioral pin survived an independent defeat test; the exhaustiveness pin's
charter obligation is demonstrated rather than asserted; every binding constraint
holds; the spec promotion is content-faithful. The three warnings are all
documentation-surface issues (two presentational, one a correctly-deferred
lifecycle task). None blocks archive.

**Recommended before merge** (optional, one-line each): fix W2's link depth and
W1's four headings. Both are safe documentation edits with no test impact.

**Next**: `sdd-archive` — which also owns task 8.3.
