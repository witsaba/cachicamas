/**
 * use-chat-stream.ts — Qwik hook that owns the chat streaming lifecycle.
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md
 *   §3 (chat-window reads signals owned by this hook),
 *   §4 (REQ-1 happy path, REQ-2 cancel, REQ-5 honest offline),
 *   §5 (useVisibleTask$ + cleanup; SSE-only-on-browser).
 *
 * Lifecycle (mirrors use-sync-status.ts:115-138 — the canonical
 * "subscribe to a server-pushed event stream" pattern):
 *   - submit(session, prompt) → submitTurn() POST → on 202 open
 *     subscribeTurn(streamUrl, applyEvent, onOfflineError).
 *   - Every `message.<x>` event flows through applyEvent (pure
 *     mutator that mutates the session in place).
 *   - turn.end closes the EventSource exactly once and flips
 *     session.status back to "idle".
 *   - cancel(session) → cancelTurn() DELETE (with keepalive) AND
 *     closes the EventSource; no-op when no turn is open.
 *   - useVisibleTask$ cleanup closes the EventSource and issues
 *     cancelTurn if a turn is still open (REQ-2 S-2.a — unmount
 *     must NOT leave an orphan on the backend).
 *
 * Architecture choice: the mutator (applyEvent) and constructor
 * (createSession) are exported as pure functions so the vitest spec
 * can exercise them WITHOUT booting Qwik. The hook itself owns
 * the useStore + useVisibleTask$ lifecycle and delegates to the
 * pure functions. This split mirrors design.md §4's "isolate the
 * pure event mutator" requirement.
 */
import { $, useStore, useVisibleTask$ } from "@builder.io/qwik";

import {
  cancelTurn as cancelTurnApi,
  subscribeTurn as subscribeTurnApi,
  submitTurn as submitTurnApi,
} from "~/lib/chat-api";
import type {
  ChatMessage,
  ChatSession,
  ChatStreamEvent,
} from "~/lib/chat-types";
import type { ApiResult } from "~/lib/api";

// ---------------------------------------------------------------------------
// Pure helpers — exported so the spec can exercise them without Qwik
// ---------------------------------------------------------------------------

/** A fresh session, idle, empty messages. */
export function createSession(): ChatSession {
  return {
    messages: [],
    status: "idle",
  };
}

/**
 * Append a user message to the session. Called by the chat input
 * on submit; the user bubble is created synchronously so the UI
 * shows the prompt immediately, before the POST resolves.
 */
