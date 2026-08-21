# Tasks: AG-22 — Add the observability boundary

## Review Workload Forecast

Counted = Go source/tests + `docs/adr/` + `docs/architecture/`. Uncounted = `openspec/` (492 lines already written, excluded per session config).

| File | Action | Est. lines (counted) |
|---|---|---|
| `docs/adr/0005-promote-agent-stack-to-own-module.md` | Modify — new § D3 subsection (span/attribute table, not-recorded list, denylist restated) | 70–90 |
| `docs/architecture/milestones/0003-...md` | Modify — status header "22 of 24"→"23 of 24" + AG-22 back-annotation sentence appended to the Wave-6 run-on paragraph (line 3 is one physical line; git will likely show it as ~2 changed lines despite the prose volume — flagged, not undercounted on purpose) | 10–20 |
| `backend/agent/src/agent/observability.go` (new) | Attribute-key constants, span-name constants, tracer-acquisition helpers, one finalizer per family | 170–220 |
| `backend/agent/src/agent/harness.go` | `TracerProvider` field; run span open/finalizer in `Run`; run span in `Compact` | 40–60 |
| `backend/agent/src/agent/loop.go` | Turn span open/finalizer; run span on the nil-continuation path | 40–60 |
| `backend/agent/src/agent/scheduler.go` | Tool span open/finalizer in `executeCall` (post-gate); detached-arm marker in `runToolWithWindDown` | 50–70 |
| `backend/agent/src/agent/compaction.go` | Compaction span open/finalizer in `runCompaction` | 40–60 |
| `backend/agent/src/agent/import_boundary_test.go` | Five § D3-annotated `allowedProductionPrefixes` entries + comment update | 30–40 |
| `backend/agent/src/agenttest/tracetest/tracetest.go` | Parent capture in `tracer.Start`, `(*Span).Parent()` accessor, `SpanContext()` non-empty enough to compare | 15–25 |
| `backend/agent/src/agenttest/tracetest/tracetest_test.go` | RED-first `Parent()` test; update `TestSpanInterface_MethodCounts` if the accessor changes counted surface | 30–50 |
| `backend/agent/src/agent/observability_nesting_test.go` (new) | R-AGO-002/006 nesting + attribute-equality proof | 150–220 |
| `backend/agent/src/agent/observability_denylist_test.go` (new) | R-AGO-004/005/008 absence proof, mirrors AI-37's 282-line precedent | 250–300 |
| `backend/agent/src/agent/observability_parity_test.go` (new) | R-AGO-001/009 no-tracer parity | 120–180 |
| `backend/agent/src/agent/observability_lifecycle_test.go` (new) | R-AGO-007 exactly-once, table-driven incl. detached fixture | 150–220 |
| **Counted total** | | **≈ 1165–1615** |

