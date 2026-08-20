# Delta for `agent-compaction` — the parent-aggregation row is ANSWERED, and the compaction family joins the mirrorable set

> **Change**: `cachicamas-agent-delegation-readiness` · **AG-19** (Layer 2, Wave 5), `0003:1793-1862`
> **Modifies**: `agent-compaction` ([`../../../../specs/agent-compaction/spec.md`](../../../../specs/agent-compaction/spec.md)) — the non-requirement row at `spec.md:320` only.
> **BACK-ANNOTATION ONLY, and scenario-free.** No requirement is amended, no bracket order moves, no scenario is added or removed, and no count changes. Nothing in AG-19 touches compaction's mechanism.
> **The proposal recorded this row as "remains true; no delta". That assessment is CORRECTED here, on evidence.** The row reads *"Aggregating compaction spend into a parent or delegated run | **AG-19.** The run's own cumulative is in scope; nothing above it is."* Read after the AG-19 merge, it promises a future fold that **AG-19 forbids permanently**. The row is not false, but it is misleading in exactly the way an un-back-annotated deferral is: it points a reader at a milestone that has closed, for an outcome that milestone ruled out. One row, one sentence, and it removes a wrong expectation from a capability that will otherwise carry it indefinitely.
> **A second reason this delta exists**: AG-19's admissibility rule **admits all three compaction kinds** onto a parent's stream. That is the first time compaction events can appear anywhere other than the run that performed the compaction, and no requirement in this capability anticipates it. The verdict — it is safe, and why — is recorded below rather than left to be rediscovered.
> **Ownership**: the seam and its admissibility rule are owned by [`../agent-delegation-readiness/spec.md`](../agent-delegation-readiness/spec.md) (`R-DEL-002`, `R-DEL-005`). This delta owns only this capability's audit statement.

## Not modified, and why — including the mirroring analysis

| Rule | Verdict | The mechanism that keeps it true |
|---|---|---|
| `R-CMP-008` — the compaction bracket's event order, and `stream_check.go` byte-unchanged | **holds** | AG-19's seam refuses every kind carrying a bracket role, so a child's compaction `turn_start`/`turn_end` **cannot** cross. Only the three compaction-family kinds themselves are admissible, and all three are turn-placed, non-terminal, cardinality *any* — legal inside the parent's open turn |
| `R-CMP-009` — exactly one of finished or failed per operation, never both | **holds** | the rule is per **operation**, and a mirrored pair belongs to the child's operation. A consumer distinguishes them by `Event.Run()`, the same discriminator AG-19 mandates for cost |
| `R-CMP-003` — compaction spend inside the bracket, folded into the run cumulative | **holds** | a child's compaction `cost_turn` is **refused** at the seam (`R-DEL-002` gate 5) exactly like any other `cost_turn`, so it reaches the child's own cumulative and never the parent's |
| `R-CMP-001`'s two-doors rule and `R-CMP-011`'s run-in-flight refusal | **holds** | AG-19 opens no third door. A child harness's on-demand door is the child's own, gated by the child's own in-flight state |
| `R-CMP-014` — compaction is inert unless requested | **holds** | AG-19 requests no compaction anywhere and installs no strategy |
| `R-CMP-013`'s byte-unchanged list, including `compaction_events.go` | **holds** | AG-19 edits none of them and registers no `EventKind` |

**Why the three compaction kinds are admitted rather than refused**, stated because a narrower rule was available: a child's compaction is legitimately visible to the human watching the parent's stream — it explains a gap in the child's transcript that would otherwise look like lost context. The rule is **derived from the registry** rather than hand-listed, so the compaction family is admitted by the same descriptor evidence that admits the message and tool families, with no special case for it.

## MODIFIED Explicit non-requirements

The list is reproduced only where AG-19 touches it; every other row is unchanged and none is removed.

- **Aggregating compaction spend into a parent or delegated run** — was: *"**AG-19.** The run's own cumulative is in scope; nothing above it is."* **ANSWERED by AG-19, and the answer is a permanent NO at Layer 2 rather than a hand-off.** No Layer-2 fold of any child figure into any parent figure occurs on any path: AG-19's publishing seam **refuses the `cost_turn` kind in production code**, so a child's compaction spend — like every other child `cost_turn` — cannot reach a parent's cumulative even by accident. A combined figure, if a frontend wants one, is a **consumer-side reconstruction** over two independently validated streams joined by walking parents. **This row therefore closes as "never at Layer 2", not as "AG-19 will do it"**, and no later milestone may reopen the fold without amending `R-CST-004` first. *(Unchanged and still true: this capability's own scope is the run's own cumulative, and nothing above it. What changed is that "above it" now has a settled answer instead of a pending owner. See `agent-cost-events`' `R-CST-007` as amended by this same change.)*
