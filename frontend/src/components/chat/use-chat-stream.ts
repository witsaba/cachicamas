/**
 * use-chat-stream.ts — the chat layer 1 stream hook (CH-05.1).
 *
 * Reference: openspec/changes/cachicamas-chat-frontend-wire/design.md §2.
 *
 * This is the RED-only scaffold of the hook. It exists so
 * `use-chat-stream.spec.ts` can import the export shape; every
 * behavioral case in that spec fails today because the action QRLs
 * throw "not implemented" and the lifecycle never executes. WU-2
 * (T2.1) replaces the body with the real implementation against
 * `submitTurn` / `cancelTurn` / `subscribeTurn`.
 */

import type { TranscriptEntry } from "~/lib/mock/chat";
import type { QRL } from "@builder.io/qwik";

/**
 * Reactive shape the chat page binds to. Mirrors `useMockTurn`'s
 * public surface without the scripted-machine fields (`script`,
 * `beat`, `step`, `play`, `settle`, `decide`). The four-state status
 * drops `held` (REQ-10 — permission protocol is CH-09/CH-10 scope).
 */
export interface ChatStreamState {
  status: "idle" | "submitting" | "streaming" | "cancelling";
  entries: TranscriptEntry[];
  error?: ChatStreamError;
  currentTurnId?: string;
  submit: QRL<(prompt: string) => Promise<void>>;
  cancel: QRL<() => void>;
  reset: QRL<(entries?: readonly TranscriptEntry[]) => void>;
}

export type ChatStreamError =
  | { readonly kind: "offline"; readonly message: string }
  | { readonly kind: "validation"; readonly message: string; readonly fields: Record<string, string> }
  | { readonly kind: "conflict"; readonly message: string }
  | { readonly kind: "not_found"; readonly message: string }
  | { readonly kind: "server"; readonly message: string };

/**
 * The RED stub. T2.1 replaces the body with the real implementation
 * against `submitTurn`, `cancelTurn`, and `subscribeTurn`. Until then
 * render produces an empty idle state so the harness surfaces its DOM;
 * every action QRL throws "not implemented", which is the right RED
 * signal for spec assertions that drive submit / cancel.
 */
export function useChatStream(
  initialEntries: ReadonlyArray<TranscriptEntry>,
): ChatStreamState {
  const entries: TranscriptEntry[] = [...initialEntries];
  const notImplemented = $(async () => {
    throw new Error("useChatStream: not implemented");
  });
  return {
    status: "idle",
    entries,
    submit: notImplemented as unknown as QRL<(prompt: string) => Promise<void>>,
    cancel: notImplemented as unknown as QRL<() => void>,
    reset: notImplemented as unknown as QRL<
      (entries?: readonly TranscriptEntry[]) => void
    >,
  };
}

// Imported here for clarity — the real implementation returns a
// `$()`-wrapped action so Qwik can serialise it across resumption.
import { $ } from "@builder.io/qwik";
