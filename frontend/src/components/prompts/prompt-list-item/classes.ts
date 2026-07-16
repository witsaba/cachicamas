/**
 * prompt-list-item classes — pure function table.
 *
 * Mirrors the pattern in components/ui/button/classes.ts and
 * components/workspace-sync-card/classes.ts.
 *
 * Aphantasic-friendly (UX-4): text-first. Selected state uses bg-slate-100
 * to ensure sufficient contrast. The text alone is sufficient to convey
 * state.
 */

export interface ListItemClasses {
  container: string;
  slug: string;
  meta: string;
}

/** Tailwind classes for a PromptListItem. */
export function listItemClasses(selected: boolean): ListItemClasses {
  const baseContainer =
    "w-full cursor-pointer rounded px-3 py-2 text-left transition-colors";
  const unselectedContainer = `${baseContainer} hover:bg-slate-50`;
  const selectedContainer = `${baseContainer} bg-slate-100`;

  return {
    container: selected ? selectedContainer : unselectedContainer,
    slug: "block truncate text-sm font-medium text-slate-900",
    meta: "mt-0.5 block truncate text-xs text-slate-500",
  };
}
