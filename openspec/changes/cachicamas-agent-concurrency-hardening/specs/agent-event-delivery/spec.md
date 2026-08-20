# Delta — `agent-event-delivery` (AG-21)

> **Change**: `cachicamas-agent-concurrency-hardening` · **AG-21** · Target: `openspec/specs/agent-event-delivery/spec.md`
> **Op**: MODIFIED `R-AGE-008` (back-annotation) with one ADDED scenario `S-AGE-031` inside its block.
> **Decision**: proposal D3, binding. AG-21 **proves invariant 3 under pressure**; it does **not** close it. `agent-event-envelope/spec.md:269` already records invariant 3 as `AG-01.1 + AG-20.2 — CLOSED`, and doc 0003's forward *"Requirements → closing nodes"* row for R-05 (`0003:2204`) names `AG-01.1, AG-20.2` and not AG-21. `0003:2265` is the **reverse** *"Nodes trace back to scope"* table; tracing to a requirement is not closing it, and that row is left as it stands.
> **`R-AGE-008`'s own bar is respected**: the requirement forbids satisfaction by prose (`agent-event-delivery/spec.md:123`). **The ADDED scenario is AG-21's contribution; the paragraph only records it.**

## Header maintenance obligation at promotion

`sdd-archive` MUST, in the target spec's header block, extend the scenario-ID line (`agent-event-delivery/spec.md:6`) so that **`S-AGE-031`** is covered. That line currently reads only the convention `S-AGE-0NN` and states no range or total; if a range is introduced it MUST be stated as a **range and never as a total** (`S-LSK-020`). No other header line changes.

## Not modified, and why

| Element | Verdict |
|---|---|
| `R-AGE-004`, `R-AGE-005`, `R-AGE-006`, `R-AGE-007` | **Byte-unchanged.** AG-21 measures against them; it amends none. `R-AGE-006`'s committed-fact rule is the standard `R-CNH-003` is checked by |
| `R-AGE-009` (multi-consumer fan-out) | **Byte-unchanged and still decided-and-unbuilt.** AG-21 attaches no second consumer and builds no fan-out |
| `S-AGE-010`, `S-AGE-011`, `S-AGE-030` | **Reproduced verbatim below, unchanged in claim.** `S-AGE-030` is AG-20's stalled-**hook** claim; `S-AGE-010` is the stalled-**consumer** claim; `agent-event-delivery/spec.md:143` states they *"must not be conflated"*, and `S-AGE-031` is a **third** member of that family, not a restatement of either |
| The AG-20 back-annotation block and its `(Previously: …)` paragraph | **Reproduced verbatim.** AG-21 appends its own block after AG-20's and removes nothing |

## MODIFIED Requirements

### R-AGE-008 — A stalled observer is structurally unable to stall the streaming path

The decision MUST make envelope invariant 3 structural. It MUST name a **decoupling mechanism** — one canonical internal stream, and per attached consumer an independently owned carrier fed by its own forwarding activity applying the same send discipline — such that the canonical producer's progress does not depend on any attached consumer's receive progress. A statement of the obligation, a convention, a review rule, or documentation alone MUST NOT satisfy this requirement. The decision MUST tabulate the rejected mechanisms by what each makes **impossible**, including the blocking synchronous multicast that makes nothing impossible, named to show what "conventional, not structural" looks like.

**Back-annotation (AG-20) — the MECHANICAL half has landed, and the requirement's own standard is what it was measured against.** AG-01.1 answered this requirement for *consumers*, at a time when Layer 2 had no observing hooks and the property was therefore unfalsifiable in code. AG-20.2 lands three observing hook points and the discipline that makes them safe. Four mechanisms are recorded here as **shipped structure**, not as reassurance:

