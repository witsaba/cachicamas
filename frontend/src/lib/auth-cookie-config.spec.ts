import { describe, expect, it } from "vitest";
import { SESSION_COOKIE_OPTIONS } from "~/lib/auth-cookie-config";

/**
 * TDD coverage for the explicit session-cookie attribute policy.
 *
 * Reference: sdd/security-vulnerability-remediation/spec/session-cookie-attr-policy
 *   REQ-01 — HttpOnly, SameSite=Lax, Path=/ MUST be declared.
 *   REQ-02 — `secure` is derived by Auth.js from the URL scheme.
 *
 * The values are baked into a typed constant (`SESSION_COOKIE_OPTIONS`)
 * that the Auth.js config callback applies to `cookies.sessionToken`.
 * The E2E test (`e2e/sign-in-cookie-attrs.spec.ts`) verifies the
 * runtime behaviour; this unit test pins the source-of-truth constants
 * so a refactor cannot accidentally weaken the policy.
 */
describe("SESSION_COOKIE_OPTIONS", () => {
  it("RED-T-SEC-COOKIE-001: httpOnly is true", () => {
    expect(SESSION_COOKIE_OPTIONS.httpOnly).toBe(true);
  });

  it("RED-T-SEC-COOKIE-002: sameSite is 'lax'", () => {
    expect(SESSION_COOKIE_OPTIONS.sameSite).toBe("lax");
  });

  it("RED-T-SEC-COOKIE-003: path is '/'", () => {
    expect(SESSION_COOKIE_OPTIONS.path).toBe("/");
  });

  it("RED-T-SEC-COOKIE-004: does NOT statically set `secure` (Auth.js derives it)", () => {
    // The `secure` flag is intentionally omitted so Auth.js can
    // toggle it based on the request URL. Pin that contract.
    expect(
      "secure" in SESSION_COOKIE_OPTIONS,
      "secure should be omitted from the explicit config",
    ).toBe(false);
  });
});
