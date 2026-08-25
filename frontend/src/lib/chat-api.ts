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
  type ConversationSummary,
  type ExchangeDTO,
} from "./chat-types";

// Re-export the discriminated union types this module's API returns so
// callers can `import type { ChatTurnError } from "~/lib/chat-api"`.
// (The errors are typed identically to lib/api.ts's ApiResult error
// half — we mirror the shape, not the source.)
export type { ApiResult };
export type {
  ChatCancelRequest,
  ChatStreamError,
  ChatStreamEvent,
  ChatTurnRequest,
  ChatTurnResponse,
  ConversationSummary,
  ExchangeDTO,
};

// ---------------------------------------------------------------------------
// SSR-vs-browser base URL.
//
// DELIBERATELY DIFFERENT from lib/api.ts / lib/prompts-api.ts: the chat
// wire paths passed to fullUrl() are ALREADY backend-absolute
// ("/api/agent/turns", "/api/agent/conversations", and the server-issued
// streamUrl "/api/agent/turns/:id/events"). The db_admin-oriented clients
// pass UNPREFIXED paths ("/organizations") and rely on PUBLIC_API_BASE_URL
// ("= /api") to add the proxy prefix. Reusing that convention here produced
// "/api/api/agent/..." in the browser (2026-08-25 fix, fix/chat-stack-wiring).
//
// Browser: same-origin relative URLs — the Qwik Node server proxies
// /api/agent/* to the chat service (entry.express.tsx, AGENT_CHAT_TARGET).
//
// Node SSR: today no SSR path fetches chat endpoints (every call site is
// client-guarded), but if one ever does, it must target the chat service
// directly — SERVER_AGENT_CHAT_BASE_URL (compose: http://agent_chat:8080),
// falling back to the documented dev port 8090. NEVER SERVER_API_BASE_URL:
// that points at database_administrator, which does not serve /api/agent/*.
// ---------------------------------------------------------------------------

const DEFAULT_NODE_CHAT_BASE_URL = "http://localhost:8090";

function isNode(): boolean {
  return (
    typeof process !== "undefined" &&
    typeof (process as { versions?: unknown }).versions !== "undefined"
  );
}

function apiBaseUrl(): string {
  if (isNode()) {
    const fromEnv = process.env.SERVER_AGENT_CHAT_BASE_URL;
    return (
      fromEnv && fromEnv.trim().length > 0
        ? fromEnv
        : DEFAULT_NODE_CHAT_BASE_URL
    ).replace(/\/+$/, "");
  }
  // Browser: same origin. The /api/agent prefix in each path is what the
  // Qwik proxy matches on; adding a base would double-prefix it.
  return "";
}

// Build the full URL for an already-prefixed chat path. Absolute URLs
// (server-issued stream URLs) pass through untouched; relative paths get
// the base prepended (empty string in the browser → same-origin request).
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
// CH-10 POST /api/agent/turns/:id/permissions/:callID (R-CPM-004, D-10)
// ---------------------------------------------------------------------------

/**
 * CH-10 — the closed 2-value vocabulary for permission decisions
 * (D-12 collapse of Layer 2's 4-value PermissionOutcome). The
 * HTTP body carries exactly one of these strings; out-of-vocab
 * values yield 422 validation (S-CPM-013).
 */
export type PermissionDecisionOutcome = "allow_once" | "deny";

/**
 * Submit a permission decision for a parked tool (R-CPM-004).
 *
 * The participant's click on the inline `hold` row triggers this
 * call. The backend writes the verdict into the chat-owned policy
 * state, then wakes the parked gate via conv.Scheduler().WakeParked
 * (D-11). On success the parked tool resumes; the wire carries
 * the resulting `permission.decision.made` event (mirrored locally
 * by the click — the SSE event closes the hold entry).
 *
 * Responses:
 *   - 200 OK — verdict recorded; WakeParked returned nil.
 *   - 403 not_found — cross-participant (R-CHS-004.b shape).
 *   - 404 not_found — unknown callID; WakeParked returned
 *     ErrStrayDecision (parked entry was already cleared or never
 *     parked).
 *   - 409 conflict — second click on the same callID; RecordVerdict
 *     returned ErrDecisionAlreadyMade (S-CPM-017a.5).
 *   - 422 validation — outcome outside the closed 2-value vocab.
 */
