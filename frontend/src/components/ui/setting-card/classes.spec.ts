/**
 * Pure-function unit tests for `setting-card/classes.ts`.
 *
 * Reference: `sdd/settings-app-grid/{spec,design}.md` (engram).
 *   - REQ-5 (consumer `class` appended after system tokens)
 *   - REQ-6 (focus-visible ring with offset)
 *   - REQ-7 (hover affordance respects monochrome rule)
 *
 * Why this spec has no DOM:
 *   The className contract is the source of truth for the visual
 *   affordances. If `cursor-pointer`, `focus-visible:ring-indigo-500`,
 *   or `transition-[background-color,...]` drops out, this spec fails
 *   before any component render and before any visual review.
 *
 * Companion: `setting-card.spec.tsx` covers the Qwik component layer
 * (polymorphism, icon container, label, slot, props forwarding).
 */
import { describe, it, expect } from "vitest";
import {
  CARD_BASE,
  FOCUS,
  ICON_CONTAINER,
  ICON_SIZE,
  LABEL,
  settingCardClassName,
} from "./classes";

describe("setting-card/classes", () => {
  describe("CARD_BASE (shared vertical tile chrome)", () => {
    it("includes group + flex + flex-col + items-center + no-underline", () => {
      // The parent carries `group` so the icon container's
      // `group-hover:text-slate-900` fires when the tile is hovered.
      expect(CARD_BASE).toContain("group");
      expect(CARD_BASE).toContain("flex");
      expect(CARD_BASE).toContain("flex-col");
      expect(CARD_BASE).toContain("items-center");
      // Defensive against Tailwind 4 preflight — anchors must not
      // underline even when dropped into a parent with underline.
      expect(CARD_BASE).toContain("no-underline");
    });
  });

  describe("ICON_CONTAINER (64×64 slate-100 rounded-xl icon well)", () => {
    it("includes h-16 w-16 rounded-xl bg-slate-100 text-slate-700", () => {
      expect(ICON_CONTAINER).toContain("h-16");
      expect(ICON_CONTAINER).toContain("w-16");
      expect(ICON_CONTAINER).toContain("rounded-xl");
      expect(ICON_CONTAINER).toContain("bg-slate-100");
      expect(ICON_CONTAINER).toContain("text-slate-700");
    });
  });

  describe("ICON_SIZE (consumer SVG size hint)", () => {
    it("is h-12 w-12 (48×48 — fits the 64×64 container with 8px gutter)", () => {
      expect(ICON_SIZE).toBe("h-12 w-12");
    });
  });

  describe("LABEL (text below the icon well)", () => {
    it("includes mt-3 text-sm font-medium text-slate-900 text-center", () => {
      expect(LABEL).toContain("mt-3");
      expect(LABEL).toContain("text-sm");
      expect(LABEL).toContain("font-medium");
      expect(LABEL).toContain("text-slate-900");
      expect(LABEL).toContain("text-center");
    });
  });

  describe("FOCUS (visible focus ring with offset)", () => {
    it("includes focus-visible:ring-2 + indigo + offset-2", () => {
      expect(FOCUS).toContain("focus-visible:ring-2");
      expect(FOCUS).toContain("focus-visible:ring-indigo-500");
      // The offset is the divergence vs Button's link variant — the
      // 2px gap is critical on a square shape so the ring does not
      // touch the inner label/icon.
      expect(FOCUS).toContain("focus-visible:ring-offset-2");
    });
  });

  describe("hover/press affordances (monochrome rule)", () => {
    it("CARD_BASE includes transition + active:translate-y-px (hover lift)", () => {
      // The tile lifts on press; transitions animate background, box-shadow,
      // and transform so hover and active are smooth.
      expect(CARD_BASE).toMatch(/transition-\[/);
      expect(CARD_BASE).toContain("active:translate-y-px");
    });

    it("CARD_BASE does NOT introduce a colored hover state (monochrome rule)", () => {
      // Drift regression guard: a future contributor must not add
      // `hover:bg-indigo-*` or `hover:bg-emerald-*` to the tile —
      // the monochrome rule (init §Design system conventions) forbids
      // tinted hover backgrounds without an ADR.
      expect(CARD_BASE).not.toMatch(/hover:bg-indigo/);
      expect(CARD_BASE).not.toMatch(/hover:bg-emerald/);
      expect(CARD_BASE).not.toMatch(/hover:bg-blue/);
    });
  });

  describe("settingCardClassName() composer (variant is no-op in v1)", () => {
    it("returns CARD_BASE alone when as='a' and no consumer class", () => {
      expect(settingCardClassName("a")).toBe(CARD_BASE);
    });

    it("returns CARD_BASE alone when as='button' and no consumer class", () => {
      // v1: variant is informational; both render the same tokens.
      expect(settingCardClassName("button")).toBe(CARD_BASE);
    });

    it("appends consumer class LAST when as='a'", () => {
      const cls = settingCardClassName("a", "!bg-pink-500");
      expect(cls).toContain(CARD_BASE);
      expect(cls.endsWith("!bg-pink-500")).toBe(true);
      // Consumer class position: after CARD_BASE, not before.
      expect(cls.indexOf("!bg-pink-500")).toBeGreaterThan(
        cls.indexOf(CARD_BASE),
      );
    });

    it("appends consumer class LAST when as='button'", () => {
      const cls = settingCardClassName("button", "!bg-pink-500");
      expect(cls).toContain(CARD_BASE);
      expect(cls.endsWith("!bg-pink-500")).toBe(true);
      expect(cls.indexOf("!bg-pink-500")).toBeGreaterThan(
        cls.indexOf(CARD_BASE),
      );
    });

    it("consumer class WITHOUT !important does NOT erase the system focus ring", () => {
      // SCN-5.2: a plain `bg-pink-500` consumer override must not
      // strip the focus-visible tokens from the rendered className.
      // (Tailwind 4 emission order, not class attribute order, decides
      // cascade — but the system tokens must be PRESENT in the string
      // so the cascade can apply them. Use `!important` to override.)
      const cls = settingCardClassName("a", "bg-pink-500");
      expect(cls).toContain("focus-visible:ring-2");
      expect(cls).toContain("focus-visible:ring-indigo-500");
    });
  });
});
