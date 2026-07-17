/**
 * EmptyState — shown when no skills exist in the SkillStudio.
 *
 * Props:
 *   onCreate$ — handler for the "Create your first skill" CTA
 *
 * Vocabulary:
 *   "Skills" not "prompts" — the empty state copy belongs to this
 *   feature. The shape is identical to the prompts EmptyState so a
 *   future shared primitive is plausible, but the per-feature
 *   vocabulary keeps the contract explicit at each call site.
 */

import { component$, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";

export interface EmptyStateProps {
  onCreate$: QRL<() => void>;
}

export const EmptyState = component$<EmptyStateProps>(({ onCreate$ }) => {
  return (
    <div class="flex flex-col items-center justify-center py-20 text-center">
      {/* Skill icon — book-marked (mirrors SkillsIcon for visual consistency) */}
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
            d="M10 2v8l3-3 3 3V2"
          />
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H19a1 1 0 0 1 1 1v18a1 1 0 0 1-1 1H6.5a1 1 0 0 1 0-5H20"
          />
        </svg>
      </div>

      <h2 class="text-lg font-semibold text-slate-900">No skills yet</h2>
      <p class="mt-1 max-w-xs text-sm text-slate-600">
        Skills are reusable Agent Skills (SKILL.md files). Create your first
        skill to get started.
      </p>

      <div class="mt-6">
        <Button
          type="button"
          variant="primary"
          onClick$={onCreate$}
          testId="empty-state-create"
        >
          Create your first skill
        </Button>
      </div>
    </div>
  );
});