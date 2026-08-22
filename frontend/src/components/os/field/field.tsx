import { component$, Slot } from "@builder.io/qwik";

/**
 * Field — one labelled reading.
 *
 * A dim label on the left, the value on the right, a dotted leader between
 * them so the eye can cross a wide panel without losing the row. This is the
 * atom every dense readout in the product is built from; a screen that needs
 * a "spec table" uses a stack of these rather than inventing a table.
 */
export interface FieldProps {
  readonly label: string;
  readonly testId?: string;
}

export const Field = component$<FieldProps>((props) => (
  <div
    data-testid={props.testId}
    class="text-data flex items-baseline gap-2 py-0.5"
  >
    <span class="text-legend text-fg-dim tracking-[0.12em] whitespace-nowrap uppercase">
      {props.label}
    </span>
    <span
      aria-hidden="true"
      class="border-rule min-w-4 flex-1 border-b border-dotted"
    />
    <span class="text-fg text-right whitespace-nowrap">
      <Slot />
    </span>
  </div>
));
