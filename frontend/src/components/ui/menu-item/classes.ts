/**
 * MenuItem primitive — className table, in the workplace's vocabulary.
 *
 * Why this is a separate primitive from `<Button>`: menu items live inside
 * floating panels. They are full-width rows, left-aligned, and they tint their
 * row on hover rather than filling with brand — a dropped panel that flashed
 * blue under every pointer move would be unreadable.
 *
 * Promoting this to its own primitive prevents the temptation to reach for a
 * `<Button size="sm">` inside a panel, which would bring the wrong padding, the
 * wrong alignment and the wrong hover.
 *
 * Focus is the system-wide treatment from `global.css`; nothing is restyled
 * here.
 */
export const MENU_ITEM_BASE = [
  "flex",
  "w-full",
  "items-center",
  "gap-2",
  "rounded-sm",
  "px-2",
  "py-1.5",
  "text-left",
  "text-base",
  "text-ink-mid",
  "cursor-pointer",
  "disabled:cursor-not-allowed",
  "disabled:opacity-45",
  "transition-[background-color,color]",
  "duration-150",
  "hover:bg-sunken",
  "hover:text-ink",
].join(" ");
