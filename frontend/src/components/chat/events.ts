/**
 * Shared DOM event names for the chat surface.
 *
 * The chat hook (use-chat-stream.ts) and the chat page (chat-app.tsx)
 * communicate browser-only side-effects (auto-scroll on entry
 * mutation) through CustomEvents on `window` rather than Qwik signal
 * reactivity. The dispatcher runs in plain browser time, AFTER Qwik's
 * render commit, so the listener reads the post-commit DOM without
 * racing the commit.
 *
 * Why an event and not a Qwik signal counter (CH-11 / S-SCROLL-001)?
 * Qwik's `useVisibleTask$` re-runs in the SAME microtask as the signal
 * change, BEFORE the render commit. The rAF inside fires before the
 * new <li> is in the DOM — scrollTop = scrollHeight reads the OLD
 * height, and the new content lands below the fold. An event
 * dispatched from the mutation site fires AFTER Qwik's render
 * commit, so the listener runs against the post-commit DOM and the
 * scroll lands.
 *
 * Centralised here so the dispatcher (use-chat-stream.ts) and the
 * listener (chat-app.tsx) share one constant — no magic strings.
 */

/**
 * chat:scroll-to-bottom — fired on `window` whenever
 * `useChatStream`'s `turn.entries` mutates (submit, cancel, SSE
 * chunks, reset). The chat-app's visible-task listens for it and
 * runs the actual scroll inside a double rAF.
 */
export const SCROLL_EVENT = "chat:scroll-to-bottom";