- **Enqueue is a lock-append that never blocks.** The run's fire sites append to a per-run lane under a mutex and continue; they never wait on an observer's progress. That, and not scheduling luck, is the non-blocking property, and it is the same *shape* this requirement demanded of consumers — an independently owned carrier fed by its own activity.
- **Dispatch is on the lane's own goroutine.** An observing hook is never invoked on the goroutine that drives the run, stamps events, or delivers them. AG-20 asserts this as a **goroutine-placement** property — the observer captures its own stack and the test asserts the harness run frame and the forwarder frame are **absent** — because a placement property is what asynchrony actually is, and because the alternative shape (holding a gate and waiting for an observable that a synchronous implementation never produces) is unbounded without a clock, which `R-RUN-010` and six shipped NFRs ban.
- **The observers' context is value-stripped, so `S-AGE-010`'s trace is preserved by CONSTRUCTION.** Observers run on a freshly rooted context — deliberately **not** a cancellation-stripped derivation of the run context, because such a derivation preserves context **values**, and in a hosted child run the run context carries the delegation publishing seam retrievable by a plain value lookup (`delegation_seam.go:101-104`). A value-preserving observer context would hand an **observing** hook the one sanctioned door back onto the **parent's** streaming lane, asynchronously, after `Run` returned — a path from a stalled observer to a producer, which is precisely the path this requirement's scenario forbids. Value-stripping closes it structurally rather than by convention.
- **The stall report is a Go-side typed value and NOT an event, for this requirement's own reason.** An event announcing the stall would itself be a path from the stalled observer back onto the producer's stream. The report is delivered to a nil-defaultable reporter, off the streaming path, and **no `EventKind` is registered**. A budget argument for the same conclusion could be overturned by a budget; this one cannot.

**What the back-annotation does NOT claim.** It does not discharge `R-AGE-009`: AG-20 ships no consumer fan-out, no attachment surface and no second carrier. It does not weaken the "documentation alone MUST NOT satisfy" clause — that clause is the bar AG-20.2 was written to clear. And it does not extend the mechanism to *mutating* hooks, which run inline on the caller's goroutine exactly as AG-08 shipped, by design and out of scope.

**Back-annotation (AG-21) — the mechanism is now exercised UNDER PRESSURE and IN COMBINATION, and this paragraph closes nothing.** Two things are recorded, and the boundary between them is the point:

- **What AG-21 adds is falsification, not mechanism.** AG-21 ships no production code by default (`R-CNH-008`). It stalls a real consumer **structurally** — an unbuffered sink with no receiver, so the producer is genuinely blocked at its unconditional send rather than merely slow — while the same run is simultaneously suspended, steering, compacting or hosting a child, and then signals it. That combination is what *"under pressure"* means here, and it is the first time the decoupling mechanism is observed against anything other than a single-feature fixture.
- **What this paragraph explicitly does NOT claim.** It does **not** close envelope invariant 3: `agent-event-envelope/spec.md:269` records that as `AG-01.1 + AG-20.2 — CLOSED`, and AG-21 cannot close what is already closed. It does **not** discharge `R-AGE-009`, which remains decided-and-unbuilt. It does **not** satisfy this requirement by prose — the requirement's own clause above forbids exactly that, and **`S-AGE-031` below is the discharge; this paragraph merely records where to find it.** And it does **not** merge the consumer claim with the hook claim: `S-AGE-010`, `S-AGE-030` and `S-AGE-031` are three separate members of one family, and conflating any two of them is the error `agent-event-delivery/spec.md:143` already names.

