// use-sync-status.ts — Qwik hook that polls the
// /workspaces/:id/sync endpoint while a sync is in flight.
//
// Polling strategy (locked, see spec R-WS-019 S-WS-196):
//   - Initial state: no job, no polling.
//   - On startWorkspaceSync(): enqueue + start polling every 3s.
//   - On each poll:
//     - 404 (workspace_not_synced_yet): treat as "never synced";
//       stop polling, render the "Sync now" CTA.
//     - 200 with status=pending|running: continue polling.
//     - 200 with status=done|failed: stop polling; render the
//       final state.
//   - The hook also returns a refresh() function the UI can call
//     to force a one-shot poll (e.g. after the user retries).
//
// Why a custom hook and not just useTask$ on the workspace
// signal: useTask$ runs once per workspace id change, and we
// need to start/stop polling imperatively (the start button is
// the trigger, not the page mount).

import {
  $,
  type QRL,
  useSignal,
  useStore,
  useVisibleTask$,
} from "@builder.io/qwik";

import {
  getWorkspaceSyncStatus,
  startWorkspaceSync,
  type SyncJob,
} from "~/lib/api";

const POLL_INTERVAL_MS = 3000;

export interface UseSyncStatusResult {
  /** The latest sync_job row, or null if no job has ever been
   *  enqueued for this workspace. */
  job: { value: SyncJob | null };
  /** True while a poll is in flight (the card renders a tiny
   *  spinner). */
  polling: { value: boolean };
  /** True while a manual startWorkspaceSync is in flight. */
  starting: { value: boolean };
  /** Last error from any of the API calls. */
  error: { value: string | null };
  /** Imperatively enqueue a new sync + start polling. */
  start: QRL<() => Promise<void>>;
  /** Force a one-shot poll. Useful after the user clicks "Retry
   *  sync" and the job is already pending. */
  refresh: QRL<() => Promise<void>>;
}

export interface UseSyncStatusOptions {
  /** Initial job (the workspace detail endpoint surfaces
   *  last_synced_at + last_sync_job_id). The hook does NOT
   *  re-fetch on mount; it trusts the initial value. */
  initialJob: SyncJob | null;
  /** The workspace id, used to enqueue + poll. */
  workspaceId: number;
}

export function useSyncStatus(opts: UseSyncStatusOptions): UseSyncStatusResult {
  const job = useSignal<SyncJob | null>(opts.initialJob);
  const polling = useSignal(false);
  const starting = useSignal(false);
  const error = useSignal<string | null>(null);
  const stopFlag = useStore({ stop: false });

  const refresh = $(async () => {
    if (polling.value) return;
    polling.value = true;
    error.value = null;
    try {
      const result = await getWorkspaceSyncStatus(opts.workspaceId);
      if (result.ok) {
        job.value = result.value;
      } else {
        error.value = result.message;
      }
    } finally {
      polling.value = false;
    }
  });

  const start = $(async () => {
    if (starting.value) return;
    starting.value = true;
    error.value = null;
    stopFlag.stop = false;
    try {
      const result = await startWorkspaceSync(opts.workspaceId);
      if (result.ok) {
        job.value = result.value;
        // Only poll if the new job is in a non-terminal state.
        if (
          result.value.status === "pending" ||
          result.value.status === "running"
        ) {
          pollLoop();
        }
      } else {
        error.value = result.message;
      }
    } finally {
      starting.value = false;
    }
  });

  const pollLoop = $(async () => {
    // Tail-recursive polling using setTimeout. The loop
    // terminates on terminal state (done|failed) or when
    // stopFlag.stop is set (e.g. component unmount).
    const tick = async (): Promise<void> => {
      if (stopFlag.stop) return;
      if (!job.value) return;
      if (job.value.status !== "pending" && job.value.status !== "running")
        return;
      const result = await getWorkspaceSyncStatus(opts.workspaceId);
      if (stopFlag.stop) return;
      if (result.ok) {
        if (result.value === null) {
          // 404: never synced. The new enqueue might be racing
          // with the GET; we keep polling once more.
          setTimeout(tick, POLL_INTERVAL_MS);
          return;
        }
        job.value = result.value;
        if (
          result.value.status === "pending" ||
          result.value.status === "running"
        ) {
          setTimeout(tick, POLL_INTERVAL_MS);
        }
      } else {
        error.value = result.message;
        setTimeout(tick, POLL_INTERVAL_MS);
      }
    };
    setTimeout(tick, POLL_INTERVAL_MS);
  });

  // Stop the loop on component unmount. useVisibleTask$ runs
  // only on the client; the cleanup fires on navigation away.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$((ctx: { cleanup: (fn: () => void) => void }) => {
    ctx.cleanup(() => {
      stopFlag.stop = true;
    });
  });

  return { job, polling, starting, error, start, refresh };
}
