/**
 * Auth.js session-cookie attribute policy.
 *
 * Reference: sdd/security-vulnerability-remediation/spec/session-cookie-attr-policy
 *   REQ-01 — the session cookie MUST explicitly declare HttpOnly,
 *            SameSite=Lax, Path=/, and (under HTTPS) Secure.
 *   REQ-02 — production HTTPS deployments SHOULD use the `__Host-`
 *            prefix when Auth.js supports `cookieName`. We DO NOT
 *            use it today because our middleware stack (fronting
 *            nginx, possibly on a path-prefixed origin) does not
 *            always satisfy the `__Host-` invariants (Secure +
 *            Path=/ + no Domain). The `__Secure-` prefix that
 *            Auth.js applies automatically under HTTPS is the
 *            locked convention here.
 *
 * Why explicit and not implied: Auth.js's defaults ARE these
 * values, but every other security policy audit we have done
 * has found that "secure by default" is a top-tier risk. Making
 * the contract visible in the source means:
 *   1. A reviewer can audit the policy without consulting the
 *      upstream library.
 *   2. A future Auth.js version with a different default will
 *      not silently change our posture.
 *   3. The E2E test (`e2e/sign-in-cookie-attrs.spec.ts`) has a
 *      stable contract to pin against.
 *
 * `secure` is intentionally OMITTED. Auth.js derives it from the
 * request URL: HTTPS origin → `secure: true`, HTTP origin → `false`.
 * Coding the flag statically would either break local dev (always
 * true) or break production (always false). The library's logic is
 * correct; we just pin the other three attributes.
 */
export const SESSION_COOKIE_OPTIONS = {
  httpOnly: true,
  sameSite: "lax" as const,
  path: "/",
} as const;
