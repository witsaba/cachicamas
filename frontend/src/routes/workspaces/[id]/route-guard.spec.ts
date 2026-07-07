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
    const onRequestBlock = SOURCE.match(/export const onRequest[\s\S]*?\n\};/);
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

  // 2026-07-07 regression: the detail page's useTask$ fetches the
  // workspace via getWorkspaceSSR (the SSR-aware variant that uses
  // serverAwareFetch → ssrFetch on Node SSR, forwarding the inbound
  // Cookie header). The detail page MUST use getWorkspaceSSR, not
  // getWorkspace — the latter is the browser-only variant that uses
  // raw fetch(${apiBaseUrl()}/...) which, on Node SSR, hits the
  // backend DIRECTLY without the cookie and 401s. The pre-fix
  // detail page imported getWorkspace and so rendered
  // "authentication required" even after the route's onRequest
  // captured the cookie (the cookie was captured into the module
  // context but no one was reading it). Pin the import + the
  // call site so a future refactor cannot reintroduce the bug.
  test("imports getWorkspaceSSR (not getWorkspace) — SSR cookie forwarding in the fetcher", () => {
    const importGetWorkspaceSSR = new RegExp(
      "import\\s*\\{[^}]*\\bgetWorkspaceSSR\\b[^}]*\\}\\s*from\\s*['\"]~/lib/api['\"]",
    );
    expect(SOURCE).toMatch(importGetWorkspaceSSR);
    // The non-SSR variant MUST NOT be imported alongside — that would
    // be a code smell indicating someone is mixing the two. We allow
    // it in the file only if both are imported AND useTask$ uses the
    // SSR one (the next test pins the call site).
    const importBlock =
      SOURCE.match(
        new RegExp(
          "import\\\\s*\\\\{[\\\\s\\\\S]*?\\\\}\\\\s*from\\\\s*['\"]~/lib/api['\"]",
        ),
      )?.[0] ?? "";
    const importsGetWorkspace = new RegExp(
      "\\bgetWorkspace\\b\\s*(?:,|\\})",
    ).test(importBlock);
    if (importsGetWorkspace) {
      // If both are imported, the useTask$ call MUST reference
      // getWorkspaceSSR (the next test will fail if it doesn't).
      const usesNonSSR =
        new RegExp("\\bgetWorkspace\\s*\\(\\s*id\\s*\\)").test(SOURCE) &&
        !new RegExp("\\bgetWorkspaceSSR\\s*\\(\\s*id\\s*\\)").test(SOURCE);
      expect(usesNonSSR).toBe(false);
    }
  });

  test("useTask$ calls getWorkspaceSSR(id) — the SSR-aware fetcher", () => {
    const callSite = new RegExp("\\bgetWorkspaceSSR\\s*\\(\\s*id\\s*\\)");
    expect(SOURCE).toMatch(callSite);
  });
});
