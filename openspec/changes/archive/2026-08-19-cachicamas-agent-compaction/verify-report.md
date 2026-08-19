```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:86cbe80e3871488c6dcf2c2f1a9afae5c1c97233516c217c43b7806740cf51cf
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 14/14
scenarios: 36/36
test_command: cd backend/agent && go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:0c9c40a3813a7faeb918e41c527ebb8282db985ff6956d2c52881efbcfffe2ca
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report

**Change**: `cachicamas-agent-compaction` (AG-18, Layer 2 Wave 4)
**Version**: new capability `agent-compaction`, `R-CMP-001`…`R-CMP-014`
**Mode**: Strict TDD
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ag18` @ `a9fc9866`, branch `feat/agent-layer2-wave4-ag18`

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 89 |
| Tasks complete | 89 |
| Tasks incomplete | 0 |

### Build & Tests Execution

**Build**: PASSED — `make build` → `go build -trimpath ./...`, exit 0.
**Lint**: PASSED — `make lint` (`go vet` + `bin/golangci-lint run`) → `0 issues.` after `cache clean`.
**Vuln**: PASSED — `make vuln-check` exit 0.
**gofmt**: 15 files listed, **intersection with AG-18's 20 changed Go files is EMPTY** (independently re-verified). All 15 pre-exist on `main`.
**Tests**: PASSED — 12/12 packages `ok`, exit 0, **173.05s wall**, `openaicompat` 172.308s, **zero `(cached)` markers**.

```text
cd backend/agent && go test -race -count=1 ./...
ok  .../agent 9.327s      ok  .../agenttest 2.849s        ok  .../agenttest/sweep 1.497s
ok  .../tracetest 2.297s  ok  .../ai 4.704s               ok  .../ai/internal/retry 2.802s
ok  .../openaicompat 172.308s                             ok  .../conformancetest 3.379s
ok  .../openrouter 3.574s ok  .../openrouter/conformance 6.770s
ok  .../openrouter/internal/smoke 3.081s                  ok  .../handoff 2.231s
real 173.05
```

### Charter Gherkin Coverage — all 8 scenarios (`0003:1683-1756`)

| # | Leaf | Charter scenario | Covering test (opened and read) | Result |
|---|---|---|---|---|
| 1 | AG-18.1 | own model call with own spend | `TestCompaction_OwnCall_DistinctProvider`, `TestCompaction_SpendFoldedIntoCumulative`, `TestCompaction_CancellationAbortsWithoutEndingRun` | COMPLIANT |
| 2 | AG-18.1 | instruction injected, never authored | `TestCompaction_InjectionOnly` — asserts captured system instruction is exactly one segment equal to the injected string, and every request message is element-equal to the pre-compaction prefix | COMPLIANT |
| 3 | AG-18.2 | replaced span never splits a pair | `TestResolveCut_RetractsToMarkBoundary`, `TestResolveCut_SeededValidation` | COMPLIANT |
| 4 | AG-18.2 | protected turns untouched, summary typed | `TestCompaction_ProtectionValueIdentical`, `TestCompaction_IdentityRenumbers`, `TestCompaction_SummaryTypedByOriginOnly` | COMPLIANT |
| 5 | AG-18.3 | stream records what compaction did | `TestCompaction_StreamAcceptedUnmodified`, `TestCompaction_ReconstructionNamesReplacedTurns` | COMPLIANT |
| 6 | AG-18.4 | atomic-or-absent | `TestCompaction_AtomicOrAbsent`, `TestCompaction_WindDownNeverEntered` | COMPLIANT |
| 7 | AG-18.5 | demanded and triggered are one path | `TestCompaction_OneDemandSharedMechanics` | COMPLIANT |
| 8 | AG-18.5 | mid-turn demands refused typed | `TestHarness_Compact_TypedRefusalMidTurn` | COMPLIANT |

Each test was opened and its assertions read. No "proved as a corollary" record was accepted on trust.

