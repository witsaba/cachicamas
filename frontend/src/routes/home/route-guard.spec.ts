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
    const cookieAt = source.indexOf("setSsrCookieHeader(");
    const authAt = source.indexOf("requireAuthRedirect(event)");
    const onboardAt = source.indexOf("requireOwnboarding(event)");
    expect(cookieAt).toBeGreaterThan(-1);
    expect(authAt).toBeGreaterThan(cookieAt);
    expect(onboardAt).toBeGreaterThan(authAt);
  });

  it("awaits requireOwnboarding so its redirect propagates as a redirect", () => {
    // It is async; a bare call would surface as a fatal server error instead
    // of the 302 to /ownboarding.
    expect(source).toContain("await requireOwnboarding(event)");
  });

  it("owns the guard and the session, and nothing else", () => {
    // The board is `<DeskBoard>`. Keeping the route this thin is what lets the
    // board be looked at without an authenticated session — and it is why the
    // register's data has exactly one home rather than a copy per route.
    expect(source).toContain("<DeskBoard");
    expect(source).not.toContain("ARCHETYPES");
    expect(source).not.toContain("RUNTIME");
    expect(source).not.toContain("<Panel");
  });

  it("reads the register from the mock module in the board it mounts", () => {
    const boardPath = routePath.replace(
      /routes\/home\/index\.tsx$/,
      "components/os/desk-board/desk-board.tsx",
    );
    const board = readFileSync(boardPath, "utf8");
    expect(board).toMatch(/from\s+["']~\/lib\/mock\/registry["']/);
    expect(board).toContain("ARCHETYPES");
    expect(board).toContain("RUNTIME");
  });
});