export async function submitPermissionDecision(
  turnID: string,
  callID: string,
  outcome: PermissionDecisionOutcome,
): Promise<ApiResult<null>> {
	let res: Response;
	try {
		res = await stateChangingFetch(
			fullUrl(
				`/api/agent/turns/${encodeURIComponent(turnID)}/permissions/${encodeURIComponent(callID)}`,
			),
			{
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify({ outcome }),
			},
		);
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
//
// CH-09 (REQ-8..11, D-3) — four new event names. tool.call.delta and
// tool.call.end are reserved-but-unused at v1 (NFR-CTS-002 / D-6):
// the chat projector collapses Layer 2's 5-event-per-call bracket
// model into 2 chat-side events per call. The reserved variants exist
// so a future long-running MCP tool can land here without a wire
// shape change.
//
// CH-10 (REQ-12, REQ-13, NFR-CPM-003 wire-fragmentation guard) —
// two new event names. The lockstep invariant
// `KNOWN_EVENTS.length === parseTranscript switch arms === eventsource.go wireFrameName switch cases`
// is verified by S-CPM-022 (chat-api.spec.ts + wire_fragmentation_test.go).
const KNOWN_EVENTS = [
  "message.start",
  "message.delta",
  "message.end",
  "turn.end",
  "error",
  "tool.call.start",
  "tool.call.delta",
  "tool.call.end",
  "tool.result",
  "permission.decision.required",
  "permission.decision.made",
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

// ---------------------------------------------------------------------------
// CH-08 (REQ-8, R-CRI-001, R-CRI-002) — read-side wire client.
//
//   - listConversations()  → GET /api/agent/conversations
//                              returns ApiResult<ConversationSummary[]>
//   - loadConversation(id) → GET /api/agent/conversations/:id
//                              returns ApiResult<ExchangeDTO[]>
//
// Both helpers fire in parallel on the page's useVisibleTask$ mount
// (chat-app.tsx) — seed `useChatStream.reset(entries)` from the
// reload endpoint, populate the conversation rail from the list
// endpoint. Both reuse `envelopeToResult` so the five-kind
// discriminated union surface is identical to submitTurn +
// cancelTurn.
//
// Network errors map to { kind: "offline", message: <dev-honest
// phrase> } (REQ-5 S-5.a). HTTP errors map via the existing
// envelope parser. The helpers send credentials via cookie
// (same-origin; withCredentials is implicit on relative fetches
// via the EventSource path; here we set it explicitly so cross-
// origin SSR vs browser behaves identically).
// ---------------------------------------------------------------------------

/**
 * List the authenticated participant's conversations (CH-08.2).
 *
 * Returns an empty array on the success-with-no-rows path (S-CRI-004);
 * the backend emits `[]` not `null` for that case.
 *
 * @example
 *   const result = await listConversations();
 *   if (result.ok) {
 *     result.value.forEach((c) => console.log(c.conversationID, c.turnCount));
 *   }
 */
export async function listConversations(): Promise<ApiResult<ConversationSummary[]>> {
  let res: Response;
  try {
    res = await fetch(fullUrl("/api/agent/conversations"), {
      method: "GET",
      credentials: "include",
    });
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as ConversationSummary[];
    return Array.isArray(body) ? body : [];
  });
}

/**
 * Load the authenticated participant's recorded conversation
 * (CH-08.1). `id` MUST equal the authenticated participant id (the
 * backend enforces this — cross-participant requests are refused
 * as 403 not_found per R-CRI-001 / R-CHS-004.b shape; the page
 * passes its own participant id here, never a value from the URL
 * or a header).
 *
 * Returns an array of `ExchangeDTO` values in insertion order
 * (matching chat.Exchange.Position). The page reads this DTO via
 * `useChatStream.reset(entries)` to seed the transcript without
 * opening an EventSource.
 *
 * @example
 *   const result = await loadConversation(participantID);
 *   if (result.ok) {
 *     await turn.reset(result.value);
 *   }
 */
export async function loadConversation(id: string): Promise<ApiResult<ExchangeDTO[]>> {
  let res: Response;
  try {
    res = await fetch(fullUrl(`/api/agent/conversations/${encodeURIComponent(id)}`), {
      method: "GET",
      credentials: "include",
    });
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as ExchangeDTO[];
    return Array.isArray(body) ? body : [];
  });
}