### Spec Compliance Matrix

**Requirements: 14/14** — every `R-CMP-001`…`R-CMP-014` has at least one named, existing, passing covering test.

**Scenarios: 36/36 have a passing covering test at runtime — of which 34 are COMPLIANT and 2 are PARTIAL. 0 UNTESTED, 0 FAILING.**

> The envelope reports `36/36` because every scenario has runtime evidence from a passing covering test. **Do not read that as "fully asserted":** `S-CMP-008` and `S-CMP-028` are graded **PARTIAL** — their tests pass but assert less than the scenario text claims. See WARNING 5 and WARNING 6.

> **The allocated total is 36, not 37.** `S-CMP-009` is **never defined** anywhere in the spec — the header declares the allocated *range* `S-CMP-001`…`S-CMP-037` but only 36 scenario bullets exist. Confirmed in both the archived delta and the promoted spec.

| Scenario | Test | Result |
|---|---|---|
| S-CMP-001/002/003/005 | `compaction_call_test.go` | COMPLIANT |
| S-CMP-004/006 | `TestCompaction_SpendFoldedIntoCumulative`, `TestCompaction_IffHoldsOnAllThreeArms` | COMPLIANT |
| S-CMP-007/027 | `TestHistory_ReplacePrefix_SucceedsAndRenumbers`, `..._RejectsToolBearingSummary` | COMPLIANT |
| **S-CMP-008** | `TestHistory_NoBypass_EveryMutatingRouteRejectsOrphaningSequence` + `driveReplacePrefixOrphaningSequence` | **PARTIAL** |
| S-CMP-010/011/012/013 | `compaction_surgery_test.go` `TestResolveCut_*` | COMPLIANT |
| S-CMP-014/015/016 | `TestCompaction_ProtectionValueIdentical` / `_IdentityRenumbers` / `_SummaryTypedByOriginOnly` | COMPLIANT |
| S-CMP-017 | `TestCompaction_StreamAcceptedUnmodified` | COMPLIANT |
| S-CMP-018/019 | `TestCompaction_ReconstructionNamesReplacedTurns`, `_FinishedXorFailed` | COMPLIANT |
| S-CMP-020/021 | `TestCompaction_AtomicOrAbsent`, `_WindDownNeverEntered` | COMPLIANT |
| S-CMP-022/026 | `TestCompaction_OneDemandSharedMechanics`, `_AtMostOnePerBoundary` | COMPLIANT |
| S-CMP-023/024/025 | `compaction_demand_test.go` | COMPLIANT |
| **S-CMP-028** | pre-existing `TestHistoryRouteGuard_SurfaceMatchesExpectedTable` (`method:Entry` rows read-only) | **PARTIAL** |
| S-CMP-029/034/035 | `TestCompaction_MarkedSpanFromTwoRuns`, `_UnmarkedPrefixFailsTyped`, `TestHistory_MarkedCloseNotExported` | COMPLIANT |
| S-CMP-030/031/032/033 | bites — all four RED re-proved by this phase (below) | COMPLIANT |
| S-CMP-036 | `TestCompaction_SubstrateByteUnchanged` (real `git diff` against merge-base) | COMPLIANT |
| S-CMP-037 | `TestCompaction_InertUnlessRequested` | COMPLIANT |

### Bites — RED independently re-proved, not taken from the transcript

All five re-proved with `go test -overlay`, mutations held **outside** the worktree at `…/scratchpad/bites/`. Worktree verified clean afterward; base checkout untouched at `main@f6acc0d2`.

