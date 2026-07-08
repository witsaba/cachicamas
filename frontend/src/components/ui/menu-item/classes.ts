/**
 * MenuItem primitive — className table.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/{proposal,design,specs/frontend-ui-button/spec}.md`
 *
 * Why a separate primitive from `<Button>`:
 *   Menu items sit inside dropdown panels (avatar dropdown, org-pill
 *   panel). They have:
 *     - tighter padding (px-2 py-1.5 vs px-4 py-2)
 *     - block layout (full width of the panel)
 *     - text-left alignment (panel labels are left-aligned, not centered)
 *     - hover bg-slate-100 (tint the row, not swap a button surface)
 *
 * Promoting this to a separate primitive prevents the temptation to
 * "use a Button with size='xs'" inside panels — which would produce
 * the wrong padding and the wrong hover color.
 *
 * The affordances overlap with `<Button>`:
 *   - `cursor-pointer` + `disabled:cursor-not-allowed` (R-UB-002)
 *   - the same transition set + duration-150 (R-UB-003)
 *   - `focus-visible:ring-indigo-500` (R-UB-004 — same as primary/secondary)
 *   - `active:translate-y-px` (R-UB-003 — same press feedback)
 *   - `disabled:opacity-50`
 */
export const MENU_ITEM_BASE = [
  "block",
  "w-full",
  "text-left",
  "px-2",
  "py-1.5",
  "text-sm",
  "cursor-pointer",
  "disabled:cursor-not-allowed",
  "transition-[background-color,box-shadow,transform,border-color]",
  "duration-150",
  "hover:bg-slate-100",
  "focus:outline-none",
  "focus-visible:ring-2",
  "focus-visible:ring-indigo-500",
  "active:translate-y-px",
  "disabled:opacity-50",
].join(" ");
