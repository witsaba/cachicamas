/**
 * chat-api.ts — the chat layer 1 typed wire client.
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md
 *   §3 (D1 SSE wire format, D2 HTTP shape, D3 auth, D4 honest offline)
 *   §4 (state flow — POST then EventSource, turn.end closes,
 *        unmount issues DELETE with keepalive).
 *
 * Three responsibilities, exported individually so the hook
 * (use-chat-stream.ts) and the window (chat-window.tsx) can each
 * hold the lifecycle they care about:
 *
 *   - submitTurn(req)  → POST /api/agent/turns (REQ-1)
 *   - cancelTurn(req)  → DELETE /api/agent/turns/:id with keepalive (REQ-2)
 *   - subscribeTurn(url, onEvent, onError?)
 *                       → EventSource(url, {withCredentials:true}); dispatches
 *                          typed `message.<x>` events via addEventListener
 *                          (design §3 D1 refinement); turn.end closes the
 *                          stream exactly once.
 *
 * Error envelope parity (REQ-4, REQ-5):
 *   - HTTP error responses map to ApiResult<T> via the same five-kind
 *     discriminated union as lib/api.ts:96-110 (validation/conflict/
 *     not_found/server/offline).
 *   - Network errors map to { ok: false, kind: "offline", message }
 *     where the message is the browser-portable dev-honest phrase
 *     (replaced by CH-05.2 — see openspec/specs/frontend-chat-layer1
 *     REQ-5 amendment).
 *   - EventSource onerror before any message.delta surfaces the same
 *     offline phrase via the onError callback (REQ-5 S-5.b).
 *
 * Frontend-only ownership (per proposal §2 hard constraints):
 *   - NO provider / model / credential fields.
 *   - NO env vars for LLM selection.
 *   - NO new HTTP-client libs (we reuse stateChangingFetch from
 *     lib/csrf.ts and the EventSource browser global).
 *   - NO backend code; this file is a consumer of the wire CH-04
 *     mounted (backend/agent/src/chat/http.go:305-307).
 */

import { stateChangingFetch } from "./csrf";
import type {
  ApiResult,
} from "./api";
import {
  type ChatCancelRequest,
  type ChatStreamError,
  type ChatStreamEvent,
  type ChatTurnRequest,
  type ChatTurnResponse,
} from "./chat-types";

// Re-export the discriminated union types this module's API returns so
// callers can `import type { ChatTurnError } from "~/lib/chat-api"`.
// (The errors are typed identically to lib/api.ts's ApiResult error
// half — we mirror the shape, not the source.)
export type { ApiResult };
export type { ChatCancelRequest, ChatStreamError, ChatStreamEvent, ChatTurnRequest, ChatTurnResponse };

// ---------------------------------------------------------------------------
// SSR-vs-browser base URL (mirrors lib/api.ts:133-157 + lib/prompts-api.ts:85-102)
// ---------------------------------------------------------------------------

const DEFAULT_BASE_URL = "http://localhost:8080";

function isNode(): boolean {
  return (
    typeof process !== "undefined" &&
    typeof (process as { versions?: unknown }).versions !== "undefined"
  );
}

function apiBaseUrl(): string {
  if (isNode()) {
    const fromEnv = process.env.SERVER_API_BASE_URL;
    return (
      fromEnv && fromEnv.trim().length > 0 ? fromEnv : DEFAULT_BASE_URL
    ).replace(/\/+$/, "");
  }
  // Browser: Vite-inlined PUBLIC_* env at build time.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const fromEnv = (import.meta as any).env?.PUBLIC_API_BASE_URL as
    | string
    | undefined;
  return (
    fromEnv && fromEnv.trim().length > 0 ? fromEnv : DEFAULT_BASE_URL
  ).replace(/\/+$/, "");
}

// Build the full URL for a relative path. SSR uses the absolute base
// URL; browser uses the same base URL (relative paths work too, but
// we always go absolute to keep the chat client identical to
// lib/api.ts's serverAwareFetch and lib/prompts-api.ts's safeFetch).
function fullUrl(path: string): string {
  if (path.startsWith("http")) return path;
  return `${apiBaseUrl()}${path.startsWith("/") ? path : `/${path}`}`;
}

// ---------------------------------------------------------------------------
// REQ-5 — dev-honest offline phrase (amended by CH-05.2)
// ---------------------------------------------------------------------------
//
// CH-05.2 retires the original literal phrasing (mandated by the
// REQ-5 promoted spec; see openspec/specs/frontend-chat-layer1 REQ-5
// amendment for the discharge rationale). The amended REQ-5 keeps
// the kind:"offline" arm (D-1) and the dev-honest contract — a
// network failure is greppable, never silently retried, never
// fabricated — but replaces the phrase with one that points a
// developer at the right command (`docker compose up`) without
// claiming a backend is unwired when one now is.
//
// The `err` parameter is ignored (D-2 — browsers differ; explore
// §7.1). The phrase is browser-portable across Chrome / Firefox /
// Safari.
// ---------------------------------------------------------------------------
const OFFLINE_GENERIC =
  "Couldn't reach the chat service. Is docker compose up? (network error)";

