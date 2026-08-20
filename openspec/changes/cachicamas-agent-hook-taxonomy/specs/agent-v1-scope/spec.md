# Delta for `agent-v1-scope` — AG-20 discharges G11 / R-17 and seam 1's widening, and the CONCRETE HOOKS stay Layer 3's

> **Change**: `cachicamas-agent-hook-taxonomy` · **AG-20** (Layer 2, Wave 5), `0003:1864-1918`
> **Modifies**: `agent-v1-scope` ([`../../../../specs/agent-v1-scope/spec.md`](../../../../specs/agent-v1-scope/spec.md)) — **by APPENDING one requirement**, `R-AGS-016`, alongside `R-AGS-007` (`spec.md:148-183`), `R-AGS-012` (`spec.md:243-251`) and `R-AGS-013` (`spec.md:255-267`). **All three are byte-unchanged.**
> **BACK-ANNOTATION ONLY. No verdict is renumbered, removed or re-owned, no seam moves, and NO COUNT MOVES** — `R-AGS-014`'s count-consistency rule is satisfied by there being nothing to update, because AG-20 adds no verdict, no seam and no row.
> **Why an ADDED requirement rather than three MODIFIED blocks.** None of the three is falsified. All three are large and already carry AG-15, AG-17, AG-18 and AG-19 back-annotations; a `MODIFIED` block must reproduce its requirement in full or the archive step silently drops what it omits, and re-transcribing three such requirements to add one paragraph each would risk that three times for no change in substance. The `R-LSK-007` precedent (AG-19) is exact. `R-AGS-016` is the next free requirement identifier (`R-AGS-001`…`R-AGS-015` are allocated); `S-AGS-065` and `S-AGS-066` are used below.
> **A promotion-integrity finding, recorded rather than worked around.** `S-AGS-064` — allocated by AG-19's `agent-v1-scope` delta under `R-AGS-013` — **does not appear in the promoted spec**, though `S-AGS-061`, `S-AGS-062` and `S-AGS-063` do. Either the AG-19 promotion dropped it or it was never merged. This delta therefore starts at **`S-AGS-065`**, leaving `S-AGS-064` unreused so the append-only rule is not violated whichever way the discrepancy resolves. `sdd-archive` MUST record this discrepancy when promoting, and MUST NOT silently reuse `S-AGS-064`.
> **This delta carries the same anti-over-claim obligation AG-19's did.** AG-20 discharges **G11 / R-17** and widens **seam 1**. It does **not** ship any concrete hook, and `S-AGS-048` maps *"G11's concrete hooks"* to **doc 0004 CO-24.1 / CO-24.2** — Layer 3's nodes, not AG-20's. A reviewer reading "AG-20 shipped" beside G11 will be tempted to close the Layer 3 half. **Closing it would be a false claim.**
> **Ownership**: the taxonomy is owned by [`../agent-hook-taxonomy/spec.md`](../agent-hook-taxonomy/spec.md). This delta owns only the audit statement.

## Not modified, and why

| Not modified | Why |
|---|---|
| Every count in the artifact | **Unchanged.** AG-20 adds no verdict, no seam and no row; `R-AGS-014`'s consistency rule has nothing to update |
| `S-AGS-023`'s seam mapping — seam 1 to AG-08.1 | **Confirmed by delivery rather than amended.** The mapping is exactly what it was. AG-20 **widens** what the seam carries; it does not re-own the seam or move it to another node |
| `S-AGS-048`'s orphan-check row — G11's concrete hooks to CO-24.1 / CO-24.2 | **Untouched and still Layer 3's.** AG-20 ships the taxonomy, never a concrete hook. The charter's own out-of-scope line is verbatim: *"Any concrete hook implementation (Layer 3 wires them)"* (`0003:1874`) |
| `R-AGS-006`'s `AGS-D` deferred entries — the subagent tool, the failover implementation, compaction quality | **All three untouched and still deferred.** AG-20 touches none of those three surfaces |
| `S-AGS-052` — AG-09, AG-15, AG-16 and AG-20 each carry a pointer to their governing verdict | **Confirmed by delivery.** AG-20's pointer is unchanged; the assertion this scenario makes is that a pointer exists, which is exactly what it was |
| `R-AGS-015` — Layer 2 does not decide Layer 3's content | **Confirmed and load-bearing.** It is the reason AG-20 ships **no** "slow hook" threshold: a threshold is a Layer 3 policy number |

## ADDED Requirements

### R-AGS-016 — AG-20 discharges G11 / R-17 and widens seam 1; the concrete hooks stay Layer 3's and the two MUST NOT be conflated

**G11 / R-17 is DISCHARGED for its Layer 2 half, and what discharged it is recorded as a mechanism rather than as a reassurance.** The forward pass maps **R-17 to G11** (`S-AGS-030`), and doc 0003's AG-20 charter closes it. What AG-20 shipped:

