/**
 * Test for `lib/sign-in-callback.ts` — Auth.js `events.signIn` callback
 * that persists the GitHub identity to the local Postgres.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-010..R-FA-030 — auto-link-on-email-match; one `identity.user`
 *   per email; multiple `identity.account` rows for the same user.
 *
 * Test strategy:
 *   - Mock the `SqlClient` to capture every SQL invocation and to
 *     script its return values per test (hit / miss / new-user).
 *   - Assert the captured SQL strings and bind values, NOT the
 *     internal implementation, so the spec scenarios pass against
 *     any future refactor that keeps the same wire contract.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { handleSignIn, type SqlClient, type SignInEvent } from "./sign-in-callback";

/**
 * Build a fake `sql` client that:
 *   - records every call into `sqlCalls[]` (template + bind values).
 *   - returns whatever `scriptedResponses` produces for the call index,
 *     or an empty array by default.
 */
function makeFakeSql(
  scriptedResponses: Array<unknown[]>,
): { sql: SqlClient; calls: Array<{ template: string; values: unknown[] }> } {
  const calls: Array<{ template: string; values: unknown[] }> = [];
  let callIndex = 0;
  const sql: SqlClient = (strings, ...values) => {
    const template = strings.join("?");
    calls.push({ template, values });
    const response = scriptedResponses[callIndex] ?? [];
    callIndex += 1;
    return Promise.resolve(response);
  };
  return { sql, calls };
}

const GITHUB_EVENT_BASE: SignInEvent = {
  user: {
    id: "12345",
    name: "Octocat",
    email: "octo@example.com",
    image: "https://github.com/avatars/octo.png",
  },
  account: {
    provider: "github",
    providerAccountId: "12345",
    access_token: "gho_test_access_token",
    refresh_token: "ghr_test_refresh_token",
    expires_at: 9999999999,
    token_type: "bearer",
    scope: "read:user,user:email",
  },
  isNewUser: true,
};

