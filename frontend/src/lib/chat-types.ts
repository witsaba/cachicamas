/**
 * chat-types.ts — typed wire contracts for the chat layer 1 consumer.
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md §2.
 *   The discriminated union, request/response shapes, error envelopes,
 *   session model, and fixture interface are pinned verbatim from the
 *   design contract. Any change to a field name or discriminator value
 *   here is a contract change — update design.md and the spec together.
 *
 * Scope of this file (REQ-7.a):
 *   "Each `*.ts` and `*.tsx` file under frontend/src/lib/chat-api.ts
 *    and frontend/src/components/chat/ SHALL have a colocated
 *    `*.spec.ts` / `*.spec.tsx` that asserts at least one
 *    Given/When/Then scenario from REQ-1 .. REQ-6."
 *
 *   The exhaustive coverage of the union lives in chat-api.spec.ts
 *   (T-01), which imports these types and asserts assertNever()
 *   narrows every variant. Adding a new variant to ChatStreamEvent
 *   without updating that switch is a TypeScript compile error.
 */

// REQ-1 — discriminated stream event union. The variants match
// design.md §2; the order matches doc 0001 §4.3 message lifecycle
// (start → deltas → end → terminal). Doc 0001 §4.3 invariant 1
// (deltas carry an index, consumers accumulate) is honored by the
// `index` field on message.* variants.
//
// CH-09 (REQ-8..11, D-3, D-7) — four new variants widen the union
// additively. REQ-7's spec text forbids NEW FIELDS ON EXISTING
// VARIANTS; CH-09 adds NEW VARIANTS to the closed union. Each new
// variant carries explicit "D-7: new variant, not new field"
// documentation in the frontend-chat-layer1 spec amendment. The
// wire-side mirror at chat/wire.go (Go) and the parseTranscript
// switch below cover the same field shapes; `assertNever` in
// the default branch surfaces a compile error if a future variant
// is added without a `case`.
//
// CH-10 (REQ-12, REQ-13, D-3, D-7) — two new variants widen the
// union for the permission event family. The wire carries
// `permission.decision.required` (the ask) and
// `permission.decision.made` (the answer) — both NEW VARIANTS on
// the closed union, not new fields on existing variants (the
// D-7 explicit wording carried into REQ-12/13). The chat wire's
// CLOSED 2-value Outcome vocabulary "allow_once" | "deny" (D-12
// collapse of Layer 2's 4-value PermissionOutcome) lands in the
// made variant.
export type ChatStreamEvent =
  | { readonly kind: "message.start"; readonly messageId: string; readonly index: number }
  | { readonly kind: "message.delta"; readonly index: number; readonly delta: string }
  | { readonly kind: "message.end"; readonly index: number; readonly finishReason: ChatFinishReason }
  | { readonly kind: "turn.end"; readonly usage?: ChatUsage; readonly finishReason?: ChatFinishReason }
  | { readonly kind: "error"; readonly error: ChatStreamError }
  | { readonly kind: "tool.call.start"; readonly wireCallId: string; readonly tool: string; readonly arguments: string }
  | { readonly kind: "tool.call.delta"; readonly wireCallId: string; readonly delta: string }
  | { readonly kind: "tool.call.end"; readonly wireCallId: string; readonly outcome: string }
  | { readonly kind: "tool.result"; readonly wireCallId: string; readonly tool: string; readonly outcome: ToolResultOutcome; readonly content: string; readonly failureCategory: string }
  | { readonly kind: "permission.decision.required"; readonly wireCallId: string; readonly tool: string; readonly arguments: string }
  | { readonly kind: "permission.decision.made"; readonly wireCallId: string; readonly outcome: "allow_once" | "deny" };

// AI-13.1 seven-value finish-reason vocabulary (design §2). The UI
// surfaces these as plain labels in v1.
export type ChatFinishReason =
  | "stop"
  | "length"
  | "tool_calls"
  | "refusal"
  | "pause_turn"
  | "content_filter"
  | "unknown";

// REQ-4 — token counts deferred beyond input/output; cost-display is
// doc 0001 §7 G10 and out of scope for v1. The seam is preserved so a
// future cost-display change touches the type, not the wire.
export interface ChatUsage {
  readonly inputTokens?: number;
  readonly outputTokens?: number;
}

