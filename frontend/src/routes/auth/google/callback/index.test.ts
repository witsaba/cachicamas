/**
 * `/auth/google/callback` — pure handler test.
 *
 * Covers:
 *   - S-FE-010  state mismatch ⇒ /auth/error?reason=invalid_state
 *   - S-FE-011  success ⇒ session cookie set + 302 /home
 *   - S-FE-012  token exchange failure ⇒ /auth/error?reason=token_exchange_failed
 *   - S-FE-013  userinfo failure ⇒ /auth/error?reason=userinfo_failed
 *   - S-FE-080  status='blocked' ⇒ /auth/error?reason=blocked, NO session
 *   - S-FE-081  status='inactive' ⇒ session set + 302 /home
 *   - R-NFR-001 cookie attributes (HttpOnly, SameSite=Lax, Path=/, Max-Age=604800)
 *
 * We test `handleCallbackGet` (exported) rather than the Qwik `onGet`
 * wrapper.
 */
import { describe, expect, test, vi } from "vitest";
import {
  handleCallbackGet,
  type CallbackInput,
  type CallbackOutcome,
} from "./index";
import { verifySession } from "~/lib/server/session";
import { OAUTH_STATE_COOKIE } from "~/routes/auth/google/login";

const NOW_MS = 1_700_000_000_000;
const FIXED_QUERY = "https://example.com/auth/google/callback";

function makeInput(overrides: Partial<CallbackInput> = {}): CallbackInput {
  const cookies: Array<{ name: string; value: string; opts: object }> = [];
  const deleted: string[] = [];
  const redirects: string[] = [];

  return {
    query: new URLSearchParams("code=the-code&state=the-state"),
    url: `${FIXED_QUERY}?code=the-code&state=the-state`,
    stateCookieValue: "the-state",
    setSessionCookie: (name, value, opts) => {
      cookies.push({ name, value, opts });
    },
    clearStateCookie: () => {
      deleted.push(OAUTH_STATE_COOKIE);
    },
    doRedirect: (url) => {
      redirects.push(url);
      throw new Error("redirect:" + url);
    },
    doError: (status, message) => {
      throw new Error(`error ${status}: ${message}`);
    },
    env: {
      clientId: "google-client-id",
      clientSecret: "google-client-secret",
      cookieSecret: "test-cookie-secret",
      internalSecret: "test-internal-secret",
      backendUrl: "http://database_administrator:8080",
    },
    isProduction: false,
    nowMs: () => NOW_MS,
    fetchImpl: undefined,
    ...overrides,
  };
}

/**
 * Build a fetch mock that returns:
 *   - 1st call: token exchange response
 *   - 2nd call: userinfo response
 *   - 3rd call: backend bootstrap response
 */
function buildFetchMock(opts: {
  token?: { ok: boolean; body?: unknown };
  userinfo?: { ok: boolean; body?: unknown };
  bootstrap?: { ok: boolean; body?: unknown };
} = {}) {
  const token = {
    ok: opts.token?.ok ?? true,
    body: opts.token?.body ?? { access_token: "tok", token_type: "Bearer" },
  };
  const userinfo = {
    ok: opts.userinfo?.ok ?? true,
    body: opts.userinfo?.body ?? {
      id: "sub-1",
      email: "founder@example.com",
      email_verified: true,
      name: "Founder",
      picture: "https://example.com/p.png",
    },
  };
  const bootstrap = {
    ok: opts.bootstrap?.ok ?? true,
    body: opts.bootstrap?.body ?? {
      user_id: 42,
      organization_id: 7,
      status: "active",
    },
  };
  const fn = vi.fn(async (url: unknown) => {
    const u = typeof url === "string" ? url : String(url);
    if (u.includes("oauth2.googleapis.com/token")) {
      return new Response(JSON.stringify(token.body), {
        status: token.ok ? 200 : 400,
        headers: { "content-type": "application/json" },
      });
    }
    if (u.includes("userinfo")) {
      return new Response(JSON.stringify(userinfo.body), {
        status: userinfo.ok ? 200 : 500,
        headers: { "content-type": "application/json" },
      });
    }
    if (u.includes("/internal/auth/bootstrap")) {
      return new Response(JSON.stringify(bootstrap.body), {
        status: bootstrap.ok ? 200 : 500,
        headers: { "content-type": "application/json" },
      });
    }
    throw new Error(`unexpected URL: ${u}`);
  });
  return fn as unknown as typeof fetch;
}

