# Exploration: AG-21 — Harden concurrency, backpressure and leaks

> Change `cachicamas-agent-concurrency-hardening` · Milestone AG-21 (Layer 2, Wave 6, milestone 22 of 24; doc `0003:1963-2043`)
> Worktree `cachicamas-worktrees/ag21-concurrency-hardening` · base `main@54476ded` (PR #182, AG-20)
> Depends on AG-14, AG-15, AG-16, AG-18, AG-19, AG-20. Closes `R-05` (invariant 3 under pressure) and "the assembled whole".

## 1. Milestone text (verbatim, doc 0003 lines 1963-2043)

Charter (1963-1973): Goal "Prove the assembled harness clean under adversarial schedules: no goroutine leaks on any exit path, lossless ordered delivery under slow consumers, race-free across every feature of waves 2-5 in combination." Deliverable: "A hardening suite over combined scenarios (suspension + interrupt + steering + compaction in one run), leak checks via the AI-22 leak-detection mechanism." Acceptance: full-package leak check passes; slow consumer loses nothing and observes contract order; combined-scenario matrix passes under race detector; abandoned consumer who cancels winds down within bounds. Out of scope: performance targets.

**AG-21.1** (1983-2011) — 12-row Scenario Outline (state x signal): states = {a suspension pending, steering queued, compaction mid-run, a child harness active} x signals = {interrupt, shutdown, provider failure}. Then: "the run completes its wind-down with valid history and contract-ordered events — the interactions no single-milestone test exercises." **Split if:** "any state row proves more than a sitting — rows become children (AG-21.1.x), one per state, same signals" (line 2010) — the milestone itself anticipates this may not fit one sitting.

**AG-21.2** (2012-2028) — two scenarios: "a stalled consumer loses nothing" (progress limited by "the delivery decision", sanctioned loss path if granted is the only loss) and "cancellation unblocks a stalled stream within bounds" (winds down within "the documented bound").

**AG-21.3** (2030-2041) — one scenario: "the whole package leaks nothing" via "the AI-22 leak-detection mechanism, wholesale", with "any detached-tool report from the bounded wind-down accounted, not leaked."

## 2. The AI-22 leak-detection mechanism

Doc 0002 lines 1329-1390 define AI-22 ("Build stream recording and assertion helpers"), leaf AI-22.4 = "Leak-detection approach `[decision]`" — chose a hand-rolled mechanism over `go.uber.org/goleak` (rejected: needs its own ADR per `openspec/AGENTS.md` rule 5; would break the package's dependency-free pin before AI-24). Amended 2026-08-10: narrowed to cancellation + abandoned-then-cancelled paths only; abandoned-never-cancelled is untestable to termination (documented, not asserted).

Implementation: `backend/agent/src/agenttest/stream_kit_leak.go`. Public API: `func RequireNoGoroutineLeak(tb testing.TB, scenario func())` (line 107). Mechanism: repeats `scenario` `leakRepeats=50` times (line 61), sleeps `leakSettle=50ms` (line 77), compares `runtime.NumGoroutine()` before/after, fails if growth exceeds `leakTolerance=leakRepeats/2=25` (line 68).

**It is a per-scenario opt-in wrapper around one callback, NOT a package-wide `TestMain` sweep** — there is no `TestMain` anywhere under `backend/agent` (grep for `func TestMain` returned zero files). It is **serial-only, mechanically enforced**: it calls `tb.Setenv(leakSerialOnlySentinel, "1")` (line 110), which the `testing` package itself panics on if the calling test or an ancestor has called `t.Parallel()` — so a test using this helper CANNOT be `t.Parallel()`. Test proof: `backend/agent/src/agenttest/stream_kit_leak_test.go`.

"Detached-tool report ... accounted, not leaked" concretely means: `R-CAN-006` (`openspec/specs/agent-cancellation-tree/spec.md:130-156`) requires a call still running past the 100ms wind-down bound to be reported via a typed execution-failure `Result` (`tool_end_execution_failure`, category `Cancellation`, cause = detached-call carrier, `errors.As`-extractable) rather than silently abandoned. `S-CAN-006` (spec.md:154) is the existing test pattern: "when the same scenario is repeated under the package's goroutine-leak harness, each iteration releasing its own blocked tool after the run returned, then no harness-owned goroutine survives the wind-down and the released third-party goroutine is accounted for — proven alive past the bound by the typed report and proven to exit once the third-party code returns — rather than excluded from the count. The leak-harness test MUST NOT call `t.Parallel()`."

## 3. Existing test surface under `backend/agent/src/agent/` (68 test files)

Fixture/harness fabric (in `backend/agent/src/agenttest/`, all reusable, no new production code needed):

- `agenttest.Gate` (`fake_gate.go`) — two-channel sync point (`Reached()`/`Release()`), the existing "hold and release" primitive used for suspension, steering, compaction-hold and AG-20's hook-stall proofs (`S-HKS-017`).
- `agenttest.Provider` + `agenttest.Script`/`Step` (`fake_provider.go`, `fake_script.go`) — scripted fake `ai.ModelProvider`; a `Step` is only `Emit(event)` or `Hold(gate)` — no third shape.
- Provider-failure injection is via `Emit` of an `ai.MidStreamFailure(...)` + `ai.ErrorEvent(...)` step (pattern in `backend/agent/src/agent/turn_failure_test.go:32-61`, `scriptTextThenTerminalFailure`) — no separate "Fail" primitive exists; failure is just another emitted event.
- Child-harness / delegation pattern: `delegatingTool` + a second `agent.Harness` value hosted inside a tool (`nested_run_test.go:14-40`).
- `RequireNoGoroutineLeak` (leak checks), as above.

Per matrix cell, what already exists:

| State | Existing driver | Signal | Existing driver |
|---|---|---|---|
| suspension pending | `harness_suspension_test.go` (permission-defer + `Gate`) | interrupt | `cancellation_interrupt_test.go` |
| steering queued | `harness_steering_test.go` (`heldTurnScript` + `Gate`) | shutdown | `cancellation_shutdown_test.go` |
| compaction mid-run | `compaction_call_test.go`, `compaction_stream_test.go` (strategy + `Gate`-holdable script) | provider failure | `turn_failure_test.go` pattern (`Emit(ErrorEvent)`) |
| child harness active | `nested_run_test.go` (`delegatingTool`) | — | — |

No single existing test exercises any of the 12 **combinations** (e.g. "compaction mid-run + provider failure", "child harness active + shutdown") — the charter's own words ("the interactions no single-milestone test exercises") are accurate. Every state and every signal individually has a ready-made fixture; combining them is new **test code composing existing primitives**, not new mechanism.

## 4. The delivery decision (AG-21.2)

Owning capability: `openspec/specs/agent-event-delivery/spec.md` (AG-01.1 decision + AG-20.2 back-annotation). Key requirements:

- `R-AGE-004` (line 72): "agent events are lossless... on every path other than the sanctioned loss paths this decision itself names."
- `R-AGE-005` (line 82): loop-internal boundary — Layer 1's rule verbatim: on cancellation with a saturated buffer, late events are dropped, stream closes without terminal event. This IS the sanctioned loss path AG-21.2's first scenario refers to ("the sanctioned loss path, if the decision granted one").
- `R-AGE-006` (line 92): harness-facing boundary is **strictly narrower** — MUST NOT drop an event describing a fact already committed to history; bounded wind-down MUST finish delivering every such event before/with run-end.
- `R-AGE-007` (line 110): states positively what remains droppable — facts the harness never learned before cancellation (in-flight events of work cut short).
- `R-AGE-008`/`S-AGE-010`/`S-AGE-030` (lines 121-164): the structural non-blocking-observer mechanism (lock-append enqueue, own-goroutine dispatch, value-stripped context, typed non-event stall report) — this is "invariant 3" that AG-21 is closing "under pressure" (doc 0003 line 2265: `R-05` register row).

Wind-down bound: `R-CAN-006` (`openspec/specs/agent-cancellation-tree/spec.md:130-156`) — "Wind-down MUST complete within a documented bound... a package constant serving as the default — `100ms`... overridable through a zero-default field on the caller-owned `Scheduler` value" (line 132, citing `stream_kit_leak.go:77`'s 50ms settle window as the jitter baseline). The three-way disjoint-set table (third-party tool code / harness-owned tasks / observing-hook code, spec.md:140-144) is exactly what AG-21.2's second scenario and AG-21.3's "detached-tool report... accounted, not leaked" clause are proving empirically.

## 5. Documented bound (AG-21.2 scenario 2 / AG-21.3)

Exact value: **100ms**, a Go package constant, armed only by cancellation (never on an uncancelled path — `R-CAN-006` paragraph 2, spec.md:134), overridable via a zero-default field on `*agent.Scheduler` (the injection seam tests already use, per `S-CAN-006`, spec.md:154). No single canonical constant name/file was located in this pass (the spec cites it as "a package constant" without naming the Go symbol) — the implementing symbol must be grepped in `backend/agent/src/agent/scheduler.go` or `harness.go` before design; not confirmed by file:line in this exploration.

## 6. Race-detector and leak-check plumbing today

- `backend/agent/Makefile:109-110` — `make test` runs `go test -race -v ./...`. **`-race` is already the default/only test invocation** (no non-race target exists). `make test` does NOT pass `-count=1`, so a cached run is possible and must never be accepted as evidence.
- No `TestMain` exists anywhere under `backend/agent` (grep returned zero matches) — no package-level goroutine-count hook.
- No `goleak` (or any third-party leak detector) is wired anywhere — confirmed rejected by design (AI-22.4); the module is dependency-free.
- Real uncached runtime: AG-20's own `tasks.md` header (`openspec/changes/archive/2026-08-20-cachicamas-agent-hook-taxonomy/tasks.md:5`) records: "Runner: `cd backend/agent && go test -race -count=1 ./...`, **`-count=1` mandatory** (~170s uncached)." AG-21 adds 12 combined scenarios + 2 pressure scenarios + 1 leak-sweep wrapper on top of that baseline, so real runtime is likely materially higher than 170s.

## 7. Prior-art precedent (AG-19, AG-20 archived changes)

- Directory shape: `openspec/changes/archive/{YYYY-MM-DD}-{change-name}/{explore,proposal,design,tasks,apply-progress,verify-report,archive-report}.md` + `specs/{capability}/spec.md` per touched capability.
- ID convention: `R-{PREFIX}-0NN` requirements, `S-{PREFIX}-0NN` scenarios, append-only, one prefix per capability (e.g. `R-CAN-`, `R-AGE-`, `R-HKS-`, `R-DEL-`). Spec headers state "**Allocated IDs**: ... This header states the allocated range and never a total" (`agent-cancellation-tree/spec.md:8`) — never claim a total count.
- Cross-milestone amendment shape: a later milestone that changes an earlier capability's requirement writes a **delta** file under its own change's `specs/{capability}/spec.md` reproducing the **MODIFIED requirement in full** (not a diff), with a "Not modified, and why" table, a "(Previously: ...)" paragraph recording the prior text, and a "Header maintenance obligation at promotion" note telling `sdd-archive` which new `S-*` IDs to add to the target spec's Allocated IDs line. AG-20's delta to `agent-cancellation-tree` (discovered mid-`sdd-verify`, CRITICAL-2) is the closest precedent for what AG-21 will likely need to do again to `R-CAN-006`/`R-CAN-008`.
- doc-0003 + counter update lands in the **same PR** as the SDD cycle. The checkbox to flip is doc 0003 line 2179: `- [ ] The package is race-clean and leak-free under the combined-scenario matrix — closed by AG-21.`
- Delivery/budget precedent: AG-20's `tasks.md` used `delivery: single-pr (exception-ok)`, 1000-line budget excluding `openspec/`, with a **pre-accepted extension** because its own forecast (1295-1755 lines) exceeded 1000.

## 8. Capability landscape — likely amend targets

- `openspec/specs/agent-cancellation-tree/spec.md` — near-certain amend target: `R-CAN-006` (wind-down bound / disjoint-set table) may need a fourth row or scenario if the 12-cell matrix's "compaction mid-run" or "child harness active" combos surface a goroutine class not yet enumerated; `R-CAN-008`/`L2C-08` doc-contract row may need widening again (identical mechanism AG-20 already used twice).
- `openspec/specs/agent-run-driver/spec.md:87` — `R-RUN-001` explicitly states **"Cross-run transcript state remains AG-21's"** — a live, unaddressed reference that does NOT appear anywhere in the AG-21 charter's three leaves. A genuine open scope question (§10).
- `openspec/specs/agent-event-delivery/spec.md` — likely amend or at least back-annotation target for `R-AGE-004`/`R-AGE-008` ("invariant 3 under pressure" — doc 0003 line 2265 names this as what AG-21 closes on the R-05 register).
- Possibly a **new** capability `agent-concurrency-hardening` (matching the change name) to hold AG-21-specific requirements (the combined-matrix and pressure-test obligations themselves), following the AG-14/AG-19/AG-20 precedent, with cross-cutting deltas to the above three.

Absolute-quantifier / count-phrasing risks — concretely at risk of falsification by AG-21's own combined matrix:

- `R-CAN-006`'s three-row disjoint-set table ("every goroutine the package itself owns has exited") — AG-20 already had to widen this once (two rows -> three); a fourth combined-state discovery would require the identical widen-not-reword treatment.
- `R-RUN-001`'s "One run at a time per harness value... Concurrent runs on one value stay out of scope" — AG-21's race-detector matrix must not silently start testing concurrent `Run` calls on one value.
- `R-CAN-004`'s "A second signal fired... MUST be a no-op" — the matrix's interrupt/shutdown/provider-failure combinations must respect the already-decided first-cause-wins rule rather than re-deciding it.

## 9. Sizing

No production code is anticipated (see §10) — the entire diff is new **test files** (agent package) plus doc 0003's checklist/status-line update. Estimate:

- 12 matrix-cell tests (AG-21.1), plausibly 60-110 lines/cell including local fixture glue → **720-1320 lines**.
- 2 pressure scenarios (AG-21.2) needing a "stalled consumer" harness (a Gate-blocked sink read) → **150-300 lines**.
- 1 combined leak-sweep wrapper (AG-21.3) reusing `RequireNoGoroutineLeak` → **80-150 lines**.
- Doc 0003 checklist + status-line + counter update → **10-25 lines**.
- **Counted total estimate: ~960-1795 lines**, excluding `openspec/**`.

This straddles or exceeds the 1000-line budget on the same shape AG-20 did. Two structurally sound splits exist in the charter itself: (1) the milestone's own escape hatch (line 2010) splitting AG-21.1 into 4 children, one per state; (2) shipping AG-21.1 and AG-21.2+AG-21.3 as two changes.

## 10. Open questions for the proposal phase

1. **Is any production code needed at all?** Every fixture already exists and composes to drive all 4 states x 3 signals. Working hypothesis: **AG-21 is a pure test-hardening milestone** — UNLESS the combined matrix surfaces a genuinely new interaction the existing production code mishandles. AG-20's own history (the CRITICAL-2 mid-verify discovery of the observer-lane goroutine class) is precedent that a "prove it under combination" milestone CAN surface a real spec gap. The proposal must state this hypothesis explicitly and commit to fixing in place, not deferring, if a cell fails.
2. **The "Cross-run transcript state remains AG-21's" reference** (`agent-run-driver/spec.md:87`) is live in the current promoted spec and cites no AG-21 charter leaf. Either (a) it is already fully discharged by AG-14's serial-reuse scenarios (`S-RUN-003`/`S-CAN-002`) and this reference is stale and should be corrected at archive, or (b) AG-21 must add an explicit scenario proving transcript state does NOT carry over between two serial runs on one harness value under adversarial (cancelled) conditions. Unresolved; must be decided before `sdd-spec`.
3. **"Wholesale" application of `RequireNoGoroutineLeak` (AG-21.3) cannot mean literally wrapping the entire pre-existing `go test ./...` suite** as one repeated scenario: (a) that suite is saturated with `t.Parallel()`, and `RequireNoGoroutineLeak`'s own `tb.Setenv` call panics under a parallel ancestor; (b) repeating a ~170s-uncached suite 50x is computationally infeasible. The only coherent reading is: build ONE new top-level (non-parallel) test that runs the full **combined-scenario matrix + pressure cells this change adds** as its `scenario func()`, and pass THAT to `RequireNoGoroutineLeak`.
4. **The exact wind-down-bound Go constant name/location** was not pinned to a file:line in this pass — must be grepped in `scheduler.go`/`harness.go` at design time.
5. **R-05's "under pressure" closure**: doc 0003's register (line 2265) treats AG-21 as closing `R-05` empirically rather than by new decision. If AG-21.1/2 pass cleanly, a back-annotation paragraph on `agent-event-delivery` may be sufficient rather than a MODIFIED requirement — confirm rather than assume.

## Recommendation

Treat AG-21 as primarily a **test-authoring milestone** composing existing `agenttest` primitives, with a **contingent** production-code path only if a matrix cell fails. The proposal should: (a) decide the split question against the measured, not estimated, line count; (b) explicitly scope AG-21.3's "wholesale" leak check as a new non-parallel wrapper test over the new combined scenarios; (c) explicitly resolve the `agent-run-driver` "cross-run transcript state" reference as discharged-by-citation or as a new scenario; (d) record the user's pre-authorized budget extension rather than re-asking.

## Ready for proposal

Yes — with the three decisions above made explicitly rather than silently.
