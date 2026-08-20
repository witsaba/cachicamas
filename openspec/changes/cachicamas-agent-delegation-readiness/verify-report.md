```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:455ee7aa7b4e64c0c19039852fb82adfe7bd2241eda73cf5d5423cf969ffd2c0
verdict: fail
blockers: 1
critical_findings: 1
requirements: 17/22
scenarios: 30/36
test_command: cd backend/agent && go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:3f15cc740b033b706841f05f08ff0c8dce9b9d88bd77c5ca3ef898806c8db94d
build_command: cd backend/agent && go vet ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

> **This file holds two rounds.** Round 1 (below, verbatim) is the record that
> produced the correction commit `d5a9c364`. Round 2 (at the end) adjudicates
> that correction and is the **current** verdict; the envelope above is Round 2's.

---

# Round 1 — FAIL (6 blockers), against `85344247`


## Verification Report

**Change**: `cachicamas-agent-delegation-readiness` (AG-19, Layer 2 Wave 5)
**Branch**: `feat/agent-layer2-wave5-ag19` @ `85344247`, base `558641f3`
**Mode**: Strict TDD
**Verdict**: **FAIL** — 0 runtime failures, 0 production defects, but 3 scenarios have **no covering assertion** and 3 tests **always pass**. Archive would promote claims no test defends, and two bite scenarios state a failure mechanism I disproved by command.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 32 |
| Tasks complete | 30 |
| Tasks incomplete | 2 (5.2, 5.4 — both `sdd-archive`'s work by design) |

### Build & Tests Execution

**Build**: PASS — `go vet ./...` exit 0, empty output.

**Tests**: PASS — `go test -race -count=1 ./...` exit 0, 12 packages `ok`, **173s wall clock** (18:21:58 → 18:24:51). No `(cached)` marker on any package; `src/agent` itself ran 8.920s. This independently reproduces the orchestrator's result.

**Coverage**: not collected — the change's proof obligations are behavioral scenarios, not line coverage.

### The four bites, re-derived by command (not accepted from the report)

Every bite was reproduced in this phase with `go test -overlay`, against the current tree.

| Bite | Overlay applied | Observed RED | Matches spec? |
|---|---|---|---|
| `S-DEL-021` | gate 2 disabled in `delegation_seam.go:165` | `CheckStream` → `event[4]: value repeats another the collection already carries`, plus 3 refused-kind errors | **Yes**, exactly |
| `S-DEL-022` | `revoke()` body emptied | `panic: send on closed channel` at `delegation_seam.go:127`, raised in the **grandchild goroutine** (`revocation_test.go:217`, created at `:201`), non-zero exit, **no `--- FAIL` line** | **Yes**, exactly |
| `S-DEL-020` | gate 5 disabled in `delegation_seam.go:174` | fails **only** at `cost_test.go:169` (child-identity `cost_turn` present) | **No** — see CRITICAL-4 |
| `S-DEL-023` | child ctx → `context.Background()` | fails at `cancellation_test.go:146`, `drainSink: sink did not close within 1s` | **No** — see CRITICAL-5 |

**`S-DEL-022`'s hazard is the genuine one.** The panic trace proves the send originates in a goroutine `detachedPublishTool.Run` spawns and does not join — outside `runToolWithWindDown`'s inner `recover()`, which absorbs only the synchronous body. The absorbed path was not mistaken for the real one.

### Cost non-fold, measured two-sidedly (hazard #2 and #3)

Instrumented probe under overlay, both directions:

| Tree | Parent reported cumulative | Unfiltered parent-stream `cost_turn` sum | Child's own sum |
|---|---|---|---|
| Shipped | **15** | 15 | **77** |
| Gate 5 removed | **92** | 92 | 77 |

- **Hazard #2 resolved.** The child provably spends non-zero tokens (`cost_test.go:61` — `Input: 77, Output: 33`), and `cost_test.go:127-129` fails hard if it ever reaches zero. `S-DEL-016`'s strict inequality is **not** vacuous.
- **Hazard #3(a) confirmed.** The seam refuses `cost_turn` (`delegation_seam.go:174-176`); `S-CST-024` proves the sentinel and zero sink delivery.
- **Hazard #3(b) confirmed, but by the absence loop only.** 15 vs 92 is observable, so exclusion is real. However the equality assertion at `cost_test.go:157` is **fold-blind**: both sides moved 15→92 together. Only the absence loop at `cost_test.go:164-171` catches the fold. See WARNING-1.

### Revocation ordering and admissibility totality (source-verified)

- **Lock covers the send.** `delegation_seam.go:122-128` — `s.mu.Lock()`, `defer s.mu.Unlock()`, revoked check, then `s.emissions <- emission{ev: ev}` **inside** the critical section. Not a flag check around an unguarded send.
- **LIFO ordering correct.** `scheduler.go:494` registers `defer seam.revoke()` after `defer recoverCall(...)`, so revocation runs first on normal, detach and re-panic exits.
- **Registry gate precedes descriptor gates.** `delegation_seam.go:160-163` calls `eventRegistryEntry` and returns before reading `entry.descriptor`. A zero `Event` (kind 0) hits `event.go:373`'s `int(k) <= 0` guard and is **refused**, never admitted-then-skipped by `CheckStream` (`stream_check.go:112-118`). Totality holds.

### Spec Compliance Matrix (36 scenarios)

| Scenario | Test | Result |
|---|---|---|
| S-DEL-001 | (none) | ❌ UNTESTED |
| S-DEL-002 | `delegation_seam_test.go` > `TestDelegationSeam_UnmintableSurface_ExactlyFourExportedIdentifiers` | ✅ |
| S-DEL-003 | `scheduler_test.go:623` guard (partial) | ⚠️ PARTIAL |
| S-DEL-004 | `TestDelegationSeam_AdmissibilityTable_TotalOverEveryRegisteredKind` | ✅ |
| S-DEL-005 | `TestDelegationSeam_ZeroEvent_RefusedByMembershipGate` | ✅ |
| S-DEL-006 | `TestDelegationSeam_Revocation_NormalPath_...` | ✅ |
| S-DEL-007 | `TestDelegationSeam_Revocation_DetachedPath_...` | ✅ |
| S-DEL-008 | `TestDelegationSeam_Revocation_RePanicPath_...` | ✅ |
| S-DEL-009 | `TestDelegationSeam_TwoRefusals_DistinguishableByErrorsIs` | ✅ |
| S-DEL-010 | `TestNestedRun_ParentStreamCarriesSubagentBracket...` | ✅ |
| S-DEL-011 | `TestWalkableTree_EveryMirroredEventResolvesInOneHop` | ✅ |
| S-DEL-012 | `TestWalkableTree_TwoLanesIndependentlyContiguous` | ✅ |
| S-DEL-013 | `TestSiblings_ConcurrentChildrenNoCrossTalk` | ✅ |
| S-DEL-014 | `TestCancellation_NestedRunCancelsLeafFirst` | ⚠️ PARTIAL |
| S-DEL-015 | (comment only) | ❌ UNTESTED |
| S-DEL-016 | `TestCost_ParentAloneStrictlyLessThanCombined` | ✅ |
| S-DEL-017 | `TestCost_CrossedCostSession_PresentAttributableInert` | ✅ |
| S-DEL-018 | `TestPermissionScope_NarrowsNeverWidens` (7-row table) | ✅ |
| S-DEL-019 | `TestPermissionScope_AskUpDecisionDown` | ✅ |
| S-DEL-020 | bite, reproduced | ⚠️ PARTIAL |
| S-DEL-021 | bite, reproduced | ✅ |
| S-DEL-022 | bite, reproduced | ✅ |
| S-DEL-023 | bite, reproduced | ⚠️ PARTIAL |
| S-DEL-024 | `TestScopeFence_S_DEL_024_ByteUnchangedFilesAndNoNewKind` | ✅ |
| S-DEL-025 | `TestInertPath_ByteIdenticalModulo...` | ⚠️ PARTIAL |
| S-CST-023 | `TestCost_S_CST_023_...` | ✅ |
| S-CST-024 | `TestCost_S_CST_024_FoldUnreachableNotMerelyUnused` | ✅ |
| S-AEV-125 | `TestDelegationSeam_S_AEV_125_...` | ✅ |
| S-CAN-015 | `TestCancellation_S_CAN_015_...` | ✅ |
| S-AGE-028 | `TestSiblings_S_AGE_028_ClosersCountedInARunningSystem` | ✅ |
| S-AGE-029 | `TestSiblings_S_AGE_029_...` | ⚠️ PARTIAL |
| S-TLS-020 | (none) | ❌ UNTESTED |
| S-AGS-062 | `TestV1Scope_S_AGS_062_...` | ✅ |
| S-AGS-063 | `TestV1Scope_S_AGS_063_...` | ⚠️ PARTIAL |
| S-AGS-064 | `TestV1Scope_S_AGS_064_...` | ⚠️ PARTIAL |
| S-LSK-031 | `TestScopeFence_S_LSK_031_SubstrateFiltersByteInSyncExactWidening` | ✅ |

**Compliance summary**: 25 COMPLIANT / 8 PARTIAL / 3 UNTESTED.

### Issues Found

#### CRITICAL

**CRITICAL-1 — `S-DEL-015` has zero executable assertions.** `backend/agent/src/agent/cancellation_test.go:188-195`. The scenario's production-source half ("the sources introduce no cancel function, no cause value, no deadline and no context derivation of their own") is followed by a **comment**, which itself says "grep-verified at apply time". A citation is not evidence. I verified the underlying claim TRUE by command (no `WithCancel`/`WithCancelCause`/`WithTimeout`/`WithDeadline` in `delegation_seam.go`), so this is a coverage gap, not a defect — but the scenario archives as proven while nothing defends it.

**CRITICAL-2 — three tautological assertions.** `backend/agent/src/agent/v1_scope_test.go:104`, `:117`, `:142`. All three are `errors.Is(x, x)` on the same package-level sentinel, which is unconditionally true. `:104`'s subtest is named `admissibility_rule_refuses_and_admits_by_errors_Is` but never publishes an event.

**CRITICAL-3 — an empty subtest that always passes.** `backend/agent/src/agent/v1_scope_test.go:133-137`. `sibling_isolation_has_its_own_dedicated_race_proven_scenario` has a body of comments only — zero assertions.

**CRITICAL-4 — `S-DEL-020`'s stated failure mechanism is false.** `openspec/changes/cachicamas-agent-delegation-readiness/specs/agent-delegation-readiness/spec.md:213` claims the bite fails "with the parent's cumulative inflated by exactly the child's spend **and the strict inequality broken**". Reproduced by overlay: the strict inequality does **not** break (parent-own 92 vs combined 169), and the cumulative-equality assertion does not fire either. The bite fails on one assertion only — `cost_test.go:169`. The cumulative *is* inflated (15→92, measured), but no assertion observes it. Apply recorded this honestly in `apply-progress.md:48`; the spec text was never corrected.

**CRITICAL-5 — `S-DEL-023`'s stated failure mechanism is false.** `.../agent-delegation-readiness/spec.md:192` claims the bite "FAILS on assertion 1 — the child's run error does not match the interrupt sentinel". Reproduced by overlay: it fails earlier, at `cancellation_test.go:146`, via `drainSink`'s 1s timeout. Assertion 1 is never reached. Apply recorded this at `apply-progress.md:46`; the spec text was never corrected.

**CRITICAL-6 — `S-DEL-001` and `S-TLS-020` have no covering test.** `tasks.md:37,45` map both to task **1.4**, a production-wiring task with no test file. `S-DEL-001`'s load-bearing claim — the identities the seam reports "equal the hosting run's and the hosting turn's, **compared against the `run_start` and `turn_start` events observed on that same recorded stream**" — is asserted nowhere; `seam.Parent()` is consumed by fixtures but never compared to the stream's own bracket events. `S-TLS-020`/`S-DEL-003`'s raw-byte claim ("`scheduler.go` contains no type assertion of any kind added by this change") is not asserted either: the `PolicySlot` guard at `scheduler_test.go:623-658` scans only for `policy.(...)` and `.(PolicySlot)` shapes. Both claims are TRUE — I verified the sole type assertion in `scheduler.go:1146` (`cause, _ := r.(error)`) is pre-existing and that this change's diff adds zero — but neither is defended.

#### WARNING

**WARNING-1 — the cost exclusion assertion is fold-blind.** `cost_test.go:123` and `:289` compute the "parent's own" sum with `sumCostTurnInputTokens(got.parentEvents)`, unfiltered by run identity, while `S-DEL-016` says "the sum over the `cost_turn` events observed on the parent's **own** stream". Measured: under a fold both sides move 15→92 in lockstep, so `cost_test.go:157` cannot detect it. Only the absence loop at `:164-171` does. `S-CST-023` filters `cost_session` by `ev.Run() == parentRun` at `:270` but does not apply the same filter to the `cost_turn` sum. Suggested fix: filter `sumCostTurnInputTokens` by parent run identity.

**WARNING-2 — `S-DEL-014` assertion 4 checks the wrong object.** `cancellation_test.go:183-186` tests `errors.As(got.childRunErr, &detachedErr)` — the **child's run error**, already asserted at `:148` to be `ErrInterrupted`, so the check is trivially satisfied. The scenario says "the **parent's tool result** is not a detached-call failure". Corroborating: the `parentToolErr` field is declared at `cancellation_test.go:25`, set to `nil` at `:116`, and **never read anywhere**.

**WARNING-3 — `S-AGE-029`'s source-level half is unasserted and its justification is wrong.** `siblings_test.go:238-243` claims the "no channel type / no buffer capacity" half is proven by `S-DEL-002`'s scan because it catches "ANY top-level declaration". It does not: `collectTopLevelExportedNames` filters on `IsExported()` at `delegation_seam_test.go:434,441,446`, so an **unexported** channel type or buffer constant would pass unnoticed. The test body (`siblings_test.go:244-251`) asserts contiguity only. The underlying claim holds (`delegation_seam.go` contains no `make(chan`).

**WARNING-4 — `S-DEL-025` proves a weaker proposition than stated.** The scenario says "two runs … **one on the merge base and one on this change**". `inert_path_test.go:99-100` runs the fixture **twice on this change**. The test would pass even if this change had altered the inert path, provided it did so deterministically. The in-code justification (`:86-95`) is sound reasoning, but reasoning is not the assertion the scenario promises.

**WARNING-5 — `S-AGS-063` / `S-AGS-064` assert less than their names and comments claim.** `v1_scope_test.go:147-159` (`cost_refusal_makes_the_fold_unreachable`) constructs a `cost_turn` and discards it with `_ = ev`, never publishing. `v1_scope_test.go:181-188` labels its block "Half 1 — implement-now, delivered: all three leaves' own exported entry points exist and are reachable", then asserts the **absence** of a seam on a background context — the opposite proposition.

**WARNING-6 — a stale citation carried into this change's delta.** `.../specs/agent-cancellation-tree/spec.md:50` (`S-CAN-011`) cites `harness.go:200-207` as "the harness's cause-aware routing … unconditional failure path". That range is `takeOrClose`, the Steer-queue terminal-decision step. Cause-aware routing lives at `harness.go:502,565,667,693`. **Provenance**: copied verbatim from the promoted `openspec/specs/agent-cancellation-tree/spec.md:83`; not introduced by AG-19, but it now sits inside this change's own delta and is in scope for task 5.1.

#### SUGGESTION

- **SUGGESTION-1** — pre-existing count discrepancy, **not falsified by this change**: promoted `openspec/specs/agent-permission-protocol/spec.md:7,15` claims "12 requirements → 13 spec scenarios + 4 bites (`S-PPB-001`…`S-PPB-004`)". The file defines 12 requirements ✅ but **18** `S-APP-*` scenarios and **zero** `S-PPB-*` identifiers. AG-19 touches no promoted spec (`git diff 558641f3..HEAD -- openspec/specs/` is empty) and its delta declares zero scenarios, so the assertion is untouched. Flag for task 5.2.
- **SUGGESTION-2** — `S-DEL-018` says "when **Layer 2's exported surface** is enumerated"; `permission_scope_test.go:136` enumerates `delegation_seam.go` only.
- **SUGGESTION-3** — `apply-progress.md:44` records the `S-DEL-022` panic at `delegation_seam.go:126`; the send is at `:127` in the final tree (off-by-one from a mid-implementation recording).
- **SUGGESTION-4** — doc 0003's frozen AG-19 **Acceptance** bullet still reads "child cost aggregated into parent cumulative", the literal reading the spec deliberately reinterprets. The status line and the AG-19 narrative are both consistent with what shipped; only the frozen charter bullet is not.

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|---|---|---|
| R-DEL-001 | ⚠️ Partial | Surface is exactly 4 identifiers, verified by `go/ast` scan; `S-DEL-001` untested |
| R-DEL-002 | ✅ Implemented | 5 gates in the required order; 25-row table cross-checked against `agent.EventKinds()` |
| R-DEL-003 | ✅ Implemented | Send under mutex; LIFO revoke; genuine unrecovered-panic hazard reproduced |
| R-DEL-004 | ✅ Implemented | One-hop walk; constructor surface enumerated by reflection |
| R-DEL-005 | ✅ Implemented | Split-then-validate; two siblings under `-race` |
| R-DEL-006 | ⚠️ Partial | Assertion 4 mis-targeted; `S-DEL-015` unasserted |
| R-DEL-007 | ✅ Implemented | Non-fold measured two-sidedly (15 vs 92) |
| R-DEL-008 | ✅ Implemented | Both directions in a 7-row table; real `WakeParked` round trip |
| R-DEL-009 | ✅ Implemented | Real `git diff` byte-unchanged checks, 25 kinds, signature arities |
| R-DEL-010 | ⚠️ Partial | `S-DEL-025` compares two runs on the same tree |

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| AD-1 — revocation serialized with the send | ✅ | `delegation_seam.go:122-128` |
| AD-2 — registry-membership gate first | ✅ | `delegation_seam.go:160-163` |
| AD-5 — concrete seam unexported | ✅ | Only 4 exported identifiers |
| AD-7 — derived scope is an ordinary composition | ✅ | `derivedScope` lives in `package agent_test` |
| 4-identifier surface | ✅ | Matches `design.md:85,157`; not scope creep |

### Charter fidelity

No subagent tool, configuration or depth limit ships. `delegatingTool`, `derivedScope` and every subagent concept live in `package agent_test`. Production's exported surface is `DelegationSeam`, `DelegationSeamFrom`, `ErrDelegationInadmissible`, `ErrDelegationRevoked` — none names a subagent concept, and `DelegationSeamFrom` on a plain context returns `(nil, false)`. Enforcement is structural.

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD evidence reported | ✅ | Table at `apply-progress.md:37-50` |
| All tasks have tests | ⚠️ | 3 scenarios map to a production-wiring task with no test |
| RED confirmed | ✅ | All 4 bites independently reproduced by `go test -overlay` |
| GREEN confirmed | ✅ | Full suite exit 0, 173s, uncached |
| Triangulation | ✅ | 25-row admissibility table; 7-row narrowing table |
| Safety net | ✅ | Substrate filters widened and pinned byte-in-sync by `S-LSK-031` |

### Assertion Quality

| File | Line | Assertion | Issue | Severity |
|---|---|---|---|---|
| `v1_scope_test.go` | 104 | `errors.Is(ErrDelegationInadmissible, ErrDelegationInadmissible)` | Tautology | CRITICAL |
| `v1_scope_test.go` | 117 | `errors.Is(ErrDelegationRevoked, ErrDelegationRevoked)` | Tautology | CRITICAL |
| `v1_scope_test.go` | 142 | `errors.Is(ErrInterrupted, ErrInterrupted)` | Tautology | CRITICAL |
| `v1_scope_test.go` | 133-137 | (empty subtest) | Zero assertions | CRITICAL |
| `v1_scope_test.go` | 158 | `_ = ev` | Constructs, never publishes | WARNING |
| `cancellation_test.go` | 188-195 | (comment) | Scenario half unasserted | CRITICAL |
| `cancellation_test.go` | 183-186 | `errors.As(childRunErr, &detachedErr)` | Wrong object | WARNING |
| `cost_test.go` | 294 | `parentOwnInput >= parentOwnInput+childInput` | Arithmetically self-referential; real content is `childInput > 0` | WARNING |

**Assertion quality**: 5 CRITICAL, 3 WARNING.

### Quality Metrics

**Linter**: not re-run this phase (apply reports `0 issues`). **Vet**: ✅ clean. **gofmt**: 15 pre-existing baseline files unclean — known, unrelated, excluded.

### Verdict

**FAIL** — the production implementation is correct and I verified its two highest-risk mechanisms by command, but 6 CRITICAL findings block archive: 3 scenarios with no covering assertion, 3 always-passing tests, and 2 bite scenarios whose stated failure mechanism I disproved by overlay. Promoting these specs would archive claims no test defends. Corrections are small and local — add ~5 assertions, correct 2 scenario sentences, delete 4 tautologies.

---

# Round 2 — FAIL (1 blocker), against `d5a9c364`

**Change**: `cachicamas-agent-delegation-readiness` (AG-19, Layer 2 Wave 5)
**Branch**: `feat/agent-layer2-wave5-ag19` @ `d5a9c364`, base `558641f3`
**Mode**: Strict TDD — focused re-verification, scoped to the correction round
**Verdict**: **FAIL** — 5 of 6 round-1 blockers fully closed and WARNING-1 closed
decisively, but `S-DEL-001`'s third clause is defended by an assertion that
**passes whether or not the mechanism works**, which is the same defect class
round 1 blocked on.

**Scope**: this round adjudicates only the 6 closed blockers, WARNING-1's fix,
regression from the correction, the scenario-coverage delta, and stale
restatements. Round 1's verification of the production seam (revocation
ordering, admissibility totality, the `S-DEL-022` grandchild-goroutine hazard,
cost non-fold at 15 vs 92) was not redone and still stands.

### Build & Tests Execution

**Tests**: PASS — `cd backend/agent && go test -race -count=1 ./...` exit 0, 12
packages `ok`, **0 `(cached)` markers**, `src/agent` 8.597s. Independently
re-run in this phase after `d5a9c364`.
**Build**: PASS — `go vet ./...` exit 0, empty output.
**Hygiene**: both worktrees `git status --porcelain` empty after every scratch
edit was reverted; base checkout `/Users/braejan/workspace/witsaba/repositories/cachicamas` untouched.

### Adjudication of the 6 round-1 blockers

| # | Round-1 finding | Fix | Re-derived by | Verdict |
|---|---|---|---|---|
| B1 | `S-DEL-015` had zero executable assertions (comment only) | git-diff source guard scanning added lines for the four context derivations | **scratch edit** — `context.WithTimeout` inserted into `delegation_seam.go`, real file | ✅ **CLOSED** |
| B2 | 3 × `errors.Is(x, x)` tautologies | deleted; replaced with cross-sentinel distinctness checks | source proof of the non-redundancy claim | ✅ **CLOSED** |
| B3 | empty subtest always passes | deleted; coverage delegated to `siblings_test.go` | read `TestSiblings_ConcurrentChildrenNoCrossTalk` | ✅ **CLOSED** |
| B4 | `S-DEL-020`'s stated failure mechanism false | scenario sentence rewritten | **overlay** — gate 5 removed, instrumented probe | ✅ **CLOSED** (one imprecise clause → WARNING-7) |
| B5 | `S-DEL-023`'s stated failure mechanism false | scenario sentence rewritten | **overlay** — `childCtxOverride` → `context.Background()` | ✅ **CLOSED** |
| B6 | `S-DEL-001` / `S-TLS-020` had no covering test | two new tests added | **overlay + scratch edit**, 4 independent defeats | ❌ **PARTIALLY CLOSED** — see CRITICAL-7 |

#### B1 — the overlay-unsoundness reasoning is correct, and the guard is real

Apply claimed `go test -overlay` cannot prove this guard RED because the guard
shells out to `git diff` through `os/exec`. **Verified, in both directions.**

`gitDiff` (`backend/agent/src/agent/loop_test.go:793-804`) builds
`exec.Command("git", "diff", ref, "--", paths...)` with `cmd.Dir = root`. The
subprocess reads the on-disk working tree; the Go build overlay never reaches it.

1. **Overlay applied** — `context.WithTimeout` compiled into `delegation_seam.go`
   via `-overlay`: `ok github.com/cachicamas/backend/agent/src/agent 0.536s`.
   An overlay proof here would have been a **false GREEN**. Apply was right to
   refuse it.
2. **Real scratch edit** — same four lines written to the real file:

```
--- FAIL: TestCancellation_NestedRunCancelsLeafFirst (0.05s)
    cancellation_test.go:217: the production diff for backend/agent/src/agent/delegation_seam.go
    introduces "context.WithTimeout" — S-DEL-015 requires the child's context derivation to live
    entirely in package agent_test, not in production sources
```

Reverted; `git status --porcelain` empty. The guard genuinely fails when a
context derivation is introduced. **B1 closed.**

#### B2 — the non-redundancy claim checks out

Apply claimed the three replacements are "non-redundant under Go's `errors.Is`
symmetry for plain sentinels". Verified rather than accepted:

- `ErrDelegationInadmissible` and `ErrDelegationRevoked` are `errors.New` values
  (`delegation_seam.go:35,41`); `ErrInterrupted` likewise (`cancellation.go:39`).
- `grep 'func (.*) Is(' *.go` (production) returns **nothing** — no custom `Is`
  method, no `Unwrap`. So `errors.Is(a, b)` reduces to `a == b` and is symmetric.
- The three checks therefore cover the three **distinct unordered pairs** over
  three sentinels — `{Inadmissible, Revoked}`, `{Revoked, Interrupted}`,
  `{Interrupted, Inadmissible}` — with no pair repeated. Non-redundant, as claimed.
- Each is falsifiable: it fails if any two sentinels are ever aliased. Weak, but
  no longer unconditionally true. **B2 closed.**

#### B3 — the replacement coverage genuinely exists

`TestSiblings_ConcurrentChildrenNoCrossTalk` (`siblings_test.go`) asserts sibling
isolation for real: two distinct non-empty child run identities; `CheckStream`
on all three streams; exactly 2 `subagent_started` / 2 `subagent_ended`; both
children resolving to the **same** parent run; every parent event's `Run()`
confined to `{parentRun, runA, runB}`; and each child's own captured stream
carrying **only** its own run identity. Runs under `-race` in the green suite.
The deleted subtest had a comments-only body, so nothing was lost. **B3 closed.**

#### B4 / WARNING-1 — measured two-sidedly, before and after the fold

Single instrumented probe, `parentOwn` / `childInput` / `combined` printed in
both trees:

| Tree | `parentOwn` (filtered) | `childInput` | `combined` | parent terminal `cost_session` | Fires |
|---|---|---|---|---|---|
| Shipped (GREEN) | 15 | 77 | 92 | **15** | — (PASS) |
| Gate 5 removed (bite) | 15 | 77 | 92 | **92** | `cost_test.go:168` **and** `:179` |

```
--- FAIL: TestCost_S_CST_023_MirroredFigureDoesNotDisturbOwnProtocol
    cost_test.go:302: parent's own final cost_session input tokens = (92, present=true), want (15, true)
--- FAIL: TestCost_ParentAloneStrictlyLessThanCombined
    cost_test.go:168: parent's terminal cost_session input tokens = 92, want 15 (the sum over the parent's own cost_turn events)
    cost_test.go:179: parent event[7] is a cost_turn carrying the CHILD's run identity — R-DEL-002 gate 5 must have refused this
