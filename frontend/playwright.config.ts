import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright config for cachicamas end-to-end tests.
 *
 * Dual mode via `E2E_BASE_URL`:
 *   - Sin E2E_BASE_URL: comportamiento legacy. Levanta `pnpm dev`
 *     en 5173 y corre contra el dev server. Útil para dev local
 *     sin compose.
 *   - Con E2E_BASE_URL=http://localhost:3015 (o el dominio del VPS):
 *     NO levanta `webServer` — apunta a la URL provista. Útil para
 *     CI, post-deploy, y validación de la imagen dockerizada.
 *
 * Pre-requisites antes de `pnpm test:e2e`:
 *   - Sin E2E_BASE_URL: docker compose up (Postgres + database_administrator).
 *     Playwright levanta `pnpm dev` y todo el flujo corre local.
 *   - Con E2E_BASE_URL: el stack completo debe estar corriendo en la URL
 *     apuntada. Playwright NO inicia nada — confía en que el stack existe.
 *
 * The Go binary must have the CORS middleware wired (commit `81f16fb`
 * on `main`, or later).  Without it every POST fails in the browser
 * with `Load failed`.
 */
const e2eBaseUrl = process.env.E2E_BASE_URL ?? "http://localhost:5173";
const useDevServer = !process.env.E2E_BASE_URL;

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 5_000 },
  // One e2e at a time keeps DB writes serial (each test uses a
  // unique slug so it would be safe in parallel, but the run
  // noise stays manageable).
  fullyParallel: false,
  workers: 1,
  retries: 0,

  reporter: [["list"]],

  use: {
    baseURL: e2eBaseUrl,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },

  webServer: useDevServer
    ? {
        command: "pnpm dev",
        url: "http://localhost:5173",
        reuseExistingServer: !process.env.CI,
        timeout: 60_000,
        stdout: "ignore",
        stderr: "pipe",
      }
    : undefined,

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
