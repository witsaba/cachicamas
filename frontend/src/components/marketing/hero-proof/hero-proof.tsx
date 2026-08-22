/**
 * The public page's proof, playing.
 *
 * The product's entire argument is that a colleague stops before doing
 * anything that leaves the building. A still picture of a stopped colleague
 * does not demonstrate stopping — so this plays: the answer arrives as it is
 * written, then the work halts on a permission and waits. The buttons are
 * real; answering one is the fastest way to understand what the product does.
 *
 * It runs on the same machine as the workspace conversation (`useMockTurn`)
 * and renders through the same `TranscriptLine`. That is deliberate: the
 * public page cannot drift into showing something the product does not do,
 * because it is the product doing it.
 *
 * This is the page's ONE authored moment. Nothing else on it moves, and under
 * `prefers-reduced-motion` the whole exchange is present on first paint —
 * finished, not animated at zero duration.
 */
import { $, component$, useSignal, useVisibleTask$ } from "@builder.io/qwik";
import { TranscriptLine } from "~/components/chat/transcript-line";
import { useMockTurn } from "~/components/chat/use-mock-turn";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import { HERO_OPENING, HERO_SCRIPT } from "~/lib/mock/chat";
import type { Agent } from "~/lib/mock/staff";

export interface HeroProofProps {
  readonly agent: Agent;
}

export const HeroProof = component$<HeroProofProps>(({ agent }) => {
  const turn = useMockTurn([HERO_OPENING]);
  const started = useSignal(false);

  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(async ({ cleanup }) => {
    if (typeof window === "undefined") return;
    if (started.value) return;
    started.value = true;

    const reduced = window.matchMedia?.(
      "(prefers-reduced-motion: reduce)",
    )?.matches;
    if (reduced) {
      await turn.settle(HERO_SCRIPT);
      return;
    }

    // A short beat before it starts, so the headline is read first and the
    // movement is noticed rather than missed.
    const handle = window.setTimeout(() => void turn.play(HERO_SCRIPT), 900);
    cleanup(() => window.clearTimeout(handle));
  });

  return (
    <div
      class="border-line bg-surface rounded-lg border shadow-[var(--shadow-raised)]"
      data-testid="hero-proof"
    >
      <header class="border-line flex items-center gap-3 border-b px-4 py-3">
        <AgentAvatar agent={agent} size="md" />
        <span class="min-w-0 flex-1">
          <span class="flex items-center gap-2">
            <span class="text-ink text-base font-semibold">{agent.name}</span>
            <span class="border-brand/25 bg-brand-tint text-brand text-2xs rounded-sm border px-1.5 py-px font-semibold tracking-wide uppercase">
              Agent
            </span>
          </span>
          <span class="text-ink-soft block text-xs">
            {agent.departmentName} · on staff {agent.tenure}
          </span>
        </span>
      </header>

      {/* A fixed floor, so the card does not grow under the reader as the
          answer arrives. Growth is the kind of motion nobody asked for. */}
      <ol
        class="min-h-[19rem] px-4 pb-1"
        aria-live="polite"
        aria-label="An example conversation"
      >
        {turn.state.entries.map((entry) => (
          <TranscriptLine
            key={entry.id}
            entry={entry}
            agent={agent}
            youName="Ana Rivas"
            youInitials="AR"
            onDecide$={$((granted: boolean) => {
              void turn.decide(granted);
            })}
          />
        ))}
      </ol>

      <p class="border-line text-ink-soft border-t px-4 py-2.5 text-xs">
        An example conversation. Nothing is sent until a person answers.
      </p>
    </div>
  );
});
