/**
 * workspace-card.spec.tsx — TDD coverage for WorkspaceCard.
 *
 * Reference: T-WS-2i-008..010.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { WorkspaceCard } from "./workspace-card";
import type { WorkspaceSummary } from "~/lib/api";

function makeWorkspace(over: Partial<WorkspaceSummary> = {}): WorkspaceSummary {
  return {
    id: 42,
    name: "alpha",
    primary_repository: {
      github_id: 100,
      full_name: "octocat/hello-world",
      owner: "octocat",
      name: "hello-world",
    },
    linked_repos_count: 2,
    created_at: "2026-07-01T00:00:00Z",
    ...over,
  };
}

describe("WorkspaceCard", () => {
  it("RED-T-WS-2i-008: renders name, primary repository, linked count, and Open link", async () => {
    const { screen, render } = await createDOM();
    await render(<WorkspaceCard workspace={makeWorkspace()} />);

    const name = screen.querySelector('[data-testid="workspace-card-name"]');
    const repo = screen.querySelector('[data-testid="workspace-card-repo"]');
    const linked = screen.querySelector(
      '[data-testid="workspace-card-linked-count"]',
    );
    const open = screen.querySelector('[data-testid="workspace-card-open"]');
    expect(name?.textContent).toBe("alpha");
    expect(repo?.textContent).toContain("octocat/hello-world");
    expect(linked?.textContent).toBe("2 linked repos");
    expect(open?.getAttribute("href")).toBe("/workspaces/42");
  });

  it("TRIANGULATE-T-WS-2i-010(a): linked_repos_count=0 → 'No linked repos'", async () => {
    const { screen, render } = await createDOM();
    await render(
      <WorkspaceCard workspace={makeWorkspace({ linked_repos_count: 0 })} />,
    );
    const linked = screen.querySelector(
      '[data-testid="workspace-card-linked-count"]',
    );
    expect(linked?.textContent).toBe("No linked repos");
  });

  it("TRIANGULATE-T-WS-2i-010(b): linked_repos_count=5 → '5 linked repos'", async () => {
    const { screen, render } = await createDOM();
    await render(
      <WorkspaceCard workspace={makeWorkspace({ linked_repos_count: 5 })} />,
    );
    const linked = screen.querySelector(
      '[data-testid="workspace-card-linked-count"]',
    );
    expect(linked?.textContent).toBe("5 linked repos");
  });

  it("TRIANGULATE-T-WS-2i-010(c): linked_repos_count=1 → '1 linked repo' (singular)", async () => {
    const { screen, render } = await createDOM();
    await render(
      <WorkspaceCard workspace={makeWorkspace({ linked_repos_count: 1 })} />,
    );
    const linked = screen.querySelector(
      '[data-testid="workspace-card-linked-count"]',
    );
    expect(linked?.textContent).toBe("1 linked repo");
  });
});
