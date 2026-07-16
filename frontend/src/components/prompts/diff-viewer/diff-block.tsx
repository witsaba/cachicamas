/**
 * DiffBlock — renders a side-by-side diff between two revisions.
 *
 * Props:
 *   oldRevision  — the older revision (N-1)
 *   newRevision — the newer revision (N)
 *   promptSlug  — used for the restore button
 *   onRestore$  — handler for the restore action
 */

import { component$, type QRL } from "@builder.io/qwik";
import type { PromptRevision } from "~/lib/prompts-api";
import { computeLineDiff } from "~/lib/diff";
import { diffLineClasses } from "./classes";

export interface DiffBlockProps {
  oldRevision: PromptRevision;
  newRevision: PromptRevision;
  onRestore$: QRL<() => void>;
  testId?: string;
}

/** Format a timestamp for display. */
function formatDateTime(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export const DiffBlock = component$<DiffBlockProps>(
  ({ oldRevision, newRevision, onRestore$, testId }) => {
    const diff = computeLineDiff(oldRevision.body, newRevision.body);

    return (
      <div
        class="rounded border border-slate-200 bg-white"
        data-testid={testId ?? "diff-block"}
      >
        {/* Header */}
        <div class="flex items-center justify-between border-b border-slate-100 px-3 py-2">
          <span class="text-xs text-slate-500">
            v{oldRevision.revision_number} &rarr; v{newRevision.revision_number}{" "}
            <span class="text-slate-400">
              ({formatDateTime(oldRevision.created_at)})
            </span>
          </span>
          <button
            type="button"
            onClick$={onRestore$}
            class="text-xs text-blue-600 hover:text-blue-800 hover:underline"
            data-testid="diff-block-restore"
          >
            Restore v{oldRevision.revision_number}
          </button>
        </div>

        {/* Diff lines — side by side */}
        <div class="overflow-x-auto">
          <table class="w-full border-collapse text-sm">
            <tbody>
              {diff.lines.map((line, idx) => {
                const classes = diffLineClasses(line.type);
                const prefix =
                  line.type === "delete"
                    ? "-"
                    : line.type === "insert"
                      ? "+"
                      : " ";
                return (
                  <tr
                    key={idx}
                    class={line.type === "equal" ? "bg-slate-50" : ""}
                    data-testid={`diff-line-${line.type}`}
                  >
                    {/* Left side — old text */}
                    <td class="w-6 min-w-6 border-r border-slate-100 py-0.5 text-center font-mono text-slate-400">
                      {prefix}
                    </td>
                    <td
                      class={`${classes.container} border-r border-slate-100 py-0.5 font-mono text-xs`}
                    >
                      {line.type === "delete" ? line.text : "\u00A0"}
                    </td>
                    {/* Right side — new text */}
                    <td class={`${classes.container} py-0.5 font-mono text-xs`}>
                      {line.type === "insert" ? line.text : "\u00A0"}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    );
  },
);
