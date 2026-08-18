# Delta for `agent-cancellation-tree` — AG-15 confirms the cancelled-turn row closed by gate ordering, not weakened

> **Change**: `cachicamas-agent-retry-failover` · **AG-15** (Layer 2, Wave 3), `0003:1444-1525`
> **Modifies**: `agent-cancellation-tree` ([`../../../../specs/agent-cancellation-tree/spec.md`](../../../../specs/agent-cancellation-tree/spec.md)) — the Explicit non-requirements table (`spec.md:160-177`).
> **Back-annotation only.** No requirement text is amended, no requirement gains or loses a scenario, and no cancellation obligation is relaxed. The table is reproduced in full; exactly one row is back-annotated and none is removed.
> **Why this delta**: the row "Retry or failover on a cancelled turn — **AG-15**" (`spec.md:168`) names this change as its owner. AG-15 is the milestone that could have weakened it — it is the one that adds retry — so leaving the row unannotated would leave a reader unable to tell whether AG-15 honoured it or quietly overrode it. The answer is recorded as a **mechanism**, not as a reassurance: the cancellation check is gate **G0** of `R-RTY-001`'s ordered predicate, evaluated **ahead of** every gate that can produce a retry, and AG-15 adds no second cancellation check and moves no existing one.
> **Ownership**: the retry predicate and its gate ordering are owned by [`../agent-retry-failover/spec.md`](../agent-retry-failover/spec.md) (`R-RTY-001`, `R-RTY-008`).

## MODIFIED Explicit non-requirements

Stated so that no test, guard or acceptance line is written as if AG-14 closes more than it does.

| Not claimed | Owner |
|---|---|
| Subagent cancellation inheritance | **AG-19.2.** Charter "Out of scope" line (`0003:1381`); no subagent tool ships in v1 (`0003:1794`) |
| A keypress, `SIGINT` or any other frontend signal reaching the harness | **Layer 3.** AG-14 defines the mechanism; the composition root calls it |
| Retry or failover on a cancelled turn | **AG-15.** A cancellation is never retried; `R-RUN-011`'s no-retry rule extends verbatim to the cancellation path. **CONFIRMED CLOSED by AG-15, and closed by mechanism rather than by promise.** AG-15 keeps the existing cause check at its existing site as gate **G0** of `R-RTY-001`'s ordered, first-match-wins predicate, evaluated **before** G1–G5, so no cancelled turn can reach a retry, a backoff wait, or the failover consult. AG-15 adds no second cancellation check, moves no existing one, and introduces no cancellation vocabulary. A cancellation arriving **during a backoff wait** is the one new arrival point, and it routes to the same wind-down (`R-RTY-008`) with the same interrupted-or-shutdown run outcome and the same nil `*Failure` — one more way *into* the wind-down, never a way out of it. A bare cancellation of the caller's own context, matching neither sentinel, keeps AG-14's scope-line routing and is likewise never retried: it reaches the failure surface through the untyped or non-retryable gates, not through the retry gate |
| Cost accounting for an interrupted turn | **AG-16** |
| Compaction cancellation-safety | **AG-17 / AG-18** |
| The package-wide goroutine-leak sweep | **AG-21** (`0003:2178`). AG-14 proves only its own goroutines and makes the detached-call report precise enough for that sweep (`0003:1439`) |
| A **deadline** as a third signal | **Not this milestone.** `0003:1373` is a distinctness claim, not a mandate; AG-14.3's bound is a wind-down bound, not a run deadline. *(Still unclaimed at AG-15: the backoff wait is bounded by the retry policy's own computed or reported delay and is not a run deadline)* |
| Concurrent runs on one harness value | **Not this milestone.** `R-CAN-002` re-scopes the rule to *one run at a time*, not to concurrency |
| Cross-run transcript state beyond the terminal shutdown flag | **AG-21** (`agent-run-driver`'s `R-RUN-001`, its one-run-at-a-time clause). *(Still true at AG-15: the attempt counter lives inside one `Run` call and outlives nothing)* |
| A new `EventKind`, a new `TurnOutcome`, or a new exported `History` method | **Not this milestone**, and forbidden under this change. *(Still true at AG-15, which adds none of the three)* |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2.** Layer 1 is consumed, never edited. *(Still true at AG-15, which reads one Layer 1 documentation file and writes nothing under that tree)* |