describe("handleCallbackGet", () => {
  test("missing `code` query ⇒ /auth/error?reason=missing_code", async () => {
    const input = makeInput({
      query: new URLSearchParams("state=the-state"),
    });
    await expect(handleCallbackGet(input)).rejects.toThrow(/redirect:/);
    // The handler throws via doRedirect — re-call without the throw by
    // mocking doRedirect to capture instead.
    const captured: string[] = [];
    const input2 = makeInput({
      query: new URLSearchParams("state=the-state"),
      doRedirect: (url) => {
        captured.push(url);
        // don't throw
        return undefined as never;
      },
    });
    await handleCallbackGet(input2);
    expect(captured[0]).toBe("/auth/error?reason=missing_code");
  });

  test("state mismatch (query ≠ cookie) ⇒ /auth/error?reason=invalid_state", async () => {
    const captured: string[] = [];
    const input = makeInput({
      query: new URLSearchParams("code=the-code&state=different-state"),
      stateCookieValue: "the-state",
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    await handleCallbackGet(input);
    expect(captured[0]).toBe("/auth/error?reason=invalid_state");
  });

  test("missing state cookie ⇒ /auth/error?reason=invalid_state", async () => {
    const captured: string[] = [];
    const input = makeInput({ stateCookieValue: null });
    const input2 = {
      ...input,
      doRedirect: (url: string) => {
        captured.push(url);
        return undefined as never;
      },
    };
    await handleCallbackGet(input2);
    expect(captured[0]).toBe("/auth/error?reason=invalid_state");
  });

  test("missing state query ⇒ /auth/error?reason=invalid_state", async () => {
    const captured: string[] = [];
    const input = makeInput({
      query: new URLSearchParams("code=the-code"),
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    await handleCallbackGet(input);
    expect(captured[0]).toBe("/auth/error?reason=invalid_state");
  });

  test("token exchange failure (400) ⇒ /auth/error?reason=token_exchange_failed", async () => {
    const captured: string[] = [];
    const input = makeInput({
      fetchImpl: buildFetchMock({ token: { ok: false } }),
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    await handleCallbackGet(input);
    expect(captured[0]).toBe(
      "/auth/error?reason=token_exchange_failed",
    );
  });

  test("userinfo failure (500) ⇒ /auth/error?reason=userinfo_failed", async () => {
    const captured: string[] = [];
    const input = makeInput({
      fetchImpl: buildFetchMock({ userinfo: { ok: false } }),
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    await handleCallbackGet(input);
    expect(captured[0]).toBe("/auth/error?reason=userinfo_failed");
  });

  test("backend bootstrap failure (500) ⇒ /auth/error?reason=internal_error", async () => {
    const captured: string[] = [];
    const input = makeInput({
      fetchImpl: buildFetchMock({ bootstrap: { ok: false } }),
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    await handleCallbackGet(input);
    expect(captured[0]).toBe("/auth/error?reason=internal_error");
  });

  test("status='blocked' ⇒ /auth/error?reason=blocked, NO session cookie set", async () => {
    const cookies: Array<{ name: string }> = [];
    const captured: string[] = [];
    const input = makeInput({
      fetchImpl: buildFetchMock({
        bootstrap: {
          ok: true,
          body: { user_id: 1, organization_id: 1, status: "blocked" },
        },
      }),
      setSessionCookie: (name, _v, _o) => {
        cookies.push({ name });
      },
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    await handleCallbackGet(input);
    expect(captured[0]).toBe("/auth/error?reason=blocked");
    expect(cookies).toHaveLength(0); // critical: no session cookie for blocked
  });

  test("status='inactive' ⇒ session set + 302 /home (allowed login)", async () => {
    const cookies: Array<{ name: string; value: string }> = [];
    const captured: string[] = [];
    const input = makeInput({
      fetchImpl: buildFetchMock({
        bootstrap: {
          ok: true,
          body: { user_id: 1, organization_id: 1, status: "inactive" },
        },
      }),
      setSessionCookie: (name, value) => {
        cookies.push({ name, value });
      },
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    await handleCallbackGet(input);
    expect(captured[0]).toBe("/home");
    expect(cookies).toHaveLength(1);
    expect(cookies[0]?.name).toBe("cachicamas_session");
  });

  test("success ⇒ sets cachicamas_session + 302 /home", async () => {
    const cookies: Array<{ name: string; value: string; opts: object }> = [];
    const deleted: string[] = [];
    const captured: string[] = [];
    const input = makeInput({
      fetchImpl: buildFetchMock(),
      setSessionCookie: (name, value, opts) => {
        cookies.push({ name, value, opts });
      },
      clearStateCookie: () => {
        deleted.push(OAUTH_STATE_COOKIE);
      },
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    await handleCallbackGet(input);
    expect(captured[0]).toBe("/home");
    expect(cookies).toHaveLength(1);
    expect(cookies[0]?.name).toBe("cachicamas_session");
    expect(cookies[0]?.value).toBeTruthy();
    // The state cookie MUST be cleared on the success path.
    expect(deleted).toContain(OAUTH_STATE_COOKIE);
  });

  test("the signed session cookie verifies with the same secret", async () => {
    const cookies: Array<{ value: string }> = [];
    const input = makeInput({
      fetchImpl: buildFetchMock(),
      setSessionCookie: (_n, value) => cookies.push({ value }),
      doRedirect: () => undefined as never,
    });
    await handleCallbackGet(input);
    const verified = await verifySession(
      cookies[0]!.value,
      "test-cookie-secret",
    );
    expect(verified).not.toBeNull();
    expect(verified?.user_id).toBe(42);
    expect(verified?.organization_id).toBe(7);
  });

  test("session cookie attributes: HttpOnly, SameSite=Lax, Path=/, Max-Age=604800", async () => {
    const opts: Record<string, unknown> = {};
    const input = makeInput({
      fetchImpl: buildFetchMock(),
      setSessionCookie: (_n, _v, o) => Object.assign(opts, o),
      doRedirect: () => undefined as never,
    });
    await handleCallbackGet(input);
    expect(opts.httpOnly).toBe(true);
    expect(opts.sameSite).toBe("lax");
    expect(opts.path).toBe("/");
    expect(opts.maxAgeSeconds).toBe(604800);
  });

  test("Secure=true on the session cookie when isProduction", async () => {
    const opts: Record<string, unknown> = {};
    const input = makeInput({
      fetchImpl: buildFetchMock(),
      isProduction: true,
      setSessionCookie: (_n, _v, o) => Object.assign(opts, o),
      doRedirect: () => undefined as never,
    });
    await handleCallbackGet(input);
    expect(opts.secure).toBe(true);
  });

  test("Secure=false on the session cookie in dev", async () => {
    const opts: Record<string, unknown> = {};
    const input = makeInput({
      fetchImpl: buildFetchMock(),
      isProduction: false,
      setSessionCookie: (_n, _v, o) => Object.assign(opts, o),
      doRedirect: () => undefined as never,
    });
    await handleCallbackGet(input);
    expect(opts.secure).toBe(false);
  });

  test("calls token endpoint with the OAuth code", async () => {
    const fetchMock = buildFetchMock();
    const input = makeInput({
      fetchImpl: fetchMock,
      doRedirect: () => undefined as never,
    });
    await handleCallbackGet(input);
    // 1st fetch = token, 2nd = userinfo, 3rd = bootstrap
    const firstCall = (fetchMock as unknown as { mock: { calls: unknown[][] } })
      .mock.calls[0]!;
    const calledUrl = firstCall[0] as string;
    expect(calledUrl).toContain("oauth2.googleapis.com/token");
    const init = firstCall[1] as RequestInit;
    const body = init.body as URLSearchParams;
    expect(body.get("code")).toBe("the-code");
  });

  test("calls bootstrap with X-Internal-Secret", async () => {
    const fetchMock = buildFetchMock();
    const input = makeInput({
      fetchImpl: fetchMock,
      doRedirect: () => undefined as never,
    });
    await handleCallbackGet(input);
    const calls = (fetchMock as unknown as { mock: { calls: unknown[][] } })
      .mock.calls;
    const bootstrapCall = calls.find((c) =>
      String(c[0]).includes("/internal/auth/bootstrap"),
    );
    expect(bootstrapCall).toBeDefined();
    const init = bootstrapCall![1] as RequestInit;
    const headers = init.headers as Record<string, string>;
    expect(headers["x-internal-secret"]).toBe("test-internal-secret");
  });
});

describe("CallbackOutcome shape (smoke)", () => {
  test("the happy-path outcome is reason='ok' with a session cookie", async () => {
    const outcome: CallbackOutcome = await handleCallbackGet(
      makeInput({
        fetchImpl: buildFetchMock(),
        doRedirect: () => undefined as never,
      }),
    );
    expect(outcome.reason).toBe("ok");
    expect(outcome.sessionCookieValue).toBeTruthy();
    expect(outcome.redirectTo).toBe("/home");
  });
});