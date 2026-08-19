# Delta for `agent-permission-protocol` — AG-19.3 closes the subagent-scope row, and the gate is REUSED rather than re-opened

> **Change**: `cachicamas-agent-delegation-readiness` · **AG-19** (Layer 2, Wave 5), `0003:1793-1862`
> **Modifies**: `agent-permission-protocol` ([`../../../../specs/agent-permission-protocol/spec.md`](../../../../specs/agent-permission-protocol/spec.md)) — the non-requirement row at `spec.md:169` and the two `Blocks` statements at `spec.md:9` and `spec.md:177`.
> **BACK-ANNOTATION ONLY, and deliberately scenario-free.** No requirement is amended, no scenario is added or removed, and **no count moves** — the target spec's header states `12 requirements → 13 spec scenarios + 4 bites` (`spec.md:7`) and its coverage table repeats it (`spec.md:15`). AG-19 adds neither, so both stay true untouched. This was checked before writing: a delta that appended one scenario here would have silently falsified two count assertions, which is this repository's known drift class.
> **Where AG-19's permission behaviour actually lives**: [`../agent-delegation-readiness/spec.md`](../agent-delegation-readiness/spec.md) `R-DEL-008` / `S-DEL-018` / `S-DEL-019`, and the routing discharge is recorded in [`../agent-event-delivery/spec.md`](../agent-event-delivery/spec.md) `R-AGE-016`. This delta owns only this capability's audit statement.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-APP-001`…`R-APP-012` and all 13 scenarios and 4 bites | **Untouched.** AG-19 reuses the gate exactly as shipped: the ask, the park, the four outcomes, the sibling isolation and the stray-wake rule all apply to a child's calls unchanged, because a child harness runs an ordinary `Scheduler` |
| `R-APP-003`'s stray-wake rule | **Confirmed.** The delegating tool wakes the **child's** scheduler with the child's own call identity; a wake aimed at the wrong scheduler still resolves to the existing stray-decision sentinel |
| `R-APP-010`'s remembered-resolution scope — one `Schedule` call | **Not widened.** `permission_resolution_remembered` is `CardinalityAtMostOne`, so AG-19's seam **refuses** it and it never crosses onto the parent's stream. A remembered rule stays inside the child's own `Schedule` call, exactly as this row already says |
| `NFR-APP-004` — no new `EventKind`, no new `PermissionOutcome` member | **Confirmed.** AG-19 registers none and adds none |
| The `PermissionPolicy` interface signature (`permission_protocol.go:80-94`) | **Byte-unchanged.** A derived scope is an ordinary implementing value, not a new type in this capability |

## MODIFIED Explicit non-requirements

The list is reproduced only where AG-19 touches it; every other row is unchanged and none is removed.

- **Subagent tool scope** — was: *"AG-19.3."* **CLOSED by AG-19.3 — and closed by composition, with no new Layer 2 type.** The child's policy is an ordinary Go value composing the parent's, implementing this capability's existing interface by delegating to it. Four facts, recorded as mechanism rather than reassurance:
  1. **"What the parent's policy allowed flows down" is given an operational meaning, and both directions are asserted.** A derived scope may only **narrow** — mapping a parent allow to a deny or defer for a tool outside the child's grant — and MUST NOT **widen** — it may never map a parent deny or defer to an allow. AG-19 asserts the widening direction as impossible, not merely unused (`S-DEL-018`).
  2. **The ask goes up and the decision comes down through the EXISTING surfaces.** The child's `permission_decision_required` and its answering `permission_decision_made` are mirrored onto the parent's stream — the one place a human watches — and the human's verdict reaches the child's suspension through the existing wake surface (`scheduler.go:264-272`), which the delegating tool calls on the child `Scheduler` value it already owns. **Zero new production routing surface ships**, which is precisely what `S-AGE-022` required of this milestone.
  3. **The pair crosses together or not at all.** A mirrored ask with no mirrored answer is unreadable to the human, so both kinds are admissible at the seam and neither is admissible alone.
  4. **"Scope" does NOT become a Layer 2 concept.** No scope type, rule set or mode flag enters this capability; that stays Layer 3's (CO-03, `permission_protocol.go:77-79`). *(A reader must not take this row as "a subagent tool shipped"; it did not. The scope composition lives in `package agent_test`, which production code cannot import.)*

## MODIFIED Dependencies

The `Blocks` statements at `spec.md:9` and `spec.md:177` are reproduced with their AG-19 clause discharged; neither is removed and no dependency is added.

- **Blocks**: AG-13 (`Harness` owns the upward-path wake wiring — **CLOSED by AG-13**); AG-19 (subagent tool scope reuses the gate — **CLOSED by AG-19.3**: the gate is reused verbatim by a child `Scheduler`, the wake surface is the one AG-10 shipped and AG-13 wired, and this capability gained no requirement, no scenario, no type and no exported identifier from that reuse).