```

**WARNING-1 is closed decisively.** Round 1's finding was that both sides of the
equality moved 15→92 in lockstep, so `cost_test.go` certified the very defect it
was supposed to catch and only the absence loop dissented. With the run-identity
filter, the filtered reconstruction is **pinned at 15** while the harness's
unfiltered fold inflates the reported figure to 92, so the equality assertion
now fires — in **two** tests (`S-DEL-016` and `S-CST-023`). The single most
important check of this round passes.

**`S-DEL-020`'s new sentence, clause by clause:**

| Clause | Measured | Verdict |
|---|---|---|
| "terminal `cost_session` (92) no longer equals the run-filtered reconstruction (15)" | exactly `cost_test.go:168` | ✅ true |
| "a `cost_turn` carrying the child's run identity is found on the parent's stream" | exactly `cost_test.go:179` | ✅ true |
| "the parent's cumulative **is** inflated by exactly the child's spend" | 15→92, Δ = 77 = `childInput` | ✅ true |
| "the **strict inequality itself does not break**" | 15 < 92 in both trees; `:141` never fires | ✅ true |
| "the filtered parent-own figure and the parent-plus-child combined figure **move together** (15 vs 92)" | **neither figure moves**; both are constant at 15 and 92 across the two trees | ⚠️ **WARNING-7** |

The load-bearing failure mode is now correct and exact. The one residual clause
is an explanatory leftover from round 1's description of the *pre-fix*
unfiltered behavior — see WARNING-7. **B4 closed.**

#### B5 — `S-DEL-023`'s new sentence is exactly right

Overlay is sound here (the bite lives entirely in `package agent_test`, via the
existing `childCtxOverride` hook). Replacing `buildCancellationFixture(t, nil)`
with a `context.Background()` override:

```
--- FAIL: TestCancellation_NestedRunCancelsLeafFirst (1.00s)
    cancellation_test.go:148: drainSink: sink did not close within 1s (6 event(s) received so far)
    — the loop is not closing the sink it owns
