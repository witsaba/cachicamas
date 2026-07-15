// classes.spec.ts — strict TDD coverage for the pure function
// table in classes.ts. No DOM, no Qwik — runs in node.
import { describe, expect, it } from "vitest";

import { formatCommitSha, formatTimestamp, statusPillClasses, syncButtonClasses } from "./classes";
import type { SyncJob } from "~/lib/api";

function makeJob(status: SyncJob["status"]): SyncJob {
  return {
    job_id: 1,
    workspace_id: 7,
    status,
    triggered_by: "manual",
    started_at: null,
    finished_at: null,
    commit_sha_after: null,
    error_message: null,
    error_code: null,
    attempts: 0,
    created_at: "2026-07-08T12:00:00Z",
  };
}

describe("statusPillClasses", () => {
  it("pending: slate background", () => {
    const c = statusPillClasses("pending");
    expect(c.label).toBe("Pending");
    expect(c.container).toMatch(/bg-slate-100/);
  });

  it("running: blue background", () => {
    const c = statusPillClasses("running");
    expect(c.label).toBe("Syncing");
    expect(c.container).toMatch(/bg-blue-100/);
  });

  it("done: emerald background", () => {
    const c = statusPillClasses("done");
    expect(c.label).toBe("Synced");
    expect(c.container).toMatch(/bg-emerald-100/);
  });

  it("failed: red background", () => {
    const c = statusPillClasses("failed");
    expect(c.label).toBe("Failed");
    expect(c.container).toMatch(/bg-red-100/);
  });
});

describe("syncButtonClasses", () => {
  it("no job: enabled, 'Sync now'", () => {
    const c = syncButtonClasses(null, false);
    expect(c.label).toBe("Sync now");
    expect(c.root).not.toMatch(/cursor-not-allowed/);
  });

  it("isStarting: disabled, 'Pending…'", () => {
    const c = syncButtonClasses(null, true);
    expect(c.label).toBe("Pending\u2026");
    expect(c.root).toMatch(/cursor-not-allowed/);
  });

  it("job status=pending: disabled", () => {
    const c = syncButtonClasses(makeJob("pending"), false);
    expect(c.label).toBe("Pending\u2026");
    expect(c.root).toMatch(/cursor-not-allowed/);
  });

  it("job status=running: disabled, 'Syncing…'", () => {
    const c = syncButtonClasses(makeJob("running"), false);
    expect(c.label).toBe("Syncing\u2026");
    expect(c.root).toMatch(/cursor-not-allowed/);
  });

  it("job status=done: enabled, 'Sync now'", () => {
    const c = syncButtonClasses(makeJob("done"), false);
    expect(c.label).toBe("Sync now");
    expect(c.root).not.toMatch(/cursor-not-allowed/);
  });

  it("job status=failed: enabled, 'Retry sync'", () => {
    const c = syncButtonClasses(makeJob("failed"), false);
    expect(c.label).toBe("Retry sync");
    expect(c.root).not.toMatch(/cursor-not-allowed/);
  });
});

describe("formatCommitSha", () => {
  it("returns em-dash for null", () => {
    expect(formatCommitSha(null)).toBe("\u2014");
  });

  it("returns em-dash for empty string", () => {
    expect(formatCommitSha("")).toBe("\u2014");
  });

  it("truncates to 7 chars", () => {
    expect(formatCommitSha("abc1234567890def")).toBe("abc1234");
  });

  it("returns full string when <= 7 chars", () => {
    expect(formatCommitSha("abc123")).toBe("abc123");
  });
});

describe("formatTimestamp", () => {
  it("returns em-dash for null", () => {
    expect(formatTimestamp(null)).toBe("\u2014");
  });

  it("returns em-dash for invalid input", () => {
    expect(formatTimestamp("not a date")).toBe("\u2014");
  });

  it("formats ISO-8601 to YYYY-MM-DD HH:MM (local)", () => {
    const out = formatTimestamp("2026-07-08T12:34:00Z");
    expect(out).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/);
  });
});