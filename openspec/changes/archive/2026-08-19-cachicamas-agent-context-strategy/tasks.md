# Tasks: AG-17 — Add the context strategy seam and token accounting

> Change: `cachicamas-agent-context-strategy`. Binding inputs: `proposal.md` (Decisions 1–5), `design.md` (DD1–DD7, Approach steps 1–6, NFR-CTX-005's own slicing rule), six spec files under `specs/` (`agent-context-strategy` NEW, R-CTX-001…012, S-CTX-001…021, 4 bites; `agent-run-driver`, `agent-history`, `agent-v1-scope`, `agent-retry-failover`, `agent-loop-skeleton` MODIFIED). Strict TDD: every behavior task is RED-first (`cd backend/agent && make test -race -count=1`), RED output recorded in `apply-progress.md` before GREEN. Ships in **one PR**: implementation, doc 0003 update, and the OpenSpec archive together.

## Ordering note — AG-17.1 vs AG-17.2, reconciled not contradictory

Doc 0003's DAG (`0003:1575`, `AG17_1 --> AG17_2`) reads as "AG-17.2 depends on AG-17.1." `design.md`'s NFR-CTX-005 instead orders build work "AG-17.2's types, estimate, resolver and fixture first … then AG-17.1's consultation." These are **not** in conflict once split by testability: `ContextPrompt.Accounting` is typed `TokenAccounting` (an AG-17.2 type), so AG-17.1's own prompt struct cannot even compile before AG-17.2's types exist — and several AG-17.2 scenarios (`S-CTX-011`, `S-CTX-019`) are only observable **through a driven harness run**, which needs AG-17.1's consultation wired in first. The plan below therefore builds AG-17.2's non-behavioral pieces first (Phases 1–4, zero behavior change), wires AG-17.1's seam next (Phase 5, closing every AG-17.1 scenario), then closes AG-17.2's remaining externally-observed scenarios last (Phase 6) — satisfying both the DAG edge and the NFR's slicing rule at once. `S-CTX-005`, `S-CTX-009/010`, and `S-CTX-014`/bite `S-CTX-015` are pure type-level scenarios (reflection + zero-value construction) and close in Phase 1 without any wiring.

## Review Workload Forecast

Refined against measured, already-committed SDD source (not estimated): `proposal.md` 361 lines, `spec.md` 311, `design.md` 325, and the five deltas 39+33+39+45+73 = 229 — **1226 lines already in the branch**, before this file, `apply-progress.md`, `archive-report.md`, `verify-report.md`, or the doc 0003 edit are added. This raises the proposal's own SDD-markdown estimate (750–1150) meaningfully.

| Field | Value |
|---|---|
| Already-committed SDD markdown (measured) | 1226 |
| Remaining SDD markdown (this file ~150–280, apply-progress ~150–350, archive-report ~80–160, verify-report ~200–400, doc 0003 diff ~30–60) | 610–1250 |
| Estimated SDD markdown total | 1836–2476 |
| Estimated Go (production + tests, per proposal's own component table) | 810–1295 |
| **Estimated total changed lines** | **2646–3771** |
| 400-line budget risk | High |
| Chained PRs recommended | No — `size:exception` pre-accepted (user, "extend if needed") |
| Suggested split (if ever needed) | U1 (AG-17.2 foundation: types, estimate, resolver, fixture — zero behavior change) → U2 (AG-17.1 seam wiring + its scenarios) → U3 (AG-17.2's externally-observed scenarios) → U4 (substrate + delta back-annotations) → U5 (doc 0003) → U6 (OpenSpec archive) |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

Judgment: the refined estimate exceeds even the pre-authorised 1000-line nominal exception bar, mirroring AG-16's own precedent (forecast 1730–2785, shipped as one PR under the same exception). The excess is markdown density, not code: the Go subtotal (810–1295) is comparable to AG-16's. If a seam is ever needed, it falls at the six work-unit boundaries above.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| U1 | AG-17.2 types + budget + estimate + resolver + fixture (inert, zero behavior change) | PR 1 (single-pr) | `go test -race -count=1 -run 'TestContextBudget|TestEstimateTokens|TestResolveTokenAccounting|TestTokenAccounting|TestContextVerdict'` | N/A — no observable behavior differs yet | `context_strategy.go` (types only), `token_accounting.go`, `agenttest/counting_provider.go` |
| U2 | Seam wiring in `harness.go` + AG-17.1 scenarios | PR 1 | `go test -race -count=1 -run 'TestHarness_ContextStrategy'` | `make test` full suite (proves the insertion broke nothing else) | `harness.go` consultation block, two new fields |
| U3 | AG-17.2's externally-observed scenarios (discovery, pre-hook divergence) | PR 1 | `go test -race -count=1 -run 'TestHarness_Accounting|TestHarness_ContextStrategy_.*Discover'` | `make test` | new test cases in `context_strategy_test.go` |
| U4 | Substrate verification + filter widening + 5 delta back-annotations | PR 1 | `go test -race -count=1 -run 'TestLoop_Substrate|TestLayer2DocContract'` | `make lint` + `make build` | filter entries, delta files under `specs/` |
| U5 | Doc 0003 update | PR 1 | N/A — documentation only | N/A | status line, counter, checklist tick |
| U6 | OpenSpec promotion + archive | PR 1 | N/A — spec promotion only | N/A | six spec merges, `git mv` of the change dir |

## Exact new/modified files

| File | Action | Hosts |
|---|---|---|
| `backend/agent/src/agent/context_strategy.go` | Create | `ContextStrategy`, `ContextPrompt`, `ContextVerdict`, `NoOpContextStrategy`, `ContextBudget`, `ContextBudgetOf`, `var _` guards |
| `backend/agent/src/agent/token_accounting.go` | Create | `TokenSource`, `TokenAccounting`, `resolveTokenAccounting`, `estimateTokens`, `estimateBytesPerToken`, `estimateTokensPerMessage` |
| `backend/agent/src/agent/token_accounting_test.go` | Create (internal `package agent`) | `S-CTX-009/010/012/013(bite)/016/017/018` |
| `backend/agent/src/agent/context_strategy_test.go` | Create (external `package agent_test`) | `S-CTX-001…008, 011, 014, 015(bite), 019, 020` |
| `backend/agent/src/agenttest/counting_provider.go` | Create | `CountingProvider`, `NewCountingProvider`, `NewFailingCountingProvider`, `CountTokens`, `CountRequests`, guards |
| `backend/agent/src/agenttest/counting_provider_test.go` | Create | fixture physics: report, error, absent-count, capture order |
| `backend/agent/src/agent/harness.go` | Modify | `ContextStrategy`/`ContextBudget` fields after `Failover` (`:91`); consultation block after `:498` |
| `backend/agent/src/agent/loop_test.go` (`filterOutLoopFiles`) | Modify | append exact-filename entries for AG-17's new files |
| `backend/agent/src/agent/loop_hook_test.go` (`filterOutLoopHookFiles`) | Modify | append exact-filename entries, byte-in-sync |

## Phase 0 — Foundation: RED baseline & substrate verification

- [x] 0.1 Grep `backend/` for `ContextStrategy`, `ContextBudget`, `TokenAccounting`, `TokenSource`, `ContextPrompt`, `ContextVerdict` — confirm **zero** hits, proving the pre-existing state the spec pins. Record in `apply-progress.md`.
- [x] 0.2 Read `doc_contract_guard_test.go`'s `expectedLayer2ContractRows` — confirm exactly **eight** rows, `L2C-01`…`L2C-08`, and read `doc.go`'s own contract paragraph count to match.
- [x] 0.3 Re-open `harness.go` around `:498` (`transcript := transcriptFromHistory(hist)`) and `:530` (the attempt loop `for attempt := 1; ; attempt++`); confirm the window's content matches `design.md` DD2's quoted before/after, by exact line content not by line number (AG-16 lesson: this file shifts with every milestone).
- [x] 0.4 Re-locate `filterOutLoopFiles` (`loop_test.go`) and `filterOutLoopHookFiles` (`loop_hook_test.go`)'s current tail by grep, not by cited line number; record the true ranges in `apply-progress.md`.
- [x] 0.5 Confirm `agenttest.Provider.Stream` is on the **pointer** receiver (`fake_provider.go:74`) and re-read `stubProviderWithTokenCounter`'s embedding shape (`agenttest/provider_test.go:133-142`) as the fixture template.
- [x] 0.6 Confirm `ai.TokenCounter` (`ai/provider.go:130-134`), `ai.TokenCount.Count()` (`ai/usage.go:57-77`), and `buildLoopRequest` (`loop.go:713-726`) signatures match `design.md`'s citations.

## Phase 1 — AG-17.2 foundation: types (wired to nothing, zero behavior change)

Covers `R-CTX-003`, `R-CTX-005`, `R-CTX-008` type-shape half. `S-CTX-005`, `S-CTX-009`, `S-CTX-010`, `S-CTX-014`, `S-CTX-015` are all pure type-level scenarios needing no seam wiring.

- [x] 1.1 RED: `context_strategy_test.go` (external) — `TestContextVerdict_HasNoFields` (`S-CTX-005`): `reflect.TypeOf(ContextVerdict{}).NumField() == 0`; scan the exported package surface for any constructor or method returning a value distinguishable from the zero value. Record RED (undefined symbol).
- [x] 1.2 GREEN: create `context_strategy.go` per DD1 — `ContextStrategy` interface, `ContextPrompt` (exported fields `Transcript`, `Budget`, `Accounting`), `ContextVerdict struct{}`, `NoOpContextStrategy{}` + `var _ ContextStrategy = NoOpContextStrategy{}`.
- [x] 1.3 Add `ContextBudget`/`ContextBudgetOf`/`Limit()` per DD3 in the same file.
- [x] 1.4 RED: `TestContextBudget_AbsentVsStatedZero` (`S-CTX-009`): zero value reads absent via `Limit()`; `ContextBudgetOf(0)` reads `(0, true)`; the two are unequal.
- [x] 1.5 RED: `TestContextBudget_NegativeIsAbsentTotal` (`S-CTX-010`): `ContextBudgetOf(-1)` returns no error/panic, equals the zero value, reads absent.
- [x] 1.6 GREEN: 1.4–1.5 pass once `ContextBudgetOf` is total per DD3 (negative → absent zero value).
- [x] 1.7 RED: create `token_accounting.go` shell — `TokenSource` (`iota`: `Unavailable`, `Reported`, `Estimated`) + `String()`; `TokenAccounting{tokens int64; source TokenSource}` + `Tokens() (int64, TokenSource)` — no field exported, no single-result accessor (DD4).
- [x] 1.8 GREEN: `token_accounting_test.go` (internal) — table asserting `TokenSourceUnavailable`/`Reported`/`Estimated`.String() render distinctly (part of `S-CTX-014`).
- [x] 1.9 RED: `context_strategy_test.go` — `TestTokenAccounting_UnreadableWithoutSource` (`S-CTX-014`): from the external package, reflect on `TokenAccounting`'s exported field count (0) and exported method set (exactly `Tokens() (int64, TokenSource)`, no single-result accessor); zero `TokenAccounting{}` reads `(0, Unavailable)`.
- [x] 1.10 Bite **`S-CTX-015`** (RED-first, then revert): add a bare `func (a TokenAccounting) TokensOnly() int64 { return a.tokens }` beside `Tokens()`; re-run 1.9; confirm it FAILS the "no single-result figure accessor" assertion — proving distinguishability is mechanical. Record RED, revert. Note for the implementer per spec: if this bite is instead taken as *removing* the two-result accessor, the resulting **compile failure** is itself valid RED evidence — record either shape.

## Phase 2 — The estimate, in isolation (`R-CTX-009`, `R-CTX-010`, DD5)

- [x] 2.1 RED: `token_accounting_test.go` — `TestEstimateTokens_TableDriven` (`S-CTX-016`): rows for pure ASCII, CJK, empty transcript, tool-schema-bearing, system-instruction-bearing requests; each row's figure equals `⌈B/D⌉ + M×K` computed independently over that row's own byte total and message count; the CJK row's figure exceeds a rune-count equivalent. Record RED (undefined symbol).
- [x] 2.2 GREEN: implement `estimateTokens(req ai.Request) int64` per DD5's exact accessor walk (system instruction segments, message content parts including tool-call/tool-result/reasoning, tool schemas name+description+schema); constants `estimateBytesPerToken = 4`, `estimateTokensPerMessage = 4`, each documented with its stated rationale and the accuracy caveat.
- [x] 2.3 `TestEstimateTokens_Determinism` (`S-CTX-018`): computed twice over the same request value and once over an independently re-constructed equal request → three identical figures; read the estimate's imports/call graph — no clock, no env, no randomness, no I/O.
- [x] 2.4 `TestEstimateTokens_NeverConvertedToTokenCount` (`S-CTX-017`, mechanical half): a consumer holding an estimated `TokenAccounting` necessarily receives the `Estimated` provenance in the same call (already proved by `Tokens()`'s shape from Phase 1); assert that no path converts an estimate into `ai.TokenCount`. **Post-verify correction**: this tick originally stood on a Phase-7 diff check with **no test of this name anywhere in `backend/`**. The test now exists (`token_accounting_test.go`) and carries two independent locks — (1) `reflect` on `estimateTokens`' result type, which must be a bare `int64` and never `ai.TokenCount`; (2) a `go/ast` walk over **every** non-test file in the package asserting none calls `ai.Tokens`, the only exported constructor of an `ai.TokenCount` (`ai/usage.go:53-55`; the struct's fields are unexported, so a composite literal is impossible from outside package `ai`). The AST walk closes the residual gap the verify report named: it ignores comments (`cost_usage.go` mentions `ai.Tokens(0)` in prose) and it covers **files that do not exist yet**, which a byte-unchanged diff over a fixed file list cannot. Proved load-bearing by defeat: a temporary production file calling `ai.Tokens` makes it FAIL naming that file and line.

## Phase 3 — The accounting resolver (`R-CTX-006`, `R-CTX-007`, DD4)

- [x] 3.1 RED: `token_accounting_test.go` — `TestResolveTokenAccounting_ThreeStates` (`S-CTX-012`): four fixtures — (a) counting fixture reporting a present figure, (b) non-counting `agenttest.Provider`, (c) advertised counter returning a non-nil error, (d) advertised counter returning nil error + absent count. Assert (a) `Reported` + figure, (b) `Estimated` + estimate's figure, (c) and (d) both `Unavailable` + zero figure, equal to the zero `TokenAccounting` value. Record RED (fixtures from Phase 4 not yet built — stub minimally or sequence after 3.x if needed).
- [x] 3.2 GREEN: implement `resolveTokenAccounting(ctx, provider, opts, system, transcript)` per DD4 — `buildLoopRequest` failure → `Unavailable`; type assertion fails → `estimateTokens` + `Estimated`; type assertion succeeds, `CountTokens` errors or returns absent → `Unavailable`; succeeds with present count → `Reported`.
- [x] 3.3 Bite **`S-CTX-013`** (RED-first, then revert): collapse the three states to two — route an advertised counter's error through `estimateTokens` instead of `Unavailable` (scratch edit); re-run 3.1; confirm rows (c) and (d) FAIL reporting `Estimated`. Record RED, revert.

## Phase 4 — The exported `agenttest` fixture (DD7)

- [x] 4.1 Create `agenttest/counting_provider.go` — `CountingProvider` embedding `*Provider` (pointer, per `fake_provider.go:74`); `NewCountingProvider(result ai.TokenCount, scripts ...Script)`; `NewFailingCountingProvider(err error, scripts ...Script)`; `CountTokens`; `CountRequests()` (fresh slice); `var _ ai.ModelProvider` and `var _ ai.TokenCounter` guards.
- [x] 4.2 `counting_provider_test.go` — fixture physics: `NewCountingProvider` reports the scripted result and captures the request; `NewFailingCountingProvider` returns the scripted error and still captures the request; `CountRequests()` returns a fresh slice each call (mutation-safety, `Requests()` posture at `fake_provider.go:157-161`).
- [x] 4.3 Re-run Phase 3's `TestResolveTokenAccounting_ThreeStates` (3.1) now wired to the real `CountingProvider`/`NewFailingCountingProvider` fixtures; confirm GREEN.

## Phase 5 — The seam consultation (`R-CTX-001`, `R-CTX-002`, `R-CTX-003`, `R-CTX-004`)

- [x] 5.1 Add `ContextStrategy ContextStrategy` (nil-default) and `ContextBudget ContextBudget` (zero-default) fields to `Harness`, after `Failover` (`harness.go:91`).
- [x] 5.2 RED: `context_strategy_test.go` — `TestHarness_ContextStrategy_ConsultedOncePerLogicalTurn` (`S-CTX-001`): recording strategy, 2 clean logical turns; assert exactly 2 consultations, each before that turn's first provider call (interleaved against `Provider.Requests()`); run outcome/error/event stream unchanged vs. no strategy. Record RED (nil field never consulted).
- [x] 5.3 GREEN: insert the consultation block immediately after `harness.go:498`, guarded by `h.ContextStrategy != nil`, per DD2's exact quoted comment and code — `Transcript: slices.Clone(transcript)`, `Budget: h.ContextBudget`, `Accounting: resolveTokenAccounting(runCtx, h.Provider, h.Turn, h.System, transcript)`; discard the returned `ContextVerdict`.
- [x] 5.4 RED→GREEN: `TestHarness_ContextStrategy_RetriedTurnConsultsOnce` (`S-CTX-002`): turn 1 fails retryably once (no partial output) then completes; turn 2 clean — 3 provider attempts, 2 logical turns; assert exactly 2 consultations (not 3); the two transcripts the strategy received are the two distinct per-logical-turn transcripts, never the same one twice.
- [x] 5.5 Bite **`S-CTX-003`** (RED-first, then revert): move the consultation block inside the attempt loop (`harness.go:530`) on a scratch tree; re-run 5.4; confirm it FAILS reporting **3** consultations. Record RED, revert to 5.3's placement.
- [x] 5.6 RED→GREEN: `TestHarness_ContextStrategy_PromptCarriesTranscriptAndBudget` (`S-CTX-004`): each recorded prompt's `Transcript` is element-equal to the sent request's messages (read back via `Provider.Requests()`); `Budget` echoes a stated `ContextBudgetOf(n)`; each prompt carries an `Accounting` value. Additionally: a strategy that appends to/overwrites its received `Transcript` slice → the provider's recorded requests are byte-identical to the non-mutating run (proves the clone).
- [x] 5.7 RED→GREEN: `TestHarness_ContextStrategy_NoOpInstalled_ZeroCompactionEvents` (`S-CTX-006`): install `NoOpContextStrategy{}`, drive a run; compile-time guard binds; every consultation returns the zero verdict; scan the recorded stream — zero compaction-family kinds.
- [x] 5.8 RED→GREEN: `TestHarness_ContextStrategy_NilVsNoOpByteIdentical` (`S-CTX-007`): identical scripts on plain `agenttest.Provider`; run A nil field/absent budget, run B `NoOpContextStrategy{}`/present budget; assert **byte-identical** event streams and **byte-identical** `hist.Entries()` read-backs; zero compaction kinds on either; equal returned values/outcomes.
- [x] 5.9 Bite **`S-CTX-008`** (RED-first, then revert): make the consultation block emit one `CompactionStarted` event on a scratch tree; re-run 5.8; confirm the byte-identity comparison FAILS. Record RED, revert.

## Phase 6 — AG-17.2's externally-observed scenarios (need the wired seam)

- [x] 6.1 RED→GREEN: `context_strategy_test.go` — `TestHarness_ContextStrategy_Accounting_DiscoveryAndFallback` (`S-CTX-011`): counting-capable `CountingProvider` scripted with a figure, driven with a recording strategy → prompt's accounting reports that figure with `Reported`; the fixture recorded exactly one `CountTokens` call built from the turn's own `buildLoopRequest`; given the non-counting `agenttest.Provider` instead, same script → no counting call is made, accounting reports `Estimated`. Enumerate `agent`'s declared interfaces — confirm no Layer 2 token-counting interface exists.
- [x] 6.2 RED→GREEN: `TestHarness_Accounting_PreHookDivergenceRecorded` (`S-CTX-019`): counting-capable fixture + a request-mutating `PreRequestHook`; assert the accounting the strategy received corresponds to the **pre-hook** request; the fixture's captured counting request and the provider's recorded sent request **differ**; the test records that divergence rather than asserting equality. Read the shipped doc comments on `TokenSourceReported` and `ContextPrompt.Accounting` — confirm each states "for the pre-hook request."

## Phase 7 — Closed-sequence proof & substrate verification (`R-CTX-012`, `NFR-CTX-002…004`)

- [x] 7.1 `TestHarness_ContextStrategy_ClosedSequencesUnaffected` (`S-CTX-020`): run the shipped `S-LSK-001` and `S-CAN-013` tests unmodified and byte-unchanged — confirm both pass; scan the recorded stream of a multi-logical-turn run with `NoOpContextStrategy` installed — zero compaction-family events, kind sequence position-for-position equal to the nil-field run from `S-CTX-007`.
- [x] 7.2 `TestNoRelease_SubstrateByteUnchanged` (`S-CTX-021`): diff `doc.go`, `doc_contract_guard_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `compaction_events.go`, `cost_events.go`, `cost_usage.go`, `history.go` against `origin/main@b0de5bf6` — all empty; diff under `backend/agent/src/ai/` is empty; `go.mod`/`go.sum` diff empty; the every-kind-constructible guard passes at its committed kind count; `expectedLayer2ContractRows` still carries exactly eight rows.
- [x] 7.3 Confirm `-race -count=1` evidence recorded for every new test (`NFR-CTX-002`); confirm no test synchronizes by sleep/timeout/wall-clock (grep for `time.Sleep` in new test files).
- [x] 7.4 Confirm import-boundary and ambient-authority guards pass with zero change (`NFR-CTX-003`); confirm `time` package is not flagged (it isn't forbidden) and that `R-CTX-010`'s determinism relies on the direct assertion (2.3), never on this guard.
- [x] 7.5 Re-run `git diff main -- backend/agent/src/agent/` non-test files (`S-LSK-027`): confirm the set is exactly `{harness.go}`.

## Phase 8 — Substrate filter widening (`R-LSK-004`, `S-LSK-028`)

- [x] 8.1 Widen `filterOutLoopFiles` and `filterOutLoopHookFiles` at their re-grepped tail (task 0.4) — append, byte-in-sync, one exact-filename entry per file `src/agent` covers under the filters' own population rule (at minimum `/context_strategy.go`, `/token_accounting.go`, plus whichever new test files the rule covers, determined by reading the two filters' predicates at apply time). **No pre-existing filename is added** — AG-17 takes no release. `cost_events.go`, `cost_events_test.go`, `stream_check_test.go` stay absent from both; no file under `src/agenttest/` is added (these filters do not govern it).
- [x] 8.2 Confirm both filters carry an identical entry set (diff the two suffix lists); no wildcard, prefix, or directory pattern; an unnamed new file still fails both guards.

## Phase 9 — Spec deltas: five back-annotations (Capabilities table, proposal)

- [x] 9.1 Apply the `agent-run-driver` delta's back-annotation to row `:342` in `openspec/specs/agent-run-driver/spec.md` — the full non-requirements table reproduced, exactly the one row amended, verbatim per this change's delta file. Diff against the delta source to confirm nothing dropped.
- [x] 9.2 Apply the `agent-history` delta's back-annotation to row `:260` — history needed no change; `history.go` byte-unchanged, confirmed by 7.2.
- [x] 9.3 **Explicit task — `agent-v1-scope` back-annotation, enforced by NO Go test (proposal risk 6).** Apply `R-AGS-007`'s back-annotation paragraph and the `S-AGS-023` scenario update per the delta file, mapping seam 5 → AG-17.1 and seam 6 → AG-17.2 with their shipped mechanism recorded, following AG-15's template at `:130`. Manually confirm both rows carry the annotation — no automated check will catch a skip here.
- [x] 9.4 Apply the `agent-retry-failover` delta's back-annotation to `R-RTY-002` — reproduce both scenarios verbatim, append only the AG-17 parenthetical note to `S-RTY-002`. Confirm `S-RTY-002`'s implementing test (retry policy suite) stays byte-unchanged and green.
- [x] 9.5 Apply the `agent-loop-skeleton` delta — extend the header's allocated range to `S-LSK-001` through `S-LSK-028`; land the "AG-17 requests NO release" paragraph inside `R-LSK-004` with its four numbered consequences; append scenarios `S-LSK-027` and `S-LSK-028` verbatim.
- [x] 9.6 For all five deltas: re-read every cited `file:line` against the actually-shipped change (not the design's citations, which may have shifted) — record any drift found.

## Phase 10 — Documentation (part of the deliverable)

- [x] 10.1 `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` — status line (`:3`): extend "Wave 3 opens with AG-12…AG-16" to close Wave 3 and open Wave 4 with AG-17, following AG-13/14/15/16's narrative-sentence pattern (one sentence naming what AG-17 closes, mirroring AG-16's own added sentence); bump the milestone counter from **16 of 24** to **17 of 24**.
- [x] 10.2 Line `:2175` completion checklist — tick `- [ ] The context strategy is consulted before every call; token counting is capability-discovered with a labelled fallback — closed by AG-17.` to `- [x]`, since AG-17 fully closes both halves.
- [x] 10.3 Confirm no `L2C-0N` row is owed (task 7.2 covers `doc.go`/`doc_contract_guard_test.go` byte-unchanged); confirm `backend/agent/README.md` needs no edit (grep for a context/token-accounting section).
- [x] 10.4 Confirm the `R-18` traceability-spine row (`:2217`, "seam 5 → AG-17.1 · seam 6 → AG-17.2") already reflects the shipped mapping — no edit needed, since the mapping was never wrong, only undischarged.

## Phase 11 — OpenSpec promotion + archive (same PR)

- [x] 11.1 Create `openspec/specs/agent-context-strategy/spec.md` — the new capability, verbatim from `openspec/changes/cachicamas-agent-context-strategy/specs/agent-context-strategy/spec.md` (adjust only the "Becomes … at archive" framing line).
- [x] 11.2 For each of the five delta specs, apply the MODIFIED blocks in place into `openspec/specs/<capability>/spec.md`, full-block preservation, per Phase 9's tasks. Diff each promoted file against its change-dir delta source to confirm nothing dropped or truncated.
- [x] 11.3 `git mv openspec/changes/cachicamas-agent-context-strategy/ openspec/changes/archive/2026-08-19-cachicamas-agent-context-strategy/` (AG-16 archive-directory naming precedent). Verify every moved file's content against its pre-move git blob — diff byte-for-byte, do not trust a "moved" report (known repo risk: sub-agent artifact truncation).
- [x] 11.4 Write `archive-report.md` following AG-16's shape: what shipped, commits, capabilities promoted, verification at close, carried-forward follow-ups (AG-18 now startable), state at close — Layer 2 stands at 17 of 24, AG-18 next.
- [x] 11.5 Update `apply-progress.md` with every RED/GREEN transition, every bite's RED evidence (`S-CTX-003`, `S-CTX-008`, `S-CTX-013`, `S-CTX-015`), and the byte-unchanged verification results (Phase 7), per the spec's Evidence discipline section.

## Phase 12 — Final gate

- [x] 12.1 `cd backend/agent && make test` green under `-race -count=1`, full suite. A cached run is not evidence.
- [x] 12.2 `golangci-lint cache clean && make lint` — 0 issues. **Never `make fmt`/`make all`** — the `fmt` step rewrites committed files in place (known repo lesson).
- [x] 12.3 `gofmt -l backend/agent` — empty output.
- [x] 12.4 `make build` clean; `make vuln-check` clean (not part of `make all`).
- [x] 12.5 `tasks.md` self-check: every `[ ]` closed to `[x]`; coverage table below fully mapped.

## Coverage table — every scenario has a task

| Scenario | Task(s) |
|---|---|
| S-CTX-001 | 5.2–5.3 |
| S-CTX-002 | 5.4 |
| S-CTX-003 (bite) | 5.5 |
| S-CTX-004 | 5.6 |
| S-CTX-005 | 1.1–1.2 |
| S-CTX-006 | 5.7 |
| S-CTX-007 | 5.8 |
| S-CTX-008 (bite) | 5.9 |
| S-CTX-009 | 1.4, 1.6 |
| S-CTX-010 | 1.5–1.6 |
| S-CTX-011 | 4.3, 6.1 |
| S-CTX-012 | 3.1–3.2 |
| S-CTX-013 (bite) | 3.3 |
| S-CTX-014 | 1.7–1.9 |
| S-CTX-015 (bite) | 1.10 |
| S-CTX-016 | 2.1–2.2 |
| S-CTX-017 | 2.4, 7.2 |
| S-CTX-018 | 2.3 |
| S-CTX-019 | 6.2 |
| S-CTX-020 | 7.1 |
| S-CTX-021 | 7.2, 7.5 |
| S-AGS-023/024/025/026 (agent-v1-scope) | 9.3 |
| S-RTY-002 (annotated), S-RTY-011 (held) | 9.4 |
| S-LSK-020 (held), S-LSK-027 | 7.5, 9.5 |
| S-LSK-028 | 8.1–8.2, 9.5 |
| Doc 0003 status / counter / checklist / spine | 10.1–10.4 |
| OpenSpec promotion + archive | 11.1–11.5 |

## Known risks carried forward

- The AG-17.1/AG-17.2 build-order reconciliation (top of this file) must be read before starting Phase 5 or 6 — building the seam consultation before Phase 1–4's types will not compile, since `ContextPrompt.Accounting` is typed `TokenAccounting`.
- `S-CTX-013`'s and `S-CTX-015`'s RED-first bites both touch code that Phase 3/1 already made GREEN — revert discipline must be exact (scratch-tree edit, observe FAIL, revert to the prior GREEN commit state) or the bite evidence is not trustworthy.
- `agent-v1-scope`'s back-annotation (9.3) has no Go test — the single highest-likelihood skip in this change per the proposal's own risk register (risk 6).
- The refined line-count estimate (2646–3771) is meaningfully above the proposal's own forecast (1560–2445); flagged for the orchestrator before apply even though `Decision needed before apply: No` per the pre-accepted exception.
- Both substrate filters' exact tail must be re-grepped at apply time (task 0.4) — never trust a cited line number, this file shifts with every milestone that touches it (AG-16 lesson, repeated here because it will repeat again).
