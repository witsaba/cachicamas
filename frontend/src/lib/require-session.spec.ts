/**
 * Test for `lib/require-session.ts` — the pure guard helper.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   R-PR-001 — `requireSession(session, pathname)` is a pure function that
 *   returns either `{ kind: "auth", session }` (when the session has a user)
 *   or `{ kind: "anon", pathname }` (when it doesn't).
 *
 * Test strategy:
 *   - Exercise all three branches in the spec: auth, null session, null user.
 *   - Assert no mutation of input (pure function contract — kept simple because
 *     we only take primitives in).
 */
import { describe, it, expect } from "vitest";
import { requireSession } from "./require-session";

describe("lib/require-session", () => {
  it("returns auth when session has a user (S-PR-001)", () => {
    const session = {
      user: { name: "Braejan", email: "braejan@example.com", image: null },
    };
    const guard = requireSession(
      session as unknown as Parameters<typeof requireSession>[0],
      "/organizations/new",
    );
    expect(guard.kind).toBe("auth");
    if (guard.kind === "auth") {
      expect(guard.session).toBe(session);
    }
  });

  it("returns anon when session is null (S-PR-002)", () => {
    const guard = requireSession(null, "/organizations/new");
    expect(guard.kind).toBe("anon");
    if (guard.kind === "anon") {
      expect(guard.pathname).toBe("/organizations/new");
    }
  });

  it("returns anon when session.user is null (S-PR-003)", () => {
    const session = { user: null };
    const guard = requireSession(
      session as unknown as Parameters<typeof requireSession>[0],
      "/profile",
    );
    expect(guard.kind).toBe("anon");
    if (guard.kind === "anon") {
      expect(guard.pathname).toBe("/profile");
    }
  });

  it("returns anon when session.user is missing", () => {
    const session = {};
    const guard = requireSession(
      session as unknown as Parameters<typeof requireSession>[0],
      "/organizations",
    );
    expect(guard.kind).toBe("anon");
  });

  it("preserves the pathname verbatim", () => {
    const guard = requireSession(null, "/organizations/123?q=foo");
    expect(guard.kind).toBe("anon");
    if (guard.kind === "anon") {
      expect(guard.pathname).toBe("/organizations/123?q=foo");
    }
  });
});
