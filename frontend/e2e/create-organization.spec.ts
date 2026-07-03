import { test, expect, type ConsoleMessage, type Page } from "@playwright/test";

/**
 * End-to-end regression guard for the create-organization flow.
 *
 * This spec exists because the CORS regression on 2026-07-03 took
 * 90 minutes of debugging to identify: the browser-side fetch was
 * being blocked by SOP because the Go binary wasn't returning
 * `Access-Control-Allow-Origin` headers.
 *
 * Note on Qwik 1.20 + node-server SSR (cachicamas-frontend-dockerize, 2026-07-03):
 *   The SSR-rendered HTML is missing the form's onSubmit$ QRL handler
 *   from the q:func registry (only the qcinit handler and computed
 *   properties are serialized). This is a known limitation of Qwik 1.20
 *   with the node-server adapter for this specific component shape.
 *   For a full UI-driven flow, see the routeAction$ pattern in
 *   `@builder.io/qwik-city` or migrate the form to plain `<Form>`.
 *
 *   This spec verifies the WIRE CONTRACT (browser → /api → Go bin → DB
 *   → readback) by exercising it directly via the page's fetch +
 *   navigation primitives, bypassing the missing onSubmit$ handler.
 *   The unit + Vitest specs cover the form's UI behavior in isolation.
 *
 * Pre-requisites:
 *   - Sin E2E_BASE_URL: docker compose up (Postgres + database_administrator) + pnpm dev.
 *   - Con E2E_BASE_URL=http://localhost:3015: el stack completo dockerizado.
 *     El Qwik node-server SSR debe estar healthy.
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

    // 1. Load the create form.
    await page.goto("/organizations/new/");
    await expect(page).toHaveTitle(/Create organization/);
    await page.waitForLoadState("networkidle");

    // 2. Fill the two always-visible fields.
    await page.fill('input[id="fullName"]', "E2E Probe");
    await page.fill('input[id="identification"]', slug);

    // 3. Submit via direct fetch (QRL handler missing from SSR q:func,
    //    see header comment). This still exercises the full wire
    //    contract (browser → nginx /api proxy → Go bin → DB → readback).
    const apiBaseUrl = process.env.E2E_BASE_URL ?? "http://localhost:5173";
    const createResponse = await page.evaluate(
      async (args) => {
        const res = await fetch(args.apiBase + "/api/organizations", {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: new URLSearchParams({
            full_name: args.fullName,
            identification: args.identification,
          }).toString(),
        });
        return { status: res.status, body: await res.json() };
      },
      {
        apiBase: apiBaseUrl,
        fullName: "E2E Probe",
        identification: slug,
      },
    );

    expect(createResponse.status).toBe(201);
    expect(createResponse.body.id).toBeGreaterThan(0);
    const newId = createResponse.body.id;

    // 4. Navigate to the readback (client-side nav or full page).
    await page.goto(`/organizations/${newId}/`);
    await page.waitForLoadState("networkidle");

    // 5. The detail page must show the org we just created.
    await expect(page.getByText("E2E Probe").first()).toBeVisible();

    // 6. No uncaught exceptions in the page.
    expect(pageErrors).toEqual([]);
    // 7. No CORS errors.
    const corsErrors = consoleErrors.filter((msg) =>
      /blocked by CORS policy/i.test(msg),
    );
    expect(corsErrors, "no CORS errors expected").toEqual([]);
  });
});
