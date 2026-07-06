/**
 * OrgPill spec — the header chrome that surfaces the current
 * organization context (R-FIX-002, 2026-07-06).
 *
 * Spec surfaces:
 *   - OrgPill renders the monogram + name when an org exists.
 *   - OrgPill renders a disabled "No organization yet" pill when no
 *     org exists (ownboarding empty state).
 *   - Monogram = first letter of full_name, uppercased. Empty /
 *     whitespace-only names fall back to "?".
 *   - Clicking the trigger opens the panel; the panel shows the
 *     org's full_name + identification, plus a disabled Settings row.
 *   - The empty-state pill does NOT render a button or open a panel.
 *
 * Mocking strategy: the `organization` prop is the documented test
 * escape hatch — the spec supplies the org directly and skips the
 * useResource$ fetch. Mirrors AvatarDropdown's `forceOpen` pattern.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { test, expect, vi } from "vitest";
import { orgMonogram, OrgPill } from "./org-pill";

const ACME_ORG = {
  id: 1,
  shortname: null,
  full_name: "Acme Industrial",
  identification: "acme",
  is_active: true,
  email: null,
  phone: null,
  created_at: "2026-07-06T00:00:00Z",
  updated_at: "2026-07-06T00:00:00Z",
};

// Mock the API client so the useResource$ fetch never runs in tests.
// Each scenario passes the `organization` prop directly, but the
// mock prevents any accidental fetch if the component decides to
// ignore the override (defensive).
vi.mock("~/lib/api", () => ({
  getCurrentOrganization: vi.fn(() => Promise.resolve(null)),
}));

async function renderPill(
  organization: typeof ACME_ORG | null | undefined,
  forceOpen = false,
) {
  const { screen, render } = await createDOM();
  await render(
    <OrgPill organization={organization} forceOpen={forceOpen} />,
  );
  return screen;
}

test("[org-pill]: orgMonogram returns the uppercased first letter of full_name", () => {
  expect(orgMonogram(ACME_ORG)).toBe("A");
  expect(orgMonogram({ ...ACME_ORG, full_name: "braejan" })).toBe("B");
});

test("[org-pill]: orgMonogram falls back to ? for null org", () => {
  expect(orgMonogram(null)).toBe("?");
});

test("[org-pill]: orgMonogram falls back to ? for empty/whitespace full_name", () => {
  expect(orgMonogram({ ...ACME_ORG, full_name: "" })).toBe("?");
  expect(orgMonogram({ ...ACME_ORG, full_name: "   " })).toBe("?");
});

test("[org-pill]: renders the monogram + full_name when an org exists", async () => {
  const screen = await renderPill(ACME_ORG);
  const pill = screen.querySelector('[data-testid="org-pill"]');
  expect(pill).toBeTruthy();
  expect(pill?.textContent).toContain("Acme Industrial");
  const monogram = screen.querySelector(
    '[data-testid="org-pill-monogram"]',
  );
  expect(monogram?.textContent?.trim()).toBe("A");
  // Empty-state pill must NOT render when an org exists.
  expect(screen.querySelector('[data-testid="org-pill-empty"]')).toBeFalsy();
});

test("[org-pill]: panel renders full_name + identification + disabled Settings (forceOpen)", async () => {
  const screen = await renderPill(ACME_ORG, true);
  const panel = screen.querySelector('[data-testid="org-pill-panel"]');
  expect(panel).toBeTruthy();
  const panelName = screen.querySelector(
    '[data-testid="org-pill-panel-name"]',
  );
  expect(panelName?.textContent).toBe("Acme Industrial");
  const panelId = screen.querySelector(
    '[data-testid="org-pill-panel-identification"]',
  );
  expect(panelId?.textContent?.trim()).toBe("id: acme");
  const settings = screen.querySelector(
    '[data-testid="org-pill-panel-settings"]',
  );
  expect(settings).toBeTruthy();
  expect(settings?.getAttribute("aria-disabled")).toBe("true");
  expect(settings?.textContent).toContain("Settings");
});

test("[org-pill]: panel is NOT rendered by default (no forceOpen)", async () => {
  const screen = await renderPill(ACME_ORG);
  expect(screen.querySelector('[data-testid="org-pill-panel"]')).toBeFalsy();
});

test("[org-pill]: empty state renders a disabled-looking pill when org is null", async () => {
  const screen = await renderPill(null);
  const empty = screen.querySelector('[data-testid="org-pill-empty"]');
  expect(empty).toBeTruthy();
  expect(empty?.textContent).toContain("No organization yet");
  // Empty state is a div, NOT a button. No panel.
  expect(screen.querySelector('[data-testid="org-pill"]')).toBeFalsy();
  expect(screen.querySelector('[data-testid="org-pill-panel"]')).toBeFalsy();
  // Monogram placeholder is "?".
  const monogram = screen.querySelector(
    '[data-testid="org-pill-monogram"]',
  );
  expect(monogram?.textContent?.trim()).toBe("?");
});

test("[org-pill]: pill trigger carries aria-label with the org's full_name", async () => {
  const screen = await renderPill(ACME_ORG);
  const pill = screen.querySelector('[data-testid="org-pill"]');
  expect(pill?.getAttribute("aria-label")).toBe("Acme Industrial menu");
  expect(pill?.getAttribute("aria-haspopup")).toBe("menu");
});

test("[org-pill]: loading state renders a neutral pill when org is undefined", async () => {
  // undefined signals "useResource$ still resolving".
  const screen = await renderPill(undefined);
  const loading = screen.querySelector('[data-testid="org-pill-loading"]');
  expect(loading).toBeTruthy();
  expect(loading?.textContent).toContain("Loading");
  expect(screen.querySelector('[data-testid="org-pill"]')).toBeFalsy();
  expect(screen.querySelector('[data-testid="org-pill-empty"]')).toBeFalsy();
});