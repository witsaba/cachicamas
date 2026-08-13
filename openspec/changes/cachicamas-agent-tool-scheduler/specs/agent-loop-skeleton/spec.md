# Delta for `agent-loop-skeleton` — AG-09 tool registration and scheduler call site

> **Change**: `cachicamas-agent-tool-scheduler` · **AG-09** (Layer 2, Wave 2, milestone 9 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-09--define-the-tool-execution-contract-and-scheduler), `0003:902-1004`
> **Modifies**: `agent-loop-skeleton` spec (`openspec/specs/agent-loop-skeleton/spec.md`, AG-07 PR #167 merged at `93077c07`).
> **Format**: per `openspec/config.yaml` `rules.specs`. The archive step REPLACES the requirement block in the main spec with the MODIFIED block below; full-block preservation is therefore mandatory.

## ADDED Requirements

### Requirement: One cycle per turn — the wording-trap boundary with AG-13

The loop SHALL schedule at most one cycle of `model → tools → finalize` per `Turn` invocation. Iteration across cycles (`model → tools → model`) is **AG-13's `Harness` contract**; AG-09 MUST NOT loop the cycle within `Turn`. This is the wording trap from `0003:107-112` — the loop schedules, it does not iterate. The scheduler's `Schedule` function is the seam: callable from `Turn` (AG-09) or from `Harness` (AG-13), but `Turn` MUST call it at most once per invocation.

(Reason: enforce the AG-09 / AG-13 scope boundary at the spec level. The wording trap is the design — AG-09 ships the scheduler + one-cycle wire-up; AG-13 ships the harness that wraps `Turn` in an iteration loop.)

#### Scenarios

- **S-LSK-008** — One cycle per turn. Given a provider that streams one round of tool-call events (`ToolCallStart` / `Delta` / `End`) followed by a `Completion` with `FinishReasonToolCalls`, when `Turn` runs, then `Schedule` is invoked exactly once per `Turn` (asserted via a scheduler mock counter) and `Turn` returns without re-entering `provider.Stream` — even if the tool results would warrant a follow-up model call.
- **S-LSK-008a** — **(bite)** RED-first. Given a `Turn` whose post-`Schedule` path erroneously re-invokes `provider.Stream`, when the cycle-count scenario runs, then the mock counter reports > 1 — proves the one-cycle invariant is non-vacuous. RED-recorded BEFORE `S-LSK-008` is GREEN.

## MODIFIED Requirements

### Requirement: Loop surface: single-turn function form (D1)

The system SHALL expose `func Turn(ctx context.Context, provider ai.ModelProvider, system string, transcript []ai.Message, opts TurnOptions, sink chan<- *Event) (msg ai.Message, finish ai.FinishReason, err error)` as the only public surface for one assistant turn (per D1, D2, D3). AG-13 introduces a value-form `Harness` that wraps `Turn` without changing its signature. `TurnOptions` carries a `Tools map[string]Tool` field for AG-09 (non-breaking zero-value extension; nil `Tools` = the scheduler returns typed `ExecutionFailure` results in their ordinal slots, consistent with R-TLS-009 "one bad tool does not abort the turn"). `TurnOptions` also carries the AG-08 `PreRequestHook` field (nil = identity default).

(Previously: `TurnOptions` carried only trivial/zero fields for AG-07; the AG-08 `PreRequestHook` seam and the AG-09 `Tools` registration path were both absent.)

#### Scenarios

- **S-LSK-001** — AG-07.1 walking skeleton. Given a text response scripted on the fake provider (one `ai.MessageStart`/`ai.MessageDelta`/`ai.MessageEnd` text stream + one `ai.Completion`), when `Turn(ctx, provider, system, transcript, opts, sink)` runs, then the consumer (draining `sink`) observes in order: `run_start`, `turn_start`, `message_start_text`, the deltas in order, `message_end_text`, `turn_end` (`TurnOutcomeFinished`), `run_end`; the sink is closed after `run_end`; the function returns `(msg, finish, nil)` where `finish` is the provider's `ai.FinishReason` (per D3).
- **S-LSK-002** — AG-07.1 provider stream drained and caller's context respected. Given a turn in progress with a non-cancelled `ctx`, when the provider stream reaches its terminal event (`ai.Completion` or typed error), then the loop has drained the provider's channel fully (no goroutine leak), and the loop has passed `ctx` unchanged to `provider.Stream(ctx, req)` (per D5), and the consumer's drain unblocks without blocking on a stranded producer.
- **S-LSK-003** — AG-07.1 one source of truth for the assistant message. Given a completed turn, when the caller reads the loop's returned `msg` AND a consumer reconstructs an `ai.Message` from the emitted deltas via the AG-05.3 helper (`reconstruction_test.go:54-114`), then the two `ai.Message` values are equal as Layer 1 message values (fragment-for-fragment byte-equal).
- **S-LSK-003a** — **(bite)** Given a complete turn with three text deltas, when the loop's emitted event sequence is REWRITTEN to drop the middle delta, then the reconstructed message differs from the loop's returned `msg` — proving the property is non-vacuous. RED-recorded BEFORE `S-LSK-003` is GREEN.
- **S-LSK-003b** — **(bite)** Given a complete turn with three text deltas, when the loop's emitted event sequence is REWRITTEN to double the middle delta, then the reconstructed message differs from the loop's returned `msg`. RED-recorded BEFORE `S-LSK-003` is GREEN.
- **S-LSK-009** — AG-09 wire-up: `Turn` consumes AI-18 tool-call events and calls `Schedule`. Given a `TurnOptions{Tools: map[string]Tool{...}}` with one registered read-class tool and a provider that streams one `ToolCallStart` / `ToolCallDelta` / `ToolCallEnd` triplet followed by a `Completion{FinishReason: FinishReasonToolCalls}`, when `Turn` runs, then the loop converts the AI-18 events into a `[]ScheduledCall`, invokes `Schedule` exactly once between `provider.Stream` close and `finalize`, and emits the AG-05.2 tool events (`ToolStart`, `ToolEnd*`) on `sink` in rejoin order — proving the AG-09 wire-up.

### Requirement: Test coverage on `loop.go`

The system SHALL achieve ≥ 80% line coverage on `backend/agent/src/agent/loop.go` per `make test` race-gated run, satisfying AG-04 W8. The coverage SHALL include the AG-09 wire-up: the path from `provider.Stream` close through `Schedule` invocation through tool-event emission SHALL be covered; the "loop schedules; does not iterate" invariant (`S-LSK-008`) SHALL be covered by a test that asserts `Schedule` is invoked exactly once per `Turn` even when tool results would warrant a follow-up model call.

(Previously: coverage was scoped only to the AG-07 walking skeleton + the AG-08 `PreRequestHook` seam; the AG-09 wire-up (Tools registration → translate() widening on AI-18 events → Schedule call site → tool-event emission) and the one-cycle-per-turn invariant were both absent from the coverage discipline.)

#### Scenarios

- **S-LSK-007** — Given `make test` green in `backend/agent/`, when the coverage report is read for `backend/agent/src/agent/loop.go`, then the line coverage is ≥ 80%, and `loop_tool_dispatch_test.go` is part of the covered surface.