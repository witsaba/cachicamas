# Archive — `cachicamas-chat-tool-source` (CH-09)

> **Status**: CLOSED IN ENGRAM — verify-report PASS (24/24 scenarios covered; 23 COMPLIANT + 1 DEFERRED pending `INTEGRATION=1`).
> **Branch**: `feat/chat-tool-source-ch09` from `main @ 670cef7d`.
> **Final commit**: `45fb69b9` (T-08 part 2 amended — F-CHT-9.2 / F-CHT-9.3 spec wording recovery).
> **Date**: 2026-08-25.
> **10 of 12** milestones shipped in doc 0005.

This folder archives the CH-09 SDD change — Wave 3 of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md) (`0005:923-934`). Closes R-03 (the tool seam's real answer) and R-09's seam. Unblocks CH-10 (`cachicamas-chat-permission`).

## Artifacts

| Artifact | Path | Source |
|-----------|------|--------|
| Proposal | [`proposal.md`](./proposal.md) | Mirror of engram `#3959` (`sdd/cachicamas-chat-tool-source/proposal`) |
| Design | [`design.md`](./design.md) | Mirror of engram `#3965` (`sdd/cachicamas-chat-tool-source/design`) |
| Tasks | [`tasks.md`](./tasks.md) | Mirror of engram `#3967` (`sdd/cachicamas-chat-tool-source/tasks`) |
| Apply progress | [`apply-progress.md`](./apply-progress.md) | Mirror of engram `#3971` (post-recovery; 12 commits) |
| Verify report | [`verify-report.md`](./verify-report.md) | Mirror of engram `#3974` (RE-RUN PASS after recovery) |
| Archive report | [`archive-report.md`](./archive-report.md) | This phase's product |
| Spec (new) | [`specs/cachicamas-chat-tool-source/spec.md`](./specs/cachicamas-chat-tool-source/spec.md) | Mirror of engram `#3961` (verbatim transcription + F-CHT-9.1/9.2/9.3 amendments) |

## What shipped

**5 leaves + 1 guard** closed in CH-09.1..5 — Offer tools through a tool-source port.

- `chat.ToolSource` is the chat-owned wrapper-direction port over `agent.Registry`; `Config.ToolSource` is required with typed `ErrNilToolSource`. Chat depends on `agent.Registry` by interface, never by import-of-internals (D-1).
- `chat.CurrentTimeTool` is the first tool — RFC3339 system clock with injectable `NowFunc`, `EffectClass == EffectClassRead` (D-2).
- 4 new `WireEvent` variants — `ToolCallStart` and `ToolResult` emitted at v1; `ToolCallDelta` and `ToolCallEnd` reserved-but-unused for v2 dynamic-source surfaces (D-3 + D-6 collapse).
- 5-arm projector collapsing Layer 2's 5-event-per-call bracket model into 2 chat-side events per call (`ToolCallStart` + `ToolResult`); `EventKindToolProgress` dropped at the chat wire (NFR-CTS-002).
- `chat.Exchange` widens additively with `[]ToolCallRecord` + `[]ToolResultRecord`; forward-only sibling tables `chat_tool_calls` + `chat_tool_results` keyed by `(exchange_id, position)` (NFR-CCS-006).
- Frontend `parseTranscript` switch covers all 9 event names; `useChatStream` accumulates tool entries (`state "running" → "done" | "failed"` with typed category); `exchangesToEntries` renders tool entries from reload DTOs after the assistant said entry (D-4 + D-7).
- `TestChat_SubstrateUntouched` runs inside `make test` and guards the 10-file substrate list under `backend/agent/src/agent/`.
- `TestWire_FrameNameSet_IsClosed` (`S-CTS-024`) and `chat-api.spec.ts`'s `assertNever` probe lockstep the wire vocabulary with the `parseTranscript` switch.

## Notable decisions (D-1..D-8)

See [`proposal.md` § Resolved Decisions](./proposal.md) and [`archive-report.md` § 4 Locked Decisions](./archive-report.md).

1. **D-1** — Wrapper-direction port: `chat.ToolSource` wraps `agent.Registry` by interface (chat owns the port; composition-root factory closure at `cmd/chat/main.go` gains exactly one `ToolSource:` line).
2. **D-2** — First tool: `current_time` proves the seam (args validated, result rendered, error path closed, time injection preserves test determinism).
3. **D-3** — Wire shape: 4 new variants on the closed union (`ToolCallStart` / `ToolResult` emitted; `ToolCallDelta` / `ToolCallEnd` reserved), NOT new fields on existing variants.
4. **D-4** — Rendering: separate `kind: "tool"` transcript entry between turns; closed state union `"running" | "done" | "denied" | "failed"` from `mock/chat.ts:42`.
5. **D-5** — Persistence: persist both call AND result; `Exchange` widens additively; sibling-table migration is forward-only.
6. **D-6** — Wire collapse (deliberate surfacing): chat wire drops `EventKindToolProgress`; reserved variants make progress a single-line wire-shape extension.
7. **D-7** — REQ-7 widening language (deliberate surfacing): REQ-8..11 each carry explicit "new variants, not new fields" wording; REQ-7 verbatim text unmodified.
8. **D-8** — Re-billing question deferred (deliberate surfacing): v1's static source makes the question moot; v2 dynamic-source surface registered at ADR 0009 § D4 attachment point.

## Spec defects

**F-CHT-9.1, F-CHT-9.2, F-CHT-9.3** — all RESOLVED.

| ID | Status | Resolution |
|----|--------|------------|
| F-CHT-9.1 | RESOLVED at design (`#3965` §13) | `state: "complete"` → `state: "done"` alignment in `S-CTS-019` + `S-FCL-014`. Matches `mock/chat.ts:42` closed union. |
| F-CHT-9.2 | RESOLVED at recovery (`45fb69b9`) | `S-CTS-007` wording: Go 1.26 does NOT enforce type-switch exhaustiveness at compile time when a `default` arm exists — runtime panic from `default` branch is the binding invariant; `TestWire_AllNewVariants_SerialiseViaWireFrameName` enforces it by round-tripping every variant. |
| F-CHT-9.3 | RESOLVED at recovery (`45fb69b9`) | `S-CTS-022` wording: `finishReason: "tool_calls"` is "by construction" via assistantId-keyed delta accumulation at `use-chat-stream.ts:202-209`; not an explicit early-return gate. `S-FCL-017` mirror amended. |

See [`verify-report.md` § Spec defects status](./verify-report.md) for full evidence and `openspec/specs/cachicamas-chat-tool-source/spec.md:253-257` for the RESOLVED section in the spec itself.

## Substrate preservation (NFR-TLS-003 / NFR-CTS-003)

`git diff --stat main..HEAD -- backend/agent/src/agent/` is **empty** after CH-09 lands — verified at every WU and re-verified after both recovery commits. The 10-file substrate list survives byte-clean:

```
backend/agent/src/agent/event_descriptor.go
backend/agent/src/agent/stream_check.go
backend/agent/src/agent/failure.go
backend/agent/src/agent/sequence.go
backend/agent/src/agent/event.go
backend/agent/src/agent/go.mod
backend/agent/src/agent/go.sum
backend/agent/src/agent/Makefile
backend/agent/src/agent/.golangci.yml
backend/agent/src/agent/import_boundary_test.go
```

The chat-archetype substrate guard lives at `backend/agent/src/chat/store_substrate_test.go::TestChat_SubstrateUntouched` and runs inside `cd backend/agent && make test`. The wire-fragmentation guard (`S-CTS-024`) lives at `backend/agent/src/chat/wire_fragmentation_test.go` (Go runtime exhaustiveness probe) and `frontend/src/lib/chat-api.spec.ts` (`assertNever` TypeScript compile-time exhaustiveness probe). The AGENTS.md CH-09 pointer at `openspec/AGENTS.md:120-135` records this invariant.

## Spec/doc deliverables (in this PR)

- **NEW** `openspec/specs/cachicamas-chat-tool-source/spec.md` — R-CTS-001..008, NFR-CTS-001..003, S-CTS-001..024 + S-CTT-001..003.
- **ADDITIVE** `openspec/specs/chat-conversation-store/spec.md` — R-CCS-015/016, NFR-CCS-008, S-CCS-019..022.
- **ADDITIVE** `openspec/specs/frontend-chat-layer1/spec.md` — REQ-8..11 (each with explicit "new variants, not new fields" wording per D-7), S-FCL-012..017.
- `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` — status bumped to "10 of 12 shipped" at `:3`; CH-09.1..5 ticked at `:997-1001`.
- `openspec/AGENTS.md` — CH-09 pointer appended to the "Substrate preservation in `backend/agent`" section.

## Acceptance

Charter `0005:931` (verbatim): *"Given a conversation whose tool source offers one tool, When the model calls it, Then the call and its result appear on the participant's stream and the turn continues with the result in the transcript."*

CH-09 meets this acceptance per verify-report `#3974`:

- `cd backend/agent && go test -count=1 -race ./...` → 17/17 packages green, race-clean
- `cd backend/agent && make lint` → 0 issues
- `cd backend/agent && make build/chat` → produces `./bin/chat` (29,805,330 bytes)
- `pnpm --filter @cachicamas/frontend test:ci` → 594/594 tests pass across 159 suites
- `pnpm --filter @cachicamas/frontend lint` → 0 errors / 0 warnings
- `pnpm --filter @cachicamas/frontend build.types` → clean
- `git diff --stat main..HEAD -- backend/agent/src/agent/` → empty (NFR-TLS-003 / NFR-CTS-003 binding)
- `git diff main..HEAD -- backend/agent/go.mod backend/agent/go.sum` → empty (no new Go top-level deps)

24/24 scenarios covered (23 COMPLIANT + 1 DEFERRED). Doc 0005 status bumped to 10/12; CH-09.1..5 ticked. CH-10 (permission) is UNBLOCKED on PR merge.

## Re-billing question (D-8)

v1's static `chat.FromAgentRegistry(agent.NewMapRegistry(...))` is constructed once per conversation; the cache prefix is byte-stable per conversation (V-REQ-14 / `ai.ToolSet` deterministic ordering). v2 dynamic-source surfaces (MCP, where `chat.ToolSource.Resolve` may return different tools between turns) are deferred to a future spec per doc 0005 register row 5 + ADR 0009 § D4.

## Rollback

Single PR revert of `feat/chat-tool-source-ch09` (12 commits in reverse). The additive widening of `Exchange` is removable by dropping the 2 new struct fields and reverting `chat/migrations/0002_tool_records.sql` via mirror-DROP on a CH-07 forward-only migration. The additive widening of `ChatStreamEvent` is removable by reverting the 4 variant declarations and the 9-arm `parseTranscript` switch arms + `KNOWN_EVENTS` extension. If already merged, the amending path is mandatory (CH-00 `F-1`/`F-2`/`F-3` recorded-not-repaired pattern): a follow-up PR with an additive amendment header naming this archive-report, the proposed amendment, and the disposition.

## Next phase

CH-10 (`cachicamas-chat-permission`) is unblocked on PR merge. CH-11 (`cachicamas-chat-v1-completion`) follows after CH-10 closes; CH-11 depends on the deterministic end-to-end acceptance run that exercises conversation, streaming, cancellation, failure, persistence, reload, tool call, and approval.