| Bite | Mutation applied | Observed RED | Predicted by spec? |
|---|---|---|---|
| `S-CMP-030` | `resolveCut` returns the naive cut directly | `resolveCut(hist,2) = 2, want <= 1 … the boundary must not split the call/result pair` (+2 more) | YES — "FAILS reporting a split pair across the boundary" |
| `S-CMP-031` | commit moved to **before** the provider call returns | `turn 2's request carries 1 message(s), want 3 (the uncompacted transcript, unchanged by the failed compaction)` | YES — "FAILS reporting a mutated transcript after a failed compaction" |
| `S-CMP-032` | `total.add(ct)` removed, **`cost_turn` event still emitted** | `cost_session input tokens = 0, want 111 (the compaction's own usage, folded)` | YES — fails on the **cumulative figure short**, not on a missing event. Strongest form. |
| `S-CMP-033` | on-demand door enqueues for the next boundary instead of refusing | `Compact returned nil, want a typed refusal` | YES |
| `S-HIS-102` | `ReplacePrefix` row dropped from `expectedHistoryRoutes`, route still exported | `surface route "ReplacePrefix" … is not in expectedHistoryRoutes` | YES — apply recorded this as a *natural* regression rather than a staged bite; the synthetic bite reproduces the identical failure, so the substitution is sound. |

### Design Coherence — the decisions the brief singled out

| Decision | Followed? | Evidence |
|---|---|---|
| **AD-5** explicit cost fold | YES | `compaction.go:315-320` accumulates against `total` at the emission site. Independently defeated: with the fold removed but the event retained, `S-CMP-004` still fails **on the cumulative figure** (0 vs 111). The fold is a written statement, not an inherited one. |
| **AD-8** protection semantics | YES | `compaction_surgery_test.go:295-300` compares `Message().Equal` **and** `Origin()` **positionally** (`pre[cut+i]` vs `post[1+i]`). Fixture replaced prefix = **2** (≥2 satisfied). `pre.ID() != post.ID()` is genuinely asserted at `:324` and is non-vacuous — IDs shift 3→2 and 4→3. `reflect.DeepEqual` appears only inside a comment declaring it forbidden; the file does not import `reflect`. |
| **AD-11** atomicity by ordering | YES | Defeat-tested by walking the path, not by trusting intent. Every statement before the single commit at `compaction.go:331` is read-only over `hist`: `resolveCut`→`transcriptFromHistory`+`markBoundaryAtOrBefore` (read-only), `markSpan` (read-only), `buildLoopRequest` (pure), `Provider.Stream`, `drainCompactionCall`. `commitReplacePrefixOp` runs **all** validation before **any** field assignment and allocates a fresh `newEntries`, so a rejected replacement leaves `entries`/`open`/`marks` unchanged. No journal, snapshot or undo log exists. Bite `S-CMP-031` proves the assertion detects the loss. |
| **AD-9** turn marks | YES | `commitCloseTurnMarkedOp` delegates to `commitCloseTurnOp`'s rule 4, so a mark can only be recorded with an empty open set — the structural guarantee `resolveCut` relies on. Preserved across `ReplacePrefix` (surviving marks shift by `count-1`; the shifted prefix stays pairing-closed). |
| **AD-12 / NFR-CMP-004** substrate | YES | `TestCompaction_SubstrateByteUnchanged` runs a real `git diff` against the merge-base for 13 named files, `src/ai/`, and `go.mod`/`go.sum`. Independently confirmed. |
| `agenttest`→`agent` import cycle | RESOLVED | `go vet ./...` exit 0; `agenttest` contains no import of `src/agent`. |

### Relaxation Ledger — every relaxed MUST has a replacement AND a test

