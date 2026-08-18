# Delta for `agent-v1-scope` — AG-15.3 records that the failover placeholder node shipped its seam

> **Change**: `cachicamas-agent-retry-failover` · **AG-15** (Layer 2, Wave 3), `0003:1444-1525`
> **Modifies**: `agent-v1-scope` ([`../../../../specs/agent-v1-scope/spec.md`](../../../../specs/agent-v1-scope/spec.md)) — `R-AGS-006` (`spec.md:126-133`).
> **Back-annotation only.** The requirement's normative sentences and **both** of its scenarios are reproduced **verbatim** and are not amended. `AGS-D` still holds three deferrals, the failover **implementation** is still deferred, and `S-AGS-022`'s claim that AG-15.3 is the failover deferral's placeholder node is still exactly true. The only addition is a dated annotation recording that the placeholder node has now shipped the **seam** it holds the place for — so a later reader does not read "AG-15.3 is the placeholder" and conclude nothing exists.
> **Format**: the archive step REPLACES this block in the main spec; **full-block preservation is mandatory**.
> **Ownership**: the seam itself is owned by [`../agent-retry-failover/spec.md`](../agent-retry-failover/spec.md) (`R-RTY-010`, `R-RTY-011`).

## MODIFIED Requirements

## R-AGS-006 — The deferred list restates doc 0003's own deferrals with identifiers

The artifact MUST record as `AGS-D` the three items doc 0003's "Explicitly deferred" table already cites to AG-02: a production subagent tool with delegation depth limits, the failover implementation, and compaction quality. Each MUST name the seam or node that holds its place and the reason for deferral, and each MUST be traceable to its row in that table.

**Back-annotation (AG-15, 2026-08-18) — the failover row's placeholder node has shipped its seam; the deferral stands.** `AGS-D`'s failover entry names AG-15.3 as the node holding the place for the failover **implementation**. AG-15.3 has now shipped that place: a named injection point on the caller-owned harness, consulted exactly once when the retry budget for a logical turn is exhausted, whose v1 implementation **declines** and whose typed verdict makes acceptance unconstructible in v1 (`R-RTY-010`), together with a pin proving the seam's presence changes no observable behavior (`R-RTY-011`). **What remains deferred is unchanged and is the whole of the entry's substance**: choosing a substitute route, re-counting the context budget, re-pricing, and restarting the cache prefix — the obligations the interface's documentation now names for a real implementer, and which cross into AG-17/Layer 3 territory per seam 8's rationale (`0003:1454`). The entry MUST NOT be removed, reworded as closed, or re-owned: a shipped seam with a declining implementation is a held place, not a delivered capability.

### Scenarios

- **S-AGS-021** — Given the artifact, when a reviewer greps doc 0003's "Explicitly deferred" table for `AG-02`, then each of the three rows returned corresponds to exactly one `AGS-D` entry, AND no `AGS-D` entry lacks such a row.
- **S-AGS-022** — Given each `AGS-D` entry, when a reviewer reads it, then the placeholder node is named (AG-19 for the subagent tool, AG-15.3 for failover, AG-18.1 for compaction quality) AND the reason is stated in terms of what deferring protects, not in terms of effort.
