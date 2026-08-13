# Design: AG-08 — Add the pre-request hook seam

> **Change**: `cachicamas-agent-pre-request-hook` · **AG-08** (Layer 2, Wave 2, milestone 8 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-08--add-the-pre-request-hook-seam), `0003:833-900`
> **Nodes**: AG-08.1 `[leaf]` (hook seam) · AG-08.2 `[leaf]` (prefix stability)
> **Format**: mirrors AG-07's `openspec/changes/archive/2026-08-13-cachicamas-agent-loop-skeleton/design.md` exactly — Technical Approach, Architecture Decisions, Data Flow, File Changes, Interfaces/Contracts, Testing Strategy, Threat Matrix, Migration/Rollout.
> **Decisions**: D1a–D5a restated from `proposal.md` with **a/b/c rejected alternatives** for traceability, per AG-07's decision-row format.

## Technical Approach

The seam wraps `Turn` (`backend/agent/src/agent/loop.go:103-179`) at the single point where the assembled `ai.Request` exists as data and `provider.Stream` has not been called yet: between `buildLoopRequest` (`loop.go:132`) and `provider.Stream` (`loop.go:140`). The hook receives the assembled request and returns a derived `ai.Request` via `req.With(...)` copy-on-write rebuild (R-REX-001, `backend/agent/src/ai/request.go:325-336`) — a hook that calls `With(...)` returns a fresh value; the loop's `req` is left observably unmodified (the substrate's no-mutation promise, exercised by AG-08.1 #4). The hook failure path reuses the existing pre-stream-failure pattern verbatim (`loop.go:140-147`): `closeSink(sink)` and return `(ai.Message{}, 0, typedErr)` with `typedErr` constructed via `ai.PreStreamFailure` (`backend/agent/src/ai/provider_failure.go:34-93`) carrying a hook-attributing `FailureReport.Category`. Identity default (D4a): nil hook = no-op, identical inputs → identical outputs (AG-07 R-LSK-002 byte-stability preserved, asserted by AG-08.1 #2 / S-PRH-002). AG-08 is the first public-surface extension of `TurnOptions` since AG-07 and the first live consumer of Layer 1's `Request.With`; the substrate stays byte-untouched — 5th consecutive "substrate untouched" milestone.

## Architecture Decisions

### Decision: D1a — Hook surface is a single callable on `TurnOptions`

| | |
|---|---|
| **Choice** | `TurnOptions.PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)` — a function-typed field, not an interface, not a slice |
| **Alternatives** | (D1b) `Hook` interface with `OnPreRequest(ctx, req) (req, err)` — over-engineered for one hook, forecloses AG-20's chain-composition widening; (D1c) `[]Hook` chain on `TurnOptions` — AG-20's scope, not AG-08's |
| **Rationale** | AG-07 D1a chose function-form for `Turn` itself; the loop's surface is the thinnest end-to-end path. Charter says "the hook" (singular, `0003:833-900`). AG-20 widens to `[]Hook` chain composition — its scope (doc 0001 G11 row, "the chain composition widens from this single hook"). |

### Decision: D2a — Wrap between `buildLoopRequest` and `provider.Stream` (failure mirrors pre-stream-failure)

| | |
|---|---|
| **Choice** | Branch inserted between `loop.go:132` (`req, err := buildLoopRequest(...)`) and `loop.go:140` (`pCh, streamErr := provider.Stream(...)`). On hook error: `closeSink(sink)`, return `(ai.Message{}, 0, typedErr)` mirroring `loop.go:140-147` |
| **Alternatives** | (D2b) Inside `buildLoopRequest` as a parameter — pollutes the helper's signature; helpers stay pure, the seam lives at the call site; (D2c) Separate function `wrapWithPreRequestHook(Turn)` called by `Turn` before `provider.Stream` — same shape, less obvious because it hides the seam from `Turn`'s own body |
| **Rationale** | The pre-stream-failure path (`loop.go:140-147`) is the established typed-error precedent: closes `sink`, returns `ai.Message{}, 0, streamErr`. A hook failure must look identical to a provider pre-stream failure from the caller's side. Two-line change inside `Turn` keeps the seam in the obvious place. |