| Relaxed | Replacement invariant | Enforced by |
|---|---|---|
| `R-HIS-001` "MUST NOT support removal" | exactly one prefix-shaped removal, mark-aligned, pairing-closed, through the single commit primitive | `checkReplacePrefixCount` + `replacementOpenSet`; `TestHistory_ReplacePrefix_SucceedsAndRenumbers`, `_NothingElseRemovable`, `_RejectsToolBearingSummary`, `driveReplacePrefixOrphaningSequence` |
| `R-HIS-005` identity stability | scoped to a transcript **generation**; identities stay ordinal-derived and caller-unmintable | `TestCompaction_IdentityRenumbers` asserts both the change AND the exact new ordinal sequence |
| `R-CTX-003` verdict unconstructible | never-compact moves to the **zero value** (`Compaction *CompactionRequest` nil ⇒ nothing) | `TestCompaction_InertUnlessRequested` (S-CMP-037) — byte-identical stream and read-back vs. no strategy |
| `L2C-07` two clauses | route enumeration re-closes at **four** naming prefix replacement; identity stability becomes generation-scoped | `doc.go:35` and `doc_contract_guard_test.go:76` are **byte-in-sync** (guard asserts equality at `:183`); both replacement clauses separately asserted at `:187` and `:190` |
| run-driver "no new exported `History` method" row | breach confined to one prefix-shaped route, no mid-span variant, guard updated in the same commit | `S-HIS-102` bite (RED re-proved above) |
| `R-CST-001` per-path iff | nothing relaxed; one additive row | `TestCompaction_IffHoldsOnAllThreeArms` |

**No relaxation was found with nothing in its place.**

### `S-HIS-080` row count

Correctly **re-scoped off the count** ("The claim is deliberately scoped off the literal row count"), not merely renumbered. The pre-existing-drift labelling is intact and explicitly reads *"Correction of a PRE-EXISTING defect, not an AG-18 regression"*, naming AG-14 as the appender and AG-17 as having correctly declined to repair. Not misreadable as an AG-18 regression.

### Archive Integrity — compared by blob hash, not by eye

- 7 SDD artifacts + 8 spec files present under `openspec/changes/archive/2026-08-19-cachicamas-agent-compaction/`.
- `openspec/changes/` contains only `archive/`.
- Promoted `openspec/specs/agent-compaction/spec.md` vs the archived delta: **332 lines each**, `diff` shows **only the 3-line promotion header transform**. No truncation, no placeholder body.
- 7 promoted specs modified + 1 added = the 8 the `82e0b5ef` diff touches.
- `tasks.md`: 89 ticked / 0 unticked.
- Both acceptance-checklist rows AG-18 closes (`0003:2174`, `0003:2176`) are ticked at `a9fc9866`.

### Issues Found

**CRITICAL**: None.

**WARNING**

1. **`context_strategy.go:25` carries a stale citation that now resolves to AG-18's own inserted code.**
   `backend/agent/src/agent/context_strategy.go:25` reads *"see the R-RTY-002 argument at `harness.go:550-561`"*. On `main` that range **was** the `reused BY REFERENCE across attempts` argument; in this tree it is AG-18's compaction wiring block (`if verdict.Compaction != nil { … }`). The real argument moved to **`harness.go:601-612`**.
   This is the exact AG-17 defect (`3946638c`) recurring, and it is aggravated three ways: (a) `context_strategy.go` **is** a file this change edited; (b) the `harness.go` diff explicitly identifies this same citation as fragile and replaces its own copy with a grep instruction, citing *"the AG-16/17 line-shift lesson"* — the lesson was applied at one site and missed at the sibling site pointing at the **same** target; (c) the stale target resolves to plausible code, so a reader following it lands on compaction wiring while looking for the argument against per-attempt consultation.

2. **`openspec/specs/agent-run-driver/spec.md` cites `history.go:327-332` as "re-verified in this worktree" — and it does not resolve there.**
   The AG-18-authored text claims the prefix replacement is *"dispatched from the same single validating commit primitive (`history.go:327-332`, re-verified in this worktree)"*. Lines 327-332 in this tree are mid-doc-comment prose of `ReplacePrefix`; the function is at `history.go:343` and the dispatch is in `commit` at `history.go:434-449`. No Go file changed after `82e0b5ef`, so the citation was wrong when written. This asserts a verification that did not happen.

3. **Two pre-existing AG-17 citations in `openspec/specs/agent-run-driver/spec.md` were silently shifted by AG-18's insertion and not re-annotated.**
   `harness.go:550-561` (R-RTY-002 argument) and `harness.go:562` (the attempt loop) both now resolve to AG-18's compaction wiring. The attempt loop is now `harness.go:613`. (`harness.go:512` survives — it sits above the insertion.)

