import { component$, Slot } from "@builder.io/qwik";

/**
 * Panel — the only container in the system.
 *
 * Everything on a screen is a panel: a hard rectangle, a 1px rule around it,
 * and a header band carrying an amber label on the left and an optional
 * machine-readable note on the right. There is no card, no tile, no well and
 * no second container shape. A screen is composed by tiling panels, the way a
 * dealing-room board is composed by tiling readouts.
 *
 * The header band is what makes a panel legible at a glance across a dense
 * board. Its label is deliberately NEUTRAL: panel headings were amber in the
 * first build, and twenty amber marks on one screen turned the world's five
 * working colours into a single accent. Structure is neutral; the lamps and
 * the caret are what carry colour. The note on the right is where a count, a
 * state or a reference goes — never a control.
 */
export interface PanelProps {
  /** The region's name. Rendered uppercase, in the machine voice. */
  readonly label: string;
  /** Optional right-hand readout: a count, a reference, a timestamp. */
  readonly note?: string;
  /** Set false when the body supplies its own padding (tables, lists). */
  readonly padded?: boolean;
  /** Heading level, so a screen's panels form a real document outline. */
  readonly as?: "h2" | "h3";
  readonly testId?: string;
  readonly class?: string;
  /** Extra tokens for the body wrapper, for panels that must grow or scroll. */
  readonly bodyClass?: string;
}

export const Panel = component$<PanelProps>((props) => {
  const padded = props.padded !== false;
  const Heading = props.as ?? "h2";
  return (
    <section
      data-testid={props.testId}
      class={`border-rule bg-panel border ${props.class ?? ""}`}
    >
      <header class="border-rule flex items-baseline justify-between gap-3 border-b px-3 py-1.5">
        <Heading class="text-label text-fg tracking-[0.14em] uppercase">
          {props.label}
        </Heading>
        {props.note ? (
          <span
            data-testid={props.testId ? `${props.testId}-note` : undefined}
            class="text-legend text-fg-dim tracking-[0.14em] uppercase"
          >
            {props.note}
          </span>
        ) : null}
      </header>
      <div class={`${padded ? "p-3" : ""} ${props.bodyClass ?? ""}`}>
        <Slot />
      </div>
    </section>
  );
});
