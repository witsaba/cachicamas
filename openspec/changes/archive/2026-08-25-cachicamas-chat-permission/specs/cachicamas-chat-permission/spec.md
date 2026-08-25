# Spec — `cachicamas-chat-permission` (CH-10)

> **Change**: `cachicamas-chat-permission` · **CH-10** (Wave 3, 11 of 12) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-10--approve-a-tool-call-from-the-browser) (`0005:936-947`) · leaves CH-10.1 `[leaf]` (`0005:940`) · CH-10.2 `[leaf]` · CH-10.3 `[leaf]` · CH-10.4 `[leaf]` · CH-10.5 `[guard]`
> **Closes**: R-15 (v2 § 6 seam 2 — approval is a suspension **inside the loop on the same stream**, not a parallel approval beside it). **Depends on**: CH-09 (`cachicamas-chat-tool-source`, PR #199 at `45fb69b9`). **Blocks**: CH-11 (`cachicamas-chat-v1-completion`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario independently verifiable. The `S-CPM-NNN` scenarios are the falsifiable contract; the `S-CCS-NNN` persistence scenarios (R-CPM-006) are duplicated in the `chat-conversation-store` additive amendment as the source of truth for the store-side widening; the `S-FCL-NNN` frontend scenarios are stated in the `frontend-chat-layer1` additive amendment and reference back here.
> **IDs**: requirements `R-CPM-NNN`, scenarios `S-CPM-NNN`, non-functional `NFR-CPM-NNN`. Distinct from `R-CTS-`/`S-CTS-` (CH-09), `R-CCS-`/`S-CCS-` (CH-06/07/08/09), `R-APP-`/`S-APP-` (Layer 2 `agent-permission-protocol`, AG-10, **CLOSED** at AG-13.5), and `REQ-`/`S-FCL-` (`frontend-chat-layer1`). **Append-only.**
> **Allocated ranges**: `R-CPM-001` … `R-CPM-099`; `S-CPM-001` … `S-CPM-199`; `NFR-CPM-001` … `NFR-CPM-099`. **Prefix verification**: `CPM` verified collision-free across `openspec/specs/`, `openspec/changes/`, `openspec/changes/archive/` and `docs/` (proposal #3987 §14a).
> **Scenario count**: **8 functional requirements → 18 functional scenarios** + **3 persistence scenarios** (S-CCS-023..025, mirrored in `chat-conversation-store` amendment) + **5 NFR scenarios** = **26 scenarios**. Plus 4 frontend mirror scenarios (S-FCL-018..021) in `frontend-chat-layer1` amendment.
> **Traces to**: R-15 (v2 § 6 seam 2), R-04 (chat owns its ports), R-08 (chat owns its tables), R-09 (MCP deferred against the policy port).
> **Depends on**: CH-09 (the chat-owned `ToolSource` port and the wire shape for tool events). **Layer 2 surface consumed (read-only)**: `agent.PermissionPolicy` (R-APP-001), `agent.PermissionVerdict` (R-APP-001), `agent.PermissionOutcome` (R-APP-001..007), `agent.PermissionDefer` (R-APP-001), `agent.Scheduler.WakeParked` (R-APP-003), `agent.Harness.Scheduler` (R-AGH-…), `agent.EventKindPermission*` constants. CH-10 does not widen Layer 2's surface — it consumes AG-10 / AG-13 / AG-14 verbatim.
> **Status**: **new capability**, promoted verbatim to `openspec/specs/cachicamas-chat-permission/spec.md` at archive. Additive amendments land in `chat-conversation-store` (R-CCS-017/018, NFR-CCS-009, S-CCS-023..025) and `frontend-chat-layer1` (REQ-12..13, S-FCL-018..022). Identifier-append-only per CH-07 / CH-08 / CH-09 precedent.

## Coverage — charter and decisions traced

| Charter / decision | Requirements | Spec scenarios |
|---|---|---|
| Charter `0005:944` acceptance (defer → approve → tool result on stream; decline → refusal recorded, no execution) | R-CPM-001, R-CPM-002, R-CPM-003, R-CPM-004, R-CPM-005 | S-CPM-001..016, S-CCS-023..025 |
| D-1 chat-owned policy port (`chat.PermissionPolicy` is a type alias of `agent.PermissionPolicy`) | R-CPM-001, NFR-CPM-001 | S-CPM-001..003 |
| D-2 default v1 policy (`chat.NewDefaultPermissionPolicy(deferToolNames)`; `AllowAlways` → `AllowOnce` collapse; `ModifyInput` → `Defer` bypass) | R-CPM-002 | S-CPM-004..006 |
| D-3 wire shape (2 new SSE event names + adjacent DTOs; **new variants on the union, not new fields**) | R-CPM-003, REQ-12..13 in `frontend-chat-layer1` amendment, NFR-CPM-003 | S-CPM-007..009, S-FCL-018..021 |
| D-4 frontend affordance (inline `hold` row in transcript; 3-state closed union `"waiting" | "allowed" | "denied"`) | R-CPM-005, REQ-12..13 in `frontend-chat-layer1` amendment | S-CPM-014..016, S-FCL-018..021 |
| D-5 persistence widening (`Exchange` gains `PermissionDecisions`; sibling table `chat_permission_decisions`; defensive copy) | R-CPM-006, R-CCS-017/018, NFR-CCS-009 | S-CCS-023..025 |
| D-6 projector-accumulator fix (F-CPM-001 closure; CH-09 inherited defect — 3 parallel accumulators threaded into `buildTerminalExchange`) | R-CPM-007 | S-CPM-017, S-CPM-018 |
| D-7 REQ-7 widening language (REQ-12..13 carry explicit "new variants, not new fields on existing variants" wording) | REQ-12, REQ-13 in `frontend-chat-layer1` amendment | (carried in FCL amendment) |
| D-8 deny-state collapse (F-CPM-002/003 closure at design time; projector suppresses `tool.result` for the same `wireCallId` after `permission.decision.made{outcome: "deny"}`) | R-CPM-008 | S-CPM-019, S-FCL-021 |
| D-9 4th port method `ConversationStore.UpdateSummary` (forward-only `ALTER TABLE chat_conversations ADD COLUMN summary TEXT`) | R-CCS-017 in `chat-conversation-store` amendment, NFR-CPM-005 | S-CCS-023, S-CPM-024 |
| D-10 HTTP reverse-channel endpoint (`POST /api/agent/turns/:id/permissions/:callID`) | R-CPM-004 | S-CPM-010..013 |
| D-11 Scheduler accessor (`Conversation.Scheduler() *agent.Scheduler` — method-only, returns `c.harness.Scheduler`) | R-CPM-001, R-CPM-004 | (covered by S-CPM-001 + S-CPM-010) |
| D-12 Layer-2 outcome collapse (4 → 2 at chat projector; `AllowAlways` → `"allow_once"`; `ModifyInput` → `"deny"`) | R-CPM-002, R-CPM-003 | S-CPM-005, S-CPM-006, S-CPM-009 |
| Substrate preservation (NFR-TLS-003 carry; 10-file Layer-2 list byte-unchanged) | NFR-CPM-001 | S-CPM-020 |
| Cache-prefix stability (V-REQ-14 carry) | NFR-CPM-002 | S-CPM-021 |
| Wire-fragmentation guard (`parseTranscript` covers all 11 event names) | NFR-CPM-003 | S-CPM-022 |
| Defensive copy on `Load` extends to `PermissionDecisions` | NFR-CPM-004 | S-CPM-023 |
| Forward-only schema migration (`ALTER TABLE ADD COLUMN nullable`; sibling table `CREATE TABLE`) | NFR-CPM-005 | S-CPM-024 |

## Purpose

CH-00.1's empty seam left the chat archetype without a permission policy for the entire `Run` cycle; every tool call ran through `runPermissionGate` against a nil policy (AG-10 nil-bypass), and the harness never asked the human anything. CH-10 closes the seam: it ships the chat-owned `chat.PermissionPolicy` port (D-1) that wraps `agent.PermissionPolicy` by interface, the `chat.NewDefaultPermissionPolicy(deferToolNames)` v1 default (D-2) that asks on a per-tool-name basis, the two new SSE event variants that carry the ask and the answer on the existing event stream (D-3 + v2 § 6 seam 2's "suspension **inside** the loop" binding constraint), the HTTP reverse-channel endpoint that delivers the user's click back to the parked gate (D-10), the inline transcript row that surfaces the decision to the user (D-4), the `Exchange` persistence widening that round-trips permission records on reload (D-5), and — critically — the F-CPM-001 projector-accumulator fix (D-6) that closes the CH-09 theatrical-widening defect before CH-10 inherits it. Substrate preservation (NFR-TLS-003): zero files under `backend/agent/src/agent/` are modified — chat depends on `agent.PermissionPolicy` by interface, never by import-of-internals; the Layer 2 surface is consumed verbatim (AG-10 closed, AG-13 wired the upward-path wake, AG-14 added no third release path per R-APP-009 / S-APP-018).

The acceptance criterion is observable end-to-end (`0005:944`): given a tool call the policy defers, when the participant approves it from the browser, then the suspended turn resumes and the tool's result reaches the transcript; and when they decline, the turn continues with the refusal recorded and no execution. The Gherkin at S-CPM-001..024 in this spec is the binding contract; the scenario names are the test names in `chat/permission_policy_test.go`, `chat/projection_permission_test.go`, `chat/http_permission_test.go`, `chat/store_permission_roundtrip_test.go`, `chat/substrate_guard_test.go`, `chat/wire_fragmentation_test.go`, `frontend/src/lib/chat-types.spec.ts`, `frontend/src/lib/chat-api.spec.ts`, `frontend/src/components/chat/use-chat-stream.spec.tsx`, and `frontend/src/components/chat/chat-app.spec.tsx`. The frontend mirror scenarios (S-CPM-014..016, S-CPM-019) are re-stated as **S-FCL-018..021** in the `frontend-chat-layer1` amendment; `S-CPM-*` is the source of truth, `S-FCL-*` references it.

## Requirements

### R-CPM-001 — Chat-owned permission port

The system MUST expose `chat.PermissionPolicy` in `backend/agent/src/chat/permission_policy.go` (D-1) as a **type alias** of `agent.PermissionPolicy` (`backend/agent/src/agent/permission_protocol.go:80-94`). The alias MUST be `type PermissionPolicy = agent.PermissionPolicy`. The chat package MUST depend on `agent.PermissionPolicy` by interface, never by import of Layer 2's internals (NFR-CPM-001; CH-09 precedent at `chat/tool_source.go` for the same posture). `Config.PermissionPolicy agent.PermissionPolicy` MUST be added to `chat.Config` (`backend/agent/src/chat/conversation.go:35-73`); `NewConversation` MUST reject a nil `PermissionPolicy` with the typed sentinel `ErrNilPermissionPolicy` (never panic). The composition root at `backend/agent/src/cmd/chat/main.go:227-245` is the **only** place that constructs the production `chat.NewDefaultPermissionPolicy(deferToolNames)` value; the factory closure gains exactly one line. `Conversation` MUST expose a method `Scheduler() *agent.Scheduler { return c.harness.Scheduler }` (D-11, option (b)) — no stored field on `Conversation`; the harness's own `Scheduler` is the single source of truth.

#### Scenarios

- **S-CPM-001** — `chat.PermissionPolicy` is a byte-identical alias of `agent.PermissionPolicy`. Given the `chat.PermissionPolicy` declaration in `chat/permission_policy.go`, when `var p chat.PermissionPolicy = chat.NewDefaultPermissionPolicy(nil)` is assigned, then `reflect.TypeOf(p) == reflect.TypeOf((*agent.PermissionPolicy)(nil)).Elem()` holds — the alias is byte-identical, not a redeclaration (the AG-09 `PreRequestHook` precedent at `cachicamas-chat-tool-source/spec.md` § "Explicit non-requirements" disallows a function-value port; an interface alias preserves the AG-10 D-1 contract).
- **S-CPM-002** — `Config{PermissionPolicy: nil}` is refused typed, never panics. Given a `Config{Provider, Store, ParticipantID, ToolSource, PermissionPolicy: nil}`, when `NewConversation(cfg)` runs, then it returns `(*Conversation, ErrNilPermissionPolicy)` typed (NFR-CTS-001 mirror — typed refusal, never a panic), and the typed `errors.Is(err, chat.ErrNilPermissionPolicy)` test succeeds.
- **S-CPM-003** — composition-root-only construction; the wire surface is independent of the implementation choice. Given the merged change, when `git grep -n 'chat.NewDefaultPermissionPolicy\|chat.NewPermissionPolicy' -- backend/agent/src/` runs, then exactly one match exists — the `cmd/chat/main.go` factory closure line — and zero matches under `chat/conversation.go` or any test file (composition-root-only discipline, per CH-09 D-1 precedent at `cachicamas-chat-tool-source/spec.md` S-CTS-001..003). The test surface MAY construct additional implementations to inject typed verdicts (mirrors CH-09's test-only `FromAgentRegistry` adapter at `chat/tool_source.go`); the production path is closed.

### R-CPM-002 — Default v1 policy: per-tool defer rule

The system MUST ship `chat.NewDefaultPermissionPolicy(deferToolNames []string) chat.PermissionPolicy` in `backend/agent/src/chat/permission_policy.go` (D-2). The returned `*defaultPermissionPolicy` MUST hold a `pendingDecision map[string]pendingDecision` keyed by `callID string`, where `pendingDecision{WireCallID string, Outcome string}`. `Resolve(ctx, call ai.ToolCall)` MUST return `agent.PermissionDefer` when `call.Name` is in `deferToolNames` and `agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}` otherwise (G7 — synchronous verdict for every tool not on the defer list). The default implementation MUST collapse `AllowAlways` to `AllowOnce` internally (D-12): if a future Layer 2 ask ever requires `AllowAlways` on the wire (it cannot at v1 because the chat projector already collapses at D-12), the default MUST return `AllowOnce` (CH-10's deferred register row 6, "remembered decisions"). The default implementation MUST NOT substitute arguments: if a future Layer 2 ask ever requires `ModifyInput`, the default MUST return `Defer` (CH-10's deferred register row 7, "ModifyInput on wire"). `Remember(ctx, toolName, outcome) bool` MUST return `false` unconditionally at v1 (CH-10 does not surface remembered rules; Layer 2's `CardinalityAtMostOne` for `permission_resolution_remembered` is suppressed at the chat wire by R-CPM-003's collapse). `recordVerdict(callID, outcome)` (a method on the `*defaultPermissionPolicy`, NOT a method of the `agent.PermissionPolicy` interface) MUST write `pendingDecision[callID] = pendingDecision{callID, outcome}` and return `ErrDecisionAlreadyMade` typed if the entry is already populated (D-10 — second click races are refused typed).

#### Scenarios

- **S-CPM-004** — default policy defers calls for tools in the defer list and allows everything else synchronously. Given `chat.NewDefaultPermissionPolicy([]string{"summarize_conversation"})` and two scripted `ai.ToolCall` values — one named `summarize_conversation` (in defer list) and one named `current_time` (not in defer list) — when `Resolve(ctx, call)` runs for each, then the `summarize_conversation` call returns `agent.PermissionDefer` and the `current_time` call returns `agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}` (G7 — the gate always sees a synchronous answer for non-deferred tools). Verified by `TestDefaultPermissionPolicy_DefersConfigured_AllowsRest`.
- **S-CPM-005** — `AllowAlways` collapses to `AllowOnce` in the default implementation. Given `chat.NewDefaultPermissionPolicy([]string{"current_time"})` and a `Resolve(ctx, call)` call path that would, if the policy were not collapsing, observe an `AllowAlways` ask, when `Resolve` returns, then the returned verdict is `agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}` — `AllowAlways` never reaches the chat wire (CH-10 deferred register row 6; D-12 wire collapse). Verified by `TestDefaultPermissionPolicy_AllowAlwaysCollapsed`.
- **S-CPM-006** — `ModifyInput` is bypassed by the default policy (no argument substitution). Given `chat.NewDefaultPermissionPolicy(nil)` and a `Resolve(ctx, call)` path that would, if the policy were substituting, return `ModifyInput` with `ModifiedArgs`, when `Resolve` returns, then the returned verdict is `agent.PermissionDefer` — the default policy does not substitute arguments and does not pretend to (CH-10 deferred register row 7). Verified by `TestDefaultPermissionPolicy_ModifyInputBypassed`. The reasoning: substituting arguments is the chat's job, not the policy's; a policy that substitutes silently would be a Layer-3 effect inside a Layer-2 surface, violating R-APP-011's "Layer 2 owns the protocol; Layer 3 owns the policy" boundary.

### R-CPM-003 — Forward-direction wire shape (permission ride the existing stream)

The system MUST add two new variants to the closed `chat.WireEvent` interface (`backend/agent/src/chat/wire.go:13-15`) — `PermissionDecisionRequired{WireCallID string, Tool string, Arguments string}` projected from `agent.EventKindPermissionDecisionRequired` (`backend/agent/src/agent/event.go:161-176`); `PermissionDecisionMade{WireCallID string, Outcome string}` projected from `agent.EventKindPermissionDecisionMade` with `Outcome` being the chat-owned **2-value vocabulary** `"allow_once" | "deny"` (D-12 collapse; Layer 2's 4-value `AllowOnce | AllowAlways | Deny | ModifyInput` collapses to 2 at the chat wire — `AllowAlways` → `"allow_once"`, `ModifyInput` → `"deny"`). `permission_resolution_remembered` from Layer 2 MUST be DROPPED at the chat wire (D-12 wire collapse — chat does not surface rule persistence; the closed 2-value chat vocabulary is binding). The two new variants MUST carry an `isWireEvent()` marker (mirroring `wire.go:25, 35, 46, 57, 69`). The two new `wireFrameName` cases (`permission.decision.required`, `permission.decision.made`) MUST extend the exhaustive switch at `backend/agent/src/chat/eventsource.go:31-48` to 11 cases total; a new variant without a case is a compile error (S-CPM-022, mirror of `cachicamas-chat-tool-source/spec.md` S-CTS-007). The two new `parseTranscript` cases in `frontend/src/lib/chat-types.ts:284-360` and the two new `KNOWN_EVENTS` entries in `frontend/src/lib/chat-api.ts:285-291` MUST extend the same exhaustive surface — NFR-CPM-003 wire-fragmentation guard.

#### Scenarios

- **S-CPM-007** — `permission.decision.required` serialises to the closed SSE shape. Given a `PermissionDecisionRequired{WireCallID: "c1", Tool: "summarize_conversation", Arguments: "{}"}`, when `writeFrame(w, flusher, ev)` serializes via `httptest.ResponseRecorder`, then the SSE bytes are exactly `event: permission.decision.required\ndata: {"wireCallId":"c1","tool":"summarize_conversation","arguments":"{}"}\n\n` (lowercase JSON keys, parallel to the CH-09 `ToolCallStart` SSE frame at `cachicamas-chat-tool-source/spec.md` S-CTS-006).
- **S-CPM-008** — `permission.decision.made{outcome: "allow_once"}` serialises to the closed SSE shape (D-12 collapse applied). Given a `PermissionDecisionMade{WireCallID: "c1", Outcome: "allow_once"}` projected from a Layer 2 `permission_decision_made{outcome: AllowAlways}` event (the collapse path), when `writeFrame` serialises, then the SSE bytes are exactly `event: permission.decision.made\ndata: {"wireCallId":"c1","outcome":"allow_once"}\n\n` — `AllowAlways` never reaches the browser as `"allow_always"`.
- **S-CPM-009** — `permission_resolution_remembered` from Layer 2 is dropped at the chat wire. Given a recorded Layer 2 stream `[permission_decision_required, permission_decision_made{outcome: AllowAlways, modifiedArgs}, permission_resolution_remembered{toolName: "summarize_conversation"}]`, when the chat projector (`backend/agent/src/chat/projection.go`) drains it, then the wire channel emits exactly 2 permission events (`permission.decision.required` + `permission.decision.made{outcome: "allow_once"}`) and ZERO `permission.decision.made` events with the `"remembered"` outcome — `permission_resolution_remembered` has no chat-side vocabulary and is dropped at the wire (D-12). Verified by `TestProjection_DropsRememberedEvent`.

### R-CPM-004 — Reverse-direction HTTP endpoint

The system MUST expose `POST /api/agent/turns/:id/permissions/:callID` behind the existing `identityMiddleware` (`backend/agent/src/chat/http.go:193-205`). The handler MUST be `HandlePermissionDecision(registry, resolver) echo.HandlerFunc` and MUST be mounted via a fresh `RegisterPermissionRoutes(e, resolver, registry)` helper (mirror of CH-08's `RegisterResumeRoutes` pattern at `http.go:568-569`). The body MUST be `{outcome: "allow_once" | "deny"}` — the closed 2-value chat wire vocabulary from R-CPM-003. On a successful POST the handler MUST (a) verify the `:callID` is registered in the conversation's active policy state via `conv.PermissionPolicy().recordVerdict(callID, outcome)` (D-2 — typed sentinel `ErrDecisionAlreadyMade` on second click; the handler maps this to 409 conflict); (b) call `Conversation.Scheduler().WakeParked(callID)` (D-11 — method-only accessor). The handler MUST return 200 OK on success, 403 `not_found` on a cross-participant `:callID` (the `registry.OwnerOf(turnID) == ident.ParticipantID()` guard from `HandleStreamEvents` at `http.go:308-358` is the binding precedent), 404 `not_found` on an unknown `:callID` (caller-side race: the parked entry may have already resolved and `WakeParked` returns `agent.ErrStrayDecision` per R-APP-003 — the handler maps this to 404 typed), 422 `validation` on a malformed body (the JSON body's `outcome` field MUST be one of `"allow_once" | "deny"`; out-of-vocab is 422). The handler MUST NOT widen `ApiResult` (CH-08's 5-kind envelope at `frontend/src/lib/api.ts:96-110` covers all four cases: 200 → `ok`, 403 → `not_found`, 404 → `not_found`, 422 → `validation`).

#### Scenarios

- **S-CPM-010** — happy path: a participant clicks Allow on a parked tool. Given a participant whose `conv.PermissionPolicy()` holds a parked `pendingDecision["c1"] = pendingDecision{outcome: "allow_once"}` (set by the first `Resolve` returning `PermissionDefer`), when `POST /api/agent/turns/T/permissions/c1` with body `{outcome: "allow_once"}` arrives and `identityMiddleware` resolves the participant as the owner of turn `T`, then the handler returns 200 OK, `conv.PermissionPolicy().pendingDecision["c1"].Outcome` is `"allow_once"`, and `conv.Scheduler().WakeParked("c1")` returns `nil` (R-APP-003 satisfied — the parked entry is live). Verified by `TestHandlePermissionDecision_HappyPath_AllowOnce`.
- **S-CPM-011** — cross-participant guard refuses a foreign turn's `callID`. Given two participants `p1` (owner of turn `T`) and `p2`, when `POST /api/agent/turns/T/permissions/c1` arrives with `p2`'s identity, then the handler returns 403 `not_found` (the existing `HandleStreamEvents` cross-participant guard at `http.go:308-358` is the binding precedent; the `not_found` kind is reused to avoid leaking existence), `p1`'s `pendingDecision["c1"]` is unchanged, and `WakeParked("c1")` was NOT called. Verified by `TestHandlePermissionDecision_CrossParticipant_403NotFound`.
- **S-CPM-012** — unknown `callID` is refused as 404 not_found (caller-side race: parked entry already cleared). Given a parked entry that has been woken + cleared by a previous request, when `POST /api/agent/turns/T/permissions/c1` arrives with `{outcome: "allow_once"}`, then the handler returns 404 `not_found` (typed — `agent.ErrStrayDecision` is mapped at the boundary per R-APP-003 / S-APP-004), no second click races through, and the participant's transcript is unchanged. Verified by `TestHandlePermissionDecision_UnknownCallID_404NotFound`.
- **S-CPM-013** — malformed body is refused as 422 validation. Given `POST /api/agent/turns/T/permissions/c1` with body `{outcome: "allow_always"}` (out-of-vocab; the closed 2-value vocabulary is `"allow_once" | "deny"` per R-CPM-003), when the handler runs, then it returns 422 `validation`, `pendingDecision["c1"]` is unchanged, and `WakeParked("c1")` was NOT called (a forged `"allow_always"` body MUST NOT reach the chat wire because the chat wire does not carry `"allow_always"`). Verified by `TestHandlePermissionDecision_MalformedBody_422Validation`. The body validation also rejects: missing `outcome` field, `outcome: null`, `outcome: ""`, `outcome: "modify_input"`, `outcome: 42` — all return 422 typed.

### R-CPM-005 — Inline transcript row (frontend affordance)

The system MUST render a permission row inline in the transcript between the assistant said entry and the tool result entry (D-4). The row's state MUST be a closed 3-value union: `"waiting" | "allowed" | "denied"` (`frontend/src/components/chat/transcript-line.tsx:163-230` already supports this 3-value state for `kind: "hold"` entries; CH-10 wires the state plumbing only). In `"waiting"` state the row MUST show two buttons: Allow once, Deny. Clicking Allow once MUST POST `{outcome: "allow_once"}` to `POST /api/agent/turns/:id/permissions/:callID` (R-CPM-004); clicking Deny MUST POST `{outcome: "deny"}`. On `permission.decision.made` SSE frame arrival (R-CPM-003) the row MUST morph to `"allowed"` when the frame's `outcome` was `"allow_once"` or to `"denied"` when the frame's `outcome` was `"deny"`. The closed union is the only state machine; no other transitions are valid (a `"waiting"` row cannot morph to `"allowed"` without a `permission.decision.made` frame; a `"denied"` row cannot morph back to `"waiting"`). `chat-app.tsx` MUST pass `onDecide$` to every `<TranscriptLine>` for `kind === "hold"` entries (the existing renderer at `transcript-line.tsx:163-230` already accepts the QRL — CH-10 only closes the seam).

#### Scenarios

- **S-CPM-014** — initial render on `permission.decision.required` is `waiting`. Given a `permission.decision.required{wireCallId: "c1", tool: "summarize_conversation", arguments: "{}"}` SSE frame arriving mid-turn, when the page receives it, then a new transcript entry with `kind: "hold"`, `id: "c1"`, `decision: "waiting"`, two visible buttons (Allow once, Deny) is appended between the assistant said entry and the tool result entry. Verified by `TestUseChatStream_PermissionRequired_PushesWaitingHoldEntry`.
- **S-CPM-015** — `allow` click morphs the row to `allowed` on the matching SSE frame. Given a `waiting` hold entry for `wireCallId: "c1"`, when the user clicks Allow once and the server emits `permission.decision.made{wireCallId: "c1", outcome: "allow_once"}`, then the entry's `decision` becomes `"allowed"`, the buttons are removed, and the result entry that follows carries the tool's actual content (mirrors the chat projector arms at R-CPM-007..009). Verified by `TestUseChatStream_AllowClick_MorphsToAllowed`.
- **S-CPM-016** — `deny` click morphs the row to `denied` and the tool entry is suppressed. Given a `waiting` hold entry for `wireCallId: "c1"`, when the user clicks Deny and the server emits `permission.decision.made{wireCallId: "c1", outcome: "deny"}`, then the entry's `decision` becomes `"denied"`, the buttons are removed, and NO `kind: "tool"` entry for `wireCallId: "c1"` is appended — the D-8 deny collapse from R-CPM-008 means the matching `tool.result` frame was suppressed at the projector (this scenario asserts the frontend consumer side; S-CPM-019 asserts the projector side). Verified by `TestUseChatStream_DenyClick_MorphsToDenied_NoToolEntry`.

### R-CPM-006 — Persistence widening: Exchange adds PermissionDecisions

The system MUST widen `chat.Exchange` (`backend/agent/src/chat/store.go:41-50`) additively with `PermissionDecisions []PermissionDecisionRecord{WireCallID string, Tool string, Outcome string}` (D-5). `PermissionDecisionRecord` is the port-side projection of the chat wire DTO `permissionDecisionDTO`; `Outcome` is the chat wire's closed 2-value vocabulary `"allow_once" | "deny"` (NOT Layer 2's 4-value vocabulary — R-CPM-003 D-12 collapse already happened at the projector; persistence only sees the collapsed form). The forward-only sibling table `chat_permission_decisions` MUST be created by `backend/agent/src/chat/migrations/0004_permission_decisions.sql`, keyed by `(participant_id, exchange_position, position)` with FK to `chat_exchanges` ON DELETE CASCADE (mirrors `chat/migrations/0002_tool_records.sql:60` template). The 4th method on the `ConversationStore` port — `UpdateSummary(participantID, summary string) error` (R-CCS-017 additive per the CH-08 R-CCS-013 precedent at `chat-conversation-store/spec.md:311-313`) — MUST be added in the same port declaration; `MemoryConversationStore` and `PostgresConversationStore` MUST both implement it; `chat/migrations/0003_summarize.sql` MUST add `summary TEXT` (nullable) to `chat_conversations` (NFR-CPM-005 forward-only `ADD COLUMN nullable` affordance). The widening MUST preserve REQ-7 closed-union enforcement on `ExchangeDTO`: `permissionDecisions?: readonly PermissionDecisionDTO[]` is a new optional slice field, never a new variant on an existing DTO (`cachicamas-chat-tool-source/spec.md` R-CTS-006 mirror). `MemoryConversationStore.Load` MUST deep-copy `PermissionDecisions` via a `copyPermissionDecisionRecords` helper (NFR-CPM-004 / NFR-CCS-009 carry NFR-CCS-004 / NFR-CCS-008 forward).

#### Scenarios (mirrored to `chat-conversation-store` amendment as **S-CCS-023..025**)

- **S-CCS-023** — `UpdateSummary + Load` round-trips the summary verbatim (Gherkin verbatim, explore #3985). Given a postgres adapter recording `ConversationSummaryDTO.Summary = "the participant asked about cats"`, when `UpdateSummary(participantID, "the participant asked about cats")` is called and then `Load(participantID)` is called, then the returned summary is byte-equal to the input. Verified by `TestUpdateSummary_RoundTrip_Memory` (both adapters, scenario text unchanged; cross-process variant gated `INTEGRATION=1`).
- **S-CCS-024** — defensive copy on `Load` extends to `PermissionDecisions` (Gherkin verbatim, explore #3985; mirrors CH-09 S-CCS-020). Given an in-memory adapter recording `Exchange{PermissionDecisions: [d1, d2]}` for participant `p`, when `Load(p)` is called, then the returned slice's `PermissionDecisions` field is byte-equal to the input AND the caller's later mutation does not corrupt the store (NFR-CCS-009 carries NFR-CCS-008 forward, which carries NFR-CCS-004 forward).
- **S-CCS-025** — permission decisions never leak across participants (Gherkin verbatim, explore #3985; mirrors CH-09 S-CCS-022). Given a recording under participant `p1` with `PermissionDecisions: [d1]`, when `Load("p2")` is called, then `p2`'s slice contains no permission decisions from `p1` (R-CHS-004.b shape preserved).

### R-CPM-007 — F-CPM-001 closure: projector accumulators are live

The system MUST accumulate wire events into the persisted `Exchange` as they pass through the chat projector (`backend/agent/src/chat/projection.go`). THREE parallel accumulators MUST thread state into `buildTerminalExchange`: (a) `toolCalls []ToolCallRecord` + `toolResults []ToolResultRecord` for tool events (closes F-CPM-001's CH-09 defect at `projection.go:208-251` where `ToolCalls`/`ToolResults` were always zero/nil); (b) `permissionDecisions []PermissionDecisionRecord` for permission events (closes the same defect for CH-10's `PermissionDecisions` field). The accumulator state MUST be built DURING the sink range, NOT derived from a separate test fixture (the F-CPM-001 defect is that `chat/store_scenarios_test.go:296-308` and `chat/store_postgres_test.go:344-348` build test exchanges with the fields populated directly — they bypass the projector entirely; production reload returns empty arrays). `buildTerminalExchange` MUST receive the accumulated state as additional parameters and construct the full `Exchange`. After the fix: a reload transcript for a turn that included 2 tool calls and 1 permission decision MUST byte-equal the live transcript of the same turn (the live transcript is the ordered sequence of `tool.call.start`, `tool.result`, `permission.decision.required`, `permission.decision.made` SSE frames the user saw; the reload transcript is the ordered sequence of `kind: "tool"` and `kind: "hold"` entries the user sees on reload). The fix MUST live in `chat/projection.go`; no other file is touched by this requirement (the WU-2 GREEN commit may include the related `wireFrameName` extension and `parseTranscript` extension as separate commits in the same PR per CH-08 commit discipline).

#### Scenarios

- **S-CPM-017** — live transcript byte-equals reload for tool + permission events. Given a scripted turn that includes 2 tool calls (`summarize_conversation` once, `current_time` once) and 1 permission decision (`summarize_conversation` deferred and approved), when the live stream is drained AND a reload reads `Load(participantID)` and `exchangesToEntries(exchanges)`, then the reload-side entries' `kind`, `id`, `args`, `result`, `decision`, and order match the live-side SSE frames' `wireCallId`, `tool`, `arguments`, `outcome`, `content`, and arrival order byte-for-byte. Verified by `TestProjection_AccumulatorsThreaded_LiveByteEqualsReload` (F-CPM-001 closure proof).
- **S-CPM-018** — `Exchange` carries the right number of records. Given a scripted turn with N tool calls and M permission decisions (where N and M are scripted: 2 and 1 for the canonical scenario), when `buildTerminalExchange(...)` returns the persisted `Exchange`, then `len(Exchange.ToolCalls) == N`, `len(Exchange.ToolResults) == N`, and `len(Exchange.PermissionDecisions) == M` — and the F-CPM-001 pre-fix zero-fields outcome is impossible (a scratch edit that removes the accumulator threading makes this test fail with `len = 0, want 2` / `len = 0, want 1`). Verified by `TestBuildTerminalExchange_AccumulatorsPopulated`.

### R-CPM-008 — F-CPM-002/003 closure: deny suppresses the matching tool.result

The system MUST observe `permission.decision.made{wireCallID: c1, outcome: "deny"}` and suppress the corresponding `tool.result{WireCallID: c1, ...}` SSE frame for the same `wireCallId`. The suppression is a chat-side projection rule (D-8), NOT a Layer 2 protocol change (Layer 2's `runPermissionGate` Deny branch at `scheduler.go:998-1034` legitimately emits both `permission_decision_made{Deny}` AND `tool_end_execution_failure` per R-APP-005 — the chat projector is the suppression site). The permission row alone renders; the tool entry never reaches the wire. The closed 3-value union's `"denied"` state is reachable only via this path (the wire never carries a `tool.result{outcome: "denied"}` event; `state: "denied"` on a `kind: "tool"` entry is the frontend's correlation between the prior hold decision and the (suppressed) tool entry, not a direct wire signal). The implementation MUST live in `chat/projection.go` (Go) and `use-chat-stream.ts` (TS) — Go holds the projector-side suppression rule; TS holds the redundant frontend-side correlation so reload-side entries that bypassed the projector also map correctly.

#### Scenarios

- **S-CPM-019** — Deny suppresses the matching `tool.result` on the wire (Gherkin verbatim, explore #3985; design-time resolution of F-CPM-002/003). Given a deferred tool call `c1` and the participant clicking Deny, when the projector drains `[permission_decision_required, permission_decision_made{outcome: Deny}, tool_end_execution_failure]` from `sink`, then the wire channel emits `permission.decision.required{c1}`, `permission.decision.made{c1, outcome: "deny"}`, and ZERO `tool.result` events for `c1` — the suppression is by-`wireCallId` (a different `wireCallId`'s `tool.result` flows through unchanged). Verified by `TestProjection_DenySuppressesToolResult`. The frontend-side mirror: `use-chat-stream.ts:248-260` correlates `wireCallId` to the prior hold decision and would emit `state: "denied"` if the wire ever carried the frame — but the projector suppresses the frame, so the wire never carries it. Implementation ripple: `chat/projection.go` gains ~10 LOC for the `deniedSet map[string]bool` accumulator; `use-chat-stream.ts` gains ~5 LOC for the by-`wireCallId` correlation lookup; no other file is touched.

## Non-functional requirements

### NFR-CPM-001 — Substrate preservation

The system MUST NOT modify any file under `backend/agent/src/agent/`. The 10-file Layer-2 substrate list at `openspec/AGENTS.md:92-110` (`event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `permission_events.go`, `permission_protocol.go`, `scheduler.go` (CH-10 only READS `WakeParked`), `harness.go` (CH-10 only READS `Scheduler` field + `TurnOptions.PermissionPolicy`), plus `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`) carries forward byte-unchanged. `TestChat_SubstrateUntouched` (`backend/agent/src/chat/substrate_guard_test.go`) MUST run inside `cd backend/agent && make test` and MUST fail on any non-empty `git diff --stat main..HEAD -- backend/agent/src/agent/`.

#### Scenario

- **S-CPM-020** — substrate byte-unchanged against main. Given the merged change, when `cd backend/agent && make test` runs, then `TestChat_SubstrateUntouched` passes AND `git diff --stat main..HEAD -- backend/agent/src/agent/` is empty (the 10-file list plus substrate tooling is byte-clean). Verified by `TestChat_SubstrateUntouched`.

### NFR-CPM-002 — Cache-prefix stability

The system MUST NOT invalidate the cache prefix between turns. `chat.PermissionPolicy` MUST be constructed ONCE at the composition root (`cmd/chat/main.go:227-245` factory closure, one `PermissionPolicy:` line mirroring CH-09's `ToolSource:` line at `cachicamas-chat-tool-source/spec.md` S-CTS-001..003). `NewConversation` MUST resolve it into `Turn.PermissionPolicy = cfg.PermissionPolicy` exactly once (mirrors how CH-09's `Turn.Tools` is set once at `conversation.go:108-113`). Across N turns in the same conversation, `Turn.PermissionPolicy` MUST be byte-stable. Layer 1's deterministic ordering at `ai/tool_set.go:32-94` ensures the JSON tool-list payload is identical turn-to-turn. `PermissionPolicy` does NOT enter the cache prefix — it is a per-turn *behavior*, not a per-turn *payload*. V-REQ-14's invariant is satisfied by construction.

#### Scenario

- **S-CPM-021** — `PermissionPolicy` does not enter the cache prefix. Given a factory closure that constructs `chat.NewDefaultPermissionPolicy(deferToolNames)` once per process, when the same conversation drives 3 turns in sequence, then `Turn.PermissionPolicy` is the SAME value across all 3 turns (the harness's `Tools` field at `Harness.Turn` (`harness.go:62`) is set once in `NewConversation` and never re-set). The acceptance is by construction — no scenario text; verified by `TestCachePrefix_PermissionPolicyStable_ThreeTurns` which asserts the harness's `Turn.PermissionPolicy` pointer is the same value across turns.

### NFR-CPM-003 — Wire-fragmentation guard

The system MUST keep the `parseTranscript` switch in `frontend/src/lib/chat-types.ts:284-360` in lockstep with `KNOWN_EVENTS` in `frontend/src/lib/chat-api.ts:285-291`. The 11-name set is `{message.start, message.delta, message.end, turn.end, error, tool.call.start, tool.call.delta, tool.call.end, tool.result, permission.decision.required, permission.decision.made}`. A missing `parseTranscript` case is a TypeScript compile error (the `assertNever` probe at `chat-types.ts:233-235` enforces exhaustiveness); a missing `KNOWN_EVENTS` entry breaks the dispatch. The backend-side mirror at `backend/agent/src/chat/eventsource.go:wireFrameName` MUST extend to 11 cases; a missing case is a Go compile error (CH-09 S-CTS-007 mirror).

#### Scenario

- **S-CPM-022** — wire-fragmentation guard covers all 11 event names. Given the merged `chat-types.ts`, when `pnpm --filter @cachicamas/frontend test:ci` runs, then the wire-fragmentation test passes (the `parseTranscript` switch covers all 11 known event names from `KNOWN_EVENTS`; a missing case is a TypeScript compile error from `assertNever`; `KNOWN_EVENTS` is exactly 11 names). Mirrors `cachicamas-chat-tool-source/spec.md` S-CTS-024 with the 9 → 11 extension.

### NFR-CPM-004 — Defensive copy on Load

The system MUST perform a defensive copy of `PermissionDecisions` on `ConversationStore.Load(participantID)` such that caller-side mutation of the returned slice does not corrupt the store. The discipline mirrors NFR-CCS-008 (CH-09) and NFR-CCS-004 (CH-06): `MemoryConversationStore.Load` returns a fresh slice via `copyPermissionDecisionRecords` (and likewise for `ToolCalls`/`ToolResults` carried forward); `PostgresConversationStore.Load` materialises a fresh slice from `chat_permission_decisions` sibling-table rows. Carries NFR-CCS-008 / NFR-CCS-004 forward. The widening lands as **NFR-CCS-009** in the `chat-conversation-store` additive amendment.

#### Scenario

- **S-CPM-023** — defensive copy on `Load` extends to `PermissionDecisions`. Given an in-memory adapter recording `Exchange{PermissionDecisions: [d1, d2]}` for participant `p`, when `Load(p)` is called and the caller mutates `result[0].Outcome = "deny"` (where `result` is the returned `Exchange.PermissionDecisions`), then a subsequent `Load(p)` returns the original `[d1, d2]` with `d1.Outcome` unchanged (the store's state is uncorrupted). Mirrors CH-09 S-CCS-020 verbatim.

### NFR-CPM-005 — Forward-only schema migration

The system MUST NOT alter pre-existing tables. `chat/migrations/0003_summarize.sql` MUST be `ALTER TABLE chat_conversations ADD COLUMN summary TEXT` (nullable — the documented forward-only affordance per `openspec/AGENTS.md` "Substrate preservation in `backend/agent`" paragraph on CH-07's `pgx/v5` admission precedent). `chat/migrations/0004_permission_decisions.sql` MUST be `CREATE TABLE chat_permission_decisions (...)` (new sibling table keyed by `(participant_id, exchange_position, position)` with FK to `chat_exchanges` ON DELETE CASCADE, mirroring `chat/migrations/0002_tool_records.sql:60`). No DROP, no ALTER of pre-existing columns.

#### Scenario

- **S-CPM-024** — migrations are forward-only. Given the migration set at CH-10 application time (`0001_init.sql`, `0002_tool_records.sql`, `0003_summarize.sql`, `0004_permission_decisions.sql`), when a fresh postgres adapter is constructed against an empty database, then `migrator.Up(ctx)` succeeds in order (no DROP, no ALTER of pre-existing columns; `0003` is `ADD COLUMN nullable`; `0004` is `CREATE TABLE`); and when the adapter's `Load(participantID)` is called, the `summary` column reads back as `NULL` for a participant who has never called `UpdateSummary`, byte-clean against `0001_init.sql`'s `chat_conversations` schema (no implicit defaults written by the new migration). Verified by `TestMigrationForwardOnly_ApplyOrder`.

## Spec defect F-CPM-002/003 resolution (recorded, design-time fixed)

**Issue** (`#3985` §4.2 + §5 F-CPM-002/003): `use-chat-stream.ts:249` collapses tool results to `state: "done" | "failed"` but `mock/chat.ts:42` closed tool-state union is `"running" | "done" | "denied" | "failed"`. Layer 2's Deny branch emits BOTH `permission_decision_made{deny}` AND `tool_end_execution_failure` for the same `wireCallId` (R-APP-005 + `scheduler.go:998-1034`). Without resolution, the same `wireCallId` would render twice: once as a permission row (R-CPM-005's `"denied"` state on the hold entry), once as a failed tool entry (`use-chat-stream.ts:249`'s `"failed"` state on the tool entry).

**Verified across the repo**:

| Source | State | Note |
|---|---|---|
| `frontend/src/components/chat/transcript-line.tsx:120-161` | tool branch filters on `running`, `denied`, `failed` | impl supports `"denied"` (R-CCP-008 mirror) |
| `frontend/src/lib/mock/chat.ts:42` | `"running" | "done" | "denied" | "failed"` | impl supports `"denied"` |
| `frontend/src/components/chat/use-chat-stream.ts:249` | sets `state: "done" | "failed"` | impl does NOT support `"denied"` |
| `openspec/specs/**` | no occurrence of `state: "denied"` for tool entries | empty |
| engram `#3985` §4.2 | `use-chat-stream.ts:249` only emits `done | failed` | transcribed |

**Resolution at design time (D-8)**: the chat projector at `chat/projection.go` tracks per-`wireCallId` "denied" state set on `permission.decision.made{outcome: "deny"}`; the `tool.result` arm checks the set and SUPPRESSES emission if present (R-CPM-008 / S-CPM-019). The denial renders the permission row alone with state `"denied"` (R-CPM-005 / S-CPM-016). Implementation is a single switch case in `chat/projection.go` (Go) and one by-`wireCallId` correlation in `use-chat-stream.ts` (TS). NO ripple to `mock/chat.ts`, `transcript-line.tsx`, `use-mock-turn.ts`, or any other spec. The implementation already supports `"denied"`; only the live-stream code path that emits state needs the new arm.

**Pattern precedent**: F-CHT-9.1 (CH-09's `#3964` record) — same shape: spec-text defect resolved at design phase, one spec line amendment, no implementation ripple outside chat.

**F-CPM-002/003 resolved at spec phase** — recorded as the design-time resolution (this section). The apply phase implements the suppression (S-CPM-019 / S-FCL-021).

## Acceptance — proposal success criteria mapped to scenarios

| Acceptance criterion | Evidence |
|---|---|
| Charter `0005:944` acceptance (defer → approve → tool result on stream; decline → refusal recorded, no execution) | S-CPM-001..016, S-CCS-023..025 |
| D-1 chat-owned policy port (`chat.PermissionPolicy` type alias of `agent.PermissionPolicy`) | S-CPM-001..003 |
| D-2 default v1 policy (`chat.NewDefaultPermissionPolicy(deferToolNames)`; `AllowAlways` → `AllowOnce` collapse; `ModifyInput` → `Defer` bypass) | S-CPM-004..006 |
| D-3 wire shape (2 new SSE variants on closed union; "new variants, not new fields" wording) | S-CPM-007..009, S-CPM-022 |
| D-4 frontend affordance (inline `hold` row in transcript; 3-state closed union) | S-CPM-014..016, S-FCL-018..021 |
| D-5 persistence widening (`Exchange` widens with `PermissionDecisions`; sibling table) | S-CCS-023..025 |
| D-6 F-CPM-001 projector-accumulator fix (3 parallel accumulators threaded into `buildTerminalExchange`) | S-CPM-017, S-CPM-018 |
| D-7 REQ-7 widening language (REQ-12..13 with explicit "new variants, not new fields" wording) | REQ-12, REQ-13 in `frontend-chat-layer1` amendment |
| D-8 deny-state collapse (F-CPM-002/003 closure at design time) | S-CPM-019, S-FCL-021 |
| D-9 4th port method (`ConversationStore.UpdateSummary`; forward-only `ADD COLUMN nullable`) | S-CCS-023, S-CPM-024 |
| D-10 HTTP reverse-channel endpoint (`POST /api/agent/turns/:id/permissions/:callID`) | S-CPM-010..013 |
| D-11 Scheduler accessor (method-only, returns `c.harness.Scheduler`) | S-CPM-001 + S-CPM-010 (composition-root construction) |
| D-12 Layer-2 outcome collapse (4 → 2 at chat projector) | S-CPM-005, S-CPM-006, S-CPM-008, S-CPM-009 |
| Substrate preservation (NFR-TLS-003 — 10-file list byte-unchanged) | NFR-CPM-001, S-CPM-020 |
| Cache-prefix stability (V-REQ-14 carry) | NFR-CPM-002, S-CPM-021 |
| Wire-fragmentation guard (parseTranscript covers all 11 event names) | NFR-CPM-003, S-CPM-022 |
| Defensive copy on `Load` extends to `PermissionDecisions` | NFR-CPM-004, S-CPM-023 |
| Forward-only schema migration (`ADD COLUMN nullable`; sibling `CREATE TABLE`) | NFR-CPM-005, S-CPM-024 |
| `cd backend/agent && make test` race-clean (uncached; cached runs are not evidence) | (whole spec) |
| `cd backend/agent && make lint && make build/chat` clean | (whole spec) |
| Frontend `pnpm --filter @cachicamas/frontend test:ci` green; `pnpm lint` 0/0; `pnpm build.types` clean | (whole spec) |
| Postgres cross-process (`S-CCS-023 INTEGRATION=1`) | S-CCS-023 |

## Explicit non-requirements

| Not required here | Why, and who owns it |
|---|---|
| `Remembered decisions and rule sets over tool names and arguments` | Charter `0005:946` — out of scope. Future spec |
| `A permission mode selector` (per-conversation allow-all / allow-none toggle) | Charter `0005:946` — out of scope. Future spec |
| `MCP tool sources` | Charter `0005:933` — out of scope. CH-09 / doc 0004 own |
| `Sandboxing / policy injection (agent.Tool.PolicySlot interpretation)` | Charter `0005:933`; v2 § 6 seam 3. Layer 3 owns |
| `The coding archetype's tools / permission policy` | Owned by doc 0004 (`cachicamas-ai-coding-archetype`) |
| `Widening the closed agent.PermissionPolicy interface` | D-1: chassis is `chat.PermissionPolicy`; `agent.PermissionPolicy` is the executor-side seam, untouched |
| `Widening the closed agent.PermissionOutcome vocabulary` | D-12: Layer 2's 4 outcomes stay; chat collapses 4 → 2 at the wire |
| `Widening the closed WireEvent union with new FIELDS on existing variants` | R-CPM-003 D-3 / D-7: widening is by NEW VARIANTS, not new fields (REQ-12/13 wording) |
| `Touching any of the 10 substrate files under backend/agent/src/agent/` | NFR-CPM-001 / NFR-TLS-003 |
| `Adding new top-level Go dependencies` | CH-07's `pgx/v5` + `pressly/goose/v3` cover the postgres surface; CH-10 needs no new deps |
| `A second constructor NewConversationWithPermissionPolicy` | D-1: `Config.PermissionPolicy` keeps the choice in the factory closure |
| `A dynamic policy surface (policy that changes between turns)` | NFR-CPM-002: v1 is static; v2 attaches here per ADR 0009 § D4 |
| `Surfacing Layer 2's typed Failure on Deny` | D-12: chat collapses 4 outcomes to 2; the typed Failure rides on the (suppressed) `tool.result` execution_failure arm per R-CPM-008. Failure details surface only via the explicit `tool.result{outcome: "execution_failure", failureCategory: ...}` carry |
| `Surfacing Layer 2's ModifiedArguments on ModifyInput` | D-12: chat collapses `ModifyInput` to `Defer` and drops modified arguments; future Layer-3 integration requiring modified arguments re-derives through a different path (deferred beyond CH-10) |
| `Branching / renaming / searching permission decisions` | Out of scope for permission surface; deferred |
| `A live acceptance test against a real provider` | CH-04.3 owns the only live check; CH-10 stays deterministic |
| `Repairing the citation defect noted in the "Spec defects found" section` | Recorded, not repaired (CH-00 `F-1`/`F-2`/`F-3` pattern) |

## Spec defects found (recorded, not repaired)

- **F-CPM-002/003** — resolved at design time per R-CPM-008 / S-CPM-019 above. The pattern precedent is F-CHT-9.1 (CH-09's `#3964` record) and F-CHT-9.2 (CH-09's wording amendment in `cachicamas-chat-tool-source/spec.md` § "Spec defects found"). The fix is a single switch case in `chat/projection.go` and one by-`wireCallId` correlation in `use-chat-stream.ts`; no other spec, no other file. **Recorded here as resolved, not as open** — the resolved pattern is what other future defects may mirror.

## Cross-references

- Doc 0005 § CH-10 charter (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:936-947`).
- Doc 0005 § CH-09 charter (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:923-934`) — the dependency that wired `chat.ToolSource` and the tool wire shape.
- `openspec/specs/agent-permission-protocol/spec.md` — Layer 2 protocol that CH-10 consumes (R-APP-001 per-call decision gate, R-APP-002 ack-before-wait, R-APP-003 stray-wake typed rejection, R-APP-004 AllowOnce, R-APP-005 Deny, R-APP-006 ModifyInput, R-APP-007 AllowAlways+Remember, R-APP-008 sibling isolation, R-APP-009 cancellation, R-APP-010 remembered suppression, R-APP-011 layer boundary, R-APP-012 substrate preservation).
- `backend/agent/src/agent/permission_protocol.go:80-94` — `agent.PermissionPolicy` interface that `chat.PermissionPolicy` aliases.
- `backend/agent/src/agent/permission_protocol.go:102-122` — `PermissionVerdict` shape (`Outcome`, `ModifiedArgs`, `Failure`); `PermissionDefer = 0` at `:133`.
- `backend/agent/src/agent/permission_protocol.go:58` — `ErrStrayDecision` typed sentinel (R-APP-003).
- `backend/agent/src/agent/permission_events.go:53-80` — closed `PermissionOutcome` 4-value vocabulary (`AllowOnce | AllowAlways | Deny | ModifyInput`).
- `backend/agent/src/agent/permission_events.go:184-292` — `PermissionDecisionMade` payload (chat wire collapses to 2-value vocabulary).
- `backend/agent/src/agent/scheduler.go:140` — `Schedule` signature.
- `backend/agent/src/agent/scheduler.go:284-292` — `WakeParked` typed-rejection contract.
- `backend/agent/src/agent/scheduler.go:750-1065` — `runPermissionGate` 5 verdict branches (the Deny branch at `:998-1034` is the F-CPM-002/003 source).
- `backend/agent/src/agent/scheduler.go:257-259` — `s.parked = nil` after `wg.Wait()` before `sink` close (R-APP-009 / S-APP-010 carry).
- `backend/agent/src/agent/harness.go:81` — `Harness.Scheduler *Scheduler` field (CH-10 reads, does not modify).
- `backend/agent/src/agent/harness.go:446-449` — per-Run scheduler construction (CH-09 / CH-10 pattern).
- `backend/agent/src/agent/harness.go:59` — `TurnOptions.PermissionPolicy` (CH-10 sets, Layer 2 reads).
- `backend/agent/src/chat/permission_policy.go` (new, R-CPM-001 / R-CPM-002) — `chat.PermissionPolicy` alias + `defaultPermissionPolicy` struct + `pendingDecision` map + `ErrNilPermissionPolicy` + `ErrDecisionAlreadyMade`.
- `backend/agent/src/chat/permission_decisions.go` (new, R-CPM-006) — `PermissionDecisionRecord` port-side projection.
- `backend/agent/src/chat/summarize_conversation.go` (new, R-CCS-017 / Q1) — `SummarizeConversationTool` (`EffectClassMutate`; writes via `store.UpdateSummary(p, summary)`).
- `backend/agent/src/chat/conversation.go:35-73` — `Config` struct; CH-10 adds `PermissionPolicy` field.
- `backend/agent/src/chat/conversation.go:101-141` — `NewConversation`; CH-10 widens with `Config.PermissionPolicy` rejection + `Turn.PermissionPolicy` injection.
- `backend/agent/src/chat/conversation.go` (modified, R-CPM-001 / D-11) — `Scheduler()` method-only accessor returns `c.harness.Scheduler`.
- `backend/agent/src/chat/projection.go:59-147` — closed 9-arm switch; CH-10 adds 2 new arms (`EventKindPermissionDecisionRequired` → `WireEvent.PermissionDecisionRequired`; `EventKindPermissionDecisionMade` → `WireEvent.PermissionDecisionMade` with D-12 outcome collapse inside).
- `backend/agent/src/chat/projection.go:208-251` — `buildTerminalExchange` 8-field return; CH-10 widens with `PermissionDecisions` accumulator + F-CPM-001 fix threads the tool-record accumulators.
- `backend/agent/src/chat/wire.go:13-15` — closed 9-variant `WireEvent` interface; CH-10 widens with 2 new variants.
- `backend/agent/src/chat/eventsource.go:31-48` — `wireFrameName` exhaustive switch; CH-10 adds 2 cases (total 11).
- `backend/agent/src/chat/store.go:41-50` — closed 10-field `Exchange`; CH-10 widens with `PermissionDecisions` (CH-09 already widened with `ToolCalls`/`ToolResults`).
- `backend/agent/src/chat/store.go:106-131` — closed 2-method `ConversationStore` port; CH-09 extended to N+1 via R-CCS-013; CH-10 extends to N+2 via R-CCS-017 (`UpdateSummary`).
- `backend/agent/src/chat/store.go:228-235` — `MemoryConversationStore.Append`; CH-10 widens with `PermissionDecisions` + `Summary`.
- `backend/agent/src/chat/store.go:249-264` — `MemoryConversationStore.Load`; CH-10 deep-copies `PermissionDecisions` via `copyPermissionDecisionRecords`.
- `backend/agent/src/chat/store_postgres.go` (modified, R-CPM-006 / R-CCS-017) — sibling-table INSERT/SELECT for `chat_permission_decisions`; `UPDATE chat_conversations SET summary` for `UpdateSummary`; `loadPermissionDecisions` helper.
- `backend/agent/src/chat/migrations/0003_summarize.sql` (new, R-CCS-017) — `ALTER TABLE chat_conversations ADD COLUMN summary TEXT` (nullable).
- `backend/agent/src/chat/migrations/0004_permission_decisions.sql` (new, R-CPM-006) — sibling table `chat_permission_decisions` keyed by `(participant_id, exchange_position, position)` with FK to `chat_exchanges` ON DELETE CASCADE.
- `backend/agent/src/chat/migrations/0002_tool_records.sql:60` — sibling-table template CH-10 mirrors.
- `backend/agent/src/chat/http.go:61-77` — `exchangeDTO` 10 fields; CH-10 widens with `permissionDecisions`.
- `backend/agent/src/chat/http.go:123-152` — `exchangeToDTO` defensive projection.
- `backend/agent/src/chat/http.go:193-205` — `identityMiddleware` (CH-10 rides it).
- `backend/agent/src/chat/http.go:241-293` — `HandleOpenTurn` precedent (CH-10 mirrors).
- `backend/agent/src/chat/http.go:308-358` — `HandleStreamEvents` cross-participant guard precedent (CH-10 mirrors).
- `backend/agent/src/chat/http.go` (modified, R-CPM-004) — `HandlePermissionDecision(registry, resolver)` + `RegisterPermissionRoutes(e, resolver, registry)`.
- `backend/agent/src/chat/registry.go:35-40` — `Registry` struct; CH-10 reaches `conv` via `registry.OwnerOf` + `ConversationByTurnID` patterns.
- `backend/agent/src/chat/substrate_guard_test.go` (new, NFR-CPM-001) — `TestChat_SubstrateUntouched` runs inside `make test`.
- `backend/agent/src/chat/wire_fragmentation_test.go` (new or extended, NFR-CPM-003) — `parseTranscript` covers all 11 names; KNOWN_EVENTS is exactly 11.
- `backend/agent/src/cmd/chat/main.go:227-245` — composition-root factory closure; CH-10 gains 1 `PermissionPolicy:` line.
- `backend/agent/src/cmd/chat/main.go:261-275` — `RegisterRoutes` + `RegisterResumeRoutes`; CH-10 calls `RegisterPermissionRoutes(e, resolver, registry)` after `RegisterResumeRoutes`.
- `frontend/src/lib/chat-types.ts:8-46` — closed 9-variant `ChatStreamEvent` union; REQ-7 forbids new fields on existing variants; CH-10 widens with 2 new variants (REQ-12, REQ-13 in `frontend-chat-layer1` amendment).
- `frontend/src/lib/chat-types.ts:166-182` — `ExchangeDTO` 10 fields; CH-10 widens with `permissionDecisions?: readonly PermissionDecisionDTO[]`.
- `frontend/src/lib/chat-types.ts:233-235` — `assertNever` compile-time exhaustiveness probe.
- `frontend/src/lib/chat-types.ts:284-360` — `parseTranscript` switch; CH-10 extends with 2 new cases.
- `frontend/src/lib/chat-api.ts:285-291` — `KNOWN_EVENTS` 9 names; CH-10 extends to 11.
- `frontend/src/lib/chat-api.ts:205-265` — `submitTurn`/`cancelTurn` precedent; CH-10 adds `submitPermissionDecision(turnID, callID, outcome)`.
- `frontend/src/lib/mock/chat.ts:42` — 4-state tool union `"running" | "done" | "denied" | "failed"`; `"denied"` reachable only via the `hold` row correlation (F-CPM-002/003).
- `frontend/src/components/chat/transcript-line.tsx:120-161` — `tool` variant branch.
- `frontend/src/components/chat/transcript-line.tsx:163-230` — `hold` variant branch (already implemented from CH-05; CH-10 wires `onDecide$`).
- `frontend/src/components/chat/chat-app.tsx:93-152` — `exchangesToEntries`; CH-10 extends with `hold` entry walking.
- `frontend/src/components/chat/chat-app.tsx:341-349` — `<TranscriptLine>` render loop; CH-10 passes `onDecide$` for hold entries.
- `frontend/src/components/chat/use-chat-stream.ts:200-307` — `subscribeTurn` callback switch; CH-10 adds 2 new permission cases + by-`wireCallId` correlation for F-CPM-002/003.
- `openspec/specs/chat-conversation-store/spec.md:9, 14` — identifier-append-only rule.
- `openspec/specs/chat-conversation-store/spec.md:307-313` — CH-08 additive widening precedent (R-CCS-013/014); CH-09 mirrored (R-CCS-015/016, NFR-CCS-008); CH-10 mirrors (R-CCS-017/018, NFR-CCS-009).
- `openspec/specs/frontend-chat-layer1/spec.md:83-94` — REQ-7 closed-union wording preserved verbatim (CH-10 does NOT modify).
- `openspec/specs/frontend-chat-layer1/spec.md:154-157` — CH-08 amendment header format precedent.
- `openspec/specs/cachicamas-chat-tool-source/spec.md` — CH-09 mirror spec (R-CTS-001..008, S-CTS-001..024, NFR-CTS-001..003) — the template this spec mirrors.
- `openspec/specs/cachicamas-chat-tool-source/spec.md:111-150` — CH-09 "Spec defects found" section pattern.
- `openspec/AGENTS.md` "Substrate preservation in `backend/agent` (NFR-TLS-003)" — the 10-file list CH-10 leaves byte-unchanged; CH-10 appends its one-line pointer here per the CH-07 / CH-08 / CH-09 convention.
- `docs/adr/0010-add-pgx-and-goose-to-backend-agent.md` — ADR justifying the postgres deps CH-07 admitted; CH-10 inherits without widening.
- engram observations `#3987`, `#3988` — CH-10 proposal (split across two observations for the 50K-char limit).
- engram observation `#3983` — CH-10 locked decisions (G1-G8, F-CPM-001, Q1, F-CPM-002/003).
- engram observation `#3985` — CH-10 explore report (9 sections; 16 areas mapped).
- engram observation `#3964` — F-CHT-9.1 spec defect resolution pattern (the precedent F-CPM-002/003 follows).
- engram observation `#3945` — apply-progress + commit discipline.
- engram observation `#3974` — CH-09 verify report RE-RUN pattern (4 UNTESTED scenarios caught; CH-10 mirror risk applies).