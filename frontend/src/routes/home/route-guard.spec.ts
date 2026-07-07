/**
 * Structural wiring spec for `routes/home/index.tsx`.
 *
 * Reference: `openspec/changes/home-page-placeholder/specs/home-page/spec.md`
 *   R-HP-001 (S-HP-001..S-HP-004) — personalised greeting for authed users.
 *   R-HP-002 (S-HP-010..S-HP-012) — single paragraph placeholder, no imagery.
 *   R-HP-003 (S-HP-020..S-HP-023) — anonymous renders SignInRequiredCard.
 *   R-HP-008 (S-HP-071) — this file follows R-PR-003.
 *
 * Same rationale as `routes/profile/route-guard.spec.ts`: the route uses
 * `routeLoader$` + `useSession()` (request-context), so vitest cannot
 * exercise the branch directly. We assert the wiring is in place; the
 * auth-aware render is exercised by `routes/home/index.spec.tsx` via
 * `vi.mock("~/routes/plugin@auth")`.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
    
const here = fileURLToPath(import.meta.url);
const routePath = here.replace(/\/route-guard\.spec\.ts$/, "/index.tsx");
    
describe("[routes/home] protected-route wiring", () => {
  it("imports requireSession and SignInRequiredCard (R-HP-008 / S-HP-071)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(/from\s+["']~\/lib\/require-session["']/);
    expect(source).toMatch(
      /from\s+["']~\/components\/sign-in-required-card\/sign-in-required-card["']/,
    );
  });
    
  it("reads useSession, dispatches the anon branch with /home pathname (R-HP-003 / S-HP-020)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain("useSession()");
    expect(source).toContain('requireSession(sessionSig.value, "/home")');
    expect(source).toContain('kind === "anon"');
    expect(source).toContain("<SignInRequiredCard");
    expect(source).toContain('description="Sign in to view your home."');
    expect(source).toContain("redirectTo={guard.pathname}");
  });
    
  it("renders the personalised greeting heading on the auth branch (R-HP-001 / S-HP-001)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain('data-testid="home-heading"');
    expect(source).toContain("Welcome, ");
    // The fallback heading for empty/null name claims.
    expect(source).toContain("Welcome");
  });
    
it("renders the placeholder paragraph on the auth branch (R-HP-002 / S-HP-010)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain('data-testid="home-paragraph"');
  });
    
  // R-OW-007 / S-OW-063 — ownboarding helper wired in.
  it("imports requireOwnboarding (R-OW-007 / S-OW-063)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toMatch(
/from\s+["']~\/lib\/require-ownboarding["']/,
    );
  });
    
  // R-OW-007 / S-OW-064 amended (S-WS-AUTH-CHAIN-SSR-001):
  //   The ownboarding guard moved out of a routeLoader$ and into the
  //   onRequest middleware so the SSR cookie context can be captured
  //   first. The route still must call `requireOwnboarding(event)`
  //   for the guard to fire — we assert the call site instead of
  //   the old routeLoader$ shape.
  it("calls requireOwnboarding in the onRequest middleware chain (R-OW-007 / S-OW-064 amended)", () => {
    const source = readFileSync(routePath, "utf8");
    expect(source).toContain("onRequest");
    expect(source).toContain("requireOwnboarding(event)");
    // Captures the inbound cookie before the guards run so SSR
    // fetches in useTask$ can re-attach it.
    expect(source).toContain("withSsrCookieContext(event");
  });
});
