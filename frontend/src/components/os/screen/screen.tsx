import { component$, Slot } from "@builder.io/qwik";

/**
 * ScreenTitle — the title block at the head of an application.
 *
 * Borrowed from the title block of an engineering drawing, which is the same
 * problem solved a century earlier: say what this sheet is, what revision it
 * is at, and under whose authority, in a fixed place, every time.
 *
 * `code` is the command you would have typed to get here, so the title block
 * teaches the command line by repetition rather than by a tour.
 */
export interface ScreenTitleProps {
  /** The command-line code for this screen. */
  readonly code: string;
  readonly title: string;
  /** One line on what this screen is for. */
  readonly lead?: string;
  readonly testId?: string;
}

export const ScreenTitle = component$<ScreenTitleProps>((props) => (
  <div
    data-testid={props.testId ?? "screen-title"}
    class="border-rule border-b pb-3"
  >
    <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
      <span class="border-amber-dim text-legend text-amber border px-1.5 py-px tracking-[0.18em] uppercase">
        {props.code}
      </span>
      <h1 class="text-screen text-fg leading-none font-semibold tracking-tight">
        {props.title}
      </h1>
      <span class="flex-1" />
      <Slot />
    </div>
    {props.lead ? (
      <p class="font-human text-body text-fg-mid mt-2 max-w-[74ch] leading-relaxed">
        {props.lead}
      </p>
    ) : null}
  </div>
));
