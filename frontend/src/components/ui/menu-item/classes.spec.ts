/**
 * Pure-function unit tests for `menu-item/classes.ts`.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
 *   R-UB-006 — MenuItem primitive renders panel-row affordances.
 */
import { describe, it, expect } from "vitest";
import { MENU_ITEM_BASE } from "./classes";

describe("menu-item/classes", () => {
  describe("MENU_ITEM_BASE (panel-row affordances)", () => {
    it("uses block + w-full + text-left (panel-row layout, not button flex)", () => {
      expect(MENU_ITEM_BASE).toContain("block");
      expect(MENU_ITEM_BASE).toContain("w-full");
      expect(MENU_ITEM_BASE).toContain("text-left");
    });

    it("uses tight padding (px-2 py-1.5)", () => {
      expect(MENU_ITEM_BASE).toContain("px-2");
      expect(MENU_ITEM_BASE).toContain("py-1.5");
    });

    it("uses text-sm (panel label size)", () => {
      expect(MENU_ITEM_BASE).toContain("text-sm");
    });

    it("pins cursor-pointer for cross-OS affordance", () => {
      expect(MENU_ITEM_BASE).toContain("cursor-pointer");
      expect(MENU_ITEM_BASE).toContain("disabled:cursor-not-allowed");
    });

    it("pins the transition set (matches Button)", () => {
      expect(MENU_ITEM_BASE).toMatch(/transition-\[background-color/);
      expect(MENU_ITEM_BASE).toContain("duration-150");
    });

    it("uses hover:bg-slate-100 (panel-row tint, NOT button surface swap)", () => {
      expect(MENU_ITEM_BASE).toContain("hover:bg-slate-100");
      // Bug guard: should NOT be the bare `hover:bg-white` or
      // `hover:bg-slate-50` from button variants.
      expect(MENU_ITEM_BASE).not.toContain("hover:bg-slate-50");
      expect(MENU_ITEM_BASE).not.toContain("hover:bg-white");
    });

    it("has the same focus ring color as primary/secondary buttons (indigo-500)", () => {
      expect(MENU_ITEM_BASE).toContain("focus:outline-none");
      expect(MENU_ITEM_BASE).toContain("focus-visible:ring-2");
      expect(MENU_ITEM_BASE).toContain("focus-visible:ring-indigo-500");
    });

    it("has active:translate-y-px press feedback", () => {
      expect(MENU_ITEM_BASE).toContain("active:translate-y-px");
    });

    it("has disabled:opacity-50", () => {
      expect(MENU_ITEM_BASE).toContain("disabled:opacity-50");
    });
  });
});