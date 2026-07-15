// use-sync-status.ts — Qwik hook that subscribes to the
// /workspaces/:id/sync/stream SSE endpoint and surfaces the
// latest sync_job state.
//
// Architecture (UAT fix 2026-07-08):
//   - PRIOR: polling every 3s via getWorkspaceSyncStatus. Fragile:
//     a click that returned 202 + job_id=1 sometimes never updated
//     the card UI because the polling tick fired from inside a QRL
//     closure whose signal subscription didn't propagate. The card
//     stayed on "Pending…" until a hard refresh re-ran the SSR
//     fetch.
//   - CURRENT: SSE. The server streams a single JSON event per
//     state change. The client subscribes once and updates
//     job.value on every event. Zero polling, zero lag, zero
//     QRL closure issues (EventSource is a browser global,
//     the callbacks are regular async functions).
//
// Lifecycle:
//   - On mount: open the EventSource, subscribe to events.
//   - On every event: update job.value (the Qwik signal that
//     the card's render reads).
//   - On unmount: close the EventSource (no leaks).
//   - On EventSource error: log and try a one-shot refresh() as
//     a recovery (the stream may have died; a fresh fetch on the
//     REST endpoint can repair the card state).

import { $, type QRL, useSignal, useVisibleTask$ } from "@builder.io/qwik";

import {
  getWorkspaceSyncStatus,
  startWorkspaceSync,
  subscribeWorkspaceSyncStream,
  type SyncJob,
} from "~/lib/api";

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
  /** Imperatively enqueue a new sync. The SSE subscription will
   *  receive the new state and update job.value. */
  start: QRL<() => Promise<void>>;
  /** Force a one-shot poll. Useful after the user clicks "Retry
   *  sync" and the SSE stream happens to be down. */
  refresh: QRL<() => Promise<void>>;
}

export interface UseSyncStatusOptions {
  /** Initial job (the workspace detail endpoint surfaces
   *  last_synced_at + last_sync_job_id). The hook does NOT
   *  re-fetch on mount; it trusts the initial value AND
   *  subscribes to the SSE stream for live updates. */
  initialJob: SyncJob | null;
  /** The workspace id, used to enqueue + subscribe. */
  workspaceId: number;
}

export function useSyncStatus(opts: UseSyncStatusOptions): UseSyncStatusResult {
  const job = useSignal<SyncJob | null>(opts.initialJob);
  const polling = useSignal(false);
  const starting = useSignal(false);
  const error = useSignal<string | null>(null);

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
    try {
      const result = await startWorkspaceSync(opts.workspaceId);
      if (result.ok) {
        // The SSE stream will deliver the next state event
        // (status=pending immediately, then status=done within
        // ~1s of the syncer's callback). We optimistically set
        // job.value here so the button disables immediately.
        job.value = result.value;
      } else {
        error.value = result.message;
      }
    } finally {
      starting.value = false;
    }
  });

  // SSE subscription. Runs only on the client (useVisibleTask$
  // fires when the component becomes visible). The cleanup
  // function closes the EventSource on unmount.
  //
  // We do NOT subscribe on the server (Qwik's SSR passes
  // initialJob through the route's useTask$ — the SSE is purely
  // for live updates from the browser).
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$((ctx: { cleanup: (fn: () => void) => void }) => {
    const unsubscribe = subscribeWorkspaceSyncStream(
      opts.workspaceId,
      (updated) => {
        // Each event updates the signal. The card re-renders
        // automatically because the Qwik component subscribes to
        // the signal via the render.
        job.value = updated;
      },
      () => {
        // Stream error: log and try a one-shot refresh. The
        // refresh may recover the state if the stream is down
        // but the REST endpoint is up. If the refresh also
        // fails, the error.value signal is set and the card
        // shows the error banner.
        error.value = "Live updates lost; refreshing…";
        // Fire-and-forget: we don't await because the cleanup
        // is about to run anyway on a real unmount. The
        // refresh() QRL handles its own state.
        refresh();
      },
    );
    ctx.cleanup(unsubscribe);
  });

  return { job, polling, starting, error, start, refresh };
}