### Decision: D3a — Hook receives the loop's `ctx`

| | |
|---|---|
| **Choice** | `func(ctx context.Context, req ai.Request) (ai.Request, error)` — the loop passes its own `ctx` |
| **Alternatives** | (D3b) `func(req ai.Request) (ai.Request, error)` — without `ctx`; forecloses cancellation/deadline/tracing the hook may legitimately need |
| **Rationale** | `ctx` already flows to `provider.Stream(ctx, req)` (AG-07 D5a, `loop.go:140`). Carrying it to the hook is cheap; not carrying it forecloses any future hook that wants tracing or cancellation. AG-20's chain composition will inherit the parameter. |

### Decision: D4a — Identity default is nil

| | |
|---|---|
| **Choice** | `TurnOptions.PreRequestHook == nil` → skip the seam, proceed to `provider.Stream(ctx, req)` unchanged. Zero-value `TurnOptions` produces byte-identical output to AG-07's skeleton for identical inputs |
| **Alternatives** | (D4b) Always-invoke a default no-op closure `func(...) (ai.Request, error) { return req, nil }` — wastes a function call per turn and obscures the no-cost path |
| **Rationale** | AG-07 R-LSK-002 byte-stability on identical inputs is the established precedent (proven by `TestTurn_TwoSequentialTurnsShareNothing`); AG-08 must preserve it (AG-08.1 #2 / S-PRH-002 asserts this against the skeleton's captured request). |

### Decision: D5a — Spec prefix `R-PRH-`; scenarios `S-PRH-NNN`

| | |
|---|---|
| **Choice** | New prefix `R-PRH-` (pre-request hook); scenarios `S-PRH-NNN` |
| **Alternatives** | (D5b) Extend `R-LSK-` (AG-07's prefix) — AG-08 closes a different requirement (R-12, G4 Layer 2 half, not AG-07's R-LSK-001..005). AG-07's quintet stays a closed set |
| **Rationale** | Distinct prefix signals a different requirement closure (R-12, prefix stability) from AG-07's contract event order. Two-letter match to the slug (`Pre-Request-Hook`). AG-09 / AG-10 / AG-11 open their own; AG-20 takes `R-HK-` / `R-FHT-`. |

## Data Flow

```
Turn(ctx, provider, system, transcript, opts, sink)
  │
  ├─► mint runID, turnID, LaneStamper (fresh)
  ├─► emit run_start → sink
  ├─► emit turn_start → sink
  ├─► req, err := buildLoopRequest(opts, system, transcript)   ← loop.go:132
  ├─► if err != nil {
  │     closeSink(sink)
  │     return ai.Message{}, 0, err
  │   }
  ├─► if opts.PreRequestHook != nil {                          ← NEW (AG-08.1)
  │     req, err = opts.PreRequestHook(ctx, req)
  │     if err != nil {
  │       typedErr := ai.PreStreamFailure(ai.FailureReport{
  │         Category:   ai.FailureCategoryUnsupportedCapability, // hook-attributing
  │         StatusClass: 4,
  │       })
  │       closeSink(sink)
  │       return ai.Message{}, 0, typedErr                      ← mirrors loop.go:140-147
  │     }
  │   }
  ├─► pCh, pErr := provider.Stream(ctx, req)                   ← loop.go:140
  ├─► for ev in pCh: translate(ev)
  ├─► emit turn_end / run_end → sink
  └─► return (msg, finish, err)
```

`req` flows through the seam by value. The hook receives the assembled `ai.Request`, calls `req.With(...)` to derive a new value, and returns it; the loop's local `req` rebinds to the returned value. The hook's return value is what `provider.Stream(ctx, req)` ultimately receives — verified by `provider.Requests()[0]` byte-equality against the hook's return (AG-08.1 #1 / S-PRH-001).

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/loop.go` | MODIFY | (1) New `PreRequestHook` field on `TurnOptions` (~3 lines: name, type, doc comment); (2) new private helper `applyPreRequestHook(ctx, req, hook) (ai.Request, error)` (~10 lines); (3) 8-line branch in `Turn` between line 132 and line 140 (typed-error wrap, close sink, return zero). Total +20 to +40 lines on top of the AG-07 457-line file (forecast loop.go → ~480). |
| `backend/agent/src/agent/loop_hook_test.go` (new) OR `backend/agent/src/agent/loop_test.go` (append) | CREATE / APPEND | 10 test functions (see Testing Strategy). 6 charter scenarios + 2 bites (S-PRH-001a/b) + 1 AG-07 W1 unbuffered-sink carry (S-PRH-007) + 1 substrate-untouched guard (NFR-PRH-003). Forecast +300–500 lines. |
| `backend/agent/src/agent/{event,event_descriptor,stream_check,failure,sequence,run_events,turn_events,message_text,message_reasoning,permission_events,cost_events,delegation_events,compaction_events,tool_event,event_registry_test,doc,doc_contract_guard_test,ambient_authority_test,import_boundary_test,reconstruction_test}.go` | UNTOUCHED | 20-file substrate preserved (NFR-PRH-003, R-LSK-004 carry). 5th consecutive "substrate untouched" milestone. |
| `backend/agent/go.mod`, `go.sum` | UNTOUCHED | No new deps. |
| `backend/agent/Makefile`, `.golangci.yml` | UNTOUCHED | No Makefile / lint-config edits. |
| `openspec/specs/agent-pre-request-hook/spec.md` | NEW (sdd-spec) | 7 requirements + 7 scenarios + 2 bites = 9 total. |
| `openspec/changes/cachicamas-agent-pre-request-hook/{exploration,proposal,design,tasks,verify-report}.md` | NEW | Phase artifacts. |

## Interfaces / Contracts

```go
// In backend/agent/src/agent/loop.go

// TurnOptions is the options struct for a single assistant turn.
// AG-08 extends AG-07's surface with a pre-request hook seam (R-PRH-001..007):
// the hook is invoked once per Turn between buildLoopRequest (loop.go:132)
// and provider.Stream (loop.go:140); nil is the identity default; the
// hook may derive a new ai.Request via req.With(...) and must NOT mutate
// the loop's input in place (R-REX-001).
type TurnOptions struct {
    // Model is the model identifier passed to the provider. Empty = provider default.
    // (AG-07 — unchanged)
    Model string

    // MaxTokens is the optional max-tokens budget. Zero = provider default.
    // (AG-07 — unchanged)
    MaxTokens int

    // PreRequestHook is invoked once per Turn, after buildLoopRequest
    // (loop.go:132) and before provider.Stream (loop.go:140). It receives
    // the fully-assembled ai.Request and the loop's own ctx. It returns a
    // derived ai.Request (typically via req.With(...)) or an error.
    //
    // Nil is the identity default: Turn skips the seam and proceeds
    // unchanged, preserving AG-07's R-LSK-002 byte-stability on identical
    // inputs (R-PRH-005, S-PRH-002).
    //
    // A non-nil hook that returns an error aborts the turn before I/O:
    // sink is closed, and Turn returns (ai.Message{}, 0, *ai.PreStreamFailure)
    // reusing the existing pre-stream-failure path (R-PRH-003, S-PRH-003).
    //
    // Hooks must NOT mutate the loop's input in place; ai.Request is a
    // value type whose With(...) rebuilds from r.options (R-REX-001,
    // request.go:325-336). For identical inputs across calls, the hook
    // must produce byte-equal outputs (R-PRH-007, S-PRH-006).
    PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)
}

