# Delta for `agent-tool-scheduler` — AG-13 adds the sink-ownership seam and re-homes the `ToolSource` line

> **Change**: `cachicamas-agent-run-driver` · **AG-13** (Layer 2, Wave 3), `0003:1294-1370`
> **Modifies**: `agent-tool-scheduler` ([`../../../../specs/agent-tool-scheduler/spec.md`](../../../../specs/agent-tool-scheduler/spec.md)) — adds `R-TLS-012`; modifies the Explicit non-requirements list (`spec.md:133`, `spec.md:138`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES each block in the main spec with the block below; **full-block preservation is mandatory**.
> **Why this delta**: three things. (a) `spec.md:133` — "**Iteration of the model ↔ tools ↔ model cycle** — AG-13. AG-09 schedules one cycle; AG-13 iterates" — closes here. (b) `Schedule` closes the caller's `sink` unconditionally after the rejoin (`scheduler.go:219`), which the continuation path's schedule-before-finalize ordering (`R-LSK-001`) turns into a send on a closed channel; a zero-default sink-ownership flag on `Scheduler` (`tool.go:229`) is the seam, and no existing requirement states who owns the sink, so this delta adds one rather than amending one. (c) `spec.md:138` names AG-13 as the widener of the `ToolSource` port, which the AG-13 charter never mentions; it is re-homed to AG-20.
> **Ownership**: the sink-ownership contract is owned here by the new `R-TLS-012`. The **decision to set** the flag, and everything the run driver does with the scheduler value, are owned by `R-RUN-001`/`R-RUN-010`/`R-RUN-012` in [`../agent-run-driver/spec.md`](../agent-run-driver/spec.md). `WakeParked`'s own semantics stay owned by `agent-permission-protocol`.

## ADDED Requirements

### Requirement: Sink ownership is a caller-selectable, zero-default contract — `R-TLS-012`

`Schedule` today closes the caller's `sink` after the rejoin, unconditionally. That is correct when `Schedule` is the last writer to the sink, and wrong when it is not — and on the continuation path of `R-LSK-001` it is not, because the schedule-before-finalize reorder puts the turn-close event **after** the rejoin. Closing there would make that emission a send on a closed channel.

The `Scheduler` MUST therefore carry a **sink-ownership flag whose zero value preserves AG-09 behavior exactly**: with the flag unset, `Schedule` closes the sink after the rejoin, as it always has. With the flag set, `Schedule` MUST leave the sink open after the rejoin and the **caller** becomes responsible for closing it exactly once. Every other step of `Schedule` — the parked-set clear, the emissions close, the dispatcher join, the ordered rejoin — MUST be unchanged in behavior and in order; only the close is conditional.

The flag MUST be a field on the `Scheduler` value rather than a new `Schedule` parameter, because a parameter would break the public `Schedule` signature that AG-09's and AG-10's suites are written against. Because every existing construction site builds the scheduler with a keyed struct literal, a new field is invisible to them and every AG-09/AG-10 scheduler test MUST pass with its source **byte-unchanged**.

The flag MUST NOT change how many times the sink is closed overall. Exactly one close MUST happen per sink, by exactly one owner. A caller that sets the flag and never closes, or closes twice, is a caller defect; the scheduler makes ownership selectable, it does not make it optional.

Setting the flag MUST NOT affect the dispatcher's need for a reader: the pre-existing AG-09 hazard that `sink <- &stamped` requires a live consumer is unchanged by this seam.

#### Scenarios

- **S-TLS-013** — Given a `Scheduler` constructed with the sink-ownership flag at its zero value and a set of scheduled calls, when `Schedule` runs and a consumer drains `sink`, then the ordered rejoin is fully populated, the consumer observes the sink close after the last tool event, and the emitted event sequence is byte-identical to the pre-AG-13 sequence for the same input.
- **S-TLS-014** — Given a `Scheduler` constructed with the sink-ownership flag set, when `Schedule` runs to its rejoin, then the sink is **not** closed, a subsequent send on it by the caller succeeds and is observed by the consumer, and the consumer observes the close only when the caller closes it; and the ordered rejoin is fully populated exactly as in `S-TLS-013`.
- **S-TLS-015** — Given every existing AG-09 and AG-10 scheduler test, when the suite runs against this change, then all of them pass with their source files **byte-unchanged**, because each constructs the scheduler with a keyed struct literal and therefore leaves the new field at its zero value.

## MODIFIED Explicit non-requirements

The list is reproduced in full; one line is back-annotated as closed, one is re-homed, and none is removed.

- **No edit** to any substrate file in `NFR-TLS-003`. AG-09 is the 6th consecutive extensibility demonstration. *(Still true at AG-13: `tool.go` and `scheduler.go` are not members of that list — it names `tool_event.go`, a different file — so AG-13's edits to them need no release. Verified against `agent-loop-skeleton/spec.md:60`.)*
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit. Hand-rolled `chan struct{}` semaphore + serialized channel + `defer/recover` per call goroutine; `errgroup` is FORBIDDEN (new top-level dep + first-error cancellation conflicts with `R-TLS-010`). *(Still true at AG-13: the run driver adds no dependency.)*
- **Permission protocol around the scheduler** — AG-10. AG-09's scheduler accepts a single `policy PolicySlot` per call; AG-10 wraps that with decision-required / decision-made events.
- **Iteration of the model ↔ tools ↔ model cycle** — AG-13. AG-09 schedules one cycle; AG-13 iterates. **CLOSED by AG-13**: iteration ships in the run driver, which calls `Turn` repeatedly (`R-RUN-002`). The scheduler is unchanged in this respect — it still schedules exactly one cycle's calls per `Schedule` invocation, and the run driver never calls `Schedule` itself (`R-LSK-006`, as reconciled).
- **What any tool does** — Layer 3 built-in tools (doc 0004). AG-09 ships the contract; doc 0004 implements built-ins against it.
- **Sandbox semantics** — Layer 3 interprets `PolicySlot`. AG-09 forwards it byte-exact (R-TLS-002); AG-09 does not read it.
- **Subagent tool** — v1 non-goal per doc 0003 § 8.
- **The four-hook taxonomy** (AG-20) — AG-09 ships `Tools` as a map; AG-20 widens to a `ToolSource` port if/when needed.
- **`ToolSource` port** (G6) — **AG-20** widening, not AG-09 and **not AG-13**. Re-homed by this change for two recorded reasons: the AG-13 charter (`0003:1294-1370`) never mentions tool sources anywhere in its goal, deliverable, acceptance clauses, or three leaves; and the immediately preceding line already assigns the four-hook taxonomy and the `ToolSource` widening to AG-20, so the two adjacent lines contradicted each other. AG-13 implements no part of it. **The identical claim also lives in code**, as a comment on the `Registry` interface at `backend/agent/src/agent/tool.go:239-240` — "`ToolSource` port (G6) is AG-13's widening" — and that comment MUST be re-homed to AG-20 in the **same pull request** as this delta. Spec and code drifting apart on the same sentence is the failure mode this repo has recorded before; changing one and not the other is not an acceptable outcome of this change.

## Verification note for `sdd-apply`

`S-TLS-015` and the code-comment re-home are both checkable by command, not by claim:

- `git diff` over `backend/agent/src/agent/` MUST show every AG-09/AG-10 scheduler test file byte-unchanged;
- `grep -n "AG-13" backend/agent/src/agent/tool.go` MUST return nothing about a `ToolSource` widening after the change lands.
