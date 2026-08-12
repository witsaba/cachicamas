# Apply progress — `cachicamas-agent-event-envelope` (AG-04)

> Executor: sdd-apply. Artifact store: hybrid (Engram + OpenSpec). Strict TDD mode active. Scope: Phase 1 through Phase 5 (Phase 6, spec promotion, deferred to archive per convention).

## Status

**45/45 tasks in `tasks.md`'s Phases 1–5 marked `[x]`** (45/47 of the whole file — Phase 6's 2 tasks, spec promotion, are correctly left `[ ]`: that phase is archive-scoped, per this file's own scope line above and AG-03's identical precedent). All 45 unique `S-AEV-001`…`S-AEV-102` scenario ids (plus 6 restated `S-AGP-010`…`S-AGP-015` ids from the `agent-package-scaffold` delta — 51 spec scenario ids total across the change) have recorded evidence. `make test` is green (all 12 packages, zero FAIL, zero DATA RACE, pre- and post-change). `make lint` scoped to `./src/agent/...` is 0 issues; the full-module `make lint` exits 1 solely because of one pre-existing, unrelated finding outside AG-04's charter (see Deviations #4). AG-03's three boundary guards pass with zero changes to their own logic.

**Post-verify status**: four scoped corrections applied after `sdd-verify` — see "Post-verify corrections" section below. `make test -race` remains green after all four.

