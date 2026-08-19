# Delta for `agent-event-delivery` — AG-19 exercises the delivery decision WITHOUT amending it, and discharges the routing obligation it pre-stated

> **Change**: `cachicamas-agent-delegation-readiness` · **AG-19** (Layer 2, Wave 5), `0003:1793-1862`
> **Modifies**: `agent-event-delivery` ([`../../../../specs/agent-event-delivery/spec.md`](../../../../specs/agent-event-delivery/spec.md)) — `R-AGE-010` (`spec.md:149-163`), `R-AGE-016` (`spec.md:229-237`) and `R-AGE-017` (`spec.md:239-248`).
> **BACK-ANNOTATION ONLY. Not one requirement is amended in substance**, no scenario's falsifiable claim moves, no number changes, and `R-AGE-017` in particular is **exercised unamended** — that is the delta's whole point. Each touched requirement is reproduced in full and gains one AG-19 paragraph.
> **Why this delta is not optional.** This capability wrote three obligations *for* AG-19 before AG-19 existed: a per-delegated-run ownership scope (`R-AGE-010`, `S-AGE-014`), a nested routing obligation stated rather than left to be invented (`R-AGE-016`, `S-AGE-022`), and an inheritance row forbidding an invented channel (`R-AGE-017`). AG-19 is where all three come due. Leaving them unannotated after the merge is this repository's un-back-annotated-merge staleness shape, against which **no Go test defends**. `sdd-verify` MUST open each line cited here.
> **The `R-AGE-017` question answered head-on, because "we reused the funnel" is a claim and not evidence.** `R-AGE-017` forbids a downstream milestone inventing *its own channel*, *its own loss rule*, or *its own way back into a live run*. AG-19 adds a **named exported accessor onto the channel this decision already owns** — the scheduler's existing emission funnel, whose single stamping dispatcher every concurrent sibling tool call already uses. It adds **no** second channel, **no** buffer, **no** second stamping writer and **no** loss rule of its own. Whether a named door onto an existing channel is "a way back into a live run" is exactly the judgement `R-AGE-017` says must be **stated rather than assumed** — so it is stated here, in the open, rather than settled by silence.
> **Ownership**: the seam and its admissibility rule are owned by [`../agent-delegation-readiness/spec.md`](../agent-delegation-readiness/spec.md) (`R-DEL-001`…`R-DEL-008`). This delta owns only the delivery decision's audit statement.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-AGE-011`'s exactly-one-terminal discipline | **Confirmed rather than amended.** The seam's admissibility rule refuses every bracket-opening, bracket-closing and terminal kind (`R-DEL-002` gates 2 and 4), so no mirrored event can be a second run-start or arrive after a terminal |
| Every buffer, capacity and loss-posture requirement | **Untouched.** AG-19 adds no channel, so it adds no capacity and no loss rule |
| `R-AGE-018`'s deferred-number rule | **Untouched.** AG-19 names no numeric capacity |
| The sink-ownership rule (exactly one close per sink, by exactly one owner) | **Confirmed.** Each child harness owns and closes its own sink; the parent never closes the child's and the child never closes the parent's |

## MODIFIED Requirements

### R-AGE-010 — Three nested ownership scopes, each with exactly one closer

The decision MUST state, for each of the per-turn, per-run and per-delegated-run scopes, its sole owner and sole closer: the loop for the turn scope, the harness for the run scope, and the child harness for a delegated run with the parent separately owning the subagent bracket on its own stream. It MUST state that ownership is never shared, that nesting is strict and sequential, and that nothing else closes — not a consumer, not a test helper, not a party above the layer.

**Back-annotation (AG-19) — the per-delegated-run scope now exists in fact, and the rule it was written for held without amendment.** AG-04 wrote this scope before any delegated run could exist. AG-19 creates one, and every clause survives literally:

