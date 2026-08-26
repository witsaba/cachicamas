/**
 * use-chat-stream.ts — the chat layer 1 stream hook (CH-05.1).
 *
 * Reference: openspec/changes/cachicamas-chat-frontend-wire/design.md §2.
 *
 * ?with=<slug> deep-link manual smoke (R-7):
 *   1. docker compose up backend agent
 *   2. pnpm --filter @cachicamas/frontend dev
 *   3. load /chat?with=finance
 *   4. verify the page resolves the agent from staff.ts via
 *      agentBySlug(slug), AND `useChatStream`'s first render completes
 *      BEFORE the chat-app `useVisibleTask$` reads
 *      `window.location.href` (the ordering dependency from
 *      chat-app.tsx:47-54).
 *
 * Wire clients consumed verbatim (chat-api.ts:186-413) — no parser
 * rewrite; the byte-exact `single-turn.sse` fixture at
 * chat-api.spec.ts:60-83 is the regression trip.
 */

import {
  $,
  noSerialize,
  useSignal,
  useStore,
  useVisibleTask$,
  type NoSerialize,
  type QRL,
  type Signal,
} from "@builder.io/qwik";
import type { TranscriptEntry } from "~/lib/mock/chat";
import {
  cancelTurn,
  submitTurn,
  subscribeTurn,
  type ChatStreamError,
} from "~/lib/chat-api";
import type { ChatTurnError } from "~/lib/chat-types";
import { SCROLL_EVENT } from "./events";

// Union of submitTurn's typed errors (5-kind, with offline) and the
// stream's narrower event error (4-kind, no offline). The hook
// surfaces both through the same `state.error` field — REQ-4
// (ChatStreamError) and REQ-5 (ChatTurnError with offline) share the
// surface per design.md §2.
type HookError = ChatTurnError | ChatStreamError;

// Re-export the typed event/error shapes from chat-types so chat-app.tsx
// (and consumers of this hook) import everything from one module.
export type { ChatStreamEvent } from "~/lib/chat-types";
export type { ChatStreamError } from "~/lib/chat-types";

/**
 * Reactive shape the chat page binds to. Mirrors `useMockTurn`'s
 * public surface without the scripted-machine fields (`script`,
 * `beat`, `step`, `play`, `settle`, `decide`). The four-state status
 * drops `held` (REQ-10 — permission protocol is CH-09/CH-10 scope).
 *
 * `error` carries the broader `ChatTurnError` shape (with offline kind)
 * so submitTurn's network-failure arm and the stream's onError arm
 * surface through the same field (REQ-4 + REQ-5 — D-1).
 */
export interface ChatStreamState {
  status: "idle" | "submitting" | "streaming" | "cancelling";
  entries: TranscriptEntry[];
  error?: HookError;
  currentTurnId?: string;
  submit: QRL<(prompt: string) => Promise<void>>;
  cancel: QRL<() => void>;
  reset: QRL<(entries?: readonly TranscriptEntry[]) => void>;
}

// Internal handle for an in-flight turn. `unsubscribe` carries the
// `subscribeTurn` return value, which is a plain JS function — not
// serialisable across SSR resumption. `noSerialize()` is Qwik's
// blessed escape hatch: the value lives in memory but is dropped on
// serialisation, exactly what an EventSource close handle needs.
interface InFlight {
  id: string;
  unsubscribe: NoSerialize<() => void>;
}

// A monotonic id source — every entry minted by this hook has a
// `chat`-side entry.id stable across renames, so transcript-line
// data-testid queries don't drift when an entry finalises.
function nextId(s: { seq: number }, prefix: "u" | "a"): string {
  s.seq += 1;
  return `${prefix}${s.seq}`;
}

