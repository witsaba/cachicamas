/**
 * Test for `lib/require-auth-redirect.ts` — the server-side guard that
 * redirects anonymous visitors from protected routes to the native
 * `/auth/signin` page.
 *
 * Reference: engram observation `cachicamas/auth/anatomy` (2026-07-06)
 * and the native-auth-UI decision (2026-07-06).
 *
 * What we assert:
 *   - `buildSigninRedirectUrl("/foo")` produces
 *     `/auth/signin?callbackUrl=%2Ffoo` (URL-encoded).
 *   - `buildSigninRedirectUrl("/foo?bar=baz")` preserves the search
 *     string in the callback (the post-signin round-trip lands on
 *     the same view).
 *   - `requireAuthRedirect` is a no-op when `sharedMap.session.user`
 *     is present (visitor is authenticated).
 *   - `requireAuthRedirect` throws a 302 redirect to the canonical
 *     signin URL when `sharedMap.session` is null (anon visitor).
 *   - `requireAuthRedirect` throws a 302 redirect when `sharedMap.session`
 *     is present but `.user` is missing/malformed (e.g. an expired
 *     session that the JWT loader still returned).
 *   - The redirect target's `callbackUrl` is the request's pathname
 *     + search, URL-encoded.
 */
import { describe, it, expect } from "vitest";
import {
  buildSigninRedirectUrl,
  requireAuthRedirect,
} from "./require-auth-redirect";

/**
 * Build a synthetic `RequestEventCommon`-shaped object for unit-testing
 * the request handler. We only populate what the implementation reads
 * (`sharedMap.session` + `url.pathname` + `url.search`); the rest is
 * stubbed. The cast to `RequestEventCommon` happens at the call site
 * via `as unknown as` because TS structurally rejects our partial stub.
 */
function makeEvent(opts: {
  pathname: string;
  search?: string;
  session?: { user?: unknown } | null;
}): {
  sharedMap: Map<string, unknown>;
  url: URL;
  redirect: (status: number, url: string) => never;
} {
  const fullUrl = `http://localhost${opts.pathname}${opts.search ?? ""}`;
  const url = new URL(fullUrl);
  const sharedMap = new Map<string, unknown>();
  if (opts.session !== undefined) {
    sharedMap.set("session", opts.session);
  }
  return {
    sharedMap,
    url,
    redirect: (status: number, target: string) => {
      throw new RedirectError(status, target);
    },
  };
}

/**
 * Sentinel thrown by the synthetic `event.redirect` so the test can
 * assert the redirect was triggered with the expected status + URL.
 * Mirrors the Qwik City `RedirectMessage` semantics without pulling
 * the whole Qwik runtime.
 */
class RedirectError extends Error {
  constructor(
    public readonly status: number,
    public readonly location: string,
  ) {
    super(`RedirectError(${status}, ${location})`);
    this.name = "RedirectError";
  }
}

