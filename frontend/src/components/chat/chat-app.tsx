import {
  $,
  component$,
  useSignal,
  useTask$,
  useVisibleTask$,
} from "@builder.io/qwik";
import { Panel } from "~/components/os/panel/panel";
import { StateLamp } from "~/components/os/lamp/lamp";
import { ScreenTitle } from "~/components/os/screen/screen";
import { CONVERSATIONS } from "~/lib/mock/chat";
import { Composer } from "./composer";
import { ConversationList } from "./conversation-list";
import { TranscriptLine } from "./transcript-line";
import { useMockTurn } from "./use-mock-turn";

/**
 * ChatApp — the chat archetype's screen.
 *
 * Two columns: the archetype's own memory on the left, the run on the right.
 * The composer sits at the foot of the run column, and the whole thing is one
 * panel pair inside the OS shell — the same frame every other application gets.
 *
 * What is worth pressing here, because it is the product's actual argument:
 * ask it to drop or delete something. The turn stops mid-run and asks, showing
 * the exact call it wants to make, and nothing moves until you answer. That
 * suspension is a Layer 2 mechanism, not a modal this screen invented, and it
 * is the reason a company can let a specialist agent near its systems at all.
 */
export const ChatApp = component$(() => {
  const selected = useSignal(CONVERSATIONS[0].id);
  const turn = useMockTurn(CONVERSATIONS[0].entries);
  const scroller = useSignal<HTMLElement>();

  // Selecting a conversation loads its transcript. A turn in flight belongs to
  // the conversation it was opened in, so switching ends it rather than
  // carrying it across.
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

  // Following a stream to the bottom of a scroller is a browser-only concern
  // by definition; there is no server-side equivalent to fall back to.
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

  const statusWord =
    turn.state.status === "running"
      ? "Streaming"
      : turn.state.status === "held"
        ? "Suspended"
        : "Idle";

  return (
    <main
      id="main"
      class="mx-auto flex min-h-0 w-full max-w-[1800px] flex-1 flex-col px-3 py-4 sm:px-4"
    >
      <ScreenTitle
        code="CHAT"
        title="Chat"
        lead="The thinnest archetype there is: one conversation, one model, and a hand-off to whichever specialist actually owns the work."
      >
        <StateLamp tone="build" word="In build · doc 0005 · 0 of 12" />
      </ScreenTitle>

      {/* The archetype's memory, twice: a standing panel where there is room
          for one, and a disclosure where there is not. Hiding it outright below
          `lg` would take multi-conversation history off phones entirely, which
          is a capability disappearing, not a layout adapting. */}
      <details
        data-testid="conversations-disclosure"
        class="border-rule bg-panel mt-4 border lg:hidden"
      >
        <summary class="border-rule text-label text-fg cursor-pointer border-b px-3 py-1.5 tracking-[0.14em] uppercase">
          Conversations · {CONVERSATIONS.length} · demo
        </summary>
        <ConversationList
          conversations={CONVERSATIONS}
          selectedId={selected.value}
          onSelect$={$((id: string) => {
            selected.value = id;
          })}
        />
      </details>

      <div class="mt-3 grid min-h-0 flex-1 grid-cols-1 gap-3 lg:mt-4 lg:grid-cols-[18rem_minmax(0,1fr)]">
        <Panel
          label="Conversations"
          note={`${CONVERSATIONS.length} · demo`}
          padded={false}
          testId="conversations-panel"
          class="hidden lg:block"
        >
          <ConversationList
            conversations={CONVERSATIONS}
            selectedId={selected.value}
            onSelect$={$((id: string) => {
              selected.value = id;
            })}
          />
        </Panel>

        <Panel
          label={conversation.title}
          note={`${statusWord} · ${conversation.turns} turns`}
          padded={false}
          testId="transcript-panel"
          class="flex min-h-0 flex-col"
          bodyClass="flex min-h-0 flex-1 flex-col"
        >
          <ol
            ref={scroller}
            data-testid="transcript"
            aria-live="polite"
            aria-label="Conversation"
            class="divide-rule min-h-0 flex-1 divide-y overflow-y-auto px-3"
          >
            {turn.state.entries.map((entry) => (
              <TranscriptLine
                key={entry.id}
                entry={entry}
                onDecide$={$((granted: boolean) => {
                  void turn.decide(granted);
                })}
              />
            ))}
          </ol>

          <Composer
            status={turn.state.status}
            onSubmit$={turn.submit}
            onCancel$={turn.cancel}
          />
        </Panel>
      </div>
    </main>
  );
});
