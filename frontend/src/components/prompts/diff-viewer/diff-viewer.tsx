/**
 * DiffViewer — collapsible history panel.
 *
 * Collapsed by default (user preference). Expands to show all revisions
 * with diff blocks between consecutive versions.
 *
 * Props:
 *   revisions        — all revisions, sorted newest first
 *   currentRevision — the current revision number
 *   promptSlug      — for restore calls
 *   onRestore$      — handler called with (slug, revisionNumber)
 */

import { component$, useSignal, type QRL } from "@builder.io/qwik";
import type { PromptRevision } from "~/lib/prompts-api";
import { DiffBlock } from "./diff-block";

export interface DiffViewerProps {
  revisions: PromptRevision[];
  currentRevision: number;
  promptSlug: string;
  onRestore$: QRL<(slug: string, revisionNumber: number) => void>;
  testId?: string;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export const DiffViewer = component$<DiffViewerProps>(
  ({ revisions, currentRevision, onRestore$, testId }) => {
    const expanded = useSignal(false);

    if (!expanded.value) {
      return (
        <div class="mt-2" data-testid={testId ?? "diff-viewer"}>
          <button
            type="button"
            onClick$={() => {
              expanded.value = true;
            }}
            class="flex items-center gap-1 text-sm text-slate-600 hover:text-slate-900"
            data-testid="diff-viewer-expand"
          >
            <svg
              class="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width={2}
              aria-hidden="true"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M19 9l-7 7-7-7"
              />
            </svg>
            History ({revisions.length})
          </button>
        </div>
      );
    }

    return (
      <div
        class="mt-2 space-y-3"
        data-testid={testId ?? "diff-viewer"}
        role="region"
        aria-label="Version history"
      >
        {/* Collapse button */}
        <button
          type="button"
          onClick$={() => {
            expanded.value = false;
          }}
          class="flex items-center gap-1 text-sm text-slate-600 hover:text-slate-900"
          data-testid="diff-viewer-collapse"
        >
          <svg
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width={2}
            aria-hidden="true"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M5 15l7-7 7 7"
            />
          </svg>
          Hide history
        </button>

        {/* Revision list — newest first */}
        {revisions.map((rev, idx) => {
          const isCurrent = rev.revision_number === currentRevision;

          return (
            <div
              key={rev.id}
              data-testid={`diff-revision-${rev.revision_number}`}
            >
              {idx < revisions.length - 1 ? (
                // Show diff between this and the previous revision
                <DiffBlock
                  oldRevision={revisions[idx + 1]}
                  newRevision={rev}
                  onRestore$={() => {
                    if (!isCurrent)
                      onRestore$(rev.prompt_id.toString(), rev.revision_number);
                  }}
                  testId={`diff-block-${rev.revision_number}`}
                />
              ) : (
                // Oldest revision — no diff, just show it
                <div class="rounded border border-slate-200 bg-white px-3 py-2 text-sm text-slate-500">
                  v{rev.revision_number} — {formatDate(rev.created_at)}
                  {isCurrent && (
                    <span class="ml-2 text-xs font-medium text-slate-700">
                      (current)
                    </span>
                  )}
                  <p class="mt-1 truncate font-mono text-xs text-slate-400">
                    {rev.body.slice(0, 80)}
                    {rev.body.length > 80 ? "…" : ""}
                  </p>
                </div>
              )}
            </div>
          );
        })}
      </div>
    );
  },
);
