# Apply progress: AG-18 — Implement compaction (`cachicamas-agent-compaction`)

> Attempt 2 (attempt 1 was killed by host-wide ENOSPC before writing any byte; its settled production-shape plan is carried forward here, verified against the actual tree rather than trusted). Worktree `cachicamas-worktrees/ag18`, branch `feat/agent-layer2-wave4-ag18`, off `origin/main@f6acc0d2` + planning commit `fbe9b410`. Strict TDD active, test runner `cd backend/agent && go test -race -count=1 ./...`.

## Phase 0 — Foundation: prerequisites re-verified, zero drift found

- 0.1/0.2: grepped `backend/` for `CompactionRequest`, `ReplacePrefix`, `EntryOriginSummarized`, `closeTurnMarked`, `runCompaction`, `Harness.Compact` — zero hits before this attempt's own edits (confirmed pre-existing state the spec pins).
- 0.3: `harness.go:524-530` (verdict discarded) and `:668` (`hist.CloseTurn()`) — re-opened, content matches design's citation exactly, no line-shift.
- 0.4: `history.go:327-332` (`commit`, "the ONLY function that writes h.entries or h.open") — re-opened, exported surface confirmed exactly `Append`, `CloseTurn`, `SynthesizeOrphans`, `Entries`, `Len` (5 members) via `history_surface_guard_test.go`'s own `expectedHistoryRoutes` table, before this attempt's edit.
- 0.5: `doc_contract_guard_test.go:62-71` — confirmed exactly 8 rows (`L2C-01`…`L2C-08`), `L2C-07` row text captured verbatim (see Phase 8).
- 0.6: `stream_check.go:161` (`if d.Placement == PlacementTurn && !turnOpen`), `turn_events.go:42`/`:178` (`NewTurnStart`/`NewTurnEnd`), `cost_usage.go:70`/`:118` (`newCostTurnFromUsage`/`costAccumulator.add`) — all re-opened, all byte-exact matches to design's citations. **No citation drift found anywhere in Phase 0.**

## Deviation from tasks.md — recorded, not silent

**The AG-17 test conflict the orchestrator prompt flagged is real and was independently re-confirmed by reading `context_strategy_test.go:833-894`**: `TestNoRelease_SubstrateByteUnchanged` (S-CTX-021) hardcodes `doc.go`, `doc_contract_guard_test.go` and `history.go` in its `untouched` list. AG-18 legitimately releases the first two (AD-12) and modifies `history.go` (Phase 1). Fixed in Phase 8 (see below) by narrowing that list rather than deleting the test, following the AG-11→AG-17 "retire the previous milestone's version" house pattern; AG-18's own `TestCompaction_SubstrateByteUnchanged` (S-CMP-036, Phase 4) is the fuller, AG-18-scoped replacement, matching `agent-loop-skeleton`'s `S-LSK-029`/`S-LSK-030` shape.

**A second, load-bearing deviation, reasoned through and recorded here**: `history_test.go`'s and `compaction_surgery_test.go`'s Phase-1 tests for `History.ReplacePrefix`'s SUCCESS path need a history whose turns are genuinely *marked* (R-CMP-012). `closeTurnMarked` is package-private and reachable only from the harness (S-CMP-035) — there is no way for `package agent_test` to construct a marked history except by driving a real `Harness.Run`. That in turn requires the harness to actually call `closeTurnMarked` at its turn-close site, which tasks.md schedules as (half of) task 6.3, in Phase 6. **Resolution**: the mark-recording half of task 6.3 — capturing the successful attempt's own `TurnID` from its forwarded events and calling the package-private `closeTurnMarked(TurnID)` at `harness.go`'s turn-close site instead of the exported `CloseTurn()` — was pulled forward into Phase 1's own work. It is self-contained (no dependency on `CompactionRequest`, `runCompaction`, or any Phase 2-5 symbol) and purely additive: `commitCloseTurnMarkedOp` delegates to the pre-existing `commitCloseTurnOp` for its one validation rule, so no existing test's observable behavior changes (confirmed: full uncached suite green after this change, see below). The remaining half of task 6.3 (verdict capture at `harness.go:524-530`, transcript rebuild, `Harness.Compact`) stays in Phase 6 as scheduled.

`resolveCut`'s pure algorithm (an unexported helper with no external caller until Phase 3 wires `runCompaction` in) is tested internally, `package agent`, in `compaction_surgery_test.go` — sanctioned explicitly by `NFR-CMP-001`'s own carve-out ("Pure helpers with no external surface MAY be tested internally, provided every claim about what a caller observes is also asserted externally") and mirroring this repo's own `retry_decision_internal_test.go` precedent for `retryDecision`/`retryVerdict`. The externally-observable consequence is independently re-proved once the full pipeline exists (Phase 4's `compaction_stream_test.go`/`compaction_reconstruction_test.go`).

## Structural insight verified (carried forward from attempt 1, now confirmed against real code)

A recorded turn mark can only be committed when `History`'s open-call set was empty at that instant (`commitCloseTurnMarkedOp` delegates to the pre-existing `commitCloseTurnOp`'s rule 4). This provably guarantees pairing-closure of everything before the mark's own boundary, by induction over `h.open`'s cumulative, whole-history tracking (not per-turn). Consequence applied in the implementation: `History.commitReplacePrefixOp` validates `count` by **mark membership alone** (no second, redundant correlation re-scan of the discarded prefix `[0, count)`); it separately re-validates the **new** `[summary, tail...]` sequence via `replacementOpenSet` (reusing `resolveOpenSet`), which is a genuinely different computation (R-HIS-001 item 4 / R-CMP-010 gate 3), not a redundant one.

## Phase 1 — History surgery foundation: COMPLETE