`400-line budget risk: High` against the generic 400-line default. Against this session's **granted 1000-line budget**, the midpoint (~1390) still exceeds it. **Per session config this is pre-authorised**: the overage is intrinsic to a milestone that must ship a decided vocabulary, a bite-proven guard, four instrumented seams and four independent proof surfaces (nesting, denylist, parity, lifecycle) in one PR — splitting AG-22.1/AG-22.2 was considered and rejected because AG-22.1 has zero shippable code of its own (the vocabulary is prose inside an ADR) and would produce a PR with no test evidence. Delivery strategy stays `single-pr`; no chaining or slicing is proposed.

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 (whole change) | Vocabulary decision, guard amendment, four-seam instrumentation, four proof suites, doc/archive | PR 1 (single, `size:exception`) | `cd backend/agent && go test -race -count=1 ./...` | Real: `go test -race -count=1 ./src/agent/... ./src/agenttest/...` against the full fixture set (tools, delegation, compaction, permission, failure arms) | Revert the merge commit — restores the two-entry allowlist, removes every production OTel import, leaves `src/ai/` and the Layer 1 allowlist untouched (proposal's rollback plan) |

---

## Phase 1 — AG-22.1: the Layer 2 attribute vocabulary (gate, no code)

- [x] 1.1 In `docs/adr/0005-promote-agent-stack-to-own-module.md`, insert a new subsection "**Layer 2 attribute allowlist (AG-22, extends § D3)**" immediately after the Layer 1 attribute-allowlist paragraph (currently `0005:246-250`) and before the absolute denylist blockquote (currently `0005:252-255`). Content: the four span-name/attribute tables from design D-B (run/turn/tool/compaction, each key → exact event accessor), the eight-item deferral-to-Layer-1 note (D-B), the seven-item not-recorded list with reasons (D-C), and the denylist restated byte-for-byte identical to `0005:252-255`.
- [x] 1.2 Diff the new subsection's denylist text against `0005:252-255` and confirm byte-identity (discharges `S-AGO-011`).
- [x] 1.3 Confirm no sibling ADR was created — the extension lands inside ADR 0005 only (discharges `S-AGO-010`).

**Discharges**: `R-AGO-002` (table), `R-AGO-003` (deferral note), `R-AGO-004` (denylist restated), `R-AGO-005` (not-recorded list) — the recorded-artifact half of each; `S-AGO-010`, `S-AGO-011`, `S-AGO-022`, `S-AGO-030`. This phase gates Phase 3 onward: no instrumentation task may start before 1.1–1.3 are committed, per the milestone's own "decided before anything is recorded" clause.

## Phase 2 — Test substrate: `tracetest` parent capture (RED-first, `agenttest` only)

- [x] 2.1 **RED**: in `backend/agent/src/agenttest/tracetest/tracetest_test.go`, add a test asserting `(*Span).Parent()` returns the parent `*Span` recorded from `trace.SpanFromContext(ctx)` at `Start`, and `nil` for a root span. Run `cd backend/agent && go test -race -count=1 ./src/agenttest/...` and observe it fail to compile (`Parent` undefined) — record the compile-time RED (design's bite #1).
- [x] 2.2 **GREEN**: in `tracetest.go`, extend `tracer.Start` to capture `parent, _ := trace.SpanFromContext(ctx).(*Span)` onto the new `Span`, add the `parent *Span` field and `(*Span) Parent() *Span` accessor. Re-run 2.1's test `-count=1` and confirm it passes.
- [x] 2.3 Check `TestSpanInterface_MethodCounts` (`tracetest_test.go:261`, the AD-4 pin) still passes unchanged — `Parent()` is a test-support struct accessor, not a `trace.Span` interface method, so the pinned counts (Span 11, Tracer 2, TracerProvider 2) must not move. If they do, the design's AD-4 note was wrong and this task blocks until resolved.

**Discharges**: the substrate half of `R-AGO-006`; unblocks Phase 4's nesting-test RED and Phase 8's nesting-test GREEN.

## Phase 3 — Observability scaffolding + AG-03.2 guard amendment (SAME COMMIT, RED-first bite)

- [x] 3.1 **RED (bite, design's bite #5 / `S-AGP-034`)**: create `backend/agent/src/agent/observability.go` importing `go.opentelemetry.io/otel/trace`, `.../otel/attribute`, `.../otel/codes` and defining the span-name/attribute-key constants (from Phase 1's ADR table) plus a tracer-acquisition helper (`trace.SpanFromContext(ctx).TracerProvider().Tracer(scope)`) and one finalizer stub per span family (run/turn/tool/compaction). Do NOT touch `import_boundary_test.go` yet. Run `cd backend/agent && go test -race -count=1 ./src/agent/... -run TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault -v` and observe it **FAIL**, naming the unadmitted OTel import path. Record the red output (this is the guard proving it still bites, per `S-AGP-034`).
- [x] 3.2 **GREEN**, same commit as 3.1: amend `allowedProductionPrefixes` in `import_boundary_test.go` with the five entries from design's *Import selection (C4)*: `go.opentelemetry.io/otel/trace` (§ D3 row `0005:240`), `.../otel/attribute` (`0005:241`), `.../otel/codes` (`0005:241`), `.../otel/semconv` as a **prefix** citing the forced-closure clause + the addendum's `go list -deps` measurement (never a § D3 row), `github.com/cespare/xxhash/v2` citing the same forced-closure clause. Update the package comment at `:123-126` that currently states no OTel/xxhash entry appears. Re-run 3.1's command `-count=1` and confirm GREEN.
- [x] 3.3 Re-run the full import-boundary suite (`TestLayer2_TestClosure_...`, `TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage`) `-count=1` to confirm no unrelated regression. **Deviation found and fixed**: `TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage` genuinely regressed — `go.opentelemetry.io/otel/trace@v1.44.0` itself (not a rejected SDK path) transitively imports `os`/`io/fs` via its own unreachable `auto.go` auto-instrumentation bridge code. Replaced the guard's blanket os/io-fs denial with a direct-importer check (`forcedStandardLibraryImporters`, `productionImportGraph`): a denied stdlib path is admitted only when every non-standard direct importer is on a small evidence-cited allowlist (`go.opentelemetry.io/otel/trace`); standard-library-on-standard-library plumbing is treated as inert. Bite-tested with a scratch direct `os` import inside `src/agent` — the check still fails, naming `src/agent` as the unexpected importer; reverted, suite green again.
- [x] 3.4 Run `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/agent/...` before and after 3.1–3.2, diff the two outputs, and confirm the diff introduces exactly the ten packages the orchestrator-measured `go.opentelemetry.io/otel/trace/noop` command already enumerated, with none beyond it. Record both listings and the diff as evidence (discharges `S-AGO-091`, `S-AGO-092`). **Measured**: diff introduces exactly 9 of the 10 packages (all except `trace/noop`, which nothing in this commit imports directly — `noop` is Phase 5's own import, when `Harness.Run`'s nil-default logic lands); all 9 are covered by the five allowlist entries, none beyond.

**Discharges**: `R-AGO-010`, `S-AGP-024`, `S-AGP-031`, `S-AGP-032`, `S-AGP-033`, `S-AGP-034`, `S-AGO-090`. Single commit obligation (constraint 1) is met by 3.1+3.2 landing together.

## Phase 4 — RED: write the four proof-type test files against not-yet-instrumented code

- [x] 4.1 **RED**: create `observability_nesting_test.go` (`package agent_test`) scripting a run using tools, the `delegating_tool_test.go` / `siblings_test.go` fixtures (verify at this task whether the design's cited `nested_run_test.go` is a separate file or already folded into one of these two — resolve before writing) and a compaction, wired to a fresh `tracetest.Provider` via `Harness.TracerProvider`. Assert the parent chain (`R-AGO-006`). Run `-count=1` and observe it fail on **zero spans recorded** (design's bite #2 / `S-AGO-044`) — not a compile error. Record the red output. **Resolved**: `nested_run_test.go` DOES exist (confirmed by `Read`, contrary to this file's own earlier "not found" note) and its `delegatingTool` fixture is reused directly (same `agent_test` package). **Deviation, disclosed**: `Harness.TracerProvider trace.TracerProvider` (a bare, zero-behavior field declaration — task 5.1's field only, none of its span-opening logic) was pulled forward into this same commit; without it this file could not compile at all, which would make the RED a compile error rather than the genuine "zero spans" assertion failure this task requires.
- [x] 4.2 **RED**: create `observability_denylist_test.go` scripting a full-featured run (tools, reasoning, permission, compaction, one failure arm) with runtime-concatenated sentinel markers for every denied category, driving `sweep.SelfTest(deny)` first, then `sweep.Scan` over `Provider.Corpus()`, plus the corpus-non-empty floor and attribute-count-≥-cardinality floor from `R-AGO-008`. Run `-count=1` and observe the **corpus-non-empty floor FAIL** (design's bite #3) because no span has been recorded yet — not the scan itself passing vacuously. Record the red output.
- [x] 4.3 **RED**: create `observability_parity_test.go` driving one scripted run twice — no tracer, then with `tracetest.Provider` — comparing drained event sequences and asserting `provider.Started() > 0` on the traced arm. Run `-count=1` and observe the **`Started() > 0` floor FAIL** (design's bite #4), not the sequence-equality assertion (which passes vacuously pre-instrumentation). Record the red output.
- [x] 4.4 **RED**: create `observability_lifecycle_test.go` as a table over every exit family per `R-AGO-007` (normal, each typed failure arm, cancellation, detached, panic-re-raise) asserting `AssertAllEndedOnce()` and the started-count-equals-ended-count invariant per family. Run `-count=1` and observe every row **FAIL on zero spans started** — record it.

**Cross-phase discovery (Phases 2/3, fixed in an intermediate commit)**: three pre-existing, actively-maintained "substrate untouched" guards (`TestTurn_SubstrateUntouched` R-LSK-004, `TestTurn_PreRequestHook_SubstrateUntouched` NFR-PRH-003, `TestHooks_ScopeFence_ByteUnchangedFilesAndNoNewKind`) — none anticipated by design or this file — assert most of `backend/agent/src/agent/` and all of `backend/agent/src/agenttest/` stay byte-unchanged against `origin/main`'s merge-base. AG-22's own approved Phase 2/3 work legitimately touches exactly the files these freeze. Widened each list following its own established per-milestone convention (the same "AG-10 widening" pattern prior milestones already use); see the dedicated `fix(agent)` commit for the full evidence.

**Discharges**: the RED half of `R-AGO-001/002/004/006/007/008/009`; all four are the substrate that Phases 5–8 turn GREEN incrementally. No task in this phase edits `harness.go`, `loop.go`, `scheduler.go`, or `compaction.go`.

## Phase 5 — Instrument `Harness.Run` and `Harness.Compact` (run span)

- [x] 5.1 In `harness.go`, add the exported `TracerProvider trace.TracerProvider` field (D-A), documented as nil-defaults-to-noop, never read via a global getter. **Pulled forward into Phase 4's commit (`d049c741`)**, disclosed there — this task discharges by confirming the field already exists unchanged.
- [x] 5.2 In `Run` (`harness.go:421`), open the run span **after** the pre-identity refusals (Hooks/shutdown, emit nothing → no span, per D-D's placement rule), co-located with `run_start`. Finalization is uniform, not per-return-site: `failRun`/`windDownRun` each gained a `span trace.Span` parameter and call `finalizeRunSpan` at the SAME point they already emit `run_end` — every one of Run's 9 `failRun`/`windDownRun` call sites plus the success tail funnels through this, so every exit is covered without a scattered per-site `span.End()`. Sets `gen_ai.operation.name`, `cachicamas.run.id`, `cachicamas.run.parent_id` (iff `RunStart.Parent()` reports present — always absent today, since `Harness.Run` only ever calls `NewRunStart`, never `NewDelegatedRunStart`), and at close `cachicamas.run.outcome` + `error.type` (iff failed) plus the error/ok status.
- [x] 5.3 In `Compact` (`compaction.go:413`), open a run span the same way (after both refusals, before `run_start`), finalized at its own two closing sites (the `compactErr != nil` arm, the success tail).
- [x] 5.4 Ran `-count=1` on `observability_parity_test.go` and `observability_lifecycle_test.go` — **all 7 lifecycle rows AND both parity tests are now GREEN** (every row's outermost bracket is the run bracket, which Phase 5 fully covers). `observability_nesting_test.go`'s `run_span_root_no_parent` subtest is also GREEN. Remaining RED (expected): `turn_span_parent_is_run_span`, `tool_span_parent_is_turn_span`, `delegation_child_run_span_parent_is_delegating_tool_span`, `compaction_span_recorded_standalone_root` (4 nesting subtests) and all of `observability_denylist_test.go` (attribute-count floor 10<14, missing `permission_decision_required` coverage) — both instrument in Phases 6–8 as designed.

**Discharges**: the run-span rows of `R-AGO-002`, `R-AGO-007`, `R-AGO-009`; `S-AGO-054`'s run-refusal-no-span clause.

## Phase 6 — Instrument `Turn` (turn span)

- [x] 6.1 In `loop.go`, open the turn span co-located with `turnStart`'s emission (line ~343 post-edit), nothing on the pre-emission refusals (`validateContinuation`, the nil-continuation `NewRunStart` failure, the `NewTurnStart` failure itself — per D-D's iff rule, a `NewTurnStart` construction failure opens no turn span either); additionally opens the run span on the nil-continuation path, ambiently acquired via `tracerFromContext(ctx)` (not h.TracerProvider — Turn has no Harness in scope), so a bare `Turn(context.Background(), ...)` call with no recording span on ctx records nothing (R-AGO-001).
- [x] 6.2 At turn close, sets `cachicamas.run.id`, `cachicamas.turn.id`, `cachicamas.turn.outcome`, `error.type` (iff aborted) from the SAME `TurnOutcome`/`*Failure` values each closing site already computes for the `turn_end` event it emits. **Finalizer discipline**: `emitPreStreamAbort` and `finishContinuationTurn` each gained `turnSpan`/`runSpan` parameters and finalize at the exact point they already emit `turn_end`/`run_end` — mirroring Phase 5's `failRun`/`windDownRun` pattern — so every one of Turn's 9 exit points (3 `emitPreStreamAbort` call sites incl. its own best-effort `perr!=nil` fallback, `finishContinuationTurn`, the nil-continuation completion tail, the mid-stream fatal block's two branches, the cancellation branch, the "ran to empty" tail) converges on a uniform close, never a scattered per-site `span.End()`.
- [x] 6.3 Confirmed: Layer 1's own `chat` request span (AI-37) nests under the turn span by construction — no code change needed. `ctx` is reassigned at both the run-span-open (nil-continuation only) and turn-span-open points, so every downstream call (`provider.Stream`, `Schedule`, `finishContinuationTurn`) already receives the span-carrying context by the time it runs.
- [x] 6.4 Ran `-count=1` on `observability_nesting_test.go` — the `turn_span_parent_is_run_span` subtest is now GREEN (parent-chain proven); `run_span_root_no_parent` stays GREEN. `tool_span_parent_is_turn_span`, `delegation_child_run_span_parent_is_delegating_tool_span` and `compaction_span_recorded_standalone_root` remain RED (Phase 7/8, as designed). **Bonus**: `observability_denylist_test.go`'s corpus-non-empty AND attribute-count-≥14 floors both now pass (turn span's own 2 keys push the recorded count past the vocabulary's cardinality) — only the event-kind-coverage guard (`permission_decision_required` missing, Phase 9's own explicit job per task 9.1) remains RED, exactly matching task 8.4's own stated expectation that Phase 9 — not Phase 8 — finishes the absence proof.

**Discharges**: the turn-span rows of `R-AGO-002`, `R-AGO-006` (turn's-parent-is-run), `R-AGO-007`, `R-AGO-009`; `S-AGO-054`'s turn-refusal-no-span clause.

## Phase 7 — Instrument `Scheduler.executeCall` + the detached arm (tool span)

- [ ] 7.1 In `scheduler.go`, open the tool span **after** the permission gate proceeds (`:433-440` is gated-out, no span, per D-D and `R-AGO-005` item 3 — no permission span ever), set `gen_ai.tool.name`, `gen_ai.tool.call.id`, `cachicamas.tool.ordinal` (widen `ToolStart.Ordinal()`'s `uint32` losslessly to `int64`).
- [ ] 7.2 Register the finalizer so every post-gate exit converges: start-constructor failure (`:456-463`), panic re-raise (`:499-501`), the detached arm (`:502-506`), `runErr` (`:514-518`), the four outcome-arm switch cases (`:524-557`). Set `cachicamas.tool.outcome`, `error.type` (iff execution failure) at each terminal write.
- [ ] 7.3 **The detached arm, decided explicitly (D-D)**: at `scheduler.go:502-506`, end the tool span at that exact point with error status, `error.type` = the typed detached cancellation category, `cachicamas.tool.detached=true`. Confirm the still-running goroutine inside `runToolWithWindDown` never receives the span handle — it must hold none.
- [ ] 7.4 Run `-count=1` on `observability_lifecycle_test.go`'s detached row (4.4) — the fixture must release and join the detached goroutine **before** `AssertAllEndedOnce()`, per `R-AGO-007`'s own rule; confirm exactly-once holds.
- [ ] 7.5 Run `-count=1` on `observability_nesting_test.go`'s tool-parent-is-turn assertion and the delegation-child-run-parent-is-tool assertion (via the delegating-tool fixture, child `Harness.TracerProvider` wired to the same `Provider`) — confirm GREEN.

**Discharges**: `R-AGO-002` (tool rows), `R-AGO-006` (tool's-parent-is-turn, delegation), `R-AGO-007` (all tool-exit rows incl. detached), `S-AGO-042`, `S-AGO-050`–`S-AGO-058` (detached rows), `S-AGO-054`'s gated-call-no-span clause.

## Phase 8 — Instrument `runCompaction` (compaction span, exactly one per bracket)

- [ ] 8.1 In `compaction.go`, open the compaction span at the `CompactionStarted` emission (`:265-267`); nothing on the pre-flight `Compact` refusals (already covered by 5.3).
- [ ] 8.2 Register the finalizer so every one of the twelve returns — each `emitCompactionFailedArm` call and the success tail (`:381-387`) — converges on it. Set `cachicamas.run.id`, `cachicamas.turn.id` (the minted compaction turn), `cachicamas.compaction.id`, `cachicamas.compaction.summary_id` (iff finished), `cachicamas.turn.outcome`, `error.type` (iff fail arm).
- [ ] 8.3 Confirm exactly ONE span covers the bracket — no separate turn span opened alongside it (`R-AGO-002`'s "one bracket, one span" clause).
- [ ] 8.4 Run `-count=1` across all four new test files. Every row of `observability_nesting_test.go`, `observability_lifecycle_test.go` and the run-family assertions of `observability_parity_test.go` should now be GREEN. `observability_denylist_test.go`'s corpus-non-empty and attribute-floor guards should now pass too (Phase 9 finishes the absence proof itself).

**Discharges**: `R-AGO-002` (compaction rows), `R-AGO-006` (compaction's-parent-is-requester), `R-AGO-007` (compaction rows), `S-AGO-019`, `S-AGO-041`, `S-AGO-053`, `S-AGO-054`'s compaction-refusal-no-span clause.

## Phase 9 — Denylist absence proof: non-vacuity guards and the recorded bite

- [ ] 9.1 Complete `observability_denylist_test.go`'s event-kind coverage guard (`R-AGO-008` item 4) and every-needle-non-empty guard (item 5); run `-count=1` and confirm the full absence scan passes GREEN with all five non-vacuity guards satisfied plus the exactly-once precondition (item 2, reusing `R-AGO-007`'s assertion on the same run).
- [ ] 9.2 **Bite (`S-AGO-028`)**: temporarily record `ToolStart.Arguments()` as a span attribute in `observability.go`'s tool-span setter. Run `-count=1`, observe the scan **FAIL** naming the tool-argument vector without reprinting the matched bytes. Record the red output, then revert the defeat, re-run `-count=1` GREEN.
- [ ] 9.3 Confirm the absence test's own source is not itself flagged by the sweep (`S-AGO-029`) — every needle is assembled at runtime, none is a contiguous literal.
- [ ] 9.4 Confirm no use of any per-field denied accessor (`R-AGO-004`'s table) reaches a span, and confirm `Span.RecordError` is called nowhere in `src/agent` production code (`S-AGO-026`).

**Discharges**: `R-AGO-004`, `R-AGO-005`, `R-AGO-008` in full; `S-AGO-024`–`S-AGO-031`, `S-AGO-070`–`S-AGO-075`.

## Phase 10 — Guard re-proof bites and closure-measurement evidence

- [ ] 10.1 **Bite (`S-AGP-027`/`S-AGP-028`/`S-AGP-029` reproof)**: with the widened allowlist in place, re-run the three existing scratch-import bites (application-layer path, `net/http`, vendor adapter subtree) and confirm each still FAILS on the same rule as before — the widened allowlist did not shadow the forbidden-prefix table.
- [ ] 10.2 **Bite (`S-AGP-035` / design bite, AG-22-specific)**: add a scratch production file in `src/agent/` importing `go.opentelemetry.io/otel/sdk/...`. Run the forward guard, confirm it FAILS **on the forbidden-prefix rule**, naming the offending path. Record the red output, remove the scratch file, confirm the tree is clean (`git status`).
- [ ] 10.3 **Bite (`S-AGO-094`)**: add a scratch production file in `src/agent/` importing a package under the ecosystem's SDK path (may reuse 10.2's file before removal). Confirm the forward import guard fails naming the path and the forbidden-prefix rule. Remove; confirm clean tree.
- [ ] 10.4 Confirm `git status --short backend/agent/go.mod backend/agent/go.sum` is empty and `git diff <base> -- backend/agent/src/ai/` is empty — re-running `TestHooks_S_AIV_032_ExclusionCheckedAgainstShippedCode` (`hooks_test.go:2386`) `-count=1` as the authoritative check.
- [ ] 10.5 Record this task's own fresh `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/agent/...` output (post all instrumentation), diff it against Phase 3's pre-instrumentation listing and the orchestrator's pre-spec addendum measurement; confirm no drift beyond the five allowlist entries.

**Discharges**: `S-AGP-027`–`S-AGP-030`, `S-AGP-035`, `S-AGO-093`, `S-AGO-094`.

## Phase 11 — S-AGS-068 (agent-v1-scope delta) and full-suite evidence

- [ ] 11.1 Write (or confirm covered by 9.x/10.x) an external `agent_test`-package test enumerating the shipped exported surface and asserting: exactly one telemetry-related exported member (`Harness.TracerProvider`, an OTel API provider interface type); no exporter, SDK type, provider-registration function, or process-global setter on the surface; a harness built with the field unset completes a full run normally with no panic and no span reaching any process-global-registered provider. This discharges `S-AGS-068` — the scenario that keeps `S-AGS-066` true.
- [ ] 11.2 Run the complete suite: `cd backend/agent && go test -race -v -count=1 ./...`. Record the **wall-clock duration** of this uncached run (a `(cached)` result is not evidence, per session discipline). Confirm PASS.
- [ ] 11.3 `golangci-lint cache clean && make lint`; `make build`; `make vuln-check`. Do **NOT** run `make all` (its `tidy`/`fmt` steps rewrite committed files and would trip the go.mod/go.sum freeze guard). Confirm `gofmt -l ./...` returns no output (check-only, non-mutating) and `go vet ./...` is clean.
- [ ] 11.4 Confirm a clean working tree (`git status --short`) with no scratch/bite artifacts left behind.

**Discharges**: `S-AGS-068`; the acceptance-criteria evidence-gate items of `agent-observability-boundary/spec.md`.

## Phase 12 — Doc 0003 back-annotation and OpenSpec archive (same PR)

- [ ] 12.1 In `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` line 3, change "**22 of 24**" to "**23 of 24**" and append an AG-22 sentence to the Wave-6 run-on narrative, in the same style as AG-21's, summarizing: the ADR 0005 § D3 vocabulary extension, the four instrumented seams, the no-tracer parity guarantee, and the absence proof — derived from re-reading this document's own Wave 0–6 table of contents, never restated from memory.
- [ ] 12.2 Run `sdd-archive`'s promotion step for this change: promote `agent-observability-boundary/spec.md` to `openspec/specs/agent-observability-boundary/spec.md` (new); merge the `agent-package-scaffold` MODIFIED block (`R-AGP-003`, `S-AGP-031`–`S-AGP-035`) and the `agent-v1-scope` MODIFIED block (`S-AGS-066` amendment, `S-AGS-068`) into their respective promoted specs, each as a range addition per `S-LSK-020` — never a restated total.
- [ ] 12.3 Move `openspec/changes/cachicamas-agent-observability/` to `openspec/changes/archive/` per the AG-14/AG-19/AG-20/AG-21 precedent; verify via `git diff --stat` that the promoted spec bodies gained the expected net-positive line counts (guard against sub-agent truncation, per prior-session discipline).
- [ ] 12.4 Confirm both delta specs' "Header maintenance obligation at promotion" notes are satisfied: `S-AGP-031`…`S-AGP-035` and `S-AGS-068` each appear wherever their host spec's scenario identifiers are enumerated, as ranges.

**Discharges**: the milestone doc-contract obligation and the archive step; not tied to a specific `R-AGO-`/`S-AGO-` ID.

---

## Scenario coverage map

| Requirement | Scenarios | Phase(s) |
|---|---|---|
| `R-AGO-001` | S-AGO-001…004 | 5, 6, 7, 8 (injection/no-global-getter), 4.3/10 (bite) |
| `R-AGO-002` | S-AGO-010…019 | 1 (ADR table), 5–8 (per-span rows) |
| `R-AGO-003` | S-AGO-020…023 | 1 (deferral note), 6 (no re-record verified) |
| `R-AGO-004` | S-AGO-024…029 | 1 (denylist restated), 9 (absence proof + bite) |
| `R-AGO-005` | S-AGO-030, 031 | 1 (list), 9 (verified absent) |
| `R-AGO-006` | S-AGO-040…044 | 2 (substrate), 5–8 (per-family parent), 4.1 (RED) |
| `R-AGO-007` | S-AGO-050…058 | 5–8 (per-family finalizers), 7.3 (detached), 4.4 (RED) |
| `R-AGO-008` | S-AGO-070…075 | 9 |
| `R-AGO-009` | S-AGO-080…083 | 5 (Started floor), 4.3 (RED), 6–8 (full-parity coverage) |
| `R-AGO-010` | S-AGO-090…094 | 3 (measurement + allowlist), 10 (bites) |
| `NFR-AGO-001` | — (behavioral, proven by `S-AGO-080`) | 5–8 |
| `NFR-AGO-002` | — (key-literacy, structural) | 3.1 (constants file) |
| `NFR-AGO-003` | — (substrate/boundary) | 3, 10.4 |
| `NFR-AGO-004` | — (test isolation) | 4, 9, 11.2 |
| `S-AGP-020`…`030` (unchanged, reproved) | — | 10.1, 3.3 |
| `S-AGP-031`…`035` | — | 3.2, 3.4, 10.2 |
| `S-AGS-065`, `S-AGS-067` (unchanged) | — | not touched — verified byte-identical at 12.2 |
| `S-AGS-066` (amended), `S-AGS-068` (new) | — | 1.1 (ADR back-annotation is separate from this spec but the surface claim), 11.1 |

No scenario in the three delta spec files is left unmapped.

## Key Learnings

1. The AG-03.2 guard's genuine RED comes from a real compile-and-run cycle over `observability.go`'s first OTel import, not from a hypothetical — Phase 3 sequences the constants file and the allowlist amendment as one commit precisely because splitting them either leaves an unimportable file or an unexercised allowlist row.
2. `tracetest.Span.SpanContext()` still returns an empty `trace.SpanContext{}` after Phase 2 — only `Parent()` gains real linkage; nothing in this milestone's proofs needs a non-zero W3C trace/span ID, so that method is deliberately left as-is.
3. The four proof-type test files (nesting/denylist/parity/lifecycle) are written once in Phase 4 and turned green incrementally across Phases 5–8, rather than one test file per seam — this is what makes each seam's RED-first bite genuine rather than a restated compile error.
4. `nested_run_test.go` (cited by design D-F) was not found as a literal filename in this worktree; Phase 4.1 carries an explicit apply-time verification step rather than assuming its existence.
5. Doc 0003's Wave-6 narrative lives on one physical markdown line, so its git diff line count will understate the actual prose added — flagged in the Review Workload Forecast rather than silently undercounted.
