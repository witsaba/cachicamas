/**
 * Structural wiring spec for `routes/home/index.tsx`.
 *
 * The route uses `onRequest` guards that vitest cannot exercise without a
 * Qwik City request context, so the chain is asserted at the source level.
 * The order matters and is the reason this file exists: the SSR cookie must
 * be captured BEFORE either guard throws, or every downstream fetch in the
 * request loses its credentials and the backend answers 401.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const here = fileURLToPath(import.meta.url);
const routePath = here.replace(/\/route-guard\.spec\.ts$/, "/index.tsx");
const source = readFileSync(routePath, "utf8");
// The `onRequest` chain moved to the section layout when the workspace shell
// was introduced, so every screen under it inherits one copy rather than each
// route repeating it. The order assertions below read that file, and they
// read only the body of `onRequest` — the import block is alphabetical and
// says nothing about execution order.
const layoutPath = here.replace(/\/route-guard\.spec\.ts$/, "/layout.tsx");
const guardBody = readFileSync(layoutPath, "utf8").split("export const onRequest")[1] ?? "";

describe("[routes/home] protected-route wiring", () => {
  it("imports requireSession and SignInRequiredCard", () => {
    expect(source).toMatch(/from\s+["']~\/lib\/require-session["']/);
    expect(source).toMatch(
      /from\s+["']~\/components\/sign-in-required-card\/sign-in-required-card["']/,
    );
  });

  it("dispatches the anon branch with the /home pathname", () => {
    expect(source).toContain("useSession()");
    expect(source).toContain('requireSession(sessionSig.value, "/home")');
    expect(source).toContain('kind === "anon"');
    expect(source).toContain("<SignInRequiredCard");
    expect(source).toContain("redirectTo={guard.pathname}");
  });

  it("captures the SSR cookie BEFORE either guard can throw", () => {
    const cookieAt = guardBody.indexOf("setSsrCookieHeader(");
    const authAt = guardBody.indexOf("requireAuthRedirect(event)");
    const onboardAt = guardBody.indexOf("requireOwnboarding(event)");
    expect(cookieAt).toBeGreaterThan(-1);
    expect(authAt).toBeGreaterThan(cookieAt);
    expect(onboardAt).toBeGreaterThan(authAt);
  });

  it("awaits requireOwnboarding so its redirect propagates as a redirect", () => {
    // It is async; a bare call would surface as a fatal server error instead
    // of the 302 to /ownboarding.
    expect(guardBody).toContain("await requireOwnboarding(event)");
  });

  it("owns the guard and the session, and nothing else", () => {
    // The screen is `<FrontDesk>`. Keeping the route this thin is what lets
    // the screen be looked at, and tested, without an authenticated session —
    // and it is why the staff data has exactly one home rather than a copy
    // per route.
    expect(source).toContain("<FrontDesk");
    expect(source).not.toContain("AGENTS");
    expect(source).not.toContain("TEAMS");
    expect(source).not.toContain("CONVERSATIONS");
  });

  it("reads the staff from the mock module in the screen it mounts", () => {
    const screenPath = routePath.replace(
      /routes\/home\/index\.tsx$/,
      "components/workspace/screens/front-desk.tsx",
    );
    const screenSource = readFileSync(screenPath, "utf8");
    expect(screenSource).toMatch(/from\s+["']~\/lib\/mock\/staff["']/);
    expect(screenSource).toContain("AGENTS");
    expect(screenSource).toContain("TEAMS");
  });
});
