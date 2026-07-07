/**
 * Structural test for the workspace detail route guard.
 *
 * Mirrors the locked route-guard pattern from openspec/AGENTS.md:
 *   R-PR-003 — every authed-only route MUST publish a structural
 *   spec asserting (a) the import of requireAuthRedirect, (b) the
 *   re-export via `export { requireAuthRedirect as onRequest }`,
 *   and (c) the routeLoader$ guard hook (requireOwnboarding for
 *   workspaced routes).
 *
 * The regexes below use `new RegExp(string)` form so the TypeScript
 * parser does not interpret backslash escapes inside the literal.
 */
import { describe, test, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const SOURCE = readFileSync(join(__dirname, "index.tsx"), "utf8");

const REQUIRE_AUTH_REDIRECT_IMPORT = new RegExp(
  "import\\s*\\{\\s*requireAuthRedirect\\s*\\}\\s*from\\s*['\"]~/lib/require-auth-redirect['\"]",
);
const SET_SSR_COOKIE_HEADER_IMPORT = new RegExp(
  "import\\s*\\{\\s*setSsrCookieHeader\\s*\\}\\s*from\\s*['\"]~/lib/ssr-cookie-context['\"]",
);
const SET_SSR_COOKIE_HEADER_CALL = new RegExp("setSsrCookieHeader\\(");
const ROUTE_LOADER_PRESENT = new RegExp("routeLoader\\$");
const REQUIRE_OWNBOARDING_PRESENT = new RegExp("requireOwnboarding");

describe("workspaces/:id — route guard structural wiring (R-PR-003)", () => {
  test("imports requireAuthRedirect", () => {
    expect(REQUIRE_AUTH_REDIRECT_IMPORT.test(SOURCE)).toBe(true);
  });

  test("imports setSsrCookieHeader (SSR cookie forwarding, S-WS-AUTH-CHAIN-SSR-001)", () => {
    // 2026-07-07 regression: this route forgot to import + call
    // setSsrCookieHeader in its onRequest, so the SSR cookie context
    // was empty when getWorkspace() ran from useTask$ during SSR. The
    // backend's IdentityFromCookie middleware rejected every SSR fetch
    // with 401 "authentication required" and the page rendered the
    // error alert instead of the workspace detail. Pin the import so a
    // future refactor cannot silently drop the SSR cookie capture.
    expect(SET_SSR_COOKIE_HEADER_IMPORT.test(SOURCE)).toBe(true);
  });

  test("calls setSsrCookieHeader in onRequest BEFORE the auth + ownboarding guards", () => {
    // The capture MUST happen first — if requireAuthRedirect or
    // requireOwnboarding throws (anonymous user or no-org), the
    // render aborts before useTask$ ever fires, but for the AUTHED
    // path the order matters because the module-level
    // currentRequestCookie variable must be populated BEFORE any
    // ssrFetch call (which happens inside useTask$ during SSR).
    expect(SET_SSR_COOKIE_HEADER_CALL.test(SOURCE)).toBe(true);
    const onRequestBlock = SOURCE.match(
      /export const onRequest[\s\S]*?\n\};/,
    );
    expect(onRequestBlock).not.toBeNull();
    if (onRequestBlock) {
      const block = onRequestBlock[0];
      const cookieIdx = block.indexOf("setSsrCookieHeader(");
      const authIdx = block.indexOf("requireAuthRedirect(");
      expect(cookieIdx).toBeGreaterThanOrEqual(0);
      expect(authIdx).toBeGreaterThan(cookieIdx);
    }
  });

  test("uses routeLoader$ to call requireOwnboarding (ownboarding gate)", () => {
    expect(ROUTE_LOADER_PRESENT.test(SOURCE)).toBe(true);
    expect(REQUIRE_OWNBOARDING_PRESENT.test(SOURCE)).toBe(true);
  });
});
