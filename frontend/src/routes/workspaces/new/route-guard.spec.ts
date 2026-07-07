/**
 * Structural wiring spec for `routes/workspaces/new/index.tsx`.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-012 (S-WS-110..113) — workspace create form route.
 *
 * R-PR-003 pattern: the route uses routeLoader$ + requireAuthRedirect +
 * requireOwnboarding (server-context), so vitest cannot exercise the
 * branch directly. We assert the wiring is in place; behavioural
 * coverage lives in index.spec.tsx via mocked useSession.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const here = fileURLToPath(import.meta.url);
const routePath = here.replace(/\/route-guard\.spec\.ts$/, "/index.tsx");

describe("[routes/workspaces/new] protected-route wiring", () => {
  it("imports requireAuthRedirect and exports it as onRequest (R-WS-012 / S-WS-110)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(/import\s*\{\s*requireAuthRedirect\s*\}/);
    expect(source).toMatch(
      /export\s*\{\s*requireAuthRedirect\s+as\s+onRequest\s*\}/,
    );
  });

  it("uses routeLoader$ + requireOwnboarding for the setup-state gate (R-WS-012 / S-WS-110)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(/requireOwnboarding/);
    expect(source).toMatch(/routeLoader\$/);
  });

  it("renders the WorkspaceForm component (R-WS-012 / S-WS-111)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(/WorkspaceForm/);
  });
});
