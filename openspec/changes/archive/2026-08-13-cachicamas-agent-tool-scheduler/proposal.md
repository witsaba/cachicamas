# Proposal: AG-09 — Define the tool execution contract and scheduler

## Intent

AG-09 ships the **execution contract and scheduler** for the agent runtime. It closes **G5** (R-13: parallel tool execution with deterministic call-ordered rejoin + per-tool concurrency policy) and **R-18 seams 2 & 3** of v2 § 6 — seam 3 is the **opaque per-call policy slot** Layer 2 carries but never reads (the sandbox seam: "confinement is a property of the call site, not of the code being called", `0001:613-616`). This is the **wording trap** doc 0003:109 calls out: the loop does NOT execute tools. The loop **schedules** execution against an injected contract and drives the permission protocol around it. What a tool does, whether it is allowed, and under what confinement it runs are all decided above; the loop owns ordering, concurrency, suspension, and the rejoin. AG-09 ships **the contract + the scheduler**; **AG-10 wraps the permission protocol** around it; **AG-13 iterates** the model ↔ tools ↔ model cycle. AG-09 ships ONE cycle per `Turn`. Iterating the cycle is AG-13's `Harness` job.

## Why now

AG-05.2 shipped the three-state tool event family (PR #164). AG-07 shipped the walking skeleton (PR #167). AG-08 shipped the pre-request hook seam (PR #168, branch base `e27e8411`). AG-09 is the first Layer 2 milestone that **actually invokes tools**: it widens `loop.go:468-472`'s default-branch `translate()` switch to consume AI-18's `ToolCallStart/Delta/End` events and dispatches them to the scheduler. AG-10 blocks on AG-09 (the permission protocol wraps each scheduled call). AG-11 blocks on AG-09 (turn termination types the scheduler's outcomes). doc 0004's built-in tools implement the AG-09 contract.

## Scope

### In scope

- **The tool contract** (AG-09.1): declaration, effect class (`Read | Mutating | Execute`), opaque policy slot, typed `Result` mirroring AG-05.2's `ToolOutcome` family. Three-method interface (`Name()`, `EffectClass()`, `Run(ctx, args, policy) (Result, error)`).
- **Concurrency policy** (AG-09.2): reads concurrent (bounded fan-out), mutating + execute serialized in call order, start events emitted at execution start (not at rejoin).
- **Ordered rejoin** (AG-09.3): call-order semantics regardless of completion order; Layer 1 call/result correlation identities preserved.
- **Failure isolation** (AG-09.4): one bad tool → siblings complete, typed result in its call position, panicking tool contained under `-race`.
- **Tool execution events** emitted per AG-05.2 (`ToolStart` / `ToolProgress` / `ToolEndSuccess` / `ToolEndResultFailure` / `ToolEndExecutionFailure`). AG-09 consumes the family; does not modify it.
- **Loop wire-up**: `TurnOptions.Tools map[string]Tool` (zero-value safe; mirrors AG-08 `PreRequestHook` precedent). `Turn` calls `Schedule` once per turn when the provider stream closes with `FinishReasonToolCalls`; AG-13 iterates.

### Out of scope

- **Permission protocol** (AG-10) — wraps the scheduler around each call; its scope.
- **Iteration of the model ↔ tools ↔ model cycle** (AG-13) — the wording trap. AG-09 ships ONE cycle; AG-13 iterates.
- **What any tool does** — Layer 3 built-in tools (doc 0004).
- **Sandbox semantics** — Layer 3 interprets the opaque policy slot; AG-09 forwards it byte-exact.
- **Subagent tool** — v1 non-goal per doc 0003 § 8.
- **The four-hook taxonomy** (AG-20) — AG-09 ships `Tools` as a map; AG-20 widens to a `ToolSource` port if/when needed.

## Capabilities

### New

- **`agent-tool-scheduler`** — the execution contract (`Tool` interface, `EffectClass` enum, `PolicySlot` opaque type, `Result` typed outcome) and the scheduler (`Schedule(ctx, calls, policy, sink) []Result`). New `openspec/specs/agent-tool-scheduler/spec.md` (full spec, requirements `R-TLS-NNN` / scenarios `S-TLS-NNN` per AG-09's per-milestone prefix).

### Modified

- **`agent-loop-skeleton`** — `TurnOptions` gains a `Tools map[string]Tool` field (non-breaking zero-value extension; mirrors AG-08's `PreRequestHook` field precedent). `Turn` widens `translate()`'s default branch (`loop.go:468-472`) to consume AI-18 `ToolCallStart/Delta/End` events and dispatch them to `Schedule`. Delta spec at `openspec/changes/cachicamas-agent-tool-scheduler/specs/agent-loop-skeleton/spec.md` adds requirements to the AG-07 skeleton spec; the skeleton's NFR-LSK-003 substrate-untouched list extends by one entry: `loop_tool_dispatch_test.go` joins the filter widening pattern (precedent: AG-07 W3 → AG-08 W3 → AG-09).

### None

- **`agent-message-tool-events`** — AG-05.2 already shipped the three-state tool event family AG-09 consumes. AG-09 does NOT add kinds, fields, or invariants to the family. No spec change.
- **`agent-pre-request-hook`** — AG-08's `PreRequestHook` is orthogonal to `Tools`; no spec change.

## Approach

Introduce `Tool` (interface), `EffectClass` (typed enum, 3 members), `PolicySlot` (`type PolicySlot any` — named type over `any`, the "Layer 2 never reads" promise is the type's doc + a guard test), and `Result` (3 outcomes mirroring `ToolOutcome`) in `backend/agent/src/agent/tool.go`. The scheduler (`backend/agent/src/agent/scheduler.go`) is **hand-rolled** — `chan struct{}` semaphore for reads (bounded fan-out), single-goroutine + channel-of-one for mutating + execute (serialized, call order), `[]Result` indexed by ordinal for rejoin, `defer/recover` per call goroutine for panic containment. **No new top-level deps** (`errgroup` is forbidden — new dep + first-error cancellation conflicts with AG-09.4 #1 "siblings complete"; `openspec/AGENTS.md` `## Hard rules`). The scheduler consumes AG-05.2's event family; emits `ToolStart` per call at execution start (not at rejoin — AG-09.2 #3 "a frontend shows live progress in true order"). The loop's `Turn` widens `translate()`'s default branch to convert AI-18 tool-call events into a `[]ScheduledCall`; on `FinishReasonToolCalls` it calls `Schedule`, then emits each event through the existing `emitStamped(sink, stamper, ev)` path. The `LaneStamper` single-writer invariant (`sequence.go:8-24`) is preserved: the scheduler emits through one dispatcher goroutine, not per call. Test substrate: scripted tool helper in `agenttest/scripted_tool.go` — configurable per-call behavior (success / result-failure / execution-failure / panic / delayed completion), records start time + policy value for assertions. Implementation details in the design phase; this proposal stays at the right altitude.

## Decisions log

| # | Decision | Adopted value | Rationale |
|---|---|---|---|
| **D1** | `Tool` interface shape | Three-method interface: `Name() string`, `EffectClass() EffectClass`, `Run(ctx, args, policy) (Result, error)` | Mirrors `ToolOutcome`'s typed-result posture (R-AMT-006). Explicit policy slot in signature (seam 3 demand). `EffectClass()` lets the scheduler branch without calling `Run`. |
| **D2** | `EffectClass` shape | Typed enum `type EffectClass uint8` with `EffectClassRead` / `EffectClassMutating` / `EffectClassExecute`; closed; `String()` method | Mirrors `ToolOutcome` posture (tool_event.go:227-246). Charter floor "at minimum read, mutating, execute". A future ADR can add `EffectClassNetwork` without breaking signatures. |
| **D3** | `PolicySlot` shape | `type PolicySlot any` (named type over `interface{}`) | Documents the "opaque to Layer 2" promise at the call site. Scheduler never type-asserts the value (enforced by source guard test). Charter seam 3: "confinement is a property of the call site". |
| **D4** | Scheduler concurrency primitives | Hand-rolled: `chan struct{}` semaphore (reads), channel-of-one + single goroutine (mutate/execute), indexed `[]Result` (rejoin), `defer/recover` per call (panic containment) | No new top-level deps (NFR-PRH-002 carry; `openspec/AGENTS.md` hard rule). Explicit concurrency shape matches the charter's three policies. Indexed slice is the cleanest "rejoin in call order regardless of completion order". |
| **D5** | Rejoin shape | Indexed result slice: `results := make([]Result, len(calls))`; each call goroutine writes `results[call.ordinal] = res`; panic containment writes a typed `ToolEndExecutionFailure` into the slot before returning | O(n) memory; rejoin loop is slice iteration; every ordinal slot is always populated. |
| **D6** | Panic containment | Explicit `defer func() { if r := recover(); r != nil { results[ordinal] = Result{Outcome: ExecutionFailure, Failure: ...} } }()` in every call goroutine | Stdlib-only. Sibling results preserved. Typed failure constructed before recover returns. Race-detector clean (AG-09.4 #2's proof: panicking scripted tool → typed event + siblings complete + `runtime.NumGoroutine()` baseline). |
| **D7** | Spec prefix | `R-TLS-` / `S-TLS-NNN` (tool-scheduler) | Two-letter match to slug. Per-milestone prefix continues AG-04 (`R-AEV-`) / AG-05 (`R-AMT-`) / AG-06 (`R-APE-`) / AG-07 (`R-LSK-`) / AG-08 (`R-PRH-`). |
| **D8** | Loop wire-up | `Turn` calls `Schedule` once per turn; **AG-13 owns iteration** | The wording trap is the design. AG-09 ships ONE cycle; AG-13 (harness) loops model ↔ tools ↔ model. AG-11 owns finish-reason dispatch. |
| **D9** | `TurnOptions.Tools` shape | `Tools map[string]Tool` — keyed by name; nil zero value = no tools (the scheduler returns typed `ExecutionFailure` results in their ordinal slots — consistent with AG-09.4 #1) | O(1) resolution. Matches model's name → tool lookup. AG-13 may widen to a `ToolSource` port without changing the field's signature. |

## Carry-forwards from AG-04/05/06/07/08

| Source | Finding | AG-09 mitigation |
|---|---|---|
| **AG-08 W1** | Back-pressure path unproven (only one unbuffered-sink test, AG-08 S-PRH-007) | **MUST** add ≥1 scheduler test with unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline. The scheduler's fan-out is the natural place to prove concurrent reads under back-pressure. |
| **AG-08 W2** | NFR-PRH-002 spec-prose drift (corrected at archive time) | NFR-TLS-001 inherits: every behavioral test in `package agent_test`. |
| **AG-08 W3** | Substrate-untouched env-var fallback (`AG08_BASE_REF`) | NFR-TLS-002 inherits: `AG09_BASE_REF` env-var + dynamic `git merge-base HEAD origin/main`. |
| **AG-08 SUGG 1** | `drainSink` (`loop_test.go:147`) has no timeout | **MUST carry** — AG-09 is the first named consumer of the scheduler's results via `sink`. Add `select` deadline inside `drainSink` once; closes AG-07 SUGG 1, AG-08 SUGG 1, and AG-09's need. |
| **AG-08 SUGG 2** | Helper promotion (`loopRequestSystemText` etc.) | Defer — AG-09 introduces `ordinalFromToolStart` local to `loop_tool_dispatch_test.go`; promotion at AG-23. |
| **AG-08 SUGG 3** | `applyPreRequestHook` 100% covered by bites | `Schedule` should be similarly covered by `S-TLS-004a/b` bites before `S-TLS-004` GREEN — defense against vacuous scheduling. |
| **AG-05 W1** | Vacuous reconstruction helper | Apply bite pattern to `R-TLS-006` (rejoin in call order): bite (a) out-of-order completions → rejoin still in order; bite (b) inverted completions → rejoin still in order. Both bites RED-recorded BEFORE `S-TLS-006` GREEN. |
| **AG-05 W2** | Reconstruction property AG-05.3 | AG-09 is the **second live consumer** of AI-18 streamed tool-call events (after AG-05.3's reconstruction property). The AG-09 dispatch must widen `translate()`'s switch on `ai.EventKindToolCallStart / Delta / End`. |
| **AG-04 W8** | Loop coverage ≥ 80% | Forecast ~150 lines added to `loop.go`; current coverage 86.13% (AG-08 final); AG-09 must hold ≥ 80%. |
| **AG-04 W9** | Scenario count drift | State identically: **11 charter scenarios → ~14–18 spec + ~6 bites**. |

## Forecast

**Estimated changed lines: ~2,380** (forecast ~2,280 new + ~100 modified). braejan pre-authorized `size:exception` up to 1000 lines with the same directive as AG-08 (AG-08 shipped 2,315 lines single PR, merged). **Single PR delivery** — NOT chained PRs. The doc 0003 mermaid (AG-09.1 → AG-09.2 → AG-09.3; AG-09.1 → AG-09.4) defines a DAG with a small overlap (AG-09.2 and AG-09.4 both depend on AG-09.1), but the substrate preservation rule, the unified test substrate, and the loop wire-up (AG-09.4's `loop_tool_dispatch_test.go`) couple the leaves enough that splitting would land AG-09 with one PR still exercising AG-09.4's `loop.go` edit + tests while waiting for AG-09.1/AG-09.2/AG-09.3. Single PR keeps the substrate-untouched test meaningful (the test can be widened once in the same commit that adds `loop_tool_dispatch_test.go`).

| File | Action | Forecast |
|---|---|---|
| `backend/agent/src/agent/tool.go` | NEW | ~180 |
| `backend/agent/src/agent/tool_test.go` | NEW | ~280 |
| `backend/agent/src/agent/scheduler.go` | NEW | ~340 |
| `backend/agent/src/agent/scheduler_test.go` | NEW | ~900 |
| `backend/agent/src/agent/loop.go` | MODIFY | +~150 (Tools field + translate() widening + Schedule call site) |
| `backend/agent/src/agent/loop_tool_dispatch_test.go` | NEW | ~280 |
| `backend/agent/src/agent/loop_test.go` | MODIFY | +~5 (substrate-untouched filter widening) |
| `backend/agent/src/agent/agent_test_helpers_test.go` | MODIFY | +~15 (`ordinalFromToolStart` helper) |
| `backend/agent/src/agent/loop_test.go` `drainSink` | MODIFY | +~3 (select deadline) |
| `backend/agent/src/agenttest/scripted_tool.go` | NEW | ~180 |
| `backend/agent/src/agenttest/scripted_tool_test.go` | NEW | ~120 |
| 21 substrate files (AG-07 `R-LSK-004` list) | UNTOUCHED | 6th consecutive "substrate untouched" milestone |
| `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` | UNTOUCHED | No new deps |
| `openspec/specs/agent-tool-scheduler/spec.md` | NEW (sdd-spec) | Full spec for new capability |
| `openspec/changes/cachicamas-agent-tool-scheduler/specs/agent-loop-skeleton/spec.md` | NEW (sdd-spec) | Delta spec for modified capability |
| `openspec/changes/cachicamas-agent-tool-scheduler/{proposal,design,tasks}.md` | NEW | Phase artifacts |

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/tool.go` | NEW | `Tool` interface (3 methods); `EffectClass` typed enum (3 members); `PolicySlot` named type over `any`; `Result` value type (3 outcomes mirroring `ToolOutcome`); constructor validations |
| `backend/agent/src/agent/tool_test.go` | NEW | Charter AG-09.1: contract from outside; policy passthrough opaque; result vs execution-failure distinct |
| `backend/agent/src/agent/scheduler.go` | NEW | `Schedule(ctx, calls, policy, sink) []Result` — hand-rolled: semaphore + serialized channel + indexed slice + `defer/recover` |
| `backend/agent/src/agent/scheduler_test.go` | NEW | Charter AG-09.2 / 09.3 / 09.4: bounded fan-out, reads concurrent, writes serialized, ordered rejoin, correlation survives, panic isolation, typed result in call position |
| `backend/agent/src/agent/loop.go` (TurnOptions + translate() + Schedule call) | MODIFY | `Tools map[string]Tool` field; widen `translate()` default branch on AI-18 tool-call events; between `provider.Stream` close and `finalize`, call `Schedule` once and emit its returned events through `emitStamped` |
| `backend/agent/src/agent/loop_test.go` (filter widening + drainSink) | MODIFY | Widen `TestTurn_SubstrateUntouched` to also exclude `loop_tool_dispatch_test.go`; add `select` deadline to `drainSink` |
| `backend/agent/src/agent/loop_tool_dispatch_test.go` | NEW | Loop wire-up: provider streams tool-call events → loop calls `Schedule` → emits tool events on `sink` |
| `backend/agent/src/agent/agent_test_helpers_test.go` | MODIFY | Add `ordinalFromToolStart(events []agent.Event, callID string) uint32` helper |
| `backend/agent/src/agenttest/scripted_tool.go` | NEW | In-memory `agent.Tool` implementation: configurable behavior per call (success / result-failure / execution-failure / panic / delayed completion); records start time + policy value |
| `backend/agent/src/agenttest/scripted_tool_test.go` | NEW | Self-test: scripted tool's own discipline |
| 21 substrate files (AG-07 `R-LSK-004`) | UNTOUCHED | 6th consecutive "substrate untouched" milestone |
| `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` | UNTOUCHED | No new deps; no AG-03 guard edit |
| `openspec/specs/agent-tool-scheduler/spec.md` | NEW (sdd-spec) | New capability spec, full |
| `openspec/changes/cachicamas-agent-tool-scheduler/specs/agent-loop-skeleton/spec.md` | NEW (sdd-spec) | Delta spec for modified capability (Tools field + Schedule call) |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| **R1** — Size (~2,380 lines forecast, over 1000-line budget) | Medium | `size:exception` pre-authorized (AG-04/05/06/07/08 standing rule; AG-08 merged at 2,315 single PR). Single PR delivery — NOT chained (substrate preservation + filter widening + loop wire-up couple the leaves). |
| **R2** — Wire-up scope creep into AG-13 territory (loop iterates model ↔ tools ↔ model) | **CRITICAL if not surfaced** | D8 (one cycle's tools; AG-13 iterates). Surface explicitly: "AG-09 ships ONE cycle; iteration is AG-13's job." `Schedule`'s signature is the seam — callable from `Turn` (AG-09) or `Harness` (AG-13). |
| **R3** — Layer 1 effect class collision (a future Layer 1 milestone adds effect class to `ai.Tool`) | Medium | AG-09 owns the Layer 2 effect class registry (`Tool.EffectClass()` reads from the registry; `ai.Tool` stays untouched). A future Layer 1 amendment can move the type up. ADR-able cross-layer concern. |
| **R4** — Zero-value `Tools` (model emits tool call, no tool registered) | Medium | AG-09 ships the typed-failure path: scheduler returns `ToolEndExecutionFailure` with `*Failure{Category: ai.FailureCategoryUnsupportedCapability}` for each orphaned call. Consistent with AG-09.4 #1 "one bad tool does not abort the turn". |
| **R5** — Policy slot opaque-to-Layer-2 promise violation (future contributor type-asserts in scheduler) | Medium | (a) `type PolicySlot any` doc; (b) scheduler never type-asserts the value; (c) test asserts scripted tool receives the EXACT value injected byte-for-byte (`bytes.Equal`); (d) source guard test scans `scheduler.go` for any type assertion on `PolicySlot`. Pattern: AG-04's "no payload-convention inference" guard for `ToolEndSuccess` vs `ToolEndResultFailure` (R-AMT-006, S-AMT-050). |
| **R6** — Concurrent reads under `-race` (shared state touched without sync) | Medium | `[]Result` slice is the only shared state. Writes via `atomic.StoreUintptr` (per-slot) or mutex-guarded slice. Scripted tool's own state uses `sync.Mutex`. Each emit goes through ONE dispatcher goroutine (the `LaneStamper` single-writer invariant from `sequence.go:8-24` is preserved). |
| **R7** — Substrate-untouched streak: filter widening pattern (cumulative by AG-23) | Low | Pattern is the file-granularity escape hatch AG-07 W3 established. AG-09 widens to exclude `loop_tool_dispatch_test.go`. Consolidation deferred to AG-23 (matches AG-08 SUGG 2 deferral). |
| **R8** — `drainSink` timeout carry-forward | Low | AG-09 is the first named consumer of the scheduler's results via `sink`. Add `select` deadline inside `drainSink` once — closes AG-07 SUGG 1 + AG-08 SUGG 1 + AG-09. |
| **R9** — Review budget: AG-08 merged at 2,315 lines; AG-09 forecast ~2,380 | Low | `size:exception` pre-authorized. braejan's directive: single PR + same threshold as AG-08. Review focus: the contract surface (Tool interface, EffectClass, PolicySlot) and the loop wire-up (Turn's Schedule call site). |

## Dependencies

- **AG-05.2** tool execution event family — already shipped at `tool_event.go:1-501` (PR #164). Three-state `ToolOutcome` enum at `tool_event.go:227-246`. AG-09 consumes the family; does not modify it.
- **AG-07** walking skeleton — already merged at `93077c07` (PR #167). `Turn(ctx, provider, system, transcript, opts, sink)`, `TurnOptions`, `buildLoopRequest`, `provider.Stream` call site, `translate()`, `finalize()`.
- **AG-08** pre-request hook seam — already merged at `e27e8411` (PR #168). `TurnOptions.PreRequestHook`; AG-08's `applyPreRequestHook` is the precedent for AG-09's per-`Turn` once-shot dispatch pattern.
- **AI-21** `agenttest.Provider.Requests() []ai.Request` at `fake_provider.go:157-161` — already shipped.
- **AI-22** stream kit (`agenttest.Script`, `Emit`, `Hold`, `NewIter`) — already shipped.
- **AI-18** streamed tool-call events at `tool_call_event.go:74-336` — already shipped.
- **AG-03** boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) — must stay green untouched.

## Acceptance criteria

- [ ] `Tool` interface exposes declaration + effect class from an external package (AG-09.1 #1)
- [ ] The policy slot passes through byte-exact (`bytes.Equal` in scripted tool; source guard test for type-assert absence in `scheduler.go`) (AG-09.1 #2)
- [ ] Result and execution failure are distinct typed outcomes mirroring AG-05.2's `ToolOutcome` (AG-09.1 #3)
- [ ] Reads run concurrently with bounded fan-out (`runtime.NumGoroutine()` assertion); mutating + execute serialize in call order (AG-09.2 #1, #2)
- [ ] Start events emitted at execution start (not at rejoin) — observable on `sink` (AG-09.2 #3)
- [ ] Completions rejoin in call order regardless of completion order (AG-09.3 #1, both bites RED-recorded before GREEN)
- [ ] Layer 1 call/result correlation identities preserved exactly (AG-09.3 #2)
- [ ] One bad tool does not abort the turn: siblings complete, typed result in call position (AG-09.4 #1)
- [ ] Panicking tool contained under `-race`: typed `ToolEndExecutionFailure` event, `runtime.NumGoroutine()` returns to baseline (AG-09.4 #2)
- [ ] All tests green under `cd backend/agent && make test` with `-race -v ./...`
- [ ] `make lint` green, `make build` green, `make vuln-check` clean
- [ ] `backend/agent/go.mod` and `go.sum` byte-identical to `e27e8411`
- [ ] 21 substrate files (AG-07 `R-LSK-004` list) byte-unchanged — **6th consecutive "substrate untouched" milestone**
- [ ] `loop.go` line coverage ≥ 80% (AG-04 W8 carry)
- [ ] AG-03 boundary guards stay green untouched
- [ ] **11 charter scenarios → ~14–18 spec + ~6 bites** stated identically with proposal, tasks, apply-progress, verify-report
- [ ] AG-08 W1 carry: ≥1 scheduler test uses unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline
- [ ] AG-08 SUGG 1 closed: `drainSink` has a `select` deadline

## Rollback plan

Single PR — easy to revert. If before merge: `git revert <merge-or-commit-sha>`, delete branch, close PR. If after merge: `git revert -m 1 <merge-sha>` (mainline revert — AG-09 has no merge commit of its own dependencies; main is the only parent). Modified files: `loop.go` + `loop_test.go` (filter widening + drainSink deadline) + `agent_test_helpers_test.go` (`ordinalFromToolStart`) + 7 new files (3 in `backend/agent/src/agent/`, 2 in `backend/agent/src/agenttest/`, 2 in `openspec/changes/.../specs/...`). **AG-09 does not modify any Layer 1 substrate or any merged Wave 1 file** — the blast radius is limited to AG-09's own additions plus the two narrow modifications to `loop.go` + `loop_test.go` (both with AG-07/AG-08 precedent). No data migration, no schema change. `TurnOptions` gains one field (`Tools`); reverting restores AG-08's surface verbatim. The 6th consecutive substrate-untouched bet remains intact on revert.

## Next step

Launch `sdd-spec` next — write `R-TLS-001..NNN` + `S-TLS-001..NNN` (**~14–18 scenarios + ~6 bites**) at `openspec/specs/agent-tool-scheduler/spec.md`, mirroring AG-08's spec format (`openspec/specs/agent-pre-request-hook/spec.md`). Plus a delta spec at `openspec/changes/cachicamas-agent-tool-scheduler/specs/agent-loop-skeleton/spec.md` adding the `Tools` field + `Schedule` call site requirements to the AG-07 skeleton spec.