describe("lib/require-auth-redirect — buildSigninRedirectUrl", () => {
  it("encodes a simple pathname into a /auth/signin?callbackUrl=... URL", () => {
    expect(buildSigninRedirectUrl("/home")).toBe(
      "/auth/signin?callbackUrl=%2Fhome",
    );
  });

  it("preserves the search string in the callback (post-signin round-trip lands on the same view)", () => {
    // Protected routes may carry meaningful query params (e.g.
    // `/organizations/[id]?tab=members`). Stripping them would silently
    // lose context on the round trip — assert we keep them.
    expect(buildSigninRedirectUrl("/organizations/42?tab=members")).toBe(
      "/auth/signin?callbackUrl=%2Forganizations%2F42%3Ftab%3Dmembers",
    );
  });

  it("encodes special characters (spaces, &, =, #)", () => {
    // The callback URL is itself a URL with possible `?` and `&`
    // characters; encoding it once makes it a safe value for the
    // OUTER `?callbackUrl=...` query parameter. Assert the encoder
    // handles the awkward edge case of an ampersand inside the value.
    expect(buildSigninRedirectUrl("/foo?bar=baz&qux=hello%20world")).toBe(
      "/auth/signin?callbackUrl=%2Ffoo%3Fbar%3Dbaz%26qux%3Dhello%2520world",
    );
  });

  it("handles the root path", () => {
    expect(buildSigninRedirectUrl("/")).toBe("/auth/signin?callbackUrl=%2F");
  });

  it("strips trailing slashes from the callback URL (Qwik trailing-slash canonicalisation defence)", async () => {
    // Qwik City auto-redirects /home → /home/ BEFORE middleware runs,
    // so by the time the handler sees the request, the pathname is
    // already `/home/`. Stripping the trailing slash keeps the
    // post-signin redirect URL clean (`/home` instead of `/home/`).
    expect(buildSigninRedirectUrl("/home/")).toBe(
      "/auth/signin?callbackUrl=%2Fhome",
    );
    expect(buildSigninRedirectUrl("/organizations/42/")).toBe(
      "/auth/signin?callbackUrl=%2Forganizations%2F42",
    );
    expect(buildSigninRedirectUrl("/")).toBe("/auth/signin?callbackUrl=%2F");
    expect(buildSigninRedirectUrl("//")).toBe("/auth/signin?callbackUrl=%2F");
  });
});

describe("lib/require-auth-redirect — requireAuthRedirect handler", () => {
  /**
   * Helper: drive the handler and capture the (status, location) of
   * the thrown redirect. Returns `null` when the handler resolves
   * (the authed visitor path).
   */
  async function driveAndCapture(
    event: ReturnType<typeof makeEvent>,
  ): Promise<{ status: number; location: string } | null> {
    try {
      await requireAuthRedirect(
        event as unknown as Parameters<typeof requireAuthRedirect>[0],
      );
      return null;
    } catch (err) {
      // Extract only the redirect properties — Error carries extra
      // fields (message, name, stack) that we don't want to assert on.
      const re = err as { status: number; location: string };
      return { status: re.status, location: re.location };
    }
  }

  it("no-ops when sharedMap.session.user is present (visitor is authenticated)", async () => {
    const event = makeEvent({
      pathname: "/home",
      session: { user: { name: "Alice" } },
    });
    const result = await driveAndCapture(event);
    expect(result).toBeNull();
  });

  it("redirects (302) to /auth/signin when sharedMap.session is null (anon visitor)", async () => {
    const event = makeEvent({
      pathname: "/home",
      session: null,
    });
    const result = await driveAndCapture(event);
    expect(result).toEqual({
      status: 302,
      location: "/auth/signin?callbackUrl=%2Fhome",
    });
  });

  it("redirects when sharedMap.session is undefined (handler ran before the session loader populated sharedMap — defensive)", async () => {
    const event = makeEvent({
      pathname: "/profile",
    });
    // session is undefined (not set in sharedMap). The handler MUST
    // treat this as anon and redirect, NOT pass through.
    const result = await driveAndCapture(event);
    expect(result).toEqual({
      status: 302,
      location: "/auth/signin?callbackUrl=%2Fprofile",
    });
  });

  it("redirects when sharedMap.session.user is missing (expired / malformed session)", async () => {
    const event = makeEvent({
      pathname: "/organizations",
      session: { user: undefined },
    });
    const result = await driveAndCapture(event);
    expect(result).toEqual({
      status: 302,
      location: "/auth/signin?callbackUrl=%2Forganizations",
    });
  });

  it("the redirect target's callbackUrl is the request's pathname + search, URL-encoded", async () => {
    const event = makeEvent({
      pathname: "/organizations/42",
      search: "?tab=members",
      session: null,
    });
    const result = await driveAndCapture(event);
    expect(result).toEqual({
      status: 302,
      location:
        "/auth/signin?callbackUrl=%2Forganizations%2F42%3Ftab%3Dmembers",
    });
  });
});
