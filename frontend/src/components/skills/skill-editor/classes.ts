/**
 * skill-editor classes — pure function table for form-control class strings.
 *
 * The editor's layout (flex containers, dividers, padding) stays
 * inline because it's structural and one-off. This file captures the
 * field class strings that may be reused or themed in the future
 * (e.g. for the empty-state's "create your first skill" affordance
 * in PR2c, which reuses the description input look-and-feel).
 */

export interface SkillEditorClasses {
  /** Description input (text). */
  descriptionInput: string;
  /** Body textarea (multi-line). */
  bodyTextarea: string;
  /** Footer divider / row container. */
  footerRow: string;
  /** "No changes to save" hint text. */
  noChangesHint: string;
}

/**
 * Tailwind classes for the SkillEditor form controls.
 *
 * Returns a static table — there's no per-instance variation, but the
 * helper exists so (a) tests can verify the contract and (b) future
 * variants (compact mode, disabled mode) can be added without
 * rewriting call sites.
 */
export function skillEditorClasses(): SkillEditorClasses {
  return {
    descriptionInput:
      "flex-1 rounded border border-slate-200 px-2 py-1 text-sm text-slate-900 placeholder-slate-400 focus:border-slate-400 focus:ring-1 focus:ring-slate-400 focus:outline-none",
    bodyTextarea:
      "flex-1 resize-none rounded border border-slate-200 px-3 py-2 font-mono text-sm text-slate-900 placeholder-slate-400 focus:border-slate-400 focus:ring-1 focus:ring-slate-400 focus:outline-none",
    footerRow:
      "flex items-center justify-between border-t border-slate-200 px-4 py-3",
    noChangesHint: "text-xs text-slate-400",
  };
}
