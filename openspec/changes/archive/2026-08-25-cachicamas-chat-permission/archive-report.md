# CH-10 — Archive Report (cachicamas-chat-permission)

## Lineage

preflight → explore (#3985) → decisions (#3983) → proposal (#3987, #3988) → spec (#3989) → design (#3990, #3991) → tasks (#3992) → apply-progress (#3994) → archive (this folder).

## What landed

11 conventional commits on `feat/chat-permission-ch10` from `main @ b4562280`:

1. `04b7c59f` — docs(chat): CH-10 apply-progress baseline
2. `75724077` — test(chat): CH-10 RED scaffold #1 — 2 empty WireEvent variants
3. `8d6239a1` — test(chat): CH-10 RED scaffold #2 — 4-arm stub projector
4. `601e2230` — feat(chat): CH-10 WU-1 — permission port + SummarizeConversationTool + composition-root wire
5. `f59790bf` — feat(chat): CH-10 WU-1b — POST /turns/:id/permissions/:callID reverse-channel handler
6. `6518db17` — feat(chat): CH-10 WU-2 — wire projection + SSE framing + D-8 deny collapse
7. `41f1cfcb` — feat(chat): CH-10 WU-3a — Exchange widens + memory + 0003 summary migration
8. `1b9fd35d` — feat(chat): CH-10 WU-3b — postgres sibling-table + 0004 permission_decisions migration
9. `09e76584` — feat(chat,web): CH-10 WU-4 — frontend delta + 11-arm parseTranscript
10. `b918abba` — feat(chat): CH-10 WU-5 — F-CPM-001 projector-accumulator fix
11. `fdcb4d96` — feat(chat): CH-10 WU-6 — substrate + wire-fragmentation guards + AGENTS pointer

## Charter closure

Acceptance criterion from doc 0005 (`0005:944`):
> *Given a tool call the policy defers, When the participant approves it from the browser, Then the suspended turn resumes and the tool's result reaches the transcript; and when they decline, the turn continues with the refusal recorded and no execution.*

Implemented end-to-end via:
- chat.PermissionPolicy (D-1, R-CPM-001) — chat-owned port
- chat.NewDefaultPermissionPolicy (D-2, R-CPM-002) — defer + AllowOnce + collapse
- 2 new SSE events (D-3, R-CPM-003) — permission.decision.required / made
- POST /api/agent/turns/:id/permissions/:callID (D-10, R-CPM-004) — reverse-channel
- Inline hold row in transcript (D-4) — use-chat-stream + transcript-line
- Exchange.PermissionDecisions + sibling table 0004 (D-5, R-CPM-006) — reload surface
- D-8 deny collapse (R-CPM-008) — projector suppresses tool.result after deny
- ConversationStore.UpdateSummary (D-9) — SummarizeConversationTool mutation target

## Status bump

Doc 0005 status: "11 of 12 milestones shipped" (CH-10.1..6 ticked).

## Spec promotion

NEW spec `openspec/specs/cachicamas-chat-permission/spec.md` (was already on disk from spec phase). Additive amendments to `chat-conversation-store/spec.md` (R-CCS-017/018 + NFR-CCS-009 + S-CCS-023..025) and `frontend-chat-layer1/spec.md` (REQ-12/13 + S-FCL-018..022). Identifier-append-only per CH-07/08/09 precedent.

## Substrate preservation

Empty diff on `git diff --stat main..HEAD -- backend/agent/src/agent/` — verified at T-08. The ten-file substrate list survives byte-clean.

## Next milestone

CH-11 (`cachicamas-chat-v1-completion`) is unblocked. Will proceed with the same `/sdd-new` cycle once CH-10's verify-report lands.