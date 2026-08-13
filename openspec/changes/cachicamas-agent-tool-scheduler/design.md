# Design: AG-09 — Define the tool execution contract and scheduler

> **Change**: `cachicamas-agent-tool-scheduler` · **AG-09** (Layer 2, Wave 2, milestone 9 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-09--define-the-tool-execution-contract-and-scheduler), `0003:902-1004`
> **Nodes**: AG-09.1 `[leaf]` execution contract · AG-09.2 `[leaf]` concurrency policy · AG-09.3 `[leaf]` ordered rejoin · AG-09.4 `[leaf]` failure isolation
> **Format**: mirrors AG-08's `openspec/changes/archive/2026-08-13-cachicamas-agent-pre-request-hook/design.md` — Technical Approach, Architecture Decisions, Data Flow, File Changes, Interfaces/Contracts, Testing Strategy, Threat Matrix, Migration/Rollout.
> **Decisions**: D1a–D9a restated from `proposal.md` (and surfaced by `exploration.md` D1–D9) with a/b/c rejected alternatives for traceability.

## Technical Approach

AG-09 introduces `Tool` (`Name()` / `EffectClass()` / `Run(ctx, args, policy) (Result, error)`), `EffectClass` (closed `uint8` enum, 3 members mirroring `ToolOutcome`'s discipline at `tool_event.go:227-246`), `PolicySlot any` (the "Layer 2 never reads" promise is the type's doc + a source-guard test), and a `Scheduler` in `backend/agent/src/agent/`. The scheduler is **hand-rolled** (`chan struct{}` semaphore for reads with bounded fan-out default 8, single-goroutine channel for mutating/execute serialized in call order, indexed `[]Result` for rejoin, `defer/recover` per call goroutine) — **no new top-level deps**, no `errgroup` (its first-error cancellation would conflict with R-TLS-010's "siblings complete"). `Turn` (AG-07) gains a `Tools map[string]Tool` field on `TurnOptions`; on `FinishReasonToolCalls` it calls `Schedule` exactly once between `provider.Stream` close and `finalize`, emitting `ToolStart` + `ToolEnd*` through the existing `emitStamped(sink, stamper, ev)` path. A single dispatcher goroutine receives all emit work via a `chan emission` so the `LaneStamper` single-writer invariant (`sequence.go:8-24`) is preserved. **ONE cycle per turn** — the wording trap (AG-09/AG-13 boundary; `S-LSK-008`). The 21-path substrate list (`NFR-TLS-003`) stays byte-untouched for the 6th consecutive milestone; only `loop.go` + `loop_test.go` (filter widening) + a 1-line `drainSink` deadline (`sequence.go` analog) are touched. AG-09 ships `R-TLS-001..011` (12 spec scenarios + 6 bites = 18 total).

## Architecture Decisions

### Decision: D1a — `Tool` interface: three methods

| | |
|---|---|
| **Choice** | `Tool interface { Name() string; EffectClass() EffectClass; Run(ctx context.Context, args []byte, policy PolicySlot) (Result, error) }` |
| **Alternatives** | (D1b) two methods + sidecar `ToolInfo`: scheduler can't branch without calling `Run` (no declaration before invocation); (D1c) value-type `Tool struct` with a `Run` func field: not idiomatic for executables; AG-13's `ToolSource` widening doesn't shape the contract |
| **Rationale** | `Name()` enables map keying (D9a); `EffectClass()` makes the scheduler's branching obvious; `Run` carries the explicit `policy` parameter (seam 3 demand: "execution call carries a policy parameter it does not interpret", `0001:613-616`). Closes R-TLS-001, S-TLS-001. Mirrors `ToolOutcome`'s typed-result posture (R-AMT-006 carry). |

### Decision: D2a — `EffectClass`: typed enum, 3 members, `String()`

| | |
|---|---|
| **Choice** | `type EffectClass uint8` with `EffectClassRead`, `EffectClassMutating`, `EffectClassExecute`; sentinel `effectClassLimit`; `String()` mirrors `ToolOutcome.String()` (zero renders `"unset"`) |
| **Alternatives** | (D2b) string constants: comparison cost per scheduling decision, not substrate-shaped; (D2c) bitmask struct: charter floor is "three disjoint classes", not combinable flags |
| **Rationale** | Closed enum matches `ToolOutcome` (mirror tool_event.go:227-246's `iota+1` discipline; the floor not the ceiling — a future ADR can add `EffectClassNetwork`). Closes R-TLS-003, S-TLS-003. |

### Decision: D3a — `PolicySlot any` is the sandbox seam

| | |
|---|---|
| **Choice** | `type PolicySlot any` — named type over `interface{}`; doc states "Layer 2 never reads"; scheduler MUST NOT type-assert, type-switch, or rewrite |
| **Alternatives** | (D3b) `type PolicySlot []byte`: forces Layer 3 to serialize its sandbox, breaking `0001:613-616`'s "confinement is a property of the call site"; (D3c) untyped `any` parameter: no grep affordance, no seam name; (D3d) typed interface: defines a vocabulary Layer 2 doesn't need — exactly the seam 3 violation |
| **Rationale** | Documents opacity at call site; scheduler respects by construction; bit-equality test (`bytes.Equal(recorded, injected)`) + source-guard test scan for `policy.(*` / `.(PolicySlot)` in `scheduler.go` proves it. Closes R-TLS-002, S-TLS-002, S-TLS-002a (bite). |

### Decision: D4a — Hand-rolled concurrency primitives (no `errgroup`)

| | |
|---|---|
| **Choice** | `chan struct{}` semaphore (size `MaxConcurrentReads`, default `8`) for read class; single goroutine + channel for mutating+execute (serialized); indexed `[]Result` for rejoin; `defer/recover` per call goroutine |
| **Alternatives** | (D4b) `golang.org/x/sync/errgroup`: **forbidden** — new top-level dep violates `openspec/AGENTS.md` `## Hard rules`, and first-error cancellation conflicts with R-TLS-010 "one bad tool, siblings complete"; (D4c) worker pool: harder to express two-class scheduling |
| **Rationale** | Stdlib-only (NFR-PRH-002 carry → NFR-TLS-001); explicit shape matches the charter's three policies; S-TLS-005a bite (no bound → counter exceeds) proves the fan-out bound is non-vacuous. Closes R-TLS-004, R-TLS-005, S-TLS-004, S-TLS-005, S-TLS-005a. |

### Decision: D5a — Indexed result slice for ordered rejoin

| | |
|---|---|
| **Choice** | `results := make([]Result, len(calls))`; each call goroutine writes `results[call.ordinal] = res`; every ordinal slot ALWAYS populated (panic containment writes `ToolEndExecutionFailure` before returning) |
| **Alternatives** | (D5b) priority-queue rejoin: heap dep, over-engineered at n < 100; (D5c) `WaitGroup` + atomic counter: rejoin order needs extra sort, race-prone |
| **Rationale** | O(n) memory, rejoin loop is slice iteration, call identity preserves via `Result.CallID()`. S-TLS-008 + S-TLS-006a/b bites prove the property non-vacuous. Closes R-TLS-008, R-TLS-009, S-TLS-008, S-TLS-009. |

### Decision: D6a — Panic containment per call goroutine

| | |
|---|---|
| **Choice** | Each call goroutine: `defer func() { if r := recover(); r != nil { results[ordinal] = Result{Outcome: ExecutionFailure, Failure: NewFailure(...)} } }()` before `Tool.Run` |
| **Alternatives** | (D6b) `errgroup` with recover middleware: same dep / cancellation conflict as D4b; (D6c) `runtime/debug.Stack` capture: pulled-by-ambient-authority guard concern (`ambient_authority_test.go`) |
| **Rationale** | Stdlib-only. Recovery path constructs the typed `ToolEndExecutionFailure` (R-AMT-006 carry) before returning. S-TLS-011a bite proves non-vacuous (scheduler without recover → process aborts under `-race`). Closes R-TLS-010, R-TLS-011, S-TLS-010, S-TLS-011, S-TLS-010a, S-TLS-011a. |

### Decision: D6b — Single dispatcher goroutine preserves `LaneStamper` invariant

| | |
|---|---|
| **Choice** | `Scheduler` owns ONE goroutine reading `chan emission`; every call goroutine sends `(*Event, ordinal, position)` to it; only the dispatcher calls `stamper.Stamp(ev)`. Mirrors the `LaneStamper` contract: "touched only by that lane's own forwarding activity — exactly one goroutine" (`sequence.go:8-24`) |
| **Alternatives** | Per-call goroutine stamps its own events: violates the substrate's single-writer rule; would break `MakeStamp` race-detector contract under parallel reads |
| **Rationale** | The substrate invariant is stronger than a lock would provide. Closes R-TLS-006 ("start events at execution start"): dispatcher pushes `ToolStart` to `sink` immediately on receipt, NOT at rejoin. S-TLS-006a/b bites prove the timing property non-vacuous. Closes R-TLS-006, S-TLS-006, S-TLS-006a, S-TLS-006b. |

### Decision: D7a — Spec prefix `R-TLS-` / `S-TLS-NNN`

| | |
|---|---|
| **Choice** | Open `R-TLS-` (tool-scheduler) prefix; scenarios `S-TLS-NNN` |
| **Alternatives** | (D7b) extend `R-LSK-`: AG-08 already opened `R-PRH-` for the same reason (separate concern); (D7c) `R-SCH-`: doesn't name the milestone's concern |
| **Rationale** | Two-letter match to slug; per-milestone prefix continues the AG-04→AG-08 chain (`R-AEV-`/`R-AMT-`/`R-APE-`/`R-LSK-`/`R-PRH-`). |

### Decision: D8a — `Turn` calls `Schedule` once per turn; AG-13 iterates

| | |
|---|---|
| **Choice** | After `provider.Stream` returns events, `translate()` widens its switch on `ai.EventKindToolCallStart / Delta / End` (currently dropped at `loop.go:468-472`); on `ai.Completion{FinishReason: FinishReasonToolCalls}` the loop calls `Schedule(ctx, calls, opts.Tools, runID, turnID, stamper, sink)` and emits the resulting events on `sink` |
| **Alternatives** | (D8b) loop iterates model ↔ tools ↔ model: violates the wording trap (`0003:107-112`), eats AG-13's scope; (D8c) pure functional `Schedule` with loop re-emitting: blurs the seam; (D8d) `[]Result` only (no sink argument): loop must re-stamp, breaks single-writer |
| **Rationale** | The wording trap is the design. AG-09 ships ONE cycle; AG-13 wraps `Turn` in a `Harness`. S-LSK-008 + S-LSK-008a bite (counter > 1 fails) prove the one-cycle invariant non-vacuous. Closes S-LSK-008 (cross-cut in delta spec). |

### Decision: D9a — `Tools map[string]Tool`; `drainSink` 1-line deadline (carry forward)

| | |
|---|---|
| **Choice** | `TurnOptions.Tools map[string]Tool` (non-breaking zero-value extension; nil = no tools, scheduler returns typed `ExecutionFailure` per orphan). `drainSink` (`loop_test.go:147`) gains `select` with 1s deadline — closes AG-07 SUGG 1 + AG-08 SUGG 1 + AG-09 (first named consumer of the sink via the scheduler) |
| **Alternatives** | (D9b) `Tools []Tool`: O(n) scan, no keying benefit; (D9c) `ToolSource` port (G6): `ToolSource` is G6/AG-13 territory, AG-09 ships `map[string]Tool` |
| **Rationale** | O(1) resolution mirrors model name → tool lookup; nil-safe. `drainSink` timeout closes 3 successive SUGG carries with one 3-line edit. AG-08 W1 carry: at least one scheduler test uses unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline. |

## Data Flow

```
Turn(ctx, provider, system, transcript, opts, sink)         ← AG-07 + AG-08 + AG-09
  │
  ├─► mint runID, turnID, LaneStamper (fresh)
  ├─► emit run_start → sink
  ├─► emit turn_start → sink
  ├─► req := buildLoopRequest(opts, system, transcript)
  ├─► applyPreRequestHook(ctx, req, opts.PreRequestHook)     ← AG-08 seam
  ├─► pCh, _ := provider.Stream(ctx, req)
  ├─► turn := newTurnAccumulator(...)
  ├─► for ev in pCh:
  │     if done := turn.translate(ev); done {
  │        if len(turn.calls) > 0 {                         ← NEW (AG-09) tool-call events accumulated
  │          results := scheduleCalls(                      ← NEW (AG-09) — convenience wrapper
  │            ctx, turn.calls, opts.Tools, runID, turnID,
  │            sink, stamper,
  │          )
  │          turn.toolResults = results                      ← appended to transcript
  │        }
  │        msg, finish := turn.finalize()
  │        closeSink(sink)
  │        return msg, finish, nil
  │     }
  ├─► emit turn_end → sink
  ├─► emit run_end → sink
  ├─► closeSink(sink)
  └─► return (msg, finish, nil)

scheduleCalls(ctx, calls, registry, runID, turnID, sink, stamper):
  semaphore := make(chan struct{}, MaxConcurrentReads)       ← bounded fan-out
  results   := make([]Result, len(calls))                    ← indexed by ordinal
  emissions := make(chan emission)                           ← dispatcher channel
  var wg sync.WaitGroup
  go runDispatcher(stamper, sink, emissions)                 ← ONE stamper owner
  for i, call := range calls {
    i, call := i, call                                       ← closure capture
    if call.EffectClass() == EffectClassRead {
      wg.Add(1)
      go func() {                                            ← bounded fan-out
        defer wg.Done()
        semaphore <- struct{}{}
        defer func() { <-semaphore }()
        executeCall(ctx, i, call, results, emissions)        ← recover inside
      }()
    } else {
      wg.Add(1)
      go func() {                                            ← serialized channel
        defer wg.Done()
        order <- struct{}{}
        defer func() { <-order }()
        executeCall(ctx, i, call, results, emissions)
      }()
    }
  }
  wg.Wait()
  close(emissions)
  return results
```

`executeCall` resolves the tool via `registry.Resolve(call.name)`, emits `ToolStart` through `emissions` BEFORE calling `Tool.Run` (R-TLS-006: start events at execution start), records the `Result` (success / result-failure / execution-failure / panic), emits the appropriate `ToolEnd*`. The dispatcher goroutine is the single writer of `stamper` — preserves the `LaneStamper` single-writer invariant.

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/tool.go` | Create | `Tool` interface (3 methods); `EffectClass` enum (3 members, `String()`); `PolicySlot = any`; `Result` struct (3 outcomes mirroring `ToolOutcome`); constructor validations. ~180 lines. |
| `backend/agent/src/agent/tool_test.go` | Create | Contract-from-outside bytes (R-TLS-001..007); policy passthrough opaque (R-TLS-002, S-TLS-002 + S-TLS-002a bite); effect class vocabulary (R-TLS-003). ~280 lines. |
| `backend/agent/src/agent/scheduler.go` | Create | `Registry interface{ Resolve(string) (Tool, bool) }`; `Scheduler{MaxConcurrentReads int}`; `Schedule(ctx, calls, reg, runID, turnID, stamper, sink) []Result` — semaphore + serialized channel + indexed slice + `defer/recover` + one dispatcher goroutine. ~340 lines. |
| `backend/agent/src/agent/scheduler_test.go` | Create | Concurrency (R-TLS-004/005, S-TLS-004/005 + S-TLS-005a bite); start-at-execution (R-TLS-006, S-TLS-006 + S-TLS-006a/b bites); rejoin + correlation (R-TLS-008/009, S-TLS-008/009); failure isolation (R-TLS-010/011, S-TLS-010/011 + S-TLS-010a/011a bites); source-guard test (S-TLS-002 + scan-no-type-assert). ~900 lines. |
| `backend/agent/src/agent/loop.go` | Modify | New `Tools map[string]Tool` field on `TurnOptions`; widen `translate()` default branch on `ai.EventKindToolCallStart / Delta / End` (currently dropped at `:468-472`); on `Completion{FinishReason: FinishReasonToolCalls}` call `Schedule` once between provider stream close and `finalize`; append `tool_results` to the assistant message; add helper `scheduleCalls` (or inline equivalent) — ~150 lines added. |
| `backend/agent/src/agent/loop_tool_dispatch_test.go` | Create | Integration: AI-21 fake provider streams tool-call events + `Completion` → `Turn` calls `Schedule` → emits AG-05.2 events on sink; one S-LSK-008 bite in this file. ~280 lines. |
| `backend/agent/src/agent/loop_test.go` | Modify | Widen `TestTurn_SubstrateUntouched` filter to also exclude `loop_tool_dispatch_test.go`. Add 1-line `select` deadline to `drainSink` (~3 lines). |
| `backend/agent/src/agent/agent_test_helpers_test.go` | Modify (optional) | Add `ordinalFromToolStart(events, callID) uint32` helper local to dispatch test. |
| `backend/agent/src/agenttest/scripted_tool.go` | Create | In-memory `agent.Tool`: configurable per-call outcome (success / result-failure / execution-failure / panic / delayed completion); records `start time` + `policy value` byte-for-byte. Mirror `scripted_provider.go` shape. ~180 lines. |
| `backend/agent/src/agenttest/scripted_tool_test.go` | Create | Self-tests for the scripted tool. ~120 lines. |
| `backend/agent/src/agent/{event,event_descriptor,stream_check,failure,sequence,run_events,turn_events,message_text,message_reasoning,permission_events,cost_events,delegation_events,compaction_events,tool_event,event_registry_test,doc,doc_contract_guard_test,ambient_authority_test,import_boundary_test,reconstruction_test}.go` + `go.mod` + `go.sum` + `Makefile` + `.golangci.yml` | **UNTOUCHED** | 21 substrate files (per `R-LSK-004` carry, `NFR-TLS-003`). 6th consecutive "substrate untouched" milestone. NO new deps (errgroup / semaphore forbidden). |

## Interfaces / Contracts

```go
// backend/agent/src/agent/tool.go

// EffectClass is the closed vocabulary of tool effects (D2). Mirrors
// ToolOutcome's posture (tool_event.go:227-246) — the zero value is NOT
// a member; `String()` renders it as "unset".
type EffectClass uint8

const (
    EffectClassRead     EffectClass = iota + 1
    EffectClassMutating
    EffectClassExecute
    effectClassLimit
)

func (c EffectClass) String() string { /* "unset" / "read" / "mutating" / "execute" / "effectclass(N)" */ }

// PolicySlot is the per-call sandbox seam Layer 2 carries opaquely.
// Charter AG-09.1 #2 + v2 § 5.1 quote (0001:613-616): "confinement is
// a property of the call site, not of the code being called." The
// scheduler MUST NOT type-assert, type-switch, or rewrite this value.
type PolicySlot any

// Result is the typed outcome of one Tool.Run call (R-TLS-007). Three
// outcomes distinct by kind (mirror ToolOutcome); Run returns
// (Result{Outcome: ...}, nil) for Success / ResultFailure, and
// (Result{}, err) for ExecutionFailure — the two channels are disjoint.
type Result struct {
    Outcome ToolOutcome // Success | ResultFailure | ExecutionFailure (mirror AG-05.2)
    Content []byte      // success / result-failure payload (empty for ExecutionFailure)
    Failure *Failure    // required iff Outcome == ExecutionFailure (R-AMT-006 carry)
}

// CallID is the correlation identity the loop reconstructed at translate().
// Mirrored on Result so the rejoin test can assert byte-equality (R-TLS-009).
func (r Result) CallID() string

type Tool interface {
    Name() string
    EffectClass() EffectClass
    Run(ctx context.Context, args []byte, policy PolicySlot) (Result, error)
}
```

```go
// backend/agent/src/agent/scheduler.go

// Registry resolves a tool name to its implementation. Map-backed or
// any source; resolution miss yields a typed Result{ExecutionFailure}
// in the call's ordinal slot (consistent with R-TLS-010 "one bad tool,
// siblings complete"). ToolSource port (G6) is AG-13's widening.
type Registry interface {
    Resolve(name string) (Tool, bool)
}

// NewMapRegistry returns a Registry backed by a map (D9a). Nil-safe.
func NewMapRegistry(tools map[string]Tool) Registry

// Scheduler runs a slice of tool calls under the AG-09.2 concurrency
// rules (reads concurrent with bounded fan-out, mutating + execute
// serialized in call order), rejoins results in call order, contains
// panics, and preserves the LaneStamper single-writer invariant via
// one dispatcher goroutine.
type Scheduler struct {
    MaxConcurrentReads int // default 8; 0 → default
}

func (s *Scheduler) Schedule(
    ctx context.Context,
    calls []ai.ToolCall,
    reg Registry,
    runID RunID,
    turnID TurnID,
    stamper *LaneStamper,
    sink chan<- *Event,
) []Result
```

```go
// backend/agent/src/agent/loop.go (TurnOptions delta)

type TurnOptions struct {
    // AG-07 — unchanged
    Model     string
    MaxTokens int
    // AG-08 — unchanged (R-PRH-001..007)
    PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)
    // AG-09 — NEW (R-TLS-001..011).
    //
    // Tools is the registry keyed by tool name (D9a). The loop
    // resolves each model request via registry.Resolve(name); an
    // unresolved name yields a typed
    // Result{Outcome: ExecutionFailure, Failure: ...} in that call's
    // ordinal slot — consistent with R-TLS-010 "one bad tool does
    // not abort the turn".
    //
    // Nil is the identity default: a zero-value TurnOptions
    // produces the same tool-call failure results as a TurnOptions
    // with a registry whose Resolve always returns false.
    Tools Registry
}
```

Cross-references: `tool_event.go:227-246` (`ToolOutcome`); `sequence.go:8-24` (`LaneStamper` single-writer); `failure.go:24-79` (`Failure` wrap, `NewFailure`); `ai/tool_call.go:45` (`ai.ToolCall.id / name / arguments`); `ai/tool_call_event.go:74-336` (AI-18 streamed events); `loop.go:468-472` (current `translate()` default branch — to be widened).

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (contract) | `Tool` interface from outside; `EffectClass.String()` (3 members + zero); `Result` construction; `PolicySlot` byte-exact passthrough; source-guard (scan `scheduler.go` for `policy.(*` / `.(PolicySlot)`) | `tool_test.go` (RED bites) |
| Unit (concurrency) | S-TLS-004 (reads concurrent / mutatings serialized); S-TLS-005 + S-TLS-005a bite (bounded fan-out, fan-out counter assertion); unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline (AG-08 W1 carry) | `scheduler_test.go` with `scripted_tool.go` channels |
| Unit (start events) | S-TLS-006: staggered delays → start events in execution order; S-TLS-006a bite (inverted completions → start order differs from completion); S-TLS-006b bite (start precedes end ≥ 1 tick) | `scheduler_test.go` with delayed scripted tools |
| Unit (rejoin) | S-TLS-008: inverted completions → slot[i].CallID() == calls[i].id; S-TLS-009: synthetic IDs preserved | `scheduler_test.go` |
| Unit (failure isolation) | S-TLS-010: failing call → slot typed ExecutionFailure + siblings; S-TLS-010a bite (errgroup-shaped mock fails sibling slot); S-TLS-011 panic containment; S-TLS-011a bite (no recover → process aborts under `-race`); `runtime.NumGoroutine()` baseline restored | `scheduler_test.go` |
| Integration | `Turn` end-to-end: provider streams tool-call triplet + `Completion{FinishReasonToolCalls}` → loop accumulates calls → calls `Schedule` exactly once → emits AG-05.2 events on sink → finalizes with `tool_results` appended | `loop_tool_dispatch_test.go`; S-LSK-008 + S-LSK-008a bites |
| Substrate | 21 files (AG-07 R-LSK-004 list + `go.mod`+`go.sum`+`Makefile`+`.golangci.yml`) byte-unchanged; `AG09_BASE_REF` env fallback + dynamic `git merge-base HEAD origin/main`; filter widened to exclude `loop_tool_dispatch_test.go` | `TestTurn_SubstrateUntouched` (existing, widened) |
| Coverage | `loop.go` ≥ 80% (AG-04 W8 carry) | `cd backend/agent && make test/cover` |
| Race | every test under `go test -race -v ./...` | `make test` |

External posture (NFR-TLS-001 inheritance): every behavioral test in `package agent_test` or external — AG-07 W6 carry. Strict TDD (AG-04 + AG-05 + AG-08 pattern): all 6 bites RED-recorded BEFORE their property GREEN per `proposal.md` carry-forwards (AG-05 W1, AG-08 SUGG 3).

## Threat Matrix

| Threat | Severity | Mitigation |
|---|---|---|
| `errgroup` introduced (forbidden new top-level dep) | CRITICAL | D4a explicit; the source guard test scans `scheduler.go` for `golang.org/x/sync/errgroup`; `go.mod` byte-unchanged check |
| Scheduler type-asserts `PolicySlot` (seam 3 violation) | CRITICAL | D3a explicit; source guard test regex-scans `scheduler.go` for `policy\.(\*` / `\.(PolicySlot)`; S-TLS-002 + S-TLS-002a bite enforces byte-exact passthrough |
| `LaneStamper` written by >1 goroutine (race) | CRITICAL | D6b: single dispatcher goroutine owns the stamper; call goroutines send `emission` records to it. Substrate invariant `sequence.go:8-24` preserved. AG-22 sweep under `-race` |
| Concurrency race on the `[]Result` slice | High | Slice is the only shared state; each call goroutine writes its OWN index (no overlap); reads happen after `wg.Wait()` (single-writer-per-slot, no race) |
| Concurrent reads under `-race` | High | Semaphore = bounded fan-out; per-call scripted tool state uses `sync.Mutex` |
| Substrate-untouched streak (filter widening) | Low | 6th consecutive milestone; pattern from AG-07 W3 carries; consolidation deferred to AG-23 |
| Unbuffered `sink` test (AG-08 W1) | Low | At least 1 scheduler test uses unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline |
| `drainSink` deadlock (AG-07 + AG-08 + AG-09 SUGG carry) | Low | 1-line `select` deadline in `drainSink` (`loop_test.go:147`); first named consumer of the new sink path is the scheduler |
| Wire-up creep into AG-13 (iteration) | Low | S-LSK-008 + S-LSK-008a bites enforce one-cycle invariant |
| Review budget (~2,380 line forecast) | Low | `size:exception` pre-authorized; single PR (substrate preservation + filter widening couple the leaves); 8-commit work-unit split per `work-unit-commits` |
| External-package test posture (NFR-TLS-001) | Low | All behavioral tests in `package agent_test` |

**N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.** AG-09 introduces `Tool` + `EffectClass` + `PolicySlot` + `Scheduler` and the loop wire-up; all are pure Go over the Layer 2 envelope and the Layer 1 provider contract. `agenttest.ScriptedTool` operates over in-memory channels (no filesystem, no environment, no subprocess). The scheduler's goroutines are stdlib `chan struct{}` semantics, not `errgroup`. The `LaneStamper` is in-process. No row of `references/threat-matrix.md` applies.

## Migration / Rollout

No migration required. AG-09 is a strict addition:

- **`Turn` signature**: byte-identical to AG-08 (the new field lives on `TurnOptions`, not on `Turn` itself).
- **`TurnOptions` signature**: gains one field (`Tools Registry`). Zero-value = identity default (nil → typed `ExecutionFailure` per orphan call). Non-breaking.
- **Layer 1 substrate**: zero edits (`ai.Tool` stays declarative; AG-09 owns the effect-class registry at Layer 2; ADR-able if future Layer 1 widens).
- **Layer 2 substrate**: zero edits (envelope / descriptor / validator / stamper unchanged; only `loop.go` + `loop_test.go` filter widening + `drainSink` 1-line deadline + dispatch test).
- **No external caller of `Tools` exists yet** — AG-09 is the first user. AG-10 wraps the scheduler with permission events; AG-13 wraps `Turn` in `Harness` for iteration; doc 0004 (Layer 3) implements built-in tools against this contract.

Single PR (`feat/agent-layer2-wave2-ag09`) — substrate preservation, filter widening, and loop wire-up couple the leaves (the doc 0003 mermaid has AG-09.1 → AG-09.2, AG-09.2 → AG-09.3, AG-09.1 → AG-09.4). `size:exception` pre-authorized (`braejan` standing rule, AG-04/05/06/07/08 precedent).

## Open Questions

- [ ] **`MaxConcurrentReads` default**: proposal says `8`; AG-13 may benchmark (4 vs 8 vs 16).
- [ ] **Dispatcher goroutine lifetime**: closes when `Schedule` returns (after `wg.Wait()` + `close(emissions)`); no goroutine leak.
- [ ] **Panic → `*Failure` category**: `ai.FailureCategoryUnavailable` placeholder per exploration U4; ADR-able if AG-11 prefers `FailureCategoryPanic` (semantic gap: a panic ≠ "unavailable"). AG-09 does not block.
- [ ] **Zero-value `Tools` / no tool registered**: nil → typed `ExecutionFailure` per orphan; structurally identical to R-TLS-010's "bad tool" path.

## Next step

Launch `sdd-tasks` next — write `tasks.md` with phases 1–5 (spec recapped; tool contract + scheduler; loop wire-up; panic containment + start events; verify-report), 8-commit work-unit breakdown per `work-unit-commits`, and the strict TDD ratchet (RED bites for S-TLS-002a/005a/006a/006b/008/010a/011a RED-recorded BEFORE their property scenarios GREEN). Forecast ~2,280 new + ~100 modified = ~2,380 lines; 400-line budget risk **Low** (chained PRs **No**, single PR under pre-authorized `size:exception`); `21 substrate files byte-unchanged` check is the load-bearing acceptance criterion.
