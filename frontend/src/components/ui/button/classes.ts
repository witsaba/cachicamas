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
 * Variant tokens are tuned for the project:
 *   - `primary` (create / update): bg-slate-900, the project monochrome.
 *   - `secondary` (general-purpose): bg-white + border-slate-300, the
 *     outline pair that the org-pill trigger and cancel actions use.
 *   - `destructive` (delete / remove): bg-red-700, the user's preferred
 *     destructive color (per the original change request).
 *
 * All variants share the base affordances:
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
 * The fifth `link` variant is added in PR-3 (per `design.md` §6
 * "Open question resolved") — not part of PR-1.
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

/** All three PR-1 variants. The `link` variant is added in PR-3. */
export type ButtonVariant = "primary" | "secondary" | "destructive";

/** PR-1 sizes. */
export type ButtonSize = "md" | "lg";

/**
 * Compose the final className for a `<Button>` variant × size cell.
 * Pure function — unit-tested in isolation (no DOM).
 */
export function buttonClassName(
  variant: ButtonVariant,
  size: ButtonSize,
): string {
  const sizeClass = size === "lg" ? BUTTON_SIZE_LG : BUTTON_SIZE_MD;
  const variantClass = (() => {
    switch (variant) {
      case "primary":
        return VARIANT_PRIMARY;
      case "secondary":
        return VARIANT_SECONDARY;
      case "destructive":
        return VARIANT_DESTRUCTIVE;
    }
  })();
  return [BUTTON_BASE, sizeClass, variantClass].join(" ");
}