- **The child harness is the sole closer of the child's stream.** The child's sink closes only after the child's run-close was sent (`harness.go:446`), and the parent's tool never closes it.
- **The parent separately owns the subagent bracket on its own stream.** `subagent_started` and `subagent_ended` are parent-lane events, stamped by the parent's dispatcher, and the child never emits them.
- **Nesting is strict and sequential, and the ordering is structural rather than asserted.** The tool publishes `subagent_ended` only after the child's sink closed, so on the parent's lane `index(subagent_ended) < index(parent run-close)` — which is exactly `S-AGE-014`'s "the child's stream fully closes before the parent's representation of it closes", now checkable by a test instead of by a reviewer.
- **Ownership is never shared even with N children.** Sibling tool calls each host a **distinct** `Harness` value with its own context, transcript, stamper and sink; concurrent `Run` calls on one `Harness` value remain out of scope.

#### Scenario: S-AGE-013 — Every scope has exactly one closer

- **GIVEN** the merged decision
- **WHEN** a reviewer enumerates the closers per scope
- **THEN** each of the three scopes has exactly one, AND the turn-scope closer differs from the run-scope closer with the reason stated (the loop is re-instantiated per turn and does not know the run boundary)

#### Scenario: S-AGE-014 — Delegation does not break the ownership rule

- **GIVEN** a delegated run cancelled leaf-first
- **WHEN** a reviewer traces the close order under the decision's rules
- **THEN** the child's stream fully closes before the parent's representation of it closes, AND the parent never closes the child's stream, AND the child never closes the parent's

*(AG-19 update: this scenario's claim is unchanged, and it is now discharged by execution rather than by review — `S-DEL-014` runs exactly this shape and asserts the close order by observed event index on the parent's lane, never by elapsed time.)*

#### Scenario: S-AGE-028 — AG-19: the delegated scope is exercised, and the closers are counted in a running system

- **GIVEN** a parent run whose tool hosts a child harness, and separately two sibling tools each hosting their own child
- **WHEN** the runs complete and every stream is captured
- **THEN** each child's sink is closed exactly once, by that child's own harness, and neither the parent nor a sibling closes it
- **AND** `subagent_started` and `subagent_ended` appear only on the parent's lane, and no child emits either
- **AND** on the parent's lane the index of each `subagent_ended` is strictly less than the index of the parent's run-close
- **AND** the parent's stream and each child's stream are `CheckStream`-valid **validated separately**, never concatenated

---

### R-AGE-016 — The upward path recurses under delegation

The decision MUST state that a child harness has its own inbound surface, that what a child's policy scope would ask about is asked on the **parent's** stream, and that the frontend therefore answers through the parent's surface while the parent's routing must reach the nested child's own suspension lookup.

**Back-annotation (AG-19) — `S-AGE-022`'s routing obligation is DISCHARGED, and it is discharged with ZERO new production routing surface.** The obligation this requirement stated was that the routing be *stated rather than left to the delegation milestone to invent*. AG-19 is that milestone, and it invented nothing:

- **Ask up.** The child's `permission_decision_required` is emitted on the child's own stream by the child's own scheduler and is **mirrored** onto the parent's stream through the publishing seam. Its answering `permission_decision_made` crosses **with it**: a mirrored ask with no mirrored answer is unreadable to the human watching, so the pair crosses together or not at all (`R-DEL-002`, `R-DEL-008`).
- **Decision down.** The human's verdict, given on the parent's surface, reaches the child's suspension through the **existing** wake surface (`scheduler.go:264-272`), on the child `Scheduler` value the delegating tool already owns. The gate re-enters resolution on wake (`scheduler.go:772-783`) and the child's derived scope answers with that verdict. **No new routing type, method, channel or registry ships.**
- **What "the parent's policy allowed flows down" operationally means, since the phrase admits a weaker reading.** The child's scope is an ordinary composition of the parent's policy; it may only **narrow** and MUST NOT **widen**, and AG-19 asserts **both** directions rather than only the convenient one (`S-DEL-018`). No scope type, rule set or mode flag enters Layer 2.
- **What is not carried up.** `permission_resolution_remembered` is `CardinalityAtMostOne`, so it is refused at the seam and stays on the child's own stream. A remembered rule remains scoped to one `Schedule` call, exactly as `R-APP-010`'s deferral already says.

