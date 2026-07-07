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
const REQUIRE_AUTH_REDIRECT_REEXPORT = new RegExp(
  "export\\s*\\{\\s*requireAuthRedirect\\s+as\\s+onRequest\\s*\\}",
);
const ROUTE_LOADER_PRESENT = new RegExp("routeLoader\\$");
const REQUIRE_OWNBOARDING_PRESENT = new RegExp("requireOwnboarding");

describe("workspaces/:id — route guard structural wiring (R-PR-003)", () => {
  test("imports requireAuthRedirect", () => {
    expect(REQUIRE_AUTH_REDIRECT_IMPORT.test(SOURCE)).toBe(true);
  });

  test("re-exports requireAuthRedirect as onRequest (authed-only gate)", () => {
    expect(REQUIRE_AUTH_REDIRECT_REEXPORT.test(SOURCE)).toBe(true);
  });

  test("uses routeLoader$ to call requireOwnboarding (ownboarding gate)", () => {
    expect(ROUTE_LOADER_PRESENT.test(SOURCE)).toBe(true);
    expect(REQUIRE_OWNBOARDING_PRESENT.test(SOURCE)).toBe(true);
  });
});