4. **AG-18-authored citations in the promoted `openspec/specs/agent-cost-events/spec.md:59` are stale against the tree a reader resolves them in.**
   `harness.go:563` / `:567-576` / `:590` were correct against `main@f6acc0d2` but now point at unrelated lines. Unlike `agent-compaction/spec.md` — whose header honestly declares its `origin/main@f6acc0d2` baseline (verified: 7/7 sampled citations resolve there) — `agent-cost-events` carries no baseline note and says *"re-verified in this worktree"*.

5. **`S-CMP-008` is PARTIAL.** The scenario claims the prefix-replacement route *"drives the same orphaning rejection every other route drives"*. `driveReplacePrefixOrphaningSequence` cannot construct one — a turn mark is unreachable from `package agent_test` — and its own comment says so, substituting a "count names no recorded mark" rejection instead. The claim as written is not asserted by any test.

6. **`S-CMP-028` is PARTIAL.** No AG-18 test names it. Coverage leans on the pre-existing, unmodified `TestHistoryRouteGuard_SurfaceMatchesExpectedTable`. The reasoning (no exported signature accepts an `EntryOrigin`; `Entry`'s method set is unchanged) is sound but is inferred, not asserted.

7. **The header's allocated scenario range overstates by one.** `S-CMP-009` is never defined. The spec's own "no total is stated" discipline protects against count drift but not against a **gap** inside a declared range.

**SUGGESTION**

1. **`R-CMP-004` step 3 is unreachable dead code with no covering test.** `resolveCut`'s straddle branch (`compaction.go:66-77`) can never execute: `markBoundaryAtOrBefore` only ever returns a recorded mark's count, a mark is only committable with an empty open set, and `ReplacePrefix` preserves that property for shifted marks — so `earliestOpenMessageIndex` over a mark-aligned prefix always reports "not open". `S-CMP-013`'s nested/interleaved-pairs half therefore passes **through mark retraction, not through the straddle retraction the requirement describes**. The property is true; the named mechanism is untested.
2. **A new count assertion was added to the very table a count assertion was just removed from.** `TestCompaction_SubstrateByteUnchanged` asserts `len(expectedLayer2ContractRows) != 8` while `S-HIS-080` was deliberately re-scoped off that exact count as "this repository's count-assertion drift class". It will go false when AG-19+ appends `L2C-09`.
3. **`TestCompaction_AtomicOrAbsent` proves the transcript by proxy.** Its doc comment claims "the history read-back after the attempt is byte-identical to the read-back captured before it", but it compares turn 2's **captured request** against `preEntries`. Equivalent (the request is built from history) and arguably stronger, but the comment describes an assertion the test does not literally make.
4. **`dcdc350c`'s message misdescribes where two bites' evidence lands** (both are in that same commit). Assessed as cosmetic; not amended, per the repo's no-amend rule.
5. **`82e0b5ef`'s prose names six specs while its diff touches seven.** Content claim verified independently: the omitted `agent-protocol-events` back-annotation is substantive and **correct** — its three recorded facts (dedicated turn bracket, `compaction_events.go` byte-unchanged, exactly one of finished/failed per operation) each hold against this tree. Enumeration omission only.

### Verdict

**PASS WITH WARNINGS** — 89/89 tasks complete; 14/14 requirements and 36/36 scenarios covered by runtime evidence (2 of them only PARTIALLY asserted) by named, existing, passing tests; all five bites RED-reproved independently; build, lint, vuln-check and a 173s uncached race suite all green. No blocker and no CRITICAL. Seven warnings, four of which are citation-integrity defects — one in production source (`context_strategy.go:25`) that recurs the exact defect AG-17 had to repair, and one (`history.go:327-332`) that asserts a re-verification that does not resolve.