/** Client-minted UUID for the cancellation key. SSR-safe via try/catch. */
function newTurnId(): string {
	try {
		if (
			typeof globalThis !== "undefined" &&
			typeof (globalThis as { crypto?: Crypto }).crypto !== "undefined" &&
			(globalThis as { crypto?: Crypto }).crypto &&
			typeof (globalThis as { crypto?: Crypto }).crypto!.randomUUID === "function"
		) {
			return (globalThis as { crypto: Crypto }).crypto!.randomUUID();
		}
	} catch {
		// Crypto unavailable — fall through to the time-based id.
	}
	return `trn_${Date.now()}_${Math.floor(Math.random() * 1e6)}`;
}

/**
 * CH-09 — parseArgs converts the wire's `arguments` string (JSON
 * object) into the entry's args tuple array. The wire carries the
 * JSON text verbatim (chat.ToolCallStart.Arguments) per R-CTS-004
 * / D-3; the transcript entry's args shape is a list of (key,
 * value) tuples. Malformed JSON yields an empty tuple list; the
 * tool's state stays "running" until tool.result arrives.
 */
function parseArgs(
	jsonText: string,
): readonly (readonly [string, string])[] {
	if (!jsonText) return [];
	try {
		const obj = JSON.parse(jsonText) as Record<string, unknown>;
		if (typeof obj !== "object" || obj === null) return [];
		return Object.entries(obj).map(
			([k, v]) => [k, String(v)] as readonly [string, string],
		);
	} catch {
		// Malformed JSON — fall through to the empty tuple list so
		// the tool entry still renders (state stays "running" until
		// tool.result arrives).
		return [];
	}
}

/**
 * CH-12 (S-SCROLL-001) — drive auto-scroll from an explicit DOM
 * event dispatched at the mutation site, not from a Qwik signal
 * counter.
 *
 * Module-level (NOT a local closure inside `useChatStream`) so
 * the submit/cancel/reset QRLs can call it without capturing a
 * non-serialisable function reference. Qwik's QRL optimizer
 * captures closure variables and serialises them across resume;
 * a plain `const requestScroll = () => {...}` inside the hook
 * would fail `_verifySerializable` with
 * "Captured variable ... can not be serialized because it's a
 *  function named 'requestScroll'". A module-level helper escapes
 * the closure-capture pass entirely.
 *
 * What this does, in order:
 *   1. Dispatch `chat:scroll-to-bottom` on `window`. The chat-app's
 *      visible-task listens for it and runs the actual scroll inside
 *      a double rAF (so the new <li> is in the DOM and scrollHeight
 *      reflects the post-commit layout). The event fires AFTER
 *      Qwik's render commit because the dispatch happens in plain
 *      browser time, not inside Qwik's reactive task lifecycle —
 *      the bug with the previous signal-counter approach
 *      (CH-11 / commit 81a24641) was that Qwik re-ran the
 *      visible-task in the SAME microtask as the signal change,
 *      so the rAF inside fired BEFORE the new <li> was in the DOM.
 *   2. If `scrollCounter` was supplied, bump it too. The counter
 *      is a public-API fallback for external observers (tests,
 *      analytics). The actual scroll is event-driven; the counter
 *      is preserved for backward compatibility with the
 *      `scrollCounter?` hook signature introduced in CH-11.
 *
 * No-op when running under SSR (`typeof window === "undefined"`)
 * AND when no counter was supplied.
 *
 * Safe in test environments where the test window's `dispatchEvent`
 * is missing — the dispatch call is guarded by
 * `typeof window.dispatchEvent === "function"`.
 */
function requestScroll(scrollCounter: Signal<number> | undefined): void {
	if (
		typeof window !== "undefined" &&
		typeof window.dispatchEvent === "function"
	) {
		window.dispatchEvent(new CustomEvent(SCROLL_EVENT));
	}
	if (scrollCounter) {
		scrollCounter.value = scrollCounter.value + 1;
	}
}

