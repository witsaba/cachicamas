/**
 * PromptStudio empty state — shown when no prompts exist.
 *
 * Props:
 *   onCreate$ — handler for the "Create your first prompt" CTA
 */

import { component$, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";

export interface EmptyStateProps {
  onCreate$: QRL<() => void>;
}

export const EmptyState = component$<EmptyStateProps>(({ onCreate$ }) => {
  return (
    <div class="flex flex-col items-center justify-center py-20 text-center">
      {/* Prompt icon */}
      <div class="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-slate-100">
        <svg
          class="h-7 w-7 text-slate-500"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width={1.5}
          aria-hidden="true"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 01.865-.501 48.172 48.172 0 003.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0012 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018z"
          />
        </svg>
      </div>

      <h2 class="text-lg font-semibold text-slate-900">No prompts yet</h2>
      <p class="mt-1 max-w-xs text-sm text-slate-600">
        System prompts control how the AI behaves. Create your first prompt to
        get started.
      </p>

      <div class="mt-6">
        <Button
          type="button"
          variant="primary"
          onClick$={onCreate$}
          testId="empty-state-create"
        >
          Create your first prompt
        </Button>
      </div>
    </div>
  );
});
