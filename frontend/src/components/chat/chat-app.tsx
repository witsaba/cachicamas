import {
  $,
  component$,
  useSignal,
  useTask$,
  useVisibleTask$,
} from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import { Status } from "~/components/workspace/status/status";
import { initialsOf } from "~/lib/initials";
import { CONVERSATIONS } from "~/lib/mock/chat";
import { AGENTS, agentBySlug } from "~/lib/mock/staff";
import { Composer } from "./composer";
import { ConversationList } from "./conversation-list";
import { TranscriptLine } from "./transcript-line";
import { useMockTurn } from "./use-mock-turn";

/**
 * Chat — a conversation with one colleague.
 *
 * Two columns: the conversations you have had on the left, the one you are in
 * on the right, with the composer pinned to its foot. It is the shape of every
 * messaging surface, and it is the right one here for the same reason it is
 * right there: the history is context, not navigation.
 *
 * What is worth trying, because it is the product's actual argument: ask a
 * colleague to *send* something. The conversation stops, shows the exact thing
 * it is about to do, and waits. Nothing moves until a person answers. That
 * pause is the whole reason a company can let an agent near its systems.
 */
export interface ChatAppProps {
  readonly youName: string;
  readonly youEmail: string;
}

export const ChatApp = component$<ChatAppProps>(({ youName, youEmail }) => {
  const selected = useSignal(CONVERSATIONS[0].id);
  const turn = useMockTurn(CONVERSATIONS[0].entries);
  const scroller = useSignal<HTMLElement>();
  const historyOpen = useSignal(false);

  // `?with=<slug>` opens the newest conversation with that colleague. It is
  // read from the browser rather than the router so this component stays
  // renderable without a request context — the rail links here, the screen
  // answers, and neither needs to know about Qwik City's location service.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(() => {
    if (typeof window === "undefined") return;
    const slug = new URL(window.location.href).searchParams.get("with");
    if (!slug) return;
    const match = CONVERSATIONS.find((c) => c.agentSlug === slug);
    if (match) selected.value = match.id;
  });

  // Selecting a conversation loads its transcript. Work in flight belongs to
  // the conversation it started in, so switching ends it rather than carrying
  // it across.
  useTask$(({ track }) => {
    const id = track(() => selected.value);
    const conversation = CONVERSATIONS.find((c) => c.id === id);
    if (!conversation) return;
    turn.state.entries = [...conversation.entries];
    turn.state.script = [];
    turn.state.beat = 0;
    turn.state.step = 0;
    turn.state.status = "idle";
  });

  // Following an arriving answer to the bottom of a scroller is a browser-only
  // concern by definition; there is no server-side equivalent to fall back to.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(({ track }) => {
    track(() => turn.state.entries.length);
    track(() => turn.state.entries[turn.state.entries.length - 1]);
    if (typeof window === "undefined") return;
    const el = scroller.value;
    if (el) el.scrollTop = el.scrollHeight;
  });

  const conversation =
    CONVERSATIONS.find((c) => c.id === selected.value) ?? CONVERSATIONS[0];
  const agent = agentBySlug(conversation.agentSlug) ?? AGENTS[0];
  const youInitials = initialsOf(youName, youEmail);

  return (
    <div class="flex min-h-0 flex-1">
      {/* the conversations you have had */}
      <aside
        class="border-line bg-surface hidden w-[17rem] shrink-0 flex-col border-r lg:flex"
        data-testid="conversations-panel"
      >
        <div class="border-line flex items-center justify-between border-b px-4 py-3">
          <h2 class="text-ink text-base font-semibold">Conversations</h2>
          <a
            href="/agents/"
            class="text-brand inline-flex items-center gap-1 rounded-sm text-xs font-medium hover:underline"
          >
            <Icon name="plus" size={14} />
            New
          </a>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto">
          <ConversationList
            conversations={CONVERSATIONS}
            selectedId={selected.value}
            onSelect$={$((id: string) => {
              selected.value = id;
            })}
          />
        </div>
      </aside>

      <div class="bg-canvas flex min-w-0 flex-1 flex-col">
        {/* who you are talking to */}
        <header class="border-line bg-surface flex items-center gap-3 border-b px-4 py-2.5">
          <AgentAvatar agent={agent} size="md" />
          <div class="min-w-0 flex-1">
            <p class="flex items-center gap-2">
              <a
                href={`/agents/${agent.slug}/`}
                class="text-ink truncate rounded-sm text-base font-semibold hover:underline"
              >
                {agent.name}
              </a>
              <span class="text-2xs text-ink-soft shrink-0 font-medium">
                Agent
              </span>
            </p>
            <p class="text-ink-soft truncate text-xs">
              {agent.departmentName} ·{" "}
              {turn.state.status === "running"
                ? "Working now"
                : turn.state.status === "held"
                  ? "Waiting for you"
                  : agent.statusWord}
            </p>
          </div>
          <button
            type="button"
            data-testid="history-toggle"
            onClick$={() => (historyOpen.value = !historyOpen.value)}
            class="border-line-control text-ink shrink-0 cursor-pointer rounded-md border px-2.5 py-1.5 text-xs font-medium lg:hidden"
          >
            {historyOpen.value ? "Hide history" : "History"}
          </button>
        </header>

        {/* history on a phone: a capability, not a layout, so it collapses
            rather than disappearing */}
        {historyOpen.value ? (
          <div
            class="border-line bg-surface border-b lg:hidden"
            data-testid="conversations-disclosure"
          >
            <ConversationList
              conversations={CONVERSATIONS}
              selectedId={selected.value}
              onSelect$={$((id: string) => {
                selected.value = id;
                historyOpen.value = false;
              })}
            />
          </div>
        ) : null}

        <ol
          ref={scroller}
          data-testid="transcript"
          aria-live="polite"
          aria-label={`Conversation with ${agent.name}`}
          class="min-h-0 flex-1 overflow-y-auto px-4 pb-2 sm:px-6"
        >
          <li class="pt-5 pb-1">
            <p class="text-ink-soft text-xs">
              {agent.tagline}{" "}
              {agent.tenure ? <>On staff {agent.tenure}.</> : null}{" "}
              <a
                href={`/agents/${agent.slug}/`}
                class="text-brand rounded-sm font-medium hover:underline"
              >
                See what {agent.name} can do
              </a>
            </p>
            <p class="pt-2">
              <Status
                status={agent.status}
                word={agent.statusWord}
                detail={agent.statusDetail}
              />
            </p>
          </li>
          {turn.state.entries.map((entry) => (
            <TranscriptLine
              key={entry.id}
              entry={entry}
              agent={agent}
              youName={youName}
              youInitials={youInitials}
              onDecide$={$((granted: boolean) => {
                void turn.decide(granted);
              })}
            />
          ))}
        </ol>

        <Composer
          status={turn.state.status}
          agentName={agent.name}
          onSubmit$={turn.submit}
          onCancel$={turn.cancel}
        />
      </div>
    </div>
  );
});
