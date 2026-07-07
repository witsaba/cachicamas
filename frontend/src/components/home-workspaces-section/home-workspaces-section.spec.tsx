/**
 * HomeWorkspacesSection unit tests.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-015 (S-WS-140..143) — /home workspaces section.
 */
import { $ } from "@builder.io/qwik";
import { describe, test, expect, vi, beforeEach } from "vitest";
import { createDOM } from "@builder.io/qwik/testing";
import {
  HomeWorkspacesSection,
  type HomeWorkspacesSectionProps,
} from "./home-workspaces-section";

const noopQrl = $(() => undefined);
const noopIdQrl = $((_id: number) => undefined);

const DEFAULT_PROPS: HomeWorkspacesSectionProps = {
  loading: false,
  error: null,
  workspaces: [],
  truncated: false,
  onRetry: noopQrl,
  onNavigate: noopIdQrl,
  onCreateWorkspace: noopQrl,
};

describe("HomeWorkspacesSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  test("RED-T-WS-2iii-014 / GREEN: empty state CTA when zero workspaces", async () => {
    const { screen, render } = await createDOM();
    await render(<HomeWorkspacesSection {...DEFAULT_PROPS} />);
    expect(
      screen.querySelector('[data-testid="home-workspaces-empty"]'),
    ).toBeTruthy();
    expect(
      screen.querySelector('[data-testid="home-workspaces-create-cta"]'),
    ).toBeTruthy();
    const link = screen.querySelector(
      '[data-testid="home-workspaces-create-cta"]',
    );
    expect(link?.getAttribute("href")).toBe("/workspaces/new");
  });

  test("TRIANGULATE-T-WS-2iii-016a: 1+ workspaces renders cards + Create CTA", async () => {
    const { screen, render } = await createDOM();
    await render(
      <HomeWorkspacesSection
        {...DEFAULT_PROPS}
        workspaces={[
          {
            id: 1,
            name: "ws-one",
repository: {
              github_id: 1,
              full_name: "octocat/hello",
              owner: "octocat",
              name: "hello",
            },
            created_at: "2026-07-06T00:00:00Z",
          },
        ]}
      />,
    );
    expect(
      screen.querySelector('[data-testid="home-workspaces-list"]'),
    ).toBeTruthy();
    expect(
      screen.querySelector('[data-testid="home-workspaces-create-cta"]'),
    ).toBeTruthy();
  });

  test("TRIANGULATE-T-WS-2iii-016b: loading state shows 'Loading…'", async () => {
    const { screen, render } = await createDOM();
    await render(<HomeWorkspacesSection {...DEFAULT_PROPS} loading />);
    expect(screen.textContent).toContain("Loading");
  });

  test("TRIANGULATE-T-WS-2iii-016c: error state shows alert + Retry button", async () => {
    const { screen, render } = await createDOM();
    await render(
      <HomeWorkspacesSection
        {...DEFAULT_PROPS}
        error="Couldn't load workspaces. Try again."
      />,
    );
    expect(
      screen.querySelector('[data-testid="home-workspaces-error"]'),
    ).toBeTruthy();
    expect(
      screen.querySelector('[data-testid="home-workspaces-retry"]'),
    ).toBeTruthy();
  });

  test("TRIANGULATE-T-WS-2iii-016d: truncated shows the warning text", async () => {
    const { screen, render } = await createDOM();
    await render(
      <HomeWorkspacesSection
        {...DEFAULT_PROPS}
        workspaces={[
          {
            id: 1,
            name: "ws",
repository: {
              github_id: 1,
              full_name: "o/r",
              owner: "o",
              name: "r",
            },
            created_at: "2026-07-06T00:00:00Z",
          },
        ]}
        truncated
      />,
    );
    expect(screen.textContent).toContain("View all");
  });
});
