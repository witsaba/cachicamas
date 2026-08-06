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
export type ChatStreamEvent =
  | { readonly kind: "message.start"; readonly messageId: string; readonly index: number }
  | { readonly kind: "message.delta"; readonly index: number; readonly delta: string }
  | { readonly kind: "message.end"; readonly index: number; readonly finishReason: ChatFinishReason }
  | { readonly kind: "turn.end"; readonly usage?: ChatUsage; readonly finishReason?: ChatFinishReason }
  | { readonly kind: "error"; readonly error: ChatStreamError };

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

// REQ-4 + REQ-5 — union of offline-path + typed-API errors.
// Mirrors lib/api.ts:96-110 ApiResult's error half verbatim so the
// chat's error surface is indistinguishable from the rest of the
// frontend. The literal `"backend not wired — see PR for backend
// wire"` is greppable per REQ-5.
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
      default:
        // S-1.b — unknown event names are ignored.
        break;
    }
  }
  return events;
}