```

Every clause of the new sentence holds: it **fails before assertion 1 is ever
reached** (the failure is raised inside `buildCancellationFixture`, before the
`errors.Is(got.childRunErr, ...)` line); the child's `agenttest.Hold(gate)` step
is documented as "released by ctx cancellation, never by the test"
(`cancellation_test.go:60`), so the release never arrives; the run never
completes; and `drainSink`'s own 1-second timeout fires with the exact quoted
message. **B5 closed.**

#### B6 — `S-TLS-020` closed; `S-DEL-001` only two thirds closed

`TestScopeFence_S_TLS_020_SeamRidesBesidePolicySlotNotInsideIt` — **three
independent defeats, all RED:**

1. Scratch-added type assertion in `scheduler.go` → both the type-assertion scan
   **and** the bare-seam-type scan fire (`scope_fence_test.go:276`, `:279`).
2. Overlay mutating the injected `PolicySlot` → `scope_fence_test.go:298`
   ("received PolicySlot `call-tls-020-mutated`, want byte-identical to the
   injected `call-tls-020`").
3. Self-defending against vacuity: `if diff == "" { t.Fatal(...) }` at
   `scope_fence_test.go:255` — the guard refuses to pass on an empty diff. ✅

`TestDelegationSeam_S_DEL_001_IdentitiesMatchRunStartAndTurnStartOnSameStream`
— **two of three clauses RED-provable, one not:**

- **Run/turn identity clauses** — overlay making `Parent()` return a fabricated
  turn: `delegation_seam_test.go:370` fires. ✅
- **"an event published through it is observed by the consumer on the parent's
  stream"** — overlay making `Publish` silently discard the event instead of
  sending it to the funnel: the test still reports **PASS**. ❌ See CRITICAL-7.

### Issues Found

#### CRITICAL

**CRITICAL-7 — `S-DEL-001`'s publish clause is defended by a ghost assertion.**
`backend/agent/src/agent/delegation_seam_test.go:380-390`. The loop searches the
drained parent stream for *any* event satisfying
`ev.Kind() == agent.EventKindMessageStartText && ev.Run() == capturedRun`. The
parent harness's **own** turn-2 text response already emits exactly such an
event, so the predicate is satisfied regardless of whether the seam's publish
ever reached the funnel. Proven by instrumented overlay:

| Tree | `message_start_text` with parent run identity | `found` | Result |
|---|---|---|---|
| Shipped | **2** (harness's own + the seam's) | true | PASS |
| `Publish` silently discards | **1** (harness's own only) | true | **PASS** |

The count moves 2→1 — the mechanism demonstrably broke — and the assertion does
not notice. This is the same defect class as round-1 CRITICAL-2/CRITICAL-3 (an
assertion that reports PASS unconditionally with respect to the mechanism it
names), in the very scenario round-1 CRITICAL-6 raised. `S-DEL-001`'s third
clause therefore still archives as proven with nothing defending it.

*The underlying behavior is correct* — the shipped tree really does put the
published event on the parent's stream (count 2 vs 1). This is a coverage
defect, not a production defect. **The fix is ~3 lines**: the test already mints
a fresh message ID (`mustFreshMessageID(t)`) for the published event but never
captures or compares it; capture it and match on that exact ID (or assert the
count), and the assertion becomes falsifiable.

#### WARNING (new this round)

**WARNING-7 — `S-DEL-020`'s "move together" clause is measurably inaccurate.**
`specs/agent-delegation-readiness/spec.md:213`. The sentence reads "the filtered
parent-own figure and the parent-plus-child combined figure move together (15 vs
92)". Measured: across the GREEN tree and the gate-5-removed bite, `parentOwn`
is **15 in both** and `combined` is **92 in both** — neither figure moves at all.
"Move together" correctly described the *pre-fix* unfiltered code (round 1's own
finding), and survived into the corrected sentence. The clause's conclusion
("so parent-own remains strictly less than combined") is true and verified; only
its stated reason is not. A reader would take away the opposite of the actual
lesson, which is that the run filter is precisely what pins parent-own at 15 so
the equality can catch the fold. Not a blocker — the scenario's failure mode,
which is what a reader reproduces, is now exact — but it should be corrected in
the same pass as CRITICAL-7.

**WARNING-8 — a stale justification left behind in the file the correction
edited.** `backend/agent/src/agent/cancellation_test.go:140-144`. The doc comment
above `TestCancellation_NestedRunCancelsLeafFirst` still says the `S-DEL-015`
production-source half is "asserted below via a **direct positive statement
rather than re-derived per run**". The correction did exactly the opposite — it
replaced the positive statement with a real per-run `git diff` derivation. The
same comment cites "the **3-line** `scheduler.go` diff"; the actual diff against
`558641f3` is 7 insertions / 1 deletion. Comments only, no assertion affected,
but this is the project's recurring "a correction leaves its own justification
wrong" shape.

**WARNING-9 — `S-DEL-015`'s guard has no anti-vacuity floor, unlike
`S-TLS-020`'s.** `cancellation_test.go:209-220` scans the diff for forbidden
symbols but never asserts the diff is non-empty. `TestScopeFence_S_TLS_020`
guards the identical hazard at `scope_fence_test.go:255`
(`if diff == "" { t.Fatal(...) }`). Both resolve their base with
`git merge-base HEAD origin/main`, so once AG-19 lands on `main` the base
becomes the change itself and the diff goes empty — at which point the
`S-DEL-015` guard passes vacuously forever while `S-TLS-020`'s fails loudly.
The `merge-base` pattern is repo-wide and pre-existing (`loop_test.go:1320`,
`loop_hook_test.go:822`, `compaction_stream_test.go:118`,
`context_strategy_test.go:1180`, `cost_turn_emission_test.go:575`, and AG-19's
own `S-DEL-024` at `scope_fence_test.go:49`), so this is not an AG-19
regression — but the asymmetry between the two guards this correction shipped
is new, and one line closes it.

#### SUGGESTION (new this round)

- **SUGGESTION-5** — `tasks.md:115` cites `cost_test.go:167` for the `S-DEL-020`
  equality failure; Go reports it at `:168` (the `t.Errorf` line, not the `if`).
  Off-by-one.
- **SUGGESTION-6** — `tasks.md:76`'s task-3.3 lead clause still reads "FAILS on
  assertion 1", self-corrected two sentences later by "**Evidence differs from
  the literal prediction**". Historically accurate but reads as a live claim.
- **SUGGESTION-7** — `S-TLS-020`'s middle sub-clause ("the `PolicySlot` source
  guard passes with **its own source byte-unchanged**") has no named assertion:
  `scheduler_test.go` is not in `del024ByteUnchangedFiles()`. Verified true by
  command here (`git diff 558641f3 -- scheduler_test.go` is empty), but
  undefended.

#### Carried forward from round 1 — open by agreement, not re-adjudicated

Explicitly out of the correction round's scope: **WARNING-2** (`S-DEL-014`
assertion 4 checks the child's run error, not the parent's tool result),
**WARNING-3** (`S-AGE-029`'s source-level half unasserted, justification wrong),
**WARNING-4** (`S-DEL-025` compares two runs on the same tree),
**WARNING-5** (`S-AGS-063`/`S-AGS-064` assert less than their names claim),
**WARNING-6** (stale `harness.go:200-207` citation in the cancellation-tree
delta), and **SUGGESTION-1..4**. None is a blocker.

### Regression check — did the correction weaken anything?

The correction touched 5 test files and 1 spec file. The only change to an
existing assertion's semantics is `sumCostTurnInputTokens` gaining a run filter.

| Check | Result |
|---|---|
| GREEN-tree cost figures unchanged from round 1 | ✅ `parentOwn=15`, `childInput=77`, `combined=92` — identical to round 1's measurement |
| `childInput == 0` fatal still non-vacuous | ✅ 77 |
| `S-DEL-016` strict inequality still asserted | ✅ `cost_test.go:140-142`, unchanged |
| `S-CST-023` cost-session equality | ✅ strengthened — now fires at `:302` under the bite |
| Assertions deleted | 4 (3 tautologies + 1 empty subtest) — all provably unfalsifiable; **zero** falsifiable assertions removed |
| Scenario coverage lost | none |
| Suite | ✅ exit 0, 12 packages, uncached, `-race` |

**No regression.** Every deletion removed something that could never fail; every
addition is falsifiable except the one named in CRITICAL-7.

### Scenario coverage delta (36 scenarios)

| Scenario | Round 1 | Round 2 | Why |
|---|---|---|---|
| `S-DEL-001` | ❌ UNTESTED | ⚠️ **PARTIAL** | identity clauses RED-proved; publish clause is a ghost assertion (CRITICAL-7) |
| `S-DEL-003` | ⚠️ PARTIAL | ✅ **COMPLIANT** | raw-byte "no type assertion added" now asserted by `S-TLS-020`'s diff scan, RED-proved |
| `S-DEL-015` | ❌ UNTESTED | ✅ **COMPLIANT** | git-diff guard, RED-proved by scratch edit |
| `S-DEL-020` | ⚠️ PARTIAL | ✅ **COMPLIANT** | sentence re-derived by overlay; failure mode exact (WARNING-7 on one clause) |
| `S-DEL-023` | ⚠️ PARTIAL | ✅ **COMPLIANT** | sentence re-derived by overlay; every clause verified |
| `S-TLS-020` | ❌ UNTESTED | ✅ **COMPLIANT** | both halves RED-proved, 3 independent defeats |
| `S-DEL-014` | ⚠️ PARTIAL | ⚠️ PARTIAL | WARNING-2, carried forward by agreement |
| `S-DEL-025` | ⚠️ PARTIAL | ⚠️ PARTIAL | WARNING-4, carried forward |
| `S-AGE-029` | ⚠️ PARTIAL | ⚠️ PARTIAL | WARNING-3, carried forward |
| `S-AGS-063` | ⚠️ PARTIAL | ⚠️ PARTIAL | WARNING-5, carried forward |
| `S-AGS-064` | ⚠️ PARTIAL | ⚠️ PARTIAL | WARNING-5, carried forward |
| all other 25 | ✅ | ✅ | unchanged, verified in round 1 |

**Compliance summary**: **30 COMPLIANT / 6 PARTIAL / 0 UNTESTED** (round 1:
25 / 8 / 3). Every one of the 36 now maps to a named test. Five of the six
remaining PARTIALs are round-1 warnings open by agreement; the sixth
(`S-DEL-001`) is CRITICAL-7.

**Requirements**: **17/22** (round 1: 16/22). `agent-tool-scheduler`'s
requirement closes with `S-TLS-020`. `R-DEL-001` (CRITICAL-7), `R-DEL-006`
(WARNING-2), `R-DEL-010` (WARNING-4) and the two `agent-v1-scope` requirements
carrying WARNING-3/WARNING-5 remain partial.

### Stale-restatement grep

Searched every `*.md` and `*.go` in the repo for the old, corrected claims:

| Old claim | Occurrences | Assessment |
|---|---|---|
| "strict inequality broken" | `tasks.md:115` only | ✅ quoted as history ("claimed …"), not restated |
| "FAILS on assertion 1" | `apply-progress.md:46`, `tasks.md:76`, `tasks.md:116` | ✅ all three immediately self-correct; SUGGESTION-6 on the lead clause |
| "grep-verified at apply time" | none | ✅ removed |
| "positive statement rather than re-derived per run" | `cancellation_test.go:144` | ❌ **WARNING-8** — now false |
| "3-line scheduler.go diff" | `cancellation_test.go:141` | ❌ **WARNING-8** — actual diff is 7+/1- |
| deleted subtest name | `v1_scope_test.go:137`, `tasks.md:114` | ✅ both are deletion records |

No delta spec restates a corrected claim. The two live restatements are Go
comments in `cancellation_test.go`, not spec text.

### Assertion Quality (round 2 delta)

| File | Line | Assertion | Issue | Severity |
|---|---|---|---|---|
| `delegation_seam_test.go` | 380-390 | `Kind()==message_start_text && Run()==capturedRun` search | Ghost match — parent harness emits the same shape; passes when `Publish` is defeated | **CRITICAL** |
| `v1_scope_test.go` | 104, 117, 142 | `errors.Is(x, x)` | **RESOLVED** — deleted, replaced with falsifiable distinctness checks | — |
| `v1_scope_test.go` | 133-137 | empty subtest | **RESOLVED** — deleted | — |
| `cancellation_test.go` | 188-195 | comment only | **RESOLVED** — real git-diff guard, RED-proved | — |
| `cost_test.go` | 123, 289 | unfiltered `sumCostTurnInputTokens` | **RESOLVED** — run-identity filter, equality now fires 92 vs 15 | — |

**Assertion quality**: 1 CRITICAL (down from 5), 0 new WARNINGs on assertions.

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD evidence reported | ✅ | `apply-progress.md` correction table + `tasks.md:110-118` blocker-closure table |
| RED claims independently re-derived | ✅ | 7 independent defeats run this phase (2 scratch edits, 5 overlays) |
| Overlay-soundness reasoning validated | ✅ | overlay demonstrably cannot defeat the `os/exec` git guard — apply's refusal to use one was correct |
| GREEN confirmed | ✅ | full suite exit 0, uncached, `-race` |
| Every corrected assertion falsifiable | ⚠️ | 6 of 7 proved RED; `S-DEL-001`'s publish clause could not be made to fail (CRITICAL-7) |

### Verdict

**FAIL — 1 blocker, 1 CRITICAL finding, 17/22 requirements, 30/36 scenarios.**

The correction round did real work and I could not fault most of it: B1's git
guard genuinely fails on a real scratch edit and apply's reasoning about overlay
unsoundness is correct; B2's replacements are non-redundant and falsifiable for
the reason apply gave; B3's delegated coverage genuinely exists; B4 and B5's
rewritten sentences were re-derived by command and match the observed mechanism;
and WARNING-1's fix is closed decisively — the equality that previously certified
the defect now fires at 92 vs 15 in two tests. Round-1 blockers went 6 → 0,
scenarios 25 → 30 of 36, UNTESTED 3 → 0, always-passing assertions 4 → 0.

One blocker remains, and it sits inside the fix for B6. `S-DEL-001`'s third
clause — that an event published through the seam is observed on the parent's
stream — is checked by a search that the parent harness's own
`message_start_text` already satisfies. I defeated `Publish` so it discarded the
event entirely; the count on the stream fell from 2 to 1 and the test still
passed. That is precisely the class of always-passing assertion round 1 blocked
on, in the scenario round 1 named. It is a ~3-line fix (the test already mints
the message ID it needs to match on), and WARNING-7/WARNING-8/WARNING-9 are one
sentence and two comments — all four belong in a single scoped correction.

**Next**: `sdd-apply` for one scoped correction, then a third focused verify
limited to `S-DEL-001` and the three new warnings. Archive must not proceed
until CRITICAL-7 is closed.