describe("lib/sign-in-callback — auto-link on email match", () => {
  let sql: SqlClient;
  let calls: Array<{ template: string; values: unknown[] }>;
  let result: boolean;

  beforeEach(async () => {
    // T2.5: implementation throws "not implemented". Each test catches
    // and stores the throw so we can assert the RED-state contract.
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("T2.6 GREEN happy path (existing user, new account link): SELECT then INSERT INTO identity.account", async () => {
    const existingUserRow = [{ id: "u_42", email: "octo@example.com", name: "Octocat", image_url: "https://github.com/avatars/octo.png" }];
    const fake = makeFakeSql([existingUserRow, []]); // SELECT returns the user, INSERT returns []
    result = await handleSignIn(fake.sql, GITHUB_EVENT_BASE);
    expect(result).toBe(true);
    expect(fake.calls).toHaveLength(2);
    // 1st call: SELECT identity.user by case-insensitive email.
    expect(fake.calls[0].template).toMatch(/SELECT[\s\S]*identity\.user/i);
    expect(fake.calls[0].template).toMatch(/lower\(email\)\s*=\s*lower\(\?/i);
    expect(fake.calls[0].values).toContain("octo@example.com");
    // 2nd call: UPSERT identity.account.
    expect(fake.calls[1].template).toMatch(/INSERT INTO identity\.account/i);
    expect(fake.calls[1].template).toMatch(/ON CONFLICT/i);
    expect(fake.calls[1].values).toEqual(
      expect.arrayContaining(["u_42", "github", "12345", "gho_test_access_token"]),
    );
  });

  it("T2.6 GREEN miss path (no user exists): SELECT then INSERT user, then INSERT account", async () => {
    const fake = makeFakeSql([
      [], // SELECT returns []
      [{ id: "u_new_99" }], // INSERT user returns new id
      [], // INSERT account returns []
    ]);
    result = await handleSignIn(fake.sql, GITHUB_EVENT_BASE);
    expect(result).toBe(true);
    expect(fake.calls).toHaveLength(3);
    // 1st: SELECT (miss).
    expect(fake.calls[0].template).toMatch(/SELECT[\s\S]*identity\.user/i);
    // 2nd: INSERT user (creates new identity.user row).
    expect(fake.calls[1].template).toMatch(/INSERT INTO identity\.user/i);
    expect(fake.calls[1].template).toMatch(/RETURNING id/i);
    expect(fake.calls[1].values).toEqual(
      expect.arrayContaining(["octo@example.com", "Octocat", "https://github.com/avatars/octo.png"]),
    );
    // 3rd: INSERT account with the new user id.
    expect(fake.calls[2].template).toMatch(/INSERT INTO identity\.account/i);
    expect(fake.calls[2].values[0]).toBe("u_new_99");
  });

  it("T2.7 TRIANGULATE: same email, different providerAccountId creates a NEW identity.account row reusing the existing identity.user", async () => {
    // The existing user row (linktest@example.com) was created earlier
    // (e.g., via the first sign-in with providerAccountId='FIRST_ID').
    const existingUserRow = [{ id: "u_existing", email: "linktest@example.com", name: "Link Tester", image_url: "" }];
    const fake = makeFakeSql([existingUserRow, []]);
    const linkEvent: SignInEvent = {
      ...GITHUB_EVENT_BASE,
      user: { ...GITHUB_EVENT_BASE.user, email: "linktest@example.com" },
      account: { ...GITHUB_EVENT_BASE.account, providerAccountId: "NEW_ID" },
    };
    result = await handleSignIn(fake.sql, linkEvent);
    expect(result).toBe(true);
    expect(fake.calls).toHaveLength(2);
    // Only ONE call to identity.user (SELECT, no INSERT — existing user
    // is reused).
    expect(fake.calls.filter((c) => /INSERT INTO identity\.user/i.test(c.template))).toHaveLength(0);
    // One call to identity.account INSERT.
    expect(fake.calls[1].template).toMatch(/INSERT INTO identity\.account/i);
    // The new account_id must reach the INSERT.
    expect(fake.calls[1].values).toEqual(
      expect.arrayContaining(["u_existing", "github", "NEW_ID"]),
    );
  });

  it("denies sign-in when the GitHub profile has no email (privacy-protected GitHub users)", async () => {
    const fake = makeFakeSql([]);
    const noEmailEvent: SignInEvent = {
      ...GITHUB_EVENT_BASE,
      user: { ...GITHUB_EVENT_BASE.user, email: null },
    };
    result = await handleSignIn(fake.sql, noEmailEvent);
    expect(result).toBe(false);
    // Zero SQL calls when the email is missing — we deny without touching the DB.
    expect(fake.calls).toHaveLength(0);
  });

  it("denies sign-in when the GitHub profile has an empty email string", async () => {
    const fake = makeFakeSql([]);
    const emptyEmailEvent: SignInEvent = {
      ...GITHUB_EVENT_BASE,
      user: { ...GITHUB_EVENT_BASE.user, email: "" },
    };
    result = await handleSignIn(fake.sql, emptyEmailEvent);
    expect(result).toBe(false);
    expect(fake.calls).toHaveLength(0);
  });

  it("preserves the existing identity.user fields on a re-link (does not overwrite name/image on SELECT hit)", async () => {
    // The design says: when a user re-signs-in, we DO NOT overwrite
    // their stored name/image with GitHub's current snapshot — that
    // would clobber edits the user may have made in-app. The
    // account row gets refreshed, but the user row stays.
    const existingUserRow = [
      { id: "u_stable", email: "octo@example.com", name: "Octocat (custom)", image_url: "https://cdn.example.com/me.jpg" },
    ];
    const fake = makeFakeSql([existingUserRow, []]);
    result = await handleSignIn(fake.sql, GITHUB_EVENT_BASE);
    expect(result).toBe(true);
    // Assert NO UPDATE was issued on identity.user — only the INSERT
    // into identity.account.
    expect(fake.calls.filter((c) => /UPDATE\s+identity\.user/i.test(c.template))).toHaveLength(0);
    // INSERT into identity.account carries the new OAuth tokens.
    expect(fake.calls[1].template).toMatch(/INSERT INTO identity\.account/i);
    expect(fake.calls[1].values).toEqual(
      expect.arrayContaining([
        "u_stable",
        "github",
        "12345",
        "gho_test_access_token",
      ]),
    );
  });
});