/**
 * Structural wiring spec for `routes/organizations/index.tsx`.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   S-PR-023 — anonymous → SignInRequiredCard, no OrganizationList.
 *
 * Why this spec is structural, not behavioral:
 *   The project's testing convention (see the existing
 *   `routes/organizations/index.spec.tsx`) tests the presentational
 *   component, not the route module. The reason is that the route
 *   uses `routeLoader$`, which requires a Qwik City request context
 *   that vitest's `createDOM()` does not provide. We exercise the
 *   auth branch end-to-end in the Playwright suite (anon visit to
 *   /organizations shows the card).
 *
 *   This structural spec asserts the wiring: the route module imports
 *   and uses `requireSession` + `SignInRequiredCard`, the guard sits
 *   BEFORE the loader call, and the discrimination is `kind === "anon"`.
 *   If anyone removes the guard, this fails loudly.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const here = fileURLToPath(import.meta.url);
const routePath = here.replace(
  /\/route-guard\.spec\.ts$/,
  "/index.tsx",
);

describe("[routes/organizations] protected-route wiring", () => {
  it("imports requireSession and SignInRequiredCard (R-PR-003)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(/from\s+["']~\/lib\/require-session["']/);
    expect(source).toMatch(
      /from\s+["']~\/components\/sign-in-required-card\/sign-in-required-card["']/,
    );
  });

  it("calls requireSession BEFORE the loader so anon avoids the DB (design §10)", () => {
    const source = readFileSync(routePath, "utf8");
    const idxRequire = source.indexOf("requireSession(");
    const idxLoader = source.indexOf("useOrganizationsLoader()");
    expect(idxRequire).toBeGreaterThan(-1);
    expect(idxLoader).toBeGreaterThan(-1);
    expect(idxRequire).toBeLessThan(idxLoader);
  });

  it("branches on `kind === \"anon\"` and renders SignInRequiredCard on the anon path", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain('kind === "anon"');
    expect(source).toContain("<SignInRequiredCard");
  });
});