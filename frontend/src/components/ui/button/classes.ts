/**
 * Button primitive — className table, in the terminal's own vocabulary.
 *
 * The world (see `src/global.css`): hard rectangles, 1px rules, no shadow, no
 * radius, colour only ever carrying state. A button here is a keyed cell on a
 * board, not a pill — so it is a bordered rectangle with an uppercase machine
 * label, and its press feedback is REVERSE VIDEO: the fill and the text swap.
 * That is what a terminal does when you hit a key, and it is the only press
 * affordance in the product.
 *
 * Why a pure-function table separate from `button.tsx`:
 *   - The className contract is unit-tested in milliseconds, with no DOM.
 *   - Tailwind 4's content scanner picks up the literal strings here, so the
 *     utilities survive tree-shaking as long as `button.tsx` imports them.
 *   - Anti-drift: a contributor who reintroduces a radius, a shadow or a
 *     slate-era colour breaks `classes.spec.ts` immediately.
 *
 * Variants map to intent, and intent maps to one of the five working colours:
 *   - `primary`     — amber. The system's own action: submit, open, confirm.
 *   - `secondary`   — a rule-bordered cell. Everything non-committal.
 *   - `destructive` — fail red. Refuse, delete, cancel a run.
 *   - `link`        — cyan, underlined. Cyan means "you can go there".
 *
 * Focus is NOT restyled per variant. `global.css` gives the whole system one
 * focus treatment (a 1px amber outline with a 1px offset) so a focused control
 * looks identical on a panel, in a well, and on the void.
 */

export const BUTTON_BASE = [
  "inline-flex",
  "items-center",
  "justify-center",
  "gap-2",
  "border",
  "font-system",
  "uppercase",
  "tracking-[0.08em]",
  "whitespace-nowrap",
  "cursor-pointer",
  "disabled:cursor-not-allowed",
  "disabled:opacity-40",
  "transition-[background-color,color,border-color]",
  "duration-150",
].join(" ");

/** Default size: the density of a toolbar cell. */
export const BUTTON_SIZE_MD = "px-3 py-1.5 text-label";

/** The one size up, for a screen's single committing action. */
export const BUTTON_SIZE_LG = "px-4 py-2.5 text-body";

/**
 * Primary intent — an amber filled cell that reverses on interaction.
 * Amber is the machine's own voice, so a primary button reads as the system
 * offering to do the thing rather than as a decorated call to action.
 */
export const VARIANT_PRIMARY = [
  "border-amber",
  "bg-amber",
  "text-void",
  "not-disabled:hover:bg-void",
  "not-disabled:hover:text-amber",
  "not-disabled:active:bg-void",
  "not-disabled:active:text-amber",
].join(" ");

/**
 * Secondary intent — an empty cell with a rule around it. Used for anything
 * that does not commit: cancel, back, refresh, secondary navigation.
 */
export const VARIANT_SECONDARY = [
  "border-rule-strong",
  "bg-transparent",
  "text-fg",
  "not-disabled:hover:border-amber",
  "not-disabled:hover:text-amber",
  "not-disabled:active:bg-raise",
].join(" ");

/**
 * Destructive intent — the fail colour, filled. Reverses like the others, so
 * the vocabulary stays uniform; the colour alone carries the warning, and the
 * label always names the destruction in words (PRODUCT.md § Accessibility).
 */
export const VARIANT_DESTRUCTIVE = [
  "border-fail",
  "bg-fail",
  "text-void",
  "not-disabled:hover:bg-void",
  "not-disabled:hover:text-fail",
  "not-disabled:active:bg-void",
  "not-disabled:active:text-fail",
].join(" ");

/**
 * Link intent — cyan and underlined, with no cell around it.
 *
 * Cyan is reserved system-wide for "this goes somewhere", so a link button
 * inherits that meaning for free. It carries no border, no padding and no
 * uppercase treatment: it is a word in a sentence, not a control on a board.
 */
export const VARIANT_LINK = [
  "text-cyan",
  "underline",
  "underline-offset-2",
  "cursor-pointer",
  "disabled:cursor-not-allowed",
  "disabled:opacity-40",
  "transition-colors",
  "duration-150",
  "hover:text-fg",
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
 * `consumerClass` is APPENDED to the system tokens — it does NOT replace them.
 */
export function buttonClassName(
  variant: ButtonVariant,
  size: ButtonSize,
  consumerClass?: string,
): string {
  // The link variant is a word, not a cell: no base chrome, no size.
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
