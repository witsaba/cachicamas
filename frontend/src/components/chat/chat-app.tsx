import {
  $,
  component$,
  useSignal,
  useStore,
  useVisibleTask$,
} from "@builder.io/qwik";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import { Status } from "~/components/workspace/status/status";
import { initialsOf } from "~/lib/initials";
import { AGENTS, agentBySlug } from "~/lib/mock/staff";
import {
  listConversations,
  loadConversation,
} from "~/lib/chat-api";
import type { ConversationSummary } from "~/lib/chat-types";
import type { TranscriptEntry } from "~/lib/mock/chat";
import { Composer } from "./composer";
import { ConversationList } from "./conversation-list";
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
 * CH-05.1 (D-3): the conversations rail was dropped from the CH-05.1
 * wire-up. CH-08 undoes that — the rail mounts again, driven by the
 * real wire (R-CRI-002, R-CRI-005, REQ-8). The `?with=<slug>`
 * deep-link (D-6) still resolves to an `agentSlug` from staff.ts so
 * the workplace shell's `/chat?with=finance` link keeps working
 * (D-4 inert: it does not drive which conversation loads — D-1).
 *
 * CH-08 mount data flow (REQ-8):
 *   1. `useVisibleTask$` on first paint fires
 *      `GET /api/agent/conversations/:id`  and
 *      `GET /api/agent/conversations` in parallel.
 *   2. The reload endpoint's `ExchangeDTO[]` is projected to
 *      `TranscriptEntry[]` and `useChatStream.reset(entries)` seeds
 *      the buffer — no EventSource opened, no false streaming claim
 *      (REQ-9: a reload never claims the turn is still in flight).
 *   3. The list endpoint's `ConversationSummary[]` populates the
 *      rail via `<ConversationList />` (R-CRI-005).
 *   4. The cross-participant / 404 / empty-list boundaries are
 *      enforced by the backend — see
 *      `backend/agent/src/chat/http.go:HandleReloadConversation`
 *      and `HandleListConversations`.
 */
export interface ChatAppProps {
  readonly youName: string;
  readonly youEmail: string;
  /**
   * Authenticated participant id. Surfaced from `requireSession` in
   * `routes/chat/index.tsx` so the page knows which conversation to
   * load (D-1: `conversationID == participantID`). The page never
   * trusts a URL param or a header for this — the route hands the
   * session's resolved id in as a prop.
   */
  readonly participantID: string;
}

function exchangesToEntries(
  exchanges: readonly { promptText: string; assistantText: string; partial: boolean }[],
): TranscriptEntry[] {
  const out: TranscriptEntry[] = [];
  exchanges.forEach((ex, idx) => {
    out.push({
      kind: "said",
      id: `u${idx}`,
      who: "you",
      text: ex.promptText,
      state: "final",
    });
    out.push({
      kind: "said",
      id: `a${idx}`,
      who: "chat",
      text: ex.assistantText,
      state: ex.partial ? "streaming" : "final",
    });
  });
  return out;
}

export const ChatApp = component$<ChatAppProps>(({ youName, youEmail, participantID }) => {
  const activeSlug = useSignal(AGENTS[0].slug);
  const turn = useChatStream([]);
  const scroller = useSignal<HTMLElement>();

  // CH-08 (REQ-8 / REQ-9): mount-time fetch + seed. On first paint
  // the page fires the two resume GETs in parallel, seeds the
  // transcript with `reset(entries)`, populates the rail. The
  // helper returns silently when offline / cross-participant /
  // unknown — the mount does not throw, so a transient wire
  // failure leaves the page in its CH-05.1 empty-seed shape
  // rather than a blank-by-error state.
  const railState = useStore<{
    summaries: ConversationSummary[];
    loaded: boolean;
    loadError: boolean;
  }>({
    summaries: [],
    loaded: false,
    loadError: false,
  });

  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(({ track }) => {
    track(() => participantID);
    if (typeof window === "undefined") return;

    void (async () => {
      const [listResult, reloadResult] = await Promise.all([
        listConversations(),
        loadConversation(participantID),
      ]);

      // Reload — seed the buffer if the participant has recorded
      // exchanges; a miss or refusal leaves the seed empty.
      if (reloadResult.ok && reloadResult.value.length > 0) {
        await turn.reset(exchangesToEntries(reloadResult.value));
      }

      // List — populate the rail; an offline / 403 / 500 leaves
      // the rail empty rather than falsely populated.
      if (listResult.ok) {
        railState.summaries = listResult.value;
        railState.loaded = true;
        railState.loadError = false;
      } else {
        railState.summaries = [];
        railState.loaded = false;
        railState.loadError = true;
      }
    })();
  });

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
      {/* CH-08 (R-CRI-005): the rail is re-mounted against the wire.
          It was dropped in CH-05.1 (D-3); CH-08 undoes that drop
          because the page now needs to render the participant's
          own conversations. `selectedId` is the participant id
          (D-1) — the rail marks only the active row current. The
          onSelect handler is a no-op stub for v1 (the page has one
          active conversation at v1; future CH-08 leaves will
          expand the rail's interaction surface). */}
      <aside
        class="bg-surface border-line hidden w-64 shrink-0 border-r md:block"
        aria-label="Conversation list"
      >
        <ConversationList
          conversations={railState.summaries}
          selectedId={participantID}
          onSelect$={$(async (id: string) => {
            // v1: the rail is informational. The page holds one
            // active conversation; future CH work wires the rail
            // to a conversation switcher (deferred — CH-08.2
            // own-list surfaces only, the interaction model is a
            // follow-up). The id parameter is reserved for that
            // future and intentionally unused here.
            void id;
          })}
        />
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
