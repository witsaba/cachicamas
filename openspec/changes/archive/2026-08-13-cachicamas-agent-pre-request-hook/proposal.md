# Proposal: AG-08 — Add the pre-request hook seam

## Intent

AG-08 owns **seam 1 of v2 § 6** — the only point in the loop where the outgoing request still exists as data, between `buildLoopRequest` (`loop.go:132`) and `provider.Stream` (`loop.go:140`). It closes **R-12** (G4's Layer 2 half: prompt-cache prefix stability) by giving Layer 2 a single, narrow mutation point that Layer 3's breakpoint placement (doc 0004 CO-24) and other consumers will stand on. AG-08 is the first real consumer of Layer 1's AI-12 copy-on-write rebuild (`Request.With(...)`, R-REX-001) and the first public-surface extension of `TurnOptions` since AG-07. AG-04/05/06/07's stream-validator tests fed the validator hand-built events; AG-08's hook is the first live mutation of the request the loop hands to the provider.

## Why now

AG-07 produced the seam; AG-08 wraps it. AG-20 (the four-hook taxonomy registration surface) blocks on AG-08 — the chain composition widens from this single hook. AG-09 (tool execution contract) runs in parallel and wraps `Turn` from the inside (scheduler), not the outside (hook) — no dependency.

## Scope

### In scope
- `TurnOptions.PreRequestHook` field (`func(ctx context.Context, req ai.Request) (ai.Request, error)`)
- `applyPreRequestHook(ctx, req, hook) (ai.Request, error)` helper
- Two-line branch in `Turn` between `buildLoopRequest` (`loop.go:132`) and `provider.Stream` (`loop.go:140`)
- Hook-failure path mirroring pre-stream-failure (`loop.go:140-147`): close `sink`, return `(ai.Message{}, 0, typedErr)` reusing `ai.PreStreamFailure`
- Tests for 6 charter scenarios + AG-07 W1 unbuffered-sink carry-forward + AG-05 W1 bite pattern

### Out of scope
- The other three hook points (AG-20 widens to chain composition — its scope)
- Concrete cache-breakpoint placement (Layer 3 wiring — doc 0004 CO-24.1)
- Translation interface changes (AG-07 SUGG 4 — parked; AG-13 may re-introduce)
- Tools / permission / retry / cost / context-check (AG-09, AG-10, AG-11, AG-15, AG-16)

## Capabilities

### New
- `agent-pre-request-hook`: the pre-request hook seam + identity default + prefix-stability guarantee. New `openspec/specs/agent-pre-request-hook/spec.md` (full spec).

### Modified
- None. The 21 substrate files stay byte-identical.

## Approach

The seam wraps `Turn` at the single point where `req` is fully assembled and `provider.Stream` hasn't been called yet. The hook receives the assembled `ai.Request` and returns a derived `ai.Request` via `req.With(...)` copy-on-write rebuild (R-REX-001, `request.go:325-336`) — a hook that calls `With(...)` returns a fresh value; the loop's `req` is left observably unmodified. Identity default: nil hook = no-op; identical inputs → identical outputs (R-LSK-002 byte-stability preserved; AG-08.1 #2 asserts this). Typed hook failure reuses `ai.PreStreamFailure` (`provider_failure.go:34-93`) and takes the existing pre-stream-failure return path verbatim — close `sink`, return zero. AG-08.1 #3 writes the failing-hook test FIRST (RED before GREEN per AG-05 W1 bite pattern), asserting `len(provider.Requests()) == 0` AND the sink drains unblocked. AG-08.2 #1's "tools → system → messages cascade" byte-stability uses `ai.Request.Equal` (`request.go:555-627`) for region byte-equality + `ai.Request.CacheBoundaries()` (`cache_boundary.go:118-120`) for cascade-order pin (R-ACB-007 contract). Message-region growth check uses `Message.Equal` + length diff (`Equal` excludes `MessageID` by design).

Tests use `agenttest.Provider.Requests() []ai.Request` (`fake_provider.go:157-161`) to inspect what the provider actually received. AG-08 inherits AG-07's W1 carry-forward: ≥1 test uses an **unbuffered** sink + a concurrent consumer goroutine + a `runtime.NumGoroutine()` before/after assertion to close the back-pressure path. NFR-PRH-001: every AG-08 test lives in `package agent_test`.

## Decisions log

| # | Decision | Adopted value | Rationale |
|---|---|---|---|
| **D1** | Hook surface shape | Single callable on `TurnOptions`: `PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)` | Function-form matches AG-07's `Turn` surface (D1, AG-07) and AG-07's `buildLoopRequest` helper. Charter says "the hook" (singular). AG-20 widens to chain composition — its scope, not AG-08's. |
| **D2** | Where the seam wraps `Turn` | Between `buildLoopRequest` (`loop.go:132`) and `provider.Stream` (`loop.go:140`). Failure path mirrors existing pre-stream-failure path (`loop.go:140-147`): close `sink`, return `(ai.Message{}, 0, typedErr)`. | Pre-stream-failure is the existing typed-error precedent; a hook failure must look identical to a provider pre-stream failure from the caller's side. Two-line change inside `Turn`. |
| **D3** | Hook invocation signature | `func(ctx context.Context, req ai.Request) (ai.Request, error)` — passes the loop's own `ctx`. | Cheap to carry; forecloses cancellation/deadline/tracing. AG-07 D5a established `ctx` pass-through. |
| **D4** | Identity default | Nil hook = no-op (skip the seam, proceed to `provider.Stream(ctx, req)` unchanged). Zero-value `TurnOptions` produces byte-identical output to AG-07's skeleton. | AG-07 R-LSK-002 byte-stability on identical inputs is the precedent; AG-08 must preserve it (AG-08.1 #2 asserts this). |
| **D5** | Spec prefix | New prefix `R-PRH-` (pre-request hook); scenarios `S-PRH-NNN`. | Distinct prefix signals different requirement closure (R-12) from AG-07's `R-LSK-001..005`. Two-letter match to slug. AG-09/10/11 open their own; AG-20 gets `R-HK-`/`R-FHT-`. |

## Carry-forwards from AG-04/05/06/07

| Source | Finding | AG-08 mitigation |
|---|---|---|
| **AG-07 W1** | Every AG-07 test uses a **buffered** sink; concurrent-consumer / back-pressure path unproven (`verify-report.md:175`) | **MUST add ≥1 unbuffered-sink test for the hook path** + `runtime.NumGoroutine()` baseline check. Hook is the natural gating point. |
| **AG-07 W6** | External-package test posture (`archive-report.md:28`) | Every AG-08 test in `package agent_test`. NFR-PRH-001. |
| **AG-07 SUGG 4** | `translate()` could become a method on `providerEventTranslator` | Hook wrap is NOT the translation path. SUGG 4 stays parked — AG-13 may re-introduce. |
| **AG-07 W4** | `mintLoopMessageID` discards two errors (latent) | AG-08 doesn't touch. Carries to AG-23 (typed minting bridge). |
| **AG-05 W1** | Vacuous reconstruction helper | AG-08.1 #1 ("hook shapes outgoing request") is the load-bearing property test. Two bites: hook-doesn't-add-segment, hook-returns-same-request — both RED before GREEN. |
| **AG-04 W9** | Scenario count drift | State identically across proposal/spec/tasks/apply-progress/verify-report: **6 charter → 8–12 spec**. |
| **AG-07 W2** | `TestTurn_CoverageGate` is a `t.Skip` marker; real gate is `make test/cover` | Gate enforced by `make test/cover`. Forecast: `loop.go` coverage stays ≥ 80% (AG-07 hit 85.89%; AG-08's +20–40 lines). |
| **AG-07 W3** | `TestTurn_SubstrateUntouched` hard-codes `8420b2c4`; ref goes stale on merge | AG-08's substrate-untouched test uses `AG-08 BASE_REF` env-var fallback (shipped in AG-07 PR #167 W3 fix) + dynamic merge-base as default. |

## Forecast

**Estimated changed lines: 350–600 added** (well under 1000-line budget with `size:exception` pre-authorized; smaller than AG-07's 1816).

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/loop.go` | MODIFY | `TurnOptions.PreRequestHook` field + `applyPreRequestHook` helper + 2-line branch in `Turn`. Forecast +20–40 lines (loop.go 457 → ~480). |
| `backend/agent/src/agent/loop_test.go` (or new `loop_hook_test.go`) | MODIFY/CREATE | 6 charter scenarios + bites + AG-07 W1 unbuffered-sink carry. Forecast +300–500 lines. |
| 21 substrate files (AG-07 R-LSK-004 carry) | UNTOUCHED | 5th consecutive "substrate untouched" milestone. |
| `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` | UNTOUCHED | No new deps; no guard edits. |
| `openspec/specs/agent-pre-request-hook/spec.md` | NEW (sdd-spec) | Full spec for new capability. |
| `openspec/changes/cachicamas-agent-pre-request-hook/{proposal,design,specs,tasks}.md` | NEW | Phase artifacts. |

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/loop.go:46-51` | Modified | New `PreRequestHook` field on `TurnOptions` |
| `backend/agent/src/agent/loop.go:132-140` | Modified | Two-line branch wrapping `provider.Stream(...)` |
| `backend/agent/src/agent/loop.go` (new helper) | Modified | `applyPreRequestHook(ctx, req, hook) (ai.Request, error)` |
| `backend/agent/src/agent/loop_test.go` (or `loop_hook_test.go`) | Modified/Created | AG-08.1 + AG-08.2 + bites + AG-07 W1 unbuffered-sink carry |
| 21 substrate files (per AG-07 R-LSK-004) | UNTOUCHED | 5th consecutive extensibility demo |
| `openspec/specs/agent-pre-request-hook/spec.md` | NEW (sdd-spec) | New capability spec |
| `backend/agent/go.mod`, `go.sum` | UNTOUCHED | No new deps |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| **R1** — Hook return path must mirror pre-stream-failure exactly (close `sink`, return zero) | Low→Medium | Write AG-08.1 #3 failing-hook test FIRST. RED before GREEN (AG-05 W1 carry). Asserts `len(provider.Requests()) == 0` AND `sink` drains unblocked. |
| **R2** — `Request.With` derivation chain (R-REX-001): hook mutates slice from accessor; original loop request must be observably unchanged | Low→Medium | AG-08.1 #4 ("hook cannot mutate input in place") is the load-bearing test. Mutating hook + `Equal` byte-equality on captured request vs skeleton's. |
| **R3** — Prefix stability "message region grows by append" assumes the caller passes messages with monotonic append discipline | Low→Medium | Loop doesn't own this; AG-08.2 #1's test writes message-region growth externally; AG-12.1 will own the append-only-history discipline. |
| **R4** — Hook determinism (AG-08.2 #2): non-deterministic hook (e.g. `time.Now()`) would violate cascade byte-stability | Low | Combined with AG-08.2 #1's cascade byte-stability, determinism is closed. Timestamp-bearing region would differ between turns. |
| **R5** — `Request.Equal` excludes `MessageID` by design; for tools/system cascade `Equal` is correct; for message region growth use `Message.Equal` + length diff | Low→Medium | Stated in approach. AG-08.2 #1's test asserts tools/system `Equal` byte-equal across turns + message regions differ only in count (first N `Message.Equal` content-equal). |
| **R6** — `TurnOptions` field addition is a non-breaking public-surface change | Low | Zero-value = identity default (D4). No external caller of new field exists yet (AG-08 is the first user). |
| **R7** — Review budget 1000 lines | Low | Forecast 350–600. `size:exception` pre-authorized carries forward from AG-04/05/07. |
| **R8** — External-package test posture | Low | Every test in `package agent_test` (AG-07 W6 carry → NFR-PRH-001). |

## Dependencies

- **AG-07** walking skeleton + `Turn` — already merged at `93077c07` (PR #167)
- **AI-12** `Request.With(opts ...RequestOption) (Request, error)` at `request.go:325-336` — already shipped (PR #165)
- **AI-21** `agenttest.Provider.Requests() []ai.Request` at `fake_provider.go:157-161` — already shipped
- **AI-11** `ai.Request.CacheBoundaries()` at `cache_boundary.go:118-120` — already shipped
- **AG-03** boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) — must stay green untouched

## Acceptance criteria

- [ ] `TurnOptions.PreRequestHook` field callable; identity default byte-stable (AG-08.1 #1, #2)
- [ ] Hook observes + replaces outgoing request via `req.With(...)` (AG-08.1 #1)
- [ ] Failing hook aborts before I/O with typed error (reuses `ai.PreStreamFailure`); no half-mutated request sent (AG-08.1 #3)
- [ ] Hook cannot mutate loop's input in place (R-REX-001 / copy-on-write holds) (AG-08.1 #4)
- [ ] Tools + system regions byte-identical across turns via `Request.Equal`; cascade order pinned via `CacheBoundaries()` (AG-08.2 #1)
- [ ] Message region grows strictly by append across turns (AG-08.2 #1; caller discipline)
- [ ] Hook deterministic for identical inputs (AG-08.2 #2)
- [ ] AG-07 W1 back-pressure carry: ≥1 test uses unbuffered sink + concurrent consumer + `runtime.NumGoroutine()` baseline
- [ ] 21 substrate files byte-untouched (5th consecutive "substrate untouched" milestone)
- [ ] `make test` green in `backend/agent/` with `-race` (no new failures)
- [ ] `make lint` green, `make build` green, `make vuln-check` clean
- [ ] `loop.go` ≥ 80% line coverage (AG-04 W8 / AG-07 W2 carry)
- [ ] AG-03 boundary guards stay green untouched
- [ ] `go.mod` / `go.sum` byte-identical to main
- [ ] **6 charter → 8–12 spec** scenarios, stated identically across proposal, tasks, apply-progress, verify-report

## Rollback plan

Revert PR. Modified files: `loop.go` + `loop_test.go` (or `loop_hook_test.go`) + `openspec/specs/agent-pre-request-hook/spec.md`. Substrate byte-unchanged (5th "substrate untouched" bet) → clean revert. No data migration, no schema change. `TurnOptions` gains one field; reverting the field restores AG-07's surface verbatim.

## Next step

Launch `sdd-spec` next — write `R-PRH-001..005` + `S-PRH-001..NNN` (**8–12** scenarios + bites) at `openspec/specs/agent-pre-request-hook/spec.md`, mirroring AG-07's spec format (`openspec/specs/agent-loop-skeleton/spec.md`).