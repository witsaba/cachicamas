/**
 * skill-list-item classes — pure function table.
 *
 * Mirrors the pattern from `components/prompts/prompt-list-item/classes.ts`.
 *
 * Aphantasic-friendly (UX-4): text-first. Selected state uses bg-slate-100
 * for sufficient contrast. The text alone (name + version) is enough to
 * convey state — no color-only signal.
 */

export interface SkillListItemClasses {
  container: string;
  name: string;
  meta: string;
}

/**
 * Tailwind classes for a SkillListItem.
 *
 * @param selected — true if this item is the currently selected skill
 * @returns class table for the container button, name span, and meta span
 */
export function listItemClasses(selected: boolean): SkillListItemClasses {
  const baseContainer =
    "w-full cursor-pointer rounded px-3 py-2 text-left transition-colors";
  const unselectedContainer = `${baseContainer} hover:bg-slate-50`;
  const selectedContainer = `${baseContainer} bg-slate-100`;

  return {
    container: selected ? selectedContainer : unselectedContainer,
    name: "block truncate text-sm font-medium text-slate-900",
    meta: "mt-0.5 block truncate text-xs text-slate-500",
  };
}
