# Delta for `agent-event-envelope` — AG-20.2 lands invariant 3's second half, and the row CLOSES jointly with AG-01.1

> **Change**: `cachicamas-agent-hook-taxonomy` · **AG-20** (Layer 2, Wave 5), `0003:1864-1918`
> **Modifies**: `agent-event-envelope` ([`../../../../specs/agent-event-envelope/spec.md`](../../../../specs/agent-event-envelope/spec.md)) — the **invariant-3 row** of the "Explicit non-requirements" invariant table (`spec.md:268`) and nothing else.
> **This is the AG-19 precedent, followed literally.** AG-19 closed invariant 2 by reproducing that table in full, changing exactly one row's "Closed by" cell to `AG-04.1 **+ AG-19.1 — CLOSED**`, spelling out the second closer's contribution in the third column, and appending a parenthetical update to the "AG-04 closes no envelope invariant by itself" sentence. This delta does the same for invariant 3 and deviates in no particular.
> **No requirement in this capability is amended.** `R-AEV-003` was amended by AG-19 and is untouched here; `R-AEV-002`, `R-AEV-004`, `R-AEV-010` and `R-AEV-013` are untouched. AG-20 registers **no** `EventKind`, so the registry-count claims hold with the event surface byte-unchanged (`R-HKS-010` / `S-HKS-024`).
> **Header maintenance obligation at promotion.** `sdd-archive` MUST record **`S-AEV-126`** wherever the target spec enumerates allocated `S-AEV-` identifiers, in the same commit that promotes this delta. It was verified free in this phase: the spec's highest allocated identifier is `S-AEV-125`.
> **Ownership**: the mechanism is owned by [`../agent-hook-taxonomy/spec.md`](../agent-hook-taxonomy/spec.md) (`R-HKS-007`, `R-HKS-008`) and the decoupling standard by [`../agent-event-delivery/spec.md`](../agent-event-delivery/spec.md) (`R-AGE-008`). This delta owns only what the envelope contract's non-requirement table must now say.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-AEV-003` and `S-AEV-125` — AG-19's parent-identity amendment | **Untouched.** AG-20 mirrors no event, adds no constructor and stamps no parent; the walkable-tree reading is unaffected |
| `R-AEV-002` / `S-AEV-010` / `S-AEV-011` — per-lane contiguity | **Confirmed rather than amended.** The observer lane emits **nothing**: it stamps no event, holds no sink and touches no sequence. A run with observers registered produces a byte-identical stream to the same run without them (`R-HKS-007` / `S-HKS-017`) |
| `R-AEV-004` — exactly one run-start first, one run-end last | **Confirmed.** The terminal snapshot runs strictly **after** the run-close was sent and emits no event, so no event follows the terminal on any lane |
| `R-AEV-010` / `R-AEV-013` and the registry-count claims | **Confirmed, byte-unchanged.** AG-20 registers no kind: the stall report is a Go-side typed value delivered off the streaming path, and the reason is structural — an event announcing the stall **is** a path from the stalled observer back onto the producer's stream, exactly what `S-AGE-010` forbids (`R-HKS-008`) |
| Invariant rows 1, 2 and 4 | **Byte-unchanged.** Row 2's AG-19 closure is reproduced verbatim below and is not re-opened, re-worded or re-attributed |

## MODIFIED Explicit non-requirements

The invariant table is reproduced in full; **one row** is back-annotated as closed and none is removed or renumbered.

| Envelope invariant | Closed by | AG-04's part |
| --- | --- | --- |
| 1 — indexed deltas | AG-04.3 **+ AG-05.1** | the construction-surface pin only (`R-AEV-007`) |
| 2 — explicit nesting | AG-04.1 **+ AG-19.1 — CLOSED** | the parent identifier exists; **AG-19.1 delivered the delegation semantics** as a walkable tree in one hop (`R-DEL-004`), with the first production emission of `subagent_started`/`subagent_ended`, and **no** per-event parent stamp (`R-AEV-003` as amended) |
| 3 — non-blocking observers | **AG-01.1 + AG-20.2 — CLOSED** | **none.** No requirement here closes any part of invariant 3. **AG-01.1 decided the decoupling mechanism** (`R-AGE-008`: one canonical internal stream, and per attached consumer an independently owned carrier fed by its own forwarding activity), and **AG-20.2 delivered the mechanical half** the decision demanded of it: three observing hook points exist, a per-run observer lane whose **enqueue is a lock-append that never blocks** dispatches them on the lane's own goroutine, the observers' context is **value-stripped** so the publishing seam is unreachable from an observer, and a deliberately stalled observer is proven — by a goroutine-placement assertion and by stream byte-identity, with **no wall clock anywhere** — to delay no event delivery, and is then reported typed at the run's terminal boundary (`R-HKS-007`, `R-HKS-008`) |
| 4 — typed errors | AG-04.3 **+ AG-11.2** | the typed-failure surface exists; loop-level emission is AG-11's (`R-AEV-008`) |

**AG-04 closes no envelope invariant by itself.** No requirement, scenario, test name or acceptance line may claim otherwise. *(AG-19 update: invariant 2 is now closed **jointly**, exactly as this table always said — AG-04.1 built the field, AG-19.1 gave it meaning. The row must not be re-read as "AG-04 closed it".)* *(AG-20 update: invariant 3 is now closed **jointly** in the same shape — AG-01.1 decided the mechanism and AG-20.2 made it mechanical. The row must not be re-read as "AG-01.1 closed it", and it must not be re-read as AG-04 closing anything: AG-04's cell for this row was, and remains, **none**.)*

(Previously: the invariant-3 row read `**AG-01.1 + AG-20.2 — AG-04 absent**`, recording that the second closer had not yet shipped. AG-20.2 has now shipped it, and leaving the row unchanged would leave the table understating a delivered closure — the same silent staleness the invariant-2 row carried until AG-19.)

#### Scenarios

- **S-AEV-126** — **AG-20: invariant 3's closure is checked against the shipped mechanism, not against this table's prose.** Given the merged AG-20 change, when a reviewer traces every path from a stalled observing hook back toward the canonical producer, then every path terminates inside the observer lane's own drain activity and **none** reaches the producer, the sink, the stamper or any other consumer; when the reviewer looks for how the stall is announced, then it is a Go-side typed value delivered to a nil-defaultable reporter and **no** `EventKind` was registered, with the event registry byte-unchanged at its committed kind count; when the reviewer looks for a wall clock, a timeout, a deadline, a sleep or a join on a stalled observer, then there is **none** in production or in test; and when a run carrying a gated observer is recorded, then its event stream is byte-identical to the same script with no hooks installed and `CheckStream` accepts it unmodified — so a reviewer can confirm the closure without reading this artifact. Cross-referenced to `R-AGE-008` / `S-AGE-010` and to `R-HKS-007` / `R-HKS-008` / `S-HKS-017` / `S-HKS-019`.
