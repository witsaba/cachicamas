# Exploration — AG-20: Complete the hook taxonomy

> **Change**: `cachicamas-agent-hook-taxonomy` · **Milestone**: AG-20 (Layer 2, Wave 5) · **Closes**: G11 (R-17)
> **Phase**: `sdd-explore` · **Artifact store**: hybrid (Engram topic `sdd/cachicamas-agent-hook-taxonomy/explore`)
> **Orchestrator note**: the `sdd-explore` phase agent had no filesystem write capability in this run. It persisted the analysis to Engram; the orchestrator materialized this file and independently re-verified every load-bearing citation below against the worktree before `sdd-propose` consumed it. Verification results are marked **[verified]**.

## Current state

### The shipped pre-request hook (AG-08)

A single nil-default func field on `TurnOptions`:

- `R-PRH-001` — `TurnOptions.PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)`, nil = identity default (`openspec/specs/agent-pre-request-hook/spec.md`, `backend/agent/src/agent/loop.go:46-51`).
- Invoked once per `Turn` by `applyPreRequestHook` (`backend/agent/src/agent/loop.go:739-748`), sitting between `buildLoopRequest` and `provider.Stream` — v2 § 6 seam 1, the only point where the outgoing request still exists as data.
- Failure aborts pre-I/O as an `ai.PreStreamFailure` carrying a hook-attributing `Category` (`R-PRH-003`).

**[verified]** The AG-08 spec's Purpose section states, verbatim: *"the seam is a single callable on `TurnOptions` of type `func(ctx context.Context, req ai.Request) (ai.Request, error)` with a nil = identity default; **AG-20 widens to chain composition**."* The widening is a commitment AG-08 already made on AG-20's behalf, not a new idea introduced here.

### The compaction seam (AG-13 / AG-18)

**[verified]** `runCompaction` (`backend/agent/src/agent/compaction.go:252-341`) always routes its naive cut through `resolveCut` (`compaction.go:51-79`) before use. `resolveCut` **is** the invariant-safe surgery the charter names: it is pure, retraction-only (mark-boundary retraction, then open-pair straddle retraction, re-verifying from the top), and structurally terminating because `straddle < cut` strictly decreases each iteration.

Consequence: a pre-compact hook spliced immediately **before** `runCompaction` is called — at the verdict-acting gap in `harness.go:540-560`, where `if verdict.Compaction != nil { _ = runCompaction(...) }` — inherits span revalidation for free. Scenario 2 of AG-20.1 ("the adjusted span is revalidated by the invariant-safe surgery before use") needs **no new validation code**, only a splice at a point upstream of the existing `resolveCut` call.

### The turn boundary (post-turn)

`TurnOutcome` and `CostFigures` are two distinct payload types riding two distinct events (`turn_end`, `cost_turn`). No existing Go type bundles "the turn's outcome and cost". A post-turn observing hook needs one new small aggregate type; it does not need a new event kind.

### The harness boundary (session-start)

**[verified]** `Harness` is value-form with no constructor and no identity field distinct from the `RunID` it mints at `Run()` entry (`harness.go:385,467`). A delegated/child run is a **brand-new `Harness` struct literal** (`cancellation_test.go:65`, `delegation_seam_test.go:341`) minting its own fresh `run-hrn-` id through the same counter.

There is therefore **no existing concept of harness identity distinct from run identity**. "Session-start fires once per harness before the first turn" is a genuine open design decision for nested runs, not a coding gap — the charter sentence cannot be implemented by reading existing state. This is the single highest-value question for `sdd-design`.

### Envelope invariant 3

**[verified]** `openspec/specs/agent-event-envelope/spec.md:268` reserves it explicitly in the non-requirements table:

| Envelope invariant | Closed by | AG-04's part |
| --- | --- | --- |
| 3 — non-blocking observers | **AG-01.1 + AG-20.2 — AG-04 absent** | **none.** No requirement here closes any part of invariant 3 |

AG-20.2 is invariant 3's joint closer. The milestone doc's own checklist row (`0003:2163`) agrees.

### The closed-count fence — the decisive constraint

**[verified]** The event registry stands at **exactly 25 kinds** (4 AG-04 + 11 AG-05 + 10 AG-06), and that count is asserted as a closed quantity in at least six independent places:

- `R-AEV-010` / `S-AEV-090` — "exactly 25 kinds ... and no further kind of any name"
- `R-AEV-013` / `S-AEV-120` — the same, restated for AG-06
- `agent-event-envelope/spec.md:144` and `S-AEV-075` — AG-11's co-closure: "No new `EventKind` is registered for failure ... the guard stays at 25 kinds"
- `R-CST-007` (`agent-cost-events/spec.md:174`) — AG-16's scope fence: token-only, **no new kind**
- `R-DEL-009` (`agent-delegation-readiness/spec.md:234`) and `R-DEL-010` (closed-sequence safety) — AG-19's fence, with its admissibility rule derived over "the 25 kinds registered today"
- `S-AEV-092` / `S-AMT-081` / `S-APE-081` — the bites that make the fence bite by count

