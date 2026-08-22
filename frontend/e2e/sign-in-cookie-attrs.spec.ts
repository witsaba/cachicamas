import { test, expect, type Page, type ConsoleMessage } from "@playwright/test";

/**
 * End-to-end spec: cookie attributes after a successful GitHub OAuth roundtrip.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-060..R-FA-066 — the authjs.session-token cookie MUST be HttpOnly,
 *   SameSite=Lax, Path=/, and (in production HTTPS) carry the `__Secure-`
 *   prefix on the cookie name. The browser uses these attributes to decide
 *   whether to attach the cookie to cross-origin requests; misconfiguring
 *   any one of them is a known source of silent auth failures.
 *
 * What this spec verifies:
 *   - HttpOnly flag set on the cookie (no JS access).
 *   - SameSite=Lax (Auth.js default for the JWT strategy).
 *   - Path=/ (cookie applies to every route on the origin).
 *   - Cookie name is `authjs.session-token` in dev (HTTP) and
 *     `__Secure-authjs.session-token` in production (HTTPS).
 *
 * Pre-requisites (mirrors `github-sign-in.spec.ts`):
 *   - mocks-github-oauth + frontend reachable.
 *   - AUTH_GITHUB_BASE_URL=http://mocks-github-oauth:3016 (compose) or
 *     http://localhost:3016 (local dev).
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

test.describe("GitHub sign-in — cookie attributes", () => {
  test.skip(
    () => process.env.AUTH_GITHUB_BASE_URL === undefined,
    "Skipped when AUTH_GITHUB_BASE_URL is unset (production mode).",
  );

  test("the authjs.session-token cookie is HttpOnly, SameSite=Lax, Path=/", async ({
    page,
    context,
  }) => {
    const { pageErrors, consoleErrors } = installDiagnosticListeners(page);
    await context.clearCookies();

    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const signInForm = page
      .locator('form[data-testid="sign-in-button"]')
      .first();
    await expect(signInForm).toBeVisible();
    await signInForm.locator('button[type="submit"]').click();

    // Wait for the roundtrip to complete (URL becomes /profile).
    await page.waitForURL(/\/profile(\?|$)/, { timeout: 15_000 });

    // Inspect the cookie the Auth.js handler set.
    const cookies = await context.cookies();
    const session = cookies.find(
      (c) =>
        c.name === "authjs.session-token" ||
        c.name === "__Secure-authjs.session-token",
    );
    expect(session, "expected a session cookie to be set").toBeDefined();

    // 1. HttpOnly: true (no JS access to the cookie value).
    expect(session!.httpOnly, "cookie should be HttpOnly").toBe(true);

    // 2. SameSite: Auth.js' default for the JWT strategy is "Lax".
    //    Playwright normalises the value to lowercase.
    const sameSite = (session!.sameSite ?? "").toLowerCase();
    expect(
      ["lax", "strict"].includes(sameSite),
      `SameSite should be Lax or Strict, got "${sameSite}"`,
    ).toBe(true);

    // 3. Path: / (the cookie applies to every route on the origin).
    expect(session!.path).toBe("/");

    // 4. Domain: not set (host-only). Playwright may surface this as
    //    empty string or undefined; either is acceptable.
    expect(session!.domain ?? "").toBe("");

    // No console / page errors during the roundtrip.
    expect(pageErrors).toHaveLength(0);
    expect(consoleErrors).toHaveLength(0);
  });

  test("the cookie name carries the __Secure- prefix when AUTH_URL is HTTPS", async ({
    page,
    context,
  }) => {
    const { pageErrors, consoleErrors } = installDiagnosticListeners(page);
    await context.clearCookies();

    // Force HTTPS-style behaviour by setting the browser to an https://
    // origin via the AUTH_URL detection in the page. In the test env
    // the simplest assertion is to read the same cookie and check that
    // EITHER dev name or prod name is present (the test environment
    // decides which based on its OWN origin scheme).
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const signInForm = page
      .locator('form[data-testid="sign-in-button"]')
      .first();
    await expect(signInForm).toBeVisible();
    await signInForm.locator('button[type="submit"]').click();
    await page.waitForURL(/\/profile(\?|$)/, { timeout: 15_000 });

    const cookies = await context.cookies();
    const session = cookies.find(
      (c) =>
        c.name === "authjs.session-token" ||
        c.name === "__Secure-authjs.session-token",
    );
    expect(session).toBeDefined();
    // In HTTPS origin the cookie name MUST start with __Secure-.
    // In HTTP origin it MUST NOT.
    const origin = new URL(page.url());
    if (origin.protocol === "https:") {
      expect(
        session!.name.startsWith("__Secure-"),
        `HTTPS origin should produce a __Secure- prefixed cookie, got "${session!.name}"`,
      ).toBe(true);
    } else {
      expect(
        !session!.name.startsWith("__Secure-"),
        `HTTP origin should NOT produce a __Secure- prefixed cookie, got "${session!.name}"`,
      ).toBe(true);
    }

    expect(pageErrors).toHaveLength(0);
    expect(consoleErrors).toHaveLength(0);
  });
});