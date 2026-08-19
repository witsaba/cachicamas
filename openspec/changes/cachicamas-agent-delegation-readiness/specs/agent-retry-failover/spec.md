# Delta for `agent-retry-failover` — the subagent-retry row is RE-OWNED, because it named AG-19 as an owner AG-19 does not deliver

> **Change**: `cachicamas-agent-delegation-readiness` · **AG-19** (Layer 2, Wave 5), `0003:1793-1862`
> **Modifies**: `agent-retry-failover` ([`../../../../specs/agent-retry-failover/spec.md`](../../../../specs/agent-retry-failover/spec.md)) — the non-requirement row at `spec.md:227` only.
> **BACK-ANNOTATION ONLY, and scenario-free.** No requirement is amended, no gate moves, no scenario is added or removed, and no count changes.
> **This delta corrects an owner claim rather than merely annotating a deferral, and that is why it is not optional.** The row reads *"Subagent-scoped retry | **AG-19.** No subagent tool ships in v1"*. **AG-19 ships no retry surface of any kind** — no gate, no budget, no backoff, no failover consult, and no per-child anything. Leaving the row pointing at AG-19 after AG-19 merges would tell a later reader that subagent-scoped retry either shipped or is still coming from a milestone that has already closed. It is neither: it is **post-v1**, on the substrate AG-19 proved. The row is re-owned rather than deleted so the correction is visible, following this capability's own price-table precedent (`agent-protocol-events:156`).
> **Ownership**: what AG-19 does ship is owned by [`../agent-delegation-readiness/spec.md`](../agent-delegation-readiness/spec.md). This delta owns only the correction.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-RTY-001`'s ordered, first-match-wins gate predicate, G0–G5 | **Untouched.** AG-19 adds no gate, moves none, and evaluates none. A hosted child run drives its **own** attempt loop through its own `Harness`, using the retry policy its own caller configured |
| `R-RTY-002`'s attempt-is-a-turn-bracket rule | **Confirmed.** A child's attempts are turn brackets on the **child's** lane; the seam refuses turn-bracket kinds, so no child attempt bracket reaches the parent's lane |
| `R-RTY-003`'s no-retry-after-emitted-output rule | **Untouched.** AG-19 emits no provider output and consults no retry decision |
| `R-RTY-010` / `R-RTY-011`'s failover seam and its declining v1 implementation | **Untouched.** AG-19 touches no failover surface; seam 8 stays deferred |
| The cost-accounting row closed by AG-16 | **Still true at AG-19**, whose refusal of `cost_turn` at the seam keeps `R-CST-004`'s sum-over-emitted-events rule literally true on the parent's lane |
| Every count in this capability | **Unchanged.** AG-19 adds no requirement and no scenario here |

## MODIFIED Explicit non-requirements

The list is reproduced only where AG-19 touches it; every other row is unchanged and none is removed.

- **Subagent-scoped retry** — was: *"**AG-19.** No subagent tool ships in v1 (`0003:1794`). (Still true at AG-16; parent-scoped cost aggregation over delegated runs is likewise AG-19's, `0003:1533`)"* → **RE-OWNED to post-v1. AG-19 is NOT its owner, and the correction is recorded rather than propagated.**
  - **What AG-19 shipped**: a publishing seam and proof of four structural properties under a nested run. **It ships no retry surface at all** — no gate, no budget, no backoff, no failover consult, no per-child retry policy, and no notion of retrying a child run as a unit. Its charter (`0003:1799-1803`) names retry nowhere, and its production diff adds no retry identifier.
  - **What subagent-scoped retry would require, and why it is not smuggled in here**: a decision about whether a failed child run is retried as a unit or whether its own attempt loop suffices, whether a retried child re-publishes its `subagent_started`, and what a parent's stream should show for a discarded attempt's mirrored events. **None of those questions has an owner today**, and answering any of them by implication in AG-19 would be exactly the kind of silent scope creep this table exists to prevent.
  - **Owner: post-v1**, alongside the production subagent tool whose `AGS-D` entry in `agent-v1-scope` remains deferred with AG-19 named as its placeholder node. A milestone that takes this on inherits AG-19's substrate; it does not inherit a decision, because AG-19 made none.
  - **The row's parenthetical is likewise corrected.** *"Parent-scoped cost aggregation over delegated runs is likewise AG-19's"* — **AG-19 answered that, and the answer is a permanent NO at Layer 2**: no Layer-2 fold occurs on any path, the seam refuses the `cost_turn` kind in production code, and the combined figure is a consumer-side reconstruction over two independently validated streams. See `agent-cost-events`' `R-CST-007` as amended by this same change. A reader must not carry the old parenthetical forward as an outstanding obligation.
