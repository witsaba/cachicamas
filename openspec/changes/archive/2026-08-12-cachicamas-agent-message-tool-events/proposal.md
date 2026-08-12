# Proposal — Add message and tool execution event families

> **Slug**: `cachicamas-agent-message-tool-events` · **Milestone**: AG-05 (Layer 2, Wave 1) — doc 0003 lines 518–600
> **Traces to**: R-04 (full close), R-13 (ordinal) — `0003:2203`, `0003:2211` · **Depends on**: AG-04 (merged `967d043f`) · **Parallel with**: AG-06 · **Blocks**: AG-07, AG-09, AG-10, AG-11
> **Delivery**: single PR + `size:exception` pre-authorized this session (forecast 1500–2200 lines, ~1.5–2.2× the 1000-line budget — same precedent as AG-04 PR #163).
> **Spec identifier prefix**: `R-AMT-` / `S-AMT-` (matches slug; distinct from `R-AEV-`/`S-AEV-` and `R-AGE-`/`S-AGE-`).

## Intent

Layer 2 names 8 v1 families in doc 0001 § 4.3. AG-04 shipped 2 (run + turn). AG-05 ships the next 2 — **message lifecycle** and **tool execution** — the high-volume branches a live loop emits per turn. **First kind-set to exercise `PlacementTurn`** (the seam `event_descriptor.go:78-85` reserved at AG-04.3). AG-05.3's reconstruction property is the literal expression of `L2C-04`'s "a log can reconstruct it" criterion (`0003:418`). **Why now**: AG-09 scheduler, AG-11 execution, and doc 0004's built-in tools all need the registry populated; AG-07 loops over the kind set; doc 0004's tool split implements AG-05.2's three `tool_end_*` kinds.

## Scope

**In scope.** Register 11 new kinds (6 message + 5 tool, A1 locked by user): `message_start_text`, `message_delta_text`, `message_end_text`, `message_start_reasoning`, `message_delta_reasoning`, `message_end_reasoning`, `tool_start`, `tool_progress`, `tool_end_success`, `tool_end_result_failure`, `tool_end_execution_failure`. Two new family files. AG-05.3 reconstruction property test (load-bearing). AG-04.4's `S-AEV-090` retightened from "exactly 4" to "exactly 11". New spec `openspec/specs/agent-message-tool-events/spec.md` promoted (12–18 scenarios). Delta spec on `agent-event-envelope`.

**Out of scope.** Live-loop emission (AG-07, AG-09). Permission events around tools (AG-06, AG-10). Tool execution contract (AG-09.1). New top-level Go deps. Edits to `event_descriptor.go` / `stream_check.go` / `failure.go` — AG-04's rule engine stays untouched; that's the AG-05 bet.

## Capabilities

**New**: `agent-message-tool-events` — `openspec/specs/agent-message-tool-events/spec.md` (full spec, 12–18 scenarios, `R-AMT-0NN` / `S-AMT-0NN`).

**Modified**: `agent-event-envelope` — delta spec at `openspec/changes/cachicamas-agent-message-tool-events/specs/agent-event-envelope/spec.md` for the kind registry extension. Substrate remains source of truth for envelope invariants (`R-AEV-001`…`011`).

## Approach

Follow AD-1's six-step "adding a kind" procedure (`event_descriptor.go:13-31`); **call `Terminal bool` out as a separate guard step** so the W3 latent trap cannot reopen (c203f25c closed it for AG-04's 4 kinds; AG-05's 11 all declare `Terminal: false`). All 11 register `Placement: PlacementTurn` and `BracketRole: BracketRoleNone`; `CardinalityAny` zero-value. `stream_check.go:161` rejects a `message_start` outside an open turn with zero code edit. `S-AGV-019` reuses `ai.MessageID` / `ai.ToolCallID`. **Tool call ordinal is a payload field, not envelope** (R-13 traces to doc 0002 AI-30 payload-side). **`make vuln-check` runs explicitly** at apply/verify (not in `make all`). **`golangci-lint cache clean`** before any lint gate (cite `6c821c0a` precedent). No new top-level Go deps. No `go.mod`/`go.sum` edit. No AG-03 guard edit. No `event_descriptor.go`/`stream_check.go`/`failure.go` edit.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/message_text.go` | New | `MessageStartText`/`DeltaText`/`EndText` payload + constructors (3 text kinds) |
| `backend/agent/src/agent/message_reasoning.go` | New | `MessageStartReasoning`/`DeltaReasoning`/`EndReasoning` payload + constructors (3 reasoning kinds) |
| `backend/agent/src/agent/message_text_events.go` | New | Kind constants + `EventDescriptor` rows for the 3 text kinds |
| `backend/agent/src/agent/message_reasoning_events.go` | New | Kind constants + `EventDescriptor` rows for the 3 reasoning kinds |
| `backend/agent/src/agent/tool_event.go` | New | `ToolStart`/`Progress`/`End*` payload types (`ToolEnd` triples by `ToolOutcome`) |
| `backend/agent/src/agent/tool_events.go` | New | Kind constants + `EventDescriptor` rows for 5 tool kinds |
| `backend/agent/src/agent/reconstruction_test.go` | New (test) | AG-05.3 load-bearing property test (helper bite-tested first) |
| `backend/agent/src/agent/event.go` | Modified | Extend `EventKind` const block + `eventRegistry` table (11 new rows) |
| `backend/agent/src/agent/event_descriptor.go` | Modified | Restate AD-1 six-step procedure with `Terminal` explicit |
| `backend/agent/src/agent/event_registry_test.go` | Modified | Witness table 4 → 11; scope-fence `S-AEV-090` retightened |
| `backend/agent/src/agent/doc_contract_guard_test.go` | Modified | Scope-fence count updated to 11 (same commit as new kinds) |
| `backend/agent/src/agent/agent_test_helpers_test.go` | Modified | Reuse `requireViolationPosition`; add AG-05 helper |
| `backend/agent/src/agent/doc.go` | Modified | Carry `L2C-05` prose for message/tool family semantics alongside `L2C-04` |
| `openspec/specs/agent-message-tool-events/spec.md` | New | `R-AMT-`/`S-AMT-` 12–18 scenarios |
| `openspec/changes/cachicamas-agent-message-tool-events/specs/agent-event-envelope/spec.md` | New | Delta spec for kind registry extension |
| `backend/agent/{go.mod,go.sum}` | UNCHANGED | No new deps (per `openspec/AGENTS.md` rule 5) |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| W3 latent trap re-opens (new kind declares `Terminal: true` with `BracketRoleNone`) | Low | AD-1 six-step restated with `Terminal` explicit; AG-04.4's `S-AEV-092` experiment pattern |
| Reconstruction helper is vacuous (AG-05.3 failure mode) | Medium | Helper bite-tested first (drop a delta; expect non-equality) BEFORE the property test is GREEN |
| Scenario count drift (W9: 3-artifact-deep AG-04 propagation) | Medium | State "7 charter scenarios in 3 leaves, ~12–18 spec scenarios after expansion" identically in proposal, tasks, apply-progress |
| `S-AEV-090` scope-fence not retightened (every-kind-constructible invariant bypassed) | High | Update count 4 → 11 in same commit as new kinds |
| Lint cache re-introduces a phantom finding (AG-04 gotcha) | Medium | `golangci-lint cache clean` before any lint gate; cite `6c821c0a` precedent |
| `make all` does not include `vuln-check` (Engram `obs #2944`) | High | Run `make vuln-check` explicitly during apply/verify |

## Rollback Plan

Single PR. Revert the AG-05 merge commit. AG-05's 11 kinds removed; the doc-matrix guard's scope-fence reverts to "exactly 4"; AG-04's invariant 1 co-closure reverts to AG-04.3's pin alone (`0003:2203`). Cite AG-04's archive pattern (`2026-08-12-cachicamas-agent-event-envelope/`). No persisted state, no schema, no migration, no `go.mod`/`go.sum` change to unwind. Chaining is not safe (AG-05.3's reconstruction test references both message and tool kinds in one PR).

