/**
 * chat-input.tsx — the prompt entry controls.
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md
 *   §3 (textarea + Send + Cancel; disabled derived from
 *   session.status !== "idle"; Enter submits).
 *
 * Disabled-during-stream (REQ-1 S-1.a): the textarea, send, and
 * cancel buttons reflect `disabled` (which the chat-window derives
 * from `!session.isIdle`). When streaming, no further prompts are
 * accepted.
 *
 * Enter-to-submit (REQ-1 Enter-to-submit): handled at the form level
 * so the textarea + button pair both work the same way. Shift+Enter
 * inserts a newline (the textarea preserves multi-line prompts).
 *
 * Submit / Cancel QRLs are supplied by the parent (ChatWindow →
 * useChatStream). This component is purely presentational; it does
 * NOT call use* methods directly (Qwik forbids that inside event
 * handlers per the qwik/use-method-usage lint rule). The parent owns
 * the actual lifecycle.
 */
import { $, component$, type QRL, useSignal } from "@builder.io/qwik";

import { Button } from "~/components/ui/button/button";

export interface ChatInputProps {
  /** Disable every control when true (status !== 'idle'). */
  disabled?: boolean;
  /**
   * Submit QRL — fired on Enter (without Shift) or Send click. The
   * parent (ChatWindow) supplies this from `useChatStream().submit`.
   * Required: the input has no default hook fallback because Qwik
   * forbids `use*()` calls inside event handlers
   * (qwik/use-method-usage). The required prop type also pins the
   * seam for tests — a test-only QRL stub pins the wiring without
   * booting the hook.
   */
  onSubmit$: QRL<(value: string) => void>;
  /**
   * Cancel QRL — fired on Stop click. The parent (ChatWindow)
   * supplies this from `useChatStream().cancel`. Required for the
   * same reason as `onSubmit$`.
   */
  onCancel$: QRL<() => void>;
}

export const ChatInput = component$<ChatInputProps>(
  ({ disabled = false, onSubmit$, onCancel$ }) => {
    const value = useSignal("");
    const textareaRef = useSignal<HTMLTextAreaElement | undefined>();

    const handleSubmit = $(async () => {
      const text = value.value.trim();
      if (text.length === 0) return;
      value.value = "";
      await onSubmit$(text);
    });

    const handleCancel = $(async () => {
      await onCancel$();
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