**Real changed-line count** (`git diff --numstat origin/main`): **2687** lines in `backend/agent/src/agent` (2681 insertions + 6 deletions), **921** more in `openspec/` planning docs and the milestone-doc status line, **3608 total**. This exceeds the tasks.md forecast's own upper bound (2200, code-only) and the session's pre-authorized 1000-line `size:exception` ceiling — flagged explicitly below (Deviations #5), not silently rounded down, mirroring AG-03's own precedent (`archive-report.md`: "5 lines over the pre-authorized 1000-line ceiling").

## Deviations from design.md (flagged, not silently applied)

Two structural sequencing findings (both resolved the same way tasks.md already resolved `stream_check.go`'s own two-commit split), one resolved design gap, one caught-and-fixed test defect, one lint-convention fix, one pre-existing unrelated finding, one closing completeness pass, and the size overage.

### 0. Closing completeness pass: 3 spec scenarios with no explicit coverage

Before declaring done, cross-checked every `S-AEV-0NN` id in `specs/agent-event-envelope/spec.md` against every id referenced in `backend/agent/src/agent/*.go`. Found 3 gaps: `S-AEV-084` (the every-kind-constructible guard's own "recorded scope note" — R-AEV-009 requires the guard's source to state it proves construction-time exhaustiveness only, closing neither invariant 3 nor the loop-level typed-error path), `S-AEV-100` and `S-AEV-101` (independently-labeled tests for the ordering-invariants and membership-criterion doc statements — both were *substantively* covered already by `S-AEV-102`'s bite test, since the spec's own literal wording is a substring of the same paragraphs, but neither had its own independently-verifiable, explicitly-labeled scenario as the spec's format requires). Fixed: added the scope note to `event_registry_test.go`'s header comment plus `TestEventRegistryDoc_StatesTheGuardsRecordedScope`; added `TestPackageDoc_StatesOrderingInvariants` and `TestPackageDoc_StatesMembershipCriterion` to `invariant_pin_test.go`. All three RED-proved by scratch-removal, then reverted. Final cross-check: every one of the 45 unique `S-AEV-0NN` ids in spec.md is now referenced in the implementation — confirmed by `diff` of two `grep -oE` extractions.

### 1. `RunStart`'s payload and constructors moved from AG-04.2 to AG-04.1

Design's File Changes table assigns `run_events.go` (including `RunStart`) to node AG-04.2. But AG-04.1's own scenarios (S-AEV-001 kind derivation, S-AEV-004 identity reads, S-AEV-020/021/022 parent-before-delegation) need **at least one constructible event kind** to test against the public surface — task 1.10 explicitly needs `NewDelegatedRunStart`. Design's own Testing Strategy table independently confirms this: `envelope_test.go` (AG-04.1) is documented as covering "parent present on delegated, absent on top-level."

Resolved by creating `run_events.go`'s `RunStart` payload, `NewRunStart`, and `NewDelegatedRunStart` in Phase 1 — the identical shape of reconciliation tasks.md already documents and pre-authorizes for `stream_check.go` (contiguity core at AG-04.1, scope engine at AG-04.2). `RunEnd`/`RunOutcome` (needing the run bracket's closer) landed at AG-04.2 as originally planned. Recorded in tasks.md task 1.3 and this file, not silently applied.

### 2. `failure.go` (design AD-2) moved from AG-04.3 to AG-04.2

`RunEnd`/`TurnEnd`'s own constructor signatures (`f *Failure`) require the `Failure` type to exist to compile — but `failure.go` is tasks.md's task 3.1, assigned to Phase 3, one phase after `RunEnd`/`TurnEnd` (task 2.1/2.2). Since design AD-2 already fully specifies `Failure`'s shape, `failure.go` was created complete in Phase 2, ahead of its node attribution. Its own dedicated scenarios (S-AEV-070/071/072/073) stayed at Phase 3 exactly as planned — only the type's *existence* moved earlier. Recorded in tasks.md task 2.1 and this file.

### 3. `TurnEnd`'s failure-required-iff-Aborted rule — a resolved design gap

Design AD-2 states RunEnd's failure rule explicitly ("`f` required iff `RunOutcomeFailed`, forbidden otherwise") but states no equivalent rule for `TurnEnd`. Resolved by mirroring RunEnd's rule exactly (failure required iff `TurnOutcomeAborted`, forbidden otherwise) for internal consistency. Recorded in `turn_events.go`'s own file comment. A later milestone needing an unmodelled abort (interruption with no captured failure) amends this rule in its own change, per the proposal's forward-fix preference.

### 4. Pre-existing, unrelated `make lint` finding

`src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go:17` (`var-naming: don't use an underscore in package name`, `package openrouter_conformance`) fails golangci-lint v2.9.0. Verified: **zero diff** against `origin/main` for that file, and the finding **reproduces in complete isolation** (`golangci-lint run ./src/ai/openaicompat/openrouter/conformance/...` alone, no `src/agent` involvement). Last touched 2026-08-10, predating both AG-03 and this change. AG-03's own `archive-report.md`/`apply-progress.md` recorded "0 issues" at its close — the discrepancy is unexplained (possibly a golangci-lint caching artifact) but is not this change's concern to resolve: `src/ai/openaicompat/**` is Layer 1's vendor-adapter subtree, explicitly frozen and out of AG-04's charter (proposal Affected areas: "Layer 1 — Unchanged... Frozen surface"). `./src/agent/...` scoped alone is 0 issues. Reported honestly rather than claimed clean or silently patched.

### 5. Review-budget overage against the forecast

tasks.md's own Review Workload Forecast estimated 1400–2200 changed lines for `backend/agent` code. The actual figure is 2687 (code) / 3608 (code + openspec planning docs) — roughly 1.2x over the forecast's own upper bound and 2.7x over the session's pre-authorized 1000-line `size:exception` ceiling. Driven by 45 `S-AEV` spec scenarios across 11 charter Gherkin scenarios (plus 6 restated `S-AGP` scenarios from the `agent-package-scaffold` delta, 51 spec scenario ids total), each RED-first, plus the first real production Go this layer has shipped (envelope, per-lane ordering, run/turn lifecycle, typed-failure wrap, a two-level scope-engine validator, a witness-table guard). `delivery_strategy: exception-ok` was pre-authorized for exactly this shape of change; the overage is flagged here for visibility, not hidden.

### 6. Caught-and-fixed defect: a self-referencing audit test

`invariant_pin_test.go`'s S-AEV-073 audit test (no test asserts on failure message-string content) originally spelled its own scan target, `"provider failure:"`, as a literal contiguous string **inside its own doc comment** — making the test flag its own source the first time it ran (`invariant_pin_test.go contains the literal ai.Failure.Error() rendering prefix...`). Found via a real failing run, not a code-review guess. Fixed by assembling the needle from two concatenated string literals (`"provider" + " failure:"`) so neither the comment's prose nor the code's own bytes contain the contiguous sequence being scanned for — documented in the test's own comment, mirroring `ambient_authority_test.go`'s "never name this file specifically" posture. Re-verified GREEN, then re-verified the scan still bites on a genuine planted violation (S-AEV-073's own bite, below).

### 7. `package-comments` lint fix — a missed Layer 1 convention

All 7 new production files (`event.go`, `event_descriptor.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `failure.go`, `stream_check.go`) initially attached their header comment directly to `package agent` with no blank line. `backend/agent/Makefile`'s `make lint` (golangci-lint v2.9.0, revive's `package-comments` rule) flagged 3 of them (`event_descriptor.go`, `sequence.go`, `stream_check.go`) — revive requires exactly one file per package (`doc.go`) to carry a `"Package agent ..."`-formatted comment attached to the clause. Layer 1's own precedent (`ai/sequence.go`'s file comment states this explicitly) separates every non-`doc.go` file's header comment from `package ai` by a blank line, uniformly. Fixed the same way in all 7 files (not just the 3 flagged this run, for consistency with Layer 1's own uniform convention and to avoid relying on undocumented revive ordering behavior). Re-verified: `./src/agent/...` scoped lint is 0 issues.

## Phase 1 — AG-04.1: envelope, validation, per-lane ordering

Created `sequence.go` (`Sequence`, `LaneStamper`), `event_descriptor.go` (`EventDescriptor`, `BracketRole`, `Placement`, `Cardinality`, the six-step procedure doc), `event.go` (`EventKind`'s 4 constants, `Event`, `CheckEmit`, registry, `EventKinds()`), `run_events.go`'s `RunStart` half (deviation #1), `stream_check.go`'s minimal contiguity core, `doc.go`'s ordering-invariants prose (not a guarded row, AD-3), `envelope_test.go`, `agent_test_helpers_test.go`.

- **Task 1.6 RED — S-AEV-001/002/003**: disabled `CheckEmit`'s registered-kind check (`if false {...}`). `TestCheckEmit_NilPayload_FailsNamingTheMissingPayload` failed: `agent.CheckEmit(zero Event) = event.sequence: value is outside a documented bound, want errors.Is to match ai.ErrNotInVocabulary`. Reverted, GREEN.
- **Task 1.7 RED — S-AEV-004/005**: `Event.Turn()` hardcoded `true` for the second result. `TestEvent_Identity_ReadableFromExternalPackage` failed: `a run-scoped event reports a turn identity, want none`. Reverted, GREEN.
- **Task 1.8 RED — S-AEV-010/011**: `LaneStamper.Stamp` stopped incrementing. `TestLaneStamper_ConcurrentLanes_NoRaceAndBothStayContiguous` failed on every event: `sequence = 0, want N`. Reverted, GREEN.
- **Task 1.9 RED — S-AEV-012/013**: `CheckStream`'s contiguity check disabled (`if false {...}`). Both `TestCheckStream_SequenceNotStartingAtOne_RejectedNamingTheRule` and both subtests of `TestCheckStream_GapOrRepeat_RejectedNamingThePosition` failed: `no violation, want a rejection`. Reverted, GREEN.
- **Task 1.10 RED — S-AEV-020/021/022**: `NewRunStart` hardcoded a parent. `TestNewRunStart_TopLevel_ReportsNoParentAsADistinguishableState` failed: `event.Parent() reports a parent (scratch-break) for a top-level run-start, want none`. Reverted, GREEN.
- **Task 1.11/1.12 GREEN**: `go test -race -v ./src/agent/...` — all tests pass, including AG-03's three pre-existing guards, unchanged.

**Note on RED methodology**: production skeletons were written before test assertions (needed to establish the type system for 51 interdependent scenarios at this scale), so each RED above is a **targeted break-and-restore cycle** against already-implemented behavior — confirmed to fail for the right reason with a real command, then restored — rather than chronological pre-implementation RED. This is disclosed explicitly per the honesty requirement; every RED below follows the same, real, command-verified methodology.

## Phase 2 — AG-04.2: run/turn lifecycle, validator scope engine

Extended `run_events.go` with `RunEnd`/`RunOutcome`; created `turn_events.go` (`TurnStart`/`TurnEnd`/`TurnOutcome`); created `failure.go` (deviation #2); extended `stream_check.go` with the full two-level scope engine; created `stream_check_test.go`.

- **Task 2.4 RED — S-AEV-030…034**: disabled the run-start duplicate check. `TestCheckStream_TwoRunStarts_RejectedNamingDuplicate` failed: `= stream: value is not well-formed..., want errors.Is to match ai.ErrDuplicate`. Reverted, GREEN.
- **Task 2.5 RED — S-AEV-035**: disabled `RunEnd.validate`'s outcome-membership check. The "no route without an outcome" subtest of `TestNewRunEnd_TypedOutcome_DistinguishableAndRequired` failed: `= nil error, want a rejection`. Reverted, GREEN.
- **Task 2.6 RED — S-AEV-040…043**: disabled the turn-overlap check. `TestCheckStream_OverlappingTurnStart_RejectedNamingOverlap` failed: `no violation, want a rejection`. Reverted, GREEN.
- **Task 2.7 RED — S-AEV-044**: disabled `TurnEnd.validate`'s outcome-membership check. The "no route without an outcome" subtest of `TestNewTurnEnd_TypedOutcome_DistinguishableAndRequired` failed the same way. Reverted, GREEN.
- **Task 2.8 RED — S-AEV-050/051**: unexported `CheckStream` (renamed to `checkStreamScratchBreak`). `go vet ./src/agent/...` failed to compile: `vet: src/agent/envelope_test.go:180:21: undefined: agent.CheckStream` — proves the exported, production-callable surface is genuinely required by external callers, not incidentally true. Reverted, GREEN.
- **Task 2.9 RED — S-AEV-052/053/054**: corrupted the `PlacementTurn` rule's documented text. `TestCheckStream_RuleCoverage_MatchesTheDocumentedList` failed naming the missing phrase. Reverted, GREEN.
- **Task 2.10/2.11 GREEN**: `go test -race -v ./src/agent/...` green; also re-proved the "same commit" bracket split matters by fixing Phase-1-era ordering tests that a 5-run-start sequence is now correctly rejected as a duplicate bracket opening under the full engine (`buildStampedRunStarts` kept for pure-contiguity tests where bracket role is irrelevant; a new `buildValidRunBracket` helper added for tests needing full acceptance).

## Phase 3 — AG-04.3: invariant pins, `L2C-04` doc-contract row

Added `L2C-04` row to `doc.go` and its byte-identical entry to `expectedLayer2ContractRows` in `doc_contract_guard_test.go`, same commit (AD-3). Added the delta-rule prose to `doc.go`. Wrote the `agent-package-scaffold` spec delta. Created `invariant_pin_test.go`.

- **Task 3.4 RED — S-AEV-060/061**: planted `ScratchAccumulatedMessageDelta` (exported type in `event.go`). `TestConstructionSurface_NoRouteFromDeltaKindToAccumulatedPayload` failed, naming the type and each forbidden substring it matched (`"Delta"`, `"Accumulat"`, `"Message"`). Reverted, GREEN.
- **Task 3.5 RED — S-AEV-070/071**: `Failure.Delivery()` hardcoded to always return `DeliveryPreStream`. `TestFailure_Delivery_DistinguishesPreStreamFromMidStream` failed: `pre.Delivery() = 1 and mid.Delivery() = 1, want distinguishable values`. Reverted, GREEN.
- **Task 3.6 RED — S-AEV-072/073**: (a) flipped the reflection-checked wanted type to `ai.DeliveryPath`; `TestFailure_CategoryMapping_IsIdentityDeclaredInSource` failed naming the real vs. wanted type. Reverted. (b) planted a scratch `_test.go` file containing the literal string `"provider failure: timeout"`; `TestSuite_NeverAssertsOnFailureMessageStringContent` failed, naming the file. Deleted, GREEN. (See also Deviations #6 — a genuine self-reference bug this same test caught in itself first.)
- **Task 3.7 RED — S-AEV-102 (bite)**: scratch-contradicted the ordering-invariants sentence in `doc.go`; `TestDocGo_StatesOrderingInvariantsAndMembershipCriterion` failed naming it missing. Reverted. Then scratch-contradicted the `L2C-04` row text; the same test failed naming the membership-criterion statement missing. Reverted. **Task 3.7's own resolution held**: the literal textual-presence pin (not a new guarded-row mechanism) correctly closes S-AEV-102 without reopening AD-3's "pinned behaviorally" decision for the ordering-invariants prose.
- Additionally re-proved AD-3's same-commit ordering constraint directly: temporarily removed only `L2C-04`'s table entry (leaving the row in `doc.go`); `TestLayer2DocContract_MatchesTheCommittedTable` failed with a 4-of-3 row-count mismatch, reproducing AG-03's own `S-AGP-013`/`S-AGP-014` bite shape. Reverted, GREEN.
- **Task 3.8 GREEN**: `go test -race -v ./src/agent/...` green.

## Phase 4 — AG-04.4: every-kind-constructible guard, scope fence

Created `event_registry_test.go` (witness table, two legs per kind, bidirectional cross-check against `agent.EventKinds()`).

- **Task 4.2 GREEN baseline**: `TestEventKindRegistration_EveryRegisteredKind_HasConstructorAndAccessor` passes; log line `every-kind-constructible guard constructed at least one instance of 4 registered kind(s)` (S-AEV-080/081).
- **Task 4.3 RED — S-AEV-082 (bite)**: planted `EventKindScratchNoPayload` (constant + registry row, no witness entry). Guard failed: `scratch_no_payload is carried by agent.EventKinds() but has no entry in eventKindWitnesses`. Reverted; confirmed `git diff --stat` empty against the last commit.
- **Task 4.4 RED — S-AEV-083 (bite)**: planted a witness entry for `agent.EventKind(99)`, not in the registry. Guard failed: `eventKindWitnesses carries a witness for eventkind(99), which agent.EventKinds() does not enumerate` — proving the cross-check is bidirectional, not containment-only. Reverted.
- **Task 4.5 RED/GREEN — S-AEV-090**: confirmed GREEN against the real 4-kind registry, then planted `EventKindScratchMessage` (`"message_scratch"`, a 5th kind). `TestEventKinds_ScopeFence_ExactlyRunAndTurnFamilies` failed on the count mismatch (`5 kinds, want 4`) — the scope fence correctly rejects a 5th kind before ever reaching the forbidden-name scan. Reverted.
- **Task 4.6 — S-AEV-091**: confirmed `TestEventDescriptorDoc_StatesTheSixStepProcedure` GREEN; RED-proved by corrupting step 5's documented text, confirmed the test names the missing phrase, reverted.
- **Task 4.7 RED/GREEN — S-AEV-092 (extensibility experiment)**: added a full 5th kind (`EventKindScratchExperiment`/`ScratchExperiment`) following the documented six-step procedure exactly, in a separate scratch file plus the unavoidable registry/witness-table data rows. `TestEventKindRegistration_EveryRegisteredKind_HasConstructorAndAccessor` (the guard) and a new `TestScratchExperiment_BracketRoleNone_RequiresAnOpenRun` (the validator's previously-unexercised `BracketRoleNone` branch) both passed with **zero edits** to `stream_check.go`'s or `event_registry_test.go`'s own logic — only data additions. The three scope-fence tests (S-AEV-005/060/090) correctly failed during the experiment, since a 5th kind violates their own "exactly 4" pin — expected collateral, not a defect, and itself confirms those pins are exact rather than "at least." Fully reverted (`scratch_experiment.go`, `scratch_experiment_test.go` deleted; `event.go`/`event_registry_test.go` diffs empty against the last commit).
- **Task 4.8 GREEN**: `go test -race -v ./src/agent/...` green with zero scratch artifacts (`git status --short` confirmed clean of scratch residue at every step).

## Phase 5 — Regression, evidence close-out, milestone doc update

- **5.1/5.2**: `git diff --stat origin/main -- backend/agent/src/agent/import_boundary_test.go backend/agent/src/agent/ambient_authority_test.go backend/agent/go.mod backend/agent/go.sum backend/agent/Makefile backend/agent/.golangci.yml` — empty. Byte-unchanged, confirmed.
- **5.3**: `make test` (`go test -race -v ./...`) — **all 12 packages `ok`**, zero FAIL, zero DATA RACE, both before this change (baseline, verified at session start) and after (verified post-implementation):
  ```
  ok  	github.com/cachicamas/backend/agent/src/agent                                    1.451s
  ok  	github.com/cachicamas/backend/agent/src/agenttest                                (cached)
  ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep                          (cached)
  ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest                      (cached)
  ok  	github.com/cachicamas/backend/agent/src/ai                                       3.380s
  ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry                        (cached)
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat                          (cached)
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest          (cached)
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter               (cached)
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance   (cached)
  ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke (cached)
  ok  	github.com/cachicamas/backend/agent/src/handoff                                  (cached)
  ```
  `make lint`: `./src/agent/...` scoped alone is **0 issues** (after Deviation #7's fix). Full-module `make lint` exits 1 solely due to Deviation #4's pre-existing, unrelated finding. Reported honestly (see Deviations #4, #7).
- **5.4**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md`'s status header updated from "4 of 24" to "5 of 24", naming AG-04's deliverables, following `dbd7a33e`'s own pattern exactly (one-line status blockquote edit).
- **5.5**: Every recorded RED assembled above, by phase. Real changed-line count and the forecast overage recorded in Deviation #5 and this file's Status section.
- **5.6**: `git status --short` clean at every scratch-revert checkpoint (verified via `git diff --stat` against the last commit after each bite); final state below.

## TDD Cycle Evidence

Every RED above is a real, command-verified failure (not a fabricated or reported-only claim) followed by a real passing re-run after revert — see the phase-by-phase sections for exact commands and output. RED methodology note: since production skeletons were written before test files at this scale (51 interdependent spec scenario ids: 45 `S-AEV` plus 6 restated `S-AGP`), RED evidence is a **targeted break-and-restore cycle** against already-implemented behavior throughout, disclosed once here rather than repeated per row.

| Phase / Requirement | Test file | RED | GREEN | Bite proof |
|---|---|---|---|---|
| 1.6–1.10 / R-AEV-001..003 | `envelope_test.go` | 5 recorded (S-AEV-001..022) | ✅ `go test -race` | N/A (behavior, not guard) |
| 2.4–2.9 / R-AEV-004..006 | `stream_check_test.go` | 6 recorded (S-AEV-030..054) | ✅ `go test -race` | N/A |
| 3.4–3.7 / R-AEV-007, 008, 011 | `invariant_pin_test.go` | 5 recorded (S-AEV-060..073, 102) | ✅ `go test -race` | S-AEV-102 (textual pin) |
| 4.3–4.7 / R-AEV-009, 010 | `event_registry_test.go` | 4 recorded (S-AEV-082/083/090/092) | ✅ `go test -race` | ✅ S-AEV-082/083/092 — closes only on recorded bite, per spec's own Evidence discipline |

### Test Summary

- **Total new/extended test files**: `envelope_test.go`, `stream_check_test.go`, `invariant_pin_test.go`, `event_registry_test.go`, `agent_test_helpers_test.go` (shared helpers), plus `doc_contract_guard_test.go`'s one-entry amendment.
- **Total tests passing at close**: all 12 packages `ok` under `go test -race -v ./...`.
- **Layers used**: Unit/behavior/guard only — no integration or E2E layer exists in this module; no producer exists until wave 2 (0003:417-418), so every test is hand-built through the public surface.
- **Non-claims restated** (risk 4, `0003:2203`): no test or comment in this change claims AG-04 closes envelope invariant 3, or invariants 1, 2 or 4 on its own. `doc.go`'s own prose states the partial-closure map explicitly.

## Task completion

All 45 tasks in `tasks.md`'s Phases 1–5 marked `[x]` (45/47 of the whole file). Phase 6's 2 tasks (spec promotion) intentionally untouched — happens at archive, per this repo's convention.

## Post-verify corrections

Scoped correction transaction, driven by `sdd-verify`'s findings (`verify-report.md`, verdict **PASS WITH WARNINGS**). Four fixes, no binding design decision (AD-1..AD-6) changed, no test weakened or deleted, no new event kind or family, `agenttest` still never imported, AG-03's guards still byte-unchanged. `git diff --numstat` for `backend/agent/src/agent`: **175 insertions, 22 deletions** across 4 files (`stream_check.go`, `stream_check_test.go`, `envelope_test.go`, `agent_test_helpers_test.go`) — `event.go`'s two break-and-restore cycles left it byte-clean.

### Fix 1 — the reported violation position was the sequence value, not the slice index (W2)

**Defect**: every one of `stream_check.go`'s 12 `ai.AtIndex("event", ...)` call sites passed `int(seq)` — the offending event's own `Sequence()` value — instead of its position in the slice being checked. `CheckStream([seq1, seq3])` reported `event[3]` for a 2-element slice; index 3 does not exist in it.

**Choice**: 0-based slice index (Go `range`'s own `i`), not 1-based. Two reasons, both from existing code, not invented for this fix:
1. `agenttest.CheckContiguity` (`src/agenttest/stream_kit_ordering.go:47-70`) is Layer 1's own analogous check for this exact class of violation (gap/repeat in a checked slice), and it reports `event[%d]` using the 0-based `range` index `i` throughout (`event[0]`, `event[i-1]`, etc.) — never the sequence value. AG-04's `CheckStream` explicitly absorbs the job Layer 1 split off into this kit (design AD-1 divergence 1), so mirroring its position convention, not inventing a new one, is the same "mirror the architecture" posture AD-1 already commits to.
2. Layer 1's own `ai.AtIndex` call sites are dominated by this same convention: `request.go`, `content_part.go` and `request_extension.go` all pass a bare `for i, x := range slice` index (9 of 11 non-`stream_check.go` call sites). The two exceptions are `ai/stream_check.go` itself, which passes `int(seq)` — but that file never checks contiguity at all (delegated to `agenttest`), so its `seq` is only ever read for an already-contiguity-assumed stream, not a position within a slice that might itself be gapped — the exact case this fix is closing.

**Fix**: `for _, e := range events` → `for i, e := range events`; all 12 `ai.AtIndex("event", int(seq))` → `ai.AtIndex("event", i)`. `seq` is retained for the contiguity comparison itself. Doc comment updated to state the convention and why. `backend/agent/src/agent/stream_check.go`.

### Fix 2 — R-AEV-006's "MUST name the offending position" was untested (W1)

**Defect**: verify proved this by deleting `ai.AtIndex("event", int(seq))` from all 12 sites and getting a fully green suite — no test read a violation's position at all.

**Fix**: added `requireViolationPosition(t, violation, wantIndex)` to `agent_test_helpers_test.go` — reaches `*ai.Violation` via `errors.As`, reads `.Path()`, and requires the last step to carry an index equal to `wantIndex`; fails on a missing index (catches deletion) and on a wrong one (catches Fix 1's own defect). Wired into two existing tests, chosen to cover two different rule classes:
- `TestCheckStream_GapOrRepeat_RejectedNamingThePosition` (S-AEV-013, `envelope_test.go`) — contiguity. The "repeat" subtest's data was restructured from two bare run-starts (which would themselves trip the duplicate-open bracket rule before reaching four contiguous elements) to a bracket-valid 4-event prefix (`buildValidRunBracket`) plus a repeat of its first element, so the offending index (4) and its sequence value (1) cannot coincide by accident.
- `TestCheckStream_SingleRuleViolation_NamesOnlyThatRule` (S-AEV-052, `stream_check_test.go`) — R-AEV-006's own scenario, a bracket-misplaced violation.

**RED recorded** (`go test ./src/agent/... -run 'TestCheckStream_GapOrRepeat_RejectedNamingThePosition|TestCheckStream_SingleRuleViolation_NamesOnlyThatRule' -v`, against the pre-Fix-1 production code):
```
stream_check_test.go:445: violation position index = 1, want 0 (...)
--- FAIL: TestCheckStream_SingleRuleViolation_NamesOnlyThatRule
envelope_test.go:285: violation position index = 3, want 1 (...)
envelope_test.go:305: violation position index = 1, want 4 (...)
--- FAIL: TestCheckStream_GapOrRepeat_RejectedNamingThePosition (gap, repeat)
```
Exactly the sequence-value-not-index defect (W2), reproduced by the new assertions before Fix 1 landed. GREEN after Fix 1.

### Fix 3 — `Event.Turn()`'s identity value was unproven (W4)

**Defect**: `TestEvent_Identity_ReadableFromExternalPackage` (S-AEV-004) asserted only the *absent* direction (a run-scoped event reports no turn). `Event.Turn()` hardcoded to return `"SCRATCH_BREAK"` for every turn-scoped event left the suite green.

**Fix**: added a subtest constructing two turn-start events with different turn ids, asserting each `Turn()` result equals the id it was built with, and that the two results differ from each other — a hardcoded return cannot satisfy either the per-value checks or the distinguishing check. `envelope_test.go`. Production code (`event.go`) was already correct; no production change was needed for this fix.

**RED recorded** (break-and-restore, `Event.Turn()` temporarily changed to `return "SCRATCH_BREAK", e.hasTurn`):
```
envelope_test.go:154: first.Turn() = SCRATCH_BREAK, want turn-alpha
envelope_test.go:162: second.Turn() = SCRATCH_BREAK, want turn-beta
envelope_test.go:166: first.Turn() and second.Turn() both report SCRATCH_BREAK for two differently-constructed turns, want distinguishable values
--- FAIL: TestEvent_Identity_ReadableFromExternalPackage/turn-scoped_event
```
Reverted; `go test` green again; `git diff --stat -- event.go` empty, confirming a byte-clean revert.

### Fix 4 — `EventDescriptor.Terminal` was written but never read (W3)

**Defect**: verify proved this by flipping the only `Terminal: true` (on `EventKindRunEnd`'s registry row) to `false` and finding the suite still green — "nothing follows the terminal" was actually driven by the `BracketRoleClosesRun`-derived run-bracket state (`run == runBracketClosed`), never by the `Terminal` field the descriptor declares and documents for exactly this purpose.

**Choice**: (a) — made the validator read `Terminal`, rather than removing the field. `Terminal bool` is part of `EventDescriptor`'s shape as AD-1 itself states it ("the Layer 2 descriptor carries the new dimension as data: ... `Terminal bool`"), so removing it would touch AD-1's own declared contract; reading it does not. It is also the literal mechanism AD-1 says this validator mirrors from Layer 1's own engine (Layer 1's `ai.CheckStream` tracks a `terminalSeen` boolean from `descriptor.Terminal` directly — `src/ai/stream_check.go:83,101-103`) — AG-04's validator had simply not carried that specific piece of the mirror over yet.

**Fix**: added `terminalSeen bool`, set from `d.Terminal` at the end of each event's processing; the top-of-loop "nothing follows the terminal" check now reads `terminalSeen` instead of `run == runBracketClosed`. The run-bracket state machine itself is unchanged and still drives every other rule (duplicate-open, unclosed-turn-at-close, escaped-turn, incomplete-run-at-end). For AG-04's own 4 registered kinds this is behavior-preserving — `EventKindRunEnd` is the only kind carrying either property, so `terminalSeen` becomes true at exactly the point `run` becomes `runBracketClosed` — confirmed by the full suite staying green with zero other test changes. `backend/agent/src/agent/stream_check.go`.

**Test added**: `TestCheckStream_Source_ReadsTheTerminalDescriptorField` (`stream_check_test.go`) — parses `stream_check.go`, locates `CheckStream`'s function body via `go/ast`, and fails unless a `*ast.SelectorExpr` named `Terminal` is found in it. A *behavioral* test cannot distinguish "driven by `Terminal`" from "driven by `BracketRoleClosesRun`" for AG-04's own registry, because the two properties currently coincide on every reachable input (traced through all 4 kinds × all 6 rule branches) — this structural test is what actually closes W3.

**RED recorded** (against the pre-fix production code, `Terminal` not yet referenced anywhere in `stream_check.go`):
```
stream_check_test.go:542: CheckStream's body never references .Terminal — the field is written by the registry (event.go) but never read by the engine (...)
--- FAIL: TestCheckStream_Source_ReadsTheTerminalDescriptorField
```
GREEN after the fix.

**A second, independent break-and-restore**, disabling only the wiring (`if false && d.Terminal { terminalSeen = true }`, the read stays present so the structural test alone stays green) confirmed the fix is genuinely load-bearing, not coincidental: the structural test cannot detect a dead condition (it only proves a reference exists), but the pre-existing, byte-unchanged `TestCheckStream_EventAfterRunEnd_RejectedNamingTheFollowingEvent` (S-AEV-033) does — its trailing run-start's own `BracketRoleOpensRun` check independently fires `ai.ErrDuplicate` once `terminalSeen` no longer intercepts first, instead of the expected `ai.ErrMisplaced`:
```
stream_check_test.go:99: agent.CheckStream(event after run-end) = event[2]: ..., want errors.Is to match ai.ErrMisplaced
--- FAIL: TestCheckStream_EventAfterRunEnd_RejectedNamingTheFollowingEvent
```
Reverted; full suite green; `git diff --stat -- stream_check.go` matched the intended Fix 1 + Fix 4 diff exactly, confirming a clean revert of both scratch breaks.

### Scenario-count correction (verify's "51 vs 45" reconciliation, W9)

The spec declares **45** unique `S-AEV` scenario ids; the change total of **51** is 45 `S-AEV` plus **6** restated `S-AGP` ids from the `agent-package-scaffold` delta — both figures are correct, but several statements in this change's own artifacts attributed all 51 to `S-AEV`, which is wrong. Corrected:
- `tasks.md:19` — "(51 spec scenario ids)" → "(45 `S-AEV` scenario ids; together with 6 restated `S-AGP` scenario ids ..., 51 spec scenario ids total across the change)".
- `apply-progress.md` — the Status section's "All 51 spec scenarios (`S-AEV-001`…`S-AEV-102`)", the review-budget-overage paragraph's "51 spec scenarios across 11 charter Gherkin scenarios", and the TDD Cycle Evidence closing note's "(51 interdependent scenarios)" — each reworded to state 45 `S-AEV` plus 6 restated `S-AGP` explicitly. Deviation #0's own "45 unique `S-AEV-0NN` ids" was already correct and is unchanged.

### Final verification

`cd backend/agent && make test` (`go test -race -v ./...`): **all 12 packages `ok`, zero FAIL, zero DATA RACE**. `go build ./...`: clean. `bin/golangci-lint run ./src/agent/...`: **0 issues**. Full-module `make lint`: the same single pre-existing, unrelated finding as before this correction (`src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go:17`, byte-unchanged vs. `origin/main`). `git diff --stat` for `import_boundary_test.go`, `ambient_authority_test.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`: empty. `git status --short`: clean of scratch residue (both break-and-restore cycles fully reverted before this section was written).
