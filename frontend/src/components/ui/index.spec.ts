/**
 * Barrel smoke test for `frontend/src/components/ui/index.ts`.
 *
 * Reference: `sdd/settings-app-grid/{spec,design}.md` (engram).
 *   - REQ-8 (barrel re-exports SettingCard, Button, MenuItem)
 *
 * Why this exists:
 *   The README documents `import { Button, MenuItem } from
 *   "~/components/ui"` (line 188) but the barrel file does not yet
 *   exist in the codebase. This spec pins the contract:
 *   the barrel must re-export each primitive by its named identity.
 *
 * RED state: with `index.ts` missing, every import below fails to
 * resolve and the suite fails. GREEN once the barrel exists and
 * each export is the named component (no rename, no default export).
 */
import { describe, it, expect } from "vitest";
import { Button, MenuItem, SettingCard } from "./index";

describe("components/ui barrel", () => {
  it("re-exports SettingCard as a named component", () => {
    expect(SettingCard).toBeDefined();
    expect(typeof SettingCard).toBe("function");
    // SettingCard is a Qwik component$ — its function body is the
    // render closure. The named identity check is the contract.
  });

  it("re-exports Button as a named component", () => {
    expect(Button).toBeDefined();
    expect(typeof Button).toBe("function");
  });

  it("re-exports MenuItem as a named component", () => {
    expect(MenuItem).toBeDefined();
    expect(typeof MenuItem).toBe("function");
  });
});