function offlineMessage(_err: unknown): string {
  return OFFLINE_GENERIC;
}

// ---------------------------------------------------------------------------
// Error envelope parsing (mirrors lib/api.ts envelopeToResult pattern)
// ---------------------------------------------------------------------------

async function envelopeToResult<T>(
  res: Response,
  parseBody: () => Promise<T>,
): Promise<ApiResult<T>> {
  if (res.ok) {
    return { ok: true as const, value: await parseBody() };
  }
  let body: Record<string, unknown> = {};
  try {
    body = (await res.json()) as Record<string, unknown>;
  } catch {
    // Body was not JSON — fall through with defaults.
  }
  const err = body.error as string | undefined;
  const message = (body.message as string | undefined) ?? undefined;
  // 400/422 with error="validation" both map to kind=validation
  if ((res.status === 400 || res.status === 422) && err === "validation") {
    const fields = (body.fields ?? {}) as Record<string, string>;
    const firstEntry = Object.entries(fields)[0];
    const synthesized = firstEntry
      ? `${firstEntry[0]}: ${firstEntry[1]}`
      : (message ?? "Invalid request body.");
    return {
      ok: false,
      kind: "validation",
      message: synthesized,
      fields,
    };
  }
  if (res.status === 409 && err === "conflict") {
    return {
      ok: false,
      kind: "conflict",
      message: message ?? "Conflict.",
    };
  }
  if (res.status === 404 && err === "not_found") {
    return {
      ok: false,
      kind: "not_found",
      message: message ?? "Not found.",
    };
  }
  return {
    ok: false,
    kind: "server",
    message: message ?? `Server error (${res.status}).`,
  };
}

// ---------------------------------------------------------------------------
// POST /api/agent/turns (REQ-1 S-1.a)
// ---------------------------------------------------------------------------

/**
 * Open a new chat turn.
 *
 * On success resolves to { ok: true, value: { turnId, streamUrl } }.
 * The returned `streamUrl` is a RELATIVE path; pass it verbatim to
 * subscribeTurn() which prefixes apiBaseUrl().
 *
 * On network failure resolves to { ok: false, kind: "offline",
 * message: <dev-honest phrase> } (REQ-5 S-5.a amended — honest dev
 * failure, no retry, no fabricated response).
 *
 * On typed HTTP error (4xx / 5xx) resolves to the corresponding
 * ApiResult error half (REQ-4).
 */