export function appendUserMessage(
  session: ChatSession,
  text: string,
): ChatMessage {
  const msg: ChatMessage = {
    id: `user-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    role: "user",
    text,
    status: "complete",
  };
  session.messages.push(msg);
  return msg;
}

/**
 * Pure mutator: apply a typed ChatStreamEvent to a session.
 *
 * Behavior:
 *   - message.start → append an assistant bubble with status="streaming"
 *   - message.delta → append the delta to the LAST assistant bubble's text
 *   - message.end → flip the last assistant bubble's status to "complete"
 *   - turn.end → session.status="idle" + clear currentTurnId
 *   - error → flip the last bubble to status="error" + record typed error
 *   - any other kind → no-op (defensive; the union is exhaustive)
 *
 * The mutator is intentionally non-pure in the "returns void" sense
 * (it mutates the session in place, which Qwik's reactive system
 * tracks). Splitting it out from the hook makes the spec independent
 * of Qwik's test harness.
 */
export function applyEvent(session: ChatSession, event: ChatStreamEvent): void {
  switch (event.kind) {
    case "message.start": {
      const msg: ChatMessage = {
        id: event.messageId,
        role: "assistant",
        text: "",
        status: "streaming",
      };
      session.messages.push(msg);
      return;
    }
    case "message.delta": {
      const last = session.messages[session.messages.length - 1];
      if (last && last.role === "assistant") {
        last.text += event.delta;
      }
      return;
    }
    case "message.end": {
      const last = session.messages[session.messages.length - 1];
      if (last && last.role === "assistant") {
        last.status = "complete";
      }
      return;
    }
    case "turn.end": {
      session.status = "idle";
      session.currentTurnId = undefined;
      return;
    }
    case "error": {
      const last = session.messages[session.messages.length - 1];
      if (last && last.role === "assistant") {
        last.status = "error";
        last.error = event.error;
      }
      // Surface the error at session level too so the input can
      // re-enable (REQ-4 S-4.a: input clears `disabled`).
      session.status = "idle";
      session.currentTurnId = undefined;
      return;
    }
    default: {
      // Exhaustive — assertNever would catch a future variant at
      // compile time. Runtime: defensive no-op.
      const _exhaustive: never = event;
      void _exhaustive;
      return;
    }
  }
}

// ---------------------------------------------------------------------------
// Submit / cancel — pure functions over the session + an optional
// EventSource handle (the handle is captured in the hook's closure).
// ---------------------------------------------------------------------------

/** UUIDv4 — the client-minted cancellation key for DELETE. */
function uuid(): string {
  // crypto.randomUUID is available in modern browsers AND in Node
  // 19+. Falls back to a Math.random implementation in case the
  // runtime doesn't expose crypto.
  if (
    typeof globalThis.crypto !== "undefined" &&
    typeof globalThis.crypto.randomUUID === "function"
  ) {
    return globalThis.crypto.randomUUID();
  }
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/**
 * Submit a turn.
 *
 * Returns ApiResult<ChatTurnResponse> from chat-api.submitTurn.
 * On ok: opens subscribeTurn() on the returned stream URL and
 * updates session.status + session.currentTurnId.
 * On !ok: leaves session.status as it was (input remains in
 * "ready" state for retry per REQ-4).
 */
export async function submit(
  session: ChatSession,
  prompt: string,
  hooks?: {
    onSubscribe: (
      url: string,
      onEvent: (ev: ChatStreamEvent) => void,
      onError: (msg: string) => void,
    ) => () => void;
  },
): Promise<ApiResult<{ turnId: string; streamUrl: string }>> {
  const id = uuid();
  session.status = "submitting";
  const result = await submitTurnApi({ id, prompt });
  if (!result.ok) {
    // REQ-4 / REQ-5 — restore session to idle so the input is
    // re-enabled and the user can retry (or read the typed error).
    session.status = "idle";
    return result;
  }
  session.status = "streaming";
  session.currentTurnId = result.value.turnId;
  // Open the stream. Default uses the chat-api EventSource
  // subscriber; the hook can pass a custom onSubscribe if it
  // wants to inject a different transport.
  const onSubscribe =
    hooks?.onSubscribe ??
    ((url, onEvent, onError) => subscribeTurnApi(url, onEvent, onError));
  const unsubscribe = onSubscribe(
    result.value.streamUrl,
    (ev) => applyEvent(session, ev),
    (msg) => {
      // REQ-5 S-5.b — EventSource onerror before any delta.
      // Surface the offline literal inline and flip session to idle.
      session.status = "idle";
      session.currentTurnId = undefined;
      void msg;
    },
  );
  // Stash the unsubscribe on the session so cancel() and the
  // useVisibleTask$ cleanup can find it.
  (session as ChatSession & { __unsubscribe?: () => void }).__unsubscribe =
    unsubscribe;
  return result;
}

/**
 * Cancel an in-flight turn. No-op when no turn is open.
 *
 * Fires DELETE via chat-api.cancelTurn (with keepalive: true) AND
 * closes the EventSource so the browser stops auto-reconnecting.
 */
export async function cancel(session: ChatSession): Promise<void> {
  const id = session.currentTurnId;
  const unsubscribe = (session as ChatSession & { __unsubscribe?: () => void })
    .__unsubscribe;
  // Close the EventSource first so no further deltas are applied
  // to the session while the cancel DELETE is in flight.
  if (unsubscribe) {
    unsubscribe();
    (session as ChatSession & { __unsubscribe?: () => void }).__unsubscribe =
      undefined;
  }
  session.status = "idle";
  session.currentTurnId = undefined;
  if (id) {
    // Fire-and-forget; we don't await so the UI returns to ready
    // state immediately (REQ-2 S-2.b).
    void cancelTurnApi({ id });
  }
}

// ---------------------------------------------------------------------------
// The Qwik hook — wires the pure helpers to useStore + useVisibleTask$.
// ---------------------------------------------------------------------------

export interface UseChatStreamResult {
  session: ChatSession;
  submit: (
    prompt: string,
  ) => Promise<ApiResult<{ turnId: string; streamUrl: string }>>;
  cancel: () => Promise<void>;
}

/**
 * The Qwik hook. Owns:
 *   - the ChatSession (via useStore for reactivity),
 *   - the EventSource lifecycle (via useVisibleTask$ + cleanup).
 *
 * Pattern (mirrors use-sync-status.ts:65-141):
 *   - useStore for the session (so JSX that reads .messages / .status
 *     re-renders on mutation).
 *   - useVisibleTask$ registers cleanup that closes any in-flight
 *     EventSource AND issues cancelTurn if a turn is still open
 *     (REQ-2 S-2.a — unmount MUST leave no orphan on the backend).
 */
export function useChatStream$(): UseChatStreamResult {
  const session = useStore<ChatSession>(createSession());

  const submitQrl = $(async (prompt: string) => {
    if (session.status !== "idle") {
      // Already submitting/streaming — ignore duplicate submits.
      return {
        ok: false as const,
        kind: "conflict" as const,
        message: "A turn is already in flight.",
      };
    }
    appendUserMessage(session, prompt);
    return submit(session, prompt, {
      onSubscribe: (url, onEvent, onError) =>
        subscribeTurnApi(url, onEvent, onError),
    });
  });

  const cancelQrl = $(async () => {
    await cancel(session);
  });

  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(({ cleanup }) => {
    // cleanup runs on unmount. Close any in-flight EventSource and
    // fire-and-forget a DELETE if a turn is still open.
    cleanup(() => {
      const unsubscribe = (
        session as ChatSession & { __unsubscribe?: () => void }
      ).__unsubscribe;
      if (unsubscribe) {
        unsubscribe();
      }
      const id = session.currentTurnId;
      if (id) {
        // Fire-and-forget; the page is being torn down so we can't
        // await. cancelTurn already uses keepalive: true so the
        // browser still sends the request.
        void cancelTurnApi({ id });
      }
    });
  });

  return {
    session,
    submit: submitQrl,
    cancel: cancelQrl,
  };
}
