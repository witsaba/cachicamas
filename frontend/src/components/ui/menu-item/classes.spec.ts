/**
 * MenuItem className table — contract tests.
 *
 * The distinction this guards: a menu item TINTS its row; it does not reverse
 * a cell the way `<Button>` does. If these two ever converge, one of the two
 * primitives has stopped earning its existence.
 */
import { describe, it, expect } from "vitest";
import { MENU_ITEM_BASE } from "./classes";
import { VARIANT_PRIMARY } from "../button/classes";

describe("components/ui/menu-item — className table", () => {
  it("is a full-width, left-aligned row", () => {
    expect(MENU_ITEM_BASE).toContain("flex");
    expect(MENU_ITEM_BASE).toContain("w-full");
    expect(MENU_ITEM_BASE).toContain("text-left");
  });

  it("uses panel-row density, tighter than a button", () => {
    expect(MENU_ITEM_BASE).toContain("px-2");
    expect(MENU_ITEM_BASE).toContain("py-1.5");
    expect(MENU_ITEM_BASE).toContain("text-base");
  });

  it("reads as a row, not a control", () => {
    expect(MENU_ITEM_BASE).toContain("text-left");
    expect(MENU_ITEM_BASE).not.toContain("uppercase");
  });

  it("tints its row on hover rather than reversing a cell", () => {
    expect(MENU_ITEM_BASE).toContain("hover:bg-sunken");
    expect(MENU_ITEM_BASE).toContain("hover:text-ink");
    // The button's reverse-video treatment must not leak in here.
    expect(MENU_ITEM_BASE).not.toContain("bg-amber");
    expect(VARIANT_PRIMARY).not.toContain("hover:bg-sunken");
  });

  it("declares the cursor and disabled affordances", () => {
    expect(MENU_ITEM_BASE).toContain("cursor-pointer");
    expect(MENU_ITEM_BASE).toContain("disabled:cursor-not-allowed");
    expect(MENU_ITEM_BASE).toContain("disabled:opacity-45");
  });

  it("carries no shadow, ring or Tailwind-default palette token", () => {
    expect(MENU_ITEM_BASE).not.toMatch(
      /shadow-|ring-|\b(slate|zinc|gray|indigo)-\d/,
    );
  });

  it("does not move on press", () => {
    expect(MENU_ITEM_BASE).not.toContain("translate");
  });
});
