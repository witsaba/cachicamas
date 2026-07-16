/**
 * MarkdownTextarea — the left pane of the split editor.
 *
 * A plain textarea with Tailwind styling matching the workspace-sync-card
 * textarea patterns. Value is controlled by the parent.
 *
 * Props:
 *   value       — current body text
 *   onInput$    — handler called on every keystroke with the new value
 *   placeholder — hint text when empty
 */

import { component$, type QRL } from "@builder.io/qwik";

export interface MarkdownTextareaProps {
  value: string;
  onInput$: QRL<(value: string) => void>;
  placeholder?: string;
  testId?: string;
}

export const MarkdownTextarea = component$<MarkdownTextareaProps>(
  ({ value, onInput$, placeholder, testId }) => {
    return (
      <textarea
        value={value}
        onInput$={(e) => {
          onInput$((e.target as HTMLTextAreaElement).value);
        }}
        placeholder={placeholder ?? "Write your prompt body in markdown..."}
        class="h-full w-full resize-none border border-slate-200 p-3 font-mono text-sm text-slate-900 focus:border-slate-400 focus:ring-1 focus:ring-slate-400 focus:outline-none"
        spellcheck={false}
        data-testid={testId ?? "markdown-textarea"}
        aria-label="Prompt body editor"
      />
    );
  },
);
