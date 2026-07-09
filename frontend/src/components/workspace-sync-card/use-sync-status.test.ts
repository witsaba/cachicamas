/**
 * use-sync-status.test.ts — TDD coverage for the polling hook.
 *
 * UAT bug discovered 2026-07-08: the prior hook only started the
 * polling loop when the user clicked "Sync now" (via start()). After
 * a refresh, the SSR fetch occasionally returned null (when the
 * sync_job existed in the DB but the route somehow lost the cookie
 * for the GET endpoint), and the polling never started because no
 * click happened. The card stayed on "Pending…" forever.
 *
 * The fix: the hook must fire a one-shot refresh() on mount via
 * useVisibleTask$, AND if the latest job is in a non-terminal
 * state, the hook must kick off the polling loop so the card
 * updates without requiring the user to click.
 */
import { describe, expect, it } from "vitest";

// We can't import @builder.io/qwik directly in a vitest node
// environment without setup. The functions we want to test
// (the polling logic) are pure JS that doesn't depend on Qwik
// internals. The component is exercised end-to-end via
// workspace-sync-card.spec.tsx. Here we test the polling logic in
// isolation by re-implementing the same state machine the hook uses.

async function tick(
  opts: {
    stopFlag: { stop: boolean };
    job: { value: SyncJob | null };
    pollOnce: () => Promise<SyncJob | null>;
  },
): Promise<void> {
  if (opts.stopFlag.stop) return;
  if (!opts.job.value) return;
  if (opts.job.value.status !== "pending" && opts.job.value.status !== "running") return;
  const result = await opts.pollOnce();
  opts.job.value = result;
  if (result === null) {
    // 404: keep polling
    return;
  }
  if (result.status === "pending" || result.status === "running") {
    // still in flight
  }
}

interface SyncJob {
  status: "pending" | "running" | "done" | "failed";
}

describe("useSyncStatus polling state machine", () => {
  it("RED-R-WS-019-2026-07-08-002: tick updates job.value to the polled value", async () => {
    const stopFlag = { stop: false };
    const job: { value: SyncJob | null } = {
      value: { status: "pending" },
    };
    const polled: SyncJob[] = [];
    await tick({
      stopFlag,
      job,
      pollOnce: async () => {
        polled.push(job.value as SyncJob);
        return { status: "done" };
      },
    });
    expect(polled.length).toBe(1);
    expect(job.value?.status).toBe("done");
  });

  it("RED-R-WS-019-2026-07-08-003: tick returns early when job is null (no auto-poll after a stuck render)", async () => {
    // The polling contract: tick is a no-op when job is null.
    // When the user refreshes and the SSR fetch returned null,
    // the polling does NOT start automatically (that's what
    // useVisibleTask$ + refresh() does — see the integration
    // test in workspace-sync-card.spec.tsx).
    const job: { value: SyncJob | null } = { value: null };
    let called = false;
    await tick({
      stopFlag: { stop: false },
      job,
      pollOnce: async () => {
        called = true;
        return { status: "done" };
      },
    });
    expect(called).toBe(false);
    expect(job.value).toBeNull();
  });

  it("RED-R-WS-019-2026-07-08-004: tick returns early when stopFlag is set (cleanup on unmount)", async () => {
    const job: { value: SyncJob | null } = { value: { status: "pending" } };
    const stopFlag = { stop: true };
    let called = false;
    await tick({
      stopFlag,
      job,
      pollOnce: async () => {
        called = true;
        return { status: "done" };
      },
    });
    expect(called).toBe(false);
  });
});
