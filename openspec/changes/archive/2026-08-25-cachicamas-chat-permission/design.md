# CH-10 — Design (cachicamas-chat-permission)

> Head reference for the design tail; full content in `engram://sdd/cachicamas-chat-permission/design` (observation #3990).

## 1. Technical approach

CH-10 closes two deliberate emptinesses from CH-09's PR #199 at `45fb69b9`:

1. The chat archetype's permission policy port was empty (CH-00.1's "AG-10 nil-bypass" survived because chat never constructed a policy).
2. The projector never accumulates `Exchange.ToolCalls` / `Exchange.ToolResults` from the live stream — a defect CH-09 left as F-CPM-001 because widening the projector would would have widened the WU-3 review budget.

CH-10 closes both. The policy port + an F-CPM-001 fix that retroactively makes CH-09's tool widening faithful on reload.

Binding constraint (`0005:947`): *the event stream stays a complete description of the session.*

## 2. Architectural decisions (D-1 .. D-12)

The 12 decisions are captured at `engram://sdd/cachicamas-chat-permission/design` (observation #3990) and the proposal observation (#3987).

## 3. File changes

Per task graph #3992 — ~9 new files + ~12 modified.

## 4. Data flow — end-to-end

```
Harness.Run (Turn.PermissionPolicy = chat.NewDefaultPermissionPolicy(["summarize_conversation"]))
├── EventKindPermissionDecisionRequired(callID="c1", tool="summarize_conversation", args=...)
│     → chat/projection.go: arm emits PermissionDecisionRequired{WireCallID:"c1", Tool:"summarize_conversation", Arguments:string(args)}
│     → SSE: event: permission.decision.required\ndata: {"wireCallId":"c1","tool":"summarize_conversation","arguments":"..."}\n\n
│     → use-chat-stream.ts: pushes kind:"hold"{id:hold-c1, tool, intent, args, decision:"pending"}
│     → transcript-line.tsx renders pending panel with Allow/Don't buttons
│
│     ┌────── USER CLICKS "Allow" ──────┐
│     │                                 │
│     │  frontend → POST /api/agent/turns/:id/permissions/c1
│     │             body: {outcome:"allow_once"}
│     │             → identityMiddleware resolves participantID
│     │             → registry.OwnerOf(turnID) == participantID cross-participant guard
│     │             → chat.PermissionPolicy.RecordVerdict(c1, allow_once)
│     │             → conv.Scheduler().WakeParked("c1")
│     │             → parked entry closed; re-entry into runPermissionGate with stored verdict
│     │             → second Resolve returns {Outcome: AllowOnce}
│     │             → emit PermissionDecisionMade{AllowOnce}
│
│     ├ EventKindPermissionDecisionMade(callID="c1", outcome=AllowOnce)
│     │     → chat/projection.go: arm emits PermissionDecisionMade (D-12: AllowOnce → "allow_once")
│     │     → projector.pendingDecisions += PermissionDecisionRecord{c1, summarize_conversation, allow_once}
│     │     → SSE: event: permission.decision.made\ndata: {...}\n\n
│     │     → use-chat-stream.ts: hold entry c1 → decision:"granted"
│
│     ├ EventKindToolStart(callID="c1", name="summarize_conversation", args=...)
│     │     → projection.go existing CH-09 arm → ToolCallStart wire event
│     │     → projector.toolCalls += ToolCallRecord{c1, summarize_conversation, ...}
│     │     → use-chat-stream.ts: push kind:"tool"{id:tool-c1, state:"running"}
│
│     ├ EventKindToolEndSuccess(callID="c1", content="...")
│     │     → projection.go existing CH-09 arm → ToolResult wire event
│     │     → projector.toolResults += ToolResultRecord{c1, ..., Outcome:success}
│     │     → use-chat-stream.ts: tool entry c1 → state:"done", result:content
│
│     └ buildTerminalExchange (F-CPM-001 fix): returns Exchange{ToolCalls:[c1], ToolResults:[c1], PermissionDecisions:[c1,allow_once], ...}
│
└── STORED TO chat_exchanges + chat_tool_calls + chat_tool_results + chat_permission_decisions (sibling-table transaction)
```

## 5. Persistence widening

0003_summarize.sql — `ALTER TABLE chat_conversations ADD COLUMN summary TEXT`.
0004_permission_decisions.sql — sibling table keyed by `(chat_exchanges.participant_id, chat_exchanges.position)`, FK ON DELETE CASCADE.

## 6. F-CPM-001 closure

Three parallel accumulators (`toolCalls` / `toolResults` / `permissionDecisions`) populated in their respective switch arms, threaded into `buildTerminalExchange` as additional parameters. Production reload now surfaces tool + permission activity (the same defect CH-09 left unfixed).

## 7. Substrate preservation

Same 10-file Layer-2 substrate list as CH-09 — byte-clean. The migrator runner's forward-only allowlist gained `ALTER TABLE ADD COLUMN` (the documented affordance).

## 8. Frontend wire-fragmentation guard

`parseTranscript` switch covers all 11 names; `KNOWN_EVENTS` is exactly 11 entries; `wireFrameName` switch covers all 11 cases. Lockstep invariant verified.

## 9. Composition-root discipline

`chat.NewDefaultPermissionPolicy` appears EXACTLY ONCE in production source (S-CPM-003). `SummarizeConversationTool` registered per-conversation (participantID capture).

## 10. Threat matrix

Per proposal #3987 / design #3990.

## 11. Verification

Per task graph #3992 — `cd backend/agent && make test`, `make lint`, `make build/chat`; `cd frontend && pnpm test:ci`, `pnpm lint`, `pnpm build.types`. INTEGRATION=1 gated postgres test in T-05b.