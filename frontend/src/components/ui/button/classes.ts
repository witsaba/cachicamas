/**
 * Button primitive — className table.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/{proposal,design,specs/frontend-ui-button/spec}.md`
 *
 * Why a pure-function table separate from `button.tsx`:
 *   - Unit tests for the className contract run in milliseconds (no DOM,
 *     no Qwik createDOM setup).
 *   - Tailwind 4's content scanner picks up the literal strings here, so
 *     the utilities survive tree-shaking as long as `button.tsx` imports
 *     the constants — which every consumer route does.
 *   - Anti-drift: a contributor who removes `cursor-pointer` from a
 *     variant breaks `classes.spec.ts` immediately.
 *
 * Variants:
 *   - `primary` (create / update): bg-slate-900, the project monochrome.
 *   - `secondary` (general-purpose outline): bg-white + border-slate-300.
 *   - `destructive` (delete / remove): bg-red-700.
 *   - `link` (bare-underline text button): no surface, just an underlined
 *     text link styled as a button. Used by `workspace-form-clear-repo`,
 *     `github-repo-picker-refresh`, etc.
 *
 * All filled variants share the base affordances:
 *   - `cursor-pointer` for cross-OS consistency (some OSes don't apply
 *     cursor:pointer to <button> natively; the user reported this as a
 *     pain point).
 *   - `disabled:cursor-not-allowed` so disabled buttons read as
 *     non-interactive.
 *   - `transition-[background-color,box-shadow,transform,border-color]
 *     duration-150` for the hover/press feedback uniform across the
 *     product.
 *   - `focus:outline-none focus-visible:ring-2` so keyboard users see a
 *     clear ring (the focus ring color varies by variant).
 *
 * The `not-disabled:hover:*` utilities fix the a11y bug where disabled
 * buttons still respond visually to mouse-over. The `not-` variant
 * prefix is built into Tailwind 4 and renders as `:not(:disabled):hover:*`.
 *
 * The `link` variant is text-only and does not use the BUTTON_BASE chrome
 * (no flex, no padding, no rounded-md). It duplicates the cursor / focus
 * / active / disabled affordances inside VARIANT_LINK so the variant
 * stands alone — a text link styled like a button.
 *
 * Consumers can APPEND custom classes via the `<Button class="...">`
 * prop for shape overrides (e.g. circular, full-width, dark-surface
 * hover). The system tokens always apply first; consumer tokens layer on
 * top.
 */

export const BUTTON_BASE = [
  "inline-flex",
  "items-center",
  "justify-center",
  "gap-2",
  "rounded-md",
  "font-medium",
  "cursor-pointer",
  "disabled:cursor-not-allowed",
  "transition-[background-color,box-shadow,transform,border-color]",
  "duration-150",
  "focus:outline-none",
  "focus-visible:ring-2",
].join(" ");

/** Default size: `text-sm` with `px-4 py-2`. Used by all form submits, retry buttons, header CTAs. */
export const BUTTON_SIZE_MD = "px-4 py-2 text-sm";

/** Hero / empty-state size: `text-base` with `px-5 py-3`. */
export const BUTTON_SIZE_LG = "px-5 py-3 text-base";

/**
 * Primary intent — the "do something persistent" CTA.
 *
 * Ownboarding-form was previously `bg-indigo-600` (project monochrome
 * drift); after this system lands, all primary CTAs use `bg-slate-900`.
 * Documented as a visual regression aligned with the project rule.
 */
export const VARIANT_PRIMARY = [
  "bg-slate-900",
  "text-white",
  "not-disabled:hover:bg-slate-700",
  "active:translate-y-px",
  "focus-visible:ring-indigo-500",
  "disabled:opacity-50",
].join(" ");

/**
 * Secondary intent — general-purpose outline button.
 * Used by Cancel links, the org-pill trigger, secondary actions.
 */
export const VARIANT_SECONDARY = [
  "bg-white",
  "border",
  "border-slate-300",
  "text-slate-900",
  "not-disabled:hover:bg-slate-50",
  "active:translate-y-px",
  "focus-visible:ring-indigo-500",
  "disabled:opacity-50",
].join(" ");

/**
 * Destructive intent — delete / remove actions.
 *
 * NO `active:translate-y-px`: a heavy destructive button should not
 * "press" — the visual weight signals permanence, and a press would
 * feel like the page is being shaken.
 */
export const VARIANT_DESTRUCTIVE = [
  "bg-red-700",
  "text-white",
  "not-disabled:hover:bg-red-800",
  "focus-visible:ring-red-500",
  "disabled:opacity-50",
].join(" ");

/**
 * Link intent — bare-underline text button (no surface).
 *
 * Uses `transition-colors` (lighter transition than the filled variants)
 * because the hover state is a text color change, not a surface swap.
 */
export const VARIANT_LINK = [
  "text-slate-700",
  "underline",
  "transition-colors",
  "duration-150",
  "cursor-pointer",
  "disabled:cursor-not-allowed",
  "hover:text-slate-900",
  "focus:outline-none",
  "focus-visible:ring-2",
  "focus-visible:ring-indigo-500",
  "active:translate-y-px",
  "disabled:opacity-50",
].join(" ");

/** All four variants. */
export type ButtonVariant = "primary" | "secondary" | "destructive" | "link";

/** Sizes. */
export type ButtonSize = "md" | "lg";

/** Variant constants in array form for iteration in tests. */
export const ALL_VARIANTS: readonly ButtonVariant[] = [
  "primary",
  "secondary",
  "destructive",
  "link",
] as const;
export const ALL_SIZES: readonly ButtonSize[] = ["md", "lg"] as const;

/**
 * Compose the final className for a `<Button>` variant × size cell.
 * Pure function — unit-tested in isolation (no DOM).
 *
 * `consumerClass` is APPENDED to the system tokens — it does NOT
 * replace them. Use it for shape overrides (circular, full-width,
 * dark-surface hover) that the variants don't anticipate.
 */
export function buttonClassName(
  variant: ButtonVariant,
  size: ButtonSize,
  consumerClass?: string,
): string {
  // The link variant is text-only and does not use BUTTON_BASE or size.
  if (variant === "link") {
    return consumerClass
      ? [VARIANT_LINK, consumerClass].join(" ")
      : VARIANT_LINK;
  }

  const tokens = [BUTTON_BASE, size === "lg" ? BUTTON_SIZE_LG : BUTTON_SIZE_MD];
  switch (variant) {
    case "primary":
      tokens.push(VARIANT_PRIMARY);
      break;
    case "secondary":
      tokens.push(VARIANT_SECONDARY);
      break;
    case "destructive":
      tokens.push(VARIANT_DESTRUCTIVE);
      break;
  }
  if (consumerClass) tokens.push(consumerClass);
  return tokens.join(" ");
}
