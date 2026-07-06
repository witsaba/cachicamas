/**
 * Structural wiring spec for `routes/ownboarding/index.tsx`.
 *
 * Reference: `openspec/changes/2026-07-06-ownboarding/specs/ownboarding/spec.md`
 *   R-OW-001 (S-OW-001..004) — authed-only with native redirect.
 *
 * R-PR-003 pattern: the route uses `routeLoader$` + `requireAuthRedirect`
 * (server-context), so vitest cannot exercise the branch directly. We
 * assert the wiring is in place; the behavioural coverage lives in
 * `index.spec.tsx` via mocked useSession.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const here = fileURLToPath(import.meta.url);
const routePath = here.replace(/\/route-guard\.spec\.ts$/, "/index.tsx");

describe("[routes/ownboarding] protected-route wiring", () => {
  it("imports requireAuthRedirect and exports it as onRequest (R-OW-001 / S-OW-003)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(/from\s+["']~\/lib\/require-auth-redirect["']/);
    expect(source).toMatch(
      /export\s*\{\s*requireAuthRedirect\s+as\s+onRequest\s*\}/,
    );
  });

  it("does NOT render an inline SignInRequiredCard (R-OW-001 / S-OW-004, ADR-0011)", () => {
    const source = readFileSync(routePath, "utf8");
    // Negative assertion: the new pattern uses the native /auth/signin
    // redirect, not the inline card.
    expect(source).not.toMatch(
      /import.*sign-in-required-card.*sign-in-required-card["']/,
    );
    expect(source).not.toContain("<SignInRequiredCard");
  });

  it("imports OwnboardingForm for the authed branch (R-OW-002 / R-OW-003)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(
      /from\s+["']~\/components\/ownboarding-form\/ownboarding-form["']/,
    );
    expect(source).toContain("<OwnboardingForm");
    expect(source).toContain("action={submitAction}");
    expect(source).toContain("onSuccess$=");
  });

  it("the submit action uses createOrganization and navigates to /home on 201 (R-OW-003 / S-OW-020)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(
      /import.*createOrganization.*from\s+["']~\/lib\/api["']/,
    );
    expect(source).toContain('await nav("/home")');
  });
});