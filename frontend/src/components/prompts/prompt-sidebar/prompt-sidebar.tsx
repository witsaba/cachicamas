/**
 * PromptSidebar — left panel with a filterable list of prompts.
 *
 * Props:
 *   prompts     — all prompts
 *   selectedSlug — currently selected slug (or null)
 *   onSelect$   — handler when a prompt is clicked
 *   onNewPrompt$ — handler when "+ New Prompt" is clicked
 */

import { component$, type QRL, useSignal } from "@builder.io/qwik";
import type { Prompt } from "~/lib/prompts-api";
import { PromptListItem } from "~/components/prompts/prompt-list-item/prompt-list-item";
import { Button } from "~/components/ui/button/button";

export interface PromptSidebarProps {
  prompts: Prompt[];
  selectedSlug: string | null;
  onSelect$: QRL<(slug: string) => void>;
  onNewPrompt$: QRL<() => void>;
}

export const PromptSidebar = component$<PromptSidebarProps>(
  ({ prompts, selectedSlug, onSelect$, onNewPrompt$ }) => {
    const filter = useSignal("");

    const filtered = prompts.filter((p) =>
      p.slug.toLowerCase().includes(filter.value.toLowerCase()),
    );

    return (
      <div class="flex h-full w-64 flex-col border-r border-slate-200 bg-white">
        {/* Filter input */}
        <div class="border-b border-slate-100 p-3">
          <input
            type="text"
            placeholder="Filter prompts..."
            value={filter.value}
            onInput$={(e) => {
              filter.value = (e.target as HTMLInputElement).value;
            }}
            class="w-full rounded border border-slate-200 px-2 py-1 text-sm text-slate-900 placeholder-slate-400 focus:border-slate-400 focus:ring-1 focus:ring-slate-400 focus:outline-none"
            aria-label="Filter prompts"
            data-testid="prompt-sidebar-filter"
          />
        </div>

        {/* Prompt list */}
        <ul
          class="flex-1 overflow-y-auto py-1"
          role="listbox"
          aria-label="Prompts"
          data-testid="prompt-sidebar-list"
        >
          {filtered.length === 0 ? (
            <li class="px-3 py-4 text-center text-sm text-slate-400">
              {prompts.length === 0
                ? "No prompts yet"
                : "No prompts match your filter"}
            </li>
          ) : (
            filtered.map((prompt) => (
              <PromptListItem
                key={prompt.slug}
                prompt={prompt}
                selected={prompt.slug === selectedSlug}
                onClick$={() => onSelect$(prompt.slug)}
              />
            ))
          )}
        </ul>

        {/* New prompt button */}
        <div class="border-t border-slate-100 p-3">
          <Button
            type="button"
            variant="primary"
            class="w-full justify-center"
            onClick$={onNewPrompt$}
            testId="prompt-sidebar-new"
          >
            <svg
              class="mr-1.5 h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width={2}
              aria-hidden="true"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 4v16m8-8H4"
              />
            </svg>
            New Prompt
          </Button>
        </div>
      </div>
    );
  },
);
