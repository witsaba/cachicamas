# Delta for ai-stream-lifecycle

> **Change**: `cachicamas-ai-backpressure` · **Milestone**: AI-34 (doc 0002:2041–2073) · **Wave 5 — Harden**
> **Module**: `backend/agent/` (ADR 0005 § D1 row 1, § D3) · **Target spec**: `ai-stream-lifecycle/spec.md` § 6 (lines 381–451)
> **Amendment follow-up**: line 387 placeholder `N` resolves at archive phase from `decision.md`

## Purpose

Doc 0002 lines 2047–2049 chartered AI-34 with one goal (confirm or change AI-02.1's buffer decision with measurements), one deliverable (buffer constant + slow-consumer tests + rationale), and one acceptance (ordering stable, memory bounded, cancellation unblocks a saturated producer). This delta states the behavioural contract: measured capacity `N` per `decision.md`, lossless ordering under pressure, no unsanctioned loss path beyond AI-20.3 / AI-33.3.

## ADDED Requirements

### Requirement: R-AIS-039 — Real-producer capacity equals the measured decision

The carrier (`V-STR-02`) returned from the real producer's stream operation MUST have capacity equal to `N` recorded in `decision.md`. Asserted directly against the real producer over a real transport — NOT over the fake provider, whose `Script.Buffer` is independent.

#### Scenario: R-AIS-039 / S-1 — Capacity observable on stream creation *(pin: `V-STR-08`)*

- **GIVEN** the real producer and `decision.md` with measured `N`
- **WHEN** the caller invokes the stream operation
- **THEN** the carrier's capacity equals `N` exactly

#### Scenario: R-AIS-039 / S-2 — Slow consumer preserves ordering *(pin: `V-STR-23`, `R-AFP-011`)*

- **GIVEN** the real producer serving frames and a consumer with fixed inter-read interval
- **WHEN** the consumer drains to close without cancelling
- **THEN** every event arrives in order, AND no event is lost, AND no terminal is invented

#### Scenario: R-AIS-039 / S-3 — No auxiliary queue *(pin: `V-STR-08`, `R-AFP-011`)*

- **GIVEN** the real producer and a consumer paused while the carrier is at capacity
- **WHEN** the producer attempts the next send
- **THEN** it blocks on the carrier — does NOT buffer elsewhere — AND on resume the consumer receives exactly the carrier-held events

### Requirement: R-AIS-040 — No unsanctioned loss path across non-cancellation patterns

For every non-cancellation consumer pattern (**slow**, **bursty**, **pause-resume**) across both stream kinds (text + tool-call), the consumer MUST receive every emitted event in order. No loss path beyond AI-20.3 / AI-33.3. Proof MUST cover the full matrix in one serial test × 50 repeats (per `R-STK-008`).

#### Scenario: R-AIS-040 / S-1 — Slow, both kinds, no loss no leak *(pin: `R-STK-007`, `R-STK-008`)*

- **GIVEN** real producer, both kinds, consumer reads slower than producer emits
- **WHEN** consumer drains to close across 50 repeats
- **THEN** every event arrives in order on both kinds, AND no terminal invented, AND no goroutine growth

#### Scenario: R-AIS-040 / S-2 — Bursty, both kinds, no loss no leak *(pin: `R-STK-007`, `R-STK-008`)*

- **GIVEN** real producer, both kinds, consumer reads at bursty cadence
- **WHEN** consumer drains to close across 50 repeats
- **THEN** every event arrives in order on both kinds, AND no terminal invented, AND no goroutine growth

#### Scenario: R-AIS-040 / S-3 — Pause-resume, both kinds, no loss no leak *(pin: `R-STK-007`, `R-STK-008`)*

- **GIVEN** real producer, both kinds, consumer pauses then resumes
- **WHEN** consumer resumes and drains to close across 50 repeats
- **THEN** every event during pause and resume arrives in order on both kinds, AND no terminal invented, AND no goroutine growth

## MODIFIED Requirements

### Requirement: R-AIS-031 — Buffer bounded

The buffer between producer and consumer (`V-STR-08`) is **bounded**, with starting capacity of `N` events, where `N` is the value recorded in `decision.md` and re-confirmed by `sdd-verify` at archive time. The capacity `N` is the single value measured against the workload the decision artefact names; the carrier's observed capacity at runtime equals `N` (per R-AIS-039).

(Previously: § 6 line 387 named "starting capacity of 64 events" as a hypothesis, not a measurement. The hypothesis is now resolved by `decision.md`.)

#### Scenario: R-AIS-031 / S-1 — Spec wording matches the decision artefact *(pin: `V-STR-08`)*

- **GIVEN** `openspec/specs/ai-stream-lifecycle/spec.md` § 6 line 387 and `decision.md`
- **WHEN** a reviewer reads the spec wording and decision artefact side-by-side
- **THEN** the capacity named at line 387 equals `N` recorded in `decision.md`

#### Scenario: R-AIS-031 / S-2 — Capacity re-confirmed by `sdd-verify` at archive time *(pin: `R-AIS-039`)*

- **GIVEN** `decision.md` and the `sdd-verify` report at archive phase
- **WHEN** `sdd-verify` records its findings
- **THEN** the verified capacity equals `N` in `decision.md`, AND any divergence is a defect

## REMOVED Requirements

None.

## RENAMED Requirements

None.

## Unchanged Sections (explicit)

| Section | Lines | Reason |
| --- | --- | --- |
| § 6 corollaries (1, 2, 3) | 395–399 | Behaviour, not capacity. Proved by AI-34.2 / AI-34.3. |
| § 6 measurement tables (M1 / M2 / M3) | 422–430 | `decision.md` is the **evidence** for `N`; tables stay verbatim. |
| § 6 tie-break rule | 432 | Rule lives in `decision.md` once applied; spec wording unchanged. |

## References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2041–2073` (Goal 2047, Deliverable 2048, Acceptance 2049; AI-34.1:2053–2058, AI-34.2:2060–2066, AI-34.3:2068–2073)
- **Target spec § 6** — `openspec/specs/ai-stream-lifecycle/spec.md:381–451` (amendment at 387; unchanged 395–399, 422–430, 432)
- **Proposal** — `openspec/changes/2026-08-07-cachicamas-ai-backpressure/proposal.md:41–47` and `:111–113`
- **Explore** — `openspec/changes/2026-08-07-cachicamas-ai-backpressure/explore.md`
- **Engram** — `#2629` (`sdd/cachicamas-ai-backpressure/explore`); `#2630` (`sdd/cachicamas-ai-backpressure/proposal`)
- **Evidence artefact** — `openspec/changes/2026-08-07-cachicamas-ai-backpressure/decision.md` (carries `N`)
- **ADR 0005 § D1 row 1, § D3** — `docs/adr/0005-promote-agent-stack-to-own-module.md`
