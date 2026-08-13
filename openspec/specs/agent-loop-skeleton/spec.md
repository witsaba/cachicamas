# Spec — The one-turn walking skeleton (`agent-loop-skeleton`)

> **Change**: `cachicamas-agent-loop-skeleton` · **AG-07** (Layer 2, Wave 2 opening) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-07--build-the-one-turn-walking-skeleton), `0003:771-832`
> **Nodes**: AG-07.1 `[leaf]` (one text turn) · AG-07.2 `[leaf]` (statelessness + reasoning pass-through)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml`. Every scenario independently verifiable.
> **IDs**: `R-LSK-0NN` / `S-LSK-0NN`. Append-only. Distinct from `R-AEV-`/`S-AEV-`, `R-AMT-`/`S-AMT-`, `R-APE-`/`S-APE-`, `R-AGE-`/`S-AGE-`.
> **Scenario count** (AG-04 W9): **5 charter → 7 spec + 2 bites = 9 total** in 5 requirements.

## Coverage

| Charter | Requirements | Spec | Bites |
|---|---|---|---|
| **5 of 5** | 5 (`R-LSK-001`–`005`) | **7** | **2** (`S-LSK-003a`, `S-LSK-003b`) |

Charter → spec: AG-07.1#1 → `R-LSK-001`/`S-LSK-001`; AG-07.1#2 → `R-LSK-001`/`S-LSK-002`; AG-07.1#3 → `R-LSK-001`/`S-LSK-003` + bites; AG-07.2#4 → `R-LSK-002`/`S-LSK-004`; AG-07.2#5 → `R-LSK-003`/`S-LSK-005`. Cross-cuts → `R-LSK-004` (substrate), `R-LSK-005` (coverage).

## Purpose

The first Layer 2 milestone where a live loop produces events. Per doc 0003:773: "the single most important node in this document: the first time Layer 1 and Layer 2 meet." AG-07 owns the producer side of AG-01's carrier decision (D2a) and gives AG-08/AG-09/AG-11 the surface they wrap. Bites defend the AG-05.1 reconstruction property against the AG-05 W1 vacuous-helper failure mode.

## Requirements

### R-LSK-001 — Loop surface: single-turn function form (D1)

The system SHALL expose `func Turn(ctx context.Context, provider ai.ModelProvider, system string, transcript []ai.Message, opts TurnOptions, sink chan<- *Event) (msg ai.Message, finish ai.FinishReason, err error)` as the only public surface for one assistant turn (per D1, D2, D3). AG-13 introduces a value-form `Harness` that wraps `Turn` without changing its signature. `TurnOptions` carries trivial/zero fields for AG-07.

#### Scenarios

- **S-LSK-001** — AG-07.1 walking skeleton. Given a text response scripted on the fake provider (one `ai.MessageStart`/`ai.MessageDelta`/`ai.MessageEnd` text stream + one `ai.Completion`), when `Turn(ctx, provider, system, transcript, opts, sink)` runs, then the consumer (draining `sink`) observes in order: `run_start`, `turn_start`, `message_start_text`, the deltas in order, `message_end_text`, `turn_end` (`TurnOutcomeFinished`), `run_end`; the sink is closed after `run_end`; the function returns `(msg, finish, nil)` where `finish` is the provider's `ai.FinishReason` (per D3).
- **S-LSK-002** — AG-07.1 provider stream drained and caller's context respected. Given a turn in progress with a non-cancelled `ctx`, when the provider stream reaches its terminal event (`ai.Completion` or typed error), then the loop has drained the provider's channel fully (no goroutine leak), and the loop has passed `ctx` unchanged to `provider.Stream(ctx, req)` (per D5), and the consumer's drain unblocks without blocking on a stranded producer.
- **S-LSK-003** — AG-07.1 one source of truth for the assistant message. Given a completed turn, when the caller reads the loop's returned `msg` AND a consumer reconstructs an `ai.Message` from the emitted deltas via the AG-05.3 helper (`reconstruction_test.go:54-114`), then the two `ai.Message` values are equal as Layer 1 message values (fragment-for-fragment byte-equal).
- **S-LSK-003a** — **(bite)** Given a complete turn with three text deltas, when the loop's emitted event sequence is REWRITTEN to drop the middle delta, then the reconstructed message differs from the loop's returned `msg` — proving the property is non-vacuous. RED-recorded BEFORE `S-LSK-003` is GREEN.
- **S-LSK-003b** — **(bite)** Given a complete turn with three text deltas, when the loop's emitted event sequence is REWRITTEN to double the middle delta, then the reconstructed message differs from the loop's returned `msg`. RED-recorded BEFORE `S-LSK-003` is GREEN.

### R-LSK-002 — Statelessness: two sequential turns share nothing

The system SHALL treat `Turn` as stateless across calls; the second call's event sequence SHALL be independent of the first's, with fresh per-stream ordering starting at 1 (per D1, U3 path-a).

#### Scenarios

- **S-LSK-004** — AG-07.2 two sequential turns share nothing. Given one `Turn` function value that has already run a turn, when a second turn runs via a second `Turn(...)` invocation (fresh slices, fresh `opts`, fresh `sink`, fresh `provider` script), then the second turn's emitted events carry fresh per-stream ordering starting at 1, and the second turn's events do not depend on any state from the first turn (no closure captures over first-turn results, no shared `LaneStamper`).

### R-LSK-003 — Reasoning pass-through: byte-exact round-trip token

The system SHALL distinguish reasoning from text events by kind (`message_start_reasoning`/`message_delta_reasoning`/`message_end_reasoning` vs `message_start_text`/`message_delta_text`/`message_end_text` per AG-05.1), and SHALL preserve the reasoning round-trip token byte-exact into the assistant message.

#### Scenarios

- **S-LSK-005** — AG-07.2 reasoning flows through distinguished, byte-exact. Given a scripted response interleaving reasoning and text deltas (reasoning → text → reasoning → text per D4), with a non-empty `[]byte` reasoning round-trip token on each reasoning end, when the loop re-emits it via `Turn(...)`, then reasoning and text are emitted as separate bracket kinds, and the assistant message's reasoning-content round-trip token is byte-equal to the script's token, and the event order matches the script's emit calls.

### R-LSK-004 — Substrate untouched (4th consecutive milestone)

The system SHALL NOT modify any of: `event.go`, `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `event_registry_test.go`, `doc.go`, `doc_contract_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, `go.mod`, `go.sum`.

#### Scenarios

- **S-LSK-006** — Given main at `8420b2c4` (post-AG-06 merge), when the AG-07 changes are compared against main, then `git diff main -- backend/agent/src/agent/` (excluding `loop.go`, `loop_test.go`) shows zero lines changed, and `git diff main -- backend/agent/go.mod backend/agent/go.sum` is empty, and the every-kind-constructible guard (`event_registry_test.go:54-251`) still passes at 25 kinds (AG-07 adds zero), and AG-03's boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) still pass — AG-07's new code uses only stdlib + `src/ai` + `src/agenttest` (tests).

### R-LSK-005 — Test coverage on `loop.go`

The system SHALL achieve ≥ 80% line coverage on `backend/agent/src/agent/loop.go` per `make test` race-gated run, satisfying AG-04 W8.

#### Scenarios

- **S-LSK-007** — Given `make test` green in `backend/agent/`, when the coverage report is read for `backend/agent/src/agent/loop.go`, then the line coverage is ≥ 80%.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-LSK-001** | External-package verifiability: every scenario verifiable by `cd backend/agent && make test`. Every behavioral test in `package agent_test` or another external package. |
| **NFR-LSK-002** | Determinism and race cleanliness: every test MUST be deterministic, hermetic, and pass under `-race`. |
| **NFR-LSK-003** | Substrate byte-unchanged: the 21 files in `R-LSK-004` are byte-unchanged. 4th consecutive extensibility demonstration. |
| **NFR-LSK-004** | Boundary guards stay green untouched: AG-03's `import_boundary_test.go` and `ambient_authority_test.go` pass with zero changes. |
| **NFR-LSK-005** | Review budget: single PR under pre-authorised `size:exception` against 1000-line budget, forecast 400–700 lines. |

## Explicit non-requirements

- **No edit** to any substrate file in `R-LSK-004`. AG-04/05/06 rule engine, registry, descriptor vocabulary, tests stay untouched.
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit.
- **Tools, hooks, errors beyond typed pass-through, permission, retry, context-check, cost events** — AG-08…AG-18.
- **Value-form `Harness`** — AG-13. AG-07 ships the function form only (D1).
- **Multi-turn state** — AG-21. AG-07 is stateless across calls (per `R-LSK-002`).
- **Test-convenience wrappers** in `backend/agent/src/agenttest` — AG-23. AG-07 uses `agenttest.Script` directly (per D4).
- **CI** — every gate runs when a human runs `make test` in `backend/agent/`.

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active (`openspec/AGENTS.md`).

- Both leaves are behavior, so RED-first.
- **`R-LSK-001` closes only on its recorded bites** (`S-LSK-003a`, `S-LSK-003b`): the reconstruction helper bites RED twice BEFORE `S-LSK-003` is GREEN. Mirrors AG-05 `S-AMT-071`/`S-AMT-072` (`reconstruction_test.go:180-277`).
- **No new event kinds** registered by AG-07 — the every-kind-constructible guard stays at 25.
- **Scope-fence remains at 25.** AG-07 extends to 26+ only via `R-LSK-004`'s 4th consecutive extensibility demonstration, not by editing the registry.
- **Strict TDD skill gap** — `openspec/AGENTS.md` `## Strict TDD is on` is the inline fallback.

