# Spec — The one-turn walking skeleton (`agent-loop-skeleton`)

> **Change**: `cachicamas-agent-loop-skeleton` · **AG-07** (Layer 2, Wave 2 opening) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-07--build-the-one-turn-walking-skeleton), `0003:771-832`
> **Nodes**: AG-07.1 `[leaf]` (one text turn) · AG-07.2 `[leaf]` (statelessness + reasoning pass-through)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml`. Every scenario independently verifiable.
> **IDs**: `R-LSK-0NN` / `S-LSK-0NN`. Append-only. Distinct from `R-AEV-`/`S-AEV-`, `R-AMT-`/`S-AMT-`, `R-APE-`/`S-APE-`, `R-AGE-`/`S-AGE-`.
> **Scenario count** (AG-04 W9): **5 charter → 7 spec + 2 bites = 9 total** in 5 requirements (+ 1 cross-cut `S-LSK-008` + 1 bite `S-LSK-008a` added by AG-09, + 1 cross-cut `S-LSK-010` added by AG-10, total **12**).

## Coverage

| Charter | Requirements | Spec | Bites |
|---|---|---|---|
| **5 of 5** | 5 (`R-LSK-001`–`005`) + 1 added (`R-LSK-006` by AG-09) | **9** | **3** (`S-LSK-003a`, `S-LSK-003b`, `S-LSK-008a`) |

Charter → spec: AG-07.1#1 → `R-LSK-001`/`S-LSK-001`; AG-07.1#2 → `R-LSK-001`/`S-LSK-002`; AG-07.1#3 → `R-LSK-001`/`S-LSK-003` + bites; AG-07.2#4 → `R-LSK-002`/`S-LSK-004`; AG-07.2#5 → `R-LSK-003`/`S-LSK-005`. AG-09 adds: `R-LSK-006` (one cycle per turn, S-LSK-008 + S-LSK-008a). Cross-cuts → `R-LSK-004` (substrate, AG-07), `R-LSK-005` (coverage, AG-07), `R-LSK-006` (one cycle per turn, AG-09).

## Purpose

The first Layer 2 milestone where a live loop produces events. Per doc 0003:773: "the single most important node in this document: the first time Layer 1 and Layer 2 meet." AG-07 owns the producer side of AG-01's carrier decision (D2a) and gives AG-08/AG-09/AG-11 the surface they wrap. Bites defend the AG-05.1 reconstruction property against the AG-05 W1 vacuous-helper failure mode. AG-09 adds the wording-trap boundary with AG-13: the loop schedules, it does not iterate.

## Requirements

### R-LSK-001 — Loop surface: single-turn function form (D1)

The system SHALL expose `func Turn(ctx context.Context, provider ai.ModelProvider, system string, transcript []ai.Message, opts TurnOptions, sink chan<- *Event) (msg ai.Message, finish ai.FinishReason, err error)` as the only public surface for one assistant turn (per D1, D2, D3). AG-13 introduces a value-form `Harness` that wraps `Turn` without changing its signature. `TurnOptions` carries a `Tools map[string]Tool` field for AG-09 (non-breaking zero-value extension; nil `Tools` = the scheduler returns typed `ExecutionFailure` results in their ordinal slots, consistent with R-TLS-009 "one bad tool does not abort the turn"). `TurnOptions` also carries the AG-08 `PreRequestHook` field (nil = identity default). `TurnOptions` further carries the AG-10 `PermissionPolicy PermissionPolicy` field (non-breaking zero-value extension). A **nil** `PermissionPolicy` is the identity default: the scheduler bypasses the permission gate, every call is treated as an immediate allow, no permission event is emitted, and the turn behaves exactly as it did before AG-10. A non-nil policy MUST be forwarded byte-exact to `Schedule` and consulted for every scheduled call. `Turn` MUST NOT expose the scheduler or any wake handle at this milestone — `Scheduler.WakeParked` is unreachable from a `Turn` caller, and wiring the upward-path wake is AG-13's scope.

The **signature is unchanged by AG-11**, but the returned-value contract on the mid-stream-fatal path changes: when the provider stream ends in a terminal error, `Turn` MUST return the assistant message reconstructed from the content accumulated before the failure together with the non-nil error, rather than the zero `ai.Message{}` (`loop.go:275`). On that same path `Turn` MUST have emitted `turn_end`/`run_end` before closing the sink, per `R-ATT-005`. On every successful path `Turn` MUST return a `finish` that is a member of `ai.FinishReason`'s vocabulary, including the provider-closed-without-`Completion` path, where `ai.FinishReasonStop` remains the documented fallback.

(Previously: on the mid-stream-fatal path `Turn` returned `ai.Message{}, 0, turn.fatal` with no closing events emitted, and the returned message contract was stated only for successful turns.)

#### Scenarios

- **S-LSK-001** — AG-07.1 walking skeleton. Given a text response scripted on the fake provider (one `ai.MessageStart`/`ai.MessageDelta`/`ai.MessageEnd` text stream + one `ai.Completion`), when `Turn(ctx, provider, system, transcript, opts, sink)` runs, then the consumer (draining `sink`) observes in order: `run_start`, `turn_start`, `message_start_text`, the deltas in order, `message_end_text`, `turn_end` (`TurnOutcomeFinished`), `run_end`; the sink is closed after `run_end`; the function returns `(msg, finish, nil)` where `finish` is the provider's `ai.FinishReason` (per D3). Unchanged by AG-11: that turn ends in `FinishReasonStop`, which maps to `TurnOutcomeFinished` under `R-ATT-001`.
- **S-LSK-002** — AG-07.1 provider stream drained and caller's context respected. Given a turn in progress with a non-cancelled `ctx`, when the provider stream reaches its terminal event (`ai.Completion` or typed error), then the loop has drained the provider's channel fully (no goroutine leak), and the loop has passed `ctx` unchanged to `provider.Stream(ctx, req)` (per D5), and the consumer's drain unblocks without blocking on a stranded producer.
- **S-LSK-003** — AG-07.1 one source of truth for the assistant message. Given a completed turn, when the caller reads the loop's returned `msg` AND a consumer reconstructs an `ai.Message` from the emitted deltas via the AG-05.3 helper (`reconstruction_test.go:54-114`), then the two `ai.Message` values are equal as Layer 1 message values (fragment-for-fragment byte-equal).
- **S-LSK-003a** — **(bite)** Given a complete turn with three text deltas, when the loop's emitted event sequence is REWRITTEN to drop the middle delta, then the reconstructed message differs from the loop's returned `msg` — proving the property is non-vacuous. RED-recorded BEFORE `S-LSK-003` is GREEN.
- **S-LSK-003b** — **(bite)** Given a complete turn with three text deltas, when the loop's emitted event sequence is REWRITTEN to double the middle delta, then the reconstructed message differs from the loop's returned `msg`. RED-recorded BEFORE `S-LSK-003` is GREEN.
- **S-LSK-009** — AG-09 wire-up: `Turn` consumes AI-18 tool-call events and calls `Schedule`. Given a `TurnOptions{Tools: map[string]Tool{...}}` with one registered read-class tool and a provider that streams one `ToolCallStart` / `ToolCallDelta` / `ToolCallEnd` triplet followed by a `Completion{FinishReason: FinishReasonToolCalls}`, when `Turn` runs, then the loop converts the AI-18 events into a `[]ScheduledCall`, invokes `Schedule` exactly once between `provider.Stream` close and `finalize`, and emits the AG-05.2 tool events (`ToolStart`, `ToolEnd*`) on `sink` in rejoin order — proving the AG-09 wire-up.
- **S-LSK-010** — AG-10 wire-up: `TurnOptions.PermissionPolicy` reaches the gate, and nil stays a bypass. Given a `TurnOptions{Tools: ..., PermissionPolicy: p}` where `p` defers one call, denies a second, and modifies the arguments of a third, when `Turn` runs against a provider streaming those three tool calls, then the deferred call resolves into a typed abort once the run context is cancelled, the denied call's ordinal slot carries `ExecutionFailure` with a typed `*Failure`, the modified call's `tool_start.Arguments()` byte-equals `decision_made.ModifiedArguments()` and the tool records those same bytes, and `Turn` returns without a Go error; and given the identical setup with `PermissionPolicy: nil`, when `Turn` runs, then no permission event appears on `sink` and every call executes as it did before AG-10. Verified by `TestTurn_PermissionPolicy_E2E_DeferDenyModify` and `TestNoOpPermissionPolicy_AllowsEverySynchronously`.
- **S-LSK-011** — AG-11 return contract on the fatal path. Given a provider stream scripted to deliver text content and then a terminal mid-stream error, when `Turn` returns, then the returned `msg` carries that content byte-for-byte rather than the zero `ai.Message{}`, the returned `err` is non-nil, and `Turn`'s exported signature is unchanged (asserted by the package's documented-contract guard). Cross-referenced to `R-ATT-007` / `S-ATT-009`.

### R-LSK-002 — Statelessness: two sequential turns share nothing

The system SHALL treat `Turn` as stateless across calls; the second call's event sequence SHALL be independent of the first's, with fresh per-stream ordering starting at 1 (per D1, U3 path-a).

#### Scenarios

- **S-LSK-004** — AG-07.2 two sequential turns share nothing. Given one `Turn` function value that has already run a turn, when a second turn runs via a second `Turn(...)` invocation (fresh slices, fresh `opts`, fresh `sink`, fresh `provider` script), then the second turn's emitted events carry fresh per-stream ordering starting at 1, and the second turn's events do not depend on any state from the first turn (no closure captures over first-turn results, no shared `LaneStamper`).

### R-LSK-003 — Reasoning pass-through: byte-exact round-trip token

The system SHALL distinguish reasoning from text events by kind (`message_start_reasoning`/`message_delta_reasoning`/`message_end_reasoning` vs `message_start_text`/`message_delta_text`/`message_end_text` per AG-05.1), and SHALL preserve the reasoning round-trip token byte-exact into the assistant message.

#### Scenarios

- **S-LSK-005** — AG-07.2 reasoning flows through distinguished, byte-exact. Given a scripted response interleaving reasoning and text deltas (reasoning → text → reasoning → text per D4), with a non-empty `[]byte` reasoning round-trip token on each reasoning end, when the loop re-emits it via `Turn(...)`, then reasoning and text are emitted as separate bracket kinds, and the assistant message's reasoning-content round-trip token is byte-equal to the script's token, and the event order matches the script's emit calls.

### R-LSK-004 — Substrate untouched, with AG-11's recorded exact-filename release

The system SHALL NOT modify any of: `event.go`, `event_descriptor.go`, `stream_check.go`, `sequence.go`, `run_events.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `event_registry_test.go`, `doc.go`, `doc_contract_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, `go.mod`, `go.sum`.

`turn_events.go` and `failure.go` were members of this list through AG-07…AG-10 and are **released for AG-11 only**, for a structural reason recorded rather than assumed: `TurnOutcome`'s `iota` const block (`turn_events.go:74-89`), its `turnOutcomeLimit` bound-check inside `TurnEnd.validate` (`:120`) and its `String()` switch (`:93-104`) are local to `turn_events.go`, so a member declared elsewhere would be rejected by the validator and would render as the `turnoutcome(N)` placeholder; and `agent.Failure`'s accessor set is local to `failure.go` (`:44`, `:54`, `:64`, `:75`). AG-11's edits to these two files MUST be confined to `R-ATT-001` (outcome members + their `String()` forms) and `R-ATT-006` (`PartialOutput()`); `TurnEnd.validate`'s failure-iff-aborted rule (`:123-128`) and `NewTurnEnd`'s signature MUST NOT change. Every other file above remains forbidden, and the release does not extend to any milestone after AG-11 without its own recorded delta.

AG-11 is the first milestone to modify a **pre-existing** substrate file rather than only appending new ones. The substrate guards' allowlists — `filterOutLoopFiles` (`loop_test.go:831-871`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907-943`) — SHALL therefore widen by **exact filename suffixes only**: no wildcard, no prefix match, no directory-level relaxation, and the two filters SHALL carry the identical entry set. The widened set SHALL name exactly `turn_events.go`, `failure.go`, and one entry per test file AG-11 introduces (the concrete filenames are fixed by this change's `tasks.md`); any file not so named MUST still fail the guards.

(Previously: `turn_events.go` and `failure.go` were listed as forbidden alongside the other 20 files, and every prior widening added only files a milestone had newly created, so no rule governed widening for a genuinely modified substrate file.)

#### Scenarios

- **S-LSK-006** — Given main at `8420b2c4` (post-AG-06 merge), when the AG-07 changes are compared against main, then `git diff main -- backend/agent/src/agent/` (excluding `loop.go`, `loop_test.go`) shows zero lines changed, and `git diff main -- backend/agent/go.mod backend/agent/go.sum` is empty, and the every-kind-constructible guard (`event_registry_test.go:54-251`) still passes at 25 kinds (AG-07 adds zero), and AG-03's boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) still pass — AG-07's new code uses only stdlib + `src/ai` + `src/agenttest` (tests).
- **S-LSK-012** — AG-11 release is bounded and exact. Given the merge base of the AG-11 branch with `origin/main`, when `git diff` is taken over `backend/agent/src/agent/`, then the only pre-existing non-test files that differ are `turn_events.go`, `failure.go` and `loop.go`; every file still named forbidden above is byte-unchanged; `go.mod`/`go.sum` are byte-unchanged; `NewTurnEnd`'s signature and `TurnEnd.validate`'s failure-iff-aborted rule are unchanged; the two filter functions carry an identical set of exact-filename entries containing no wildcard, prefix, or directory pattern; and both substrate guards pass. Cross-referenced to `R-ATT-009` / `S-ATT-011`.

### R-LSK-005 — Test coverage on `loop.go`

The system SHALL achieve ≥ 80% line coverage on `backend/agent/src/agent/loop.go` per `make test` race-gated run, satisfying AG-04 W8. The coverage SHALL include the AG-09 wire-up: the path from `provider.Stream` close through `Schedule` invocation through tool-event emission SHALL be covered; the "loop schedules; does not iterate" invariant (`S-LSK-008`) SHALL be covered by a test that asserts `Schedule` is invoked exactly once per `Turn` even when tool results would warrant a follow-up model call.

#### Scenarios

- **S-LSK-007** — Given `make test` green in `backend/agent/`, when the coverage report is read for `backend/agent/src/agent/loop.go`, then the line coverage is ≥ 80%, and `loop_tool_dispatch_test.go` is part of the covered surface.

### R-LSK-006 — One cycle per turn (AG-09 wording-trap boundary with AG-13)

The loop SHALL schedule at most one cycle of `model → tools → finalize` per `Turn` invocation. Iteration across cycles (`model → tools → model`) is **AG-13's `Harness` contract**; AG-09 MUST NOT loop the cycle within `Turn`. This is the wording trap from `0003:107-112` — the loop schedules, it does not iterate. The scheduler's `Schedule` function is the seam: callable from `Turn` (AG-09) or from `Harness` (AG-13), but `Turn` MUST call it at most once per invocation.

#### Scenarios

- **S-LSK-008** — One cycle per turn. Given a provider that streams one round of tool-call events (`ToolCallStart` / `Delta` / `End`) followed by a `Completion` with `FinishReasonToolCalls`, when `Turn` runs, then `Schedule` is invoked exactly once per `Turn` (asserted via a per-tool invocation counter) and `Turn` returns without re-entering `provider.Stream` — even if the tool results would warrant a follow-up model call.
- **S-LSK-008a** — **(bite)** RED-first. Given a `Turn` whose post-`Schedule` path erroneously re-invokes `Schedule`, when the cycle-count scenario runs, then the per-tool invocation counter reports > 1 — proves the one-cycle invariant is non-vacuous. RED-recorded BEFORE `S-LSK-008` is GREEN.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-LSK-001** | External-package verifiability: every scenario verifiable by `cd backend/agent && make test`. Every behavioral test in `package agent_test` or another external package. |
| **NFR-LSK-002** | Determinism and race cleanliness: every test MUST be deterministic, hermetic, and pass under `-race`. |
| **NFR-LSK-003** | Substrate byte-unchanged: the 21 files in `R-LSK-004` are byte-unchanged. 6th consecutive extensibility demonstration. |
| **NFR-LSK-004** | Boundary guards stay green untouched: AG-03's `import_boundary_test.go` and `ambient_authority_test.go` pass with zero changes. |
| **NFR-LSK-005** | Review budget: single PR under pre-authorised `size:exception` against 1000-line budget, forecast 400–700 lines. |

## Explicit non-requirements

- **No edit** to any substrate file in `R-LSK-004`. AG-04/05/06 rule engine, registry, descriptor vocabulary, tests stay untouched.
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit.
- **Tools, hooks, errors beyond typed pass-through, permission, retry, context-check, cost events** — AG-08…AG-18.
- **Value-form `Harness`** — AG-13. AG-07 ships the function form only (D1).
- **Multi-turn state** — AG-21. AG-07 is stateless across calls (per `R-LSK-002`).
- **Iteration of the model ↔ tools ↔ model cycle** — AG-13. AG-09 ships ONE cycle; AG-13 iterates.
- **Test-convenience wrappers** in `backend/agent/src/agenttest` — AG-23. AG-07 uses `agenttest.Script` directly (per D4).
- **CI** — every gate runs when a human runs `make test` in `backend/agent/`.

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active (`openspec/AGENTS.md`).

- Both leaves are behavior, so RED-first.
- **`R-LSK-001` closes only on its recorded bites** (`S-LSK-003a`, `S-LSK-003b`): the reconstruction helper bites RED twice BEFORE `S-LSK-003` is GREEN. Mirrors AG-05 `S-AMT-071`/`S-AMT-072` (`reconstruction_test.go:180-277`).
- **AG-09 adds `R-LSK-006`** with bite `S-LSK-008a`: the one-cycle-per-turn invariant bites RED with an invocation-counter scenario BEFORE `S-LSK-008` is GREEN.
- **No new event kinds** registered by AG-07 — the every-kind-constructible guard stays at 25.
- **AG-09 adds zero event kinds** — the every-kind-constructible guard stays at 25.
- **Scope-fence remains at 25.** AG-07 + AG-08 + AG-09 extend via `R-LSK-004`'s 6th consecutive extensibility demonstration, not by editing the registry.
- **Strict TDD skill gap** — `openspec/AGENTS.md` `## Strict TDD is on` is the inline fallback.

## Acceptance criteria

1. Every `S-LSK-001`…`S-LSK-009` has recorded evidence.
2. `cd backend/agent && make test`, `make lint` (after `cache clean`), `make build`, and `make vuln-check` are all green.
3. `backend/agent/go.mod` and `go.sum` byte-unchanged.
4. The 21 substrate files in `R-LSK-004` are byte-unchanged.
5. `loop.go` line coverage ≥ 80% (AG-04 W8).
6. The every-kind-constructible guard still passes at 25 kinds; AG-03's two boundary guards pass with zero changes.
7. The 5 charter `AG-07.1`/`AG-07.2` Gherkin scenarios are covered; none reduced.
8. All 3 bites (`S-LSK-003a`, `S-LSK-003b`, `S-LSK-008a`) RED-recorded with failing output in `apply-progress.md`.
9. `5 charter → 7 spec + 2 bites = 9 total` (AG-07) + 1 cross-cut `S-LSK-008` + 1 bite `S-LSK-008a` (AG-09) = **11 total** scenario count stated identically with the proposal, tasks, apply-progress, and verify-report.
10. The `Turn` does not iterate (one cycle per turn, `S-LSK-008`) — the wording trap from `0003:107-112` is asserted mechanically in `loop_tool_dispatch_test.go`.

## Traceability

| Requirement | Charter node | Decisions cited | Charter scenario (`0003`) |
|---|---|---|---|
| `R-LSK-001` | AG-07.1 | D1, D2, D3, D5 | `:794-805` |
| `R-LSK-002` | AG-07.2 | D1, U3 path-a | `:821-825` |
| `R-LSK-003` | AG-07.2 | D4 | `:827-829` |
| `R-LSK-004` | AG-07 cross-cut | — | substrate preservation |
| `R-LSK-005` | AG-07 cross-cut | — | AG-04 W8 carry |
| `R-LSK-006` | AG-09 (one-cycle) | D8 (AG-09) | wording trap `0003:107-112` |
| `S-LSK-009` | AG-09 (wire-up) | D8, D9a (AG-09) | new end-to-end dispatch |
