/**
 * Test for `routes/plugin@auth.ts` — the canonical Auth.js for Qwik wiring.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-001 (S-FA-001..S-FA-005) — the Qwik app exposes the GitHub OAuth
 *   roundtrip routes via the `onRequest` middleware, with the four canonical
 *   Auth.js exports (`onRequest`, `useSession`, `useSignIn`, `useSignOut`).
 *
 * Scope of this spec (unit-test boundary):
 *   - The module under test exports the four members with the expected types.
 *   - The `QwikAuth$` factory is invoked with a callback whose return value
 *     includes a `providers` array containing `GitHub` (i.e., the GitHub OAuth
 *     provider is the only provider).
 *   - `trustHost: true` is set on the config (required for non-Vercel/Netlify
 *     deploys; see design §5.1 and ADR 0001).
 *
 * What this spec does NOT assert (covered by other layers):
 *   - Real OAuth roundtrip (Playwright e2e: `e2e/github-sign-in.spec.ts`).
 *   - Cookie attributes (Playwright e2e: `e2e/sign-in-cookie-attrs.spec.ts`).
 *   - Persistence of identity.user + identity.account (Go integration test).
 */
import { describe, it, expect, vi } from "vitest";

/**
 * Mock `@auth/qwik` BEFORE importing the module under test, so we can
 * capture the arguments passed to `QwikAuth$` and assert them.
 *
 * Why we export BOTH `QwikAuth$` and `QwikAuthQrl`:
 *   Qwik's optimizer (`@builder.io/qwik/optimizer` in vite.config.ts)
 *   rewrites every `$`-suffixed identifier at compile time. The compiler
 *   replaces `QwikAuth$` with `QwikAuthQrl` in the BUNDLED plugin@auth.ts,
 *   and the bundle imports `QwikAuthQrl` from `@auth/qwik`. Without
 *   exporting `QwikAuthQrl` from the mock, the module-load throws:
 *     "[vitest] No QwikAuthQrl export is defined on the @auth/qwik mock".
 *   Both names map to the same factory at runtime (see
 *   `node_modules/@auth/qwik/index.qwik.js`:
 *     `const QwikAuth$ = implicit$FirstArg(QwikAuthQrl);`).
 */
const qwikAuthCalls: Array<{
  qrl: unknown;
  /** Eval the lazy callback with a fake env to capture the resolved config. */
  resolveConfig: (env: Record<string, string>) => unknown;
}> = [];

function makeAuthFactory() {
  return (qrl: unknown) => {
    qwikAuthCalls.push({
      qrl,
      resolveConfig: async (env: Record<string, string>) => {
        // Re-create a RequestEventCommon-shaped stub so the qrl can read
        // `env.get("AUTH_GITHUB_ID")` etc.
        const ev = {
          env: {
            get: (key: string) => env[key],
            getAll: () => env,
          },
          request: new Request("http://localhost/"),
          url: new URL("http://localhost/"),
          params: {},
          cookie: { get: () => undefined, set: () => undefined },
        };
        // The qrl is wrapped by Qwik's optimizer into an async function
        // (QRL). It is callable as `qrl(ev)` and returns a Promise of
        // whatever the original callback returned.
        const result = await (qrl as (e: typeof ev) => Promise<unknown>)(ev);
        return result;
      },
    });
    // The real QwikAuth$ returns the four canonical exports. We mirror
    // the same shape so the bundled plugin@auth.ts can destructure.
    return {
      onRequest: () => Promise.resolve(),
      useSession: () => ({ value: null }),
      useSignIn: () => ({}),
      useSignOut: () => ({}),
    };
  };
}

vi.mock("@auth/qwik", () => {
  const factory = makeAuthFactory();
  return {
    QwikAuth$: factory,
    QwikAuthQrl: factory,
  };
});

/**
 * Capture the `GitHub` provider reference. The mock returns a sentinel
 * object that the implementation is expected to include in its
 * `providers: [...]` array. We assert identity — i.e., the SAME object —
 * to prove that the import is wired and that `GitHub(...)` was called
 * with the env bindings.
 *
 * The Qwik optimizer only touches `$`-suffixed symbols, so `GitHub` is
 * imported as-is and the default-export mock is sufficient.
 */