- **The full taxonomy behind one registration surface.** Four hook points — pre-request (AG-08's, now composed), pre-compact, post-turn and session-start — registered on **one** exported value type held as a **field** on the harness, transported to the turn loop on the harness's own per-turn copy. A registration **method** was unavailable by fence and none was added.
- **A uniform mutate-versus-observe contract enforced BY TYPE, not by documentation.** The two mutating families take a payload and return one with an error; the two observing families **have no result parameters at all**, so a hook that could signal a mutation or a failure is unconstructible. Every payload is a value type with unexported fields and read-only accessors.
- **The asynchrony discipline made mechanical**, which is the charter's own acceptance clause and the standard `R-AGE-008` sets: a per-run observer lane whose enqueue never blocks, dispatch on the lane's own goroutine asserted as a **goroutine-placement** property, a value-stripped observer context that makes the delegation publishing seam unreachable from a hook, a stalled observer proven to delay no event delivery by stream byte-identity, and **no wall clock anywhere** in production or test.
- **Typed, source-named failure on both mutating points**, attributed by source name rather than by a chain-wide ordinal, so the attribution survives a later insertion.

**SEAM 1 IS WIDENED, NOT RE-OWNED.** `S-AGS-023` maps seam 1 to **AG-08.1**, and that mapping is unchanged. AG-08 shipped the seam as a single callable and recorded, in its own spec, that *"AG-20 widens to chain composition"* (`agent-pre-request-hook/spec.md:19`). AG-20 discharges that promise **additively**: the shipped field is kept, runs **first**, and feeds the chain's element 0. Its removal is **AG-23**'s and AG-20 does not take it.

**What is NOT discharged, and this is the clause most at risk of being wrongly closed.** `S-AGS-048` records *"G11's concrete hooks to CO-24.1 / CO-24.2"* — doc 0004's Layer 3 nodes. **AG-20 ships zero concrete hooks.** Cache-breakpoint placement, compaction policy and telemetry are Layer 3's, verbatim from the charter, and this milestone's own out-of-scope table names them. **A taxonomy is not a wiring.** The G11 row's Layer 3 half stays open with CO-24.1 / CO-24.2 named as its holders, on exactly the AG-15 argument this artifact already records: *a shipped seam with a declining implementation is a held place, not a delivered capability*.

**Two further absences MUST be stated so they are inherited knowingly rather than discovered:**

1. **No wall-clock timeout, deadline or "slow hook" threshold ships, and none ever will in Layer 2.** `R-RUN-010` forbids the temporal answer on a structurally similar seam, and `R-AGS-015` forbids Layer 2 deciding Layer 3's content — a threshold is Layer 3 policy. AG-20's "eventually" is the run's **terminal boundary**, a structural moment, not a duration.
2. **`R-AGE-009`'s multi-consumer fan-out is NOT shipped.** AG-20 makes a *hook* non-blocking; it does not build consumer attachment machinery. Envelope invariant 3 closes jointly with AG-01.1; `R-AGE-009` remains decided-and-unbuilt.

**Wave 5's exit is a scheduling fact, not a scope claim.** AG-20 is Wave 5's last milestone, and AG-21 inherits this milestone's frozen postures by name: the caller-owned stalled-observer goroutine leak, the release-before-baseline test rule, unrecovered mutating hooks, and cross-run hook state.

#### Scenarios

- **S-AGS-065** — **AG-20: the discharge is auditable, not asserted.** Given the AG-20 statement above, when a reviewer opens each mechanism it names, then the one registration surface, the two type families, the pre-request chain, the pre-compact splice and its re-resolution, the post-turn firing enumeration, the session-start latch, the registration-order dispatch, the observer lane's non-blocking enqueue, the terminal-boundary snapshot and the source-named failure attribution each exist as a requirement in `agent-hook-taxonomy` with at least one independently verifiable scenario; and when the reviewer opens doc 0003's Traceability-spine row for **R-17**, then it names the same discharging node the verdict names, with no node moved and no count changed.
- **S-AGS-066** — **AG-20: the Layer 3 half is checked against the shipped surface, not against this artifact's prose.** Given the merged AG-20 change, when the package's exported surface is enumerated from an external test package, then it declares **no** concrete hook, no cache-breakpoint placement, no compaction policy and no telemetry sink; no hook is registered by default and a zero-value registration value is inert on every path; and when the reviewer looks for a wall clock, then no timeout, deadline, sleep or "slow hook" threshold exists in production or in test. And when the reviewer reads `S-AGS-048`'s orphan-check row for G11's concrete hooks, then it still names CO-24.1 / CO-24.2 as their holders — a discharge that also closed those would be over-claiming, on exactly the AG-15 and AG-19 argument this artifact already records.
