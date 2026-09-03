/**
 * `/auth/logout` — pure handler test.
 *
 * Spec reference: R-FE-003 / S-FE-020. POST clears the session cookie
 * (Max-Age=0; HttpOnly; SameSite=Lax; Path=/) and 302s to `/`.
 */
import { describe, expect, test } from "vitest";
import { handleLogout } from "./index";
import { SESSION_COOKIE_NAME } from "~/lib/server/session";

describe("handleLogout", () => {
  test("clears the cachicamas_session cookie", () => {
    const deleted: string[] = [];
    try {
      handleLogout({
        clearSessionCookie: (name) => deleted.push(name),
        doRedirect: () => {
          throw new Error("redirect");
        },
      });
    } catch {
      // expected
    }
    expect(deleted).toEqual([SESSION_COOKIE_NAME]);
  });

  test("302s to / (the public landing)", () => {
    const redirects: string[] = [];
    try {
      handleLogout({
        clearSessionCookie: () => {},
        doRedirect: (url) => {
          redirects.push(url);
          throw new Error("redirect");
        },
      });
    } catch {
      // expected: doRedirect throws
    }
    expect(redirects[0]).toBe("/");
  });

  test("is idempotent: clearing a missing cookie does not crash", () => {
    expect(() =>
      handleLogout({
        clearSessionCookie: () => {},
        doRedirect: () => {
          throw new Error("redirect");
        },
      }),
    ).toThrow(/redirect/);
  });

  test("clears BEFORE redirecting (order matters: a crash on the redirect\nmust not leak the cookie)", () => {
    const events: string[] = [];
    try {
      handleLogout({
        clearSessionCookie: () => events.push("clear"),
        doRedirect: () => {
          events.push("redirect");
          throw new Error("redirect");
        },
      });
    } catch {
      // expected
    }
    expect(events[0]).toBe("clear");
    expect(events[1]).toBe("redirect");
  });
});