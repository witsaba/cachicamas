/**
 * `(app)/layout.tsx` — pure guard test.
 *
 * Spec reference: R-LAYOUT-1 / S-FE-030 / S-FE-031 / S-NFR-010.
 *
 * We test `guardLayout` (exported) rather than the Qwik `onRequest`
 * wrapper. The wrapper is a 6-line shim.
 */
import { describe, expect, test, vi } from "vitest";
import {
  guardLayout,
  LOGIN_ROUTE,
  SHARED_MAP_SESSION_KEY,
  type GuardInput,
} from "./layout";
import { refreshIfNeeded, signSession, verifySession, type SessionPayload } from "~/lib/server/session";

const NOW_MS = 1_700_000_000_000;
const TEST_SECRET = "test-cookie-secret-do-not-ship";

function makeInput(overrides: Partial<GuardInput> = {}): GuardInput {
  const sharedMap = new Map<string, unknown>();
  const redirected: string[] = [];
  const cookieSets: Array<{ name: string; value: string; opts: object }> = [];
  return {
    sessionCookieValue: null,
    verifySession,
    signSession,
    refreshIfNeeded,
    cookieSecret: TEST_SECRET,
    isProduction: false,
    setSessionCookie: (name, value, opts) => {
      cookieSets.push({ name, value, opts });
    },
    sharedMap,
    doRedirect: (url) => {
      redirected.push(url);
      throw new Error("redirect:" + url);
    },
    nowMs: () => NOW_MS,
    ...overrides,
  };
}

async function makeSignedSession(
  overrides: Partial<SessionPayload> = {},
): Promise<string> {
  const payload: SessionPayload = {
    user_id: 42,
    organization_id: 7,
    expires_at: Math.floor(NOW_MS / 1000) + 7 * 24 * 60 * 60,
    iat: Math.floor(NOW_MS / 1000),
    ...overrides,
  };
  return signSession(payload, TEST_SECRET);
}

