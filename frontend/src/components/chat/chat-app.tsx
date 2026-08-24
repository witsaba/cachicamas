import {
  $,
  component$,
  useSignal,
  useVisibleTask$,
} from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import { Status } from "~/components/workspace/status/status";
import { initialsOf } from "~/lib/initials";
import { AGENTS, agentBySlug } from "~/lib/mock/staff";
import { Composer } from "./composer";
import { TranscriptLine } from "./transcript-line";
import { useChatStream } from "./use-chat-stream";

/**
 * Chat — a conversation with one colleague.
 *
 * What is worth trying, because it is the product's actual argument: ask a
 * colleague to *send* something. The conversation stops, shows the exact thing
 * it is about to do, and waits. Nothing moves until a person answers. That
 * pause is the whole reason a company can let an agent near its systems.
 *
 * CH-05.1 (D-3): the conversations rail is gone — there is one active
 * conversation, and the history list is owned by CH-08.2
 * (cachicamas-chat-conversation-list). The `?with=<slug>` deep-link
 * (D-6) still resolves to an `agentSlug` from staff.ts so the workplace
 * shell's `/chat?with=finance` link keeps working.
 */
export interface ChatAppProps {
  readonly youName: string;
  readonly youEmail: string;
}

export const ChatApp = component$<ChatAppProps>(({ youName, youEmail }) => {
  const activeSlug = useSignal(AGENTS[0].slug);
  const turn = useChatStream([]);
  const scroller = useSignal<HTMLElement>();

  // `?with=<slug>` opens the conversation with that colleague. It is
  // read from the browser rather than the router so this component stays
  // renderable without a request context — the front-desk panel's link
  // (`front-desk.tsx:119`) routes here and the screen answers, neither
  // needs to know about Qwik City's location service.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(() => {
    if (typeof window === "undefined") return;
    const slug = new URL(window.location.href).searchParams.get("with");
    if (!slug) return;
    const match = agentBySlug(slug);
    if (match) activeSlug.value = match.slug;
  });

  // Following an arriving answer to the bottom of a scroller is a browser-only
  // concern by definition; there is no server-side equivalent to fall back to.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(({ track }) => {
    track(() => turn.entries.length);
    track(() => turn.entries[turn.entries.length - 1]);
    if (typeof window === "undefined") return;
    const el = scroller.value;
    if (el) el.scrollTop = el.scrollHeight;
  });

  const agent =
    agentBySlug(activeSlug.value) ?? AGENTS[0];
  const youInitials = initialsOf(youName, youEmail);

  return (
    <div class="flex min-h-0 flex-1">
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
              {turn.status === "streaming" ||
              turn.status === "submitting" ||
              turn.status === "cancelling"
                ? "Working now"
                : agent.statusWord}
            </p>
          </div>
        </header>

        <ol
          ref={scroller}
          data-testid="transcript"
          aria-live="polite"
          aria-label={`Conversation with ${agent.name}`}
          // Every line keeps a reading measure. Without it an answer runs
          // the full width of a 1440px screen, which is about twice the length
          // anyone reads comfortably; the composer below is capped to the same
          // column so the two never disagree about where the conversation is.
          class="min-h-0 flex-1 overflow-y-auto px-4 pb-2 sm:px-6 [&>li]:mx-auto [&>li]:w-full [&>li]:max-w-2xl"
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
          {turn.entries.map((entry) => (
            <TranscriptLine
              key={entry.id}
              entry={entry}
              agent={agent}
              youName={youName}
              youInitials={youInitials}
            />
          ))}
        </ol>

        <Composer
          status={
            turn.status === "submitting" ||
            turn.status === "streaming" ||
            turn.status === "cancelling"
              ? ("running" as const)
              : ("idle" as const)
          }
          agentName={agent.name}
          onSubmit$={$(async (prompt: string) => {
            await turn.submit(prompt);
          })}
          onCancel$={$(async () => {
            await turn.cancel();
          })}
        />
      </div>
    </div>
  );
});