// applyPreRequestHook invokes hook if non-nil; returns (req, nil) unchanged
// when hook is nil. AG-08.1 helper extracted to keep the Turn branch small
// and to unit-test the identity-default no-op in isolation if needed.
func applyPreRequestHook(
    ctx context.Context,
    req ai.Request,
    hook func(ctx context.Context, req ai.Request) (ai.Request, error),
) (ai.Request, error) {
    if hook == nil {
        return req, nil
    }
    return hook(ctx, req)
}
```

State the typed-error construction (mirrors `loop.go:140-147`):

```go
typedErr, perr := ai.PreStreamFailure(ai.FailureReport{
    Category:    ai.FailureCategoryUnsupportedCapability, // hook-attributing; not provider-auth
    StatusClass: 4,
})
if perr != nil {
    closeSink(sink)
    return ai.Message{}, 0, perr
}
closeSink(sink)
return ai.Message{}, 0, typedErr
```

Cross-references:

- **AG-07 `TurnOptions`**: `backend/agent/src/agent/loop.go:46-51` (two existing fields: `Model`, `MaxTokens`).
- **AG-07 pre-stream-failure path**: `backend/agent/src/agent/loop.go:140-147` (`closeSink` + return zero).
- **AI-12 `Request.With`**: `backend/agent/src/ai/request.go:325-336` (the mutation mechanism AG-08's hook will call).
- **AI-21 `Requests()` accessor**: `backend/agent/src/agenttest/fake_provider.go:157-161` (the test substrate AG-08.1 reads to verify hook reception and identity default).
- **AI-11 `Request.CacheBoundaries()`**: `backend/agent/src/ai/cache_boundary.go:118-120` (cascade-order pin for AG-08.2 #1).
- **R-ACB-007 cascade contract**: `openspec/specs/ai-cache-breakpoints/spec.md:158-178` (markers readable in tools → system → messages order).
- **AG-19 typed-error substrate**: `backend/agent/src/ai/provider_failure.go:34-93` (vocabulary + `FailureReport` shape), `:610-612` (`PreStreamFailure` constructor).

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit — hook wiring | S-PRH-001 (R-PRH-002), S-PRH-001a/b bites (R-PRH-002), S-PRH-002 (R-PRH-005), S-PRH-003 (R-PRH-003), S-PRH-004 (R-PRH-004) | `agenttest.NewProvider(scriptTextResponse)` + `provider.Requests()` byte-equality + `sink` close channel assertion; bites RED-recorded first (AG-05 W1) |
| Unit — prefix stability | S-PRH-005 (R-PRH-006), S-PRH-006 (R-PRH-007) | Two consecutive `Turn(...)` calls; compare `provider.Requests()[0].Equal(...)` for tools/system regions; `provider.Requests()[i].CacheBoundaries()` for cascade-order pin; message-region growth via length diff + `Message.Equal` for content |
| Cross-cut — back-pressure | S-PRH-007 (AG-07 W1) | Unbuffered `sink` (`make(chan *agent.Event)`) + concurrent consumer goroutine + `runtime.NumGoroutine()` before/after baseline assertion |
| Substrate | NFR-PRH-003 substrate-untouched | `git diff <AG08_BASE_REF>..HEAD -- backend/agent/src/agent/` filtered to exclude `loop.go` and `loop_hook_test.go` / `loop_test.go`; dynamic merge-base fallback (AG-07 W3 fix carried forward) |
| Coverage | R-LSK-005 carry | `cd backend/agent && make test/cover` — `loop.go` ≥ 80% (AG-07 hit 85.89%; AG-08 +20–40 lines forecast stays above) |
| Race | every test | `cd backend/agent && make test` — `-race -v ./...` |

### Strict TDD order of test commits (RED-first, AG-05 W1 carry)

1. **S-PRH-001a RED**: hook returns `(req, nil)` unchanged → assert captured system region CONTAINS the marker. Assertion fails for the right reason (marker absent) — proves the property is non-vacuous.
2. **S-PRH-001b RED**: hook returns `(req, nil)` unchanged → assert captured request IS `Equal` to skeleton's (no marker). Equality holds but the system region lacks the marker — proves the test wiring distinguishes "no mutation" from "mutation applied".
3. **S-PRH-001 GREEN**: production code passes S-PRH-001a (capture has marker) and S-PRH-001b (capture is byte-equal skeleton's — actually they conflict; this is the bite harness).
4. **S-PRH-002** (R-PRH-005): nil hook → captured request `Equal` to skeleton's. RED then GREEN.
5. **S-PRH-003** (R-PRH-003): hook returns `(ai.Request{}, errors.New("hook boom"))` → assert `len(provider.Requests()) == 0`, sink drains unblocked, returned error wraps `*ai.PreStreamFailure`. RED then GREEN.
6. **S-PRH-004** (R-PRH-004): mutating hook (`msgs := req.Messages(); msgs[0] = mutated`) → assert captured request `Equal` skeleton's. RED then GREEN.
7. **S-PRH-005** (R-PRH-006): two consecutive turns, same system + same tools + same hook, transcript + 1 message → tools/system `Equal` byte-equal, `CacheBoundaries()` cascade order pinned, message region grew by 1, first N `Message.Equal`. RED then GREEN.
8. **S-PRH-006** (R-PRH-007): hook deterministic for identical inputs → both calls' hook outputs byte-equal. RED then GREEN.
9. **S-PRH-007** (AG-07 W1 carry): unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline. RED then GREEN.
10. **Substrate-untouched guard**: `AG08_BASE_REF` env var fallback + dynamic merge-base. RED then GREEN.

### Test file structure

Choice between **new `loop_hook_test.go`** vs **append `loop_test.go`** is left to `sdd-tasks` (see Open Questions). Forecast either way: 300–500 lines, ~10 functions.

### Coverage gate

`cd backend/agent && make test/cover`. AG-08's loop.go changes add ~30 lines on top of AG-07's 457; AG-07 hit 85.89% line coverage on `loop.go`. Adding 20–40 lines (10–20 helper + 8–15 branch + 5–10 doc comment) preserves coverage margin well above the 80% threshold. Forecast: 80–87% line coverage.

### External posture (NFR-PRH-001)

Every AG-08 test in `package agent_test` (AG-07 W6 carry; AG-07's `loop_test.go` already uses this package).

## Commit Plan

Per `work-unit-commits` skill (AG-07 style — 8 commits expected, each well under the 400-line review budget, well under the 1000-line `size:exception` pre-authorized):

1. `chore(agent): AG-08 spec + design committed` — spec artifacts only; no code.
2. `test(agent): AG-08.1 RED bites — S-PRH-001a + S-PRH-001b` — RED bites before any production code (AG-05 W1 carry).
3. `feat(agent): AG-08.1 hook seam — TurnOptions.PreRequestHook + applyPreRequestHook + Turn branch`.
4. `test(agent): AG-08.1 S-PRH-001 + S-PRH-002 (identity default)` — closes R-PRH-002 + R-PRH-005.
5. `test(agent): AG-08.1 S-PRH-003 + S-PRH-004 (failure + no-mutate)` — closes R-PRH-003 + R-PRH-004.
6. `test(agent): AG-08.2 S-PRH-005 + S-PRH-006 (prefix stability + determinism)` — closes R-PRH-006 + R-PRH-007.
7. `test(agent): AG-08.2 S-PRH-007 (unbuffered sink) + substrate-untouched guard` — closes AG-07 W1 carry + NFR-PRH-003.
8. `chore(agent): AG-08 archive — proposal/spec/design/tasks/verify-report + PR open`.

Each commit ≤ 400 lines; total forecast 350–600 added; well under the 1000-line `size:exception` budget.

## Threat Matrix

| Threat | Severity | Mitigation |
|---|---|---|
| Hook failure path leaves `sink` open (R1) | Medium | S-PRH-003 RED-first (AG-05 W1 carry). Assert `len(provider.Requests()) == 0` AND `sink` drains unblocked (channel closed) before any subsequent test runs. Mirrors `loop.go:140-147` verbatim. |
| R-REX-001 substrate promise break (R2) | Medium | S-PRH-004 mutating-hook test + `ai.Request.Equal` (`request.go:555-627`) on captured request vs skeleton's. RED bites first. |
| Prefix-stability cascade violation (R3) | Medium | S-PRH-005 cascade byte-stability via `Request.Equal` for tools/system regions + `Request.CacheBoundaries()` (`cache_boundary.go:118-120`) for cascade-order pin (R-ACB-007 contract). |
| Non-deterministic hook invalidates cascade (R4) | Low | S-PRH-006 deterministic for identical inputs; combined with S-PRH-005, closes the prefix-stability property. Timestamp-bearing regions would fail S-PRH-005's cascade comparison. |
| Message-region growth assumes caller discipline (R5) | Low | AG-12.1 owns loop-side append-only discipline. AG-08.2 #1's test writes the message-region growth externally; surface in spec as caller-responsibility. |
| `Request.Equal` excludes `MessageID` (R6) | Low | Tools/system cascade uses `Equal` byte-equality; message-region growth uses `Message.Equal` + length diff (`Equal` excludes `MessageID` by design, `request.go:555-559`). |
| `TurnOptions` public-surface change (R7) | Low | Zero-value = identity default (D4a); no external caller of the new field exists yet (AG-08 is the first user). Non-breaking change. |
| Review budget 1000 lines (R8) | Low | Forecast 350–600 added; `size:exception` pre-authorized. Well under budget. |
| External-package test posture (R9) | Low | Every test in `package agent_test` (AG-07 W6 → NFR-PRH-001). |
| Substrate-untouched guard ref goes stale on merge (AG-07 W3) | Low | `AG08_BASE_REF` env-var fallback (shipped in AG-07 PR #167 W3 fix) + dynamic merge-base as default. |

**N/A — routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary:** AG-08 introduces none. The loop is pure Go over the Layer 2 envelope and the Layer 1 provider contract. `agenttest.Provider` is the only "external" boundary, operating over in-memory channels. No filesystem, network, environment, or process spawn.

## Migration / Rollout

No migration required. AG-08 is additive:

- **`Turn` signature**: byte-identical to AG-07 (the new field lives on `TurnOptions`, not on `Turn` itself).
- **`TurnOptions` signature**: gains one field (`PreRequestHook`). Zero-value = identity default (D4a). Non-breaking change.
- **No external caller of `PreRequestHook` exists yet** — AG-08 is the first user.
- **No Layer 3 wiring required at AG-08 time** — CO-24 (Layer 3 cache-breakpoint placement) is doc 0004's scope, depends on the seam existing, runs separately.
- **No doc updates required at AG-08 time** beyond this design + the milestone doc's eventual amendment at archive.

Rollout: merge PR → AG-09 (parallel, no dependency), AG-20 (depends on AG-08), doc 0004 CO-24 (Layer 3, depends on seam existing).

## Open Questions

- [ ] **Test file location**: new `loop_hook_test.go` vs appending `loop_test.go`. Recommend new file (`loop_hook_test.go`) for grep-discoverability of AG-08 tests, mirroring AG-07's single-file pattern would put them all in `loop_test.go`. Decision deferred to `sdd-tasks` based on how many helpers the AG-08 tests can reuse vs need to introduce.
- [ ] **Hook failure `FailureCategory`**: proposal suggests `FailureCategoryUnsupportedCapability` (hook-attributing; not provider-auth). Alternatives: `FailureCategoryUnknown` (catch-all). Decision deferred to `sdd-tasks` based on whether the AG-08 hook may legitimately return failure for an unsupported capability (e.g. cache-breakpoint placement when the request lacks the marker region).

## Next step

Launch `sdd-tasks` next — write `tasks.md` with phases 1–5, the 8-commit work-unit breakdown, and the test-order ratchet from RED bites → GREEN property → RED-then-GREEN remaining scenarios. Review Workload Forecast at 350–600 lines added, 400-line budget risk Low, chained PRs No, single PR under pre-authorized `size:exception`.