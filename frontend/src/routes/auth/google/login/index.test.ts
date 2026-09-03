/**
 * `/auth/google/login` — pure handler test.
 *
 * We test `handleLoginGet` (exported) rather than the Qwik `onGet`
 * wrapper, so we can drive it with plain JS values instead of standing
 * up the Qwik server runtime. The Qwik wrapper is a 6-line shim that
 * just unpacks `RequestEvent` into `LoginInput`.
 *
 * Strict TDD: assertions written first; handler implemented to pass.
 */
import { describe, expect, test, vi } from "vitest";
import { handleLoginGet, OAUTH_STATE_COOKIE } from "./index";

interface CapturedRedirect extends Error {
  url: string;
}

function makeRedirectCatcher() {
  const redirects: string[] = [];
  return {
    redirects,
    doRedirect: (url: string): never => {
      redirects.push(url);
      const err = new Error("redirect") as Error & { url: string };
      err.url = url;
      throw err;
    },
  };
}

const BASE_ENV = {
  clientId: "google-client-id-123",
  redirectUri: "https://example.com/auth/google/callback",
  isProduction: false,
};

describe("handleLoginGet", () => {
  test("sets cachicamas_oauth_state cookie before redirecting", () => {
    const cookies: Array<{ name: string; value: string; opts: object }> = [];
    const catcher = makeRedirectCatcher();
    try {
      handleLoginGet({
        setStateCookie: (name, value, opts) => {
          cookies.push({ name, value, opts });
        },
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
      });
    } catch {
      // expected: doRedirect throws
    }
    expect(cookies).toHaveLength(1);
    expect(cookies[0]?.name).toBe(OAUTH_STATE_COOKIE);
    expect(cookies[0]?.value).toBeTruthy();
    // base64url alphabet only
    expect(cookies[0]?.value).not.toContain("+");
    expect(cookies[0]?.value).not.toContain("/");
    expect(cookies[0]?.value).not.toContain("=");
  });

  test("cookie attributes: HttpOnly, SameSite=Lax, Path=/, Max-Age=600", () => {
    const opts: Record<string, unknown> = {};
    const catcher = makeRedirectCatcher();
    try {
      handleLoginGet({
        setStateCookie: (_name, _value, o) => {
          Object.assign(opts, o);
        },
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
      });
    } catch {
      // expected
    }
    expect(opts.httpOnly).toBe(true);
    expect(opts.sameSite).toBe("lax");
    expect(opts.path).toBe("/");
    expect(opts.maxAge).toBe(600);
  });

  test("Secure=false in development (plain HTTP)", () => {
    const opts: Record<string, unknown> = {};
    const catcher = makeRedirectCatcher();
    try {
      handleLoginGet({
        setStateCookie: (_n, _v, o) => Object.assign(opts, o),
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
        isProduction: false,
      });
    } catch {
      // expected
    }
    expect(opts.secure).toBe(false);
  });

  test("Secure=true in production (HTTPS)", () => {
    const opts: Record<string, unknown> = {};
    const catcher = makeRedirectCatcher();
    try {
      handleLoginGet({
        setStateCookie: (_n, _v, o) => Object.assign(opts, o),
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
        isProduction: true,
      });
    } catch {
      // expected
    }
    expect(opts.secure).toBe(true);
  });

  test("redirects to the Google OAuth endpoint", () => {
    const catcher = makeRedirectCatcher();
    let url = "";
    try {
      handleLoginGet({
        setStateCookie: () => {},
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
      });
    } catch (e: unknown) {
      url = (e as CapturedRedirect).url;
    }
    expect(url).toContain("https://accounts.google.com/o/oauth2/v2/auth");
  });

  test("redirect URL contains client_id, redirect_uri, scope, state", () => {
    const catcher = makeRedirectCatcher();
    let url = "";
    try {
      handleLoginGet({
        setStateCookie: () => {},
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
      });
    } catch (e: unknown) {
      url = (e as CapturedRedirect).url;
    }
    const parsed = new URL(url);
    expect(parsed.searchParams.get("client_id")).toBe(BASE_ENV.clientId);
    expect(parsed.searchParams.get("redirect_uri")).toBe(BASE_ENV.redirectUri);
    expect(parsed.searchParams.get("scope")).toBe("openid email profile");
    expect(parsed.searchParams.get("response_type")).toBe("code");
    expect(parsed.searchParams.get("state")).toBeTruthy();
  });

  test("the state in the URL matches the state in the cookie (CSRF defence)", () => {
    const cookies: Array<{ value: string }> = [];
    const catcher = makeRedirectCatcher();
    let url = "";
    try {
      handleLoginGet({
        setStateCookie: (_n, v) => cookies.push({ value: v }),
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
      });
    } catch (e: unknown) {
      url = (e as CapturedRedirect).url;
    }
    const cookieState = cookies[0]?.value;
    const queryState = new URL(url).searchParams.get("state");
    expect(cookieState).toBe(queryState);
  });

  test("redirect URL omits client_secret", () => {
    const catcher = makeRedirectCatcher();
    let url = "";
    try {
      handleLoginGet({
        setStateCookie: () => {},
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
      });
    } catch (e: unknown) {
      url = (e as CapturedRedirect).url;
    }
    expect(url).not.toContain("client_secret");
    expect(url).not.toContain("secret");
  });

  test("each call generates a fresh state (no reuse)", () => {
    const states: string[] = [];
    for (let i = 0; i < 3; i++) {
      const catcher = makeRedirectCatcher();
      try {
        handleLoginGet({
          setStateCookie: (_n, v) => states.push(v),
          doRedirect: catcher.doRedirect,
          ...BASE_ENV,
        });
      } catch {
        // expected
      }
    }
    expect(new Set(states).size).toBe(3);
  });

  test("honours a custom generateStateFn for deterministic tests", () => {
    const cookieValues: string[] = [];
    const catcher = makeRedirectCatcher();
    try {
      handleLoginGet({
        setStateCookie: (_n, v) => cookieValues.push(v),
        doRedirect: catcher.doRedirect,
        ...BASE_ENV,
        generateStateFn: () => "deterministic-state",
      });
    } catch {
      // expected
    }
    expect(cookieValues[0]).toBe("deterministic-state");
  });

  test("throws when AUTH_GOOGLE_ID is missing", () => {
    expect(() =>
      handleLoginGet({
        setStateCookie: () => {},
        doRedirect: vi.fn() as unknown as (u: string) => never,
        clientId: "",
        redirectUri: BASE_ENV.redirectUri,
        isProduction: false,
      }),
    ).toThrow(/AUTH_GOOGLE_ID env is required/);
  });

  test("throws when PUBLIC_AUTH_REDIRECT_URI is missing", () => {
    expect(() =>
      handleLoginGet({
        setStateCookie: () => {},
        doRedirect: vi.fn() as unknown as (u: string) => never,
        clientId: BASE_ENV.clientId,
        redirectUri: "",
        isProduction: false,
      }),
    ).toThrow(/PUBLIC_AUTH_REDIRECT_URI env is required/);
  });
});