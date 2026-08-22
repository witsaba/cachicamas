/**
 * Structural wiring spec for `routes/chat/layout.tsx`.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-3.
 *   S-3.a — anon → requireAuthRedirect throws (302)
 *   S-3.b — onboarding-incomplete → requireOwnboarding redirects
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md
 *   §3 — "Auth gate (same chain as routes/home/index.tsx:39-51)."
 *
 * Same rationale as `routes/home/route-guard.spec.ts`: vitest's
 * createDOM() harness does NOT boot a Qwik City request context,
 * so the `onRequest` middleware (requireAuthRedirect's
 * event.redirect throw + requireOwnboarding's async fetch) cannot
 * be exercised directly. We assert the wiring is in place; the
 * runtime behavior is the same chain that protects /home and is
 * already covered by that route's structural + integration specs.
 *
 * Per openspec/AGENTS.md "Strict TDD is on" + REQ-7 (per-file spec
 * discipline), every `.ts`/`.tsx` file in this change has a
 * colocated spec. The route's `index.tsx` is covered by
 * `index.spec.tsx` (render + offline-error rendering); this file
 * covers the guard chain's structural correctness.
 */
import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const here = fileURLToPath(import.meta.url);
const layoutPath = here.replace(/\/route-guard\.spec\.ts$/, "/layout.tsx");

describe("[routes/chat] protected-route wiring (REQ-3 / S-3.a, S-3.b)", () => {
  it("imports the three helpers in the canonical chain (REQ-3)", () => {
    const source = readFileSync(layoutPath, "utf8");
    expect(source).toMatch(/from\s+["']~\/lib\/require-auth-redirect["']/);
    expect(source).toMatch(/from\s+["']~\/lib\/require-ownboarding["']/);
    expect(source).toMatch(/from\s+["']~\/lib\/ssr-cookie-context["']/);
  });

  it("captures the inbound cookie BEFORE the guards run (SSR forwarding)", () => {
    const source = readFileSync(layoutPath, "utf8");
    expect(source).toContain(
      'setSsrCookieHeader(event.request.headers.get("cookie") ?? "")',
    );
    // setSsrCookieHeader must precede requireAuthRedirect in the
    // chain (captured before the redirect throw short-circuits).
    // Read the body of `onRequest`, not the whole file: the import block is
    // alphabetical and says nothing about the order things run in.
    const body = source.split("export const onRequest")[1] ?? "";
    const cookieIdx = body.indexOf("setSsrCookieHeader");
    const authIdx = body.indexOf("requireAuthRedirect");
    expect(cookieIdx).toBeGreaterThanOrEqual(0);
    expect(authIdx).toBeGreaterThan(cookieIdx);
  });

  it("calls requireAuthRedirect synchronously (REQ-3 S-3.a — anon 302)", () => {
    const source = readFileSync(layoutPath, "utf8");
    expect(source).toContain("requireAuthRedirect(event)");
  });

  it("awaits requireOwnboarding (REQ-3 S-3.b — async ownboarding redirect)", () => {
    const source = readFileSync(layoutPath, "utf8");
    expect(source).toMatch(/await\s+requireOwnboarding\s*\(\s*event\s*\)/);
  });

  it("exports onRequest as a RequestHandler (the SSR hook Qwik City reads)", () => {
    const source = readFileSync(layoutPath, "utf8");
    expect(source).toMatch(/export\s+const\s+onRequest\b/);
    expect(source).toMatch(/onRequest:\s*RequestHandler/);
  });
});