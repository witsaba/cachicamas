# Tasks: AG-18 — Implement compaction

> Change: `cachicamas-agent-compaction`. Binding inputs: `proposal.md` (Deliverable 0 A1–A8), `design.md` (AD-1…AD-12, binding, Approach steps 1–9, relaxation ledger), eight spec files under `specs/` (`agent-compaction` NEW, R-CMP-001…014, S-CMP-001…037, 4 bites; `agent-history`, `agent-context-strategy`, `agent-cost-events`, `agent-loop-skeleton`, `agent-run-driver`, `agent-v1-scope`, `agent-protocol-events` MODIFIED — the last two BACK-ANNOTATION ONLY). Strict TDD: every behavior task is RED-first, `cd backend/agent && go test -race -count=1 ./...`, RED output and wall-clock duration recorded in `apply-progress.md` before GREEN. Ships in **one PR**: implementation, doc 0003 update, and the OpenSpec archive together.

## Three hard prerequisites — resolved here, before any edit

1. **Substrate-filter files, located by grep in this tree (not assumed).** `filterOutLoopFiles` is in `backend/agent/src/agent/loop_test.go` (func at `:837`); `filterOutLoopHookFiles` is in `backend/agent/src/agent/loop_hook_test.go` (func at `:907`). Both are OR-chains of `strings.HasSuffix(path, "/<exact-file>")`. Their current tail (AG-17's widening) ends at `loop_test.go:994-997` (`/context_strategy.go`, `/token_accounting.go`, `/context_strategy_test.go`, `/token_accounting_test.go`) and the byte-in-sync `loop_hook_test.go:1048-1051`. AG-18 appends immediately after these — see Phase 8.
2. **`CompactionRequest.Options` concrete type.** `TurnOptions` (`loop.go:63-141`) carries `Model string`, `MaxTokens int`, `PreRequestHook func(...)`, `Tools Registry`, `PermissionPolicy PermissionPolicy`, `Continuation *TurnContinuation`. **Decision**: reuse `TurnOptions` unchanged as `CompactionRequest.Options TurnOptions` — no narrower type is introduced. `Tools` and `Continuation` MUST be asserted zero/nil on every compaction request (R-CMP-002's "no tool-continuation path"); this is a runtime assertion in `compaction.go`, not a type-level restriction, matching AD-7's "finalize the exact type after reading `TurnOptions`."
3. **`wantKinds` re-verification (R-CMP-014).** `harness_test.go` contains exactly one length-equality kind assertion (`len(kinds) != len(wantKinds)`, line 152) and one further `wantKinds`-driven block (line 194). Re-opened: both belong to `TestSchedule_LeaveSinkOpenZeroDefault_ClosesSinkUnchanged` (`:119-160`) and `TestSchedule_LeaveSinkOpenSet_CallerOwnsClose` (`:168-252`), which call `sched.Schedule(...)` **directly** — not `Harness.Run`. `grep -c ContextStrategy harness_test.go` returns **0**: no test in this file ever installs a strategy at all (AG-17's seam tests live in `context_strategy_test.go`). Conclusion, stated precisely rather than copied: these two blocks are not merely "nil-strategy" cases, they are **outside the Harness/ContextStrategy call path entirely** — `Scheduler.Schedule` takes no `ContextStrategy` argument, so compaction cannot reach them on any path, a stronger guarantee than the design's phrasing implies. No other `[]agent.EventKind{...}` closed-sequence literal exists in `harness_test.go` (grep confirmed); the only other length check in the file (`:1657`, `messagesEqual`) is a generic helper, not a kind sequence. **Verdict: R-CMP-014 is safe to build on; no task in this plan may cite "nil strategy" for these two lines — cite unreachability instead.**

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (counted, excludes `openspec/`) | 1585–2695 |
| Estimated changed lines (full diff, attempt-budget relevant) | 2885–4795 |
| 400-line budget risk | High |
| Chained PRs recommended | No — `size:exception` pre-accepted, 1000-line nominal bar, counted excluding `openspec/` |
| Suggested split (reserve only, per `NFR-CMP-005`) | U1 (cut resolution + `ReplacePrefix` route + `agent-history` delta, no provider call, no events) → U2 (the call + atomicity) → U3 (stream record + on-demand door) |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| U1 | Cut resolution (AD-1) + `EntryOriginSummarized` (AD-2) + `ReplacePrefix` route + turn marks (AD-6, AD-9), no provider call, no events | PR 1 (single-pr) | `go test -race -count=1 -run 'TestHistory_ReplacePrefix|TestHistory_TurnMark|TestResolveCut'` | N/A — pure library surgery, no run driven | `history.go`'s new members, `compaction.go`'s cut-resolution helper |
| U2 | `runCompaction`'s call, bracket, spend fold, atomicity (AD-3, AD-5, AD-11) | PR 1 | `go test -race -count=1 -run 'TestCompaction_OwnCall|TestCompaction_Recovery'` | `go test -race -count=1 ./...` full suite | `compaction.go`'s call/bracket/commit sequence |
| U3 | Harness wiring, on-demand door, reconstruction, substrate + docs (AD-4, AD-7, AD-10, AD-12) | PR 1 | `go test -race -count=1 -run 'TestHarness_Compact|TestCompaction_Reconstruction'` | `go test -race -count=1 ./...` full suite | `harness.go`'s verdict-capture block, `Harness.Compact` |

## Files touched

| File | Action | Forbidden-list release? |
|---|---|---|
| `backend/agent/src/agent/compaction.go` | **Create** | n/a (new file) |
| `backend/agent/src/agent/compaction_call_test.go` | **Create** | n/a |
| `backend/agent/src/agent/compaction_surgery_test.go` | **Create** | n/a |
| `backend/agent/src/agent/compaction_stream_test.go` | **Create** | n/a |
| `backend/agent/src/agent/compaction_reconstruction_test.go` | **Create** | n/a |
| `backend/agent/src/agent/compaction_recovery_test.go` | **Create** | n/a |
| `backend/agent/src/agent/compaction_demand_test.go` | **Create** | n/a |
| `backend/agent/src/agenttest/compaction_fixtures.go` | **Create** | n/a — not governed by the two filters |
| `backend/agent/src/agenttest/compaction_fixtures_test.go` | **Create** | n/a |
| `backend/agent/src/agent/history.go` | Modify | Not on the forbidden list — free to edit |
| `backend/agent/src/agent/history_test.go` | Modify | Not on the forbidden list |
| `backend/agent/src/agent/history_surface_guard_test.go` | Modify | Not on the forbidden list |
| `backend/agent/src/agent/context_strategy.go` | Modify | Not on the forbidden list |
| `backend/agent/src/agent/context_strategy_test.go` | Modify | Not on the forbidden list |
| `backend/agent/src/agent/harness.go` | Modify | Not on the forbidden list |
| `backend/agent/src/agent/doc.go` | Modify | **YES — released for AG-18 only (AD-12), confined to the `L2C-07` row's two clauses** |
| `backend/agent/src/agent/doc_contract_guard_test.go` | Modify | **YES — released for AG-18 only (AD-12), byte-in-sync with `doc.go`** |
| `backend/agent/src/agent/loop_test.go` (`filterOutLoopFiles`) | Modify | n/a — filter host, not itself forbidden |
| `backend/agent/src/agent/loop_hook_test.go` (`filterOutLoopHookFiles`) | Modify | n/a — filter host, byte-in-sync |
| `docs/architecture/milestones/0003-…md` | Modify | n/a |
| `openspec/specs/{agent-compaction NEW, agent-history, agent-context-strategy, agent-cost-events, agent-loop-skeleton, agent-run-driver, agent-v1-scope, agent-protocol-events}/spec.md` | Create/Modify (promotion) | n/a |
| `reconstruction_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `compaction_events.go`, `cost_events.go`, `cost_events_test.go`, `cost_usage.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `go.mod`, `go.sum`, all of `src/ai/**` | **NOT touched** | Forbidden and untouched — byte-unchanged, verified at Phase 12 |

## Phase 0 — Foundation: RED baseline & prerequisite verification

- [ ] 0.1 Record the three hard prerequisites above verbatim in `apply-progress.md`, with the exact grep commands used.
- [ ] 0.2 Grep `backend/` for `CompactionRequest`, `ReplacePrefix`, `EntryOriginSummarized`, `closeTurnMarked`, `runCompaction`, `Harness.Compact` — confirm **zero** hits, the pre-existing state the spec pins.
- [ ] 0.3 Re-open `harness.go:524-530` (verdict discarded) and `:668` (`hist.CloseTurn()`); confirm content matches design's citation, not just the line numbers (AG-16/17 lesson: this file shifts every milestone).
- [ ] 0.4 Re-open `history.go:327-332` (`commit` — "the ONLY function that writes h.entries or h.open"), confirm the exported surface is exactly `Append`, `CloseTurn`, `SynthesizeOrphans`, `Entries`, `Len` (5 members) before the new route becomes the 6th.
- [ ] 0.5 Re-open `doc_contract_guard_test.go:62-71` — confirm exactly 8 rows (`L2C-01`…`L2C-08`) and the `L2C-07` row text verbatim, before amendment.
- [ ] 0.6 Re-open `stream_check.go:161` (`if d.Placement == PlacementTurn && !turnOpen`) and `turn_events.go:42`/`:178` (`NewTurnStart`/`NewTurnEnd`) and `cost_usage.go:70`/`:118` (`newCostTurnFromUsage`/`costAccumulator.add`) — confirm signatures match design's citations.

## Phase 1 — History surgery foundation (AD-1, AD-2, AD-6, AD-9): pure, no provider, no events

Covers `R-CMP-004`, `R-CMP-005`, `R-CMP-006`, `R-CMP-007`, `R-CMP-012`, and `agent-history`'s `R-HIS-001`/`R-HIS-004`/`R-HIS-005`/`R-HIS-009` deltas.

- [ ] 1.1 GREEN (types, no behavior change): add `EntryOriginSummarized` as the third `EntryOrigin` member in `history.go` beside `EntryOriginAppended`/`EntryOriginSynthesized` (`:81-92`); update `String()`.
- [ ] 1.2 GREEN: add a turn mark record `{TurnID, entry count}` to `History`'s internal state, written only through `commit` (AD-9); add the package-private `closeTurnMarked(TurnID) error` door beside the exported `CloseTurn()`, whose signature and semantics MUST NOT change.
- [ ] 1.3 RED: `compaction_surgery_test.go` — `TestResolveCut_RetractsToMarkBoundary` (`S-CMP-010`): naive cut splitting a call/result pair retracts to ≤ the pair's start; both halves land in the protected tail. Record RED (undefined symbol).
- [ ] 1.4 GREEN: implement the pure cut-resolution helper in `compaction.go` (AD-1) — retract to the greatest recorded turn-mark boundary ≤ the naive cut, verify pairing-closure read-only over `Entries()` via `ai.ToolCall.ID()`/`ai.ToolResult.CallID()` correlation (the `resolveOpenSet`, `history.go:408-432`, precedent), iterate on a straddling pair, terminate at 0 with typed failure.
- [ ] 1.5 GREEN: `TestResolveCut_SeededValidation` (`S-CMP-011`), `TestResolveCut_MonotoneNeverExpandsForward` (`S-CMP-012`), `TestResolveCut_TerminatesOnAdversarialInput` (`S-CMP-013`).
- [ ] 1.6 Bite **`S-CMP-030`** (RED-first, then revert): on a scratch tree, skip mark-alignment/pairing verification and use the naive cut directly; re-run 1.3; confirm it FAILS reporting a split pair. Record RED, revert.
- [ ] 1.7 RED: `history_test.go` — `TestHistory_ReplacePrefix_SucceedsAndRenumbers` (`S-HIS-100`, `S-CMP-007`): commit succeeds → read-back is `[summary, tail...]`, count `1+(len(pre)-r)`; a fresh seeded history over the read-back succeeds. Record RED (undefined symbol).
- [ ] 1.8 GREEN: implement `History.ReplacePrefix(count int, summary ai.Message) error` as a new `commitOp` dispatched from `commit` (`history.go:332`) per AD-6: validate `count` against a mark boundary and pairing-closure; reject a summary carrying tool parts (`S-CMP-027`); rebuild `[summary, tail...]` with `EntryID`s renumbered ordinally, origins preserved, `EntryOriginSummarized` for the summary; rebuild marks (in-span dropped, tail shifted); on any validation error leave `h.entries` untouched (`R-HIS-002` carried forward).
- [ ] 1.9 GREEN: `TestHistory_ReplacePrefix_NothingElseRemovable` (`S-HIS-101`), `TestHistory_ReplacePrefix_RejectsToolBearingSummary` (`S-CMP-027`).
- [ ] 1.10 RED→GREEN: `compaction_surgery_test.go` — `TestCompaction_ProtectionValueIdentical` (`S-CMP-014`, `S-HIS-103`): fixture with replaced prefix ≥ 2 and protected tail ≥ 2; assert `pre.Message().Equal(post.Message())` and `pre.Origin() == post.Origin()` positionally. **`reflect.DeepEqual` over whole `Entry` values is forbidden in this assertion (R-CMP-006, AD-8) — grep the new test files for it and fail review if present.**
- [ ] 1.11 RED→GREEN: `TestCompaction_IdentityRenumbers` (`S-CMP-015`, part of `S-HIS-103`): at least one protected entry's `ID()` differs pre/post; post-compaction identities are exactly the new ordinal sequence.
- [ ] 1.12 RED→GREEN: `TestCompaction_SummaryTypedByOriginOnly` (`S-CMP-016`): summary vs. a byte-identical-content ordinarily-appended control message, distinguished by `Origin()` alone, content and `Failed()` never read. `TestEntryEnvelope_NoExportedOriginSetter` (`S-CMP-028`).
- [ ] 1.13 GREEN: `history_surface_guard_test.go` — extend `S-HIS-030`'s closed enumeration to name `ReplacePrefix` as the 6th exported member, in the **same commit** as the route (task 1.8).
- [ ] 1.14 Bite **`S-HIS-102`** (RED-first, then revert): drop `ReplacePrefix` from the enumeration while the route stays exported; re-run 1.13; confirm it FAILS naming the unenumerated route. Record RED, revert.
- [ ] 1.15 GREEN: `history_surface_guard_test.go` — `TestHistory_MarkedCloseNotExported` (`S-CMP-035`): the exported `CloseTurn()` keeps its pre-AG-18 signature; an unmarked close still succeeds; `closeTurnMarked` is not reachable from `package agent_test`.

**Definition of done (Phase 1)**: `go test -race -count=1 -run 'TestResolveCut|TestHistory_ReplacePrefix|TestCompaction_Protection|TestCompaction_Identity|TestCompaction_SummaryTyped'  ./backend/agent/src/agent/...` green; all four Phase-1 RED transcripts (1.3, 1.6, 1.7, 1.14) recorded in `apply-progress.md`.

## Phase 2 — The verdict carrier (AD-7)

Covers `agent-context-strategy`'s `R-CTX-003`/`R-CTX-004` delta.

- [ ] 2.1 GREEN (types): in `context_strategy.go`, add `CompactionRequest{Provider ai.ModelProvider; Options TurnOptions; Instruction string; Cut int}` and `ContextVerdict.Compaction *CompactionRequest`; `NoOpContextStrategy` MUST keep compiling untouched.
- [ ] 2.2 RED→GREEN: `context_strategy_test.go` — `TestContextVerdict_ZeroRequestsNothing` (`S-CTX-005` revised): the zero verdict's `Compaction == nil`; a zero verdict on every consultation of a multi-turn run emits zero compaction-family events and a byte-identical history read-back vs. the field-nil run.
- [ ] 2.3 RED→GREEN: `TestContextStrategy_RequestConstructibleOnlyWithAllThree` (`S-CTX-022`): a strategy supplying provider+options+instruction compacts; an empty instruction fails typed, issues no provider request, and no Layer 2-authored text appears in any captured request.
- [ ] 2.4 RED→GREEN: `TestContextStrategy_BoundaryInPromptCoordinates` (`S-CTX-023`): the request's `Cut` is an index into the transcript the prompt carried; the entries actually replaced are a prefix of that same transcript ending at or before it.
- [ ] 2.5 RED→GREEN: `TestContextStrategy_PinSurvivesCapableButInert` (`S-CTX-025`): a strategy fully capable of requesting compaction but returning the zero verdict on every consultation produces byte-identical streams/read-backs vs. the nil-field run.
- [ ] 2.6 Confirm the existing bite `S-CTX-008` (emit one `compaction_started` on a scratch tree, re-run `S-CTX-007`) still FAILS the byte-identity comparison unmodified — regression check, no new bite needed.

**Definition of done (Phase 2)**: `go test -race -count=1 -run 'TestContextVerdict|TestContextStrategy_' ./backend/agent/src/agent/...` green; `NoOpContextStrategy` compile-time guard unchanged.

## Phase 3 — The compaction call and its spend (AD-3 call arm, AD-5, AD-9 span, AD-11 ordering half 1)

Covers `R-CMP-002`, `R-CMP-003`, `R-CMP-009` (span half), `R-CMP-012` (marks), and `agent-cost-events`'s `R-CST-001` delta additive row.

- [ ] 3.1 RED: `compaction_call_test.go` — `TestCompaction_OwnCall_DistinctProvider` (`S-CMP-001`): compaction request carrying provider B distinct from `h.Provider` (A); B's request count is exactly 1, A's unchanged. Record RED.
- [ ] 3.2 GREEN: implement `runCompaction` in `compaction.go` — issue exactly one non-tool completion on `req.Provider`/`req.Options`, on the run's `runCtx`; mint a fresh compaction `TurnID` in a namespace disjoint from `mintLoopTurnID`'s (e.g. `turn-cmp-N`).
- [ ] 3.3 RED→GREEN: `TestCompaction_InjectionOnly` (`S-CMP-002`, `S-CTX-022`'s instruction half): captured request carries exactly the injected instruction; no message part carries runtime-authored text, asserted over `Provider.Requests()`.
- [ ] 3.4 RED→GREEN: `TestCompaction_CancellationAbortsWithoutEndingRun` (`S-CMP-003`): Gate-held provider; run cancellation fires mid-call; bracket closes aborted with typed failure; harness survives; no `compaction_finished` anywhere. `TestCompaction_EmptyInstructionFailsTyped` (`S-CMP-005`).
- [ ] 3.5 RED→GREEN: `TestCompaction_SpendFoldedIntoCumulative` (`S-CMP-004`, `S-CST-015`, `S-CST-016`): `cost_turn(Final)` inside the bracket before its `turn_end`, figure-for-figure from the completion's `ai.Usage`; the emission site itself calls `total.add(ct)` **explicitly** — reusing `newCostTurnFromUsage` (`cost_usage.go:70`) and the same run-scoped `costAccumulator.add` (`:118`), never routed through `turnSink` (`harness.go:563/567/590`, which compaction events never traverse); `cost_session` includes it.
- [ ] 3.6 RED→GREEN: `TestCompaction_IffHoldsOnAllThreeArms` (`S-CMP-006`, `S-CST-015` arm table): completion+commit → non-aborted + 1 `cost_turn`; no completion → aborted, typed failure, 0 `cost_turn`; completion+non-`Stop`/rejected → non-aborted + 1 `cost_turn`, `compaction_failed`.
- [ ] 3.7 Bite **`S-CMP-032`** (RED-first, then revert): remove the explicit `total.add(ct)` statement at the emission site on a scratch tree; re-run 3.5; confirm it FAILS short by exactly the compaction's usage. Record RED, revert.

**Definition of done (Phase 3)**: `go test -race -count=1 -run 'TestCompaction_OwnCall|TestCompaction_Injection|TestCompaction_Cancellation|TestCompaction_Spend|TestCompaction_Iff' ./backend/agent/src/agent/...` green; the emission-site fold statement visible in the diff (not asserted, shown).

## Phase 4 — The bracket, the stream record, and reconstruction (AD-3 events, AD-9 span derivation)

Covers `R-CMP-008`, `R-CMP-009`, `R-CMP-012` (S-CMP-029/034), `R-CMP-013`, `R-CMP-014`.

- [ ] 4.1 GREEN: wrap the call of Phase 3 in the dedicated bracket — `NewTurnStart` → `compaction_started` (`NewCompactionStarted`) → (`cost_turn` iff completion exists) → `compaction_finished`|`compaction_failed` → `NewTurnEnd`, all through the **existing** exported constructors; register no new `EventKind`/`TurnOutcome`/`CostLabel`.
- [ ] 4.2 RED→GREEN: `compaction_stream_test.go` — `TestCompaction_StreamAcceptedUnmodified` (`S-CMP-017`): `CheckStream` accepts the recorded stream with `stream_check.go` byte-unchanged; bracket order matches `R-CMP-008`; bracket `TurnID` distinct from every model-turn `TurnID`; every-kind guard passes at its committed count.
- [ ] 4.3 GREEN: implement span derivation (AD-9) — marks fully contained in `[0, cut)` name `StartTurnID`(first)/`EndTurnID`(last) for `CompactionFinished`'s `Span()`; an unmarked-entry span fails typed (`compaction_failed`), no identifier fabricated.
- [ ] 4.4 RED→GREEN: `compaction_reconstruction_test.go` — `TestCompaction_ReconstructionNamesReplacedTurns` (`S-CMP-018`): given only `Span()`+`SummaryID()`, name the earlier turn brackets replaced and locate the summary entry by message ID. `TestCompaction_FinishedXorFailed` (`S-CMP-019`).
- [ ] 4.5 RED→GREEN: `TestCompaction_MarkedSpanFromTwoRuns` (`S-CMP-029`): history driven through two complete runs; span names the first/last marked turn in the prefix, both present earlier on the stream.
- [ ] 4.6 RED→GREEN: `TestCompaction_UnmarkedPrefixFailsTyped` (`S-CMP-034`): a seeded, never-run prefix → `compaction_failed`, no identifier invented, transcript unchanged.
- [ ] 4.7 RED→GREEN: `compaction_stream_test.go` — `TestCompaction_SubstrateByteUnchanged` (`S-CMP-036`): `git diff` over `backend/agent/` at merge base shows every file named byte-unchanged in `R-CMP-013` empty, `src/ai/` diff empty, `go.mod`/`go.sum` diff empty, every-kind guard at committed count.
- [ ] 4.8 RED→GREEN: `TestCompaction_InertUnlessRequested` (`S-CMP-037`): a run with no strategy vs. a run with a strategy that requests no compaction on every consultation → byte-identical streams and read-backs, no compaction-family kind on either.

**Definition of done (Phase 4)**: `go test -race -count=1 -run 'TestCompaction_Stream|TestCompaction_Reconstruction|TestCompaction_Marked|TestCompaction_Unmarked|TestCompaction_Substrate|TestCompaction_Inert' ./backend/agent/src/agent/...` green; `stream_check.go`, `compaction_events.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go` byte-unchanged (`git diff --stat`).

## Phase 5 — Atomicity and recovery (AD-11 complete)

Covers `R-CMP-010`.

- [ ] 5.1 GREEN: order the commit as build-then-commit — no statement touches `h.History`'s pointee before `ReplacePrefix` is called; every failure branch (provider error, cancellation, non-`Stop` finish, pairing rejection) reaches the failed arm before that point.
- [ ] 5.2 RED→GREEN: `compaction_recovery_test.go` — `TestCompaction_AtomicOrAbsent` (`S-CMP-020`): Gate-held-then-failed provider; post-attempt `Entries()` byte-identical to pre-attempt; `compaction_failed` present, `compaction_finished` absent; next turn's request carries the **uncompacted** transcript (proved via `Provider.Requests()`); run reaches ordinary close.
- [ ] 5.3 RED→GREEN: `TestCompaction_WindDownNeverEntered` (`S-CMP-021`): the failed-compaction stream carries no `R-CAN-002` wind-down signature; the run performs at least one further turn.
- [ ] 5.4 Bite **`S-CMP-031`** (RED-first, then revert): move the commit call to before the provider call returns on a scratch tree; re-run 5.2; confirm it FAILS reporting a mutated transcript after a failed compaction. Record RED, revert.
- [ ] 5.5 GREEN: assert (not build) that a run-level cancellation arriving during compaction closes the bracket aborted and leaves the run to wind down at its **existing** iteration-boundary cause check (`harness.go:502`) — reuse 3.4's cancellation fixture, add the wind-down-order assertion.

**Definition of done (Phase 5)**: `go test -race -count=1 -run 'TestCompaction_AtomicOrAbsent|TestCompaction_WindDown' ./backend/agent/src/agent/...` green; bite RED transcript for `S-CMP-031` recorded.

## Phase 6 — Harness wiring and the on-demand door (AD-4, AD-7 harness half, AD-10)

Covers `R-CMP-001`, `R-CMP-011`, and `agent-context-strategy`'s `S-CTX-024` (rebuilt transcript).

- [ ] 6.1 GREEN: at `harness.go:524-530`, capture the verdict (`verdict := h.ContextStrategy.Resolve(...)`) instead of discarding it; on `verdict.Compaction != nil`, invoke `runCompaction` in the same no-turn-open gap, at most once per boundary.
- [ ] 6.2 GREEN: after a successful compaction, rebuild `transcript` from the mutated `hist` **before** entering the attempt loop (`harness.go:562`), so `R-RTY-002`'s by-reference reuse serves the rebuilt slice to every attempt.
- [ ] 6.3 GREEN: at `harness.go:668`, capture the successful attempt's `TurnID` (read after `<-forwarderDone`) and call the package-private `closeTurnMarked(TurnID)` instead of the exported `CloseTurn()` on the run-driven path; `CloseTurn()` itself stays reachable and semantically unchanged for every other caller.
- [ ] 6.4 RED→GREEN: `context_strategy_test.go` — `TestContextStrategy_NextTurnUsesRebuiltTranscript` (`S-CTX-024`): the turn following a compaction sends the post-compaction transcript (summary first, tail after); every attempt of a retried following turn carries that same rebuilt slice.
- [ ] 6.5 GREEN: implement `Harness.Compact(ctx, req CompactionRequest, sink chan<- *Event) error` in `compaction.go` (AD-10) — its own minimal stream: `run_start` → compaction bracket (Phase 4) → `cost_session(Final)` → `run_end`; refuse via the **existing** `h.signalMu`/`h.cancelRun != nil` gate (`harness.go:143-146`, set `:435`, cleared `:450`) — no new state (AD-4, overturns proposal D4).
- [ ] 6.6 RED: `compaction_demand_test.go` — `TestHarness_Compact_TypedRefusalMidTurn` (`S-CMP-024`): a run held mid-turn by a test gate; concurrent `Compact` returns synchronously, typed, `errors.Is`-distinguishable; zero events on the caller's sink; in-flight turn completes unaffected; the run's own stream byte-identical to the no-concurrent-demand run. Record RED.
- [ ] 6.7 GREEN: `TestHarness_Compact_OnDemandUnmodifiedStream` (`S-CMP-023`): no run in flight → `CheckStream` accepts the emitted stream unmodified, one run bracket enclosing one compaction bracket, cumulative cost event before run close.
- [ ] 6.8 RED→GREEN: `TestHarness_Compact_RefusesAfterShutdown` (`S-CMP-025`): on the existing terminal-shutdown route, refuses typed, no event, no mutation.
- [ ] 6.9 RED→GREEN: `TestCompaction_OneDemandSharedMechanics` (`S-CMP-022`): strategy-triggered and on-demand streams' compaction-bracket sub-sequences equal `Kind()`-for-`Kind()`, excluding fresh `RunID`/`TurnID` (reuse `context_strategy_test.go:222-224`'s exclusion helper); read-back entry counts and origin sequences equal.
- [ ] 6.10 RED→GREEN: `TestCompaction_AtMostOnePerBoundary` (`S-CMP-026`): a strategy requesting compaction on every consultation produces exactly one bracket per logical turn, never more.
- [ ] 6.11 Bite **`S-CMP-033`** (RED-first, then revert): on a scratch tree, make the on-demand door enqueue the demand for the next boundary instead of refusing; re-run 6.6; confirm it FAILS (missing refusal or the later compaction the run then performs). Record RED, revert.

**Definition of done (Phase 6)**: `go test -race -count=1 -run 'TestHarness_Compact|TestCompaction_OneDemand|TestCompaction_AtMostOne|TestContextStrategy_NextTurn' ./backend/agent/src/agent/...` green; `h.History != hist` pin (`harness_test.go:1013`) still green untouched.

## Phase 7 — Fixtures

- [ ] 7.1 `agenttest/compaction_fixtures.go` — a mis-aligned-cut transcript builder (a scripted call/result pair straddling a naive cut) and a marks-bearing fixture (transcript driven through real `Harness.Run` turns so marks exist, per AD-9).
- [ ] 7.2 `agenttest/compaction_fixtures_test.go` — fixture physics: the mis-aligned builder's pair genuinely straddles the stated cut; the marks-bearing fixture's mark count equals its driven turn count.
- [ ] 7.3 Confirm no file under `agenttest/` needs substrate-filter widening (the two filters do not govern that directory, per the `agent-loop-skeleton` delta).

## Phase 8 — Substrate: `L2C-07` amendment, surface guard, filter widening

Covers `R-HIS-009` (A4), `R-LSK-004` (A6, AD-12), `NFR-CMP-004`.

- [ ] 8.1 Amend `doc.go`'s `L2C-07` row prose: route enumeration re-closed at **4** naming the prefix replacement; identity clause scoped to "stable within a transcript generation"; pairing-invariant, no-bypass, origin-distinguishability clauses preserved verbatim. Confine the edit to these two clauses (AD-12's release scope).
- [ ] 8.2 Amend `doc_contract_guard_test.go:69`'s `expectedLayer2ContractRows` `L2C-07` entry **byte-in-sync** with 8.1's `doc.go` text.
- [ ] 8.3 RED→GREEN: `TestDocContract_L2C07BothClausesTogether` (`S-HIS-104`): the two texts are byte-identical to each other; both clauses moved; a scratch edit amending only one clause FAILS on the other.
- [ ] 8.4 GREEN: `TestDocContract_RowCountReScoped` (`S-HIS-080`): assert the `L2C-07` row is present and positioned, scoped off the literal row count (correcting the pre-existing "7 rows" drift — 8 rows shipped since AG-14). Confirm existing bite `S-HIS-081` (append a row without its `expectedLayer2ContractRows` entry) still FAILS unmodified.
- [ ] 8.5 Widen `filterOutLoopFiles` (`loop_test.go`, after line 997) and `filterOutLoopHookFiles` (`loop_hook_test.go`, after line 1051), byte-in-sync: append one exact-filename `strings.HasSuffix(path, "/<file>")` entry per new `src/agent` file from Phase 1–6 that the filters' own population rule covers — `compaction.go`, `compaction_call_test.go`, `compaction_surgery_test.go`, `compaction_stream_test.go`, `compaction_reconstruction_test.go`, `compaction_recovery_test.go`, `compaction_demand_test.go`. **No pre-existing filename is added** — `doc.go`/`doc_contract_guard_test.go` are already present from AG-14. `cost_events.go`, `cost_events_test.go`, `stream_check_test.go`, `reconstruction_test.go` stay absent from both (deliberate exclusions).
- [ ] 8.6 Confirm both filters carry an identical suffix-entry set (diff the two lists); no wildcard/prefix/directory pattern; an unnamed new file still fails both guards.
- [ ] 8.7 Re-run `git diff main -- backend/agent/src/agent/` non-test files: confirm the set is exactly `{doc.go, history.go, harness.go, context_strategy.go}` (`S-LSK-029`).

**Definition of done (Phase 8)**: `go test -race -count=1 -run 'TestDocContract|TestLoop_Substrate|TestLayer2DocContract' ./backend/agent/src/agent/...` green; both filter lists diffed and confirmed identical.

## Phase 9 — Spec back-annotations (no Go test enforces these — manual)

- [ ] 9.1 **`agent-run-driver`** — confirm the delta's back-annotation to non-requirements rows `:342` (compaction check split, now fully closed) and `:351` (the forbidden-row breach, recorded rather than silent) is present verbatim in `openspec/changes/cachicamas-agent-compaction/specs/agent-run-driver/spec.md` (already authored in `sdd-spec`; this task is the promotion-time carry, see Phase 11).
- [ ] 9.2 **`agent-v1-scope`** — confirm `R-AGS-007`'s R-11/G3 discharge paragraph and `R-AGS-013`'s AG-18 inheritance-statement discharge are present, mapping every mechanism to its `agent-compaction` requirement.
- [ ] 9.3 **`agent-protocol-events`** — confirm the live-emission bullet (`:154`) and the compaction-strategy bullet (`:158`) read "CLOSED by AG-18" with the three recorded facts (dedicated bracket, byte-unchanged files, exactly-one-of-finished-or-failed).
- [ ] 9.4 For all three: re-read every cited `file:line` against the actually-shipped change (not the delta's own citations, which may have shifted during Phase 1–6) — record any drift found in `apply-progress.md`.

## Phase 10 — Documentation

- [ ] 10.1 `docs/architecture/milestones/0003-…md` status line (`:3`): extend the Wave-4 narrative to close AG-18, naming R-11/G3 discharge; bump the milestone counter from **17 of 24** to **18 of 24**.
- [ ] 10.2 Tick the AG-18 completion-checklist entry for compaction, following AG-17's pattern (one sentence naming what closes).
- [ ] 10.3 Confirm the `R-18` traceability-spine row needs no edit (R-11 was already mapped to AG-18; only its discharge status changes, recorded in the spec back-annotations, not the spine mapping).

## Phase 11 — OpenSpec promotion + archive (same PR)

- [ ] 11.1 Create `openspec/specs/agent-compaction/spec.md` — the new capability, verbatim from the change-dir source (adjust only the "Becomes … at archive" framing line).
- [ ] 11.2 For each of the seven delta specs, apply the MODIFIED blocks in place into `openspec/specs/<capability>/spec.md`, full-block preservation. Diff each promoted file against its change-dir delta source to confirm nothing dropped or truncated (known repo risk: sub-agent artifact truncation).
- [ ] 11.3 `git mv openspec/changes/cachicamas-agent-compaction/ openspec/changes/archive/2026-08-19-cachicamas-agent-compaction/`. Verify every moved file's content against its pre-move git blob byte-for-byte — do not trust a "moved" report.
- [ ] 11.4 Write `archive-report.md`: what shipped, commits, capabilities promoted, verification at close, R-11/G3 discharge, state at close — Layer 2 stands at 18 of 24, AG-19 next (delegation/subagent).
- [ ] 11.5 Update `apply-progress.md` with every RED/GREEN transition, all four bites' RED evidence (`S-CMP-030`, `S-CMP-031`, `S-CMP-032`, `S-CMP-033`) plus the pre-existing `agent-history`/`agent-context-strategy` bites' regression confirmations, and the byte-unchanged verification results.

## Phase 12 — Final gate

- [ ] 12.1 `cd backend/agent && go test -race -count=1 ./...` green, full suite. Record wall-clock duration (`NFR-CMP-002`) — a sub-second result is a cache artifact, not evidence.
- [ ] 12.2 `golangci-lint cache clean && make lint` — 0 issues. Never `make fmt`/`make all` (the `fmt` step rewrites committed files in place — known repo lesson).
- [ ] 12.3 `gofmt -l backend/agent` — empty output.
- [ ] 12.4 `make build` clean; `make vuln-check` clean (not part of `make all`).
- [ ] 12.5 Confirm import-boundary and ambient-authority guards pass with zero change (`NFR-CMP-003`).
- [ ] 12.6 `tasks.md` self-check: every `[ ]` closed to `[x]`; coverage table below fully mapped; PR description states why the change exceeds the default 400-line budget (`NFR-CMP-005`).

## Coverage table — every scenario has a task

| Scenario | Task(s) |
|---|---|
| S-CMP-001…006, 032(bite) | 3.1–3.7 |
| S-CMP-007, 008, 010–016, 027, 028, 030(bite) | 1.3–1.14 |
| S-CMP-017–019, 029, 034–036 | 4.1–4.7 |
| S-CMP-020, 021, 031(bite) | 5.1–5.4 |
| S-CMP-022–026, 033(bite), 037 | 6.5–6.11, 4.8 |
| S-HIS-100–104, 030/031/102(bite) | 1.1–1.15, 8.1–8.4 |
| S-CTX-005, 006, 022–025 | 2.2–2.6, 6.4 |
| S-CST-015, 016 | 3.5–3.6 |
| S-LSK-029, 030 | 8.5–8.7 |
| `agent-run-driver`/`agent-v1-scope`/`agent-protocol-events` back-annotations | 9.1–9.4 |
| Doc 0003 status / counter / checklist | 10.1–10.3 |
| OpenSpec promotion + archive | 11.1–11.5 |

## Known risks carried forward

- The AG-16/17 lesson repeats: `harness.go`'s line numbers shift with every milestone — re-grep at apply time, never trust a cited number blindly (Phase 0 tasks do this).
- `reflect.DeepEqual` over whole `Entry` values is the single highest-likelihood mistake in this change (design AD-8) — task 1.10 names the forbidden shape explicitly; grep new test files for `reflect.DeepEqual(.*Entry` before closing Phase 1.
- The `agent-run-driver`/`agent-v1-scope`/`agent-protocol-events` back-annotations have no Go test (Phase 9) — the proposal's own risk register names this the highest-likelihood skip.
- `S-CMP-032`'s bite is easy to record against the wrong assertion: it must fail the **cumulative** figure short, not merely fail to find a `cost_turn` event (the event is still emitted; only the fold is removed).
- Both substrate filters' exact tail must be re-grepped at apply time (task 0.1/8.5) — this file shifts with every milestone that touches it.
