# CH-10 — Proposal (cachicamas-chat-permission)

## 1. Charter restatement (verbatim `0005:940-947`)

**Goal**: a tool call the policy will not allow unilaterally becomes a question the human answers, carried on the same stream as everything else.
**Deliverable**: the archetype's permission policy, the approval round-trip on the wire, and the frontend delta that asks.
**Acceptance**: *Given a tool call the policy defers, When the participant approves it from the browser, Then the suspended turn resumes and the tool's result reaches the transcript; and when they decline, the turn continues with the refusal recorded and no execution.*
**Depends on**: CH-09. **Out of scope**: remembered decisions and rule sets over tool names and arguments; a permission mode selector.

## 2. Twelve architectural decisions

- **D-1** — Chat-owned policy port (`chat.PermissionPolicy` is a type alias of `agent.PermissionPolicy`).
- **D-2** — Default v1 policy (`chat.NewDefaultPermissionPolicy(deferToolNames)`; `AllowAlways` → `AllowOnce` collapse; `ModifyInput` → `Defer` bypass).
- **D-3** — Wire shape: 2 new SSE event names + adjacent DTOs (new variants on the union, not new fields).
- **D-4** — Frontend affordance: inline `hold` row in transcript (3-state closed union `"waiting" | "allowed" | "denied"`).
- **D-5** — Persistence widening: `Exchange` widens with `PermissionDecisions`; sibling table `chat_permission_decisions`; defensive copy.
- **D-6** — Projector-accumulator fix (F-CPM-001 closure; CH-09 inherited defect — 3 parallel accumulators threaded into `buildTerminalExchange`).
- **D-7** — REQ-7 widening language (mirroring CH-09 D-7).
- **D-8** — Deny-state collapse (F-CPM-002/003 closure at design time; projector suppresses `tool.result` for the same wireCallId after `permission.decision.made{outcome: "deny"}`).
- **D-9** — 4th port method `ConversationStore.UpdateSummary` (forward-only `ALTER TABLE chat_conversations ADD COLUMN summary TEXT`).
- **D-10** — HTTP reverse-channel endpoint (`POST /api/agent/turns/:id/permissions/:callID`).
- **D-11** — Scheduler accessor (`Conversation.Scheduler() *agent.Scheduler` — method-only, returns `c.harness.Scheduler`).
- **D-12** — Layer-2 outcome collapse (4 → 2 at chat projector; `AllowAlways` → `"allow_once"`; `ModifyInput` → `"deny"`).

## 3. File changes

See task graph at `engram://sdd/cachicamas-chat-permission/tasks` (observation #3992).

## 4. Data flow

See design observation #3990.

## 5. Persistence widening

See design observation #3990.

## 6. Substrate preservation

See design observation #3990.

## 7. Frontend wire-fragmentation guard

`KNOWN_EVENTS` at `chat-api.ts` extends from 9 to 11 entries. `parseTranscript` switch extends from 9 to 11 arms. Lockstep invariant verified by `chat-api.spec.ts` + `wire_fragmentation_test.go`.

## 8. Order of work

Per #3992: 9 WUs producing 11 commits (T-01, T-02, T-03a, T-03b, T-04, T-05a, T-05b, T-06, T-07, T-08, T-09). Pre-authorised `size:exception` at preflight (review budget 1500 lines; CH-10 forecast 1800-3000).

## 9. Threat matrix

Per #3987 (proposal).

## 10. Risks

F-CPM-001 (CRITICAL — projector accumulators; closed in T-07); F-CPM-002/003 (deny collapse; D-8 design-time fix); D-1 wrapper direction confusion (type alias — explicit docstring + AGENTS pointer); D-9 forward-only migration (0003 ADD COLUMN nullable); D-11 Scheduler accessor (method-only); D-12 outcome collapse (loses `ModifyInput` modified arguments; deferred).

## 11. Rollback

Single PR revert of `feat/chat-permission-ch10`. Additive widening of `Exchange` removable by dropping the `PermissionDecisions` field; sibling-table DDL reverses via mirror-DROP on `0004_permission_decisions.sql`. The `ALTER TABLE chat_conversations ADD COLUMN summary TEXT` from `0003_summarize.sql` is the only forward-only element; rollback is `ALTER TABLE chat_conversations DROP COLUMN summary`.

## 12. Open questions

None blocking. The 12 locked decisions cover port shape (D-1), default policy (D-2), wire shape (D-3), frontend affordance (D-4), persistence (D-5), projector fix (D-6), REQ-7 widening (D-7), deny collapse (D-8), 4th port method (D-9), HTTP route (D-10), scheduler accessor (D-11), outcome collapse (D-12).