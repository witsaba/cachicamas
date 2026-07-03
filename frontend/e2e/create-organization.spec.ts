import { expect, test, type ConsoleMessage, type Page } from "@playwright/test";

/**
 * End-to-end regression guard for the create-organization flow.
 *
 * This spec exists because the CORS regression on 2026-07-03 took
 * 90 minutes of debugging to identify: the browser-side fetch was
 * being blocked by SOP because the Go binary wasn't returning
 * `Access-Control-Allow-Origin` headers.  None of the 95 vitest
 * specs caught it because they bypass the network layer entirely
 * (they render components in linkedom and inject the submitAction
 * QRL directly).
 *
 * The chain this test exercises:
 *
 *   browser (chromium)
 *      │
 *      │  click submit on /organizations/new
 *      ▼
 *   Qwik dev server (pnpm dev → vite --mode ssr, :5173)
 *      │
 *      │  routeAction$ QRL runs IN THE BROWSER (Qwik resumable
 *      │  model ships the action chunk to the client)
 *      ▼
 *   fetch POST http://localhost:8080/organizations
 *      │       (Origin: http://localhost:5173)
 *      │       ─── needs CORS headers on the response ───
 *      ▼
 *   database_administrator (Go bin, :8080)
 *      │
 *      │  INSERT INTO organization
 *      ▼
 *   Postgres (cachicamas-postgres container)
 *
 * Pre-requisites: `docker compose up` running, both postgres and
 * database_administrator healthy.  The Qwik dev server is started
 * automatically by Playwright's `webServer` config.
 */

const unique = (label: string) => `${label}-${Date.now()}`;

/** Surface every page error and console.error from the test browser. */
function installDiagnosticListeners(page: Page): {
  pageErrors: string[];
  consoleErrors: string[];
} {
  const pageErrors: string[] = [];
  const consoleErrors: string[] = [];
  page.on("pageerror", (err) => pageErrors.push(err.message));
  page.on("console", (msg: ConsoleMessage) => {
    if (msg.type() === "error") consoleErrors.push(msg.text());
  });
  return { pageErrors, consoleErrors };
}

test.describe("create organization end-to-end", () => {
  test("submit form persists to Postgres and navigates to detail page", async ({
    page,
  }) => {
    const { pageErrors, consoleErrors } = installDiagnosticListeners(page);

    const slug = unique("e2e-org");

    // 1. Load the create form.  This is the SSR entry point; we
    //    wait for hydration (network idle) before submitting so
    //    Qwik's lazy-loaded action chunk has time to register.
    await page.goto("/organizations/new");
    await expect(page).toHaveTitle(/Create organization/);
    await page.waitForLoadState("networkidle");

    // 2. Fill the two always-visible fields.  identification is
    //    auto-derived from fullName in the form's onInput, but we
    //    override with a unique slug so we can find this exact
    //    submission if we ever extend the spec to assert on DB.
    await page.fill('input[id="fullName"]', "E2E Probe");
    await page.fill('input[id="identification"]', slug);

    // 3. Open the optional details panel if it's collapsed, so the
    //    submit path includes the email / phone fields.  This
    //    exercises the same code path the user takes when creating
    //    an org with optional contact data.
    const showDetails = page.locator('button[data-action="show-details"]');
    if (await showDetails.isVisible().catch(() => false)) {
      await showDetails.click();
      await expect(
        page.locator('fieldset[data-review-group="true"]'),
      ).toBeVisible();
    }

    // 4. Submit.  Qwik will lazy-load the action chunk from
    //    `/src/routes/organizations/new/index.tsx_submitAction_*.js`
    //    and run it in the browser.  The action calls
    //    `createOrganization(...)` in lib/api.ts which does a
    //    cross-origin POST to :8080.
    await page.locator('button[type="submit"]').click();

    // 5. The action redirects to /organizations/<id> on success.
    //    This URL match is the canonical "the POST round-trip
    //    worked" assertion.  If CORS is broken, the fetch rejects
    //    and we stay on /organizations/new — the URL won't change.
    await expect(page).toHaveURL(/\/organizations\/\d+/, { timeout: 10_000 });

    // 6. No error banner should be visible.  The routeAction$
    //    failure path renders a top-level `role="alert"` with
    //    "Couldn't reach the backend …".  If that selector is
    //    present at all, the round-trip failed in some way.
    await expect(
      page.locator('[data-organization-error], [data-error="server"]'),
    ).toHaveCount(0);

    // 7. The detail page must show the org we just created.  The
    //    read-back route loader fetches the row from the Go binary
    //    via `getOrganization(id)`, so this also exercises the
    //    cross-origin GET path.
    await expect(page.getByText("E2E Probe").first()).toBeVisible();

    // 8. No uncaught exceptions in the page (this would catch a
    //    broken event handler, a missing import, a hydration
    //    mismatch, etc.).
    expect(pageErrors).toEqual([]);
    // console.error often fires for unrelated noise (Qwik dev
    // warnings, source-map fetches) so we soft-assert with a
    // snapshot that fails ONLY on CORS-shaped errors.  This is the
    //    single assertion that would have caught the original bug
    //    automatically: a CORS-blocked fetch always logs a console
    //    error containing "blocked by CORS policy".
    const corsErrors = consoleErrors.filter((msg) =>
      /blocked by CORS policy/i.test(msg),
    );
    expect(corsErrors, "no CORS errors expected").toEqual([]);
  });
});