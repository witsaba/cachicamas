# Delta for `agent-cost-events` — AG-19 answers the parent-aggregation deferral with a permanent NO, and re-scopes the one invariant a mirrored child figure falsifies

> **Change**: `cachicamas-agent-delegation-readiness` · **AG-19** (Layer 2, Wave 5), `0003:1793-1862`
> **Modifies**: `agent-cost-events` ([`../../../../specs/agent-cost-events/spec.md`](../../../../specs/agent-cost-events/spec.md)) — `R-CST-005` (`spec.md:126-144`) and `R-CST-007` (`spec.md:165-175`), plus the non-requirement row at `spec.md:196`.
> **`R-CST-004` is UNAMENDED, and showing it still true is this delta's first obligation.** Its cumulative rule — the sum over every `cost_turn` emitted within the run bracket — survives because AG-19 makes a foreign `cost_turn` **unreachable** on the parent's lane: the seam refuses the `cost_turn` kind in production code (`R-DEL-002` gate 5), not in test discipline. `sdd-verify` MUST open `agent-cost-events:102-119` against the shipped seam; a citation is not evidence.
> **One requirement IS falsified and is amended rather than quietly left standing.** `R-CST-005` says the `cost_session` events within any harness run bracket are zero or more estimates followed by **exactly one** final emitted immediately before the run-close. AG-19 mirrors the child's final `cost_session` onto the parent's stream **mid-bracket** (`R-DEL-007`, and it is stream-legal: run placement, cardinality *any*, `event.go:333-336` with `event_descriptor.go:115-118`). Read over the parent's delivered slice the sentence goes **false**. The fix is a re-scoping by `Event.Run()`, not a relaxation: the run's own protocol is exactly what it was.
> **Header maintenance obligation at promotion.** The target spec's ID line (`spec.md:4`) enumerates allocated `S-CST-` identifiers. `sdd-archive` MUST extend it with **`S-CST-023`** and **`S-CST-024`** in the same commit that promotes this delta. Both were verified free in this phase: the spec's highest allocated identifier is `S-CST-022`. The line states an allocated **range and never a total**, so no count moves.
> **Ownership**: the seam, its refusal and the consumer-side reconstruction are owned by [`../agent-delegation-readiness/spec.md`](../agent-delegation-readiness/spec.md) (`R-DEL-002`, `R-DEL-007`). This delta owns only what the cost contract must now say about a foreign figure on its own stream.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-CST-004` and `S-CST-008` / `S-CST-021` | **Confirmed rather than amended.** The sum is over `cost_turn` events, and no child `cost_turn` can reach the parent's lane. AG-19's bite `S-DEL-020` proves the protection is enforced rather than assumed, by admitting one and watching the parent's cumulative inflate |
| `R-CST-001`'s iff and its per-path table | **Untouched.** AG-19 opens no turn bracket and closes none; it adds no path to that table |
| `R-CST-006`'s three-close rule | **Untouched.** It governs the run's own final figure, which still sits immediately before the run-close on all three closes; `R-DEL-003`'s seam-lifetime bound makes a mirrored event after the parent's turn-close impossible |
| `S-CST-004`'s "no `cost_turn` … carries `Estimate`" | **Untouched.** It is scoped to this capability's own suite, and no `cost_turn` crosses at all |
| `CostFigures`, `CostLabel`, `cost_events.go`, `cost_usage.go` | **Byte-unchanged.** AG-19 adds no field, no label member and no payload type |

## MODIFIED Requirements

### R-CST-005 — The estimate/final protocol is run-scoped: zero or more estimates, then exactly one final at the close

Within any harness run bracket, the `cost_session` events **whose `Run()` is that run's own identity** MUST be **zero or more carrying `CostLabelEstimate`, followed by exactly one carrying `CostLabelFinal` emitted immediately before the run-close event**. This invariant MUST hold on every run outcome (`R-CST-006`).

- A `cost_session` emitted as a running total **because the driver decided to run another logical turn** MUST carry `Estimate`: the run has not concluded and more tokens may follow.
- The run-terminal `cost_session` MUST carry `Final` and MUST correct every earlier estimate — its figures are the cumulative of `R-CST-004` at close time.

**The charter's "any earlier figure was labelled estimate" is a conditional, and the conditional is the requirement.** Where an earlier figure exists it is `Estimate`; in a run that never continued past its first logical turn none exists, and the sole `cost_session` is the `Final` one. A run that emits no `Estimate` therefore satisfies this requirement rather than violating it, and no scenario may assert an `Estimate` is always present.

**The run-identity scoping is AG-19's amendment, and it is a re-scoping rather than a relaxation.** A delivered parent stream may now also carry a `cost_session` belonging to a **child** run: AG-19's seam mirrors the child's final figure onto the parent's lane, where it lands **inside** the parent's hosting turn bracket and therefore not immediately before the parent's run-close. Three facts bound the amendment so it cannot be read as weakening anything:

- **The run's own protocol is untouched.** Filtering by `Run()`, the parent's own `cost_session` sequence is exactly the sequence this requirement always required — zero or more estimates then exactly one final immediately before the run-close. Nothing about the parent's own figures moved.
- **A foreign figure cannot reach the parent's cumulative.** The per-attempt forwarder folds only the `cost_turn` payload (`harness.go:633-635`); a `cost_session` payload never matches it. `R-CST-004` is therefore satisfied without an accumulator change.
- **`Run()` is the discriminator and the label is NOT.** Re-stamping rewrites only the sequence (`sequence.go:54-58`), so a crossed child figure still reports the child's run identity. A consumer summing by label would double-count; a consumer summing by run identity cannot. Any restatement of this requirement that discriminates by label is wrong.

Both kinds are `CardinalityAny`, so multi-emission is already stream-legal, and `cost_session` is `PlacementRun` (`event.go:332-334`), so the between-turns, the immediately-before-run-close **and** the inside-an-open-turn positions are all legal. `CheckStream` MUST accept the recorded stream **unmodified**, `stream_check.go` byte-unchanged.

`cost_session` is **harness-scoped**. A standalone `Turn` invocation — no harness, nil continuation — emits its `cost_turn` with nothing aggregating it and MUST emit **no** `cost_session`. That is correct behavior, stated here so it is not later read as a gap.

(Previously: the sentence quantified over the `cost_session` events on the run bracket without qualification, so a delivered parent stream carrying a mirrored child final figure falsified it — a second `Final` appearing mid-bracket — even though the parent's own protocol was intact.)

#### Scenarios

- **S-CST-009** — **Charter AG-16.1 scenario 3.** Given a run the driver continues past its first logical turn, when the stream is recorded, then at least one `cost_session` carrying `Estimate` appears between two turn brackets; the last `cost_session` **reporting this run's own identity** carries `Final`, is positioned **immediately before** the run-close event, and carries figures equal to the cumulative over the stream's `cost_turn` events; every `cost_session` **of this run** preceding it carries `Estimate` and none carries figures exceeding the final one on any reported figure; and `CheckStream` accepts the stream unmodified. *(AG-19 update: the two qualifiers are the amendment above made falsifiable at the scenario layer. In this scenario's own fixture no delegation occurs, so the filtered and unfiltered readings coincide and the assertion is exactly as strong as it was; the qualifiers exist so the scenario stays true when a delegating fixture is added.)*
- **S-CST-010** — **The conditional half.** Given a run that reaches its terminal on its first logical turn, when the stream is recorded, then its sole `cost_session` carries `Final`, sits immediately before the run-close, and **no** `cost_session` carrying `Estimate` appears anywhere in the run bracket — the conditional reading is satisfied by absence, not by a fabricated estimate.
- **S-CST-011** — **Standalone `Turn` aggregates nothing.** Given `Turn` invoked directly with a zero-value `TurnOptions` and a completing script, when the stream is recorded, then it carries the turn's `cost_turn` before `turn_end` and **no** `cost_session` of either label; and `CheckStream` accepts the stream unmodified.
- **S-CST-022** — **(bite)** RED-first. Given a scratch tree in which the run-terminal `cost_session` is labelled `Estimate`, when `S-CST-009` and `S-CST-010` run, then both **FAIL** reporting a run bracket that closed without a final figure — proving the label assertion reads the terminal event's own label rather than merely counting cost events. RED-recorded BEFORE `S-CST-009` is GREEN, then reverted.
- **S-CST-023** — **AG-19: a mirrored child figure does not disturb the run's own protocol.** Given a parent run whose tool hosts a child run spending non-zero tokens, when the parent's delivered stream is recorded, then it carries the child's final `cost_session` reporting the **child's** run identity inside the hosting turn bracket; filtering to the parent's own run identity yields exactly the sequence `R-CST-005` requires; the parent's own final figure equals the sum over the `cost_turn` events on the parent's stream and is **strictly less than** the parent-plus-child sum; and `CheckStream` accepts the parent's stream unmodified with `stream_check.go` byte-unchanged. Cross-referenced to `R-DEL-007` / `S-DEL-017`.

---

### R-CST-007 — The scope fence: token-only, no new kind, no money, no Layer-2 parent aggregation

This capability MUST NOT introduce any field that could carry money, currency or price, on any payload, at either scope. The token-only pin (`R-APE-004`, mechanically enforced by `S-APE-083`'s forbidden-substring scan, `cost_events_test.go:207-224`) is **strengthened** by this change, never weakened: `CostFigures` gains nothing. Money is Layer 3's, and the price table is out of scope by the charter's own line (`0003:1537`).

This capability MUST NOT register a new `EventKind`. AG-06 minted `cost_turn` and `cost_session`; AG-16 emits them. `event_descriptor.go` and `event_registry_test.go` MUST be byte-unchanged and the every-kind-constructible guard MUST pass at its committed kind count.

**Aggregating a delegated or subagent run's cost into a parent run is ANSWERED by AG-19, and the answer is a permanent NO at Layer 2.** The deferral this clause carried is discharged, not inherited by a later milestone:

- **No Layer-2 fold occurs, on any path.** A child's `cost_turn` is emitted inside the *child's* run bracket, so folding it into a parent figure would falsify `R-CST-004`'s own definition. AG-19 makes the fold **unreachable** rather than discouraged: its publishing seam refuses the `cost_turn` kind in production code (`R-DEL-002` gate 5), and the refusal is typed.
- **The hazard the refusal closes is live, not hypothetical.** The seam's funnel terminates in the parent's per-attempt sink, whose forwarder folds by payload type with **no run-identity filter** (`harness.go:633-635`, re-read this phase). Any child `cost_turn` reaching it would be folded **silently**. AG-19's bite `S-DEL-020` admits one and watches the parent's cumulative inflate, so the protection is asserted rather than assumed.
- **What the parent's stream does carry is the child's session-level figure**, mirrored because it is stream-legal and cannot reach the parent's cumulative. A consumer discriminates it by `Event.Run()` and **never** by label (`R-CST-005` as amended above).
- **The combined figure is a consumer-side reconstruction**, computed by walking both streams and joining them through the single `subagent_started` (`R-DEL-004`). The charter's own qualifier agrees: *"a frontend can show both 'this subagent cost X' and 'the run cost Y'"* is two displayed numbers, not one merged Layer-2 event.

Nothing in this capability may be written as if a Layer-2 parent scope exists, and no later milestone may reopen the fold without amending `R-CST-004` first.

(Previously: the clause deferred parent aggregation to AG-19 without saying what AG-19 would decide, so a reader after the AG-19 merge could not tell whether a Layer-2 fold had shipped, was still coming, or had been ruled out.)

#### Scenarios

- **S-CST-014** — Given the merged change, when the package's payload surface is scanned by `S-APE-083`'s existing forbidden-substring check and its reflection walk, then both pass with `cost_events_test.go` **byte-unchanged** and `CostFigures` **byte-unchanged**; when the every-kind-constructible guard runs, then it passes at its committed kind count with `event_descriptor.go` and `event_registry_test.go` byte-unchanged; and when the change's diff is taken over `backend/agent/src/ai/` and over `go.mod`/`go.sum`, then it is empty.
- **S-CST-024** — **AG-19: the fold is unreachable, not merely unused.** Given the merged change, when a `cost_turn` is published through the delegation seam, then the publish returns the typed inadmissibility sentinel and the event reaches **no** sink; and when the accumulating fold at `harness.go:633-635` is read, then it is byte-unchanged and gained no run-identity filter — because none is needed once no foreign `cost_turn` can arrive. Cross-referenced to `R-DEL-002` / `S-DEL-004`.

## MODIFIED Explicit non-requirements

The list is reproduced only where AG-19 touches it; every other row is unchanged and none is removed.

- **Delegated / subagent cost aggregation into a parent run** — was: *"**AG-19**, by the charter's own Goal line (`0003:1533`). No subagent tool ships in v1 (`0003:1794`)."* **ANSWERED by AG-19, and the row is re-owned rather than deleted so the correction is visible.** A **Layer-2 fold is not deferred — it is ruled out permanently** (`R-CST-007` as amended). What AG-19 shipped instead is a consumer-side reconstruction over two independently validated streams, with the child's session-level figure mirrored onto the parent's stream and discriminated by `Event.Run()`. Still unclaimed by any milestone: a single Layer-2 event carrying a combined parent-plus-child figure, which would require amending `R-CST-004` and which no charter asks for. *(A reader must not take this row as "aggregation shipped"; it did not. Two numbers ship, and the joining is the consumer's.)*
