/**
 * Pure-function tests for the button className table.
 *
 * These pin the *world*, not just the component: a contributor who reaches for
 * a gradient, a hard offset shadow, a colour outside the token set, or the
 * category's default indigo, breaks a test here before it ever reaches a
 * screenshot.
 */
import { describe, it, expect } from "vitest";
import {
  ALL_SIZES,
  ALL_VARIANTS,
  BUTTON_BASE,
  BUTTON_SIZE_LG,
  BUTTON_SIZE_MD,
  BUTTON_SIZE_SM,
  VARIANT_DESTRUCTIVE,
  VARIANT_LINK,
  VARIANT_PRIMARY,
  VARIANT_SECONDARY,
  buttonClassName,
} from "./classes";

/** Every token string the table can emit, in one bag. */
const ALL = [
  BUTTON_BASE,
  BUTTON_SIZE_SM,
  BUTTON_SIZE_MD,
  BUTTON_SIZE_LG,
  VARIANT_PRIMARY,
  VARIANT_SECONDARY,
  VARIANT_DESTRUCTIVE,
  VARIANT_LINK,
].join(" ");

/** The colour tokens this product is allowed to name. */
const PALETTE =
  /^(brand|brand-press|brand-tint|ink|ink-mid|ink-soft|ink-inverse|surface|canvas|sunken|deep|line|line-firm|line-control|ok|waiting|stop|idle|transparent|current|dept-[a-z]+)$/;

describe("components/ui/button — className table", () => {
  describe("the world's non-negotiables", () => {
    it("emits no gradient and no hard offset shadow", () => {
      expect(ALL).not.toMatch(/gradient/);
      // The only shadow allowed is the token, which carries an offset and a
      // blur. A zero-blur block shadow is a costume this world never chose.
      expect(ALL).not.toMatch(/shadow-\[\d/);
    });

    it("emits no ring token — focus belongs to global.css alone", () => {
      expect(ALL).not.toMatch(/\bring-/);
      expect(ALL).not.toMatch(/focus-visible:/);
    });

    it("names no colour outside the product's tokens", () => {
      const colours = ALL.match(
        /(?:^|\s|:)(?:bg|text|border|from|via|to)-([a-z0-9-]+)/g,
      );
      expect(colours, "the table names at least one colour").toBeTruthy();
      for (const raw of colours ?? []) {
        const name = raw.replace(/^[\s:]*/, "").split("-").slice(1).join("-");
        // Layout-ish tokens share the `text-`/`border-` prefixes.
        if (/^(2xs|xs|sm|base|md|lg|xl|left|center|right|\[)/.test(name)) continue;
        expect(name, `${raw} is outside the palette`).toMatch(PALETTE);
      }
    });

    it("carries no Tailwind default palette and no slate-era token", () => {
      expect(ALL).not.toMatch(
        /\b(slate|zinc|gray|neutral|stone|indigo|violet|sky|emerald)-\d/,
      );
    });
  });

  describe("base affordances", () => {
    it("is a rounded, bordered inline control", () => {
      expect(BUTTON_BASE).toContain("inline-flex");
      expect(BUTTON_BASE).toContain("items-center");
      expect(BUTTON_BASE).toContain("rounded-md");
      expect(BUTTON_BASE).toContain("border");
    });

    it("declares the cursor and the disabled affordance", () => {
      expect(BUTTON_BASE).toContain("cursor-pointer");
      expect(BUTTON_BASE).toContain("disabled:cursor-not-allowed");
      expect(BUTTON_BASE).toContain("disabled:opacity-45");
    });

    it("transitions colour at the product's one duration", () => {
      expect(BUTTON_BASE).toMatch(/transition-\[background-color/);
      expect(BUTTON_BASE).toContain("duration-150");
    });

    it("pins a fixed height per size rather than padding alone", () => {
      // Product UI aligns controls on a row; a padded button whose height
      // depends on its font is a button that never lines up with a select.
      expect(BUTTON_SIZE_SM).toMatch(/\bh-7\b/);
      expect(BUTTON_SIZE_MD).toMatch(/\bh-9\b/);
      expect(BUTTON_SIZE_LG).toMatch(/\bh-11\b/);
    });
  });

  describe("variants map intent onto exactly one treatment", () => {
    it("primary is the brand, filled, darkening on press", () => {
      expect(VARIANT_PRIMARY).toContain("bg-brand");
      expect(VARIANT_PRIMARY).toContain("text-ink-inverse");
      expect(VARIANT_PRIMARY).toContain("not-disabled:hover:bg-brand-press");
      expect(VARIANT_PRIMARY).toContain("not-disabled:active:bg-brand-press");
      // Nothing travels or scales under the pointer.
      expect(VARIANT_PRIMARY).not.toMatch(/translate|scale/);
    });

    it("secondary is white with a findable border", () => {
      expect(VARIANT_SECONDARY).toContain("bg-surface");
      // WCAG 1.4.11: on a white surface the border IS the control.
      expect(VARIANT_SECONDARY).toContain("border-line-control");
      expect(VARIANT_SECONDARY).toContain("text-ink");
      expect(VARIANT_SECONDARY).toContain("not-disabled:hover:bg-sunken");
    });

    it("destructive is the stop colour, filled", () => {
      expect(VARIANT_DESTRUCTIVE).toContain("bg-stop");
      expect(VARIANT_DESTRUCTIVE).toContain("text-ink-inverse");
      expect(VARIANT_DESTRUCTIVE).not.toMatch(/translate|scale/);
    });

    it("link is brand, underlined, and carries no surface at all", () => {
      expect(VARIANT_LINK).toContain("text-brand");
      expect(VARIANT_LINK).toContain("underline");
      expect(VARIANT_LINK).not.toMatch(/\bbg-/);
      expect(VARIANT_LINK).not.toMatch(/\bborder\b/);
      expect(VARIANT_LINK).not.toMatch(/\bh-\d/);
    });
  });

  describe("buttonClassName composition", () => {
    it("composes base + size + variant for every filled control", () => {
      for (const variant of ALL_VARIANTS) {
        if (variant === "link") continue;
        for (const size of ALL_SIZES) {
          const out = buttonClassName(variant, size);
          expect(out, `${variant}/${size}`).toContain(BUTTON_BASE);
          expect(out, `${variant}/${size}`).toContain(
            size === "sm"
              ? BUTTON_SIZE_SM
              : size === "lg"
                ? BUTTON_SIZE_LG
                : BUTTON_SIZE_MD,
          );
        }
      }
    });

    it("link ignores base and size entirely", () => {
      const out = buttonClassName("link", "lg");
      expect(out).toBe(VARIANT_LINK);
    });

    it("appends the consumer class last so it can override", () => {
      const out = buttonClassName("primary", "md", "w-full");
      expect(out.endsWith("w-full")).toBe(true);
    });

    it("appends the consumer class on the link variant too", () => {
      const out = buttonClassName("link", "md", "text-xs");
      expect(out).toBe(`${VARIANT_LINK} text-xs`);
    });
  });
});
