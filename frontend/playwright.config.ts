import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright config for cachicamas end-to-end tests.
 *
 * These tests drive a real browser against the full local stack
 * (Qwik dev server + Go binary on :8080 + Postgres on :5432).
 * Unlike the unit / integration specs under `src/`, e2e tests
 * exercise the actual HTTP wire contract between frontend and
 * backend, which is exactly the surface where the recent CORS
 * regression hid.
 *
 * Pre-requisites before running `pnpm test:e2e`:
 *   1. `docker compose up` — Postgres + database_administrator
 *      must be running and healthy.
 *   2. The Go binary must be the version with the CORS middleware
 *      wired (commit `81f16fb` on `main`, or later).  Without it
 *      every POST fails in the browser with `Load failed`.
 *
 * In CI / fresh checkouts Playwright will start the Qwik dev
 * server itself via `webServer.command`.  Locally (when
 * `process.env.CI` is unset) Playwright reuses an already-running
 * `pnpm dev` to keep the dev loop fast.
 */
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
    baseURL: "http://localhost:5173",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },

  webServer: {
    command: "pnpm dev",
    url: "http://localhost:5173",
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
    stdout: "ignore",
    stderr: "pipe",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
