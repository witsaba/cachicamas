/**
 * WorkspaceSyncCard — sync card on the workspace detail page.
 *
 * 2026-07-08-workspace-sync-clone PR-4: implements R-WS-019.
 * Mounted on /workspaces/:id below the existing repository
 * section. Shows the latest sync state (last_synced_at +
 * last_synced_commit_sha + status) and exposes a "Sync now"
 * button. While a sync is in flight, the button is disabled
 * with a "Syncing…" label; on failure, an inline banner shows
 * error_message + a "Retry sync" button.
 *
 * Aphantasic-friendly (UX-4): text-first. No decorative
 * iconography. The status pill uses a short uppercase label
 * (Pending / Syncing / Synced / Failed) and a color.
 *
 * Design system rule (cachicamas UAT catch): the primary CTA
 * uses bg-slate-900 (project monochrome). No tinted "open" link.
 *
 * State machine (matches backend spec R-WS-019 S-WS-190..200):
 *   - job is null + neverSynced=true → empty-state with "Sync
 *     now" button. After first click, the auto_on_create job
 *     is enqueued server-side and the card transitions.
 *   - job.status=pending|running → disabled button + status pill
 *     + polling hook updates the row.
 *   - job.status=done → enabled button, "Last synced at…" line.
 *   - job.status=failed → enabled button ("Retry sync"), inline
 *     error banner with error_message.
 */
import { component$ } from "@builder.io/qwik";

import type { SyncJob } from "~/lib/api";

import {
  formatCommitSha,
  formatTimestamp,
  statusPillClasses,
  syncButtonClasses,
} from "./classes";
import { useSyncStatus } from "./use-sync-status";

export interface WorkspaceSyncCardProps {
  /** The latest sync job for this workspace, or null if no job
   *  has ever been enqueued. Comes from the workspace detail
   *  endpoint (denormalized columns last_sync_job_id + last_synced_*). */
  initialJob: SyncJob | null;
  /** The workspace id. */
  workspaceId: number;
}

export const WorkspaceSyncCard = component$<WorkspaceSyncCardProps>(
  ({ initialJob, workspaceId }) => {
    const { job, starting, error, start } = useSyncStatus({
      initialJob,
      workspaceId,
    });

    const currentJob = job.value;
    const button = syncButtonClasses(currentJob, starting.value);
    const pill = currentJob
      ? statusPillClasses(currentJob.status)
      : statusPillClasses("pending");

    return (
      <section
        data-testid="workspace-sync-card"
        data-workspace-id={workspaceId}
        aria-labelledby="workspace-sync-card-heading"
        class="mt-8 rounded-lg border border-slate-200 bg-white px-5 py-4 shadow-sm"
      >
        <header class="flex items-start justify-between gap-4">
          <div>
            <h2
              id="workspace-sync-card-heading"
              class="text-lg font-semibold text-slate-900"
            >
              Sync
            </h2>
            <p class="mt-1 text-sm text-slate-600">
              Clone this repository server-side to enable worktrees and PRs.
            </p>
          </div>
          <span
            data-testid="workspace-sync-card-pill"
            data-status={currentJob?.status ?? "pending"}
            class={pill.container}
          >
            <span class="font-medium tracking-wide uppercase">
              {pill.label}
            </span>
          </span>
        </header>

        <dl class="mt-4 grid grid-cols-1 gap-2 text-sm sm:grid-cols-2">
          <div>
            <dt class="font-medium text-slate-700">Last synced at</dt>
            <dd
              data-testid="workspace-sync-card-last-synced-at"
              class="font-mono text-slate-900"
            >
              {formatTimestamp(currentJob?.finished_at ?? null)}
            </dd>
          </div>
          <div>
            <dt class="font-medium text-slate-700">Commit SHA</dt>
            <dd
              data-testid="workspace-sync-card-commit-sha"
              class="font-mono text-slate-900"
            >
              {formatCommitSha(currentJob?.commit_sha_after ?? null)}
            </dd>
          </div>
        </dl>

        {currentJob?.status === "failed" && currentJob.error_message ? (
          <div
            data-testid="workspace-sync-card-error"
            role="alert"
            class="mt-4 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800"
          >
            <p class="font-medium">Sync failed</p>
            <p class="mt-1">{currentJob.error_message}</p>
          </div>
        ) : null}

        {error.value ? (
          <div
            data-testid="workspace-sync-card-error-api"
            role="alert"
            class="mt-4 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800"
          >
            {error.value}
          </div>
        ) : null}

        <div class="mt-4 flex items-center justify-end">
          <button
            type="button"
            data-testid="workspace-sync-card-button"
            aria-disabled={
              starting.value ||
              (currentJob !== null &&
                (currentJob.status === "pending" ||
                  currentJob.status === "running"))
            }
            disabled={
              starting.value ||
              (currentJob !== null &&
                (currentJob.status === "pending" ||
                  currentJob.status === "running"))
            }
            class={button.root}
            onClick$={() => {
              void start();
            }}
          >
            {button.label}
          </button>
        </div>
      </section>
    );
  },
);