export function useChatStream(
  initialEntries: ReadonlyArray<TranscriptEntry>,
  /**
   * CH-12 (S-SCROLL-001) — auto-scroll bus fallback. The chat-app
   * owns a `useSignal(0)` counter and passes the signal ref here.
   * The hook bumps it on EVERY mutation of `state.entries` — kept
   * as a public-API fallback for external observers (tests,
   * analytics). The actual scroll is driven by the
   * `chat:scroll-to-bottom` CustomEvent dispatched at the same
   * mutation sites (see `requestScroll` at the top of this file);
   * the chat-app's visible-task listens for the event and scrolls
   * after Qwik's render commit, so scrollHeight reflects the
   * post-commit layout reliably (CH-11 / commit 81a24641 re-ran
   * the visible-task in the SAME microtask as the signal change —
   * BEFORE the render commit, so the rAF inside fired before the
   * new <li> was in the DOM).
   *
   * Optional so existing call sites stay wired. When omitted,
   * the counter is simply not advanced; the event dispatch
   * still runs.
   */
  scrollCounter?: Signal<number>,
): ChatStreamState {
  const state = useStore<{
    status: "idle" | "submitting" | "streaming" | "cancelling";
    entries: TranscriptEntry[];
    error?: HookError;
    currentTurnId?: string;
    seq: number;
  }>(
    {
      status: "idle",
      entries: [...initialEntries],
      seq: 0,
    },
    { deep: true },
  );

  const handle = useSignal<InFlight | null>(null);
  // Tracks whether cancel was initiated by the user (Stop button) —
  // the unmount path reuses the same close chain, so without this
  // guard an `unmount` after a `cancel` would re-issue DELETE.
  const intentionalClose = useSignal(false);

  const submit = $(async (prompt: string) => {
    intentionalClose.value = false;
    state.error = undefined;
    const text = prompt.trim();
    if (text.length === 0 || state.status !== "idle") {
      return;
    }
    const turnId = newTurnId();
    state.currentTurnId = turnId;
    state.status = "submitting";
    // Seed entries in display order: user turns are visible above
    // the assistant's response, so the user entry is pushed FIRST and
    // the assistant placeholder SECOND. Reversing this order made
    // the assistant bubble render above the user bubble during the
    // streaming window (the live render bug); reload from the
    // server's exchange list renders correctly because the backend
    // already sorts by position ascending.
    const userId = nextId(state, "u");
    state.entries.push({
      kind: "said",
      id: userId,
      who: "you",
      text,
      state: "final",
    });
    requestScroll(scrollCounter);
    const assistantId = nextId(state, "a");
    state.entries.push({
      kind: "said",
      id: assistantId,
      who: "chat",
      text: "",
      state: "streaming",
    });
    requestScroll(scrollCounter);

    const submitResult = await submitTurn({ id: turnId, prompt: text });
    if (!submitResult.ok) {
      // Drop the seeded in-flight + user entries; render the error
      // bubble in their place by leaving `state.error` set.
      state.entries = state.entries.filter(
        (e) => e.id !== assistantId && e.id !== userId,
      );
      requestScroll(scrollCounter);
      state.error = submitResult;
      state.currentTurnId = undefined;
      state.status = "idle";
      // Mirror the typed error union's surface (REQ-5 S-5.a).
      return;
    }
    state.status = "streaming";

    const streamUrl = submitResult.value.streamUrl;
    const subscribeResult = subscribeTurn(
      streamUrl,
      (ev) => {
        switch (ev.kind) {
          case "message.delta": {
            state.entries = state.entries.map((e) =>
              e.id === assistantId && e.kind === "said"
                ? { ...e, text: e.text + ev.delta }
                : e,
            );
            requestScroll(scrollCounter);
            return;
          }
          case "message.end": {
            // Mark the entry final on message.end (turn.end follows
            // shortly, but message.end is the assistant's natural close).
            state.entries = state.entries.map((e) =>
              e.id === assistantId && e.kind === "said"
                ? { ...e, state: "final" as const }
                : e,
            );
            requestScroll(scrollCounter);
            return;
          }
          // CH-09 (D-3, D-4, S-CTS-020) — tool.call.start opens a
          // new "tool" entry; the entry's id is the wire call id so
          // a later tool.result event can match and close it. v1
          // does NOT emit tool.call.delta / tool.call.end (NFR-CTS-002
          // / D-6 collapse); a future long-running MCP tool would
          // emit them.
          case "tool.call.start": {
            const newId = `tool-${ev.wireCallId}`;
            state.entries = [
              ...state.entries,
              {
                kind: "tool",
                id: newId,
                tool: ev.tool,
                intent: ev.tool,
                args: parseArgs(ev.arguments),
                state: "running",
              },
            ];
            requestScroll(scrollCounter);
            return;
          }
          case "tool.result": {
            // CH-09 (D-3, S-CTS-021) — close the matching tool entry.
            // success / result_failure → state "done"; execution_failure
            // → state "failed" with the typed failure category.
            // Provider text on failure MUST NOT leak (R-CCP-008 / D6
            // mirror) — we carry the typed category, not the raw
            // provider error.
            const toolId = `tool-${ev.wireCallId}`;
            const newState: "done" | "failed" =
              ev.outcome === "execution_failure" ? "failed" : "done";
            const resultText =
              ev.outcome === "execution_failure"
                ? ev.failureCategory
                : ev.content;
            state.entries = state.entries.map((e) =>
              e.id === toolId && e.kind === "tool"
                ? { ...e, state: newState, result: resultText }
                : e,
            );
            requestScroll(scrollCounter);
            return;
          }
          case "tool.call.delta":
          case "tool.call.end": {
            // Reserved variants (NFR-CTS-002 / D-6); v1's chat
            // projector never emits them. No-op so a future wire
            // shape change that DOES emit them doesn't crash.
            return;
          }
          // CH-10 (R-CPM-003, D-4) — permission event family.
          // The "required" event opens a `hold` entry whose
          // decision starts at "waiting"; the "made" event
          // morphs the same entry's decision to "allowed" or
          // "denied". The frontend hold entry's id is the
          // wire's wireCallId (stable across the lifetime of
          // the row). The "made" event also suppresses the
          // matching tool.result entry's rendering for "deny"
          // (D-8 collapse mirror; Layer 2 emits both
          // permission_decision_made{deny} AND
          // tool_end_execution_failure for the same callID,
          // and the user-facing surface is the hold alone).
          case "permission.decision.required": {
            const newId = `hold-${ev.wireCallId}`;
            state.entries = [
              ...state.entries,
              {
                kind: "hold",
                id: newId,
                tool: ev.tool,
                intent: ev.tool,
                args: parseArgs(ev.arguments),
                risk: "",
                decision: "pending",
              },
            ];
            requestScroll(scrollCounter);
            return;
          }
          case "permission.decision.made": {
            const holdId = `hold-${ev.wireCallId}`;
            const newDecision: "granted" | "denied" =
              ev.outcome === "allow_once" ? "granted" : "denied";
            state.entries = state.entries.map((e) =>
              e.id === holdId && e.kind === "hold"
                ? { ...e, decision: newDecision }
                : e,
            );
            requestScroll(scrollCounter);
            // D-8 mirror: on Deny, drop any tool entry for the
            // same wireCallId that arrived between required and
            // made. The chat projector suppressed the wire event
            // (see chat/projection.go D-8 suppression), but a
            // previously-buffered tool entry from a parallel
            // emission would still render. Defence in depth.
            if (newDecision === "denied") {
              const toolId = `tool-${ev.wireCallId}`;
              state.entries = state.entries.filter(
                (e) => !(e.id === toolId && e.kind === "tool"),
              );
              requestScroll(scrollCounter);
            }
            return;
          }
          case "message.start":
          case "turn.end":
          case "error": {
            // turn.end closes the stream exactly once (REQ-1 S-1.a).
            // We do not issue a DELETE on the natural close (REQ-2 S-2.c).
            if (ev.kind === "turn.end") {
              // Mark the in-flight assistant entry final here too — if
              // message.end was not emitted, this is the only close.
              state.entries = state.entries.map((e) =>
                e.id === assistantId && e.kind === "said"
                  ? { ...e, state: "final" as const }
                  : e,
              );
              requestScroll(scrollCounter);
              const unsubscribe = handle.value?.unsubscribe;
              handle.value = null;
              state.status = "idle";
              state.currentTurnId = undefined;
              if (typeof unsubscribe === "function") {
                unsubscribe();
              }
            } else if (ev.kind === "error") {
              // REQ-4 S-4.a — typed error surfaces inline; no retry.
              state.error = ev.error;
              state.status = "idle";
              // Finalise the assistant entry too — the turn is over.
              state.entries = state.entries.map((e) =>
                e.id === assistantId && e.kind === "said"
                  ? { ...e, state: "final" as const }
                  : e,
              );
              requestScroll(scrollCounter);
            }
            return;
          }
          default: {
            // REQ-1 S-1.b — unknown events are ignored.
            return;
          }
        }
      },
      // onError — REQ-5 S-5.b. The wire client surfaces the offline
      // message via the existing offline kind, but the hook shape does
      // not have a second error surface; we treat this as the same
      // offline kind for symmetry.
      (msg) => {
        state.error = { kind: "offline", message: msg };
        state.status = "idle";
        state.entries = state.entries.filter((e) => e.id !== assistantId);
        requestScroll(scrollCounter);
      },
    );

    handle.value = { id: turnId, unsubscribe: noSerialize(subscribeResult) };
  });

  const cancel = $(async () => {
    if (state.status === "idle") return;
    state.status = "cancelling";
    intentionalClose.value = true;
    const inFlight = handle.value;
    handle.value = null;
    if (inFlight) {
      // DELETE first (with keepalive: true inside cancelTurn,
      // chat-api.ts:236) — survives the unmount path. Then close
      // the EventSource.
      try {
        await cancelTurn({ id: inFlight.id });
      } catch {
        // Network failure on cancel is a no-op for the user; the
        // partial assistant text is finalised locally anyway.
      }
      if (typeof inFlight.unsubscribe === "function") {
        inFlight.unsubscribe();
      }
    }
    // Finalise the in-flight assistant entry where it stopped (REQ-2
    // S-2.b — partial text remains visible).
    const lastAssistant = [...state.entries]
      .reverse()
      .find((e) => e.kind === "said" && e.who === "chat");
    if (lastAssistant && lastAssistant.kind === "said") {
      const lastId = lastAssistant.id;
      state.entries = state.entries.map((e) =>
        e.id === lastId && e.kind === "said"
          ? { ...e, state: "final" as const }
          : e,
      );
      requestScroll(scrollCounter);
    }
    state.status = "idle";
    state.currentTurnId = undefined;
  });

  const reset = $((next?: readonly TranscriptEntry[]) => {
    state.entries = [...(next ?? [])];
    requestScroll(scrollCounter);
    state.error = undefined;
    state.status = "idle";
    state.currentTurnId = undefined;
    const inFlight = handle.value;
    handle.value = null;
    if (inFlight) {
      try {
        void cancelTurn({ id: inFlight.id });
      } catch {
        // Same as cancel — best-effort.
      }
      if (typeof inFlight.unsubscribe === "function") {
        inFlight.unsubscribe();
      }
    }
  });

  // REQ-2 S-2.a — unmount cleanup. Always registered; no-op when
  // `handle` is null. Mirrors the lifecycle in design.md §2.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(({ cleanup }) => {
    cleanup(() => {
      if (intentionalClose.value) return;
      const inFlight = handle.value;
      handle.value = null;
      if (!inFlight) return;
      try {
        // Fire-and-forget; keepalive: true makes the DELETE survive
        // the page tear-down (chat-api.ts:236).
        void cancelTurn({ id: inFlight.id });
      } catch {
        // Best-effort: cleanup must not throw or it surfaces the
        // frame teardown to the consumer.
      }
      if (typeof inFlight.unsubscribe === "function") {
        inFlight.unsubscribe();
      }
    });
  });

  return {
    get status() {
      return state.status;
    },
    get entries() {
      return state.entries;
    },
    get error() {
      return state.error;
    },
    get currentTurnId() {
      return state.currentTurnId;
    },
    submit,
    cancel,
    reset,
  };
}