## Acceptance criteria

1. Every `S-LSK-001`…`S-LSK-007` has recorded evidence.
2. `cd backend/agent && make test`, `make lint` (after `cache clean`), `make build`, and `make vuln-check` are all green.
3. `backend/agent/go.mod` and `go.sum` byte-unchanged.
4. The 21 substrate files in `R-LSK-004` are byte-unchanged.
5. `loop.go` line coverage ≥ 80% (AG-04 W8).
6. The every-kind-constructible guard still passes at 25 kinds; AG-03's two boundary guards pass with zero changes.
7. The 5 charter `AG-07.1`/`AG-07.2` Gherkin scenarios are covered; none reduced.
8. Both bites (`S-LSK-003a`, `S-LSK-003b`) RED-recorded with failing output in `apply-progress.md`.
9. `5 charter → 7 spec + 2 bites = 9 total` scenario count stated identically with the proposal, tasks, apply-progress, and verify-report.

## Traceability

| Requirement | Charter node | Decisions cited | Charter scenario (`0003`) |
|---|---|---|---|
| `R-LSK-001` | AG-07.1 | D1, D2, D3, D5 | `:794-805` |
| `R-LSK-002` | AG-07.2 | D1, U3 path-a | `:821-825` |
| `R-LSK-003` | AG-07.2 | D4 | `:827-829` |
| `R-LSK-004` | AG-07 cross-cut | — | substrate preservation |
| `R-LSK-005` | AG-07 cross-cut | — | AG-04 W8 carry |

All 5 charter Gherkin scenarios are represented; none is reduced. Scenario count stated identically with the proposal (`5 charter → 7 spec + 2 bites = 9 total`).