// REQ-4 — typed mid-stream error. Mirrors the four non-offline
// ApiError kinds (lib/api.ts:96-110) verbatim. The `offline` kind
// is reserved for transport failures (REQ-5) and is NOT carried
// in ChatStreamError because the server cannot send a network
// error — it only arrives when the client itself times out or
// fails to parse.
export interface ChatStreamError {
  readonly kind: "validation" | "conflict" | "not_found" | "server";
  readonly message: string;
  readonly fields?: Record<string, string>;
}

// D2 — POST body. Frontend MUST NOT carry provider/model/credential
// keys (user: "the back is who controls the keys"). `id` is a
// client-minted UUID used as the cancellation key for DELETE.
export interface ChatTurnRequest {
  readonly id: string;
  readonly prompt: string;
  /** v1: ignored by backend; reserved seam. */
  readonly systemHint?: string;
}

// D2 — POST response. Backend returns the URL of the SSE stream
// (relative path; client appends to apiBaseUrl()).
export interface ChatTurnResponse {
  readonly turnId: string;
  readonly streamUrl: string;
}

// REQ-2 — cancel is a discrete wire signal (DELETE), not just
// EventSource.close() (doc 0001 §4.2 cancellation tree).
export interface ChatCancelRequest {
  readonly id: string;
}

// REQ-4 + REQ-5 (amended) — union of offline-path + typed-API errors.
// Mirrors lib/api.ts:96-110 ApiResult's error half verbatim so the
// chat's error surface is indistinguishable from the rest of the
// frontend. The offline kind survives (D-1); the dev-honest phrase
// was amended by CH-05.2 (see
// openspec/specs/frontend-chat-layer1 REQ-5 amendment).
export type ChatTurnError =
  | { readonly kind: "offline"; readonly message: string }
  | { readonly kind: "validation"; readonly message: string; readonly fields: Record<string, string> }
  | { readonly kind: "conflict"; readonly message: string }
  | { readonly kind: "not_found"; readonly message: string }
  | { readonly kind: "server"; readonly message: string };

// useChatStream signal-store shapes (design §2). The hook owns a
// useStore<{messages: ChatMessage[]; ...}>(...) for the session and
// per-bubble text is held in a useSignal<string> so Qwik's signal
// reactivity can re-render only the bubble whose text changed.
export interface ChatMessage {
  readonly id: string;
  readonly role: "user" | "assistant";
  text: string;
  status: "pending" | "streaming" | "complete" | "error";
  error?: ChatStreamError;
}

export interface ChatSession {
  messages: ChatMessage[];
  status: "idle" | "submitting" | "streaming" | "cancelling";
  currentTurnId?: string;
}

// REQ-7 — typed fixture so vitest can deep-assert against recorded
// wire bytes. `raw` is the byte-identical SSE transcript; the test
// asserts parseTranscript(fixture) deep-equals `expectedEvents`.
// Any wire-shape drift surfaces as a fixture-first spec failure.
export interface TranscriptFixture {
  readonly name: string;
  readonly raw: string;
  readonly expectedEvents: readonly ChatStreamEvent[];
  readonly expectTerminalClose: boolean;
}

// CH-08 (REQ-8, R-CRI-004) — read-side wire types.
//
// These types back GET /api/agent/conversations/:id and
// GET /api/agent/conversations. They are NEW TYPES adjacent to the
// closed ChatStreamEvent union (REQ-7 preserved); the union is
// NOT widened. The DTOs are pure transport projections — the wire
// MUST NOT invent fields beyond what the backend port's types
// carry (chat.ConversationSummary + chat.Exchange, both
// field-for-field with the D7 / R-CRI-004 contract).
//
// `ConversationSummary` carries the list endpoint's row. The page
// surfaces `id`, a relative-time `age` derived from `lastActivityAt`,
// and `turnCount`. `conversationID` doubles as the participant id
// (D-1: one conversation per participant; the schema's PK is
// participant_id), so the wire re-uses the participant id field
// name with the wire-natural `conversationID` spelling.
//
// `ExchangeDTO` mirrors chat.Exchange's eight fields per D-7 —
// position, promptText, assistantText, partial, terminalKind,
// failureCategory, finishReason (optional), and messageIDs. The
// page reads this DTO into `useChatStream.reset(entries)` to seed
// the transcript on first paint (REQ-8).
export interface ExchangeDTO {
  readonly position: number;
  readonly promptText: string;
  readonly assistantText: string;
  readonly partial: boolean;
  readonly terminalKind: "completed" | "cancelled" | "failed";
  readonly failureCategory: string;
  readonly finishReason?: string;
  readonly messageIDs: readonly string[];
  // CH-09 (R-CCS-015, S-CTS-019) — two new optional fields widen
  // the read-side wire projection additively. The Go mirror
  // (backend/agent/src/chat/http.go:exchangeDTO) emits the same
  // shape with lowercase JSON keys. A tool-free turn omits both
  // keys (`omitempty`); an empty array serializes as `[]`.
  readonly toolCalls?: readonly ToolCallDTO[];
  readonly toolResults?: readonly ToolResultDTO[];
  // CH-10 (R-CPM-006, R-CCS-018) — third new optional field widens
  // the read-side wire projection additively (NFR-CPM-004 carry).
  // A turn without permission activity omits the key entirely
  // (`omitempty`); an empty array serializes as `[]`.
  readonly permissionDecisions?: readonly PermissionDecisionDTO[];
}

