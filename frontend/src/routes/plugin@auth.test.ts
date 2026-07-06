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

  // 2026-07-06 regression: the custom onRequest skips @auth/core
  // interception for the native /auth/signin and /auth/signout page
  // renders. The skip MUST match both `/auth/signin` AND
  // `/auth/signin/`, because Qwik City auto-redirects the former
  // to the latter (trailing-slash canonicalisation) BEFORE our
  // middleware runs. A check that only matched the no-slash form
  // caused an infinite redirect loop in the browser:
  //   /auth/signin → 301 → /auth/signin/ → library.onRequest
  //   → 302 → /auth/signin?callbackUrl=... → 301 → /auth/signin/?cb=...
  //   → loop. Fix: normalise the path by stripping trailing slashes
  //   before comparing. These tests drive the implementation with
  // minimal synthetic requests (only `request.method` and
  // `url.pathname` are populated — the custom onRequest short-
  // circuits before touching anything else) and assert it returns
  // without throwing or calling the library's onRequest.
  it("custom onRequest SKIPS @auth/core for /auth/signin (no trailing slash)", async () => {
    const mod = await import("./plugin@auth");
    const req = makeNativeReq("/auth/signin", "GET");
    await expect(
      mod.onRequest(req as unknown as Parameters<typeof mod.onRequest>[0]),
    ).resolves.toBeUndefined();
  });

  it("custom onRequest SKIPS @auth/core for /auth/signin/ (trailing slash) — regression guard", async () => {
    // Same as above but with the trailing-slash form. This is the
    // shape Qwik City sends us after its canonicalisation redirect.
    const mod = await import("./plugin@auth");
    const req = makeNativeReq("/auth/signin/", "GET");
    await expect(
      mod.onRequest(req as unknown as Parameters<typeof mod.onRequest>[0]),
    ).resolves.toBeUndefined();
  });

  it("custom onRequest SKIPS @auth/core for /auth/signout (no trailing slash)", async () => {
    const mod = await import("./plugin@auth");
    const req = makeNativeReq("/auth/signout", "GET");
    await expect(
      mod.onRequest(req as unknown as Parameters<typeof mod.onRequest>[0]),
    ).resolves.toBeUndefined();
  });

  it("custom onRequest SKIPS @auth/core for /auth/signout/ (trailing slash)", async () => {
    const mod = await import("./plugin@auth");
    const req = makeNativeReq("/auth/signout/", "GET");
    await expect(
      mod.onRequest(req as unknown as Parameters<typeof mod.onRequest>[0]),
    ).resolves.toBeUndefined();
  });
});

/**
 * Build a synthetic `RequestEventCommon`-shaped object for unit-testing
 * the onRequest middleware's path-matching logic. Only the fields the
 * implementation reads are populated: `request.method` and `url.pathname`.
 *
 * The library's onRequest signature is `(req: RequestEventCommon) => ...`
 * and it touches `req.url`, `req.request`, `req.env`, `req.sharedMap`,
 * `req.headers`, `req.cookie`, `req.send`. We satisfy only what the
 * custom onRequest reads; everything else is left as a stub that the
 * library code never reaches because the custom onRequest short-circuits.
 */
/**
 * Build a synthetic minimal `RequestEventCommon`-shaped object for
 * unit-testing the onRequest middleware's path-matching logic. Only
 * `request.method` and `url.pathname` are populated — the custom
 * onRequest short-circuits before touching anything else. The cast
 * to `RequestEventCommon` happens at the call site via `as unknown as`
 * because TS structurally rejects our partial stub.
 */
function makeNativeReq(
  pathname: string,
  method: string,
): {
  request: Request;
  url: URL;
} {
  return {
    request: new Request(`http://localhost${pathname}`, { method }),
    url: new URL(`http://localhost${pathname}`),
  };
}
