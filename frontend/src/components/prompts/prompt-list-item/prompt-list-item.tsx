/**
 * PromptListItem — a single prompt in the sidebar list.
 *
 * Props:
 *   prompt     — the Prompt object
 *   selected   — whether this item is currently selected
 *   onClick$   — handler called when the item is clicked
 */

import { component$, type QRL } from "@builder.io/qwik";
import type { Prompt } from "~/lib/prompts-api";
import { listItemClasses } from "./classes";

export interface PromptListItemProps {
  prompt: Prompt;
  selected: boolean;
  onClick$: QRL<() => void>;
}

/** Format a commit-SHA-like timestamp for display. */
function formatDate(iso: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export const PromptListItem = component$<PromptListItemProps>(
  ({ prompt, selected, onClick$ }) => {
    const classes = listItemClasses(selected);

    return (
      <li>
        <button
          type="button"
          onClick$={onClick$}
          class={classes.container}
          aria-current={selected ? "true" : undefined}
          data-testid={
            selected ? "prompt-list-item-selected" : "prompt-list-item"
          }
        >
          <span class={classes.slug}>{prompt.slug}</span>
          <span class={classes.meta}>
            v{prompt.current_revision} &middot; {formatDate(prompt.updated_at)}
          </span>
        </button>
      </li>
    );
  },
);
