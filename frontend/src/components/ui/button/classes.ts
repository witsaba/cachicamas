/**
 * Button primitive — className table, in the workplace's own vocabulary.
 *
 * The world (see `src/global.css`): white surfaces on a cool-neutral page, 1px
 * lines, 6px radii on controls, one brand blue for anything that commits. A
 * button here is what a button is in every modern work tool, drawn carefully:
 * a filled brand rectangle for the action that commits, a bordered white one
 * for everything that does not.
 *
 * Why a pure-function table separate from `button.tsx`:
 *   - The className contract is unit-tested in milliseconds, with no DOM.
 *   - Tailwind 4's content scanner picks up the literal strings here, so the
 *     utilities survive tree-shaking as long as `button.tsx` imports them.
 *   - Anti-drift: a contributor who reaches for a gradient, a hard shadow or a
 *     colour outside the token set breaks `classes.spec.ts` immediately.
 *
 * Variants map to intent:
 *   - `primary`     — brand fill. The one action on a screen that commits.
 *   - `secondary`   — white with a control border. Everything non-committal.
 *   - `destructive` — the stop colour. Refuse, remove, cancel.
 *   - `link`        — brand, underlined. A word in a sentence, not a control.
 *
 * Focus is NOT restyled per variant. `global.css` gives the whole product one
 * focus treatment (a 2px brand ring at 2px offset) so a focused control looks
 * identical on a white card, in a tinted row, and on a coloured fill.
 */

export const BUTTON_BASE = [
  "inline-flex",
  "items-center",
  "justify-center",
  "gap-1.5",
  "rounded-md",
  "border",
  "font-medium",
  "whitespace-nowrap",
  "cursor-pointer",
  "select-none",
  "disabled:cursor-not-allowed",
  "disabled:opacity-45",
  "transition-[background-color,color,border-color,box-shadow]",
  "duration-150",
  "ease-[var(--ease-work)]",
].join(" ");

/** Dense: toolbars, table rows, inline actions. */
export const BUTTON_SIZE_SM = "h-7 px-2.5 text-xs";

/** The product's default control height. */
export const BUTTON_SIZE_MD = "h-9 px-3.5 text-base";

/** A screen's single committing action, and every marketing call to action. */
export const BUTTON_SIZE_LG = "h-11 px-5 text-base";

/**
 * Primary intent — the brand, filled. One per screen. The press state darkens
 * rather than lifting, because a control that moves under the pointer is a
 * control you have to re-aim at.
 */
export const VARIANT_PRIMARY = [
  "border-transparent",
  "bg-brand",
  "text-ink-inverse",
  "shadow-[var(--shadow-raised)]",
  "not-disabled:hover:bg-brand-press",
  "not-disabled:active:bg-brand-press",
].join(" ");

/**
 * Secondary intent — a white control with a border strong enough to find. The
 * border value clears 3:1 on every ground in the product, because on this
 * surface the border IS the control (WCAG 1.4.11).
 */
export const VARIANT_SECONDARY = [
  "border-line-control",
  "bg-surface",
  "text-ink",
  "shadow-[var(--shadow-raised)]",
  "not-disabled:hover:bg-sunken",
  "not-disabled:active:bg-sunken",
].join(" ");

/**
 * Destructive intent — the stop colour, filled. The label always names the
 * destruction in words, because colour is never the only carrier
 * (PRODUCT.md § Accessibility).
 */
export const VARIANT_DESTRUCTIVE = [
  "border-transparent",
  "bg-stop",
  "text-ink-inverse",
  "shadow-[var(--shadow-raised)]",
  "not-disabled:hover:brightness-90",
  "not-disabled:active:brightness-90",
].join(" ");

/**
 * Link intent — brand, underlined, with no control around it. It carries no
 * border, no height and no padding: it is a word in a sentence.
 */
export const VARIANT_LINK = [
  "text-brand",
  "underline",
  "underline-offset-2",
  "font-medium",
  "cursor-pointer",
  "disabled:cursor-not-allowed",
  "disabled:opacity-45",
  "transition-colors",
  "duration-150",
  "hover:text-brand-press",
].join(" ");

/** All four variants. */
export type ButtonVariant = "primary" | "secondary" | "destructive" | "link";

/** Sizes. */
export type ButtonSize = "sm" | "md" | "lg";

/** Variant constants in array form for iteration in tests. */
export const ALL_VARIANTS: readonly ButtonVariant[] = [
  "primary",
  "secondary",
  "destructive",
  "link",
] as const;
export const ALL_SIZES: readonly ButtonSize[] = ["sm", "md", "lg"] as const;

const SIZE_TOKENS: Record<ButtonSize, string> = {
  sm: BUTTON_SIZE_SM,
  md: BUTTON_SIZE_MD,
  lg: BUTTON_SIZE_LG,
};

/**
 * Compose the final className for a `<Button>` variant × size control.
 * Pure function — unit-tested in isolation (no DOM).
 *
 * `consumerClass` is APPENDED to the system tokens — it does NOT replace them.
 */
export function buttonClassName(
  variant: ButtonVariant,
  size: ButtonSize,
  consumerClass?: string,
): string {
  // The link variant is a word, not a control: no base chrome, no size.
  if (variant === "link") {
    return consumerClass
      ? [VARIANT_LINK, consumerClass].join(" ")
      : VARIANT_LINK;
  }

  const tokens = [BUTTON_BASE, SIZE_TOKENS[size]];
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
