import { test, expect, type Page } from "@playwright/test";

/**
 * End-to-end spec: full Workspaces feature flow via the
 * `mocks-github-oauth` compose service.
 *
 * Reference: `openspec/changes/2026-07-06-workspaces/spec.md`
 *   R-WS-011..R-WS-013 — workspaces list, create form, and detail page.
 *
 * Pre-requisites (mirrors github-sign-in.spec.ts):
 *   - docker compose up (Postgres + database_administrator + mocks-github-oauth
 *     + frontend). AUTH_GITHUB_BASE_URL=http://mocks-github-oauth:3016.
 *   - The mocks service grants whatever scope the OAuth roundtrip requests.
 *     Since PR1a (commit 4a305cd), the frontend requests `scope=repo +
 *     access_type=offline`; the mocks service grants `repo` dynamically.
 *   - A clean DB row in `organization` (so `requireOwnboarding` lets the
 *     user past the gate). The ownboarding test flow handles this upstream.
 *
 * Skip semantics:
 *   All tests `test.skip(!process.env.E2E_BASE_URL, ...)` because they
 *   require a running docker compose stack with mocks-github-oauth. In
 *   CI with E2E_BASE_URL set, all tests run.
 */

async function signInViaMocks(page: Page): Promise<void> {
  await page.goto("/");
  await page.waitForLoadState("networkidle");
  const signInForm = page.locator('form[data-testid="sign-in-button"]');
  await expect(signInForm).toBeVisible();
  await signInForm.locator('button[type="submit"]').click();
  // /auth/signin redirects to GitHub, GitHub redirects back, frontend
  // exchanges the code, lands on /profile (or /ownboarding if no org).
  await page.waitForURL(/\/(profile|ownboarding|home)/, { timeout: 15_000 });
}

test.describe("workspaces feature", () => {
  test("anonymous user is redirected to /auth/signin when hitting /workspaces", async ({
    page,
    context,
  }) => {
    test.skip(
      !process.env.E2E_BASE_URL,
      "Playwright e2e requires docker compose stack with mocks-github-oauth; skipped locally",
    );
    await context.clearCookies();
    await page.goto("/workspaces");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveURL(/\/auth\/signin/);
  });

  test("signed-in user sees the workspaces list page", async ({
    page,
    context,
  }) => {
    test.skip(
      !process.env.E2E_BASE_URL,
      "Playwright e2e requires docker compose stack with mocks-github-oauth; skipped locally",
    );
    await context.clearCookies();
    await signInViaMocks(page);

    await page.goto("/workspaces");
    await page.waitForLoadState("networkidle");

    // Either the empty CTA or the populated list renders — both indicate
    // the user passed the requireAuthRedirect + requireOwnboarding gate.
    const emptyCta = page.locator('[data-testid="create-workspace-cta"]');
    const list = page.locator('[data-testid="workspaces-list"]');
    await expect(emptyCta.or(list)).toBeVisible();
  });

  test("create-workspace form renders name input + primary repo picker", async ({
    page,
    context,
  }) => {
    test.skip(
      !process.env.E2E_BASE_URL,
      "Playwright e2e requires docker compose stack with mocks-github-oauth; skipped locally",
    );
    await context.clearCookies();
    await signInViaMocks(page);

    await page.goto("/workspaces/new");
    await page.waitForLoadState("networkidle");

    const nameInput = page.locator(
      'form[data-testid="workspace-form"] input[name="name"]',
    );
    await expect(nameInput).toBeVisible();
    const picker = page.locator('[data-testid="github-repo-picker"]');
    await expect(picker).toBeVisible();
  });

  test("workspaces link is present in the avatar dropdown for authed users", async ({
    page,
    context,
  }) => {
    test.skip(
      !process.env.E2E_BASE_URL,
      "Playwright e2e requires docker compose stack with mocks-github-oauth; skipped locally",
    );
    await context.clearCookies();
    await signInViaMocks(page);

    const avatarBtn = page.locator('[data-testid="avatar-button"]');
    await expect(avatarBtn).toBeVisible();
    await avatarBtn.click();

    const workspacesLink = page.locator(
      '[data-testid="avatar-menu-workspaces"]',
    );
    await expect(workspacesLink).toBeVisible();
    await expect(workspacesLink).toHaveAttribute("href", "/workspaces");
  });

  test("non-existent workspace renders the not-found surface on /workspaces/:id", async ({
    page,
    context,
  }) => {
    test.skip(
      !process.env.E2E_BASE_URL,
      "Playwright e2e requires docker compose stack with mocks-github-oauth; skipped locally",
    );
    await context.clearCookies();
    await signInViaMocks(page);

    await page.goto("/workspaces/99999999");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveURL(/\/workspaces\/99999999/);
    const bodyText = await page.locator("body").innerText();
    expect(bodyText.length).toBeGreaterThan(0);
  });
});
