/**
 * Pure-function unit tests for `button/classes.ts`.
 *
 * Why this spec has no DOM:
 *   The className contract is the source of truth for the visual
 *   affordances. If `cursor-pointer` or `not-disabled:hover:bg-slate-700`
 *   drops out of a variant, this spec fails before any component render
 *   and before any visual review.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
 *   - R-UB-001 (variant base classes)
 *   - R-UB-002 (cursor affordance)
 *   - R-UB-003 (transition + active + hover)
 *   - R-UB-004 (focus ring)
 *   - R-UB-005 (className pinning contract)
 */
import { describe, it, expect } from "vitest";
import {
  ALL_SIZES,
  ALL_VARIANTS,
  BUTTON_BASE,
  BUTTON_SIZE_MD,
  BUTTON_SIZE_LG,
  VARIANT_DESTRUCTIVE,
  VARIANT_LINK,
  VARIANT_PRIMARY,
  VARIANT_SECONDARY,
  buttonClassName,
} from "./classes";

describe("button/classes", () => {
  describe("BUTTON_BASE (shared affordances for filled variants)", () => {
    it("renders the flexbox + layout primitives", () => {
      expect(BUTTON_BASE).toContain("inline-flex");
      expect(BUTTON_BASE).toContain("items-center");
      expect(BUTTON_BASE).toContain("justify-center");
      expect(BUTTON_BASE).toContain("gap-2");
      expect(BUTTON_BASE).toContain("rounded-md");
      expect(BUTTON_BASE).toContain("font-medium");
    });

    it("pins cursor-pointer and disabled:cursor-not-allowed (R-UB-002)", () => {
      expect(BUTTON_BASE).toContain("cursor-pointer");
      expect(BUTTON_BASE).toContain("disabled:cursor-not-allowed");
    });

    it("pins the transition set (R-UB-003)", () => {
      expect(BUTTON_BASE).toMatch(/transition-\[background-color/);
      expect(BUTTON_BASE).toContain("duration-150");
    });

    it("pins the focus-visible ring setup (R-UB-004)", () => {
      expect(BUTTON_BASE).toContain("focus:outline-none");
      expect(BUTTON_BASE).toContain("focus-visible:ring-2");
    });
  });

  describe("sizes", () => {
    it("md is text-sm with px-4 py-2", () => {
      expect(BUTTON_SIZE_MD).toBe("px-4 py-2 text-sm");
    });

    it("lg is text-base with px-5 py-3 (hero/empty-state)", () => {
      expect(BUTTON_SIZE_LG).toBe("px-5 py-3 text-base");
    });
  });

  describe("VARIANT_PRIMARY (create / update)", () => {
    it("uses bg-slate-900 + text-white (project monochrome)", () => {
      expect(VARIANT_PRIMARY).toContain("bg-slate-900");
      expect(VARIANT_PRIMARY).toContain("text-white");
    });

    it("NOT bg-indigo-600 (drift regression guard)", () => {
      // Ownboarding-form was bg-indigo-600; this spec fails if the
      // drift is reintroduced.
      expect(VARIANT_PRIMARY).not.toContain("bg-indigo-600");
    });

    it("NOT bg-red-700 (destructive color does not leak into primary)", () => {
      expect(VARIANT_PRIMARY).not.toContain("bg-red-700");
    });

    it("has the not-disabled:hover utility (a11y fix)", () => {
      expect(VARIANT_PRIMARY).toContain("not-disabled:hover:bg-slate-700");
      // Bug regression guard: must NOT be the bare hover: utility.
      expect(VARIANT_PRIMARY).not.toMatch(/(^|\s)hover:bg-slate-700(\s|$)/);
    });

    it("has active:translate-y-px press feedback (R-UB-003)", () => {
      expect(VARIANT_PRIMARY).toContain("active:translate-y-px");
    });

    it("has indigo focus ring (R-UB-004)", () => {
      expect(VARIANT_PRIMARY).toContain("focus-visible:ring-indigo-500");
    });

    it("has disabled:opacity-50", () => {
      expect(VARIANT_PRIMARY).toContain("disabled:opacity-50");
    });
  });

  describe("VARIANT_SECONDARY (general-purpose)", () => {
    it("uses bg-white + border-slate-300 + text-slate-900", () => {
      expect(VARIANT_SECONDARY).toContain("bg-white");
      expect(VARIANT_SECONDARY).toContain("border");
      expect(VARIANT_SECONDARY).toContain("border-slate-300");
      expect(VARIANT_SECONDARY).toContain("text-slate-900");
    });

    it("NOT bg-slate-900 (primary color does not leak into secondary)", () => {
      expect(VARIANT_SECONDARY).not.toContain("bg-slate-900");
    });

    it("has the not-disabled:hover:bg-slate-50 utility", () => {
      expect(VARIANT_SECONDARY).toContain("not-disabled:hover:bg-slate-50");
    });

    it("has active:translate-y-px press feedback", () => {
      expect(VARIANT_SECONDARY).toContain("active:translate-y-px");
    });

    it("has indigo focus ring (same as primary for consistency)", () => {
      expect(VARIANT_SECONDARY).toContain("focus-visible:ring-indigo-500");
    });
  });

  describe("VARIANT_DESTRUCTIVE (delete / remove)", () => {
    it("uses bg-red-700 + text-white", () => {
      expect(VARIANT_DESTRUCTIVE).toContain("bg-red-700");
      expect(VARIANT_DESTRUCTIVE).toContain("text-white");
    });

    it("has the not-disabled:hover:bg-red-800 utility", () => {
      expect(VARIANT_DESTRUCTIVE).toContain("not-disabled:hover:bg-red-800");
    });

    it("has NO active:translate-y-px (heavy button does not press)", () => {
      expect(VARIANT_DESTRUCTIVE).not.toContain("active:translate-y-px");
    });

    it("has a red focus ring (R-UB-004)", () => {
      expect(VARIANT_DESTRUCTIVE).toContain("focus-visible:ring-red-500");
    });
  });

  describe("VARIANT_LINK (bare-underline text button)", () => {
    it("uses text-slate-700 + underline + hover:text-slate-900", () => {
      expect(VARIANT_LINK).toContain("text-slate-700");
      expect(VARIANT_LINK).toContain("underline");
      expect(VARIANT_LINK).toContain("hover:text-slate-900");
    });

    it("uses transition-colors (lighter than the filled variants)", () => {
      expect(VARIANT_LINK).toContain("transition-colors");
      expect(VARIANT_LINK).toContain("duration-150");
    });

    it("does NOT use a surface (no bg-*, no border, no rounded)", () => {
      // Bug guard: a future contributor must not add bg-white or
      // rounded-md to the link variant — that would make it look
      // like a secondary button.
      expect(VARIANT_LINK).not.toMatch(/\bbg-/);
      expect(VARIANT_LINK).not.toContain("border");
      expect(VARIANT_LINK).not.toContain("rounded");
    });

    it("has cursor-pointer + disabled tokens (R-UB-002)", () => {
      expect(VARIANT_LINK).toContain("cursor-pointer");
      expect(VARIANT_LINK).toContain("disabled:cursor-not-allowed");
      expect(VARIANT_LINK).toContain("disabled:opacity-50");
    });

    it("has indigo focus ring + active:translate-y-px", () => {
      expect(VARIANT_LINK).toContain("focus-visible:ring-indigo-500");
      expect(VARIANT_LINK).toContain("active:translate-y-px");
    });
  });

  describe("buttonClassName() composer", () => {
    it("primary md composes BASE + size md + variant primary", () => {
      const cls = buttonClassName("primary", "md");
      expect(cls).toContain(BUTTON_BASE);
      expect(cls).toContain("px-4 py-2 text-sm");
      expect(cls).toContain("bg-slate-900");
    });

    it("secondary lg composes BASE + size lg + variant secondary", () => {
      const cls = buttonClassName("secondary", "lg");
      expect(cls).toContain(BUTTON_BASE);
      expect(cls).toContain("px-5 py-3 text-base");
      expect(cls).toContain("bg-white");
      expect(cls).toContain("border-slate-300");
    });

    it("destructive md composes BASE + size md + variant destructive", () => {
      const cls = buttonClassName("destructive", "md");
      expect(cls).toContain(BUTTON_BASE);
      expect(cls).toContain("px-4 py-2 text-sm");
      expect(cls).toContain("bg-red-700");
    });

    it("link md composes variant-link (no BUTTON_BASE, no size)", () => {
      const cls = buttonClassName("link", "md");
      // The link variant is text-only — no padding, no flex.
      expect(cls).not.toContain(BUTTON_BASE);
      expect(cls).not.toContain("px-4 py-2 text-sm");
      expect(cls).toContain("text-slate-700");
      expect(cls).toContain("underline");
    });

    it("consumer class is appended after the system tokens", () => {
      const cls = buttonClassName("primary", "md", "h-10 w-10 rounded-full");
      expect(cls).toContain("bg-slate-900");
      expect(cls).toContain("h-10 w-10 rounded-full");
      // The consumer tokens come AFTER the system tokens in the
      // joined string (Tailwind utility order is irrelevant for
      // specificity, but the contract is documented).
      const consumerIdx = cls.indexOf("h-10 w-10 rounded-full");
      const systemIdx = cls.indexOf("bg-slate-900");
      expect(consumerIdx).toBeGreaterThan(systemIdx);
    });

    it("consumer class on link variant is appended after VARIANT_LINK", () => {
      const cls = buttonClassName("link", "md", "text-sm");
      expect(cls).toContain("text-slate-700");
      expect(cls).toContain("text-sm");
    });

    it("every (variant, size) cell carries cursor-pointer", () => {
      for (const variant of ALL_VARIANTS) {
        for (const size of ALL_SIZES) {
          expect(buttonClassName(variant, size)).toContain("cursor-pointer");
        }
      }
    });

    it("every (variant, size) cell carries focus-visible:ring-2", () => {
      for (const variant of ALL_VARIANTS) {
        for (const size of ALL_SIZES) {
          expect(buttonClassName(variant, size)).toContain(
            "focus-visible:ring-2",
          );
        }
      }
    });

    it("every filled (variant, size) cell carries the transition set", () => {
      // The link variant uses transition-colors (a lighter transition);
      // the filled variants use the multi-property transition set.
      for (const variant of ["primary", "secondary", "destructive"] as const) {
        for (const size of ALL_SIZES) {
          expect(buttonClassName(variant, size)).toMatch(
            /transition-\[background-color/,
          );
        }
      }
    });
  });
});
