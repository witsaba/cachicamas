/**
 * Singleton Postgres client factory — `getSql(): Promise<SqlClient>`.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-010..R-FA-030 — `events.signIn` persists the GitHub identity to
 *   `identity.user` + `identity.account`. The Node SSR runs the wiring
 *   on the Qwik request lifecycle (per `frontend/src/routes/plugin@auth.ts`),
 *   so a single shared `postgres()` client per process is sufficient.
 *
 * Why a singleton:
 *   The `postgres` (porsager) driver manages a connection pool. Spinning
 *   a fresh client per request would defeat the pool and cost ~5ms
 *   of TCP+TLS handshake on every sign-in. A cached, lazy-initialized
 *   instance is the documented pattern.
 *
 * Why a dynamic import:
 *   The `postgres` package uses Node.js built-ins (`perf_hooks`, `crypto`,
 *   `stream`) that do not exist in the browser. Vite's client build
 *   (which produces the JS bundle for the browser) refuses to bundle
 *   these as built-ins (`"performance" is not exported by
 *   "__vite-browser-external"`). A dynamic `await import("postgres")`
 *   inside the function body defers the resolution to runtime so the
 *   client build can code-split the driver out of the browser bundle.
 *   The `getSql()` call site lives in `events.signIn` which only runs
 *   on the Node SSR side, so the dynamic import is never executed in
 *   the browser.
 *
 * Config (`IDENTITY_DATABASE_URL`):
 *   In compose this is wired from `QUEEN_USER`/`QUEEN_PASSWORD`/
 *   `POSTGRES_DB`/`POSTGRES_PORT` via the `${VAR:?msg}` interpolation
 *   in `docker-compose.yaml` (see the frontend service env block).
 *   Outside compose, an operator sets it by hand in `.env` (see
 *   `.env.example`).
 *
 * Pool sizing:
 *   Default `max: 4` matches the small dev Postgres (`postgres:18-alpine`).
 *   Production should set this explicitly if traffic warrants it.
 */
import { type SqlClient } from "./sign-in-callback";

/**
 * Cache one client per process. The `client` reference is held
 * separately so `resetSqlForTest()` can call `.end({ timeout: 1 })`
 * without losing the typed handle.
 */
let cached: { sql: SqlClient; client: { end: (opts?: unknown) => Promise<void> } } | null = null;

export async function getSql(): Promise<SqlClient> {
  if (cached) {
    return cached.sql;
  }
  const url = process.env.IDENTITY_DATABASE_URL;
  if (!url) {
    throw new Error(
      "[cachicamas] IDENTITY_DATABASE_URL is not set. The frontend cannot " +
        "persist identity rows during events.signIn. See frontend/README.md " +
        "or .env.example for the required format.",
    );
  }
  // The `if (import.meta.env.SSR)` gate is what makes this safe to
  // ship through the Qwik client build. Vite replaces the flag with
  // `false` at client-build time and dead-code-eliminates the entire
  // branch (including the dynamic import), so the `postgres` driver
  // and its Node built-ins never reach the browser bundle. At SSR
  // runtime, the flag is `true` and the import resolves normally.
  if (!import.meta.env.SSR) {
    throw new Error(
      "[cachicamas] getSql() is server-only. The events.signIn callback " +
        "must not run in the browser context.",
    );
  }
  // Dynamic import — see the file-level comment above. Vite code-splits
  // the `postgres` driver into a server-only chunk; the client bundle
  // never references it.
  const { default: postgres } = await import("postgres");
  const client = postgres(url, { max: 4, prepare: false });
  cached = {
    sql: client as unknown as SqlClient,
    client: client as unknown as { end: (opts?: unknown) => Promise<void> },
  };
  return cached.sql;
}

/**
 * Drop the cached client so the next `getSql()` call creates a fresh
 * one. Intended for test teardown — `client.end({ timeout: 1 })` lets
 * the pool drain before the test process exits.
 */
export function resetSqlForTest(): void {
  if (cached) {
    void cached.client.end({ timeout: 1 });
    cached = null;
  }
}
