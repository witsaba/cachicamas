/**
 * Barrel smoke test for `frontend/src/components/ui/index.ts`.
 *
 * The barrel is the public API of the interaction primitives. This pins that
 * each is exported under its own name (no rename, no default export), and that
 * the retired slate-era tile primitive has not been quietly reintroduced.
 */
import { describe, it, expect } from "vitest";
import * as barrel from "./index";
import { Button, MenuItem } from "./index";

describe("components/ui barrel", () => {
  it("re-exports Button as a named component", () => {
    expect(Button).toBeDefined();
    expect(typeof Button).toBe("function");
  });

  it("re-exports MenuItem as a named component", () => {
    expect(MenuItem).toBeDefined();
    expect(typeof MenuItem).toBe("function");
  });

  it("exports exactly the two interaction primitives", () => {
    // Structural furniture belongs to ~/components/os. If a panel, lamp or
    // gauge turns up here, the two families have started to blur.
    expect(Object.keys(barrel).sort()).toEqual(["Button", "MenuItem"]);
  });
});
