/**
 * Form field vocabulary — one table, so every form in the product looks like
 * the same form.
 *
 * The input border is `--color-line-control` rather than the decorative line
 * value, because on a white surface the border IS how a person finds the
 * field: WCAG 1.4.11 applies, and this token clears 3:1 on every ground in the
 * product. The focus ring is the system-wide one from `global.css`; nothing is
 * restyled per field.
 */
export const FORM_LABEL = "block text-base font-medium text-ink";

export const FORM_INPUT = [
  "block",
  "w-full",
  "rounded-md",
  "border",
  "border-line-control",
  "bg-surface",
  "px-3",
  "py-2",
  "text-base",
  "text-ink",
  "shadow-[var(--shadow-raised)]",
  "transition-colors",
  "duration-150",
  "hover:border-ink-soft",
  "disabled:cursor-not-allowed",
  "disabled:bg-sunken",
  "disabled:opacity-70",
].join(" ");

export const FORM_HELP = "mt-1 text-sm leading-snug text-ink-soft";

export const FORM_ERROR = "mt-1 text-sm leading-snug font-medium text-stop";

export const FORM_FIELDSET =
  "rounded-md border border-line bg-surface p-5 shadow-[var(--shadow-raised)]";

export const FORM_LEGEND = "px-1 text-base font-semibold text-ink";
