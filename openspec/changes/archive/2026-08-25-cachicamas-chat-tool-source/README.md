# Archive — `cachicamas-chat-tool-source` (CH-09)

This folder archives the CH-09 SDD change — Wave 3, 10 of 12 milestones shipped.

| File | What it carries |
|------|----------------|
| `proposal.md` | Intent, scope, decisions, success criteria. Source: engram `sdd/cachicamas-chat-tool-source/proposal` (#3959). |
| `explore.md` | Charter-vs-current-state inventory and approach comparison. Source: engram `sdd/cachicamas-chat-tool-source/explore` (#3952). |
| `decisions.md` | Five product-decision locks (port shape, first tool, wire shape, rendering, persistence). Source: engram `sdd/cachicamas-chat-tool-source/decisions` (#3947). |
| `design.md` | Technical approach — chat.ToolSource port + current_time tool + 4 new WireEvent variants + Exchange widening + frontend delta. Source: engram `sdd/cachicamas-chat-tool-source/design` (#3965). |
| `tasks.md` | 8 work units across 5 leaves + 1 guard. Source: engram `sdd/cachicamas-chat-tool-source/tasks` (#3967). |
| `apply-progress.md` | Apply-phase evidence — RED-GREEN-REFACTOR per WU, focused test commands and results, files changed. Mirror of engram `…/apply-progress` (#3971). |
| `specs/cachicamas-chat-tool-source/spec.md` | Change-local copy of the promoted spec. |
| `verify-report.md` | Produced by `sdd-verify` after this PR lands (NOT included in this PR per orchestrator). |
| `archive-report.md` | Produced by `sdd-archive` after verify closes (NOT included in this PR per orchestrator). |

## What shipped

CH-09.1 + CH-09.2 + CH-09.3 + CH-09.4 + CH-09.5 — Offer tools through a tool-source port.

- `chat.ToolSource` is the chat-owned wrapper-direction port over `agent.Registry`; `Config.ToolSource` is required with typed `ErrNilToolSource`.
- `chat.CurrentTimeTool` is the first tool — RFC3339 system clock with injectable `NowFunc`, `EffectClass == EffectClassRead`.
- Four new `WireEvent` variants (`ToolCallStart`, `ToolResult`, plus the reserved-but-unused `ToolCallDelta` / `ToolCallEnd` for future progress-bearing tools) extend the chat wire shape.
- `Exchange` widens additively with `[]ToolCallRecord` + `[]ToolResultRecord`; forward-only sibling tables `chat_tool_calls` + `chat_tool_results` key off `(participant_id, exchange_position, position)`.
- Frontend `parseTranscript` switch covers all 9 event names; `useChatStream` accumulates tool entries (state "running" → "done" / "failed" with typed category); `exchangesToEntries` renders tool entries from reload DTOs after the assistant said entry.
- `TestChat_SubstrateUntouched` runs inside `make test` and guards the 10-file substrate list under `backend/agent/src/agent/`.
- F-CHT-9.1 spec defect resolved at design phase: `state: "complete"` → `state: "done"` aligned with `mock/chat.ts:42` closed union (no code ripple).
- No new top-level Go deps; no file under `backend/agent/src/agent/` modified.

## Substrate preservation

`git diff --stat main..HEAD -- backend/agent/src/agent/` is empty after CH-09 lands (NFR-TLS-003 + NFR-CTS-003).

## Re-billing question (D-8)

v1's static `chat.FromAgentRegistry(agent.NewMapRegistry(...))` is constructed once per conversation; the cache prefix is byte-stable per conversation (V-REQ-14 / `ai.ToolSet` deterministic ordering). v2 dynamic-source surfaces (MCP, where `Resolve` may return different tools between turns) are deferred to a future spec per doc 0005 register row 5 + ADR 0009 § D4.

## Acceptance

`0005:931` (verbatim): *"Given a conversation whose tool source offers one tool, When the model calls it, Then the call and its result appear on the participant's stream and the turn continues with the result in the transcript."*

CH-09 meets this acceptance: `cd backend/agent && make test` (uncached, race-clean), `make lint`, `make build/chat`, `pnpm --filter @cachicamas/frontend test:ci`, `pnpm lint`, `pnpm build.types`, `INTEGRATION=1 make test` for S-CCS-021 — all green. 9 commits on `feat/chat-tool-source-ch09`.

Doc 0005 status bumped to 10/12; CH-09.1..5 ticked in the completion-checklist. CH-10 (permission) is unblocked.

## Next phase

`sdd-verify` validates the implementation against the 28 scenarios (S-CTS-001..024 + S-CTT-001..003 + S-CCS-019..022); `sdd-archive` produces the verify-report + archive-report.