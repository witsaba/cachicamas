# Delta for `agent-event-envelope` — AG-19 lands invariant 2's second half, and two universal claims are re-scoped rather than left false

> **Change**: `cachicamas-agent-delegation-readiness` · **AG-19** (Layer 2, Wave 5), `0003:1793-1862`
> **Modifies**: `agent-event-envelope` ([`../../../../specs/agent-event-envelope/spec.md`](../../../../specs/agent-event-envelope/spec.md)) — `R-AEV-003` (`spec.md:55-65`) and the invariant-2 row of the non-requirements table (`spec.md:257`).
> **This delta amends two claims that AG-19 genuinely falsifies.** Both are the same repository failure shape — a universally quantified sentence written before the producer that would violate it existed:
> 1. `R-AEV-003`'s *"an event belonging to a delegated harness MUST carry its parent identifier"* (`spec.md:57`). AG-19 mirrors the child's message, tool, permission, cost and compaction events onto the parent's stream, and **none of them carries a parent identifier** — only three constructors in the whole 25-kind vocabulary accept one. Read literally the sentence is false the day AG-19 merges.
> 2. `S-AEV-022`'s *"it declares no delegation, subagent, or child-harness mechanism"*. AG-19 exports `DelegationSeam` and `DelegationSeamFrom`. The word **delegation** is now in the package's exported surface, so the scenario's assertion fails on a literal reading — even though the claim it was written to protect (no subagent mechanism) is exactly as true as before.
> **`R-AEV-002`'s contiguity is UNAMENDED and holds by mechanism.** A mirrored event is re-stamped by the parent's single stamping writer, and re-stamping discards the prior sequence and returns a copy (`sequence.go:50-58`), so both lanes stay independently contiguous and 1-based. `Event` is a value type; publishing a copy cannot perturb the child's lane.
> **Header maintenance obligation at promotion.** `sdd-archive` MUST record **`S-AEV-125`** wherever the target spec enumerates allocated `S-AEV-` identifiers, in the same commit that promotes this delta. It was verified free in this phase: the spec's highest allocated identifier is `S-AEV-124`.
> **Ownership**: the walkable-tree mechanism is owned by [`../agent-delegation-readiness/spec.md`](../agent-delegation-readiness/spec.md) (`R-DEL-004`). This delta owns only what the envelope contract must now say.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-AEV-002` / `S-AEV-010` / `S-AEV-011` — per-lane contiguity | **Confirmed rather than amended.** Two lanes stamped concurrently remain independently contiguous; AG-19's sibling scenario (`S-DEL-013`) is a third `-race`-proven instance of exactly that claim |
| `R-AEV-004` — exactly one run-start first, one run-end last | **Confirmed.** The seam's admissibility rule refuses every bracket-opening, bracket-closing and terminal kind (`R-DEL-002` gates 2 and 4), so no mirrored event can be a second run-start or a premature terminal |
| The parent identifier's **field, accessor and construction surface** | **Byte-unchanged.** No constructor gained a parent parameter, `event.go` is byte-unchanged, and AG-19 registers no `EventKind` |
| `S-AEV-021` — a top-level event reports no parent as a distinguishable false | **Untouched, and this is why the literal reading of `R-AEV-003` had to go.** Making every child event carry a parent would require every constructor to take one, which would collapse the distinguishable-false state this scenario pins |

## MODIFIED Requirements

### R-AEV-003 — The parent identifier exists before any delegation mechanism does

The envelope MUST carry a parent identifier from birth: an event **constructed as belonging to a delegated harness** MUST carry its parent identifier, and a top-level event MUST carry none (`VL2-EVT-13`, `decision.md:174`). The field exists now because explicit nesting cannot be retrofitted.

The absence of a parent identifier on a top-level event MUST be distinguishable from an unset or zero-valued identifier by inspection, not by convention.

**AG-19 lands the delegation semantics this requirement held a place for, and it lands them as a WALKABLE TREE rather than as a per-event stamp.** The distinction is normative, not stylistic, and it MUST be restated in these terms wherever a later milestone reads this requirement:

- **The parent-bearing surface is exactly three constructors** — `NewDelegatedRunStart`, `NewSubagentStarted` and `NewSubagentEnded` (`delegation_events.go:62,134`). Every message, tool, permission, cost, turn and compaction constructor takes no parent parameter and never sets one (`permission_events.go:138,242,326`; `cost_events.go:193,296`). **AG-19 adds none**, and no constructor gained a parent parameter in that change.
- **Attribution is therefore one hop, not one field.** Every child event carries `Run()` equal to the child run identity; exactly one `subagent_started` on the parent's stream carries that same `Run()` together with `Parent()` equal to the parent run identity; so every child event resolves to the parent through that single event (`R-DEL-004`).
- **A mirrored child event on a parent's stream reports no parent, and that is correct rather than a gap.** It is a child-run event delivered on the parent's lane, not a top-level event: the discriminator that distinguishes the two is `Run()`, and `R-AEV-004`'s single run bracket per lane guarantees the parent's own run identity is unambiguous on that lane.
- **The literal reading is rejected on evidence.** Making every child event carry a parent identifier requires rewriting the whole construction surface, which is outside AG-19's dependency set, and it would falsify `S-AEV-021` by removing the top-level "no parent" state's meaning. That is a vocabulary-wide change and belongs to a different milestone, not to this one.

(Previously: the requirement said *"an event belonging to a delegated harness MUST carry its parent identifier"* without qualification, which AG-19's mirrored child events — message, tool, permission, cost and compaction kinds, none of which has a parent-bearing constructor — falsify on their face. The clause is now scoped to construction, and the attribution obligation it was reaching for is stated explicitly as the one-hop walk.)

#### Scenarios

- **S-AEV-020** — Given the envelope surface, when an event is constructed as belonging to a delegated harness, then it carries its parent identifier and that identifier is readable from an external package. *(AG-19 update: the three parent-bearing constructors are unchanged and this scenario's falsifiable claim is exactly what it was; AG-19 is the first production caller of two of them.)*
- **S-AEV-021** — Given an event constructed as top-level, when its parent identity is inspected, then it reports "no parent" as a distinguishable state rather than an ambiguous zero value.
- **S-AEV-022** — Given the package after the AG-04 change, when its exported surface is enumerated, then it declares no delegation, subagent, or child-harness mechanism — only the parent identifier field and its accessor. *(AG-19 update: this scenario's GIVEN is scoped to the AG-04 change, which is what it always meant; AG-19 is the milestone that ends the "no mechanism" state and does so deliberately. See `S-AEV-125` for the standing claim that replaces it.)*
- **S-AEV-125** — **AG-19: what the package declares after the seam ships, and what it still does not.** Given the package after AG-19, when its exported surface is enumerated from an external test package, then it declares exactly one delegation mechanism — a publishing seam interface, its context accessor and two sentinel errors — and **no** subagent type, child-harness type, delegation depth, delegation configuration or subagent lifecycle; no exported identifier names a subagent concept; and no external caller can construct or install a seam the scheduler will honour. Cross-referenced to `R-DEL-001` / `S-DEL-002`.

## MODIFIED Explicit non-requirements

The invariant table is reproduced in full; one row is back-annotated as closed and none is removed or renumbered.

| Envelope invariant | Closed by | AG-04's part |
| --- | --- | --- |
| 1 — indexed deltas | AG-04.3 **+ AG-05.1** | the construction-surface pin only (`R-AEV-007`) |
| 2 — explicit nesting | AG-04.1 **+ AG-19.1 — CLOSED** | the parent identifier exists; **AG-19.1 delivered the delegation semantics** as a walkable tree in one hop (`R-DEL-004`), with the first production emission of `subagent_started`/`subagent_ended`, and **no** per-event parent stamp (`R-AEV-003` as amended) |
| 3 — non-blocking observers | **AG-01.1 + AG-20.2 — AG-04 absent** | **none.** No requirement here closes any part of invariant 3 |
| 4 — typed errors | AG-04.3 **+ AG-11.2** | the typed-failure surface exists; loop-level emission is AG-11's (`R-AEV-008`) |

**AG-04 closes no envelope invariant by itself.** No requirement, scenario, test name or acceptance line may claim otherwise. *(AG-19 update: invariant 2 is now closed **jointly**, exactly as this table always said — AG-04.1 built the field, AG-19.1 gave it meaning. The row must not be re-read as "AG-04 closed it".)*
