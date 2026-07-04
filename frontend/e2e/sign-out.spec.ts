import { test, expect, type Page, type ConsoleMessage } from "@playwright/test";

/**
 * End-to-end spec: sign-out clears the cookie and lands on /.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-080..R-FA-086 — clicking the Sign out button on the
 *   /profile route MUST clear the authjs.session-token cookie and
 *   redirect the user back to the public landing page.
 *
 * What this spec verifies (against the mocks simulator):
 *   - The /profile Sign out button POSTs to the signOut action.
 *   - The cookie is cleared (either deleted or set to empty + past
 *     expiration).
 *   - The user lands on `/` (or on `/auth/signin` if the sign-out
 *     flow does an extra redirect through the auth callback).
 *
 * Pre-requisites (mirrors `github-sign-in.spec.ts`):
 *   - mocks-github-oauth + frontend reachable.
 *   - AUTH_GITHUB_BASE_URL set (skipped otherwise).
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

test.describe("GitHub sign-out", () => {
  test.skip(
    () => process.env.AUTH_GITHUB_BASE_URL === undefined,
    "Skipped when AUTH_GITHUB_BASE_URL is unset (production mode).",
  );

  test("signing out from /profile clears the session cookie and lands on /", async ({
    page,
    context,
  }) => {
    const { pageErrors, consoleErrors } = installDiagnosticListeners(page);
    await context.clearCookies();

    // 1. Sign in.
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    const signInForm = page.locator(
      'form[data-testid="sign-in-button"]',
    );
    await expect(signInForm).toBeVisible();
    await signInForm.locator('button[type="submit"]').click();
    await page.waitForURL(/\/profile(\?|$)/, { timeout: 15_000 });

    // Confirm cookie is set after sign-in.
    let cookies = await context.cookies();
    let session = cookies.find(
      (c) =>
        c.name === "authjs.session-token" ||
        c.name === "__Secure-authjs.session-token",
    );
    expect(session, "session cookie should be set after sign-in").toBeDefined();

    // UAT-5 (2026-07-04): the profile no longer renders its own
    // Sign out button — the avatar dropdown's Sign out entry
    // ([data-testid="avatar-menu-signout"]) is the sole sign-out
    // affordance across the app. This e2e was previously a
    // dual-path (avatar-dropdown preferred, profile button as a
    // fallback); the fallback is removed because the button
    // doesn't exist any more.
    const avatarTrigger = page.locator(
      'button[data-testid="avatar-dropdown"]',
    );
    await expect(avatarTrigger).toBeVisible();
    await avatarTrigger.click();
    const menuSignOut = page.locator(
      '[data-testid="avatar-menu-signout"]',
    );
    await expect(menuSignOut).toBeVisible();
    await menuSignOut.click();

    // 3. Wait for navigation away from /profile. Auth.js typically
    //    redirects to /auth/signout then to /; we tolerate either
    //    landing so long as we end up on a public route.
    await page.waitForURL(
      (u) => !u.toString().includes("/profile"),
      { timeout: 10_000 },
    );

    // 4. The session cookie MUST be cleared.
    cookies = await context.cookies();
    session = cookies.find(
      (c) =>
        c.name === "authjs.session-token" ||
        c.name === "__Secure-authjs.session-token",
    );
    if (session) {
      // Some browsers keep the cookie in the jar but set its value
      // to empty + past expiration. Either of these is acceptable.
      const isExpired = session.expires > 0 && session.expires * 1000 < Date.now();
      const isEmpty = (session.value ?? "").length === 0;
      expect(
        isExpired || isEmpty,
        `session cookie should be cleared; got value="${session.value}", expires=${session.expires}`,
      ).toBe(true);
    }
    // (If the cookie is fully removed, session is undefined — that's the
    // preferred outcome.)

    // 5. Navigating to /profile now should NOT show the signed-in state.
    await page.goto("/profile");
    await page.waitForLoadState("networkidle");
    const signedIn = await page
      .locator('[data-testid="profile-signed-in"]')
      .count();
    expect(signedIn, "/profile should NOT render signed-in state after sign-out").toBe(0);

    expect(pageErrors).toHaveLength(0);
    void consoleErrors;
  });
});