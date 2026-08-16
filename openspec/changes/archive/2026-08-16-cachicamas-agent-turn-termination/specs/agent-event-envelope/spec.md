# Delta for `agent-event-envelope` — AG-11.2 closes envelope invariant 4

> **Change**: `cachicamas-agent-turn-termination` · **AG-11** (Layer 2, Wave 2, milestone 11 of 24), `0003:1113-1176`
> **Modifies**: `agent-event-envelope` (`openspec/specs/agent-event-envelope/spec.md`) — `R-AEV-008` only.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES the requirement block in the main spec with the MODIFIED block below; full-block preservation is mandatory.
> **Why this delta is mandatory**: `R-AEV-008` already records that it "closes invariant 4 jointly with AG-11.2" and that "the loop-level typed-error emission path is AG-11's" (`spec.md:132`; `0003:2162`, `:2203`), and `invariant_pin_test.go:1-8` repeats the joint-closure claim. AG-11 is the co-closing milestone, so the requirement must name what AG-11 adds — the partial-output discriminator and the loop-level emission obligation — or the joint-closure claim becomes unauditable.

## MODIFIED Requirements

### Requirement: Invariant pin 4 — a failure payload is a typed value, never a message string — `R-AEV-008`

A failure carried by the envelope MUST be reachable through a **typed-failure surface** on which the failure's **category** and its **cause** are inspectable as values (`VL2-EVT-15`, envelope invariant 4, `decision.md:176`). The category vocabulary MUST be aligned with Layer 1's failure taxonomy — `ai.FailureCategory` (`backend/agent/src/ai/provider_failure.go:49`) and `(*ai.Failure).Category()` (`provider_failure.go:432`) — as a **wrap**, not a reuse-as-is (`S-AGV-020`).

Setup failures (pre-stream) and stream failures (mid-stream) MUST be distinguishable, mirroring Layer 1's two-constructor, one-concrete-type shape (`provider_failure.go:610`, `:622`). **No code path MAY assign meaning to a message string**, and no scenario in this spec is satisfied by asserting on message text.

The typed-failure surface MUST additionally expose the **partial-output discriminator** — `PartialOutput() bool`, delegating unchanged to `(*ai.Failure).PartialOutput()` (`provider_failure.go:515-520`) and returning `false` for a nil receiver, exactly as `Category()`, `Delivery()` and `Retryable()` do (`failure.go:44`, `:54`, `:64`). Without it, a consumer holding an `*agent.Failure` cannot answer "did output precede this failure?", which invariant 4 requires to be answerable from typed values alone.

The envelope's obligation is matched by a **loop-level emission obligation**: a turn that ends in a terminal provider error MUST carry its `*Failure` to the consumer through the already-registered typed outcomes — `turn_end` with `TurnOutcomeAborted` and `run_end` with `RunOutcomeFailed` — emitted before the sink closes. No new `EventKind` is registered for failure; failures ride the typed outcomes (`failure.go:6-7`) and the every-kind-constructible guard stays at 25 kinds.

This requirement closes invariant 4 **jointly with AG-11.2** (`0003:2162`, `:2203`); neither milestone closes it alone. AG-04.3 owns the typed surface and its pins; AG-11.2 owns the discriminator and the loop-level emission path, specified as `R-ATT-005`, `R-ATT-006` and `R-ATT-007` in [`../agent-turn-termination/spec.md`](../agent-turn-termination/spec.md).

(Previously: the requirement named only category, cause and the pre-stream/mid-stream distinction, and deferred the loop-level emission path to AG-11 without stating the partial-output discriminator or the emission obligation as requirements of this surface.)

#### Scenarios

- **S-AEV-070** — Given a failure payload carried by the envelope, when a consumer in an external package inspects it through the typed-failure surface, then the category is reachable as a typed value and the cause is reachable as an inspectable error value.
- **S-AEV-071** — Given a failure payload constructed on the pre-stream path and one constructed on the mid-stream path, when each is inspected, then the two are distinguishable through the typed surface without parsing any text.
- **S-AEV-072** — Given the Layer 2 failure category vocabulary, when it is compared with `ai.FailureCategory`'s nine members (`provider_failure.go:49-103`), then every Layer 2 category maps to a stated Layer 1 category and the mapping is declared in source, not inferred.
- **S-AEV-073** — Given every test written for this capability, when their assertions are enumerated, then none asserts on the content of a failure message string as the carrier of meaning.
- **S-AEV-074** — AG-11.2 co-closure: the discriminator is reachable through Layer 2. Given a mid-stream `*ai.Failure` constructed with the partial-output flag set and a second with it unset, when each is wrapped and inspected from an external test package in the `invariant_pin_test.go` family, then `PartialOutput()` reports `true` and `false` respectively, a nil `*agent.Failure` reports `false` without panicking, and the assertion reads no message string. Cross-referenced to `R-ATT-006` / `S-ATT-008`.
- **S-AEV-075** — AG-11.2 co-closure: the loop delivers the typed failure through the registered outcomes. Given a provider stream scripted to deliver partial content and then a terminal mid-stream failure, when `Turn` runs and a consumer drains the sink to close, then the consumer observes `turn_end(TurnOutcomeAborted)` and `run_end(RunOutcomeFailed)` each carrying a non-nil `*Failure` whose `Category()`, `Delivery()`, `Retryable()` and `PartialOutput()` are all inspectable as typed values, no new event kind appears (the guard still passes at 25 kinds), and no assertion reads a message string. Cross-referenced to `R-ATT-005` / `S-ATT-007`.
