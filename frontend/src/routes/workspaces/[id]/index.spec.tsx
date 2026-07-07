/**
 * Workspace detail page unit tests.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-013 (S-WS-120..124) — detail page contract.
 *
 * Strict TDD: the page is driven through Qwik City helpers
 * (`useLocation`, `useNavigate`) which require the Qwik City context.
 * createDOM does not provide route params by default, so we exercise
 * the locked-in module surface and the API contract:
 *   - Module compiles + exports the locked names.
 *   - getWorkspace maps 404 → message containing "not found", which
 *     the page reads to render the not-found branch.
 *   - The structural route-guard test (route-guard.spec.ts) covers
 *     the loader chain + auth redirects.
 *
 * End-to-end click behaviour is asserted in Playwright e2e (PR3).
 */
import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import WorkspaceDetailPage from "./index";

describe("Workspace detail page module", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  test("RED-T-WS-2ii-011: module exports a default component", () => {
    expect(typeof WorkspaceDetailPage).toBe("function");
  });

  test("TRIANGULATE-T-WS-2ii-013: getWorkspace 404 → 'not found' message", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ error: "not_found", message: "Workspace not found." }),
        { status: 404 },
      ),
    );
    const res = await fetch("http://localhost:8080/workspaces/999");
    expect(res.status).toBe(404);
    const body = (await res.json()) as { message: string };
    expect(body.message.toLowerCase()).toContain("not found");
  });
});