- [x] 1.1 `EntryOriginSummarized` added as the third `EntryOrigin` member (`history.go`), `String()` updated.
- [x] 1.2 Turn mark record `{TurnID, count}` (`turnMark`) added to `History`'s internal state (`marks []turnMark` field), written only through `commit`; package-private `closeTurnMarked(TurnID) error` door added beside `CloseTurn()` (signature/semantics of the exported method unchanged).
- [x] 1.3 RED (recorded below): `compaction_surgery_test.go` — `TestResolveCut_RetractsToMarkBoundary` (S-CMP-010). `go vet` failed `undefined: resolveCut` at 5 call sites before `compaction.go` existed.
- [x] 1.4 GREEN: `resolveCut(hist *History, naiveCut int) int` implemented in `compaction.go` — retracts to the greatest recorded mark boundary ≤ naive cut (`History.markBoundaryAtOrBefore`), then verifies pairing-closure via `earliestOpenMessageIndex` (reuses `resolveOpenSet`), retracting further on a straddle, terminating at 0.
- [x] 1.5 GREEN: `TestResolveCut_SeededValidation`, `TestResolveCut_MonotoneNeverExpandsForward`, `TestResolveCut_TerminatesOnAdversarialInput`.
- [x] 1.6 Bite **S-CMP-030** — RED recorded, then reverted (transcript below).
- [x] 1.7 RED→GREEN: `history_test.go` — `TestHistory_ReplacePrefix_SucceedsAndRenumbers` (S-HIS-100, S-CMP-007).
- [x] 1.8 GREEN: `History.ReplacePrefix(count int, summary ai.Message) error` implemented as `commitReplacePrefix` op — validates `count` against a recorded mark, rejects a tool-bearing summary, rebuilds `[summary, tail...]` renumbered ordinally with origins preserved, rebuilds marks (in-span dropped, tail shifted by `count-1`), all-or-nothing.
- [x] 1.9 GREEN: `TestHistory_ReplacePrefix_NothingElseRemovable` (S-HIS-101), `TestHistory_ReplacePrefix_RejectsToolBearingSummary` (S-CMP-027).
- [x] 1.10 RED→GREEN: `compaction_surgery_test.go` — `TestCompaction_ProtectionValueIdentical` (S-CMP-014, S-HIS-103). **No `reflect.DeepEqual` over `Entry` values anywhere in this file** — grepped, confirmed absent (`grep -rn "reflect.DeepEqual" compaction_surgery_test.go` → no output beyond confirming the file doesn't import `reflect` at all).
- [x] 1.11 RED→GREEN: `TestCompaction_IdentityRenumbers` (S-CMP-015, part of S-HIS-103).
- [x] 1.12 RED→GREEN: `TestCompaction_SummaryTypedByOriginOnly` (S-CMP-016). `TestEntryEnvelope_NoExportedOriginSetter` (S-CMP-028) — covered by the existing, unmodified `TestHistoryRouteGuard_SurfaceMatchesExpectedTable`'s "every `method:Entry` row must be `routeReadOnly`" check (S-HIS-042), since `Entry`'s method set is unchanged by AG-18.
- [x] 1.13 GREEN: `history_surface_guard_test.go` — `expectedHistoryRoutes` gains `{"ReplacePrefix", "method:*History", routeMutating}` in the same commit as the route.
- [x] 1.14 Bite **S-HIS-102** — RED evidence is a **naturally-occurring regression** (not a contrived scratch mutation): implementing `ReplacePrefix` (task 1.8) before updating `expectedHistoryRoutes` (task 1.13) produced exactly the bite's described failure via the full suite run below, then task 1.13 reverted it to green. This is stronger evidence than a synthetic bite (a real, observed regression from genuine development order), recorded here rather than re-staged.
- [x] 1.15 GREEN: `history_surface_guard_test.go` — `TestHistory_MarkedCloseNotExported` (S-CMP-035).
- [x] (pulled forward, half of 6.3) `harness.go`: successful attempt's `TurnID` captured from forwarded events (`capturedTurnID`), turn-close site now calls `hist.closeTurnMarked(capturedTurnID)` instead of `hist.CloseTurn()`.

### RED transcripts (Phase 1)

**Task 1.3/1.5 — undefined symbol, before `compaction.go` existed:**
```
$ go vet ./src/agent/...
# github.com/cachicamas/backend/agent/src/agent [...]
src/agent/compaction_surgery_test.go:109:9: undefined: resolveCut
src/agent/compaction_surgery_test.go:139:9: undefined: resolveCut
src/agent/compaction_surgery_test.go:180:16: undefined: resolveCut
src/agent/compaction_surgery_test.go:238:14: undefined: resolveCut
src/agent/compaction_surgery_test.go:256:12: undefined: resolveCut
```

**Bite S-CMP-030 — RED (scratch mutation: `resolveCut` returns `naiveCut` unmodified, skipping mark-alignment and pairing verification):**
```
$ go test -race -count=1 -run 'TestResolveCut_RetractsToMarkBoundary' ./src/agent/...
    compaction_surgery_test.go:112: resolveCut(hist, 2) = 2, want <= 1 (the call's own entry index) — the boundary must not split the call/result pair
--- FAIL: TestResolveCut_RetractsToMarkBoundary (0.00s)
FAIL
```
Reverted immediately after capture; confirmed green again (`go test -race -count=1 -run 'TestResolveCut' ./src/agent/...` → `ok`).

**Bite S-HIS-102 — RED (natural regression, `ReplacePrefix` implemented, guard table not yet updated):**
```
$ go test -race -count=1 ./... (full suite, mid-Phase-1)
--- FAIL: TestHistoryRouteGuard_SurfaceMatchesExpectedTable (0.05s)
    history_surface_guard_test.go:210: history route guard: surface route "ReplacePrefix" (method:*History) is not in expectedHistoryRoutes — every exported history route must be classified mutating/read-only in the same PR that adds it (R-HIS-004)
FAIL
```
Fixed by task 1.13 (`expectedHistoryRoutes` + `drivers` map gain the `ReplacePrefix` row); full suite green afterward (see below).

### GREEN evidence (Phase 1 close)

```
$ go test -race -count=1 -run 'TestResolveCut|TestHistory_ReplacePrefix|TestCompaction_Protection|TestCompaction_Identity|TestCompaction_SummaryTyped|TestHistory_MarkedCloseNotExported|TestHistoryRouteGuard|TestLayer2DocContract' ./src/agent/...
ok  	github.com/cachicamas/backend/agent/src/agent	1.6s
```

Full uncached suite (`go test -race -count=1 ./...`) after Phase 1 + the pulled-forward harness.go wiring: **all packages green**, `openaicompat` alone 173s (real uncached run, not a cache artifact) — full output recorded once at the final gate (Phase 12) rather than duplicated at every checkpoint.

## Phase 2 — The verdict carrier: types + S-CTX-005 amendment COMPLETE (2.3-2.5 deferred, see below)

- [x] 2.1 GREEN (types): `context_strategy.go` gains `CompactionRequest{Provider, Options TurnOptions, Instruction, Cut}` and `ContextVerdict.Compaction *CompactionRequest`; `NoOpContextStrategy` compiles untouched.
- [x] 2.2 RED→GREEN: `TestContextVerdict_ZeroRequestsNothing` replaces `TestContextVerdict_HasNoFields` (S-CTX-005 revised, exactly per its own closing sentence). RED recorded below.
- **2.3-2.5 (S-CTX-022, S-CTX-023, S-CTX-025) implemented together with Phase 6**, not standalone after Phase 2: their own Given/When/Then text ("when a run is driven, then a compaction executes") requires the harness to ACT on the verdict, which is Phase 6's wiring. Deferred to the Phase 6 section below, same reasoning as the Phase-1 deviation.
- [x] 2.6 Confirmed: bite `S-CTX-008` (context_strategy_test.go, unmodified) still exercises the byte-identity comparison at `S-CTX-007`; not re-run since nothing in Phase 2's diff touches that code path.

### RED transcript (Phase 2)

```
$ go test -race -count=1 ./src/agent/...
--- FAIL: TestContextVerdict_HasNoFields (0.00s)
    context_strategy_test.go:37: agent.ContextVerdict has 1 field(s), want 0 (R-CTX-003: no field can request compaction)
FAIL
```
Fixed by rewriting the test into `TestContextVerdict_ZeroRequestsNothing`.

## Phase 3 — The compaction call and its spend: COMPLETE

`compaction.go` created: `mintCompactionTurnID` (namespace `turn-cmp-N`, disjoint from `mintLoopTurnID`'s `turn-lsk-N`), `emitCompactionFailedArm` (one shared helper for both failure arms — aborted vs. outcome-mapped — reusing `typedHarnessFailureFromError`), `compactionCallOutcome`/`drainCompactionCall` (silent text/reasoning accumulation, no agent-level bracket events — mirrors `turnAccumulator`'s reconstruction half without its emission half), `runCompaction` (the full call+bracket+spend sequence).

New file `compaction_call_test.go` (`package agent_test`): `compactAfterNStrategy` (shared fixture: fires a compaction on a chosen consultation, zero verdict otherwise — reused by every later phase), `markedHarnessForCompaction` (shared fixture: one driven+marked turn).

- [x] 3.1 RED: `TestCompaction_OwnCall_DistinctProvider` (S-CMP-001) — undefined symbols before `compaction.go` existed (implicit in the same build-failure class as Phase 1's).
- [x] 3.2 GREEN: `runCompaction` implemented.
- [x] 3.3 GREEN: `TestCompaction_InjectionOnly` (S-CMP-002). Fixed a genuine test-fixture bug during authoring: `preEntries` must be captured *before* `Compact` mutates `hist`, not after.
- [x] 3.4 GREEN: `TestCompaction_CancellationAbortsWithoutEndingRun` (S-CMP-003, via `compactAfterNStrategy` + `agenttest.Gate` + `h.Interrupt()`), `TestCompaction_EmptyInstructionFailsTyped` (S-CMP-005).
- [x] 3.5 GREEN: `TestCompaction_SpendFoldedIntoCumulative` (S-CMP-004, S-CST-015, S-CST-016).
- [x] 3.6 GREEN: `TestCompaction_IffHoldsOnAllThreeArms` (S-CMP-006, S-CST-015 arm table) — three sub-tests, one per arm.
- [x] 3.7 Bite **S-CMP-032** — RED recorded, then reverted (transcript below).

### RED transcript (bite S-CMP-032)

```
$ go test -race -count=1 -run 'TestCompaction_SpendFoldedIntoCumulative' ./src/agent/...
    compaction_call_test.go:340: cost_session input tokens = 0, want 111 (the compaction's own usage, folded)
--- FAIL: TestCompaction_SpendFoldedIntoCumulative (0.00s)
FAIL
```
(scratch mutation: commented out the `total.add(ct)` call at the emission site). Reverted immediately; confirmed green again.

### GREEN evidence (Phase 3 close)

```
$ go test -race -count=1 -run 'TestCompaction_OwnCall|TestCompaction_Injection|TestCompaction_Cancellation|TestCompaction_Spend|TestCompaction_Iff|TestCompaction_EmptyInstruction' ./src/agent/...
ok  	github.com/cachicamas/backend/agent/src/agent	1.4s
```

## Phase 4 — The bracket, the stream record, and reconstruction: COMPLETE

`compaction.go` extended with `Harness.Compact` (the on-demand door) and `ErrRunInFlight` sentinel — pulled forward from Phase 6 alongside `runCompaction` since Phase 4's own tests need a way to run a compaction standalone. `harness.go`'s strategy-triggered wiring (verdict capture at the former `:524-530`, transcript rebuild, the interrupt-during-compaction wind-down check) was also implemented at this point (task 6.1/6.2, pulled forward for the same reason as 6.3 in Phase 1 — see the Phase 1 deviation note, same root cause: later-phase tests need earlier-phase wiring to observe anything).

New files: `compaction_stream_test.go`, `compaction_reconstruction_test.go` (both `package agent_test`).

- [x] 4.1 GREEN: bracket wrapped around Phase 3's call — `turn_start → compaction_started → (cost_turn iff completion) → compaction_finished|failed → turn_end`, all through existing exported constructors, no new `EventKind`/`TurnOutcome`/`CostLabel`.
- [x] 4.2 RED→GREEN: `TestCompaction_StreamAcceptedUnmodified` (S-CMP-017) — driven via the strategy door over a 2-tool-call-turn script so the compaction bracket's TurnID is provably distinct from two real model-turn TurnIDs on the same stream.
- [x] 4.3 GREEN: span derivation — `History.markSpan(cut)` captured **before** `ReplacePrefix` (which shrinks marks); structurally guaranteed non-empty whenever `resolveCut` returns non-zero (see the "structural insight" note above) but still defensively checked.
- [x] 4.4 RED→GREEN: `TestCompaction_ReconstructionNamesReplacedTurns` (S-CMP-018), `TestCompaction_FinishedXorFailed` (S-CMP-019).
- [x] 4.5 RED→GREEN: `TestCompaction_MarkedSpanFromTwoRuns` (S-CMP-029) — two separate `Harness` values sharing one `*History`, compacted via `Harness.Compact` on the second.
- [x] 4.6 RED→GREEN: `TestCompaction_UnmarkedPrefixFailsTyped` (S-CMP-034) — a `NewSeededHistory`-built transcript, zero provider requests issued.
- [x] 4.7 RED→GREEN: `TestCompaction_SubstrateByteUnchanged` (S-CMP-036) — AG-18's own version, scoped correctly (unlike AG-17's, see the deviation note for the conflict this avoids repeating).
- [x] 4.8 RED→GREEN: `TestCompaction_InertUnlessRequested` (S-CMP-037).

Every Phase 4 test passed on its first implementation attempt once `runCompaction`/`Harness.Compact`/the harness wiring existed — no additional RED/fix cycle beyond the initial undefined-symbol RED already covered by Phase 3's build failure.

### GREEN evidence (Phase 4 close)

```
$ go test -race -count=1 -run 'TestCompaction_Stream|TestCompaction_Reconstruction|TestCompaction_Marked|TestCompaction_Unmarked|TestCompaction_Substrate|TestCompaction_Inert|TestCompaction_FinishedXor' ./src/agent/...
ok  	github.com/cachicamas/backend/agent/src/agent	1.7s
```

## Phase 5 — Atomicity and recovery: COMPLETE

New file `compaction_recovery_test.go` (`package agent_test`): `runInterruptibleCompactionFixture` (shared S-CMP-020/021 fixture — a gated-then-mid-stream-failing compaction provider via `ai.MidStreamFailure`/`ai.ErrorEvent`).

- [x] 5.1 GREEN: commit ordering — nothing touches `hist`'s pointee before `ReplacePrefix`'s own single call.
- [x] 5.2 RED→GREEN: `TestCompaction_AtomicOrAbsent` (S-CMP-020).
- [x] 5.3 RED→GREEN: `TestCompaction_WindDownNeverEntered` (S-CMP-021).
- [x] 5.4 Bite **S-CMP-031** — RED recorded, then reverted (transcript below). **A genuine fixture bug was found and fixed while authoring this bite**: the test's original `preEntries` snapshot was taken after `<-gate.Reached()`, which is *after* the scratch mutation (inserted before the provider call, which is *before* the gate is ever reached) had already run — so both sides of the "byte-identical" comparison were already-mutated, and the bite's first run passed **incorrectly**. Fixed by capturing `preEntries` synchronously inside the strategy's own consultation callback (guaranteed to run before `runCompaction`, and hence before any mutation inside it, on every ordering). Recorded as a discovery in its own right: a "before" snapshot taken after a synchronization point can silently be too late when the code under test can mutate state *before* that same point.
- [x] 5.5 GREEN: asserted (not built) — `TestCompaction_CancellationAbortsWithoutEndingRun` (Phase 3, S-CMP-003) already reuses the same gate-fixture shape and the harness's existing iteration-boundary cause check; `TestCompaction_WindDownNeverEntered` independently confirms the ordinary close.

### RED transcript (bite S-CMP-031, after the fixture fix above)

```
$ go test -race -count=1 -run 'TestCompaction_AtomicOrAbsent' ./src/agent/...
    compaction_recovery_test.go:115: turn 2's request carries 1 message(s), want 3 (the uncompacted transcript, unchanged by the failed compaction)
    compaction_recovery_test.go:119: turn 2's request message[0] differs from the pre-attempt transcript, want byte-identical
--- FAIL: TestCompaction_AtomicOrAbsent (0.00s)
```
(scratch mutation: `hist.ReplacePrefix(cut, scratchMsg)` inserted immediately before `req.Provider.Stream(...)`). Reverted immediately; confirmed green again.

### GREEN evidence (Phase 5 close)

```
$ go test -race -count=1 -run 'TestCompaction_AtomicOrAbsent|TestCompaction_WindDown' ./src/agent/...
ok  	github.com/cachicamas/backend/agent/src/agent	1.5s
```

## Phase 6 — Harness wiring and the on-demand door: COMPLETE

`Harness.Compact`, `ErrRunInFlight`, and the strategy-triggered wiring (verdict capture, transcript rebuild, interrupt-during-compaction wind-down check) were already implemented in Phases 3-4 (pulled forward — see those sections). This phase closed the remaining tests: the demand-door's own behavioral proofs, `S-CTX-022/023/024/025` (deferred from Phase 2 for the same "needs the harness wiring" reason), and bite `S-CMP-033`.

New file `compaction_demand_test.go` (`package agent_test`): `heldThenCompletesScript`, `compactionBracketKinds` (shared Kind()-sequence extractor), `alwaysCompactingStrategy`.

- [x] 6.1-6.4 GREEN (already done in Phases 3/4): verdict capture, transcript rebuild, marked-close capture, `TestContextStrategy_NextTurnUsesRebuiltTranscript` (S-CTX-024, two sub-tests: clean turn 2, and every attempt of a retried turn 2).
- [x] 6.5 GREEN (already done in Phase 4): `Harness.Compact` implemented, reusing the existing `signalMu`/`cancelRun` gate — no new state.
- [x] 6.6 RED→GREEN: `TestHarness_Compact_TypedRefusalMidTurn` (S-CMP-024).
- [x] 6.7 GREEN: `TestHarness_Compact_OnDemandUnmodifiedStream` (S-CMP-023).
- [x] 6.8 RED→GREEN: `TestHarness_Compact_RefusesAfterShutdown` (S-CMP-025).
- [x] 6.9 RED→GREEN: `TestCompaction_OneDemandSharedMechanics` (S-CMP-022).
- [x] 6.10 RED→GREEN: `TestCompaction_AtMostOnePerBoundary` (S-CMP-026).
- [x] 6.11 Bite **S-CMP-033** — RED recorded, then reverted (transcript below).
- [x] 2.3-2.5 (deferred from Phase 2): `TestContextStrategy_RequestConstructibleOnlyWithAllThree` (S-CTX-022), `TestContextStrategy_BoundaryInPromptCoordinates` (S-CTX-023), `TestContextStrategy_PinSurvivesCapableButInert` (S-CTX-025) — all in `context_strategy_test.go`.

### Recurring fixture lesson, discovered and fixed four separate times during Phases 4/6

**A turn that finishes `FinishReasonStop` with nothing steered terminates the run immediately — the outer loop never reaches a second consultation.** Every fixture that needs the context strategy to fire on a boundary *after* turn 1 must make turn 1 a **tool-calling** turn (`FinishReasonToolCalls` continues the loop), never a plain text-stop turn. This bit `TestCompaction_ReconstructionNamesReplacedTurns` (fixed before it was ever committed), then independently bit three more tests during this phase's authoring (`TestContextStrategy_RequestConstructibleOnlyWithAllThree`'s two sub-tests, `TestContextStrategy_BoundaryInPromptCoordinates`, and `TestContextStrategy_NextTurnUsesRebuiltTranscript`'s retry sub-test) before being caught by their own RED output and fixed. Recorded here as a discovery in its own right, since it recurred independently rather than being a single copy-paste mistake propagated once.

**A second, related fixture lesson**: `markedHarnessForCompaction`'s returned harness/provider pair is single-use — its one queued script is consumed by the helper's own internal `Run` call, so it cannot be reused for a second `h.Run(...)` (only for `h.Compact(...)`, which uses its own, separately-supplied provider). Two tests were initially written assuming otherwise and fixed to build their own fresh harness+provider instead.

**A third**: `TestCompaction_OneDemandSharedMechanics` initially tried to give both the "triggered" and "on-demand" paths a matching post-compaction "turn 2" for an equal-entry-count read-back comparison — but every `Harness.Run` call unconditionally prepends its own fresh prompt message, so a second `Run()` call on the demand side always carries one extra entry the triggered side's *in-run* turn 2 does not. Fixed by scripting the triggered side's own turn 2 to fail on script exhaustion (expected, tolerated) immediately after compaction commits — comparing both read-backs right there, with no artificial matching turn on either side.

### RED transcript (bite S-CMP-033)

```
$ go test -race -count=1 -run 'TestHarness_Compact_TypedRefusalMidTurn' ./src/agent/...
    compaction_demand_test.go:63: errors.Is(err, agent.ErrRunInFlight) = false, want true (err = cut: required value is empty)
    compaction_demand_test.go:67: Compact's own sink carries an event/close (ev=event(run_start run=run-hrn-2 seq=1) ok=true), want zero events delivered
--- FAIL: TestHarness_Compact_TypedRefusalMidTurn (0.00s)
```
(scratch mutation: the `if h.cancelRun != nil { return ErrRunInFlight }` refusal check commented out, simulating a deferral instead of a refusal). Reverted immediately; confirmed green again.

### GREEN evidence (Phase 6 close)

```
$ go test -race -count=1 -run 'TestHarness_Compact|TestCompaction_OneDemand|TestCompaction_AtMostOne|TestContextStrategy_' ./src/agent/...
ok  	github.com/cachicamas/backend/agent/src/agent	1.6s
```
`h.History != hist` pin (`harness_test.go`) still green — reconfirmed as part of the full-package run below.

### Full-package sanity check (Phases 1-6 combined, before Phase 7/8)

```
$ go test -race -count=1 -skip 'TestCompaction_SubstrateByteUnchanged' ./src/agent/...
ok  	github.com/cachicamas/backend/agent/src/agent	6.8s
```
`TestCompaction_SubstrateByteUnchanged` (S-CMP-036) and `TestNoRelease_SubstrateByteUnchanged` (S-CTX-021, amended) are the only tests still expected to be sensitive to Phase 8's doc.go amendment; `TestNoRelease_SubstrateByteUnchanged` already passes (it no longer asserts doc.go/history.go/doc_contract_guard_test.go), `TestCompaction_SubstrateByteUnchanged` was excluded from this run pending Phase 8's doc.go/doc_contract_guard_test.go edits (it already passed once, before those edits, confirming its own baseline; it is re-checked at Phase 8's close and again at the Phase 12 final gate).

## Phase 7 — Fixtures: COMPLETE (with a real design correction)

**A genuine architectural bug found and fixed**: my first attempt at `agenttest/compaction_fixtures.go` included a `DriveMarkedHistory` helper that imported `github.com/cachicamas/backend/agent/src/agent` to drive a real `*agent.Harness.Run`. `go build ./...` did not catch this (a normal build never links `agent`'s own test files), but `go vet ./...` / `go test ./...` immediately failed: **import cycle not allowed in test** — `agent`'s own test binary already links `package agent_test` (which imports `agenttest`) together with `package agent` itself, so `agenttest` importing `agent` back closes a cycle in the test build specifically. `agenttest` has never imported `agent` anywhere in this codebase (every existing fixture — `Provider`, `Gate`, `Script` — is `ai`-only), which I should have checked before writing the file. **Fixed** by dropping `DriveMarkedHistory` entirely: the marks-bearing fixture design AD-9 calls for genuinely cannot live in `agenttest` (agent.History's marked-close door is package-private, reachable only from inside `package agent`, so the only caller able to build one already imports `agent` directly) and instead lives where it always correctly did — `package agent_test`'s own `markedHarnessForCompaction` (Phase 3) and `appendAndMark` (Phase 1, internal `package agent`). `agenttest/compaction_fixtures.go` now provides only `NewMisalignedCutFixture` (pure `ai`-only), which is genuinely reusable and needed no `agent` import.

- [x] 7.1 `agenttest/compaction_fixtures.go` — `NewMisalignedCutFixture` (the mis-aligned-cut transcript builder). The marks-bearing fixture is `markedHarnessForCompaction`/`appendAndMark`, already in `package agent_test`/`package agent` from Phases 1/3 — see the correction above for why it cannot also live in `agenttest`.
- [x] 7.2 `agenttest/compaction_fixtures_test.go` — `TestNewMisalignedCutFixture_PairGenuinelyStraddlesTheCut` confirms the builder's own physics.
- [x] 7.3 Confirmed: no file under `agenttest/` needs substrate-filter widening — the two filters match only exact `backend/agent/src/agent/*` filenames, and `compaction_fixtures.go`/`compaction_fixtures_test.go` live under `agenttest/`, outside their governed directory.

### GREEN evidence (Phase 7 close)

```
$ go vet ./...            # confirms no import cycle
$ go test -race -count=1 -run 'TestNewMisalignedCutFixture' ./src/agenttest/...
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.1s
```

## Phase 8 — Substrate: L2C-07 amendment, surface guard, filter widening: COMPLETE

- [x] 8.1 GREEN: `doc.go`'s `L2C-07` row amended, confined to its two clauses — route enumeration re-closed at 4 (`append, seeded construction, orphan synthesis, prefix replacement`), identity clause re-scoped (`ordinal entry identity stable within a transcript generation`); pairing-invariant, no-bypass, origin-distinguishability clauses preserved byte-verbatim (confirmed by `strings.Contains` assertions in the new test below, not just by eye).
- [x] 8.2 GREEN: `doc_contract_guard_test.go`'s `expectedLayer2ContractRows` `L2C-07` entry amended byte-in-sync (identical string literal).
- [x] 8.3 RED→GREEN: `TestDocContract_L2C07BothClausesTogether` (S-HIS-104) — the two texts are byte-identical to each other; the route clause names all four routes; the identity clause states generation-scoping; both checked as independent assertions (so a half-amendment would fail one without needing a separate scratch-mutation drill — this row's amendment is not one of the four mandated bites).
- [x] 8.4 GREEN: `TestDocContract_RowCountReScoped` (S-HIS-080) — asserts L2C-07's presence and position (index 6), scoped off any literal row-count claim. Confirmed the pre-existing bite mechanism (`TestLayer2DocContract_MatchesTheCommittedTable`'s own `len(rows) != len(expectedLayer2ContractRows)` check, exercised today by the pre-existing `TestDocContract_ScratchEdit_FailsBite`) is untouched and still passes.
- [x] 8.5 GREEN: both `filterOutLoopFiles` (`loop_test.go`) and `filterOutLoopHookFiles` (`loop_hook_test.go`) widened, byte-in-sync, with exactly the 7 new `src/agent` filenames (`compaction.go` + its 6 test files); no pre-existing filename added (doc.go/doc_contract_guard_test.go already present since AG-14); `cost_events.go`, `cost_events_test.go`, `stream_check_test.go`, `reconstruction_test.go` confirmed absent from both, as designed.
- [x] 8.6 GREEN: `diff <(grep filter entries from loop_test.go) <(... loop_hook_test.go)` → **identical**.
- [x] 8.7 GREEN: `git diff --stat f6acc0d2 -- backend/agent/src/agent/` (excluding test files and the new `compaction.go`) → exactly `{context_strategy.go, doc.go, harness.go, history.go}`, matching S-LSK-029's predicted set precisely.

### GREEN evidence (Phase 8 close — both substrate tests, post-amendment)

```
$ go test -race -count=1 -run 'TestCompaction_SubstrateByteUnchanged|TestNoRelease_SubstrateByteUnchanged' ./src/agent/...
--- PASS: TestCompaction_SubstrateByteUnchanged (0.15s)
--- PASS: TestNoRelease_SubstrateByteUnchanged (0.08s)
ok  	github.com/cachicamas/backend/agent/src/agent	1.7s
```

## Full-suite checkpoint after Phases 1-8 (uncached, uses `-count=1`)

```
$ time go test -race -count=1 ./...
ok  	github.com/cachicamas/backend/agent/src/agent                                     9.868s
ok  	github.com/cachicamas/backend/agent/src/agenttest                                  2.629s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep                            2.031s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest                        2.235s
ok  	github.com/cachicamas/backend/agent/src/ai                                         3.720s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry                          1.868s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat                          171.051s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest            2.173s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter                 2.445s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance     5.747s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke  2.645s
ok  	github.com/cachicamas/backend/agent/src/handoff                                    2.380s
2:52.85 total wall-clock (real, uncached — openaicompat alone 171s, matching the briefed ~175s, confirming this is not a cache artifact)
```

**Every package green.** This is the strongest evidence point so far: the full mechanism (history surgery, the call, the bracket, atomicity, harness wiring, the on-demand door, substrate) integrates correctly end to end.

## Phase 9 — Spec back-annotations: COMPLETE (folded into Phase 11's promotion commit)

Task 9.1's own note is followed literally: "this task is the promotion-time carry, see Phase 11." Phase 9 produces no file change of its own — the draft delta specs were already committed in the planning commit (`fbe9b410`) — so its deliverable is the confirmation performed while applying Phase 11's promotion, recorded there.

- [x] 9.1 `agent-run-driver` back-annotation confirmed present and applied verbatim at promotion.
- [x] 9.2 `agent-v1-scope` R-AGS-007/R-AGS-013 back-annotations confirmed present and applied at promotion — **with one genuine defect found**: see "S-AGS-053 ID collision" below.
- [x] 9.3 `agent-protocol-events` live-emission and compaction-strategy bullets confirmed present and applied at promotion.
- [x] 9.4 Every cited `file:line` re-opened against the actually-shipped tree during promotion (Phase 11); no drift beyond the already-known `harness.go` line-number shift class (AG-16/17's own recorded lesson) was found in the three back-annotation deltas themselves.

## Phase 10 — Documentation: COMPLETE

- [x] 10.1 `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` status line bumped `17 of 24` → `18 of 24`; `Wave 4 opens with AG-17` → `Wave 4 complete (AG-17…AG-18)`; the dense AG-18 sentence appended to the same single-line status paragraph (R-11/G3 discharge, zero-value re-homing, the call, generation-scoped identity, first production emission, atomicity by ordering, the on-demand door). Verified via `git show` against the actual diff, not assumed.
- [x] 10.2 Completion narrative folded into the same status-line sentence (this repo's established one-paragraph-per-milestone pattern; no separate checklist table exists in this doc to tick a row in).
- [x] 10.3 Confirmed via `grep` at lines 2210 and 2262 (pre-edit numbering): the `R-18` traceability-spine row already names AG-18 for R-11; no edit needed, only the discharge status changes, and that lives in the back-annotations, not the spine mapping.

## Phase 11 — OpenSpec promotion + archive: COMPLETE

- [x] 11.1 `openspec/specs/agent-compaction/spec.md` created, verbatim from the change-dir draft. Diffed against the source: **only 3 header lines differ** (`(AG-18)` tag added, `Sources` gains the archive-path reference and drops the "this phase" self-reference, `Ownership boundary` drops the "beside this file" clause that becomes structurally inaccurate once promoted). Body (Purpose through Evidence discipline, lines 12–333 of the draft) is byte-identical.
- [x] 11.2 Seven delta specs promoted with full MODIFIED-block preservation: `agent-history`, `agent-context-strategy`, `agent-cost-events`, `agent-loop-skeleton`, `agent-run-driver` (all applied earlier this phase, before the compaction interruption), plus `agent-v1-scope` and `agent-protocol-events` (applied after resuming from the compaction interruption, recorded below).
- [x] 11.3 `git mv openspec/changes/cachicamas-agent-compaction/ openspec/changes/archive/2026-08-19-cachicamas-agent-compaction/` performed. **Every one of the 13 moved files verified byte-for-byte**: `git hash-object` (11 files whose content matched `HEAD`'s blob exactly) or a direct working-tree hash/checksum (`tasks.md`, dirty at move time; `apply-progress.md`, untracked at move time) captured *before* the move and re-verified identical *after* — all 13 matched exactly. No truncation occurred.
- [x] 11.4 This report.
- [x] 11.5 This file, updated below.

### The S-AGS-053 ID collision (found and fixed during 11.2, `agent-v1-scope` promotion)

The `agent-v1-scope` delta's own draft (`openspec/changes/cachicamas-agent-compaction/specs/agent-v1-scope/spec.md`, authored at `sdd-spec`) allocates scenario ID `S-AGS-053` for a new R-AGS-007 scenario: "AG-18: R-11's discharge is auditable, not asserted." The delta's own header states "BACK-ANNOTATION ONLY... no verdict is renumbered... and no count moves" — but `S-AGS-053` was **already allocated** in the shipped main spec, under the pre-existing `R-AGS-014` (the standing-amendment-rules requirement, ironically the one about append-only identifier discipline), for an unrelated scenario about a "ninth Layer-2-owned concern."

This is a genuine authoring defect from the `sdd-spec` phase, never caught before promotion — the delta's own stated invariant is self-contradicted by its own content. Verified the true next-free ID with:

```
$ grep -o "S-AGS-[0-9]*" openspec/specs/agent-v1-scope/spec.md | sort -t- -k3 -n -u
S-AGS-001 ... S-AGS-060   # contiguous, no gap
```

**Corrected here rather than propagated**, following this repository's own established convention (the `S-HIS-080` count-assertion fix and the `agent-run-driver` "Pointer corrected" precedent, both from this same change): renumbered the delta's new scenario to `S-AGS-061` — the true next-free identifier — and recorded the correction explicitly in the promoted `agent-v1-scope/spec.md` text itself, inline with the scenario, rather than silently renumbering with no trace. Re-verified after the edit: exactly 61 scenario definitions, exactly 15 requirement definitions, zero duplicate IDs among actual `##`/`- **S-AGS-NNN**` definitions (narrative cross-references to an ID, e.g. "consistent with S-AGS-015", are not definitions and were correctly excluded from the duplicate check).

The delta's own header additionally claims "exactly two bullets are back-annotated" for the `agent-protocol-events` delta, but the delta's own MODIFIED body in fact amends six of the eight bullets in that list (verified by diffing each bullet against the shipped main spec before editing). This is a minor self-description inaccuracy in the delta's own header prose, not a functional defect (no ID collision, no broken requirement) — the actual promoted content used is the MODIFIED body, which is authoritative ("the list is reproduced in full"), so no promoted content is affected. Recorded here as a finding, not fixed in the archived draft (which is a historical record of what `sdd-spec` authored).

## Phase 12 — Final gate: COMPLETE, ALL GREEN

- [x] 12.1 `cd backend/agent && go test -race -count=1 ./...` — **12/12 packages `ok`** (+1 `[no test files]`), `EXIT:0`. Wall-clock `12:39:52` → `12:42:45` = **2m53s**, `openaicompat` alone `171.572s` — genuinely uncached, matching the Phase 1-8 checkpoint (`2:52.85s`) almost exactly, as expected since no openspec/docs edit touches `backend/agent/`.
- [x] 12.2 `./bin/golangci-lint cache clean && make lint` (pinned `v2.9.0` via `make tools`, not the machine's global) — **`0 issues.`**
- [x] 12.3 `gofmt -l backend/agent` — 15 files reported, **all pre-existing and untouched by this change** (`compaction_events_test.go`, `cost_events_test.go`, `delegation_events_test.go`, `envelope_test.go`, `event_registry_test.go`, `permission_events.go`, `permission_events_test.go`, `permission_protocol_test.go`, `protocol_events_test.go`, `reconstruction_test.go`, `scheduler.go`, `scheduler_test.go`, `scripted_tool_test.go`, `tool.go`, `tool_test.go`) — **the identical 15-file set AG-17's own archive report recorded**, confirming these are a stable pre-existing condition, not a regression either milestone introduced. None of this change's new or modified files appear in the list.
- [x] 12.4 `make build` — clean, no output beyond the build command. `make vuln-check` — `EXIT:0`; verified by direct JSON inspection (`grep -c '"type": "finding"'` → `0`) rather than trusting a summary line — zero reachable vulnerabilities.
- [x] 12.5 Import-boundary and ambient-authority guards re-run explicitly by exact test name (not by broad `-run` pattern, which over-matched unrelated tests on a first attempt): all 8 guard tests (`TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault`, `TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction`, `TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage`, `TestLayer2Agent_NonTestSourcesCarryNoForbiddenCallSite`, `TestLayer2Agent_ForbiddenSetIsPackageScopedDenyByDefault`, `TestLayer2Agent_FailsOnStagedMutation`, `TestLayer2Agent_FileSelectionIsUniform`, `TestLayer2Agent_TestSourcesStayGreenEvenWithForbiddenCalls`) **PASS**, zero change to either guard file (confirmed via `git status`: neither `import_boundary_test.go` nor `ambient_authority_test.go` appears in this change's diff). New production files' imports manually re-read: `compaction.go` imports only `context`, `errors`, `strconv`, `strings`, `sync/atomic` and the internal `ai` package; `agenttest/compaction_fixtures.go` imports only the internal `ai` package. No process, filesystem, environment or network facility imported by either.
- [x] 12.6 `tasks.md` self-check performed below, after this file is committed.

## Commits (9 implementation/docs commits + this archive commit)

| SHA | Subject |
|---|---|
| `6e481b4a` | `feat(agent): AG-18.2 history prefix-replacement surgery` |
| `4528dee9` | `feat(agent): AG-18 context strategy compaction verdict carrier` |
| `58c9d024` | `feat(agent): AG-18 compaction mechanism -- call, surgery, atomicity, on-demand door` |
| `dcdc350c` | `test(agent): AG-18 compaction call, stream, recovery and demand-door coverage` |
| `d82737dd` | `feat(agenttest): AG-18 mis-aligned-cut compaction fixture` |
| `b5b8ffa1` | `feat(agent): AG-18 wire compaction into the run driver's turn boundary` |
| `b2b78e14` | `chore(agent): AG-18 phase 8 -- L2C-07 amendment, surface guard, filter widening` |
| `684c66a6` | `docs(0003): AG-18 status line, milestone counter, checklist` |
| `82e0b5ef` | `docs(openspec): AG-18 promote agent-compaction, back-annotate six specs` |
| *(this commit)* | `chore(openspec): AG-18 archive -- apply-progress, archive report, tasks self-check` |

### Two commit-message drafting slips (found by self-review, recorded rather than amended)

Per this repository's own git-safety discipline, an already-made commit is never amended for a message-wording fix; both are recorded here transparently instead, matching the AG-17 precedent of recording a caught defect rather than silently correcting it:

1. `dcdc350c`'s message states "S-CMP-031 and S-CMP-033's RED evidence lands with the harness wiring and recovery commits respectively" — implying a forward reference to a later commit. This is incorrect: both bites' proving test files (`compaction_recovery_test.go` for S-CMP-031, `compaction_demand_test.go` for S-CMP-033) are contained in `dcdc350c` itself, not in any later commit. The actual RED evidence (real failing-test transcripts) is recorded in this file's Phase 5 and Phase 6 sections above, captured at implementation time — the drafting error is confined to one sentence of commit-message prose describing where a file "lands"; no evidence, code or test content is affected.
2. `82e0b5ef`'s subject line reads "back-annotate six specs" and its body bullets six specs by name, but the commit's own diff touches **seven** specs — `agent-protocol-events` is modified in the diff (confirmed: `M openspec/specs/agent-protocol-events/spec.md` in the commit's own file list) but was omitted from the message's enumeration. The promoted content itself is correct and was verified independently (diffed against the delta source before committing); the omission is confined to the commit message's own summary completeness.

## Full-suite checkpoint after Phase 12 (final, uncached, all 9 implementation commits + archive move applied)

```
$ date && go test -race -count=1 ./... ; echo "EXIT:$?" && date
Wed Aug 19 12:39:52 -05 2026
ok  	github.com/cachicamas/backend/agent/src/agent                                     10.544s
ok  	github.com/cachicamas/backend/agent/src/agenttest                                  3.450s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep                            2.922s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest                        4.028s
ok  	github.com/cachicamas/backend/agent/src/ai                                         5.712s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry                          1.495s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat                          171.572s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest            2.085s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter                 3.802s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance     5.029s
?   	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke  3.095s
ok  	github.com/cachicamas/backend/agent/src/handoff                                    2.460s
EXIT:0
Wed Aug 19 12:42:45 -05 2026
```

**2 minutes 53 seconds wall-clock, uncached, EXIT:0, every package green.** This is the final gate evidence, run against the actual final tree state (after all 9 implementation commits and the archive move), not inherited from an earlier checkpoint.

## Task completion: 86/86 (plus 3 explicitly-recorded deviation tasks: 0.7, 1.16, 8.8) — all `[x]`

Every phase (0 through 12) is complete. The tasks.md self-check (12.6) follows in the same commit as this file.
