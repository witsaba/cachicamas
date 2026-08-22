/**
 * Behavioural spec for `routes/archetypes/[slug]/index.tsx`.
 *
 * This is the screen for a specialist that does not exist, and the reason it
 * is a whole screen rather than a disabled tile is that five of the six
 * registered archetypes are in this state. The assertions are all about it
 * saying so: an honest blocked reason, the cited authority, and no route into
 * a capability that is not there.
 *
 * The route uses `routeLoader$`, which needs a Qwik City request context that
 * `createDOM()` does not provide, so the loader's own behaviour is asserted
 * through the resolver it delegates to and the rest at the source level.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { ARCHETYPES, archetypeBySlug } from "~/lib/mock/registry";

const here = fileURLToPath(import.meta.url);
const source = readFileSync(here.replace(/\.spec\.tsx$/, ".tsx"), "utf8");
// The screen itself is `<ArchetypePanel>`; the route owns only the guard chain
// and the loader. Its claims are asserted against the component that makes
// them, not against the route that mounts it.
const panel = readFileSync(
  here.replace(
    /routes\/archetypes\/\[slug\]\/index\.spec\.tsx$/,
    "components/os/archetype-panel/archetype-panel.tsx",
  ),
  "utf8",
);

describe("routes/archetypes/[slug]", () => {
  it("captures the SSR cookie before either guard can throw", () => {
    const cookieAt = source.indexOf("setSsrCookieHeader(");
    const authAt = source.indexOf("requireAuthRedirect(event)");
    const onboardAt = source.indexOf("await requireOwnboarding(event)");
    expect(cookieAt).toBeGreaterThan(-1);
    expect(authAt).toBeGreaterThan(cookieAt);
    expect(onboardAt).toBeGreaterThan(authAt);
  });

  it("resolves its subject from the register, not from a second copy of it", () => {
    expect(source).toMatch(/from\s+["']~\/lib\/mock\/registry["']/);
    expect(source).toContain("archetypeBySlug(");
  });

  it("keeps the route thin: the screen is a component, not inline JSX", () => {
    expect(source).toContain("<ArchetypePanel");
    expect(source).not.toContain("<Panel");
  });

  it("has a real subject for every slug the register links to", () => {
    for (const a of ARCHETYPES) {
      if (a.slug === "chat") continue; // chat has its own application
      expect(archetypeBySlug(a.slug), a.slug).toBeTruthy();
    }
  });

  it("handles an unknown slug rather than rendering a blank screen", () => {
    expect(archetypeBySlug("nope")).toBeUndefined();
    expect(panel).toContain("Not on the register");
  });

  it("renders the honest blocked reason rather than a 'coming soon'", () => {
    expect(panel).toContain("{a.blockedBy}");
    expect(panel.toLowerCase()).not.toContain("coming soon");
    expect(panel.toLowerCase()).not.toContain("stay tuned");
  });

  it("cites the decision record and the plan, or says there is none", () => {
    expect(panel).toContain("{a.authority}");
    expect(panel).toContain("No milestone document yet");
  });

  it("teaches the command that reaches this screen", () => {
    expect(panel).toContain("{a.code}");
    expect(panel).toContain("{a.fkey}");
  });

  it("states the ownership rule every archetype is bound by", () => {
    // ADR 0009 § D6. It is the most consequential thing about a specialist
    // that has not been built, so it is on the screen before it exists.
    expect(panel).toContain("owns its own tables");
    expect(panel).toContain("Database Administrator");
  });

  it("offers a way back and a way to the one that is being built", () => {
    expect(panel).toContain('href="/home/"');
    expect(panel).toContain('href="/chat/"');
  });
});
