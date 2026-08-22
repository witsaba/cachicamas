import { component$ } from "@builder.io/qwik";

/**
 * Gauge — a count, drawn as filled segments plus the figure itself.
 *
 * Used for milestone progress. Eight segments, filled proportionally, with the
 * literal `done/total` beside them: the segments give the shape at a glance
 * and the figure gives the fact, so neither has to be inferred from the other.
 *
 * A completed gauge fills amber; an incomplete one fills cyan and leaves the
 * remainder as the rule colour. Zero of N is a legitimate, common reading here
 * — three of the six archetypes have never started — and it must look like a
 * real zero rather than a broken component.
 */
export const GAUGE_SEGMENTS = 8;

export interface GaugeProps {
  readonly done: number;
  readonly total: number;
  readonly testId?: string;
}

/** How many of the eight segments are lit. Pure, so it is unit-tested. */
export function litSegments(done: number, total: number): number {
  if (total <= 0 || done <= 0) return 0;
  if (done >= total) return GAUGE_SEGMENTS;
  // Never round a partial up to full: 41 of 42 must not read as complete.
  return Math.max(
    1,
    Math.min(GAUGE_SEGMENTS - 1, Math.round((done / total) * GAUGE_SEGMENTS)),
  );
}

export const Gauge = component$<GaugeProps>((props) => {
  const lit = litSegments(props.done, props.total);
  const complete = props.total > 0 && props.done >= props.total;
  return (
    <span
      data-testid={props.testId}
      class="inline-flex items-center gap-2 whitespace-nowrap"
    >
      <span aria-hidden="true" class="inline-flex gap-px">
        {Array.from({ length: GAUGE_SEGMENTS }).map((_, i) => (
          <span
            key={i}
            class={`h-2 w-1.5 ${
              i < lit ? (complete ? "bg-amber" : "bg-cyan") : "bg-rule"
            }`}
          />
        ))}
      </span>
      <span class={`text-legend ${complete ? "text-amber" : "text-fg-mid"}`}>
        {props.done}/{props.total}
      </span>
    </span>
  );
});
