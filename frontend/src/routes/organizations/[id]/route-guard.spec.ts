/**
 * Structural wiring spec for `routes/organizations/[id]/index.tsx`.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   S-PR-023 — anonymous → SignInRequiredCard, no OrganizationReadback.
 *
 * Same rationale as `routes/organizations/route-guard.spec.ts`:
 * the route uses `routeLoader$` (server-side), so vitest cannot
 * exercise the auth branch directly. We assert the wiring is in
 * place; the auth branch is covered by Playwright e2e.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const here = fileURLToPath(import.meta.url);
const routePath = here.replace(
  /\[id\]\/route-guard\.spec\.ts$/,
  "[id]/index.tsx",
);

describe("[routes/organizations/[id]] protected-route wiring", () => {
  it("imports requireSession and SignInRequiredCard (R-PR-003)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(/from\s+["']~\/lib\/require-session["']/);
    expect(source).toMatch(
      /from\s+["']~\/components\/sign-in-required-card\/sign-in-required-card["']/,
    );
  });

  it("calls requireSession BEFORE the loader (anon avoids DB hit)", () => {
    const source = readFileSync(routePath, "utf8");
    const idxRequire = source.indexOf("requireSession(");
    const idxLoader = source.indexOf("useOrganizationLoader()");
    expect(idxRequire).toBeGreaterThan(-1);
    expect(idxLoader).toBeGreaterThan(-1);
    expect(idxRequire).toBeLessThan(idxLoader);
  });

  it("uses useLocation for the redirectTo so per-org path round-trips", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain("useLocation()");
    expect(source).toContain("loc.url.pathname");
  });

  it("branches on `kind === \"anon\"` and renders SignInRequiredCard on the anon path", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain('kind === "anon"');
    expect(source).toContain("<SignInRequiredCard");
  });
});