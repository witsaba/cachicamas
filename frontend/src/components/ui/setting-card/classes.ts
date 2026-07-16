/**
 * SettingCard primitive — className table.
 *
 * Reference: `sdd/settings-app-grid/{proposal,spec,design}.md` (engram).
 *
 * Renders a vertical tile: a centered 64×64 slate-100 rounded-xl
 * container that holds a 48×48 consumer-provided icon (`stroke=
 * "currentColor"`), with a text label below it. Two element kinds
 * (anchor / button) share the same tokens in v1 — no size variant.
 *
 * Why a pure-function table separate from `setting-card.tsx`:
 *   - Unit tests for the className contract run in milliseconds (no
 *     DOM, no Qwik createDOM setup).
 *   - Tailwind 4's content scanner picks up the literal strings here,
 *     so the utilities survive tree-shaking as long as `setting-card.tsx`
 *     imports the constants — which every consumer route does.
 *   - Anti-drift: a contributor who removes `cursor-pointer` or
 *     `focus-visible:ring-indigo-500` from CARD_BASE breaks
 *     `classes.spec.ts` immediately.
 *
 * Group-hover plumbing (D5 in the design):
 *   - CARD_BASE has `group` + `hover:bg-slate-50` — the tile itself
 *     tints to slate-50 on hover.
 *   - ICON_CONTAINER has `text-slate-700 group-hover:text-slate-900` —
 *     the icon container's text color flips; the inner SVG's
 *     `stroke="currentColor"` then re-renders with the new color.
 *   - LABEL has `group-hover:text-slate-950` — the label tightens.
 *
 * Tailwind 4 emission-order note:
 *   None of the system tokens collide with each other alphabetically
 *   (bg-* < text-* < group-hover:text-*, etc.), so no `!important`
 *   is needed on any system token. Consumer overrides may still need
 *   the `!important` prefix (`!bg-slate-900`, `!p-0`) per the README's
 *   escape hatch guidance when they conflict with a system variant.
 */

export const CARD_BASE = [
  "group",
  "no-underline",
  "flex",
  "flex-col",
  "items-center",
  "rounded-lg",
  "p-3",
  "cursor-pointer",
  "bg-white",
  "hover:bg-slate-50",
  "transition-[background-color,box-shadow,transform]",
  "duration-150",
  "focus:outline-none",
  "focus-visible:ring-2",
  "focus-visible:ring-indigo-500",
  "focus-visible:ring-offset-2",
  "active:translate-y-px",
  "disabled:opacity-50",
  "disabled:cursor-not-allowed",
].join(" ");

/**
 * 64×64 slate-100 rounded-xl icon well.
 *
 * Pin rationale (design §5):
 *   - `h-16 w-16` (64×64) — tile anchor size.
 *   - `rounded-xl` (12px) — slightly rounder than the tile's
 *     `rounded-lg` so the icon well reads as a distinct shape,
 *     matching macOS Launchpad's "icon in a rounded square" chrome.
 *   - `bg-slate-100` — lighter than a Button (`bg-slate-900`); reads
 *     as an icon well, NOT a CTA surface.
 *   - `text-slate-700 group-hover:text-slate-900` — drives the inner
 *     SVG's stroke via `currentColor`.
 *   - `flex items-center justify-center` — centers the icon.
 */
export const ICON_CONTAINER = [
  "flex",
  "h-16",
  "w-16",
  "items-center",
  "justify-center",
  "rounded-xl",
  "bg-slate-100",
  "text-slate-700",
  "group-hover:text-slate-900",
  "transition-colors",
  "duration-150",
].join(" ");

/**
 * Inner icon size — 48×48 (12px gutter inside the 64×64 container).
 * Reserved for consumer use (e.g. `<svg class={ICON_SIZE}>…`); the
 * `PromptsIcon` component sets `width="48" height="48"` directly.
 */
export const ICON_SIZE = "h-12 w-12";

/**
 * Label styling.
 *
 * Below the icon container, `mt-3` (12px) for the tile's visual
 * rhythm. `text-sm font-medium` so labels do not overpower the icon.
 */
export const LABEL = [
  "mt-3",
  "text-sm",
  "font-medium",
  "text-slate-900",
  "group-hover:text-slate-950",
  "text-center",
].join(" ");

/**
 * Focus ring — same as Button's link variant and MenuItem:
 * `focus-visible:ring-2 focus-visible:ring-indigo-500`.
 *
 * `focus-visible:ring-offset-2` adds a 2px gap from the tile edge —
 * critical on a square shape because the ring would otherwise touch
 * the label and icon, not just the outer border.
 */
export const FOCUS = [
  "focus:outline-none",
  "focus-visible:ring-2",
  "focus-visible:ring-indigo-500",
  "focus-visible:ring-offset-2",
].join(" ");

/**
 * The two element kinds the primitive renders. v1 has no size variant;
 * both render the same tokens via `settingCardClassName`. Kept as a
 * type so the discriminated union in `setting-card.tsx` can narrow.
 */
export type SettingCardVariant = "a" | "button";

/**
 * Compose the final className for a `<SettingCard>`.
 *
 * Variants share the same tokens in v1 (no size variant) — the
 * `variant` argument is informational and currently a no-op. Future
 * variants (e.g. compact, dense) would diverge here.
 *
 * `consumerClass` is APPENDED to the system tokens — it does NOT
 * replace them. Use it for shape overrides that the variant does not
 * anticipate. Same contract as Button and MenuItem.
 */
export function settingCardClassName(
  variant: SettingCardVariant,
  consumerClass?: string,
): string {
  const tokens = [CARD_BASE];
  void variant; // v1: variant is informational; both render the same tokens
  if (consumerClass) tokens.push(consumerClass);
  return tokens.join(" ");
}