export interface ConversationSummary {
  readonly conversationID: string;
  readonly lastActivityAt: string;
  readonly turnCount: number;
  // CH-10 (R-CCS-017) — optional summary string written by
  // SummarizeConversationTool via ConversationStore.UpdateSummary.
  // Missing when UpdateSummary has not been called; omitempty on
  // the Go side (chat_conversations.summary NULL after the
  // forward-only ADD COLUMN migration lands, before any update).
  readonly summary?: string;
}

// CH-09 — wire projections of chat.ToolCallRecord / chat.ToolResultRecord
// (frontend mirror of the chat package's port-side types). Adjacent
// to the closed `ChatStreamEvent` union; never widening it. JSON
// keys lowercase per the closed `ExchangeDTO` precedent.
export interface ToolCallDTO {
  readonly wireCallId: string;
  readonly tool: string;
  readonly arguments: string;
}

// ToolResultOutcome is the closed three-value vocabulary mirroring
// agent.ToolOutcome (backend/agent/src/agent/tool_event.go:227-246)
// projected to wire-shape strings. execution_failure carries a
// typed failure category (R-CCP-008 / D6 mirror — no provider text).
export type ToolResultOutcome = "success" | "result_failure" | "execution_failure";

export interface ToolResultDTO {
  readonly wireCallId: string;
  readonly tool: string;
  readonly outcome: ToolResultOutcome;
  readonly content: string;
  // Non-empty ONLY when outcome === "execution_failure" (typed
  // category). Mirrors R-CCP-008 / D6: provider text MUST NOT
  // reach the wire.
  readonly failureCategory: string;
}

// CH-10 (R-CPM-006, R-CCS-018) — wire projection of
// chat.PermissionDecisionRecord (chat package's port-side type).
// Adjacent to the closed `ChatStreamEvent` union; never widening
// it. The Outcome is the chat wire's CLOSED 2-value vocabulary
// "allow_once" | "deny" (D-12 collapse). The Projection stores the
// COLLAPSED form — Layer 2's 4-value PermissionOutcome reduces to
// 2 at the chat wire BEFORE the records reach the store.
export interface PermissionDecisionDTO {
  readonly wireCallId: string;
  readonly tool: string;
  readonly outcome: "allow_once" | "deny";
}

/**
 * Exhaustiveness probe for the discriminated union.
 *
 * Usage in a switch's default branch:
 *   switch (ev.kind) {
 *     case "message.start": ...
 *     default: assertNever(ev);
 *   }
 *
 * If a new variant is added to ChatStreamEvent (or any other union
 * with assertNever probes), TypeScript reports
 *   `Type '"new-variant"' is not assignable to type 'never'`
 * on the `assertNever(ev)` call site — compile-time guarantee
 * that every variant is handled. Runtime: throws so a future
 * dynamic-dispatch regression surfaces immediately.
 */
export function assertNever(x: never): never {
  throw new Error(`Non-exhaustive switch: ${JSON.stringify(x)}`);
}

/**
 * Decode a raw SSE transcript into typed ChatStreamEvent values.
 *
 * This is the wire-shape mirror of the typed EventSource listeners
 * in chat-api.ts (which register `message.<x>` event names on the
 * browser global). The runtime path uses addEventListener for live
 * streams; this pure helper exists so vitest can assert the
 * recorded fixture bytes parse into the expected typed sequence
 * WITHOUT a real EventSource (REQ-7 property-style over bytes).
 *
 * Wire shape (per design.md §3 D1 refinement):
 *   frame 1: event: message.start\ndata: {"messageId":"...","index":0}\n\n
 *   frame N: event: message.delta\ndata: {"index":0,"delta":"..."}\n\n
 *   ...    : event: message.end\ndata: {"index":0,"finishReason":"stop"}\n\n
 *   last   : event: turn.end\ndata: {"usage":{...},"finishReason":"stop"}\n\n
 *
 * S-1.b — frames with an `event:` name outside the known set are
 * silently dropped (the buffer is unchanged for those).
 *
 * JSON parse errors on a data: line are also silently dropped —
 * the parser mirrors parseSSEResponse (lib/api.ts:851-881) in
 * treating malformed data as a no-op frame.
 */
