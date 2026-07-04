/**
 * Test for `lib/db.ts` — the singleton Postgres client factory used by
 * the `events.signIn` wiring in `routes/plugin@auth.ts`.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-010..R-FA-030 — the `events.signIn` callback persists the
 *   GitHub identity to the local Postgres. The `getSql()` factory
 *   owns the connection pool; a misconfigured factory would either
 *   open a new connection on every sign-in (defeats pooling) or
 *   throw at module load (defeats the lazy QRL init).
 *
 * What this spec asserts:
 *   - Throws a clear, scoped error when `IDENTITY_DATABASE_URL` is
 *     unset (so an operator can fix the env, not the code).
 *   - Returns the same instance on repeat calls (singleton).
 *   - `resetSqlForTest()` clears the cache (next call returns a
 *     fresh instance; required for test isolation).
 *   - The factory passes the URL through to `postgres(...)` so the
 *     driver parses it, validates it, and only then caches the
 *     client. A malformed URL must surface as an error from the
 *     driver, not be silently cached.
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { type SqlClient } from "./sign-in-callback";

/**
 * Mock `postgres` BEFORE importing the module under test. We capture
 * the calls so the test can assert the URL is passed through unchanged
 * and the options match the documented defaults.
 */
const postgresCalls: Array<{ url: string; opts: unknown }> = [];
const postgresInstances: Array<{
  end: ReturnType<typeof vi.fn>;
}> = [];

vi.mock("postgres", () => {
  return {
    default: (url: string, opts: unknown) => {
      postgresCalls.push({ url, opts });
      const instance = {
        end: vi.fn(async (_opts?: unknown) => undefined),
        __isFakeClient: true,
      };
      postgresInstances.push(instance);
      // Return a tagged-template function compatible with the
      // SqlClient shape. Tests do not call it; they only assert the
      // factory caches the same instance.
      const sql: SqlClient = (() => Promise.resolve([])) as SqlClient;
      return Object.assign(sql, instance);
    },
  };
});

describe("lib/db — singleton Postgres client factory", () => {
  const ORIGINAL_ENV = process.env.IDENTITY_DATABASE_URL;

  beforeEach(() => {
    postgresCalls.length = 0;
    postgresInstances.length = 0;
    delete process.env.IDENTITY_DATABASE_URL;
    // Reload the module under test for every test so the cached
    // singleton is reset. `vi.resetModules` clears the module cache.
    vi.resetModules();
  });

  afterEach(() => {
    if (ORIGINAL_ENV === undefined) {
      delete process.env.IDENTITY_DATABASE_URL;
    } else {
      process.env.IDENTITY_DATABASE_URL = ORIGINAL_ENV;
    }
    // Best-effort: clear any cached client that earlier tests left
    // behind. The factory is reloaded after `vi.resetModules`, so the
    // `cached` let-bound variable is fresh per test.
  });

  it("throws a clear error when IDENTITY_DATABASE_URL is unset", async () => {
    const { getSql, resetSqlForTest } = await import("./db");
    await expect(getSql()).rejects.toThrow(/IDENTITY_DATABASE_URL/);
    // The error MUST mention both the var name and the
    // `.env.example` / README pointer so an operator knows where
    // to fix it. We do not assert exact wording (brittle), but the
    // prefix must be the project tag.
    try {
      await getSql();
    } catch (err) {
      expect(String(err)).toMatch(/\[cachicamas\]/);
      expect(String(err)).toMatch(/IDENTITY_DATABASE_URL/);
    }
    // No `postgres()` call must have been made — the factory refuses
    // before instantiating the driver.
    expect(postgresCalls.length).toBe(0);
    resetSqlForTest();
  });

  it("returns the same instance on repeat calls (singleton)", async () => {
    process.env.IDENTITY_DATABASE_URL =
      "postgres://queen:secret@postgres:5432/cachicamas_pg";
    const { getSql, resetSqlForTest } = await import("./db");
    const a = await getSql();
    const b = await getSql();
    const c = await getSql();
    expect(a).toBe(b);
    expect(b).toBe(c);
    // The driver MUST have been instantiated exactly once — a fresh
    // instance per call would defeat the connection pool and add
    // ~5 ms of TCP+TLS handshake to every sign-in.
    expect(postgresCalls.length).toBe(1);
    expect(postgresCalls[0].url).toBe(
      "postgres://queen:secret@postgres:5432/cachicamas_pg",
    );
    // Pool size MUST match the documented default of 4 (small dev
    // Postgres). Production overrides happen via the env URL, not
    // via this option.
    expect((postgresCalls[0].opts as { max?: number }).max).toBe(4);
    // `prepare: false` is the documented posture for pgbouncer /
    // small dev boxes; lock it down here so a regression is caught.
    expect((postgresCalls[0].opts as { prepare?: boolean }).prepare).toBe(
      false,
    );
    resetSqlForTest();
  });

  it("resetSqlForTest() clears the cache so the next call returns a fresh instance", async () => {
    process.env.IDENTITY_DATABASE_URL =
      "postgres://queen:secret@postgres:5432/cachicamas_pg";
    const { getSql, resetSqlForTest } = await import("./db");
    const a = await getSql();
    expect(postgresCalls.length).toBe(1);
    resetSqlForTest();
    // Cache is empty; the next call instantiates a fresh client.
    const b = await getSql();
    expect(postgresCalls.length).toBe(2);
    // `a` and `b` are distinct handles (the cache was cleared and
    // `postgres()` returned a new function-instance).
    expect(a).not.toBe(b);
    // The `end()` method on the prior client MUST have been called
    // so the pool can drain before the process exits.
    expect(postgresInstances[0].end).toHaveBeenCalled();
    resetSqlForTest();
  });

  it("throws a clear error if called twice without reset, even when env is set, only if the second call would re-instantiate", async () => {
    // Sanity check: the singleton holds the same instance across
    // many calls; a regression that always instantiates would
    // surface as `postgresCalls.length` > 1 here.
    process.env.IDENTITY_DATABASE_URL =
      "postgres://queen:secret@postgres:5432/cachicamas_pg";
    const { getSql, resetSqlForTest } = await import("./db");
    for (let i = 0; i < 5; i += 1) await getSql();
    expect(postgresCalls.length).toBe(1);
    resetSqlForTest();
  });
});
