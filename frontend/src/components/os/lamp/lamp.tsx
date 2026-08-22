import { component$ } from "@builder.io/qwik";

/**
 * StateLamp — a state, said twice.
 *
 * A 6px filled square plus the state's own word, always together. The lamp is
 * never rendered alone: PRODUCT.md's accessibility commitment forbids meaning
 * carried by colour or mark alone, so the word is not a caption, it is half
 * the component.
 *
 * The colour map is the system's whole state vocabulary, and every screen
 * shares it:
 *   live    — running now, connected, on duty        (green)
 *   build   — has a plan and work in flight           (amber)
 *   hold    — suspended, waiting on a person          (violet)
 *   fail    — errored, denied, cancelled              (red)
 *   ready   — planned, but nothing is happening       (neutral, bright)
 *   idle    — nothing here yet                        (neutral, dim)
 */
export type LampTone = "live" | "build" | "ready" | "hold" | "fail" | "idle";

/*
 * Only a state that is HAPPENING gets a colour. `ready` and `idle` are two
 * brightnesses of neutral, because "planned" and "unplanned" describe an
 * absence of activity — colouring them competes with the three states a person
 * actually has to notice, and cyan in particular is reserved system-wide for
 * "you can go here", which is not a state at all.
 */
const MARK: Record<LampTone, string> = {
  live: "bg-live",
  build: "bg-amber",
  ready: "bg-fg-mid",
  hold: "bg-hold",
  fail: "bg-fail",
  idle: "bg-fg-dim",
};

const WORD: Record<LampTone, string> = {
  live: "text-live",
  build: "text-amber",
  ready: "text-fg-mid",
  hold: "text-hold",
  fail: "text-fail",
  idle: "text-fg-dim",
};

export interface StateLampProps {
  readonly tone: LampTone;
  /** The literal status word. Required — the lamp never stands alone. */
  readonly word: string;
  /** Pulse the mark. Only ever for a state that is genuinely in motion. */
  readonly pulse?: boolean;
  readonly testId?: string;
}

export const StateLamp = component$<StateLampProps>((props) => (
  <span
    data-testid={props.testId}
    data-tone={props.tone}
    class="inline-flex items-center gap-1.5 whitespace-nowrap"
  >
    <span
      aria-hidden="true"
      class={`h-1.5 w-1.5 shrink-0 ${MARK[props.tone]} ${props.pulse ? "term-lamp" : ""}`}
    />
    <span class={`text-legend tracking-[0.14em] uppercase ${WORD[props.tone]}`}>
      {props.word}
    </span>
  </span>
));
