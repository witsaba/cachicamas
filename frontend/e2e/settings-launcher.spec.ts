import { test, expect, type Page } from "@playwright/test";

/**
 * End-to-end spec: `/settings` launcher grid → click Prompts tile →
 * land on `/settings/prompts` (Prompt Studio).
 *
 * Reference: `sdd/settings-app-grid/{proposal,spec,design}.md` (engram).
 *   - R-SAG-NAV-001 (avatar menu Settings links to /settings)
 *   - SCN-10.2 (tile grid renders the Prompts tile)
 *   - REQ-14 / SCN-14.1 (deep link to /settings/prompts still works
 *     — the destination route is unchanged, only the launcher grid
 *     is new in this PR)
 *
 * What this spec verifies against the mocks simulator:
 *   - The avatar dropdown's Settings menu item renders with
 *     `href="/settings"` (REQ-12 — the menu-destination flip).
 *   - Navigating to `/settings` renders the grid wrapper
 *     (`data-testid="settings-grid"`) and exactly one tile
 *     (`data-testid="settings-card-prompts"`).
 *   - Clicking the Prompts tile navigates to `/settings/prompts`
 *     and renders the Prompt Studio (its `<h1>Prompts</h1>` header).
 *
 * Skip semantics:
 *   The whole `describe` is gated on `E2E_BASE_URL` because it
 *   requires docker compose with mocks-github-oauth + database_administrator
 *   running. CI sets `E2E_BASE_URL`; local dev without compose skips
 *   the spec (matching the pattern in `workspaces.spec.ts`).
 *
 * Pre-requisites (mirrors `workspaces.spec.ts`):
 *   - `docker compose up` (Postgres + database_administrator +
 *     mocks-github-oauth + frontend).
 *   - `AUTH_GITHUB_BASE_URL=http://mocks-github-oauth:3016` set on the
 *     frontend container so the OAuth roundtrip resolves.
 *   - A clean DB row in `organization` so `requireOwnboarding` lets
 *     the user past the gate (the workspaces e2e flow handles this
 *     upstream by signing in first).
 */

async function signInViaMocks(page: Page): Promise<void> {
  await page.goto("/");
  await page.waitForLoadState("networkidle");
  const signInForm = page.locator('form[data-testid="sign-in-button"]');
  await expect(signInForm).toBeVisible();
  await signInForm.locator('button[type="submit"]').click();
  // /auth/signin redirects to GitHub, GitHub redirects back, frontend
  // exchanges the code, lands on /profile (or /ownboarding if no org,
  // or /home if both succeed).
  await page.waitForURL(/\/(profile|ownboarding|home)/, { timeout: 15_000 });
}

test.describe("/settings launcher grid", () => {
  test.skip(
    !process.env.E2E_BASE_URL,
    "Playwright e2e requires docker compose stack with mocks-github-oauth; skipped locally",
  );

  test("authenticated user can navigate from the avatar menu → /settings → /settings/prompts via the Prompts tile", async ({
    page,
    context,
  }) => {
    await context.clearCookies();
    await signInViaMocks(page);

    // 1. Avatar dropdown's "Settings" menu item carries the new
    //    destination href (`/settings`, not `/settings/prompts`).
    //    REQ-12 / R-SAG-NAV-001.
    const avatarTrigger = page.locator(
      'button[data-testid="avatar-dropdown"]',
    );
    await expect(avatarTrigger).toBeVisible();
    await avatarTrigger.click();
    const settingsMenuItem = page.locator(
      '[data-testid="avatar-menu-settings"]',
    );
    await expect(settingsMenuItem).toBeVisible();
    await expect(settingsMenuItem).toHaveAttribute("href", "/settings");

    // 2. Navigate to /settings. The grid wrapper renders with the
    //    documented testid and the Prompts tile is visible.
    //    SCN-10.2.
    await page.goto("/settings");
    await page.waitForLoadState("networkidle");
    const grid = page.locator('[data-testid="settings-grid"]');
    await expect(grid).toBeVisible();
    const promptsTile = page.locator(
      '[data-testid="settings-card-prompts"]',
    );
    await expect(promptsTile).toBeVisible();

    // 3. Click the Prompts tile → URL becomes /settings/prompts
    //    → Prompt Studio renders. SCN-14.1 (backwards compat with
    //    the destination tile — the route is unchanged).
    await promptsTile.click();
    await expect(page).toHaveURL(/\/settings\/prompts$/);
    // Prompt Studio's <h1> is "Prompts" (routes/settings/prompts/index.tsx:279).
    await expect(
      page.getByRole("heading", { name: "Prompts", level: 1 }),
    ).toBeVisible();
  });
});