const githubFactoryCalls: Array<Record<string, unknown>> = [];

    vi.mock("@auth/qwik/providers/github", () => {
      return {
        default: (config: Record<string, unknown>) => {
          githubFactoryCalls.push(config);
          return { __sentinel: "github-provider", ...config };
        },
      };
    });
    
    /**
     * Mock the `events.signIn` collaborators so the wiring test can
     * assert that the `events.signIn` field on the Auth.js config
     * actually delegates to `handleSignIn(getSql(), event)`.
     *
     * These mocks are module-scope (hoisted by Vitest) so they apply
     * to all tests in this file. The existing five tests do not
     * exercise the `events.signIn` code path (they only resolve the
     * config object, never trigger a sign-in), so the mocks are
     * observationally transparent to them.
     */
    const handleSignInMock = vi.fn().mockResolvedValue(true);
    const getSqlMock = vi.fn(() => ({ __stubSqlClient: "identity-stub" }));
    handleSignInMock.mockName("handleSignIn");
    getSqlMock.mockName("getSql");
    
    vi.mock("~/lib/sign-in-callback", () => ({
      handleSignIn: handleSignInMock,
    }));
    
    vi.mock("~/lib/db", () => ({
      getSql: getSqlMock,
    }));
    
    describe("routes/plugin@auth — Auth.js for Qwik wiring", () => {
  it("module is loaded and QwikAuth$ was invoked exactly once", async () => {
    // Reset capture state so other tests in this file do not pollute
    // the count when modules are re-evaluated by Vite.
    qwikAuthCalls.length = 0;
    githubFactoryCalls.length = 0;
    await import("./plugin@auth");
    expect(qwikAuthCalls.length).toBe(1);
  });

  it("exports the four canonical Auth.js members", async () => {
    const mod = await import("./plugin@auth");
    // The four exports documented in @auth/qwik's `index.d.ts` and in
    // openspec/.../specs/frontend-auth/spec.md R-FA-001.
    expect(typeof mod.onRequest).toBe("function");
    expect(mod.useSession).toBeDefined();
    expect(mod.useSignIn).toBeDefined();
    expect(mod.useSignOut).toBeDefined();
  });

  it("QwikAuth$ config includes the GitHub provider as the only provider", async () => {
    const call = qwikAuthCalls[0];
    expect(call).toBeDefined();
    const resolved = (await call.resolveConfig({
      AUTH_GITHUB_ID: "test-id",
      AUTH_GITHUB_SECRET: "test-secret",
      AUTH_SECRET: "test-secret-base64",
      AUTH_TRUST_HOST: "true",
      AUTH_URL: "http://localhost:3015",
    })) as {
      providers: Array<{ __sentinel?: string }>;
      trustHost?: boolean;
    };
    expect(resolved.providers).toHaveLength(1);
    expect(resolved.providers[0].__sentinel).toBe("github-provider");
  });

  it("QwikAuth$ config sets trustHost: true for non-Vercel deploys", async () => {
    const call = qwikAuthCalls[0];
    const resolved = (await call.resolveConfig({
      AUTH_GITHUB_ID: "test-id",
      AUTH_GITHUB_SECRET: "test-secret",
      AUTH_SECRET: "test-secret-base64",
      AUTH_TRUST_HOST: "true",
      AUTH_URL: "http://localhost:3015",
    })) as { trustHost?: boolean };
    expect(resolved.trustHost).toBe(true);
  });

      it("GitHub provider is configured with clientId/clientSecret pulled from env", async () => {
        const last = githubFactoryCalls[githubFactoryCalls.length - 1];
        expect(last).toBeDefined();
        expect(last.clientId).toBe("test-id");
        expect(last.clientSecret).toBe("test-secret");
      });

      // ---------------------------------------------------------------------
      // cachicamas-github-login events.signIn wiring (R-FA-010..R-FA-030).
      //
      // The shipped `sign-in-callback.ts` defines the auto-link-on-email-match
      // UPSERT logic with 6 passing unit tests, but the Auth.js config in
      // `plugin@auth.ts` never wired it as `events.signIn` — so the
      // `identity.user` / `identity.account` tables stayed empty in
      // production sign-ins. This spec closes that gap.
      // ---------------------------------------------------------------------

      it("QwikAuth$ config exposes an events.signIn field wired to handleSignIn (R-FA-010..R-FA-030)", async () => {
        // The plugin@auth module is already loaded by an earlier test in
        // this describe; the captured qwikAuthCalls[0] still resolves
        // against the mocked @auth/qwik factory.
        const call = qwikAuthCalls[0];
        expect(call).toBeDefined();
        const resolved = (await call.resolveConfig({
          AUTH_GITHUB_ID: "test-id",
          AUTH_GITHUB_SECRET: "test-secret",
          AUTH_SECRET: "test-secret-base64",
          AUTH_TRUST_HOST: "true",
          AUTH_URL: "http://localhost:3015",
        })) as {
          events?: {
            signIn?: (event: {
              user: { email?: string | null };
              account?: { provider: string; providerAccountId: string } | null;
            }) => Promise<void> | void;
          };
        };
        // RED gate: the field MUST exist and be a function. If
        // `plugin@auth.ts` does not wire it, this assertion fails.
        expect(resolved.events).toBeDefined();
        expect(typeof resolved.events?.signIn).toBe("function");
      });

      it("events.signIn delegates to handleSignIn(getSql(), event)", async () => {
        const call = qwikAuthCalls[0];
        const resolved = (await call.resolveConfig({
          AUTH_GITHUB_ID: "test-id",
          AUTH_GITHUB_SECRET: "test-secret",
          AUTH_SECRET: "test-secret-base64",
          AUTH_TRUST_HOST: "true",
          AUTH_URL: "http://localhost:3015",
        })) as {
          events: {
            signIn: (event: {
              user: { email?: string | null };
              account?: { provider: string; providerAccountId: string } | null;
            }) => Promise<void> | void;
          };
        };
        // Reset mock call history so we only see the invocation from
        // THIS test, not the residual calls from any earlier
        // resolveConfig that triggered lazy import-time setup.
        handleSignInMock.mockClear();
        getSqlMock.mockClear();
        // Re-stub handleSignIn to resolve cleanly for this test.
        handleSignInMock.mockResolvedValueOnce(true);

        const signInEvent = {
          user: { email: "x@y.z", name: "X", image: null },
          account: { provider: "github", providerAccountId: "1" },
        };
        await resolved.events.signIn(signInEvent);
        // The wiring MUST call getSql() to obtain the SQL client.
        expect(getSqlMock).toHaveBeenCalledTimes(1);
        // The SQL instance returned by getSql() MUST be the first
        // argument passed to handleSignIn (this is the triangulation
        // gate — proves wiring end-to-end, not just the call sites).
        expect(handleSignInMock).toHaveBeenCalledTimes(1);
        const firstCall = handleSignInMock.mock.calls[0];
        expect(firstCall[0]).toEqual({ __stubSqlClient: "identity-stub" });
        expect(firstCall[1]).toEqual(signInEvent);
      });

      it("events.signIn is best-effort: resolves even if handleSignIn throws (log + swallow)", async () => {
        const call = qwikAuthCalls[0];
        const resolved = (await call.resolveConfig({
          AUTH_GITHUB_ID: "test-id",
          AUTH_GITHUB_SECRET: "test-secret",
          AUTH_SECRET: "test-secret-base64",
          AUTH_TRUST_HOST: "true",
          AUTH_URL: "http://localhost:3015",
        })) as {
          events: {
            signIn: (event: {
              user: { email?: string | null };
              account?: { provider: string; providerAccountId: string } | null;
            }) => Promise<void> | void;
          };
        };
        // Simulate a transient SQL error: handleSignIn rejects.
        handleSignInMock.mockRejectedValueOnce(new Error("network glitch"));
        // The wiring MUST log the error (console.error) and still
        // resolve so the JWE cookie is minted. A sign-in is more
        // important than one persisted row.
        const consoleErrSpy = vi
          .spyOn(console, "error")
          .mockImplementation(() => {});
        await expect(
          resolved.events.signIn({
            user: { email: "x@y.z" },
            account: { provider: "github", providerAccountId: "1" },
          }),
        ).resolves.toBeUndefined();
        expect(consoleErrSpy).toHaveBeenCalled();
        const firstCallArgs = consoleErrSpy.mock.calls[0] ?? [];
        // The first arg MUST carry the [cachicamas] prefix + the
        // event identity (`events.signIn`) so an operator can find
        // this log line in production. R3-12 (4R review): also
        // assert the actual Error is passed (not just a string), so
        // a future regression that logs only the prefix without the
        // error details is caught.
        expect(String(firstCallArgs[0] ?? "")).toMatch(
          /\[cachicamas\] events\.signIn/,
        );
        expect(firstCallArgs[1]).toBeInstanceOf(Error);
        expect(String((firstCallArgs[1] as Error).message)).toBe(
          "network glitch",
        );
        consoleErrSpy.mockRestore();
      });

      it("events.signIn skips handleSignIn when account is null (e.g. JWT-only sign-in)", async () => {
        const call = qwikAuthCalls[0];
        const resolved = (await call.resolveConfig({
          AUTH_GITHUB_ID: "test-id",
          AUTH_GITHUB_SECRET: "test-secret",
          AUTH_SECRET: "test-secret-base64",
          AUTH_TRUST_HOST: "true",
          AUTH_URL: "http://localhost:3015",
        })) as {
          events: {
            signIn: (event: {
              user: { email?: string | null };
              account?: { provider: string; providerAccountId: string } | null;
            }) => Promise<void> | void;
          };
        };
        handleSignInMock.mockClear();
        getSqlMock.mockClear();
        // Account is null: persistence MUST be skipped entirely.
        await resolved.events.signIn({ user: { email: "x@y.z" }, account: null });
        expect(getSqlMock).not.toHaveBeenCalled();
        expect(handleSignInMock).not.toHaveBeenCalled();
      });
    });
    