(Previously: the requirement carried no record of whether the mechanism it demanded had ever been exercised against a real observing hook. `agent-event-envelope/spec.md:268` reserved invariant 3 as closed by "AG-01.1 + AG-20.2", so a reader after the AG-20 merge could not tell from this requirement whether the second half had landed, nor by what mechanism — and would have had to re-derive from the code whether `S-AGE-010`'s trace still held once observers existed.)

(Previously, at AG-21: the requirement recorded the mechanism as shipped and asserted against single-feature fixtures only. A reader could not tell from it whether the decoupling had ever been observed while the run was simultaneously under a second kind of pressure — a pending suspension, a queued steer, an in-flight compaction or a live child run — nor whether a structurally stalled consumer, as opposed to a merely slow one, had ever been driven at all. Both were unasserted, and doc 0003's reverse table `0003:2265` traced AG-21 to R-05 with nothing in this capability recording what that trace was owed.)

#### Scenario: S-AGE-010 — The stalled-observer trace has no path back to the producer

- **GIVEN** the decision's named mechanism, and an attached consumer that stops receiving indefinitely and never cancels
- **WHEN** a reviewer traces every path from that consumer's stalled receive back toward the canonical producer
- **THEN** every path terminates at the forwarding activity that privately owns that consumer's carrier, and none reaches the canonical producer or any other consumer
- **AND** a decision whose only defence is a stated obligation, a convention, or a documented "must not block" rule fails this scenario

*(AG-20 update: the assertion this scenario makes is exactly what it was, and it is about a stalled **consumer**. `S-AGE-030` states the parallel claim for a stalled **hook**, which AG-20 makes possible for the first time; the two are separate and must not be conflated.)*

*(AG-21 update: unchanged in claim. `S-AGE-031` states the same family's claim **under pressure and in combination**; it is a third member, not a restatement, and the three must not be conflated.)*

#### Scenario: S-AGE-011 — Rejected mechanisms are judged by impossibility, not preference

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the mechanism table
- **THEN** each row states what that mechanism makes impossible, AND the blocking synchronous multicast row records that it makes nothing impossible, AND the drop-on-overflow row is rejected against `R-AGE-004`

*(AG-20 update: unchanged. AG-20 adds no mechanism to that table; it implements the one already chosen.)*

*(AG-21 update: unchanged. AG-21 adds no mechanism to that table either; it stresses the one already chosen.)*

#### Scenario: S-AGE-030 — AG-20: the stalled-HOOK trace has no path back to the producer either, and it is checked in code

- **GIVEN** the merged AG-20 change and an observing hook held open indefinitely by the module's test gate primitive
- **WHEN** a reviewer traces every path from that hook's stalled invocation back toward the canonical producer
- **THEN** every path terminates inside the observer lane's own drain activity, and **none** reaches the producer, the sink, the stamper or any other consumer
- **AND** the observer's invocation context carries **no** delegation publishing seam, so the one sanctioned door onto a parent's lane is unreachable from a hook — asserted by a hosted child run whose observer looks the seam up and finds none
- **AND** when the run carrying that gated hook is recorded with a sink buffered to the script's full event count, then its event stream is **byte-identical** to the same script with no hooks installed, modulo the freshly minted run and turn identifiers, and `CheckStream` accepts it unmodified
- **AND** `Run` returns **while the gate is still held**, and the gate is released only afterwards
- **AND** no assertion in this scenario reads elapsed time
- **AND** a defence resting on a doc comment, a convention or a review rule fails this scenario exactly as it fails `S-AGE-010`

Cross-referenced to `R-HKS-007` / `S-HKS-017` / `S-HKS-018` and `R-HKS-008` / `S-HKS-019`.

#### Scenario: S-AGE-031 — AG-21: the decoupling holds with a STRUCTURALLY stalled consumer while the run is simultaneously under a second pressure

- **GIVEN** a run whose consumer sink is **unbuffered** and whose consumer has read a prefix and then stopped receiving — so the producer is blocked at its unconditional send **by construction**, with no receiver at all, rather than by any timing arrangement
- **AND** that same run is simultaneously in one of the four combined states of `R-CNH-001` — a suspension pending, a steer queued and undelivered, a compaction call in flight, or a child harness active — with the state proven pending by a happens-before edge the production code itself provides, never by elapsed time
- **WHEN** the consumer resumes receiving, and separately when the run is signalled while still stalled
- **THEN** every event describing a fact already committed to the transcript is present on the stream once the consumer drains to completion, checked against the scripted event identity set — count, kinds and call identities — so that a single missing committed-fact event is a divergence (`R-AGE-006`)
- **AND** on the never-cancelled arm the number of events absent is **zero**: `R-AGE-005`'s sanctioned loss fires only on cancellation with a saturated buffer, so on that arm it is unreachable and any absence is unsanctioned and fails this scenario
- **AND** `CheckStream` accepts the drained stream **unmodified**, with `stream_check.go` byte-unchanged
- **AND** on the signalled arm the run **returns**, observed by a read on its completion channel and never by a wall-clock assertion, and the run-end outcome and returned error match the firing signal
- **AND** no assertion in this scenario reads elapsed time, sleeps or polls
- **AND** this scenario asserts nothing about a stalled **hook** — that is `S-AGE-030`'s — and nothing about closing envelope invariant 3, which `AG-01.1 + AG-20.2` already closed
- **AND** a defence resting on the back-annotation paragraph above, on a doc comment, or on a convention fails this scenario exactly as it fails `S-AGE-010`

Discharged directly by `TestCombinedPressure_StalledSteering_NeverCancelled_LosesNothing` and `TestCombinedPressure_StalledSteering_Interrupted` (`slow_consumer_pressure_test.go`), which drive R-CNH-001's steering-queued state and R-CNH-003's structural stall on the SAME run — the literal conjunction this scenario's GIVEN clauses require (sdd-verify round-2, MAJOR-1: the standalone `S-CNH-007`/`S-CNH-008` evidence alone does not exercise it). Cross-referenced to `R-CNH-003` / `S-CNH-007` / `S-CNH-008` and `R-CNH-001` for the single-feature halves.