#### Scenario: S-AGE-022 — A decision for a nested call reaches the child

- **GIVEN** a call suspended inside a delegated run
- **WHEN** a reviewer traces where the frontend sends the decision and where it must arrive
- **THEN** it is sent to the parent's surface and arrives at the child's suspension, AND the routing obligation is stated rather than left to the delegation milestone to invent

*(AG-19 update: the obligation is now discharged by execution — `S-DEL-019` suspends a child call, observes the ask and its answer on the parent's stream, answers through the existing wake surface and asserts the child's call resumes with that verdict. This scenario's own claim is unchanged; the annotation adds discharge evidence.)*

---

### R-AGE-017 — Downstream milestones consume this surface rather than inventing their own

The decision MUST close with an inheritance table stating, in each milestone's own terms, what AG-04, AG-10, AG-13, AG-14, AG-19 and AG-20 take from it. It MUST state that none of them may invent its own channel, its own loss rule, or its own way back into a live run; a downstream milestone that finds itself deciding one of these properties is proposing an amendment to this decision, not exercising a judgement call.

**Back-annotation (AG-19) — AG-19's row is EXERCISED UNAMENDED, and the borderline case is adjudicated in the open rather than assumed away.** AG-19 is the first milestone whose whole subject matter is "a way back into a live run", so the row deserves an explicit verdict rather than a reassurance:

- **No channel is invented.** AG-19 reuses the scheduler's existing emission funnel verbatim — the same funnel, the same single stamping dispatcher goroutine, the same sink — that concurrent sibling tool calls already use. It creates no channel, declares no buffer capacity and names no numeric value, so `R-AGE-018` is untouched too.
- **No loss rule is invented.** A mirrored event either reaches the funnel or is refused **typed**, by one of exactly two sentinels: inadmissible (the kind can never cross) or revoked (the hosting call has completed or detached). There is no third outcome, no silent drop and no panic (`R-DEL-002`, `R-DEL-003`).
- **No second stamping writer is introduced.** Each lane keeps exactly one stamping writer: the parent's stamper is touched only by the parent's dispatcher, the child's only by the child's own. Re-stamping discards the prior sequence and returns a copy, so both lanes stay contiguous and are **never merged**.
- **The borderline judgement, stated because this requirement demands it be stated.** What AG-19 adds is a **named exported accessor** onto the channel this decision already owns, reachable only for the duration of one tool call and revoked on every exit path. Read as "a new door", it is new; read as "a new way", it is the existing way given a name and a lifetime. **This delta records the reading as a judgement made in the open, not as an amendment made by silence.** A later milestone that wants a *second* funnel, a buffer, an unrevoked handle, or a publish path that survives its hosting call is proposing an amendment to this decision — and this paragraph is what it will be measured against.

#### Scenario: S-AGE-023 — Every blocked milestone has a row

- **GIVEN** the merged decision
- **WHEN** a reviewer checks the inheritance table against AG-04, AG-10, AG-13, AG-14, AG-19 and AG-20
- **THEN** each has a row stating what it inherits, AND the no-invented-channel rule is stated once for all of them
- **AND** the acceptance criterion is checkable by reading that one table

*(AG-19 update: the table's row count and its milestone list are unchanged — AG-19 was always in it. What changed is that AG-19's row is now discharged. AG-20 remains outstanding.)*

#### Scenario: S-AGE-029 — AG-19: the no-invented-channel rule is checked against code, not against a claim

- **GIVEN** the merged AG-19 change
- **WHEN** its production diff is read
- **THEN** it declares no channel type, no channel creation, no buffer capacity and no numeric capacity anywhere
- **AND** the seam's every refusal path returns one of exactly two sentinels distinguishable by `errors.Is`, with no path that drops an event silently and no path that panics
- **AND** each lane's stamper has exactly one writing goroutine, proven under `-race` with two sibling children running concurrently
