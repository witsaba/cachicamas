/**
 * Button className table — contract tests.
 *
 * These exist as an anti-drift guard for the terminal world (`src/global.css`).
 * The three rules the world cannot survive losing are asserted directly:
 * no radius, no shadow, and no colour that is not one of the five working
 * colours. A contributor who reaches for a slate-era token, a pill, or a
 * lifted surface fails here before it reaches a screen.
 */
import { describe, it, expect } from "vitest";
import {
  ALL_SIZES,
  ALL_VARIANTS,
  BUTTON_BASE,
  BUTTON_SIZE_LG,
  BUTTON_SIZE_MD,
  VARIANT_DESTRUCTIVE,
  VARIANT_LINK,
  VARIANT_PRIMARY,
  VARIANT_SECONDARY,
  buttonClassName,
} from "./classes";

const EVERY_TOKEN = [
  BUTTON_BASE,
  BUTTON_SIZE_MD,
  BUTTON_SIZE_LG,
  VARIANT_PRIMARY,
  VARIANT_SECONDARY,
  VARIANT_DESTRUCTIVE,
  VARIANT_LINK,
].join(" ");

describe("components/ui/button — className table", () => {
  describe("the world's three non-negotiables", () => {
    it("emits no radius token anywhere", () => {
      expect(EVERY_TOKEN).not.toMatch(/\brounded/);
    });

    it("emits no shadow, blur or ring token anywhere", () => {
      expect(EVERY_TOKEN).not.toMatch(/\bshadow-/);
      expect(EVERY_TOKEN).not.toMatch(/\bblur/);
      expect(EVERY_TOKEN).not.toMatch(/\bring-/);
    });

    it("uses no colour outside the terminal palette", () => {
      // Every colour utility in the table, whatever its variant prefix.
      const colours = EVERY_TOKEN.match(
        /(?:^|\s|:)(?:bg|text|border)-([a-z-]+)/g,
      )!.map((m) => m.replace(/^.*?(?:bg|text|border)-/, ""));
      const allowed = new Set([
        "void",
        "panel",
        "raise",
        "rule",
        "rule-strong",
        "amber",
        "amber-dim",
        "cyan",
        "live",
        "hold",
        "fail",
        "fg",
        "fg-mid",
        "fg-dim",
        "transparent",
        "label", // text-label is the size scale, not a colour
        "body",
        "left",
      ]);
      for (const c of colours) {
        expect(allowed.has(c), `unexpected colour token: ${c}`).toBe(true);
      }
    });

    it("carries no slate-era token", () => {
      expect(EVERY_TOKEN).not.toMatch(/slate|indigo|white\b|red-\d/);
    });
  });

  describe("base affordances", () => {
    it("is a bordered inline cell", () => {
      expect(BUTTON_BASE).toContain("inline-flex");
      expect(BUTTON_BASE).toContain("items-center");
      expect(BUTTON_BASE).toContain("justify-center");
      expect(BUTTON_BASE).toContain("border");
    });

    it("speaks in the machine voice: mono, uppercase, tracked", () => {
      expect(BUTTON_BASE).toContain("font-system");
      expect(BUTTON_BASE).toContain("uppercase");
      expect(BUTTON_BASE).toContain("tracking-[0.08em]");
    });

    it("declares the cursor and the disabled affordance", () => {
      expect(BUTTON_BASE).toContain("cursor-pointer");
      expect(BUTTON_BASE).toContain("disabled:cursor-not-allowed");
      expect(BUTTON_BASE).toContain("disabled:opacity-40");
    });

    it("transitions colour only, at the product duration", () => {
      expect(BUTTON_BASE).toMatch(/transition-\[background-color,color,border-color\]/);
      expect(BUTTON_BASE).toContain("duration-150");
      // No transform: a terminal key does not move.
      expect(BUTTON_BASE).not.toContain("translate");
    });

    it("pins the two sizes", () => {
      expect(BUTTON_SIZE_MD).toBe("px-3 py-1.5 text-label");
      expect(BUTTON_SIZE_LG).toBe("px-4 py-2.5 text-body");
    });
  });

  describe("variants map intent onto exactly one working colour", () => {
    it("primary is amber, filled, and reverses on hover and press", () => {
      expect(VARIANT_PRIMARY).toContain("bg-amber");
      expect(VARIANT_PRIMARY).toContain("text-void");
      expect(VARIANT_PRIMARY).toContain("not-disabled:hover:bg-void");
      expect(VARIANT_PRIMARY).toContain("not-disabled:hover:text-amber");
      expect(VARIANT_PRIMARY).toContain("not-disabled:active:text-amber");
    });

    it("secondary is an empty ruled cell that warms to amber", () => {
      expect(VARIANT_SECONDARY).toContain("border-rule-strong");
      expect(VARIANT_SECONDARY).toContain("bg-transparent");
      expect(VARIANT_SECONDARY).toContain("text-fg");
      expect(VARIANT_SECONDARY).toContain("not-disabled:hover:border-amber");
      expect(VARIANT_SECONDARY).not.toContain("bg-amber");
    });

    it("destructive is the fail colour and reverses like the others", () => {
      expect(VARIANT_DESTRUCTIVE).toContain("bg-fail");
      expect(VARIANT_DESTRUCTIVE).toContain("text-void");
      expect(VARIANT_DESTRUCTIVE).toContain("not-disabled:hover:text-fail");
    });

    it("link is cyan, underlined, and carries no surface at all", () => {
      expect(VARIANT_LINK).toContain("text-cyan");
      expect(VARIANT_LINK).toContain("underline");
      expect(VARIANT_LINK).toContain("hover:text-fg");
      expect(VARIANT_LINK).not.toMatch(/\bbg-/);
      expect(VARIANT_LINK).not.toContain("border");
      expect(VARIANT_LINK).not.toContain("uppercase");
    });
  });

  describe("buttonClassName composition", () => {
    it("composes base + size + variant for every filled cell", () => {
      for (const variant of ALL_VARIANTS) {
        if (variant === "link") continue;
        for (const size of ALL_SIZES) {
          const cls = buttonClassName(variant, size);
          expect(cls).toContain(BUTTON_BASE);
          expect(cls).toContain(size === "lg" ? BUTTON_SIZE_LG : BUTTON_SIZE_MD);
        }
      }
    });

    it("link ignores base and size entirely", () => {
      expect(buttonClassName("link", "lg")).toBe(VARIANT_LINK);
      expect(buttonClassName("link", "md")).toBe(VARIANT_LINK);
    });

    it("appends the consumer class last so it can override", () => {
      const cls = buttonClassName("primary", "md", "w-full");
      expect(cls.endsWith("w-full")).toBe(true);
    });

    it("appends the consumer class on the link variant too", () => {
      expect(buttonClassName("link", "md", "text-label")).toBe(
        `${VARIANT_LINK} text-label`,
      );
    });
  });
});