Registering a 26th kind for a stalled-observer report would falsify all of them and require an additive delta on each. Two shipped precedents (AG-11's typed failure, AG-16's cost figures) already chose the other branch: carry the new information on an existing typed surface rather than mint a kind.

### Synchronization primitive for AG-20.2

`agenttest.Gate` (`backend/agent/src/agenttest/fake_gate.go:20`) is used pervasively across AG-13/17/18/19 tests to hold a goroutine at a barrier while asserting unimpeded progress elsewhere. It is the right mechanism for the stalled-observer proof.

Note the distinction: the existing `S-PRH-007` back-pressure test (`loop_hook_test.go:711-792`) proves *no stranded producer* on an unbuffered sink — a related but different property. It does **not** prove observer asynchrony, so AG-20.2 is not already covered.

### "Eventually" — a precedent that cuts against a timeout

**[verified as a constraint to reckon with]** `R-RUN-010` (`agent-run-driver/spec.md:229`) bans a wall-clock timeout on a structurally similar suspended-call seam: *"no third path and no timeout ... by design"*. There is no wall-clock timeout precedent anywhere in the module. Whatever "eventually" means for a stalled observer, it must either follow that precedent (structural, not temporal) or carry a citation-backed justification for departing from it.

## Affected areas

| Area | Path | Why |
| --- | --- | --- |
| Loop seam | `backend/agent/src/agent/loop.go` | existing pre-request seam; chain widening |
| Harness | `backend/agent/src/agent/harness.go` | session-start, post-turn, pre-compact splice points |
| Compaction | `backend/agent/src/agent/compaction.go` | pre-compact consumer of `resolveCut` |
| Typed failure | `backend/agent/src/agent/failure.go` | typed report for a stalled observer |
| Specs (certain) | `openspec/specs/agent-event-envelope/spec.md` | invariant 3 closure |
| Specs (certain) | `openspec/specs/agent-pre-request-hook/spec.md` | the chain-composition widening AG-08 forecast |
| Specs (check) | `agent-loop-skeleton`, `agent-compaction`, `agent-run-driver`, `agent-turn-termination`, `agent-cost-events`, `agent-delegation-readiness` | closed-sequence and absolute-quantifier requirements |
| Doc contract | `doc_contract_guard_test.go` | only if a new package-wide guarantee row is declared |
| Tests | `loop_hook_test.go` + new hook test files + `invariant_pin_test.go` | strict TDD landing zone |

## Approaches — the registration surface

1. **Four independent slice fields.** Matches the repo's struct-field convention; minimal new surface. Does not literally satisfy "one registration surface". Effort: low.
2. **One `Hooks` struct with four named slots.** Literally one surface. Forces a migration decision for the already-shipped singular `PreRequestHook` field. Effort: medium.
3. **Split by mutation contract** — distinct function-type families for mutating vs observing hooks. Encodes the charter's central discipline at compile time; needs (2) as an umbrella to be "one" surface. Effort: medium.

**Recommended: 2 + 3 combined.** One `Hooks` type holding two hook-kind families — mutating (chained, first error aborts, return value consumed) and observing (non-blocking, return value not consumed) — with the `TurnOptions.PreRequestHook` migration flagged explicitly to `sdd-design` rather than assumed.

## Recommendation to `sdd-propose`

- Combined registration surface (2 + 3), mutation discipline encoded in the type system.
- Pre-compact seam at the `harness.go` verdict gap, reusing `resolveCut` for revalidation.
- Post-turn seam at the harness's cost-forwarding site, with one new small outcome+cost aggregate type.
- Session-start's "once per harness" for nested runs flagged as an **unresolved design decision**, not assumed.
- Stalled-observer report as a Go-side typed value, **not** a 26th `EventKind`, given the six-site closed-count fence and the AG-11/AG-16 precedents.
- `agenttest.Gate` as AG-20.2's synchronization mechanism.

## Risks

1. **Session-start identity.** No existing type distinguishes a harness from a run. A real design decision; must be settled before `sdd-spec`.
2. **Stalled-observer stream visibility.** Roughly a 5–6× difference in spec blast radius depending on the answer. Must be resolved before `sdd-spec`, not during it.
3. **`TurnOptions.PreRequestHook` is already shipped** (AG-08, PR #167). Any widening must be additive, or the break must be explicit and deliberate.
4. **`R-RUN-010` precedent against timeouts** on a structurally similar seam. "Eventually" needs a citation-backed justification either way.
5. **`Harness` field-surface pinning** — verify whether reflection pins beyond the four-method pin at `harness_test.go:1018-1024` constrain adding hook-registration fields.
6. **Closed-sequence drift.** Every requirement phrased with an absolute quantifier ("exactly N", "only", "iff", closed enumerations) must be grepped before a fourth hook point or a new typed value lands — this repo has been bitten by exactly this class before.

## Ready for proposal

Yes. All six risks are named, and risks 1 and 2 are explicitly routed to `sdd-design` as decisions rather than carried forward as assumptions.
