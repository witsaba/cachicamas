/**
 * WorkspaceSyncCard.spec.tsx — strict TDD coverage for the
 * WorkspaceSyncCard component. We mock the api + use-sync-status
 * so we test the rendered DOM, not the polling logic.
 */
import { component$ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, expect, it, vi } from "vitest";

import type { SyncJob } from "~/lib/api";

import { WorkspaceSyncCard } from "./workspace-sync-card";

// Mock the api module so the card doesn't try to hit the real backend.
vi.mock("~/lib/api", async () => {
  const actual = await vi.importActual<typeof import("~/lib/api")>("~/lib/api");
  return {
    ...actual,
    startWorkspaceSync: vi.fn(async (_id: number) => ({
      ok: true,
      value: makeJob("running"),
    })),
    getWorkspaceSyncStatus: vi.fn(async (_id: number) => ({
      ok: true,
      value: makeJob("running"),
    })),
  };
});

function makeJob(status: SyncJob["status"]): SyncJob {
  return {
    job_id: 42,
    workspace_id: 7,
    status,
    triggered_by: "manual",
    started_at: status === "running" ? "2026-07-08T12:00:00Z" : null,
    finished_at: status === "done" ? "2026-07-08T12:05:00Z" : null,
    commit_sha_after: status === "done" ? "abc1234deadbeef" : null,
    error_message: status === "failed" ? "Permission denied" : null,
    error_code: status === "failed" ? "INSUFFICIENT_PERMISSIONS" : null,
    attempts: status === "done" || status === "failed" ? 1 : 0,
    created_at: "2026-07-08T12:00:00Z",
  };
}

async function render(props: { initialJob: SyncJob | null; workspaceId: number }) {
  // Wrap in a parent component because Qwik's createDOM needs a root.
  const Host = component$(() => <WorkspaceSyncCard {...props} />);
  const { screen, render: qwikRender } = await createDOM();
  await qwikRender(<Host />);
  return screen;
}

describe("WorkspaceSyncCard", () => {
  it("renders the heading and 'Sync now' button when no job", async () => {
    const screen = await render({ initialJob: null, workspaceId: 7 });
    const heading = screen.querySelector("#workspace-sync-card-heading");
    expect(heading?.textContent).toBe("Sync");
    const button = screen.querySelector('[data-testid="workspace-sync-card-button"]');
    expect(button?.textContent?.trim()).toBe("Sync now");
    expect(button?.hasAttribute("disabled")).toBe(false);
  });

  it("shows the 'Synced' pill + formatted commit SHA on a done job", async () => {
    const screen = await render({ initialJob: makeJob("done"), workspaceId: 7 });
    const pill = screen.querySelector('[data-testid="workspace-sync-card-pill"]');
    expect(pill?.getAttribute("data-status")).toBe("done");
    expect(pill?.textContent).toMatch(/Synced/);
    const sha = screen.querySelector('[data-testid="workspace-sync-card-commit-sha"]');
    expect(sha?.textContent?.trim()).toBe("abc1234");
  });

  it("disables the button + shows 'Syncing…' label when running", async () => {
    const screen = await render({ initialJob: makeJob("running"), workspaceId: 7 });
    const button = screen.querySelector('[data-testid="workspace-sync-card-button"]');
    expect(button?.textContent?.trim()).toBe("Syncing\u2026");
    expect(button?.hasAttribute("disabled")).toBe(true);
  });

  it("shows 'Failed' pill + inline error banner on a failed job", async () => {
    const screen = await render({ initialJob: makeJob("failed"), workspaceId: 7 });
    const pill = screen.querySelector('[data-testid="workspace-sync-card-pill"]');
    expect(pill?.getAttribute("data-status")).toBe("failed");
    const banner = screen.querySelector('[data-testid="workspace-sync-card-error"]');
    expect(banner?.textContent).toMatch(/Permission denied/);
    const button = screen.querySelector('[data-testid="workspace-sync-card-button"]');
    expect(button?.textContent?.trim()).toBe("Retry sync");
  });

  it("renders em-dash for missing commit SHA + last synced at", async () => {
    const screen = await render({ initialJob: null, workspaceId: 7 });
    const sha = screen.querySelector('[data-testid="workspace-sync-card-commit-sha"]');
    expect(sha?.textContent?.trim()).toBe("\u2014");
    const last = screen.querySelector('[data-testid="workspace-sync-card-last-synced-at"]');
    expect(last?.textContent?.trim()).toBe("\u2014");
  });
});