/**
 * Structural wiring spec for `routes/profile/index.tsx`.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   S-PR-022 — anonymous → SignInRequiredCard on /profile.
 * Reference: `openspec/changes/cachicamas-login-ux/specs/profile-home/spec.md`
 *   S-PH-020 — anonymous /profile shows the card.
 *
 * Same rationale as the other route-guard specs: the route uses
 * `useSession()` (request-context), so vitest cannot exercise the
 * branch directly. We assert the wiring is in place; the auth branch
 * is exercised in the Playwright e2e (`e2e/sign-in-landing.spec.ts`
 * drives the landing sign-in flow, `sign-in-denied.spec.ts` covers
 * the error case).
 *
 * The existing test coverage of `ProfileView` itself (sign-out
 * button, name/email rendering, etc.) lives in
 * `components/profile-view/profile-view.spec.tsx` — untouched.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const here = fileURLToPath(import.meta.url);
const routePath = here.replace(/\/route-guard\.spec\.ts$/, "/index.tsx");

describe("[routes/profile] protected-route wiring", () => {
  it("imports requireSession and SignInRequiredCard (R-PR-003)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(/from\s+["']~\/lib\/require-session["']/);
    expect(source).toMatch(
      /from\s+["']~\/components\/sign-in-required-card\/sign-in-required-card["']/,
    );
  });

  it("reads useSession and dispatches the anon branch (R-PR-003)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain("useSession()");
    expect(source).toContain('kind === "anon"');
    expect(source).toContain("<SignInRequiredCard");
    expect(source).toContain('description="Sign in to view your profile."');
    expect(source).toContain('redirectTo={guard.pathname}');
  });

  it("still renders ProfileView on the auth branch (preserves R-FA-050)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain("<ProfileView");
    expect(source).toContain("session={sessionSig.value}");
  });
});