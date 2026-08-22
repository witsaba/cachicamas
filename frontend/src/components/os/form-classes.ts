/**
 * The form vocabulary, in one place.
 *
 * Forms are rare in this product — ownboarding is essentially the only one —
 * but the two that exist must not invent their own field chrome. A field here
 * is the same object as the command line and the composer: a hard-cornered
 * well, a 1px rule, an amber rule when FOCUSED — focus is one of the few
 * things amber is still reserved for — and the machine voice on the label with
 * language in the field itself.
 *
 * Exported as constants rather than a component so Tailwind's scanner keeps
 * the utilities and so a drift test can assert them without a DOM.
 */

/** The label above a field. Machine voice: uppercase, tracked, dim. */
export const FORM_LABEL =
  "block text-legend uppercase tracking-[0.14em] text-fg-dim";

/** The field itself. Language lives inside it, so the human face. */
export const FORM_INPUT = [
  "mt-1",
  "block",
  "w-full",
  "border",
  "border-rule",
  "bg-raise",
  "px-2.5",
  "py-2",
  "font-human",
  "text-body",
  "text-fg",
  "transition-colors",
  "duration-150",
  "hover:border-rule-strong",
  "focus:border-amber",
  "aria-invalid:border-fail",
].join(" ");

/** Helper text under a field. */
export const FORM_HELP = "mt-1 font-human text-data leading-snug text-fg-dim";

/** A per-field validation message. */
export const FORM_ERROR = "mt-1 font-human text-data leading-snug text-fail";

/** A grouped set of fields. */
export const FORM_FIELDSET = "border border-rule bg-panel p-4";

/** The legend on a grouped set. */
export const FORM_LEGEND =
  "px-1.5 text-label uppercase tracking-[0.14em] text-fg";
