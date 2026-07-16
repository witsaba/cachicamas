/**
 * diff-viewer classes — pure function table for diff line styling.
 *
 * Matches the project design system (workspace-sync-card/classes.ts pattern).
 *
 * Color scheme: GitHub-style diff colors.
 *   - delete: red background (user removed content)
 *   - insert: green background (user added content)
 *   - equal:  white/gray (unchanged, for context)
 *
 * Aphantasic-friendly: each type has both color AND text label in the UI.
 */

export type DiffLineType = "delete" | "insert" | "equal";

export interface DiffLineClasses {
  container: string;
}

/** Tailwind classes for a diff line container. */
export function diffLineClasses(type: DiffLineType): DiffLineClasses {
  const base = "px-3 py-1 font-mono text-sm";
  switch (type) {
    case "delete":
      return {
        container: `${base} bg-red-50 text-red-900`,
      };
    case "insert":
      return {
        container: `${base} bg-emerald-50 text-emerald-900`,
      };
    case "equal":
      return {
        container: `${base} bg-slate-50 text-slate-700`,
      };
  }
}
