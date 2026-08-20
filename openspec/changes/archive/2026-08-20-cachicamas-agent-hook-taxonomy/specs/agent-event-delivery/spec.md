# Delta for `agent-event-delivery` — AG-20.2 delivers R-AGE-008's MECHANICAL half, and the no-path-back trace is preserved rather than weakened

> **Change**: `cachicamas-agent-hook-taxonomy` · **AG-20** (Layer 2, Wave 5), `0003:1864-1918`
> **Modifies**: `agent-event-delivery` ([`../../../../specs/agent-event-delivery/spec.md`](../../../../specs/agent-event-delivery/spec.md)) — `R-AGE-008` (`spec.md:121-136`) is **MODIFIED in full**, reproduced with both of its scenarios and one AG-20 back-annotation added.
> **BACK-ANNOTATION, not a weakening.** `R-AGE-008` is a requirement **on a decision artifact**: AG-01.1 had to *name* a decoupling mechanism, and it did. AG-20 is the first milestone that could falsify it in code, because until now Layer 2 had no observers. Every clause survives literally, and the requirement's sharpest sentence — *"A statement of the obligation, a convention, a review rule, or documentation alone MUST NOT satisfy this requirement"* (`spec.md:123`) — is the standard AG-20.2 is measured against and meets. Nothing is relaxed.
> **`R-AGE-009` is NOT discharged, and the distinction is the one this delta most needs to protect.** `R-AGE-009` is multi-consumer fan-out with no privileged consumer. **AG-20 makes a *hook* non-blocking; it does not build consumer fan-out.** A reviewer reading "AG-20 closed invariant 3" beside `R-AGE-009` will be tempted to close that too. Closing it would be a false claim.
> **Header maintenance obligation at promotion.** `sdd-archive` MUST record **`S-AGE-030`** wherever the target spec enumerates allocated `S-AGE-` identifiers, in the same commit that promotes this delta. Verified free in this phase: the spec's highest allocated identifier is `S-AGE-029`.
> **Ownership**: the lane, the stall report and the asynchrony proof are owned by [`../agent-hook-taxonomy/spec.md`](../agent-hook-taxonomy/spec.md) (`R-HKS-007`, `R-HKS-008`). This delta owns only the delivery contract's audit statement.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-AGE-009` / `S-AGE-012` — more than one attached consumer, none privileged | **Untouched and explicitly NOT discharged.** AG-20 ships no fan-out machinery, no consumer attachment surface and no second carrier. Its lane serves **hooks**, which are not consumers: it holds no `Event`, no sink and no stamper. The proposal names this deferral in its own out-of-scope table — *"AG-01.1 decided it; nobody has built it"* |
| `R-AGE-004` — the loss rule | **Untouched.** The observer lane's queue is unbounded and drops nothing; it also carries no `Event`, so no loss rule applies to it at all |
| `R-AGE-010` — three nested ownership scopes, each with exactly one closer | **Confirmed.** AG-20 introduces no scope and no closer. The terminal snapshot runs **inside** the run scope, on `Run`'s own goroutine, strictly after the run-close was sent and strictly before the sink is closed by its existing sole closer |
| `R-AGE-017` — no invented channel, one stamping writer | **Confirmed.** The observer lane is not an event channel: nothing on it is an `Event`, nothing on it is stamped, and it reaches no sink |

## MODIFIED Requirements

### R-AGE-008 — A stalled observer is structurally unable to stall the streaming path

The decision MUST make envelope invariant 3 structural. It MUST name a **decoupling mechanism** — one canonical internal stream, and per attached consumer an independently owned carrier fed by its own forwarding activity applying the same send discipline — such that the canonical producer's progress does not depend on any attached consumer's receive progress. A statement of the obligation, a convention, a review rule, or documentation alone MUST NOT satisfy this requirement. The decision MUST tabulate the rejected mechanisms by what each makes **impossible**, including the blocking synchronous multicast that makes nothing impossible, named to show what "conventional, not structural" looks like.

**Back-annotation (AG-20) — the MECHANICAL half has landed, and the requirement's own standard is what it was measured against.** AG-01.1 answered this requirement for *consumers*, at a time when Layer 2 had no observing hooks and the property was therefore unfalsifiable in code. AG-20.2 lands three observing hook points and the discipline that makes them safe. Four mechanisms are recorded here as **shipped structure**, not as reassurance:

