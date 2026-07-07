/**
 * /workspaces index.spec.tsx — TDD coverage for the list page.
 *
 * Reference: T-WS-2i-011..013.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import WorkspacesIndex from "./index";

describe("/workspaces list page (PR2-i)", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("RED-T-WS-2i-011: empty state renders when API returns no workspaces", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ workspaces: [], truncated: false }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const { screen, render } = await createDOM();
    await render(<WorkspacesIndex />);

    // Wait for the useTask$ to resolve.
    await new Promise((r) => setTimeout(r, 10));

    const empty = screen.querySelector('[data-testid="workspaces-empty"]');
    expect(empty).toBeTruthy();
    const cta = screen.querySelector('[data-testid="create-workspace-cta"]');
    expect(cta?.getAttribute("href")).toBe("/workspaces/new");
  });

  it("TRIANGULATE-T-WS-2i-013(a): populated list renders cards", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          workspaces: [
            {
              id: 1,
              name: "alpha",
              primary_repository: {
                github_id: 100,
                full_name: "octocat/hello-world",
                owner: "octocat",
                name: "hello-world",
              },
              linked_repos_count: 2,
              created_at: "2026-07-01T00:00:00Z",
            },
          ],
          truncated: false,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    const { screen, render } = await createDOM();
    await render(<WorkspacesIndex />);
    await new Promise((r) => setTimeout(r, 10));

    const list = screen.querySelector('[data-testid="workspaces-list"]');
    expect(list).toBeTruthy();
    const card = screen.querySelector('[data-testid="workspace-card"]');
    expect(card).toBeTruthy();
  });

  it("TRIANGULATE-T-WS-2i-013(b): API error renders alert + retry button", async () => {
    globalThis.fetch = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ error: "server", message: "Boom." }), {
          status: 500,
        }),
      );
    const { screen, render } = await createDOM();
    await render(<WorkspacesIndex />);
    await new Promise((r) => setTimeout(r, 10));

    const alert = screen.querySelector('[data-testid="workspaces-error"]');
    expect(alert).toBeTruthy();
    const retry = screen.querySelector('[data-testid="workspaces-retry"]');
    expect(retry).toBeTruthy();
  });
});
