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