- **Enqueue is a lock-append that never blocks.** The run's fire sites append to a per-run lane under a mutex and continue; they never wait on an observer's progress. That, and not scheduling luck, is the non-blocking property, and it is the same *shape* this requirement demanded of consumers — an independently owned carrier fed by its own activity.
- **Dispatch is on the lane's own goroutine.** An observing hook is never invoked on the goroutine that drives the run, stamps events, or delivers them. AG-20 asserts this as a **goroutine-placement** property — the observer captures its own stack and the test asserts the harness run frame and the forwarder frame are **absent** — because a placement property is what asynchrony actually is, and because the alternative shape (holding a gate and waiting for an observable that a synchronous implementation never produces) is unbounded without a clock, which `R-RUN-010` and six shipped NFRs ban.
- **The observers' context is value-stripped, so `S-AGE-010`'s trace is preserved by CONSTRUCTION.** Observers run on a freshly rooted context — deliberately **not** a cancellation-stripped derivation of the run context, because such a derivation preserves context **values**, and in a hosted child run the run context carries the delegation publishing seam retrievable by a plain value lookup (`delegation_seam.go:101-104`). A value-preserving observer context would hand an **observing** hook the one sanctioned door back onto the **parent's** streaming lane, asynchronously, after `Run` returned — a path from a stalled observer to a producer, which is precisely the path this requirement's scenario forbids. Value-stripping closes it structurally rather than by convention.
- **The stall report is a Go-side typed value and NOT an event, for this requirement's own reason.** An event announcing the stall would itself be a path from the stalled observer back onto the producer's stream. The report is delivered to a nil-defaultable reporter, off the streaming path, and **no `EventKind` is registered**. A budget argument for the same conclusion could be overturned by a budget; this one cannot.

**What the back-annotation does NOT claim.** It does not discharge `R-AGE-009`: AG-20 ships no consumer fan-out, no attachment surface and no second carrier. It does not weaken the "documentation alone MUST NOT satisfy" clause — that clause is the bar AG-20.2 was written to clear. And it does not extend the mechanism to *mutating* hooks, which run inline on the caller's goroutine exactly as AG-08 shipped, by design and out of scope.

(Previously: the requirement carried no record of whether the mechanism it demanded had ever been exercised against a real observing hook. `agent-event-envelope/spec.md:268` reserved invariant 3 as closed by "AG-01.1 + AG-20.2", so a reader after the AG-20 merge could not tell from this requirement whether the second half had landed, nor by what mechanism — and would have had to re-derive from the code whether `S-AGE-010`'s trace still held once observers existed.)

#### Scenario: S-AGE-010 — The stalled-observer trace has no path back to the producer

- **GIVEN** the decision's named mechanism, and an attached consumer that stops receiving indefinitely and never cancels
- **WHEN** a reviewer traces every path from that consumer's stalled receive back toward the canonical producer
- **THEN** every path terminates at the forwarding activity that privately owns that consumer's carrier, and none reaches the canonical producer or any other consumer
- **AND** a decision whose only defence is a stated obligation, a convention, or a documented "must not block" rule fails this scenario

*(AG-20 update: the assertion this scenario makes is exactly what it was, and it is about a stalled **consumer**. `S-AGE-030` states the parallel claim for a stalled **hook**, which AG-20 makes possible for the first time; the two are separate and must not be conflated.)*

#### Scenario: S-AGE-011 — Rejected mechanisms are judged by impossibility, not preference

- **GIVEN** the merged decision
- **WHEN** a reviewer reads the mechanism table
- **THEN** each row states what that mechanism makes impossible, AND the blocking synchronous multicast row records that it makes nothing impossible, AND the drop-on-overflow row is rejected against `R-AGE-004`

*(AG-20 update: unchanged. AG-20 adds no mechanism to that table; it implements the one already chosen.)*

#### Scenario: S-AGE-030 — AG-20: the stalled-HOOK trace has no path back to the producer either, and it is checked in code

- **GIVEN** the merged AG-20 change and an observing hook held open indefinitely by the module's test gate primitive
- **WHEN** a reviewer traces every path from that hook's stalled invocation back toward the canonical producer
- **THEN** every path terminates inside the observer lane's own drain activity, and **none** reaches the producer, the sink, the stamper or any other consumer
- **AND** the observer's invocation context carries **no** delegation publishing seam, so the one sanctioned door onto a parent's lane is unreachable from a hook — asserted by a hosted child run whose observer looks the seam up and finds none
- **AND** when the run carrying that gated hook is recorded with a sink buffered to the script's full event count, then its event stream is **byte-identical** to the same script with no hooks installed, modulo the freshly minted run and turn identifiers, and `CheckStream` accepts it unmodified
- **AND** `Run` returns **while the gate is still held**, and the gate is released only afterwards
- **AND** no assertion in this scenario reads elapsed time, sleeps or polls
- **AND** a defence resting on a doc comment, a convention or a review rule fails this scenario exactly as it fails `S-AGE-010`

Cross-referenced to `R-HKS-007` / `S-HKS-017` / `S-HKS-018` and `R-HKS-008` / `S-HKS-019`.
