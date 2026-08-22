/**
 * A screen's title block.
 *
 * One h1, one sentence under it, and whatever action belongs to the whole
 * screen on the right. Every workspace screen opens with this, in the same
 * place, at the same size — consistency screen to screen is the virtue here,
 * and surprise is what a person in a task least wants.
 */
import { Slot, component$ } from "@builder.io/qwik";

export interface PageHeaderProps {
  readonly title: string;
  readonly lede?: string;
}

export const PageHeader = component$<PageHeaderProps>(({ title, lede }) => (
  <header class="flex flex-wrap items-start justify-between gap-3 pb-6">
    <div class="min-w-0">
      <h1 class="text-ink text-xl font-semibold tracking-[-0.01em]">{title}</h1>
      {lede ? (
        <p class="text-ink-mid mt-1 max-w-[62ch] text-base">{lede}</p>
      ) : null}
    </div>
    <div class="flex shrink-0 items-center gap-2">
      <Slot />
    </div>
  </header>
));

/** The standard content well every non-conversation screen sits in. */
export const PAGE_WELL = "mx-auto w-full max-w-5xl px-4 py-7 sm:px-6 lg:px-8";
