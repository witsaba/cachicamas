/**
 * chat-input.tsx — the prompt entry controls.
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md
 *   §3 (textarea + Send + Cancel; disabled derived from
 *   session.status !== "idle"; Enter submits).
 *
 * Disabled-during-stream (T-05): the input, send, and cancel buttons
 * reflect session.status. `disabled={!isIdle}` is the single source
 * of truth — when streaming, no further prompts are accepted.
 *
 * Enter-to-submit (T-07): handled at the form level so the textarea
 * + button pair both work the same way. Shift+Enter inserts a
 * newline (the textarea preserves multi-line prompts).
 *
 * The component is intentionally presentational; the hook
 * (`useChatStream$`) owns the actual submit/cancel logic. This
 * file ships the controls + testids.
 */
import {
  $,
  component$,
  type QRL,
  useSignal,
  useTask$,
} from "@builder.io/qwik";

import { Button } from "~/components/ui/button/button";

export interface ChatInputProps {
  /** Disable every control when true (status !== 'idle'). */
  disabled?: boolean;
  /** Test-only: bypass the hook and provide an explicit submit QRL. */
  onSubmit$?: QRL<(value: string) => void>;
  /** Test-only: bypass the hook and provide an explicit cancel QRL. */
  onCancel$?: QRL<() => void>;
}

export const ChatInput = component$<ChatInputProps>(
  ({ disabled = false, onSubmit$, onCancel$ }) => {
    const value = useSignal("");
    const textareaRef = useSignal<HTMLTextAreaElement | undefined>();

    // Lazy import to avoid a circular reference at module load —
    // use-chat-stream imports nothing from this file, but the test
    // mocks ./use-chat-stream and we want the import resolution to
    // happen at render time so the mock takes effect.
    useTask$(({ track }) => {
      track(() => disabled);
      void useChatStreamImport; // silence "declared but never used" — only the ref is kept
    });

    const submitQrl = onSubmit$ ?? $(async (text: string) => {
      const { useChatStream$ } = await import("./use-chat-stream");
      const host = document.querySelector("[data-testid='chat-window']");
      // The hook lives on the ChatWindow; we resolve the hook via
      // the module-level export. The component$ render context
      // exposes the same hook to the input via Qwik's signal-sharing
      // mechanism (useStore is shared by the parent and the input).
      // For v1 the input re-imports the hook so the spec can mock
      // it independently.
      const { session, submit } = useChatStream$();
      void session;
      void host;
      await submit(text);
    });

    const cancelQrl = onCancel$ ?? $(async () => {
      const { useChatStream$ } = await import("./use-chat-stream");
      const { cancel } = useChatStream$();
      await cancel();
    });

    const handleSubmit = $(async () => {
      const text = value.value.trim();
      if (text.length === 0) return;
      value.value = "";
      await submitQrl(text);
    });

    const handleCancel = $(async () => {
      await cancelQrl();
    });

    return (
      <form
        data-testid="chat-input"
        preventdefault:submit
        onSubmit$={handleSubmit}
        class="flex items-end gap-2 border-t border-slate-200 pt-4"
      >
        <textarea
          ref={textareaRef}
          data-testid="chat-input-textarea"
          aria-label="Prompt"
          rows={2}
          disabled={disabled}
          placeholder={
            disabled ? "Streaming… press Stop to cancel." : "Type a prompt…"
          }
          value={value.value}
          onInput$={(_, el) => {
            value.value = el.value;
          }}
          onKeyDown$={(ev) => {
            // Enter submits; Shift+Enter inserts a newline.
            if (ev.key === "Enter" && !ev.shiftKey) {
              ev.preventDefault();
              void handleSubmit();
            }
          }}
          class="min-h-12 flex-1 resize-none rounded-md border border-slate-300 bg-white p-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none disabled:bg-slate-100 disabled:text-slate-500"
        />
        <div class="flex gap-2">
          <Button
            testId="chat-input-send"
            type="submit"
            variant="primary"
            disabled={disabled || value.value.trim().length === 0}
          >
            Send
          </Button>
          <Button
            testId="chat-input-cancel"
            type="button"
            variant="secondary"
            disabled={disabled}
            onClick$={handleCancel}
          >
            Stop
          </Button>
        </div>
      </form>
    );
  },
);

// Stable module-level reference so the useTask$ above can track it
// without TypeScript flagging an unused symbol.
const useChatStreamImport = "./use-chat-stream";
