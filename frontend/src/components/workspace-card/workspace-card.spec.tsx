/**
 * workspace-card.spec.tsx — TDD coverage for WorkspaceCard.
 *
 * Reference: T-WS-2i-008..010.
 *
 * 2026-07-08-workspaces-simplify changelog:
 *   - Dropped the 3 tests that exercised the linked_repos_count
 *     badge (No linked repos / N linked repos / singular/plural
 *     agreement). The card no longer renders that data — the
 *     workspace IS its repo in the 1:1 model.
 *   - Replaced those tests with one that pins the post-simplify
 *     card shape: name + repo + Open link, no linked count.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { WorkspaceCard } from "./workspace-card";
import type { WorkspaceSummary } from "~/lib/api";

function makeWorkspace(over: Partial<WorkspaceSummary> = {}): WorkspaceSummary {
  return {
    id: 42,
    name: "alpha",
    repository: {
      github_id: 100,
      full_name: "octocat/hello-world",
      owner: "octocat",
      name: "hello-world",
    },
    created_at: "2026-07-01T00:00:00Z",
    ...over,
  };
}

describe("WorkspaceCard", () => {
  it("RED-T-WS-2i-008 (updated 2026-07-08): renders name, repository, and Open link", async () => {
    const { screen, render } = await createDOM();
    await render(<WorkspaceCard workspace={makeWorkspace()} />);

    const name = screen.querySelector('[data-testid="workspace-card-name"]');
    const repo = screen.querySelector('[data-testid="workspace-card-repo"]');
    const open = screen.querySelector('[data-testid="workspace-card-open"]');
    expect(name?.textContent).toBe("alpha");
    expect(repo?.textContent).toContain("octocat/hello-world");
    expect(open?.getAttribute("href")).toBe("/workspaces/42");
  });

  it("2026-07-08: NO linked_repos_count element rendered (1:1 model)", async () => {
    const { screen, render } = await createDOM();
    await render(<WorkspaceCard workspace={makeWorkspace()} />);
    // Regression: pre-simplify, the card rendered a linked-count badge
    // (data-testid="workspace-card-linked-count"). Post-simplify it
    // MUST NOT exist on the card. (qwik/testing's querySelector
    // returns `undefined` rather than `null` when the selector misses.)
    const linked = screen.querySelector(
      '[data-testid="workspace-card-linked-count"]',
    );
    expect(linked).toBeFalsy();
  });
});