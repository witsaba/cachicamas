import { test, expect, type Page, type ConsoleMessage } from "@playwright/test";

/**
 * End-to-end spec: full GitHub OAuth roundtrip via the
 * `mocks-github-oauth` compose service.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-001 + R-FA-010..R-FA-030 — the OAuth roundtrip succeeds end-to-end;
 *   identity.user + identity.account rows land in Postgres; the session
 *   cookie is set; the user lands on /profile.
 *
 * Pre-requisites:
 *   - The full stack is up: postgres + database_administrator +
 *     mocks-github-oauth + frontend (docker compose up). The frontend's
 *     AUTH_GITHUB_BASE_URL is set to http://mocks-github-oauth:3016
 *     (compose network DNS).
 *   - For local dev without compose: pnpm dev + docker compose up for
 *     the mocks service, and a .env / shell override setting
 *     AUTH_GITHUB_BASE_URL=http://localhost:3016.
 *
 * Why this spec only runs when AUTH_GITHUB_BASE_URL points at mocks:
 *   When the env is unset (production), the spec would attempt a real
 *   GitHub OAuth roundtrip and fail with "wrong client_id". We detect
 *   the mock mode by reading AUTH_GITHUB_BASE_URL from the page (the
 *   server injects it into the SSR'd HTML via the page meta). If the
 *   env is unset, we skip with a friendly message.
 */
function installDiagnosticListeners(page: Page) {
  const pageErrors: string[] = [];
  const consoleErrors: string[] = [];
  page.on("pageerror", (err) => pageErrors.push(err.message));
  page.on("console", (msg: ConsoleMessage) => {
    if (msg.type() === "error") consoleErrors.push(msg.text());
  });
  return { pageErrors, consoleErrors };
}

test.describe("GitHub OAuth roundtrip (mocks mode)", () => {
  test.skip(
    ({ }) => process.env.AUTH_GITHUB_BASE_URL === undefined,
    "Skipped when AUTH_GITHUB_BASE_URL is unset (production mode).",
  );

  test("anonymous → click Sign in → /profile shows the test user's name", async ({
    page,
    context,
  }) => {
    const { pageErrors, consoleErrors } = installDiagnosticListeners(page);
    // Always start anonymous.
    await context.clearCookies();

    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const signInForm = page.locator(
      'form[data-testid="sign-in-button"]',
    );
    await expect(signInForm).toBeVisible();

    // Click submit. The Qwik Form intercepts the submit and POSTs to
    // the Auth.js signin action, which then 302s the browser to the
    // GitHub authorize URL — but in mocks mode that URL points at
    // mocks-github-oauth:3016, which short-circuits the consent and
    // redirects back to the callback URL with ?code=test_code. The
    // Qwik server then exchanges the code for a token, signs the JWE
    // cookie, and redirects to /profile.
    await signInForm.locator('button[type="submit"]').click();

// Wait for the URL to become /profile (or /auth/signin if denied).
        await page.waitForURL(/\/profile(\?|$)/, { timeout: 15_000 });

        // UAT-9 (2026-07-04): /profile renders a personalized welcome
        // heading instead of a standalone name <h1>. The mocks user's
        // name is "Octocat" (single token) — firstName() returns it
        // as-is, so the heading reads "Welcome back, Octocat!".
        // The retired data-testid="profile-name" / "profile-email" /
        // "profile-image" no longer exist on /profile — the shell's
        // AvatarDropdown panel header is the sole identity surface.
        const welcomeHeading = page.locator(
          '[data-testid="profile-welcome"]',
        );
        await expect(welcomeHeading).toBeVisible();
        await expect(welcomeHeading).toHaveText("Welcome back, Octocat!");

        // PR-4 (cachicamas-login-ux, T4.4.1): the github_login
        // anchor MUST render with the mocks user's login
        // (TEST_USER.login = "octocat" in scripts/mocks-github-oauth
        // /server.mjs). The href is hard-coded to
        // https://github.com/{login} per R-PH-003.
        const githubLoginLink = page.locator(
          'a[data-testid="profile-github-login"]',
        );
        await expect(githubLoginLink).toBeVisible();
        await expect(githubLoginLink).toHaveAttribute(
          "href",
          "https://github.com/octocat",
        );
        await expect(githubLoginLink).toHaveAttribute("target", "_blank");
        await expect(githubLoginLink).toHaveAttribute(
          "rel",
          "noopener noreferrer",
        );

// 2026-07-06 ownboarding (R-OW-010 / S-OW-094): the
        // manage-orgs link was removed from /profile when the
        // /organizations surface was deleted. The dashboard
        // header now shows only the avatar (with Profile + Sign
        // out in the dropdown).
        const manageOrgsLink = page.locator(
          'a[data-testid="profile-manage-orgs"]',
        );
        await expect(manageOrgsLink).toHaveCount(0);

        // The authjs.session-token cookie should be set with HttpOnly.
    const cookies = await context.cookies();
    const session = cookies.find(
      (c) => c.name === "authjs.session-token" || c.name === "__Secure-authjs.session-token",
    );
    expect(session).toBeDefined();
    expect(session?.httpOnly).toBe(true);

    // No console / page errors during the roundtrip.
    expect(pageErrors).toHaveLength(0);
    expect(consoleErrors).toHaveLength(0);
  });
});