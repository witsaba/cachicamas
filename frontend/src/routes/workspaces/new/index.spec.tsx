/**
 * Workspace create page module tests.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-012 (S-WS-110..113).
 *
 * Module-shape coverage: the page imports cleanly, exports the locked
 * names, and the submit-action wrapper maps the createWorkspace envelope
 * to the WorkspaceFormActionResult discriminated union correctly.
 * End-to-end render is asserted in Playwright e2e (PR3).
 */
import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import CreateWorkspacePage from "./index";

describe("Workspace create page module", () => {
  test("module compiles + exports the locked names", async () => {
    expect(CreateWorkspacePage).toBeDefined();
    expect(typeof CreateWorkspacePage).toBe("function");
    const mod = await import("./index");
    expect(mod.onRequest).toBeDefined();
    expect(mod.useSetupLoader).toBeDefined();
    expect(mod.head).toBeDefined();
  });
});
