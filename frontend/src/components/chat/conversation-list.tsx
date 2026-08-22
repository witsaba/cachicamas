/**
 * The conversations you have had, newest first.
 *
 * Each row carries who it was with, what it was about and when — the three
 * things a person uses to find a conversation again. The colleague's avatar
 * does the recognising before any of the words are read, which is exactly what
 * a department colour is for.
 */
import { component$, type QRL } from "@builder.io/qwik";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import type { Conversation } from "~/lib/mock/chat";
import { agentBySlug } from "~/lib/mock/staff";

export interface ConversationListProps {
  readonly conversations: readonly Conversation[];
  readonly selectedId: string;
  readonly onSelect$: QRL<(id: string) => void>;
}

export const ConversationList = component$<ConversationListProps>((props) => {
  const select = props.onSelect$;
  return (
    <ul class="space-y-0.5 p-2" data-testid="conversation-list">
      {props.conversations.map((c) => {
        const agent = agentBySlug(c.agentSlug);
        const current = c.id === props.selectedId;
        return (
          <li key={c.id}>
            <button
              type="button"
              aria-current={current ? "true" : undefined}
              data-testid={`conversation-${c.id}`}
              onClick$={() => select(c.id)}
              class={[
                "flex w-full cursor-pointer items-start gap-2.5 rounded-md px-2 py-2 text-left transition-colors duration-150",
                current ? "bg-brand-tint" : "hover:bg-sunken",
              ].join(" ")}
            >
              {agent ? <AgentAvatar agent={agent} size="sm" /> : null}
              <span class="min-w-0 flex-1">
                <span
                  class={[
                    "block truncate text-sm",
                    current ? "text-brand font-medium" : "text-ink",
                  ].join(" ")}
                >
                  {c.title}
                </span>
                <span class="text-2xs text-ink-soft block truncate">
                  {agent?.name ?? "Unknown"} · {c.age}
                </span>
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
});
