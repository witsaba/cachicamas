import { component$, type QRL } from "@builder.io/qwik";
import type { Conversation } from "~/lib/mock/chat";

/**
 * ConversationList — the archetype's own memory, as a board of runs.
 *
 * Conversations are rows, not cards: a title, how many turns it took, and how
 * long ago. That is what a person scanning for "the one where I refused the
 * schema drop" actually reads by.
 */
export interface ConversationListProps {
  readonly conversations: readonly Conversation[];
  readonly selectedId: string;
  readonly onSelect$: QRL<(id: string) => void>;
}

export const ConversationList = component$<ConversationListProps>((props) => (
  <ul data-testid="conversation-list" class="flex flex-col">
    {props.conversations.map((c) => {
      const active = c.id === props.selectedId;
      return (
        <li key={c.id}>
          <button
            type="button"
            data-testid={`conversation-${c.id}`}
            aria-current={active ? "true" : undefined}
            onClick$={() => props.onSelect$(c.id)}
            class={[
              "border-rule block w-full cursor-pointer border-b px-3 py-2 text-left transition-colors duration-150",
              active ? "bg-raise" : "hover:bg-raise",
            ].join(" ")}
          >
            <span class="flex items-baseline gap-2">
              <span
                aria-hidden="true"
                class={`h-1.5 w-1.5 shrink-0 ${active ? "bg-amber" : "bg-rule-strong"}`}
              />
              <span
                class={`font-human text-data min-w-0 flex-1 truncate ${
                  active ? "text-fg" : "text-fg-mid"
                }`}
              >
                {c.title}
              </span>
            </span>
            <span class="text-legend text-fg-dim mt-0.5 flex items-baseline gap-2 pl-3.5 tracking-[0.1em] uppercase">
              <span>{c.turns} turns</span>
              <span aria-hidden="true">·</span>
              <span>{c.age}</span>
            </span>
          </button>
        </li>
      );
    })}
  </ul>
));
