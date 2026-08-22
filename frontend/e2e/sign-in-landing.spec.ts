import { test, expect, type Page, type ConsoleMessage } from "@playwright/test";

/**
 * End-to-end spec for the GitHub sign-in CTA on the landing page.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-040 (S-FA-040..S-FA-046) — the landing page MUST expose a
 *   sign-in button reachable from `/` without an authenticated session.
 *
 * What this spec verifies (without driving the OAuth roundtrip):
 *   - The landing page renders a `<form>` whose hidden `providerId` is
 *     "github".
 *   - The button is reachable without any prior session cookie.
 *   - The CTA is text-first (no <img>, no decorative SVG).
 *
 * What this spec does NOT verify (covered by other specs):
 *   - The full OAuth roundtrip against `mocks-github-oauth` is in
 *     `github-sign-in.spec.ts` (requires the mocks service running).
 *   - Cookie attributes (HttpOnly, SameSite, etc.) are in
 *     `sign-in-cookie-attrs.spec.ts` (drives the full roundtrip).
 *
 * Pre-requisites:
 *   - Sin E2E_BASE_URL: docker compose up (Postgres + database_administrator
 *     + mocks-github-oauth) + pnpm dev. The frontend's AUTH_GITHUB_BASE_URL
 *     is set to http://localhost:3016 (the mocks host port).
 *   - Con E2E_BASE_URL=http://localhost:3015: the dockerized stack is up.
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

test.describe("GitHub sign-in — landing CTA", () => {
  test("renders a SignInButton form with providerId=github on /", async ({
    page,
  }) => {
    const { pageErrors, consoleErrors } = installDiagnosticListeners(page);

    await page.goto("/");
    // Allow Qwik SSR to settle (the form is part of the SSR'd HTML, but
    // client hydration also matters for the action wiring).
    await page.waitForLoadState("networkidle");

    // 1. The form is present and is the sign-in button.
    //
    // `.first()` because the public page now offers the SAME sign-in
    // affordance in three places — the header, the hero, and the closing band.
    // That is deliberate: one component, one thing to learn, one selector.
    // The first in document order is the header's.
    const signInForm = page
      .locator('form[data-testid="sign-in-button"]')
      .first();
    await expect(signInForm).toBeVisible();

    // 2. The hidden providerId is "github".
    const providerIdInput = signInForm.locator('input[name="providerId"]');
    await expect(providerIdInput).toHaveValue("github");

    // 3. The submit button carries the short label.
    //
    // Corrected from `/Sign in with GitHub/i`. The 2026-07-04 UX-4 amendment
    // shortened the label to "Sign in" and moved provider identification onto
    // the GitHub mark beside it; this assertion had been describing copy that
    // no longer shipped.
    const submitButton = signInForm.locator('button[type="submit"]');
    await expect(submitButton).toContainText(/Sign in/i);

    // 4. The hidden redirectTo sends a signed-in person to their workspace.
    //
    // Corrected from `/profile`. The public page's job is to get someone into
    // their company, not onto their account page.
    const redirectToInput = signInForm.locator('input[name="redirectTo"]');
    await expect(redirectToInput).toHaveValue("/home");

    // 5. No PHOTOGRAPHY and no unlabelled glyphs (UX-4 / R-FA-046).
    //
    // Corrected from `imgs === 0 && svgs === 0`, which had been false since
    // the same 2026-07-04 amendment added the GitHub mark. UX-4 forbids
    // meaning carried by an image ALONE, not vector artwork: every icon on
    // this page is `aria-hidden` and sits beside its own word. What UX-4 does
    // still forbid outright is decorative photography of people or places.
    const imgs = await page.locator("img").count();
    expect(imgs).toBe(0);
    const unlabelledSvgs = await page
      .locator('svg:not([aria-hidden="true"])')
      .count();
    expect(unlabelledSvgs).toBe(0);

    // 6. No console / page errors during the SSR render.
    expect(pageErrors).toHaveLength(0);
    expect(consoleErrors).toHaveLength(0);
  });

  test("the landing CTA is reachable without an authenticated session", async ({
    page,
  }) => {
    const { pageErrors, consoleErrors } = installDiagnosticListeners(page);

    // No cookies set — anonymous user.
    await page.context().clearCookies();
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const signInForm = page
      .locator('form[data-testid="sign-in-button"]')
      .first();
    await expect(signInForm).toBeVisible();

    expect(pageErrors).toHaveLength(0);
    expect(consoleErrors).toHaveLength(0);
  });
});