## Dependencies

None. No new top-level Go deps. No `go.mod` or `go.sum` edit (per `openspec/AGENTS.md` rule 5). Toolchain unchanged: Go 1.26.3, `golangci-lint v2.9.0`. AG-04 (merged `967d043f`) is the only direct dependency.

## Success Criteria

- [ ] 11 new kinds registered (6 message + 5 tool), constructible, validated, guarded
- [ ] `agent-message-tool-events` spec promoted with 12–18 scenarios, all Given/When/Then
- [ ] AG-05.3 reconstruction property test GREEN (helper bite-tested first RED)
- [ ] `S-AEV-090` scope-fence retightened to "exactly 11"
- [ ] `make test`, `make lint`, `make build`, `make vuln-check` all clean
- [ ] No edits to `event_descriptor.go` / `stream_check.go` / `failure.go` (the AG-05 bet)
- [ ] No new dependencies (`go.mod` / `go.sum` byte-unchanged)
- [ ] Single PR merged with `size:exception` documented
- [ ] 7 charter Gherkin scenarios from `0003:543-583` covered; post-expansion ~12–18 `S-AMT-` scenarios

## Notes for the Following Phases

- **`sdd-spec`**: write `specs/agent-message-tool-events/spec.md` + the `agent-event-envelope` delta spec. Restate 7 charter scenarios as Given/When/Then; do not reduce them. Witness-table extension and the reconstruction bite are spec requirements, not implementation details. Spec must restate the partial-closure map (invariant 1 co-closed by AG-04.3 + AG-05.1).
- **`sdd-design`**: A1 (6 message kinds), A2 (payload ordinal), A3 (per-family file split), A4 (6 symmetric reasoning+text), A5 (single PR + exception) are **locked at proposal** — design cannot silently re-open them. Mechanics (file naming, helper signatures) are design's call.
- **`sdd-tasks`**: three phases matching the node graph — AG-05.1, AG-05.2, then AG-05.3 (depends on both). Single PR, three commits. State scenario count ("7 charter → ~12–18 spec") identically in tasks and apply-progress. Restate the four `sdd-phase-common.md` § E guard lines (`Decision needed: No`, `Chained PRs: No`, `400-line budget risk: High`, `Forecast: 1500–2200`).