describe("guardLayout", () => {
  test("missing cookie ⇒ redirect to login (S-FE-030)", async () => {
    const captured: string[] = [];
    const input = makeInput({
      sessionCookieValue: null,
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("redirect");
    expect(outcome.redirectTo).toBe(LOGIN_ROUTE);
    expect(captured[0]).toBe(LOGIN_ROUTE);
  });

  test("empty cookie ⇒ redirect to login", async () => {
    const captured: string[] = [];
    const input = makeInput({
      sessionCookieValue: "",
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("redirect");
  });

  test("tampered cookie ⇒ redirect to login (S-FE-031)", async () => {
    const valid = await makeSignedSession();
    const tampered = valid.slice(0, -2) + "xx"; // corrupt last 2 chars
    const captured: string[] = [];
    const input = makeInput({
      sessionCookieValue: tampered,
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("redirect");
    expect(captured[0]).toBe(LOGIN_ROUTE);
  });

  test("cookie signed with a different secret ⇒ redirect to login", async () => {
    const payload: SessionPayload = {
      user_id: 1,
      organization_id: 1,
      expires_at: Math.floor(NOW_MS / 1000) + 7 * 24 * 60 * 60,
      iat: Math.floor(NOW_MS / 1000),
    };
    const foreign = await signSession(payload, "wrong-secret");
    const captured: string[] = [];
    const input = makeInput({
      sessionCookieValue: foreign,
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("redirect");
  });

  test("expired cookie ⇒ redirect to login", async () => {
    const expired = await makeSignedSession({
      expires_at: Math.floor(NOW_MS / 1000) - 60,
    });
    const captured: string[] = [];
    const input = makeInput({
      sessionCookieValue: expired,
      doRedirect: (url) => {
        captured.push(url);
        return undefined as never;
      },
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("redirect");
    expect(captured[0]).toBe(LOGIN_ROUTE);
  });

  test("valid cookie ⇒ continue; stores session in sharedMap", async () => {
    const input = makeInput({
      sessionCookieValue: await makeSignedSession(),
      doRedirect: () => undefined as never,
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("continue");
    expect(outcome.session?.user_id).toBe(42);
    expect(outcome.session?.organization_id).toBe(7);
    expect(input.sharedMap?.get(SHARED_MAP_SESSION_KEY)).toEqual(
      outcome.session);
  });

  test("valid cookie with >24h life left ⇒ does NOT refresh", async () => {
    const cookieSets: Array<unknown> = [];
    const input = makeInput({
      sessionCookieValue: await makeSignedSession({
        expires_at: Math.floor(NOW_MS / 1000) + 48 * 60 * 60,
      }),
      setSessionCookie: (name, value, opts) => {
        cookieSets.push({ name, value, opts });
      },
      doRedirect: () => undefined as never,
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("continue");
    expect(outcome.refreshed).toBe(false);
    expect(cookieSets).toHaveLength(0);
  });

  test("valid cookie within 24h of expiry ⇒ refreshes and re-signs (S-NFR-010)", async () => {
    const cookieSets: Array<{ name: string; value: string }> = [];
    const input = makeInput({
      sessionCookieValue: await makeSignedSession({
        expires_at: Math.floor(NOW_MS / 1000) + 6 * 60 * 60, // 6h left
      }),
      setSessionCookie: (name, value) => {
        cookieSets.push({ name, value });
      },
      doRedirect: () => undefined as never,
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("continue");
    expect(outcome.refreshed).toBe(true);
    expect(cookieSets).toHaveLength(1);
    expect(cookieSets[0]?.name).toBe("cachicamas_session");
    // The new cookie verifies with the original secret and carries the
    // refreshed expiry.
    const verified = await verifySession(cookieSets[0]!.value, TEST_SECRET);
    expect(verified).not.toBeNull();
    expect(verified?.expires_at).toBeGreaterThan(
      Math.floor(NOW_MS / 1000) + 7 * 24 * 60 * 60 - 60,
    );
  });

  test("sliding-refresh cookie attributes: HttpOnly, SameSite=Lax, Path=/, Max-Age=604800", async () => {
    const opts: Record<string, unknown> = {};
    const input = makeInput({
      sessionCookieValue: await makeSignedSession({
        expires_at: Math.floor(NOW_MS / 1000) + 60,
      }),
      setSessionCookie: (_n, _v, o) => Object.assign(opts, o),
      doRedirect: () => undefined as never,
    });
    await guardLayout(input);
    expect(opts.httpOnly).toBe(true);
    expect(opts.sameSite).toBe("lax");
    expect(opts.path).toBe("/");
    expect(opts.maxAge).toBe(604800);
  });

  test("Secure=true on refreshed cookie when isProduction", async () => {
    const opts: Record<string, unknown> = {};
    const input = makeInput({
      sessionCookieValue: await makeSignedSession({
        expires_at: Math.floor(NOW_MS / 1000) + 60,
      }),
      isProduction: true,
      setSessionCookie: (_n, _v, o) => Object.assign(opts, o),
      doRedirect: () => undefined as never,
    });
    await guardLayout(input);
    expect(opts.secure).toBe(true);
  });

  test("refresh is skipped when setSessionCookie is absent (defensive)", async () => {
    const input = makeInput({
      sessionCookieValue: await makeSignedSession({
        expires_at: Math.floor(NOW_MS / 1000) + 60,
      }),
      setSessionCookie: undefined,
      doRedirect: () => undefined as never,
    });
    const outcome = await guardLayout(input);
    expect(outcome.decision).toBe("continue");
    expect(outcome.refreshed).toBe(false);
  });

  test("refreshIfNeeded override is honoured", async () => {
    const refreshIfNeededMock = vi.fn().mockReturnValue(null);
    const input = makeInput({
      sessionCookieValue: await makeSignedSession({
        expires_at: Math.floor(NOW_MS / 1000) + 60,
      }),
      refreshIfNeeded: refreshIfNeededMock as unknown as typeof refreshIfNeeded,
      doRedirect: () => undefined as never,
    });
    await guardLayout(input);
    expect(refreshIfNeededMock).toHaveBeenCalled();
  });
});