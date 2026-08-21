# Delta — `agent-run-driver` (AG-23)

> **Change**: `cachicamas-agent-layer3-handoff` · **AG-23** · Target: `openspec/specs/agent-run-driver/spec.md`
> **Ops**: **ADDED** `R-RUN-014` — the per-attempt forwarder's exit contract on the **panic-unwind** path. This is `ADDED`, not `MODIFIED`, and the distinction is deliberate: on every non-panicking path the forwarder's observable behaviour is **unchanged**, so `R-RUN-002`'s forwarding clause, `R-RUN-003`'s bracket and lane rules and `R-RUN-013`'s two pure reads are all reproduced by nothing and amended by nothing. What AG-23 adds is a guarantee on a path that previously carried **no** guarantee at all.
> **Decision**: design **AD-1** (which supersedes proposal `D-5`'s open shape), binding.

## The defect, verified on this branch

`Run` registers its consumer-sink close **first** in its own defer stack, so LIFO runs it **last** during
unwind. The per-attempt forwarder goroutine is joined only after the one-turn surface **returns normally**.
The one-turn surface can panic without recovery — that is a documented, deliberately unrecovered v1 seam —
and on that path the join is never reached: the sink close then runs concurrently with a forwarder parked
in an unbuffered send. That is a **send on a closed channel**: an unrecovered crash in a goroutine no
consumer owns, reachable through a seam this milestone freezes.

**This is a defect, not a limitation.** A crash-class defect reachable through a frozen seam cannot be
entered in the known-limitations register (`agent-layer3-handoff`'s `R-L3H-008`), and AG-23 is the last
milestone, so it cannot be carried forward either.

## Not modified, and why

| Element | Verdict |
|---|---|
| `R-RUN-002`'s *"forwarding MUST NOT depend on the turn returning"* clause | **Byte-unchanged.** The forwarder still relays each event while the turn is in flight; nothing about mid-turn observability moves |
| `R-RUN-002`'s *"MUST NOT rewrite, synthesize, suppress or reorder"* clause | **Byte-unchanged and re-proven.** `S-RUN-117` asserts stream identity on every non-panicking path rather than assuming it |
| `R-RUN-003`'s bracket and lane rules | **Byte-unchanged.** No event is added, removed or re-placed |
| `R-RUN-013` item 3 — *"every read is downstream of the forwarder's completion"* | **Byte-unchanged and strengthened in fact.** Those reads sit on the normal path, where the existing join still runs first; the new deferred join adds a second, later happens-before edge and removes none |
| `R-RUN-010`'s no-timeout prohibition, `NFR-RUN-002`'s no-wall-clock rule | **Byte-unchanged and honoured.** The fix adds no timeout, deadline, sleep or poll; termination is by channel close, and the join is unbounded because it is now provably finite |
| `R-RUN-001`'s named method surface | **Byte-unchanged.** The change is internal to one function; no exported member and no signature arity moves |

## ADDED Requirements

### R-RUN-014 — The per-attempt forwarder exits on **every** path, and the consumer sink never closes under a live sender

The harness MUST guarantee, on **every** exit path of a run including the **panic unwind**, that the
per-attempt forwarder goroutine has **terminated** before the consumer sink is closed. Two properties MUST
hold jointly, and neither alone is sufficient:

1. **Termination.** The forwarder MUST have a cancellation path that makes both its receive from the turn's
   sink and its send to the consumer sink abandonable. On the panic path the turn's sink is never closed, so
   a forwarder without an abandonable **receive** would block forever.
2. **Happens-before.** The run MUST establish an ordering edge from the forwarder's exit to the consumer
   sink's close, so that the close provably runs after the last possible send.

**Termination without the ordering edge is a race, and the ordering edge without termination is a
deadlock.** A run whose consumer has abandoned the sink MUST still unwind: an unbuffered send with no
receiver is not ready, so an abandoned forwarder MUST deterministically take its cancellation path rather
than park. **Trading a crash for a hang is not a fix**, and both directions MUST be proven separately.

The cancellation signal MUST be **dedicated to this purpose** and MUST NOT be the run's own cancellation
context. On the interrupt path the run context is **already cancelled** while legitimate wind-down events
are still flowing through the forwarder; a select with two ready cases chooses pseudo-randomly, so keying
the send on the run context would **drop events on a non-panicking path**. The dedicated signal is raised
only inside the run's own unwind, after the last normal join has already completed, so on every
non-panicking path the forwarder's behaviour degenerates to today's and the event stream is unchanged **by
construction**, not by tolerance.

The panic MUST still propagate **uncontained**: this requirement changes what the *harness* leaves behind
during the unwind, and MUST NOT recover, swallow, wrap, or re-type the panic, and MUST NOT convert it into
an error return or an event.

On the panic path the stream MAY end without a run-close bracket. That path never carried a stream
guarantee, and this requirement does not create one.

#### Scenarios

- **S-RUN-114** — **(the RED, watched failing before the fix exists)** Given a harness whose turn panics from the unrecovered pre-request seam with a **uniquely minted runtime sentinel**, and a consumer that drains exactly the run-open event and then **abandons** the sink, when the run is driven repeatedly in one test, then the recovered value is that **exact sentinel by identity** (not by shape) and **no** send on a closed channel occurs on any iteration. Before the production fix exists this test FAILS by **process crash** naming the send on a closed channel — that failure text is recorded, and it is what proves the test fails for the intended reason rather than for any reason.
- **S-RUN-115** — **(defeat A — termination removed)** Given the fix with the forwarder's **send** reverted to an unabandonable send while the deferred join is kept, when `S-RUN-114` runs, then it **deadlocks**: the deferred join blocks forever on the forwarder's own completion signal because the forwarder itself is now permanently parked in its unabandonable send, so `close(sink)` is never reached and a send on a closed channel is structurally **unreachable** on this defeat. The failure surfaces as a test timeout naming exactly two blocked goroutines — the deferred join blocked on a channel receive, and the forwarder blocked on its own channel send. Recorded, then reverted.
- **S-RUN-116** — **(defeat B — the ordering edge removed)** Given the fix with the cancellation path kept but the deferred join **deleted**, when `S-RUN-114` runs, then it fails: with the signal raised and the sink closed back to back, a forwarder at its select has both cases ready and the runtime chooses uniformly, so the miss probability across the test's iteration count is negligible. Recorded, then reverted. **Both defeat directions are load-bearing, and each closes a different hazard, not the same one twice**: defeat A proves the **termination** half — remove it and the join has nothing left to wait for except a forwarder that can never abandon its send, so the run **hangs**; defeat B proves the **happens-before** half — remove it and the abort and the close race each other, so the run **crashes**. One direction proves termination (hang); the other proves the happens-before edge (crash). A single-direction proof would leave a guard that passes with half its mechanism defeated.
- **S-RUN-117** — **(no hang, and no stream change)** Given a run whose consumer abandons the sink and whose turn panics, when the unwind runs, then it **completes** — it does not deadlock on the join — and the sink is closed. And given every non-panicking arm (success, turn failure, retry exhaustion, interrupt, shutdown, compaction), when each is driven, then its recorded event sequence is **identical** to the same script's at the merge base, the exported stream validator accepts each unmodified, and the module's ordered-kind and wind-down suites pass byte-unchanged under `-race -count=1`.
