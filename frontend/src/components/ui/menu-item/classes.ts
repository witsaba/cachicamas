/**
 * MenuItem primitive — className table, in the terminal's vocabulary.
 *
 * Why this is a separate primitive from `<Button>`: menu items live inside
 * dropped panels (the identity menu). They are full-width rows, left-aligned,
 * and they tint their row on hover rather than reversing a cell — a dropped
 * panel that flashed amber on every pointer move would be unreadable.
 *
 * Promoting this to its own primitive prevents the temptation to reach for a
 * `<Button size="xs">` inside a panel, which would bring the wrong padding,
 * the wrong alignment and the wrong hover.
 *
 * Focus is the system-wide treatment from `global.css`; nothing is restyled
 * here.
 */
export const MENU_ITEM_BASE = [
  "block",
  "w-full",
  "text-left",
  "px-3",
  "py-2",
  "font-system",
  "text-label",
  "uppercase",
  "tracking-[0.08em]",
  "text-fg-mid",
  "cursor-pointer",
  "disabled:cursor-not-allowed",
  "disabled:opacity-40",
  "transition-[background-color,color]",
  "duration-150",
  "hover:bg-raise",
  "hover:text-amber",
].join(" ");
