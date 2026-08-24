/**
 * The conversations you have had, newest first.
 *
 * Each row carries who it was with, what it was about and when — the three
 * things a person uses to find a conversation again. The colleague's avatar
 * does the recognising before any of the words are read, which is exactly what
 * a department colour is for.
 *
 * CH-08.2 (REQ-8, R-CRI-005): the rail is wire-driven. The page's
 * useVisibleTask$ mount fetches the list and passes the
 * `ConversationSummary[]` directly — id (the participant id per
 * D-1), turnCount (TurnCount), and the relative-time `age` derived
 * from lastActivityAt (RFC3339, ISO8601). The mock `Conversation`
 * type from lib/mock/chat.ts still serves the demonstration
 * surfaces (`hero-proof`, `front-desk` — CH-05 D-3); this
 * component's typed prop surface is the wire.
 *
 * The wire does NOT carry `agentSlug` or `title` — only the
 * participant id (D-1), turnCount, lastActivityAt. v1 renders the
 * id + count + age only. A future milestone that surfaces the
 * conversation's agent title lands in a follow-up; today the row
 * matches the wire's exact contract (R-CRI-004: no fields invented
 * beyond what the port's projection carries).
 */
import { component$, type QRL } from "@builder.io/qwik";
import { formatRelativeTime } from "~/components/chat/format-relative-time";
import type { ConversationSummary } from "~/lib/chat-types";

export interface ConversationListProps {
  /** Wire-shaped summaries (R-CRI-005). Empty list renders empty rail. */
  readonly conversations: readonly ConversationSummary[];
  /** Selected conversation id — wired through; today the participant id (D-1). */
  readonly selectedId: string;
  readonly onSelect$: QRL<(id: string) => void>;
}

export const ConversationList = component$<ConversationListProps>((props) => {
  const select = props.onSelect$;
  return (
    <ul class="space-y-0.5 p-2" data-testid="conversation-list">
      {props.conversations.map((c) => {
        const current = c.conversationID === props.selectedId;
        // age is derived at render time from the wire's ISO timestamp;
        // SSR returns the absolute fallback, the next render in the
        // browser re-derives the relative phrase ("5 minutes ago").
        const age = formatRelativeTime(c.lastActivityAt);
        return (
          <li key={c.conversationID}>
            <button
              type="button"
              aria-current={current ? "true" : undefined}
              data-testid={`conversation-${c.conversationID}`}
              onClick$={() => select(c.conversationID)}
              class={[
                "flex w-full cursor-pointer items-start gap-2.5 rounded-md px-2 py-2 text-left transition-colors duration-150",
                current ? "bg-brand-tint" : "hover:bg-sunken",
              ].join(" ")}
            >
              <span class="min-w-0 flex-1">
                <span
                  class={[
                    "block truncate text-sm",
                    current ? "text-brand font-medium" : "text-ink",
                  ].join(" ")}
                >
                  {c.conversationID}
                </span>
                <span class="text-2xs text-ink-soft block truncate">
                  {c.turnCount} turn{c.turnCount === 1 ? "" : "s"} · {age}
                </span>
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
});