export async function submitTurn(
  req: ChatTurnRequest,
): Promise<ApiResult<ChatTurnResponse>> {
  let res: Response;
  try {
    res = await stateChangingFetch(fullUrl("/api/agent/turns"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as { turnId?: string; streamUrl?: string };
    if (typeof body.turnId !== "string" || typeof body.streamUrl !== "string") {
      throw new Error("chat-api: malformed turn response (turnId/streamUrl missing)");
    }
    return { turnId: body.turnId, streamUrl: body.streamUrl };
  });
}

// ---------------------------------------------------------------------------
// DELETE /api/agent/turns/:id (REQ-2 S-2.a / S-2.b)
// ---------------------------------------------------------------------------

/**
 * Cancel an in-flight turn.
 *
 * Fire-and-forget: returns ApiResult<null> for completeness; the
 * caller typically does NOT await cancel (REQ-2 S-2.b — input
 * returns to "ready" state immediately).
 *
 * The fetch is sent with `keepalive: true` so the browser fires it
 * even if the page is being torn down (REQ-2 S-2.a — useVisibleTask$
 * cleanup must issue the DELETE before unmount completes).
 */
export async function cancelTurn(
  req: ChatCancelRequest,
): Promise<ApiResult<null>> {
  let res: Response;
  try {
    res = await stateChangingFetch(fullUrl(`/api/agent/turns/${encodeURIComponent(req.id)}`), {
      method: "DELETE",
      keepalive: true,
    });
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(res, async () => null);
}

// ---------------------------------------------------------------------------
// EventSource subscription (REQ-1 S-1.a, REQ-4 S-4.a, REQ-5 S-5.b)
// ---------------------------------------------------------------------------

/**
 * Subscribed handler for chat stream events.
 */
export type ChatStreamEventHandler = (ev: ChatStreamEvent) => void;

/**
 * Offline error handler — called when the EventSource errors before
 * any message.delta is observed (REQ-5 S-5.b). The string is the
 * amended dev-honest phrase so the UI can render the typed message.
 */
export type ChatOfflineHandler = (message: string) => void;

// The full set of typed `event:` names the chat client knows. Any
// frame with an event name outside this list is dropped (REQ-1 S-1.b).
const KNOWN_EVENTS = [
  "message.start",
  "message.delta",
  "message.end",
  "turn.end",
  "error",
] as const;

type KnownEventName = (typeof KNOWN_EVENTS)[number];

/**
 * Subscribe to a chat stream.
 *
 * Opens `new EventSource(streamUrl, { withCredentials: true })` —
 * browser only; SSR returns a no-op unsubscribe (the EventSource
 * global is undefined in Node).
 *
 * Dispatches each frame through `addEventListener(name, ...)` so the
 * typed `Event` parameter is parsed server-side. The parser is a
 * thin wrapper around `parseTranscript` (chat-types.ts) that handles
 * a single frame's payload; multi-frame chunks are not produced by
 * the browser's EventSource (one event per onmessage/on<EventName>).
 *
 * Closes on `turn.end` exactly once (REQ-1 S-1.a).
 *
 * Surfaces a mid-stream `error` frame as the typed ChatStreamError
 * variant (REQ-4 S-4.a).
 *
 * Returns a function that closes the EventSource. Caller must invoke
 * it on unmount AND on user-initiated Stop.
 */
export function subscribeTurn(
  streamUrl: string,
  onEvent: ChatStreamEventHandler,
  onError?: ChatOfflineHandler,
): () => void {
  if (typeof EventSource === "undefined") {
    // SSR / unsupported — caller falls back to no-op.
    return () => {};
  }
  const url = streamUrl.startsWith("http")
    ? streamUrl
    : fullUrl(streamUrl);
  const es = new EventSource(url, { withCredentials: true });

  let intentionalClose = false;
  let sawAnyDelta = false;

  const handle = (name: string, raw: string) => {
    if (intentionalClose) return;
    let payload: unknown;
    try {
      payload = JSON.parse(raw);
    } catch {
      // Malformed JSON — drop the frame, mirror parseSSEResponse.
      return;
    }
    if (!payload || typeof payload !== "object") return;
    const obj = payload as Record<string, unknown>;
    switch (name as KnownEventName) {
      case "message.start":
        onEvent({
          kind: "message.start",
          messageId: String(obj.messageId ?? ""),
          index: Number(obj.index ?? 0),
        });
        sawAnyDelta = true;
        return;
      case "message.delta":
        onEvent({
          kind: "message.delta",
          index: Number(obj.index ?? 0),
          delta: String(obj.delta ?? ""),
        });
        sawAnyDelta = true;
        return;
      case "message.end":
        onEvent({
          kind: "message.end",
          index: Number(obj.index ?? 0),
          finishReason: (obj.finishReason ?? "unknown") as ChatStreamEvent extends {
            kind: "message.end";
            finishReason: infer R;
          }
            ? R
            : never,
        });
        sawAnyDelta = true;
        return;
      case "turn.end":
        onEvent({
          kind: "turn.end",
          ...(obj.usage ? { usage: obj.usage as ChatStreamEvent extends { kind: "turn.end"; usage?: infer U } ? U : never } : {}),
          ...(obj.finishReason
            ? {
                finishReason: obj.finishReason as ChatStreamEvent extends {
                  kind: "turn.end";
                  finishReason?: infer R;
                }
                  ? R
                  : never,
              }
            : {}),
        });
        intentionalClose = true;
        es.close();
        return;
      case "error":
        onEvent({
          kind: "error",
          error: obj as unknown as ChatStreamError,
        });
        sawAnyDelta = true;
        return;
      default:
        // REQ-1 S-1.b — unknown event names are ignored.
        return;
    }
  };

  // Typed event listeners (D1 refinement): register one per known
  // event name so the runtime dispatch matches the typed
  // ChatStreamEvent union.
  for (const name of KNOWN_EVENTS) {
    es.addEventListener(name, (ev) =>
      handle(name, (ev as MessageEvent<string>).data),
    );
  }

  // Connection-level error handler. If we haven't seen any delta
  // yet AND the user-supplied onError exists, surface the typed
  // offline phrase (REQ-5 S-5.b). If we HAVE seen deltas, the
  // stream-level error event already fired (case "error" above) and
  // this is a reconnect-after-error — call onError if provided so
  // the caller can choose to recover.
  es.addEventListener("error", () => {
    if (intentionalClose) return;
    if (!sawAnyDelta && onError) {
      onError(offlineMessage(new Error("connection error")));
    } else if (onError) {
      onError(offlineMessage(new Error("connection error")));
    }
  });

  return () => {
    intentionalClose = true;
    es.close();
  };
}