export function parseTranscript(raw: string): ChatStreamEvent[] {
  const events: ChatStreamEvent[] = [];
  const frames = raw.split("\n\n");
  for (const frame of frames) {
    if (frame.length === 0) continue;
    let eventName = "";
    let dataLine = "";
    let hasData = false;
    for (const line of frame.split("\n")) {
      if (line.startsWith("event: ")) {
        eventName = line.slice("event: ".length).trim();
      } else if (line.startsWith("data: ")) {
        dataLine = line.slice("data: ".length);
        hasData = true;
      }
      // comment lines (": keepalive") are ignored — they don't carry a typed event
    }
    if (!eventName || !hasData) continue;
    let payload: Record<string, unknown>;
    try {
      payload = JSON.parse(dataLine) as Record<string, unknown>;
    } catch {
      continue;
    }
    switch (eventName) {
      case "message.start":
        events.push({
          kind: "message.start",
          messageId: String(payload.messageId ?? ""),
          index: Number(payload.index ?? 0),
        });
        break;
      case "message.delta":
        events.push({
          kind: "message.delta",
          index: Number(payload.index ?? 0),
          delta: String(payload.delta ?? ""),
        });
        break;
      case "message.end":
        events.push({
          kind: "message.end",
          index: Number(payload.index ?? 0),
          finishReason: (payload.finishReason ?? "unknown") as ChatFinishReason,
        });
        break;
      case "turn.end":
        events.push({
          kind: "turn.end",
          ...(payload.usage ? { usage: payload.usage as ChatUsage } : {}),
          ...(payload.finishReason
            ? { finishReason: payload.finishReason as ChatFinishReason }
            : {}),
        });
        break;
      case "error":
        events.push({
          kind: "error",
          error: payload as unknown as ChatStreamError,
        });
        break;
      // CH-09 (REQ-8..11, D-3, D-6) — four new wire event names. The
      // "tool.call.delta" and "tool.call.end" cases are reserved
      // (NFR-CTS-002); v1's chat projector never emits them. parseTranscript
      // admits them so a future long-running MCP tool can land here
      // without a frontend wire change.
      case "tool.call.start":
        events.push({
          kind: "tool.call.start",
          wireCallId: String(payload.wireCallId ?? ""),
          tool: String(payload.tool ?? ""),
          arguments: String(payload.arguments ?? ""),
        });
        break;
      case "tool.call.delta":
        events.push({
          kind: "tool.call.delta",
          wireCallId: String(payload.wireCallId ?? ""),
          delta: String(payload.delta ?? ""),
        });
        break;
      case "tool.call.end":
        events.push({
          kind: "tool.call.end",
          wireCallId: String(payload.wireCallId ?? ""),
          outcome: String(payload.outcome ?? ""),
        });
        break;
      case "tool.result":
        events.push({
          kind: "tool.result",
          wireCallId: String(payload.wireCallId ?? ""),
          tool: String(payload.tool ?? ""),
          outcome: (payload.outcome ?? "success") as ToolResultOutcome,
          content: String(payload.content ?? ""),
          failureCategory: String(payload.failureCategory ?? ""),
        });
        break;
      // CH-10 (REQ-12, REQ-13, D-3) — two new wire event names
      // for the permission event family. The wire's CLOSED 2-value
      // Outcome vocabulary "allow_once" | "deny" is the source of
      // truth (D-12 collapse from Layer 2's 4-value
      // PermissionOutcome). The frontend mirror at chat/wire.go
      // (Go) emits the same field names.
      case "permission.decision.required":
        events.push({
          kind: "permission.decision.required",
          wireCallId: String(payload.wireCallId ?? ""),
          tool: String(payload.tool ?? ""),
          arguments: String(payload.arguments ?? ""),
        });
        break;
      case "permission.decision.made":
        events.push({
          kind: "permission.decision.made",
          wireCallId: String(payload.wireCallId ?? ""),
          outcome: ((payload.outcome ?? "deny") as "allow_once" | "deny"),
        });
        break;
      default:
        // S-1.b — unknown event names are ignored.
        break;
    }
  }
  return events;
}
